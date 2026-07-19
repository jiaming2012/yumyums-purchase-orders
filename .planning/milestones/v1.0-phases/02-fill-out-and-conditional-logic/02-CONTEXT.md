# Phase 2: Fill-Out and Conditional Logic - Context

**Gathered:** 2026-04-13
**Status:** Ready for planning

<domain>
## Phase Boundary

Build the crew fill-out experience — crew members can see today's active checklists (day-filtered), open one, complete all non-photo field types (checkbox, yes/no, text, temperature), encounter inline corrective actions on fail triggers, and track progress with user attribution. Skip logic and day-of-week conditions hide/show fields and sections dynamically.

</domain>

<decisions>
## Implementation Decisions

### Checklist Completion UX
- **D-01:** Tap the whole row to toggle checkbox/yes-no items. Large touch target for kitchen use.
- **D-02:** Completed items show strikethrough text with a green checkmark to the right.
- **D-03:** Text and temperature fields use inline inputs directly in the row — tap to focus, type, move on. No modal or expand pattern.

### Fail Trigger Behavior
- **D-04:** Out-of-range temperature or "No" answer triggers an inline slide-down card below the field. Red-tinted background. Doesn't block other items.
- **D-05:** Corrective action form includes three parts: text note ("Describe what you did"), photo capture stub (placeholder for Phase 3), and severity picker (minor / major / critical).

### User Attribution
- **D-06:** Completed items show initials badge + time (e.g., "JM · 2:15p") to the right of the checkmark. Compact for mobile.
- **D-07:** Hardcoded current user: `CURRENT_USER = {name: 'Jamal M.', initials: 'JM'}`. Every checked item gets stamped with this user.

### Day Filtering & Checklist List
- **D-08:** Card per checklist on the list view — shows name, section count, and progress bar. Tap to open.
- **D-09:** Empty state when no checklists active today: "No checklists for today. Enjoy your day off!" with current day name.
- **D-10:** Progress shown in both places: progress bar on list card + "X of Y items complete" counter inside the checklist at top.

### Conditional Logic Runtime
- **D-11:** Fields and sections hidden by skip logic or day-of-week conditions are not rendered. Hidden field answers are cleared from response state.

### Claude's Discretion
- Exact progress bar styling and placement
- How sections render in fill-out mode (cards vs flat headers — can differ from builder)
- Submit button behavior (mock only — show confirmation)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1 Artifacts
- `.planning/phases/01-template-builder/01-CONTEXT.md` — Data model decisions (D-10, D-11) that carry forward
- `.planning/phases/01-template-builder/01-RESEARCH.md` — Template/Response split architecture, condition evaluation approach, SortableJS details

### Research
- `.planning/research/ARCHITECTURE.md` — Template/Response split data model, condition evaluation as pure functions, component boundaries
- `.planning/research/PITFALLS.md` — DOM-as-source-of-truth (Pitfall 1), condition dependency graph (Pitfall 2), iOS Safari keyboard bugs (Pitfall 5), fail trigger must be inline not modal (Pitfall 6)
- `.planning/research/FEATURES.md` — Table stakes for fill-out: checkbox, yes/no, text, temperature, sections, user attribution, progress

### Existing Code
- `workflows.html` — Phase 1 output: MOCK_TEMPLATES data model, builder tab, all CSS variables and component classes. Fill-out tab (#s1) is currently stubbed.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Tab switching** (`show(n)`): Fill-out is Tab 1 (#s1), Builder is Tab 2 (#s2). Already wired in workflows.html.
- **MOCK_TEMPLATES data**: Templates with sections, fields, field types, conditions, fail triggers all defined. Fill-out reads this directly.
- **CSS variables**: All theming/dark mode from `:root` block. Can add fill-out-specific classes.
- **Event delegation pattern**: Single click + change listeners on container elements. Fill-out should follow the same pattern on `#fillout-body` or similar.
- **Day-of-week logic**: `DAY_INDICES` mapping and condition.days arrays already exist on templates.

### Established Patterns
- State-first rendering: mutate JS state → call render function → DOM updates
- Event delegation: ONE listener per event type, use closest() to route
- Inline CSS in `<style>` block, no external stylesheets
- `SCREAMING_SNAKE_CASE` for constants, `camelCase` for functions

### Integration Points
- Fill-out view renders in `#s1` (currently stubbed with placeholder text)
- Must read MOCK_TEMPLATES and create Response objects (template/response split per ARCHITECTURE.md)
- Progress tracking needs a response state object to track completed items

</code_context>

<specifics>
## Specific Ideas

- Corrective action severity picker: three options (minor / major / critical) as tappable pills, not a dropdown
- Photo stub in corrective action: show a camera icon button that's visually present but non-functional (tooltip or "Coming soon" label) — Phase 3 will make it real
- Green checkmark should use a simple ✓ character, not an icon library
- Strikethrough uses CSS `text-decoration: line-through` with `color: var(--mut)` for dimming

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-fill-out-and-conditional-logic*
*Context gathered: 2026-04-13*
