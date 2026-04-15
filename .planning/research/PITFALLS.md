# Pitfalls Research

**Domain:** Adding Go + Postgres backend to an existing vanilla JS PWA (v2.0 backend milestone)
**Researched:** 2026-04-15
**Confidence:** HIGH (CORS, token storage, Tailscale verified via official docs + multiple sources; offline sync strategy verified via MDN + community; migration pitfalls verified via golang-migrate GitHub + official docs)

---

## Critical Pitfalls

---

### Pitfall 1: CORS Misconfigured for Credentialed Requests

**What goes wrong:**
The frontend starts calling the Go API and immediately hits CORS errors. The developer adds `AllowedOrigins: []string{"*"}` to get unblocked, which works for unauthenticated endpoints. Then auth is wired up using `credentials: 'include'` (cookies) or `Authorization` headers, and CORS breaks again — browsers refuse credentialed requests when the origin is a wildcard. The developer then reflects the request `Origin` header back verbatim instead of validating against an allowlist, which is functionally equivalent to `*` and is a security vulnerability.

**Why it happens:**
CORS has two distinct modes: simple (no credentials) and credentialed. Using `*` unblocks the first mode but is silently disallowed for the second. The mismatch only surfaces once auth is added, by which point the developer has already moved on mentally. The `rs/cors` package's `AllowedOrigins: []string{"*"}` with `AllowCredentials: true` will produce a runtime error or block requests — the library was specifically patched to prevent this combination.

**How to avoid:**
Configure `rs/cors` with an explicit, environment-driven origin allowlist from day one:
```go
c := cors.New(cors.Options{
    AllowedOrigins:   []string{"https://yourdomain.tailnet-name.ts.net", "http://localhost:3000"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    AllowedHeaders:   []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           300,
})
```
Drive origins from an environment variable (`CORS_ALLOWED_ORIGINS=...`) so local dev, Tailscale dev, and production are separate. Never use `*` when `AllowCredentials: true`.

**Warning signs:**
- `Access-Control-Allow-Origin: *` anywhere in the Go config
- Origin reflected verbatim from the `Origin` request header without validation
- CORS middleware added in a hurry after auth was wired up rather than before
- Preflight `OPTIONS` requests returning 404 (middleware not mounted correctly)

**Phase to address:** Phase 1 (API foundation + auth) — CORS must be correct before a single authenticated endpoint goes live. Configure before writing any handler code.

---

### Pitfall 2: Service Worker Intercepts API Calls and Serves Stale Data

**What goes wrong:**
The existing service worker (`sw.js`) uses a cache-first strategy for all fetch events and falls back to `index.html` for failures. When API calls to `/api/v1/...` are introduced, the service worker intercepts them. On the first request, the response is cached. Every subsequent call — including POST and mutation requests — may be served from cache instead of the network. Crew members see stale checklist data or think they submitted a completion but the SW served the previous response.

The current SW fetch handler:
```js
// sw.js (current)
self.addEventListener('fetch', e => {
  e.respondWith(caches.match(e.request).then(r => r || fetch(e.request)
    .catch(() => caches.match('./index.html'))));
});
```
This will cache the first `/api/v1/checklists` response and serve it forever.

**Why it happens:**
The existing SW was written for a static site with no dynamic data. Its cache-first strategy is appropriate for HTML/CSS/JS assets but catastrophic for API calls. Adding a backend changes the fetch landscape without changing the SW strategy.

