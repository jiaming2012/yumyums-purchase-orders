package recipes

import (
	"context"
	"errors"
	"strings"
	"testing"
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
