# Cross-contamination surfaces — HQ

**Date:** 2026-07-21
**Branch:** `investigate/cross-contamination-20260721` (worktree `wt-0c`)
**Scope:** investigation only — nothing was fixed, no production code, tests, or Taskfile touched.

## Why this exists

Two standing operator preferences:

> "It's almost always my preference to run on a clean DB with fixtures in order to preserve determinism (unless there is a specific reason not to do so)."
>
> "It's almost always my preference to use separate schemas so as to not cross contaminate (e.g. separate schemas for dev, qa, etc); this should be applied everywhere — investigate any surfaces where cross contamination is possible."

This document applies the second clause: an inventory of every surface in this project where state can leak between runs, suites, sessions, or environments.

## The archetype that prompted it

`tests/sync.spec.js:525` (`FLD-LIVE-02`) was scheduled as a flake-fix card. It is not a flake. The Playwright suite shares one server and one database across ~500 tests; the `ops` table is an append-only journal that never shrinks; `task test` drops and recreates `hq_test` but a bare `npx playwright test` does not. The journal therefore accumulates **across runs**, and a newly-opened browser context replays it from Lamport 0. The failure is a function of how many times you have run the suite since the last clean — not of test order. Two engineers running "the same tests" get opposite results depending on which command they typed.

That shape — *shared mutable state + an entry point that does not reset it* — is the thing to hunt. It recurs seven more times below.

---

## Surface table

Severity: **S1** = has already caused a wrong result or lost time; **S2** = will bite under a plausible near-term change; **S3** = latent, currently masked by a convention.

| # | Surface | What is shared | Between what | Observable symptom | Already bitten? | Sev |
|---|---------|----------------|--------------|--------------------|-----------------|-----|
| 1 | `ops` append-only journal in `hq_test` | The sync catch-up log; never truncated by any Playwright path | Successive **runs** on the same DB | `FLD-LIVE-02` and other two-context sync tests time out once the journal is large; passes at ~98 ops, fails at 614+ | **Yes** — the card that prompted this investigation | S1 |
| 2 | `:8199` foreign-server latch (`reuseExistingServer`) | The test port, and whatever DB the squatting server points at | Concurrent **sessions / worktrees** | Either (a) suite silently runs green against another agent's DB, or (b) port is held by a wedged server and Playwright cannot bind | **Yes — four runs** (see citations) | S1 |
| 3 | `hq_test` shared by Go and Playwright suites | One database, one schema, no namespacing | **Go suite ↔ Playwright suite** | `go test` TestMains `TRUNCATE` tables (incl. `ops`, `users`) out from under a concurrently-running Playwright suite | Not observed; serial convention masks it | S2 |
| 4 | `hq_test` colocated with dev `yumyums` in `yumyums-dev-pg` | One Postgres cluster, one role, one credential pair | **test ↔ dev environment** | A wrong `DB_NAME`/`DB_HOST` makes a reset target live dev data; `DROP DATABASE` is one variable away | Not observed | S2 |
| 5 | Live production credentials in test runs | `backend/.env` via root `dotenv:`; alert queue is **not** gated | **test ↔ production side effects** | An E2E run can fire a real Zoho Cliq message and a real SMTP email | Not observed | S1 |
| 6 | Shared working tree `/home/jcole/projects/hq` | One checkout, one git index, one `.planning/`, one `refs/stash` | Concurrent **Claude sessions** | Writes land on whatever branch is actually checked out, not the one the writer believes | **Yes** — a session wrote to the tree this morning believing it was on a different branch | S1 |
| 7 | Global `afterEach` sweeps | `pendingApprovals` is not scoped by creator | **spec file ↔ spec file** | An `afterEach` approves *every* pending submission in the shared DB | Not observed; `workers:1` masks it | S3 |
| 8 | Fixed-name test fixtures | Template names like `Friday TwoTab` | Successive runs / concurrent workers | Unique-constraint collisions (`0052_template_name_unique.sql`) or wrong-row assertions | Not observed | S3 |
| 9 | Host `:5432` squatted by a foreign project | The default `DB_PORT` in both Taskfiles | **HQ ↔ slack-trading** | `task test` at defaults connects to `infra-postgres-1` (postgres:13, role `grodt`) and fails `role "yumyums" does not exist` | Not observed as such | S2 |
| 10 | Long-lived `yumyums-e2e-pg` container | One pg container, up 3 days, reused across sessions | **session ↔ session** | Accumulated `hq_test` (1132 ops observed) and a stray `hq_f5_go` DB | **Yes** — this is the substrate for #1 | S2 |

