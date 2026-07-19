# Phase 04: Onboarding App - Research

**Researched:** 2026-04-13
**Domain:** Vanilla JS PWA page — onboarding checklist tool with role-based views, sequential section progression, video-part training items, FAQ gate, and manager sign-off flow
**Confidence:** HIGH (all claims derived from direct codebase inspection — no external dependencies)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Two role-based views: "My Trainings" tab (all roles — crew sees their own onboarding checklists) and "Manager" tab (manager+ only — sees all hires with Active/Completed sub-views).
- **D-02:** Manager Active view shows card-per-hire with name, role, start date, and progress bar (e.g., "60% — 12/20 tasks").
- **D-03:** Manager taps a hire's card to see the full checklist view (same view crew sees, but read-only for manager).
- **D-04:** Onboarding checklists are auto-assigned by role. A "Line Cook" gets the "Line Cook Onboarding" template automatically.
- **D-05:** Managers can also manually assign additional training checklists to a hire beyond the role default.
- **D-06:** Simpler format than workflows — checkboxes grouped by sections (e.g., Paperwork, Training, Equipment). No temperature, skip logic, or fail triggers.
- **D-07:** Primary content is video-based training. Each training topic is a series of video parts (e.g., "Kitchen Cleanup" → Part 1, Part 2, Part 3). Each part has a title, short description, and embedded video link (YouTube/Vimeo URL).
- **D-08:** Expandable sections — tap a training topic to expand and see video parts listed with title + description. Check off each part after watching. Section header shows progress count (e.g., "[2/3]").
- **D-09:** Each onboarding checklist can have an FAQ section with Q&A pairs.
- **D-10:** Viewing the FAQ is required — crew member must expand/view the FAQ before the checklist can be considered complete.
- **D-11:** Per-section sign-off, optional per section (configurable in mock template data).
- **D-12:** Sections must be completed in order — can't start section 2 until section 1 is done + signed off (if sign-off required for that section).
- **D-13:** Items within a section can be completed in any order.
- **D-14:** When sign-off is required, crew completes all items → section enters "pending sign-off" state → manager approves or sends back → next section unlocks.
- **D-15:** Manager Active tab: card per hire with progress bar and task count.
- **D-16:** "My Trainings" view for crew: shows their assigned checklists with per-checklist progress.

### Claude's Discretion

- FAQ interaction pattern (accordion vs gate page vs other — must enforce "viewed" requirement)
- Sign-off request/approval interaction (submit-for-review button vs auto-notify)
- Empty state messaging for both crew and manager views
- Exact card styling, progress bar appearance, section expand/collapse animation
- How manager assigns additional training checklists (UI for manual assignment)
- Mock template content (realistic food truck onboarding items with video titles/descriptions)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

## Summary

Phase 04 builds `onboarding.html` — a standalone HTML page using the same vanilla JS/CSS patterns established in `workflows.html`. There are no new external dependencies; all patterns (tabs, cards, progress bars, event delegation, state-first rendering, role-gated views, approval flows) already exist in the codebase and must be reused directly.

The key architectural challenge is the sequential section lock: sections unlock one at a time as the previous section is completed and signed off. This requires a state machine per section (unlocked → in-progress → pending-sign-off → signed-off) rather than the free-order fill flow in `workflows.html`. The FAQ gate adds a second boolean flag per checklist that must be satisfied before a "submit complete" action is available.

The two-tab layout (My Trainings / Manager) with role-gating mirrors the Approvals tab gating pattern already established in Phase 3. The Manager view's drill-down (hire card → read-only checklist) introduces a navigation stack (list → detail) that is a minor extension of the existing `fillState.activeTemplate` drill-down pattern.

**Primary recommendation:** Implement `onboarding.html` as a direct structural clone of `workflows.html` — same tab/show() pattern, same event delegation, same state-first render — but with a new simpler data model (no temperature, no skip logic, no fail triggers) and the new sequential-section + FAQ-gate behaviors as the only novel logic.

---

