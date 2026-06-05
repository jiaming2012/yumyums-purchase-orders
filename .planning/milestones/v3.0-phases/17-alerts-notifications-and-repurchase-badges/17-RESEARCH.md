# Phase 17: Alerts, Notifications, and Repurchase Badges — Research

**Researched:** 2026-04-22
**Domain:** Go HTTP webhooks, async goroutine queues, SMTP email, Postgres schema additions, vanilla JS badge rendering
**Confidence:** HIGH (code patterns are established in-repo; Zoho Cliq webhook format verified from official docs)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Alert Delivery**
- D-01: Zoho Cliq incoming webhook is the primary delivery channel; email is the fallback alternative
- D-02: Zoho Cliq integration uses a single incoming webhook URL configured at the app/admin level (not per-user)
- D-03: Email delivery uses a simple SMTP or transactional email service (e.g., SendGrid, Mailgun)
- D-04: Alert queue is async — alerts are enqueued and delivered in the background, not blocking user actions

**Notification Preferences**
- D-05: Each user has a notification preference field: "zoho_cliq" (default) or "email"
- D-06: At least one channel is required — users cannot disable all notifications
- D-07: Preference is configured in the Users tab (admin edits for all users, user edits for self)

**Cutoff Reminders**
- D-08: Single reminder sent 24 hours before the configured cutoff time
- D-09: Reminder goes to all users with "order" permission (crew who can add to PO)
- D-10: Reminder message includes: week label, current item count, cutoff day/time

**Shopping Completion Alerts**
- D-11: When a shopping list is fully completed, an alert is sent listing any missing/unchecked items
- D-12: Alert goes to admin(s) and the shopper who completed the list
- D-13: Alert includes: vendor name, missing item names, quantities, who completed it

**Repurchase Badges**
- D-14: Inventory Stock tab items show a "Repurchased +[Qty]" badge after the shopping list containing that item is completed
- D-15: Badge quantity comes from the shopping list item's quantity (not PO quantity)
- D-16: Admin can configure a reset date/schedule (weekly, matching cutoff cycle)
- D-17: Badge resets for all items on the configured reset date

### Claude's Discretion
- Alert queue implementation (in-process goroutine channel vs external queue)
- Email template formatting
- Zoho Cliq message card formatting (plain text vs rich card)
- Database schema for alert history/log

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ALRT-01 | System sends reminder alerts before cutoff time | Scheduler extension pattern (runCutoffCheck) + async queue |
| ALRT-02 | System sends alerts when items are out of stock | GetStockHandler response — ClassifyStockLevel "low" items; scheduler tick |
| ALRT-03 | Alerts delivered via Zoho Cliq channel (default) or email | Webhook URL format verified; net/smtp pattern |
| ALRT-04 | Users configure communication preference in Users tab | New users column + Users tab UI addition |
| ALRT-05 | Zoho Cliq channel integration via incoming webhook | HTTP POST `{"text":"..."}` to channelsbyname endpoint |
| ALRT-06 | Missing items alert sent on shopping list completion via configured channels | NotifyVendorComplete stub already in CompleteVendorSection |
| REP-01 | Inventory items show "Repurchased +[Qty]" badge after purchase via shopping list completion | New repurchase_events table; GetStockHandler join |
| REP-02 | Badge resets on a configurable date (admin-settable reset schedule) | New badge_reset_config table; scheduler tick |
</phase_requirements>

---

## Summary

Phase 17 adds two independent systems: (1) a multi-channel alert delivery pipeline (Zoho Cliq webhook + email) triggered by scheduler events and shopping list completion, and (2) repurchase badge tracking on the inventory Stock tab.

The alert system extends two existing hook points: the `runCutoffCheck` function in `scheduler.go` (for 24-hour cutoff reminders) and the `NotifyVendorComplete` stub already planted in `CompleteVendorSection` in `service.go` (for shopping completion alerts). An in-process goroutine channel queue handles async delivery without blocking request handlers.

The repurchase badge system requires a new `repurchase_events` table (written at shopping list completion) and a `badge_reset_config` table (admin-configurable reset date). `GetStockHandler` joins against this table to include badge data in the stock response. The inventory.html Stock tab renders the badge inline with the existing `stock-badge` pattern.

