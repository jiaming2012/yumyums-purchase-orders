package recipes

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepository_CreateRecipe_HappyPath(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	menuItemID := seedMenuItem(t, pool, "Chicken Bowl")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken Thighs")

	id, sumAfter, err := CreateRecipe(ctx, pool, menuItemID, purchaseItemID, 50.0)
	if err != nil {
		t.Fatalf("CreateRecipe: %v", err)
	}
	if id == "" {
		t.Fatalf("expected non-empty id")
	}
	if sumAfter != 50.0 {
		t.Fatalf("expected sumAfter=50.0, got %v", sumAfter)
	}

	// SumPerPurchaseItem agrees.
	got, err := SumPerPurchaseItem(ctx, pool, purchaseItemID)
	if err != nil {
		t.Fatalf("SumPerPurchaseItem: %v", err)
	}
	if got != 50.0 {
		t.Fatalf("SumPerPurchaseItem: expected 50.0, got %v", got)
	}
}

func TestRepository_CreateRecipe_SumExceeds100_Rollsback(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	menuItemA := seedMenuItem(t, pool, "Bowl A")
	menuItemB := seedMenuItem(t, pool, "Bowl B")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken Thighs")

	// First create at 60%.
	_, _, err := CreateRecipe(ctx, pool, menuItemA, purchaseItemID, 60.0)
	if err != nil {
		t.Fatalf("first CreateRecipe: %v", err)
	}

	// Second create at 50% would push sum to 110 — must rollback.
	_, sumAfter, err := CreateRecipe(ctx, pool, menuItemB, purchaseItemID, 50.0)
	if !errors.Is(err, ErrSumExceeds100) {
		t.Fatalf("expected ErrSumExceeds100, got err=%v, sumAfter=%v", err, sumAfter)
	}

	// Verify DB still has only the first row (sum is 60, not 110).
	got, err := SumPerPurchaseItem(ctx, pool, purchaseItemID)
	if err != nil {
		t.Fatalf("SumPerPurchaseItem: %v", err)
	}
	if got != 60.0 {
		t.Fatalf("expected sum=60 after rollback, got %v", got)
	}
}

func TestRepository_UpdateRecipeUsagePct_ReturnsNewSum(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	menuItemID := seedMenuItem(t, pool, "Bowl")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken")
	recipeID := seedRecipe(t, pool, menuItemID, purchaseItemID, 50.0)

	piID, sumAfter, err := UpdateRecipeUsagePct(ctx, pool, recipeID, 75.0)
	if err != nil {
		t.Fatalf("UpdateRecipeUsagePct: %v", err)
	}
	if piID != purchaseItemID {
		t.Fatalf("expected returned purchaseItemID=%v, got %v", purchaseItemID, piID)
	}
	if sumAfter != 75.0 {
		t.Fatalf("expected sumAfter=75.0, got %v", sumAfter)
	}
}

func TestRepository_UpdateRecipeUsagePct_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	// Random UUID that doesn't exist.
	_, _, err := UpdateRecipeUsagePct(ctx, pool, "00000000-0000-0000-0000-000000000000", 50.0)
	if !errors.Is(err, ErrRecipeNotFound) {
		t.Fatalf("expected ErrRecipeNotFound, got %v", err)
	}
}

func TestRepository_DeleteRecipe_RemovesRow(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	menuItemID := seedMenuItem(t, pool, "Bowl")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken")
	recipeID := seedRecipe(t, pool, menuItemID, purchaseItemID, 50.0)

	if err := DeleteRecipe(ctx, pool, recipeID); err != nil {
		t.Fatalf("DeleteRecipe: %v", err)
	}

	got, err := SumPerPurchaseItem(ctx, pool, purchaseItemID)
	if err != nil {
		t.Fatalf("SumPerPurchaseItem: %v", err)
	}
	if got != 0.0 {
		t.Fatalf("expected sum=0 after delete, got %v", got)
	}

	// Deleting again returns ErrRecipeNotFound.
	if err := DeleteRecipe(ctx, pool, recipeID); !errors.Is(err, ErrRecipeNotFound) {
		t.Fatalf("expected ErrRecipeNotFound on second delete, got %v", err)
	}
}