**How to avoid:**
Partition the SW fetch strategy by request type. API calls use network-first (or network-only for mutations); static assets stay cache-first:
```js
self.addEventListener('fetch', e => {
  const url = new URL(e.request.url);
  // API calls: network-first, never cache mutations
  if (url.pathname.startsWith('/api/')) {
    if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(e.request.method)) {
      // Mutations: network-only, queue offline
      e.respondWith(fetch(e.request).catch(() =>
        new Response(JSON.stringify({error:'offline'}), {status:503, headers:{'Content-Type':'application/json'}})));
      return;
    }
    // GET: network-first with short cache TTL
    e.respondWith(fetch(e.request)
      .then(r => r)
      .catch(() => caches.match(e.request)));
    return;
  }
  // Static assets: cache-first (existing behavior)
  e.respondWith(caches.match(e.request).then(r => r || fetch(e.request)
    .catch(() => caches.match('./index.html'))));
});
```
Also bump the SW cache version (`yumyums-v40` or higher) to force all clients to adopt the new fetch strategy.

**Warning signs:**
- SW cache version not bumped when API endpoints are added
- No `pathname.startsWith('/api/')` check in the SW fetch handler
- POST requests appearing in the Cache Storage DevTools panel
- "Submit checklist" appears to succeed offline but the server never receives the request

**Phase to address:** Phase 1 (API foundation) — update SW fetch strategy before the first API call is made from the frontend. This is a blocking change; the old strategy will corrupt API behavior the moment endpoints are added.

---

### Pitfall 3: Offline Checklist Completion Conflicts on Sync

**What goes wrong:**
Crew member A completes the "Setup Checklist" while offline. Crew member B, who was online, also completes the same checklist 10 minutes later. When A's phone reconnects, the offline queue replays the completion to the server. The server now has two completion records for the same checklist submission. The approval UI shows the checklist as "submitted twice" or throws a unique constraint violation.

A subtler variant: crew member A fills items 1-5 offline, crew member B fills items 6-10 online on the same shared checklist. When A syncs, the server must merge partial completions rather than reject A's submission as a duplicate.

**Why it happens:**
Checklist completion was designed as an atomic "submit the whole thing" action in the frontend mock. In a real multi-user offline context, the atomicity assumption breaks. The backend schema and the sync protocol need to model partial completion and idempotent submission from the start.

**How to avoid:**
- Assign a **client-generated UUID** (`crypto.randomUUID()`) to each submission before it is sent. Store this as `client_submission_id` on the server. The server deduplicates on this key — a replay of the same offline queue entry is a no-op.
- Model responses at the **field level** (one row per field answer), not as a blob per submission. This allows partial sync without conflicts.
- For the offline queue in IndexedDB, store: `{id: uuid, templateId, fieldId, value, userId, timestamp}` per field answer — not one record per full checklist submission.
- The server endpoint `POST /api/v1/responses` accepts an array of field answers, each with its `client_id`. Server inserts with `ON CONFLICT (client_id) DO NOTHING`.
- Declare the conflict resolution policy explicitly: "last write wins per field, per user" — not across users. Two users completing the same field is a team coordination problem, not a data problem. Flag it in the approval UI rather than silently resolving it.

**Warning signs:**
- No `client_submission_id` or idempotency key in the API contract
- Server submission endpoint accepts a single monolithic blob (no field-level granularity)
- Offline queue stores entire checklist state, not individual field changes
- No `ON CONFLICT` clause on the responses table insert

**Phase to address:** Phase 2 (checklist persistence) — design the field-level response schema and idempotency key before writing any sync logic. This is a schema decision that is expensive to change after data exists.

---

### Pitfall 4: Auth Token Storage — localStorage Is XSS-Vulnerable

**What goes wrong:**
Bearer tokens are stored in `localStorage` because it is simple and works immediately. The existing codebase uses no third-party JS beyond SortableJS (CDN), but future dependencies or a CDN compromise introduce XSS. Any injected script can read `localStorage.getItem('token')` and exfiltrate the session token. For a food truck operations app where the owner's session controls template creation, user management, and approvals, this is a meaningful risk.

**Why it happens:**
`localStorage` is the path of least resistance. It survives page reloads, is trivially readable from JS, and requires no special headers. The complexity of httpOnly cookies (CORS `credentials: include`, `SameSite` configuration, CSRF protection) makes developers defer the correct approach.

