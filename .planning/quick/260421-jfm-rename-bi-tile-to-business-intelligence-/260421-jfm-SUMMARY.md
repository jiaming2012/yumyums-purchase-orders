---
phase: quick
plan: 260421-jfm
subsystem: frontend/index.html
tags: [ui, tiles, rename]
dependency_graph:
  requires: []
  provides: ["Updated tile labels in HQ home grid"]
  affects: ["index.html", "sw.js"]
tech_stack:
  added: []
  patterns: ["Workbox SW rebuild after HTML changes"]
key_files:
  created: []
  modified: ["index.html", "sw.js", "sw.js.map", "workbox-336fc703.js"]
decisions: []
metrics:
  duration: 3
  completed_date: "2026-04-21"
---

# Quick 260421-jfm: Rename BI and Purchasing Tiles Summary

**One-liner:** Renamed HQ home grid tiles from "BI" to "Business Intelligence" and "Purchasing" to "Purchase Orders" for clearer crew-facing labels.

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Rename BI and Purchasing tiles | 9503bdc | index.html, sw.js, sw.js.map, workbox-336fc703.js |

## What Was Done

- Changed `<div class="tile-title">BI</div>` to `<div class="tile-title">Business Intelligence</div>` on line 73 of index.html
- Changed `<div class="tile-title">Purchasing</div>` to `<div class="tile-title">Purchase Orders</div>` on line 46 of index.html
- Ran `node build-sw.js` to regenerate the service worker with updated content hashes (new workbox chunk: workbox-336fc703.js)
- TILE_SLUGS mapping and hrefs left unchanged — only display labels updated

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None.

## Self-Check: PASSED

- index.html contains "Business Intelligence": FOUND
- index.html contains "Purchase Orders": FOUND
- Old "BI" tile-title: NOT present (correct)
- Old "Purchasing" tile-title: NOT present (correct)
- Commit 9503bdc: FOUND
