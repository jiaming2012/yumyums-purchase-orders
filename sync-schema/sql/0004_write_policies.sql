-- sync-schema/sql/0004_write_policies.sql
--
-- The WRITE policies for HQ's four replicated sync tables — the WITH CHECK half
-- that makes RxDB **push** replication possible at all.
-- Card `sync-rxdb-write-policies` (overnight-20260802, A2).
--
-- ⚠ Substrate only. See 0002's header. Applied AFTER 0001, 0002 and 0003.
-- Idempotent: safe to re-run against a live stack.
--
-- ===========================================================================
-- 0. 🛑 THIS FILE IMPLEMENTS A SIGNED FOUR-ROW SPECIFICATION. IT IS NOT A
--    STARTING POINT.
-- ===========================================================================
--
-- Ledger T-30, decision 111 (2026-07-31 evening, `/nc-slate-plan`). The
-- operator was given three user stories — one uniform write predicate,
-- own-rows-only, or mirror-the-read-rule-per-table — and chose the third. The
-- resulting contract, verbatim from the ledger:
--
--   TABLE                    SELECT (0003, shipped)        INSERT/UPDATE (here)
--   ─────                    ──────────────────────        ────────────────────
--   checklist_templates      hq_can_see_template(id)       DENY-ALL, deliberately
--   checklist_submissions    hq_can_see_template(tid)      with check hq_can_see_template(template_id)
--   submission_responses     hq_can_see_field(field_id)    with check hq_can_see_field(field_id)
--   submission_rejections    none -> gains one HERE        approver-only
--
-- 🛑 FOUR ROWS. NOT FIVE. A write predicate beyond this table is an
-- OPERATOR-ONLY question and the card that owns this file is required to PARK
-- rather than invent one. If you are here because some table or column seems to
-- need a rule the table above does not give it, that is a decision to take to a
-- human, not a policy to add.
--
-- ===========================================================================
-- 1. 🛑 WHAT THIS FILE DELIBERATELY DOES NOT CONTAIN
-- ===========================================================================
--
-- (a) NO INSERT OR UPDATE POLICY FOR `checklist_templates`. ROW 1 OF THE TABLE
--     ABOVE IS AN ABSENCE, AND THE ABSENCE IS THE ANSWER.
--
--     The reason is a product one, not a security one: the template BUILDER
--     keeps the existing REST path, and NO PHONE WRITES A TEMPLATE DEFINITION.
--     A checklist template is authored by the owner in a desktop-shaped UI; it
--     is not something a crew member's offline queue can contain, so there is
--     no push to admit.
--
--     🛑 THIS WILL LOOK LIKE AN OVERSIGHT. Three of four tables below get a
--     write policy and this one does not, and "completing the set" is a
--     one-line change that no test in a normal suite would catch. So it is
--     asserted rather than left as a gap: variants W1, W2 and W3 in
--     backend/internal/sync/rowvisibility_rls_test.go push a template as an
--     assignee, rewrite another user's template, and — W3, the sharp one — do
--     both AS AN ADMIN. Deny-all here is UNCONDITIONAL. It is not "everyone
--     except admins", and a `with check (public.hq_can_see_template(id))`
--     written here by analogy with row 2 would let every admin through.
--
-- (b) NO DELETE POLICY, ANYWHERE. Decision 111's table has no DELETE column and
--     that is correct rather than incomplete.
--
--     0001's contract item 2: RxDB replication is SOFT-DELETE ONLY. A hard
--     DELETE is invisible to a pull handler — the row simply stops appearing,
--     and every offline replica keeps it forever. A tombstone is an UPDATE of
--     `_deleted`, which rows 2 and 3 below already govern. Two independent
--     gates refuse a hard delete: 0001 grants only `select, insert, update` to
--     `authenticated`, and this file writes no DELETE policy. Variant W12
--     asserts it across all three writable tables, because a future `grant all`
--     would silently remove the first gate and nothing else in the tree would
--     notice.
--
-- (c) NO CHANGE TO 0003. `hq_can_see_template` and `hq_can_see_field` are
--     byte-unchanged and this file does not redefine them. See §3.
--
-- (d) NO CHANGE TO `HQ_SYNC_REST_URL`. This file does not set it, reference it,
--     or imply it. That interlock disarms at triage on evidence, never by a
--     card asserting it.
--
-- ===========================================================================
-- 2. 🛑 THE ASYMMETRY — THE FIRST PLACE `assignment_role` MEANS ANYTHING
-- ===========================================================================
--
-- 0003's INHERITED PROPERTY 1, and migration 0073 §1, and 0002 §3a all record
-- the same thing in the strongest terms available to a comment:
--
--     `assignment_role` IS NEVER FILTERED ON. An approver sees exactly what an
--     assignee sees. …Do not add `WHERE ta.assignment_role = 'assignee'` here
--     as a "tightening": it would silently remove every approver's ability to
--     see the checklist they are supposed to approve.
--
-- THAT IS A READ PROPERTY AND IT IS UNCHANGED. Decision 111 consequence (2)
-- makes WRITES — and only writes — the place the column starts to matter:
--
--     a new `public.hq_can_approve_template(tid)` predicate = EXISTS an
--     assignment with assignment_role = 'approver' OR the unconditional
--     roles && ARRAY['admin','superadmin'] admin arm. Reads keep the old
--     property untouched; only the approval WITH CHECK uses the new one. That
--     asymmetry is the decision, not a side effect of it.
--
-- 🛑 THE TWO HALVES READ TWO DIFFERENT RELATIONS, SO NEITHER CAN DRIFT INTO
-- THE OTHER BY ACCIDENT:
--
--     READS   hq_can_see_template / hq_can_see_field  ->  hq_template_assignees
--                                                          (0002 §3a — carries
--                                                           NO assignment_role
--                                                           column at all)
--     WRITES  hq_can_approve_template / _field        ->  hq_template_approvers
--                                                          (0002 §3d, migration
--                                                           0074 — PRE-FILTERED
--                                                           to 'approver' on
--                                                           HQ's side)
--
-- 🛑 THE TWO WAYS TO BREAK THIS, AND THE TWO TESTS THAT CATCH THEM:
--
--     Make WRITES mirror READS  ("this file should be internally consistent")
--       -> every crew member can sign off their own checklist.
--       -> variant W9 turns red.
--
--     Make READS mirror WRITES  ("approvals and reads should agree")
--       -> every approver goes blind to the checklist they must approve.
--       -> variant WP5 and POSITIVE/alice turn red.
--
-- Keep both. The inconsistency is the product rule.
--
-- ===========================================================================
-- 3. Why this is a NEW FILE and not an edit to 0003
-- ===========================================================================
--
-- Because the red state must be REPRODUCIBLE, and 0003's own header explains
-- the pattern: 0002 (no policies) + a teardown is the read red; 0003 is the
-- read green. This card needs a THIRD state that neither of those can express —
-- reads working, writes deny-all — because that is the state the tree shipped
-- in, and it is the only state in which the write POSITIVES fail.
--
--     SYNC_RLS_SKIP_POLICIES=1        RLS torn down. Everything leaks.
--                                     Every REFUSAL fails; every POSITIVE
--                                     passes. Useless as this card's red.
--     SYNC_RLS_SKIP_WRITE_POLICIES=1  0003 applied, THIS FILE withheld.
--                                     Every write POSITIVE fails. This card's
--                                     real red.
--
-- 🛑 WHY THAT DISTINCTION IS NOT PEDANTRY, measured rather than argued: this
-- suite's header records that with the FDW deliberately repointed at a
-- migrated-but-empty database, TWELVE OF NINETEEN attack variants still passed
-- — and every one of the seven failures was on a POSITIVE half. A refusal is
-- blind to an empty subject set, and "writes are deny-all" is the largest empty
-- subject set this system can have. Keeping the two files apart is what keeps
-- the second red reachable with one environment variable.

-- ===========================================================================
-- 4. The approver predicates
-- ===========================================================================
--
-- SECURITY DEFINER, `stable`, pinned search_path, one scalar argument,
-- BOOLEAN-only answer about the CALLER'S OWN `sub` — identical in shape and for
-- identical reasons to 0003 §4. Read that section; every word of it applies
-- here and is not repeated.
--
-- 🛑 IT READS THE LIVE TABLE, NOT THE TOKEN, and on the write path that matters
-- MORE than on the read path, not less. An RxDB push is a device replaying a
-- queue that may be hours old, carrying a token minted before the shift
-- started. Reading entitlement live means a crew member revoked at 09:05 cannot
-- write with a token minted at 09:00 — variant W14, which is V8's shape aimed
-- at the write path, and which nothing in this repository asserted until now.

-- The approver arm. Structurally identical to hq_can_see_template with ONE
-- relation swapped: hq_template_approvers instead of hq_template_assignees. The
-- admin disjunct is BYTE-IDENTICAL, because decision 111 says the approver
-- predicate is "EXISTS an approver assignment OR the UNCONDITIONAL admin arm" —
-- INHERITED PROPERTY 2 travels onto the write path unchanged. Variant WP6 is
-- carol, who holds no assignment row of any kind, writing an approval.
create or replace function public.hq_can_approve_template(tid text)
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
             from public.hq_template_approvers a
            where a.template_id = tid
              and a.user_id = public.hq_jwt_claim('sub')
         );
