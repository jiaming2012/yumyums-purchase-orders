# External Integrations

**Analysis Date:** 2026-04-12

## APIs & External Services

**Current (live):**
- None - the app is a static mock with no API calls. All data is hardcoded in JavaScript within the HTML pages.

**Planned (documented in `docs/user-management-api.md` and `README.md`):**
- Internal Go REST API at `/api/v1` - auth, users, app permissions
- Email delivery service (unspecified) - for invite tokens and password reset emails

## Data Storage

**Databases:**
- None currently (mock data only; all state is in-memory JavaScript variables that reset on page reload)

**Planned:**
- PostgreSQL on existing Hetzner box
  - Connection: TBD (Go backend env config, not yet defined)
  - Client: Go standard `database/sql` or similar (not yet specified)
  - Schema: separate schema on shared Postgres instance (not a separate database)
  - Tables: `users`, `hq_apps`, `app_permissions`, `sessions`, `invite_tokens`, `master_items`, `requisitions`, `purchase_orders`, `po_lines`

**File Storage:**
- Local filesystem only (icons served as static assets)

**Caching:**
- Browser Cache API via service worker (`sw.js`) - shell caching of static HTML, JS, and manifest files

## Authentication & Identity

**Auth Provider:**
- None currently - `login.html` has a mock `signIn()` function that always shows an error; no real auth implemented
- Comment in `login.html`: `// In production this will POST to /api/v1/auth/login`

**Planned:**
- Custom bearer token auth (Go backend)
  - `POST /api/v1/auth/login` — email + password → bearer token
  - Token stored in `localStorage` as `hq_token`
  - Sessions table with `token_hash` (SHA-256), 8-hour expiry
  - Invite flow: `POST /api/v1/auth/accept-invite` with one-time token from email
  - Superadmin bootstrap from `config/superadmins.yaml` (not stored in DB)
  - Roles: `superadmin` (config-only), `admin`, `manager`, `team_member`

## Monitoring & Observability

**Error Tracking:**
- None detected

**Logs:**
- None (static site; no server-side logging)

## CI/CD & Deployment

**Hosting:**
- Digital Ocean App Platform (static site)
  - App ID: `0cc791e5-1626-4d7d-9a5b-f647f34c172e`
  - Live URL: `https://yumyums-purchase-orders-4iuwf.ondigitalocean.app`
  - Config: `.do/app.yaml`

**CI Pipeline:**
- GitHub auto-deploy: push to `main` branch of `jiaming2012/yumyums-purchase-orders` triggers automatic deploy on Digital Ocean App Platform
- No test pipeline

**Future planned hosting:**
- Hetzner box + Caddy at `order.yumyums.com` (same box as trading stack)

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None currently
- Considered (v2 backlog per `README.md`): POST to Zoho Books/Inventory for PO submission

## Third-Party SaaS Evaluated But Not Adopted

- **Zoho Creator** - evaluated at ~$32-40/mo for 4-5 users; rejected due to cost, capability overlap with existing stack, and per-user caps on scheduled Deluge calls (per `README.md`)
- **Temporal** - explicitly not used for purchasing cron (per `README.md`); reserved for future ordering assistant agent layer that touches LLMs, external APIs, or human-in-loop workflows

## Environment Configuration

**Required env vars:**
- None currently (static frontend)

**Planned backend env vars (not yet defined):**
- Database connection string
- Email service credentials
- Session secret / token signing key
- Superadmin config path

**Secrets location:**
- Not yet established; planned backend will require secrets management

---

*Integration audit: 2026-04-12*
