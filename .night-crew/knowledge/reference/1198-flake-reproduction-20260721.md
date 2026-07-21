# `sync.spec.js:1198` — flake reproduced under controlled concurrent load (2026-07-21)

**Branch:** `investigate/1198-under-load-20260721` · **Verdict: the test IS flaky.**
Rate **4 red / 25 legs = 16%** overall; **4/20 = 20%** under a concurrent Playwright suite.
Not "not flaky". Not ~50% either. The prior evidence was directionally right and
numerically wrong.

## Headline

Four independent reproductions after ~20 prior consecutive greens by two other parties.
All four share **one** failure signature, and it is **not** the one the card assumes.

## Method

Isolated stack throughout: own ephemeral pg16 (Docker-assigned port), own DB, own
server port `:8241`. Host `:5432` untouched; other sessions' `yumyums-e2e-pg` untouched.
`CI=1` and `--retries=0` on every leg, no exceptions.

**Load choice — a concurrent full Playwright suite, not synthetic CPU noise.** The
contaminating condition in the original report was another Claude session running the
E2E suite. That produces the specific contention that matters here: ~18 competing
headless Chromium processes, a second Go server broadcasting over WS, and shared
Postgres churn. A `stress-ng`-style spinner would raise the load average without
reproducing the event-loop/WS interleaving the failure actually depends on — and the
results below confirm that instinct: **load magnitude does not predict the failure.**

