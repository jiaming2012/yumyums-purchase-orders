-- +goose Up
BEGIN;
ALTER TABLE purchase_items ADD COLUMN photo_url TEXT;
ALTER TABLE purchase_items ADD COLUMN store_location TEXT;
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE purchase_items DROP COLUMN IF EXISTS store_location;
ALTER TABLE purchase_items DROP COLUMN IF EXISTS photo_url;
COMMIT;
