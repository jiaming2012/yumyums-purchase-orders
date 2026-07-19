# Phase 22: HQ Toast ingest — Pattern Map

**Mapped:** 2026-06-03
**Files analyzed:** 11 new + 2 modified
**Analogs found:** 13 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `backend/internal/toast/sftp.go` | client (SFTP) | streaming/file-I/O | `sales-processor/sftp/default.go` | **port verbatim** |
| `backend/internal/toast/types.go` | types | n/a | `backend/internal/receipt/types.go` | exact |
| `backend/internal/toast/parser.go` | parser | transform (CSV → structs) | `backend/cmd/import-notion/main.go` | role-match |
| `backend/internal/toast/ingest.go` | service (per-cycle ingest) | batch + DB upsert | `backend/internal/receipt/worker.go` runIngestCycle | role-match |
| `backend/internal/toast/worker.go` | worker (factory + goroutine) | event-driven (ticker) | `backend/internal/receipt/worker.go` StartWorker + `backend/internal/purchasing/scheduler.go` | exact — DEVIATES on fail-fast |
| `backend/internal/toast/handler.go` | HTTP handler | request-response | `backend/internal/inventory/handler.go` ListItemsHandler | exact |
| `backend/internal/inventory/types.go` (modify) | types | n/a | itself (existing PurchaseItem/StockItem patterns) | self |
| `backend/cmd/sync-toast/main.go` | CLI binary (one-shot) | batch | `backend/cmd/seed/main.go` + `backend/cmd/import-notion/main.go` | role-match |
| `backend/internal/db/migrations/0060_menu_items.sql` | migration | schema | `backend/internal/db/migrations/0024_inventory.sql` | exact |
| `backend/internal/db/migrations/0061_daily_menu_sales.sql` | migration | schema | `backend/internal/db/migrations/0044_low_stock_alert_log.sql` | exact (composite PK pattern) |
| `backend/cmd/server/main.go` (modify) | wiring | n/a | itself lines 482-514 (receipt block) + 522-523 (scheduler) | self |
| `inventory.html` (modify) | frontend tab | request-response | itself (existing 5-tab pattern, lines 183-189, 271-284) | self |
| `backend/go.mod` (modify) | config | n/a | itself | self |

**Naming note on package:** `backend/internal/toast/` — sibling of `receipt/` and `purchasing/`. The internal SFTP file is `sftp.go`, not its own sub-package, to keep the import path flat (`github.com/yumyums/hq/internal/toast`) and avoid colliding with the upstream `github.com/pkg/sftp` import.

---

## Pattern Assignments

### `backend/internal/toast/sftp.go` (client, streaming/file-I/O)

**Analog:** `/Users/jamal/projects/yumyums/sales-processor/sftp/default.go` — **port verbatim**

**Action:** Copy the entire 170-line file into `backend/internal/toast/sftp.go`. Change only the `package` declaration from `package sftp` to `package toast`. The struct, methods, and connect logic are correct as-is.

**Surface to preserve (used by Phase 22 ingest):**

```go
// Config represents SSH connection parameters.
type Config struct {
    Username     string
    Password     string
    PrivateKey   string
    Server       string
    KeyExchanges []string
    Timeout      time.Duration
}

func New(config Config) (*Client, error)
func (c *Client) Download(filePath string) (io.ReadCloser, error)
func (c *Client) Close()
```

**Naming collision risk:** Sales-processor names this struct `sftp.Config` and the package itself `sftp`. After porting into `package toast`, this becomes `toast.Config` — which collides conceptually with the worker config. **Rename on port:** call the SFTP struct `SFTPConfig` and the worker struct `Config`. The connect/Download/Close methods remain on `*Client`. This deviation is intentional and small; document it at the top of `sftp.go`.

**HostKeyCallback is `InsecureIgnoreHostKey`-equivalent** (returns nil for any host key). Keep as-is — Toast's SFTP host doesn't publish a known host key the same way an SSH server does, and the upstream sales-processor has run in prod this way for years.

---

### `backend/internal/toast/types.go` (types)

**Analog:** `backend/internal/receipt/types.go` lines 51-61

**Imports + struct convention pattern:**

