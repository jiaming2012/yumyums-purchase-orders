# Merge intent — B2 `sync-rxdb-row-visibility-rls`

Run `overnight-20260801`, branch `card/b2-sync-rxdb-row-visibility-rls`,
worktree `/home/jcole/projects/hq-worktrees/b2-sync-rxdb-row-visibility-rls`.

Written **before** implementing, per the run's per-change mechanics. Merged
`overnight-20260801` into this branch first: **fast-forward, zero conflicts**
(`33bd8c0..01107fc`, docs/planning files only).

---

## Shared files I touch — each with one line of why

| File | Mine alone? | Why I touch it |
|---|---|---|
| `backend/internal/db/migrations/0073_sync_fdw_views.sql` | **new file, number ASSIGNED to me** | The HQ-side half of the fdw: three read-only views + the least-privilege role the substrate logs in as. `0073` verified unclaimed — latest on the merged base is `0071`, and A1 holds `0072`. |
| `backend/internal/sync/**` | **YES — mine alone tonight** | New attack-variant suite lives here. The slate states C1 touches `backend/Dockerfile`, not `internal/sync`. |
| `sync-schema/sql/0002_hq_fdw.sql` | **new file** | Substrate-side fdw wiring (extension, server, user mapping, foreign tables). New number in a directory B1 created and left at `0001`. |
| `sync-schema/sql/0003_rls_policies.sql` | **new file** | The policies themselves. Deliberately a **separate file from `0002`** so the red state is reproducible by applying `0002` and not `0003` — the same split discipline `hq-bridge-fixture.sql` / `hq-bridge-policies.sql` uses. |
| `.night-crew/knowledge/roadmap.md` | **NO — every card touches it** | The card's status flip, required in the same change set. **One bullet only.** Expect a conflict; resolve per-bullet, never per-file. |
| `.night-crew/runs/2026-08-01-autonomous/*-b2-*.md` | **YES** | This note and the closeout. Card-scoped filenames; no other card writes them. |
| `.night-crew/qa/spike-supabase/captures/{red,green}-20260801-row-visibility.txt` | **YES** | *Added to this table after the fact, once the shape of the evidence was known — flagged rather than backdated.* The red/green captures, in the directory `sync-jwt-bridge-endpoint` established. Date- and card-scoped filenames; additive only, no existing capture touched. |

### Files I do NOT touch — stated so absence reads as considered

- **`workflows.html`, `sw.js`, `package.json`, `version.go`** — nothing here. This card
  ships no frontend and no user-visible version change. I take **no side** in the
  A1 → C1 → C2 landing order and I create **no** `sw.js` conflict.
- **`tests/sync.spec.js`** — nothing here. My suite is Go, in its own file, per the
  slate's explicit instruction that B2's suite does not belong in that spec.
- **`backend/Dockerfile`** — C1's. Untouched.
- **`sync-schema/sql/0001_sync_tables.sql`** — B1's, read-only for me. My policies are
  additive in `0003`; I do not edit B1's file, so `tests/sync-schema.spec.js`'s
  assertion that `0001` contains no `CREATE POLICY` **stays true and stays meaningful**.
- **`.night-crew/qa/spike-supabase/sql/*`** — the `sync-jwt-bridge-endpoint` fixtures.
  Read and reused as a model; **not edited**. `hq_uid_trap` keeps re-proving its finding
  under the suite that already owns it.

---

## What must survive any merge

1. **`backend/internal/db/migrations/0073_sync_fdw_views.sql` — whole file, unmodified,
   at that exact number.** If a merge finds `0073` taken, that is a stop-and-report, not
   a renumber. The substrate's foreign tables name these views; a renumber that changes
   nothing else is harmless, but a *collision* means two migrations claim one version and
   goose will refuse.
2. **The two-file split in `sync-schema/sql/`.** `0002` (fdw, no policies) and `0003`
   (policies) must not be merged into one file. Squashing them destroys the red-first
   reproduction — `SYNC_RLS_SKIP_POLICIES=1` works precisely because `0003` can be
   withheld while `0002` is applied.
3. **Every `🛑` comment block in the two SQL files and in the Go suite.** They are not
   decoration: they record *why* `auth.uid()` is wrong here, why a 200 proves nothing,
   why the `service_role` control cannot be deleted, and which of the two inherited
   permission properties are knowing rather than accidental. A merge that keeps the SQL
   and drops the comments keeps the mechanism and loses the reason.
4. **The absence of INSERT/UPDATE policies on the four B1 tables.** See the note below —
   this is a decision, and a merge that "helpfully" adds them reverses it silently.
5. **The roadmap bullet's status flip.**

## What is safe to drop

- **The prose of this note itself**, once triage has read it.
- **Any whitespace/ordering resolution inside `.night-crew/knowledge/roadmap.md`** — only
  my one bullet's text is load-bearing.