**Primary recommendation:** Implement the alert notifier as a package-level `chan AlertJob` goroutine (matching the `receipt.StartWorker` pattern already in the codebase), extend scheduler.go for cutoff reminders, wire `NotifyVendorComplete` to enqueue completion alerts, and add two migrations for repurchase tracking.

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net/http` (stdlib) | Go stdlib | Zoho Cliq webhook POST | No external dependency; simple HTTP client pattern already used in photos/spaces.go |
| `net/smtp` (stdlib) | Go stdlib | Email fallback delivery | No dependency needed; SMTP is the universal fallback for simple transactional email |
| `encoding/json` (stdlib) | Go stdlib | Webhook payload marshaling | Already used throughout codebase |
| `time` (stdlib) | Go stdlib | 24h reminder window calculation | Already used in scheduler.go |
| pgx/v5 | v5.9.1 (in go.mod) | Repurchase table queries | Existing DB driver |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| SendGrid (optional) | — | Transactional email with delivery tracking | If plain SMTP is unreliable in production; requires `github.com/sendgrid/sendgrid-go` |
| Mailgun (optional) | — | Alternative transactional email | If SendGrid is not preferred |

**Recommendation:** Use `net/smtp` + app-password for the email fallback initially. The codebase has zero email infrastructure today (invites return a raw token in the API response, not emailed). Adding an SMTP sender is a single function. If reliability becomes an issue in production, swap to SendGrid — the interface stays the same.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| goroutine channel queue | Redis/external queue | Redis adds infra cost and complexity; goroutine channel is sufficient for 1-5 users with low message volume |
| net/smtp | SendGrid SDK | SendGrid adds a dependency; net/smtp works fine with Gmail App Password or Mailgun SMTP relay |

**Installation:** No new dependencies required for the primary implementation (stdlib only). If SendGrid is chosen: `go get github.com/sendgrid/sendgrid-go`

---

## Architecture Patterns

### Recommended Project Structure

New files:
```
backend/internal/purchasing/
├── alerts.go           # AlertJob type, channel, StartAlertWorker, Enqueue
├── notifier.go         # CliqNotifier, SMTPNotifier (implement Notifier interface)
backend/internal/db/migrations/
├── 0041_notification_pref.sql   # users.notification_pref column
├── 0042_repurchase_events.sql   # repurchase_events table
├── 0043_badge_reset_config.sql  # badge_reset_config single-row table
```

### Pattern 1: Async Alert Queue (goroutine channel)

**What:** A buffered `chan AlertJob` consumed by a dedicated goroutine. Callers enqueue and return immediately. Worker delivers to Cliq or email based on recipient preference.

**When to use:** All alert sends. Never block request handlers or the scheduler tick.

**Example:**
```go
// Source: pattern from backend/internal/receipt/worker.go (StartWorker)
// In alerts.go

type AlertJob struct {
    Recipients []AlertRecipient // {UserID, Email, Pref: "zoho_cliq"|"email"}
    Subject    string
    Body       string
}

var alertQueue = make(chan AlertJob, 100)

func StartAlertWorker(ctx context.Context, notifier Notifier) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case job := <-alertQueue:
                for _, r := range job.Recipients {
                    if err := notifier.Send(ctx, r, job.Subject, job.Body); err != nil {
                        log.Printf("alert send to %s (%s): %v", r.Email, r.Pref, err)
                    }
                }
            }
        }
    }()
}

func EnqueueAlert(job AlertJob) {
    select {
    case alertQueue <- job:
    default:
        log.Printf("alert queue full — dropping alert: %s", job.Subject)
    }
}
```

### Pattern 2: Notifier Interface

**What:** An interface with two implementations — `CliqNotifier` (HTTP POST to Zoho webhook URL) and `SMTPNotifier` (net/smtp). Selected per-recipient based on `notification_pref` field.

**Example:**
```go
// Source: analogous to photos.SpacesConfig pattern

type Notifier interface {
    Send(ctx context.Context, r AlertRecipient, subject, body string) error
}

// CliqNotifier posts to a single shared channel webhook URL.
type CliqNotifier struct {
    WebhookURL string // CLIQ_WEBHOOK_URL env var
    HTTPClient *http.Client
}

func (c *CliqNotifier) Send(ctx context.Context, r AlertRecipient, subject, body string) error {
    payload, _ := json.Marshal(map[string]string{"text": subject + "\n" + body})
    req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.WebhookURL, bytes.NewReader(payload))
    req.Header.Set("Content-Type", "application/json")
    resp, err := c.HTTPClient.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 {
        return fmt.Errorf("cliq webhook: status %d", resp.StatusCode)
    }
    return nil
}
```

### Pattern 3: Zoho Cliq Webhook URL and Payload

**What:** POST to `https://cliq.zoho.com/api/v2/channelsbyname/{channel}/message?zapikey={token}` with `{"text":"..."}`.

