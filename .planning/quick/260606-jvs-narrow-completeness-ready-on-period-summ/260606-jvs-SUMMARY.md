---
phase: 260606-jvs
plan: 01
subsystem: backend-inventory
tags: [period-summary, completeness, cogs, pending-purchases, mercury-category]
requires:
  - HQ_INVENTORY_SERVICE_TOKEN env var (Phase 21)
  - cogsAllowlist allowlist injected into PeriodSummaryHandler (260605-v0n)
  - pending_review_details parallel array on /period-summary (260606-hvy)
provides:
  - Narrowed completeness.ready gate: blocks IFF (∃ pending in period with mercury_category in cogsAllowlist AND reason='no_attachment_on_bank_tx') OR (∃ unlinked line item)
  - pending_review_ids + pending_review_details narrowed to blocking rows only
  - cogs_excl_tax / cogs_incl_tax / by_vendor rolled to include non-blocking eligible pending (food category + receipt attached but parse-failed) at ABS(bank_total)
  - pending_purchases.mercury_category column + worker populate + refresh
affects:
  - sales-processor weekly payroll: stops blocking on non-food card swipes (Amazon) and food pending whose receipt is attached but parse-failed (Save A Lot). Only blocker now is food + no receipt (Restaurant Depot)
  - sales-processor PDF: per-vendor totals + COGS aggregate already include Save A Lot's $19.28 even before operator triages the parsing failure
tech-stack:
  added: []
  patterns:
    - "Goose migration mirroring 0065"
    - "UPDATE … IS DISTINCT FROM idempotent refresh pattern (mirroring purchase_events)"
    - "CTE-based UNION with outer GROUP BY for confirmed + matched-pending + unmatched-pending vendor merge"
key-files:
  created:
    - backend/internal/db/migrations/0066_mercury_category_on_pending_purchases.sql
  modified:
    - backend/internal/receipt/worker.go
    - backend/internal/inventory/handler.go
    - backend/internal/inventory/period_summary_test.go
decisions:
  - "Default mercury_category in test helpers to 'COGS' so pre-existing pending-row test sites retain blocking semantics post-narrowing"
  - "Removed 'pending_review_details reason=NULL omitted from JSON' subtest — NULL-reason rows are unreachable in pending_review_details under the narrowed gate; omitempty tag remains structural on the struct"
  - "Updated 'populates vendor/event_date/bank_total/reason' to assert on 'no_attachment_on_bank_tx' (the only surfaceable reason post-narrowing) instead of 'tax_mismatch'"
metrics:
  duration_minutes: 22
  completed_date: 2026-06-06
---

# Phase 260606-jvs Plan 01: Narrow completeness.ready on /period-summary Summary

## One-liner

Narrowed `/period-summary` `completeness.ready` to fire `false` only on real food-category card swipes without receipts, and rolled non-blocking food pending (parse-failed-but-attached) into `cogs_excl_tax` + `by_vendor` at `ABS(bank_total)` — stops blocking weekly payroll on Amazon refunds and Save A Lot parse failures while keeping Restaurant Depot's no-receipt charge as the legitimate blocker.

## What changed

| Task | Subject | Commit |
|---|---|---|
| 1 | 0066 migration: `pending_purchases.mercury_category TEXT` (nullable, goose Up/Down) | `bd7e9f3` |
| 2 | worker.go: refresh pass for pending_purchases.mercury_category (mirrors purchase_events refresh) + insertPendingPurchase writes 10 columns (mercury_category derived nil-safe from tx.CategoryData) | `0351f12` |
| 3 | handler.go: three SQL rewrites — narrow pending-IDs query, UNION pending into cogs aggregate, UNION pending into by_vendor (3-CTE confirmed + matched + unmatched with outer GROUP BY) | `d41faef` |
| 4 | period_summary_test.go: 10 new subtests for the 2×2 truth table + COGS roll-up + by_vendor merge; audited and patched pre-existing pending-row subtests | `b324b04` |

## Truth table pinned by tests

| # | Sub-test name | Category | Reason | Blocks? | In COGS? |
|---|---|---|---|---|---|
| 1 | `case_a_food_no_attachment_blocks` | COGS | `no_attachment_on_bank_tx` | YES (only blocker) | no |
| 2 | `case_b_food_parse_failed_rolls_into_cogs` | COGS | parse-failed | no | YES @ ABS(bank_total) |
| 3 | `case_c_non_food_no_attachment_does_not_block_or_roll` | non-COGS | `no_attachment_on_bank_tx` | no | no |
| 4 | `case_d_non_food_parse_failed_does_not_block_or_roll` | non-COGS | parse-failed | no | no |
| 5 | `null_mercury_category_with_no_attachment_does_not_block` | NULL | `no_attachment_on_bank_tx` | no (NULL fails `= ANY`) | no |
| 6 | `date_filter_still_applies_to_blocking_row_out_of_period` | COGS | `no_attachment_on_bank_tx` | no (out of period) | no |
| 7 | `pending_review_details_parity_under_narrowed_gate` | COGS | `no_attachment_on_bank_tx` | ids+details parity, both length 1 | — |
| 8 | `case_b_by_vendor_match_merges_into_vendor_row` | COGS (RD confirmed + SAL pending) | parse-failed | no | `cogs_excl_tax==119`, RD + SAL rows with real vendor_id |
| 9 | `case_b_by_vendor_unmatched_renders_with_empty_vendor_id` | COGS | parse-failed | no | unmatched row with `vendor_id==""` |
| 10 | `case_b_by_vendor_vendor_name_fuzz_joins_case_and_trim_insensitive` | COGS | parse-failed | no | joins via `LOWER(TRIM())`, no duplicate |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug, test collateral] Pre-existing tests would have started failing under the narrowed gate**

