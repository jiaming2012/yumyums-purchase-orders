# Phase 3: Photo, Approval, and Integration - Research

**Researched:** 2026-04-13
**Domain:** Vanilla JS PWA — photo capture via file input, in-memory approval state, three-tab navigation, pre-built template data
**Confidence:** HIGH — primary sources are the existing codebase, prior project pitfall research, and the approved UI-SPEC from this phase

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Uses `<input type="file" capture="environment">` — opens rear camera directly. No JS camera API. Works in iOS standalone PWA.
- **D-02:** Two-step flow after capture: (1) Full-size preview with checkmark icon and retake button. (2) After checkmark is pressed, collapses to small thumbnail inline in the field row with a "Retake" button next to it.
- **D-03:** Corrective action card photo stub becomes functional — the "Add photo" button in fail cards also opens the camera for evidence capture. Same two-step preview flow.
- **D-04:** New third tab: "Approvals" — sits between My Checklists and Builder. Shows list of submitted checklists awaiting manager review.
- **D-05:** Review view is a summary card: checklist name, who submitted, when, completion %, any fail triggers fired, and photos captured. Approve/Reject buttons at bottom. Not a full read-only runner.
- **D-06:** Approval is one-tap (toast "Approved ✓"). Rejection requires the manager to type a reason before confirming (modal or inline text input + confirm).
- **D-07:** After approval/rejection, submission moves to a completed/rejected state. No notification to crew (mock only).
- **D-08:** Two pre-built templates: Setup Checklist and Closing Checklist.
- **D-09:** Minimal placeholder items — enough to demonstrate the feature, not realistic food truck items.
- **D-10:** Operations tile already exists on index.html (added previously). INTG-01 is already satisfied — just verify it links correctly.
- **D-11:** INTG-02 (role-based tab access) — Builder tab already gated. Approvals tab should also be gated to manager+ roles. My Checklists visible to all.

### Claude's Discretion

- Exact summary card layout and information density
- Tab ordering (My Checklists / Approvals / Builder)
- How "requires approval" flag from builder connects to the approval queue
- Photo thumbnail size and retake button styling

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FILL-06 | Crew member can capture a photo using the device camera for a photo field | iOS file input workaround (Pitfall 9); `URL.createObjectURL()`; full-screen preview modal; state stored in `MOCK_RESPONSES[fldId]` |
| APRV-01 | Owner/manager can mark a template as requiring manager approval before it's considered complete | `requires_approval` toggle already exists in builder meta card; connect to `submitChecklist()` routing logic |
| APRV-02 | When a checklist requires approval, the completed submission shows a "Pending Approval" status | `PENDING_APPROVALS` dict; `.approval-badge` on checklist list card; submit confirmation message change |
| APRV-03 | Manager can approve or reject a completed checklist with optional notes | Approvals tab; approval card components; one-tap approve vs. two-step rejection with required reason; `APPROVED_SUBMISSIONS` / `REJECTED_SUBMISSIONS` dicts |
| INTG-01 | Workflows app appears as an active tile on the HQ launcher (index.html) | Already present as `<a class="tile active" href="workflows.html">` — verification only |
| INTG-02 | Fill-out tab visible to all roles; builder and approvals tabs restricted by permissions | JS role check on tab render; same pattern as existing builder gate (`MOCK_CURRENT_USER.permissions`) |
| INTG-03 | Mock includes 2-3 pre-built food truck templates | Two templates added to `MOCK_TEMPLATES`: "Setup Checklist" + "Closing Checklist" with minimal placeholder sections/fields |

</phase_requirements>

---

## Summary

Phase 3 completes the workflow engine with three independent deliverable groups that share a single HTML file (`workflows.html`): functional photo capture, manager approval flow, and integration cleanup. All deliverables are pure vanilla JS mutations on the existing codebase — no new libraries, no new files, no build step.

The most technically fraught piece is photo capture. The iOS PWA single-use `<input type="file">` bug (Pitfall 9) means the standard DOM approach fails after the first capture. The established workaround — create a fresh `<input>` node each time, append to body, programmatically click, remove after change fires — is well-documented in project pitfall research and must be applied consistently in both the standalone photo field row and the corrective action card photo button.

