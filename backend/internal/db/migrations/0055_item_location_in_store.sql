-- +goose Up
ALTER TABLE purchase_items ADD COLUMN IF NOT EXISTS location_in_store TEXT;

-- +goose Down
ALTER TABLE purchase_items DROP COLUMN IF EXISTS location_in_store;
