---
phase: 04-onboarding-app
plan: 02
subsystem: ui
tags: [vanilla-js, pwa, manager-tab, read-only, sign-off, event-delegation, onboarding]

# Dependency graph
requires:
  - 04-01 (onboarding.html scaffold, MOCK_HIRES, MOCK_OB_TEMPLATES, SECTION_STATES, OB_CHECKS, FAQ_VIEWED, EXPANDED_SECTIONS)
provides:
  - Manager tab fully functional in onboarding.html
  - renderManager() with Active/Completed sub-views
  - renderMgrList() hire cards with progress bars and assign training
  - renderMgrRunner() read-only checklist view with approve/send-back
  - renderRunnerContent(hireId, tplIdx, readOnly) shared crew+manager runner
  - isHireComplete() helper for active vs completed classification
  - mgrState navigation state for manager tab SPA routing
  - Event delegation on #mgr-body for all manager actions
affects: [04-03-hq-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - readOnly parameter on renderRunnerContent() to disable data-action on checkboxes
    - mgrState { view, subView, activeHireId, activeTemplateIdx } mirrors obState pattern
    - assignPickerOpen module-level variable tracks which hire's picker is open
    - isHireComplete() checks all templates: requiresSignOff -> signed_off, FAQ -> FAQ_VIEWED, else isSectionComplete
    - assign-training inline picker below hire card, no modal required

key-files:
  created:
    - .planning/phases/04-onboarding-app/04-02-SUMMARY.md
  modified:
    - onboarding.html
    - sw.js

key-decisions:
  - "renderRunnerContent() is shared between crew and manager views via readOnly flag — avoids code duplication (~80 lines) while preventing manager from editing crew progress"
  - "isHireComplete() checks requiresSignOff sections against signed_off state, FAQ sections against FAQ_VIEWED — matches the same logic used by tryAdvanceSections for consistency"
  - "assign-training action patches SECTION_STATES[hire.id] directly (no delete+reinit) to preserve existing section progress when adding a new template"
  - "assignPickerOpen is a module-level var (not inside mgrState) to simplify toggling without deep-cloning mgrState on each mutation"
  - "sw.js bumped from v40 to v41 per CLAUDE.md convention (SW cache must be bumped before every deploy)"

patterns-established:
  - "readOnly flag on shared runner content function — prevents data-action on checkboxes without forking render logic"
  - "Manager approve-signoff: set signed_off + call tryAdvanceSections to chain unlock next section"
  - "Manager send-back: set active, no tryAdvanceSections call needed (section re-opens for crew)"

requirements-completed: [D-02, D-03, D-05, D-11, D-14, D-15]

# Metrics
duration: 3min
completed: 2026-04-14
---

# Phase 4 Plan 02: Manager Tab Summary

**Manager tab for onboarding.html — read-only hire checklist drill-down, Active/Completed sub-views, sign-off approve/send-back actions, and inline training assignment via shared renderRunnerContent(readOnly) refactor**

## Performance

- **Duration:** 3 min
- **Started:** 2026-04-14T02:59:48Z
- **Completed:** 2026-04-14T03:02:27Z
- **Tasks:** 1
- **Files modified:** 2 (onboarding.html updated, sw.js bumped)

## Accomplishments

- Added `mgrState` navigation state variable mirroring `obState` for manager tab SPA routing
- Added `let assignPickerOpen` to track which hire's training picker is open
- Refactored `renderRunner()` to use shared `renderRunnerContent(hireId, tplIdx, readOnly)` — crew and manager views share one render path, ~80 lines saved
- When `readOnly=true`: checkboxes have no `data-action`, no cursor:pointer; "Request Sign-Off" button hidden; approve/send-back buttons appear for pending_signoff sections
- Added `renderManager()` — branches on `mgrState.view` ('list' vs 'runner'), renders sub-tabs for Active/Completed
- Added `renderMgrList()` — filters hires by `isHireComplete()`, renders hire cards with name+role pill, start date, per-template progress bars, "X% — X/Y tasks" text, and "+ Assign Training" link
- Added `isHireComplete()` helper — checks all sections: `requiresSignOff` sections must be `signed_off`, FAQ sections must have `FAQ_VIEWED[hireId]`, others checked via `isSectionComplete()`
- Added `renderMgrRunner()` — calls `renderRunnerContent(..., true)` and injects into `#mgr-body` with "All Hires" back link
- Added event delegation on `#mgr-body` for: `switch-sub`, `open-hire`, `back-to-hires`, `approve-signoff`, `send-back`, `show-assign`, `assign-training`, `toggle-section`, `toggle-faq`, `watch-video`
- assign-training patches SECTION_STATES directly (no delete+reinit) to preserve existing progress on other templates
- Added CSS classes: `.sub-tabs`, `.sub-tabs button`, `.sub-tabs button.on`, `.hire-role`, `.assign-picker`, `.assign-row`, `.assign-row:active`
- Added empty states: "No active onboarding." and "No completed onboardings yet." and "No additional trainings available."
- Bumped sw.js v40 -> v41 per CLAUDE.md convention

## Task Commits

1. **Task 1: Add Manager tab with hire cards, Active/Completed sub-views, and drill-down** - `fe28cbe` (feat)

## Files Created/Modified

- `onboarding.html` — +210 lines net; Manager tab functions, shared renderRunnerContent, mgrState, event delegation
- `sw.js` — Cache version bumped v40 -> v41

## Decisions Made

- Shared `renderRunnerContent(readOnly)` function avoids duplicating the entire runner render path. The `readOnly` flag is the sole difference between crew and manager views — checkboxes lose `data-action` and "Request Sign-Off" is hidden.
- `isHireComplete()` uses the same completion semantics as `tryAdvanceSections`: requiresSignOff sections need `signed_off` state, non-signoff sections just need all items checked, FAQ sections need `FAQ_VIEWED[hireId]`.
- `assign-training` patches `SECTION_STATES[hire.id]` for only the new template's sections (not delete+reinit entire entry) to preserve existing crew progress on already-assigned templates.
- `assignPickerOpen` is a module-level variable (not nested in `mgrState`) to avoid deep-clone complexity on mgrState mutations.
- sw.js bumped v40 -> v41 per CLAUDE.md "SW cache must be bumped before every deploy" rule.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] FAQ toggle-faq handler in crew view needed hire-id and tpl-id**
- **Found during:** Task 1 (reviewing renderRunnerContent refactor)
- **Issue:** The original `renderRunner()` rendered `toggle-faq` items with only `data-faq-id`. When `renderRunnerContent()` was extracted as shared code used by both crew and manager runners, the `toggle-faq` rows needed `data-hire-id` and `data-tpl-id` attributes to support the manager's `toggle-faq` event handler (which may call `tryAdvanceSections` after FAQ expand). Added those data attributes to the shared renderRunnerContent output.
- **Fix:** Added `data-hire-id` and `data-tpl-id` to all `toggle-faq` rendered elements in `renderRunnerContent()`
- **Files modified:** onboarding.html
- **Commit:** fe28cbe (included in Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 2 — missing data attributes on FAQ toggle elements for manager handler compatibility)
**Impact on plan:** Corrective — prevents manager FAQ toggle from breaking due to missing hire/tpl context

## Known Stubs

None — Manager tab is fully functional. All hire data, section states, and completion logic are wired to the same in-memory state dicts (SECTION_STATES, OB_CHECKS, FAQ_VIEWED) as the crew view. State mutations (approve, send-back, assign-training) immediately reflect in re-renders.

## Issues Encountered

None — plan executed cleanly in single task.

## Next Phase Readiness

- Plan 03 (HQ integration) can wire the Onboarding tile in index.html and verify full end-to-end navigation
- All state dicts (SECTION_STATES, OB_CHECKS, FAQ_VIEWED) are shared between My Trainings and Manager tabs — state changes made in crew view are immediately visible in manager view and vice versa

## Self-Check: PASSED

- onboarding.html: FOUND
- sw.js: FOUND
- 04-02-SUMMARY.md: FOUND
- commit fe28cbe: FOUND

---
*Phase: 04-onboarding-app*
*Completed: 2026-04-14*
