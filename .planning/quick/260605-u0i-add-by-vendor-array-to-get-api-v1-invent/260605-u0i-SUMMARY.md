---
phase: 260605-u0i
plan: 01
subsystem: inventory
tags: [backend, period-summary, cogs, by-vendor, sales-processor-contract]
dependency_graph:
  requires:
    - GET /api/v1/inventory/period-summary (existing — Phase 21)
    - purchase_events / purchase_line_items / vendors schema (no changes)
  provides:
    - by_vendor[] array on /period-summary response (purely additive)
  affects:
    - sales-processor weekly payroll PDF (per-vendor table now renderable)
tech_stack:
  added: []
  patterns:
    - "Mirror the pendingIDs scan-loop pattern for new SQL block"
    - "Initialize result slice as []T{} (never nil) so JSON renders [] not null"
    - "Correlated subquery for tax so join cardinality doesn't multiply totals"
key_files:
  created:
    - .planning/quick/260605-u0i-add-by-vendor-array-to-get-api-v1-invent/260605-u0i-SUMMARY.md
  modified:
    - backend/internal/inventory/types.go
    - backend/internal/inventory/handler.go
    - backend/internal/inventory/period_summary_test.go
decisions:
  - "Combined shape + sums + order assertions into ONE multi-vendor subtest (shared fixture); SPEC §3 lists them as 'areas' not separate tests"
  - "Used direct SQL INSERT (not insertEventAndLine, not confirm handler) for the zero-line-items regression subtest — guards future code paths that skip the placeholder"
  - "Migration N/A — purely additive response field, no schema work"
metrics:
  duration: "~2m"
  completed: "2026-06-06"
  tasks_completed: 2
  files_modified: 3
  commits: 2
---

# Phase 260605-u0i Plan 01: Add by_vendor Array to /period-summary Summary

Per-vendor COGS breakdown added to GET /api/v1/inventory/period-summary so sales-processor's weekly payroll PDF can render a "Restaurant Depot $X, Save-A-Lot $Y, …" table — purely additive, no breaking change.

## What Shipped

- **New struct:** `VendorCOGS` in `backend/internal/inventory/types.go:169` with fields `VendorID`, `VendorName`, `TotalExclTax`, `TotalInclTax`, `TripCount` and JSON tags per SPEC §1.
- **New response field:** `PeriodSummary.ByVendor []VendorCOGS` (json `by_vendor`) placed between `PurchaseEventCount` and `Completeness` (matches documented JSON layout).
- **New SQL block:** `backend/internal/inventory/handler.go:1143-1192` — runs between the existing step-1 aggregate and the step-2 pending-IDs query. Uses LEFT JOIN purchase_line_items + correlated subquery for tax + ORDER BY total_excl_tax DESC, v.name ASC.
- **Initialization:** `byVendor := []VendorCOGS{}` (never nil) so empty periods render `"by_vendor":[]` not `"by_vendor":null`.
- **Step comment renumbering:** PeriodSummaryHandler comments now read 1/2/3/4 in order (aggregate / by-vendor / pending-IDs / unlinked-line-items).
- **Tests:** Three new subtests appended to `TestPeriodSummary`:
  - `by_vendor: shape + sums match + order` — 4 vendors incl. tied pair (Acme/Zeta at $10.00 each) exercising name-ASC tiebreaker; asserts Σ row.TotalExclTax == COGSExclTax within 0.01.
  - `by_vendor: empty period renders [] not null` — re-encodes the response and asserts literal substring `"by_vendor":[]`.
  - `by_vendor: zero-line-items event still appears (regression)` — direct INSERT into purchase_events (bypassing both `insertEventAndLine` and the confirm handler) and asserts the vendor still appears with `TotalExclTax=0`, `TripCount=1`.

## Tasks

| # | Name | Commit | Files |
|---|------|--------|-------|
| 1 | Add VendorCOGS struct + per-vendor SQL block + scan loop | `518a395` | `backend/internal/inventory/types.go`, `backend/internal/inventory/handler.go` |
| 2 | Add by_vendor subtests to TestPeriodSummary | `7579fe7` | `backend/internal/inventory/period_summary_test.go` |

## Verification

**Build + vet (passed in sandbox):**

```
$ cd backend && go build ./...
(exit 0)

$ cd backend && go vet ./internal/inventory/...
(exit 0)
```

**Tests (RED-by-construction in sandbox — DB_TEST_URL not reachable, same precedent as pk1/q7b):**

```
$ cd backend && go test ./internal/inventory/ -run TestPeriodSummary -count=1 -v
=== RUN   TestPeriodSummary
    period_summary_test.go:228: DB_TEST_URL not reachable; skipping integration test
--- SKIP: TestPeriodSummary (0.00s)
PASS
ok  	github.com/yumyums/hq/internal/inventory	0.445s
```

This matches the established sandbox behavior (`pk1`, `q7b`) — the test code compiles and `go vet` is clean. The new subtests will run on the next CI/dev push where DB_TEST_URL points at hq_test. The 3 new subtests are designed to exercise:

1. Multi-vendor shape, sums-match invariant (rounding tolerance 0.01), and the order contract (DESC spend, ASC name tiebreaker).
2. JSON-level `"by_vendor":[]` (not `null`) when the period has no events.
3. Zero-line-items regression: a vendor with a purchase_events row but no purchase_line_items still surfaces (`TotalExclTax=0`, `TripCount=1`).

## Migration

N/A — purely additive on the response. No schema, no migration, no version bump, no feature flag. Existing consumers ignoring `by_vendor` are unaffected.

## Downstream Impact

Once this ships, sales-processor will start rendering the per-vendor COGS table in its weekly payroll PDF — see `sales-processor/service/external/hq.go` and `sales-processor/docs/payroll-report.md`. Until then the consumer degrades gracefully: the COGS summary (food cost %, gross profit) still prints; only the per-vendor table is omitted.

## Deviations from Plan

None — plan executed exactly as written. Both `go build` and `go vet` passed on first compile; no auto-fixes (Rules 1-3) triggered.

## Self-Check: PASSED

- `backend/internal/inventory/types.go` — FOUND: `type VendorCOGS struct` at line 169, `ByVendor` field at line 162
- `backend/internal/inventory/handler.go` — FOUND: `byVendor := []VendorCOGS{}` at line 1149, `ByVendor: byVendor,` at line 1265, step comments 1/2/3/4 at lines 1114/1143/1194/1227
- `backend/internal/inventory/period_summary_test.go` — FOUND: 3 new `t.Run("by_vendor:…")` blocks
- Commit `518a395` — FOUND in `git log`
- Commit `7579fe7` — FOUND in `git log`
- `go build ./...` — exit 0
- `go vet ./internal/inventory/...` — exit 0
