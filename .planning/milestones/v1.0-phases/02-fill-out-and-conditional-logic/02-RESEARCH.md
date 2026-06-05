# Phase 2: Fill-Out and Conditional Logic - Research

**Researched:** 2026-04-13
**Domain:** Vanilla JS checklist fill-out engine, condition evaluation, fail triggers, mobile touch UX
**Confidence:** HIGH — primary sources are the existing codebase (workflows.html), prior project-level research docs, and the Phase 2 UI-SPEC already approved upstream.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Tap the whole row to toggle checkbox/yes-no items. Large touch target for kitchen use.
- **D-02:** Completed items show strikethrough text with a green checkmark to the right.
- **D-03:** Text and temperature fields use inline inputs directly in the row — tap to focus, type, move on. No modal or expand pattern.
- **D-04:** Out-of-range temperature or "No" answer triggers an inline slide-down card below the field. Red-tinted background. Doesn't block other items.
- **D-05:** Corrective action form includes three parts: text note ("Describe what you did"), photo capture stub (placeholder for Phase 3), and severity picker (minor / major / critical).
- **D-06:** Completed items show initials badge + time (e.g., "JM · 2:15p") to the right of the checkmark. Compact for mobile.
- **D-07:** Hardcoded current user: `CURRENT_USER = {name: 'Jamal M.', initials: 'JM'}`. Every checked item gets stamped with this user.
- **D-08:** Card per checklist on the list view — shows name, section count, and progress bar. Tap to open.
- **D-09:** Empty state when no checklists active today: "No checklists for today. Enjoy your day off!" with current day name.
- **D-10:** Progress shown in both places: progress bar on list card + "X of Y items complete" counter inside the checklist at top.
- **D-11:** Fields and sections hidden by skip logic or day-of-week conditions are not rendered. Hidden field answers are cleared from response state.

### Claude's Discretion

- Exact progress bar styling and placement
- How sections render in fill-out mode (cards vs flat headers — can differ from builder)
- Submit button behavior (mock only — show confirmation)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FILL-01 | Crew member can see a list of available workflow checklists to complete | Day-filter on MOCK_TEMPLATES via `active_days.includes(today)` + checklist card render with progress bar |
| FILL-02 | Crew member can open a checklist and see items grouped by sections | Runner view renders sections in order, `.cat` header per section; field list per section |
| FILL-03 | Crew member can check off checkbox / yes-no items with large touch targets | Full-row tap on `.fill-row` (min-height 44px); toggle answer in response state; re-render row |
| FILL-04 | Crew member can enter text notes on any item | Inline `<input type="text">` in `.fill-row.text-row`; `input` event updates response state without re-render (no cursor jump) |
| FILL-05 | Crew member can enter temperature readings with unit display (°F) | Inline `<input type="number">` + `°F` suffix; `input` event evaluates fail trigger; fail card shown/hidden reactively |
| FILL-07 | Each completed item shows who checked it (user attribution) | CURRENT_USER hardcoded per D-07; `.attr-badge` "JM · 2:15p" rendered when `completedItems[fieldId]` is truthy |
| FILL-08 | Crew member can see completion progress | Progress counter "X of Y items complete" at top of runner; progress bar on list card; denominator is visible fields only (D-11) |
| COND-01 | When a temperature reading is out of range, inline corrective action prompt appears | `evaluateFailTrigger(fail_trigger, value)` pure function; `.fail-card` rendered below field; amber out-of-range label above |
| COND-02 | When an item is answered "No", inline corrective action prompt appears | Same `.fail-card` pattern; triggered on yes_no answer = false |
| COND-03 | Sections/items can be configured to appear only on certain days | `section.condition.days` and `field.condition.days` evaluated against `new Date().getDay()` at render time |
| COND-04 | Items can be conditionally shown/hidden based on a prior answer (skip logic) | `evaluateCondition(condition, completedItems)` pure function; hidden fields removed from DOM + cleared from state |

</phase_requirements>

---

## Summary

Phase 2 builds the crew fill-out experience entirely within `#s1` — the tab that has existed as a stub since Phase 1. The implementation follows the same vanilla JS state-first pattern established in Phase 1: mutate JS state, call render, DOM updates. No new libraries are required.

