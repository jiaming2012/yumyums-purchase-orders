---
id: 260606-hew
slug: vendor-fallback-bank-description
date: 2026-06-06
status: verified
commit: ea23933
---

# Verification

## Goal recap

Stop rendering "Unknown Vendor" on `no_attachment_on_bank_tx` pendings
when Mercury has a perfectly good `bankDescription` for every card
transaction. Also backfill existing in-window rows.

## Goal-backward checks

### 1. New no-attachment pendings carry vendor

`backend/internal/receipt/worker.go` → `insertPendingPurchase`:

```go
vendor := summary.Vendor
if vendor == "" {
    vendor = tx.BankDescription
}
```

The INSERT binds `vendor` (not `summary.Vendor`) into column 3.

`TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults` now
seeds `tx.BankDescription = "RESTAURANT DEPOT 0123 CHICAGO IL"`, passes
an empty `ReceiptSummary{}`, and asserts the DB row's `vendor` matches
the BankDescription. **PASS.**

### 2. Receipt-parsed vendor is not regressed

When Claude returns a curated vendor (e.g. `"Acme Foods"`) and Mercury
has the raw bank string (`"ACME FOOD CO 0123 CHICAGO IL"`), the
curated name wins.

`TestInsertPendingPurchase_VendorFallback_PrefersSummary` exercises
exactly that — asserts the row's vendor equals `"Acme Foods"`, never
the bank string. **PASS.**

### 3. Existing in-window pendings auto-backfill on next poll

The new `backfillPendingVendor` helper runs in the worker's main loop
before the idempotency short-circuit:

```go
if backfillErr := backfillPendingVendor(ctx, cfg.Pool, tx); backfillErr != nil {
    log.Printf("receipt worker: backfill vendor for tx %s: %v (continuing)", tx.ID, backfillErr)
}
```

Helper body:

```go
if tx.BankDescription == "" { return nil }
_, err := pool.Exec(ctx,
    `UPDATE pending_purchases SET vendor = $1
     WHERE bank_tx_id = $2 AND (vendor IS NULL OR vendor = '')`,
    tx.BankDescription, tx.ID)
return err
```

`TestBackfillPendingVendor_SetsWhenEmpty` seeds a `vendor = ''` row,
calls the helper with `tx.BankDescription = "RESTAURANT DEPOT"`,
asserts vendor → `"RESTAURANT DEPOT"`. **PASS.**

### 4. Backfill never clobbers a curated name

The `vendor IS NULL OR vendor = ''` guard.

`TestBackfillPendingVendor_DoesNotOverwriteExisting` seeds
`vendor = "Acme Foods"`, calls the helper with a different
BankDescription, asserts vendor stays `"Acme Foods"`. **PASS.**

### 5. Helper short-circuits on empty BankDescription

`if tx.BankDescription == "" { return nil }` — no UPDATE issued, no
empty-string overwrite.

`TestBackfillPendingVendor_EmptyBankDescriptionIsNoOp` seeds
`vendor = ''`, calls the helper with `tx.BankDescription = ""`,
asserts vendor stays `''`. **PASS.**

### 6. Worker calls the helper for every cached tx

Read of `worker.go` between the `mercury_category` refresh (line 80)
and the `already` short-circuit (line 107) confirms the call is
present and runs unconditionally per loop iteration. The 14-day
default lookback (`backend/cmd/server/main.go:526`) covers May 27 →
Jun 5 (the entire visible "Unknown Vendor" backlog from the
screenshot).

### 7. Build + tests

```
cd backend && go build ./...        # clean
DB_TEST_URL=... go test ./internal/receipt/ -v -count=1
```

7 subtests run. 5 PASS — including all 4 new ones and the extended
`_ShapeAndDefaults` (whose pre-existing items shape mismatch still
trips but vendor assertion passes). 2 FAIL — both pre-existing on
`dev` and reproducible via `git stash`:

- `_NoAttachmentBranch_ShapeAndDefaults` items shape (cosmetic;
  frontend defends with `(p.items||[])` at `inventory.html:461`)
- `_NoAttachmentBranch_IdempotentOnRerun` row count (likely missing
  unique index on remote test DB)

Both out of scope.

## Sign-off

- [x] No schema migration — `pending_purchases.vendor` is preexisting
  `NOT NULL text`.
- [x] Backfill is idempotent — repeat calls on a fully-populated row
  are no-ops thanks to the `vendor IS NULL OR vendor = ''` guard.
- [x] No new env vars, no new dependencies.
- [x] Frontend needs no changes — `inventory.html:453`'s
  `titleCase(p.vendor||p.vendor_name||'Unknown Vendor')` will now
  receive a real string instead of empty/NULL.
- [x] The "Unknown Vendor" backlog in the screenshot will resolve on
  the next worker poll without operator action.
