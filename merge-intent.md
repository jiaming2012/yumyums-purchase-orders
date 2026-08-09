# Merge intent — Card 2 `sync-live-in-dev-app-proof` (run 20260810)

Branch: `wo-sync-live-in-dev-app-proof` off `overnight-20260810` @ 694bdd7
(which INCLUDES Card 1's merged substrate work).
Card: Activity 5, roadmap leg 3 — prove the sync capability is **usable in the
app**: the REAL `workflows.html` (flag `hq_sync_read` ON, NO `page.route` stub)
surfaces one `/saveResponse`-written field via RxDB through the `/sync/*` proxy,
against the live persistent substrate + FDW pointing, in `#sync-one-row` →
`data-state="served"`. Red-first: the SAME served-asserting spec FAILS when the
relay/carrier is withheld.

## Shared files touched (outside this card's own OWNS footprint)

- **`night-crew.toml`** — `[e2e.seams]` **roll-call comment only**. I added the
  new promoted spec name to the machine-checked `#   selects:` roll-call line that
  `tests/repo-hygiene.spec.js` asserts, because the spec's filename contains the
  `sync` seam token (Playwright's positional filter reads the arg as a path regex,
  so any `tests/`-anchored spec whose name matches `sync` is selected by the
  Operations seam). **NO new KEY. NO new TOKEN.** A roll-call name added to an
  existing seam is explicitly not a PARK (card instructions + repo-hygiene
  precedent). The count in `tests/repo-hygiene.spec.js` is bumped in the SAME
  commit (the assertion couples them).
  🛑 Whether this fires at all depends on WHERE the promoted spec lands — see the
  gate-harness decision below. It landed as a STANDALONE harness (form a), so the
  promoted spec is NOT under `tests/` and does NOT match the seam glob. See the
  §"night-crew.toml — what actually changed" note below for the resolved outcome.

Everything else is inside this card's OWNS footprint:
- `.night-crew/qa/spike-supabase/sync-app-proof.sh` — NEW: the promoted red-first
  app-surface harness (drives the real `workflows.html` against the persistent
  substrate; served-asserting spec relay-UP → exit 0, relay-DOWN → exit non-zero).
- `.night-crew/qa/spike-supabase/app-proof/workflows-live.spec.js` — NEW:
  promoted from `browser-live/workflows-live.spec.js` (same assertions; env prefix
  renamed to the card's `HQ_APP_PROOF_*`).
- `.night-crew/qa/spike-supabase/app-proof/playwright.app-proof.config.js` — NEW:
  promoted config (no `webServer`; the harness owns bring-up).
- `Taskfile.yml` (root) — NEW `sync:app-proof` thin wrapper (same rule as the
  `spike:*` / `sync:dev:*` families: all logic in the script, gate on the SCRIPT's
  exit, never on `task` — B-163).

## What of Card 1 must survive any merge (explicit)

1. **The `sync:dev:*` task family** — `sync:dev:up` / `:status` / `:down` / `:env`
   / `sync:dev:proof` in the root `Taskfile.yml`. This card drives the live
   substrate they stand up.
2. **The 4× `HQ_SYNC_*` dev-target wiring** in `backend/Taskfile.yml` (`dev` /
   `dev:tailscale` / `dev:lan` + `:log` variants). The proxy door the app read
   walks through is opened by exactly these four vars.
3. Card 1's shipped SQL/relay/bring-up files
   (`sync-dev-up.sh`, `sync-dev-proof.sh`, `sql/persistent-dev-fdw-pointing.sql`,
   `spikec_relay.go`). This card consumes them read-only.

I edited none of Card 1's files. My root-Taskfile addition is an ADDITIVE stanza
(`sync:app-proof`) that touches no existing `sync:dev:*` target.

## The gate-harness decision (engineer-level — DECIDED, not parked)

**Chose (a): a STANDALONE spike-style harness gated on its own exit code** — the
`browser-live/` precedent, B-345-aligned.

Reasoning:
- The spec drives a **live external substrate** (Docker substrate + FDW repoint +
  ephemeral scratch HQ). It cannot run inside the standard `:5434` `task test`
  harness — the card and roadmap both say so.
- Spike F already proved this exact harness shape works end-to-end
  (`spike-f-browser-live.sh` exit 0). "Promote" = make it a **card-owned repo
  harness** with the card's red-first framing, not shoehorn a live-substrate spec
  into `tests/`.
- Form (b) (`tests/` self-skip) would run **green-by-skipping** in every normal
  `:5434` run — a vacuous green — and would add a real roll-call name to the
  `sync` seam that never exercises the live path there. Strictly worse.
- The exit-code IS the verdict (0 pass / 1 red / 2 could-not-run); `task
  sync:app-proof` is a thin wrapper, gated on the SCRIPT not on `task` (B-163).

## night-crew.toml — what actually changed

Because form (a) was chosen, the promoted spec lives under
`.night-crew/qa/spike-supabase/app-proof/`, NOT under `tests/`. The
`[e2e.seams]` roll-call in `night-crew.toml` and `tests/repo-hygiene.spec.js`
enumerate `tests/*.spec.js` ONLY. A standalone spec outside `tests/` does NOT
match the seam glob, so it adds NO roll-call name and forces NO count bump. The
honoring of the roadmap's "seam entry so the app-surface test is in a card's
footprint" is done as a **footprint-mapping comment** in `night-crew.toml` that
maps the promoted harness paths → this card, without a new key/token (which would
PARK). `tests/repo-hygiene.spec.js` is UNCHANGED (count stays 11).

## What must survive / what is safe to drop

- Survive: the promoted harness (`sync-app-proof.sh` + `app-proof/` spec+config)
  and the `sync:app-proof` wrapper — they are the card's deliverable (the repo's
  first app-surface red-first proof).
- Safe to drop: nothing in Card 1; nothing this card touches conflicts with any
  other card's OWNS footprint. The spike-f `browser-live/` artifacts are LEFT in
  place (the spike ledger cites them); the promotion is additive.

## Red-first

The headline gate. Filled after the harness ran — see the run evidence below.

- **Relay/carrier UP (armed):** the served-asserting spec drives the real
  `workflows.html?hq_sync_read=on` (NO stub) and `#sync-one-row` reaches
  `data-state="served"` carrying the `/saveResponse` sentinel within the bound →
  spec exit 0.
- **Relay/carrier DOWN (red-first):** the SAME served-asserting spec, carrier
  withheld — the app stays `data-state="waiting"`, never reaches `served` → spec
  exit non-zero.

The asymmetry (UP pass / DOWN fail) proves the assertion is non-vacuous. Gated on
the spec/script exit code, never on `task` (B-163).

### EVIDENCE — `sync-app-proof.sh` run `ap20260809153943`, exit **0** (GREEN)

Command (gated on the SCRIPT, never on `task`):

    ./.night-crew/qa/spike-supabase/sync-app-proof.sh > app-proof-run3.log 2>&1
    echo "SCRIPT_EXIT=$?"   # => SCRIPT_EXIT=0

Setup all green: substrate reconciled (rest :63264, realtime :63263, db :63239);
scratch HQ (project `sync-app-proof-hq`) on a Docker-assigned ephemeral port
(never 5432/5433/5434); FDW repointed at the scratch HQ via Card 1's SHIPPED
`sql/persistent-dev-fdw-pointing.sql`, resolving **7** rows through the bridge;
`GET /sync/rest/` → **200** (Card 1's 4 HQ_SYNC_* vars set, door open);
`POST /api/v1/workflow/saveResponse` → **204**; HQ draft row carried the sentinel.

- **RED-FIRST — relay/carrier DOWN → spec exit 1 (must FAIL):**
  The SAME served-asserting spec ran, the real `workflows.html?hq_sync_read=on`
  (NO `page.route` stub) opened replication — all FOUR collections fetched through
  the proxy at `[sync 200]` (checklist_templates, submission_responses with the
  exact `or=(submission_id.eq.<cid>, and(submission_id.is.null, field_id.in.(<F>)))`
  draft filter, checklist_submissions, submission_rejections) — and the app sat at
  `data-state="waiting"` (`note="replication live; waiting for the row…"`) for the
  full 20 000 ms (`2× opening, 22× waiting`), never reaching `served`.
  `toHaveAttribute('data-state','served')` → Received `"waiting"`. **Spec exit 1.**
  Non-vacuous: the app was demonstrably LIVE and querying, and still could not
  serve with no row carried.

- **ARMED — relay/carrier UP → spec exit 0 (THE VERDICT):**
  Carrier landed `appproof-ap20260809153943` into `submission_responses`. The SAME
  spec then reached `#sync-one-row` →
  `served: id=appproof-ap20260809153943 value="appproof-ap20260809153943-1786289983N"`
  — the sentinel written through `/saveResponse`, served out of `db.responses` in
  the real app — in **508 ms**. **1 passed. Spec exit 0.**

Restore VERIFIED: `submission_responses` id-set byte-identical to the pre-run
baseline; FDW `hq_pg` restored to `host.docker.internal:5434/hq_test_b2_fdw`;
scratch HQ torn down.

### Could-not-run triage (two rounds before GREEN — B-164 discipline)

The first two harness runs exited 1 but were **could-not-run masquerading as red**,
triaged as infra (never counted as a card RED):
1. Run 1: `npx playwright test` fell through to a FOREIGN `playwright`
   (`/Users/jamal/miniconda3/bin/playwright`, no `test` subcommand →
   `error: unknown command 'test'`) because this fresh worktree's
   `node_modules/.bin/playwright` symlink was missing (`npm ci` hit its internal
   "Exit handler never called" error and skipped bin-linking). BOTH the red-first
   and armed runs failed identically — a fake asymmetry. **Fix (in-footprint,
   one round):** the harness now resolves the Playwright CLI deterministically
   (`node node_modules/@playwright/test/cli.js`), asserts its `test` subcommand in
   preflight, and reports a clear could-not-run if a foreign playwright shadows it.
2. Run 2 (CLI fixed): both runs failed at `browserType.launch` — Playwright 1.59.1
   wanted Chromium headless-shell build **1217**, absent from the cache. Installed
   via `node node_modules/@playwright/test/cli.js install chromium
   chromium-headless-shell`. **Environment, not code.**

Run 3 (CLI + browser sound) reached the real verdict above: GREEN.
