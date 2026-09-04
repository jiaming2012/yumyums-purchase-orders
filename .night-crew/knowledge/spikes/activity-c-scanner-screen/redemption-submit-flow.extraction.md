# Extraction — redemption-submit-flow

Outcome: learned

Approach used: the commissioned fork (XState vs hand-rolled, "decided by a
spike near Activity C") was run as THREE candidates against one shared
instrument — an 18-sequence conformance suite transcribed from the §19.4
acceptance criteria (`js/conformance.mjs`, machine-agnostic) plus seeded
lockstep fuzzing (40,000 walks × 20 steps × 2 seeds). The candidates, all in
`.night-crew/spikes/activity-c-scanner-screen/redemption-submit-flow/js/`:

- **hand-rolled** parallel-region machine — **156 lines / 7,082 bytes /
  0 dependencies**, direct sibling-region reads;
- **flat XState v5.32.6** — `xstate.umd.min.js` vendored as one classic
  `<script>` (**47,268 bytes**, the SortableJS pattern), connectivity mirror
  in context for cross-region guards, 13-branch dynamic return for prompts;
- **overlay-region XState variant** (199 lines + the same lib) — the prompt as
  a third parallel region, so go-back does not exist (`DISMISS: 'none'` is the
  whole mechanism), at the price of a second cross-region mirror and an
  `overlay === none` gate on every user-driven scan transition.

All three pass the same suite 18/18 and fuzz observably identical (0
divergence). Every one is a **candidate**, not an adoption (NFR-6). Full
evidence: the "Scanner Machine Fork" and "The Go-Back Problem" artifacts
(URLs in the operator's memory note `scanner-fork-artifacts`); the ledger's
Addenda 1–3 are the record of truth.

Confirmed: both doors of the fork are genuinely open — the §19.1 model is
expressible and no-build-loadable in XState v5 (UMD classic script drives a
parallel machine in real Chromium), AND a dependency-free machine passes the
identical externally-derived suite. The fork is decidable on recorded
evidence, which was the commission.

Learned: four things the fork question alone would not have surfaced.

1. **The design had eight missing-state gap classes** (lockstep fuzz + code
   diff/design-back audit, each catching what the other could not): dead-end
   terminals without `NEXT_CUSTOMER`, dead-end `overridePending` /
   `promptFinishCurrent`, NO P-KR4 auto-resume (literally in the card's
   done_when), mid-submit scans yanking in-flight verdicts, no error retry,
   XState flag leaks across customers, and the stale-probe hole. Fixed
   red-first in both machines; the suite grew 10 → 18 sequences. Anchored as
   GAP-1 in this goal's ledger `## Comebacks`, validated same sitting (18/18
   both + 40k-walk fuzz, exit 0).
2. **Throwing strictness is a trap** (Addendum 2): a wildcard-throw guard
   found ZERO defects, costs ~29 whitelist declarations plus ~40 more for
   field safety, would brick the scanner on reachable benign events
   (`PROBE_TIMEOUT` while offline guarantees a dead actor within seconds),
   and degrades tests via pass-by-death (a frozen corpse satisfies "state
   must not change"). If loudness is wanted: a RECORDING wildcard in dev
   builds, never a throwing one.
3. **Engine ergonomics under real requirements** (Addenda 1 + 3): XState's
   bubbling advantage evaporated (exclusions and per-source return capture
   forced per-state entries); history states do fix go-back under a split
   compound model (the flat model's "never applies" was a model property, not
   a domain one) but re-import the problem for layered interruptions; the
   overlay remodel dissolves go-back in BOTH engines. Gap-class score across
   the deep dive: hand-rolled 2 (both shared design gaps) vs XState 4 (the
   shared two plus two engine-usage slips).
4. **The suite, not the engine, guarantees UX**: once pinned, the machines
   are observably equivalent — engine choice affects future-regression
   probability and tooling, not behavior.

Plan change: the fork's recommendation is **build the hand-rolled machine**,
with `conformance.mjs` (18 sequences) AND the lockstep fuzz shipping as the
card's tests — that pinning is what makes the engine swappable later if
XState's visualization/tooling is ever wanted, and it is how the card's
done_when should phrase behavior. The overlay-region shape is the strongest
XState-side candidate if the card's design goes the other way. Design calls
already made and stated in the ledger bind the card regardless of engine:
`overridePending` is a terminal-class "queued — will verify on reconnect"
card; the finish-first prompt dismisses back to the interrupted state or
clears; the gate auto-resumes on `RESUBSCRIBED`-while-stale (P-KR4);
mid-submit and mid-confirmation scans are ignored; failures are retryable
(UI-R). No throwing strictness. Adoption itself lands at the card's design.md
under slate sign-off (NFR-6) — this record is the candidate evidence, not the
decision.