**Confirmed from:** Zoho Cliq community docs (help.zoho.com)

```
POST https://cliq.zoho.com/api/v2/channelsbyname/{channel-unique-name}/message?zapikey={token}
Content-Type: application/json

{"text": "Cutoff reminder: 3 items on this week's order. Cutoff is Tuesday 2:00 PM CT."}
```

The `zapikey` webhook token is generated in Zoho Cliq admin under "Webhook Tokens". The channel name is the unique identifier visible in the channel's API Endpoint URL in Cliq settings. The full URL including `zapikey` is stored as a single env var `CLIQ_WEBHOOK_URL`.

### Pattern 4: Scheduler Extension for 24h Reminder

**What:** Add a reminder check to `runCutoffCheck`. Calculate `cutoffTime - 24h`; if current time is within the last 15m window before that point, send the reminder.

**Existing hook:** `runCutoffCheck` in `scheduler.go` already has full cutoff config, timezone, and PO item count queries. The 15-minute tick ensures the window is always checked.

**Logic:**
```go
// In runCutoffCheck, after computing cutoffCandidate:
reminderTime := cutoffCandidate.Add(-24 * time.Hour)
reminderWindowStart := reminderTime.Add(-15 * time.Minute)
if now.After(reminderWindowStart) && now.Before(reminderTime.Add(15*time.Minute)) {
    // Send reminder — check if already sent this cycle to avoid duplicates
    sendCutoffReminder(ctx, pool, cutoffCandidate, itemCount)
}
```

**Deduplication:** Track `last_reminder_sent_at` in `cutoff_config` table (add column), or use a separate `alert_log` table. The simplest approach: add `reminder_sent_week` TEXT column to `cutoff_config`; set it to `week_start` when the reminder is sent; skip if already set for this week.

### Pattern 5: Repurchase Events Table

**What:** Written when a shopping list is completed (inside `CompleteVendorSection` after list-complete detection). Records which `purchase_item_id` was bought and the quantity.

**Migration 0042:**
```sql
CREATE TABLE repurchase_events (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  shopping_list_id  UUID NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
  purchase_item_id  UUID NOT NULL REFERENCES purchase_items(id) ON DELETE CASCADE,
  quantity          INTEGER NOT NULL,
  repurchased_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_repurchase_events_item ON repurchase_events(purchase_item_id);
CREATE INDEX idx_repurchase_events_list ON repurchase_events(shopping_list_id);
```

**Reset config (migration 0043):**
```sql
CREATE TABLE badge_reset_config (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  -- day_of_week matches cutoff_config.day_of_week semantics (0=Sunday)
  day_of_week INTEGER NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
  reset_time  TIME NOT NULL DEFAULT '06:00:00',
  timezone    TEXT NOT NULL DEFAULT 'America/Chicago',
  last_reset_at TIMESTAMPTZ,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**Data written at completion:**
```go
// In service.go, when listCompleted == true, inside the same transaction:
for _, section := range loadedSections {
    for _, item := range section.Items {
        tx.Exec(ctx, `
            INSERT INTO repurchase_events (shopping_list_id, purchase_item_id, quantity)
            VALUES ($1, $2, $3)
        `, listID, item.PurchaseItemID, item.Quantity)
    }
}
```

### Pattern 6: GetStockHandler — Repurchase Badge Join

**What:** Extend the existing stock query to LEFT JOIN repurchase_events, summing quantity since last reset.

```sql
-- Add to the outer SELECT in GetStockHandler:
COALESCE(re_agg.repurchased_qty, 0) AS repurchased_qty

