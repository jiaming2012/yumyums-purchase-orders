# Phase 9: Foundation + Auth - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-14
**Phase:** 09-foundation-auth
**Areas discussed:** Email provider, Login flow details, Dev workflow, Seed data & bootstrap

---

## Email Provider

| Option | Description | Selected |
|--------|-------------|----------|
| Resend | Developer-friendly API, generous free tier (100 emails/day), Go SDK, simple DNS setup | ✓ |
| Postmark | Battle-tested transactional email, excellent deliverability, $15/mo minimum after trial | |
| net/smtp direct | Zero dependency — Go stdlib sends via Gmail/Zoho SMTP relay. Free but fragile | |
| Skip email for now | Generate invite links in server logs or CLI output. Wire email later | |

**User's choice:** Resend
**Notes:** Best fit for low-volume transactional email at food truck scale.

---

## Login Flow Details

### Post-login redirect

| Option | Description | Selected |
|--------|-------------|----------|
| Always index.html | Login always redirects to the HQ launcher grid | ✓ |
| Return to original page | Store return URL, redirect back after login | |

**User's choice:** Always index.html

### Session duration

**User's clarification:** "Can we assume that once a user is logged in, they will remain logged in indefinitely, until either they log out or the system kicks them out?"

**Decision:** Sessions live forever — no automatic expiry. End on explicit logout or admin revocation.

### Session revocation

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, from users.html | Admin can force-logout a crew member from the Users app | ✓ |
| Server-side only | Revoke via direct DB delete or CLI command | |

**User's choice:** Admin force-logout from users.html

### Failed login handling

| Option | Description | Selected |
|--------|-------------|----------|
| Simple error message only | Show 'Invalid credentials' on bad login. No lockout, no delay | ✓ |
| Lock after 5 failures | Temporarily lock the account after 5 failed attempts | |

**User's choice:** Simple error message only

### Unauthenticated page access

| Option | Description | Selected |
|--------|-------------|----------|
| Client-side redirect to login.html | Frontend JS checks for 401 on first API call and redirects | ✓ |
| Server-side redirect | Go server returns 302 for non-API requests without session cookie | |

**User's choice:** Client-side redirect

---

## Dev Workflow

### Frontend serving in dev

| Option | Description | Selected |
|--------|-------------|----------|
| Disk in dev, embed in prod | Build flag or env var switches between os.DirFS and embed.FS | ✓ |
| Always embed.FS | Rebuild Go binary on every frontend change | |
| Separate servers | Python/npx serves frontend, Go API separate. Requires CORS | |

**User's choice:** Disk in dev, embed in prod

### Postgres in dev

| Option | Description | Selected |
|--------|-------------|----------|
| Docker container | docker run postgres:16 with local volume | ✓ |
| Local install | Homebrew Postgres directly on macOS | |
| Docker Compose | Full docker-compose with Go + Postgres | |

**User's choice:** Docker container

---

## Seed Data & Bootstrap

### Superadmin creation

| Option | Description | Selected |
|--------|-------------|----------|
| Config file | superadmins.yaml lists email(s). Server ensures on startup | ✓ |
| CLI command | Run `hq create-admin` to seed first user | |
| First-run setup page | Setup wizard if no users exist | |

**User's choice:** Config file (matches existing API doc design)

### Seed data approach

| Option | Description | Selected |
|--------|-------------|----------|
| Schema only | Migrations create tables. Seed data separate | ✓ |
| Dev seed migration | Numbered migration inserts test data, gated by env var | |
| Makefile target | `make seed` runs a Go script for test data | |

**User's choice:** Schema only — clean separation

---

## Claude's Discretion

- Go project layout (internal/ structure)
- Makefile targets
- Health check endpoint design
- Session token generation approach
- Tailscale Serve configuration

## Deferred Ideas

None — discussion stayed within phase scope
