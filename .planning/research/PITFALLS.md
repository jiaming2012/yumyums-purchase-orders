# Pitfalls Research

**Domain:** Purchase orders, shopping lists, external alert integrations (Zoho Cliq / email), scheduled cutoffs, and Notion data import — added to an existing Go+Postgres food service PWA (v3.0 milestone)
**Researched:** 2026-04-22
**Confidence:** HIGH (PO state machine, timezone, Notion image URLs — multiple verified sources); MEDIUM (Zoho Cliq failure modes — official docs sparse on operational failure details, community evidence supports conclusions)

---

## Critical Pitfalls

### Pitfall 1: Zoho Cliq Webhook Token Silently Revoked — Alerts Stop Silently

**What goes wrong:**
Zoho Cliq channel webhooks authenticate via a `zapikey` token appended to the webhook URL as a query parameter. Each user account holds a maximum of 5 webhook tokens. If the token owner's password changes, the account is deactivated (common with crew turnover), or Zoho revokes the token after a security event, every subsequent alert POST returns 401. If the Go handler doesn't log the response code, alerts vanish without any indication on the app side. The crew never gets notified. The owner doesn't find out until a shift is over.

**Why it happens:**
The webhook URL is generated once during setup, stored in the DB, and assumed to be permanent. Community reports confirm 401s appear after account-level changes (the Zoho help forum has a dedicated thread on this). Developers treat the URL as a static credential and don't build observability around it.

**How to avoid:**
- Log every outgoing webhook response code and body to an `outgoing_alerts` table — including non-2xx codes.
- Expose an admin-accessible health-check button in the settings UI that fires a test message and shows the last delivery timestamp + status code.
- Tie the webhook token to a dedicated service account (not tied to a crew member) so crew turnover doesn't break alerts.
- Store the webhook URL in an admin-configurable settings table, not as an env var or hardcoded constant. Rotation must not require a redeploy.

**Warning signs:**
- No log of non-2xx alert delivery attempts.
- Webhook URL stored in env var with no admin UI to update it.
- Token created under a named crew member's account.
- Last successful delivery timestamp is stale and nobody noticed.

**Phase to address:** Zoho Cliq integration phase — build delivery logging and the health-check UI before wiring any alert trigger.

---

### Pitfall 2: Alert Delivery Blocks the API Response (Synchronous HTTP Call)

**What goes wrong:**
The "Complete Shopping List" API handler fires `http.Post(webhookURL, ...)` inline before returning. If Zoho Cliq is slow (common during their maintenance windows), times out, or returns an error, the crew member's phone waits for the full timeout (default 30s in Go's http.Client). The "Complete" button appears frozen. On retry, the list is marked complete twice. Alert fails but the shopping action succeeded — the two are now coupled in the worst possible way.

**Why it happens:**
The inline POST is the simplest implementation. Decoupling via a queue feels like over-engineering for a small app. But external HTTP calls from within a request handler are universally considered a mistake in production Go services, regardless of app size.

**How to avoid:**
- Use an async alert queue: on shopping list completion, write a row to `outgoing_alerts` (status=pending) as part of the same DB transaction that marks the list complete. Return 200 to the client immediately.
- A background goroutine polls `outgoing_alerts` for pending rows and delivers them with exponential backoff (3 attempts: immediate → 30s → 5min).
- If all attempts fail, fall back to email using the configured fallback email address.
- Set a short timeout (5s) on the webhook HTTP client regardless — never use the default.

**Warning signs:**
- `http.Post(webhookURL, ...)` called directly inside an HTTP handler goroutine.
- No `outgoing_alerts` table in the schema.
- Alert delivery tested by asserting "the HTTP call was made" not "the HTTP call succeeded and was retried on failure."

**Phase to address:** Zoho Cliq integration phase — queue design must be specified in the DB schema before any alert trigger is built.

---

### Pitfall 3: Cutoff Race Condition — Edit Accepted After Deadline Passes

**What goes wrong:**
The cutoff is enforced by a guard clause: `if time.Now().After(cutoff) { return 403 }`. The crew member opens the PO form 45 seconds before cutoff. The guard passes. The crew member edits. The cutoff passes. The crew member submits. The guard has already been evaluated — the write goes through on a locked PO.