The data model is already fully defined in MOCK_TEMPLATES (Phase 1 output). Phase 2 only adds a response state structure (`state.responses`, `state.fillView`) alongside the existing builder state. Condition evaluation is a set of pure functions already designed in ARCHITECTURE.md — the fill-out view calls them; the builder UI stored the definitions.

The most technically complex interactions are the fail trigger / corrective action card (inline slide-down, reactive to temperature input), the condition evaluation loop (clearing hidden field answers on every state mutation), and progress tracking that correctly excludes hidden fields from the denominator.

**Primary recommendation:** Build in the order `list view → runner shell → field types (checkbox first, then yes_no, text, temperature) → fail triggers → conditions`. Each layer is testable before the next. The UI-SPEC is fully approved and prescribes exact class names, copy, and interaction contracts — implement directly against it.

---

## Standard Stack

### Core (No Build Step — unchanged from Phase 1)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Vanilla JS (ES2020+) | native | All interactivity and state | Established project convention; no framework |
| CSS custom properties | native | Theming and dark mode | Already in every page via `:root` block; Phase 2 adds zero new tokens |
| SortableJS | 1.15.7 (unpkg) | Drag-to-reorder in builder (Phase 1 carry-forward) | Already loaded in `<head>` — not used in fill-out, but present on page |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| ptr.js (local) | local | Pull-to-refresh | Already included via `<script src="ptr.js">` at bottom of page |

### What NOT to Add

| Rejected | Reason |
|----------|--------|
| JsonLogic CDN | Condition evaluation for this domain is simple enough for a hand-written switch statement (per ARCHITECTURE.md). Condition schema uses `equals`, `not_equals`, `lt`, `gt`, `in_range` — no library needed. |
| Any animation library | Instant state change + re-render is the established pattern (no CSS transitions for show/hide). |
| Any new CDN script | Phase 2 adds zero new third-party dependencies (confirmed by 02-UI-SPEC.md Registry Safety section). |

**No installation step required.** All dependencies are already present on the page from Phase 1.

---

## Architecture Patterns

### Recommended State Structure

The fill-out engine adds a `fillState` object alongside the existing builder `state` object:

```javascript
// Fill-out state — separate from builder state
const CURRENT_USER = { name: 'Jamal M.', initials: 'JM' };

let fillState = {
  fillView: 'list',         // 'list' | 'runner'
  activeTemplateId: null,   // which template is open in runner
  responses: {}             // keyed by templateId; each holds completedItems + failNotes
};

// Response shape per template
// fillState.responses['tpl_morning_open'] = {
//   templateId: 'tpl_morning_open',
//   startedAt: '2026-04-13T09:00:00.000Z',
//   completedItems: {
//     'fld_grill_clean': {
//       value: true,
//       answeredBy: CURRENT_USER,
//       answeredAt: '2026-04-13T09:03:00.000Z'
//     }
//   },
//   failNotes: {
//     'fld_grill_temp': { note: 'Called repair', severity: 'major' }
//   },
//   submitted: false
// }
```

### Recommended Project Structure (additions to workflows.html `<script>` block)

```
workflows.html <script>
  // === EXISTING (Phase 1 — do not modify) ===
  MOCK_CURRENT_USER, MOCK_TEMPLATES, state, renderBuilder(), ...

  // === NEW: FILL-OUT CONSTANTS ===
  CURRENT_USER             — { name: 'Jamal M.', initials: 'JM' }

  // === NEW: FILL-OUT STATE ===
  fillState                — { fillView, activeTemplateId, responses }

  // === NEW: PURE FUNCTIONS (no DOM, no state side effects) ===
  isDayActive(template)    — true if template.active_days includes today
  isSectionVisible(sec)    — true if section has no day condition or days include today
  isFieldVisible(fld, completedItems)  — evaluates field.condition against completedItems
  evaluateCondition(cond, completedItems) → boolean
  evaluateFailTrigger(fail_trigger, value) → boolean
  getVisibleFields(template, completedItems) → flat array of visible field objects
  computeProgress(template, completedItems) → { total, completed }
  formatTime(isoString)    — "2:15p" from ISO timestamp

  // === NEW: FILL-OUT RENDER FUNCTIONS ===
  renderFillout()          — renders into #s1; switches between list and runner
  renderFillList()         — today's checklist cards
  renderChecklistCard(tpl) — one card with progress bar
  renderRunner(tpl)        — full runner view for one template
  renderFillSection(sec, completedItems, failNotes) — section header + visible fields
  renderFillField(fld, answer, failNote) — one field row by type
  renderFailCard(fld, failNote) — inline corrective action card

  // === NEW: FILL-OUT MUTATIONS ===
  openChecklist(tplId)     — set fillView='runner', create/resume response
  backToFillList()         — set fillView='list', keep response state
  toggleCheckItem(tplId, fldId)  — toggle completedItems entry
  setYesNo(tplId, fldId, value)  — set yes_no answer, trigger fail card if false
  setTextAnswer(tplId, fldId, value) — update text/temp answer in state (no re-render)
  setFailNote(tplId, fldId, patch)   — update failNotes entry
  setSeverity(tplId, fldId, severity) — update severity in failNotes
  submitChecklist(tplId)   — mark submitted, show confirmation

  // === NEW: FILL-OUT EVENT DELEGATION (attached ONCE at boot) ===
  #s1 click listener       — routes by data-action attribute
  #s1 input listener       — handles text/temperature inline inputs

  // === EXISTING (Phase 1 — do not modify) ===
  show(n), checkBuilderAccess(), renderBuilder(), initSortable()
```

