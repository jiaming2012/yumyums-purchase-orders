---
phase: quick-260607-fxl
plan: 01
type: execute
status: complete
completed: 2026-06-07
tasks_completed: 4
commits:
  - c71cd34: feat(260607-fxl) worker retries parse-failed rows when parse_error null AND items empty
  - 8a728c1: test(260607-fxl) worker retries parse-failed rows only when parse_error null AND items empty
  - 35ed701: feat(260607-fxl) handler gates empty-items confirm + structured 422 envelopes for empty_items_not_allowed and total_mismatch
  - e0b0610: feat(260607-fxl) fe disables Confirm Receipt button when items empty (non-no-attachment reason) or totals mismatch
key_files:
  modified:
    - backend/internal/receipt/worker.go
    - backend/internal/receipt/worker_test.go
    - backend/internal/inventory/handler.go
    - backend/internal/inventory/period_summary_test.go
    - inventory.html
    - sw.js
    - tests/inventory.spec.js
requirements_met:
  - FIX-A-WORKER-RETRY
  - FIX-A-WORKER-TESTS
  - FIX-B-BACKEND-CONFIRM-GATE
  - FIX-B-FE-CONFIRM-DISABLED
---

# Quick Task 260607-fxl: Retry parse-failed rows + confirm gate Summary

Two bundled fixes for the pending-purchases pipeline that share the same DB column (`pending_purchases.reason`) and the same review-form FE surface — retry pre-260607-e1c parse-failed rows once with Sonnet, and block accidental confirms on rows whose receipts were never itemized.

## What shipped

### Task 1 — FIX A worker: parse-failed retry branch
- Extended `classifyExistingTx` (`backend/internal/receipt/worker.go:339-405`) to return 5 values: `(kind, reason string, hasParseError, hasItems bool, err error)`.
  - SQL now also selects `(parse_error IS NOT NULL)` and `(COALESCE(jsonb_array_length(items), 0) > 0)` for the `kind="pending"` branch.
- `runIngestCycle` (`worker.go:159-205`) now OR's `noAttachmentUpgrade` with a new `parseFailedRetry` condition gated by `parse_error IS NULL AND !hasItems AND len(tx.Attachments) > 0`.
- No changes to `routePending` — when the retry itself fails, existing `updatePendingPurchase` writes `parse_error`, so the next sync sees `hasParseError=true` and skips.

### Task 2 — FIX A tests
- 3 new tests in `backend/internal/receipt/worker_test.go`:
  - `TestRunIngestCycle_RetriesPriorHaikuFailureWhenParseErrorNull` — happy path retry via Sonnet (`worker_test.go:691-770`).
  - `TestRunIngestCycle_DoesNotRetryWhenParseErrorSet` — parse_error guard prevents loop (`worker_test.go:772-825`).
  - `TestRunIngestCycle_DoesNotRetryParseFailedWithItems` — items guard preserves operator edits (`worker_test.go:827-884`).
- Updated 5 direct callers of `classifyExistingTx` in `TestClassifyExistingTx` for the 5-return signature.
- Updated `TestRunIngestCycle_SkipsRealCached` seed to populate `parse_error='haiku boom; sonnet boom'` so it remains "cached" under the new retry-on-NULL-parse_error logic.

### Task 3 — FIX B backend
- `ConfirmPendingPurchaseHandler` (`backend/internal/inventory/handler.go:634-720`) now fetches `reason` alongside `bank_tx_id`/`bank_total`.
- New 422 envelope `{error:"empty_items_not_allowed", reason:"add at least one line item or set pending reason to no_attachment_on_bank_tx"}` returned when `len(LineItems)==0 AND reason != "no_attachment_on_bank_tx"`.
- Total-mismatch upgraded from text-400 to structured 422 envelope `{error:"total_mismatch", line_total:X.XX, bank_total:Y.YY}` using `math.Round(*100)/100` to avoid floating-point noise. Imports: added `database/sql` + `math`; removed now-unused `fmt`.
- Migrated the existing `t.Run("end-to-end empty-items confirm increments cogs", ...)` seed in `period_summary_test.go:528-540` to `UPDATE pending_purchases SET reason = 'no_attachment_on_bank_tx'` so the existing test continues to land in the positive-allowlist branch.
- 3 new top-level tests added at the bottom of `period_summary_test.go:1550-1718`:
  - `TestConfirmPending_RejectsEmptyItemsWhenReceiptAttached`
  - `TestConfirmPending_AcceptsEmptyItemsWhenNoAttachment` (regression guard for 260605-pk1 flow)
  - `TestConfirmPending_RejectsTotalMismatchWith422`
