-- +goose Up
BEGIN;
CREATE TABLE submission_responses (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),
  value         JSONB NOT NULL,
  answered_by   UUID NOT NULL REFERENCES users(id),
  answered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (submission_id, field_id)
);
CREATE UNIQUE INDEX submission_responses_draft_idx
  ON submission_responses(field_id, answered_by)
  WHERE submission_id IS NULL;
COMMIT;

-- +goose Down
DROP INDEX IF EXISTS submission_responses_draft_idx;
DROP TABLE IF EXISTS submission_responses;