### Pattern 1: State-First Render (same as Phase 1, extended to fill-out)

Every mutation writes to `fillState` first, then calls `renderFillout()`. The DOM is never read inside business logic.

```javascript
// Source: established pattern from purchasing.html + Pitfall 1 prevention
function toggleCheckItem(tplId, fldId) {
  const resp = fillState.responses[tplId];
  if (resp.completedItems[fldId]) {
    delete resp.completedItems[fldId];
  } else {
    resp.completedItems[fldId] = {
      value: true,
      answeredBy: CURRENT_USER,
      answeredAt: new Date().toISOString()
    };
  }
  renderFillout();
}
```

### Pattern 2: Event Delegation on #s1

ONE click listener and ONE input listener on `#s1`, attached at boot. Routes via `data-action` attributes.

```javascript
// Source: Pitfall 4 prevention + 02-UI-SPEC.md Interaction Contract
document.getElementById('s1').addEventListener('click', function(e) {
  const action = e.target.closest('[data-action]')?.dataset.action;
  if (!action) return;

  const tplId = e.target.closest('[data-tpl-id]')?.dataset.tplId;
  const fldId = e.target.closest('[data-fld-id]')?.dataset.fldId;

  if (action === 'open-checklist')      { openChecklist(tplId); return; }
  if (action === 'back-to-list')        { backToFillList(); return; }
  if (action === 'toggle-check')        { toggleCheckItem(tplId, fldId); return; }
  if (action === 'set-yes')             { setYesNo(tplId, fldId, true); return; }
  if (action === 'set-no')              { setYesNo(tplId, fldId, false); return; }
  if (action === 'set-severity')        {
    const sev = e.target.closest('[data-severity]')?.dataset.severity;
    setSeverity(tplId, fldId, sev);
    return;
  }
  if (action === 'submit')              { submitChecklist(tplId); return; }
});

document.getElementById('s1').addEventListener('input', function(e) {
  const action = e.target.dataset.action;
  const tplId = e.target.dataset.tplId;
  const fldId = e.target.dataset.fldId;
  if (!tplId || !fldId) return;

  if (action === 'text-input') {
    setTextAnswer(tplId, fldId, e.target.value);
    return;
  }
  if (action === 'temp-input') {
    setTextAnswer(tplId, fldId, e.target.value);
    // Re-render only the fail trigger state — but full re-render is acceptable
    // for temperature because blur/input is less frequent than checkbox taps.
    renderFillout();
    return;
  }
  if (action === 'fail-note-input') {
    setFailNote(tplId, fldId, { note: e.target.value });
    return;
  }
});
```

**Important:** Text inputs (`action === 'text-input'`) update state but do NOT call `renderFillout()` — this prevents cursor jump (same pattern established in Phase 1 for label inputs in the builder). Temperature inputs DO call `renderFillout()` after update because we need the fail card to appear/disappear reactively.

### Pattern 3: Condition Evaluation (pure functions)

