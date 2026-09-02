-- sync-schema/sql/0001_sync_tables.sql
--
-- The self-hosted per-table contract for HQ's four replicated sync tables.
-- Card `sync-rxdb-collections-and-table-contract` (overnight-20260729-2, B1).
--
-- ===========================================================================
-- ⚠  THIS FILE DOES NOT RUN AGAINST HQ's POSTGRES.
-- ===========================================================================
-- It targets the SELF-HOSTED SUPABASE Postgres (docker-compose.supabase.yml's
-- `db` service) — the sync substrate, not HQ's own database. It is deliberately
-- NOT a goose migration under backend/internal/db/migrations/, because anything
-- placed there runs against HQ's database on every backend start, and these
-- tables have no business being there: `authenticated`/`anon` do not exist as
-- roles in HQ's Postgres, and `supabase_realtime` is not a publication it has.
--
-- Idempotent. Safe to re-run against a live stack.
--
-- ===========================================================================
-- WHAT THIS FILE DELIBERATELY DOES NOT CONTAIN
-- ===========================================================================
--
-- 1. NO POLICIES. Every table below has row level security ENABLED and ZERO
--    policies, which in Postgres is DENY-ALL for every non-owner role. That is
--    the correct and intended state: the RLS predicates are the whole of card
--    `sync-rxdb-row-visibility-rls` (B2), which PORTS `ResolveEntityAccess`
--    (backend/internal/sync/ops.go:474) and projects `template_assignments ⋈
--    users` the way `hq_grant_projection` projects grants. A permissive policy
--    landing in THIS file would silently open the door that card exists to
--    guard. tests/sync-schema.spec.js asserts there is no CREATE POLICY here.
--
--    Corollary: `HQ_SYNC_REST_URL` must not be set on a deploy whose substrate
--    carries no RLS policies. `sync-proxy-endpoint` forwards every method to
--    PostgREST with a `role: authenticated` token and no row filtering of its
--    own, deliberately, because filtering was always meant to be the substrate's
--    RLS. On such a deploy a set `HQ_SYNC_REST_URL` gives every logged-in crew
--    member full read AND write on the whole exposed schema. The card that ports
--    those policies (`sync-rxdb-row-visibility-rls`, B2) MERGED 2026-08-01, so
--    the milestone is not "until B2 lands" — it is the cutover: no page calls
--    startHQReplication, and the substrate must carry the policies before the
--    door opens. With no policies present, the tables below are deny-all rather
--    than open — but "deny-all" is a property of THIS file that the substrate's
--    RLS replaces, not a standing guarantee.
--
-- 2. NO TABLE FOR THE OVERWRITTEN-ANSWER RECORD. Ledger T-27 decision 89: it is
--    a personal, per-device undo held in a LOCAL RxDB collection
--    (`conflict_records` in sync-schema/collections.js). No server table, no
--    endpoint, no replication. Its absence from this file IS the decision.
--
-- 3. NO `lamport_ts`. That column belongs to the op-log/Lamport-clock layer this
--    cycle replaces. Under RxDB the pull cursor is `_modified` and ordering is
--    the replication protocol's business.
--
-- ===========================================================================
-- THE SIX-ITEM CONTRACT, measured on public.spike_notes by the feasibility
-- spike (.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md,
-- "The per-table contract"). Every table below carries all six.
-- ===========================================================================
--
--   1. `id text primary key`  — RxDB documents carry client-generated string
--      ids. A device offline on the truck invents the id before any server sees
--      the row, so a bigserial/uuid-default PK cannot round-trip it.
--   2. `_deleted boolean not null default false` — RxDB replication is
--      soft-delete only. A hard DELETE is invisible to a pull handler: the row
--      simply stops appearing and every offline replica keeps it forever.
--   3. `_modified timestamptz not null default now()` — the pull checkpoint.
--   4. A BEFORE INSERT OR UPDATE trigger stamping `_modified` server-side. NOT
--      optional: if `_modified` were ever client-set, a skewed clock writes a
--      checkpoint in the past (every replica silently re-pulls) or in the future
--      (every replica silently MISSES rows). Server stamping is what makes the
--      cursor trustworthy. The function is shared; the trigger is per table.
--   5. `ENABLE ROW LEVEL SECURITY` + `REVOKE ... FROM anon` +
--      `GRANT ... TO authenticated`. Grants and RLS are two independent gates
--      and BOTH are needed: RLS without the revoke leaves an unauthenticated
--      hole; the grant without RLS lets every user read every user's rows.
--      (The policy set — item 5's fourth part — is B2's, see above.)
--   6. `ALTER PUBLICATION supabase_realtime ADD TABLE` + `REPLICA IDENTITY
--      FULL`. Per table, manual, with no UI in self-hosted. Forgetting it is not
--      silent but IS easy to miss: the `phx_join` still replies
--      `{"status":"ok"}` with a subscription id, and the failure arrives
--      afterwards as a separate `system` frame. A client that resolves its
--      "subscribed" state on the join reply believes it is subscribed to a table
--      that will never fire. REPLICA IDENTITY FULL makes the pre-image available
--      so Realtime can evaluate RLS against the OLD row.
--
-- Note on what is NOT declared on the RxDB side: neither `_deleted` nor
-- `_modified` appears in any collection schema. `_deleted` is RxDB's own field
-- (the plugin maps this column onto it); `_modified` is left out by ledger
-- decision 78 so it stays a pure pull cursor. Both columns still exist HERE —
-- the contract is a property of the table, not of the document.

