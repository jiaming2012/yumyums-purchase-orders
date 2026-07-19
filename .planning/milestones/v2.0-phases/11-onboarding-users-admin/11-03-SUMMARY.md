---
phase: 11-onboarding-users-admin
plan: 03
subsystem: ui
tags: [vanilla-js, users, auth, invite-flow, api-integration]

# Dependency graph
requires:
  - phase: 11-02
    provides: "User admin REST API endpoints (invite, edit, reset-password, revoke, delete, apps/permissions)"
provides:
  - "users.html fully wired to API — no mock data"
  - "Invite flow with copy-link panel"
  - "Edit form with nickname collision error display"
  - "Force logout, reset password, delete user actions"
  - "Access tab with per-app role toggles and user grants via PUT"
  - "login.html accept-invite mode triggered by ?token= URL param"
  - "Set-password form with validation and redirect to index.html"
affects:
  - 11-04
  - 11-05
  - onboarding

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "api() wrapper pattern with 401 redirect — same as workflows.html"
    - "Skeleton loading rows with CSS pulse animation"
    - "Event delegation via data-action attributes throughout users.html"
    - "Dual-mode login.html: normal login when no token, accept-invite when ?token= present"

key-files:
  created: []
  modified:
    - users.html
    - login.html
    - sw.js

key-decisions:
  - "Event delegation replaces old inline onclick handlers — consistent with workflows.html pattern"
  - "show() function updated to render access tab for editingUser context when switching to tab 3"
  - "invite-done and back-to-edit as separate data-action values to handle both invite and reset-password link panels"
  - "navigator.clipboard.writeText with execCommand fallback for older browsers"
  - "login.html fetches invite-info on page load before showing form to get first_name for welcome heading"

patterns-established:
  - "Dual-mode page: single HTML file shows different UX based on URL params"
  - "Invite link panel reused for both new-user invite and reset-password link"

requirements-completed: [USER-03]

# Metrics
duration: 4min
completed: 2026-04-18
---

# Phase 11 Plan 03: Users UI API Wiring + Accept-Invite Summary

**users.html fully API-backed with invite/edit/access management, and login.html extended with token-based set-password flow**

## Performance

- **Duration:** 4 min
- **Started:** 2026-04-18T01:13:48Z
- **Completed:** 2026-04-18T01:18:00Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Replaced all mock data arrays in users.html (USERS, APPS, DEFAULT_PERMS, USER_GRANTS) with real API calls
- Implemented complete invite, edit, access management flows with skeleton loading and error states
- Added accept-invite mode to login.html: detects ?token=, fetches first_name, shows set-password form, redirects on success

## Task Commits

1. **Task 1: Big-bang mock-to-API swap for users.html** - `c9eb832` (feat)
2. **Task 2: Add accept-invite mode to login.html** - `165ff01` (feat)

## Files Created/Modified

- `/Users/jamal/projects/yumyums/hq/users.html` - Fully API-backed user admin UI with invite/edit/access flows
- `/Users/jamal/projects/yumyums/hq/login.html` - Dual-mode: normal login + accept-invite set-password form
- `/Users/jamal/projects/yumyums/hq/sw.js` - Rebuilt with updated content hashes

## Decisions Made

- Event delegation via `data-action` attributes used throughout users.html for consistency with workflows.html
- Access tab renders for `editingUser` context — show(3) checks if editingUser is set and calls renderAccess()
- Separate `invite-done` and `back-to-edit` data-action values distinguish returning from invite link panel vs reset password panel
- `navigator.clipboard.writeText` with `document.execCommand('copy')` fallback for environments without clipboard API
- login.html fetches `/api/v1/auth/invite-info?token=` on page load to get first_name before showing the welcome form

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- users.html and login.html are fully wired to Phase 11-02 API endpoints
- Accept-invite flow complete: invite link generated in users.html, accepted in login.html
- Ready for Phase 11-04 (onboarding API wiring) or Phase 11-05 (E2E tests)

---
*Phase: 11-onboarding-users-admin*
*Completed: 2026-04-18*