```javascript
// Source: .planning/research/ARCHITECTURE.md — Condition Evaluation Logic

function evaluateCondition(condition, completedItems) {
  if (!condition) return true;  // no condition = always show

  // Day-of-week condition
  if (condition.days) {
    const today = new Date().getDay();  // 0=Sun … 6=Sat
    if (!condition.days.includes(today)) return false;
  }

  // Skip logic condition
  if (condition.field_id) {
    const answer = completedItems[condition.field_id];
    if (!answer) return false;  // referenced field not answered = hide
    const val = answer.value;
    switch (condition.operator) {
      case 'equals':     return val === condition.value;
      case 'not_equals': return val !== condition.value;
      case 'lt':         return val < condition.value;
      case 'gt':         return val > condition.value;
      case 'in_range':   return val >= condition.min && val <= condition.max;
      default:           return true;
    }
  }

  return true;
}

function isSectionVisible(sec) {
  if (!sec.condition) return true;
  if (!sec.condition.days) return true;
  const today = new Date().getDay();
  return sec.condition.days.includes(today);
}

function isFieldVisible(fld, completedItems) {
  return evaluateCondition(fld.condition, completedItems);
}

function evaluateFailTrigger(fail_trigger, value) {
  if (!fail_trigger || value === '' || value === null || value === undefined) return false;
  if (fail_trigger.type === 'out_of_range') {
    const num = Number(value);
    return num < fail_trigger.min || num > fail_trigger.max;
  }
  if (fail_trigger.type === 'equals') return value === fail_trigger.value;
  if (fail_trigger.type === 'not_equals') return value !== fail_trigger.value;
  return false;
}
```

### Pattern 4: Hidden Field Answer Clearing (D-11)

When a field transitions from visible to hidden, its answer must be removed from `completedItems`. This is done inside `renderFillout()` before rendering, as a pre-pass:

```javascript
function clearHiddenFieldAnswers(tplId) {
  const tpl = MOCK_TEMPLATES.find(t => t.id === tplId);
  const resp = fillState.responses[tplId];
  if (!tpl || !resp) return;

  for (const sec of tpl.sections) {
    if (!isSectionVisible(sec)) {
      // Clear ALL fields in a hidden section
      for (const fld of sec.fields) {
        delete resp.completedItems[fld.id];
        delete resp.failNotes[fld.id];
      }
      continue;
    }
    for (const fld of sec.fields) {
      if (!isFieldVisible(fld, resp.completedItems)) {
        delete resp.completedItems[fld.id];
        delete resp.failNotes[fld.id];
      }
    }
  }
}

function renderFillout() {
  const el = document.getElementById('s1');
  if (!el) return;
  if (fillState.fillView === 'runner' && fillState.activeTemplateId) {
    clearHiddenFieldAnswers(fillState.activeTemplateId);  // MUST run before render
  }
  el.innerHTML = fillState.fillView === 'list'
    ? renderFillList()
    : renderRunner(MOCK_TEMPLATES.find(t => t.id === fillState.activeTemplateId));
}
```

### Pattern 5: Progress Computation

Progress denominator is visible fields only (not hidden, not photo fields which are skipped in Phase 2):

```javascript
function computeProgress(template, completedItems) {
  let total = 0;
  let completed = 0;
  for (const sec of template.sections) {
    if (!isSectionVisible(sec)) continue;
    for (const fld of sec.fields) {
      if (fld.type === 'photo') continue;  // photo skipped in Phase 2
      if (!isFieldVisible(fld, completedItems)) continue;
      total++;
      if (completedItems[fld.id]) completed++;
    }
  }
  return { total, completed };
}
```

### Pattern 6: Attribution Badge Formatting

```javascript
function formatTime(isoString) {
  const d = new Date(isoString);
  let h = d.getHours();
  const m = String(d.getMinutes()).padStart(2, '0');
  const ampm = h >= 12 ? 'p' : 'a';
  h = h % 12 || 12;
  return `${h}:${m}${ampm}`;  // e.g. "2:15p"
}

// Used in renderFillField() for completed items:
// `<span class="attr-badge">${CURRENT_USER.initials} · ${formatTime(answer.answeredAt)}</span>`
```

### Pattern 7: Day Name for Empty State

```javascript
const DAY_NAMES = ['Sunday','Monday','Tuesday','Wednesday','Thursday','Friday','Saturday'];
// Empty state: "Enjoy your day off, " + DAY_NAMES[new Date().getDay()] + "!"
```

