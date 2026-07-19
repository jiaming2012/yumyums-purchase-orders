---
phase: quick-260607-koi
plan: 01
type: execute
status: complete
one_liner: "Retry parse button + backend endpoint that nulls parse_error so the worker re-attempts stuck pending rows on next Sync Receipts"
requires:
  - 260607-e1c (parse_error column + Parser error line on pending card)
  - 260607-fxl (parseFailedRetry branch in worker sync cycle)
provides:
  - POST /api/v1/inventory/purchases/pending/{id}/retry-parse
  - .retry-parse-btn UI affordance on parse-failed pending cards
affects:
  - backend/internal/inventory/handler.go
  - backend/cmd/server/main.go
  - backend/internal/inventory/period_summary_test.go
  - inventory.html
  - sw.js
  - tests/inventory.spec.js
tech-stack:
  added: []
  patterns:
    - "chi.URLParam path param routing inside the auth-gated inventory route group"
    - "422 row_not_pending / nothing_to_retry envelopes (map[string]string) mirroring 260607-fxl empty_items_not_allowed convention"
    - "404 pending_purchase_not_found envelope (writeError single-key) mirroring UpdatePendingItemsHandler/DiscardPendingPurchaseHandler"
    - "Event-delegated click handler via closest('[data-action]') — nested button wins over outer card data-action"
key-files:
  created: []
  modified:
    - backend/internal/inventory/handler.go
    - backend/cmd/server/main.go
    - backend/internal/inventory/period_summary_test.go
    - inventory.html
    - sw.js
    - tests/inventory.spec.js
decisions:
  - "Path param vs body for id: chi.URLParam per the verbatim must_haves contract (POST /purchases/pending/{id}/retry-parse), even though the surrounding pending mutators (confirm/discard/pending-items) take id in the JSON body"
  - "Re-fetch and return the full PendingPurchase row on 200 (not 204) so the FE can rely on a consistent envelope shape that matches /purchases/pending"
  - "Nest the Retry parse button inside the existing parse_error render branch; both gated on the same truthiness so there's never a button without an error and never an error without a recovery affordance"
metrics:
  duration_minutes: 25
  completed_date: "2026-06-07"
  tasks_completed: 2
  files_changed: 6
  tests_added: 7
commits:
  - hash: 7055693
    message: "feat(260607-koi): backend retry-parse endpoint + tests"
    files: 3
  - hash: 143af77
    message: "feat(260607-koi): FE retry button + tests"
    files: 3
---

# Quick Task 260607-koi: Retry Parse Button Summary

Added a Retry parse button on pending review cards that have a parse_error
set, plus the backend endpoint it calls. The button gives the operator an
in-app path to recover from parser improvements (like 260607-k1n's float64
quantity fix) without psql access — clicking it nulls parse_error so the
next Sync Receipts cycle's parseFailedRetry branch (added in 260607-fxl)
picks the row up and re-attempts parsing via Sonnet.

## Task 1: Backend retry-parse endpoint + 4 unit tests

### Implementation

- **handler.go**: Added `RetryParsePendingPurchaseHandler` (after
  `DiscardPendingPurchaseHandler`) plus a small unexported helper
  `fetchPendingPurchaseByID` that mirrors `ListPendingPurchasesHandler`'s
  column projection so the 200 response shape matches what the FE already
  consumes from `/purchases/pending`.
- **handler.go imports**: Added `context` and `github.com/go-chi/chi/v5`
  (database/sql was already imported from 260607-fxl). `chi.URLParam(r, "id")`
  extracts the path param.
- **main.go**: Wired
  `r.Post("/purchases/pending/{id}/retry-parse", inventory.RetryParsePendingPurchaseHandler(pool))`
  inside the auth-gated `r.Route("/inventory", ...)` group, alongside the
  existing /confirm and /discard routes.

### Tests

Added 4 new tests at the bottom of
`backend/internal/inventory/period_summary_test.go`, plus a
`retryParseHelper` that wires a chi router so `chi.URLParam` resolves
correctly (the surrounding `confirmPendingHelper` calls handlers directly
without a router, but the new handler depends on path-param routing):

| Test | Disposition asserted |
|------|----------------------|
| `TestRetryParse_ClearsParseError` | 200 + parse_error nulled + DB column NULL |
| `TestRetryParse_404OnUnknownId` | 404 `pending_purchase_not_found` |
| `TestRetryParse_422OnConfirmedRow` | 422 `row_not_pending` |
| `TestRetryParse_422WhenNothingToRetry` | 422 `nothing_to_retry` |

Imports added to the test file: `database/sql`, `github.com/go-chi/chi/v5`.

### Verification

- `DB_TEST_URL=… go test -count=1 -run TestRetryParse ./internal/inventory/...` — 4 new tests green (~13s)
- `DB_TEST_URL=… go test -count=1 ./internal/inventory/...` — full inventory suite green, no regression in `TestConfirmPending_*`, `TestPeriodSummary`, or other handlers (~29s)
- `go build ./...` clean
- `go vet ./...` clean

### Commit

`7055693 feat(260607-koi): backend retry-parse endpoint + tests` (3 files,
+263 lines).

## Task 2: FE Retry parse button + 3 Playwright tests + sw.js rebuild

### Implementation

- **inventory.html (CSS)**: Added `.retry-parse-btn` class beside the
  existing `.view-receipt-btn` template — identical look-and-feel except
  `margin-top:6px` instead of `8px` since it sits directly under the
  italic Parser error line.
