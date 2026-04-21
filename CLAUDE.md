## Project

**Yumyums HQ — Operations Console**

A mobile-first PWA operations console for a food truck business. One app shell with a launcher grid linking to independent workflow tools. Each tool is a standalone HTML page inside a shared PWA, designed for a small crew (1-5 people) to use on their phones.

**Core Value:** A workflow engine that lets the owner build checklist templates and have crew members fill them out on mobile — with accountability and smart conditions.

### Current Tools

| Tool | Status | Route |
|------|--------|-------|
| Operations | Complete (v2.0) | workflows.html |
| Inventory | Active (v2.0) | inventory.html |
| Onboarding | Active (v2.1) | onboarding.html |
| Users | Active (v2.0) | users.html |
| Purchasing | Mockup | purchasing.html |
| Login | Active (v2.0) | login.html |
| Payroll | Placeholder | — |
| Scheduling | Placeholder | — |
| Hiring | Placeholder | — |
| BI | Placeholder | — |

### Architecture

- **Shell:** `index.html` — launcher grid with emoji tiles, links to tool pages
- **Tools:** Each tool is a standalone HTML page with a back link to HQ
- **Workflows:** `workflows.html` — 3-tab layout (My Checklists / Approvals / Builder), ~1500 lines vanilla JS
- **PWA:** Workbox-generated service worker (`sw.js`) with content-hashed precaching
- **Auto-reload:** `ptr.js` listens for `controllerchange` to reload on new SW deploy
- **Manifest:** `manifest.json` — "Yumyums HQ", standalone display, portrait orientation
- **Styling:** Shared CSS variables with automatic dark mode, mobile-first (max-width 480px)
- **Inventory:** `inventory.html` — 5-tab layout (Purchases / Stock / Trends / Cost / Setup), receipt review pipeline, item catalog with groups/tags, stock level thresholds
- **Receipt pipeline:** Mercury banking → receipt download → DO Spaces upload → Claude Haiku parse → validate → pending review queue → manual confirm
- **Testing:** 170+ Playwright E2E tests across `tests/workflows.spec.js`, `tests/persistence.spec.js`, `tests/inventory.spec.js`, `tests/onboarding.spec.js`
- **Backend:** Go + Postgres, REST API at `/api/v1/workflow/*`, `/api/v1/inventory/*`, `/api/v1/auth/*`, `/api/v1/onboarding/*`, `/api/v1/users/*`
- **Data flow:** See `docs/data-flow-audit.md` for the full state persistence inventory

### workflows.html Key Concepts

- **State-first rendering:** Mutate JS state → call render function → DOM updates from state
- **Event delegation:** ONE click + ONE input listener per container div, routes via `data-action` attributes
- **Data from API:** All data loaded from Go backend (`/api/v1/workflow/*`), no mock data
- **SortableJS 1.15.7:** Only external dependency, loaded via CDN for drag-to-reorder

### inventory.html Key Concepts

- **5-tab layout:** Purchases (purchase events + pending review), Stock (levels + reorder suggestions), Trends (coming soon), Cost (coming soon), Setup (items + vendors management)
- **Receipt review pipeline:** Pending purchases from Mercury receipt worker → user reviews line items → links each to catalog item via fullscreen picker modal → confirms when total matches bank transaction
- **Item catalog:** Items are created from actual receipts (not pre-seeded). Each item belongs to a group (Proteins, Beverages, etc.). Groups have configurable stock level thresholds (low/high).
- **Auto-match:** When review form opens, line item names are matched case-insensitively against catalog. Matched items show no border; unlinked items show orange warning border.
- **Item selection persistence:** Selecting an item in the picker modal saves to `pending_purchases.items` JSONB via `PUT /purchases/pending-items` so selections survive page reloads.
- **Stock count overrides:** `stock_count_overrides` table stores manual quantity counts. Stock query uses `COALESCE(override, sum)`. Reason is required (preset chips: Counted shelf, Spoiled item, Damaged item).
- **Name normalization:** `normalizeItemName()` in Go uses `cases.Title(language.English)` for title case. Applied on confirm, item create, and vendor create. Frontend `titleCase()` mirrors this.
- **Merge:** Vendors and items can be merged (re-points all FKs, deletes source). Cannot merge into self.
- **Magic links:** Stock item detail → "View in Setup" navigates to Setup tab with item expanded. Reorder suggestion tap scrolls to and expands the stock item below.