-- Add LEFT JOIN:
LEFT JOIN (
    SELECT purchase_item_id, SUM(quantity) AS repurchased_qty
    FROM repurchase_events
    WHERE repurchased_at > COALESCE(
        (SELECT last_reset_at FROM badge_reset_config LIMIT 1),
        '1970-01-01'::timestamptz
    )
    GROUP BY purchase_item_id
) re_agg ON re_agg.purchase_item_id = pi.id
```

The existing `StockItem` struct in inventory package gets a new field `RepurchasedQty int`.

### Pattern 7: Frontend Badge Rendering

**What:** The Stock tab already has `.stock-badge` CSS class with the same pattern this badge should follow. Add a second badge inline.

**Existing rendering location (inventory.html line 559):**
```javascript
var badgeAndIndicator = badgeHtml(s.level) + ' ';
// ADD:
if (s.repurchased_qty > 0) {
    badgeAndIndicator += '<span class="stock-badge" style="background:var(--info-bg,#e3f2fd);color:var(--info-tx,#1565c0)">Repurchased +' + s.repurchased_qty + '</span> ';
}
html += '<div class="stock-item" ...>' + ... + badgeAndIndicator + '</div>';
```

No new CSS class needed — reuse `.stock-badge` with an inline color override, or add a `.stock-repurchased` modifier class.

### Pattern 8: Notification Preference in Users Tab

**What:** Add `notification_pref TEXT DEFAULT 'zoho_cliq'` column to `users` table via migration 0041. Expose it in `UserRow`, `ListUsers`, `GetUser`, and `UpdateUserInput`. Add a preference selector in users.html.

**Migration 0041:**
```sql
ALTER TABLE users ADD COLUMN notification_pref TEXT NOT NULL DEFAULT 'zoho_cliq'
  CHECK (notification_pref IN ('zoho_cliq', 'email'));
