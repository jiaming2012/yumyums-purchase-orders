---
phase: 01-onboarding-video-upgrade
plan: 01
subsystem: api
tags: [go, postgres, ffmpeg, do-spaces, presigned-url, video, onboarding]

requires:
  - phase: 12-inventory-photos-tile-permissions
    provides: photos.GeneratePresignedPutURL, photos.PublicURL, photos.NewSpacesPresigner reused for video upload
  - phase: 11-onboarding-users-admin
    provides: ob_video_parts, ob_progress, SaveProgress, GetHireTraining baseline

provides:
  - "Migration 0028: ob_video_parts.thumbnail_url, ob_progress.max_watched_time, video_watch_position progress type"
  - "VideoPresignHandler: POST /api/v1/videos/presign — browser upload to DO Spaces"
  - "VideoProcessHandler: POST /api/v1/videos/process — background FFmpeg conversion + thumbnail"
  - "Extended VideoPart struct with ThumbnailURL and MaxWatchedTime"
  - "Extended SaveProgress with maxWatchedTime parameter"
  - "GetHireTraining populates MaxWatchedTime per video part from both progress types"

affects:
  - 01-02
  - 01-03
  - onboarding-video-playback

tech-stack:
  added: ["os/exec (FFmpeg)", "os.MkdirTemp for temp video processing"]
  patterns:
    - "Fire-and-forget goroutine for FFmpeg processing returns 202 Accepted immediately"
    - "video_watch_position progress type tracks furthest watch position without marking complete"
    - "GREATEST() upsert for max_watched_time ensures only forward progress is stored"

key-files:
  created:
    - backend/internal/db/migrations/0028_ob_video_upgrade.sql
    - backend/internal/onboarding/video.go
  modified:
    - backend/internal/onboarding/db.go
    - backend/internal/onboarding/handler.go
    - backend/cmd/server/main.go

key-decisions:
  - "video_watch_position is a separate progress_type from video_part — isSectionComplete only checks video_part so watch position tracking never falsely marks sections complete"
  - "GREATEST(ob_progress.max_watched_time, EXCLUDED.max_watched_time) in ON CONFLICT clause ensures only forward-progress watch positions are persisted"
  - "VideoProcessHandler fires goroutine and returns 202 immediately — FFmpeg can take minutes for large videos"
  - "content_type validated against allowlist (video/mp4, video/quicktime, video/webm) before presigning"

patterns-established:
  - "Video presign pattern: validate content_type → build key with videos/onboarding/{tmpl}/{part}/{file} → 30min TTL"
  - "FFmpeg wrapper pattern: exec.CommandContext with separate arg slice (not shell string), combined output on error"
  - "processVideo cleanup: os.MkdirTemp + defer os.RemoveAll ensures no temp file leaks"

requirements-completed: [SC-01, SC-02, SC-03, SC-05]

duration: 15min
completed: 2026-04-19
---

# Phase 01 Plan 01: Video Backend Infrastructure Summary

**Go backend for video upload (presigned PUT URLs), FFmpeg conversion + thumbnail extraction, and watch-position persistence — extending ob_progress and ob_video_parts schemas**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-04-19T21:47:00Z
- **Completed:** 2026-04-19T21:51:10Z
- **Tasks:** 2
- **Files modified:** 5 (created 3, modified 2)

## Accomplishments

- Migration 0028 adds `thumbnail_url` to `ob_video_parts`, `max_watched_time` to `ob_progress`, and `video_watch_position` to the CHECK constraint
- `video.go` implements FFmpeg wrapper (convert + thumbnail), Spaces download/upload helpers, and a `processVideo` orchestrator
- Two new API endpoints: `POST /api/v1/videos/presign` (browser upload URL) and `POST /api/v1/videos/process` (background processing, returns 202)
- `VideoPart` struct extended with `ThumbnailURL` and `MaxWatchedTime` — both returned in `hireTraining` API response
- `SaveProgress` extended with optional `maxWatchedTime *float64` — existing callers unaffected (nil passed through handler)

## Task Commits

1. **Task 1: DB migration + VideoPart struct extension** - `fd61b8f` (feat)
2. **Task 2: Video presign + FFmpeg process endpoints and route wiring** - `b3f97aa` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified

- `backend/internal/db/migrations/0028_ob_video_upgrade.sql` - Schema additions: thumbnail_url, max_watched_time, video_watch_position type
- `backend/internal/onboarding/video.go` - FFmpeg conversion, thumbnail extraction, Spaces download/upload, processVideo orchestrator
- `backend/internal/onboarding/db.go` - Extended VideoPart struct, getVideoParts, GetHireTraining, SaveProgress, insertSectionsTx, CreateVideoPartInput
- `backend/internal/onboarding/handler.go` - VideoPresignHandler, VideoProcessHandler, extended SaveProgressHandler body
- `backend/cmd/server/main.go` - Route registration for /videos/presign and /videos/process

## Decisions Made

- `video_watch_position` is a separate `progress_type` from `video_part` — `isSectionComplete` only checks `video_part` rows, so watch position tracking never falsely marks sections complete
- GREATEST() upsert semantics for `max_watched_time` — ensures only forward progress is stored (rewind doesn't reset)
- `VideoProcessHandler` returns 202 Accepted immediately, fires goroutine — FFmpeg conversion can take minutes for large files
- Content-type allowlist validation before presigning (video/mp4, video/quicktime, video/webm)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None — build compiled cleanly on first attempt after adding `context` import to handler.go (needed for `context.Background()` in goroutine).

## Known Stubs

None — all wired. The `thumbnail_url` and `max_watched_time` fields are nullable but fully functional: populated after `POST /videos/process` runs FFmpeg. Frontend plans (01-02, 01-03) will consume these fields.

## Next Phase Readiness

- Backend APIs complete: presign upload, trigger processing, watch position save, hireTraining response
- Migration ready to apply against production DB
- Frontend Builder (01-02) can implement video upload using `POST /videos/presign` + `POST /videos/process`
- Frontend Runner (01-03) can implement video playback with thumbnail display and watch progress via `saveProgress` with `max_watched_time`

---
*Phase: 01-onboarding-video-upgrade*
*Completed: 2026-04-19*
