# ui-jury — running the visual jury against hq

`/ui-jury` is a multi-agent UI review pipeline that captures screenshots + DOM + a11y trees across declared routes, then runs three critics (visual hierarchy, mobile-viewport, correctness) and a calibrated verifier.

All ui-jury config lives under `.ui-jury/`:

| Path | Status | Purpose |
|---|---|---|
| `.ui-jury/routes.template.yaml` | committed | Source-of-truth routes declaration (7 user-visible pages at 393×852) with `${VAR}` placeholders |
| `.ui-jury/.env.example` | committed | Seed for the gitignored `.env` |
| `.ui-jury/.env` | **gitignored** | Real values (e.g., `HQ_TEST_PASSWORD`) |
| `.ui-jury/hooks.yaml` | committed | Declares the `db_reset` hook |
| `.ui-jury/scripts/render-routes.sh` | committed | Renders `.ui-jury/routes.template.yaml` → `routes.yaml` using `.env` (project-agnostic — works in any project that mirrors this layout) |
| `.ui-jury/scripts/db-reset.sh` | committed | Wraps `task backend:db-reset-inventory` |
| `routes.yaml` | **gitignored**, at repo root | Rendered output read by `/ui-jury` (the skill requires it at repo root) |

## First-time setup

```sh
cp .ui-jury/.env.example .ui-jury/.env
# edit .ui-jury/.env and set HQ_TEST_PASSWORD to a real test-account password
```

`.ui-jury/.env` is gitignored — your real password never enters git history.

## Smoke command (after the first-time setup)

```sh
# terminal 1 — backend with stdout/stderr teed to a log
task backend:dev:log

# terminal 2 — render routes.yaml from the template, then run /ui-jury
./.ui-jury/scripts/render-routes.sh
/ui-jury http://localhost:8484 --backend-log-path /tmp/hq-server-8484.log
```

`render-routes.sh` re-reads `.ui-jury/.env` every time, so updating the password is just an edit + re-render.

## Prerequisites

1. **Backend dev server running** with stdout/stderr teed to `/tmp/hq-server-8484.log`. Use `task backend:dev:log` in another terminal — the `:log` variant is the one that creates the file `--backend-log-path` points at.
2. **`.ui-jury/.env` populated.** If you skip the first-time setup, `render-routes.sh` exits non-zero with instructions. ui-jury v1's `routes.yaml` schema does not support env-var substitution at run time, so substitution happens at *render* time.
3. **Local DB reachable.** `.ui-jury/scripts/db-reset.sh` runs `task backend:db-reset-inventory`, which TRUNCATEs three tables via psql against `DB_URL`. If the DB is unreachable, ui-jury aborts at preflight with `SETUP_HOOK_FAILED`.

## What gets reset between runs

Only inventory + receipt-pipeline data: `purchase_line_items`, `purchase_events`, `pending_purchases` (CASCADE). Users and workflow templates are preserved so login works and `/workflows.html` renders templates the operator already saved.

If you want a fully empty DB instead, point `db_reset` at a script that calls `task backend:db-reset` (drops the docker volume) — but be aware you'll lose the test-account user and need to re-seed before login can succeed.

## Reusing this layout in another project

`.ui-jury/scripts/render-routes.sh` is project-agnostic — it auto-discovers variable names from `.env`. To reuse:

1. Copy `.ui-jury/scripts/render-routes.sh` into the other project's `.ui-jury/scripts/`.
2. Create `.ui-jury/.env.example` declaring your own variables (any names).
3. Create `.ui-jury/routes.template.yaml` and reference your variables as `${YOUR_VAR}`.
4. Add `/routes.yaml` and `/.ui-jury/.env` to that project's `.gitignore`.

No script edits needed — drop-in.

## Known caveats (v1)

- **`/login.html` shows the post-login redirect state**, not the logged-out form. Per-route `setup:` in routes.yaml v1 schema is *additive* to top-level `setup:`, not an override (the schema example explicitly says "Driver applies this AFTER the top-level setup"). To capture the logged-out form, render `routes.yaml` with `HQ_TEST_PASSWORD=wrong` (or any value the backend will reject) — top-level setup will fail and the Driver will capture login.html as-is. The other routes will also fail auth in that run, so treat it as a separate "logged-out only" pass.
- **Tab-default state only.** `workflows.html` is captured on its default "My Checklists" tab; `inventory.html` on "Purchases". Other tabs need their own `states[]` entries with an interaction that clicks the tab button — deferred to v2 once the first run shows where depth pays off.
- **No `db_seed:` map.** Single default fixture for v1. Add per-fixture `db_seed:` entries when a state needs DB data the default reset doesn't provide.
