package inventory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// These are the RED-FIRST regression tests for the "printing charge shows up in
// the Purchases tab" bug.
//
// Root cause: the Mercury sync surfaces EVERY unreceipted card swipe as a
// pending_purchases row so the payroll completeness gate can block on
// unresolved COGS spend (receipt/worker.go). A non-food charge like
// "Dri*Uprinting" (Mercury category "Office Supplies") therefore lands in the
// Inventory → Purchases review queue as a red "Missing Receipt" row even though
// the data model already knows it is not food: mercury_category is not in the
// COGS allowlist, so /period-summary correctly excludes it from payroll.
//
// The list handlers (ListPendingPurchasesHandler / ListPurchaseEventsHandler),
// however, ignored mercury_category and returned everything — so definitively
// non-food charges cluttered the food-review UI. The fix hides rows whose
// mercury_category is KNOWN and NOT in the allowlist. NULL (uncategorised —
// Mercury hasn't classified yet) and allowlisted rows still show, so a food
// purchase Mercury hasn't tagged never silently vanishes.

// TestListPendingPurchases_HidesNonCOGSCategory seeds three unconfirmed pending
// rows — one allowlisted (COGS), one uncategorised (NULL), one definitively
// non-food (Office Supplies) — and asserts the Purchases queue returns the
// first two and drops the third.
func TestListPendingPurchases_HidesNonCOGSCategory(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not set — skipping DB-coupled list filter test")
	}
	resetFixtures(t)

	// event_date "" → NULL, createdAt drives ordering; reason is the
	// no-attachment sentinel so all three carry the "Missing Receipt" shape.
	insertPendingPurchaseFull(t, "tx-cogs", "", "2026-09-01T12:00:00Z", "no_attachment_on_bank_tx", "COGS", "US Foods", -100.00)
	insertPendingPurchaseFull(t, "tx-null", "", "2026-09-01T12:00:00Z", "no_attachment_on_bank_tx", "", "Restaurant Depot", -50.00)
	insertPendingPurchaseFull(t, "tx-print", "", "2026-09-01T12:00:00Z", "no_attachment_on_bank_tx", "Office Supplies", "Dri*Uprinting", -114.51)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ListPendingPurchasesHandler(testPool, []string{"COGS"}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var got []PendingPurchase
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}

	vendors := map[string]bool{}
	for _, p := range got {
		vendors[p.Vendor] = true
	}

	if !vendors["US Foods"] {
		t.Errorf("COGS-category pending row (US Foods) missing from queue; got vendors %v", vendors)
	}
	if !vendors["Restaurant Depot"] {
		t.Errorf("uncategorised (NULL) pending row (Restaurant Depot) missing from queue — NULL must stay visible; got vendors %v", vendors)
	}
	if vendors["Dri*Uprinting"] {
		t.Errorf("non-food pending row (Dri*Uprinting, category Office Supplies) leaked into the Purchases queue — it should be auto-hidden; got vendors %v", vendors)
	}
}

// TestListPurchaseEvents_HidesNonCOGSCategory does the same for confirmed
// purchase_events (the Purchases-tab history list): a non-food event with a
// parsed receipt must not appear either.
func TestListPurchaseEvents_HidesNonCOGSCategory(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not set — skipping DB-coupled list filter test")
	}
	resetFixtures(t)

	piID := insertPurchaseItem(t, "Salmon Fillet")

	// Distinct prices keep the helper's derived bank_tx_id unique per row.
	vCogs := insertVendor(t, "US Foods")
	insertEventAndLineWithCategory(t, vCogs, "2026-09-01", 0, 100.00, 10.00, 10, piID, "COGS")

	vNull := insertVendor(t, "Restaurant Depot")
	insertEventAndLineWithCategory(t, vNull, "2026-09-01", 0, 110.00, 11.00, 10, piID, "")

	vPrint := insertVendor(t, "Dri Uprinting")
	insertEventAndLineWithCategory(t, vPrint, "2026-09-01", 0, 120.00, 12.00, 10, piID, "Office Supplies")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ListPurchaseEventsHandler(testPool, []string{"COGS"}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var got []PurchaseEvent
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}

	vendors := map[string]bool{}
	for _, e := range got {
		vendors[e.VendorName] = true
	}

	if !vendors["US Foods"] {
		t.Errorf("COGS-category event (US Foods) missing from history; got vendors %v", vendors)
	}
	if !vendors["Restaurant Depot"] {
		t.Errorf("uncategorised (NULL) event (Restaurant Depot) missing from history — NULL must stay visible; got vendors %v", vendors)
	}
	if vendors["Dri Uprinting"] {
		t.Errorf("non-food event (Dri Uprinting, category Office Supplies) leaked into purchase history — it should be auto-hidden; got vendors %v", vendors)
	}
}
