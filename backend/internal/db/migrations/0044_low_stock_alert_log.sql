-- +goose Up
BEGIN;

-- Tracks which low-stock items have been alerted per week to prevent duplicate sends (ALRT-02 idempotency).
-- One row per (item_description, week_start) — INSERT ON CONFLICT DO NOTHING prevents duplicates.
CREATE TABLE low_stock_alert_log (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  item_description TEXT NOT NULL,
  week_start       DATE NOT NULL,
  sent_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (item_description, week_start)
);
CREATE INDEX idx_low_stock_alert_log_week ON low_stock_alert_log(week_start);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS low_stock_alert_log;
COMMIT;