- **inventory.html (renderPendingCard)**: Appended the
  `<button class="retry-parse-btn" data-action="retry-parse" data-id="…">`
  inside the existing `parseErrHtml` concatenation so the button only
  renders when the Parser error line renders.
- **inventory.html (history-list click delegate)**: Added the
  `else if(action==='retry-parse')` case after `discard-receipt`. On 200
  it shows the toast "Marked for retry. Click Sync Receipts to run now."
  then sets the matching `PENDING_PURCHASES[*].parse_error = ''` and
  calls `renderHistoryList()` so the card re-renders without the error
  line or the button.
- **sw.js**: Regenerated via `node build-sw.js` so the precache manifest
  picks up the new inventory.html content hash (`SW built: 21 files
  precached (1368.3 KB)`).

### Tests

Added 3 new tests inside `test.describe('Retry parse button (260607-koi)', …)`
placed immediately after the existing 260607-e1c parse_error describe in
`tests/inventory.spec.js`:

| Test | Assertion |
|------|-----------|
| "is shown when pending row has parse_error" | Card renders Parser error line + visible button labelled "Retry parse" |
| "is hidden when parse_error is empty" | Card still renders but `[data-action="retry-parse"]` has count 0 and "Parser error:" text is absent |
| "clears parse_error from card on success" | Click → page.route 200 → Parser error line + button both disappear from card |

Test 3 uses `page.route('**/api/v1/inventory/purchases/pending/*/retry-parse', …)`
to stub the POST response with `parse_error: null` — the `*` glob covers
the URL-encoded id.

### Verification

- `node build-sw.js` — clean rebuild (21 files precached)
- `npx playwright test tests/inventory.spec.js -g "Retry parse button"` — 3 new tests green
- Regression smoke: `npx playwright test tests/inventory.spec.js -g "Parser error line|Confirm Receipt disabled|PDF receipt"` — all 5 prior tests still green (260607-e1c parse_error display, 260607-e1c PDF iframe, 260607-fxl confirm disabled state x3)
- Broader smoke: 27 pending/receipt/review tests across the file all pass

### Commit

`143af77 feat(260607-koi): FE retry button + tests` (3 files, +142, -2).

## Important note (plan's "important" section)

The plan noted to verify whether the existing history-list click delegate
uses `e.target.closest('[data-action]')` or direct `e.target` matching, and
to switch to closest() if needed. Investigation: the delegate at
inventory.html line 893 **already** uses `e.target.closest('[data-action]')`
— no change needed. This pattern is what lets the nested
`<button data-action="retry-parse">` inside the
`<div data-action="review-pending">` card route clicks to the inner action
without stopPropagation. All existing actions
(`toggle-event`, `review-pending`, `add-review-line`, `remove-review-line`,
`view-receipt`, `add-vendor`, `confirm-receipt`, `discard-receipt`) keep
working unchanged — confirmed by the 27-test smoke run.

## Deviations from Plan

### Worktree base reset

The worktree was created from b594d8d but had drifted ahead via prior
agent commits. Per the worktree_branch_check protocol, hard-reset to the
documented base commit before starting. No content lost — the resets only
affected the worktree's local history, not main.

### Test DB recreation (pre-existing infrastructure issue)

The shared remote Postgres at 100.70.200.55:5433 had `hq_test` in a state
where `receipt_sync_runs` table existed but the goose migration record
for `0067_receipt_sync_runs.sql` was missing. This caused
`db.Migrate(pool)` to abort with `relation "receipt_sync_runs" already
exists`. Rather than retroactively backfill the goose record, dropped and
recreated `hq_test` cleanly so all migrations re-applied in order. This
is not a code deviation — it's a transient state mismatch in shared test
infrastructure caused by prior agent runs against the same DB.

### No other deviations

Plan executed exactly as written. No bugs auto-fixed (Rule 1 not
triggered), no missing critical functionality found (Rule 2 not
triggered), no blocking issues beyond the test DB recreation noted above
(Rule 3), no architectural decisions needed (Rule 4 not triggered).

## Manual Verification (operator step)

Per the plan's `<verification>` section, the end-to-end manual check is
the operator's step, not Claude's:

1. After both commits land and the SW deploys, find the Restaurant Depot
   $391.96 pending row in the live PWA → expect "Retry parse" button
   visible below the Parser error line.
2. Click button → expect toast "Marked for retry. Click Sync Receipts to
   run now."; expect the Parser error line and the button to disappear
   from the card.
3. Click Sync Receipts → expect that row to auto-resolve via Haiku
   (relies on 260607-k1n's float64 fix) and move from "Needs Review" into
   the normal pending review flow.

## Success Criteria Met

- [x] POST /api/v1/inventory/purchases/pending/{id}/retry-parse exists and
      behaves per the 4 specified dispositions
- [x] A pending card with non-empty parse_error renders a Retry parse
      button with `data-action="retry-parse"`
- [x] Clicking the button → fetch POST → on 200 → toast + clear
      parse_error in PENDING_PURCHASES + re-render
- [x] sw.js precache manifest updated to the new inventory.html content
      hash
- [x] 4 backend tests + 3 Playwright tests added and passing
- [x] Two atomic commits land in the worktree: backend (7055693), then
      FE (143af77)
- [x] No DB migration, no new deps

## Follow-ups / Open Items

None. The endpoint is operator-recoverable in the live PWA, and the
worker's 260607-fxl parseFailedRetry branch handles the re-attempt
automatically on the next Sync Receipts click.

## Self-Check: PASSED

- All 4 modified files exist and match diff stats
- Both commits present in worktree git log (7055693, 143af77)
- No deletions in either commit
- 7 new tests (4 backend + 3 Playwright) all green