```go
package toast

import (
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// Config holds everything the background worker needs.
type Config struct {
    SFTPHost       string        // e.g. s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22
    SFTPUser       string        // YumYumsExportUser
    SFTPKeyPath    string        // filesystem path to private key (REQUIRED, fail-fast)
    ExportID       string        // 113866
    Pool           *pgxpool.Pool
    Interval       time.Duration // 12h
    SyncWindowDays int           // 7 — re-pull last N days per tick
    BackfillDays   int           // 90 — used on cold-start only
}

// MenuItem mirrors a row in menu_items.
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

// MenuItemWithSales is the API row served by GET /menu-items?since=.
type MenuItemWithSales struct {
    MenuItem
    UnitsSoldThisWeek int     `json:"units_sold_this_week"`
    GrossThisWeek     float64 `json:"gross_this_week"`
}
```

**JSON-tag rule (mirror inventory/types.go lines 22-29):** lowercase snake_case; nullable fields use `*T` with `omitempty`.

---

### `backend/internal/toast/parser.go` (parser, CSV → structs)

**Analog:** `backend/cmd/import-notion/main.go` lines 138-185

**Core CSV reader pattern to copy:**

```go
import "encoding/csv"

func parseItemSelectionDetails(r io.Reader, businessDate string) ([]AggregatedRow, error) {
    rdr := csv.NewReader(r)
    headers, err := rdr.Read()
    if err != nil {
        return nil, fmt.Errorf("read header: %w", err)
    }

    // Build column index map (strip UTF-8 BOM from first header if present)
    colIdx := map[string]int{}
    for i, h := range headers {
        h = strings.TrimSpace(h)
        if i == 0 {
            h = strings.TrimPrefix(h, "\xef\xbb\xbf")
        }
        colIdx[h] = i
    }

    // Verify required columns exist (FAIL on missing — Toast's schema is stable)
    for _, col := range []string{"Master Id", "Menu Item", "Menu", "Menu Group", "Menu Subgroup(s)", "Qty", "Gross Price", "Void?"} {
        if _, ok := colIdx[col]; !ok {
            return nil, fmt.Errorf("CSV missing required column %q. Found: %v", col, headers)
        }
    }

    // Per-row read loop with EOF guard
    for {
        row, err := rdr.Read()
        if err == io.EOF { break }
        if err != nil { return nil, fmt.Errorf("read row: %w", err) }

        get := func(col string) string {
            idx, ok := colIdx[col]
            if !ok || idx >= len(row) { return "" }
            return strings.TrimSpace(row[idx])
        }
        // ... use get("Master Id"), get("Qty"), etc.
    }
}
```

**Phase-22-specific transforms:**
- `Void?` parsing: lowercase compare against `"true"` / `"1"` / `"yes"` — Toast's CSVs use plain `true`/`false`. Voided rows are **excluded entirely** from both `units_sold` and `gross_amount` (D-06).
- `Qty` → `int` via `strconv.Atoi`; tolerate decimals like `1.0` by `strings.Split(s, ".")[0]` or by parsing as float and casting.
- `Gross Price` → `float64` via `strconv.ParseFloat`. Round to 2dp at SQL boundary via `NUMERIC(10,2)` cast in upsert.
- Aggregation: group rows by `Master Id` for the day, sum non-voided `Qty` into `units_sold`, sum non-voided `Gross Price` into `gross_amount`. One output row per (master_id, business_date).
- `Menu Subgroup(s)` → `*string`: empty string becomes `nil` (mirrors `nullableString` in `receipt/worker.go` lines 359-364).

---

### `backend/internal/toast/ingest.go` (service, per-cycle ingest)

**Analog:** `backend/internal/receipt/worker.go` lines 57-181 (runIngestCycle) + lines 217-292 (createPurchaseEvent transaction pattern)

**Imports pattern (lines 1-16 of worker.go):**

```go
package toast

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)
```

**Per-cycle entry point — exported so both StartWorker and cmd/sync-toast call it:**

```go
// RunIngest executes one Toast ingest cycle over [fromDate, toDate] inclusive.
// Returns counts for the per-cycle log line (D-13).
type IngestResult struct {
    Dates             []string // YYYYMMDD
    ItemsUpserted     int
    SalesRowsUpserted int
    Duration          time.Duration
}

func RunIngest(ctx context.Context, pool *pgxpool.Pool, cfg Config, fromDate, toDate time.Time) (*IngestResult, error)
```

**Date-range loop pattern (port from sales-processor/main.go lines 210-263):**

