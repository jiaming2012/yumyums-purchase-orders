# Merge intent — card C · `spike-c-round-trip` (run `20260807-2`)

Branch `card/c-spike-c-round-trip`, cut from the run branch. Activity 2, spike C —
the 3rd of D-KR1's four spike verdicts.

**What the card is.** Establish whether *one row written through the REAL write path*
(`POST /api/v1/workflow/saveResponse`) reaches an **RxDB-served read** within bounded
time, **and by what mechanism**. Decision 126 measured the "rows flow back" premise
false on night nine of nine; this spike measures it on purpose, by script, and the
script's **exit status is the verdict**. A red is a successful spike.

---

## Shared files touched

| File | Why |
|---|---|
| `Taskfile.yml` | Adds ONE new target, `spike:roundtrip`, beside the existing `spike:up` / `spike:migration`. 🛑 The `prod:backup` stanza and every `test:*` block are untouched — verify with `git diff Taskfile.yml`. |
| `backend/cmd/spikec-relay/main.go` | **Outside the declared footprint** (`backend/internal/sync/**` was declared; a Go library cannot be executed without a `main` package). ~40 lines, no build tags removed, no route registered, imported by nothing in `cmd/server`. Marked `SPIKE CODE` in its banner. Declared here rather than smuggled. |
| `backend/internal/sync/spikec_relay.go` | **Spike probe/prototype code, clearly marked.** New file, additive only. Nothing in `handler.go` / `hub.go` / `listener.go` / `ops.go` / `proxy.go` / `jwtbridge*.go` is edited. It exports exactly one symbol, `RunSpikeCRelay`, and **no production call site references it** — `grep -rn RunSpikeCRelay backend/cmd/server` returns nothing. Its banner states in the first paragraph that it is spike code and not a proposal for HQ. |

Everything else is inside the card's own footprint: `.night-crew/qa/spike-supabase/`
(the script, its scratch-Postgres compose file, its trigger SQL, its RxDB read leg)
and `.night-crew/runs/2026-08-07-2-autonomous/` (gate logs + this file).