### Surfaces that already conform — recorded as clean

These were checked and found genuinely isolated. Not padding; a clean surface is a useful finding.

| Surface | Why it is clean |
|---------|-----------------|
| **Service worker caches** | `playwright.config.js` sets `serviceWorkers: 'block'`. No SW is registered in tests, so no SW cache can leak between runs. |
| **IndexedDB (`hq_offline_v1`) and the client Lamport clock** | Playwright gives each test a fresh `BrowserContext` with its own storage partition. `LamportClock._ts` is seeded from IndexedDB, but that IndexedDB is per-context and dies with it. Shared-IDB behaviour inside `FLD-LIVE-02` is *deliberate* (two tabs, one context) and correctly scoped. |
| **`localStorage`** | Only use is `APPS_CACHE_KEY` in `index.html`. Per-context, cleaned with the context. |
| **`device_id`** | Generated fresh per tab (`sync.js:113-115`), explicitly *not* shared via IndexedDB. Correct by design and commented as such. |
| **night-crew ephemeral env** | `docker-compose.nc.yml` brings up a throwaway app + Postgres per run, torn down with `down --volumes`, host port left to Docker so concurrent envs do not collide. `NIGHTCREW_ENV_URL` makes Playwright skip its own `webServer` entirely. **This is the one surface that already implements the operator's stated preference end-to-end.** |
| **`test-results` output dir** | `TEST_OUTPUT_DIR` gives each stack its own artifact dir; the `task test:ui` collision was already fixed. |
| **`task test:ui`** | Runs on its own port (`:8091`) against its own freshly-reset DB (`hq_test_ui`). Correctly isolated from `task test`. |
| **`drift_check_results`** | Append-only in principle, but `week_start` is the primary key with `INSERT ON CONFLICT DO NOTHING` — bounded at one row per week, and truncated by `recipes/helpers_test.go`. Not a growth surface. |
| **`receipt_sync_runs`** | Grows, but is only ever read as `ORDER BY started_at DESC LIMIT 1`. Row count does not affect behaviour. Not a contamination surface. |

---

## Detail and remediation, per violating surface

### 1 — `ops` append-only journal (S1)

**Shared:** `ops` (`0015_ops.sql`) is a pure append log. Verified: the *only* place in the entire repo that removes rows is `backend/internal/sync/access_test.go:36`, a Go test. **No Playwright path ever truncates it**, and `playwright.config.js` has **no `globalSetup`/`globalTeardown`** — so the reset exists only as a shell command inside the `task test` Taskfile target.

Entry points, verified by reading the definitions:

| Entry point | Resets `hq_test`? |
|---|---|
| `task test` | **Yes** — `DROP DATABASE` + `CREATE DATABASE` before the run |
| `task test:all` | **After** the run (EXIT trap), not before |
| `npx playwright test` (bare) | **No** |
| `task bdd` | **No** — `deps: [sw, bdd:gen]` only |
| `task test:go` | **No** — `deps: [backend:db-test]` only ensures existence |
| night-crew ephemeral | **Yes** — fresh container per run |

Observed: 1132 `ops` rows in `yumyums-e2e-pg/hq_test`, 131 in `yumyums-dev-pg/hq_test`.

**Remediation — real card.** Add a `globalSetup` to `playwright.config.js` that truncates the journal (and other unbounded tables) regardless of entry point. Putting the reset in the harness rather than the Taskfile is what makes `npx playwright test` and `task test` agree — which is the actual defect. One-line-config alternative (weaker): have the sync tests assert against a Lamport watermark captured at test start rather than replaying from 0.

