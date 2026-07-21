package inventory

import (
	"log/slog"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── GET /api/v1/inventory/trends — design §2.2 AS AMENDED 2026-07-20 ─────────
//
// Amendments 1/2/3 (ledger §T-19, decisions 29/30/31) are the spec; the
// originally-signed §2.2 text is superseded where it conflicts.

// TrendsWindow is the fixed server-computed window (Assumption A1): 12 weeks
// ending today, bucketed by date_trunc('week', …) (ISO weeks, Monday start).
type TrendsWindow struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Weeks int    `json:"weeks"`
}

// TrendsGroup is one item_group appearing in the window. The D2 "Ungrouped"
// pseudo-group is emitted here too, with ID UngroupedGroupID.
type TrendsGroup struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TrendsCell is one week×group spend bucket. Sparse — zero cells are omitted
// and the client renders missing cells as 0.
type TrendsCell struct {
	WeekStart string  `json:"week_start"`
	GroupID   string  `json:"group_id"`
	Spend     float64 `json:"spend"`
}

// TrendsUnlinkedWeek is per-week spend on line items with
// purchase_item_id IS NULL (FR-6b) — excluded from group buckets.
type TrendsUnlinkedWeek struct {
	WeekStart string  `json:"week_start"`
	Spend     float64 `json:"spend"`
}

// TrendsCompleteness carries the Amendment 2 and Amendment 3 figures: money
// that is real but cannot be placed in a week×group cell.
type TrendsCompleteness struct {
	// PendingTotal / PendingCount — Amendment 2. Unreviewed receipts have no
	// linked line items, so they are excluded from `cells` and surfaced here.
	PendingTotal float64 `json:"pending_total"`
	PendingCount int     `json:"pending_count"`
	// UnitemizedRemainder — Amendment 3, window-summed. NOT an addend to the
	// reconciliation identity; period-summary does not count it either.
	UnitemizedRemainder float64 `json:"unitemized_remainder"`
	// ReconcilesToCogsExclTax is the endpoint's own Σcells + Σunlinked +
	// pending_total, published so a mismatch is visible in the response.
	ReconcilesToCogsExclTax float64 `json:"reconciles_to_cogs_excl_tax"`
}

// TrendsResponse is the amended §2.2 response shape.
type TrendsResponse struct {
	Window        TrendsWindow         `json:"window"`
	Groups        []TrendsGroup        `json:"groups"`
	Cells         []TrendsCell         `json:"cells"`
	Unlinked      []TrendsUnlinkedWeek `json:"unlinked"`
	UnlinkedTotal float64              `json:"unlinked_total"`
	Completeness  TrendsCompleteness   `json:"completeness"`
}

// UngroupedGroupID is the sentinel group id for the D2 "Ungrouped"
// pseudo-group: line items linked to a purchase_item whose group_id IS NULL.
const UngroupedGroupID = "ungrouped"

// UngroupedGroupName is the display name for that pseudo-group.
const UngroupedGroupName = "Ungrouped"

// TrendsWeeks is the fixed window length (Assumption A1).
const TrendsWeeks = 12

// trendsWindow computes the fixed window server-side: the 12 ISO weeks ending
// today. `from` is the Monday of the week TrendsWeeks-1 weeks before the
// current week, `to` is today.
func trendsWindow(now time.Time) TrendsWindow {
	off := (int(now.Weekday()) + 6) % 7 // days since Monday
	monday := now.AddDate(0, 0, -off)
	from := monday.AddDate(0, 0, -7*(TrendsWeeks-1))
	return TrendsWindow{
		From:  from.Format("2006-01-02"),
		To:    now.Format("2006-01-02"),
		Weeks: TrendsWeeks,
	}
}

