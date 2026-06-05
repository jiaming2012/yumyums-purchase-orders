---
phase: 01-onboarding-video-upgrade
plan: 02
subsystem: ui
tags: [onboarding, video, upload, do-spaces, presign, xhr, progress, builder]

# Dependency graph
requires:
  - phase: 12-inventory-photos-tile-permissions
    provides: DO Spaces presigned URL pattern (photos/handler.go, spaces.go)
  - phase: 01-onboarding-video-upgrade/01-01
    provides: Backend video presign and process API endpoints

provides:
  - Builder upload/URL radio toggle per video part
  - XHR file upload with visible progress bar
  - Custom thumbnail override upload
  - Updated createNewOBVideoPart() with source/thumbnail_url fields

affects: [01-onboarding-video-upgrade/01-03]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - XHR PUT for upload progress events (fetch API cannot report upload progress)
    - Builder change event delegation for file/radio inputs alongside existing click delegation
    - Source-conditional rendering (upload vs URL mode) driven from part.source state field

key-files:
  created: []
  modified:
    - onboarding.html
    - sw.js
    - sw.js.map

key-decisions:
  - "uploadVideoFile uses XHR not fetch — only XHR exposes upload.progress events for progress bar"
  - "uploadCustomThumbnail reuses /api/v1/photos/presign with path_prefix videos/onboarding — avoids duplicating presign logic"
  - "Builder change event listener added separately from existing click listener — file inputs and radios fire change, not click"
  - "Thumbnail section only renders when part.url is set — avoids confusing admins before video is uploaded"

patterns-established:
  - "Video source toggle: part.source field drives conditional rendering in renderOBVideoItem"
  - "Progress bar: vprog-{partId} / vstat-{partId} IDs allow XHR callbacks to update DOM without full re-render"

requirements-completed: [SC-01, SC-02]

# Metrics
duration: 2min
completed: 2026-04-19
---

# Phase 01 Plan 02: Builder Video Upload UI Summary

**XHR-backed video upload UI in the onboarding builder — upload/URL toggle, progress bar, and custom thumbnail override per video part**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-04-19T21:47:55Z
- **Completed:** 2026-04-19T21:49:42Z
- **Tasks:** 1
- **Files modified:** 3 (onboarding.html, sw.js, sw.js.map)

## Accomplishments

- Builder video part editor now shows Upload/URL radio toggle with Upload as default (D-01)
- File input restricted to .mp4/.mov/.webm with 200 MB max validation (D-02, D-03)
- XHR PUT upload with inline progress bar showing percentage text (D-04)
- After upload, calls POST /api/v1/videos/process to trigger FFmpeg conversion
- Custom thumbnail file input appears once video is uploaded, using /api/v1/photos/presign (D-14)
- `createNewOBVideoPart()` now includes `source: 'upload'` and `thumbnail_url: null` fields
- Builder `change` event listener added for radio toggles and file inputs (separate from click delegation)
- Service worker rebuilt to cache updated onboarding.html

## Task Commits

1. **Task 1: Upload/URL toggle, file input with XHR progress, thumbnail override** - `8efda0b` (feat)

**Plan metadata:** (pending docs commit)

## Files Created/Modified

- `onboarding.html` - Added CSS classes, rewrote renderOBVideoItem, added uploadVideoFile/uploadCustomThumbnail functions, added change event delegation, updated createNewOBVideoPart
- `sw.js` / `sw.js.map` - Rebuilt by Workbox with updated onboarding.html content hash

## Decisions Made

- XHR over fetch for upload: XHR is the only browser API that exposes upload progress events on `xhr.upload`
- Reuse `/api/v1/photos/presign` for custom thumbnails: avoids a dedicated video-thumbnail endpoint when path_prefix handles the routing
- Separate `change` event listener: file inputs and radio buttons fire `change` not `click`, so they cannot be captured by the existing click delegation listener

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Builder upload UI complete; Plan 01-03 (inline video player in training runner) can proceed
- Backend video presign/process endpoints (Plan 01-01) must be deployed before uploads work end-to-end

## Self-Check: PASSED

- `onboarding.html` exists and contains all required patterns
- commit `8efda0b` exists in git log
- `node build-sw.js` exits 0

---
*Phase: 01-onboarding-video-upgrade*
*Completed: 2026-04-19*