```go
// Iterate YYYYMMDD directories on the SFTP server, oldest first.
client, err := New(SFTPConfig{
    Username:   cfg.SFTPUser,
    PrivateKey: string(pkBytes),
    Server:     cfg.SFTPHost,
    Timeout:    30 * time.Second, // mirrors sales-processor
})
if err != nil {
    return nil, fmt.Errorf("sftp dial: %w", err)
}
defer client.Close()

for d := fromDate; !d.After(toDate); d = d.AddDate(0, 0, 1) {
    dateDir := d.Format("20060102")
    remotePath := fmt.Sprintf("/%s/%s/ItemSelectionDetails.csv", cfg.ExportID, dateDir)
    f, err := client.Download(remotePath)
    if err != nil {
        // Toast directories may be missing for closed days — log + continue.
        log.Printf("toast ingest: skip %s (download: %v)", dateDir, err)
        continue
    }
    rows, parseErr := parseItemSelectionDetails(f, d.Format("2006-01-02"))
    f.Close()
    if parseErr != nil {
        log.Printf("toast ingest: skip %s (parse: %v)", dateDir, parseErr)
        continue
    }
    if err := upsertDayInTx(ctx, pool, rows); err != nil {
        log.Printf("toast ingest: skip %s (db: %v)", dateDir, err)
        continue
    }
}
```

**DB transaction pattern (mirror receipt/worker.go lines 219-291):**

```go
func upsertDayInTx(ctx context.Context, pool *pgxpool.Pool, rows []AggregatedRow) error {
    dbTx, err := pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin: %w", err)
    }
    defer dbTx.Rollback(ctx) //nolint:errcheck

    for _, r := range rows {
        // 1. Upsert menu_items (by master_id), bump last_seen if r.UnitsSold > 0.
        var menuItemID string
        err = dbTx.QueryRow(ctx, `
            INSERT INTO menu_items (master_id, name, menu, menu_group, menu_subgroup, last_seen)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (master_id) DO UPDATE SET
                name          = EXCLUDED.name,
                menu          = EXCLUDED.menu,
                menu_group    = EXCLUDED.menu_group,
                menu_subgroup = EXCLUDED.menu_subgroup,
                last_seen     = GREATEST(menu_items.last_seen, EXCLUDED.last_seen)
            RETURNING id`,
            r.MasterID, r.Name, r.Menu, r.MenuGroup, r.MenuSubgroup, r.BusinessDate,
        ).Scan(&menuItemID)
        if err != nil {
            return fmt.Errorf("upsert menu_item %q: %w", r.MasterID, err)
        }

        // 2. Upsert daily_menu_sales (composite PK = menu_item_id, business_date).
        //    Last-pull wins (D-05).
        _, err = dbTx.Exec(ctx, `
            INSERT INTO daily_menu_sales (menu_item_id, business_date, units_sold, gross_amount, updated_at)
            VALUES ($1, $2, $3, $4, now())
            ON CONFLICT (menu_item_id, business_date) DO UPDATE SET
                units_sold   = EXCLUDED.units_sold,
                gross_amount = EXCLUDED.gross_amount,
                updated_at   = now()`,
            menuItemID, r.BusinessDate, r.UnitsSold, r.GrossAmount,
        )
        if err != nil {
            return fmt.Errorf("upsert daily_menu_sales: %w", err)
        }
    }

    return dbTx.Commit(ctx)
}
```

**Cold-start detection (D-02):**

```go
func isColdStart(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
    var n int
    err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM daily_menu_sales LIMIT 1`).Scan(&n)
    if err != nil {
        return false, fmt.Errorf("isColdStart: %w", err)
    }
    return n == 0, nil
}
```

**SFTP retry pattern (D-10) — wrap the `New()` call:**

```go
func dialWithRetry(cfg Config, pkBytes []byte) (*Client, error) {
    backoffs := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Second}
    var lastErr error
    for i, wait := range backoffs {
        client, err := New(SFTPConfig{ /* ... */ })
        if err == nil {
            return client, nil
        }
        lastErr = err
        if i < len(backoffs)-1 {
            log.Printf("toast ingest: SFTP dial attempt %d failed: %v — retrying in %s", i+1, err, wait)
            time.Sleep(wait)
        }
    }
    return nil, fmt.Errorf("sftp dial failed after 3 attempts: %w", lastErr)
}
```

**Per-cycle log line (D-13):**

```go
log.Printf("toast ingest: dates=[%s..%s] items_upserted=%d sales_rows_upserted=%d duration=%s",
    fromDate.Format("20060102"), toDate.Format("20060102"),
    result.ItemsUpserted, result.SalesRowsUpserted, result.Duration)
