# Phase 1: Onboarding Video Upgrade - Context

**Gathered:** 2026-04-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Training videos can be uploaded to DO Spaces or linked by URL, play inline with poster thumbnails in full-screen mode, and the watched checkbox only checks after 95%+ of the video is played. Server-side FFmpeg converts non-MP4 uploads to MP4. Thumbnails auto-extracted with optional admin override.

</domain>

<decisions>
## Implementation Decisions

### Video Upload (Builder)
- **D-01:** Upload/URL radio toggle per video part — Upload is the first/default option, URL is second
- **D-02:** Accepted formats: MP4, MOV, WebM. API rejects all other formats. Server converts MOV/WebM to MP4 via FFmpeg after upload — stored format is always MP4
- **D-03:** Max file size: 200 MB
- **D-04:** Inline progress bar during upload (important for large files on mobile)
- **D-05:** Presigned PUT to DO Spaces — reuse existing photos presign pattern with `videos/onboarding/{template_id}/{part_id}.mp4` path prefix

### Inline Player (Training Runner)
- **D-06:** Always full-screen — tapping play enters full-screen mode immediately via HTML5 `<video>` requestFullscreen API
- **D-07:** Two-tap flow: tap video part row → shows thumbnail with play button overlay → tap play → full-screen playback
- **D-08:** Native `<video>` controls in full-screen (no custom player)

### Watch Enforcement
- **D-09:** 95%+ playback threshold required to mark video part as watched — track via `timeupdate` events, check `currentTime / duration >= 0.95` or `ended` event
- **D-10:** Seeking restricted to already-watched positions only — track `maxWatchedTime`, intercept `seeking` event and clamp `currentTime` to `maxWatchedTime` if user tries to skip ahead
- **D-11:** Progress saves on close — persist furthest-watched position (maxWatchedTime) via saveProgress API so user can resume from where they left off
- **D-12:** Checkbox is NOT manually toggleable for video parts — only the watch-to-completion mechanism can check it

### Thumbnails
- **D-13:** Auto-extract frame from video via server-side FFmpeg (grab frame at ~2 seconds) after upload/conversion completes
- **D-14:** Admin can upload a custom thumbnail image to override the auto-generated one (optional — auto-generated used by default)
- **D-15:** Thumbnail stored in DO Spaces alongside video: `videos/onboarding/{template_id}/{part_id}_thumb.jpg`

### Claude's Discretion
- FFmpeg invocation strategy (subprocess, worker goroutine, or background job)
- Thumbnail extraction timing (inline during conversion or separate async step)
- Video player CSS/styling for the thumbnail + play button overlay
- Error handling for failed conversions (retry, notify admin, fallback to original format)
- Whether to store maxWatchedTime in ob_progress or a new column

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing Presign Infrastructure
- `backend/internal/photos/spaces.go` — DO Spaces presigner (NewSpacesPresigner, BaseEndpoint, UsePathStyle)
- `backend/internal/photos/handler.go` — PresignUploadHandler pattern (POST presign → return url + public_url)
- `workflows.html` lines 1529-1602 — handleFailPhotoCaptureClick shows the full presign → PUT → store URL flow

### Onboarding Data Model
- `backend/internal/onboarding/db.go` — SaveProgress, video_part progress_type, ob_progress table
- `backend/internal/onboarding/handler.go` — SaveProgressHandler reads {item_id, progress_type, checked}
- `backend/internal/onboarding/seed.go` — Video part structure (Title, Description, URL, SortOrder)

### Onboarding Frontend
- `onboarding.html` lines 315-335 — Current video part rendering (checkbox + "Watch Video" link)
- `onboarding.html` lines 660-678 — Builder video part editor (title, description, URL inputs)
- `onboarding.html` lines 886-910 — toggle-video-part handler

### Project Conventions
- `CLAUDE.md` — Event delegation via data-action, state-first rendering, autoSaveField persistence rule

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `photos/spaces.go` presigner — extend for video uploads with different path prefix and larger size limits
- `photos/handler.go` PresignUploadHandler — same pattern for video presign endpoint
- `saveProgress(itemId, 'video_part', checked, sections, hireId)` — existing progress persistence for video parts
- `ob_progress` table with `progress_type` discriminator — can store video watch position

### Established Patterns
- Presign flow: frontend requests presigned URL → PUT file directly to Spaces → store public_url in app state
- Event delegation: all click handlers routed via `data-action` attributes on a single container listener
- State-first rendering: mutate JS state → call render function → DOM updates from state

### Integration Points
- Builder: `renderOBBuilderItem()` renders video part editors — add upload/URL toggle here
- Runner: video part rendering at line 315 — replace "Watch Video" link with inline thumbnail + player
- Backend: new presign endpoint for videos (or extend existing photos presign with path_prefix param)
- `saveProgress` API: extend to accept `max_watched_time` for resume support

</code_context>

<specifics>
## Specific Ideas

- DO Spaces path: `videos/onboarding/{template_id}/{part_id}.mp4` for easy admin identification in S3 browser
- Thumbnail path: `videos/onboarding/{template_id}/{part_id}_thumb.jpg`
- FFmpeg frame extraction: `ffmpeg -i input.mp4 -ss 00:00:02 -frames:v 1 -q:v 2 thumb.jpg`
- FFmpeg conversion: `ffmpeg -i input.mov -c:v libx264 -c:a aac -movflags +faststart output.mp4`

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-onboarding-video-upgrade*
*Context gathered: 2026-04-19*