### Anti-Patterns to Avoid

- **Re-reading DOM for answer values:** Never `document.querySelector('[data-fld-id]').value` inside condition evaluators or progress computors. State owns all values.
- **Text input re-render causing cursor jump:** For `text` field types, update state on `input` but do NOT call `renderFillout()`. The DOM input is live — only the surrounding row context (attribution, strikethrough) needs re-render, and that happens naturally on the next unrelated state change. Exception: temperature inputs must trigger re-render to show/hide fail cards.
- **Fail card as modal or overlay:** Fail card is always an inline DOM element below the triggering field row — never `position: fixed`, never `display: block` overlay covering the checklist (Pitfall 6).
- **Photo fields in Phase 2:** `type === 'photo'` fields are skipped entirely in the runner render — not rendered, not counted in progress. (Phase 3 responsibility.)
- **Condition evaluation in the wrong layer:** Day-of-week evaluation happens at fill-out render time using `new Date().getDay()`. Never mutate MOCK_TEMPLATES based on today's date (Pitfall 7).
- **While loop in condition evaluator:** Use a linear top-to-bottom pass. The Phase 1 builder already enforces that conditions can only reference preceding fields, so a single pass evaluates correctly (Pitfall 2).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Condition evaluation rule engine | Custom eval() or Function() string eval | `evaluateCondition()` switch statement (from ARCHITECTURE.md) | Covers all 5 operators needed; eval adds security surface area for zero benefit |
| Time formatting library | moment.js / date-fns | `formatTime()` pure function (2-line implementation) | Only need "H:MMa/p" — no library justified |
| Animation for fail card slide-down | CSS transitions on dynamic elements | Instant render (no animation) | Matches codebase convention; animation on dynamically inserted elements is non-trivial in vanilla JS |
| Scroll-to-field logic | Custom IntersectionObserver or scroll tracking | `el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })` | Native API; single call after fail card renders |

**Key insight:** The fill-out engine has no new library dependencies. All complexity is in the state management pattern and the condition evaluation logic — both of which are already fully designed in upstream research docs.

---

## Common Pitfalls

### Pitfall 1 (CRITICAL): DOM as Source of Truth in Fill-Out

**What goes wrong:** Reading `<input>` values from the DOM inside `computeProgress()` or `evaluateCondition()` — e.g., `document.querySelector('[data-fld-id="X"]').value`.

**Why it happens:** Text/temperature inputs are inline in the DOM and it feels natural to read them directly.

**How to avoid:** All `input` events update `fillState.responses[tplId].completedItems[fldId]` immediately. `evaluateCondition()` and `computeProgress()` only read from `fillState`, never the DOM.

**Warning signs:** Any `document.querySelector` call inside a pure function.

### Pitfall 2 (CRITICAL): Cursor Jump on Text Input Re-render

**What goes wrong:** On every keystroke in a text field, calling `renderFillout()` replaces the input's DOM node, which resets cursor position to the end of the string.

**Why it happens:** Full re-render is the correct pattern for checkboxes/toggles. It's wrong for live text input.

**How to avoid:** For `text` field type `input` events: update `fillState` but do NOT call `renderFillout()`. Only call `renderFillout()` for discrete state changes (checkbox tap, yes/no tap, severity pick, submit). Temperature is the exception: re-render IS needed to show/hide the fail card, but the tradeoff is acceptable because temp entry is fewer keystrokes.

**Warning signs:** Cursor jumping to end of text field after each character.

### Pitfall 3: Hidden Field Answers Persisting in State

**What goes wrong:** A field is hidden by skip logic but its previous answer value is still in `completedItems`. This inflates the completed count, and if the hidden field's answer would have triggered a fail, that fail note persists invisibly.

**How to avoid:** `clearHiddenFieldAnswers()` runs inside `renderFillout()` before any HTML is generated. When a field transitions from visible to hidden, its `completedItems` and `failNotes` entries are deleted.

**Warning signs:** Progress counter shows "5 of 4 items complete" — impossible if hidden fields are properly excluded.

### Pitfall 4: Progress Denominator Includes Hidden/Photo Fields

**What goes wrong:** `computeProgress()` counts all fields in all sections. Skipped fields (photo in Phase 2, day-hidden, skip-logic-hidden) inflate the denominator, making the checklist appear less complete than it is.

