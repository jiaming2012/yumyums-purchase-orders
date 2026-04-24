-- +goose Up
ALTER TABLE users
  ALTER COLUMN notification_channel SET DEFAULT ARRAY['zoho_cliq','email']::TEXT[];

-- +goose Down
ALTER TABLE users
  ALTER COLUMN notification_channel SET DEFAULT ARRAY['zoho_cliq']::TEXT[];
