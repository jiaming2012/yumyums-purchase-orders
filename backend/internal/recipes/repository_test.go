package recipes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	if !errors.Is(err, ErrMergeIntoSelf) {
		t.Fatalf("self-merge: error = %v, want ErrMergeIntoSelf", err)
	}
}

// The same dish spelled in two cases is still a self-merge. Compared as
// strings the two ids differ, the guard passes, nothing is re-pointed and the
// DELETE takes the dish — its recipes and daily_menu_sales with it — and the
// call answers 200. The guard compares parsed uuids.
func TestMergeMenuItem_CaseVariantSelfIsRefused(t *testing.T) {
	t.Run("repository", func(t *testing.T) {
		pool := setupTestDB(t)
		dish := seedMenuItem(t, pool, "Bowl")
		seedDailyMenuSales(t, pool, dish, "2026-09-28", 7, 84.00)

		rows, err := MergeMenuItem(context.Background(), pool, strings.ToUpper(dish), strings.ToLower(dish))
		if !errors.Is(err, ErrMergeIntoSelf) {
			t.Errorf("MergeMenuItem of a dish into its own id in another case returned (%d, %v), want ErrMergeIntoSelf", rows, err)
		}
		assertDishAndSalesSurvive(t, pool, "repository", dish)
	})

	t.Run("handler", func(t *testing.T) {
		pool := setupTestDB(t)
		dish := seedMenuItem(t, pool, "Bowl")
		seedDailyMenuSales(t, pool, dish, "2026-09-28", 7, 84.00)

		mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
		rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
			"source_menu_item_id": strings.ToUpper(dish),
			"target_menu_item_id": strings.ToLower(dish),
		})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d body=%s, want 400", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "cannot_merge_into_self") {
			t.Errorf("body = %s, want it to name cannot_merge_into_self", rec.Body.String())
		}
		assertDishAndSalesSurvive(t, pool, "handler", dish)
	})
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
		t.Errorf("%s: the dish has %d rows after a refused merge, want 1", leg, dishes)
	}
	if sales != 1 {
		t.Errorf("%s: the dish has %d daily_menu_sales rows after a refused merge, want 1", leg, sales)
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

// Card K2 (BACKLOG B-485): the two wrong answers a dish merge still gave after
// card J2 guarded the target.
//
// A SOURCE that names no dish re-pointed 0 rows, deleted 0 rows and answered
// 200 {"rows_re_pointed":0} — a merge that did nothing reported as a merge.
// It is refused now: 404 source_not_found, the same shape as a missing target.
func TestMergeMenuItem_MissingSourceIs404(t *testing.T) {
	t.Run("handler", func(t *testing.T) {
		pool := setupTestDB(t)
		target := seedMenuItem(t, pool, "Surviving Bowl")
		seedDailyMenuSales(t, pool, target, "2026-09-28", 7, 84.00)

		mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
		rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
			"source_menu_item_id": mergeMissingTargetID, // names no dish
			"target_menu_item_id": target,
		})
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d body=%s, want 404", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "source_not_found") {
			t.Errorf("body = %s, want it to name source_not_found", rec.Body.String())
		}
		assertDishAndSalesSurvive(t, pool, "handler", target)
	})

	t.Run("repository", func(t *testing.T) {
		pool := setupTestDB(t)
		target := seedMenuItem(t, pool, "Surviving Bowl")
		seedDailyMenuSales(t, pool, target, "2026-09-28", 7, 84.00)

		rows, err := MergeMenuItem(context.Background(), pool, mergeMissingTargetID, target)
		if !errors.Is(err, ErrMergeSourceNotFound) {
			t.Errorf("MergeMenuItem from a missing source returned (%d, %v), want ErrMergeSourceNotFound", rows, err)
		}
		assertDishAndSalesSurvive(t, pool, "repository", target)
	})

	// Neither id names a dish: the target is read first, so that is the answer.
	t.Run("both missing answers target_not_found", func(t *testing.T) {
		pool := setupTestDB(t)
		_, err := MergeMenuItem(context.Background(), pool,
			"00000000-0000-4000-8000-000000000485", mergeMissingTargetID)
		if !errors.Is(err, ErrMergeTargetNotFound) {
			t.Errorf("error = %v, want ErrMergeTargetNotFound", err)
		}
	})
}

