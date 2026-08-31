package toast

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/yumyums/hq/internal/alerts"
)

// alertQueue is the package-level async alert dispatcher (D-06).
// Set via SetAlertQueue at server boot. Nil-safe — if unset, the worker
// logs the failure but doesn't dispatch (matches purchasing.SetAlertQueue
// posture at internal/purchasing/service.go:18-25).
var alertQueue *alerts.Queue

// SetAlertQueue wires the Cliq alert delivery queue into the toast worker.
// Call once at startup BEFORE StartWorker, from cmd/server/main.go.
func SetAlertQueue(q *alerts.Queue) {
	alertQueue = q
}

// syncStatus is the package-level last-sync tracker surfaced at /api/v1/health
// (B-146 fail-loud). Set via SetSyncStatus at server boot. Nil-safe — Snapshot
// reports "unknown" and the recorders no-op when unset.
var syncStatus *SyncStatus

// SetSyncStatus wires the health-facing sync tracker into the toast worker.
// Call once at startup BEFORE StartWorker, from cmd/server/main.go. The health
// handler holds the same *SyncStatus and reads it via Snapshot.
func SetSyncStatus(s *SyncStatus) {
	syncStatus = s
}

// alertEnqueuer is the minimal enqueue seam handleSyncOutcome routes through.
// Production passes alertQueue.Enqueue; tests pass a fake sink's enqueue so the
// fail-loud alert is asserted without a live webhook.
type alertEnqueuer func(alerts.Alert)

// syncOutcome is the per-cycle verdict handleSyncOutcome routes on. Exactly one
// of the failure flags drives the loud path; reachedSFTP marks a clean cycle.
type syncOutcome struct {
	// reachedSFTP is true if at least one date opened SFTP successfully (dial +
	// auth OK), i.e. the transport is alive.
	reachedSFTP bool
	// sftpUnavailable is true if any date hit ErrSFTPUnavailable — dial/auth
	// failed. This is the B-146 loud case: health failing + immediate alert.
	sftpUnavailable bool
	// lastErr is the most recent error for the alert/health summary.
	lastErr error
}

// handleSyncOutcome is the fail-loud routing decision, extracted so it is unit
// testable without a DB or live SFTP. It updates the health-facing SyncStatus
// and, on an SFTP-unavailable cycle, enqueues a Cliq alert via enqueue.
//
// Precedence: an SFTP-unavailable cycle fails loud even if some other date
// happened to read from Spaces — a dead transport means no fresh data lands.
func handleSyncOutcome(o syncOutcome, status *SyncStatus, enqueue alertEnqueuer) {
	if o.sftpUnavailable {
		summary := "sftp dial/auth failed"
		if o.lastErr != nil {
			summary = o.lastErr.Error()
		}
		status.RecordFailure(summary)
		msg := fmt.Sprintf("Toast sync FAILING: cannot open/authenticate SFTP — no sales data is landing. Last error: %s", summary)
		if enqueue != nil {
			enqueue(alerts.Alert{
				Channel: alerts.ChannelZohoCliq,
				Message: msg,
			})
		}
		slog.Error("toast worker: ALERT dispatched (SFTP unavailable)", "message", msg)
		return
	}
	if o.reachedSFTP {
		status.RecordSuccess()
	}
}

// consecSpacesFails tracks consecutive ticks where the worker hit a non-miss
// Spaces error. Resets to 0 on the next clean tick. When it crosses 3 the
// worker fires one Cliq alert (D-06). Single-worker assumption — no mutex
// needed (only runCycle reads/writes it, and runCycle is invoked serially
// by the ticker).
var consecSpacesFails int

// alertFiredAtThreshold is true once we've fired the 3-consec alert in the
// current failure storm. Reset to false on the next clean tick. Prevents
// re-firing on the 4th, 5th, ... consecutive failure.
var alertFiredAtThreshold bool

