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

- **Duration:** 15 min
- **Started:** 2026-04-14T19:10:00Z
- **Completed:** 2026-04-14T19:25:40Z
- **Tasks:** 2 (Task 3 is checkpoint - awaiting human-verify)
- **Files modified:** 3

## Accomplishments
- renderCost() implemented with two sub-tabs: Menu Items (COST-01/02) and Ingredients (COST-03)
- Cost calculation helpers: calcAvgUnitPrice (weighted avg from purchases), calcMenuItemCost (equal proportion model), calcIngredientUsage (reverse-lookup with return ratio)
- 8 new Playwright tests (TRND-01 through INTG-02) — all passing; replaced old "Coming Soon" test with COST-01
- SW cache bumped from v45 to v46 for deployment

## Task Commits

Each task was committed atomically:

1. **Task 1: Cost Intelligence tab** - `38137ea` (feat)
2. **Task 2: Playwright tests + SW bump** - `d276eea` (feat)

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

- Cost tab and Trends tab are fully functional with mock data
- Human visual verification (Task 3 checkpoint) needed before deployment
- After checkpoint approval, both Trends and Cost tabs ready for deployment

---
*Phase: 08-trends-and-cost-intelligence-tabs*
*Completed: 2026-04-14*
