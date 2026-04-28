---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: executing
stopped_at: Completed 19-02-PLAN.md
last_updated: "2026-04-28T15:21:40.501Z"
last_activity: 2026-04-23
progress:
  total_phases: 2
  completed_phases: 0
  total_plans: 3
  completed_plans: 1
  percent: 33
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-15)

**Core value:** Operational tools that let the owner manage crew workflows and training from one mobile app — with accountability and smart conditions.
**Current focus:** Phase 17 — alerts-notifications-and-repurchase-badges

## Current Position

Phase: 999.1
Plan: Not started
Status: Ready to execute
Last activity: 2026-04-23

Progress: [░░░░░░░░░░] 0% (v2.0 milestone)

## Performance Metrics

**Velocity:**

- Total plans completed: 18 (v1.0 + v1.1)
- Average duration: ~12 min
- Total execution time: ~3.6 hours

**By Phase (v2.0):**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 09-foundation-auth | TBD | - | - |
| 10-workflows-api | TBD | - | - |
| 11-onboarding-users-admin | TBD | - | - |
| 12-inventory-photos | TBD | - | - |

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

## Accumulated Context

### Roadmap Evolution

- Phase 10.1 inserted after Phase 10: Cross-Device State Sync (URGENT)
- Phase 10.2 inserted after Phase 10.1: Reactive Sync Framework (URGENT) — shared Store + single write channel before Phase 11
- v2.1 milestone started: Onboarding Video Upgrade — Phase 1 added

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

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

### Pending Todos

None yet.

### Blockers/Concerns

- Email provider must be chosen before Phase 9 planning — Resend vs Postmark vs net/smtp (affects invite flow in Phase 9)
- Onboarding schema not in docs/user-management-api.md — must be designed at Phase 11 planning by inspecting onboarding.html data structures
- DO Spaces bucket CORS policy for direct browser PUT uploads must be verified during Phase 12 planning
- Auth must be tested on a physical iPhone in standalone mode before Phase 9 is declared done — not in Safari or Chrome DevTools

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260415-axs | Convert backend/Makefile to backend/Taskfile.yml (go-task format) | 2026-04-15 | 691e616 | [260415-axs-convert-backend-makefile-to-backend-task](./quick/260415-axs-convert-backend-makefile-to-backend-task/) |
| 260417-x0g | Add user display name and logout button to index.html | 2026-04-17 | 5edc1e1 | [260417-x0g-add-user-display-name-and-logout-button-](./quick/260417-x0g-add-user-display-name-and-logout-button-/) |
| 260418-0tz | Multi-role support for users and training templates | 2026-04-18 | bf6a7c2 | [260418-0tz-multi-role-support-for-users-and-trainin](./quick/260418-0tz-multi-role-support-for-users-and-trainin/) |
| 260421-im4 | Rename History tab to Purchases in inventory.html | 2026-04-21 | f46a88c | [260421-im4-rename-history-tab-to-purchases-in-inven](./quick/260421-im4-rename-history-tab-to-purchases-in-inven/) |
| 260421-iug | Move Stock tab before Trends in inventory.html | 2026-04-21 | ab6806f | [260421-iug-move-the-stock-tab-before-trends-in-inve](./quick/260421-iug-move-the-stock-tab-before-trends-in-inve/) |
| 260422-not | Move stock level classification to shared Go backend function | 2026-04-22 | 6a974ac | [260422-not-shared-stock-level-classification-move-f](./quick/260422-not-shared-stock-level-classification-move-f/) |

## Session Continuity

Last session: 2026-04-28T15:21:33.513Z
Stopped at: Completed 19-02-PLAN.md
Resume file: None
