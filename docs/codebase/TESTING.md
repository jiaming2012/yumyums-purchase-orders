# Testing Patterns

**Analysis Date:** 2026-04-12

## Test Framework

**Runner:** None
**Assertion Library:** None
**Config:** Not present

No test framework, test runner, or testing infrastructure exists in this project. There are no `package.json`, `jest.config.*`, `vitest.config.*`, or any other test configuration files. No `*.test.*` or `*.spec.*` files exist.

## Current State

**Testing approach:** Manual browser testing only.

The project is a vanilla HTML/CSS/JS PWA with static files and no build pipeline. All logic is inline script blocks inside HTML pages. No test infrastructure has been established.

## Mock/Prototype Data

All pages currently use hard-coded in-memory data arrays that serve as prototype fixtures. These are the closest thing to test data in the codebase:

**`users.html`** — `USERS` array (line 129):
```js
const USERS = [
  {id:1, name:'Jamal M.', email:'jamal@yumyums.com', role:'superadmin', status:'config'},
  {id:2, name:'Sarah K.', email:'sarah@yumyums.com', role:'admin', status:'active'},
  ...
];
```

**`purchasing.html`** — `CATS` array (line 80):
```js
const CATS = [
  ['Proteins', [['Salmon fillet','lb','par 6', 3], ...]],
  ['Produce', [['Lemons','case','par 2', 1], ...]],
  ...
];
```

**`login.html`** — explicit mock comment:
```js
// Mock: any non-empty email/password combination
// In production this will POST to /api/v1/auth/login
```

User-facing mock actions use `alert()` and `confirm()`:
```js
alert('Invite sent to ' + email + ' (mockup)');
alert('Password reset email sent to ' + editingUser.email + ' (mockup)');
```

## What to Test When Testing Is Added

**Unit-testable logic (currently inline in HTML):**
- `pillClass(role, status)` in `users.html` — pure function, deterministic output
- `pillText(user)` in `users.html` — pure function, deterministic output
- `show(n)` tab switching — DOM state assertion
- `addGrant(slug)` / `removeGrant(slug, uid)` — array mutation logic
- `togglePerm(slug, role, val)` — object mutation logic
- Stepper increment/decrement logic in `purchasing.html`

**Integration-testable flows:**
- Tab switching renders correct sections visible/hidden
- Edit user form populates fields from USERS data
- Access tab rebuilds correctly after grant add/remove

## Recommended Testing Approach (when introduced)

Given the no-build, vanilla JS nature of the project:

**Option A — Playwright or Cypress (E2E):**
- Load pages directly via `file://` or a local static server
- Test tab navigation, form interactions, and UI state
- No code changes required to existing files

**Option B — Extract and unit test pure functions:**
- Move helper functions (`pillClass`, `pillText`, `show`, etc.) to a shared `utils.js`
- Test with Vitest or Jest with jsdom
- Requires modest refactoring of inline scripts

**Option C — Keep manual testing:**
- Acceptable at current prototype scale
- Document manual test cases in `.planning/` if this is chosen

## Coverage

**Requirements:** None enforced
**Current coverage:** 0% automated

## Notes for Future Implementors

- No `node_modules`, no `package.json` — any test tooling introduction requires running `npm init` first
- The `sw.js` service worker caching logic (`CACHE = 'yumyums-v5'`) should be version-bumped whenever cached assets change — this is currently a manual convention
- The `ptr.js` pull-to-refresh script is a self-contained IIFE and could be unit tested in isolation without DOM setup

---

*Testing analysis: 2026-04-12*
