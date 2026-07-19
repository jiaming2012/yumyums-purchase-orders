---
phase: 14-po-backend-order-form
plan: "02"
subsystem: frontend/purchasing
tags: [html, vanilla-js, pwa, purchase-orders, api-integration]
dependency_graph:
  requires: [14-01]
  provides: [purchasing-ui, po-order-form]
  affects: [purchasing.html, sw.js]
tech_stack:
  added: []
  patterns: [state-first-rendering, event-delegation, debounced-save, optimistic-ui, fullscreen-modal]
key_files:
  created: []
  modified:
    - purchasing.html
    - sw.js
decisions:
  - "Full file rewrite (not incremental edit) — old mock had no reusable state management"
  - "addItemToPO uses added_by_name: 'You' as placeholder until API response overwrites on next save"
  - "Suggestions in picker-list show as clickable badge (not disabled) so duplicate popup triggers correctly"
  - "Error card auto-removes after 4 seconds to avoid cluttering the UI"
metrics:
  duration: "~3 minutes"
  completed_date: "2026-04-22"
  tasks_completed: 1
  files_created: 0
  files_modified: 2
---

# Phase 14 Plan 02: PO Frontend — Order Form Summary

Complete rewrite of purchasing.html from 89-line static mock into a 454-line live API-backed purchase order tool with 4-tab layout, reorder suggestions checklist, photo card item layout with stepper, fullscreen item picker with search and duplicate detection, and debounced auto-save.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Rewrite purchasing.html — tab shell, Order tab with item cards, suggestions, stepper | 187a4d1 | purchasing.html, sw.js |

## Checkpoint Pending

| Task | Name | Status |
|------|------|--------|
| 2 | Verify PO form end-to-end | Awaiting human verification |

## What Was Built

### purchasing.html (454 lines)

**4-Tab Layout:**
- Order tab — fully interactive, rendered from PO_STATE
- Shopping tab — stub: "Shopping list will appear here after the PO is approved"
- PO tab — stub: "Locked PO will appear here after the weekly cutoff"
- History tab — stub: "Past shopping runs will appear here"

**Module-Level State:**
- `PO_STATE` — current week's PO from `POST /api/v1/purchasing/orders`
- `SUGGESTIONS` — reorder items from `GET /api/v1/purchasing/orders/:id/suggestions`
- `ALL_ITEMS` — full catalog from `GET /api/v1/inventory/items`
- `SUGGESTION_OPEN`, `DEBOUNCE_TIMER`, `DUP_ITEM_ID`

**Order Tab Components (11 UI-SPEC components):**

1. 4-Tab Bar — extends existing `.tabs` pattern to 4 tabs with data-action routing
2. Reorder Suggestion Banner — amber warn color, item count, [+] toggle
3. Suggestion Checklist — inline checklist, checkboxes, suggested qty, Add Selected CTA
4. PO Item Card — 48×48 thumbnail (letter fallback for null photo_url), name, unit+location, initials, stepper +/-
5. Category Sticky Header — `.cat` class, uppercase, position:sticky
6. Item Picker Modal — fullscreen, search input with 300ms debounce, max 30 results
7. Duplicate Popup — bottom sheet with backdrop, Go to item (scrollIntoView) and Dismiss
8. + Add Item Button — full-width accent CTA at bottom of order list
9. Order Header Card — week label (weekLabel() helper), contributor count, cutoff pill
10. Empty State — 🛒 icon, "Nothing on the order yet", instruction text
11. Stub States — simple card with muted centered text for Shopping/PO/History

**Key Functions:**
- `init()` — parallel load PO + catalog, then suggestions, then renderOrder()
- `renderOrder()` — state-first: builds entire Order tab HTML from PO_STATE
- `renderItemCard(item)` — returns HTML string for photo card layout
- `updateQty(itemId, delta)` — optimistic DOM update + debouncedSave()
- `debouncedSave()` / `savePOItems()` — 600ms debounce, PUT to /orders/:id/items
- `addItemToPO(itemId)` — append to PO_STATE.line_items + immediate save
- `addSelectedSuggestions()` — bulk add from checked suggestion checkboxes
- `openItemModal()` / `closeItemModal()` / `renderPickerList(query)` — picker lifecycle
- `showDuplicatePopup(itemId)` / `dismissDuplicatePopup()` / `gotoItem(itemId)` — dup detection
- `show(n)` — 4-tab visibility toggle
- `weekLabel(dateStr)` — formats "Week of Apr 20" from ISO date
- `initials(name)` — extracts 2-char initials from full name

**Error Handling:**
- Load failure: inline error card "Couldn't load the order. Pull down to refresh."
- Save failure: inline error card "Couldn't save. Check connection and try again." (auto-removes after 4s)
- 401 redirect: all API calls via `api()` wrapper redirect to /login.html

**SW rebuild:** `node build-sw.js` run after rewrite — 20 files precached, new content hashes.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

- `photo_url` on all items is null until Phase 15 Notion import — frontend renders letter placeholder gracefully
- `store_location` is null for all items until Phase 15 — omitted silently from card meta line
- `unit` on OrderSuggestion may be empty string — frontend renders without unit when empty
- Shopping, PO, History tabs are static stub text — will be wired in Phases 16-17

## Self-Check: PASSED

- [x] purchasing.html exists at 454 lines (target: 400-500)
- [x] `let PO_STATE = null` — present
- [x] `let SUGGESTIONS = []` — present
- [x] `let ALL_ITEMS = []` — present
- [x] `api('/api/v1/purchasing/orders'` — present (POST on init)
- [x] `api('/api/v1/purchasing/orders/' + PO_STATE.id + '/suggestions'` — present
- [x] `api('/api/v1/inventory/items'` — present
- [x] `/api/v1/purchasing/orders/' + PO_STATE.id + '/items'` — present (PUT in savePOItems)
- [x] `function renderOrder()` — present
- [x] `function openItemModal()` — present
- [x] `function renderPickerList(` — present
- [x] `function showDuplicatePopup(` — present
- [x] `function updateQty(` — present
- [x] `function savePOItems(` — present
- [x] All data-action attributes: qty-inc, qty-dec, open-picker, toggle-suggestions, add-to-po, picker-item-on-po, goto-item — present
- [x] All 4 stub/state texts present: Shopping list..., Locked PO..., Past shopping runs..., Nothing on the order yet
- [x] `scrollIntoView` — present in gotoItem()
- [x] `DEBOUNCE_MS` — present
- [x] `id="t4"` and `id="s4"` — present
- [x] `On PO:` badge text — present
- [x] `already on PO` — present in showDuplicatePopup
- [x] `Added by:` — present in renderItemCard
- [x] Load error message (Couldn\'t load) — present (JS-escaped apostrophe)
- [x] Save error message (Couldn\'t save) — present (JS-escaped apostrophe)
- [x] `aria-label="Quantity:` — present on stepper span
- [x] `show(+el.dataset.tab)` — present in event delegation
- [x] Commit 187a4d1 exists
- [x] `node build-sw.js` succeeded (20 files, 712.8 KB)
