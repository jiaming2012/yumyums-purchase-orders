-- +goose Up
BEGIN;

-- Add notification_channel preference per user (D-05: zoho_cliq default, email alternative).
-- At least one channel is required; frontend enforces no-disable rule (D-06).
ALTER TABLE users
  ADD COLUMN notification_channel TEXT NOT NULL DEFAULT 'zoho_cliq'
    CHECK (notification_channel IN ('zoho_cliq', 'email'));

COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE users DROP COLUMN IF EXISTS notification_channel;
COMMIT;
