-- hq-source-fixture.sql — the data Spike B actually migrates.
--
-- Card S `spike-b-migration-rehearsal`. Applied to the throwaway `spike-b-hq`
-- Postgres only. Never HQ, never :5433, never :5434.
--
-- ═══════════════════════════════════════════════════════════════════════════
-- EVERY UUID HERE IS FIXED, NOT gen_random_uuid()
--
-- HQ's real DDL defaults these to gen_random_uuid(). The subset schema
-- deliberately drops the default so the fixture can pin every id. That is not a
-- convenience: the RLS assertions compare a JWT's `sub` claim against a migrated
-- row's owner_id, and the RxDB assertions compare an exact id SET. Both need ids
-- that are the same on every run, in the source and in the substrate, so that
-- "the migration preserved identity" is a checkable claim rather than a vibe.
--
-- ═══════════════════════════════════════════════════════════════════════════
-- WHY THIS SHAPE — every row below is load-bearing for one assertion
--
-- Three users, two apps, three templates, seven submissions, fifteen responses.
-- Small, but every axis the substrate discriminates on is exercised, and each
-- has a NEGATIVE control so "let everything through" and "discriminates" cannot
-- look the same:
--
--  IDENTITY axis      Bob holds a live `inventory` grant and owns NO inventory
--                     row. Alice owns one. If Bob can see Alice's inventory
--                     submission, the owner check is not working — and no
--                     entitlement check could have caught it, because Bob's
--                     entitlement is genuine.
--
--  ENTITLEMENT axis   Alice OWNS an inventory submission and holds NO live
--                     `inventory` grant. If she can see it, the grant check is
--                     not working — and no owner check could have caught it,
--                     because her ownership is genuine.
--
--  ROLE-vs-USER       Two of the four app_permissions rows grant by ROLE TIER
--  grant resolution   (no user_id at all); two grant to a single user. A
--                     migration that copied only the user-shaped rows would
--                     produce a projection missing Alice and Carol entirely.
--
--  ARCHIVED filter    One template is archived and carries a submission. It
--                     must NOT migrate. A rehearsal that moves every row cannot
--                     tell a filter that works from a filter that is absent.
--
--  DRAFT response     One submission_response has submission_id IS NULL — HQ's
--                     in-progress draft shape. It must not be counted into any
--                     migrated payload.
--
--  JSONB payload      template_snapshot and every response `value` are real
--                     JSONB. A migration never tested against JSONB has not
--                     been tested against HQ's actual rows.
--
-- Idempotent (`on conflict do nothing`) for hand-debugging; the script destroys
-- the container each run regardless.

begin;

-- ---------------------------------------------------------------------------
-- Users. Alice is a manager, Bob a team member, Carol an admin — the three
-- tiers, because the grant projection resolves role-tier grants through them.
-- ---------------------------------------------------------------------------
insert into users (id, email, display_name, role, status) values
  ('11111111-1111-4111-8111-111111111111', 'alice@yumyums.test', 'Alice M.', 'manager',     'active'),
  ('22222222-2222-4222-8222-222222222222', 'bob@yumyums.test',   'Bob R.',   'team_member', 'active'),
  ('33333333-3333-4333-8333-333333333333', 'carol@yumyums.test', 'Carol T.', 'admin',       'active')
on conflict (id) do nothing;

-- ---------------------------------------------------------------------------
-- Apps. Two, with real HQ slugs, because one app cannot show an entitlement
-- axis discriminating.
-- ---------------------------------------------------------------------------
insert into hq_apps (id, slug, name, icon, enabled) values
  ('0a000000-0000-4000-8000-000000000001', 'operations', 'Operations', '📋', true),
  ('0a000000-0000-4000-8000-000000000002', 'inventory',  'Inventory',  '📦', true)
on conflict (id) do nothing;

