# Phase 1: Template Builder - Research

**Researched:** 2026-04-12
**Domain:** Vanilla JS drag-and-drop form builder, mobile-first PWA, in-memory state management
**Confidence:** HIGH — primary sources are the existing codebase and prior project-level research already verified against official docs

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Single scrollable page showing sections + fields inline. No split view, no sidebar.
- **D-02:** Each section is a flat header with divider line (matches purchasing.html style), not collapsible cards.
- **D-03:** Each section has a "+ Add field" button at the bottom that opens a field type picker (checkbox, yes/no, text, temperature, photo).
- **D-04:** Editor only — no live preview pane. The fill-out view (Phase 2) serves as the preview.
- **D-05:** Tap a field row to expand an inline settings panel below it. Settings are specific to the field type. Tap again to collapse.
- **D-06:** Skip logic uses simple dropdowns: "Show this field only when [field picker] is [value picker]" — two dropdowns per rule.
- **D-07:** Day-of-week conditions use toggle chips: M T W T F Sa Su — tappable to include/exclude days.
- **D-08:** Temperature fields show min/max range inputs when expanded.
- **D-09:** Drag handle (☰ grip icon) on the left side of each field row. Uses SortableJS via CDN for smooth touch-native reordering.
- **D-10:** Mock data matches future backend schema — UUID field IDs, nested sections array, typed field objects with conditions. This ensures seamless migration when backend arrives.
- **D-11:** Templates are editable in memory during the session. Changes reset on page refresh (no localStorage).

### Claude's Discretion

- Template name input and "requires approval" toggle placement — prefer top of page if there's enough space, otherwise Claude decides positioning.

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BLDR-01 | Owner/manager can create a new workflow template with a name and sections | Template data model (D-10) + TemplateEditor + SectionEditor component pattern from ARCHITECTURE.md |
| BLDR-02 | Owner/manager can add field types to a section: checkbox, yes/no, text, temperature, photo | Field type picker (D-03) + FieldTypePicker component + field schema from ARCHITECTURE.md |
| BLDR-03 | Owner/manager can reorder fields within a section via drag | SortableJS 1.15.7 via unpkg CDN (D-09) — verified available |
| BLDR-04 | Owner/manager can delete fields from a template | Event delegation pattern (Pitfall 4) — delete button per field row; removes from state then re-renders |
| BLDR-05 | Owner/manager can set temperature range thresholds (min/max) for fail trigger | Inline settings panel (D-05/D-08) — min/max inputs exposed when temperature field is expanded |
| BLDR-06 | Owner/manager can configure day-of-week conditions on sections or fields | Toggle chips M T W T F Sa Su (D-07) — stored as active_days array on section or field |
| BLDR-07 | Owner/manager can configure skip logic rules (if field X = Y, show/hide field Z) | ConditionEditor (D-06) — two dropdowns: field picker (preceding fields only) + value picker |
| BLDR-08 | Owner/manager can mark a photo field as required (must upload before submission) | Inline settings panel (D-05) — "required" toggle shown when photo field is expanded |

</phase_requirements>

---

## Summary

Phase 1 builds `workflows.html` (builder tab only — Tab 2). The page follows the established one-HTML-file-per-tool convention: all styles inline, all logic in a `<script>` block at the bottom, no build step, no framework. The builder tab is the only deliverable; the fill-out tab (Tab 1) gets scaffolded but not implemented.

The most technically complex decisions are already locked by CONTEXT.md and validated by prior project-level research. The data model schema (ARCHITECTURE.md) is the foundation everything else builds on — it must be defined first and kept stable, as downstream phases (fill-out view, photo capture) depend on it. SortableJS handles drag-to-reorder; the condition/skip logic editor is a pure vanilla JS dropdown pair, not a rule engine.

The primary risk is the DOM-as-source-of-truth trap (Pitfall 1) and event listener accumulation (Pitfall 4) — both well-documented in project pitfalls and preventable with the state-first render pattern already used in `purchasing.html` and `users.html`.

**Primary recommendation:** Establish MOCK_TEMPLATES data and render function first. Wire all mutations through state → re-render rather than reading from DOM. Use event delegation on section/field containers. SortableJS via unpkg@1.15.7.

