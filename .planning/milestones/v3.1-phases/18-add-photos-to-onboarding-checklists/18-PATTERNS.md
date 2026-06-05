# Phase 18: Add Photos to Onboarding Checklists - Pattern Map

**Mapped:** 2026-05-20
**Files analyzed:** 9 (1 new migration, 2 modified backend, 1 modified frontend)
**Analogs found:** 5 / 5

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `backend/internal/db/migrations/0056_ob_progress_photo.sql` | migration | batch | `backend/internal/db/migrations/0050_ob_progress_sub_item.sql` | exact |
| `backend/internal/onboarding/db.go` (modify) | service | CRUD | self (existing patterns for checkbox/video_series/faq) | exact |
| `backend/internal/onboarding/handler.go` (modify) | controller | request-response | self (`SaveProgressHandler` lines 169-217) | exact |
| `onboarding.html` (modify — My Trainings render) | component | request-response | `workflows.html` (`handlePhotoCaptureClick` lines 1516-1578) | exact |
| `onboarding.html` (modify — Builder render) | component | request-response | self (`renderOBCheckboxItem` lines 696-716) | exact |
| `onboarding.html` (modify — count/progress functions) | utility | transform | self (`countSectionItems` line 414, `countCheckedItems` line 425) | exact |
| `onboarding.html` (modify — CSS) | component | n/a | `workflows.html` (lines 107-159) | exact |
| `onboarding.html` (modify — openCamera/showPhotoPreview) | utility | event-driven | `workflows.html` (lines 427-459) | exact |

## Pattern Assignments

### `backend/internal/db/migrations/0056_ob_progress_photo.sql` (migration, batch)

**Analog:** `backend/internal/db/migrations/0050_ob_progress_sub_item.sql`

**Full migration pattern** (lines 1-7):
```sql
-- +goose Up
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check CHECK (progress_type IN ('item', 'video_part', 'faq', 'video_watch_position', 'sub_item'));

-- +goose Down
ALTER TABLE ob_progress DROP CONSTRAINT ob_progress_progress_type_check;
ALTER TABLE ob_progress ADD CONSTRAINT ob_progress_progress_type_check CHECK (progress_type IN ('item', 'video_part', 'faq', 'video_watch_position'));
```

**What to add:** Wrap in `BEGIN;`/`COMMIT;` for atomicity, add `'photo'` to the CHECK constraint list, add `ALTER TABLE ob_progress ADD COLUMN value TEXT;` in Up and `DROP COLUMN value` in Down.

---

### `backend/internal/onboarding/db.go` (service, CRUD)

**Analog:** Self — multiple locations to modify.

**Item struct pattern** (lines 62-75):
```go
type Item struct {
    ID         string      `json:"id"`
    Type       string      `json:"type"`
    Label      string      `json:"label"`
    Answer     *string     `json:"answer,omitempty"`
    SortOrder  int         `json:"sort_order"`
    SubItems   []SubItem   `json:"sub_items,omitempty"`
    VideoParts []VideoPart `json:"video_parts,omitempty"`
    Checked bool `json:"checked"`
    Viewed bool `json:"viewed,omitempty"`
}
```
**Add:** `PhotoURL string \`json:"photo_url,omitempty"\`` field.

**SaveProgress function pattern** (lines 920-936):
```go
func SaveProgress(ctx context.Context, pool *pgxpool.Pool, hireID, itemID, progressType string, checked bool, maxWatchedTime *float64) error {
    if checked {
        _, err := pool.Exec(ctx, `
            INSERT INTO ob_progress (hire_id, item_id, progress_type, max_watched_time)
            VALUES ($1, $2, $3, $4)
            ON CONFLICT (hire_id, item_id, progress_type) DO UPDATE SET max_watched_time = GREATEST(ob_progress.max_watched_time, EXCLUDED.max_watched_time)
        `, hireID, itemID, progressType, maxWatchedTime)
        return err
    }
    _, err := pool.Exec(ctx, `
        DELETE FROM ob_progress
        WHERE hire_id = $1 AND item_id = $2 AND progress_type = $3
    `, hireID, itemID, progressType)
    return err
}
```
**Modify:** Add `value *string` parameter. Add `value` to INSERT columns and UPSERT logic: `value = COALESCE(EXCLUDED.value, ob_progress.value)`.

