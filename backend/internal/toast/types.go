package toast

import (
	"time"

	s3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds everything the background worker / CLI / handler needs to run
// a Toast SFTP ingest cycle. SFTPConfig (in sftp.go) is the lower-level
// SSH-transport config; this Config wraps it with scheduling and DB state.
type Config struct {
	SFTPHost       string        // e.g. s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22
	SFTPUser       string        // YumYumsExportUser
	SFTPKeyPath    string        // filesystem path to private key (REQUIRED, fail-fast at LoadConfigFromEnv)
	ExportID       string        // 113866 — path prefix on the SFTP server
	Pool           *pgxpool.Pool // injected by caller (server main / sync-toast main)
	Interval       time.Duration // 12h default; 0 disables the worker
	SyncWindowDays int           // 7 — re-pull last N days per tick
	BackfillDays   int           // 90 — used on cold-start only

	// Phase 22.1 additions — DO Spaces is the durable archive (D-01).
	// SpacesClient is injected by the caller (cmd/server/main.go around line 532).
	// If nil at StartWorker time, the worker logs WARNING and returns gracefully
	// (D-06 deviation from Phase 22 D-12: Spaces is broader than Toast — server keeps running).
	SpacesClient   *s3.Client
	SpacesBucket   string // e.g. "hq.yumyums"
	SpacesEndpoint string // e.g. "https://nyc3.digitaloceanspaces.com"
	CacheDir       string // e.g. "backend/cache/toast" (D-12)

	// Dialer opens an authenticated SFTP client. Nil means "use the real
	// dialWithRetry" (the production path). Tests inject a failing/fake dialer
	// so SyncDate's dial/auth-failure classification (B-146 fail-loud) can be
	// exercised without a live SFTP endpoint.
	Dialer func(cfg Config, pemKey string) (*Client, error)
}

// AggregatedRow is the parser's output: one row per (master_id, business_date)
// with units/gross already summed across the day's CSV rows (voided lines
// excluded entirely per D-06).
type AggregatedRow struct {
	MasterID     string
	Name         string
	Menu         string
	MenuGroup    string
	MenuSubgroup *string // nil when the CSV column was empty/whitespace
	BusinessDate string  // YYYY-MM-DD
	UnitsSold    int
	GrossAmount  float64
}

// MenuItem mirrors a row in menu_items, used by the GET /menu-items handler.
type MenuItem struct {
	ID           string    `json:"id"`
	MasterID     string    `json:"master_id"`
	Name         string    `json:"name"`
	Menu         string    `json:"menu"`
	MenuGroup    string    `json:"menu_group"`
	MenuSubgroup *string   `json:"menu_subgroup,omitempty"`
	LastSeen     string    `json:"last_seen"` // YYYY-MM-DD
	CreatedAt    time.Time `json:"created_at"`
}

// MenuItemWithSales decorates MenuItem with this-week aggregate stats, served
// by GET /api/v1/inventory/menu-items?since=YYYY-MM-DD.
type MenuItemWithSales struct {
	MenuItem
	UnitsSoldThisWeek int     `json:"units_sold_this_week"`
	GrossThisWeek     float64 `json:"gross_this_week"`
}

// IngestResult is the per-cycle summary returned by RunIngest. Plan 04's
// per-cycle log line (D-13) reads from this struct.
type IngestResult struct {
	Dates             []string // YYYYMMDD strings, oldest first
	ItemsUpserted     int
	SalesRowsUpserted int
	Duration          time.Duration
}
