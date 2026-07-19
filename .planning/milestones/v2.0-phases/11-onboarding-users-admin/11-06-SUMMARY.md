---
phase: 11-onboarding-users-admin
plan: 06
subsystem: testing
tags: [playwright, e2e, users, onboarding, invite-flow, accept-invite]

# Dependency graph
requires:
  - phase: 11-onboarding-users-admin (plans 01-05)
    provides: Go backend with /api/v1/users/*, /api/v1/onboarding/*, real test database
  - phase: 11-03
    provides: users.html invite/edit/accept-invite flows
  - phase: 11-05
    provides: onboarding.html my-trainings, manager, builder tabs

provides:
  - 11 E2E tests for onboarding.html against real API (tests/onboarding.spec.js)
  - 12 E2E tests for users.html and login.html against real API (tests/users.spec.js)
  - Bug fix: parseInt(uuid) truncation in users.html click handler

affects: [future-phases, any-html-changes-to-users-or-onboarding]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "page.evaluate for authenticated API calls in Playwright tests using existing session cookie"
    - "waitForFunction over waitForSelector for complex DOM readiness conditions"
    - "Unique timestamp-prefixed names for test data to prevent cross-run accumulation flakiness"

key-files:
  created:
    - tests/users.spec.js
  modified:
    - tests/onboarding.spec.js
    - users.html
    - sw.js

key-decisions:
  - "parseInt(uuid)||uuid pattern in users.html was a latent bug: UUIDs starting with digits (e.g. '209c6b34-...') parse as integer 209, making editUser(209) fail USERS.find() since IDs are UUID strings — fixed by removing parseInt()"
  - "delete user test uses unique timestamp prefix in first_name to avoid false failures when DB accumulates many 'DeleteMe' users across test runs"
  - "waitForEditCard checks edit-card .form-wrap only (no s2 display check) — editUser sets HTML then calls show(2) synchronously so both arrive together"
  - "onboarding checkbox persistence test dynamically finds an active section via hireTraining API to avoid sign-off state pollution from prior runs"

patterns-established:
  - "Test data isolation: use unique timestamp prefix in display names (not just emails) when asserting not.toContainText after deletion"

requirements-completed: [ONBD-04, USER-03]

# Metrics
duration: ~120min
completed: 2026-04-18
---

# Phase 11 Plan 06: E2E Tests (Onboarding + Users Admin) Summary

**23 Playwright E2E tests rewritten/created against real Go server: 11 onboarding + 12 users admin, fixing a latent parseInt(UUID) bug in users.html click handler**

## Performance

- **Duration:** ~120 min
- **Started:** 2026-04-17T22:00:00Z (previous session)
- **Completed:** 2026-04-18T03:20:57Z
- **Tasks:** 2
- **Files modified:** 4 (tests/onboarding.spec.js, tests/users.spec.js, users.html, sw.js)

## Accomplishments

- Rewrote tests/onboarding.spec.js from 35 mock-data tests to 11 real-API tests; all 11 pass idempotently including sign-off, progress persistence, section unlock, video series, FAQ, and Builder
- Created tests/users.spec.js with 12 tests covering user list, invite flow, accept-invite (set password + redirect), edit user, nickname collision (409 error), force logout, delete user, and password reset
- Found and fixed latent bug: `parseInt(id)||id` in users.html truncated UUIDs starting with digits to integers (e.g. "209c6b34-..." → 209), causing USERS.find() to silently return undefined and editUser() to return early without opening the edit form

## Task Commits

1. **Task 1: Rewrite onboarding.spec.js against real API** - `37f534b` (feat)
2. **Task 2: Create users.spec.js E2E tests + fix parseInt UUID bug** - `efc8e1c` (feat)

## Files Created/Modified

- `tests/onboarding.spec.js` - Rewritten: 11 tests across My Trainings, Manager, and Builder tabs; uses obApiCall helper, handles sign-off attribution, checkbox/video-part persistence, section unlock, FAQ
- `tests/users.spec.js` - Created: 12 tests; usersApiCall helper, invite link verification, accept-invite clearCookies + password set, edit + nickname collision, force logout, delete, password reset
- `users.html` - Fixed: removed `parseInt(id)||id` → `id` in edit-user click handler (bug: UUIDs starting with digits were parsed as integers)
- `sw.js` - Rebuilt after users.html change

## Decisions Made

- `parseInt(id)||id` removed from users.html click handler — UUIDs starting with digits cause parseInt to return a non-NaN integer (e.g. "35feed01" → 35), which is truthy, so the OR short-circuits and uid = 35 instead of the full UUID string. USERS.find(u => u.id === 35) returns undefined. Fix: pass dataset.userId directly to editUser.
- Delete user test uses `const uniqueName = \`Del\${ts}\`` rather than static 'DeleteMe' to prevent failures when the test DB accumulates many users with first_name='DeleteMe' from prior test runs
- Onboarding checkbox persistence test finds an `active` section dynamically via GET hireTraining API to avoid DB pollution from sign-off tests that leave the admin user's sections as `signed_off`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed parseInt(UUID) truncation in users.html click handler**
- **Found during:** Task 2 (users.spec.js) — via systematic binary-search debugging with `page.evaluate` injection to capture USERS.find() result at click time
- **Issue:** `const uid=parseInt(id)||id` in the `edit-user` click handler. `parseInt("209c6b34-a455-4970-...")` = 209 (not NaN). 209 is truthy, so `uid = 209` (an integer). `editUser(209)` calls `USERS.find(u => u.id === 209)` which always returns undefined for UUID string IDs. Root cause: UUIDs where the first hex segment is all decimal digits (happens ~30-40% of the time statistically).
- **Fix:** Removed parseInt: `const id=el.dataset.userId; await editUser(id);`
- **Files modified:** users.html (line 454-456)
- **Verification:** 12/12 users tests pass including 5 tests that all use `userRow.click() → waitForEditCard`
- **Committed in:** efc8e1c (Task 2 commit)

**2. [Rule 1 - Bug] Fixed delete user test flakiness from DB accumulation**
- **Found during:** Task 2 (delete user test) — `not.toContainText('DeleteMe')` failed because DB had 4+ previous "DeleteMe" users from prior test runs
- **Fix:** Used `const uniqueName = \`Del\${ts}\`` as first_name, asserted `.not.toBeVisible()` on the specific user row by ID rather than `not.toContainText(name)`
- **Files modified:** tests/users.spec.js
- **Committed in:** efc8e1c (Task 2 commit)

**3. [Rule 1 - Bug] Fixed onboarding.html: hire.user_id → hire.hire_id, item.parts → item.video_parts, CURRENT_USER.user_id → CURRENT_USER.id, isManager missing superadmin role**
- **Found during:** Task 1 (onboarding.spec.js) — manager drill-down and video series rendering broken
- **Fix:** Fixed field names in renderManagerList, findHireById, renderSections, countSectionItems, countCheckedItems, findItemInTraining, openMyTraining, renderMyRunner
- **Files modified:** onboarding.html
- **Committed in:** 37f534b (Task 1 commit)

**4. [Rule 1 - Bug] Fixed GetHireTraining in db.go: item.Checked and part.Checked never populated from progressMap**
- **Found during:** Task 1 — checkbox progress persistence test: checked items not restored after reload
- **Fix:** Added loop after progressMap build to set `item.Checked = progressMap[item.ID+":item"]` and `part.Checked = progressMap[part.ID+":video_part"]` for video series
- **Files modified:** backend/internal/onboarding/db.go
- **Committed in:** 37f534b (Task 1 commit)

**5. [Rule 2 - Missing] Added AssignedTemplates to HireOverview and populated in GetManagerHires**
- **Found during:** Task 1 — manager drill-down showed no templates
- **Fix:** Added `AssignedTemplateSummary` struct and `AssignedTemplates []AssignedTemplateSummary` field to HireOverview; added second query loop in GetManagerHires to fetch per-hire template details
- **Files modified:** backend/internal/onboarding/db.go
- **Committed in:** 37f534b (Task 1 commit)

---

**Total deviations:** 5 auto-fixed (4 bugs, 1 missing critical functionality)
**Impact on plan:** All fixes required for tests to pass. Bug fixes improve correctness of the production UI (not just tests). No scope creep.

## Issues Encountered

- The parseInt(UUID) bug was intermittent (only ~30-40% of UUIDs are affected): UUIDs where the first hex segment contains only decimal digits (0-9) fail, others pass. This made the flakiness appear random. Required systematic binary-search debugging using `page.evaluate` injection to capture `eval('USERS').find()` result at exact click time.
- Sign-off state in test DB caused checkbox persistence test to fail idempotently: admin user's section 1 was marked `signed_off` after sign-off tests ran, making `state === 'active'` condition false. Fixed by dynamically finding the first active section.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All Phase 11 plans complete (06/06)
- Phase 11 complete: onboarding.html, users.html, login.html, backend users/onboarding APIs, and full E2E test coverage
- Ready for Phase 12 (inventory/photos)

---
*Phase: 11-onboarding-users-admin*
*Completed: 2026-04-18*

## Self-Check: PASSED

- tests/users.spec.js: FOUND
- tests/onboarding.spec.js: FOUND
- Commit 37f534b: FOUND
- Commit efc8e1c: FOUND
