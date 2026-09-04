# Spikes — scan-attempts-push-conflict

Activity: Activity B — Offline-first replica (RxDB ↔ Supabase)

> No `usm/roadmap.txt` on this target — hand-run convention (see the sibling
> ledger `rxdb-pull-replica.md` for the full preamble, the #6 prerequisite
> stub, and the substrate discipline; both apply verbatim here).

## The goal, and which legs need a spike

The card (roadmap Activity B): the device-owned, **push-only** `scan_attempts`
collection (§4 — the opposite replication direction, the key structural
decision). The push handler batches pending attempts through `redeem()` and
writes the outcome back onto the local row; the `conflictHandler` flips a
losing device's UI from "redeemed ✓" to "already used at 6:42pm" (§6).
done_when: a lost-race attempt renders "already used" with the winning
time/device.

Already proven elsewhere: `redeem()`'s exactly-one-winner atomicity (verify 04),
RLS letting a device insert `scan_attempts` only as itself and never read them
back (Activity A spike 02 — push-only holds at the API surface).

The unproven, falsifiable premise: the **round trip on the losing device**.
Queued-offline attempts → push replication → per-attempt `redeem()` → outcome
written back onto the device's own local row — under the constraint the RLS
design imposes: the device **cannot SELECT** `scan_attempts` server-side, so
RxDB's normal push machinery (which may re-read masters to detect conflicts)
and the outcome write-back must work against a write-only table. A push design
that quietly depends on reading the server rows back falsifies here. Second
premise folded in: the write-back (a local patch) must not re-trigger an
infinite push loop — the handler has to be idempotent over already-resolved
rows.

## Spike: push-lost-race-flips

- proves: two RxDB device clients (device-a, device-b), each holding a
  locally-queued `pending` attempt for the SAME seeded unredeemed code
  (the §8 offline double-accept, reconstructed), both start push-only
  replication concurrently; every pending attempt flows through the committed
  `redeem()` RPC exactly once; **exactly one** device's local row ends
  `accepted`; the OTHER device's local row ends `rejected / already_used` AND
  carries the winning device + redemption time (read from the pulled-side
  `codes` row — the data the conflictHandler flip renders); both attempt rows
  land server-side with each device's own `device_id` (RLS with-check intact);
  and the outcome write-back does not loop (push handler invocation count is
  bounded and enumerated). The server rows are verified via psql
  (supabase_admin), NOT via device reads — the devices provably cannot read
  them (a device-role SELECT on scan_attempts must still refuse, asserted).
- plan: source `supabase/verify/lib.sh`; substrate up; `reset_bare` +
  `apply_all`; seed one per-run unredeemed code; Node script builds two RxDB
  dbs (schemas mirroring `scan_attempts`), inserts one local `pending` attempt
  each BEFORE any replication starts (the offline queue), then starts both
  push replications (custom handler: for each pending row → POST /rpc/redeem
  as the device JWT → POST /scan_attempts with the resolved status/reason →
  return []; resolved rows skipped), patches local rows with outcomes, awaits
  both in-sync, and runs the assertions above; psql enumerates the server
  attempt rows and the single `redeemed_by` winner.
- script: .night-crew/spikes/activity-b-offline-first-replica/scan-attempts-push-conflict/01-push-lost-race-flips.sh

## Verdict (run 2026-09-04, hand-run per the no-story-map convention)

- **push-lost-race-flips: passed** — exit 0 (second run; the first was RED on a
  schema fact the spike surfaced, see Corrections). Both devices queued a
  pending attempt for one code; concurrent push; device-a's `redeem()` answered
  `ok=true`, device-b's `already_used`; device-b's LOCAL row flipped to
  `rejected / already_used` carrying `winner_device=device-a` and the winning
  time (read from its own codes-side pull — the render data for "already used
  at 6:42pm"); handler invocations bounded at 2 per device (no write-back
  loop); device SELECT on `scan_attempts` answered **403** (push-only holds at
  the API surface); server-side enumeration (supabase_admin): exactly 2
  attempt rows, one accepted one rejected, `codes.redeemed_by` matching the
  accepted row's device.

**Conclusion:** the losing device's round trip works against a write-only
table — the push handler needs no server read-back of `scan_attempts`; the
loser's display data comes from the codes replica. Build-facts the card
inherits: (1) `scan_attempts.id` is a **uuid** — devices must generate real
uuids (`crypto.randomUUID()`), not app-prefixed strings; (2) redeem-then-land
is NOT atomic client-side: if landing the attempt row fails after `redeem()`
succeeded, RxDB's push retry re-runs the handler and the re-burn answers
`already_used` **to the winning device itself** — the card's handler must
treat `already_used where codes.redeemed_by == own device` as accepted (or
persist the burn outcome locally before landing), or a transient landing
failure mis-flips the winner's UI. The spike hit the retry path on its red
first run; the green run's `status !== 'pending'` skip covers the
patched-row loop but not that partial-failure window.

## Corrections

- **Schema fact surfaced by the first run (fixed, re-run green):
  `scan_attempts.id` is `uuid`, and the spike's app-prefixed string ids
  (`att-device-b-…`) drew PostgREST 400 on the landing insert.** The fix —
  `crypto.randomUUID()` on-device — is also the build-fact the scanner card
  inherits (§4 "id uuid primary key — generated on device" is literal). The
  failing insert also demonstrated the redeem-then-land partial-failure window
  recorded in the conclusion: the push retry re-ran `redeem()` for a code the
  same flow had already burned. No premise changed; the premise held once the
  id shape matched the schema.

## Review

- signed: operator, 2026-09-04 — covers 1 correction(s) (one-sitting batch
  across Activities B/C/D; "Sign off all three" on the phrase-checked batch
  question at the sitting's close).