A second race: two people edit the same open PO simultaneously. Both load the form at version 3. Person A saves first (version becomes 4). Person B's save runs — the WHERE clause only checks `po_id`, not version — Person B's write silently overwrites Person A's changes.

**Why it happens:**
State validation is done in Go (application layer) before the DB write. The check and the write are separate operations with a gap between them. Concurrent edits are not considered because "only one person edits at a time" — which is true until the first time it isn't.

**How to avoid:**
- Enforce cutoff and state inside the DB transaction: `UPDATE purchase_orders SET ..., version = version + 1 WHERE id = $1 AND version = $2 AND (status = 'open') AND (cutoff_at IS NULL OR NOW() < cutoff_at) RETURNING id`. If 0 rows returned → 409 Conflict.
- Add a `version INTEGER NOT NULL DEFAULT 1` column to `purchase_orders` from the start.
- The edit form loads the current version and includes it in the submit body. The handler passes it to the WHERE clause.
- The frontend handles 409 with a clear message: "The PO was changed by someone else — refresh to see the latest version."

**Warning signs:**
- Cutoff check is a Go `if` statement before the `db.Exec(...)` call, not inside the WHERE clause.
- No `version` column on `purchase_orders`.
- No 409 handling in the frontend.
- The edit endpoint tests only pass/fail — no concurrent edit test.

**Phase to address:** PO state machine phase — add `version` column in the migration before building any edit endpoint.

---

### Pitfall 4: Go Cron DST Skip / Double-Fire for Cutoff Reminders

**What goes wrong:**
The weekly cutoff reminder ("PO cutoff in 2 hours") is scheduled via `robfig/cron`. The cron expression is written in local time without an explicit timezone: `0 7 * * 1` (every Monday at 7 AM). The server is on a US timezone with DST. When clocks spring forward:
- 2:00 AM jumps to 3:00 AM — any job scheduled in that window is skipped entirely.
- When clocks fall back, 1:30 AM occurs twice — the job fires twice.

For a food business with strict purchase deadlines, a skipped cutoff reminder means the owner misses the ordering window.

**Why it happens:**
`robfig/cron` v3 defaults to UTC but does not warn when cron expressions are written expecting local time. The mismatch only surfaces twice a year (DST transitions), which makes it hard to catch in testing.

**How to avoid:**
- Store all cutoff times in UTC in the database. Never store local-time strings.
- Initialize cron with explicit UTC: `cron.New(cron.WithLocation(time.UTC))`.
- Compute schedule fire times from the stored UTC cutoff timestamp, not from a static cron expression (e.g., `time.Until(nextCutoff) - 2*time.Hour`).
- Use `time.AfterFunc` for one-shot reminders computed from a specific stored timestamp — more precise than a recurring cron expression when the cutoff time is user-configured.
- Display cutoff times to users in browser local time using `Intl.DateTimeFormat` — never show UTC in the UI.
- Add an admin debug endpoint that shows the next scheduled reminder fire time in both UTC and local time.

**Warning signs:**
- Cron expression contains a hard-coded hour in local time.
- `cron.New()` called without `cron.WithLocation(time.UTC)`.
- Cutoff time stored as a string like "Monday 9 AM" instead of a UTC timestamp.
- No way to inspect next scheduled fire time without looking at server logs.

**Phase to address:** Cutoff scheduling phase — UTC-only timestamp storage must be in the schema design before any scheduling logic is written.

---

### Pitfall 5: Notion Image URLs Expire 1 Hour After Export

**What goes wrong:**
The Notion item catalog is exported as a CSV. Each row's image field contains a presigned AWS S3 URL with a 1-hour expiry (`X-Amz-Expires=3600`). The import script runs, inserts the raw Notion URLs into `catalog_items.photo_url`, and the item photos all display correctly during the import run. The next morning — or even 90 minutes later — every photo in the catalog shows a broken image. This has been widely documented by developers using the Notion API and export tooling.

**Why it happens:**
Notion generates time-limited presigned S3 URLs for all file attachments. The export CSV shows the full URL with no visible indication it will expire. Developers assume a URL in a database is permanent.

**How to avoid:**
- During the import run: for each row with a photo URL, immediately download the bytes (`http.Get(notionURL)`) and re-upload to DO Spaces using the existing presigned upload flow. Store the DO Spaces URL (permanent) in `catalog_items.photo_url`.
- Validate during import: log download failures to stderr; insert a NULL `photo_url` rather than a broken Notion URL.
- Run the import in a single pass — download + upload + insert atomically per row. Do not insert first and re-host later.
- After import, spot-check: verify photos still load 2 hours after the import completes.