```

**users.html UI:** Add a single `<select>` in the edit user form (same area as role selector). Admin and self can edit. Simple two-option select: "Zoho Cliq (default)" / "Email".

### Anti-Patterns to Avoid
- **Sending alerts synchronously inside request handlers:** `CompleteVendorSection` is called inside a transaction; blocking on HTTP delivery risks timeout and dirty state. Always enqueue.
- **Sending duplicate reminders:** The 15-min scheduler tick fires every 15 minutes. Without deduplication, the 24h reminder fires multiple times. Track sent state in DB.
- **Blocking `runCutoffCheck` on alert delivery:** `runCutoffCheck` is called from scheduler goroutine; `EnqueueAlert` must be non-blocking (use select/default).
- **Storing the full webhook URL with secret token in code:** Store as `CLIQ_WEBHOOK_URL` env var only; never commit.
- **Joining repurchase_events without a reset date guard:** Without filtering by `last_reset_at`, badges accumulate forever across weeks.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP client for webhook | Custom retry/backoff loop | stdlib `net/http` with simple error log | 1-5 users; single fire-and-forget is sufficient. No SLA requiring retries. |
| Email delivery | Raw SMTP implementation | `net/smtp` + `smtp.SendMail()` | stdlib one-liner handles STARTTLS; no need to parse MIME manually |
| Queue persistence | Redis/persistent queue | Buffered goroutine channel | Memory loss on crash is acceptable — missed alert is not a critical failure for this scale |

**Key insight:** This is a 1-5 user food truck app. Simple, synchronous-looking async (goroutine channel) is the right fit. Do not over-engineer with external message brokers.

---

## Runtime State Inventory

> Not a rename/refactor/migration phase — this section is omitted.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Zoho Cliq webhook URL | ALRT-03, ALRT-05 | Unknown — env var `CLIQ_WEBHOOK_URL` | — | Skip Cliq delivery; log warning |
| SMTP relay credentials | ALRT-03 (email fallback) | Unknown — env vars `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS` | — | Skip email delivery; log warning |
| PostgreSQL | REP-01, REP-02 | Yes (existing) | pgx/v5 in go.mod | — |

**Missing dependencies with no fallback:**
- None that block feature delivery. Both notification channels are optional at the transport level — if env vars are absent, the alert worker logs and skips. The badge system (REP-01, REP-02) has no external deps.

**Missing dependencies with fallback:**
- `CLIQ_WEBHOOK_URL` absent → skip Cliq sends, log. (Graceful degradation matching `DO_SPACES_KEY` pattern in main.go)
- `SMTP_HOST` absent → skip email sends, log.

**New env vars required (add to server startup wiring in main.go):**
```
CLIQ_WEBHOOK_URL      # Full Zoho Cliq channelsbyname webhook URL including zapikey token
SMTP_HOST             # e.g. smtp.gmail.com
SMTP_PORT             # e.g. 587
SMTP_USER             # e.g. alerts@yumyums.kitchen
SMTP_PASS             # App password (not account password)
SMTP_FROM             # From address, e.g. "Yumyums HQ <alerts@yumyums.kitchen>"
```

---

## Common Pitfalls

### Pitfall 1: Duplicate 24h Reminders on Every Scheduler Tick
**What goes wrong:** The 15-minute scheduler tick fires every tick within the 24h reminder window. Without deduplication, the crew receives multiple identical reminders.
**Why it happens:** `runCutoffCheck` has no memory of previous runs.
**How to avoid:** Add `reminder_sent_week TEXT` column to `cutoff_config`. When sending the reminder, set it to the current `week_start`. Check before sending: `if cfg.ReminderSentWeek == currentWeek { return }`.
**Warning signs:** Multiple Cliq messages within minutes of each other.

### Pitfall 2: Alert Sent Inside Transaction Before Commit
**What goes wrong:** `CompleteVendorSection` calls `NotifyVendorComplete` before `tx.Commit()`. If HTTP delivery blocks and then the commit fails, the alert was sent for a state that was rolled back.
**Why it happens:** The current stub is placed before `tx.Commit()` — the right position for the hook but wrong for synchronous I/O.
**How to avoid:** `NotifyVendorComplete` must only **enqueue** (non-blocking channel send). The actual HTTP request happens in the worker goroutine after the caller's transaction has committed.
**Warning signs:** Alerts for shopping lists that don't appear as completed.

### Pitfall 3: Badge Accumulation Without Reset Date
**What goes wrong:** `repurchase_events` accumulates all time without a reset; badges show lifetime repurchase counts instead of the current week's.
**Why it happens:** If `badge_reset_config` has no row (unconfigured), the JOIN falls back to `'1970-01-01'` and sums everything.
**How to avoid:** Scheduler checks `badge_reset_config` on each tick and fires reset by updating `last_reset_at = now()` when the configured day/time passes. Frontend shows badge only if `repurchased_qty > 0` AND `repurchased_at > last_reset_at`.
**Warning signs:** Badge counts growing week over week with no reset.

### Pitfall 4: Cliq Channel Name vs. Channel Unique Name
**What goes wrong:** The Zoho Cliq webhook URL requires the channel's *unique name* (used in API endpoints), not the display name.
**Why it happens:** Cliq channels have a display name ("Kitchen Crew") and a separate unique name ("kitchen-crew") used in API URLs.
**How to avoid:** Store the full webhook URL (including `zapikey`) as the single env var. The admin generates it in Zoho Cliq under Webhook Tokens and pastes the full URL — no name parsing needed.
**Warning signs:** 404 responses from Cliq webhook endpoint.

### Pitfall 5: ALRT-02 Out-of-Stock Alert Scope
**What goes wrong:** ALRT-02 says "alerts when items are out of stock" but GetStockHandler returns stock level as a classification, not a real-time event trigger.
**Why it happens:** Stock levels are computed from purchase history + overrides — there is no INSERT event that fires when an item goes low.
**How to avoid:** Implement ALRT-02 as a periodic scheduler check (same ticker): query stock items with `level = 'low'`, compare against a `last_low_stock_alert_at` table to deduplicate, and enqueue alerts for items newly below threshold.
**Warning signs:** Continuous daily alerts for items that have been low for weeks.

---

## Code Examples

### Verified: Zoho Cliq POST Format
```go
// Source: help.zoho.com community docs (verified 2026-04-22)
// POST https://cliq.zoho.com/api/v2/channelsbyname/{channel}/message?zapikey={token}
// Content-Type: application/json
payload := map[string]string{"text": "Your message here"}
body, _ := json.Marshal(payload)
req, _ := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
resp, err := http.DefaultClient.Do(req)
```

### Verified: net/smtp SendMail (stdlib)
```go
// Source: Go stdlib net/smtp documentation
import "net/smtp"

auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
to := []string{recipientEmail}
msg := []byte("To: " + recipientEmail + "\r\n" +
    "From: " + fromAddr + "\r\n" +
    "Subject: " + subject + "\r\n" +
    "\r\n" + body + "\r\n")
