package recipes

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// writeJSON / writeError are intentionally duplicated from internal/inventory and
// internal/toast handlers — the project convention (documented at
// internal/toast/handler.go:12-23) is per-package 5-line helpers rather than a
// shared httpx package.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// MenuCogsHandler returns per-menu-item COGS attribution for the [from, to] window.
//
// Service-token-authenticated peer of inventory.PeriodSummaryHandler (Phase 21).
// Mounted under the service-token chi group in cmd/server/main.go alongside
// /api/v1/inventory/period-summary. The Bearer envelope (503 / 401) is handled
// by auth.ServiceTokenMiddleware; this handler only sees authenticated requests.
//
// Contract: see .planning/phases/999.2-.../999.2-SALES-PROCESSOR-CONTRACT.md
// (authored in Plan 06). Error envelope strings mirror Phase 21 EXACTLY per D-18.
//
// Default mode returns summary rows (one per menu_item). With ?breakdown=true
// each row gains an ingredients array and the unallocated field becomes a
// structured object with by_ingredient breakdown (D-17).
func MenuCogsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		if _, err := time.Parse("2006-01-02", fromStr); err != nil {
			writeError(w, http.StatusBadRequest, "from must be YYYY-MM-DD")
			return
		}
		if _, err := time.Parse("2006-01-02", toStr); err != nil {
			writeError(w, http.StatusBadRequest, "to must be YYYY-MM-DD")
			return
		}
		if fromStr > toStr {
			// Lexicographic compare is correct ONLY for YYYY-MM-DD (Pitfall 1).
			writeError(w, http.StatusBadRequest, "from must be <= to")
			return
		}
		breakdown := r.URL.Query().Get("breakdown") == "true"

		ctx := r.Context()

		// ── Query 1: summary rows (one per menu_item) ──
		// The CTE is the canonical pattern from RESEARCH.md lines 678-749.
		// window_spend tax-prorates per Pitfall 4 (subtotal = total - tax).
		// ingredient_cost_per_unit divides by NULLIF(units, 0) per Pitfall 5.
		summaryRows, err := pool.Query(ctx, `
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
menu_units AS (
  SELECT dms.menu_item_id, SUM(dms.units_sold) AS units_sold
  FROM daily_menu_sales dms
  WHERE dms.business_date BETWEEN $1 AND $2
  GROUP BY dms.menu_item_id
),
alloc AS (
  SELECT
    r.menu_item_id,
    r.purchase_item_id,
    r.usage_pct,
    COALESCE(ws.spend_incl_tax, 0) * (r.usage_pct / 100.0) AS alloc_cost
  FROM recipes r
  LEFT JOIN window_spend ws ON ws.purchase_item_id = r.purchase_item_id
)
SELECT
  mi.id::text                   AS menu_item_id,
  mi.master_id                  AS toast_master_id,
  mi.name                       AS menu_item_name,
  mi.menu_group                 AS menu_group,
  mi.menu_subgroup              AS menu_subgroup,
  COALESCE(mu.units_sold, 0)    AS units_sold,
  ROUND(COALESCE(SUM(a.alloc_cost), 0)::numeric, 2) AS ingredient_cost_total,
  CASE
    WHEN COALESCE(mu.units_sold, 0) = 0 THEN NULL
    ELSE ROUND((COALESCE(SUM(a.alloc_cost), 0) / mu.units_sold)::numeric, 4)
  END AS ingredient_cost_per_unit
FROM menu_items mi
JOIN alloc a ON a.menu_item_id = mi.id
LEFT JOIN menu_units mu ON mu.menu_item_id = mi.id
GROUP BY mi.id, mi.master_id, mi.name, mi.menu_group, mi.menu_subgroup, mu.units_sold
ORDER BY mi.name`, fromStr, toStr)
		if err != nil {
			slog.Error("MenuCogs summary query", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer summaryRows.Close()

		rowsOut := []MenuCOGSRow{}
		rowIndex := map[string]int{} // menu_item_id -> position in rowsOut
		for summaryRows.Next() {
			var row MenuCOGSRow
			var perUnit *float64
			if err := summaryRows.Scan(
				&row.MenuItemID, &row.ToastMasterID, &row.MenuItemName,
				&row.MenuGroup, &row.MenuSubgroup, &row.UnitsSold,
				&row.IngredientCostTotal, &perUnit,
			); err != nil {
				slog.Error("MenuCogs summary scan", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			row.IngredientCostPerUnit = perUnit
			rowIndex[row.MenuItemID] = len(rowsOut)
			rowsOut = append(rowsOut, row)
		}
		if err := summaryRows.Err(); err != nil {
			slog.Error("MenuCogs summary iter", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// ── Query 2: total unallocated_cogs ──
		var totalUnalloc float64
		err = pool.QueryRow(ctx, `
WITH
window_spend AS (
  SELECT pli.purchase_item_id,
         SUM((pli.quantity * pli.price) *
             COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)) AS spend_incl_tax
  FROM purchase_line_items pli
  JOIN purchase_events pe ON pe.id = pli.purchase_event_id
  WHERE pe.event_date BETWEEN $1 AND $2 AND pli.purchase_item_id IS NOT NULL
  GROUP BY pli.purchase_item_id
),
purchase_item_alloc AS (
  SELECT purchase_item_id, COALESCE(SUM(usage_pct), 0) AS sum_pct
  FROM recipes GROUP BY purchase_item_id
)
SELECT ROUND(COALESCE(SUM(
  ws.spend_incl_tax * (1 - COALESCE(pia.sum_pct, 0) / 100.0)
), 0)::numeric, 2)
FROM window_spend ws
LEFT JOIN purchase_item_alloc pia ON pia.purchase_item_id = ws.purchase_item_id`,
			fromStr, toStr,
		).Scan(&totalUnalloc)
		if err != nil {
			slog.Error("MenuCogs unallocated query", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		resp := MenuCOGSResponse{
			From:      fromStr,
			To:        toStr,
			MenuItems: rowsOut,
		}

		if !breakdown {
			// Default mode: unallocated_cogs is a single number.
			resp.UnallocatedCogs = &totalUnalloc
		} else {
			// ── Query 3 (breakdown only): per-ingredient detail per menu_item ──
			// Plus query 4: by-ingredient unallocated breakdown.
			if err := loadBreakdown(ctx, pool, fromStr, toStr, rowsOut, rowIndex); err != nil {
				slog.Error("MenuCogs breakdown ingredient query", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			unalloc, err := loadUnallocatedBreakdown(ctx, pool, fromStr, toStr, totalUnalloc)
			if err != nil {
				slog.Error("MenuCogs unallocated breakdown query", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			resp.Unallocated = &unalloc
		}

		w.Header().Set("Cache-Control", "private, max-age=3600")
		writeJSON(w, http.StatusOK, resp)
	}
}

// loadBreakdown populates each row's Ingredients slice with per-ingredient detail.
func loadBreakdown(ctx context.Context, pool *pgxpool.Pool, from, to string, rows []MenuCOGSRow, rowIndex map[string]int) error {
	breakdownRows, err := pool.Query(ctx, `
WITH window_spend AS (
  SELECT pli.purchase_item_id,
         SUM((pli.quantity * pli.price) *
             COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)) AS spend_incl_tax
  FROM purchase_line_items pli
  JOIN purchase_events pe ON pe.id = pli.purchase_event_id
  WHERE pe.event_date BETWEEN $1 AND $2 AND pli.purchase_item_id IS NOT NULL
  GROUP BY pli.purchase_item_id
)
SELECT
  r.menu_item_id::text,
  pi.description AS purchase_item_description,
  r.usage_pct,
  ROUND((COALESCE(ws.spend_incl_tax, 0) * (r.usage_pct / 100.0))::numeric, 2) AS allocated_cost
FROM recipes r
JOIN purchase_items pi ON pi.id = r.purchase_item_id
LEFT JOIN window_spend ws ON ws.purchase_item_id = r.purchase_item_id
ORDER BY r.menu_item_id, r.usage_pct DESC`, from, to)
	if err != nil {
		return err
	}
	defer breakdownRows.Close()
	for breakdownRows.Next() {
		var menuItemID string
		var ingr IngredientAlloc
		if err := breakdownRows.Scan(&menuItemID, &ingr.PurchaseItemDescription, &ingr.UsagePct, &ingr.AllocatedCost); err != nil {
			return err
		}
		if idx, ok := rowIndex[menuItemID]; ok {
			rows[idx].Ingredients = append(rows[idx].Ingredients, ingr)
		}
	}
	return breakdownRows.Err()
}

// loadUnallocatedBreakdown returns the by_ingredient breakdown of unallocated dollars per D-17.
func loadUnallocatedBreakdown(ctx context.Context, pool *pgxpool.Pool, from, to string, total float64) (UnallocatedBreakdown, error) {
	rows, err := pool.Query(ctx, `
WITH window_spend AS (
  SELECT pli.purchase_item_id,
         SUM((pli.quantity * pli.price) *
             COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)) AS spend_incl_tax
  FROM purchase_line_items pli
  JOIN purchase_events pe ON pe.id = pli.purchase_event_id
  WHERE pe.event_date BETWEEN $1 AND $2 AND pli.purchase_item_id IS NOT NULL
  GROUP BY pli.purchase_item_id
),
purchase_item_alloc AS (
  SELECT purchase_item_id, COALESCE(SUM(usage_pct), 0) AS sum_pct
  FROM recipes GROUP BY purchase_item_id
)
SELECT
  pi.description AS purchase_item_description,
  ROUND((ws.spend_incl_tax * (1 - COALESCE(pia.sum_pct, 0) / 100.0))::numeric, 2) AS amount,
  CASE
    WHEN COALESCE(pia.sum_pct, 0) = 0   THEN 'no recipe'
    WHEN COALESCE(pia.sum_pct, 0) < 100 THEN 'partial allocation (' || pia.sum_pct || '%)'
    ELSE NULL
  END AS reason
FROM window_spend ws
JOIN purchase_items pi ON pi.id = ws.purchase_item_id
LEFT JOIN purchase_item_alloc pia ON pia.purchase_item_id = ws.purchase_item_id
WHERE COALESCE(pia.sum_pct, 0) < 100
ORDER BY amount DESC`, from, to)
	if err != nil {
		return UnallocatedBreakdown{}, err
	}
	defer rows.Close()
	out := UnallocatedBreakdown{Total: total, ByIngredient: []UnallocatedDetail{}}
	for rows.Next() {
		var d UnallocatedDetail
		var reason *string
		if err := rows.Scan(&d.PurchaseItemDescription, &d.Amount, &reason); err != nil {
			return UnallocatedBreakdown{}, err
		}
		if reason != nil {
			d.Reason = *reason
		}
		out.ByIngredient = append(out.ByIngredient, d)
	}
	return out, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// Recipes CRUD handlers (Plan 03) — cookie-auth-protected. Mounted under the
// auth.Middleware group at main.go inside r.Route("/inventory/recipes", ...).
// The 422 sum_exceeds_100 envelope shape `{"error":"sum_exceeds_100",
// "conflict_menu_item":"<name>","conflict_pct":<n>}` is the contract Plan 05's
// frontend depends on for rollback messaging (D-03).
// ─────────────────────────────────────────────────────────────────────────────

// validateUsagePct returns "" if value is valid; otherwise an error sentinel string
// suitable for inclusion in a 422 envelope. Per Pitfall 7: 0..100 and multiple of 5.
// Float math: NUMERIC(5,2) stored as float64 — `usage_pct*100` integer-converts
// safely for values like 45.0 (= 4500) and we check % 500 == 0.
func validateUsagePct(v float64) string {
	if v < 0 || v > 100 {
		return "usage_pct must be between 0 and 100"
	}
	cents := int(v * 100)
	if cents%500 != 0 {
		return "usage_pct must be a multiple of 5"
	}
	return ""
}

// chicagoWeekWindow returns (from, to) as YYYY-MM-DD for the last 7 days ending TODAY
// in America/Chicago, used as the default range for ListRecipesHandler when query
// params are missing. Falls back to the past 7 calendar days in UTC if TZ load fails.
func chicagoWeekWindow() (string, string) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		now := time.Now().UTC()
		return now.AddDate(0, 0, -7).Format("2006-01-02"), now.Format("2006-01-02")
	}
	now := time.Now().In(loc)
	return now.AddDate(0, 0, -7).Format("2006-01-02"), now.Format("2006-01-02")
}

// ListRecipesHandler returns the ingredient-first list for the Recipes tab.
// Cookie-auth-protected (registered under the auth.Middleware group in main.go).
// Query params:
//
//	from, to — YYYY-MM-DD (optional; defaults to last 7 days ending today in Chicago)
func ListRecipesHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		if fromStr == "" && toStr == "" {
			fromStr, toStr = chicagoWeekWindow()
		}
		if _, err := time.Parse("2006-01-02", fromStr); err != nil {
			writeError(w, http.StatusBadRequest, "from must be YYYY-MM-DD")
			return
		}
		if _, err := time.Parse("2006-01-02", toStr); err != nil {
			writeError(w, http.StatusBadRequest, "to must be YYYY-MM-DD")
			return
		}
		if fromStr > toStr {
			writeError(w, http.StatusBadRequest, "from must be <= to")
			return
		}
		ingredients, err := ListIngredientsWithSpend(r.Context(), pool, fromStr, toStr)
		if err != nil {
			slog.Error("ListRecipes", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"from":        fromStr,
			"to":          toStr,
			"ingredients": ingredients,
		})
	}
}

// CreateRecipeHandler — POST /inventory/recipes
// Body: {"menu_item_id":"<uuid>","purchase_item_id":"<uuid>","usage_pct":<n>}
// On 422 sum_exceeds_100: response body names the largest sibling allocation per D-03.
func CreateRecipeHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			MenuItemID     string  `json:"menu_item_id"`
			PurchaseItemID string  `json:"purchase_item_id"`
			UsagePct       float64 `json:"usage_pct"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.MenuItemID == "" || input.PurchaseItemID == "" {
			writeError(w, http.StatusBadRequest, "menu_item_id and purchase_item_id required")
			return
		}
		if msg := validateUsagePct(input.UsagePct); msg != "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":  "invalid_usage_pct",
				"detail": msg,
			})
			return
		}
		id, sumAfter, err := CreateRecipe(r.Context(), pool, input.MenuItemID, input.PurchaseItemID, input.UsagePct)
		if errors.Is(err, ErrSumExceeds100) {
			// Find the largest sibling to name in the message. No specific recipe to exclude —
			// the rejected insert was rolled back, so all surviving rows are siblings.
			name, pct, _ := LargestSiblingAllocation(r.Context(), pool, input.PurchaseItemID, "00000000-0000-0000-0000-000000000000")
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":              "sum_exceeds_100",
				"conflict_menu_item": name,
				"conflict_pct":       pct,
			})
			return
		}
		if err != nil {
			// pgx returns SQLSTATE 23505 for unique_violation; the error string typically
			// contains "duplicate key value violates unique constraint".
			if strings.Contains(err.Error(), "unique_violation") ||
				strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "23505") {
				writeError(w, http.StatusConflict, "recipe_already_exists")
				return
			}
			slog.Error("CreateRecipe", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "sum_after": sumAfter})
	}
}

// UpdateRecipeHandler — PUT /inventory/recipes/{id}
// Body: {"usage_pct":<n>}
// On 422 sum_exceeds_100: names the largest sibling allocation (excluding this recipe).
func UpdateRecipeHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recipeID := chi.URLParam(r, "id")
		if recipeID == "" {
			writeError(w, http.StatusBadRequest, "recipe_id required")
			return
		}
		var input struct {
			UsagePct float64 `json:"usage_pct"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if msg := validateUsagePct(input.UsagePct); msg != "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":  "invalid_usage_pct",
				"detail": msg,
			})
			return
		}
		purchaseItemID, _, err := UpdateRecipeUsagePct(r.Context(), pool, recipeID, input.UsagePct)
		if errors.Is(err, ErrRecipeNotFound) {
			writeError(w, http.StatusNotFound, "recipe_not_found")
			return
		}
		if errors.Is(err, ErrSumExceeds100) {
			name, pct, _ := LargestSiblingAllocation(r.Context(), pool, purchaseItemID, recipeID)
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error":              "sum_exceeds_100",
				"conflict_menu_item": name,
				"conflict_pct":       pct,
			})
			return
		}
		if err != nil {
			slog.Error("UpdateRecipe", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteRecipeHandler — DELETE /inventory/recipes/{id}
func DeleteRecipeHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recipeID := chi.URLParam(r, "id")
		if recipeID == "" {
			writeError(w, http.StatusBadRequest, "recipe_id required")
			return
		}
		err := DeleteRecipe(r.Context(), pool, recipeID)
		if errors.Is(err, ErrRecipeNotFound) {
			writeError(w, http.StatusNotFound, "recipe_not_found")
			return
		}
		if err != nil {
			slog.Error("DeleteRecipe", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// MergeMenuItemHandler — POST /inventory/recipes/merge
// Body: {"source_menu_item_id":"<uuid>","target_menu_item_id":"<uuid>"}
// Re-points all recipe rows from source to target, then deletes the source menu_items row.
// Mirrors inventory.MergeItemsHandler / MergeVendorsHandler semantics (D-08).
func MergeMenuItemHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			SourceMenuItemID string `json:"source_menu_item_id"`
			TargetMenuItemID string `json:"target_menu_item_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if input.SourceMenuItemID == "" || input.TargetMenuItemID == "" {
			writeError(w, http.StatusBadRequest, "source_menu_item_id and target_menu_item_id required")
			return
		}
		rows, err := MergeMenuItem(r.Context(), pool, input.SourceMenuItemID, input.TargetMenuItemID)
		if err != nil {
			if strings.Contains(err.Error(), "cannot_merge_into_self") {
				writeError(w, http.StatusBadRequest, "cannot_merge_into_self")
				return
			}
			slog.Error("MergeMenuItem", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]int{"rows_re_pointed": rows})
	}
}

// DriftBannerHandler — GET /inventory/recipes/drift
// Returns the latest week's drift_check_results.payload as JSON. Empty object
// `{}` when no drift_check_results rows exist (D-22 clean week → banner hidden).
//
// Cookie-auth-protected (registered under the auth.Middleware group in main.go,
// inside the existing /inventory/recipes route block from Plan 03).
func DriftBannerHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payloadBytes []byte
		err := pool.QueryRow(r.Context(),
			`SELECT payload FROM drift_check_results ORDER BY week_start DESC LIMIT 1`,
		).Scan(&payloadBytes)
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
		if err != nil {
			slog.Error("DriftBanner", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payloadBytes)
	}
}
