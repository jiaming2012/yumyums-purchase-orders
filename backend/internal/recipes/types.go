package recipes

import "time"

// Recipe is a single (menu_item_id, purchase_item_id, usage_pct) row from the recipes table.
type Recipe struct {
	ID             string    `json:"id"`
	MenuItemID     string    `json:"menu_item_id"`
	PurchaseItemID string    `json:"purchase_item_id"`
	UsagePct       float64   `json:"usage_pct"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RecipeWithMenu joins recipes to menu_items for list rendering (per D-09: include
// menu_group + menu_subgroup for disambiguation).
type RecipeWithMenu struct {
	ID             string    `json:"id"`
	MenuItemID     string    `json:"menu_item_id"`
	MenuItemName   string    `json:"menu_item_name"`
	MenuGroup      string    `json:"menu_group"`
	MenuSubgroup   *string   `json:"menu_subgroup,omitempty"`
	PurchaseItemID string    `json:"purchase_item_id"`
	UsagePct       float64   `json:"usage_pct"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// IngredientWithSpend is the per-ingredient row the Recipes tab list endpoint returns.
// One row per purchase_item, with last-week spend computed from purchase_line_items.
type IngredientWithSpend struct {
	PurchaseItemID string           `json:"purchase_item_id"`
	Description    string           `json:"description"`
	LastWeekSpend  float64          `json:"last_week_spend"`
	SumPct         float64          `json:"sum_pct"`
	Recipes        []RecipeWithMenu `json:"recipes"`
}

// MenuCOGSRow is one row of the menu-cogs response per D-16.
type MenuCOGSRow struct {
	MenuItemID            string            `json:"menu_item_id"`
	ToastMasterID         string            `json:"toast_master_id"`
	MenuItemName          string            `json:"menu_item_name"`
	MenuGroup             string            `json:"menu_group"`
	MenuSubgroup          *string           `json:"menu_subgroup,omitempty"`
	UnitsSold             float64           `json:"units_sold"`
	IngredientCostPerUnit *float64          `json:"ingredient_cost_per_unit"` // nil when units_sold=0 (Pitfall 5)
	IngredientCostTotal   float64           `json:"ingredient_cost_total"`
	Ingredients           []IngredientAlloc `json:"ingredients,omitempty"` // only when ?breakdown=true
}

// IngredientAlloc is the per-ingredient detail under a menu item (only with ?breakdown=true).
type IngredientAlloc struct {
	PurchaseItemDescription string  `json:"purchase_item_description"`
	UsagePct                float64 `json:"usage_pct"`
	AllocatedCost           float64 `json:"allocated_cost"`
}

// MenuCOGSResponse is the full menu-cogs envelope. UnallocatedCogs has dual shape per D-17:
// default mode = number (UnallocatedCogs); breakdown mode = struct (Unallocated). Exactly
// one of the two is non-nil; controlled by ?breakdown=true.
type MenuCOGSResponse struct {
	From            string                `json:"from"`
	To              string                `json:"to"`
	MenuItems       []MenuCOGSRow         `json:"menu_items"`
	UnallocatedCogs *float64              `json:"unallocated_cogs,omitempty"`
	Unallocated     *UnallocatedBreakdown `json:"unallocated,omitempty"`
}

// UnallocatedBreakdown is the ?breakdown=true shape per D-17.
type UnallocatedBreakdown struct {
	Total        float64             `json:"total"`
	ByIngredient []UnallocatedDetail `json:"by_ingredient"`
}

type UnallocatedDetail struct {
	PurchaseItemDescription string  `json:"purchase_item_description"`
	Amount                  float64 `json:"amount"`
	Reason                  string  `json:"reason"` // "no recipe" or "partial allocation (X%)"
}

// DriftCheckResult is the structured payload persisted in drift_check_results.payload (JSONB).
// Used by Plan 04 (scheduler) to write and Plan 04 again (DriftBannerHandler) to read.
type DriftCheckResult struct {
	WeekStart string         `json:"week_start"`
	Sections  []DriftSection `json:"sections"`
}

type DriftSection struct {
	Kind    string      `json:"kind"`    // "unallocated" | "divergence" | "zero_sales"
	Heading string      `json:"heading"` // e.g. "2 unallocated"
	Items   []DriftItem `json:"items"`
}

type DriftItem struct {
	PurchaseItemID string   `json:"purchase_item_id,omitempty"`
	MenuItemID     string   `json:"menu_item_id,omitempty"`
	Label          string   `json:"label"` // e.g. "Chicken Thighs ($89 unalloc)"
	AmountUnalloc  *float64 `json:"amount_unalloc,omitempty"`
	ConfiguredPct  *float64 `json:"configured_pct,omitempty"`
	ActualPct      *float64 `json:"actual_pct,omitempty"`
}

// HasDrift returns true if any section has at least one item.
func (r *DriftCheckResult) HasDrift() bool {
	for _, s := range r.Sections {
		if len(s.Items) > 0 {
			return true
		}
	}
	return false
}

// TotalFlagged returns the total count across all sections.
func (r *DriftCheckResult) TotalFlagged() int {
	n := 0
	for _, s := range r.Sections {
		n += len(s.Items)
	}
	return n
}
