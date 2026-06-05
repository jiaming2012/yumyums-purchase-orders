---
phase: 10-workflows-api
plan: 05
subsystem: testing
tags: [playwright, e2e, testing, go-server, indexeddb, offline-sync]

# Dependency graph
requires:
  - phase: 10-workflows-api/10-04
    provides: offline sync, IndexedDB queue, drain logic, API endpoints

provides:
  - E2E test suite for workflows against real Go server
  - playwright.config.js webServer config for Go test server

affects: [ci, verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "apiCall() helper — page.evaluate fetch calls using Go API during tests"
    - "cleanupTemplates() — archive all templates before each test for isolation"
    - "createTestTemplate() — seed templates via API for test setup"
    - "context.setOffline(true/false) — Playwright browser offline simulation for sync tests"

key-files:
  created: []
  modified:
    - tests/workflows.spec.js
    - playwright.config.js

key-decisions:
  - "Admin email corrected to jamal@yumyums.kitchen — plan had wrong email (jamal@yumyums.com)"
  - "15 tests organized in 6 describe blocks: Builder, My Checklists, Approvals, Offline sync, Access control, Loading states"
  - "apiCall() uses page.evaluate to make fetch calls with session cookie context"
  - "Idempotency test uses direct fetch to verify 200 or 409 (not 500) on duplicate submit"

requirements-completed: [WKFL-04, SYNC-01, SYNC-02, SYNC-03]

# Metrics
duration: 2min
completed: 2026-04-15
---

# Phase 10 Plan 05: E2E Test Rewrite Summary

**Playwright test suite rewritten against real Go server — 54 mock-data tests deleted, 15 new full-stack E2E tests covering Builder CRUD, checklist fill/submit, approval flow, offline sync, and access control**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-04-15T16:27:47Z
- **Completed:** 2026-04-15T16:29:43Z
- **Tasks:** 1 of 2 (Task 2 is a human-verify checkpoint)
- **Files modified:** 2 (tests/workflows.spec.js, playwright.config.js)

## Accomplishments

- Deleted all 54 existing mock-data E2E tests (D-19 compliant)
- Updated `playwright.config.js` `webServer` block to start Go backend with `hq_test` database, checking `/api/v1/health` for readiness
- Wrote 15 new E2E tests against real Go server covering:
  - Builder: create template, empty state, edit existing, archive template
  - My Checklists: API-driven checklist list, fill + auto-save + submit, empty state
  - Approvals: approve submission, reject item with comment, empty state
  - Offline sync: offline queue in IndexedDB with drain-on-reconnect, idempotency key dedup
  - Access control: superadmin Builder access
  - Loading states: skeleton screens, tab rendering

## Task Commits

1. **Task 1: Configure Playwright for real Go server and rewrite test suite** - `ec38178` (feat)

## Files Created/Modified

- `tests/workflows.spec.js` — All 54 mock-data tests deleted; 15 new full-stack E2E tests
- `playwright.config.js` — webServer updated to Go server with hq_test DB

## Decisions Made

- Admin email corrected to `jamal@yumyums.kitchen` per superadmins.yaml — plan had incorrect `jamal@yumyums.com`
- `apiCall()` helper uses `page.evaluate` to call Go API endpoints with the browser's session cookie context (no need for separate HTTP clients)
- `cleanupTemplates()` archives all existing templates before each test group to ensure test isolation
- Idempotency test verifies the server returns 200 or 409 on duplicate submit (not 500) — confirms dedup is working without asserting specific success vs. idempotent behavior
- `context.setOffline(true/false)` used for offline simulation — Playwright's built-in network isolation, which works with IndexedDB persistence

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected admin email from plan**
- **Found during:** Task 1 (reading superadmins.yaml)
- **Issue:** Plan specified `ADMIN_EMAIL = 'jamal@yumyums.com'` but `backend/config/superadmins.yaml` has `email: jamal@yumyums.kitchen`
- **Fix:** Used `jamal@yumyums.kitchen` in the test helpers
- **Files modified:** tests/workflows.spec.js
- **Commit:** ec38178

## Known Stubs

None introduced in this plan. Pre-existing stubs from Plans 02-03 remain:
- `handlePhotoCaptureClick()` — no-op stub (Phase 12)
- `unsubmit` action — shows "not yet available" toast

## Status

**Paused at checkpoint.** Task 2 (human-verify) requires:
1. Starting the Go server and running `npm test` to verify all tests pass
2. Physical phone verification of end-to-end workflow on Tailscale

## Self-Check: PASSED

- tests/workflows.spec.js exists and contains 15 `test(` calls with 0 MOCK_TEMPLATES references
- playwright.config.js contains `webServer`, `go run`, and `/api/v1/health`
- Commit ec38178 exists in git log
