-- +goose Up
-- ===========================================================================
-- 0073 — the HQ-SIDE half of the sync substrate's row-visibility RLS.
-- Card `sync-rxdb-row-visibility-rls` (overnight-20260801, B2).
-- ===========================================================================
--
-- 🛑 WHAT THIS MIGRATION IS FOR, AND WHY IT LOOKS BACKWARDS
--
-- Nothing in HQ's own backend reads the three views below. They exist so that
-- ANOTHER Postgres — the self-hosted Supabase sync substrate, a different
-- server — can read them through `postgres_fdw` foreign tables and evaluate
-- row level security against HQ's LIVE data.
--
-- Ledger decision 92 (T-28, 2026-07-29) reverses decision 61 and chooses this
-- shape over a written projection. The reason is the one property no
-- asynchronous mechanism has: THERE IS NOTHING TO KEEP IN SYNC. A revoked
-- assignment stops being visible on the substrate at the instant HQ's
-- transaction commits, because the substrate was never holding a copy — it
-- reads through. Zero stale-permissive window, not a small one.
--
-- The accepted standing cost, recorded at sign-off and repeated here so it is
-- not rediscovered as a surprise: HQ's POSTGRES IS NOW ON THE NETWORK PATH OF
-- EVERY RLS ROW CHECK ON THE SUBSTRATE. If HQ's database is down or slow, the
-- substrate's reads are down or slow.
--
-- The substrate-side half is sync-schema/sql/0002_hq_fdw.sql (foreign tables)
-- and sync-schema/sql/0003_rls_policies.sql (the policies). Neither is a goose
-- migration, deliberately: anything under this directory runs against HQ's
-- database on every backend start, and `authenticated`/`anon` do not exist as
-- roles here.
--
-- ---------------------------------------------------------------------------
-- WHY VIEWS AND NOT `IMPORT FOREIGN SCHEMA` ON THE BASE TABLES
-- ---------------------------------------------------------------------------
-- Because `public.users` carries `password_hash`.
--
-- A foreign table over `users` would put every column of that table on the
-- wire to a second server and inside the reach of anything running there. The
-- views below are the narrowest surface that answers the ported predicate and
-- nothing else: no email, no phone number, no salary, no hash. The remote role
-- is granted SELECT on THESE VIEWS ONLY and on no base table, so the substrate
-- cannot widen its own view by asking for a different relation.
--
-- The views are NOT `security_invoker`. That is on purpose: they run with the
-- owner's privileges, which is what lets the remote role read them while
-- holding no privilege at all on `users` or `template_assignments`.
--
-- ---------------------------------------------------------------------------
-- 🛑 A CONSTRAINT THIS MIGRATION PLACES ON EVERY FUTURE ONE — read before
--    writing 0074+
-- ---------------------------------------------------------------------------
-- A view is a hard dependency on the columns it names. From here on, a
-- migration that ALTERS THE TYPE of any of these columns will FAIL:
--
--     users.id, users.roles
--     template_assignments.template_id, .assignee_type, .assignee_id
--     checklist_fields.id, .section_id
--     checklist_sections.id, .template_id
--
-- Measured, not assumed — `ALTER TABLE users ALTER COLUMN roles TYPE varchar[]`
-- against a migrated database returns:
--     ERROR: cannot alter type of a column used by a view or rule
--     DETAIL: rule _RETURN on view hq_sync_template_assignees depends on
--             column "roles"
--
-- ADDING a column and DROPPING an unreferenced one are both unaffected (both
-- verified). If a future migration genuinely needs to retype one of the columns
-- above, the pattern is: DROP the three views, ALTER, recreate them — in that
-- migration, not by weakening this one.
BEGIN;

