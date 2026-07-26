-- hq-bridge-policies.sql — the RLS half of card `sync-jwt-bridge-endpoint`.
--
-- Applied AFTER hq-bridge-fixture.sql. The split is not cosmetic: the card's
-- gate is red-first, and every variant in backend/internal/sync/
-- jwtbridge_rls_test.go was captured REFUSING against the fixture alone, with
-- this file absent, before a line of it was written. The capture is at
-- .night-crew/qa/spike-supabase/captures/red-20260726-attack-variants.txt.
--
-- Idempotent: safe to re-run against a live stack.
--
-- ⚠ LOCAL throwaway spike stack only. Never a hosted Supabase project, never
--   production, never real HQ data. Does not touch W1's spike_notes.

-- ═══════════════════════════════════════════════════════════════════════════
-- 0. 🛑 READ THIS BEFORE COPYING ANY POLICY FROM SUPABASE'S DOCS
-- ═══════════════════════════════════════════════════════════════════════════
--
-- `auth.uid()` IS WRONG FOR THIS STACK, and it fails NON-OBVIOUSLY — it does
-- not error in a way that points at itself, it quietly returns nothing, which
-- reads as "this user has no data."
--
-- Without GoTrue's migrations the `auth` schema ships only three functions
-- (email, role, uid), and `uid` is:
--
--     select nullif(current_setting('request.jwt.claim.sub', true), '')::uuid;
--
-- Two problems — and they are NOT both live here. Which one actually bites is
-- worth being precise about, because the observable symptom differs:
--
--   1. ✅ LIVE ON THIS STACK. `request.jwt.claim.sub` is the LEGACY SINGULAR
--      GUC. PostgREST populates it only when PGRST_DB_USE_LEGACY_GUCS=true.
--      This stack sets it "false" (docker-compose.supabase.yml). There is NO
--      plural fallback inside auth.uid(). It returns NULL, the predicate is
--      NULL, and the policy selects nothing. This is the failure variant V13
--      observes, and it observes it as `HTTP 200 []` — silence, not an error.
--
--   2. ⚠ LATENT, NOT REACHABLE HERE. auth.uid() also casts to `uuid`, so a
--      non-UUID `sub` would raise `invalid input syntax for type uuid`. On
--      this stack that raise is UNREACHABLE: problem 1 feeds nullif() a NULL,
--      and `NULL::uuid` is perfectly legal, so the cast never sees a bad
--      string. It would become live only if someone set
--      PGRST_DB_USE_LEGACY_GUCS=true — at which point the failure mode flips
--      from a silent empty result to a loud 500. Recorded because that flip
--      is confusing if you have not been told to expect it, NOT because it
--      happens today.
--
-- The practical consequence is that on THIS stack auth.uid() fails SILENTLY,
-- which is the worse of the two: a raise would at least point at itself.
--
-- `auth.jwt()` does not exist here at all.
--
-- ✅ THE CORRECT FORM, used by every policy below:
--
--     current_setting('request.jwt.claims', true)::json ->> '<claim>'
--
-- Section 4 builds a deliberate auth.uid() trap so this finding is
-- RE-VERIFIED BY THE SUITE on every run instead of decaying into a comment
-- nobody checks. If that trap ever starts admitting rows, the stack changed
-- and this whole file needs re-reading — the suite says so in its failure text.

-- ═══════════════════════════════════════════════════════════════════════════
-- 1. Claim accessors
-- ═══════════════════════════════════════════════════════════════════════════

-- hq_jwt_claim is the single place the plural GUC is read. Everything else goes
-- through it, so there is exactly one line in this system that could ever be
-- "fixed" back to the legacy singular form, and it is the one under the banner
-- above.
--
-- `true` (missing_ok) is load-bearing: an unauthenticated request has no such
-- setting at all, and without it the function would ERROR instead of returning
-- NULL — turning a clean refusal into a 500.
create or replace function public.hq_jwt_claim(claim text)
returns text
language sql
stable
set search_path = public, pg_temp
as $$
  select current_setting('request.jwt.claims', true)::json ->> claim;
$$;