## Standard Stack

### Core

No new packages. This phase is pure static HTML/CSS/JS.

| Technology | Version | Purpose | Why Standard |
|------------|---------|---------|--------------|
| Vanilla JS (ES6+) | Browser-native | Page logic | Project constraint — no build step, no framework |
| CSS custom properties | Browser-native | Theming and dark mode | Established in every existing page |
| Service Worker Cache API | Browser-native | PWA offline | Required by sw.js for cache-first PWA |

### Supporting

| Asset | Version | Purpose | When to Use |
|-------|---------|---------|-------------|
| `ptr.js` | Project file | Pull-to-refresh on iOS PWA | Load as `<script src="ptr.js">` at bottom of body — identical to every other page |
| SortableJS | NOT needed | Drag-to-reorder | Only needed in Builder — onboarding has no drag reorder |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Inline `<style>` block | Shared stylesheet | Project convention is inline — do not break |
| Vanilla JS state | React/Vue | No build step; no framework is the hard constraint |

**Installation:** None required. No npm, no CDN additions beyond what already exists.

---

## Architecture Patterns

### Recommended File Structure

No new directories. One new file at project root:

```
hq/
├── onboarding.html    ← NEW: full standalone page
├── sw.js              ← EDIT: add './onboarding.html' to ASSETS, bump CACHE version
├── index.html         ← EDIT: activate the Hiring tile (or add new tile) linking to onboarding.html
└── users.html         ← EDIT: add {slug:'onboarding', name:'Onboarding', icon:'🎓'} to APPS array
```

### Pattern 1: Tab Layout with Role Gate (2 tabs)

`onboarding.html` has two top-level tabs: "My Trainings" (all roles) and "Manager" (manager+ only).

**Tab HTML:**
```html
<div class="tabs">
  <button id="t1" class="on" onclick="show(1)">My Trainings</button>
  <button id="t2" onclick="show(2)">Manager</button>
</div>
<div id="s1"><div id="my-body"></div></div>
<div id="s2" style="display:none"><div id="mgr-body"></div></div>
```

**show() function:** Identical to all other pages — iterate `[1,2]`, toggle `style.display`.

**Role gate:** Hide the Manager tab button entirely if `CURRENT_USER.role` is `team_member`. Copy the role-gate pattern from Phase 3 (Approvals tab was hidden for non-managers).

```js
// After rendering tabs, hide Manager tab if not manager+
if (!['admin','manager','superadmin'].includes(CURRENT_USER.role)) {
  document.getElementById('t2').style.display = 'none';
}
```

### Pattern 2: Mock Data Model (ONBOARDING-specific)

Define two top-level constants at the top of the inline `<script>`:

```js
// Hired crew members (the people being onboarded)
const MOCK_HIRES = [
  { id: 'hire_1', userId: 'usr_tina', name: 'Tina L.', role: 'line_cook', startDate: '2026-04-01',
    assignedTemplates: ['tpl_line_cook'] },
  { id: 'hire_2', userId: 'usr_dev', name: 'Dev P.', role: 'cashier', startDate: '2026-04-10',
    assignedTemplates: ['tpl_cashier', 'tpl_food_safety'] },
];

// Onboarding template definitions
const MOCK_OB_TEMPLATES = [
  {
    id: 'tpl_line_cook',
    name: 'Line Cook Onboarding',
    role: 'line_cook',
    sections: [
      {
        id: 'sec_paperwork',
        title: 'Paperwork',
        requiresSignOff: true,
        items: [
          { id: 'itm_w4', type: 'checkbox', label: 'W-4 tax form signed' },
          { id: 'itm_id', type: 'checkbox', label: 'ID photocopied and filed' },
        ]
      },
      {
        id: 'sec_training',
        title: 'Video Training',
        requiresSignOff: true,
        items: [
          {
            id: 'itm_kitchen_cleanup', type: 'video_series', label: 'Kitchen Cleanup',
            parts: [
              { id: 'part_kc1', title: 'Part 1: Surfaces and Equipment', desc: 'How to sanitize prep surfaces between orders.', url: 'https://youtube.com/watch?v=example1' },
              { id: 'part_kc2', title: 'Part 2: Deep Clean Protocol', desc: 'End-of-day deep clean procedure.', url: 'https://youtube.com/watch?v=example2' },
            ]
          },
        ]
      },
      {
        id: 'sec_faq',
        title: 'FAQ',
        isFaq: true,     // special flag — gate logic applies
        requiresSignOff: false,
        items: [
          { id: 'faq_1', q: 'What do I do if I run out of supplies mid-shift?', a: 'Text the owner immediately and check the backup cabinet under the grill.' },
          { id: 'faq_2', q: 'What is the dress code?', a: 'Non-slip shoes, clean black shirt, and a Yumyums apron at all times.' },
        ]
      }
    ]
  }
];
```

