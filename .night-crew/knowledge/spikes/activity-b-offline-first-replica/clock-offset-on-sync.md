# Spikes — clock-offset-on-sync

Activity: Activity B — Offline-first replica (RxDB ↔ Supabase)

> No `usm/roadmap.txt` on this target — hand-run convention (see
> `rxdb-pull-replica.md` for the full preamble, the #6 prerequisite stub, and
> the substrate discipline; both apply verbatim here).

## The goal, and which legs need a spike

The card (roadmap Activity B): on every successful sync, store
`serverNow − deviceNow` and apply that offset in the offline `expires_at`
comparison (§5.1) — a tablet with a wrong date must not silently accept dead
codes.

Two falsifiable premises, one spike (they only make sense run together):

1. **A server-time signal is obtainable from traffic the sync already makes.**
   The design says "on every successful sync" but names no source. The
   candidate this spike falsifies-or-confirms: the HTTP `Date` header on the
   PostgREST pull response (whole-second resolution — plenty for an
   expiry-window comparison measured in hours). If PostgREST doesn't emit a
   usable `Date`, the card needs a `now()` RPC instead — the spike's failure
   mode names the fallback.
2. **The offset-adjusted comparison actually closes the §5.1 hole.** The
   dangerous direction is a device clock running SLOW: a recently-expired code
   reads as still-valid and gets accepted offline. The naive comparison must
   demonstrably ACCEPT the dead code under a 2-days-slow clock (the red
   analog — without it the green is unfalsifiable), and the adjusted
   comparison must reject it.

## Spike: skewed-clock-still-rejects

- proves: against the built schema on the local substrate, (a) the PostgREST
  pull response carries a `Date` header from which
  `offset = serverNow − deviceNow` is computable, and under an injected
  2-days-slow device clock the measured offset recovers the skew to within
  ±10s (enumerated: header value, skewed deviceNow, computed offset);
  (b) red analog — the NAIVE local check (`deviceNow < expires_at`) ACCEPTS a
  code that expired ~1 day ago when the device clock is 2 days slow, proving
  the defect class §5.1 names is real on this data; (c) the offset-adjusted
  check (`deviceNow + offset < expires_at`) REJECTS that same code under the
  same skewed clock. The comparison is pure local arithmetic — exactly what
  the offline path runs.
- plan: source `supabase/verify/lib.sh`; substrate up; `reset_bare` +
  `apply_all`; seed one per-run code with `expires_at = now() - interval
  '1 day'` (inside the §5.3 replica window — the exact dangerous row); Node
  script fetches the codes pull query as a device JWT, reads the `Date`
  header, injects `deviceNow = realNow - 2 days`, computes the offset, runs
  the naive and adjusted comparisons, asserts (a)/(b)/(c); per-run row deleted
  on green.
- script: .night-crew/spikes/activity-b-offline-first-replica/clock-offset-on-sync/01-skewed-clock-still-rejects.sh

## Verdict (run 2026-09-04, hand-run per the no-story-map convention)

- **skewed-clock-still-rejects: passed** — exit 0, first run. (a) the PostgREST
  pull response carries a `Date` header; under the injected 2-days-slow device
  clock the computed offset recovered the skew to **196 ms** error (rtt 85 ms —
  well under the 10s tolerance, plenty for hour-scale expiry windows); (b) red
  analog held: the naive `deviceNow < expires_at` check ACCEPTED the code that
  expired a day earlier — the §5.1 defect class is real on this data; (c) the
  offset-adjusted comparison REJECTED the same code under the same skew.

**Conclusion:** the card needs no new endpoint — the `Date` header on the pull
response the sync already makes is the serverNow source (whole-second
resolution, fine for expiry comparisons measured in hours). Build-facts:
capture the offset on every successful pull, store it beside the checkpoint,
and use `deviceNow + offset` in every offline `expires_at` comparison; the
sign convention is `offset = serverNow − deviceNow`.

## Corrections

- none — no agent-reached corrections. The script passed on its first run.

## Review

- n/a — no corrections, so no batch review is owed for this goal (§4 fires
  only when the ledger holds agent-reached corrections).
