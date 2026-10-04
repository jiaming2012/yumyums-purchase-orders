-- +goose Up
BEGIN;

-- ===========================================================================
-- 0086 — blank-on-delete backstop for the campaign and subscriber tables
-- (card I3 `dish-merge-and-erasure-backstop`, ledger T-62 decision 194).
--
-- 0083 and 0085 declared their foreign keys with no ON DELETE, so three
-- ordinary deletes were refused with 23503:
--
--   * DELETE FROM menu_items  — blocked by campaigns_admin.item_id and
--     qr_codes.item_id, which is what broke recipes.MergeMenuItem (re-point
--     recipes, delete the source dish) once any campaign named the dish;
--   * DELETE FROM subscribers — blocked by subscriber_events.subscriber_id,
--     so a right-to-erasure request needed table-by-table surgery;
--   * DELETE FROM qr_codes    — blocked by subscribers.source_short for any
--     code a subscriber first-touched.
--
-- The merge itself now RE-POINTS campaigns and codes to the surviving dish
-- (internal/recipes/repository.go) — that is the house convention. These
-- ALTERs are the BACKSTOP: a future table or code path that forgets the
-- re-point degrades to an empty label instead of an opaque 500.
--
-- Shape — SET NULL on every nullable link, and exactly ONE cascade:
--
--   campaigns_admin.item_id          SET NULL
--   qr_codes.item_id                 SET NULL
--   subscribers.source_short         SET NULL
--   subscriber_events.subscriber_id  CASCADE   (NOT NULL; a timeline row
--                                               cannot be blanked, and it is
--                                               the subscriber's own data)
--   qr_scans.subscriber_id           SET NULL  (NEW — 0083 declared the column
--                                               as a bare uuid with no FK, so
--                                               a subscriber delete left the
--                                               id dangling; spike correction 2)
--
-- 🛑 `qr_scans.short -> qr_codes(short)` IS DELIBERATELY NOT TOUCHED. A code
-- that has been scanned stays undeletable: codes are deactivated, never
-- deleted, and scan history is attribution evidence (spike correction 1).
-- A cascade there — or anywhere other than subscriber_events — could destroy
-- a campaign's history and is decision 194's explicit rejection.
-- internal/marketing/erasure_test.go pins the refusal.
--
-- Constraint names are Postgres' auto-generated `<table>_<column>_fkey`, READ
-- from pg_constraint on a database migrated to 85 (the read-back is committed
-- at .night-crew/runs/2026-10-03-autonomous/logs/dish-merge-and-erasure-backstop/
-- pg-constraint-at-85.log), not guessed. The four DROP/ADD statements are the
-- ones spike 02 printed, verbatim.
-- ===========================================================================

ALTER TABLE campaigns_admin   DROP CONSTRAINT campaigns_admin_item_id_fkey,         ADD CONSTRAINT campaigns_admin_item_id_fkey         FOREIGN KEY (item_id)       REFERENCES menu_items(id)  ON DELETE SET NULL;
ALTER TABLE qr_codes          DROP CONSTRAINT qr_codes_item_id_fkey,                ADD CONSTRAINT qr_codes_item_id_fkey                FOREIGN KEY (item_id)       REFERENCES menu_items(id)  ON DELETE SET NULL;
ALTER TABLE subscribers       DROP CONSTRAINT subscribers_source_short_fkey,        ADD CONSTRAINT subscribers_source_short_fkey        FOREIGN KEY (source_short)  REFERENCES qr_codes(short) ON DELETE SET NULL;
ALTER TABLE subscriber_events DROP CONSTRAINT subscriber_events_subscriber_id_fkey, ADD CONSTRAINT subscriber_events_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES subscribers(id) ON DELETE CASCADE;

-- The fifth. The column has had no FK since 0083, so an id naming a subscriber
-- who no longer exists is possible in principle and would make ADD CONSTRAINT
-- fail the whole migration at deploy. Blank any such id first — exactly what
-- the new ON DELETE SET NULL would have done had the constraint existed when
-- the subscriber went. On a database with no dangling ids this updates 0 rows.
UPDATE qr_scans s SET subscriber_id = NULL
 WHERE s.subscriber_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM subscribers b WHERE b.id = s.subscriber_id);

ALTER TABLE qr_scans ADD CONSTRAINT qr_scans_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES subscribers(id) ON DELETE SET NULL;

COMMIT;

-- +goose Down
BEGIN;
-- Back to 0083/0085's shape: the four FKs lose their ON DELETE action (same
-- names), and the FK 0083 never declared is dropped. Rows are not touched —
-- every row that satisfies the Up constraints satisfies these. (The Up's
-- blanking of dangling qr_scans.subscriber_id values is not reversed: the ids
-- named nobody.)
ALTER TABLE qr_scans DROP CONSTRAINT IF EXISTS qr_scans_subscriber_id_fkey;

ALTER TABLE subscriber_events DROP CONSTRAINT subscriber_events_subscriber_id_fkey, ADD CONSTRAINT subscriber_events_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES subscribers(id);
ALTER TABLE subscribers       DROP CONSTRAINT subscribers_source_short_fkey,        ADD CONSTRAINT subscribers_source_short_fkey        FOREIGN KEY (source_short)  REFERENCES qr_codes(short);
ALTER TABLE qr_codes          DROP CONSTRAINT qr_codes_item_id_fkey,                ADD CONSTRAINT qr_codes_item_id_fkey                FOREIGN KEY (item_id)       REFERENCES menu_items(id);
ALTER TABLE campaigns_admin   DROP CONSTRAINT campaigns_admin_item_id_fkey,         ADD CONSTRAINT campaigns_admin_item_id_fkey         FOREIGN KEY (item_id)       REFERENCES menu_items(id);
COMMIT;