-- ---------------------------------------------------------------------------
-- Permissions — HQ's dual shape, both halves populated.
--
--   role-tier grants:  operations→manager      resolves to Alice
--                      inventory →admin        resolves to Carol
--   per-user grants:   operations→Bob
--                      inventory →Bob
--
-- Resolved projection (what the migration must produce, 4 rows):
--   Alice→operations   Bob→operations   Bob→inventory   Carol→inventory
--
-- 🛑 Alice→inventory is ABSENT ON PURPOSE and she owns an inventory row.
-- ---------------------------------------------------------------------------
insert into app_permissions (id, app_id, role, user_id) values
  ('09000000-0000-4000-8000-000000000001', '0a000000-0000-4000-8000-000000000001', 'manager', null),
  ('09000000-0000-4000-8000-000000000002', '0a000000-0000-4000-8000-000000000002', 'admin',   null),
  ('09000000-0000-4000-8000-000000000003', '0a000000-0000-4000-8000-000000000001', null, '22222222-2222-4222-8222-222222222222'),
  ('09000000-0000-4000-8000-000000000004', '0a000000-0000-4000-8000-000000000002', null, '22222222-2222-4222-8222-222222222222')
on conflict (id) do nothing;

-- ---------------------------------------------------------------------------
-- Templates. The third is ARCHIVED and its submission must not migrate.
-- app_slug is the declared spike-only extension — see hq-source-schema.sql.
-- ---------------------------------------------------------------------------
insert into checklist_templates (id, name, requires_approval, created_by, created_at, archived_at, app_slug) values
  ('0b000000-0000-4000-8000-000000000001', 'Opening Checklist',     true,  '33333333-3333-4333-8333-333333333333', '2026-07-01 08:00:00+00', null,                     'operations'),
  ('0b000000-0000-4000-8000-000000000002', 'Weekly Stock Count',    false, '33333333-3333-4333-8333-333333333333', '2026-07-01 08:05:00+00', null,                     'inventory'),
  ('0b000000-0000-4000-8000-000000000003', 'Retired Prep Checklist',false, '33333333-3333-4333-8333-333333333333', '2026-05-01 08:00:00+00', '2026-06-15 12:00:00+00', 'operations')
on conflict (id) do nothing;

insert into checklist_sections (id, template_id, title, "order") values
  ('0c000000-0000-4000-8000-000000000001', '0b000000-0000-4000-8000-000000000001', 'Open',  1),
  ('0c000000-0000-4000-8000-000000000002', '0b000000-0000-4000-8000-000000000002', 'Count', 1),
  ('0c000000-0000-4000-8000-000000000003', '0b000000-0000-4000-8000-000000000003', 'Prep',  1)
on conflict (id) do nothing;

insert into checklist_fields (id, section_id, type, label, required, "order") values
  ('0d000000-0000-4000-8000-000000000001', '0c000000-0000-4000-8000-000000000001', 'checkbox',    'Fryer filters cleaned', true,  1),
  ('0d000000-0000-4000-8000-000000000002', '0c000000-0000-4000-8000-000000000001', 'temperature', 'Walk-in temp',          true,  2),
  ('0d000000-0000-4000-8000-000000000003', '0c000000-0000-4000-8000-000000000001', 'text',        'Notes',                 false, 3),
  ('0d000000-0000-4000-8000-000000000004', '0c000000-0000-4000-8000-000000000002', 'text',        'Protein count',         true,  1),
  ('0d000000-0000-4000-8000-000000000005', '0c000000-0000-4000-8000-000000000002', 'checkbox',    'Freezer sealed',        true,  2),
  ('0d000000-0000-4000-8000-000000000006', '0c000000-0000-4000-8000-000000000003', 'checkbox',    'Prep bins labelled',    true,  1)
on conflict (id) do nothing;

