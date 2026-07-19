-- +goose Up
BEGIN;

CREATE TABLE menu_items (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  master_id     TEXT UNIQUE NOT NULL,
  name          TEXT NOT NULL,
  menu          TEXT NOT NULL,
  menu_group    TEXT NOT NULL,
  menu_subgroup TEXT,
  last_seen     DATE NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX menu_items_last_seen_idx ON menu_items(last_seen DESC);
CREATE INDEX menu_items_menu_group_idx ON menu_items(menu_group);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS menu_items;
COMMIT;
