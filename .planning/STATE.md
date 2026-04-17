---
gsd_state_version: 1.0
milestone: v2.0
milestone_name: Backend
status: executing
stopped_at: Completed 10.2-02-PLAN.md
last_updated: "2026-04-17T15:03:33.890Z"
last_activity: 2026-04-17
progress:
  total_phases: 6
  completed_phases: 3
  total_plans: 17
  completed_plans: 16
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-15)

**Core value:** Operational tools that let the owner manage crew workflows and training from one mobile app — with accountability and smart conditions.
**Current focus:** Phase 10.2 — reactive-sync-framework

## Current Position

Phase: 10.2 (reactive-sync-framework) — EXECUTING
Plan: 3 of 3
Status: Ready to execute
Last activity: 2026-04-17

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

## Accumulated Context

### Roadmap Evolution

- Phase 10.1 inserted after Phase 10: Cross-Device State Sync (URGENT)
- Phase 10.2 inserted after Phase 10.1: Reactive Sync Framework (URGENT) — shared Store + single write channel before Phase 11

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

## Session Continuity

Last session: 2026-04-17T15:03:33.883Z
Stopped at: Completed 10.2-02-PLAN.md
Resume file: None