---

## Standard Stack

### Core (No Build Step)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Vanilla JS (ES2020+) | native | All interactivity and state | Established project convention; no framework allowed |
| CSS custom properties | native | Theming and dark mode | Already in every page via `:root` block |
| SortableJS | 1.15.7 | Touch-native drag-to-reorder for field rows | Only viable drag library for vanilla JS + iOS touch; cdnjs has 1.15.6 only — use unpkg |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| ptr.js (project utility) | local | Pull-to-refresh | Include via `<script src="ptr.js">` on every page |

### What NOT to Use

| Rejected | Reason |
|----------|--------|
| JsonLogic CDN | Phase 1 skip logic is two simple dropdowns (D-06); JsonLogic is overkill for the builder UI and its CDN availability is unverified. Condition objects are stored as plain JS objects. Evaluation logic deferred to Phase 2 fill-out view. |
| Alpine.js / any framework | Inconsistent with existing codebase pattern |
| Any CSS framework | Conflicts with established CSS custom property system |
| `temporalio/web`, Elasticsearch | Irrelevant — those are from CLAUDE.md Temporal project, not this phase |

### Installation

```html
<!-- SortableJS — use unpkg, NOT cdnjs (cdnjs only has 1.15.6, not 1.15.7) -->
<script src="https://unpkg.com/sortablejs@1.15.7/Sortable.min.js"></script>
```

**Version verification (performed 2026-04-12):**
- `cdnjs.cloudflare.com/ajax/libs/Sortable/1.15.7/Sortable.min.js` → 404 (cdnjs is on 1.15.6)
- `unpkg.com/sortablejs@1.15.7/Sortable.min.js` → 200 OK (confirmed)
- npm registry latest: 1.15.7

Use unpkg for 1.15.7. Alternatively use cdnjs 1.15.6 (also confirmed 200 OK) — no breaking changes between patch versions.

---

## Architecture Patterns

### Recommended Project Structure

```
workflows.html
  <style>        — all CSS inline (matches purchasing.html pattern)
  <body>
    .app
      .back         — "‹ HQ" link to index.html
      .tabs         — "Fill Out" | "Builder" (Tab 2 hidden if no builder permission)
      #s1           — fill-out tab (stubbed in Phase 1)
      #s2           — builder tab (full implementation in Phase 1)
  <script>
    // === CONSTANTS ===
    MOCK_CURRENT_USER     — { id, name, role, permissions }
    MOCK_BUILDER_ROLES    — ['admin', 'manager']
    MOCK_TEMPLATES        — [ ...template objects ]

    // === STATE ===
    let state = { activeTemplate: null, expandedField: null }

    // === PURE FUNCTIONS ===
    generateId()          — 'fld_' + Date.now() + Math.random()
    getDefaultConfig(type) — returns config defaults per field type

    // === RENDER ===
    renderBuilder()       — full re-render of builder tab from state
    renderSection(sec)    — one section header + its fields
    renderField(fld, secId) — one field row (collapsed state)
    renderFieldExpanded(fld, secId) — inline settings panel (expanded state)
    renderFieldTypePicker(secId) — "+ Add field" picker

    // === MUTATIONS (all go through state → re-render) ===
    addSection()
    deleteSection(secId)
    addField(secId, type)
    deleteField(secId, fldId)
    updateField(secId, fldId, patch)
    updateSection(secId, patch)
    toggleFieldExpanded(fldId)

    // === SORTABLE INIT ===
    initSortable()        — called after renderBuilder(); one Sortable per section

    // === BOOT ===
    show(n)               — tab switching (identical to purchasing.html)
    checkBuilderAccess()  — hide Tab 2 button if role not in MOCK_BUILDER_ROLES
    renderBuilder()       — initial render
```

### Pattern 1: State-First Render (critical)

**What:** Maintain `MOCK_TEMPLATES` as the single source of truth. All mutations write to state. Re-render is triggered after every mutation. Never read field values from the DOM inside business logic.

**When to use:** Every interaction — add field, delete field, change label, set condition.

