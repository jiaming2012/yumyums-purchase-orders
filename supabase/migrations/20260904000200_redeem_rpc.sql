-- 20260904000200_redeem_rpc.sql
-- Activity A, card `redeem-rpc-race-proof` (night-crew run 20260904).
--
-- The atomic arbiter: `redeem(p_code, p_device)`. The single conditional
-- UPDATE below — `WHERE redeemed_by IS NULL AND expires_at > now()` — is the
-- ONLY thing enforcing single use in the whole design (§6, E-KR1): two
-- concurrent clients firing at one code get exactly one `ok=true`, arbitrated
-- by Postgres row locking, no advisory locks, no serializable isolation.
-- Proven under 20 rounds × 2 concurrent clients by
-- supabase/verify/04-redeem-race.sh (red-first against the naive
-- check-then-update analog, which demonstrably double-wins).
--
-- BODY: the spike's `redeem_v2` — operator-signed at the 2026-09-03 batch
-- review — NOT the handoff §6 text verbatim. The verbatim draft's reason
-- subquery returns NULL for a code id with no row, so a forged/unknown code
-- would surface downstream with an empty reason and read as a SYSTEM OUTAGE
-- (§18 edge-case 3 routes it to `failed`) instead of the `not_found` that
-- §9/§19's arbitration taxonomy names and the F2 reconciliation keys on.
-- GAP-1 (spike ledger `redeem-rpc-race-proof.md`, Comebacks). The v2 body
-- closes it: explicit `already_used` / `expired` arms, `coalesce(…,
-- 'not_found')`. Race behavior is identical (spike legs G vs V).
--
-- One deliberate delta from the spike's v2 text: the winning UPDATE also sets
-- `updated_at = now()`. `codes.updated_at` is the replication checkpoint key
-- (§4 — hit on every pull tick); Card 1's schema has no touch trigger, so a
-- redemption that does not advance it would be INVISIBLE to Activity B's pull
-- replica — a device would keep showing a burned code as live until some other
-- write touched the row. Asserted by the harness (leg H: updated_at ADVANCES).
-- The enforcement predicate and the signed taxonomy are untouched.
--
-- Wiring:
--   SECURITY DEFINER — callers hold only the RLS-scoped table grants
--     (authenticated: SELECT on codes, INSERT on scan_attempts); the burn
--     itself must not require a device-writable codes table. The definer
--     (supabase_admin here / postgres on a hosted project) owns the table.
--     search_path is pinned empty and every reference schema-qualified — a
--     definer function without that is hijackable via the caller's path.
--   EXECUTE: authenticated + service_role, NOT anon/PUBLIC. Devices call it
--     through PostgREST `/rpc/redeem` (outcomes come back via the return
--     value + the codes pull — Card 1's §7.2 note); HQ's Go orchestration
--     (Activity D) calls it as service_role. The revoke below names BOTH
--     public AND anon, and both halves are load-bearing: CREATE FUNCTION
--     grants EXECUTE to PUBLIC by default, and this substrate (like hosted
--     Supabase) also carries ALTER DEFAULT PRIVILEGES that hand anon an
--     EXPLICIT grant at create time — revoking PUBLIC alone leaves anon
--     executable (caught by 01-structure.sh's anon:false assertion on this
--     card's first green attempt). The structural harness asserts the final
--     state by name.
--
-- Idempotent per the Card 1 convention: `drop function if exists` + create
-- (survives return-type changes, unlike bare `create or replace`), then
-- re-grant. Applies clean on a bare substrate and on top of its own output.
-- Apply order and target discipline: see supabase/README.md.

drop function if exists public.redeem(uuid, text);
create function public.redeem(p_code uuid, p_device text)
returns table (ok boolean, reason text)
language plpgsql
security definer
set search_path = ''
as $$
begin
  update public.codes c
     set redeemed_by = p_device,
         redeemed_at = now(),
         updated_at  = now()   -- the replication checkpoint key must move (§4)
   where c.id = p_code
     and c.redeemed_by is null
     and c.expires_at > now();
  if found then
    return query select true, null::text;
  else
    return query select false, coalesce(
      (select case when c.redeemed_by is not null then 'already_used'
                   when c.expires_at <= now()     then 'expired'
              end
         from public.codes c where c.id = p_code),
      'not_found');   -- never NULL for a missing row — GAP-1 (§9/§19 taxonomy)
  end if;
end $$;

revoke all on function public.redeem(uuid, text) from public, anon;
grant execute on function public.redeem(uuid, text) to authenticated, service_role;

-- Nudge PostgREST's schema cache so /rpc/redeem is servable immediately
-- (same rationale as Card 1's migration; a no-op when nothing listens).
notify pgrst, 'reload schema';
