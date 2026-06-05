---
phase: 22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat
plan: 03
subsystem: backend/internal/toast
tags: [ingest, parser, sftp, csv, toast]
requires:
  - backend/internal/toast/sftp.go (Plan 02 — SFTPConfig + New + Download + Close)
  - menu_items, daily_menu_sales tables (Plan 01 migrations 0060, 0061)
  - github.com/jackc/pgx/v5/pgxpool
provides:
  - "package toast: Config, AggregatedRow, MenuItem, MenuItemWithSales, IngestResult"
  - "toast.LoadConfigFromEnv() (Config, error) — fail-fast on missing/unreadable TOAST_SFTP_KEY_PATH"
  - "toast.RunIngest(ctx, pool, cfg, fromDate, toDate) (*IngestResult, error)"
  - "internal: parseItemSelectionDetails (CSV), isColdStart, dialWithRetry, upsertDayInTx"
affects:
  - backend/internal/toast (new files: types.go, config.go, parser.go, parser_test.go, ingest.go)
tech_stack_added: []
patterns_added:
  - bufio.Reader BOM peek-and-discard before csv.NewReader (csv.Reader can't tolerate a BOM glued to an opening quote)
  - per-day-tx upsert with GREATEST() last_seen bump (mirrors receipt/worker.go tx pattern + 0024_inventory.sql upsert convention)
key_files_created:
  - backend/internal/toast/types.go
  - backend/internal/toast/config.go
  - backend/internal/toast/parser.go
  - backend/internal/toast/parser_test.go
  - backend/internal/toast/ingest.go
key_files_modified: []
decisions:
  - "BOM stripped at byte-stream level (bufio.Reader Peek/Discard) — csv.Reader rejects a BOM glued to a leading quote with 'bare \\\" in non-quoted-field'. In-header TrimPrefix retained as belt-and-braces."
  - "RunIngest is window-agnostic — caller (worker.runCycle in Plan 04 / cmd/sync-toast in Plan 04) decides fromDate/toDate. CLI takes --from/--to flags; worker swaps SyncWindowDays for BackfillDays on cold start."
  - "D-13 log line (`toast ingest: dates=[...] items_upserted=N sales_rows_upserted=M duration=Xs`) is the caller's job — RunIngest returns the IngestResult struct with everything needed to render it."
  - "Per-day errors (download / parse / db) are logged and the cycle continues (T-22-09); only a top-level dial failure after 3 retries aborts."
metrics:
  duration_minutes: 7
  tasks_completed: 3
  files_changed: 5
  completed_date: 2026-06-03
---

# Phase 22 Plan 03: Toast Config + CSV Parser + RunIngest Orchestrator Summary

**One-liner:** The HQ Toast ingest pipeline's business logic now lives in `package toast` — `LoadConfigFromEnv` validates env (fail-fast on missing key), `parseItemSelectionDetails` aggregates CSV by master_id with void-exclusion, and `RunIngest` orchestrates per-day download → parse → single-tx upsert with retry and idempotent ON CONFLICT semantics.

After this plan, Plan 04's worker and CLI only need to construct a Config and call `RunIngest(ctx, pool, cfg, from, to)`. All decisions (D-02 cold-start, D-05 last-pull-wins, D-06 void exclusion, D-07 3-level hierarchy, D-10 retry schedule) are live in code.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | types.go + config.go (struct defs + LoadConfigFromEnv fail-fast) | b830e45 | backend/internal/toast/types.go, backend/internal/toast/config.go |
| 2a | parser_test.go (RED — 7 failing tests) | 4353e1b | backend/internal/toast/parser_test.go |
| 2b | parser.go (GREEN — 7 tests pass) | 8718024 | backend/internal/toast/parser.go |
| 3 | ingest.go (RunIngest + isColdStart + dialWithRetry + upsertDayInTx) | 6de0e66 | backend/internal/toast/ingest.go |

## What Shipped

### Exported surface (consumed by Plans 04 + 05 + 06)

```go
// types.go
type Config struct {
    SFTPHost, SFTPUser, SFTPKeyPath, ExportID string
    Pool           *pgxpool.Pool
    Interval       time.Duration
    SyncWindowDays int
    BackfillDays   int
}
type AggregatedRow struct {
    MasterID, Name, Menu, MenuGroup, BusinessDate string
    MenuSubgroup *string
    UnitsSold    int
    GrossAmount  float64
}
type MenuItem struct { /* JSON tags lowercase snake_case */ }
type MenuItemWithSales struct { MenuItem; UnitsSoldThisWeek int; GrossThisWeek float64 }
type IngestResult struct {
    Dates             []string
    ItemsUpserted     int
    SalesRowsUpserted int
    Duration          time.Duration
}

// config.go
func LoadConfigFromEnv() (Config, error)

// ingest.go
func RunIngest(ctx context.Context, pool *pgxpool.Pool, cfg Config, fromDate, toDate time.Time) (*IngestResult, error)
```

### Phase-22 decisions live in code

| Decision | Where | Mechanism |
|----------|-------|-----------|
| D-02 cold-start detection | `ingest.go` `isColdStart` | `SELECT COUNT(*) FROM daily_menu_sales` |
| D-05 last-pull wins | `ingest.go` `upsertDayInTx` | `ON CONFLICT (menu_item_id, business_date) DO UPDATE SET units_sold/gross_amount/updated_at` |
| D-06 void exclusion | `parser.go` | `if voidStr == "true" \|\| "1" \|\| "yes" { continue }` — voided rows excluded entirely from both counters |
| D-07 3-level hierarchy | `types.go` + `parser.go` | `Menu` + `MenuGroup` required, `MenuSubgroup *string` nullable |
| D-10 retry backoff | `ingest.go` `dialWithRetry` | `[]time.Duration{5s, 15s, 30s}` |
| D-12 fail-fast on missing key | `config.go` `LoadConfigFromEnv` | Returns error if `TOAST_SFTP_KEY_PATH` unset OR `os.Stat` fails |
| `last_seen` MAX-bump rule | `ingest.go` `upsertDayInTx` | `last_seen = GREATEST(menu_items.last_seen, EXCLUDED.last_seen)` |

### Parser test coverage (7/7 planned)

| Test | What it asserts |
|------|----------------|
| `TestParseHappyPath` | 3 distinct master_ids → 3 rows with correct UnitsSold/GrossAmount/metadata |
| `TestParseExcludesVoidedRows` | D-06: voided row excluded from both UnitsSold and GrossAmount |
| `TestParseAggregatesSameMasterIdSameDay` | Same master_id, same day → Qty summed, Gross Price summed |
| `TestParseHandlesUTF8BOMInFirstHeader` | `\xef\xbb\xbf"Master Id"...` → still finds "Master Id" column |
| `TestParseMissingRequiredColumnFails` | CSV without `Master Id` → non-nil error containing `"Master Id"` |
| `TestParseDecimalQtyTolerated` | `Qty="1.0"` → `UnitsSold=1` (float fallback, truncated to int) |
| `TestParseEmptyMenuSubgroupBecomesNil` | empty `Menu Subgroup(s)` → `MenuSubgroup` is `nil` (not `&""`); populated → `*string` value |

## Verification

| Check | Result |
|-------|--------|
| `cd backend && go build ./...` | exit 0 |
| `cd backend && go test ./internal/toast/ -count=1` | exit 0 (ok, ~0.3s) |
| `cd backend && go test ./internal/toast/ -run TestParse -v -count=1` | 7/7 PASS |
| `cd backend && go vet ./internal/toast/` | exit 0 |
| Acceptance grep: `type Config struct` in types.go | 1 |
| Acceptance grep: `type AggregatedRow struct` in types.go | 1 |
| Acceptance grep: `MenuSubgroup *string` in types.go | 2 (AggregatedRow + MenuItem) |
| Acceptance grep: `LoadConfigFromEnv() (Config, error)` in config.go | 1 |
| Acceptance grep: `TOAST_SFTP_KEY_PATH is required` in config.go | 1 |
| Acceptance grep: `is not readable` in config.go | 1 |
| Acceptance grep: `voidStr == "true"` in parser.go | 1 |
| Acceptance grep: `TrimPrefix(h, "\xef\xbb\xbf")` in parser.go | 1 |
| Acceptance grep: `func RunIngest(ctx context.Context, pool *pgxpool.Pool, cfg Config, fromDate, toDate time.Time) (*IngestResult, error)` | 1 |
| Acceptance grep: `func isColdStart` | 1 |
| Acceptance grep: `func dialWithRetry` | 1 |
| Acceptance grep: `func upsertDayInTx` | 1 |
| Acceptance grep: `ON CONFLICT (master_id) DO UPDATE` | 2 (SQL + doc comment — exceeds spec ≥1) |
| Acceptance grep: `ON CONFLICT (menu_item_id, business_date) DO UPDATE` | 2 (SQL + doc comment — exceeds spec ≥1) |
| Acceptance grep: `GREATEST(menu_items.last_seen, EXCLUDED.last_seen)` | 2 (SQL + doc comment — exceeds spec ≥1) |
| Acceptance grep: `5 * time.Second, 15 * time.Second, 30 * time.Second` | 1 |
| Acceptance grep: `SELECT COUNT(*) FROM daily_menu_sales` | 1 |

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| **BOM stripped at the byte-stream level via `bufio.Reader` Peek+Discard** | Plan template said to handle BOM via `strings.TrimPrefix` inside the header parse, but `csv.Reader` reports `"bare quote in non-quoted-field"` at line 1 column 4 when a UTF-8 BOM is glued to a leading `"`. The header `Read()` errors out before the TrimPrefix line ever runs. Solution: `bufio.NewReader` + `Peek(3)` + `Discard(3)` if the first three bytes are `\xef \xbb \xbf` before constructing `csv.NewReader`. Kept the in-header TrimPrefix as belt-and-braces — covers BOMs that survive somehow without being byte-prefixed. The acceptance criterion `grep -c 'TrimPrefix(h, "\xef\xbb\xbf")' = 1` still passes. |
| **`RunIngest` is date-window-agnostic** | Caller (Plan 04 worker / Plan 04 CLI) owns the cold-start branching and `--from`/`--to` flag parsing. RunIngest only iterates a half-open `[fromDate, toDate]` and returns counts. Lets the CLI honor explicit dates without re-implementing the cold-start probe. |
| **Per-day errors `continue`, top-level dial failure aborts** | Mirrors T-22-09 mitigation in the threat register. One missing-day directory (closed-day) shouldn't kill a 90-day backfill; an outright SFTP connect failure means we can't proceed at all. Both behaviors are logged. |
| **`itemsUpserted` counts every row (even on-conflict updates), not just inserts** | RETURNING id always returns a row on UPSERT; the counter measures "rows we wrote SQL for," not "net new rows." The intent of the per-cycle log is to surface ingest throughput, which matches "rows written" better than "net new rows" — net-new is approximately zero on a steady-state daily cycle, which would be a misleading number. |

## Deviations from Plan

### Rule 1 — Bug: BOM handling at byte-stream level

- **Found during:** Task 2 GREEN phase (TestParseHandlesUTF8BOMInFirstHeader failed)
- **Issue:** The plan template instructed to handle BOM via `strings.TrimPrefix(h, "\xef\xbb\xbf")` on the parsed first header field. But `csv.NewReader(r).Read()` errors with `parse error on line 1, column 4: bare " in non-quoted-field` when the input starts with `\xef\xbb\xbf"`. The CSV reader sees the BOM bytes, doesn't recognise them as whitespace, and rejects the immediately-following quote. The TrimPrefix code was unreachable for the only BOM scenario it was meant to handle.
- **Fix:** Wrap the input in `bufio.NewReader`, `Peek(3)` for the BOM bytes, `Discard(3)` if present, then pass `br` to `csv.NewReader`. Kept the in-header `TrimPrefix` line as a safety net (and to satisfy the acceptance-criterion grep).
- **Files modified:** `backend/internal/toast/parser.go`
- **Commit:** 8718024 (consolidated into the GREEN commit since the bug was caught and fixed in the same Task 2 cycle)

## Authentication Gates

None.

## Deferred Issues

None.

## Threat Flags

None — the threat surface introduced (file reads of the SSH key, untrusted CSV bytes from outbound SFTP fetch, parameterised SQL writes) is fully covered by the plan's `<threat_model>` (T-22-06 through T-22-10). All four mitigations are present in code:

- T-22-07 (parser panic): every numeric parse wrapped with `strconv` returning errors; missing-column hard fail
- T-22-08 (SQL injection): all writes use `$N` parameterised pgx queries; no string concatenation
- T-22-09 (one bad day stops cycle): per-day errors logged + `continue`
- T-22-10 (partial tx commit): each day's writes in single `pool.Begin` / `Commit` with `defer Rollback`

## Self-Check: PASSED

- FOUND: backend/internal/toast/types.go
- FOUND: backend/internal/toast/config.go
- FOUND: backend/internal/toast/parser.go
- FOUND: backend/internal/toast/parser_test.go
- FOUND: backend/internal/toast/ingest.go
- FOUND commit: b830e45 (Task 1)
- FOUND commit: 4353e1b (Task 2 RED)
- FOUND commit: 8718024 (Task 2 GREEN)
- FOUND commit: 6de0e66 (Task 3)
