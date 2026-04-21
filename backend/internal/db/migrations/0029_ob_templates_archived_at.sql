-- +goose Up
ALTER TABLE ob_templates ADD COLUMN archived_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE ob_templates DROP COLUMN archived_at;
