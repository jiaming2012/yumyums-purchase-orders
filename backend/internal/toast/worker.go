package toast

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		log.Println("WARNING: toast worker: DO Spaces not configured — ingest disabled (server continues)")
		return
	}

	interval := cfg.Interval
	if interval <= 0 {
		// Defensive — main.go should have already returned early on interval=0.
		log.Println("toast worker: refusing to start with non-positive interval; check TOAST_SYNC_INTERVAL")
		return
	}

	log.Printf("toast worker: starting (interval=%s, window=%dd, backfill=%dd)",
		interval, cfg.SyncWindowDays, cfg.BackfillDays)

	go func() {
		// Run immediately on start (mirrors receipt.StartWorker + purchasing.StartScheduler).
		runCycle(ctx, cfg)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("toast worker: shutting down")
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
		log.Printf("toast worker: cold-start check failed: %v", err)
		return
	}

	windowDays := cfg.SyncWindowDays
	if cold {
		windowDays = cfg.BackfillDays
		log.Printf("toast worker: cold start detected — pulling last %d days", windowDays)
	}

	toDate := time.Now()
	fromDate := toDate.AddDate(0, 0, -windowDays)

	// --- Sync phase ---
	var lastSyncErr error
	syncSystemicErr := false
	syncedCount := 0

	for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
		wrote, sErr := SyncDate(ctx, cfg, d)
		dateDir := d.Format("20060102")
		switch {
		case errors.Is(sErr, ErrSFTPMiss):
			// D-05: expected past Toast retention horizon — INFO log, no counter bump.
			log.Printf("toast sync: skip %s (not in SFTP)", dateDir)
		case sErr != nil:
			// Non-miss Spaces error (PutObject failure, network, auth, etc.) — systemic.
			log.Printf("ERROR toast sync: %s: %v", dateDir, sErr)
			syncSystemicErr = true
			lastSyncErr = sErr
		case wrote:
			syncedCount++
		}
	}

	// --- Ingest phase ---
	result, iErr := RunIngest(ctx, cfg.Pool, cfg, fromDate, toDate)
	if iErr != nil {
		log.Printf("toast worker: ingest cycle aborted: %v", iErr)
		// RunIngest returns non-nil only on systemic precondition failure
		// (e.g., SpacesClient nil) — count as systemic.
		syncSystemicErr = true
		lastSyncErr = iErr
	}

	// --- Failure-counter bookkeeping (D-06) ---
	if syncSystemicErr {
		consecSpacesFails++
		log.Printf("toast worker: consecutive Spaces failures = %d", consecSpacesFails)
		if consecSpacesFails >= 3 && !alertFiredAtThreshold {
			fireDegradedAlert(lastSyncErr)
			alertFiredAtThreshold = true
		}
		// Per-cycle summary still emitted even on partial failure if ingest produced anything.
		if result != nil {
			log.Printf("toast ingest: dates=[%s..%s] items_upserted=%d sales_rows_upserted=%d duration=%s synced=%d (degraded)",
				fromDate.Format("20060102"), toDate.Format("20060102"),
				result.ItemsUpserted, result.SalesRowsUpserted, result.Duration, syncedCount)
		}
		return
	}

	// Clean tick — reset counter + alert latch.
	if consecSpacesFails > 0 || alertFiredAtThreshold {
		log.Printf("toast worker: Spaces recovered after %d consecutive failures", consecSpacesFails)
	}
	consecSpacesFails = 0
	alertFiredAtThreshold = false

	// D-13: single INFO line per cycle.
	log.Printf("toast ingest: dates=[%s..%s] items_upserted=%d sales_rows_upserted=%d duration=%s synced=%d",
		fromDate.Format("20060102"), toDate.Format("20060102"),
		result.ItemsUpserted, result.SalesRowsUpserted, result.Duration, syncedCount)
}

// fireDegradedAlert dispatches one Cliq alert to the purchaseandinventory
// channel (routing already wired via ZOHO_CLIQ_PURCHASE_AND_INVENTORY_CHANNEL
// in alerts/config.go). Nil-safe — no-op if SetAlertQueue was not called.
func fireDegradedAlert(lastErr error) {
	if alertQueue == nil {
		log.Println("toast worker: degraded alert would be sent but alertQueue is not configured")
		return
	}
	msg := fmt.Sprintf("Toast sync degraded: 3 consecutive failed ticks talking to DO Spaces. Last error: %v", lastErr)
	alertQueue.Enqueue(alerts.Alert{
		Channel: alerts.ChannelZohoCliq,
		Message: msg,
	})
	log.Printf("toast worker: ALERT dispatched: %s", msg)
}
