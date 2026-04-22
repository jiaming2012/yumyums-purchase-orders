package purchasing

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// ErrPONotDraft is returned when an operation requires the PO to be in draft status.
var ErrPONotDraft = errors.New("purchase order is not in draft status")

// ErrPONotLocked is returned when an operation requires the PO to be in locked status.
var ErrPONotLocked = errors.New("po_not_locked")

// ErrPOAlreadyApproved is returned when the PO has already been approved.
var ErrPOAlreadyApproved = errors.New("po_already_approved")

// ErrActiveShoppingListExists is returned when an active shopping list already exists.
var ErrActiveShoppingListExists = errors.New("active_shopping_list_exists")

// ErrUnlockAfterApproval is returned when trying to unlock an already-approved PO.
var ErrUnlockAfterApproval = errors.New("cannot_unlock_after_approval")

// isAdmin returns true if the user is a superadmin or has the "admin" role.
func isAdmin(user *auth.User) bool {
	if user == nil {
		return false
	}
	if user.IsSuperadmin {
		return true
	}
	for _, r := range user.Roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

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

// GetOrCreateOrder returns the most recent draft PO. If the current week's PO
// is locked/approved, it finds or creates the next week's draft instead.
// The Order tab always shows an editable draft — never a locked PO.
func GetOrCreateOrder(ctx context.Context, pool *pgxpool.Pool) (*PurchaseOrder, error) {
	// First, try to find any existing draft PO (could be this week or next)
	var po PurchaseOrder
	err := pool.QueryRow(ctx, `
		SELECT id, week_start::text, status, version, created_at,
		       locked_at, approved_at, approved_by
		FROM purchase_orders
		WHERE status = 'draft'
		ORDER BY week_start DESC
		LIMIT 1
	`).Scan(&po.ID, &po.WeekStart, &po.Status, &po.Version, &po.CreatedAt,
		&po.LockedAt, &po.ApprovedAt, &po.ApprovedBy)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		// No draft exists — create one for the current week (or next week if current is taken)
		weekStart := CurrentWeekStart()
		err = pool.QueryRow(ctx, `
			INSERT INTO purchase_orders (week_start, status)
			VALUES ($1, 'draft')
			ON CONFLICT (week_start) DO NOTHING
			RETURNING id, week_start::text, status, version, created_at
		`, weekStart).Scan(&po.ID, &po.WeekStart, &po.Status, &po.Version, &po.CreatedAt)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// Current week exists but is not draft — try next week
				ws, _ := time.Parse("2006-01-02", weekStart)
				nextWeek := ws.AddDate(0, 0, 7).Format("2006-01-02")
				err = pool.QueryRow(ctx, `
					INSERT INTO purchase_orders (week_start, status)
					VALUES ($1, 'draft')
					ON CONFLICT (week_start) DO NOTHING
					RETURNING id, week_start::text, status, version, created_at
				`, nextWeek).Scan(&po.ID, &po.WeekStart, &po.Status, &po.Version, &po.CreatedAt)
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						// Next week draft already exists — fetch it
						return GetOrdersByStatus(ctx, pool, "draft")
					}
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}
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

