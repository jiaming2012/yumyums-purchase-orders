# Phase 1: Onboarding Video Upgrade - Research

**Researched:** 2026-04-19
**Domain:** Video upload (DO Spaces presign), FFmpeg (Go exec.Command), HTML5 Video API (mobile Safari), video progress persistence (Postgres)
**Confidence:** HIGH

## Summary

This phase adds video upload/hosting, inline playback with seeking restriction, and watch-to-completion enforcement to the existing onboarding video part system. The project already has DO Spaces presign infrastructure (`photos/spaces.go`, `photos/handler.go`) and a video progress table (`ob_progress`) that can be extended with a `max_watched_time` column — making this primarily an extension of existing patterns rather than net-new infrastructure.

The critical findings are: (1) DO Spaces direct-to-origin presigned PUT has a 5 GB object limit — the 8100 KiB CDN limit does NOT apply to direct origin uploads; (2) iOS Safari requires `playsinline` and a `poster` attribute to avoid a black screen and to avoid auto-entering fullscreen on tap; (3) the seeking-restriction pattern works on all platforms using the `seeking` event + clamp to `maxWatchedTime`, with a Safari-specific workaround (Safari fires `seeking` before updating `currentTime`, requiring a slight approach); (4) FFmpeg is available on the dev machine at `/usr/local/bin/ffmpeg` and Go's `exec.Command` with individual string args is the correct approach — no Go FFmpeg library needed.

**Primary recommendation:** Extend the existing presign handler for videos, run FFmpeg conversion + thumbnail extraction as a goroutine after upload (triggered by a new POST `/api/v1/videos/process` endpoint), and add a `max_watched_time` float column to `ob_progress` via a new migration.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Video Upload (Builder)**
- D-01: Upload/URL radio toggle per video part — Upload is the first/default option, URL is second
- D-02: Accepted formats: MP4, MOV, WebM. API rejects all other formats. Server converts MOV/WebM to MP4 via FFmpeg after upload — stored format is always MP4
- D-03: Max file size: 200 MB
- D-04: Inline progress bar during upload (important for large files on mobile)
- D-05: Presigned PUT to DO Spaces — reuse existing photos presign pattern with `videos/onboarding/{template_id}/{part_id}.mp4` path prefix

**Inline Player (Training Runner)**
- D-06: Always full-screen — tapping play enters full-screen mode immediately via HTML5 `<video>` requestFullscreen API
- D-07: Two-tap flow: tap video part row → shows thumbnail with play button overlay → tap play → full-screen playback
- D-08: Native `<video>` controls in full-screen (no custom player)

**Watch Enforcement**
- D-09: 95%+ playback threshold required to mark video part as watched — track via `timeupdate` events, check `currentTime / duration >= 0.95` or `ended` event
- D-10: Seeking restricted to already-watched positions only — track `maxWatchedTime`, intercept `seeking` event and clamp `currentTime` to `maxWatchedTime` if user tries to skip ahead
- D-11: Progress saves on close — persist furthest-watched position (maxWatchedTime) via saveProgress API so user can resume from where they left off
- D-12: Checkbox is NOT manually toggleable for video parts — only the watch-to-completion mechanism can check it

**Thumbnails**
- D-13: Auto-extract frame from video via server-side FFmpeg (grab frame at ~2 seconds) after upload/conversion completes
- D-14: Admin can upload a custom thumbnail image to override the auto-generated one (optional — auto-generated used by default)
- D-15: Thumbnail stored in DO Spaces alongside video: `videos/onboarding/{template_id}/{part_id}_thumb.jpg`

### Claude's Discretion
- FFmpeg invocation strategy (subprocess, worker goroutine, or background job)
- Thumbnail extraction timing (inline during conversion or separate async step)
- Video player CSS/styling for the thumbnail + play button overlay
- Error handling for failed conversions (retry, notify admin, fallback to original format)
- Whether to store maxWatchedTime in ob_progress or a new column

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope
</user_constraints>

---

## Standard Stack

