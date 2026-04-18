---
phase: 11-onboarding-users-admin
plan: 02
subsystem: backend/users
tags: [go, users, admin, invite, permissions, auth]
dependency_graph:
  requires: [11-01]
  provides: [user-admin-api, invite-flow, app-permissions-api]
  affects: [backend/cmd/server/main.go]
tech_stack:
  added: []
  patterns: [RPC-style handlers, atomic token claim, partial update pattern, chi URLParam]
key_files:
  created:
    - backend/internal/users/db.go
    - backend/internal/users/handler.go
  modified:
    - backend/cmd/server/main.go
decisions:
  - "Nickname collision query checks both u.nickname and derived display_name using COALESCE — prevents shadowing either form"
  - "SetAppPermissions uses a transaction with DELETE + re-INSERT for atomic full replacement"
  - "AcceptInviteHandler uses ClaimInviteToken atomic UPDATE RETURNING to prevent double-claim race"
  - "ErrTokenInvalid sentinel returned for all invalid/expired/used token cases — callers map to 400 token_expired"
  - "UpdateUser builds SET clause dynamically from non-nil pointer fields — no spurious overwrites on partial updates"
metrics:
  duration: 12
  completed_date: "2026-04-17"
  tasks_completed: 2
  files_modified: 3
---

# Phase 11 Plan 02: Users Admin API Summary

**One-liner:** Full user admin and invite/accept-invite Go backend with app permissions, wired into chi router.

## What Was Built

Complete `backend/internal/users/` package providing all user management HTTP handlers and SQL query functions. Routes registered in `main.go` for both unauthenticated invite flows and admin-only CRUD operations.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create users package with db.go and handler.go | 93ea9e2 | backend/internal/users/db.go, backend/internal/users/handler.go |
| 2 | Wire users and apps routes into main.go chi router | 6895c87 | backend/cmd/server/main.go |

## Handlers Built

| Handler | Route | Auth |
|---------|-------|------|
| ListUsersHandler | GET /api/v1/users | admin |
| InviteHandler | POST /api/v1/users/invite | admin |
| UpdateUserHandler | PATCH /api/v1/users/{id} | admin |
| ResetPasswordHandler | POST /api/v1/users/{id}/reset-password | admin |
| RevokeHandler | POST /api/v1/users/{id}/revoke | admin |
| DeleteUserHandler | DELETE /api/v1/users/{id} | admin |
| InviteInfoHandler | GET /api/v1/auth/invite-info | unauthenticated |
| AcceptInviteHandler | POST /api/v1/auth/accept-invite | unauthenticated |
| GetAppPermissionsHandler | GET /api/v1/apps/permissions | admin |
| SetAppPermissionsHandler | PUT /api/v1/apps/{slug}/permissions | admin |

## DB Functions Built

`ListUsers`, `GetUser`, `CreateInvitedUser`, `UpdateUser`, `CheckNicknameCollision`, `InsertInviteToken`, `ClaimInviteToken`, `ActivateUser`, `DeleteUser`, `GetAppPermissions`, `SetAppPermissions`, `GetInviteInfo`

## Key Decisions

1. **Nickname collision query** — checks both `u.nickname` and the derived `COALESCE(...)` display_name so neither form can be silently shadowed.
2. **SetAppPermissions transaction** — uses BEGIN/COMMIT with DELETE + re-INSERT for atomic full replacement; no partial state possible.
3. **Atomic token claim** — `ClaimInviteToken` uses `UPDATE ... WHERE used_at IS NULL AND expires_at > now() RETURNING user_id` to prevent double-claim races without application-level locking.
4. **ErrTokenInvalid sentinel** — single error type covers invalid/expired/used tokens; HTTP layer maps all to 400 `token_expired`.
5. **Dynamic partial updates** — `UpdateUser` builds SET clause only from non-nil pointer fields, preventing accidental overwrites on partial PATCH requests.

## Verification

- `go build ./internal/users/...` — PASSED
- `go build ./cmd/server/...` — PASSED
- `go build ./...` — PASSED

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None — all handlers connect to real SQL query functions. No mock data or placeholder values.

## Self-Check: PASSED

- backend/internal/users/db.go — FOUND
- backend/internal/users/handler.go — FOUND
- backend/cmd/server/main.go modified with /users and /apps routes — FOUND
- Commit 93ea9e2 — FOUND
- Commit 6895c87 — FOUND
