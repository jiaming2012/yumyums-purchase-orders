package recipes

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// computeDrift queries the database for the three drift signals (D-21) over the
// inclusive date range [from, to] and returns a DriftCheckResult.
//
// Thresholds (D-21):
//
//	Unallocated: SUM(usage_pct) < 90 AND last-week spend > 0
//	Divergence:  abs(configured - actualImpliedPct) > 20  (per-recipe-row)
//	Zero-sales:  menu_items appearing in any recipe row with 0 units_sold in window
//
// Only sections with at least one item are appended — HasDrift() therefore
// reflects whether any threshold tripped.
func computeDrift(ctx context.Context, pool *pgxpool.Pool, from, to string) (DriftCheckResult, error) {
	result := DriftCheckResult{
		WeekStart: from,
		Sections:  []DriftSection{},
	}

	unallocItems, err := queryUnallocatedDrift(ctx, pool, from, to)
	if err != nil {
		return result, fmt.Errorf("unallocated: %w", err)
	}
	if len(unallocItems) > 0 {
		result.Sections = append(result.Sections, DriftSection{
			Kind:    "unallocated",
			Heading: fmt.Sprintf("%d unallocated", len(unallocItems)),
			Items:   unallocItems,
		})
	}

	divItems, err := queryDivergenceDrift(ctx, pool, from, to)
	if err != nil {
		return result, fmt.Errorf("divergence: %w", err)
	}
	if len(divItems) > 0 {
		result.Sections = append(result.Sections, DriftSection{
			Kind:    "divergence",
			Heading: fmt.Sprintf("%d stale", len(divItems)),
			Items:   divItems,
		})
	}

	zeroItems, err := queryZeroSalesDrift(ctx, pool, from, to)
	if err != nil {
		return result, fmt.Errorf("zero_sales: %w", err)
	}
	if len(zeroItems) > 0 {
		result.Sections = append(result.Sections, DriftSection{
			Kind:    "zero_sales",
			Heading: fmt.Sprintf("%d dish gone", len(zeroItems)),
			Items:   zeroItems,
		})
	}

	return result, nil
}

