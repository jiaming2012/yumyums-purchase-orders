package receipt

import (
	"fmt"
	"math"
)

// ValidateReceiptData validates the parsed receipt items and summary against
// the Mercury bank transaction amount. Mercury debits are NEGATIVE, so the
// receipt total must equal -bankAmount.
func ValidateReceiptData(items []ReceiptItem, summary ReceiptSummary, bankAmount float64) ValidationResult {
	// Check 1: receipt total must match the negated bank transaction amount
	// Mercury records debits as negative values.
	if summary.Total != -bankAmount {
		return ValidationResult{
			Valid:  false,
			Reason: fmt.Sprintf("Receipt total $%.2f does not match transaction amount $%.2f", summary.Total, -bankAmount),
		}
	}

	// Check 2: sum of (item.Price * item.Quantity) must equal (summary.Total - summary.Tax) within $0.01
	itemsTotal := 0.0
	for _, item := range items {
		itemsTotal += item.Price * float64(item.Quantity)
	}
	subtotal := summary.Total - summary.Tax
	if math.Abs(subtotal-itemsTotal) > 0.01 {
		return ValidationResult{
			Valid:  false,
			Reason: fmt.Sprintf("Line item sum $%.2f does not match receipt subtotal $%.2f (diff $%.2f)", itemsTotal, subtotal, math.Abs(subtotal-itemsTotal)),
		}
	}

	// Check 3: sum of item quantities must equal totalUnits + totalCases
	totalQty := 0
	for _, item := range items {
		totalQty += item.Quantity
	}
	if totalQty != summary.TotalUnits+summary.TotalCases {
		return ValidationResult{
			Valid:  false,
			Reason: fmt.Sprintf("item count %d does not match summary units+cases %d", totalQty, summary.TotalUnits+summary.TotalCases),
		}
	}

	return ValidationResult{Valid: true}
}
