---
quick_task: 260412-sb3
subsystem: hq-pwa
tags: [user-management, auth, mockup, backend-design, pwa]
tech-stack:
  added: []
  patterns: [css-variables, dark-mode, tab-switching, toggle-switches, chip-ui]
key-files:
  created:
    - docs/user-management-api.md
    - login.html
    - users.html
  modified:
    - index.html
    - sw.js
    - /Users/jamal/projects/yumyums/hq/CLAUDE.md
decisions:
  - Superadmins managed via config/superadmins.yaml, not stored in users table
  - app_permissions uses XOR constraint (role OR user_id, never both)
  - invite_tokens table tracks email invite acceptance flow
  - SW cache bumped to yumyums-v4 for new pages
metrics:
  duration: ~20 minutes
  completed: 2026-04-12
  tasks_completed: 4
  files_changed: 6
---

# Quick Task 260412-sb3: Implement User Management App for HQ Summary

**One-liner:** Static mockup of 3-tab user management tool (Users/Edit/Access) with backend API design doc targeting Go + Postgres implementation.

---

## Tasks Completed

| # | Task | Commit | Files |
|---|------|--------|-------|
| 1 | Backend design doc | c54cb5a | docs/user-management-api.md |
| 2 | login.html auth screen | 6929ea2 | login.html |
| 3 | users.html 3-tab mockup | 50adc42 | users.html |
| 4 | Wire tile + cache update | 0cb0bba | index.html, sw.js |

---

## What Was Built

### docs/user-management-api.md
Complete backend design document covering:
- 5 tables: `users`, `hq_apps`, `app_permissions`, `sessions`, `invite_tokens`
- `config/superadmins.yaml` format — superadmins bootstrapped on startup, not in DB
- REST API contracts: auth (login/logout), users CRUD, invite, reset-password, accept-invite
- `GET /api/v1/me/apps` — drives PWA tile filtering when backend goes live
- Invite acceptance flow (token-based email link, single-use)
- Error codes table
- PWA integration plan (localStorage token, offline cache fallback)

### login.html
- Centered card layout, vertically centered in viewport
- "Yumyums HQ" wordmark + "Operations console" tagline
- Email + password fields with Sign In button
- Error state via JS toggle (`.error.show` class)
- Exact CSS variable and dark mode match to purchasing.html

### users.html
- **Tab 1 (Users):** 6 mock users rendered from JS array with CONFIG/INVITED/role pills. Clicking non-config rows switches to Tab 2 pre-filled.
- **Tab 2 (Edit):** Dual-mode form — add/invite mode (blank form, Send Invite button) and edit mode (read-only email, role dropdown, Reset Password, Delete User buttons). Superadmin rows jump to edit mode showing extern-notice instead of form.
- **Tab 3 (Access):** One card per app (6 apps) with CSS toggle switches per role (admin/manager/team_member) and individual user grant chips with remove button. Chip removal triggers re-render.

### index.html + sw.js
- Users tile added to grid: locked emoji, "Users" title, "Team and permissions" desc, active (not Soon)
- SW cache bumped from yumyums-v3 to yumyums-v4
- users.html and login.html added to ASSETS array

---

## Deviations from Plan

None - plan executed exactly as written.

---

## Known Stubs

| File | Description |
|------|-------------|
| login.html | Sign In always shows error (mockup — no real auth backend). Wire to `POST /api/v1/auth/login` when backend exists. |
| users.html | Send Invite, Reset Password, Delete User use `alert()` (mockup). Wire to API endpoints from docs/user-management-api.md. |
| users.html | Access tab toggle state is in-memory JS only — not persisted. Wire to `PUT /api/v1/apps/:slug/permissions`. |
| index.html | Users tile links directly to users.html with no auth gate. When backend exists, add token check + redirect to login.html. |

These stubs are intentional — this is a mockup phase. The backend design doc defines exactly which endpoints will resolve each stub.

---

## Self-Check

Checking created files exist:

- FOUND: docs/user-management-api.md
- FOUND: login.html
- FOUND: users.html
- FOUND: c54cb5a (backend design doc)
- FOUND: 6929ea2 (login.html)
- FOUND: 50adc42 (users.html)
- FOUND: 0cb0bba (index.html + sw.js)

## Self-Check: PASSED
