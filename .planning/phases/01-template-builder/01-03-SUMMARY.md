---
phase: 01-template-builder
plan: 03
subsystem: ui
tags: [vanilla-js, pwa, sortablejs, drag-reorder, day-chips, skip-logic, event-delegation]

requires:
  - 01-02: renderFieldExpanded() with placeholder comment for day chips + skip logic

provides:
  - initSortable() — SortableJS drag-to-reorder for field-list elements with cleanup on re-render
  - renderDayChips(activeDays, targetId, targetType) — Mon-first day chip row for sections and fields
  - getPrecedingFields(template, sectionId, fieldId) — returns all fields before current in template order
  - getValuesForFieldType(type) — type-appropriate value options for skip logic second dropdown
  - renderSkipLogic(fld, secId) — two-dropdown skip logic editor with "Remove condition" button
  - Day chip click delegation (section + field targets, coexists with skip logic on condition object)
  - skip-field-select change delegation (sets field_id + operator=equals, resets value)
  - skip-value-select change delegation (JSON.parse value into condition.value)
  - skip-clear click delegation (removes skip logic, preserves day conditions)

affects: [02-fill-out]

tech-stack:
  added:
    - "SortableJS 1.15.7 via unpkg CDN (drag-to-reorder)"
  patterns:
    - "sortableInstances destroy+recreate pattern: destroy all before initSortable() on every re-render"
    - "DAY_INDICES [1,2,3,4,5,6,0] mapping: display position 0=Mon maps to JS day index 1"
    - "condition object coexistence: days[] and field_id/operator/value can coexist on same field.condition"
    - "Skip logic only for preceding fields: getPrecedingFields returns early when it hits the current field"
    - "No re-render for skip-value-select: value update is state-only, no DOM side effects"

key-files:
  created: []
  modified:
    - workflows.html

key-decisions:
  - "DAY_INDICES Mon-first mapping: display labels M T W T F Sa Su map to JS Date.getDay() indices 1 2 3 4 5 6 0"
  - "Day chips on sections rendered in a padded div below sec-hd, not inside sec-hd flex row"
  - "renderSkipLogic returns empty string when no preceding fields (first field in first section has no skip logic)"
  - "skip-value-select uses JSON.stringify/JSON.parse to serialize boolean/string values through option value attributes"
  - "sw.js was already at v6 with workflows.html cached (done in Plan 01-01) — no sw.js changes needed"

requirements-completed: [BLDR-03, BLDR-06, BLDR-07]

duration: 5min
completed: 2026-04-13
---

# Phase 1 Plan 03: Drag, Day Chips, and Skip Logic Summary

**SortableJS drag-to-reorder, Mon-first day-of-week chips on sections and fields, and two-dropdown skip logic editor — all wired via state-first mutations and existing event delegation, completing all BLDR requirements**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-04-13T08:23:00Z
- **Completed:** 2026-04-13T08:28:26Z
- **Tasks:** 2 autonomous + 1 pending human verification
- **Files modified:** 1

## Accomplishments

### Task 1: SortableJS drag-to-reorder and day-of-week chips (c5af3ea)

- Added SortableJS 1.15.7 CDN script tag in `<head>`
- Added `initSortable()` function: destroys previous instances, creates Sortable on each `.field-list` with `handle: '.drag-handle'`, `animation: 150`, and `onEnd` splice logic updating `field.order`
- `initSortable()` called at end of `renderBuilder()` when `state.view === 'editor'`
- Added `DAYS` label array and `DAY_INDICES = [1,2,3,4,5,6,0]` for Mon-first mapping to JS Date.getDay() values
- Added `renderDayChips(activeDays, targetId, targetType)` rendering 7 chip buttons with `.on` class for active days
- Day chips rendered on sections (in padded div below `sec-hd`) and in field expanded settings panel
- Day chip click handler in event delegation: toggling day in/out of `sec.condition.days` or `fld.condition.days`, coexists with skip logic on field condition, resets condition to null when days array empties