**Warning signs:**
- Import script inserts raw `notion.so` or `amazonaws.com` URLs with expiry parameters.
- Photos are verified to work immediately after import but not after a delay.
- No image download + re-upload step in the import plan.

**Phase to address:** Notion data import phase — image re-hosting must be part of the import script, not a follow-up task.

---

### Pitfall 6: PO-to-Shopping-List State Drift After Admin Unlock

**What goes wrong:**
The shopping list is generated from an approved PO (1:1 mapping, snapshot copy of line items). The admin later unlocks the PO for a correction and edits quantities. The PO now has updated quantities. The shopping list still shows the original quantities. The shopper goes to the store based on the old list, buys the wrong quantities, and marks the list complete. The owner's approved PO no longer reflects what was actually purchased.

**Why it happens:**
The shopping list is treated as an independent entity after generation (snapshot approach). The PO is not locked when the list is generated, and no mechanism regenerates or flags the list when the PO changes. The state machine has a gap: "PO amended after shopping list created" is an unhandled transition.

**How to avoid:**
- Define the state machine explicitly before building any UI: once a shopping list is generated, the PO status changes to `shopping_in_progress` (locked for editing). Admin unlock must explicitly transition the PO back to `approved` and either invalidate (delete) the shopping list or surface a warning: "Shopping list was already sent — regenerating will reset shopper progress."
- Add a `generated_from_po_version INTEGER` column to shopping lists. When the PO version changes, flag the list as stale in the admin UI.
- Alternatively, the shopping list references PO line items by FK (not snapshot copy) — changes to the PO immediately reflect in the list. Choose this only if stale mid-shop data is acceptable (risky).

**Warning signs:**
- Shopping list items inserted via `INSERT INTO shopping_items SELECT ... FROM po_line_items` with no version tracking.
- No status transition on `purchase_orders` when the shopping list is generated.
- Admin unlock flow exists but shopping list state is not addressed in the PR.

**Phase to address:** Shopping list phase — state machine transitions must be specified in the task plan before any shopping list endpoint is built.

---

### Pitfall 7: "Repurchased" Badge Reset at UTC Midnight Instead of Business Timezone Midnight

**What goes wrong:**
The "Repurchased +3" badge on inventory items resets every Monday. The reset logic runs `time.Now().UTC().Truncate(7 * 24 * time.Hour)` — midnight UTC. For a food truck in Chicago (UTC-6 in winter, UTC-5 in summer), this means the badge resets on Sunday evening local time (6 PM Sunday or 7 PM Sunday). Crew members see the badge disappear mid-Sunday-shift. Worse, a purchase at 11 PM Sunday local time is attributed to Monday's count instead of Sunday's, breaking weekly totals.

**Why it happens:**
UTC truncation is the simplest calculation. Timezone-aware truncation requires knowing the business's local timezone, which developers defer because "we can add it later." It's the kind of bug that doesn't surface in tests using UTC fixtures.

**How to avoid:**
- Add a `business_timezone TEXT NOT NULL DEFAULT 'America/Chicago'` field to the settings table from the start.
- All week-boundary and reset computations: `time.Now().In(businessTZ)` → truncate to week → convert back to UTC for storage.
- Test with fixtures that cross midnight in the business timezone, not UTC.
- Display reset dates in the UI using the business timezone: "Resets Monday (Chicago time)."

**Warning signs:**
- Badge reset logic uses `time.Now().UTC()` without a timezone conversion.
- No `business_timezone` field in any settings table.
- Badge reset tests use UTC timestamps only.

