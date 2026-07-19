---
status: partial
phase: 21-cogs-in-sales-processor-report-receipt-completeness-gate-bef
source: [21-VERIFICATION.md]
started: 2026-06-02T18:00:00Z
updated: 2026-06-02T18:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Live curl 503 path against running server with HQ_INVENTORY_SERVICE_TOKEN unset
expected: Start backend with env var unset, then `curl http://localhost:8080/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31` returns 503 + `{"error":"service_token_not_configured"}`, server log shows the WARNING line at startup.
result: [pending]

### 2. Live curl 401 path against running server
expected: Start backend with `HQ_INVENTORY_SERVICE_TOKEN=test-token`, then curl WITHOUT Authorization header returns 401 + `{"error":"unauthorized"}`. Same again with a wrong bearer token also returns 401.
result: [pending]

### 3. Live curl 200 path against running server
expected: Start backend with `HQ_INVENTORY_SERVICE_TOKEN=test-token`, populate hq DB with at least one purchase_event, then curl with `Authorization: Bearer test-token` returns 200 + JSON matching PeriodSummary contract (from/to/cogs_excl_tax/cogs_incl_tax/purchase_event_count/completeness).
result: [pending]

### 4. (Optional) Re-run integration tests against hq_test DB
expected: `DB_TEST_URL='postgres://yumyums:yumyums@192.168.8.164:5433/hq_test?sslmode=disable' go test ./internal/inventory -run TestPeriodSummary -v` shows 6 PASS subtests.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
