package purchasing

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecordRepurchase writes checked shopping list items to repurchase_log for badge display (REP-01).
// Called inside the CompleteVendorSection transaction — uses the same pool (not tx)
// for simplicity, accepting minor inconsistency if the parent tx rolls back (badges are cosmetic).
func RecordRepurchase(ctx context.Context, pool *pgxpool.Pool, sectionID string) error {
	// Fetch checked items for the vendor section
	rows, err := pool.Query(ctx, `
		SELECT purchase_item_id::text, shopping_list_id::text, quantity
		FROM shopping_list_items
		WHERE vendor_section_id = $1 AND checked = true
	`, sectionID)
	if err != nil {
		return fmt.Errorf("RecordRepurchase: query items: %w", err)
	}
	defer rows.Close()

	type item struct {
		purchaseItemID string
		listID         string
		qty            int
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.purchaseItemID, &it.listID, &it.qty); err != nil {
			return fmt.Errorf("RecordRepurchase: scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("RecordRepurchase: rows err: %w", err)
	}

	for _, it := range items {
		if _, err := pool.Exec(ctx, `
			INSERT INTO repurchase_log (purchase_item_id, shopping_list_id, quantity)
			VALUES ($1, $2, $3)
		`, it.purchaseItemID, it.listID, it.qty); err != nil {
			log.Printf("RecordRepurchase: insert log for item %s: %v", it.purchaseItemID, err)
			// Continue — other items should still be recorded
		}
	}
	return nil
}

// TriggerRepurchaseReset updates last_reset_at on repurchase_reset_config to now.
// Called by admin via POST /api/v1/purchasing/repurchase-reset.
// Badges will not appear for items repurchased before this timestamp.
func TriggerRepurchaseReset(ctx context.Context, pool *pgxpool.Pool) error {
	tag, err := pool.Exec(ctx, `
		UPDATE repurchase_reset_config SET last_reset_at = now(), updated_at = now()
	`)
	if err != nil {
		return fmt.Errorf("TriggerRepurchaseReset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// No row exists yet — insert a default one with last_reset_at = now
		if _, err := pool.Exec(ctx, `
			INSERT INTO repurchase_reset_config (day_of_week, reset_time, timezone, last_reset_at)
			VALUES (1, '06:00', 'America/Chicago', now())
		`); err != nil {
			return fmt.Errorf("TriggerRepurchaseReset: insert default: %w", err)
		}
	}
	return nil
}

// GetRepurchaseResetConfig returns the current repurchase reset configuration.
// Returns nil if not yet configured.
func GetRepurchaseResetConfig(ctx context.Context, pool *pgxpool.Pool) (*RepurchaseResetConfig, error) {
	var cfg RepurchaseResetConfig
	err := pool.QueryRow(ctx, `
		SELECT id::text, day_of_week, reset_time::text, timezone, last_reset_at, updated_at
		FROM repurchase_reset_config
		LIMIT 1
	`).Scan(&cfg.ID, &cfg.DayOfWeek, &cfg.ResetTime, &cfg.Timezone, &cfg.LastResetAt, &cfg.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetRepurchaseResetConfig: %w", err)
	}
	return &cfg, nil
}

// UpsertRepurchaseResetConfig sets the weekly badge reset schedule.
func UpsertRepurchaseResetConfig(ctx context.Context, pool *pgxpool.Pool, dayOfWeek int, resetTime, timezone string) (*RepurchaseResetConfig, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("UpsertRepurchaseResetConfig: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `DELETE FROM repurchase_reset_config`); err != nil {
		return nil, fmt.Errorf("UpsertRepurchaseResetConfig: delete: %w", err)
	}

	var cfg RepurchaseResetConfig
	err = tx.QueryRow(ctx, `
		INSERT INTO repurchase_reset_config (day_of_week, reset_time, timezone)
		VALUES ($1, $2::time, $3)
		RETURNING id::text, day_of_week, reset_time::text, timezone, last_reset_at, updated_at
	`, dayOfWeek, resetTime, timezone).Scan(
		&cfg.ID, &cfg.DayOfWeek, &cfg.ResetTime, &cfg.Timezone, &cfg.LastResetAt, &cfg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("UpsertRepurchaseResetConfig: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("UpsertRepurchaseResetConfig: commit: %w", err)
	}
	return &cfg, nil
}

// runRepurchaseResetCheck checks if the configured reset time has passed since last_reset_at
// and resets badge visibility if so. Called from the scheduler tick.
func runRepurchaseResetCheck(ctx context.Context, pool *pgxpool.Pool) {
	cfg, err := GetRepurchaseResetConfig(ctx, pool)
	if err != nil {
		log.Printf("repurchase reset: GetRepurchaseResetConfig error: %v", err)
		return
	}
	if cfg == nil {
		return // not configured
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Printf("repurchase reset: invalid timezone %q: %v", cfg.Timezone, err)
		return
	}

	hour, minute, err := parseCutoffTime(cfg.ResetTime)
	if err != nil {
		log.Printf("repurchase reset: parse reset_time %q: %v", cfg.ResetTime, err)
		return
	}

	now := time.Now().In(loc)

	// Find the most recent occurrence of (day_of_week, hour, minute) in the past
	targetWeekday := time.Weekday(cfg.DayOfWeek)
	daysBack := int(now.Weekday()) - int(targetWeekday)
	if daysBack < 0 {
		daysBack += 7
	}
	resetCandidate := time.Date(now.Year(), now.Month(), now.Day()-daysBack, hour, minute, 0, 0, loc)

	if !now.After(resetCandidate) {
		return // reset time hasn't passed this week
	}

	// If last_reset_at is after the candidate, already reset this week
	if cfg.LastResetAt != nil && cfg.LastResetAt.After(resetCandidate) {
		return
	}

	if _, err := pool.Exec(ctx, `
		UPDATE repurchase_reset_config SET last_reset_at = $1, updated_at = now()
	`, resetCandidate); err != nil {
		log.Printf("repurchase reset: update error: %v", err)
		return
	}
	log.Printf("repurchase reset: triggered automatic badge reset at %s", resetCandidate.Format(time.RFC3339))
}