**Phase to address:** Badge and repurchase tracking phase — add `business_timezone` to the settings table in its migration before writing any reset logic.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Inline webhook POST in HTTP handler | Simpler code, no queue to maintain | Blocks API response; silent drop on transient failure; no retry | Never — decouple from day one |
| Storing Notion image URLs without re-hosting | Fast import script | All photos broken within 1 hour; requires re-import | Never — always re-host during the import run |
| Cutoff check as Go guard clause before DB write | Simple and readable | Race condition window on cutoff boundary; last-write-wins on concurrent edits | Never for deadline-enforced state transitions |
| UTC-naive timestamps for cutoff and reset logic | No timezone conversion code during early dev | Wrong week boundaries for US business; DST bugs appear twice a year | Acceptable only in single-timezone dev with an explicit TODO and same-sprint fix |
| Hardcoded webhook URL in env var | Easy to set up initially | Requires redeploy to rotate; no admin recovery path on token expiry | Acceptable only if admin UI to update it is planned in the same phase |
| Shopping list as full snapshot copy of PO with no version link | Simpler queries | Silent drift on PO amendment | Acceptable if PO is hard-locked after list generation and admin unlock is explicitly blocked |
| SMTP relay for email fallback | No third-party API needed | Shared IP → poor sender reputation; lands in spam; 550/421 errors cryptic to debug | Never for a food business where email alerts must be reliable |

---

## Integration Gotchas

Common mistakes when connecting to external services.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Zoho Cliq channel webhook | Assume `zapikey` token is permanent | Log every response; surface delivery status in admin UI; tie token to a service account |
| Zoho Cliq | No retry on non-2xx | Async queue with 3-attempt exponential backoff (immediate → 30s → 5min) |
| Zoho Cliq | No fallback when all retries exhausted | Fall back to email on third failure; log the exhausted alert to `outgoing_alerts` |
| Zoho Cliq | Blocking API response on webhook delivery | Decouple: write intent to `outgoing_alerts` first, return 200, deliver in background goroutine |
| Email (transactional) | Direct SMTP from Go server on a VPS | Shared IP = poor deliverability; use an API-based transactional service (Resend, Mailgun, Postmark) with SPF/DKIM/DMARC configured |
| Email | No SPF/DKIM records for sending domain | Email sent from `yumyums.com` via DO server IP with no SPF goes straight to spam |
| Notion export | Using raw CSV image URLs as permanent references | Notion S3 presigned URLs expire in 1 hour — download and re-host to DO Spaces during the import run |
| Notion export | Trusting item name uniqueness in the export | Notion has no enforced unique constraint; duplicate names possible — deduplicate on lowercase-trimmed name during import |
| DO Spaces | Reusing a presigned upload URL across multiple items | Presigned upload URLs are single-use; generate a fresh URL per item during import |
| `robfig/cron` | Cron expression in local time without CRON_TZ | Schedule fires at the wrong UTC time; silently double-fires or skips on DST transitions |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| N+1 query loading stock levels for PO reorder suggestions | PO suggestion page slow as catalog grows | Batch: `SELECT ... WHERE item_id = ANY($1)` | Noticeable at ~200 catalog items |
| Polling `outgoing_alerts` without an index on `(status, created_at)` | Background delivery goroutine causes sequential scans every poll interval | Add the index in the `outgoing_alerts` migration | Noticeable at ~1,000 alerts (unlikely here, but the index is trivially cheap) |
| Shopping list render joining PO → line items → catalog → photos with no index on FK columns | List render slow on mobile, especially on spotty food truck WiFi | Index `po_line_items.po_id` and `catalog_items.id` from the migration | Noticeable at ~50-item POs |
| Cron goroutine blocks during webhook delivery | Cron misses its next tick; subsequent reminders delayed | Alert delivery must be fire-and-forget from the cron tick perspective; use a goroutine or the async queue | From day one if Zoho response time is unpredictable (often 1-3s) |
| Full PO history loaded for suggestion UI | PO form slow to open as order history grows | Limit suggestion query to last 8 weeks of purchases; add `LIMIT` and index on `purchased_at` | Noticeable at ~52 weeks of weekly orders |

---

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Webhook token visible in server logs | Token exposure → attacker can post messages to your Cliq channel | Mask token in all log output; never log the raw webhook URL |
| Shopping list "Complete" endpoint not role-checked on the server | Any authenticated user can mark any list complete regardless of assignment | Enforce RBAC in the Go handler; the frontend hiding the button is not sufficient |
| PO approval endpoint accessible by crew role | Crew member approves their own PO | Handler must check `currentUser.Role >= manager`; return 403 otherwise |
| PO status transitions not written to audit log | Owner can't tell who locked, unlocked, or approved a PO | Write to the existing audit log on every `purchase_orders.status` transition |
| Email fallback reveals other crew members' email addresses | Privacy violation; crew members can see each other's personal email | Send alerts only to the configured recipient; never CC or mention other crew email addresses in alert bodies |
| Admin settings endpoint (webhook URL update) not restricted to admin role | Any crew member can redirect alerts to an attacker-controlled endpoint | Settings write endpoints must enforce `role = admin` check in the handler |

