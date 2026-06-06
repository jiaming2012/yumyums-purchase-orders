-- +goose Up
BEGIN;

-- Mercury's categoryData.name at the time HQ ingested the receipt.
-- Nullable because: (a) existing rows pre-date this column, (b) future
-- rows where Mercury hasn't been categorized yet by sales-processor's
-- classify pipeline. The receipt worker re-syncs the column on every
-- poll for events within its lookback window (14 days), so NULLs
-- self-heal as Mercury catches up.
ALTER TABLE purchase_events
  ADD COLUMN mercury_category TEXT;

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE purchase_events
  DROP COLUMN mercury_category;

COMMIT;
