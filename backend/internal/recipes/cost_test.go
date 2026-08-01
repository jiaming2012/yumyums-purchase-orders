package recipes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/users"
)

// ─────────────────────────────────────────────────────────────────────────────
// F2 — GET /api/v1/inventory/cost (design §2.3, SIGNED).
//
// FIXTURE HONESTY NOTE (this suite exists because sister card F1 was parked for
// a rigged fixture). The seed below is deliberately NOT uniform:
//
//   - E1 is only PARTIALLY catalog-linked: it carries a third line item with
//     purchase_item_id = NULL ("Delivery"), which the inherited window_spend CTE
//     drops. So Σ(linked lines) != subtotal — the normal case, not the tidy one.
//   - E2 has an UNITEMIZED REMAINDER: its two lines sum to 54.99 against a 55.00
//     subtotal. One cent of the receipt has no line row at all.
//   - E1 has mercury_category NULL; E2 has 'COGS'. Both are counted, because the
//     inherited menu-cogs window_spend CTE does NOT filter on mercury_category.
//     TestCost_InheritsMenuCogs_NoCategoryFilter pins that as observed behavior.
//   - A pending_purchases row is seeded and must NOT move any number, because the
//     inherited CTE reads purchase_events only.
//   - Prices are non-round (12.35 / 9.10 / 21.99 / 16.50) and one event has a
//     108/100 tax-proration factor, so every total is a real rounding decision.
//   - Menu items cover five distinct shapes: normal, zero-revenue-with-cost
//     (comped), cost-but-no-sales (retired), sales-but-no-recipe, and neither
//     (dormant — must be absent).
//   - Out-of-window rows (a 2026-06-15 event AND a 2026-06-15 sale) must be
//     excluded; they are large enough that leaking them would be unmissable.
//
// HAND-COMPUTED EXPECTATIONS (to the cent):
//
//	window_spend (tax-prorated, [2026-05-04, 2026-05-31]):
//	  E1 factor = 108/(108-8) = 1.08
//	    P_salmon  3 x 12.35 = 37.05 -> 40.014
//	    P_rice    2 x  9.10 = 18.20 -> 19.656
//	    (NULL item 1 x 44.75 -> dropped by the inherited CTE)
//	  E2 factor = 55/(55-0) = 1.00
//	    P_salmon  1 x 21.99 = 21.99 -> 21.990
//	    P_chicken 2 x 16.50 = 33.00 -> 33.000
//	  => P_salmon 62.004 | P_rice 19.656 | P_chicken 33.000
//
//	alloc -> ingredient_cost_total (ROUNDed to cents at this site):
//	  Salmon Bowl    = 62.004*.60 + 19.656*.25 = 37.2024 + 4.9140 = 42.1164 -> 42.12
//	  Comped Special = 62.004*.15                                 =  9.3006 ->  9.30
//	  Chicken Wrap   = 33.000*.80                                 = 26.4000 -> 26.40
//	  Retired Burrito= 19.656*.20                                 =  3.9312 ->  3.93
//	  Side Salad     = (no recipe rows)                           -> NULL
//
//	revenue / margin / food_cost_pct (margin and pct derive from the ROUNDED cost,
//	so the published invariant margin == revenue - ingredient_cost_total holds):
//	  Chicken Wrap    9u   71.55  26.40  ->  45.15   36.90%  (26.40/71.55 = 36.8973%)
//	  Comped Special  4u    0.00   9.30  ->  -9.30   NULL    (zero-revenue rule)
//	  Retired Burrito 0u    0.00   3.93  ->  -3.93   NULL    (zero-revenue rule)
//	  Salmon Bowl    13u  194.87  42.12  -> 152.75   21.61%  (42.12/194.87 = 21.6144%)
//	  Side Salad      5u   37.75   NULL  ->  NULL    NULL    unallocated="no recipe"
//
// ─────────────────────────────────────────────────────────────────────────────

const (
	costFixtureFrom = "2026-05-04"
	costFixtureTo   = "2026-05-31"
)

type costFixtureIDs struct {
	bowl, comped, wrap, retired, salad, dormant string
}