**Example:**
```javascript
// Source: established pattern from purchasing.html + Pitfall 1 prevention
function deleteField(secId, fldId) {
  const tpl = state.activeTemplate;
  const sec = tpl.sections.find(s => s.id === secId);
  sec.fields = sec.fields.filter(f => f.id !== fldId);
  renderBuilder();  // full re-render from state
}

// WRONG — never do this:
// document.querySelector(`[data-field-id="${fldId}"] input`).value
```

### Pattern 2: Event Delegation

**What:** Attach one listener to the parent container. Identify the target field via `data-field-id` and `data-section-id` attributes. Never attach listeners to individual field rows.

**When to use:** All field-level interactions (expand/collapse, delete, label change).

**Example:**
```javascript
// Source: Pitfall 4 prevention pattern from PITFALLS.md
document.getElementById('builder-body').addEventListener('click', e => {
  const fieldRow = e.target.closest('[data-field-id]');
  if (!fieldRow) return;
  const fldId = fieldRow.dataset.fieldId;
  const secId = fieldRow.dataset.sectionId;

  if (e.target.closest('.field-delete')) {
    deleteField(secId, fldId);
  } else if (e.target.closest('.field-row-tap')) {
    toggleFieldExpanded(fldId);
  }
});
```

### Pattern 3: SortableJS Initialization

**What:** One Sortable instance per section's field list. Initialized/re-initialized after each full re-render. On drag end, reads new order from DOM and syncs back to state.

**When to use:** After `renderBuilder()` completes; destroy old instances before creating new ones.

**Example:**
```javascript
// Source: SortableJS docs — https://github.com/SortableJS/Sortable
let sortableInstances = [];

function initSortable() {
  sortableInstances.forEach(s => s.destroy());
  sortableInstances = [];

  document.querySelectorAll('.field-list').forEach(el => {
    const secId = el.dataset.sectionId;
    sortableInstances.push(Sortable.create(el, {
      handle: '.drag-handle',
      animation: 150,
      onEnd(evt) {
        const sec = state.activeTemplate.sections.find(s => s.id === secId);
        const moved = sec.fields.splice(evt.oldIndex, 1)[0];
        sec.fields.splice(evt.newIndex, 0, moved);
        sec.fields.forEach((f, i) => f.order = i);
        // No re-render needed — SortableJS already moved the DOM node
        // Only sync state; re-render on next mutation
      }
    }));
  });
}
```

### Pattern 4: Template List → Editor Navigation

**What:** Builder tab has two sub-views: (a) template list and (b) template editor. Use a `state.view` flag to switch between them within Tab 2 rather than a new page or tab.

**When to use:** When user clicks a template row or "New template" button.

**Example:**
```javascript
// Sub-view switching within builder tab
function openEditor(tplId) {
  state.activeTemplate = MOCK_TEMPLATES.find(t => t.id === tplId)
    || createNewTemplate();
  state.view = 'editor';
  renderBuilder();
}

function backToList() {
  state.view = 'list';
  state.activeTemplate = null;
  state.expandedField = null;
  renderBuilder();
}
```

### Pattern 5: Day-of-Week Toggle Chips

**What:** Seven tappable chips (M T W T F Sa Su). Active chips are styled with `--info-bg` / `--info-tx`. Tap toggles the day in/out of the field/section's `active_days` array.

**When to use:** On section headers (template-level day gating) and on field expanded panels (field-level day conditions per D-07).

**Example:**
```javascript
// Chip rendering — inline in renderFieldExpanded() / renderSection()
const DAYS = ['M','T','W','T','F','Sa','Su'];
// Note: use index for storage (0=Mon…6=Sun) to avoid T/T label collision
function renderDayChips(activeDays, onChange) {
  return DAYS.map((label, i) => `
    <button class="day-chip ${activeDays.includes(i) ? 'on' : ''}"
            data-day="${i}">${label}</button>
  `).join('');
}
// Event handled via delegation on the expanded panel container
```

### Pattern 6: Skip Logic Editor (ConditionEditor)

**What:** Two dropdowns — (1) field picker showing only fields that appear before this one in section order, (2) value picker showing valid values for that field type. Stores result as a condition object on the field.

**When to use:** Inside expanded panel for any field except the first field in the template.

