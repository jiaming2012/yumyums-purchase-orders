package toast

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// parseItemSelectionDetails reads a Toast ItemSelectionDetails.csv stream and
// returns one AggregatedRow per master_id for the given businessDate, with
// units_sold and gross_amount summed across non-voided rows only (D-06).
//
// Required columns (FAIL on missing): Master Id, Menu Item, Menu, Menu Group,
// Menu Subgroup(s), Qty, Gross Price, Void?. Toast's schema is stable; a
// missing column means the report changed and we should not silently mis-aggregate.
func parseItemSelectionDetails(r io.Reader, businessDate string) ([]AggregatedRow, error) {
	// Strip UTF-8 BOM at the byte-stream level. csv.Reader can't tolerate a BOM
	// glued to an opening quote (parses as "bare quote in non-quoted-field").
	br := bufio.NewReader(r)
	if peek, _ := br.Peek(3); len(peek) >= 3 && peek[0] == 0xEF && peek[1] == 0xBB && peek[2] == 0xBF {
		_, _ = br.Discard(3)
	}

	rdr := csv.NewReader(br)
	headers, err := rdr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	colIdx := map[string]int{}
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if i == 0 {
			h = strings.TrimPrefix(h, "\xef\xbb\xbf")
		}
		colIdx[h] = i
	}

	for _, col := range []string{"Master Id", "Menu Item", "Menu", "Menu Group", "Menu Subgroup(s)", "Qty", "Gross Price", "Void?"} {
		if _, ok := colIdx[col]; !ok {
			return nil, fmt.Errorf("CSV missing required column %q. Found: %v", col, headers)
		}
	}

	// Aggregate by master_id (preserve first-seen order so tests are deterministic).
	type acc struct {
		row   AggregatedRow
		order int
	}
	agg := map[string]*acc{}
	var order int

	for {
		row, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}

		get := func(col string) string {
			idx, ok := colIdx[col]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}

		// D-06: voided rows are excluded entirely from both counters.
		voidStr := strings.ToLower(get("Void?"))
		if voidStr == "true" || voidStr == "1" || voidStr == "yes" {
			continue
		}

		masterID := get("Master Id")
		if masterID == "" {
			continue // skip rows without a master id (defensive — Toast always populates this)
		}

		qty, err := parseQty(get("Qty"))
		if err != nil {
			return nil, fmt.Errorf("row master_id=%q: parse Qty %q: %w", masterID, get("Qty"), err)
		}
		gross, err := strconv.ParseFloat(get("Gross Price"), 64)
		if err != nil {
			return nil, fmt.Errorf("row master_id=%q: parse Gross Price %q: %w", masterID, get("Gross Price"), err)
		}

		a, ok := agg[masterID]
		if !ok {
			subgroupStr := get("Menu Subgroup(s)")
			var subgroup *string
			if subgroupStr != "" {
				s := subgroupStr
				subgroup = &s
			}
			a = &acc{
				row: AggregatedRow{
					MasterID:     masterID,
					Name:         get("Menu Item"),
					Menu:         get("Menu"),
					MenuGroup:    get("Menu Group"),
					MenuSubgroup: subgroup,
					BusinessDate: businessDate,
				},
				order: order,
			}
			order++
			agg[masterID] = a
		}
		a.row.UnitsSold += qty
		a.row.GrossAmount += gross
	}

	out := make([]AggregatedRow, len(agg))
	for _, a := range agg {
		out[a.order] = a.row
	}
	return out, nil
}

// parseQty tolerates decimal-string qty (e.g. "1.0", "2.5") by parsing as float
// and truncating to int. Toast's CSVs usually emit integers but rare items
// (weighed by ounce) may carry decimals.
func parseQty(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}
