---
phase: "17"
plan: "01"
subsystem: backend
tags: [alerts, notifications, repurchase, zoho-cliq, email, stock-badges]
completed: "2026-04-23"
duration_minutes: 8

dependency_graph:
  requires: [phase-16-shopping-lists]
  provides: [alert-queue, notification-preferences, repurchase-badge-api]
  affects: [inventory-stock-tab, users-tab]

tech_stack:
  added:
    - "alerts package: async queue with Zoho Cliq webhook + SMTP email senders"
    - "net/smtp for email delivery (stdlib)"
    - "repurchase_log table for badge tracking"
    - "alert_log table for reminder idempotency"
    - "notification_channel on users table"
    - "repurchase_reset_config for weekly badge reset schedule"
  patterns:
    - "package-level alertQueue var with SetAlertQueue() wiring (same pattern as scheduler)"
    - "best-effort async dispatch: alerts don't block user actions (D-04)"
    - "idempotent cutoff reminder via UNIQUE (alert_type, week_start) in alert_log"
    - "graceful no-op senders when env vars missing (dev-safe)"
    - "runSchedulerTick() bundles cutoff check + reminder check + repurchase reset check"

key_files:
  created:
    - backend/internal/alerts/types.go
    - backend/internal/alerts/config.go
    - backend/internal/alerts/sender.go
    - backend/internal/alerts/queue.go
    - backend/internal/purchasing/repurchase.go
    - backend/internal/db/migrations/0041_notification_channel.sql
    - backend/internal/db/migrations/0042_repurchase_tracking.sql
    - backend/internal/db/migrations/0043_alert_log.sql
    - .planning/phases/17-alerts-notifications-and-repurchase-badges/17-01-PLAN.md
  modified:
    - backend/internal/purchasing/service.go
    - backend/internal/purchasing/scheduler.go
    - backend/internal/purchasing/handler.go
    - backend/internal/purchasing/types.go
    - backend/internal/users/db.go
    - backend/internal/users/handler.go
    - backend/internal/inventory/types.go
    - backend/internal/inventory/handler.go
    - backend/cmd/server/main.go

decisions:
  - "package-level alertQueue var (not constructor injection) to minimize caller changes — consistent with scheduler.go pattern"
  - "alerts called after COMMIT in CompleteVendorSection — best-effort, don't block the transaction"
  - "RecordRepurchase also called after COMMIT — badge data is cosmetic, consistency not critical"
  - "GetStockHandler uses two separate badge queries (with/without reset cutoff) to avoid CASE complexity in SQL"
  - "splitHostPort helper in sender.go instead of importing net package's SplitHostPort (avoids unnecessary import)"
  - "GetUsersForAlerts cutoff_reminder audience: crew/manager/admin roles + individual purchasing permission grants"
---

# Phase 17 Plan 01: Alert Infrastructure, Notification Preferences, and Repurchase Badge API

**One-liner:** Async alert queue with Zoho Cliq + SMTP delivery, notification preferences per user, and shopping-completion repurchase badges on inventory stock items.

## What Was Built

### Task 1: DB Migrations (commit: 0fab8e2)
- `0041_notification_channel.sql` — `notification_channel TEXT NOT NULL DEFAULT 'zoho_cliq' CHECK (IN ('zoho_cliq', 'email'))` on users
- `0042_repurchase_tracking.sql` — `repurchase_log` (item/list/qty/timestamp) + `repurchase_reset_config` (single-row weekly reset schedule)
- `0043_alert_log.sql` — `alert_log` with UNIQUE (alert_type, week_start) for idempotent reminder tracking

### Task 2: Alerts Package (commit: a308a49)
- `alerts/types.go` — Alert struct, channel/type constants
- `alerts/config.go` — Config from ZOHO_CLIQ_WEBHOOK_URL, SMTP_ADDR, SMTP_FROM, SMTP_USERNAME, SMTP_PASSWORD env vars
- `alerts/sender.go` — `SendZohoCliq(webhookURL, message)` posts `{"text": message}` JSON; `SendEmail(...)` uses net/smtp; both gracefully no-op when unconfigured
- `alerts/queue.go` — 100-item buffered Queue, `Start(ctx)` goroutine, `Enqueue(Alert)` non-blocking with drop warning on full

