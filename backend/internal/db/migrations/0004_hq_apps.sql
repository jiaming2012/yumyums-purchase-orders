-- +goose Up
BEGIN;
CREATE TABLE hq_apps (
  id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug    TEXT UNIQUE NOT NULL,
  name    TEXT NOT NULL,
  icon    TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT true
);
COMMIT;

-- +goose Down
DROP TABLE hq_apps;