-- ═══════════════════════════════════════════════════════════════════════════
-- 2. 🛑 The live grant check — why a claim is NOT the gate
-- ═══════════════════════════════════════════════════════════════════════════
--
-- The token carries an `hq_grants` claim. THE POLICIES DELIBERATELY IGNORE IT.
--
-- A JWT's claims are frozen at mint. A token minted at 09:00 still asserts the
-- grants held at 09:00 after an admin revokes them at 09:05. If RLS trusted the
-- claim, revocation would not take effect until the token expired — a replay
-- window as long as the TTL, on the one operation an admin performs precisely
-- because they want it to take effect NOW.
--
-- So entitlement is read live, from public.hq_grant_projection, on every row.
-- Revoke the grant, delete the projection row, and the SAME unexpired token
-- stops seeing the rows immediately. That is variant V9.
--
-- The claim survives only so the client can render its UI without a second
-- round trip. Variant V8 mints a token whose claim LIES — asserting a grant the
-- projection does not carry — and proves it opens nothing.
--
-- SECURITY DEFINER is required and is not a shortcut: section 5 puts RLS on
-- hq_grant_projection itself (a user may read their own grants, not everyone
-- else's), and a policy that had to read that table as the calling user would
-- either recurse or need the table left open. The function is owned by the
-- superuser applying this file, runs with a pinned search_path, takes one
-- scalar argument, and can only ever answer a boolean about the CALLER'S OWN
-- `sub` — it cannot be coaxed into answering about anybody else.
create or replace function public.hq_has_grant(slug text)
returns boolean
language sql
stable
security definer
set search_path = public, pg_temp
as $$
  select exists (
    select 1
    from public.hq_grant_projection g
    where g.user_id  = public.hq_jwt_claim('sub')
      and g.app_slug = slug
  );
$$;

grant execute on function public.hq_jwt_claim(text) to authenticated;
grant execute on function public.hq_has_grant(text) to authenticated;

-- ═══════════════════════════════════════════════════════════════════════════
-- 3. hq_sync_checklists — the two-axis policy set
-- ═══════════════════════════════════════════════════════════════════════════
--
-- BOTH conjuncts are required, and each is proved separately by the suite:
--
--   owner_id = sub        the IDENTITY axis.    V7 (cross-owner read),
--                                               V10 (forged owner write)
--   hq_has_grant(slug)    the ENTITLEMENT axis. V8 (stale claim),
--                                               V9 (revocation replay),
--                                               V11 (forged grant write),
--                                               V12 (lockout)
--
-- Keeping them independent mirrors HQ, where roles and grants are separate axes
-- (see the AXIS test in tests/grant-enforcement-parity.spec.js: a grant does
-- not manufacture a role tier). Alice OWNS chk-alice-inv-1 and still cannot
-- read it, because she holds no live `inventory` grant. One axis alone cannot
-- express that, which is exactly why a single-predicate fixture would have
-- looked like it worked.
--
-- 🛑 This shape is the SIMPLEST correct bridge, not HQ's final policy. Real HQ
-- rows are frequently not single-owner — a checklist submission belongs to a
-- submitter AND an approver. Extending to that is the cutover card's work and
-- is a PRODUCT question about who may see whose submissions; this card
-- deliberately does not answer it and does not invent a permission concept to
-- pre-empt it.
alter table public.hq_sync_checklists enable row level security;

drop policy if exists hq_sync_checklists_select on public.hq_sync_checklists;
create policy hq_sync_checklists_select on public.hq_sync_checklists
  for select to authenticated
  using (
    owner_id = public.hq_jwt_claim('sub')
    and public.hq_has_grant(app_slug)
  );

-- WITH CHECK, not USING: an INSERT has no existing row to evaluate USING
-- against. A policy set with only USING clauses passes every read test in the
-- suite and still lets an attacker write rows owned by somebody else — which
-- is precisely what V10 and V11 caught in the red capture (both returned
-- HTTP 201 and the forged rows LANDED).
drop policy if exists hq_sync_checklists_insert on public.hq_sync_checklists;
create policy hq_sync_checklists_insert on public.hq_sync_checklists
  for insert to authenticated
  with check (
    owner_id = public.hq_jwt_claim('sub')
    and public.hq_has_grant(app_slug)
  );

-- UPDATE needs BOTH: USING decides which rows are updatable, WITH CHECK decides
-- what they may be updated INTO. With USING alone, Alice could re-own her own
-- row to Bob; with WITH CHECK alone, she could edit rows she cannot see.
drop policy if exists hq_sync_checklists_update on public.hq_sync_checklists;
create policy hq_sync_checklists_update on public.hq_sync_checklists
  for update to authenticated
  using (
    owner_id = public.hq_jwt_claim('sub')
    and public.hq_has_grant(app_slug)
  )
  with check (
    owner_id = public.hq_jwt_claim('sub')
    and public.hq_has_grant(app_slug)
  );

-- ═══════════════════════════════════════════════════════════════════════════
-- 4. hq_uid_trap — the auth.uid() NEGATIVE CONTROL
-- ═══════════════════════════════════════════════════════════════════════════
--
-- 🛑 NOT a product table and NOT a mistake. This is the copy-pasted-from-the-
-- hosted-docs policy, written on purpose, so variant V13 can show — same token,
-- same instant, same stack — that:
--
--     the plural-GUC policy on hq_sync_checklists  DISCRIMINATES
--     the auth.uid() policy on hq_uid_trap          RETURNS NOTHING
--
-- and that hq_uid_trap is not simply empty (the service_role control reads both
-- of its rows).
--
-- Do not "fix" this policy. Fixing it deletes the proof.
alter table public.hq_uid_trap enable row level security;

drop policy if exists hq_uid_trap_select on public.hq_uid_trap;
create policy hq_uid_trap_select on public.hq_uid_trap
  for select to authenticated
  using (owner_id = auth.uid()::text);  -- ← WRONG ON PURPOSE. See above.

-- ═══════════════════════════════════════════════════════════════════════════
-- 5. hq_grant_projection — the entitlement table is itself scoped
-- ═══════════════════════════════════════════════════════════════════════════
--
-- Without this, any authenticated user could read the entire grant map for the
-- whole company: who has access to what, in one GET. A user may read their own
-- grants (useful, and no more than their own token already tells them) and
-- nothing else. Nobody may write it over PostgREST at all — no INSERT/UPDATE/
-- DELETE policy exists, and section 4 of the fixture grants only SELECT.
--
-- hq_has_grant is SECURITY DEFINER precisely so this lockdown does not break
-- the policies that depend on it.
alter table public.hq_grant_projection enable row level security;

drop policy if exists hq_grant_projection_select on public.hq_grant_projection;
create policy hq_grant_projection_select on public.hq_grant_projection
  for select to authenticated
  using (user_id = public.hq_jwt_claim('sub'));

-- PostgREST caches the schema; policy changes need the reload signal.
notify pgrst, 'reload schema';
