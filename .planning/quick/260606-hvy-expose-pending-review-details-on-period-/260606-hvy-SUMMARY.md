---
phase: 260606-hvy
plan: 01
subsystem: api
tags: [go, postgres, period-summary, inventory, sales-processor, pending-purchases, completeness-gate]

# Dependency graph
requires:
  - phase: 21-cogs-in-sales-processor-report-receipt-completeness-gate-bef
    provides: GET /api/v1/inventory/period-summary endpoint, CompletenessBlock struct, HQ_INVENTORY_SERVICE_TOKEN auth
  - phase: 260606-0gh
    provides: pending_purchases period filter uses COALESCE(event_date, created_at::Chicago)
  - phase: 260606-hew
    provides: pending_purchases.vendor populated from Mercury BankDescription (NOT NULL guarantee)
provides:
  - pending_review_details parallel array on /period-summary completeness block
  - PendingReviewDetail struct (id, bank_tx_id, vendor, event_date, bank_total, reason)
  - Parity-by-construction invariant — single SELECT + single scan loop guarantees pending_review_ids[i] == pending_review_details[i].id
affects: [sales-processor weekly payroll failure messaging, downstream pending-review renderers]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sibling parity arrays — write IDs and details from one SELECT/scan loop to make order-equality structural"
    - "Service-token-friendly endpoint extension — extend existing handler instead of adding a second cookie-gated round-trip"

key-files:
  created: []
  modified:
    - backend/internal/inventory/types.go
    - backend/internal/inventory/handler.go
    - backend/internal/inventory/period_summary_test.go

key-decisions:
  - "Parity guaranteed by construction (single SELECT, single scan loop, both slices appended in same iteration) — not by a runtime assertion"
  - "Reason as *string with omitempty so NULL serialises as absent JSON key, not the noisy {reason:null}"
  - "Reused existing log strings ('PeriodSummary pending query/scan/rows.Err') because sales-processor alerts grep on them"

patterns-established:
  - "Sibling parity arrays in JSON responses — append ID and detail from one row scan to lock order-equality at write time"

requirements-completed: [260606-hvy]

# Metrics
duration: 7min
completed: 2026-06-06
---

# Phase 260606-hvy Plan 01: Expose pending_review_details on /period-summary Summary

**Added a parallel `pending_review_details` array to /period-summary's completeness block exposing bank_tx_id, vendor, event_date, bank_total, and (omitempty) reason per pending review — built from one SELECT so order matches `pending_review_ids` by construction**

## Performance

- **Duration:** ~7 min
- **Started:** 2026-06-06T16:57:29Z
- **Completed:** 2026-06-06T17:03:28Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments
- New `PendingReviewDetail` struct + `CompletenessBlock.PendingReviewDetails` field with exact JSON tags sales-processor will consume verbatim
- Pending-purchases query expanded to return all six fields in one row; scan loop builds both `pendingIDs` and `pendingDetails` from the same iteration so parity is structural (no runtime assertion needed)
- Six new subtests pin shape, parity, field population, Chicago-cast fallback, empty-string vendor passthrough, NULL-reason omitempty, and empty-period `[]` (not null) — all pass against the remote test DB
- Build + vet clean, no changes to `Ready` calculation, no schema migration, `pending_review_ids` semantics unchanged for existing consumers

## Task Commits

1. **Task 1: Add `PendingReviewDetail` struct + extend `CompletenessBlock`** — `1c260f0` (feat)
2. **Task 2: Expand pending query + wire `pendingDetails` into response** — `31f82a7` (feat)
3. **Task 3: Append six `pending_review_details` subtests** — `2e33e81` (test)

_Note: This was a `tdd="true"` task by frontmatter but the plan ordered implementation (Tasks 1-2) before tests (Task 3). The implementation pre-existed when tests were written, so tests passed on first authoring — no RED commit was required because the gate was already green at commit time. This is a known pattern for "additive endpoint extension" plans where the plan authors carefully sequence types → handler → tests in one wave._

## Files Created/Modified
- `backend/internal/inventory/types.go` — Added `PendingReviewDetails []PendingReviewDetail` to `CompletenessBlock`; added new exported `PendingReviewDetail` struct (id, bank_tx_id, vendor, event_date, bank_total, reason)
- `backend/internal/inventory/handler.go` — Expanded `PeriodSummaryHandler` step 3 SELECT to 6 columns; rewrote scan loop to populate both ID and detail slices from one row; wired `PendingReviewDetails: pendingDetails` into response literal between `PendingReviewIDs` and `UnlinkedLineItemIDs`
- `backend/internal/inventory/period_summary_test.go` — Added `strings` import; appended six new subtests under `TestPeriodSummary` covering parity, field population, Chicago-cast event_date fallback, empty-string vendor, NULL reason omitempty, and empty-period `[]` serialisation

