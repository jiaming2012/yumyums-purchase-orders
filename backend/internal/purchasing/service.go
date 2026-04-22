package purchasing

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrPONotDraft is returned when an operation requires the PO to be in draft status.
var ErrPONotDraft = errors.New("purchase order is not in draft status")

// CurrentWeekStart returns the Monday of the current week in America/Chicago timezone as YYYY-MM-DD.
func CurrentWeekStart() string {
	loc, _ := time.LoadLocation("America/Chicago")
	now := time.Now().In(loc)
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	return monday.Format("2006-01-02")
}

// GetOrCreateOrder gets or creates the purchase order for the current week.
func GetOrCreateOrder(ctx context.Context, pool *pgxpool.Pool) (*PurchaseOrder, error) {
	weekStart := CurrentWeekStart()

	var po PurchaseOrder
	err := pool.QueryRow(ctx, `
		INSERT INTO purchase_orders (week_start, status)
		VALUES ($1, 'draft')
		ON CONFLICT (week_start) DO UPDATE SET week_start = EXCLUDED.week_start
		RETURNING id, week_start, status, version, created_at
	`, weekStart).Scan(&po.ID, &po.WeekStart, &po.Status, &po.Version, &po.CreatedAt)
	if err != nil {
		return nil, err
	}

	lineItems, err := GetOrderLineItems(ctx, pool, po.ID)
	if err != nil {
		return nil, err
	}
	po.LineItems = lineItems
	return &po, nil
}

// GetOrderByID fetches a purchase order by ID with its line items.
func GetOrderByID(ctx context.Context, pool *pgxpool.Pool, poID string) (*PurchaseOrder, error) {
	var po PurchaseOrder
	err := pool.QueryRow(ctx, `
		SELECT id, week_start, status, version, created_at
		FROM purchase_orders
		WHERE id = $1
	`, poID).Scan(&po.ID, &po.WeekStart, &po.Status, &po.Version, &po.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	lineItems, err := GetOrderLineItems(ctx, pool, po.ID)
	if err != nil {
		return nil, err
	}
	po.LineItems = lineItems
	return &po, nil
}

// GetOrderLineItems fetches line items for a PO, joined with item and user info.
func GetOrderLineItems(ctx context.Context, pool *pgxpool.Pool, poID string) ([]POLineItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			li.id, li.po_id, li.purchase_item_id,
			pi.description AS item_name,
			ig.name AS group_name,
			pi.photo_url, pi.store_location,
			li.quantity, li.unit,
			li.added_by AS added_by_id,
			COALESCE(NULLIF(u.nickname, ''), u.first_name || ' ' || LEFT(u.last_name, 1) || '.') AS added_by_name,
			li.created_at, li.updated_at
		FROM po_line_items li
		JOIN purchase_items pi ON pi.id = li.purchase_item_id
		LEFT JOIN item_groups ig ON ig.id = pi.group_id
		JOIN users u ON u.id = li.added_by
		WHERE li.po_id = $1
		ORDER BY ig.name NULLS LAST, pi.description
	`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []POLineItem{}
	for rows.Next() {
		var li POLineItem
		if err := rows.Scan(
			&li.ID, &li.POID, &li.PurchaseItemID,
			&li.ItemName,
			&li.GroupName,
			&li.PhotoURL, &li.StoreLocation,
			&li.Quantity, &li.Unit,
			&li.AddedByID,
			&li.AddedByName,
			&li.CreatedAt, &li.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, li)
	}
	return items, rows.Err()
}

// UpsertLineItems replaces the line items on a draft PO inside a transaction.
// Returns ErrPONotDraft if the PO is not in draft status.
func UpsertLineItems(ctx context.Context, pool *pgxpool.Pool, poID string, userID string, items []UpsertLineItemInput) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Verify PO is draft
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id = $1`, poID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPONotDraft
		}
		return err
	}
	if status != "draft" {
		return ErrPONotDraft
	}

	// Collect incoming purchase_item_ids
	keepIDs := make([]string, 0, len(items))
	for _, it := range items {
		if it.Quantity > 0 {
			keepIDs = append(keepIDs, it.PurchaseItemID)
		}
	}

	// Delete line items not in the incoming list
	if _, err = tx.Exec(ctx, `
		DELETE FROM po_line_items
		WHERE po_id = $1 AND purchase_item_id != ALL($2::uuid[])
	`, poID, keepIDs); err != nil {
		return err
	}

	// Upsert each item with quantity > 0
	for _, it := range items {
		if it.Quantity <= 0 {
			continue
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO po_line_items (po_id, purchase_item_id, quantity, unit, added_by, updated_at)
			VALUES ($1, $2, $3, $4, $5, now())
			ON CONFLICT (po_id, purchase_item_id)
			DO UPDATE SET quantity = EXCLUDED.quantity, unit = EXCLUDED.unit, updated_at = now()
		`, poID, it.PurchaseItemID, it.Quantity, it.Unit, userID); err != nil {
			return err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// GetSuggestions returns items below their group's low threshold not already on the PO.
func GetSuggestions(ctx context.Context, pool *pgxpool.Pool, poID string) ([]OrderSuggestion, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			pi.id AS purchase_item_id,
			pi.description AS item_name,
			pi.photo_url, pi.store_location,
			ig.name AS group_name,
			ig.low_threshold,
			COALESCE(sco.quantity, COALESCE(stock.total_qty, 0)) AS current_stock,
			GREATEST(1, ig.low_threshold - COALESCE(sco.quantity, COALESCE(stock.total_qty, 0))) AS suggested_qty,
			'' AS unit
		FROM purchase_items pi
		JOIN item_groups ig ON ig.id = pi.group_id
		LEFT JOIN stock_count_overrides sco ON sco.item_description = pi.description
		LEFT JOIN (
			SELECT pli.purchase_item_id, SUM(pli.quantity)::int AS total_qty
			FROM purchase_line_items pli
			JOIN purchase_events pe ON pe.id = pli.purchase_event_id
			GROUP BY pli.purchase_item_id
		) stock ON stock.purchase_item_id = pi.id
		WHERE NOT EXISTS (
			SELECT 1 FROM po_line_items pol WHERE pol.po_id = $1 AND pol.purchase_item_id = pi.id
		)
		GROUP BY pi.id, pi.description, pi.photo_url, pi.store_location, ig.name, ig.low_threshold, sco.quantity, stock.total_qty
		HAVING COALESCE(sco.quantity, COALESCE(stock.total_qty, 0)) < ig.low_threshold
		ORDER BY ig.name, pi.description
	`, poID)
	if err != nil {
		log.Printf("GetSuggestions query: %v", err)
		return nil, err
	}
	defer rows.Close()

	suggestions := []OrderSuggestion{}
	for rows.Next() {
		var s OrderSuggestion
		if err := rows.Scan(
			&s.PurchaseItemID, &s.ItemName,
			&s.PhotoURL, &s.StoreLocation,
			&s.GroupName, &s.LowThreshold,
			&s.CurrentStock, &s.SuggestedQty,
			&s.Unit,
		); err != nil {
			return nil, err
		}
		suggestions = append(suggestions, s)
	}
	return suggestions, rows.Err()
}
