---
phase: 13-integration-fixes
plan: "01"
subsystem: backend+frontend
tags: [bug-fix, role-check, auth, onboarding, builder-tab]
dependency_graph:
  requires: []
  provides: [is_superadmin in /me response, Builder tab role check, delete template endpoint]
  affects: [workflows.html Builder tab, onboarding.html delete template, /api/v1/me, /api/v1/onboarding/deleteTemplate]
tech_stack:
  added: []
  patterns: [roles array check instead of scalar role property, DELETE endpoint for resource removal]
key_files:
  created: []
  modified:
    - backend/internal/me/handler.go
    - backend/internal/onboarding/handler.go
    - backend/cmd/server/main.go
    - workflows.html
    - onboarding.html
    - sw.js
decisions:
  - is_superadmin exposed in /me response (was already on User struct, just not serialized)
  - DeleteTemplateHandler uses isManagerOrAdmin guard (consistent with other onboarding endpoints)
  - Frontend api() call upgraded from PUT /updateTemplate/{id}/delete to DELETE /deleteTemplate/{id}
metrics:
  duration_minutes: 8
  completed_date: "2026-04-19"
  tasks_completed: 2
  files_modified: 6
---

# Phase 13 Plan 01: Integration Fixes — Superadmin, Builder Tab, Delete Template Summary

**One-liner:** Fixed 3 integration breaks: /me missing is_superadmin, Builder tab scalar role check, delete template calling wrong HTTP method/route.

## What Was Done

### Task 1: Fix /me response and Builder tab role check (commit c316f56)

**Problem 1 — `/me` missing `is_superadmin`:** `MeHandler()` in `backend/internal/me/handler.go` serialized the user struct fields manually but omitted `is_superadmin`. The field existed on `auth.User.IsSuperadmin` but was not included in the response map.

**Fix:** Added `"is_superadmin": user.IsSuperadmin` to the JSON map in `MeHandler()`.

**Problem 2 — Scalar role check in `renderBuilder()`:** `workflows.html` line 568 checked `CURRENT_USER.role === 'admin'` (singular scalar property) instead of the array `CURRENT_USER.roles`. The backend always returns `roles` (array), never `role` (scalar). This caused all users to see `ADMIN_RESTRICTED` in the Builder tab.

**Fix:** Replaced the broken scalar check with:
```js
var isAdmin = CURRENT_USER && ((CURRENT_USER.roles||[]).includes('admin') || (CURRENT_USER.roles||[]).includes('superadmin') || CURRENT_USER.is_superadmin);
```
This matches the pattern already used elsewhere in `workflows.html` and handles the `is_superadmin` shortcut.

**onboarding.html:** Lines 344-345 were already correct — they use `CURRENT_USER.roles` (array) and `CURRENT_USER.is_superadmin`. No changes needed; the backend fix (adding `is_superadmin` to `/me`) is sufficient for onboarding sign-off to work.

### Task 2: Add DELETE /onboarding/deleteTemplate/{id} endpoint (commit 18add7f)

**Problem:** `onboarding.html` called `api('PUT', '/api/v1/onboarding/updateTemplate/'+tpl.id+'/delete', {})` — a route that didn't exist, causing 404s when deleting templates.

**Fix — Backend:** Added `DeleteTemplateHandler` to `backend/internal/onboarding/handler.go`:
- Requires authenticated user with manager/admin privileges (`isManagerOrAdmin`)
- Reads `{id}` from chi URL param
- Executes `DELETE FROM ob_templates WHERE id = $1` (CASCADE cleans sections/items)
- Returns `{"ok": true}` on success

**Fix — Route registration:** Added `r.Delete("/deleteTemplate/{id}", onboarding.DeleteTemplateHandler(pool))` in `backend/cmd/server/main.go` onboarding route group.

**Fix — Frontend:** Changed `onboarding.html` line 1166 from `api('PUT', '/api/v1/onboarding/updateTemplate/'+tpl.id+'/delete', {})` to `api('DELETE', '/api/v1/onboarding/deleteTemplate/'+tpl.id)`.

## Verification

- `grep -c "CURRENT_USER.role ===" workflows.html` → 0 (no remaining scalar checks)
- `grep "is_superadmin" backend/internal/me/handler.go` → matches `"is_superadmin": user.IsSuperadmin`
- `grep "deleteTemplate" backend/cmd/server/main.go` → matches `r.Delete("/deleteTemplate/{id}")`
- `grep -c "updateTemplate.*delete" onboarding.html` → 0 (old broken call removed)
- `cd backend && go build ./cmd/server/` → exits 0

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | c316f56 | fix(13-01): add is_superadmin to /me response and fix Builder tab role check |
| 2 | 18add7f | fix(13-01): add DELETE /onboarding/deleteTemplate/{id} endpoint and fix frontend call |

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all changes wire real backend behavior.

## Self-Check: PASSED

- `/Users/jamal/projects/yumyums/hq/backend/internal/me/handler.go` — verified contains `is_superadmin`
- `/Users/jamal/projects/yumyums/hq/backend/internal/onboarding/handler.go` — verified contains `DeleteTemplateHandler`
- `/Users/jamal/projects/yumyums/hq/backend/cmd/server/main.go` — verified contains `r.Delete("/deleteTemplate/{id}"`
- `/Users/jamal/projects/yumyums/hq/workflows.html` — verified contains `(CURRENT_USER.roles||[]).includes('admin')`
- `/Users/jamal/projects/yumyums/hq/onboarding.html` — verified contains `api('DELETE', '/api/v1/onboarding/deleteTemplate/'`
- Commits c316f56 and 18add7f confirmed in git log