-- ---------------------------------------------------------------------------
-- 1. hq_sync_template_assignees — the assignment arm of ResolveEntityAccess
-- ---------------------------------------------------------------------------
-- 🛑 THIS IS A TRANSPOSITION, NOT A NEW PREDICATE. The Go resolver
-- (backend/internal/sync/ops.go, ResolveEntityAccess) asks "given a template,
-- which users?"; a policy asks "given a user, which rows?". Same relation, read
-- along the other axis. The EXISTS arm of that query, verbatim:
--
--     EXISTS (SELECT 1 FROM template_assignments ta
--             WHERE ta.template_id = $1::uuid
--               AND ( (ta.assignee_type = 'user' AND u.id::text = ta.assignee_id)
--                  OR (ta.assignee_type = 'role' AND ta.assignee_id = ANY(u.roles)) ))
--
-- becomes the join below. Both disjuncts, same order, same operators.
--
-- 🛑 INHERITED PROPERTY 1, KNOWING AND PRESERVED: `assignment_role` IS NEVER
-- FILTERED ON. `template_assignments.assignment_role` is `'assignee'` or
-- `'approver'`, and the resolver reads neither — so an approver sees exactly
-- what an assignee sees. That is not an oversight being copied forward; it is
-- the shipped behaviour, and narrowing it is a SEPARATE CARD. Do not add
-- `WHERE ta.assignment_role = 'assignee'` here as a "tightening": it would
-- silently remove every approver's ability to see the checklist they are
-- supposed to approve.
--
-- DISTINCT mirrors the resolver's own `SELECT DISTINCT`: a user can match both
-- disjuncts at once (named directly AND holding an assigned role), and without
-- it the same (template, user) pair appears twice.
--
-- Both id columns are cast to text because the substrate's tables carry
-- `id text primary key` (RxDB documents carry client-generated string ids —
-- sync-schema/sql/0001_sync_tables.sql, contract item 1). The cast happens HERE
-- rather than in the policy so the foreign table's declared types match what
-- it is compared against, and no per-row cast is pushed across the wire.
CREATE OR REPLACE VIEW hq_sync_template_assignees AS
  SELECT DISTINCT
         ta.template_id::text AS template_id,
         u.id::text           AS user_id
    FROM template_assignments ta
    JOIN users u
      ON (    (ta.assignee_type = 'user' AND u.id::text     = ta.assignee_id)
           OR (ta.assignee_type = 'role' AND ta.assignee_id = ANY(u.roles)) );

COMMENT ON VIEW hq_sync_template_assignees IS
  'Read-through source for the sync substrate''s RLS (card sync-rxdb-row-visibility-rls). '
  'Transposition of ResolveEntityAccess''s assignment arm. Deliberately does NOT filter on '
  'assignment_role — an approver sees what an assignee sees. Narrowing that is a separate card.';

-- ---------------------------------------------------------------------------
-- 2. hq_sync_user_roles — the admin arm of ResolveEntityAccess
-- ---------------------------------------------------------------------------
-- 🛑 INHERITED PROPERTY 2, KNOWING AND PRESERVED: the admin arm is
-- UNCONDITIONAL. In the resolver it is a free-standing disjunct —
--
--     WHERE u.roles && ARRAY['admin','superadmin'] OR EXISTS (...)
--
-- gated on nothing: not on assignment, not on template, not on app. EVERY
-- ADMIN SEES EVERY TEMPLATE. The resolver's own comment says why (myChecklists
-- grants `roles && {admin,superadmin}` view-all, and live sync must mirror the
-- access the UI already gives, or an admin's own edit never converges on their
-- own second device). Restricting it is a SEPARATE CARD.
--
-- This is a separate view rather than extra rows in hq_sync_template_assignees
-- because `roles && ARRAY[...]` is a property of the USER, not of the
-- (template, user) pair. Materialising it into the join would be an
-- admins x templates cross product that has to be recomputed on every template
-- insert — and one of the two things decision 92 bought was not having to
-- recompute anything.
--
-- The whole `roles` array is exposed rather than a precomputed boolean so the
-- predicate that consumes it stays a visible transposition of the Go rather
-- than a name that has to be trusted.
CREATE OR REPLACE VIEW hq_sync_user_roles AS
  SELECT u.id::text AS user_id,
         u.roles    AS roles
    FROM users u;

COMMENT ON VIEW hq_sync_user_roles IS
  'Read-through source for the sync substrate''s RLS (card sync-rxdb-row-visibility-rls). '
  'The admin arm of ResolveEntityAccess is unconditional: every admin/superadmin sees every '
  'template. Restricting that is a separate card.';

-- ---------------------------------------------------------------------------
-- 3. hq_sync_field_templates — the field_response resolution step
-- ---------------------------------------------------------------------------
-- ResolveEntityAccess resolves a `field_response` entity BEFORE it evaluates
-- the two arms above:
--
--     SELECT s.template_id FROM checklist_fields f
--      JOIN checklist_sections s ON f.section_id = s.id
--      WHERE f.id = $1
--
-- The park note recorded this as an open gap (§4d), because under the REJECTED
-- projection design neither `checklist_fields` nor `checklist_sections` exists
-- on the substrate and a THIRD projection would have needed a writer. Reading
-- through closes the gap for free: the tables are right here.
--
-- Resolving by `field_id` and not by `submission_id` is the faithful port and
-- also the only one that works. `submission_responses.submission_id` is
-- NULLABLE on purpose — a DRAFT response has no submission yet, and drafts are
-- exactly what a crew member fills offline. Resolving through the submission
-- would leave every draft unresolvable, which is the one case that must work.
CREATE OR REPLACE VIEW hq_sync_field_templates AS
  SELECT f.id::text          AS field_id,
         s.template_id::text AS template_id
    FROM checklist_fields f
    JOIN checklist_sections s ON f.section_id = s.id;

