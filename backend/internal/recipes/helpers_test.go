package recipes

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// setupTestDB connects to hq_test and TRUNCATES the recipes-relevant fixture tables
// so each test starts clean. Mirrors the TestMain-based connection idiom in
// backend/internal/receipt/worker_test.go: any connect/ping failure causes the
// test to t.Skip rather than t.Fatalf, so a missing or unreachable DB doesn't
// register as a regression.
//
// Required env (checked in order): DB_TEST_URL, TEST_DATABASE_URL. The
// project's Taskfile sets DB_TEST_URL; the alternate is tolerated for CI
// environments using the more conventional name. DATABASE_URL is intentionally
// NOT consulted — it's overloaded for live-dev connections in many shells and
// pointing tests at the wrong DB has bitten the suite before.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = os.Getenv("TEST_DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("DB_TEST_URL / TEST_DATABASE_URL not set — skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("DB_TEST_URL not reachable (connect failed): %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("DB_TEST_URL not reachable (ping failed): %v", err)
	}
	// Truncate in dependency order. CASCADE handles recipes/drift_check_results cleanup.
	_, err = pool.Exec(ctx,
		`TRUNCATE drift_check_results, recipes, daily_menu_sales,
		          purchase_line_items, purchase_events, menu_items, purchase_items
		 RESTART IDENTITY CASCADE`,
	)
	if err != nil {
		pool.Close()
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

// ── Plan 04 additions — promoted from menu_cogs_test.go ──
//
// Plan 02 originally placed seedVendor / seedPurchaseEvent / seedPurchaseLineItem /
// seedDailyMenuSales inside menu_cogs_test.go. Plan 04 expected to define them
// here for the first time (the planner wasn't aware of the menu_cogs_test.go
// fixtures). Deviation Rule 1 (consolidate, don't duplicate): the canonical
// definitions live here so all package tests share one source of truth, and
// menu_cogs_test.go references them by name.
//
// Vendor handling: setupTestDB does NOT TRUNCATE vendors (the table is not
// truncated to avoid breaking adjacent app tests that share the DB). seedVendor
// DELETEs by name first to make repeated `go test` runs idempotent. CASCADE
// from prior test events has already cleared the FK-referencing rows by the
// time setupTestDB returns.

// seedVendor inserts a vendor and returns its UUID.
func seedVendor(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM vendors WHERE name = $1`, name); err != nil {
		t.Fatalf("seedVendor cleanup: %v", err)
	}
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO vendors (name) VALUES ($1) RETURNING id::text`, name,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedVendor(%q): %v", name, err)
	}
	return id
}

// seedPurchaseEvent inserts a purchase_event with given total/tax/date.
// bank_tx_id is synthesized from the date+total+tax tuple to satisfy UNIQUE.
func seedPurchaseEvent(t *testing.T, pool *pgxpool.Pool, vendorID, eventDate string, tax, total float64) string {
	t.Helper()
	bankTx := fmt.Sprintf("tx-%s-%.2f-%.2f", eventDate, total, tax)
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total)
		 VALUES ($1, $2, $3::date, $4, $5) RETURNING id::text`,
		vendorID, bankTx, eventDate, tax, total,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedPurchaseEvent: %v", err)
	}
	return id
}

// seedPurchaseLineItem inserts a purchase_line_item linked to a purchase_event +
// purchase_item. If purchaseItemID is empty, purchase_item_id is NULL.
func seedPurchaseLineItem(t *testing.T, pool *pgxpool.Pool, eventID, purchaseItemID, description string, qty int, price float64) string {
	t.Helper()
	var id string
	var err error
	if purchaseItemID == "" {
		err = pool.QueryRow(context.Background(),
			`INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price, is_case)
			 VALUES ($1, NULL, $2, $3, $4, false) RETURNING id::text`,
			eventID, description, qty, price,
		).Scan(&id)
	} else {
		err = pool.QueryRow(context.Background(),
			`INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price, is_case)
			 VALUES ($1, $2, $3, $4, $5, false) RETURNING id::text`,
			eventID, purchaseItemID, description, qty, price,
		).Scan(&id)
	}
	if err != nil {
		t.Fatalf("seedPurchaseLineItem: %v", err)
	}
	return id
}

// seedDailyMenuSales inserts a daily_menu_sales row. Idempotent via ON CONFLICT
// DO UPDATE so tests can re-seed the same (menu_item, date) key.
func seedDailyMenuSales(t *testing.T, pool *pgxpool.Pool, menuItemID, businessDate string, unitsSold int, gross float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO daily_menu_sales (menu_item_id, business_date, units_sold, gross_amount)
		 VALUES ($1, $2::date, $3, $4)
		 ON CONFLICT (menu_item_id, business_date)
		 DO UPDATE SET units_sold = EXCLUDED.units_sold, gross_amount = EXCLUDED.gross_amount`,
		menuItemID, businessDate, unitsSold, gross,
	)
	if err != nil {
		t.Fatalf("seedDailyMenuSales: %v", err)
	}
}
