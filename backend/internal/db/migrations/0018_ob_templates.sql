-- +goose Up
BEGIN;
CREATE TABLE ob_templates (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name       TEXT NOT NULL,
  role       TEXT, -- nullable: some templates are role-agnostic
  created_by UUID REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ob_sections (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  template_id       UUID NOT NULL REFERENCES ob_templates(id) ON DELETE CASCADE,
  title             TEXT NOT NULL,
  sort_order        INT NOT NULL DEFAULT 0,
  requires_sign_off BOOLEAN NOT NULL DEFAULT false,
  is_faq            BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX idx_ob_sections_template ON ob_sections(template_id);
COMMIT;

-- +goose Down
BEGIN;
DROP TABLE ob_sections;
DROP TABLE ob_templates;
COMMIT;