The approval flow is pure in-memory state management: two new dicts (`PENDING_APPROVALS`, `APPROVED_SUBMISSIONS`, `REJECTED_SUBMISSIONS`) and a third tab section. The existing `show(n)` function handles 2 tabs and must be extended to handle 3. The existing builder tab shifts from `#s2` to `#s3`. The approval tab becomes `#s2`. Tab button IDs `t1`/`t2` are joined by `t3`. The existing builder and fill-out code does not change — only the tab IDs/sections renumber.

Integration is mostly verification: the Operations tile (`index.html`) already links to `workflows.html`, and `sw.js` already includes `workflows.html` in ASSETS at v19. The only integration action required is bumping the sw.js cache version after any code changes in this phase.

**Primary recommendation:** Implement in three independent tasks: (1) photo capture + state, (2) three-tab restructure + approval flow, (3) pre-built templates + INTG verification. Photo capture must be tested on a real iOS device in standalone mode before the task is closed.

---

## Standard Stack

### Core (No Build Step)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Vanilla JS (ES2020+) | native | All interactivity and state | Established project convention; no framework allowed |
| CSS custom properties | native | Theming and dark mode | Already in every page via `:root` block |
| `URL.createObjectURL()` | browser API | Convert captured file to displayable URL | Standard browser API; works in all modern mobile browsers; no library needed |
| SortableJS | 1.15.7 (carry-forward) | Touch-native drag reorder in builder | Already loaded via unpkg CDN — no new import needed |

### What NOT to Use

| Rejected | Reason |
|----------|--------|
| `MediaDevices.getUserMedia()` / WebRTC camera API | D-01 locked `<input type="file">` — simpler, works in iOS standalone PWA, no permissions dialog friction |
| `FileReader.readAsDataURL()` | `URL.createObjectURL()` is faster (no base64 encoding), lower memory; sufficient for in-memory display |
| Any state library | Project convention: plain JS objects, no framework |
| `capture="environment"` without iOS workaround | Causes single-use stall bug (Pitfall 9) — must use fresh-input creation pattern |

---

## Architecture Patterns

### Current Tab Structure (Phase 2 State)

```
.tabs
  <button id="t1" class="on">My Checklists</button>  → #s1
  <button id="t2">Builder</button>                   → #s2
```

### Target Tab Structure (Phase 3)

```
.tabs
  <button id="t1" class="on">My Checklists</button>  → #s1 (unchanged)
  <button id="t2">Approvals</button>                 → #s2 (NEW — gated to manager+)
  <button id="t3">Builder</button>                   → #s3 (renumbered from #s2)
```

**Critical:** All existing builder code references `#s2`. After renaming to `#s3`, the HTML section container changes from `<div id="s2">` to `<div id="s3">`. The JS `show(n)` loop must iterate `[1,2,3]`. Tab button `t2` becomes the Approvals tab; builder tab becomes `t3`. No logic inside the builder itself changes — only the container ID.

### New State Dicts

```javascript
// Approval queue — populated by submitChecklist() when requires_approval === true
const PENDING_APPROVALS = {};
// { [tplId]: { tplId, tplName, submittedBy: CURRENT_USER, submittedAt: ISO string,
//              responsesSnapshot: {...MOCK_RESPONSES}, failCount: N, photoCount: N } }

const APPROVED_SUBMISSIONS = {};
// { [tplId]: { ...pendingEntry, approvedBy, approvedAt } }

const REJECTED_SUBMISSIONS = {};
// { [tplId]: { ...pendingEntry, rejectedBy, rejectedAt, reason } }
```

### Photo Capture State

Photo responses are stored in `MOCK_RESPONSES` alongside other field types:

```javascript
// After capture confirm:
MOCK_RESPONSES[fldId] = {
  value: objectURL,       // string from URL.createObjectURL(file)
  answeredBy: CURRENT_USER.name,
  answeredAt: new Date()
};
```

For corrective action card photos, stored in `FAIL_NOTES`:

```javascript
FAIL_NOTES[fldId].photo = objectURL;  // string
```

### iOS Photo Capture Workaround Pattern

**Problem:** `<input type="file" capture="environment">` stalls after first use on iOS 12+ PWA. After dismissal or reuse, the `change` event no longer fires.

