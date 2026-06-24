-- +goose Up
BEGIN;

-- Adds receipt_urls (JSONB array of strings) to both tables so all
-- attachments from a Mercury transaction are stored, not just the first.
-- NULL on existing rows — the singular receipt_url column is preserved
-- as the backward-compat fallback for rows written before this migration.
ALTER TABLE pending_purchases ADD COLUMN IF NOT EXISTS receipt_urls JSONB;
ALTER TABLE purchase_events   ADD COLUMN IF NOT EXISTS receipt_urls JSONB;

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE pending_purchases DROP COLUMN IF EXISTS receipt_urls;
ALTER TABLE purchase_events   DROP COLUMN IF EXISTS receipt_urls;

COMMIT;
