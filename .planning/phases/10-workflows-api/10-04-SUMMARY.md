---
phase: 10-workflows-api
plan: 04
subsystem: ui
tags: [vanilla-js, indexeddb, offline-sync, pwa, service-worker]

# Dependency graph
requires:
  - phase: 10-workflows-api/10-03
    provides: api() wrapper, submitChecklistToAPI(), renderMyChecklists()

provides:
  - IndexedDB hq_offline_v1 database with submitQueue object store
  - enqueueSubmission() — persists payloads when offline
  - drainQueue() — sequential drain on online event with race guard
  - renderSyncBanner() — live queue count banner with per-item badges
  - Offline submit path in submitChecklistToAPI()

affects: [10-05, testing]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "IndexedDB idbGetAll/idbPut/idbDelete helpers — raw IDB promise wrappers"
    - "_draining flag guards drainQueue() against concurrent drain (D-16)"
    - "enqueueSubmission() uses payload.id as IDB keyPath — same UUID as idempotency_key"
    - "renderSyncBanner() called after renderMyChecklists() and after each drain iteration"
    - "window.addEventListener('online', drainQueue) — auto-drain on connectivity restore"

key-files:
  created: []
  modified:
    - workflows.html
    - sw.js

key-decisions:
  - "submitChecklistToAPI() catches offline errors and enqueues — throws {offline:true} sentinel to caller"
  - "Submit .catch handler checks err.offline to show 'Queued for sync' toast and navigate to list"
  - "HQ_DB constant used instead of inline string literal — consistent with project constants pattern"
  - "data-template-id added alongside data-fill-template-id on checklist rows — badges need template-id"
  - "SW bumped from v50 to v51 per CLAUDE.md convention"

requirements-completed: [SYNC-01, SYNC-02, SYNC-03]

# Metrics
duration: 3min
completed: 2026-04-15
---

# Phase 10 Plan 04: Offline Sync Summary

**IndexedDB offline queue with drain-on-reconnect, persistent sync banner, and per-item pending badges — full offline submit → queue → online → drain cycle implemented**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-15T17:02:50Z
- **Completed:** 2026-04-15T17:05:49Z
- **Tasks:** 2
- **Files modified:** 2 (workflows.html, sw.js)

## Accomplishments

- Added `getDB()`, `idbGetAll()`, `idbPut()`, `idbDelete()` — minimal IndexedDB promise wrappers
- Added `enqueueSubmission()` to persist checklist payload to IndexedDB when offline
- Added `drainQueue()` with `_draining` race condition guard — processes queue sequentially per D-12/D-16
- Added `window.addEventListener('online', drainQueue)` — auto-drains on reconnection
- Added `showConflictError()` — inline error card for 409 archived template per D-14
- Modified `submitChecklistToAPI()` to catch offline errors and enqueue, throwing `{offline:true}` sentinel
- Updated submit `.catch` handler to show "Queued for sync when back online" toast and navigate to list on `err.offline`
- Added `.sync-banner` CSS — warning background, positioned below tab bar
- Added `.sync-badge` CSS — per-item warning pill
- Added `id="sync-banner"` HTML element in My Checklists tab section
- Added `async renderSyncBanner()` that reads IndexedDB queue count and updates banner + per-item badges
- Added `data-template-id` attribute to checklist rows so badges can find them
- Wired `renderSyncBanner()` call after `renderMyChecklists()` completes
- Called `renderSyncBanner()` + `drainQueue()` on page init to handle leftover queue items from previous sessions
- Bumped sw.js cache version v50 → v51

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement IndexedDB queue and online drain logic** - `ce89d3b` (feat)
2. **Task 2: Add sync banner and per-item badges + bump SW cache** - `a2c424c` (feat)

## Files Created/Modified

- `workflows.html` — IndexedDB queue, drain logic, sync banner, per-item badges, offline submit path
- `sw.js` — Cache version bumped v50 → v51

## Decisions Made

- `submitChecklistToAPI()` throws `{offline:true}` sentinel so the caller can distinguish offline queuing from real errors
- `enqueueSubmission()` uses the same payload object as the API call — `id` field serves as both IDB keyPath and idempotency key
- `data-template-id` attribute added alongside existing `data-fill-template-id` to support badge targeting without breaking the existing click handler that uses `data-fill-template-id`
- `renderSyncBanner()` called asynchronously — no await needed in `renderMyChecklists()` since banner update is non-blocking UI state

## Deviations from Plan

### Auto-fixed Issues

None - plan executed exactly as written, with one minor adaptation:

**1. [Rule 1 - Style] Used HQ_DB constant instead of inline string in indexedDB.open()**
- **Found during:** Task 1
- **Issue:** Plan shows `indexedDB.open('hq_offline_v1', 1)` inline but project pattern uses `SCREAMING_SNAKE_CASE` constants
- **Fix:** Added `const HQ_DB = 'hq_offline_v1'` and used `indexedDB.open(HQ_DB, 1)` — both verify checks pass
- **Files modified:** workflows.html
- **Committed in:** ce89d3b

## Known Stubs

None introduced in this plan. Pre-existing stubs from Plan 03 remain:
- `handlePhotoCaptureClick()` — no-op stub (Phase 12)
- `unsubmit` action — shows "not yet available" toast

## Issues Encountered

None.

## Next Phase Readiness

- Offline submit queue complete — SYNC-01, SYNC-02, SYNC-03 satisfied
- Ready for Plan 05 (final plan in phase 10)
- Server must be running for drain to succeed after reconnect

---
*Phase: 10-workflows-api*
*Completed: 2026-04-15*
