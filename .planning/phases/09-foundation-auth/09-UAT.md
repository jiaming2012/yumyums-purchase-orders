---
status: testing
phase: 09-foundation-auth
source: 09-01-SUMMARY.md, 09-02-SUMMARY.md, 09-03-SUMMARY.md, 09-04-SUMMARY.md
started: 2026-04-15T12:00:00Z
updated: 2026-04-15T12:00:00Z
---

## Current Test

number: 1
name: Cold Start Smoke Test
expected: |
  Kill any running server. Run `cd backend && make db-reset && make dev` (or db-start + seed + dev). Server boots without errors, migrations complete, seed runs, and `curl http://localhost:8080/api/v1/health` returns `{"status":"ok"}`.
awaiting: user response

## Tests

### 1. Cold Start Smoke Test
expected: Kill any running server. Run `cd backend && make db-reset && make dev` (or db-start + seed + dev). Server boots without errors, migrations complete, seed runs, and `curl http://localhost:8080/api/v1/health` returns `{"status":"ok"}`.
result: [pending]

### 2. Login Page Loads
expected: Navigate to http://localhost:8080/login.html. You see an email and password form with a sign-in button. No console errors.
result: [pending]

### 3. Login with Valid Credentials
expected: After seeding (`make seed`), enter jamal@yumyums.com / test123 on the login page and submit. You are redirected to index.html (the launcher grid). No error message appears.
result: [pending]

### 4. Auth Guard on Index Page
expected: Open a private/incognito browser window (no existing session cookie). Navigate directly to http://localhost:8080/index.html. You are automatically redirected to login.html because you're not authenticated.
result: [pending]

### 5. Launcher Grid After Login
expected: After a successful login, index.html shows the tile grid with emoji icons for each tool. The page does not redirect you away — your session cookie is valid.
result: [pending]

### 6. Login with Wrong Password
expected: On the login page, enter jamal@yumyums.com with an incorrect password (e.g., "wrong"). An error message appears on the form. You are NOT redirected.
result: [pending]

### 7. Logout Flow
expected: While logged in, open browser dev tools Network tab. Send POST to /api/v1/auth/logout (or trigger logout if a button exists). The session cookie is cleared. Navigating to index.html now redirects to login.html.
result: [pending]

### 8. /me Endpoint Returns User Profile
expected: While logged in, visit http://localhost:8080/api/v1/me in browser or curl with the session cookie. Returns JSON with your user info (email, role, name).
result: [pending]

### 9. Service Worker API Partition
expected: With the PWA running, open dev tools Application > Service Workers. API calls (/api/*) should use network-first strategy — they are NOT served from cache. Static assets (HTML, JS) are served cache-first.
result: [pending]

## Summary

total: 9
passed: 0
issues: 0
pending: 9
skipped: 0
blocked: 0

## Gaps

[none yet]