**How to avoid:**
Use **httpOnly, Secure, SameSite=Strict cookies** for the session token. This is the OWASP-recommended approach for 2025. The cookie is invisible to JavaScript — XSS cannot steal it.

The Go backend sets the cookie on login:
```go
http.SetCookie(w, &http.Cookie{
    Name:     "session",
    Value:    sessionToken,
    HttpOnly: true,
    Secure:   true,          // HTTPS only (Tailscale Serve provides this)
    SameSite: http.SameSiteStrictMode,
    Path:     "/",
    MaxAge:   86400 * 30,    // 30 days
})
```

The frontend `fetch` calls include `credentials: 'include'` to send the cookie. No token is ever stored in JS-readable storage.

**Warning signs:**
- `localStorage.setItem('token', ...)` anywhere in the frontend code
- Authorization header built by reading from `localStorage` or `sessionStorage`
- IndexedDB used to persist a session token across page loads

**Phase to address:** Phase 1 (auth implementation) — the token storage strategy must be decided before login.html is wired to the real API. Changing from localStorage to cookies after the fact requires updates to every fetch call.

---

### Pitfall 5: iOS Safari PWA Cookie Isolation Breaks Auth

**What goes wrong:**
Auth works perfectly in desktop Chrome and desktop Safari. On iOS, crew members install the PWA to the home screen. When the PWA opens in standalone mode, iOS creates a **separate storage partition** from Safari proper — cookies set in the browser do not transfer to the standalone PWA context. The crew member logs in, the cookie is set, and the next time they open the PWA from the home screen the session is gone.

A subtler issue: iOS Safari treats cross-origin fetch requests from a standalone PWA as a "full CORS request" even when the domain matches, causing `credentials: 'include'` fetches to fail unless `Access-Control-Allow-Credentials: true` and a specific (not wildcard) `Access-Control-Allow-Origin` header are set.

**Why it happens:**
iOS PWA standalone mode is a separate browsing context with storage isolation by design. This is documented but not widely understood. SameSite=Strict cookies from a standalone PWA that calls a different origin (even on the same Tailscale network) will not be sent.

**How to avoid:**
- **Same origin is the cleanest solution:** Serve both the static PWA and the Go API from the same domain (e.g., proxy via Tailscale Serve or Caddy). With same-origin, SameSite=Strict cookies work. Cross-origin calls become same-origin calls.
- If cross-origin is unavoidable: use `SameSite=None; Secure` cookies and ensure the CORS `AllowedOrigins` list includes the exact PWA origin.
- Test auth on a real iOS device with the PWA installed to the home screen — **not** in Safari, **not** in Chrome DevTools device emulation — before declaring auth done.
- `fetch('/api/v1/auth/login', { credentials: 'same-origin' })` when serving from same origin is simpler than `credentials: 'include'` cross-origin.

**Warning signs:**
- Auth tested only in desktop browsers during development
- API server running on a different port than the static file server (different origins)
- No end-to-end test that opens the app in standalone mode on a real iOS device
- `SameSite=Strict` set while making cross-origin fetch calls

**Phase to address:** Phase 1 (auth + Tailscale dev setup) — verify cookie auth on a real iOS device before moving to Phase 2. This is a blocker; discovery after multiple phases of backend work have been built around cross-origin auth is expensive to fix.

---

### Pitfall 6: Migration Dirty State Blocks Startup

**What goes wrong:**
A migration fails midway — a Postgres DDL statement in a multi-statement migration partially executes, or the Go process is killed during `migrate up`. `golang-migrate` marks the current version as "dirty" in the `schema_migrations` table. The next time the server starts and runs `migrate.Up()`, it refuses to proceed with error: `Dirty database version N. Fix and force version`. The server will not start. In production or Tailscale dev, this blocks the entire team.

A related issue: migrations are written without considering foreign key dependency order. The `responses` table references `submissions`, `submissions` references `templates` and `users`. If these are created in the wrong order, the migration fails with a FK constraint error and leaves the schema in a partially applied dirty state.

