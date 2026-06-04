package recipes

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// setupTestDB connects to hq_test and TRUNCATES the recipes-relevant fixture tables
// so each test starts clean. Mirrors the connection idiom in
// backend/internal/inventory/period_summary_test.go.
//
// Required env (checked in order): DB_TEST_URL, TEST_DATABASE_URL, DATABASE_URL.
// The project's existing Taskfile sets DB_TEST_URL; the alternates are tolerated
// for CI environments that use the more conventional names. Tests t.Skip()
// when none is set.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = os.Getenv("TEST_DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("DB_TEST_URL / TEST_DATABASE_URL / DATABASE_URL not set — skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("setupTestDB: %v", err)
	}
	// Truncate in dependency order. CASCADE handles recipes/drift_check_results cleanup.
	_, err = pool.Exec(context.Background(),
		`TRUNCATE drift_check_results, recipes, daily_menu_sales,
		          purchase_line_items, purchase_events, menu_items, purchase_items
		 RESTART IDENTITY CASCADE`,
	)
	if err != nil {
		t.Fatalf("setupTestDB truncate: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// seedMenuItem inserts a menu_item with sensible defaults and returns its UUID.
// menu_group defaults to "Lunch"; use seedMenuItemFull to override.
func seedMenuItem(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	return seedMenuItemFull(t, pool, name, "Lunch")
}

// seedMenuItemFull lets the caller set menu_group; menu_subgroup is left NULL.
// master_id is set to a synthetic value derived from name to satisfy the UNIQUE constraint.
func seedMenuItemFull(t *testing.T, pool *pgxpool.Pool, name, menuGroup string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO menu_items (master_id, name, menu, menu_group, last_seen)
		 VALUES ($1, $2, $3, $4, CURRENT_DATE) RETURNING id::text`,
		"synthetic-"+name, name, "Main", menuGroup,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedMenuItem(%q): %v", name, err)
	}
	return id
}

// seedPurchaseItem inserts a purchase_item and returns its UUID. Description
// is the only required field for the recipes tests; other columns default.
func seedPurchaseItem(t *testing.T, pool *pgxpool.Pool, description string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO purchase_items (description) VALUES ($1) RETURNING id::text`,
		description,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedPurchaseItem(%q): %v", description, err)
	}
	return id
}

// seedRecipe inserts a recipe row directly (bypassing the sum-check tx) and returns the id.
// Useful for setting up "existing allocations" scenarios in handler tests.
func seedRecipe(t *testing.T, pool *pgxpool.Pool, menuItemID, purchaseItemID string, usagePct float64) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO recipes (menu_item_id, purchase_item_id, usage_pct, updated_at)
		 VALUES ($1, $2, $3, now()) RETURNING id::text`,
		menuItemID, purchaseItemID, usagePct,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedRecipe: %v", err)
	}
	return id
}
