package receipt

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MercuryTransaction represents a single transaction from the Mercury banking API.
type MercuryTransaction struct {
	ID              string        `json:"id"`
	Amount          float64       `json:"amount"`
	BankDescription string        `json:"bankDescription"`
	Status          string        `json:"status"`
	Kind            string        `json:"kind"`
	Attachments     []Attachment  `json:"attachments"`
	Note            string        `json:"note"`
	CreatedAt       string        `json:"createdAt"`
}

// Attachment is a file attachment on a Mercury transaction (e.g. a receipt scan).
type Attachment struct {
	URL      string `json:"url"`
	FileName string `json:"fileName"`
}

// ReceiptItem is a single line item parsed from a receipt.
type ReceiptItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
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
