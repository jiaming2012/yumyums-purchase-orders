package recipes

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

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
			log.Printf("MenuCogs summary query: %v", err)
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
				log.Printf("MenuCogs summary scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			row.IngredientCostPerUnit = perUnit
			rowIndex[row.MenuItemID] = len(rowsOut)
			rowsOut = append(rowsOut, row)
		}
		if err := summaryRows.Err(); err != nil {
			log.Printf("MenuCogs summary iter: %v", err)
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
			log.Printf("MenuCogs unallocated query: %v", err)
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
				log.Printf("MenuCogs breakdown ingredient query: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			unalloc, err := loadUnallocatedBreakdown(ctx, pool, fromStr, toStr, totalUnalloc)
			if err != nil {
				log.Printf("MenuCogs unallocated breakdown query: %v", err)
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
