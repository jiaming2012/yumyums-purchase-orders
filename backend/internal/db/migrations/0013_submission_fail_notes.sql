-- +goose Up
BEGIN;
CREATE TABLE submission_fail_notes (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),
  note          TEXT NOT NULL DEFAULT '',
  severity      TEXT CHECK (severity IN ('minor','major','critical')),
  photo_url     TEXT
);
COMMIT;

-- +goose Down
DROP TABLE IF EXISTS submission_fail_notes;