$$;

-- The field-resolution step for approvals.
--
-- 🛑 THE NESTING ORDER IS THE SAME DECISION 0003's hq_can_see_field DOCUMENTS,
-- AND IT MUST BE MADE AGAIN HERE BECAUSE THIS IS A SEPARATE FUNCTION THAT CAN
-- BE GOT WRONG INDEPENDENTLY. Written the obvious way —
-- hq_can_approve_template(<lookup field_id>) — an unresolvable field_id yields
-- NULL, and hq_can_approve_template(NULL) is STILL TRUE for an admin, because
-- the admin arm does not mention the template at all. An admin would then be
-- able to write a rejection onto a field HQ has never heard of. Wrapping the
-- lookup in EXISTS reproduces the Go resolver's `[]string{}` on ErrNoRows: no
-- row, no authority, for anyone. Variant W8 asks carol exactly that, twice —
-- once against this function and once against hq_can_see_field — because one
-- of the two being right says nothing about the other.
--
-- 🛑 FIELD-SCOPED, NOT SUBMISSION-SCOPED, even though
-- `submission_rejections.submission_id` is `text not null` and the submission
-- route is structurally available (0003 §2(b) records that it is). Decision 111
-- gives this table the SELECT rule `hq_can_see_field(field_id)`, matching
-- submission_responses; the write is the same axis with the role arm tightened,
-- and nothing else. Choosing a different axis for the write than for the read
-- would mean a device could be authorised to write rows it cannot pull back —
-- which is the exact failure consequence (1) exists to prevent.
create or replace function public.hq_can_approve_field(fid text)
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
              and public.hq_can_approve_template(ft.template_id)
         );
