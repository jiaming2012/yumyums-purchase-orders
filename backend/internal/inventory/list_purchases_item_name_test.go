package inventory

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestListPurchaseEvents_ReturnsLinkedItemName: a confirmed purchase line stores
// the receipt's own text as its description ("Ff Big-C Excalibur 6/4.5#"),
// which is rarely what the crew calls the item. The Purchases tab shows the
// catalog name as the primary label when a line is linked, so the events list
// must carry it — one more column per line rather than a second fetch of the
// whole catalog from the frontend. Unlinked lines carry no item_name.
func TestListPurchaseEvents_ReturnsLinkedItemName(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not set — skipping DB-coupled list test")
	}
	resetFixtures(t)

	piID := insertPurchaseItem(t, "Flour Big-C Excalibur")
	vendor := insertVendor(t, "Restaurant Depot")
	insertEventAndLine(t, vendor, "2026-09-27", 0, 44.09, 31.99, 1, piID)
	insertEventAndLine(t, vendor, "2026-09-27", 0, 12.10, 12.10, 1, "")

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

	var linked, unlinked *LineItem
	for i := range got {
		for j := range got[i].LineItems {
			li := &got[i].LineItems[j]
			if li.PurchaseItemID != nil && *li.PurchaseItemID == piID {
				linked = li
			} else if li.PurchaseItemID == nil {
				unlinked = li
			}
		}
	}
	if linked == nil || unlinked == nil {
		t.Fatalf("expected one linked and one unlinked line in the history; got %+v", got)
	}
	if linked.ItemName != "Flour Big-C Excalibur" {
		t.Errorf("linked line item_name = %q, want the catalog name %q", linked.ItemName, "Flour Big-C Excalibur")
	}
	if unlinked.ItemName != "" {
		t.Errorf("unlinked line item_name = %q, want empty", unlinked.ItemName)
	}
}
