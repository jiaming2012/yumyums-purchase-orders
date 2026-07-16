package receipt

import (
	"strings"
	"testing"
)

// Regression: a parsed receipt whose line items include an empty (or
// whitespace-only) name must FAIL validation and route to pending review.
// Before the fix, such receipts passed Check 1 (totals match) and
// auto-created a purchase_items row with description='' — a ghost catalog
// item that every future unnamed line item merges into (description is
// UNIQUE) and that renders as a blank first row in the review item picker.
// Seen live: Mercury tx aef104e6-7d45-11f1 auto-created 3 unnamed lines.
func TestValidateReceiptData_EmptyItemName_RoutesToReview(t *testing.T) {
	cases := []struct {
		name  string
		items []ReceiptItem
	}{
		{"empty name", []ReceiptItem{
			{Name: "Chicken Thighs", Quantity: 1, Price: 13.59},
			{Name: "", Quantity: 1, Price: 4.39},
			{Name: "Peppers", Quantity: 1, Price: 6.49},
		}},
		{"whitespace-only name", []ReceiptItem{
			{Name: "   ", Quantity: 1, Price: 24.47},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sum := 0.0
			for _, it := range tc.items {
				sum += it.Price * it.Quantity
			}
			// Bank amount exactly matches the derived total, so Check 1
			// passes — only the unnamed item should trip validation.
			result := ValidateReceiptData(tc.items, ReceiptSummary{Total: sum}, -sum)
			if result.Valid {
				t.Fatalf("expected Valid=false for receipt with unnamed line item, got Valid=true")
			}
			if !strings.Contains(result.Reason, "unnamed") {
				t.Fatalf("expected Reason to mention unnamed line items, got %q", result.Reason)
			}
		})
	}
}

// Companion: fully-named receipts with matching totals still validate.
func TestValidateReceiptData_NamedItemsMatchingTotal_Valid(t *testing.T) {
	items := []ReceiptItem{
		{Name: "Chicken Thighs", Quantity: 2, Price: 10.00},
		{Name: "Peppers", Quantity: 1, Price: 4.86},
	}
	result := ValidateReceiptData(items, ReceiptSummary{Total: 24.86}, -24.86)
	if !result.Valid {
		t.Fatalf("expected Valid=true, got Valid=false (reason=%q)", result.Reason)
	}
}
