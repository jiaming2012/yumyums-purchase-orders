# Deferred Items — 260607-co0

These were observed during execution but are OUT OF SCOPE for this quick task. They are pre-existing failures, verified to fail at the plan's base commit (9415f22) before any 260607-co0 changes were applied. Documenting here so they aren't lost.

## 1. `TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults` fails — `items = "null", want "[]"`

- **Where:** `backend/internal/receipt/worker_test.go` line 100
- **Root cause:** `insertPendingPurchase` calls `json.Marshal(items)` where `items` is `nil`. `json.Marshal(nil)` returns the JSON literal `null`, not `[]`. The bytes only fall back to `[]byte("[]")` when Marshal errors — and Marshal succeeds on nil.
- **Fix sketch:** In `insertPendingPurchase`, guard `if items == nil { itemsJSON = []byte("[]") }` before the Marshal call. Same fix needed in `updatePendingPurchase`.
- **Impact:** The DB persists the JSON literal `null` for the items column on the no-attachment branch instead of `[]`. Frontend may need to defensively coerce.

## 2. `TestInsertPendingPurchase_NoAttachmentBranch_IdempotentOnRerun` fails — `pending_purchases rows = 2, want 1`

- **Where:** `backend/internal/receipt/worker_test.go` line 293
- **Root cause:** `pending_purchases.bank_tx_id` has only a btree INDEX (`pending_purchases_bank_tx_id_idx` in `migrations/0025_pending_purchases.sql`), not a UNIQUE constraint. `insertPendingPurchase` uses `ON CONFLICT DO NOTHING` with no target — that clause only suppresses errors on a constraint violation, and there is no unique constraint to violate. So a second insert with the same `bank_tx_id` creates a duplicate row.
- **Fix sketch:** New migration adding `CREATE UNIQUE INDEX pending_purchases_bank_tx_id_uniq ON pending_purchases(bank_tx_id) WHERE confirmed_at IS NULL AND discarded_at IS NULL;` (partial unique so historical confirmed/discarded rows don't conflict). Plus changing `ON CONFLICT DO NOTHING` → `ON CONFLICT (bank_tx_id) WHERE confirmed_at IS NULL AND discarded_at IS NULL DO NOTHING`.
- **Impact:** In production, repeated polls of the same unreceipted card swipe in the 14-day lookback window create duplicate pending rows. The 260607-co0 upgrade flow does not touch this — the classify helper just uses `LIMIT 1`, so it picks one of the duplicates and upgrades it; the other(s) stick around forever as orphans. Worth a follow-up phase.
