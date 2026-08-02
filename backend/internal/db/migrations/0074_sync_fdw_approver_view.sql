-- +goose Up
-- ===========================================================================
-- 0074 — the HQ-SIDE half of the sync substrate's WRITE policies.
-- Card `sync-rxdb-write-policies` (overnight-20260802, A2).
-- Authority: ledger T-30 decision 111, consequence (2).
-- ===========================================================================
--
-- 🛑 READ 0073's BANNER FIRST. This migration is its sibling and inherits every
-- property it argues for: read-only views, never `IMPORT FOREIGN SCHEMA`, never
-- a base table (because `public.users` carries `password_hash`), not
-- `security_invoker`, and `current_schema()` rather than a literal `public`
-- because production runs with `search_path=production`.
--
-- ---------------------------------------------------------------------------
-- WHY A FOURTH VIEW EXISTS AT ALL — and why it is a NEW relation rather than a
-- column added to `hq_sync_template_assignees`
-- ---------------------------------------------------------------------------
--
-- 0073 §1 and sync-schema/sql/0002_hq_fdw.sql §3a both record the same property
-- in the strongest terms available to a comment:
--
--     INHERITED PROPERTY 1 — `assignment_role` IS NEVER FILTERED ON. An
--     approver sees exactly what an assignee sees. …The column is not even
--     carried across the FDW so it cannot be filtered on by accident.
--
-- That is a READ property and it is UNCHANGED by this migration. Decision 111
-- makes writes — and only writes — the place `assignment_role` starts to
-- matter:
--
--     a new `public.hq_can_approve_template(tid)` predicate = `EXISTS` an
--     assignment with `assignment_role = 'approver'` OR the unconditional
--     `roles && ARRAY['admin','superadmin']` admin arm. Reads keep the old
--     property untouched; only the approval WITH CHECK uses the new one. That
--     asymmetry is the decision, not a side effect of it.
--
-- 🛑 SO THE APPROVER ARM IS ITS OWN RELATION, PRE-FILTERED, RATHER THAN A
-- COLUMN ON THE EXISTING ONE. Three reasons, in order of how much they matter:
--
--   1. `hq_sync_template_assignees` stays BYTE-IDENTICAL, so 0002 §3a's
--      "cannot be filtered on by accident" survives literally rather than
--      becoming a promise. The read path never names the relation below and
--      the write path never names the relation above.
--   2. This view contains ONLY approver rows. It cannot be used to WIDEN a
--      read — the worst a misuse can do is show fewer rows, never more. A
--      nullable `assignment_role` column on the shared view has the opposite
--      failure mode.
--   3. It is auditable in one screen, which is the same argument 0002 §3 makes
--      for declaring three foreign tables by hand instead of importing a
--      schema.
--
-- 🛑 BOTH DISJUNCTS ARE CARRIED, and that is not decoration. HQ can express a
-- ROLE-assigned approver — `(template_id, 'role', 'manager', 'approver')` — and
-- an approver view written with only the `assignee_type = 'user'` arm would
-- refuse every one of them while looking perfectly healthy on a user-assigned
-- fixture. Variant WP5b in backend/internal/sync/rowvisibility_rls_test.go is a
-- role-reached approver for exactly this reason: delete the role disjunct below
-- and that one line turns red, and nothing else does.
--
-- ---------------------------------------------------------------------------
-- 🛑 THIS MIGRATION ADDS ONE COLUMN TO 0073's "MAY NO LONGER BE RETYPED" LIST
-- ---------------------------------------------------------------------------
-- 0073's banner enumerates the columns a future migration can no longer
-- `ALTER TYPE`, because a view is a hard dependency on the columns it names.
-- The list gains one entry here:
--
--     template_assignments.assignment_role
--
-- (`users.id`, `users.roles`, `template_assignments.template_id`,
-- `.assignee_type`, `.assignee_id` were already on it and are named again
-- below.) ADDING a column and DROPPING an unreferenced one remain unaffected.
-- If a future migration genuinely needs to retype `assignment_role`, the
-- pattern is 0073's: DROP the view, ALTER, recreate — in that migration.
--
-- ---------------------------------------------------------------------------
-- NO EXPLICIT BEGIN;/COMMIT; — see 0073's banner, finding F2 of run
-- overnight-20260801. goose already wraps this file in one transaction, and an
-- inner COMMIT commits goose's own transaction early, leaving a schema the
-- version table cannot describe. Do not add one.
-- ---------------------------------------------------------------------------

