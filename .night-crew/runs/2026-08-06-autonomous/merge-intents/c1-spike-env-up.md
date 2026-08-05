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
  verdict run's stdout/stderr and exit status. Two of them:
  `green-20260806-env-up.txt` (build round; its line 3 is **corrected** by the fix round
  to say which of its runs carry a captured `$?` and which do not) and
  `green-20260806b-env-up-fixround.txt` (**the verdict of record** — every run in it ends
  with a captured `EXIT=`).
- `.night-crew/qa/spike-supabase/README.md` — **edited.** W1's runbook now points at the
  one-command bring-up, and (fix round) records that `spike:up` was itself destructive
  until the two F1 mechanisms were closed.
- `Taskfile.yml` — **sole owner tonight** (slate §"Shared-file map"). Adds `spike:up`,
  `spike:up:fresh`, `spike:health`, `spike:down` targets. No existing target is edited.

## Red-first

**n/a — no code change.** Spike WO: the deliverable is tooling under
`.night-crew/qa/spike-supabase/**` plus `Taskfile.yml` targets. No production code and no
`tests/` change.

**Re-verified after the fix round, and still true.** The full diff against `dev` for this
card is: `docker-compose.supabase.yml`, `Taskfile.yml`, and files under
`.night-crew/qa/spike-supabase/**` + `.night-crew/runs/2026-08-06-autonomous/**`. Nothing
under `backend/`, `sync-rxdb/`, `tests/`, or any shipped `*.html`/`sw.js` is touched, so
there is no Playwright or Go spec that could have been written red first.

The card's **equivalent discipline is negative controls** — the exit status is the
deliverable, so proving it can go RED is the red-first. Recorded, each with a captured
`EXIT=` in `captures/`:

| Control | Break applied | Result |
|---|---|---|
| RLS disabled | `alter table public.spike_notes disable row level security` | `FAIL rest — alice can see bob's row note-bob-1 — RLS is NOT discriminating`, **EXIT=1** |
| Policy `using(false)` | policy rewritten to deny everything | `FAIL rest — alice cannot see her own seed row`, **EXIT=1** (original capture) |
| Stack torn down | `down --volumes` | `VERDICT: RED — the db service has no container`, **EXIT=1** (original capture) |
| **hq_\* tables dropped** *(new, fix round)* | `drop table hq_grant_projection, hq_sync_checklists, hq_uid_trap cascade` | `FAIL schema — missing table(s): …; missing policy(ies): …`, **EXIT=1** — and the **pre-fix** `healthcheck.js`, run on that same stack at that same moment, **EXIT=0 GREEN**. That pair is the blind spot demonstrated rather than asserted. |

## Shared files touched outside the footprint

### 🛑 `docker-compose.supabase.yml` — DECLARED, edited, was previously disclaimed

