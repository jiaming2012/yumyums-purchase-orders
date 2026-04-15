-- +goose Up
BEGIN;
CREATE TABLE checklist_schedules (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id UUID NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
  active_days INTEGER[],
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMIT;

-- +goose Down
DROP TABLE IF EXISTS checklist_schedules;
