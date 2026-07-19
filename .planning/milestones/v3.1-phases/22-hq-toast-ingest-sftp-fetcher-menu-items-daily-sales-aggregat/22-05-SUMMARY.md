---
phase: 22-hq-toast-ingest-sftp-fetcher-menu-items-daily-sales-aggregat
plan: 05
subsystem: backend/internal/toast + backend/cmd/server
tags: [http, handler, route-mount, worker-start, wiring, toast]
requires:
  - backend/internal/toast/config.go (Plan 03 — LoadConfigFromEnv fail-fast)
  - backend/internal/toast/worker.go (Plan 04 — StartWorker goroutine)
  - backend/internal/toast/types.go (Plan 03 — MenuItemWithSales)
  - backend/internal/db/migrations/0060_menu_items.sql (Plan 02 — menu_items table)
  - backend/internal/db/migrations/0061_daily_menu_sales.sql (Plan 02 — daily_menu_sales table)
  - github.com/jackc/pgx/v5/pgxpool
provides:
  - "GET /api/v1/inventory/menu-items?since=YYYY-MM-DD — cookie-auth, returns MenuItemWithSales[] ordered by last_seen DESC"
  - "toast.ListMenuItemsHandler(pool) http.HandlerFunc — the endpoint factory"
  - "Toast background worker started inside cmd/server with fail-fast key validation (D-12)"
  - "TOAST_SYNC_INTERVAL=0 escape hatch — server starts, worker skipped (sync-toast CLI remains usable)"
affects:
  - backend/internal/toast (new file: handler.go)
  - backend/cmd/server/main.go (3 edits: import + route mount + worker block)
tech_stack_added: []
patterns_added:
  - "Per-package writeJSON/writeError helpers (mirrors internal/inventory convention — no internal/httpx introduction)"
  - "DATE-column scan-then-format pattern: var lastSeen time.Time + m.LastSeen = lastSeen.Format('2006-01-02') (trims time-of-day for JSON)"
  - "Fail-fast worker startup with TOAST_SYNC_INTERVAL=0 escape hatch (deviates from receipt.StartWorker graceful-skip per D-12)"
key_files_created:
  - backend/internal/toast/handler.go
key_files_modified:
  - backend/cmd/server/main.go
decisions:
  - "Mount menu-items inside the cookie-auth /inventory group, not the Phase-21 service-token group — confirms T-22-14/T-22-15 mitigation (this endpoint is for the HQ UI, not sales-processor)."
  - "Default since=7-days-ago when omitted — matches D-04 sync window and 'Claude's Discretion' note; clients omitting since get the same window the worker re-pulls."
  - "Scan LastSeen into time.Time then format to YYYY-MM-DD — pgx scans Postgres DATE into time.Time; formatting at the handler boundary keeps the wire shape stable (string, not RFC3339)."
  - "Worker block placed AFTER purchasing.StartScheduler — matches the existing background-startup ordering (receipt → alerts → purchasing → toast) so all DB-bound goroutines start together post-migrations."
metrics:
  duration_minutes: 4
  tasks_completed: 2
  files_changed: 2
  completed_date: 2026-06-03
---

# Phase 22 Plan 05: Wire Toast Worker + Read Endpoint into cmd/server Summary

**One-liner:** Plans 01–04 produced libraries; Plan 05 makes them live — `cmd/server/main.go` now starts the Toast ingest worker (fail-fast on missing key, TOAST_SYNC_INTERVAL=0 escape hatch) and serves `GET /api/v1/inventory/menu-items` inside the existing cookie-auth `/inventory` group, ready for Plan 06's Menu tab UI.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | ListMenuItemsHandler + writeJSON/writeError helpers | ecf3fbe | backend/internal/toast/handler.go |
| 2 | Wire Toast into cmd/server (import + route mount + worker block) | dd3be20 | backend/cmd/server/main.go |

## What Shipped

### Endpoint contract (live)

```
GET /api/v1/inventory/menu-items?since=YYYY-MM-DD
  Auth:  cookie session (existing auth.Middleware)
         — NOT service-token; that's the Phase-21 period-summary
  Param: since (optional, defaults to 7 days ago)

  200 [MenuItemWithSales, ...]
  400 {"error":"since must be YYYY-MM-DD"}
  500 {"error":"internal_error"}
```