**Solution (from Pitfall 9 + UI-SPEC):** Create a fresh `<input>` element each time the camera is triggered. Do not reuse the same input node across captures.

```javascript
function openCamera(onCapture) {
  var inp = document.createElement('input');
  inp.type = 'file';
  inp.accept = 'image/*';
  inp.capture = 'environment';
  inp.style.cssText = 'position:fixed;top:-100px;left:-100px;opacity:0;pointer-events:none';
  document.body.appendChild(inp);
  inp.addEventListener('change', function() {
    var file = inp.files && inp.files[0];
    document.body.removeChild(inp);
    if (file) onCapture(file);
  });
  inp.click();
}
```

**Note:** If user dismisses the camera without taking a photo, the `change` event does not fire and the dangling `<input>` stays in the DOM. Add a `blur`/`focus` trick or a `setTimeout` cleanup to remove it if no file is selected. A simple approach: remove after 30 seconds as a fallback.

### Photo Preview Modal Pattern

Modal must be appended to `document.body` directly (outside `.app`) to achieve true full-screen overlay:

```javascript
function showPhotoPreview(objectURL, onConfirm, onRetake) {
  var modal = document.createElement('div');
  modal.className = 'photo-modal';
  modal.innerHTML = '<img class="photo-preview" src="' + objectURL + '">' +
    '<div class="photo-modal-actions">' +
    '<button class="photo-confirm-btn" aria-label="Confirm photo">✓</button>' +
    '<button class="photo-retake-btn">↻ Retake</button>' +
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

Key CSS for the modal (add to `<style>` block):

```css
.photo-modal{position:fixed;top:0;left:0;width:100vw;height:100vh;background:rgba(0,0,0,0.85);z-index:1000;display:flex;align-items:center;justify-content:center;flex-direction:column}
.photo-preview{max-width:100%;max-height:70vh;object-fit:contain}
.photo-modal-actions{position:fixed;bottom:32px;left:0;right:0;display:flex;justify-content:center;gap:16px;align-items:center}
.photo-confirm-btn{min-width:56px;min-height:56px;border-radius:28px;background:var(--info-bg);color:var(--info-tx);border:none;font-size:24px;cursor:pointer;display:flex;align-items:center;justify-content:center}
.photo-retake-btn{padding:12px 16px;background:transparent;color:var(--info-tx);border:none;font-size:14px;font-weight:500;cursor:pointer;min-height:44px;font-family:inherit}
```

### `renderRunnerField()` Photo Field Replacement

Current code (line 1122–1127 of `workflows.html`):

```javascript
if (fld.type === 'photo') {
  return `<div class="fill-field" data-field-id="${fld.id}">
  <div class="fill-field-label">${escapeHtml(fld.label)}</div>
  <div style="padding:8px 0;color:var(--mut);font-size:13px">Photo capture coming in Phase 3</div>
</div>`;
}
```

Replace with: field label + either capture button (no photo) or thumbnail + retake link (photo captured).

```javascript
if (fld.type === 'photo') {
  const photoResp = MOCK_RESPONSES[fld.id];
  const photoUrl = photoResp ? photoResp.value : null;
  const attribution = photoResp ? `<div class="fill-attribution">${escapeHtml(photoResp.answeredBy)} · ${formatTime(photoResp.answeredAt)}</div>` : '';
  if (photoUrl) {
    return `<div class="fill-field" data-field-id="${fld.id}">
  <div class="fill-field-label">${escapeHtml(fld.label)}</div>
  <div class="fill-field-row" style="gap:12px;margin-top:4px">
    <img class="photo-thumb" src="${photoUrl}" alt="Photo">
    <button class="photo-retake-link" data-action="photo-retake" data-fld-id="${fld.id}">Retake</button>
  </div>
  ${attribution}
</div>`;
  }
  return `<div class="fill-field" data-field-id="${fld.id}">
  <div class="fill-field-label">${escapeHtml(fld.label)}</div>
  <button class="photo-capture-btn" data-action="photo-capture" data-fld-id="${fld.id}" style="margin-top:4px">📷 Tap to take photo</button>
  <div style="font-size:12px;color:var(--mut);margin-top:4px">Tap to take photo</div>
</div>`;
}
```

Additional CSS:

```css
.photo-thumb{width:72px;height:72px;border-radius:8px;object-fit:cover;border:0.5px solid var(--brd)}
.photo-capture-btn{padding:8px 12px;background:var(--bg);border:0.5px solid var(--brd);border-radius:8px;font-size:14px;cursor:pointer;font-family:inherit;color:var(--txt)}
.photo-retake-link{background:none;border:none;color:var(--info-tx);font-size:12px;cursor:pointer;padding:0;font-family:inherit}
```

### `renderFailCard()` Photo Stub Replacement

Current stub in `renderFailCard()` (line 949):

```javascript
'<button class="photo-stub-btn" disabled>📷 Add photo</button>' +
'<span style="font-size:12px;color:var(--mut)">Coming in Phase 3</span>'
```

Replace with functional button (remove `disabled`, remove "Coming in Phase 3" span):

```javascript
// If photo already captured for this fail:
var failPhoto = (FAIL_NOTES[fld.id] && FAIL_NOTES[fld.id].photo) ? FAIL_NOTES[fld.id].photo : null;
var photoHtml = failPhoto
  ? '<img class="photo-thumb" src="' + failPhoto + '" alt="Evidence photo"><button class="photo-retake-link" data-action="fail-photo-retake" data-fld-id="' + fld.id + '">Retake</button>'
  : '<button class="photo-stub-btn" data-action="fail-photo-capture" data-fld-id="' + fld.id + '">📷 Add photo</button>';
