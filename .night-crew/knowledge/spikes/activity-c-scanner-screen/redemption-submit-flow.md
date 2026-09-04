# Spikes — redemption-submit-flow

Activity: Activity C — The scanner screen (staff redemption at the window)

> No `usm/roadmap.txt` on this target — hand-run convention (full preamble in
> `../activity-b-offline-first-replica/rxdb-pull-replica.md`). No substrate
> needed: the machine under test is pure client logic; server verdicts are
> injected as §19.3 boundary events, exactly as the real screen receives them.

## The goal, and which legs need a spike

This goal carries the spike the roadmap round **explicitly commissioned**
("decided by a spike near Activity C", operator's call): **XState vs a
hand-rolled parallel-region machine** for the §19.1 client statechart, proved
against the decided forks F1/F2/F3/F6 in HQ's vanilla-JS no-build context.
Adopting XState is a new client dependency in a deliberately no-framework app;
the fork is decided on evidence from the real behavioral contract, not in the
abstract.

The shared instrument — what makes the two spikes comparable rather than two
demos — is **one conformance suite** both machines must pass, its sequences
transcribed from the §19.4 acceptance criteria (given/when/then, F1 F2 F3 F6
plus the §8 `requires_online` branch and the `stale` routing). A machine
passing its own hand-picked tests proves nothing; two machines passing the
SAME externally-derived suite is the comparison. The suite drives each machine
through an adapter (send events / read region states / read emitted effects)
and asserts, per sequence, the resulting configuration and the effect log
(e.g. F2's override must emit an attempt-write effect carrying
`offline_override=true` AND `unverified_code=true`).

Conformance sequences (both spikes run all of them):

1. **F1-parallel** — mid-scan (`offerReady`), `CONN_DOWN` arrives: connectivity
   region transitions, scan region provably does NOT reset.
2. **F1-gate** — `readyToSubmit` while offline, `SUBMIT` → `offlineGate`,
   never `submitting`.
3. **F1-stale** — `stale` (online, replica not refetched) routes `SUBMIT` to
   `offlineGate` exactly like offline.
4. **F2-no-perm** — offline `unknownCode` without override permission →
   `blockedOffline`; no override path offered.
5. **F2-with-perm** — offline `unknownCode` with permission →
   `overrideConfirm` flagged "neither offer nor prior use verifiable" →
   `OVERRIDE_CONFIRM` → attempt-write effect with `offline_override=true` and
   `unverified_code=true`.
6. **F3-offline** — replica says redeemed + offline → `spentLocally` reject;
   no submit path.
7. **F3-online** — replica says redeemed + online → submit proceeds
   (`submitting`), `SRV_ALREADY_USED` → `alreadyUsed` (server wins, both
   directions exercised).
8. **F6-same-code** — re-scan of the in-session code mid-flow: no-op, state
   unchanged, no new attempt effect; after a terminal result: re-shows that
   result.
9. **F6-different-code** — different code mid-session → prompt to finish the
   current customer; session intact.
10. **§8-high-value** — offline + `requires_online=true` campaign →
    `blockedOffline` with NO override even with permission.

Falsifiable premises per approach:

- **XState:** (a) the §19.1 parallel-region model is expressible in XState v5
  and passes all 10 sequences; (b) XState is usable in HQ's no-build context —
  a single committed file loads in real Chromium as a plain page dependency
  and creates a working actor (a bundler-only library falsifies here
  regardless of how nice the model is). Weight enumerated (file bytes).
- **Hand-rolled:** the same 10 sequences pass against a small dependency-free
  parallel-region machine (regions, guarded transitions, effect emission)
  written to HQ conventions — proving the no-dependency door is genuinely open,
  and pricing it in lines of owned code (enumerated).

Both green → the fork is decidable on recorded evidence (dependency weight vs
owned-code weight, expressiveness friction found while modeling); one red →
decided by falsification. The DECISION itself is not made in this ledger: the
close's extraction record carries the recommendation, and adoption is the
card's design + the operator's slate sign-off (approaches are candidates, not
adoptions — NFR-6).

## Spike: machine-xstate

- proves: premises (a) and (b) above — §19.1 F1/F2/F3/F6 + §8 + stale, modeled
  as an XState v5 parallel machine, passes the shared 10-sequence conformance
  suite in Node; and a single-file XState artifact loads in real Chromium as a
  plain page (spike-local Playwright config, no repo webServer) and runs
  sequence 1 there (the parallel-region core) — the no-build loadability
  proof. Enumerated: xstate dist artifacts + the chosen file's bytes.
- plan: spike-local `package.json`; `npm install xstate` (network →
  could-not-run); `js/machine-xstate.mjs` implements the machine + adapter;
  `js/conformance.mjs` (shared, machine-agnostic) runs the suite; a minimal
  page + spec run the browser leg.
- script: .night-crew/spikes/activity-c-scanner-screen/redemption-submit-flow/01-machine-xstate.sh

## Spike: machine-handrolled

- proves: the hand-rolled premise — the same 10 sequences pass against
  `js/machine-handrolled.mjs`, a dependency-free parallel-region machine in
  plain ES (regions {connectivity, scan}, guarded transitions on context
  {canOverride, requiresOnline, inReplica, locallyRedeemed}, effect log), via
  the IDENTICAL `js/conformance.mjs` — not a hand-picked suite. Enumerated:
  the machine's line count and byte size (the owned-code price the comparison
  needs).
