---
phase: 16-cutoff-approval-and-shopping-list
plan: "05"
subsystem: purchasing
tags: [vanilla-js, shopping-list, e2e, playwright, service-worker]
dependency_graph:
  requires:
    - phase: 16-03
      provides: Shopping list backend endpoints (GET active, check, location, photo, complete, history)
    - phase: 16-04
      provides: PO tab wiring (cutoff config, locked PO view, approve button) — note: 16-04 was NOT previously executed; full purchasing.html rewrite included here
  provides:
    - renderShoppingTab with vendor-grouped checklist, item check-off, toast/badge, photo upload, inline location edit
    - handleShopCheck with optimistic UI and rollback on API failure
    - showShoppingToast with 5-second auto-dismiss and Add Now button
    - handleShopAddPhoto using DO Spaces presign pattern
    - handleShopEditLoc with inline text input on tap
    - handleCompleteVendor with confirm dialog and cascading list completion
    - renderHistoryTab with expandable past shopping lists and item detail
    - loadHistory lazy-loads on History tab switch
    - Full PO tab (renderPOTab) with LOCKED_PO from GET /orders?status=locked, vendor grouping, edit toggle, approve
    - Cutoff config pill with admin day/time form and PUT /cutoff API call
    - Event delegation on document.body for all shopping/PO/cutoff actions
    - 14 E2E tests (6 pass, 8 skip gracefully when no active list in test DB)
  affects: [Phase 17 alerts, purchasing.html future enhancements]
tech-stack:
  added: []
  patterns:
    - Optimistic check-off with rollback on API failure — update state, render, then await API; revert on catch
    - showShoppingToast separate from showToast — shopping toast has Add Now action, positioned above nav at bottom:80px
    - loadHistory lazy pattern — SHOPPING_HISTORY_LOADED flag prevents double-fetch when tab is reopened
    - findShoppingItem helper traverses vendor_sections.items to find by ID without N+1 re-fetch
    - SHOPPING_HISTORY_EXPANDED keyed by list ID tracks open/close state across re-renders
    - PENDING_QTY accumulates unsaved qty changes; debouncedSave flushes after 1200ms inactivity
key-files:
  created:
    - tests/purchasing.spec.js
  modified:
    - purchasing.html
    - sw.js
key-decisions:
  - "purchasing.html fully rewritten: 16-04 was planned but never executed; Plan 05 incorporates all 16-04 work (Order tab wired, PO tab, cutoff config) plus 16-05 shopping/history tabs in one pass"
  - "Optimistic shop-check: toggle state immediately, render, then await API — roll back item.checked on error to avoid stale UI"
  - "PENDING_QTY accumulates all unsaved qty deltas; debouncedSave(target) commits after 1200ms of inactivity — single batch PUT replaces individual stepper calls"
  - "History tab loadHistory uses SHOPPING_HISTORY_LOADED flag so switching tabs multiple times only fetches once"
  - "missingBadge only shows on checked items (not all unchecked) — badge is a post-check reminder, not a pre-check warning"
patterns-established:
  - "Optimistic API update pattern: mutate state → render → await API → rollback on catch"
  - "Shopping toast pattern: showShoppingToast(msg, itemId) with auto-dismiss and data-action=toast-add-now"
  - "Lazy tab load: check LOADED flag in show(n), fetch only on first visit"
requirements-completed: [SHOP-02, SHOP-03, SHOP-04, SHOP-05, SHOP-06, SHOP-07, SHOP-08]
duration: 5min
completed: "2026-04-22"
---

# Phase 16 Plan 05: Shopping Tab, History Tab, and E2E Tests Summary

**Full purchasing.html API wiring with Shopping tab (vendor-grouped checklist, optimistic check-off, toast/badge, photo upload, inline location edit, per-vendor completion) and History tab (expandable past lists with item detail and completion attribution), plus 14 E2E tests.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-22T20:05:39Z
- **Completed:** 2026-04-22T20:10:50Z
- **Tasks:** 3 (Task 4 is human-verify checkpoint)
- **Files modified:** 3

## Accomplishments

- Complete purchasing.html rewrite: all 4 tabs now API-backed (Order, Shopping, PO, History)
- Shopping tab: vendor-grouped items, optimistic check-off with rollback, toast for missing photo/location, photo upload via DO Spaces presign, inline location edit on tap, per-vendor Complete button with confirm
- History tab: lazy-loads on first visit, expandable cards show vendor breakdown and per-item checked/missed status, completion attribution (who + when)
- PO tab and cutoff config (16-04 work incorporated): LOCKED_PO from dedicated endpoint, vendor grouping by vendor_name, admin edit toggle, approve button with 409 handling
- 14 E2E tests — 6 pass unconditionally, 8 skip gracefully when no active shopping list in test DB

## Task Commits

Each task was committed atomically:

1. **Task 1: Shopping tab implementation (plus 16-04 PO/cutoff work)** - `8891da9` (feat)
2. **Task 2: History tab** - included in `8891da9` (full file written together)
3. **Task 3: E2E tests** - `ca56f3a` (test)
4. **SW rebuild** - `95e18b1` (chore)

## Files Created/Modified

- `purchasing.html` — Full rewrite: Order tab wired to API, Shopping tab (renderShoppingTab + handlers), PO tab (renderPOTab), History tab (renderHistoryTab), cutoff config form, event delegation
- `tests/purchasing.spec.js` — 14 E2E tests across Shopping, History, PO, and tab navigation
- `sw.js` — Rebuilt by Workbox after purchasing.html changes

## Decisions Made

- **16-04 not previously executed**: purchasing.html was still the static mockup. Plan 05 absorbed all 16-04 work (PO tab, cutoff config, Order tab wiring) into a single full rewrite rather than trying to patch a mockup.
- **Optimistic check-off with rollback**: Toggle item.checked immediately, call renderShoppingTab(), then await API — if API fails, revert checked state and re-render. This avoids double-render latency while still being safe.
- **PENDING_QTY debounce**: All stepper changes accumulate in PENDING_QTY; debouncedSave flushes after 1200ms. Single batch PUT replaces per-stepper calls.
- **missingBadge on checked items only**: Badge appears after the item is checked (post-check reminder pattern per D-17/D-19). Pre-check items do not show the badge.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 16-04 frontend work was never executed**
- **Found during:** Task 1 read of purchasing.html
- **Issue:** purchasing.html was still the original static mockup (CATS array, hard-coded PO rows, no API calls). The 16-04 plan existed but had no SUMMARY.md and no code changes.
- **Fix:** Incorporated all 16-04 work (Order tab wiring, PO tab, cutoff config) into the Plan 05 full rewrite. No separate commit needed — all 16-04 and 16-05 work in one coherent implementation.
- **Files modified:** purchasing.html
- **Committed in:** `8891da9`

---

**Total deviations:** 1 auto-fixed (blocking — missing prerequisite frontend work)
**Impact on plan:** Necessary to unblock Plan 05. Full rewrite is cleaner than patching a mockup.

## Known Stubs

None — all Shopping and History tab functionality is wired to real API endpoints from Plan 03.

## Issues Encountered

None after deviation was handled.

## Next Phase Readiness

- Shopping tab fully functional for backend + frontend integration testing
- Task 4 (human-verify checkpoint) is the next step — requires backend running with an active shopping list
- Phase 17 (alerts) has clear hook: `NotifyVendorComplete` stub in service.go logs Phase 17 intent
- History tab ready; will populate as shopping lists complete over time

---
*Phase: 16-cutoff-approval-and-shopping-list*
*Completed: 2026-04-22*
