# Card actuals — cycle-time ledger

> **Purpose:** the empirical basis future slates size against. This slate
> (`overnight-20260710`) is the **first** — there was no prior ledger, so its
> estimates were wide first-of-kind ranges. These are the first real data points.
> **What one "card" covers:** draft (2 enumeration passes) → G6 adversarial review
> → any REVISE loop → orchestrator commit + roadmap flip. Wall-clock, orchestrated
> serially by one control loop dispatching fresh subagents per card.
> **Caveat (read before using these to size):** these actuals are **subagent
> wall-clock**, not human-drafting time. They came in **6–10× under** the slate's
> first-of-kind estimates (which implicitly assumed human-paced drafting). The
> *ratio between cards* (Inventory ≈ 1.5× the small cards) is the more transferable
> signal than the absolute minutes.

## Run: overnight-20260710 (Activity 1 — enumerate & mark PRD cards)

| Card | Est. (slate) | Actual (wall-clock) | Passes | G6 verdict | Draft→G6→revise breakdown |
|---|---|---|---|---|---|
| `users-hardening-prd` | 30–55 min | **8m 20s** | 2 | REVISE→pass | draft 3m16s · G6 1m07s · revise 2m00s |
| `onboarding-hardening-prd` | 45–75 min | **9m 48s** | 2 | REVISE→pass | draft 4m14s · G6 1m54s · revise 2m03s |
| `purchasing-hardening-prd` | 45–75 min | **8m 02s** | 2 | REVISE→pass | draft 3m58s · G6 1m06s · revise 1m03s |
| `inventory-hardening-prd` | 60–110 min | **11m 48s** | 2 | ACCEPT (1st pass) | draft 8m29s · G6 1m46s · no revise |
| **Total (4 cards, serial)** | 3.5–5 h | **~38 min** | — | 3 REVISE + 1 ACCEPT | — |

## First-slate observations (seed for next cycle's sizing)

- **Absolute minutes are not portable.** All four cards finished 6–10× under
  first-of-kind estimates. Do not carry the raw minutes forward as a budget; carry
  the *shape*.
- **Surface size tracks draft time, weakly.** Inventory (2498 L HTML, 40 reqs,
  154 KB spec) drafted in ~8.5 min vs ~3–4 min for the small apps — ~2× the draft
  time for ~1.5× the requirement count. Draft time scales sub-linearly with surface
  because the subagent parallel-reads; G6 time is near-flat (~1–2 min) regardless.
- **REVISE loops are cheap but common.** 3 of 4 cards took one REVISE round
  (~1–2 min each). The revises were *bookkeeping*, never re-enumeration: a status
  mis-mark (Users), tally arithmetic (Onboarding), a route-count typo (Purchasing).
  Budget one short revise round per card by default; a clean first-pass ACCEPT
  (Inventory) is the exception, not the rule.
- **The load-bearing gate was G6 tally/status scrutiny, not draft depth.** Every
  REVISE came from the adversarial reviewer catching an internal inconsistency the
  drafter's own report asserted was clean. Keep G6 fresh-context and separate.

## Run: overnight-20260712 (Activity 2 confirm-absence + Activity 3 test-audit cards)

> **What one "card" covers here (different shape from Activity 1):** a fresh AUDIT
> subagent (two-pass static inspection of an existing PRD + source/tests, returns a
> report of proposed reclassifications with cited lines) → a SEPARATE fresh G6
> subagent (re-checks every citation at the line) → the ORCHESTRATOR applies the PRD
> status-mark edits, flips the roadmap row, appends backlog, commits. Subagents are
> read-only; only the orchestrator writes. Serial, one card at a time.
> **Times below are subagent wall-clock (audit leg + G6 leg).** Orchestrator
> apply/commit time between legs is control-loop time, not separately instrumented
> (~1–3 min/card). No REVISE loops were needed — every card's G6 returned PASS on the
> first review (contrast Activity 1's 3-of-4 REVISE rate; audit reports are smaller and
> more mechanical to check than a full PRD draft).

| Card | Type | Audit leg | G6 leg | G6 verdict | Reclassification |
|---|---|---|---|---|---|
| ops-confirm-absence | CA | 1m56s | 1m56s | PASS | 2 BROKEN (FR-4, NFR-3); FR-12 NEG |
| ops-test-audit | TA | 0m56s | 1m34s | PASS | 0 drops |
| users-confirm-absence | CA | 0m55s | 1m31s | PASS | 0 graduations |
| users-test-audit | TA | 0m54s | 1m31s | PASS | 0 drops |
| onboarding-confirm-absence | CA | 2m26s | 2m01s | PASS | 1 BROKEN (NFR-5) |
| onboarding-test-audit | TA | 1m22s | 3m18s | PASS | 0 drops (6 guard flags) |
| purchasing-confirm-absence | CA | 1m16s | 1m12s | PASS | 0 graduations; FR-13 reconciled |
| purchasing-test-audit | TA | 1m08s | 0m50s | PASS | 1 drop (FR-7 → UNPROVEN) |
| inventory-confirm-absence | CA | 2m01s | 1m42s | PASS | 0 graduations; NFR-1 gaps |
| inventory-test-audit | TA | 2m49s | 2m09s | PASS | 0 drops (~40 guard cleanup) |
| **Total (10 cards, 20 subagents, serial)** | — | **~16m** | **~18m** | 10 PASS / 0 REVISE | 3 new BROKEN + 1 drop |

### Second-slate observations (seed for the Activity-4 slate's sizing)

- **Estimate vs actual:** the slate estimated ~6–10 min subagent wall-clock/card; actuals
  came in **~2–5 min combined (audit+G6)/card** — again well under, same 6–10× pattern as
  Activity 1. The slate's *ratio* held: the big surfaces (Onboarding, Inventory) ran ~1.5–2×
  the small ones (Users, Purchasing-TA). Critical path (serial, 20 subagents) ≈ **~34 min of
  subagent wall-clock** + orchestrator apply — comfortably inside the estimated ~2h20m, which
  had assumed the higher per-card range.
- **G6 can cost more than the audit it reviews.** On low-graduation cards the G6 leg often
  ran *longer* than the audit (onboarding-TA: audit 1m22s, G6 3m18s) because "prove a negative"
  (no hidden BROKEN / no missed vacuity) means re-opening every cited line independently. Budget
  G6 ≈ 1–1.5× the audit leg, not a fixed small constant.
- **0 REVISE this run** (vs 3/4 in Activity 1). Audit reports are structured tables of cited
  claims — the G6 either confirms a citation at the line or rejects it; there's no "tally
  arithmetic" to slip. When G6 found extra signal (e.g. the confirm-vendor normalization gap
  beyond NFR-1's UpdateItem gap) it was an *addition*, not a REVISE of the audit's verdict.
- **The two-pass was load-bearing exactly where predicted:** every BROKEN graduation
  (Ops FR-4/NFR-3 enforcement-absence, Onboarding NFR-5 key-mismatch) was invisible to the
  UI-first pass and only surfaced in the backend-only cross-check — matching the Activity-1
  finding that sub-90% recall hides in backend-only surfaces. Keep the pass-2 angle explicit
  in every card prompt.
</content>
