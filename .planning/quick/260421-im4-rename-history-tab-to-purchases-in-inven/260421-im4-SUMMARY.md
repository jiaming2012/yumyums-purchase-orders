---
phase: quick
plan: 260421-im4
subsystem: inventory
tags: [rename, ui, tabs, inventory]
dependency_graph:
  requires: []
  provides: ["Purchases tab label in inventory.html"]
  affects: ["inventory.html", "tests/inventory.spec.js", "CLAUDE.md"]
tech_stack:
  added: []
  patterns: []
key_files:
  modified:
    - inventory.html
    - tests/inventory.spec.js
    - CLAUDE.md
    - sw.js
decisions:
  - "Internal JS identifiers (loadHistory, renderHistoryList, updateHistoryTabLabel, history-list) left unchanged — only user-visible text renamed"
metrics:
  duration: 5
  completed_date: "2026-04-21"
  tasks_completed: 2
  files_modified: 4
---

# Quick 260421-im4: Rename History Tab to Purchases in Inventory Summary

**One-liner:** Renamed inventory "History" tab to "Purchases" in tab button, count label, tests, and CLAUDE.md docs.

## What Was Done

Renamed the first tab in `inventory.html` from "History" to "Purchases" — a more accurate label since the tab shows purchase events and pending receipt reviews. Updated all user-visible text and test assertions to match.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Rename History to Purchases in inventory.html | c2c2609 | inventory.html |
| 2 | Update test assertions and CLAUDE.md docs | f46a88c | tests/inventory.spec.js, CLAUDE.md, sw.js |

## Changes Made

### inventory.html
- Tab button (line 173): `History` → `Purchases`
- `updateHistoryTabLabel()` (line 499): `'History (N)'` → `'Purchases (N)'`

### tests/inventory.spec.js
- Test name: `'shows 4 tabs: History, Trends, Stock, Cost'` → `'shows 4 tabs: Purchases, Trends, Stock, Cost'`
- Assertion: `toContainText('History')` → `toContainText('Purchases')`
- Test name: `'History tab is active by default'` → `'Purchases tab is active by default'`
- Comment: `// HIST-01: History tab loads purchase events from API` → `// HIST-01: Purchases tab loads purchase events from API`
- Test name: `'History tab loads purchase events from API'` → `'Purchases tab loads purchase events from API'`
- Test name: `'History tab shows empty state when no purchases exist'` → `'Purchases tab shows empty state when no purchases exist'`

### CLAUDE.md
- Architecture section: `5-tab layout (History / Trends / Stock / Cost / Setup)` → `5-tab layout (Purchases / Trends / Stock / Cost / Setup)`
- Key Concepts section: `History (purchase events + pending review)` → `Purchases (purchase events + pending review)`

### sw.js
- Rebuilt via `task sw` to update content-hashed precache entry for inventory.html

## Deviations from Plan

None - plan executed exactly as written.

## Verification

- `grep -n "History" inventory.html` — no user-visible tab text; only internal identifiers remain
- `grep -n "Purchases" inventory.html` — 2 matches: tab button and count label
- `grep "History / Trends" CLAUDE.md` — 0 matches
- All 86 inventory tests pass (86 passed, 0 failed)

## Self-Check: PASSED

- inventory.html modified: FOUND
- tests/inventory.spec.js modified: FOUND
- CLAUDE.md modified: FOUND
- Commit c2c2609: FOUND
- Commit f46a88c: FOUND
- All 86 inventory tests: PASSED
