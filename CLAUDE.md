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
| Purchasing | Active | purchasing.html |
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
- **Inventory:** `inventory.html` — 7-tab layout (Purchases / Stock / Menu / Recipes / Trends / Cost / Setup), receipt review pipeline, item catalog with groups/tags, stock level thresholds, recipe/BOM editing for per-menu-item COGS
- **Receipt pipeline:** Mercury banking → receipt download → DO Spaces upload → Claude Haiku parse → validate → pending review queue → manual confirm
- **Period summary endpoint (Phase 21):** GET /api/v1/inventory/period-summary returns COGS + completeness gate for sales-processor's weekly payroll. Auth via HQ_INVENTORY_SERVICE_TOKEN (Bearer); unset → 503. See docs/contracts/inventory-period-summary.md.
- **Menu-COGS endpoint (Phase 999.2):** GET /api/v1/inventory/menu-cogs?from=YYYY-MM-DD&to=YYYY-MM-DD returns per-menu-item COGS attribution (units_sold + ingredient_cost_per_unit + ingredient_cost_total) for sales-processor's weekly report. Optional `?breakdown=true` adds per-ingredient detail per menu item. Auth via the SAME HQ_INVENTORY_SERVICE_TOKEN (Bearer) Phase 21 uses; unset → 503. HQ is truth source for units_sold (joins recipes → menu_items → daily_menu_sales internally). No completeness gate — drift surfaces in-app via the Recipes-tab banner + weekly Cliq alert. See docs/contracts/inventory-menu-cogs.md.
- **Testing:** 170+ Playwright E2E tests across `tests/workflows.spec.js`, `tests/persistence.spec.js`, `tests/inventory.spec.js`, `tests/onboarding.spec.js`, `tests/recipes.spec.js`
- **Backend:** Go + Postgres, REST API at `/api/v1/workflow/*`, `/api/v1/inventory/*`, `/api/v1/auth/*`, `/api/v1/onboarding/*`, `/api/v1/users/*`
- **Data flow:** See `docs/data-flow-audit.md` for the full state persistence inventory

### workflows.html Key Concepts

- **State-first rendering:** Mutate JS state → call render function → DOM updates from state
- **Event delegation:** ONE click + ONE input listener per container div, routes via `data-action` attributes
- **Data from API:** All data loaded from Go backend (`/api/v1/workflow/*`), no mock data
- **SortableJS 1.15.7:** Only external dependency, loaded via CDN for drag-to-reorder

### inventory.html Key Concepts

- **7-tab layout:** Purchases (purchase events + pending review), Stock (levels + reorder suggestions), Menu (Toast menu items, read-only), Recipes (BOM editing — ingredient-first slider allocation for per-menu-item COGS), Trends (coming soon), Cost (coming soon), Setup (items + vendors management)
- **Receipt review pipeline:** Pending purchases from Mercury receipt worker → user reviews line items → links each to catalog item via fullscreen picker modal → confirms when total matches bank transaction
- **Item catalog:** Items are created from actual receipts (not pre-seeded). Each item belongs to a group (Proteins, Beverages, etc.). Groups have configurable stock level thresholds (low/high).
- **Auto-match:** When review form opens, line item names are matched case-insensitively against catalog. Matched items show no border; unlinked items show orange warning border.
- **Item selection persistence:** Selecting an item in the picker modal saves to `pending_purchases.items` JSONB via `PUT /purchases/pending-items` so selections survive page reloads.
- **Stock count overrides:** `stock_count_overrides` table stores manual quantity counts. Stock query uses `COALESCE(override, sum)`. Reason is required (preset chips: Counted shelf, Spoiled item, Damaged item).
- **Name normalization:** `normalizeItemName()` in Go uses `cases.Title(language.English)` for title case. Applied on confirm, item create, and vendor create. Frontend `titleCase()` mirrors this.
- **Merge:** Vendors and items can be merged (re-points all FKs, deletes source). Cannot merge into self. Menu items in the Recipes tab can be merged the same way.
- **Magic links:** Stock item detail → "View in Setup" navigates to Setup tab with item expanded. Reorder suggestion tap scrolls to and expands the stock item below. Menu tab card tap → jumps to Recipes tab with that menu item's cost summary auto-selected.
- **Recipes (Phase 999.2):** Each ingredient (purchase_item) has a collapsed row showing last-week spend + unallocated %. Expand to edit `recipes.usage_pct` per menu item via 5%-snap sliders (autosave on release via PUT /api/v1/inventory/recipes/{id}). Server enforces sum-per-purchase_item ≤ 100 — 422 envelope `{error:"sum_exceeds_100",conflict_menu_item,conflict_pct}` triggers slider rollback + inline error. Weekly drift check (Monday 09:00 Chicago) writes to `drift_check_results` + fires Cliq message; banner reads from /api/v1/inventory/recipes/drift (200 `{}` when clean).

