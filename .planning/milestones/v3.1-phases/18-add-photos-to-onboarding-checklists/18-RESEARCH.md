# Phase 18: Add Photos to Onboarding Checklists - Research

**Researched:** 2026-05-20
**Domain:** Onboarding photo capture/upload (frontend + backend + DB migration)
**Confidence:** HIGH

## Summary

This phase adds a `photo` item type to the onboarding checklist system. The implementation mirrors the existing photo capture flow in `workflows.html` but adapts it to onboarding's item-based progress model. All three onboarding tabs (My Trainings, Manager, Builder) need updates, plus a backend migration and handler changes.

The codebase already has all the infrastructure: DO Spaces presigned URL generation (`/api/v1/photos/presign`), photo capture/preview functions (`openCamera`, `showPhotoPreview`), and photo CSS classes in `workflows.html`. The main work is: (1) a DB migration to add `value TEXT` to `ob_progress` and update the `progress_type` CHECK constraint, (2) propagating the `value` field through the save/read Go code, (3) adding `'photo'` to all SQL type filter lists, and (4) copying the photo UI into `onboarding.html`.

**Primary recommendation:** Implement in three waves: DB migration + backend changes first, then frontend My Trainings + Manager rendering, then Builder support. This order lets each wave be independently testable.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- New item type: `photo` (alongside existing `checkbox`, `video_series`, `faq`)
- Photo items count as 1 toward section progress (like checkbox)
- A photo item is "checked" when a photo has been uploaded
- Builder: Add `+ Photo` button alongside existing `+ Checkbox` and `+ Video Series` buttons
- Photo item in builder editor shows: drag handle, label input, delete button
- `createNewOBPhotoItem(label)` returns `{ id: generateOBId('itm'), type: 'photo', label: label || '', sub_items: [] }`
- My Trainings: Photo item renders label + photo capture button, tapping opens device camera
- After capture: fullscreen preview modal with confirm/retake buttons
- On confirm: upload to DO Spaces via presigned URL, save progress with photo URL
- After upload: show thumbnail (72x72) with "Retake" link
- Photo URL stored in `ob_progress.value` column (new column)
- Manager View: Show uploaded photo as thumbnail (72x72, clickable to open full-size)
- If not uploaded: show "No photo" placeholder text
- Add `value TEXT` column to `ob_progress` table (nullable)
- `SaveProgress` handler accepts optional `value` field in request body
- `Item` struct gets `PhotoURL string` field, populated from `ob_progress.value`
- Update `sectionIncompleteItem` SQL to include `'photo'` in the type list
- All SQL queries filtering on `oi.type IN (...)` must add `'photo'`
- Reuse existing `/api/v1/photos/presign` endpoint
- Path convention: `onboarding/{hireId}/{itemId}.jpg`
- Progress saved via existing `POST /api/v1/onboarding/saveProgress` with new `value` field
- `progress_type` for photos: `"photo"`
- Reuse photo CSS classes from `workflows.html`

### Claude's Discretion
- Exact CSS values (copy from workflows.html)
- Error handling UI for failed uploads
- Whether to add photo-specific E2E tests (recommended but not blocking)

### Deferred Ideas (OUT OF SCOPE)
- Multiple photos per item (only single photo for now)
- Photo annotations/comments
- Photo compression before upload
- Photo items in workflows.html onboarding cross-linking
</user_constraints>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Photo capture (camera) | Browser / Client | -- | Device camera access is browser-only via `<input type="file" capture>` |
| Photo upload | Browser / Client | CDN / Static (DO Spaces) | Browser PUTs directly to DO Spaces via presigned URL |
| Presigned URL generation | API / Backend | -- | Server generates time-limited S3 presigned URLs |
| Photo URL storage | Database / Storage | -- | `ob_progress.value` column stores the public URL |
| Progress tracking | API / Backend | Database / Storage | `SaveProgress` handler writes to `ob_progress` |
| Photo rendering (thumb) | Browser / Client | -- | Frontend renders thumbnails from stored URL |
| Section completion logic | API / Backend | Browser / Client | Both compute section completeness; server is authoritative |

