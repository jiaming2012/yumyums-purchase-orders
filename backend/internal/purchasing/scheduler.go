package purchasing

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/alerts"
	"github.com/yumyums/hq/internal/users"
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
		runSchedulerTick(ctx, pool)

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("cutoff scheduler: shutting down")
				return
			case <-ticker.C:
				runSchedulerTick(ctx, pool)
			}
		}
	}()
}

// runSchedulerTick runs both the cutoff check and the cutoff reminder check on each tick.
func runSchedulerTick(ctx context.Context, pool *pgxpool.Pool) {
	runCutoffCheck(ctx, pool)
	runReminderCheck(ctx, pool)
}

// runReminderCheck sends a 24-hour cutoff reminder to crew members if it hasn't been
// sent yet this week (D-08: single reminder, idempotent via alert_log table).
func runReminderCheck(ctx context.Context, pool *pgxpool.Pool) {
	if alertQueue == nil {
		return // alerts not configured — skip silently
	}

	config, err := GetCutoffConfig(ctx, pool)
	if err != nil {
		log.Printf("reminder check: GetCutoffConfig error: %v", err)
		return
	}
	if config == nil {
		return // no cutoff configured
	}

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		log.Printf("reminder check: invalid timezone %q: %v", config.Timezone, err)
		return
	}

	hour, minute, err := parseCutoffTime(config.CutoffTime)
	if err != nil {
		log.Printf("reminder check: parse cutoff_time %q: %v", config.CutoffTime, err)
		return
	}

	now := time.Now().In(loc)

	// Compute the cutoff time this week
	targetWeekday := time.Weekday(config.DayOfWeek)
	daysAhead := int(targetWeekday) - int(now.Weekday())
	if daysAhead < 0 {
		daysAhead += 7
	}
	cutoffTime := time.Date(now.Year(), now.Month(), now.Day()+daysAhead, hour, minute, 0, 0, loc)

	// Reminder window: 24h to 23h before cutoff (check within a 1-hour window to match 15m tick)
	reminderWindowStart := cutoffTime.Add(-24 * time.Hour)
	reminderWindowEnd := cutoffTime.Add(-23 * time.Hour)

	if now.Before(reminderWindowStart) || now.After(reminderWindowEnd) {
		return // not in reminder window
	}

	// Determine week_start for this cutoff (Monday of the week the cutoff applies to)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	weekStart := monday.Format("2006-01-02")

	// Idempotency check: was the reminder already sent this week?
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM alert_log WHERE alert_type = 'cutoff_reminder' AND week_start = $1)
	`, weekStart).Scan(&exists)
	if err != nil {
		log.Printf("reminder check: idempotency query error: %v", err)
		return
	}
	if exists {
		return // already sent
	}

	// Get current draft PO item count (D-10: include in reminder)
	var itemCount int
	_ = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM po_line_items pli
		JOIN purchase_orders po ON po.id = pli.po_id
		WHERE po.status = 'draft'
	`).Scan(&itemCount)

	// Day name for message
	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	cutoffDay := dayNames[config.DayOfWeek]
	msg := fmt.Sprintf(
		"Reminder: PO cutoff is %s at %s. Current order has %d item(s). Add anything else before the deadline!",
		cutoffDay, config.CutoffTime, itemCount,
	)

	// Get crew members who can add to PO (D-09)
	contacts, err := users.GetUsersForAlerts(ctx, pool, alerts.TypeCutoffReminder)
	if err != nil {
		log.Printf("reminder check: GetUsersForAlerts: %v", err)
		return
	}

	for _, c := range contacts {
		alertQueue.Enqueue(alerts.Alert{
			Channel:        c.NotificationChannel,
			RecipientEmail: c.Email,
			Subject:        "PO Cutoff Reminder",
			Message:        msg,
		})
	}

	// Record that reminder was sent this week
	_, err = pool.Exec(ctx, `
		INSERT INTO alert_log (alert_type, week_start) VALUES ('cutoff_reminder', $1)
		ON CONFLICT (alert_type, week_start) DO NOTHING
	`, weekStart)
	if err != nil {
		log.Printf("reminder check: insert alert_log: %v", err)
	}

	log.Printf("reminder check: sent cutoff reminder for week %s (%d recipients)", weekStart, len(contacts))
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
