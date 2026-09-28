-- 20260906000300_scan_attempts_policy_unresolved.sql
-- Activity B, card `refusal-holds-before-sync` (night-crew run 20260906-2).
-- Closes B-432's second half (done_when clause 2) and disposes B-434 (a).
--
-- ═══ 1. THE DISCRIMINATOR ═══════════════════════════════════════════════════
--
-- done_when: "the campaigns-replica failure path is distinguishable from a
-- genuinely-unknown campaign in the attempt record." Both cases land as
-- `unverified_code = true` offline overrides with `status = 'accepted'` — the
-- §9/§19 taxonomy is UNCHANGED, no new terminal status (that is the card's
-- PARK line and it was never approached). What tells them apart is this
-- column:
--
--   unverified_code | policy_unresolved | what actually happened
--   ----------------+-------------------+---------------------------------------
--   t               | t                 | the campaigns replica had NOT delivered
--                   |                   | (empty / still pulling / erroring — the
--                   |                   | B-432 window). The device could not
--                   |                   | resolve §8 policy at scan time.
--   t               | f                 | the campaigns replica was healthy and the
--                   |                   | code genuinely names no campaign we know
--                   |                   | — decision 166's ratified F2 case.
--
-- Spike 03 (`discriminator-lands-without-poison`) measured this end to end
-- against a spike-local copy of exactly this DDL: the discriminated override
-- landed `a1… | accepted | t | t | t`, the genuinely-unknown control
-- `b2… | accepted | t | t | f`, and a legitimate redeem queued BEHIND the
-- discriminated attempt still landed `accepted` with exactly one redeem call —
-- no head-of-line poison, the F-2 check constraint intact.
--
-- 🛑 SEQUENCING IS NOT OPTIONAL. Spike 03 also measured the PRE-migration
-- server's answer to a landing body carrying this field: HTTP 400 `PGRST204`
-- ("Could not find the 'policy_unresolved' column"). That is the same
-- throw-retry head-of-line poison class F-2 measured (12 redeem attempts, 0
-- landings). **This file must be in the tree, and applied to the arbiter,
-- BEFORE any client sends the field.** On this branch it is committed one
-- commit ahead of the client change that populates it, deliberately.
--
-- NOT NULL DEFAULT false is safe on an existing table: PG11+ rewrites nothing
-- for a non-volatile default, and every row landed before this card genuinely
-- had a resolvable policy (the fail-closed predicate did not exist yet, so no
-- pre-card attempt can have been an unresolved-policy override).
--
-- ═══ 2. RIDER B-434(a) — the flags get COUPLED, deliberately ════════════════
--
-- B-434(a), the carried finding "with teeth": nothing coupled `unverified_code`
-- to `offline_override`. The landed constraint permitted
-- `code_id NULL + unverified_code + token_hash` REGARDLESS of the override
-- flag, so a future producer that set `unverified_code` without
-- `offline_override` would land an unverified code as `status='accepted'` —
-- an accepted redemption for a code nobody can name and nobody authorised.
-- Unreachable today only because the sole producer (`submit-flow.js`'s
-- doOverrideWrite) happens to always set both.
--
-- The card offered two dispositions: couple them, or state the decoupling as
-- deliberate. **We couple them.** The decoupled shape has no reader and no
-- use case: §19 F2's unverified path IS the permissioned offline-override
-- path — an unverified attempt that is not an override has nothing to be
-- (it cannot be an online submit; those go to /rpc/redeem with a real
-- code_id). Leaving the door open costs nothing today and costs an
-- unauthorised accepted redemption the first time someone builds a second
-- producer. A constraint is where that belongs, not a comment.
--
-- Verified compatible with the landed producers before tightening:
--   * marketing/sync/push-replication.js `land-unverified` body — sends
--     `offline_override: doc.offline_override`, and the only path that sets
--     `unverified_code` on a local row is doOverrideWrite, which sets
--     `offline_override: true` unconditionally;
--   * marketing/sync/harness/f2-harness.mjs — both legs set
--     `offline_override: true` (green and red-unflagged).
--
-- A NEW numbered file, never an edit to a landed one (20260906000200 stays
-- byte-identical). Idempotent per the Activity A convention: ADD COLUMN IF NOT
-- EXISTS + drop/add constraint — applies clean on a bare substrate AND on top
-- of its own output.

alter table public.scan_attempts
  add column if not exists policy_unresolved boolean not null default false;

comment on column public.scan_attempts.policy_unresolved is
  'true when the device could not resolve §8 campaign policy for this attempt '
  '(campaigns replica empty, still delivering, or erroring). With '
  'unverified_code it distinguishes a campaigns-replica FAILURE override (t,t) '
  'from a genuinely-unknown-campaign override (t,f). Card '
  'refusal-holds-before-sync / B-432.';

-- Rider B-434(a): an unverified attempt must also be an authorised override.
alter table public.scan_attempts drop constraint if exists scan_attempts_names_a_code;
alter table public.scan_attempts add constraint scan_attempts_names_a_code
  check (code_id is not null
         or (unverified_code and offline_override and token_hash is not null));

-- Nudge PostgREST's schema cache (Card 1's convention; a no-op when nothing
-- listens). Without this the new column draws PGRST204 until the next reload —
-- the exact poison class this migration exists to sequence around.
notify pgrst, 'reload schema';
