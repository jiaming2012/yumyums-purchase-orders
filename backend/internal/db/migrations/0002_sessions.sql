-- +goose Up
BEGIN;
CREATE TABLE sessions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token_hash  TEXT UNIQUE NOT NULL,
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at  TIMESTAMPTZ
);
CREATE INDEX sessions_user_idx ON sessions(user_id);
COMMIT;

-- +goose Down
DROP TABLE sessions;