err := smtp.SendMail(smtpHost+":"+smtpPort, auth, fromAddr, to, msg)
```

### Verified: Non-blocking enqueue pattern
```go
// Source: existing pattern in this codebase (service.go fire-and-forget goroutine)
func EnqueueAlert(job AlertJob) {
    select {
    case alertQueue <- job:
        // queued
    default:
        log.Printf("alert queue full — dropping: %s", job.Subject)
    }
}
```

### Verified: Goose migration pattern (existing convention)
```sql
-- Source: all migrations in backend/internal/db/migrations/
-- +goose Up
BEGIN;
ALTER TABLE users ADD COLUMN notification_pref TEXT NOT NULL DEFAULT 'zoho_cliq'
  CHECK (notification_pref IN ('zoho_cliq', 'email'));
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE users DROP COLUMN notification_pref;
COMMIT;
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Alerts not implemented | NotifyVendorComplete stub in service.go | Phase 16 (current) | Clean hook point already exists — no refactor needed |
| SMTP for all emails | Transactional services (SendGrid, Postmark) | ~2018-2020 industry | net/smtp still valid for small scale; upgrade path is available if needed |

**Deprecated/outdated:**
- Zoho Cliq channel "outgoing webhooks" are deprecated — use bot incoming webhook or the `channelsbyname/message` channel API with `zapikey` instead.

---

## Open Questions

1. **ALRT-02 scope: what counts as "out of stock"?**
   - What we know: `ClassifyStockLevel` returns "low", "medium", "high", "unknown". ALRT-02 says "out of stock" but no item reaches true 0 in the current data model unless stock override is explicitly set to 0.
   - What's unclear: Does ALRT-02 mean `level = 'low'` or `total_quantity = 0`?
   - Recommendation: Treat `level = 'low'` as the trigger condition. Document this in the plan task.

2. **ALRT-02 deduplication period**
   - What we know: Stock levels don't change in real-time; a scheduler check every 15 minutes would find the same low items repeatedly.
   - What's unclear: Should low-stock alerts repeat weekly? Daily? Only once per item until restocked?
   - Recommendation: Alert once per item per week (keyed by item+week_start). Add a `low_stock_alerts` table or use `alert_log` to track sent alerts.

3. **Email not previously implemented**
   - What we know: Invite flow returns raw token in API response; no SMTP code exists in the codebase today.
   - What's unclear: Which SMTP provider will be configured in production?
   - Recommendation: Implement with `net/smtp` using env var configuration. Provider-agnostic. Document in task that SMTP env vars must be configured in DO App Platform.

4. **Repurchase badge and purchase_items linked by UUID**
   - What we know: `GetStockHandler` currently joins via `COALESCE(pi.description, pli.description)` — items without a `purchase_items` row are matched by description string only.
   - What's unclear: Repurchase events require `purchase_item_id` UUID. Items without a linked `purchase_items` record won't get a badge.
   - Recommendation: Only insert `repurchase_events` for `shopping_list_items` with a non-null `purchase_item_id`. Document this limitation in the task.

---

## Sources

### Primary (HIGH confidence)
- `backend/internal/purchasing/scheduler.go` — existing scheduler pattern, DST-safe timezone logic
- `backend/internal/purchasing/service.go` — `NotifyVendorComplete` stub, `CompleteVendorSection` hook point, `ShoppingListItem` struct with `Quantity` and `PurchaseItemID`
- `backend/internal/purchasing/types.go` — all type definitions confirmed
- `backend/internal/users/db.go` — `UserRow`, `UpdateUserInput`, migration pattern
- `backend/internal/db/migrations/0040_shopping_lists.sql` — confirmed schema for shopping_list_items
- `backend/internal/db/migrations/0037_cutoff_config.sql` — confirmed schema for cutoff_config
- `backend/internal/inventory/handler.go` — `GetStockHandler` SQL query confirmed
- `backend/cmd/server/main.go` — env var pattern, graceful degradation pattern, StartWorker wiring

### Secondary (MEDIUM confidence)
- Zoho Cliq community docs (help.zoho.com/portal/en/community/topic/post-message-to-a-channel-using-a-simple-one-line-command) — confirmed POST endpoint format `channelsbyname/{name}/message?zapikey={token}` and `{"text":"..."}` body
- Zoho Cliq Webhook Tokens docs (zoho.com/cliq/help/platform/webhook-tokens.html) — confirmed `zapikey` parameter usage

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all Go stdlib; Zoho webhook POST format confirmed from official community docs
- Architecture: HIGH — all patterns replicate existing in-repo conventions (StartWorker, scheduler, migrations)
- Pitfalls: HIGH — pitfalls derived from direct code inspection of hook points and existing scheduler

**Research date:** 2026-04-22
**Valid until:** 2026-08-22 (stable — Zoho Cliq webhook format unlikely to change; Go stdlib stable)