**Why it happens:**
Migrations that contain multiple DDL statements are not atomic unless explicitly wrapped in a transaction. Postgres supports transactional DDL (`CREATE TABLE`, `ADD COLUMN` inside `BEGIN/COMMIT`), but developers forget to add the transaction wrapper, or they mix transactional and non-transactional statements in one file.

**How to avoid:**
- Each migration file contains **exactly one logical change** — create one table, add one column. Never bundle multiple unrelated changes.
- Wrap every migration in `BEGIN; ... COMMIT;` explicitly. This ensures a midway failure rolls back cleanly and does not leave the DB dirty.
- Define creation order to respect FK dependencies: `users` → `sessions` → `templates` → `submissions` → `responses` → `rejection_flags` → `audit_log`. Down migrations drop in reverse order.
- Recovery procedure: fix the SQL error, then `migrate force N` (where N is the failing version) to clear the dirty flag, then `migrate up`. Document this in the project CLAUDE.md so any developer can recover.

**Warning signs:**
- Migration files with 5+ DDL statements bundled together
- No `BEGIN;` / `COMMIT;` wrapping in migration SQL files
- Tables created without verifying the FK dependency order
- `migrate.Up()` called in the server startup path without error handling that distinguishes "dirty" from other errors

**Phase to address:** Phase 1 (database schema) — migration strategy (one change per file, transaction-wrapped, dependency-ordered) must be established as the pattern for the first migration. All subsequent migrations inherit the pattern.

---

### Pitfall 7: Tailscale Dev Setup — Mobile Devices Cannot Reach the Server

**What goes wrong:**
The Go API is running on `localhost:8080`. The developer's laptop is on the Tailscale network. The crew member's phone has Tailscale installed but the phone cannot reach `<machine-name>.tailnet.ts.net:8080` because:
1. The Go server is bound to `127.0.0.1` (loopback) instead of `0.0.0.0` — Tailscale traffic comes in on a separate interface
2. MagicDNS is not enabled in the Tailscale admin console, so the hostname does not resolve on the phone
3. The PWA was served without HTTPS; iOS standalone mode requires HTTPS for service workers and `Secure` cookies — the app installs but auth fails silently

**Why it happens:**
Local development defaults to `localhost`/`127.0.0.1` throughout. Tailscale creates an additional network interface (`100.x.x.x` address range) that requires the server to bind to `0.0.0.0` or specifically to the Tailscale interface address.

**How to avoid:**
Use `tailscale serve` as the HTTPS terminating proxy in front of the Go API:
```bash
tailscale serve --bg http://localhost:8080
```
This:
- Provides automatic HTTPS via Tailscale's Let's Encrypt integration
- Exposes the service at `https://<machine>.tailnet-name.ts.net` on port 443
- Resolves via MagicDNS on all tailnet devices automatically (enable MagicDNS in admin console)
- Handles HTTPS cert distribution — no manual cert import on mobile devices required

The Go server continues to bind to `localhost:8080`; Tailscale Serve handles the external interface. The static PWA files can be served from the same domain via `tailscale serve` proxying to a local static file server, making the whole setup same-origin.

**Warning signs:**
- Go server started with `http.ListenAndServe(":8080", ...)` but tested only via `localhost`
- MagicDNS not enabled in the Tailscale admin panel
- No `tailscale serve` config in the project documentation
- HTTPS tested only via self-signed cert installed manually on one device

**Phase to address:** Phase 1 (infrastructure + dev setup) — Tailscale Serve configuration must be documented and verified on at least one mobile device before backend development begins. Every subsequent phase depends on mobile testing being possible.

---

## Moderate Pitfalls

---

### Pitfall 8: Background Sync API Has No iOS Support

