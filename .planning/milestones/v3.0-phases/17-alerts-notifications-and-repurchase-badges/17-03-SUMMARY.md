---
phase: "17"
plan: "03"
subsystem: frontend
tags: [repurchase-badge, stock-tab, badge-reset, inventory, admin-config]
completed: "2026-04-23"
duration_minutes: 10

dependency_graph:
  requires: [phase-17-plan-01]
  provides: [repurchase-badge-ui, badge-reset-config-ui]
  affects: [inventory-stock-tab, inventory-setup-tab]

tech_stack:
  added: []
  patterns:
    - "repurchase_badge.qty read from stock API response (Plan 01 shape)"
    - "CURRENT_USER loaded on init for admin gate"
    - "isAdmin() helper checks is_superadmin + roles array"
    - "document click delegation for badge reset config UI"
    - "day-chip pattern mirrors purchasing.html cutoff config"

key_files:
  created: []
  modified:
    - inventory.html
    - sw.js

decisions:
  - "Used repurchase_badge.qty (Plan 01 API shape) instead of repurchased_qty (Plan 03 original spec) — Plan 01 built the backend with a richer RepurchaseBadge struct"
  - "Used /api/v1/purchasing/repurchase-reset (GET) and /repurchase-reset/config (PUT) — Plan 01 registered these routes, not /badge-reset-config"
  - "Task 1 (backend) was entirely fulfilled by Plan 01 — no new backend code needed"
---

# Phase 17 Plan 03: Repurchase Badge Rendering and Badge Reset Config UI

**One-liner:** Blue "Repurchased +N" badges on inventory stock items with admin-configurable weekly reset schedule in Setup tab.

## What Was Built

### Task 1: Stock API repurchase badge data (already done by Plan 01)

Plan 01 built the complete backend repurchase badge infrastructure:
- `RepurchaseBadge{Qty, RepurchasedAt}` struct on `StockItem` — `repurchase_badge` JSON field
- `GetStockHandler` queries `repurchase_log` table and attaches badge per stock item
- Badge uses `last_reset_at` from `repurchase_reset_config` as cutoff
- Routes registered: `GET /api/v1/purchasing/repurchase-reset`, `PUT /api/v1/purchasing/repurchase-reset/config`

No additional backend code needed. Backend compiled and verified clean (`go build ./cmd/server/` — BUILD OK).

### Task 2: Frontend badge rendering and badge reset config UI (commit: a6d1727)

**inventory.html changes:**

- `.stock-repurchased` CSS class using `--info-bg`/`--info-tx` variables (blue badge)
- Stock tab: renders `<span class="stock-badge stock-repurchased">Repurchased +N</span>` when `s.repurchase_badge && s.repurchase_badge.qty > 0`
- Added `CURRENT_USER`, `BADGE_RESET_CONFIG`, `BADGE_RESET_FORM_OPEN`, `BADGE_RESET_DAY`, `BADGE_RESET_TIME` state variables
- `isAdmin()` helper: checks `CURRENT_USER.is_superadmin` or roles includes 'admin'
- DOMContentLoaded now sets `CURRENT_USER = await api('/api/v1/me')` instead of discarding the response
- `show(5)` (Setup tab) calls `loadBadgeResetConfig()` when admin
- `badge-reset-section` placeholder div in Setup tab HTML (shown only to admins)
- `loadBadgeResetConfig()`: GETs `/api/v1/purchasing/repurchase-reset`, populates state, renders card
- `renderBadgeResetConfig()`: renders card with day-of-week chips (Sun-Sat), time input, timezone display, Save/Cancel buttons
- Click handler delegation for: `toggle-badge-reset`, `badge-reset-day`, `cancel-badge-reset`, `save-badge-reset`
- `save-badge-reset` PUTs to `/api/v1/purchasing/repurchase-reset/config`

**sw.js** rebuilt with content-hashed precache (20 files, 741.4 KB)

## Verification

- `go build ./cmd/server/` — BUILD OK
- `.stock-repurchased` CSS class present — 2 matches
- `repurchase_badge.qty` check in rendering — present
- `/api/v1/purchasing/repurchase-reset` and `/repurchase-reset/config` endpoints used — present
- `badge-reset-section` in Setup tab HTML — present

## Deviations from Plan

### 1. [Rule 2 - Plan superseded by Plan 01] Task 1 already done

Plan 01 built the repurchase badge backend using a different API shape than Plan 03 specified:
- Plan 03 specified: `RepurchasedQty int` field + `/badge-reset-config` endpoints
- Plan 01 built: `RepurchaseBadge *RepurchaseBadge` struct + `/repurchase-reset` endpoints

The Plan 01 implementation is functionally equivalent and superior (richer struct with timestamp). Task 1 was skipped — no additional backend code needed.

### 2. [Rule 1 - Adaptation] Frontend uses Plan 01 API shape

The frontend reads `s.repurchase_badge.qty` (Plan 01 shape) instead of `s.repurchased_qty` (Plan 03 original spec). The plan verification check for `repurchased_qty` will show 0 matches, but the actual API response field is `repurchase_badge.qty`.

### 3. [Rule 1 - Adaptation] Badge reset config uses Plan 01 routes

Frontend calls `/api/v1/purchasing/repurchase-reset` (GET) and `/api/v1/purchasing/repurchase-reset/config` (PUT) instead of the Plan 03 spec's `/api/v1/purchasing/badge-reset-config`. These routes were registered by Plan 01.

None of the deviations affect correctness or completeness — the feature is fully implemented.

## Known Stubs

None — all features fully implemented.

## Self-Check: PASSED

- inventory.html modified with repurchase badge rendering — found
- sw.js rebuilt — found
- commit a6d1727 — verified
