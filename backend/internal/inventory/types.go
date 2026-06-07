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
	ID            string `json:"id"`
	Name          string `json:"name"`
	ParDays       *int   `json:"par_days,omitempty"`
	LowThreshold  int    `json:"low_threshold"`
	HighThreshold int    `json:"high_threshold"`
	Tags          []Tag  `json:"tags,omitempty"`
}

// PurchaseItem is a canonical product that appears on purchase line items.
type PurchaseItem struct {
	ID            string  `json:"id"`
	Description   string  `json:"description"`
	GroupID       *string `json:"group_id,omitempty"`
	GroupName     *string `json:"group_name,omitempty"`
	StoreLocation   *string `json:"store_location,omitempty"`
	LocationInStore *string `json:"location_in_store,omitempty"`
	PhotoURL        *string `json:"photo_url,omitempty"`
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
	ParseError  *string          `json:"parse_error,omitempty"`
	Items       json.RawMessage  `json:"items"`
	ConfirmedAt *time.Time       `json:"confirmed_at,omitempty"`
	ConfirmedBy *string          `json:"confirmed_by,omitempty"`
	DiscardedAt *time.Time       `json:"discarded_at,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

// RepurchaseBadge holds repurchase tracking data for a stock item (REP-01).
// Present when the item was purchased via a completed shopping list since the last badge reset.
type RepurchaseBadge struct {
	Qty           int       `json:"qty"`
	RepurchasedAt time.Time `json:"repurchased_at"`
}

// StockItem is an aggregated stock level for one purchase item description.
type StockItem struct {
	Description      string           `json:"description"`
	GroupName        *string          `json:"group_name,omitempty"`
	TotalQuantity    int              `json:"total_quantity"`
	TotalSpend       float64          `json:"total_spend"`
	AvgPrice         float64          `json:"avg_price"`
	LastPurchaseDate string           `json:"last_purchase_date"` // YYYY-MM-DD
	LowThreshold     int              `json:"low_threshold"`
	HighThreshold    int              `json:"high_threshold"`
	Level            string           `json:"level"`
	NeedsReorder     bool             `json:"needs_reorder"`
	RepurchaseBadge  *RepurchaseBadge `json:"repurchase_badge,omitempty"`
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

// PeriodSummary is the response body for GET /api/v1/inventory/period-summary.
// COGS aggregates use purchase_events.event_date (DATE — no TZ).
// Completeness gate uses pending_purchases.created_at cast to America/Chicago calendar date.
type PeriodSummary struct {
	From               string            `json:"from"`                 // YYYY-MM-DD
	To                 string            `json:"to"`                   // YYYY-MM-DD
	COGSExclTax        float64           `json:"cogs_excl_tax"`
	COGSInclTax        float64           `json:"cogs_incl_tax"`
	PurchaseEventCount int               `json:"purchase_event_count"`
	ByVendor           []VendorCOGS      `json:"by_vendor"`
	// TrackedBankTxIDs is every Mercury bank_tx_id HQ has touched for
	// the period, across all states (confirmed in purchase_events,
	// pending/confirmed/discarded in pending_purchases). Consumers diff
	// this against Mercury's own transaction list for the same period
	// to detect "Mercury has it, HQ hasn't ingested it yet" gaps. See
	// sales-processor/docs/payroll-mercury-gap-check.md.
	TrackedBankTxIDs []string          `json:"tracked_bank_tx_ids"`
	Completeness     CompletenessBlock `json:"completeness"`
}

// VendorCOGS is one row of the per-vendor breakdown returned by
// /period-summary. trip_count counts distinct purchase_events for the vendor
// in the period; tax is allocated per event (not per line item).
type VendorCOGS struct {
	VendorID     string  `json:"vendor_id"`
	VendorName   string  `json:"vendor_name"`
	TotalExclTax float64 `json:"total_excl_tax"`
	TotalInclTax float64 `json:"total_incl_tax"`
	TripCount    int     `json:"trip_count"`
}

// CompletenessBlock reports whether HQ receipts for the period are fully
// ingested + reviewed + catalog-linked. `ready` is true iff both ID lists are empty.
// PendingReviewIDs lists pending_purchases.id rows where confirmed_at IS NULL
// AND discarded_at IS NULL within the period.
// UnlinkedLineItemIDs lists purchase_line_items.id rows where purchase_item_id
// IS NULL for purchase_events in the period.
type CompletenessBlock struct {
	Ready                bool                  `json:"ready"`
	PendingReviewIDs     []string              `json:"pending_review_ids"`
	PendingReviewDetails []PendingReviewDetail `json:"pending_review_details"`
	UnlinkedLineItemIDs  []string              `json:"unlinked_line_item_ids"`
}

// PendingReviewDetail is one row of operator-facing context per pending
// review. Exposed on /period-summary so service-token callers
// (sales-processor) can render a meaningful failure message without a
// second round trip to the cookie-auth-only /purchases/pending
// endpoint.
type PendingReviewDetail struct {
	ID        string  `json:"id"`
	BankTxID  string  `json:"bank_tx_id"`
	Vendor    string  `json:"vendor"`     // "" when receipt parser couldn't extract one
	EventDate string  `json:"event_date"` // YYYY-MM-DD; falls back to created_at::date
	BankTotal float64 `json:"bank_total"`
	Reason    *string `json:"reason,omitempty"`
}
