---
type: quick
task_id: 260418-0tz
subsystem: users, onboarding, auth, me
tags: [multi-role, migration, TEXT-array, Go, frontend, E2E]
dependency_graph:
  requires: [0022_ob_template_assignments migration]
  provides: [multi-role users, multi-role templates, roles[] API]
  affects: [users.html, onboarding.html, auth session, me endpoint]
tech_stack:
  added: [slices (Go stdlib)]
  patterns: [TEXT[] PostgreSQL, slices.Contains, ANY($1) SQL operator, checkbox group multi-select]
key_files:
  created:
    - backend/internal/db/migrations/0023_multi_role.sql
    - tests/multi-role.spec.js
  modified:
    - backend/internal/auth/service.go
    - backend/internal/auth/handler.go
    - backend/internal/users/db.go
    - backend/internal/users/handler.go
    - backend/internal/onboarding/db.go
    - backend/internal/onboarding/handler.go
    - backend/internal/onboarding/seed.go
    - backend/internal/me/handler.go
    - backend/internal/workflow/handler.go
    - users.html
    - onboarding.html
    - sw.js
decisions:
  - "Use slices.Contains from Go stdlib (Go 1.21+) for role membership checks — no external dependency"
  - "queryUserApps uses p.role = ANY($1) to match any of user's roles against app_permissions"
  - "Roles serializes as null JSON when empty (pgx nil slice) for ob_templates — all-roles semantic"
metrics:
  duration_minutes: 20
  tasks_completed: 3
  files_modified: 12
  completed_date: "2026-04-18"
---

# Quick Task 260418-0tz: Multi-Role Support for Users and Training Templates Summary

**One-liner:** Migrated users.role and ob_templates.role from single TEXT to TEXT[] arrays with full backend Go and frontend HTML updates for multi-role assignment.

## What Was Done

### Task 1: Database migration + all backend Go code updates

Created migration `0023_multi_role.sql` to alter both `users.role` (renamed to `roles`) and `ob_templates.role` (renamed to `roles`) to TEXT[] arrays. The users table gets a CHECK constraint ensuring at least one valid role; ob_templates roles column is nullable for "all roles" semantics.

Updated all backend Go code:
- `auth/service.go`: `User.Role string` → `User.Roles []string`; LookupSession and AuthenticateUser scan `u.roles`; superadmin sets `Roles = []string{"superadmin"}`; UpsertSuperadmins uses `ARRAY['admin']::TEXT[]`
- `auth/handler.go`: login response sends `"roles": user.Roles` (not `"role"`)
- `users/db.go`: UserRow, CreateUserInput, UpdateUserInput use `Roles []string`; all queries use `u.roles`
- `users/handler.go`: `isAdmin` uses `slices.Contains`; invite/update body structs use `Roles []string`
- `onboarding/db.go`: Template, AssignedTemplate, CreateTemplateInput use `Roles []string`; queries use `ot.roles`
- `onboarding/handler.go`: `isAdmin` and `isManagerOrAdmin` use `slices.Contains`
- `onboarding/seed.go`: `kitchenBasics` uses `Roles: nil`
- `me/handler.go`: `/me` response includes `"roles": user.Roles`; `queryUserApps` uses `p.role = ANY($1)`

### Task 2: Frontend multi-select role pickers

**users.html:**
- Replaced `<select id="f-role">` in both invite form and edit form with `<div id="f-roles" class="role-checks">` checkbox group (admin, manager, team_member)
- `submitInvite()` collects checked values and sends `roles: [...]` array
- `saveUser()` collects checked values and sends `roles: [...]` array
- `pillClass()` and `pillText()` updated to check `u.roles` array; multi-role display shows "Admin / Manager"
- Added `.role-checks` CSS

**onboarding.html:**
- Replaced `<select data-action="tpl-role-select">` with `.role-checks` checkbox group (line_cook, cashier, manager, admin)
- Input change handler uses `data-action="tpl-role-check"` to collect array
- Template list display shows joined roles: `line cook, cashier` or `All roles`
- Save/update API calls send `roles: tpl.roles || null`
- `isManager` check uses `CURRENT_USER.roles` array
- `createNewOBTemplate` updated to accept `roles` array

### Task 3: E2E tests for multi-role behavior

9 tests in `tests/multi-role.spec.js` — all passing:
1. Invite user with multiple roles returns roles as array
2. List users returns roles as array
3. Update user roles returns updated roles array
4. Onboarding template create with multiple roles
5. Onboarding template update roles from single to multi
6. Me endpoint returns roles array
7. users.html role checkboxes render for edit form
8. users.html invite form shows role checkboxes with team_member default-checked
9. onboarding.html builder shows role checkboxes

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] workflow/handler.go also referenced user.Role**
- **Found during:** Task 1 go build
- **Issue:** `internal/workflow/handler.go:117: user.Role undefined` — plan did not mention this file
- **Fix:** Added `slices` import and changed `isAdmin` to use `slices.Contains(user.Roles, "admin")`
- **Files modified:** `backend/internal/workflow/handler.go`
- **Commit:** 6c5a64b

**2. [Rule 1 - Bug] Test cleanup for template delete used non-existent endpoint**
- **Found during:** Task 3 test run
- **Issue:** No DELETE endpoint for onboarding templates; cleanup call returned HTML causing JSON parse error
- **Fix:** Removed cleanup call; test just verifies state without cleanup
- **Files modified:** `tests/multi-role.spec.js`
- **Commit:** bf6a7c2 (same as Task 3 commit after fix)

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| Task 1 | 6c5a64b | feat(260418-0tz): migrate role to roles TEXT[] with multi-role support |
| Task 2 | 7f5fef2 | feat(260418-0tz): multi-select role checkboxes in users.html and onboarding.html |
| Task 3 | bf6a7c2 | test(260418-0tz): add E2E tests for multi-role support |

## Known Stubs

None — all role data is wired from API; no placeholders or hardcoded values.

## Self-Check: PASSED
