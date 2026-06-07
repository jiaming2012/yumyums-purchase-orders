-- +goose Up
BEGIN;

-- Step 1: Dedupe existing active pending rows. For each bank_tx_id with
-- multiple rows where confirmed_at IS NULL AND discarded_at IS NULL, keep
-- the row with the latest created_at (tiebreak by id DESC for determinism).
-- This is safe on re-run: after the first apply there are no duplicates,
-- so the DELETE removes zero rows on subsequent goose runs (though goose
-- only runs each migration once anyway -- belt + suspenders).
WITH ranked AS (
  SELECT id,
         ROW_NUMBER() OVER (
           PARTITION BY bank_tx_id
           ORDER BY created_at DESC, id DESC
         ) AS rn
    FROM pending_purchases
   WHERE confirmed_at IS NULL
     AND discarded_at IS NULL
)
DELETE FROM pending_purchases
 WHERE id IN (SELECT id FROM ranked WHERE rn > 1);

-- Step 2: Create the partial unique index. Partial so historical
-- confirmed/discarded rows for the same bank_tx_id (e.g. user discarded then
-- Mercury re-shows the tx -- ingest is allowed) don't conflict with a fresh
-- active pending row. The non-unique pending_purchases_bank_tx_id_idx from
-- migration 0025 is left in place -- it serves classifyExistingTx +
-- backfillPendingVendor lookups across both active and inactive rows.
CREATE UNIQUE INDEX IF NOT EXISTS pending_purchases_bank_tx_id_uniq
  ON pending_purchases(bank_tx_id)
  WHERE confirmed_at IS NULL AND discarded_at IS NULL;

COMMIT;

-- +goose Down
BEGIN;

DROP INDEX IF EXISTS pending_purchases_bank_tx_id_uniq;

COMMIT;