## Standard Stack

No new dependencies needed. All required infrastructure exists.

### Core (already in project)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| DO Spaces (S3-compatible) | -- | Object storage for photos | Already used for workflow photos and video uploads [VERIFIED: backend/internal/photos/spaces.go] |
| aws-sdk-go-v2/service/s3 | v1.x | Presigned URL generation | Already imported in photos package [VERIFIED: backend/internal/photos/spaces.go] |
| goose | -- | DB migrations | Project standard for schema changes [VERIFIED: migration files in backend/internal/db/migrations/] |

### Alternatives Considered
None -- all decisions are locked. This phase reuses existing infrastructure end-to-end.

## Architecture Patterns

### System Architecture Diagram

```
[Mobile Browser]
    |
    |-- tap "Take Photo" --> <input type="file" capture="environment">
    |-- camera returns file --> showPhotoPreview() modal
    |-- confirm --> POST /api/v1/photos/presign
    |                   |
    |                   v
    |              [Go Backend]
    |              generates presigned PUT URL
    |                   |
    |                   v
    |-- PUT file --> [DO Spaces] --> public URL returned
    |
    |-- POST /api/v1/onboarding/saveProgress
    |   { item_id, progress_type: "photo", checked: true, value: publicUrl }
    |                   |
    |                   v
    |              [Go Backend]
    |              SaveProgress() writes to ob_progress
    |                   |
    |                   v
    |              [PostgreSQL]
    |              ob_progress row: hire_id, item_id, progress_type="photo",
    |                               value=publicUrl
    |
    |-- GET /api/v1/onboarding/hireTraining/{id}?templateId=...
    |                   |
    |                   v
    |              [Go Backend]
    |              GetHireTraining() reads ob_progress.value
    |              populates Item.PhotoURL
    |                   |
    |                   v
    |-- render thumbnail from Item.photo_url
```

### Component Responsibilities

| Component | File | Changes Required |
|-----------|------|-----------------|
| DB Migration | `backend/internal/db/migrations/0056_ob_progress_photo.sql` | Add `value TEXT`, update CHECK constraint |
| Item struct | `backend/internal/onboarding/db.go` | Add `PhotoURL string` field |
| ProgressEntry struct | `backend/internal/onboarding/db.go` | Add `Value string` field |
| SaveProgress func | `backend/internal/onboarding/db.go` | Accept + store `value` parameter |
| SaveProgressHandler | `backend/internal/onboarding/handler.go` | Parse `value` from request body |
| GetHireTraining | `backend/internal/onboarding/db.go` | Read `op.value`, populate `Item.PhotoURL` |
| sectionIncompleteItem | `backend/internal/onboarding/db.go` | Add `'photo'` to type IN list |
| 6 SQL queries in db.go | `backend/internal/onboarding/db.go` | Add `'photo'` to all `oi.type IN (...)` lists |
| countSectionItems | `onboarding.html` | Add `photo` type handling |
| countCheckedItems | `onboarding.html` | Add `photo` type handling |
| renderSections | `onboarding.html` | Add `photo` item rendering for My Trainings + Manager |
| saveProgress (frontend) | `onboarding.html` | Accept + pass `value` parameter |
| openCamera/showPhotoPreview | `onboarding.html` | Copy from workflows.html |
| Photo CSS | `onboarding.html` | Copy from workflows.html |
| renderOBSection (Builder) | `onboarding.html` | Add photo item rendering in editor |
| add-ob-item handler | `onboarding.html` | Handle `itemType === 'photo'` |
| createNewOBPhotoItem | `onboarding.html` | New function |
| isSectionComplete (Go) | `backend/internal/onboarding/db.go` | Add `photo` type handling |