// StartWorker launches a background goroutine that runs the combined
// Toast sync+ingest cycle on cfg.Interval (default 12h via TOAST_SYNC_INTERVAL).
//
// Phase 22.1 behavior (D-04, D-06):
//   - Each tick: per-date SFTP fetch → write cache + Spaces (sync.SyncDate),
//     then a single RunIngest over the full window reads from Spaces → DB.
//   - Spaces unreachable does NOT crash the server (Phase 22.1 D-06 deviation
//     from Phase 22 D-12). Instead, a consecutive-failure counter fires a
//     Cliq alert after 3 ticks (~36h at default 12h interval).
//   - SFTP misses are silent INFO logs (D-05) — expected past Toast's retention.
//
// Configuration knobs (set in cmd/server/main.go via LoadConfigFromEnv +
// post-load injection of Spaces fields):
//   - cfg.Interval        — 12h default; 0 disables the worker at the caller
//   - cfg.SyncWindowDays  — 7
//   - cfg.BackfillDays    — 90 (cold start per Phase 22 D-01)
//   - cfg.SpacesClient    — REQUIRED; nil means "graceful skip, worker doesn't start"
func StartWorker(ctx context.Context, cfg Config) {
	// Graceful guard for D-06: if Spaces isn't configured, log + return.
	// Server keeps running; receipt worker and other subsystems are unaffected.
	if cfg.SpacesClient == nil || cfg.SpacesBucket == "" {
		slog.Warn("toast worker: DO Spaces not configured — ingest disabled (server continues)")
		return
	}

	interval := cfg.Interval
	if interval <= 0 {
		// Defensive — main.go should have already returned early on interval=0.
		slog.Warn("toast worker: refusing to start with non-positive interval; check TOAST_SYNC_INTERVAL")
		return
	}

	slog.Info("toast worker: starting", "interval", interval, "window_days", cfg.SyncWindowDays, "backfill_days", cfg.BackfillDays)

	go func() {
		// Run immediately on start (mirrors receipt.StartWorker + purchasing.StartScheduler).
		runCycle(ctx, cfg)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("toast worker: shutting down")
				return
			case <-ticker.C:
				runCycle(ctx, cfg)
			}
		}
	}()
}