$$;

grant execute on function public.hq_can_approve_template(text) to authenticated;
grant execute on function public.hq_can_approve_field(text)    to authenticated;

-- ===========================================================================
-- 5. The policies
-- ===========================================================================
-- RLS is already enabled on all four tables by 0001 and re-enabled by 0003;
-- doing it again is a no-op and is repeated so this file is correct applied on
-- its own.
--
-- 🛑 EVERY UPDATE POLICY BELOW CARRIES **BOTH** `using` AND `with check`, AND
-- THAT IS THE SINGLE MOST LOAD-BEARING SENTENCE IN THIS FILE.
--
--     `using`       decides which rows may be TARGETED (the OLD row).
--     `with check`  decides what they may BECOME     (the NEW row).
--
-- Postgres does not require the second and does not warn about its absence. An
-- UPDATE policy written with `using` alone passes every cross-user attack in
-- the suite — the attacker legitimately owns the row they are targeting — and
-- lets them MOVE it somewhere they do not. Alice takes her own submission and
-- re-parents it under a template no non-admin can see; alice takes her own
-- response and repoints its `field_id` at bob's field. Variants W5 and W7.
--
-- Neither is visible in the status code: PostgREST answers HTTP 200 `[]` for a
-- row `using` excluded and 403 for one `with check` rejected, and both attacks
-- are the second kind. Only the table says which happened, which is why every
-- write assertion in the suite verifies through the pool and not the response.