This merge intent originally said the card does **not** touch this file ("W1's committed
substrate — the script drives it, it does not edit it"). **That is no longer true.** The
fix round edits it, which is not a breach (§15ad.65) but must be declared:

- **Added a named volume `spike-db-data` for PGDATA** (`db.volumes`, plus the top-level
  `volumes:` block). Until this existed there was **no volume for the data directory at
  all** — `docker inspect spike-supabase-db-1` showed only the two `initdb/*.sql` bind
  mounts — so PGDATA lived in the **container's writable layer** and *any* recreate was
  total, silent data loss. This is the durable half of the F1 fix.
- **Documentation only** otherwise: a banner note that the relative initdb bind mounts
  make the file path-sensitive and that callers must pin `--project-directory`, and a
  comment on the `realtime` service pointing at `env-up.sh`'s reconcile leg. No service,
  image, port, credential or command line is otherwise changed.

**No other card in run `20260806` declares `docker-compose.supabase.yml`.** The slate's
Shared-file map names no owner for it; C1 is the only card that touches it.

Everything else is unchanged: no `sql/` fixture, no existing `rxdb/proof-*.js`, no
`spike-env.js`, no `backend/**`, `sync-rxdb/**`, `tests/**`, `night-crew.toml`,
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
4. **Ports are resolved, never hardcoded, and re-resolved after any restart.** W1's
   compose publishes bare container ports; Docker assigns the host side and it changes on
   every `up` — **and on every `restart`** (measured: Realtime 50959 → 50135 across one
   `docker compose restart`). That is why resolution lives in the `resolve_ports` function
   and the reconcile leg calls it again. Collapsing it back to three top-level assignments
   reintroduces a 180s timeout against an already-healthy service.
5. **The `Taskfile.yml` `spike:` targets thin-wrap the script.** The logic lives in the
   script so the verdict is reproducible without `task` installed. Keep it that way.
6. 🛑 **`spike-db-data`, the named volume for PGDATA.** Without it PGDATA is in the
   container's writable layer and *any* recreate silently destroys the whole database
   while `docker ps` keeps reading `Up (healthy)`. This is almost certainly what produced
   the card's own headline finding. Do not remove it.
7. 🛑 **`--project-directory "$ANCHOR"` on every compose invocation in `env-up.sh`.** The
   compose file's initdb bind mounts are relative, so the caller's absolute path enters
   the container config hash and `up -d` from a worktree recreates the db container. The
   anchor comes from `git rev-parse --git-common-dir` (shared by every linked worktree of
   a clone) and is overridable with `SPIKE_ANCHOR`. Dropping the flag brings the
   gratuitous recreates back.
8. 🛑 **The `schema` leg in `healthcheck.js` asserts the `hq_*` tables and policies BY
   NAME.** Asserting only `spike_notes` let `task spike:health` exit 0 on a database with
   none of `hq_grant_projection` / `hq_sync_checklists` / `hq_uid_trap` — the tables
   `internal/sync`'s `TestRowVisibilityRLS` (59 subtests) drives. Narrowing this leg
   restores the same conflation the card exists to retire.
9. **The reconcile leg RE-ASSERTS.** It is the file's one self-healing action: restart
   Realtime when it is `CHANNEL_ERROR` against a healthy db, then re-run the full
   assertion. A failing re-assertion is still RED. Turning it into a `|| true`, or
   dropping the re-assert, converts a reconciler into the silent no-op this cycle exists
   to retire. `--health` must keep NOT restarting anything — it promises to touch nothing,
   and names the remedy in its failure text instead.

## What is safe to drop

- The `captures/` file**s**, if triage prefers the figures to live only in the run's
  `HANDOFF.md`. They are evidence, not machinery. (Prefer keeping
  `green-20260806b-env-up-fixround.txt` over the build-round one if only one survives.)
- The `spike:down` Taskfile target — teardown is a deliberate, separate act per W1's
  README, and the raw `docker compose … down --volumes` line is documented there.
- Any prose in this merge-intent file itself.

**Not safe to drop:** anything in "What MUST survive" items 6–9. Each of those is a
confirmed defect's fix, and each defect was silent — the failure mode in every case is a
green that means nothing, not a visible break.

## Conflict expectations

None. Slate §"Shared-file map" names C1 as `Taskfile.yml`'s **sole owner** tonight, and no
other card in the slate declares `.night-crew/qa/spike-supabase/**`.
`docker-compose.supabase.yml` is now edited by this card (declared above) and no other
card tonight touches it, so it introduces no new conflict either. The only ordering
constraint is the global Playwright/Go suite lock, which is a runtime mutex, not a merge
concern — and this card's diff touches nothing either suite reads.

## Fix round — what changed after review

Reviewer verdict on the build round: **MERGE WITH NOTE**. The deliverable's GREEN was
independently reproduced and broken three ways; seven defects were confirmed in it. All
seven are addressed on this branch:

| # | Defect | Fix | Proof |
|---|---|---|---|
| F1 | `task spike:up` could **wipe the spike database** — no PGDATA volume, plus a path-sensitive config hash | `spike-db-data` named volume; pinned `--project-directory` | Sentinel row survives a forced recreate (new container id); control still prints `Recreate` without the anchor, `Running` with it from two different compose-file paths |
| F2 | Could break the stack and then report RED forever — Realtime stays `CHANNEL_ERROR`, `up -d` won't restart a running-but-broken service | reconcile leg: restart realtime, re-resolve ports, **re-assert** | PGDATA destroyed under a running realtime → FAIL → reconcile → `PASS realtime SUBSCRIBED` → EXIT=0 |
| F3 | `task spike:health` exited 0 on a stack missing **every** `hq_*` table | `schema` leg asserting 4 tables + 5 policies by name | New: EXIT=1 naming each missing object. Old (`git show HEAD:`), same stack same moment: EXIT=0 GREEN |
| F4 | No `## Red-first` section | added above, with the negative-control table | this file |
| F5 | GREEN evidenced by prose | new capture with a captured `EXIT=` on every run; old capture's line 3 corrected | `captures/green-20260806b-env-up-fixround.txt` |
| F6 | `env-up.sh` hardcoded `/home/jcole/go/bin` | probes GOROOT/GOPATH/conventional dirs; preflight still names `go` | RUN A–C green with it applied |
| F7 | `port_of` accepted `:0` from a defined-but-stopped service | numeric, non-zero check, fails immediately | code; the old path surfaced as a 180s `wait_http` timeout naming the wrong thing |

Three of the card's own claims asserted the **opposite** of F1 and are corrected rather
than left standing: `Taskfile.yml`'s `spike:down` comment ("never run by `spike:up`"),
the README's "teardown is still a deliberate and separate act", and this file's
*What MUST survive*. All three are now true **because of** the fixes, and each says so.