### 2 — `:8199` foreign-server latch (S1)

**This was live during the investigation.** Verified at 08:44: PID 10063 listening on `*:8199`, `cwd=.../wt-fix/backend`, `DB_URL=postgres://yumyums:yumyums@localhost:33359/hq_test` — i.e. **another agent's worktree, pointed at another agent's Postgres** (`wtfix-pg-93534`). `curl http://localhost:8199/api/v1/health` **timed out after 3s**: the server is wedged, holding the port without serving.

Two distinct failure modes from one root cause (`reuseExistingServer: !process.env.CI`):

- **Stale but healthy** → Playwright's health probe succeeds, Playwright silently *reuses* the foreign server, and the suite runs green against the wrong database. This is the dangerous one: a false pass.
- **Stale and wedged** (observed today) → probe fails, Playwright tries to spawn its own server, `EADDRINUSE`. Loud, costly, but honest.

`CI=1` converts mode (a) into mode (b) — which is exactly why the standing harness rule mandates it. The rule is a mitigation, not a fix; it depends on every operator remembering.

Prior incidents:
- `.night-crew/runs/2026-07-18-autonomous/HANDOFF.md:75` — "fighting a stale foreign server on :8199 that `reuseExistingServer` latched onto … the code under review was never the problem"
- `.night-crew/knowledge/reference/slate-20260720c.md:224` — "recurred three runs"
- `.night-crew/runs/2026-07-22-autonomous/HANDOFF.md:180` and `DECISIONS-NEEDED.md:223` — "recurred (third run)"
- `.night-crew/knowledge/reference/slate-20260722.md:200` — "cost two prior runs"

**Remediation — one-line config, plus a card.** Immediate: set `reuseExistingServer: false` unconditionally. The comment in `playwright.config.js` justifies port 8199 as protection against reusing the *dev* server on 8089 — but the same mechanism is what enables reuse of a *foreign test* server, so the guard is self-defeating. Better (card): derive `TEST_PORT` per-invocation from a Docker-assigned or OS-assigned free port, as `docker-compose.nc.yml` already does for the ephemeral env. Port exclusivity should never be assumed on a box running five sessions.

### 3 — Go and Playwright share `hq_test` (S1→S2)

`task test:go` sets `DB_TEST_URL=…/hq_test` — the same database `task test` uses. Go `TestMain`s truncate aggressively and with *different* table sets per package:

- `sync/access_test.go` — `ops, submission_responses, template_assignments, checklist_fields, checklist_sections, checklist_templates, users`
- `recipes/helpers_test.go` — `drift_check_results, recipes, daily_menu_sales, …`
- `inventory/period_summary_test.go` — `purchase_line_items, purchase_events, pending_purchases, …`
- `purchasing/scheduler_cron_test.go` — `cutoff_config, repurchase_reset_config, alert_log, low_stock_alert_log, …`

`-p 1` keeps Go packages from stomping each other. Nothing keeps a Go run from stomping a *concurrent Playwright* run — including truncating `users`, which would log every browser context out mid-suite.

**Remediation — this is the canonical "separate schemas" case.** Concretely: give the Go suite `hq_test_go` and Playwright `hq_test_e2e`, both created by `backend:db-test` (parameterise its hardcoded `hq_test`) and both dropped/recreated by their own task. Note that `yumyums-e2e-pg` already contains a stray `hq_f5_go` database — someone has reached for exactly this separation before, ad hoc.

### 4 — `hq_test` colocated with dev data (S2)

Verified: `yumyums-dev-pg` (host `:5433`) contains **both** `yumyums` (live dev) and `hq_test`, same role, same credentials. Separate databases in one cluster is weaker isolation than the operator asked for: a mistyped `DB_NAME` in a `DROP DATABASE` line reaches live dev data, and `backend:db-test` will `CREATE DATABASE hq_test` on whatever `DB_HOST`/`DB_PORT` it is pointed at — including, per the `WIN_TS`/`WIN_LAN` matrix, the Windows box.

