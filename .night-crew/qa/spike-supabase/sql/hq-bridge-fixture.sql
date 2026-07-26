-- hq-bridge-fixture.sql — the SUBSTRATE for card `sync-jwt-bridge-endpoint`.
--
-- 🛑 THIS FILE DELIBERATELY CONTAINS NO RLS POLICIES, AND THAT IS THE POINT.
--
-- The card's real gate is red-first: every attack variant must be captured
-- REFUSING before the policy that refuses it is written. So the fixture splits
-- in two:
--
--   hq-bridge-fixture.sql   (this file)  tables, grants, projection, seed.
--                                        RLS is NOT enabled. Every attacker
--                                        gets everything. This is the RED state.
--   hq-bridge-policies.sql               enables RLS and writes the policies.
--                                        This is the GREEN state.
--
-- Applying only this file and running jwtbridge_rls_test.go reproduces the red
-- capture. Note that RLS-is-off is the correct red: enabling RLS with NO
-- policies would deny everything and every variant would pass VACUOUSLY —
-- indistinguishable from a policy that works.
--
-- Idempotent: safe to re-run against a live stack.
--
-- ⚠ Runs against the LOCAL throwaway spike stack only
--   (`docker compose -p spike-supabase -f docker-compose.supabase.yml`).
--   Never a hosted Supabase project, never production, never real HQ data.
--
-- ⚠ This file does NOT touch sql/spike-fixture.sql or `public.spike_notes`.
--   W1's proof artifacts must keep re-verifying, and another card shares this
--   stack concurrently.

-- ---------------------------------------------------------------------------
-- 1. hq_grant_projection — the LIVE grant table the policies join against
-- ---------------------------------------------------------------------------
-- 🛑 This is the answer to "token replay after grant revocation", and it is the
-- reason this card does not simply read the `hq_grants` claim.
--
-- A JWT's claims are frozen at mint. A token minted at 09:00 still asserts the
-- grants held at 09:00 after an admin revokes them at 09:05. If RLS trusted the
-- claim, revocation would not bite until the token expired.
--
-- So the claim is advisory and THIS TABLE is the gate. It is a projection of
-- HQ's existing `app_permissions` ⋈ `hq_apps` — resolved to (user_id, slug)
-- pairs, exactly the predicate auth.RequirePermission already evaluates. It
-- introduces NO new permission concept; it changes only WHERE the existing
-- answer is read from, which is what makes revocation immediate.
--
-- (Who writes this table — a push on grant change, a periodic reconcile, or
-- postgres_fdw — is a mechanism question for the cutover card. The card that
-- owns it inherits an explicit contract, not a blank page.)
create table if not exists public.hq_grant_projection (
  user_id   text not null,
  app_slug  text not null,
  primary key (user_id, app_slug)
);

-- ---------------------------------------------------------------------------
-- 2. hq_sync_checklists — one replicated table carrying BOTH authz axes
-- ---------------------------------------------------------------------------
-- Two axes, deliberately, because HQ has two and they are independent (see the
-- AXIS test in tests/grant-enforcement-parity.spec.js: a grant does not
-- manufacture a role tier, and a role tier does not manufacture a grant):
--
--   owner_id  — WHOSE row it is.        Compared against the `sub` claim.
--   app_slug  — WHICH app it belongs to. Compared against the live projection.
--
-- A single-axis fixture cannot tell a policy that checks identity from one that
-- checks entitlement. Both must be shown discriminating separately.
--
-- text PK / _deleted / _modified are W1's per-table replication contract,
-- carried verbatim so this table is a realistic RxDB target and not a toy.
create table if not exists public.hq_sync_checklists (
  id         text        primary key,
  owner_id   text        not null,
  app_slug   text        not null,
  body       text        not null,
  _deleted   boolean     not null default false,
  _modified  timestamptz not null default now()
);

create or replace function public.hq_sync_checklists_set_modified()
returns trigger language plpgsql as $$
begin
  new._modified := now();
  return new;
end;
$$;

drop trigger if exists hq_sync_checklists_set_modified on public.hq_sync_checklists;
create trigger hq_sync_checklists_set_modified
  before insert or update on public.hq_sync_checklists
  for each row execute function public.hq_sync_checklists_set_modified();