```

---

### `backend/internal/toast/worker.go` (worker factory + goroutine)

**Analog:** `backend/internal/receipt/worker.go` lines 21-55 (StartWorker) — **DEVIATES** on missing-key handling per D-12 (fail-fast vs graceful skip).

**Factory pattern to mirror:**

```go
// StartWorker launches a background goroutine that runs Toast ingest on
// `cfg.Interval` (default 12h). DEVIATION from receipt.StartWorker:
// missing TOAST_SFTP_KEY_PATH or unreadable key file is fail-fast — the
// caller (main.go) should validate before calling StartWorker; this function
// expects cfg to be already-validated.
func StartWorker(ctx context.Context, cfg Config) {
    interval := cfg.Interval
    if interval <= 0 {
        interval = 12 * time.Hour
    }

    log.Printf("toast worker: starting (interval=%s, window=%dd, backfill=%dd)",
        interval, cfg.SyncWindowDays, cfg.BackfillDays)

    go func() {
        // Run immediately on start (mirrors receipt.StartWorker + purchasing.StartScheduler).
        runCycle(ctx, cfg)

        ticker := time.NewTicker(interval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                log.Println("toast worker: shutting down")
                return
            case <-ticker.C:
                runCycle(ctx, cfg)
            }
        }
    }()
}

func runCycle(ctx context.Context, cfg Config) {
    cold, err := isColdStart(ctx, cfg.Pool)
    if err != nil {
        log.Printf("toast worker: cold-start check failed: %v", err)
        return
    }

    windowDays := cfg.SyncWindowDays
    if cold {
        windowDays = cfg.BackfillDays
        log.Printf("toast worker: cold start detected — pulling last %d days", windowDays)
    }

    toDate := time.Now()
    fromDate := toDate.AddDate(0, 0, -windowDays)

    if _, err := RunIngest(ctx, cfg.Pool, cfg, fromDate, toDate); err != nil {
        log.Printf("toast worker: ingest cycle error: %v", err)
    }
}
```

**DEVIATION call-out for planner:** Unlike `receipt.StartWorker` (which guards `if cfg.MercuryAPIKey == "" { log + return }`), the Toast worker does **NOT** self-skip. Key validation lives in `cmd/server/main.go` so a misconfigured server fails to start. Document this at the top of `worker.go` with a comment pointing to D-12.

---

### `backend/internal/toast/handler.go` (HTTP handler)

**Analog:** `backend/internal/inventory/handler.go` lines 35-58 (ListVendorsHandler) + lines 1064-1080 (PeriodSummaryHandler — for `?since=` query param parsing)

**Imports + helper convention:**

```go
package toast