**SQL type filter pattern** (line 18 — `sectionIncompleteItem`):
```go
return `
    SELECT 1 FROM ob_items oi
    WHERE oi.section_id = os.id AND oi.type IN ('checkbox', 'video_series', 'faq')
    AND (
        (oi.type != 'video_series' AND NOT EXISTS (
            SELECT 1 FROM ob_progress op WHERE op.item_id = oi.id AND op.hire_id = ` + hireParam + `
        ))
        OR
        (oi.type = 'video_series' AND (
            ...
        ))
    )`
```
**Modify:** Add `'photo'` to `IN ('checkbox', 'video_series', 'faq', 'photo')`. Photo items follow the same non-`video_series` branch (check for existence of a progress row). Apply to ALL 7 SQL locations listed in RESEARCH.md.

**Progress query pattern** (lines 416-426 — `GetHireTraining`):
```go
progressRows, err := pool.Query(ctx, `
    SELECT op.item_id, op.progress_type, op.checked_at
    FROM ob_progress op
    LEFT JOIN ob_items oi ON oi.id = op.item_id
    ...
    WHERE op.hire_id = $1 AND os.template_id = $2
`, hireID, templateID)
```
**Modify:** Add `op.value` to SELECT. Build a `valueMap map[string]string` alongside `progressMap`. Populate `item.PhotoURL` from `valueMap[item.ID]` in the photo type branch.

**Item type switch pattern — GetHireTraining** (lines 510-541):
```go
for j := range sec.Items {
    item := &sec.Items[j]
    if item.Type == "video_series" {
        // ... populate VideoParts
    } else if item.Type == "faq" {
        item.Viewed = progressMap[item.ID+":faq"]
    } else {
        // Checkbox: check sub-items or direct progress
        item.Checked = progressMap[item.ID+":item"]
    }
}
```
**Add:** `else if item.Type == "photo"` branch before the `else` block. Set `item.Checked = progressMap[item.ID+":photo"]` and `item.PhotoURL = valueMap[item.ID]`.

**isSectionComplete pattern** (lines 597-629):
```go
func isSectionComplete(sec Section, progressMap map[string]bool) bool {
    totalCheckable := 0
    for _, item := range sec.Items {
        if item.Type == "faq" {
            totalCheckable++
            if !progressMap[item.ID+":faq"] { return false }
        } else if item.Type == "video_series" {
            // ... check all parts
        } else {
            // checkbox — check sub-items or direct
            totalCheckable++
            if !progressMap[item.ID+":item"] { return false }
        }
    }
    return totalCheckable > 0
}
```
**Add:** `else if item.Type == "photo"` branch: increment `totalCheckable`, check `progressMap[item.ID+":photo"]`.

---

### `backend/internal/onboarding/handler.go` (controller, request-response)

**Analog:** Self — `SaveProgressHandler` (lines 169-217).

**Request body struct pattern** (lines 179-184):
```go
var body struct {
    ItemID         string   `json:"item_id"`
    ProgressType   string   `json:"progress_type"`
    Checked        bool     `json:"checked"`
    MaxWatchedTime *float64 `json:"max_watched_time,omitempty"`
}
```
**Add:** `Value *string \`json:"value,omitempty"\`` field.

**SaveProgress call pattern** (line 210):
```go
if err := SaveProgress(r.Context(), pool, user.ID, body.ItemID, body.ProgressType, body.Checked, body.MaxWatchedTime); err != nil {
```
**Modify:** Pass `body.Value` as the new parameter.

---

### `onboarding.html` — Photo CSS (copy from `workflows.html`)

**Analog:** `workflows.html` lines 107-159