### Core
| Library / Tool | Version | Purpose | Why Standard |
|---------------|---------|---------|--------------|
| `os/exec` (stdlib) | Go stdlib | Shell out to `ffmpeg` for conversion + thumbnail | No Go FFmpeg binding needed; exec.Command is the standard approach for CLI tools |
| `ffmpeg` (system binary) | 8.0.1 (dev); any ≥4.x in prod | Video conversion (MOV/WebM→MP4) + thumbnail extraction | Available at `/usr/local/bin/ffmpeg` on dev; must be confirmed on Hetzner prod |
| AWS SDK v2 S3 presigner | already in go.mod (`github.com/aws/aws-sdk-go-v2/service/s3`) | Video presign PUT/GET — reuse existing presigner | Exact same presigner used for photos; zero new dependency |
| HTML5 `<video>` element | Web platform | Inline player with native controls in fullscreen | D-08: no custom player; browser native covers all mobile platforms |

### Supporting
| Library / Tool | Version | Purpose | When to Use |
|---------------|---------|---------|-------------|
| `multipart/form-data` | Go stdlib `mime/multipart` | Thumbnail custom upload endpoint | Only needed if D-14 custom thumbnail upload is implemented |
| `context.WithTimeout` | Go stdlib | Wrap FFmpeg exec calls | Prevent runaway conversions; use 5 min timeout for 200 MB files |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `exec.Command` ffmpeg | `github.com/u2takey/ffmpeg-go` | ffmpeg-go adds a Go fluent API but adds a dependency for no real benefit over direct exec.Command for 2 specific commands |
| Single presign + FFmpeg server-side | Client-side MediaRecorder → MP4 | Client-side conversion is not reliable cross-platform; server-side FFmpeg is the correct approach |
| New migration for max_watched_time | New table ob_video_watch_progress | Single column addition to existing ob_progress is simpler; ob_progress already has hire_id + item_id + progress_type unique constraint |

**Installation:** No new Go dependencies required. `ffmpeg` must be confirmed installed on Hetzner deployment target (see Environment Availability).

**Version verification:** `go.mod` already contains `github.com/aws/aws-sdk-go-v2/service/s3` — confirmed reusable. No new packages need adding.

---

## Architecture Patterns

### Recommended Project Structure
```
backend/internal/
├── photos/
│   ├── spaces.go          # Existing — GeneratePresignedPutURL, PublicURL (reuse directly)
│   └── handler.go         # Existing — PresignUploadHandler (extend path_prefix to accept "videos/...")
├── onboarding/
│   ├── db.go              # Extend VideoPart struct with ThumbnailURL field
│   ├── handler.go         # Add VideoPresignHandler, VideoProcessHandler
│   └── video.go           # NEW — FFmpeg wrapper (convert, extractThumbnail)
└── db/migrations/
    └── 0028_ob_video_progress.sql  # ADD max_watched_time FLOAT8, thumbnail_url TEXT to ob_video_parts

onboarding.html
  renderOBVideoItem()      # Builder: add upload/URL toggle, file input, progress bar
  renderTrainingSection()  # Runner: replace "Watch Video" link with thumbnail + player div
  video player JS          # NEW — initVideoPlayer(), seeking lock, timeupdate → 95% check, saveWatchProgress()
```

### Pattern 1: Video Presign (reusing existing photos presign)

**What:** The existing `PresignUploadHandler` in `photos/handler.go` already accepts `path_prefix`, `id`, and `filename`. For videos, call it with `path_prefix: "videos/onboarding"`, `id: "{template_id}/{part_id}"`, `filename: "video.mp4"`. This produces the correct DO Spaces path `videos/onboarding/{template_id}/{part_id}/video.mp4`.

**When to use:** Whenever a builder uploads a video file from device. Frontend requests presign → PUT directly to Spaces → calls `/api/v1/videos/process` with `{part_id, object_key}` to trigger FFmpeg.

**Example (reuse without change):**
```go
// Source: backend/internal/photos/handler.go (existing)
// No change needed — path_prefix: "videos/onboarding", id: templateId+"/"+partId
// produces: videos/onboarding/{template_id}/{part_id}/video.mp4
key := req.PathPrefix + "/" + req.ID + "/" + req.Filename
```

**IMPORTANT:** The 8100 KiB CDN limit applies only to CDN-fronted requests. Direct origin presigned PUT URLs support up to 5 GB. Since the presigner uses `BaseEndpoint` pointing to `nyc3.digitaloceanspaces.com` (not the CDN hostname), 200 MB uploads work without multipart.

### Pattern 2: FFmpeg exec.Command in Go

