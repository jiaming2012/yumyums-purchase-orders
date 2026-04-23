# Phase 9: Foundation + Auth - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Go backend shell serving the existing PWA from a single origin, Postgres database with migrations, Tailscale HTTPS for phone testing, service worker fetch handler partitioned for API calls, and working login/logout with real credentials. No new UI features — this phase wires the existing login.html to a real auth API and ensures all protected pages redirect unauthenticated users.

</domain>

<decisions>
## Implementation Decisions

### Email Provider
- **D-01:** Use Resend for transactional email (invite flow, password reset). Generous free tier (100 emails/day), Go SDK, simple DNS setup. Fits low-volume food truck scale.

### Login Flow
- **D-02:** After successful login, always redirect to index.html (HQ launcher grid). No return-to-original-page logic.
- **D-03:** Sessions live indefinitely until explicit logout or admin revocation. No automatic expiry / TTL.
- **D-04:** Admin can force-logout (revoke session) for a crew member from users.html. Covers lost phone / fired employee scenarios.
- **D-05:** Failed login shows simple "Invalid credentials" error message. No rate limiting or account lockout — not a realistic threat at 1-5 users behind Tailscale.
- **D-06:** Unauthenticated page access handled client-side: frontend JS checks for 401 on first API call and redirects to login.html.

### Dev Workflow
- **D-07:** Go server serves frontend files from disk (os.DirFS) in development, embed.FS in production. Switched via build flag or env var. No rebuild needed for frontend changes in dev.
- **D-08:** Postgres runs as a Docker container during development (docker run postgres:16 with local volume). Independent of the Go server lifecycle.

### Seed Data & Bootstrap
- **D-09:** Superadmin bootstrapped from config/superadmins.yaml on server startup. Server ensures listed emails exist with admin role. Password set via invite flow.
- **D-10:** Goose migrations create schema only — no seed data in migrations. Dev seed data handled separately (Makefile target or seed script).

### Claude's Discretion
- Go project layout (internal/ structure, handler/service split) — follow idiomatic Go patterns
- Makefile targets for common dev tasks (run, migrate, seed, test)
- Health check endpoint design (GET /api/v1/health)
- Session token generation approach (crypto/rand + SHA-256 as specified in research)
- Tailscale Serve configuration specifics

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### API Design
- `docs/user-management-api.md` — Full SQL schema for users, sessions, hq_apps, app_permissions tables. REST API contracts for auth, user CRUD, app permissions. This is the implementation target.

### Stack Research
- `.planning/research/STACK.md` — Go + Postgres stack decisions, version pins, "what not to use" list
- `.planning/research/SUMMARY.md` — Architecture approach, dependency chain, critical prerequisites

### Existing Frontend
- `login.html` — Current mock login page. Wire signIn() to POST /api/v1/auth/login. Currently always shows error.
- `sw.js` — Service worker with cache-first strategy. Must be partitioned: network-first for /api/*, cache-first for static.
- `index.html` — HQ launcher grid. Post-login redirect target.

### Codebase Context
- `.planning/codebase/ARCHITECTURE.md` — Current multi-page PWA architecture
- `.planning/codebase/CONVENTIONS.md` — Naming patterns, code style conventions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `login.html` — Complete login form with email/password fields, error display, dark mode, shared CSS variables. Only needs signIn() rewired from mock to real API call.
- `sw.js` — Working service worker with install/activate/fetch handlers. Needs fetch handler split, not rewrite.
- `ptr.js` — Pull-to-refresh utility, shared across all pages. No changes needed.
- CSS variable system — Shared `:root` block with dark mode. New pages (if any) should reuse.

### Established Patterns
- Each HTML page registers sw.js and includes ptr.js
- SW cache version string must be bumped on every deploy
- `ASSETS` array in sw.js lists all cached files — new files must be added here
- Event delegation with data-action attributes (workflows.html pattern)

### Integration Points
- `sw.js` fetch handler — must be updated to route /api/* requests network-first
- `login.html` signIn() function — replace mock alert with real POST /api/v1/auth/login
- Every tool page needs a 401 check on first API call → redirect to login.html
- `users.html` — will need admin session revocation UI (D-04), but that's Phase 11 scope

</code_context>

<specifics>
## Specific Ideas

- Superadmins.yaml pattern comes from the existing API design doc (docs/user-management-api.md) — superadmins are NOT in the users table, they're bootstrapped from config
- Session revocation from users.html is decided but the UI change belongs in Phase 11 (Users Admin). Phase 9 just needs the API endpoint (DELETE /api/v1/admin/sessions/:id or similar)
- Resend email provider — evaluate free tier limits against actual invite volume (realistically <10 invites total for the life of the app)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 09-foundation-auth*
*Context gathered: 2026-04-14*
