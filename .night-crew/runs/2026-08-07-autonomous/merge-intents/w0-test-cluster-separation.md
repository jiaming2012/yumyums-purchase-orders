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

### AFTER — pending; evidence lands in a later commit on this branch

---

## Gate results

Pending — filled in below before the last commit on this branch.
