-- +goose Up
BEGIN;

-- Remove duplicate item_groups, keeping the oldest (first inserted) row per name.
-- Re-point purchase_items and item_group_tags to the kept row before deleting dupes.
WITH keepers AS (
  SELECT DISTINCT ON (name) id, name
  FROM item_groups
  ORDER BY name, id
),
dupes AS (
  SELECT ig.id AS dupe_id, k.id AS keep_id
  FROM item_groups ig
  JOIN keepers k ON k.name = ig.name
  WHERE ig.id != k.id
)
UPDATE purchase_items SET group_id = d.keep_id
FROM dupes d WHERE purchase_items.group_id = d.dupe_id;

WITH keepers AS (
  SELECT DISTINCT ON (name) id, name
  FROM item_groups
  ORDER BY name, id
),
dupes AS (
  SELECT ig.id AS dupe_id, k.id AS keep_id
  FROM item_groups ig
  JOIN keepers k ON k.name = ig.name
  WHERE ig.id != k.id
)
DELETE FROM item_group_tags WHERE group_id IN (SELECT dupe_id FROM dupes);

WITH keepers AS (
  SELECT DISTINCT ON (name) id, name
  FROM item_groups
  ORDER BY name, id
)
DELETE FROM item_groups WHERE id NOT IN (SELECT id FROM keepers);

ALTER TABLE item_groups ADD CONSTRAINT item_groups_name_key UNIQUE (name);

COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE item_groups DROP CONSTRAINT IF EXISTS item_groups_name_key;
COMMIT;
