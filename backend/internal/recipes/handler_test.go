package recipes

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ─────────────────────────────────────────────────────────────────────────────
// Recipes CRUD handler tests (Plan 03).
// Reuses setupTestDB / seedMenuItem / seedPurchaseItem / seedRecipe from
// helpers_test.go (Wave 1) and seedVendor / seedPurchaseEvent / seedPurchaseLineItem
// from menu_cogs_test.go (Plan 02).
// ─────────────────────────────────────────────────────────────────────────────

// mountRouter builds a chi.Router with a single route at the given method+path,
// suitable for exercising chi.URLParam path params via httptest.
func mountRouter(method, path string, h http.HandlerFunc) *chi.Mux {
	r := chi.NewRouter()
	r.Method(method, path, h)
	return r
}

// doJSON encodes payload as JSON, sends to mux, returns recorder.
func doJSON(t *testing.T, mux *chi.Mux, method, target string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = b
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// ─────────────────────────────────────────────────────────────────────────────
// validateUsagePct unit tests — no DB required.
// ─────────────────────────────────────────────────────────────────────────────

func TestValidateUsagePct_RejectsNon5Multiple(t *testing.T) {
	for _, v := range []float64{1, 3, 7, 17, 42.5, 99} {
		if validateUsagePct(v) == "" {
			t.Errorf("validateUsagePct(%v) accepted non-multiple-of-5", v)
		}
	}
	for _, v := range []float64{0, 5, 25, 50, 95, 100} {
		if validateUsagePct(v) != "" {
			t.Errorf("validateUsagePct(%v) rejected valid snap value", v)
		}
	}
}

func TestValidateUsagePct_RejectsOutOfRange(t *testing.T) {
	if validateUsagePct(-5) == "" {
		t.Error("accepted -5")
	}
	if validateUsagePct(105) == "" {
		t.Error("accepted 105")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// CreateRecipeHandler
// ─────────────────────────────────────────────────────────────────────────────

func TestCreateRecipe_HappyPath(t *testing.T) {
	pool := setupTestDB(t)
	menuItemID := seedMenuItem(t, pool, "Sliders")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken Thighs")

	mux := mountRouter(http.MethodPost, "/inventory/recipes", CreateRecipeHandler(pool))
	rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes", map[string]any{
		"menu_item_id":     menuItemID,
		"purchase_item_id": purchaseItemID,
		"usage_pct":        45.0,
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] == "" || resp["id"] == nil {
		t.Error("missing id in response")
	}
	if resp["sum_after"] != 45.0 {
		t.Errorf("expected sum_after=45, got %v", resp["sum_after"])
	}
}

func TestCreateRecipe_SumExceeds100_NamesLargestSibling(t *testing.T) {
	pool := setupTestDB(t)
	menuItem1 := seedMenuItem(t, pool, "Sliders")
	menuItem2 := seedMenuItem(t, pool, "Tacos")
	purchaseItem := seedPurchaseItem(t, pool, "Chicken Thighs")
	// Existing 70% allocation on menu_item_1.
	_ = seedRecipe(t, pool, menuItem1, purchaseItem, 70.0)

	mux := mountRouter(http.MethodPost, "/inventory/recipes", CreateRecipeHandler(pool))
	rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes", map[string]any{
		"menu_item_id":     menuItem2,
		"purchase_item_id": purchaseItem,
		"usage_pct":        50.0,
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["error"] != "sum_exceeds_100" {
		t.Errorf("wrong error: %v", resp["error"])
	}
	if resp["conflict_menu_item"] != "Sliders" {
		t.Errorf("expected conflict 'Sliders', got %v", resp["conflict_menu_item"])
	}
	if resp["conflict_pct"] != 70.0 {
		t.Errorf("expected conflict_pct=70, got %v", resp["conflict_pct"])
	}

	// Defense in depth: the rolled-back recipe must NOT be in the DB. Sum should still be 70.
	got, err := SumPerPurchaseItem(context.Background(), pool, purchaseItem)
	if err != nil {
		t.Fatalf("SumPerPurchaseItem: %v", err)
	}
	if got != 70.0 {
		t.Errorf("expected sum=70 after rollback, got %v", got)
	}
}

func TestCreateRecipe_DuplicateReturns409(t *testing.T) {
	pool := setupTestDB(t)
	menuItemID := seedMenuItem(t, pool, "Bowl")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken")
	mux := mountRouter(http.MethodPost, "/inventory/recipes", CreateRecipeHandler(pool))

	body := map[string]any{
		"menu_item_id":     menuItemID,
		"purchase_item_id": purchaseItemID,
		"usage_pct":        40.0,
	}
	// First create succeeds.
	rec1 := doJSON(t, mux, http.MethodPost, "/inventory/recipes", body)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("first create: expected 201, got %d body=%s", rec1.Code, rec1.Body.String())
	}
	// Second create with same pair → 409.
	rec2 := doJSON(t, mux, http.MethodPost, "/inventory/recipes", body)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("second create: expected 409, got %d body=%s", rec2.Code, rec2.Body.String())
	}
	if !bytes.Contains(rec2.Body.Bytes(), []byte("recipe_already_exists")) {
		t.Errorf("expected 'recipe_already_exists' in body, got %s", rec2.Body.String())
	}
}