- **The `SYNC_RLS_*` env-var defaults baked into the Go suite** (`51737`/`51717` etc. are
  this box's current ephemeral compose ports and will differ on any other box). They are
  documented defaults behind `env()` overrides, not configuration. If a merge needs to
  change them, change them.
- **Nothing else.** There is no other droppable content in this change set.

---

## Two things the orchestrator must know at merge time

**1. `HQ_SYNC_REST_URL` is still not set by this branch, and this branch does not set it.**
I add no env var, no compose entry, no config key, no test fixture that sets it. The
interlock stands exactly as B1 and `sync-proxy-endpoint` left it. Whether it *disarms*
is triage's call on evidence, not mine.

**2. This card ships SELECT policies only — INSERT/UPDATE stay policy-less (deny-all),
deliberately.** `ResolveEntityAccess` is a *fan-out* resolver: it answers "who receives
this op", which is a read-visibility question. Extending its predicate to govern writes
would be inventing a permission semantic ("who may create a template?"), and the card
says in terms: do not vary substrate and permission semantics in one night. So writes
remain refused by policy-absence, exactly as B1 left them — no regression, no new door.
**The consequence C1/C2 need to hear: RxDB push replication will be refused until a
follow-up card writes `WITH CHECK` policies.** Nothing is live tonight
(`HQ_SYNC_REST_URL` unset), so this blocks nothing that ships; it is stated here so it is
discovered at merge rather than at first push.

---

## Addendum — fix pass for F2 / F1 / F4 (same night, after adversarial review)

Three defects closed on this same branch after the card passed G6. No new
shared file enters the change set; two rows above change meaning and one
"safe to drop" bullet is now wrong. That is the whole delta.

### Files touched by the fix pass

| File | Already in the table above? | What changed |
|---|---|---|
| `backend/internal/db/migrations/0073_sync_fdw_views.sql` | **yes** | **F2.** All explicit `BEGIN;`/`COMMIT;` removed (the inner `COMMIT` was committing *goose's* transaction early), and the `DROP ROLE` is now interlocked on `pg_shdepend`. |
| `backend/internal/sync/spikestack_gate_test.go` | **new file, inside `backend/internal/sync/**`** | **F1.** The substrate resolver + the asymmetric gate. Same tree the table already declares mine alone tonight. |
| `backend/internal/sync/{jwtbridge_rls,rowvisibility_rls,proxy_live}_test.go` | **yes** (`backend/internal/sync/**`) | **F1.** Wired to the resolver; hard-coded port constants deleted. |
| `sync-schema/sql/0002_hq_fdw.sql` | **yes** | **F4.** `service_role` added to §4's revoke list, with the measurement and the reason for excluding `postgres`. |

### 🛑 One "safe to drop" bullet above is now FALSE

> *"The `SYNC_RLS_*` env-var defaults baked into the Go suite (`51737`/`51717`
> etc.) … If a merge needs to change them, change them."*

**They are gone, and they must stay gone.** They were the F1 defect: the
compose file publishes ephemeral ports, so any committed default is wrong on
the next `up`, and the suites skipped silently — printing `ok` while the card's
entire deliverable ran zero assertions. Ports are now resolved with
`docker compose port`. **A merge that reintroduces a port constant in
`backend/internal/sync/` reintroduces the bug.**

### Added to "what must survive any merge"

6. **No explicit transaction control in `0073`.** goose already wraps each
   migration; an inner `COMMIT` there is what let a failed Down leave
   `is_applied = true` with the views already dropped and `goose up` refusing to
   repair it. Do not "restore" the `BEGIN;`/`COMMIT;`.
7. **The `pg_shdepend` interlock in `0073`'s Down.** `hq_sync_fdw` is
   cluster-wide and prod shares a cluster with dev and every `hq_test_*`
   database (`docker-compose.prod.yml:41`). An unconditional `DROP ROLE` either
   errors or — worse — succeeds and breaks every other database's substrate.
8. **The asymmetric gate in `spikestack_gate_test.go`.** Configured-but-dead
   must FAIL, never skip. `TestSpikeGate_Asymmetry` pins the row.
9. **`service_role` on `0002` §4's revoke list.** Measured non-empty default
   privileges (`service_role=arwdDxt/supabase_admin`) on the supabase image, and
   `authenticator` is a member of `service_role`.

### Not touched by the fix pass, stated so absence reads as considered

- **`sync-schema/sql/0003_rls_policies.sql`** — untouched. In particular
  `hq_can_see_template`'s two arms are exactly as G6 approved them.
- **`HQ_SYNC_REST_URL`** — still not set anywhere by this branch, fix pass
  included.
- **F3 (the ~23 ms/row read-through cost)** — deliberately NOT addressed. It is
  an operator question about accepted network-path cost, recorded for morning
  triage; nothing in the fix pass changes the design that produces it.