---

## UX Pitfalls

Common user experience mistakes in this domain.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No indication when cutoff has passed mid-edit | Crew submits form, gets 409, sees a generic error, data appears lost | Show a sticky banner "Cutoff passed — only admins can now edit this PO" when the page detects cutoff has passed (check on save attempt; show graceful message) |
| Badge shows "Repurchased +3" with no visible reset date | Crew doesn't know when count resets; badge loses meaning after a week | Show "Resets Monday" or the exact configured reset date directly below the badge count |
| Shopping list has no item-level persistence | Shopper completes half the list, navigates away, completion is lost | Persist item-level completion using the same autoSave pattern from workflows — one row per item in a `shopping_completions` table |
| Alert message body contains no item detail | Alert says "Shopping complete — missing items" with no list of what's missing | Include item names, quantities, and store location in the alert message body |
| Cutoff time shown in UTC in the admin settings UI | Owner sets cutoff at the wrong wall-clock time | Always display and accept times in browser local timezone using `Intl.DateTimeFormat`; convert to UTC for storage in Go |
| PO "saved" confirmation shown even when webhook alert failed | Owner believes the alert was sent; shopper never notified | Decouple save success from alert delivery status; show alert delivery status separately (e.g., "Alert sent" / "Alert pending") |
| Notion import has no progress indicator | Import of ~100 items with photos (each requiring a DO upload) looks frozen | Show import progress: "Importing item 42 of 100 (downloading photo...)" — or a simple spinner with estimated time |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Cutoff enforcement:** Cutoff guard is in the Go handler — verify it is also enforced inside the DB transaction WHERE clause (not just a pre-check in Go code).
- [ ] **Zoho Cliq integration:** Webhook fires in dev — verify `outgoing_alerts` table exists, non-2xx responses are logged, and the admin health-check shows last delivery timestamp.
- [ ] **Notion import photos:** Photos display immediately after import — verify they still display 2 hours later (Notion URL expiry window is 1 hour).
- [ ] **Shopping list RBAC:** "Complete" button is hidden from non-assigned users — verify the `/complete` API endpoint also rejects non-assigned users with 403.
- [ ] **Badge reset timezone:** Badge clears on Monday — verify it clears at Monday midnight in the business's configured local timezone, not UTC midnight.
- [ ] **Email fallback deliverability:** Email is configured as fallback — verify SPF and DKIM DNS records are set for the sending domain. Send a test email and check spam folder.
- [ ] **Alert retry:** Alert delivery goroutine runs — verify that a transient 500 from Zoho triggers a retry attempt, not a permanent failed mark after the first attempt.
- [ ] **Optimistic locking:** PO edit saves — open the PO on two devices simultaneously, edit on both, submit both — verify the second returns a 409 with a meaningful message, not a silent overwrite.
- [ ] **Admin unlock + shopping list:** PO is unlocked after shopping list is generated — verify the shopping list is either regenerated or flagged as stale; it must not silently show outdated quantities.

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Webhook token expired, alerts stopped | LOW | Update token in admin settings UI; send test via health-check; review `outgoing_alerts` for missed alerts and determine if manual notification is needed |
| Notion images broken after import | MEDIUM | Query all `catalog_items` with Notion/S3 URLs → re-download → re-upload to DO Spaces → UPDATE rows. Script takes ~10min for 100 items |
| Cutoff race caused wrong data saved | MEDIUM | Point-in-time restore from Postgres WAL if within backup window; post-fix: add optimistic locking + WHERE clause cutoff enforcement |
| Shopping list diverged from PO after admin unlock | HIGH | Manual reconciliation: compare `po_line_items` with `shopping_items` for the affected PO; patch shopping list quantities; add PO version tracking to prevent recurrence |
| Alert sent to wrong channel (misconfigured webhook URL) | LOW | Update webhook URL in admin settings; the alert was noise on the wrong channel — no data loss |
| DST caused missed cutoff reminder | LOW | Manually trigger reminder via admin debug endpoint; convert all schedules to UTC-based `time.AfterFunc` to prevent recurrence |
| UTC midnight badge reset fires Sunday evening local time | MEDIUM | Add `business_timezone` to settings table; migrate existing reset logic; backfill: the affected week's counts are off by one calendar day — acceptable to leave, fix going forward |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Zoho Cliq token expiry / silent drop | Zoho Cliq integration (build alert queue + delivery log first) | Admin health-check fires test message and returns last delivery timestamp + status code |
| Synchronous webhook blocking API response | Zoho Cliq integration (async queue design before any trigger) | Load test: delay Zoho response by 10s — shopping list Complete returns in <100ms |
| Cutoff race condition | PO state machine phase (version column + WHERE clause enforcement) | Concurrent edit test: two clients edit the same PO simultaneously → one gets 409 with clear message |
| Go cron DST timezone | Cutoff scheduling phase (UTC-only from DB schema) | Admin debug endpoint shows next reminder fire time in both UTC and local time; verify across DST boundary |
| Notion image URL expiry | Data import phase (re-host images in import script) | Check photo URLs 2 hours after import — must still load |
| PO → shopping list state drift | Shopping list phase (state machine spec before any UI) | PO amendment after list generation → list flagged stale or regenerated; shopper sees updated quantities |
| Badge reset timezone mismatch | Badge/repurchase phase (business_timezone in settings) | Simulate purchase at 11 PM local Sunday → badge appears in Sunday count; Monday local midnight resets it |
| Shopping list RBAC not enforced server-side | Shopping list phase (RBAC on every endpoint) | Non-assigned crew member calls Complete endpoint directly → 403 |

