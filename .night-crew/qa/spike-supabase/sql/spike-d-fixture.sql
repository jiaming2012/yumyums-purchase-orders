-- spike-d-fixture.sql — SPIKE D's substrate. Card `spike-d-realtime-live`, B-62.
--
-- ⚠ Runs against the LOCAL throwaway spike stack only (compose project
--   `spike-supabase`). Never a hosted Supabase project, never production,
--   never :5433, never :5434. See spike-d-realtime.sh's isolation assertions,
--   which are executable checks and not comments.
--
-- Idempotent: safe to re-run against a live stack. `spike-d-teardown.sql` is
-- its exact inverse.
--
-- ===========================================================================
-- 🛑 WHY THESE ARE CLONES AND NOT THE REAL SYNC TABLES
-- ===========================================================================
-- The substrate already carries HQ's four real replicated tables
-- (`checklist_templates`, `checklist_submissions`, `submission_responses`,
-- `submission_rejections` — sync-schema/sql/0001_sync_tables.sql, applied by
-- env-up.sh). Driving the filter against those directly would be the obvious
-- move, and it was rejected for two independent reasons:
--
--   1. THEY ARE SHARED, AND A SPIKE THAT LEAVES ROWS BEHIND REDS A COMMITTED
--      GO SUITE. That is not hypothetical — spike B's own first G2 run went red
--      on four subtests of TestJWTBridgeRLS for exactly this, and card C's
--      teardown exists because of it. This card writes to NO shared table at
--      all, so the failure mode is removed rather than mitigated.
--
--   2. THEIR RLS PREDICATE IS CURRENTLY UNEVALUABLE ON THIS STACK. The real
--      policies read `hq_user_roles` / `hq_template_assignees`, which are
--      postgres_fdw foreign tables pointing at an HQ source Postgres that is
--      not running (`could not connect to server "hq_pg"`). An `authenticated`
--      subscriber would therefore see nothing for a reason that has nothing to
--      do with the filter — and "the in-scope row did not arrive" is precisely
--      the observation this card must be able to attribute. Mixing an
--      unevaluable RLS predicate into a filter measurement makes the verdict
--      un-assignable.
--
-- 🛑 AND THE CLONE IS NOT ASSERTED TO BE REPRESENTATIVE — IT IS CHECKED.
--    `create table ... (like <real> including all)` copies column names, types,
--    defaults, constraints and indexes from the real table by construction, and
--    spike-d-realtime.sh then queries the catalog and REFUSES TO RUN unless the
--    clone's column list, the clone's `relreplident`, and the clone's
--    `supabase_realtime` publication membership are identical to the real
--    table's. Realtime's filter evaluation
--    (`realtime.is_visible_through_filters` -> `realtime.check_equality_op`)
--    reads the WAL column list and the column's type oid; it never reads the
--    table's NAME. So an identical column set at an identical replica identity
--    in the same publication is the same measurement.
--
-- ===========================================================================
-- 🛑 RLS IS ENABLED WITH NO POLICIES, AND THAT IS DELIBERATE
-- ===========================================================================
-- This mirrors the posture 0001 leaves the real tables in before 0003 writes
-- the read policies, and it is the ONLY posture that keeps this card out of
-- permissions-boundary territory (its PARK note): writing a policy here would
-- be inventing a permission semantic, which the card explicitly may not do.
--
-- The subscriber is therefore `service_role` (rolbypassrls=t, already present
-- and already used by spike B and spike C for exactly this reason). The
-- consequence is the point: RLS is a CONSTANT for every channel in this run, so
-- the ONLY thing that can differ between the filtered and unfiltered channels
-- is the filter. The unfiltered controls make that a measurement rather than an
-- argument — if RLS were the discriminator, the unfiltered channels would go
-- quiet too, and the script reds.

-- ---------------------------------------------------------------------------
-- 1. The clones
-- ---------------------------------------------------------------------------
create table if not exists public.spike_d_templates
  (like public.checklist_templates including all);

create table if not exists public.spike_d_submissions
  (like public.checklist_submissions including all);

create table if not exists public.spike_d_responses
  (like public.submission_responses including all);

-- ---------------------------------------------------------------------------
-- 2. Grants
-- ---------------------------------------------------------------------------
-- `realtime.subscription_check_filters` resolves the filterable column set with
-- has_column_privilege(<claims role>, ...). A role with no SELECT grant gets an
-- EMPTY column list and every filter is rejected as `invalid column for filter`
-- — which would look exactly like an unusable filter grammar. Granting is what
-- keeps that confusion out of the verdict.
grant select, insert, update on public.spike_d_templates   to service_role, authenticated;
grant select, insert, update on public.spike_d_submissions to service_role, authenticated;
grant select, insert, update on public.spike_d_responses   to service_role, authenticated;
revoke all on public.spike_d_templates   from anon;
revoke all on public.spike_d_submissions from anon;
revoke all on public.spike_d_responses   from anon;

-- ---------------------------------------------------------------------------
-- 3. RLS on, no policies — deny-all for `authenticated`, bypassed by
--    `service_role`. See the block above.
-- ---------------------------------------------------------------------------
alter table public.spike_d_templates   enable row level security;
alter table public.spike_d_submissions enable row level security;
alter table public.spike_d_responses   enable row level security;

-- ---------------------------------------------------------------------------
-- 4. Realtime enrolment + REPLICA IDENTITY FULL
-- ---------------------------------------------------------------------------
-- REPLICA IDENTITY FULL is the half of B-62's stated risk surface: the card
-- names "Realtime's filter semantics vs REPLICA IDENTITY FULL payloads" as the
-- quiet-failure surface. The real tables carry `relreplident='f'`; the clones
-- must match, and spike-d-realtime.sh asserts that they do.
do $$
begin
  alter publication supabase_realtime add table public.spike_d_templates;
exception when duplicate_object then null;
end;
$$;
do $$
begin
  alter publication supabase_realtime add table public.spike_d_submissions;
exception when duplicate_object then null;
end;
$$;
do $$
begin
  alter publication supabase_realtime add table public.spike_d_responses;
exception when duplicate_object then null;
end;
$$;

alter table public.spike_d_templates   replica identity full;
alter table public.spike_d_submissions replica identity full;
alter table public.spike_d_responses   replica identity full;

-- PostgREST caches the schema; this is the reload signal. Harmless here (this
-- card never speaks PostgREST) and kept so the fixture leaves the stack in the
-- state the next reader expects.
notify pgrst, 'reload schema';