**Example:**
```javascript
// Condition object shape (from ARCHITECTURE.md)
condition: {
  field_id: "fld_abc123",
  operator: "equals",   // "equals" | "not_equals"
  value: false          // boolean for checkbox/yes_no, string for text
}

// Fields eligible as condition sources: all fields in same section
// with order < this field's order, plus all fields in prior sections
function getPrecedingFields(template, sectionId, fieldId) {
  const fields = [];
  for (const sec of template.sections) {
    for (const fld of sec.fields) {
      if (sec.id === sectionId && fld.id === fieldId) return fields;
      fields.push({ ...fld, sectionTitle: sec.title });
    }
  }
  return fields;
}
```

### Anti-Patterns to Avoid

- **Reading from DOM:** Any `document.querySelector('.field-row').value` inside a mutation function. State owns all values.
- **Attaching listeners in loops:** Any `fields.forEach(f => f.el.addEventListener(...))`. Use delegation instead.
- **Using array index as field ID:** `template.sections[0].fields[2]` as a reference. Always use `field.id`.
- **CSS-only builder tab gating:** `display:none` via CSS only. Must be JS-gated (`checkBuilderAccess()`) to establish correct pattern for when real auth arrives (Pitfall 13).
- **Calling `new Date()` in the builder:** Day-of-week conditions are stored as definitions in the builder; evaluation happens in Phase 2 fill-out view at runtime (Pitfall 7).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Touch-native drag-to-reorder | Custom touchstart/touchmove/touchend drag logic | SortableJS 1.15.7 | iOS touch events for drag are non-trivial; SortableJS handles cross-browser, ghost element, animation, and axis constraints |
| Field ID generation | Sequential integers or array indices | `'fld_' + Date.now() + '_' + Math.random().toString(36).slice(2,6)` | Integers break on reorder; IDs must survive array mutations (Pitfall 3) |
| Toggle chip state management | Manual class toggling with multiple conditions | `data-day` attribute + toggle pattern from users.html `.toggle` pattern | Reinventing the wheel; the pattern is already in the codebase |

**Key insight:** The builder's complexity is in the state model and render loop — not in specialized libraries. The only external dependency needed is SortableJS.

---

## Common Pitfalls

### Pitfall 1 (CRITICAL): DOM as Source of Truth

**What goes wrong:** Reading field label values from `<input>` elements inside mutation functions instead of from `state.activeTemplate`.

**Why it happens:** It feels like a shortcut — the input is right there.

**How to avoid:** All mutations accept a `patch` object. Labels, types, and config values are read from `state.activeTemplate`, never from the DOM. The DOM is a rendering artifact.

**Warning signs:** Any function that calls `document.querySelector` and passes the result into a state mutation.

### Pitfall 4 (CRITICAL): Event Listener Accumulation

**What goes wrong:** Adding/deleting fields re-renders the field list and re-attaches click handlers, causing double/triple fires on delete and collapse buttons.

**Why it happens:** Listener attached inside the render loop, not to the parent container.

**How to avoid:** One delegated listener on `#builder-body` attached once at boot. Never inside `renderBuilder()`.

**Warning signs:** Delete fires twice after the second field is added.

### Pitfall 3: Field ID Instability

**What goes wrong:** Referencing fields by array index when passing to mutation functions; the index changes after reorder.

**How to avoid:** Always capture `data-field-id` from the DOM. Never capture `data-index`. IDs are generated with `Date.now()` at field creation and never change.

### Pitfall 13: CSS-Only Role Gating

**What goes wrong:** Builder tab hidden with just `display:none` — accessible via DevTools.

**How to avoid:** JS check: `if (!MOCK_CURRENT_USER.permissions.includes('workflow_builder'))` — do not render the tab button at all. CSS reflects the JS decision.

### Pitfall 11: Multi-Column Builder on Mobile

**What goes wrong:** Showing field type and config side-by-side in a two-column layout that collapses badly at 375px.

**How to avoid:** All builder controls are single-column stacked layout, max-width 480px, matching purchasing.html and users.html conventions.

### Pitfall 10: Service Worker Cache Miss

**What goes wrong:** `workflows.html` is not in the service worker ASSETS list. Crew phones get a 404 from the cache after deploy.

