---
phase: "17"
plan: "02"
subsystem: backend, frontend
tags: [alerts, low-stock, notification-preference, scheduler, users]
completed: "2026-04-23"
duration_minutes: 12

dependency_graph:
  requires: [phase-17-plan-01-alert-infrastructure]
  provides: [low-stock-alerts, notification-pref-in-edit-form, patch-endpoint-notification-pref]
  affects: [purchasing-scheduler, users-tab, users-api]

tech_stack:
  added:
    - "low_stock_alert_log table: per-item weekly deduplication for low-stock alerts"
    - "inventory package imported into scheduler for ClassifyStockLevel"
  patterns:
    - "INSERT ON CONFLICT DO NOTHING for idempotent per-item per-week low-stock alert deduplication"
    - "notification_pref accepted in PATCH /users/{id} body alongside other profile fields"
    - "select#f-notif-pref pre-populated from user.notification_channel in editUser()"

key_files:
  created:
    - backend/internal/db/migrations/0044_low_stock_alert_log.sql
  modified:
    - backend/internal/purchasing/scheduler.go
    - backend/internal/users/db.go
    - backend/internal/users/handler.go
    - users.html
    - sw.js

decisions:
  - "Low-stock alert uses batch message: all new low items in one alert per week rather than one alert per item — avoids notification spam"
  - "Reuse alerts.TypeShoppingComplete audience (admins only) for low-stock alerts — only admins need to act on stock issues"
  - "notification_pref added to PATCH /users/{id} (not a separate endpoint) to keep edit form a single save action"

metrics:
  duration_minutes: 12
  completed: "2026-04-23"
  tasks_completed: 2
  files_modified: 5
---

# Phase 17 Plan 02: Alert Triggers and User Notification Preference UI

**One-liner:** Low-stock weekly alerts via scheduler with per-item idempotency, and notification channel preference selector integrated into the Users tab edit form.

## What Was Built

### Task 1: Low-Stock Alert Check and Migration (commit: 6dfdc76)

- `0044_low_stock_alert_log.sql` — `UNIQUE (item_description, week_start)` for per-item weekly deduplication of low-stock alerts
- `scheduler.go` imports `inventory` package for `ClassifyStockLevel`
- `runLowStockCheck(ctx, pool)` added to scheduler package:
  1. Computes current Monday-based week_start in America/Chicago timezone
  2. Queries all stock items with quantities and thresholds (same join approach as GetSuggestions)
  3. Classifies each item via `inventory.ClassifyStockLevel` — collects items with level='low'
  4. For each low item: `INSERT INTO low_stock_alert_log ... ON CONFLICT DO NOTHING` — only newly-inserted items are batched for alert
  5. Builds single batch message: "Low Stock Alert: N item(s) below threshold: item1, item2, ..."
  6. Sends to admin contacts via `users.GetUsersForAlerts` + alertQueue.Enqueue
- `runSchedulerTick` updated to call `runLowStockCheck` between reminder check and reset check

### Task 2: Notification Preference in Edit Form (commit: bee5f8b)

**db.go:**
- Added `NotificationPref *string` to `UpdateUserInput` with validation (`zoho_cliq` or `email`)
- `UpdateUser` SET clause extended: when `NotificationPref` is non-nil, adds `notification_channel = $N`

**handler.go:**
- `UpdateUserHandler` PATCH body now accepts `notification_pref *string` field
- Passed through to `UpdateUserInput.NotificationPref` — validation error returns 400

**users.html:**
- "Alert Channel" `<select id="f-notif-pref">` added to editUser() form below email field
- Two options: "Zoho Cliq" (value=`zoho_cliq`) and "Email" (value=`email`)
- Pre-selects from `u.notification_channel` (defaults to `zoho_cliq` if empty)
- `saveUser()` reads `f-notif-pref` and includes `notification_pref` in PATCH body
- Service worker rebuilt (`sw.js`)

## Verification

- `go build ./cmd/server/` — PASS
- scheduler.go contains `runLowStockCheck` wired into `runSchedulerTick`
- `NotifyVendorComplete` (from Plan 01) calls `EnqueueAlert` with missing items
- users.html has "Alert Channel" select in edit form with Zoho Cliq and Email options
- `UserRow.NotificationChannel` is populated and returned in API responses (from Plan 01)
- `UpdateUser` accepts and validates `NotificationPref`

## Deviations from Plan

**1. [Context] Plan 02 partially pre-implemented by Plan 01**

Plan 01 (executed by a parallel agent) already implemented several items listed in Plan 02's task requirements:
- `runReminderCheck` (cutoff reminder) was in Plan 02 task 1 but done in Plan 01
- `runRepurchaseResetCheck` (badge reset) was in Plan 02 task 1 but done in Plan 01
- `NotifyVendorComplete` with real implementation (shopping completion alert) was in Plan 02 task 1 but done in Plan 01
- `RecordRepurchase` for repurchase_events was in Plan 02 but done in Plan 01
- `NotificationPref` on UserRow + `GetUsersForAlerts` were in Plan 02 task 2 but done in Plan 01

**Plan 02 scope reduced to what was genuinely missing:**
- `runLowStockCheck` (not in Plan 01)
- `low_stock_alert_log` migration (not in Plan 01)
- `users.html` notification preference selector (not in Plan 01)
- `UpdateUserInput.NotificationPref` in PATCH body (not in Plan 01 — Plan 01 used a separate endpoint)

**2. [Deviation] Merged main into worktree before execution**

The worktree branch was 59+ commits behind main. The Phase 17-01 work only existed in the local main branch (not yet pushed to origin). Used `git fetch /path/to/repo main && git merge FETCH_HEAD` to bring the worktree current before proceeding.

None of the deviations affect correctness, security, or completeness.

## Known Stubs

None — all features fully implemented.

## Self-Check: PASSED
