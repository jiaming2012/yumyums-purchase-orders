-- +goose Up
BEGIN;
CREATE TABLE submission_rejections (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id UUID NOT NULL REFERENCES checklist_submissions(id) ON DELETE CASCADE,
  field_id      UUID NOT NULL REFERENCES checklist_fields(id),
  comment       TEXT NOT NULL,
  require_photo BOOLEAN NOT NULL DEFAULT false,
  rejected_by   UUID NOT NULL REFERENCES users(id),
  rejected_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMIT;

-- +goose Down
DROP TABLE IF EXISTS submission_rejections;