func TestRepository_MergeMenuItem_RePointsRows(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	menuItemA := seedMenuItem(t, pool, "Old Bowl")
	menuItemB := seedMenuItem(t, pool, "New Bowl")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken")
	recipeID := seedRecipe(t, pool, menuItemA, purchaseItemID, 50.0)

	rowsRePointed, err := MergeMenuItem(ctx, pool, menuItemA, menuItemB)
	if err != nil {
		t.Fatalf("MergeMenuItem: %v", err)
	}
	if rowsRePointed != 1 {
		t.Fatalf("expected 1 row re-pointed, got %v", rowsRePointed)
	}

	// Verify recipe.menu_item_id is now B.
	var nowMenuItemID string
	if err := pool.QueryRow(ctx, `SELECT menu_item_id::text FROM recipes WHERE id = $1`, recipeID).Scan(&nowMenuItemID); err != nil {
		t.Fatalf("post-merge query: %v", err)
	}
	if nowMenuItemID != menuItemB {
		t.Fatalf("expected recipe.menu_item_id=B (%v), got %v", menuItemB, nowMenuItemID)
	}

	// Verify menu_item A was deleted.
	var aExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM menu_items WHERE id = $1)`, menuItemA).Scan(&aExists); err != nil {
		t.Fatalf("exists query: %v", err)
	}
	if aExists {
		t.Fatalf("expected menu_item A to be deleted, but it still exists")
	}
}

// Card I3 (decision 194): a campaign and one of its codes name dish A. Merging
// A into B must re-point BOTH to B before the source dish is deleted — the
// house convention ("merge re-points all FKs, deletes source"). Before the
// card the delete was refused with 23503 (campaigns_admin_item_id_fkey); with
// only the migration's SET NULL backstop and no re-point, both rows would
// survive with item_id NULL, which this test also reds on.
func TestRepository_MergeMenuItem_RePointsCampaignsAndCodes(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	menuItemA := seedMenuItem(t, pool, "Old Wings")
	menuItemB := seedMenuItem(t, pool, "New Wings")

	var userID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, first_name, last_name, roles, status)
		 VALUES ('merge-' || gen_random_uuid()::text || '@test.invalid', 'Merge', 'Test', ARRAY['manager'], 'active')
		 RETURNING id::text`).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	var campaignID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO campaigns_admin (id, slug, name, offer_text, face_value_cents, requires_online, item_id, ends_at, created_by)
		 VALUES (gen_random_uuid(), 'merge-' || gen_random_uuid()::text, 'Wing Wednesday', '$2 off', 200, false, $1, now() + interval '7 days', $2)
		 RETURNING id::text`, menuItemA, userID).Scan(&campaignID); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	var codeID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO qr_codes (short, campaign_id, channel, item_id, created_by)
		 VALUES ('MERGE2', $1, 'flyer', $2, $3) RETURNING id::text`,
		campaignID, menuItemA, userID).Scan(&codeID); err != nil {
		t.Fatalf("seed code: %v", err)
	}

	rowsRePointed, err := MergeMenuItem(ctx, pool, menuItemA, menuItemB)
	if err != nil {
		t.Fatalf("MergeMenuItem: %v", err)
	}
	// No recipes here: the figure counts the campaign and the code.
	if rowsRePointed != 2 {
		t.Errorf("rows re-pointed = %d, want 2 (one campaign + one code)", rowsRePointed)
	}

	var campaignItem, codeItem *string
	if err := pool.QueryRow(ctx,
		`SELECT item_id::text FROM campaigns_admin WHERE id = $1`, campaignID).Scan(&campaignItem); err != nil {
		t.Fatalf("the campaign did not survive the merge: %v", err)
	}
	if campaignItem == nil || *campaignItem != menuItemB {
		t.Errorf("campaigns_admin.item_id = %v, want B (%s)", deref(campaignItem), menuItemB)
	}
	if err := pool.QueryRow(ctx,
		`SELECT item_id::text FROM qr_codes WHERE id = $1`, codeID).Scan(&codeItem); err != nil {
		t.Fatalf("the code did not survive the merge: %v", err)
	}
	if codeItem == nil || *codeItem != menuItemB {
		t.Errorf("qr_codes.item_id = %v, want B (%s)", deref(codeItem), menuItemB)
	}

	var aExists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM menu_items WHERE id = $1)`, menuItemA).Scan(&aExists); err != nil {
		t.Fatalf("exists query: %v", err)
	}
	if aExists {
		t.Errorf("menu_item A still exists after the merge")
	}
}

func deref(s *string) string {
	if s == nil {
		return "NULL"
	}
	return *s
}

func TestRepository_MergeMenuItem_SelfFails(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	menuItemID := seedMenuItem(t, pool, "Bowl")
	_, err := MergeMenuItem(ctx, pool, menuItemID, menuItemID)
	if err == nil {
		t.Fatalf("expected error on self-merge, got nil")
	}
	if !strings.Contains(err.Error(), "cannot_merge_into_self") {
		t.Fatalf("expected error to contain 'cannot_merge_into_self', got %v", err)
	}
}

func TestRepository_LargestSiblingAllocation_PicksDescByPct(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	menuItem30 := seedMenuItem(t, pool, "Bowl 30")
	menuItem45 := seedMenuItem(t, pool, "Bowl 45")
	menuItem20 := seedMenuItem(t, pool, "Bowl 20")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken")

	_ = seedRecipe(t, pool, menuItem30, purchaseItemID, 30.0)
	middleRecipeID := seedRecipe(t, pool, menuItem45, purchaseItemID, 45.0)
	_ = seedRecipe(t, pool, menuItem20, purchaseItemID, 20.0)

	// Exclude the middle (45%) recipe — largest sibling should be 30%, not 20%.
	name, pct, err := LargestSiblingAllocation(ctx, pool, purchaseItemID, middleRecipeID)
	if err != nil {
		t.Fatalf("LargestSiblingAllocation: %v", err)
	}
	if pct != 30.0 {
		t.Fatalf("expected pct=30, got %v", pct)
	}
	if name != "Bowl 30" {
		t.Fatalf("expected name='Bowl 30', got %q", name)
	}
}

// Card J2 (BACKLOG B-479): a dish merge aimed at a dish that does not exist.
// The source here is UNATTACHED — no recipe, no campaign, no code — which is
// the shape that used to slip through: nothing refused the three re-points
// (0 rows each), the DELETE removed the dish, `daily_menu_sales` went with it
// (ON DELETE CASCADE, migration 0061) and the call answered 200
// {"rows_re_pointed":0}. (An ATTACHED source was refused by the campaign FK
// and answered 500 — same request, different answer.)
//
// Both legs must refuse and change nothing: the repository call returns an
// error naming target_not_found, and the handler answers 404 with it.
const mergeMissingTargetID = "00000000-0000-4000-8000-000000000479" // names no dish

func assertDishAndSalesSurvive(t *testing.T, pool *pgxpool.Pool, leg, menuItemID string) {
	t.Helper()
	ctx := context.Background()
	var dishes, sales int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM menu_items WHERE id = $1`, menuItemID).Scan(&dishes); err != nil {
		t.Fatalf("%s: count menu_items: %v", leg, err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM daily_menu_sales WHERE menu_item_id = $1`, menuItemID).Scan(&sales); err != nil {
		t.Fatalf("%s: count daily_menu_sales: %v", leg, err)
	}
	if dishes != 1 {
		t.Errorf("%s: the source dish has %d rows after a merge into a dish that does not exist, want 1 (it was deleted)", leg, dishes)
	}
	if sales != 1 {
		t.Errorf("%s: the source dish has %d daily_menu_sales rows, want 1 (its sales history went with it)", leg, sales)
	}
}

func TestMergeMenuItem_MissingTargetIsRefused(t *testing.T) {
	t.Run("repository", func(t *testing.T) {
		pool := setupTestDB(t)
		ctx := context.Background()
		source := seedMenuItem(t, pool, "Orphan Bowl")
		seedDailyMenuSales(t, pool, source, "2026-09-28", 7, 84.00)

		rows, err := MergeMenuItem(ctx, pool, source, mergeMissingTargetID)
		if err == nil {
			t.Errorf("MergeMenuItem into a missing target returned (%d, nil), want an error naming target_not_found", rows)
		} else if !strings.Contains(err.Error(), "target_not_found") {
			t.Errorf("MergeMenuItem into a missing target: error = %q, want it to name target_not_found", err.Error())
		}
		assertDishAndSalesSurvive(t, pool, "repository", source)
	})

	t.Run("handler", func(t *testing.T) {
		pool := setupTestDB(t)
		source := seedMenuItem(t, pool, "Orphan Bowl")
		seedDailyMenuSales(t, pool, source, "2026-09-28", 7, 84.00)

		mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
		rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
			"source_menu_item_id": source,
			"target_menu_item_id": mergeMissingTargetID,
		})
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d body=%s, want 404", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "target_not_found") {
			t.Errorf("body = %s, want it to name target_not_found", rec.Body.String())
		}
		assertDishAndSalesSurvive(t, pool, "handler", source)
	})

	// The same refusal for a source a campaign names — before the guard this
	// shape answered 500 (23503 campaigns_admin_item_id_fkey), not 404.
	t.Run("attached source answers the same 404", func(t *testing.T) {
		pool := setupTestDB(t)
		ctx := context.Background()
		source := seedMenuItem(t, pool, "Campaign Bowl")
		seedDailyMenuSales(t, pool, source, "2026-09-28", 3, 36.00)
		var userID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO users (email, first_name, last_name, roles, status)
			 VALUES ('merge-' || gen_random_uuid()::text || '@test.invalid', 'Merge', 'Test', ARRAY['manager'], 'active')
			 RETURNING id::text`).Scan(&userID); err != nil {
			t.Fatalf("seed user: %v", err)
		}
		var campaignID string
		if err := pool.QueryRow(ctx,
			`INSERT INTO campaigns_admin (id, slug, name, offer_text, face_value_cents, requires_online, item_id, ends_at, created_by)
			 VALUES (gen_random_uuid(), 'merge-' || gen_random_uuid()::text, 'Bowl Friday', '$1 off', 100, false, $1, now() + interval '7 days', $2)
			 RETURNING id::text`, source, userID).Scan(&campaignID); err != nil {
			t.Fatalf("seed campaign: %v", err)
		}

		mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
		rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
			"source_menu_item_id": source,
			"target_menu_item_id": mergeMissingTargetID,
		})
		if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "target_not_found") {
			t.Errorf("status = %d body=%s, want 404 target_not_found", rec.Code, rec.Body.String())
		}
		assertDishAndSalesSurvive(t, pool, "attached", source)
		var item *string
		if err := pool.QueryRow(ctx,
			`SELECT item_id::text FROM campaigns_admin WHERE id = $1`, campaignID).Scan(&item); err != nil {
			t.Fatalf("read campaign: %v", err)
		}
		if item == nil || *item != source {
			t.Errorf("the campaign's item changed on a refused merge: %v, want %s", item, source)
		}
	})
}