COMMENT ON VIEW hq_sync_field_templates IS
  'Read-through source for the sync substrate''s RLS (card sync-rxdb-row-visibility-rls). '
  'field_id -> template_id, the resolution step ResolveEntityAccess performs for '
  'field_response entities. Resolves by field_id, not submission_id, because draft '
  'responses have a NULL submission_id.';

COMMIT;

-- ---------------------------------------------------------------------------
-- 4. hq_sync_fdw — the login role the substrate connects AS
-- ---------------------------------------------------------------------------
-- 🛑 CREATED **NOLOGIN AND WITHOUT A PASSWORD**, ON PURPOSE.
--
-- This file is committed to a public-facing repository and runs on every
-- backend start in every environment including production. A password written
-- here would be a committed credential, and one that every environment shares.
--
-- So the migration creates the role and the privilege surface — the parts that
-- must be identical everywhere and must not drift — and STOPS. Enabling the
-- role is a deliberate, per-environment operator step:
--
--     ALTER ROLE hq_sync_fdw LOGIN PASSWORD '<generated, per environment>';
--
-- Until that is run the role cannot connect, which means an environment that
-- has not been consciously wired for sync fails CLOSED rather than accepting a
-- default credential. The Go suite performs this step explicitly against its
-- own throwaway test database, which is also where the step is demonstrated.
--
-- Privileges are least-privilege and deliberately narrow:
--   * USAGE on schema public — needed to name anything at all.
--   * SELECT on the THREE VIEWS above and NOTHING ELSE. No base table, no
--     write privilege of any kind, on anything.
-- The views are not security_invoker, so they resolve with their owner's rights
-- and this role never needs (and never gets) a privilege on `users`.
--
-- 🛑 A future migration that adds a table must NOT casually
-- `GRANT SELECT ON ALL TABLES IN SCHEMA public TO hq_sync_fdw`. The whole point
-- of enumerating three relations is that the list is auditable.

-- +goose StatementBegin
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hq_sync_fdw') THEN
    CREATE ROLE hq_sync_fdw NOLOGIN;
  END IF;
END
$$;
-- +goose StatementEnd

COMMENT ON ROLE hq_sync_fdw IS
  'postgres_fdw read-through role for the sync substrate (card sync-rxdb-row-visibility-rls). '
  'Created NOLOGIN with no password; enable per environment with ALTER ROLE ... LOGIN PASSWORD. '
  'Holds SELECT on hq_sync_template_assignees, hq_sync_user_roles, hq_sync_field_templates only.';

-- 🛑 `current_schema()`, NOT a hard-coded `public`. Production runs this
-- backend with `search_path=production` (docker-compose.prod.yml's DB_URL), so
-- the three views above — created with unqualified names, like every other
-- migration in this directory — land in `production` there and in `public` on
-- dev. A literal `GRANT USAGE ON SCHEMA public` would therefore grant on the
-- wrong schema in exactly one environment, and would do it silently: the
-- migration succeeds, the views exist, and the remote role simply cannot see
-- them. Following search_path keeps the grant attached to whichever schema the
-- views actually went into.
--
-- The matching substrate-side knob is `hq_fdw.schema` in
-- sync-schema/sql/0002_hq_fdw.sql — set it to 'production' when pointing the
-- foreign server at prod.
-- +goose StatementBegin
DO $$
BEGIN
  EXECUTE format('GRANT USAGE ON SCHEMA %I TO hq_sync_fdw', current_schema());
END
$$;
-- +goose StatementEnd

GRANT SELECT ON hq_sync_template_assignees TO hq_sync_fdw;
GRANT SELECT ON hq_sync_user_roles          TO hq_sync_fdw;
GRANT SELECT ON hq_sync_field_templates     TO hq_sync_fdw;

-- +goose Down
BEGIN;

REVOKE SELECT ON hq_sync_field_templates     FROM hq_sync_fdw;
REVOKE SELECT ON hq_sync_user_roles          FROM hq_sync_fdw;
REVOKE SELECT ON hq_sync_template_assignees  FROM hq_sync_fdw;

DROP VIEW IF EXISTS hq_sync_field_templates;
DROP VIEW IF EXISTS hq_sync_user_roles;
DROP VIEW IF EXISTS hq_sync_template_assignees;

COMMIT;

-- Outside the transaction: DROP ROLE is not transactional-safe to pair with the
-- revokes above on every Postgres, and a role left behind is harmless (NOLOGIN,
-- no privileges after the revokes) whereas a half-rolled-back drop is not.
--
-- The schema USAGE revoke lives here rather than in the transaction above for
-- the same reason the grant needs a DO block: the schema name is
-- environment-dependent (`public` on dev, `production` on prod).
-- +goose StatementBegin
DO $$
BEGIN
  EXECUTE format('REVOKE USAGE ON SCHEMA %I FROM hq_sync_fdw', current_schema());
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'hq_sync_fdw') THEN
    DROP ROLE hq_sync_fdw;
  END IF;
END
$$;
-- +goose StatementEnd
