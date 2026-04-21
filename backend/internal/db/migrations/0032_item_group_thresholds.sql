-- +goose Up
BEGIN;

ALTER TABLE item_groups ADD COLUMN low_threshold INTEGER NOT NULL DEFAULT 3;
ALTER TABLE item_groups ADD COLUMN high_threshold INTEGER NOT NULL DEFAULT 10;

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE item_groups DROP COLUMN IF EXISTS low_threshold;
ALTER TABLE item_groups DROP COLUMN IF EXISTS high_threshold;

COMMIT;
