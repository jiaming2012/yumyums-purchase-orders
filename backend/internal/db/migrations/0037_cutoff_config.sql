-- +goose Up
BEGIN;

CREATE TABLE cutoff_config (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  day_of_week INTEGER NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),  -- 0=Sunday, 6=Saturday
  cutoff_time TIME NOT NULL,
  timezone    TEXT NOT NULL DEFAULT 'America/Chicago',
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS cutoff_config;
COMMIT;