### Pattern: Photo Upload Flow (from workflows.html)
**What:** Camera capture -> preview modal -> presigned upload -> save URL as progress
**When to use:** Every photo item interaction
**Example:**
```javascript
// Source: workflows.html lines 1516-1578 [VERIFIED: codebase]
function handleOBPhotoCaptureClick(itemId, hireId, secId) {
  openCamera(function(file) {
    var objectURL = URL.createObjectURL(file);
    showPhotoPreview(objectURL, function() {
      // Show uploading spinner
      fetch('/api/v1/photos/presign', {
        method: 'POST', credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          path_prefix: 'onboarding',
          id: hireId,
          filename: itemId + '.jpg'
        })
      }).then(r => r.json())
        .then(presignResp => {
          return fetch(presignResp.url, {
            method: 'PUT', body: file,
            headers: { 'Content-Type': 'image/jpeg' }
          }).then(() => presignResp.public_url);
        })
        .then(publicUrl => {
          // Save progress with photo URL
          saveProgress(itemId, 'photo', true, sections, hireId, publicUrl);
          // Update UI to show thumbnail
        });
    }, function() {
      URL.revokeObjectURL(objectURL);
      handleOBPhotoCaptureClick(itemId, hireId, secId); // retake
    });
  });
}
```

### Anti-Patterns to Avoid
- **Server-side photo upload:** Do NOT route photo bytes through the Go server. Use presigned URLs for direct browser-to-DO-Spaces upload. [VERIFIED: existing pattern in photos/handler.go]
- **Using `api()` helper for presign:** The `api()` helper in onboarding.html prepends `/api/v1/` -- but `/api/v1/photos/presign` already includes the prefix. Use `fetch()` directly like workflows.html does. [VERIFIED: workflows.html line 1529 uses fetch() directly]
- **Forgetting the CHECK constraint:** Adding `'photo'` to `progress_type` allowed values requires updating the CHECK constraint on `ob_progress`. Without this, INSERT will fail at runtime.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Photo upload | Custom upload endpoint | DO Spaces presigned PUT | Already exists, battle-tested, no server load [VERIFIED: photos/handler.go] |
| Camera access | Custom camera UI | `<input type="file" capture="environment">` | Native browser camera integration works on iOS + Android [VERIFIED: workflows.html line 431] |
| Photo preview modal | Custom modal framework | `showPhotoPreview()` function | Copy from workflows.html, proven on mobile [VERIFIED: workflows.html lines 444-459] |

## Common Pitfalls

### Pitfall 1: Missing 'photo' in SQL Type Filters
**What goes wrong:** Photo items exist in `ob_items` but aren't counted in section progress queries, section completion checks, or manager hire overviews.
**Why it happens:** There are 6+ SQL queries in `db.go` that filter on `oi.type IN ('checkbox', 'video_series', 'faq')` and all must be updated.
**How to avoid:** Grep for `oi.type IN` in db.go and update every occurrence. There are instances in:
1. `sectionIncompleteItem()` (line 18) -- used by GetMyTrainings, GetManagerHires (twice)
2. `GetMyTrainings()` (line 670) -- `EXISTS` subquery checking section has items
3. `GetManagerHires()` (lines 735, 749, 804, 818) -- inline copies of the pattern
4. `IsSectionLockedForEdits()` (line 900) -- inline copy
5. `isSectionComplete()` Go function (line 597) -- needs `photo` case in switch
**Warning signs:** Photo items don't count toward progress percentages; sections with only photo items never complete.

### Pitfall 2: Frontend countSectionItems/countCheckedItems Not Updated
**What goes wrong:** Progress counter shows wrong counts (e.g., "0/0" when section has photo items).
**Why it happens:** `countSectionItems` (line 414) only handles `checkbox` and `video_series`. `countCheckedItems` (line 425) same. Photo items silently ignored.
**How to avoid:** Add `else if (item.type === 'photo') total += 1` and similar for checked (check `item.checked`).
**Warning signs:** Progress bar shows "0/0" for sections containing photo items.

