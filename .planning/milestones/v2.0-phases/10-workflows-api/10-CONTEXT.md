# Phase 10: Workflows API - Context

**Gathered:** 2026-04-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Persist checklist templates, submissions, and approval flow in Postgres. Replace all in-memory mock data in workflows.html (MOCK_TEMPLATES, MOCK_RESPONSES, FAIL_NOTES, REJECTION_FLAGS, PENDING_APPROVALS) with real API calls. Add offline sync with IndexedDB queue and pending-sync indicator.

Photos are deferred to Phase 12 — photo fields show "Coming soon" placeholder.

</domain>

<decisions>
## Implementation Decisions

### Schema Design
- **D-01:** JSONB columns for conditions (day-of-week, skip logic, fail triggers) and field config (temp min/max) on sections and fields tables. Flexible for new condition types later.
- **D-02:** Normalized submission + field_responses tables. One row per answered field enables per-field queries, user attribution, and item-level approval.
- **D-03:** Self-referencing fields table for sub-steps: nullable `parent_field_id UUID REFERENCES fields(id)`. Sub-steps are just fields whose parent is another checkbox field.
- **D-04:** Separate schedules table for template scheduling (not array column on template). Allows future per-user scheduling, one-off overrides, complex recurrence.
- **D-05:** Junction table `template_assignments` for assignees and approvers: `assignee_type` ('role'|'user'), `assignee_id`, `assignment_role` ('assignee'|'approver').
- **D-06:** Approval/rejection is per-field, matching the current mock UI. Rejection flags (comment, requirePhoto) live on field_responses. Submission status derived from field states.
- **D-07:** Soft delete for templates via `archived_at TIMESTAMP`. Archived templates stop appearing in My Checklists but historical submissions remain. Can be unarchived.

### Builder Save Flow
- **D-08:** Explicit Save button in the Builder tab. User builds/edits, then taps Save to persist. Discard button to abandon changes.
- **D-09:** Full replace on save — PUT entire template JSON (sections + fields). Server deletes old rows, inserts new ones in a transaction. No diff tracking.
- **D-10:** Snapshot on submit — when crew submits a checklist, freeze the template as JSONB in the submission row. Template edits don't affect in-progress or completed submissions.
- **D-11:** Builder access restricted to admin + superadmin only for now. Design the permission check so it's easy to swap in granular permissions later (from Users app).

### Offline Sync Strategy
- **D-12:** Submit queue only — if connection drops mid-checklist, submission is queued in IndexedDB. Syncs on `online` event. Browsing/starting checklists requires being online.
- **D-13:** Banner + per-item badge for pending-sync indicator. Persistent banner at top ("N submissions pending sync"), badge on each queued checklist. Clears automatically when synced.
- **D-14:** Reject and notify on conflict — if template was archived while offline, server returns 409 Conflict. Client shows error with explanation. Crew's offline work is not silently accepted.
- **D-15:** Client-generated UUID idempotency keys. Each submission gets a UUID at submit time, stored in IndexedDB queue. Server uses it as dedup key (UNIQUE constraint). Handles retries and double-taps.
- **D-16:** Race condition tests required — ensure concurrent sync operations, double-submits, and offline→online transitions are handled gracefully.

### API Migration Approach
- **D-17:** Big-bang swap — replace all mock data (MOCK_TEMPLATES, MOCK_RESPONSES, etc.) with fetch calls in one pass. No feature flags, no two code paths.
- **D-18:** Full RPC for writes, GET for reads. Write endpoints: POST /api/v1/workflow/createTemplate, submitChecklist, approveItem, rejectItem, etc. Read endpoints: GET /api/v1/workflow/templates, myChecklists, pendingApprovals, etc.
- **D-19:** Rewrite existing 54 Playwright E2E tests against real Go server with test database. Seed test data via API/SQL before each run. Mock data tests deleted.
- **D-20:** Empty start — no pre-seeded templates. App launches with empty state ("Go to Builder to create your first template"). Support YAML-based template seeding (like superadmins pattern) for dev/testing.

