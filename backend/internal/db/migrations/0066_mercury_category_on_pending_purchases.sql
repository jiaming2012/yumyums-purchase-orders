-- +goose Up
BEGIN;

-- Mercury's categoryData.name at the time HQ ingested the bank transaction.
-- Nullable because: (a) existing rows pre-date this column, (b) future rows
-- where Mercury hasn't been categorized yet by the classify pipeline.
-- The receipt worker re-syncs this column on every poll within its 14-day
-- lookback window via UPDATE … IS DISTINCT FROM, so NULLs self-heal as
-- Mercury catches up. /period-summary uses this column to decide whether a
-- pending row blocks payroll: COGS-category + no_attachment_on_bank_tx
-- blocks; everything else either rolls into COGS (food + attached receipt
-- that parse-failed) or stays out of the payroll endpoint entirely
-- (non-food, NULL category). See 260606-jvs-HANDOFF.md.
ALTER TABLE pending_purchases
  ADD COLUMN mercury_category TEXT;

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE pending_purchases
  DROP COLUMN mercury_category;

COMMIT;
