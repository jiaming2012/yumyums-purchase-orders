# Project Research Summary

**Project:** Yumyums HQ v3.0 — Purchase Orders & Shopping Lists
**Domain:** Food service procurement workflow — PO creation, cutoff enforcement, shopping execution, and alert delivery
**Researched:** 2026-04-22
**Confidence:** HIGH

## Executive Summary

Yumyums v3.0 closes the ordering loop that the existing `purchasing.html` mockup only sketches: from reorder suggestions drawn from live inventory par levels, through a weekly cutoff window, admin approval, shopping checklist execution, and back to the inventory Stock tab via a "Repurchased" badge. The foundation is already built — item catalog, stock thresholds, vendor assignments, receipt pipeline, RBAC, and the fullscreen picker modal pattern all exist in `inventory.html` and the backend. v3.0 is an extension, not a rebuild. The recommended implementation path mirrors the receipt worker pattern already in the codebase: a new `internal/purchasing` Go package, a goroutine-based cutoff scheduler, and an async alert queue that decouples external notification delivery from API response time.

The most important architectural decision is treating the shopping list as an immutable snapshot created at approval time (not a live JOIN view of the PO), and enforcing PO state transitions inside Postgres WHERE clauses — not only as Go guard clauses — to prevent cutoff race conditions and concurrent-edit overwrites. External alerts to Zoho Cliq must be written to an `outgoing_alerts` queue table as part of the same DB transaction that triggers the alert event, then delivered by a background goroutine with exponential backoff. Blocking the API handler on an outbound webhook call is the single highest-risk implementation shortcut in this milestone.

Two pitfalls have the highest recovery cost: Notion item photos stored as raw Notion S3 URLs expire in one hour and must be re-hosted to DO Spaces during the import run (not as a follow-up step); and shopping list state drift when an admin unlocks a PO after the list is already generated — the state machine must explicitly define this transition before any shopping list endpoint is built.

## Key Findings

### Recommended Stack

The v2.0 backend stack (Go 1.22+, chi, pgx/v5, sqlc, goose) is stable and unchanged. v3.0 adds three targeted libraries: `go-co-op/gocron/v2@v2.21.1` for the cutoff scheduler (replaces the unmaintained `robfig/cron` which has open DST panic bugs), `wneessen/go-mail@v0.7.2` for transactional email alerts (the standard `net/smtp` is frozen), and Go stdlib `net/http` and `encoding/csv` for the Zoho Cliq webhook and Notion CSV import respectively — no third-party library is justified for either. See `.planning/research/STACK.md` for installation commands and full integration patterns.

**Core technologies:**
- `go-co-op/gocron/v2@v2.21.1`: cutoff scheduler — only actively-maintained Go cron library with timezone support and test mocks; `robfig/cron` is unmaintained since 2020 with open panic bugs
- `wneessen/go-mail@v0.7.2`: email fallback alerts — `net/smtp` is frozen; `gomail.v2` unmaintained since 2016
- `net/http` stdlib: Zoho Cliq webhook outbound POST — no third-party library justified for one webhook
- `encoding/csv` stdlib: Notion CSV import — 100-row one-time seed; struct-tag mapping libraries are overkill
- `go-co-op/gocron/mocks/v2@v2.21.1`: scheduler test mocks — required for deterministic cutoff tests without waiting for real deadlines

### Expected Features

v3.0 must close the end-to-end ordering loop. Every P1 item is required to ship — none can be deferred without breaking the loop. See `.planning/research/FEATURES.md` for the full prioritization matrix and anti-feature rationale.

**Must have (table stakes — v3.0 launch):**
- PO form pre-populated from live inventory reorder suggestions (items where `current_stock < low_threshold`)
- Item search / add from catalog via fullscreen picker modal (established pattern from inventory)
- Admin-configurable weekly cutoff schedule (day-of-week + time, stored in DB — not hardcoded)
- Cutoff enforcement: backend auto-lock after configured deadline; admin edit override with reason
- PO approval by admin generates shopping list as immutable snapshot of approved line items
- Shopping list check-off per item, store location notes, assignable to role or user
- Shopping list completion with missing items report
- Zoho Cliq channel alert on cutoff reminder and shopping completion
- "Repurchased +[Qty]" badge on inventory Stock tab items after shopping confirmed
- Backend cutoff simulation command (admin-only, required for QA without waiting for the real weekly cutoff)