**What goes wrong:**
The team builds offline checklist submission using the Background Sync API (`ServiceWorkerRegistration.sync.register('sync-checklists')`). It works in Chrome on Android and desktop. On iOS (which is the primary target device — food truck crew use iPhones), the `sync` event never fires. Safari does not implement Background Sync as of 2025. The offline queue silently does nothing on iOS.

**How to avoid:**
Do not depend on Background Sync API. Use the **online event** as the sync trigger instead:
```js
window.addEventListener('online', () => replayOfflineQueue());
```
Also retry on page visibility change (`visibilitychange` to visible) and on each page load when `navigator.onLine` is true. Store the offline queue in IndexedDB (not `localStorage` — the queue can be large and must survive SW restarts). Each queued item carries a `clientId` (UUID) for server-side deduplication.

This approach works on iOS, Android, and desktop without any browser-specific API.

**Warning signs:**
- `registration.sync.register(...)` in the SW or page JS without a fallback
- Offline sync tested only in Chrome DevTools or Android

**Phase to address:** Phase 3 (offline mode) — design the sync trigger mechanism correctly from the start. Retrofitting from Background Sync to event-driven sync after offline mode is shipped requires touching every sync point.

---

### Pitfall 9: CSRF Exposure When Using httpOnly Cookies

**What goes wrong:**
Switching from localStorage bearer tokens to httpOnly cookies introduces CSRF vulnerability. A malicious third-party site can trigger a state-changing request (e.g., `POST /api/v1/submissions`) from the user's browser, and the browser will automatically attach the httpOnly session cookie. The API cannot tell whether the request came from the legitimate PWA or from a malicious page.

**How to avoid:**
Three-layer defense:
1. **`SameSite=Strict`** on all session cookies. This prevents the cookie from being sent on cross-site requests entirely. Effective for same-origin deployments.
2. For any cross-origin endpoints: add a **CSRF token** as a custom request header (`X-CSRF-Token`). Custom headers trigger a CORS preflight — cross-origin pages cannot set custom headers without your CORS policy allowing them.
3. **`Content-Type: application/json`** enforcement on all POST handlers. Non-simple content types trigger preflight; HTML form submissions use `application/x-www-form-urlencoded` which is a simple request.

For this project (SameSite=Strict, same-origin serving via Tailscale Serve / Caddy), SameSite alone is sufficient. Document the protection rationale so future maintainers do not loosen the cookie config.

**Warning signs:**
- `SameSite=None` set without a documented reason
- API endpoints that accept `application/x-www-form-urlencoded` POST requests
- No CSRF protection documented in the API design

**Phase to address:** Phase 1 (auth implementation) — cookie flags are set once at login implementation. Getting them right initially costs nothing; fixing a CSRF gap after the fact requires auditing every state-changing endpoint.

---

### Pitfall 10: Postgres Schema Has No Migration for the Auth Tables First

**What goes wrong:**
The Go backend is developed in the order "let me get the interesting stuff working first." Template persistence is wired up in migration `001`, checklist responses in migration `002`, and auth/users in migration `003`. When the application server starts, it runs migrations in sequence. But the checklist response handler needs a `user_id` foreign key to the `users` table — which does not exist until migration `003`. The app can start with an inconsistent schema if migrations are run partially, or the team ends up rewriting the first two migrations to add the FK after the fact.

**How to avoid:**
Auth is not "interesting" from a product perspective, but it is the **dependency root** of the entire schema. Create migrations in dependency order:
1. `001_create_users.up.sql`
2. `002_create_sessions.up.sql`
3. `003_create_templates.up.sql`
4. `004_create_submissions.up.sql`
5. `005_create_responses.up.sql`
6. `006_create_rejection_flags.up.sql`
7. `007_create_audit_log.up.sql`

No migration should reference a table created by a later-numbered migration. Treat this ordering as a constraint at migration creation time, not something to check later.

**Warning signs:**
- Migration `001` creates a `templates` or `submissions` table
- FK constraints are deferred or omitted from early migrations "to add later"
- The migration numbering does not match the documented 7-table schema in `docs/user-management-api.md`

