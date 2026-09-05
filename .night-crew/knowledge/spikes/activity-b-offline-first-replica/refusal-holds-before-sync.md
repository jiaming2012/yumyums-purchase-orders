# Spikes — refusal-holds-before-sync

Activity: Activity B — Offline-first replica (RxDB ↔ Supabase)

> This target repo has no `usm/roadmap.txt` story map (the layout the
> `night-crew spikes gate/run` verbs read), so those verbs cannot drive here —
> the established hand-run convention (activity-5, Activities A + G, and this
> activity's four sibling ledgers). This ledger and the runnable scripts that
> ARE the verdict (B-345) are authored to the skill's paths anyway.
>
> **Substrate:** the committed LOCAL `spike-supabase` compose project only
> (reconcile mode, never `--fresh`), against the **built** artifacts —
> `supabase/migrations/*.sql` (now including run 20260906's `20260906000100`
> touch-trigger/publication and `20260906000200` F-2 landing-path migrations)
> + `supabase/seed.sql`, applied by the committed `supabase/verify/lib.sh`
> helpers. Never :5433, never :5434, no hosted project.
>
> **Prerequisites: all shipped, none stubbed.** The full Activity B surface is
> merged to `dev` (runs 20260904–20260906): the pull mechanism with the GAP-1
> keyset checkpoint, the campaigns replica + `createCampaignPolicySource`, the
> push module with the F-2 guard, and the strict XState submit machine. These
> spikes import the shipped modules rather than re-implementing them — this
> card's job is to *extend* those exact files.

## The goal, and which legs need a spike

The card (roadmap Activity B, authored at morning triage 2026-09-05 — ledger
T-54, decision 170, the B-432 unblocking card): `requires-online-replication`
armed the §8 refusal on real data but left it **failing open during the window
before the campaigns replica has delivered** — `createCampaignPolicySource`
answers `null` for any campaign not yet in its Map, `submit-flow.js` coerces
`null → false`, and the three replicas start together with `SCAN_STATE.synced`
display-only. A known, replicated, entitlement-bearing `requires_online=true`
code whose *campaign* has not arrived is offline-overridable. 🛑 Decision 171:
this is NOT the ratified unknown→false default (decision 166) — that covers
genuinely-unknown *codes*; the ratification never considered a known code with
an unreplicated campaign.

done_when: with codes + offers replicated and the campaigns replica empty or
erroring, a `requires_online=true` code scanned offline shows "can't verify —
try again" with NO override for an entitlement holder — proven by the shipped
branch-3 e2e with the `campaigns:` seed removed (the exact triage
reproduction), red before, green after; and the campaigns-replica failure path
is distinguishable from a genuinely-unknown campaign in the attempt record.

What is already proven and NOT re-spiked here: the campaigns replica delivers
the flag and its flip (sibling spike 01), the shipped machine refuses on
`requiresOnline:true` with zero new (state,event) pairs when the flag is
*known* (sibling spike 02 + Card 6's 460-pair strictness proof), and the F-2
distinct landing path lands unverified attempts without poisoning the queue
(sibling spike 03, validated against the shipped guard in run 20260906).

Also NOT spiked, stated: the third candidate shape ("carry the flag on the
code row") is priced by the sibling ledger's evidence — the codes-embed
alternative is CLOSED for flip-staleness (`max(codes.updated_at)`
byte-identical across a campaign write), and reopening it would need a
campaign→codes touch-cascade whose write amplification (every code of a
flipped campaign re-pulls on every device) is enumerable at card time without
a new spike. The branch-3 e2e flip (red with the `campaigns:` seed removed,
green after the fix) is the card's regression test, not a spike — the triage
already reproduced the red half with zero production code mutated.

What is NOT yet proven — the falsifiable premises of THIS card:

1. **The window is detectable.** Every candidate shape ("gate override on
   campaigns-replica readiness", "fail closed for unresolved campaigns")
   presumes the client can tell, at policy-lookup time, *replica ready* from
   *still delivering* from *pull erroring*. `scan-page.js` already calls
   `awaitInitialReplication()` (display-only `SCAN_STATE.synced`), but the
   shipped pull handler THROWS on any non-200 — and whether
   `awaitInitialReplication()` then hangs, whether `error$` emits something
   attributable, whether recovery resolves it without restarting the replica,
   and whether the policy source's Map is actually populated at the moment
   readiness fires (a subscription race) have never been *run*.
2. **Fail-closed at the policy seam refuses without collateral.** The fix
   shape must (a) refuse the override for a KNOWN code whose campaign is
   unresolved, (b) NOT delete F2's decided affordance for genuinely-unknown
   codes (decision 166 ratified unknown→false; an over-broad fail-closed
   silently kills the offline path decision 166 exists to permit), (c) stop
   over-refusing the moment the replica is ready, and (d) add ZERO new
   (state,event) pairs to the strict machine. The B-432 fail-open must be
   demonstrated in the same harness, not asserted.
3. **The discriminator can land.** done_when's second half wants the
   campaigns-replica failure path distinguishable from a genuinely-unknown
   campaign *in the attempt record*. Today that field would be silently
   dropped TWICE — `enqueueAttempt` destructures a fixed field list, and the
   push handler's landing insert is an explicit whitelist — and if the card
   extends the landing body before the server column exists, PostgREST's
   unknown-column refusal is the same throw-retry head-of-line poison class
   F-2 measured (12 redeem attempts, 0 landings). Which layers drop it, what
   the pre-migration server answers, and whether the extended row lands and
   reads back distinguishable post-DDL must be measured, not inferred.

## Spike: window-is-detectable

- proves: on the SHIPPED `startCampaignsReplica` handle (the raw
  `replicateRxCollection` state), the three window states are distinguishable
  at policy-lookup time. Enumerates, as a set and not a sample (B-216): the
  handle's observable surface (`error$`, `active$`, `remoteEvents$`,
  `awaitInitialReplication`, `awaitInSync`, `reSync`, `cancel` — typeof each)
  on the QA rxdb, with the vendored `vendor/rxdb.bundle.js` grepped for the
  same names in the .sh; then measures: with the campaigns pull answering
  HTTP 503, `awaitInitialReplication()` stays pending while `error$` emits
  (erroring ≠ in-flight); a late `error$` subscriber sees no replay (the card
  must latch); after the endpoint recovers, the SAME handle resolves initial
  replication without a restart; and at the resolve moment, the shipped
  `createCampaignPolicySource` Map's population is measured immediately vs
  after a settle tick (the subscription race the readiness gate must respect).
  Falsified if no signal distinguishes erroring from in-flight, or recovery
  requires tearing the replica down.
- plan: source `supabase/verify/lib.sh`; substrate up (reconcile);
  `reset_bare` + `apply_all`. A Node client importing the SHIPPED
  `marketing/sync/replicas.js` starts `startCampaignsReplica` with a gated
  fetchImpl (synthetic 503 until released, then passthrough to PostgREST as a
  device JWT). Every signal observation is printed with its timestamp; the
  verdict is the script's exit status.
- script: .night-crew/spikes/activity-b-offline-first-replica/refusal-holds-before-sync/01-window-is-detectable.sh

## Spike: refusal-holds-during-window

- proves: the done_when's first half, at the seam, against the shipped
  machine — plus the F2 non-regression the fix could silently break. With
  codes + offers replicated for real (shipped replicas, built schema) and the
  campaigns replica erroring (503, never delivered), a PROTOTYPE fail-closed
  policy source — known campaign_id + replica not ready → `{requiresOnline:
  true}`; campaignId null (genuinely-unknown code) → null, preserving decision
  166 — drives the SHIPPED `marketing/submit-machine.js` on the SHIPPED
  `lib/xstate.umd.min.js`, `mode: 'throw'`, `canOverride: true`, through five
  runs: (1) window + HIGH known code → override REFUSED (`overrideAvailable
  false`, OVERRIDE_REQUEST a no-op in `blockedOffline`); (2) window +
  genuinely-unknown code → override OFFERED with the F2 unverified warning
  (166 survives); (3) window + HIGH under the SHIPPED policyFor semantics
  (empty Map → null → false) → override OFFERED — the B-432 fail-open
  DEMONSTRATED, not asserted; (4)+(5) after recovery + readiness settle: HIGH
  refused, LOW offered (no over-refusal once the replica is ready). Zero
  undeclared (state,event) pairs across all runs — the fix is expressible
  entirely at the policy seam. Falsified if the refusal needs new machine
  surface, if fail-closed eats F2, or if the over-refusal survives readiness.
- plan: substrate as spike 01. Node client: shipped codes + offers replicas
  pulled to sync (seeded HIGH …0005 → campaign …0002 `requires_online=true`,
  LOW …0001 → campaign …0001 `false`); shipped campaigns replica behind the
  gated fetch; the prototype policy wraps the SHIPPED
  `createCampaignPolicySource` + a readiness latch fed by
  `awaitInitialReplication()` + settle (spike 01's race finding). The
  `policyFor` coercion is copied verbatim from `submit-flow.js`.
- script: .night-crew/spikes/activity-b-offline-first-replica/refusal-holds-before-sync/02-refusal-holds-during-window.sh

## Spike: discriminator-lands-without-poison

- proves: premise 3, measured per layer. (a) The shipped local write path
  drops a `policy_unresolved` discriminator: `enqueueAttempt` (destructured
  field list) and a direct insert against the shipped `SCAN_ATTEMPTS_SCHEMA`
  under ajv validation — each measured, deciding whether the card owes a
  schema version bump + enqueue plumbing. (b) The shipped push handler
  processes an EXTENDED local row without breaking (forward compat), while
  its whitelisted landing body provably never sends the field. (c) The card's
  future landing body (the F-2 `land-unverified` shape + the discriminator)
  against the PRE-migration server draws PostgREST's unknown-column status —
  enumerated, because a non-2xx there is the F-2 throw-retry poison class:
  the migration MUST land before any client sends the field. (d) After
  spike-local DDL (`alter table … add column policy_unresolved boolean not
  null default false` — applied inside the spike, never to the committed
  migrations, reported as the migration the card owes), the extended body
  lands 201, the F-2 check constraint holds, and the server rows read back
  distinguishable: replica-failure override `unverified_code=t,
  policy_unresolved=t` vs genuinely-unknown-code override `unverified_code=t,
  policy_unresolved=f`, both `status='accepted'` — no new terminal status
  (§9/§19 re-read, run 20260906). Falsified if the discriminator cannot reach
  the server without a new status or without the poison class, or if the
  extended local row breaks the shipped handler.
- plan: substrate as spike 01. The .sh orchestrates: node PRE leg (drop
  measurements + pre-DDL landing status) → psql DDL + `notify pgrst, 'reload
  schema'` → node POST leg (extended body lands; shipped handler drains a
  queue holding one discriminated unverified attempt + one legitimate redeem
  attempt on a freshly-minted live code — build-fact 5: never reuse a seeded
  code) → psql reads the rows back and prints the distinguishability table.
- script: .night-crew/spikes/activity-b-offline-first-replica/refusal-holds-before-sync/03-discriminator-lands-without-poison.sh

## Verdicts (run 2026-09-05, hand-run per the no-story-map convention)

- **window-is-detectable: passed** — exit 0, first run, no corrections.
- **refusal-holds-during-window: passed** — exit 0, first run, no corrections.
- **discriminator-lands-without-poison: passed** — exit 0, first run, no
  corrections.

### What the spikes decided

**1 — the window is fully observable on the shipped handle; nothing needs a
restart.** With the campaigns pull answering 503, `error$` emitted on the
first attempt (t+145ms) and each retry cycle while `awaitInitialReplication()`
stayed pending; the emission carries the thrown handler message —
`"[marketing-sync] pull campaigns answered HTTP 503"` inside
`.parameters.errors` — so failure is **attributable for free** (the
distinguishability record can name the HTTP status without wrapping the
handler). After recovery the SAME handle resolved initial replication within
one retry cycle (t+10.2s, gate released t+7s) — no teardown, no re-create.
The three states are disjoint: erroring = `error$` emitting + promise
pending; in-flight = `error$` silent + promise pending; ready = promise
resolved. Enumerated surface: `error$`/`active$`/`remoteEvents$` observables +
`awaitInitialReplication`/`awaitInSync`/`reSync`/`cancel` functions, present
on the QA rxdb AND grepped present in `vendor/rxdb.bundle.js`.

**2 — fail-closed at the policy seam is the fix shape, and it costs zero
machine surface.** Five runs of the SHIPPED `marketing/submit-machine.js` on
the SHIPPED `lib/xstate.umd.min.js`, `mode: 'throw'`, `canOverride: true`,
policy from real replicas over the built schema:

| policy source | window state | code | requiresOnline | overrideAvailable | after OVERRIDE_REQUEST |
|---|---|---|---|---|---|
| prototype fail-closed | erroring | HIGH (…0005, known) | **true** | **false** | `blockedOffline` |
| prototype fail-closed | erroring | genuinely-unknown | `false` | true | `overrideConfirm` + unverified warning |
| SHIPPED null→false | erroring | HIGH (…0005, known) | `false` | **true** | `overrideConfirm` |
| prototype fail-closed | ready | HIGH | `true` | `false` | `blockedOffline` |
| prototype fail-closed | ready | LOW (…0001) | `false` | `true` | `overrideConfirm` |

Row 3 is B-432 **demonstrated**: the $40 catering-credit code, present in the
codes replica, offline-overridable because its campaign had not arrived. Rows
1/4/5: the prototype refuses through the window and stops refusing the moment
the replica is ready. Row 2: decision 166's ratified F2 affordance survives —
fail-closed keys on `campaignId != null`, so the genuinely-unknown code keeps
its override + unverified warning. **Zero undeclared (state,event) pairs, all
actors alive** — the fix lives entirely in the policy answer; Card 6's
460-pair strictness proof is untouched.

**3 — the discriminator lands, and every drop layer is now named.**
`policy_unresolved` is dropped at the `enqueueAttempt` destructure (measured:
stored row lacks the field), REJECTED by the shipped `SCAN_ATTEMPTS_SCHEMA`
under ajv (error VD2 — the card owes a schema version bump + device
migration), and never sent by the push handler's whitelisted landing body
(measured in the drain leg: the c3 row landed `policy_unresolved=f` while the
local row held `true`). Pre-migration, the extended landing body draws **HTTP
400 `PGRST204`** ("Could not find the 'policy_unresolved' column") — the same
throw-retry head-of-line poison class F-2 measured, so **the server migration
must land before any client sends the field**. Post-DDL: discriminated
override `a1… | accepted | t | t | t`, genuinely-unknown-campaign control
`b2… | accepted | t | t | f`, legitimate redeem behind the discriminated
attempt landed `accepted` with exactly one `redeem` call — no head-of-line,
the F-2 check constraint (`scan_attempts_names_a_code`) holds, and no new
terminal status (§9/§19 taxonomy unchanged).

### Build-facts the card inherits

1. **The card owes an error latch.** `error$` does NOT replay to late
   subscribers (measured: 0 emissions to a subscriber attached mid-error) —
   the policy source must subscribe before the replica starts and hold the
   last error itself.
2. **The readiness→Map race did not bite on memory storage** — the shipped
   `createCampaignPolicySource` Map held both campaigns AT the
   `awaitInitialReplication` resolve tick (2 at tick, 2 after settle). The
   browser runs Dexie, not memory storage; the 150ms settle in the latch is
   cheap insurance, but the tick measurement here says a bare-promise latch
   would have worked.
3. **The latch alone does not close ALL of B-432.** The spiked prototype maps
   *ready + campaign absent from the Map* to the shipped null→false — which is
   B-432's "new campaign whose codes arrive first" window still open.
   Candidate shape (b) — fail closed for any KNOWN code (`campaignId != null`)
   whose campaign is unresolved, readiness aside — subsumes shape (a) and
   closes it, and it cannot touch decision 166: a known code always names a
   campaign, so "genuinely unknown" (campaignId null) never enters the
   fail-closed arm. The mechanics are identical to the spiked seam; the card
   states its predicate and cites this.
4. **The migration the card owes:**
   `alter table public.scan_attempts add column policy_unresolved boolean not
   null default false` — as a NEW numbered `supabase/migrations/` file, and it
   must merge/deploy BEFORE any client writes the field (build-fact 3's
   poison class, sequencing not optional). The field name is the spike's
   candidate; the card may rename it, the shape is what is proven.
5. **The client-side cost is three named edits, not one:** enqueue plumbing
   (the destructure drops extra fields), a `SCAN_ATTEMPTS_SCHEMA` version
   bump + device migration strategy (ajv rejects the field at v0), and one
   landing-body line (the whitelist). The shipped handler is otherwise
   forward-compatible with the extended local row (drain leg green).
6. **Open for the card, not settled here:** the `blockedOffline`
   `requires-online` branch copy reads "High-value offer: online verification
   is required" — misleading when the policy is *unresolved* rather than
   *known true*. Whether the unresolved case gets its own `data-branch` and
   copy ("can't verify this campaign yet — syncing") is a UI call the card
   makes against `docs/ui-design-rules.md`; the branch-3 e2e's expected
   `data-branch` value depends on it.

## Corrections

- none — no agent-reached corrections

## Comebacks

- gap: GAP-1 — B-432, the pre-sync fail-open, **confirmed by measurement**
  (spike 02 row 3): with codes + offers in sync and the campaigns replica
  erroring, the shipped policy path (`createCampaignPolicySource` Map empty →
  null; `submit-flow.js` `policyFor` null → false) offers the offline
  override on the seeded `requires_online=true` $40 code to a `canOverride`
  holder — `overrideAvailable=true`, OVERRIDE_REQUEST → `overrideConfirm`.
  Fix lands with card `refusal-holds-before-sync`; that card owes ONE
  validation run re-executing spike 02 with the prototype policy replaced by
  the SHIPPED policy source (the f2-run.sh precedent: wrapper-free re-run of
  the spike against the shipped guard), plus its own branch-3 e2e flip
  (red with the `campaigns:` seed removed, green after — the triage
  reproduction). Build-fact 3 binds the validation: the codes-arrive-first
  sub-case must be covered or explicitly carried (found 2026-09-05,
  /nc-spike).
