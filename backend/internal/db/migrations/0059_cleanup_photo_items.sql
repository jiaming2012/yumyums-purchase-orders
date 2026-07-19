-- +goose Up
-- Remove orphaned items with type='photo' created during development.
-- The photo feature was pivoted: photos are now properties of checkbox items,
-- not a separate item type.
DELETE FROM ob_items WHERE type = 'photo';

-- +goose Down
-- No-op: cannot restore deleted items
SELECT 1;
