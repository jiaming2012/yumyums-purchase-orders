---
phase: 12-inventory-photos-tile-permissions
plan: 02
subsystem: api
tags: [do-spaces, s3, presigned-urls, photo-upload, workflows, go, aws-sdk-v2]

requires:
  - phase: 10-workflows-api
    provides: "workflows.html photo-modal UI, autoSaveField/debouncedSaveField persistence pipeline"
  - phase: 09-foundation-auth
    provides: "auth middleware for protected /api/v1/* endpoints"
provides:
  - "backend/internal/photos package: SpacesConfig, NewSpacesPresigner, GeneratePresignedPutURL, GeneratePresignedGetURL, PublicURL"
  - "POST /api/v1/photos/presign — returns presigned PUT URL + object key + public URL"
  - "GET /api/v1/photos/presign?key=... — returns short-lived GET URL for private objects"
  - "workflows.html photo capture: openCamera → showPhotoPreview → presign → PUT to DO Spaces → debouncedSaveField(publicUrl)"
  - "Approvals tab photo thumbnails for submitted https:// field values"
affects:
  - 12-inventory-photos-tile-permissions
  - workflows-photo-evidence

tech-stack:
  added:
    - "github.com/aws/aws-sdk-go-v2/service/s3 v1.99.1 — S3-compatible presigned URL generation"
    - "github.com/aws/aws-sdk-go-v2/config v1.32.16 — AWS SDK config"
    - "github.com/aws/aws-sdk-go-v2/credentials v1.19.15 — static credentials provider"
  patterns:
    - "DO Spaces uses BaseEndpoint + UsePathStyle:true (not legacy EndpointResolverWithOptions)"
    - "Graceful degradation: missing env vars → 503 on photo endpoints, server starts cleanly"
    - "Two-step presign-then-upload: POST /presign gets URL, browser PUTs blob directly to Spaces"
    - "Never autoSaveField blob: URLs — only save permanent https:// public_url from Spaces"
    - "Photo capture area uses .photo-capture-area div for in-place DOM mutation (uploading → thumb/error)"

key-files:
  created:
    - "backend/internal/photos/spaces.go"
    - "backend/internal/photos/handler.go"
  modified:
    - "backend/cmd/server/main.go"
    - "workflows.html"
    - "sw.js"

key-decisions:
  - "Used s3.Options.BaseEndpoint instead of deprecated EndpointResolverWithOptions for DO Spaces custom endpoint (SDK v2 v1.99.1)"
  - "Photos presign route is inside authenticated group — no unauthenticated access to generate upload URLs"
  - "publicUrl stored as field response value (not presigned GET URL) — photo is public in Spaces; presigned GET is only needed for private buckets"
  - "api() function prepends /api/v1/workflow/ — used fetch() directly for /api/v1/photos/presign to avoid the workflow prefix"
  - "failPhotos (corrective action evidence) remain blob: URLs in local state — only checklist photo fields use Spaces upload"
  - "blob: URL check in REJECTION_FLAGS updated to https:// — photos now stored as Spaces URLs not blob: URLs"

requirements-completed: [PHOT-01, PHOT-02]

duration: 5min
completed: 2026-04-18
---

# Phase 12 Plan 02: Photos Presigned Upload Summary

**DO Spaces presigned PUT upload for checklist photo fields — Go backend generates URLs, browser uploads blobs directly, persistent https:// URLs stored in field responses**

## Performance

- **Duration:** 5 min
- **Started:** 2026-04-18T11:40:05Z
- **Completed:** 2026-04-18T11:45:31Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Created `backend/internal/photos` package with DO Spaces presigner (UsePathStyle:true), presigned PUT/GET URL generation, and public URL helper
- Registered `POST /api/v1/photos/presign` and `GET /api/v1/photos/presign` routes behind auth middleware with graceful 503 degradation when env vars missing
- Replaced `handlePhotoCaptureClick` stub in workflows.html with full presign → PUT blob → debouncedSaveField(publicUrl) flow
- Added upload state UI (spinner/Uploading... text), success state (thumbnail + Retake), and error state (Upload failed. Tap to retry.)
- Wired Approvals tab to show 72x72 photo thumbnails for submitted fields with https:// values

## Task Commits

1. **Task 1: Create photos Go package with presigned URL handlers** - `7c9f6de` (feat)
2. **Task 2: Wire workflows.html photo capture to presigned upload** - `76bca70` (feat)

**Plan metadata:** (see final commit)

## Files Created/Modified