**Key data model decisions:**
- `requiresSignOff: true` on a section means it must enter "pending-sign-off" state before the next section unlocks.
- `isFaq: true` marks the FAQ section — items are Q&A pairs, not checkboxes. The section is considered "viewed" when the crew member explicitly expands it.
- `type: 'video_series'` items expand to show a list of parts. Each part has a checkbox to mark as watched.
- `type: 'checkbox'` items are simple check-off items.

### Pattern 3: Section State Machine

Each section per hire has a status. Store in a mutable dict:

```js
// SECTION_STATES[hireId][sectionId] = 'locked' | 'active' | 'pending_signoff' | 'signed_off'
const SECTION_STATES = {};

function initSectionStates(hire) {
  const tpl = MOCK_OB_TEMPLATES.find(t => t.id === hire.assignedTemplates[0]);
  if (!tpl) return;
  SECTION_STATES[hire.id] = {};
  tpl.sections.forEach((sec, i) => {
    SECTION_STATES[hire.id][sec.id] = i === 0 ? 'active' : 'locked';
  });
}
```

**Section transition rules:**
- `locked` → `active`: all items in the previous section checked + previous section signed off (if required) or completed (if no sign-off required)
- `active` → `pending_signoff`: all items checked AND section has `requiresSignOff: true` AND crew taps "Request Sign-Off" button
- `active` → `signed_off`: all items checked AND section has `requiresSignOff: false` (auto-transition on last checkbox)
- `pending_signoff` → `signed_off`: manager taps "Approve" in Manager tab
- `pending_signoff` → `active`: manager taps "Send Back" (resets section to active so crew can redo items)

### Pattern 4: Checkbox and Video-Part Responses

Flat dict keyed by item or part ID. Copy the `MOCK_RESPONSES` pattern from `workflows.html`:

```js
// OB_CHECKS[itemId] = { checked: true, checkedBy: 'Tina L.', checkedAt: Date }
const OB_CHECKS = {};

// FAQ_VIEWED[hireId] = true  (set when FAQ section is expanded)
const FAQ_VIEWED = {};
```

### Pattern 5: Event Delegation (established in workflows.html)

ONE click listener on the container. Route via `data-action` and `data-*` attributes. Do NOT attach inline `onclick` on dynamically rendered elements.

```js
document.getElementById('my-body').addEventListener('click', function(e) {
  var btn = e.target.closest('[data-action]');
  if (!btn) return;
  var action = btn.dataset.action;
  if (action === 'open-checklist') { /* ... */ }
  if (action === 'toggle-item')    { /* ... */ }
  if (action === 'toggle-section') { /* ... */ }
  if (action === 'request-signoff'){ /* ... */ }
  if (action === 'expand-faq')     { /* ... */ }
  // etc.
});
```

### Pattern 6: Progress Bar (reuse from renderChecklistList)

Inline `<div>` progress bar — identical to the pattern in `renderChecklistList()` in `workflows.html`:

```html
<div style="height:4px;background:var(--brd);border-radius:2px;margin-top:6px;overflow:hidden">
  <div style="height:100%;background:#27ae60;border-radius:2px;width:${pct}%"></div>
</div>
```

