package recipes

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

// CostMoverPair is one ordering's best/worst id lists.
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
// before the current week, evaluated in America/Chicago.
func costWindow(now time.Time) (string, string, int) {
	return "", "", 0
}

// queryCost runs the cost/margin aggregation over [from, to].
func queryCost(ctx context.Context, pool *pgxpool.Pool, from, to string) (CostResponse, error) {
	return CostResponse{}, nil
}

// CostHandler serves GET /api/v1/inventory/cost (cookie-auth).
func CostHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, CostResponse{})
	}
}
