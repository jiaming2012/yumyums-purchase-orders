# Technology Stack

**Analysis Date:** 2026-04-12

## Languages

**Primary:**
- HTML5 - All UI pages (`index.html`, `login.html`, `purchasing.html`, `users.html`)
- JavaScript (vanilla, ES6+) - Inline scripts in all HTML pages and `ptr.js`
- CSS (inline, custom properties) - All styling inline within HTML `<style>` blocks

**Planned (not yet implemented):**
- Go - Backend API server (per `README.md` and `docs/user-management-api.md`)
- SQL (PostgreSQL) - Data persistence (per `README.md`)

## Runtime

**Environment:**
- Browser (no server-side runtime currently; pure static files)
- Service Worker runtime (`sw.js`) for PWA offline caching

**Package Manager:**
- None - zero build tooling; no `package.json`, no lockfile
- Files are served directly as static assets

## Frameworks

**Core:**
- None - plain HTML, vanilla JavaScript, and CSS custom properties
- No frontend framework (no React, Vue, Angular, HTMX, etc. — HTMX is planned per README but not yet present)

**PWA:**
- Web App Manifest (`manifest.json`) - install prompt and standalone display mode
- Service Worker (`sw.js`, cache version `yumyums-v5`) - shell-only offline caching

**Testing:**
- None detected - no test files, no test framework

**Build/Dev:**
- None - no build step; deploy is direct static file serving

## Key Dependencies

**Runtime (browser-native only):**
- Cache API (`caches`) - service worker shell caching in `sw.js`
- `navigator.serviceWorker` - PWA registration in all HTML pages
- `navigator.standalone` / `matchMedia('display-mode: standalone')` - iOS standalone detection in `ptr.js`

**No third-party JavaScript libraries** - no CDN imports, no npm packages

## Configuration

**Environment:**
- No environment variables (static frontend only)
- No `.env` file present

**Build:**
- None - no build config files
- Deployment config: `.do/app.yaml` (Digital Ocean App Platform spec)

**PWA Cache:**
- Cache key in `sw.js`: `const CACHE = 'yumyums-v5'` — must be bumped manually on deploy to invalidate cached assets
- Cached assets list: `['./','./index.html','./purchasing.html','./users.html','./login.html','./ptr.js','./manifest.json']`

## Platform Requirements

**Development:**
- Any HTTP server (HTTPS required for iOS PWA install prompt and service worker)
- No local tooling required — files can be opened directly or served with `python3 -m http.server`

**Production:**
- Static file host with HTTPS
- Currently: Digital Ocean App Platform (static site), auto-deploy on push to `main` branch of GitHub repo `jiaming2012/yumyums-purchase-orders`
- Live URL: `https://yumyums-purchase-orders-4iuwf.ondigitalocean.app`
- Planned production: Hetzner box with Caddy reverse proxy at `order.yumyums.com`

## Planned Backend Stack (not yet built)

Per `README.md` and `docs/user-management-api.md`:
- **Language:** Go
- **Database:** PostgreSQL (separate schema on existing Hetzner box)
- **Reverse proxy:** Caddy (automatic Let's Encrypt HTTPS)
- **Scheduling:** Go stdlib `cron` / `time.AfterFunc` (not Temporal)
- **Auth:** Bearer token sessions, password hash in DB, invite token flow
- **API base:** `/api/v1` (REST, JSON)
- **Frontend upgrade:** Plain HTML + HTMX (no build step retained)

---

*Stack analysis: 2026-04-12*
