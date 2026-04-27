-- +goose Up
-- Unique name constraint excluding soft-deleted (archived) templates
CREATE UNIQUE INDEX idx_checklist_templates_name_active ON checklist_templates (name) WHERE archived_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_checklist_templates_name_active;
