---
gsd_state_version: 1.0
milestone: v3.1
milestone_name: Toast Integration & COGS Pipeline
status: milestone_archived
stopped_at: v3.1 archived; awaiting next milestone
last_updated: "2026-06-06T12:40:00Z"
last_activity: 2026-06-06 -- Completed quick task 260606-hew: Persist Mercury BankDescription as pending vendor
progress:
  total_phases: 8
  completed_phases: 6
  total_plans: 25
  completed_plans: 20
  percent: 75
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-15)

**Core value:** Operational tools that let the owner manage crew workflows and training from one mobile app — with accountability and smart conditions.
**Current focus:** Phase 999.2 — per-menu-item-cogs-attribution-via-recipe-bom-mapping

## Current Position

Phase: 999.2
Plan: Not started
Status: Milestone complete
Last activity: 2026-06-30 - Completed quick task 260630-mav: Inventory Purchases tab — red "Missing Receipt" badge for no-attachment pending purchases

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**

- Total plans completed: 38 (v1.0 + v1.1)
- Average duration: ~12 min
- Total execution time: ~3.6 hours

**By Phase (v2.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 09-foundation-auth | TBD | - | - |
| 10-workflows-api | TBD | - | - |
| 11-onboarding-users-admin | TBD | - | - |
| 12-inventory-photos | TBD | - | - |
| 21 | 3 | - | - |
| 22.1 | 5 | - | - |
| 22 | 6 | - | - |
| 999.2 | 6 | - | - |

**Recent Trend:**

- Last 5 plans (v1.1): 5, 20, 25, 3, 30 min
- Trend: Variable

*Updated after each plan completion*
| Phase 09-foundation-auth P02 | 3 | 2 tasks | 10 files |
| Phase 10-workflows-api P01 | 3 | 3 tasks | 12 files |
| Phase 10-workflows-api P02 | 5 | 3 tasks | 3 files |
| Phase 10 P03 | 8 | 2 tasks | 2 files |
| Phase 10-workflows-api P04 | 3 | 2 tasks | 2 files |
| Phase 10-workflows-api P05 | 2 | 1 tasks | 2 files |
| Phase 10.1-cross-device-state-sync P01 | 2 | 2 tasks | 3 files |
| Phase 10.1-cross-device-state-sync P02 | 515585 | 2 tasks | 5 files |
| Phase 10.1 P03 | 3 | 1 tasks | 1 files |
| Phase 10.1-cross-device-state-sync P04 | 515643 | 2 tasks | 3 files |
| Phase 10.1-cross-device-state-sync P05 | 8 | 1 tasks | 3 files |
| Phase 10.2-reactive-sync-framework P01 | 3 | 2 tasks | 3 files |
| Phase 10.2-reactive-sync-framework P02 | 30 | 1 tasks | 3 files |
| Phase 10.2-reactive-sync-framework P03 | 58 | 2 tasks | 6 files |
| Phase 11-onboarding-users-admin P01 | 126 | 2 tasks | 7 files |
| Phase 11-onboarding-users-admin P02 | 12 | 2 tasks | 3 files |
| Phase 11-onboarding-users-admin P04 | 420 | 2 tasks | 4 files |
| Phase 11-onboarding-users-admin P03 | 4 | 2 tasks | 3 files |
| Phase 11-onboarding-users-admin P05 | 327 | 2 tasks | 2 files |
| Phase 11-onboarding-users-admin P06 | 120 | 2 tasks | 4 files |
| Phase 12-inventory-photos-tile-permissions P03 | 1 | 2 tasks | 2 files |
| Phase 12-inventory-photos-tile-permissions P01 | 3 | 2 tasks | 8 files |
| Phase 12-inventory-photos-tile-permissions P02 | 5 | 2 tasks | 5 files |
| Phase 12-inventory-photos-tile-permissions P05 | 4 | 2 tasks | 7 files |
| Phase 12 P04 | 4 | 2 tasks | 2 files |
| Phase 12 P06 | 4 | 2 tasks | 4 files |
| Phase 13-integration-fixes P01 | 8 | 2 tasks | 6 files |
| Phase 13-integration-fixes P02 | 583 | 3 tasks | 4 files |
| Phase 01-onboarding-video-upgrade P02 | 2 | 1 tasks | 3 files |
| Phase 01-onboarding-video-upgrade P01 | 15 | 2 tasks | 5 files |
| Phase 01-onboarding-video-upgrade P03 | 30 | 2 tasks | 3 files |
| Phase 16-cutoff-approval-and-shopping-list P05 | 5 | 3 tasks | 3 files |
| Phase 16-cutoff-approval-and-shopping-list P04 | 35 | 2 tasks | 3 files |
| Phase 17 P01 | 8 | 5 tasks | 18 files |
| Phase 17 P02 | 12 | 2 tasks | 5 files |
| Phase 17 P05 | 12 | 2 tasks | 7 files |
| Phase 19 P02 | 267 | 3 tasks | 5 files |
| Phase 19 P01 | 5 | 4 tasks | 2 files |
| Phase 18 P01 | 159 | 2 tasks | 3 files |
| Phase 18 P02 | 264 | 2 tasks | 1 files |

## Accumulated Context

### Roadmap Evolution

- Phase 10.1 inserted after Phase 10: Cross-Device State Sync (URGENT)
- Phase 10.2 inserted after Phase 10.1: Reactive Sync Framework (URGENT) — shared Store + single write channel before Phase 11
- v2.1 milestone started: Onboarding Video Upgrade — Phase 1 added
- Phase 21 added: COGS in sales-processor report + receipt completeness gate before payroll — exposes HQ `purchase_events` data to the cross-repo `sales-processor` weekly payroll flow via new `GET /api/v1/inventory/period-summary` endpoint; blocks payroll PDF / OnPay CSV / Mercury transfers when receipts incomplete (unconfirmed `pending_purchases` or `purchase_line_items.purchase_item_id IS NULL`). Per-menu-item COGS attribution deferred.
- Phase 22.1 inserted after Phase 22 (URGENT — Phase 22 stays open until 22.1 verifies; both close together): HQ Toast ingest re-architecture. Verification of Phase 22 revealed Toast SFTP purges files faster than the 90-day backfill window assumes. Source of truth moves to DO Spaces. Phase 22.1 ships: (1) Spaces-first ingest worker (no SFTP in read path); (2) HQ-owned SFTP→Spaces sync worker + CLI (sales-processor decoupled, peer model); (3) one-time sales-processor archive → DO Spaces migration CLI; (4) local cache at `backend/cache/toast/YYYYMMDD/` (gitignored).

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

> **Status tags (new entries only — see `.planning/PLANNING-TEMPLATES.md` Block C):**
> Format: `- [Phase NN] [LOCKED|PROBATIONARY|FLUID] [YYYY-MM-DD]: <decision>`
> - **LOCKED** — frozen architecture; changing requires a superseding ADR-style row.
> - **PROBATIONARY** — recently changed; watch for regression. Auto-promotes to LOCKED after 2 phases without re-litigation.
> - **FLUID** — current best guess; cheap to revise.
> Untagged rows below predate this convention and are implicitly LOCKED unless contradicted.

- [v2.0 roadmap]: httpOnly, Secure, SameSite=Strict cookies — NOT localStorage — for session tokens (XSS risk; iOS standalone partition breaks localStorage anyway)
- [v2.0 roadmap]: Same-origin serving — Go binary embeds frontend via embed.FS, serves both `/api/v1/*` and static files from same host — eliminates CORS entirely
- [v2.0 roadmap]: IndexedDB + `online` event for offline queue — NOT Background Sync API (zero iOS Safari support)
- [v2.0 roadmap]: SW fetch handler must be partitioned before first API call — network-first for `/api/*`, cache-first for static (SW cache-first would corrupt API responses)
- [v2.0 roadmap]: DO Spaces presigned PUT URLs for photos — Go server generates URL, browser uploads directly; server never touches file bytes
- [v2.0 roadmap]: goose migrations — one logical change per numbered file, each in BEGIN/COMMIT — prevents dirty state on startup
- [Phase 09-foundation-auth]: sessions.expires_at is nullable (D-03) — sessions live indefinitely until explicit logout or admin revocation
- [Phase 09-foundation-auth]: 0004_hq_apps.sql schema only, no seed data (D-10) — db-seed Makefile target seeds 7 hq_apps rows separately
- [Phase 09-foundation-auth]: stdlib.OpenDBFromPool bridges pgxpool.Pool to *sql.DB for goose migration runner compatibility
- [Phase 09]: SW version bumped to v48 from v42 (plan assumed v47 as prior state but actual was v42; target v48 correct)
- [Phase 10-workflows-api]: JSONB for conditions/config/fail_trigger/template_snapshot — flexible schema without migrations per new field type
- [Phase 10-workflows-api]: SeedTemplates idempotent on template name — safe to run on every startup; insertField recursive for arbitrary sub-step depth
- [Phase 10-workflows-api]: errors.Is(err, os.ErrNotExist) for wrapped error detection from LoadTemplateConfig
- [Phase 10-workflows-api]: cleanupOldDrafts fired as fire-and-forget goroutine in MyChecklistsHandler to avoid blocking the response
- [Phase 10]: FIELD_RESPONSES replaces MOCK_RESPONSES as local optimistic state, backed by autoSaveField() POST saveResponse on every interaction
- [Phase 10]: api() wrapper pattern: async fetch with 401 redirect to login.html, 204 short-circuit, JSON error parse with status code
- [Phase 10-workflows-api]: submitChecklistToAPI() throws {offline:true} sentinel so caller can distinguish offline queuing from real errors
- [Phase 10-workflows-api]: IndexedDB hq_offline_v1 submitQueue: payload.id as keyPath = same UUID as idempotency_key, queuedAt added on enqueue
- [Phase 10-workflows-api]: _draining flag guards drainQueue() against concurrent invocations — window.addEventListener('online', drainQueue) auto-drains on reconnect
- [Phase 10-workflows-api]: Admin email in E2E tests corrected to jamal@yumyums.kitchen per superadmins.yaml
- [Phase 10.1-cross-device-state-sync]: CheckLWW uses device_id lexicographic tiebreaker when lamport_ts values are equal (D-10)
- [Phase 10.1-cross-device-state-sync]: OpsSince resolves access via template_assignments subquery so assignees receive ops from other devices (D-09)
- [Phase 10.1-cross-device-state-sync]: EmitOp is fire-and-forget with 5-second timeout; ErrConflict from EmitOp is logged not propagated
- [Phase 10.1-cross-device-state-sync]: pgconn import path is github.com/jackc/pgx/v5/pgconn (not standalone) for pgx v5 compatibility
- [Phase 10.1-cross-device-state-sync]: WebSocket hub uses channel-based concurrency (no mutex), single goroutine owns client map
- [Phase 10.1]: RejectItemHandler uses input.Comment (not body.Note) — actual field name in RejectItemInput struct
- [Phase 10.1-cross-device-state-sync]: /ws mounted at top-level router in its own auth group (not inside /api/v1) to avoid chi prefix collision
- [Phase 10.1-04]: flashField uses CSS background transition (info-bg) for 600ms to indicate incoming remote change
- [Phase 10.1-04]: drainQueue() in wsConnect.onopen called without db arg — function already calls getDB() internally
- [Phase 10.1-04]: LAMPORT_CLOCK guarded with null checks in wsConnect/wsCatchUp in case IndexedDB init fails
- [Phase 10.1-05]: showSyncToast separate from showToast — sync notifications use #sync-toast (themed, bottom:70px) to avoid collision with existing #toast action banner
- [Phase 10.1-05]: flashField uses CSS class animation with offsetWidth reflow trick — restart-safe, declarative, no inline style conflicts
- [Phase 10.2-reactive-sync-framework]: sync.js Store uses typeof guards for page globals (FIELD_RESPONSES, DRAFT_RESPONSES) — safe to load before page script initializes those globals
- [Phase 10.2-reactive-sync-framework]: submitOp routes to existing HTTP endpoints in Plan 01 — Plan 03 switches to POST /ops with optimistic apply and rollback per D-08
- [Phase 10.2-reactive-sync-framework]: debouncedSaveField uses _recentSaves (exposed on window) to suppress WS echo — LAMPORT_CLOCK device_id check alone insufficient when clock not yet initialized
- [Phase 10.2-reactive-sync-framework]: Kept explicit renderMyChecklists() after hydrateFieldState in loadMyChecklists — store subscriber fires before hydration, causing stale FIELD_RESPONSES
- [Phase 10.2-reactive-sync-framework]: OpHandler uses injected OpRouter to break circular sync<->workflow import; workflowOpRouter in main.go is the wiring point
- [Phase 10.2-reactive-sync-framework]: _recentSaves timing hack fully eliminated; self-echo via op.device_id === LAMPORT_CLOCK.deviceId
- [Phase 11-01]: displayNameExpr constant used in all SELECT queries — single source of truth for derived display_name from first_name/last_name/nickname
- [Phase 11-01]: ob_progress uses discriminator column (progress_type) rather than FK constraint to support both item and video_part progress in one table
- [Phase 11-onboarding-users-admin]: Nickname collision checks both u.nickname and derived display_name via COALESCE — prevents either form from being silently shadowed
- [Phase 11-onboarding-users-admin]: ClaimInviteToken uses atomic UPDATE RETURNING to prevent double-claim race without application-level locking
- [Phase 11-onboarding-users-admin]: isManagerOrAdmin helper used for sign-off and management endpoints — manager role can sign off per D-05
- [Phase 11-onboarding-users-admin]: GetHireTraining computes section state server-side: signed_off → complete → active → locked; isSectionComplete returns true for is_faq sections
- [Phase 11-onboarding-users-admin]: users.html event delegation via data-action attributes replaces old inline onclick handlers — consistent with workflows.html pattern
- [Phase 11-onboarding-users-admin]: login.html dual-mode: normal login when no token param, accept-invite set-password form when ?token= present
- [Phase 11-onboarding-users-admin]: localCopy pattern in Builder editor — deep-copy template before editing; Save calls PUT/POST API, Discard reverts without re-fetch
- [Phase 11-onboarding-users-admin]: SIGNOFF_FORM keyed by hireId_sectionId — supports concurrent sign-offs on multiple sections
- [Phase 11-onboarding-users-admin]: parseInt(uuid)||uuid removed from users.html click handler — UUIDs starting with hex digits that are valid decimal integers (e.g. '209c6b34') parse as integers via parseInt, causing editUser(209) to fail USERS.find() since IDs are UUID strings
- [Phase 12-03]: Cache-then-network pattern for tile permissions: apply cached from localStorage immediately, refresh from /me/apps in background
- [Phase 12-03]: tile.remove() used instead of display:none — grid reflows naturally with no gaps
- [Phase 12-inventory-photos-tile-permissions]: Go embed requires fixtures inside package dir — inventory fixtures at internal/inventory/fixtures/ not config/fixtures/
- [Phase 12-02]: Used s3.Options.BaseEndpoint instead of deprecated EndpointResolverWithOptions for DO Spaces custom endpoint (SDK v2 v1.99.1 removed old field)
- [Phase 12-02]: Photos presign route inside authenticated group — no unauthenticated upload URL generation
- [Phase 12-02]: workflows.html uses fetch() directly for /api/v1/photos/presign — api() helper prepends /api/v1/workflow/ which would create wrong path
- [Phase 12-inventory-photos-tile-permissions]: Receipt worker uses claude-haiku-4-5 via anthropic-sdk-go; Jaro-Winkler 0.85 threshold for purchase item fuzzy matching; worker gracefully skips on missing API keys; bank_tx_id idempotency check across purchase_events and pending_purchases
- [Phase 12]: renderHistoryList used for in-memory re-renders; loadHistory fetches fresh data — avoids double-fetch on tab switch
- [Phase 12]: Chart.js removed from inventory.html — Trends/Cost are coming-soon stubs, sales data deferred per D-13
- [Phase 12]: renderPendingCard separates pending from confirmed event rendering to avoid nested conditional logic in renderPurchaseEvent
- [Phase 12]: Fixed 0024_inventory.sql migration: is_active -> enabled (hq_apps column mismatch caused server startup failure)
- [Phase 13-01]: is_superadmin exposed in /me response (was on User struct but not serialized)
- [Phase 13-01]: Builder tab uses roles array check (CURRENT_USER.roles||[]).includes() instead of scalar CURRENT_USER.role ===
- [Phase 13-01]: DeleteTemplateHandler uses DELETE method and CASCADE via ob_templates FK; frontend upgraded from PUT /updateTemplate/{id}/delete
- [Phase 13-integration-fixes]: Hire detail view replaces auto-open-first-template in Manager tab for discoverability
- [Phase 13-integration-fixes]: Photo URL included in _fail_note bundle for autoSaveField and hydrateFieldState photo persistence
- [Phase 01-02]: uploadVideoFile uses XHR not fetch — only XHR exposes upload.progress events for progress bar
- [Phase 01-02]: Builder change event listener added for file inputs and radios (change not click) alongside existing click delegation
- [Phase 01-01]: video_watch_position is a separate progress_type from video_part — isSectionComplete only checks video_part so watch position tracking never falsely marks sections complete
- [Phase 01-01]: GREATEST() upsert for max_watched_time in ON CONFLICT clause ensures only forward-progress watch positions are stored
- [Phase 01-01]: VideoProcessHandler fires goroutine and returns 202 Accepted immediately — FFmpeg can take minutes for large videos
- [Phase 01-onboarding-video-upgrade]: cloneNode(true) resets video element listeners on each initVideoPlayer call
- [Phase 01-onboarding-video-upgrade]: renderTrainingDetail() dispatches to active runner (my or mgr) based on obState/mgrState view
- [Phase 16-05]: purchasing.html fully rewritten: 16-04 was planned but never executed; Plan 05 incorporates all 16-04 work (Order tab wired, PO tab, cutoff config) plus 16-05 shopping/history tabs in one pass
- [Phase 16-05]: Optimistic shop-check: toggle state immediately, render, then await API — roll back item.checked on error
- [Phase 16-cutoff-approval-and-shopping-list]: LOCKED_PO loaded from GET /orders?status=locked independently — not derived from PO_STATE status checks
- [Phase 16-cutoff-approval-and-shopping-list]: PO tab groups items by vendor_name on POLineItem (from LEFT JOIN in Plan 02), not by item group_name — per D-09
- [Phase 17]: package-level alertQueue var in purchasing package (not constructor injection) — consistent with scheduler.go pattern, minimal caller changes
- [Phase 17]: RecordRepurchase and NotifyVendorComplete called after COMMIT — badge data and alerts are best-effort, don't block the transaction
- [Phase 17]: alert_log UNIQUE (alert_type, week_start) provides idempotent cutoff reminder — INSERT ON CONFLICT DO NOTHING prevents duplicates
- [Phase 17]: Low-stock alert uses batch message per week rather than one alert per item to avoid notification spam
- [Phase 17]: notification_pref added to PATCH /users/{id} body (not separate endpoint) to keep edit form a single save action
- [Phase 17]: users.DefaultTimezone exported as public const for cross-package timezone fallback
- [Phase 17]: runLowStockCheck loads cutoff_config timezone instead of hardcoded America/Chicago, falls back to America/New_York
- Item rows show store_location label in Setup (category in composite header); backend item endpoints accept store_location
- Phase 19-01: Reused shop-toast for store_location guard; picker grouped by store_location with sticky headers; unassigned items blocked with hint text
- Photo items use COALESCE upsert for value column; sql.NullString for scanning; same incomplete-item detection as checkboxes
- Copied openCamera/showPhotoPreview from workflows.html for photo capture in onboarding

### Pending Todos

None yet.

### Blockers/Concerns

- Email provider must be chosen before Phase 9 planning — Resend vs Postmark vs net/smtp (affects invite flow in Phase 9)
- Onboarding schema not in docs/user-management-api.md — must be designed at Phase 11 planning by inspecting onboarding.html data structures
- DO Spaces bucket CORS policy for direct browser PUT uploads must be verified during Phase 12 planning
- Auth must be tested on a physical iPhone in standalone mode before Phase 9 is declared done — not in Safari or Chrome DevTools

### Quick Tasks Completed

| # | Description | Date | Commit | Status | Directory |
|---|-------------|------|--------|--------|-----------|
| 260415-axs | Convert backend/Makefile to backend/Taskfile.yml (go-task format) | 2026-04-15 | 691e616 |  | [260415-axs-convert-backend-makefile-to-backend-task](./quick/260415-axs-convert-backend-makefile-to-backend-task/) |
| 260417-x0g | Add user display name and logout button to index.html | 2026-04-17 | 5edc1e1 |  | [260417-x0g-add-user-display-name-and-logout-button-](./quick/260417-x0g-add-user-display-name-and-logout-button-/) |
| 260418-0tz | Multi-role support for users and training templates | 2026-04-18 | bf6a7c2 |  | [260418-0tz-multi-role-support-for-users-and-trainin](./quick/260418-0tz-multi-role-support-for-users-and-trainin/) |
| 260421-im4 | Rename History tab to Purchases in inventory.html | 2026-04-21 | f46a88c |  | [260421-im4-rename-history-tab-to-purchases-in-inven](./quick/260421-im4-rename-history-tab-to-purchases-in-inven/) |
| 260421-iug | Move Stock tab before Trends in inventory.html | 2026-04-21 | ab6806f |  | [260421-iug-move-the-stock-tab-before-trends-in-inve](./quick/260421-iug-move-the-stock-tab-before-trends-in-inve/) |
| 260422-not | Move stock level classification to shared Go backend function | 2026-04-22 | 6a974ac |  | [260422-not-shared-stock-level-classification-move-f](./quick/260422-not-shared-stock-level-classification-move-f/) |
| 260428-g36 | Remove 30-item cap from item picker renderPickerList | 2026-04-28 | 4cf4ea0 |  | [260428-g36-picker-item-cap](./quick/260428-g36-picker-item-cap/) |
| 260428-j2r | Add photo upload/change to Inventory Setup item edit form | 2026-04-28 | f7ed57c |  | [260428-j2r-photo-upload-on-inventory-setup](./quick/260428-j2r-photo-upload-on-inventory-setup/) |
| 260605-pk1 | Receipt-gate Mercury card transactions — surface no-attachment txns + accept empty-items confirm | 2026-06-05 | 4fe8dd2 |  | [260605-pk1-receipt-gate-mercury-card-transactions-m](./quick/260605-pk1-receipt-gate-mercury-card-transactions-m/) |
| 260605-q7b | Fix COGS undercount on confirm-without-receipt — seed placeholder catalog row + handler else-branch + backfill orphan events | 2026-06-05 | ef48640 |  | [260605-q7b-fix-cogs-undercount-on-confirm-without-r](./quick/260605-q7b-fix-cogs-undercount-on-confirm-without-r/) |
| 260605-u0i | Add by_vendor array to GET /api/v1/inventory/period-summary — completes sales-processor per-vendor COGS table contract | 2026-06-06 | 7579fe7 |  | [260605-u0i-add-by-vendor-array-to-get-api-v1-invent](./quick/260605-u0i-add-by-vendor-array-to-get-api-v1-invent/) |
| 260605-v0n | Filter COGS by Mercury category — cache mercury_category on purchase_events, filter /period-summary (main + by_vendor) by allowlist, ingest categoryData + worker re-sync, HQ_COGS_CATEGORY_ALLOWLIST env var | 2026-06-06 | b351804 |  | [260605-v0n-filter-cogs-by-mercury-category](./quick/260605-v0n-filter-cogs-by-mercury-category/) |
| 260606-0gh | Fix completeness gate to filter pending_purchases by COALESCE(event_date, created_at::Chicago) — late-ingested May receipts now block ready for the May period | 2026-06-06 | cf959bd | Verified | [260606-0gh-completeness-gate-filters-pending-review](./quick/260606-0gh-completeness-gate-filters-pending-review/) |
| 260606-9y0 | Expose tracked_bank_tx_ids on /period-summary — UNION of bank_tx_id across purchase_events + pending_purchases (all states) so sales-processor can diff Mercury and detect un-ingested transactions | 2026-06-06 | f730485 | Verified | [260606-9y0-tracked-bank-tx-ids-on-period](./quick/260606-9y0-tracked-bank-tx-ids-on-period/) |
| 260606-hew | Persist Mercury BankDescription as pending_purchases.vendor — fixes "Unknown Vendor" rendering on no_attachment_on_bank_tx rows + backfills existing in-window rows on next worker poll | 2026-06-06 | ea23933 | Verified | [260606-hew-vendor-fallback-bank-description](./quick/260606-hew-vendor-fallback-bank-description/) |
| 260606-hvy | Expose pending_review_details on /period-summary — parallel array next to pending_review_ids so sales-processor renders vendor + event_date + bank_total + reason instead of bare UUIDs | 2026-06-06 | d31b628 |  | [260606-hvy-expose-pending-review-details-on-period-](./quick/260606-hvy-expose-pending-review-details-on-period-/) |
| 260606-jvs | Narrow /period-summary completeness.ready to COGS-blocking pending receipts only (food category + no_attachment_on_bank_tx) + roll non-blocking food-category pending into cogs_excl_tax / by_vendor at ABS(bank_total); adds pending_purchases.mercury_category column + worker populate/refresh | 2026-06-06 | f3f9b2a |  | [260606-jvs-narrow-completeness-ready-on-period-summ](./quick/260606-jvs-narrow-completeness-ready-on-period-summ/) |
| 260607-bir | Sync Receipts to Mercury button on inventory Purchases tab — POST /api/v1/inventory/sync-receipts (single-flight via partial unique index + panic-safe goroutine wrapping runIngestCycle) + GET /sync-receipts/status; durable receipt_sync_runs row survives reload/PWA close-reopen; FE renders idle/running/done summary/failed states with visibility-aware 3s polling | 2026-06-07 | aebb84d |  | [260607-bir-mercury-receipt-sync-button](./quick/260607-bir-mercury-receipt-sync-button/) |
| 260607-co0 | Reprocess pending no-attachment rows when Mercury attachment is added on re-sync — classifyExistingTx 3-way (event/pending/none) replaces binary bankTxIDExists; pending-no-attachment + now-has-attachment upgrades via atomic DELETE+INSERT in createPurchaseEvent tx (auto-create path) or in-place updatePendingPurchase (parse/validate failure path); FE sync chip surfaces auto-added + cached counts so "0 pending review" with cached>0 stops being confusing; 5 new worker tests via parseReceipt/FetchTransactions seam | 2026-06-07 | 8360333 |  | [260607-co0-reprocess-pending-on-reupload](./quick/260607-co0-reprocess-pending-on-reupload/) |
| 260607-dg9 | Close two deferred 260607-co0 bugs: (1) guard items==nil before json.Marshal so pending_purchases.items is []' not 'null' in both insertPendingPurchase + updatePendingPurchase; (2) migration 0068 adds partial UNIQUE INDEX on pending_purchases(bank_tx_id) WHERE confirmed_at IS NULL AND discarded_at IS NULL after deduping existing active rows; ON CONFLICT clause updated to target the partial index so re-polls during the 14-day lookback no longer create duplicate pending rows; 2 new worker tests + flips 2 previously-failing tests to passing | 2026-06-07 | e5353d5 |  | [260607-dg9-pending-purchases-unique-index-and-items](./quick/260607-dg9-pending-purchases-unique-index-and-items/) |
| 260607-e1c | Receipt pipeline reliability: (1) migration 0069 adds pending_purchases.parse_error column; worker persists actual ParseReceipt error text so users see "Parser error: …" inline on the pending review card instead of generic "Receipt could not be parsed automatically"; (2) Sonnet 4.6 fallback when Haiku ParseReceipt fails — Haiku attempts first (cheap, fast), Sonnet retries on error (more capable for complex table-layout PDFs like Restaurant Depot's 19-item Prawn-generated invoice); only routes to pending review when BOTH fail; parse_error then contains "haiku: …; sonnet: …"; (3) inventory.html view-receipt overlay renders PDFs via iframe instead of img, with "Open in new tab" link as iOS PWA fallback — fixes the "Receipt image unavailable" error every PDF receipt was hitting; 3 new backend tests + 2 new Playwright tests | 2026-06-07 | 039b485 |  | [260607-e1c-persist-parse-error-sonnet-fallback-pdf-](./quick/260607-e1c-persist-parse-error-sonnet-fallback-pdf-/) |
| 260607-fxl | Close two gaps from 260607-e1c: (A) classifyExistingTx + runIngestCycle now retry rows where Haiku failed in a prior sync — gated by reason='Receipt could not be parsed automatically' AND parse_error IS NULL AND items empty AND attachments present so each row gets exactly one Sonnet attempt and user edits are never clobbered; rows stuck since before e1c deployed auto-resolve via Sonnet on next sync. (B) ConfirmPendingPurchaseHandler now rejects empty-items confirm with 422 envelope `{"error":"empty_items_not_allowed","reason":"…"}` unless pending reason='no_attachment_on_bank_tx' (the 260605-pk1 no-receipt swipe path stays accepted); existing total-mismatch rejection upgraded from text-400 to 422 envelope `{"error":"total_mismatch","line_total":X,"bank_total":Y}`; FE inventory.html computes canConfirm from line items + match check and renders Confirm Receipt with disabled attribute when ineligible (re-renders live on input change); 3 new worker tests + 3 new handler tests + 3 new Playwright tests | 2026-06-07 | e0b0610 |  | [260607-fxl-retry-parse-failed-rows-and-confirm-gate](./quick/260607-fxl-retry-parse-failed-rows-and-confirm-gate/) |
| 260607-fxl-hotfix | Production-spam fix: classifyExistingTx SQL used COALESCE(jsonb_array_length(items), 0) but jsonb_array_length raises SQLSTATE 22023 on the JSON literal 'null' (a scalar, not SQL NULL) — pre-260607-dg9 rows have items='null'::jsonb so the worker was erroring on every Mercury tx after 260607-fxl deploy. Guard with jsonb_typeof(items)='array' first; one-line SQL fix + regression test seeding items='null'::jsonb to lock in the read-side tolerance | 2026-06-07 | c171c92 |  | (inline hotfix, no .planning/quick directory) |
| 260607-k1n | Receipt parser test strategy (3-layer): (L1) ReceiptItem.Quantity int→float64 so LLM responses with float-formatted integers (e.g. quantity:40.0) unmarshal cleanly; persistence boundary in worker.go's createPurchaseEvent rounds via int(math.Round(li.Quantity)) so DB column stays INTEGER; validate.go totalQty accumulator switched to float64 with the same rounding for Check 3; (L2) backend/internal/receipt/testdata/llm_responses/ corpus of 5 captured-shape .txt fixtures + .expected.json or .expected-err.txt sibling pairs, walker test exercises parseJSONBody against each so future shape regressions surface immediately (deterministic, fast, free); (L3) worker scenario-table TestRunIngestCycle_ScenarioTable with 5 sub-cases (happy/haiku→sonnet/both-fail-realistic/both-fail-decimal-qty/total-mismatch) using existing workerStubs helper, exact production error string from today's decimal-qty incident locked in as a fixture | 2026-06-07 | 824a224 |  | [260607-k1n-receipt-parser-test-strategy](./quick/260607-k1n-receipt-parser-test-strategy/) |
| 260607-koi | Retry parse button on pending review cards — POST /api/v1/inventory/purchases/pending/{id}/retry-parse clears parse_error so the 260607-fxl upgrade gate matches again (reason=parse-failed AND parse_error IS NULL AND items empty AND attachments present); 404 on unknown id, 422 envelope `row_not_pending` on confirmed/discarded, 422 `nothing_to_retry` on already-null parse_error. FE renders a "Retry parse" button below the inline parse_error chip when present; click POSTs, on 200 shows toast "Marked for retry. Click Sync Receipts to run now." and re-renders the card with the error cleared. Re-arms stuck rows after parser fixes without DB access; 4 new handler tests + 3 new Playwright tests | 2026-06-07 | 2d16ed0 |  | [260607-koi-retry-parse-button](./quick/260607-koi-retry-parse-button/) |
| 260607-l9m | 260607-k1n follow-up — widen ReceiptSummary.TotalUnits + TotalCases int→float64 (k1n only widened ReceiptItem.Quantity; today's production log showed "cannot unmarshal number 85.56 into Go struct field ReceiptSummary.summary.total_units of type int"). validate.go Check 3 now rounds at the int(math.Round(summary.TotalUnits + summary.TotalCases)) comparison boundary matching the k1n totalQty pattern. New fixture pair testdata/llm_responses/02b_float_summary_units.{txt,expected.json} + TestParseJSONBody_FloatSummary regression test exercising the exact 2026-06-07 failing payload. The 6-fixture walker now covers item-level AND summary-level float-formatted decimals | 2026-06-07 | 479824c |  | [260607-l9m-summary-float-fields](./quick/260607-l9m-summary-float-fields/) |
| 260607-s6r | 260607-koi follow-up — broaden Retry parse button gate so it also fires when items are populated but line_total doesn't match bank_total (koi only rendered the button on parse_error rows). Recovery UX for today's stuck pending row: Restaurant Depot receipt content ($1515.16) paired with the wrong bank tx (Amazon Mktplace Pmts, $65.62) — parse succeeded so parse_error was NULL and koi's button never appeared. Handler now also accepts items-mismatch rows; UPDATE sets items='[]'::jsonb (satisfies worker.go classifyExistingTx jsonb_typeof+length predicate via failed second conjunct) AND reason='Receipt could not be parsed automatically' so worker's parseFailedRetry gate matches on next sync. FE click in items-mismatch path triggers confirm("Discard parsed items and re-parse from receipt?") before POST; parse_error-only path stays zero-friction (koi behavior preserved). Reuses the existing 0.01 epsilon — no new tolerance constant. 2 new handler tests (accepted-then-cleared, items-match-still-rejected). Wrong-attachment matching root-cause bug deferred to next /gsd:debug session | 2026-06-07 | 7767f40 |  | [260607-s6r-broaden-retry-parse-button-gate-to-also-](./quick/260607-s6r-broaden-retry-parse-button-gate-to-also-/) |
| 260612-uxt | Apply Definition of Done conventions — insert ### Definition of Done block into CLAUDE.md, create .planning/PLANNING-TEMPLATES.md (Block A/B/C), insert LOCKED/PROBATIONARY/FLUID status tags legend into .planning/STATE.md; delete APPLYdefinitionofdone.md staging file | 2026-06-12 | 62774d5 |  | [260612-uxt-apply-definition-of-done-conventions-fro](./quick/260612-uxt-apply-definition-of-done-conventions-fro/) |
| 260627-i3z | Scaffold ui-jury config for hq — routes.yaml (7 routes, 393×852 viewport, login setup with `<FILL-IN-BEFORE-RUNNING>` password placeholder), .ui-jury/hooks.yaml (db_reset only), scripts/ui-jury/db-reset.sh (wraps task backend:db-reset-inventory, preserves users/templates), docs/ui-jury.md. Schema-validated with ajv-cli against routes.schema.json + hooks.schema.json; db-reset.sh smoke-tested against live local Postgres (exit 0, TRUNCATE TABLE confirmed). login.html included with documented caveat that per-route setup is additive (not override) per schema — captures post-login redirect state in v1 | 2026-06-27 | d7cb670 |  | [260627-i3z-scaffold-ui-jury-config](./quick/260627-i3z-scaffold-ui-jury-config/) |
| 260630-mav | Inventory Purchases tab — red "Missing Receipt" badge for no-attachment pending purchases. Adds `.approval-badge.missing` CSS modifier (red palette, light + dark mode); `renderPendingCard()` branches on `p.reason==='no_attachment_on_bank_tx'` to render "Missing Receipt" instead of "Needs Review" and suppress the raw sentinel string from the meta row. Other reasons (parse failed, totals mismatch) keep the existing amber "Needs Review" badge and reason text. Local var renamed to `pendingBadgeHtml` to avoid shadowing the global `badgeHtml(level)` stock helper | 2026-06-30 | (commit pending) |  | [260630-mav-missing-receipt-badge](./quick/260630-mav-missing-receipt-badge/) |

## Deferred Items

Items acknowledged and deferred at v3.1 milestone close on 2026-06-05:

| Category | Item | Status | Note |
|----------|------|--------|------|
| uat_gap | Phase 21 21-HUMAN-UAT.md | partial — 4 pending | v3.1 scope, human-only acceptance |
| uat_gap | Phase 999.2 999.2-HUMAN-UAT.md | partial — 5 pending | v3.1 scope, deferred to manual ack |
| uat_gap | Phase 09 09-UAT.md | testing — 9 pending | v2.0 carry-over (Tailscale/iPhone UAT) |
| verification_gap | Phase 21 21-VERIFICATION.md | human_needed | v3.1 scope |
| verification_gap | Phase 999.2 999.2-VERIFICATION.md | human_needed | v3.1 scope |
| verification_gap | Phase 01 01-VERIFICATION.md | human_needed | v1.0 carry-over |
| verification_gap | Phase 02 02-VERIFICATION.md | human_needed | v1.0 carry-over |
| verification_gap | Phase 14 14-VERIFICATION.md | human_needed | v3.0 carry-over |
| quick_task | 11 quick-task slugs dated 260412–260428 | missing | pre-v3.0 era, never acted on |

Total: 21 items. Pre-v3.0 carry-over items will roll forward and should be reviewed at next milestone audit.

## Session Continuity

Last session: 2026-06-06T00:27:00Z
Stopped at: Completed quick task 260606-0gh: completeness gate filters by event_date
Resume file: --resume-file

**Planned Phase:** 999.2 (Per-menu-item COGS attribution via recipe/BOM mapping) — 6 plans — 2026-06-04T13:45:31.758Z
