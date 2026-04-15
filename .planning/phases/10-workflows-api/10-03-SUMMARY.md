---
phase: 10-workflows-api
plan: 03
subsystem: ui
tags: [vanilla-js, fetch-api, skeleton-screens, auto-save, debounce, pwa]

# Dependency graph
requires:
  - phase: 10-workflows-api/10-02
    provides: Go REST API endpoints for workflow CRUD, submissions, approvals

provides:
  - workflows.html fully wired to live API — no mock data remains
  - async api() wrapper with 401 redirect, 204/error handling
  - autoSaveField() with 400ms debounce per-field auto-save
  - skeleton screen CSS + renderSkeletons() for all 3 tabs
  - inline error + retry UI pattern
  - photo placeholder (D-26)
  - empty states for My Checklists, Builder, Approvals tabs
  - loadMyChecklists / loadTemplates / loadPendingApprovals data loaders

affects: [10-04, 10-05, testing, onboarding-api]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "api() wrapper: async fetch with 401 redirect, 204 short-circuit, JSON error parse"
    - "autoSaveField(): debounce 400ms, optimistic FIELD_RESPONSES write, POST saveResponse"
    - "loadX() pattern: renderSkeletons → api() → renderX() or renderError()"
    - "Tab switch triggers fresh data load (loadMyChecklists/loadPendingApprovals/loadTemplates)"
    - "CURRENT_USER loaded from /api/v1/me on page init before any render"

key-files:
  created: []
  modified:
    - workflows.html
    - sw.js

key-decisions:
  - "FIELD_RESPONSES replaces MOCK_RESPONSES — same in-memory optimistic shape, backed by autoSaveField on every interaction"
  - "renderApprovals() works from PENDING array (server shape: {id, template_name, submitted_by_name, fields[]}) not old dict"
  - "Assignment picker simplified to role-only (users list not loaded in this phase)"
  - "handlePhotoCaptureClick() is a no-op stub — photo upload deferred to Phase 12 per D-26"
  - "unsubmit action shows toast 'not yet available' — server-side unsubmit not implemented in 10-02"
  - "renderBuilder() checks CURRENT_USER.role for admin/superadmin before showing template list (D-11)"
  - "SW bumped from v49 to v50 per CLAUDE.md convention"

patterns-established:
  - "Empty state constants (EMPTY_MY, EMPTY_BUILDER, EMPTY_APPROVALS, ADMIN_RESTRICTED) used in all 3 tab render functions"
  - "getUserName() / getUserInitials() helpers handle null CURRENT_USER gracefully"

requirements-completed: [WKFL-04]

# Metrics
duration: 8min
completed: 2026-04-15
---

# Phase 10 Plan 03: API Wiring Summary

**workflows.html rewritten from mock data to live API — zero MOCK_ constants remain, all 3 tabs load from Go REST endpoints with skeleton screens, auto-save, and inline error retry**

## Performance

- **Duration:** 8 min
- **Started:** 2026-04-15T16:51:21Z
- **Completed:** 2026-04-15T16:59:30Z
- **Tasks:** 2
- **Files modified:** 2 (workflows.html, sw.js)

## Accomplishments
- Deleted all 11 mock data constants (MOCK_TEMPLATES, MOCK_RESPONSES, MOCK_CURRENT_USER, MOCK_USERS, PENDING_APPROVALS, APPROVED_SUBMISSIONS, REJECTED_SUBMISSIONS, SUBMITTED_TEMPLATES, REJECTION_FLAGS, WAS_REJECTED, FAIL_NOTES kept for local state)
- Added api() wrapper, autoSaveField() with 400ms debounce, renderSkeletons(), renderError() infrastructure
- All 3 tabs (My Checklists, Approvals, Builder) load data fresh from API on each tab visit
- Builder save/archive use PUT/POST/DELETE API calls with admin restriction (D-11)
- submitChecklistToAPI() with crypto.randomUUID() idempotency key (D-15)
- Photo fields show "Photo upload coming soon" placeholder (D-26)
- approveSubmission and per-item rejectItem use API calls
- Page initializes by fetching /api/v1/me for CURRENT_USER

## Task Commits

Each task was committed atomically:

1. **Task 1: Add API wrapper, skeleton screens, and error handling infrastructure** - `699dd22` (feat)
2. **Task 2: Big-bang mock data swap** - `98b6d13` (feat)

## Files Created/Modified
- `workflows.html` - Full API wiring — zero MOCK_ constants, all data from REST endpoints
- `sw.js` - Cache version bumped v49 → v50

## Decisions Made
- renderApprovals() adapted to server submission shape (fields array on submission, not per-tplId dict)
- Assignment picker simplified to role-only (individual user list not loaded — future phase)
- handlePhotoCaptureClick() is a no-op stub per D-26 (Phase 12 deferred)
- Unsubmit shows "not yet available" toast (server-side endpoint not in Plan 02 scope)
- getUserName() / getUserInitials() helpers added to handle null CURRENT_USER gracefully during init
- loadX() pattern established: renderSkeletons → api() → renderX() or renderError(container, msg, retryFn)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added null guard for CURRENT_USER throughout**
- **Found during:** Task 2
- **Issue:** checkBuilderAccess() and renderBuilder() called before /api/v1/me response returns — would throw on null
- **Fix:** getUserName()/getUserInitials() helpers with null guard; init() async IIFE loads user before first render
- **Files modified:** workflows.html
- **Committed in:** 98b6d13

**2. [Rule 2 - Missing Critical] SW cache bumped to v50**
- **Found during:** Task 2
- **Issue:** CLAUDE.md mandates SW cache bump before every deploy and human-verify checkpoint
- **Fix:** Bumped sw.js CACHE from yumyums-v49 to yumyums-v50
- **Files modified:** sw.js
- **Committed in:** 98b6d13

---

**Total deviations:** 2 auto-fixed (both Rule 2 — missing critical)
**Impact on plan:** Necessary for correctness and project conventions. No scope creep.

## Known Stubs

- `handlePhotoCaptureClick()` — no-op stub, returns immediately. Photo upload deferred to Phase 12 per D-26. Photo fields render `.photo-placeholder` div instead.
- `unsubmit` action — shows toast "Unsubmit not yet available". Server-side unsubmit endpoint not implemented in Plan 02.
- Assignment picker — role-only (no individual user picker). Users list not fetched in this phase.

## Issues Encountered
- renderApprovals() shape mismatch: old code used PENDING_APPROVALS as dict keyed by tplId; server returns array of submission objects with `fields[]`. Rewrote render function to match server shape.

## Next Phase Readiness
- workflows.html is fully API-wired — ready for Plan 04 (Playwright test rewrite against real server)
- Server must be running with seeded templates for UI to work end-to-end
- Photo upload integration ready for Phase 12

---
*Phase: 10-workflows-api*
*Completed: 2026-04-15*
