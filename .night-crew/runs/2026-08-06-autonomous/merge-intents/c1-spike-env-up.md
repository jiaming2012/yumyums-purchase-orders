# Merge intent — C1 `spike-a-environment-up`

Run `20260806` · branch `card/c1-spike-env-up` · based at `ef314e0`.

## Files owned by this card

- `.night-crew/qa/spike-supabase/env-up.sh` — **new.** The deliverable: one script that
  takes a machine to "Supabase + RxDB both up, schema applied, healthy", unattended. Its
  **exit status is the verdict** (spike↔script rule).
- `.night-crew/qa/spike-supabase/rxdb/healthcheck.js` — **new.** The RxDB half of the
  health assertion, invoked by `env-up.sh`. Reuses W1/W2's existing `spike-env.js`
  (port resolution + Go-minted token + collection schema) rather than re-deriving them.
- `.night-crew/qa/spike-supabase/captures/` — **new capture file(s)** recording the
  verdict run's stdout/stderr and exit status.
- `Taskfile.yml` — **sole owner tonight** (slate §"Shared-file map"). Adds `spike:up`,
  `spike:up:fresh`, `spike:health`, `spike:down` targets. No existing target is edited.

## Shared files touched outside the footprint

**Nothing here explicitly.** No file outside `.night-crew/qa/spike-supabase/**`,
`.night-crew/runs/2026-08-06-autonomous/**` and `Taskfile.yml` is modified by this card.

In particular this card does **not** touch: `docker-compose.supabase.yml` (W1's committed
substrate — the script drives it, it does not edit it), any `sql/` fixture, any existing
`rxdb/proof-*.js`, `backend/**`, `sync-rxdb/**`, `tests/**`, `night-crew.toml`,
`package.json`, `sw.js`.

## What MUST survive any merge

1. **`env-up.sh`'s exit status semantics.** It exits **non-zero** whenever the stack is
   not fully up-and-healthy, and 0 only when every leg asserted green. Anything that makes
   a leg advisory, `|| true`, or "warn and continue" destroys the card's entire point —
   that is the silent-no-op defect class this cycle exists to retire. `set -euo pipefail`
   and the explicit per-leg `fail()` calls are load-bearing.
2. **`npm ci` in `rxdb/`, not `npm install`.** `package-lock.json` is committed and
   `node_modules/` is gitignored; a clean clone has no `node_modules`, so installing is
   part of "unattended" and the lockfile is what makes it reproducible.
3. **`-p spike-supabase` on every compose invocation.** The distinct project name is what
   guarantees this stack can never adopt, restart or delete HQ's `docker-compose.nc.yml`
   containers or the dev Postgres on :5433. Never drop the flag.
4. **Ports are resolved, never hardcoded.** W1's compose publishes bare container ports;
   Docker assigns the host side and it changes on every `up`. The script resolves them via
   `docker compose … port`.
5. **The `Taskfile.yml` `spike:` targets thin-wrap the script.** The logic lives in the
   script so the verdict is reproducible without `task` installed. Keep it that way.

## What is safe to drop

- The `captures/` file, if triage prefers the figures to live only in the run's
  `HANDOFF.md`. It is evidence, not machinery.
- The `spike:down` Taskfile target — teardown is a deliberate, separate act per W1's
  README, and the raw `docker compose … down --volumes` line is documented there.
- Any prose in this merge-intent file itself.

## Conflict expectations

None. Slate §"Shared-file map" names C1 as `Taskfile.yml`'s **sole owner** tonight, and no
other card in the slate declares `.night-crew/qa/spike-supabase/**`. The only ordering
constraint is the global Playwright/Go suite lock, which is a runtime mutex, not a merge
concern.
