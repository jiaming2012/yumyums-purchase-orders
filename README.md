# Yumyums HQ — Operations Console

A mobile-first PWA operations console for the Yumyums food truck. One installable app shell
with a launcher grid linking to independent workflow tools, designed for a small crew (1–5
people) on their phones. Live at **https://hq.yumyums.kitchen**.

**Core value:** a workflow engine that lets the owner build checklist templates and have crew
fill them out on mobile — with accountability (who checked what, live across devices) and
smart conditions (day-of-week schedules, fail triggers, approval flows) — plus an inventory
system that turns bank receipts into per-menu-item food-cost intelligence.

## Tools

| Tool | Status | Route |
|------|--------|-------|
| Operations (workflows/checklists) | Complete (v2.0) | `workflows.html` |
| Inventory | Active (v2.0) | `inventory.html` |
| Onboarding | Active (v2.1) | `onboarding.html` |
| Users | Active (v2.0) | `users.html` |
| Purchasing | Active | `purchasing.html` |
| Login | Active (v2.0) | `login.html` |
| Payroll / Scheduling / Hiring / BI | Placeholders | — |

## Stack

- **Backend:** Go + Postgres, REST API at `/api/v1/*`, frontend embedded in the binary
- **Frontend:** static HTML/CSS/vanilla JS — no framework, one build step (`node build-sw.js`
  for the Workbox service worker). SortableJS is the only external dependency.
- **PWA:** Workbox-generated `sw.js` with content-hashed precaching; `ptr.js` auto-reloads
  clients on new deploys; dark mode via CSS variables; 480px mobile-first
- **Prod:** Docker on the Windows box (container `yumyums-prod`), Cloudflare Tunnel to
  `hq.yumyums.kitchen`; dev/prod share one Postgres cluster separated by schema
- **Tests:** 550+ Playwright E2E specs + Go unit tests

## Key subsystems

### Workflow engine (`workflows.html`)

3-tab layout (My Checklists / Approvals / Builder), state-first rendering (mutate JS state →
render → DOM), event delegation via `data-action` attributes. Checklist templates carry
sections, 7 field types, day-of-week schedules, role assignments, and approval flows.

**Cross-device live sync:** every mutation is an op (`SET_FIELD`, `SUBMIT_CHECKLIST`,
`APPROVE_ITEM`, …) in a Lamport-stamped ops journal (`ops` table), broadcast over WebSocket
(`/ws`) and replayed by `sync.js` on catch-up. A 32-cell convergence matrix
({viewer}×{editor}×{op-type}×{derived-view}) is E2E-tested.

**Persistence rule (non-negotiable):** every user-entered value follows
`autoSaveField(fieldId, value)` → `POST /saveResponse` → `DRAFT_RESPONSES` →
`hydrateFieldState` on reopen. Every new field type ships with a back-and-reopen regression
test in `tests/persistence.spec.js` — enter data → back to list → reopen → data still there.
The feature is not complete without that test. See `docs/data-flow-audit.md`.

### Inventory (`inventory.html`)

7-tab layout: Purchases / Stock / Menu / Recipes / **Trends** / **Cost** / Setup.

- **Receipt pipeline:** Mercury banking → receipt download → DO Spaces upload → Claude Haiku
  parse → validate → pending review queue → manual confirm. Items are cataloged from real
  receipts, not pre-seeded.
- **Recipes/BOM:** per-ingredient `usage_pct` sliders allocate purchase spend to menu items
  (server enforces sum ≤ 100); weekly drift check (Mon 09:00 Chicago) fires a Cliq alert and
  an in-app banner.
- **Trends tab:** weekly spend by item group over a 12-week window, COGS-allowlist filtered,
  inline SVG charts (no chart dependency).
- **Cost tab:** per-menu-item margin table (`gross − ingredient_cost`, food-cost %) with
  top/bottom movers.
- **Service endpoints:** `GET /api/v1/inventory/period-summary` and
  `GET /api/v1/inventory/menu-cogs` feed the sales-processor's weekly payroll/report, gated
  by `HQ_INVENTORY_SERVICE_TOKEN` (Bearer; unset → 503).

### Access control — grants are a data boundary

The Users tool issues per-app and per-tab grants (`app_permissions`). Since the
grant-enforcement-parity work, **every live app surface is server-enforced** via
`auth.RequirePermission` middleware — operations (incl. `/ws`), inventory (+ recipes,
menu-items), purchasing, onboarding (+ videos), users. Revoking a grant removes data access,
not just UI. Rules:

- No grant → surface hidden in the client AND its endpoints return
  `403 {"error":"forbidden","missing_grant":"<slug>"}`. Fail-closed on DB error.
- `tests/grant-enforcement-parity.spec.js` is a standing parity guard: it derives the app
  list from `SeedHQApps` and asserts every slug is enforced somewhere (placeholders carry
  N/A-with-reason + a stale-N/A tripwire).
- One deliberate cross-app READ: `GET /inventory/items` passes with inventory **or**
  purchasing grant (the purchasing order form is built from the catalog; payload carries no
  cost fields). `/api/v1/photos/*` is the documented authenticated-only exception pending a
  key-binding design.
- **Tab-grant semantics (standing rule, ledger decision 45):** a tab with its own granular
  permission slug requires that explicit grant; tabs without one are covered by the app
  grant. (Reversed the earlier "app grant = all tabs" umbrella after operator play-testing.)

### Alerts — outbound delivery is opt-in

