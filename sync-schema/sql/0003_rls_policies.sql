-- sync-schema/sql/0003_rls_policies.sql
--
-- The row-visibility POLICIES for HQ's four replicated sync tables.
-- Card `sync-rxdb-row-visibility-rls` (overnight-20260801, B2).
--
-- ⚠ Substrate only. See 0002's header. Applied AFTER 0001 and 0002.
-- Idempotent: safe to re-run against a live stack.
--
-- ===========================================================================
-- 0. 🛑 READ THIS BEFORE COPYING ANY POLICY FROM SUPABASE'S DOCS
-- ===========================================================================
--
-- `auth.uid()` IS WRONG FOR THIS STACK. This is not a new finding and it must
-- not be rediscovered a third time — card `sync-jwt-bridge-endpoint` banked it
-- on 2026-07-26 and built a permanent negative control so it re-proves itself
-- on every run.
--
-- Without GoTrue's migrations the `auth` schema ships three functions, and
-- `uid` reads `request.jwt.claim.sub` — the LEGACY SINGULAR GUC, which
-- PostgREST populates only when PGRST_DB_USE_LEGACY_GUCS=true. This stack sets
-- it "false" (docker-compose.supabase.yml). There is no plural fallback inside
-- auth.uid(). It returns NULL, the predicate is NULL, and the policy selects
-- nothing.
--
-- 🛑 THE FAILURE IS SILENT. It does not raise, it does not point at itself. It
-- returns `HTTP 200 []`, which reads as "this user has no checklists" — the
-- single most plausible-looking wrong answer this system can give.
--
-- ✅ THE CORRECT FORM, used by everything below:
--
--     current_setting('request.jwt.claims', true)::json ->> '<claim>'
--
-- wrapped once, in public.hq_jwt_claim.
--
-- The standing proof lives in .night-crew/qa/spike-supabase/sql/
-- hq-bridge-policies.sql section 4: `public.hq_uid_trap`, a table governed by a
-- deliberately-wrong auth.uid() policy, so a suite can show — same token, same
-- instant, same stack — that the plural-GUC policy DISCRIMINATES while the
-- auth.uid() policy RETURNS NOTHING. This card does not duplicate that trap; it
-- re-runs it (variant V19) so the finding stays live rather than remembered.
--
-- ===========================================================================
-- 1. WHAT THIS FILE IS A PORT OF, AND WHAT IT DELIBERATELY DOES NOT CHANGE
-- ===========================================================================
--
-- Source of truth: `ResolveEntityAccess`, backend/internal/sync/ops.go. Its
-- final query, verbatim:
--
--     SELECT DISTINCT u.id::text
--       FROM users u
--      WHERE u.roles && ARRAY['admin','superadmin']
--         OR EXISTS (
--              SELECT 1 FROM template_assignments ta
--               WHERE ta.template_id = $1::uuid
--                 AND ( (ta.assignee_type = 'user' AND u.id::text = ta.assignee_id)
--                    OR (ta.assignee_type = 'role' AND ta.assignee_id = ANY(u.roles)) ))
--
-- preceded by a per-entity-type resolution step:
--     template        -> itself
--     submission      -> SELECT template_id FROM checklist_submissions WHERE id=$1
--     field_response  -> SELECT s.template_id FROM checklist_fields f
--                          JOIN checklist_sections s ON f.section_id = s.id
--                         WHERE f.id = $1
--     anything else   -> [] (nobody)
--
-- The Go asks "given a template, which users?"; a policy asks "given a user,
-- which rows?". Same relation, read along the other axis. That is why this is a
-- PORT and not a new predicate, and it is why the two properties below travel
-- with it whether or not anyone likes them.
--
-- 🛑 INHERITED PROPERTY 1 — `assignment_role` IS NEVER FILTERED ON.
--    `template_assignments.assignment_role` is 'assignee' or 'approver'. The
--    resolver reads neither. AN APPROVER SEES EXACTLY WHAT AN ASSIGNEE SEES.
--    Preserved here on purpose. The column is not even carried across the FDW
--    (0002 §3a) so it cannot be filtered on by accident.
--
-- 🛑 INHERITED PROPERTY 2 — THE ADMIN ARM IS UNCONDITIONAL.
--    `roles && ARRAY['admin','superadmin']` is a free-standing disjunct gated on
--    nothing. EVERY ADMIN SEES EVERY TEMPLATE. The resolver's own comment gives
--    the reason: myChecklists already grants admins view-all, and live sync must
--    mirror the access the UI gives or an admin's own edit never converges on
--    their own second device.
--
-- BOTH ARE KNOWING, NOT ACCIDENTAL. Tightening either is a SEPARATE CARD.
-- Do not vary substrate and permission semantics in one night — a change to
-- either one, made here, would be indistinguishable at review from the port.
--
-- ===========================================================================
-- 2. 🛑 WHAT THIS FILE DELIBERATELY DOES NOT CONTAIN
-- ===========================================================================
--
-- (a) NO INSERT OR UPDATE POLICIES. The four tables keep `grant select, insert,
--     update to authenticated` from 0001 and no write policy, which in Postgres
--     is DENY-ALL for writes. That is not an oversight, and the suite proves it
--     rather than assuming it (V10, V11).
--
--     `ResolveEntityAccess` is a FAN-OUT resolver. It answers "who should
--     RECEIVE this op" — a read-visibility question. Reusing its predicate to
--     decide who may WRITE would be inventing a permission semantic ("who may
--     create a template?", "who may edit someone else's answer?") that no
--     shipped code asserts. The card forbids exactly that.
--
--     🛑 CONSEQUENCE THE RxDB CARDS NEED TO HEAR: PUSH REPLICATION WILL BE
--     REFUSED until a follow-up card writes WITH CHECK policies. That card owns
--     a product question this one has no authority over. Nothing is live today
--     (HQ_SYNC_REST_URL is unset), so this blocks nothing that ships.
--
-- (b) NO POLICY FOR `submission_rejections`. It stays deny-all, exactly as 0001
--     left it. `ResolveEntityAccess`'s switch has NO CASE for it and falls
--     through to `return []string{}` — the current WebSocket layer does not fan
--     rejections out at all. Giving it a policy would therefore be an EXTENSION,
--     not a port.
--
--     The minimal consistent rule ("a rejection is visible iff its submission
--     is") is structurally available — `submission_rejections.submission_id` is
--     `text not null`, so unlike responses there is no null case. STRUCTURAL
--     AVAILABILITY IS NOT AUTHORITY. Nothing in shipped code asserts that rule,
--     so writing it is still inventing a permission semantic. Recorded so the
--     next card knows the shape is there and does not re-derive it.
--
-- (c) NO CHANGE TO `HQ_SYNC_REST_URL`. This file does not set it, reference it,
--     or imply it. That interlock disarms at triage on evidence, never by a
--     card asserting it.

-- ===========================================================================
-- 3. Claim accessor
-- ===========================================================================
-- Identical to hq-bridge-policies.sql's. Re-declared with CREATE OR REPLACE so
-- this file stands alone (0001+0002+0003 is a complete substrate), rather than
-- depending on a spike fixture that is not part of the shipped schema.
--
-- `true` (missing_ok) is load-bearing: an unauthenticated request has no such
-- setting at all, and without it this would ERROR instead of returning NULL —
-- turning a clean refusal into a 500.
create or replace function public.hq_jwt_claim(claim text)
returns text
language sql
stable
set search_path = public, pg_temp
as $$
  select current_setting('request.jwt.claims', true)::json ->> claim;
$$;

-- ===========================================================================
-- 4. The ported predicate
-- ===========================================================================
--
-- SECURITY DEFINER is REQUIRED and is not a shortcut. The foreign tables are
-- revoked from `authenticated` entirely (0002 §4) precisely so a GET cannot
-- read HQ's whole role map — so a policy that had to read them as the calling
-- user could not work at all. Running as the owner is also what resolves the
-- postgres_fdw user mapping, which exists only for the owner.
--
-- The function is owned by the superuser applying this file, runs with a pinned
-- search_path, takes one scalar argument, and can only ever answer a BOOLEAN
-- about the CALLER'S OWN `sub`. It cannot be coaxed into answering about
-- anybody else, and it cannot be used to read a row out of HQ.
--
-- 🛑 IT READS THE LIVE TABLE, NOT THE TOKEN. The JWT carries HQ role and grant
-- claims; this predicate ignores them, for the reason hq-bridge-policies.sql
-- documents at length: a JWT's claims are frozen at mint, so a token minted at
-- 09:00 still asserts the assignments held at 09:00 after they are revoked at
-- 09:05. Entitlement is read live, per row, through the FDW. Revoke the
-- assignment in HQ and the SAME UNEXPIRED TOKEN stops seeing the rows on the
-- next request. That is variant V9, and it is the whole reason decision 92
-- chose read-through over any asynchronous projection.
create or replace function public.hq_can_see_template(tid text)
returns boolean
language sql
stable
security definer
set search_path = public, pg_temp
as $$
  select exists (
           select 1
             from public.hq_user_roles r
            where r.user_id = public.hq_jwt_claim('sub')
              and r.roles && array['admin','superadmin']
         )
      or exists (
           select 1
             from public.hq_template_assignees a
            where a.template_id = tid
              and a.user_id = public.hq_jwt_claim('sub')
         );
$$;

-- The `field_response` resolution step, kept as its own function so the
-- resolver's EARLY RETURN is preserved exactly.
--
-- 🛑 THE NESTING ORDER MATTERS AND IS NOT INTERCHANGEABLE. Written the obvious
-- way — hq_can_see_template(<lookup field_id>) — an unresolvable field_id
-- yields NULL, and `hq_can_see_template(NULL)` is still TRUE for an admin
-- (the admin arm does not mention the template at all). The Go does the
-- opposite: on pgx.ErrNoRows it returns `[]string{}` — NOBODY, admins included.
-- Wrapping the lookup in EXISTS reproduces that: no row, no visibility, for
-- anyone. An orphan response is invisible rather than admin-visible.
create or replace function public.hq_can_see_field(fid text)
returns boolean
language sql
stable
security definer
set search_path = public, pg_temp
as $$
  select exists (
           select 1
             from public.hq_field_templates ft
            where ft.field_id = fid
              and public.hq_can_see_template(ft.template_id)
         );
$$;

grant execute on function public.hq_jwt_claim(text)         to authenticated;
grant execute on function public.hq_can_see_template(text)  to authenticated;
grant execute on function public.hq_can_see_field(text)     to authenticated;

-- ===========================================================================
-- 5. The policies
-- ===========================================================================
-- RLS is already enabled on all four tables by 0001; enabling again is a no-op
-- and is repeated here so this file is correct applied on its own.

-- ---------------------------------------------------------------------------
-- 5a. checklist_templates — the template IS the entity
-- ---------------------------------------------------------------------------
-- The resolver's `case "template": templateID = entityID`.
alter table public.checklist_templates enable row level security;

drop policy if exists checklist_templates_select on public.checklist_templates;
create policy checklist_templates_select on public.checklist_templates
  for select to authenticated
  using ( public.hq_can_see_template(id) );

-- ---------------------------------------------------------------------------
-- 5b. checklist_submissions — scoped by the submission's own template
-- ---------------------------------------------------------------------------
-- The resolver looks `template_id` up from `checklist_submissions` by id; here
-- the row being tested already carries it, so the lookup collapses into a
-- column read. Same value, one fewer hop.
--
-- Worth stating because it looks like a place a client could lie: a pushed row
-- claiming a template_id its author can see would indeed be visible to its
-- author. It cannot happen — §2(a), writes are deny-all — and when the write
-- card lands, its WITH CHECK is where that must be handled.
alter table public.checklist_submissions enable row level security;

drop policy if exists checklist_submissions_select on public.checklist_submissions;
create policy checklist_submissions_select on public.checklist_submissions
  for select to authenticated
  using ( public.hq_can_see_template(template_id) );

-- ---------------------------------------------------------------------------
-- 5c. submission_responses — scoped by FIELD, not by submission
-- ---------------------------------------------------------------------------
-- 🛑 `field_id` AND NOT `submission_id`, and the difference is the whole
-- offline story. `submission_responses.submission_id` is NULLABLE on purpose —
-- a DRAFT response has no submission yet (0001, contract note; HQ migration
-- 0012's partial unique index). Drafts are exactly what a crew member fills
-- offline on the truck, so scoping by submission would leave the one collection
-- that MUST sync unresolvable, and would do it by returning nothing rather than
-- by erroring.
--
-- `field_id` is `not null` and resolves through the FDW for drafts and
-- submitted responses alike. This is also the resolver's own choice, and its
-- comment says why: "This works for both draft responses (no submission) and
-- submitted ones."
alter table public.submission_responses enable row level security;

drop policy if exists submission_responses_select on public.submission_responses;
create policy submission_responses_select on public.submission_responses
  for select to authenticated
  using ( public.hq_can_see_field(field_id) );

-- ---------------------------------------------------------------------------
-- 5d. submission_rejections — NO POLICY. See §2(b).
-- ---------------------------------------------------------------------------
-- RLS stays enabled with zero policies: deny-all, exactly as 0001 left it. The
-- resolver has no case for this entity type, so a policy here would be an
-- extension rather than a port. The suite asserts the deny-all (variant V17) so
-- that this stays a decision with evidence rather than a gap nobody noticed.
alter table public.submission_rejections enable row level security;

-- PostgREST caches the schema; policy changes need the reload signal.
notify pgrst, 'reload schema';
