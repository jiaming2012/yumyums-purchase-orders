---
quick_id: 260607-l9m
status: complete
date: 2026-06-07
---

# 260607-l9m — Widen ReceiptSummary numeric fields to float64

## What shipped

Follow-up to 260607-k1n, motivated by today's production log:

```
cannot unmarshal number 85.56 into Go struct field
ReceiptSummary.summary.total_units of type int
```

260607-k1n only widened `ReceiptItem.Quantity`. `ReceiptSummary.TotalUnits` and `.TotalCases` were still `int` and tripped on the same class of LLM output.

### T1 — `bf03285` feat(260607-l9m): widen ReceiptSummary numeric fields int->float64

- `backend/internal/receipt/types.go` — `TotalUnits int` + `TotalCases int` → `float64`
- `backend/internal/receipt/validate.go` — Check 3 boundary uses `int(math.Round(summary.TotalUnits + summary.TotalCases))` for comparison, matching the 260607-k1n pattern
- `backend/internal/receipt/parser_test.go` — `%d` format verb → `%v` for the widened fields

Caller audit (per plan):
- `inventory/handler.go:622,925` references a separate `inventory.PendingPurchase` struct backed by INTEGER columns — **out of scope** (different struct, different storage)
- `receipt/worker.go` — no `.TotalUnits` / `.TotalCases` references; nothing to round
- Test sites in `worker_test.go` — untyped int literals (`TotalUnits: 1`) assign to `float64` cleanly per Go spec; no changes needed

### T2 — `c993506` test(260607-l9m): fixture + regression test for float summary values

- New fixture pair `backend/internal/receipt/testdata/llm_responses/02b_float_summary_units.{txt,expected.json}` — items with `quantity: 40.0` AND summary with `total_units: 85.56` + `total_cases: 3.5`
- `TestParseJSONBody_FloatSummary` in `parser_test.go` — exercises the EXACT failing JSON shape from today's log
- The walker test in `parser_fixtures_test.go` auto-discovers the new fixture (6 sub-tests now, was 5)

## Verification

- `go vet ./internal/receipt/...` — clean
- `go build ./...` — clean
- Full `./internal/receipt/...` package suite — PASS
- `TestParseJSONBody_FloatSummary` — PASS
- 6-fixture walker — PASS
- No regression in 260607-k1n / co0 / fxl tests

## Deferred follow-up

`inventory.PendingPurchase.{TotalUnits,TotalCases}` remain `*int` — these mirror the DB column which is INTEGER. Out of scope per the no-bulk-audit constraint; surfaces here for visibility if a future receipt with decimal units AND those columns becomes load-bearing.

## User verification

1. Click **Retry parse** on the Restaurant Depot row → parse_error chip clears
2. Click **Sync Receipts** → row auto-resolves via Haiku alone (both `40.0` quantity AND `85.56` total_units now unmarshal cleanly)
3. Row appears in confirmed purchases with parsed line items

## Commits

- `bf03285` — feat(260607-l9m): widen ReceiptSummary numeric fields int->float64
- `c993506` — test(260607-l9m): fixture + regression test for float summary values
- `479824c` — chore: merge quick task worktree
