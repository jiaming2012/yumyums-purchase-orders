package toast

import (
	"strings"
	"testing"
)

// Test 1 — happy path: 3 distinct master_ids, no voids → 3 AggregatedRows.
func TestParseHappyPath(t *testing.T) {
	csvData := `"Master Id","Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"M1","Jerk Sliders","Main","Sandwiches","","2","20.00","false"
"M2","Plantains","Main","Sides","","1","5.00","false"
"M3","Sorrel","Beverage","Drinks","","3","9.00","false"
`
	rows, err := parseItemSelectionDetails(strings.NewReader(csvData), "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d", len(rows))
	}
	byID := map[string]AggregatedRow{}
	for _, r := range rows {
		byID[r.MasterID] = r
	}
	if byID["M1"].UnitsSold != 2 || byID["M1"].GrossAmount != 20.00 {
		t.Errorf("M1: want 2/20.00, got %d/%.2f", byID["M1"].UnitsSold, byID["M1"].GrossAmount)
	}
	if byID["M2"].UnitsSold != 1 || byID["M2"].GrossAmount != 5.00 {
		t.Errorf("M2: want 1/5.00, got %d/%.2f", byID["M2"].UnitsSold, byID["M2"].GrossAmount)
	}
	if byID["M3"].UnitsSold != 3 || byID["M3"].GrossAmount != 9.00 {
		t.Errorf("M3: want 3/9.00, got %d/%.2f", byID["M3"].UnitsSold, byID["M3"].GrossAmount)
	}
	if byID["M1"].BusinessDate != "2026-05-31" {
		t.Errorf("BusinessDate: want 2026-05-31, got %q", byID["M1"].BusinessDate)
	}
	if byID["M1"].Name != "Jerk Sliders" || byID["M1"].Menu != "Main" || byID["M1"].MenuGroup != "Sandwiches" {
		t.Errorf("M1 metadata mismatch: name=%q menu=%q group=%q", byID["M1"].Name, byID["M1"].Menu, byID["M1"].MenuGroup)
	}
}

// Test 2 — D-06: voided rows excluded entirely from units_sold AND gross_amount.
func TestParseExcludesVoidedRows(t *testing.T) {
	csvData := `"Master Id","Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"M1","Jerk Sliders","Main","Sandwiches","","2","20.00","false"
"M1","Jerk Sliders","Main","Sandwiches","","1","10.00","true"
`
	rows, err := parseItemSelectionDetails(strings.NewReader(csvData), "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 aggregated row, got %d", len(rows))
	}
	if rows[0].UnitsSold != 2 {
		t.Errorf("UnitsSold: want 2 (voided line excluded), got %d", rows[0].UnitsSold)
	}
	if rows[0].GrossAmount != 20.00 {
		t.Errorf("GrossAmount: want 20.00 (voided line excluded), got %.2f", rows[0].GrossAmount)
	}
}

// Test 3 — same master_id, same day, multiple non-voided rows → sum qty + sum gross.
func TestParseAggregatesSameMasterIdSameDay(t *testing.T) {
	csvData := `"Master Id","Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"M1","Jerk Sliders","Main","Sandwiches","","2","20.00","false"
"M1","Jerk Sliders","Main","Sandwiches","","3","30.00","false"
"M1","Jerk Sliders","Main","Sandwiches","","1","10.00","false"
`
	rows, err := parseItemSelectionDetails(strings.NewReader(csvData), "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 aggregated row, got %d", len(rows))
	}
	if rows[0].UnitsSold != 6 {
		t.Errorf("UnitsSold: want 6 (summed), got %d", rows[0].UnitsSold)
	}
	if rows[0].GrossAmount != 60.00 {
		t.Errorf("GrossAmount: want 60.00 (summed), got %.2f", rows[0].GrossAmount)
	}
}

