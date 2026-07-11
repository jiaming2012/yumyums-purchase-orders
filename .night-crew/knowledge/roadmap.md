# Roadmap — HQ hardening cycle

> **Cycle:** HQ hardening — first night-crew guinea-pig run. **Traces to:**
> `.night-crew/knowledge/okrs.md` (Product / Delivery / Engineering / QA).
> **Produced:** 2026-07-09 attended session. *(Design note: this roadmap should be
> produced by `/nc-okr-session` at cycle-planning time — see the run's
> `design-findings.md`, "no orphan inputs." It was authored here to close that gap
> retroactively; treat it as the artifact the corrected OKR session would emit.)*

## How this roadmap works

- **Activity-level cards.** Each card is WO-sized-ish work the PjM/`nc-slate-plan`
  sizes to a night. Cards carry a **module footprint** (for parallel tracks) and a
  **KR trace**.
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
- **Cadence is the PjM's, not the operator's.** How many cards land per overnight
  pass is the planner's call against the night budget + the quality bar. The
  operator owns the *bar*, not the *throughput*.
- **Two passes are mandatory.** Every app's flow enumeration = a first pass + an
  independent cross-check. *(Empirical: single-pass recall ≈ 85% on Operations,
  under the 90% KR — the cross-check found 4 missed flows. Non-negotiable for the
  other four apps.)*

## App footprints (independent → parallelizable)

| App | Frontend | Backend | Tests |
|---|---|---|---|
| Operations | `workflows.html` | `backend/internal/workflow` | `workflows.spec.js`, `persistence.spec.js` |
| Inventory | `inventory.html` | `backend/internal/inventory` | `inventory.spec.js`, `recipes.spec.js` |
| Onboarding | `onboarding.html` | `backend/internal/onboarding` | `onboarding.spec.js` |
| Users | `users.html` | `backend/internal/users` | *(audit for tests)* |
| Purchasing | `purchasing.html` | *(mockup — minimal backend)* | *(none yet)* |

The five apps share no source files → their cards run as **parallel tracks** (only
the cross-app cycle-gate card serializes at the end).

---

## Activity 1 — Enumerate & mark (the PRD gate) · *blocking, first*

The OKR's early blocking gate: 5/5 apps get a hardening PRD (flow map + honest
status) before WO build. **Exemplar-first** — Operations sets the pattern the other
four copy. Each non-exemplar card = *first pass + cross-check → draft-for-sign-off →
morning-triage sign-off*.

| Card | Status | Depends on | KR |
|---|---|---|---|
| `ops-hardening-prd` (exemplar) | **DONE** ✅ signed 2026-07-09 | — | Product KR-1/2 |
| `inventory-hardening-prd` | **DONE** ✅ signed 2026-07-10 | exemplar | Product KR-1/2 |
| `onboarding-hardening-prd` | **DONE** ✅ signed 2026-07-10 | exemplar | Product KR-1/2 |
| `users-hardening-prd` | **DONE** ✅ signed 2026-07-10 | exemplar | Product KR-1/2 |
| `purchasing-hardening-prd` | **DONE** ✅ signed 2026-07-10 | exemplar | Product KR-1/2 |

> ✅ **Activity 1 complete — 5/5 apps have a signed hardening PRD (Product KR-1 gate closed).**
> All four overnight PRDs signed at morning triage 2026-07-10 (all G6-passed; the 3
> confirmed-BROKEN citations re-verified at cited lines). **Activities 2–5 now unblock** —
> they need localhost Postgres + the E2E suite, which the enumerate-only run deliberately
> did not touch. Triage resolutions recorded in `ledger.md` (2026-07-10).

> ✅ **Purchasing fork resolved 2026-07-09 → (a) enumerate + mark as a real app.**
> The "bare mockup" premise was stale: Purchasing has a real, tested backend
> (`backend/internal/purchasing/*` — ~20 endpoints), a 1,078-line `purchasing.html`,
> and a 31 KB `tests/purchasing.spec.js`. Its hardening PRD copies the exemplar shape
> and marks WORKING/UNPROVEN/BROKEN honestly (confirmed-only-BROKEN rule), same as the
> other three; the PRD's scope note must correct CLAUDE.md's stale "Mockup" label.
> Slated on `overnight-20260710`. (Recorded in `ledger.md`.)

## Activity 2 — Confirm-absence sweeps

For each app's **priority-UNPROVEN** flows (the "we're not even sure it works"
ones), a quick grep/inspect step; anything confirmed missing/stubbed graduates to
BROKEN → a code-fix card.

