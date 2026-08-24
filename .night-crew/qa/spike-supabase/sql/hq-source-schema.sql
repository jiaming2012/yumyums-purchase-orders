-- hq-source-schema.sql — the HQ-SHAPED source schema for Spike B.
--
-- Card S `spike-b-migration-rehearsal`. Applied to the throwaway `spike-b-hq`
-- Postgres (docker-compose.hq-source.yml), NEVER to HQ's real database, NEVER
-- to :5433, NEVER to :5434.
--
-- ═══════════════════════════════════════════════════════════════════════════
-- WHAT THIS IS, PRECISELY
--
-- A SMALL SUBSET of HQ's real schema, transcribed from the real migrations so a
-- reader can diff it against them:
--
--   users                   backend/internal/db/migrations/0001_users.sql
--   hq_apps                                               0004_hq_apps.sql
--   app_permissions                                       0005_app_permissions.sql
--   checklist_templates                                   0006_checklist_templates.sql
--   checklist_sections                                    0009_checklist_sections.sql
--   checklist_fields                                      0010_checklist_fields.sql
--   checklist_submissions                                 0011_checklist_submissions.sql
--   submission_responses                                  0012_submission_responses.sql
--
-- Columns HQ has that this subset drops are listed per table below. Everything
-- present is present with HQ's own type, HQ's own nullability and HQ's own CHECK
-- constraint text — the point of a "schema that mimics HQ's" is that a UUID
-- primary key really is a UUID, a JSONB payload really is JSONB, and the role
-- vocabulary really is HQ's three-value one, because those are the things a
-- migration into a text-primary-keyed sync table has to actually cope with.
--
-- ═══════════════════════════════════════════════════════════════════════════
-- 🛑 ONE DECLARED DEVIATION FROM HQ, AND IT IS ITSELF A FINDING
--
--   checklist_templates.app_slug DOES NOT EXIST IN HQ TODAY.
--
-- HQ stores no association between a checklist template and an `hq_apps` row.
-- The workflow tool is simply "the operations app" by convention, in the
-- frontend, nowhere in the schema. The sync bridge NEEDS that association:
-- hq-bridge-policies.sql's entitlement axis is `hq_has_grant(app_slug)`, and
-- there is no column in HQ to populate `app_slug` from.
--
-- Rather than paper over it with a hardcoded 'operations' — which would make the
-- entitlement axis untestable over migrated data, because every migrated row
-- would carry the same slug — this file adds the column EXPLICITLY and labels it
-- an extension. It is not a claim about HQ's current schema and it is not a
-- decision about HQ's future one; it is the spike surfacing that the cutover
-- card inherits an open question: WHERE does a replicated row's app_slug come
-- from? Options a cutover card would have to choose between: a column on
-- checklist_templates, a template->app join table, or a static per-tool mapping
-- in the projector. This spike does not pick one.
--
-- ═══════════════════════════════════════════════════════════════════════════
--
-- Idempotent: safe to re-run. (The script drops the whole container anyway; the
-- idempotency is for hand-debugging a live scratch container.)

begin;

-- ---------------------------------------------------------------------------
-- users — 0001_users.sql
-- Dropped from the subset: password_hash, invited_at, accepted_at.
-- Kept: the role CHECK, verbatim. The projection below expands ROLE-based
-- grants, so the role vocabulary is load-bearing, not decoration.
-- ---------------------------------------------------------------------------
create table if not exists users (
  id            uuid primary key,
  email         text unique not null,
  display_name  text not null,
  role          text not null check (role in ('admin', 'manager', 'team_member')),
  status        text not null default 'invited' check (status in ('invited', 'active'))
);

-- ---------------------------------------------------------------------------
-- hq_apps — 0004_hq_apps.sql. Complete; nothing dropped.
-- ---------------------------------------------------------------------------
create table if not exists hq_apps (
  id      uuid primary key,
  slug    text unique not null,
  name    text not null,
  icon    text not null,
  enabled boolean not null default true
);

