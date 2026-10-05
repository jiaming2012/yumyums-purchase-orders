-- +goose Up
BEGIN;

-- ===========================================================================
-- 0087 — the scan dedupe becomes atomic (card I4 `atomic-scan-dedupe`,
-- ledger T-62 decision 195).
--
-- The public landing (GET /q/{short}) counted a scan with
--   INSERT … SELECT … WHERE NOT EXISTS (… scanned_at > now() - 10 minutes)
-- which is a read-then-write under READ COMMITTED: hits that arrive together
-- all read "no row yet" and all insert. Twelve barrier-aligned clients left
-- 12/12/12/11/12 rows where one was wanted, on an unauthenticated route.
--
-- Decided shape: a 10-minute TUMBLING bucket, a unique index over
-- (short, ip_hash, bucket), and ON CONFLICT DO NOTHING in the handler
-- (internal/marketing/landing.go insertScan). The database refuses the second
-- row; nothing reads before it writes.
--
--   * The window is still ten minutes. It is now a fixed bucket rather than a
--     sliding window, so two taps that straddle a bucket edge count twice —
--     decision 195's accepted trade-off.
--   * A NULL ip_hash is outside the partial index and inserts every time, as
--     before (two anonymous scans cannot be told apart; over-counting is the
--     honest failure).
--   * A scan is still one qr_scans row; every reader stays a plain count(*).
--
-- The column and the index are the two statements spike 02 ran on PostgreSQL
-- 16.13, verbatim. date_bin over timestamptz with a constant origin is
-- immutable, which is what lets the column be GENERATED … STORED.
-- ===========================================================================

ALTER TABLE qr_scans ADD COLUMN bucket timestamptz GENERATED ALWAYS AS (date_bin('10 minutes', scanned_at, timestamptz '2000-01-01 00:00:00+00')) STORED;

-- Rows the old statement already let through. A live table can hold several
-- rows with one (short, ip_hash, bucket), and the unique index below cannot be
-- built over them — the migration would fail and the server would not boot.
--
-- Two rows in one 10-minute bucket are less than ten minutes apart, so every
-- surplus row here is one the 10-minute rule always meant to refuse. Per
-- group, ONE row keeps its ip_hash: a row a signup already claimed
-- (subscriber_id set) if there is one, otherwise the earliest. Of the rest:
--
--   * a surplus row carrying a subscriber_id is NEVER deleted (it is a
--     converted visit). Its ip_hash is blanked, so it stays in the table and
--     in the count as an anonymous scan;
--   * a surplus row with no subscriber_id is deleted.
--
-- NULL-ip rows are not looked at. On a table with no duplicates both
-- statements touch 0 rows. The Down does not bring deleted rows back.
WITH ranked AS (
  SELECT id, subscriber_id,
         row_number() OVER (PARTITION BY short, ip_hash, bucket
                            ORDER BY (subscriber_id IS NULL), scanned_at, id) AS rn
    FROM qr_scans
   WHERE ip_hash IS NOT NULL
), blanked AS (
  UPDATE qr_scans s SET ip_hash = NULL
    FROM ranked r
   WHERE s.id = r.id AND r.rn > 1 AND r.subscriber_id IS NOT NULL
)
DELETE FROM qr_scans s
 USING ranked r
 WHERE s.id = r.id AND r.rn > 1 AND r.subscriber_id IS NULL;

CREATE UNIQUE INDEX qr_scans_dedupe_idx ON qr_scans (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL;

COMMIT;

-- +goose Down
BEGIN;
-- Back to 0086's shape: the index and the generated column go. Rows are not
-- touched, and the duplicates the Up collapsed are not restored. A binary that
-- still sends ON CONFLICT (short, ip_hash, bucket) cannot run against this
-- shape — roll the image back with the schema, not the schema alone.
DROP INDEX IF EXISTS qr_scans_dedupe_idx;
ALTER TABLE qr_scans DROP COLUMN IF EXISTS bucket;
COMMIT;