import (
    "log"
    "net/http"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

// writeJSON / writeError — duplicate the inventory/handler.go helpers in this
// package OR (cleaner) import from a future internal/httpx package.
// For Phase 22, duplicate locally — matches the existing per-package convention
// (inventory/handler.go also defines its own writeJSON/writeError at lines 24-32).
```

**Handler pattern (mirror ListVendorsHandler closely):**

```go
// ListMenuItemsHandler returns menu_items joined with this-week aggregate
// (units_sold + gross from daily_menu_sales), filtered by ?since=YYYY-MM-DD,
// ordered by last_seen DESC. No pagination (D, item count is low hundreds).
func ListMenuItemsHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        sinceStr := r.URL.Query().Get("since")
        if sinceStr == "" {
            // Default: 7 days ago.
            sinceStr = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
        }
        if _, err := time.Parse("2006-01-02", sinceStr); err != nil {
            writeError(w, http.StatusBadRequest, "since must be YYYY-MM-DD")
            return
        }

        rows, err := pool.Query(r.Context(), `
            SELECT mi.id, mi.master_id, mi.name, mi.menu, mi.menu_group, mi.menu_subgroup,
                   mi.last_seen, mi.created_at,
                   COALESCE(SUM(CASE WHEN dms.business_date >= $1 THEN dms.units_sold ELSE 0 END), 0)::int AS units_week,
                   COALESCE(SUM(CASE WHEN dms.business_date >= $1 THEN dms.gross_amount ELSE 0 END), 0)::float8 AS gross_week
            FROM menu_items mi
            LEFT JOIN daily_menu_sales dms ON dms.menu_item_id = mi.id
            WHERE mi.last_seen >= $1
            GROUP BY mi.id
            ORDER BY mi.last_seen DESC`, sinceStr)
        if err != nil {
            log.Printf("ListMenuItems query: %v", err)
            writeError(w, http.StatusInternalServerError, "internal_error")
            return
        }
        defer rows.Close()

        out := []MenuItemWithSales{}
        for rows.Next() {
            var m MenuItemWithSales
            if err := rows.Scan(&m.ID, &m.MasterID, &m.Name, &m.Menu, &m.MenuGroup, &m.MenuSubgroup,
                &m.LastSeen, &m.CreatedAt, &m.UnitsSoldThisWeek, &m.GrossThisWeek); err != nil {
                log.Printf("ListMenuItems scan: %v", err)
                writeError(w, http.StatusInternalServerError, "internal_error")
                return
            }
            out = append(out, m)
        }
        writeJSON(w, http.StatusOK, out)
    }
}
```

---

### `backend/cmd/sync-toast/main.go` (CLI binary, one-shot)

**Analogs:**
- Structure: `backend/cmd/seed/main.go` (entire file, lines 1-91) — DB connect, ctx, single-purpose, `log.Fatal` on errors.
- Flags: `backend/cmd/import-notion/main.go` lines 91-101 — `flag.String`, `flag.Parse`, required-flag validation.

**Full structure:**

```go
// sync-toast pulls Toast ItemSelectionDetails.csv for [--from, --to] and
// upserts into menu_items + daily_menu_sales. Reuses internal/toast.RunIngest.
//
// Usage:
//   go run ./cmd/sync-toast/ --from 2026-05-01 --to 2026-05-31
//
// Env:
//   DB_URL                  (required)
//   TOAST_SFTP_KEY_PATH     (required, fail-fast)
//   TOAST_SFTP_USER         (default YumYumsExportUser)
//   TOAST_SFTP_HOST         (default s-9b0f88558b264dfda...:22)
//   TOAST_EXPORT_ID         (default 113866)
package main

import (
    "context"
    "flag"
    "log"
    "os"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/yumyums/hq/internal/toast"
)

func main() {
    fromStr := flag.String("from", "", "Start date YYYY-MM-DD (required)")
    toStr := flag.String("to", "", "End date YYYY-MM-DD (required)")
    flag.Parse()

    if *fromStr == "" || *toStr == "" {
        flag.Usage()
        log.Fatal("--from and --to are required")
    }
    fromDate, err := time.Parse("2006-01-02", *fromStr)
    if err != nil { log.Fatalf("--from invalid: %v", err) }
    toDate, err := time.Parse("2006-01-02", *toStr)
    if err != nil { log.Fatalf("--to invalid: %v", err) }

    dbURL := os.Getenv("DB_URL")
    if dbURL == "" { log.Fatal("DB_URL is required") }

    cfg, err := loadToastConfig() // SHARED helper, see below
    if err != nil { log.Fatal(err) }

    ctx := context.Background()
    pool, err := pgxpool.New(ctx, dbURL)
    if err != nil { log.Fatalf("db connect: %v", err) }
    defer pool.Close()
    cfg.Pool = pool

    result, err := toast.RunIngest(ctx, pool, cfg, fromDate, toDate)
    if err != nil { log.Fatalf("ingest: %v", err) }

    log.Printf("done. items_upserted=%d sales_rows_upserted=%d duration=%s",
        result.ItemsUpserted, result.SalesRowsUpserted, result.Duration)
}
```

**Shared `loadToastConfig` helper:** Put in `backend/internal/toast/config.go` (NEW small file) so both `cmd/server/main.go` and `cmd/sync-toast/main.go` call the same env-loading + fail-fast logic. Function signature:

```go
func LoadConfigFromEnv() (Config, error) // returns error on missing/unreadable TOAST_SFTP_KEY_PATH
```

---

### `backend/internal/db/migrations/0060_menu_items.sql` (migration)

**Analog:** `backend/internal/db/migrations/0024_inventory.sql` (full file)

**Schema convention to mirror exactly:**

```sql
-- +goose Up
BEGIN;