### Pitfall 3: SaveProgress Frontend Missing Value Parameter
**What goes wrong:** Photo URL not saved to database; photo appears uploaded but is lost on page reload.
**Why it happens:** Current `saveProgress` function (line 456) sends `{ item_id, progress_type, checked }` without `value`.
**How to avoid:** Add optional `value` parameter to `saveProgress` and include it in the POST body.
**Warning signs:** Photo shows as uploaded, but reopening the training shows empty photo item.

### Pitfall 4: api() vs fetch() for Presign Endpoint
**What goes wrong:** Presign request goes to wrong URL.
**Why it happens:** The `api()` helper in onboarding.html constructs the URL differently than a raw `fetch()`. The photos presign endpoint is at `/api/v1/photos/presign`.
**How to avoid:** Check how `api()` constructs URLs in onboarding.html. If it prepends a base path, use `fetch()` directly like workflows.html does (line 1529).
**Warning signs:** 404 errors on presign requests.

### Pitfall 5: CHECK Constraint on progress_type
**What goes wrong:** `INSERT INTO ob_progress` fails with constraint violation when saving photo progress.
**Why it happens:** Current CHECK constraint only allows: `'item', 'video_part', 'faq', 'video_watch_position', 'sub_item'`. Must add `'photo'`.
**How to avoid:** Migration must DROP and re-ADD the CHECK constraint (same pattern as migrations 0026, 0028, 0050).
**Warning signs:** 500 error on saveProgress for photo items.

### Pitfall 6: GetHireTraining Not Populating PhotoURL
**What goes wrong:** Frontend receives item with `checked: true` but no `photo_url`, so it shows checkbox-style checkmark instead of photo thumbnail.
**Why it happens:** `GetHireTraining` (line 508-541) has a switch on item type to populate Checked/Viewed. Currently no branch for `photo` type. Also, progress query (line 417) doesn't SELECT `op.value`.
**How to avoid:** Add `photo` case in the item type switch; query `op.value` alongside other fields; build a value map keyed by item_id.
**Warning signs:** Photo items render as plain checkboxes with checkmarks.

## Code Examples

### Migration: Add value column and photo progress_type
```sql
-- Source: pattern from 0050_ob_progress_sub_item.sql [VERIFIED: codebase]
-- +goose Up
BEGIN;
ALTER TABLE ob_progress ADD COLUMN value TEXT;
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check
  CHECK (progress_type IN ('item', 'video_part', 'faq', 'video_watch_position', 'sub_item', 'photo'));
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check
  CHECK (progress_type IN ('item', 'video_part', 'faq', 'video_watch_position', 'sub_item'));
ALTER TABLE ob_progress DROP COLUMN value;
COMMIT;
```

### Go: Updated SaveProgress with value parameter
```go
// Source: adapted from db.go SaveProgress [VERIFIED: codebase line 922]
func SaveProgress(ctx context.Context, pool *pgxpool.Pool, hireID, itemID, progressType string, checked bool, maxWatchedTime *float64, value *string) error {
    if checked {
        _, err := pool.Exec(ctx, `
            INSERT INTO ob_progress (hire_id, item_id, progress_type, max_watched_time, value)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (hire_id, item_id, progress_type) DO UPDATE
            SET max_watched_time = GREATEST(ob_progress.max_watched_time, EXCLUDED.max_watched_time),
                value = COALESCE(EXCLUDED.value, ob_progress.value)
        `, hireID, itemID, progressType, maxWatchedTime, value)
        return err
    }
    // delete unchanged
}
```

### Go: Item struct with PhotoURL
```go
// Source: adapted from db.go Item struct [VERIFIED: codebase line 63]
type Item struct {
    // ... existing fields ...
    PhotoURL string `json:"photo_url,omitempty"`
}
```

