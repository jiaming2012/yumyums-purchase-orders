package recipes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// ─────────────────────────────────────────────────────────────────────────────
// Fixture helpers specific to menu-cogs tests (purchase_events,
// purchase_line_items, daily_menu_sales). Shared helpers (setupTestDB,
// seedMenuItem, seedPurchaseItem, seedRecipe) come from helpers_test.go.
// ─────────────────────────────────────────────────────────────────────────────

// seedVendor inserts a vendor and returns its UUID. Note: setupTestDB's TRUNCATE
// does not include vendors (Wave 1's helpers_test.go stops at purchase_items, and
// extending it would break Plan 04's contract). To keep tests idempotent across
// repeated `go test` runs, we DELETE FROM vendors first, then INSERT. CASCADE
// from prior test events has already cleared the FK-referencing rows.
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

// seedPurchaseEvent inserts a purchase_event with given total/tax/date. bank_tx_id
// is synthesized from the date+total to satisfy UNIQUE.
func seedPurchaseEvent(t *testing.T, pool *pgxpool.Pool, vendorID, eventDate string, tax, total float64) string {
	t.Helper()
	var id string
	bankTx := "tx-" + eventDate + "-" + strconv.FormatFloat(total, 'f', 2, 64) + "-" + strconv.FormatFloat(tax, 'f', 2, 64)
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

// seedDailyMenuSales inserts a daily_menu_sales row.
func seedDailyMenuSales(t *testing.T, pool *pgxpool.Pool, menuItemID, businessDate string, unitsSold int, gross float64) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO daily_menu_sales (menu_item_id, business_date, units_sold, gross_amount)
		 VALUES ($1, $2::date, $3, $4)`,
		menuItemID, businessDate, unitsSold, gross,
	)
	if err != nil {
		t.Fatalf("seedDailyMenuSales: %v", err)
	}
}

// callMenuCogs invokes MenuCogsHandler directly with the given query string.
func callMenuCogs(t *testing.T, pool *pgxpool.Pool, query string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	rec := httptest.NewRecorder()
	MenuCogsHandler(pool).ServeHTTP(rec, req)
	return rec
}

// decodeBody decodes the response body into v; t.Fatalf on error.
func decodeBody(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("decode body: %v (body=%s)", err, rec.Body.String())
	}
}

// approxEqual returns true if |a-b| <= eps.
func approxEqual(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

// ─────────────────────────────────────────────────────────────────────────────
// Happy path: summary mode
// Seed: 1 menu_item M, 1 purchase_item P, 1 event total=110 / tax=10 → subtotal=100
//
//	1 line_item qty=2 price=10 → raw 20, tax-prorated 20 * (110/100) = 22.00
//	1 recipe (M, P, 50%) → alloc_cost = 22.00 * 0.5 = 11.00
//	1 daily_menu_sales (M, in_window, units_sold=4) → cost_per_unit = 11/4 = 2.7500
//	unallocated_cogs = 22.00 * (1 - 50/100) = 11.00
//
// ─────────────────────────────────────────────────────────────────────────────
func TestMenuCogs_HappyPath_SummaryMode(t *testing.T) {
	pool := setupTestDB(t)
	vendorID := seedVendor(t, pool, "Acme-Summary")
	miID := seedMenuItem(t, pool, "Salmon Plate")
	piID := seedPurchaseItem(t, pool, "Salmon Fillet")
	eventID := seedPurchaseEvent(t, pool, vendorID, "2026-05-27", 10.00, 110.00)
	seedPurchaseLineItem(t, pool, eventID, piID, "Salmon Fillet", 2, 10.00)
	seedRecipe(t, pool, miID, piID, 50.0)
	seedDailyMenuSales(t, pool, miID, "2026-05-27", 4, 80.00)

	rec := callMenuCogs(t, pool, "from=2026-05-25&to=2026-05-31")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp MenuCOGSResponse
	decodeBody(t, rec, &resp)

	if resp.From != "2026-05-25" || resp.To != "2026-05-31" {
		t.Errorf("From/To = (%s, %s)", resp.From, resp.To)
	}
	if len(resp.MenuItems) != 1 {
		t.Fatalf("MenuItems len = %d, want 1; payload=%s", len(resp.MenuItems), rec.Body.String())
	}
	row := resp.MenuItems[0]
	if row.MenuItemName != "Salmon Plate" {
		t.Errorf("MenuItemName = %q, want Salmon Plate", row.MenuItemName)
	}
	if !approxEqual(row.UnitsSold, 4.0, 0.001) {
		t.Errorf("UnitsSold = %v, want 4", row.UnitsSold)
	}
	if !approxEqual(row.IngredientCostTotal, 11.00, 0.01) {
		t.Errorf("IngredientCostTotal = %v, want 11.00", row.IngredientCostTotal)
	}
	if row.IngredientCostPerUnit == nil {
		t.Fatalf("IngredientCostPerUnit is nil, want ~2.75")
	}
	if !approxEqual(*row.IngredientCostPerUnit, 2.75, 0.01) {
		t.Errorf("IngredientCostPerUnit = %v, want 2.75", *row.IngredientCostPerUnit)
	}
	if resp.UnallocatedCogs == nil {
		t.Fatalf("UnallocatedCogs is nil in summary mode")
	}
	if !approxEqual(*resp.UnallocatedCogs, 11.00, 0.01) {
		t.Errorf("UnallocatedCogs = %v, want 11.00", *resp.UnallocatedCogs)
	}
	if resp.Unallocated != nil {
		t.Errorf("Unallocated should be nil in summary mode; got %+v", resp.Unallocated)
	}
	if row.Ingredients != nil && len(row.Ingredients) > 0 {
		t.Errorf("Ingredients should be empty/nil in summary mode; got %v", row.Ingredients)
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=3600" {
		t.Errorf("Cache-Control = %q, want %q", got, "private, max-age=3600")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Happy path: breakdown mode
// Same seed; expect ingredients[] populated + Unallocated{} object.
// ─────────────────────────────────────────────────────────────────────────────
func TestMenuCogs_HappyPath_BreakdownMode(t *testing.T) {
	pool := setupTestDB(t)
	vendorID := seedVendor(t, pool, "Acme-Breakdown")
	miID := seedMenuItem(t, pool, "Salmon Plate")
	piID := seedPurchaseItem(t, pool, "Salmon Fillet")
	eventID := seedPurchaseEvent(t, pool, vendorID, "2026-05-27", 10.00, 110.00)
	seedPurchaseLineItem(t, pool, eventID, piID, "Salmon Fillet", 2, 10.00)
	seedRecipe(t, pool, miID, piID, 50.0)
	seedDailyMenuSales(t, pool, miID, "2026-05-27", 4, 80.00)

	rec := callMenuCogs(t, pool, "from=2026-05-25&to=2026-05-31&breakdown=true")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var resp MenuCOGSResponse
	decodeBody(t, rec, &resp)

	if len(resp.MenuItems) != 1 {
		t.Fatalf("MenuItems len = %d, want 1; payload=%s", len(resp.MenuItems), rec.Body.String())
	}
	row := resp.MenuItems[0]
	if len(row.Ingredients) != 1 {
		t.Fatalf("Ingredients len = %d, want 1", len(row.Ingredients))
	}
	ingr := row.Ingredients[0]
	if ingr.PurchaseItemDescription != "Salmon Fillet" {
		t.Errorf("PurchaseItemDescription = %q, want Salmon Fillet", ingr.PurchaseItemDescription)
	}
	if !approxEqual(ingr.UsagePct, 50.0, 0.001) {
		t.Errorf("UsagePct = %v, want 50", ingr.UsagePct)
	}
	if !approxEqual(ingr.AllocatedCost, 11.00, 0.01) {
		t.Errorf("AllocatedCost = %v, want 11.00", ingr.AllocatedCost)
	}

	if resp.UnallocatedCogs != nil {
		t.Errorf("UnallocatedCogs should be nil in breakdown mode; got %v", *resp.UnallocatedCogs)
	}
	if resp.Unallocated == nil {
		t.Fatalf("Unallocated is nil in breakdown mode")
	}
	if !approxEqual(resp.Unallocated.Total, 11.00, 0.01) {
		t.Errorf("Unallocated.Total = %v, want 11.00", resp.Unallocated.Total)
	}
	if len(resp.Unallocated.ByIngredient) != 1 {
		t.Fatalf("ByIngredient len = %d, want 1", len(resp.Unallocated.ByIngredient))
	}
	detail := resp.Unallocated.ByIngredient[0]
	if detail.PurchaseItemDescription != "Salmon Fillet" {
		t.Errorf("by_ingredient description = %q", detail.PurchaseItemDescription)
	}
	if !approxEqual(detail.Amount, 11.00, 0.01) {
		t.Errorf("by_ingredient amount = %v, want 11.00", detail.Amount)
	}
	// reason should be "partial allocation (50.00%)" or similar (Postgres formats numeric)
	if detail.Reason == "" {
		t.Errorf("by_ingredient reason is empty, want non-empty 'partial allocation' string")
	}
	if !bytes.Contains([]byte(detail.Reason), []byte("partial allocation")) {
		t.Errorf("by_ingredient reason = %q, want substring 'partial allocation'", detail.Reason)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// units_sold = 0: ingredient_cost_per_unit must serialize to literal `null`.
// Detect by re-parsing the raw body — Go's json decoding can't distinguish
// missing vs explicit null for a *float64 field that uses omitempty=false.
// ─────────────────────────────────────────────────────────────────────────────
func TestMenuCogs_UnitsSoldZero_PerUnitIsNull(t *testing.T) {
	pool := setupTestDB(t)
	vendorID := seedVendor(t, pool, "Acme-ZeroUnits")
	miID := seedMenuItem(t, pool, "Salmon Plate")
	piID := seedPurchaseItem(t, pool, "Salmon Fillet")
	eventID := seedPurchaseEvent(t, pool, vendorID, "2026-05-27", 10.00, 110.00)
	seedPurchaseLineItem(t, pool, eventID, piID, "Salmon Fillet", 2, 10.00)
	seedRecipe(t, pool, miID, piID, 50.0)
	// NOTE: no daily_menu_sales row → units_sold = 0

	rec := callMenuCogs(t, pool, "from=2026-05-25&to=2026-05-31")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	// String-level assertion that the JSON contains "ingredient_cost_per_unit":null
	body := rec.Body.String()
	if !bytes.Contains([]byte(body), []byte(`"ingredient_cost_per_unit":null`)) {
		t.Errorf("expected literal null for ingredient_cost_per_unit; body=%s", body)
	}

	// Also verify the struct decodes with a nil pointer (defense in depth).
	var resp MenuCOGSResponse
	decodeBody(t, rec, &resp)
	if len(resp.MenuItems) != 1 {
		t.Fatalf("MenuItems len = %d, want 1", len(resp.MenuItems))
	}
	if resp.MenuItems[0].IngredientCostPerUnit != nil {
		t.Errorf("IngredientCostPerUnit = %v, want nil", *resp.MenuItems[0].IngredientCostPerUnit)
	}
	if !approxEqual(resp.MenuItems[0].UnitsSold, 0, 0.001) {
		t.Errorf("UnitsSold = %v, want 0", resp.MenuItems[0].UnitsSold)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 400 error: missing from
// ─────────────────────────────────────────────────────────────────────────────
func TestMenuCogs_FromMissing_400(t *testing.T) {
	pool := setupTestDB(t)
	rec := callMenuCogs(t, pool, "to=2026-05-31")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	decodeBody(t, rec, &body)
	if body["error"] != "from must be YYYY-MM-DD" {
		t.Errorf("error = %q, want %q", body["error"], "from must be YYYY-MM-DD")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 400 error: from > to
// ─────────────────────────────────────────────────────────────────────────────
func TestMenuCogs_FromAfterTo_400(t *testing.T) {
	pool := setupTestDB(t)
	rec := callMenuCogs(t, pool, "from=2026-12-01&to=2026-01-01")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	decodeBody(t, rec, &body)
	if body["error"] != "from must be <= to" {
		t.Errorf("error = %q, want %q", body["error"], "from must be <= to")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Cache-Control header on 200 success
// ─────────────────────────────────────────────────────────────────────────────
func TestMenuCogs_CacheControlHeader_OnSuccess(t *testing.T) {
	pool := setupTestDB(t)
	rec := callMenuCogs(t, pool, "from=2026-05-25&to=2026-05-31")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	got := rec.Header().Get("Cache-Control")
	if got != "private, max-age=3600" {
		t.Errorf("Cache-Control = %q, want %q", got, "private, max-age=3600")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Middleware tests: build a router with auth.ServiceTokenMiddleware wrapping
// the handler so we can exercise 401 / 503 paths exactly as production does.
// ─────────────────────────────────────────────────────────────────────────────

// newAuthRouter builds a chi router with the service-token middleware applied
// to /inventory/menu-cogs, mirroring main.go's group registration.
func newAuthRouter(pool *pgxpool.Pool, token string) http.Handler {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(auth.ServiceTokenMiddleware(token))
		r.Get("/inventory/menu-cogs", MenuCogsHandler(pool))
	})
	return r
}

func TestMenuCogs_BearerMissing_401(t *testing.T) {
	pool := setupTestDB(t)
	srv := newAuthRouter(pool, "test-token")
	req := httptest.NewRequest(http.MethodGet, "/inventory/menu-cogs?from=2026-05-25&to=2026-05-31", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"unauthorized"`)) {
		t.Errorf("body = %s, want substring 'unauthorized'", rec.Body.String())
	}
}

func TestMenuCogs_ServiceTokenUnset_503(t *testing.T) {
	pool := setupTestDB(t)
	srv := newAuthRouter(pool, "")
	req := httptest.NewRequest(http.MethodGet, "/inventory/menu-cogs?from=2026-05-25&to=2026-05-31", nil)
	req.Header.Set("Authorization", "Bearer anything")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"service_token_not_configured"`)) {
		t.Errorf("body = %s, want substring 'service_token_not_configured'", rec.Body.String())
	}
}

func TestMenuCogs_BearerWrong_401(t *testing.T) {
	pool := setupTestDB(t)
	srv := newAuthRouter(pool, "test-token")
	req := httptest.NewRequest(http.MethodGet, "/inventory/menu-cogs?from=2026-05-25&to=2026-05-31", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401; body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"unauthorized"`)) {
		t.Errorf("body = %s, want substring 'unauthorized'", rec.Body.String())
	}
}
