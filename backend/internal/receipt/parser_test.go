package receipt

import (
	"strings"
	"testing"
)

// TestParseJSONBody_DecimalQuantity is the regression test for the
// production bug found on 2026-06-07 (quick task 260607-k1n): both Haiku
// and Sonnet returned `"quantity": 40.0` for a Restaurant Depot receipt
// and Go's strict int unmarshaler rejected it, routing the row to pending
// review with parse_error =
//
//	"json: cannot unmarshal number 40.0 into Go struct field
//	 ReceiptItem.items.quantity of type int"
//
// The fix (260607-k1n): ReceiptItem.Quantity is now float64 at parse-time,
// rounded to int at the createPurchaseEvent INSERT boundary.
func TestParseJSONBody_DecimalQuantity(t *testing.T) {
	// Exact shape from the failing payload (vendor anonymized).
	body := `{
	  "items": [
	    {"name": "Chicken Thighs Case", "quantity": 40.0, "price": 1.0, "is_case": false}
	  ],
	  "summary": {"vendor": "Test Vendor", "total_units": 40, "total_cases": 0, "tax": 0.0, "total": 40.0}
	}`
	items, summary, err := parseJSONBody(body)
	if err != nil {
		t.Fatalf("parseJSONBody returned error for decimal quantity: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Quantity != 40.0 {
		t.Errorf("items[0].Quantity = %v, want 40.0", items[0].Quantity)
	}
	if summary.TotalUnits != 40 {
		t.Errorf("summary.TotalUnits = %d, want 40", summary.TotalUnits)
	}
}

// TestParseJSONBody_IntegerQuantity is the companion test proving that
// bare integer quantities (the common case) still parse — widening
// ReceiptItem.Quantity to float64 must not regress this.
func TestParseJSONBody_IntegerQuantity(t *testing.T) {
	body := `{"items":[{"name":"x","quantity":3,"price":1.5,"is_case":false}],"summary":{"vendor":"v","total_units":3,"total_cases":0,"tax":0,"total":4.5}}`
	items, _, err := parseJSONBody(body)
	if err != nil {
		t.Fatalf("parseJSONBody: %v", err)
	}
	if items[0].Quantity != 3.0 {
		t.Errorf("items[0].Quantity = %v, want 3.0", items[0].Quantity)
	}
}

// TestParseJSONBody_MalformedReturnsError ensures parse failures still
// surface as errors (sanity check — the type widening must not swallow
// genuine unmarshal failures).
func TestParseJSONBody_MalformedReturnsError(t *testing.T) {
	_, _, err := parseJSONBody(`{"items": [`) // truncated
	if err == nil {
		t.Fatal("expected error for truncated JSON, got nil")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal") {
		t.Errorf("error %q does not contain 'failed to unmarshal'", err.Error())
	}
}
