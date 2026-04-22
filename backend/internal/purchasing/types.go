package purchasing

import "time"

// PurchaseOrder is one week's draft/locked/approved order.
type PurchaseOrder struct {
	ID         string       `json:"id"`
	WeekStart  string       `json:"week_start"`  // YYYY-MM-DD (Monday)
	Status     string       `json:"status"`       // draft | locked | approved | shopping_active | completed
	Version    int          `json:"version"`
	CreatedAt  time.Time    `json:"created_at"`
	LockedAt   *time.Time   `json:"locked_at,omitempty"`
	ApprovedAt *time.Time   `json:"approved_at,omitempty"`
	ApprovedBy *string      `json:"approved_by,omitempty"`
	LineItems  []POLineItem `json:"line_items"`
}

// POLineItem is one item on a purchase order.
type POLineItem struct {
	ID             string    `json:"id"`
	POID           string    `json:"po_id"`
	PurchaseItemID string    `json:"purchase_item_id"`
	ItemName       string    `json:"item_name"`
	GroupName      *string   `json:"group_name,omitempty"`
	PhotoURL       *string   `json:"photo_url,omitempty"`
	StoreLocation  *string   `json:"store_location,omitempty"`
	VendorName     *string   `json:"vendor_name,omitempty"` // populated via LEFT JOIN for vendor grouping (D-09)
	Quantity       int       `json:"quantity"`
	Unit           string    `json:"unit"`
	AddedByID      string    `json:"added_by_id"`
	AddedByName    string    `json:"added_by_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// OrderSuggestion is an item below its group's restock threshold.
type OrderSuggestion struct {
	PurchaseItemID string  `json:"purchase_item_id"`
	ItemName       string  `json:"item_name"`
	PhotoURL       *string `json:"photo_url,omitempty"`
	StoreLocation  *string `json:"store_location,omitempty"`
	GroupName      *string `json:"group_name,omitempty"`
	LowThreshold   int     `json:"low_threshold"`
	CurrentStock   int     `json:"current_stock"`
	SuggestedQty   int     `json:"suggested_qty"`
	Unit           string  `json:"unit"`
	StockLevel     string  `json:"stock_level"`
}

// UpsertLineItemInput is one item in the PUT /orders/:id/items body.
type UpsertLineItemInput struct {
	PurchaseItemID string `json:"purchase_item_id"`
	Quantity       int    `json:"quantity"`
	Unit           string `json:"unit"`
}

// UpsertLineItemsRequest is the body for PUT /orders/:id/items.
type UpsertLineItemsRequest struct {
	Items []UpsertLineItemInput `json:"items"`
}

// CutoffConfig stores the weekly cutoff schedule (single row upserted, per D-03).
type CutoffConfig struct {
	ID         string    `json:"id"`
	DayOfWeek  int       `json:"day_of_week"` // 0=Sunday, 6=Saturday
	CutoffTime string    `json:"cutoff_time"` // HH:MM format
	Timezone   string    `json:"timezone"`    // e.g. "America/Chicago"
	UpdatedAt  time.Time `json:"updated_at"`
}

// ShoppingList is an immutable snapshot created at PO approval time (per D-15).
type ShoppingList struct {
	ID             string                      `json:"id"`
	POID           string                      `json:"po_id"`
	WeekStart      string                      `json:"week_start"`
	Status         string                      `json:"status"`                    // active | completed
	AssignedTo     *string                     `json:"assigned_to,omitempty"`
	AssignedRole   *string                     `json:"assigned_role,omitempty"`
	CreatedAt      time.Time                   `json:"created_at"`
	CompletedAt    *time.Time                  `json:"completed_at,omitempty"`
	VendorSections []ShoppingListVendorSection `json:"vendor_sections"`
}

// ShoppingListVendorSection groups items by vendor within a shopping list (per D-21).
type ShoppingListVendorSection struct {
	ID              string             `json:"id"`
	ShoppingListID  string             `json:"shopping_list_id"`
	VendorID        *string            `json:"vendor_id,omitempty"`
	VendorName      string             `json:"vendor_name"`
	Status          string             `json:"status"`                    // pending | completed
	CompletedBy     *string            `json:"completed_by,omitempty"`
	CompletedByName *string            `json:"completed_by_name,omitempty"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	Items           []ShoppingListItem `json:"items"`
}

// ShoppingListItem is one item on the shopping checklist (snapshotted from PO line item).
type ShoppingListItem struct {
	ID              string     `json:"id"`
	ShoppingListID  string     `json:"shopping_list_id"`
	VendorSectionID string     `json:"vendor_section_id"`
	PurchaseItemID  string     `json:"purchase_item_id"`
	ItemName        string     `json:"item_name"`
	PhotoURL        *string    `json:"photo_url,omitempty"`
	StoreLocation   *string    `json:"store_location,omitempty"`
	Quantity        int        `json:"quantity"`
	Unit            string     `json:"unit"`
	Checked         bool       `json:"checked"`
	CheckedBy       *string    `json:"checked_by,omitempty"`
	CheckedByName   *string    `json:"checked_by_name,omitempty"`
	CheckedAt       *time.Time `json:"checked_at,omitempty"`
}