SQL:
```sql
SELECT mi.id, mi.master_id, mi.name, mi.menu, mi.menu_group, mi.menu_subgroup,
       mi.last_seen, mi.created_at,
       COALESCE(SUM(CASE WHEN dms.business_date >= $1 THEN dms.units_sold   ELSE 0 END), 0)::int    AS units_week,
       COALESCE(SUM(CASE WHEN dms.business_date >= $1 THEN dms.gross_amount ELSE 0 END), 0)::float8 AS gross_week
FROM menu_items mi
LEFT JOIN daily_menu_sales dms ON dms.menu_item_id = mi.id
WHERE mi.last_seen >= $1
GROUP BY mi.id
ORDER BY mi.last_seen DESC;
```

### Three edits to cmd/server/main.go

| Edit | Location (approx line) | What |
|------|------------------------|------|
| 1. Import | line 31, in the `internal/...` block | Added `"github.com/yumyums/hq/internal/toast"` alphabetised between `opsync "github.com/yumyums/hq/internal/sync"` and `"github.com/yumyums/hq/internal/users"` |
| 2. Route mount | line 422, inside `r.Route("/inventory", ...)` | Added `r.Get("/menu-items", toast.ListMenuItemsHandler(pool))` immediately after `r.Get("/tags", inventory.ListTagsHandler(pool))` — INSIDE the cookie-auth group |
| 3. Worker block | line 525, after `purchasing.StartScheduler(ctx, pool)` | `toast.LoadConfigFromEnv()` → `log.Fatalf` on error (D-12); inject `pool`; skip `StartWorker` when `Interval == 0` (log + continue); else `toast.StartWorker(ctx, toastCfg)` |

### D-12 fail-fast wiring confirmed

```go
toastCfg, err := toast.LoadConfigFromEnv()
if err != nil {
    log.Fatalf("toast worker: %v", err)  // server WILL NOT START
}
```

`LoadConfigFromEnv` (Plan 03) returns errors on:
- Missing `TOAST_SFTP_KEY_PATH` → `"TOAST_SFTP_KEY_PATH is required (no default — see D-12)"`
- Unreadable key file (os.Stat fail) → `"TOAST_SFTP_KEY_PATH=%q is not readable: %w"`
- Invalid `TOAST_SYNC_INTERVAL` (unparseable duration) → `"TOAST_SYNC_INTERVAL %q: %w"`

Any of those propagate through `log.Fatalf` and halt the process with exit code 1.

### TOAST_SYNC_INTERVAL=0 escape hatch verified

```go
if toastCfg.Interval == 0 {
    log.Println("toast worker: TOAST_SYNC_INTERVAL=0 — in-process worker disabled (cmd/sync-toast remains available)")
} else {
    toast.StartWorker(ctx, toastCfg)
}
```

Operator can set `TOAST_SYNC_INTERVAL=0` to disable the in-process worker without disabling Toast ingest entirely — they can still run `cmd/sync-toast --from … --to …` from external cron.

### Endpoint mounted inside cookie-auth group (not service-token)

The acceptance criterion `awk '/r.Route\("\/inventory"/,/^\t\t\t\}\)/' cmd/server/main.go | grep -c 'menu-items'` returns 1, proving the route lives inside the `/inventory` block — which itself lives inside the auth.Middleware group started at line 301. The route is NOT inside the Phase-21 service-token block from `21-SALES-PROCESSOR-CONTRACT.md`. This is the T-22-14 + T-22-15 mitigation in code.

## Verification