**Remediation — card.** Test databases should live only in ephemeral containers, never in the dev cluster. Add a guard to `backend:db-test` that refuses to create `hq_test` on a host that also serves `DB_NAME` (`yumyums`).

### 5 — Live production credentials in test runs (S1) — *not on the candidate list*

The root `Taskfile.yml` opens with `dotenv: ['backend/.env']`. That file (present in the main checkout, gitignored, absent from worktrees) holds **live** `MERCURY_API_KEY`, `ANTHROPIC_API_KEY`, `DO_SPACES_*`, `ZOHO_CLIQ_*`, `SMTP_*`, `TOAST_SFTP_*`. Every `task test` process inherits all of them.

The defences are real but **partial**:
- `playwright.config.js` blanks `MERCURY_API_KEY` and `ANTHROPIC_API_KEY` for the spawned server, and sets `TOAST_SYNC_INTERVAL=0`.
- `E2E_DISABLE_SCHEDULERS=1` gates the receipt worker, cutoff scheduler, and drift scheduler (`backend/cmd/server/main.go:666-696`).

The gap: **`ZOHO_CLIQ_*` and `SMTP_*` are never blanked, and the alert queue is never gated.** `alertQ.Start(ctx)` at `main.go:678` runs unconditionally — it sits *between* the two `schedulersDisabled` branches and is not covered by either. And the enqueue path is not scheduler-only: `backend/internal/purchasing/service.go:645` (`NotifyVendorComplete`) is a **request-path** call reached from HTTP handlers that `tests/purchasing.spec.js` exercises.

So an E2E run launched from the main checkout can deliver a real Zoho Cliq message to the live `purchaseandinventory` channel and a real SMTP email to real crew addresses. This is contamination in the most literal direction: test → production.

Note this is currently *masked in worktrees* — `backend/.env` is gitignored and absent there, so the queue no-ops. The exposure is specific to running from `/home/jcole/projects/hq`.

**Remediation — one-line config now, small card after.** Immediately: add `ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_REFRESH_TOKEN= SMTP_ADDR= SMTP_PASSWORD=` to the blanked set in the `playwright.config.js` `webServer.command`, alongside the two keys already blanked. Then (card): gate `alertQ.Start` on `E2E_DISABLE_SCHEDULERS`, or better, invert the default — require an explicit `ALERTS_ENABLED=1` so that *forgetting* a flag fails safe instead of paging the crew.

### 6 — The shared working tree (S1)

Verified via `/proc`: multiple `claude` processes with `cwd=/home/jcole/projects/hq` — PID 45057 alive since **Thu Jul 16** (5 days), PID 37560 since Jul 20, plus session host PID 88300 and a long-lived shell PID 75822 since Jul 18. `git worktree list` shows the main checkout on `overnight-20260720c` with four sibling worktrees.

One checkout, one git index, one `.planning/`, many writers. There is no locking. A session that reasons about "the current branch" from memory rather than from `git branch --show-current` will write to whatever is actually checked out — which is what happened this morning.

`refs/stash` is shared across all worktrees (`git rev-parse --git-common-dir` → `/home/jcole/projects/hq/.git`), which is why `git stash` is already a standing prohibition. **Verified: `stash@{0}` currently exists** — `WIP on dev: acd2c7f refactor: migrate all server logging…`. Someone's work is parked in a shared, unattributed slot right now. Worth confirming with its owner before it is lost.

Worktrees also share, beyond the stash: all refs (branches, tags, remotes), the object store, `.git/config`, hooks, and the index for `git worktree`-external operations. They do **not** share `HEAD`, the working tree, or the per-worktree index.

Also shared and unmentioned: `/home/jcole/projects/hq/backend/.env` (surface 5) and the long-running dev server PID 75921 on `:8484`, alive since Jul 18, pointed at `postgres://…@100.70.200.55:5433/yumyums` — the **live dev database on the Windows box**. It responds healthy. Anything that reuses `:8484` writes to live dev data.

