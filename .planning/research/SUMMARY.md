# Project Research Summary

**Project:** Yumyums HQ — Go + Postgres Backend (v2.0)
**Domain:** PWA backend integration — REST API, auth, offline sync, food truck operations
**Researched:** 2026-04-15
**Confidence:** HIGH

## Executive Summary

Yumyums HQ v2.0 is a backend milestone, not a UI milestone. The frontend is fully built and validated (54 Playwright E2E tests passing). The job is to replace every hardcoded mock data array with a real Go + Postgres API, wire the existing HTML pages to that API, and get the app running on real devices via Tailscale for phone testing. The recommended approach is a single Go binary that embeds all frontend static files (via `embed.FS`) and serves both `/api/v1/*` and the PWA shell from the same origin — this eliminates CORS entirely, keeps service worker scope intact, and makes deployment a single artifact on Hetzner with Caddy.

The stack is well-established: Go 1.22+, chi v5 router, pgx/v5 for Postgres access, sqlc for type-safe query generation, goose for migrations, and DO Spaces (presigned URL pattern) for photo storage. Auth uses opaque bearer tokens (SHA-256 hash stored in Postgres sessions table) delivered via httpOnly cookies — not localStorage, which is XSS-vulnerable. Offline sync uses an IndexedDB queue drained on the `online` event, not the Background Sync API, which has zero iOS support. These decisions are explicit because the wrong alternatives are tempting and expensive to fix after the fact.

The critical dependency chain is: infrastructure (Go binary + Tailscale serve) → auth (sessions, invite flow, email) → workflows (template CRUD, submissions, approvals) → onboarding and inventory → users admin. Auth must go first because every other endpoint requires a valid session. The service worker fetch strategy must be updated before the first API call is made from the frontend, or the existing cache-first strategy will intercept and cache API responses, corrupting live data. These two actions — Tailscale setup and SW strategy partition — are blocking prerequisites that must land before any other backend work.

## Key Findings

### Recommended Stack

The backend is straightforward Go with minimal dependencies. Chi provides a lightweight router compatible with the Go stdlib `http.Handler` interface, making it easy to compose middleware without framework lock-in. sqlc + pgx/v5 is the recommended persistence layer — the schema is already fully designed in `docs/user-management-api.md` as SQL, so sqlc can generate type-safe Go functions directly from those files without an ORM re-expression. Goose handles migrations as numbered SQL files embedded in the binary and run automatically on startup.

For development on real phones, Tailscale Serve provides automatic HTTPS to the tailnet without router changes or manual cert import. In production, Caddy on Hetzner with automatic Let's Encrypt replaces it. Both serve from the same domain as the static files, keeping auth cookies same-origin and service worker scope intact.

**Core technologies:**
- Go 1.22+: backend language — team choice, strong stdlib, single-binary deployment
- chi v5.2.5: HTTP router — stdlib-compatible, lightweight, no framework lock-in
- pgx/v5: Postgres driver — native protocol, connection pooling, prepared statements
- sqlc: query codegen — type-safe Go from SQL schema already written; no ORM complexity
- goose v3.27+: DB migrations — SQL-file migrations, embedded in binary, runs on startup
- DO Spaces (S3-compatible): photo storage — presigned PUT URLs; Go server never touches file data
- Tailscale Serve: dev HTTPS — phone testing on real devices with automatic cert
- Caddy: production reverse proxy — auto Let's Encrypt, single `reverse_proxy` line config

**What not to use:**
- JWT for sessions (irrevocable without a denylist; opaque tokens are simpler and equally performant at 1-5 users)
- Background Sync API (no iOS Safari support — use `online` event instead)
- Separate API origin (breaks SW scope, requires CORS, complicates offline queuing)
- GORM or any ORM (schema is already in SQL; sqlc generates from it directly)
- Echo or Gin (chi is closer to stdlib and lighter)

### Expected Features

The v2.0 scope is defined: replace all mock JS data with a real backend. No new UI features. The feature dependency tree is clear and the MVP scope is specific.