### Workflows Data Persistence Rule

**Every user-entered value MUST follow this path — no exceptions.**

```
User action
  → Update FIELD_RESPONSES[fieldId] (optimistic UI)
  → debouncedSaveField(fieldId, value)          // workflows.html:389 — 400ms debounce
      → submitOp('SET_FIELD', fieldId, 'field_response', {value, field_id})
          → POST /api/v1/workflow/ops           // sync.js:781, Lamport-stamped
          → workflowOpRouter → workflow.SaveResponseFunc (persists to Postgres)
      → Update the draftResponses store (DRAFT_RESPONSES is its live alias)
  → On checklist open: hydrateFieldState(filterFieldIds)
      → Reads DRAFT_RESPONSES + MY_SUBMISSIONS.responses
      → Populates FIELD_RESPONSES + FAIL_NOTES
```

🛑 **The function is `debouncedSaveField`. There is no `autoSaveField`** — that name was
carried by this file, the README, `docs/data-flow-audit.md`, two `sync-rxdb/` header
comments and one test assertion for months while being defined **nowhere in the tree**, and
the one place production code actually *called* it (the fail-photo path) threw a silent
`ReferenceError` and dropped the crew's evidence photo on every capture. B-65 / card A2,
run `20260804`. If you are about to write `autoSaveField`, you are about to ship a
`ReferenceError`.

Note also that the transport is **`POST /ops`, not `POST /saveResponse`**. The
`/saveResponse` endpoint still exists on the backend and the Go + Playwright suites still
drive it directly, but no frontend code posts to it — the op journal is the single write
channel (D-08).

**When adding a new field type or user-entered state:**
1. The click/input handler MUST call `debouncedSaveField(fieldId, value)` — pass the **answer**, nothing else
2. If the state has metadata (like fail notes or a correction photo), do NOT pass it as the value — put it in its store and let the bundler pick it up: `debouncedSaveField` reads `store.get('failNotes', fieldId)` and `CORRECTION_PHOTOS[fieldId]` itself and sends `{_v: value, _fail_note: {...}, _correction_photo: url}`
3. `hydrateFieldState` MUST unpack and restore it
4. Write a regression test: enter data → back to list → reopen → assert data is still there
5. See `docs/data-flow-audit.md` for the full state inventory

**9 persisted states:** checkbox, yes/no, text, temperature, sub-steps, fail note text, fail severity, fail photo, correction photo

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
- `task test` auto-runs `task sw` as a dependency. **`task prod:deploy` does NOT** — see "Deploying to prod" below
- `build-sw.js` also writes `version.json` (frontend semver from `package.json`) which the SW precaches
- **`sw.js` is a committed artifact.** `build-sw.js` reads **git HEAD**, not the working tree and not the index, so the manifest names only what a fresh clone can serve. Commit `sw.js` in the same change set as whatever you changed under it, or the change does not ship
- **`build-sw.js` exits non-zero when a precached file references something not precached** — `<script src>`, `import`, `import()`. The invariant is *reachability*, not completeness: a skipped file nobody references still exits 0. The failure message names both the referrer and the fix. Precache count is currently **31**; if it moves without an asset being deliberately added or removed, that is the silent drop (B-37) coming back

### Versioning & Deployment

**Versioning model** — two independently-bumpable semvers, one per side:

- `backend/internal/version/version.go` — `Backend` and `Frontend` constants (**authoritative**)
- `package.json` `"version"` — **must mirror** the `Frontend` constant exactly
- Build-time injection — `task backend:build` and `backend/Dockerfile` pass `-ldflags` to set `version.GitSHA` and `version.BuiltAt`
- Runtime surfacing — `GET /api/v1/health` returns `{status, backend_version, frontend_version, git_sha, built_at}`

**Bumping versions** — done by `.claude/skills/save-project/SKILL.md` (invoke `/save-project`). It detects which side(s) the diff touched and applies semver rules. Never bump `package.json` and `version.go` separately.

**Deploying to prod** — single command:

```
task prod:deploy    # prod clone → git fetch + reset --hard origin/main → docker compose build → up -d → health check
task prod:logs      # tail container logs
task prod:rollback  # restore the previous image (the :-rollback tag from the last deploy)
```

