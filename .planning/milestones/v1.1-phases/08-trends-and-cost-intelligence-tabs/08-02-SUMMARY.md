---
phase: 08-trends-and-cost-intelligence-tabs
plan: 02
subsystem: ui
tags: [chart.js, inventory, cost-intelligence, playwright, pwa]

# Dependency graph
requires:
  - phase: 08-01
    provides: renderTrends() with Chart.js bar/doughnut/line charts, chip filter, trends-container DOM
provides:
  - renderCost() with Menu Items and Ingredients sub-tabs
  - calcAvgUnitPrice(), calcMenuItemCost(), calcIngredientUsage() helpers
  - Expandable menu item cards showing ingredient proportion tables and margin
  - Ingredient reverse-lookup cards showing which menu items use each ingredient
  - Category drill-down on Trends tab (purchase line items per tag with name, vendor, qty, case badge)
  - Time filter dropdown on Trends and Cost tabs (All Time, Last Week, 1/3/6 Months, Last Year)
  - Tag chip filter and time filter on Cost tab (mirrors Trends)
  - Chart.js click handler on bar/doughnut charts for drill-down
  - Shared buildTimeFilterHtml() and buildChipBarHtml() helpers
  - Playwright E2E tests: TRND-01, TRND-02, TRND-03, TRND-04, COST-01, COST-02, COST-03, INTG-02
  - SW cache bumped to yumyums-v46
affects: [09-future-phases, metabase-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - calcAvgUnitPrice() uses weighted average: sum(price*qty)/sum(qty) across all purchases for a group
    - calcMenuItemCost() uses equal ingredient proportions: 1/ingredientGroupIds.length
    - Cost sub-tabs share .sub-tabs CSS class with Trends sub-tabs (reuse pattern)
    - EXPANDED_MENU_ITEMS/EXPANDED_INGREDIENTS state dict keyed by ID for expand/collapse

key-files:
  created: []
  modified:
    - inventory.html
    - tests/inventory.spec.js
    - sw.js

key-decisions:
  - "Equal ingredient proportion model (1/n per ingredient) keeps Cost tab mock calculations simple and intuitive"
  - "INTG-02 Metabase swap is TODO comment only — no runtime switching infrastructure needed at mock stage"
  - "Ingredient sub-tab shows return ratio (Xx return) as primary metric for ingredient ROI"

patterns-established:
  - "Cost event delegation on #s4 — separate from #s2 Trends listener, keeps click handlers isolated"
  - "renderCost() rebuilds entire #cost-container innerHTML on each state change (same pattern as renderTrends)"

requirements-completed: [COST-01, COST-02, COST-03, INTG-02]

# Metrics
duration: 15min
completed: 2026-04-14
---

# Phase 8 Plan 02: Trends and Cost Intelligence Tabs Summary

**Cost Intelligence tab with menu item cost breakdown (ingredient proportions + margin) and ingredient reverse-lookup (usage by menu item with ROI), plus full Playwright test coverage for TRND-01 through COST-03**

## Performance

- **Duration:** ~30 min (including human-verify and post-approval enhancements)
- **Started:** 2026-04-14T19:10:00Z
- **Completed:** 2026-04-14T20:26:31Z
- **Tasks:** 3 (all complete including human-verify checkpoint)
- **Files modified:** 3

## Accomplishments
- renderCost() implemented with two sub-tabs: Menu Items (COST-01/02) and Ingredients (COST-03)
- Cost calculation helpers: calcAvgUnitPrice (weighted avg from purchases), calcMenuItemCost (equal proportion model), calcIngredientUsage (reverse-lookup with return ratio)
- 8 new Playwright tests (TRND-01 through INTG-02) — all passing; replaced old "Coming Soon" test with COST-01
- SW cache bumped from v45 to v46 for deployment
- Human-verify checkpoint approved by user
- Post-approval enhancements: category drill-down on Trends, time filter dropdown (All Time, Last Week, 1/3/6 Months, Last Year) on both Trends and Cost tabs, tag chip filter added to Cost tab, Chart.js click-to-drill-down on bar/doughnut charts, shared buildTimeFilterHtml() and buildChipBarHtml() helper functions

## Task Commits

Each task was committed atomically:

1. **Task 1: Cost Intelligence tab** - `38137ea` (feat)
2. **Task 2: Playwright tests + SW bump** - `d276eea` (feat)
3. **Task 3: Human-verify approved + post-approval enhancements** - `b35bb9c` (feat)

## Files Created/Modified
- `inventory.html` - Added CSS for menu-card/ingredient-row/revenue-row, state vars COST_SUB_TAB/EXPANDED_MENU_ITEMS/EXPANDED_INGREDIENTS, calcAvgUnitPrice/calcMenuItemCost/calcIngredientUsage helpers, renderCost() function, #s4 event delegation
- `tests/inventory.spec.js` - Replaced Coming Soon test with COST-01; added TRND-01, TRND-02, TRND-03, TRND-04, tab round-trip stability, COST-02, COST-03, INTG-02 tests
- `sw.js` - Bumped cache version from yumyums-v45 to yumyums-v46

## Decisions Made
- Equal ingredient proportion (1/n per ingredient group) — simple, transparent, no external weighting data needed
- Return ratio displayed as "Xx return" (e.g., "3.2x return") for ingredient ROI in the reverse-lookup view
- INTG-02 TODO comment only — no runtime Metabase switching infrastructure at mock stage

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

One pre-existing flaky test in workflows.spec.js (tab switching) that was failing before and after this plan's changes — out of scope per deviation rules.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Cost tab and Trends tab are fully functional with mock data, drill-down, time filters, and chip filters
- Human visual verification (Task 3 checkpoint) completed and approved
- Both Trends and Cost tabs ready for deployment
- Phase 08 complete — all plans (08-01, 08-02) executed successfully

---
*Phase: 08-trends-and-cost-intelligence-tabs*
*Completed: 2026-04-14*
