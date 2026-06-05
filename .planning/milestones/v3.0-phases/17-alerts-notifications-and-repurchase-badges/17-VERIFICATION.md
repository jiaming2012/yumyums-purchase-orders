---
phase: 17-alerts-notifications-and-repurchase-badges
verified: 2026-04-23T12:45:00Z
status: passed
score: 8/8 must-haves verified
re_verification: false
gaps: []
human_verification:
  - test: "Send a real cutoff reminder via Zoho Cliq"
    expected: "Message appears in configured Zoho Cliq channel 24h before cutoff"
    why_human: "Requires live Zoho OAuth credentials (ZOHO_CLIQ_CLIENT_ID, ZOHO_CLIQ_REFRESH_TOKEN) and a configured channel; cannot test without the live service"
  - test: "Send a real alert via email SMTP"
    expected: "Email delivered to recipient address with correct subject and body"
    why_human: "Requires live SMTP credentials (SMTP_ADDR, SMTP_USER, SMTP_PASS, SMTP_FROM)"
  - test: "Complete a vendor section and observe Repurchased badge on Stock tab"
    expected: "Stock tab shows blue 'Repurchased +N' badge on the item that was purchased"
    why_human: "Requires live DB with a seeded shopping list, a purchasable item in repurchase_log, and a browser to observe the rendered badge"
  - test: "Configure badge reset schedule and confirm badge disappears after reset time"
    expected: "After admin sets a past reset time and page reloads, Repurchased badges are gone"
    why_human: "Requires live DB state (repurchase_log and repurchase_reset_config rows) and browser observation"
---

# Phase 17: Alerts, Notifications, and Repurchase Badges — Verification Report

**Phase Goal:** The system delivers cutoff reminders and shopping completion alerts via Zoho Cliq or email based on user preference, and inventory Stock tab shows a "Repurchased" badge after shopping is confirmed.
**Verified:** 2026-04-23T12:45:00Z
**Status:** PASSED
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Crew receives a cutoff reminder 24h before deadline, sent once per week (no duplicates) | VERIFIED | `runReminderCheck` in `scheduler.go:52`; inserts `alert_log ON CONFLICT DO NOTHING` at line 155 for UNIQUE(alert_type, week_start) deduplication |
| 2 | Shopping list completion triggers an alert listing missing items | VERIFIED | `NotifyVendorComplete` in `service.go:591` queries unchecked items and calls `alertQueue.Enqueue` per channel per contact (line 648) |
| 3 | Low-stock items trigger an alert once per item per week to admins | VERIFIED | `runLowStockCheck` in `scheduler.go:247`; inserts `low_stock_alert_log ON CONFLICT DO NOTHING` at line 318; fires to admin contacts via double-loop Enqueue |
| 4 | Users can select multiple alert channels (Zoho Cliq and/or Email) in Users tab | VERIFIED | Checkboxes `f-notif-zoho` and `f-notif-email` in `users.html:342-343`; at least one required enforced client-side at line 377 |
| 5 | Alert queue dispatches to ALL selected channels per user | VERIFIED | Double-loop `for _, ch := range c.NotificationChannels` in `scheduler.go:143,349` and `service.go:648`; enqueues one `Alert` per channel per contact |
| 6 | Inventory Stock tab shows "Repurchased +N" badge when item was repurchased via shopping list | VERIFIED | `inventory.html:576` renders `<span class="stock-badge stock-repurchased">Repurchased +N</span>` when `s.repurchase_badge && s.repurchase_badge.qty > 0` |
| 7 | Badge disappears after admin-configured reset date passes | VERIFIED | `GetStockHandler` in `inventory/handler.go:363` reads `last_reset_at` from `repurchase_reset_config`; filters `repurchase_log` by that cutoff; `runRepurchaseResetCheck` updates `last_reset_at` when reset time passes |
| 8 | Scheduler uses cutoff_config timezone (not hardcoded) for low-stock week boundary | VERIFIED | `runLowStockCheck` at `scheduler.go:253-264` loads `GetCutoffConfig` timezone with `users.DefaultTimezone` fallback; zero occurrences of `America/Chicago` in scheduler.go |

