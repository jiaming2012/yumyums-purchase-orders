-- +goose Up
BEGIN;
CREATE TABLE template_assignments (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id     UUID NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
  assignee_type   TEXT NOT NULL CHECK (assignee_type IN ('role', 'user')),
  assignee_id     TEXT NOT NULL,
  assignment_role TEXT NOT NULL CHECK (assignment_role IN ('assignee', 'approver'))
);
CREATE INDEX template_assignments_template_idx ON template_assignments(template_id);
COMMIT;

-- +goose Down
DROP INDEX IF EXISTS template_assignments_template_idx;
DROP TABLE IF EXISTS template_assignments;
