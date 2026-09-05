# Spikes — requires-online-replication

Activity: Activity B — Offline-first replica (RxDB ↔ Supabase)

> This target repo has no `usm/roadmap.txt` story map (the layout the
> `night-crew spikes gate/run` verbs read), so those verbs cannot drive here —
> the established hand-run convention (activity-5, Activities A + G, and this
> activity's three sibling ledgers). This ledger and the runnable scripts that
> ARE the verdict (B-345) are authored to the skill's paths anyway.
>
> **Substrate:** the committed LOCAL `spike-supabase` compose project only
> (reconcile mode, never `--fresh`), against the **built** Activity A artifacts
> — `supabase/migrations/*.sql` + `supabase/seed.sql` applied by the committed
> `supabase/verify/lib.sh` helpers. Never :5433, never :5434, no hosted project.
>
> **Prerequisites: all shipped, none stubbed.** Unlike this activity's first
> three ledgers, nothing here is authored ahead of its dependency. Activity A's
> schema + `redeem()` are merged; Cards 2/3/5/6 (`marketing/sync/*.js`,
> `marketing/submit-machine.js`, `lib/xstate.umd.min.js`) are merged to `dev`
> at `8123f76`. **These spikes import the shipped modules rather than
> re-implementing them** — a spike that proved a re-implementation would prove
> the wrong thing, because this card's job is to *extend* those exact files.

## The goal, and which legs need a spike

The card (roadmap Activity B, authored at morning triage 2026-09-05 — ledger
T-53, decisions 166/167): replicate each campaign's `requires_online` flag to
devices so the §8 refusal **arms on real data**, and own **F-2** (the
unknown-code write puts a 64-hex `token_hash` into `scan_attempts.code_id
uuid not null`). done_when: a `requires_online=true` campaign's code scanned
offline shows "can't verify — try again" with NO override even for an
entitlement holder (branch-3 e2e flips from seam-injected to real data); an
unknown-code override lands without poisoning the push queue.

🛑 The card is **REQUIRED BEFORE any real campaign is provisioned** (decision
166's rider); close-bar leg 3 / Q-KR1 cannot be attested until it lands. That
is why the spikes come first: being wrong about the replication mechanism here
costs a night, and the two failure surfaces both arm at the same moment.

What is already proven and NOT re-spiked here: the bounded, keyset-checkpointed
pull mechanism itself (`rxdb-pull-replica`, GAP-1 validated in run 20260905),
the device-owned push with GAP-1's two belts
(`scan-attempts-push-conflict`), `authenticated`'s SELECT grant + `using (true)`
policy on `public.campaigns` (Activity A verify 01/02 — the flag is *readable*;
whether it *replicates* is a different question), and the shipped machine's §8
guard `overrideAvailable = canOverride && !requiresOnline` (Card 6 conformance
seq 10, 18/18 — guarded and pinned **wherever the flag is known**).

What is NOT yet proven — the falsifiable premises of THIS card:

1. **A mechanism carries the flag AND a change to it.** The roadmap names two
   candidates without choosing ("a campaigns replica, or embed the flag in the
   codes pull"). Landing the flag on initial sync is the easy half; the half
   that decides the design is whether a **flip** of `requires_online` reaches a
   device whose replica is already checkpointed. A mechanism that lands the
   flag once and then goes permanently stale arms the refusal *wrong* — the
   campaign the operator downgrades keeps refusing, the one they upgrade stays
   overridable — which is worse than today's honest unknown→false.
2. **The refusal actually arms on real data.** Card 6 built the §8 refusal
   behind an injectable policy seam (`setCampaignPolicy`) and pinned it with a
   seam-injected e2e. Whether the shipped strict machine refuses when the
   policy is read from a **real replicated row** — and whether it demonstrably
   does NOT today — is this card's done_when, and is not proven by the seam
   test.
3. **F-2's landing path does not poison the queue.** The claim in the card is
   an inference from reading two files (`submit-flow.js:240` writes
   `code_id: SUBMIT_CTX.code_id || SUBMIT_CTX.token_hash`;
   `scan_attempts.code_id` is `uuid not null`; `push-replication.js` throws on
   any non-200 and RxDB retries forever). It has never been *run*. If the
   handler happens to drop the row instead of blocking, the fix is a different
   fix; if it blocks, the blast radius (does one poison row strand every
   legitimate attempt behind it?) is the number the card's guard must clear.

Not spiked, stated: the §16 provisioning UI that *sets* `requires_online` at
campaign creation is Activity A/E surface, not this card. The
`embeddedOffer` → `unknownCode` policy mapping (`submit-flow.js:304-309`) means
a customer not yet in the offers replica is policy-unknown **by construction**
even after this card lands — an embedded high-value offer stays overridable.
That is a real residual, recorded as GAP-2 below rather than spiked, because it
is Activity E's `identity-code-and-qr` payload question (does the embedded
offer carry the flag?), not this card's replication question.

## Spike: flag-replicates-and-flips

- proves: on the BUILT Activity A schema, which of the two candidate mechanisms
  delivers `requires_online` to an offline device **and re-delivers it after a
  flip**. Enumerates, as a set and not a sample (B-216): the columns
  `authenticated` can actually read on `public.campaigns`; the
  `supabase_realtime` publication membership (a campaigns replica with no
  publication row gets no nudge); whether the SHIPPED `makePullHandler` can
  serve `campaigns` unchanged; and — the decisive leg — for EACH mechanism,
  what a running, already-checkpointed replica sees after
  `update campaigns set requires_online = ...`. Falsified if neither mechanism
  carries the flip, or if the winner needs a change to `marketing/sync/` that
  breaks the codes replica.
- plan: source `supabase/verify/lib.sh`; substrate up (reconcile); `reset_bare`
  + `apply_all`. Then a Node client importing the SHIPPED
  `marketing/sync/pull-replication.js` runs both mechanisms side by side against
  PostgREST as a device JWT: (A) a `campaigns` pull replica via
  `replicateRxCollection`, (B) the codes pull widened with the PostgREST FK
  embed `campaigns(requires_online)`. Both reach steady state; then flip
  campaign …0002's flag via psql and observe each replica for a bounded window.
  Print every pull URL and the row deltas. The spike applies whatever extra DDL
  a working mechanism needs **inside the spike** (never to the committed
  migrations) and reports it as a migration the card owes.
- script: .night-crew/spikes/activity-b-offline-first-replica/requires-online-replication/01-flag-replicates-and-flips.sh

## Spike: refusal-arms-on-real-data

- proves: the done_when's first half, against the shipped artifacts and not a
  mock. With the flag in a local replica (spike 1's winning mechanism), the
  SHIPPED `marketing/submit-machine.js` running on the SHIPPED vendored
  `lib/xstate.umd.min.js` — the same wiring `tests/machine/run-conformance.mjs`
  gates — refuses the override for the seeded HIGH code (…0005 → campaign
  …0002, `requires_online=true`) while offline **with** `canOverride: true`, and
  offers it for a LOW code (…0001 → campaign …0001, `false`). The leg that
  makes it a real proof rather than a restatement of conformance seq 10 is the
  **negative**: the identical run with today's shipped policy source (none —
  `CAMPAIGN_POLICY = null`, `policyFor` returns false) must show BOTH codes
  overridable. That demonstrates "the refusal is unreachable today" instead of
  asserting it, and is what the branch-3 e2e converts from seam-injected to
  real-data. Assumes spike 1 chose a mechanism; stubs nothing.
- plan: substrate up + built schema (as spike 1); pull a real replica; build the
  policy lookup as a function of the REPLICA (the shape `setCampaignPolicy`
  will be handed), not of a literal; drive the shipped machine through
  `SCAN → QR_DECODED → RESOLVED{offerReady, requiresOnline} → CONN_DOWN →
  OVERRIDE_REQUEST` for both codes under both policy sources (4 runs), and
  assert `flags().overrideAvailable` and the resulting scan state each time.
  Machine constructed in `mode: 'throw'` so an undeclared pair reds the spike
  rather than being modeled away.
- script: .night-crew/spikes/activity-b-offline-first-replica/requires-online-replication/02-refusal-arms-on-real-data.sh

## Spike: f2-push-poison-and-guard

- proves: F-2, measured rather than inferred. (a) The unknown-code override
  write's `code_id = token_hash` (64 hex) draws a real non-success status from
  the real PostgREST — enumerated per endpoint (`/rpc/redeem`, whose `p_code` is
  `uuid`, and `/scan_attempts`, whose `code_id` is `uuid not null`), because
  which one fails first decides where the guard goes. (b) The blast radius: with
  that row queued first, a **legitimate** attempt behind it never lands across a
  bounded number of RxDB retry cycles — head-of-line poisoning, not a dropped
  row. (c) A guard clears it: the legitimate attempt lands while the unverified
  attempt is handled honestly. Falsified if the poison does not reproduce (the
  card's premise would be wrong and its guard unnecessary) or if the guard
  clears the queue only by silently discarding the audit-flagged attempt —
  decision 166 ratified unknown→false *because* every such attempt is
  audit-flagged, so a guard that strands the audit row on-device retroactively
  falsifies the ratification's own reasoning.
- plan: substrate up + built schema; a Node client importing the SHIPPED
  `marketing/sync/push-replication.js` (`enqueueAttempt`, `makePushHandler`,
  `startScanAttemptsReplica`) with the real `fetch` against PostgREST as a
  device JWT. Leg (a): enqueue one attempt with a 64-hex `code_id` exactly as
  `submit-flow.js:240` composes it, call the handler directly, record both HTTP
  statuses. Leg (b): queue the poison row, then a legitimate attempt for seeded
  code …0001; run the live push replica with a short `retryTime` for a bounded
  window; assert the legitimate attempt is still `pending` and its landing
  request never appears in the enumerated request log. Leg (c): apply the
  candidate guard — `code_id` nullable + a `token_hash` column + a check
  constraint, and a handler that skips `redeem()` for an unverified attempt and
  lands it directly (the alternative, skip-until-arbitration, is enumerated and
  rejected in the same run because it strands the audit row) — then re-run and
  assert the legitimate attempt lands AND the unverified attempt's audit row
  reaches the server.
- script: .night-crew/spikes/activity-b-offline-first-replica/requires-online-replication/03-f2-push-poison-and-guard.sh

## Verdicts (run 2026-09-05, hand-run per the no-story-map convention)

- **flag-replicates-and-flips: passed** — exit 0. GREEN on the first run; one
  evidence line corrected afterwards (correction 1) and re-run green.
- **refusal-arms-on-real-data: passed** — exit 0, first run, no corrections.
- **f2-push-poison-and-guard: passed** — exit 0. RED on the first run at leg
  (c2) for a harness reason (correction 2); green after, legs (a)/(b) unchanged
  and green on both runs.

### What the spikes decided

**1 — the mechanism is a campaigns replica; "embed the flag in the codes pull"
is REJECTED.** Both mechanisms land the flag on initial sync, so the initial
leg decides nothing. The flip decides it: with `requires_online` changed and
`updated_at` stamped, the campaigns replica delivered `true` on the next
RESYNC, while the codes-embed replica still read `false` — and
`max(codes.updated_at)` was **byte-identical before and after the campaign
write** (`2026-09-05 09:24:41.989173+00` both sides). The embedded flag is
re-read only when the CODE row's own `updated_at` advances, so a campaign whose
policy changes while its codes sit still never re-delivers — the device would
keep offering the offline override on a campaign the operator just made
online-only. Enumerated, not sampled: 3 pull requests per mechanism, each URL
printed with its checkpoint.

**2 — the refusal arms, and is provably dead today.** Four runs of the SHIPPED
`marketing/submit-machine.js` on the SHIPPED `lib/xstate.umd.min.js`
(sha256 `e7f04e1f…`), `mode: 'throw'`, policy read from a real replica:

| policy source | code | requiresOnline | overrideAvailable | after OVERRIDE_REQUEST |
|---|---|---|---|---|
| replica | HIGH (…0005, campaign …0002) | `true` | **false** | `blockedOffline` |
| replica | LOW (…0001, campaign …0001) | `false` | true | `overrideConfirm` |
| none | HIGH | `false` | **true** | `overrideConfirm` |
| none | LOW | `false` | true | `overrideConfirm` |

Rows 3–4 are the point: today the $40 catering-credit code is overridable
offline exactly like the $2 one. **Zero undeclared (state,event) pairs and the
actor alive across all four runs** — feeding `requiresOnline` from a replica
adds no new machine surface, so Card 6's 460-pair strictness proof still holds.

**3 — F-2 is real, it is head-of-line, and one guard survives.** `/rpc/redeem`
refuses **first** (HTTP 400, Postgres `22P02` on `p_code uuid`), so the guard
belongs **before** the redeem call, not at the landing insert (which also 400s).
Blast radius, measured: with the poison row queued first, **12 redeem attempts
over ~12 retry cycles produced 0 landing requests** and the legitimate attempt
behind it never left `pending` — one unknown-code override strands every later
redemption on that device. Two guards were run, not argued:
*skip-until-arbitration* drains the queue (legit lands) but the audit-flagged
attempt never reaches the server — **rejected**, because decision 166 ratified
unknown→false precisely on the strength of every such attempt being
audit-flagged; *distinct landing path* (`code_id` nullable + `token_hash` +
a check constraint) lands both, storing the audit row as
`code_id=(null), token_hash=ffffffff0123…, offline_override=t, unverified_code=t`.

### Build-facts the card inherits

1. `buildPullUrl()` unconditionally appends `expires_at=gt.<windowIso>`;
   `campaigns` has no `expires_at`, so the shipped handler answers **HTTP 400**
   on that table. The card owes the module an **optional bound** — and only
   that: the GAP-1 keyset checkpoint (`keysetPredicate`) carried over to
   `campaigns` unchanged and is proven working in both spikes 1 and 2.
2. **`campaigns` has no touch trigger** — enumerated. A plain
   `update campaigns set requires_online = …` does not advance `updated_at` and
   is therefore **invisible to any checkpointed replica**. The card owes the
   write path an explicit `updated_at = now()` (the rule decision 163 already
   put on `redeem()`) or a touch trigger. Both mechanisms were blind to the
   unstamped write — this is not mechanism-specific.
3. **`campaigns` is not in the `supabase_realtime` publication** — enumerated
   (`codes` is; `campaigns` is not). A pull replica does **not** poll: 3s after
   a stamped write with no RESYNC, the replica still read the old value. So the
   card must either add `campaigns` to the publication or fan the codes
   channel's RESYNC into the campaigns replica.
4. `authenticated` holds **SELECT only** on `campaigns`, with
   `using (true)` — no RLS work needed; the flag is readable today.
5. A code is **single use**, and the shipped push handler's GAP-1 belt is
   working as designed: a second `redeem()` on a burned code answers
   `already_used`, and with no codes pull replica to name the winner the handler
   **blocks and retries rather than guessing**. Any harness that reuses one code
   across legs will look like a product failure and is not one (correction 2).
6. **Open for the card, not settled here:** the server-side `status` taxonomy
   for an unverified override. The spike landed it `accepted` +
   `offline_override` + `unverified_code` (already distinguishable from a real
   accept, and card 3's slate PARK line said "no new terminal status"), but
   §9/§19 should be re-read before building on it.

## Corrections

- **1 — spike 1's staleness evidence measured the wrong thing.** The line
  supporting "mechanism B is permanently stale" originally reported *"codes rows
  touched in the last 10s: true"* — which was true merely because `apply_all`
  had just seeded them, and therefore read as evidence for the OPPOSITE
  conclusion. Changed to capture `max(codes.updated_at)` immediately before and
  after the campaign flip and report both. The conclusion did not change; the
  evidence now actually supports it (`before == after`, `moved=false`).
- **2 — spike 3 was red on its first run for a harness reason that looked
  exactly like a product finding.** All three legs shared one seeded code
  (…0001). A code is single use, so by leg (c2) `redeem()` answered
  `already_used`; the shipped handler then correctly blocked awaiting a winner
  the spike could never supply (it runs no codes pull replica) and the
  legitimate attempt stayed `pending` — presenting as "the guard does not clear
  the queue". Fixed by minting a fresh live code per leg via `gen_random_uuid()`.
  **No product code was changed** and no shipped behaviour was found wanting;
  the blocking is GAP-1 belt 2 doing its job. Recorded as build-fact 5 so the
  card's own harness does not repeat it.

## Review

- signed: operator, 2026-09-05 — covers 2 correction(s)

## Comebacks

- gap: GAP-1 — F-2 head-of-line poison, **confirmed by measurement**: an
  unknown-code offline override enqueues `code_id = <64-hex token_hash>`
  (`submit-flow.js:240`), `/rpc/redeem` answers HTTP 400 `22P02` on
  `p_code uuid`, the shipped push handler throws, and RxDB retries the batch
  forever — 12 redeem attempts, 0 landing requests, and the legitimate attempt
  queued behind it never leaves `pending`. Every later redemption on that device
  is stranded until the queue is cleared by hand. Fix lands with this card's
  guard; that card owes ONE validation run re-executing spike 03 against the
  shipped guard (found 2026-09-05, /nc-spike-open).
  - validated: GAP-1 — run 20260906, card `requires-online-replication` (branch
    `wo-requires-online-replication`). The guard shipped INSIDE
    `marketing/sync/push-replication.js`'s `makePushHandler` (before the
    redeem call, `unverified_code` the discriminator) with the landing-path
    DDL as committed migration `20260906000200` (`code_id` nullable +
    `token_hash` + check `scan_attempts_names_a_code`), and the owed
    validation re-executed spike 03 against it wrapper-free
    (`marketing/sync/harness/f2-run.sh`). Red first, on the PRE-change tree:
    the shipped handler redeem-firsted the 64-hex `code_id` 10× (HTTP 400
    `22P02`), 0 landing requests, both attempts stuck `pending` — the
    head-of-line poison reproduced (`card1-red.log`, EXIT=1). Then green
    against the shipped guard: **0 redeem calls for the unverified attempt,
    exactly 1 for the legitimate one behind it, both landed, +2 server rows,
    audit row stored `code_id=(null) | token_hash=<64-hex> | override=t |
    unverified=t | status=accepted`** — the §9/§19 taxonomy unchanged
    (`card1-f2-harness.log`, EXIT=0; red-unflagged mode EXIT=1 proves the
    assertions catch the defect class). Build-fact 5 honored: the harness
    mints a fresh live code per leg; belt-2's block-on-unknown-winner
    untouched (push-run.sh green, `card1-regression-c3.log`).
- gap: GAP-2 — a campaign policy write that does not advance
  `campaigns.updated_at` is **invisible to every checkpointed replica**, and
  `campaigns` carries no touch trigger (enumerated). Nothing is broken today
  because no campaign write path exists; it arms the moment the §16 provisioning
  surface lands, and it fails **silently** — the operator sees the flag change
  in the UI and no device ever hears about it. Fix lands with whichever card
  builds the campaign write path (a touch trigger, or an explicit
  `updated_at = now()` on every write); that card owes ONE validation run
  re-executing spike 01's flip #1 leg, which must stop being blind (found
  2026-09-05, /nc-spike-open).
- gap: GAP-3 — the embedded-offer path stays policy-unknown even after this
  card. `submit-flow.js:304-309` maps `embeddedOffer` → `unknownCode` and never
  carries a `campaign_id`, so a customer not yet in the offers replica whose QR
  embeds a **high-value** offer remains offline-overridable — the §8 refusal
  cannot arm for them by construction. Deliberately NOT spiked here: whether the
  embedded offer can carry the flag is Activity E's `identity-code-and-qr`
  payload question, not this card's replication question. Recorded so the
  residual is not mistaken for coverage when Q-KR1 is attested (found
  2026-09-05, /nc-spike-open).
