-- +goose Up
BEGIN;

-- One row per Monday weekly drift check; payload JSONB carries the structured
-- banner data (sections + flagged ingredients) so the banner endpoint can read
-- the latest week without recomputing. Idempotency: week_start is PK so the
-- scheduler's INSERT ON CONFLICT DO NOTHING guarantees one row per week.
CREATE TABLE drift_check_results (
  week_start  DATE PRIMARY KEY,
  payload     JSONB NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS drift_check_results;
COMMIT;