- Shared `confirmPendingHelper` reduces boilerplate.

### Task 4 — FIX B FE
- `renderReviewForm` (`inventory.html:617-633`) computes `canConfirm = (hasItems AND matches) OR (!hasItems AND isNoAttachment)` and emits the `disabled` attribute on the Confirm Receipt button (`line 658`).
- Input listener block (`inventory.html:1011-1023`) recomputes `canConfirm` after every change using `PENDING_PURCHASES.find(...).reason` so the disabled state stays in sync as items are added/removed and totals change.
- Click handler short-circuits on `el.disabled` (`inventory.html:951-952`) as defense-in-depth against stale events.
- `.btn-primary:disabled` CSS (`inventory.html:70`) adds `opacity:0.5; cursor:not-allowed`.
- `sw.js` regenerated via `node build-sw.js` so the updated `inventory.html` lands in the precache manifest with a fresh content hash.
- 3 new Playwright tests under `test.describe('Confirm Receipt disabled state (260607-fxl)')` in `tests/inventory.spec.js:2992-3068`.

## Test results

### Backend

```
go test -count=1 ./internal/receipt/...     → PASS (~12s)
go test -count=1 ./internal/inventory/...   → PASS (~25s)
go build ./...                              → PASS
```

New backend tests:
- `TestRunIngestCycle_RetriesPriorHaikuFailureWhenParseErrorNull` PASS
- `TestRunIngestCycle_DoesNotRetryWhenParseErrorSet` PASS
- `TestRunIngestCycle_DoesNotRetryParseFailedWithItems` PASS
- `TestConfirmPending_RejectsEmptyItemsWhenReceiptAttached` PASS
- `TestConfirmPending_AcceptsEmptyItemsWhenNoAttachment` PASS
- `TestConfirmPending_RejectsTotalMismatchWith422` PASS

Existing tests confirmed not regressed:
- `TestRunIngestCycle_SkipsRealCached` PASS (with the `parse_error` seed update)
- `TestRunIngestCycle_UpgradesPendingNoAttachmentRow` PASS
- `TestRunIngestCycle_UpgradesPendingNoAttachmentRow_ParseFails` PASS
- `TestRunIngestCycle_SkipsExistingPurchaseEvent` PASS
- `TestClassifyExistingTx` (all 5 subtests) PASS
- `TestPeriodSummary/end-to-end_empty-items_confirm_increments_cogs` PASS (with the `UPDATE reason` injection)
- `TestPeriodSummary/*` full suite PASS

### Frontend (Playwright)

```
DB_HOST=100.70.200.55 DB_PORT=5433 TOAST_SFTP_KEY_PATH=/tmp/yumyums-test/fake-toast-key TOAST_SYNC_INTERVAL=0 \
  npx playwright test tests/inventory.spec.js -g "Confirm Receipt disabled state"
```

- `Confirm Receipt button is disabled when totals do not match and items are non-empty` PASS
- `Confirm Receipt button is disabled when items are empty and pending reason is parse-failed` PASS
- `Confirm Receipt button is enabled when items match bank total` PASS

Prior-phase smoke tests (no regressions):
- `Receipt sync button` (4 tests) PASS
- `Pending card — parse_error display (260607-e1c)` PASS
- `PDF receipt iframe (260607-e1c)` PASS

