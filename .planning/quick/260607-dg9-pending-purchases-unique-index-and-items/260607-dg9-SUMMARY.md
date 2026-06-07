---
phase: 260607-dg9
plan: 01
subsystem: receipt-ingest
tags: [backend, postgres, migration, receipt-worker, idempotency, bug-fix]
status: shipped
requires:
  - 260607-co0  # builds on updatePendingPurchase introduced there (same items-nil fix applied)
provides:
  - pending-purchases-bank-tx-uniq
  - items-nil-guard
decisions:
  - Partial unique index (WHERE confirmed_at IS NULL AND discarded_at IS NULL) keeps the original non-unique bank_tx_id btree index from 0025 in place — the partial index covers active rows only, the btree serves classifyExistingTx + backfillPendingVendor lookups across all states.
  - Dedupe-then-index sequence (CTE DELETE → CREATE UNIQUE INDEX) inside a single BEGIN/COMMIT so the migration is atomic; if dedupe fails the index is never created.
  - Dedupe tiebreak is ORDER BY created_at DESC, id DESC — newer rows reflect later worker observations, so they win.
  - Reason preservation tested explicitly (TestInsertPendingPurchase_DedupesOnReinsertWithDifferentReason) to guard against a future regression that swaps DO NOTHING for DO UPDATE.
  - feat() commit prefix used for Task 2 (not fix) because the schema change is the primary artifact — matches project convention seen in feat(260606-jvs).
key-files:
  created:
    - backend/internal/db/migrations/0068_pending_purchases_bank_tx_uniq.sql
  modified:
    - backend/internal/receipt/worker.go
    - backend/internal/receipt/worker_test.go
metrics:
  duration_min: ~12
  tasks_completed: 2
  files_changed: 3
  commits: 2
  tests_added: 2
  tests_red_to_green: 2
  completed_date: 2026-06-07
---

# 260607-dg9: pending_purchases unique index + items=[] guard

Quick task closing two pre-existing bugs in the receipt worker's pending_purchases insert path. Both already had failing tests at base; this plan makes them pass and ships two new regression tests proving the surrounding contracts.

## What Shipped

**Task 1 — items=nil guard (BUG 1)**
- `insertPendingPurchase` and `updatePendingPurchase` now explicitly check `items == nil` and write the JSON literal `[]` instead of `json.Marshal(nil)`'s `null`.
- Stops the FE from receiving `items: null` on no_attachment_on_bank_tx rows (and on parse-failure UPDATE paths from 260607-co0). Any FE consumer that runs `.length` or `.map` on the field is now safe.

**Task 2 — partial unique index on bank_tx_id (BUG 2)**
- New migration `0068_pending_purchases_bank_tx_uniq.sql` that:
  1. Dedupes any active duplicates already in the table (CTE keeps latest `created_at` per `bank_tx_id`, deletes the rest).
  2. Creates `pending_purchases_bank_tx_id_uniq` — a partial UNIQUE INDEX on `bank_tx_id` WHERE `confirmed_at IS NULL AND discarded_at IS NULL`.
- `insertPendingPurchase` ON CONFLICT clause updated to target the new partial index explicitly: `ON CONFLICT (bank_tx_id) WHERE confirmed_at IS NULL AND discarded_at IS NULL DO NOTHING`.
- Re-polls of the same unreceipted Mercury card swipe are now idempotent at the SQL layer (previously, the bare `ON CONFLICT DO NOTHING` was a no-op because no unique constraint existed to fire on).
- `updatePendingPurchase` left untouched — UPDATE has no ON CONFLICT path; nothing to fix.
- The non-unique `pending_purchases_bank_tx_id_idx` btree from 0025 is left in place — it still serves `classifyExistingTx` + `backfillPendingVendor` lookups across all states.

## Files Changed

| File | Change | Key lines |
|------|--------|-----------|
| `backend/internal/receipt/worker.go` | items==nil guard in `insertPendingPurchase` (~L487-505) and `updatePendingPurchase` (~L556-572); `ON CONFLICT` clause updated (~L527) | +17 / -7 |
| `backend/internal/receipt/worker_test.go` | 2 new tests appended at end of file | +173 / -0 |
| `backend/internal/db/migrations/0068_pending_purchases_bank_tx_uniq.sql` | NEW — dedupe CTE + partial unique index, with Down that drops the index | +42 |

## Commits

| # | Hash | Type | Message |
|---|------|------|---------|
| 1 | `28d98c2` | fix | `fix(260607-dg9): write items=[] not items=null for nil receipt items` |
| 2 | `e5353d5` | feat | `feat(260607-dg9): partial-unique index on pending_purchases.bank_tx_id + ON CONFLICT target` |

Full hashes:
- `28d98c236ae0d78de1f939845a57b5846ed258f7`
- `e5353d5dcc767e763543539df72bb72cc9259fb5`

## Test Results

### Pre-existing failing tests — RED → GREEN

| Test | Before | After | Made green by |
|------|--------|-------|---------------|
| `TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults` | FAIL (`items = "null", want "[]"`) | PASS | Task 1 |
| `TestInsertPendingPurchase_NoAttachmentBranch_IdempotentOnRerun` | FAIL (`rows = 2, want 1`) | PASS | Task 2 |

