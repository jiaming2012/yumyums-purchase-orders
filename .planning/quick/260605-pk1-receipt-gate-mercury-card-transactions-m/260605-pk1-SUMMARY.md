---
phase: 260605-pk1
plan: 01
subsystem: receipt-ingest + inventory-completeness
status: complete
tags: [receipt-gate, mercury, pending-purchases, completeness, cogs]
dependency_graph:
  requires:
    - "pending_purchases table with bank_tx_id unique index + nullable columns (pre-existing)"
    - "insertPendingPurchase helper with ON CONFLICT DO NOTHING (pre-existing)"
  provides:
    - "Mercury ingest now surfaces every supported sent card/debit-card txn — not just photographed ones"
    - "pending_purchases.reason='no_attachment_on_bank_tx' as the sentinel for unreceipted card spend"
    - "ConfirmPendingPurchaseHandler accepts empty-LineItems resolution (writes purchase_event with abs(bank_total), tax=0, no line items)"
  affects:
    - "completeness.ready (gate now blocks until operator triages every no-attachment row in the window)"
    - "sales-processor weekly payroll (will refuse to render until HQ pending queue is empty)"
tech_stack:
  added: []
  patterns:
    - "sentinel-string reason column to distinguish ingestion routing outcomes"
    - "ON CONFLICT DO NOTHING on bank_tx_id for idempotent re-poll without dedup logic in the worker"
key_files:
  created:
    - backend/internal/receipt/worker_test.go
  modified:
    - backend/internal/receipt/mercury.go
    - backend/internal/receipt/worker.go
    - backend/internal/inventory/handler.go
    - backend/internal/inventory/period_summary_test.go
decisions:
  - "Use literal sentinel 'no_attachment_on_bank_tx' (not an enum) — keeps the DB column free-form and Phase 2 merchant-learning rules can match other reasons later"
  - "Place the no-attachment branch AFTER the bankTxIDExists early-continue so already-ingested rows still increment skippedCached"
  - "Empty-items confirm uses abs(bank_total) for total and forces tax=0 per HANDOFF §3 minimum-change variant — keeps cogs_incl_tax accurate at the txn level, accepts cogs_excl_tax under-counting (operator-acknowledged trade-off)"
  - "New worker_test.go drives insertPendingPurchase + createPurchaseEvent directly (not runIngestCycle end-to-end) because FetchTransactions hits real Mercury HTTP — direct unit-of-behavior testing is cleaner than a mock HTTP server for this MVP"
metrics:
  duration_minutes: 2.75
  completed_date: "2026-06-05"
  files_changed: 5
  commits: 2
---

# Phase 260605-pk1 Plan 01: Receipt-Gate Mercury Card Transactions Summary

One-liner: Mercury ingest now surfaces every supported card/debit-card transaction as a `pending_purchases` row (sentinel `reason='no_attachment_on_bank_tx'`) so the period-summary completeness gate blocks payroll on unreceipted card spend; confirm endpoint also accepts empty-items resolution using `abs(bank_total)` when the operator picks "food, no receipt".

## What Changed

| File | Change |
|---|---|
| `backend/internal/receipt/mercury.go` | Dropped the `len(tx.Attachments) > 0` filter in `FetchTransactions`; updated the function doc comment; attachment classification moves to the worker. |
| `backend/internal/receipt/worker.go` | Removed stale `if len(tx.Attachments) == 0 { continue }` skip; added a new branch that calls `insertPendingPurchase(..., "no_attachment_on_bank_tx")` for unreceipted txns; updated the "no transactions found" log message to reflect the relaxed filter. |
| `backend/internal/inventory/handler.go` | `ConfirmPendingPurchaseHandler` now detects `len(input.LineItems) == 0`, skips the receipt-vs-bank mismatch check, writes the `purchase_events` row with `abs(bank_total)` as total and tax=0, and skips the line-items insert loop. Non-empty path unchanged. |
| `backend/internal/receipt/worker_test.go` (new) | Three integration tests covering the no-attachment branch: row-shape correctness (bank_total, items=`[]`, NULL total, sentinel reason, derived event_date), idempotency on rerun, and coexistence with the parsed-receipt path. |
| `backend/internal/inventory/period_summary_test.go` | Added `insertPendingPurchaseWithReason` helper + subtest `"ready=false when no_attachment_on_bank_tx pending row in range"` asserting the gate blocks and `pending_review_ids` lists the new row. |

## Sentinel String

The exact literal stored in `pending_purchases.reason` for unreceipted card transactions:

