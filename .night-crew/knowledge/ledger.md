# Ledger

> Scaffolded by `night-crew init`. The running record of decisions and their
> rationale for this repo. Entries accrue over cycles; start empty.

- **2026-07-09 — Purchasing hardening-PRD card shape → (a) enumerate + mark as a real app.**
  Fork was queued in the roadmap on the premise Purchasing is a bare mockup. That
  premise is stale: `backend/internal/purchasing/*` (~20 endpoints: orders, shopping
  lists, cutoff, suggestions, repurchase-reset + scheduler), `purchasing.html` (1,078 L),
  and `tests/purchasing.spec.js` (31 KB) are real and tested. Resolution: treat Purchasing
  like the other four apps — its PRD copies the Operations exemplar and honestly marks
  WORKING/UNPROVEN/BROKEN (confirmed-only-BROKEN). Rejected (b) out-of-scope (would hole the
  5/5 Product KR-1 denominator) and (c) thin stub (dishonest given real code). Slated on
  `overnight-20260710`. *Rationale: enumerate against what the build actually surfaces.*

## Morning-triage resolutions (2026-07-10) — `overnight-20260710`

Review verdict: 5 commits, all `docs(night-crew):`; diff 100% docs-only (1974 insertions,
7 files); 3 confirmed-BROKEN citations independently re-verified at cited lines; roadmap
flips clean; `build`/`vet`/`test` + G4 discipline greps N/A (no Go/app code touched — the
night-crew orchestration framework those greps target doesn't exist in this repo). 0 cards
parked. Merged to `dev` `--no-ff`.

- **T-1 — Sign off all four Activity-1 PRDs → DONE.** Users, Onboarding, Purchasing, and
  Inventory hardening PRDs all cleared G6 (3 REVISE→pass + 1 first-pass ACCEPT) and their
  load-bearing BROKEN marks were re-verified at cited lines. Signed at triage; roadmap rows
  flipped DRAFTING → DONE. **Closes Product KR-1 (5/5 apps have a signed hardening PRD)** and
  unblocks Activities 2–5. Chosen over holding sign-offs for per-PRD re-read — the two-pass +
  G6 gate plus the independent BROKEN-citation re-verify is the evidence sign-off requires;
  a cold re-read of four honest flow-maps adds no signal the gate didn't already produce.

- **T-2 (D-3) — Inventory Trends/Cost tabs waived as unbuilt-future.** FR-24/FR-25 are
  confirmed-BROKEN `.coming-soon` stubs (`inventory.html:993-999`, no API/state). Waived —
  standing up the charts is net-new feature work, not hardening — so they ship as-is and are
  **excluded from the Engineering-KR "0 known-broken flows" denominator** by explicit
  sign-off. Chosen over (b) a hide-the-tabs WO and (c) building the charts this cycle: the
  cycle's remit is hardening existing flows, not shipping new features; dead-but-labeled
  "coming soon" tabs are an honest placeholder, not a broken shipped flow. Contrast Purchasing
  FR-18 (History), which is **not** waived — its backend endpoint exists, so the absent UI is
  a real gap in a shipped feature → Activity-4 WO.

- **T-3 (D-1) — Purchasing's 5 no-UI admin endpoints are intentionally API/scheduler-only.**
  `POST /simulate-cutoff`, order `lock`/`unlock`, and the 3× `repurchase-reset*` routes are
  real handlers reachable only via API or the scheduler. Ruled **by-design admin/cron
  surfaces** — no UI build required. This scopes the Purchasing **Activity-4 WOs as test-only**
  (prove the handlers behave via seeded assertions), not test-plus-UI-build. Chosen over
  surfacing any of them in `purchasing.html`: they are operator/automation affordances, not
  crew-facing flows; adding UI would invent scope the cycle didn't ask for.

- **T-4 (D-2) — `CLAUDE.md` Purchasing label corrected Mockup → Active (attended, now).**
  The repo-root Current-Tools table still called Purchasing a "Mockup" despite 21 routed
  endpoints, a transactional service layer, a 15-min scheduler, a live 4-tab UI, and a 31 KB
  spec. Fixed inline at triage rather than filing a doc-update WO — a one-line table edit is
  cheaper to do than to schedule.

- **T-5 (D-4) — Users stale-E2E repair graduated to BACKLOG (rides the Users Activity-3/4 WO).**
  `tests/users.spec.js` has two tests navigating dead `#t3`/`#s3` DOM (removed in the 3-tab→
  2-tab refactor; Access now renders into `#s2`). Features work; tests can't run → marked
  UNPROVEN (stale-test), not BROKEN. Recorded in `BACKLOG.md` so it survives run-to-run; repoint
  `#t3`/`#s3` → `#t2`/`#s2` folds into the Users Activity-4 prove-UNPROVEN WO. Runs unattended.

- **T-6 (housekeeping) — Operations exemplar PRD committed to git.**
  `PRD-operations-hardening.md` (the DONE exemplar the other four copy) was on disk but
  **never committed** — the artifact for a DONE card was untracked. Added in the triage commit
  so version control matches the roadmap's DONE status.

- **Process finding (carried, not a decision): single-pass recall < 90% on 4 of 5 apps**
  (Users 73%, Purchasing 77%, Inventory 82.5%, Ops 85%; only Onboarding 91.2%). The two-pass
  mandate was load-bearing every time; every miss hid in **backend-only / no-UI surfaces**
  (auth enforcement, crons, service-token contracts). Recommend the post-run design batch name
  that angle explicitly as a standing enumeration rule ("second pass must sweep backend-only
  surfaces"). Deferred — night-crew stays frozen this run.

## Morning-triage resolutions (2026-07-13) — `overnight-20260712`

Review verdict: 12 commits, all `docs(night-crew):`; diff 100% docs-only (695 insertions,
10 files, all under `.night-crew/`; **0 Go files touched**); build+vet green; G4 discipline
greps clean; `replay`/`testdata` untouched. `go test ./...` = one pre-existing env-gated red
only — `internal/receipt TestRunIngestCycle_ScenarioTable` AI-matching subcases return 401
(no `ANTHROPIC_API_KEY` in shell); not merge-introduced (0 Go changed, result identical to
pre-merge `dev`). 0 cards parked; 10/10 cards G6-passed. Merged to `dev` `--no-ff`.

- **T-7 — Sign off all 10 Activity-2/3 cards → DONE.** The five confirm-absence sweeps
  (Activity 2, Eng KR-1) and five test-audits (Activity 3, QA KR-1) all cleared G6, and the
  three new confirmed-BROKEN citations (Ops FR-4, Ops NFR-3, Onboarding NFR-5) were each
  independently re-verified at the cited line. Signed at triage; roadmap rows flipped DRAFTING
  → DONE. **Net cycle movement:** Eng KR-1 known-broken denominator is now exactly **4 built
  flows** (Ops FR-4, Ops NFR-3, Onboarding NFR-5, Purchasing FR-18); QA KR-1 saw its first hard
  data point — **1 WORKING→UNPROVEN drop** (Purchasing FR-7, a generic-content tautology test).
  Chosen over holding any row for cold re-read: the two-pass + G6 gate plus the independent
  BROKEN-citation re-verify is the evidence sign-off requires; the run added no code, only
  honest reclassification.

- **T-8 (D-5) — Onboarding FR-16 + NFR-4 (video pipeline) waived, fixture preserved to BACKLOG.**
  Both are **fully implemented** (`handler.go:540-640`, `video.go:22-206`) but untestable in E2E
  without DO Spaces creds + an `ffmpeg` binary — not broken, only unprovable-in-env. Waived from
  the Engineering-KR "0 known-broken flows" denominator as **environment-gated** (parallel to D-3's
  Trends/Cost waiver), so they ship as-is and don't block the cycle gate. Chosen as **waive-now-
  but-preserve** over a plain waive: a BACKLOG item to stand up a Spaces+ffmpeg E2E fixture and
  prove FR-16/NFR-4 is queued for when creds exist — the prove-path survives rather than being
  dropped silently. Rejected building the fixture this cycle (net-new test-infra, ~4–5h + creds the
  operator would supply — the cycle's remit is hardening, not env build-out). **Contrast:** NFR-5
  (video-led reopen no-op) is **not** waived — real code, wrong behavior → Activity-4 fix-card.

- **Standing DB flag re-armed for Activity 4 (unchanged).** This run touched no DB/E2E by design;
  the localhost-Postgres + E2E-suite precondition (`brew postgresql@16`) bites at the next slate
  (Activity 4 writes app code + runs red-first proofs). Arm it before the Activity-4 slate.

## Morning-triage resolutions (2026-07-14) — `overnight-20260714`

Review verdict: the **first app-code + red-first + E2E slate** — 7 atomic commits + 1 `docs`
closeout; diff +907 / −84 across 17 files (2 backend `.go`, 4 HTML/`sw.js`, 4 spec.js, docs),
per-card footprint clean. On the run's final tree **and** the merged tree: `go build ./...` +
`go vet ./...` **green**. `go test ./...` = **pre-existing, environment-gated reds only**,
confined to `internal/receipt` + `internal/inventory` (invalid `ANTHROPIC_API_KEY` → 401 AI
matching + DB-dependent visibility tests) — **identical failures on pre-merge `dev`** (verified
by checkout), and **neither package was touched by this run** (the only Go edits were
`workflow/handler.go` and `onboarding/db.go`, both `[no test files]` in Go — their proof is the
Playwright E2E suite, G6-reverified on the ephemeral pg16 stack). G4 discipline greps N/A
(night-crew orchestration pkgs — `journal`/`orchestration`/`workorder` — don't exist in this app
repo); `replay`/`testdata` absent. The 3 backend fix diffs were spot-checked attended (real
red→green logic, well-commented, in-seam). 7/7 cards G6-PASS, 0 parked, 0 REVISE, 0 Docker
crashes. Merged to `dev` `--no-ff`; re-ran build+vet+test on the merged tree — same env reds, no
new failures.

- **T-9 — Sign off all 7 `slate-20260714` cards → DONE.** Wave-0 `hq-infra-docker-standardize`
  (local Docker DB → pg16, `test:all`/`bdd` repointed off the remote box) + the 4 confirmed-BROKEN
  fix-cards (Ops **FR-4** yes/no-"No" corrective gate, Ops **NFR-3** photo-required gate, Purchasing
  **FR-18** History tab, Onboarding **NFR-5** video-led reopen/reject) + the 2 stretch tests
  (Purchasing **FR-7** proved WORKING, Users **stale-E2E** repaired 17/2 → 19/0). Every fix was
  **red-first** — the new regression test was captured FAILING against the unfixed build before the
  fix, then flipped green (no test passed without its fix); each was independently G6-reverified at
  the diff + evidence, and I re-verified the 3 backend diffs and build/vet attended. **Net cycle
  movement:** Eng KR-1 known-broken denominator **4 → 0**; QA KR-1 **+3 hardened tests** (1
  vacuous→genuine, 2 stale→repaired). Chosen over holding any row for a cold re-read — red-first
  capture + G6 diff-reverify + the attended backend spot-check is the evidence sign-off requires;
  a cold re-read of honest, proven fixes adds no signal the gate didn't already produce. Roadmap
  rows flipped DRAFTING → DONE.

- **T-10 (F-1) — Ops NFR-3 backend resubmit `require_photo` gate: scheduling delegated to the
  planning agents, not hand-picked at triage.** F-1 is the in-footprint deferral from `ops-nfr3`:
  the field-level required-photo gate ships front+back (`ad105f7`), but the **rejection-driven
  resubmit** case is frontend-only because `SubmitChecklistInput` carries no `submission_id`/
  rejection context, so a direct-API resubmit can bypass it (fix = a `submission_rejections` join +
  red-first test). **Operator rider (verbatim intent):** whether F-1 rides the next slate or waits
  in backlog is a *throughput / queue-placement* decision that belongs to the PjM (`/nc-slate-plan`),
  PM (`/nc-pm-session`), and engineering agents — not an operator hand-pick at triage — per "operator
  owns the quality bar, not the throughput." **Resolution:** F-1 stays in `BACKLOG.md` as a scoped,
  ready candidate; its promotion into a slate is decided by those agents at slate-planning time.
  **Rule recorded:** backlog-vs-schedule (queue placement / cadence) is a planner decision — triage
  surfaces the item + its scope + its risk, the planning agents decide *when* it runs. In the
  meantime the frontend gate covers the normal path; the residual direct-API bypass is documented in
  BACKLOG. (F-2, the orphaned `users.html:122` `<div id="s3">`, likewise stays backlogged — trivial,
  folds into any future Users card; no decision.)

- **Standing DB flag consumed + superseded by the ephemeral pg16 Docker env.** Activity 4 wrote app
  code and ran red-first E2E proofs on `docker-compose.nc.yml` (pg16) with **zero Docker crashes** —
  Wave-0 (`hq-infra-docker-standardize`) made that env the **canonical local DB path** (local Docker
  DB standardized to `postgres:16`, matching prod + the ephemeral env; `task test:all`/`bdd`
  repointed off the remote Windows box). The prior "arm localhost `brew postgresql@16`" flag is
  therefore satisfied via Docker pg16 and no longer a precondition; it re-arms only if the
  verify/merge DB path changes underneath it.
