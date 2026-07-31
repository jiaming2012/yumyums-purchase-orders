package recipes

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/users"
)

// ─────────────────────────────────────────────────────────────────────────────
// GET /api/v1/inventory/cost — per-menu-item cost, margin and food-cost-%.
// Design: .night-crew/knowledge/designs/prove-surface-gating-and-endpoints.md
// §2.1 (shared rules) + §2.3 (this endpoint), SIGNED 2026-07-20.
//
// Cookie-auth, logged-in-only. NO permission check here by design: the per-tab
// grant gate is a separate change and owns that surface.
//
// Relationship to menu-cogs (Phase 999.2, a live sales-processor contract):
// this endpoint REUSES menu-cogs's window_spend + alloc CTEs verbatim so the two
// agree to the cent on ingredient cost, and it widens menu-cogs's menu_units CTE
// to also select SUM(daily_menu_sales.gross_amount) as revenue. menu-cogs itself
// is not modified in any way.
//
// INHERITED SEMANTICS (stated, not re-litigated). window_spend reads
// purchase_events/purchase_line_items only and filters on neither
// mercury_category nor confirmation state. Consequently this endpoint:
//   - counts events whose mercury_category is NULL or non-COGS, and
//   - ignores pending_purchases entirely.
//
// inventory.PeriodSummaryHandler does the opposite on both counts. The two
// endpoints therefore answer different questions by construction; this is
// menu-cogs's pre-existing behavior, inherited deliberately so that Cost and
// menu-cogs reconcile. TestCost_InheritsMenuCogs_NoCategoryFilter_NoPendingRows
// pins it so a future change to either side breaks loudly.
// ─────────────────────────────────────────────────────────────────────────────

// costWindowWeeks is the fixed window length per design §2.1 (Assumption A1):
// 12 complete ISO weeks, no query params in v1.
const costWindowWeeks = 12

// CostWindow is the window envelope of the /inventory/cost response.
type CostWindow struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Weeks int    `json:"weeks"`
}

// CostRow is one menu item's cost/margin row per design §2.3.
//
// IngredientCostTotal / Margin / FoodCostPct are pointers so the wire can carry
// JSON null. Two distinct null cases, both required by §2.3's "never a silent 0":
//
//   - No recipe rows at all -> cost, margin and pct are all null and Unallocated
//     carries menu-cogs's "no recipe" reason string. Revenue and units are still
//     reported, so the money is visible rather than dropped.
//   - Revenue is exactly 0 (comped / retired items) -> pct is null (the signed
//     zero-revenue rule). Cost and the resulting negative margin are still real
//     numbers and are still reported.
type CostRow struct {
	MenuItemID          string   `json:"menu_item_id"`
	MenuItemName        string   `json:"menu_item_name"`
	MenuGroup           string   `json:"menu_group"`
	UnitsSold           float64  `json:"units_sold"`
	Revenue             float64  `json:"revenue"`
	IngredientCostTotal *float64 `json:"ingredient_cost_total"`
	Margin              *float64 `json:"margin"`
	FoodCostPct         *float64 `json:"food_cost_pct"`
	Unallocated         *string  `json:"unallocated"`
}

// CostMoverPair is one ordering's best/worst id lists. Both lists are the FULL
// participating ordering (reversed views of each other); the client slices to
// however many it renders.
type CostMoverPair struct {
	Best  []string `json:"best"`
	Worst []string `json:"worst"`
}

// CostMovers carries both orderings per design §2.3.
type CostMovers struct {
	ByFoodCostPct CostMoverPair `json:"by_food_cost_pct"`
	ByMargin      CostMoverPair `json:"by_margin"`
}

// CostResponse is the full /inventory/cost envelope.
type CostResponse struct {
	Window CostWindow `json:"window"`
	Rows   []CostRow  `json:"rows"`
	Movers CostMovers `json:"movers"`
}

