---
phase: 09-foundation-auth
plan: 03
subsystem: auth
tags: [go, bcrypt, httponly-cookie, session-auth, chi, pgx, postgres]

# Dependency graph
requires:
  - phase: 09-02
    provides: pgxpool.Pool, db.Migrate, config.LoadSuperadmins, users+sessions schema in Postgres
provides:
  - auth.GenerateToken (crypto/rand 32 bytes, SHA-256 hash)
  - auth.HashToken (SHA-256 for session lookup by middleware)
  - auth.VerifyPassword / auth.HashPassword (bcrypt wrappers)
  - auth.CreateSession / auth.DeleteSessionByHash / auth.LookupSession (session CRUD)
  - auth.AuthenticateUser (email+password verify with superadmin overlay)
  - auth.UpsertSuperadmins (on-startup upsert to users table)
  - auth.Middleware (chi middleware: validates hq_session cookie, attaches User to context)
  - auth.LoginHandler (POST /api/v1/auth/login -> httpOnly cookie)
  - auth.LogoutHandler (POST /api/v1/auth/logout -> delete session, clear cookie)
  - me.MeHandler (GET /api/v1/me -> user profile JSON)
  - me.MeAppsHandler (GET /api/v1/me/apps -> accessible apps list)
affects: [10-workflows-api, 11-onboarding-users-admin, 12-inventory-photos, 09-04]

# Tech tracking
tech-stack:
  added:
    - golang.org/x/crypto (bcrypt for password hashing/verification)
  patterns:
    - Opaque session token: crypto/rand 32 bytes hex-encoded for cookie, SHA-256 hash stored in DB
    - superadmin in-memory overlay: superadmins.yaml users get IsSuperadmin=true + Role="superadmin" override at lookup time
    - chi protected route group: r.Group with r.Use(auth.Middleware(...)) applied to logout+me+me/apps
    - httpOnly + Secure + SameSite=Strict cookies with no MaxAge (indefinite sessions per D-03)
    - Idempotent logout: returns 204 even if no cookie present

key-files:
  created:
    - backend/internal/auth/service.go
    - backend/internal/auth/middleware.go
    - backend/internal/auth/handler.go
    - backend/internal/me/handler.go
  modified:
    - backend/cmd/server/main.go
    - backend/go.mod
    - backend/go.sum

key-decisions:
  - "Token body exclusion: LoginHandler returns user JSON only — raw token is in Set-Cookie, never in response body"
  - "LookupSession uses nil,nil (not error) for missing session — auth failures are not server errors"
  - "UpsertSuperadmins called before HTTP server starts — ensures superadmin rows exist before any login attempt"
  - "MeAppsHandler returns [] (not null) when user has no app grants — empty array is safer for frontend"

patterns-established:
  - "Pattern: auth.UserFromContext(r.Context()) is the standard way for handlers behind Middleware to get current user"
  - "Pattern: pgx no-rows check uses err.Error() == 'no rows in result set' (pgx v5 does not export ErrNoRows as typed error from QueryRow)"

requirements-completed: [AUTH-01, AUTH-02, AUTH-03]

# Metrics
duration: 8min
completed: 2026-04-15
---

# Phase 09 Plan 03: Auth Service + Session Middleware Summary

**httpOnly cookie session auth with bcrypt, opaque tokens, chi middleware, login/logout handlers, and /me endpoint — superadmin bootstrapped from YAML on startup**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-04-15T03:00:00Z
- **Completed:** 2026-04-15T03:07:30Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments

- Auth service with 9 functions: token generation (crypto/rand), SHA-256 hashing, bcrypt verify/hash, session CRUD, user authentication with superadmin overlay, startup upsert
- Chi middleware validates hq_session cookie on every protected request, attaches typed User to context
- Login handler returns user JSON + httpOnly Secure SameSite=Strict cookie (token never in body); logout is idempotent 204
- /me and /me/apps endpoints behind auth middleware; superadmins see all apps, others filtered by role+grant

## Task Commits

1. **Task 1: Auth service (service.go)** - `e6e2c0b` (feat)
2. **Task 2: Middleware + handlers + route wiring** - `c754e59` (feat)

## Files Created/Modified

- `backend/internal/auth/service.go` - 9 auth functions: GenerateToken, HashToken, VerifyPassword, HashPassword, CreateSession, DeleteSessionByHash, LookupSession, AuthenticateUser, UpsertSuperadmins
- `backend/internal/auth/middleware.go` - Middleware factory + UserFromContext helper
- `backend/internal/auth/handler.go` - LoginHandler + LogoutHandler
- `backend/internal/me/handler.go` - MeHandler + MeAppsHandler
- `backend/cmd/server/main.go` - Added auth/me imports, UpsertSuperadmins call, chi route group with protected subroutes
- `backend/go.mod` / `backend/go.sum` - Added golang.org/x/crypto

## Decisions Made

- Token excluded from response body: plan noted docs/user-management-api.md had a `token` field but architecture decision overrides — cookies only (XSS safe, iOS standalone compatible)
- Superadmin privilege is purely in-memory overlay: LookupSession and AuthenticateUser check the in-memory map and set IsSuperadmin=true + Role="superadmin" without touching the DB role column

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. Build and vet passed cleanly on first attempt.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Auth layer complete. POST /api/v1/auth/login, POST /auth/logout, GET /me, GET /me/apps all wired and compiling.
- Plan 09-04 (login.html wiring + Tailscale deploy + real phone test) can proceed immediately.
- Requires a running Postgres instance to test end-to-end (see Makefile `make db-up`).

---
*Phase: 09-foundation-auth*
*Completed: 2026-04-15*