Compute `pct` = checked items / total items × 100.

### Pattern 7: Drill-Down Navigation (hire card → checklist runner)

Use the same `fillState` pattern from `workflows.html`:

```js
let obState = {
  view: 'list',       // 'list' | 'runner'
  activeHireId: null,
  activeTemplateIdx: 0,
};

function openChecklist(hireId, tplIdx) {
  obState = { view: 'runner', activeHireId: hireId, activeTemplateIdx: tplIdx || 0 };
  renderMyTrainings();
}
function backToList() {
  obState = { view: 'list', activeHireId: null, activeTemplateIdx: 0 };
  renderMyTrainings();
}
```

`renderMyTrainings()` branches on `obState.view` to render either the list or the runner.

### Pattern 8: FAQ Interaction (recommended accordion with viewed gate)

Use accordion pattern (expand/collapse Q&A pairs on tap) with a one-time "viewed" gate.

**Recommendation (Claude's Discretion):** When the crew member taps to expand the FAQ section, set `FAQ_VIEWED[hireId] = true` and re-render. The "checklist complete" check must include `FAQ_VIEWED[hireId] === true`. The section header shows a "View Required" badge until viewed, then a green checkmark. This is simpler than a gate page and keeps the crew in context.

### Anti-Patterns to Avoid

- **Attaching event handlers inside render functions:** Causes duplicate listeners on re-render. Use event delegation on the container div only.
- **Calling render functions from within input event handlers for text fields:** Causes cursor jump (established pitfall from Phase 2). For any future text input in onboarding, use `input` event + update state without re-rendering.
- **Hard-coding role checks as string comparisons without a helper:** Collect all role-check logic into a small `isManager()` helper to avoid inconsistency.
- **Forgetting to bump sw.js cache version:** New page won't be cached by existing service workers. This is Pitfall 10 — the SW must be bumped in the same commit as `onboarding.html` is created.
- **Using `latest` cache key in sw.js:** Not applicable here (sw.js uses a numeric version string), but the version must be bumped monotonically.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Drag-to-reorder | Custom drag logic | N/A — onboarding has no reordering | N/A |
| Accordion expand/collapse | Animation library | CSS `max-height` transition or `display` toggle | Native CSS is sufficient for this use case; no animation library needed |
| Progress calculation | Complex state tracker | Simple ratio: `checked / total * 100` | Already proven in `renderChecklistList()` |
| Role gating | Auth middleware | `CURRENT_USER.role` check before render | Mock-only; full auth is v2 |
| Video embed | Custom video player | Plain `<a href="...">` link opening in new tab/browser | iOS PWA cannot embed YouTube iframes reliably without user gesture; a tap-to-open-link is simpler and works offline |

**Key insight on video links:** Do NOT use `<iframe>` embeds for YouTube/Vimeo in a PWA. iOS Safari in standalone mode blocks cross-origin iframes without a user gesture, and embedded video breaks offline behavior. The correct pattern is a styled link/button that opens the URL in the system browser (`target="_blank"` or `window.open(url)`). The part is marked "watched" by checking the checkbox after tapping the link — the app cannot verify actual viewing, which is acceptable for a mock.

---

## Runtime State Inventory

Step 2.5: SKIPPED — this is a greenfield page addition, not a rename/refactor/migration phase. No existing runtime state involves the string "onboarding" that would require migration.

---

## Environment Availability

Step 2.6: This phase is purely static file creation with no external tools, CLIs, runtimes, or services beyond a browser and a text editor. No dependency audit required.

All assets (ptr.js, sw.js, manifest.json, icons/) are already present in the project root.

---

## Common Pitfalls

### Pitfall 1: SW Cache Not Bumped on New Page

**What goes wrong:** `onboarding.html` is created but not added to `ASSETS` in `sw.js` and/or the `CACHE` version string is not bumped. Installed PWA users are served stale cache that doesn't include `onboarding.html` — they get a 404 or the index.html fallback.

**Why it happens:** Service worker caches only what was in `ASSETS` at the time of the last install. Adding a new file to the repo doesn't update existing caches.

**How to avoid:** In the same task that creates `onboarding.html`, also: (1) add `'./onboarding.html'` to the `ASSETS` array in `sw.js`, and (2) bump `CACHE` from `'yumyums-v39'` to `'yumyums-v40'` (or next available number).

**Warning signs:** Page loads fine in a desktop browser but shows HQ home screen on mobile PWA.

### Pitfall 2: Event Listeners Accumulate on Re-Render

**What goes wrong:** If a click handler is attached inside a render function (e.g., `el.onclick = fn` after `innerHTML` assignment), every call to re-render attaches another handler. State mutation causes exponential handler calls.

**Why it happens:** `innerHTML` reassignment destroys old DOM elements (good) but the render function re-attaches handlers to the new elements each time (accumulates).

**How to avoid:** Use the event delegation pattern — ONE listener attached once on the static container element (`#my-body`, `#mgr-body`), never inside render functions. Route to handlers via `data-action` attributes on dynamically generated elements.

**Warning signs:** Clicking a checkbox causes multiple state changes or multiple re-renders.

### Pitfall 3: Section Lock Logic Ordering Bug

**What goes wrong:** Section 2 unlocks before section 1 is truly complete because the completion check runs before the state update, or the FAQ section is treated as a regular section for ordering purposes.

**Why it happens:** Off-by-one errors in "unlock next section" logic; FAQ gate not checked alongside item completion.

**How to avoid:** Implement a single `isSectionComplete(hireId, sectionId)` predicate function. Call it in one place. The checklist "complete" predicate is: all sections are `signed_off` AND `FAQ_VIEWED[hireId] === true`. Run `isSectionComplete` after every state mutation before rendering, not before.

**Warning signs:** Section 3 is accessible when section 2 still shows incomplete items.

### Pitfall 4: Manager Read-Only View Accidentally Allows Edits

**What goes wrong:** Manager drills into a hire's checklist and can check items, which corrupts the hire's actual progress state.

**Why it happens:** The same render function is used for both crew and manager views, and checkboxes are always interactive.

**How to avoid:** Pass a `readOnly` flag into the runner render function. When `readOnly === true`, render checkboxes as static display elements (no `data-action`), not interactive inputs. The Manager view always passes `readOnly: true`.

**Warning signs:** Manager checks a box in a hire's checklist and the hire's progress bar changes.

### Pitfall 5: FAQ "Viewed" Gate Not Enforced for Completion

**What goes wrong:** Crew member completes all checkboxes and video parts but never expands the FAQ section — the checklist is still counted as complete.

**Why it happens:** The completion check only counts checked items and doesn't include the FAQ_VIEWED gate.

**How to avoid:** The "request completion" or "checklist complete" predicate must be:
```js
function isChecklistComplete(hireId, tplId) {
  // All items across all non-FAQ sections checked
  // + FAQ_VIEWED[hireId] = true
  // + All sections are signed_off (or no sign-off required)
}
```
Never derive completion from item counts alone.

**Warning signs:** Checklist shows "100%" progress but FAQ was never opened.

### Pitfall 6: index.html Tile Not Updated

**What goes wrong:** `onboarding.html` exists and works directly, but there is no way to reach it from the HQ home screen.

**Why it happens:** Developer forgets to update `index.html`.

**How to avoid:** In the same plan that creates `onboarding.html`, also convert the most appropriate "Soon" tile in `index.html` to an `active` tile linking to `onboarding.html`. The "Hiring" tile (`🧑‍🤝‍🧑`) is the closest semantic match in the current grid.

**Warning signs:** Navigating to `index.html` shows the Hiring tile as "Soon" after deploy.

---

## Code Examples

Verified patterns from the existing codebase:

### Progress Bar (from workflows.html line 1247)

```js
const pct = total ? Math.round(checked / total * 100) : 0;
// In template string:
`<div style="height:4px;background:var(--brd);border-radius:2px;margin-top:6px;overflow:hidden">
  <div style="height:100%;background:#27ae60;border-radius:2px;width:${pct}%"></div>
</div>`
```

### Tab Show/Hide Pattern (from workflows.html line 148-154)

```js
function show(n) {
  [1,2].forEach(i => {
    document.getElementById('s'+i).style.display = i===n ? '' : 'none';
    document.getElementById('t'+i).classList.toggle('on', i===n);
  });
}
```

### Event Delegation with data-action (from workflows.html, Phase 1 pattern)

```js
document.getElementById('my-body').addEventListener('click', function(e) {
  var btn = e.target.closest('[data-action]');
  if (!btn) return;
  var action = btn.dataset.action;
  var hireId = btn.dataset.hireId;
  if (action === 'toggle-item') {
    var itemId = btn.dataset.itemId;
    if (OB_CHECKS[itemId]) { delete OB_CHECKS[itemId]; }
    else { OB_CHECKS[itemId] = { checkedBy: CURRENT_USER.name, checkedAt: new Date() }; }
    renderMyTrainings();
  }
});
```

### SW Cache Bump Pattern (sw.js)

```js
// Before: const CACHE = 'yumyums-v39';
// After:
const CACHE = 'yumyums-v40';
const ASSETS = ['./', './index.html', './purchasing.html', './users.html',
  './login.html', './workflows.html', './onboarding.html', './ptr.js', './manifest.json'];
```

### CSS Variable Block (copy verbatim to onboarding.html `<style>` block)

```css
:root{--bg:#f5f5f3;--card:#fff;--txt:#1a1a1a;--mut:#6b6b6b;--brd:rgba(0,0,0,0.08);--info-bg:#e6f1fb;--info-tx:#0c447c;--warn-bg:#faeeda;--warn-tx:#854f0b}
@media(prefers-color-scheme:dark){:root{--bg:#1a1a1a;--card:#262626;--txt:#f0f0f0;--mut:#999;--brd:rgba(255,255,255,0.1);--info-bg:#0c447c;--info-tx:#b5d4f4;--warn-bg:#633806;--warn-tx:#fac775}}
```

### PWA Boilerplate (copy verbatim to onboarding.html `<head>`)

```html
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover">
<meta name="theme-color" content="#0c447c">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="default">
<link rel="manifest" href="manifest.json">
```

### PWA Script Boilerplate (end of inline `<script>`)

```js
document.addEventListener('dblclick', e => e.preventDefault());
if('serviceWorker' in navigator) navigator.serviceWorker.register('sw.js');
```

---

## State of the Art

This phase uses no external libraries beyond what is already in the project. All patterns are stabilized from Phases 1-3.

| Pattern | Established In | Status |
|---------|---------------|--------|
| Event delegation via `data-action` | Phase 1 | Locked — use this |
| State-first rendering | Phase 1 | Locked — use this |
| `CURRENT_USER` mock for attribution | Phase 2 | Locked — reuse constant name |
| Role-gated tab visibility | Phase 3 | Locked — copy pattern |
| Card + progress bar + drill-down | Phase 2-3 | Locked — copy pattern |
| SW cache bump on new page | Phase 1 | Required — always do this |

---

## Open Questions

1. **Which "Soon" tile in index.html becomes the Onboarding tile?**
   - What we know: The "Hiring" tile (row 4, `🧑‍🤝‍🧑`) has the closest semantic match. There is no dedicated "Onboarding" tile currently. The grid has 7 tiles (2 active, 5 "Soon").
   - What's unclear: Owner preference — Onboarding could be its own tile, or the Hiring tile could be repurposed.
   - Recommendation: Repurpose the Hiring tile (`🧑‍🤝‍🧑 Hiring`) to `🎓 Onboarding` pointing to `onboarding.html`. Update both the icon, title, and description. This is Claude's discretion territory.

2. **CURRENT_USER mock: which role to default to for testing?**
   - What we know: `MOCK_CURRENT_USER` in `workflows.html` defaults to `admin`. Onboarding needs to test both `team_member` (My Trainings only) and `manager`/`admin` (Manager tab visible).
   - Recommendation: Default to `admin` in the mock (same as workflows.html) so Manager tab is visible during development. Add a comment noting that setting role to `team_member` hides the Manager tab.

3. **How many mock onboarding templates are needed?**
   - What we know: CONTEXT.md says "mock data only" with role-based auto-assignment.
   - Recommendation: 2 templates (Line Cook Onboarding, Cashier Onboarding) with 2-3 mock hires assigned to them. This is enough to demonstrate both tabs without making the mock unwieldy.

---

## Project Constraints (from CLAUDE.md)

These directives are mandatory — planner must not propose approaches that contradict them:

| Directive | Source | Implication for This Phase |
|-----------|--------|---------------------------|
| Static HTML/CSS/JS only — no build step, no framework | CLAUDE.md Conventions | `onboarding.html` must be a standalone static file with inline `<style>` and `<script>` |
| Minified inline CSS in each page | CLAUDE.md Conventions | CSS block must use the compact single-line format (no whitespace between rules) |
| Event delegation in workflows.html (not inline onclick on dynamic elements) | CLAUDE.md Conventions | Use `data-action` + container listener — no inline `onclick` on dynamically generated elements |
| `SCREAMING_SNAKE_CASE` for constants, `camelCase` for functions | CLAUDE.md Conventions | `MOCK_HIRES`, `MOCK_OB_TEMPLATES`, `OB_CHECKS`, `SECTION_STATES`, `FAQ_VIEWED` as constants; `renderMyTrainings()`, `renderManager()`, `openChecklist()` as functions |
| SW cache must be bumped before every deploy and human-verify checkpoint | CLAUDE.md Conventions | Bump `CACHE` version in `sw.js` in the same commit as `onboarding.html` creation |
| Mocks only — no localStorage, no API calls | CLAUDE.md Architecture | All data in JS constants/variables; no persistence |
| Mobile-first (max-width 480px) | CLAUDE.md Constraints | `.app{max-width:480px;margin:0 auto}` required; all touch targets min 44px |
| GSD Workflow Enforcement | CLAUDE.md | Use GSD commands; no direct edits outside a GSD workflow |

---

## Sources

### Primary (HIGH confidence)

- Direct inspection of `/Users/jamal/projects/yumyums/hq/workflows.html` — state model, event delegation, render functions, progress bar pattern, approval flow, data structures
- Direct inspection of `/Users/jamal/projects/yumyums/hq/sw.js` — cache version (v39), ASSETS array, cache-first pattern
- Direct inspection of `/Users/jamal/projects/yumyums/hq/index.html` — tile grid, active/soon tile pattern, current tiles
- Direct inspection of `/Users/jamal/projects/yumyums/hq/users.html` — USERS array, role definitions, APPS array
- `.planning/codebase/CONVENTIONS.md` — naming, CSS, JS conventions
- `.planning/codebase/STRUCTURE.md` — "Where to Add New Code" checklist
- `.planning/phases/04-onboarding-app/04-CONTEXT.md` — locked decisions D-01 through D-16

### Secondary (MEDIUM confidence)

- CLAUDE.md project instructions — stack constraints and conventions (authoritative for this project)
- STATE.md accumulated decisions — Phase 1-3 implementation decisions

### Tertiary (LOW confidence)

- None. All findings are from direct codebase inspection.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; all patterns are in the existing codebase
- Architecture: HIGH — data model and state machine derived from locked decisions + existing patterns
- Pitfalls: HIGH — all pitfalls are documented bugs or patterns from Phases 1-3 (SW bump is Pitfall 10 from Phase 1, event accumulation is Phase 1 pattern, cursor jump is Phase 2 pattern)

**Research date:** 2026-04-13
**Valid until:** Stable — no external dependencies to expire. Valid until codebase structure changes.