**Must have (table stakes) — v2.0 launch blockers:**
- Login / logout / session middleware — every other endpoint requires valid session
- Invite flow + email delivery — owner cannot add crew without it; blocks onboarding
- User CRUD — team management; admin-only surface
- App permissions (GET /me/apps) — tile filtering on index.html depends on this
- Checklist template CRUD — Builder tab currently saves nothing
- Checklist fill-out + submission — core daily crew workflow
- Today's checklists endpoint — My Checklists tab requires server-side filtering
- Approval flow (approve / reject / unapprove + audit log) — Approvals tab is a stub without this
- Item-level correction loop — rejection + re-submit cycle; the accountability payoff
- Photo upload to DO Spaces — photo fields are in the UI; must work end-to-end
- Onboarding template + progress persistence — My Trainings tab saves nothing currently
- Purchase event + line item CRUD — Inventory History tab uses hardcoded mock data
- Tailscale dev deployment — required to test real PWA on a real phone

**Should have (v2.x — add when triggered by real usage):**
- PWA offline queue (IndexedDB + SW sync) — trigger: crew reports losing a submission on the truck
- Receipt ingestion pipeline (photo to structured purchase event) — trigger: owner asks to stop manual entry; validate OCR quality on real receipts first
- Password reset flow — trigger: crew member forgets password (low urgency for small team)
- Stock estimation from real purchase data — trigger: 30+ days of real events accumulated

**Defer to v3+:**
- Food cost % of revenue — requires revenue data source decision (POS integration or manual entry)
- Recipe BOM / menu-level food cost — requires additional data model; separate milestone
- Push notifications for approval events — VAPID infrastructure not justified at 1-5 person scale
- Real-time collaborative checklist completion — single-owner model covers the actual workflow

### Architecture Approach

The system is a single Go binary that serves both the PWA shell (via `embed.FS`) and the REST API from the same HTTP server. This same-origin design is the foundational constraint that makes everything else work cleanly — service worker scope, httpOnly cookies, and offline queue all rely on it. The server exposes chi route groups under `/api/v1/` with auth middleware applied; everything else falls through to the embedded static file server. PostgreSQL is the only data store. Photos go directly to DO Spaces via presigned PUT URLs generated by the server. An unauthenticated `GET /api/v1/health` endpoint validates the deploy chain before any business logic is written.

**Major components:**
1. Go HTTP server (chi) — routes API and serves embedded static files from single binary on `:8080`
2. Auth middleware — Bearer token extraction, SHA-256 hash lookup against sessions table, user attached to request context
3. Domain handlers (auth, users, apps, checklists, onboarding, inventory, photos) — one `handler.go` + `service.go` per domain inside `internal/`
4. sqlc-generated queries via pgx/v5 — type-safe DB access; all SQL lives in `.sql` files in `internal/db/queries/`
5. goose migrations — runs on startup; embedded in binary; idempotent; one logical change per numbered file
6. Service Worker (modified) — partitioned fetch strategy: network-first for `/api/*`, cache-first for static assets
7. IndexedDB sync queue (client-side) — offline checklist submissions queued; drained on `online` + `visibilitychange` events

**Project structure:** `backend/` subdirectory contains the Go module; frontend files remain at repo root unchanged. Go embeds the root HTML/JS/JSON files at compile time via `//go:embed`.

### Critical Pitfalls

1. **Service worker caches API responses** — The existing SW uses cache-first for all fetches. The first API call will be cached and served stale forever, including POST mutation responses. Fix: partition the SW fetch handler to use network-first for `/api/*` paths and network-only for mutations. Bump the SW cache version. This must happen before the first API endpoint is wired to the frontend — it is a blocking prerequisite.

2. **iOS standalone cookie isolation** — Auth works in the desktop browser but silently fails when the PWA is installed to the iOS home screen. Standalone mode is a separate storage partition by design. Fix: serve static files and the API from the same origin. Test auth on a physical iPhone in standalone mode before declaring auth done — not in Safari, not in Chrome DevTools device emulation.