// GetOrderLineItems fetches line items for a PO, joined with item, user, and vendor info.
// Returns VendorName via LEFT JOIN on vendors (D-09).
func GetOrderLineItems(ctx context.Context, pool *pgxpool.Pool, poID string) ([]POLineItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			li.id, li.po_id, li.purchase_item_id,
			pi.description AS item_name,
			ig.name AS group_name,
			pi.photo_url, pi.store_location,
			v.name AS vendor_name,
			li.quantity, li.unit,
			li.added_by AS added_by_id,
			COALESCE(NULLIF(u.nickname, ''), u.first_name || ' ' || LEFT(u.last_name, 1) || '.') AS added_by_name,
			li.created_at, li.updated_at
		FROM po_line_items li
		JOIN purchase_items pi ON pi.id = li.purchase_item_id
		LEFT JOIN item_groups ig ON ig.id = pi.group_id
		LEFT JOIN vendors v ON v.id = pi.vendor_id
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
			&li.VendorName,
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

// ErrPOLockedAdminOnly is returned when a non-admin tries to edit a locked PO.
var ErrPOLockedAdminOnly = errors.New("po_locked_admin_only")

// UpsertLineItems replaces the line items on a PO inside a transaction.
// Draft POs: any authenticated user can edit.
// Locked POs: only admin can edit (D-08). Returns ErrPOLockedAdminOnly for non-admin.
// Other statuses: returns ErrPONotDraft.
func UpsertLineItems(ctx context.Context, pool *pgxpool.Pool, poID string, userID string, items []UpsertLineItemInput, userIsAdmin bool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Verify PO status — draft is open to all, locked requires admin (D-08)
	var status string
	if err = tx.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id = $1`, poID).Scan(&status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPONotDraft
		}
		return err
	}
	if status == "locked" && !userIsAdmin {
		return ErrPOLockedAdminOnly
	}
	if status != "draft" && status != "locked" {
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

// GetCutoffConfig returns the single-row cutoff config, or nil if not yet set.
func GetCutoffConfig(ctx context.Context, pool *pgxpool.Pool) (*CutoffConfig, error) {
	var cfg CutoffConfig
	err := pool.QueryRow(ctx, `
		SELECT id::text, day_of_week, cutoff_time::text, timezone, updated_at
		FROM cutoff_config
		LIMIT 1
	`).Scan(&cfg.ID, &cfg.DayOfWeek, &cfg.CutoffTime, &cfg.Timezone, &cfg.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetCutoffConfig: %w", err)
	}
	return &cfg, nil
}

// UpsertCutoffConfig replaces the single-row cutoff configuration.
// Deletes all existing rows and inserts a new one (single-row table pattern).
func UpsertCutoffConfig(ctx context.Context, pool *pgxpool.Pool, dayOfWeek int, cutoffTime string, timezone string) (*CutoffConfig, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("UpsertCutoffConfig: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `DELETE FROM cutoff_config`); err != nil {
		return nil, fmt.Errorf("UpsertCutoffConfig: delete: %w", err)
	}

	var cfg CutoffConfig
	err = tx.QueryRow(ctx, `
		INSERT INTO cutoff_config (day_of_week, cutoff_time, timezone)
		VALUES ($1, $2::time, $3)
		RETURNING id::text, day_of_week, cutoff_time::text, timezone, updated_at
	`, dayOfWeek, cutoffTime, timezone).Scan(&cfg.ID, &cfg.DayOfWeek, &cfg.CutoffTime, &cfg.Timezone, &cfg.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("UpsertCutoffConfig: insert: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("UpsertCutoffConfig: commit: %w", err)
	}
	return &cfg, nil
}

// LockPO transitions a draft PO to locked status and creates next week's draft.
// Uses optimistic locking via WHERE status = 'draft' (Pitfall 2).
func LockPO(ctx context.Context, pool *pgxpool.Pool, poID string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("LockPO: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var weekStart string
	err = tx.QueryRow(ctx, `
		UPDATE purchase_orders
		SET status = 'locked', locked_at = now(), version = version + 1
		WHERE id = $1 AND status = 'draft'
		RETURNING week_start::text
	`, poID).Scan(&weekStart)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPONotDraft
		}
		return fmt.Errorf("LockPO: update: %w", err)
	}

	// Compute next Monday (+7 days from week_start)
	weekStartTime, err := time.Parse("2006-01-02", weekStart)
	if err != nil {
		return fmt.Errorf("LockPO: parse week_start %q: %w", weekStart, err)
	}
	nextWeekStart := weekStartTime.AddDate(0, 0, 7).Format("2006-01-02")

	// Create next week's draft (idempotent — DO NOTHING if already exists)
	if _, err := tx.Exec(ctx, `
		INSERT INTO purchase_orders (week_start, status)
		VALUES ($1, 'draft')
		ON CONFLICT (week_start) DO NOTHING
	`, nextWeekStart); err != nil {
		return fmt.Errorf("LockPO: create next draft: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("LockPO: commit: %w", err)
	}
	return nil
}

// UnlockPO transitions a locked PO back to draft status.
// Returns ErrUnlockAfterApproval if the PO has been approved (D-13).
func UnlockPO(ctx context.Context, pool *pgxpool.Pool, poID string) error {
	// Check current status first — cannot unlock after approval
	var status string
	err := pool.QueryRow(ctx, `SELECT status FROM purchase_orders WHERE id = $1`, poID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPONotLocked
		}
		return fmt.Errorf("UnlockPO: check status: %w", err)
	}
	if status == "approved" || status == "shopping_active" || status == "completed" {
		return ErrUnlockAfterApproval
	}

	tag, err := pool.Exec(ctx, `
		UPDATE purchase_orders
		SET status = 'draft', locked_at = NULL, version = version + 1
		WHERE id = $1 AND status = 'locked'
	`, poID)
	if err != nil {
		return fmt.Errorf("UnlockPO: update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPONotLocked
	}
	return nil
}

// ApprovePO transitions a locked PO to shopping_active and creates an immutable
// shopping list snapshot in a single transaction (D-10, D-11, D-15).
// Returns ErrPONotLocked if the PO is not locked, ErrActiveShoppingListExists if
// another shopping list is already active.
func ApprovePO(ctx context.Context, pool *pgxpool.Pool, poID string, approvedBy string) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("ApprovePO: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Check no active shopping list exists (D-11)
	var activeCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM shopping_lists WHERE status = 'active'`).Scan(&activeCount); err != nil {
		return "", fmt.Errorf("ApprovePO: check active lists: %w", err)
	}
	if activeCount > 0 {
		return "", ErrActiveShoppingListExists
	}

	// Transition PO to shopping_active
	var weekStart string
	err = tx.QueryRow(ctx, `
		UPDATE purchase_orders
		SET status = 'shopping_active', approved_at = now(), approved_by = $2, version = version + 1
		WHERE id = $1 AND status = 'locked'
		RETURNING week_start::text
	`, poID, approvedBy).Scan(&weekStart)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrPONotLocked
		}
		return "", fmt.Errorf("ApprovePO: update po: %w", err)
	}
	_ = weekStart // confirms row found

	// Create shopping list
	var listID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO shopping_lists (po_id) VALUES ($1) RETURNING id::text
	`, poID).Scan(&listID); err != nil {
		return "", fmt.Errorf("ApprovePO: create shopping list: %w", err)
	}

	// Get distinct vendors from PO line items
	type vendorRow struct {
		vid   string
		vname string
	}
	vendorRows, err := tx.Query(ctx, `
		SELECT COALESCE(pi.vendor_id::text, 'unassigned') AS vid, COALESCE(v.name, 'Unassigned') AS vname
		FROM po_line_items pli
		JOIN purchase_items pi ON pi.id = pli.purchase_item_id
		LEFT JOIN vendors v ON v.id = pi.vendor_id
		WHERE pli.po_id = $1
		GROUP BY pi.vendor_id, v.name
	`, poID)
	if err != nil {
		return "", fmt.Errorf("ApprovePO: query vendors: %w", err)
	}

	var vendors []vendorRow
	for vendorRows.Next() {
		var vr vendorRow
		if err := vendorRows.Scan(&vr.vid, &vr.vname); err != nil {
			vendorRows.Close()
			return "", fmt.Errorf("ApprovePO: scan vendor: %w", err)
		}
		vendors = append(vendors, vr)
	}
	if err := vendorRows.Err(); err != nil {
		vendorRows.Close()
		return "", fmt.Errorf("ApprovePO: vendors rows err: %w", err)
	}
	vendorRows.Close()

	// Create vendor sections and snapshot items for each vendor
	for _, vr := range vendors {
		var sectionID string
		var vendorIDParam interface{}
		if vr.vid != "unassigned" {
			vendorIDParam = vr.vid
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO shopping_list_vendor_sections (shopping_list_id, vendor_id, vendor_name)
			VALUES ($1, $2, $3)
			RETURNING id::text
		`, listID, vendorIDParam, vr.vname).Scan(&sectionID); err != nil {
			return "", fmt.Errorf("ApprovePO: create vendor section %q: %w", vr.vname, err)
		}

		// Snapshot items for this vendor
		var itemsQuery string
		var itemsArgs []interface{}
		if vr.vid == "unassigned" {
			itemsQuery = `
				INSERT INTO shopping_list_items (shopping_list_id, vendor_section_id, purchase_item_id, item_name, photo_url, store_location, quantity, unit)
				SELECT $1, $2, pi.id, COALESCE(pi.full_name, pi.description), pi.photo_url, pi.store_location, pli.quantity, pli.unit
				FROM po_line_items pli
				JOIN purchase_items pi ON pi.id = pli.purchase_item_id
				WHERE pli.po_id = $3 AND pi.vendor_id IS NULL
			`
			itemsArgs = []interface{}{listID, sectionID, poID}
		} else {
			itemsQuery = `
				INSERT INTO shopping_list_items (shopping_list_id, vendor_section_id, purchase_item_id, item_name, photo_url, store_location, quantity, unit)
				SELECT $1, $2, pi.id, COALESCE(pi.full_name, pi.description), pi.photo_url, pi.store_location, pli.quantity, pli.unit
				FROM po_line_items pli
				JOIN purchase_items pi ON pi.id = pli.purchase_item_id
				WHERE pli.po_id = $3 AND pi.vendor_id::text = $4
			`
			itemsArgs = []interface{}{listID, sectionID, poID, vr.vid}
		}
		if _, err := tx.Exec(ctx, itemsQuery, itemsArgs...); err != nil {
			return "", fmt.Errorf("ApprovePO: snapshot items for vendor %q: %w", vr.vname, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("ApprovePO: commit: %w", err)
	}
	return listID, nil
}

// GetOrdersByStatus returns the most recent PO with the given status, or nil if none exists.
// Used by the frontend to load locked/approved POs for the PO tab (D-09).
func GetOrdersByStatus(ctx context.Context, pool *pgxpool.Pool, status string) (*PurchaseOrder, error) {
	var po PurchaseOrder
	err := pool.QueryRow(ctx, `
		SELECT id, week_start::text, status, version, created_at,
		       locked_at, approved_at, approved_by
		FROM purchase_orders
		WHERE status = $1
		ORDER BY week_start DESC
		LIMIT 1
	`, status).Scan(
		&po.ID, &po.WeekStart, &po.Status, &po.Version, &po.CreatedAt,
		&po.LockedAt, &po.ApprovedAt, &po.ApprovedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("GetOrdersByStatus: %w", err)
	}

	lineItems, err := GetOrderLineItems(ctx, pool, po.ID)
	if err != nil {
		return nil, fmt.Errorf("GetOrdersByStatus: get line items: %w", err)
	}
	po.LineItems = lineItems
	return &po, nil
}
