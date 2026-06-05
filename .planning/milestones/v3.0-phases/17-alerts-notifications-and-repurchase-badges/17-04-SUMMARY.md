---
phase: 17-alerts-notifications-and-repurchase-badges
plan: "04"
subsystem: alerts-users-frontend
tags: [alerts, multi-channel, notification, inventory, badge-spacing, migration]
dependency_graph:
  requires: [17-03]
  provides: [multi-channel-alert-dispatch, notification-channel-array, checkbox-ui]
  affects: [users.html, inventory.html, backend/users, backend/purchasing]
tech_stack:
  added: []
  patterns:
    - "TEXT[] for multi-value DB columns (vs TEXT with CHECK)"
    - "Double-loop alert dispatch: for contact → for channel → Enqueue"
    - "CSS adjacent sibling selector (.badge+.badge) for controlled spacing"
key_files:
  created:
    - backend/internal/db/migrations/0045_notification_channel_array.sql
  modified:
    - backend/internal/users/db.go
    - backend/internal/users/handler.go
    - backend/internal/purchasing/scheduler.go
    - backend/internal/purchasing/service.go
    - users.html
    - inventory.html
    - sw.js
decisions:
  - "notification_channels JSON key is plural to signal array type at API boundary"
  - "Migration uses <@ operator (subset) for valid-values CHECK on TEXT[] column"
  - "Frontend builds channels array from checkboxes; empty array blocked with alert() before API call"
  - "Badge spacing uses CSS adjacent-sibling margin instead of space character (space renders inconsistently across contexts)"
metrics:
  duration_minutes: 12
  completed_date: "2026-04-23"
  tasks_completed: 2
  files_changed: 7
requirements: [ALRT-03]
---

# Phase 17 Plan 04: Multi-Channel Alerts and Badge Spacing Summary

**One-liner:** TEXT[] migration for notification_channel + checkbox UI enables users to select both Zoho Cliq AND email simultaneously; badge spacing fixed via CSS margin.

## What Was Built

### Task 1: Backend — notification_channel to TEXT[], multi-channel dispatch

**Migration 0045** (`backend/internal/db/migrations/0045_notification_channel_array.sql`):
- Drops existing `users_notification_channel_check` CHECK constraint
- Converts `notification_channel` from `TEXT` to `TEXT[]` using `ARRAY[notification_channel]`
- Sets default to `ARRAY['zoho_cliq']::TEXT[]`
- Adds `users_notification_channel_valid` CHECK: `notification_channel <@ ARRAY['zoho_cliq','email']`
- Adds `users_notification_channel_nonempty` CHECK: `array_length(notification_channel, 1) >= 1`
- DOWN path converts back to TEXT using `[1]` element

**`backend/internal/users/db.go`:**
- `UserRow.NotificationChannels []string` (json: `notification_channels`, plural)
- `NotificationPreferenceContact.NotificationChannels []string`
- `UpdateUserInput.NotificationPref *[]string`
- `GetNotificationPreference` returns `[]string`
- `UpdateNotificationPreference` accepts `[]string`, validates each element and length >= 1
- `UpdateUser` validates each channel element, sends `[]string` to Postgres (pgx handles TEXT[] natively)
- `GetUsersForAlerts` scans `NotificationChannels` as `[]string`

**`backend/internal/users/handler.go`:**
- `UpdateUserHandler` body struct: `NotificationPref *[]string`
- `GetNotificationPreferenceHandler` returns `{"notification_channels": [...]}`
- `UpdateNotificationPreferenceHandler` accepts/returns `notification_channels` as JSON array

**`backend/internal/purchasing/scheduler.go`:**
- `runReminderCheck`: double-loop — one `Enqueue` per channel per contact
- `runLowStockCheck`: same double-loop pattern

**`backend/internal/purchasing/service.go`:**
- `NotifyVendorComplete`: double-loop — one `Enqueue` per channel per contact

### Task 2: Frontend — checkboxes + badge spacing

**`users.html`:**
- Replaced `<select id="f-notif-pref">` dropdown with two checkboxes (`f-notif-zoho`, `f-notif-email`)
- `editUser()` pre-checks boxes from `u.notification_channels` array (with `['zoho_cliq']` fallback)
- `saveUser()` builds `notifChannels` array from checked boxes; blocks with `alert()` if empty
- Sends `notification_pref: notifChannels` (array) in PATCH body

**`inventory.html`:**
- Line 575: Removed trailing space char after `badgeHtml(s.level)`
- Line 576: Removed trailing space after repurchase badge span
- Added `.stock-badge+.stock-badge{margin-left:4px}` adjacent-sibling CSS rule for controlled gap

**`sw.js`:** Rebuilt via `node build-sw.js` with updated content hashes.

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1    | bf82c80 | feat(17-04): convert notification_channel to TEXT[], multi-channel alert dispatch |
| 2    | 6a4f765 | feat(17-04): checkboxes for multi-channel alerts + badge spacing CSS fix |

## Deviations from Plan

None — plan executed exactly as written.

## Verification

- `cd backend && go build ./cmd/server/` — PASS
- `cd backend && go vet ./...` — PASS
- `grep -n "NotificationChannels" backend/internal/users/db.go` — shows `[]string` type on lines 30, 40, 95, 119, 209
- `grep -n "for.*ch.*range.*NotificationChannels" backend/internal/purchasing/scheduler.go` — shows loops on lines 143, 337
- `grep -n "f-notif-zoho" users.html` — shows checkbox on lines 342, 363
- `grep -n "stock-badge+.stock-badge" inventory.html` — shows CSS rule on line 94
- SW rebuilt: 20 files precached (742.3 KB)

## Self-Check: PASSED

All created/modified files verified present and commits confirmed in git log.