### New regression tests — both PASS

| Test | Asserts |
|------|---------|
| `TestInsertPendingPurchase_DedupesOnReinsertWithDifferentReason` | Second insert with different `reason` is silently absorbed by the partial unique index and the **original** reason is preserved (guards against accidentally switching to DO UPDATE). |
| `TestMigration_DedupesExistingPendingDuplicates` | Drops index → seeds duplicates → re-runs dedupe CTE → re-creates index → asserts: exactly 1 row remains; it's the row with the **latest** `created_at`; ON CONFLICT DO NOTHING blocks a fresh dup; discarded rows DON'T conflict (partial predicate excludes them). |

### Full receipt package suite — no regressions

```
DB_TEST_URL=... go test ./internal/receipt/ -count=1 -v
```

15 tests + 5 sub-tests under `TestClassifyExistingTx` — all PASS. Notably, the two 260607-co0 upgrade-flow tests still pass:
- `TestRunIngestCycle_UpgradesPendingNoAttachmentRow` (DELETE+INSERT upgrade inside one tx)
- `TestRunIngestCycle_UpgradesPendingNoAttachmentRow_ParseFails` (in-place updatePendingPurchase)

The DELETE-then-INSERT pattern inside `createPurchaseEvent` runs the pending DELETE FIRST inside the tx, so the partial unique index has nothing to conflict against by the time the next insert happens.

### Build + vet

- `go build ./...` — clean
- `go vet ./...` — clean

## Acceptance Criteria Verification

- [x] `TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults` passes
- [x] `TestInsertPendingPurchase_NoAttachmentBranch_IdempotentOnRerun` passes
- [x] `TestInsertPendingPurchase_DedupesOnReinsertWithDifferentReason` passes
- [x] `TestMigration_DedupesExistingPendingDuplicates` passes
- [x] All previously-green receipt tests still pass — confirmed no regressions in: `TestInsertPendingPurchase_VendorFallback_PrefersSummary`, `TestInsertPendingPurchase_CoexistsWithAttachmentBranch`, `TestBackfillPendingVendor_*` (3 tests), `TestClassifyExistingTx` (+ 5 sub-tests), `TestRunIngestCycle_*` (5 tests including both Upgrades from 260607-co0)
- [x] `go build ./...` and `go vet ./...` clean
- [x] Migration 0068 present; idempotent on second goose Up (dedupe CTE removes zero rows, IF NOT EXISTS guards index creation)
- [x] 2 atomic commits on `dev`: `fix(260607-dg9):` + `feat(260607-dg9):`
- [x] No FE changes; no `sw.js` regen; no Playwright tests run
- [x] Out-of-scope items not touched: `insertPendingPurchase` signature unchanged, 260607-co0 upgrade flow unchanged, no backfill of historical confirmed/discarded rows, `LIMIT 1` in `classifyExistingTx` left as defensive

## Deviations from Plan

**None.** Plan executed exactly as written:
- Migration SQL matches the plan's SQL verbatim.
- ON CONFLICT clause matches plan's specified text verbatim.
- Both new tests added at end of `worker_test.go` (after the last existing test `TestClassifyExistingTx`, separated by a section comment header).
- Commit prefixes (`fix` for Task 1, `feat` for Task 2) match plan.
- Verified at base commit `cb1b68c` that the two target tests fail (RED state); after each task confirmed the targeted test goes GREEN.

## Closes 260607-co0 Deferred Items

Both deferred items from `260607-co0/deferred-items.md` are now resolved:

1. **items=null on no_attachment rows** — fixed by Task 1's items==nil guard (applied to both insert and update paths, including the updatePendingPurchase introduced in 260607-co0).
2. **No SQL-level idempotency on pending_purchases.bank_tx_id** — fixed by Task 2's partial unique index + targeted ON CONFLICT clause.

## Self-Check: PASSED

**Created files exist:**
- FOUND: `backend/internal/db/migrations/0068_pending_purchases_bank_tx_uniq.sql`

**Modified files exist:**
- FOUND: `backend/internal/receipt/worker.go` (verified items==nil guard + new ON CONFLICT clause present)
- FOUND: `backend/internal/receipt/worker_test.go` (verified 2 new tests present)

**Commits exist:**
- FOUND: `28d98c2` — fix(260607-dg9): write items=[] not items=null for nil receipt items
- FOUND: `e5353d5` — feat(260607-dg9): partial-unique index on pending_purchases.bank_tx_id + ON CONFLICT target

**Tests pass:**
- FOUND in test output: TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults — PASS
- FOUND in test output: TestInsertPendingPurchase_NoAttachmentBranch_IdempotentOnRerun — PASS
- FOUND in test output: TestInsertPendingPurchase_DedupesOnReinsertWithDifferentReason — PASS
- FOUND in test output: TestMigration_DedupesExistingPendingDuplicates — PASS
- FOUND in test output: TestRunIngestCycle_UpgradesPendingNoAttachmentRow — PASS (260607-co0 regression check)
- FOUND in test output: TestRunIngestCycle_UpgradesPendingNoAttachmentRow_ParseFails — PASS (260607-co0 regression check)