**Phase to address:** Phase 1 (database schema design) — write the migration order as the very first step, before any Go handler code.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `localStorage` for session tokens | Zero config, works immediately | XSS-stealable; must be replaced before any real data | Never — use httpOnly cookies from the start |
| `AllowedOrigins: []string{"*"}` in CORS config | Unblocks development immediately | Blocks credentialed requests; security vulnerability | Never with credentials; only acceptable for fully public, unauthenticated read endpoints |
| Bundle multiple DDL changes in one migration file | Fewer files to manage | Dirty state on failure is hard to recover; rollback is destructive | Never — one logical change per file |
| Skip offline sync for MVP | Faster to ship | Crew in a food truck (spotty WiFi) cannot submit completions; core use case broken | Acceptable only if explicitly scoped out and documented |
| Hardcode Tailscale hostname in CORS config | Works immediately | Breaks when machine or tailnet changes | MVP acceptable — extract to env var before production |
| Use `window.fetch` without retry on 503 | Simple client code | Mutations silently fail when server is briefly unavailable | Acceptable for MVP; add retry with backoff before production |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Go `rs/cors` + cookies | `AllowedOrigins: []string{"*"}` with `AllowCredentials: true` | Library rejects this combination; use explicit origin list |
| Go `rs/cors` + OPTIONS preflight | Mounting CORS middleware after router — preflight hits the route handler first and returns 404 | Mount CORS middleware as the outermost handler, before the router |
| `golang-migrate` + Postgres | Running `migrate.Up()` and ignoring the error type — dirty state causes server to loop on startup | Check for `migrate.ErrDirty` specifically; log instructions for manual recovery |
| Tailscale Serve + Go | Go server bound to `127.0.0.1` — Tailscale proxy cannot reach it | Bind Go server to `0.0.0.0:PORT`; or rely on Tailscale Serve forwarding to localhost (Serve proxies to localhost by design) |
| iOS PWA + httpOnly cookies | Testing auth in Safari browser and assuming standalone mode behaves identically | Test auth in standalone mode (Add to Home Screen) on a physical iOS device |
| service worker + API calls | SW fetch handler caches `/api/` responses with cache-first strategy | Partition fetch strategy: network-first or network-only for all `/api/` URLs |
| Background Sync API + iOS | Registering sync events, testing in Chrome, shipping to iOS crew | iOS does not support Background Sync; use `online` event + `navigator.onLine` check as fallback |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| N+1 queries for checklist list endpoint | Checklist list loads slowly as template count grows | Use `JOIN` or batch fetch for template metadata; never query template per submission in a loop | At ~20 submissions |
| Full response history returned per checklist | "My Checklists" tab is slow to render | Paginate or limit response history to last 30 days; server-side filter | At ~500 field responses |
| Unindexed FK lookups on `user_id` in submissions/responses | Approval queue slow for managers | Add indexes on all FK columns at migration time, not retroactively | At ~100 submissions |
| Replaying entire IndexedDB offline queue on reconnect | Sync takes 30+ seconds after extended offline period | Batch offline writes into one `POST /api/v1/sync` call, not one request per field | At ~50 offline field answers |
| JWT verification on every request without caching | Auth overhead on each API call | Use opaque session tokens stored in Postgres (`sessions` table) with a short-lived cache; avoid stateless JWT for a small-crew app where session revocation matters | At ~10 concurrent users |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Session tokens in `localStorage` | XSS can exfiltrate tokens; any script injection compromises all crew sessions | httpOnly cookies only; no JS-readable token storage |
| Reflecting `Origin` header verbatim in CORS response | Equivalent to wildcard; any site can make credentialed requests | Validate origin against an environment-configured allowlist |
| No token expiry or session revocation | Stolen or shared device retains access indefinitely | Session tokens expire (30 days max); provide a `/api/v1/auth/logout` endpoint that deletes the session row |
| Weak invite token generation | Brute-forceable invite links allow unauthorized user creation | Use `crypto/rand` to generate 32-byte invite tokens (256 bits of entropy); expire after 48 hours |
| Photo uploads stored without access control | Crew photos (potentially food safety evidence) publicly accessible | Serve photo blobs through an authenticated endpoint, not as static public URLs |
| Migration files committed with plaintext credentials | DB password in source history | Use environment variables for all DB connection strings; never hardcode credentials in SQL files or Go source |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| No loading states when API calls replace mock data | Crew members see blank screens while fetch completes — app appears broken | Add `<div class="loading">Loading...</div>` state between tab switch and first API response |
| Optimistic UI without rollback | "Submitted" confirmation shown, then API fails — crew member thinks checklist is done when it is not | Show optimistic completion only for offline-queued items; for online submissions, wait for 200 OK before showing success |
| Login form that submits then loses the return URL | Crew member follows a deep link, gets redirected to login, after login lands on home screen instead of original destination | Store `returnUrl` in `sessionStorage` before redirecting to login; restore after auth |
| Silent offline mode | Crew member fills out checklist offline, submits, sees no indication the form is queued | Show "Saved offline — will sync when connected" banner on offline submission |
| Auth session expiry with no prompt | Session expires mid-shift; crew member submits checklist and gets a 401 — data is lost | Intercept 401 responses globally; pause the queue, prompt re-login, then replay |

