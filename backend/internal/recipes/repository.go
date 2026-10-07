package recipes

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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

// ErrMergeTargetNotFound is returned by MergeMenuItem when the target id names
// no menu_items row. Nothing has been written when it is returned.
var ErrMergeTargetNotFound = errors.New("recipes: target_not_found")

// ErrMergeSourceNotFound is returned by MergeMenuItem when the source id names
// no menu_items row. Nothing has been written when it is returned.
var ErrMergeSourceNotFound = errors.New("recipes: source_not_found")

// ErrMergeIntoSelf is returned by MergeMenuItem when source and target are the
// same dish.
var ErrMergeIntoSelf = errors.New("recipes: cannot_merge_into_self")

// ErrBadID is returned when an id that must be a uuid is not one. It is
// returned before any query runs.
var ErrBadID = errors.New("recipes: bad_id")

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

// MergeMenuItem re-points every row that names sourceMenuItemID to
// targetMenuItemID — recipes, and (card I3, decision 194) the campaigns and QR
// codes whose item is the source dish — then deletes the source menu_items row,
// all in one transaction. Mirrors the inventory.MergeItemsHandler tx pattern at
// backend/internal/inventory/handler.go:174-234. Returns error if source == target.
//
// The returned count is the TOTAL rows re-pointed across the three tables
// (recipes + campaigns_admin + qr_codes); the handler ships it as
// `rows_re_pointed`.
//
// The re-point is the contract; migration 0086's ON DELETE SET NULL on the two
// item_id columns is only the backstop for a table that forgets this path.
//
// Every wrong input is refused before anything is written, each with its own
// sentinel (the handler's answer in brackets):
//   - an id that is not a 36-character hyphenated uuid → ErrBadID (400 bad_id)
//   - source and target the same dish → ErrMergeIntoSelf       (400 cannot_merge_into_self)
//   - a target that names no dish     → ErrMergeTargetNotFound (404 target_not_found)
//   - a source that names no dish     → ErrMergeSourceNotFound (404 source_not_found)
func MergeMenuItem(ctx context.Context, pool *pgxpool.Pool, sourceMenuItemID, targetMenuItemID string) (int, error) {
	// Both ids are parsed before the transaction (card K2, B-485): a non-uuid
	// used to reach Postgres, come back 22P02 and answer 500. Only the plain
	// 36-character form is an id: uuid.Parse also takes `urn:uuid:` and any
	// 38-character string without checking its first and last byte, so
	// "X<uuid>Y" would parse to the dish inside it and be merged away. That
	// refuses the 32-hex and {braced} spellings too, which Postgres itself
	// would take for a real dish — chosen, and pinned by
	// TestMergeMenuItem_BadIDIs400.
	sourceID, err := uuid.Parse(sourceMenuItemID)
	if err != nil || len(sourceMenuItemID) != 36 {
		return 0, fmt.Errorf("%w: source_menu_item_id", ErrBadID)
	}
	targetID, err := uuid.Parse(targetMenuItemID)
	if err != nil || len(targetMenuItemID) != 36 {
		return 0, fmt.Errorf("%w: target_menu_item_id", ErrBadID)
	}
	if sourceID == targetID {
		return 0, ErrMergeIntoSelf
	}
	// The canonical form from here on, so the queries see what was compared.
	sourceMenuItemID, targetMenuItemID = sourceID.String(), targetID.String()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// The target must be a dish that exists (card J2, B-479). Without this read
	// an UNATTACHED source re-pointed 0 rows, was deleted — its daily_menu_sales
	// with it — and the call answered 200; an attached one hit the campaign FK
	// and answered 500. FOR SHARE holds the target row until commit, so it
	// cannot be deleted between this read and the re-points below. It runs
	// BEFORE any UPDATE so both shapes get the same refusal.
	var targetExists int
	if err := tx.QueryRow(ctx,
		`SELECT 1 FROM menu_items WHERE id = $1 FOR SHARE`, targetMenuItemID,
	).Scan(&targetExists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrMergeTargetNotFound
		}
		return 0, err
	}

	// The source must be a dish too (card K2, B-485). A source that named no
	// dish re-pointed 0 rows, deleted 0 rows and answered 200 — a merge that
	// did nothing, reported as done. Read AFTER the target's FOR SHARE, inside
	// the same transaction; FOR UPDATE because the row is deleted below.
	var sourceExists int
	if err := tx.QueryRow(ctx,
		`SELECT 1 FROM menu_items WHERE id = $1 FOR UPDATE`, sourceMenuItemID,
	).Scan(&sourceExists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrMergeSourceNotFound
		}
		return 0, err
	}

	ct, err := tx.Exec(ctx,
		`UPDATE recipes SET menu_item_id = $1, updated_at = now()
		 WHERE menu_item_id = $2`,
		targetMenuItemID, sourceMenuItemID,
	)
	if err != nil {
		return 0, err
	}
	rows := int(ct.RowsAffected())

	// Campaigns and codes follow the dish to its survivor. Without these two
	// statements the DELETE below would blank their item (0086) — or, before
	// 0086, be refused with 23503 campaigns_admin_item_id_fkey.
	ct, err = tx.Exec(ctx,
		`UPDATE campaigns_admin SET item_id = $1, updated_at = now()
		 WHERE item_id = $2`,
		targetMenuItemID, sourceMenuItemID,
	)
	if err != nil {
		return 0, err
	}
	rows += int(ct.RowsAffected())

	ct, err = tx.Exec(ctx,
		`UPDATE qr_codes SET item_id = $1 WHERE item_id = $2`,
		targetMenuItemID, sourceMenuItemID,
	)
	if err != nil {
		return 0, err
	}
	rows += int(ct.RowsAffected())

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
  item_display_name(pi.id, pi.description)               AS display_name,
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
		if err := rows.Scan(&ing.PurchaseItemID, &ing.Description, &ing.DisplayName, &ing.LastWeekSpend, &ing.SumPct); err != nil {
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
