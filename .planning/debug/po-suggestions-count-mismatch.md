---
status: resolved
trigger: "po-suggestions-count-mismatch — PO suggestions count (4) doesn't match inventory reorder suggestions count (3)"
created: 2026-04-22T00:00:00Z
updated: 2026-04-22T17:00:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

hypothesis: CONFIRMED — PO suggestions used `<=` high_threshold but inventory uses `<` high_threshold
test: Boundary test with item at exactly high_threshold value (qty=10, threshold=10)
expecting: Item at high_threshold is excluded from both PO suggestions and inventory reorder
next_action: DONE — fix verified and passes

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: PO suggestions count should exactly match inventory reorder suggestions count (both should show the same items needing reorder)
actual: PO suggestions API returns 4 items, but inventory Stock tab shows only 3 reorder suggestions
errors: No errors — the test passes incorrectly because it uses toBeLessThanOrEqual instead of toBe
reproduction: Run the test `PO suggestions count matches inventory reorder suggestions` in tests/inventory.spec.js:1916 — it passes with toBeLessThanOrEqual but would fail with strict equality
started: Known issue since Phase 16 development of v3.0

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: Items with qty=0 ('unknown' in inventory) included in PO suggestions
  evidence: PO suggestions already had `AND sub.current_stock > 0` which correctly excludes qty=0
  timestamp: 2026-04-22T16:45:00Z

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-04-22T16:40:00Z
  checked: tests/inventory.spec.js:1916-1952
  found: Test uses toBeLessThanOrEqual(inventoryCount) — soft assertion that masks mismatch
  implication: The test was designed to never catch exact count mismatch

- timestamp: 2026-04-22T16:45:00Z
  checked: backend/internal/purchasing/service.go:574-641 (GetSuggestions)
  found: PO suggestions WHERE clause: `current_stock <= COALESCE(ig.high_threshold, 10)`
  implication: Includes items where qty == high_threshold (at the boundary)

- timestamp: 2026-04-22T16:45:00Z
  checked: inventory.html:534
  found: Inventory level formula: `tq === 0 ? 'unknown' : tq >= highT ? 'high' : tq > lowT ? 'medium' : 'low'`
  implication: When qty == highT, inventory marks as 'high' (NOT a reorder candidate)

- timestamp: 2026-04-22T16:48:00Z
  checked: inventory.html:539
  found: Reorder filter: `needsReorder = stocks.filter(s => s.level === 'low' || s.level === 'medium')`
  implication: 'high' items are excluded. So items at exactly high_threshold are in PO suggestions (count=4) but not inventory reorder (count=3)

- timestamp: 2026-04-22T17:00:00Z
  checked: Test run with boundary data (item with qty=high_threshold)
  found: PO suggestions includes item with current_stock=10 and stock_level="Medium" — incorrect
  implication: Confirms `<=` vs `<` boundary discrepancy is the root cause

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: >
  GetSuggestions in backend/internal/purchasing/service.go used `sub.current_stock <= COALESCE(ig.high_threshold, 10)`
  but the inventory frontend classifies items with `qty >= highT` as 'high' (not a reorder candidate).
  An item with qty == high_threshold is included in PO suggestions (`<= 10` is true) but excluded from
  inventory reorder suggestions (`qty >= highT` → 'high' level). This caused PO to show 1 extra item.

fix: >
  Changed `sub.current_stock <= COALESCE(ig.high_threshold, 10)` to
  `sub.current_stock < COALESCE(ig.high_threshold, 10)` in the GetSuggestions SQL query.
  This mirrors the inventory frontend's strict less-than check for the high threshold boundary.

verification: >
  Regression test rewritten to explicitly create a boundary item (qty == high_threshold),
  assert it does NOT appear in PO suggestions, and assert PO count exactly matches inventory count.
  Test passes with fix applied, fails without it.

files_changed:
  - backend/internal/purchasing/service.go
  - tests/inventory.spec.js