**Score:** 8/8 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `backend/internal/alerts/types.go` | Alert struct, channel constants | VERIFIED | `Alert` struct, `ChannelZohoCliq`, `ChannelEmail`, `TypeCutoffReminder`, `TypeShoppingComplete` |
| `backend/internal/alerts/sender.go` | Zoho Cliq OAuth sender + SMTP email sender | VERIFIED | `SendZohoCliq` uses OAuth2 token refresh; `SendEmail` uses `net/smtp`; both gracefully no-op when unconfigured |
| `backend/internal/alerts/queue.go` | 100-item buffered async queue | VERIFIED | `NewQueue` creates 100-item buffer; `Start` goroutine; `Enqueue` non-blocking with drop warning |
| `backend/internal/alerts/config.go` | Config from env vars | VERIFIED | `LoadConfig()` reads `ZOHO_CLIQ_CLIENT_ID`, `ZOHO_CLIQ_REFRESH_TOKEN`, `ZOHO_CLIQ_PURCHASE_AND_INVENTORY_CHANNEL`, `SMTP_ADDR`, `SMTP_FROM`, etc. |
| `backend/internal/purchasing/scheduler.go` | 24h reminder, low-stock check, badge reset check | VERIFIED | `runReminderCheck`, `runLowStockCheck`, `runRepurchaseResetCheck` all called from `runSchedulerTick`; multi-channel dispatch loops verified |
| `backend/internal/purchasing/service.go` | NotifyVendorComplete wired to real alert dispatch | VERIFIED | Queries unchecked items, gets admin contacts, enqueues per-channel; `RecordRepurchase` called after commit |
| `backend/internal/purchasing/repurchase.go` | RecordRepurchase, reset config functions | VERIFIED | `RecordRepurchase` inserts checked items into `repurchase_log`; `runRepurchaseResetCheck` updates `last_reset_at`; `TriggerRepurchaseReset`, `GetRepurchaseResetConfig`, `UpsertRepurchaseResetConfig` |
| `backend/internal/users/db.go` | NotificationChannels []string, Timezone, GetUsersForAlerts | VERIFIED | `NotificationChannels []string` on `UserRow` and `NotificationPreferenceContact`; `Timezone string` on both; `GetUsersForAlerts` returns multi-channel contacts; `DefaultTimezone = "America/New_York"` |
| `backend/internal/inventory/types.go` | RepurchaseBadge struct on StockItem | VERIFIED | `RepurchaseBadge{Qty, RepurchasedAt}` at line 95; `RepurchaseBadge *RepurchaseBadge` on `StockItem` line 112 |
| `backend/internal/inventory/handler.go` | GetStockHandler with repurchase badge query | VERIFIED | Reads `last_reset_at`, runs `repurchase_log` query, attaches `RepurchaseBadge` per item |
| `inventory.html` | "Repurchased +N" badge rendering; badge reset config UI for admins | VERIFIED | `.stock-repurchased` CSS class; badge rendered at line 576; `badge-reset-section` shown only to `isAdmin()`; `loadBadgeResetConfig` / `renderBadgeResetConfig` at lines 1137-1193 |
| `users.html` | Multi-channel checkboxes + timezone dropdown in edit form | VERIFIED | `f-notif-zoho` and `f-notif-email` checkboxes at lines 342-343; `f-timezone` select at line 348 with 7 US+UTC options; `saveUser` sends array and timezone in PATCH body |
| `backend/internal/db/migrations/0041_notification_channel.sql` | notification_channel TEXT column on users | VERIFIED | File exists in migrations directory |
| `backend/internal/db/migrations/0042_repurchase_tracking.sql` | repurchase_log + repurchase_reset_config tables | VERIFIED | File exists |
| `backend/internal/db/migrations/0043_alert_log.sql` | alert_log with UNIQUE(alert_type, week_start) | VERIFIED | UNIQUE constraint confirmed; goose Up/Down format verified |
| `backend/internal/db/migrations/0044_low_stock_alert_log.sql` | low_stock_alert_log with UNIQUE(item_description, week_start) | VERIFIED | File exists |
| `backend/internal/db/migrations/0045_notification_channel_array.sql` | Converts notification_channel to TEXT[] | VERIFIED | File exists |
| `backend/internal/db/migrations/0046_user_timezone.sql` | timezone TEXT NOT NULL DEFAULT 'America/New_York' on users | VERIFIED | `ALTER TABLE users ADD COLUMN timezone TEXT NOT NULL DEFAULT 'America/New_York'` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `scheduler.go` | `alerts.Queue.Enqueue` | `for _, ch := range c.NotificationChannels` loop in `runReminderCheck` | WIRED | Line 143-148; one `Enqueue` per channel per contact |
| `scheduler.go` | `alerts.Queue.Enqueue` | Double-loop in `runLowStockCheck` | WIRED | Line 349-355 |
| `service.go` | `alerts.Queue.Enqueue` | Double-loop in `NotifyVendorComplete` | WIRED | Line 648-654 |
| `users.html` | `PATCH /api/v1/users/{id}` | `notification_pref: notifChannels` (array) in PATCH body | WIRED | `saveUser()` at line 384 sends array to API |
| `inventory.html` | `/api/v1/inventory/stock` | `s.repurchase_badge.qty` read in stock rendering | WIRED | Line 576 reads badge from stock API response |
| `inventory.html` | `/api/v1/purchasing/repurchase-reset` | `loadBadgeResetConfig` GETs config; `save-badge-reset` PUTs to `/config` | WIRED | Lines 1139, 1190 |
| `main.go` | `alerts.Queue` | `alertQ.Start(ctx)` + `purchasing.SetAlertQueue(alertQ)` | WIRED | Lines 497-499 |
| `main.go` | `/api/v1/users/{id}/notification-preference` routes | `GetNotificationPreferenceHandler`, `UpdateNotificationPreferenceHandler` registered | WIRED | Lines 340-341 |
| `main.go` | `/api/v1/purchasing/repurchase-reset` routes | `GetRepurchaseResetConfigHandler`, `RepurchaseResetHandler`, `UpsertRepurchaseResetConfigHandler` registered | WIRED | Lines 436-438 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|--------------------|--------|
| `inventory.html` Stock tab | `s.repurchase_badge` | `GET /api/v1/inventory/stock` — `GetStockHandler` queries `repurchase_log` with `last_reset_at` cutoff | DB query against `repurchase_log` + `repurchase_reset_config` | FLOWING |
| `users.html` edit form | `u.notification_channels` | `GET /api/v1/users` — `ListUsers` scans `u.notification_channel` TEXT[] column | Real DB scan with pgx `[]string` array scanning | FLOWING |
| `users.html` edit form | `u.timezone` | Same `ListUsers` query — scans `u.timezone` column added in migration 0046 | Real DB scan | FLOWING |
| `scheduler.go` reminder | contacts list | `GetUsersForAlerts("cutoff_reminder")` — queries users with purchasing-related roles and individual grants | Real DB query returning email + channels + timezone | FLOWING |
| `scheduler.go` low-stock | low-stock items | Queries stock with thresholds, classifies via `inventory.ClassifyStockLevel` | Real DB query joining purchase data | FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Backend compiles cleanly | `cd backend && go build ./cmd/server/` | BUILD OK | PASS |
| Go vet passes | `cd backend && go vet ./...` | VET OK | PASS |
| Alert queue start wired in main | `grep alertQ.Start backend/cmd/server/main.go` | Found at line 498 | PASS |
| Double-loop dispatch in scheduler | `grep "for.*ch.*range.*NotificationChannels" scheduler.go` | Lines 143, 349 | PASS |
| No hardcoded America/Chicago in scheduler | `grep "America/Chicago" scheduler.go` | 0 matches | PASS |
| Repurchase badge renders in inventory.html | `grep "stock-repurchased" inventory.html` | Found at lines 95, 576 | PASS |
| Checkboxes replace dropdown in users.html | `grep "f-notif-zoho" users.html` | Found at lines 342, 363 | PASS |
| Badge spacing CSS present | `grep "stock-badge+.stock-badge" inventory.html` | 1 match (line 94) | PASS |
| Timezone dropdown in users.html | `grep "f-timezone" users.html` | Found at lines 348, 383 | PASS |
| Handler defaults to America/New_York | `grep "America/New_York" backend/internal/purchasing/handler.go` | Line 645 | PASS |

