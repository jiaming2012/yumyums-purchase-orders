---
id: 260606-hew
slug: vendor-fallback-bank-description
date: 2026-06-06
status: in_progress
---

# Persist Mercury BankDescription as vendor on `no_attachment_on_bank_tx` pendings

## Problem

Every `no_attachment_on_bank_tx` row in the Purchases tab renders as
"Unknown Vendor" even though Mercury attaches a `bankDescription` to
every card transaction (the cleaned merchant string, e.g.
`RESTAURANT DEPOT 0123 CHICAGO IL`).

Root cause — `backend/internal/receipt/worker.go:112-124`:

```go
if len(tx.Attachments) == 0 {
    insertPendingPurchase(ctx, cfg.Pool, tx,
        nil,
        ReceiptSummary{},   // empty: summary.Vendor == ""
        "",
        "no_attachment_on_bank_tx")
}
```

`insertPendingPurchase` writes `summary.Vendor` into
`pending_purchases.vendor` (worker.go:348). Empty string →
`nullableString` → NULL → frontend (`inventory.html:453`) falls back to
"Unknown Vendor".

The receipt-parsed branch works because Claude fills `summary.Vendor`
from the image; the no-attachment branch has no receipt, but
`tx.BankDescription` is right there and never makes it into the row.

## Fix

Two changes in `backend/internal/receipt/worker.go`:

### A) Vendor fallback for new pendings

In `insertPendingPurchase`, when `summary.Vendor` is empty, fall back to
`tx.BankDescription`. This covers:
- The no-attachment branch (always empty summary).
- Any future receipt-parse where Claude fails to extract a vendor.

The receipt-parsed branch is unaffected when Claude successfully
returns a vendor name (the curated name wins over the raw bank string).

### B) Backfill via re-poll UPDATE

Alongside the existing `mercury_category` refresh block (worker.go:80-95)
— which runs every poll for cached transactions — add a parallel UPDATE
for `pending_purchases.vendor`:

```sql
UPDATE pending_purchases
SET vendor = $1
WHERE bank_tx_id = $2
  AND (vendor IS NULL OR vendor = '')
```

- Runs before the `already` short-circuit, so cached rows get touched.
- The `IS NULL OR = ''` guard means a receipt-parsed pending whose
  vendor was already set by Claude is never overwritten.
- 14-day default lookback (`backend/cmd/server/main.go:526`) covers
  every row in the current "Unknown Vendor" backlog from the screenshot
  (May 27 → Jun 5).
- Older-than-14-day pendings won't auto-backfill; if any exist, raise
  the env-var lookback for one cycle.

No separate one-shot migration — the next worker poll does it.

## Files

| File | Change |
|---|---|
| `backend/internal/receipt/worker.go` | (A) BankDescription fallback inside `insertPendingPurchase`. (B) New UPDATE block alongside the mercury_category refresh. |
| `backend/internal/receipt/worker_test.go` | Extend no-attachment test to assert `vendor == tx.BankDescription`. Add re-poll backfill test (insert NULL-vendor row, run worker, assert backfill; insert receipt-parsed vendor row, run worker, assert NOT overwritten). |

No schema migration — `pending_purchases.vendor` already exists as
nullable text.

## Out of scope

- `purchase_events.vendor_id` rows. They go through `createPurchaseEvent`
  which upserts into the `vendors` table by name. Different surface; the
  screenshot is pending_purchases only ("Needs Review" pill).
- Normalizing Mercury's UPPERCASE bank strings. The frontend already
  applies `titleCase()` at render. If raw uppercase still bleeds through
  in places, fix display-side separately.
- Backfilling pendings older than the worker's lookback window.