```

### `submitChecklist()` Approval Routing

Current (line 918–921):

```javascript
function submitChecklist(tplId) {
  SUBMITTED_TEMPLATES[tplId] = true;
  renderFillOut();
}
```

Replace with approval-aware routing:

```javascript
function submitChecklist(tplId) {
  const tpl = MOCK_TEMPLATES.find(t => t.id === tplId);
  if (tpl && tpl.requires_approval) {
    const allFields = tpl.sections.flatMap(s => s.fields);
    const answered = allFields.filter(f => MOCK_RESPONSES[f.id] !== undefined).length;
    const photoCount = allFields.filter(f => f.type === 'photo' && MOCK_RESPONSES[f.id]).length;
    const failCount = Object.keys(FAIL_NOTES).length;
    PENDING_APPROVALS[tplId] = {
      tplId, tplName: tpl.name,
      submittedBy: CURRENT_USER.name, submittedAt: new Date().toISOString(),
      totalFields: allFields.length, answeredCount: answered,
      failCount, photoCount,
      responses: Object.assign({}, MOCK_RESPONSES),
      failNotes: Object.assign({}, FAIL_NOTES)
    };
  } else {
    SUBMITTED_TEMPLATES[tplId] = true;
  }
  renderFillOut();
  if (!tpl || !tpl.requires_approval) fireworks();
}
```

### `show(n)` Extension

Current (2-tab version must be extended to 3):

```javascript
function show(n) {
  [1,2,3].forEach(function(i) {
    document.getElementById('s'+i).style.display = i===n?'':'none';
    var btn = document.getElementById('t'+i);
    if (btn) btn.classList.toggle('on', i===n);
  });
}
```

**Note:** `#s3` is the builder container. All builder-related code uses `document.getElementById('builder-body')` which is inside `#s3`. Since `builder-body` is the direct target of event delegation and rendering, renaming the outer container from `#s2` to `#s3` requires changing only the HTML wrapper and the `show()` function — no internal builder JS changes are needed.

### Approval Card Rendering