### Requirements Coverage

| Requirement | Source Plans | Description | Status | Evidence |
|-------------|-------------|-------------|--------|---------|
| ALRT-01 | 17-01, 17-02 | System sends reminder alerts before cutoff time | SATISFIED | `runReminderCheck` fires 24h before cutoff; idempotent via `alert_log` UNIQUE constraint |
| ALRT-02 | 17-01, 17-02 | System sends alerts when items are out of stock | SATISFIED | `runLowStockCheck` queries below-threshold items; deduplicates via `low_stock_alert_log ON CONFLICT DO NOTHING` |
| ALRT-03 | 17-04 | Alerts delivered via Zoho Cliq channel (default) or email — multi-select | SATISFIED | `notification_channel` is `TEXT[]`; migration 0045 converts; dispatch loops over all selected channels |
| ALRT-04 | 17-01, 17-02 | Users configure communication preference in Users tab (at least one required) | SATISFIED | Checkboxes `f-notif-zoho`/`f-notif-email` in `users.html`; client-side `channels.length===0` guard; `UpdateUser` validates at DB layer |
| ALRT-05 | 17-05 | Zoho Cliq channel integration via OAuth (not webhook) | SATISFIED | `SendZohoCliq` uses OAuth2 token refresh (env: `ZOHO_CLIQ_CLIENT_ID`, `ZOHO_CLIQ_REFRESH_TOKEN`); no deprecated webhook approach |
| ALRT-06 | 17-01, 17-05 | Missing items alert sent on shopping list completion via configured channels | SATISFIED | `NotifyVendorComplete` queries unchecked items, builds per-D-13 message, enqueues per channel per contact; per-user timezone stored (migration 0046) |
| REP-01 | 17-01, 17-03 | Inventory items show "Repurchased +[Qty]" badge after shopping list completion | SATISFIED | `RecordRepurchase` inserts to `repurchase_log` after `CompleteVendorSection`; `GetStockHandler` attaches `RepurchaseBadge`; `inventory.html:576` renders blue badge |
| REP-02 | 17-01, 17-03 | Badge resets on a configurable date (admin-settable) | SATISFIED | `runRepurchaseResetCheck` updates `last_reset_at`; `UpsertRepurchaseResetConfigHandler` exposed at `PUT /api/v1/purchasing/repurchase-reset/config`; badge reset UI in Setup tab (admin-gated) |

