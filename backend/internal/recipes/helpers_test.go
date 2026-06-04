package recipes

import (
	"context"
	"fmt"
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

// ── Plan 04 additions — purchase event + line item + daily menu sales fixtures ──

// seedTestVendorCounter generates a unique vendor name per call so concurrent
// seedPurchaseEvent calls don't collide on vendors.name UNIQUE.
var seedTestVendorCounter int64

// seedPurchaseEvent inserts a purchase_event with the given event_date / total /
// tax and returns the event id. Required NOT NULL columns (vendor_id,
// bank_tx_id) are populated internally — vendor is auto-created with a unique
// synthetic name, bank_tx_id is derived from (eventDate, counter) to satisfy
// the UNIQUE constraint across multiple seeds in the same test.
//
// Per plan: signature stable across the package; column adjustments made
// internally per the actual 0024_inventory.sql shape.
func seedPurchaseEvent(t *testing.T, pool *pgxpool.Pool, eventDate string, total, tax float64) string {
	t.Helper()
	seedTestVendorCounter++
	vendorName := fmt.Sprintf("test-vendor-%d", seedTestVendorCounter)
	var vendorID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO vendors (name) VALUES ($1) RETURNING id::text`, vendorName,
	).Scan(&vendorID)
	if err != nil {
		t.Fatalf("seedPurchaseEvent vendor: %v", err)
	}
	bankTxID := fmt.Sprintf("tx-%s-%d", eventDate, seedTestVendorCounter)
	var id string
	err = pool.QueryRow(context.Background(),
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total)
		 VALUES ($1, $2, $3::date, $4, $5) RETURNING id::text`,
		vendorID, bankTxID, eventDate, tax, total,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedPurchaseEvent(%q, %.2f, %.2f): %v", eventDate, total, tax, err)
	}
	return id
}

// seedPurchaseLineItem inserts a purchase_line_item linked to the given event
// and purchase_item. Returns the line item id. The description NOT NULL column
// is set to "Test Line" since tests reading via purchase_item_id don't depend on
// it (the joins in scheduler.go/drift.go go through purchase_items).
func seedPurchaseLineItem(t *testing.T, pool *pgxpool.Pool, eventID, purchaseItemID string, quantity, price float64) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id::text`,
		eventID, purchaseItemID, "Test Line", int(quantity), price,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedPurchaseLineItem(%q, %q): %v", eventID, purchaseItemID, err)
	}
	return id
}

// seedDailyMenuSales inserts a daily_menu_sales row for the given menu_item +
// business_date. Idempotent via PRIMARY KEY (menu_item_id, business_date) using
// ON CONFLICT DO UPDATE so tests can re-seed the same key.
func seedDailyMenuSales(t *testing.T, pool *pgxpool.Pool, menuItemID, businessDate string, unitsSold, grossAmount float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO daily_menu_sales (menu_item_id, business_date, units_sold, gross_amount)
		 VALUES ($1, $2::date, $3, $4)
		 ON CONFLICT (menu_item_id, business_date)
		 DO UPDATE SET units_sold = EXCLUDED.units_sold, gross_amount = EXCLUDED.gross_amount`,
		menuItemID, businessDate, int(unitsSold), grossAmount,
	)
	if err != nil {
		t.Fatalf("seedDailyMenuSales(%q, %q): %v", menuItemID, businessDate, err)
	}
}