CREATE TABLE menu_items (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  master_id     TEXT UNIQUE NOT NULL,           -- stable Toast identifier
  name          TEXT NOT NULL,
  menu          TEXT NOT NULL,
  menu_group    TEXT NOT NULL,
  menu_subgroup TEXT,                            -- nullable; rare today (D-07)
  last_seen     DATE NOT NULL,                  -- bumped to MAX(business_date) when units_sold > 0
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX menu_items_last_seen_idx ON menu_items(last_seen DESC);
CREATE INDEX menu_items_menu_group_idx ON menu_items(menu_group);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS menu_items;
COMMIT;
```

**Conventions taken from 0024_inventory.sql:**
- `+goose Up` / `+goose Down` markers.
- `BEGIN; ... COMMIT;` envelope on both directions.
- UUID PK via `gen_random_uuid()`.
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()` boilerplate.
- `IF NOT EXISTS` on the Down `DROP`s — see line 69-75 of 0024.
- Index naming: `<table>_<col>_idx`.

---

### `backend/internal/db/migrations/0061_daily_menu_sales.sql` (migration)

**Analog:** `backend/internal/db/migrations/0044_low_stock_alert_log.sql` (composite-unique pattern) + `0024_inventory.sql` (FK + ON DELETE CASCADE)

```sql
-- +goose Up
BEGIN;

CREATE TABLE daily_menu_sales (
  menu_item_id  UUID NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
  business_date DATE NOT NULL,
  units_sold    INTEGER NOT NULL,
  gross_amount  NUMERIC(10,2) NOT NULL,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (menu_item_id, business_date)
);

CREATE INDEX daily_menu_sales_business_date_idx ON daily_menu_sales(business_date DESC);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS daily_menu_sales;
COMMIT;
```

**Why composite PK (not UNIQUE + surrogate id):** Mirrors `low_stock_alert_log` (which uses `UNIQUE (item_description, week_start)` alongside a surrogate UUID PK) and `alert_log`. Phase 22 chooses **composite PK directly** — there's no need for a surrogate ID because no other table references a single sales row. Saves one column and one index.

---

### `backend/cmd/server/main.go` (modify — wiring)

**Analog:** itself, lines 482-514 (receipt block) + 522-523 (scheduler).

**Action:** After the existing `purchasing.StartScheduler(ctx, pool)` call at line 523, insert this block:

```go
// Toast SFTP ingest — fails fast on missing key (D-12).
{
    cfg, err := toast.LoadConfigFromEnv()
    if err != nil {
        log.Fatalf("toast worker: %v", err) // fail-fast — server will not start
    }
    cfg.Pool = pool

    // 0 interval disables the worker (operator override; cmd/sync-toast still works).
    if cfg.Interval == 0 {
        log.Println("toast worker: TOAST_SYNC_INTERVAL=0 — worker disabled (cmd/sync-toast still available)")
    } else {
        toast.StartWorker(ctx, cfg)
    }
}
```

**And** the new endpoint inside the existing cookie-auth group (line 400, inside `r.Route("/inventory", ...)`):

```go
r.Get("/menu-items", toast.ListMenuItemsHandler(pool))
```

**Import to add at top of main.go:**

```go
"github.com/yumyums/hq/internal/toast"
```

**Env parsing inside `toast.LoadConfigFromEnv`:**

```go
func LoadConfigFromEnv() (Config, error) {
    keyPath := os.Getenv("TOAST_SFTP_KEY_PATH")
    if keyPath == "" {
        return Config{}, fmt.Errorf("TOAST_SFTP_KEY_PATH is required (no default)")
    }
    if _, err := os.Stat(keyPath); err != nil {
        return Config{}, fmt.Errorf("TOAST_SFTP_KEY_PATH=%q is not readable: %w", keyPath, err)
    }

    interval := 12 * time.Hour
    if s := os.Getenv("TOAST_SYNC_INTERVAL"); s != "" {
        d, err := time.ParseDuration(s)
        if err != nil {
            return Config{}, fmt.Errorf("TOAST_SYNC_INTERVAL %q: %w", s, err)
        }
        interval = d
    }

    cfg := Config{
        SFTPHost:       envOr("TOAST_SFTP_HOST", "s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22"),
        SFTPUser:       envOr("TOAST_SFTP_USER", "YumYumsExportUser"),
        SFTPKeyPath:    keyPath,
        ExportID:       envOr("TOAST_EXPORT_ID", "113866"),
        Interval:       interval,
        SyncWindowDays: 7,
        BackfillDays:   90,
    }
    return cfg, nil
}

func envOr(k, d string) string {
    if v := os.Getenv(k); v != "" { return v }
    return d
}
```

---

### `inventory.html` (modify — frontend tab)

**Analog:** itself — existing 5-tab pattern.

**Action 1 — tab bar (lines 183-189):** Insert new button between Stock (t2) and Trends (t3). Renumber t3/t4/t5:

```html
<div class="tabs">
  <button id="t1" class="on" onclick="show(1)">Purchases</button>
  <button id="t2" onclick="show(2)">Stock</button>
  <button id="t3" onclick="show(3)">Menu</button>      <!-- NEW -->
  <button id="t4" onclick="show(4)">Trends</button>    <!-- was t3 -->
  <button id="t5" onclick="show(5)">Cost</button>      <!-- was t4 -->
  <button id="t6" onclick="show(6)">Setup</button>     <!-- was t5 -->
</div>
```

**Action 2 — section divs (lines 190-233):** Insert new `<div id="s3">` between Stock and Trends, shift others to s4/s5/s6.

```html
<div id="s3" style="display:none">
  <div id="menu-list"></div>
</div>
```

**Action 3 — tab count (line 235):** Bump `data-tabs` from `5` to `6`:

```html
<script src="tab.js" data-tabs="6"></script>
```

**Action 4 — `show()` function (lines 271-284):** Extend the loop to `[1,2,3,4,5,6]` and add a `loadMenu()` branch:

```js
function show(n){
  [1,2,3,4,5,6].forEach(i=>{
    document.getElementById('s'+i).style.display=i===n?'':'none';
    var btn=document.getElementById('t'+i);
    if(btn)btn.className=i===n?'on':'';
  });
  ACTIVE_TAB=n;
  location.hash='tab='+n;
  if(n===1){loadHistory();}
  if(n===2){loadStock();}
  if(n===3){loadMenu();}                       // NEW
  if(n===6){loadItems();if(isAdmin())loadBadgeResetConfig();}
  render();
}
```

**Action 5 — also update `render()` (lines 286-289)** to handle new s3 and renumbered Trends/Cost.

**Action 6 — `loadMenu()` + `renderMenu()` (mirror loadStock pattern, lines 549-560):**

```js
let MENU_DATA = [];

async function loadMenu(){
  showSkeleton('menu-list');
  try{
    var since=new Date(Date.now()-7*24*60*60*1000).toISOString().slice(0,10);
    var data=await api('/api/v1/inventory/menu-items?since='+since);
    MENU_DATA=data||[];
    renderMenu();
  }catch(e){
    if(e.message==='unauthorized')return;
    showInlineError('menu-list','Couldn’t load menu items.','loadMenu()');
  }
}

function renderMenu(){
  var list=document.getElementById('menu-list');
  if(!MENU_DATA.length){
    list.innerHTML='<div class="empty"><h3 style="font-size:15px;font-weight:500;margin:0 0 6px">No menu items</h3><p>Toast ingest has not populated the menu yet. Run sync-toast or wait for the next 12h cycle.</p></div>';
    return;
  }
  var html='';
  MENU_DATA.forEach(function(m){
    html+='<div class="stock-item">'+
      '<div style="flex:1">'+
        '<div class="stock-item-name">'+escHtml(m.name)+'</div>'+
        '<div style="font-size:12px;color:var(--mut)">'+escHtml(m.menu_group||'')+' · last sold '+formatDate(m.last_seen)+'</div>'+
      '</div>'+
      '<div style="text-align:right">'+
        '<div style="font-weight:500">'+(m.units_sold_this_week||0)+' units</div>'+
        '<div style="font-size:11px;color:var(--mut)">this week</div>'+
      '</div>'+
    '</div>';
  });
  list.innerHTML=html;
}
```

**Re-uses existing helpers:** `api()`, `escHtml()`, `formatDate()`, `showSkeleton()`, `showInlineError()`, `.stock-item` CSS class (already styled in inventory.html).

---

### `backend/go.mod` (modify)

**Action:** Promote `github.com/pkg/sftp` and `golang.org/x/crypto/ssh` from indirect to direct imports.

```bash
cd backend && go get github.com/pkg/sftp golang.org/x/crypto/ssh
go mod tidy
```

Use the same version `github.com/pkg/sftp` that sales-processor uses — `cat /Users/jamal/projects/yumyums/sales-processor/go.mod | grep sftp` to confirm before running `go get`.

---

## Shared Patterns

### Worker factory + goroutine
**Source:** `backend/internal/receipt/worker.go` lines 21-55, `backend/internal/purchasing/scheduler.go` lines 22-42
**Apply to:** `backend/internal/toast/worker.go`

Template:
```go
func StartWorker(ctx context.Context, cfg Config) {
    // pre-flight checks (skip-if-disabled OR fail-fast per phase rules)
    log.Printf("xxx worker: starting (interval=%s)", interval)
    go func() {
        runCycle(ctx, cfg) // immediate tick
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                log.Println("xxx worker: shutting down")
                return
            case <-ticker.C:
                runCycle(ctx, cfg)
            }
        }
    }()
}
```

### DB upsert in transaction
**Source:** `backend/internal/receipt/worker.go` lines 219-291
**Apply to:** `backend/internal/toast/ingest.go` upsertDayInTx

Template:
```go
dbTx, err := pool.Begin(ctx)
if err != nil { return fmt.Errorf("...: begin: %w", err) }
defer dbTx.Rollback(ctx) //nolint:errcheck

// ... INSERT ... ON CONFLICT (key) DO UPDATE SET ...

return dbTx.Commit(ctx)
```

### writeJSON / writeError handler helpers
**Source:** `backend/internal/inventory/handler.go` lines 24-32
**Apply to:** `backend/internal/toast/handler.go`

Each package owns its own copies — they're 5 lines each and avoid an internal/httpx package introduction for Phase 22.

```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v) //nolint:errcheck
}
func writeError(w http.ResponseWriter, status int, msg string) {
    writeJSON(w, status, map[string]string{"error": msg})
}
```

### Migration envelope
**Source:** `backend/internal/db/migrations/0024_inventory.sql`, `0044_low_stock_alert_log.sql`
**Apply to:** `0060_menu_items.sql`, `0061_daily_menu_sales.sql`

Template:
```sql
-- +goose Up
BEGIN;
CREATE TABLE ...;
CREATE INDEX ...;
COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS ...;
COMMIT;
```

### env var loader with default
**Source:** `backend/cmd/server/main.go` lines 159-178 (port, superadminPath, templatePath)
**Apply to:** `backend/internal/toast/config.go` LoadConfigFromEnv

Pattern:
```go
v := os.Getenv("KEY")
if v == "" { v = "default" }
```

### CSV column-index map
**Source:** `backend/cmd/import-notion/main.go` lines 138-159
**Apply to:** `backend/internal/toast/parser.go`

Pattern (BOM strip + missing-column hard fail):
```go
colIdx := map[string]int{}
for i, h := range headers {
    h = strings.TrimSpace(h)
    if i == 0 { h = strings.TrimPrefix(h, "\xef\xbb\xbf") }
    colIdx[h] = i
}
for _, col := range required { if _, ok := colIdx[col]; !ok { return error } }
```

### Tab UI extension
**Source:** `inventory.html` lines 183-189, 235, 271-284
**Apply to:** `inventory.html` (Menu tab insertion)

Rules:
1. Update `<button id="tN">` and `<div id="sN">` in tab bar + sections.
2. Bump `<script src="tab.js" data-tabs="N">` count.
3. Extend `[1,2,3,...].forEach` in `show()`.
4. Add `if(n===N){loadX();}` branch in `show()`.
5. Add `if(ACTIVE_TAB===N)renderX();` branch in `render()`.
6. Re-use `.stock-item`, `.tag-section`, `.empty` CSS classes already in the file.

---

## No Analog Found

All Phase 22 files have an in-codebase or in-sister-repo analog. Nothing needs to invent from scratch.

| File | Notes |
|------|-------|
| (none) | every new file maps cleanly to an existing pattern |

---

## Metadata

**Analog search scope:**
- `/Users/jamal/projects/yumyums/sales-processor/sftp/` (verbatim port source)
- `/Users/jamal/projects/yumyums/sales-processor/main.go` (per-date loop reference)
- `/Users/jamal/projects/yumyums/hq/backend/internal/receipt/` (worker + DB-tx patterns)
- `/Users/jamal/projects/yumyums/hq/backend/internal/purchasing/` (ticker + scheduler pattern)
- `/Users/jamal/projects/yumyums/hq/backend/internal/inventory/` (handler + types convention)
- `/Users/jamal/projects/yumyums/hq/backend/internal/db/migrations/` (SQL schema convention)
- `/Users/jamal/projects/yumyums/hq/backend/cmd/server/main.go` (chi routing + env load)
- `/Users/jamal/projects/yumyums/hq/backend/cmd/seed/`, `cmd/import-notion/` (CLI binary structure)
- `/Users/jamal/projects/yumyums/hq/inventory.html` (tab UI pattern)

**Files scanned:** 11
**Pattern extraction date:** 2026-06-03
