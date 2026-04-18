---
phase: quick
plan: 260417-x0g
subsystem: frontend/auth
tags: [index, auth, logout, me-endpoint]
key-files:
  modified:
    - index.html
    - sw.js
  created:
    - tests/index.spec.js
decisions:
  - "renderUserHeader only called on successful /api/v1/me response (not on error or 401)"
  - "logout() always redirects to /login.html regardless of POST response status"
metrics:
  duration: ~5 min
  completed: "2026-04-18T03:49:00Z"
  tasks: 2
  files: 3
---

# Quick Task 260417-x0g: Add User Display Name and Logout Button

**One-liner:** User greeting ("Hi, {name}") and Log out button in index.html header, wired to /api/v1/me + /api/v1/auth/logout with 3 E2E tests proving session destruction.

## What Was Done

### Task 1: User greeting header and logout button in index.html
- Extended `checkAuth()` to parse JSON from `/api/v1/me` and extract `display_name`
- Added `renderUserHeader(name)` that injects a `.user-bar` div between the tagline and grid
- Added `logout()` async function — POSTs to `/api/v1/auth/logout`, then navigates to `/login.html` regardless of response
- Added 4 CSS rules for `.user-bar`, `.greeting`, `.logout-btn`, `.logout-btn:active`
- Rebuilt `sw.js` with updated content hash for `index.html`

### Task 2: E2E tests for logout state-flush
Created `tests/index.spec.js` with 3 tests:
1. Display name visible in `.greeting` after login
2. `#btn-logout` click redirects to `login.html`
3. Revisiting `index.html` after logout redirects to `login.html` (proves server-side session destroyed)

All 3 tests passed on first run.

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- `/Users/jamal/projects/yumyums/hq/index.html` — modified (user-bar CSS + renderUserHeader + logout functions)
- `/Users/jamal/projects/yumyums/hq/tests/index.spec.js` — created (3 E2E tests)
- `/Users/jamal/projects/yumyums/hq/sw.js` — rebuilt with new content hash
- Commits: `3de8271` (feat), `5edc1e1` (test)
