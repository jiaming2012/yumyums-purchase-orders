-- +goose Up
BEGIN;
ALTER TABLE purchase_items ADD COLUMN full_name TEXT;
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE purchase_items DROP COLUMN IF EXISTS full_name;
COMMIT;