```
no_attachment_on_bank_tx
```

Used in: `worker.go` (insertPendingPurchase call), `worker_test.go` (×3 — call + 2 assertions), `period_summary_test.go` (×2 — helper docstring + new subtest).

## Test Results

`cd backend && go test ./internal/receipt/... ./internal/inventory/... -count=1`

| Package | Result | Notes |
|---|---|---|
| `internal/receipt` | PASS (0.60s) | 3 new subtests cleanly SKIP — `DB_TEST_URL` unreachable (expected, no local Postgres in this env) |
| `internal/inventory` | PASS (1.15s) | All existing subtests + new no-attachment subtest cleanly SKIP — `DB_TEST_URL` unreachable |

Pass/skip count:

- receipt package: 3 tests, 3 skipped (DB unreachable), 0 failed
- inventory package: 1 test (`TestPeriodSummary` is the top-level test holding all `t.Run` subtests), 1 skipped (DB unreachable), 0 failed

Per PLAN.md `<verify>` clause: "passes (DB up) or all subtests cleanly skip (DB not reachable)" — the latter applies here, which the plan explicitly accepts.

`go build ./...` is clean.

## Deviations from HANDOFF / PLAN

None. The plan was prescriptive; implementation matches it line-for-line. The only judgement calls (all documented in `decisions` above):

- Sentinel string is a literal, not an enum constant.
- Worker branch placement after the existing `bankTxIDExists` early-continue (PLAN explicitly recommends this to keep `skippedCached` accurate).
- worker_test.go tests the unit of behavior (`insertPendingPurchase` + `createPurchaseEvent`) rather than `runIngestCycle` end-to-end (PLAN's "PREFERRED" approach).

## Verification Block (PLAN.md)

| Check | Result |
|---|---|
| `cd backend && go build ./...` | exit 0 |
| `cd backend && go test ./internal/receipt/... ./internal/inventory/... -count=1` | exit 0 (passes-by-skip) |
| `grep -rn "no_attachment_on_bank_tx"` returns ≥3 hits across worker.go, worker_test.go, period_summary_test.go | 10 hits across the 3 expected files |
| `grep "len(tx.Attachments) > 0" mercury.go` returns nothing | confirmed absent |
| `grep "if len(tx.Attachments) == 0 { continue" worker.go` returns nothing | confirmed absent |

## Commits

| Hash | Message |
|---|---|
| `4fe8dd2` | `feat(receipt-gate): surface unreceipted Mercury card txns + accept empty-items confirm` |
| `470663e` | `test(receipt-gate): cover no-attachment branch + completeness gate` |

## Followup Notes (informational — do not auto-create tasks)

Phase 2 work flagged by the HANDOFF for later consideration (out of scope here):

1. **Merchant learning rules** — table `merchant_dismissal_rules`, operator prompt "Always dismiss txns matching `<bank_description>`?", worker auto-discards matched rows on ingest. Reduces first-run triage pain from "every personal card use" to "real food vendors only". Estimated 2-week stabilization window after rules table ships.
2. **`/pending/{id}/attach-receipt` endpoint** — operator-uploaded receipt re-parses and confirms in one round-trip; current MVP requires the operator to upload via the existing photo flow and then call `/confirm`.
3. **Cursor pagination on `FetchTransactions`** — pagination guard at the top of `FetchTransactions` becomes more likely to trip now that every card swipe counts toward the 1000-row response limit. Loud-error is fine for now; pagination becomes urgent if it starts firing in practice.
4. **Cash purchase capture** — Mercury can't see cash; mitigation is operator discipline (pay with the card). No code change planned.
5. **Split `pending_review_ids` by reason in the sales-processor message** — payroll message will eventually read e.g. "12 receipts failed to parse, 35 card txns with no receipt"; not needed for v1 since operator opens HQ dashboard either way.

## Self-Check: PASSED

- backend/internal/receipt/mercury.go — modified (Edit applied)
- backend/internal/receipt/worker.go — modified (Edit applied)
- backend/internal/inventory/handler.go — modified (Edit applied)
- backend/internal/receipt/worker_test.go — created (Write applied)
- backend/internal/inventory/period_summary_test.go — modified (Edit applied)
- commit `4fe8dd2` — FOUND
- commit `470663e` — FOUND
- `go build ./...` — exit 0
- `go test ./internal/receipt/... ./internal/inventory/...` — exit 0 (skip-by-design when DB_TEST_URL unreachable; plan explicitly allows)