// Test 4 — header has UTF-8 BOM prefix on first column name.
func TestParseHandlesUTF8BOMInFirstHeader(t *testing.T) {
	// \xef\xbb\xbf is the UTF-8 BOM. Toast does occasionally emit this.
	csvData := "\xef\xbb\xbf\"Master Id\",\"Menu Item\",\"Menu\",\"Menu Group\",\"Menu Subgroup(s)\",\"Qty\",\"Gross Price\",\"Void?\"\n" +
		"\"M1\",\"Jerk Sliders\",\"Main\",\"Sandwiches\",\"\",\"2\",\"20.00\",\"false\"\n"
	rows, err := parseItemSelectionDetails(strings.NewReader(csvData), "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].MasterID != "M1" {
		t.Errorf("MasterID: want M1 (BOM-stripped header found 'Master Id'), got %q", rows[0].MasterID)
	}
}

// Test 5 — missing required column ("Master Id") → error.
func TestParseMissingRequiredColumnFails(t *testing.T) {
	csvData := `"Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"Jerk Sliders","Main","Sandwiches","","2","20.00","false"
`
	_, err := parseItemSelectionDetails(strings.NewReader(csvData), "2026-05-31")
	if err == nil {
		t.Fatalf("expected error when Master Id column missing, got nil")
	}
	if !strings.Contains(err.Error(), "Master Id") {
		t.Errorf("expected error to mention 'Master Id', got %q", err.Error())
	}
}

// Test 6 — Qty="1.0" decimal-string tolerated, produces UnitsSold=1.
func TestParseDecimalQtyTolerated(t *testing.T) {
	csvData := `"Master Id","Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"M1","Weighed Item","Main","Misc","","1.0","5.00","false"
`
	rows, err := parseItemSelectionDetails(strings.NewReader(csvData), "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if rows[0].UnitsSold != 1 {
		t.Errorf("UnitsSold: want 1 (decimal '1.0' truncated), got %d", rows[0].UnitsSold)
	}
}

// Test 7 — empty Menu Subgroup(s) field → AggregatedRow.MenuSubgroup is nil (not &"").
func TestParseEmptyMenuSubgroupBecomesNil(t *testing.T) {
	csvData := `"Master Id","Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"M1","Jerk Sliders","Main","Sandwiches","","2","20.00","false"
"M2","Coke","Beverage","Drinks","Cans","1","2.50","false"
`
	rows, err := parseItemSelectionDetails(strings.NewReader(csvData), "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	byID := map[string]AggregatedRow{}
	for _, r := range rows {
		byID[r.MasterID] = r
	}
	if byID["M1"].MenuSubgroup != nil {
		t.Errorf("M1 MenuSubgroup: want nil (empty CSV column), got %v", *byID["M1"].MenuSubgroup)
	}
	if byID["M2"].MenuSubgroup == nil {
		t.Errorf("M2 MenuSubgroup: want non-nil 'Cans', got nil")
	} else if *byID["M2"].MenuSubgroup != "Cans" {
		t.Errorf("M2 MenuSubgroup: want 'Cans', got %q", *byID["M2"].MenuSubgroup)
	}
}

// Regression — sales-processor's historical archive concatenates multiple Toast
// exports into one daily file, leaving duplicate header rows mid-stream. Each
// embedded header row carries `"Master Id"` literally in the master_id column.
// The parser must skip these silently (not crash on strconv.ParseFloat("Qty")).
func TestParseEmbeddedDuplicateHeader(t *testing.T) {
	csvData := `"Master Id","Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"M1","Jerk Sliders","Main","Sandwiches","","2","20.00","false"
"Master Id","Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"M2","Plantains","Main","Sides","","1","5.00","false"
"Master Id","Menu Item","Menu","Menu Group","Menu Subgroup(s)","Qty","Gross Price","Void?"
"M3","Sorrel","Beverage","Drinks","","3","9.00","false"
`
	rows, err := parseItemSelectionDetails(strings.NewReader(csvData), "2026-05-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 rows (embedded headers skipped), got %d", len(rows))
	}
	byID := map[string]AggregatedRow{}
	for _, r := range rows {
		byID[r.MasterID] = r
	}
	for _, id := range []string{"M1", "M2", "M3"} {
		if _, ok := byID[id]; !ok {
			t.Errorf("missing master_id %s — embedded header may have masked real data", id)
		}
	}
	if _, ok := byID["Master Id"]; ok {
		t.Errorf("master_id 'Master Id' should have been skipped as embedded header")
	}
}