-- ---------------------------------------------------------------------------
-- app_permissions — 0005_app_permissions.sql. Complete, including the
-- role_or_user CHECK and both partial unique indexes.
--
-- 🛑 THE DUAL SHAPE IS THE WHOLE REASON THIS TABLE IS IN THE SUBSET. A row
-- grants an app EITHER to a role tier (role set, user_id null) OR to one user
-- (user_id set, role null). hq_grant_projection is a flat (user_id, app_slug)
-- table, so the migration has to RESOLVE the role-based half by joining through
-- users.role. A migration that only copied the user-based rows would look
-- correct against a user-only fixture and silently drop most real grants.
-- ---------------------------------------------------------------------------
create table if not exists app_permissions (
  id      uuid primary key,
  app_id  uuid not null references hq_apps(id) on delete cascade,
  role    text check (role in ('admin', 'manager', 'team_member')),
  user_id uuid references users(id) on delete cascade,
  constraint role_or_user check (
    (role is not null and user_id is null) or
    (role is null and user_id is not null)
  )
);
create unique index if not exists app_permissions_role_idx
  on app_permissions(app_id, role) where role is not null;
create unique index if not exists app_permissions_user_idx
  on app_permissions(app_id, user_id) where user_id is not null;

-- ---------------------------------------------------------------------------
-- checklist_templates — 0006_checklist_templates.sql
-- Dropped from the subset: updated_at.
-- ADDED (declared extension, see the banner): app_slug.
-- Kept: archived_at, because the migration must not carry archived templates'
-- submissions and a subset without it could not show that.
-- ---------------------------------------------------------------------------
create table if not exists checklist_templates (
  id                uuid primary key,
  name              text not null,
  requires_approval boolean not null default false,
  created_by        uuid references users(id),
  created_at        timestamptz not null default now(),
  archived_at       timestamptz,
  -- 🛑 SPIKE-ONLY EXTENSION. Not in HQ. See the banner above.
  app_slug          text not null references hq_apps(slug)
);

-- ---------------------------------------------------------------------------
-- checklist_sections — 0009_checklist_sections.sql
-- Dropped from the subset: condition.
-- ---------------------------------------------------------------------------
create table if not exists checklist_sections (
  id          uuid primary key,
  template_id uuid not null references checklist_templates(id) on delete cascade,
  title       text not null,
  "order"     integer not null
);

-- ---------------------------------------------------------------------------
-- checklist_fields — 0010_checklist_fields.sql
-- Dropped from the subset: parent_field_id, config, fail_trigger, condition.
-- Kept: the type CHECK, verbatim.
-- ---------------------------------------------------------------------------
create table if not exists checklist_fields (
  id         uuid primary key,
  section_id uuid not null references checklist_sections(id) on delete cascade,
  type       text not null check (type in ('checkbox','yes_no','text','temperature','photo')),
  label      text not null,
  required   boolean not null default false,
  "order"    integer not null
);

-- ---------------------------------------------------------------------------
-- checklist_submissions — 0011_checklist_submissions.sql
-- Dropped from the subset: reviewed_by, reviewed_at, idempotency_key.
-- Kept: template_snapshot JSONB and the status CHECK, verbatim. The JSONB
-- column is deliberately kept — a migration that has never moved a JSONB
-- payload has not been tested on HQ's actual row shape.
-- ---------------------------------------------------------------------------
create table if not exists checklist_submissions (
  id                uuid primary key,
  template_id       uuid not null references checklist_templates(id),
  template_snapshot jsonb not null,
  submitted_by      uuid not null references users(id),
  submitted_at      timestamptz not null default now(),
  status            text not null default 'pending'
                    check (status in ('pending','approved','rejected','completed'))
);

-- ---------------------------------------------------------------------------
-- submission_responses — 0012_submission_responses.sql
-- Complete, including the nullable submission_id (HQ's draft-response shape)
-- and the partial unique draft index.
--
-- Drafts matter to the rehearsal: a response with submission_id IS NULL is a
-- crew member's in-progress answer, and it must NOT be counted into a
-- submission's migrated payload. The fixture seeds one so the projection is
-- forced to be explicit about it.
-- ---------------------------------------------------------------------------
create table if not exists submission_responses (
  id            uuid primary key,
  submission_id uuid references checklist_submissions(id) on delete cascade,
  field_id      uuid not null references checklist_fields(id),
  value         jsonb not null,
  answered_by   uuid not null references users(id),
  answered_at   timestamptz not null default now(),
  unique (submission_id, field_id)
);
create unique index if not exists submission_responses_draft_idx
  on submission_responses(field_id, answered_by) where submission_id is null;

commit;