### Workflows Data Persistence Rule

**Every user-entered value MUST follow this path — no exceptions.**

```
User action
  → Update FIELD_RESPONSES[fieldId] (optimistic UI)
  → autoSaveField(fieldId, value)
      → POST /saveResponse (persists to Postgres)
      → Update DRAFT_RESPONSES[] (in-memory cache)
  → On checklist open: hydrateFieldState(filterFieldIds)
      → Reads DRAFT_RESPONSES + MY_SUBMISSIONS.responses
      → Populates FIELD_RESPONSES + FAIL_NOTES
```

**When adding a new field type or user-entered state:**
1. The click/input handler MUST call `autoSaveField(fieldId, value)`
2. If the state has metadata (like fail notes), bundle it: `autoSaveField(fieldId, value)` checks `FAIL_NOTES[fieldId]` and sends `{_v: value, _fail_note: {...}}`
3. `hydrateFieldState` MUST unpack and restore it
4. Write a regression test: enter data → back to list → reopen → assert data is still there
5. See `docs/data-flow-audit.md` for the full state inventory

**7 persisted states:** checkbox, yes/no, text, temperature, sub-steps, fail note text, fail severity

**Required test for every new field type or data entry feature:**
```
test('FIELDTYPE survives back-to-list and reopen', async ({ page }) => {
  // 1. Create template with the field type
  // 2. Open checklist, enter data
  // 3. Wait for auto-save (1500ms)
  // 4. Click back to list
  // 5. Assert list shows correct progress (e.g., 1/1)
  // 6. Reopen the same checklist
  // 7. Assert the entered data is still there
  // 8. Assert other fields are NOT affected
});
```
Add this test to `tests/persistence.spec.js` under the "Draft response persistence" section. If this test is missing for a field type, the feature is not complete.

### Service Worker (Workbox)

- `sw.js` is generated by `node build-sw.js` (Workbox `generateSW`)
- Static assets: precached with content hashes — **no manual version bumps**
- API calls: network-first with offline JSON fallback
- Run `task sw` to rebuild after changing any HTML/JS files
- `task test` and `task deploy` auto-run `task sw` as a dependency

### Adding a New Tool

1. Create `toolname.html` with the shared CSS variables and a back link to `index.html`
2. Add a tile to the grid in `index.html` (change `tile soon` to `tile active` with an `<a>` tag)
3. Run `node build-sw.js` to regenerate the service worker with the new file
4. Run `task test` to verify no regressions

## Conventions

- Static HTML/CSS/JS — one build step: `node build-sw.js` (Workbox SW generation)
- Minified inline CSS in each page (shared variable block at top)
- Dark mode via CSS variables and `prefers-color-scheme` media query
- Double-tap zoom prevention via `dblclick` event listener
- Service worker generated by Workbox — cache-first for static, network-first for API
- **Run `task sw` or `node build-sw.js` after changing HTML/JS files**
- Event delegation in workflows.html (not inline onclick on dynamic elements)
- `SCREAMING_SNAKE_CASE` for constants, `camelCase` for functions
- Playwright E2E tests: `task test` (headless, auto-rebuilds SW + creates test DB)
- Tests block service workers (`serviceWorkers: 'block'` in Playwright config)
- **Persistence rule:** Every user-entered value → `autoSaveField` → `DRAFT_RESPONSES` → `hydrateFieldState` (see docs/data-flow-audit.md)
- **Required test:** Every new field type or data entry feature MUST have a back-and-reopen test in `tests/persistence.spec.js` — enter data → back → reopen → data still there. Feature is not complete without this test.
- **Bug fix protocol (approval phase):** When a bug is found during human verification, write the regression test FIRST — before applying the fix. The test must fail (proving it captures the bug), then apply the fix, then verify the test passes. Only run the new test(s) during iteration, not the full suite: `npx playwright test tests/<file>.spec.js -g "<test name>"`. This ensures the test actually guards against the regression, not just passing by coincidence.

