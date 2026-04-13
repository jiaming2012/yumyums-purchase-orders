# Codebase Structure

**Analysis Date:** 2026-04-12

## Directory Layout

```
hq/                         # Project root — also the web root (served as-is)
├── index.html              # Shell: PWA home screen / tile launcher
├── login.html              # Login page (mock — no backend yet)
├── purchasing.html         # Purchasing tool page
├── users.html              # Users & permissions tool page
├── sw.js                   # Service worker (cache-first PWA)
├── ptr.js                  # Pull-to-refresh shared utility
├── manifest.json           # PWA manifest (name, icons, display mode)
├── icons/                  # PWA icon set (48px–512px PNGs)
├── docs/                   # Internal design docs
│   └── user-management-api.md
├── .do/                    # Digital Ocean deploy config
│   └── app.yaml
├── .planning/              # GSD planning artifacts (not deployed)
│   ├── codebase/           # Codebase analysis docs (this directory)
│   └── quick/              # Quick task planning docs
├── .claude/                # Claude agent worktree state (not deployed)
├── README.md               # Project overview, stack decisions, deploy guide
└── CLAUDE.md               # Project instructions for Claude
```

## Directory Purposes

**Root (`hq/`):**
- Purpose: Web root — everything here is served directly; no subdirectory routing
- Contains: All HTML pages, JS utilities, manifest, service worker
- Key files: `index.html` (shell), `sw.js` (service worker), `ptr.js` (shared utility)

**`icons/`:**
- Purpose: PWA icon set at all required sizes for Android/iOS home screen installation
- Contains: PNG icons from 48×48 to 512×512
- Key files: `icon-192x192.png` (used as `apple-touch-icon`), `icon-512x512.png`

**`docs/`:**
- Purpose: Internal design and API specification documents
- Contains: Markdown design docs
- Key files: `user-management-api.md`

**`.do/`:**
- Purpose: Digital Ocean App Platform deployment specification
- Contains: `app.yaml` — defines static site, GitHub repo, branch, auto-deploy setting
- Generated: No — hand-maintained
- Committed: Yes

**`.planning/`:**
- Purpose: GSD workflow planning artifacts — not deployed, not served
- Contains: Codebase analysis docs, per-task quick plans
- Generated: By GSD commands
- Committed: Yes (as planning record)

**`.claude/`:**
- Purpose: Claude agent worktree state for parallel agent execution
- Contains: Agent-specific worktree copies
- Generated: Yes — by Claude agent tooling
- Committed: Yes (currently)

## Key File Locations

**Entry Points:**
- `index.html`: PWA shell, home screen launcher, registers service worker
- `login.html`: Authentication entry point (mock — always fails currently)

**Tool Pages:**
- `purchasing.html`: Weekly order form, locked view, PO summary
- `users.html`: Team member list, invite/edit form, per-app access control

**PWA Infrastructure:**
- `sw.js`: Service worker — cache list must be updated when new pages are added
- `manifest.json`: PWA metadata — icons, theme color, display mode, start URL
- `ptr.js`: Pull-to-refresh gesture — shared across all pages

**Configuration:**
- `.do/app.yaml`: Digital Ocean App Platform spec
- `manifest.json`: PWA configuration

**Documentation:**
- `README.md`: Stack decisions, data model, service shape, deploy instructions
- `CLAUDE.md`: Project context and conventions for Claude

## Naming Conventions

**Files:**
- Tool pages: lowercase, one-word name, `.html` extension — `purchasing.html`, `users.html`
- Utilities: lowercase, descriptive abbreviation — `ptr.js` (pull-to-refresh), `sw.js` (service worker)
- Icons: `icon-{width}x{height}.png` — `icon-192x192.png`
- Docs: kebab-case markdown — `user-management-api.md`

**CSS Classes (within HTML files):**
- Semantic single-word or abbreviated: `.card`, `.hd`, `.row`, `.pill`, `.stp`, `.ft`
- State modifiers: `.on` (active tab), `.soon` (placeholder tile), `.show` (visible error)
- Component variants: `.pill-admin`, `.pill-manager`, `.pill-member`, `.pill-invited`, `.pill-config`
- Layout containers: `.app` (max-width wrapper), `.grid` (tile grid), `.tabs` (tab bar)

**JS Variables:**
- Constants (data arrays): SCREAMING_SNAKE_CASE — `USERS`, `CATS`, `APPS`, `DEFAULT_PERMS`, `USER_GRANTS`
- DOM helpers: short lowercase — `ul`, `lst`, `ind`
- State variables: camelCase — `editingUser`, `pulling`, `startY`
- Functions: camelCase verbs — `show()`, `buildAccess()`, `editUser()`, `clearEdit()`, `sendInvite()`, `togglePerm()`, `addGrant()`, `removeGrant()`

## Where to Add New Code

**New Tool Page:**
1. Create `toolname.html` at project root
2. Copy the shared CSS variable block from any existing tool page (lines 13–16 of `purchasing.html`)
3. Include back link: `<a class="back" href="index.html">HQ</a>`
4. Register service worker and load ptr.js at bottom of body (copy from existing tool)
5. Add tile in `index.html` grid: change `<div class="tile soon">` to `<a class="tile active" href="toolname.html">`
6. Add `'./toolname.html'` to the `ASSETS` array in `sw.js` and bump the `CACHE` version string
7. Add entry to `APPS` array in `users.html` with matching slug for permissions management

**New CSS Styling:**
- Add to the inline `<style>` block within the specific page — there is no shared stylesheet
- CSS variables are defined in `:root` — use them for all colors, do not hardcode values
- Dark mode is handled automatically by the `:root` override in `@media(prefers-color-scheme:dark)`

**New Shared JS Behavior:**
- If needed across multiple pages: add a new `.js` file at project root, load via `<script src="...">` in each page, add to `ASSETS` in `sw.js`
- Currently only `ptr.js` follows this pattern

**Documentation:**
- Design docs: `docs/`
- Deploy notes: `README.md`
- Architecture analysis: `.planning/codebase/`

## Special Directories

**`.planning/codebase/`:**
- Purpose: Persistent codebase analysis for GSD commands
- Generated: By `/gsd:map-codebase`
- Committed: Yes

**`.do/`:**
- Purpose: Digital Ocean App Platform deployment spec
- Generated: No
- Committed: Yes — required for DO auto-deploy to work

**`icons/`:**
- Purpose: PWA icon set required by `manifest.json` and iOS `apple-touch-icon`
- Generated: No (manually added)
- Committed: Yes — icons must be present for PWA install to work

---

*Structure analysis: 2026-04-12*