**Files NOT touched, deliberately:** `backend/go.mod`, `backend/go.sum`
(the relay uses `github.com/jackc/pgxlisten`, `pgx/v5` and `pgxpool` — all already
DIRECT dependencies of `backend/go.mod`, because `internal/sync/listener.go` already
uses every one of them. Nothing new enters HQ's supply chain). `package.json`,
`playwright.config.js`, `night-crew.toml`, `sw.js` inputs (no HTML/JS asset changed —
so `node build-sw.js` must stay at precache **31**), and every file under
`.night-crew/qa/spike-supabase/` that spike A or spike B owns
(`env-up.sh`, `spike-b-migration.sh`, `rxdb/spike-env.js`, `rxdb/hq-*.js`,
`sql/hq-bridge-*.sql`, `sql/hq-source-*.sql`, `docker-compose.hq-source.yml`,
`docker-compose.supabase.yml`) — all consumed read-only so spike A's and spike B's
GREEN verdicts keep reproducing byte-for-byte.

## What must survive any merge

1. **`.night-crew/qa/spike-supabase/spike-c-roundtrip.sh` and its exit-code contract.**
   The contract is the deliverable, not the prose: `0` = the round trip closed,
   `1` = **ran and the mechanism is disproven** (RED — a successful spike),
   `2` = **could not run** (infra/setup failure — NOT a verdict), `64` = usage.
   Merging a version that collapses 1 and 2 destroys the only thing the card is for.
2. **The `--no-relay` flag.** It is how the red-first capture is *reproduced on demand*
   rather than merely reported. Do not "tidy" it away as dead code.
3. **The teardown's substrate restore, and its self-verification.** `hq_sync_checklists`
   and `hq_grant_projection` are spike A's SHARED tables and
   `backend/internal/sync/jwtbridge_rls_test.go`'s `service_role` CONTROL asserts an
   EXACT full-table row set — rows left behind by this spike RED a committed Go suite
   (measured on spike B's own first G2 run). This script snapshots the baseline id set
   BEFORE it writes anything and **asserts the id set is byte-identical after teardown**;
   a failed restore turns a green run RED. B-148's residual is that spike B's red path was
   never re-rehearsed, so this card does not reuse `rxdb/hq-reset.js` — it verifies its
   own recovery, and that verification is exercised on every run including the red one.
4. **Isolation assertions as executable checks, not comments.** The scratch Postgres
   publishes no fixed host port (grep-asserted against the compose file) and the resolved
   ephemeral port is re-checked against `5432/5433/5434` at runtime. :5433 is the
   PRODUCTION cluster; a probe there destroyed the prod DB on 2026-08-06.
5. **The mechanism finding**, recorded in the run's HANDOFF regardless of colour.
6. **The double-fire finding**, recorded in `spikec_relay.go` beside the code it affects:
   ONE call to `/saveResponse` fires the relay **twice** for the same response row —
   `workflow/repository.go:826` (the save) and `sync/ops.go:148`
   (`UPDATE submission_responses SET lamport_ts`, EmitOp's LWW stamp). A row-level
   trigger cannot tell them apart, so any change-data-capture mechanism on this table
   sees 2× the write volume a reader would predict. Harmless here only because the
   projection is an idempotent upsert. The cutover card inherits the decision.

## What is safe to drop

- The scratch containers and the two compose projects (`spike-c-hq`) — created and
  destroyed every run by construction; nothing persists.
- The captured gate logs under `.night-crew/runs/2026-08-07-2-autonomous/` may be
  pruned once the run is triaged and the verdict is in the roadmap.
- `backend/cmd/spikec-relay/` and `backend/internal/sync/spikec_relay.go` are **spike
  artefacts with a defined end of life**: the Activity 3 card `skeleton-one-row-end-to-end`
  either adopts the mechanism properly (in which case it replaces both files) or the
  spike verdict retires them. Neither is a proposal for HQ's production shape.

## Conflicts expected with the run's other card

- Card D (`spike-d-realtime-live`) also consumes spike A's substrate in reconcile mode
  and also adds a `spike:*` Taskfile target. **The Taskfile is the one real conflict
  surface** — both cards append a target to the same `spike:` block. Resolution is
  additive: keep both targets. This card's target is named `spike:roundtrip`.
- Both cards read `hq_sync_checklists`. This card restores it exactly; if D's leg runs
  concurrently the run is serial by slate design, so there is no interleaving.

---

## Red-first

Gate RF. The natural red for a spike is **the script red before the mechanism exists**.

The harness is built end-to-end first — scratch HQ Postgres on the REAL HQ schema
(all 74 migrations, applied by HQ's own `db.Migrate`), the REAL `cmd/server` booted
against it, a REAL `/api/v1/auth/login` session, a REAL
`POST /api/v1/workflow/saveResponse`, and a REAL RxDB client replicating from spike A's
substrate over a real signed token with RLS live — with the **relay stage absent**.
The round trip then cannot close, and the script must say so loudly and exit `1`,
never `0` and never `2`.

That capture is reproducible on demand at any later date with:

```
.night-crew/qa/spike-supabase/spike-c-roundtrip.sh --no-relay
```

**Captured red (pre-mechanism):**

- command: `.night-crew/qa/spike-supabase/spike-c-roundtrip.sh --no-relay`
- log: `.night-crew/runs/2026-08-07-2-autonomous/card-c-rf-red.log`
- exit code: **1** (ran, mechanism disproven — distinct from `2` "could not run")
- failing leg: `9. ROUND TRIP` — the row written through `/saveResponse` never reached
  the RxDB collection; the read leg exited 1 on its own deadline with
  `no relay is running`.
- proof the red is *the mechanism* and not a broken harness: legs 1–8 all PASS in the
  same log — substrate up (spike A GREEN, reconcile), scratch Postgres healthy on
  ephemeral port 51693, **75 goose versions / 52 public tables applied by HQ's own
  binary**, `POST /api/v1/auth/login` HTTP 200 with a real `hq_session` cookie, a real
  text field resolved from the seeded template (`Notes [text]`),
  `POST /api/v1/workflow/saveResponse` **HTTP 204**, and — the load-bearing line —
  `HQ Postgres holds 1 row(s) carrying the sentinel`. The RxDB client was live and
  replicating before the write and held **0 docs**, which is *correct*: spike A's
  fixture rows are owned by `hq-user-alice`/`hq-user-bob`, and this run's token carries
  HQ's real uuid, so the identity axis of the RLS policy hides them. The harness is
  demonstrably alive on both sides of a gap that nothing bridges.
- teardown on the red path **verified** `hq_sync_checklists` and `hq_grant_projection`
  byte-identical to the pre-run baseline — i.e. the recovery path B-148 flagged as
  never re-rehearsed was exercised on the red run, not only the green one.

**Green run:** the same command with the relay armed —
`.night-crew/runs/2026-08-07-2-autonomous/card-c-spike-verdict.log`, exit **0**,
round trip closed in **248 ms** against a 20 000 ms bound. Both logs are committed.

**One harness defect the first armed run surfaced, fixed and worth carrying forward:**
the server was started in a subshell, so `$!` was the subshell's pid and teardown's
`kill` left the server running as an orphan bound to the port. The next run's health
poll went green in milliseconds against that orphan while its own migrator was three
migrations in, and the script reported "the migrator did not run" against a foreign
database. Fixed both ways — `env … binary &` so the pid is the server, plus a pre-flight
refusal if anything already answers on the port. This is the same class
`playwright.config.js` documents at length for `reuseExistingServer`; any future card
that spawns a server from a shell should copy both halves.