```javascript
function renderApprovals() {
  const body = document.getElementById('approvals-body');
  if (!body) return;
  const keys = Object.keys(PENDING_APPROVALS);
  if (keys.length === 0) {
    body.innerHTML = '<div class="empty-state"><h2>No pending approvals.</h2><p>All submitted checklists have been reviewed.</p></div>';
    return;
  }
  body.innerHTML = keys.map(function(tplId) {
    const entry = PENDING_APPROVALS[tplId];
    const pct = entry.totalFields ? Math.round(entry.answeredCount / entry.totalFields * 100) : 0;
    const photoStrip = entry.photoCount > 0
      ? '<div class="photo-strip">' + Object.values(entry.responses)
          .filter(r => r && r.value && typeof r.value === 'string' && r.value.startsWith('blob:'))
          .map(r => '<img class="photo-thumb" src="' + r.value + '" alt="Photo">').join('') + '</div>'
      : '';
    return '<div class="approval-card card" data-tpl-id="' + tplId + '">' +
      '<div class="hd"><h1>' + escapeHtml(entry.tplName) + '</h1>' +
      '<div class="sub">' + escapeHtml(entry.submittedBy) + ' · ' + formatTime(new Date(entry.submittedAt)) + '</div>' +
      '<div class="sub">' + entry.answeredCount + ' of ' + entry.totalFields + ' complete · ' + entry.failCount + ' fail trigger(s) · ' + entry.photoCount + ' photo(s)</div></div>' +
      photoStrip +
      '<div class="approval-actions" style="padding:12px 16px">' +
      '<button class="btn-approve" data-action="approve" data-tpl-id="' + tplId + '">Approve</button>' +
      '<button class="btn-reject" data-action="reject" data-tpl-id="' + tplId + '" style="margin-top:8px">Reject</button>' +
      '</div></div>';
  }).join('');
}
```

### Pre-Built Templates Data

Two templates replace or supplement current `MOCK_TEMPLATES`. Each must include:
- `requires_approval: true` on at least one to demonstrate the approval flow end-to-end
- At least one `photo` type field to demonstrate FILL-06
- At least one `yes_no` field to demonstrate fail triggers

```javascript
{
  id: 'tpl_setup',
  name: 'Setup Checklist',
  active_days: [1,2,3,4,5,6,0],
  requires_approval: true,
  sections: [
    {
      id: 'sec_setup_equip', title: 'Equipment', order: 0, condition: null,
      fields: [
        { id: 'fld_setup_power', type: 'yes_no', label: 'All equipment powered on?', required: true, order: 0, config: {}, fail_trigger: null, corrective_action: null, condition: null },
        { id: 'fld_setup_photo', type: 'photo', label: 'Setup station photo', required: false, order: 1, config: {}, fail_trigger: null, corrective_action: null, condition: null }
      ]
    },
    {
      id: 'sec_setup_temp', title: 'Temperature Check', order: 1, condition: null,
      fields: [
        { id: 'fld_setup_grill', type: 'temperature', label: 'Grill temperature', required: true, order: 0, config: { unit: '°F', min: 300, max: 500 }, fail_trigger: { type: 'out_of_range', min: 300, max: 500 }, corrective_action: { label: 'Action taken', type: 'text', required: true }, condition: null }
      ]
    }
  ]
},
{
  id: 'tpl_closing',
  name: 'Closing Checklist',
  active_days: [1,2,3,4,5,6,0],
  requires_approval: false,
  sections: [
    {
      id: 'sec_close_clean', title: 'Cleaning', order: 0, condition: null,
      fields: [
        { id: 'fld_close_surfaces', type: 'yes_no', label: 'All surfaces wiped down?', required: true, order: 0, config: {}, fail_trigger: null, corrective_action: null, condition: null },
        { id: 'fld_close_photo', type: 'photo', label: 'Clean station photo', required: false, order: 1, config: {}, fail_trigger: null, corrective_action: null, condition: null }
      ]
    },
    {
      id: 'sec_close_secure', title: 'Secure', order: 1, condition: null,
      fields: [
        { id: 'fld_close_locked', type: 'checkbox', label: 'Truck locked and secured', required: true, order: 0, config: {}, fail_trigger: null, corrective_action: null, condition: null }
      ]
    }
  ]
}
```

### Anti-Patterns to Avoid

