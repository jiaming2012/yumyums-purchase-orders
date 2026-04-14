---
phase: 05-onboarding-builder
plan: 02
subsystem: ui
tags: [vanilla-js, sortablejs, html, pwa, onboarding, builder]

# Dependency graph
requires:
  - phase: 05-01
    provides: Builder tab infrastructure with template list, section CRUD, and event delegation skeleton
  - phase: 04-onboarding-app
    provides: MOCK_OB_TEMPLATES data model with checkbox, video_series, and FAQ item types

provides:
  - Checkbox items with nested sub-items in onboarding builder
  - Video series items with multi-part title/desc/URL editor
  - FAQ Q&A editor (question + answer textarea per entry)
  - Item type picker (+ Checkbox, + Video Series buttons per section)
  - SortableJS drag-to-reorder for items within sections, FAQ entries, and sections within template
  - All add/delete operations for items, sub-items, video parts, and FAQ entries
  - renderOBCheckboxItem() and renderOBVideoItem() rendering functions
  - initOBSortable() managing Sortable instances for all .ob-field-list and .ob-section-list containers
  - Input handlers for all item text fields (no re-render to prevent cursor jump)

affects:
  - 05-onboarding-builder (human verification checkpoint)

# Tech tracking
tech-stack:
  added: [sortablejs@1.15.7 via CDN]
  patterns:
    - renderOBCheckboxItem/renderOBVideoItem as separate render functions per item type
    - ob-field-list class on all sortable item containers (regular + FAQ) enables single initOBSortable() loop
    - Input events mutate state only (no re-render) to prevent cursor jump on text fields
    - Click events trigger renderOBBuilder() re-render for structural changes (add/delete)

key-files:
  created: []
  modified:
    - onboarding.html
    - sw.js (already at yumyums-v42 from Plan 01)

key-decisions:
  - "initOBSortable() targets all .ob-field-list containers — both regular item lists and FAQ Q&A lists use same class, enabling a single querySelectorAll loop"
  - "No template duplication feature per D-17 — scope kept tight"
  - "sub-items use subItems array with {id, label} objects; existing MOCK_OB_TEMPLATES checkbox items get subItems via || [] fallback when user adds first sub-item"

patterns-established:
  - "renderOBCheckboxItem(item, secId, idx) / renderOBVideoItem(item, secId, idx): item-type-specific render functions returning HTML strings"
  - "ob-field-list on all sortable containers: enables unified SortableJS initialization"
  - "Input handlers: mutate state silently (no re-render). Click handlers: re-render after mutation."

requirements-completed: [D-04, D-05, D-06, D-07, D-08, D-17]

# Metrics
duration: 15min
completed: 2026-04-14
---

# Phase 05 Plan 02: Onboarding Builder Item Types Summary

**Onboarding builder completed with checkbox sub-items, video series multi-part editor, FAQ Q&A editor, item type picker, and SortableJS drag-to-reorder for items, FAQ entries, and sections**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-04-14T03:00:00Z
- **Completed:** 2026-04-14T03:15:00Z
- **Tasks:** 2 (Task 3 is human-verify checkpoint)
- **Files modified:** 1 (onboarding.html; sw.js already at v42 from Plan 01)

## Accomplishments

- Full item-level editing in the onboarding builder: checkbox items with nested sub-items, video series items with multi-part (title/desc/URL) editors
- FAQ Q&A editor on FAQ-mode sections: question input + answer textarea per entry, with add/delete and drag-to-reorder
- Item type picker per section: "+ Checkbox" and "+ Video Series" buttons
- SortableJS drag-to-reorder for all three levels: items within section, FAQ entries within FAQ section, sections within template
- All existing 54 Playwright E2E tests pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Item types, sub-items, video parts, FAQ Q&A editor, SortableJS** - `4857601` (feat)
2. **Task 2: Bump sw.js cache version** - Already complete in Plan 01 (`ba82e4f` — sw.js at yumyums-v42)

## Files Created/Modified

- `/Users/jamal/projects/yumyums/hq/onboarding.html` — Added CSS, factory functions, renderOBCheckboxItem, renderOBVideoItem, replaced renderOBSection placeholder body, added initOBSortable, updated renderOBBuilder, wrapped sections in ob-section-list, added 12 new click action handlers, 8 new input action handlers, SortableJS CDN script tag (+237 lines)

## Decisions Made

- `initOBSortable()` uses `querySelectorAll('.ob-field-list')` to cover both regular item lists and FAQ Q&A lists — both use same class with `data-section-id` attribute
- No `duplicate-template` or `clone-template` feature added (per D-17)
- Sub-items: existing mock checkbox items don't have `subItems` — handled via `|| []` fallback when user adds first sub-item

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

sw.js was already at `yumyums-v42` from Plan 01 execution. Task 2 acceptance criteria were already met — no additional change required.

## Next Phase Readiness

- Task 3 (checkpoint:human-verify) awaits human review of the complete onboarding builder end-to-end
- Human must verify: template CRUD, section management (SO/FAQ toggles), checkbox items with sub-items, video series with parts, FAQ Q&A editor, SortableJS drag-to-reorder, delete operations, existing templates render correctly, mobile layout

---
*Phase: 05-onboarding-builder*
*Completed: 2026-04-14*