begin;

-- ---------------------------------------------------------------------------
-- 0. The shared stamping function (item 4). One function, four triggers.
-- ---------------------------------------------------------------------------
create or replace function public.hq_sync_set_modified()
returns trigger language plpgsql as $$
begin
  new._modified := now();
  return new;
end;
$$;

-- ---------------------------------------------------------------------------
-- 1. templates  ←  HQ's checklist_templates (migration 0006)
--
--    `updated_by` is new: ledger decision 79. Replicated rows carry who-and-when
--    so the conflict sheet's "Dana M., 6:12 PM" is real rather than "someone
--    else". Nullable — a server-side touch has no human actor, and a null is
--    what lets the UI say "someone else" honestly instead of inventing a name.
-- ---------------------------------------------------------------------------
create table if not exists public.checklist_templates (
  id                text        primary key,
  name              text        not null,
  requires_approval boolean     not null default false,
  created_by        text,
  created_at        timestamptz not null default now(),
  updated_by        text,
  updated_at        timestamptz not null default now(),
  archived_at       timestamptz,
  _deleted          boolean     not null default false,
  _modified         timestamptz not null default now()
);

drop trigger if exists checklist_templates_set_modified on public.checklist_templates;
create trigger checklist_templates_set_modified
  before insert or update on public.checklist_templates
  for each row execute function public.hq_sync_set_modified();

-- The pull cursor's index. Every pull is "everything since X ordered by X".
create index if not exists checklist_templates_modified_idx
  on public.checklist_templates (_modified);

alter table public.checklist_templates enable row level security;
revoke all on public.checklist_templates from anon;
grant select, insert, update on public.checklist_templates to authenticated;

alter table public.checklist_templates replica identity full;

-- ---------------------------------------------------------------------------
-- 2. checklists  ←  HQ's checklist_submissions (migration 0011)
--
--    `template_snapshot` stays jsonb: a submission carries a frozen copy of the
--    structure it was filled against, which is what makes a filled checklist
--    self-contained offline even after the template is edited.
--
--    `updated_by`/`updated_at` are both new (decision 79). `submitted_by` /
--    `submitted_at` answer "who submitted", which is NOT "who last changed this
--    row" — an approver's review changes the row without touching either.
--
--    NOTE for B2 and for whoever writes the backfill: there is no FK to
--    `checklist_templates` here. HQ has one, but the sync substrate deliberately
--    does not: a replica can legitimately receive a submission before the
--    template row it references (pull order is by `_modified`, not by
--    dependency), and an FK would turn ordinary replication lag into a rejected
--    push. Referential integrity stays HQ's Postgres's job.
-- ---------------------------------------------------------------------------
create table if not exists public.checklist_submissions (
  id                text        primary key,
  template_id       text        not null,
  template_snapshot jsonb       not null,
  submitted_by      text        not null,
  submitted_at      timestamptz not null default now(),
  status            text        not null default 'pending'
                                check (status in ('pending','approved','rejected','completed')),
  reviewed_by       text,
  reviewed_at       timestamptz,
  idempotency_key   text,
  updated_by        text,
  updated_at        timestamptz not null default now(),
  _deleted          boolean     not null default false,
  _modified         timestamptz not null default now()
);

drop trigger if exists checklist_submissions_set_modified on public.checklist_submissions;
create trigger checklist_submissions_set_modified
  before insert or update on public.checklist_submissions
  for each row execute function public.hq_sync_set_modified();

create index if not exists checklist_submissions_modified_idx
  on public.checklist_submissions (_modified);
create index if not exists checklist_submissions_template_idx
  on public.checklist_submissions (template_id);

alter table public.checklist_submissions enable row level security;
revoke all on public.checklist_submissions from anon;
grant select, insert, update on public.checklist_submissions to authenticated;

alter table public.checklist_submissions replica identity full;

