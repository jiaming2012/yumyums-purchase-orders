---
phase: 09-foundation-auth
plan: "04"
subsystem: auth
tags: [auth, login, fetch, cookie, seed, bcrypt, tailscale, pwa, service-worker]

# Dependency graph
requires:
  - phase: 09-foundation-auth/09-03
    provides: Auth API endpoints (POST /api/v1/auth/login, GET /api/v1/me) with httpOnly cookie sessions

provides:
  - login.html wired to real POST /api/v1/auth/login with JSON body and redirect on success
  - index.html auth guard calling GET /api/v1/me and redirecting to login.html on 401
  - backend/cmd/seed/main.go CLI tool to set superadmin password and seed hq_apps
  - SW cache bumped to v48 to push auth changes to clients

affects: [10-workflows-api, 11-onboarding-users-admin, 12-inventory-photos]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Client-side 401 guard: fetch /api/v1/me on page load, redirect to login.html if 401, allow cached page on network error"
    - "Login form: POST JSON to /api/v1/auth/login, redirect on res.ok, show error on non-ok or catch — no localStorage, cookie-only"
    - "Seed script: standalone Go CLI using pgxpool directly, bcrypt hash, UPDATE users + INSERT hq_apps ON CONFLICT DO NOTHING"

key-files:
  created:
    - backend/cmd/seed/main.go
  modified:
    - login.html
    - index.html
    - sw.js
    - backend/Makefile

key-decisions:
  - "SW version bumped to v48 from v42 (plan specified v47→v48 but actual was v42; bumped to v48 as specified target)"
  - "Seed script uses pgxpool directly (not auth.HashPassword wrapper) matching plan's exact bcrypt import"
  - "Tailscale not installed on dev machine; checkpoint returns for human to install and verify on physical iPhone"

patterns-established:
  - "Pattern 1: Offline PWA grace — catch network errors in checkAuth() and allow cached page rather than hard-fail redirect"
  - "Pattern 2: Seed script as standalone cmd — not coupled to server startup, safe to re-run (idempotent via ON CONFLICT)"

requirements-completed: [AUTH-04]

# Metrics
duration: 12min
completed: 2026-04-14
---

# Phase 09 Plan 04: Frontend Auth Wiring + Seed Script Summary

**login.html wired to POST /api/v1/auth/login with httpOnly cookie redirect; index.html 401 guard; seed script sets superadmin test password via bcrypt; SW bumped to v48**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-04-14T23:10:00Z
- **Completed:** 2026-04-14T23:22:00Z
- **Tasks:** 2 of 3 complete (Task 3 is human-verify checkpoint)
- **Files modified:** 4

## Accomplishments

- Replaced mock signIn() in login.html with real fetch POST to /api/v1/auth/login — redirects to index.html on 200, shows error on non-ok/network failure
- Added checkAuth() to index.html that fires on load, calls GET /api/v1/me, redirects to login.html on 401 (allows offline cached page on network error)
- Created backend/cmd/seed/main.go: hashes "test123" with bcrypt, UPDATE users SET password_hash + status=active for jamal@yumyums.com, seeds hq_apps if empty
- Added `seed` Makefile target for convenient dev invocation
- Bumped SW cache version to yumyums-v48 to force client pickup of auth changes
- All 128 Playwright E2E tests still pass

## Task Commits

Each task was committed atomically:

1. **Task 1: Wire login.html to auth API and add 401 guard to index.html** - `6fb5e23` (feat)
2. **Task 2: Create seed script for superadmin test user** - `f420c03` (feat)
3. **Task 3: Verify auth flow on physical iPhone via Tailscale** - CHECKPOINT (awaiting human verification)

## Files Created/Modified

- `login.html` - signIn() replaced with real fetch POST; mock alert() removed; no localStorage
- `index.html` - checkAuth() added as first function in script block; fires on load
- `sw.js` - cache version bumped from v42 to v48
- `backend/cmd/seed/main.go` - standalone seed CLI with bcrypt hash + UPDATE users + seed hq_apps
- `backend/Makefile` - added `seed:` target

## Decisions Made

- SW was v42 (not v47 as plan assumed) — bumped to v48 as plan specified for the target version
- Seed script uses `golang.org/x/crypto/bcrypt` directly rather than calling `auth.HashPassword` — equivalent behavior, matches plan's code example exactly
- Tailscale not installed; checkpoint paused for human to install + verify on iPhone

## Deviations from Plan

None - plan executed exactly as written. SW version deviation (v42→v48 vs assumed v47→v48) is harmless — net effect identical.

## Issues Encountered

- SW cache was v42, not v47 as the plan assumed from prior plans. The bump to v48 is correct regardless since v48 > v42 and will still force a full cache refresh.
- Tailscale binary not found on dev Mac — plan pre-work step cannot be completed by Claude. Checkpoint returns for human Tailscale setup.

## User Setup Required

**Tailscale must be installed and configured for iPhone verification.** See Task 3 checkpoint details:

- Install Tailscale on Mac: https://tailscale.com/download/mac
- Install Tailscale on iPhone from App Store
- Join both devices to same tailnet
- Run: `tailscale serve http://localhost:8080`
- Start server: `cd backend && make dev`
- Seed user: `cd backend && make seed`
- Test iPhone standalone mode login flow

## Known Stubs

None — login.html and index.html are now fully wired to real API calls. The seed script is a real bcrypt operation against Postgres. No hardcoded empty values flow to UI rendering.

## Next Phase Readiness

- Auth flow is fully wired frontend → backend for the happy path
- Physical iPhone verification (INFRA-03) is the only remaining gate
- Once Task 3 is approved, Phase 09 foundation-auth is complete
- Phase 10 (workflows-api) can begin immediately after checkpoint approval

## Self-Check: PASSED

All created files verified present. Both task commits verified in git log.

---
*Phase: 09-foundation-auth*
*Completed: 2026-04-14*