// runCycle: per-tick combined flow. Sync phase fetches SFTP→Spaces+cache for
// each date in the window; ingest phase reads Spaces→DB for the same window.
// Tracks consecutive Spaces failures and fires a Cliq alert at the threshold.
func runCycle(ctx context.Context, cfg Config) {
	cold, err := isColdStart(ctx, cfg.Pool)
	if err != nil {
		slog.Error("toast worker: cold-start check failed", "error", err)
		return
	}

	windowDays := cfg.SyncWindowDays
	if cold {
		windowDays = cfg.BackfillDays
		slog.Info("toast worker: cold start detected", "backfill_days", windowDays)
	}

	toDate := time.Now()
	fromDate := toDate.AddDate(0, 0, -windowDays)

	// --- Sync phase ---
	var lastSyncErr error
	syncSystemicErr := false
	syncedCount := 0
	// B-146 fail-loud tracking.
	sftpUnavailable := false // any date hit ErrSFTPUnavailable (dead transport)
	reachedSFTP := false     // any date opened SFTP successfully

	for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
		wrote, sErr := SyncDate(ctx, cfg, d)
		dateDir := d.Format("20060102")
		switch {
		case errors.Is(sErr, ErrSFTPUnavailable):
			// B-146: dial/auth failed — the transport is dead. Fail loud.
			// Every date in this cycle will hit the same wall, so log once per
			// date at WARN (SyncDate already logged the ERROR) and mark the
			// cycle unavailable; the alert fires once, after the loop.
			slog.Warn("toast sync: SFTP unavailable", "date", dateDir, "error", sErr)
			sftpUnavailable = true
			lastSyncErr = sErr
		case errors.Is(sErr, ErrSFTPMiss):
			// D-05: expected past Toast retention horizon — INFO log, no counter bump.
			// Distinguish "already archived in Spaces" (fine — ingest will pick it up)
			// from "MISSING from Spaces too" (truly absent — operator may need to migrate).
			csvKey := SpacesCSVKey(dateDir)
			exists, headErr := SpacesKeyExists(ctx, cfg.SpacesClient, cfg.SpacesBucket, csvKey)
			switch {
			case headErr != nil:
				slog.Warn("toast sync: skip (not in SFTP, Spaces check failed)", "date", dateDir, "error", headErr)
			case exists:
				slog.Info("toast sync: skip (not in SFTP, archived in Spaces)", "date", dateDir)
			default:
				slog.Warn("toast sync: skip (not in SFTP, MISSING from Spaces)", "date", dateDir)
			}
		case sErr != nil:
			// Non-miss Spaces error (PutObject failure, network, auth, etc.) — systemic.
			slog.Error("toast sync: systemic error", "date", dateDir, "error", sErr)
			syncSystemicErr = true
			lastSyncErr = sErr
			// Reaching a Spaces PutObject failure means SFTP opened and the CSV
			// downloaded — the transport is alive even though the write failed.
			reachedSFTP = true
		case wrote:
			syncedCount++
			reachedSFTP = true
		}
	}

	// --- Fail-loud routing (B-146): health status + immediate alert ---
	// Runs BEFORE the Spaces-degraded bookkeeping so a dead transport is the
	// dominant signal. A genuine ErrSFTPMiss (date-not-found) is NOT loud.
	if syncStatus != nil {
		var enqueue alertEnqueuer
		if alertQueue != nil {
			enqueue = alertQueue.Enqueue
		}
		handleSyncOutcome(syncOutcome{
			reachedSFTP:     reachedSFTP,
			sftpUnavailable: sftpUnavailable,
			lastErr:         lastSyncErr,
		}, syncStatus, enqueue)
	}

	// --- Ingest phase ---
	result, iErr := RunIngest(ctx, cfg.Pool, cfg, fromDate, toDate)
	if iErr != nil {
		slog.Error("toast worker: ingest cycle aborted", "error", iErr)
		// RunIngest returns non-nil only on systemic precondition failure
		// (e.g., SpacesClient nil) — count as systemic.
		syncSystemicErr = true
		lastSyncErr = iErr
	}

	// --- Failure-counter bookkeeping (D-06) ---
	if syncSystemicErr {
		consecSpacesFails++
		slog.Warn("toast worker: consecutive Spaces failures", "count", consecSpacesFails)
		if consecSpacesFails >= 3 && !alertFiredAtThreshold {
			fireDegradedAlert(lastSyncErr)
			alertFiredAtThreshold = true
		}
		// Per-cycle summary still emitted even on partial failure if ingest produced anything.
		if result != nil {
			slog.Warn("toast ingest: cycle complete (degraded)",
				"from", fromDate.Format("20060102"), "to", toDate.Format("20060102"),
				"items_upserted", result.ItemsUpserted, "sales_rows_upserted", result.SalesRowsUpserted,
				"duration", result.Duration, "synced", syncedCount)
		}
		return
	}

	// Clean tick — reset counter + alert latch.
	if consecSpacesFails > 0 || alertFiredAtThreshold {
		slog.Info("toast worker: Spaces recovered", "previous_failures", consecSpacesFails)
	}
	consecSpacesFails = 0
	alertFiredAtThreshold = false

	// D-13: single INFO line per cycle.
	slog.Info("toast ingest: cycle complete",
		"from", fromDate.Format("20060102"), "to", toDate.Format("20060102"),
		"items_upserted", result.ItemsUpserted, "sales_rows_upserted", result.SalesRowsUpserted,
		"duration", result.Duration, "synced", syncedCount)
}

// fireDegradedAlert dispatches one Cliq alert to the purchaseandinventory
// channel (routing already wired via ZOHO_CLIQ_PURCHASE_AND_INVENTORY_CHANNEL
// in alerts/config.go). Nil-safe — no-op if SetAlertQueue was not called.
func fireDegradedAlert(lastErr error) {
	if alertQueue == nil {
		slog.Warn("toast worker: degraded alert would be sent but alertQueue is not configured")
		return
	}
	msg := fmt.Sprintf("Toast sync degraded: 3 consecutive failed ticks talking to DO Spaces. Last error: %v", lastErr)
	alertQueue.Enqueue(alerts.Alert{
		Channel: alerts.ChannelZohoCliq,
		Message: msg,
	})
	slog.Warn("toast worker: ALERT dispatched", "message", msg)
}