**How to avoid:** sw.js must be updated in Phase 1 to add `'./workflows.html'` to ASSETS and bump the CACHE version string. This is a Phase 1 task even though full integration is Phase 3.

---

## Code Examples

### Template Data Shape (from ARCHITECTURE.md — canonical)

```javascript
// Source: .planning/research/ARCHITECTURE.md
const MOCK_TEMPLATES = [
  {
    id: "tpl_morning_open",
    name: "Morning Opening Checklist",
    active_days: [1, 2, 3, 4, 5],   // 0=Sun…6=Sat; Mon–Fri
    requires_approval: false,
    assignable_to: ["admin", "manager", "team_member"],
    sections: [
      {
        id: "sec_equipment",
        title: "Equipment Check",
        order: 0,
        condition: null,   // null = always show; { days: [1,2,3,4,5] } for day-gating
        fields: [
          {
            id: "fld_grill_temp",
            type: "temperature",   // "checkbox"|"yes_no"|"text"|"temperature"|"photo"
            label: "Grill surface temperature",
            required: true,
            order: 0,
            config: { unit: "°F", min: 350, max: 500 },
            fail_trigger: { type: "out_of_range", min: 350, max: 500 },
            corrective_action: { label: "Corrective action taken", type: "text", required: true },
            condition: null   // skip logic condition; null = always show
          },
          {
            id: "fld_grill_clean",
            type: "checkbox",
            label: "Grill grates cleaned",
            required: true,
            order: 1,
            config: {},
            fail_trigger: null,
            corrective_action: null,
            condition: null
          }
        ]
      }
    ]
  }
];
```

### Role Gating Pattern

```javascript
// Source: .planning/research/ARCHITECTURE.md + docs/user-management-api.md pattern
const MOCK_CURRENT_USER = {
  id: "usr_jamal",
  name: "Jamal",
  role: "admin",
  permissions: ["workflow_fill", "workflow_builder"]
};
const MOCK_BUILDER_ROLES = ["admin", "manager"];

function checkBuilderAccess() {
  const canBuild = MOCK_CURRENT_USER.permissions.includes("workflow_builder")
    || MOCK_BUILDER_ROLES.includes(MOCK_CURRENT_USER.role);
  if (!canBuild) {
    document.getElementById("t2").style.display = "none";
  }
}
```

### Section Header (flat, matches purchasing.html style)

```javascript
// Source: purchasing.html .cat pattern
function renderSection(sec) {
  return `
    <div class="sec-hd" data-section-id="${sec.id}">
      <span class="sec-title">${sec.title}</span>
      <button class="sec-delete" data-section-id="${sec.id}">✕</button>
    </div>
    <div class="field-list" data-section-id="${sec.id}">
      ${sec.fields.map(f => renderField(f, sec.id)).join('')}
    </div>
    <button class="add-field-btn" data-section-id="${sec.id}">+ Add field</button>
    <div class="section-divider"></div>
  `;
}
```

### Inline Settings Panel Toggle

