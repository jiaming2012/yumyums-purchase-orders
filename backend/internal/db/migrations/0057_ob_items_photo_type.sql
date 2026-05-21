-- +goose Up
BEGIN;
ALTER TABLE ob_items ADD COLUMN reference_photo_url TEXT;
ALTER TABLE ob_items ADD COLUMN require_proof_photo BOOLEAN NOT NULL DEFAULT false;
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE ob_items DROP COLUMN require_proof_photo;
ALTER TABLE ob_items DROP COLUMN reference_photo_url;
COMMIT;