**CSS classes to copy verbatim:**
```css
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
@keyframes photo-spin{to{transform:rotate(360deg)}}
.photo-thumb-wrap{display:flex;flex-direction:column;gap:4px;align-items:flex-start;padding:6px 0}
.photo-thumb-link{display:block;cursor:pointer}
```

---

### `onboarding.html` — openCamera / showPhotoPreview (copy from `workflows.html`)

**Analog:** `workflows.html` lines 427-459

**openCamera function** (lines 427-442):
```javascript
function openCamera(onCapture) {
  var inp = document.createElement('input');
  inp.type = 'file';
  inp.accept = 'image/*';
  inp.capture = 'environment';
  inp.style.cssText = 'position:fixed;top:-100px;left:-100px;opacity:0;pointer-events:none';
  document.body.appendChild(inp);
  var cleanup = setTimeout(function() { if (inp.parentNode) inp.parentNode.removeChild(inp); }, 30000);
  inp.addEventListener('change', function() {
    clearTimeout(cleanup);
    var file = inp.files && inp.files[0];
    if (inp.parentNode) inp.parentNode.removeChild(inp);
    if (file) onCapture(file);
  });
  inp.click();
}
```

**showPhotoPreview function** (lines 444-459):
```javascript
function showPhotoPreview(objectURL, onConfirm, onRetake) {
  var modal = document.createElement('div');
  modal.className = 'photo-modal';
  modal.innerHTML = '<img class="photo-preview" src="' + objectURL + '">' +
    '<div class="photo-modal-actions">' +
    '<button class="photo-confirm-btn" aria-label="Confirm photo">\u2713</button>' +
    '<button class="photo-retake-btn">\u21bb Retake</button>' +
    '</div>';
  document.body.appendChild(modal);
  modal.querySelector('.photo-confirm-btn').onclick = function() {
    modal.remove(); onConfirm(objectURL);
  };
  modal.querySelector('.photo-retake-btn').onclick = function() {
    modal.remove(); onRetake();
  };
}
```

---

### `onboarding.html` — Photo upload handler (adapt from `workflows.html`)

**Analog:** `workflows.html` `handlePhotoCaptureClick` lines 1516-1578

**Upload flow pattern:**
```javascript
function handlePhotoCaptureClick(fldId) {
  openCamera(function(file) {
    var objectURL = URL.createObjectURL(file);
    showPhotoPreview(objectURL, function() {
      var captureArea = /* locate DOM element for this item */;
      if (captureArea) {
        captureArea.innerHTML = '<div class="photo-uploading"><div class="photo-spinner"></div>Uploading...</div>';
      }
      fetch('/api/v1/photos/presign', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path_prefix: 'checklists', id: tplId, filename: fldId + '.jpg' })
      }).then(function(r) {
        if (r.status === 401) { location.href = 'login.html'; throw new Error('unauthorized'); }
        if (!r.ok) throw new Error('presign failed: ' + r.status);
        return r.json();
      }).then(function(presignResp) {
          return fetch(presignResp.url, {
            method: 'PUT', body: file,
            headers: { 'Content-Type': 'image/jpeg' }
          }).then(function(putResp) {
            if (!putResp.ok) throw new Error('PUT failed: ' + putResp.status);
            return presignResp.public_url;
          });
        })
        .then(function(publicUrl) {
          URL.revokeObjectURL(objectURL);
          // Save progress + update UI with thumbnail
        })
        .catch(function(err) {
          console.error('Photo upload failed:', err);
          URL.revokeObjectURL(objectURL);
          // Show retry button
        });
    }, function() {
      URL.revokeObjectURL(objectURL);
      handlePhotoCaptureClick(fldId); // retake: re-open camera
    });
  });
}
```
**Adapt for onboarding:** Change `path_prefix` to `'onboarding'`, `id` to `hireId`, `filename` to `itemId + '.jpg'`. After upload, call `saveProgress(itemId, 'photo', true, sections, hireId, publicUrl)` and render thumbnail.

---

### `onboarding.html` — My Trainings item rendering (modify)

**Analog:** Self — checkbox item rendering (lines 326-354)