**How to avoid:** `computeProgress()` iterates sections and fields through both `isSectionVisible()` and `isFieldVisible()` gates, AND skips `type === 'photo'` explicitly.

**Warning signs:** Checklist shows "2 of 5 complete" when only 3 fields are actually visible.

### Pitfall 5: iOS Keyboard / Viewport on Text/Temperature Fields (Pitfall 5 from PITFALLS.md)

**What goes wrong:** In standalone PWA mode on iOS, the virtual keyboard pushes the layout up without resizing the viewport. Fields with `position: fixed` bottom bars overlap the keyboard.

**How to avoid:**
- Phase 2 has no fixed-position bottom bar (submit button is in-flow at page bottom, not fixed).
- After temperature input focus: `e.target.scrollIntoView({ behavior: 'smooth', block: 'nearest' })`.
- Never use `100vh` for full-screen layout — the body already uses `env(safe-area-inset-bottom)` from Phase 1.

**Warning signs:** Submit button or corrective action card hidden behind keyboard on real iPhone in standalone mode.

### Pitfall 6: Yes/No Field Rendering (Full-Row Tap vs Pill Tap Conflict)

**What goes wrong:** Yes/no fields use full-row tap (D-01) for toggle AND show "Yes" / "No" pills for explicit choice. If both trigger on the same tap, the answer toggles AND the pill click fires — causing double-mutation.

**How to avoid:** Yes/no rows do NOT use the full-row toggle pattern. Instead, only the "Yes" and "No" pill buttons have `data-action`. The row itself has no `data-action`. The row IS still `min-height: 44px` for visual consistency, but the tap target is limited to the pills.

From the UI-SPEC: "yes_no: Same as checkbox row. Below the row: two tappable answer pills 'Yes' / 'No'" — meaning the pills are the tap targets, not the row.

**Warning signs:** Tapping "No" pill immediately toggles back to uncompleted state.

### Pitfall 7: Fail Card Appearing Before User Has Entered Anything

**What goes wrong:** Temperature fail trigger fires on empty input (value = "", which is < 350) and shows the fail card immediately when the user taps on the field before typing anything.

**How to avoid:** `evaluateFailTrigger()` returns false when value is empty string, null, or undefined — check this guard first (already in the Pattern 3 code above).

**Warning signs:** Corrective action card appears when clicking a temperature field before typing.

---

## Code Examples

### Checklist List Card Render

```javascript
// Source: 02-UI-SPEC.md Component Inventory + Page Layout Contract
function renderChecklistCard(tpl) {
  const resp = fillState.responses[tpl.id] || { completedItems: {} };
  const { total, completed } = computeProgress(tpl, resp.completedItems);
  const pct = total > 0 ? Math.round(completed / total * 100) : 0;
  const secLabel = tpl.sections.length === 1 ? '1 section' : tpl.sections.length + ' sections';
  const approvalBadge = tpl.requires_approval
    ? `<span style="font-size:11px;padding:2px 8px;border-radius:8px;background:var(--warn-bg);color:var(--warn-tx);margin-left:6px">Requires approval</span>`
    : '';
  return `
    <div class="checklist-card card" data-action="open-checklist" data-tpl-id="${tpl.id}" style="cursor:pointer;margin-bottom:8px">
      <div class="hd">
        <div style="font-size:15px;font-weight:500">${escapeHtml(tpl.name)}</div>
        <div class="sub">${secLabel}${approvalBadge}</div>
      </div>
      <div class="fill-progress-bar" style="height:4px;background:var(--brd);margin:0 16px 12px">
        <div class="fill-progress-bar-fill" style="height:100%;background:var(--info-tx);width:${pct}%;transition:width 0.2s"></div>
      </div>
    </div>`;
}
```

### Checkbox Field Row Render