3. **Token storage in localStorage** — The existing `docs/user-management-api.md` spec calls for `localStorage.hq_token`. Research recommends upgrading to httpOnly, Secure, SameSite=Strict cookies instead. localStorage is XSS-readable; any injected script can exfiltrate the session. The cost of switching from localStorage to cookies after multiple phases are built around it is HIGH — decide before Phase 2 begins.

4. **CORS wildcard with credentials** — Using `AllowedOrigins: []string{"*"}` with `AllowCredentials: true` is explicitly rejected by the `rs/cors` library and breaks all credentialed requests. Fix: configure an explicit environment-variable-driven origin allowlist from day one. Configure CORS before writing any handler code.

5. **Migration dirty state on startup** — A failed multi-statement migration leaves the DB in "dirty" state and `golang-migrate` refuses to start the server until manually resolved. Fix: one logical change per migration file, each wrapped in `BEGIN; ... COMMIT;`. Define FK-dependency order in migration numbering (users → sessions → templates → submissions → responses → audit_log). Document the `migrate force <N>` recovery procedure.

6. **Background Sync API not supported on iOS** — Using `ServiceWorkerRegistration.sync.register()` for offline queue drain works on Chrome/Android but silently does nothing on every iPhone. Fix: use `window.addEventListener('online', drainQueue)` plus `visibilitychange` as the drain triggers. Works on all platforms without any browser-specific API.

## Implications for Roadmap

Based on the combined research, a 6-phase structure is recommended. The ordering is strictly dependency-driven: infrastructure must be verified on real mobile before auth is built; auth must be complete before any protected endpoint exists; workflow persistence is the highest-value feature and goes next.

### Phase 1: Foundation — Go Server Shell + Tailscale + SW Partition

**Rationale:** Validates the entire deploy chain (Go binary + Tailscale HTTPS + phone access) before any business logic is written. Also corrects the service worker fetch strategy before the first API call is introduced. Both are blocking — discovering them later is expensive. Phase 1 is complete only when the health endpoint returns 200 on a real phone over Tailscale HTTPS and the SW partition is confirmed via DevTools Cache Storage (no `/api/` URLs appear).

**Delivers:**
- Go binary with chi router serving embedded frontend files on `:8080`
- Unauthenticated `GET /api/v1/health` endpoint
- Tailscale Serve configured; app accessible on real mobile device over HTTPS
- SW fetch handler partitioned: network-first for `/api/*`, network-only for mutations, cache-first for static assets
- SW cache version bumped to force all clients to adopt the new fetch strategy

**Addresses:** Pitfall 2 (SW caches API responses), Pitfall 7 (Tailscale mobile reach)
**Avoids:** Discovering after 3 phases of backend work that mobile access or SW intercepts are broken

### Phase 2: Auth + Database Foundation

**Rationale:** Auth is the dependency root of the entire schema and every protected endpoint. Building it second — immediately after infrastructure is verified — means all subsequent phases test against real sessions. Migration order discipline established here (users table first) prevents FK constraint errors in all later migrations.

