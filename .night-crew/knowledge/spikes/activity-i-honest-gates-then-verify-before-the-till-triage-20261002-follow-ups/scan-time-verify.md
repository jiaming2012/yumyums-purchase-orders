# Spikes — scan-time-verify

Activity: Activity I — Honest gates, then verify before the till (triage 20261002 follow-ups)

> Hand-run convention (see README.md in this directory). Runs against the LOCAL
> spike-supabase substrate's PostgREST with a device JWT (the shape of
> `../activity-h-designed-tabs/scanner-polish` spike 02); read-only.

## The goal, and which legs need a spike

The card (I2 — **B-468**, promoted by the operator above the backlog, ledger T-62 decision
200): an online phone never verifies a code it has not synced. `marketing/scanner.js`
`resolve()` consults three LOCAL sources only (offers replica, codes replica, the embedded
offer) and contains no network call, so a code minted since the device last synced resolves
`unknownCode` regardless of connectivity, the crew keys the discount into Toast, and only the
submit asks the server. Direction decided: the device already holds a connection to the
substrate and `public.codes.token_hash` is unique and indexed, so an ONLINE phone looks up the
hash it already computed, on the connection it already has — one query, no new backend
surface, falling back to today's behaviour on timeout. Behaviour change: a freshly-printed
code scanned on an online phone shows its real offer (or "already used") before any discount
is applied; offline behaviour is unchanged (decisions 166/199).

One premise a script can settle now: the device (`authenticated`) role may read ONE code row by
`token_hash` from PostgREST, with the columns the resolver needs, inside the connectivity
probe's 3.5 s budget, and an unknown hash answers an empty list rather than an error.

## Spike: device-reads-one-code-by-token-hash

- proves: `GET /codes?token_hash=eq.<seeded hash>&select=id,campaign_id,expires_at,redeemed_at,redeemed_by`
  with a device JWT answers 200 and exactly one row carrying `campaign_id`; the same query for
  a random hash answers 200 `[]`; both complete well inside 3500 ms. A redeemed seeded row (if
  the seed has one) is also readable, so "already used" can be shown at scan time.
- plan: mint a device JWT with `.night-crew/qa/spike-supabase/mintjwt`, curl the local
  PostgREST, time each call, assert shape and count.
- script: .night-crew/spikes/activity-i-honest-gates-then-verify-before-the-till-triage-20261002-follow-ups/scan-time-verify/01-device-reads-one-code-by-token-hash.sh

### Runs

- 2026-10-03T13:37:11Z · exit 0 · passed

## Verdict (hand-run 2026-10-02)

- **device-reads-one-code-by-token-hash: passed** — exit 0, first run, no retry: (a) the
  seeded unredeemed hash → HTTP 200, 1 row, `campaign_id` populated, all five columns present,
  0.081 s (connection setup; later calls ~10 ms); (b) a random 64-hex hash → 200 `[]`, 0.013 s;
  (c) the seeded REDEEMED row (`…-0004`) → 200, 1 row with `redeemed_at` / `redeemed_by` set —
  "already used" is visible at scan time; (d) every call two orders of magnitude inside the
  3.5 s probe budget; (e) observation: `HEAD … Prefer: count=exact` answers `Content-Range:
  0-0/1` / `*/0`, so a body-less existence check is also available to the device role.

## Corrections

- none — no agent-reached corrections. Two build-facts for the card, not corrections: RLS
  (`codes_select_device … using (true)` + table-wide `grant select`) hides neither redeemed nor
  expired rows and every column the resolver needs is selectable; PostgREST returns timestamps
  with a `+00:00` offset, not `Z`, so the expiry comparison must parse ISO offsets (`Date.parse`
  does).

## Comebacks

- none recorded — the latency budget is dominated by the real LTE network, not PostgREST; the
  card's timeout is the connectivity probe's, and `[SV-03]` kills the request at the network
  layer to prove the fallback.

## Review

- signed: operator, 2026-10-02 — covers 0 correction(s) (no agent-reached corrections; reviewed
  in the same slate sitting)
