# Architecture Research

**Domain:** Go + Postgres backend integration with existing static PWA
**Researched:** 2026-04-15
**Confidence:** HIGH (Go serving static + API is well-documented; offline sync limitations on iOS are confirmed by MDN and caniuse)

---

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                     Client: Phone Browser                        │
│  PWA (installed) — iOS Safari / Android Chrome                   │
│                                                                  │
│  ┌──────────────────┐  ┌──────────────────────────────────────┐ │
│  │  Service Worker   │  │  IndexedDB (sync queue + cache)      │ │
│  │  (cache-first)    │  │  localStorage (token, apps cache)    │ │
│  └────────┬─────────┘  └──────────────────────────────────────┘ │
│           │ fetch intercept                                       │
└───────────┼─────────────────────────────────────────────────────┘
            │ HTTPS (Tailscale Serve in dev / Caddy in prod)
┌───────────┼─────────────────────────────────────────────────────┐
│           │          Go HTTP Server (single binary)              │
│  ┌────────▼────────────────────────────────────────────────┐    │
│  │  Router (chi v5)                                         │    │
│  │  GET /  →  static file server (embed.FS)                 │    │
│  │  /api/v1/*  →  API handlers                              │    │
│  └────────┬────────────────────────────────────────────────┘    │
│           │                                                       │
│  ┌────────▼─────────────────┐  ┌────────────────────────────┐   │
│  │  Middleware               │  │  Handlers                   │   │
│  │  - auth (bearer token)    │  │  auth, users, checklists,   │   │
│  │  - request logging        │  │  onboarding, inventory      │   │
│  │  - CORS (dev only)        │  └────────────┬───────────────┘   │
│  └──────────────────────────┘               │                    │
│                                  ┌──────────▼───────────────┐   │
│                                  │  Service layer            │   │
│                                  │  (business logic)         │   │
│                                  └──────────┬───────────────┘   │
│                                  ┌──────────▼───────────────┐   │
│                                  │  sqlc-generated queries   │   │
│                                  │  (type-safe, pgx/v5)      │   │
│                                  └──────────┬───────────────┘   │
└─────────────────────────────────────────────┼───────────────────┘
                                              │
┌─────────────────────────────────────────────┼───────────────────┐
│                      PostgreSQL 16           │                    │
│  ┌────────────────────────────────────────┐ │                    │
│  │  users · sessions · invite_tokens      │ │                    │
│  │  hq_apps · app_permissions             │ │                    │
│  │  checklist_templates · sections        │ │                    │
│  │  checklist_fields · submissions        │ │                    │
│  │  submission_responses · fail_notes     │ │                    │
│  │  submission_rejections · audit_log     │ │                    │
│  │  onboarding_templates · sections       │ │                    │
│  │  onboarding_progress · sign_offs       │ │                    │
│  │  purchase_events · line_items          │ │                    │
│  └────────────────────────────────────────┘ │                    │
└─────────────────────────────────────────────┴───────────────────┘
                                              │
┌─────────────────────────────────────────────┴───────────────────┐
│                   Digital Ocean Spaces (S3-compatible)           │
│   Photo uploads (presigned URL pattern — browser uploads direct) │
└─────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Implementation |
|-----------|----------------|----------------|
| Go HTTP server | Serve static PWA files + REST API from single binary | `net/http` + chi router; embed.FS for static assets |
| chi v5 router | URL routing, middleware grouping, subrouters | `go-chi/chi/v5`; route groups for `/api/v1/` |
| Auth middleware | Bearer token validation on every protected route | Reads `Authorization: Bearer <token>`, hashes it, looks up `sessions` table |
| sqlc + pgx/v5 | Type-safe database queries with Postgres-native driver | sqlc generates Go structs + query functions from SQL schema; pgx/v5 is the driver |
| golang-migrate | Database schema versioning | SQL migration files in `migrations/`; runs on startup via embedded migration runner |
| IndexedDB (client) | Offline queue for pending checklist submissions | Sync queue stores `{id, endpoint, payload, timestamp}`; drained on reconnect |
| Service Worker | Cache-first asset serving + online/offline detection | Existing sw.js extended to fire sync drain on `online` event |
| Digital Ocean Spaces | Photo storage for fail notes and corrective actions | Presigned PUT URL from server; browser uploads direct to Spaces; URL stored in DB |
| Tailscale Serve | Dev-only HTTPS tunnel from dev machine to phone | `tailscale serve https:443 localhost:8080`; phone connects via Tailscale app |
| Caddy | Production HTTPS reverse proxy on Hetzner | `reverse_proxy localhost:8080`; auto Let's Encrypt |

---

## Recommended Project Structure

```
backend/                        # Go module root (separate from frontend)
├── cmd/
│   └── server/
│       └── main.go             # Entry point: config, DB, migrations, start server
├── internal/
│   ├── auth/
│   │   ├── handler.go          # POST /auth/login, /auth/logout, /auth/accept-invite
│   │   ├── middleware.go       # Bearer token extraction + session lookup
│   │   └── service.go          # Password hashing (bcrypt), token generation
│   ├── users/
│   │   ├── handler.go          # GET/POST/PATCH/DELETE /users, /users/invite
│   │   └── service.go          # Invite token lifecycle, role validation
│   ├── apps/
│   │   ├── handler.go          # GET /me/apps, GET/PUT /apps/:slug/permissions
│   │   └── service.go          # Access resolution (role grants + user grants)
│   ├── checklists/
│   │   ├── handler.go          # Templates CRUD, submissions, approvals
│   │   └── service.go          # Approval flow, rejection + re-submit logic
│   ├── onboarding/
│   │   ├── handler.go          # Training templates, progress, sign-offs
│   │   └── service.go          # Sequential section unlock logic
│   ├── inventory/
│   │   ├── handler.go          # Purchase events, line items, spending queries
│   │   └── service.go          # Food cost calculations
│   ├── photos/
│   │   └── handler.go          # POST /photos/presign — generates Spaces presigned URL
│   ├── db/
│   │   ├── queries/            # sqlc input: .sql query files
│   │   │   ├── users.sql
│   │   │   ├── sessions.sql
│   │   │   ├── checklists.sql
│   │   │   └── ...
│   │   └── generated/          # sqlc output: auto-generated Go code (do not edit)
│   │       ├── db.go
│   │       ├── models.go
│   │       ├── users.sql.go
│   │       └── ...
│   ├── config/
│   │   └── config.go           # Loads env vars + superadmins.yaml
│   └── server/
│       └── server.go           # Router setup, middleware wiring, static file embedding
├── migrations/                 # golang-migrate SQL files
│   ├── 001_create_users.up.sql
│   ├── 001_create_users.down.sql
│   ├── 002_create_sessions.up.sql
│   └── ...
├── config/
│   └── superadmins.yaml        # Bootstrapped superadmin list (not in DB)
├── sqlc.yaml                   # sqlc config
├── go.mod
└── go.sum

# Frontend: unchanged flat structure at repo root
index.html
workflows.html
onboarding.html
inventory.html
purchasing.html
users.html
login.html
sw.js
ptr.js
manifest.json
tests/
```

### Structure Rationale

- **`backend/` subdirectory:** Keeps Go code separate from the static frontend files. Frontend continues to live at repo root — no changes to existing flat layout.
- **`internal/` per domain:** Go's `internal/` enforcement prevents accidental cross-package leakage. Domain folders (auth, users, checklists) keep handler + service together — less navigation than deep layered folders.
- **`db/generated/`:** sqlc output is committed (not gitignored) so CI does not need sqlc installed. Treat it as read-only; regenerate with `sqlc generate`.
- **`migrations/`:** Numbered SQL files; golang-migrate runs them in order on startup. Embedded into the binary via `//go:embed migrations/*.sql`.
- **No `pkg/` folder:** This is a service, not a library. Nothing is intended for external import.

---

## Architectural Patterns

### Pattern 1: Go Serves Static Files + API from Same Origin

**What:** The Go binary embeds all frontend HTML/CSS/JS files using `//go:embed` and serves them from the same HTTP server as the API. `/api/v1/` routes go to API handlers; everything else falls through to the embedded file server.

**When to use:** Always. This is the core integration pattern. Same origin eliminates CORS entirely. PWA service worker requires same-origin for registration. Single binary is easy to deploy and run under Caddy.

**Trade-offs:** Re-deploying the Go binary redeploys the frontend too (no separate static CDN). Acceptable for a 1-5 person crew tool.

**Example:**
```go
// internal/server/server.go

//go:embed ../../*.html ../../*.js ../../*.json
var staticFiles embed.FS

func New(cfg *config.Config) *chi.Mux {
    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // API routes — mounted first so they take priority
    r.Route("/api/v1", func(r chi.Router) {
        r.Use(auth.Middleware(cfg))
        r.Mount("/auth", auth.NewHandler(cfg).Routes())
        r.Mount("/users", users.NewHandler(cfg).Routes())
        r.Mount("/checklists", checklists.NewHandler(cfg).Routes())
        // ... other domain handlers
    })

    // Static file fallback — catches everything not matched above
    // Serve from embed.FS; fallback to index.html for PWA navigation
    r.Handle("/*", spaHandler(staticFiles))
    return r
}

func spaHandler(fs embed.FS) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Try to serve the exact file; fall back to index.html
        _, err := fs.Open(r.URL.Path)
        if err != nil {
            r.URL.Path = "/"
        }
        http.FileServer(http.FS(fs)).ServeHTTP(w, r)
    })
}
```

### Pattern 2: Offline Queue with Online-Event Drain (iOS-Safe)

**What:** Background Sync API is not supported on iOS Safari (as of 2026). The fallback is a durable IndexedDB queue that drains opportunistically: on page load, on tab focus, and on the `window` `online` event. This works on all platforms.

**When to use:** For checklist submissions, onboarding progress updates, and any write that could fail due to connectivity. Read-heavy operations (loading today's checklists) use stale-while-revalidate from service worker cache instead.

**Trade-offs:** If the user closes the app while offline and never reopens it, the queued data is not sent. For a food truck crew using the app daily, this is acceptable — the next morning's open triggers the drain.

**Pattern (client-side JS, added to each tool page that has writes):**
```javascript
// Offline queue — IndexedDB store: 'sync_queue'
// Entry shape: { id: uuid, url, method, body, queued_at }

async function queueOrFetch(url, options) {
  if (navigator.onLine) {
    return fetch(url, options);
  }
  // Store in IndexedDB sync queue
  await enqueue({ url, method: options.method, body: options.body });
  return { ok: true, queued: true }; // optimistic response
}

async function drainQueue() {
  const pending = await getAllQueued();
  for (const req of pending) {
    try {
      const resp = await fetch(req.url, { method: req.method, body: req.body,
        headers: { 'Authorization': 'Bearer ' + localStorage.getItem('hq_token'),
                   'Content-Type': 'application/json' } });
      if (resp.ok) await removeQueued(req.id);
    } catch (e) {
      break; // still offline — stop draining
    }
  }
}

window.addEventListener('online', drainQueue);
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') drainQueue();
});
drainQueue(); // drain on page load too
```

**Service worker caching strategy per resource type:**

| Resource | Strategy | Rationale |
|----------|----------|-----------|
| HTML, JS, CSS, manifest | Cache-first (existing sw.js) | PWA shell must work offline |
| `GET /api/v1/checklists/today` | Stale-while-revalidate | Show cached list instantly; refresh in background |
| `GET /api/v1/me` | Network-first (short timeout) | Auth state must be current |
| `POST /api/v1/checklists/submissions` | Network, queue on failure | Write — offline queue pattern above |
| Photo uploads to Spaces | Network only | Presigned URLs are time-limited; cannot queue |

### Pattern 3: sqlc for Type-Safe Queries (No ORM)

**What:** Write SQL schema and named queries in `.sql` files. Run `sqlc generate` to produce type-safe Go functions. No ORM, no query builder — raw SQL with compile-time safety.

**When to use:** All database access in this project. The schema in `docs/user-management-api.md` is already written as SQL — sqlc is a natural fit.

**Trade-offs:** Must re-run `sqlc generate` after any schema or query change. No dynamic query building (use multiple named queries for filter variations). Minor friction; large correctness payoff.

**Example:**
```sql
-- internal/db/queries/checklists.sql

-- name: GetTodayChecklists :many
SELECT ct.*, array_agg(cs.id) as section_ids
FROM checklist_templates ct
JOIN checklist_sections cs ON cs.template_id = ct.id
WHERE (ct.active_days IS NULL OR $1 = ANY(ct.active_days))
  AND (
    $2 = ANY(ct.assigned_to_roles)
    OR $3 = ANY(ct.assigned_to_users)
  )
GROUP BY ct.id;
```

Generated Go usage:
```go
checklists, err := q.GetTodayChecklists(ctx, db.GetTodayChecklistsParams{
    DayOfWeek:   int32(time.Now().Weekday()),
    UserRole:    user.Role,
    UserID:      user.ID,
})
```

### Pattern 4: Presigned URL for Photo Uploads

**What:** The browser requests a presigned S3 PUT URL from the Go server. The Go server generates it using the AWS SDK (Spaces is S3-compatible). The browser then uploads the photo file directly to Spaces using HTTP PUT — the file never passes through the Go server.

**When to use:** All photo captures (fail notes, corrective actions, onboarding evidence). Keeps the Go server stateless with respect to file data.

**Trade-offs:** Presigned URLs expire (15 minutes is typical). If the user is offline when they capture a photo, they cannot upload until connectivity returns and they get a fresh presigned URL. Photo uploads cannot be queued offline — this is an acceptable limitation.

**Flow:**
```
1. Browser: POST /api/v1/photos/presign  { field_id, content_type: "image/jpeg" }
2. Server: generates presigned PUT URL (expires 15m), returns { upload_url, public_url }
3. Browser: PUT <upload_url> with photo blob (directly to Spaces, no server involvement)
4. Browser: stores public_url in IndexedDB, includes it in submission payload
```

### Pattern 5: Tailscale Serve for Dev-to-Phone Access

**What:** `tailscale serve` proxies your local Go server to your tailnet with automatic HTTPS. No router config, no firewall rules, no ngrok account needed. Phone connects via Tailscale app.

**When to use:** Development only. Every phone and laptop that needs to test the PWA installs Tailscale and joins the tailnet.

**Trade-offs:** Requires Tailscale installed on the dev machine and on the test phone. Acceptable for a 1-person dev team.

**Setup:**
```bash
# Start Go server locally on port 8080
./backend/server

# Expose it to the tailnet with HTTPS
tailscale serve https:443 localhost:8080

# Phone accesses via: https://<machine-name>.tail1234.ts.net
# PWA install prompt works because HTTPS is automatic
# Service worker registration works — same-origin, HTTPS satisfied
```

In production, Caddy replaces Tailscale Serve:
```
# Caddyfile on Hetzner
hq.yumyums.com {
    reverse_proxy localhost:8080
}
```

---

## Data Flow

### Authentication Flow

```
login.html: POST /api/v1/auth/login { email, password }
    ↓
auth handler: hash input password (bcrypt.CompareHashAndPassword)
    ↓
DB query: SELECT * FROM users WHERE email = $1
    ↓
If match: generate 32-byte random token, store SHA-256(token) in sessions table
    ↓
Response: { token, expires_at, user }
    ↓
login.html: localStorage.setItem('hq_token', token)
    ↓
Redirect to index.html
    ↓
index.html: GET /api/v1/me/apps (Authorization: Bearer <token>)
    ↓
Server: hash token → lookup sessions → get user → evaluate app_permissions
    ↓
Response: [{ slug, name, icon }, ...]
    ↓
index.html: render only the tiles the user has access to
           localStorage.setItem('hq_apps_cache', JSON.stringify(apps))
```

### Checklist Submission Flow (Online)

```
workflows.html: crew fills out checklist, taps Submit
    ↓
POST /api/v1/checklists/submissions { template_id, version, responses[], fail_notes[] }
    ↓
Server: validate user assignment, create checklist_submissions row
    ↓
If requires_approval: status = 'pending'
If no approval needed: status = 'completed'
    ↓
Insert submission_responses rows (one per field)
Insert submission_fail_notes rows (for triggered fails)
    ↓
Response: { id, status, message }
    ↓
workflows.html: show fireworks (completed) or "awaiting approval" banner
```

### Checklist Submission Flow (Offline → Sync)

```
workflows.html: crew fills out checklist, device offline
    ↓
queueOrFetch() detects navigator.onLine === false
    ↓
Payload stored in IndexedDB sync_queue with { url, method, body }
    ↓
workflows.html: show "Saved offline — will submit when connected" banner
    ↓
--- later, device reconnects ---
    ↓
window 'online' event fires → drainQueue()
    ↓
Each queued item sent to server in order
    ↓
On success: remove from IndexedDB queue
    ↓
UI refreshes: pull-to-refresh or next tab open triggers re-render from API
```

### Manager Approval Flow

```
Manager opens Approvals tab
    ↓
GET /api/v1/checklists/approvals
Server: finds submissions where status = 'pending'
        AND (approver_roles @> [user.role] OR approver_users @> [user.id])
    ↓
Manager reviews, taps Approve or Reject
    ↓
Approve: POST /api/v1/checklists/approvals/:id/approve { reason? }
Server: SET status = 'approved', INSERT submission_audit_log
    ↓
Reject: POST /api/v1/checklists/approvals/:id/reject { rejected_items[] }
Server: SET status = 'rejected', INSERT submission_rejections, INSERT audit_log
    ↓
Crew's next GET /api/v1/checklists/today returns existing_submission with rejection flags
Crew re-completes flagged items → new submission → status = 'pending' again
```

### Photo Upload Flow

```
Crew captures photo (fail note or corrective action)
    ↓
POST /api/v1/photos/presign { field_id, content_type }
Server: AWS SDK generates presigned PUT URL for Spaces bucket
    ↓
Browser: PUT <presigned_url> with photo blob (direct to Spaces, bypasses server)
    ↓
Browser: stores public_url in memory, includes in submission payload
    ↓
POST /api/v1/checklists/submissions includes photo_url in fail_notes[]
```

---

## Integration Points

### New vs Modified Components

| Component | Change Type | What Changes |
|-----------|-------------|--------------|
| `backend/` directory | NEW | Entire Go server; does not touch any frontend file |
| `login.html` | MODIFIED | Wire form POST to `POST /api/v1/auth/login`; store token in localStorage |
| `index.html` | MODIFIED | On load, fetch `/api/v1/me/apps`; render tiles from API response; redirect to login on 401 |
| `workflows.html` | MODIFIED | Replace `MOCK_TEMPLATES`/`MOCK_RESPONSES` reads with API calls; add offline queue logic |
| `onboarding.html` | MODIFIED | Replace mock training data with API calls; add progress persistence |
| `inventory.html` | MODIFIED | Replace mock purchase data with API calls to `/api/v1/inventory` |
| `sw.js` | MODIFIED | Add stale-while-revalidate for select API endpoints; add online-event listener for drain |
| `users.html` | MODIFIED | Wire invite/edit/delete to real user API endpoints |
| `docs/user-management-api.md` | REFERENCE | Already specifies the schema and API contracts — implementation target |
| `migrations/` | NEW | SQL migration files for all 14+ tables |
| `config/superadmins.yaml` | NEW | Bootstrapped superadmin list |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Frontend HTML → Go API | `fetch()` with `Authorization: Bearer <token>` | All API calls use this pattern; token from localStorage |
| Go server → PostgreSQL | pgx/v5 connection pool via sqlc-generated queries | Single pool shared across handlers |
| Go server → Spaces | AWS SDK v2 (`aws-sdk-go-v2`) S3 client | Only used for presigned URL generation; no file data passes through server |
| Service Worker → API | Intercepts fetch; stale-while-revalidate for GET endpoints | SW caches GET responses; writes bypass SW and go direct (or queue) |
| IndexedDB → Server | Drain on online event; ordered by queued_at | Client-side only; server sees normal POST requests |
| Frontend → Tailscale (dev) | HTTPS via tailscale serve | Dev only; phone and laptop join same tailnet |
| Frontend → Caddy (prod) | HTTPS via Caddy reverse proxy | Caddy proxies to `localhost:8080` on Hetzner box |

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Digital Ocean Spaces | Presigned PUT URL from Go; browser uploads direct | Use `aws-sdk-go-v2` with custom Spaces endpoint; same API as S3 |
| Email (invite/reset) | Transactional email via SMTP or API | SendGrid or Postmark; Go `net/smtp` for simple setup. Not MVP-blocking — can log invite URLs to stdout initially |
| Tailscale | `tailscale serve` CLI command on dev machine | No code changes; pure infrastructure |
| Caddy | Caddyfile reverse proxy on Hetzner | No code changes; pure infrastructure |

---

## Suggested Build Order

Dependencies determine this order. Each step is testable in isolation before the next begins.

### Phase 1: Go Server Shell + Static Serving (no auth)

Stand up the Go binary with `chi`, embed the frontend files, serve them on `:8080`. Add one unauthenticated health endpoint `GET /api/v1/health`. Run `tailscale serve https:443 localhost:8080` and confirm the PWA loads on the phone over HTTPS. Service worker registers. No database yet.

**Why first:** Validates the entire deploy chain (Go binary + static serving + Tailscale + phone access) before any business logic is written.

### Phase 2: Database + Migrations + Auth

Add Postgres connection (pgx/v5), golang-migrate running on startup, and the users/sessions/invite_tokens schema. Implement auth endpoints: login, logout, accept-invite. Wire `login.html` to call the real API. Wire `index.html` to call `/api/v1/me/apps`.

**Why second:** Auth gates everything else. Building it early means all subsequent phases can test with real sessions.

### Phase 3: Workflows API

Implement checklist templates CRUD, today's checklists endpoint, submission flow, and approval/reject/unapprove endpoints. Wire `workflows.html` to API calls (replace mock data one endpoint at a time). Add offline queue logic to `workflows.html`.

**Why third:** Workflows is the highest-value feature (core daily use). Approval flow complexity is isolated here before onboarding or inventory is added.

### Phase 4: Onboarding API

Implement onboarding templates CRUD, progress tracking, and sign-off endpoints. Wire `onboarding.html` to replace mock data.

**Why fourth:** Onboarding has similar CRUD patterns to workflows but simpler state machine (no approval loop). Builds on Phase 3 patterns.

### Phase 5: Inventory + Photos

Implement purchase events, line items, and spending query endpoints. Wire `inventory.html`. Implement presigned URL endpoint for Spaces photo uploads. Wire photo capture in `workflows.html` to use presigned URL flow.

**Why fifth:** Inventory is read-heavy and lower-risk. Photos depend on Spaces credentials which can be set up last.

### Phase 6: Users App + Permissions

Wire `users.html` to real user API endpoints (invite, edit, delete, reset password, app permissions). This is intentionally last because users.html is admin-only and doesn't block crew daily use.

---

## Anti-Patterns

### Anti-Pattern 1: Separate Origin for the API Server

**What people do:** Run the Go API on a different domain or port from the static files (e.g., `api.yumyums.com` vs `hq.yumyums.com`).
**Why it's wrong:** Service workers are strictly same-origin. A service worker registered on `hq.yumyums.com` cannot intercept requests to `api.yumyums.com`. Offline queueing breaks. CORS must be configured and maintained. PWA install is complicated.
**Do this instead:** Go serves both static files and API from the same origin. One binary, one port, one domain.

### Anti-Pattern 2: Relying on Background Sync API for iOS

**What people do:** Register a `sync` event in the service worker and assume offline submissions will drain automatically.
**Why it's wrong:** Background Sync API has no support in Safari or iOS Safari as of 2026. For a food truck crew using iPhones, this silently fails.
**Do this instead:** Use the `window` `online` event + page `visibilitychange` event to drain the IndexedDB queue. Works on all platforms including iOS. Background Sync is an enhancement for Android Chrome only, not a requirement.

### Anti-Pattern 3: Storing Bearer Tokens in Cookies

**What people do:** Set session tokens as cookies to avoid localStorage management.
**Why it's wrong:** The existing design (docs/user-management-api.md) uses Bearer tokens in Authorization headers. Cookies introduce CSRF complexity. The PWA's offline queue holds payloads that include the token — localStorage fits this model.
**Do this instead:** Store bearer token in `localStorage` as `hq_token`. Clear on logout. All fetch calls include `Authorization: Bearer ${localStorage.getItem('hq_token')}`.

### Anti-Pattern 4: ORM (GORM) for Database Access

**What people do:** Add GORM to avoid writing SQL.
**Why it's wrong:** The schema is already written as SQL in `docs/user-management-api.md`. GORM would require re-expressing it in Go struct tags. GORM's query abstraction hides bugs and performs worse than pgx with prepared statements. sqlc generates type-safe code from the SQL you already have.
**Do this instead:** sqlc + pgx/v5. Write SQL once, get type-safe Go functions. Schema and queries stay in `.sql` files where they are readable.

### Anti-Pattern 5: Routing Photo Uploads Through the Go Server

**What people do:** Have the browser POST the photo to the Go API, which then uploads to Spaces.
**Why it's wrong:** Photo files can be 2-5 MB. Routing through the Go server doubles the bandwidth cost and adds server latency. The server must buffer the entire file in memory or disk before uploading.
**Do this instead:** Presigned URL pattern. Server generates a time-limited PUT URL. Browser uploads directly to Spaces. Server never touches the file data.

### Anti-Pattern 6: Running DB Migrations Manually

**What people do:** SSH into the server and run `migrate up` by hand on each deploy.
**Why it's wrong:** Easy to forget. Creates state drift between dev and prod. Requires SSH access at deploy time.
**Do this instead:** Embed migrations in the binary (`//go:embed migrations/*.sql`) and run `migrate.Up()` automatically on server startup. Idempotent — running against an already-migrated database is a no-op. Consistent across dev and prod.

---

## Scaling Considerations

This is a 1-5 person crew tool. The architecture is intentionally simple. These are the only scaling concerns worth noting:

| Concern | At current scale (1-5 users) | If it ever grows |
|---------|------------------------------|------------------|
| DB connections | Single pgx pool (10 conns is plenty) | PgBouncer if connection count spikes |
| Concurrent submissions | Postgres handles this fine with row-level locking | No action needed |
| Photo storage | Spaces is effectively unlimited | No action needed |
| Binary size with embedded assets | ~5-10 MB total; fine | No action needed |
| Offline queue conflicts | Rare at 1-5 users; last-write-wins is acceptable | Conflict resolution UI if team grows |

The only likely bottleneck is photo uploads over a poor mobile data connection during a busy service. The presigned URL pattern already handles this correctly — the Go server is not the bottleneck.

---

## Sources

- [Serving static files and web apps in Go — Eli Bendersky](https://eli.thegreenplace.net/2022/serving-static-files-and-web-apps-in-go/) — same-binary static + API pattern
- [Background Synchronization API — MDN](https://developer.mozilla.org/en-US/docs/Web/API/Background_Synchronization_API) — confirmed no iOS support
- [Background Sync API — Can I Use](https://caniuse.com/background-sync) — browser support table
- [Offline-first frontend apps in 2025 — LogRocket](https://blog.logrocket.com/offline-first-frontend-apps-2025-indexeddb-sqlite/) — IndexedDB queue pattern
- [tailscale serve command — Tailscale Docs](https://tailscale.com/kb/1242/tailscale-serve) — HTTPS tailnet exposure
- [sqlc — Compile SQL to type-safe code](https://sqlc.dev/) — query generation from SQL schema
- [pgx v5 — Go Packages](https://pkg.go.dev/github.com/jackc/pgx/v5) — Postgres driver
- [golang-migrate/migrate — GitHub](https://github.com/golang-migrate/migrate) — DB migration tool
- [go-chi/chi v5 — GitHub](https://github.com/go-chi/chi) — HTTP router
- [Digital Ocean Spaces presigned URL — GitHub sample](https://github.com/digitalocean/sample-functions-golang-presigned-url) — Go presigned PUT pattern
- [How We Went All In on sqlc/pgx — brandur.org](https://brandur.org/sqlc) — production endorsement of sqlc + pgx
- `docs/user-management-api.md` — existing schema and API contracts (implementation target)
- `.planning/PROJECT.md` — v2.0 milestone requirements and constraints

---
*Architecture research for: Yumyums HQ v2.0 Go + Postgres backend*
*Researched: 2026-04-15*
