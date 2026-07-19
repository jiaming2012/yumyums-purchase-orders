# Phase 9: Foundation + Auth - Research

**Researched:** 2026-04-14
**Domain:** Go backend shell, Postgres + goose migrations, Tailscale HTTPS, SW fetch partition, httpOnly cookie auth
**Confidence:** HIGH

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Use Resend for transactional email (invite flow, password reset). Generous free tier (100 emails/day), Go SDK, simple DNS setup.
- **D-02:** After successful login, always redirect to index.html (HQ launcher grid). No return-to-original-page logic.
- **D-03:** Sessions live indefinitely until explicit logout or admin revocation. No automatic expiry / TTL.
- **D-04:** Admin can force-logout (revoke session) for a crew member from users.html. Covers lost phone / fired employee scenarios.
- **D-05:** Failed login shows simple "Invalid credentials" error message. No rate limiting or account lockout.
- **D-06:** Unauthenticated page access handled client-side: frontend JS checks for 401 on first API call and redirects to login.html.
- **D-07:** Go server serves frontend files from disk (os.DirFS) in development, embed.FS in production. Switched via build flag or env var. No rebuild needed for frontend changes in dev.
- **D-08:** Postgres runs as a Docker container during development (docker run postgres:16 with local volume). Independent of the Go server lifecycle.
- **D-09:** Superadmin bootstrapped from config/superadmins.yaml on server startup. Server ensures listed emails exist with admin role. Password set via invite flow.
- **D-10:** Goose migrations create schema only — no seed data in migrations. Dev seed data handled separately (Makefile target or seed script).

### Claude's Discretion