-- ---------------------------------------------------------------------------
-- Submissions. Seven; SIX must migrate (the seventh hangs off the archived
-- template). Timestamps are fixed so the migrated payload is byte-deterministic.
--
--   ...001  Alice   operations   completed   ← visible to Alice
--   ...002  Alice   operations   pending     ← visible to Alice
--   ...003  Bob     operations   completed   ← visible to Bob
--   ...004  Bob     operations   pending     ← visible to Bob
--   ...005  Alice   inventory    pending     ← MIGRATES, VISIBLE TO NOBODY.
--                                              Alice owns it but holds no live
--                                              inventory grant; Bob holds the
--                                              grant but does not own it. It is
--                                              the two-axis negative control.
--   ...006  Carol   inventory    completed   ← visible to Carol
--   ...007  Bob     operations   completed   ← ARCHIVED template: MUST NOT MIGRATE
-- ---------------------------------------------------------------------------
insert into checklist_submissions (id, template_id, template_snapshot, submitted_by, submitted_at, status) values
  ('0e000000-0000-4000-8000-000000000001', '0b000000-0000-4000-8000-000000000001',
   '{"name":"Opening Checklist","sections":[{"title":"Open","fields":3}]}'::jsonb,
   '11111111-1111-4111-8111-111111111111', '2026-07-20 06:12:00+00', 'completed'),
  ('0e000000-0000-4000-8000-000000000002', '0b000000-0000-4000-8000-000000000001',
   '{"name":"Opening Checklist","sections":[{"title":"Open","fields":3}]}'::jsonb,
   '11111111-1111-4111-8111-111111111111', '2026-07-21 06:09:00+00', 'pending'),
  ('0e000000-0000-4000-8000-000000000003', '0b000000-0000-4000-8000-000000000001',
   '{"name":"Opening Checklist","sections":[{"title":"Open","fields":3}]}'::jsonb,
   '22222222-2222-4222-8222-222222222222', '2026-07-20 06:31:00+00', 'completed'),
  ('0e000000-0000-4000-8000-000000000004', '0b000000-0000-4000-8000-000000000001',
   '{"name":"Opening Checklist","sections":[{"title":"Open","fields":3}]}'::jsonb,
   '22222222-2222-4222-8222-222222222222', '2026-07-21 06:28:00+00', 'pending'),
  ('0e000000-0000-4000-8000-000000000005', '0b000000-0000-4000-8000-000000000002',
   '{"name":"Weekly Stock Count","sections":[{"title":"Count","fields":2}]}'::jsonb,
   '11111111-1111-4111-8111-111111111111', '2026-07-22 15:02:00+00', 'pending'),
  ('0e000000-0000-4000-8000-000000000006', '0b000000-0000-4000-8000-000000000002',
   '{"name":"Weekly Stock Count","sections":[{"title":"Count","fields":2}]}'::jsonb,
   '33333333-3333-4333-8333-333333333333', '2026-07-22 15:40:00+00', 'completed'),
  ('0e000000-0000-4000-8000-000000000007', '0b000000-0000-4000-8000-000000000003',
   '{"name":"Retired Prep Checklist","sections":[{"title":"Prep","fields":1}]}'::jsonb,
   '22222222-2222-4222-8222-222222222222', '2026-06-01 05:55:00+00', 'completed')
on conflict (id) do nothing;

