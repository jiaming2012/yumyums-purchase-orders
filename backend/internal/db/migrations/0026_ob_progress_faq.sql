-- +goose Up
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check CHECK (progress_type IN ('item', 'video_part', 'faq'));

-- +goose Down
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check CHECK (progress_type IN ('item', 'video_part'));