- Go project layout (internal/ structure, handler/service split) — follow idiomatic Go patterns
- Makefile targets for common dev tasks (run, migrate, seed, test)
- Health check endpoint design (GET /api/v1/health)
- Session token generation approach (crypto/rand + SHA-256 as specified in research)
- Tailscale Serve configuration specifics

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-01 | Go binary serves PWA static files via embed.FS and API via chi router from same origin | embed.FS + chi static file handler pattern; os.DirFS for dev mode via D-07 |
| INFRA-02 | Postgres 16 database with goose migrations for schema management | goose v3 SQL-file migrations; Docker postgres:16 container per D-08 |
| INFRA-03 | Tailscale Serve provides HTTPS dev access for mobile device testing | `tailscale serve https:443 localhost:8080` — automatic tailnet HTTPS |
| INFRA-04 | Service worker fetch handler partitioned — network-first for /api/*, cache-first for static files | sw.js fetch handler rewrite; SW cache version bump required |
| AUTH-01 | User can log in with email + password via POST /api/v1/auth/login and receive httpOnly session cookie | bcrypt password verify; crypto/rand token; httpOnly Secure SameSite=Strict Set-Cookie |
| AUTH-02 | User can log out via POST /api/v1/auth/logout which invalidates the session | DELETE sessions row by token_hash; clear cookie |
| AUTH-03 | Protected API endpoints reject unauthenticated requests with 401 | chi middleware reads cookie, hashes token, queries sessions table |
| AUTH-04 | login.html wired to real auth API (replacing mock alert()) | Replace signIn() fetch call; store nothing in localStorage; redirect to index.html on 200 |
</phase_requirements>

---

## Summary

Phase 9 is the infrastructure and auth foundation for the entire v2.0 backend. The frontend is completely built (54 passing E2E tests). The work here is: stand up a Go binary with chi that serves the existing PWA files from the same origin as the API, connect it to a Postgres database managed by goose migrations, expose it over Tailscale HTTPS for real-phone testing, partition the service worker fetch strategy before the first API call lands, and wire login.html to a real auth endpoint using httpOnly cookies.

The schema for this phase comes directly from `docs/user-management-api.md`: `users`, `sessions`, `invite_tokens`, `hq_apps`, and `app_permissions` tables. The auth token pattern is opaque: 32 bytes from `crypto/rand`, stored as a SHA-256 hash in the `sessions` table. Per D-03, sessions have no TTL — they live until explicit logout or admin revocation. Per D-06, the 401 redirect is client-side only: login.html redirects to index.html on success, and index.html redirects back to login.html if `GET /api/v1/me` returns 401.

Two blocking prerequisites must land first: (1) the service worker fetch partition — the existing `sw.js` uses cache-first for ALL fetches, which would corrupt every API response on first cache hit; (2) the Go binary serving the PWA from the same origin — Tailscale Serve must be verified on a real iPhone in standalone mode before auth is declared done, because iOS standalone mode uses a separate storage partition and auth testing in Safari or DevTools is not equivalent.

**Primary recommendation:** Build in wave order — Go shell + SW partition first (validate deploy chain on a real phone), then Postgres + migrations, then auth endpoints, then wire login.html. Never declare auth complete without testing on a physical iOS device in standalone mode.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.25.5 (local) | Backend language | Already installed; team choice; single-binary deployment |
| github.com/go-chi/chi/v5 | v5.2.1+ | HTTP router | Stdlib-compatible; no framework lock-in; easy middleware composition |
| github.com/jackc/pgx/v5 | v5.7+ | Postgres driver + pool | Native protocol; connection pooling; `pgxpool` for concurrent API requests |
| github.com/pressly/goose/v3 | v3.27+ | DB migrations | SQL-file migrations; embeddable in binary; `goose.SetBaseFS` for embedded |
| golang.org/x/crypto | latest | bcrypt password hashing | `bcrypt.GenerateFromPassword` / `bcrypt.CompareHashAndPassword`; not in stdlib |
| gopkg.in/yaml.v3 | v3 | Parse superadmins.yaml | D-09 config file; standard yaml package for Go |
| github.com/resend/resend-go/v2 | v2 | Resend email SDK | D-01: invite emails via Resend; official Go client |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| sqlc | latest CLI | Type-safe query codegen | Generates Go from SQL; use after schema is final — not needed for Phase 9 if raw pgx queries are simpler to bootstrap |
| github.com/rs/cors | v1.11+ | CORS middleware (chi) | Not needed for Phase 9 (same-origin serving, no CORS required) — omit unless a use case appears |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| chi v5 | Echo / Gin | chi is closer to stdlib; no magic; middleware composable with `http.Handler` |
| pgx/v5 | database/sql + lib/pq | pgx is faster, has better Postgres type support, and handles `pgxpool` natively |
| goose | golang-migrate | goose runs cleanly embedded; golang-migrate has "dirty state" recovery complexity |
| bcrypt (golang.org/x/crypto) | argon2 | bcrypt is simpler and well-tested at 1-5 user scale; argon2 offers no practical benefit here |
| opaque tokens (crypto/rand) | JWT | JWTs cannot be revoked without a denylist; D-03/D-04 require instant revocation |
| httpOnly cookies | localStorage Bearer tokens | docs/user-management-api.md specified localStorage but research + decisions override: cookies are XSS-safe and work correctly in iOS standalone mode |

**Installation:**
```bash
cd backend
go mod init github.com/yumyums/hq
go get github.com/go-chi/chi/v5
go get github.com/jackc/pgx/v5
go get github.com/pressly/goose/v3
go get golang.org/x/crypto
go get gopkg.in/yaml.v3
go get github.com/resend/resend-go/v2
```

**Version verification:** Go 1.25.5 is confirmed installed. goose, sqlc, and chi are not yet installed — they will be fetched by `go get` during Phase 9 execution.

---

## Architecture Patterns

### Recommended Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go          # Entry point; reads env; runs goose; starts HTTP server
├── internal/
│   ├── auth/
│   │   ├── handler.go       # POST /auth/login, /auth/logout, /auth/accept-invite
│   │   ├── service.go       # Token generation, bcrypt verify, session create/delete
│   │   └── middleware.go    # Session auth middleware — reads cookie, attaches user to ctx
│   ├── db/
│   │   ├── migrations/      # Numbered goose SQL files
│   │   │   ├── 0001_users.sql
│   │   │   ├── 0002_sessions.sql
│   │   │   ├── 0003_invite_tokens.sql
│   │   │   ├── 0004_hq_apps.sql
│   │   │   └── 0005_app_permissions.sql
│   │   └── db.go            # pgxpool setup; Migrate() function
│   ├── config/
│   │   └── config.go        # Env var loading; superadmins.yaml parsing
│   └── me/
│       └── handler.go       # GET /me, GET /me/apps
├── config/
│   └── superadmins.yaml     # D-09: bootstrapped superadmin list
├── go.mod
└── go.sum
```

Static frontend files stay at **repo root** (unchanged). The Go binary embeds them relative to the repo root using `//go:embed` directives in `main.go`.

### Pattern 1: Dev vs Prod Static File Serving (D-07)

**What:** Switch between `os.DirFS` (disk — no rebuild needed during dev) and `embed.FS` (baked into binary for prod) via an environment variable or build tag.

**When to use:** Always — D-07 is a locked decision.

**Example (env var approach):**
```go
// main.go
var staticFS fs.FS

//go:embed *.html *.js *.json manifest.json icons lib
var embeddedFS embed.FS

func main() {
    if os.Getenv("STATIC_DIR") != "" {
        // Dev: serve from disk — no rebuild needed for frontend changes
        staticFS = os.DirFS(os.Getenv("STATIC_DIR"))
    } else {
        // Prod: serve from embedded FS
        staticFS = embeddedFS
    }
    // ...
    r.Handle("/*", http.FileServerFS(staticFS))
}
```

### Pattern 2: Chi Auth Middleware

**What:** Middleware reads the session cookie, SHA-256 hashes the token, queries `sessions` table, attaches user to request context. Endpoints retrieve the user via a typed context key.

**When to use:** Applied to the entire `/api/v1/` route group, with explicit exceptions for `/auth/login`, `/auth/accept-invite`, and `/health`.

**Example:**
```go
// internal/auth/middleware.go
func Middleware(db *pgxpool.Pool, superadmins map[string]SuperadminUser) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            cookie, err := r.Cookie("hq_session")
            if err != nil {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            hash := sha256Token(cookie.Value)
            user, err := lookupSession(r.Context(), db, hash)
            if err != nil {
                http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), ctxKeyUser, user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Pattern 3: Opaque Token Generation + Storage

**What:** 32 bytes from `crypto/rand` encoded as hex. SHA-256 hash stored in `sessions.token_hash`. Raw token sent to client via `Set-Cookie`.

**When to use:** Login endpoint creates token; logout deletes by hash.

**Example:**
```go
// internal/auth/service.go
func generateToken() (raw string, hash string, err error) {
    b := make([]byte, 32)
    if _, err = rand.Read(b); err != nil {
        return
    }
    raw = hex.EncodeToString(b)
    h := sha256.Sum256([]byte(raw))
    hash = hex.EncodeToString(h[:])
    return
}

func setSessionCookie(w http.ResponseWriter, token string) {
    http.SetCookie(w, &http.Cookie{
        Name:     "hq_session",
        Value:    token,
        Path:     "/",
        HttpOnly: true,
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        // No MaxAge/Expires — D-03: sessions are indefinite
    })
}
```

**Note on D-03:** Sessions have no TTL. The `sessions.expires_at` column in `docs/user-management-api.md` was designed with an 8-hour default. Per D-03, this column should be set to a far-future date (e.g., year 9999) or the column definition needs to be optional. The planner must decide: keep `expires_at` nullable (simpler schema) or set a far-future sentinel. **Recommendation:** Make `expires_at` nullable; NULL means no expiry. Session is invalid only when row is deleted (logout/revocation).

### Pattern 4: SW Fetch Handler Partition (INFRA-04)

**What:** The current `sw.js` fetch handler uses cache-first for ALL requests. This must be changed to: network-first (always hit network, fallback to cache) for `/api/*` routes, and cache-first (serve from cache, fallback to network) for static assets.

**When to use:** This is a blocking prerequisite — must land before any API endpoint is wired to the frontend.

**Current sw.js fetch handler (lines 18-22) — needs replacement:**
```js
// CURRENT (broken for API):
self.addEventListener('fetch', e => {
  e.respondWith(
    caches.match(e.request).then(r => r || fetch(e.request).catch(() => caches.match('./index.html')))
  );
});
```

**Replacement:**
```js
// NEW — partitioned by URL path:
self.addEventListener('fetch', e => {
  const url = new URL(e.request.url);
  if (url.pathname.startsWith('/api/')) {
    // Network-first: API calls always go to network; no caching
    e.respondWith(fetch(e.request).catch(() => new Response('{"error":"offline"}', {
      status: 503,
      headers: {'Content-Type':'application/json'}
    })));
    return;
  }
  // Cache-first: static assets served from cache; fallback to network
  e.respondWith(
    caches.match(e.request).then(r => r || fetch(e.request).catch(() => caches.match('./index.html')))
  );
});
```

**Also required:** Bump `CACHE` constant (currently `yumyums-v46`) to `yumyums-v47` or higher to force all existing clients to adopt the new strategy.

### Pattern 5: Goose Migration Setup

**What:** goose migrations embedded in binary, run automatically on startup before the HTTP server starts accepting requests. Migration files in `internal/db/migrations/`, each wrapped in `BEGIN;` / `COMMIT;` with one logical change per file.

**Migration order (FK dependency chain):**
```
0001_users.sql           -- no FK deps
0002_sessions.sql        -- FK: users
0003_invite_tokens.sql   -- FK: users
0004_hq_apps.sql         -- no FK deps (seed data goes here as INSERT, per D-10 only if static)
0005_app_permissions.sql -- FK: hq_apps, users
```

**Example migration file:**
```sql
-- 0001_users.sql
-- +goose Up
BEGIN;
CREATE TABLE users (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email         TEXT UNIQUE NOT NULL,
  display_name  TEXT NOT NULL,
  password_hash TEXT,
  role          TEXT NOT NULL CHECK (role IN ('admin', 'manager', 'team_member')),
  status        TEXT NOT NULL DEFAULT 'invited' CHECK (status IN ('invited', 'active')),
  invited_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  accepted_at   TIMESTAMPTZ
);
COMMIT;

-- +goose Down
DROP TABLE users;
```

**Example embed + run pattern:**
```go
//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
    goose.SetBaseFS(migrationsFS)
    db, err := pool.Acquire(ctx)
    // ... convert pgxpool conn to *sql.DB for goose
    return goose.Up(db.Conn(), "migrations")
}
```

**Note:** goose requires a `*sql.DB` connection (not pgxpool). Use `stdlib.OpenDBFromPool(pool)` from `github.com/jackc/pgx/v5/stdlib` to get a `*sql.DB` from the pgxpool.

### Pattern 6: Superadmin Bootstrap (D-09)

**What:** On startup, parse `config/superadmins.yaml` and hold the list in memory. At the API layer, check if the authenticated user's email is in the in-memory set. Superadmins are NOT stored in the `users` table.

**Critical implication:** `POST /api/v1/auth/login` must check in-memory superadmins first (email match → check password), then fall through to `users` table. Login path must not reveal whether an email is a superadmin or a regular user — return the same 401 for both failure cases.

**Example config parsing:**
```go
// internal/config/config.go
type SuperadminEntry struct {
    Email       string `yaml:"email"`
    DisplayName string `yaml:"display_name"`
}

type SuperadminsConfig struct {
    Superadmins []SuperadminEntry `yaml:"superadmins"`
}

func LoadSuperadmins(path string) (map[string]SuperadminEntry, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var cfg SuperadminsConfig
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    m := make(map[string]SuperadminEntry, len(cfg.Superadmins))
    for _, s := range cfg.Superadmins {
        m[s.Email] = s
    }
    return m, nil
}
```

**Password for superadmin:** The `docs/user-management-api.md` says "Password set via invite flow." This means the superadmin's `password_hash` is either (a) stored separately in config or (b) the superadmin bootstraps themselves via the invite flow. The planner should clarify: superadmin invite flow bootstraps the password hash and stores it WHERE? Options:
- Option A: Superadmin `password_hash` stored in `users` table (superadmin upserted as a special user row on startup) — simple, consistent
- Option B: Superadmin `password_hash` stored in `config/superadmins.yaml` — config-only, but requires manual bcrypt hash generation

**Recommendation (for planner):** Option A is simpler. Upsert superadmin rows into the `users` table on startup (role = 'admin' but with a special in-memory flag). The API layer checks the in-memory set for superadmin privileges, while the users table holds their password_hash.

### Pattern 7: Client-Side 401 Guard (D-06)

**What:** Every tool page calls `GET /api/v1/me` on load. If the response is 401, redirect to `login.html`. No server-side redirect middleware needed.

**Implementation in index.html (and other tool pages):**
```js
// Called on page load, before rendering tiles
async function checkAuth() {
  const res = await fetch('/api/v1/me');
  if (res.status === 401) {
    window.location.href = '/login.html';
    return null;
  }
  return res.json();
}
```

**Note:** `login.html` itself does NOT need a 401 guard (it is the auth page). The SW must not cache the `/api/v1/me` response (handled by the INFRA-04 network-first partition).

### Pattern 8: Tailscale Serve for Dev HTTPS (INFRA-03)

**What:** `tailscale serve` proxies `https://<machine-name>.tailnet.ts.net:443` to `localhost:8080`. Automatic TLS certificate from Tailscale's CA — trusted by devices enrolled in the tailnet.

**Command:**
```bash
tailscale serve https:443 localhost:8080
```

**Note on Tailscale availability:** Tailscale is NOT installed on the development Mac (checked: not in PATH, not in /Applications). It needs to be installed before INFRA-03 can be completed. The Tailscale CLI for macOS is distributed via the Mac App Store or direct download from tailscale.com. The Go server can be developed and tested locally on port 8080 without Tailscale — Tailscale is only required for phone testing.

### Anti-Patterns to Avoid

- **Cache-first for /api/ routes:** The existing SW fetch handler is cache-first for everything. Adding API endpoints without updating sw.js first will silently cache API responses (including 401s, empty arrays, stale data) and serve them forever until the cache is cleared. This is the #1 blocking prerequisite.
- **localStorage for session tokens:** `docs/user-management-api.md` still references `localStorage.hq_token` in the PWA Integration Plan section. Ignore that — use httpOnly cookies per the architecture decisions. The cookie is automatically sent with every fetch request to the same origin, invisible to JS.
- **JWT instead of opaque tokens:** JWTs cannot be revoked (D-03/D-04 require instant admin revocation). Do not use JWTs.
- **Calling `tailscale serve` before confirming auth on a real iPhone:** Desktop testing does not reveal iOS standalone cookie isolation issues. Auth must be verified on a physical iPhone in standalone mode before Phase 9 is declared done.
- **Multi-statement SQL in a single migration without transaction:** If a migration fails mid-way, the DB is left dirty. Wrap every migration in `BEGIN; ... COMMIT;` and goose will roll back on failure cleanly.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Password hashing | Custom hash | `golang.org/x/crypto/bcrypt` | bcrypt handles salt and cost factor; timing-safe compare |
| Session token generation | Math/random | `crypto/rand` | Math/random is not cryptographically secure |
| DB migrations | Custom migration runner | `github.com/pressly/goose/v3` | Handles up/down, state tracking, embedded FS, `*sql.DB` compatibility |
| HTTP routing + middleware | Manual mux | `github.com/go-chi/chi/v5` | stdlib-compatible; middleware chains; URL params |
| Env config | Custom parser | `os.Getenv` + `os.LookupEnv` | Simple enough at this scale; no struct tags needed yet |
| YAML parsing | Custom parser | `gopkg.in/yaml.v3` | superadmins.yaml is the only YAML file; stdlib-adjacent |
| Email sending | `net/smtp` custom | `github.com/resend/resend-go/v2` | D-01 locks Resend; SDK handles retries and API auth |

**Key insight:** The auth token pattern (crypto/rand → hex encode → SHA-256 hash → store in DB) is simple enough to write directly with stdlib. The only external deps genuinely needed for Phase 9 are: chi (routing), pgx/v5 (DB), goose (migrations), golang.org/x/crypto (bcrypt), and resend-go (email). sqlc can be deferred until Phase 10 when query volume justifies codegen.

---

## Common Pitfalls

### Pitfall 1: SW Cache-First Intercepts API Calls

**What goes wrong:** Before sw.js is updated, the existing cache-first handler returns a cached response for any URL that was previously fetched — including a 200 from a static file that happens to share a path prefix, or worse, a cached API response from a prior session.

**Why it happens:** The current fetch handler in sw.js (line 19) calls `caches.match(e.request)` first for ALL requests with no URL filtering.

**How to avoid:** SW partition (INFRA-04) must be the first commit in the phase. Bump CACHE version simultaneously to force all connected clients to re-install the SW.

**Warning signs:** DevTools > Application > Cache Storage shows any URL containing `/api/` after a page load with the API server running.

### Pitfall 2: Auth Works in Desktop Browser but Fails on iPhone Standalone

**What goes wrong:** Login appears to work in Safari on iPhone or in Chrome DevTools device emulation. The app fails silently on the actual installed PWA on a real iPhone — requests are sent without the session cookie, and every API call returns 401.

**Why it happens:** iOS PWA standalone mode uses a separate storage partition from the browser. Cookies set in the browser context are NOT automatically available in the standalone context. Same-origin serving (D-07/INFRA-01) eliminates the cross-origin dimension, but the partition issue still requires the cookie to have been set WHILE in the standalone context.

**How to avoid:** Test login end-to-end on a real iPhone in standalone mode (Home Screen → tap app icon). This is the only valid test for auth completion. Do not declare Phase 9 done based on desktop or Safari testing.

**Warning signs:** Login succeeds in browser, but opening the installed app shows login.html every time without having been logged out.

### Pitfall 3: Superadmin Password Hash Storage Ambiguity

**What goes wrong:** D-09 says superadmin is bootstrapped from config/superadmins.yaml, but docs/user-management-api.md says password is set via invite flow. If the superadmin is NOT in the users table, there is nowhere to store their password_hash, and login fails with a nil pointer or "user not found" error.

**Why it happens:** The two documents were written at different times with different assumptions.

**How to avoid:** Resolve at plan time (before execution). Recommended: upsert superadmin into users table on startup (with role override applied at API layer from in-memory set). The invite flow then works normally for the superadmin to set their initial password.

**Warning signs:** `POST /api/v1/auth/login` with superadmin email returns 500 or "invalid credentials" even with correct password.

### Pitfall 4: expires_at NOT NULL Conflicts with D-03

**What goes wrong:** `docs/user-management-api.md` defines `sessions.expires_at TIMESTAMPTZ NOT NULL` with an "8-hour default" note. D-03 says sessions live indefinitely. Inserting a session without an `expires_at` value violates the NOT NULL constraint.

**Why it happens:** The API design doc predates the D-03 decision.

**How to avoid:** In migration `0002_sessions.sql`, define `expires_at TIMESTAMPTZ` as nullable (drop NOT NULL). NULL means "no expiry." Session middleware skips expiry check when `expires_at IS NULL`.

**Warning signs:** `INSERT INTO sessions` panics at runtime because `expires_at` is not provided.

### Pitfall 5: goose Requires *sql.DB, Not pgxpool

**What goes wrong:** You pass a `*pgxpool.Pool` to `goose.Up()` and get a compile error because goose expects `*sql.DB`.

**Why it happens:** goose uses the standard `database/sql` interface for portability.

**How to avoid:** Use `stdlib.OpenDBFromPool(pool)` from `github.com/jackc/pgx/v5/stdlib` to obtain a `*sql.DB` wrapper that delegates to the pgxpool. Call goose with that DB object.

**Warning signs:** `cannot use pool (type *pgxpool.Pool) as type *sql.DB`.

### Pitfall 6: SameSite=Strict Breaks POST Redirect Flows

**What goes wrong:** If any flow involves an external redirect back to the app (e.g., an OAuth flow or a magic link from an email client), `SameSite=Strict` cookies are NOT sent on the initial cross-site navigation. Login flow works but invite acceptance from email client fails — the cookie from a prior session is absent, and the user gets 401 immediately.

**Why it happens:** Strict mode doesn't send the cookie on cross-site navigation, including clicking links from email clients.

**How to avoid:** For Phase 9, the invite acceptance flow sends the user to a URL with a token query parameter, not relying on an existing session. Accept-invite does NOT require an existing cookie — it creates a new session. So SameSite=Strict is safe for Phase 9.

**Warning signs:** POST /api/v1/auth/accept-invite fails with 401 even when the token is valid.

### Pitfall 7: Docker Daemon Not Running on Dev Mac

**What goes wrong:** `docker run postgres:16 ...` fails immediately because the Docker daemon is not running.

**Why it happens:** Docker Desktop on macOS doesn't auto-start; it must be launched from Applications.

**How to avoid:** Document in the Makefile that Docker must be running. Add a `make check-docker` target that runs `docker info` and exits with a useful error message if the daemon is down.

**Warning signs:** `Cannot connect to the Docker daemon at unix:///var/run/docker.sock` — confirmed present in this environment.

### Pitfall 8: Tailscale Not Installed

**What goes wrong:** INFRA-03 requires `tailscale serve` but Tailscale is not installed on the dev Mac (confirmed: not found in PATH or /Applications).

**Why it happens:** Tailscale must be installed separately — it is not part of Go, Docker, or the project dependencies.

**How to avoid:** Install Tailscale first as part of the wave that addresses INFRA-03. The Go server can be built and the first waves of auth testing done on localhost before Tailscale is set up. Tailscale is only needed for real-phone testing.

**Warning signs:** `command not found: tailscale` — confirmed in this environment.

---

## Code Examples

### Health endpoint (unauthenticated)
```go
// Registered OUTSIDE the auth middleware group
r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
})
```

### Chi route group with auth middleware
```go
// Source: go-chi/chi README — middleware chaining
r.Route("/api/v1", func(r chi.Router) {
    // Unauthenticated
    r.Get("/health", healthHandler)
    r.Post("/auth/login", loginHandler)
    r.Post("/auth/accept-invite", acceptInviteHandler)

    // Protected — auth middleware applied
    r.Group(func(r chi.Router) {
        r.Use(auth.Middleware(pool, superadmins))
        r.Post("/auth/logout", logoutHandler)
        r.Get("/me", meHandler)
        r.Get("/me/apps", meAppsHandler)
        // Phase 9 admin endpoint: force-logout (D-04)
        r.Delete("/admin/sessions/{session_id}", revokeSessionHandler)
    })
})

// Static files — MUST be after all /api/ routes
r.Handle("/*", http.FileServerFS(staticFS))
```

### login.html signIn() replacement
```js
// Replace the existing mock signIn() function entirely:
async function signIn() {
  const email = document.getElementById('email').value.trim();
  const pw = document.getElementById('pw').value;
  const err = document.getElementById('err');
  err.classList.remove('show');
  if (!email || !pw) { err.classList.add('show'); return; }

  try {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password: pw }),
      credentials: 'same-origin'  // ensures cookie is sent/received
    });
    if (res.ok) {
      window.location.href = '/index.html';  // D-02: always land on HQ launcher
    } else {
      err.classList.add('show');  // D-05: simple error, no lockout
    }
  } catch {
    err.classList.add('show');
  }
}
```

### Pgxpool setup
```go
// internal/db/db.go
func NewPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
    config, err := pgxpool.ParseConfig(connStr)
    if err != nil {
        return nil, err
    }
    config.MaxConns = 10
    return pgxpool.NewWithConfig(ctx, config)
}
```

### Makefile targets (D-07, D-08, D-10)
```makefile
# Makefile at repo root or backend/

DB_URL ?= postgres://yumyums:yumyums@localhost:5432/yumyums?sslmode=disable

.PHONY: db-start db-stop migrate dev

db-start:
	docker run -d --name yumyums-pg \
	  -e POSTGRES_USER=yumyums \
	  -e POSTGRES_PASSWORD=yumyums \
	  -e POSTGRES_DB=yumyums \
	  -p 5432:5432 \
	  -v yumyums-pgdata:/var/lib/postgresql/data \
	  postgres:16

db-stop:
	docker stop yumyums-pg && docker rm yumyums-pg

migrate:
	cd backend && go run ./cmd/migrate

dev:
	cd backend && STATIC_DIR=.. DB_URL=$(DB_URL) go run ./cmd/server
```

---

## Runtime State Inventory

> This is NOT a rename/refactor phase. This section is not applicable.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | INFRA-01, all backend | Yes | 1.25.5 | — |
| Docker | INFRA-02 (D-08: Postgres dev container) | Yes (installed) | 20.10.14 | Docker daemon not running — must start Docker Desktop before use |
| Postgres (client psql) | Dev debugging | Yes | 17.2 | — |
| Tailscale | INFRA-03 | No | — | Develop + unit test on localhost:8080; Tailscale install required before phone testing |
| goose CLI | Developer convenience (migrate target) | No | — | Run via `go run ./cmd/migrate` — no CLI install needed |
| sqlc CLI | Query codegen | No | — | Phase 9 can use raw pgx queries; defer sqlc until Phase 10 |

**Missing dependencies with no fallback:**
- Tailscale — required for INFRA-03 (real-phone HTTPS testing). Must be installed before the phone-testing wave of Phase 9. macOS install: https://tailscale.com/download/mac or Mac App Store.
- Docker Desktop must be started before running `make db-start` (daemon was not running at research time).

**Missing dependencies with fallback:**
- goose CLI: use `go run ./cmd/migrate` instead — no CLI install required.
- sqlc CLI: defer to Phase 10; raw pgx queries are fine for Phase 9's small schema.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `database/sql + lib/pq` | `pgx/v5 + pgxpool` | ~2021 | Better Postgres type support, native pooling, faster |
| `golang-migrate` | `goose v3` | Preference shift ~2022 | goose embeds more cleanly; dirty state recovery is simpler |
| `github.com/golang/crypto/bcrypt` | `golang.org/x/crypto/bcrypt` | Same package, moved to x/ | Module path changed — use `golang.org/x/crypto` |
| localStorage Bearer token | httpOnly SameSite=Strict cookie | Architecture decision | XSS-safe; works correctly in iOS standalone mode |
| JWT for sessions | Opaque bearer token | Design decision | Revocable (D-03/D-04); simpler; no decode overhead |
| `net/http.ServeMux` | chi v5 | Chi v5 (2022) | URL params, middleware groups, subrouters |
| `go:embed` directory | Individual file patterns | Go 1.16+ | `//go:embed *.html *.js *.json` — explicit, no surprises |

**Deprecated/outdated:**
- `temporalio/web` Docker image: archived — not relevant to this phase but noted in CLAUDE.md
- `lib/pq` Postgres driver: superseded by `pgx`; no new features being added
- `github.com/dgrijalva/jwt-go`: unmaintained; not to be used

---

## Open Questions

1. **Superadmin password_hash storage**
   - What we know: D-09 says bootstrapped from config; D-09 also says "password set via invite flow"
   - What's unclear: WHERE the password_hash is stored if superadmins are not in the users table
   - Recommendation: Planner should decide. Suggested: upsert superadmin into `users` table on startup (role field in users table is 'admin'; superadmin privilege is an in-memory overlay). This allows the standard invite flow and login path to work for superadmins without special-casing the DB layer.

2. **sessions.expires_at column — nullable or sentinel?**
   - What we know: D-03 says no TTL; the schema in user-management-api.md defines NOT NULL with 8-hour default
   - What's unclear: Whether to make column nullable or use a far-future sentinel (year 9999)
   - Recommendation: Nullable (`expires_at TIMESTAMPTZ`). NULL = no expiry. Simpler semantics; no sentinel magic values.

3. **hq_apps seed data — migration or seed script?**
   - What we know: D-10 says migrations are schema-only; seed data handled separately
   - What's unclear: The `hq_apps` seed data (7 app rows) is static and app-structural — it must exist before the app can function. Is it "seed data" (D-10) or "structural data" that belongs in a migration?
   - Recommendation: Put the `hq_apps` INSERT statements in migration `0004_hq_apps.sql` after the CREATE TABLE. These 7 rows are structural constants, not dev-only test data. The D-10 separation applies to dev-only data like mock templates or test users.

4. **Resend DNS setup**
   - What we know: D-01 locks Resend; invite email is needed for superadmin bootstrap (so they can set a password)
   - What's unclear: Whether yumyums.com DNS is already configured for Resend (SPF/DKIM records)
   - Recommendation: Resend DNS setup requires access to the domain registrar. This is a prerequisite for the invite flow wave. If DNS is not set up, invite emails will be delivered to spam or rejected. Plan should include a Resend DNS setup task as a prerequisite before testing the invite flow.

---

## Sources

### Primary (HIGH confidence)

- `docs/user-management-api.md` — full SQL schema (users, sessions, invite_tokens, hq_apps, app_permissions) and API contracts; primary implementation target
- `.planning/phases/09-foundation-auth/09-CONTEXT.md` — locked decisions D-01 through D-10
- `.planning/research/STACK.md` — stack decisions and version pins (verified April 2026)
- `.planning/research/SUMMARY.md` — architecture approach, critical pitfalls, phase ordering rationale
- `sw.js` (current) — exact fetch handler that needs partitioning; CACHE = 'yumyums-v46'
- `login.html` (current) — exact mock signIn() function that needs replacement
- [go-chi/chi v5 GitHub](https://github.com/go-chi/chi) — routing and middleware patterns
- [pressly/goose v3 GitHub](https://github.com/pressly/goose) — embedded migrations, goose.SetBaseFS
- [pgx v5 pkg.go.dev](https://pkg.go.dev/github.com/jackc/pgx/v5) — pgxpool setup, stdlib wrapper
- [Tailscale Serve docs](https://tailscale.com/kb/1242/tailscale-serve) — HTTPS tailnet proxy command
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) — httpOnly cookie recommendation
- [MDN: SameSite cookies](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#samesite_attribute) — SameSite=Strict behavior on cross-site navigation
- Environment audit (2026-04-14): Go 1.25.5 confirmed; Docker 20.10.14 installed (daemon not running); psql 17.2 installed; Tailscale NOT installed; goose/sqlc NOT installed

### Secondary (MEDIUM confidence)

- [Eli Bendersky: Serving static files and web apps in Go](https://eli.thegreenplace.net/2022/serving-static-files-and-web-apps-in-go/) — same-binary static + API serving with embed.FS
- [Resend Go SDK](https://github.com/resend/resend-go) — official Resend Go client (D-01)
- [Apple Developer Forums: iOS standalone cookie isolation](https://developer.apple.com/forums/thread/125109) — standalone mode storage partition behavior

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all technologies in STACK.md verified against official sources April 2026; local environment audited
- Architecture: HIGH — same-origin serving + embed.FS pattern well-documented; cookie auth on iOS verified via Apple Developer Forums
- Auth patterns: HIGH — opaque token pattern is stdlib-implementable; bcrypt is standard; decisions are locked
- Pitfalls: HIGH — SW cache pitfall observed in current sw.js (confirmed cache-first); iOS standalone is a known documented behavior; Docker daemon down confirmed locally

**Research date:** 2026-04-14
**Valid until:** 2026-05-14 (stack is stable; Go releases quarterly; Tailscale Serve API unlikely to change)
