-- +goose Up
BEGIN;

CREATE TABLE stock_count_overrides (
  item_description TEXT PRIMARY KEY,
  quantity         INTEGER NOT NULL,
  reason           TEXT,
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS stock_count_overrides;
COMMIT;