## Deviations from plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestRunIngestCycle_SkipsRealCached seed required parse_error to stay "cached"**
- **Found during:** Task 1 (during baseline test run before adding T2's new tests)
- **Issue:** Existing test seeds a row with `reason='Receipt could not be parsed automatically'`, `parse_error=NULL`, `items='[]'`, and a tx with attachments. Under the NEW logic, this is exactly the parseFailedRetry case → would now retry through Sonnet instead of being cached. The test's existing assertion `parseCallCount == 0` would fail.
- **Fix:** Added `parse_error='haiku boom; sonnet boom'` to the seed INSERT so the row correctly classifies as already-tried-both-models (NOT eligible for retry).
- **Files modified:** `backend/internal/receipt/worker_test.go` (line ~628-635 region)
- **Commit:** 8a728c1 (T2)

**2. [Rule 3 - Blocking] insertTestUser helper used pre-migration schema columns**
- **Found during:** Task 3 (when running new TestConfirmPending tests)
- **Issue:** Helper at `period_summary_test.go:214` used `display_name` and `role` columns. Migration `0017_users_naming.sql` dropped `display_name` (replaced by `first_name + last_name`), and the role-array refactor in Phase 11 replaced `role` (text) with `roles` (text[]). The helper was already broken on HEAD before this task — confirmed by running the pre-existing `TestPeriodSummary/end-to-end_empty-items_confirm_increments_cogs` test directly, which failed with the same schema error. Without fixing this, my new T3 tests cannot resolve `confirmed_by` FK.
- **Fix:** Updated INSERT to use `first_name, last_name, roles` (with `ARRAY['admin']::text[]`); added `ON CONFLICT (email) DO UPDATE SET status = EXCLUDED.status` so reruns against a non-truncated `users` table stay idempotent (resetFixtures does not truncate users).
- **Files modified:** `backend/internal/inventory/period_summary_test.go:214-234`
- **Commit:** 35ed701 (T3)

**3. [Rule 3 - Blocking] Removed unused `fmt` import after handler edits**
- **Found during:** Task 3 (post-edit `go build` failure)
- **Issue:** Removing the `fmt.Sprintf` in the total-mismatch branch made `fmt` import unused → `go build` errored with "imported and not used".
- **Fix:** Dropped `"fmt"` from the import block in `backend/internal/inventory/handler.go`.
- **Files modified:** `backend/internal/inventory/handler.go:1-17`
- **Commit:** 35ed701 (T3)

## Authentication / environment gates

The Playwright `task test` infrastructure expects local Docker Postgres at `localhost:5432`; this worktree ran with `DB_HOST=100.70.200.55 DB_PORT=5433` (Tailscale). Server startup also requires `TOAST_SFTP_KEY_PATH` to point at a readable file (a `log.Fatalf` happens otherwise) — used a 0-byte stub at `/tmp/yumyums-test/fake-toast-key` together with `TOAST_SYNC_INTERVAL=0` to disable the worker. These are environment-setup considerations, not deviations to the plan logic.

## Follow-ups

- `insertTestUser` schema-drift fix removed friction for future tests in the inventory package. Other long-standing helpers in this file (`insertPendingPurchase` etc.) may have similar drift — out of scope for this task but worth a dedicated audit pass.
- The end-to-end manual smoke list at the bottom of the plan (Restaurant Depot $391.96 auto-resolve, etc.) requires a real Mercury connection and a live Anthropic API key; not executed here.

## Self-Check: PASSED

Verified files exist:
- FOUND: backend/internal/receipt/worker.go (touched in c71cd34)
- FOUND: backend/internal/receipt/worker_test.go (touched in 8a728c1)
- FOUND: backend/internal/inventory/handler.go (touched in 35ed701)
- FOUND: backend/internal/inventory/period_summary_test.go (touched in 35ed701)
- FOUND: inventory.html (touched in e0b0610)
- FOUND: sw.js (touched in e0b0610)
- FOUND: tests/inventory.spec.js (touched in e0b0610)

Verified commits exist:
- FOUND: c71cd34
- FOUND: 8a728c1
- FOUND: 35ed701
- FOUND: e0b0610