- plan: same directory, same conformance module, no dependencies; runs in Node
  only (it is vanilla by construction — browser loadability is not in
  question for plain ES).
- script: .night-crew/spikes/activity-c-scanner-screen/redemption-submit-flow/02-machine-handrolled.sh

## Spike: machines-equivalent

*(Added by the missing-states deep dive, operator-directed, same sitting.)*

- proves: the two candidates are observably THE SAME machine outside the
  suite — seeded lockstep fuzzing (25-event alphabet, canonical payloads)
  drives both through identical random walks and fails on any divergence in
  region states, gate flags, or emitted effects. Without this, choosing an
  engine silently chooses undocumented behavior: the pre-fix pair diverged in
  114/5000 walks (PROBE_TIMEOUT while `stale`), and code diff found
  session-lifecycle gaps the fuzzer's depth could not reach — both kinds of
  hole are exactly what this spike exists to catch. Deterministic and
  reproducible from the printed walk seed; two seeds so one lucky seed cannot
  green a real divergence.
- plan: `js/lockstep-fuzz.mjs` — mulberry32 PRNG, fresh machine pair per walk
  (random `canOverride`, same for both), compare `{conn, scan, flags,
  effects}` after every event; 20,000 walks × 20 steps × 2 seeds.
- script: .night-crew/spikes/activity-c-scanner-screen/redemption-submit-flow/03-machines-equivalent.sh

## Verdict (run 2026-09-04, hand-run per the no-story-map convention)

- **machine-xstate: passed** — exit 0 (second run; the first was RED on a
  harness/artifact fact, see Corrections — the conformance suite itself passed
  on the first run). XState **v5.32.6** passes all 10 shared sequences; the
  dist enumeration shows real single-file builds, and the convention-fit one —
  `xstate.umd.min.js`, **47,268 bytes** — vendored as ONE classic `<script>`
  (the SortableJS/html5-qrcode pattern) drives a parallel machine in real
  Chromium (`{"a":"y","b":"p"}` — a region transition leaving its sibling
  untouched).
- **machine-handrolled: passed** — exit 0, first run. The IDENTICAL suite
  passes against `js/machine-handrolled.mjs`: **113 lines, 4,794 bytes,
  0 dependencies**.

**Conclusion:** BOTH doors are open — the commissioned fork is decidable on
recorded evidence, and the decision moves to the close's extraction record +
the card's design (candidates, not adoptions — NFR-6). The comparison facts:
- **Weight:** XState adds a 47 KB vendored file + a version to track;
  hand-rolled is 113 owned lines protected by the same conformance suite
  (which is machine-agnostic and survives as a build-time test either way).
- **Expressiveness friction found while modeling:** XState v5 inline guards
  receive only `{context, event}` — a cross-region guard (F1/F3's
  "scan flow guards on connectivity") needs either `stateIn()` registered via
  `setup()` or a connectivity mirror kept in context by entry actions (the
  spike used the mirror). The hand-rolled machine reads the sibling region
  directly. Neither blocked any sequence.
- **What XState buys that the hand-rolled one lacks:** declarative statechart
  (visualizable, XState ecosystem), bubbling semantics for the F6 dedupe
  (parent-level `on` with the innermost handler winning), and a maintained
  library for the §19.1 states the spike deferred (probing/syncing timers).