**Should have (competitive differentiators — add post-launch when triggered by real usage):**
- Out-of-stock / low-stock alerts (validate cutoff reminder alert fatigue first)
- Shopping list grouped by store/vendor (add when crew reports backtracking during shopping runs)
- Per-user notification preference (Cliq vs. email) (add when team grows beyond owner + 1 crew member)
- Email fallback notification channel (add if owner reports missing Cliq alerts on mobile)

**Defer (v3.x or v4+):**
- Notion item catalog import as a recurring feature — run as a one-shot admin endpoint before launch; not a user-facing recurring feature
- PO history spending trends (extend existing Chart.js Trends tab)
- Multi-location PO (explicitly out-of-scope until second truck is operational)
- Vendor EDI/API integration (never appropriate at 1-5 person scale)
- Auto-submit PO to vendor (removes human judgment; wrong for this scale)

### Architecture Approach

v3.0 adds a new `internal/purchasing` Go package (handler, service, types, scheduler, notifier) mounted at `/api/v1/purchasing/*`, mirroring the structure of `internal/inventory`. Six new goose migrations (0034–0039) create the required tables. `purchasing.html` is rewritten from a ~90-line hardcoded mock to a real 3-tab tool (Order / Locked / Shopping). The core data flow: crew edits PO via upsert endpoint → scheduler auto-locks at cutoff → admin approves in Tab 2 → shopping list snapshot created in a single Postgres transaction → shopper checks off items in Tab 3 → completion triggers async Cliq alert and writes repurchase badges → Stock tab picks up badge via LEFT JOIN in existing stock query. No new pages are added to `index.html`. See `.planning/research/ARCHITECTURE.md` for the full schema, API route table, and phase dependency graph.

**Major components:**
1. `purchasing/handler.go` — HTTP I/O, auth/RBAC checks, JSON marshal for all purchasing routes
2. `purchasing/service.go` — business logic: generate shopping list snapshot, lock PO, approval side effects
3. `purchasing/scheduler.go` — cutoff auto-lock goroutine following `receipt/worker.go` model; 15-minute ticker
4. `purchasing/notifier.go` — single entry point for Zoho Cliq + email; reads `outgoing_alerts` queue; exponential backoff (immediate → 30s → 5min)
5. `purchasing.html` (full rewrite) — 3-tab tool: Order form, Locked PO view, Shopping checklist
6. 6 new goose migrations (0034–0039) — `purchase_orders`, `po_line_items`, `shopping_lists`, `shopping_list_items`, `cutoff_config`, `alert_log`, `repurchase_badges`, `user_notification_prefs`, plus ALTER to `purchase_items` for `photo_url` and `store_location`

### Critical Pitfalls

Full detail including verification checklists in `.planning/research/PITFALLS.md`.

1. **Zoho Cliq webhook token silently revoked** — tie token to a service account (not a crew member), store URL in admin-configurable DB settings (not env var), log every outbound response, expose health-check button in admin UI. Build delivery logging before wiring any alert trigger.

2. **Synchronous webhook call blocks API response** — never call `http.Post(webhookURL, ...)` inside a request handler. Write intent to `outgoing_alerts` table in the same DB transaction, return 200 immediately, deliver in background goroutine with retries. Queue design must be in the DB schema before any alert trigger is built.

3. **Cutoff race condition (edit accepted after deadline)** — enforce cutoff state inside the Postgres WHERE clause: `UPDATE purchase_orders ... WHERE status = 'draft' AND NOW() < cutoff_at AND version = $version RETURNING id`. Add a `version INTEGER` column from the first migration. Application-layer guard clauses alone are insufficient.

4. **Notion image URLs expire in 1 hour** — Notion exports S3 presigned URLs with 1-hour TTL. Download and re-upload each image to DO Spaces during the import run, per item, before inserting the row. Never insert a raw Notion URL. Verify photos still load 2 hours after import.