// seedPurchaseEventCat is seedPurchaseEvent plus an explicit mercury_category,
// so this suite can prove the inherited CTE ignores the column.
func seedPurchaseEventCat(t *testing.T, pool *pgxpool.Pool, vendorID, eventDate string, tax, total float64, category *string) string {
	t.Helper()
	var id string
	bankTx := "f2-tx-" + eventDate + "-" + time.Now().Format("150405.000000000")
	err := pool.QueryRow(context.Background(),
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, mercury_category)
		 VALUES ($1, $2, $3::date, $4, $5, $6) RETURNING id::text`,
		vendorID, bankTx, eventDate, tax, total, category,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedPurchaseEventCat: %v", err)
	}
	return id
}

// seedCostFixture lays down the deliberately-messy fixture documented above.
func seedCostFixture(t *testing.T, pool *pgxpool.Pool) costFixtureIDs {
	t.Helper()
	cogs := "COGS"

	vendorID := seedVendor(t, pool, "F2-Vendor")

	pSalmon := seedPurchaseItem(t, pool, "Salmon Fillet")
	pRice := seedPurchaseItem(t, pool, "Jasmine Rice")
	pChicken := seedPurchaseItem(t, pool, "Chicken Thigh")

	// E1 — partially catalog-linked, NULL mercury_category, 8% tax proration.
	e1 := seedPurchaseEventCat(t, pool, vendorID, "2026-05-06", 8.00, 108.00, nil)
	seedPurchaseLineItem(t, pool, e1, pSalmon, "Salmon Fillet", 3, 12.35)
	seedPurchaseLineItem(t, pool, e1, pRice, "Jasmine Rice", 2, 9.10)
	seedPurchaseLineItem(t, pool, e1, "", "Delivery", 1, 44.75) // unlinked -> dropped

	// E2 — 'COGS' category, no tax, one cent of unitemized remainder.
	e2 := seedPurchaseEventCat(t, pool, vendorID, "2026-05-20", 0.00, 55.00, &cogs)
	seedPurchaseLineItem(t, pool, e2, pSalmon, "Salmon Fillet", 1, 21.99)
	seedPurchaseLineItem(t, pool, e2, pChicken, "Chicken Thigh", 2, 16.50)

	// E3 — OUT of window; large enough that a leak would be unmissable.
	e3 := seedPurchaseEventCat(t, pool, vendorID, "2026-06-15", 0.00, 500.00, &cogs)
	seedPurchaseLineItem(t, pool, e3, pSalmon, "Salmon Fillet", 10, 50.00)

	// A pending (unconfirmed) purchase — inherited CTE reads purchase_events only.
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, reason, created_at, mercury_category)
		 VALUES ($1, $2, $3, $4::jsonb, $5::date, $6, now(), $7)`,
		"f2-pending-tx", 250.00, "F2-Vendor", `[]`, "2026-05-11", "no_attachment_on_bank_tx", "COGS",
	); err != nil {
		t.Fatalf("seed pending_purchases: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM pending_purchases WHERE bank_tx_id = 'f2-pending-tx'`)
	})

	ids := costFixtureIDs{
		bowl:    seedMenuItemFull(t, pool, "Salmon Bowl", "Bowls"),
		comped:  seedMenuItemFull(t, pool, "Comped Special", "Specials"),
		wrap:    seedMenuItemFull(t, pool, "Chicken Wrap", "Wraps"),
		retired: seedMenuItemFull(t, pool, "Retired Burrito", "Wraps"),
		salad:   seedMenuItemFull(t, pool, "Side Salad", "Sides"),
		dormant: seedMenuItemFull(t, pool, "Dormant Item", "Sides"),
	}

	seedRecipe(t, pool, ids.bowl, pSalmon, 60.0)
	seedRecipe(t, pool, ids.bowl, pRice, 25.0)
	seedRecipe(t, pool, ids.comped, pSalmon, 15.0)
	seedRecipe(t, pool, ids.wrap, pChicken, 80.0)
	seedRecipe(t, pool, ids.retired, pRice, 20.0)
	// ids.salad: intentionally NO recipe rows (sales-but-no-recipe).
	// ids.dormant: no recipe, no sales -> must not appear at all.

	seedDailyMenuSales(t, pool, ids.bowl, "2026-05-06", 7, 104.93)
	seedDailyMenuSales(t, pool, ids.bowl, "2026-05-20", 6, 89.94)
	seedDailyMenuSales(t, pool, ids.comped, "2026-05-20", 4, 0.00) // zero revenue
	seedDailyMenuSales(t, pool, ids.wrap, "2026-05-20", 9, 71.55)
	seedDailyMenuSales(t, pool, ids.salad, "2026-05-20", 5, 37.75)
	// OUT of window — must be excluded.
	seedDailyMenuSales(t, pool, ids.bowl, "2026-06-15", 99, 999.99)

	return ids
}

// centsEqual compares two money floats to the cent.
func centsEqual(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.005
}

func rowByName(t *testing.T, resp CostResponse, name string) CostRow {
	t.Helper()
	for _, r := range resp.Rows {
		if r.MenuItemName == name {
			return r
		}
	}
	t.Fatalf("row %q not found; got %d rows: %+v", name, len(resp.Rows), resp.Rows)
	return CostRow{}
}