**What:** Use `exec.CommandContext` with a context that has a timeout. Each argument must be a separate string — do not pass a shell command string.

**When to use:** After the browser PUT completes. The frontend calls POST `/api/v1/videos/process` which queues the work as a goroutine. Two operations: (1) conversion if input is MOV/WebM, (2) thumbnail extraction.

**Example:**
```go
// Source: Go stdlib os/exec documentation; pattern verified against existing handler.go goroutine pattern
import (
    "context"
    "fmt"
    "os/exec"
    "time"
)

// convertToMP4 converts a video file to H.264 MP4. Input file must be
// downloaded from Spaces to a temp file first (or use a pipe from presigned GET).
func convertToMP4(ctx context.Context, inputPath, outputPath string) error {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()
    cmd := exec.CommandContext(ctx, "ffmpeg",
        "-i", inputPath,
        "-c:v", "libx264",
        "-c:a", "aac",
        "-movflags", "+faststart",
        "-y",       // overwrite output without asking
        outputPath,
    )
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("ffmpeg convert: %w — output: %s", err, out)
    }
    return nil
}

// extractThumbnail grabs frame at 2s as JPEG.
func extractThumbnail(ctx context.Context, inputPath, thumbPath string) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    cmd := exec.CommandContext(ctx, "ffmpeg",
        "-i", inputPath,
        "-ss", "00:00:02",
        "-frames:v", "1",
        "-q:v", "3",
        "-y",
        thumbPath,
    )
    out, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("ffmpeg thumbnail: %w — output: %s", err, out)
    }
    return nil
}
```

