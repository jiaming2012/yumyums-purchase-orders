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
  created:
    - tests/onboarding.spec.js
  modified:
    - onboarding.html
    - sw.js (already at yumyums-v42 from Plan 01)
    - index.html (emoji + tagline update, Needs Attention badge)

key-decisions:
  - "initOBSortable() targets all .ob-field-list containers — both regular item lists and FAQ Q&A lists use same class, enabling a single querySelectorAll loop"
  - "No template duplication feature per D-17 — scope kept tight"
  - "sub-items use subItems array with {id, label} objects; existing MOCK_OB_TEMPLATES checkbox items get subItems via || [] fallback when user adds first sub-item"
  - "show() re-renders all tabs on switch to fix stale builder state bug found during human verification"
  - "obState.view reset to null on tab switch to prevent stale editor reappearing"
  - "SO pill label renamed to Require Sign-off for clarity"

patterns-established:
  - "renderOBCheckboxItem(item, secId, idx) / renderOBVideoItem(item, secId, idx): item-type-specific render functions returning HTML strings"
  - "ob-field-list on all sortable containers: enables unified SortableJS initialization"
  - "Input handlers: mutate state silently (no re-render). Click handlers: re-render after mutation."

requirements-completed: [D-04, D-05, D-06, D-07, D-08, D-17]

# Metrics
duration: 45min
completed: 2026-04-14
---

# Phase 05 Plan 02: Onboarding Builder Item Types Summary

**Full onboarding builder human-verified and approved: checkbox/sub-item editing, video series multi-part editor, FAQ Q&A, SortableJS drag-to-reorder, and 35 E2E tests**

## Performance

- **Duration:** ~45 min (including verification fixes and E2E test suite)
- **Started:** 2026-04-14T03:00:00Z
- **Completed:** 2026-04-14T07:30:00Z
- **Tasks:** 3 (Task 1: implementation, Task 2: SW bump, Task 3: human-verify — APPROVED)
- **Files modified:** 3 (onboarding.html, tests/onboarding.spec.js, index.html)

## Accomplishments

- Full item-level editing in the onboarding builder: checkbox items with nested sub-items, video series items with multi-part (title/desc/URL) editors
- FAQ Q&A editor on FAQ-mode sections: question input + answer textarea per entry, with add/delete and drag-to-reorder
- Item type picker per section: "+ Checkbox" and "+ Video Series" buttons
- SortableJS drag-to-reorder for all three levels: items within section, FAQ entries within FAQ section, sections within template
- 35 Playwright E2E tests added in tests/onboarding.spec.js covering full builder flow
- Human verification approved after 5 bugs/gaps were found and auto-fixed during review
- All existing 54 Playwright workflows E2E tests continue to pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Item types, sub-items, video parts, FAQ Q&A editor, SortableJS** — `4857601` (feat)
2. **Task 2: Bump sw.js cache version** — Already complete in Plan 01 (`ba82e4f` — sw.js at yumyums-v42)
3. **Task 3: Human verify + verification fixes** — `c656621` (feat), `56a3caf` (feat)

**Plan metadata:** `5f2d145` (docs: complete item types and builder plan, checkpoint for human-verify)

## Files Created/Modified

- `/Users/jamal/projects/yumyums/hq/onboarding.html` — Added CSS, factory functions, renderOBCheckboxItem, renderOBVideoItem, replaced renderOBSection placeholder body, added initOBSortable, updated renderOBBuilder, wrapped sections in ob-section-list, added 12 new click action handlers, 8 new input action handlers, SortableJS CDN script tag; verification fixes: tab re-rendering, FAQ textarea overflow, "SO"→"Require Sign-off" rename, pending sign-off badge, section expansion persistence, obState.view reset
- `/Users/jamal/projects/yumyums/hq/tests/onboarding.spec.js` — 35 E2E tests covering builder tab: template CRUD, section management, item add/delete, sub-items, video parts, FAQ Q&A
- `/Users/jamal/projects/yumyums/hq/index.html` — Updated onboarding tile emoji, tagline, added Needs Attention badge

## Decisions Made

- `initOBSortable()` uses `querySelectorAll('.ob-field-list')` to cover both regular item lists and FAQ Q&A lists — both use same class with `data-section-id` attribute
- No `duplicate-template` or `clone-template` feature added (per D-17)
- Sub-items: existing mock checkbox items don't have `subItems` — handled via `|| []` fallback when user adds first sub-item
- `show()` re-renders all tabs on switch to fix stale builder state bug found during human verification
- `obState.view` reset to null on tab switch to prevent stale editor reappearing

## Deviations from Plan

### Auto-fixed Issues During Human Verification

**1. [Rule 1 - Bug] Tab switch re-rendering — stale builder state**
- **Found during:** Task 3 (human verification)
- **Issue:** Switching tabs did not re-render the builder, so stale DOM would persist
- **Fix:** `show()` now calls renderOBBuilder() / renderMgrTab() on every tab switch; `obState.view` reset to null on switch
- **Files modified:** onboarding.html
- **Committed in:** c656621

**2. [Rule 1 - Bug] FAQ answer textarea overflow on mobile**
- **Found during:** Task 3 (human verification)
- **Issue:** Textarea width exceeded card bounds on narrow viewports
- **Fix:** Added `max-width:100%` and `box-sizing:border-box` to textarea style
- **Files modified:** onboarding.html
- **Committed in:** c656621

**3. [Rule 1 - Bug] "SO" pill label was cryptic**
- **Found during:** Task 3 (human verification)
- **Issue:** Section toggle labeled "SO" was not self-explanatory to crew
- **Fix:** Renamed to "Require Sign-off" for clarity
- **Files modified:** onboarding.html
- **Committed in:** c656621

**4. [Rule 2 - Missing Critical] Pending sign-off badge on Manager hire cards**
- **Found during:** Task 3 (human verification)
- **Issue:** Manager view showed hire cards with no indication which ones needed sign-off action
- **Fix:** Added yellow "Pending sign-off" badge to hire cards that have sign-off sections not yet approved
- **Files modified:** onboarding.html
- **Committed in:** c656621

**5. [Rule 2 - Missing Critical] Section stays expanded after approve/send-back**
- **Found during:** Task 3 (human verification)
- **Issue:** Section collapsed after manager approve/send-back action, losing scroll context
- **Fix:** Section expansion state preserved after approve/send-back actions
- **Files modified:** onboarding.html
- **Committed in:** c656621

---

**Total deviations:** 5 auto-fixed (3 bugs, 2 missing critical)
**Impact on plan:** All fixes were correctness/UX issues found during live verification. No scope creep — all changes directly serve the builder and manager review goals.

## Issues Encountered

- sw.js was already at `yumyums-v42` from Plan 01 execution. Task 2 acceptance criteria were already met — no additional change required.
- 35 Playwright E2E tests for onboarding.spec.js were added during verification (not originally in plan scope, treated as Rule 2 — missing critical test coverage for the new feature).

## Human Verification Result

**APPROVED** — User confirmed all 20 verification steps passed. Builder works end-to-end on desktop and mobile viewport. Fixes applied during verification were reviewed and accepted.

## Next Phase Readiness

- Phase 05-onboarding-builder is fully complete — both plans shipped and human-verified
- MOCK_OB_TEMPLATES data model supports all item types: checkbox + subItems, video_series + parts, FAQ Q&A entries
- Builder is role-gated (owner/manager only) and edits are in-memory (page refresh resets per D-18)
- No blockers for future phases

---
*Phase: 05-onboarding-builder*
*Completed: 2026-04-14*
