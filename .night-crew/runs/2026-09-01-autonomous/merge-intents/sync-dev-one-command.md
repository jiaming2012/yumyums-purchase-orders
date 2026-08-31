# Merge-intent — `sync-dev-one-command` (Card 6, Track B, run 20260901)

Closes **B-171**. Wires convenience only: one command — `task sync:dev` — that
brings up the persistent sync data plane **and** launches the dev server together
for the operator's daily `dev-complete-attestation` use. Also discharges the
B-170 bare-`npx playwright` hardening in the one spike script this card touches.

The credential boundary is already in shape (dev targets carry the 4 `HQ_SYNC_*`
vars; the prod compose does not; the standing preference is capability-not-guards).
This card is **convenience, not policy** — it adds no new safety machinery and
carries no `:5433` bypass.

## Shared files touched

### `Taskfile.yml` (root) — SECTION: "ONE-COMMAND DEV — `task sync:dev`"
- **ADDED** one new top-level task `sync:dev` **immediately after `sync:dev:proof`**
  (line ~374), inside the existing "PERSISTENT SYNC DATA PLANE" family, with its own
  banner comment.
- Composes, in order:
  1. `task: sync:dev:up` — brings the substrate + relay + FDW up (reconcile). This
     is the SAME target that carries the B-164 :5433 refusal (in `sync-dev-up.sh`);
     it is invoked unchanged, so the refusal is preserved by construction.
  2. `task: backend:dev:tailscale` — launches the dev server. That target already
     resolves + exports the 4 `HQ_SYNC_*` vars itself (via its `sh:` env blocks
     reading the live substrate ports). We reuse it — we do NOT re-derive or
     duplicate the credential wiring.
- **No collision** with Card 3 (`toast-ingest-resurrection`): that card touched
  `docker-compose.prod.yml`, NOT the Taskfile. Card 6 touches ONLY the Taskfile
  (root) among shared infra files. No other Track-B card in this run edits the
  root Taskfile's sync family.

### `.night-crew/qa/spike-supabase/spike-f-browser-live.sh` (B-170 hardening)
- **CHANGED** the `run_browser()` closure (was bare `npx playwright test` at
  line 438) to resolve the Playwright CLI **deterministically** —
  `node "$PW_CLI"` where `PW_CLI="$REPO_ROOT/node_modules/@playwright/test/cli.js"`
  (fallback `node_modules/playwright/cli.js`), with a preflight guard that the CLI
  file exists — the exact pattern already shipped in `sync-app-proof.sh:224`.
  This is the script named in B-170's lead as still vulnerable; it is the one
  Playwright-running script in this card's footprint, so it gets the fix.
- The `require.resolve('@playwright/test')` preflight at line 237 already existed;
  this change makes the ACTUAL invocation use that resolved CLI instead of PATH.

## What must survive
- The `sync:dev:up` → `sync-dev-up.sh` **B-164 :5433 refusal** (unchanged; invoked,
  not bypassed).
- The existing `sync:dev:*` family (`up`/`status`/`down`/`env`/`proof`) — untouched.
- `backend:dev:tailscale`'s own `HQ_SYNC_*` env resolution — reused, not duplicated.
- `spike-f-browser-live.sh`'s tri-state exit contract and every other line — only
  `run_browser()`'s invocation form changed.

## What is safe to drop / not carried
- No new `:5433` guard machinery (B-171's whole point — capability, not detection).
- No new spec file, no seam key/token in `night-crew.toml` (footprint is Taskfile +
  a spike script, both outside the `[e2e.seams]` map → no spec subset; the
  standalone-harness note already documents that spike scripts live outside tests/).

## Gate posture
- **G1** (backend build+vet): green — but this card touches no Go, so it's trivially
  unaffected.
- **G2-Go / G2-Playwright / G4**: N/A-by-footprint — no Go, no HTML/JS, no `tests/*.spec.js`.
  `night-crew.toml` `[e2e.seams]` maps this footprint (root `Taskfile.yml` +
  `.night-crew/qa/spike-supabase/*.sh`) to **no** seam key → no spec subset.
- **RF**: `task sync:dev` is Taskfile convenience wiring — a code red-first isn't
  meaningful; verified by `task --dry sync:dev` showing the resolved step chain.
  The B-170 hardening's red-first is the bare-`npx` grep-assertion (reds on bare
  `npx playwright`, greens on the deterministic `node "$PW_CLI"` resolver).