**Remediation — process, not code.** Agent sessions should work exclusively in `/tmp` worktrees (as this investigation did). The main checkout should be treated as read-only for automated sessions. This is a convention to write down and enforce in `CLAUDE.md`, not a code change.

### 7 — Unscoped `afterEach` sweeps (S3)

`tests/ops-authz-coverage.spec.js:466` approves **every** pending submission in the shared database, not only the ones the file created. The same shortcut appears in `workflows.spec.js:38` and `persistence.spec.js:59` (`cleanupPendingApprovals`).

To the authors' credit this is **documented in-place** (`ops-authz-coverage.spec.js:452-464`) as a deliberate caveat, harmless under `workers:1`, and explicitly flagged as "real cross-file interference under `PW_WORKERS>1`." It is correctly understood, not accidental.

**Remediation — small card, only if parallelising.** Scope the sweep to submissions on the file's own templates, or add a server-side `created_by` filter to `pendingApprovals`. Leave alone while `workers:1` holds.

### 8 — Fixed-name fixtures (S3)

`tests/sync.spec.js:526` creates a template literally named `Friday TwoTab`; `0052_template_name_unique.sql` puts a unique constraint on template names. Fixture-uniqueness discipline is uneven across the suite — `inventory.spec.js` uses 81 `Date.now()`/`Math.random()` calls, while `recipes.spec.js`, `broadcast-rerender.spec.js`, and `repro-cut-task.spec.js` use **zero**.

Currently masked by the fact that `task test` starts from a dropped database. It stops being masked the moment anyone runs the suite twice without a reset — which is precisely surface #1.

**Remediation — small card.** Suffix fixture names with a per-run nonce in the three zero-uniqueness spec files.

### 9 — Host `:5432` squatted by a foreign project (S2)

Both Taskfiles default `DB_PORT` to `5432`. Verified: host `:5432` is bound by **`infra-postgres-1`** — postgres **13**, from `com.docker.compose.project.working_dir=/home/jcole/projects/slack-trading/infra`, with `POSTGRES_USER=grodt` / `POSTGRES_DB=playground`. Its role list contains `grodt` and `temporal`; **no `yumyums` role exists**.

The good news: this fails closed. `task test` at defaults gets `role "yumyums" does not exist` rather than silently writing into another project's database. The bad news: the HQ default points at a port HQ does not own, on a box where `backend:db-start` would try to bind the same port for `yumyums-pg` (a container which does not currently exist). The intended local DB is `yumyums-dev-pg` on `:5433`.

**Remediation — one-line config.** Change the `DB_PORT` default in `backend/Taskfile.yml` from `5432` to `5433` so the default matches the container that actually serves HQ, and drop the `-p 5432:5432` host mapping from `db-start`.

### 10 — Long-lived shared `yumyums-e2e-pg` (S2)

Up **3 days**, bound `127.0.0.1:32772`. Contains `hq_test` (1132 `ops`) and a stray `hq_f5_go`. It is the substrate that lets surface #1 accumulate across sessions: because the container outlives any single run, so does its data.

Concurrent runs do **not** collide on the container itself — each session appears to bring up its own (`wt0a-1198-pg` on `:57260`, `wtfix-pg-93534` on `:33359` were both live during this investigation, correctly on Docker-assigned ports). The problem is longevity, not collision.

**Remediation — process.** Treat every test Postgres as ephemeral: `docker compose down --volumes` at run end, as `docker-compose.nc.yml` already prescribes. Nothing named `-e2e-` should be three days old.

---

## Verified vs. inferred

This project has twice in one night been bitten by plausible-but-unverified claims in operator-facing documents. The line below is drawn strictly.

### Verified by execution

