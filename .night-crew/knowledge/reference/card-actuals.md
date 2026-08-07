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

## Run: overnight-20260714 (Activity 4 — first APP-CODE + red-first + E2E slate)

> **What one "card" covers here (new shape — the first app-code cards):** a fresh IMPLEMENTER
> subagent in a per-card git worktree brings up the ephemeral Docker stack
> (`docker-compose.nc.yml`, postgres:16, Docker-assigned port, `NIGHTCREW_ENV_URL`), writes the
> red-first regression test, captures it FAILING against the unfixed build, applies the fix,
> **rebuilds the app image**, greens the new test, runs the affected seam subset for no-new-reds,
> and tears the stack down `--volumes` — then a SEPARATE fresh G6 subagent reviews the diff +
> red→green evidence, then the ORCHESTRATOR merges (squash → `task sw` if HTML → roadmap flip →
> atomic commit → worktree remove). Serial, one ephemeral env at a time.
> **Times below are subagent wall-clock.** "Implement" = the full implementer leg (env up +
> baseline + test + up to 2 Docker builds + fix + green + no-new-reds + teardown). "G6" = the
> reviewer leg. Orchestrator merge (squash + `node build-sw.js` + roadmap + commit + worktree
> cleanup) ran ~1.5–3 min/card and is NOT separately instrumented.

| Card | Type | Slate est. | Implement (agent) | G6 | Verdict | Notes |
|---|---|---|---|---|---|---|
| `hq-infra-docker-standardize` (Wave 0) | infra | 15–30 min | **6m36s** | 0m26s | PASS | Taskfile pg13→16 + remote→local; no app code; box env restored |
| `ops-fr4-no-enforcement` | app fix (front+back) | 45–75 min | **17m23s** | 1m14s | PASS | yes/no "No" corrective gate; RED captured; 2 submit entrypoints |
| `ops-nfr3-photo-required` | app fix (front+back) | 45–75 min | **23m23s** | 2m32s | PASS | photo gate; backend resubmit case DEFERRED (follow-up) |
| `purchasing-fr18-history` | app build (frontend) | 60–90 min | **27m12s** | 1m11s | PASS | net-new UI + 4-test rewrite + API fixture (no SQL) |
| `onboarding-nfr5-video-reopen` | app fix (backend) | 60–90 min | **21m27s** | 0m56s | PASS | video-part resolution; covers FR-9 + FR-15; seed had fixture |
| `purchasing-fr7-retest` (stretch) | test-only | 30–45 min | **12m37s** | 0m53s | PASS | proved FR-7 WORKING; old test was baseline-RED, not just vacuous |
| `users-stale-e2e-repair` (stretch) | test-repair | 30–45 min | **6m28s** | 0m49s | PASS | #t3/#s3→#t2/#s2 + var rename; users.spec 17/2 → 19/0 |
| **Total (7 cards, serial)** | — | 3.25–5.5h + stretch | **~115 min implement** | **~8 min G6** | 7 PASS / 0 park | + ~14 min orchestrator merge ≈ **~2h20m active** |

### Third-slate observations (the first app-code data — seed for future prove/fix slates)

- **App-code + E2E cards land ~2–3× the docs-card wall-clock, and MUCH closer to slate estimate
  than the doc runs did.** Docs-only cards came in 6–10× under estimate; these app-code cards came
  in **~2–4× under** (e.g. ops-nfr3 23m vs 45–75m est). The Docker-build + E2E wall-clock is real
  and non-compressible — it doesn't parallel-read away. **Carry the ~15–30 min/app-fix-card shape
  forward**, not the doc-card minutes.
- **The two Docker image builds per card dominate the floor.** Every fix-card pays: build#1 (unfixed,
  for baseline + RED) + build#2 (rebuild with the fix, for GREEN). Frontend changes still require a
  full image rebuild (frontend is embedded in the Go binary). This is why the frontend-only
  `purchasing-fr18` (27m) ran LONGER than the backend `onboarding-nfr5` (21m) — net-new UI + a
  4-test rewrite + seeding a completed shopping list, not build count.
- **Fixture availability is a bigger time lever than logic subtlety.** The subtle-logic
  `onboarding-nfr5` (a 30-line `db.go` branch) ran FASTER than `purchasing-fr18` (77-line new UI) —
  because the seed already carried the video fixture, while fr18 had to build the render AND author a
  completed-list fixture. Cards whose fixture pre-exists in the seed are cheapest; verify seed
  coverage when sizing.
- **G6 stayed cheap and flat (~0.5–2.5 min) even on app code.** Reviewing a diff + red→green evidence
  is fast; the one longer G6 (ops-nfr3, 2m32s) was the double-JSON-encoding peel + the deferred-gate
  assessment. Budget G6 ≈ 1–3 min/card regardless of card type.
- **Zero REVISE, zero park, zero Docker crashes** across 14 subagent legs — the 2026-07-04 Docker
  instability did NOT recur this run (v20.10.14, serial one-env-at-a-time). The ephemeral env held.
- **Red-first was load-bearing and honest on all 5 fix/prove cards:** each new test was captured
  FAILING against the unfixed build before the fix (ops-fr4 submit-succeeded, ops-nfr3
  submission-created, onboarding-nfr5 stayed-complete, users-stale locator-timeout), then flipped
  green — no test passed without its fix. `purchasing-fr7` inverted cleanly (proved WORKING; the old
  test was actually broken, targeting a nonexistent `#shopping-content`).
- **"No-new-reds vs baseline" is the right gate, not "clean suite."** HQ's ~37–41 documented flaky
  reds (offline-sync, tab-persistence, cross-test DB-pollution) shifted run-to-run; every card
  confirmed its flips in isolation. No card was judged on a globally-green suite.

## Run: overnight-20260715 (Activity 4 — the CONCURRENT prove-UNPROVEN sweep, all 5 apps)

