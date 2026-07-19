---
phase: 14-po-backend-order-form
verified: 2026-04-22T14:30:00Z
status: human_needed
score: 6/6 must-haves verified
human_verification:
  - test: "Verify PO form loads live data and reorder suggestions appear"
    expected: "Order tab loads with 'Week of [current Monday]' header; if items are below stock threshold, a banner appears showing N items need restock"
    why_human: "Requires live app with real inventory stock data — cannot verify API data content programmatically without a running server"
  - test: "Verify adding a reorder suggestion to the order"
    expected: "Tap [+] banner to expand suggestion checklist, check items, tap 'Add Selected' — items appear in order list with pre-filled quantities"
    why_human: "Interactive UI flow with checkbox state and DOM mutation requires browser"
  - test: "Verify stepper persistence across page reload"
    expected: "Tap + on a stepper to increment quantity; reload the page; same quantity appears (loaded from API)"
    why_human: "Requires live server to confirm PUT /items persists and GET /orders returns updated quantities"
  - test: "Verify item picker search and duplicate detection"
    expected: "Tap 'Add item'; search for an item; items already on PO show 'On PO: N' badge (not +Add); tapping the badge shows duplicate popup with 'Go to item' and 'Dismiss'"
    why_human: "Multi-step interactive flow requiring browser and live item catalog data"
  - test: "Verify 'Go to item' scrolls to the card in Order list"
    expected: "From duplicate popup, tapping 'Go to item' closes modal and smoothly scrolls to the item card in the Order tab"
    why_human: "scrollIntoView behavior requires browser"
---

# Phase 14: PO Backend + Order Form Verification Report

