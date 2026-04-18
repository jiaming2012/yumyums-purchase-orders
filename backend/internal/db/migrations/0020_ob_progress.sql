-- +goose Up
BEGIN;
CREATE TABLE ob_progress (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  hire_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  item_id       UUID NOT NULL, -- references ob_items.id OR ob_video_parts.id
  progress_type TEXT NOT NULL CHECK (progress_type IN ('item', 'video_part')),
  checked_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(hire_id, item_id, progress_type)
);
CREATE INDEX idx_ob_progress_hire ON ob_progress(hire_id);
COMMIT;

-- +goose Down
BEGIN;
DROP TABLE ob_progress;
COMMIT;