### My Checklists Tab
- **D-21:** Auto-save responses on each field tap/entry. Progress persists if crew leaves and comes back. Submit is explicit and separate from field saves.
- **D-22:** Today only + history link. My Checklists shows today's scheduled checklists by default. "View History" link at bottom for past submissions.

### Approvals Tab
- **D-23:** Pending approvals grouped by template. All Setup Checklist submissions together, then all Closing Checklist submissions. Within each group, sorted by time.

### Error Handling & Loading States
- **D-24:** Skeleton screens for loading states. Placeholder shapes matching content layout, pulsing gray. No spinners.
- **D-25:** Inline error with retry button on API failures. No modals. Per-field error indicator on failed auto-saves (tap to retry).

### Photos
- **D-26:** Photos deferred to Phase 12. Photo fields render but camera/upload shows "Coming soon" placeholder. Text notes and severity on corrective actions work normally.

### Claude's Discretion
- IndexedDB schema design (table names, key structure)
- Debounce timing for auto-save field responses
- Error retry backoff strategy
- Skeleton screen CSS implementation details

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing Frontend
- `workflows.html` — Current 1500-line implementation with all mock data structures (MOCK_TEMPLATES, MOCK_RESPONSES, FAIL_NOTES, REJECTION_FLAGS, PENDING_APPROVALS, APPROVED_SUBMISSIONS)

### Backend Foundation (Phase 9)
- `backend/cmd/server/main.go` — Server startup sequence, chi router, auth middleware wiring
- `backend/internal/auth/service.go` — Auth patterns (GenerateToken, session CRUD, UserFromContext)
- `backend/internal/auth/handler.go` — Handler patterns (LoginHandler, cookie management)
- `backend/internal/auth/middleware.go` — Auth middleware pattern for protected routes
- `backend/internal/db/db.go` — Database pool setup, goose migration pattern, SeedHQApps pattern
- `backend/internal/db/migrations/` — Existing migration files (0001-0005) for schema patterns
- `backend/internal/config/config.go` — YAML config loading pattern (SuperadminEntry)
- `backend/config/superadmins.yaml` — YAML config pattern for template seeding reference

### API Design
- `docs/user-management-api.md` — REST API contracts and table schema reference

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `auth.Middleware` — Chi middleware for protecting routes, provides `UserFromContext(ctx)`
- `db.NewPool` / `db.Migrate` / `db.SeedHQApps` — Established patterns for DB operations
- `config.LoadSuperadmins` — YAML config loading pattern, reusable for template seeding
- SortableJS 1.15.7 — Already loaded via CDN in workflows.html for drag-to-reorder

### Established Patterns
- State-first rendering: mutate JS state → call render function → DOM updates from state
- Event delegation: ONE click + ONE input listener per container div, routes via `data-action`
- Chi router with middleware groups for auth-protected routes
- Goose embedded SQL migrations with sequential numbering
- httpOnly session cookies (Secure in prod, not in dev)

### Integration Points
- `r.Route("/api/v1", ...)` in main.go — new workflow routes added here
- SW fetch partition — `/api/*` already network-first, static cache-first
- `sw.js` ASSETS array — no changes needed (API routes aren't cached)
- Index.html auth guard — already wired via checkAuth()

</code_context>

<specifics>
## Specific Ideas

- Template seeding from YAML file (like superadmins.yaml) for dev/test environments
- Approval flow must exactly mimic current mock data behavior (per-field rejection with comments, optional photo requirements, correction loop)
- Schedule table designed for future expansion (per-user assignments, time-of-day, complex recurrence)

</specifics>

<deferred>
## Deferred Ideas

- Photo storage and camera integration — Phase 12
- Granular per-template Builder permissions — Phase 11 (Users app)
- Complex scheduling (per-user, time-of-day, recurrence) — future milestone
- Template versioning with diff/rollback — future milestone if needed

</deferred>

---

*Phase: 10-workflows-api*
*Context gathered: 2026-04-15*