- `backend/internal/photos/spaces.go` — SpacesConfig struct, NewSpacesPresigner (BaseEndpoint + UsePathStyle:true), GeneratePresignedPutURL (15min TTL), GeneratePresignedGetURL (1hr TTL), PublicURL
- `backend/internal/photos/handler.go` — PresignUploadHandler (POST, builds key from path_prefix/id/filename), PresignGetHandler (GET ?key=), both return 503 if presigner nil
- `backend/cmd/server/main.go` — imports photos + s3 packages, reads DO_SPACES_* env vars, initializes presigner, registers /api/v1/photos/presign routes
- `workflows.html` — full handlePhotoCaptureClick implementation, photo field renderer with capture-area, Approvals photo thumbnails, blob:→https:// flag checks
- `sw.js` — rebuilt with updated workflows.html content hash

## Decisions Made

- `s3.Options.BaseEndpoint` used instead of deprecated `EndpointResolverWithOptions` — SDK v2 v1.99.1 removed the old field from s3.Options struct literal
- `fetch()` used directly for `/api/v1/photos/presign` instead of `api()` helper — `api()` prepends `/api/v1/workflow/` and photos is a separate route group
- Photos stored as public permanent URLs (not presigned GET URLs) — bucket CORS + public access assumed for checklist evidence photos
- Fail card photos (corrective action evidence) NOT wired to Spaces in this plan — they remain blob: URLs (different UX, can be addressed in a follow-up)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed deprecated EndpointResolverWithOptions field removed from s3.Options**
- **Found during:** Task 1 (spaces.go compilation)
- **Issue:** Plan specified using `EndpointResolverWithOptions` but AWS SDK v2 v1.99.1 removed this field from `s3.Options` struct
- **Fix:** Replaced with `BaseEndpoint: aws.String(cfg.Endpoint)` which is the current v2 API for custom endpoints
- **Files modified:** backend/internal/photos/spaces.go
- **Verification:** `go build ./cmd/server/...` succeeds
- **Committed in:** 7c9f6de (Task 1 commit)

**2. [Rule 1 - Bug] Used fetch() directly instead of api() for presign endpoint**
- **Found during:** Task 2 (wiring workflows.html)
- **Issue:** `api()` in sync.js prepends `/api/v1/workflow/` — calling `api('POST', '../photos/presign', ...)` would produce a path-traversal URL that works but is fragile
- **Fix:** Used `fetch('/api/v1/photos/presign', {...})` directly with explicit auth headers
- **Files modified:** workflows.html
- **Verification:** Correct absolute URL used
- **Committed in:** 76bca70 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 API change, 1 code correctness)
**Impact on plan:** Both fixes required for correctness. No scope creep.

## Issues Encountered

None - plan executed successfully after the two auto-fixes above.

## Known Stubs

- **Fail card photos (corrective action evidence)** — `handleFailPhotoCaptureClick` still uses blob: URLs stored in `FAIL_NOTES[fldId].photo`. These are not persisted to Spaces. This is intentional for Phase 12 (plan focused on photo fields, not fail card evidence); fail card photos can be wired to Spaces in a follow-up plan if needed.

## User Setup Required

**DO Spaces environment variables required for photo upload to work:**

```
DO_SPACES_KEY=<your-spaces-access-key>
DO_SPACES_SECRET=<your-spaces-secret-key>
DO_SPACES_BUCKET=<your-bucket-name>
DO_SPACES_REGION=<region, e.g. nyc3>
DO_SPACES_ENDPOINT=<optional, defaults to https://{region}.digitaloceanspaces.com>
```

Without these variables, the server starts cleanly and photo endpoints return HTTP 503 with `{"error":"photo storage not configured"}`.

**CORS policy required on the DO Spaces bucket** to allow browser PUT uploads:
```json
[{
  "AllowedOrigins": ["https://your-domain.com"],
  "AllowedMethods": ["PUT"],
  "AllowedHeaders": ["Content-Type"],
  "MaxAgeSeconds": 3000
}]
```

## Next Phase Readiness

- Photos backend fully wired — missing only DO Spaces bucket CORS config and env vars
- Checklist photo evidence is now persistent across sessions (https:// URLs survive reload)
- Manager Approvals tab displays photo thumbnails inline for submitted photo fields
- Fail card photos (corrective action evidence) remain local-only (blob: URLs) — can be wired to Spaces in a future plan if needed

---
*Phase: 12-inventory-photos-tile-permissions*
*Completed: 2026-04-18*