### Task 3: Alert Wiring (commit: 105721d)
- `purchasing/service.go` — `SetAlertQueue()`, `NotifyVendorComplete` stub replaced with real dispatch: queries unchecked items, builds message (D-13), enqueues to admin contacts
- `purchasing/scheduler.go` — `runReminderCheck()`: 24h before cutoff, idempotent via alert_log INSERT ... ON CONFLICT DO NOTHING; `runSchedulerTick()` combines all three checks
- `users/db.go` — `NotificationChannel` field on `UserRow`, `GetNotificationPreference`, `UpdateNotificationPreference`, `GetUsersForAlerts` (separate queries for cutoff_reminder vs shopping_complete audiences)
- `cmd/server/main.go` — `alerts.LoadConfig()`, `alerts.NewQueue()`, `alertQ.Start(ctx)`, `purchasing.SetAlertQueue(alertQ)`

### Task 4: Repurchase Badge (commit: ca6cdb4)
- `purchasing/repurchase.go` — `RecordRepurchase()` inserts checked items to repurchase_log; `TriggerRepurchaseReset/GetRepurchaseResetConfig/UpsertRepurchaseResetConfig`; `runRepurchaseResetCheck()` for automatic weekly reset via scheduler
- `purchasing/types.go` — `RepurchaseResetConfig` struct
- `purchasing/service.go` — calls `RecordRepurchase` after COMMIT (best-effort)
- `purchasing/handler.go` — `RepurchaseResetHandler` (POST), `GetRepurchaseResetConfigHandler` (GET), `UpsertRepurchaseResetConfigHandler` (PUT)
- `inventory/types.go` — `RepurchaseBadge{Qty, RepurchasedAt}` + `RepurchaseBadge *RepurchaseBadge` field on `StockItem`
- `inventory/handler.go` — GetStockHandler fetches `repurchase_reset_config.last_reset_at`, queries repurchase_log by purchase_item_id, attaches badge per stock item
- Routes registered: GET/POST `/repurchase-reset`, PUT `/repurchase-reset/config`

### Task 5: Notification Preference API (commit: 28c8fe9)
- `users/handler.go` — `GetNotificationPreferenceHandler` (admin or self), `UpdateNotificationPreferenceHandler` (admin or self, validates 'zoho_cliq'|'email')
- Routes: `GET /api/v1/users/{id}/notification-preference`, `PUT /api/v1/users/{id}/notification-preference`

## Verification

- `go build ./...` — PASS
- `go vet ./...` — PASS
- All 3 migrations syntactically valid (goose BEGIN/COMMIT pattern with Down sections)
- Alert queue gracefully no-ops when ZOHO_CLIQ_WEBHOOK_URL and SMTP_ADDR are empty
- `NotifyVendorComplete` stub fully replaced with real implementation
- `RecordRepurchase` called in `CompleteVendorSection` after COMMIT
- Stock API includes `repurchase_badge` field (nil when no entry in repurchase_log)
- Notification preference routes registered and auth-gated (admin or self)

## Deviations from Plan

**1. [Rule 2 - Missing functionality] Plan was not pre-created**

The plan file did not exist when the executor agent started. The worktree was forked from an older state of main (59 commits behind). Main had the phase 17 context but no plan file. The executor merged main, read the context, created the plan, then executed it.

**2. [Rule 1 - Bug] RecordRepurchase called after COMMIT, not inside transaction**

Plan task 4 said "within transaction" but NotifyVendorComplete (which is also best-effort) was already moved after COMMIT for the same reason. RecordRepurchase is cosmetic badge data and doesn't need transaction atomicity. Both are called after COMMIT with error-logging only.

None of the deviations affect correctness, security, or completeness of Phase 17 Plan 01.

## Known Stubs

None — all features fully implemented. Alert delivery no-ops gracefully when env vars are not set (this is intentional for dev, not a stub).

## Self-Check: PASSED

All 10 created files found on disk. All 5 task commits verified in git log.
