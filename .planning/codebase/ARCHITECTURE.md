# Architecture

**Analysis Date:** 2026-04-12

## Pattern Overview

**Overall:** Multi-page PWA shell (ops console / internal tools portal)

**Key Characteristics:**
- One installable PWA icon that acts as a launcher for multiple operational tools
- Each tool is a fully self-contained HTML page — no shared JS module system, no framework
- All data is currently hardcoded mock state in inline `<script>` blocks; no backend exists yet
- Single service worker provides cache-first offline coverage across all pages
- No build step, no bundler, no package manager

## Layers

**Shell / Launcher:**
- Purpose: Home screen that links to each tool; the "one icon on the phone" entry point
- Location: `index.html`
- Contains: 2×3 tile grid with emoji icons, active links for live tools, "Soon" placeholders for future tools
- Depends on: `sw.js` (for PWA install), `ptr.js` (pull-to-refresh), `manifest.json`
- Used by: Users navigating from home screen

**Tool Pages:**
- Purpose: Each operational workflow, standalone and independently navigable
- Location: `purchasing.html`, `users.html`, `login.html`
- Contains: All CSS (inlined), all JS (inline `<script>` block), HTML structure for that workflow
- Depends on: `ptr.js` (shared behavior), `sw.js` (cached), `manifest.json`
- Used by: Shell tiles linking to them; users bookmarking directly

**Shared Utilities:**
- Purpose: Behavior shared across every page
- Location: `ptr.js`
- Contains: Pull-to-refresh gesture handler (iOS standalone PWA only — Android has it natively)
- Depends on: Nothing
- Used by: All pages via `<script src="ptr.js">`

**Service Worker:**
- Purpose: PWA offline support, cache management
- Location: `sw.js`
- Contains: Install (pre-cache all assets), activate (purge stale caches), fetch (cache-first with offline fallback to `index.html`)
- Depends on: Cache version string `CACHE = 'yumyums-v5'` — must be bumped manually on deploys
- Used by: Browser; registered from every page

## Data Flow

**Current (mock-only):**

1. Page loads, inline `<script>` initializes hardcoded JS arrays (`USERS`, `CATS`, `APPS`, `DEFAULT_PERMS`, `USER_GRANTS`)
2. DOM is built by iterating those arrays with `document.createElement` and `innerHTML`
3. User interactions (stepper buttons, tab switches, form submits) mutate in-memory JS state only
4. Refreshing the page resets all state to initial mock values

**Planned (production):**

1. `login.html` POSTs credentials to `/api/v1/auth/login` — not yet implemented (currently always shows error)
2. Tool pages will fetch from Go backend handlers (e.g., `GET /requisition`, `POST /requisition`, `GET /po/:week`)
3. Backend: Go service + Postgres on Hetzner, behind Caddy reverse proxy at `order.yumyums.com`

**Tab Switching (all tool pages):**

1. `show(n)` called with tab index
2. All sibling sections `display:none`, target section `display:block`
3. Tab button classes updated to reflect active state (`class="on"`)

**State Management:**
- No framework state management; plain JS variables in page scope
- `editingUser` variable in `users.html` tracks which user is being edited
- `DEFAULT_PERMS` and `USER_GRANTS` objects in `users.html` are mutated directly on toggle/add/remove actions

## Key Abstractions

**Tool Page Contract:**
- Purpose: Implicit interface every tool page follows
- Pattern: Each page must include: shared CSS variable block, `<meta viewport>` with zoom disabled, back link to `index.html`, `sw.js` registration, `ptr.js` script tag
- Examples: `purchasing.html`, `users.html`, `login.html`

**Tile (shell grid item):**
- Purpose: Navigation unit on the home screen
- Two variants: `<a class="tile active" href="...">` for live tools, `<div class="tile soon">` with `<span class="badge">Soon</span>` for placeholders
- Location: `index.html` lines 39–78

**Tab Switcher:**
- Purpose: In-page navigation between views within a tool
- Pattern: `<div class="tabs">` containing `<button>` elements, `show(n)` JS function controls visibility of `id="s1"`, `id="s2"`, `id="s3"` sections
- Used in: `purchasing.html` (Form / Locked / PO), `users.html` (Users / Edit / Access)

**APPS Registry (users.html):**
- Purpose: Central list of all tools and their slugs used for permission management
- Location: `users.html` inline `<script>`, `APPS` array
- Pattern: `{slug, name, icon}` — slug matches the tool route name; permissions keyed off slug

## Entry Points

**PWA Home Screen:**
- Location: `index.html`
- Triggers: User launches installed PWA, or navigates to root URL
- Responsibilities: Renders tile grid, registers service worker, links to tools

**Login Screen:**
- Location: `login.html`
- Triggers: Not yet gated — login page exists but no auth middleware enforces it
- Responsibilities: Collect email + password; planned to POST to `/api/v1/auth/login`

**Purchasing Tool:**
- Location: `purchasing.html`
- Triggers: Tapping the Purchasing tile from `index.html`
- Responsibilities: Weekly order form (stepper inputs per catalog item), locked view, PO summary by vendor

**Users Tool:**
- Location: `users.html`
- Triggers: Tapping the Users tile from `index.html`
- Responsibilities: Team member list, invite/edit user form, per-app role and individual grant management

**Service Worker:**
- Location: `sw.js`
- Triggers: Registered by every page on load; browser-managed lifecycle
- Responsibilities: Pre-caches all HTML assets on install, serves cache-first on fetch, falls back to `index.html` on network error

## Error Handling

**Strategy:** Minimal — mock UI shows no real error states except login

**Patterns:**
- Login form: renders error div with class `show` on bad input (always fires in current mock)
- Purchasing stepper: min-clamp at 0 (cannot go below zero)
- Access grants: guards against duplicate adds with `if(!USER_GRANTS[slug].includes(uid))`
- No network error handling (planned for production handlers)

## Cross-Cutting Concerns

**Styling:** Shared CSS custom property block duplicated verbatim in every page's `<style>` tag — no external stylesheet
**Dark Mode:** Automatic via `@media(prefers-color-scheme:dark)` overriding CSS variables — no JS involved
**Zoom Prevention:** `dblclick` event listener (`e.preventDefault()`) + viewport meta `maximum-scale=1,user-scalable=no` on every page
**Pull-to-Refresh:** `ptr.js` loaded on every page; only activates in standalone PWA mode
**Cache Versioning:** Manual — developer must bump `CACHE` string in `sw.js` before each deploy

---

*Architecture analysis: 2026-04-12*
