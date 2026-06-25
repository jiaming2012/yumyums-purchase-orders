package receipt

import (
	"fmt"
	"math"
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
//   - Check 3 (units+cases count) is unchanged.
func ValidateReceiptData(items []ReceiptItem, summary ReceiptSummary, bankAmount float64) ValidationResult {
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

	// Check 3: sum of item quantities must equal totalUnits + totalCases.
	// item.Quantity AND summary.TotalUnits/.TotalCases are float64 (tolerate
	// LLM-returned decimals like 40.0 / 85.56); round both sides to int for
	// the comparison — this mirrors the DB-write rounding in createPurchaseEvent
	// and matches the 260607-k1n Check 3 pattern (260607-l9m extends the
	// widening from items to the summary block).
	totalQty := 0.0
	for _, item := range items {
		totalQty += item.Quantity
	}
	roundedQty := int(math.Round(totalQty))
	roundedSummary := int(math.Round(summary.TotalUnits + summary.TotalCases))
	if roundedQty != roundedSummary {
		return ValidationResult{
			Valid:  false,
			Reason: fmt.Sprintf("item count %d does not match summary units+cases %d", roundedQty, roundedSummary),
		}
	}

	return ValidationResult{Valid: true}
}
