# Stack Research

**Domain:** Go + Postgres backend for Yumyums HQ PWA
**Researched:** 2026-04-15
**Scope:** NEW backend stack only. Existing frontend stack (vanilla JS, CSS variables, SortableJS, Chart.js, Playwright) is validated and not re-researched.
**Confidence:** HIGH

---

## Recommended Stack

### Core Backend

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.22+ | Backend language | Team choice; strong stdlib, easy deployment, fast compilation |
| chi | v5.2.5 | HTTP router | Lightweight, stdlib-compatible, middleware ecosystem, no framework lock-in |
| pgx/v5 | latest | Postgres driver | Native Postgres protocol, connection pooling, prepared statements, COPY support |
| sqlc | latest | Query codegen | Generates type-safe Go from SQL — schema stays in SQL, no ORM |
| goose | v3.27+ | DB migrations | SQL-file migrations, up/down support, embeddable in Go binary |
| crypto/rand + SHA-256 | stdlib | Auth tokens | Opaque bearer tokens, SHA-256 hash stored in sessions table |

### Infrastructure

| Technology | Purpose | Why |
|------------|---------|-----|
| Postgres 16 | Database | Already chosen in project constraints; LTS, resource-efficient |
| Caddy | Production reverse proxy | Automatic HTTPS via Let's Encrypt, simple config |
| Tailscale Serve | Dev deployment | `tailscale serve https:443 localhost:8080` — automatic HTTPS to tailnet, phone testing |
| embed.FS | Static file serving | Go serves both static PWA files and API from one binary — same-origin, no CORS |

### PWA Offline

| Technology | Purpose | Why |
|------------|---------|-----|
| IndexedDB | Client-side offline queue | Store pending submissions when offline |
| online event listener | Sync trigger | `window.addEventListener('online', replayQueue)` — works on iOS (Background Sync API does NOT) |
| Idempotency keys (UUID) | Dedup on sync | Each field answer gets a client UUID; server uses `ON CONFLICT (client_id) DO NOTHING` |

### What NOT to Use

| Technology | Reason |
|------------|--------|
| GORM / other ORMs | sqlc is simpler — SQL stays in SQL, no magic |
| JWT for sessions | Overkill for 1-5 users; opaque tokens are simpler and revocable |
| Background Sync API | Not supported on iOS Safari — must use online event fallback |
| Separate API origin | Breaks SW scope, requires CORS config, complicates offline queuing |
| Echo / Gin | chi is closer to stdlib, lighter dependency |

---

## Key Architectural Decisions

1. **Same-origin serving** — Go binary serves both `/api/v1/*` and static files via `embed.FS`. No CORS needed.
2. **httpOnly cookies for auth** — Not localStorage (XSS vulnerable), not IndexedDB (not sent automatically). `SameSite=Strict` works because same-origin.
3. **Presigned URLs for photos** — Photos uploaded direct to DigitalOcean Spaces; Go generates presigned PUT URLs. Server stays stateless on file data.
4. **SW fetch handler partitioned** — API calls (`/api/*`) use network-first strategy. Static files keep cache-first. Must be updated before first API call.

---

*Stack research: 2026-04-15*
