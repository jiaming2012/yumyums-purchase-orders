# Phase 11: Onboarding + Users Admin - Context

**Gathered:** 2026-04-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Persist onboarding training progress and sign-offs in Postgres via a new `ob_*` table set. Wire users.html to the real admin API (invite, roles, permissions, session revocation). Replace all mock data in both onboarding.html and users.html with API calls. Add invite acceptance flow to login.html. No new UI features beyond what's needed for API wiring and the invite/reset/revocation flows.

Requirements: ONBD-01, ONBD-02, ONBD-03, ONBD-04, USER-01, USER-02, USER-03

</domain>

<decisions>
## Implementation Decisions

### Onboarding Schema + API
- **D-01:** Separate `ob_*` tables (ob_templates, ob_sections, ob_items, ob_progress, ob_signoffs) — NOT reusing checklist_* tables. Onboarding has video parts, FAQ Q&A, and sequential sections that don't map to checklist fields.
- **D-02:** Auto-save per item — each checkbox tap or video-watched mark auto-saves to the server immediately. Progress persists if crew closes the app. Consistent with workflows pattern (Phase 10 D-21).
- **D-03:** RPC-style API endpoints (POST /api/v1/onboarding/saveProgress, /signOff, etc.) to match workflows Phase 10 write pattern.
- **D-04:** Builder save uses explicit Save button + full PUT replace — same as workflows builder (Phase 10 D-08/D-09). Discard button to abandon changes.

### Sign-off Journal
- **D-05:** Sign-off form requires notes (free text) AND rating (Ready / Needs Practice / Struggling) before confirming. Manager must fill both.
- **D-06:** Sign-off display shows only name + timestamp inline on the hire's checklist view. Notes and rating are stored in the backend but not shown in the hire's UX — available for future manager reporting.
- **D-07:** Sign-off records stored in `ob_signoffs` table with: manager_id, section_id, hire_id, timestamp, notes, rating.

### User Naming Model
- **D-08:** Users table stores `first_name`, `last_name`, `nickname` (optional). Replaces the single `display_name` column from the original API doc.
- **D-09:** display_name is a derived function: `nickname` if set, otherwise `"{first_name} {last_initial}."` (e.g., "Jamal C.").
- **D-10:** Nickname uniqueness constraint: nickname must not collide with any other user's nickname OR any other user's derived display_name. Enforced server-side.
- **D-11:** Nickname collision surfaced as error on save — server returns 409 with message "'X' is already taken by Y." Inline error under nickname field.

### Invite Acceptance Flow
- **D-12:** Admin creates user in users.html: first name, last name, email, role. User appears immediately in user list with "Invited" badge.
- **D-13:** Invited users can be assigned to checklists/approvals BEFORE accepting the invite. They can't log in until they accept.
- **D-14:** Opaque token for Phase 11 — no Resend email integration yet. Server generates invite link, admin copies and shares manually.
- **D-15:** After creating user, UI shows invite link + copy-to-clipboard button + "An email has been sent" text (future-proofing for when Resend is wired up).
- **D-16:** login.html detects `?token=` in URL → switches to accept-invite mode: shows "Welcome, {name}" heading, password + confirm password fields, "Set Password & Log In" button. Hides email field.
- **D-17:** On successful accept-invite: user status changes from 'invited' to 'active', "Invited" badge disappears, user redirected to home page (index.html).
- **D-18:** Expired/used token shows inline error on login.html: "This invite link has expired. Ask your manager to send a new one."

### Password Reset Flow
- **D-19:** Password reset reuses the invite token flow entirely. Admin taps "Reset Password" → server generates new token → admin sees link + copy button (same UI as invite). Crew clicks link → same set-password screen on login.html.

### Session Revocation UI
- **D-20:** Force Logout button in the Edit user form, grouped with Reset Password and Delete User actions (below a separator line).
- **D-21:** Confirmation dialog required: "Force logout {name}? They will need to log in again." [Cancel] [Force Logout].
- **D-22:** Lazy 401 detection — no push notification. On crew member's next API call, request returns 401, client-side auth check redirects to login.html.

### Users.html Migration
- **D-23:** Big-bang mock→API swap — replace USERS array, APPS, DEFAULT_PERMS, USER_GRANTS with API calls in one pass. No feature flags. Consistent with workflows Phase 10 D-17.
- **D-24:** Editable after creation: role, nickname, first name, last name. Email is read-only (set at invite time, never changes).
- **D-25:** Edit form layout: role dropdown, first name, last name, nickname fields, then separator, then action buttons (Reset Password, Force Logout, Delete User).

### Onboarding Migration
- **D-26:** Big-bang mock→API swap for onboarding.html — replace MOCK_OB_TEMPLATES and SECTION_STATES with API calls. Delete all mock data.
- **D-27:** Existing 35 onboarding E2E tests rewritten against real Go server with test database. Mock data tests deleted. Consistent with workflows test migration (Phase 10 D-19).

