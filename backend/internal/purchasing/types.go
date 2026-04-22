package purchasing

import "time"

// PurchaseOrder is one week's draft/locked/approved order.
type PurchaseOrder struct {
	ID        string       `json:"id"`
	WeekStart string       `json:"week_start"` // YYYY-MM-DD (Monday)
	Status    string       `json:"status"`      // draft | locked | approved
	Version   int          `json:"version"`
	CreatedAt time.Time    `json:"created_at"`
	LineItems []POLineItem `json:"line_items"`
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
