# 🛑 PARK — Card B2 `sync-rxdb-row-visibility-rls`

Run `overnight-20260729-2`, branch `card/b2-sync-rxdb-row-visibility-rls`
(worktree preserved at `/home/jcole/projects/hq-worktrees/b2-sync-rxdb-row-visibility-rls`).

**The card's own PARK trigger fired. No production code, SQL, policy or migration
was written. The `HQ_SYNC_REST_URL` interlock is unchanged and still in force.**

> **Repair round — this note was corrected after a G6 verification pass.** G6
> re-executed every probe on both servers and found the park **CORRECT**; the verdict
> is unchanged and the card stays parked. But it found four inaccuracies in this
> note, and a note whose authority is *"measured, not assumed"* cannot carry an
> extrapolation. Four corrections were applied and are marked where they land:
> **C-1** (§3 E4 + §5 option (a) — the FDW extensions were never measured at the HQ
> end and are *installable*, not merely available, at **both**); **C-2** (§5 new
> option (b′) — native logical replication); **C-3** (new §2b — this note silently
> substituted `template_assignments` for the `app_permissions` the card and decision
> 61 name); **C-4** (§6 — one sentence was false and is struck in place).
> Struck text is left visible on purpose: what the note got wrong is part of what
> triage is reading. §7 records the provenance of every first-hand claim.

---

## 1. The trigger, verbatim, and why it fired

> 🛑 **Riskiest card tonight — PARK note:** if porting `ResolveEntityAccess` reveals
> that the projection **cannot** be written in the same transaction as the
> `app_permissions` mutation without restructuring that mutation, **PARK**.
> Decision 61 chose push-on-change over reconcile for a stated reason; restructuring
> the grants write path to honour it is a different card, and **improvising a
> reconcile fallback silently reintroduces the replay window.**

It fired, and for a reason stronger than the one the trigger anticipated. The
trigger imagines the obstacle is the *shape* of the mutation. The actual obstacle is
that **the projection and the mutation are in two different database servers**, so
there is no transaction that can contain both — no restructuring of the mutation
changes that.

The predicate to port is unambiguous and was fully read (§4 below). What has no
answer tonight is **how the table the predicate reads stays live.**

## 2. The load-bearing fact, and the card that established it

RLS policies evaluate inside the **sync substrate's Postgres**. Every table the
projection is derived from — `template_assignments`, `users`, `checklist_fields`,
`checklist_sections` — lives in **HQ's Postgres**. Two servers.

This is not an inference. It is the explicit, signed position of this card's own
dependency, **B1 `sync-rxdb-collections-and-table-contract`, merged tonight**
(`95a2657`), stated in the header of `sync-schema/sql/0001_sync_tables.sql`:

> ⚠ THIS FILE DOES NOT RUN AGAINST HQ's POSTGRES. It targets the SELF-HOSTED
> SUPABASE Postgres (`docker-compose.supabase.yml`'s `db` service) — the sync
> substrate, not HQ's own database. It is deliberately NOT a goose migration under
> `backend/internal/db/migrations/`, because anything placed there runs against HQ's
> database on every backend start, and these tables have no business being there:
> `authenticated`/`anon` do not exist as roles in HQ's Postgres, and
> `supabase_realtime` is not a publication it has.

**This is the timing that matters, and morning triage should see it plainly.**
Decision 61 (T-23, 2026-07-26) answered D-4 with "same transaction as the
`app_permissions` mutation." D-4's own text had said the choice *"is only meaningful
once the sync database's relationship to HQ's own database is settled — which is the
cutover card's subject, not this one's."* That relationship was settled **three days
later, tonight, by B1** — and it settled in the direction that makes decision 61's
contract unsatisfiable. Decision 61 is not wrong about *what is wanted*; it is a
contract written against a topology that had not yet been chosen.

## 2b. A substitution this note makes — flagged, not hidden (C-3)

**The card text and decision 61 both name the `app_permissions` mutation. The
projection this card actually needs is not fed by `app_permissions`. It is fed by
`template_assignments`.** Everything from §1 onward silently adopted the latter. It
is named here so nobody discovers the swap by reading two documents side by side at
3am.

- The park trigger, verbatim, says *"cannot be written in the same transaction as the
  **`app_permissions`** mutation."*
- The predicate being ported (`ops.go:474`, §4a) reads `template_assignments ⋈ users`.
  It does not read `app_permissions` at all.
- The mutation that would therefore have to be transactional with the projection is
  `backend/internal/workflow/repository.go:236` (`DELETE FROM template_assignments
  WHERE template_id = $1`) and `:249` (the wholesale re-insert) — **not** any
  `app_permissions` write.

**This does not change the conclusion.** The topology is identical for both tables:
`app_permissions` is likewise on HQ's Postgres (5433) and likewise absent from the
substrate (46011) — measured, §3 E6/E6b. Whichever table you name, the projection
write and the mutation are on opposite sides of a server boundary and no transaction
spans them.

**What it does change is whose text applies.** Decision 61's literal wording may have
been written about a *different table* than this card needs. So triage has a prior
question before it touches the §5 fork:

> **Does decision 61 govern this card by letter, or only by analogy?**
> By letter: it names `app_permissions`, this card's projection is fed by
> `template_assignments`, and a contract about one table does not automatically bind
> the other — in which case the fork below is freer than it looks.
> By analogy: 61's *reason* (push-on-change over reconcile, no replay window) is about
> the class of grant-shaped projections, and `template_assignments` is plainly in that
> class — in which case the fork is exactly as constrained as §5 describes.

**That is an operator call and is stated here, not made here.** This note assumes
neither reading. The §5 options are written to be legible under both.

## 3. Evidence — measured on this host, not assumed

HQ Postgres `127.0.0.1:5433` (`yumyums-dev-pg`); sync substrate `127.0.0.1:46011`
(`spike-supabase-db-1`, up 3 days, healthy). Verbatim results:

| # | Question | Where | Result |
|---|---|---|---|
| E1 | Does HQ's Postgres have `anon`/`authenticated`/`service_role`/`authenticator`/`supabase_admin`? | 5433 | **0 of 5 present** |
| E2 | Does the sync Postgres? | 46011 | **all 5 present** |
| E3 | Does the sync Postgres hold `template_assignments`, `users`, `checklist_fields`, `checklist_sections`? | 46011 | **0 of 4 present** |
| ~~E4~~ | ~~Are `postgres_fdw` / `dblink` installed?~~ **STRUCK (C-1)** — measured one end only, and understated the state. Replaced by E4a–E4c. | ~~46011~~ | ~~**both NOT INSTALLED** (available, uninstalled)~~ |
| E4a | Are `postgres_fdw` / `dblink` **installed**? | **5433 AND 46011** | **Neither, at either end.** `pg_extension` on 5433 holds only `plpgsql`; on 46011 the query for the two names returns empty. |
| E4b | Are they **available** (control file present)? | **5433 AND 46011** | **Both, at both ends.** `pg_available_extensions`: `dblink 1.2`, `postgres_fdw 1.1` — identical versions on both servers. |
| E4c | Does `CREATE EXTENSION` **actually succeed**? | **5433 AND 46011** | **YES — all four combinations.** Not inferred from the control file: executed. |
| E5 | `max_prepared_transactions` (2PC capability) | 46011 | **0** |
| E5b | `max_prepared_transactions` | 5433 | **0** |
| E6 | Does HQ's Postgres hold `app_permissions` (the table decision 61 names)? | 5433 | **present** (`to_regclass` → `app_permissions`) |
| E6b | Does the substrate? | 46011 | **ABSENT** — same topology as the four §2 tables |
| E7 | `wal_level` (native logical replication capability — for option (b′)) | 5433 / 46011 | **`replica` on HQ, `logical` on the substrate.** Publisher side needs a bump + restart; subscriber side is already there. |

### E4 in full, because it is the correction that moves a decision (C-1)

The struck E4 row listed `46011` in its "Where" column and read "available,
uninstalled." Two things were wrong with it. **It never measured HQ at all** — yet §5
option (a) asserted the extensions were "NOT INSTALLED at *either* end (E4)," which
was an extrapolation from a one-ended probe. And "available" understates what is
true: an available control file does not prove the shared object is present or that
the extension will install.

Both were re-measured here, on both servers, **by executing the DDL rather than
reading a catalog**:

- On HQ (`127.0.0.1:5433`, PostgreSQL 16.13, in this card's own `hq_test_go_b2`) and
  on the substrate (`spike-supabase-db-1`, PostgreSQL 15.8, in `postgres`):
  `BEGIN; CREATE EXTENSION dblink; CREATE EXTENSION postgres_fdw; … ROLLBACK;`
- **All four `CREATE EXTENSION` statements returned `CREATE EXTENSION`** —
  `dblink 1.2` and `postgres_fdw 1.1` on both servers.
- The `.so` genuinely loads; the C symbol was not merely declared but **executed**:
  `SELECT dblink_get_connections()` returned `0` open connections on both servers, and
  `CREATE SERVER … FOREIGN DATA WRAPPER postgres_fdw` returned `CREATE SERVER` on
  both. On the substrate the objects are on disk at
  `/nix/store/…-postgresql-15.8-lib/lib/{dblink,postgres_fdw}.so`.
- **Nothing was left behind.** Every probe ran inside a transaction that was rolled
  back; the post-rollback re-query returned `NONE` for `pg_extension` and
  `NO_SERVERS` for `pg_foreign_server` on both servers. No database was created or
  dropped for this. (`hq_test_go_b2` is this card's own and stays.)

**Precise state to carry into the fork: not installed, but one `CREATE EXTENSION`
away at both ends.** This is why §5 option (a) was rewritten. It previously read as
though it faced two obstacles — a reversed decision and an absent capability. There
is one obstacle. The capability is a one-line DDL.

Code-side, on this branch:

- **The backend server has exactly one database.** `backend/cmd/server/main.go:264`
  reads one DSN, `DB_URL`, into one pool (`backend/internal/db/db.go:17` `NewPool`).
  A repo-wide grep for `pgxpool.New` outside tests finds only that pool plus two
  one-shot CLIs (`cmd/seed`, `cmd/sync-toast`), both on the same `DB_URL`. **There is
  no connection from HQ's backend to the sync Postgres and no env var that could
  create one.**
- The only channel HQ has to the substrate is the `/sync/*` reverse proxy
  (`main.go:439`, `backend/internal/sync/proxy.go`), which forwards HTTP to PostgREST
  with a `role: authenticated` token. `Sign()` **refuses** to mint `service_role`
  (`TestMint_NeverMintsServiceRole`), so the backend cannot even write the projection
  through that channel today, let alone write it transactionally.
- The mutation the projection would have to be transactional with is
  `backend/internal/workflow/repository.go` — `insertTemplateInTx` (`:112`) and
  `updateTemplateInTx`, which at `:236` does `DELETE FROM template_assignments WHERE
  template_id = $1` and re-inserts wholesale at `:249`. Both are `pgx.Tx` on HQ's
  pool. **A `tx.Exec` on that transaction cannot reach 46011.**

E5/E5b are the sharpest single measurement: `max_prepared_transactions = 0` on
**both** servers means two-phase commit is not merely unchosen, it is disabled. The
one mechanism that could technically make a cross-server write atomic is off at both
ends.

## 4. The port was done. Here it is, so the follow-up card does not redo it.

**Nothing below is applied.** It is recorded as the design output of the port so the
resuming card starts from a read predicate rather than an unread one.

### 4a. The Go predicate (`backend/internal/sync/ops.go:474`, verbatim)

```sql
SELECT DISTINCT u.id::text
 FROM users u
 WHERE u.roles && ARRAY['admin','superadmin']
    OR EXISTS (
         SELECT 1 FROM template_assignments ta
         WHERE ta.template_id = $1::uuid
           AND ( (ta.assignee_type = 'user' AND u.id::text = ta.assignee_id)
              OR (ta.assignee_type = 'role' AND ta.assignee_id = ANY(u.roles)) )
       )
```

Preceded by a resolution step per entity type: `template` → itself;
`submission` → `SELECT template_id FROM checklist_submissions WHERE id=$1`;
`field_response` → `SELECT s.template_id FROM checklist_fields f JOIN
checklist_sections s ON f.section_id = s.id WHERE f.id = $1`; anything else → `[]`.

### 4b. The same predicate, transposed to RLS

The Go asks *"given a template, which users?"*; a policy asks *"given a user, which
rows?"*. Same relation, read along the other axis — which is exactly why it is a
**port** and not a new predicate.

```sql
-- Projections. Neither has a writer; that absence IS the park.
create table public.hq_template_assignees (   -- projection of template_assignments ⋈ users
  template_id text not null,
  user_id     text not null,
  primary key (template_id, user_id)
);
create table public.hq_user_roles (           -- live roles, for the admin arm
  user_id text primary key,
  roles   text[] not null
);

create or replace function public.hq_can_see_template(tid text)
returns boolean language sql stable security definer
set search_path = public, pg_temp as $$
  select exists (select 1 from public.hq_user_roles r
                  where r.user_id = public.hq_jwt_claim('sub')
                    and r.roles && array['admin','superadmin'])
      or exists (select 1 from public.hq_template_assignees a
                  where a.template_id = tid
                    and a.user_id = public.hq_jwt_claim('sub'));
$$;
```

The two arms are the two arms of the Go `WHERE`, in the same order, disjoined the
same way. The admin arm is a separate projection rather than rows in
`hq_template_assignees` because `u.roles && ARRAY[...]` is a property of the *user*,
not of the (template, user) pair — materialising it into the join table would be an
admins × templates cross product and a second write trigger on template creation.

`hq_jwt_claim` is reused, not reinvented: the plural-GUC accessor from
`.night-crew/qa/spike-supabase/sql/hq-bridge-policies.sql`. `auth.uid()` stays wrong
for HQ (legacy singular GUC, `PGRST_DB_USE_LEGACY_GUCS=false`, silent NULL), and
`public.hq_uid_trap` would have re-proved it every run.

### 4c. The two inherited properties — preserved, and confirmed preserved

1. **`assignment_role` is never filtered.** `template_assignments.assignment_role` is
   not referenced anywhere in 4a or 4b. An `'approver'` sees exactly what an
   `'assignee'` sees. Unchanged.
2. **The admin arm is unconditional.** `roles && ARRAY['admin','superadmin']` is a
   free-standing disjunct gated on nothing — not on assignment, not on template.
   Every admin sees every template. Unchanged.

Tightening either remains a separate card.

### 4d. Two gaps the port surfaced, which the resuming card must decide

Recorded because they are findings, not oversights, and a reader should not
rediscover them at 3am.

- **`submission_responses` has no ported path.** `ResolveEntityAccess` resolves a
  `field_response` through `checklist_fields ⋈ checklist_sections` — **neither table
  exists in the sync substrate** (E3). And `submission_id` is deliberately nullable
  there (a draft has no submission — B1's comment calls this load-bearing), so it
  cannot be resolved through the submission either. A faithful port needs a **third**
  projection, `field_id → template_id`. This is the collection that must sync best
  (offline drafts) and it is the one with no resolvable predicate.
- **`submission_rejections` has no ported path at all.** `ResolveEntityAccess`'s
  `switch` has no case for it and falls through to `return []string{}` — the current
  WebSocket layer does not fan rejections out. Giving it a policy is therefore an
  *extension*, not a port. The minimal consistent one is "a rejection is visible iff
  its submission is", but nothing in shipped code says that yet, and the card
  forbids inventing permission semantics.

  **One refinement, added in the repair round.** That minimal rule is at least
  *structurally available*, which the original text did not say:
  `submission_rejections.submission_id` is `text not null`
  (`sync-schema/sql/0001_sync_tables.sql:250`), so unlike `submission_responses`
  above there is no null case to fall through — every rejection row has a submission
  to be scoped by. **This does not promote it from extension to port.** Structural
  availability is not authority: nothing in shipped code asserts the rule, so writing
  it is still inventing a permission semantic. It is recorded only so the resuming
  card knows the shape is there and does not re-derive it.

## 5. The fork for morning triage

**How does the row-visibility projection stay live across two Postgres servers?**
Every option changes something already decided, which is why it is not mine.

Read §2b first: whether decision 61 binds this card **by letter or by analogy** is a
prior question, and it changes how heavy the "reopens decision 61" costs below are.

**The order below is the order I would have triage consider them** — cheapest real
obstacle first, not alphabetical. (a) leads because after the C-1 correction its only
obstacle is a decision, and a decision is exactly what morning triage is for.

| | Option | What it costs / reopens |
|---|---|---|
| a | **`postgres_fdw`/`dblink` from the sync DB into HQ's DB** | The only genuinely zero-window answer — the policy reads HQ's live tables, so **no projection needs writing at all**, and the whole cross-server-transaction problem stops existing rather than being managed. **The real and only blocker is that it reverses decision 61**, which rejected it ("couples the two databases in a way the cutover card has not settled"). That rejection predates B1 settling the topology, so it is re-openable — but reversing it is an operator/architect call, and it is the *sole* thing standing here. ~~Also NOT INSTALLED at either end (E4)~~ — **struck (C-1): the extensions are not an obstacle.** Measured on both servers by executing the DDL: not installed, but **one `CREATE EXTENSION` away at both ends** (E4a–E4c). The genuine standing cost is not setup, it is runtime: this puts HQ's Postgres on the network path of **every RLS row check**, so HQ's availability and latency become the substrate's. |
| d | **Defer to `sync-hard-cutover` — co-locate the two databases** | If HQ's workflow tables end up in the substrate Postgres, decision 61's contract becomes trivially satisfiable, because "same transaction" becomes "same database". This is the option that makes the original contract come true rather than amending it. It also means B2 cannot land before the cutover, which reorders the roadmap. |
| b′ | **Native logical replication — publication on HQ for `template_assignments` + `users`, subscription on the substrate** | **Added in the repair round (C-2).** Materially nicer than (b) on every axis but one: crash-safe, built into Postgres, and **zero code in the mutation path** — `repository.go:236`/`:249` is not touched, not wrapped, not re-ordered. It also strictly dominates (b): same window, none of the outbox table, pusher process, or retry logic. **The axis it does not improve is the one the park is about: it is still asynchronous.** Its window is the replication lag — bounded in normal operation by apply latency (sub-second on a loopback/LAN), and degrading to *unbounded* in the pathological case where the subscription is down and the slot backs up. So it sits in **the same risk class as (b)**: between HQ's commit and the substrate's apply, a revoked assignment is still readable. **It refines the fork; it does not refute the park.** Setup is real but not a blocker: HQ is `wal_level=replica` and needs a bump to `logical` **plus a restart** (E7); the substrate is already `wal_level=logical` with 4 logical workers. Note it replicates `users` wholesale, which is a data-surface decision of its own. |
| b | **Transactional outbox in HQ's DB + pusher** | Genuinely push-on-change and crash-safe, but the window moves rather than closing: bounded lag between HQ commit and substrate apply. Is a *bounded* window acceptable where decision 61 refused an *unbounded* one? That is the operator's risk call, not a mechanism choice. **Now largely superseded by (b′)** — same window, strictly more moving parts and the only one of the two that puts code in the mutation path. Prefer it over (b′) only if you specifically want application-level control over *what* is projected rather than replicating whole tables. |
| c | **Two-phase commit** | Truly atomic. `max_prepared_transactions=0` on both servers (E5/E5b), so it is off by configuration; and it makes a stuck prepared transaction on the sync box able to wedge HQ's writes. |
| e | **Restructure the assignment write path** | The "different card" the park trigger itself names. |

**Recommendation, offered not taken:** (a) and (d) are the only two that honour
decision 61's *reason* rather than merely its words, and after C-1 the distance
between (a) and shipping is one operator decision plus one line of DDL — it is the
cheapest of the two to *evaluate*, whatever you conclude. (b′) is the best of the
asynchronous family and should displace (b) if the operator accepts a bounded window
at all. What has not changed is the reason the park exists: **an asynchronous option
is still an asynchronous option**, and the failure mode the park guards against is
someone reaching for one at 3am without having read decision 61 — (b′) being *nicer*
than (b) makes that reach easier, not safer.

## 6. What did NOT change, stated so the merger can trust it unattended

- `HQ_SYNC_REST_URL` is **not** set, referenced, or added anywhere by this branch.
  The activation interlock stands exactly as B1 and `sync-proxy-endpoint` left it.
- B1's four tables keep RLS enabled with **zero policies** — deny-all. This card
  wrote no policy, so it did not weaken that by a single row.
- `ops.go:474` is byte-identical to the base branch. It was read, not edited.
- **No migration was created. A1 takes `0072` uncontested.**
- The roadmap bullet stays `PLANNED — SLATE-READY`. Flipping a parked card to DONE
  would tell the morning the door is closable.
- ~~The shared spike Supabase stack was **read only**. No database was created,
  dropped, or written; B1's SQL was not applied to it.~~ **STRUCK — FALSE (C-4).**
  Corrected: **no database was created or dropped on the shared spike stack, and
  B1's `0001_sync_tables.sql` was not applied to it — but the stack WAS written to,
  twice over, and the note should have said so.**
  1. **By `TestJWTBridgeRLS`.** Its setup calls `applySQL()` on
     `.night-crew/qa/spike-supabase/sql/hq-bridge-fixture.sql`
     (`backend/internal/sync/jwtbridge_rls_test.go:162`) and on
     `hq-bridge-policies.sql` (`:169`) against 46011 **on every run**. The fixture
     creates three tables (`hq_grant_projection`, `hq_sync_checklists`,
     `hq_uid_trap`), inserts seed rows into all three, and sets `replica identity
     full`; the policies file enables RLS and creates 8 policy/RLS statements. Several
     variants then `insert`/`delete` in `hq_grant_projection` mid-test
     (`:512`, `:518`, `:590`, `:595`). That is DDL and DML, not reads.
  2. **By this repair round's own C-1 probe.** `CREATE EXTENSION dblink` /
     `postgres_fdw` and `CREATE SERVER probe_srv` were executed against 46011 — inside
     a transaction that was **rolled back**, with the post-rollback state re-queried
     and confirmed `NONE` / `NO_SERVERS`. Nothing persisted, but a write was attempted
     and must be declared.

  **Why this is substantively harmless, stated so triage does not have to re-derive
  it.** Everything written is the spike's own idempotent fixture — `create table if
  not exists`, `drop policy if exists`, fixed seed ids — living in three
  `hq_*`-prefixed tables that exist for exactly this purpose. **None of it touches
  B1's four tables.** Verified after the fact: `checklist_templates`,
  `checklist_submissions`, `submission_responses` and `submission_rejections` are
  **not present in the substrate at all** (queried post-run: `NONE_PRESENT`), so their
  deny-all state could not have been altered by a write that never reached them.
  The bullet above it — B1's four tables, RLS enabled, zero policies — stands
  unqualified.

## 7. Provenance, and two things worth a line each

### 7a. What was run, by whom, when — stated plainly because the record was ambiguous

G6 flagged that it **could not confirm** the original run actually executed
`TestJWTBridgeRLS`, as opposed to reporting a green banked by an earlier card:
`hq_grant_projection` held 2 rows beforehand, and the 2026-07-26 card B run explains
that just as well as a fresh run does. The ambiguity is real and is not resolvable
after the fact. So rather than defend the earlier claim, the record is replaced with
a first-hand one:

| Who | What | When | Result |
|---|---|---|---|
| G6 | `TestJWTBridgeRLS` | earlier today | **16/16 PASS**, incl. both `service_role` BYPASSRLS controls |
| **This repair round** | `go test ./internal/sync/ -run TestJWTBridgeRLS -v -count=1` | **2026-07-29 02:02:20–02:02:23 UTC** | **PASS — 16/16 subtests**, `ok … 0.799s`. Both `service_role` BYPASSRLS controls green (`CONTROL/…BYPASSRLS_proves_the_rows_are_there` and `CONTROL/…re-taken_after_all_variants`, both seeing all 4 rows), plus V1–V13 and the positive. |
| **This repair round** | Every §3 probe, both servers, incl. the four `CREATE EXTENSION` executions | **2026-07-29, this round** | As tabulated in §3. Re-measured; not copied from G6. |

**The original note's green is not being relied on.** Every claim in §3 and every
claim in this section was measured in the repair round on this host, and G6's
independent numbers agree with them.

### 7b. Backlog candidate — a harness sharp edge, not a card defect

G6 reports that a parallel `go test ./...` produces `recipes` failures (a deadlock
plus a fixture-precondition failure) that **vanish under `-p 1`**. That is
cross-package contention on a single shared test database — Go's default
`-p = NumCPU` runs packages concurrently against one DSN. It is **pre-existing and
unrelated to this card** (this branch adds no Go code and no test). Recorded as a
backlog candidate, not as a B2 finding: either give packages separate schemas/DBs or
pin the suite to `-p 1`. Filed here only so it is not rediscovered as a mystery flake.

### 7c. Corroboration for the no-red decision (G6 found this; the note had not cited it)

The merge-intent note argues no red-first suite was shipped because it would have
been a proof about a table nothing in production writes. That argument now has direct
evidence, which the original note asserted but did not demonstrate. Repo-wide,
`hq_grant_projection` is **written** from exactly two places:

- `backend/internal/sync/jwtbridge_rls_test.go` (`:512`, `:518`, `:590`, `:595`) — a test
- `.night-crew/qa/spike-supabase/sql/hq-bridge-fixture.sql` (`:153`) — a fixture

Every other hit in the repo is a `select`, a policy, a proxy route-mapping table, or
prose. **There is no push-on-change writer anywhere in the repo to imitate.** The
projection pattern decision 61 assumes exists in production does not exist in
production; it exists only in the spike. That strengthens the park rather than
weakening it — B2 was not asked to copy a shipped mechanism, it was asked to invent
the first one, across a server boundary, in a transaction that cannot span it.
