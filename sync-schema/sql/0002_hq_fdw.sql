-- sync-schema/sql/0002_hq_fdw.sql
--
-- The SUBSTRATE-side foreign tables that read HQ's live permission data.
-- Card `sync-rxdb-row-visibility-rls` (overnight-20260801, B2).
--
-- ===========================================================================
-- ⚠  THIS FILE DOES NOT RUN AGAINST HQ's POSTGRES.
-- ===========================================================================
-- Same warning 0001 carries, for the same reason. This targets the SELF-HOSTED
-- SUPABASE Postgres (docker-compose.supabase.yml's `db` service). Its HQ-side
-- counterpart is the goose migration backend/internal/db/migrations/
-- 0073_sync_fdw_views.sql, which creates the three views this file imports and
-- the least-privilege role it logs in as.
--
-- Idempotent. Safe to re-run against a live stack.
--
-- ===========================================================================
-- 🛑 THIS FILE CONTAINS NO POLICIES, AND THAT IS THE POINT.
-- ===========================================================================
--
-- The card's gate is RED-FIRST, so the two halves are split exactly the way
-- .night-crew/qa/spike-supabase/sql/{hq-bridge-fixture,hq-bridge-policies}.sql
-- are split, and for exactly the same reason:
--
--   0002_hq_fdw.sql       (this file)  extension, server, mapping, foreign
--                                      tables. NO POLICIES. Applied together
--                                      with a teardown that DISABLES RLS on
--                                      the four replicated tables, this is the
--                                      RED state: every attacker sees
--                                      everything.
--   0003_rls_policies.sql              the predicates and the policies.
--                                      This is the GREEN state.
--
-- 🛑 NOTE WHY THE RED IS "RLS OFF" AND NOT "RLS ON, NO POLICIES". 0001 leaves
-- the four tables RLS-enabled with zero policies, which in Postgres is
-- DENY-ALL. Running the attack suite against THAT would pass every variant
-- VACUOUSLY — a table nobody can read is indistinguishable from a policy that
-- works, which is precisely the failure mode this run is armed against
-- (guard integrity, B-22/B-23/B-24). So the red state actively tears RLS down
-- and the variants are captured LEAKING. Reproduce it at any time:
--
--     SYNC_RLS_SKIP_POLICIES=1 go test ./internal/sync/ -run TestRowVisibilityRLS -v
--
-- ===========================================================================
-- WHY FOREIGN TABLES AT ALL — ledger decision 92, which REVERSES decision 61
-- ===========================================================================
--
-- Decision 61 asked for a projection table on the substrate, written in the
-- same transaction as the HQ-side mutation. Card B1 then settled the topology
-- in the direction that makes that impossible: the projection and the mutation
-- are on two different Postgres servers, `max_prepared_transactions` is 0 at
-- both ends, and Sign() is an allowlist that can only emit `authenticated`. No
-- transaction can contain both, and no restructuring of the mutation changes
-- that. B2 parked on exactly this; the park was correct.
--
-- Reading through removes the problem rather than managing it. There is no
-- projection, so "same transaction" is vacuous and the stale-permissive window
-- is not small — IT DOES NOT EXIST. A revoked assignment stops being visible at
-- the instant HQ's transaction commits.
--
-- The accepted standing cost, recorded at sign-off: HQ'S POSTGRES IS ON THE
-- NETWORK PATH OF EVERY RLS ROW CHECK. Every row of every read on the four
-- replicated tables costs at least one round trip to HQ. If HQ is down, the
-- substrate serves nothing. That was known and accepted; it is repeated here
-- because the person debugging a slow sync at 6am should find it in the file
-- rather than in a ledger.
--
-- ===========================================================================
-- CONNECTION PARAMETERS
-- ===========================================================================
-- Set before applying to point at a different HQ. All optional; the defaults
-- are the local dev stack.
--
--     set hq_fdw.host     = 'host.docker.internal';  -- HQ as seen FROM the
--                                                    -- substrate CONTAINER,
--                                                    -- not from your shell
--     set hq_fdw.port     = '5433';                  -- yumyums-dev-pg
--     set hq_fdw.dbname   = 'yumyums';
--     set hq_fdw.schema   = 'public';                -- 🛑 'production' on prod:
--                                                    -- docker-compose.prod.yml
--                                                    -- runs the backend with
--                                                    -- search_path=production,
--                                                    -- so migration 0073's views
--                                                    -- land THERE, not in public.
--     set hq_fdw.username = 'hq_sync_fdw';
--     set hq_fdw.password = '...';
--
-- 🛑 `hq_fdw.host` is the single most common way to get this wrong. The
-- substrate runs in a container; `localhost` there is the container, not your
-- machine. `host.docker.internal` is the default for that reason.
--
-- 🛑 There is NO DEFAULT PASSWORD, deliberately. `hq_sync_fdw` is created
-- NOLOGIN with no password by migration 0073 — an environment that has not
-- been consciously wired for sync fails closed instead of accepting a shared
-- committed credential. If `hq_fdw.password` is unset this file raises rather
-- than creating a mapping that cannot connect, because a mapping that cannot
-- connect fails later, further away, and as a confusing permission error.

-- ---------------------------------------------------------------------------
-- 1. The extension
-- ---------------------------------------------------------------------------
-- Proven installable at BOTH ends before this card resumed, by executing the C
-- symbol rather than reading pg_available_extensions: `dblink_get_connections()`
-- returned and `CREATE SERVER ... FOREIGN DATA WRAPPER postgres_fdw` succeeded
-- on both servers (park note §3, E4a-E4c). The control file being present does
-- not prove the shared object loads; running it does.
create extension if not exists postgres_fdw;

-- ---------------------------------------------------------------------------
-- 2. Server + user mapping
-- ---------------------------------------------------------------------------
-- 🛑 THE MAPPING IS FOR THE FUNCTION OWNER, NOT FOR `authenticated`.
--
-- postgres_fdw resolves a user mapping by the EFFECTIVE user. Every read of
-- these foreign tables happens inside a SECURITY DEFINER function in 0003, so
-- the effective user is that function's owner — the superuser applying this
-- file. `authenticated` is given no mapping and no privilege on the foreign
-- tables (section 4), so it cannot reach HQ through them by any route it can
-- name.
do $$
declare
  v_host text := coalesce(current_setting('hq_fdw.host',     true), 'host.docker.internal');
  v_port text := coalesce(current_setting('hq_fdw.port',     true), '5433');
  v_db   text := coalesce(current_setting('hq_fdw.dbname',   true), 'yumyums');
  v_user text := coalesce(current_setting('hq_fdw.username', true), 'hq_sync_fdw');
  v_pass text := nullif(current_setting('hq_fdw.password',   true), '');
begin
  if v_pass is null then
    raise exception
      'hq_fdw.password is not set. Migration 0073 creates hq_sync_fdw NOLOGIN with no '
      'password on purpose; enable it per environment with '
      '"ALTER ROLE hq_sync_fdw LOGIN PASSWORD ''<generated>''" and then '
      '"set hq_fdw.password = ''<same>''" before applying this file.';
  end if;

  -- ALTER rather than DROP/CREATE: dropping the server would cascade the
  -- foreign tables away, and re-running an idempotent file must not do that.
  if exists (select 1 from pg_foreign_server where srvname = 'hq_pg') then
    execute format(
      'alter server hq_pg options (set host %L, set port %L, set dbname %L)',
      v_host, v_port, v_db);
  else
    execute format(
      'create server hq_pg foreign data wrapper postgres_fdw '
      'options (host %L, port %L, dbname %L)',
      v_host, v_port, v_db);
  end if;

  if exists (select 1 from pg_user_mappings
              where srvname = 'hq_pg' and usename = current_user) then
    execute format(
      'alter user mapping for current_user server hq_pg options (set user %L, set password %L)',
      v_user, v_pass);
  else
    execute format(
      'create user mapping for current_user server hq_pg options (user %L, password %L)',
      v_user, v_pass);
  end if;
end
$$;

-- ---------------------------------------------------------------------------
-- 3. The three foreign tables
-- ---------------------------------------------------------------------------
-- Declared column-by-column rather than via IMPORT FOREIGN SCHEMA. IMPORT would
-- be shorter and is the wrong tool here: it makes the local shape a silent
-- function of whatever HQ happens to expose, so a future HQ migration that
-- widens a view would widen this substrate's reach without anyone editing this
-- file. Three explicit declarations are auditable in one screen.
--
-- The `table_name` options point at the three VIEWS migration 0073 creates —
-- never at `users` or `template_assignments` directly. `public.users` carries
-- `password_hash`; a foreign table over it would put that column on the wire.

-- 🛑 The `schema_name` option is the environment-dependent half and is driven by
-- `hq_fdw.schema` (default 'public', 'production' on prod). Migration 0073
-- creates its views with UNQUALIFIED names like every other migration in that
-- directory, so they follow HQ's own search_path — which prod sets to
-- `production` in docker-compose.prod.yml's DB_URL. A hard-coded 'public' here
-- would fail on prod ALONE, and would fail as "relation does not exist" inside a
-- policy, at request time, long after anyone was looking at this file.
do $$
declare
  v_schema text := coalesce(current_setting('hq_fdw.schema', true), 'public');
begin
  -- 3a. The assignment arm. Transposition of ResolveEntityAccess's EXISTS
  --     clause.
  --     🛑 Does NOT carry `assignment_role` — the resolver never filters on it,
  --     so an approver sees what an assignee sees. Inherited, knowing, and a
  --     separate card to change. The column is absent here so it cannot be
  --     filtered on by accident.
  if to_regclass('public.hq_template_assignees') is null then
    execute format(
      'create foreign table public.hq_template_assignees ('
      '  template_id text, user_id text'
      ') server hq_pg options (schema_name %L, table_name %L)',
      v_schema, 'hq_sync_template_assignees');
  else
    execute format(
      'alter foreign table public.hq_template_assignees options (set schema_name %L)', v_schema);
  end if;

  -- 3b. The admin arm. `roles && array['admin','superadmin']` is UNCONDITIONAL
  --     in the resolver — every admin sees every template. Inherited, knowing,
  --     and a separate card to change.
  if to_regclass('public.hq_user_roles') is null then
    execute format(
      'create foreign table public.hq_user_roles ('
      '  user_id text, roles text[]'
      ') server hq_pg options (schema_name %L, table_name %L)',
      v_schema, 'hq_sync_user_roles');
  else
    execute format(
      'alter foreign table public.hq_user_roles options (set schema_name %L)', v_schema);
  end if;

  -- 3c. field_id -> template_id, the resolution step the resolver performs for
  --     `field_response` entities. Resolving by field_id rather than
  --     submission_id is what makes DRAFT responses (submission_id IS NULL —
  --     0001's comment calls that load-bearing) resolvable at all.
  if to_regclass('public.hq_field_templates') is null then
    execute format(
      'create foreign table public.hq_field_templates ('
      '  field_id text, template_id text'
      ') server hq_pg options (schema_name %L, table_name %L)',
      v_schema, 'hq_sync_field_templates');
  else
    execute format(
      'alter foreign table public.hq_field_templates options (set schema_name %L)', v_schema);
  end if;

  -- 3d. 🛑 THE APPROVER ARM — added by card `sync-rxdb-write-policies`
  --     (overnight-20260802, A2) under ledger T-30 decision 111,
  --     consequence (2). Its HQ-side counterpart is migration
  --     0074_sync_fdw_approver_view.sql.
  --
  --     READ §3a AGAIN BEFORE TOUCHING EITHER OF THESE TWO BLOCKS. §3a says
  --     `hq_template_assignees` does NOT carry `assignment_role` so it cannot
  --     be filtered on by accident, and THAT IS STILL TRUE — 3a above is
  --     byte-unchanged. This is a DIFFERENT relation, over a DIFFERENT view,
  --     which is PRE-FILTERED to `assignment_role = 'approver'` on HQ's side.
  --
  --     The asymmetry is decision 111 itself, not an inconsistency to tidy:
  --
  --       READS   scope by `hq_can_see_template` / `hq_can_see_field`, which
  --               name `hq_template_assignees` and never this table. An
  --               approver sees exactly what an assignee sees. UNCHANGED.
  --       WRITES  to `submission_rejections` scope by `hq_can_approve_field`,
  --               which names THIS table and never the other. A plain
  --               assignee cannot reject their own work.
  --
  --     🛑 A future edit that "unifies" the two by adding an `assignment_role`
  --     column to 3a and deleting this block has handed every assignee the
  --     approver's write. A future edit that makes READS use this table has
  --     blinded every approver to the checklist they are supposed to approve
  --     (0073 §1's warning, verbatim). Variants W9 and WP5 in
  --     backend/internal/sync/rowvisibility_rls_test.go fail in one direction
  --     each; keep both.
  if to_regclass('public.hq_template_approvers') is null then
    execute format(
      'create foreign table public.hq_template_approvers ('
      '  template_id text, user_id text'
      ') server hq_pg options (schema_name %L, table_name %L)',
      v_schema, 'hq_sync_template_approvers');
  else
    execute format(
      'alter foreign table public.hq_template_approvers options (set schema_name %L)', v_schema);
  end if;
end
$$;

-- ---------------------------------------------------------------------------
-- 4. 🛑 Lock the foreign tables away from every PostgREST-reachable role
-- ---------------------------------------------------------------------------
-- WITHOUT THIS SECTION THE CARD IS A NET LOSS. PostgREST exposes every relation
-- in the `public` schema that its role can read — including foreign tables. A
-- readable `hq_user_roles` is a single GET that returns the entire company's
-- role map; a readable `hq_template_assignees` returns who is assigned to what,
-- for everyone. That would leak MORE than the policies below protect.
--
-- Postgres grants no privilege on a new relation by default, so these revokes
-- LOOK like belt-and-braces — but "default privileges are currently empty" is a
-- property of a database that a later ALTER DEFAULT PRIVILEGES can change
-- silently, and this is exactly the class of check that must not depend on a
-- subject set staying empty. The suite asserts the refusal directly (variants
-- V14-V16) rather than trusting it.
--
-- 🛑 AND ON THIS IMAGE THEY WERE NOT EMPTY — MEASURED, run overnight-20260801
-- finding F4. Straight after creating the three foreign tables:
--
--   hq_user_roles -> supabase_admin=arwdDxt/supabase_admin
--                  | postgres=arwdDxt/supabase_admin
--                  | service_role=arwdDxt/supabase_admin
--   has_table_privilege('service_role','public.hq_user_roles','SELECT') = true
--
-- `service_role` IS PostgREST-reachable — `authenticator` is a member of it
-- (pg_auth_members: authenticator -> anon, authenticated, service_role), which
-- is precisely what makes the BYPASSRLS control token in the suites work. So a
-- service_role token was a single GET away from HQ's whole role map, through the
-- one role RLS cannot stop. It is on the revoke list below now.
--
-- This is NOT NEW EXPOSURE and is not being reported as a breach: service_role
-- holds BYPASSRLS anyway, and Sign() refuses to mint it (see
-- TestMint_NeverMintsServiceRole), so no token this backend issues can reach it.
-- It is on the list because the paragraph above is this section's own stated
-- argument, and the argument turned out to be load-bearing rather than
-- theoretical.
--
-- `postgres` is deliberately NOT on the list: authenticator is not a member of
-- it (verified in the same query), so it is not PostgREST-reachable, and it is
-- the image's own superuser-adjacent owner role — revoking from it would be
-- fighting the platform rather than closing a door.
--
-- The SECURITY DEFINER functions in 0003 are unaffected: they run as the owner.
revoke all on public.hq_template_assignees from anon, authenticated, service_role, public;
revoke all on public.hq_user_roles         from anon, authenticated, service_role, public;
revoke all on public.hq_field_templates    from anon, authenticated, service_role, public;
-- 🛑 §3d's relation. A readable `hq_template_approvers` is one GET returning
-- every approver of every template — narrower than `hq_template_assignees` but
-- the same KIND of leak, and the F4 measurement above (service_role held SELECT
-- on a brand-new foreign table on this image) applies to it identically. It is
-- on this list from the moment it exists, not after somebody measures it again.
-- Variant V20 asserts the refusal directly rather than trusting the revoke.
revoke all on public.hq_template_approvers from anon, authenticated, service_role, public;

-- PostgREST caches the schema; new relations are 404 until it reloads.
notify pgrst, 'reload schema';
