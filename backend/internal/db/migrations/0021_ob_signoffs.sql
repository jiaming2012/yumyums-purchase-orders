-- +goose Up
BEGIN;
CREATE TABLE ob_signoffs (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  section_id    UUID NOT NULL REFERENCES ob_sections(id) ON DELETE CASCADE,
  manager_id    UUID NOT NULL REFERENCES users(id),
  hire_id       UUID NOT NULL REFERENCES users(id),
  notes         TEXT NOT NULL,
  rating        TEXT NOT NULL CHECK (rating IN ('ready', 'needs_practice', 'struggling')),
  signed_off_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(section_id, hire_id) -- one sign-off per section per hire
);
COMMIT;

-- +goose Down
BEGIN;
DROP TABLE ob_signoffs;
COMMIT;