**Checkbox rendering pattern:**
```javascript
if (item.type === 'checkbox') {
    var checked = item.checked === true;
    var checkClass = 'ob-check' + (checked ? ' checked' : '');
    var canInteract = !readOnly && state === 'active';
    var checkable = canInteract ? ' data-action="toggle-item" data-item-id="'+item.id+'" ...' : '';
    html += '<div class="item-row">';
    html += '<div class="'+checkClass+'"'+checkable+'>'+(checked?'&#10003;':'')+'</div>';
    html += '<div style="flex:1"><div style="font-size:14px;font-weight:500">'+escapeOBAttr(item.label)+'</div>';
    // ... content ...
    html += '</div></div>';
}
```
**Add:** `else if (item.type === 'photo')` block after video_series. Follow same `item-row` structure. Show thumbnail if `item.photo_url`, capture button if interactive, "No photo" if read-only.

---

### `onboarding.html` — countSectionItems / countCheckedItems (modify)

**Analog:** Self (lines 414-436)

**countSectionItems pattern:**
```javascript
function countSectionItems(sec) {
  var total = 0;
  (sec.items || []).forEach(function(item) {
    if (item.type === 'checkbox') {
      var subs = item.sub_items || [];
      total += subs.length > 0 ? subs.length : 1;
    } else if (item.type === 'video_series') total += (item.video_parts || []).length;
  });
  return total;
}
```
**Add:** `else if (item.type === 'photo') total += 1;`

**countCheckedItems pattern:**
```javascript
function countCheckedItems(sec) {
  var checked = 0;
  (sec.items || []).forEach(function(item) {
    if (item.type === 'checkbox') {
      // ... sub-items logic
    } else if (item.type === 'video_series') /* ... */;
  });
  return checked;
}
```
**Add:** `else if (item.type === 'photo') { if (item.checked) checked++; }`

---

### `onboarding.html` — saveProgress function (modify)

**Analog:** Self (lines 456-465)

**Current pattern:**
```javascript
async function saveProgress(itemId, progressType, checked, sections, hireId) {
  try {
    await api('POST', '/api/v1/onboarding/saveProgress', { item_id: itemId, progress_type: progressType, checked: checked });
    SAVE_RETRY[itemId] = 0;
    return true;
  } catch(e) {
    SAVE_RETRY[itemId] = (SAVE_RETRY[itemId] || 0) + 1;
    return false;
  }
}
```
**Modify:** Add optional `value` parameter. Include `value` in the POST body when truthy.

**Note:** The `api()` helper (line 138) passes URL through to `fetch()` directly, so it can also be used for `/api/v1/photos/presign` if preferred over raw `fetch()`.

---

### `onboarding.html` — Builder rendering (modify)

**Analog:** Self — `renderOBCheckboxItem` (lines 696-716)

**Builder item card pattern:**
```javascript
function renderOBCheckboxItem(item, secId, idx) {
  var html = '<div class="ob-item-card" data-item-id="'+item.id+'">';
  html += '<div class="ob-item-header">';
  html += '<span class="drag-handle">&#9776;</span>';
  html += '<input type="text" value="'+escapeOBAttr(item.label)+'" data-action="item-label-input" data-sec-id="'+secId+'" data-item-idx="'+idx+'" placeholder="Checkbox label" style="flex:1;padding:6px 8px;...">';
  html += '<span style="font-size:11px;color:var(--mut);white-space:nowrap">Checkbox</span>';
  html += '<span data-action="delete-ob-item" data-sec-id="'+secId+'" data-item-idx="'+idx+'" style="color:#c0392b;cursor:pointer;font-size:14px">&#10005;</span>';
  html += '</div>';
  // No body for photo items — no sub-items or video parts
  html += '</div>';
  return html;
}
```
**Create:** `renderOBPhotoItem(item, secId, idx)` — same header pattern (drag handle, label input, type badge "Photo", delete button). No body section needed.

