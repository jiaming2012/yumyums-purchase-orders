---
phase: 260607-bir
plan: 01
type: execute
completed: 2026-06-07
tasks_completed: 3
files_changed: 7
commits:
  - 2ae1705
  - 8d48cec
  - 70b78a8
---

# 260607-bir: Mercury Receipt Sync Button — Summary

On-demand "Sync Receipts" button in the inventory.html Purchases tab. Triggers
the existing Mercury receipt worker on demand with status durability that
survives full page reload and PWA close/reopen, single-flight enforcement via
a partial unique index, and panic-safe goroutine recovery.

## Files Changed

| File                                                      | Change                                                                                                                            |
| --------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `backend/internal/db/migrations/0067_receipt_sync_runs.sql` | NEW — `receipt_sync_runs` table + partial unique index on status='running' + index on started_at DESC                            |
| `backend/internal/receipt/worker.go`                      | `runIngestCycle` now returns `(IngestResult, error)`; exported `RunIngestCycle` wrapper; `StartWorker` callers discard the result |
| `backend/internal/receipt/worker_test.go`                 | + `TestRunIngestCycle_NoTransactions_ReturnsZeroResult` (covers new signature)                                                    |
| `backend/internal/inventory/sync_receipts.go`             | NEW — `SyncReceiptsHandler`, `SyncReceiptsStatusHandler`, `IngestRunner` injection seam, `runSyncGoroutine` with `defer recover()` |
| `backend/internal/inventory/sync_receipts_test.go`        | NEW — 3 tests: 409 single-flight, success counts persisted, panic → status=failed + no orphan row                                  |
| `backend/cmd/server/main.go`                              | Hoist `receiptCfg` construction above route block; register `POST /sync-receipts` + `GET /sync-receipts/status` inside `/inventory` |
| `inventory.html`                                          | Sync button + status chip in `#s1`; visibility-aware polling JS; wired into `show(1)` and DOMContentLoaded bootstrap              |
| `sw.js`                                                   | Regenerated via `node build-sw.js` (Workbox precache covers updated HTML)                                                          |
| `tests/inventory.spec.js`                                 | + `Receipt sync button` describe block with 3 tests using `page.route` stubs                                                       |

## Migration Applied

`0067_receipt_sync_runs.sql` — successfully applied to remote `hq_test` DB
during test runs:

```
2026/06/07 08:29:23 OK   0067_receipt_sync_runs.sql (384.42ms)
2026/06/07 08:29:23 goose: successfully migrated database to version: 67
```

Schema:
```sql
CREATE TABLE receipt_sync_runs (
  id              BIGSERIAL PRIMARY KEY,
  started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at     TIMESTAMPTZ,
  status          TEXT NOT NULL CHECK (status IN ('running','done','failed')),
  processed       INTEGER NOT NULL DEFAULT 0,
  auto_created    INTEGER NOT NULL DEFAULT 0,
  pending_review  INTEGER NOT NULL DEFAULT 0,
  cached          INTEGER NOT NULL DEFAULT 0,
  error           TEXT,
  triggered_by    TEXT NOT NULL DEFAULT 'manual'
);
CREATE UNIQUE INDEX receipt_sync_runs_single_running
  ON receipt_sync_runs ((1)) WHERE status = 'running';
CREATE INDEX receipt_sync_runs_started_at_desc
  ON receipt_sync_runs (started_at DESC);
```

## Endpoint Contracts

### `POST /api/v1/inventory/sync-receipts`

Trigger an on-demand Mercury receipt sync. Bearer/cookie auth (inside the
existing `/inventory` chi group).

**Success (200)** — first POST while no run is active:
```json
{
  "id": 42,
  "status": "running",
  "started_at": "2026-06-07T13:44:12.391Z"
}
```
Side-effect: goroutine spawned that calls `receipt.RunIngestCycle` and
writes the terminal status (`done` or `failed`) back to the row.

**Conflict (409)** — another POST while the previous run is still running:
```json
{ "error": "sync_already_running" }
```

### `GET /api/v1/inventory/sync-receipts/status`

Returns the latest row from `receipt_sync_runs`, or `null` when the table
is empty.

**Empty table (200)**:
```
null
```

**Latest row (200)**:
```json
{
  "id": 42,
  "started_at": "2026-06-07T13:44:12.391Z",
  "finished_at": "2026-06-07T13:44:18.214Z",
  "status": "done",
  "processed": 5,
  "auto_created": 3,
  "pending_review": 2,
  "cached": 0,
  "error": null,
  "triggered_by": "manual"
}
```

When `status: "failed"`, the `error` field contains the runtime error or
`"panic: <recovered value>"`.

## Test Results

### Backend (Go)

