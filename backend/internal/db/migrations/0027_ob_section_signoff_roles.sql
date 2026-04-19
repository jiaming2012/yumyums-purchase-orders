-- +goose Up
ALTER TABLE ob_sections ADD COLUMN sign_off_roles TEXT[];

-- +goose Down
ALTER TABLE ob_sections DROP COLUMN sign_off_roles;
