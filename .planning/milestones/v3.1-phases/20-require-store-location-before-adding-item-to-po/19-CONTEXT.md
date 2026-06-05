---
phase: 19
type: context
status: decided
---

# Phase 19: Require store location before adding item to PO — Context

## Decisions

### D-01: Block at add-to-PO
Items without `store_location` CANNOT be added to a PO. The `addItemToPO()` function in purchasing.html must check `catalogItem.store_location` before adding. If null/empty, show a message with a link to set it (Inventory > Setup).

### D-02: Location is set in Inventory Setup
Users set `store_location` via Inventory > Setup tab. No inline location entry in the PO item picker. The picker shows "Add location in Setup" link for unassigned items instead of an "+ Add" button.

### D-03: Item picker grouped by store_location
The item picker modal in purchasing.html should group items by `store_location`. Groups are sorted alphabetically. Items without a location appear under an "Unassigned" section at the bottom. Each group header shows the location name.

### D-04: Inventory Setup items grouped by Category + Store Location
Items in Inventory > Setup should display grouped as: `Category, Store Location` — e.g., "(Bread, Giant)", "(Bread, Restaurant Depot)". This replaces the current flat or category-only grouping.

### D-05: Existing items grandfathered with null
Existing items keep `store_location = null`. They appear in the picker under "Unassigned" but cannot be added to a PO until the user assigns a location via Setup.

### D-06: No retroactive blocking
Existing PO line items that already have null locations are NOT blocked or removed. The constraint only applies when adding NEW items to a PO.

## Codebase Assets

| Asset | Location | Notes |
|-------|----------|-------|
| `store_location` column | `purchase_items.store_location TEXT` | Migration 0035 |
| Add item to PO | `purchasing.html:835-861` `addItemToPO()` | Check location here |
| Item picker render | `purchasing.html:892-931` `renderPickerList()` | Group by location |
| ALL_ITEMS array | Loaded from `/api/v1/inventory/items` | Has `store_location` field |
| Setup tab items | `inventory.html` Setup tab | Currently grouped by group_name only |
| Shopping location edit | `purchasing/service.go:478-493` | Updates both shopping + catalog |

## Out of Scope

- Editing store_location inline in the PO picker (D-02 decided against)
- Blocking PO approval based on missing locations (D-01 blocks at add time)
- Migrating existing null locations (D-05, D-06)