---

## "Looks Done But Isn't" Checklist

- [ ] **CORS:** Test from a different origin (phone browser) with credentials — not just same-machine fetch with no credentials
- [ ] **Cookie auth on iOS standalone:** Install PWA to home screen on a physical iPhone; log in, close app, reopen — session must persist
- [ ] **SW API partition:** Open DevTools → Application → Cache Storage — no `/api/` URLs should appear as cached entries
- [ ] **Offline queue:** Put phone in airplane mode, complete a checklist, restore connection — submission must reach the server (check the Postgres `responses` table)
- [ ] **Migration dirty recovery:** Deliberately corrupt a migration (add a syntax error), run the server, verify the error message is actionable and the team knows the `migrate force` recovery step
- [ ] **Tailscale mobile access:** Load the PWA on a real mobile device (not desktop browser) via the Tailscale hostname — full auth flow must work
- [ ] **Duplicate sync prevention:** Submit an offline completion twice (simulate by replaying the IndexedDB queue manually) — the server must return 200 (idempotent), not a duplicate record
- [ ] **Session expiry handling:** Let a session token expire (or manually delete the sessions row); make an API call from the frontend — must prompt re-login, not show a blank error

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| CORS wildcard in production | LOW | Update `CORS_ALLOWED_ORIGINS` env var; redeploy; no schema changes |
| SW caching API responses | MEDIUM | Update SW fetch handler; bump cache version; all clients must update SW (can take hours if cached SW is sticky) — use `skipWaiting()` + `clients.claim()` to force faster propagation |
| localStorage tokens discovered after launch | HIGH | Migrate frontend to cookies; update every fetch call to add `credentials: 'include'`; force all users to log in again |
| Migration dirty state | LOW | SSH to server; run `migrate force <version>`; fix the SQL error; run `migrate up` again |
| iOS standalone cookie failure | MEDIUM | Switch to same-origin serving (proxy static files through Go or via Caddy/Tailscale Serve on same domain); re-test auth flow |
| Duplicate records from offline sync replay | MEDIUM | Add `ON CONFLICT (client_id) DO NOTHING` to responses insert; backfill dedup via `SELECT DISTINCT ON (client_id)` query to identify duplicates |
| Background Sync API not working on iOS | LOW | Replace `registration.sync.register()` with `window.addEventListener('online', replayQueue)` — no backend changes required |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| CORS misconfiguration (Pitfall 1) | Phase 1: API foundation | Credentialed fetch from mobile browser returns 200; DevTools shows correct CORS headers |
| SW caches API responses (Pitfall 2) | Phase 1: API foundation | No `/api/` URLs in Cache Storage DevTools panel; POST requests never return cached response |
| Offline sync conflicts (Pitfall 3) | Phase 2: Checklist persistence | Replaying an offline queue entry twice results in one DB row, not two |
| Token storage in localStorage (Pitfall 4) | Phase 1: Auth | No `localStorage.getItem('token')` or `sessionStorage` reads in auth-related code |
| iOS standalone cookie isolation (Pitfall 5) | Phase 1: Auth + Tailscale setup | Auth flow complete on physical iPhone in standalone mode |
| Migration dirty state (Pitfall 6) | Phase 1: Database schema | All migrations wrapped in BEGIN/COMMIT; recovery procedure documented |
| Tailscale mobile reach (Pitfall 7) | Phase 1: Infrastructure | Full app accessible on crew member's phone via Tailscale hostname with HTTPS |
| Background Sync iOS gap (Pitfall 8) | Phase 3: Offline mode | Offline queue syncs on reconnect on an iOS device (not just Chrome/Android) |
| CSRF with cookies (Pitfall 9) | Phase 1: Auth | `SameSite=Strict` confirmed on session cookie; no `application/x-www-form-urlencoded` POST endpoints |
| Migration dependency order (Pitfall 10) | Phase 1: Database schema | Migration 001 is `create_users`; all FK references point to lower-numbered migrations |