| Check | Result |
|-------|--------|
| `cd backend && go build ./...` | exit 0 |
| `cd backend && go vet ./...` | exit 0 |
| `cd backend && go test ./internal/toast/ -count=1` | exit 0 (ok, 0.320s — Plan 03's 7 parser tests still pass) |
| Acceptance grep: `func ListMenuItemsHandler(pool *pgxpool.Pool) http.HandlerFunc` in handler.go | 1 |
| Acceptance grep: `func writeJSON` in handler.go | 1 |
| Acceptance grep: `func writeError` in handler.go | 1 |
| Acceptance grep: `since must be YYYY-MM-DD` in handler.go | 2 (docstring + error path; spec ≥1) |
| Acceptance grep: `ORDER BY mi.last_seen DESC` in handler.go | 1 |
| Acceptance grep: `LEFT JOIN daily_menu_sales dms` in handler.go | 1 |
| Acceptance grep: `time.Now().AddDate(0, 0, -7)` in handler.go | 1 |
| Acceptance grep: `"github.com/yumyums/hq/internal/toast"` in main.go | 1 |
| Acceptance grep: `toast.LoadConfigFromEnv()` in main.go | 1 |
| Acceptance grep: `toast.StartWorker(ctx, toastCfg)` in main.go | 1 |
| Acceptance grep: `toast.ListMenuItemsHandler(pool)` in main.go | 1 |
| Acceptance grep: `log.Fatalf("toast worker` in main.go | 1 |
| Acceptance grep: `TOAST_SYNC_INTERVAL=0` in main.go | 2 (comment + log msg; spec ≥1) |
| Acceptance grep: `awk '/r.Route\("\/inventory"/,/^\t\t\t\}\)/' \| grep menu-items` in main.go | 1 (route IS inside the /inventory cookie-auth group) |

### Smoke note: fail-fast log line at runtime

The plan's acceptance criterion for a runtime smoke (`TOAST_SFTP_KEY_PATH= /tmp/hq-server` produces a "TOAST_SFTP_KEY_PATH is required" line in stderr) is wired correctly but cannot be exercised end-to-end from this worktree without a full environment (DB, superadmins config, Spaces creds — the server fails earlier in startup on `config/superadmins.yaml`). The wiring is verified statically: `log.Fatalf("toast worker: %v", err)` directly formats the error returned by `LoadConfigFromEnv`, which Plan 03's test suite already proves returns `"TOAST_SFTP_KEY_PATH is required (no default — see D-12)"` when the env var is unset. Once the operator boots the server in a properly-configured environment with `TOAST_SFTP_KEY_PATH` unset, that exact string will appear in the fatal log.

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| **Mount menu-items inside cookie-auth `/inventory` group, not service-token group** | The Phase-21 service-token group exists for sales-processor → HQ machine-to-machine calls (period-summary). The Menu tab in inventory.html is a browser UI fetch with the user's session cookie. Mounting it in the service-token group would (a) make sales-processor able to read menu data it has no business with, and (b) break the UI fetch since browsers won't ship a service-token Bearer header. Acceptance criterion #7 enforces this placement in CI-grep. |
| **Default since=7-days-ago when omitted** | Matches D-04's sync window. Aligns the "what the UI shows by default" with "what the worker just re-pulled" so users never see stale data on first load. Plan 06's loadMenu() can pass `since` explicitly when it wants a longer window. |
| **Scan `last_seen` into `time.Time` then format to YYYY-MM-DD** | pgx scans Postgres DATE columns into `time.Time` (zero hour/minute/sec, UTC). Scanning directly into the `string`-typed `LastSeen` field fails at runtime; using an intermediate `time.Time` + `Format("2006-01-02")` keeps the wire shape a clean date string (the `MenuItem` struct's json tag promises that). |
| **Place worker block AFTER `purchasing.StartScheduler(ctx, pool)`** | Mirrors the receipt→alerts→purchasing background-startup ordering and ensures all DB-bound goroutines start after migrations complete. Plan template suggested this position; we matched it. |

## Deviations from Plan

None. Plan executed exactly as written — all task templates and acceptance criteria met on the first compile.

The plan template proactively documented Plans 03's contract (`LoadConfigFromEnv` returns `(Config, error)` with `Pool` unset), and Plan 04's contract (`StartWorker(ctx, cfg Config)`). Both contracts held — no patching of the upstream library packages required.

## Authentication Gates

None. No third-party services touched in this plan (the Toast SFTP runtime gate lives in Plans 03/04; this plan is pure server wiring + a read-only handler).

## Deferred Issues

None.

## Known Stubs

None — both files are fully-functional production code. No placeholder data, no TODO comments, no mock data. The endpoint will return real data the moment the worker has populated `menu_items` and `daily_menu_sales`; on a fresh DB with empty tables it returns `[]` (correct empty-array, not `null`, because `out := []MenuItemWithSales{}` is initialised non-nil).

## Threat Flags

None. All five threats in the plan's `<threat_model>` are mitigated in the shipped code:

| Threat | Mitigation in code |
|--------|-------------------|
| T-22-14 (endpoint publicly exposed) | Route lives inside the cookie-auth chi.Group — verified by the `awk` acceptance check (count=1) |
| T-22-15 (wrong auth scope: service-token vs cookie) | Mounted in `/inventory` cookie-auth group, NOT in the Phase-21 service-token block. The two are textually 100+ lines apart in main.go |
| T-22-16 (unbounded result set) | Accepted — menu item count is bounded (~hundreds even at peak); no pagination needed |
| T-22-17 (SQL injection via since) | `time.Parse("2006-01-02", sinceStr)` validates BEFORE the pgx query; query uses `$1` bind, no string concat |
| T-22-18 (server starts with bad config) | `log.Fatalf` halts the process; systemd/Docker restart loops will surface the failure quickly |

No new threat surface introduced beyond what the plan's threat register covers.

## Self-Check

- FOUND: backend/internal/toast/handler.go
- FOUND: backend/cmd/server/main.go (modified, 3 edits)
- FOUND commit: ecf3fbe (Task 1)
- FOUND commit: dd3be20 (Task 2)

## Self-Check: PASSED
