-- +goose Up
BEGIN;

CREATE TABLE purchase_orders (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  week_start DATE NOT NULL UNIQUE,
  status     TEXT NOT NULL DEFAULT 'draft'
               CHECK (status IN ('draft', 'locked', 'approved')),
  version    INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE po_line_items (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  po_id            UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
  purchase_item_id UUID NOT NULL REFERENCES purchase_items(id),
  quantity         INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
  unit             TEXT NOT NULL DEFAULT '',
  added_by         UUID NOT NULL REFERENCES users(id),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (po_id, purchase_item_id)
);

CREATE INDEX idx_po_line_items_po_id ON po_line_items(po_id);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS po_line_items;
DROP TABLE IF EXISTS purchase_orders;
COMMIT;
