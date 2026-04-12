---
quick_task: 260412-c3f
title: Restructure PWA into ops console shell
tags: [pwa, launcher, navigation, service-worker]
key_files:
  created:
    - purchasing.html
  modified:
    - index.html
    - sw.js
    - manifest.json
decisions:
  - Back link uses plain anchor tag pointing to index.html for simplicity and SW compatibility
  - "Soon" tiles use div (not anchor) to prevent navigation and signal non-clickability
  - 2-column grid chosen for balanced layout at 480px max-width
metrics:
  duration: 2min
  completed: 2026-04-12
  tasks: 4
  files: 4
---

# Quick Task 260412-c3f: Restructure PWA into Ops Console Shell

**One-liner:** Launcher home screen with emoji tile grid replacing purchasing mockup as index.html, purchasing moved to purchasing.html with back link.

## What Was Done

The PWA was restructured from a single-purpose purchasing tool into an ops console shell. The existing purchasing mockup became `purchasing.html` (unchanged, plus a back link). A new `index.html` serves as the launcher with a 2-column grid of emoji tiles for each ops module.

## Tasks Completed

| # | Description | Commit | Files |
|---|-------------|--------|-------|
| 1 | Create purchasing.html with back-to-HQ link | 1eb3dcd | purchasing.html |
| 2 | Replace index.html with ops console launcher | 483960e | index.html |
| 3 | Bump SW cache to v3 and add purchasing.html | 2290a8d | sw.js |
| 4 | Rename PWA manifest to Yumyums HQ | 15fa6a9 | manifest.json |

## Design Decisions

- **Back link**: Simple `<a href="index.html">HQ</a>` with a chevron prefix — no JS, works offline, SW-friendly
- **Soon tiles**: `<div>` elements (not anchors) with `cursor:default` and `opacity:0.55` — visually distinct, not accidentally tappable
- **2-column grid**: `grid-template-columns: 1fr 1fr` at 480px fits 5 tiles cleanly (2+2+1)
- **CSS reuse**: All `--var` tokens, dark mode query, and `touch-action:manipulation` preserved verbatim from original

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

- Payroll, Scheduling, Hiring, BI tiles link to `#` (intentional placeholders, each requires its own future plan)
- These are cosmetic/navigational stubs only — they do not affect the plan's goal (launching purchasing.html)

## Self-Check

- [x] purchasing.html exists and contains purchasing mockup with back link
- [x] index.html exists with launcher grid
- [x] sw.js updated to v3 cache with purchasing.html in ASSETS
- [x] manifest.json updated to "Yumyums HQ"
- [x] All 4 commits present (1eb3dcd, 483960e, 2290a8d, 15fa6a9)

## Self-Check: PASSED