- **Tier A (legs 01-05)** — ambient only (other sessions' Go builds).
- **Tier B/C (legs 06-25)** — Tier A plus one continuously-looping full 529-test
  Playwright suite on its own stack, sharing the same Postgres instance.

## Per-leg results

| Leg | Tier | Result | Wall | Load (start→end) | Ops journal |
|---|---|---|---|---|---|
| 01 | A | GREEN | 148s | 3.94 → 4.74 | — |
| 02 | A | GREEN | 140s | 4.08 → 1.93 | 9 |
| 03 | A | GREEN | 136s | 1.93 → 2.56 | 14 |
| 04 | A | GREEN | 139s | 2.56 → 2.48 | 19 |
| 05 | A | GREEN | 139s | 2.48 → 3.82 | 24 |
| 06 | B | GREEN | 141s | 5.28 → 11.80 | 29 |
| 07 | B | GREEN | 140s | 11.80 → 6.50 | 34 |
| **08** | **B** | **RED** | 151s | **6.50 → 5.09** | **37 (+3)** |
| 09 | B | GREEN | 142s | 5.09 → 5.69 | 42 |
| 10 | B | GREEN | 170s | 5.69 → 24.77 | 47 |
| 11 | B | GREEN | 148s | 24.77 → 12.40 | 52 |
| 12 | B | GREEN | 142s | 12.40 → 8.59 | 57 |
| 13 | B | GREEN | 146s | 8.59 → 21.46 | 62 |
| 14 | C | GREEN | 146s | 19.77 → 10.67 | 67 |
| 15 | C | GREEN | 143s | 10.67 → 5.84 | 72 |
| 16 | C | GREEN | 141s | 5.84 → 7.65 | 77 |
| 17 | C | GREEN | 141s | 7.65 → 7.21 | 82 |
| **18** | **C** | **RED** | 150s | **7.21 → 2.88** | **85 (+3)** |
| 19 | C | GREEN | 142s | 2.88 → 3.44 | 90 |
| **20** | **C** | **RED** | 151s | **3.44 → 5.00** | **93 (+3)** |
| 21 | C | GREEN | 139s | 5.00 → 5.78 | 98 |
| 22 | C | GREEN | 137s | 5.78 → 7.47 | 103 |
| 23 | C | GREEN | 137s | 7.47 → 2.70 | 108 |
| **24** | **C** | **RED** | 149s | **2.70 → 3.40** | **111 (+3)** |
| 25 | C | GREEN | 140s | 3.40 → 3.92 | 116 |

Journal depth grew monotonically 9 → 116 across the run (DB never reset between legs).

## Load does not predict the failure

All four reds started at load **≤ 7.21** (6.50, 7.21, 3.44, 2.70). Greens ran clean at
**24.77, 21.46, 19.77, 12.40**. The discriminator is plausibly the *presence* of a
competing suite rather than the load average — but Tier A is only n=5, and
p(0 red in 5 | 20%) ≈ 0.33, so **Tier A's clean sweep does not establish a quiet box is
safe.** Report it as "not established", not "safe". Note the original red also occurred
at moderate load (3.96).

## The failure signature — all 4 reds identical

```
TimeoutError: page.waitForResponse: Timeout 12000ms exceeded while waiting for event "response"
  > 1119 |   const committed = pageB.waitForResponse(
        at survivalCell (tests/sync.spec.js:1119:27)
        at tests/sync.spec.js:1200:5
```

This is the **`POST /ops` commit wait**, with its own hardcoded `12000`. It is **not**
`CONVERGE_TIMEOUT` and **not** a convergence assertion. Three consequences:

1. **The card's stated next step is aimed at the wrong wait.** BACKLOG and
   `slate-20260720c` both say "`survivalCell`'s 12s `CONVERGE_TIMEOUT` budget vs the real
   WS round trip." The failing wait is a different one.
2. **Raising the timeout cannot fix it.** The autosave debounce is **400ms**
   (`workflows.html:278`) — a 30× margin. A 12s expiry means the op never fired.
3. **The journal proves the op never fired.** Green legs add **+5** ops; every red leg
   adds **+3**. No `SET_FIELD` row is written.

## Mechanism: the clobber race was relocated, not closed

The de-flake comment above `survivalCell` names a stray WS catch-up `loadMyChecklists`
re-render that "can re-render the runner and CLOBBER that value." That mechanism is
**still live**, one step earlier in the flow. The re-render detaches the temperature
input between `fill('375')` and `dispatchEvent('change')`; the change handler never runs
on the attached node, `debouncedSaveField` is never armed, and **no POST is ever issued**.

The de-flake's step 1 claims gating on the `POST /ops` 2xx is "a race-free 'draft is
durable' signal." It is race-free *against a clobber of an already-sent value* — but it
is **not** race-free against a re-render that prevents the POST from existing at all. The
gate can only observe a POST that happens. What the de-flake achieved was moving the
observable from `Received ""` at the baseline assert to a timeout at the commit gate.

This is the honest label: **rare, mechanism understood, bounded at ~16-20%** — *not*
"not flaky", and *not* a clean bill of health.

## Corrections to the record

- **"red 1-of-2 `--retries=0` legs at load 0.84"** misattributes the load. The source
  table (`runs/2026-07-22-autonomous/DECISIONS-NEEDED.md` §"S1 tail") reads: leg 1, load
  **0.84**, **PASSED**; leg 2, load **3.96**, **RED**; leg 3, load 6.48, aborted. Load
  0.84 belongs to the green leg. **"Proven flaky on a quiet box" is not supported by its
  own evidence** — though the conclusion "flaky" is now independently confirmed.
- **The implied ~50% rate is refuted.** Measured 16% (95% CI ≈ 5-36%) overall, 20%
  (95% CI ≈ 6-44%) under concurrent suite. The two parties' ~20 greens are consistent
  with this once conditioning on contention is accounted for.
- **`:525 FLD-LIVE-02` was not examined here.** Its scope in the card stands unchanged.

## Recommended disposition

1. **Keep the de-flake card. Do not mark `:1198` "not flaky."** The flake is real and
   reproducible on demand under a concurrent suite.
2. **Re-aim it.** Fix is test-side and does not need a production change: settle the
   runner before answer entry, and/or re-assert the value and re-dispatch `change` if no
   `POST /ops` is observed within a short window. Do **not** raise `CONVERGE_TIMEOUT` or
   the `:1119` timeout — both are already 30× the debounce.
3. **The `cycle-gate` no-retry premise stands.** It genuinely cannot pass while this
   test reds ~1-in-5 under contention. That dependency was correctly identified.
4. **Reproduction recipe** for whoever takes the card: run `:1198` with `CI=1
   --retries=0` while a second full Playwright suite loops against the same Postgres.
   Expect a red within ~5-10 legs; ~150s per leg.

## Footprint

Investigation only. `tests/sync.spec.js` — comment annotation. This file + the BACKLOG
entry. No production code touched.
