-- spike-fixture.sql — ONE table that carries the full self-hosted contract an
-- RxDB-replicated table has to satisfy, plus the smallest possible RLS policy
-- pair that can be shown to DISCRIMINATE.
--
-- Idempotent: safe to re-run against a live stack.
--
-- ⚠ THE POLICIES BELOW ARE A PROOF DEVICE, NOT A PRODUCT DESIGN.
--   `owner_id = sub` is the simplest predicate that can be observed either
--   admitting or refusing a request, which is all this spike needs. Which HQ
--   claims map to which grants — the actual policy semantics — belongs to the
--   card `sync-jwt-bridge-endpoint` and to the operator. Do not read this file
--   as a proposal.

-- ---------------------------------------------------------------------------
-- 1. text PK, _deleted, _modified — the RxDB replication contract
-- ---------------------------------------------------------------------------
-- text PK:   RxDB documents carry client-generated string ids. A bigserial PK
--            cannot round-trip an id the client invented while offline.
-- _deleted:  RxDB replication is soft-delete only. A hard DELETE is invisible
--            to a pull handler — the row simply stops appearing, and every
--            offline replica keeps it forever.
-- _modified: the pull checkpoint. Without a monotonically-updated column the
--            client has no "everything since X" cursor and must full-sync.
create table if not exists public.spike_notes (
  id         text        primary key,
  owner_id   text        not null,
  body       text        not null,
  _deleted   boolean     not null default false,
  _modified  timestamptz not null default now()
);

-- The trigger is NOT optional. If _modified is only ever set by the client,
-- a client with a skewed clock writes a checkpoint in the past and every other
-- replica silently re-pulls, or writes one in the future and other replicas
-- silently MISS rows. Server-side stamping is what makes the cursor trustworthy.
create or replace function public.spike_notes_set_modified()
returns trigger language plpgsql as $$
begin
  new._modified := now();
  return new;
end;
$$;

drop trigger if exists spike_notes_set_modified on public.spike_notes;
create trigger spike_notes_set_modified
  before insert or update on public.spike_notes
  for each row execute function public.spike_notes_set_modified();

-- ---------------------------------------------------------------------------
-- 2. RLS
-- ---------------------------------------------------------------------------
-- Grants and RLS are two independent gates and BOTH are needed. Grants decide
-- whether the role may touch the table at all; RLS decides which rows. Enabling
-- RLS without revoking from `anon` leaves an unauthenticated hole; granting to
-- `authenticated` without RLS lets every user read every user's rows.
alter table public.spike_notes enable row level security;

revoke all on public.spike_notes from anon;
grant select, insert, update on public.spike_notes to authenticated;

drop policy if exists spike_notes_select_own on public.spike_notes;
create policy spike_notes_select_own on public.spike_notes
  for select to authenticated
  using (owner_id = current_setting('request.jwt.claims', true)::json ->> 'sub');

drop policy if exists spike_notes_insert_own on public.spike_notes;
create policy spike_notes_insert_own on public.spike_notes
  for insert to authenticated
  with check (owner_id = current_setting('request.jwt.claims', true)::json ->> 'sub');

drop policy if exists spike_notes_update_own on public.spike_notes;
create policy spike_notes_update_own on public.spike_notes
  for update to authenticated
  using (owner_id = current_setting('request.jwt.claims', true)::json ->> 'sub')
  with check (owner_id = current_setting('request.jwt.claims', true)::json ->> 'sub');

-- ---------------------------------------------------------------------------
-- 3. Realtime enrolment — the step with NO dashboard toggle in self-hosted
-- ---------------------------------------------------------------------------
-- On hosted Supabase this is a checkbox in the Database > Replication UI. In
-- self-hosted there is no UI (we deliberately did not deploy Studio, and even
-- with Studio this toggle is a hosted-platform affordance). It is a per-table
-- ALTER that must be run for EVERY table you want replicated.
--
-- MEASURED (see README.md, proof R3 — this corrects an earlier guess of ours):
-- forgetting it is NOT silent, but it IS easy to miss. The `phx_join` still
-- replies `{"status":"ok"}` with a postgres_changes id, so a client that only
-- checks the join reply believes it is subscribed. The real signal arrives
-- afterwards as a separate `system` frame:
--   {"status":"error","extension":"postgres_changes",
--    "message":"Unable to subscribe to changes with given parameters..."}
-- Any client wrapper HQ writes must treat that system/error frame as a
-- subscription failure, because the join reply alone will not tell it.
do $$
begin
  alter publication supabase_realtime add table public.spike_notes;
exception
  when duplicate_object then null;  -- already enrolled; re-run is a no-op
end;
$$;

-- REPLICA IDENTITY FULL makes the pre-image available on UPDATE/DELETE. Realtime
-- needs it to evaluate RLS against the OLD row, and RxDB conflict handling wants
-- the previous revision. The default (`d` = primary key only) yields change
-- events whose `old_record` is just the id.
alter table public.spike_notes replica identity full;

-- ---------------------------------------------------------------------------
-- 4. Seed — two owners, so a policy that lets everything through looks
--    different from one that discriminates
-- ---------------------------------------------------------------------------
insert into public.spike_notes (id, owner_id, body) values
  ('note-alice-1', 'user-alice', 'alice seed row'),
  ('note-bob-1',   'user-bob',   'bob seed row')
on conflict (id) do nothing;

-- PostgREST caches the schema. After DDL it will 404 a brand-new table until
-- it reloads; this NOTIFY is the reload signal (cheaper than restarting it).
notify pgrst, 'reload schema';
