-- Card A1 · app-timezone-unify-new-york · built on run overnight-20260729-2
-- (parked), resumed and merged on run overnight-20260801. Neither run deployed.
--
-- The operator ruled that the app's timezone is America/New_York
-- (ledger T-26 decision 83). Two config tables were created with an
-- America/Chicago column default:
--
--   0037_cutoff_config.sql        cutoff_config.timezone
--   0042_repurchase_tracking.sql  repurchase_reset_config.timezone
--
-- Those two migrations are deliberately NOT edited — applied migrations are
-- immutable history. This migration is the forward fix, and it is the
-- authoritative statement of the column defaults from here on.
--
-- 🛑 CHANGEOVER DATE: THE DEPLOY THAT RUNS THIS MIGRATION — DATE TBD.
--
-- This migration was written and merged without a deploy. There is no date to
-- write here yet, and writing the merge date would point a future reader at a
-- day on which no boundary moved. **The changeover date IS the date this
-- migration first ran in production** — recover it from goose's
-- `goose_db_version.tstamp` for version 72, or from the deploy log.
--
-- On that date, and exactly once, every weekly/daily boundary in the app moves
-- one hour earlier in wall-clock terms (Chicago -> New York):
--
--   * purchasing.CurrentWeekStart  — the Monday every purchasing week hangs off
--   * recipes costWindow           — the 12-complete-ISO-week cost window
--   * recipes drift scheduler      — now Monday 09:00 New York, was 09:00 Chicago
--   * inventory pending-period date cast (COGS + the completeness gate that
--     feeds sales-processor's weekly payroll)
--   * the frontend's "today" for submissions and the offline queue's period
--
-- FIX FORWARD ONLY. Weekly COGS and payroll figures produced BEFORE that deploy
-- were already acted on and are NOT restated. A future reader comparing two
-- weeks either side of it will find one boundary that moved; this is why, and it
-- happened once.
--
-- The UPDATE statements matter as much as the DEFAULTs: without them, rows
-- already written by 0037/0042 (or by purchasing/repurchase.go's insert-default
-- path, or by purchasing.html's cutoff form, which was actively POSTing
-- America/Chicago) keep the app on the old boundary and the two zones go on
-- disagreeing — which is the bug this card exists to end.

-- +goose Up
BEGIN;

ALTER TABLE cutoff_config
  ALTER COLUMN timezone SET DEFAULT 'America/New_York';

ALTER TABLE repurchase_reset_config
  ALTER COLUMN timezone SET DEFAULT 'America/New_York';

UPDATE cutoff_config
   SET timezone   = 'America/New_York',
       updated_at = now()
 WHERE timezone = 'America/Chicago';

UPDATE repurchase_reset_config
   SET timezone   = 'America/New_York',
       updated_at = now()
 WHERE timezone = 'America/Chicago';

COMMIT;

-- +goose Down
BEGIN;

-- Restores the pre-0072 defaults only. Row values are NOT reverted: a rollback
-- must not silently move a live cutoff or badge-reset boundary a second time.
ALTER TABLE cutoff_config
  ALTER COLUMN timezone SET DEFAULT 'America/Chicago';

ALTER TABLE repurchase_reset_config
  ALTER COLUMN timezone SET DEFAULT 'America/Chicago';

COMMIT;