// costWindow returns the fixed 12-complete-ISO-week window ending the Sunday
// before the current week, evaluated in the APP timezone (users.DefaultTimezone
// — America/New_York, ledger T-26 decision 83).
//
// The timezone is explicit (not server-local) because every other date boundary
// in this system — the Monday drift check, period-summary's payroll week,
// purchasing's CurrentWeekStart — is stated in that same one zone. Falling back
// to the passed-in location on tzdata failure keeps the endpoint serving rather
// than 500ing.
//
// 🛑 CHANGEOVER: ON THE DEPLOY THAT FOLLOWS THIS MERGE — DATE TBD. This window
// is America/Chicago in production until then, so between 23:00 Sunday Chicago
// and midnight Monday New York it names the previous 12-week window. Merging
// does not move it; no deploy is scheduled as of this writing. Fix-forward: cost
// figures produced before that deploy are NOT restated. To date this changeover:
// find the first deploy after this comment's commit. See migration
// 0072_app_timezone_new_york.sql, whose header carries the same instruction.
func costWindow(now time.Time) (string, string, int) {
	if loc, err := time.LoadLocation(users.DefaultTimezone); err == nil {
		now = now.In(loc)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// ISO weeks start Monday. Go's Weekday() has Sunday=0, so shift.
	offsetFromMonday := (int(today.Weekday()) + 6) % 7
	weekStart := today.AddDate(0, 0, -offsetFromMonday)

	to := weekStart.AddDate(0, 0, -1)                   // the Sunday just past
	from := weekStart.AddDate(0, 0, -7*costWindowWeeks) // 12 Mondays back

	return from.Format("2006-01-02"), to.Format("2006-01-02"), costWindowWeeks
}

// costQuery is menu-cogs's window_spend + alloc, widened with a menu_sales CTE
// that adds revenue alongside units.
//
// Row inclusion: a menu item appears if it had sales in the window OR has at
// least one recipe row. Items with neither are omitted (they would be all-null
// noise). Note this is a deliberate widening of menu-cogs's row set, which
// INNER-JOINs alloc and therefore drops sales-but-no-recipe items entirely —
// dropping them here would hide real revenue, which §2.1 forbids.
//
// Rounding sites, in order:
//  1. ingredient_cost_total = ROUND(SUM(alloc_cost), 2)  — same site and same
//     expression as menu-cogs, which is why the two agree to the cent.
//  2. revenue = ROUND(SUM(gross_amount), 2) — gross_amount is already
//     NUMERIC(10,2); the ROUND is defensive against SUM widening.
//  3. margin and food_cost_pct are derived from the ALREADY-ROUNDED cost and
//     revenue, so the invariant margin == revenue - ingredient_cost_total holds
//     on the numbers the client actually receives, then each is itself rounded
//     to 2 decimals.
//
// The non-positive-revenue guard is an explicit `WHEN revenue <= 0 THEN NULL`,
// not NULLIF: revenue stays in the numeric domain (exact), and a NULLIF-style
// guard risks yielding a silent 0 rather than the specified null.
//
// Signed §2.3 words the rule as "zero-revenue -> NULL". The `<= 0` here EXTENDS
// it to negative revenue, which refunds produce (SUM(gross_amount) can go
// negative). With a bare `= 0` guard a refunded dish published
// food_cost_pct = -500000 and, because by_food_cost_pct.best ranks lowest-first,
// sorted to the top of the "best food cost %" strip. Reading the extension as
// the design's evident intent ("never a divide-by-zero or Inf", "never a silent
// 0") rather than a new decision — flagged for operator ratification at triage.
//
// ── KNOWN GAP, DELIBERATELY NOT FIXED HERE (routed to the operator) ──
// A menu item that HAS recipe rows but whose ingredients saw NO purchases inside
// the window yields alloc_cost = 0, so this query publishes
// ingredient_cost_total = 0, margin = revenue, food_cost_pct = 0 — i.e. a
// flattering 0% food cost / 100% margin presented as fact. That is the same
// class of meaningless number that justified returning NULL (not 0) for the
// no-recipe case above, and it is arguably the MORE likely production case,
// since bulk buys routinely land outside a rolling 12-week window.
//
// It is not fixed here because the honest fix needs a reason string, and
// menu-cogs's vocabulary supplies only "no recipe" and "partial allocation
// (X%)" — neither covers "recipe exists, zero window spend". Coining a third
// string is a design amendment to §2.3, not an implementation choice, so the
// behavior is left as-is pending that decision. Do not mistake the current
// output for intended behavior.
const costQuery = `
WITH
window_spend AS (
  SELECT
    pli.purchase_item_id,
    SUM(
      (pli.quantity * pli.price) *
      COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)
    ) AS spend_incl_tax
  FROM purchase_line_items pli
  JOIN purchase_events pe ON pe.id = pli.purchase_event_id
  WHERE pe.event_date BETWEEN $1 AND $2
    AND pli.purchase_item_id IS NOT NULL
  GROUP BY pli.purchase_item_id
),
menu_sales AS (
  SELECT dms.menu_item_id,
         SUM(dms.units_sold)   AS units_sold,
         SUM(dms.gross_amount) AS revenue
  FROM daily_menu_sales dms
  WHERE dms.business_date BETWEEN $1 AND $2
  GROUP BY dms.menu_item_id
),
alloc AS (
  SELECT r.menu_item_id,
         SUM(COALESCE(ws.spend_incl_tax, 0) * (r.usage_pct / 100.0)) AS alloc_cost
  FROM recipes r
  LEFT JOIN window_spend ws ON ws.purchase_item_id = r.purchase_item_id
  GROUP BY r.menu_item_id
),
base AS (
  SELECT
    mi.id::text                             AS menu_item_id,
    mi.name                                 AS menu_item_name,
    mi.menu_group                           AS menu_group,
    COALESCE(ms.units_sold, 0)::float8      AS units_sold,
    ROUND(COALESCE(ms.revenue, 0)::numeric, 2) AS revenue,
    CASE WHEN a.menu_item_id IS NULL THEN NULL
         ELSE ROUND(COALESCE(a.alloc_cost, 0)::numeric, 2)
    END                                     AS ingredient_cost_total
  FROM menu_items mi
  LEFT JOIN menu_sales ms ON ms.menu_item_id = mi.id
  LEFT JOIN alloc a       ON a.menu_item_id  = mi.id
  WHERE ms.menu_item_id IS NOT NULL OR a.menu_item_id IS NOT NULL
)
SELECT
  menu_item_id,
  menu_item_name,
  menu_group,
  units_sold,
  revenue,
  ingredient_cost_total,
  CASE WHEN ingredient_cost_total IS NULL THEN NULL
       ELSE ROUND(revenue - ingredient_cost_total, 2)
  END AS margin,
  CASE WHEN ingredient_cost_total IS NULL THEN NULL
       WHEN revenue <= 0                  THEN NULL
       ELSE ROUND(ingredient_cost_total / revenue * 100, 2)
  END AS food_cost_pct,
  CASE WHEN ingredient_cost_total IS NULL THEN 'no recipe'
       ELSE NULL
  END AS unallocated
FROM base
ORDER BY menu_item_name`

// queryCost runs the cost/margin aggregation over [from, to] and computes both
// movers orderings.
func queryCost(ctx context.Context, pool *pgxpool.Pool, from, to string) (CostResponse, error) {
	resp := CostResponse{
		Rows: []CostRow{},
		Movers: CostMovers{
			ByFoodCostPct: CostMoverPair{Best: []string{}, Worst: []string{}},
			ByMargin:      CostMoverPair{Best: []string{}, Worst: []string{}},
		},
	}

	rows, err := pool.Query(ctx, costQuery, from, to)
	if err != nil {
		return resp, err
	}
	defer rows.Close()

	for rows.Next() {
		var r CostRow
		if err := rows.Scan(
			&r.MenuItemID, &r.MenuItemName, &r.MenuGroup,
			&r.UnitsSold, &r.Revenue,
			&r.IngredientCostTotal, &r.Margin, &r.FoodCostPct, &r.Unallocated,
		); err != nil {
			return resp, err
		}
		resp.Rows = append(resp.Rows, r)
	}
	if err := rows.Err(); err != nil {
		return resp, err
	}

	resp.Movers = computeMovers(resp.Rows)
	return resp, nil
}

// computeMovers builds both orderings per §2.3.
//
//   - by_food_cost_pct: best = lowest %, worst = highest %. Rows with a null %
//     (non-positive revenue, or no recipe) are EXCLUDED — there is no % to rank
//     on. The `Revenue > 0` half of the guard is redundant with the SQL's
//     `revenue <= 0 -> NULL` and is kept deliberately: ranking is where a bogus
//     % does real damage (a -500000% refund sorts to "best"), so the invariant
//     is enforced locally rather than trusted from one query away.
//   - by_margin: best = highest dollars, worst = lowest. Rows with a null margin
//     (no recipe) are excluded, but zero-revenue rows DO participate: a negative
//     margin is a real worst-mover.
//
// Ties break on menu item name so the two strips are deterministic.
func computeMovers(rows []CostRow) CostMovers {
	type entry struct {
		id   string
		name string
		val  float64
	}

	order := func(items []entry, ascending bool) ([]string, []string) {
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].val != items[j].val {
				if ascending {
					return items[i].val < items[j].val
				}
				return items[i].val > items[j].val
			}
			return items[i].name < items[j].name
		})
		best := make([]string, 0, len(items))
		for _, it := range items {
			best = append(best, it.id)
		}
		worst := make([]string, 0, len(best))
		for i := len(best) - 1; i >= 0; i-- {
			worst = append(worst, best[i])
		}
		return best, worst
	}

	pctItems := make([]entry, 0, len(rows))
	marginItems := make([]entry, 0, len(rows))
	for _, r := range rows {
		if r.FoodCostPct != nil && r.Revenue > 0 {
			pctItems = append(pctItems, entry{r.MenuItemID, r.MenuItemName, *r.FoodCostPct})
		}
		if r.Margin != nil {
			marginItems = append(marginItems, entry{r.MenuItemID, r.MenuItemName, *r.Margin})
		}
	}

	// Lowest food-cost-% is best; highest margin is best.
	pctBest, pctWorst := order(pctItems, true)
	marginBest, marginWorst := order(marginItems, false)

	return CostMovers{
		ByFoodCostPct: CostMoverPair{Best: pctBest, Worst: pctWorst},
		ByMargin:      CostMoverPair{Best: marginBest, Worst: marginWorst},
	}
}

// CostHandler serves GET /api/v1/inventory/cost (cookie-auth).
// v1 takes no query params: the window is fixed and computed server-side.
func CostHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, weeks := costWindow(time.Now())

		resp, err := queryCost(r.Context(), pool, from, to)
		if err != nil {
			slog.Error("Cost query", "error", err, "from", from, "to", to)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		resp.Window = CostWindow{From: from, To: to, Weeks: weeks}

		writeJSON(w, http.StatusOK, resp)
	}
}
