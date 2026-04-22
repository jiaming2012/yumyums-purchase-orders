-- +goose Up
BEGIN;

ALTER TABLE purchase_items ADD COLUMN vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL;
CREATE INDEX idx_purchase_items_vendor ON purchase_items(vendor_id);

COMMIT;

-- +goose Down
BEGIN;
DROP INDEX IF EXISTS idx_purchase_items_vendor;
ALTER TABLE purchase_items DROP COLUMN IF EXISTS vendor_id;
COMMIT;
