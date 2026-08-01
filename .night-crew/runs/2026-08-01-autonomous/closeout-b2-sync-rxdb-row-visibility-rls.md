# Closeout — B2 `sync-rxdb-row-visibility-rls` · **LANDED**

Run `overnight-20260801`, branch `card/b2-sync-rxdb-row-visibility-rls`.
Resumed from the 2026-07-29 park; **that park is now discharged, not worked around.**

Merged `overnight-20260801` into the card branch first: **fast-forward,
`33bd8c0..01107fc`, zero conflicts** (planning documents only).

---

## 1. The PARK trigger was checked FIRST, and did not fire

The card's trigger was: *park if `postgres_fdw` cannot be made to work across the two
servers as the port assumes.* That was the first thing measured, before a line was
written, because parking after building is expensive and parking after asserting is
worse:

```
docker exec spike-supabase-db-1 \
  psql postgres://yumyums:yumyums@host.docker.internal:5433/yumyums \
  -tAc "select current_database(), (select count(*) from users)"
→ yumyums|5
```

The substrate **container** reaches HQ's Postgres. The extension was already proven
installable at both ends by the park note's own probe. Nothing operator-only surfaced:
the accepted network-path cost was recorded at sign-off, so there was no new question
to take back.

One thing the resumption changed for free, which the park note had filed as an open
gap (§4d): **`submission_responses` now has a ported path.** Under the rejected
projection design, `checklist_fields`/`checklist_sections` do not exist on the
substrate and a third projection would have needed a writer. Reading through, the
tables are simply there — `hq_sync_field_templates` is a view, not a projection.

## 2. What shipped

| File | Side | What |
|---|---|---|
| `backend/internal/db/migrations/0073_sync_fdw_views.sql` | HQ | Three read-through **views** + the `hq_sync_fdw` role |
| `sync-schema/sql/0002_hq_fdw.sql` | substrate | extension, server, mapping, three foreign tables, **revoked from every PostgREST role** |
| `sync-schema/sql/0003_rls_policies.sql` | substrate | `hq_can_see_template` / `hq_can_see_field` + SELECT policies |
| `backend/internal/sync/rowvisibility_rls_test.go` | test | 27-subtest attack suite (19 numbered variants) |
| `.night-crew/qa/spike-supabase/captures/{red,green}-20260801-row-visibility.txt` | evidence | red-first captures |

**The port was NOT redone.** The transposition in the park note §4b was applied as
written; the only two additions are `hq_can_see_field` (the `field_response`
resolution step, which the park note recorded as an open gap) and the fact that
`hq_template_assignees` / `hq_user_roles` are now **foreign tables** rather than
locally-written projections — which is exactly what decision 92 asks for and is why
there is no writer to build.

### Three design calls worth reading before merging

**(a) Views, not `IMPORT FOREIGN SCHEMA` on the base tables.** `public.users` carries
`password_hash`. A foreign table over `users` would put that column on the wire to a
second server. The three views are the narrowest surface that answers the predicate:
no email, no phone, no salary, no hash. The remote role holds SELECT on the views and
**no privilege on any base table**, so the substrate cannot widen its own reach by
asking for a different relation.

**(b) The role ships NOLOGIN with no password.** The migration runs in every
environment including production; a password in it would be a committed credential
shared by all of them. So the migration creates the role and the privilege surface
— the parts that must not drift — and stops. `ALTER ROLE hq_sync_fdw LOGIN PASSWORD
'<generated>'` is a deliberate per-environment step, and an environment that has not
been consciously wired for sync **fails closed** rather than accepting a default. The
suite performs that step against its own throwaway database and restores NOLOGIN on
cleanup, so the step is demonstrated rather than described.