All 8 requirement IDs satisfied. No orphaned requirements.

### Anti-Patterns Found

| File | Pattern | Severity | Assessment |
|------|---------|----------|------------|
| `inventory.html:1190` | Badge reset config hardcodes `timezone:'America/New_York'` in PUT body rather than reading from a dropdown | Info | Cosmetic limitation — the reset config timezone is a single admin setting. The inventory.html badge reset form does not expose a timezone selector; it defaults to `America/New_York`. Admin-facing limitation but does not break the feature since the backend correctly uses the stored value. Not a stub — the PUT is real and persists the value. |

No blocker anti-patterns. No placeholder stubs. No empty return values on user-visible paths.

### Human Verification Required

#### 1. Live Zoho Cliq alert delivery

**Test:** With `ZOHO_CLIQ_CLIENT_ID`, `ZOHO_CLIQ_CLIENT_SECRET`, `ZOHO_CLIQ_REFRESH_TOKEN`, and `ZOHO_CLIQ_PURCHASE_AND_INVENTORY_CHANNEL` set, trigger a cutoff reminder (advance system clock or call `runReminderCheck` directly) and observe a message in the configured Zoho Cliq channel.
**Expected:** "Cutoff Reminder: [week label] order has [N] items. Cutoff is [Day] at [HH:MM] [TZ]." appears in channel.
**Why human:** Cannot test without live OAuth credentials and Zoho service access.

#### 2. Live SMTP email delivery

**Test:** With `SMTP_ADDR`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM` set, set a user's notification channel to "email", trigger a shopping completion alert, and check the inbox.
**Expected:** Email arrives with correct subject and body listing missing items.
**Why human:** Requires live SMTP credentials and email inbox access.

#### 3. Repurchase badge visible on Stock tab after shopping completion

**Test:** Open a shopping list, mark items as purchased, confirm vendor section complete. Navigate to Inventory > Stock tab.
**Expected:** Items that were in the shopping list show a blue "Repurchased +N" badge next to the stock level badge.
**Why human:** Requires seeded DB with a shopping list linked to catalog items, and a browser to observe rendered output.

#### 4. Badge disappears after reset date passes

**Test:** In Setup tab (as admin), configure a badge reset time in the past. Reload Inventory > Stock tab.
**Expected:** "Repurchased +N" badges are no longer visible (last_reset_at cutoff filters out old repurchase_log entries).
**Why human:** Requires live DB state manipulation and browser observation.

### Gaps Summary

No gaps found. All 8 requirements are satisfied, all artifacts are substantive and wired, all data flows are traced to real DB queries, and the backend compiles and passes vet cleanly.

The one cosmetic limitation (badge reset timezone hardcoded to America/New_York in inventory.html PUT body) does not block any requirement — REP-02 only requires an admin-settable reset schedule, not per-user timezone on the badge reset config.

---

_Verified: 2026-04-23T12:45:00Z_
_Verifier: Claude (gsd-verifier)_
