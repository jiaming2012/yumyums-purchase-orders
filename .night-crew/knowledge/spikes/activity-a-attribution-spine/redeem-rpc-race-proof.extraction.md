# Extraction — redeem-rpc-race-proof

Outcome: learned

Approach used: the single conditional UPDATE as the mutex —
`UPDATE codes SET redeemed_by=…, redeemed_at=now() WHERE id=… AND redeemed_by
IS NULL AND expires_at > now()` in plpgsql, returning `(ok, reason)` — raced
by two independent connections (one `docker exec psql` each, autocommit)
launched simultaneously, 20 rounds on fresh codes, with a deliberately
widened-TOCTOU naive check-then-update function as the red analog proving the
harness detects the defect class. The proven function body is the **v2
draft** in
`.night-crew/spikes/activity-a-attribution-spine/redeem-rpc-race-proof/sql/redeem-fns.sql`
— a candidate the card's design.md adopts or not (NFR-6).

Confirmed: the cycle's most load-bearing premise (§6, E-KR1) — exactly one
winner per round across all 20 rounds, 0 double-wins, 0 lost redemptions,
every loser `already_used`; an expired code refuses both clients with
`expired` and stays unredeemed; and the naive analog double-wins on demand, so
the race test the card must ship can red on the defect and green on the fix.

Learned: the handoff §6 text has a reason-taxonomy hole — for a code id with
no row, its reason subquery returns NULL, so a forged or mistyped code
surfaces as `(false, NULL)`. Downstream (§18 edge-case 3) that empty reason
routes to `failed`: a forged code would read as a system outage instead of
the `not_found` that §9's audit trail and §19 F2's reconciliation key on. The
v2 body (explicit `already_used`/`expired` arms, `coalesce(…, 'not_found')`)
closes the hole with race behavior proven identical.

Plan change: Activity A's card builds `redeem()` with the **v2 body**, not
the handoff §6 text verbatim — operator-signed at the 2026-09-03 batch
review. Consequences downstream: Activity D's route_outcome can treat
`not_found` as a first-class reason (its `failed` fallback stays for
genuinely empty/unknown results), and Activity F's reconciliation buckets get
`not_found` for forged/unknown codes rather than an error bucket. The gap →
fix → validated link is anchored as GAP-1 in the ledger's `## Comebacks`; the
card that ships `redeem()` owes the one re-validation run against the built
migration in the same sitting it lands.