---

## Sources

- [rs/cors — AllowedOrigins + AllowCredentials security note (August 2024)](https://github.com/rs/cors)
- [rs/cors Go package docs — configuration options](https://pkg.go.dev/github.com/rs/cors)
- [MDN: Progressive web apps — Caching strategies](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Caching)
- [MDN: Offline and background operation — service worker patterns](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Offline_and_background_operation)
- [web.dev/learn/pwa/serving — fetch strategy partitioning](https://web.dev/learn/pwa/serving)
- [OWASP: Session Management Cheat Sheet — httpOnly cookie recommendation](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [netguru: Sharing cookies between PWA standalone and Safari on iOS](https://www.netguru.com/blog/how-to-share-session-cookie-or-state-between-pwa-in-standalone-mode-and-safari-on-ios)
- [Apple Developer Forums: iOS PWA standalone cookie isolation](https://developer.apple.com/forums/thread/125109)
- [golang-migrate: Dirty database version recovery](https://medium.com/@kazutaka.yoshinaga/how-to-resolve-dirty-status-of-database-migration-in-golang-7735fb3138da)
- [golang-migrate GitHub Issue #282 — force version after dirty state](https://github.com/golang-migrate/migrate/issues/282)
- [Tailscale Docs: Enabling HTTPS certificates](https://tailscale.com/docs/how-to/set-up-https-certificates)
- [Tailscale Docs: tailscale serve command](https://tailscale.com/kb/1242/tailscale-serve)
- [whatpwacando.today: Background Sync — browser support table (Safari: no)](https://whatpwacando.today/background-sync/)
- [Sachith Dassanayake: Offline sync & conflict resolution patterns (April 2026)](https://www.sachith.co.uk/offline-sync-conflict-resolution-patterns-crash-course-practical-guide-apr-8-2026/)
- [ObjectBox: Customizable conflict resolution for offline-first apps](https://objectbox.io/customizable-conflict-resolution-for-offline-first-apps/)
- [CyberChief: LocalStorage vs Cookies — JWT token storage security](https://www.cyberchief.ai/2023/05/secure-jwt-token-storage.html)
- [BetterStack: Database migrations in Go with golang-migrate](https://betterstack.com/community/guides/scaling-go/golang-migrate/)

---
*Pitfalls research for: adding Go + Postgres backend to existing vanilla JS PWA (v2.0 backend milestone)*
*Researched: 2026-04-15*