- **What the suite guarantees either way:** the 10 §19.4 sequences are pinned
  by `js/conformance.mjs`, which the card should inherit as its test — the
  machine choice cannot silently change behavior.

## Corrections

- **Browser-leg artifact fact (fixed, re-run green): Chromium CORS-blocks ES
  module imports from `file://`, so the ESM build cannot be smoke-tested the
  way classic scripts can — and the convention-fit artifact for HQ was the
  UMD build all along.** First run red with
  `Failed to fetch dynamically imported module` while leg (a) had already
  passed. Fixed by vendoring `xstate.umd.min.js` and loading it as a classic
  `<script>` — which is also how HQ actually ships third-party JS (SortableJS,
  and html5-qrcode in the sibling spike), so the correction landed the spike
  CLOSER to the real premise, not further. The suite itself needed no change.

## Amended verdict — the missing-states deep dive (operator-directed, same sitting)

The operator asked whether either candidate leaves states uncovered, then
commissioned a deep dive: discover the missing states mechanically, resolve
them in both machines, and answer which engine serves UX outcomes best.

**Discovery (two instruments, each catching what the other cannot):**

1. **Lockstep fuzz** (the new `machines-equivalent` spike, run RED against the
   pre-fix pair): 114/5000 walks diverged — all one root cause, XState's
   `stale` ignoring `PROBE_TIMEOUT` while the hand-rolled region sent any dead
   probe to `offline`. The fuzzer's blind spot was instructive: the deeper
   session-lifecycle gaps need precise 6-event paths random walks rarely hit.
