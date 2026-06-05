---
phase: 19
plan: 2
subsystem: inventory
tags: [setup, grouping, store-location, ui]
dependency_graph:
  requires: [19-CONTEXT]
  provides: [setup-composite-grouping, store-location-edit-in-setup]
  affects: [inventory.html, handler.go, types.go]
tech_stack:
  added: []
  patterns: [composite-key-grouping]
key_files:
  created: []
  modified:
    - inventory.html
    - backend/internal/inventory/handler.go
    - backend/internal/inventory/types.go
    - tests/inventory.spec.js
decisions:
  - "Item rows show store_location label (not group_name) since category is now in the composite group header"
  - "Backend UpdateItemHandler uses store_location field with explicit null support for clearing locations"
  - "CreateItemHandler ON CONFLICT preserves existing store_location via COALESCE"
metrics:
  duration: 267
  completed: "2026-04-28T15:20:47Z"
---

# Phase 19 Plan 2: Group Inventory Setup Items by Category + Store Location Summary

Composite (category, store_location) grouping in Inventory Setup tab with editable store_location field and backend support.

## What Was Done

### Task 1+2: Update item grouping and add store_location edit (46867aa)

**Frontend (inventory.html):**
- `renderItemsList()` now groups items by `(group_name, store_location)` composite key instead of `group_name` alone
- Items with null `store_location` appear under "Category, Unassigned" headers
- Groups sorted alphabetically by composite key; items within groups sorted alphabetically by description
- Item edit form includes a "Store Location" text input field
- Save handler sends `store_location` in the PUT request body
- Item row label shows `store_location` instead of `group_name` (since category is in the header)

**Backend (handler.go, types.go):**
- `PurchaseItem` struct: added `StoreLocation *string` field
- `ListItemsHandler`: query now selects `pi.store_location`, scan includes it
- `UpdateItemHandler`: input struct accepts `store_location`, UPDATE query sets it
- `CreateItemHandler`: input struct accepts `store_location`, INSERT includes it, ON CONFLICT uses COALESCE to preserve existing location

### Task 3: Regression tests (996d54a)

Three new tests in `tests/inventory.spec.js`:
1. **Grouping test:** Items with different store_locations in same category appear under separate composite group headers
2. **Unassigned test:** Items with null store_location appear under "Category, Unassigned"
3. **Edit + persistence test:** User can set store_location via Setup edit form, value persists after page reload

## Deviations from Plan

None - plan executed exactly as written.

## Self-Check: PASSED
