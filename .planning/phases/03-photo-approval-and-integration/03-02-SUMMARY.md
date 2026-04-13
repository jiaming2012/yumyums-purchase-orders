---
phase: 03-photo-approval-and-integration
plan: 02
subsystem: ui
tags: [approval-flow, three-tab-nav, role-gating, vanilla-js, pwa]

requires:
  - phase: 03-photo-approval-and-integration/03-01
    provides: openCamera, showPhotoPreview, photo-thumb CSS, fill-body photo delegation, MOCK_RESPONSES blob URLs, FAIL_NOTES.photo

provides:
  - Three-tab navigation: My Checklists / Approvals / Builder (Builder renumbered to t3/s3)
  - PENDING_APPROVALS / APPROVED_SUBMISSIONS / REJECTED_SUBMISSIONS state dicts
  - submitChecklist routing: requires_approval=true routes to PENDING_APPROVALS, false routes to SUBMITTED_TEMPLATES
  - renderApprovals: pending summary cards with approve/reject actions
  - approvals-body click delegation: one-tap approve (toast), two-step reject (required reason)
  - Approval badge on checklist list card when pending
  - isPending state in renderRunner shows waiting-for-review message
  - checkBuilderAccess gates Approvals tab to manager+ roles, Builder to workflow_builder permission
  - Two pre-built templates: Setup Checklist (requires_approval=true) and Closing Checklist (requires_approval=false)
  - sw.js bumped to v21
affects:
  - human-verify checkpoint (Task 3 — pending)

tech-stack:
  added: []
  patterns:
    - Three-state submit: not-submitted / pending-approval / submitted — checked before any submit action
    - approvals-body click delegation mirrors fill-body pattern — one listener, data-action routing
    - Approve: moves entry from PENDING_APPROVALS to APPROVED_SUBMISSIONS, sets SUBMITTED_TEMPLATES, re-renders both tabs
    - Reject: two-step inline — first click adds textarea + confirm button, second click validates non-empty reason
    - Role gate check: isManagerPlus = role === 'admin' || role === 'manager'

key-files:
  created: []
  modified:
    - workflows.html
    - sw.js

key-decisions:
  - "PENDING_APPROVALS stores snapshot of responses/failNotes at submit time — approval flow reads snapshot, not live state"
  - "Approve action also sets SUBMITTED_TEMPLATES[tplId]=true so checklist shows 'Submitted' on return to My Checklists"
  - "sw.js bumped v20 to v21 (v20 was Plan 01's bump; Plan 02 adds more changes requiring fresh cache bust)"
  - "renderApprovals called on init — approvals-body pre-rendered even when tab hidden, so delegation listener attaches cleanly"
  - "Closing Checklist changed to requires_approval=false (per plan spec) — fireworks path for non-approval submit"

patterns-established:
  - "Three-state submit check: if (SUBMITTED_TEMPLATES[tplId] || PENDING_APPROVALS[tplId]) return;"
  - "Tab switching show(n) calls renderApprovals() when n===2 to refresh pending queue on tab open"

requirements-completed: [APRV-01, APRV-02, APRV-03, INTG-01, INTG-02, INTG-03]

duration: 8min
completed: 2026-04-13
---

# Phase 3 Plan 02: Approval Flow and Integration Summary

**Three-tab nav (My Checklists / Approvals / Builder) with manager approval flow — one-tap approve, two-step reject with required reason — two pre-built food truck templates, and role-gated tab access.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-13T16:26:00Z
- **Completed:** 2026-04-13T16:34:09Z
- **Tasks:** 2 of 3 (Task 3 is human-verify checkpoint — pending)
- **Files modified:** 2

## Accomplishments

- Added Approvals tab (t2/s2) between My Checklists and Builder; Builder renumbered to t3/s3 with all inner IDs unchanged
- Implemented full approval routing: `submitChecklist()` checks `requires_approval` flag and routes to `PENDING_APPROVALS` or `SUBMITTED_TEMPLATES`
- Approval summary cards show submitter, timestamp, completion count, fail trigger count, photo count, and photo strip
- Approve action: one-tap, moves to `APPROVED_SUBMISSIONS`, shows toast, refreshes both tabs
- Reject action: two-step inline — textarea appears, confirm validates non-empty reason, moves to `REJECTED_SUBMISSIONS`
- Pending Approval badge appears on checklist list card after submission
- Runner shows "Waiting for manager review" message when checklist is pending approval
- Approvals tab gated to manager+ roles; Builder tab gated by workflow_builder permission
- Replaced old MOCK_TEMPLATES with Setup Checklist (requires_approval=true, has photo + temperature fields) and Closing Checklist (requires_approval=false, has photo + checkbox fields)
- sw.js bumped v20 → v21 per MEMORY note (always bump before human-verify checkpoint)

