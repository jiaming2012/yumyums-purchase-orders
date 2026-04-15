-- +goose Up
BEGIN;
CREATE TABLE checklist_templates (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name              TEXT NOT NULL,
  requires_approval BOOLEAN NOT NULL DEFAULT false,
  created_by        UUID REFERENCES users(id),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  archived_at       TIMESTAMPTZ
);
COMMIT;

-- +goose Down
DROP TABLE IF EXISTS checklist_templates;
