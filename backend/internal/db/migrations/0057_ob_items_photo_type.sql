-- +goose Up
BEGIN;
ALTER TABLE ob_items DROP CONSTRAINT ob_items_type_check;
ALTER TABLE ob_items ADD CONSTRAINT ob_items_type_check
  CHECK (type IN ('checkbox', 'video_series', 'faq', 'photo'));
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE ob_items DROP CONSTRAINT ob_items_type_check;
ALTER TABLE ob_items ADD CONSTRAINT ob_items_type_check
  CHECK (type IN ('checkbox', 'video_series', 'faq'));
COMMIT;
