package receipt

import (
	"fmt"
	"math"
	"strings"
)

// ValidateReceiptData validates the parsed receipt items and summary against
// the Mercury bank transaction amount. Mercury debits are NEGATIVE, so the
// receipt total must equal -bankAmount.
//
// Architecture (post-derived-total refactor):
//   - Check 1 (DERIVED TOTAL): compute derivedTotal = sum(item.Price * item.Quantity) + summary.Tax
//     and require |derivedTotal - (-bankAmount)| ≤ $0.01. This makes Claude's
//     summary.Total irrelevant — items ARE the source of truth for the total.
//     On multi-image inputs Claude often gets items right (attempt 1) but reports
//     a wrong summary.Total (e.g. purchase-only on a purchase+refund receipt).
//     Deriving from items absorbs that inconsistency and auto-creates without retry.
//   - Check 2 (items_sum vs subtotal) is REMOVED. It was redundant once items
//     are the source of total — if derived total passes Check 1 the items are
//     already consistent with the bank amount.
//   - Check 3 (units+cases count) is REMOVED. summary.TotalUnits/TotalCases are
//     noisy Claude-reported fields that don't reflect data integrity. Items are
//     the source of truth; the summary quantity block is unreliable (seen live:
//     quantity sum=74, summary=53 on a correct parse). Same trust-the-items
//     logic that justified dropping Check 2.
func ValidateReceiptData(items []ReceiptItem, summary ReceiptSummary, bankAmount float64) ValidationResult {
	// Check 0: every line item needs a non-empty name. An unnamed item would
	// auto-create a purchase_items row with description='' — a ghost catalog
	// item that all future unnamed lines merge into (description is UNIQUE)
	// and a blank first row in the review item picker. Route to review
	// instead so a human names the lines.
	unnamed := 0
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			unnamed++
		}
	}
	if unnamed > 0 {
		return ValidationResult{
			Valid:  false,
			Reason: fmt.Sprintf("Receipt has %d unnamed line item(s) — verify line items", unnamed),
		}
	}

	// Check 1: derived total must match the negated bank transaction amount.
	// derivedTotal = sum(item.Price * item.Quantity) + summary.Tax
	// Mercury records debits as negative values; we compare against -bankAmount.
	itemsSum := 0.0
	for _, item := range items {
		itemsSum += item.Price * item.Quantity
	}
	derivedTotal := itemsSum + summary.Tax
	if math.Abs(derivedTotal-(-bankAmount)) > 0.01 {
		return ValidationResult{
			Valid: false,
			Reason: fmt.Sprintf(
				"Receipt derived total $%.2f does not match transaction amount $%.2f (items_sum=$%.2f + tax=$%.2f)",
				derivedTotal, -bankAmount, itemsSum, summary.Tax,
			),
		}
	}

	return ValidationResult{Valid: true}
}
