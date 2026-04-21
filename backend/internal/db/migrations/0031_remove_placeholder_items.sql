-- +goose Up
BEGIN;

-- Remove seeded placeholder items that were never referenced by any purchase_line_item.
-- Real items are created by the receipt worker from actual receipt text.
DELETE FROM purchase_items
WHERE id NOT IN (
  SELECT DISTINCT purchase_item_id FROM purchase_line_items WHERE purchase_item_id IS NOT NULL
);

COMMIT;

-- +goose Down
-- No rollback — placeholder items are not worth restoring.
