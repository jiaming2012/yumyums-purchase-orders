---
phase: 260606-0gh
plan: 01
subsystem: inventory/completeness-gate
tags: [bug-fix, sql, tdd, period-summary]
one_liner: "Fix completeness gate to filter pending_purchases by COALESCE(event_date, created_at::Chicago) so late-ingested receipts with in-range event_date are caught"
dependency_graph:
  requires: []
  provides: [correct-pending-gate-filter]
  affects: [GET /api/v1/inventory/period-summary]
tech_stack:
  added: []
  patterns: [COALESCE-null-fallback, TDD-red-green]
key_files:
  modified:
    - backend/internal/inventory/handler.go
    - backend/internal/inventory/period_summary_test.go
decisions:
  - "Filter on COALESCE(event_date, created_at::Chicago::date) — event_date is authoritative, created_at is fallback for rows where worker could not extract it"
  - "ORDER BY uses same COALESCE expression + created_at tiebreaker for deterministic output"
  - "TDD order enforced: tests committed failing (RED) before handler fix (GREEN)"
metrics:
  duration_minutes: 12
  completed_date: "2026-06-06"
  tasks_completed: 2
  files_changed: 2
---

# Phase 260606-0gh Plan 01: Completeness Gate Filters Pending Review Summary

## One-liner

Fix completeness gate to filter `pending_purchases` by `COALESCE(event_date, created_at::Chicago)` so late-ingested receipts with in-range `event_date` are caught and silently-dropped COGS is eliminated.

## What Was Done

### The Bug

`PeriodSummaryHandler` filtered `pending_purchases` via `(created_at AT TIME ZONE 'America/Chicago')::date BETWEEN $1 AND $2`. The receipt worker's 14-day lookback can ingest a May 29 transaction on June 2, setting `event_date='2026-05-29'` and `created_at='2026-06-02'`. A query for the May 25-31 period would miss this row — returning `ready=true` and dropping ~$391 of food cost from weekly COGS.

### The Fix (handler.go lines 1197-1208)

Replaced the WHERE clause with:
```sql
WHERE COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date) BETWEEN $1 AND $2
```

And updated ORDER BY to mirror the expression:
```sql
ORDER BY COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date), created_at
```

Updated the comment block to explain the semantics: `event_date` reflects when the purchase happened; `created_at` is the fallback when the worker couldn't extract it.

### Tests (period_summary_test.go)

**New helper:** `insertPendingPurchaseWithEventDate(t, bankTxID, eventDate, createdAt, reason)` — takes nullable `eventDate` (empty string → SQL NULL) and inserts an unconfirmed/undiscarded pending row. Uses `*string` pointers so nil passes NULL through pgx naturally.

**Modified test:** `ready=false when no_attachment_on_bank_tx pending row in range` — switched from `insertPendingPurchaseWithReason` to the new helper with explicit `event_date="2026-05-28"` to confirm the COALESCE filter catches it when event_date is set in range.

**Four new subtests (event_date × created_at axis):**

| Test name | event_date | created_at | Expected Ready |
|-----------|------------|------------|----------------|
| event_date in range, created_at out of range | 2026-05-29 | 2026-06-02 | false |
| event_date out of range, created_at in range | 2026-05-20 | 2026-05-27 | true |
| NULL event_date, created_at in range | NULL | 2026-05-27 | false |
| NULL event_date, created_at out of range | NULL | 2026-06-05 | true |

All four pass after the fix. The first two were the failing RED gate cases that proved the old filter was broken.

## Commits

| Hash | Type | Description |
|------|------|-------------|
| 8c64046 | test | Add failing event_date × created_at axis tests (RED gate) |
| cf959bd | fix | Filter pending_purchases by COALESCE(event_date, created_at::Chicago) |

## Deviations from Plan

None — plan executed exactly as written. TDD order observed (tests committed before fix).

## Known Stubs

None.

## Threat Flags

None — no new network endpoints or auth paths introduced.

## Deferred Issues

One pre-existing test failure observed during execution (out of scope):

`TestPeriodSummary/end-to-end_empty-items_confirm_increments_cogs` fails with `column "display_name" of relation "users" does not exist`. This is a test DB schema mismatch (the `hq_test` instance at `100.70.200.55:5433` is missing a migration that adds `display_name` to `users`). The test was failing before this task's changes and was not introduced by this task. Logged for follow-up.

## Self-Check: PASSED

- handler.go: FOUND
- period_summary_test.go: FOUND
- Commit 8c64046 (RED gate tests): FOUND
- Commit cf959bd (handler fix): FOUND
