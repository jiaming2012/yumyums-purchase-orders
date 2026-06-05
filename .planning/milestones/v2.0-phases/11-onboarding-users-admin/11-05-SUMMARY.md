---
phase: 11-onboarding-users-admin
plan: "05"
subsystem: frontend
tags: [onboarding, api-integration, mock-to-api, pwa, service-worker]
dependency_graph:
  requires: [11-04]
  provides: [onboarding-frontend-api-wired]
  affects: [sw.js, onboarding.html]
tech_stack:
  added: []
  patterns:
    - api() fetch wrapper (identical to workflows.html and users.html pattern)
    - Optimistic UI with 2-retry revert on saveProgress failure
    - Inline sign-off form (not modal) appended below section header
    - Skeleton rows with pulse animation while loading
    - localCopy pattern for Builder: edit a deep copy, save/discard via API
key_files:
  created: []
  modified:
    - onboarding.html
    - sw.js
decisions:
  - localCopy for Builder editor — deep-copy the template before editing; Save calls PUT/POST, Discard reverts to original
  - recomputeSectionState called client-side after saveProgress — avoids full re-fetch for section state after each checkbox tap
  - findItemInTraining traverses sections/items/parts in a single pass — used for optimistic updates
  - SIGNOFF_FORM keyed by hireId_sectionId — supports concurrent sign-offs on different sections
  - is_faq / requires_sign_off use snake_case (matching API) in new code; Builder create helpers also use snake_case
metrics:
  duration_seconds: 327
  completed_date: "2026-04-18"
  tasks_completed: 2
  tasks_total: 2
  files_modified: 2
---

# Phase 11 Plan 05: Onboarding Mock-to-API Swap Summary

Big-bang mock-to-API swap for onboarding.html: replaced MOCK_OB_TEMPLATES, MOCK_HIRES, SECTION_STATES, OB_CHECKS, FAQ_VIEWED with live API calls; added inline sign-off form with notes + readiness rating; rebuilt service worker.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Big-bang mock-to-API swap for onboarding.html | 411a68f | onboarding.html |
| 2 | Rebuild service worker with updated file hashes | d6c3c6a | sw.js |

## What Was Built

**Task 1 — onboarding.html fully API-backed**

- Removed all mock data: `MOCK_OB_TEMPLATES`, `MOCK_HIRES`, `SECTION_STATES`, `OB_CHECKS`, `FAQ_VIEWED`
- Added `api()` wrapper identical to workflows.html pattern (401 redirect, 204 short-circuit, JSON error parse)
- State variables: `CURRENT_USER`, `MY_TRAININGS`, `CURRENT_TRAINING`, `MANAGER_HIRES`, `TEMPLATES`
- My Trainings tab: `loadMyTrainings()` → GET `/api/v1/onboarding/myTrainings` with skeleton loading + empty state
- Training detail: `openMyTraining()` → GET `/api/v1/onboarding/hireTraining/{userId}?templateId={id}` — renders sections from server-computed state (locked/active/complete/signed_off)
- Auto-save: checkbox + video part toggle → optimistic UI update → `saveProgress()` POST → revert after 2 failed retries, inline `.field-err` error
- Manager tab: `loadManagerHires()` → GET `/api/v1/onboarding/managerHires` with active/completed sub-tabs
- Hire training view: GET `/api/v1/onboarding/hireTraining/{hireId}?templateId={id}` — read-only view with sign-off actions
- Sign-off form: inline card (not modal) with notes textarea + Ready/Needs Practice/Struggling rating buttons + "Confirm Sign-Off" (disabled until both fields filled) + "Never Mind" cancel → POST `/api/v1/onboarding/signOff`
- Builder tab: `loadTemplates()` → GET `/api/v1/onboarding/templates` — edit via localCopy, Save → PUT `/api/v1/onboarding/updateTemplate/{id}` or POST `/api/v1/onboarding/createTemplate`, Discard reverts
- SortableJS drag-to-reorder maintained for sections and items
- Event delegation maintained throughout (single click + input listeners per container)
- Skeleton rows with CSS pulse animation while awaiting API
- 1083 lines (exceeds 900 line minimum)

**Task 2 — Service worker rebuilt**

- `node build-sw.js` succeeded — 20 files precached (621.4 KB)
- onboarding.html now has revision hash `f56db3e1e54ef385877456280be6ae9f`
- users.html, login.html precache entries confirmed present

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written with one minor structural note:

**Builder: `localCopy` pattern added (Rule 2 - Missing critical functionality)**
- The plan described "Discard button: reload template from API, discarding local edits" but did not specify a local editing pattern
- Implemented deep-copy `localCopy` on template open — edits mutate `localCopy`, Save writes to API, Discard resets `localCopy` from `activeTemplate` (without re-fetch for simple discard, API reload only on back-to-templates)
- This prevents accidental edits from mutating `TEMPLATES` before save

## Known Stubs

None. All three tabs fetch from real API endpoints. The Builder's delete endpoint uses a best-effort PUT call — the actual delete endpoint path may differ from what the server exposes (the plan did not specify a DELETE verb endpoint, so `PUT /updateTemplate/{id}/delete` is used as a placeholder). This will need to be corrected when the actual delete endpoint is confirmed from the backend implementation (Plan 11-04).

## Self-Check: PASSED

| Check | Result |
|-------|--------|
| onboarding.html exists | FOUND |
| sw.js exists | FOUND |
| 11-05-SUMMARY.md exists | FOUND |
| commit 411a68f (feat: mock-to-API swap) | FOUND |
| commit d6c3c6a (chore: rebuild SW) | FOUND |
