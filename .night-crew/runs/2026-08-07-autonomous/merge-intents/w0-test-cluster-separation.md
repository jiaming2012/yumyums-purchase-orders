# Merge intent — W0 · `test-cluster-separation`

Run `20260807` · branch `card/w0-test-cluster-separation` · executes ledger **decision 155**
(*"D-2b: the test suites leave the production cluster"*), closes the **structural half of B-141**
(*a test file holding admin credentials to the cluster that serves `hq.yumyums.kitchen`*).

---

## The container this card creates

| | |
|---|---|
| Compose file | `docker-compose.test.yml` (new, repo root) |
| Compose project | `yumyums-test` |
| Service | `postgres-test` |
| Container name | `yumyums-test-pg` |
| Image | `postgres:16` (same major as the shared cluster) |
| Host port | **`5434`** — not 5433 (`yumyums-dev-pg`, which serves prod), not 5432 (`infra-postgres-1`) |
| Named volume | `yumyums-test-pgdata` (its own; `yumyums_yumyums-pgdata` is prod's and is untouched) |
| Credentials | `hqtest` / `hqtest`, maintenance DB `postgres` |
| Lifecycle | `task test:db:up` (idempotent, waits healthy) / `task test:db:down` (destroys volume) |

🛑 **The credentials are deliberately NOT `yumyums:yumyums`.** Separation by port alone leaves a
stale default one typo away from prod; a different *role name* means a missed site fails with
`role "hqtest" does not exist` rather than authenticating against the production cluster.
`yumyums:yumyums` no longer appears in any test-path default in the tree.

🛑 **The port is published on all interfaces (`5434:5432`), not loopback-only.** The RLS suite's
FDW leg connects from *inside* the spike-supabase container to `host.docker.internal:<port>`; a
`127.0.0.1:`-bound publish is unreachable from a container and would have silently broken that
leg. This matches how `yumyums-dev-pg` already publishes `5433`.

Every later leg of this run inherits these coordinates.

---

## Shared files touched

| File | Why |
|---|---|
| `Taskfile.yml` | Shared with stretch card **S** (`spike:migration` target) — **disjoint stanzas**: this card only edits the `test:*` env blocks + adds a `TEST_DB_*` vars block and three `test:db:*` / `test:targets` tasks. 🛑 The attended `prod:backup` stanza (merged `0a764a9`) is **not touched** — rebase over it. |
| `docker-compose.test.yml` | New file, sole owner. `docker-compose.nc.yml` and `docker-compose.prod.yml` are **not modified** — production topology is untouched (PARK condition not tripped). |
| `backend/internal/sync/rowvisibility_rls_test.go` | The two default constants only (+ the one stack-precondition comment line that names the old host, which would otherwise be a lie). 🛑 The preserved branch `card/a3-rls-fixture-own` edits this same file; see **what must survive** below. |
| `scripts/verify-test-harness.sh` | `DEAD_URL` re-point (both arms of the `H1_DEAD_PORT` branch, lines 204/207 — the dead-port arm carried the same `yumyums:yumyums` literal). |
| `scripts/reset-e2e-db.js` | **Outside the declared footprint — declared deviation.** This file is *the ONE place the Playwright e2e Postgres coordinates are computed* (`playwright.config.js` imports `resolveE2eDb` from it and holds no copy). Its defaults were `localhost:5433` + `yumyums:yumyums`, and `webServer.command` **DROPs the database it resolves** as its first act. Leaving it would mean a bare `npx playwright test` — the exact invocation `night-crew.toml`'s `[e2e]` stanzas run — still issuing `DROP DATABASE` against the production cluster. The card cannot deliver its scope sentence without this file. Nobody else owns it tonight. |
| `backend/Taskfile.yml` | **Outside the declared footprint — declared deviation.** One-token change: `ALLOW_TEST_DB_ON_DEV_HOST` default `1` → `0`, arming the dormant guard in `db-test` exactly as its own comment promises ("*will start failing loudly the moment a proper test cluster exists*"). No other line changed. |
| `.night-crew/knowledge/BACKLOG.md` | B-141 structural half marked closed by this card; the prefix-guard half + B-142 explicitly left open on the attended A3 re-gate. |
| `.night-crew/knowledge/roadmap.md` | Card status flip for `test-cluster-separation`, per that file's existing convention. |

---

## What must survive any merge

🛑 **The re-pointed defaults in `backend/internal/sync/rowvisibility_rls_test.go`.** The attended
A3 re-gate (`gate-rls-fixture-ownership`, branch `card/a3-rls-fixture-own`) edits this same file
and was authored against the OLD values. When it rebases, these values win:

```go
defaultHQAdminURL = "postgres://hqtest:hqtest@localhost:5434/postgres"   // was postgres://yumyums:yumyums@localhost:5433/postgres
defaultFDWPort    = "5434"                                              // was "5433"
```

`defaultFDWHost` stays `host.docker.internal` — unchanged, still correct.

A3's prefix guard (`^hq_rls_[a-z0-9_]+$` on `HQ_RLS_TEST_DB`) is **orthogonal and additive**: it
constrains the database *name*, this card moves the *cluster*. Both are wanted. If a rebase
presents a conflict in that `const` block, the resolution is *A3's guard + these two values*.

Also must survive:

- **The container coordinates table above.** Every gate leg after W0 resolves against
  `localhost:5434` / `hqtest`. A merge that restores any `:5433` or `yumyums:yumyums` literal into
  a test path re-opens B-141's mechanism.
- **`scripts/reset-e2e-db.js`'s new defaults** — it is the single source of the Playwright
  coordinates; a revert there silently re-arms the `DROP DATABASE`-against-prod path.

## What is safe to drop

- `task test:targets` (the read-only "print every resolved test-DB coordinate" task). Convenience
  and red-first evidence surface only; nothing depends on it.
- The prose/comment rewrites explaining *why* `5434`/`hqtest` — informative, not load-bearing.
- The `backend/Taskfile.yml` `ALLOW_TEST_DB_ON_DEV_HOST` default flip, if a merge conflicts on it:
  it is a belt-and-braces guard, not the separation itself.
- `.night-crew/runs/2026-08-07-autonomous/w0-logs/**` — gate evidence, append-only, never conflicts.

## Nothing here

- **No production topology changed.** `docker-compose.prod.yml`, `Taskfile.yml`'s `prod:*` stanzas
  (including the attended `prod:backup`), and everything `task prod:deploy` reads are byte-identical.
- **No `night-crew.toml` key added or changed.** (Both PARK conditions therefore untripped.)
- **No schema, migration, API, or frontend change.** No `sw.js`-affecting file touched.
- **No contact with `:5433` at any point in this card** — no test, no probe, no psql, read-only
  included. The red-first evidence is resolution/render only.

---

## Red-first

**Gate RF.** One command, run **before** and **after** the re-point, resolving/rendering every
test-path Postgres default. 🛑 It opens **no connection** — `resolveE2eDb()` is pure string
construction and the rest is `grep`. No suite was run against `:5433`.

Command (identical both times):

```
bash <scratch>/probe-test-db-defaults.sh
```

which is:

```bash
cd /home/jcole/projects/hq-worktrees/w0-test-cluster-separation
node -e 'const{resolveE2eDb}=require("./scripts/reset-e2e-db");const d=resolveE2eDb({});console.log("  adminUrl (DROP/CREATE target):",d.adminUrl);console.log("  testUrl                      :",d.testUrl)'
grep -n "DB_TEST_URL:\|DB_PORT:\|DB_HOST:" Taskfile.yml
grep -n 'defaultHQAdminURL = \|defaultFDWPort = ' backend/internal/sync/rowvisibility_rls_test.go
grep -n "DEAD_URL='postgres" scripts/verify-test-harness.sh
```

### BEFORE — every test path resolves to `:5433`, the cluster serving `hq.yumyums.kitchen`

`EXIT=0` · full capture: `.night-crew/runs/2026-08-07-autonomous/w0-logs/redfirst-before.log`

```
== 1. Playwright e2e coordinates, RESOLVED by scripts/reset-e2e-db.js (no connection) ==
  adminUrl (DROP/CREATE target): postgres://yumyums:yumyums@localhost:5433/postgres?sslmode=disable
  testUrl                      : postgres://yumyums:yumyums@localhost:5433/hq_test_e2e?sslmode=disable&TimeZone=America/New_York
== 2. task test:* DB coordinates (Taskfile.yml) ==
83:      DB_HOST: localhost
84:      DB_PORT: '5432'
92:      DB_TEST_URL: 'postgres://yumyums:yumyums@{{.DB_HOST | default "localhost"}}:{{.DB_PORT | default "5433"}}/hq_test_go?sslmode=disable'
107:      DB_HOST: localhost
108:      DB_PORT: '5433'
110:      DB_TEST_URL: 'postgres://yumyums:yumyums@localhost:5433/hq_test_go?sslmode=disable'
124:          echo "── resetting hq_test_go + hq_test_e2e on $DB_HOST:$DB_PORT ──"
125:          psql "postgres://yumyums:yumyums@$DB_HOST:$DB_PORT/postgres?sslmode=disable" \
== 3. RLS suite defaults (backend/internal/sync/rowvisibility_rls_test.go) ==
221:	defaultHQAdminURL = "postgres://yumyums:yumyums@localhost:5433/postgres"
226:	defaultFDWPort = "5433"
== 4. harness probe URL (scripts/verify-test-harness.sh) ==
204:	DEAD_URL='postgres://yumyums:yumyums@127.0.0.1:5599/hq_test_go_does_not_exist?sslmode=disable'
207:	DEAD_URL='postgres://yumyums:yumyums@127.0.0.1:5433/hq_test_go_dropped_by_a_reviewer?sslmode=disable'
```

Note site **#125** (`test:all`'s post-run reset trap) — an admin `psql` against the shared
cluster that the card's verification list did not name. It is re-pointed with the rest.

### AFTER — every test path resolves to `:5434` / `hqtest`, the test-only container

`EXIT=0` · full capture: `.night-crew/runs/2026-08-07-autonomous/w0-logs/redfirst-after.log`

```
== 1. Playwright e2e coordinates, RESOLVED by scripts/reset-e2e-db.js (no connection) ==
  adminUrl (DROP/CREATE target): postgres://hqtest:hqtest@localhost:5434/postgres?sslmode=disable
  testUrl                      : postgres://hqtest:hqtest@localhost:5434/hq_test_e2e?sslmode=disable&TimeZone=America/New_York
== 2. task test:* DB coordinates (Taskfile.yml) ==
45:  TEST_DB_HOST: '{{.TEST_DB_HOST | default "localhost"}}'
46:  TEST_DB_PORT: '{{.TEST_DB_PORT | default "5434"}}'
   … every test:* env block + backend:db-test dep now interpolates {{.TEST_DB_*}} …
223:      DB_TEST_URL: 'postgres://{{.TEST_DB_USER}}:{{.TEST_DB_PASS}}@{{.TEST_DB_HOST}}:{{.TEST_DB_PORT}}/hq_test_go?sslmode=disable'
267:          psql "postgres://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/postgres?sslmode=disable" \
== 3. RLS suite defaults (backend/internal/sync/rowvisibility_rls_test.go) ==
243:	defaultHQAdminURL = "postgres://hqtest:hqtest@localhost:5434/postgres"
251:	defaultFDWPort = "5434"
== 4. harness probe URL (scripts/verify-test-harness.sh) ==
215:	DEAD_URL='postgres://hqtest:hqtest@127.0.0.1:5599/hq_test_go_does_not_exist?sslmode=disable'
218:	DEAD_URL='postgres://hqtest:hqtest@127.0.0.1:5434/hq_test_go_dropped_by_a_reviewer?sslmode=disable'
```

Corroborated by `task test:targets` (`w0-logs/test-targets.log`, `EXIT=0`) and — the strongest
evidence, because it is the harness resolving at runtime rather than a render — by the first line
of the Playwright gate log:

```
[WebServer] ── reset hq_test_w0 on localhost:5434 ──
```

Tree-wide check after the change: `git grep` finds **no `yumyums:yumyums` and no `:5433` in any
test-path default**. The `:5433` sites that remain are `backend/Taskfile.yml`'s dev/migrate/
db-ping coordinates (which are *supposed* to name the dev cluster), `sync-schema/sql/0002_hq_fdw.sql`'s
operator-facing GUC fallback (the RLS suite always sets `hq_fdw.port` explicitly, so the test path
never reads it), and prose. The `:5432` fallbacks inside seven `TestMain`s are unchanged and are
not a production path — `:5432` is `infra-postgres-1`, and those DSNs are only reached when
`DB_TEST_URL` is unset, which is the deliberate skip case.

**Standing rule 1 attestation:** nothing in this card opened a connection to `:5433`. The
Playwright gate log's only four matches for the string `5433` are fractional-second digits inside
timestamps (`10:11:40.116755433-04:00` and three like it), not port references — verified line by
line.

---

## Gate results

All gates run in this worktree, `export PATH="/usr/local/go/bin:$PATH"` on every Go/Playwright leg.

| Gate | Result | Evidence |
|---|---|---|
| **G1 build** | `go build ./...` from `backend/` — **exit 0**, no output | `w0-logs/g1-build.log` |
| **G1 vet** | `go vet ./...` from `backend/` — **exit 0**, no output | `w0-logs/g1-vet.log` |
| **G2 Go** | **exit 0** · **9 packages with tests** (alerts, auth, inventory, purchasing, receipt, recipes, sync, toast, workflow) · **246 top-level tests, 456 including subtests** · **0 failures** · 2 skips, both `TestProxyLive_*` behind the `HQ_SYNC_SPIKE_LIVE` opt-out · `internal/workflow` = **35** (the expected figure exactly) · `internal/sync` = 47 top-level / 111 subtests, with `TestRowVisibilitySubtestCount_Structural`, `TestRVTopLevelSubtestCount_CountsWhatItClaims` and `TestRowVisibilitySubtestCount_Executed` (48.5s) all **PASS** — the suite's own count assertion held against the new cluster | `w0-logs/g2-go.log` |
| **G2 Playwright** | **exit 0** · **`791 passed (26.1m)` · `6 skipped` · 0 failed · 0 flaky** · **exactly ONE summary block** over the complete log (`grep -c "^ +[0-9]+ passed \("` = 1) · slowest file `sync.spec.js` 5.5m | `w0-logs/g2-playwright.log` (4536 lines, whole log committed) |
| **G4** | `node build-sw.js` **exit 0** twice · precache **31 files** (2167.0 KB), import reachability 18 parsed / 30 resolved / 0 outside · **idempotent** — `sw.js` unchanged after both runs, tree clean · parity **1.4.0 ≡ 1.4.0 ≡ 1.4.0** (`version.go` Frontend / `package.json` / `version.json`; Backend 0.3.0 untouched) | `w0-logs/g4-sw-1.log`, `g4-sw-2.log`, `g4-parity.log` |
| **RF** | Complete — before/after captures above | this section |

### Go run — env attestation

Measured with `env` and per-variable expansion **before** the run, not assumed:

```
HQ_SYNC_SUBSTRATE_OPTIONAL=[<UNSET>]   HQ_SYNC_GATE_CHILD=[<UNSET>]
DB_TEST_URL=postgres://hqtest:hqtest@localhost:5434/hq_test_go?sslmode=disable
HQ_RLS_TEST_DB=hq_rls_test_w0
```

Both gate-defeating variables were **UNSET**. `DB_TEST_URL` was set and pointed at the new
container, so the DB-coupled packages ran rather than silently skipping. The RLS suite really
executed — it created `hq_rls_test_w0` **on the test cluster**, and the substrate container reached
it over `host.docker.internal:5434`, which is the proof that publishing 5434 on all interfaces was
the right call.

### Playwright run — parameters

`npx bddgen` first (exit 0, `.features-gen/` populated), then
`TEST_PORT=4517 TEST_DB_NAME=hq_test_w0 HQ_RLS_TEST_DB=hq_rls_test_w0 npx playwright test --retries=0`.

🛑 `DB_HOST`/`DB_PORT`/`DB_USER`/`DB_PASS` were **deliberately left unset** so the run exercised the
new **defaults** in `scripts/reset-e2e-db.js` rather than an env override. The reset banner proves
they resolved: `── reset hq_test_w0 on localhost:5434 ──`.

### Armed reds observed

- **B-27 / B-30 / B-32** (`sync.spec.js` load-sensitive flakes): **did not fire.** 0 flaky, 0 retries.
- **B-132**: present as expected — 2 482 `loadMyChecklists error: Failed to fetch` client-log lines
  from the server. 🛑 This is materially more than the "fires 28×/run" figure in the launch prompt.
  It changed nothing about the verdict (no test asserts on it and nothing red), and this card did not
  touch the code that emits it, but the figure the next card inherits should be the measured one, not
  28. Flagging rather than silently re-baselining.
- 6 skips, all pre-existing PARKs/skips (S3-dependent upload tests in `inventory`/`onboarding`, one
  `persistence` recipe test, three `purchasing` tests).

### Wall-clock

**26.1m for 797 tests (791 passed + 6 skipped)**, against the inherited baseline of ~24.2m for 785.
Recorded, not discarded: the tree carries 12 more tests than the baseline figure, and the first
~22 minutes of this leg overlapped nothing — but see the observation below about the box.

---

## Observation for triage — a foreign suite was running against `:5433` mid-card

🛑 Not caused by this card, and reported because standing rules 1 and 2 make it material.

At the point W0 was ready to run its Playwright leg, a **second Playwright suite was already in
flight from the main checkout** `/home/jcole/projects/hq` (pid 95177, started 09:27, ~29m
elapsed). Its webServer command was, verbatim from `ps`:

```
PORT=8199 DB_URL="postgres://yumyums:yumyums@localhost:5433/hq_test_e2e?sslmode=disable&TimeZone=America/New_York" … go run ./cmd/server/
```

— i.e. a pre-W0 tree running the harness against **the cluster that serves
`hq.yumyums.kitchen`**, `DROP DATABASE hq_test_e2e` included. The `production` database itself was
never named, so the blast radius was a test database on the production cluster, but it is the exact
posture decision 155 exists to end, and it was live while Wave 0 was landing the fix.

**What W0 did about it:** nothing destructive. It was not this card's process to kill, and killing
it would have destroyed another run's evidence. W0 **waited** for it to exit rather than break the
global one-suite lock, then ran its own leg on a clear box. The wait is most of the gap between
this card's elapsed time and its work time.

**Identified after the fact.** That run has since landed on `dev` as **`eb8e415`** — *"docs(nc):
T-39 addendum — de-confinement gate green (791 passed, 6 skipped) on the dev tip"*. So it was a
deliberate de-confinement gate on the dev tip, not a stray process. Two consequences:

1. **Standing rule 2 was not honoured by it** — a suite ran, on the shared production cluster,
   before W0's container existed. Worth a triage note, since rule 2 is the reason W0 is Wave 0.
2. **It is unexpectedly good corroboration.** That leg measured **791 passed / 6 skipped** on the
   dev tip against `:5433`; this card measured **791 passed / 6 skipped** against `yumyums-test-pg`
   on `:5434`. Identical figures, different clusters — which is exactly the evidence that the
   re-point moved *where* the suite runs and changed *nothing* about what it proves.

## Branch state at hand-off

`dev` moved forward by one commit while this card ran (`eb8e415`, the addendum above —
`ledger.md` only). This branch does **not** touch `ledger.md`, so the two are disjoint and the
merge is clean. The branch was deliberately **not rebased**, so the SHAs reported to the
orchestrator stay the SHAs on the branch.