// TrendsHandler returns confirmed COGS spend bucketed by ISO week × item group
// over the fixed 12-week window (design §2.2 as amended, FR-1/FR-6b).
//
// Cookie-auth, mounted inside the authenticated /inventory group. The per-tab
// grant gate is NOT applied here — it arrives with the inventory-tab-gating
// card and wraps this route from the outside.
//
// Rules implemented:
//   - Amendment 1 — `pe.mercury_category = ANY($3)` against the SAME
//     cogsAllowlist period-summary is constructed with. NULL categories are
//     excluded, exactly as period-summary excludes them. Do not hardcode.
//   - Amendment 2 — unreviewed (pending) receipts have no linked line items,
//     so they cannot be bucketed; they are excluded from `cells` and surfaced
//     in `completeness.pending_total` / `pending_count`. The eligible
//     population mirrors period-summary's pending CTE clause-for-clause.
//   - Amendment 3 — NO tax proration. Cell spend is SUM(quantity * price) at
//     face value, matching period-summary's `lines` term. The per-event
//     unitemized remainder ((total - tax) - Σlines) is reported separately and
//     is deliberately NOT an addend to the identity.
//   - FR-6b — lines with purchase_item_id IS NULL are excluded from group
//     buckets and reported per-week in `unlinked` + `unlinked_total`.
//   - D2 — lines linked to a purchase_item whose group_id IS NULL bucket into
//     an explicit "Ungrouped" pseudo-group; never dropped, never folded into
//     `unlinked`.
//
// The reconciliation identity this guarantees, asserted in trends_test.go by
// calling PeriodSummaryHandler on the same window:
//
//	Σcells + Σunlinked + pending_total == period_summary.cogs_excl_tax
//
// `cells` is sparse: only non-empty week×group buckets are emitted.
//
// Response (200): inventory.TrendsResponse. Errors: 500 on internal DB error.
func TrendsHandler(pool *pgxpool.Pool, cogsAllowlist []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		win := trendsWindow(time.Now())

		// 1) Week × group cells + per-week unlinked. Face value, no proration
		//    (Amendment 3); allowlist-filtered (Amendment 1).
		//
		//    `linked` is what separates the two group_id = '' populations:
		//    purchase_item_id IS NULL (→ unlinked) from a linked item whose
		//    group_id IS NULL (→ D2 Ungrouped).
		rows, err := pool.Query(r.Context(), `
			SELECT date_trunc('week', pe.event_date)::date::text        AS week_start,
			       (pli.purchase_item_id IS NOT NULL)                   AS linked,
			       COALESCE(pi.group_id::text, '')                      AS group_id,
			       COALESCE(ig.name, '')                                AS group_name,
			       ROUND(SUM(pli.quantity * pli.price)::numeric, 2)     AS spend
			FROM purchase_line_items pli
			JOIN purchase_events pe ON pe.id = pli.purchase_event_id
			LEFT JOIN purchase_items pi ON pi.id = pli.purchase_item_id
			LEFT JOIN item_groups   ig ON ig.id = pi.group_id
			WHERE pe.event_date BETWEEN $1::date AND $2::date
			  AND pe.mercury_category = ANY($3)
			GROUP BY 1, 2, 3, 4
			ORDER BY 1, 4, 3`, win.From, win.To, cogsAllowlist)
		if err != nil {
			slog.Error("Trends cells query failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		cells := []TrendsCell{}
		groupNames := map[string]string{}
		unlinkedByWeek := map[string]float64{}
		unlinkedOrder := []string{}
		var unlinkedTotal float64

		for rows.Next() {
			var weekStart, groupID, groupName string
			var linked bool
			var spend float64
			if err := rows.Scan(&weekStart, &linked, &groupID, &groupName, &spend); err != nil {
				slog.Error("Trends cells scan failed", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			if spend == 0 {
				continue // sparse
			}
			if !linked {
				// FR-6b: purchase_item_id IS NULL — never a group bucket.
				if _, seen := unlinkedByWeek[weekStart]; !seen {
					unlinkedOrder = append(unlinkedOrder, weekStart)
				}
				unlinkedByWeek[weekStart] += spend
				unlinkedTotal += spend
				continue
			}
			if groupID == "" {
				// D2: linked, but the item has no group.
				groupID, groupName = UngroupedGroupID, UngroupedGroupName
			}
			groupNames[groupID] = groupName
			cells = append(cells, TrendsCell{WeekStart: weekStart, GroupID: groupID, Spend: spend})
		}
		if err := rows.Err(); err != nil {
			slog.Error("Trends cells iteration failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Multiple raw rows can collapse into the same week×Ungrouped cell
		// (several groupless items in one week), so merge before emitting.
		merged := map[string]int{}
		out := cells[:0]
		for _, c := range cells {
			k := c.WeekStart + "|" + c.GroupID
			if i, ok := merged[k]; ok {
				out[i].Spend = round2(out[i].Spend + c.Spend)
				continue
			}
			merged[k] = len(out)
			out = append(out, c)
		}
		cells = out

		// 2) Amendment 2 — eligible pending. Clause-for-clause identical to
		//    the pending CTE in PeriodSummaryHandler (handler.go:1345-1351) so
		//    the two endpoints agree on the population by construction.
		var pendingTotal float64
		var pendingCount int
		err = pool.QueryRow(r.Context(), `
			SELECT ROUND(COALESCE(SUM(ABS(bank_total)), 0)::numeric, 2), COUNT(*)
			FROM pending_purchases
			WHERE COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date)
			        BETWEEN $1::date AND $2::date
			  AND confirmed_at IS NULL
			  AND discarded_at IS NULL
			  AND mercury_category = ANY($3)
			  AND reason != 'no_attachment_on_bank_tx'`,
			win.From, win.To, cogsAllowlist).Scan(&pendingTotal, &pendingCount)
		if err != nil {
			slog.Error("Trends pending query failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// 3) Amendment 3 — window-summed unitemized remainder. Explains
		//    receipt-total vs line-item coverage; NOT part of the identity.
		var unitemized float64
		err = pool.QueryRow(r.Context(), `
			SELECT ROUND(COALESCE(SUM((pe.total - pe.tax) - COALESCE(l.line_total, 0)), 0)::numeric, 2)
			FROM purchase_events pe
			LEFT JOIN (
			    SELECT purchase_event_id, SUM(quantity * price) AS line_total
			    FROM purchase_line_items
			    GROUP BY purchase_event_id
			) l ON l.purchase_event_id = pe.id
			WHERE pe.event_date BETWEEN $1::date AND $2::date
			  AND pe.mercury_category = ANY($3)`,
			win.From, win.To, cogsAllowlist).Scan(&unitemized)
		if err != nil {
			slog.Error("Trends unitemized-remainder query failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Groups: real groups by name, Ungrouped pinned last.
		groups := make([]TrendsGroup, 0, len(groupNames))
		for id, name := range groupNames {
			if id == UngroupedGroupID {
				continue
			}
			groups = append(groups, TrendsGroup{ID: id, Name: name})
		}
		sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
		if _, ok := groupNames[UngroupedGroupID]; ok {
			groups = append(groups, TrendsGroup{ID: UngroupedGroupID, Name: UngroupedGroupName})
		}

		unlinked := make([]TrendsUnlinkedWeek, 0, len(unlinkedOrder))
		for _, wk := range unlinkedOrder {
			unlinked = append(unlinked, TrendsUnlinkedWeek{WeekStart: wk, Spend: round2(unlinkedByWeek[wk])})
		}
		sort.Slice(unlinked, func(i, j int) bool { return unlinked[i].WeekStart < unlinked[j].WeekStart })

		// The published left-hand side of the identity, computed from exactly
		// the numbers this response carries — so any disagreement with
		// period-summary is visible in the payload itself.
		var cellSum float64
		for _, c := range cells {
			cellSum += c.Spend
		}
		reconciles := round2(cellSum + round2(unlinkedTotal) + pendingTotal)

		writeJSON(w, http.StatusOK, TrendsResponse{
			Window:        win,
			Groups:        groups,
			Cells:         cells,
			Unlinked:      unlinked,
			UnlinkedTotal: round2(unlinkedTotal),
			Completeness: TrendsCompleteness{
				PendingTotal:            pendingTotal,
				PendingCount:            pendingCount,
				UnitemizedRemainder:     unitemized,
				ReconcilesToCogsExclTax: reconciles,
			},
		})
	}
}

// round2 snaps a float to cents, keeping Go-side sums of DB-rounded cell
// values free of float64 representation drift.
func round2(f float64) float64 { return math.Round(f*100) / 100 }
