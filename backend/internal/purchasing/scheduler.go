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
	"github.com/yumyums/hq/internal/inventory"
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

// runSchedulerTick runs cutoff check, reminder check, low-stock alert check, and repurchase reset check on each tick.
func runSchedulerTick(ctx context.Context, pool *pgxpool.Pool) {
	runCutoffCheck(ctx, pool)
	runReminderCheck(ctx, pool)
	runLowStockCheck(ctx, pool)
	runRepurchaseResetCheck(ctx, pool)
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
		for _, ch := range c.NotificationChannels {
			alertQueue.Enqueue(alerts.Alert{
				Channel:        ch,
				RecipientEmail: c.Email,
				Subject:        "PO Cutoff Reminder",
				Message:        msg,
			})
		}
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

// runLowStockCheck queries items below their group's low threshold and sends an alert for any
// that haven't been alerted this week (ALRT-02 idempotency via low_stock_alert_log).
func runLowStockCheck(ctx context.Context, pool *pgxpool.Pool) {
	if alertQueue == nil {
		return // alerts not configured — skip silently
	}

	// Compute current week_start (Monday-based, America/Chicago timezone)
	loc, _ := time.LoadLocation("America/Chicago")
	now := time.Now().In(loc)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	weekStart := monday.Format("2006-01-02")

	// Query all stock items with their quantities and thresholds.
	// Uses the same approach as GetSuggestions: purchase_line_items → purchase_items → item_groups.
	type stockRow struct {
		description  string
		currentStock int
		lowThreshold int
		highThreshold int
	}

	rows, err := pool.Query(ctx, `
		SELECT
			COALESCE(pi.description, pli.description) AS item_description,
			COALESCE(sco.quantity, SUM(pli.quantity)::int) AS current_stock,
			COALESCE(ig.low_threshold, 3) AS low_threshold,
			COALESCE(ig.high_threshold, 10) AS high_threshold
		FROM purchase_line_items pli
		JOIN purchase_events pe ON pe.id = pli.purchase_event_id
		LEFT JOIN purchase_items pi ON pi.id = pli.purchase_item_id
		LEFT JOIN item_groups ig ON ig.id = pi.group_id
		LEFT JOIN stock_count_overrides sco ON sco.item_description = COALESCE(pi.description, pli.description)
		WHERE pi.id IS NOT NULL
		GROUP BY COALESCE(pi.description, pli.description), sco.quantity, ig.low_threshold, ig.high_threshold
	`)
	if err != nil {
		log.Printf("low-stock check: query stock: %v", err)
		return
	}
	defer rows.Close()

	var lowItems []string
	for rows.Next() {
		var sr stockRow
		if err := rows.Scan(&sr.description, &sr.currentStock, &sr.lowThreshold, &sr.highThreshold); err != nil {
			log.Printf("low-stock check: scan row: %v", err)
			continue
		}

		level, _ := inventory.ClassifyStockLevel(sr.currentStock, sr.lowThreshold, sr.highThreshold)
		if level != "low" {
			continue
		}

		// Attempt idempotent insert — DO NOTHING if already logged this week
		tag, err := pool.Exec(ctx, `
			INSERT INTO low_stock_alert_log (item_description, week_start) VALUES ($1, $2)
			ON CONFLICT (item_description, week_start) DO NOTHING
		`, sr.description, weekStart)
		if err != nil {
			log.Printf("low-stock check: insert log for %q: %v", sr.description, err)
			continue
		}
		if tag.RowsAffected() > 0 {
			// This item was not yet alerted this week — include it in the batch alert
			lowItems = append(lowItems, sr.description)
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("low-stock check: rows err: %v", err)
	}

	if len(lowItems) == 0 {
		return // nothing new to alert
	}

	// Build alert message
	msg := fmt.Sprintf("Low Stock Alert: %d item(s) below threshold: %s", len(lowItems), strings.Join(lowItems, ", "))

	// Get admin recipients
	contacts, err := users.GetUsersForAlerts(ctx, pool, alerts.TypeShoppingComplete) // admins only
	if err != nil {
		log.Printf("low-stock check: GetUsersForAlerts: %v", err)
		return
	}

	for _, c := range contacts {
		for _, ch := range c.NotificationChannels {
			alertQueue.Enqueue(alerts.Alert{
				Channel:        ch,
				RecipientEmail: c.Email,
				Subject:        "Low Stock Alert",
				Message:        msg,
			})
		}
	}

	log.Printf("low-stock check: sent alert for %d new low-stock item(s) for week %s", len(lowItems), weekStart)
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
