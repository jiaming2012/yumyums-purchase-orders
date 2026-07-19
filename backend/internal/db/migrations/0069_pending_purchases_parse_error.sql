-- +goose Up
BEGIN;

-- Adds a nullable parse_error column so the worker can persist the actual
-- Anthropic/parse error string on a pending_purchases row when the
-- receipt could not be parsed automatically. Surfaced on the FE pending
-- review card so the owner can see WHY parsing failed (e.g. the receipt
-- URL returned HTML instead of a PDF, or Sonnet timed out).
--
-- Stays NULL on successful parses and on the no_attachment_on_bank_tx /
-- validate-fail / save-fail branches. Only the (haiku→sonnet) double-fail
-- path writes a non-empty value.
ALTER TABLE pending_purchases ADD COLUMN IF NOT EXISTS parse_error TEXT;

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE pending_purchases DROP COLUMN IF EXISTS parse_error;

COMMIT;
