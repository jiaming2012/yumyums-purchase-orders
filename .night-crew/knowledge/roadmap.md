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
| `ops-confirm-absence` (FR-12 reject, NFR-3 photo-required) | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 2 BROKEN: FR-4, NFR-3; FR-12 confirm NEGATIVE) | `ops-hardening-prd` | Eng KR-1 |
| `users-confirm-absence` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 0 graduations, 16 stay UNPROVEN present-but-untested) | `users-hardening-prd` | Eng KR-1 |
| `onboarding-confirm-absence` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 1 BROKEN: NFR-5 video-led reopen no-op; 10 UNPROVEN; FR-16 waiver recommended) | `onboarding-hardening-prd` | Eng KR-1 |
| `purchasing-confirm-absence` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 0 graduations, 18 stay UNPROVEN; FR-18 remains only BROKEN; D-1 honored; FR-13 inline mark reconciled) | `purchasing-hardening-prd` | Eng KR-1 |
| `inventory-confirm-absence` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 0 graduations, 19 stay UNPROVEN; FR-24/25 remain waived-BROKEN; NFR-1 double normalization-gap flagged for WO) | `inventory-hardening-prd` | Eng KR-1 |

## Activity 3 — Audit the WORKING (non-vacuous test check)

Spot-audit every flow marked WORKING that its test actually asserts (no
`test.skip`, no guard-return). A vacuous test drops the flow to UNPROVEN.

| Card | Status | Depends on | KR |
|---|---|---|---|
| `ops-test-audit` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 0 drops, all 10 WORKING non-vacuous; FR-15 photo/builder-UI coverage gap noted for downstream WO) | `ops-hardening-prd` | QA KR-1 |
| `users-test-audit` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 0 drops, 10 WORKING non-vacuous; stale-test fold-in confirmed for FR-16/17 test-repair WO) | `users-hardening-prd` | QA KR-1 |
| `onboarding-test-audit` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 0 drops, 23 WORKING non-vacuous; 6 conditional-skip guard flags for the test-hardening WO) | `onboarding-hardening-prd` | QA KR-1 |
| `purchasing-test-audit` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 1 drop: FR-7 → UNPROVEN (vacuous generic-content tail); 6 WORKING confirmed non-vacuous) | `purchasing-hardening-prd` | QA KR-1 |
| `inventory-test-audit` | **DONE** ✅ signed 2026-07-13 (overnight-20260712; G6-passed → 0 drops, 19 WORKING non-vacuous; Go DB-guard = env-not-vacuous; FR-2 ~40-guard cleanup noted) | `inventory-hardening-prd` | QA KR-1 |

> ✅ **Activities 2 & 3 complete — all 10 cards signed at morning triage 2026-07-13 (overnight-20260712).**
> Every card G6-passed; docs-only run (footprint 100% under `.night-crew/`). Net movement:
> **3 new confirmed-BROKEN** (Ops FR-4, Ops NFR-3, Onboarding NFR-5) + Purchasing FR-18 (re-confirmed)
> = **Eng KR-1 known-broken denominator now exactly 4 built flows**; **1 WORKING→UNPROVEN drop**
> (Purchasing FR-7, vacuous test — first QA KR-1 data point). All BROKEN citations re-verified at
> cited lines. Fix-cards + test-hardening notes itemized in `BACKLOG.md`. Triage resolutions in
> `ledger.md` (2026-07-13, T-7/T-8). **Activity 4 (prove & fix) now unblocks** — it needs localhost
> Postgres + the E2E suite armed first (the standing DB flag bites there).

## Activity 4 — Prove & fix the UNPROVEN (the WO backlog) · *bulk delivery*

Per the sign-off policy: **test-only WO first** (write the red-first assertion);
graduate to a **fix WO only if the test goes red**. Operations alone = 17 UNPROVEN.

**First build slate (`slate-20260714`, serial, 6h) — Wave-0 infra + the 4 confirmed-BROKEN fix-cards:**

