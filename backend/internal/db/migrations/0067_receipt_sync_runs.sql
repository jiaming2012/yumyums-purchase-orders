-- +goose Up
BEGIN;

-- receipt_sync_runs tracks each on-demand Mercury receipt sync invocation
-- triggered by the user via the inventory.html Purchases tab "Sync Receipts"
-- button. The row is created at status='running' before the goroutine spawns,
-- then updated to 'done' or 'failed' when the ingest cycle completes (or the
-- goroutine panics — defer recover() guarantees a terminal status).
--
-- This table is the durability layer that lets the UI survive a full page
-- reload or PWA close/reopen while a sync is in flight: the frontend polls
-- GET /api/v1/inventory/sync-receipts/status which reads the latest row.
CREATE TABLE receipt_sync_runs (
  id              BIGSERIAL PRIMARY KEY,
  started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  finished_at     TIMESTAMPTZ,
  status          TEXT NOT NULL CHECK (status IN ('running','done','failed')),
  processed       INTEGER NOT NULL DEFAULT 0,
  auto_created    INTEGER NOT NULL DEFAULT 0,
  pending_review  INTEGER NOT NULL DEFAULT 0,
  cached          INTEGER NOT NULL DEFAULT 0,
  error           TEXT,
  triggered_by    TEXT NOT NULL DEFAULT 'manual'
);

-- Single-flight guard: partial unique index — at most one running row at a
-- time. The INSERT in SyncReceiptsHandler relies on the resulting unique
-- violation (SQLSTATE 23505) to return 409 sync_already_running.
CREATE UNIQUE INDEX receipt_sync_runs_single_running
  ON receipt_sync_runs ((1)) WHERE status = 'running';

-- Latest-row lookup used by GET /sync-receipts/status (ORDER BY started_at DESC LIMIT 1).
CREATE INDEX receipt_sync_runs_started_at_desc
  ON receipt_sync_runs (started_at DESC);

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE receipt_sync_runs;

COMMIT;
