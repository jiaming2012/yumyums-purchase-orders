package purchasing

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StartScheduler launches a background goroutine that checks whether the cutoff
// has passed every 15 minutes. It auto-locks the current draft PO when the
// configured cutoff time is reached.
//
// Follows the same goroutine pattern as receipt.StartWorker.
func StartScheduler(ctx context.Context, pool *pgxpool.Pool) {
	log.Println("cutoff scheduler: starting (15m tick)")

	go func() {
		// Run immediately on start
		runCutoffCheck(ctx, pool)

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("cutoff scheduler: shutting down")
				return
			case <-ticker.C:
				runCutoffCheck(ctx, pool)
			}
		}
	}()
}

// runCutoffCheck loads the cutoff config, determines whether the cutoff time
// has passed this week, and locks the current draft PO if so.
func runCutoffCheck(ctx context.Context, pool *pgxpool.Pool) {
	config, err := GetCutoffConfig(ctx, pool)
	if err != nil {
		log.Printf("cutoff scheduler: GetCutoffConfig error: %v", err)
		return
	}
	if config == nil {
		// No cutoff configured yet — nothing to do
		return
	}

	// Load timezone (DST-safe via time.LoadLocation — Pitfall 1)
	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		log.Printf("cutoff scheduler: invalid timezone %q: %v", config.Timezone, err)
		return
	}

	// Parse cutoff time HH:MM (or HH:MM:SS from Postgres TIME cast)
	hour, minute, err := parseCutoffTime(config.CutoffTime)
	if err != nil {
		log.Printf("cutoff scheduler: parse cutoff_time %q: %v", config.CutoffTime, err)
		return
	}

	now := time.Now().In(loc)

	// Find the most recent occurrence of day_of_week + cutoff hour:minute in loc
	// config.DayOfWeek: 0=Sunday, 6=Saturday (matches time.Weekday)
	targetWeekday := time.Weekday(config.DayOfWeek)
	daysBack := int(now.Weekday()) - int(targetWeekday)
	if daysBack < 0 {
		daysBack += 7
	}
	cutoffCandidate := time.Date(now.Year(), now.Month(), now.Day()-daysBack, hour, minute, 0, 0, loc)

	// If the cutoff is in the future (i.e., today is cutoff day but not yet time), it hasn't passed
	if !now.After(cutoffCandidate) {
		log.Printf("cutoff scheduler: cutoff at %s has not yet passed (now: %s)", cutoffCandidate.Format(time.RFC3339), now.Format(time.RFC3339))
		return
	}

	// Block if there's already a locked PO awaiting approval
	var lockedCount int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM purchase_orders WHERE status = 'locked'`).Scan(&lockedCount)
	if lockedCount > 0 {
		log.Println("cutoff scheduler: locked PO pending approval — skipping auto-lock")
		return
	}

	// Find current draft PO
	var draftID string
	err = pool.QueryRow(ctx, `
		SELECT id FROM purchase_orders WHERE status = 'draft' ORDER BY week_start DESC LIMIT 1
	`).Scan(&draftID)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Println("cutoff scheduler: no draft PO found — nothing to lock")
			return
		}
		log.Printf("cutoff scheduler: query draft PO: %v", err)
		return
	}

	// Lock it
	if err := LockPO(ctx, pool, draftID); err != nil {
		if err == ErrPONotDraft {
			// Already locked or transitioned — not an error
			log.Printf("cutoff scheduler: draft PO %s is no longer draft — skipping", draftID)
			return
		}
		log.Printf("cutoff scheduler: LockPO %s: %v", draftID, err)
		return
	}

	log.Printf("cutoff scheduler: locked PO %s (cutoff passed at %s)", draftID, cutoffCandidate.Format(time.RFC3339))
}

// parseCutoffTime splits "HH:MM" or "HH:MM:SS" into hour and minute integers.
func parseCutoffTime(cutoffTime string) (hour, minute int, err error) {
	parts := strings.Split(cutoffTime, ":")
	if len(parts) < 2 {
		return 0, 0, fmt.Errorf("expected HH:MM, got %q", cutoffTime)
	}
	if _, err := fmt.Sscanf(parts[0], "%d", &hour); err != nil {
		return 0, 0, fmt.Errorf("invalid hour in %q: %w", cutoffTime, err)
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minute); err != nil {
		return 0, 0, fmt.Errorf("invalid minute in %q: %w", cutoffTime, err)
	}
	return hour, minute, nil
}