Prod runs in Docker on **this** box (Docker Desktop), building from a **separate clone pinned to `origin/main`** (`PROD_REPO`, default `/mnt/c/Users/jcole/projects/yumyums-purchase-orders`) — so prod only ever runs pushed code. The backend embeds the frontend; Cloudflare Tunnel routes `https://hq.yumyums.kitchen` to it. There is no separate frontend host — both ship as one image.

🛑 **`task prod:deploy` does NOT run `task sw`, and must not be "fixed" to.** `Taskfile.yml:178-221` does `git fetch origin main` → `git reset --hard origin/main` → `docker compose build`, so **the committed `sw.js` is what ships**. That is correct by construction, not an omission: `build-sw.js` reads **git HEAD**, and after the hard reset the prod clone's HEAD *is* the tree being built, so regenerating on the box could only reproduce the committed file. What this does mean is that **an `sw.js` you did not commit does not deploy** — the release flow is *commit a fresh `sw.js` on dev → merge dev→main → push → `task prod:deploy`*. (B-13; the doc claimed the dependency for months and the Taskfile never had it.)

**Verifying after deploy**:

```
task version       # diffs local source / dev server / prod /api/v1/health side-by-side
task health:prod   # raw /api/v1/health JSON from prod
```

If `task version` shows the local `Backend` / `Frontend` constants ahead of the prod values, the running container is stale — re-run `task prod:deploy`.

**Override deploy targets** via env vars: `PROD_REPO`, `PROD_COMPOSE`, `PROD_CONTAINER`, `PROD_IMAGE`, `PROD_PORT`, `PROD_URL`. Defaults are in the root `Taskfile.yml` `vars:` block.

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
- **Persistence rule:** Every user-entered value → `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops` → `DRAFT_RESPONSES` → `hydrateFieldState` (see docs/data-flow-audit.md). There is no `autoSaveField` — B-65
- **Required test:** Every new field type or data entry feature MUST have a back-and-reopen test in `tests/persistence.spec.js` — enter data → back → reopen → data still there. Feature is not complete without this test.
- **Bug fix protocol (approval phase):** When a bug is found during human verification, write the regression test FIRST — before applying the fix. The test must fail (proving it captures the bug), then apply the fix, then verify the test passes. Only run the new test(s) during iteration, not the full suite: `npx playwright test tests/<file>.spec.js -g "<test name>"`. This ensures the test actually guards against the regression, not just passing by coincidence.

### Definition of Done

Templates for all blocks below live in `docs/planning-templates.md`.

- **`done_when:` block required in every PLAN.md and UI-SPEC.md.** Every criterion names the observable behavior AND the check that proves it ("Empty state renders 'No X yet' when DB returns [] — load page with empty fixture, screenshot"). Banned words: "looks good," "feels right," "polished," "clean," "nice."
- **State Enumeration Table required in every UI-SPEC.md.** One table covering empty, loading, error, success, plus **at least 2 phase-specific edge rows** (long content, offline, 409 conflict, race — whichever apply). Each row names the trigger and the visual contract. The table is incomplete without the edge rows.
- **Self-verification ritual before declaring a UI phase done.** This environment is headless — verify via screenshots, not imagination:
  1. Write/extend a Playwright spec at `tests/states-<phase>.spec.js` that forces each State Enumeration Table row (fixture/mock/DB seed), navigates, and screenshots.
  2. Run it (`--update-snapshots` first run; without it thereafter, as a regression suite).
  3. Read the PNGs back with the Read tool (multimodal) and compare row-by-row against the visual contract.
  4. Report what was *observed* — not what was intended — in the phase SUMMARY.md, with screenshots referenced.
  5. If the dev server / DB / creds aren't available, say so explicitly and stop. Never declare done from code reading alone.
