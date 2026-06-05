---
phase: 12-inventory-photos-tile-permissions
plan: "04"
subsystem: frontend
tags: [inventory, api, mock-to-real, service-worker]
dependency_graph:
  requires: [12-01]
  provides: [api-backed-inventory-ui]
  affects: [inventory.html, sw.js]
tech_stack:
  added: []
  patterns: [api-wrapper, skeleton-loading, inline-error-retry, pagination]
key_files:
  created: []
  modified:
    - inventory.html
    - sw.js
decisions:
  - renderHistoryList used for in-memory re-renders; loadHistory fetches fresh data — avoids double-fetch on tab switch
  - render() now calls renderHistoryList() instead of renderHistory() since data flow is async
  - Chart.js script tag removed — Trends/Cost are coming-soon, no longer need Chart.js
  - Unused state vars for Chart.js (trendBarChart, ACTIVE_CHIPS etc.) removed to reduce dead code
metrics:
  duration_minutes: 4
  completed_date: "2026-04-18"
  tasks_completed: 2
  files_modified: 2
requirements: [INVT-02]
---

# Phase 12 Plan 04: Inventory Mock-to-API Swap Summary

Big-bang replacement of all 8 MOCK_ arrays in inventory.html with live API fetch calls to the inventory backend created in Plan 01 — includes skeleton loading, inline errors, pagination, and pending review badges.

## What Was Built

**inventory.html — fully API-backed:**
- `api()` wrapper: async fetch with 401 redirect to login.html, 204 short-circuit, JSON error parse
- `loadVendors()`: populates vendor filter dropdown from `GET /api/v1/inventory/vendors`
- `loadHistory()`: fetches purchase events from `GET /api/v1/inventory/purchases?page=N&vendor_id=X` with pagination; fetches pending items from `GET /api/v1/inventory/purchases/pending`
- `loadStock()`: fetches computed stock levels from `GET /api/v1/inventory/stock`
- Skeleton loading state during all fetches (`showSkeleton()`)
- Inline error + retry on fetch failure (`showInlineError()`)
- Empty states: "No purchases yet" (History), "No stock data" (Stock)
- "Needs Review" badge on pending purchase events at top of history list
- "Load more" pagination button in History tab
- History tab label shows pending count in warn color: "History (N)"
- Auth check via `GET /api/v1/me` on DOMContentLoaded — redirects to login on 401
- Trends and Cost tabs: "coming soon" copy updated per D-13 deferral
- `@keyframes pulse` animation added for skeleton loader

**sw.js rebuilt** with updated content hash for inventory.html (`ff7c3140`).

## Deviations from Plan

**1. [Rule 1 - Bug] render() called renderHistory() which no longer exists**
- **Found during:** Task 1 review
- **Issue:** `show()` calls `render()` which called `renderHistory()`, but new architecture splits this into `loadHistory()` (async fetch) and `renderHistoryList()` (in-memory render). Direct call to `renderHistory()` would throw ReferenceError.
- **Fix:** Changed `render()` to call `renderHistoryList()` — re-renders from PURCHASES cache on tab switch; `loadHistory()` only called on init and vendor filter change.
- **Files modified:** inventory.html
- **Commit:** 4361261

**2. [Rule 2 - Missing] Unused Chart.js state vars removed**
- **Found during:** Task 1
- **Issue:** Old state vars (trendBarChart, ACTIVE_CHIPS, TRENDS_SUB_TAB, COST_* etc.) left from Chart.js implementation — dead code since Trends/Cost are now "coming soon".
- **Fix:** Removed unused vars; removed Chart.js script tag.
- **Files modified:** inventory.html
- **Commit:** 4361261

## Known Stubs

The following are intentional stubs per plan decisions (not errors):

- **Trends tab** (`#trends-container`): "coming soon" — sales data integration deferred per D-13. Future plan will wire Chart.js charts to real sales API.
- **Cost tab** (`#cost-container`): "coming soon" — food cost calculations deferred per D-13. Future plan will implement ingredient cost computation.
- **Stock level algorithm**: The API returns `total_quantity` and `avg_price`; client-side level determination (`>10 = high, >3 = medium, else low`) is a fallback. Production should have the backend compute levels with proper par-day logic.

## Self-Check: PASSED

- inventory.html exists: FOUND
- sw.js exists: FOUND
- Commits 4361261 and 34a2515 exist: FOUND
- MOCK_ count in inventory.html: 0
- api/v1/inventory endpoints: 4 (purchases, stock, vendors, purchases/pending)