**Phase Goal:** Users can create and edit a real purchase order from live inventory data, with reorder suggestions pre-populated and item search available
**Verified:** 2026-04-22T14:30:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (from Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User sees a pre-populated list of items flagged for reorder when opening the PO form | ✓ VERIFIED | `renderOrder()` checks `SUGGESTIONS.length > 0` and renders `.suggest-banner` with count; `SUGGESTIONS` populated from `GET /api/v1/purchasing/orders/:id/suggestions` on init; service query compares `current_stock <= high_threshold` |
| 2 | User can tap a reorder suggestion to add it to the order with the suggested quantity | ✓ VERIFIED | `addSelectedSuggestions()` reads checked checkboxes, pushes items with `suggestion.suggested_qty` into `PO_STATE.line_items`, calls `savePOItems()`; event delegation handles `add-selected` action |
| 3 | User can search the item catalog via fullscreen picker modal and add any item not already in the list | ✓ VERIFIED | `openItemModal()` opens `.item-modal.open`; `renderPickerList(query)` filters `ALL_ITEMS` by case-insensitive substring; `ALL_ITEMS` loaded from `GET /api/v1/inventory/items` on init; items not on PO show `+ Add` button |
| 4 | Each line item shows photo, store location note, suggested quantity, and who added it | ✓ VERIFIED | `renderItemCard()` renders `.item-thumb` with photo URL or letter fallback, `.mt` with `item.unit + store_location`, `.attr` with "Added by: [initials]"; `POLineItem` type includes `photo_url`, `store_location`, `added_by_name`; JOIN in `GetOrderLineItems` fetches all fields |
| 5 | User can adjust quantity on each line item and changes persist to the API | ✓ VERIFIED | `updateQty()` mutates `PO_STATE.line_items`, calls `debouncedSave()`; `savePOItems()` PUTs to `/api/v1/purchasing/orders/:id/items`; `PO_STATE` updated from API response; `UpsertLineItems` in Go commits to DB in transaction |
| 6 | When adding an item, user can see if it's already on the PO and its current quantity | ✓ VERIFIED | `renderPickerList()` checks `PO_STATE.line_items` for each catalog item; if found, renders `On PO: N [unit]` badge with `data-action="picker-item-on-po"`; tapping triggers `showDuplicatePopup()` showing item name, qty, and who added |

**Score:** 6/6 truths verified

### Required Artifacts

#### Plan 01 (Backend)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `backend/internal/db/migrations/0034_purchase_orders.sql` | purchase_orders and po_line_items tables | ✓ VERIFIED | Contains `CREATE TABLE purchase_orders`, `CREATE TABLE po_line_items`, `UNIQUE (po_id, purchase_item_id)`, `added_by UUID NOT NULL REFERENCES users(id)`, `version INTEGER NOT NULL DEFAULT 1`, `CHECK (status IN ('draft', 'locked', 'approved'))` |
| `backend/internal/db/migrations/0035_purchase_items_photo_store.sql` | photo_url and store_location columns | ✓ VERIFIED | Contains `ALTER TABLE purchase_items ADD COLUMN photo_url TEXT` and `ALTER TABLE purchase_items ADD COLUMN store_location TEXT`; both nullable |
| `backend/internal/purchasing/types.go` | PurchaseOrder, POLineItem, OrderSuggestion Go types | ✓ VERIFIED | All 4 types exported: `PurchaseOrder`, `POLineItem`, `OrderSuggestion`, `UpsertLineItemsRequest` |
| `backend/internal/purchasing/service.go` | Business logic: week start computation, suggestions query, upsert | ✓ VERIFIED | `CurrentWeekStart()`, `GetOrCreateOrder()`, `GetOrderByID()`, `GetOrderLineItems()`, `UpsertLineItems()`, `GetSuggestions()` — all substantive, real SQL queries |
| `backend/internal/purchasing/handler.go` | HTTP handlers for all 4 purchasing endpoints | ✓ VERIFIED | `GetOrCreateOrderHandler`, `GetOrderHandler`, `UpsertLineItemsHandler`, `GetSuggestionsHandler` — all exported, non-stub implementations |
| `backend/cmd/server/main.go` | Route registration under /api/v1/purchasing/* | ✓ VERIFIED | `r.Route("/purchasing", ...)` with all 4 handler registrations at lines 399-404 |

#### Plan 02 (Frontend)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `purchasing.html` | Complete PO form UI, min 300 lines, contains PO_STATE | ✓ VERIFIED | 458 lines; `let PO_STATE = null` present; full rewrite from 89-line mock; event delegation, debounced save, fullscreen modal, all 4 tabs |

### Key Link Verification

#### Plan 01 Links

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `backend/cmd/server/main.go` | `backend/internal/purchasing/handler.go` | chi route registration | ✓ WIRED | `r.Route("/purchasing"` at line 399; all 4 handlers registered |
| `backend/internal/purchasing/handler.go` | `backend/internal/auth/middleware.go` | auth.UserFromContext for added_by | ✓ WIRED | `auth.UserFromContext` at lines 28 and 68 |
| `backend/internal/purchasing/service.go` | purchase_orders table | SQL INSERT ON CONFLICT for get-or-create | ✓ WIRED | `ON CONFLICT (week_start) DO UPDATE SET week_start = EXCLUDED.week_start` at line 36 |

#### Plan 02 Links

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `purchasing.html` | `/api/v1/purchasing/orders` | POST on page load (get-or-create) | ✓ WIRED | `api('/api/v1/purchasing/orders', { method: 'POST', ... })` in `init()` at line 167; result assigned to `PO_STATE` |
| `purchasing.html` | `/api/v1/purchasing/orders/:id/items` | PUT on debounced save | ✓ WIRED | `api('/api/v1/purchasing/orders/' + PO_STATE.id + '/items', { method: 'PUT', ... })` in `savePOItems()` at line 297 |
| `purchasing.html` | `/api/v1/purchasing/orders/:id/suggestions` | GET on page load | ✓ WIRED | `api('/api/v1/purchasing/orders/' + PO_STATE.id + '/suggestions')` at line 172; result assigned to `SUGGESTIONS` |
| `purchasing.html` | `/api/v1/inventory/items` | GET for item picker catalog | ✓ WIRED | `api('/api/v1/inventory/items')` at line 168 in `Promise.all`; result assigned to `ALL_ITEMS` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `purchasing.html` → Order tab | `PO_STATE.line_items` | `POST /api/v1/purchasing/orders` → `GetOrCreateOrder()` → `GetOrderLineItems()` SQL JOIN | Yes — DB query with JOINs across `po_line_items`, `purchase_items`, `item_groups`, `users` | ✓ FLOWING |
| `purchasing.html` → Suggestions banner | `SUGGESTIONS` | `GET .../suggestions` → `GetSuggestions()` SQL subquery with stock threshold comparison | Yes — complex SQL comparing `current_stock` vs `high_threshold` | ✓ FLOWING |
| `purchasing.html` → Item picker | `ALL_ITEMS` | `GET /api/v1/inventory/items` (existing endpoint) | Yes — existing endpoint returns catalog items from DB | ✓ FLOWING |
| `purchasing.html` → Stepper save | `PO_STATE` updated | `PUT .../items` → `UpsertLineItems()` → `GetOrderByID()` | Yes — transactional DB upsert, returns fresh PO data | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Check | Status |
|----------|-------|--------|
| Backend compiles | `go build ./cmd/server/...` | ✓ PASS — exit 0, no output |
| Purchasing package compiles | `go build ./internal/purchasing/...` | ✓ PASS — exit 0 (verified via server build) |
| purchasing.html in SW precache | sw.js contains purchasing.html with revision hash | ✓ PASS — `{url:"purchasing.html",revision:"d02d7e4fdd6e2f381407e47fe2fcd147"}` present |
| purchasing.html min_lines check | ≥ 300 lines | ✓ PASS — 458 lines |
| `PO_STATE` in purchasing.html | Required state variable | ✓ PASS — `let PO_STATE = null` at line 134 |

Step 7b: Behavioral spot-checks completed on runnable/static artifacts. Live API endpoint tests skipped — require running server.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| PO-01 | 14-01, 14-02 | User can view reorder suggestions from inventory Stock tab on the PO form | ✓ SATISFIED | `GetSuggestions()` queries below-threshold items; `SUGGESTIONS` rendered in `.suggest-banner` |
| PO-02 | 14-01, 14-02 | User can tap a reorder suggestion to add it to the purchase order | ✓ SATISFIED | `addSelectedSuggestions()` with checkbox selection and `suggested_qty` pre-population |
| PO-03 | 14-02 | User can search and add items from the inventory catalog (Setup) | ✓ SATISFIED | `renderPickerList(query)` filters `ALL_ITEMS` from `/api/v1/inventory/items`; search input with 300ms debounce. **Note: REQUIREMENTS.md shows `[ ]` but implementation is complete — requirements file status is stale** |
| PO-04 | 14-01, 14-02 | Each PO line item shows the item's photo, store location note, and suggested quantity | ✓ SATISFIED | `renderItemCard()` renders photo thumbnail or letter fallback, store location note, unit; `POLineItem.PhotoURL` and `StoreLocation` populated from DB |
| PO-05 | 14-01, 14-02 | User can adjust quantity on each line item before submission | ✓ SATISFIED | Stepper +/- via `updateQty()`, debounced `savePOItems()` PUTs to API |
| PO-06 | 14-01, 14-02 | PO form is backed by real API data (replaces current purchasing.html mock) | ✓ SATISFIED | Full rewrite from 89-line mock to 458-line live API integration; no hardcoded data |
| PO-07 | 14-01, 14-02 | Each PO line item shows who added it to the order | ✓ SATISFIED | `added_by UUID NOT NULL REFERENCES users(id)` in schema; JOIN in `GetOrderLineItems` fetches display name; `renderItemCard()` shows "Added by: [initials]" |
| PO-08 | 14-02 | When adding an item, user can see if it's already on the PO and its current quantity | ✓ SATISFIED | `renderPickerList()` checks `PO_STATE.line_items` for each item; "On PO: N" badge shown; duplicate popup via `showDuplicatePopup()`. **Note: REQUIREMENTS.md shows `[ ]` but implementation is complete — requirements file status is stale** |

**Requirements file discrepancy:** PO-03 and PO-08 are marked `[ ]` in REQUIREMENTS.md but are fully implemented in the codebase. The traceability table at the bottom of REQUIREMENTS.md also shows both as "Pending". These statuses are stale — the code satisfies both requirements. This is not a gap; it is a documentation drift that should be corrected in REQUIREMENTS.md.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `purchasing.html` | 116 | HTML `placeholder` attribute on search input | ℹ️ Info | Not a stub — this is correct HTML usage for `<input placeholder="Search items...">` |
| `purchasing.html` | 329 | `added_by_name: 'You'` in optimistic add | ℹ️ Info | Documented design decision in 14-02-SUMMARY — API response overwrites on next save; not a blocker |

No blocker or warning anti-patterns found. Both flagged items are intentional and documented.

### Human Verification Required

#### 1. Order Tab Loads Live Data

**Test:** Open purchasing.html in the PWA (or browser after login). Observe the Order tab on load.
**Expected:** Header shows "Week of [current Monday date]"; contributor and item counts shown; no JS errors in console.
**Why human:** Requires running server with authenticated session and real DB data.

#### 2. Reorder Suggestions Banner

**Test:** Ensure some inventory items are below their group's high_threshold. Open purchasing.html. Observe the suggestion banner.
**Expected:** Yellow/orange banner appears: "N items need restock". Tapping expands a checklist with item names, stock level badge (Low/Medium), and "Suggested: N" quantities.
**Why human:** Requires specific stock data state in the database to trigger suggestions.

#### 3. Stepper Quantity Persists Across Reload

**Test:** Add an item or adjust a stepper on an existing item. Wait 600ms (debounce). Reload the page.
**Expected:** The same item appears with the updated quantity after reload — data round-tripped through PUT and re-loaded via POST on page load.
**Why human:** Requires running server to confirm DB persistence.

#### 4. Item Picker Search and Duplicate Detection

**Test:** Tap "+ Add item" button. Type a partial item name. Observe filtered results. Tap an item already on the order.
**Expected:** Fullscreen picker opens; results filter as you type (after 300ms); items on PO show "On PO: N" badge; tapping badge shows bottom sheet popup with item name, qty, "Go to item" and "Dismiss" buttons.
**Why human:** Multi-step interactive browser flow.

#### 5. "Go to item" Scroll Behavior

**Test:** From the duplicate popup, tap "Go to item".
**Expected:** Modal closes, popup dismisses, and the page smoothly scrolls to the item's card in the Order list.
**Why human:** `scrollIntoView({ behavior: 'smooth' })` requires browser rendering.

### Gaps Summary

No automated gaps found. All 6 success criteria are verified to be implemented and wired. The phase is pending human verification only — specifically the end-to-end data flow through the live API (checkpoint Task 2 in Plan 02 is documented as "Awaiting human verification" in the summary).

One documentation issue flagged: PO-03 and PO-08 in REQUIREMENTS.md show `[ ]` (pending) but the code fully implements both. Consider updating REQUIREMENTS.md to mark these as `[x]` complete.

---

_Verified: 2026-04-22T14:30:00Z_
_Verifier: Claude (gsd-verifier)_