### Frontend: Photo item rendering in My Trainings
```javascript
// Source: adapted from workflows.html photo rendering [VERIFIED: codebase line 2111]
} else if (item.type === 'photo') {
    var hasPhoto = !!item.photo_url;
    var canInteract = !readOnly && state === 'active';
    html += '<div class="item-row">';
    html += '<div class="ob-check' + (hasPhoto ? ' checked' : '') + '" style="cursor:default">' + (hasPhoto ? '&#10003;' : '') + '</div>';
    html += '<div style="flex:1"><div style="font-size:14px;font-weight:500">' + escapeOBAttr(item.label) + '</div>';
    if (hasPhoto) {
        html += '<div class="photo-thumb-wrap">' +
            '<a class="photo-thumb-link" href="' + item.photo_url + '" target="_blank">' +
            '<img class="photo-thumb" src="' + item.photo_url + '" alt="Photo">' +
            '</a>';
        if (canInteract) html += '<button class="photo-retake-link" data-action="ob-photo-retake" data-item-id="' + item.id + '" data-sec-id="' + sec.id + '" data-hire-id="' + hireId + '">Retake</button>';
        html += '</div>';
    } else if (canInteract) {
        html += '<div class="photo-capture-area"><button class="photo-capture-btn" data-action="ob-photo-capture" data-item-id="' + item.id + '" data-sec-id="' + sec.id + '" data-hire-id="' + hireId + '">&#128247; Take Photo</button></div>';
    } else if (readOnly) {
        html += '<div style="font-size:12px;color:var(--mut);padding:4px 0">No photo</div>';
    }
    html += '</div></div>';
}
```

### CSS to copy from workflows.html
```css
/* Source: workflows.html lines 108-161 [VERIFIED: codebase] */
.photo-modal{position:fixed;top:0;left:0;width:100vw;height:100vh;background:rgba(0,0,0,0.85);z-index:1000;display:flex;align-items:center;justify-content:center;flex-direction:column}
.photo-preview{max-width:100%;max-height:70vh;object-fit:contain}
.photo-modal-actions{position:fixed;bottom:32px;left:0;right:0;display:flex;justify-content:center;gap:16px;align-items:center}
.photo-confirm-btn{min-width:56px;min-height:56px;border-radius:28px;background:var(--info-bg);color:var(--info-tx);border:none;font-size:24px;cursor:pointer;display:flex;align-items:center;justify-content:center}
.photo-retake-btn{padding:12px 16px;background:transparent;color:var(--info-tx);border:none;font-size:14px;font-weight:500;cursor:pointer;min-height:44px;font-family:inherit}
.photo-thumb{width:72px;height:72px;border-radius:8px;object-fit:cover;border:0.5px solid var(--brd)}
.photo-capture-btn{padding:8px 12px;background:var(--bg);border:0.5px solid var(--brd);border-radius:8px;font-size:14px;cursor:pointer;font-family:inherit;color:var(--txt)}
.photo-retake-link{background:none;border:none;color:var(--info-tx);font-size:12px;cursor:pointer;padding:0;font-family:inherit}
.photo-uploading{display:flex;align-items:center;gap:10px;padding:8px 0;font-size:13px;color:var(--mut)}
.photo-spinner{width:18px;height:18px;border:2px solid var(--brd);border-top-color:var(--info-tx);border-radius:50%;animation:photo-spin 0.8s linear infinite;flex-shrink:0}
.photo-thumb-wrap{display:flex;flex-direction:column;gap:4px;align-items:flex-start;padding:6px 0}
.photo-thumb-link{display:block;cursor:pointer}
/* Also need @keyframes photo-spin — check if it exists in workflows.html */
```

## SQL Queries Requiring 'photo' Addition

Complete inventory of every SQL location that filters on item type [VERIFIED: grep of db.go]:

