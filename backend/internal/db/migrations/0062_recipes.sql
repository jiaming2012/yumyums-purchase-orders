-- +goose Up
BEGIN;

CREATE TABLE recipes (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  menu_item_id     UUID NOT NULL REFERENCES menu_items(id)    ON DELETE CASCADE,
  purchase_item_id UUID NOT NULL REFERENCES purchase_items(id) ON DELETE CASCADE,
  usage_pct        NUMERIC(5,2) NOT NULL CHECK (usage_pct >= 0 AND usage_pct <= 100),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (menu_item_id, purchase_item_id)
);

CREATE INDEX recipes_purchase_item_id_idx ON recipes(purchase_item_id);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS recipes;
COMMIT;
