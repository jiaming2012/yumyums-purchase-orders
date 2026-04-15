-- +goose Up
BEGIN;
CREATE TABLE checklist_fields (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  section_id       UUID NOT NULL REFERENCES checklist_sections(id) ON DELETE CASCADE,
  parent_field_id  UUID REFERENCES checklist_fields(id),
  type             TEXT NOT NULL CHECK (type IN ('checkbox','yes_no','text','temperature','photo')),
  label            TEXT NOT NULL,
  required         BOOLEAN NOT NULL DEFAULT false,
  "order"          INTEGER NOT NULL,
  config           JSONB,
  fail_trigger     JSONB,
  condition        JSONB
);
COMMIT;

-- +goose Down
DROP TABLE IF EXISTS checklist_fields;
