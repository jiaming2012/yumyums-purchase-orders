package receipt

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MercuryTransaction represents a single transaction from the Mercury banking API.
type MercuryTransaction struct {
	ID              string               `json:"id"`
	Amount          float64              `json:"amount"`
	BankDescription string               `json:"bankDescription"`
	Status          string               `json:"status"`
	Kind            string               `json:"kind"`
	Attachments     []Attachment         `json:"attachments"`
	Note            string               `json:"note"`
	CreatedAt       string               `json:"createdAt"`
	CategoryData    *MercuryCategoryData `json:"categoryData"` // nullable
}

// MercuryCategoryData mirrors Mercury's per-transaction categoryData
// field. Set by the sales-processor classify pipeline (Claude) via
// PATCH /transaction/{id}. Null until classified.
type MercuryCategoryData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Attachment is a file attachment on a Mercury transaction (e.g. a receipt scan).
type Attachment struct {
	URL      string `json:"url"`
	FileName string `json:"fileName"`
}

// ReceiptItem is a single line item parsed from a receipt.
//
// Quantity is float64 so JSON values like `40.0` (which Anthropic returns for
// some receipts — e.g. Restaurant Depot, 260607-k1n) unmarshal cleanly instead
// of failing the strict int decoder. The DB column purchase_line_items.quantity
// is INTEGER — the int coercion happens at the single DB-write boundary in
// worker.go (createPurchaseEvent) via math.Round.
type ReceiptItem struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	IsCase   bool    `json:"is_case"`
}

// ReceiptSummary is the summary block parsed from a receipt.
type ReceiptSummary struct {
	Vendor     string  `json:"vendor"`
	TotalUnits int     `json:"total_units"`
	TotalCases int     `json:"total_cases"`
	Tax        float64 `json:"tax"`
	Total      float64 `json:"total"`
}

// ValidationResult carries the outcome of ValidateReceiptData.
type ValidationResult struct {
	Valid  bool
	Reason string
}

// WorkerConfig holds everything the background worker needs.
type WorkerConfig struct {
	MercuryAPIKey    string
	AnthropicAPIKey  string
	Pool             *pgxpool.Pool
	SpacesPresigner  *s3.PresignClient // presign client (optional)
	SpacesEndpoint   string            // endpoint base URL for public URLs
	SpacesBucket     string
	Interval         time.Duration
	LookbackDays     int
}

// mercuryListTransactionsResponse is the envelope returned by the Mercury
// GET /transactions endpoint.
type mercuryListTransactionsResponse struct {
	Transactions []MercuryTransaction `json:"transactions"`
	Total        int                  `json:"total"`
}