- **Re-using the same `<input type="file">` node for re-capture:** Causes iOS stall bug (Pitfall 9). Always create fresh.
- **Attaching event listeners inside `renderFailCard()` or `renderRunnerField()`:** Causes accumulation bug (Pitfall 4). Route all photo-related actions through the existing `#fill-body` click delegation.
- **Reading photo URL from DOM `<img>.src`:** DOM is not state. Read from `MOCK_RESPONSES[fldId].value`.
- **Inline `onclick` on approval buttons:** Use `data-action` delegation pattern on `#approvals-body` click listener.
- **Gating tabs with CSS `display:none` only:** Use JS role check before rendering the tab button (Pitfall 13 pattern; consistent with existing builder gate).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| File-to-URL conversion | Custom base64 encoding / upload | `URL.createObjectURL(file)` | Browser-native, synchronous, zero encoding overhead, works offline |
| Camera access | `getUserMedia()` stream + canvas | `<input type="file" accept="image/*">` | Works in iOS standalone PWA without permissions; D-01 is locked |
| Drag-to-reorder | Custom touch drag | SortableJS (already loaded) | Already in page; no new CDN import needed |
| Modal overlay | Framework modal | DOM-appended `<div>` with fixed positioning | Zero deps; existing pattern in the codebase (fireworks canvas is appended same way) |

**Key insight:** Every photo-related operation uses browser-native APIs that require no library. The complexity is entirely in iOS workaround discipline, not in API surface.

---

## Common Pitfalls

### Pitfall A: iOS Single-Use `<input type="file">` Stall

**What goes wrong:** Using one `<input>` node for repeated captures. After the first use (or a dismissed picker), iOS WebKit does not fire `change` again on the same element.
**Why it happens:** iOS standalone PWA mode and Safari have a documented bug with `<input type="file" capture="environment">` reuse.
**How to avoid:** Create a new `<input>` element each time. Append to body, click, remove on `change`. See Architecture Patterns section for the exact function.
**Warning signs:** Photo capture works on first tap but not on "Retake".
**Phase note:** Confirmed by Pitfall 9 in project PITFALLS.md and UI-SPEC.md Interaction Contract.

### Pitfall B: Tab Renumbering Breaks Builder

**What goes wrong:** Builder section moves from `#s2` to `#s3`. If the `show()` function, HTML markup, or any JS that calls `show(2)` is not updated, the builder tab stops working.
**How to avoid:** Update in one atomic edit: HTML (`id="s3"` on builder wrapper), `show(n)` loop to iterate `[1,2,3]`, tab button `id="t3"`, and any call to `show(2)` for the builder.
**Warning signs:** Tapping Builder tab shows the approvals panel instead.

### Pitfall C: `URL.createObjectURL()` Memory Leak

**What goes wrong:** Every call to `URL.createObjectURL()` creates a reference held in browser memory until `URL.revokeObjectURL()` is called or the page is unloaded. In a long-running PWA session where a crew member takes many photos, this adds up.
**How to avoid:** For a mock with in-memory state only, this is acceptable. However: if a field's photo is replaced via Retake, call `URL.revokeObjectURL(oldUrl)` before overwriting `MOCK_RESPONSES[fldId].value`.
**Warning signs:** Memory usage growing continuously across multiple captures in one session.

### Pitfall D: Dangling `<input>` if Camera Dismissed

**What goes wrong:** User taps "Tap to take photo", camera opens, then user cancels without taking a photo. The `change` event never fires. The detached `<input>` element appended to `<body>` is never removed.
**How to avoid:** Add a cleanup timeout — if `change` has not fired within 30 seconds of `.click()`, remove the element from the DOM. Not a blocker for mock purposes but keeps the DOM clean.

### Pitfall E: Approval State Not Reflected on Checklist List

**What goes wrong:** A template with `requires_approval: true` is submitted. The crew member is returned to the checklist list view. The card still shows the green progress bar as if nothing happened — no "Pending Approval" badge visible.
**How to avoid:** In `renderChecklistList()`, check `PENDING_APPROVALS[tpl.id]` and render `.approval-badge` if present, in place of or below the progress bar.
**Warning signs:** Submitting a requires-approval checklist shows no visual confirmation in the list view.

### Pitfall F: Rejection Confirm with Empty Reason

**What goes wrong:** Manager taps "Reject", textarea appears, immediately taps "Confirm rejection" without entering a reason. An empty rejection is logged.
**How to avoid:** Guard: `if (!reason.trim()) return;` before moving entry to `REJECTED_SUBMISSIONS`. Optionally add a red border on the textarea if empty submit is attempted.

---

## Code Examples