// ─────────────────────────────────────────────────────────────────────────────
// Window: 12 complete ISO weeks ending the Sunday before the current week,
// evaluated in the APP timezone (users.DefaultTimezone — America/New_York,
// ledger T-26 decision 83). The design §2.2 example pins today=2026-07-20 ->
// from=2026-04-27, to=2026-07-19, weeks=12.
//
// Card A1 changed one case here for real, not to make a test pass. The old
// "Sunday 2026-07-19 23:30 Chicago" fixture is 00:30 MONDAY 2026-07-20 in New
// York, so its window genuinely moves forward one week. It is kept below,
// relabelled, as the boundary case — with a 22:30 New York sibling that is
// Sunday in BOTH zones, so the pair brackets the boundary instead of merely
// following it.
// ─────────────────────────────────────────────────────────────────────────────
func TestCostWindow_TwelveCompleteISOWeeksAppTimezone(t *testing.T) {
	loc, err := time.LoadLocation(users.DefaultTimezone)
	if err != nil {
		t.Skipf("%s tzdata unavailable: %v", users.DefaultTimezone, err)
	}
	chi, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Skipf("America/Chicago tzdata unavailable: %v", err)
	}
	cases := []struct {
		name     string
		now      time.Time
		from, to string
		weeks    int
	}{
		{
			name: "Monday 2026-07-20 (design §2.2 worked example)",
			now:  time.Date(2026, 7, 20, 9, 0, 0, 0, loc),
			from: "2026-04-27", to: "2026-07-19", weeks: 12,
		},
		{
			// THE BOUNDARY. 23:30 Sunday in Chicago is 00:30 Monday in New
			// York: the app has already rolled into the new ISO week.
			name: "Sunday 2026-07-19 23:30 Chicago = Monday 00:30 New York",
			now:  time.Date(2026, 7, 19, 23, 30, 0, 0, chi),
			from: "2026-04-27", to: "2026-07-19", weeks: 12,
		},
		{
			// One hour earlier — Sunday in both zones. Brackets the boundary.
			name: "Sunday 2026-07-19 22:30 New York still in prior ISO week",
			now:  time.Date(2026, 7, 19, 22, 30, 0, 0, loc),
			from: "2026-04-20", to: "2026-07-12", weeks: 12,
		},
		{
			name: "UTC instant that is still Sunday in New York",
			now:  time.Date(2026, 7, 20, 3, 0, 0, 0, time.UTC), // 23:00 Sun in New York
			from: "2026-04-20", to: "2026-07-12", weeks: 12,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			from, to, weeks := costWindow(tc.now)
			if from != tc.from || to != tc.to || weeks != tc.weeks {
				t.Errorf("costWindow(%s) = (%q, %q, %d), want (%q, %q, %d)",
					tc.now.Format(time.RFC3339), from, to, weeks, tc.from, tc.to, tc.weeks)
			}
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Every published number matched against the hand computation, to the cent.
// ─────────────────────────────────────────────────────────────────────────────
func TestCost_Fixture_HandComputedToTheCent(t *testing.T) {
	pool := setupTestDB(t)
	seedCostFixture(t, pool)

	resp, err := queryCost(context.Background(), pool, costFixtureFrom, costFixtureTo)
	if err != nil {
		t.Fatalf("queryCost: %v", err)
	}

	// Dormant Item has neither sales nor recipes -> absent.
	if len(resp.Rows) != 5 {
		names := []string{}
		for _, r := range resp.Rows {
			names = append(names, r.MenuItemName)
		}
		t.Fatalf("len(Rows) = %d, want 5 (Dormant Item must be absent); got %v", len(resp.Rows), names)
	}
	for _, r := range resp.Rows {
		if r.MenuItemName == "Dormant Item" {
			t.Errorf("Dormant Item (no sales, no recipe) must not appear")
		}
	}

	// Rows are name-ordered for deterministic rendering.
	wantOrder := []string{"Chicken Wrap", "Comped Special", "Retired Burrito", "Salmon Bowl", "Side Salad"}
	for i, want := range wantOrder {
		if resp.Rows[i].MenuItemName != want {
			t.Errorf("Rows[%d].MenuItemName = %q, want %q", i, resp.Rows[i].MenuItemName, want)
		}
	}

	type want struct {
		units, revenue float64
		cost           *float64 // nil => JSON null
		margin         *float64
		pct            *float64
		unallocated    *string
	}
	f := func(v float64) *float64 { return &v }
	s := func(v string) *string { return &v }

	noRecipe := "no recipe"
	cases := map[string]want{
		"Salmon Bowl":     {units: 13, revenue: 194.87, cost: f(42.12), margin: f(152.75), pct: f(21.61)},
		"Chicken Wrap":    {units: 9, revenue: 71.55, cost: f(26.40), margin: f(45.15), pct: f(36.90)},
		"Comped Special":  {units: 4, revenue: 0.00, cost: f(9.30), margin: f(-9.30), pct: nil},
		"Retired Burrito": {units: 0, revenue: 0.00, cost: f(3.93), margin: f(-3.93), pct: nil},
		"Side Salad":      {units: 5, revenue: 37.75, cost: nil, margin: nil, pct: nil, unallocated: s(noRecipe)},
	}

	for name, w := range cases {
		row := rowByName(t, resp, name)
		if !centsEqual(row.UnitsSold, w.units) {
			t.Errorf("%s: UnitsSold = %v, want %v", name, row.UnitsSold, w.units)
		}
		if !centsEqual(row.Revenue, w.revenue) {
			t.Errorf("%s: Revenue = %v, want %v", name, row.Revenue, w.revenue)
		}
		checkPtr := func(field string, got, wantV *float64) {
			switch {
			case wantV == nil && got != nil:
				t.Errorf("%s: %s = %v, want null", name, field, *got)
			case wantV != nil && got == nil:
				t.Errorf("%s: %s = null, want %v", name, field, *wantV)
			case wantV != nil && got != nil && !centsEqual(*got, *wantV):
				t.Errorf("%s: %s = %v, want %v (to the cent)", name, field, *got, *wantV)
			}
		}
		checkPtr("IngredientCostTotal", row.IngredientCostTotal, w.cost)
		checkPtr("Margin", row.Margin, w.margin)
		checkPtr("FoodCostPct", row.FoodCostPct, w.pct)

		switch {
		case w.unallocated == nil && row.Unallocated != nil:
			t.Errorf("%s: Unallocated = %q, want null", name, *row.Unallocated)
		case w.unallocated != nil && row.Unallocated == nil:
			t.Errorf("%s: Unallocated = null, want %q", name, *w.unallocated)
		case w.unallocated != nil && row.Unallocated != nil && *row.Unallocated != *w.unallocated:
			t.Errorf("%s: Unallocated = %q, want %q", name, *row.Unallocated, *w.unallocated)
		}

		// The published invariant: margin == revenue - ingredient_cost_total,
		// using the numbers the client actually receives.
		if row.Margin != nil && row.IngredientCostTotal != nil {
			if !centsEqual(*row.Margin, row.Revenue-*row.IngredientCostTotal) {
				t.Errorf("%s: published margin %v != revenue %v - cost %v",
					name, *row.Margin, row.Revenue, *row.IngredientCostTotal)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Zero-revenue rule (A2, signed): food_cost_pct is JSON null — never 0, never Inf.
// This asserts on the wire bytes, not just the Go struct, because a silent 0
// would misrepresent money to the client.
// ─────────────────────────────────────────────────────────────────────────────
func TestCost_ZeroRevenue_FoodCostPctIsJSONNull(t *testing.T) {
	pool := setupTestDB(t)
	seedCostFixture(t, pool)

	resp, err := queryCost(context.Background(), pool, costFixtureFrom, costFixtureTo)
	if err != nil {
		t.Fatalf("queryCost: %v", err)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var envelope struct {
		Rows []map[string]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	found := false
	for _, r := range envelope.Rows {
		var name string
		_ = json.Unmarshal(r["menu_item_name"], &name)
		if name != "Comped Special" {
			continue
		}
		found = true
		if got := string(r["food_cost_pct"]); got != "null" {
			t.Errorf("Comped Special food_cost_pct on the wire = %s, want null", got)
		}
		if got := string(r["revenue"]); got != "0" {
			t.Errorf("Comped Special revenue on the wire = %s, want 0", got)
		}
		// The row must keep its units and its negative margin.
		if got := string(r["units_sold"]); got != "4" {
			t.Errorf("Comped Special units_sold = %s, want 4", got)
		}
		var margin float64
		if err := json.Unmarshal(r["margin"], &margin); err != nil {
			t.Fatalf("margin not a number: %s", string(r["margin"]))
		}
		if !centsEqual(margin, -9.30) {
			t.Errorf("Comped Special margin = %v, want -9.30", margin)
		}
	}
	if !found {
		t.Fatal("Comped Special row missing from the wire payload")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Both movers orderings (§2.3): null-% rows excluded from by_food_cost_pct;
// comped/negative rows DO participate in by_margin.
// ─────────────────────────────────────────────────────────────────────────────
func TestCost_Movers_BothOrderings(t *testing.T) {
	pool := setupTestDB(t)
	ids := seedCostFixture(t, pool)

	resp, err := queryCost(context.Background(), pool, costFixtureFrom, costFixtureTo)
	if err != nil {
		t.Fatalf("queryCost: %v", err)
	}

	eq := func(label string, got, want []string) {
		if len(got) != len(want) {
			t.Errorf("%s: len = %d, want %d (got %v)", label, len(got), len(want), got)
			return
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %s, want %s (full: %v)", label, i, got[i], want[i], got)
			}
		}
	}

	// by_food_cost_pct: best = lowest %, worst = highest %.
	// Only Salmon Bowl (21.61) and Chicken Wrap (36.90) have a non-null %.
	eq("movers.by_food_cost_pct.best", resp.Movers.ByFoodCostPct.Best, []string{ids.bowl, ids.wrap})
	eq("movers.by_food_cost_pct.worst", resp.Movers.ByFoodCostPct.Worst, []string{ids.wrap, ids.bowl})

	// by_margin: best = highest dollars, worst = lowest. Negative margins are
	// real worst-movers and must participate; Side Salad (null margin) must not.
	eq("movers.by_margin.best", resp.Movers.ByMargin.Best,
		[]string{ids.bowl, ids.wrap, ids.retired, ids.comped})
	eq("movers.by_margin.worst", resp.Movers.ByMargin.Worst,
		[]string{ids.comped, ids.retired, ids.wrap, ids.bowl})

	for _, id := range resp.Movers.ByFoodCostPct.Best {
		if id == ids.comped || id == ids.salad || id == ids.retired {
			t.Errorf("null-%% row %s must not appear in by_food_cost_pct", id)
		}
	}
	for _, id := range resp.Movers.ByMargin.Best {
		if id == ids.salad {
			t.Errorf("null-margin row Side Salad must not appear in by_margin")
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Cross-endpoint agreement on the SAME messy fixture: cost's units_sold and
// ingredient_cost_total must equal menu-cogs's for every menu item menu-cogs
// returns. This is the probe F1 was parked for skipping.
//
// Known, asserted shape difference: menu-cogs INNER-JOINs alloc, so a menu item
// with sales but NO recipe rows (Side Salad) is absent from menu-cogs entirely.
// Cost surfaces it with a "no recipe" marker rather than dropping the revenue.
// ─────────────────────────────────────────────────────────────────────────────
func TestCost_ReconcilesWithMenuCogs_OnMessyData(t *testing.T) {
	pool := setupTestDB(t)
	seedCostFixture(t, pool)

	cost, err := queryCost(context.Background(), pool, costFixtureFrom, costFixtureTo)
	if err != nil {
		t.Fatalf("queryCost: %v", err)
	}

	rec := callMenuCogs(t, pool, "from="+costFixtureFrom+"&to="+costFixtureTo)
	if rec.Code != http.StatusOK {
		t.Fatalf("menu-cogs status = %d body=%s", rec.Code, rec.Body.String())
	}
	var mc MenuCOGSResponse
	decodeBody(t, rec, &mc)

	costByID := map[string]CostRow{}
	for _, r := range cost.Rows {
		costByID[r.MenuItemID] = r
	}

	if len(mc.MenuItems) == 0 {
		t.Fatal("menu-cogs returned no rows — fixture did not land")
	}
	for _, m := range mc.MenuItems {
		c, ok := costByID[m.MenuItemID]
		if !ok {
			t.Errorf("menu-cogs has %s but cost dropped it", m.MenuItemName)
			continue
		}
		if !centsEqual(c.UnitsSold, m.UnitsSold) {
			t.Errorf("%s: cost units_sold %v != menu-cogs units_sold %v",
				m.MenuItemName, c.UnitsSold, m.UnitsSold)
		}
		if c.IngredientCostTotal == nil {
			t.Errorf("%s: cost ingredient_cost_total is null but menu-cogs reports %v",
				m.MenuItemName, m.IngredientCostTotal)
			continue
		}
		if !centsEqual(*c.IngredientCostTotal, m.IngredientCostTotal) {
			t.Errorf("%s: cost ingredient_cost_total %v != menu-cogs %v (must agree to the cent)",
				m.MenuItemName, *c.IngredientCostTotal, m.IngredientCostTotal)
		}
	}

	// Side Salad: revenue is visible in cost even though menu-cogs omits the row.
	for _, m := range mc.MenuItems {
		if m.MenuItemName == "Side Salad" {
			t.Errorf("precondition changed: menu-cogs now returns no-recipe rows")
		}
	}
	salad := rowByName(t, cost, "Side Salad")
	if !centsEqual(salad.Revenue, 37.75) {
		t.Errorf("Side Salad revenue = %v, want 37.75 (revenue must not be silently dropped)", salad.Revenue)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Inherited semantics, pinned as observed behavior (NOT re-litigated here):
// the menu-cogs window_spend CTE filters neither on mercury_category nor on
// pending_purchases. period-summary DOES filter on mercury_category and DOES
// count pending rows. Cost inherits menu-cogs, so the two endpoints answer
// different questions by construction. This test documents that divergence so a
// future change to either side breaks loudly instead of silently.
// ─────────────────────────────────────────────────────────────────────────────
func TestCost_InheritsMenuCogs_NoCategoryFilter_NoPendingRows(t *testing.T) {
	pool := setupTestDB(t)
	seedCostFixture(t, pool)

	resp, err := queryCost(context.Background(), pool, costFixtureFrom, costFixtureTo)
	if err != nil {
		t.Fatalf("queryCost: %v", err)
	}

	// E1 carries mercury_category = NULL. Salmon Bowl's cost (42.12) is only
	// reachable if that NULL-category event was counted: without it the salmon
	// and rice spend from E1 vanishes and the cost collapses to 62.004->21.99
	// derived numbers. Assert the full inclusive value.
	bowl := rowByName(t, resp, "Salmon Bowl")
	if bowl.IngredientCostTotal == nil || !centsEqual(*bowl.IngredientCostTotal, 42.12) {
		t.Errorf("Salmon Bowl cost = %v, want 42.12 — a NULL mercury_category event MUST still be counted (inherited from menu-cogs)", bowl.IngredientCostTotal)
	}

	// The seeded pending_purchases row ($250, in window, COGS) must move nothing.
	// If pending rows ever start counting, Salmon Bowl's cost changes.
	var pendingCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM pending_purchases WHERE bank_tx_id = 'f2-pending-tx'`).Scan(&pendingCount); err != nil {
		t.Fatalf("pending count: %v", err)
	}
	if pendingCount != 1 {
		t.Fatalf("fixture precondition: expected 1 pending row, got %d", pendingCount)
	}
	if bowl.IngredientCostTotal != nil && !centsEqual(*bowl.IngredientCostTotal, 42.12) {
		t.Errorf("pending_purchases row leaked into cost: Salmon Bowl = %v", *bowl.IngredientCostTotal)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Handler wiring: no query params, window computed server-side, 200 + envelope.
// Seeds inside the LIVE window so this exercises the real date computation.
// ─────────────────────────────────────────────────────────────────────────────
func TestCostHandler_ServerComputedWindow(t *testing.T) {
	pool := setupTestDB(t)

	wantFrom, wantTo, wantWeeks := costWindow(time.Now())
	if wantFrom == "" || wantTo == "" {
		t.Fatalf("costWindow returned empty window (%q, %q, %d)", wantFrom, wantTo, wantWeeks)
	}

	// Place a purchase + a sale one day after `from`, safely inside the window.
	fromT, err := time.Parse("2006-01-02", wantFrom)
	if err != nil {
		t.Fatalf("parse from: %v", err)
	}
	inWindow := fromT.AddDate(0, 0, 1).Format("2006-01-02")

	vendorID := seedVendor(t, pool, "F2-Live-Vendor")
	pi := seedPurchaseItem(t, pool, "Live Salmon")
	mi := seedMenuItemFull(t, pool, "Live Bowl", "Bowls")
	ev := seedPurchaseEventCat(t, pool, vendorID, inWindow, 0.00, 40.00, nil)
	seedPurchaseLineItem(t, pool, ev, pi, "Live Salmon", 1, 40.00)
	seedRecipe(t, pool, mi, pi, 50.0) // 40.00 * .50 = 20.00
	seedDailyMenuSales(t, pool, mi, inWindow, 8, 80.00)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/cost", nil)
	rec := httptest.NewRecorder()
	CostHandler(pool).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp CostResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Window.From != wantFrom || resp.Window.To != wantTo || resp.Window.Weeks != wantWeeks {
		t.Errorf("Window = %+v, want {From:%s To:%s Weeks:%d}", resp.Window, wantFrom, wantTo, wantWeeks)
	}
	row := rowByName(t, resp, "Live Bowl")
	if row.IngredientCostTotal == nil || !centsEqual(*row.IngredientCostTotal, 20.00) {
		t.Errorf("Live Bowl cost = %v, want 20.00", row.IngredientCostTotal)
	}
	if !centsEqual(row.Revenue, 80.00) {
		t.Errorf("Live Bowl revenue = %v, want 80.00", row.Revenue)
	}
	if row.Margin == nil || !centsEqual(*row.Margin, 60.00) {
		t.Errorf("Live Bowl margin = %v, want 60.00", row.Margin)
	}
	if row.FoodCostPct == nil || !centsEqual(*row.FoodCostPct, 25.00) {
		t.Errorf("Live Bowl food_cost_pct = %v, want 25.00", row.FoodCostPct)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Sparse prod (A7): empty daily_menu_sales + empty recipes -> rows: [] (a JSON
// array, never null) and empty movers. This is the state F4's low-data card
// renders against.
// ─────────────────────────────────────────────────────────────────────────────
func TestCost_EmptyDataset_ReturnsEmptyArrays(t *testing.T) {
	pool := setupTestDB(t)

	resp, err := queryCost(context.Background(), pool, costFixtureFrom, costFixtureTo)
	if err != nil {
		t.Fatalf("queryCost: %v", err)
	}
	if len(resp.Rows) != 0 {
		t.Errorf("Rows = %+v, want empty", resp.Rows)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var probe struct {
		Rows   json.RawMessage `json:"rows"`
		Movers struct {
			ByMargin struct {
				Best json.RawMessage `json:"best"`
			} `json:"by_margin"`
		} `json:"movers"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(probe.Rows) != "[]" {
		t.Errorf("rows on the wire = %s, want []", string(probe.Rows))
	}
	if string(probe.Movers.ByMargin.Best) != "[]" {
		t.Errorf("movers.by_margin.best on the wire = %s, want []", string(probe.Movers.ByMargin.Best))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// G6 FIX 1 — negative revenue must be treated exactly like zero revenue.
//
// A refund can drive SUM(gross_amount) negative. A `revenue = 0` guard does not
// catch it, so the row published food_cost_pct = -500000 (cost 50 / revenue
// -0.01). Because by_food_cost_pct.best is ordered lowest-%-first, that nonsense
// value sorted to position 0 and would have top-billed a refunded dish as the
// BEST food-cost performer on the tab.
//
// The signed §2.3 rule reads "zero-revenue -> NULL"; this extends it to
// non-positive revenue, on the design's own stated intent ("never a
// divide-by-zero or Inf", "never a silent 0"). Flagged for operator ratification.
// ─────────────────────────────────────────────────────────────────────────────
func TestCost_NegativeRevenue_PctNullAndExcludedFromMovers(t *testing.T) {
	pool := setupTestDB(t)

	vendorID := seedVendor(t, pool, "F2-Refund-Vendor")
	pi := seedPurchaseItem(t, pool, "Refund Salmon")
	ev := seedPurchaseEventCat(t, pool, vendorID, "2026-05-11", 0.00, 100.00, nil)
	seedPurchaseLineItem(t, pool, ev, pi, "Refund Salmon", 1, 100.00)

	// Refunded Dish: 50% of 100.00 = 50.00 cost, against a NEGATIVE net revenue.
	refunded := seedMenuItemFull(t, pool, "Refunded Dish", "Specials")
	seedRecipe(t, pool, refunded, pi, 50.0)
	seedDailyMenuSales(t, pool, refunded, "2026-05-11", 1, -0.01)

	// A healthy row so the movers lists are non-trivial and we can prove the
	// refunded row is excluded rather than the lists just being empty.
	healthy := seedMenuItemFull(t, pool, "Healthy Dish", "Bowls")
	seedRecipe(t, pool, healthy, pi, 25.0) // 25.00 cost
	seedDailyMenuSales(t, pool, healthy, "2026-05-11", 10, 100.00)

	resp, err := queryCost(context.Background(), pool, costFixtureFrom, costFixtureTo)
	if err != nil {
		t.Fatalf("queryCost: %v", err)
	}

	// (i) food_cost_pct is JSON null on the wire.
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var envelope struct {
		Rows []map[string]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	found := false
	for _, r := range envelope.Rows {
		var name string
		_ = json.Unmarshal(r["menu_item_name"], &name)
		if name != "Refunded Dish" {
			continue
		}
		found = true
		if got := string(r["food_cost_pct"]); got != "null" {
			t.Errorf("Refunded Dish food_cost_pct on the wire = %s, want null", got)
		}
		// The row keeps its real cost and its real (negative) margin.
		var margin float64
		if err := json.Unmarshal(r["margin"], &margin); err != nil {
			t.Fatalf("margin not a number: %s", string(r["margin"]))
		}
		if !centsEqual(margin, -50.01) {
			t.Errorf("Refunded Dish margin = %v, want -50.01", margin)
		}
	}
	if !found {
		t.Fatal("Refunded Dish row missing from the wire payload")
	}

	// (ii) absent from BOTH by_food_cost_pct lists.
	refundedRow := rowByName(t, resp, "Refunded Dish")
	for _, id := range resp.Movers.ByFoodCostPct.Best {
		if id == refundedRow.MenuItemID {
			t.Errorf("negative-revenue row must not appear in by_food_cost_pct.best (got %v)", resp.Movers.ByFoodCostPct.Best)
		}
	}
	for _, id := range resp.Movers.ByFoodCostPct.Worst {
		if id == refundedRow.MenuItemID {
			t.Errorf("negative-revenue row must not appear in by_food_cost_pct.worst (got %v)", resp.Movers.ByFoodCostPct.Worst)
		}
	}
	// The healthy row is still ranked, so exclusion is targeted not blanket.
	healthyRow := rowByName(t, resp, "Healthy Dish")
	if len(resp.Movers.ByFoodCostPct.Best) != 1 || resp.Movers.ByFoodCostPct.Best[0] != healthyRow.MenuItemID {
		t.Errorf("by_food_cost_pct.best = %v, want exactly [Healthy Dish]", resp.Movers.ByFoodCostPct.Best)
	}
	// Negative margin still participates in by_margin (a real worst-mover).
	if len(resp.Movers.ByMargin.Worst) == 0 || resp.Movers.ByMargin.Worst[0] != refundedRow.MenuItemID {
		t.Errorf("by_margin.worst[0] = %v, want Refunded Dish (negative margins DO rank)", resp.Movers.ByMargin.Worst)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// G6 FIX 2 — the movers tie-break was unproven (reviewer mutation M3 survived:
// no fixture created a tie in either ordering).
//
// Note the subtlety that made M3 survive: the SQL already emits rows
// ORDER BY menu_item_name, and sort.SliceStable preserves input order, so with
// DB-ordered input the explicit name tie-break is redundant and invisible. This
// unit test therefore feeds computeMovers rows in DELIBERATELY NON-NAME ORDER,
// which is the only way to actually observe (and thus protect) the tie-break.
// ─────────────────────────────────────────────────────────────────────────────
func TestComputeMovers_TieBreakOnName_IndependentOfInputOrder(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	// Three-way tie in BOTH orderings: identical margin AND identical pct.
	// Supplied in reverse-name order to defeat sort stability.
	rows := []CostRow{
		{MenuItemID: "id-charlie", MenuItemName: "Charlie", Revenue: 100, IngredientCostTotal: f(25), Margin: f(75), FoodCostPct: f(25)},
		{MenuItemID: "id-bravo", MenuItemName: "Bravo", Revenue: 100, IngredientCostTotal: f(25), Margin: f(75), FoodCostPct: f(25)},
		{MenuItemID: "id-alpha", MenuItemName: "Alpha", Revenue: 100, IngredientCostTotal: f(25), Margin: f(75), FoodCostPct: f(25)},
	}

	m := computeMovers(rows)

	wantBest := []string{"id-alpha", "id-bravo", "id-charlie"}
	wantWorst := []string{"id-charlie", "id-bravo", "id-alpha"}

	eq := func(label string, got, want []string) {
		if len(got) != len(want) {
			t.Errorf("%s: len = %d, want %d (got %v)", label, len(got), len(want), got)
			return
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s[%d] = %s, want %s (full: %v)", label, i, got[i], want[i], got)
			}
		}
	}
	eq("tie by_food_cost_pct.best", m.ByFoodCostPct.Best, wantBest)
	eq("tie by_food_cost_pct.worst", m.ByFoodCostPct.Worst, wantWorst)
	eq("tie by_margin.best", m.ByMargin.Best, wantBest)
	eq("tie by_margin.worst", m.ByMargin.Worst, wantWorst)

	// Repeated calls must be byte-identical (no map iteration leaking in).
	for i := 0; i < 5; i++ {
		again := computeMovers(rows)
		eq("repeat by_margin.best", again.ByMargin.Best, wantBest)
		eq("repeat by_food_cost_pct.best", again.ByFoodCostPct.Best, wantBest)
	}
}

// End-to-end tie determinism through the DB path, across repeated calls.
func TestCost_TiedRows_DeterministicAcrossCalls(t *testing.T) {
	pool := setupTestDB(t)

	vendorID := seedVendor(t, pool, "F2-Tie-Vendor")
	pi := seedPurchaseItem(t, pool, "Tie Ingredient")
	ev := seedPurchaseEventCat(t, pool, vendorID, "2026-05-11", 0.00, 100.00, nil)
	seedPurchaseLineItem(t, pool, ev, pi, "Tie Ingredient", 1, 100.00)

	// Two items, identical allocation and identical revenue -> identical margin
	// AND identical food_cost_pct. Seeded in reverse-name order on purpose.
	beta := seedMenuItemFull(t, pool, "Tie Beta", "Bowls")
	alpha := seedMenuItemFull(t, pool, "Tie Alpha", "Bowls")
	seedRecipe(t, pool, beta, pi, 25.0)
	seedRecipe(t, pool, alpha, pi, 25.0)
	seedDailyMenuSales(t, pool, beta, "2026-05-11", 10, 100.00)
	seedDailyMenuSales(t, pool, alpha, "2026-05-11", 10, 100.00)

	var first CostResponse
	for i := 0; i < 5; i++ {
		resp, err := queryCost(context.Background(), pool, costFixtureFrom, costFixtureTo)
		if err != nil {
			t.Fatalf("queryCost call %d: %v", i, err)
		}
		// Confirm the tie is real before asserting the tie-break.
		a := rowByName(t, resp, "Tie Alpha")
		b := rowByName(t, resp, "Tie Beta")
		if a.Margin == nil || b.Margin == nil || !centsEqual(*a.Margin, *b.Margin) {
			t.Fatalf("fixture precondition: margins are not tied (%v vs %v)", a.Margin, b.Margin)
		}
		if a.FoodCostPct == nil || b.FoodCostPct == nil || !centsEqual(*a.FoodCostPct, *b.FoodCostPct) {
			t.Fatalf("fixture precondition: pcts are not tied (%v vs %v)", a.FoodCostPct, b.FoodCostPct)
		}
		if i == 0 {
			first = resp
			if got := resp.Movers.ByMargin.Best; len(got) != 2 || got[0] != alpha || got[1] != beta {
				t.Errorf("tied by_margin.best = %v, want [Tie Alpha, Tie Beta] (name-ascending tie-break)", got)
			}
			if got := resp.Movers.ByFoodCostPct.Best; len(got) != 2 || got[0] != alpha || got[1] != beta {
				t.Errorf("tied by_food_cost_pct.best = %v, want [Tie Alpha, Tie Beta]", got)
			}
			continue
		}
		if !reflect.DeepEqual(resp.Movers, first.Movers) {
			t.Fatalf("movers not deterministic across calls: call0=%+v call%d=%+v", first.Movers, i, resp.Movers)
		}
	}
}
