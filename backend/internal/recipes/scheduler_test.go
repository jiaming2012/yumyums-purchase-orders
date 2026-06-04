package recipes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yumyums/hq/internal/alerts"
)

// fakeAlertQueue captures enqueued alerts for assertions. Thread-safe.
// Satisfies the package-internal alertEnqueuer interface defined in service.go.
type fakeAlertQueue struct {
	mu     sync.Mutex
	alerts []alerts.Alert
}

func (f *fakeAlertQueue) Enqueue(a alerts.Alert) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alerts = append(f.alerts, a)
}

// installFakeAlertSink overrides the package-level alertSink and returns a
// cleanup hook via t.Cleanup so the prior sink is restored after the test.
func installFakeAlertSink(t *testing.T) *fakeAlertQueue {
	t.Helper()
	fake := &fakeAlertQueue{}
	old := alertSink
	alertSink = fake
	t.Cleanup(func() { alertSink = old })
	return fake
}

func TestComputeDrift_Unallocated_FlagsAbove10Pct(t *testing.T) {
	pool := setupTestDB(t)
	menuItem := seedMenuItem(t, pool, "Sliders")
	purchaseItem := seedPurchaseItem(t, pool, "Chicken Thighs")
	_ = seedRecipe(t, pool, menuItem, purchaseItem, 80.0) // 20% unalloc
	vendorID := seedVendor(t, pool, "drift-unalloc-vendor")
	// (vendorID, "2026-05-25", tax=10, total=110): subtotal 100, tax-incl multiplier 1.10
	eventID := seedPurchaseEvent(t, pool, vendorID, "2026-05-25", 10.0, 110.0)
	// (eventID, purchaseItem, "Chicken Thighs", qty=2, price=10): raw 20 → tax-incl 22
	_ = seedPurchaseLineItem(t, pool, eventID, purchaseItem, "Chicken Thighs", 2, 10)
	seedDailyMenuSales(t, pool, menuItem, "2026-05-25", 5, 50.0)

	result, err := computeDrift(context.Background(), pool, "2026-05-25", "2026-05-31")
	if err != nil {
		t.Fatalf("computeDrift: %v", err)
	}
	found := false
	for _, sec := range result.Sections {
		if sec.Kind == "unallocated" && len(sec.Items) >= 1 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unallocated section with at least 1 item, got %+v", result.Sections)
	}
}

func TestComputeDrift_ZeroSales_FlagsMissingItems(t *testing.T) {
	pool := setupTestDB(t)
	menuItem := seedMenuItem(t, pool, "Beef Tacos")
	purchaseItem := seedPurchaseItem(t, pool, "Beef")
	_ = seedRecipe(t, pool, menuItem, purchaseItem, 100.0)
	// NO daily_menu_sales for this menu_item in window.
	result, err := computeDrift(context.Background(), pool, "2026-05-25", "2026-05-31")
	if err != nil {
		t.Fatalf("computeDrift: %v", err)
	}
	found := false
	for _, sec := range result.Sections {
		if sec.Kind == "zero_sales" {
			for _, it := range sec.Items {
				if it.MenuItemID == menuItem {
					found = true
				}
			}
		}
	}
	if !found {
		t.Errorf("expected zero_sales drift for Beef Tacos")
	}
}

func TestComputeDrift_Divergence_FlagsAbove20Pct(t *testing.T) {
	// Set two recipes on the same ingredient with configured 60/40, then drive
	// daily_menu_sales gross_amount distribution so implied is roughly 30/70.
	// The 60% row should show up as divergent (|60 - 30| > 20).
	pool := setupTestDB(t)
	miA := seedMenuItem(t, pool, "A")
	miB := seedMenuItem(t, pool, "B")
	pi := seedPurchaseItem(t, pool, "Shared Ingredient")
	_ = seedRecipe(t, pool, miA, pi, 60.0)
	_ = seedRecipe(t, pool, miB, pi, 40.0)
	seedDailyMenuSales(t, pool, miA, "2026-05-25", 1, 30.0)
	seedDailyMenuSales(t, pool, miB, "2026-05-25", 1, 70.0)
	result, err := computeDrift(context.Background(), pool, "2026-05-25", "2026-05-31")
	if err != nil {
		t.Fatalf("computeDrift: %v", err)
	}
	found := false
	for _, sec := range result.Sections {
		if sec.Kind == "divergence" && len(sec.Items) >= 1 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected divergence section, got %+v", result.Sections)
	}
}

func TestComputeDrift_HappyClean_NoSections(t *testing.T) {
	pool := setupTestDB(t)
	result, err := computeDrift(context.Background(), pool, "2026-05-25", "2026-05-31")
	if err != nil {
		t.Fatalf("computeDrift: %v", err)
	}
	if len(result.Sections) != 0 {
		t.Errorf("expected 0 sections, got %d", len(result.Sections))
	}
	if result.HasDrift() {
		t.Error("expected HasDrift() = false")
	}
}

func TestFormatCliqMessage_IncludesDeepLink(t *testing.T) {
	result := DriftCheckResult{
		WeekStart: "2026-05-25",
		Sections: []DriftSection{{
			Kind:    "unallocated",
			Heading: "1 unallocated",
			Items:   []DriftItem{{Label: "Chicken Thighs ($89 unalloc)"}},
		}},
	}
	msg := formatCliqMessage(result, "2026-05-25", "https://hq.yumyums.kitchen")
	if !strings.Contains(msg, "[HQ Recipes drift check — week of 2026-05-25]") {
		t.Errorf("missing prefix in: %s", msg)
	}
	if !strings.Contains(msg, "https://hq.yumyums.kitchen/inventory.html#tab=4") {
		t.Errorf("missing deep link in: %s", msg)
	}
	if !strings.Contains(msg, "Chicken Thighs ($89 unalloc)") {
		t.Errorf("missing item label in: %s", msg)
	}
}

