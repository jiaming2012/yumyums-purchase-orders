-- +goose Up
BEGIN;

-- Stable seed row referenced by the "confirm without receipt" path in
-- ConfirmPendingPurchaseHandler (see internal/inventory/handler.go,
-- emptyResolution else-branch). The handler inserts one
-- purchase_line_items row per empty-items confirm, linked to this seed
-- so the COGS aggregate (SUM(quantity * price) in PeriodSummaryHandler)
-- picks up abs(bank_total) and the completeness gate's
-- unlinked_line_item_ids check stays empty (purchase_item_id non-NULL).
--
-- The all-zeros UUID with a 1 in the last position is intentionally
-- non-random — it's a sentinel any developer can recognize and grep
-- for. ON CONFLICT (description) DO NOTHING because purchase_items
-- .description is UNIQUE, making the seed insert idempotent.
INSERT INTO purchase_items (id, description, group_id)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  '(no itemized receipt)',
  NULL
)
ON CONFLICT (description) DO NOTHING;

-- Backfill: every existing purchase_events row that has zero line items
-- and a positive total contributed $0 to cogs_excl_tax before this
-- migration shipped (the bug this migration + the handler else-branch
-- fix). Insert one placeholder line item per orphan event so past
-- weekly reports become accurate retroactively.
--
-- Idempotent: re-running the migration is a no-op because once a
-- placeholder exists on an event, the LEFT JOIN's `pli.id IS NULL`
-- filter no longer matches that event. The CASCADE on
-- purchase_line_items handles cleanup if an event is later deleted.
INSERT INTO purchase_line_items
  (purchase_event_id, purchase_item_id, description, quantity, price, is_case)
SELECT
  pe.id,
  '00000000-0000-0000-0000-000000000001',
  '(no itemized receipt)',
  1,
  pe.total,
  false
FROM purchase_events pe
LEFT JOIN purchase_line_items pli ON pli.purchase_event_id = pe.id
WHERE pli.id IS NULL
  AND pe.total > 0;

COMMIT;

-- +goose Down
BEGIN;

-- Only the seed row is rolled back. Placeholder line_items created by
-- the backfill are not individually tracked (no marker column), and
-- removing them en masse would risk deleting real line items inserted
-- after the migration ran. They will CASCADE-delete if their parent
-- purchase_events row is dropped, which is the only safe cleanup path.
DELETE FROM purchase_items
WHERE id = '00000000-0000-0000-0000-000000000001';

COMMIT;