**Delivers:**
- Postgres connection (pgx/v5 pool configured)
- goose migration runner on startup; migrations: users → sessions → invite_tokens → hq_apps → app_permissions
- Auth endpoints: `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, `POST /api/v1/auth/accept-invite`
- httpOnly, Secure, SameSite=Strict session cookie set on login
- Session middleware applied to all `/api/v1/` route group (except auth and health)
- `GET /api/v1/me` and `GET /api/v1/me/apps` endpoints
- `login.html` wired to real API; `index.html` tile filtering from `/me/apps`
- Email delivery configured (Resend or Postmark) for invite flow
- Auth tested on physical iPhone in standalone mode — not declared done until this passes

**Addresses:** Pitfall 1 (CORS), Pitfall 3 (localStorage tokens), Pitfall 4 (iOS cookie isolation), Pitfall 5 (migration dirty state), Pitfall 8 (CSRF), Pitfall 9 (migration dependency order)

**Research flag:** Email provider selection (Resend vs Postmark vs net/smtp) needs a quick decision before Phase 2 begins. Both Resend and Postmark have straightforward Go clients; either is fine. The choice affects transactional email configuration setup time.

### Phase 3: Workflows API

**Rationale:** Workflows (checklists) is the highest-value feature — the core daily crew workflow and the reason the app exists. The submission state machine (pending → approved/rejected → correction loop) is the most complex business logic in the system. Isolating it in its own phase gives it focused attention before onboarding or inventory patterns are layered on top.

**Delivers:**
- Migrations: checklist_templates → checklist_sections → checklist_fields → checklist_submissions → submission_responses → submission_fail_notes → submission_rejections → submission_audit_log
- Checklist template CRUD (atomic 3-table write: template + sections + fields in one transaction)
- Template versioning: freeze current version and bump when submissions exist against it
- `GET /api/v1/checklists/today` — filter by active_days, assigned_to_roles, assigned_to_users
- `POST /api/v1/checklists/submissions` with idempotency key (client UUID stored as `client_submission_id`)
- Approval endpoints: approve / reject / unapprove; all transitions write to audit_log
- Item-level correction loop (`submission_rejections` rows returned in today's checklists response)
- `workflows.html` wired to API (mock data arrays replaced one endpoint at a time)
- Offline queue (IndexedDB + `online` event drain) for checklist submissions

**Addresses:** Pitfall 3 (offline sync conflicts), Pitfall 6 (Background Sync iOS gap)

**Research flag:** Template versioning with frozen history needs a focused design session before implementation. The freeze-and-fork pattern (bump version, create new if submissions reference current) has schema implications that are expensive to revise after data exists.

### Phase 4: Onboarding API

**Rationale:** Onboarding has the same CRUD + progress tracking pattern as workflows but a simpler state machine (no approval loop). Building it after workflows means the patterns — sqlc queries, handler structure, migration conventions — are established and can be repeated with less friction.

**Delivers:**
- Migrations: onboarding_templates → onboarding_sections → onboarding_fields → onboarding_progress → onboarding_signoffs → journal_entries
- Onboarding template CRUD
- Section progress tracking (mark complete, record sign-off by manager, store journal entries)
- `onboarding.html` wired to API (mock data replaced)

**Research flag:** Onboarding data model is not specified in `docs/user-management-api.md` (which covers auth and workflows only). The onboarding schema must be designed as the first step of Phase 4 planning, not during execution. Inspect `onboarding.html` frontend data structures to infer the required shape.

### Phase 5: Inventory + Photos

**Rationale:** Inventory is read-heavy and lower-risk than workflows. Photos (presigned URL flow) depend on DO Spaces credentials which can be set up at this stage. Photo upload infrastructure is shared by checklist fail note photos (partially enabled in Phase 3) and future receipt ingestion — implementing it here completes the end-to-end photo workflow.

**Delivers:**
- Migrations: vendors → purchase_item_groups → purchase_events → purchase_line_items
- Purchase event CRUD and line item endpoints
- Spending query endpoints (by category, month-over-month trend)
- `inventory.html` wired to API (mock data replaced)
- `POST /api/v1/photos/presign` — generates DO Spaces presigned PUT URL (expires 15 min)
- Photo capture in `workflows.html` wired to presigned URL flow (fail note photos now actually upload to Spaces)

**Research flag:** DO Spaces presigned URL setup is well-documented but the S3-compatible endpoint for Spaces differs from standard AWS S3 — verify `aws-sdk-go-v2` custom endpoint configuration. The Spaces bucket CORS policy also needs to allow `PUT` from the PWA origin for direct browser uploads.

**Note:** Receipt ingestion pipeline is explicitly NOT included. It depends on external OCR API quality that must be validated on real receipts before committing implementation effort.

### Phase 6: Users Admin + App Permissions

**Rationale:** `users.html` is admin-only and does not block crew daily use. Deferring it until last means the highest-value crew features (workflows, onboarding, inventory) are live and being used before admin tooling is needed. The `hq_apps` and `app_permissions` tables are created in Phase 2 and seeded with static data — Phase 6 makes them fully manageable via the UI.

**Delivers:**
- User CRUD endpoints (list, invite, update role, delete) fully wired to `users.html`
- App permission management (role grants + user grants per app slug) fully wired to Access tab
- Password reset flow (reuses invite token infrastructure from Phase 2)
- `users.html` invite/edit/delete wired to real user API (currently all mock alert/confirm calls)

### Phase Ordering Rationale

- **SW partition before first API call:** If the existing cache-first strategy is not updated in Phase 1, it will corrupt every API interaction from the moment the first endpoint is wired. Non-negotiable prerequisite.
- **Auth before everything else:** Every protected endpoint requires a valid session. Building auth in Phase 2 means all subsequent phases can be built and tested with real sessions.
- **Workflows before onboarding/inventory:** Highest business value plus most complex state machine. Isolating it reduces risk of the approval loop logic getting entangled with other domain logic.
- **Onboarding after workflows:** Same CRUD patterns, simpler state machine — flows naturally from established Phase 3 conventions.
- **Inventory/Photos in Phase 5:** DO Spaces credentials and presigned URL integration can be set up after core workflow functionality is validated. Photo infrastructure also unlocks full end-to-end checklist photo fields.
- **Users admin last:** Admin-only; does not block any crew daily use; app permissions seeded in Phase 2 and only fully manageable in Phase 6.

### Research Flags

Phases likely needing deeper research or design sessions during planning:

- **Phase 3 (Workflows):** Template versioning with frozen history — the freeze-and-fork schema pattern needs a design session before implementation. Also verify that the idempotency key approach for offline sync matches the existing `MOCK_RESPONSES` flat dict structure in `workflows.html` so the API response shape is correct for the frontend.
- **Phase 4 (Onboarding):** Schema not specified in `docs/user-management-api.md`. Requires an explicit schema design step at the start of Phase 4 planning.
- **Phase 5 (Inventory/Photos):** DO Spaces presigned URL + `aws-sdk-go-v2` custom endpoint configuration needs verification. Spaces bucket CORS policy for direct browser `PUT` uploads needs to be confirmed.

Phases with standard patterns (skip deep research):

- **Phase 1 (Infrastructure):** Go + chi + embed.FS + Tailscale Serve are all well-documented with official sources. Execute directly.
- **Phase 2 (Auth):** Opaque bearer tokens, httpOnly cookies, bcrypt — well-documented patterns. `docs/user-management-api.md` is the implementation target. Execute directly once email provider is chosen.
- **Phase 6 (Users Admin):** CRUD endpoints + permission evaluation follow the same patterns established in Phases 2-3. No novel complexity.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technologies verified against official docs and release pages; versions pinned and current as of research date (April 2026) |
| Features | HIGH | Grounded in existing `docs/user-management-api.md` (designed specifically for this project) and `.planning/PROJECT.md`; complexity estimates based on direct codebase inspection |
| Architecture | HIGH | Go serving static + API from same binary verified via multiple authoritative sources; iOS PWA limitations verified via MDN and Apple Developer Forums |
| Pitfalls | HIGH | CORS, cookie isolation, Background Sync iOS gap — all verified via official sources (MDN, OWASP, Apple Developer Forums, rs/cors library docs, golang-migrate GitHub) |

**Overall confidence:** HIGH

### Gaps to Address

- **Onboarding schema:** Not specified in `docs/user-management-api.md`. Must be designed before Phase 4 begins. Inspect `onboarding.html` frontend data structures to infer the required shape; treat that as a design input, not a spec.

- **httpOnly cookies vs localStorage for token storage:** The existing `docs/user-management-api.md` spec explicitly calls for `localStorage.hq_token`. Research recommends upgrading to httpOnly cookies. This is a decision that must be made and documented before Phase 2 begins — changing it after multiple phases are built around one approach is HIGH cost.

- **Receipt OCR API quality on real receipts:** The receipt ingestion pipeline is intentionally deferred. Mindee/Veryfi accuracy on food truck receipts (often thermal paper, handwritten totals, faded ink) is unknown. Validate OCR quality on 5-10 real receipts before committing implementation effort.

- **Revenue data source for true food cost %:** Food cost as a percentage of revenue is not calculable without a revenue source. Spending-by-category ("Spending Insights") is achievable in v2.0. True food cost % is deferred to a future milestone requiring a POS integration or manual revenue entry decision.

- **Email provider selection:** Resend vs Postmark vs raw `net/smtp` — both Resend and Postmark have clean Go clients. Decision should happen before Phase 2 begins so invite flow email is testable during auth development.

- **Photo access control:** Photos stored as public Spaces URLs in `submission_responses.value` are readable by anyone with the URL. A decision is needed before Phase 5: serve photo access through an authenticated endpoint (generating short-lived presigned GET URLs) or accept public URLs for v2.0 and add access control later.

## Sources

### Primary (HIGH confidence)

- `docs/user-management-api.md` — existing schema + API contracts; the primary implementation target for auth, users, and workflows
- `.planning/PROJECT.md` — v2.0 active requirements and milestone scope
- [MDN: Background Synchronization API](https://developer.mozilla.org/en-US/docs/Web/API/Background_Synchronization_API) — confirmed no iOS Safari support
- [OWASP: Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) — httpOnly cookie recommendation
- [Apple Developer Forums: iOS PWA standalone cookie isolation](https://developer.apple.com/forums/thread/125109) — standalone mode storage partition behavior
- [rs/cors library docs](https://github.com/rs/cors) — AllowedOrigins + AllowCredentials behavior; wildcard rejection with credentials
- [golang-migrate/migrate GitHub](https://github.com/golang-migrate/migrate) — dirty state and recovery pattern
- [Tailscale Docs: tailscale serve](https://tailscale.com/kb/1242/tailscale-serve) — HTTPS tailnet proxy setup
- [sqlc](https://sqlc.dev/) — query generation from SQL schema
- [pgx v5](https://pkg.go.dev/github.com/jackc/pgx/v5) — Postgres driver documentation
- [go-chi/chi v5](https://github.com/go-chi/chi) — HTTP router

### Secondary (MEDIUM confidence)

- [Eli Bendersky: Serving static files and web apps in Go](https://eli.thegreenplace.net/2022/serving-static-files-and-web-apps-in-go/) — same-binary static + API pattern
- [brandur.org: How We Went All In on sqlc/pgx](https://brandur.org/sqlc) — production endorsement of sqlc + pgx approach
- [LogRocket: Offline-first frontend apps in 2025](https://blog.logrocket.com/offline-first-frontend-apps-2025-indexeddb-sqlite/) — IndexedDB queue pattern; Background Sync iOS gap
- [Netguru: Sharing cookies between PWA standalone and Safari on iOS](https://www.netguru.com/blog/how-to-share-session-cookie-or-state-between-pwa-in-standalone-mode-and-safari-on-ios) — standalone cookie behavior
- [BetterStack: Database migrations in Go with golang-migrate](https://betterstack.com/community/guides/scaling-go/golang-migrate/) — migration patterns
- [Sachith Dassanayake: Offline sync and conflict resolution patterns (April 2026)](https://www.sachith.co.uk/offline-sync-conflict-resolution-patterns-crash-course-practical-guide-apr-8-2026/) — conflict resolution overview
- [Mindee Receipt OCR API](https://developers.mindee.com/docs/receipt-ocr) — structured receipt extraction, free tier 250 pages/month
- [Digital Ocean Spaces presigned URL sample](https://github.com/digitalocean/sample-functions-golang-presigned-url) — Go presigned PUT pattern

### Tertiary (LOW confidence)

- [Supy: Food cost percentage benchmarks](https://supy.io/blog/how-to-calculate-food-cost-percentage) — restaurant food cost target ranges 25-35% (contextual; not directly used in implementation)

---
*Research completed: 2026-04-15*
*Ready for roadmap: yes*
