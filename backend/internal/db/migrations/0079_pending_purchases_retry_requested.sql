-- +goose Up
BEGIN;

-- ===========================================================================
-- 0079 — a retry request stops destroying the evidence it was asking about.
--
-- Until now "Retry parse" re-armed a row by making it LOOK unparsed:
--
--     UPDATE pending_purchases
--        SET items = '[]'::jsonb,
--            reason = 'Receipt could not be parsed automatically',
--            parse_error = NULL
--
-- because the worker's re-parse gate (receipt/worker.go, parseFailedRetry)
-- recognised an eligible row by `parse_error IS NULL AND items = []`. Two
-- columns were doing double duty: carrying the parse result AND signalling
-- "please try again".
--
-- The cost landed on the operator. Pressing Retry parse deleted the only
-- record of WHY the receipt failed, so the card fell back to the bare
-- "Receipt could not be parsed automatically" with no cause; it deleted any
-- line items the parse HAD extracted, so the review form opened empty; and
-- because inventory.html gates the button on those same two fields, the
-- button removed itself. A row could be retried exactly once, and doing so
-- made it strictly less reviewable.
--
-- retry_requested_at separates the signal from the data: the worker gates on
-- this column and clears it when it reprocesses the row, while parse_error
-- and items survive to be shown and pre-filled.
-- ===========================================================================
ALTER TABLE pending_purchases ADD COLUMN IF NOT EXISTS retry_requested_at TIMESTAMPTZ;

-- Partial index: the worker only ever asks "is a retry pending on this open
-- row?", so index only the rows that can answer yes.
CREATE INDEX IF NOT EXISTS pending_purchases_retry_requested_idx
  ON pending_purchases (retry_requested_at)
  WHERE retry_requested_at IS NOT NULL;

COMMIT;

-- +goose Down
BEGIN;

DROP INDEX IF EXISTS pending_purchases_retry_requested_idx;
ALTER TABLE pending_purchases DROP COLUMN IF EXISTS retry_requested_at;

COMMIT;