// queryUnallocatedDrift finds purchase_items with SUM(usage_pct) < 90 AND
// last-week spend > 0. Tax-inclusive spend per Pitfall 4 (Plan 02 pattern).
func queryUnallocatedDrift(ctx context.Context, pool *pgxpool.Pool, from, to string) ([]DriftItem, error) {
	rows, err := pool.Query(ctx, `
WITH window_spend AS (
  SELECT pli.purchase_item_id,
         SUM((pli.quantity * pli.price) *
             COALESCE(pe.total / NULLIF(pe.total - pe.tax, 0), 1)) AS spend
  FROM purchase_line_items pli
  JOIN purchase_events pe ON pe.id = pli.purchase_event_id
  WHERE pe.event_date BETWEEN $1 AND $2 AND pli.purchase_item_id IS NOT NULL
  GROUP BY pli.purchase_item_id
),
sum_pct AS (
  SELECT purchase_item_id, COALESCE(SUM(usage_pct), 0) AS sp
  FROM recipes GROUP BY purchase_item_id
)
SELECT pi.id::text, pi.description,
       ROUND((ws.spend * (1 - COALESCE(s.sp, 0) / 100.0))::numeric, 2) AS unalloc
FROM window_spend ws
JOIN purchase_items pi ON pi.id = ws.purchase_item_id
LEFT JOIN sum_pct s ON s.purchase_item_id = ws.purchase_item_id
WHERE COALESCE(s.sp, 0) < 90 AND ws.spend > 0
ORDER BY unalloc DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DriftItem{}
	for rows.Next() {
		var id, desc string
		var amount float64
		if err := rows.Scan(&id, &desc, &amount); err != nil {
			return nil, err
		}
		a := amount
		items = append(items, DriftItem{
			PurchaseItemID: id,
			Label:          fmt.Sprintf("%s ($%.0f unalloc)", desc, amount),
			AmountUnalloc:  &a,
		})
	}
	return items, rows.Err()
}

// queryDivergenceDrift inspects each recipe row and compares its configured
// usage_pct against the actual implied % derived from the prior week's revenue
// distribution across menu_items sharing the ingredient. Diff > 20 → flagged.
func queryDivergenceDrift(ctx context.Context, pool *pgxpool.Pool, from, to string) ([]DriftItem, error) {
	rows, err := pool.Query(ctx, `
WITH menu_revenue AS (
  SELECT menu_item_id, COALESCE(SUM(gross_amount), 0) AS rev
  FROM daily_menu_sales WHERE business_date BETWEEN $1 AND $2
  GROUP BY menu_item_id
),
ingredient_totals AS (
  SELECT r.purchase_item_id, SUM(COALESCE(mr.rev, 0)) AS total_rev
  FROM recipes r LEFT JOIN menu_revenue mr ON mr.menu_item_id = r.menu_item_id
  GROUP BY r.purchase_item_id
)
SELECT r.id::text, mi.name, pi.description,
       r.usage_pct AS configured,
       CASE WHEN it.total_rev > 0
            THEN ROUND((COALESCE(mr.rev, 0) / it.total_rev * 100)::numeric, 2)
            ELSE NULL END AS actual_implied
FROM recipes r
JOIN menu_items mi ON mi.id = r.menu_item_id
JOIN purchase_items pi ON pi.id = r.purchase_item_id
LEFT JOIN menu_revenue mr ON mr.menu_item_id = r.menu_item_id
LEFT JOIN ingredient_totals it ON it.purchase_item_id = r.purchase_item_id`,
		from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DriftItem{}
	for rows.Next() {
		var recipeID, menuName, ingrDesc string
		var configured float64
		var actual *float64
		if err := rows.Scan(&recipeID, &menuName, &ingrDesc, &configured, &actual); err != nil {
			return nil, err
		}
		if actual == nil {
			continue
		}
		if math.Abs(configured-*actual) > 20 {
			c, a := configured, *actual
			items = append(items, DriftItem{
				Label: fmt.Sprintf("%s — set to %.0f%% on %s, actual mix says %.0f%%",
					ingrDesc, configured, menuName, *actual),
				ConfiguredPct: &c,
				ActualPct:     &a,
			})
		}
	}
	return items, rows.Err()
}

// queryZeroSalesDrift finds menu_items appearing in any recipe row with 0
// units_sold in the window (covers D-07 persistence + zero-sales drift).
func queryZeroSalesDrift(ctx context.Context, pool *pgxpool.Pool, from, to string) ([]DriftItem, error) {
	rows, err := pool.Query(ctx, `
SELECT DISTINCT mi.id::text, mi.name
FROM recipes r
JOIN menu_items mi ON mi.id = r.menu_item_id
LEFT JOIN (
  SELECT menu_item_id, SUM(units_sold) AS units
  FROM daily_menu_sales WHERE business_date BETWEEN $1 AND $2
  GROUP BY menu_item_id
) ms ON ms.menu_item_id = mi.id
WHERE COALESCE(ms.units, 0) = 0
ORDER BY mi.name`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DriftItem{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		items = append(items, DriftItem{
			MenuItemID: id,
			Label:      fmt.Sprintf("%s — no sales last week", name),
		})
	}
	return items, rows.Err()
}

// formatCliqMessage produces the Cliq body for the drift alert per UI-SPEC
// Component 6. Mirrors the in-app banner with a prefix and a deep link to the
// Recipes tab (D-19).
func formatCliqMessage(result DriftCheckResult, weekStart, baseURL string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[HQ Recipes drift check — week of %s]\n", weekStart))
	sb.WriteString(fmt.Sprintf("%d ingredients drifted last week.\n\n", result.TotalFlagged()))
	for _, sec := range result.Sections {
		sb.WriteString(sec.Heading)
		sb.WriteString(": ")
		labels := make([]string, len(sec.Items))
		for i, it := range sec.Items {
			labels[i] = it.Label
		}
		sb.WriteString(strings.Join(labels, ", "))
		sb.WriteString("\n")
	}
	sb.WriteString(fmt.Sprintf("\nOpen Recipes: %s/inventory.html#tab=4", baseURL))
	return sb.String()
}
