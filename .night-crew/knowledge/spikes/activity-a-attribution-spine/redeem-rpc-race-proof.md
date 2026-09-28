# Spikes — redeem-rpc-race-proof

Activity: Activity A — The attribution spine (the Supabase arbiter)

> Same conventions as the sibling `supabase-schema-and-rls.md` ledger: no
> `usm/roadmap.txt` here, so the `spikes gate/run` verbs cannot drive; the
> script is the verdict (B-345). Substrate: the committed throwaway
> spike-supabase stack — the premise is pure plpgsql + Postgres row locking,
> which is identical hosted or local.
>
> **Dependency, not a stub:** this goal's script applies the sibling goal's
> REAL schema fixture (`supabase-schema-and-rls/sql/qr-schema.sql`) before
> creating the functions — the prerequisite is satisfied by sharing the actual
> draft, not by stubbing what that goal will deliver.

## The goal, and which legs need a spike

The card (roadmap Activity A): the `redeem(p_code, p_device)` plpgsql function
(§6) — conditional `UPDATE … WHERE redeemed_by IS NULL AND expires_at > now()`
— and the race test that is the point of the card. done_when: "the race test
passes 20× with 0 double-wins", red-first against a naive check-then-update.
This is E-KR1's premise and the single most load-bearing claim of the cycle
(§6: the atomic UPDATE "is the **only** thing actually enforcing single use").
B-345 says exactly this premise class must be falsified by a one-row script on
day one — this is that script.

## Spike: race-atomic

