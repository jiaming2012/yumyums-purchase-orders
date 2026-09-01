# Merge-intent — `period-summary-visibility` (Card 9, Track C)

Split (b) of roadmap card `pipeline-fail-loud`; closes **B-139**.

## What this card does

The `/period-summary` handler (`backend/internal/inventory/handler.go`,
`PeriodSummaryHandler`) carries ten `slog.Error` calls and NOT ONE success log
line. The only record a payroll week was blocked was the ABSENCE of a report on
the consumer's disk — indistinguishable from a skipped/closed week (B-139). This
card adds one `slog.Info` at the END of a successful `/period-summary` response
emitting: period `from`, `to`, `ready` (the completeness verdict), count of
`pending_review_ids`, count of `unlinked_line_item_ids`.

The sibling endpoint `/menu-cogs` (`backend/internal/recipes/handler.go`,
`MenuCogsHandler`) had no success reader at all; it gets an analogous
`slog.Info` at the end of a successful response emitting `from`, `to`,
`menu_item_count`, and `breakdown` (the query mode).

## Shared files touched

- `backend/internal/inventory/handler.go` — inside `backend/internal/inventory/`,
  NOT shared. (The card's own module.)
- `backend/internal/recipes/handler.go` — the `/menu-cogs` handler lives in the
  **recipes** package, not inventory. This is outside `backend/internal/inventory/`
  but is still an inventory-endpoint file by footprint; the only change is one
  additive `slog.Info` line at the end of the success path. No signature, no
  response-shape, no query change.
- `backend/internal/inventory/period_summary_test.go` — new red-first test
  (`TestPeriodSummary_EmitsVisibilityLog`) + a slog-capture helper. Test file,
  additive.
- `backend/internal/recipes/menu_cogs_test.go` — new red-first test
  (`TestMenuCogs_EmitsVisibilityLog`) + a slog-capture helper. Test file, additive.
- `.night-crew/knowledge/roadmap.md` — note that this half landed; DO NOT flip
  `pipeline-fail-loud` (awaits sibling `toast-sync-fail-loud`, Track A).

No production files outside the two handler files are touched.

## What must survive any merge

- The new `slog.Info` visibility line in `PeriodSummaryHandler` (keys:
  `msg="period-summary served"`, `from`, `to`, `ready`, `pending_review_count`,
  `unlinked_line_item_count`).
- The new `slog.Info` visibility line in `MenuCogsHandler` (keys:
  `msg="menu-cogs served"`, `from`, `to`, `menu_item_count`, `breakdown`).
- Both red-first tests that assert those lines are emitted.

## What is safe to drop

Nothing.
