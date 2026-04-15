-- +goose Up
BEGIN;
CREATE TABLE checklist_sections (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id UUID NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
  title       TEXT NOT NULL,
  "order"     INTEGER NOT NULL,
  condition   JSONB
);
COMMIT;

-- +goose Down
DROP TABLE IF EXISTS checklist_sections;