// An id that is not a uuid reached Postgres, came back 22P02 and answered 500
// internal_error. It is the caller's mistake: 400 bad_id, before any query.
func TestMergeMenuItem_BadIDIs400(t *testing.T) {
	cases := []struct {
		name        string
		sourceIsBad bool
		targetIsBad bool
		badID       string
		// respell turns the real dish's id into the bad id: a spelling that
		// names the dish to Postgres or to uuid.Parse but is not the plain
		// 36-character hyphenated form.
		respell func(dish string) string
	}{
		{name: "target is not a uuid", targetIsBad: true, badID: "not-a-uuid"},
		{name: "source is not a uuid", sourceIsBad: true, badID: "not-a-uuid"},
		{name: "both are not uuids", sourceIsBad: true, targetIsBad: true, badID: "42"},
		// 38 characters: uuid.Parse drops the first and last byte unchecked, so
		// this used to parse to the real dish, which was then merged and deleted.
		{name: "source is a uuid wrapped in two stray characters", sourceIsBad: true,
			respell: func(dish string) string { return "X" + dish + "Y" }},
		// Postgres's uuid input takes both of these for the real dish, so
		// before the id was parsed here they merged. Only the hyphenated
		// 36-character form is an id now; braces are the same unchecked
		// 38-character path as the wrapped id above.
		{name: "source is a real dish id without hyphens", sourceIsBad: true,
			respell: func(dish string) string { return strings.ReplaceAll(dish, "-", "") }},
		{name: "source is a real dish id in braces", sourceIsBad: true,
			respell: func(dish string) string { return "{" + dish + "}" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool := setupTestDB(t)
			dish := seedMenuItem(t, pool, "Real Bowl")
			seedDailyMenuSales(t, pool, dish, "2026-09-28", 7, 84.00)
			source, target := dish, dish
			if tc.respell != nil {
				tc.badID = tc.respell(dish)
				// A second real dish, so the only thing wrong is the respelled id.
				target = seedMenuItem(t, pool, "Other Bowl")
			}
			if tc.sourceIsBad {
				source = tc.badID
			}
			if tc.targetIsBad {
				// Distinct from a bad source so "both" is not a self-merge.
				target = tc.badID + "-t"
			}

			mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
			rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
				"source_menu_item_id": source,
				"target_menu_item_id": target,
			})
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d body=%s, want 400", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "bad_id") {
				t.Errorf("body = %s, want it to name bad_id", rec.Body.String())
			}
			assertDishAndSalesSurvive(t, pool, tc.name, dish)
		})
	}

	t.Run("repository returns ErrBadID", func(t *testing.T) {
		pool := setupTestDB(t)
		dish := seedMenuItem(t, pool, "Real Bowl")
		seedDailyMenuSales(t, pool, dish, "2026-09-28", 7, 84.00)
		if _, err := MergeMenuItem(context.Background(), pool, dish, "not-a-uuid"); !errors.Is(err, ErrBadID) {
			t.Errorf("bad target: error = %v, want ErrBadID", err)
		}
		if _, err := MergeMenuItem(context.Background(), pool, "not-a-uuid", dish); !errors.Is(err, ErrBadID) {
			t.Errorf("bad source: error = %v, want ErrBadID", err)
		}
		assertDishAndSalesSurvive(t, pool, "repository", dish)
	})
}

