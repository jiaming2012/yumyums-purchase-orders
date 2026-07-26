# Coding Conventions

**Analysis Date:** 2026-04-12

## Overview

This is a vanilla HTML/CSS/JS PWA with no build tooling, no framework, and no transpilation. All code ships as-is to the browser. Conventions are implied by the existing files — no linter or formatter config is present.

## Naming Patterns

**Files:**
- All lowercase, no separators: `index.html`, `purchasing.html`, `users.html`, `login.html`, `sw.js`, `ptr.js`
- HTML pages named after their feature/module (one page = one feature)
- Utility JS named by function: `ptr.js` (pull-to-refresh), `sw.js` (service worker)

**CSS Classes:**
- Short, abbreviated names preferred: `.hd` (header), `.nm` (name), `.em` (email), `.mut` (muted), `.brd` (border), `.txt` (text), `.bg` (background)
- BEM-adjacent for variants: `.pill`, `.pill-admin`, `.pill-manager`, `.pill-member`, `.pill-invited`, `.pill-config`
- State modifier using `.on` for active tab button: `button.on`
- Layout helpers: `.card`, `.row`, `.hd`, `.ft`, `.grid`, `.tabs`

**JavaScript Variables:**
- `SCREAMING_SNAKE_CASE` for module-level data constants: `CATS`, `USERS`, `APPS`, `DEFAULT_PERMS`, `USER_GRANTS`, `CACHE`, `ASSETS`
- camelCase for functions: `pillClass()`, `pillText()`, `buildAccess()`, `togglePerm()`, `addGrant()`, `removeGrant()`, `editUser()`, `clearEdit()`, `sendInvite()`, `resetPw()`, `deleteUser()`, `show()`
- Single-letter locals inside loops and lambdas: `u` (user), `d` (div), `r` (row), `n` (number), `e` (event), `k` (cache key), `h` (element), `c` (cache)

**IDs:**
- Semantic kebab-like IDs: `user-list`, `edit-title`, `extern-notice`, `edit-fields`, `back-hint`, `btn-invite`, `btn-reset`, `btn-delete`, `f-email`, `f-role`
- Dynamic IDs use prefix + slug pattern: `s1`, `s2`, `s3` (sections), `t1`, `t2`, `t3` (tabs), `pick-${app.slug}`, `access-${app.slug}`

## Code Style

**Formatting:**
- No formatter configured. Inline styles and scripts are heavily minified/compacted.
- CSS rules are written single-line with no spaces: `:root{--bg:#f5f5f3;--card:#fff;...}`
- JS inside `<script>` tags is compact — no blank lines, short variable names, arrow functions
- Standalone `.js` files (e.g., `ptr.js`) use slightly more readable formatting with newlines

**No build tooling:** No webpack, vite, esbuild, rollup, or npm. Files are served as static assets directly.

**CSS custom properties (variables) for theming:**
All color values are defined as CSS variables on `:root` in every HTML file. The same variable set is repeated in each page (copy-paste pattern, not shared).
```css
:root{--bg:#f5f5f3;--card:#fff;--txt:#1a1a1a;--mut:#6b6b6b;--brd:rgba(0,0,0,0.08);
      --info-bg:#e6f1fb;--info-tx:#0c447c;--warn-bg:#faeeda;--warn-tx:#854f0b}
```
Dark mode overrides use `@media(prefers-color-scheme:dark)`.

## Import Organization

No imports. All code is inline `<script>` blocks or separate `.js` files loaded via `<script src="...">` at the bottom of `<body>`.

**Script loading order** (bottom of each page's `<body>`):
1. Inline `<script>` block with page logic
2. `<script src="ptr.js">` last

**Service worker registration** is always inside the inline script block:
```js
if('serviceWorker' in navigator) navigator.serviceWorker.register('sw.js');
```

**Double-tap prevention** is always the first line of the inline script block:
```js
document.addEventListener('dblclick', e => e.preventDefault());
```

## HTML Structure Pattern

Every page follows the same shell:
```html
<!DOCTYPE html>
<html lang="en">
<head>
  <!-- PWA meta tags (same set every page) -->
  <!-- <style> block with all page CSS inline -->
</head>
<body>
  <div class="app">
    <!-- Page content -->
  </div>
  <script>/* page logic */</script>
  <script src="ptr.js"></script>
</body>
</html>
```

The `.app` wrapper is always `max-width:480px; margin:0 auto` — mobile-first, centered.

## DOM Manipulation

- Direct DOM API: `document.getElementById()`, `document.createElement()`, `el.appendChild()`
- Template strings for innerHTML when building lists: `` el.innerHTML = `<div class="nm">${u.name}</div>` ``
- Event handlers attached as `.onclick =` properties or inline `onclick="fn()"` attributes
- No event delegation pattern — each element gets its own handler

## Data Patterns

- Static in-memory data defined as `const` arrays/objects at the top of each script block
- Data is hard-coded (mock/prototype state) — no fetch calls or API integration yet
- Array of tuples used for compact data: `['Salmon fillet','lb','par 6',3]`
- Array of objects for richer data: `{id:1,name:'Jamal M.',email:'jamal@yumyums.com',role:'superadmin',status:'config'}`

## UI State Management

- Tab visibility controlled by `show(n)` function pattern — iterates `[1,2,3]` and toggles `style.display`
- Form state held in module-level `let editingUser = null` variable
- No state library; all state is in-page JS variables and the DOM

## Error Handling

- No error handling for DOM operations
- Form validation is minimal: `if(!email) return;`
- Mock actions use `alert()` and `confirm()` built-ins
- Service worker fetch failures fall back to `caches.match('./index.html')`

## Comments

- Comments are sparse. Used only for section headers in longer files:
  ```js
  // Build user list
  // Build access tab
  // Default role toggles: admin has all, manager has some, team_member has basics
  // Individual user grants per app (user ids)
  ```
- HTML section markers used in `users.html`:
  ```html
  <!-- TAB 1: USERS LIST -->
  <!-- TAB 2: EDIT / ADD -->
  <!-- TAB 3: ACCESS -->
  ```
- Production intention noted in comments: `// In production this will POST to /api/v1/auth/login`

## Function Design

- Functions are short and focused on a single DOM action
- Functions named after their action: `show()`, `editUser()`, `clearEdit()`, `sendInvite()`, `buildAccess()`
- `buildAccess()` is the most complex function — rebuilds the entire access tab DOM from scratch on each state change (no diffing)
- Parameters are positional, minimal: `show(n)`, `togglePerm(slug, role, val)`, `addGrant(slug)`, `removeGrant(slug, uid)`

## PWA Boilerplate (repeated on every page)

Every HTML file includes this identical block in `<head>`:
```html
<meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no, viewport-fit=cover">
<meta name="theme-color" content="#0c447c">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="default">
<link rel="manifest" href="manifest.json">
```

And this block at the end of every inline `<script>`:
```js
document.addEventListener('dblclick', e => e.preventDefault());
if('serviceWorker' in navigator) navigator.serviceWorker.register('sw.js');
```

---

*Convention analysis: 2026-04-12*
