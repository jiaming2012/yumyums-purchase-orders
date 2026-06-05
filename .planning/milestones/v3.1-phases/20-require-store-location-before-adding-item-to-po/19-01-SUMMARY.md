---
phase: 19-require-store-location-before-adding-item-to-po
plan: 01
subsystem: ui
tags: [purchasing, item-picker, store-location, vanilla-js, event-delegation]

requires:
  - phase: 16-cutoff-approval-and-shopping-list
    provides: purchasing.html item picker modal, addItemToPO function, ALL_ITEMS catalog
provides:
  - store_location guard in addItemToPO preventing null-location items from being added to PO
  - grouped item picker by store_location with sticky headers
  - Unassigned group at bottom with "Set location in Setup" hint
affects: [19-02, purchasing, inventory-setup]

tech-stack:
  added: []
  patterns: [picker-group-header for categorized modal lists]

key-files:
  created: []
  modified: [purchasing.html, tests/purchasing.spec.js]

key-decisions:
  - "Reused shop-toast pattern for guard message instead of adding new toast system"
  - "pr-unassigned class for italic hint text on unlocated items"
  - "30-item cap applied across all groups combined, not per group"

patterns-established:
  - "picker-group-header: sticky uppercase section header for grouped modal lists"

requirements-completed: []

duration: 5min
completed: 2026-04-28
---

# Phase 19 Plan 01: Block unlocated items from PO and group picker by store location Summary

**Item picker grouped by store_location with sticky headers; unlocated items blocked from PO with toast guard and "Set location in Setup" hint**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-28T15:16:17Z
- **Completed:** 2026-04-28T15:21:56Z
- **Tasks:** 4
- **Files modified:** 2

## Accomplishments
- addItemToPO guard prevents items without store_location from being added to PO
- Item picker redesigned to group items by store_location alphabetically with sticky headers
- Unassigned items appear at bottom with "Set location in Setup" instead of Add button
- 4 regression tests covering guard, grouping, located items, and unlocated items

## Task Commits

Each task was committed atomically:

1. **Task 1: Block addItemToPO for items without store_location** - `e5e488e` (feat)
2. **Task 2+3: Group item picker by store_location + CSS** - `b05af54` (feat)
3. **Task 4: Regression tests** - `488dba7` (test)

**Plan metadata:** pending (docs: complete plan)

## Files Created/Modified
- `purchasing.html` - Added store_location guard in addItemToPO, redesigned renderPickerList with grouping, CSS for picker-group-header and pr-unassigned
- `tests/purchasing.spec.js` - 4 new regression tests for store_location enforcement

## Decisions Made
- Reused existing shop-toast pattern for guard message rather than introducing a new notification mechanism
- Combined Tasks 2 (JS) and 3 (CSS) into one commit since CSS is a dependency of the JS change
- 30-item cap applies across all groups combined to maintain performance

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Pre-existing unstaged changes in purchasing.html required careful partial staging via patches to isolate task-specific commits
- Backend server not running in execution environment so tests could not be run end-to-end; syntax verified via node -c

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Plan 19-02 (group Setup items by Category + Store Location) has already been executed
- Phase 19 is complete pending verification

## Self-Check: PASSED

- 19-01-SUMMARY.md: FOUND
- Commit e5e488e: FOUND
- Commit b05af54: FOUND
- Commit 488dba7: FOUND

---
*Phase: 19-require-store-location-before-adding-item-to-po*
*Completed: 2026-04-28*
