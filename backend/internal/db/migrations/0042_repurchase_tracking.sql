-- +goose Up
BEGIN;

-- Tracks items repurchased via a completed shopping list (REP-01).
-- Written by CompleteVendorSection; read by GetStockHandler for badge display.
CREATE TABLE repurchase_log (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  purchase_item_id UUID NOT NULL REFERENCES purchase_items(id) ON DELETE CASCADE,
  shopping_list_id UUID NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
  quantity         INTEGER NOT NULL,
  repurchased_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_repurchase_log_item ON repurchase_log(purchase_item_id);
CREATE INDEX idx_repurchase_log_list ON repurchase_log(shopping_list_id);

-- Single-row config for weekly badge reset (REP-02).
-- day_of_week: 0=Sunday, 6=Saturday (matches cutoff_config convention).
CREATE TABLE repurchase_reset_config (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  day_of_week   INTEGER NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
  reset_time    TIME NOT NULL DEFAULT '06:00',
  timezone      TEXT NOT NULL DEFAULT 'America/Chicago',
  last_reset_at TIMESTAMPTZ,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS repurchase_log;
DROP TABLE IF EXISTS repurchase_reset_config;
COMMIT;
