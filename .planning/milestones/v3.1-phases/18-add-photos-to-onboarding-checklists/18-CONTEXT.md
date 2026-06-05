# Phase 18: Add Photos to Onboarding Checklists - Context

**Gathered:** 2026-05-20
**Status:** Ready for planning
**Source:** Codebase exploration in conversation

<domain>
## Phase Boundary

Add a `photo` item type to the onboarding checklist system. Crew members can capture/upload photos as part of their training checklists. Managers can view uploaded photos. Builders can add photo steps to templates.

This mirrors the existing `photo` field type in `workflows.html` but adapted for the onboarding system's item-based progress model.

</domain>

<decisions>
## Implementation Decisions

### Item Type
- New item type: `photo` (alongside existing `checkbox`, `video_series`, `faq`)
- Photo items count as 1 toward section progress (like checkbox)
- A photo item is "checked" when a photo has been uploaded

### Builder (Tab 3)
- Add `+ Photo` button alongside existing `+ Checkbox` and `+ Video Series` buttons
- Photo item in builder editor shows: drag handle, label input, delete button
- No additional config needed (no "require photo" toggle — it's always required if the item exists)
- `createNewOBPhotoItem(label)` returns `{ id: generateOBId('itm'), type: 'photo', label: label || '', sub_items: [] }`

### My Trainings (Tab 1) — Crew Fill-Out
- Photo item renders: label + photo capture button (📷 Take Photo)
- Tapping capture opens device camera via `<input type="file" accept="image/*" capture="environment">`
- After capture: show fullscreen preview modal with confirm/retake buttons
- On confirm: upload to DO Spaces via presigned URL, save progress with photo URL
- After upload: show thumbnail (72x72) with "Retake" link
- Photo URL stored in `ob_progress.value` column (new column)

### Manager View (Tab 2) — Read-Only
- Show uploaded photo as thumbnail (72x72, clickable to open full-size in new tab)
- If not uploaded: show "No photo" placeholder text

### Backend Changes
- Add `value TEXT` column to `ob_progress` table (nullable) — stores photo URL
- `SaveProgress` handler accepts optional `value` field in request body
- `SaveProgress` DB function stores value alongside progress entry
- `Item` struct gets `PhotoURL string` field, populated from `ob_progress.value` when returning hire training
- Update `sectionIncompleteItem` SQL to include `'photo'` in the type list
- All other SQL queries that filter on `oi.type IN ('checkbox', 'video_series', 'faq')` must add `'photo'`

### Photo Upload Flow
- Reuse existing `/api/v1/photos/presign` endpoint
- Path convention: `onboarding/{hireId}/{itemId}.jpg`
- Progress saved via existing `POST /api/v1/onboarding/saveProgress` with new `value` field
- `progress_type` for photos: `"photo"`

### CSS
- Reuse photo styles from `workflows.html`: `.photo-capture-btn`, `.photo-thumb`, `.photo-thumb-wrap`, `.photo-retake-link`, `.photo-modal`, `.photo-preview`, `.photo-confirm-btn`, `.photo-retake-btn`, `.photo-uploading`, `.photo-spinner`

### Claude's Discretion
- Exact CSS values (copy from workflows.html)
- Error handling UI for failed uploads
- Whether to add photo-specific E2E tests (recommended but not blocking)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Onboarding Frontend
- `onboarding.html` — Full onboarding app (Builder, My Trainings, Manager tabs)

### Onboarding Backend
- `backend/internal/onboarding/handler.go` — HTTP handlers for all onboarding endpoints
- `backend/internal/onboarding/db.go` — DB queries, types, and progress tracking

### Photo Infrastructure (existing)
- `backend/internal/photos/handler.go` — Presigned upload handler (`/api/v1/photos/presign`)
- `backend/internal/photos/spaces.go` — DO Spaces client and URL generation

### Photo UI Reference (existing in workflows)
- `workflows.html` — Lines ~430-580: `openCamera()`, `showPhotoPreview()`, photo upload flow

</canonical_refs>

<specifics>
## Specific Ideas

- The `openCamera()` and `showPhotoPreview()` functions in workflows.html can be copied into onboarding.html since there's no shared JS module system
- The `ob_progress` table uses `(hire_id, item_id, progress_type)` as unique key — adding `value` column is a simple ALTER TABLE
- Photo items should follow the same section-completion and unlock-next-section logic as checkboxes
- The `saveProgress` function in the frontend already handles the API call — just needs to pass `value` for photo URLs

</specifics>

<deferred>
## Deferred Ideas

- Multiple photos per item (only single photo for now)
- Photo annotations/comments
- Photo compression before upload
- Photo items in workflows.html onboarding cross-linking

</deferred>

---

*Phase: 18-add-photos-to-onboarding-checklists*
*Context gathered: 2026-05-20 via conversation research*
