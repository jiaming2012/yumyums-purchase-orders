---
phase: 04-onboarding-app
plan: 01
subsystem: ui
tags: [vanilla-js, pwa, service-worker, state-machine, event-delegation, checklist, onboarding]

# Dependency graph
requires: []
provides:
  - onboarding.html with My Trainings tab — crew-facing onboarding checklist runner
  - MOCK_OB_TEMPLATES with 3 templates (line cook, cashier, food safety)
  - SECTION_STATES state machine (locked/active/pending_signoff/signed_off)
  - Sequential section unlock logic via tryAdvanceSections
  - FAQ gate enforcement via FAQ_VIEWED dict
  - OB_CHECKS tracking for checkbox items and video parts
  - Onboarding tile in index.html launcher grid
  - sw.js cache bumped to v40 with onboarding.html added
affects: [04-02-manager-tab, 04-03-hq-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - SECTION_STATES[hireId][sectionId] state machine for sequential unlocking
    - OB_CHECKS[itemId] flat dict for checkbox + video part completion tracking
    - FAQ_VIEWED[hireId] boolean gate for FAQ section completion
    - tryAdvanceSections() as single transition point for section state changes
    - obState.view ('list'|'runner') for SPA-style navigation within a single page

key-files:
  created:
    - onboarding.html
    - .planning/phases/04-onboarding-app/04-01-SUMMARY.md
  modified:
    - sw.js
    - index.html

key-decisions:
  - "Manager tab (#mgr-body) intentionally left empty — Plan 02 adds Manager tab logic on top of this scaffold"
  - "initSectionStates is idempotent (guard on SECTION_STATES[hire.id]) so calling it multiple times for all hires is safe"
  - "Video Watch links open via window.open(_blank) — checkbox toggling is independent so crew can mark watched without clicking"
  - "Onboarding tile replaces the Hiring Soon tile in index.html launcher grid (Hiring placeholder removed)"
  - "sw.js bumped from v39 to v40 to cache-bust onboarding.html into service worker"

patterns-established:
  - "SECTION_STATES[hireId][sectionId] — section state machine keyed by hire+section pair"
  - "FAQ gate: expand FAQ section -> set FAQ_VIEWED[hireId]=true -> call tryAdvanceSections"
  - "obState navigation: mutate obState.view + activeHireId + activeTemplateIdx, then re-render"
  - "Event delegation on #my-body — ONE listener, all actions routed via data-action attributes"

requirements-completed: [D-01, D-04, D-06, D-07, D-08, D-09, D-10, D-12, D-13, D-16]

# Metrics
duration: 4min
completed: 2026-04-14
---

# Phase 4 Plan 01: Onboarding App Scaffold Summary

**Crew-facing onboarding HTML page with sequential section state machine, checkbox+video-series items, FAQ gate, and sign-off request flow — all event-delegated and state-first rendered**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-14T02:52:34Z
- **Completed:** 2026-04-14T02:56:07Z
- **Tasks:** 1
- **Files modified:** 3 (onboarding.html created, sw.js bumped, index.html updated)

## Accomplishments

- Created onboarding.html (534 lines) with full My Trainings tab — list view and runner view
- Implemented SECTION_STATES state machine: locked -> active -> pending_signoff -> signed_off with sequential unlock logic
- Implemented 3 mock templates (Line Cook, Cashier, Food Safety) with checkbox items, video_series items, and FAQ sections
- FAQ gate enforced via FAQ_VIEWED[hireId] — FAQ section must be expanded before checklist can be considered complete
- Request Sign-Off button appears only when all items in a requiresSignOff section are checked (active state only)
- Watch Video opens URL in new tab independently of checkbox state (no accidental auto-check)
- Manager tab role-gated: hidden when MOCK_CURRENT_USER.role is not admin/manager/superadmin
- Added Onboarding tile to index.html launcher grid (replaced Hiring Soon placeholder)
- Bumped sw.js cache to v40 and added onboarding.html to ASSETS

## Task Commits

1. **Task 1: Create onboarding.html scaffold with data model and My Trainings tab** - `b7f1dba` (feat)

## Files Created/Modified

- `onboarding.html` — Standalone onboarding tool, 534 lines, My Trainings tab with full runner logic
- `sw.js` — Cache version bumped v39 -> v40, onboarding.html added to ASSETS list
- `index.html` — Hiring Soon tile replaced with active Onboarding tile linking to onboarding.html

## Decisions Made

- Manager tab (#mgr-body) intentionally left empty — Plan 02 will add Manager tab logic on top of this scaffold
- `initSectionStates` uses early-return guard (`if (SECTION_STATES[hire.id]) return`) making it safe to call for all hires at init time
- Video Watch links use `window.open(url, '_blank')` — completely independent from the checkbox toggle action
- Hiring "Soon" tile in index.html replaced with Onboarding active tile (the tile previously said "Candidates and onboarding" making it the most natural replacement)
- sw.js bumped from v39 to v40 per CLAUDE.md convention (SW cache must be bumped before deploy)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added sw.js and index.html updates alongside onboarding.html**
- **Found during:** Task 1 (onboarding.html creation)
- **Issue:** CLAUDE.md "Adding a New Tool" specifies 3 required steps: create page, add index.html tile, add to sw.js ASSETS and bump cache. The plan only mentioned creating onboarding.html but not the companion file updates.
- **Fix:** Updated sw.js (v39->v40, added ./onboarding.html to ASSETS) and index.html (Hiring Soon tile -> Onboarding active tile)
- **Files modified:** sw.js, index.html
- **Verification:** sw.js contains 'yumyums-v40' and './onboarding.html'; index.html has active tile href="onboarding.html"
- **Committed in:** b7f1dba (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (Rule 2 — missing critical companion file updates per CLAUDE.md)
**Impact on plan:** Required for PWA correctness — without sw.js update, onboarding.html would not be cached; without index.html update, no entry point from HQ launcher.

## Known Stubs

- **Manager tab (#mgr-body):** Empty div — intentionally deferred to Plan 02. The Manager tab button is visible to admin/manager/superadmin users but tapping it shows an empty section. This is documented in the plan: "Do NOT render anything into #mgr-body — Plan 02 will add the Manager tab logic."

## Issues Encountered

None — plan executed cleanly.

## Next Phase Readiness

- Plan 02 (Manager tab) can build directly on top of this scaffold — all MOCK_HIRES, MOCK_OB_TEMPLATES, SECTION_STATES, OB_CHECKS are already initialized at page load for all hires
- Manager tab button is present (id="t2"), visible to admins, and show() function already handles tab switching
- SECTION_STATES, OB_CHECKS, FAQ_VIEWED dicts are shared module-level state — Manager tab can read and mutate them directly

---
*Phase: 04-onboarding-app*
*Completed: 2026-04-14*
