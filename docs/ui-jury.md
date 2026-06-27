# ui-jury — running the visual jury against hq

`/ui-jury` is a multi-agent UI review pipeline that captures screenshots + DOM + a11y trees across declared routes, then runs three critics (visual hierarchy, mobile-viewport, correctness) and a calibrated verifier. The scaffolding lives in:

- `routes.yaml` — what to capture (7 user-visible pages at 393×852)
- `.ui-jury/hooks.yaml` — declares the `db_reset` hook
- `scripts/ui-jury/db-reset.sh` — wraps `task backend:db-reset-inventory`

## Smoke command

```sh
/ui-jury http://localhost:8484 --backend-log-path /tmp/hq-server-8484.log
```

## Prerequisites

1. **Backend dev server running** with stdout/stderr teed to `/tmp/hq-server-8484.log`. Use `task backend:dev:log` in another terminal — the `:log` variant is the one that creates the file `--backend-log-path` points at.
2. **Edit `routes.yaml` first.** The top-level `setup:` block has `password: "<FILL-IN-BEFORE-RUNNING>"` as a placeholder. Replace it with a real test-account password before running. ui-jury v1's routes.yaml does not support env-var substitution in `value` fields, so the placeholder is a literal string — committing a real password would be a footgun. Forgot to swap it? The setup will fail loudly, no captures will produce noise.
3. **Local DB reachable.** `scripts/ui-jury/db-reset.sh` runs `task backend:db-reset-inventory`, which TRUNCATEs three tables via psql against `DB_URL`. If the DB is unreachable, ui-jury aborts at preflight with `SETUP_HOOK_FAILED`.

## What gets reset between runs

Only inventory + receipt-pipeline data: `purchase_line_items`, `purchase_events`, `pending_purchases` (CASCADE). Users and workflow templates are preserved so login works and `/workflows.html` renders templates the operator already saved.

If you want a fully empty DB instead, point `db_reset` at a script that calls `task backend:db-reset` (drops the docker volume) — but be aware you'll lose the test-account user and need to re-seed before login can succeed.

## Known caveats (v1)

- **`/login.html` shows the post-login redirect state**, not the logged-out form. Per-route `setup:` in routes.yaml v1 schema is *additive* to top-level `setup:`, not an override (the schema example explicitly says "Driver applies this AFTER the top-level setup"). To capture the logged-out form, run /ui-jury with the password placeholder unfilled — top-level setup will fail and the Driver will capture login.html as-is. The other routes will also fail auth in that run, so treat it as a separate "logged-out only" pass.
- **Tab-default state only.** `workflows.html` is captured on its default "My Checklists" tab; `inventory.html` on "Purchases". Other tabs need their own `states[]` entries with an interaction that clicks the tab button — deferred to v2 once the first run shows where depth pays off.
- **No `db_seed:` map.** Single default fixture for v1. Add per-fixture `db_seed:` entries when a state needs DB data the default reset doesn't provide.
