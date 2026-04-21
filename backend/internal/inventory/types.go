package inventory

import (
	"encoding/json"
	"time"
)

// Vendor is a food supplier.
type Vendor struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Tag is a label applied to item groups.
type Tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ItemGroup groups related purchase items (e.g. "Proteins", "Produce").
type ItemGroup struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	ParDays *int   `json:"par_days,omitempty"`
	Tags    []Tag  `json:"tags,omitempty"`
}

// PurchaseItem is a canonical product that appears on purchase line items.
type PurchaseItem struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	GroupID     *string `json:"group_id,omitempty"`
	GroupName   *string `json:"group_name,omitempty"`
}

// ItemGroupWithItems is an item group with its items included.
type ItemGroupWithItems struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	ParDays *int           `json:"par_days,omitempty"`
	Tags    []Tag          `json:"tags,omitempty"`
	Items   []PurchaseItem `json:"items"`
}

// LineItem is one line on a purchase event.
type LineItem struct {
	ID              string  `json:"id"`
	PurchaseEventID string  `json:"purchase_event_id"`
	PurchaseItemID  *string `json:"purchase_item_id,omitempty"`
	Description     string  `json:"description"`
	Quantity        int     `json:"quantity"`
	Price           float64 `json:"price"`
	IsCase          bool    `json:"is_case"`
}

// PurchaseEvent is a single vendor purchase (one receipt).
type PurchaseEvent struct {
	ID         string     `json:"id"`
	VendorID   string     `json:"vendor_id"`
	VendorName string     `json:"vendor_name"`
	BankTxID   string     `json:"bank_tx_id"`
	EventDate  string     `json:"event_date"` // YYYY-MM-DD
	Tax        float64    `json:"tax"`
	Total      float64    `json:"total"`
	ReceiptURL *string    `json:"receipt_url,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	LineItems  []LineItem `json:"line_items,omitempty"`
}

// PendingPurchase is a receipt awaiting review before becoming a real purchase event.
type PendingPurchase struct {
	ID          string           `json:"id"`
	BankTxID    string           `json:"bank_tx_id"`
	BankTotal   float64          `json:"bank_total"`
	Vendor      string           `json:"vendor"`
	EventDate   *string          `json:"event_date,omitempty"`
	Tax         *float64         `json:"tax,omitempty"`
	Total       *float64         `json:"total,omitempty"`
	TotalUnits  *int             `json:"total_units,omitempty"`
	TotalCases  *int             `json:"total_cases,omitempty"`
	ReceiptURL  *string          `json:"receipt_url,omitempty"`
	Reason      *string          `json:"reason,omitempty"`
	Items       json.RawMessage  `json:"items"`
	ConfirmedAt *time.Time       `json:"confirmed_at,omitempty"`
	ConfirmedBy *string          `json:"confirmed_by,omitempty"`
	DiscardedAt *time.Time       `json:"discarded_at,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

// StockItem is an aggregated stock level for one purchase item description.
type StockItem struct {
	Description     string   `json:"description"`
	GroupName       *string  `json:"group_name,omitempty"`
	TotalQuantity   int      `json:"total_quantity"`
	TotalSpend      float64  `json:"total_spend"`
	AvgPrice        float64  `json:"avg_price"`
	LastPurchaseDate string  `json:"last_purchase_date"` // YYYY-MM-DD
}

// CreateLineItemInput is one line item in a CreatePurchaseEventInput.
type CreateLineItemInput struct {
	PurchaseItemID *string `json:"purchase_item_id,omitempty"`
	Description    string  `json:"description"`
	Quantity       int     `json:"quantity"`
	Price          float64 `json:"price"`
	IsCase         bool    `json:"is_case"`
}

// CreatePurchaseEventInput is the body for POST /api/v1/inventory/purchases.
type CreatePurchaseEventInput struct {
	VendorID   string               `json:"vendor_id"`
	BankTxID   string               `json:"bank_tx_id"`
	EventDate  string               `json:"event_date"` // YYYY-MM-DD
	Tax        float64              `json:"tax"`
	Total      float64              `json:"total"`
	ReceiptURL *string              `json:"receipt_url,omitempty"`
	LineItems  []CreateLineItemInput `json:"line_items"`
}

// ConfirmPendingInput is the body for POST /api/v1/inventory/purchases/confirm.
type ConfirmPendingInput struct {
	ID         string               `json:"id"`
	VendorName string               `json:"vendor_name"`
	EventDate  string               `json:"event_date"`
	Tax        float64              `json:"tax"`
	Total      float64              `json:"total"`
	LineItems  []CreateLineItemInput `json:"line_items"`
}

// DiscardPendingInput is the body for POST /api/v1/inventory/purchases/discard.
type DiscardPendingInput struct {
	ID string `json:"id"`
}
