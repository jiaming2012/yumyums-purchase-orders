---
phase: 17-alerts-notifications-and-repurchase-badges
plan: 05
subsystem: api, database, ui
tags: [timezone, users, scheduler, postgres, go, vanilla-js, pwa]

# Dependency graph
requires:
  - phase: 17-04
    provides: notification_channels as []string on UserRow and NotificationPreferenceContact
provides:
  - timezone TEXT column on users table (migration 0046, default America/New_York)
  - UserRow.Timezone field exposed in JSON API responses
  - NotificationPreferenceContact.Timezone field from GetUsersForAlerts
  - UpdateUser accepts and validates IANA timezone via time.LoadLocation
  - runLowStockCheck uses cutoff_config timezone (not hardcoded America/Chicago)
  - Timezone dropdown in Users edit form (f-timezone select)
  - Badge reset config displays config.timezone from API (not hardcoded)
  - users.DefaultTimezone exported constant (America/New_York)
affects: [17-06, scheduler, alerts]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "users.DefaultTimezone exported constant used as fallback across packages"
    - "IANA timezone validation via time.LoadLocation at API boundary"
    - "Scheduler reads admin config for timezone instead of hardcoding"

key-files:
  created:
    - backend/internal/db/migrations/0046_user_timezone.sql
  modified:
    - backend/internal/users/db.go
    - backend/internal/users/handler.go
    - backend/internal/purchasing/scheduler.go
    - backend/internal/purchasing/handler.go
    - users.html
    - inventory.html
    - sw.js

key-decisions:
  - "users.DefaultTimezone exported as public const so purchasing package can reference it as fallback without duplication"
  - "runLowStockCheck loads cutoff_config on each tick to get timezone — same pattern as runCutoffCheck/runReminderCheck"
  - "Timezone dropdown in users.html uses inline selected check via template literal — consistent with existing notification_channels checkbox pattern"
  - "inventory.html badge reset config shows config.timezone from API response — no longer hardcoded America/Chicago"
  - "Default timezone throughout system changed from America/Chicago to America/New_York"

patterns-established:
  - "IANA timezone validation: time.LoadLocation(*input.Timezone) at DB layer, error returned as 400 from handler"
  - "Scheduler timezone fallback: load config first, fall back to users.DefaultTimezone if config nil or error"

requirements-completed: [ALRT-05, ALRT-06]

# Metrics
duration: 12min
completed: 2026-04-23
---

# Phase 17 Plan 05: Per-User Timezone Summary

**Per-user timezone column (migration 0046) with Go type updates, IANA validation, scheduler hardcode removal, and timezone dropdown in Users edit form**

## Performance

- **Duration:** 12 min
- **Started:** 2026-04-23T12:12:00Z
- **Completed:** 2026-04-23T12:24:51Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- Migration 0046 adds `timezone TEXT NOT NULL DEFAULT 'America/New_York'` to users table
- UserRow and NotificationPreferenceContact both expose Timezone field in Go and JSON
- UpdateUser validates IANA timezone via time.LoadLocation before writing to DB
- runLowStockCheck no longer uses hardcoded `America/Chicago` — loads cutoff_config timezone with America/New_York fallback
- users.html edit form has timezone dropdown with 7 options (Eastern, Central, Mountain, Pacific, Alaska, Hawaii, UTC)
- saveUser() includes timezone in PATCH body; editUser() pre-selects from user.timezone
- inventory.html badge reset config shows and sends correct timezone from API config

## Task Commits

Each task was committed atomically:

1. **Task 1: Backend — timezone column, Go types, scheduler fix** - `6675c6a` (feat)
2. **Task 2: Frontend — timezone dropdown and badge reset defaults** - `04e187b` (feat)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `backend/internal/db/migrations/0046_user_timezone.sql` - UP adds timezone column, DOWN drops it
- `backend/internal/users/db.go` - DefaultTimezone const, Timezone on UserRow/NotificationPreferenceContact/UpdateUserInput, IANA validation, updated SELECT queries
- `backend/internal/users/handler.go` - PATCH body accepts timezone field, passes to UpdateUserInput
- `backend/internal/purchasing/scheduler.go` - runLowStockCheck loads cutoff_config timezone instead of hardcoded America/Chicago
- `backend/internal/purchasing/handler.go` - UpsertRepurchaseResetConfig defaults to America/New_York
- `users.html` - Timezone dropdown in edit form, saveUser() includes timezone in PATCH
- `inventory.html` - Badge reset config shows/sends config.timezone from API
- `sw.js` - Rebuilt with new content hashes

## Decisions Made
- Exported `users.DefaultTimezone = "America/New_York"` as a public constant so the purchasing package can reference it for fallback without duplicating the string.
- `runLowStockCheck` loads `GetCutoffConfig` on each tick — same established pattern as `runCutoffCheck` and `runReminderCheck`. Low-stock is an admin concern, so it uses the admin-configured cutoff timezone.
- Inventory badge reset form still passes a timezone value to the API (currently hardcoded to America/New_York as default) since the badge reset config is a single admin setting, not per-user.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Per-user timezone is now stored and returned by the API
- Scheduler uses correct timezone for low-stock week boundary calculation
- Users tab allows admins to configure timezone per crew member
- Ready for Phase 17-06 or any follow-on alerting work that needs per-user timezone

## Self-Check: PASSED
- `backend/internal/db/migrations/0046_user_timezone.sql` — exists
- `backend/internal/users/db.go` — Timezone on UserRow, NotificationPreferenceContact, UpdateUserInput confirmed
- `backend/internal/purchasing/scheduler.go` — no America/Chicago hardcode confirmed
- `users.html` — f-timezone select confirmed
- `backend/internal/purchasing/handler.go` — America/New_York default confirmed
- Commits 6675c6a and 04e187b exist in git log

---
*Phase: 17-alerts-notifications-and-repurchase-badges*
*Completed: 2026-04-23*
