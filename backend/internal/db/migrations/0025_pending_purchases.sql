-- +goose Up
BEGIN;

CREATE TABLE pending_purchases (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  bank_tx_id   TEXT NOT NULL,
  bank_total   NUMERIC(10,2) NOT NULL,
  vendor       TEXT NOT NULL,
  event_date   DATE,
  tax          NUMERIC(10,2),
  total        NUMERIC(10,2),
  total_units  INTEGER,
  total_cases  INTEGER,
  receipt_url  TEXT,
  reason       TEXT,
  items        JSONB NOT NULL DEFAULT '[]',
  confirmed_at TIMESTAMPTZ,
  confirmed_by UUID REFERENCES users(id) ON DELETE SET NULL,
  discarded_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX pending_purchases_bank_tx_id_idx ON pending_purchases(bank_tx_id);

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS pending_purchases;

COMMIT;
