# RUN-10 attribution — the orchestrator measurement

`tests/workflows.spec.js` › *Loading states › unsubmit returns checklist to editable draft* **[RUN-10]**

## Why this file exists

RUN-10 went red on **two independent full-suite legs tonight** — card B's and card C's — on trees
that both contain card A, which was already merged. Both cards **refused to attribute it**, which
was the correct call and is the standing rule (guessing "pre-existing" is exactly as wrong as
guessing "mine"). But a refusal is not an answer, and an unattributed red hanging over an
already-merged card is not something to hand the operator at 7am without having spent the
measurement.

This is the same move that on 20260725 turned 21 minutes of orchestrator measurement into a proven
root cause instead of a wrong fact in the record.

## What was already known before the orchestrator measured anything

| Condition | Tree | Result | Source |
|---|---|---|---|
| Card A's own gate, 4-spec subset (`workflows` 80 tests) | post-A | **RUN-10 green**, 0 failed / 0 flaky, load 2.43→5.36 | card A gate leg |
| RUN-10 alone, ×3, `--retries=0` | post-A | **green 3/3** | card C |
| Whole `workflows.spec.js`, 80 tests, `--retries=0` | post-A | **green 80/80** | card C |
| FULL suite, load 35→61 | post-A | **RUN-10 RED**, failed both attempts | card C gate leg |
| FULL suite, load →57.6 | post-A | **RUN-10 RED** (plus LC-02 `:2355`) | card B gate leg |

**83 attempts outside the whole-suite condition, zero reproductions.** So the failure lives in the
*whole-suite* condition, not in the test and not in load alone. That is exactly why an isolated
re-run cannot settle it — which the first orchestrator attempt had to learn.

## Measurement 1 — isolated RUN-10, both trees (`--repeat-each=6 --retries=0`)

**post-A (card A present), `overnight-20260726` @ `c9dc440`, port 8599:**

    6 passed (4.5m)    load 44.33 → 21.21

🛑 **Note the load: 44.33.** RUN-10 passed 6/6 *in isolation under load higher than either gate leg
that saw it fail.* This is a real result and it is informative: **load alone, in isolation, does not
reproduce RUN-10.** Combined with the 83 prior attempts, that is 89 isolated attempts clean.

**pre-A (card A absent) @ `4bcd63d`, port 8699: ⚠️ VOID — NOT EVIDENCE.**

The leg reported `6 failed`, but every failure is `page.waitForURL: Test ended` inside the
`beforeEach` login helper (`workflows.spec.js:21`) — the suite never reached RUN-10's body. **Cause:
the orchestrator `pkill`ed this leg** while redesigning the experiment. The failures are the kill,
not the tree.