## Task Commits

1. **Task 1: Three-tab restructure, approval state/rendering, submitChecklist routing, pre-built templates, role gating** - `4fce6fb` (feat)
2. **Task 2: Bump sw.js cache version and verify INTG-01** - `3dcf029` (chore)

**Task 3 (checkpoint:human-verify):** Pending — awaiting human verification.

**Plan metadata:** _(to be added after docs commit)_

## Files Created/Modified

- `workflows.html` - Three-tab HTML, approval CSS, PENDING_APPROVALS/APPROVED_SUBMISSIONS/REJECTED_SUBMISSIONS dicts, submitChecklist rewrite, renderApprovals, approvals-body delegation, renderChecklistList badge, renderRunner isPending state, show(n) extended to [1,2,3], checkBuilderAccess role gate, new MOCK_TEMPLATES
- `sw.js` - Cache version bumped v20 → v21

## Decisions Made

- PENDING_APPROVALS stores a snapshot of MOCK_RESPONSES and FAIL_NOTES at submit time so the approval card displays accurate data even if the user edits responses after submission
- Approve action sets `SUBMITTED_TEMPLATES[tplId] = true` so returning to My Checklists shows "Submitted" badge (not just "nothing")
- sw.js bumped v20 → v21 because Plan 01 already used v20; new changes require another cache bust
- `renderApprovals()` called during init (not just on tab click) so the approvals-body listener attaches cleanly from page load
- Closing Checklist `requires_approval` set to false per plan spec — fireworks + toast path on submit, no approval queue

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Merged main branch before starting — worktree was missing Plan 01 photo changes**
- **Found during:** Pre-execution setup
- **Issue:** Worktree branch `worktree-agent-a62c4d70` was based on commit `54cbaa3`, missing Plan 01 photo capture commit `36705df`. sw.js showed v19, not v20, and photo CSS/functions were absent.
- **Fix:** `git merge main --no-edit` — fast-forward to `725c72f`, bringing in all Plan 01 changes and planning artifacts
- **Files modified:** workflows.html, sw.js, .planning/ files
- **Verification:** `grep -c "photo-modal\|openCamera\|showPhotoPreview" workflows.html` returned 10; sw.js showed v20
- **Committed in:** merge commit (not a separate task commit — prerequisite setup)

**2. [Rule 2 - Missing Critical] sw.js bumped to v21 not v20**
- **Found during:** Task 2
- **Issue:** Plan says "bump v19 → v20" but Plan 01 already bumped to v20. Bumping to v20 again is a no-op — cache wouldn't invalidate.
- **Fix:** Bumped v20 → v21 per the objective instructions ("bump to v21 in Task 2")
- **Files modified:** sw.js
- **Verification:** `grep "yumyums-v21" sw.js` returns match
- **Committed in:** `3dcf029`

---

**Total deviations:** 2 auto-fixed (1 blocking merge, 1 version correction)
**Impact on plan:** Both fixes essential for correct execution. No scope creep.

## Issues Encountered

- Worktree was missing Plan 01 changes — resolved by merging main. Root cause: worktrees are created from a specific commit and don't auto-update when main advances.

## Known Stubs

None — all approval flows are fully functional with in-memory state. Photo thumbnails in approval cards use the actual blob URLs captured during fill-out. No hardcoded empty values flow to UI.

## Next Phase Readiness

- Task 3 (human-verify checkpoint) is pending — user needs to verify end-to-end in browser
- All APRV-01/02/03 and INTG-01/02/03 requirements satisfied by Tasks 1 and 2
- sw.js at v21, deployed changes will be served fresh

---
*Phase: 03-photo-approval-and-integration*
*Completed: 2026-04-13*
