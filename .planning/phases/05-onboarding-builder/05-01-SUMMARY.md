---
phase: 05-onboarding-builder
plan: "01"
subsystem: onboarding-builder
tags: [builder, template-crud, section-management, role-gating]
dependency_graph:
  requires: []
  provides: [builder-tab-infrastructure, template-list, template-editor, section-crud]
  affects: [onboarding.html, sw.js]
tech_stack:
  added: []
  patterns: [event-delegation, state-first-rendering, role-gating]
key_files:
  created: []
  modified:
    - onboarding.html
    - sw.js
decisions:
  - "Builder tab added as third tab (t3/s3) using existing show(n) pattern — no new UI primitives needed"
  - "Role gate hides t3 for non-managers alongside t2 (same isManager() check)"
  - "obBuilderState is separate from obState/mgrState — avoids collision with existing view state machines"
  - "renderOBBuilder called on show(3) and on initial page load — consistent with renderMyTrainings/renderManager pattern"
  - "Section placeholder bodies ('Items — managed in Plan 02') document intent clearly for next plan"
metrics:
  duration: "~7 min"
  completed_date: "2026-04-14T07:09:34Z"
  tasks_completed: 1
  files_modified: 2
---

# Phase 05 Plan 01: Onboarding Builder Tab Infrastructure Summary

**One-liner:** Builder tab with role-gated access, template list CRUD, and section management (sign-off + FAQ toggles) using in-memory mutation + state-first rendering.

## What Was Built

Added the Builder tab infrastructure to `onboarding.html`:

- **Third tab** (`#t3`) added to the tabs bar, hidden for non-managers via `isManager()` check
- **`show(n)` updated** from `[1,2]` to `[1,2,3]` loop, calls `renderOBBuilder()` on switch to tab 3
- **Builder state** `obBuilderState = { view: 'list', activeTemplate: null }` — separate from existing `obState`/`mgrState`
- **Helper functions:** `generateOBId(prefix)`, `createNewOBTemplate(name, role)`, `createNewOBSection(title)`, `escapeOBAttr(str)`
- **`renderOBBuilder()`** — top-level dispatcher between list and editor views
- **`renderOBTemplateList()`** — shows template cards (name, role label, section count) with empty state; "+ New Template" button at bottom
- **`renderOBEditor()`** — template name input, role selector dropdown, sections list, "+ Add Section", "Delete Template" danger button
- **`renderOBSection(sec)`** — section card with SO/FAQ toggle pills, title input, placeholder body for items/Q&A
- **Event delegation** — three listeners on `#builder-body`: click (CRUD actions), input (name/title without re-render), change (role select with re-render)
- **SW cache** bumped from v41 to v42

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Builder tab scaffold, template list, and CRUD | ba82e4f | onboarding.html, sw.js |

## Acceptance Criteria — All Met

- [x] `<button id="t3" onclick="show(3)">Builder</button>` in HTML
- [x] `<div id="builder-body"></div>` in HTML
- [x] `function renderOBBuilder()` present
- [x] `function renderOBTemplateList()` present
- [x] `function renderOBEditor()` present
- [x] `function renderOBSection(` present
- [x] `function createNewOBTemplate(` present
- [x] `function createNewOBSection(` present
- [x] `obBuilderState` present
- [x] `data-action="new-template"` present
- [x] `data-action="open-template"` present
- [x] `data-action="delete-template"` present
- [x] `data-action="toggle-signoff"` present
- [x] `data-action="toggle-faq-mode"` present
- [x] `data-action="add-ob-section"` present
- [x] `data-action="delete-ob-section"` present
- [x] `data-action="back-to-templates"` present
- [x] `[1,2,3].forEach` in show() function
- [x] `t3` tab hidden for non-manager
- [x] 54 existing Playwright tests pass — no regressions

## Deviations from Plan

**1. [Rule 2 - Missing Critical Functionality] Combined t2 and t3 role-gate into single isManager() block**
- **Found during:** Task 1
- **Issue:** Plan spec showed `t3` hidden separately from `t2`, but both use the same `isManager()` check
- **Fix:** Combined into one `if (!isManager())` block that hides both `t2` and `t3`, cleaner and consistent
- **Files modified:** onboarding.html
- **Commit:** ba82e4f

None otherwise — plan executed as written.

## Known Stubs

| Stub | File | Reason |
|------|------|--------|
| "Items — managed in Plan 02" placeholder in `renderOBSection` | onboarding.html | Item types (Checkbox, Video Series) are Plan 02 scope |
| "Q&A items — managed in Plan 02" placeholder in `renderOBSection` | onboarding.html | FAQ Q&A editor is Plan 02 scope |

These stubs are intentional — they document which plan will wire the real content. The Builder tab goal for Plan 01 (template list CRUD and section management) is fully achieved.

## Self-Check: PASSED

- [x] `onboarding.html` exists and contains all required functions
- [x] `sw.js` updated to v42
- [x] Commit ba82e4f exists
- [x] 54/54 Playwright tests pass
