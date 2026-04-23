-- +goose Up
BEGIN;

-- Tracks which alert types have been sent per week to prevent duplicate sends (D-08 idempotency).
-- alert_type: 'cutoff_reminder' | 'shopping_completion'
CREATE TABLE alert_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  alert_type TEXT NOT NULL,
  week_start DATE NOT NULL,
  sent_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (alert_type, week_start)
);
CREATE INDEX idx_alert_log_week ON alert_log(week_start);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS alert_log;
COMMIT;