5. **Shopping list state drift after admin PO unlock** — once a shopping list is generated, the PO must transition to `shopping_in_progress` (write-locked). Admin unlock must either regenerate the list or flag it stale with a warning. Specify all state machine transitions before building any shopping list endpoint.

6. **Badge reset at UTC midnight instead of business timezone midnight** — add `business_timezone TEXT` to settings table from the start. All week-boundary reset computations use `time.Now().In(businessTZ)`. DST-naive UTC truncation fires at the wrong local time twice a year.

## Implications for Roadmap

The architecture research defines a hard-dependency build order. Phases cannot be reordered without breaking downstream features. Each phase is a deployable increment that can be tested in isolation.

### Phase 1: Data Foundation

**Rationale:** All subsequent phases require the new tables. Migrations must land and be verified before any handler writes to them. The `version` column on `purchase_orders` and `business_timezone` in a settings table must be in this migration — adding them later requires a table rebuild or a second ALTER that risks missing data.
**Delivers:** Migrations 0034–0039, `purchase_items` extended with `photo_url` and `store_location`, existing `/api/v1/inventory/items` API returns new fields.
**Addresses:** Schema for all v3.0 features
**Avoids:** Pitfall 3 (`version` column on `purchase_orders` from day one); Pitfall 6 (`business_timezone` in settings from day one)

### Phase 2: PO Form (purchasing.html Tab 1)

**Rationale:** PO creation is the entry point of the entire workflow. Nothing else can be tested end-to-end without it. The upsert pattern (`PUT /orders/:id/items` with full item list, qty=0 means remove) must be established here — it is referenced by shopping list generation in Phase 5.
**Delivers:** `purchasing.html` rewritten from mock to real API, Tab 1 functional — reorder suggestions from live stock data, item search via fullscreen picker, stepper qty input.
**Uses:** Existing `GET /api/v1/inventory/stock` and `GET /api/v1/inventory/items` as data sources; new `POST /api/v1/purchasing/orders` (get-or-create) and `PUT /api/v1/purchasing/orders/:id/items` upsert.
**Implements:** `purchasing/handler.go` + `purchasing/service.go` skeleton

### Phase 3: Notion Catalog Import

**Rationale:** Seeds item photos and store locations before the PO form UI needs to display them. Independent of Phases 4-7 — can run in parallel with Phase 4 if capacity allows. Must happen early so catalog is complete before crew uses the PO form in production.
**Delivers:** `POST /api/v1/purchasing/import/notion` admin endpoint; Notion CSV parsed, images downloaded and re-hosted to DO Spaces, items upserted idempotently into `purchase_items`.
**Avoids:** Pitfall 4 (image re-hosting must happen during import — not a follow-up task)

### Phase 4: Cutoff Enforcement + Lock

**Rationale:** Cutoff is the gate between the editing phase and the approval/shopping phase. A locked PO must exist before Phase 5 can approve it. The scheduler goroutine must be operational before Phase 6 can wire reminder alerts to it.
**Delivers:** Admin-configurable cutoff schedule (`cutoff_config` table), `purchasing/scheduler.go` goroutine (auto-lock on 15-minute ticker), manual lock endpoint, Tab 2 locked view in `purchasing.html`, backend cutoff simulation endpoint.
**Uses:** `go-co-op/gocron/v2` for scheduling; UTC-only timestamp storage for cutoff times
**Avoids:** Pitfall 3 (WHERE clause cutoff enforcement — not a Go guard clause); Pitfall 4b (DST-safe scheduling via UTC storage and `gocron.WithLocation`)
**Research flag:** Standard pattern — directly mirrors `receipt/worker.go` goroutine model; no additional research needed

### Phase 5: Admin Approval + Shopping List

