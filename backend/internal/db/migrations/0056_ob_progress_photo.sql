-- +goose Up
BEGIN;
ALTER TABLE ob_progress ADD COLUMN value TEXT;
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check
  CHECK (progress_type IN ('item', 'video_part', 'faq', 'video_watch_position', 'sub_item', 'photo'));
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check
  CHECK (progress_type IN ('item', 'video_part', 'faq', 'video_watch_position', 'sub_item'));
ALTER TABLE ob_progress DROP COLUMN value;
COMMIT;
