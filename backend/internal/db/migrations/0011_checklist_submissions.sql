-- +goose Up
BEGIN;
CREATE TABLE checklist_submissions (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id       UUID NOT NULL REFERENCES checklist_templates(id),
  template_snapshot JSONB NOT NULL,
  submitted_by      UUID NOT NULL REFERENCES users(id),
  submitted_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  status            TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected','completed')),
  reviewed_by       UUID REFERENCES users(id),
  reviewed_at       TIMESTAMPTZ,
  idempotency_key   UUID UNIQUE
);
COMMIT;

-- +goose Down
DROP TABLE IF EXISTS checklist_submissions;