**Rationale:** Shopping list is the execution artifact derived from an approved PO. This is the highest-complexity phase and the gate for Tab 3, alerts, and badges. The state machine (draft → locked → approved → shopping_in_progress → complete) must be fully defined in the task plan before any endpoint is built.
**Delivers:** Approval endpoint (status transition + shopping list snapshot in single transaction), Tab 3 shopping checklist (check-off, store notes, assignment), completion endpoint with missing items report.
**Avoids:** Pitfall 5 (state machine transition for admin unlock must be specified; shopping list flagged stale or regenerated — not silently stale)
**Research flag:** Full state machine transition diagram required before implementation begins. Unhandled transitions (PO unlock after list generated, list cancellation) are the highest-risk gap in this milestone.

### Phase 6: Alerts + Notification Prefs

**Rationale:** Alert infrastructure depends on the scheduler (Phase 4) and the shopping completion event (Phase 5). Notifier must be built after both triggers exist. The `outgoing_alerts` queue schema must be finalized here even though it is referenced in Phase 1 migrations — include the table in Phase 1, wire the delivery goroutine in Phase 6.
**Delivers:** `purchasing/notifier.go` with async `outgoing_alerts` queue delivery goroutine, Zoho Cliq webhook with 3-attempt exponential backoff, email fallback, cutoff reminder scheduling, shopping completion alert with missing item detail. Notification prefs UI section added to `users.html`.
**Uses:** `wneessen/go-mail@v0.7.2` for SMTP; `net/http` stdlib for Cliq webhook
**Avoids:** Pitfall 1 (delivery logging + admin health-check UI); Pitfall 2 (async queue — never inline webhook in handler)
**Research flag:** Zoho Cliq token lifecycle and operational failure modes are sparsely documented by Zoho. Build the admin health-check button and `outgoing_alerts` delivery log before wiring any production alert trigger. Validate with a live test message to a real Cliq channel before declaring done.

### Phase 7: Repurchase Badges

**Rationale:** Badge is written at shopping completion — requires Phase 5. The inventory Stock tab's existing `COALESCE(override, sum)` query picks up badge data via an additive LEFT JOIN, making this the least risky phase. Badge reset timezone correctness depends on `business_timezone` from Phase 1.
**Delivers:** Badge write on shopping completion, badge display in Stock tab (additive JOIN in `GetStockHandler`), badge reset (manual per item + scheduled weekly), admin badge reset-date configuration.
**Avoids:** Pitfall 6 (badge reset uses `time.Now().In(businessTZ)` — not UTC truncation)

### Phase Ordering Rationale

- Phases 1-3 can be built and tested without any cutoff or shopping logic, allowing early catalog setup and UI validation on real devices.
- Phase 3 (Notion import) is independent of Phases 4-7 and can run in parallel with Phase 4 if capacity allows.
- Phase 4 is the dependency gate for Phase 5 (a locked PO must exist to approve).
- Phase 5 is the dependency gate for Phases 6 and 7 (alerts and badges both trigger from shopping completion).
- The WebSocket hub (already in `sync/hub.go`) should broadcast a `PO_LOCKED` event in Phase 4 to avoid frontend polling — this matches the existing `workflows.html` real-time pattern.
- The `outgoing_alerts` and `alert_log` tables serve different purposes and must be defined separately: `alert_log` for deduplication (prevents sending the same alert twice), `outgoing_alerts` for async delivery queue (retry state). Both tables belong in Phase 1 migrations even though they are activated in Phase 6.

### Research Flags

Phases needing deeper research or explicit specification during planning:
- **Phase 5 (Shopping List):** Full PO state machine transition diagram required before any endpoint is built. The transition when an admin unlocks a PO after the shopping list is generated is the highest-risk unspecified path.
- **Phase 6 (Alerts):** Zoho Cliq token lifecycle, `outgoing_alerts` queue schema and retry logic need explicit task plan before implementation. Validate with a live Cliq channel test before wiring production triggers.