**Add button pattern** (lines 844-847):
```javascript
'<button data-action="add-ob-item" data-sec-id="'+sec.id+'" data-item-type="checkbox" style="padding:6px 12px;background:var(--info-bg);color:var(--info-tx);border:none;border-radius:8px;font-size:12px;font-weight:500;cursor:pointer">+ Checkbox</button>'+
'<button data-action="add-ob-item" data-sec-id="'+sec.id+'" data-item-type="video_series" style="...">+ Video Series</button>'+
```
**Add:** Third button with `data-item-type="photo"` text `+ Photo`.

**Add-item handler pattern** (lines 1602-1608):
```javascript
} else if (action === 'add-ob-item') {
    var sec = (obBuilderState.localCopy.sections||[]).find(function(s) { return s.id === btn.dataset.secId; });
    if (sec) {
      var itemType = btn.dataset.itemType;
      if (itemType === 'checkbox') sec.items.push(createNewOBCheckbox(''));
      else if (itemType === 'video_series') sec.items.push(createNewOBVideoSeries(''));
      renderOBBuilder();
    }
```
**Add:** `else if (itemType === 'photo') sec.items.push(createNewOBPhotoItem(''));`

**Builder item render switch** (lines 837-841):
```javascript
(sec.items || []).forEach(function(item, idx) {
  if (item.type === 'checkbox') {
    itemsHtml += renderOBCheckboxItem(item, sec.id, idx);
  } else if (item.type === 'video_series') {
    itemsHtml += renderOBVideoItem(item, sec.id, idx);
  }
});
```
**Add:** `else if (item.type === 'photo') { itemsHtml += renderOBPhotoItem(item, sec.id, idx); }`

---

### `onboarding.html` — createNewOBPhotoItem (new function)

**Analog:** `createNewOBCheckbox` (line 171)

**Pattern:**
```javascript
function createNewOBCheckbox(label) {
  return { id: generateOBId('itm'), type: 'checkbox', label: label || '', sub_items: [] };
}
```
**Create:**
```javascript
function createNewOBPhotoItem(label) {
  return { id: generateOBId('itm'), type: 'photo', label: label || '', sub_items: [] };
}
```

---

## Shared Patterns

### Authentication
**Source:** `backend/internal/onboarding/handler.go` lines 33-41, 46-51
**Apply to:** No new handlers needed — `SaveProgressHandler` already exists and handles auth.
```go
user := auth.UserFromContext(r.Context())
if user == nil {
    writeError(w, http.StatusUnauthorized, "unauthorized")
    return
}
```

### Error Handling (Backend)
**Source:** `backend/internal/onboarding/handler.go` lines 17-31
**Apply to:** All handler modifications.
```go
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if v != nil { json.NewEncoder(w).Encode(v) }
}
func writeError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
```

### API Helper (Frontend)
**Source:** `onboarding.html` lines 138-147
**Apply to:** All frontend API calls from onboarding.
```javascript
async function api(method, path, body) {
  var opts = { method: method, headers: { 'Content-Type': 'application/json' } };
  if (body) opts.body = JSON.stringify(body);
  var res = await fetch(path, opts);
  if (res.status === 401) { window.location.href = '/login.html'; throw new Error('unauthorized'); }
  if (res.status === 204) return null;
  var data = await res.json();
  if (!res.ok) throw data;
  return data;
}
```
**Note:** `api()` passes URL through directly to `fetch()`. Can be used for `/api/v1/photos/presign`. However, the presigned PUT to DO Spaces must use raw `fetch()` (no JSON headers, binary body).

### Event Delegation (Frontend)
**Source:** `onboarding.html` — click handler uses `data-action` attributes
**Apply to:** New photo capture/retake actions.
**Pattern:** Add `ob-photo-capture` and `ob-photo-retake` action handlers in the existing click delegation block.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| (none) | -- | -- | All files have exact analogs in the existing codebase |

## Metadata

**Analog search scope:** `backend/internal/onboarding/`, `backend/internal/photos/`, `backend/internal/db/migrations/`, `onboarding.html`, `workflows.html`
**Files scanned:** 8 source files read
**Pattern extraction date:** 2026-05-20
