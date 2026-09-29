-- +goose Up
BEGIN;

-- ===========================================================================
-- 0081 — one alias per item can be promoted to the name the crew reads.
--
-- item_aliases (0075) is a BAG of alternate names, and it mixes two kinds of
-- string that look identical in the column but are not the same thing:
--
--   * machine-learned receipt text  — "100% CL HNY 24Z BRAM", written by the
--     receipt worker / by linking a receipt line, purely a MATCHING key;
--   * a name a human typed          — "Honey", written into the Setup chips
--     because the catalog description is unreadable on a phone.
--
-- Nothing in the schema told them apart, so "show the nickname" had no
-- answer: picking any alias off the bag renders the raw receipt string about
-- as often as the readable one. inventory.html:1123 worked around this on the
-- Purchases card only, by excluding the alias equal to the receipt text it
-- happened to have in hand — a trick no other screen can play, because no
-- other screen has the receipt text.
--
-- is_display marks the one alias a human chose as the label. The partial
-- unique index makes "at most one per item" a database fact, which is what
-- lets item_display_name() below use LIMIT 1 and still be deterministic.
-- ===========================================================================
ALTER TABLE item_aliases ADD COLUMN IF NOT EXISTS is_display BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX IF NOT EXISTS item_aliases_one_display_per_item
  ON item_aliases (purchase_item_id)
  WHERE is_display;

-- item_display_name is THE definition of "what do we call this item" for
-- every read path in the app: the promoted alias if one is set, otherwise the
-- caller's fallback (normally purchase_items.description, or the receipt
-- line's own text when the line is not linked to a catalog item).
--
-- It is a function rather than a view deliberately: a view over purchase_items
-- would have to spell out its column list, and Postgres freezes `SELECT *` at
-- definition time — so the next column added to purchase_items would silently
-- not appear to any caller. A function has nothing to go stale.
--
-- STABLE (not IMMUTABLE): it reads a table. PARALLEL SAFE: read-only.
CREATE OR REPLACE FUNCTION item_display_name(p_item_id UUID, p_fallback TEXT)
RETURNS TEXT
LANGUAGE sql
STABLE
PARALLEL SAFE
AS $$
  SELECT COALESCE(
    (SELECT ia.alias
       FROM item_aliases ia
      WHERE ia.purchase_item_id = p_item_id
        AND ia.is_display
      LIMIT 1),
    p_fallback
  )
$$;

COMMIT;

-- +goose Down
BEGIN;

DROP FUNCTION IF EXISTS item_display_name(UUID, TEXT);
DROP INDEX IF EXISTS item_aliases_one_display_per_item;
ALTER TABLE item_aliases DROP COLUMN IF EXISTS is_display;

COMMIT;
