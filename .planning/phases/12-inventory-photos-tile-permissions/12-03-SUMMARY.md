---
phase: 12-inventory-photos-tile-permissions
plan: "03"
subsystem: frontend
tags: [permissions, tile-grid, pwa, index, service-worker]
dependency_graph:
  requires: [Phase 11 /me/apps endpoint]
  provides: [Permission-filtered tile grid in index.html]
  affects: [index.html, sw.js]
tech_stack:
  added: []
  patterns: [localStorage cache-then-network, DOM removal via tile.remove()]
key_files:
  created: []
  modified:
    - index.html
    - sw.js
decisions:
  - Cache-then-network pattern: apply cached permissions immediately, then refresh from /api/v1/me/apps in background
  - tile.remove() used (not display:none) so grid reflows naturally with no gaps
  - No loading skeleton: tiles stay visible until /me/apps resolves (fast on local network; first-load only)
  - Offline graceful degradation: if /me/apps fetch fails, last-known cached permissions remain applied
metrics:
  duration_minutes: 1
  completed_date: "2026-04-18"
  tasks_completed: 2
  files_changed: 2
---

# Phase 12 Plan 03: Tile Permission Filtering Summary

**One-liner:** Permission-filtered HQ launcher grid using localStorage cache-then-network pattern against /api/v1/me/apps.

## What Was Built

index.html now filters the tile grid by user permissions on every page load. Users only see tools they have access to. The filtering uses a cache-then-network approach: cached permissions (from localStorage) are applied instantly on load, then refreshed from `/api/v1/me/apps` in the background. Grid reflows cleanly when tiles are removed (no gaps, no display:none).

## Tasks Completed

| Task | Description | Commit | Files |
|------|-------------|--------|-------|
| 1 | Add tile permission filtering to index.html | 4e3486d | index.html |
| 2 | Rebuild service worker with updated index.html | 92b8eb3 | sw.js |

## Implementation Details

**New constants in index.html:**
- `TILE_SLUGS` — maps HTML filenames to app slugs (e.g. `workflows.html` → `operations`)
- `APPS_CACHE_KEY = 'hq_apps'` — localStorage key for permissions cache

**New function `filterTilesByPermissions(apps)`:**
- Builds a `Set` of allowed slugs from the apps array
- Iterates `.grid .tile` elements; looks up slug via `href` in TILE_SLUGS
- Calls `tile.remove()` for tiles not in the allowed set
- "Soon" tiles (no `href`) are untouched

**Extended `checkAuth()`:**
1. Fetches `/api/v1/me` as before (401 redirects to login.html)
2. Reads `localStorage[APPS_CACHE_KEY]` — if cached, calls `filterTilesByPermissions` immediately
3. Fetches `/api/v1/me/apps` — on success, updates cache and re-calls `filterTilesByPermissions`
4. On network error: keeps cached tiles (graceful offline degradation)

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all tile filtering is wired to live `/api/v1/me/apps` endpoint.

## Self-Check

- [x] index.html modified with filterTilesByPermissions, TILE_SLUGS, APPS_CACHE_KEY, /me/apps fetch
- [x] sw.js rebuilt with updated index.html content hash (42bf446957873c77e8735e2631e74b65)
- [x] Commit 4e3486d exists (Task 1)
- [x] Commit 92b8eb3 exists (Task 2)

## Self-Check: PASSED