## Decisions Made
- **Parity by construction, not assertion** — both slices appended in the same scan iteration from `d.ID` and `d`. Eliminates a class of "they drifted apart somehow" bugs.
- **`reason *string` + `omitempty`** — NULL reason → nil pointer → key absent from JSON. Cleaner downstream rendering than `{reason:null}`.
- **Preserved exact log strings** — `"PeriodSummary pending query: %v"` etc. unchanged because sales-processor's log-tailing alerts grep on these. Rule: don't rename observable surfaces during additive plans.
- **Reused existing `callHandler` helper** for typed-struct subtests, and the inline `httptest.NewRecorder()` pattern (mirroring the `tracked_bank_tx_ids: empty period renders [] not null` subtest at line 902) for raw-body assertions.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] vendor=NULL test case updated to vendor=''**
- **Found during:** Task 3 verification (running new subtests against remote test DB)
- **Issue:** The plan's `pending_review_details vendor=NULL serializes as empty string` subtest tried `INSERT INTO pending_purchases (..., vendor, ...) VALUES (..., NULL, ...)` but Postgres rejected it: `null value in column "vendor" of relation "pending_purchases" violates not-null constraint`. The `pending_purchases.vendor` column is NOT NULL at the schema level — almost certainly tightened by quick task 260606-hew which always populates vendor from Mercury BankDescription. The plan's `COALESCE(vendor, '')` in the SQL is now defensive code that no live row will ever exercise, but it's still correct.
- **Fix:** Changed the test to insert `vendor=''` (empty string) — the closest observable analogue. Updated the subtest name to `vendor='' serializes as empty string` and added an in-test comment explaining the schema constraint and the COALESCE rationale.
- **Files modified:** `backend/internal/inventory/period_summary_test.go`
- **Verification:** Subtest passes against the remote test DB (Tailscale-hosted Windows box). The COALESCE in handler.go is left in place as defensive code.
- **Committed in:** `2e33e81` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (1 bug — outdated assumption about column nullability)
**Impact on plan:** No code-path change. The handler's `COALESCE(vendor, '')` remains as a defensive guard; the test now reflects current schema reality.

## Issues Encountered

- **Pre-existing failure in unrelated subtest** — `TestPeriodSummary/end-to-end_empty-items_confirm_increments_cogs` fails against the remote test DB at `period_summary_test.go:484` (`insertTestUser`) with `column "display_name" of relation "users" does not exist`. This is schema drift between the test helper (uses `display_name`) and the remote `hq_test` DB. **Out of scope** — not touched by this plan, not caused by these changes (helper at line 214 is unmodified). Worth re-running migrations on the test DB or refreshing the helper to match current `users` schema in a follow-up.

## Deferred Issues

- `TestPeriodSummary/end-to-end_empty-items_confirm_increments_cogs` schema drift — pre-existing, see "Issues Encountered" above. Recommended follow-up: drop+recreate `hq_test` and re-run migrations from scratch, OR update `insertTestUser` to match the post-Phase-11 users schema (likely `display_name` was migrated to `first_name`/`last_name`/`nickname`-derived column).

## User Setup Required

None — purely additive API change. Field rolls out the next time the backend is deployed; sales-processor can begin consuming `pending_review_details` immediately and existing consumers of `pending_review_ids` continue working unchanged.

## Next Phase Readiness

- Sales-processor can now render rich pending-review failure messages from one call to `/period-summary` (no second round trip to `/purchases/pending`).
- The pre-existing `display_name` test failure should be addressed before the next plan that needs `insertTestUser`.

---
*Phase: 260606-hvy-expose-pending-review-details-on-period-*
*Completed: 2026-06-06*

## Self-Check: PASSED

- File `backend/internal/inventory/types.go`: FOUND
- File `backend/internal/inventory/handler.go`: FOUND
- File `backend/internal/inventory/period_summary_test.go`: FOUND
- File `.planning/quick/260606-hvy-expose-pending-review-details-on-period-/260606-hvy-SUMMARY.md`: FOUND
- Commit `1c260f0` (Task 1 — types.go): FOUND
- Commit `31f82a7` (Task 2 — handler.go): FOUND
- Commit `2e33e81` (Task 3 — period_summary_test.go): FOUND