**This leg is recorded as void and must not be read as "RUN-10 fails without card A."** It would be
a very convenient result to accept — which is exactly why it is being thrown out rather than
quietly used. (Incidental confirmation the trees are the right ones: RUN-10 sits at `:2296` pre-A
and `:2466` post-A, the ~170-line shift card A's added tests introduce.)

## Measurement 2 — PAIRED FULL SUITES, run concurrently

The design measurement 1 could not be. Isolated runs cannot discriminate, because the failure does
not occur in isolation **on either tree**. So: the **full suite on both trees at once**, same box,
same wall-clock, same contention — post-A on `:8599`/`hq_test_m1`, pre-A on `:8699`/`hq_test_m2`,
`--retries=0` so nothing is masked.

**How to read it — decided BEFORE the result, so the reading is not fitted to it:**

| Outcome | Conclusion |
|---|---|
| RUN-10 red on **both** trees | **Card A exonerated.** Whole-suite/contention-sensitive pre-existing flake. |
| RUN-10 red on **post-A only** | **Card A implicated** — strong, and the merged card needs review. |
| RUN-10 red on **pre-A only** | Anomalous. Report as anomalous; do **not** rationalize it. |
| RUN-10 **green on both** | **Inconclusive at this load.** Report as a bound, **never** as "fixed". |

Expected and *not* evidence about RUN-10: the pre-A tree should be RED on
`tests/repro-cut-task.spec.js:153` and `tests/sync.spec.js:1581` — those are precisely the two specs
card A repaired, and their being red pre-A is a control confirming the pre-A tree is what it claims
to be.

### RESULT

    launched together, load_at_launch=26.91 → load_at_finish=9.57

| Leg | Result | RUN-10 |
|---|---|---|
| **POST-A** (card A present), 549 tests | **544 passed / 0 failed / 6 skipped**, 32.2 m | ✅ **GREEN** (test 526) |
| **PRE-A** (card A absent), 547 tests | **539 passed / 2 failed / 6 skipped**, 32.4 m | ✅ **GREEN** (test 524) |

**🛑 The control fired exactly as predicted, and that is what makes this result trustworthy.**
The pre-A leg's only two failures are:

    ✘ tests/repro-cut-task.spec.js:153  AC-6b: submitted record byte-identical after later template edits
    ✘ tests/sync.spec.js:1581           unsubmit transition converges live on the observing device

**Those are precisely the two specs card A was written to repair — and nothing else failed.** The
pre-A tree is genuinely pre-card-A, the harness discriminates, and post-A is genuinely card-A-fixed.
A measurement whose control had *not* fired would have been worthless regardless of what RUN-10 did.

### VERDICT — read against the rules written down beforehand

Both trees green → by the pre-committed table this is **"inconclusive at this load — report as a
bound, NEVER as *fixed*."** That is the ruling, and it is not being upgraded after the fact.

**What this run legitimately establishes, and it is not nothing:**

1. **Card A did NOT deterministically break RUN-10.** This is the first full-suite run in which
   RUN-10 has passed on a post-A tree — it was red twice before, and green here. A regression is
   deterministic; **non-determinism on the same code rules out "card A broke it."** That is a real
   narrowing, arrived at by measurement.
2. **RUN-10 remains genuinely non-deterministic in the whole-suite condition**, now demonstrated on
   post-A in both directions (2 red, 1 green) across three full-suite runs.
3. **The reproducing condition is heavier than this run reproduced.** The two failing legs peaked at
   load **57.6** and **61+**, with a full Playwright suite, a Go suite, and G6 agents all in flight.
   This paired run peaked far lower (26.91 → 9.57, and both suites finished in **32 m** against card
   C's **47 m** — the wall-clock gap is itself the load evidence). **Two concurrent full suites at
   moderate load is not the condition that failed.**

**What it does NOT establish, stated plainly:**

- ❌ It does **not** exonerate card A. Green-on-both is consistent with card A being innocent *and*
  with a contention-sensitive interaction that this load never reached. **"Card A is cleared" is not
  a claim this measurement supports**, and it would be the easy thing to write.
- ❌ It does **not** prove a pre-existing flake. Nobody has reproduced RUN-10 red on a **pre-A** tree
  in any condition — the one attempt (measurement 1's pre-A leg) was void by the orchestrator's own
  hand. **The symmetric evidence does not exist.**
- ❌ It does **not** mean RUN-10 is fixed. Nothing was changed.

### The honest one-line answer for morning triage

**RUN-10 is a non-deterministic whole-suite failure that requires heavier contention than a paired
full-suite run at moderate load. Card A is not deterministically responsible. Beyond that it remains
UNATTRIBUTED, and the run declines to close it.**

### What would actually settle it, for whoever picks this up

Reproduce the *failing* condition, not a milder one: **a full 549-test suite on post-A with
deliberate competing load sufficient to hold 1-min load ≥ 55** (a second full suite plus a Go suite,
which is what tonight had), repeated enough times to get a rate. Then the same on pre-A. **A red on
pre-A closes it immediately** and is the cheapest possible win — it is the single observation nobody
has yet made, and it is worth trying first.

Note the standing context: `sync.spec.js` is a known load-sensitive file, `sync.spec.js:1198` is a
proven ~16–20 % flake, and **card A's merged seam fix deliberately raises that exposure**. Tonight
`:1198` did not red once across five suite legs — one clean night, not a fix.

### Methodological caveat, stated rather than buried

Cards C and B were merged into the same working tree the post-A leg runs from, while it was running.
Verified before trusting the result: `workflows.html`, `tests/workflows.spec.js` and
`playwright.config.js` are **unchanged** since the leg launched (`git diff 9b200a5 HEAD`), and the
Go server binary was compiled at webServer start, before the merges. The only diff on the test
surface is an 11-line **comment-only** addition to `tests/grant-enforcement-parity.spec.js`, a spec
Playwright had already loaded into memory. The post-A leg's RUN-10 result stands; it would not have
been reported without this check.
