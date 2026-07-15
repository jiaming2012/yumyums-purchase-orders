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