func TestRunDriftWeek_IdempotentOnSecondCall(t *testing.T) {
	pool := setupTestDB(t)
	mi := seedMenuItem(t, pool, "Sliders")
	// Seed 7 days of daily_menu_sales so ingest-stale guard passes.
	for i := 18; i <= 24; i++ {
		d := "2026-05-" + zeroPad(i)
		seedDailyMenuSales(t, pool, mi, d, 1, 1.0)
	}
	runDriftWeek(context.Background(), pool, "2026-05-25")
	runDriftWeek(context.Background(), pool, "2026-05-25")
	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM drift_check_results WHERE week_start = $1`, "2026-05-25",
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}

func TestRunDriftWeek_SkipsWhenIngestStale(t *testing.T) {
	pool := setupTestDB(t)
	mi := seedMenuItem(t, pool, "Sliders")
	// Seed only 3 days of sales — below 5/7 threshold.
	for i := 18; i <= 20; i++ {
		d := "2026-05-" + zeroPad(i)
		seedDailyMenuSales(t, pool, mi, d, 1, 1.0)
	}
	runDriftWeek(context.Background(), pool, "2026-05-25")
	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM drift_check_results WHERE week_start = $1`, "2026-05-25",
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows (ingest stale), got %d", count)
	}
}

func TestRunDriftWeek_EnqueuesCliqWhenDriftFound(t *testing.T) {
	pool := setupTestDB(t)
	fake := installFakeAlertSink(t)

	// Seed a drift scenario: ingredient with 50% allocation + spend → unallocated > 10%.
	mi := seedMenuItem(t, pool, "Sliders")
	pi := seedPurchaseItem(t, pool, "Chicken Thighs")
	_ = seedRecipe(t, pool, mi, pi, 50.0)
	vendorID := seedVendor(t, pool, "drift-enqueue-vendor")
	// (vendorID, "2026-05-20", tax=10, total=110)
	eid := seedPurchaseEvent(t, pool, vendorID, "2026-05-20", 10, 110)
	// (eid, pi, "Chicken Thighs", qty=5, price=10)
	_ = seedPurchaseLineItem(t, pool, eid, pi, "Chicken Thighs", 5, 10)
	// Seed 7 days of sales to clear ingest-stale guard.
	for i := 18; i <= 24; i++ {
		d := "2026-05-" + zeroPad(i)
		seedDailyMenuSales(t, pool, mi, d, 1, 5.0)
	}

	runDriftWeek(context.Background(), pool, "2026-05-25")

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.alerts) != 1 {
		t.Errorf("expected exactly 1 alert enqueued, got %d", len(fake.alerts))
		return
	}
	if fake.alerts[0].Channel != alerts.ChannelZohoCliq {
		t.Errorf("wrong channel: %s", fake.alerts[0].Channel)
	}
	if !strings.Contains(fake.alerts[0].Message, "inventory.html#tab=4") {
		t.Errorf("alert missing deep link: %s", fake.alerts[0].Message)
	}
}

func TestRunDriftWeek_NoEnqueueWhenClean(t *testing.T) {
	pool := setupTestDB(t)
	fake := installFakeAlertSink(t)
	mi := seedMenuItem(t, pool, "Sliders")
	for i := 18; i <= 24; i++ {
		d := "2026-05-" + zeroPad(i)
		seedDailyMenuSales(t, pool, mi, d, 1, 1.0)
	}
	runDriftWeek(context.Background(), pool, "2026-05-25")
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.alerts) != 0 {
		t.Errorf("expected 0 alerts on clean week, got %d", len(fake.alerts))
	}
}

func TestDriftBannerHandler_ReturnsLatest(t *testing.T) {
	pool := setupTestDB(t)
	payload := DriftCheckResult{
		WeekStart: "2026-05-25",
		Sections:  []DriftSection{{Kind: "unallocated", Heading: "1 unallocated"}},
	}
	payloadBytes, _ := json.Marshal(payload)
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO drift_check_results (week_start, payload) VALUES ($1, $2)`,
		"2026-05-25", payloadBytes,
	); err != nil {
		t.Fatalf("insert: %v", err)
	}
	req := httptest.NewRequest("GET", "/inventory/recipes/drift", nil)
	w := httptest.NewRecorder()
	DriftBannerHandler(pool).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp DriftCheckResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.WeekStart != "2026-05-25" {
		t.Errorf("expected week_start=2026-05-25, got %s", resp.WeekStart)
	}
}

func TestDriftBannerHandler_EmptyObjectWhenNoRows(t *testing.T) {
	pool := setupTestDB(t)
	if _, err := pool.Exec(context.Background(), `TRUNCATE drift_check_results`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	req := httptest.NewRequest("GET", "/inventory/recipes/drift", nil)
	w := httptest.NewRecorder()
	DriftBannerHandler(pool).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if body != "{}" {
		t.Errorf("expected '{}', got '%s'", body)
	}
}

// zeroPad returns the 2-digit string of n (n in [0, 99]).
func zeroPad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

// itoa is a tiny base-10 formatter for ints used in fixture date strings.
// Avoids pulling in strconv just for test fixtures.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
