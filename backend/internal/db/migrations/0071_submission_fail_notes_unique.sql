-- +goose Up
BEGIN;

-- One fail note per (submission, field) — the constraint submission_responses
-- has carried since 0012 (`UNIQUE (submission_id, field_id)`) and that
-- submission_fail_notes was created without in 0013.
--
-- The bare INSERT in submitChecklist (repository.go) became a duplicate factory
-- when the client started REUSING a queued idempotency_key on a second press
-- (ledger T-23 decision 60): two POSTs deliberately land on one submission row,
-- responses upsert, fail notes appended. Measured 2026-07-27 —
-- `submission_rows=1 response_rows=1 fail_note_rows=2`. The approver read the
-- same note twice. This index is one half of the fix; the matching
-- `ON CONFLICT (submission_id, field_id) DO UPDATE` in the same change set is
-- the other, and neither works alone (index without upsert = a hard 500 on the
-- second POST).
--
-- NO DEDUPLICATION IS PERFORMED HERE, deliberately. Which of two duplicate rows
-- to keep is a data decision, not a schema one, and this migration refuses to
-- improvise it. It is safe to write bare because the question was CHECKED
-- rather than assumed before it was authored: on the live Postgres both the
-- `production` schema (prod's, per docker-compose.prod.yml) and `public`
-- (dev's) held 0 rows in submission_fail_notes, hence 0 duplicates. If this
-- migration ever fails on some environment with `could not create unique
-- index`, that environment has duplicates and the dedup rule is an operator
-- decision — do not add a DELETE here unattended.
--
-- Rows with submission_id IS NULL are intentionally unconstrained: Postgres
-- treats NULLs as distinct in a unique index. unsubmitChecklist detaches fail
-- notes to submission_id = NULL and nothing re-attaches them (they carry no
-- answered_by, so submitChecklist cannot re-claim them the way it does
-- responses). Those orphans are a separate, pre-existing defect; this index
-- neither fixes nor collides with them.
CREATE UNIQUE INDEX IF NOT EXISTS submission_fail_notes_submission_field_uniq
  ON submission_fail_notes (submission_id, field_id);

COMMIT;

-- +goose Down
BEGIN;

DROP INDEX IF EXISTS submission_fail_notes_submission_field_uniq;

COMMIT;