-- ---------------------------------------------------------------------------
-- 1. hq_sync_template_approvers — the approver arm of decision 111
-- ---------------------------------------------------------------------------
-- The join is 0073 §1's join, verbatim, plus ONE predicate. Written that way on
-- purpose: a reader comparing the two files should see exactly one difference,
-- and that difference is the whole of the operator's answer.
--
-- What is deliberately NOT here: the admin arm. `roles && ARRAY['admin',
-- 'superadmin']` is a property of the USER, not of the (template, user) pair —
-- 0073 §2's argument, unchanged — so it stays in `hq_sync_user_roles` and the
-- substrate-side predicate ORs the two. Materialising admins into this view
-- would be an admins x templates cross product recomputed on every template
-- insert, which is precisely what decision 92 bought its way out of.
CREATE OR REPLACE VIEW hq_sync_template_approvers AS
  SELECT DISTINCT
         ta.template_id::text AS template_id,
         u.id::text           AS user_id
    FROM template_assignments ta
    JOIN users u
      ON (    (ta.assignee_type = 'user' AND u.id::text     = ta.assignee_id)
           OR (ta.assignee_type = 'role' AND ta.assignee_id = ANY(u.roles)) )
   WHERE ta.assignment_role = 'approver';

COMMENT ON VIEW hq_sync_template_approvers IS
  'Read-through source for the sync substrate''s WRITE policies (card sync-rxdb-write-policies, '
  'ledger T-30 decision 111). The APPROVER arm only — assignment_role = ''approver''. '
  'Deliberately a separate relation from hq_sync_template_assignees so the READ path keeps '
  'INHERITED PROPERTY 1 (an approver sees what an assignee sees) byte-for-byte. Reads must '
  'never name this view; writes to submission_rejections are the only consumer.';

-- ---------------------------------------------------------------------------
-- 2. The grant. FOUR relations now, still enumerated one by one.
-- ---------------------------------------------------------------------------
-- 🛑 0073 §4's standing instruction, restated because this migration is the
-- first one that had to obey it: a migration that adds a table must NOT
-- casually `GRANT SELECT ON ALL TABLES IN SCHEMA public TO hq_sync_fdw`. The
-- whole point of enumerating relations is that the list is auditable. This one
-- adds exactly one line.
--
-- No role creation, no USAGE grant: 0073 created `hq_sync_fdw` NOLOGIN with no
-- password and granted it USAGE on current_schema(), and both are still in
-- force. This migration is strictly additive to that surface.
GRANT SELECT ON hq_sync_template_approvers TO hq_sync_fdw;

-- +goose Down
-- ===========================================================================
-- Strictly the inverse, and DELIBERATELY NOT A COPY OF 0073's Down.
-- ===========================================================================
-- 0073's Down carries a `pg_shdepend` interlock because it is the migration
-- that CREATES the cluster-wide `hq_sync_fdw` role, and dropping a role other
-- databases still depend on is somebody else's outage. This migration creates
-- no role, so it has no such hazard and must not pretend to: dropping the view
-- removes its ACL entry with it, which is why there is no REVOKE here either
-- (0073's Down records the same reasoning — a REVOKE that errors because
-- another database's Down already dropped the role is a state this cluster can
-- genuinely be in).
--
-- After this Down the substrate's `hq_template_approvers` foreign table resolves
-- to a missing relation and every write to `submission_rejections` fails inside
-- a policy at request time. That is the correct direction to fail: approvals
-- stop being writable rather than becoming writable by anyone.
DROP VIEW IF EXISTS hq_sync_template_approvers;