-- ---------------------------------------------------------------------------
-- 5a. checklist_templates — NO WRITE POLICY. See §1(a).
-- ---------------------------------------------------------------------------
-- RLS stays enabled with SELECT-only coverage: deny-all for INSERT and UPDATE,
-- for every role including admins. Asserted by W1, W2 and W3 rather than left
-- as an absence somebody later reads as an oversight.
alter table public.checklist_templates enable row level security;

-- ---------------------------------------------------------------------------
-- 5b. checklist_submissions — ROW 2. Mirrors the SELECT rule exactly.
-- ---------------------------------------------------------------------------
-- 🛑 THIS CLOSES THE LIE 0003:243 NAMES BY HAND:
--
--     "a pushed row claiming a template_id its author can see would indeed be
--      visible to its author. It cannot happen — §2(a), writes are deny-all —
--      and when the write card lands, its WITH CHECK is where that must be
--      handled."
--
-- This is that WITH CHECK. A row claiming a `template_id` its author CANNOT see
-- is refused at insert (W4) and cannot be moved there by update (W5).
--
-- 🛑 WHAT IT DELIBERATELY DOES NOT CHECK: `submitted_by`. The predicate is the
-- template and nothing else, so ALICE MAY PUSH A ROW ATTRIBUTED TO BOB provided
-- it sits inside a template she can see. That is decision 111's answer, not a
-- gap: the operator was offered own-rows-only and did not take it, because an
-- RxDB push is a device replaying a local queue and that queue can legitimately
-- carry a row another crew member created on a shared truck phone before this
-- device came back online. Variant WP1b asserts it as a property, so that the
-- day someone decides otherwise they are changing a stated rule rather than
-- reporting a vulnerability.
alter table public.checklist_submissions enable row level security;

drop policy if exists checklist_submissions_insert on public.checklist_submissions;
create policy checklist_submissions_insert on public.checklist_submissions
  for insert to authenticated
  with check ( public.hq_can_see_template(template_id) );

drop policy if exists checklist_submissions_update on public.checklist_submissions;
create policy checklist_submissions_update on public.checklist_submissions
  for update to authenticated
  using      ( public.hq_can_see_template(template_id) )   -- the OLD row
  with check ( public.hq_can_see_template(template_id) );  -- the NEW row (W5)

-- ---------------------------------------------------------------------------
-- 5c. submission_responses — ROW 3. FIELD-scoped, and that is the whole
--     offline story.
-- ---------------------------------------------------------------------------
-- 🛑 `field_id` AND NOT `submission_id`, for exactly the reason 0003 §5c gives
-- for the read: `submission_responses.submission_id` is NULLABLE on purpose,
-- because a DRAFT response has no submission yet, and drafts are what a crew
-- member fills offline on the truck.
--
-- The consequence is sharper on the write side than on the read side. A
-- submission-scoped read predicate returns an empty set for drafts — visibly
-- wrong to anyone who looks. A submission-scoped WRITE predicate REFUSES EVERY
-- OFFLINE DRAFT PUSH, which is the one thing this collection exists to carry —
-- and it passes every attack variant in the suite while doing so, because
-- refusing everything refuses attackers too. Variant WP3 pushes a row with
-- `submission_id: null` and is the line that fails against it.
alter table public.submission_responses enable row level security;