// Card K2 (BACKLOG B-484): the merge reads its target FOR SHARE so the
// surviving dish cannot be deleted between that read and the re-points. Nothing
// pinned it — TestMergeMenuItem_MissingTargetIsRefused passes with the lock
// removed, because a single caller never sees the window.
//
// Three connections. A helper takes a row lock on the SOURCE dish, which holds
// the merge in flight AFTER it has read the target (the source is read second).
// A second pgx connection then DELETEs the target. With the lock that delete
// must WAIT — it is still blocked when the merge is let go, and by the time it
// runs the recipe already names the target. Without the lock the delete
// returns at once and the merge is re-pointing at a dish that is gone.
func TestMergeMenuItem_LockHoldsAgainstConcurrentTargetDelete(t *testing.T) {
	pool := setupTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	source := seedMenuItem(t, pool, "Old Wings")
	target := seedMenuItem(t, pool, "New Wings")
	ingredient := seedPurchaseItem(t, pool, "Chicken Wings")
	recipeID := seedRecipe(t, pool, source, ingredient, 40.0)

	connString := pool.Config().ConnString()
	holder, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatalf("connect holder: %v", err)
	}
	defer holder.Close(context.Background()) //nolint:errcheck
	deleter, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatalf("connect deleter: %v", err)
	}
	defer deleter.Close(context.Background()) //nolint:errcheck

	var holderPID, deleterPID uint32
	if err := holder.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&holderPID); err != nil {
		t.Fatalf("holder pid: %v", err)
	}
	if err := deleter.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&deleterPID); err != nil {
		t.Fatalf("deleter pid: %v", err)
	}

	// 1. Hold the source dish. The merge will queue behind this.
	holderTx, err := holder.Begin(ctx)
	if err != nil {
		t.Fatalf("holder begin: %v", err)
	}
	released := false
	release := func() {
		if !released {
			released = true
			holderTx.Rollback(context.Background()) //nolint:errcheck
		}
	}
	if _, err := holderTx.Exec(ctx, `SELECT 1 FROM menu_items WHERE id = $1 FOR UPDATE`, source); err != nil {
		t.Fatalf("holder lock source: %v", err)
	}

	// 2. Start the merge and wait until Postgres says it is queued on a lock.
	type mergeResult struct {
		rows int
		err  error
	}
	mergeDone := make(chan mergeResult, 1)
	go func() {
		rows, err := MergeMenuItem(ctx, pool, source, target)
		mergeDone <- mergeResult{rows, err}
	}()
	// Whatever happens below, let the merge go and wait for it before the pool closes.
	// The delete (step 3) runs on `deleter` in its own goroutine; it is waited
	// for too, so that connection is never closed under an Exec still in flight.
	mergeCollected := false
	deleteDone := make(chan error, 1)
	deleteStarted, deleteCollected := false, false
	defer func() {
		release()
		if !mergeCollected {
			<-mergeDone
		}
		if deleteStarted && !deleteCollected {
			<-deleteDone
		}
	}()

	waitFor := func(what string, cond func() (bool, error)) {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for {
			ok, err := cond()
			if err != nil {
				t.Fatalf("%s: %v", what, err)
			}
			if ok {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s: not seen within 15s", what)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	waitFor("the merge queued behind the held source dish", func() (bool, error) {
		select {
		case r := <-mergeDone:
			mergeCollected = true
			return false, fmt.Errorf("the merge returned (%d, %v) while its source dish was held", r.rows, r.err)
		default:
		}
		var n int
		err := pool.QueryRow(ctx,
			`SELECT count(*) FROM pg_stat_activity
			  WHERE datname = current_database() AND wait_event_type = 'Lock'
			    AND $1 = ANY(pg_blocking_pids(pid))`,
			holderPID).Scan(&n)
		return n > 0, err
	})

	// 3. The second connection deletes the target while the merge is in flight.
	deleteStarted = true
	go func() {
		_, err := deleter.Exec(ctx, `DELETE FROM menu_items WHERE id = $1`, target)
		deleteDone <- err
	}()
	var deleteErr error
	waitFor("the target delete queued behind the merge", func() (bool, error) {
		select {
		case deleteErr = <-deleteDone:
			deleteCollected = true
			return true, nil
		default:
		}
		var blocked bool
		err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE pid = $1 AND wait_event_type = 'Lock')`,
			deleterPID).Scan(&blocked)
		return blocked, err
	})
	if deleteCollected {
		t.Fatalf("DELETE of the target dish returned (err=%v) while the merge into it was still in flight — "+
			"nothing held the target; the merge is now re-pointing at a dish that is gone", deleteErr)
	}

	// 4. Let the merge go. It commits with the target present.
	release()
	var merged mergeResult
	select {
	case merged = <-mergeDone:
		mergeCollected = true
	case <-ctx.Done():
		t.Fatalf("the merge did not return after its source dish was released")
	}
	if merged.err != nil {
		t.Fatalf("MergeMenuItem: %v, want the merge to commit", merged.err)
	}
	if merged.rows != 1 {
		t.Errorf("rows re-pointed = %d, want 1 (the recipe)", merged.rows)
	}

	// 5. Only now does the delete run — against a target the recipe names.
	select {
	case deleteErr = <-deleteDone:
		deleteCollected = true
	case <-ctx.Done():
		t.Fatalf("the target delete did not return after the merge committed")
	}
	var targetRows, sourceRows int
	var recipeDish *string
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE id = $1`, target).Scan(&targetRows); err != nil {
		t.Fatalf("count target: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM menu_items WHERE id = $1`, source).Scan(&sourceRows); err != nil {
		t.Fatalf("count source: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT max(menu_item_id::text) FROM recipes WHERE id = $1`, recipeID).Scan(&recipeDish); err != nil {
		t.Fatalf("read recipe: %v", err)
	}
	if sourceRows != 0 {
		t.Errorf("the source dish has %d rows after a committed merge, want 0", sourceRows)
	}
	if deleteErr != nil {
		// Refused: the re-pointed recipe holds the target.
		var pgErr *pgconn.PgError
		if !errors.As(deleteErr, &pgErr) || pgErr.Code != "23503" {
			t.Fatalf("the target delete failed with %v, want nil or a 23503 foreign-key refusal", deleteErr)
		}
		if targetRows != 1 {
			t.Errorf("the target delete was refused yet the target has %d rows, want 1", targetRows)
		}
		if recipeDish == nil || *recipeDish != target {
			t.Errorf("recipe.menu_item_id = %s, want the target %s", deref(recipeDish), target)
		}
	} else {
		// Allowed: it ran after the merge, found the recipe re-pointed at the
		// target and took it along (recipes.menu_item_id cascades).
		if targetRows != 0 {
			t.Errorf("the target delete returned nil yet the target has %d rows, want 0", targetRows)
		}
		if recipeDish != nil {
			t.Errorf("the target is gone but the recipe still names %s", *recipeDish)
		}
	}
	t.Logf("K2-lock: delete blocked until the merge committed; merge rows=%d; delete err=%v; target rows=%d recipe dish=%s",
		merged.rows, deleteErr, targetRows, deref(recipeDish))
}
