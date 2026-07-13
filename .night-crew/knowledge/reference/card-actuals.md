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