-- ---------------------------------------------------------------------------
-- 3. hq_uid_trap — the auth.uid() NEGATIVE CONTROL
-- ---------------------------------------------------------------------------
-- 🛑 Not a product table. It exists to make finding #1 REPRODUCIBLE rather than
-- merely believed.
--
-- Without GoTrue's migrations the `auth` schema ships only email/role/uid, and
-- auth.uid() is:
--     select nullif(current_setting('request.jwt.claim.sub', true), '')::uuid;
-- i.e. the LEGACY SINGULAR GUC, which PostgREST populates only when
-- PGRST_DB_USE_LEGACY_GUCS=true. This stack sets it "false".
--
-- Every copy-pasted policy from Supabase's hosted docs uses auth.uid(). This
-- table carries the same rows as hq_sync_checklists but is governed by an
-- auth.uid() policy, so the suite can show, side by side and in the same run:
--   the plural-GUC policy DISCRIMINATES  ·  the auth.uid() policy returns NOTHING
-- for the very same token. Delete this table and the finding degrades back into
-- a comment nobody re-verifies.
create table if not exists public.hq_uid_trap (
  id         text primary key,
  owner_id   text not null,
  body       text not null
);

-- ---------------------------------------------------------------------------
-- 4. Grants — necessary but NOT sufficient
-- ---------------------------------------------------------------------------
-- Grants and RLS are two independent gates and both are needed. Grants decide
-- whether a role may touch the table at all; RLS decides which rows. Revoking
-- from `anon` is what makes the anon variant a real refusal rather than an
-- empty-result coincidence.
revoke all on public.hq_sync_checklists   from anon;
revoke all on public.hq_grant_projection  from anon;
revoke all on public.hq_uid_trap          from anon;

grant select, insert, update on public.hq_sync_checklists to authenticated;
grant select                 on public.hq_grant_projection to authenticated;
grant select                 on public.hq_uid_trap          to authenticated;

-- ---------------------------------------------------------------------------
-- 5. Seed — two owners × two apps, so "lets everything through" and
--    "discriminates" cannot look the same
-- ---------------------------------------------------------------------------
-- A single-owner seed makes a broken policy indistinguishable from a working
-- one. Alice and Bob each own rows in `operations`, and Alice additionally owns
-- one in `inventory` that she holds no live grant on — that last row is what
-- separates the identity axis from the entitlement axis.
insert into public.hq_sync_checklists (id, owner_id, app_slug, body) values
  ('chk-alice-ops-1', 'hq-user-alice', 'operations', 'alice opening checklist'),
  ('chk-alice-ops-2', 'hq-user-alice', 'operations', 'alice closing checklist'),
  ('chk-bob-ops-1',   'hq-user-bob',   'operations', 'bob opening checklist'),
  ('chk-alice-inv-1', 'hq-user-alice', 'inventory',  'alice stock count')
on conflict (id) do nothing;

insert into public.hq_uid_trap (id, owner_id, body) values
  ('trap-alice-1', 'hq-user-alice', 'alice trap row'),
  ('trap-bob-1',   'hq-user-bob',   'bob trap row')
on conflict (id) do nothing;

-- Alice holds `operations` live. She does NOT hold `inventory`, even though she
-- OWNS a row in it. Bob holds `operations` live.
insert into public.hq_grant_projection (user_id, app_slug) values
  ('hq-user-alice', 'operations'),
  ('hq-user-bob',   'operations')
on conflict (user_id, app_slug) do nothing;

-- ---------------------------------------------------------------------------
-- 6. Realtime enrolment — the step with no dashboard toggle in self-hosted
-- ---------------------------------------------------------------------------
-- Per-table ALTER, required for every replicated table. Forgetting it is not
-- silent but IS easy to miss: the phx_join still replies ok, and the real
-- signal arrives afterwards as a separate `system` frame (W1 proof R3).
do $$
begin
  alter publication supabase_realtime add table public.hq_sync_checklists;
exception
  when duplicate_object then null;
end;
$$;

alter table public.hq_sync_checklists replica identity full;

-- PostgREST caches the schema; it will 404 a brand-new table until it reloads.
notify pgrst, 'reload schema';
