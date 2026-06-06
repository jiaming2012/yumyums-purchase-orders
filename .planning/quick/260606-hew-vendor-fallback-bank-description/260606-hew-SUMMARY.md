---
id: 260606-hew
slug: vendor-fallback-bank-description
date: 2026-06-06
status: complete
commit: ea23933
---

# Summary

Pending purchases that landed via the `no_attachment_on_bank_tx` branch
no longer render as "Unknown Vendor". The fix has two halves shipped
together in `backend/internal/receipt/worker.go`:

1. **Forward fix** — `insertPendingPurchase` now falls back to
   `tx.BankDescription` when `summary.Vendor` is empty. Every newly
   ingested unreceipted card transaction comes in with its merchant
   string already set.
2. **Backfill** — a new `backfillPendingVendor` helper runs alongside
   the existing `mercury_category` refresh inside the worker's main
   loop, before the idempotency short-circuit. On every poll, for
   every Mercury transaction within the 14-day lookback, any
   `pending_purchases` row with `bank_tx_id` match and an empty/NULL
   `vendor` gets backfilled to `tx.BankDescription`. The
   `vendor IS NULL OR vendor = ''` guard protects rows whose vendor
   the receipt parser already set.

The screenshot's "Unknown Vendor" rows (May 27 → Jun 5) are all within
the lookback window — the next worker poll will backfill them
automatically. No one-shot migration required.

## Files changed

| File | Change |
|---|---|
| `backend/internal/receipt/worker.go` | `insertPendingPurchase` now picks `tx.BankDescription` when `summary.Vendor == ""`. New `backfillPendingVendor(ctx, pool, tx)` helper. The main loop calls the helper before the idempotency check. |
| `backend/internal/receipt/worker_test.go` | Extended `TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults` to assert vendor falls through to BankDescription. New tests: `_VendorFallback_PrefersSummary`, `TestBackfillPendingVendor_SetsWhenEmpty`, `_DoesNotOverwriteExisting`, `_EmptyBankDescriptionIsNoOp`. |

No schema migration — `pending_purchases.vendor` already exists as
`NOT NULL text`.

## Tests added (all PASS)

- `_VendorFallback_PrefersSummary` — Claude-parsed `summary.Vendor =
  "Acme Foods"` beats `tx.BankDescription = "ACME FOOD CO 0123 CHICAGO
  IL"` on insert.
- `TestBackfillPendingVendor_SetsWhenEmpty` — seeded `vendor=''` row +
  `tx.BankDescription="RESTAURANT DEPOT"` → row's vendor becomes
  `"RESTAURANT DEPOT"`.
- `_DoesNotOverwriteExisting` — seeded `vendor="Acme Foods"` row +
  any `tx.BankDescription` → vendor stays `"Acme Foods"`.
- `_EmptyBankDescriptionIsNoOp` — seeded `vendor=''` row +
  `tx.BankDescription=""` → vendor stays `''` (no spurious overwrite).
- Existing `_ShapeAndDefaults` now also asserts the no-att branch
  writes BankDescription.

## Pre-existing failures (unrelated)

`go test ./internal/receipt/ -v -count=1` against the remote test DB
still reports two failures on `dev` independent of this change,
reproducible via `git stash`:

- `TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults`
  → `items = "null", want "[]"`. `json.Marshal(nil)` returns
  `"null"`; the test asserts `"[]"`. The frontend already defends with
  `(p.items||[])` at `inventory.html:461` so this is cosmetic.
- `TestInsertPendingPurchase_NoAttachmentBranch_IdempotentOnRerun`
  → two rows after re-run rather than one. Suggests the
  `ON CONFLICT DO NOTHING` guard isn't catching a duplicate
  bank_tx_id; likely a missing unique index on the remote test DB.

Both are out of scope for this phase.

## Verification path

The next scheduled receipt-worker poll will:
1. Loop through every Mercury tx in the 14-day lookback.
2. Call `backfillPendingVendor` for each — sets
   `pending_purchases.vendor` for any row whose vendor is still empty.
3. Hit the existing `bankTxIDExists` idempotency check and skip the
   insert path for already-ingested rows.

To confirm visually: refresh the Purchases tab after the next poll
(~6h). The May 27 → Jun 5 rows should show "Restaurant Depot",
"Save A Lot", etc. instead of "Unknown Vendor".

To trigger the poll early, kick the worker manually.
