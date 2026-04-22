-- +goose Up
BEGIN;

CREATE TABLE shopping_lists (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  po_id         UUID NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
  status        TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'completed')),
  assigned_to   UUID REFERENCES users(id),
  assigned_role TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  completed_at  TIMESTAMPTZ
);
CREATE INDEX idx_shopping_lists_po ON shopping_lists(po_id);

-- Per-vendor sections within a shopping list; vendor_id may be NULL for unassigned items.
-- vendor_name is snapshotted at creation time so display works even if vendor is later deleted.
CREATE TABLE shopping_list_vendor_sections (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shopping_list_id UUID NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
  vendor_id        UUID REFERENCES vendors(id),
  vendor_name      TEXT NOT NULL,
  status           TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'completed')),
  completed_by     UUID REFERENCES users(id),
  completed_at     TIMESTAMPTZ,
  UNIQUE (shopping_list_id, vendor_id)
);

-- Individual items (snapshotted from PO line items at approval time).
CREATE TABLE shopping_list_items (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shopping_list_id  UUID NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
  vendor_section_id UUID NOT NULL REFERENCES shopping_list_vendor_sections(id) ON DELETE CASCADE,
  purchase_item_id  UUID NOT NULL REFERENCES purchase_items(id),
  item_name         TEXT NOT NULL,
  photo_url         TEXT,
  store_location    TEXT,
  quantity          INTEGER NOT NULL,
  unit              TEXT NOT NULL DEFAULT '',
  checked           BOOLEAN NOT NULL DEFAULT false,
  checked_by        UUID REFERENCES users(id),
  checked_at        TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_shopping_list_items_list ON shopping_list_items(shopping_list_id);
CREATE INDEX idx_shopping_list_items_section ON shopping_list_items(vendor_section_id);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS shopping_list_items;
DROP TABLE IF EXISTS shopping_list_vendor_sections;
DROP TABLE IF EXISTS shopping_lists;
COMMIT;