| Card | Status | Depends on | KR |
|---|---|---|---|
| `ops-confirm-absence` (FR-12 reject, NFR-3 photo-required) | **DRAFTING** (overnight-20260712; G6-passed → 2 BROKEN: FR-4, NFR-3; FR-12 confirm NEGATIVE) | `ops-hardening-prd` | Eng KR-1 |
| `users-confirm-absence` | **DRAFTING** (overnight-20260712; G6-passed → 0 graduations, 16 stay UNPROVEN present-but-untested) | `users-hardening-prd` | Eng KR-1 |
| `onboarding-confirm-absence` | PLANNED | `onboarding-hardening-prd` | Eng KR-1 |
| `purchasing-confirm-absence` | PLANNED | `purchasing-hardening-prd` | Eng KR-1 |
| `inventory-confirm-absence` | PLANNED | `inventory-hardening-prd` | Eng KR-1 |

## Activity 3 — Audit the WORKING (non-vacuous test check)

Spot-audit every flow marked WORKING that its test actually asserts (no
`test.skip`, no guard-return). A vacuous test drops the flow to UNPROVEN.

| Card | Status | Depends on | KR |
|---|---|---|---|
| `ops-test-audit` | **DRAFTING** (overnight-20260712; G6-passed → 0 drops, all 10 WORKING non-vacuous; FR-15 photo/builder-UI coverage gap noted for downstream WO) | `ops-hardening-prd` | QA KR-1 |
| `users-test-audit` | PLANNED | `users-hardening-prd` | QA KR-1 |
| `onboarding-test-audit` | PLANNED | `onboarding-hardening-prd` | QA KR-1 |
| `purchasing-test-audit` | PLANNED | `purchasing-hardening-prd` | QA KR-1 |
| `inventory-test-audit` | PLANNED | `inventory-hardening-prd` | QA KR-1 |

## Activity 4 — Prove & fix the UNPROVEN (the WO backlog) · *bulk delivery*

Per the sign-off policy: **test-only WO first** (write the red-first assertion);
graduate to a **fix WO only if the test goes red**. Operations alone = 17 UNPROVEN.

| Card | Status | Depends on | KR |
|---|---|---|---|
| `ops-prove-unproven` (17 flows) | PLANNED | ops confirm-absence + audit | Delivery/Eng/QA |
| `<app>-prove-unproven` (×4) | PLANNED | that app's PRD-chain | Delivery/Eng/QA |

## Activity 5 — Cycle gate (closeout) · *serializes last*

The OKR closeout, across all five apps.

| Card | Status | Depends on | KR |
|---|---|---|---|
| `cycle-gate` | PLANNED | all Activity-4 cards | Eng KR-1/2, QA KR-1/2/3 |

Gate = **0 known-broken flows** · **full E2E suite green on localhost Postgres, 0
pre-existing reds** · **vacuous tests 23 → 0** · **every repaired flow carries a
red-first proof** · **median WO cycle time recorded over ≥5 WOs** (baseline).

> **Waived from the "0 known-broken flows" denominator (triage 2026-07-10, D-3):**
> Inventory FR-24 (Trends) + FR-25 (Cost) — confirmed-BROKEN `.coming-soon` stubs at
> `inventory.html:993-999`. Waived as **unbuilt-future** (charts are net-new feature work,
> not hardening); they ship as-is and are excluded from the Engineering-KR denominator by
> explicit operator sign-off. Purchasing FR-18 (History) is **not** waived — it's a real
> stub of a shipped feature (backend endpoint exists) → test-repair/build WO in Activity 4.

---

## Standing method rules (apply to every card)

- **Two enumeration passes minimum** (first + cross-check) — recall ≥ 90%.
- **No orphan inputs** — every card's inputs trace to a produced upstream artifact.
- **Enumerate + mark ≠ fix** — PRD cards list/mark; WO cards fix.
- **Tool pin** — night-crew tracks `c55cbdd` (current `dev`). The prior `e4b43ba`
  freeze held the tool steady through one run while the design-change batch
  (PRD-verifier gate, cadence decision, roadmap-producer) was in flight; that batch
  has landed and been triaged, so the freeze is lifted and the pin advanced to
  current `dev` by operator decision (2026-07-10). E2E config support (`[e2e]`
  parsing, the runner, the `NIGHTCREW_ENV_URL` handoff) predates `e4b43ba`, so the
  E2E enablement does not depend on this re-pin.