> **What one "card" covers here:** same shape as overnight-20260714 (fresh implementer subagent →
> ephemeral pg16 Docker stack → red-first assertions → G1–G4 → separate fresh G6 → orchestrator
> merge), but **CONCURRENT**: 5 tracks, the orchestrator held a **rolling 3 in-flight implementers**
> (a deliberate throttle below the slate's authorized 5 — Docker 20.10.14, prudent strain mgmt).
> 16 prove cards (test-only) + 1 graduated fix (the only RED). **Times = subagent wall-clock.**
> "Impl" = full implementer leg (env up + build + red-first + classify + no-new-reds + teardown).

| Card | Type | Slate est. | Impl (agent) | G6 | Verdict | Notes |
|---|---|---|---|---|---|---|
| ops-prove-checklists (A1) | test-only | ~20m | 13m08s | 1m35s | PASS | FR-6/7/8 GREEN |
| ops-prove-approvals (A2) | test-only | ~25m | 10m00s | 1m54s | PASS | +rewrote vacuous flag test |
| ops-prove-builder (A3) | test-only | ~30m | 25m51s | 1m37s | PASS | FR-16/17/18 GREEN |
| ops-prove-cross (A4) | test-only | ~45m | 25m56s | 1m50s | PASS | 5 GREEN + 3 PARK (S3/IndexedDB) |
| purchasing-prove-order (B1) | test-only | ~30m | 22m28s | 2m44s | PASS | FR-1/6 predicted-RED → GREEN |
| purchasing-prove-po-approval (B2) | test-only | ~35m | 17m04s | 2m02s | PASS | FR-14/15/16 predicted-RED → GREEN |
| purchasing-prove-shopping (B3) | test-only | ~25m | 32m26s | 1m39s | PASS | +rewrote vacuous tail (1 flaky, flagged) |
| purchasing-prove-state-auth-scheduler (B4) | test + new Go | ~50m | 15m22s | 1m43s | PASS | NFR-1/2/FR-23 GREEN + 3 Go units; 4 crons PARK |
| users-prove-security (C1) | test-only | ~30m | 8m26s | 2m21s | PASS | NFR-1..5 GREEN (fastest card) |
| users-prove-ui-access (C2) | test-only | ~35m | 9m03s | 1m59s | PASS | 9 flows GREEN |
| onboarding-prove-assignments (D1) | test-only | ~15m | 23m33s | 1m38s | PASS | FR-26 GREEN |
| onboarding-prove-progress (D2) | test-only | ~40m | 27m04s | 2m08s | PASS | 5 GREEN + FR-18 PARK + FR-28 UNTESTABLE |
| inventory-prove-purchases (E1) | test-only | ~20m | 28m54s | 2m32s | PASS | FR-3/5 predicted-RED → GREEN |
| inventory-prove-stock (E2) | test-only | ~30m | 23m24s | 2m16s | PASS | PRIORITY FR-12/13/14 → all GREEN |
| inventory-prove-setup (E3) | test-only | ~45m | 19m21s | 1m28s | PASS | 5 GREEN + FR-27 photo PARK |
| inventory-prove-recipes-cross (E4) | test-only | ~35m | 32m44s | 2m12s | PASS | 5 GREEN + **NFR-1 RED** (the only one) |
| inventory-nfr1-normalize-fix (FIX) | app fix (Go) | (graduated) | 28m19s | 1m35s | PASS | 2-line handler.go; NFR-1 RED→GREEN |
| **Total (17 cards, ROLLING-3 CONCURRENT)** | — | serial ~8h30m | — | **~31m G6** | 17 PASS / 0 park-card / 0 REVISE | **~2h45m orchestrator wall-clock** |

### Fourth-slate observations (the first CONCURRENT run — seed for future concurrent slates)

- **Concurrency ≈ 3.5× throughput, at ~1.5–2× per-card latency.** Test-only cards ran **8–33 min**
  here (median ~23m) vs. **6–13 min** on the prior SERIAL run — the same card is slower under load
  because 3 concurrent Docker builds + 3 Playwright runs share CPU/IO. But total wall-clock was
  **~2h45m for 17 cards** vs. the slate's serial estimate of **~8h30m**. **Size concurrent slates on
  total-wall = (Σ impl)/3 + G6/merge tail, NOT on per-card minutes.** The per-card estimate inflates
  under concurrency; don't read a 33-min card as "slow."
- **G6 stayed cheap and flat (~1.5–2.7 min) regardless of concurrency** — it's a fresh-context diff
  read, unaffected by Docker load. Budget G6 ≈ 2 min/card. One G6 mutation-verified a Go decision-rule
  test (inverted the rule → RED) — worth the ~2 min.
- **The forecast was wrong in the RIGHT direction: UNPROVEN ≈ UNTESTED, not BROKEN.** Scoping forecast
  ~34–40 of ~78 RED; **actual = 1**. Every "likely RED" prediction (FR-1/6/14/15/16, FR-3/5,
  FR-12/13/14, NFR-8, FR-18, FR-10/14) proved GREEN — the handlers existed and were correct, the
  flows were merely untested. **Discount "likely RED" scoping guesses heavily for mature apps;** the
  real RED surfaced only where create/edit paths genuinely diverged (NFR-1 normalization).
- **Rolling-3 throttle held zero Docker incidents** across ~18 env cycles (17 cards, one card ran a
  2nd baseline stack) on Docker 20.10.14. Pre-warming the build cache once up-front made per-worktree
  builds seconds not minutes — **do both every concurrent run.** 4–5 concurrent is likely safe next
  time given this margin.
- **⚠ Shared-git-stash is a real concurrency hazard.** Worktrees share `refs/stash`; one early
  `stash pop` pulled a concurrent card's WIP into the wrong tree (recovered, no loss). Fixed mid-run
  by forbidding `git stash` in the implementer runbook (baseline-first / `cp` instead) — zero further
  incidents. **This rule must live in the standing runbook for every concurrent run.**
- **Only 1 card wrote Go (B4's new `*_test.go`) + the 1 fix (handler.go).** Adding `*_test.go` is a
  clean way to prove pure-logic decision rules (parseCutoffTime, isAdmin, cutoff-rule) without an
  E2E stack — but crons that read `time.Now()` inline are NOT unit-testable without a production
  clock-seam (→ PARK + future WO, not a tonight fix).

## Run: overnight-20260716 (Activity 5 — the CYCLE-GATE closeout; read-only, one full-suite run)

> **New card shape:** read-only closeout, no app code. Card 1 = orchestrator-run full-suite baseline
> on an isolated pg16 (Go units + Playwright + a fix-adjacent isolation re-run to categorize reds);
> Cards 2 & 3 = fresh read-only audit subagents (PRD/git attestation; metric + KR scorecard) returning
> their section as a report → the orchestrator assembled the closeout doc, flipped the roadmap, and
> appended the ledger. **Serial.** No G6 leg per card (the suite run IS the evidence; the orchestrator
> cross-verified — independently re-checked the median, spot-checked fix commits in `git log`, and ran
> the 7-test isolation re-run to prove the red categorization).

| Card | Type | Slate est. | Actual (wall-clock) | Verdict | Notes |
|---|---|---|---|---|---|
| `cycle-gate-suite-baseline` (Wave 0, gating) | read-only + 1 stack run | ~30–40 min | **~20 min** | ATTEST PASS | isolated pg16 (:5455, no touch to :5432); Go 5 ok + 1 env-gated red; Playwright **387 pass · 38 fail · 0 flaky · 6 skip** in **12.7m**; +~3m 7-test isolation re-run; all 38 reds categorized, 0 uncategorized → no PARK |
| `cycle-gate-attestation` | read-only audit (subagent) | ~15–20 min | **~3m26s** | PASS | 4→0 built-broken; 1 git-verifiable red→green pair (NFR-1), 4 on ledger record (squash caveat surfaced) |
| `cycle-gate-scorecard` | read-only compute (subagent) | ~15–20 min | **~2m09s** | PASS | median WO cycle time N=23, 22m28s; 9-KR scorecard 6 PASS · 2 PARTIAL · 1 WAIVED |
| **Total (3 cards, serial)** | — | ~1h58m (critical path) | **~26m subagent/suite wall-clock** + orchestrator assembly | 3 attest/PASS / 0 park | + closeout doc + roadmap flip + ledger append |

### Fifth-slate observations (seed for future closeout/gate cards)

- **Closeout cards are audit-shaped, not build-shaped.** The two audit subagents came in **~2–3.5 min**
  — same order as the Activity-2/3 audit cards, and again **6–8× under** the slate's first-of-kind
  estimate. The cost center is the **one full-suite run** (Card 1, ~20m), which is real,
  non-compressible Playwright wall-clock (12.7m suite + Docker + Go + the isolation re-run). Size a
  gate/closeout night on **1 full-suite run (~15–25m) + a few-minute audit per evidence leg**, not on
  the build-card shape.
- **The isolation re-run is the load-bearing move on a gate.** `0 flaky` on the full run means every
  red was deterministic-within-run, so "flaky" couldn't be assumed — a 7-test fix-adjacent re-run on a
  fresh single-test DB was what actually separated cross-test pollution (1 greened alone) from
  structural/seed causes (6 reproduced). Budget one targeted isolation re-run whenever a full-suite
  gate must attest "0 new uncategorized reds" — it's the difference between an evidenced attestation
  and a hand-wave.
- **A bare `task test:all` reproduces the documented ~37–41 baseline (got 38).** The full suite seeds
  only self-seeded superadmins; the per-card domain fixtures/personas prior prove cards used are absent,
  so the reds concentrate in the data-dependent/persona guards + the SW-blocked/offline tests. This is
  exactly why the gate criterion is judged "no-new-reds-vs-baseline," not "clean suite" — the bare
  full-suite red count is a documented, stable property, not a regression signal.

## Run: overnight-20260717 (new cycle — editprop build + engine-trust + carried fixes + test-debt)

> **Card shape:** back to build cards (fresh implementer per card → ephemeral pg16 worktree →
> red-first → G1–G4 → separate fresh G6 → orchestrator merge), **SERIAL** (one in-flight card at a
> time), 9 cards. **⚠ Per-card wall-clock was NOT separately instrumented this run** — the closeout
> HANDOFF recorded verdicts + evidence but only *qualitative* sizing (below), not per-card minutes.
> The times column records what the HANDOFF asserts; treat the ranges as sizing signal, not
> measured actuals. (Fix forward: future runs should emit per-card impl/G6 wall-clock like the
> 07-14/07-15 tables so this ledger stays measured, not narrated.)

| Card | Type | Class | Passes | G6 verdict | Sizing signal (HANDOFF — not instrumented) |
|---|---|---|---|---|---|
| W-1 `editprop-stable-field-identity` | app fix + Go/E2E (backend + runner) | XL (first-of-kind structural) | — | PASS | XL structural; `replaceTemplate` deleted, 422 path + cross-device E2E |
| W-2 `editprop-broadcast-rerender` | app fix + Go/E2E (front + back sync) | XL | 2 (1 REVISE→PASS) | PASS | **longest**; 5 sub-behaviors; INV-6 sent back into scope (revision) |
| W-3 `editprop-convergence-matrix` | E2E matrix (two-device) | XL | 2 (G6 FAIL-REVISE→PASS) | PASS | **longest**; 36-cell two-device matrix; revision landed denominator cell + de-flaked |
| W-4 `engine-approval-feedback-loud` | app fix (Go) | S (red-first fix) | — | PASS | ~15–30m class; 200→500 red→green |
| W-5 `ops-nfr3-resubmit-photo-gate` | app fix (Go, carried) | S | — | PASS | ~15–30m class; 201→400 both submit paths |
| W-6 `engine-conflict-refetch` | app fix (`sync.js`) | S | — | PASS | ~15–30m class; deterministic 3/3; 409 double-wrap fix |
| U-1 `users-s3-orphan-cleanup` | hygiene (no behavior change) | XS | — | DONE (inline verify) | trivial; dead `#s3` div removed, no red-first per spec |
| T-1 `carried-fix-wos-sweep` | test + new Go (clock seam) | M | — | PASS | clock seam + 13 mock-time cron subtests; unblocks Purchasing FR-19–22 |
| T-2 `vacuous-tests-18-to-0` | test-only (16 conversions) | M | — | PASS | 18 = 16 converted + 2 already-hardened; retires waiver #2 |
| **Total (9 cards, SERIAL)** | — | — | 2 cards took 1 revision loop each | **9/9 G6-verified · 0 park · 0 footprint breach** | 9 atomic commits; per-card minutes not captured |

### Sixth-slate observations (build cards return; carry the *shape*, and fix the instrumentation gap)

- **Instrumentation regressed vs 07-14/07-15.** Those runs tabled per-card impl + G6 wall-clock;
  this run's HANDOFF recorded only verdicts + qualitative sizing ("app-fix + red-first cards ran
  ~15–30m impl; the two XL editprop cards ran longest and each needed one revision loop"). The
  ledger's whole value is *measured* actuals — future build runs should re-adopt the per-card
  timing table so estimates don't drift back onto narration.
- **First-of-kind structural cards carry a real revision cost — budget it.** Both XL editprop
  cards (W-2 broadcast, W-3 matrix) took exactly one G6-driven REVISE loop before PASS, and both
  revisions were *in-scope work the first pass under-delivered* (W-2's INV-6 discard warning
  parked as "out of footprint" then sent back; W-3's parked denominator cell + two-device
  de-flake), NOT bookkeeping. Contrast Activity-1's REVISE loops, which were tally/status
  bookkeeping. **On a first-of-kind structural card, budget one substantive revision round.**
- **Small red-first fix cards stayed the cheap, reliable population** (W-4/W-5/W-6, all clean
  first-pass PASS, ~15–30m class) — the same profile as 07-14's fix cards. Serial dispatch on a
  9-card slate with 3 XL cards is a full night; the XL cards dominate the critical path.
- **The two-device WS-convergence E2E cells are timing-sensitive** — green under `retries:1`, ~3/6
  under no-retry (F-A). Any future night leaning on that suite as a *hard no-retry gate* must first
  land `editprop-convergence-cell-hardening` (scheduled at triage). This is a distinct population
  from clean-path fix cards — treat convergence-hardening as its own size class when it's planned.

## Run: overnight-20260718 (Activity 6 final card — convergence-cell de-flake + conflict coverage)

> Single-card slate, serial by definition. **Instrumentation gap fixed** — the run recorded
> harness-measured per-card wall-clock (the 07-17 note's ask). Merged `6291ef2` at triage.

| Card | Type | Class | Impl (measured) | G6 (measured) | Merge | Verdict | Notes |
|---|---|---|---|---|---|---|---|
| `editprop-convergence-cell-hardening` | test-only (de-flake + conflict coverage) | convergence-hardening (own class) | **~73m** (4,387,969 ms) | **~16m** (964,145 ms) | ~2m | **PASS** | slate est. ~60–90m impl — inside band. Half 1 de-flake (survivalCell deterministic waits) + Half 2 W-6b conflict coverage (4 types landed, 2 fail-note types parked footprint-blocked → D-1). `sync.js` untouched. 0 footprint breach. |

### Seventh-slate observations (convergence-hardening as its own size class — now measured)

- **Convergence-hardening ran ~73m impl / ~16m G6 — the de-flake *proof* is the non-compressible
  cost, exactly as the 07-17 note predicted.** Impl wall was dominated not by one red→green but by
  the repeated `--retries=0` streak runs (10/10 isolated + 8 whole-describe under-load repros)
  needed to *demonstrate* zero-flake. This confirms the sizing signal from 07-17: **treat
  convergence/flake-hardening as its own size class (~60–90m impl), separate from clean-path fix
  cards (~15–30m)** — proving determinism costs repeated Playwright wall-clock, not a single fix.
- **G6 wall (~16m) included fighting a stale foreign server on `:8199` that `reuseExistingServer`
  latched onto** — resolved by running G6 against its own ephemeral env. A harness-env hazard, not
  a code cost; note it so future G6 legs on this stack budget for env isolation up front.
- **A test-only footprint held under pressure.** The 2 fail-note conflict types were genuinely
  reachable only via a production `applyOp` change; the implementer parked rather than breach the
  declared footprint, and G6 confirmed the block was real. Parks-for-footprint are legitimate and
  should be read as discipline, not incompleteness — the residual rode to BACKLOG (D-1) intact.

## Run: overnight-20260719 (Activity 8 — the CYCLE-GATE closeout; read-only, matches the 07-16 class)

> 3 read-only cards + orchestrator closeout, serial, one isolated pg16. **Instrumentation recorded**
> (per-card wall-clock below). Gate **PASS attested**; scorecard 11 PASS · 2 PARTIAL · 2 PENDING ·
> 1 N/A. Evidence: `reference/cycle-closeout-20260719.md`.

| Card | Type | Valid wall-clock | Verdict | Notes |
|---|---|---|---|---|
| `cycle-gate-attestation` (Card 2) | read-only subagent | **~4m31s** (270,482 ms) | done | 4/4 audit areas verified; corrected slate's dangling pre-squash SHAs → landed squashes |
| `cycle-gate-scorecard` (Card 3) | read-only subagent | **~4m11s** (250,631 ms) | done | 16-KR table; median not computable (07-17 gap) → Delivery KR4 PARTIAL, none fabricated |
| `cycle-gate-suite-baseline` (Card 1) | orchestrator-run suite | **~34.6m** valid | ATTEST (0 uncategorized) | migrate+Go ~2m · Playwright 16.3m (450/1/0/6) · isolation re-run 2.5m · convergence 3×4.6m. +~10–15m WASTED on 2 invalid attempts (below) |

### Eighth-slate observations (second gate closeout — carry the shape, and STANDARDIZE the env fix)

- **The gate class holds at ~35–45m suite wall-clock + ~10m assembly** (07-16 was ~20m for a
  38-red suite; this run's suite was cleaner but added the convergence 3× streak (~14m) + isolation
  re-run — so total suite wall grew even as red-count fell). Size future gates on **1 full suite +
  a convergence-streak proof + 1–2 targeted isolation re-runs**, not on red-count.
- **The `:8199` `reuseExistingServer` foreign-server hazard bit AGAIN — this is now the 2nd run to
  lose wall-clock to it** (07-18 G6 hit the same latch). The fix is cheap and must be **standard**:
  run any suite/G6 leg with **`CI=1`** (forces `reuseExistingServer:false` → own webServer + teardown)
  **and** provision the isolated pg16 with an **explicit pre-migration boot** before Go units.
  Both were missing from this orchestrator's first two passes → ~10–15m wasted. Bake into the gate
  run-mechanics so the 3rd gate doesn't re-learn it. *(Recorded as a run-mechanics fix-forward; the
  suite itself, once provisioned right, was clean.)*
- **A cleaner suite than the sizing basis.** The documented ~37–41 flaky/data-dependent baseline
  (07-16) is **gone** — this run saw exactly 1 red (cross-test pollution, isolation-green) and Go
  exit-0. `42eeb39`'s "425/6/0 deterministic" + this cycle's editprop/vacuous work paid down the
  baseline. **Implication for the next gate:** the sizing basis for "suite reds to categorize" is
  now ~1–5, not ~40 — the categorization leg is fast; the streak-proof leg is the cost.
- **Honesty over a clean scorecard.** Eng KR5 was pre-computed PASS by Card 3 but downgraded to
  PARTIAL once §1 showed literal `task test` exit-1 (1 pollution red). The gate refused the
  definitional carve-out that would have declared waiver #1 formally retired — waiver #1 is carried,
  reduced 38→1. This is the pattern to keep: **let §1's real result reconcile the pre-computed
  scorecard, downgrade in the open.**

## overnight-20260721 (Trust track + design draft; serial dispatch, per-card worktree + fresh G6, ephemeral pg16 per leg)

| Card | Class | Implement | G6 | Land (merge+flip) | Card cycle |
|---|---|---|---|---|---|
| `convergence-matrix-systematic` (A1) | XL (test matrix) | 101m15s ⚠ | 9m15s | ~1m | ~111m |
| `sync-pkg-unit-coverage` (B1) | S–M (Go unit) | 8m11s | 4m27s | <1m | ~13m |
| `prove-surface-design-draft` (C1) | M (design doc) | 7m30s | 4m49s | <1m | ~13m |
| `waiver1-isolation-fix` (A2) | S + exit-0 proof | 88m00s | 57m34s | <1m | ~146m |

- **A1 ⚠ — ~25–30m of the impl leg was orchestration stall, not work:** the implementer twice
  backgrounded its suite runs and had to be resumed. Productive wall ~70–75m — INSIDE the slate's
  70–90m estimate. Sizing lesson: the XL estimate held; the stall is a run-mechanics defect (brief
  rule now: never background; foreground ≤10m, detach+`tail --pid` beyond).
- **A2 — the outlier is the PROOF, not the fix:** the fix itself was small; the card class
  "S + full-suite exit-0 proof" costs 2 complete `task test` runs in G6 alone (18.8m + 19.1m).
  Treat "literal exit-0 headline" as its own ~60m G6 class; that spend is what surfaced the
  successor intermittent (§B2 → `replay-fetchstorm-gate`).
- **Clean-path population:** B1/C1 (~13m card cycles) extend the S–M clean-path band established
  by prior slates; no repair cycles this run (0 parks).
- Run total 08:27 → 13:10 (~4h43m) vs serial estimate ~175m + 30m closeout — overage is A1 stall
  + A2's double-suite G6, both now priced classes.

## overnight-20260722 (autonomous, CONCURRENT 2-track dispatch)

| Card | Class | Impl | G6 | Land | Cycle | Outcome |
|---|---|---|---|---|---|---|
| S1 `replay-fetchstorm-gate` | S-fix + de-flake proof | **47m23s** | **44m52s** | 0m24s | **~93m** | MERGED (PARTIAL — tail parked) |
| F1 `trends-spend-by-group-endpoint` | Go endpoint, S–M | **9m55s** | **11m14s** | — | **~21m** | **PARKED at G6** |
| F2 `cost-margin-endpoint` | Go endpoint, S–M | **12m23s** | **15m42s** | 5m31s (incl. revision) | **~34m** | MERGED |
| F4 `cost-tab-frontend` | net-new UI + states spec, M | **58m34s** | **47m55s** | 22m09s (incl. revision) | **~129m** | MERGED |

Run total **184m54s (~3h05m)** card time + ~13m quiet determinism streak + closeout.
Slate estimated ~3h40m–5h25m for the concurrent critical path; actual was under it **only because
two of five Track-F cards never ran** (F1 parked → F3 blocked; F5 dropped by budget).

**The estimate lesson — G6 was mispriced by an order of magnitude.** The slate budgeted G6 at
**2–3m** for endpoint cards and **2–3m** for tab cards. Actuals: **11m14s / 15m42s / 44m52s /
47m55s.** This is not overrun — it is what an adversarial gate costs when it does real work: every
G6 this run built its own fixtures and ran its own mutations, and three of four booted their own
database. **Reprice G6 for app-code cards at 15–45m**, and expect the UI/de-flake classes at the
top of that band. A slate that prices G6 at 2–5m will systematically under-budget its nights.

**Revision rounds are a real, recurring leg.** Two of three merged cards needed one (F2 5m31s,
F4 22m09s, both including merge). Budget a revision round for *any* card whose G6 can produce
in-footprint findings — not just first-of-kind cards, as slate-20260722 assumed for F5 alone.

**Clean-path vs repair populations:** F1 and F2 are the same size class (Go endpoint, S–M) with
near-identical impl times (9m55s / 12m23s) — the divergence is entirely in the gate and its
aftermath. Park cost is cheap when it happens at G6 (~21m total for F1) and expensive downstream
(F3 never ran).

**Concurrency note:** the 2-track dispatch worked mechanically (0 collisions, disjoint footprints,
one env per track) but **cost S1 its determinism proof for the entire run** — no quiet window
existed until Track F finished, and a `--retries=0` streak under load proves nothing. If a card's
deliverable *is* a flake proof, serialize it or reserve the quiet window up front.

---

## overnight-20260720c (autonomous, serial dispatch, per-card worktree + fresh G6)

Derived from `runs/2026-07-20c-autonomous/timings.log` (epoch-stamped) and the closeout HANDOFF.
Wall clock 407m (~6h47m) against a 6h10m–9h40m slate envelope — **in band**.

| Card | Class | Impl | G6 | Land | Cycle | Outcome |
|---|---|---|---|---|---|---|
| Wave 0 (`.gitignore` symlink fix) | XS | 1m | n/a | direct | 1m | DONE |
| F1 · trends-spend-by-group-endpoint | Go endpoint, M | 24m (+18m revision) | 11m → REVISE → confirm | 5m | 29m | **MERGED** |
| F3 · trends-tab-frontend | UI tab, L | 55m | 18m, PASS first pass | 18m | 73m | **MERGED** |
| F5 · inventory-tab-gating | authz, L | 60m (+50m revision) | 65m → **FAIL** → confirm | 8m | 124m | **MERGED** |
| D1 · syncspec-deflake | de-flake, L | 180m | diagnosis confirmed | — | 181m | **PARKED** (net-zero diff) |
| Follow-up sweep (`/ops` authz enumeration) | test+docs, M | 84m | inline | 3m | 84m | MERGED (attended, post-closeout) |

**G6 repricing held.** The slate repriced G6 at 15–45m per code card rather than 2–5m; actuals were
18m / 18m / 8m plus two revision rounds. **Both revision rounds were load-bearing** — one caught a
payroll-disagreeing rounding bug (`Σ(round) ≠ round(Σ)` on `NUMERIC(10,4)` prices), one caught a
live authentication bypass. Budgeting a revision for *every* card, not just first-of-kind ones, is
now evidence-backed twice over and should stand.

**Estimates ran long on every card that landed** — F1 52m vs 50–95m, F3 55m vs 100–150m, F5 110m vs
110–180m. F3 and F5 both credited prior cards for leaving reserved test blocks and delegation-safe
containers where the slate promised; the prep compounded.

**D1 is the inverse and the more useful datapoint: 180m against 80–125m.** The overrun is entirely
the honest path — the implementer made the two target tests green, ran the full suite, saw the fix
had *moved* the clobber rather than removed it, and reverted (~24m). Park-at-implement is expensive
in a way park-at-G6 is not (cf. F1's ~21m park last cycle). **Price de-flake cards assuming a
full-suite verification leg and a possible revert, not just the fix.**

**Timing numbers from D1 are weak evidence.** The orchestrator briefed D1 that the box would be
quiet; it was not — a concurrent night-crew run in a separate Claude session held the machine for
most of D1's window. Serial dispatch guarantees *this run* is serial, not that the machine is idle.
Any slate promising a "quiet box" deliverable needs a **measured** load precondition.

### Triage-day actuals (2026-07-21, attended) — a new population worth tracking

The morning triage itself ran long and is not currently budgeted anywhere:

| Leg | Wall | Note |
|---|---|---|
| Re-verify (build/vet/go test/full E2E) | ~55m | Two E2E runs: one died on the `:5432` default (audit surface #9, live) |
| Flake fix (red-first, fix, verify, **re-fix**) | ~70m | First fix was wrong — see below |
| DB separation (`hq_test_go`/`hq_test_e2e`) + concurrency proof | ~65m | Three full-suite runs to land it |

**Triage is not free and is not 15 minutes.** Three of the four full-suite runs this triage were
consumed by *harness* faults, not by reviewing the run's work. Budget attended triage at 2–4h when
the run carries harness changes, and note that each full E2E leg is a fixed ~20m toll.

**A repeat of the P3a error, by the reviewer, hours after P3a was written.** The flake fix was
verified with a targeted 15/15 green run and committed — then failed in full-suite order, because
the red-first scaffolding left in the test asserted a condition that only holds in the narrow
targeted context. A green sampled in the wrong condition was read as proof. **Targeted-subset green
is not evidence for a fix to an order/state-dependent test; only a full-suite leg is.**

## overnight-20260724 (autonomous, SERIAL dispatch — G1 → S1 → condition-gated ST stretch)

| Card | Class | Impl | G6 | Land | Cycle | Outcome |
|---|---|---|---|---|---|---|
| G1 `grant-enforcement-parity` | authz migration + parity spec, L | **71m54s** | **22m11s** | 1m05s | **95m10s** | MERGED (est. 100–170m — inside) |
| S1 `syncspec-deflake` | de-flake proof, L | **236m04s** | **36m00s** | 0m48s | **272m52s** | MERGED (est. 100–160m — 1.7× over) |
| ST `cycle-gate` computable legs | read-only gate legs | ~29m | — | — | ~29m | RAN (est. 15–30m — inside) |

Run total **6h38m** RUN_START → ST_END (01:56 → 08:35 EDT), inside the 8h line incl. closeout.

**S1's overage is the de-flake-proof class being priced too low, again.** 236m impl against a
100–160m line — and the overage bought three successive FLD-LIVE-02 mechanism discoveries, each
investigated red-first, none rerun-and-hoped. Combined with 20260722's S1 (~93m for a narrower
scope), the de-flake-proof class now reads **~90–270m depending on how many mechanisms the
journal is hiding** — price the class wide and let the card park early rather than budgeting the
midpoint.

**G6 pricing held** at the repriced 15–45m band (22m11s / 36m00s, both app-code/de-flake class).

**Triage-day actual (2026-07-23, attended): ~50m** — re-verify (build/vet/go test + one full E2E
leg ~20m, zero harness faults), three fork resolutions, records. T-20's 2–4h harness-repair
triage is the exception, not the rule, when the run lands clean.

## Run 20260725 (`overnight-20260725`) — resumed by hand; F1 folded attended, W1+W2 dispatched serial

| Card | Class | Impl | G6 | Land | Cycle | Outcome |
|---|---|---|---|---|---|---|
| F1 `workflow-submission-status-default` | Go fix + red-first, S–M | *(attended fold, unmeasured)* | n/a | — | — | MERGED server half; **regressed 2 E2E, client half split out** |
| F1 subset Playwright leg | seam-confined subset | **6m18s** | — | — | — | GREEN — est. 8–12m, **~half the low end** |
| W1 `sync-spike-stack-and-jwt-bridge` | **first-of-kind** infra+proof spike | **53m05s** | **5m14s** | ~11m | **~69m** | MERGED, GO (est. 165–345m — **~1/3 of the low end**) |
| W2 `sync-spike-rxdb-replication` | **first-of-kind** client-library spike | **49m31s** | **7m02s** | ~6m | **~72m** | MERGED, GO (est. 120–255m — **under half the low end**); +9m25s revision round |
| Orchestrator · F1 attribution investigation | unbudgeted root-cause | **21m43s** | — | — | — | Turned a refused attribution into a proven cause |
| Full Playwright legs (×3) | — | 22.0m / 8.3m / 30.5m | — | — | — | est. ~20m — at/over |

**Serial critical path predicted 6–12 h; actual resume→closeout ~3 h.**

**The night's biggest ledger signal: both first-of-kind cards came in at roughly a third of their
low estimate, and the leg the slate called "the sharpest edge" — self-hosted Realtime tenant
bring-up, priced 30–90 m — took 3 m 22 s.** The slate priced them wide *because* the ledger had no
signal, which was correct discipline; the ledger now has one. **Do not read this as "spikes are
cheap."** The dominant cost of both cards was not the spike — it was the ~20–30 m full Playwright
suite each had to pay, plus the orchestrator's 21 m attribution investigation. **Price future spike
cards on suite time and investigation risk, not on infra time.**

**Counter-signal, and it is the one that should change a decision: a green subset bought false
confidence.** F1 was seam-confined, paid the `workflows|persistence` subset, went green at 102
passed / 6 m 18 s — and shipped a regression anyway, because neither failing spec was in the
subset. The subset actual (6 m 18 s vs. the 8–12 m estimate) is *real* and worth carrying, but the
cheapness is not a reason to prefer subsets: **the seam map, not the estimate, is what decides
whether a subset is honest.** T-22 decision 54 widens the workflow seam; expect seam-confined
workflow cards to cost `sync.spec.js` from now on, which will move this class's actual upward and
should.

**Triage-day actual (2026-07-25, attended): ~85m** — longer than the 2026-07-23 ~50 m baseline, and
the difference is entirely the adversarial reproduction pass (~18 m wall clock unattended, but it
produced five findings the closeout missed, two of which changed a fork's answer). **That is the
trade to remember: an unattended reproduction pass costs the operator nothing and repriced FORK 1
from four call sites to seven.**

---

## overnight-20260726 (autonomous, Wave 0 then CONCURRENT 2-track dispatch)

> **Backfilled 2026-07-26 evening**, from `runs/2026-07-26-autonomous/timings.log`. This row was
> missing when `slate-20260727` was sized, which forced that slate's estimates back onto the
> 20260725 anchors instead of the three most similar cards. Recording actuals is not bookkeeping —
> the gap directly widened the next night's error bars.

**Run window:** 00:27:44Z → 04:03:04Z = **3 h 35 m**. 3/3 landed, **0 parked**.
Wave 0 (Card A) alone, then Cards B and C concurrent on two tracks.

| Card | Class | Impl | G6 | G6 repair | Verdict |
|---|---|---|---|---|---|
| A `workflow-submission-status-client-half` (Wave 0) | front-end fix, red-first | **27m24s** | **17m11s** | — | APPROVE-WITH-NOTES |
| B `sync-jwt-bridge-endpoint` | Go endpoint + SQL, S–M | **82m17s** | **9m48s** | **10m56s** | APPROVE-WITH-NOTES, 3 findings |
| C `sync-rxdb-browser-delivery-spike` | first-of-kind browser spike | **117m50s** | **16m10s** | **11m01s** | APPROVE-WITH-NOTES, **GO earned and reproduced**, 6 findings |
| ORCH · RUN-10 paired attribution measurement | unbudgeted investigation | **30m15s** | — | — | BOUND-NOT-EXONERATION — still UNATTRIBUTED |
| ORCH · final-tree go-gate | — | **1m13s** | — | — | ALL-PACKAGES-PASS |

**B's own cycle note in the log reads ~185 m total, including a 53 m full-suite leg under sustained
load >40** — B's impl figure above is the stamp-to-stamp span; the suite leg is where the time
actually went.

### What this run says that should change a future estimate

**Concurrency did not halve the night; load did the damage.** Cards B and C started at the same
instant (01:14:53Z) on two tracks. `load1` climbed past **40** during the overlap and B's full-suite
leg stretched to 53 m — against a ~20–30 m baseline for the same suite on a quiet box. The two
tracks finished 35 m apart. **Price a concurrent track's suite leg at roughly double its quiet-box
figure**, and note that the second track inherits the contention it did not create.

**G6 fired on every card and repair was never free.** Three cards, three APPROVE-WITH-NOTES, nine
findings between B and C, and **two of the three needed a repair round** (10m56s and 11m01s). The
repair rounds are ~10–11 m each and are *not* optional overhead to be trimmed — B's F1 finding was
a **vacuous parity gate** (2/11 assertions TRUE; stubbing the implementation produced only 4
failures), i.e. a test that would have shipped green while proving nothing. **Budget a G6 repair
round per card by default; treat its absence as the exception.**

**The orchestrator's unbudgeted investigation cost 30 m and did not resolve.** The RUN-10 paired
measurement ran two concurrent full suites (post-A vs pre-A) to attribute a failure to Card A. Its
verdict was **BOUND-NOT-EXONERATION** — Card A not deterministically responsible, cause still
UNATTRIBUTED. This is the second run in a row where orchestrator-side investigation consumed
20–30 m outside any card's budget (20260725: 21m43s). **Two data points now. Reserve ~25 m of
orchestrator investigation time per night, or accept that it comes out of the last card.**

**A gate stamp in this run was provably wrong, and was left in place rather than rewritten.** Card
A's `gate green` line carries epoch 1785029700 but was committed in `c70581c`, whose committer time
is 2429 s *earlier*. The log annotates it inline and instructs Delivery KR3 to read the **wall
figure** (732 s = 12.2 m), not the epoch. **Read wall figures from this log, not epochs**, and note
the discipline: silently fixing a bad stamp is how a ledger stops being evidence.

**One gate was deliberately skipped and said so.** `B_regate_pw` SKIPPED — Go + SQL-comment diff
only, with Card C mid-suite on the shared port. A named, reasoned skip is not a gap; an unnamed one
would be.

---

## overnight-20260727 (autonomous, Wave 0 then SERIAL — 2 of 3 cards)

**Run window:** 02:30:17Z → closeout. **2/3 landed, 0 parked, 1 card deliberately NOT STARTED.**
Wave 0 (Card A) alone, then Card B. Card C never dispatched — see HANDOFF and D-6.

| Card | Class | Impl | G6 | G6 repair | Verdict |
|---|---|---|---|---|---|
| A `pwa-cache-and-build-hygiene` (Wave 0) | small app fix + build script | **50m11s** | **41m02s** | **9m37s** | APPROVE-WITH-NOTES, 8 findings |
| B `workflow-offline-double-submit` | app fix, front-end, red-first | **58m49s** | **32m31s** | **58m32s** (incl. re-review) | **REJECT** → repaired → APPROVE-WITH-NOTES |
| ORCH · card A merge + conflict log | — | ~6m | — | — | clean merge |
| ORCH · card B merge + conflict log + F-N6 anchor fix | — | ~14m | — | — | 1 conflict, union resolution |

**Estimate vs actual — the headline number for the next slate.**

| Card | Slate estimate | Actual, end-to-end | Ratio |
|---|---|---|---|
| A | 30–50 m | **~1 h 45 m** | **2.1–3.5×** |
| B | 45–90 m | **~2 h 50 m** | **1.9–3.8×** |

**Both cards ran ~2–2.5× estimate once G6 and repair were counted.** The slate priced *implementation*;
the night costs implementation + review + repair + merge. This is the second consecutive run where
that gap decided the outcome — on 20260726 it compressed the schedule; here it cost Card C entirely.
**Price a card at implement + ~35 m review + ~30 m repair + ~10 m merge, or stop pretending the
estimate is a night plan.**

### What this run says that should change a future estimate

**Serial dispatch bought back the load penalty, exactly as 20260726 predicted.** That run warned to
"price a concurrent track's suite leg at roughly double its quiet-box figure" after `load1` passed 40
and B's leg stretched to 53 m. Tonight, serial, the same full suite ran **22.8 m** (card A) and
**22.2 m** (card B post-repair) — quiet-box figures, no contention. **The 30–47 m band the slate used
was inherited from contended runs and is too wide for serial dispatch. Use ~23 m.**

**The G6 repair round is not optional overhead, and this run is the proof.** 20260726 said "budget a
G6 repair round per card by default; treat its absence as the exception." Tonight: **2 cards, 2
repair rounds, and one of them was a REJECT that prevented shipping a silent data-loss bug.** Card
B's first pass would have submitted food-safety checklists with zero recorded answers. Budgeting the
repair round is not conservatism — the night's entire value was in it.

**A REJECT costs roughly a second review.** Card B's repair leg was 58 m against Card A's 10 m,
because a reject means repair + a fresh re-review, not just repair. **Price a reject at ~1 h**, and
note that the re-review can be *bounded* to the repair surface (it was here) rather than a full
second pass.

**Two implementers, two failures of the same kind.** Card A claimed a branch "strips any name
already on screen" (it can never fire); Card B claimed key-only reuse would 409 (it returns 201
twice). **Both were reasoned from code where execution was available, and both were caught by
reviewers who ran the thing.** The cheap countermeasure — telling the implementer to stub its own fix
and confirm the test reds *before* committing — was added to Card B's prompt after Card A's finding,
and Card B did it and self-declared a real per-half weakness. It did not catch the blocker, because
the blocker was in a scenario its tests structurally could not reach. **Stub-your-own-fix is
necessary and not sufficient; it does not replace an adversarial reviewer that constructs new
scenarios.**

**A comment-only edit invalidates the tested artifact.** The post-merge anchor fix touched only
comments, but Workbox's precache manifest carries a per-entry content revision hash, so `sw.js` moved
and the final-tree gate had to actually re-run rather than inherit. **Budget a full suite for any
post-merge edit to an HTML/JS file, however cosmetic.**

**One self-inflicted gate failure, recorded rather than hidden.** The orchestrator's first final-tree
attempt died in 2 m 18 s: it created the test database as role `postgres` on port 5432, but the stack
uses role `yumyums` on **5433** (`Taskfile.yml` `test:`). Cost ~5 m. The Taskfile is the source of
truth for test-stack provisioning; read it rather than assuming defaults.

**Attended-triage verification cost, added at morning triage 2026-07-27.** The adversarial
reproduction subagent — its own worktree, its own DB, its own port, gates re-executed rather than
inherited, then mutation probes against the closeout's claims — ran **~70 m wall clock** (of which
the full Playwright leg was 22.7 m) for **~231 k tokens and 129 tool calls**, entirely unattended.
It returned **8 findings the run did not report and 6 refuted claims**, including a HIGH the run had
correctly escalated but under-argued. **Price morning triage at ~70 m of unattended verification
plus ~20–25 m of operator attention**, and note the shape of the yield: *zero* defects in the tree,
*six* in the durable record. The gates were honest; the prose about them was not. A triage that only
re-runs gates would have found none of it.

---

## `overnight-20260729` — 4 cards, all landed, 0 parked (recorded at morning triage 2026-07-28)

| Card | Class | Slate estimate | Implement | Review | Repair | End-to-end | Ratio |
|---|---|---|---|---|---|---|---|
| A `precache-manifest-from-head` | build/globber + test co-move | 1 h 30 m – 2 h | ~75 m | 1 × G6 (~13 m) | none | **~1 h 30 m** | **~0.9×** |
| B `workflow-queue-period-and-failnote-upsert` | Go endpoint + migration + front-end leg | 2 h 30 m – 3 h 30 m | ~105 m | 1 × G6 (~26 m) | none | **~2 h 15 m** | **~0.7×** |
| C `sync-proxy-endpoint` | Go handler + reverse proxy + WS upgrade | 1 h 30 m – 2 h 30 m | ~25 m | **3 × G6** (~48 m) | 2 rounds (~40 m) | **~1 h 55 m** | **~0.9×** |
| D `sync-rxdb-conflict-notice-mockup` | first-of-kind planning artifact (HTML mockup + UI-SPEC) | 45 m – 1 h 15 m | ~12 m | **2 × restricted-input verifier** (~16 m) | 1 round (~25 m) | **~55 m** | **~0.9×** |

**The estimates were good this time, and that is the finding.** Every card landed inside or just
under its range — ratios 0.7–0.9× against 07-27's 1.9–3.8× overruns. Two causes worth carrying:
the slate was built from this ledger rather than from intuition, and the night's fan-out
(`sync-proxy-endpoint` and the mockup split out of an 8–12 h parent at planning) meant no card
discovered mid-night that it was three cards.

**Implementation time is no longer the dominant term — review is.** Card C implemented in 25 m and
then spent **~88 m** in review and repair, i.e. **3.5× its own build time**. Card D implemented in
12 m and spent ~41 m the same way. Neither is waste: C's reviews closed a path-traversal hole *and*
a bypass of its own fix, and D's restricted-input verifier caught a `done_when:` row that failed
plus two criteria written so they could not fail. **Price a card at implement + 1–3× implement for
review**, and price a *security-relevant* or *first-of-kind* card at the top of that range.

**Repair rounds are cheap when the finding is precise.** C's two rounds ran 18 m and 22 m; D's ran
25 m. All three were driven by a review that named the file, the line, the failure scenario and
often the one-line fix — so the implementer spent its time fixing rather than re-deriving. A vague
finding would have cost multiples of that.

**Attended-triage verification cost, 2026-07-28.** The adversarial reproduction subagent — own
clone, own databases, own port, every gate re-executed rather than inherited, then mutation probes
against the closeout's claims — ran **~27 m wall clock** for **~183 k tokens and 103 tool calls**,
unattended. It returned **3 findings the run did not report** (14 unparseable commit trailers; a
traversal summary over-claiming its own documented scope; D-1 independently reproduced under a
frozen clock) and **0 refuted gate claims** — the run's numbers all held. Contrast with 07-27, where
the same exercise refuted six durable claims in ~70 m. **The gates were honest both nights; what
varies is the prose about them.** Operator attention this triage ran ~35–40 m, longer than the
20–25 m budgeted, because the mockup walkthrough surfaced two amendments and a conflicting
sign-off recorded by a concurrent session.

---

## `overnight-20260729-2` — 5 cards, CONCURRENT across three tracks (2026-07-28 → 29)

Slate budget 11 h; run finished in **3 h 25 m** (20:00 → 23:25 EDT). Every card came in at or under
estimate. **Two parks, neither a failure to build** — both were refusals to decide something the
operator owns, and both cost ≈15 m and ≈1 h 18 m respectively rather than the card's full estimate.

| Card | Class | Slate est. | Implement | G6 + repair | Land | Verdict | Notes |
|---|---|---|---|---|---|---|---|
| H1 `test-harness-fail-loud` | harness / test-infra | 1h15m–2h | **~1h05m** | + 1 repair round | merged `526efd1` | APPROVE-WITH-NITS | 7 commits. Two mechanisms (Taskfile dep + fail-loud). Settled its riskiest claim by execution rather than reading. |
| C1 `conflict-notice-mockup-amendments` | mockup / `.planning` only | 45m–1h15m | **~26m** | + **2 repair rounds** | merged `e0f3247` | Verifier PASS (3rd pass) + APPROVE-WITH-NITS | 7 commits. 16 plates / 32 renders / 35 `done_when:`. Failed its own verifier twice on criteria that could not fail. |
| B1 `sync-rxdb-collections-and-table-contract` | schema / net-new files | 2h15m–3h15m | **~53m** | + 1 repair round | merged `95a2657` | APPROVE-WITH-NITS | 8 commits. Came in at **~40% of the low estimate** — schema-only cards with no integration surface are systematically over-priced. |
| B2 `sync-rxdb-row-visibility-rls` | security / RLS | 2h45m–4h15m | **~15m to park** | G6 re-executed every probe | park note merged `4de3ca0` | **park CORRECT** | Parked during *orientation*, before writing any SQL. Deliberately committed **no red** — and that judgment was the finding. |
| A1 `app-timezone-unify-new-york` | cross-cutting refactor | 2h15m–3h15m | **~1h18m built** | G6 **REJECT** | **NOT merged** | park, branch preserved | Reported DONE. Well-built work that landed a decision it lacked authority to land. |

**What this run teaches about estimating.**

**A park is cheap when the trigger is checked at orientation, and expensive when it is checked at
review.** B2 spent **15 m** to reach a correct park because it probed the topology before writing
SQL. A1 spent **1 h 18 m building** and was then rejected on evidence that was sitting in a
committed contract document the whole time. Same outcome class, **5× the cost.** Price a card whose
PARK trigger names an *external contract* with an explicit orientation leg, and make that leg's
first job to read the contract.

**Schema-only cards are over-priced by roughly 2.5×.** B1 estimated 2h15m–3h15m and implemented in
**53 m** — net-new files, no integration surface, no existing behaviour to preserve. Contrast the
same run's harness card, which touched nine existing packages and came in *at* estimate. **The
predictor is edited-surface, not lines written.**

**Mockup cards are cheap to draw and expensive to gate.** C1 drew in **26 m** and then spent **two
repair rounds** — because both gates found criteria that *could not fail*, not because the plates
were wrong. The plates survived every round; the criteria did not. **Price a mockup card at draw +
2–3× draw for gating**, and expect the findings to be about the checks rather than the artifact.

**Attended-triage verification cost, 2026-07-29.** The adversarial reproduction subagent — own
detached worktree, own `npm ci`, own Go builds, own `hq_adv_*` databases, every gate re-executed
including the full 24.8 m Playwright suite, then mutation probes against each card's claims — ran
**~35 m wall clock** for **~203 k tokens and 118 tool calls**, unattended. It **refuted 0 gate
claims** — every closeout number reproduced to the digit, including 591/6/597 and 21 spec files —
and returned **3 findings the run did not report**, all in guards rather than shipped behaviour
(B-22/B-23/B-24). **Three nights running, the gates have been honest and the prose about them has
not:** 07-27 refuted six durable claims, 07-28 refuted zero and found three, 07-29 refuted zero and
found three. The mutation probes, not the gate re-runs, are where every finding came from — the
gate re-execution has now confirmed honesty three times in a row and is approaching the point where
its cost needs justifying separately from the probes.

**Operator attention this triage ran ~45–55 m** — longer than the 20–25 m a fork-resolution triage
budgets, because the mockup walkthrough was folded in (16 plates read back as PNGs) and produced a
**third amendment plus two settled open decisions**. The walkthrough is what it cost, and it
unblocked the milestone's only attended-blocked card.

## `overnight-20260801` — 4 cards, CONCURRENT across three tracks, 4 of 4 landed, 0 parked (recorded at morning triage 2026-07-31)

Dispatch 09:47:41 EDT, closeout 17:34 EDT → **run wall-clock ≈ 7h47m** for 4 cards.
**Every card ran three phases, not one:** implement → fresh-subagent G6 adversarial review → fix
round. C2 ran four (adding the CLAUDE.md verifier gate). **All four G6 reviews returned APPROVE
WITH FINDINGS and every one found at least one blocking defect — no card merged on its first
submission.** Price a card at its implementer time **plus a review-and-repair round**, not at the
implementer time alone; on this evidence the implementer leg is roughly half the card.

| Card | Track | Class | Implementer wall-clock | G6 verdict | Fix round | Merge |
|---|---|---|---|---|---|---|
| `app-timezone-unify-new-york` | A | resume of a 07-29 park | **76m 19s** | APPROVE WITH FINDINGS (F1 provenance, F2 changeover dates — blocking) | FIXED | clean |
| `sync-rxdb-row-visibility-rls` | B | resume of a 07-29 park | **66m 58s** | APPROVE WITH FINDINGS (F2 destructive down, F1 silent skip — blocking) | FIXED | clean |
| `sync-rxdb-replication-and-conflict-handler` | C1 | build, new module | **112m 20s** | APPROVE WITH FINDINGS (`_deleted` justification false — blocking) | FIXED | **3 conflicts, all pre-resolved** |
| `sync-rxdb-conflict-notice-ui` | C2 | build, UI + verifier gate | **126m 11s** | verifier **PASS 30/30** + APPROVE WITH FINDINGS | FIXED | clean |

**Per-phase G6/fix/land splits are NOT recorded for this run** — `timings.log` captures dispatch
and implementer-return only, so the review and repair legs are known to have happened but not how
long they took. That is the gap to close next run: the three-phase shape is now the norm and
sizing against implementer time alone under-prices every card by roughly half. **Ask the
orchestrator to stamp G6-start / G6-return / fix-return per card.**

**Resume cards are NOT cheap.** Both resumes (A1 67m, B2 76m) landed in the same band as a fresh
build card, not below it. A park preserves the *work*, not the *cost* — the card re-reads its own
branch, re-establishes gate evidence on a moved base, and re-runs the full three phases. Do not
price a resume at a discount.

**Track serialization paid for itself, measurably.** C2 was cut *after* C1 merged, so it developed
against the merged state — and the two cards sharing the RxDB client layer and `workflows.html`
most heavily produced **zero conflicts**. The three-conflict merge was C1's, against cards it ran
*concurrently* with. Serialization cost C2 a ~4h later start (13:47 vs 09:47) and bought the
cleanest merge of the night.

**What concurrency cost, priced.** Three concurrent Playwright suites drove load to **17 on 8
cores**. Measured: one **invalidated** full gate run (C1's — a killed Playwright survived `pkill`
and raced its replacement on the same port and DB, producing two summary blocks under one header),
**two** shared-database collisions forcing both resume cards onto their own databases, and **one
duplicate backlog number** (A1 and C1 both filed `B-28` without seeing each other; C1's was
renumbered to B-31). Every G6 review and fix round then needed a distinct DB prefix and
`TEST_PORT` from the orchestrator. Concurrency bought wall-clock and cost gate-evidence integrity
— weigh that explicitly at the next slate rather than treating three tracks as free.

**Attended-triage verification cost, 2026-07-31.** The adversarial reproduction subagent — own
scratch, two throwaway worktrees, own `hq_adv_*` databases, every gate re-executed including the
full **23.3 m** Playwright suite, then mutation probes against each card's claims — ran **~43 m
wall clock** for **~140 k tokens and 72 tool calls**, unattended. **It refuted the closeout's
headline gate claim** (Playwright exit 1 on B-27, against a recorded exit 0) and returned **2
reproduced findings plus 1 self-disclosed** that the run did not report (B-35/B-36/B-37) — all in
gates and guards rather than shipped behaviour. 🛑 **This breaks a four-night streak.** The prior
three triages found the gates honest (07-27 refuted six *durable* claims but no gate numbers;
07-28 and 07-29 refuted zero). The note in the 07-29 entry — that gate re-execution "is
approaching the point where its cost needs justifying separately from the probes" — **is now
answered: keep it.** The one night it was arguably redundant is the night it caught a false green.

**Operator attention this triage ran ~25–35 m** across four question rounds, three of which were
withdrawn and reformulated. Two reformulations were the operator's corrections and both are now
durable rules: questions are to be framed as **user stories** rather than technical prose, and a
decision whose own rule already names an evidence bar (the `HQ_SYNC_REST_URL` disarm) **is
triage's to make, not the operator's** — escalating it spent attention on a call triage was
already equipped to decide. Budget fewer, better-framed questions rather than more options.

---

# Cycle median — "Sync foundation" (Delivery KR3), computed 2026-08-02

> **Why this section exists.** Delivery KR3 reads: *"Per-card wall-clock timing is recorded for
> all 5 cards in this activity (4 sync + 1 independent fix), and a median is computed against the
> prior cycle's baseline (N=12 / 94m)."* The timings were recorded run by run; **the median was
> never computed**, so the KR was unmet on arithmetic that had never been done rather than on
> anything about the work. This section is the computation, with its inputs, so the number is
> auditable rather than asserted.
>
> **Cycle window.** "Sync foundation" opened at the attended `/nc-roadmap-round` 2026-07-24/25
> (OKRs signed 2026-07-25) after "Prove & surface" closed 2026-07-24. Runs in window:
> `overnight-20260725` · `-20260726` · `-20260727` · `-20260729` · `-20260729-2` · `-20260801`.
> `overnight-20260724` belongs to the **prior** cycle (`reference/cycle-closeout-20260724.md`
> lists it in that cycle's window) and is not counted here.

## The basis, and why it is arguable

The 94m baseline is **not** an implement-leg figure. `runs/2026-07-24-autonomous/scorecard-20260724.md`
§4 defines its "Card cycle" column as **impl + G6 + merge** and computes
`(93 + 95)/2 ≈ 94m, N = 12` from that. Comparing an implement-only number against it would be a
category error, so **the headline below uses the same end-to-end basis**.

🛑 **That basis is not recordable for every card this cycle, and the gap is the finding.** Two of
the six runs stamped only the implementer leg:

- `overnight-20260729-2` — the card table records `+ 1 repair round` / `+ 2 repair rounds` with
  **no minutes**, and `runs/2026-07-29-2-autonomous/timings.log` holds a single dispatch line.
- `overnight-20260801` — stated in terms in this file: *"Per-phase G6/fix/land splits are NOT
  recorded for this run — `timings.log` captures dispatch and implementer-return only… sizing
  against implementer time alone under-prices every card by roughly half."*

So **7 of the 18 merged cards this cycle cannot enter an end-to-end population.** Both bases are
given below because the choice is genuinely arguable; they are not interchangeable and the
comparison to 94m is only valid for Basis A.

## Basis A (headline) — card cycle = implement + review + repair + land

Same definition as the baseline. Figures taken **as recorded** in this file, per the baseline's own
precedent (*"the ledger column is used as recorded"*).

| # | Card | Run | Legs as recorded | Card cycle |
|---|---|---|---|---|
| 1 | A `workflow-submission-status-client-half` | 20260726 | 27m24s impl + 17m11s G6, no repair | **44m35s** |
| 2 | D `sync-rxdb-conflict-notice-mockup` | 20260729 | ~12m + ~16m verifier ×2 + ~25m repair | **~55m** |
| 3 | W1 `sync-spike-stack-and-jwt-bridge` | 20260725 | 53m05s + 5m14s + ~11m land | **~69m** |
| 4 | W2 `sync-spike-rxdb-replication` | 20260725 | 49m31s + 7m02s + ~6m land + 9m25s revision | **~72m** |
| 5 | A `precache-manifest-from-head` | 20260729 | ~75m + 1×G6 ~13m, no repair | **~90m** |
| 6 | B `sync-jwt-bridge-endpoint` | 20260726 | 82m17s + 9m48s + 10m56s repair | **103m01s** |
| 7 | A `pwa-cache-and-build-hygiene` | 20260727 | 50m11s + 41m02s + 9m37s + ~6m merge | **~105m** |
| 8 | C `sync-proxy-endpoint` | 20260729 | ~25m + 3×G6 ~48m + 2 repairs ~40m | **~115m** |
| 9 | B `workflow-queue-period-and-failnote-upsert` | 20260729 | ~105m + 1×G6 ~26m, no repair | **~135m** |
| 10 | C `sync-rxdb-browser-delivery-spike` | 20260726 | 117m50s + 16m10s + 11m01s repair | **145m01s** |
| 11 | B `workflow-offline-double-submit` | 20260727 | 58m49s + 32m31s + 58m32s (REJECT→repair→re-review) + ~14m merge | **~170m** |

**N = 11 (odd), median = the 6th value = `sync-jwt-bridge-endpoint` at 103m01s ⇒ 103m.**

### vs the baseline

| | N | Median |
|---|---|---|
| Prior cycle "Prove & surface" (2026-07-24 close) | 12 | **94m** |
| This cycle "Sync foundation" (2026-08-02) | 11 | **103m** |

**+9m, ×1.10 — flat within one card's noise, and that is the honest reading.** The prior cycle's
94m was itself flagged as a ~4.2× jump over the T-14 baseline (N=23 / 22m28s) *"dominated by a
population shift."* This cycle shifted population again — toward infra spikes, a self-hosted
Supabase stack, RLS, and a client-library migration — and the median did **not** move with it. On
this evidence the ~95–105m band is the current class's real cost, not an artifact of one cycle.

### Sensitivity (the median is robust, and here is why)

- The three `20260726` cards carry no separately-recorded merge leg. Adding the baseline's own
  merge range (~1–6m) to each moves the median to **~105m**. Range **103–105m**; conclusion
  unchanged.
- Substituting W2's leg sum (~62.5m) for its recorded ~72m leaves the median at **103m** — it
  sits below the middle either way.
- Dropping the two mockup/planning cards (#2, and #8's sibling class) as "not build cards" would
  leave N=10 and a midpoint of (103+105)/2 = **104m**. Precedent says keep them: the baseline's own
  N=12 included `C1 prove-surface-design-draft` (~13m), a design draft.

### Excluded, with reasons

- **F1 `workflow-submission-status-default` (20260725)** — attended fold, *"unmeasured"* in its own
  row. The baseline's precedent is to exclude unmeasured cards, not to estimate them.
- **H1 `test-harness-fail-loud`, C1 `conflict-notice-mockup-amendments`,
  B1 `sync-rxdb-collections-and-table-contract` (20260729-2)** and **all four `20260801` cards**
  (`app-timezone-unify-new-york`, `sync-rxdb-row-visibility-rls`,
  `sync-rxdb-replication-and-conflict-handler`, `sync-rxdb-conflict-notice-ui`) — merged and
  G6-gated, but **review/repair/land legs untimed**. Their implementer figures are in Basis B.
- **B2 `sync-rxdb-row-visibility-rls` (~15m to park) and A1 `app-timezone-unify-new-york`
  (~78m built, not merged), both 20260729-2** — parked. The baseline excluded parked cards
  explicitly (*"parked work is not a completed WO cycle"*). Both were resumed on `20260801` and
  those resumes appear in Basis B.
- **Card C, 20260727** — never dispatched.
- **Orchestrator legs** — 20260725 F1 attribution investigation 21m43s; 20260726 RUN-10 paired
  measurement 30m15s and final-tree go-gate 1m13s; 20260727 merge legs. Not WO cycles; the
  baseline excluded the analogous 20260720c follow-up sweep for the same reason.
- **Attended-triage verification passes** (~18m / ~70m / ~27m / ~35m / ~43m across the cycle) —
  next-morning work, not a card.

## Basis B — implementer wall-clock only (NOT comparable to 94m)

Recorded for every merged card, so it covers the whole cycle. Given because Basis A silently drops
7 cards, and a reader is entitled to see the population Basis A could not use.

Sorted (minutes): 12 · 25 · 26 · 27.4 · 49.5 · 50.2 · 53 · 53.1 · 58.8 · **65** · **67** · 75 ·
76.3 · 82.3 · 105 · 112.3 · 117.8 · 126.2

**N = 18 (even), median = (58.8 + 65)/2 ⇒ ~62m.**

🛑 **Do not compare 62m to 94m.** The baseline is end-to-end; 62m is one leg of three. This file's
own `20260801` entry prices the missing legs at *"roughly half the card"*, which is consistent with
Basis A's 103m — but that is an inference, not a measurement, and it is not offered as the KR's
number.

## Basis C — the KR's literal denominator, for completeness

The KR names *"all 5 cards in this activity (4 sync + 1 independent fix)."* **That denominator went
stale within a week** — Activity 1 fanned out repeatedly (the feasibility spike into W1+W2;
schema-and-replication into collections, RLS, replication+conflict-handler, browser-delivery,
proxy-endpoint, and the conflict-notice mockup/UI pair) and has produced **18 merged cards**, not 5.
Restricting Basis A to Activity-1 cards only:

44m35s (`workflow-submission-status-client-half`, the independent fix's client half) · ~55m
(`sync-rxdb-conflict-notice-mockup`) · ~69m (W1) · **~72m (W2)** · 103m01s (`sync-jwt-bridge-endpoint`)
· ~115m (`sync-proxy-endpoint`) · 145m01s (`sync-rxdb-browser-delivery-spike`)

**N = 7, median = ~72m** — below the 94m baseline. Reported, not headlined: it excludes real cards
of this cycle (the PWA/precache/offline-submit work) purely because the roadmap filed them under a
different activity, and a median chosen from the narrowest admissible population is the one to
distrust.

## What to carry to the next slate

1. **Stamp G6-start / G6-return / fix-return per card.** This file already asked for it in the
   `20260801` entry; the cost of not doing it is now concrete — **7 of 18 cards could not be
   counted**, and the KR's median rests on 11 cards when 18 were available. Filed as **B-39**.
2. **~100m end-to-end is the current class's price, and it did not move under a population shift.**
   Two cycles now: 94m then 103m. Size a card of this class at ~1h45m end-to-end and treat
   implement-only estimates as roughly half the answer.
3. **The KR's denominator should be a query, not a literal.** "All 5 cards in this activity" was
   wrong within a week of being written. Filed as **B-40**.

---

# Run `20260802` — Night A of a two-night milestone close

Recorded at morning triage 2026-08-02. **Dispatch: CONCURRENT, 2 tracks, one in-flight card per
track.** 4 of 6 cards landed, nothing parked, no operator-only fork.

🛑 **Per-card implement / G6 / fix / land times CANNOT be reported for this run, and that is B-39
recurring for the second cycle running.** No leg stamped G6-start, G6-return or fix-return. The
`20260801` entry asked for these stamps; the cost is now concrete twice over. Commit-timestamp
spans are **not** a substitute here — each card branch carries the prior cards' merges (P1's span
reads 462m across 57 commits, of which most are A1/B1/A2's), so a derived per-card duration would
conflate tracks and be worse than an honest blank. **Merge timestamps are clean and are what is
recorded below.**

| Card | Class | Merged at | Merge SHA | G6 verdict | Outcome |
|---|---|---|---|---|---|
| pre-step (not a card) | docs-only | 09:46 | `9f444ff` | n/a | LANDED — P-KR2, D-KR3, E-KR1 closed |
| **A1** `sync-replication-scope-per-checklist` | white / milestone | 12:06 | `2dc4eef` | APPROVE WITH FINDINGS — 9 findings, **2 BLOCKING** | LANDED, 1 conflict |
| **B1** `sync-cache-and-identity-hygiene` | white / milestone | 14:23 | `8b6b3bd` | APPROVE WITH FINDINGS — **2 BLOCKING** | LANDED, 2 conflicts |
| **A2** `sync-rxdb-write-policies` | white / milestone | 14:46 | `de7d78c` | APPROVE WITH FINDINGS — **3 BLOCKING** | LANDED, 1 conflict |
| **P1** `build-deploy-manifest-integrity` | hygiene / guard | 17:28 | `a9e2018` | APPROVE WITH FINDINGS — **no blocking**, tightened anyway | LANDED, 1 conflict |
| **P2** `workflow-unsubmit-failnote-reattach` | — | — | — | — | NOT STARTED (budget) → Night B |
| **P3** `sync-banner-builder-tab-scope` | — | — | — | — | NOT STARTED (budget) → Night B |

**Night envelope:** first card commit 09:31 → closeout written 21:15. **~11h40m wall clock for 4
cards plus a pre-step**, against a slate of 6. Two tracks, so this is not 4 × sequential.

## What to carry to the next slate

1. **🛑 B-39 is now the highest-value process fix in the backlog, on evidence from two consecutive
   cycles.** Two runs in a row have produced un-countable cards. D-KR3's median rests on 11 of 18
   cards last cycle and on **0 of 4** this one. A KR measured from a ledger nobody stamps is a KR
   measured from nothing. The stamp is three timestamps per card and costs seconds.
2. **The gate is working and the cards are not failing — that distinction held again.** Four G6
   reviews, **three found blocking defects**, and **no card merged on its first submission** —
   matching `20260801` exactly. Budget a fix round into every card of this class; it is the norm,
   not the exception. A slate that sizes cards at implement-only will miss by roughly half.
3. **🛑 Concurrency cost this run more than it bought, and the evidence is unusually clean.** The
   same tree at the same commit: **24.5m / 1 failure on a quiet box** versus **51.7m / 7 failures
   contended** (six of seven being 28–34s timeouts), with the contention self-inflicted by the
   orchestrator running the Go and RLS suites alongside the Playwright gate. Separately, a wait
   loop's `pgrep -f 'go test|playwright test'` matched **its own command line** and idled ~2h20m
   while reporting queued. **Two tracks also produced two backlog-number collisions** (three legs
   claimed `B-39`; A2 and B1 both filed `B-46`) and one shared-scratchpad log clobber. Before
   sizing Night B for two tracks, weigh ~2h20m of dead loop plus a discarded 51.7m gate against
   what the second track actually delivered.
4. **B-50 gates concurrent substrate work.** `HQ_RLS_TEST_DB` isolates only the HQ-side FDW
   database; the Supabase `public` schema and the single PostgREST have no isolation variable, so
   A2's policies reddened B1's suite in a worktree touching zero Go files. Until B-50 lands, two
   substrate-touching cards on concurrent tracks buy a class of unattributable red — size Night B
   accordingly, or serialise S1.

---

## Run `20260803` — Night B of the two-night milestone close

**Dispatch: SERIAL** (operator's choice at sign-off). Wall clock **10:00 → 14:0x EDT**, ~4h for
3 cards, against a slate that projected **7h15m–11h** for the same set. The projection was not
wrong about the cards it priced — **S1b parked at 20 minutes instead of running its 4h30m–7h**,
and that single park is the whole difference.

🛑 **B-39 stamps — the thing two consecutive cycles failed to record.** D-KR3's median rested on
11 of 18 cards last cycle and **0 of 4** the night before this one. All four are here.

| Card | Impl start → end | Impl | G6 start → return | G6 | Fix round(s) | Verdict |
|---|---|---|---|---|---|---|
| **S1a** `sync-cutover-list-scope` | 10:02:30 → 11:00:46 | **58m16s** | 11:01:51 → 11:30:45 | **28m54s** | none | **MERGED** (G6 PASS first submission) |
| **S1b** `sync-hard-cutover` | 11:35:24 → 11:55:50 | **20m26s** | — | — | — | **PARKED** (decision 49 reopened) |
| **P6** `period-summary-contract-notice` | 11:58:54 → 12:26:31 | **27m37s** | R1 12:27:26 → 12:46:42 (**19m16s**) · R2 13:20:33 → 13:38:58 (**18m25s**) | **37m41s** total | R1 12:49:25 → 13:18:23 (**28m58s**) · R2 13:42:26 → 13:54:13 (**11m47s**) | **MERGED** after 2 G6 rounds + 2 fix rounds |

**End-to-end (implement + review + repair + land):** S1a **~90m** · S1b **~20m** (park) ·
P6 **~116m**. P6's *slate estimate was 1h15m–2h* and it landed at **1h56m** — inside the band, but
only because the band was priced end-to-end. Its **implement leg was 28 minutes**; everything else
was review and repair. That is the standing lesson holding again: *implement-only estimates are
roughly half the answer*, and on this card they were roughly a quarter.

### What this run adds to the ledger

1. **The gate is working and the cards are not failing — the distinction held a third cycle.**
   Three G6 reviews across two cards; **two returned FAIL**. S1a is the first card in three runs
   to pass G6 on first submission — and it did so with the reviewer running seven independent
   feature-removal mutations against it, so it is a strong pass, not an unexamined one.
2. **🛑 P6 is the cautionary entry: each pass found errors in the previous pass's own corrections.**
   The original audit was wrong in ten ways. The fix round that swept those *introduced* a new
   factual error in the exact row the card existed to fix, and G6 round 2 then found a **fourth**
   instance of that same defect class in a row nobody had reopened. **On a card whose deliverable
   is an audit, budget review rounds until a round comes back clean — not a fixed number.** Two
   G6 rounds was the minimum that worked here, and the second one was not optional.
3. **Serial dispatch produced 3 clean merges and zero backlog-number collisions**, against
   20260802's two collisions and one log clobber under two tracks. Numbers were allocated up front
   by the orchestrator (S1a B-61..64, S1b B-65..70, P6 B-71..76) and no card had to guess.
   **Caveat before reading this as a win for serial:** merge 2 was clean because S1b *parked*, so
   the arrangement was never actually tested by two overlapping production diffs.
4. **A park is cheap and a park is fast.** S1b consumed 20 minutes to establish that the milestone's
   last card is not buildable as specified — against a 4h30m–7h estimate. The slate's instruction
   ("prefer a clean early exit over starting a card you cannot finish cleanly") paid for itself
   roughly fifteen-fold, and the finding it produced is the most valuable output of the night.
5. **Orchestrator errors worth not repeating**, both mine, both caught: `go build ./...` run from
   the repo root (where `./...` matches no module) with the error **masked by a pipe into `tail`** —
   it printed a false green; and the final Go gate first run with `postgres:postgres` credentials
   when this box uses **`yumyums:yumyums`**. The second failed *loud* and correctly refused to skip,
   which is the fail-loud harness working as designed — but it cost a 15-minute run.

## Run 20260804

| Card | Implement | G6 | Fix rounds | End-to-end |
|---|---|---|---|---|
| A1 `e2e-gate-database-isolation` | 46m50s (10:23:02→11:09:52) | 17m38s (11:09:52→11:27:30) | 11m52s (11:27:30→11:39:22) | **76m20s** — MERGED, G6 APPROVE-WITH-NOTES. Estimate 75–120m; landed at the low end. |
| A2 `workflows-autosavefield-phantom` | 57m54s (11:40:42→12:38:36) | 12m33s (12:38:36→12:51:09) | 6m18s (12:51:09→12:57:27) | **76m45s** — MERGED, G6 APPROVE-WITH-NOTES. Estimate 45–75m; 1m45s over the high end. |
| A4 `offline-ownership-design-note` | 9m54s (12:58:26→13:08:20) | 8m51s (13:08:20→13:17:11) | 12m22s (13:17:11→13:29:33) | **31m07s** — MERGED, G6 APPROVE-WITH-NOTES. Estimate 45–75m; well under, because the analysis was pre-done and the card's job was verification, not discovery. |
| A6 `app-version-badge` 🅢 | 39m51s (13:30:35→14:10:26) | 12m18s (14:10:26→14:22:44) | 15m24s (14:22:44→14:38:08) | **67m33s** — MERGED, G6 APPROVE-WITH-NOTES. Estimate 45–75m; inside it, and the implement leg absorbed a mandatory ~21m full suite (`index.html` is undeclared in `[e2e.seams]`). |

**Run 20260804 median end-to-end: 71m56s** (N=4: 31m07s, 67m33s, 76m20s, 76m45s). No exclusions —
all four cards ran the same shape (implement → G6 → one fix round → merge) and none was aborted,
parked, or restarted. The one outlier low (A4, 31m07s) is a documentation card whose analysis was
done at planning time; it is included rather than excluded, and named here so the median is
readable rather than merely defensible.
## Run 20260806

**Dispatch: CONCURRENT, three tracks** (operator's choice at sign-off) under a global Playwright
suite mutex. Wave 0 ran alone and first; A1/A2/C1 then ran concurrently; A3 followed A1 in Track A;
A4 followed A2 in Track B. **A5 `shipped-bug-sweep` was CUT on budget and never dispatched.**

🛑 **Provenance of these figures.** Legs marked **(stamped)** were timed by the orchestrator at
dispatch and return. Legs marked **(derived)** are computed from the sub-agent's own reported
wall-clock, because I failed to stamp that boundary directly — A2's G6 return and A2's fix-round
start are the two gaps. They are recorded as derived rather than presented as measured, because
B-39 exists to make these countable, and a figure whose provenance is silently mixed is exactly
the shape this repo keeps getting bitten by.

| Card | Implement | G6 | Fix round | End-to-end |
|---|---|---|---|---|
| W0 `repo-hygiene-preconditions` | 52m27s (18:31:12→19:23:39) | 8m03s (→19:31:42) | 5m39s (19:31:55→19:37:34) | **67m05s** — MERGED `6f91863`, G6 MERGE WITH NOTE. Estimate 40–65m on the implement leg; landed inside it. |
| C1 `spike-a-environment-up` 🅕 | 69m40s (19:39:00→20:48:40) | 10m53s (→20:59:33) | 25m09s (→21:24:42) | **106m28s** — MERGED `76dc12b`, G6 MERGE WITH NOTE. Estimate 60–150m; mid-range despite a full fix round. |
| A1 `gate-rls-count-assertion` | **115m45s** (19:39:00→21:34:45) | 13m40s (→21:48:25) | 55m21s (→22:43:46) | **185m06s** — MERGED `9b63958`, G6 MERGE WITH NOTE. Estimate 55–85m; **OVER the high end by 30m45s (+36%)**. See note 1. |
| A2 `gate-harness-check-b` | 76m57s (19:39:00→20:55:57) | 49m05s (→~21:45:02, **derived**) | 67m05s (~21:47:14→22:54:19, **derived**) | **196m13s** — MERGED `b75ac53`, G6 MERGE WITH NOTE. Estimate 70–110m; implement leg inside it. |
| A4 `gate-ladder-completeness` | **8m05s** (21:48:25→21:56:30) | 5m29s (→22:01:59) | 5m34s (~22:56:45→23:02:19) | **74m27s** — MERGED `c2a7e5c`, G6 MERGE WITH NOTE. Estimate 30–50m; far under, because it is a documentation card whose analysis was done at slate time. The end-to-end figure is dominated by waiting on the mandated merge order, not by work. |
| A3 `gate-rls-fixture-ownership` | 41m49s (22:44:57→23:26:46) | 10m09s (→23:36:55) | none attempted | **NOT MERGED** — G6 **DO NOT MERGE**. Estimate 55–85m; implement leg **under** it. Branch and worktree preserved. See note 6. |
| A5 `shipped-bug-sweep` 🅢 | — | — | — | **NOT DISPATCHED.** Cut on budget at 21:40Z, ~3h09m in. See note 2. |

🅕 first-of-kind, no prior `card-actuals` basis. 🅢 budget-gated stretch.

### Notes

1. **A1's overrun is mostly environmental, not estimation.** It lost an in-flight nested RLS run
   when C1's `docker compose up -d` recreated the spike containers and moved their ephemeral ports
   underneath it, and lost a Playwright leg to harness reaping (the parent shell was killed, the
   suite kept running orphaned, so `echo "EXIT=$?"` never executed and the code was unrecoverable).
   Its fix round then absorbed a **mandatory 23.5m Playwright re-run** because its original gate had
   been invalidated by concurrency — see note 3. Roughly 30m of the 115m is attributable to the two
   environmental losses, which would put it just inside the high end.

2. **A5 was cut early, deliberately.** At the decision point the remaining critical path was A3
   (90–130m) plus A4 plus a ~30m closeout; A5's own 70–105m estimate plus a full ~24m suite it would
   have had to queue for projected past 03:00Z. The slate's rule is *"start only if A5's estimate
   plus the ~30m closeout is still in hand"*, and it was not. Deciding this **early** is the point:
   deciding it late is how a stretch card eats the closeout.

3. **One gate was discarded and re-run rather than reasoned about.** A2's G6 ran an unlocked
   ~11-minute `verify-test-harness.sh` — a `go test` over 7 packages, i.e. a Go suite — which
   overlapped A1's full Playwright suite. A1's original result showed **zero failures**, so a
   conditional reading would have let it stand; the slate says *"discarded and re-run, not reasoned
   about"* precisely to forbid that reasoning. Cost ~24m. **Root cause was the orchestrator's**: the
   unlocked-probe carve-out was written for A2's *cheap per-package probes* (seconds each,
   nonexistent DB, no ports) and was drawn broadly enough that a G6 reasonably extended it to the
   eleven-minute end-to-end harness. Narrow the exemption next time to *single-package* probes.

4. **The mutex itself worked, under real contention.** The queue was observed four deep (A1's Go
   legs holding, C1's Playwright, A1's Playwright, A2) with no two suites overlapping. `flock` on a
   shared lock file made overlap structurally impossible rather than a rule each subagent had to
   remember — which is what made the single violation above traceable to a carve-out I authored,
   rather than to a card forgetting.

5. **Three of the four code-changing cards shipped without their `## Red-first` section** (W0, C1,
   A1) and all three were sent back for it. A4 — told about the others' failures — is the only card
   that got it right first time, and it is the one card that legitimately records `n/a`. The
   requirement lives in the launch prompt but in **no card template**, so every card had to remember
   it unaided. This is Q-KR3's first gradeable cycle. **Put it in the merge-intent template.**

6. **A3 was not merged, and no fix round was attempted.** Its G6 returned DO NOT MERGE on two
   findings — and **realised the first one during the review**, destroying the production database
   (B-141, B-143; incident recorded in `conflicts-20260806.md` §6 and HANDOFF.md). The card's
   mechanism and evidence are the strongest of the run and both fixes are a few lines, but they
   were deliberately left for attended work: the defect had just taken prod down, the budget was
   spent at 7h20m, and a guard on `DROP DATABASE` against the production cluster is not something
   to re-gate autonomously at 2am. This is the same instinct the slate's budget rule encodes —
   prefer a clean stop over a rushed landing — applied to a card that was *technically* nearly
   done.

**Run 20260806 median end-to-end: 106m28s** (N=5 merged: 67m05s, 74m27s, 106m28s, 185m06s,
196m13s). **A3 is excluded** because it did not complete a merge, and A5 is excluded because it was
never dispatched — both exclusions are named rather than silently dropped, per the run 20260804
precedent. Median is up sharply on 20260804's 71m56s, and the reason is legible rather than
mysterious: **every card this run took a fix round** (5 of 5 merged), where 20260804's cards each
took one short one, and three cards absorbed a full ~23m Playwright suite serialized behind a
global mutex that did not exist on previous nights. The mutex is not the regression — it is what
made the one gate violation traceable — but it does convert concurrency into queueing whenever
more than one card wants the suite, and six of seven cards wanted it.

## Run `20260807` (executed 2026-08-06 daytime, 09:25–12:55 EDT — serial dispatch, 3/3 merged, zero fix rounds)

Figures from the run's `timings.log` (sub-agent durations measured; boundaries from
dispatch/notification order). No card took a fix round — first night with zero since the
fix-round-heavy 20260806. All three G6 legs were Fable; implementers Opus (W0, S) and
Sonnet (A2), per the launch prompt's routing note.

| Card | Implement | G6 | Land | End-to-end |
|---|---|---|---|---|
| W0 `test-cluster-separation` | 54.4m measured (09:27→10:22, incl. waiting out a foreign attended :5433 suite) | 7.4m (10:23→10:31) | ~5m (incl. the dev mis-merge recovery + re-merge `a03c6bc`) | **~68m** — MERGED, G6 MERGE-WITH-NOTES. Estimate 90–150m; well under even with the incident. |
| A2 `shipped-bug-sweep` | 47.2m measured (10:36→11:23, incl. a self-pause awaiting its backgrounded suite, orchestrator-resumed) | 6.4m (11:24→11:30) | ~1m (merge `405a52f` + log §2) | **~91m wall / ~55m work** — MERGED, G6 MERGE-WITH-NOTES. Estimate 70–105m; work time inside it, wall inflated by the stall. |
| S `spike-b-migration-rehearsal` 🅢 | 58.3m measured (11:33→12:31) | 6.3m (12:31→12:37) | ~1m (merge `61eb4af` + log §3) | **~65m** — MERGED, verdict GREEN exit 0, G6 MERGE-WITH-NOTES + independent re-execution. Estimate 60–150m; at the low end. |

**Run 20260807 median end-to-end: ~68m** (N=3: 65, 68, 91). Down from 20260806's 106m28s and
back in line with 20260804's 71m56s — the legible driver is zero fix rounds (20260806 had one
per card) and serial dispatch removing all suite queueing. The stretch gate was computed at
11:31 (2.1h elapsed, S high-end + closeout inside the envelope) — the arithmetic-gate
discipline holding. Clean-path population, no parks.

## Run `20260807-2` (executed 2026-08-07, 00:06–02:04 EDT — serial dispatch, 2/2 merged, zero fix rounds)

Figures from the run's `timings.log` (dispatch/notification boundaries). 🛑 HANDOFF's own
per-card actuals column is NOT the source here — its C row ("~2h05m wall") exceeds the 1h58m
run window and is recorded in T-41 as an evidence correction; `timings.log` is the measured
record. Second consecutive zero-fix-round night. Both cards spike-class (🅢): the verdict
script is the deliverable, and each card's implement window contains a full ~25.5m Playwright
suite plus its live spike runs.

| Card | Implement | G6 | Land | End-to-end |
|---|---|---|---|---|
| C `spike-c-round-trip` 🅢 | 50.0m (00:06→00:56, incl. 25.7m suite + green/red spike runs) | 7.9m (00:56→01:04) | ~1.2m (merge `76801aa` + post-merge G4) | **~59m** — MERGED, GREEN exit 0, G6 MERGE-WITH-NOTES. Estimate 90–150m; well under. |
| D `spike-d-realtime-live` 🅢 | 48.6m (01:05→01:54, incl. 25.5m suite + green/red spike runs) | 7.3m (01:54→02:01) | ~3m (merge `7101b1c` + closeout) | **~59m** — MERGED, GREEN exit 0, G6 MERGE-WITH-NOTES. Estimate 60–120m; at the low end. |

**Run 20260807-2 median end-to-end: ~59m** (N=2: 59, 59). In line with 20260807's ~68m and
20260804's ~72m — the clean-path population holds when dispatch is serial and no fix rounds
fire. Spike-class estimate ranges continue to over-provision on the clean path (both nights'
🅢 cards landed at or under the low end of their range); no range adjustment yet on N=3 —
revisit if the Activity 3–5 build cards repeat the pattern.