2. **Code diff + design-back audit** (§19.1 tables, §13, P-KR4) found what the
   fuzz could not: `expired`/`notFound`/`error` were dead ends in the XState
   build only (no `NEXT_CUSTOMER`); `overridePending` and
   `promptFinishCurrent` were dead ends in BOTH; NEITHER machine had the
   P-KR4 recovery (submit re-arming on its own when reachability returns —
   literally in the card's done_when); a scan mid-submit could yank an
   in-flight verdict in both; `error` offered no retry; and the XState reset
   leaked `overrideAvailable`/`unverifiedWarning` across customers.

**Resolution (red-first):** the suite grew 10 → **18 sequences** (session
reset from every parked state with full flag clear; P-KR4 gate resume ×2;
prompt dismiss-with-memory / clear; §13 double-entry no-op; stale-probe +
override-cancel hardening; mid-submit protection; error RETRY). Against the
unfixed machines the new sequences red exactly as predicted — hand-rolled
6/8, XState 7/8 (the extra being the fuzzer's stale-probe find; the §13
double-entry no-op was already structural in both). Both machines were then
fixed; design calls made and stated: `overridePending` is a terminal-class
"queued — will verify on reconnect" card; the finish-first prompt dismisses
back to the interrupted state (progress preserved, F1's principle) or clears;
the gate auto-resumes its pre-gate state on `RESUBSCRIBED`-while-stale;
mid-submit and mid-confirmation scans are ignored; failures are retryable
(UI-R).

**Post-fix evidence:** 18/18 on both (`01`/`02` re-run exit 0; XState browser
leg unchanged green); `03-machines-equivalent.sh` exit 0 — 40,000 walks × 20
steps × 2 seeds, ZERO divergence. New weights: hand-rolled **156 lines /
7,082 bytes / 0 deps**; XState adapter grew comparably and the 47 KB library
is unchanged.

**Expressiveness findings the fix pass surfaced (both engines, honestly):**

- XState history states do NOT solve "go back to the interrupted state" — they
  apply when a compound state is re-entered, and the scan region never exits.
  Both engines needed an explicit return field; in XState the dynamic return
  costs one guarded branch per returnable state (targets are static, 13
  branches, generated from a list), in the hand-rolled machine it is one
  assignment.
- The parent-level event bubbling praised in the first verdict could not
  carry the deep dive's exclusions (mid-submit protection) or per-source
  return capture — the dedupe moved to per-state entries generated by a
  helper. The bubbling advantage largely evaporated under real requirements.
- One semantic trap each: XState guards evaluate against PRE-event context
  (the connectivity mirror updates in the same event's actions), so
  same-event cross-region reactions guard on the pre-state (`conn ===
  'stale'` meaning "about to be online"); the hand-rolled broadcast runs
  regions sequentially, so its scan region reads POST-transition connectivity.
  Same observable behavior, different reasoning — the suite + fuzz pin it.

**UX-outcomes answer (the operator's question):** once pinned by the suite,
the two machines are observably equivalent, so neither engine yields better
UX *behavior* — the suite is what guarantees UX, and it is engine-agnostic.
What differs is the probability of future UX regressions: the deep dive's
own score is hand-rolled 2 gap classes (both shared design gaps) vs XState 4
(the shared two plus two engine-usage slips — per-state reset omissions and
the stale-probe hole), and every discovered UX-critical behavior (P-KR4
resume, mid-submit protection, dismiss-with-memory) was cheaper to express in
the hand-rolled machine. The recommendation to build hand-rolled STRENGTHENS,
with the suite + lockstep fuzz shipping as the card's tests. XState's
remaining edge is visualization/tooling, which the card can buy later — the
suite makes the swap safe.

- **Corrections from the deep dive** (operator-directed scope, recorded for
  the batch sitting): the eight missing-state fixes above, applied to both
  machines red-first; and the first verdict's "bubbling semantics" claim
  amended as described.

## Addendum 2 — the strict-mode experiment (operator-directed, same sitting)

The operator asked whether XState could throw on unhandled events instead of
no-op'ing (v4's removed `strict: true`), then commissioned the experiment:
rewrite the candidate with the v5 migration guide's replacement — a `'*'`
wildcard whose action throws — and observe what additionally fails. Run as
`js/strict-experiment.mjs` against the real machine (parameterized via opts;
the green path is unchanged — base suite 18/18 ×2 and fuzz no-divergence
re-verified after the edit).

**Selection-semantics probes (measured, load-bearing):** (a) a wildcard DOES
fire when an explicit key's guards all fail — every guarded transition is a
potential trip and needs an explicit `{}` fallback under strictness; (b)/(c) a
child's unhandled or guard-failed event reaches the parent's explicit key
before the parent's wildcard — whitelists work.

**Tier A (naive region-root tripwires): 18/18 sequences FAILED.** The first
cross-region event killed the actor (`connectivity: unhandled SCAN`) —
broadcast semantics make a bare per-region wildcard mean "this region didn't
handle it," which is every sibling event.

**Tier B (sibling alphabets whitelisted): 3 contract ignores tripped — and
only 1/18 sequences failed, which is the sharpest finding of the experiment.**
The trips on `OVERRIDE_REQUEST` (guard-fail) and `SUBMIT` (double-entry no-op)
killed their actors, yet sequences 4/6/10/15 still PASSED: a dead actor is
frozen at its pre-trip state, and an assertion of the form "state must not
change" cannot tell a corpse from a correct no-op. Only sequence 17 failed,
because it needed the actor alive afterwards. A throwing guard therefore
does not just fail loudly — it can make no-op tests pass vacuously.

**Tier C (suite contracts whitelisted too, ~9 more entries): suite 18/18,
0 trips.** Then the lockstep fuzz: **all 3000 walks were killed by the
tripwire before any divergence** — the residual silent-ignore map is ~40
distinct (state × event) sites, every one an early-state stray (`idle`/
`scanning` × late or misplaced events) or a redundant own-region connectivity
event. **Zero of the ~40 is a defect** (consistent with the pre-strict
equivalence proof) — but at least three are REACHABLE in production and
correct-to-ignore, where the throwing guard would brick the screen:
`PROBE_TIMEOUT` while already offline (the probe loop keeps timing out —
guaranteed dead actor within seconds of going offline), a spurious
`RESUBSCRIBED` while online (Realtime re-subscribes), and a late `RESOLVED`
landing after a `NEXT_CUSTOMER` reset. The hand-rolled machine "handles" the
connectivity cases by idempotent assignment, so strictness also exposes that
the two engines *represent* the same behavior differently (handled-as-no-op
vs unhandled).

**Conclusion recorded:** the wildcard-plus-throw guard found no defects, the
whitelist tax to make it suite-green was ~29 declarations with ~40 more owed
for field safety, its actor-fatal semantics would crash the scanner on
reachable benign events, and it degrades test trustworthiness via
pass-by-death. The card should NOT adopt throwing strictness; if loudness is
wanted, a RECORDING wildcard (log/telemetry, no throw) in dev builds gives
the same enumeration — this experiment's trip map is exactly what it would
report — with none of the deaths. The suite + fuzzer remain the real guard.

## Addendum 3 — the operator's history-state challenge (same sitting)

The operator challenged Mechanic Three's "history never applies" claim:
couldn't the region have been SPLIT so the boundary is crossed by design —
flow states in a compound, the prompt outside it, return via history? Probed
against real XState v5: **yes — measured, the 13 DISMISS branches collapse to
ONE `target: 'flow.hist'` transition, and history is genuinely dynamic.** The
"never applies" claim was a property of the spike's FLAT model, not of the
domain; the fork evidence carries this caveat now. What the same probe also
measured: (a) interruption stops being an overlay — the interrupted state's
EXIT actions fire on the way out and its ENTRY actions RE-FIRE on restore
(F1's progress-preservation principle, extended to prompts, wants
literally-nothing-happened, which the flat internal no-op gives for free);
(b) state values become heterogeneous (`"modal"` vs `{"flow":"b"}`) — every
read/assertion handles nesting; (c) layered interruptions (prompt OVER gate)
re-import the problem — history remembers only compound children, and the
gate isn't one, so either nested compound+history pairs per layer or the
`promptReturn` variable returns for exactly the case our special branches
covered. Also noted: the truest remodel the challenge points at is a third
parallel OVERLAY region ({none, prompt}), which dissolves go-back in BOTH
engines (nothing moves, nothing to remember) at the price of more
cross-region guards — XState's weaker suit, hand-rolled's strong one. The
Mechanic-Two tally in the comparison narrows under a better model; the
gate-resume mechanics (mirror, pre-event guards) and the strict-mode findings
are untouched by it. Candidate for the card's design, not adopted here.

**Overlay variant built and proven (operator-commissioned, same sitting):**
`js/machine-xstate-overlay.mjs` (199 lines) models the prompt as a third
parallel region — the scan region NEVER moves when the prompt appears, so
go-back does not exist: `DISMISS: 'none'` is the entire mechanism, no memory,
no branches. Results: **18/18 on the shared suite, first run**, and
**observably equivalent to the hand-rolled machine across 40,000 lockstep
walks × 20 steps × 2 seeds (0 divergence — `overlay-fuzz.mjs`)**. The flat
model's two "reconnected while prompting" special branches dissolved: the
gate auto-resumes UNDER the prompt (RESUBSCRIBED bypasses the overlay gate)
and DISMISS lands on the already-live submit. Relocated costs, measured in
the file: a SECOND cross-region mirror (`sc` — the overlay's open guard needs
the scan state; 17 entry assigns) and a mechanical `overlay === none` gate
injected into every user-driven scan transition (a modal blocks the controls
beneath it — with NEXT_CUSTOMER and RESUBSCRIBED deliberately bypassing).
So the challenge's best form is confirmed: under the overlay shape, Mechanic
Two's cost goes to ~zero for XState and the fork's remaining friction is all
Mechanic-One territory (cross-region knowledge: mirrors and gates vs the
hand-rolled machine's direct sibling reads). All three candidates now pass
the same suite and fuzz observably identical; the overlay shape is the
strongest XState-side candidate for the card's design.

## Review

- signed: operator, 2026-09-04 — covers 1 agent-reached correction (the UMD
  browser-leg artifact fact; one-sitting batch across Activities B/C/D,
  "Sign off all three" on the phrase-checked batch question). The deep-dive
  fixes, the strict-mode experiment, and the overlay variant (Addenda 1–3)
  were operator-directed in the same sitting — reviewed by direction, not
  part of the agent-corrections batch.

## Comebacks

- gap: GAP-1 — the §19.1 design carried eight missing-state gap classes
  (dead-end terminals without `NEXT_CUSTOMER`, dead-end `overridePending` /
  `promptFinishCurrent`, no P-KR4 auto-resume, mid-submit scans yanking
  in-flight verdicts, no error retry, XState flag leaks across customers,
  stale-probe hole), found by the missing-states deep dive (lockstep fuzz +
  design-back audit, 2026-09-04).
- validated: GAP-1 — fixed red-first in both machines the same sitting; suite
  grew 10 → 18 and re-ran 18/18 on all candidates, lockstep fuzz 40,000
  walks × 20 × 2 seeds with zero divergence, exit 0 (2026-09-04). The card
  inherits the 18-sequence suite + fuzz as its tests, so the fix survives the
  throwaway scripts.
