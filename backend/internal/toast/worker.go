package toast

import (
	"context"
	"log"
	"time"
)

// StartWorker launches a background goroutine that runs Toast ingest on
// cfg.Interval (default 12h, configurable via TOAST_SYNC_INTERVAL).
//
// DEVIATION from receipt.StartWorker (D-12): missing TOAST_SFTP_KEY_PATH or
// unreadable key file is fail-fast at the CALLER (cmd/server/main.go) — by the
// time this function runs, cfg.SFTPKeyPath has already been Stat'd. If you're
// looking here for graceful-skip behaviour, you won't find it. That's
// intentional: Phase 999.2 (per-menu-item COGS) depends on this ingest, and
// running HQ without it would be misleading.
//
// Configuration knobs (set in cmd/server/main.go via LoadConfigFromEnv):
//   - cfg.Interval        — 12h default; "0" disables this function entirely
//     (caller checks and skips StartWorker; see plan 05).
//   - cfg.SyncWindowDays  — 7
//   - cfg.BackfillDays    — 90 (used on cold start per D-01/D-02)
func StartWorker(ctx context.Context, cfg Config) {
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

// runCycle selects the date window for this tick and calls RunIngest. Owns the
// cold-start branch (D-02) and the per-cycle log line (D-13).
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

	result, err := RunIngest(ctx, cfg.Pool, cfg, fromDate, toDate)
	if err != nil {
		log.Printf("toast worker: ingest cycle aborted: %v", err)
		return
	}

	// D-13: single INFO line per cycle.
	log.Printf("toast ingest: dates=[%s..%s] items_upserted=%d sales_rows_upserted=%d duration=%s",
		fromDate.Format("20060102"), toDate.Format("20060102"),
		result.ItemsUpserted, result.SalesRowsUpserted, result.Duration)
}
