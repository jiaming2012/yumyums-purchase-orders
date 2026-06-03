package toast

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// writeJSON / writeError are duplicated from internal/inventory/handler.go.
// Phase 22 intentionally avoids introducing an internal/httpx package for two
// 5-line helpers — matches the existing per-package convention.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ListMenuItemsHandler returns menu_items joined with this-week aggregate
// (units_sold + gross from daily_menu_sales), filtered to last_seen >= since,
// ordered by last_seen DESC. No pagination — the menu item count is in the
// low hundreds even at peak menu size.
//
// Auth: caller mounts this inside the cookie-auth chi.Group (see Plan 05's
// cmd/server/main.go edits). It is NOT a service-token endpoint — that's the
// Phase 21 period-summary, which is different.
//
// Query params:
//
//	since=YYYY-MM-DD  optional, defaults to 7 days ago
//
// Response:
//
//	200 [MenuItemWithSales, ...]  (may be empty array)
//	400 {"error":"since must be YYYY-MM-DD"}
//	500 {"error":"internal_error"}
func ListMenuItemsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sinceStr := r.URL.Query().Get("since")
		if sinceStr == "" {
			sinceStr = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
		}
		if _, err := time.Parse("2006-01-02", sinceStr); err != nil {
			writeError(w, http.StatusBadRequest, "since must be YYYY-MM-DD")
			return
		}

		rows, err := pool.Query(r.Context(), `
			SELECT mi.id, mi.master_id, mi.name, mi.menu, mi.menu_group, mi.menu_subgroup,
				mi.last_seen, mi.created_at,
				COALESCE(SUM(CASE WHEN dms.business_date >= $1 THEN dms.units_sold ELSE 0 END), 0)::int AS units_week,
				COALESCE(SUM(CASE WHEN dms.business_date >= $1 THEN dms.gross_amount ELSE 0 END), 0)::float8 AS gross_week
			FROM menu_items mi
			LEFT JOIN daily_menu_sales dms ON dms.menu_item_id = mi.id
			WHERE mi.last_seen >= $1
			GROUP BY mi.id
			ORDER BY mi.last_seen DESC`, sinceStr)
		if err != nil {
			log.Printf("ListMenuItems query: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		out := []MenuItemWithSales{}
		for rows.Next() {
			var m MenuItemWithSales
			var lastSeen time.Time
			if err := rows.Scan(
				&m.ID, &m.MasterID, &m.Name, &m.Menu, &m.MenuGroup, &m.MenuSubgroup,
				&lastSeen, &m.CreatedAt,
				&m.UnitsSoldThisWeek, &m.GrossThisWeek,
			); err != nil {
				log.Printf("ListMenuItems scan: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			m.LastSeen = lastSeen.Format("2006-01-02")
			out = append(out, m)
		}
		if err := rows.Err(); err != nil {
			log.Printf("ListMenuItems rows.Err: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
