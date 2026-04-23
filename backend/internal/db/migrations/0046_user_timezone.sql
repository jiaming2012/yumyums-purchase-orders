-- +goose Up
ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'America/New_York';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS timezone;