| Card | Status | Depends on | KR |
|---|---|---|---|
| `hq-infra-docker-standardize` (Wave 0) | **DONE** ✅ signed 2026-07-14 (overnight-20260714; G6-PASS — local Docker DB → pg16, 70 migrations clean, `test:all`/`bdd` repointed off remote) | — | infra / ledger T-9 |
| `ops-fr4-no-enforcement` (fix) | **DONE** ✅ signed 2026-07-14 (overnight-20260714; G6-PASS, red-first — yes/no "No" corrective gate front+back) | ops confirm-absence | Eng KR-1 |
| `ops-nfr3-photo-required` (fix) | **DONE** ✅ signed 2026-07-14 (overnight-20260714; G6-PASS, red-first — required-photo gate front+back; resubmit-context case deferred → BACKLOG F-1) | ops confirm-absence | Eng KR-1 |
| `purchasing-fr18-history` (fix) | **DONE** ✅ signed 2026-07-14 (overnight-20260714; G6-PASS, red-first — History tab built, `renderHistory` + `GET /shopping/history`) | purchasing confirm-absence | Eng KR-1 |
| `onboarding-nfr5-video-reopen` (fix) | **DONE** ✅ signed 2026-07-14 (overnight-20260714; G6-PASS, red-first — video-led reopen/reject reverts to active; FR-9 + FR-15) | onboarding confirm-absence | Eng KR-1 |

> ✅ **Eng KR-1 known-broken denominator: 4 → 0.** All four confirmed-BROKEN built flows fixed +
> red-first guarded (Ops FR-4, Ops NFR-3, Onboarding NFR-5, Purchasing FR-18). What remains excluded
> from the denominator is the two operator waivers only: Inventory FR-24/25 (D-3, unbuilt-future) and
> Onboarding FR-16/NFR-4 (D-5, env-gated). Signed at triage 2026-07-14 (ledger T-9).

**Test-only prove-UNPROVEN bulk (2 landed as stretch on slate-20260714; the full sweep is `slate-20260715`):**

| Card | Status | Depends on | KR |
|---|---|---|---|
| `purchasing-fr7-retest` (stretch) | **DONE** ✅ signed 2026-07-14 (overnight-20260714; G6-PASS — Shopping-tab empty+populated render proven; vacuous tautology replaced) | purchasing test-audit | QA KR-1 |
| `users-stale-e2e-repair` (stretch) | **DONE** ✅ signed 2026-07-14 (overnight-20260714; G6-PASS — 2 dead Access-tab tests repointed `#t3/#s3`→`#t2/#s2`; `users.spec.js` 17/2 → 19/0) | users test-audit | QA KR-1 |

**Prove-UNPROVEN sweep — `slate-20260715` (16 cards, 5 concurrent tracks, ~78 flows). Signed 2026-07-14; run branch `overnight-20260715`. Coarse `ops-prove-unproven`/`<app>-prove-unproven` placeholders fanned out below.**

| Track | Cards | Flows | Status | KR |
|---|---|---|---|---|
| A · Operations | `ops-prove-checklists` · `ops-prove-approvals` · `ops-prove-builder` · `ops-prove-cross` | 15 | **DONE** ✅ signed 2026-07-15 (4/4 cards G6-PASS; all GREEN; PARK NFR-5/NFR-2-PUT/NFR-7-draft) | Delivery/Eng/QA |
| B · Purchasing | `purchasing-prove-order` · `purchasing-prove-po-approval` · `purchasing-prove-shopping` · `purchasing-prove-state-auth-scheduler` | 18 | **DONE** ✅ signed 2026-07-15 (4/4 cards G6-PASS; all GREEN + 3 Go units; PARK FR-19/20/21/22 crons — clock-seam) | Delivery/Eng/QA |
| C · Users | `users-prove-security` · `users-prove-ui-access` | 16 | **DONE** ✅ signed 2026-07-15 (2/2 cards G6-PASS; NFR-1..5 + 9 UI-access flows GREEN) | Delivery/Eng/QA |
| D · Onboarding | `onboarding-prove-assignments` · `onboarding-prove-progress` | 10 | **DONE** ✅ signed 2026-07-15 (2/2 cards G6-PASS; GREEN; PARK FR-18 thumbnail; UNTESTABLE FR-28 re-seed) | Delivery/Eng/QA |
| E · Inventory | `inventory-prove-purchases` · `inventory-prove-stock` · `inventory-prove-setup` · `inventory-prove-recipes-cross` | 19 | **DONE** ✅ signed 2026-07-15 (4/4 cards G6-PASS; PRIORITY NFR-8/Stock proved WORKING; NFR-1 RED→fixed; PARK FR-27 photo) | Delivery/Eng/QA |