func TestCreateRecipe_InvalidSnapReturns422(t *testing.T) {
	pool := setupTestDB(t)
	menuItemID := seedMenuItem(t, pool, "Bowl")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken")
	mux := mountRouter(http.MethodPost, "/inventory/recipes", CreateRecipeHandler(pool))

	rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes", map[string]any{
		"menu_item_id":     menuItemID,
		"purchase_item_id": purchaseItemID,
		"usage_pct":        17.0, // non-multiple of 5
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "invalid_usage_pct" {
		t.Errorf("expected error=invalid_usage_pct, got %v", resp["error"])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// UpdateRecipeHandler
// ─────────────────────────────────────────────────────────────────────────────

func TestUpdateRecipe_HappyPath(t *testing.T) {
	pool := setupTestDB(t)
	menuItemID := seedMenuItem(t, pool, "Bowl")
	purchaseItemID := seedPurchaseItem(t, pool, "Chicken")
	recipeID := seedRecipe(t, pool, menuItemID, purchaseItemID, 40.0)

	mux := mountRouter(http.MethodPut, "/inventory/recipes/{id}", UpdateRecipeHandler(pool))
	rec := doJSON(t, mux, http.MethodPut, "/inventory/recipes/"+recipeID, map[string]any{
		"usage_pct": 55.0,
	})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Re-query DB to assert usage_pct was stored as 55.
	got, err := SumPerPurchaseItem(context.Background(), pool, purchaseItemID)
	if err != nil {
		t.Fatalf("SumPerPurchaseItem: %v", err)
	}
	if got != 55.0 {
		t.Errorf("expected SUM=55 after update, got %v", got)
	}
}

func TestUpdateRecipe_NotFoundReturns404(t *testing.T) {
	pool := setupTestDB(t)
	mux := mountRouter(http.MethodPut, "/inventory/recipes/{id}", UpdateRecipeHandler(pool))
	rec := doJSON(t, mux, http.MethodPut, "/inventory/recipes/00000000-0000-0000-0000-000000000000", map[string]any{
		"usage_pct": 50.0,
	})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("recipe_not_found")) {
		t.Errorf("expected 'recipe_not_found' in body, got %s", rec.Body.String())
	}
}

func TestUpdateRecipe_SumExceeds100_NamesSibling(t *testing.T) {
	pool := setupTestDB(t)
	// Sibling at 60%, this recipe starts at 30%. Attempt to PUT this to 50% → sum=110 → 422.
	siblingMI := seedMenuItem(t, pool, "Sibling-60")
	thisMI := seedMenuItem(t, pool, "This-30")
	piID := seedPurchaseItem(t, pool, "Chicken")
	_ = seedRecipe(t, pool, siblingMI, piID, 60.0)
	thisRecipeID := seedRecipe(t, pool, thisMI, piID, 30.0)

	mux := mountRouter(http.MethodPut, "/inventory/recipes/{id}", UpdateRecipeHandler(pool))
	rec := doJSON(t, mux, http.MethodPut, "/inventory/recipes/"+thisRecipeID, map[string]any{
		"usage_pct": 50.0,
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "sum_exceeds_100" {
		t.Errorf("wrong error: %v", resp["error"])
	}
	if resp["conflict_menu_item"] != "Sibling-60" {
		t.Errorf("expected conflict_menu_item='Sibling-60', got %v", resp["conflict_menu_item"])
	}
	if resp["conflict_pct"] != 60.0 {
		t.Errorf("expected conflict_pct=60, got %v", resp["conflict_pct"])
	}
}

func TestUpdateRecipe_InvalidSnapReturns422(t *testing.T) {
	pool := setupTestDB(t)
	miID := seedMenuItem(t, pool, "Bowl")
	piID := seedPurchaseItem(t, pool, "Chicken")
	recipeID := seedRecipe(t, pool, miID, piID, 40.0)

	mux := mountRouter(http.MethodPut, "/inventory/recipes/{id}", UpdateRecipeHandler(pool))
	rec := doJSON(t, mux, http.MethodPut, "/inventory/recipes/"+recipeID, map[string]any{
		"usage_pct": 23.0,
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["error"] != "invalid_usage_pct" {
		t.Errorf("expected error=invalid_usage_pct, got %v", resp["error"])
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DeleteRecipeHandler
// ─────────────────────────────────────────────────────────────────────────────

func TestDeleteRecipe_RemovesRow(t *testing.T) {
	pool := setupTestDB(t)
	miID := seedMenuItem(t, pool, "Bowl")
	piID := seedPurchaseItem(t, pool, "Chicken")
	recipeID := seedRecipe(t, pool, miID, piID, 50.0)

	mux := mountRouter(http.MethodDelete, "/inventory/recipes/{id}", DeleteRecipeHandler(pool))
	req := httptest.NewRequest(http.MethodDelete, "/inventory/recipes/"+recipeID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}

	got, err := SumPerPurchaseItem(context.Background(), pool, piID)
	if err != nil {
		t.Fatalf("SumPerPurchaseItem: %v", err)
	}
	if got != 0.0 {
		t.Errorf("expected sum=0 after delete, got %v", got)
	}
}

func TestDeleteRecipe_NotFoundReturns404(t *testing.T) {
	pool := setupTestDB(t)
	mux := mountRouter(http.MethodDelete, "/inventory/recipes/{id}", DeleteRecipeHandler(pool))
	req := httptest.NewRequest(http.MethodDelete, "/inventory/recipes/00000000-0000-0000-0000-000000000000", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("recipe_not_found")) {
		t.Errorf("expected 'recipe_not_found' in body, got %s", rec.Body.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// MergeMenuItemHandler
// ─────────────────────────────────────────────────────────────────────────────

func TestMergeMenuItem_RePointsRows(t *testing.T) {
	pool := setupTestDB(t)
	mi1 := seedMenuItem(t, pool, "Old-Bowl")
	mi2 := seedMenuItem(t, pool, "New-Bowl")
	pi1 := seedPurchaseItem(t, pool, "Chicken")
	pi2 := seedPurchaseItem(t, pool, "Salmon")
	r1 := seedRecipe(t, pool, mi1, pi1, 50.0)
	r2 := seedRecipe(t, pool, mi1, pi2, 30.0)

	mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
	rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
		"source_menu_item_id": mi1,
		"target_menu_item_id": mi2,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]int
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["rows_re_pointed"] != 2 {
		t.Errorf("expected rows_re_pointed=2, got %v", resp["rows_re_pointed"])
	}

	// Verify both recipes now point to mi2.
	for _, rid := range []string{r1, r2} {
		var nowMI string
		if err := pool.QueryRow(context.Background(),
			`SELECT menu_item_id::text FROM recipes WHERE id = $1`, rid).Scan(&nowMI); err != nil {
			t.Fatalf("post-merge query for %s: %v", rid, err)
		}
		if nowMI != mi2 {
			t.Errorf("recipe %s: expected menu_item_id=%s, got %s", rid, mi2, nowMI)
		}
	}

	// Verify mi1 was deleted.
	var exists bool
	if err := pool.QueryRow(context.Background(),
		`SELECT EXISTS(SELECT 1 FROM menu_items WHERE id = $1)`, mi1).Scan(&exists); err != nil {
		t.Fatalf("exists query: %v", err)
	}
	if exists {
		t.Errorf("expected mi1 to be deleted, but it still exists")
	}
}

func TestMergeMenuItem_SelfReturns400(t *testing.T) {
	pool := setupTestDB(t)
	miID := seedMenuItem(t, pool, "Bowl")

	mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
	rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
		"source_menu_item_id": miID,
		"target_menu_item_id": miID,
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("cannot_merge_into_self")) {
		t.Errorf("expected 'cannot_merge_into_self' in body, got %s", rec.Body.String())
	}
}

func TestMergeMenuItem_MissingFields_400(t *testing.T) {
	pool := setupTestDB(t)
	mux := mountRouter(http.MethodPost, "/inventory/recipes/merge", MergeMenuItemHandler(pool))
	rec := doJSON(t, mux, http.MethodPost, "/inventory/recipes/merge", map[string]any{
		"source_menu_item_id": "",
		"target_menu_item_id": "some-id",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// ListRecipesHandler
// ─────────────────────────────────────────────────────────────────────────────

func TestListRecipes_HappyPath(t *testing.T) {
	pool := setupTestDB(t)
	// Seed 2 ingredients: one with recipe + spend, one with only spend (no recipe).
	vendorID := seedVendor(t, pool, "Acme-ListTest")
	mi := seedMenuItem(t, pool, "Bowl")
	pi1 := seedPurchaseItem(t, pool, "Chicken") // recipe + spend
	pi2 := seedPurchaseItem(t, pool, "Paper Towels") // spend only

	eventID := seedPurchaseEvent(t, pool, vendorID, "2026-05-27", 0.0, 100.00)
	// pi1: $80 spend, pi2: $20 spend
	seedPurchaseLineItem(t, pool, eventID, pi1, "Chicken", 1, 80.00)
	seedPurchaseLineItem(t, pool, eventID, pi2, "Paper Towels", 1, 20.00)
	_ = seedRecipe(t, pool, mi, pi1, 40.0)

	mux := mountRouter(http.MethodGet, "/inventory/recipes", ListRecipesHandler(pool))
	req := httptest.NewRequest(http.MethodGet, "/inventory/recipes?from=2026-05-25&to=2026-05-31", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		From        string                `json:"from"`
		To          string                `json:"to"`
		Ingredients []IngredientWithSpend `json:"ingredients"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.From != "2026-05-25" || resp.To != "2026-05-31" {
		t.Errorf("From/To = (%s, %s)", resp.From, resp.To)
	}
	if len(resp.Ingredients) != 2 {
		t.Fatalf("Ingredients len = %d, want 2; body=%s", len(resp.Ingredients), rec.Body.String())
	}
	// Sorted by last_week_spend DESC → Chicken ($80) first, Paper Towels ($20) second.
	if resp.Ingredients[0].Description != "Chicken" {
		t.Errorf("first ingredient = %q, want Chicken", resp.Ingredients[0].Description)
	}
	if resp.Ingredients[1].Description != "Paper Towels" {
		t.Errorf("second ingredient = %q, want Paper Towels", resp.Ingredients[1].Description)
	}
	// Chicken has 1 recipe, Paper Towels has 0.
	if len(resp.Ingredients[0].Recipes) != 1 {
		t.Errorf("Chicken Recipes len = %d, want 1", len(resp.Ingredients[0].Recipes))
	}
	if len(resp.Ingredients[1].Recipes) != 0 {
		t.Errorf("Paper Towels Recipes len = %d, want 0", len(resp.Ingredients[1].Recipes))
	}
	// sum_pct should match.
	if resp.Ingredients[0].SumPct != 40.0 {
		t.Errorf("Chicken sum_pct = %v, want 40", resp.Ingredients[0].SumPct)
	}
}

func TestListRecipes_InvalidDates_400(t *testing.T) {
	pool := setupTestDB(t)
	mux := mountRouter(http.MethodGet, "/inventory/recipes", ListRecipesHandler(pool))
	req := httptest.NewRequest(http.MethodGet, "/inventory/recipes?from=garbage&to=2026-05-31", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "from must be YYYY-MM-DD") {
		t.Errorf("expected 'from must be YYYY-MM-DD' in body, got %s", rec.Body.String())
	}
}

func TestListRecipes_DefaultWindow_WhenNoQueryParams(t *testing.T) {
	pool := setupTestDB(t)
	// No seed data — handler should still return 200 with empty ingredients
	// (proves the default Chicago-week window is applied).
	mux := mountRouter(http.MethodGet, "/inventory/recipes", ListRecipesHandler(pool))
	req := httptest.NewRequest(http.MethodGet, "/inventory/recipes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (default window applied), got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		From        string `json:"from"`
		To          string `json:"to"`
		Ingredients []any  `json:"ingredients"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.From == "" || resp.To == "" {
		t.Errorf("expected from/to to be populated from default window, got from=%q to=%q", resp.From, resp.To)
	}
}

// Compile-time check: pool variable used to silence unused (pgxpool import only).
var _ = func(*pgxpool.Pool) {}
