# Phase 1: Template Builder - Context

**Gathered:** 2026-04-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Build the template builder tab — owner/manager can create and edit workflow checklist templates with named sections, all field types (checkbox, yes/no, text, temperature, photo), drag-to-reorder, temperature range thresholds, required photo marking, day-of-week conditions, and skip logic rules. The builder tab is role-gated via User Management permissions.

</domain>

<decisions>
## Implementation Decisions

### Builder Layout
- **D-01:** Single scrollable page showing sections + fields inline. No split view, no sidebar.
- **D-02:** Each section is a flat header with divider line (matches purchasing.html style), not collapsible cards.
- **D-03:** Each section has a "+ Add field" button at the bottom that opens a field type picker (checkbox, yes/no, text, temperature, photo).
- **D-04:** Editor only — no live preview pane. The fill-out view (Phase 2) serves as the preview.

### Field Configuration UX
- **D-05:** Tap a field row to expand an inline settings panel below it. Settings are specific to the field type. Tap again to collapse.
- **D-06:** Skip logic uses simple dropdowns: "Show this field only when [field picker] is [value picker]" — two dropdowns per rule.
- **D-07:** Day-of-week conditions use toggle chips: M T W T F Sa Su — tappable to include/exclude days.
- **D-08:** Temperature fields show min/max range inputs when expanded.

### Drag-Reorder
- **D-09:** Drag handle (☰ grip icon) on the left side of each field row. Uses SortableJS via CDN for smooth touch-native reordering.

### Template Data Shape
- **D-10:** Mock data matches future backend schema — UUID field IDs, nested sections array, typed field objects with conditions. This ensures seamless migration when backend arrives.
- **D-11:** Templates are editable in memory during the session. Changes reset on page refresh (no localStorage).

### Claude's Discretion
- Template name input and "requires approval" toggle placement — prefer top of page if there's enough space, otherwise Claude decides positioning.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Backend Design
- `docs/user-management-api.md` — Data model for users, roles, sessions, app permissions. Relevant for understanding the role-gating pattern the builder tab must respect.

### Research
- `.planning/research/ARCHITECTURE.md` — Template/Response split data model, condition evaluation approach, component boundaries and build order.
- `.planning/research/STACK.md` — SortableJS 1.15.7 recommendation, JsonLogic for conditional rules, photo capture approach.
- `.planning/research/PITFALLS.md` — DOM-as-source-of-truth warning (Pitfall 1), field ID stability requirement (Pitfall 3), condition dependency graph (Pitfall 2).
- `.planning/research/FEATURES.md` — Table stakes vs differentiators for checklist builder features.

### Existing Code
- `users.html` — Reference for tab-based layout, role pills, toggle switches, inline form patterns.
- `purchasing.html` — Reference for section headers, card layout, tab switching pattern.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Tab switching pattern** (`show(n)` function): Used in purchasing.html and users.html — same pattern for the two-tab layout (fill-out + builder).
- **CSS variables**: All pages share the same `:root` variable block for theming/dark mode.
- **Toggle switch CSS**: Already built in users.html (`.toggle`, `.slider`) — reusable for day-of-week chips and approval toggles.
- **Pill badges**: `.pill-*` classes from users.html — reusable for field type indicators.
- **Back link**: `.back` with `::before` content `'‹'` — standard navigation pattern.
- **ptr.js**: Pull-to-refresh shared utility — include via `<script src="ptr.js">`.

### Established Patterns
- One HTML page per tool, all styles inline, vanilla JS in a `<script>` tag at bottom.
- Module-level data constants in `SCREAMING_SNAKE_CASE` (e.g., `USERS`, `APPS`, `DEFAULT_PERMS`).
- Short abbreviated CSS class names (`.hd`, `.nm`, `.mt`, `.ft`).
- Dynamic DOM building via `createElement` + `innerHTML` template literals.
- `dblclick` prevention and service worker registration at bottom of every page.

### Integration Points
- `index.html` — Will need a "Workflows" tile added (Phase 3, not this phase).
- `sw.js` — Will need `workflows.html` added to ASSETS and cache bumped (Phase 3).
- User Management permissions — The builder tab checks role/permission to show/hide. In the mock, this can be simulated with a JS constant.

</code_context>

<specifics>
## Specific Ideas

- Day-of-week chips should use two-letter abbreviations: M T W T F Sa Su (not single letters — avoids T/T ambiguity for Tuesday/Thursday).
- SortableJS loaded via CDN `<script>` tag, not npm — matches zero-dependency convention.
- Field type picker after "+ Add field" should feel like a simple dropdown or small inline menu, not a full modal.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-template-builder*
*Context gathered: 2026-04-13*
