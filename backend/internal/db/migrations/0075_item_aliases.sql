-- +goose Up
BEGIN;

-- Alternate names for catalog items, learned from receipts. A receipt line
-- like "100% CL HNY 24Z BRAM" can be linked to the catalog item "Honey";
-- storing the receipt text as an alias lets every future receipt carrying
-- that name (or the same item bought at another store under yet another
-- name, once linked once) auto-match without fuzzy/AI help.
--
-- One alias maps to exactly one item (unique on lower(alias)); at match
-- time the item description always wins over any alias, so an alias that
-- shadows a description can never steal its matches.
CREATE TABLE item_aliases (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  purchase_item_id UUID NOT NULL REFERENCES purchase_items(id) ON DELETE CASCADE,
  alias            TEXT NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX item_aliases_alias_lower_uniq ON item_aliases (LOWER(alias));
CREATE INDEX item_aliases_item_idx ON item_aliases (purchase_item_id);

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS item_aliases;

COMMIT;
