-- +goose Up
BEGIN;

ALTER TABLE purchase_orders ADD COLUMN locked_at   TIMESTAMPTZ;
ALTER TABLE purchase_orders ADD COLUMN approved_at TIMESTAMPTZ;
ALTER TABLE purchase_orders ADD COLUMN approved_by UUID REFERENCES users(id);

-- Drop the existing CHECK constraint (named by Postgres from the inline CHECK in 0034)
ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_status_check;

-- Recreate with all 5 status values
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_status_check
  CHECK (status IN ('draft', 'locked', 'approved', 'shopping_active', 'completed'));

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE purchase_orders DROP CONSTRAINT IF EXISTS purchase_orders_status_check;
ALTER TABLE purchase_orders ADD CONSTRAINT purchase_orders_status_check
  CHECK (status IN ('draft', 'locked', 'approved'));

ALTER TABLE purchase_orders DROP COLUMN IF EXISTS approved_by;
ALTER TABLE purchase_orders DROP COLUMN IF EXISTS approved_at;
ALTER TABLE purchase_orders DROP COLUMN IF EXISTS locked_at;

COMMIT;
