---
phase: quick
plan: 260421-iug
subsystem: inventory
tags: [tab-order, inventory, ui]
dependency_graph:
  requires: []
  provides: [inventory-tab-order-stock-before-trends]
  affects: [inventory.html, tests/inventory.spec.js, CLAUDE.md]
tech_stack:
  added: []
  patterns: []
key_files:
  modified:
    - inventory.html
    - tests/inventory.spec.js
    - CLAUDE.md
    - sw.js
decisions:
  - Stock moved to tab 2 (before Trends) to improve daily usability since Stock is used frequently and Trends is a coming-soon placeholder
metrics:
  duration: 7
  completed_date: 2026-04-21
---

# Quick Task 260421-iug: Move Stock Tab Before Trends in Inventory

**One-liner:** Swapped Stock (tab 2) and Trends (tab 3) in inventory.html so frequently-used Stock appears before the coming-soon Trends placeholder.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Swap Stock and Trends tab positions in inventory.html | 485473c | inventory.html |
| 2 | Update test assertions and CLAUDE.md for new tab order | ab6806f | tests/inventory.spec.js, CLAUDE.md, sw.js |

## What Changed

**inventory.html:**
- Tab buttons: t2 now reads "Stock", t3 now reads "Trends"
- Section s2 now contains stock content (`reorder-section`, `stock-list`); s3 now contains trends content (`trends-container`)
- `show()` function: `if(n===3){loadStock()}` changed to `if(n===2){loadStock()}`
- `render()` function: `ACTIVE_TAB===2` dispatches to `renderStock()`, `ACTIVE_TAB===3` dispatches to `renderTrends()`
- Click delegation guard: `ACTIVE_TAB!==3` changed to `ACTIVE_TAB!==2` (stock override form handler)

**tests/inventory.spec.js:**
- Tab label assertions updated: `#t2` expects "Stock", `#t3` expects "Trends"
- Test title updated: "shows 4 tabs: Purchases, Stock, Trends, Cost"
- All 13 Stock tab clicks changed from `#t3` to `#t2`
- Trends tab click changed from `#t2` to `#t3`
- Trends section assertions changed from `#s2` to `#s3`

**CLAUDE.md:**
- Line 33: `5-tab layout (Purchases / Stock / Trends / Cost / Setup)`
- Line 48: `Purchases (...), Stock (levels + reorder suggestions), Trends (coming soon), ...`

## Verification

All 86 inventory Playwright tests pass with the new tab order.

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None introduced by this task.

## Self-Check: PASSED

- inventory.html modified: FOUND
- tests/inventory.spec.js modified: FOUND
- CLAUDE.md modified: FOUND
- Commit 485473c: FOUND
- Commit ab6806f: FOUND