- Container inventory, ports, uptimes, image tags — `docker ps -a`.
- `infra-postgres-1` owns host `:5432`, is postgres 13, belongs to `slack-trading/infra`, and has **no `yumyums` role** — `docker inspect` + `psql -U grodt` via `docker exec`. **Host `:5432` was never connected to over the network**, per the standing rule; all inspection went through `docker exec`.
- `yumyums-e2e-pg/hq_test` has **1132** `ops` rows; `yumyums-dev-pg/hq_test` has **131** — `docker exec psql`.
- `yumyums-dev-pg` contains **both** `yumyums` and `hq_test`; `yumyums-e2e-pg` contains `hq_test` and `hq_f5_go` — `pg_database` listing.
- `wtfix-pg-93534/hq_test` has **no `ops` table** (migrations never ran) — `docker exec psql`.
- PID 10063 holds `*:8199`, `cwd=…/wt-fix/backend`, `DB_URL=…@localhost:33359/hq_test` — `ss -ltnp` + `/proc/10063/environ`.
- `http://localhost:8199/api/v1/health` **times out** (wedged, not merely stale) — `curl --max-time 3`.
- PID 75921 holds `:8484`, alive since Jul 18, `DB_URL=…@100.70.200.55:5433/yumyums`, and **responds healthy** — `/proc` + `curl`.
- Multiple `claude` processes with `cwd=/home/jcole/projects/hq`, oldest PID 45057 since Jul 16 — `/proc/*/cwd`.
- `refs/stash` is shared and **`stash@{0}` currently exists** — `git stash list`, `git rev-parse --git-common-dir`.
- Five worktrees on one common `.git` — `git worktree list`.
- `playwright.config.js` has **no `globalSetup`/`globalTeardown`** — grep.
- The **only** `ops` deletion in the repo is `sync/access_test.go:36` — repo-wide grep across `.go`/`.js`/`.sql`.
- `backend/.env` exists in the main checkout with the 21 keys listed, and is **absent** from worktrees — `cut -d=` on both paths.
- `alertQ.Start(ctx)` at `main.go:678` is not inside either `schedulersDisabled` branch; `purchasing/service.go:645` enqueues from a request path — direct file read.

### Verified by reading (definitions are unambiguous; not executed)

- Which task targets reset `hq_test` (the table in §1). The reset is a literal `psql … DROP DATABASE` in the `task test` cmd block and appears in no other target's dependency chain. Combined with the verified absence of `globalSetup`, this is conclusive: no non-`task test` Playwright entry point can reset the DB.
- Go `TestMain` truncation table sets, per package.
- night-crew ephemeral env isolation semantics (`docker-compose.nc.yml`, `night-crew.toml`).
- Playwright per-context storage isolation for IndexedDB/localStorage — standard, documented Playwright behaviour.

### Inferred — **not** verified

- **That `reuseExistingServer: true` causes a silent green run against a foreign database.** The config value is verified and the mechanism follows from documented Playwright behaviour, but I did **not** execute a run that reused a healthy foreign server. Today's observed instance was the *wedged* variant (loud `EADDRINUSE`), not the silent variant. The prior-run citations describe latching, and `2026-07-18/HANDOFF.md:75` describes a run being misattributed to the code under review — consistent with silent reuse, but not proof of it. **Treat the silent-false-pass mode as inference.**
- **That a live E2E run has ever actually delivered a Zoho Cliq message or SMTP email.** The ungated code path is verified by reading; I did not run the suite against the main checkout to observe a delivery, and deliberately did not, since doing so is the very side effect at issue.
- **The exact `ops` count at which `FLD-LIVE-02` flips** (98 pass / 614+ fail). Taken from the task brief, not re-measured.
- **That a concurrent Go + Playwright run corrupts the Playwright suite** (surface 3). The truncation sets and the shared DB name are verified; the concurrent collision was not staged.
- Which specific session owns `stash@{0}`, and whether its contents are still wanted.

---

## Two things worth raising immediately

Per the "report loudly and stop" instruction, neither was acted on:

1. **`stash@{0}` holds unattributed WIP** (`WIP on dev: acd2c7f refactor: migrate all server logging from log to slog NDJSON output`) in a slot shared by five worktrees. Confirm ownership before anything touches it.
2. **A dev server (PID 75921) has been running since Jul 18 against the live dev database on the Windows box**, responding healthy on `:8484`. It is not a test artifact and was not started by this investigation — but it is a live writer to dev data that nobody may currently be tracking.