### Opening Camera (Reusable Utility)

```javascript
// Source: project PITFALLS.md Pitfall 9 + UI-SPEC.md Interaction Contract
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

### Two-Step Photo Flow (Field Row)

```javascript
// Called from click delegation on #fill-body
function handlePhotoCaptureClick(fldId) {
  openCamera(function(file) {
    var objectURL = URL.createObjectURL(file);
    showPhotoPreview(objectURL,
      function confirm(url) {
        // Revoke old URL if retaking
        var old = MOCK_RESPONSES[fldId];
        if (old && old.value && old.value !== url) URL.revokeObjectURL(old.value);
        MOCK_RESPONSES[fldId] = { value: url, answeredBy: CURRENT_USER.name, answeredAt: new Date() };
        renderFillOut();
        updateProgress();
      },
      function retake() {
        URL.revokeObjectURL(objectURL);
        handlePhotoCaptureClick(fldId); // recurse for retake
      }
    );
  });
}
```

### Role-Based Tab Rendering

```javascript
// Source: existing builder tab gate pattern (MOCK_CURRENT_USER.permissions)
var isManagerPlus = MOCK_CURRENT_USER.role === 'admin' || MOCK_CURRENT_USER.role === 'manager';
var tabsHtml = '<button id="t1" class="on" onclick="show(1)">My Checklists</button>';
if (isManagerPlus) tabsHtml += '<button id="t2" onclick="show(2)">Approvals</button>';
tabsHtml += '<button id="t3" onclick="show(3)">Builder</button>';
```

**Note from UI-SPEC.md:** Tab ordering is My Checklists → Approvals → Builder. The Builder tab must remain at position 3 regardless of whether Approvals tab is visible. If manager role, `show(n)` must still handle n=3 even if n=2 renders nothing for non-managers. Simplest approach: always render all three section divs but conditionally render the tab buttons.

### Approval Approve Action (Click Delegation)

```javascript
// Inside #approvals-body click listener
if (action === 'approve') {
  var tplId = actionEl.dataset.tplId;
  var entry = PENDING_APPROVALS[tplId];
  if (!entry) return;
  APPROVED_SUBMISSIONS[tplId] = Object.assign({}, entry, {
    approvedBy: MOCK_CURRENT_USER.name, approvedAt: new Date().toISOString()
  });
  delete PENDING_APPROVALS[tplId];
  showToast('Approved \u2713');
  renderApprovals();
  return;
}
```

### Approval Reject Action (Two-Step Inline)

```javascript
if (action === 'reject') {
  var card = actionEl.closest('.approval-card');
  var actions = card.querySelector('.approval-actions');
  // Reveal textarea and confirm button inline
  if (!actions.querySelector('.rejection-reason')) {
    var ta = document.createElement('textarea');
    ta.className = 'rejection-reason fill-textarea';
    ta.rows = 3; ta.placeholder = 'Reason for rejection\u2026'; ta.style.marginTop = '8px';
    var confirmBtn = document.createElement('button');
    confirmBtn.className = 'btn-reject-confirm btn-primary';
    confirmBtn.style.cssText = 'background:#c0392b;color:#fff;margin-top:8px';
    confirmBtn.textContent = 'Confirm rejection';
    confirmBtn.dataset.action = 'reject-confirm';
    confirmBtn.dataset.tplId = actionEl.dataset.tplId;
    actions.appendChild(ta); actions.appendChild(confirmBtn);
  }
  return;
}

if (action === 'reject-confirm') {
  var tplId = actionEl.dataset.tplId;
  var card = actionEl.closest('.approval-card');
  var reason = card.querySelector('.rejection-reason').value.trim();
  if (!reason) return;
  var entry = PENDING_APPROVALS[tplId];
  REJECTED_SUBMISSIONS[tplId] = Object.assign({}, entry, {
    rejectedBy: MOCK_CURRENT_USER.name, rejectedAt: new Date().toISOString(), reason: reason
  });
  delete PENDING_APPROVALS[tplId];
  showToast('Rejected');
  renderApprovals();
  return;
}
```

---

## Integration Verification Checklist

### INTG-01: Operations Tile

Verified from codebase read (2026-04-13):

```html
<!-- index.html line 73-77 — already present and correct -->
<a class="tile active" href="workflows.html">
  <div class="tile-icon">📋</div>
  <div class="tile-title">Operations</div>
  <div class="tile-desc">Checklists and workflows</div>
