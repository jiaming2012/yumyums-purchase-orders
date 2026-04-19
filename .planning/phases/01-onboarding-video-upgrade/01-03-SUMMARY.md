---
phase: 01-onboarding-video-upgrade
plan: 03
subsystem: onboarding-frontend
status: checkpoint
tags: [video-player, inline-player, seeking-restriction, watch-enforcement, persistence]
dependency_graph:
  requires: [01-01, 01-02]
  provides: [inline-video-player, watch-enforcement, video-persistence-tests]
  affects: [onboarding.html, tests/persistence.spec.js]
tech_stack:
  added: []
  patterns: [video-element-native-controls, fullscreen-api, timeupdate-enforcement, cloneNode-listener-reset]
key_files:
  created: []
  modified:
    - onboarding.html
    - tests/persistence.spec.js
    - sw.js
decisions:
  - "Use cloneNode(true) to reset video element listeners on each initVideoPlayer call — prevents stacked event handlers"
  - "renderTrainingDetail() dispatches to active runner (my or mgr) based on obState/mgrState view"
  - "Test 1 uses API round-trip only (not UI navigation) to avoid section-lock fragility"
  - "Test assertion for checked=false removed from Test 1 (covered by Test 2) to prevent cross-test state leak"
metrics:
  duration_min: 30
  completed_date: "2026-04-19"
  tasks_completed: 2
  tasks_total: 3
  files_modified: 3
---

# Phase 01 Plan 03: Inline Video Player with Watch Enforcement — Summary

**One-liner:** Inline video player with iOS-safe poster, seeking restriction to watched position, 95%+/ended auto-check, watch position persistence via video_watch_position progress_type, and 3 API-level persistence regression tests.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Inline video player with thumbnail, fullscreen, seeking restriction, watch enforcement | c133b69 | onboarding.html, sw.js, sw.js.map |
| 2 | Persistence regression test for max_watched_time | f896646 | tests/persistence.spec.js |

## Checkpoint Pending

**Task 3:** Human verification of complete video upload and playback flow — awaiting approval.

## What Was Built

### Task 1: Inline Video Player

**CSS:**
- `.video-thumb-wrap` — thumbnail wrapper with play button overlay (`:after` pseudo-element)
- `.video-thumb-wrap.watched` — green tint on play button when part is complete
- `.video-player-modal` — fixed-position fullscreen modal container
- `.video-player-close` — close button overlay

**HTML:**
- `<div class="video-player-modal" id="video-modal">` — modal with `<video id="video-player" playsinline controls>`

**JavaScript functions:**
- `initVideoPlayer(videoEl, partId, videoUrl, thumbUrl, initialMaxWatched, secId, hireId)` — sets poster, seeking restriction, 95% check, pause save, autoplay+fullscreen
- `enterFullscreen(videoEl)` — iOS `webkitEnterFullscreen` + standard `requestFullscreen` fallback
- `markVideoPartWatched(partId, secId, hireId, maxWatchedTime)` — POST `video_part` progress_type to saveProgress, recompute section state, re-render
- `saveVideoWatchProgress(partId, maxWatchedTime)` — POST `video_watch_position` progress_type (fire-and-forget)
- `renderTrainingDetail()` — dispatches to `renderMyRunner()` or `renderMgrRunner()` based on active view

**Event delegation updates:**
- Replaced `toggle-video-part` + `watch-video` handlers with `play-video` (show modal, init player) and `close-video` (pause, save, teardown)
- Added `fullscreenchange` + `webkitfullscreenchange` listeners to save progress when user exits fullscreen

**Runner rendering changes:**
- Video part checkbox now `disabled` (no data-action) per D-12
- Each video part renders a `.video-thumb-wrap[data-action="play-video"]` with `data-part-id`, `data-video-url`, `data-thumb-url`, `data-max-watched`, `data-sec-id`, `data-hire-id`
- Thumbnail shown if `thumbnail_url` present; no-thumbnail fallback shows play button only

### Task 2: Persistence Regression Tests

Three tests added under `// --- Video watch progress persistence ---` in `tests/persistence.spec.js`:

1. **video max_watched_time survives back-to-list and reopen** — Saves `video_watch_position` with 30s, re-fetches via API, asserts `max_watched_time >= 30`
2. **video_watch_position progress does not mark section complete** — Saves `video_watch_position` only, verifies section state is not `complete`/`signed_off` and video part `checked=false`
3. **max_watched_time persists after video marked as watched** — Saves `video_part` with 120s, re-fetches, verifies `checked=true` and `max_watched_time >= 120`

All tests use direct API calls against the seeded Kitchen Basics Training template (no UI fragility from locked sections).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing feature] Added renderTrainingDetail() helper**
- **Found during:** Task 1 — markVideoPartWatched needed to call back into the active runner render
- **Issue:** Plan showed `renderTrainingDetail()` called in markVideoPartWatched but function didn't exist
- **Fix:** Added `renderTrainingDetail()` that routes to `renderMyRunner()` or `renderMgrRunner()` based on view state
- **Files modified:** onboarding.html
- **Commit:** c133b69

**2. [Rule 1 - Bug] Test 1 assertion on checked=false removed**
- **Found during:** Task 2 RED phase — Test 1 checked `videoPartAfter.checked === false` but that state can be true if Test 3 ran first (shared DB)
- **Fix:** Removed the redundant `checked` assertion from Test 1 (Test 2 already covers that behavior); replaced with `max_watched_time > 0` confirmation
- **Files modified:** tests/persistence.spec.js
- **Commit:** f896646

## Known Stubs

None — all code paths implemented and functional. Thumbnail display degrades gracefully when `thumbnail_url` is null (shows play button without image). Task 3 human verification pending.

## Self-Check: PASSED

Files confirmed:
- onboarding.html — contains `initVideoPlayer`, `video-modal`, `play-video` data-action, `maxWatchedTime`, `video_watch_position`
- sw.js — rebuilt with updated onboarding.html content hash
- tests/persistence.spec.js — contains 3 new video persistence tests all passing

Commits confirmed:
- c133b69 — feat(01-03): inline video player...
- f896646 — test(01-03): video watch progress persistence...
