-- +goose Up
BEGIN;

-- ===========================================================================
-- 0080 — a sync run can be cancelled.
--
-- A Mercury sync or a bulk re-parse is one receipt download plus one model
-- call per row, so a queue of a dozen runs for minutes. Until now the only way
-- out was to wait: the goroutine detaches to context.Background() and nothing
-- holds a handle to it. The operator watching "Syncing… 3m 10s" had no way to
-- say "stop, I started the wrong one".
--
-- 'cancelled' is a TERMINAL status, distinct from 'failed': nothing went
-- wrong, a person decided. The single-running partial unique index keys off
-- status = 'running', so a cancelled row releases the single-flight slot
-- exactly like done/failed and the next sync can start immediately.
--
-- The counts on a cancelled row are real, not zeroed: each receipt is parsed
-- and persisted independently, so the ones finished before the cancel keep
-- their results. The chip says "cancelled after N" rather than implying the
-- run left nothing behind.
-- ===========================================================================
ALTER TABLE receipt_sync_runs DROP CONSTRAINT IF EXISTS receipt_sync_runs_status_check;
ALTER TABLE receipt_sync_runs ADD CONSTRAINT receipt_sync_runs_status_check
  CHECK (status IN ('running','done','failed','cancelled'));

COMMIT;

-- +goose Down
BEGIN;

-- Rows already cancelled would violate the narrowed constraint; fold them into
-- 'failed' (the closest terminal state that predates this migration) first.
UPDATE receipt_sync_runs SET status = 'failed' WHERE status = 'cancelled';
ALTER TABLE receipt_sync_runs DROP CONSTRAINT IF EXISTS receipt_sync_runs_status_check;
ALTER TABLE receipt_sync_runs ADD CONSTRAINT receipt_sync_runs_status_check
  CHECK (status IN ('running','done','failed'));

COMMIT;