```javascript
// Source: D-05 decision — tap to expand/collapse
function toggleFieldExpanded(fldId) {
  state.expandedField = state.expandedField === fldId ? null : fldId;
  renderBuilder();
}

function renderField(fld, secId) {
  const isExpanded = state.expandedField === fld.id;
  return `
    <div class="field-row ${isExpanded ? 'expanded' : ''}"
         data-field-id="${fld.id}"
         data-section-id="${secId}">
      <span class="drag-handle">☰</span>
      <span class="field-tap-target field-row-tap">${fld.label || fld.type}</span>
      <span class="field-type-pill">${fld.type}</span>
      <button class="field-delete" data-field-id="${fld.id}" data-section-id="${secId}">✕</button>
      ${isExpanded ? renderFieldExpanded(fld, secId) : ''}
    </div>
  `;
}
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|-----------------|--------|
| cdnjs SortableJS 1.15.7 (404) | unpkg.com/sortablejs@1.15.7 | Use unpkg URL — cdnjs is on 1.15.6 |
| `temporalio/web` | `temporalio/ui` | Irrelevant to this phase |

**Verified versions:**
- SortableJS latest npm: 1.15.7 (npm registry confirmed 2026-04-12)
- SortableJS latest cdnjs: 1.15.6 (cdnjs API confirmed 2026-04-12)

---

## Open Questions

1. **"Requires approval" toggle placement**
   - What we know: User left this to Claude's discretion (CONTEXT.md)
   - What's unclear: Whether it fits cleanly at the top of the editor without competing with template name
   - Recommendation: Place template name input + requires-approval toggle in a single "meta" card at the top of the editor view, before the first section. This matches the card+hd pattern from purchasing.html and keeps the most important settings visible without scrolling.

2. **Section-level vs field-level day-of-week conditions**
   - What we know: D-07 mentions day-of-week chips on fields; BLDR-06 says "on sections or fields"
   - What's unclear: Should the builder expose day chips on both section headers and individual fields, or only on sections?
   - Recommendation: Expose on both. Sections get day chips in the section header (tappable chips inline). Fields get day chips inside their expanded settings panel. Both store to the same `condition` shape.

3. **Field type picker UX after "+ Add field"**
   - What we know: CONTEXT.md says "simple dropdown or small inline menu, not a full modal"
   - What's unclear: Exact rendering (inline below the button, or a popover?)
   - Recommendation: Inline expansion below the "+ Add field" button — a small card with 5 tappable rows (one per field type with an icon). Dismissed by tapping a type or tapping elsewhere. Matches the no-modal principle.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| SortableJS (unpkg CDN) | BLDR-03 drag reorder | Network-dependent | 1.15.7 | cdnjs 1.15.6 (200 OK) |
| ptr.js (local) | Pull-to-refresh | Yes (in repo) | local | Omit if missing |
| sw.js (local) | Service worker | Yes (in repo) | yumyums-v5 | Must bump to v6 when workflows.html added |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:**
- If unpkg is unavailable at build time: fall back to `cdnjs.cloudflare.com/ajax/libs/Sortable/1.15.6/Sortable.min.js` (confirmed 200 OK; 1.15.6 vs 1.15.7 is a patch difference with no breaking changes for this use case).

---

## Sources

### Primary (HIGH confidence)
- `/Users/jamal/projects/yumyums/hq/.planning/research/ARCHITECTURE.md` — template/response data model, component boundaries, condition evaluation pattern, build order
- `/Users/jamal/projects/yumyums/hq/.planning/research/PITFALLS.md` — all 13 pitfalls, phase-specific warnings
- `/Users/jamal/projects/yumyums/hq/.planning/research/STACK.md` — SortableJS rationale, CDN choice, JsonLogic deferral reasoning
- `/Users/jamal/projects/yumyums/hq/purchasing.html` — canonical reference for CSS classes, tab switching, section header, card patterns
- `/Users/jamal/projects/yumyums/hq/users.html` — canonical reference for toggle switch, pill badges, form field patterns, `.toggle` / `.slider` CSS
- `/Users/jamal/projects/yumyums/hq/sw.js` — current ASSETS list (must add workflows.html in Phase 1)
- `/Users/jamal/projects/yumyums/hq/docs/user-management-api.md` — role/permission model for builder access gating

### Secondary (MEDIUM confidence)
- npm registry: SortableJS 1.15.7 confirmed as latest stable (2026-04-12)
- cdnjs API: SortableJS latest on cdnjs is 1.15.6 (not 1.15.7 — CORRECTS prior research doc claim)
- unpkg: 1.15.7 confirmed 200 OK at `https://unpkg.com/sortablejs@1.15.7/Sortable.min.js`

### Tertiary (LOW confidence — not used)
- None required; all claims are verified from primary sources

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — SortableJS CDN URL verified live; all other stack choices are established project conventions
- Architecture: HIGH — patterns are drawn directly from existing codebase files; no new patterns introduced
- Pitfalls: HIGH — sourced from dedicated project-level PITFALLS.md which cites production experience

**Research date:** 2026-04-12
**Valid until:** 2026-07-12 (stable domain; SortableJS version may update but patch-level only)

---

## Project Constraints (from CLAUDE.md)

The CLAUDE.md in this project covers a Temporal server deployment in the `infra/` repo — it is not the yumyums HQ front-end project. Its constraints do not apply to this phase. The relevant project conventions are captured in the existing codebase files read above.

No conflicting directives identified.
