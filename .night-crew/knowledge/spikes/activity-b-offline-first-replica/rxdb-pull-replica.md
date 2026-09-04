# Spikes — rxdb-pull-replica

Activity: Activity B — Offline-first replica (RxDB ↔ Supabase)

> This target repo has no `usm/roadmap.txt` story map (the layout the
> `night-crew spikes gate/run` verbs read), so those verbs cannot drive here —
> the established hand-run convention (activity-5, Activities A + G last
> sitting). This ledger and the runnable scripts that ARE the verdict (B-345)
> are authored to the skill's paths anyway.
>
> **Prerequisite stub (§15cc.253, stated):** Activity B is gated on Activity 0's
> field observation #6 (do the three devices go offline independently?). That
> observation has not run. These spikes are authored ahead of it and **assume
> the offline-first shape stands**. If #6 answers "always together on one
> hotspot" and Activity B collapses to a thin live cache, what these spikes
> prove — bounded checkpointed pull against the real schema — is the mechanism
> the thin cache would use too; the evidence prices both shapes. The spike does
> NOT wait on #6 because being wrong about the replication premise is the exact
> B-345 class this cycle must not repeat (R1: the "reuse" is greenfield; the
> cutover never happened in prod).
>
> **Substrate:** the committed LOCAL `spike-supabase` compose project only
> (reconcile mode, never `--fresh`), against the **built** Activity A artifacts
> — `supabase/migrations/*.sql` + `supabase/seed.sql` applied by the committed
> `supabase/verify/lib.sh` helpers. Never :5433, never :5434, no hosted project
> (none exists yet — Activity 0 provisions it).

## The goal, and which legs need a spike

The card (roadmap Activity B): two server-owned pull-only replicas via
`replicateRxCollection` with an `updated_at` checkpoint — (1) `codes` filtered
`expires_at > now() - interval '2 days'` (§5.3); (2) non-expired offers keyed on
customer hash (§10). done_when: a code redeemed on device A shows spent on
device B after a pull tick; offers resolve offline; un-synced customers fall
back to the embedded offer.

What is already proven and NOT re-spiked here: RxDB pull replication converging
live against this substrate (last cycle's proof-pull.js — insert/update/
soft-delete, no restart, no manual reSync), the browser as client against the
real substrate (spike F browser-live), Realtime frames reaching an
authenticated subscriber on `public.codes` (Activity A spike 03), and
`redeem()`'s atomicity + its `updated_at` stamp advancing (verify 04 leg H).

What is NOT yet proven — the falsifiable premises of THIS card:

1. the pull can be **bounded** (the §5.3 expiry-window filter) while staying
   **checkpointed** on `updated_at` — a full-table pull that ignores the bound,
   or a bound that breaks checkpoint resumption, each falsifies the design;
2. a **real redemption** (the committed `redeem()` RPC, not a hand UPDATE)
   propagates through that bounded, checkpointed pull to a second device's
   local replica without restart — the done_when core.

The offers replica is the same mechanism with a different filter and key; it
rides premise 1–2 and is not a separate premise (stated, not silently skipped).
The embedded-offer fallback is Activity E's `identity-code-and-qr` premise, not
this card's.

## Spike: pull-bounded-checkpoint

- proves: an RxDB client (`replicateRxCollection`, custom pull handler against
  PostgREST as a device JWT — the §3 "chosen" shape) over the BUILT Activity A
  schema (a) lands the seeded in-window codes on initial sync while a code
  outside the §5.3 two-day window does NOT land (the bounded negative — a
  let-everything-through pull passes the positive and fails this); (b) resumes
  by checkpoint: the post-initial pull request observably carries a non-initial
  `updated_at` cursor (request log enumerated, B-216 — not inferred from
  results); and (c) converges the done_when core: a code burned server-side by
  the committed `redeem()` RPC (device A) surfaces as redeemed in device B's
  running local replica via a Realtime nudge → pull tick, with NO client
  restart and NO manual reSync (proof-pull discipline). Assumes Activity A done
  (it is) and stubs nothing else; the #6 stub note above covers the activity
  gate.
- plan: source `supabase/verify/lib.sh`; substrate up (reconcile); `reset_bare`
  + `apply_all` (real migrations + seed); seed two per-run extra codes — one
  expired ~1 day (IN the 2-day window) and one expired ~5 days (OUT); run a
  Node RxDB client (memory storage, schema mirroring `codes`) importing the
  proven `spike-env.js` bridge (ports, mintjwt, supabase-js shim), with a
  custom pull handler querying PostgREST
  (`order=updated_at.asc,id.asc` + `updated_at=gt.<checkpoint>` +
  `expires_at=gt.<now-2d>`) and `pull.stream$` emitting RESYNC on
  postgres_changes / re-SUBSCRIBED (§7.3); assert (a) then fire
  `select public.redeem(<code>, 'device-a')` via psql and assert (b) and (c)
  within a bounded window; print the full pull-request log.
- script: .night-crew/spikes/activity-b-offline-first-replica/rxdb-pull-replica/01-pull-bounded-checkpoint.sh

## Verdict (run 2026-09-04, hand-run per the no-story-map convention)

- **pull-bounded-checkpoint: passed** — exit 0, first run. Enumerated evidence:
  initial pull landed exactly 6 docs (the four in-window seed fixtures + the
  per-run in-window-expired + the live target); the out-of-window per-run code
  (expired 5d) and seed fixture …0003 (expired 2026-01-01) did NOT land — the
  §5.3 bound held both ways. The committed `redeem()` fired through PostgREST
  as device-a (`ok=true`), a Realtime frame nudged RESYNC, and device-b's
  RUNNING replica showed `redeemed_by=device-a` **254 ms** later — no restart,
  no manual reSync. The pull-request log shows 3 requests: #1 epoch, #2 (the
  SUBSCRIBED refetch, §7.3) and #3 both carrying the real
  `updated_at=gt.2026-09-04T13:16:08.176714+00:00` cursor — checkpointed, not
  re-reading the world.

**Conclusion:** the card is buildable as designed — `replicateRxCollection`
with a custom PostgREST pull handler + RESYNC-on-Realtime carries the bounded,
checkpointed replica on the built schema. Build-facts the card inherits:
(1) the checkpoint cursor MUST be URL-encoded (`+00:00` in a raw query string
decodes as a space and breaks the timestamptz parse) — the spike's handler
does `updated_at=gt.${encodeURIComponent(cursor)}`; (2) the spike uses a `gt`
cursor with `order=updated_at.asc,id.asc` — same-timestamp ties are a
theoretical missed-row edge at batch boundaries; the card should keep the id
tiebreak in the checkpoint or over-fetch one row; (3) the offers replica is
this same mechanism with an `expires_at > now()` filter keyed on customer hash
— no separate premise; (4) `redeem()`'s `updated_at = now()` stamp (decision
163) is what makes the redemption visible to the pull tick — verify 04 leg H
already guards it.

## Corrections

- none — no agent-reached corrections. The script passed on its first run.

## Review

- n/a for this goal alone — no corrections here; the sitting's batch review
  (see the sibling ledgers) covers the goals that have them.

## Comebacks

- gap: GAP-1 — the spiked `gt`-cursor checkpoint can skip same-`updated_at`
  rows at a batch boundary (theoretical missed-row edge the spike's own
  conclusion names; the silent-miss class is a redemption never reaching
  device B). Fix lands with Activity B's pull-replica card keeping the id
  tiebreak in the checkpoint (or over-fetching one row); that card owes ONE
  validation run exercising a same-timestamp batch boundary (found
  2026-09-04, /nc-spike-close).
