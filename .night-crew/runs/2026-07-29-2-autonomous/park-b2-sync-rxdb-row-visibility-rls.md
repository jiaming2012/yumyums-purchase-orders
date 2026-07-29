# 🛑 PARK — Card B2 `sync-rxdb-row-visibility-rls`

Run `overnight-20260729-2`, branch `card/b2-sync-rxdb-row-visibility-rls`
(worktree preserved at `/home/jcole/projects/hq-worktrees/b2-sync-rxdb-row-visibility-rls`).

**The card's own PARK trigger fired. No production code, SQL, policy or migration
was written. The `HQ_SYNC_REST_URL` interlock is unchanged and still in force.**

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

## 3. Evidence — measured on this host, not assumed

HQ Postgres `127.0.0.1:5433` (`yumyums-dev-pg`); sync substrate `127.0.0.1:46011`
(`spike-supabase-db-1`, up 3 days, healthy). Verbatim results:

| # | Question | Where | Result |
|---|---|---|---|
| E1 | Does HQ's Postgres have `anon`/`authenticated`/`service_role`/`authenticator`/`supabase_admin`? | 5433 | **0 of 5 present** |
| E2 | Does the sync Postgres? | 46011 | **all 5 present** |
| E3 | Does the sync Postgres hold `template_assignments`, `users`, `checklist_fields`, `checklist_sections`? | 46011 | **0 of 4 present** |
| E4 | Are `postgres_fdw` / `dblink` installed? | 46011 | **both NOT INSTALLED** (available, uninstalled) |
| E5 | `max_prepared_transactions` (2PC capability) | 46011 | **0** |
| E5b | `max_prepared_transactions` | 5433 | **0** |

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

## 5. The fork for morning triage

**How does the row-visibility projection stay live across two Postgres servers?**
Every option changes something already decided, which is why it is not mine.

| | Option | What it costs / reopens |
|---|---|---|
| a | **`postgres_fdw`/`dblink` from the sync DB into HQ's DB** | The only genuinely zero-window answer — the policy reads HQ's live tables, so no projection needs writing at all. **Explicitly rejected by decision 61** ("couples the two databases in a way the cutover card has not settled"). That rejection predates B1 settling the topology, so it is re-openable — but reversing it is an operator/architect call. Also NOT INSTALLED at either end (E4), and it puts HQ's Postgres on the network path of every RLS row check. |
| b | **Transactional outbox in HQ's DB + pusher** | Genuinely push-on-change and crash-safe, but the window moves rather than closing: bounded lag between HQ commit and substrate apply. Is a *bounded* window acceptable where decision 61 refused an *unbounded* one? That is the operator's risk call, not a mechanism choice. |
| c | **Two-phase commit** | Truly atomic. `max_prepared_transactions=0` on both servers (E5/E5b), so it is off by configuration; and it makes a stuck prepared transaction on the sync box able to wedge HQ's writes. |
| d | **Defer to `sync-hard-cutover` — co-locate the two databases** | If HQ's workflow tables end up in the substrate Postgres, decision 61's contract becomes trivially satisfiable, because "same transaction" becomes "same database". This is the option that makes the original contract come true rather than amending it. It also means B2 cannot land before the cutover, which reorders the roadmap. |
| e | **Restructure the assignment write path** | The "different card" the park trigger itself names. |

**Recommendation, offered not taken:** (d) and (a) are the only two that honour
decision 61's *reason* rather than merely its words. (b) is the pragmatic one and is
the one most likely to get improvised at 3am by someone who has not read decision 61
— which is precisely why the park exists.

## 6. What did NOT change, stated so the merger can trust it unattended

- `HQ_SYNC_REST_URL` is **not** set, referenced, or added anywhere by this branch.
  The activation interlock stands exactly as B1 and `sync-proxy-endpoint` left it.
- B1's four tables keep RLS enabled with **zero policies** — deny-all. This card
  wrote no policy, so it did not weaken that by a single row.
- `ops.go:474` is byte-identical to the base branch. It was read, not edited.
- **No migration was created. A1 takes `0072` uncontested.**
- The roadmap bullet stays `PLANNED — SLATE-READY`. Flipping a parked card to DONE
  would tell the morning the door is closable.
- The shared spike Supabase stack was **read only**. No database was created,
  dropped, or written; B1's SQL was not applied to it.