**(c) 🛑 Prod runs in the `production` schema, and two hard-coded `public`s would have
been wrong there and nowhere else.** Found by reading `docker-compose.prod.yml`
(`DB_URL: ...&search_path=production`), not by a failing test — **no test in this repo
runs against a `production` search_path.** 0073's views are created unqualified like
every other migration, so they land in `production` on prod. The `GRANT USAGE ON
SCHEMA public` would have granted the wrong schema *silently* (migration succeeds,
views exist, remote role cannot see them) and the foreign tables' `schema_name
'public'` would have failed as *"relation does not exist"* inside a policy, at request
time, far from the file. Both now follow `current_schema()` / a `hq_fdw.schema` GUC,
and the prod path was verified end to end against a database with
`CREATE SCHEMA production` + `search_path=production`.

## 3. Two decisions triage must rule on

**(a) SELECT policies only. INSERT/UPDATE stay policy-less — deny-all.**
`ResolveEntityAccess` is a **fan-out** resolver: it answers *"who should RECEIVE this
op"*, a read-visibility question. Reusing its predicate to decide who may WRITE would
invent a permission semantic no shipped code asserts (*who may create a template? who
may edit someone else's answer?*), and the card forbids exactly that. Writes stay
refused by policy-absence, precisely as B1 left them — no regression, no new door, and
V10/V11 prove the refusal rather than assuming it.

> 🛑 **The consequence C1/C2 need to hear: RxDB push replication is refused until a
> follow-up card writes `WITH CHECK` policies.** Nothing is live tonight
> (`HQ_SYNC_REST_URL` unset), so this blocks nothing that ships — it is stated here so
> it is discovered at merge rather than at first push.

**(b) `submission_rejections` keeps no policy.** `ResolveEntityAccess`'s switch has no
case for it and falls through to `return []string{}`. A policy there would be an
**extension**, not a port. The minimal rule ("visible iff its submission is") is
structurally available — `submission_id` is `text not null` — but **structural
availability is not authority**, and nothing in shipped code asserts it. V18 proves the
deny-all with an admin *and* a `service_role` control, so this is a decision with
evidence rather than a gap nobody noticed.

## 4. `HQ_SYNC_REST_URL` — unchanged

This branch does not set, add or reference it in any deploy, config, test fixture or
compose file. Audited across the tree: every hit is prose, a constant naming the env
var (`proxy.go:111`), or a warning comment. **The interlock disarms at triage on
evidence, never by this card asserting it.** This card is what supplies the evidence;
it does not draw the conclusion.

## 5. Guard integrity — the card's central risk, discharged by measurement

This card's whole deliverable is a check, and B-22/B-23/B-24 say a guard printing PASS
is not evidence until its subject set is shown non-empty. So the guard was **broken on
purpose three times** and observed catching each:

| Mutation | Caught by |
|---|---|
| `WHERE ta.assignment_role = 'assignee'` added to 0073's view — the exact "tightening" the comments warn against | 9 subtests, **`POSITIVE/alice` by name**, plus both population floors (the filter drops the view 4 rows → 3, under the floor) |
| admin arm deleted from `hq_can_see_template` | **exactly 3**: `POSITIVE/carol`, `V12`, `V14` — the admin-dependent assertions and no others |
| 🛑 foreign server repointed at a **migrated-but-empty** database | 13 FAIL / 14 PASS. `rvAssertFDWPopulated` FATALS with a diagnostic naming the cause — **and 12 of the 19 numbered attack variants still PASS** (V1–V6, V10, V11, V15–V18) |

The third is the one that matters. An unreachable server *raises*; a wrong-database
mapping returns a **calm empty set**, measured directly:

```
alter server hq_pg options (set dbname 'hq_b2_empty');
select count(*) from public.hq_user_roles;   →  0     (no error)
```

🛑 **The finding generalises past this card, and is stated precisely because a first
draft of it overstated it.** Twelve of nineteen numbered attack variants pass against a
permission system that has silently stopped reading anything. The seven that fail all
fail on their **positive half**, never on their refusal. So: **a refusal-only variant is
blind to an empty subject set** — *"the attacker saw nothing"* is satisfied perfectly by
a system that shows nobody anything. What catches it is an assertion that DEMANDS ROWS:
the four positives, the two population floors, the two BYPASSRLS controls. A suite of
pure attack variants, however long, would have printed green here. That is this repo's
characteristic bug arriving through the one mechanism this card added, which is why the
floor asserts on **both sides of the wire** rather than only on HQ.

## 6. Gates — raw exit codes

| Gate | Command | Exit | Counts |
|---|---|---|---|
| G1 build | `go build ./...` | **0** | — |
| G1 vet | `go vet ./...` | **0** | — |
| G2 Go | `go test -p 1 ./...` | **0** | 11 packages `ok`, 0 fail. `internal/sync` 105 PASS / 4 SKIP; `TestRowVisibilityRLS` 27/27 |
| G2 Playwright | `npx bddgen` then `npx playwright test --retries=0` | **0** | **591 passed / 6 skipped / 0 failed** in 32.5m across **21 spec files** (20 static in `tests/` + 1 generated by `bddgen`). 591 + 6 = 597, the announced total. |
| G3 openspec | — | **N/A** (`openspec: absent`) | — |
| G4 sw idempotence | `node build-sw.js` ×2 | **0, 0** | identical md5 `2ee7c220…`; `git status` clean |
| G4 version parity | — | — | `Frontend = "1.2.2"` ≡ `package.json 1.2.2` ✓ |
| G4 nc discipline greps | `internal/journal`, `internal/workorder` | — | **N/A-VACUOUS, not clean** (B-14) — neither directory exists in this repo |

**DB liveness proved, not assumed** (`hq_test_go`): `tables=51`, `goose_version=73`,
`hq_sync_views=3`.

🛑 **The known pre-existing red did NOT reproduce, and that is reported as a
non-reproduction, not as a fix.** `tests/inventory.spec.js:883` (*"item modal pre-fills
search with current line item text"*, B-27 cross-spec pollution, normally masked by
`retries: 1`) **ran at 81/597 and passed** at `--retries=0`. This card changed nothing
in its path, so the honest reading is that the ordering that provokes it did not occur
this run — **the flake is unreproduced, not resolved.** Do not let a green here retire
B-27.

The armed red was also verified present by grep handle, not by line anchor (decision
100): `list page progress decrements when another device unchecks a field [LST-17]` is
live at `tests/sync.spec.js:446`, and the bare tag `[LST-17]` does match two tests, as
the slate says.

The 4 `internal/sync` skips are the two pre-existing opt-in live proofs
(`TestProxyLive_*`, gated on `HQ_SYNC_SPIKE_LIVE=1`), which say in their own skip
message that *skipped is not passed*. `TestJWTBridgeRLS` **ran and passed 16/16**,
including both `service_role` BYPASSRLS controls.

## 7. Notes for the orchestrator at merge time

1. **Migration `0073` was unclaimed and is taken cleanly.** Latest on the merged base
   is `0071`; A1 holds `0072`. Up and Down were both executed against a scratch
   database and re-applied.
2. **No `sw.js` conflict from this card.** It ships no frontend file. `sw.js` and
   `version.json` are byte-identical to the base after two `build-sw.js` runs.
3. **No `version.go` change**, so no per-constant merge needed from this side.
4. **`.night-crew/knowledge/roadmap.md` is the only shared file touched** — one
   bullet. Resolve per-bullet.
5. **The spike Supabase stack was DOWN at the start of this run** and was brought up
   (`docker compose -p spike-supabase … up -d`). Its host ports are **ephemeral by
   design** and are now `db → 51737`, `rest → 51717` — so the defaults hard-coded in
   `jwtbridge_rls_test.go` (46011/46233) are stale, and both RLS suites **skip
   silently** without `SPIKE_DB_URL`/`SPIKE_REST_URL` exported. That is a pre-existing
   sharp edge this card inherited rather than introduced, but it is now load-bearing
   for two suites instead of one. **Backlog candidate:** have the suites resolve the
   port from `docker compose port` instead of a constant, so an unexported env var
   fails loud instead of skipping.
6. **Two scratch databases were created and dropped** on HQ's cluster during
   verification (`hq_b2_probe`, `hq_b2_empty`, `hq_b2_prodschema`) — all dropped.
   `hq_test_b2_fdw` is the suite's own and is recreated on every run.
   `hq_sync_fdw` was left **NOLOGIN with no password**, its shipped state.
