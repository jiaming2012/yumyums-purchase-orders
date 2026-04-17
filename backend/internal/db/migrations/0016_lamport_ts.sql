-- +goose Up
BEGIN;
ALTER TABLE submission_responses  ADD COLUMN lamport_ts BIGINT NOT NULL DEFAULT 0;
ALTER TABLE checklist_submissions ADD COLUMN lamport_ts BIGINT NOT NULL DEFAULT 0;
ALTER TABLE checklist_templates   ADD COLUMN lamport_ts BIGINT NOT NULL DEFAULT 0;
COMMIT;

-- +goose Down
ALTER TABLE checklist_templates   DROP COLUMN IF EXISTS lamport_ts;
ALTER TABLE checklist_submissions DROP COLUMN IF EXISTS lamport_ts;
ALTER TABLE submission_responses  DROP COLUMN IF EXISTS lamport_ts;