- proves: five legs, one script, in order —
  1. **R (red analog)**: a naive check-then-act (`naive_redeem`, TOCTOU window
     widened with `pg_sleep(0.4)`) DOUBLE-WINS under two concurrent clients —
     the harness demonstrably detects the defect class, so the green legs are
     falsifiable (the roadmap's greenfield red-first rule).
  2. **G (the premise)**: `redeem_verbatim` (§6's text) under 20 rounds × 2
     concurrent clients (fresh code per round): exactly one `ok=true` per
     round, 0 double-wins, 0 zero-win rounds, every loser `already_used`.
  3. **E**: an expired code refuses both clients with `expired` and stays
     unredeemed.
  4. **N (gap characterization)**: an unknown uuid — the verbatim §6 draft
     returns `(false, NULL)` because its reason subquery has no row, while
     §9/§19's arbitration taxonomy names `not_found`. The assertion pins the
     gap precisely.
  5. **V (gap closed)**: `redeem_v2` (the corrected draft: `coalesce(…,
     'not_found')` + an explicit expired arm) returns `not_found` for the
     unknown uuid and races identically (1 winner) on a fresh code.
- plan: substrate up via `env-up.sh`; apply the sibling goal's schema fixture
  + `sql/redeem-fns.sql`; each "client" is its own `docker exec psql`
  connection (autocommit), pairs launched with `&` + `wait`; rounds
  configurable via `SPIKE_RACE_ROUNDS` (default 20, the card's done_when
  number).
- script: .night-crew/spikes/activity-a-attribution-spine/redeem-rpc-race-proof/01-race-atomic.sh

## Verdict (run 2026-09-03, hand-run per the no-story-map convention)

- **race-atomic: passed** — exit 0. The five legs, in order:
  - **R** — `naive_redeem` double-won on try 1 (both clients `true`): the
    harness demonstrably detects the defect class, so the greens below are
    falsifiable (the greenfield red-first requirement, satisfied).
  - **G** — `redeem_verbatim`, 20 rounds × 2 concurrent clients: **exactly one
    winner every round, 0 double-wins, 0 lost redemptions, every loser
    `already_used`**. E-KR1's core claim (§6: the atomic UPDATE is the only
    thing enforcing single use) is proven, at the count the card's done_when
    names.
  - **E** — an expired code refused both clients with `expired`; the row
    stayed unredeemed.
  - **N** — unknown uuid against the verbatim §6 text: `(false, NULL)`. The
    gap is pinned: §9/§19's taxonomy names `not_found`, and the handoff's
    reason subquery cannot produce it (no row → NULL).
  - **V** — `redeem_v2` (the corrected draft) returns `(false, 'not_found')`
    for the unknown uuid and races identically (1 winner on a fresh code).

**Conclusion:** the cycle's most load-bearing premise holds — a conditional
`UPDATE … WHERE redeemed_by IS NULL AND expires_at > now()` is a real mutex on
Supabase-shaped Postgres, and the race test the card must ship is proven able
to red on the defect and green on the fix. Build-facts the card inherits:
(1) **adopt the `redeem_v2` shape, not the handoff §6 text verbatim** — see
Corrections; (2) each racing client must be its own connection (the spike's
`docker exec psql` pairs); (3) `sql/redeem-fns.sql` here carries both drafts
and the naive red-analog for the card's red-first leg.

## Corrections

- **Design delta from the handoff §6 text (spike-proven): the verbatim
  `redeem()` returns `(false, NULL)` for a code that does not exist, while the
  design's own arbitration taxonomy (§9 audit reasons, §19 F2 "not_found /
  already_used / expired") requires `not_found`.** Left as-is, an unknown or
  forged code surfaces to the gstate machine (Activity D) with an empty
  reason — §18 edge-case 3 routes that to `failed`, so a *forged code* would
  read as a *system error* instead of the `not_found` the F2 reconciliation
  and the reconciliation view (Activity F) key on. The corrected draft
  (`redeem_v2`: `coalesce` over an explicit `already_used`/`expired` case,
  falling through to `'not_found'`) closes it without changing race behavior
  — proven in legs V. **The card should build `redeem()` with the v2 body.**
- **Assertion-shape defect fixed pre-run (same class as the sibling goal's):**
  greps expected psql bare-boolean `t|`/`f|` where the concatenated
  `ok||'|'||…` output casts to `true|`/`false|`. Fixed before this script's
  first execution as part of working the sibling's first-run RED. No premise
  changed.

## Review

- signed: operator, 2026-09-03 — covers 2 correction(s) (batch sitting; the
  not_found design delta was presented as the product-changing item and signed
  "Sign off all three" — the card builds `redeem()` with the v2 body).

## Comebacks

- gap: GAP-1 — the handoff §6 `redeem()` returns a NULL reason for a
  nonexistent code, so a forged/unknown code reads as a system failure instead
  of `not_found` (spike leg N, 2026-09-03). Fix lands with Activity A's
  `redeem-rpc-race-proof` card building the v2 body; that card owes ONE
  re-validation run of this spike (or a successor against the built
  `supabase/` migration) in the sitting it lands, recorded as a `validated:`
  line naming GAP-1.
- validated: GAP-1 — re-validated 2026-09-04 against the BUILT migration, in the
  `redeem-rpc-race-proof` card's own sitting (run 20260904, branch
  `wo-redeem-rpc-race-proof`). The card built `redeem()` with the v2 body
  (`supabase/migrations/20260904000200_redeem_rpc.sql`) and the spike's successor
  harness (`supabase/verify/04-redeem-race.sh`, self-contained in-repo) ran
  against it: **exit 0** — unknown uuid → `(false, 'not_found')` (leg N, never a
  NULL reason), 20 rounds × 2 concurrent clients with exactly one winner each and
  every loser `already_used` (race behavior unchanged by the fix, as legs G/V
  predicted), expired arm `(false, 'expired')`. Red-first held: the same legs
  exited 1 with an observed round-01 double-win when the naive check-then-update
  body was installed as `public.redeem`. Evidence:
  `.night-crew/runs/2026-09-04-autonomous/card2-green-04-redeem-race.log`
  (EXIT=0) and `card2-red-04-race-naive.log` (EXIT=1). GAP-1 is closed.
