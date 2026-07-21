package inventory

import (
	"net/http"
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

// TrendsHandler — STUB. Red-first (G3): trends_test.go is written and run
// against this no-op first, so the fixture is proven to fail before any
// implementation exists. The real handler lands in the next commit.
func TrendsHandler(pool *pgxpool.Pool, cogsAllowlist []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, TrendsResponse{
			Window:   trendsWindow(time.Now()),
			Groups:   []TrendsGroup{},
			Cells:    []TrendsCell{},
			Unlinked: []TrendsUnlinkedWeek{},
		})
	}
}