<!-- GSD:project-start source:PROJECT.md -->
## Project

**Yumyums HQ — Operations Console**

A mobile-first PWA operations console for a food truck business. One app shell with a launcher grid that links to independent workflow tools — purchasing, user management, and a workflow/checklist engine inspired by Lumiform. Each tool is a standalone HTML page inside a shared PWA, designed for a small crew (1-5 people) to use on their phones.

**Core Value:** A workflow engine that lets the owner build checklist templates and have crew members fill them out on mobile — with accountability (who checked what) and smart conditions (day-of-week, fail triggers, skip logic).

### Constraints

- **Static only:** No build step, no framework — plain HTML, CSS, vanilla JS (matches existing convention)
- **PWA:** Must work as installed app on iOS and Android, offline-capable via service worker
- **Mobile-first:** All UI designed for 480px max-width, touch-optimized
- **Design consistency:** Must use existing CSS variables and dark mode support from other HQ pages
- **API-backed:** All data persisted in Postgres via Go backend — no mock data, no localStorage
<!-- GSD:project-end -->

<!-- GSD:stack-start source:codebase/STACK.md -->
## Technology Stack

## Languages
- HTML5 - All UI pages (`index.html`, `login.html`, `purchasing.html`, `users.html`)
- JavaScript (vanilla, ES6+) - Inline scripts in all HTML pages and `ptr.js`
- CSS (inline, custom properties) - All styling inline within HTML `<style>` blocks
- Go - Backend API server (per `README.md` and `docs/user-management-api.md`)
- SQL (PostgreSQL) - Data persistence (per `README.md`)
## Runtime
- Browser (no server-side runtime currently; pure static files)
- Service Worker runtime (`sw.js`) for PWA offline caching
- None - zero build tooling; no `package.json`, no lockfile
- Files are served directly as static assets
## Frameworks
- None - plain HTML, vanilla JavaScript, and CSS custom properties
- No frontend framework (no React, Vue, Angular, HTMX, etc. — HTMX is planned per README but not yet present)
- Web App Manifest (`manifest.json`) - install prompt and standalone display mode
- Service Worker (`sw.js`, cache version `yumyums-v5`) - shell-only offline caching
- None detected - no test files, no test framework
- None - no build step; deploy is direct static file serving
## Key Dependencies
- Cache API (`caches`) - service worker shell caching in `sw.js`
- `navigator.serviceWorker` - PWA registration in all HTML pages
- `navigator.standalone` / `matchMedia('display-mode: standalone')` - iOS standalone detection in `ptr.js`
## Configuration
- No environment variables (static frontend only)
- No `.env` file present
- None - no build config files
- Deployment config: `.do/app.yaml` (Digital Ocean App Platform spec)
- Cache key in `sw.js`: `const CACHE = 'yumyums-v5'` — must be bumped manually on deploy to invalidate cached assets
- Cached assets list: `['./','./index.html','./purchasing.html','./users.html','./login.html','./ptr.js','./manifest.json']`
## Platform Requirements
- Any HTTP server (HTTPS required for iOS PWA install prompt and service worker)
- No local tooling required — files can be opened directly or served with `python3 -m http.server`
- Static file host with HTTPS
- Currently: Digital Ocean App Platform (static site), auto-deploy on push to `main` branch of GitHub repo `jiaming2012/yumyums-purchase-orders`
- Live URL: `https://yumyums-purchase-orders-4iuwf.ondigitalocean.app`
- Planned production: Hetzner box with Caddy reverse proxy at `order.yumyums.com`
## Planned Backend Stack (not yet built)
- **Language:** Go
- **Database:** PostgreSQL (separate schema on existing Hetzner box)
- **Reverse proxy:** Caddy (automatic Let's Encrypt HTTPS)
- **Scheduling:** Go stdlib `cron` / `time.AfterFunc` (not Temporal)
- **Auth:** Bearer token sessions, password hash in DB, invite token flow
- **API base:** `/api/v1` (REST, JSON)
- **Frontend upgrade:** Plain HTML + HTMX (no build step retained)
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

## Overview
## Naming Patterns
- All lowercase, no separators: `index.html`, `purchasing.html`, `users.html`, `login.html`, `sw.js`, `ptr.js`
- HTML pages named after their feature/module (one page = one feature)
- Utility JS named by function: `ptr.js` (pull-to-refresh), `sw.js` (service worker)
- Short, abbreviated names preferred: `.hd` (header), `.nm` (name), `.em` (email), `.mut` (muted), `.brd` (border), `.txt` (text), `.bg` (background)
- BEM-adjacent for variants: `.pill`, `.pill-admin`, `.pill-manager`, `.pill-member`, `.pill-invited`, `.pill-config`
- State modifier using `.on` for active tab button: `button.on`
- Layout helpers: `.card`, `.row`, `.hd`, `.ft`, `.grid`, `.tabs`
- `SCREAMING_SNAKE_CASE` for module-level data constants: `CATS`, `USERS`, `APPS`, `DEFAULT_PERMS`, `USER_GRANTS`, `CACHE`, `ASSETS`
- camelCase for functions: `pillClass()`, `pillText()`, `buildAccess()`, `togglePerm()`, `addGrant()`, `removeGrant()`, `editUser()`, `clearEdit()`, `sendInvite()`, `resetPw()`, `deleteUser()`, `show()`
- Single-letter locals inside loops and lambdas: `u` (user), `d` (div), `r` (row), `n` (number), `e` (event), `k` (cache key), `h` (element), `c` (cache)
- Semantic kebab-like IDs: `user-list`, `edit-title`, `extern-notice`, `edit-fields`, `back-hint`, `btn-invite`, `btn-reset`, `btn-delete`, `f-email`, `f-role`
- Dynamic IDs use prefix + slug pattern: `s1`, `s2`, `s3` (sections), `t1`, `t2`, `t3` (tabs), `pick-${app.slug}`, `access-${app.slug}`
## Code Style
- No formatter configured. Inline styles and scripts are heavily minified/compacted.
- CSS rules are written single-line with no spaces: `:root{--bg:#f5f5f3;--card:#fff;...}`
- JS inside `<script>` tags is compact — no blank lines, short variable names, arrow functions
- Standalone `.js` files (e.g., `ptr.js`) use slightly more readable formatting with newlines
## Import Organization
## HTML Structure Pattern
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
- HTML section markers used in `users.html`:
- Production intention noted in comments: `// In production this will POST to /api/v1/auth/login`
## Function Design
- Functions are short and focused on a single DOM action
- Functions named after their action: `show()`, `editUser()`, `clearEdit()`, `sendInvite()`, `buildAccess()`
- `buildAccess()` is the most complex function — rebuilds the entire access tab DOM from scratch on each state change (no diffing)
- Parameters are positional, minimal: `show(n)`, `togglePerm(slug, role, val)`, `addGrant(slug)`, `removeGrant(slug, uid)`
## PWA Boilerplate (repeated on every page)
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

## Pattern Overview
- One installable PWA icon that acts as a launcher for multiple operational tools
- Each tool is a fully self-contained HTML page — no shared JS module system, no framework
- All data is currently hardcoded mock state in inline `<script>` blocks; no backend exists yet
- Single service worker provides cache-first offline coverage across all pages
- No build step, no bundler, no package manager
## Layers
- Purpose: Home screen that links to each tool; the "one icon on the phone" entry point
- Location: `index.html`
- Contains: 2×3 tile grid with emoji icons, active links for live tools, "Soon" placeholders for future tools
- Depends on: `sw.js` (for PWA install), `ptr.js` (pull-to-refresh), `manifest.json`
- Used by: Users navigating from home screen
- Purpose: Each operational workflow, standalone and independently navigable
- Location: `purchasing.html`, `users.html`, `login.html`
- Contains: All CSS (inlined), all JS (inline `<script>` block), HTML structure for that workflow
- Depends on: `ptr.js` (shared behavior), `sw.js` (cached), `manifest.json`
- Used by: Shell tiles linking to them; users bookmarking directly
- Purpose: Behavior shared across every page
- Location: `ptr.js`
- Contains: Pull-to-refresh gesture handler (iOS standalone PWA only — Android has it natively)
- Depends on: Nothing
- Used by: All pages via `<script src="ptr.js">`
- Purpose: PWA offline support, cache management
- Location: `sw.js`
- Contains: Install (pre-cache all assets), activate (purge stale caches), fetch (cache-first with offline fallback to `index.html`)
- Depends on: Cache version string `CACHE = 'yumyums-v5'` — must be bumped manually on deploys
- Used by: Browser; registered from every page
## Data Flow
- No framework state management; plain JS variables in page scope
- `editingUser` variable in `users.html` tracks which user is being edited
- `DEFAULT_PERMS` and `USER_GRANTS` objects in `users.html` are mutated directly on toggle/add/remove actions
## Key Abstractions
- Purpose: Implicit interface every tool page follows
- Pattern: Each page must include: shared CSS variable block, `<meta viewport>` with zoom disabled, back link to `index.html`, `sw.js` registration, `ptr.js` script tag
- Examples: `purchasing.html`, `users.html`, `login.html`
- Purpose: Navigation unit on the home screen
- Two variants: `<a class="tile active" href="...">` for live tools, `<div class="tile soon">` with `<span class="badge">Soon</span>` for placeholders
- Location: `index.html` lines 39–78
- Purpose: In-page navigation between views within a tool
- Pattern: `<div class="tabs">` containing `<button>` elements, `show(n)` JS function controls visibility of `id="s1"`, `id="s2"`, `id="s3"` sections
- Used in: `purchasing.html` (Form / Locked / PO), `users.html` (Users / Edit / Access)
- Purpose: Central list of all tools and their slugs used for permission management
- Location: `users.html` inline `<script>`, `APPS` array
- Pattern: `{slug, name, icon}` — slug matches the tool route name; permissions keyed off slug
## Entry Points
- Location: `index.html`
- Triggers: User launches installed PWA, or navigates to root URL
- Responsibilities: Renders tile grid, registers service worker, links to tools
- Location: `login.html`
- Triggers: Not yet gated — login page exists but no auth middleware enforces it
- Responsibilities: Collect email + password; planned to POST to `/api/v1/auth/login`
- Location: `purchasing.html`
- Triggers: Tapping the Purchasing tile from `index.html`
- Responsibilities: Weekly order form (stepper inputs per catalog item), locked view, PO summary by vendor
- Location: `users.html`
- Triggers: Tapping the Users tile from `index.html`
- Responsibilities: Team member list, invite/edit user form, per-app role and individual grant management
- Location: `sw.js`
- Triggers: Registered by every page on load; browser-managed lifecycle
- Responsibilities: Pre-caches all HTML assets on install, serves cache-first on fetch, falls back to `index.html` on network error
## Error Handling
- Login form: renders error div with class `show` on bad input (always fires in current mock)
- Purchasing stepper: min-clamp at 0 (cannot go below zero)
- Access grants: guards against duplicate adds with `if(!USER_GRANTS[slug].includes(uid))`
- No network error handling (planned for production handlers)
## Cross-Cutting Concerns
<!-- GSD:architecture-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd:quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd:debug` for investigation and bug fixing
- `/gsd:execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->

<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd:profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
