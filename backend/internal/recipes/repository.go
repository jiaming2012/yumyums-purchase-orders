package recipes

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrSumExceeds100 is returned by CreateRecipe / UpdateRecipeUsagePct when the new
// SUM(usage_pct) for the purchase_item exceeds 100. Caller is responsible for
// composing the 422 response (this layer doesn't know about HTTP).
var ErrSumExceeds100 = errors.New("recipes: sum_exceeds_100")

// ErrRecipeNotFound is returned by UpdateRecipeUsagePct / DeleteRecipe when the
// recipe id does not exist.
var ErrRecipeNotFound = errors.New("recipes: not_found")

// ListRecipes returns all recipes; if purchaseItemID is non-nil, filters to that ingredient.
// Joined to menu_items for the menu_group / menu_subgroup display fields (D-09).
func ListRecipes(ctx context.Context, pool *pgxpool.Pool, purchaseItemID *string) ([]RecipeWithMenu, error) {
	q := `SELECT r.id, r.menu_item_id, mi.name, mi.menu_group, mi.menu_subgroup,
	             r.purchase_item_id, r.usage_pct, r.updated_at
	      FROM recipes r
	      JOIN menu_items mi ON mi.id = r.menu_item_id`
	var rows pgx.Rows
	var err error
	if purchaseItemID != nil {
		rows, err = pool.Query(ctx, q+" WHERE r.purchase_item_id = $1 ORDER BY r.usage_pct DESC", *purchaseItemID)
	} else {
		rows, err = pool.Query(ctx, q+" ORDER BY mi.name")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RecipeWithMenu{}
	for rows.Next() {
		var r RecipeWithMenu
		if err := rows.Scan(&r.ID, &r.MenuItemID, &r.MenuItemName, &r.MenuGroup, &r.MenuSubgroup,
			&r.PurchaseItemID, &r.UsagePct, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateRecipe inserts a new recipe and returns its id plus the new SUM for the purchase_item.
// If the new SUM would exceed 100, the tx is rolled back and ErrSumExceeds100 is returned.
func CreateRecipe(ctx context.Context, pool *pgxpool.Pool, menuItemID, purchaseItemID string, usagePct float64) (id string, sumAfter float64, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	err = tx.QueryRow(ctx,
		`INSERT INTO recipes (menu_item_id, purchase_item_id, usage_pct, updated_at)
		 VALUES ($1, $2, $3, now()) RETURNING id::text`,
		menuItemID, purchaseItemID, usagePct,
	).Scan(&id)
	if err != nil {
		return "", 0, err
	}

	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(usage_pct), 0) FROM recipes WHERE purchase_item_id = $1`,
		purchaseItemID,
	).Scan(&sumAfter)
	if err != nil {
		return "", 0, err
	}
	if sumAfter > 100 {
		return "", sumAfter, ErrSumExceeds100
	}

	if err := tx.Commit(ctx); err != nil {
		return "", 0, err
	}
	return id, sumAfter, nil
}

// UpdateRecipeUsagePct sets the usage_pct on an existing recipe. Returns the purchase_item_id
// owning the recipe and the new SUM. ErrSumExceeds100 on sum > 100. ErrRecipeNotFound if id
// doesn't exist.
func UpdateRecipeUsagePct(ctx context.Context, pool *pgxpool.Pool, recipeID string, usagePct float64) (purchaseItemID string, sumAfter float64, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	err = tx.QueryRow(ctx,
		`UPDATE recipes SET usage_pct = $1, updated_at = now()
		 WHERE id = $2 RETURNING purchase_item_id::text`,
		usagePct, recipeID,
	).Scan(&purchaseItemID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, ErrRecipeNotFound
	}
	if err != nil {
		return "", 0, err
	}

	err = tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(usage_pct), 0) FROM recipes WHERE purchase_item_id = $1`,
		purchaseItemID,
	).Scan(&sumAfter)
	if err != nil {
		return "", 0, err
	}
	if sumAfter > 100 {
		return purchaseItemID, sumAfter, ErrSumExceeds100
	}

	if err := tx.Commit(ctx); err != nil {
		return "", 0, err
	}
	return purchaseItemID, sumAfter, nil
}

// DeleteRecipe removes a recipe row by id. Returns ErrRecipeNotFound if missing.
func DeleteRecipe(ctx context.Context, pool *pgxpool.Pool, recipeID string) error {
	ct, err := pool.Exec(ctx, `DELETE FROM recipes WHERE id = $1`, recipeID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrRecipeNotFound
	}
	return nil
}

// MergeMenuItem re-points all recipe rows from sourceMenuItemID to targetMenuItemID and
// deletes the source menu_items row. Returns rowsRePointed count. Mirrors the
// inventory.MergeItemsHandler tx pattern at backend/internal/inventory/handler.go:174-234.
// Returns error if source == target.
func MergeMenuItem(ctx context.Context, pool *pgxpool.Pool, sourceMenuItemID, targetMenuItemID string) (int, error) {
	if sourceMenuItemID == targetMenuItemID {
		return 0, fmt.Errorf("recipes: cannot_merge_into_self")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	ct, err := tx.Exec(ctx,
		`UPDATE recipes SET menu_item_id = $1, updated_at = now()
		 WHERE menu_item_id = $2`,
		targetMenuItemID, sourceMenuItemID,
	)
	if err != nil {
		return 0, err
	}
	rows := int(ct.RowsAffected())

	// Delete the source menu_items row.
	_, err = tx.Exec(ctx, `DELETE FROM menu_items WHERE id = $1`, sourceMenuItemID)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return rows, nil
}

// SumPerPurchaseItem returns the current SUM(usage_pct) for a purchase_item.
func SumPerPurchaseItem(ctx context.Context, pool *pgxpool.Pool, purchaseItemID string) (float64, error) {
	var sum float64
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(usage_pct), 0) FROM recipes WHERE purchase_item_id = $1`,
		purchaseItemID,
	).Scan(&sum)
	return sum, err
}

// LargestSiblingAllocation returns the largest sibling recipe row's menu_item_name + pct
// for the given purchase_item, excluding the recipe id passed in. Used to populate the
// 422 conflict_menu_item field per D-03.
func LargestSiblingAllocation(ctx context.Context, pool *pgxpool.Pool, purchaseItemID, excludeRecipeID string) (string, float64, error) {
	var name string
	var pct float64
	err := pool.QueryRow(ctx,
		`SELECT mi.name, r.usage_pct
		 FROM recipes r JOIN menu_items mi ON mi.id = r.menu_item_id
		 WHERE r.purchase_item_id = $1 AND r.id <> $2
		 ORDER BY r.usage_pct DESC LIMIT 1`,
		purchaseItemID, excludeRecipeID,
	).Scan(&name, &pct)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, nil
	}
	return name, pct, err
}

// ListIngredientsWithSpend returns one row per purchase_item that has either
// (a) recipe rows or (b) non-zero spend in [from, to]. Each row carries the
// last-week spend (tax-inclusive per D-11) and the nested recipes joined to
// menu_items for D-09 disambiguation.
//
// Implementation: two queries — first for the ingredient summary rows
// (purchase_items + window spend + sum_pct), second for the nested recipes joined
// to menu_items. Sorted by last_week_spend DESC then description ASC.
func ListIngredientsWithSpend(ctx context.Context, pool *pgxpool.Pool, from, to string) ([]IngredientWithSpend, error) {
	rows, err := pool.Query(ctx, `
WITH window_spend AS (
  SELECT
    pli.purchase_item_id,
    SUM((pli.quantity * pli.price) *
        COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)) AS spend_incl_tax
  FROM purchase_line_items pli
  JOIN purchase_events pe ON pe.id = pli.purchase_event_id
  WHERE pe.event_date BETWEEN $1 AND $2
    AND pli.purchase_item_id IS NOT NULL
  GROUP BY pli.purchase_item_id
)
SELECT
  pi.id::text                                            AS purchase_item_id,
  pi.description                                         AS description,
  ROUND(COALESCE(ws.spend_incl_tax, 0)::numeric, 2)      AS last_week_spend,
  COALESCE((SELECT SUM(r2.usage_pct) FROM recipes r2 WHERE r2.purchase_item_id = pi.id), 0) AS sum_pct
FROM purchase_items pi
LEFT JOIN window_spend ws ON ws.purchase_item_id = pi.id
WHERE EXISTS (SELECT 1 FROM recipes WHERE purchase_item_id = pi.id)
   OR COALESCE(ws.spend_incl_tax, 0) > 0
ORDER BY COALESCE(ws.spend_incl_tax, 0) DESC, pi.description ASC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ingredients := []IngredientWithSpend{}
	indexByID := map[string]int{}
	for rows.Next() {
		var ing IngredientWithSpend
		if err := rows.Scan(&ing.PurchaseItemID, &ing.Description, &ing.LastWeekSpend, &ing.SumPct); err != nil {
			return nil, err
		}
		ing.Recipes = []RecipeWithMenu{}
		indexByID[ing.PurchaseItemID] = len(ingredients)
		ingredients = append(ingredients, ing)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Second query: load recipes for these ingredients in one shot, distribute into rows.
	if len(ingredients) == 0 {
		return ingredients, nil
	}
	ids := make([]string, len(ingredients))
	for i, ing := range ingredients {
		ids[i] = ing.PurchaseItemID
	}
	recRows, err := pool.Query(ctx, `
SELECT r.id, r.menu_item_id, mi.name, mi.menu_group, mi.menu_subgroup,
       r.purchase_item_id::text, r.usage_pct, r.updated_at
FROM recipes r
JOIN menu_items mi ON mi.id = r.menu_item_id
WHERE r.purchase_item_id::text = ANY($1)
ORDER BY r.usage_pct DESC`, ids)
	if err != nil {
		return nil, err
	}
	defer recRows.Close()
	for recRows.Next() {
		var r RecipeWithMenu
		if err := recRows.Scan(&r.ID, &r.MenuItemID, &r.MenuItemName, &r.MenuGroup, &r.MenuSubgroup,
			&r.PurchaseItemID, &r.UsagePct, &r.UpdatedAt); err != nil {
			return nil, err
		}
		if idx, ok := indexByID[r.PurchaseItemID]; ok {
			ingredients[idx].Recipes = append(ingredients[idx].Recipes, r)
		}
	}
	return ingredients, recRows.Err()
}
