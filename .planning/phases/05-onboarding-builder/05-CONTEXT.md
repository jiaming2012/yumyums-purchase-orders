# Phase 5: Onboarding Builder - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Add a Builder tab to onboarding.html for creating and editing onboarding training templates. Owner/manager can create templates with sections, checkbox items (with sub-items), video series (multi-part with title, description, URL per part), FAQ Q&A pairs, per-section sign-off toggle, and drag-to-reorder. Reuses the simpler data model from Phase 4 — no temperature, skip logic, or fail triggers.

</domain>

<decisions>
## Implementation Decisions

### Builder Layout
- **D-01:** Mirror the workflows builder pattern — flat scrollable sections with inline item editing, "+ Add Item" per section, "+ Add Section" at bottom.
- **D-02:** Third tab added to onboarding.html: My Trainings / Manager / Builder. Builder tab is role-gated to manager+ (same pattern as workflows).
- **D-03:** Editor only — no live preview pane. The My Trainings view serves as preview.
- **D-04:** SortableJS for drag-to-reorder on both items within a section and sections within a template. SortableJS loaded via CDN (same as workflows.html).

### Item Types
- **D-05:** Two item types in the picker: Checkbox and Video Series.
- **D-06:** Checkbox items support sub-items (nested checkboxes, like the operations/workflows sub-steps feature). Owner can add sub-items under a checkbox.
- **D-07:** Video Series items expand inline to show a list of parts. Each part has title, description, and URL fields — all editable inline. "+ Add Part" button at bottom of the part list.
- **D-08:** FAQ is NOT an item type — it's a section-level toggle (see Section Configuration below). When FAQ is enabled on a section, the section shows a Q&A editor instead of items.

### Section Configuration
- **D-09:** Inline toggles on the section header row: [SO] for Requires Sign-Off and [FAQ] for FAQ Section.
- **D-10:** When FAQ toggle is on, the section switches to a Q&A editor — each entry has a Question and Answer field. "+ Add Q&A" button at bottom.
- **D-11:** When FAQ toggle is off, the section shows the normal item list with "+ Add Item".
- **D-12:** Sections are drag-to-reorder via SortableJS.

### Template Management
- **D-13:** Builder tab shows a template list — card per template with name, role, and section count. Tap to open in editor.
- **D-14:** "+ New Template" button — owner enters name, selects a role, starts building from scratch.
- **D-15:** Edit existing templates — tap from list to open in builder.
- **D-16:** Delete template with confirmation dialog.
- **D-17:** No duplicate feature (deferred — keep scope tight).

### Template Data Shape
- **D-18:** Templates are editable in memory during the session. Changes apply immediately to MOCK_OB_TEMPLATES. Reset on page refresh (no localStorage) — same pattern as workflows builder.
- **D-19:** Role field on template drives auto-assignment (Phase 4 D-04). Role selection is a dropdown of known roles from the mock data.

### Claude's Discretion
- Template name input and role selector placement (top of editor page)
- Delete button placement and confirmation dialog style
- Empty state for builder tab when no templates exist
- How sub-items are visually nested under checkbox items in the builder
- Drag handle styling (grip icon pattern from workflows builder)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing Onboarding Code
- `onboarding.html` — Current 730-line implementation with My Trainings + Manager tabs, `MOCK_OB_TEMPLATES` data model, `SECTION_STATES` state machine, event delegation pattern on `#my-body` and `#mgr-body`. Builder tab adds to this file.

### Workflows Builder Reference
- `workflows.html` — The workflows builder tab implementation. Reference for: section CRUD, field type picker, inline settings expansion, SortableJS drag-to-reorder, event delegation with `data-action`, sub-steps feature on checkbox fields.

### Phase 4 Context
- `.planning/phases/04-onboarding-app/04-CONTEXT.md` — Data model decisions (D-06 through D-10) that define the template shape the builder must produce.
- `.planning/phases/04-onboarding-app/04-RESEARCH.md` — Data model details, pitfalls, patterns.

### Phase 1 Context (Workflows Builder)
- `.planning/phases/01-template-builder/01-CONTEXT.md` — Builder layout decisions (D-01 through D-11). Reference for proven patterns: flat sections, inline settings, SortableJS, in-memory editing.

### Infrastructure
- `sw.js` — Must bump cache version after builder changes.
- `.planning/codebase/CONVENTIONS.md` — Naming patterns, CSS variable block, event delegation conventions.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `onboarding.html` tabs pattern (`show(n)` with `#t1`, `#t2` — extend to `#t3` for Builder)
- `MOCK_OB_TEMPLATES` — existing data model that builder will edit in-place
- CSS variable block and dark mode already in onboarding.html
- SortableJS CDN pattern from workflows.html — reuse same version (1.15.7)

### Established Patterns
- Event delegation with `data-action` attributes — add third listener on `#builder-body`
- State-first rendering: mutate `MOCK_OB_TEMPLATES` → call `renderBuilder()` → DOM updates
- `isManager()` role check already exists — reuse for builder tab gating
- Sub-steps pattern from workflows.html — checkbox items with nested sub-items

### Integration Points
- `onboarding.html` — Add Builder tab (third tab), add `#builder-body` container, add `renderBuilder()` and related functions
- `sw.js` — Bump cache version
- SortableJS CDN — Add `<script>` tag to onboarding.html (same as workflows.html)

</code_context>

<specifics>
## Specific Ideas

- Sub-items on checkboxes should work like the operations/workflows sub-steps feature — nested checkboxes under a parent checkbox
- FAQ section editor switches between item mode and Q&A mode based on the FAQ toggle — clean visual distinction needed
- Video part editing is inline with editable title/description/URL fields, not a separate modal

</specifics>

<deferred>
## Deferred Ideas

- Template duplication (clone existing as starting point) — future enhancement
- Template versioning / change history — out of scope for mocks

</deferred>

---

*Phase: 05-onboarding-builder*
*Context gathered: 2026-04-14*