```javascript
// Source: 02-UI-SPEC.md — Checkbox / Yes-No Completion interaction contract
function renderCheckboxRow(fld, tplId, answer) {
  const done = !!answer;
  const labelStyle = done
    ? 'text-decoration:line-through;color:var(--mut)'
    : '';
  const right = done
    ? `<span style="color:var(--txt);font-size:16px">✓</span>
       <span class="attr-badge" style="font-size:12px;color:var(--mut);margin-left:6px">${escapeHtml(answer.answeredBy.initials)} · ${formatTime(answer.answeredAt)}</span>`
    : `<span style="color:var(--mut);font-size:16px">○</span>`;
  return `
    <div class="fill-row" data-action="toggle-check" data-tpl-id="${tplId}" data-fld-id="${fld.id}"
         style="display:flex;align-items:center;justify-content:space-between;padding:12px 16px;min-height:44px;border-bottom:0.5px solid var(--brd);cursor:pointer">
      <span style="font-size:14px;${labelStyle}">${escapeHtml(fld.label)}</span>
      <span style="display:flex;align-items:center">${right}</span>
    </div>`;
}
```

### Fail Card Render

```javascript
// Source: 02-UI-SPEC.md — Corrective Action Card + Specifics from CONTEXT.md
function renderFailCard(fld, tplId, failNote) {
  const note = (failNote && failNote.note) || '';
  const severity = (failNote && failNote.severity) || '';
  const severities = ['minor', 'major', 'critical'];
  const pillsHtml = severities.map(s => {
    const isOn = severity === s;
    const bg = isOn ? 'var(--info-bg)' : 'transparent';
    const color = isOn ? 'var(--info-tx)' : 'var(--txt)';
    return `<button class="severity-pill" data-action="set-severity" data-severity="${s}" data-tpl-id="${tplId}" data-fld-id="${fld.id}"
              style="padding:8px 16px;border-radius:8px;border:0.5px solid var(--brd);background:${bg};color:${color};font-size:12px;font-weight:500;cursor:pointer">${s.charAt(0).toUpperCase()+s.slice(1)}</button>`;
  }).join('');
  return `
    <div class="fail-card" style="background:rgba(192,57,43,0.08);border-radius:8px;padding:12px 16px;margin:4px 16px 8px">
      <div style="font-size:12px;font-weight:500;color:#c0392b;margin-bottom:10px">Corrective action required</div>
      <textarea data-action="fail-note-input" data-tpl-id="${tplId}" data-fld-id="${fld.id}"
                placeholder="Describe what you did…" rows="3"
                style="width:100%;padding:10px 12px;background:var(--bg);border:0.5px solid var(--brd);border-radius:8px;font-size:14px;color:var(--txt);font-family:inherit;resize:none;outline:none">${escapeHtml(note)}</textarea>
      <div class="photo-stub-row" style="display:flex;align-items:center;gap:8px;margin:10px 0">
        <button class="photo-stub-btn" disabled style="padding:8px 12px;background:var(--bg);border:0.5px solid var(--brd);border-radius:8px;font-size:14px;opacity:0.6;cursor:not-allowed">📷 Add photo</button>
        <span style="font-size:12px;color:var(--mut)">Coming in Phase 3</span>
      </div>
      <div class="severity-pills" style="display:flex;gap:8px;flex-wrap:wrap">${pillsHtml}</div>
    </div>`;
}
```

### Day Filter on List View

```javascript
// Source: 02-UI-SPEC.md Checklist List View interaction contract
function renderFillList() {
  const today = new Date().getDay();  // 0=Sun … 6=Sat
  const active = MOCK_TEMPLATES.filter(t =>
    !t.active_days || t.active_days.length === 0 || t.active_days.includes(today)
  );
  if (active.length === 0) {
    const dayName = DAY_NAMES[today];
    return `<div class="empty-state" style="text-align:center;padding:32px 16px;color:var(--mut)">
      <h2 style="font-size:15px;font-weight:500;color:var(--txt);margin:0 0 8px">No checklists for today.</h2>
      <p style="font-size:14px;margin:0">Enjoy your day off, ${dayName}!</p>
    </div>`;
  }
  const cardsHtml = active.map(t => renderChecklistCard(t)).join('');
  return `<div class="cat">Today's Checklists</div>${cardsHtml}`;
}
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|-----------------|--------|
| Modal dialogs for corrective actions | Inline `.fail-card` below triggering field | Mobile-first, non-blocking, crew stays oriented |
| Form reads from DOM | State-first: all answers in `fillState.responses` | Enables condition evaluation without DOM queries |
| One render triggers per-element listeners | Event delegation on `#s1` container | No listener leaks across re-renders |

---

## Open Questions