| Location | Line | Current Filter | Context |
|----------|------|---------------|---------|
| `sectionIncompleteItem()` | 18 | `oi.type IN ('checkbox', 'video_series', 'faq')` | Reusable SQL fragment for incomplete section detection |
| `GetMyTrainings()` | 670 | `oi.type IN ('checkbox', 'video_series', 'faq')` | EXISTS check: section has countable items |
| `GetManagerHires()` | 735 | `oi.type IN ('checkbox', 'video_series', 'faq')` | Inline incomplete item check (copy of sectionIncompleteItem) |
| `GetManagerHires()` | 749 | `oi.type IN ('checkbox', 'video_series', 'faq')` | EXISTS check: section has items (for counting) |
| `GetManagerHires()` | 804 | `oi.type IN ('checkbox', 'video_series', 'faq')` | Per-hire template detail: incomplete item check |
| `GetManagerHires()` | 818 | `oi.type IN ('checkbox', 'video_series', 'faq')` | Per-hire template detail: EXISTS section has items |
| `IsSectionLockedForEdits()` | 900 | `oi.type IN ('checkbox', 'video_series', 'faq')` | Inline incomplete item check |
| `isSectionComplete()` Go func | 597-628 | Switch on `item.Type` | Go logic: checkbox/video_series/faq branches |
| `GetHireTraining()` Go func | 510-541 | Switch on `item.Type` | Populates Checked/Viewed on items |
| `countSectionItems()` JS func | 414-422 | `item.type === 'checkbox'` / `'video_series'` | Frontend progress counter |
| `countCheckedItems()` JS func | 425-436 | Same | Frontend checked counter |
| `recomputeSectionState()` JS func | 467-489 | Calls countSectionItems/countCheckedItems | Section state after user action |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No photo type in onboarding | Photo type added (this phase) | Now | Enables visual verification in training checklists |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `api()` helper in onboarding.html will send `value` field if included in the body object | Architecture Patterns | LOW -- `api()` is a thin wrapper around fetch; verified it sends the body as-is |
| A2 | The `@keyframes photo-spin` animation exists in workflows.html and needs to be copied | Code Examples (CSS) | LOW -- if missing the spinner won't animate but upload still works |

## Open Questions

1. **Does `api()` in onboarding.html prepend a path?**
   - What we know: In workflows.html, `fetch('/api/v1/photos/presign', ...)` is used directly. The onboarding `api()` helper is called like `api('POST', '/api/v1/onboarding/saveProgress', ...)`.
   - What's unclear: Whether `api()` modifies the URL or passes it through.
   - Recommendation: Check `api()` implementation in onboarding.html. If it passes URL through, use it for presign too. If it prepends something, use raw `fetch()` for presign.

## Sources

### Primary (HIGH confidence)
- `backend/internal/onboarding/db.go` -- All SQL queries, Item/ProgressEntry structs, SaveProgress function
- `backend/internal/onboarding/handler.go` -- SaveProgressHandler, all HTTP handlers
- `backend/internal/photos/handler.go` -- PresignUploadHandler interface (path_prefix, id, filename)
- `backend/internal/photos/spaces.go` -- GeneratePresignedPutURL, PublicURL
- `backend/cmd/server/main.go` -- Route registration confirming `/api/v1/photos/presign` exists
- `backend/internal/db/migrations/0050_ob_progress_sub_item.sql` -- Latest CHECK constraint pattern
- `onboarding.html` -- Frontend: saveProgress, countSectionItems, countCheckedItems, renderSections, renderOBSection, add-ob-item handler, createNewOB* functions
- `workflows.html` -- Photo infrastructure: openCamera (line 427), showPhotoPreview (line 444), handlePhotoCaptureClick (line 1516), photo CSS (lines 108-161)

### Secondary (MEDIUM confidence)
- None

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, all infrastructure verified in codebase
- Architecture: HIGH -- exact code paths identified and line numbers verified
- Pitfalls: HIGH -- every SQL query location inventoried by grep

**Research date:** 2026-05-20
**Valid until:** 2026-06-20 (stable codebase, no external dependency drift)