-- ---------------------------------------------------------------------------
-- Responses. Counts DELIBERATELY DIFFER per submission (3,2,3,1,2,2,1) so the
-- migrated payload's `responses` figure proves an aggregate join really ran
-- rather than a constant being emitted.
--
-- The last row is a DRAFT: submission_id IS NULL, HQ's in-progress shape. It is
-- attached to no submission and must be counted into none.
-- ---------------------------------------------------------------------------
insert into submission_responses (id, submission_id, field_id, value, answered_by, answered_at) values
  ('0f000000-0000-4000-8000-000000000001', '0e000000-0000-4000-8000-000000000001', '0d000000-0000-4000-8000-000000000001', 'true'::jsonb,          '11111111-1111-4111-8111-111111111111', '2026-07-20 06:12:10+00'),
  ('0f000000-0000-4000-8000-000000000002', '0e000000-0000-4000-8000-000000000001', '0d000000-0000-4000-8000-000000000002', '{"f":38.2}'::jsonb,    '11111111-1111-4111-8111-111111111111', '2026-07-20 06:12:20+00'),
  ('0f000000-0000-4000-8000-000000000003', '0e000000-0000-4000-8000-000000000001', '0d000000-0000-4000-8000-000000000003', '"all clear"'::jsonb,   '11111111-1111-4111-8111-111111111111', '2026-07-20 06:12:30+00'),
  ('0f000000-0000-4000-8000-000000000004', '0e000000-0000-4000-8000-000000000002', '0d000000-0000-4000-8000-000000000001', 'true'::jsonb,          '11111111-1111-4111-8111-111111111111', '2026-07-21 06:09:10+00'),
  ('0f000000-0000-4000-8000-000000000005', '0e000000-0000-4000-8000-000000000002', '0d000000-0000-4000-8000-000000000002', '{"f":37.4}'::jsonb,    '11111111-1111-4111-8111-111111111111', '2026-07-21 06:09:20+00'),
  ('0f000000-0000-4000-8000-000000000006', '0e000000-0000-4000-8000-000000000003', '0d000000-0000-4000-8000-000000000001', 'true'::jsonb,          '22222222-2222-4222-8222-222222222222', '2026-07-20 06:31:10+00'),
  ('0f000000-0000-4000-8000-000000000007', '0e000000-0000-4000-8000-000000000003', '0d000000-0000-4000-8000-000000000002', '{"f":39.0}'::jsonb,    '22222222-2222-4222-8222-222222222222', '2026-07-20 06:31:20+00'),
  ('0f000000-0000-4000-8000-000000000008', '0e000000-0000-4000-8000-000000000003', '0d000000-0000-4000-8000-000000000003', '"restocked oil"'::jsonb,'22222222-2222-4222-8222-222222222222','2026-07-20 06:31:30+00'),
  ('0f000000-0000-4000-8000-000000000009', '0e000000-0000-4000-8000-000000000004', '0d000000-0000-4000-8000-000000000001', 'false'::jsonb,         '22222222-2222-4222-8222-222222222222', '2026-07-21 06:28:10+00'),
  ('0f000000-0000-4000-8000-000000000010', '0e000000-0000-4000-8000-000000000005', '0d000000-0000-4000-8000-000000000004', '"12 lb"'::jsonb,       '11111111-1111-4111-8111-111111111111', '2026-07-22 15:02:10+00'),
  ('0f000000-0000-4000-8000-000000000011', '0e000000-0000-4000-8000-000000000005', '0d000000-0000-4000-8000-000000000005', 'true'::jsonb,          '11111111-1111-4111-8111-111111111111', '2026-07-22 15:02:20+00'),
  ('0f000000-0000-4000-8000-000000000012', '0e000000-0000-4000-8000-000000000006', '0d000000-0000-4000-8000-000000000004', '"31 lb"'::jsonb,       '33333333-3333-4333-8333-333333333333', '2026-07-22 15:40:10+00'),
  ('0f000000-0000-4000-8000-000000000013', '0e000000-0000-4000-8000-000000000006', '0d000000-0000-4000-8000-000000000005', 'true'::jsonb,          '33333333-3333-4333-8333-333333333333', '2026-07-22 15:40:20+00'),
  ('0f000000-0000-4000-8000-000000000014', '0e000000-0000-4000-8000-000000000007', '0d000000-0000-4000-8000-000000000006', 'true'::jsonb,          '22222222-2222-4222-8222-222222222222', '2026-06-01 05:55:10+00'),
  -- DRAFT — submission_id IS NULL. Belongs to no submission; must be counted
  -- into none. HQ's `submission_responses_draft_idx` is what makes this legal.
  ('0f000000-0000-4000-8000-000000000015', null,                                   '0d000000-0000-4000-8000-000000000003', '"half typed"'::jsonb,  '11111111-1111-4111-8111-111111111111', '2026-07-23 06:00:00+00')
on conflict (id) do nothing;

commit;