Phases with standard patterns (skip additional research):
- **Phase 1 (Migrations):** Standard goose migration pattern; next migration number is 0034
- **Phase 2 (PO Form):** Mirrors existing inventory handler patterns directly; fullscreen picker modal already implemented
- **Phase 3 (Notion Import):** Scoped; image re-hosting via existing DO Spaces presigned flow
- **Phase 4 (Scheduler):** Directly mirrors `receipt/worker.go` goroutine + ticker model
- **Phase 7 (Badges):** Additive LEFT JOIN to existing stock query; same override table already in use

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | gocron v2 and go-mail verified against pkg.go.dev and GitHub releases; versions pinned; stdlib choices have no compatibility concerns |
| Features | MEDIUM-HIGH | Table stakes verified against MarketMan/BlueCart feature analysis; Zoho Cliq endpoint confirmed via official docs; cutoff/badge patterns inferred from existing mockup and industry norms |
| Architecture | HIGH | Derived from direct live codebase read — handler, migration, worker, and hub files inspected directly; not inferred from docs |
| Pitfalls | HIGH (technical) / MEDIUM (Zoho) | PO state machine, timezone, Notion URL expiry verified via multiple sources; Zoho failure modes based on community evidence — official docs are sparse on operational failures |

**Overall confidence:** HIGH

### Gaps to Address

- **`outgoing_alerts` vs `alert_log` naming:** ARCHITECTURE.md uses `alert_log` for deduplication; PITFALLS.md uses `outgoing_alerts` for the async delivery queue. These are two distinct tables with different purposes. The Phase 1 migration must define both with explicit column contracts — clarify naming and schema before migration is written.

- **Email provider in use for v2.0 invite flow:** FEATURES.md mentions "Resend or Postmark already used for invite emails in v2.0" but the specific provider was not confirmed from the codebase. Verify in `main.go` / env vars before specifying the Phase 6 email fallback implementation — the go-mail SMTP host/port config depends on which provider is in use.

- **Zoho Cliq service account setup:** The webhook token must be tied to a service account (not a crew member) before any production alert is configured. This is an operational prerequisite for Phase 6, not a code change — owner must create the service account and generate the webhook token against it before Phase 6 is testable.

- **`business_timezone` value:** Research recommends `America/Chicago` as the default for a Chicago food truck. Confirm the owner's actual timezone before Phase 1 migration is written so the default is correct.

## Sources

### Primary (HIGH confidence)
- Live codebase — `backend/cmd/server/main.go`, `internal/inventory/handler.go`, `internal/receipt/worker.go`, `internal/sync/hub.go`, `internal/db/migrations/0033_stock_count_overrides.sql`, `purchasing.html` (mockup)
- [go-co-op/gocron/v2 pkg.go.dev](https://pkg.go.dev/github.com/go-co-op/gocron/v2) — v2.21.1 verified, API signatures
- [wneessen/go-mail GitHub releases](https://github.com/wneessen/go-mail/releases) — v0.7.2 security patch, maintenance status
- [Notion export help](https://www.notion.com/help/export-your-content) — CSV format, UTF-8, column behavior
- [Notion S3 image URL expiry](https://snugl.dev/archive/fixing-notions-1-hour-expiring-image-problem) — 1-hour TTL confirmed via multiple developer reports
- [Zoho Cliq Post to Channel API](https://www.zoho.com/cliq/help/platform/post-to-channel.html) — endpoint URL and zapikey auth confirmed

### Secondary (MEDIUM confidence)
- [MarketMan Purchase & Order Management](https://www.marketman.com/platform/restaurant-purchasing-software-and-order-management) — food service procurement feature patterns
- [Zoho Cliq community: 401 on incoming webhook](https://help.zoho.com/portal/en/community/topic/zoho-cliq-incoming-webhook-changes-and-401-issues) — token expiry failure mode
- [PostgreSQL optimistic locking with Go](https://hackernoon.com/comparing-optimistic-and-pessimistic-locking-with-go-and-postgresql) — version column + WHERE clause pattern
- [DST pitfalls for cron — spring skip / fall double-fire](https://cronjob.live/docs/dst-pitfalls) — DST scheduling risk
- [SMTP deliverability from VPS](https://mailtrap.io/blog/smtp-vs-email-api/) — shared IP reputation risk; API-based transactional email recommended

### Tertiary (LOW confidence / needs validation)
- Existing `purchasing.html` mockup — UI intent for tab layout, contributor initials, locked view (HIGH confidence as UI spec; LOW confidence on backend assumptions embedded in the hardcoded mock)

---
*Research completed: 2026-04-22*
*Ready for roadmap: yes*
