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

_Result appended below when the paired run completes._