> ✅ **Prove-UNPROVEN sweep complete — 16/16 cards G6-PASS, signed morning-triage 2026-07-15**
> (merged `overnight-20260715` → `dev` `--no-ff`). ~52 UNPROVEN flows now carry real red-first
> assertions. The headline finding inverts the forecast: scoping expected ~34–40 RED; **actual =
> exactly 1** (Inventory NFR-1 item-edit + confirm-vendor normalization) — the backlog was
> **untested, not broken**. That one RED was **fixed the same night** (Eng KR-1 +1 → 0; 2-line
> `internal/inventory/handler.go`). 11 flows PARKED honestly + 1 UNTESTABLE, all visible in the
> suite. 3 future fix-WOs (cron clock-seam, photo-S3 harness, offline-IndexedDB harness) → `BACKLOG.md`
> for the planners. Triage resolutions: `ledger.md` T-11/T-12. See `reference/slate-20260715.md`.

## Activity 5 — Cycle gate (closeout) · *serializes last*

The OKR closeout, across all five apps.

| Card | Status | Depends on | KR |
|---|---|---|---|
| `cycle-gate` | **DONE** ✅ attested 2026-07-16 (overnight-20260716; 3 cards G6-equiv PASS — suite-baseline · attestation · scorecard; **ATTEST & WAIVE** per operator 2026-07-15) | all Activity-4 cards | Eng KR-1/2, QA KR-1/2/3 |

> ✅ **Activity 5 complete — CYCLE GATE ATTESTED (PASS). Milestone boundary reached.**
> Fanned into 3 read-only closeout cards, serial. **Scorecard: 6 PASS · 2 PARTIAL · 1 WAIVED**
> (see `reference/cycle-closeout-20260716.md`). Attested: 0 known-broken flows (built 4→0 +1 NFR-1;
> §1 confirms no repaired flow regressed) · every repaired flow red-first · median WO cycle time
> baseline recorded (**N=23, 22m28s**; serial ~19.4m / concurrent ~23.4m).
> **Two criteria formally WAIVED** and carried to the next cycle's roadmap (extends D-3/D-5):
> (a) "full suite green / 0 pre-existing reds" → attested substitute **"0 new uncategorized reds vs
> the documented ~37–41 flaky baseline"** — full suite ran **387 pass · 38 fail · 0 flaky · 6 skip**,
> all 38 categorized, 0 uncategorized, **PARK trigger did not fire** (`DECISIONS-NEEDED.md` empty);
> `playwright.config.js:29` blocks SWs so a clean suite is structurally unreachable. (b) "vacuous
> 23→0" → ~4–5 rewritten, ~18 remainder deferred (`BACKLOG.md`). **Next move on this clean gate:**
> `/nc-morning-triage` then `/nc-okr-session` (consume carried-forward backlog) — **not**
> `/nc-slate-plan`.

Gate = **0 known-broken flows** · **full E2E suite green on localhost Postgres, 0
pre-existing reds** · **vacuous tests 23 → 0** · **every repaired flow carries a
red-first proof** · **median WO cycle time recorded over ≥5 WOs** (baseline).

> **Waived from the "0 known-broken flows" denominator (triage 2026-07-10, D-3):**
> Inventory FR-24 (Trends) + FR-25 (Cost) — confirmed-BROKEN `.coming-soon` stubs at
> `inventory.html:993-999`. Waived as **unbuilt-future** (charts are net-new feature work,
> not hardening); they ship as-is and are excluded from the Engineering-KR denominator by
> explicit operator sign-off. Purchasing FR-18 (History) is **not** waived — it's a real
> stub of a shipped feature (backend endpoint exists) → test-repair/build WO in Activity 4.

> **Waived from the "0 known-broken flows" denominator (triage 2026-07-13, D-5):**
> Onboarding FR-16 (video presign→PUT→FFmpeg transcode/thumbnail) + NFR-4 (`503
> video_storage_not_configured` fallback). **Fully implemented** (`handler.go:540-640`,
> `video.go:22-206`) — **not broken**, only untestable in E2E without DO Spaces creds + an
> `ffmpeg` binary. Both stay UNPROVEN and are excluded from the Engineering-KR denominator by
> explicit operator sign-off (parallel to D-3). **Waive-now-but-preserve:** a BACKLOG item to
> stand up a Spaces+ffmpeg E2E fixture and prove them is queued for when creds are available.
> Contrast Onboarding NFR-5 (video-led reopen no-op) — **not** waived; a confirmed BROKEN in a
> shipped flow → Activity-4 fix-card.

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