1. **Yes/no row: full-row tap vs pills only**
   - What we know: D-01 says "tap whole row to toggle checkbox/yes-no". UI-SPEC says yes_no shows "Yes" / "No" pills.
   - The conflict: If the whole row is tappable, a tap on the "No" pill would first toggle via the row handler, then fire the pill handler — double mutation.
   - Recommendation: For `yes_no` fields specifically, the full-row tap is suppressed. Only the "Yes" and "No" pills have `data-action`. The row remains visually at 44px min-height but has no click action itself. This interpretation is consistent with "tap whole row" meaning the row IS the target (which the pills are part of) while avoiding double-fire. Treat pills as the complete tap surface for yes_no; treat the whole row as the tap surface for checkbox.

2. **Temperature input — re-render on every keystroke?**
   - What we know: Fail trigger must show/hide reactively. Text inputs must not re-render (cursor jump).
   - Recommendation: Re-render on `input` event for temperature (acceptable — shorter strings, numeric, less cursor sensitivity). This is already reflected in Pattern 2 above. If cursor issues arise in practice, alternative: update only the `temp-warn` span and `fail-card` via targeted DOM update instead of full re-render — but optimize only if needed.

3. **Response persistence across page reload**
   - What we know: Phase 1 established no localStorage (D-11 analogue). STATE.md confirms no persistence for mocks.
   - Recommendation: `fillState.responses` is in-memory only. Page refresh clears all in-progress fill-out. This is acceptable for mock phase. Document this expectation clearly with a comment in the code.

---

## Environment Availability

Step 2.6: SKIPPED (no new external dependencies in Phase 2 — all required assets are already loaded by Phase 1 HTML setup).

Existing Phase 1 verified dependencies (carry-forward):

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| SortableJS (unpkg) | Builder drag (Phase 1 carry-forward) | Network-dependent | 1.15.7 | cdnjs 1.15.6 |
| ptr.js (local) | Pull-to-refresh | Yes (in repo) | local | Omit if missing |

---

## Sources

### Primary (HIGH confidence)

- `/Users/jamal/projects/yumyums/hq/workflows.html` — Phase 1 output: MOCK_TEMPLATES data model, CURRENT_USER shape, CSS variables, builder state pattern, event delegation structure, `escapeHtml()` / `escapeAttr()` utilities, `DAY_INDICES` mapping
- `/Users/jamal/projects/yumyums/hq/.planning/phases/02-fill-out-and-conditional-logic/02-UI-SPEC.md` — Approved UI contract: exact class names, component inventory, interaction contracts, copywriting, page layout tree
- `/Users/jamal/projects/yumyums/hq/.planning/phases/02-fill-out-and-conditional-logic/02-CONTEXT.md` — All 11 locked decisions
- `/Users/jamal/projects/yumyums/hq/.planning/research/ARCHITECTURE.md` — Template/response split, condition evaluation pure function, component boundaries, data flow
- `/Users/jamal/projects/yumyums/hq/.planning/research/PITFALLS.md` — Pitfalls 1, 2, 4, 5, 6, 7, 8 directly applicable to Phase 2

### Secondary (MEDIUM confidence)

- `/Users/jamal/projects/yumyums/hq/.planning/research/FEATURES.md` — Table stakes confirmation, corrective action UX rationale, progress indicator priority

### Tertiary (LOW confidence — not used)

- None required; all claims verified from primary sources

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all from Phase 1 carry-forward
- Architecture: HIGH — patterns drawn directly from ARCHITECTURE.md and Phase 1 workflows.html
- Pitfalls: HIGH — sourced from dedicated PITFALLS.md + UI-SPEC interaction contracts resolve ambiguities

**Research date:** 2026-04-13
**Valid until:** 2026-07-13 (stable domain; no new CDN dependencies to monitor)

---

## Project Constraints (from CLAUDE.md)

The CLAUDE.md in this repository covers a Temporal server deployment in the `infra/` repo — those constraints (Docker, WSL2, Temporal versions) do not apply to this phase. The relevant project conventions are:

- Single HTML file per tool, all styles and logic inline
- No framework, no build step, no external stylesheets
- State-first render pattern (from purchasing.html convention)
- Event delegation: one listener per event type per container
- CSS custom properties from `:root` block — no new tokens in Phase 2
- `SCREAMING_SNAKE_CASE` for constants, `camelCase` for functions

No conflicting directives identified.
