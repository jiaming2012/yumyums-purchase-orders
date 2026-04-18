-- +goose Up
BEGIN;
CREATE TABLE ob_template_assignments (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  hire_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  template_id UUID NOT NULL REFERENCES ob_templates(id) ON DELETE CASCADE,
  assigned_by UUID REFERENCES users(id),
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(hire_id, template_id)
);
CREATE INDEX idx_ob_assignments_hire ON ob_template_assignments(hire_id);
COMMIT;

-- +goose Down
BEGIN;
DROP TABLE ob_template_assignments;
COMMIT;
