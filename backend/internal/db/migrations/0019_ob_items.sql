-- +goose Up
BEGIN;
CREATE TABLE ob_items (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  section_id UUID NOT NULL REFERENCES ob_sections(id) ON DELETE CASCADE,
  type       TEXT NOT NULL CHECK (type IN ('checkbox', 'video_series', 'faq')),
  label      TEXT NOT NULL, -- for checkbox: the item text; for faq: the question
  answer     TEXT, -- for faq items: the answer text (null for other types)
  sort_order INT NOT NULL DEFAULT 0
);
CREATE INDEX idx_ob_items_section ON ob_items(section_id);

CREATE TABLE ob_video_parts (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item_id     UUID NOT NULL REFERENCES ob_items(id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  description TEXT,
  url         TEXT NOT NULL,
  sort_order  INT NOT NULL DEFAULT 0
);
CREATE INDEX idx_ob_video_parts_item ON ob_video_parts(item_id);
COMMIT;

-- +goose Down
BEGIN;
DROP TABLE ob_video_parts;
DROP TABLE ob_items;
COMMIT;