-- ---------------------------------------------------------------------------
-- 3. responses  ←  HQ's submission_responses (migration 0012)
--
--    `submission_id` is NULLABLE and that is load-bearing: a DRAFT response has
--    no submission yet (0012's partial unique index on
--    `(field_id, answered_by) WHERE submission_id IS NULL`). Drafts are exactly
--    what a crew member fills offline, so the collection that must sync best is
--    the one whose foreign key is absent.
--
--    `answered_by`/`answered_at` already satisfied decision 79 in HQ — zero new
--    columns here.
--
--    The two uniqueness rules are ported verbatim rather than re-derived. They
--    are the reason a double-press does not create a second answer.
-- ---------------------------------------------------------------------------
create table if not exists public.submission_responses (
  id            text        primary key,
  submission_id text,
  field_id      text        not null,
  value         jsonb       not null,
  answered_by   text        not null,
  answered_at   timestamptz not null default now(),
  _deleted      boolean     not null default false,
  _modified     timestamptz not null default now()
);

create unique index if not exists submission_responses_submission_field_uniq
  on public.submission_responses (submission_id, field_id);
create unique index if not exists submission_responses_draft_idx
  on public.submission_responses (field_id, answered_by)
  where submission_id is null;

drop trigger if exists submission_responses_set_modified on public.submission_responses;
create trigger submission_responses_set_modified
  before insert or update on public.submission_responses
  for each row execute function public.hq_sync_set_modified();

create index if not exists submission_responses_modified_idx
  on public.submission_responses (_modified);

alter table public.submission_responses enable row level security;
revoke all on public.submission_responses from anon;
grant select, insert, update on public.submission_responses to authenticated;

alter table public.submission_responses replica identity full;

-- ---------------------------------------------------------------------------
-- 4. approvals  ←  HQ's submission_rejections (migration 0014)
--
--    An approval is recorded in HQ as the ABSENCE of rejection rows plus
--    `checklist_submissions.status`; a rejection is a per-field row carrying the
--    reviewer's comment. Per-field is also the granularity the conflict sheet
--    works at, so the mirror is faithful in both directions.
--
--    `field_id` carries NO foreign key, here as in HQ: migration 0051 dropped it
--    because `replaceTemplate` deletes and re-creates fields with new UUIDs, and
--    rejections reference the ids frozen in the submission's snapshot.
--
--    `rejected_by`/`rejected_at` already satisfied decision 79 — zero new
--    columns here.
-- ---------------------------------------------------------------------------
create table if not exists public.submission_rejections (
  id            text        primary key,
  submission_id text        not null,
  field_id      text        not null,
  comment       text        not null,
  require_photo boolean     not null default false,
  rejected_by   text        not null,
  rejected_at   timestamptz not null default now(),
  _deleted      boolean     not null default false,
  _modified     timestamptz not null default now()
);

drop trigger if exists submission_rejections_set_modified on public.submission_rejections;
create trigger submission_rejections_set_modified
  before insert or update on public.submission_rejections
  for each row execute function public.hq_sync_set_modified();

create index if not exists submission_rejections_modified_idx
  on public.submission_rejections (_modified);
create index if not exists submission_rejections_submission_idx
  on public.submission_rejections (submission_id);

alter table public.submission_rejections enable row level security;
revoke all on public.submission_rejections from anon;
grant select, insert, update on public.submission_rejections to authenticated;

alter table public.submission_rejections replica identity full;

commit;

-- ---------------------------------------------------------------------------
-- 5. Realtime enrolment (item 6, first half).
--
--    Outside the transaction above on purpose: `alter publication` is the one
--    statement here that is expected to raise on re-run, and the exception
--    handler below swallows only that. The publication itself always exists in a
--    fresh stack — supabase/postgres's own 00000000000001-initial-schema.sql
--    runs `create publication supabase_realtime;` — so this is only ever
--    ADD TABLE.
-- ---------------------------------------------------------------------------
do $$
begin
  alter publication supabase_realtime add table public.checklist_templates;
exception when duplicate_object then null;
end;
$$;

do $$
begin
  alter publication supabase_realtime add table public.checklist_submissions;
exception when duplicate_object then null;
end;
$$;

do $$
begin
  alter publication supabase_realtime add table public.submission_responses;
exception when duplicate_object then null;
end;
$$;

do $$
begin
  alter publication supabase_realtime add table public.submission_rejections;
exception when duplicate_object then null;
end;
$$;

-- PostgREST caches the schema and will 404 a brand-new table until it reloads.
-- This NOTIFY is the reload signal — cheaper than restarting the service, and
-- forgetting it looks exactly like "the table was never created".
notify pgrst, 'reload schema';
