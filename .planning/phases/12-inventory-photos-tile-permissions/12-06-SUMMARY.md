---
phase: 12-inventory-photos-tile-permissions
plan: "06"
subsystem: frontend/test
tags: [inventory, receipt-review, e2e-tests, service-worker, bug-fix]
dependency_graph:
  requires: [12-04, 12-05]
  provides: [receipt-review-queue-ui, inventory-e2e-tests]
  affects: [inventory.html, tests/inventory.spec.js, sw.js, backend/internal/db/migrations/0024_inventory.sql]
tech_stack:
  added: []
  patterns: [event-delegation, real-time-form-state, confirm-dialog, in-place-form-expansion]
key_files:
  created: []
  modified:
    - inventory.html
    - tests/inventory.spec.js
    - sw.js
    - backend/internal/db/migrations/0024_inventory.sql
decisions:
  - renderPendingCard separates pending from confirmed event rendering — avoids nested conditional logic
  - captureReviewFormInputs reads DOM live inputs before re-render so form state persists through re-renders
  - Input event re-renders the review form in-place (not full list re-render) to update mismatch banner while preserving focus
  - REVIEW_FORM_STATE keyed by pending id — supports concurrent review state if ever multiple forms needed
  - Fixed is_active -> enabled in 0024_inventory.sql migration (hq_apps table column mismatch caused server startup failure)
metrics:
  duration_minutes: 4
  completed_date: "2026-04-18"
  tasks_completed: 2
  files_modified: 4
requirements: [INVT-03]
---

# Phase 12 Plan 06: Receipt Review Queue UI and E2E Tests Summary

**One-liner:** Receipt review queue UI in inventory.html — pending items with Needs Review badge, in-place editable review form with real-time mismatch banner, confirm/discard actions wired to API; 27 E2E tests rewritten for fully API-backed inventory flow.

## What Was Built

### Task 1: Receipt Review Queue UI (inventory.html)

Added the complete INVT-03 human review step:

**CSS additions:**
- `.approval-badge` — warn-colored "Needs Review" badge
- `.review-form` — full-width bordered form card for in-place expansion
- `.correction-banner` — reused pattern for total mismatch warning
- `.review-line-item-row` — editable line item row with name/qty/price/is_case inputs
- `.btn-primary` / `.btn-secondary` — confirm and discard action buttons
- `.new-entity-warning`, `.inline-error`, `.add-line-btn`, `.line-total-row`, `.field-group`, `.field-label`, `.field-input`

**JS functions added:**
- `renderPendingCard(p)` — renders collapsed pending item card with Needs Review badge
- `renderReviewForm(p)` — renders expanded in-place review form, reads from REVIEW_FORM_STATE cache
- `captureReviewFormInputs(pendingId)` — reads live DOM inputs into REVIEW_FORM_STATE before re-render
- `escHtml(s)` — XSS-safe HTML escaping for user-facing values
- `confirmReceipt(id)` — validates form, POSTs to /api/v1/inventory/purchases/confirm, removes from pending list
- `discardReceipt(id)` — native confirm() dialog, POSTs to /api/v1/inventory/purchases/discard
- `REVIEW_OPEN_ID`, `REVIEW_FORM_STATE` state vars

**Interaction flow:**
1. Pending items rendered at top of History tab with Needs Review badge (warn colors)
2. Tap pending card → expands to review form in-place
3. Review form pre-fills vendor, date, line items from pending purchase data
4. Line item total computed in real-time; mismatch banner shows if ≠ bank_total (>$0.01 difference)
5. Add/remove line item rows dynamically
6. Confirm Receipt → POST confirm endpoint → removes from pending list, refreshes history
7. Discard Receipt → native confirm() → POST discard endpoint → removes from pending list
8. "All caught up" empty state shows when no pending items but confirmed events exist

### Task 2: E2E Tests and SW Rebuild

Rewrote `tests/inventory.spec.js` completely (27 tests):

| Test Category | Tests |
|---------------|-------|
| Tab navigation | 2 |
| History tab API load | 5 |
| Vendor filter | 3 |
| Stock tab | 4 |
| Reorder suggestions | 1 |
| Stock override | 1 |
| Trends/Cost stubs | 2 |
| Receipt review queue | 6 |
| Navigation/PWA | 3 |

All 27 tests pass.

**SW rebuilt** with updated inventory.html content hash.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Migration 0024_inventory.sql used wrong column name for hq_apps**
- **Found during:** Task 2 test run
- **Issue:** `INSERT INTO hq_apps (slug, name, icon, is_active)` — the `hq_apps` table (created in 0004) uses `enabled`, not `is_active`. This caused server startup failure: `ERROR: column "is_active" of relation "hq_apps" does not exist`.
- **Fix:** Changed `is_active` to `enabled` in the INSERT statement.
- **Files modified:** `backend/internal/db/migrations/0024_inventory.sql`
- **Commit:** b21aa47

**2. [Rule 2 - Missing] E2E tests replaced stale mock-data tests**
- **Found during:** Task 2 planning
- **Issue:** Existing `tests/inventory.spec.js` tested mock data features (Chart.js window.Chart, MOCK_ arrays, chart sub-tabs, trend/cost mock data) that were removed in Plan 04. Running these tests against the API-backed inventory would result in ~15 test failures.
- **Fix:** Full rewrite — all 27 new tests test the actual API-backed inventory behavior.
- **Files modified:** `tests/inventory.spec.js`
- **Commit:** b21aa47

## Known Stubs

The following are intentional per prior plan decisions:

- **Trends tab** (`#trends-container`): "coming soon" — sales data integration deferred per D-13
- **Cost tab** (`#cost-container`): "coming soon" — food cost calculations deferred per D-13
- **`seedPendingPurchase` helper in tests**: Has a `pending-seed` API endpoint placeholder that returns null gracefully — pending purchase tests only assert on currently-present data. A future plan could add a test-seeder endpoint to enable deterministic pending purchase testing.

## Self-Check: PASSED

- inventory.html exists: FOUND
- tests/inventory.spec.js exists: FOUND
- sw.js exists: FOUND
- backend/internal/db/migrations/0024_inventory.sql exists: FOUND
- Commit 50e2f20 (Task 1): FOUND
- Commit b21aa47 (Task 2): FOUND
- "Needs Review" in inventory.html: 1 occurrence
- "purchases/confirm" in inventory.html: 1 occurrence
- "purchases/discard" in inventory.html: 1 occurrence
- "correction-banner" in inventory.html: 5 occurrences
- "All caught up" in inventory.html: 1 occurrence
- Test count in inventory.spec.js: 27 (all pass)