`internal/alerts` queues Zoho Cliq + SMTP notifications. Delivery only occurs when
`ALERTS_ENABLED=1`, which **only `docker-compose.prod.yml` sets** — so prod is the single
live sender and a dev server started from `backend/.env` (which holds the same live creds)
can never double-send (ledger decision 46). The value check is strict; anything but `"1"`
fails closed. Transactional email (invite/password reset) is deliberately outside the gate.

## Development methodology

### Overnight runs (night-crew)

Most build work ships through planned overnight autonomous runs, with attended human
checkpoints at both ends. State lives in `.night-crew/` (knowledge base, per-run records) —
that is the only planning state; durable reference docs live in `docs/`.

- **Cycle cadence:** attended OKR session → PM session drafts a PRD (traced to key results,
  grill-back resolves gray areas) → design gate (OpenSpec-style doc, operator sign-off
  before feature build) → `/nc-slate-plan` sizes WO-cards to a night budget → batch sign-off
  → `/nc-run` executes overnight → `/nc-morning-triage` reviews, merges, resolves forks →
  `/nc-milestone-close` grades the cycle's OKRs at the boundary.
- **Per-card discipline:** each card runs in its own worktree with a fresh implementer
  subagent, then a **separate fresh adversarial reviewer ("G6")** with its own binary and
  database, which attacks the change (path tricks, method confusion, authz axes, mutation
  probes) before approving. Only the orchestrator merges.
- **Merge intents:** every card's first commit includes a merge-intent note (shared files
  touched / what must survive any merge / safe to drop) so 3am conflict resolution is
  reviewable. Every merge — clean or conflicted — is logged in `reference/conflicts-<runid>.md`.
- **Open questions fork to the operator:** anything ambiguous parks into the run's
  `DECISIONS-NEEDED.md` rather than being guessed at; morning triage resolves each fork and
  records it as a numbered decision in `ledger.md` (§T-nn). The ledger is the ADR record.
- **Timing is a standing output:** per-card wall-clock actuals accumulate in
  `reference/card-actuals.md` and feed the next slate's estimates.

### Testing & verification

- `task test` — full Playwright suite (headless, auto-rebuilds SW, creates test DB). Go and
  E2E suites use **separate databases** (`hq_test_go` / `hq_test_e2e`), proven
  concurrent-safe. Tests block service workers; `E2E_DISABLE_SCHEDULERS=1` and blanked
  Mercury/Anthropic/Zoho/SMTP creds keep suites from touching live services.
- **Red-first bug protocol:** when a bug is found, write the regression test FIRST, watch it
  fail (proving it captures the bug), then fix, then watch it pass. Run only the new tests
  during iteration.
- **No-retry gate:** suite health is attested with `--retries=0` on an isolated ephemeral
  stack (fresh Docker Postgres, fresh DB, own server port). A pass-on-retry is a flake to
  investigate, not a pass. Flakes get controlled reproduction (load/contention legs) and a
  named mechanism; "rare, mechanism known" is never laundered into "not flaky", and a fix
  that only greens a targeted subset is not evidence — only a full-suite leg is.
- **Definition of Done for UI phases:** `done_when:` blocks name observable behavior + the
  proving check; a State Enumeration Table covers empty/loading/error/success + edge rows;
  self-verification is screenshot-based (write a states spec, screenshot each row, read the
  PNGs back); a verifier subagent gates the summary. Never declare done from code reading.
- **Play-testing is part of the loop:** operator hands-on testing regularly finds cells no
  matrix enumerated; findings are reproduced attended, evidenced in
  `.night-crew/knowledge/reference/`, and backlogged with a product ruling queued.

### Versioning & deploy

Two independently-bumpable semvers, one per side — `backend/internal/version/version.go`
(`Backend` + `Frontend`, authoritative) and `package.json` (must mirror `Frontend` exactly).
`/save-project` detects which side the diff touched and applies semver rules; **any backend
or frontend change bumps its side** so `task version` can surface content drift. Build-time
`-ldflags` inject `GitSHA`/`BuiltAt`; `GET /api/v1/health` reports all of it.

```
task sw             # rebuild service worker after ANY html/js change
task test           # full E2E suite
task prod:deploy    # hard-sync prod clone to origin/main → docker build → restart → health-check
task version        # diff local source / dev server / prod side-by-side
task health:prod    # raw prod /api/v1/health
task prod:logs      # tail the prod container
```

Prod only ever runs pushed-to-main code; the deploy tags the previous image for one-command
rollback. Release promotion (dev → main) is an attended decision, never folded into an
overnight run — runs never push; the morning triage's attended review is what earns the push.

## Local dev

```
cd backend && go run ./cmd/server/     # serves API + frontend (STATIC_DIR=../)
node build-sw.js                       # regenerate sw.js after frontend edits
npx playwright test tests/<file>.spec.js -g "<name>"   # targeted test run
```

Start dev servers with alert/external creds blanked (mirror the `playwright.config.js`
webServer command) — `backend/.env` holds live credentials and `ALERTS_ENABLED` must stay
unset outside prod.

## Adding a new tool

1. Create `toolname.html` with the shared CSS variables and a back link to `index.html`.
2. Flip its tile in `index.html` from `tile soon` to `tile active`.
3. Gate its API surface with `auth.RequirePermission` and register its slug in `SeedHQApps`;
   the parity spec will fail until every slug is enforced or carries a reasoned N/A.
4. `node build-sw.js`, then `task test`.