- **Found during:** Task 4
- **Issue:** Plan flagged this risk explicitly: existing `TestPeriodSummary` subtests that insert pending rows expecting them to surface in `pending_review_ids` would now silently exclude those rows since none had `mercury_category` set and most did not use `reason='no_attachment_on_bank_tx'`.
- **Fix:** Threaded the right defaults through the two shared helpers, and patched two hand-rolled INSERTs + two subtests that explicitly relied on the prior shape. Details:
  - `insertPendingPurchase` (line 163): now defaults `mercury_category='COGS'` + `reason='no_attachment_on_bank_tx'`. Used by `ready=false when pending purchase in range is unconfirmed` (line 306, blocking-row assertion preserved) and `discarded pending purchase does NOT block ready` (line 354, discarded_at filter independent of category/reason, still passes).
  - `insertPendingPurchaseWithEventDate` (line 242): now defaults `mercury_category='COGS'` and (when caller passes `""`) defaults reason to `'no_attachment_on_bank_tx'`. All 12 call sites verified — the two `ready=true ... out of range` subtests stay green because their assertion is `len(PendingReviewIDs)==0` and the rows remain excluded by the existing date filter.
  - `pending_review_details populates vendor/event_date/bank_total/reason` (line 1076): hand-rolled INSERT was using `reason='tax_mismatch'` with no category — that combination is now non-blocking, so the row would no longer appear in `pending_review_details`. Patched to use `reason='no_attachment_on_bank_tx'` + `mercury_category='COGS'` and updated the assertion accordingly. The serialisation shape the test pins (Reason pointer non-nil with the inserted text) is preserved.
  - `pending_review_details vendor='' serializes as empty string` (line 1138): hand-rolled INSERT had no reason and no category. Patched to add both so the row remains blocking and surfaces in details.
  - `pending_review_details reason=NULL omitted from JSON` (line 1166): under the narrowed gate, NULL-reason rows are unreachable in `pending_review_details` (the WHERE clause requires `reason = 'no_attachment_on_bank_tx'`). Removed the subtest and documented the removal inline. The `json:"reason,omitempty"` tag on `PendingReviewDetail.Reason` remains as a structural property of the struct — not exercisable through `/period-summary` post-narrowing.
- **Files modified:** `backend/internal/inventory/period_summary_test.go`
- **Commit:** `b324b04`

### Rule 2 — Add missing critical functionality

None. Plan covered all required correctness mitigations.

### Rule 3 — Auto-fix blocking issues

None.

### Rule 4 — Architectural changes asked

None.

## Verification

Build + tests run from `backend/`:

```
$ go build ./...                     # clean, no output
$ go vet ./internal/inventory/... ./internal/receipt/...    # clean
$ go test ./internal/inventory/... ./internal/receipt/... -count=1
ok  	github.com/yumyums/hq/internal/inventory	0.572s
ok  	github.com/yumyums/hq/internal/receipt	0.887s
```

`TestPeriodSummary` is gated on `DB_TEST_URL` (skips when unreachable). On this worktree the env var was not set, so the subtests **skipped cleanly with no compile errors**. The user must run the full subtest suite against a reachable Postgres after applying the 0066 migration:

```bash
cd backend && DB_TEST_URL=postgres://… go test ./internal/inventory/ -run TestPeriodSummary -count=1 -v
```

The migration file is in `backend/internal/db/migrations/0066_…sql`; per executor constraint it was NOT applied here — the user runs migrations against the remote dev Postgres separately.

## End-to-end sanity (deferred — needs reachable HQ deploy + seeded data)

Per HANDOFF.md §Verification:

```bash
curl -s -H "Authorization: Bearer $HQ_INVENTORY_SERVICE_TOKEN" \
  "$HQ_BASE_URL/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31" \
  | python3 -m json.tool | grep -E '(ready|pending_review_ids|pending_review_details|cogs_excl_tax|by_vendor)'
```

Expected post-deploy: `ready: false`, `pending_review_ids` has only the RD id, `cogs_excl_tax` includes Save A Lot's $19.28, `by_vendor` lists Save A Lot.

## Files

- Created: `backend/internal/db/migrations/0066_mercury_category_on_pending_purchases.sql`
- Modified: `backend/internal/receipt/worker.go`, `backend/internal/inventory/handler.go`, `backend/internal/inventory/period_summary_test.go`

## Self-Check: PASSED

- Created file present: `backend/internal/db/migrations/0066_mercury_category_on_pending_purchases.sql` (verified via Bash `ls`)
- All 4 task commits found in `git log`: `bd7e9f3`, `0351f12`, `d41faef`, `b324b04` (verified)
- `go build ./...` clean
- `go test ./internal/inventory/... ./internal/receipt/...` passes (TestPeriodSummary skips when DB unreachable; subtest compile verified via `go test -run=NONE`)
