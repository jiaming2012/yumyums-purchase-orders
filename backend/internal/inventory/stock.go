package inventory

// ClassifyStockLevel determines stock level and reorder need from a quantity and thresholds.
//
// Rules:
//   - qty == 0 → "unknown", false
//   - qty >= highT → "high", false
//   - qty > lowT → "medium", true
//   - else (qty <= lowT, qty > 0) → "low", true
//
// NeedsReorder is true for low and medium levels.
func ClassifyStockLevel(qty, lowT, highT int) (level string, needsReorder bool) {
	switch {
	case qty == 0:
		return "unknown", false
	case qty >= highT:
		return "high", false
	case qty > lowT:
		return "medium", true
	default:
		return "low", true
	}
}