</a>
```

No action required. Verification only.

### sw.js Cache Bump

Current: `const CACHE = 'yumyums-v19'`
`workflows.html` already in ASSETS list.

**Required action:** Bump to `yumyums-v20` (or next available) after any code changes in this phase. Per project MEMORY.md: "Always bump sw.js version + push before human-verify checkpoints."

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `<input capture="environment">` reused | Fresh `<input>` created per capture | iOS 12+ bug | Required for re-capture to work on iOS PWA |
| `FileReader.readAsDataURL()` | `URL.createObjectURL()` | ~2017 (broad support) | Faster, no encoding overhead, suitable for in-memory display |
| 2-tab `show(n)` with `[1,2]` | 3-tab `show(n)` with `[1,2,3]` | Phase 3 | Builder shifts from `#s2` to `#s3` |

---

## Open Questions

1. **`show(n)` when Approvals tab is hidden for crew role**
   - What we know: Tab button `t2` is not rendered for crew. But `#s2` div still exists in HTML.
   - What's unclear: If crew member somehow calls `show(2)` (e.g., hash navigation), will layout break?
   - Recommendation: Guard `show(n)` — if `n === 2` and user is not manager+, default to `show(1)`. Or: always render all three `#s1`/`#s2`/`#s3` divs; only conditionally render tab buttons.

2. **Object URL lifespan across re-renders**
   - What we know: `renderFillOut()` rebuilds innerHTML, replacing `<img>` nodes. The `src` attribute with the blob URL still points to a valid object in memory.
   - What's unclear: No explicit revocation on re-render. This is fine for the mock but could accumulate across many retakes.
   - Recommendation: Revoke old URL on retake (already covered in code example above). No other action needed for mock scope.

3. **`MOCK_TEMPLATES` — replace or supplement existing templates?**
   - What we know: Current `MOCK_TEMPLATES` includes "Morning Opening Checklist" and a partial "Closing Checklist". D-08/D-09 says add "Setup Checklist" and "Closing Checklist".
   - What's unclear: Whether to keep the existing templates alongside the new ones.
   - Recommendation (Claude's Discretion): Replace both existing templates with the two new ones from D-08. The new templates include photo fields and the approval routing flag, making them more useful for demonstrating all Phase 3 features. Keeping 4 templates adds noise without adding coverage.

---

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — all changes are static HTML/CSS/JS with no new CDN scripts, no build tooling, no external services)

---

## Sources

### Primary (HIGH confidence)

- `workflows.html` (current implementation) — direct code read, lines 1–1200+
- `index.html` — INTG-01 tile verification
- `sw.js` — cache version and ASSETS list verification
- `.planning/research/PITFALLS.md` — Pitfall 9 (iOS single-use input bug), Pitfall 4 (event delegation), Pitfall 13 (CSS-only gating)
- `.planning/phases/03-photo-approval-and-integration/03-CONTEXT.md` — all D-01 through D-11 decisions
- `.planning/phases/03-photo-approval-and-integration/03-UI-SPEC.md` — component inventory, interaction contracts, copywriting contract
- `.planning/REQUIREMENTS.md` — FILL-06, APRV-01, APRV-02, APRV-03, INTG-01, INTG-02, INTG-03

### Secondary (MEDIUM confidence)

- web.dev: Capturing images from the user — `<input type="file">` patterns (cited in PITFALLS.md sources)
- PWA-POLICE GitHub issue 12 — iOS 12.2+ `input type file` camera bug (cited in PITFALLS.md sources)

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new libraries; all browser-native APIs
- Architecture: HIGH — derived directly from existing code structure and UI-SPEC
- Pitfalls: HIGH — iOS bug confirmed in project pitfall research; other pitfalls derived from existing codebase patterns
- Photo capture: HIGH — established pattern; browser API well-documented

**Research date:** 2026-04-13
**Valid until:** 2026-05-13 (stable APIs; iOS PWA behavior unlikely to change in 30 days)
