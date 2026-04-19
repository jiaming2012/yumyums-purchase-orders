-- +goose Up
BEGIN;
ALTER TABLE ob_video_parts ADD COLUMN thumbnail_url TEXT;
ALTER TABLE ob_progress ADD COLUMN max_watched_time FLOAT8;
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check CHECK (progress_type IN ('item', 'video_part', 'faq', 'video_watch_position'));
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check CHECK (progress_type IN ('item', 'video_part', 'faq'));
ALTER TABLE ob_progress DROP COLUMN max_watched_time;
ALTER TABLE ob_video_parts DROP COLUMN thumbnail_url;
COMMIT;