---

## Sources

- [Zoho Cliq Channel Webhooks documentation](https://www.zoho.com/cliq/help/platform/channel-webhooks.html)
- [Zoho Cliq webhook token limit (5 per user) and 2FA session expiry](https://www.zoho.com/cliq/help/platform/webhook-tokens.html)
- [Zoho community: 401 Unauthorized on incoming webhook — known failure mode](https://help.zoho.com/portal/en/community/topic/zoho-cliq-incoming-webhook-changes-and-401-issues)
- [robfig/cron GitHub — CRON_TZ support, DST-related issues, v3 changelog](https://github.com/robfig/cron)
- [netresearch/go-cron fork — DST handling notes and upstream gaps](https://github.com/netresearch/go-cron)
- [CronMonitor: Handling Timezone Issues in Cron Jobs 2025](https://cronmonitor.app/blog/handling-timezone-issues-in-cron-jobs)
- [DST pitfalls — spring forward skip, fall back double-fire](https://cronjob.live/docs/dst-pitfalls)
- [PostgreSQL race conditions — SELECT FOR UPDATE, optimistic locking patterns](https://oneuptime.com/blog/post/2026-01-25-postgresql-race-conditions/view)
- [Optimistic vs pessimistic locking in PostgreSQL with Go](https://hackernoon.com/comparing-optimistic-and-pessimistic-locking-with-go-and-postgresql)
- [Notion image URL 1-hour expiry — real-world developer problem and fix](https://snugl.dev/archive/fixing-notions-1-hour-expiring-image-problem)
- [Notion API file expiry — developer experience report](https://www.danvega.dev/blog/notion-api-file-expired)
- [Notion S3 image URL expiry confirmed via nuxt/image issue discussion](https://github.com/nuxt/image/issues/1340)
- [Notion export: imports append rows, no deduplication on native import](https://clonepartner.com/blog/how-to-import-data-into-notion-formats-limits-data-mapping)
- [SMTP vs API email delivery — transactional pitfalls and shared IP reputation risk](https://www.mailgun.com/blog/email/difference-between-smtp-and-api/)
- [SMTP from VPS deliverability — goes to spam without SPF/DKIM](https://mailtrap.io/blog/smtp-vs-email-api/)
- [PWA Badging API — platform support matrix, iOS requirements](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/How_to/Display_badge_on_app_icon)
- [HackerNews: Avoid 2:00–3:00 AM cron jobs — DST discussion](https://news.ycombinator.com/item?id=45723554)

---
*Pitfalls research for: Purchase orders, shopping lists, Zoho Cliq alerts, scheduled cutoffs, Notion data import — v3.0 milestone*
*Researched: 2026-04-22*