### Claude's Discretion
- ob_* table schema details (exact columns, indexes, constraints)
- API endpoint naming within the RPC pattern
- Onboarding API response shapes
- Error handling and loading states (follow Phase 10 patterns: skeleton screens D-24, inline error with retry D-25)
- Sign-off form UI layout and styling
- Invite link URL structure and token format
- Test seed data for onboarding templates

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### API Design
- `docs/user-management-api.md` — Full SQL schema for users, sessions, invite_tokens, hq_apps, app_permissions tables. REST API contracts for auth, user CRUD, app permissions, invite acceptance flow. **NOTE:** `display_name` column is being replaced with `first_name`, `last_name`, `nickname` per D-08.

### Backend Foundation (Phase 9)
- `backend/internal/auth/service.go` — Auth patterns (GenerateToken, session CRUD, UserFromContext)
- `backend/internal/auth/handler.go` — Handler patterns (LoginHandler, cookie management)
- `backend/internal/auth/middleware.go` — Auth middleware pattern for protected routes
- `backend/internal/db/db.go` — Database pool setup, goose migration pattern, SeedHQApps pattern
- `backend/internal/db/migrations/` — Existing migrations 0001-0016 for schema patterns
- `backend/internal/config/config.go` — YAML config loading pattern (SuperadminEntry)

### Workflows API (Phase 10) — Pattern Reference
- `backend/internal/workflow/` — RPC-style handler patterns, auto-save per field, full PUT template replace
- `workflows.html` — Big-bang mock→API swap reference implementation

### Existing Frontend
- `onboarding.html` — Current 1121-line implementation with MOCK_OB_TEMPLATES, SECTION_STATES, Builder tab, My Trainings + Manager views
- `users.html` — Current 303-line implementation with USERS, APPS, DEFAULT_PERMS, USER_GRANTS mock arrays
- `login.html` — Login page to be extended with accept-invite mode (token detection, set-password form)

### Prior Phase Context
- `.planning/phases/04-onboarding-app/04-CONTEXT.md` — Onboarding data model, sequential sections, video training, FAQ gate, sign-off flow (D-01 through D-16)
- `.planning/phases/05-onboarding-builder/05-CONTEXT.md` — Builder tab, template CRUD, SortableJS, section toggles (D-01 through D-19)
- `.planning/phases/09-foundation-auth/09-CONTEXT.md` — Auth foundation, Resend decision (D-01), session revocation API (D-04), invite flow
- `.planning/phases/10-workflows-api/10-CONTEXT.md` — Workflows API patterns, big-bang swap (D-17), auto-save (D-21), skeleton screens (D-24), inline errors (D-25)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `backend/internal/auth/` — Session management, middleware, token generation — reuse for invite tokens and password reset tokens
- `backend/internal/workflow/` — RPC handler pattern, auto-save pattern, full PUT template replace — mirror for onboarding API
- `backend/internal/db/` — Pool setup, goose migrations, seed patterns — extend with ob_* migrations
- `onboarding.html` tabs, CSS variables, event delegation — wire to API instead of mock data
- `users.html` edit form, permission management — wire to API instead of mock data

### Established Patterns
- State-first rendering: mutate JS state → call render function → DOM updates from state
- Event delegation with `data-action` attributes
- Chi router with middleware groups for auth-protected routes
- Goose embedded SQL migrations with sequential numbering (currently at 0016)
- httpOnly session cookies (Secure in prod, not in dev)
- RPC-style write endpoints (POST /api/v1/workflow/createTemplate, etc.)

### Integration Points
- `backend/cmd/server/main.go` — Add onboarding routes and user admin routes to chi router
- `login.html` — Extend with token detection and set-password mode
- `users.html` — Replace mock arrays, add invite creation form, add first/last/nickname fields, add force logout button
- `onboarding.html` — Replace MOCK_OB_TEMPLATES and SECTION_STATES with API fetch calls
- `sw.js` — No changes needed (API routes already network-first)

</code_context>

<specifics>
## Specific Ideas

- Invited users are "real" in the system — they can be assigned to checklists, appear in user lists, have permissions set — they just can't log in until accepting the invite
- display_name is ALWAYS derived, never stored directly: `nickname || "{first_name} {last_initial}."`
- Password reset and invite acceptance share the exact same token + login.html set-password UX — no duplicate code paths
- Sign-off captures rich data (notes + rating) from the start, even though the crew-facing UX only shows name + timestamp — the data is there for future manager reporting
- "An email has been sent" message shown even though Phase 11 uses opaque tokens — future-proofing for Resend integration

</specifics>

<deferred>
## Deferred Ideas

- Resend email integration for real invite emails — future phase when DNS/email setup is done
- Self-service "Forgot password?" on login.html — future phase (currently admin-triggered only)
- Manager reporting dashboard for sign-off ratings — future phase (data captured in Phase 11)
- Granular per-template builder permissions — noted in Phase 10 deferred, Phase 11 adds basic role-based access

</deferred>

---

*Phase: 11-onboarding-users-admin*
*Context gathered: 2026-04-17*
