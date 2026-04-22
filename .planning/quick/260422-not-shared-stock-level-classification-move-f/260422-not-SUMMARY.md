---
phase: quick
plan: 260422-not
subsystem: inventory
tags: [backend, go, stock-levels, classification, refactor]
dependency_graph:
  requires: []
  provides: [shared-classify-stock-level]
  affects: [inventory-stock-api, purchasing-suggestions, inventory-frontend]
tech_stack:
  added: []
  patterns: [shared-utility-function, tdd]
key_files:
  created:
    - backend/internal/inventory/stock.go
    - backend/internal/inventory/stock_test.go
  modified:
    - backend/internal/inventory/types.go
    - backend/internal/inventory/handler.go
    - backend/internal/purchasing/service.go
    - inventory.html
    - sw.js
decisions:
  - ClassifyStockLevel uses qty==0 as unknown boundary (not low) to preserve existing behavior
  - WHERE clause in GetSuggestions kept in SQL for efficiency; shared function used for labeling only
  - StockLevel field in OrderSuggestion kept lowercase in Go call (matches prior behavior for frontend display)
metrics:
  duration: 15
  completed_date: "2026-04-22"
  tasks: 3
  files: 7
---

# Quick Task 260422-not: Shared Stock Level Classification

**One-liner:** Moved stock level classification from frontend JS into Go backend with ClassifyStockLevel shared function, eliminating the 4-vs-3 PO suggestions count divergence.

## What Was Done

Extracted stock level classification logic into a single Go function used by both the inventory stock endpoint and the purchasing suggestions endpoint. The frontend now reads `level` and `needs_reorder` from the API response instead of computing them locally.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create shared ClassifyStockLevel function with tests | 22db0c8 | stock.go, stock_test.go, types.go |
| 2 | Wire classification into GetStockHandler and GetSuggestions | 6182a5b | handler.go, service.go |
| 3 | Update frontend to consume backend-provided levels | 6a974ac | inventory.html, sw.js |

## Classification Logic (single source of truth)

```go
func ClassifyStockLevel(qty, lowT, highT int) (level string, needsReorder bool) {
    switch {
    case qty == 0:    return "unknown", false
    case qty >= highT: return "high", false
    case qty > lowT:   return "medium", true
    default:           return "low", true
    }
}
```

Default thresholds: low=3, high=10 (COALESCE defaults in SQL).

## Test Results

- `go test ./internal/inventory/ -run TestClassifyStockLevel -v`: 7/7 PASS
- `go build ./...`: clean compilation
- `npx playwright test tests/inventory.spec.js`: 96/97 pass (1 pre-existing failure in simulate-cutoff unrelated to this work)
- Regression test "PO suggestions count matches inventory reorder suggestions": PASS

## Deviations from Plan

None — plan executed exactly as written. The `<=` fix on line 614 noted in the additional context was already present (`sub.current_stock < COALESCE(ig.high_threshold, 10)`) and was preserved as-is.

## Known Stubs

None.

## Self-Check: PASSED

- `backend/internal/inventory/stock.go`: FOUND
- `backend/internal/inventory/stock_test.go`: FOUND
- `backend/internal/inventory/types.go`: FOUND (NeedsReorder field present)
- `backend/internal/inventory/handler.go`: FOUND (ClassifyStockLevel call present)
- `backend/internal/purchasing/service.go`: FOUND (inventory.ClassifyStockLevel call present)
- Commits: 22db0c8, 6182a5b, 6a974ac: FOUND
