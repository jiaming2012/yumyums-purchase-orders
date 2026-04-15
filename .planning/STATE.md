---
gsd_state_version: 1.0
milestone: v2.0
milestone_name: Backend
status: executing
stopped_at: Completed 10-workflows-api/10-04-PLAN.md
last_updated: "2026-04-15T16:26:37.946Z"
last_activity: 2026-04-15
progress:
  total_phases: 4
  completed_phases: 1
  total_plans: 9
  completed_plans: 8
  percent: 0
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-04-15)

**Core value:** Operational tools that let the owner manage crew workflows and training from one mobile app — with accountability and smart conditions.
**Current focus:** Phase 10 — workflows-api

## Current Position

Phase: 10 (workflows-api) — EXECUTING
Plan: 5 of 5
Status: Ready to execute
Last activity: 2026-04-15

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

## Accumulated Context

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

Last session: 2026-04-15T16:26:37.939Z
Stopped at: Completed 10-workflows-api/10-04-PLAN.md
Resume file: None
