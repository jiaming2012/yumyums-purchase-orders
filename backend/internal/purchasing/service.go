package purchasing

import (
	"context"
	"errors"
	"fmt"
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
		RETURNING id, week_start::text, status, version, created_at
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
		SELECT id, week_start::text, status, version, created_at
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

// GetActiveShoppingList returns the active shopping list with vendor sections and items, or nil if none.
func GetActiveShoppingList(ctx context.Context, pool *pgxpool.Pool) (*ShoppingList, error) {
	var sl ShoppingList
	err := pool.QueryRow(ctx, `
		SELECT sl.id, sl.po_id, po.week_start::text, sl.status, sl.assigned_to, sl.assigned_role, sl.created_at, sl.completed_at
		FROM shopping_lists sl
		JOIN purchase_orders po ON po.id = sl.po_id
		WHERE sl.status = 'active'
		LIMIT 1
	`).Scan(&sl.ID, &sl.POID, &sl.WeekStart, &sl.Status, &sl.AssignedTo, &sl.AssignedRole, &sl.CreatedAt, &sl.CompletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if err := loadShoppingListSections(ctx, pool, &sl); err != nil {
		return nil, err
	}
	return &sl, nil
}

// GetShoppingListByID returns a shopping list by ID with vendor sections and items. Used for history view.
func GetShoppingListByID(ctx context.Context, pool *pgxpool.Pool, listID string) (*ShoppingList, error) {
	var sl ShoppingList
	err := pool.QueryRow(ctx, `
		SELECT sl.id, sl.po_id, po.week_start::text, sl.status, sl.assigned_to, sl.assigned_role, sl.created_at, sl.completed_at
		FROM shopping_lists sl
		JOIN purchase_orders po ON po.id = sl.po_id
		WHERE sl.id = $1
	`, listID).Scan(&sl.ID, &sl.POID, &sl.WeekStart, &sl.Status, &sl.AssignedTo, &sl.AssignedRole, &sl.CreatedAt, &sl.CompletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if err := loadShoppingListSections(ctx, pool, &sl); err != nil {
		return nil, err
	}
	return &sl, nil
}

// loadShoppingListSections loads vendor sections and their items into a ShoppingList.
func loadShoppingListSections(ctx context.Context, pool *pgxpool.Pool, sl *ShoppingList) error {
	rows, err := pool.Query(ctx, `
		SELECT svs.id, svs.shopping_list_id, svs.vendor_id, svs.vendor_name, svs.status,
		       svs.completed_by, u.display_name, svs.completed_at
		FROM shopping_list_vendor_sections svs
		LEFT JOIN users u ON u.id = svs.completed_by
		WHERE svs.shopping_list_id = $1
		ORDER BY svs.vendor_name
	`, sl.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	sections := []ShoppingListVendorSection{}
	for rows.Next() {
		var sec ShoppingListVendorSection
		if err := rows.Scan(
			&sec.ID, &sec.ShoppingListID, &sec.VendorID, &sec.VendorName, &sec.Status,
			&sec.CompletedBy, &sec.CompletedByName, &sec.CompletedAt,
		); err != nil {
			return err
		}
		sections = append(sections, sec)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Load items for all sections in one query
	itemRows, err := pool.Query(ctx, `
		SELECT sli.id, sli.shopping_list_id, sli.vendor_section_id, sli.purchase_item_id,
		       sli.item_name, sli.photo_url, sli.store_location, sli.quantity, sli.unit,
		       sli.checked, sli.checked_by, u.display_name, sli.checked_at
		FROM shopping_list_items sli
		LEFT JOIN users u ON u.id = sli.checked_by
		WHERE sli.shopping_list_id = $1
		ORDER BY sli.item_name
	`, sl.ID)
	if err != nil {
		return err
	}
	defer itemRows.Close()

	// Index items by vendor_section_id
	itemsBySection := map[string][]ShoppingListItem{}
	for itemRows.Next() {
		var item ShoppingListItem
		if err := itemRows.Scan(
			&item.ID, &item.ShoppingListID, &item.VendorSectionID, &item.PurchaseItemID,
			&item.ItemName, &item.PhotoURL, &item.StoreLocation, &item.Quantity, &item.Unit,
			&item.Checked, &item.CheckedBy, &item.CheckedByName, &item.CheckedAt,
		); err != nil {
			return err
		}
		itemsBySection[item.VendorSectionID] = append(itemsBySection[item.VendorSectionID], item)
	}
	if err := itemRows.Err(); err != nil {
		return err
	}

	// Assign items to sections
	for i, sec := range sections {
		if items, ok := itemsBySection[sec.ID]; ok {
			sections[i].Items = items
		} else {
			sections[i].Items = []ShoppingListItem{}
		}
	}
	sl.VendorSections = sections
	return nil
}

// GetShoppingListHistory returns completed shopping lists with vendor sections (no items — items loaded on expand).
func GetShoppingListHistory(ctx context.Context, pool *pgxpool.Pool) ([]ShoppingList, error) {
	rows, err := pool.Query(ctx, `
		SELECT sl.id, sl.po_id, po.week_start::text, sl.status, sl.assigned_to, sl.assigned_role, sl.created_at, sl.completed_at
		FROM shopping_lists sl
		JOIN purchase_orders po ON po.id = sl.po_id
		WHERE sl.status = 'completed'
		ORDER BY po.week_start DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lists := []ShoppingList{}
	for rows.Next() {
		var sl ShoppingList
		if err := rows.Scan(
			&sl.ID, &sl.POID, &sl.WeekStart, &sl.Status, &sl.AssignedTo, &sl.AssignedRole, &sl.CreatedAt, &sl.CompletedAt,
		); err != nil {
			return nil, err
		}
		lists = append(lists, sl)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load vendor sections (with missing_count) for each list
	for i := range lists {
		secRows, err := pool.Query(ctx, `
			SELECT svs.id, svs.shopping_list_id, svs.vendor_id, svs.vendor_name, svs.status,
			       svs.completed_by, u.display_name, svs.completed_at
			FROM shopping_list_vendor_sections svs
			LEFT JOIN users u ON u.id = svs.completed_by
			WHERE svs.shopping_list_id = $1
			ORDER BY svs.vendor_name
		`, lists[i].ID)
		if err != nil {
			return nil, err
		}

		sections := []ShoppingListVendorSection{}
		for secRows.Next() {
			var sec ShoppingListVendorSection
			if err := secRows.Scan(
				&sec.ID, &sec.ShoppingListID, &sec.VendorID, &sec.VendorName, &sec.Status,
				&sec.CompletedBy, &sec.CompletedByName, &sec.CompletedAt,
			); err != nil {
				secRows.Close()
				return nil, err
			}
			sec.Items = []ShoppingListItem{}
			sections = append(sections, sec)
		}
		secRows.Close()
		if err := secRows.Err(); err != nil {
			return nil, err
		}
		lists[i].VendorSections = sections
	}

	return lists, nil
}

// CheckShoppingItem toggles the checked state on a shopping list item.
func CheckShoppingItem(ctx context.Context, pool *pgxpool.Pool, itemID string, checked bool, userID string) error {
	tag, err := pool.Exec(ctx, `
		UPDATE shopping_list_items
		SET checked = $2,
		    checked_by = CASE WHEN $2 THEN $3::uuid ELSE NULL END,
		    checked_at = CASE WHEN $2 THEN now() ELSE NULL END
		WHERE id = $1
	`, itemID, checked, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("shopping list item not found: %s", itemID)
	}
	return nil
}

// UpdateShoppingItemLocation updates store_location on both shopping_list_items AND purchase_items.
func UpdateShoppingItemLocation(ctx context.Context, pool *pgxpool.Pool, itemID string, storeLocation string) error {
	var purchaseItemID string
	err := pool.QueryRow(ctx, `
		UPDATE shopping_list_items SET store_location = $2 WHERE id = $1 RETURNING purchase_item_id
	`, itemID, storeLocation).Scan(&purchaseItemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("shopping list item not found: %s", itemID)
		}
		return err
	}

	_, err = pool.Exec(ctx, `UPDATE purchase_items SET store_location = $2 WHERE id = $1`, purchaseItemID, storeLocation)
	return err
}

// UpdateShoppingItemPhoto updates photo_url on both shopping_list_items AND purchase_items.
func UpdateShoppingItemPhoto(ctx context.Context, pool *pgxpool.Pool, itemID string, photoURL string) error {
	var purchaseItemID string
	err := pool.QueryRow(ctx, `
		UPDATE shopping_list_items SET photo_url = $2 WHERE id = $1 RETURNING purchase_item_id
	`, itemID, photoURL).Scan(&purchaseItemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("shopping list item not found: %s", itemID)
		}
		return err
	}

	_, err = pool.Exec(ctx, `UPDATE purchase_items SET photo_url = $2 WHERE id = $1`, purchaseItemID, photoURL)
	return err
}

// CompleteVendorSection marks a vendor section as completed and cascades to list/PO if all sections done.
// Returns whether the entire shopping list is now completed.
func CompleteVendorSection(ctx context.Context, pool *pgxpool.Pool, sectionID string, userID string) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Mark section completed
	var listID string
	err = tx.QueryRow(ctx, `
		UPDATE shopping_list_vendor_sections
		SET status = 'completed', completed_by = $2, completed_at = now()
		WHERE id = $1 AND status = 'pending'
		RETURNING shopping_list_id
	`, sectionID, userID).Scan(&listID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("vendor section not found or already completed: %s", sectionID)
		}
		return false, err
	}

	// Check if all sections for this list are completed
	var pendingCount int
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM shopping_list_vendor_sections WHERE shopping_list_id = $1 AND status = 'pending'
	`, listID).Scan(&pendingCount)
	if err != nil {
		return false, err
	}

	listCompleted := pendingCount == 0
	if listCompleted {
		// Update shopping list status
		_, err = tx.Exec(ctx, `
			UPDATE shopping_lists SET status = 'completed', completed_at = now() WHERE id = $1
		`, listID)
		if err != nil {
			return false, err
		}

		// Update associated PO status
		_, err = tx.Exec(ctx, `
			UPDATE purchase_orders SET status = 'completed' WHERE id = (SELECT po_id FROM shopping_lists WHERE id = $1)
		`, listID)
		if err != nil {
			return false, err
		}
	}

	// Call notify stub (Phase 17 alert hook) — called after section completed but before COMMIT
	if notifyErr := NotifyVendorComplete(ctx, pool, listID); notifyErr != nil {
		log.Printf("NotifyVendorComplete: %v", notifyErr)
	}

	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return listCompleted, nil
}

// NotifyVendorComplete is a no-op stub for Phase 17 alert wiring.
// Phase 17 will replace this with actual Zoho Cliq / email delivery.
func NotifyVendorComplete(ctx context.Context, pool *pgxpool.Pool, listID string) error {
	log.Printf("alert pending (Phase 17): vendor section completed for shopping list %s", listID)
	return nil
}

// GetSuggestions returns items below their group's low threshold not already on the PO.
// Mirrors the inventory stock query approach: starts from purchase_line_items, resolves
// catalog items via purchase_item_id or description fallback, applies group thresholds.
func GetSuggestions(ctx context.Context, pool *pgxpool.Pool, poID string) ([]OrderSuggestion, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			sub.purchase_item_id,
			sub.item_name,
			sub.photo_url,
			sub.store_location,
			sub.group_name,
			COALESCE(ig.low_threshold, 3) AS low_threshold,
			sub.current_stock,
			GREATEST(1, COALESCE(ig.low_threshold, 3) - sub.current_stock) AS suggested_qty,
			'' AS unit,
			CASE
				WHEN sub.current_stock <= COALESCE(ig.low_threshold, 3) THEN 'Low'
				ELSE 'Medium'
			END AS stock_level
		FROM (
			SELECT
				pi.id AS purchase_item_id,
				COALESCE(pi.description, pli.description) AS item_name,
				pi.photo_url,
				pi.store_location,
				ig.name AS group_name,
				ig.id AS group_id,
				COALESCE(
					sco.quantity,
					SUM(pli.quantity)::int
				) AS current_stock
			FROM purchase_line_items pli
			JOIN purchase_events pe ON pe.id = pli.purchase_event_id
			LEFT JOIN purchase_items pi ON pi.id = pli.purchase_item_id
			LEFT JOIN item_groups ig ON ig.id = pi.group_id
			LEFT JOIN stock_count_overrides sco ON sco.item_description = COALESCE(pi.description, pli.description)
			GROUP BY pi.id, COALESCE(pi.description, pli.description), pi.photo_url, pi.store_location, ig.name, ig.id, sco.quantity
		) sub
		LEFT JOIN item_groups ig ON ig.id = sub.group_id
		WHERE sub.purchase_item_id IS NOT NULL
		AND sub.current_stock <= COALESCE(ig.high_threshold, 10)
		AND sub.current_stock > 0
		AND NOT EXISTS (
			SELECT 1 FROM po_line_items pol WHERE pol.po_id = $1 AND pol.purchase_item_id = sub.purchase_item_id
		)
		ORDER BY sub.group_name NULLS LAST, sub.item_name
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
			&s.Unit, &s.StockLevel,
		); err != nil {
			return nil, err
		}
		suggestions = append(suggestions, s)
	}
	return suggestions, rows.Err()
}