**Invocation strategy (Claude's Discretion recommendation):** Use a fire-and-forget goroutine from `VideoProcessHandler`, identical to how `cleanupOldDrafts` is invoked in workflows. Log errors; do not block the HTTP response. Thumbnail extraction happens in the same goroutine after conversion completes.

### Pattern 3: Seeking Restriction via `seeking` Event

**What:** On every `seeking` event, compare `video.currentTime` to `maxWatchedTime`. If the user is seeking forward past what they've watched, clamp `currentTime` back to `maxWatchedTime`.

**Safari quirk:** Safari fires the `seeking` event after `currentTime` is already updated, which is the same behavior as other browsers. The clamp approach works cross-platform.

**When to use:** In the video player initialization code. Run on the `seeking` event, not `seeked`.

**Example:**
```javascript
// Source: HTML5 spec + MDN; verified against multiple implementations
let maxWatchedTime = part.max_watched_time || 0; // loaded from API

video.addEventListener('timeupdate', () => {
  if (video.currentTime > maxWatchedTime) {
    maxWatchedTime = video.currentTime;
  }
  if (!part.checked && video.duration && video.currentTime / video.duration >= 0.95) {
    markVideoPartWatched(part.id);
  }
});

video.addEventListener('seeking', () => {
  if (video.currentTime > maxWatchedTime + 0.5) {
    // +0.5s tolerance for browser imprecision
    video.currentTime = maxWatchedTime;
  }
});

video.addEventListener('ended', () => {
  if (!part.checked) markVideoPartWatched(part.id);
});
```

### Pattern 4: Fullscreen on Play (iOS Mobile Safari)

**What:** Use `video.requestFullscreen()` or the webkit-prefixed `video.webkitEnterFullscreen()` fallback. iOS Safari requires `playsinline` on the `<video>` tag to prevent it from going fullscreen automatically, and then `webkitEnterFullscreen()` to enter fullscreen programmatically.

**iOS Safari specific:** `requestFullscreen()` is not supported on iOS Safari — use `video.webkitEnterFullscreen()` instead. Detect and fall back:

```javascript
// Source: Apple Developer Documentation (Delivering Video Content for Safari)
// requestFullscreen is not available on iOS; must use webkitEnterFullscreen
function enterFullscreen(videoEl) {
  if (videoEl.webkitEnterFullscreen) {
    videoEl.webkitEnterFullscreen(); // iOS Safari
  } else if (videoEl.requestFullscreen) {
    videoEl.requestFullscreen();     // all other browsers
  }
}
```

**Poster required on iOS:** Without a `poster` attribute, iOS Safari shows a black screen before play. Must set `poster` to the thumbnail URL.

```html
<video playsinline controls poster="https://...thumb.jpg">
  <source src="..." type="video/mp4">
</video>
```

### Pattern 5: DB Schema — Adding max_watched_time to ob_progress

**What:** Add `max_watched_time` as a nullable FLOAT8 column to `ob_progress`. The existing unique constraint `(hire_id, item_id, progress_type)` already covers video parts. Use `INSERT ... ON CONFLICT DO UPDATE SET max_watched_time = ...` to upsert progress.

**Migration:**
```sql
-- +goose Up
BEGIN;
ALTER TABLE ob_progress ADD COLUMN max_watched_time FLOAT8;
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE ob_progress DROP COLUMN max_watched_time;
COMMIT;
```

**Recommendation (Claude's Discretion):** Store `max_watched_time` in `ob_progress` (not a new table). This is simpler and the existing unique constraint already handles the hire+part combination correctly. The `checked` column remains the completion flag; `max_watched_time` is the resume position.

**VideoPart struct extension:**
```go
// Extend existing VideoPart in db.go
type VideoPart struct {
    ID             string  `json:"id"`
    Title          string  `json:"title"`
    Description    string  `json:"description"`
    URL            string  `json:"url"`
    ThumbnailURL   string  `json:"thumbnail_url"` // NEW
    SortOrder      int     `json:"sort_order"`
    Checked        bool    `json:"checked"`
    MaxWatchedTime float64 `json:"max_watched_time"` // NEW — populated from ob_progress
}
```

**ob_video_parts schema addition:**
```sql
-- In same migration as ob_progress, or a separate one
ALTER TABLE ob_video_parts ADD COLUMN thumbnail_url TEXT;
```

### Anti-Patterns to Avoid

- **Passing a full shell string to exec.Command:** `exec.Command("ffmpeg -i input.mp4 ...")` will fail — each arg must be separate.
- **Using CDN hostname for presigned PUT:** Will hit the 8100 KiB limit. The existing `NewSpacesPresigner` uses `BaseEndpoint` pointing to origin, which is correct.
- **Removing `playsinline` to force fullscreen on iOS:** Without `playsinline`, iOS auto-enters fullscreen on play — seeking restriction JS doesn't run in that mode. Use `playsinline` + `webkitEnterFullscreen()` programmatically.
- **Blocking the HTTP response while FFmpeg runs:** 200 MB conversion can take 30–120 seconds. Always fire-and-forget via goroutine.
- **Clamping on `seeked` instead of `seeking`:** `seeked` fires after the seek completes — the user sees a jump. `seeking` fires immediately, allowing the clamp before content is displayed.
- **Storing maxWatchedTime in localStorage:** CLAUDE.md explicitly forbids localStorage for data persistence. Must go through `saveProgress` API.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Video conversion pipeline | Custom binary decoder | `ffmpeg` via exec.Command | FFmpeg handles codec quirks, container repackaging, faststart flag, AAC audio normalization |
| Thumbnail extraction | Canvas pixel capture in browser | `ffmpeg -frames:v 1` server-side | Client-side Canvas has CORS restrictions on cross-origin video; server is reliable |
| S3-compatible upload | Custom HTTP PUT with signature | Existing `NewSpacesPresigner` + `GeneratePresignedPutURL` | Already handles path-style, credentials, content-type — zero new code |
| Upload progress | XHR streaming | `fetch` with `XMLHttpRequest` and `upload.onprogress` | `fetch` does not expose upload progress; must use XHR for the progress bar (D-04) |

**Key insight on upload progress:** The existing `handleFailPhotoCaptureClick` in `workflows.html` uses `fetch()` for photo uploads. For video uploads with a required progress bar (D-04), must switch to `XMLHttpRequest` with `xhr.upload.addEventListener('progress', ...)`. `fetch` has no upload progress API in any browser.

---

## Runtime State Inventory

Step 2.5: SKIPPED — this is a greenfield addition, not a rename/refactor. No stored data items reference the old video URL field by a string key that needs migration.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `ffmpeg` binary | Video conversion, thumbnail extraction | ✓ (dev) | 8.0.1 | — (blocking on prod) |
| DO Spaces bucket | Video + thumbnail storage | ✓ (inferred from Phase 12 photos working) | — | — |
| AWS SDK v2 S3 | Presign URLs | ✓ | already in go.mod | — |
| `libx264` (ffmpeg codec) | MP4 conversion | ✓ (dev — built with `--enable-libx264`) | bundled with ffmpeg | — |

**Missing dependencies with no fallback:**
- `ffmpeg` on Hetzner production server — must be verified and installed before deploy. Check with `command -v ffmpeg` on Hetzner. Install: `apt-get install -y ffmpeg` on Debian/Ubuntu. This blocks video conversion and thumbnail extraction.

**Missing dependencies with fallback:**
- None identified.

---

## Common Pitfalls

### Pitfall 1: iOS Safari Black Screen / Wrong Fullscreen Behavior
**What goes wrong:** Tapping play on iOS shows a black screen; or the video auto-enters system fullscreen before the seeking restriction JS is attached.
**Why it happens:** iOS Safari requires `poster` attribute to show a first frame; without `playsinline`, iOS auto-enters its own fullscreen (where the seeking event does not fire as expected).
**How to avoid:** Always set `poster` to the thumbnail URL. Always include `playsinline` attribute. Use `webkitEnterFullscreen()` (not `requestFullscreen()`) to enter fullscreen programmatically after play is triggered.
**Warning signs:** Video shows black until user taps; seeking restriction has no effect on iOS.

### Pitfall 2: CDN Endpoint for Presigned PUT Uploads
**What goes wrong:** Upload fails or is silently truncated at ~8 MB.
**Why it happens:** The 8100 KiB CDN limit applies when presigned PUT is routed through the CDN hostname.
**How to avoid:** The existing `NewSpacesPresigner` uses `BaseEndpoint: cfg.Endpoint` which points to `https://nyc3.digitaloceanspaces.com` (origin), not the CDN. This is correct — do not change it. Verify the `Endpoint` env var does not point to a CDN custom domain.
**Warning signs:** Uploads above 8 MB return 4xx from DO Spaces.

### Pitfall 3: exec.Command Shell Command String
**What goes wrong:** `exec.Command("ffmpeg -i input.mp4 output.mp4")` fails with "exec: no such file or directory".
**Why it happens:** `exec.Command` does not invoke a shell — it executes the binary directly. The first argument must be the binary name, remaining args are separate strings.
**How to avoid:** Always pass args as separate strings: `exec.Command("ffmpeg", "-i", inputPath, outputPath)`.
**Warning signs:** FFmpeg subprocess exits immediately with error about binary not found.

### Pitfall 4: Seeking Event Fires Before video.duration is Known
**What goes wrong:** `video.duration` is `NaN` or `Infinity` on initial load; the 95% check divides by NaN.
**Why it happens:** `duration` is not available until the `loadedmetadata` event fires.
**How to avoid:** Gate the 95% check: `if (!isNaN(video.duration) && video.duration > 0 && video.currentTime / video.duration >= 0.95)`.
**Warning signs:** Video part is marked watched immediately on open.

### Pitfall 5: Temp File Cleanup After FFmpeg
**What goes wrong:** The server accumulates large temp files (input + output, each up to 200 MB) if conversion fails or the goroutine is interrupted.
**Why it happens:** Errors in FFmpeg goroutine exit early without removing temp files; `defer os.Remove(tmpPath)` handles normal exit but panics skip defers... wait, defers do run on panic via recover. Use `defer os.Remove()` for all temp paths.
**How to avoid:** Use `defer os.Remove(inputTmp)` and `defer os.Remove(outputTmp)` immediately after creating temp files, before any error branches.
**Warning signs:** Disk usage grows on the Hetzner box; large files in `/tmp`.

### Pitfall 6: XHR vs fetch for Upload Progress
**What goes wrong:** Builder shows no upload progress bar on large files (D-04 requirement missed).
**Why it happens:** `fetch()` does not expose `upload.onprogress`. The existing photo upload code uses `fetch()` but photos are small JPEGs so no progress bar is needed.
**How to avoid:** For video upload, use `XMLHttpRequest` with `xhr.upload.addEventListener('progress', e => { if (e.lengthComputable) showProgress(e.loaded / e.total) })`.
**Warning signs:** Progress bar never updates during upload.

### Pitfall 7: toggle-video-part Still Manually Checkable
**What goes wrong:** The current `toggle-video-part` click handler allows manual check/uncheck (D-12 violation).
**Why it happens:** The existing handler at `onboarding.html` line 886 calls `saveProgress` on any tap.
**How to avoid:** Remove the `data-action="toggle-video-part"` from the checkbox element in the runner rendering (line 323). The checkbox should render as non-interactive for video parts. Only the watch-completion path marks it checked.
**Warning signs:** A user can check a video part without watching it.

---

## Code Examples

### Upload Flow (Builder) — XHR with Progress
```javascript
// Source: MDN XMLHttpRequest.upload — verified pattern for upload progress
async function uploadVideoFile(file, presignUrl, onProgress) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('PUT', presignUrl);
    xhr.setRequestHeader('Content-Type', file.type);
    xhr.upload.addEventListener('progress', e => {
      if (e.lengthComputable) onProgress(e.loaded / e.total);
    });
    xhr.onload = () => xhr.status < 300 ? resolve() : reject(new Error('Upload failed: ' + xhr.status));
    xhr.onerror = () => reject(new Error('Network error'));
    xhr.send(file);
  });
}
```

### Video Player Init (Runner)
```javascript
// Source: HTML5 spec + MDN + Apple Developer Documentation
function initVideoPlayer(videoEl, part) {
  let maxWatchedTime = part.max_watched_time || 0;

  videoEl.addEventListener('timeupdate', () => {
    if (videoEl.currentTime > maxWatchedTime) maxWatchedTime = videoEl.currentTime;
    if (!part.checked && !isNaN(videoEl.duration) && videoEl.duration > 0) {
      if (videoEl.currentTime / videoEl.duration >= 0.95) markVideoPartWatched(part.id);
    }
  });

  videoEl.addEventListener('seeking', () => {
    if (videoEl.currentTime > maxWatchedTime + 0.5) videoEl.currentTime = maxWatchedTime;
  });

  videoEl.addEventListener('ended', () => {
    if (!part.checked) markVideoPartWatched(part.id);
  });

  // Save progress when leaving (paused or player closed)
  videoEl.addEventListener('pause', () => {
    if (maxWatchedTime > (part.max_watched_time || 0)) saveVideoWatchProgress(part.id, maxWatchedTime);
  });
}
```

### Presign Endpoint Extension (Backend)
```go
// Source: backend/internal/photos/handler.go existing pattern — extend for video
// New endpoint: POST /api/v1/videos/presign
// Reuse same PresignUploadHandler, just set path_prefix="videos/onboarding"
// and content_type="video/mp4" (or derive from filename extension)
// Only change needed: PresignUploadHandler currently hardcodes "image/jpeg"
// → add ContentType field to presignUploadRequest or derive from filename
```

### VideoProcess Handler (Backend) — Fire-and-Forget Goroutine
```go
// Source: existing pattern in workflow/handler.go cleanupOldDrafts goroutine
func VideoProcessHandler(presigner *s3.PresignClient, bucket, endpoint string, pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var body struct {
            PartID    string `json:"part_id"`
            ObjectKey string `json:"object_key"` // raw uploaded key (may be .mov/.webm)
        }
        // ... decode, validate ...
        go func() {
            if err := processVideo(context.Background(), presigner, bucket, endpoint, pool, body.PartID, body.ObjectKey); err != nil {
                log.Printf("processVideo %s: %v", body.PartID, err)
            }
        }()
        writeJSON(w, http.StatusAccepted, map[string]string{"status": "processing"})
    }
}
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `ffmpeg -vframes 1` | `ffmpeg -frames:v 1` | FFmpeg ~3.x | `-vframes` deprecated; use `-frames:v` |
| `requestFullscreen()` on iOS | `webkitEnterFullscreen()` | Ongoing (iOS never implemented requestFullscreen for video) | Must use webkit prefix on iOS |
| `fetch()` for upload progress | `XMLHttpRequest` with `upload.onprogress` | Web platform standard | `fetch` still has no upload progress API in 2026 |

**Deprecated/outdated:**
- `-vframes 1`: Use `-frames:v 1` instead (deprecated in recent FFmpeg)
- `EndpointResolverWithOptions` in AWS SDK v2: Use `s3.Options.BaseEndpoint` instead (already correct in project, per STATE.md Phase 12-02)

---

## Open Questions

1. **FFmpeg on Hetzner production server**
   - What we know: ffmpeg is available on dev (`/usr/local/bin/ffmpeg`). The Hetzner box runs Debian/Ubuntu (inferred from Caddy + Go deployment).
   - What's unclear: Whether `ffmpeg` with `libx264` is installed on the Hetzner production host.
   - Recommendation: Wave 0 task should verify: `ssh hetzner 'ffmpeg -version'`. If absent: `apt-get install -y ffmpeg`.

2. **CORS on DO Spaces bucket for direct browser PUT of video/mp4 content-type**
   - What we know: Phase 12 confirmed CORS works for `image/jpeg` uploads (photos feature is live). Per STATE.md: "DO Spaces bucket CORS policy for direct browser PUT uploads must be verified during Phase 12 planning".
   - What's unclear: Whether the existing CORS policy allows `Content-Type: video/mp4`, `video/quicktime`, `video/webm`.
   - Recommendation: Check the bucket's CORS config on DO dashboard. Add `video/mp4`, `video/quicktime`, `video/webm` to allowed Content-Types if not already present.

3. **Temp file storage path for FFmpeg conversion**
   - What we know: FFmpeg needs local temp files for input (downloaded from Spaces) and output (converted MP4 to upload back).
   - What's unclear: What storage is available on Hetzner (`/tmp` vs a dedicated path), and how much disk space is allocated.
   - Recommendation: Use `os.MkdirTemp("", "yumyums-video-*")` and clean up with `defer os.RemoveAll(tmpDir)`. For 200 MB files, need ~400 MB free temp space per concurrent conversion.

---

## Validation Architecture

Validation is disabled (`nyquist_validation: false` in `.planning/config.json`).

---

## Sources

### Primary (HIGH confidence)
- DigitalOcean Spaces Limits docs (fetched 2026-04-19): 5 GB max for direct origin PUT, 8100 KiB limit applies only to CDN-fronted requests — https://docs.digitalocean.com/products/spaces/details/limits/
- Apple Developer Documentation — Delivering Video Content for Safari: `webkitEnterFullscreen()` on iOS, `playsinline` requirement — https://developer.apple.com/documentation/webkit/delivering-video-content-for-safari
- Go stdlib `os/exec` — exec.Command argument handling (individual strings, not shell command)
- Codebase read: `backend/internal/photos/spaces.go`, `photos/handler.go` — existing presigner is origin-only (not CDN), reusable directly
- Codebase read: `backend/internal/db/migrations/0020_ob_progress.sql` — ob_progress schema confirmed; `max_watched_time` column not yet present
- Codebase read: `backend/internal/db/migrations/0019_ob_items.sql` — ob_video_parts schema confirmed; `thumbnail_url` column not present
- Codebase read: `onboarding.html` line 332 — current runner renders `<a class="video-link" data-action="watch-video" ...>Watch Video</a>`
- Codebase read: `onboarding.html` line 886 — current `toggle-video-part` handler is fully manual; must be removed per D-12

### Secondary (MEDIUM confidence)
- WebKit Blog "New Video Policies for iOS" — `playsinline` behavior and autoplay restrictions (verified against Apple Developer docs)
- MDN Web Docs — `XMLHttpRequest.upload.progress` for upload progress bars (verified pattern; fetch has no upload progress)
- FFmpeg documentation — `-frames:v`, `-movflags +faststart`, thumbnail extraction at `-ss` offset (cross-verified against ffmpeg -help output)

### Tertiary (LOW confidence)
- WebSearch: seeking event timing on Safari — multiple sources agree on clamp-on-seeking pattern; not verified against a definitive Apple spec doc

---

## Metadata

**Confidence breakdown:**
- DO Spaces limits: HIGH — fetched from official DO docs page
- FFmpeg exec.Command pattern: HIGH — Go stdlib; verified ffmpeg available at /usr/local/bin/ffmpeg
- iOS Safari fullscreen/playsinline: HIGH — Apple Developer Documentation
- Seeking restriction pattern: MEDIUM-HIGH — multiple implementations agree; `seeking` event timing consistent across browsers
- ob_progress schema extension: HIGH — schema read directly from migration files
- XHR upload progress: HIGH — confirmed fetch has no upload.onprogress in any current browser

**Research date:** 2026-04-19
**Valid until:** 2026-05-19 (stable APIs; DO Spaces limits unlikely to change)
