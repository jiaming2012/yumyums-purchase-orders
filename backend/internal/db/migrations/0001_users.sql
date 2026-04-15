-- +goose Up
BEGIN;
CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT UNIQUE NOT NULL,
  display_name  TEXT NOT NULL,
  password_hash TEXT,
  role          TEXT NOT NULL CHECK (role IN ('admin', 'manager', 'team_member')),
  status        TEXT NOT NULL DEFAULT 'invited' CHECK (status IN ('invited', 'active')),
  invited_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  accepted_at   TIMESTAMPTZ
);
COMMIT;

-- +goose Down
DROP TABLE users;
