-- +goose Up
BEGIN;

-- Drop existing CHECK constraint on notification_channel.
-- The column was created in migration 0041 with:
--   notification_channel TEXT NOT NULL DEFAULT 'zoho_cliq' CHECK (notification_channel IN ('zoho_cliq', 'email'))
-- Postgres CHECK constraints must be dropped by name; auto-generated name follows pattern.
-- Use ALTER TABLE ... DROP CONSTRAINT ... IF EXISTS to handle environments where the name varies.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_notification_channel_check;

-- Drop the existing default before type change (TEXT default can't auto-cast to TEXT[]).
ALTER TABLE users
  ALTER COLUMN notification_channel DROP DEFAULT;

-- Convert notification_channel from TEXT to TEXT[] (array of channels).
-- Existing single-channel rows become single-element arrays.
ALTER TABLE users
  ALTER COLUMN notification_channel TYPE TEXT[]
  USING ARRAY[notification_channel]::TEXT[];

-- Set default to single-element Zoho Cliq array.
ALTER TABLE users
  ALTER COLUMN notification_channel SET DEFAULT ARRAY['zoho_cliq']::TEXT[];

-- Enforce: array must be a subset of valid channel values.
ALTER TABLE users
  ADD CONSTRAINT users_notification_channel_valid
  CHECK (notification_channel <@ ARRAY['zoho_cliq','email']::TEXT[]);

-- Enforce: at least one channel must be selected.
ALTER TABLE users
  ADD CONSTRAINT users_notification_channel_nonempty
  CHECK (array_length(notification_channel, 1) >= 1);

COMMIT;

-- +goose Down
BEGIN;

-- Drop the new constraints.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_notification_channel_valid;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_notification_channel_nonempty;

-- Convert TEXT[] back to TEXT, taking the first element.
ALTER TABLE users
  ALTER COLUMN notification_channel TYPE TEXT
  USING notification_channel[1];

-- Restore original default and CHECK constraint.
ALTER TABLE users
  ALTER COLUMN notification_channel SET DEFAULT 'zoho_cliq';

ALTER TABLE users
  ADD CONSTRAINT users_notification_channel_check
  CHECK (notification_channel IN ('zoho_cliq', 'email'));

COMMIT;