```
$ DB_TEST_URL=postgres://yumyums:yumyums@100.70.200.55:5433/hq_test?sslmode=disable \
  go test ./internal/receipt/ -run TestRunIngestCycle_NoTransactions_ReturnsZeroResult \
  ./internal/inventory/ -run TestSyncReceipts -count=1 -v

=== RUN   TestRunIngestCycle_NoTransactions_ReturnsZeroResult
--- PASS: TestRunIngestCycle_NoTransactions_ReturnsZeroResult (1.74s)
=== RUN   TestSyncReceipts_SingleFlight_Returns409
--- PASS: TestSyncReceipts_SingleFlight_Returns409 (0.46s)
=== RUN   TestSyncReceipts_Goroutine_UpdatesRowToDone
--- PASS: TestSyncReceipts_Goroutine_UpdatesRowToDone (0.19s)
=== RUN   TestSyncReceipts_Goroutine_RecoversFromPanic
2026/06/07 08:29:24 SyncReceipts goroutine panic for run 1: boom
--- PASS: TestSyncReceipts_Goroutine_RecoversFromPanic (0.14s)
PASS
```

`go build ./...` clean.

### Frontend (Playwright)

```
$ npx playwright test tests/inventory.spec.js -g "Receipt sync button" --reporter=line

Running 3 tests using 1 worker
[1/3] [chromium] › Receipt sync button › clicking Sync Receipts disables button and shows Syncing…
[2/3] [chromium] › Receipt sync button › reload mid-run shows Syncing… (state survives via GET /status)
[3/3] [chromium] › Receipt sync button › completed run shows summary chip with counts
  3 passed (14.3s)
```

## Commits

| Hash    | Message                                                                       |
| ------- | ----------------------------------------------------------------------------- |
| 2ae1705 | feat(260607-bir): migration 0067 + runIngestCycle returns IngestResult        |
| 8d48cec | feat(260607-bir): POST /sync-receipts (single-flight + panic-safe) and GET /sync-receipts/status |
| 70b78a8 | feat(260607-bir): Sync Receipts button + status chip in inventory Purchases tab |

## Deviations from Plan

**1. typed pgconn check vs substring match.** The plan offered two options for
`isUniqueViolation`: a cheap substring match against `err.Error()` containing
`"23505"`, or a typed `errors.As(err, &pgErr); pgErr.Code == "23505"` check.
The plan said "use the typed version if the import is already common in the
package" — `pgconn` is already imported in
`backend/internal/sync/listener.go`, so I went with the typed check. This is
slightly more robust (no string-match false positives) and removed the
fragile `contains`/`indexOf` helpers the plan sketched.

**2. receiptCfg hoist location.** The plan called out that `receiptCfg` was
declared after the route block and offered two options for the smaller diff:
hoist it before the routes OR add a follow-up `r.Route` extension after it.
I chose the hoist: it lifts the `workerInterval`/`lookbackDays`/`receiptCfg`
block as a contiguous unit just before `r := chi.NewRouter()` and collapses
the original construction site to a single `receipt.StartWorker(ctx,
receiptCfg)` call. Net: one declaration, no duplicated env parsing.

**3. `escapeHTML` → `syncEscapeHTML`.** The plan sketched a top-level
`escapeHTML(str)` helper. Renamed to `syncEscapeHTML` to avoid stomping on
any potential future name collision with the rest of `inventory.html` — the
file already defines several short-named helpers and a generic
`escapeHTML` would be too easy to redeclare by accident.

**4. `loadHistory()` refresh on done.** The plan sketched
`if(SYNC_STATE&&SYNC_STATE.status==='done'){loadHistory();}` which would
re-fire `loadHistory` on every poll once the row reaches `done`. Tightened
the guard so it only fires on the transition `running → done` (Rule 1 fix —
prevents redundant list reloads on subsequent visibility-resume polls). Also
guarded with `typeof loadHistory==='function'` to defend against ordering.

No other deviations.

## How to Use (for the owner)

1. Open Yumyums HQ → Inventory → Purchases tab.
2. Tap **Sync Receipts** above the vendor filter. The button switches to
   **Syncing…** and disables.
3. While the sync is running you can switch to a different tab, reload the
   page, or close and reopen the PWA — the next time you visit the
   Purchases tab the button will still read **Syncing…** until the run
   finishes.
4. When the sync completes, a green chip appears: **"Last synced 0m ago — N
   processed, M pending review"**. Tap **×** to dismiss it. The Purchases
   list auto-refreshes with any newly ingested rows.
5. If the sync fails, a red chip shows the error — tap **Sync Receipts**
   again to retry.
6. If two browser tabs (or two crew members) tap the button at the same
   time, the second request gets a 409 and the UI silently picks up the
   first run's state — no double-spend on Mercury API quota.

## Self-Check: PASSED

**Files exist:**
- backend/internal/db/migrations/0067_receipt_sync_runs.sql — FOUND
- backend/internal/inventory/sync_receipts.go — FOUND
- backend/internal/inventory/sync_receipts_test.go — FOUND
- inventory.html — modified (Purchases tab `#s1` contains `#sync-receipts-btn`)
- tests/inventory.spec.js — modified (`Receipt sync button` describe block at tail)
- sw.js — regenerated (21 files precached, 1362.9 KB)

**Commits exist:**
- 2ae1705 — FOUND
- 8d48cec — FOUND
- 70b78a8 — FOUND