### Task 2: Skip logic editor (ff35287)

- Added `getPrecedingFields(template, sectionId, fieldId)` — iterates sections/fields in order, returns early when current field is hit, collects `{ id, label, type, sectionTitle }` for each preceding field
- Added `getValuesForFieldType(type)` — returns value option pairs for checkbox, yes_no, text, temperature, photo
- Added `renderSkipLogic(fld, secId)` — renders "SHOW THIS FIELD ONLY WHEN" label + field picker select (only when preceding fields exist) + value picker select (only when field selected) + "Remove condition" button
- Wired `renderSkipLogic()` into `renderFieldExpanded()` after day chips
- Added `skip-field-select` change handler: sets `condition.field_id`, `condition.operator = 'equals'`, clears `condition.value`; clearing selection removes skip logic while preserving days
- Added `skip-value-select` change handler: `JSON.parse(select.value)` into `condition.value`, no re-render
- Added `skip-clear` click handler: removes field_id/operator/value from condition, nullifies if no days remain

### Service Worker

sw.js was already updated to `yumyums-v6` with `workflows.html` in ASSETS (completed in Plan 01-01). No changes needed.

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | SortableJS drag-to-reorder and day-of-week chips | c5af3ea | workflows.html |
| 2 | Skip logic editor | ff35287 | workflows.html |
| 3 | Human verification | PENDING | — |

## Files Created/Modified

- `workflows.html` — +184 lines total across both tasks: SortableJS CDN, DAYS/DAY_INDICES constants, renderDayChips, getPrecedingFields, getValuesForFieldType, renderSkipLogic, initSortable, day chip section/field rendering, skip logic rendering, click/change delegation updates

## Decisions Made

- `DAY_INDICES [1,2,3,4,5,6,0]` chosen to map Mon-first display order to JS `Date.getDay()` values, ensuring day conditions work correctly in Phase 2 fill-out evaluation
- Day chips on sections placed in a separate padded `<div>` below the `sec-hd` flex row rather than inside it — avoids layout conflicts with the section title and delete button
- `renderSkipLogic()` returns empty string when preceding fields is empty — first fields in first section get no skip logic UI (correct UX, not an error state)
- `skip-value-select` uses `JSON.stringify`/`JSON.parse` round-trip for option `value` attributes to handle booleans and special strings correctly through HTML attribute serialization
- sw.js not modified — already at correct state from Plan 01-01

## Deviations from Plan

None — plan executed exactly as written. The only notable difference is that sw.js required no changes (already updated in Plan 01-01 per STATE.md decision log), which is consistent with plan expectations since Task 2 noted "Update sw.js" but the state was already correct.

## Checkpoint: Task 3 Pending

**Task 3: Visual and functional verification** is a `type="checkpoint:human-verify"` task. Autonomous execution is complete. Human must verify in browser:

1. Open workflows.html in a browser
2. Verify Fill Out and Builder tabs
3. Verify 2 pre-built templates in list
4. Open Morning Opening Checklist — verify sections/fields display
5. Add temperature field — verify day chips appear in settings
6. Tap a photo field — verify Required toggle appears
7. Drag a field via grip handle — verify reorder within section
8. Toggle day chips on section and field — verify accent styling
9. On a field with preceding fields, verify skip logic dropdowns appear
10. Select a preceding yes/no field — verify Yes/No value options
11. Back to template list — verify navigation
12. Create new template — verify blank editor opens
13. Remove 'workflow_builder' permission and refresh — verify Builder tab disappears

Resume: Type "approved" or describe issues.

## Known Stubs

None — all planned functionality is implemented and wired to state.

## Self-Check: PASSED

- `workflows.html` exists and modified: FOUND
- Commit c5af3ea exists: FOUND
- Commit ff35287 exists: FOUND
- All acceptance criteria grep checks passed (see task execution above)

---
*Phase: 01-template-builder*
*Completed: 2026-04-13 (Tasks 1-2; Task 3 pending human verify)*
