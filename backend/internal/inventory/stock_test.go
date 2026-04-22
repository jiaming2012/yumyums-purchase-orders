package inventory

import "testing"

func TestClassifyStockLevel(t *testing.T) {
	tests := []struct {
		name          string
		qty           int
		lowT          int
		highT         int
		wantLevel     string
		wantReorder   bool
	}{
		{
			name: "zero quantity is unknown",
			qty: 0, lowT: 3, highT: 10,
			wantLevel: "unknown", wantReorder: false,
		},
		{
			name: "below low threshold is low",
			qty: 2, lowT: 3, highT: 10,
			wantLevel: "low", wantReorder: true,
		},
		{
			name: "equal to low threshold is low",
			qty: 3, lowT: 3, highT: 10,
			wantLevel: "low", wantReorder: true,
		},
		{
			name: "between low and high is medium",
			qty: 5, lowT: 3, highT: 10,
			wantLevel: "medium", wantReorder: true,
		},
		{
			name: "just below high threshold is medium",
			qty: 9, lowT: 3, highT: 10,
			wantLevel: "medium", wantReorder: true,
		},
		{
			name: "equal to high threshold is high",
			qty: 10, lowT: 3, highT: 10,
			wantLevel: "high", wantReorder: false,
		},
		{
			name: "above high threshold is high",
			qty: 15, lowT: 3, highT: 10,
			wantLevel: "high", wantReorder: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			level, needsReorder := ClassifyStockLevel(tc.qty, tc.lowT, tc.highT)
			if level != tc.wantLevel {
				t.Errorf("ClassifyStockLevel(%d, %d, %d) level = %q, want %q",
					tc.qty, tc.lowT, tc.highT, level, tc.wantLevel)
			}
			if needsReorder != tc.wantReorder {
				t.Errorf("ClassifyStockLevel(%d, %d, %d) needsReorder = %v, want %v",
					tc.qty, tc.lowT, tc.highT, needsReorder, tc.wantReorder)
			}
		})
	}
}