drop policy if exists submission_responses_insert on public.submission_responses;
create policy submission_responses_insert on public.submission_responses
  for insert to authenticated
  with check ( public.hq_can_see_field(field_id) );

drop policy if exists submission_responses_update on public.submission_responses;
create policy submission_responses_update on public.submission_responses
  for update to authenticated
  using      ( public.hq_can_see_field(field_id) )   -- the OLD row
  with check ( public.hq_can_see_field(field_id) );  -- the NEW row (W7)

-- ---------------------------------------------------------------------------
-- 5d. submission_rejections — ROW 4. READABLE at last, and APPROVER-ONLY to
--     write.
-- ---------------------------------------------------------------------------
-- 🛑 TWO CHANGES HERE, AND THE FIRST ONE REVERSES A SHIPPED DECISION ON PURPOSE.
--
-- (1) THE SELECT POLICY. 0003 §5d left this table RLS-enabled with ZERO
--     policies — deny-all both ways — and that was right for card B2:
--     `ResolveEntityAccess` has no case for this entity type, so any policy
--     would have been an EXTENSION rather than a port, and B2 was forbidden
--     from inventing permission semantics. Variant V18 asserted the deny-all so
--     it stayed a decision with evidence.
--
--     Decision 111 consequence (1) makes the extension, with the authority B2
--     did not have, and gives a reason that is mechanical rather than a
--     preference: A DEVICE THAT CAN WRITE A ROW IT CANNOT READ BACK BREAKS
--     REPLICATION. RxDB's push is followed by a pull; a write-only table means
--     the pull never returns what the push wrote, the client believes its write
--     was lost, and it re-pushes forever.
--
--     The product half is the one crew members depend on: THE ASSIGNEE WHOSE
--     FIELD WAS REJECTED MUST BE ABLE TO READ THEIR OWN FEEDBACK. That is the
--     reject-with-comment path. So the rule is `hq_can_see_field(field_id)`,
--     matching submission_responses exactly.
--
--     🛑 V18 IN THE GO SUITE NOW ASSERTS THE OPPOSITE OF WHAT IT USED TO. It
--     was rewritten in place, not deleted, and says so. A merge that restores
--     the old V18 restores a suite that contradicts this file.
--
-- (2) THE WRITE IS APPROVER-ONLY — §2's asymmetry, and the only consumer of
--     hq_can_approve_field in the tree.
--
--     Read broad, write narrow, ON THE SAME AXIS: an assignee RECEIVES the
--     rejection written about their work and cannot WRITE one. Alice is an
--     assignee on tplAlice; she pulls `rej-alice` on every sync (V18, WP7) and
--     is refused when she tries to create one on the same field (W9) or soften
--     the one already there (W11). The UPDATE's `using` clause is deliberately
--     the APPROVE predicate and not the SEE predicate: copying the SELECT
--     policy's `using` here — the natural thing to write, and the thing that
--     makes the two clauses of this table agree — is exactly what lets W11
--     through.
alter table public.submission_rejections enable row level security;

drop policy if exists submission_rejections_select on public.submission_rejections;
create policy submission_rejections_select on public.submission_rejections
  for select to authenticated
  using ( public.hq_can_see_field(field_id) );

drop policy if exists submission_rejections_insert on public.submission_rejections;
create policy submission_rejections_insert on public.submission_rejections
  for insert to authenticated
  with check ( public.hq_can_approve_field(field_id) );

drop policy if exists submission_rejections_update on public.submission_rejections;
create policy submission_rejections_update on public.submission_rejections
  for update to authenticated
  -- 🛑 approve, NOT see. See (2) above; W11 is the variant.
  using      ( public.hq_can_approve_field(field_id) )
  with check ( public.hq_can_approve_field(field_id) );

-- PostgREST caches the schema; policy changes need the reload signal.
notify pgrst, 'reload schema';