- **Mockup sign-off before UI code on phases introducing new components.** Commit the mockup (HTML or annotated screenshot) at `docs/mockups/<phase>.html` and wait for an explicit human "ok, build this" before touching production code. Note any deviation from the approved mockup in SUMMARY.md.
- **Verifier subagent gate between build and SUMMARY.md on UI phases.** Spawn one verifier subagent whose inputs are ONLY: the UI-SPEC.md, the `done_when:` block, the diff, and the self-verify screenshots — not the planning conversation or the implementer's reasoning. It outputs pass/fail per `done_when:` row plus issues beyond the contract. SUMMARY.md may not be written until every row passes or is explicitly waived (waiver + reason noted in SUMMARY.md, e.g. "requires live Mercury creds").

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
- Deployment config: `backend/Dockerfile` (multi-stage Go build embedding frontend assets)
- Service worker: generated by `node build-sw.js` (Workbox content-hashed precache, no manual cache key)
## Platform Requirements
- Any HTTP server (HTTPS required for iOS PWA install prompt and service worker)
- No local tooling required — files can be opened directly or served with `python3 -m http.server`
- Production: Go backend in Docker on Windows box, frontend embedded into the binary, served via Cloudflare Tunnel
- Live URL: `https://hq.yumyums.kitchen`
- Deploy: `task prod:deploy` (prod clone → `git reset --hard origin/main` → `docker compose build` → `up -d`; the **committed** `sw.js` ships — nothing regenerates it on the box)
## Planned Backend Stack (not yet built)
- **Language:** Go
- **Database:** PostgreSQL (separate schema on existing Hetzner box)
- **Reverse proxy:** Caddy (automatic Let's Encrypt HTTPS)
- **Scheduling:** Go stdlib `cron` / `time.AfterFunc` (not Temporal)
- **Auth:** Bearer token sessions, password hash in DB, invite token flow
- **API base:** `/api/v1` (REST, JSON)
- **Frontend upgrade:** Plain HTML + HTMX (no build step retained)

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

## Night-Crew Workflow

**This repo is a night-crew _target_ repo.** Planning and execution run through the
night-crew cycle.

Real project state lives in **`.night-crew/knowledge/`** — `roadmap.md`, `okrs.md`,
`BACKLOG.md`, `ledger.md`, `bugs.md`, `prds/`, `reference/`. That is the only planning
state; there is no second planning directory to reconcile against.

Durable reference docs that outlive any single cycle live in `docs/`:
`docs/contracts/` holds the sales-processor API contracts (`inventory-period-summary.md`,
`inventory-menu-cogs.md`) — those are consumed by an external system, so treat them as
the contract of record. `docs/codebase/` holds the stack/architecture/conventions notes
the sections above summarise.

### Entry points

Full cheatsheet: `.night-crew/knowledge/COMMANDS.md`, or the global `/nc-help`.
`/nc-status` answers "where are we, what next?" and is the right first move in a
fresh session.

| Stage | Command |
|---|---|
| Attended, evening | `/nc-okr-session` → `/nc-pm-session` → `/nc-pm-grill-back` → `/nc-slate-plan` |
| Unattended, overnight | the slate's launch prompt, pasted into a **fresh** session |
| Attended, morning | `/nc-morning-triage` |
| Milestone / ship | `/nc-milestone-close`, `/nc-release`, `/nc-scorecard` |

### Rules that bind any agent working here

- **There is no per-edit gate.** Ad-hoc fixes, doc updates and investigations do not
  need a ceremony command — branch off `dev`, commit normally. Slates govern *planned*
  overnight work, not every keystroke.
- **Overnight runs** branch from `dev` as `overnight-YYYYMMDD`. Inside a run: never
  push, never tag, never touch `main`.
- **Do not edit a run branch once its `HANDOFF.md` is written.** It is the artifact
  under triage and its SHAs get cited as evidence. Follow-up fixes go on a fresh
  branch off `dev`, or off the merge commit after triage.
- **Check `night-crew.toml` when you change a file.** It maps changed paths → the
  Playwright spec subset a card must run. A narrow or missing footprint entry is
  exactly how a regression escapes a green gate — this has happened.
- **Run artifacts** live in `.night-crew/runs/<date>-autonomous/`: `HANDOFF.md`,
  `DECISIONS-NEEDED.md`, `merge-intents/`, `timings.log`. Conflict logs live at
  `.night-crew/knowledge/reference/conflicts-<runid>.md`.
- **Decide role-level calls yourself.** PM / PjM / Engineer-level questions get
  decided and stated, not handed to the operator to bless. Escalate only genuine
  product forks.

### Run mechanics (non-obvious, costs time to rediscover)

- `export PATH="/usr/local/go/bin:$PATH"` before **any** Go or Playwright leg. The
  non-interactive shell does not carry Go; Playwright's `webServer` dies with
  `go: not found` / exit 127, which **looks like a test failure and is not**.
- Postgres is on **:5433**, not 5432.
- `go test ./... -p 1` — **`-p 1` is load-bearing.** Packages share one test DB and
  each `TestMain` truncates; without it six packages red on cross-package
  interference. Not a production defect.
- Set `DB_TEST_URL` or the Go suite **exits 0 while skipping every DB-coupled test** —
  `internal/workflow` runs zero tests and still prints `ok`. Always check counts, not
  just `ok`/`FAIL`.
