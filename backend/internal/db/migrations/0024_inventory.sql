-- +goose Up
BEGIN;

CREATE TABLE vendors (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tags (
  id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT UNIQUE NOT NULL
);

CREATE TABLE item_groups (
  id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name     TEXT NOT NULL,
  par_days INTEGER
);

CREATE TABLE item_group_tags (
  group_id UUID NOT NULL REFERENCES item_groups(id) ON DELETE CASCADE,
  tag_id   UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (group_id, tag_id)
);

CREATE TABLE purchase_items (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  description TEXT UNIQUE NOT NULL,
  group_id    UUID REFERENCES item_groups(id) ON DELETE SET NULL
);

CREATE TABLE purchase_events (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id   UUID NOT NULL REFERENCES vendors(id),
  bank_tx_id  TEXT UNIQUE NOT NULL,
  event_date  DATE NOT NULL,
  tax         NUMERIC(10,2) NOT NULL DEFAULT 0,
  total       NUMERIC(10,2) NOT NULL,
  receipt_url TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE purchase_line_items (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  purchase_event_id UUID NOT NULL REFERENCES purchase_events(id) ON DELETE CASCADE,
  purchase_item_id  UUID REFERENCES purchase_items(id),
  description       TEXT NOT NULL,
  quantity          INTEGER NOT NULL,
  price             NUMERIC(10,4) NOT NULL,
  is_case           BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX purchase_events_vendor_id_idx ON purchase_events(vendor_id);
CREATE INDEX purchase_events_event_date_idx ON purchase_events(event_date DESC);
CREATE INDEX purchase_line_items_event_id_idx ON purchase_line_items(purchase_event_id);

INSERT INTO hq_apps (slug, name, icon, is_active)
VALUES ('inventory', 'Inventory', '📦', true)
ON CONFLICT (slug) DO NOTHING;

COMMIT;

-- +goose Down
BEGIN;

DELETE FROM hq_apps WHERE slug = 'inventory';

DROP TABLE IF EXISTS purchase_line_items;
DROP TABLE IF EXISTS purchase_events;
DROP TABLE IF EXISTS purchase_items;
DROP TABLE IF EXISTS item_group_tags;
DROP TABLE IF EXISTS item_groups;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS vendors;

COMMIT;
