---
quick_id: 260607-s6r
status: complete
date: 2026-06-07
---

# 260607-s6r — Broaden Retry parse button gate to items-mismatch rows

## What shipped

Follow-up to 260607-koi. Today's production state: a pending row was parsed
successfully by Haiku (items populated, `parse_error=NULL`) but the parsed
receipt content was for an entirely different bank transaction
(Restaurant Depot, $1515.16) than the bank tx it was paired with
(Amazon Mktplace Pmts, $65.62). koi's "Retry parse" button only renders when
`parse_error` is non-empty, so the user had no FE recovery path.

This quick task broadens the same code path (FE button + BE handler) to also
cover rows where items are populated but line_total doesn't match bank_total,
without scope-creeping into the underlying wrong-attachment matching bug
(handled by the next /gsd:debug session).

### T1 — `2a25140` feat(260607-s6r): broaden retry-parse to items-mismatch rows

- `backend/internal/inventory/handler.go` — `RetryParsePendingPurchaseHandler`
  now branches:
  - `parse_error IS NOT NULL` (koi path, unchanged) → clear `parse_error` only
  - else `itemsMismatch` (new s6r path) → set `items='[]'::jsonb` AND set
    `reason='Receipt could not be parsed automatically'` so the worker's
    `parseFailedRetry` gate matches on the next sync
  - else → preserve the existing `422 nothing_to_retry` envelope
- Mismatch check reuses the existing `0.01` epsilon (no new tolerance constant).
- Handler-side log on the new path:
  `log.Printf("RetryParse s6r: row %s re-queued (items-mismatch: line_total=%.2f bank_total=%.2f)", ...)`
- `inventory.html` — `renderPendingCard` computes `itemsMismatch` from
  `p.items` + `p.bank_total` (no dependency on the review form being open) and
  renders the Retry parse button whenever `showRetry`. The button gets
  `data-mismatch="1"` only in the new path; the click handler triggers
  `confirm("Discard parsed items and re-parse from receipt?")` before POSTing.
  koi's parse_error-only path stays zero-friction (no confirm dialog —
  regression-safe).
- `sw.js` — regenerated via `node build-sw.js` to pick up the inventory.html
  changes.

**Sentinel decision documented in PLAN.md:** `items='[]'::jsonb` chosen
(not `NULL`) so it satisfies the `worker.go` `classifyExistingTx` predicate
(`jsonb_typeof(items)='array' AND jsonb_array_length(items)>0`) by failing on
the second conjunct → `hasItems=false`. Matches the 260607-dg9 codebase-wide
convention.

### T2 — `6e592d4` test(260607-s6r): regression for items-mismatch retry path

Two new tests appended to `backend/internal/inventory/period_summary_test.go`
after `TestRetryParse_422WhenNothingToRetry`:

- `TestRetryParse_ItemsMismatch_Accepted` — seeds a row with
  `parse_error=NULL`, `items='[{"name":"x","quantity":1,"price":100,...}]'`,
  `bank_total=50`, `reason='no_attachment_on_bank_tx'`. POSTs to
  `/api/v1/inventory/purchases/pending/{id}/retry-parse`, asserts 200, and
  asserts DB row now has `items='[]'`, `parse_error=NULL`, and
  `reason='Receipt could not be parsed automatically'`.
- `TestRetryParse_ItemsMatchTotals_StillRejected` — locks in the no-scope-creep
  guarantee: a row with items populated AND totals matching still returns
  `422 nothing_to_retry`.

Both reuse the existing `retryParseHelper` and `insertPendingPurchaseFull`
helpers — no duplicate scaffolding.

## Verification

- `go vet ./...` — clean
- `go build ./...` — clean
- All 6 `TestRetryParse*` tests pass (4 existing koi + 2 new s6r) against the
  Tailscale Postgres at `100.70.200.55:5433/hq_test`. Total runtime ~4.1s.
- The s6r log line emitted during the new test:
  `RetryParse s6r: row <uuid> re-queued (items-mismatch: line_total=100.00 bank_total=50.00)`
- No koi regression (all 4 existing tests still green).

## User verification

1. Reload inventory.html → the Amazon Mktplace Pmts row now shows a
   **Retry parse** button next to the line items
2. Tap → confirm dialog → items clear, mismatch banner stays
3. Tap **Sync Receipts** → row gets re-processed by the worker (the parse may
   still pair to the wrong receipt — that's the next /gsd:debug session)
4. Existing rows with `parse_error` populated still show + work the same as
   before (no koi regression)

## Deferred follow-up

The underlying wrong-attachment matching bug (Restaurant Depot PDF paired with
Amazon Mktplace Pmts bank tx) is **not** fixed by s6r — this is the recovery
UX only. Next session: `/gsd:debug` against
`classifyExistingTx` / the co0 upgrade path.

## Commits

- `2a25140` — feat(260607-s6r): broaden retry-parse to items-mismatch rows
- `6e592d4` — test(260607-s6r): regression for items-mismatch retry path
- `7767f40` — chore: merge quick task worktree (worktree-agent-a492f058ad676563a)
