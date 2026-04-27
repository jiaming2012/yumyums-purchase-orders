-- +goose Up
BEGIN;
CREATE TABLE ob_sub_items (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item_id    UUID NOT NULL REFERENCES ob_items(id) ON DELETE CASCADE,
  label      TEXT NOT NULL,
  sort_order INT NOT NULL DEFAULT 0
);
CREATE INDEX idx_ob_sub_items_item ON ob_sub_items(item_id);
COMMIT;

-- +goose Down
BEGIN;
DROP TABLE ob_sub_items;
COMMIT;
