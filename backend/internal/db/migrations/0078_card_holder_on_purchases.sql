-- +goose Up
BEGIN;

-- ===========================================================================
-- 0078 — who swiped which card. The Purchases tab showed "Mercury: COGS" and
-- nothing about the person: with two active cards in use (and more crew
-- coming) the operator could not tell whose swipe a pending charge was.
--
-- Mercury's transaction JSON carries only `cardId`; the holder's name and the
-- last four digits come from GET /account/{id}/cards. The receipt worker
-- resolves them at ingest (and refreshes existing rows inside its lookback
-- window, the same IS DISTINCT FROM pattern as mercury_category) and stores
-- the two display facts here so the API never has to call Mercury to render
-- a card. Nullable: rows that pre-date this column, transactions that are not
-- card swipes, and a cards lookup that failed all leave them NULL and the UI
-- simply omits the label.
-- ===========================================================================

ALTER TABLE pending_purchases
  ADD COLUMN card_holder TEXT,
  ADD COLUMN card_last4  TEXT;

ALTER TABLE purchase_events
  ADD COLUMN card_holder TEXT,
  ADD COLUMN card_last4  TEXT;

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE purchase_events
  DROP COLUMN card_holder,
  DROP COLUMN card_last4;

ALTER TABLE pending_purchases
  DROP COLUMN card_holder,
  DROP COLUMN card_last4;

COMMIT;
