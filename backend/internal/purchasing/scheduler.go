package purchasing

import (
	"context"
	"fmt"
	"log/slog"
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
	slog.Info("cutoff scheduler starting", "tick_interval", "15m")

	go func() {
		// Run immediately on start
		runSchedulerTick(ctx, pool)

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("cutoff scheduler shutting down")
				return
			case <-ticker.C:
				runSchedulerTick(ctx, pool)
			}
		}
	}()
}

// runSchedulerTick runs cutoff check, reminder check, low-stock alert check, and repurchase reset check on each tick.
//
// Production passes time.Now as the injectable clock seam; unit tests inject a
// frozen clock to assert cron decisions deterministically (carried-fix-wos-sweep).
func runSchedulerTick(ctx context.Context, pool *pgxpool.Pool) {
	runCutoffCheck(ctx, pool, time.Now)
	runReminderCheck(ctx, pool, time.Now)
	runLowStockCheck(ctx, pool, time.Now)
	runRepurchaseResetCheck(ctx, pool, time.Now)
}

// runReminderCheck sends a 24-hour cutoff reminder to crew members if it hasn't been
// sent yet this week (D-08: single reminder, idempotent via alert_log table).
func runReminderCheck(ctx context.Context, pool *pgxpool.Pool, now func() time.Time) {
	if alertQueue == nil {
		return // alerts not configured — skip silently
	}

	config, err := GetCutoffConfig(ctx, pool)
	if err != nil {
		slog.Error("reminder check GetCutoffConfig error", "error", err)
		return
	}
	if config == nil {
		return // no cutoff configured
	}

	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		slog.Error("reminder check invalid timezone", "timezone", config.Timezone, "error", err)
		return
	}

	hour, minute, err := parseCutoffTime(config.CutoffTime)
	if err != nil {
		slog.Error("reminder check parse cutoff_time failed", "cutoff_time", config.CutoffTime, "error", err)
		return
	}

	nowT := now().In(loc)

	// Compute the cutoff time this week
	targetWeekday := time.Weekday(config.DayOfWeek)
	daysAhead := int(targetWeekday) - int(nowT.Weekday())
	if daysAhead < 0 {
		daysAhead += 7
	}
	cutoffTime := time.Date(nowT.Year(), nowT.Month(), nowT.Day()+daysAhead, hour, minute, 0, 0, loc)

	// Reminder window: 24h to 23h before cutoff (check within a 1-hour window to match 15m tick)
	reminderWindowStart := cutoffTime.Add(-24 * time.Hour)
	reminderWindowEnd := cutoffTime.Add(-23 * time.Hour)

	if nowT.Before(reminderWindowStart) || nowT.After(reminderWindowEnd) {
		return // not in reminder window
	}

	// Determine week_start for this cutoff (Monday of the week the cutoff applies to)
	weekday := int(nowT.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := nowT.AddDate(0, 0, -(weekday - 1))
	weekStart := monday.Format("2006-01-02")

	// Idempotency check: was the reminder already sent this week?
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM alert_log WHERE alert_type = 'cutoff_reminder' AND week_start = $1)
	`, weekStart).Scan(&exists)
	if err != nil {
		slog.Error("reminder check idempotency query error", "error", err)
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
		slog.Error("reminder check GetUsersForAlerts error", "error", err)
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
		slog.Error("reminder check insert alert_log error", "error", err)
	}

	slog.Info("reminder check sent cutoff reminder", "week_start", weekStart, "recipients", len(contacts))
}

// runCutoffCheck loads the cutoff config, determines whether the cutoff time
// has passed this week, and locks the current draft PO if so.
func runCutoffCheck(ctx context.Context, pool *pgxpool.Pool, now func() time.Time) {
	config, err := GetCutoffConfig(ctx, pool)
	if err != nil {
		slog.Error("cutoff scheduler GetCutoffConfig error", "error", err)
		return
	}
	if config == nil {
		// No cutoff configured yet — nothing to do
		return
	}

	// Load timezone (DST-safe via time.LoadLocation — Pitfall 1)
	loc, err := time.LoadLocation(config.Timezone)
	if err != nil {
		slog.Error("cutoff scheduler invalid timezone", "timezone", config.Timezone, "error", err)
		return
	}

	// Parse cutoff time HH:MM (or HH:MM:SS from Postgres TIME cast)
	hour, minute, err := parseCutoffTime(config.CutoffTime)
	if err != nil {
		slog.Error("cutoff scheduler parse cutoff_time failed", "cutoff_time", config.CutoffTime, "error", err)
		return
	}

	nowT := now().In(loc)

	// Find the most recent occurrence of day_of_week + cutoff hour:minute in loc
	// config.DayOfWeek: 0=Sunday, 6=Saturday (matches time.Weekday)
	targetWeekday := time.Weekday(config.DayOfWeek)
	daysBack := int(nowT.Weekday()) - int(targetWeekday)
	if daysBack < 0 {
		daysBack += 7
	}
	cutoffCandidate := time.Date(nowT.Year(), nowT.Month(), nowT.Day()-daysBack, hour, minute, 0, 0, loc)

	// If the cutoff is in the future (i.e., today is cutoff day but not yet time), it hasn't passed
	if !nowT.After(cutoffCandidate) {
		slog.Info("cutoff scheduler cutoff has not yet passed", "cutoff_at", cutoffCandidate.Format(time.RFC3339), "now", nowT.Format(time.RFC3339))
		return
	}

	// Block if there's already a locked PO awaiting approval
	var lockedCount int
	_ = pool.QueryRow(ctx, `SELECT COUNT(*) FROM purchase_orders WHERE status = 'locked'`).Scan(&lockedCount)
	if lockedCount > 0 {
		slog.Info("cutoff scheduler locked PO pending approval, skipping auto-lock")
		return
	}

	// Find current draft PO
	var draftID string
	err = pool.QueryRow(ctx, `
		SELECT id FROM purchase_orders WHERE status = 'draft' ORDER BY week_start DESC LIMIT 1
	`).Scan(&draftID)
	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Info("cutoff scheduler no draft PO found, nothing to lock")
			return
		}
		slog.Error("cutoff scheduler query draft PO failed", "error", err)
		return
	}

	// Lock it
	if err := LockPO(ctx, pool, draftID); err != nil {
		if err == ErrPONotDraft {
			// Already locked or transitioned — not an error
			slog.Info("cutoff scheduler draft PO is no longer draft, skipping", "po_id", draftID)
			return
		}
		slog.Error("cutoff scheduler LockPO failed", "po_id", draftID, "error", err)
		return
	}

	slog.Info("cutoff scheduler locked PO", "po_id", draftID, "cutoff_at", cutoffCandidate.Format(time.RFC3339))
}

// runLowStockCheck queries items below their group's low threshold and sends an alert for any
// that haven't been alerted this week (ALRT-02 idempotency via low_stock_alert_log).
func runLowStockCheck(ctx context.Context, pool *pgxpool.Pool, now func() time.Time) {
	if alertQueue == nil {
		return // alerts not configured — skip silently
	}

	// Compute current week_start using the admin-configured cutoff timezone.
	// Falls back to America/New_York if no cutoff config is set.
	tzName := users.DefaultTimezone
	config, err := GetCutoffConfig(ctx, pool)
	if err != nil {
		slog.Error("low-stock check GetCutoffConfig error", "error", err)
	} else if config != nil && config.Timezone != "" {
		tzName = config.Timezone
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		slog.Error("low-stock check invalid timezone", "timezone", tzName, "error", err)
		loc, _ = time.LoadLocation(users.DefaultTimezone)
	}
	nowT := now().In(loc)
	weekday := int(nowT.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := nowT.AddDate(0, 0, -(weekday - 1))
	weekStart := monday.Format("2006-01-02")

	// Query all stock items with their quantities and thresholds.
	// Uses the same approach as GetSuggestions: purchase_line_items → purchase_items → item_groups.
	type stockRow struct {
		description   string
		currentStock  int
		lowThreshold  int
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
		slog.Error("low-stock check query stock error", "error", err)
		return
	}
	defer rows.Close()

	var lowItems []string
	for rows.Next() {
		var sr stockRow
		if err := rows.Scan(&sr.description, &sr.currentStock, &sr.lowThreshold, &sr.highThreshold); err != nil {
			slog.Error("low-stock check scan row error", "error", err)
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
			slog.Error("low-stock check insert log error", "item", sr.description, "error", err)
			continue
		}
		if tag.RowsAffected() > 0 {
			// This item was not yet alerted this week — include it in the batch alert
			lowItems = append(lowItems, sr.description)
		}
	}
	if err := rows.Err(); err != nil {
		slog.Error("low-stock check rows error", "error", err)
	}

	if len(lowItems) == 0 {
		return // nothing new to alert
	}

	// Build alert message
	msg := fmt.Sprintf("Low Stock Alert: %d item(s) below threshold: %s", len(lowItems), strings.Join(lowItems, ", "))

	// Get admin recipients
	contacts, err := users.GetUsersForAlerts(ctx, pool, alerts.TypeShoppingComplete) // admins only
	if err != nil {
		slog.Error("low-stock check GetUsersForAlerts error", "error", err)
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

	slog.Info("low-stock check sent alert", "count", len(lowItems), "week_start", weekStart)
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
