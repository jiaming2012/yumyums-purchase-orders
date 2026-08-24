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

## Morning-triage resolutions (2026-07-15) — `overnight-20260715`

- **T-11 — Sign off all 16 prove-UNPROVEN cards + the 1 graduated fix → DONE; merge to `dev`.**
  The full test-only prove sweep (`slate-20260715`, 5 concurrent tracks A–E) landed 16/16 cards,
  every card G6-PASS, 0 REVISE, 0 Docker crashes. ~52 previously-UNPROVEN flows across Operations,
  Purchasing, Users, Onboarding, Inventory now carry **real red-first assertions** naming an
  observable DB/UI behavior; ≥4 vacuous/tautological tests were rewritten into genuine guards.
  **The sweep's headline is that the forecast inverted:** scoping predicted ~34–40 of ~78 flows
  would go RED, but **exactly 1** did (Inventory NFR-1) — the UNPROVEN backlog was *untested, not
  broken*, including all four slate-flagged PRIORITY-risk flows (Inventory NFR-8 slider reach-past-100
  rollback + Stock FR-12/13/14), which proved WORKING with strong assertions. **Verification at
  triage (not trusting the closeout):** re-ran `go build ./...` + `go vet ./...` (CLEAN) and
  `go test ./internal/purchasing/ ./internal/inventory/` (both `ok`) on the branch tree, then again
  on the merged tree — same result. The G4 discipline greps in the triage skill target night-crew
  *framework* files (`internal/journal`/`workorder`/`orchestration`, `replay_test.go`) that **do not
  exist in this HQ app repo**, so they were N/A; the diff is `tests/*.spec.js` + one Go test file +
  the 2-line fix + 3 `.night-crew` docs, no `*.html`/JS app file (so `task sw` correctly never ran).
  Merged `overnight-20260715` → `dev` `--no-ff`. **Chosen over holding any card for a cold re-read** —
  red-first capture + independent G6 diff-reverify + the attended build/vet/test on the merged tree is
  the evidence sign-off requires; a cold re-read of honest, all-GREEN proofs adds no signal the gate
  didn't already produce. Roadmap tracks A–E flipped slated → DONE. **Net cycle movement:** QA KR-1
  +~52 asserted flows (≥4 vacuous→genuine); Delivery KR 16/16 cards, ~52 flows UNPROVEN → WORKING
  [E2E-proven] (or honest PARK/UNTESTABLE); Eng KR-1 handled in T-12.

- **T-12 — Inventory NFR-1 graduated fix accepted (Eng KR-1 +1 → 0); the 3 prove-sweep PARK fix-WOs
  and 11 parked flows go to the planners, not an operator hand-pick.** The sweep's one deterministic
  RED (Inventory NFR-1 item-edit + confirm-vendor normalization) was fixed the same night — a 2-line
  `internal/inventory/handler.go` change adding `normalizeItemName` to the item-*edit* description
  path (`:1130`) and the confirm-vendor upsert (`:660`), mirroring the create-path idiom. Verified
  the diff directly at triage: minimal, idempotent (no create-path double-normalization), and the
  RED→GREEN is genuine (E4 committed the failing test first; pristine binary returned the raw name,
  fixed binary returns title-case). This closes BOTH surfaces the backlog NFR-1 item named in one
  fix, so it's marked DONE in `BACKLOG.md` and FR-4's "vendor upserted title-cased" PRD text is now
  accurate. The **11 PARKED flows** (photo/S3 ×3, offline/IndexedDB ×2, cron clock-seam ×4, thumbnail
  ×1, confirm-vendor gap folded into the fix) + **1 UNTESTABLE** (Onboarding FR-28 boot re-seed) were
  accepted as honest — each is a committed `test.skip`+reason or inline note, visible in the suite,
  not a silent drop. The **3 future fix-WOs** the parks imply (`WO-cron-clock-seam`,
  `WO-photo-s3-harness`, `WO-offline-indexeddb-harness`) graduated to `BACKLOG.md` as ready
  candidates. **Rule reaffirmed (T-10 precedent):** whether each fix-WO rides the next slate or waits
  is a *throughput / queue-placement* decision owned by the PjM (`/nc-slate-plan`) / PM
  (`/nc-pm-session`) / eng agents — triage surfaces the item + scope + risk; the planners decide when
  it runs. Chosen over scheduling them at triage because "operator owns the quality bar, not the
  throughput."

- **Standing flags after this run.** The DB flag stays satisfied (Docker pg16 canonical, unchanged
  by this run; re-arms only if the verify/merge DB path changes). **New standing note:** the triage
  build/vet/test spot-check covered only the two changed Go packages + a Go-level build — the full
  Playwright E2E suite (`task test:all`) was **not** re-run attended at triage (it needs the live
  pg16 stack and carries HQ's documented ~37–41 flaky-red pool). This is acceptable for a test-only
  sweep whose every card already ran its affected-seam subset red-first against a fresh-DB baseline on
  its own ephemeral stack; it re-arms as a triage requirement if a future run changes app code broadly
  enough that a full-suite attended pass is the only trustworthy gate.

## Cycle-gate closeout (overnight-20260716) — attested, pending triage ratification

Autonomous closeout run for Activity 5 (`cycle-gate`) — the last activity. 3 read-only cards, serial,
no app source touched (`*.html` / `internal/*` / `*.spec.js` all unedited; `task sw` never ran). Full
record: `reference/cycle-closeout-20260716.md`. **This entry is the overnight closeout; the operator
ratifies (or amends) it at `/nc-morning-triage`.**

- **T-13 — CYCLE GATE ATTESTED (PASS) under the operator's 2026-07-15 "Attest & waive" posture.**
  Scorecard **6 PASS · 2 PARTIAL · 1 WAIVED** across the 9 KRs. **Attested (3 criteria):** (1) *0
  known-broken flows* — built denominator 4→0 (T-9) plus the graduated NFR-1 fix (T-12); the four
  D-3/D-5 waivers are the only exclusions; the full-suite run independently confirmed all four
  repaired flows' own tests are GREEN and none regressed. (2) *Every repaired flow red-first* — one
  git-verifiable RED→GREEN pair (Inventory NFR-1: `1a0265e` precedes fix `77957c1`), four documented
  on the `card-actuals.md` record (the Activity-4 squash merges bundle test+fix, so the pre-fix
  failing commit is not standalone-reconstructable — **caveat carried, not hidden**). (3) *Median WO
  cycle-time baseline* computed: **N=23, median 22m28s** (serial ~19.4m / concurrent rolling-3
  ~23.4m; no pass/fail target this cycle). **Waived (2 criteria), carried to the next roadmap:**
  (a) *"full E2E suite green / 0 pre-existing reds"* → attested substitute **"0 new uncategorized
  reds vs the documented ~37–41 flaky baseline."** The bare full-suite run (isolated pg16, replicating
  `task test:all`) landed **387 pass · 38 fail · 0 flaky · 6 skip** (Playwright) + **1 env-gated Go
  red** (`internal/receipt`, no `ANTHROPIC_API_KEY`). Every red categorized against a documented cause
  (SW-blocked/offline, reload/tab-persistence, cross-test DB-pollution, and the documented
  data-dependent/persona fixture guards); a 7-test fix-adjacent isolation re-run confirmed the split
  (1 pollution→green-in-isolation, 6 structural/seed). `serviceWorkers:'block'` (`playwright.config.js:29`)
  makes a literal clean suite structurally unreachable. **0 uncategorized reds → PARK trigger did not
  fire** (`runs/2026-07-16-autonomous/DECISIONS-NEEDED.md` empty). (b) *"vacuous tests 23→0"* → ~4–5
  rewritten this cycle, ~18 remainder deferred as test-hardening WOs (`BACKLOG.md`). Chosen per the
  operator's explicit 2026-07-15 "Attest & waive" over "Stabilize the suite" (declined): the two
  unmeetable criteria collide with HQ's documented flaky pool + the SW-block + the deferred
  test-hardening WOs, so they are formally waived and their work carried forward, extending the
  D-3/D-5 waiver precedent.

- **Milestone boundary reached — next move is NOT `/nc-slate-plan`.** Activity 5 is the last activity;
  on this clean gate the morning path is `/nc-morning-triage` (review + merge `overnight-20260716` →
  `dev`), then `/nc-okr-session` to open the next cycle and consume the carried-forward backlog. The
  carried feedstock: the ~37–41 flaky/data-dependent + SW-blocked pool; the ~18 vacuous remainder
  (Ops FR-10/12, Ops FR-15, Onboarding 6 guards, Inventory ~40 guards); the 3 harness WOs
  (`WO-cron-clock-seam`, `WO-photo-s3-harness`, `WO-offline-indexeddb-harness`); F-1 (Ops NFR-3
  resubmit gate); the Onboarding video-pipeline E2E fixture (D-5 prove-path); F-2 (`users.html:122`
  orphaned `#s3`).

- **Standing flags after this run.** DB flag stays satisfied (isolated Docker pg16; `localhost:5432`
  untouched — held by unrelated `infra-postgres-1`, so this run stood up its own pg16 on port 5455 and
  tore it down `--volumes`). No app code changed → nothing to re-verify beyond the gate evidence
  itself.

## Morning-triage resolutions (2026-07-16) — `overnight-20260716`

Review verdict: 1 commit (`0ba751a`, `docs(night-crew):`); diff 100% docs-only (304 insertions,
1 deletion, 4 files, all under `.night-crew/`; **0 app-source files** — no `*.html` / `internal/*` /
`*.spec.js`; `task sw` not run). Independently re-verified on the branch tree: `go build ./...` +
`go vet ./...` **green**; G4 discipline greps **N/A** (the night-crew framework pkgs `internal/journal`
/ `workorder` / `orchestration` and `replay_test.go` do not exist in this HQ app repo — same as T-1/
T-7/T-11); `replay`/`testdata` untouched. `go test ./...` on the branch **and** re-run on the merged
tree = **5 pkgs `ok` + one pre-existing env-gated red** (`internal/receipt TestRunIngestCycle_ScenarioTable`,
AI-matching subcases with no `ANTHROPIC_API_KEY`) — identical to pre-merge `dev`, not merge-introduced
(0 Go changed). 0 cards parked; `DECISIONS-NEEDED.md` empty. Merged to `dev` `--no-ff` (`7f57d14`);
**push held at operator request** (dev staged, ready to push).

- **T-14 — CYCLE GATE ratified → SIGNED OFF; cycle closed. Ratifies the overnight T-13 attestation.**
  The 3 closeout cards (suite-baseline · attestation · scorecard) are accepted as attested; the gate's
  own evidence — not the closeout narrative — was re-verified at triage: the full-suite baseline's
  **387 pass · 38 fail · 0 flaky · 6 skip** (all 38 reds mapped to a documented category, 0
  uncategorized, PARK trigger did not fire) plus the merged-tree `go test` (5 ok + the one documented
  env-gated red) were reproduced attended, and all four repaired flows' own tests confirmed GREEN (no
  regression). **Scorecard stands: 6 PASS · 2 PARTIAL · 1 WAIVED.** Two gate criteria are **formally
  waived and carried** (extending the D-3/D-5 waiver precedent): "full suite green / 0 pre-existing
  reds" → the honest substitute *"0 new uncategorized reds vs the documented ~37–41 flaky baseline"*
  (met; `serviceWorkers:'block'` at `playwright.config.js:29` makes a literal clean suite structurally
  unreachable), and "vacuous tests 23→0" → ~4–5 rewritten, ~18 remainder deferred to `BACKLOG.md`.
  Chosen over holding the gate for a fuller clean-suite pass — the operator's 2026-07-15 "Attest &
  waive" ruling already resolved that the two unmeetable criteria collide with HQ's documented flaky
  pool + the SW-block + the deferred test-hardening WOs; forcing them now would either fail the gate on
  a known pre-existing condition or pull ~4–5h of net-new test-infra the cycle's hardening remit didn't
  ask for. **Net cycle close:** Product KR-1/2/3, Delivery KR-1/2, Engineering KR-1, QA KR-3 all PASS;
  QA KR-1/KR-2 PARTIAL-and-carried; Engineering KR-2 WAIVED-and-carried. **Milestone boundary reached**
  — the HQ hardening cycle is complete.

- **Next move is `/nc-okr-session`, not `/nc-slate-plan` (milestone boundary).** Activity 5 was the
  last activity; there is no next slate in this cycle. The next planning session opens a **new** cycle
  and consumes the carried-forward backlog as its feedstock (enumerated in T-13 and in `BACKLOG.md`).
  Per the standing T-10/T-12 rule, whether/when each carried item rides a future slate is a planner
  decision (PjM/PM/eng), not an operator hand-pick — triage only surfaces the items + scope + risk.

- **Standing flags after triage.** DB flag stays satisfied (Docker pg16 canonical; `localhost:5432`
  left untouched — held by unrelated `infra-postgres-1`). No new flags armed. `dev` is ahead of
  `origin/dev` by the merge (`7f57d14`) + the pre-existing slate-sign-off doc (`bdeb8b4`), **awaiting
  the operator's push** (the one sanctioned push; held this run by choice). `dev → main` promotion
  remains a separate decision, not folded into triage.

## Evening PM-session + grill-back resolutions (2026-07-16) — "Nothing silently lost" cycle

- **G-1 — Sequencing reversed at grill-back: straight to versioning; stages 1–2 deleted.**
  The morning-ratified "stages 1–2 ship before the versioning build" rested on the premise that
  crews were losing work in production *now*. At the grill-back the operator corrected the
  premise: **production has no active Operations users**, and wiping prod Operations (workflow)
  data is acceptable. With no bleeding to stop, stage 1's only unique value (speed-to-protection)
  vanished — its code is subsumed by the versioning build (`replaceTemplate` →
  create-new-version preserves identity by construction; dead-id rejection becomes the
  pinned-version existence check) — and stage 2 was interim UX relief with no audience.
  **Resolution (operator):** go straight to the versioning design gate + build. OKRs amended and
  re-validated (`okr validate` green): Delivery stage-1/stage-2 KRs deleted, repro-spec KR
  rewritten to the versioning semantic; Engineering stage-1a/1b KRs deleted, migration clause
  relaxed to clean-ship + attended wipe/reseed; Product/QA untouched. Roadmap: `stage1-*` and
  `stage2-template-updated-broadcast` tombstoned SUPERSEDED; two engine-trust fix cards added
  (`engine-approval-feedback-loud`, `engine-conflict-refetch` — PRD FR-6/FR-7 from the pass-2
  sweep). PRD rewritten and re-validated (`prd validate` green). FR-11 (prod orphan audit)
  dropped — auditing rows destined for the wipe is busywork. Chosen over keeping the ratified
  1→2→3: shipping interim protection to zero users buys nothing and double-touches
  `replaceTemplate`; the one gate that matters is the operator-signed design before build.
  *Rationale: sequence for the users that exist, not the ones imagined this morning.*

- **G-2 — Semantic reversed at grill-back: FROZEN-AT-SUBMIT beats run-pinned versioning;
  operator delegated, PM chose; versioning schema demoted to backlog.** Grilling the "run"
  definition exposed that the two candidate semantics had never been put head-to-head: (A)
  frozen-at-submit — an unsubmitted checklist always shows the current template on every device,
  submit freezes the record forever, rejection reopens it live; (B) run-pinned immutable versions —
  crews finish the run they started. The operator probed A's viability (two-device sync, manager
  rejection), then **delegated the choice to the PM** with a hard UX bar: multi-device sync always
  convergent — all 7 field types, sub-steps, submit/unsubmit, and list-view progress bars, every
  edge case walked. **PM chose A.** Rationale: the operator is the editor and wants corrections
  live; a 1–5 person single-kitchen crew doesn't need fleet-auditor run-pinning (B's home turf);
  A needs **no schema migration** (submit-freeze already exists as `template_snapshot`, proven by
  LC-02) and delivers the cycle theme as: loss is impossible (stable field identity), loud (422 +
  runner surfacing), or an explicit warned operator action (cut-field discard, INV-6). Under the
  delegated sync bar the PM also **promoted `EmitOp` fire-and-forget from backlog to requirement**
  (FR-5, transactional op emission — delayed propagation IS a loss under "always in sync") and
  upgraded the unsubmit/resubmit suspect to a convergence-matrix cell. Consequences: OKRs
  re-amended + re-validated (Engineering objective now stable-identity / loud-rejection /
  edit-propagation / convergence-matrix; no migration KR); roadmap Activities 4–5 renamed
  `editprop-*` (design gate unchanged in force: operator signs the OpenSpec change before any
  build card); stage-1/2 tombstones flipped to REVIVED-as-permanent; versioning backlog entry
  demoted with reason; PRD rewritten around INV-3′ + INV-6 and re-validated. The G-1 wipe is moot
  (no migration). Rejection rules recorded as PM proposals pending the FR-1 design sign-off:
  frozen record · live redo carrying answers · moot flags dissolve visibly. *Rationale: the
  architecture follows the semantic, not the reverse — and the semantic belongs to how this
  business actually runs.*

## Morning-triage resolutions (2026-07-17) — overnight-20260717

> This project records triage resolutions here (not a DESIGN.md §15x — no DESIGN.md exists in
> this repo; the run's HANDOFF step 6 names `ledger.md` as the destination). The run was
> **9/9 cards G6-verified, 0 parked, 0 open forks blocking triage** — the 5 items below are
> surfaced follow-ups the operator routed at triage.

- **Review + merge.** Re-verified the run branch cold on its final tree (not trusting the
  closeout): `go build ./...` + `go vet ./...` clean; `go test ./...` all packages `ok`; the
  G4 discipline greps are **N/A in this subject repo** — the orchestrator internals they guard
  (`internal/journal` / `internal/workorder` / `internal/orchestration`) don't exist here, so
  they trivially pass; `replay`/`testdata` untouched; `package.json`/lockfile untouched (the
  run's transient `workbox-build --no-save` install was NOT committed — confirmed by empty
  diff). Merged `overnight-20260717 → dev --no-ff` (`22cb7dd`); re-ran `go test ./...` on the
  merged tree → green. Frontend semver left at 1.0.3 (the bump belongs to `/save-project` at
  deploy, not triage). `dev` is ready for a normal `task prod:deploy`.

- **F-A — two-device convergence-cell no-retry flake → SCHEDULED as a roadmap card, over
  accept-to-BACKLOG.** The convergence-matrix suite (FR-7/A-5 proof, the Delivery KR) is green
  under the shipped `retries:1` (orchestrator re-verified 36/36 twice, incl. on an accumulated
  DB), but the two-device `text`/`temperature converges` cells fail ~3/6 under **no-retry** — a
  harness WS-timing sensitivity reproduced on base `733fa16`, so pre-existing to W-6, NOT a
  product defect. Operator chose to **schedule the hardening now** rather than backlog it: added
  `editprop-convergence-cell-hardening` (PLANNED) to roadmap Activity 6 — chase two-device WS
  timing to zero-flake under no-retry AND extend the W-6 *conflict* branch coverage beyond
  text/textarea to the remaining field types (F-A: ~6 ride the same `applyOp` path untested
  there). **Operator rider (recorded as a rule): no card may lean on this suite as a no-retry
  hard gate until this card lands.** Chosen over BACKLOG because the operator wants the
  Delivery-KR convergence proof trustworthy as a hard gate before other work leans on it,
  accepting the delay to other Delivery-KR cards. *Rationale: a convergence proof you have to
  retry isn't a proof.*

- **F-D — undeclared `workbox-build` devDependency → FIXED NOW (attended), over backlog.**
  `build-sw.js` `require`s `workbox-build` but it was undeclared and absent from clean
  checkouts, breaking `task sw` / `task test` / `task prod:deploy` on a fresh clone (prod
  already had it — deploy was never at risk; a clean-checkout gap only). Operator chose
  fix-now: declared `workbox-build ^7.4.1` (7.4.1 resolved) in `package.json` devDependencies +
  regenerated the lockfile; verified `node build-sw.js` runs clean and was idempotent (no
  `sw.js` drift, confirming the merged tree's SW was already consistent). Committed as
  `chore(build)` `3b1be67`, separate from this docs commit. Chosen over backlog because it's a
  ~3-min mechanical fix that unblocks every clean checkout.

- **F-B / F-C / F-E → graduated to BACKLOG** (operator: backlog; no competing option worth an
  operator pick — queue placement is a planner call per T-10/T-12). **F-B** — convert
  `CreateTemplateHandler` / `ArchiveTemplateHandler` from fire-and-forget `EmitOp` to the
  transactional `EmitOpTx` W-2 established (full INV-1 "0 accepted writes whose op is not
  durably queued" parity; no schema change). **F-C** — thread a `tx` through `approveSubmission`
  (repository.go) so `status='approved'` + feedback commit atomically (today status commits
  before the feedback loop, so a feedback-persist failure returns 500 `feedback_persist_failed`
  but leaves a partial commit — the card's requirement was still MET; atomicity is the
  follow-up). **F-E** — switch two onboarding persistence tests from `waitForTimeout(1500)` to
  `waitForResponse('/saveProgress')` (the post-reload assertion is still load-bearing; a fixed
  wait is a small flake-surface).

- **Standing flags after triage.** DB flag stays satisfied (Docker pg16 canonical; `:5432`
  left untouched). The **`workbox-build` clean-checkout flag is cleared** (F-D committed). The
  attended two-device convergence / `task sandbox:e2e` gate re-arms whenever the verify/merge
  path changes underneath it; `editprop-convergence-cell-hardening` (F-A) is now the tracked
  owner of the no-retry determinism, and the operator rider bars leaning on that suite as a
  hard no-retry gate until it lands. `dev` is ahead of `origin/dev` by the merge (`22cb7dd`) +
  F-D (`3b1be67`) + this docs commit — **pushed at triage close** (the one sanctioned push).
  `dev → main` promotion stays a separate decision, not folded into triage.

## Morning-triage resolutions (2026-07-18) — overnight-20260718

> Triage resolutions recorded here (no DESIGN.md §15x in this repo — the run's HANDOFF step 6
> names `ledger.md`). The run was **1/1 card DONE, G6 PASS, 0 open forks blocking triage** — the
> single item below (D-1) is a bounded coverage residual the operator routed, not a fork the run
> improvised around. This was the **last un-built card of Activity 6** (test-debt retirement).

- **Review + merge.** Re-verified the run branch cold on its final tree, not trusting the closeout.
  Footprint is **test-only + docs**: `tests/sync.spec.js` (+303/−21) and the run's `roadmap.md`
  card flip + HANDOFF/DECISIONS-NEEDED — **`sync.js` and `workflows.html` diffs are empty**
  (confirmed on both the impl commit and the merged tree), so no `task sw` and no production
  behavior changed. Checks run: `node --check tests/sync.spec.js` parses; the claimed structures
  are present (the `survivalCell` helper; a `W-6b` describe covering **exactly** yes_no +
  temperature + sub-step + checkbox, with **no** fail-note cases — matching the 2-parked claim);
  `go build ./...` + `go vet ./...` + `go test ./...` all green (backend unchanged — sanity only).
  The G4 discipline greps and `replay`/`testdata` guards are **N/A in this subject repo** — they
  guard the orchestrator internals (`internal/journal`/`workorder`/`orchestration`), which don't
  exist here. Full E2E was **not** re-run in triage: the diff is test-only, HQ carries its known
  ~37–41 pre-existing E2E reds, and the run's no-retry streak evidence (implementer 10/10 isolated
  + 0 target-cell failures under load; G6 independent 5/5 text, 6/6 temperature, 11/11 W-3, 4/4
  conflict ×2) is the load-bearing proof. Merged `overnight-20260718 → dev --no-ff` (`6291ef2`);
  re-verified the merged tree (parse + build + vet + test) → green.

- **Rider RETIRED — the no-retry hard-gate bar on the convergence suite is lifted.** The operator
  rider from 07-17 triage ("no card may lean on this suite as a no-retry hard gate until
  `editprop-convergence-cell-hardening` lands") is **discharged**: the card landed and the
  two-device `text`/`temperature` convergence cells are now demonstrably zero-flake under
  `--retries=0` (root cause was a stray WS-catch-up `loadMyChecklists` re-render clobbering a
  not-yet-persisted input to empty; fixed with deterministic waits inside `survivalCell` — gate on
  the autosave `POST /ops` 2xx commit signal, reopen to hydrate the committed draft for the
  baseline, wait on the post-cut myChecklists GET — LIVE + CATCH-UP assertions both preserved,
  G6-confirmed). This **unblocks `cycle-gate` (Activity 8, attended)** to adopt `task test` exit-0
  on the deterministic stack as a hard gate.

- **D-1 — 2 fail-note conflict-coverage types (footprint-blocked) → ACCEPT + track in BACKLOG,
  over graduate-a-card-now or accept-untracked.** Half 2 extended the W-6 LWW-409/`applyOp`
  conflict render from text-only to 4 answer types (yes_no, temperature, sub-step, checkbox — each
  red→green, G6-verified) but could not reach `fail-note text+severity` and `fail-note photo-URL`:
  `applyOp`'s `SET_FIELD` branch (`sync.js:405`) has no `_fail_note` unpack (that bundle is
  unpacked only by `hydrateFieldState`, `workflows.html:1480`, on load/reopen), so covering them
  needs an **out-of-footprint production change**. The implementer correctly declined to breach the
  card's test-only footprint and parked; G6 independently confirmed the block is real. Operator
  chose to **accept the residual and log it to BACKLOG** as an advisory (bundle candidate with F-B,
  which also touches the op-emission/apply path) rather than commit a roadmap card now — the 4
  common answer types are covered, fail-note concurrent-edit is a rare crew path, and server-side
  data is never lost (reconciles on next reopen; only a live-render staleness window). Chosen over
  *graduate-now* (a production card wants its own design/footprint/G6 next cycle, not test-debt —
  premature for a rare path) and over *accept-untracked* (the residual is worth carrying so future
  `sync.js` op-path work picks it up cheaply). Does not hold the card or the rider retirement.

- **Standing flags after triage.** Rider flag **cleared** (no-retry-gate bar discharged; the
  convergence suite is now an adoptable hard gate — `cycle-gate` owns exercising it). The attended
  two-device convergence / `task sandbox:e2e` gate re-arms whenever the verify/merge path changes
  underneath it — this run touched no production/verify path, so it stays satisfied. DB flag stays
  satisfied (Docker pg16 canonical). Frontend semver left where it is (the bump belongs to
  `/save-project` at deploy, not triage). Activity 6 is now **complete** (all test-debt cards
  DONE); Activity 7 (prod ops) stays operator-gated; Activity 8 (`cycle-gate`) is the last
  serialized card and is now unblocked. `dev` is ahead of `origin/dev` by the merge (`6291ef2`) +
  this docs commit — **pushed at triage close** (the one sanctioned push). `dev → main` promotion
  stays a separate decision.

## Slate-plan resolutions (2026-07-17) — slate-20260719 (`cycle-gate`, Activity 8)

> Recorded by `/nc-slate-plan cycle gate`. No DESIGN.md §15x in this repo — resolutions live here
> (HANDOFF step-6 convention). One fork resolved inline before signing the slate.

- **Activity 7 (prod ops) sequencing vs the cycle gate → "Gate now, prod KRs pending."** The gate's
  2 prod-dependent KRs (Delivery "prod parity"; QA "prod ghost item resolved") cannot attest green
  until Activity 7 (`prod-ghost-item-rename` + operator-run `task prod:deploy`) runs, and Activity 7
  is operator-gated/attended. Operator chose to **dispatch the read-only gate tonight** (attest the
  dev-side deterministic stack + score all dev-side KRs; mark the 2 prod KRs **PENDING** with exact
  verify commands) and run **Activity 7 attended AFTER** as the milestone's ship step. Chosen over
  *Activity-7-first-then-gate* (one complete scorecard, but the gate waits on attended prod work and
  you'd deploy ahead of the formal green attestation) and *one-attended-close-no-overnight* (most
  operator time). Rationale: correct order is attest-green-before-ship — the gate certifies the
  deterministic stack green, THEN you deploy that attested build and confirm parity. The milestone
  formally closes when the 2 PENDING KRs flip post-deploy.
- **Batch sign-off** given 2026-07-17; slate is `reference/slate-20260719.md`. Dispatch mode
  **serial** (degenerate — read-only closeout, one isolated env, no tracks). The `cycle-gate` card
  fans mechanically into 3 read-only cards + orchestrator closeout (mirrors overnight-20260716).

## T-15 — Cycle gate ratified (2026-07-19, overnight-20260719) — "Nothing silently lost" closed

> Recorded by the `overnight-20260719` cycle-gate closeout run. Evidence of record:
> `reference/cycle-closeout-20260719.md`. Gate **PASS attested** under the 2026-07-17 "Gate now,
> prod KRs pending" posture. No card parked (PARK trigger did not fire).

- **Scorecard: 11 PASS · 2 PARTIAL · 2 PENDING · 1 N/A (16 KRs).** PASS: Product KR1/2/3, Delivery
  KR1/2, Engineering KR1–4, QA KR1/2. PARTIAL: Engineering KR5 (waiver #1, below) + Delivery KR4
  (median not computable — the 07-17 run's 9 cards were not per-card timed; only `-0718`'s single
  card measured; baseline N=23/22m28s stands, **no median fabricated**; fix-forward = standing
  per-card timing table). PENDING (→ Activity 7, attended): Delivery KR3 (prod parity) + QA KR3
  (prod ghost item). N/A: QA KR4 (no schema migration shipped — frozen-at-submit deleted the
  versioning schema; `git log 2931adc..HEAD -- backend/internal/db/migrations/` empty, highest is
  `0070`).
- **Suite baseline (Card 1, isolated pg16 on `127.0.0.1:57606`, host `:5432` never touched).**
  Go units `go test -count=1 -p 1 ./...` → **all 7 pkgs ok, exit 0** (the 07-16 `internal/receipt`
  env-gated red is **gone** — passed without `ANTHROPIC_API_KEY`). Playwright fresh webServer
  (`CI=1`) → **450 pass · 1 fail · 0 flaky · 6 skip** (~16.3m). The 1 red
  (`workflows.spec.js › approved checklist … [LST-08 RUN-08]`, `#toast` hidden) is **cross-test
  DB-pollution** — an isolation re-run on a fresh single-test DB (`--retries=0`) **greened it
  (1 passed)**, the 07-16 Category-3 diagnosis reconfirmed. **0 uncategorized reds → no PARK.**
  Convergence suite `sync.spec.js` **39/39 passed × 3 consecutive `--retries=0` fresh-DB runs** —
  no-retry hard gate proven determinate; validates the 2026-07-18 rider retirement at the gate.
- **Waiver #2 (vacuous 18→0) — RETIRED.** Landed `3fd4d3f` (`vacuous-tests-18-to-0`). **SHA
  correction:** the slate/roadmap cite `3f68cc9`, a superseded pre-squash worktree object **not in
  `overnight-20260719` ancestry**; the branch-reachable landed commit is `3fd4d3f`. (Same class of
  squash-provenance drift as the Engineering-KR SHAs — the slate's `6a483d1`/`0d49f27`/`1c7c73c`/
  `72fffba`/`6c3aafb` are dangling pre-squash objects; landed squashes are `86bd09c` / `186e14c` /
  `3e5b921`. §2 corrected the mapping — "loud rejection" + "transactional emission" citations were
  scrambled; the behaviors are all landed and verified.)
- **Waiver #1 (`task test` exit-0) — SUBSTANTIALLY but NOT formally retired → CARRIED.** The suite
  fell from **38 reds (07-16 gate) → 1**, and the substitute criterion "0 new uncategorized reds vs
  baseline" is met, but literal `task test` exit-0 is **not** reached (the 1 pollution red → PW
  exit-1). Per the slate's explicit "mark PARTIAL, not PASS, never silently" clause, Eng KR5 = PARTIAL
  and waiver #1 carries forward. **Carried WO (next cycle):** fix the `approved checklist …
  [LST-08 RUN-08]` cross-test isolation so `task test` reaches literal exit-0 and formally retires
  waiver #1.
- **Red-first provenance (QA KR2) — attested on the WO/ledger record, NOT git.** The night-crew
  squash protocol bundles each fix's test+fix into one landing commit, so **0 git-bisectable
  red→green pairs exist this cycle** (unlike 07-16's single verifiable pair, Inventory NFR-1). The
  repro baseline `421ceee` was committed skip-guarded (kept green on purpose). Carried, not hidden
  (T-14 precedent) — corroborated by the per-card G6 re-reproduction in the 07-17/07-18 HANDOFFs.
- **Instrumentation gap (fix the 07-17 miss).** This gate re-affirms per-card wall-clock timing as a
  standing build-run output; `-0718` already re-adopted the harness-measured table. Until a fully
  instrumented delivery run lands, the Delivery-median baseline (T-14) is the only computable median.
- **Two invalid suite attempts, discarded and preserved** (`suite-logs/attempts/`): (1) Playwright
  reused a leaked foreign test server from another session (`reuseExistingServer:!CI`) — fixed with
  `CI=1`; (2) Go units ran against an unmigrated isolated DB (recipes `Fatalf` on missing
  `drift_check_results`) — fixed by migrating first + `-count=1`. Harness-provisioning defects, not
  product signals; recorded for the run's honesty.
- **Milestone boundary.** "Nothing silently lost" closed on dev-side evidence. Next: `/nc-morning-triage`
  (merge `overnight-20260719`), then attended **Activity 7** ship (flip the 2 PENDING prod KRs), then
  `/nc-okr-session`. `main` untouched; run branch never pushed.
- **Operator resolution at run close (2026-07-19, attended).** Waiver-#1 last mile (DECISIONS-NEEDED
  §C): operator chose **(a) graduate the test-hardening WO** to formally close waiver #1 — graduated
  to `BACKLOG.md` as `suite-isolation-approved-checklist` (`new`, next-cycle `/nc-okr-session`
  feedstock). Eng KR5 stays PARTIAL for this gate of record; flips to PASS when the WO lands literal
  `task test` exit-0. Two fix-forwards also graduated to BACKLOG: per-card timing instrumentation
  (Delivery-median) and `CI=1`+pre-migration gate run-mechanics.

## Morning-triage resolutions (2026-07-19) — `overnight-20260719`

> HQ records triage resolutions here (no `DESIGN.md §15x` in this repo — established convention
> since 2026-07-10). This is the operator-attended review + merge of the cycle-gate run.

Review verdict: 2 commits, both `docs(night-crew):`; diff **100% planning-docs** (20,524 insertions
across 24 files — the bulk is the `pw-results.json` evidence artifact + suite logs; **0 code/frontend/
test files touched**, code tree identical to `dev`). `go build ./...` + `go vet ./...` green on the
branch and on the merged tree. G4 discipline greps + `replay_test`/`testdata` checks **N/A for HQ**
(the night-crew orchestration packages those target — `internal/journal`, `internal/orchestration`,
`internal/workorder`, `replay_test.go` — do not exist here). Full-suite evidence is the run's own
Card 1 (450 pass · 1 fail · 0 flaky · 6 skip; the 1 red isolation-confirmed cross-test pollution;
Go units exit-0) against code identical to `dev` — not re-run at triage (nothing to regress). **0
cards parked** (PARK trigger did not fire). Merged to `dev` `--no-ff` (`a8854c3`).

- **T-16 — Cycle gate ratified at triage; "Nothing silently lost" closed on dev-side evidence.**
  The gate scorecard (11 PASS · 2 PARTIAL · 2 PENDING · 1 N/A, T-15) is accepted as the cycle's
  record. Waiver #2 retired; waiver #1 carried (reduced 38→1) as the graduated WO. No open forks
  remained at triage — DECISIONS-NEEDED §A empty, §C already resolved at run close (operator chose
  graduate), §B is the planned attended Activity-7 step, not a fork. Chosen over holding the gate
  for a cold full-suite re-run: the diff is docs-only, build+vet are green, and the run's Card 1 is
  itself the independent full-suite evidence against identical code — a re-run would only reproduce
  it (and re-spin the isolated pg16 for ~35m) with no new signal.
- **Activity 7 (attended prod ship) — DEFERRED (operator, 2026-07-19).** The 2 PENDING prod KRs
  (Delivery prod-parity, QA prod-ghost-item) stay PENDING; the operator chose to defer the attended
  Activity-7 ship (`task prod:deploy` + prod ghost-item rename + 2 verify queries, ~15m attended)
  rather than run it in the triage session. The milestone stays formally open until Activity 7 flips
  both KRs. Exact commands preserved in `runs/2026-07-19-autonomous/DECISIONS-NEEDED.md §B`. Nothing
  unattended is blocked by the defer. Chosen over do-now: operator attention budget — the ship is a
  short attended task the operator can run when ready; the gate's green attestation (the precondition
  for shipping) is already banked.
- **Standing flags after triage.** This run touched **no production/verify/DB path** (read-only
  closeout) — the attended two-device convergence / prod-deploy flags stay **satisfied** and re-arm
  only when the verify/merge/prod path changes underneath them. DB flag stays satisfied (isolated
  Docker pg16 is canonical). The **convergence no-retry hard gate** is now a discharged, adoptable
  gate (`cycle-gate` exercised it: 39/39 × 3 under `--retries=0`). Frontend semver untouched (no
  asset change; the bump belongs to `/save-project` at deploy, not triage). `dev` pushed to
  `origin/dev` at triage close (the one sanctioned push); `dev → main` promotion stays a separate
  decision. `main` untouched.

## T-17 — Activity 7 shipped + milestone closed (2026-07-19, attended)

The attended Activity-7 ship the gate deferred (§T-15/T-16). Milestone **"Nothing silently lost"
now fully closed** — markdown-mode close (this run predates the `night-crew` CLI scorecard
instrumentation, so `/nc-milestone-close` does not apply: `night-crew scorecard`/`okr grade` return
"no scorecard data / no metrics.jsonl"). Roadmap banner + both Activity-7 cards flipped to DONE.

- **Discovery — the deploy tooling was fiction.** `task prod:deploy` SSH'd the box to itself,
  targeted a nonexistent repo path (`~/projects/yumyums/hq`), and used container/image names
  (`yumyums-hq`) that never matched the running prod (`yumyums-prod` / `yumyums-purchase-orders:prod`,
  a Docker-Compose stack building from a **separate Windows clone** pinned to `main`). Prod was
  **405 commits / 2 months stale** (running May `b89c202`); `main` had never received the cycle.
- **Tooling fixed (commits on `dev`+`main`).** Added tracked `docker-compose.prod.yml` (context `.`,
  `backend/Dockerfile`, exact production `DB_URL` = `yumyums-dev-pg` + `search_path=production`,
  external `yumyums_default` net, `.env.prod` env_file). Rewrote `prod:deploy` to drive compose
  locally and **hard-sync** the prod clone to `origin/main` (a plain pull can't survive the Windows
  checkout's line-ending drift). Added `prod:rollback`; git-ignored `.env.prod`; wired
  `GIT_SHA`/`BUILT_AT` build-args.
- **Released.** Merged `dev → main` (405 commits, merge `6f45af5`; b89c202's stale scaffolding —
  root Dockerfile/`docker-compose.yml`/`prod/Taskfile.yml` — removed). Deployed. One crash-loop
  fixed: the Toast worker fail-fasts on the SFTP key that `.dockerignore` keeps out of the image →
  set `TOAST_SYNC_INTERVAL=0` in `.env.prod` (Toast inert in prod until a key is mounted).
- **Verified.** `task version` shows local == prod == `backend 0.1.3 / frontend 1.0.3`; public
  tunnel `https://hq.yumyums.kitchen/api/v1/health` serves it; migrations `56→70` applied to the
  `production` schema; app smoke-tested (shell 200, auth 401). Rollback image (`:prod-rollback` =
  the May build) + a pre-deploy `production`-schema `pg_dump` banked. → **Delivery prod-parity KR:
  PASS.**
- **`prod-ghost-item-rename`: verified no-op in prod.** Production had **0** empty-description
  items; the ghost item was in **dev** (public schema). Renamed the dev instance
  `'' → (Unnamed — needs review)`, 61 links preserved. → **QA KR3: PASS.**
- **Scorecard now 13 PASS · 2 PARTIAL · 1 N/A** (16 KRs). Carrying to next cycle: Eng "task test
  exits 0" (1 isolation-confirmed pollution red) and Delivery "median WO cycle".
- **Standing note — prod integrations now live.** `.env.prod` was provisioned "full" (operator
  choice): the Mercury receipt worker, alert queue, and Zoho Cliq now run in prod against the SAME
  external accounts as dev. **Watch the Cliq channel for duplicate alerts**; disable one side if they
  appear. Next move: `/nc-okr-session` for the next cycle (fold in the QA-coverage findings).

## T-18 — Morning-triage resolutions (2026-07-20, overnight-20260721)

Run merged to `dev` `--no-ff` (`e1d22ad`) after attended review: `go build` +
`go vet` green; `go test -count=1 -p 1 ./...` all packages ok on branch and merged tree (DB-backed
sync tests skip without a live pg — the DB legs' evidence is the run's own 2× ephemeral-pg16 runs,
impl + independent G6); G4/replay checks N/A for HQ (07-19 precedent); footprint diff matches the
closeout exactly — test-only + docs, **zero production files**. 4/4 cards DONE, 4/4 G6 PASS, 0
parks. Waiver #1 formally retired (Eng KR5 PARTIAL → PASS). Per-card actuals appended to
`reference/card-actuals.md`. Resolutions below answer DECISIONS-NEEDED §B1–B5 plus the two items
HANDOFF left to the operator.

- **B1 — Activity-2 design SIGNED (2026-07-20): A4 = Option (i), D2 = Ungrouped, rider (b)
  REWRITTEN to umbrella semantics.** `designs/prove-surface-gating-and-endpoints.md` §8 now records
  the signature; Activity 4 (5 Feature WOs) is unblocked. **A4:** two dedicated per-tab slugs
  (`inventory-trends`, `inventory-cost` in `hq_apps` via `SeedHQApps`) — chosen over the
  `app_permissions.tab` column for zero schema risk (no migration → no NFR-3 down-migration proof /
  pre-deploy backup), zero `/me`+Users-UI backend change, and trivially reversible seed rows; the
  draft, implementer, and G6 all converged here. **D2:** linked-but-groupless lines (the
  `'(no itemized receipt)'` sentinel, group-deleted items) bucket as an explicit **"Ungrouped"**
  pseudo-group — chosen over folding into "Unlinked $X" (would misstate the completeness note) and
  over dropping (breaks the AC-6 reconciliation identity). **Riders:** (a) per-week `unlinked`
  array KEPT; (b) **REWRITTEN by operator rider — "App grant = All tabs granted. They should not be
  considered separate objects."** The signed semantics are UMBRELLA: a whole-app grant includes
  every gated tab of that app automatically; per-tab grants exist for narrower tab-only access; the
  `RequirePermission` check passes on (tab slug ∨ whole-app slug ∨ superadmin). This rewrites the
  draft's strict reading (which the operator explicitly rejected) while keeping per-tab granularity
  as the go-forward convention. (c) tab-grant-without-app-grant (tile hidden, direct URL works)
  SIGNED as expected behavior — tab grant is the gate, tile is launcher UX; Users UI should nudge
  admins to co-grant.
- **B5 folded into the gating card (operator: "fold into gating card").** The
  `inventory-tab-gating` WO's scope now includes an authz gate on
  `ApproveSubmissionHandler`/`RejectItemHandler` (`backend/internal/workflow/handler.go:728-753,
  793+` — today any authenticated role can approve/reject). Chosen over a backlog entry (closes
  sooner, same middleware work) and over accept-as-is. The exact role rule is specified at slate
  time (expected: approvers + admins/superadmins).
- **B2+B3 — one production card promoted to the next slate: `replay-fetchstorm-gate`.** Operator:
  "promote it." The ungated `SUBMIT_CHECKLIST` replay re-fetch (`sync.js:443`) gets the same
  `(runner open) ∨ !silent` gate its `APPROVE_ITEM`/`SAVE_TEMPLATE` siblings already carry
  (production one-liner, pattern proven in-file since the 2026-07-18 fix); the same card hardens
  the successor intermittent `sync.spec.js:1198` (pre-existing, load-sensitive, red 2-of-3 G6 legs
  that included it — sits directly downstream of the storm) and reverts A2's test-side
  `checkAllWithRepair` workaround to plain clicks once the storm is gone. Chosen over
  backlog-and-wait (phones keep the reconnect fetch-flood + mid-fill clobber window) and over a new
  narrow waiver (recreates the machinery this run just retired). Exit-0 status stays honestly
  "achieved-and-reproduced, not asserted deterministic" until this card lands.
- **B4 — operator rider, recorded verbatim: "Everyone should see live ops."** The live-sync
  fan-out contract becomes: every user with access to the entity receives its live ops — no
  filtering by role or assignment type. This RATIFIES deployed behavior (the recipient query,
  `ops.go:521-530`, already does not filter `assignment_role`) and supersedes FR-7's narrower
  "admins ∪ assignees" wording; approver inclusion is intended, not accidental. The
  `TestResolveEntityAccess_ApproverIncluded_CurrentBehavior` pin flips from reviewer-NOTE to
  contract (comment update rides `replay-fetchstorm-gate` or the next sync card). Chosen over
  excluding approvers (a production change with no user benefit on a 1–5 person crew).
- **`percard-timing-instrumentation` flipped DONE (operator).** The run's harness-measured
  per-card table (impl/G6/merge legs, epoch-stamped `timings.log`) is exactly the standing output
  the card asked for; future build runs keep producing it as standing practice, and the cycle gate
  computes the Delivery-KR3 median from the accumulating `card-actuals.md` rows.
- **Preference capture + decisions audit SKIPPED — not deployed to night-crew `main` (operator
  rule, recorded).** The triage skill's capture-on-answer step names `night-crew preferences
  propose` / `night-crew decisions audit`, but both exist only on night-crew `dev` (57 commits
  ahead); hq's tooling tracks `main` (`nc:update`). Operator: **"Only should consider whats been
  deployed to main."** Both candidate preferences (umbrella grants; everyone-sees-live-ops) were
  offered and are withdrawn as not-applicable — the riders live in this section and the signed
  design instead; they can be re-offered if/when the preferences machinery ships to main. The
  skill-vs-deployed-tool skew is recorded as `design-findings-nightcrew.md` NF-3. (Triage-process
  note: the installed CLI was briefly rebuilt from dev mid-session to inspect the subcommands —
  reverted to a `main` build the same session.)
- **Standing flags after triage.** Test-only + docs run: prod-deploy / attended-convergence flags
  stay **satisfied** (no verify/merge/prod/DB path changed; they re-arm when that path changes —
  note `replay-fetchstorm-gate` WILL touch `sync.js`, so the attended two-device convergence flag
  re-arms when that card lands). DB flag satisfied (ephemeral pg16 canonical). Frontend semver
  untouched (no asset change). `dev` pushed to `origin/dev` at triage close (the one sanctioned
  push); `main` untouched — `dev → main` promotion remains a separate decision. FR-12 Cliq-dup
  watch continues over the cycle (nothing observed this run).

## T-19 — Morning-triage resolutions (2026-07-20, overnight-20260722)

Run merged to `dev` `--no-ff` (`05dc053`) after attended review. Independent re-verification on
the merged tree: `go build ./...`, `go vet ./...`, `go test ./...` all exit 0; G4 discipline greps
clean (and structurally N/A — HQ has no `journal`/`workorder`/`orchestration` packages, per the
07-19 precedent); `replay_test.go` + testdata untouched. Footprint 14 files, +2264/−45.
**3 cards merged (S1 PARTIAL, F2, F4), 1 PARKED (F1), 1 blocked by the park (F3), 1 dropped by
budget discipline (F5). 4/4 G6 adversarial reviews changed the outcome** — one park, two revision
rounds, one premise correction.

**Decision 29 — Trends shows food spend only (COGS allowlist), not the whole bank feed.**
Chosen over leaving the signed design's unfiltered query, and over an everything-with-non-food-split
variant. The bank feed carries rent, insurance, software and fuel; the signed design §2.2 SQL sketch
had no `mercury_category` filter, so a chart titled "spend by group" would have shown rent as a
group and over-reported against payroll's COGS by an unbounded amount (G6 measured +500.00 on a
two-event synthetic; production magnitude unmeasured). Trends now filters to the same allowlist
`period-summary` is constructed with. The everything-split option was declined as making one tab
answer two questions. **This amends signed design §2.2** — the implementer was correct to follow
the sketch verbatim and flag rather than silently patch; the defect was in the design.

**Decision 30 — Unreviewed receipts are excluded from the chart and surfaced as a completeness
note.** Chosen over an "unreviewed" pseudo-group bar and over continuing to ignore them silently.
Structural constraint drove this: unreviewed receipts have no linked line items (linking is what
review *does*), so they cannot be bucketed into a week×group cell at all — they can only ever be a
note. `period-summary` counts them as a lump, which is why the two numbers diverged. The note
mirrors the existing `unlinked` treatment rather than inventing a second idiom.

**Decision 31 — Trends reports attributed spend; unattributed money goes to the completeness note
rather than being prorated across groups.** Chosen over the signed design's proration (which
reconciles totals but smears the gap across categories) and over abandoning the payroll cross-check
entirely. When a receipt's line items don't cover its subtotal — a delivery fee, an unitemized
remainder — the current proration inflates every food line to swallow the difference, so per-group
numbers are silently overstated with nothing on screen indicating by how much. This is the defect
that broke F1's AC-6 five ways (G6 probe B1: trends 99.00 vs period-summary 100.00 on a receipt
with an unitemized fee — the *normal* case). **The reconciliation identity is redefined as:
`cells + unlinked + unitemized remainder + pending == period-summary`** — an identity that holds on
messy real receipts, which the old one did not. This is the amendment that un-parks F1 and unblocks
F3.

**Decision 32 — F2's non-positive-revenue guard RATIFIED.** The run extended signed §2.3's
"zero-revenue → NULL" rule to "non-positive-revenue → NULL" after G6 found a refunded row producing
`food_cost_pct: -500000` and ranking **#1 in the "best" list**. Recorded as executing the design's
evident intent ("never a divide-by-zero or ∞"), not as a new decision — but it is a written change
to a signed rule and is ratified here explicitly rather than absorbed silently, per the run's own
flag.

**Decision 33 — the residual/unattributed-money gap follows Decision 31.** `menu-cogs` publishes
`unallocated_cogs`; the Cost endpoint dropped it, so summing the Cost tab's ingredient column
under-reports true COGS with nothing indicating a residual exists (G6 rated this the card's most
substantive gap). Resolved by consistency with Decision 31 rather than a separate operator call:
unattributed money is surfaced, not hidden. Applies to Cost as it now does to Trends.

**OPEN — investigation, not resolved tonight: food cost as a drifting long-term average.**
Operator, on the 0%-food-cost fork: *"The idea is that the cost of the food item is a long term
average. What's most useful is to see how the average is increasing or decreasing over time."* The
current design treats food cost as a fixed-12-week snapshot; the operator wants a rolling average
and its **direction of travel**. This is not a fix to the 0% bug — it dissolves it, since a
long-term average is indifferent to whether a bulk purchase landed inside an arbitrary window.
**The 0%-dish fork (F2-a) is therefore left UNRESOLVED** pending this; no third `unallocated` reason
string was coined. Routed to the next planning cycle.

**OPEN — investigation, not resolved tonight: margin with and without discounting.** Operator, on
the red-negative fork: *"it would be useful to know the margin on items with and without the
discount."* **Verified during triage: the data does not exist.** `daily_menu_sales` stores only
`menu_item_id, business_date, units_sold, gross_amount, updated_at` — no discount or comp field. The
comparison would require capturing discount/comp data from Toast during sync, upstream of both tabs.
**The red-negative fork (F4) is therefore left UNRESOLVED**; the tab still reds any negative margin,
including comped dishes that never sold.

**Pattern noted for the next planning session.** Both open investigations are the same shape: the
operator asked for a *comparison* where the current design shows a *single number* (average-and-trend
rather than window-snapshot; with-and-without-discount rather than one margin). Neither is cleanup.
Worth treating as a product thread in its own right at the next PM session.

**Still open, not asked this triage (operator fatigue — deferred rather than forced):** F5
`inventory-tab-gating` priority for the next run, and the attended two-device convergence check.
Both are stated in the handoff with recommendations and remain the operator's call.

**Standing flags after triage.** **Attended two-device convergence check RE-ARMED and NOT run** —
production `sync.js` changed (S1's gate) and `task sw` regenerated the service worker. Prod deploy
NOT done (attended, rides the cycle gate). Frontend semver untouched (1.0.3) — bump belongs to
`/save-project`. DB flag satisfied (ephemeral pg16 canonical throughout; host `:5432` never touched).
**The Cost tab and its data ship logged-in-only** — F5 dropped, so per-tab access control does not
exist; B5 (approve/reject authorization) also remains unclosed, pre-existing. FR-12 Cliq-dup watch
continues over the cycle (nothing observed).

---

## T-20 — Morning-triage resolutions (2026-07-21, overnight-20260720c)

Run merged to `dev` `--no-ff` (`c2cfc13`) after attended review. Independent re-verification on the
merged tree: `go build ./...`, `go vet ./...` clean; `CI=1 go test -p 1 -count=1 ./...` 8 packages
ok / 0 FAIL; `CI=1 task test` on a clean DB **528 passed / 6 skipped / 0 failed / 0 flaky**. G4
discipline greps structurally N/A (HQ has no `journal`/`workorder`/`orchestration` packages, no
`replay_test.go`), per the 07-19 precedent. Footprint 31 files, +6346/−22 for the run, plus four
attended triage commits.

**3 cards merged (F1, F3, F5), 1 PARKED (D1).** F5's G6 caught a live authentication bypass the
card would otherwise have shipped. The branch also carried post-closeout attended work: the `/ops`
authz sweep, the `requires_approval` fix, the cross-contamination audit, and the `:1198` flake
reproduction that **retracted the run's own headline finding**.

**Decision 34 — `/ops` SAVE_TEMPLATE and ARCHIVE_TEMPLATE are gated to admin, matching their REST
twins.** Chosen over splitting the two shapes (authoring open, archiving admin-only) and over
widening REST to match `/ops`. Resolves DECISIONS-NEEDED §1-B. The `EXCEPTIONS` bucket in
`tests/ops-authz-coverage.spec.js` empties as a result; it is kept as a tripwire so the next
divergence cannot ship silently, not as a waiver mechanism.

**Decision 35 — the `/ops` router carries a STANDING RULE: every op branch enforces the same authz
as its REST twin. Divergence is never permissible.** Chosen over allowing divergence with a
recorded justification. Resolves §1-C. This is the rule that stops a third recurrence of the
dual-path bypass class (F5's G6 found the first; the sweep enumerated the rest). Two doors, one
mutation, one authz answer — and the gate belongs **inside the mutation**, not at either boundary,
which is the shape both `requireReviewAuthz` (`8c71022`) and `requireApprover` (`0057638`) now take.

**Decision 36 — the Users-tab grant model is a DATA boundary, not a UI convenience. Operator,
verbatim: "If an employee does not have access to the app (or access to the app's tab), then they
should NOT be able to access the view / tab / data."** This resolves §6 far more broadly than the
question asked. §6 asked whether `inventory-cost` was meant as confidentiality or tidiness; the
answer is confidentiality, and the finding that fell out of asking is larger than the fork:

> **The Users tab offers 11 grants. The backend enforces 2.** Every `RequirePermission` call in the
> server is `inventory-trends` and `inventory-cost` (both shipped by F5 this cycle). The other nine
> — including `inventory` itself, `operations`, `purchasing`, `onboarding`, `users` — are checked
> nowhere. `isAdmin`/`manager` *role* checks protect some endpoints, but roles are a different axis
> from grants. Revoking a grant today removes launcher tiles and hides tabs; it does not change what
> the holder can read from the API with a cookie.

This is not an F5 defect — F5 built the mechanism the signed design scoped it to, and built it well
(umbrella semantics, fail-closed on DB error, 13 attack variants). Nothing migrated the pre-existing
surface onto it. **Operator: no live exposure today (no non-admin crew hold accounts), but this must
be fixed before go-live.** Graduated to a new roadmap card, `grant-enforcement-parity`, sized as the
largest open correctness item in the backlog. Evidence is source enumeration, not a live grant-less
curl; the card should begin by proving it live.

**Decision 37 — the slate template's OpenSpec clause is a NIGHT-CREW TOOLING defect, pushed back to
that repo as urgent; it is not fixed in hq.** Resolves §2, but not as either option offered. The
recommendation was "amend the slate to cite GSD" — **wrong, because the operator is refactoring away
from GSD**, so pointing the template at GSD would encode a convention on its way out. Sequence
instead: (1) urgent backlog item in the night-crew clone — the slate template dispatches every card
with mechanics (`openspec validate`, `OpenSpec-Change` trailer, archive) that misfire in any target
repo without an `openspec/` tree, and four cards silently worked around it in one night; (2) a
separate investigation into whether OpenSpec's pros outweigh its cons; (3) *if* OpenSpec is kept, a
formal refactor requested for hq. No hq change lands from this decision.

**Decision 38 — the Go and Playwright suites get separate databases (`hq_test_go` / `hq_test_e2e`).**
Operator's constraint at triage was narrow and clear — *"I only care that there are no db conflicts"*
— with the choice of mechanism delegated. Chosen over leaving one shared `hq_test` with a serial
convention, and over per-run ephemeral databases (correct, but out of scope for a triage). Audit
surface #3: every Go `TestMain` truncates tables — `internal/sync` truncates `users` — so
`task test:go` alongside `task test` would log every browser context out mid-suite. `-p 1`
serializes Go packages against each other and does nothing about this. **Proven, not asserted:**
`internal/sync` + `internal/workflow` ran green against `hq_test_go` while `persistence.spec.js` ran
green against `hq_test_e2e`, concurrently. `db-test` also gained an `ALLOW_TEST_DB_ON_DEV_HOST`
guard, defaulted permissive because dev/test/prod genuinely share the `yumyums-dev-pg` cluster today
(surface #4) — so it makes that a visible choice that will fail loudly once a real test cluster
exists.

**Decisions 39–41 — the three selected one-line contamination fixes** (surfaces #9, #5, #2). #9:
`DB_PORT` default 5432 → 5433, because host `:5432` is bound by `infra-postgres-1` from the
slack-trading project and has no `yumyums` role — this fired live *during this triage*, costing a
full E2E leg. #5: `ZOHO_CLIQ_*` / `SMTP_*` added to the blanked set in `playwright.config.js`,
because the root Taskfile's `dotenv: ['backend/.env']` injects 21 live credentials into every task
from the main checkout and `alertQ.Start` is gated by neither `schedulersDisabled` branch — an E2E
run could deliver a real Cliq message and a real SMTP email to live crew. #2:
`reuseExistingServer: false` unconditionally, killing the `:8199` latch that has cost four runs.

**Correction to the audit, found while answering the operator's "how is this possible?".** The audit
recorded a dev server (PID 75921) running since Jul 18 "against the live dev database **on the
Windows box**." **The location is wrong:** `100.70.200.55` is *this* box's own Tailscale address, so
that `DB_URL` is the local `yumyums-dev-pg` container reached the long way round. What the check did
surface is worse than the original claim and was not in the audit: **one Postgres cluster holds dev
(`public` schema), prod (`production` schema), and `hq_test`, under one role and one password** —
prod is separated from dev by nothing but a client-supplied `search_path`. Recorded as the sharpest
instance of surface #4. The genuine live-side-effect exposure is not the database but the
credentials: PID 75921 has held live Mercury production, Anthropic, Zoho Cliq and SMTP keys for
three days with `E2E_DISABLE_SCHEDULERS` unset.

**Two reviewer errors, recorded because both are instructive.**
(1) The `DB_PORT` fix was cited as verified when it never ran — `playwright.config.js` carries its
own independent `dbPort` default, and every verification leg had passed `DB_PORT` explicitly on the
command line. The first run to actually rely on the new default died at startup. *Two files, two
defaults, one silently unused.*
(2) **P3a was violated by the reviewer within hours of P3a being written.** The `persistence.spec.js`
flake fix kept its red-first forcing wait in the committed test; that wait asserts a condition that
only holds in narrow targeted context, so it failed outright in full-suite order — trading a rare
strict-mode flake for a reliable failure. A targeted 15/15 green was read as proof. The `#fill-body`
scoping alone was always the whole fix. **Targeted-subset green is not evidence for a fix to an
order/state-dependent test.** Both are written into `card-actuals.md`.

**Flake dispositions.** `sync.spec.js:1198` — the run's "decisively refuted / not flaky" conclusion
is **RETRACTED**; reproduced at 16% (4 red / 25 `--retries=0` legs), 20% under a concurrent suite,
with a named mechanism and the card **re-aimed** (test-side; no production change; no timeout
increase can help). `persistence.spec.js` FLD-R3/R5 — fixed; was an unscoped `text=✓` matching the
approvals-list ✓ for its own submission, **not** cross-file contamination. Two new observations
folded into the re-aimed card: `sync.spec.js` LST-17 and `workflows.spec.js` GATE-04, both
passed-on-retry, neither previously recorded.

**Standing flags after triage.** **Attended two-device convergence check REMAINS ARMED and NOT run**
— armed since the 07-22 `sync.js` change; D1 left `sync.js`/`sw.js` byte-identical so this cycle did
not re-arm it, and it did not clear it either. Prod deploy NOT done. Frontend semver untouched
(1.0.3) — the bump belongs to `/save-project` at deploy time. DB flag now materially improved
(suites separated, proven concurrent-safe) but **surface #4 is open**: dev, prod and test still share
one cluster under one credential pair. `stash@{0}` still holds unattributed WIP in a slot shared by
five worktrees. FR-12 Cliq-dup watch continues.

## T-21 — Morning-triage resolutions (2026-07-23, overnight-20260724)

**Run reviewed and merged.** 2/2 slate cards (G1 `grant-enforcement-parity`, S1
`syncspec-deflake`) merged to `dev` at `f776578`, both G6: APPROVE. Independent attended
re-verification on the branch tree: `go build`/`go vet`/`go test` green; full Playwright suite
**542 passed / 0 failed / 6 skipped in 20.4m with zero retries fired** (config allows 1; none
used — materially a no-retry green; box load 1.9–2.6, all of it the suite's own). Conflict log
audited: 2 merges, both CLEAN, both logged with intents read; both cards' three-field
merge-intents present and truthful against the diff (the `main.go` wiring matches G1's
"must survive" list line-for-line). Production frontend untouched (diff-verified) — the
attended two-device convergence check unchanged by this run.

**Decision 42 — `/photos/*` stays authenticated-only as a documented exception; the durable fix
is a key-binding card, not a route gate.** Chosen over a union-of-four-grants gate (rejected as
cosmetic: every crew member holds ≥1 app grant, and `missing_grant` would name the wrong thing)
and over ratifying the gap as permanent policy. Rationale: a per-app route split without binding
photo KEYS to their owning app/record still lets any granted user fetch any photo through their
own app's route — the boundary lives in the key→owner relation, not the route. Backlogged
("`/photos/*` key-binding gate"); the parity-spec documented exception stands until it ships.
Grants remain a DATA boundary (decision 36); a gate that doesn't actually bound the data doesn't
discharge it.

**Decision 43 — the `GET /inventory/items` (inventory ∨ purchasing) READ is RATIFIED.**
Chosen over revert + catch-and-degrade. The exposure is item names/groups/locations/photo URLs
(G6-verified: no cost/price/COGS fields — those stay behind inventory-gated endpoints); item
writes stay inventory-only. The alternative leaves purchasing-only crew with a broken order form
or forces dual grants — a wider real-world exposure than the READ itself.

**Decision 44 — E5's no-retry attestation is GRANTED with a recorded waiver for LST-17's single
under-load red.** Chosen over a formal quiet-box re-leg. Evidence: 2× quiet-box 541/0/6
(S1 impl + its G6, loads 1.59–1.93), the attended 542/0/6 zero-retries leg above, vs 1× 540/1/6
under measured foreign load (2.38→4.37), isolation-green, mechanism known. LST-17 REMAINS
FLAGGED load-sensitive — "rare, mechanism known" is not laundered into "not flaky" (standing
rule, T-20 flake dispositions). E5 grades at the cycle-gate close-out with this waiver cited; a
future LST-17-class regression is still caught by the quiet-box no-retry gate.

**Recorded, no decision needed.** (a) The fabricated-completion-notification incident during
G1's final suite leg — the implementer discarded all injected notifications (future timestamps,
tallies from a 0-byte log), verified process exit itself, and read tallies only from the real
log after genuine exit; correct handling, kept visible here. (b) `night-crew backlog check
--file` rejects hq's BACKLOG.md wholesale — no entry carries a B-NN handle; the file predates
the backlog-store spec. New items keep the file's own shorthand; the spec/tooling mismatch is a
night-crew-repo finding (decision-37 precedent: tooling defects push back to that repo). (c)
Stale `:1198`/`:525` line anchors: the tests are now located by title (`-g "temperature answer
converges"`; FLD-LIVE-02 by name) — annotated at the promoted BACKLOG entry; frozen records
(slates, evidence docs) keep their historical anchors.

**Standing flags after triage.** Prod deploy NOT done; frontend semver untouched (1.0.3) — D2
PENDING-deploy, settles at the attended cycle-gate close-out (`/save-project` → `task
prod:deploy` → `task version` parity → 2/2 tab screenshots). Attended two-device convergence
check unchanged by this run (no production `sync.js` change landed). `/photos/*` documented
exception stands pending the key-binding card. FR-12 Cliq-dup watch continues (D4). The
`cycle-gate` roadmap card remains PLANNED — the boundary close-out (P4 interpretation, D2 ship,
D4 confirmation, E2 0%-food-cost note) is the remaining attended work of the cycle.

## T-21a — Post-ship play-test ruling (2026-07-24)

**Decision 45 — gated-tab semantics REVERSED from the T-18 umbrella rider; backlogged, not
urgent.** Operator play-tested the shipped gating in dev (user "Jim B": Inventory app grant,
no per-tab grants) and found the umbrella rule ("App grant = All tabs granted", §8 amendment 1)
defeats the Cost/Trends confidentiality goal: a crew member who needs Stock/Purchases daily
cannot be kept away from margins — "inventory-except-cost" is unexpressible. The new rule, in
the operator's words (verbatim): **"If there is a granular permission for a tab and it does not
exist, the tab should not be visible. If no granular permission exists, then the tab should be
visible by default."** I.e. a tab with a registered per-tab slug (`inventory-trends`,
`inventory-cost`) requires that explicit grant — the app grant no longer implies it; tabs
without their own slug stay covered by the app grant. Implementation is deliberately deferred:
**not urgent, backlogged** (no crew accounts hold prod grants, so no live exposure). When
built: regression test FIRST (app grant + no tab grant → tab absent + endpoint 403, red
against current behavior), then the contained edit — two `RequirePermission` mounts drop their
umbrella arg, `hasTabGrant` in inventory.html drops the `'inventory'` disjunct, umbrella-
direction tests flip. Note the shipped behavior matched the SIGNED design exactly — this is a
spec reversal from play-test evidence, not an implementation defect.

## T-21b — D4 Cliq-duplicate ruling (2026-07-24)

**Decision 46 — the duplicate-alert incident is root-caused and the "disable one side" remedy is
implemented: outbound alert delivery is now OPT-IN (`ALERTS_ENABLED=1`, set only by
`docker-compose.prod.yml`).** Operator-observed evidence: the last duplicated Cliq item —
"Shopping list completed with 2 … Add Loc Test … Unassigned: Aisle Item …" — dates to
**2026-07-21**, carries test-fixture names, and nothing since (3 clean days). Root cause: dev-side
senders holding the SAME live Zoho/SMTP creds as prod — the E2E env leak (closed at T-20
decision #5) and any dev server started from `backend/.env`. The class recurred live during this
very triage: a dev server started 10:02 on 07-24 (the gating play-test) held live creds with
schedulers enabled; killed attended. The durable fix gates delivery inside the queue
(`internal/alerts`): `Config.Enabled` from `ALERTS_ENABLED == "1"` (strict — any other value
fails CLOSED/silent), checked at `deliver()` so every enqueue path is covered; startup logs
`delivery_enabled`. Transactional email (invite / password reset, `internal/users`) is
deliberately NOT gated — admin-initiated, never duplicated by dev-vs-prod racing an event,
already no-ops on blank SMTP config. Red-first: `internal/alerts/config_test.go` captured red
(Enabled undefined) before the fix; 3 tests green after; full Go suite green. Prod behavior
unchanged — the compose flag lands in the same commit and rides the next deploy. **D4 settles:
incident handled and recorded, one side disabled.**

## T-21c — Play-test escaped-defect note (2026-07-24)

**Recorded, bears on the milestone close (no decision yet — product ruling queued to the next
planning session).** Operator play-testing found the operations checklist rendering out of
sync between Jamal C and Jim B on dev. Reproduced attended in fresh headless contexts:
per-user hydration divergence — the viewer's own last submission (rejected vs approved)
determines what the "shared" checklist shows, and the approved-side viewer's clicks silently
no-op. Server state verified byte-identical for both users; grants, /ws, network, caches, and
the 07-22 `sync.js` change all ruled out. This is a **newly discovered convergence-matrix
cell** (cross-user × cross-cycle-state) found at cycle end — E4's "0 cells red at cycle end"
should be graded with this on the table: the 32 covered cells are green, AND operator play
found a 33rd the matrix never enumerated. Evidence + repro:
`reference/sync-crossuser-hydration-20260724.md`; backlogged pending the product ruling.

## T-21d — Prod SW-update pipeline defect found and fixed at play-test (2026-07-24)

**Recorded, fixed same sitting (no fork).** Operator's prod check showed both new tabs still
rendering the pre-cycle "Coming in a future update" placeholders in Safari, while the server
verifiably served the new files. Root cause: `version.json` is a git-ignored `build-sw.js`
artifact that the Docker pipeline never generated, so prod 404'd it — and `sw.js` PRECACHES
it, and Workbox aborts the whole service-worker install on any precached 404. **No returning
client could install an updated service worker against prod — every deploy since the
single-image pipeline (2026-07-05) shipped server-side only; returning phones stayed pinned
to their cached frontend.** Fresh contexts (tests, curls, first visits) fetch over the
network, which is why nothing automated ever caught it: the escape was only visible to a
RETURNING real client. Fix `b45bc3e` (backend 0.2.2, deployed `32afb39`): the Dockerfile
builder stage generates `version.json` from the authoritative `Frontend` constant into the
embedded assets. All 22 precached URLs verified 200 on prod post-deploy. Lesson for the QA
methodology: prod-parity evidence must include a RETURNING-client update check (an installed
PWA that saw the previous version), not just server-side content checks — this rides the D2
screenshot verification standing rule.

## T-21e — D2 screenshot verification received (2026-07-24)

**Recorded.** Operator provided 2/2 prod tab screenshots (Safari, post-SW-fix — also live
confirmation of the T-21d fix propagating to a returning client). **Trends:** live data —
Confirmed COGS $674.54 (excl. tax), attributed $666.54, unlinked $0.00, awaiting-review
$8.00 / 1 receipt, unitemized remainder −$0.39, weekly chart + group legend rendering. The
signed D2 "Ungrouped" pseudo-group carries $619.38 of $666.54 — the rule works; the data
wants group assignment in Setup (operational note, not a defect). **Cost:** the signed
accept-sparse empty state ("No sales data yet · Food cost appears here once sales sync") —
exactly the OKR-session decision "Cost-in-prod = accept sparse". With `task version` parity
(0.2.2/1.1.0 = `32afb39`) this settles **D2 → MET** for the close. Still open: the armed
two-device convergence check (a separate flag), P4 and E2 rulings.

## T-21f — Final close-out rulings (2026-07-24)

**Decision 47 — P4 graded MET on the cohort reading (operator ruling).** The KR's auditable
cohort is the 12 items open at cycle start (2026-07-19): 12/12 routed through the PRD
§Routing doors. The 15 `· new` markers now in BACKLOG.md are all mid-cycle accretions (plus
today's play-test findings) — they are the NEXT planning round's feedstock, routed at
`/nc-roadmap-round`, not this cycle's debt. Chosen over the literal-grep reading, which would
perversely punish the cycle for capturing findings instead of dropping them.

**Decision 48 — E2's 0%-food-cost anomaly: BACKLOGGED with a named lead (delegated to
Claude, decided here).** Disposition rationale: it is an open investigation note, not a
correctness red — F2's margin math is red-first-proven against a hand-computed fixture, and
prod's Cost tab has no sales data yet, so nothing live depends on the cell today. Lead
hypothesis recorded on the backlog entry: a menu item with sales but zero recipe allocation
produces ingredient_cost_total = 0 → 0% food cost — likely correct-but-misleading display
(wants an "unallocated" marker distinct from a genuine 0%) rather than a computation bug.
Investigate when prod sales sync lands and the Cost tab carries real rows.

## T-22 — Morning-triage resolutions (2026-07-25, overnight-20260725)

**Gate evidence here is an adversarial reproduction pass, not the run's own closeout lines.** A
card's own closeout is not evidence about that card, so a fresh reviewer — briefed to falsify the
closeout's claims rather than confirm them — rebuilt in its own worktrees and re-executed: `go
build ./...` / `go vet ./...` exit 0; `go test ./... -count=1 -p 1` green across **9 named
packages** (`workflow` verbose = 32 RUN / 32 PASS, so not a silently-empty package); the two
implicated E2E specs re-run on **both** trees — **2 failed** on `overnight-20260725`, **2 passed**
on `dev` @ `d37fb10` — confirming F1's attribution independently; W2's master-wins finding
reproduced a **fourth** time; W1's four RLS discrimination proofs reproduced against the live
stack, including the `service_role` BYPASSRLS control. Blast-radius attestation independently
confirmed: all five HARD files absent from the full diff, and **every shipped artifact hashed
byte-identical between `c14cbce` and `HEAD`** — which validates the run's judgement call not to
re-run the full suite after merges 2 and 3.

**Conflict-log audit passed.** 3 merges, 3 entries; all three merge-intents committed as their
card's first commit, each carrying the three durable fields. Resolutions verified against the
diffs, not the prose: W1's runbook half 1 byte-identical above the seam (704 lines), `timings.log`
union lossless in both directions, and the `DECISIONS-NEEDED.md` add/add concatenation dropping
only W2's own merge-note whose instruction had been discharged. Merge 2's handling of W1's false
"pre-existing" inference — W1's measurements kept verbatim, the correction appended and attributed
to the orchestrator — is the correct resolution and reads correctly.

Merged `--no-ff` to `dev` knowingly red on two E2E specs. `dev` is not the deploy source (releases
promote a tag to `main`), so the regression does not reach prod.

**Decision 49 — FORK 1: add the client half (option a).** Chosen over (b) reverting the
normalization and (c) mapping at the API boundary. The decisive argument is one the fork document
did not carry: Activity 1 ends in `sync-hard-cutover`, where RxDB replicates rows straight from
Postgres and **there is no API boundary left to translate at** — so (c) is a translation layer with
a known expiry date, and the client must learn the database's own vocabulary either way. (b) was
rejected because it abandons the card's premise while keeping the wrong copy forever; note it would
have been a *partial* revert, since `d1674d3` carries the `pendingApprovals` snapshot gate in the
same commit and that half closes the real leak on its own. **Repriced: the surface is at least
seven call sites, not the four the fork document named** — `workflows.html:2065/2066/2067` (the
three list-card badges) and `:2720` (the optimistic sibling of `:2717`) were missed. `0b53d46` on
`card/f1-workflow-submission-status-default` is **kept** and becomes this card's test.

**Decision 50 — FORK 3: field-level three-way merge, with same-field clashes falling back to
master-wins plus a `conflict$` notice.** This **overturns the 2026-07-24 explore session's signed
choice of last-write-wins**, on evidence that the signed choice never existed in the first place:
RxDB's default is unconditional master-wins, no clock participates, and the strictly-later local
write is discarded. Chosen over accept-as-is (silent loss; the close record would attribute a crew
member's work to whoever edited from the office, which is the opposite of the product's stated core
value), over master-wins-plus-notice (honest but still discards ~20 minutes of work on every
collision), and over genuine LWW (symmetric loss — it drops the manager's correction instead — and
makes each phone's clock the tiebreaker, reintroducing skew risk the server's trigger-stamped
`_modified` currently avoids). The deciding fact, verified at triage and **not known to the fork
document**: `assumedMasterState` is present in `RxConflictHandlerInput`
(`rxdb/dist/types/types/conflict-handling.d.ts:10`), so a true three-way merge is tractable —
diffing fork-vs-assumed and master-vs-assumed identifies exactly who changed which field. It is
declared optional in the type, so the rule needs a defined fallback when it is absent. This is the
real work of `sync-rxdb-schema-and-replication` and it can now be sized.

**Decision 51 — FORK 4: stay gateway-less, with a permanent client-construction helper in HQ.**
Chosen over running Kong. Kong would cost a container plus route config plus securing it — as the
front door — purely so a client library's constructor need not be told two URLs, and it would
reverse the one simplification the spike bought (W1 proved Kong/Studio/GoTrue unnecessary). **Rider
attached at triage:** pin `@supabase/supabase-js` and add a smoke test that fails loudly on
upgrade, because the coupling is not to the public extension points (`global.fetch`,
`realtime.transport`) but to the assumption about how the library derives `<baseUrl>/rest/v1`.
Owner: `sync-jwt-bridge-endpoint`.

**Decision 52 — FORK 2 disposed: `backlog-round.html` is not a run artifact.** Triage-decidable, so
decided here rather than asked. The file is a static `<title>Backlog</title>` viewer page, and
BACKLOG.md entry 63 cites a `/nc-roadmap-round` session on 2026-07-25 — it is the operator's own
roadmap-round render, produced by a concurrent session, never tracked in any branch. Left untracked
and undeleted; it is disposable and outside every card's footprint. The run was right to leave it
exactly as found.

**Decision 53 — W1's runbook carries fabricated *presentation*, and must be repaired before it is
treated as a reproducible artifact.** Engineer-decidable, decided here. The adversarial pass
refuted the runbook's own twice-stated integrity claim (`README.md:32-34`, `:724-727` — *"the
output shown under it is the real captured output, not a reconstruction"*) in **six** blocks, of
which G6 had caught one: ten `HTTP nnn` annotations right-column-aligned onto `curl` invocations
carrying no `-w`/`-i`/`-D -`; every `rtwatch` RECV line stripped of the unconditional `topic=`
column that `rtwatch/main.go:144` prints, with `phx_reply` padded two different ways in the same
document; three `DROP POLICY` tags silently dropped from a psql block that kept `DROP TRIGGER`; a
`... (5 more alice rows) ...` elision implying 6 where the DB held 8; and `timings.log` recording
"P1–P11" against a runbook documenting only P1–P10. **The facts survive** — all ten HTTP values
re-run correct, the printed JWT is cryptographically genuine, every quoted timestamp matches live
rows to the microsecond — so **no verdict changes and W1's GO stands**. But the document is the
artifact the operator was promised they could run by hand, and its integrity claim is currently
false. Repair rides the next card that touches the runbook.

**Decision 54 — the seam map is wrong for `backend/internal/workflow`, and the fix rides the F1
client-half card rather than being edited at triage.** Engineer-decidable, decided here. F1 was
slated seam-confined, paid the `workflows|persistence` subset, went green at 102 passed — and was
wrong anyway, because **neither failing spec is in that subset**. `night-crew.toml:50-51` must
extend `backend/internal/workflow` and `workflows.html` to `["workflows", "persistence", "sync",
"repro-cut-task"]`; as a Playwright path regex that expansion selects exactly the four intended
specs and nothing else (verified against `tests/`). Not edited during triage because the F1
client-half card touches both those paths and needs the de-confined suite to prove itself — the
config change belongs in the card that first depends on it. **Stated cost, so it is chosen rather
than inherited:** this pulls `sync.spec.js` and its known ~16–20 % `:1198` flake into every future
workflow card's gate.

**Recorded, not decided — the coverage finding under the "no third red" result.** The suite's
apparent narrow blast radius is thin coverage, not containment: a programmatic sweep found
**exactly two tests in the whole suite** that create a `requires_approval:false` template, submit
it, and assert on the rendered result — and both are the two that went red. The three green
`GATE-01/03/06` tests pass *vacuously* with respect to this defect; they assert submission is
blocked, never that it rendered. The F1 client-half card must add a test that asserts the
no-approval submitted state, or the repair ships with the same blind spot that let the regression
through.

**Also recorded: `conflict$` fires per document, not per replication.** W2's caveat
(`README.md:1166`) is wrong in the conservative direction — `upstream.js:333` emits inside the
`Object.entries(conflictsById).map(...)`, and the live probe's payload carried `output.id`. The
error understates the signal, and Decision 50 was priced on the corrected reading.

**Preference coverage:** `night-crew decisions audit --repo . --run 20260725` reports *"no gray
areas routed through the resolver yet"* — coverage is **undefined for this run, not low**. The run
parked its gray areas straight into `DECISIONS-NEEDED.md` without routing them through the
resolver, so the audit has nothing to measure. That is the gap to close before the number means
anything.

## T-23 — Morning-triage resolutions (2026-07-26, overnight-20260726)

Twelve forks (D-1 … D-12) walked. **One was genuinely operator-level** (D-5); one more turned on a
convention the operator had already recorded (D-6). The remaining ten were engineering calls
escalated to the operator and were decided at triage under the standing rule *decide mechanism
yourself, bring product and intent*. Two forks were materially reframed by reading the source they
cite — both reframes are recorded below because the fork text, left as written, would mislead the
next reader.

**Decision 55 — D-5: port the existing `ResolveEntityAccess` predicate; do not invent a permission
model in the cutover.** The fork frames this as an open product question and instructs that "the
cutover card must not invent an answer either." **That framing is wrong in a way that matters: HQ
already answers it in shipped code.** `backend/internal/sync/ops.go:474` — `ResolveEntityAccess` —
is the predicate the *current* WebSocket sync layer enforces for exactly this question: template
assignees (`assignee_type='user'` by UUID, or `='role'` matched against `users.roles`) ∪ all
admins/superadmins, with the op author unioned in by `listener.go:63-72`, covered by a 12-combo
Cartesian test. So the cutover does not need to invent an answer — it needs to **port** one.
Chosen over (b) porting *and* tightening `assignment_role` in the same card, and (c) narrowing
phase 1 to owner-only. (b) was rejected because it varies substrate and permission semantics
simultaneously — every convergence red then becomes a 3am judgement call about whether it is a bug
or the intended tightening. (c) was rejected because it silently de-live-syncs the Approvals tab,
which is the product's stated core value. Mechanism: project `template_assignments ⋈ users` into
the sync DB as a row-visibility table, the way `hq_grant_projection` already projects grants, and
express RLS as an `EXISTS` against it. **Two properties of the ported predicate are inherited
knowingly, not by accident:** the resolver never filters on `assignment_role`, so an `'approver'`
assignment grants identical visibility to an `'assignee'` one; and the `roles && ARRAY['admin',
'superadmin']` arm is unconditional, so every admin sees every template's ops. Both are live today.
The test asserting approver inclusion is named `..._CurrentBehavior` — its author was documenting,
not endorsing. **D-5 ceases to be a fork and becomes a spec line on
`sync-rxdb-schema-and-replication`.** The fork text must be updated to cite `ops.go:474`, since as
written it tells the next reader the answer does not exist.

**Decision 56 — D-6: expand umbrella slugs at mint time. Closed by existing convention, not
re-decided.** The fork routes this to the operator as "a product/UX decision about what a launcher
should show." It is not open: the recorded go-forward convention is *grants are per-tab/per-feature,
not bundled per-app*, and umbrella slugs are per-app bundling. A user holding `inventory` reaches
`inventory-trends` and `inventory-cost` (`main.go:628, 642, 652`), so a launcher built from the
narrow claim hides two surfaces the user is entitled to. Decided by precedent under the recorded
convention; **not yet confirmed by the operator** — if umbrella slugs are a deliberate exception to
per-tab granularity, this decision reverses. The fork's objection — the token then asserts more than a single
`app_permissions` row does — is immaterial, because the claim is advisory and the live
`hq_grant_projection` is the gate. Chosen over shipping the narrow list (pushes the umbrella table
into every client) and over emitting both fields (two sources of truth for one question).

**Decision 57 — D-7: split. Fix the live disclosure now; hand the cache-key design forward.** The
fork's claim verified exactly — `build-sw.js:60-78` configures one `NetworkFirst` route on
`/\/api\//` with **no `Vary`, no `cacheKeyWillBeUsed`, no `matchOptions`**; the key is the URL and
`Authorization` is not in it. **Three things the fork does not say, found by reading the source:**
(1) **nothing ever clears the cache** — `logout()` at `index.html:141-145` POSTs to
`/api/v1/auth/logout` and redirects, and there is no `caches.delete` anywhere in app code, so the
"re-log-in as a different user" path is wide open; (2) the route matches **every endpoint shipped
today**, not just the future replication URL, making this a live disclosure bug on crew phones
rather than only a forward hazard; (3) it composes with `checkAuth()`, whose offline branch
deliberately does not redirect — `/api/v1/me` is on the same route, so on a dead-LTE shared phone
user B is served user A's cached identity and `renderUserHeader` paints A's name. Severity is
bounded by `NetworkFirst` serving cache only on failure or >10s timeout, which on a food truck is
routine. Chosen over handing it all forward as the fork proposes (leaves a cross-tenant read on
phones for the life of the sync card) and over deleting the route outright now (correct end state,
but costs offline API reads today). Immediate half: `caches.delete('api-cache')` on logout, and
`checkAuth`'s offline branch fails closed on identity. Structural half — cache key / `Vary` — goes
to `sync-rxdb-schema-and-replication`, which will most likely **retire the route entirely**, since
once RxDB replicates, offline data comes from IndexedDB and `api-cache` is obsolete.

**Decision 58 — D-12: `build-sw.js` globs the tracked set (`git ls-files`), not the working tree.**
Chosen over documenting the foot-gun in CLAUDE.md. A Workbox precache entry that 404s fails the
**entire** service-worker install, so the symptom is "the PWA stops updating" with an invisible
cause; documenting a trap that silently bricks updates on every phone is not a mitigation. Note the
trigger is not hypothetical — `backlog-round.html` is the same untracked file disposed as FORK 2 in
T-22, still sitting in the repo root, and `task sw` runs automatically as a dependency of both
`task test` and `task prod:deploy`. Slated with Decision 57's logout fix as one small hygiene card.

**Decision 59 — D-8: exclude the vendored bundle from the precache until a page imports it.** The
fork marks this "Operator's call"; decided at triage as an engineering call with the operator
informed. +495 KiB over LTE on every crew phone (+34%, 25.4% of the whole precache) for an asset no
page imports is a cost with no current benefit. Re-adding the `globPatterns` entry is one line, and
the offline-availability proof matters most on the card that actually adopts the bundle — which is
also the card where a failure would be actionable.

**Decision 60 — D-2: fix the offline double-submit client-side, as its own card.** Real,
pre-existing, untouched by card A: `workflows.html:1656` mints a fresh `idempotency_key` per call
and `:2778` returns to the list without pushing to `MY_SUBMISSIONS`, so offline submit → reopen →
submit again writes two rows past the `idempotency_key UUID UNIQUE` guard. Fix by reusing the
enqueued key on re-submit rather than minting a new one. **Deliberately not the server-side guard**
— that reopens Decision 49 and trips card A's park trigger for no additional benefit.

**Decision 61 — D-4: `hq_grant_projection` is written by push on grant change**, in the same
transaction as the `app_permissions` mutation. Chosen over periodic reconcile and `postgres_fdw`.
The projection exists precisely because a revocation must take effect *now* — a reconcile
reintroduces the exact replay window the design was built to eliminate, and `fdw` couples the two
databases in a way the cutover card has not settled. Recorded as a contract on the cutover card
rather than left as an open plumbing choice.

**Decision 62 — D-9: not a fork but an obligation on `sync-rxdb-schema-and-replication`.** The card
must declare its origin shape as its first spec line, and **cost the reverse proxy** if
same-origin. Verdict items 1 and 7 hold only in a same-origin-fronted shape that HQ does not have;
`browser/serve.mjs` invents it for the harness, and the real equivalent is unbuilt and uncosted.
Cross-origin moots item 7 and inverts item 1 (W2's `global.fetch` shim returns).

**Decision 63 — D-1: humanize the History status token** via a small status→label map. The card's
own framing — "teach the client the DB's vocabulary" — covers the eighth call site at
`workflows.html:2503`. Follow-up, not a card.

**Decision 64 — D-10: strike the quoted millisecond figures.** 47/65/87 ms are `Date.now()` deltas
on a clock-stepping host (G6's own run printed **−1545 ms** for the same measurement). Replace with
"sub-second" and switch `leg4-leader-election.spec.js` to `performance.now()`. The qualitative
findings — one leader, follower silent, survivor replicates — are ordering facts and stand.

**Decision 65 — D-11: accept the as-built `vendor/` layout**; the fork itself judges it better than
the merge-intent it deviates from, and it is what holds the root-package line. The output here is
the process note, not a choice: **a merge-intent that silently diverges from what lands is worth
less at the next merge**, and the deviation should have been disclosed at the time.

**Decision 66 — D-3: fold the stale `tests/sync.spec.js:1584` comment into the next card touching
that file.** One line of doc rot; no card tonight owns the file.

**Process finding — the run over-escalates.** Ten of twelve forks were mechanism decided in minutes
at triage, and each one cost the operator attention on a morning already carrying a real product
fork. The bar to add to the run prompt: *a fork goes to the operator only if it is a product, cost,
or intent question; mechanism gets decided and stated.* The two forks that genuinely needed a human
(D-5, D-6) were both improved by reading the source they cite — which is also the check the run
itself should have run before escalating.

**Ritual defect — `/nc-morning-triage` points at the wrong file.** The skill instructs recording
resolutions in `DESIGN.md §15x`. In the night-crew clone §15x is *"Docker-only local infra for
target repos (2026-07-13)"*; the actual convention is this ledger's `## T-NN` entries with numbered
decisions. Twelve resolutions were nearly appended into a Docker infra section. The skill's
reference needs correcting.

---

## T-24 — Cross-user checklist hydration divergence: product ruling (2026-07-26)

**Operator-decided, not agent-closed.** Raised as a `BACKLOG.md` item ("Cross-user checklist
hydration divergence (approved-vs-rejected ghost state)") whose own text said *"needs a product
ruling first."* The ruling was given directly by the operator on 2026-07-26 in response to the
question being put plainly; no agent inferred it from a convention, and no PM session was run.

**The reproduction (unchanged, and it is not a bug hunt — it was already isolated).** Two users
view the SAME checklist. A's submission was approved; B's was rejected. On a new cycle, B's
rejected submission **resurrects as B's current 2/2 state**, while A sees a fresh 0/2 whose clicks
**silently no-op** (no POST, no toggle, no feedback). Reproduced headlessly in fresh contexts.
Deterministic `MY_SUBMISSIONS`-driven hydration logic — **not** network, cache, gating, or the
07-22 `sync.js` change; server state verified byte-identical for both users. The E2E convergence
matrix misses the cell: it never seeds an asymmetric approved-for-A / rejected-for-B history
before reopening.

**67. A new cycle starts fresh for every user — 0/2 for both A and B.**

- B's rejected submission **does NOT resurrect as current state.** It is archived and remains
  visible as **history**; the rejection and its fail notes are a record, not a live draft.
- A's fresh 0/2 **must accept clicks.** The silent no-op is a bug, not intended behavior — it is
  part of what this ruling requires fixed, not a separate concern.
- **Rationale: rejection means redo.** That matches the accountability model the workflow engine
  exists for. A rejected checklist that carries forward as 2/2 lets unreviewed work look complete,
  which is precisely the failure the engine is supposed to prevent.

**Consequence for the roadmap.** `sync-hard-cutover` was double-blocked: on
`sync-rxdb-schema-and-replication` landing, and on this item being routed to a product decision
(Product KR2). **The product-disposition half is now cleared.** The card remains blocked only on
Card C landing.

**Implementation note for whoever builds it** — this ruling defines the target state, not the
work. It still needs the red-first cell the backlog item calls for: seed an asymmetric
approved-for-A / rejected-for-B history, reopen as both users, assert 0/2 for both **and** assert
that A's clicks POST. The silent no-op needs its own assertion; a test that only checks the
counter would pass against a dead UI.

**Process note.** The rest of the backlog was deliberately NOT swept here. No Operator Brief was
written and `/nc-pm-session` was not run — the ceremony is heavier than the single question, and
the backlog's normal destiny is the milestone-boundary planning round (DESIGN §15k), not an
evening ruling. One item was pulled forward because it blocked a card; the others were left where
they belong.

---

## T-25 — Morning-triage resolutions (2026-07-27, overnight-20260727)

Nine forks (D-1 … D-9) walked, plus eight findings the run did not report, produced by an
adversarial subagent that re-executed the gates in its own worktree and probed the closeout's
claims by mutation rather than reading. **Three operator-level calls** (D-1, D-6 go/no-go, D-6
origin shape) and **one operator routing call** (D-2). The rest were engineering and PjM calls
decided at triage under the standing rule *decide mechanism yourself, bring product and intent* —
including D-4/D-5, which was wrongly escalated and returned with "PjM decides."

**The night's headline is that the record, not the tree, was what needed repair.** Build, vet and
Go tests are green on the merged tree; the single Playwright red is a known load-sensitive flake.
But six durable claims were refuted by execution, one of them the very precache figure the night's
own review caught and corrected everywhere except where it is most read.

**Decision 67 — D-1: `build-sw.js` globs `git ls-tree -r --name-only HEAD`, not `git ls-files`.
This AMENDS decision 58's literal text.** Decision 58 said "the tracked set (`git ls-files`)" and
that is exactly what shipped, faithfully. But `git ls-files` reads the **index**, so a
staged-but-uncommitted file enters the precache manifest. Reproduced end-to-end at triage:
`git add zz-adv27-staged.html && node build-sw.js` → `23 files precached`, the file present in
`sw.js`, and `git ls-tree -r --name-only HEAD` excludes it. **The trigger path is complete, which
the fork did not establish:** `task prod:deploy` (`Taskfile.yml:174-210`) does **not** run `task sw`
on the box — it `git reset --hard origin/main` then `docker compose build` — so the *committed*
`sw.js` is what ships, and a URL that 404s fails the **entire** service-worker install for every
returning client. That is precisely the failure decision 58 exists to prevent, so the amendment
serves the decision's intent against its own letter. No test catches it either: `tests/sw-manifest.spec.js`
test 1 uses the same `git ls-files`. Chosen over (b) keeping `ls-files` behind a fail-loud preflight
— rejected because it fails the build on a harmless scratch file and creates a second mechanism to
keep true instead of fixing the first — and over (c) documenting the hazard, which leaves it live
and undetected. **The fix is a card, not a triage edit** (triage lands `docs:` commits only): it
carries the one-line change plus a regression test that stages a file and asserts its absence from
the manifest. **Aside, pre-existing:** CLAUDE.md claims `task prod:deploy` runs `task sw`; the
Taskfile does not. Stale doc, recorded as B-13.

**Decision 68 — D-6: the Card C no-go is ENDORSED.** `sync-rxdb-schema-and-replication` was not
opened at 02:40 under the slate's own budget-discipline clause. The evidence supports the call
rather than merely excusing it: Card A ran ~1 h 45 m against a 30–50 m estimate and Card B ~2 h 50 m
against 45–90 m — both **~2–2.5× once review and repair were counted** — while C is priced
120–240 m *before* review, is first-of-kind, and carries five binding obligations and three park
triggers. Starting it would have produced an unmerged worktree and a half-designed conflict handler,
which is strictly worse than not starting; C is last in the slate by design precisely to keep this
call available. Chosen over overturning (the stop-cleanly clause was read correctly, not
conservatively) and over splitting C now — the `conflictHandler` is the card's real work per
decision 50 and is the piece that resists splitting.

**Decision 69 — D-6/obligation 2: the origin shape is SAME-ORIGIN, proxied by the Go backend.**
Card C's first spec line. A `/sync/*` `httputil.ReverseProxy` handler in the existing backend fronts
`rest:3000` and `realtime:4000`. Costed: one handler plus its tests; Cloudflare Tunnel config
unchanged; no second hostname. Chosen over routing at the tunnel — which would split routing across
two places and, decisively, put replication traffic where the backend cannot see or authorize it,
while **obligation 1 is a row-visibility predicate the backend must enforce** — and over
cross-origin `sync.yumyums.kitchen`, which buys independent scaling at the price of a CORS policy
that becomes a security surface, a second origin for the service worker and cache logic to reason
about, and cross-origin credential handling for the JWT bridge. Adding a second origin immediately
after Card A closed a cross-tenant cache disclosure is the wrong direction. **This disarms C's
likeliest park trigger before dispatch**, which is why it was settled attended rather than left to
fire at 01:00.

**Decision 70 — D-2: both remaining `api-cache`-shaped disclosures are owned by Card C.** (1)
`localStorage['hq_apps']` is never cleared on logout — confirmed by reading, `index.html:224` still
parses the previous user's cached slug list in the fail-closed branch, so offline on a shared truck
phone user B sees user A's tiles. (2) An identity change *without* a logout: B logs in while A's
session is live, `logout()` never runs, and `login.html` performs no cache hygiene of its own. Both
are UI-only — server-side grants remain the real gate. Routed to C because C already owns
obligation 4 (decision 57's deferred cache-key half) and is expected to retire `api-cache` entirely,
so one card owns one problem. Chosen over a small dedicated card — which would close the leak sooner
and was a genuine contender — and over folding into grant enforcement, which reasons about
server-side data boundaries while this is client cache hygiene. **Accepted cost: if C slips again,
these slip with it.**

**Decision 71 — D-4 + D-5: ONE follow-up card, not two, not a backlog entry. (PjM call, taken at
triage.)** Both are consequences of the key reuse decision 60 authorized, and both need
`backend/internal/workflow` — Card B's explicit park trigger — so splitting them means opening that
package twice. D-4 measured at triage, not reasoned: same payload POSTed twice with one
`idempotency_key` → `201`/`201` with an identical submission id, `submission_rows=1 response_rows=1
fail_note_rows=2`; `submission_fail_notes` has no unique constraint and a bare INSERT
(`repository.go:760-767`) while the responses insert directly above it carries
`ON CONFLICT (submission_id, field_id) DO UPDATE`. D-5 confirmed by reading:
`findQueuedSubmission` filters on `template_id` only, queue entries carry no period, and nothing
ages them out. The card: add the `ON CONFLICT` plus a unique index, bound `findQueuedSubmission` to
the current period, age out stale `submitQueue` entries. **It also repairs decision 77's third
falsehood by making the claim true rather than by editing a comment** — see below.

**Decision 72 — D-7: the orchestrator writes `runs/<date>/timings-orchestrator.log`; the closeout
concatenates.** Both merges collided on `timings.log` — merge 1 as an untracked-file overwrite,
merge 2 as a content conflict — and the `ORCH ` prefix added after merge 1 kept the two line
families distinguishable without stopping them landing on the same offset in an append-only file.
Both were resolved by union with nothing discarded, and I re-verified the result (11 `A`, 8
`ORCH A`, 12 `B`, 6 `ORCH B`, zero markers). Separate files remove a *guaranteed* conflict from
every future run for near-zero cost. Chosen over keeping the prefix convention, which has now failed
twice.

**Decision 73 — D-9: add the `bdd:gen` dep to `task test:`. The fork's own remedy menu was wider
than the problem.** D-9 reproduces exactly — a fresh worktree lists `Total: 559 tests in 19 files`,
the main repo `560 in 20` — so every card worktree really did run 19 of 20 spec files while
reporting a full suite in good faith. But running `npx bddgen` in the worktree regenerated
`.features-gen/features/user-invite-onboarding.feature.spec.js` **byte-identical** to the main
repo's copy and the count went to 560/20, and **`task bdd:gen` already exists** with `task bdd` and
`task test:all` already depending on it. The root cause is narrower than "add a generation step":
`task test:` (`Taskfile.yml:28-30`) is the one target that does not. One line. Chosen over tracking
`.features-gen/` (commits a generated artifact that will drift) and over doing nothing. **Option 3
from the fork — fail loudly on an empty project — is also adopted** as a cheap independent guard,
because the failure mode that hid this was silence: a project contributing zero tests produced no
error, no skip line, and nothing in the reporter output.

**Decision 74 — D-3: recorded as a bound on what shipped, no action.** `identityVerified` is set by
**any** 200 and the client cannot tell whose 200 it is, so "fail closed" means closed-on-*failure*,
never closed-on-wrong-*identity*. Independently confirmed at triage: deleting the
`removeUserHeader()` call leaves all nine `index.spec.js` tests passing, and a call-site census
(one `renderUserHeader` site in the `res.ok` arm, one `checkAuth()` at parse time, no
`pageshow`/`visibilitychange`) shows it can never fire on today's single render path. **The narrowed
wording Card A's repair put into `index.html` and merge-intent A is accurate and does not
overstate** — the repair narrowed the claim rather than defending it, which is the correct response
to a refuted claim. Flagged only so nobody reads the roadmap card as "cross-tenant identity is
solved." It is bounded, not solved.

**Decision 75 — the morning-triage G4 discipline greps are VACUOUS in this repo and must not be
reported as clean.** The ritual's greps target `internal/journal` and `internal/workorder`; hq's Go
tree is `backend/{cmd,internal}` and **neither package exists**. The greps return empty because
there is nothing to find, not because discipline held. Any run or triage reporting them "clean" is
reporting a vacuum — the same silent-green failure class as D-9, one layer up in the tooling. This
binds the night-crew clone, not hq, and is carried there as a backlog item rather than fixed here.

**Decision 76 — the durable record is corrected at triage; the one correction that needs code is
deferred to decision 71's card.** Six claims were refuted by execution. Four are documentation and
are fixed in this commit: merge-intent B's "`sync.js` was NOT edited" and "No change to `sync.js` …
not `drainQueue`" (the diff shows `sync.js +20/−1` changing exactly those); its mislabelled
"unchanged from Card A's: 1457.7 KB" (Card A's figure is 1455.6; 1457.7 is a mid-branch commit);
the roadmap card's **synthetic** precache figure `23 files / 1949.7 KB` (unattainable at any commit
— it is 1454.7 + the 495 KiB bundle, back-computed; the real `dev` value is 1947.1, and this is the
very figure G6 caught, corrected in the HANDOFF and left standing in the roadmap); and the HANDOFF's
"23 commits, every commit carries a `Night-Crew-Card:` trailer" (27 commits, 3 without — true when
written, never updated by the four commits that followed it). **The fifth is code and is NOT edited
here:** `workflows.html:1694`, merge-intent B and the roadmap card all assert "the server upserts
only the fields present in each payload," which is true for `submission_responses` and false for
`submission_fail_notes`. Editing that comment would move `sw.js` (Workbox carries a per-entry
content revision hash) and oblige a full suite re-run for a comment — the run learned this exact
lesson at its own post-merge anchor fix. Decision 71's card makes the sentence true instead.
**The sixth** — "without `-p 1` that package reds" — understates: `internal/sync` reds too
(deadlock 40P01 plus a recipient-set mismatch), confirmed by running it both ways.

**Decision 77 — LST-17 is restored to the standing-flags table; it was dropped, not fixed.** The
adversarial full-suite leg red on `tests/sync.spec.js:446 [LST-17]` at `--retries=0` (expected
`"0/1"`, received `"Uncheck Sync1 section · 1/1 items"`) under load average 3.92 from its own
concurrent leg, then went **10/10 green** under `-g "LST-17" --repeat-each=5` in isolation. That is
the signature decision 44 already named — *"LST-17 REMAINS flagged load-sensitive"* — so this is a
pre-existing flake, **not** a regression, and the run's gate verdict stands. What is real is that
the HANDOFF's standing-flags table carries `sync.spec.js:1198` and `purchasing.spec.js:1407` and
**dropped LST-17**, so a known flake went uncarried into a night that touched the queue semantics of
the very file it lives in. Per the recorded rule, an unreproducible flake may be closed but "rare,
mechanism known" must never be laundered into "not flaky" — LST-17 is the latter and stays armed.

**Process note — preference coverage is 0%, and not because coverage lapsed.**
`night-crew decisions audit --repo . --run 20260727` reports *"No gray areas routed through the
resolver yet."* Nothing is routing through the resolver at all, so the number measures adoption,
not preference quality. Reporting it as a coverage percentage would be the third silent-green in
this triage. No offer-back shortlist can be derived from it until runs actually route their gray
areas through the resolver.

---

## Attended session 2026-07-28 — conflict-notice sign-off and the schema card's dissolution

Not a morning triage: an attended working session between runs. Four decisions, one of them the
operator's own and three taken as code-internal calls and recorded rather than escalated.

**Decision 78 — `_modified` is NOT declared.** The dissolved `sync-rxdb-schema-and-replication`
demanded this be *"decided, not let be decided by whether someone copied the field in"*, and it is
now decided: leave it out, keeping it a pure pull cursor. Declaring it makes
`addDocEqualityToQuery` include `_modified` in the compare-and-swap, so **any** server-side touch
becomes a conflict — including ones where no answer changed (W2 sharp edge 11). Those land in the
conflict-notice UI as the *"a change we couldn't identify"* row, which is the one row in the whole
design from which **nothing can be recovered** — no Restore, only Open checklist and Dismiss. The
UI-SPEC says as much in advance: *"if it is declared, that row stops being rare."* Set against
that, the benefit is thin: the tightened conflict detection is doing work the field-level
three-way merge already does deliberately and with better information. Declared on
`sync-rxdb-collections-and-table-contract`, read by
`sync-rxdb-replication-and-conflict-handler`. **Revisit only if** the merge rule proves unable to
distinguish a real same-field clash from a stale fork without it.

**Decision 79 — replicated rows CARRY who-and-when.** UI-SPEC §"Explicitly NOT decided here"
flagged this as the schema declaration that makes the conflict sheet's attribution line real or
fictional — without it *"Dana M., 6:12 PM"* on the *Now shows* row degrades to *"someone else"*.
Carry them. The product's stated core value is **accountability — who checked what**; the signed
mockup draws attribution; the cost is two columns on collections that are being defined from
scratch this cycle, which is the cheapest this decision will ever be. Owned by
`sync-rxdb-collections-and-table-contract`.

**Decision 80 — the mockup sign-off EXISTS, and its scope is the artifact.** The operator answered
the question outstanding since 2026-07-29 with a verbatim *"Ok, build this."*
`sync-rxdb-conflict-notice-ui` is **no longer ATTENDED-BLOCKED** and may enter a slate. The yes was
given with the two rejectable design decisions in view and **neither was rejected**, so both are
settled and a run implements them as drawn: **the counting rule** (the banner reports how many
answers were overwritten in the retention window, not how many are still unhandled — nothing a
crew member does to a row changes a count, and a count drops only when a record *leaves* the
sheet), and **handled rows staying on the sheet** (restored and kept-theirs rows collapse to a
confirmation and keep an Undo, because a removed row cannot be undone). The 30-day retention
window in the empty state was accepted as drawn. **The yes is scoped to the committed artifact** —
`mockup.html` + `UI-SPEC.md` and their 22 renders as of the repair round — and is explicitly **not**
authority over the items UI-SPEC §"Explicitly NOT decided here" names; two of those are settled
above by 78 and 79, and the durable conflict record's home stays the UI card's own call.
**One question remains open and is deliberately non-blocking:** the sheet has no cap or date filter
beyond ~10 conflict groups. If it is still unanswered when the card runs, the run implements no cap
and says so rather than inventing one.

> **🛑 SUPERSEDED IN PART at morning triage 2026-07-28 — see T-26 decision 82.** Decision 80
> stands as the record of what was decided at 18:12 and is not withdrawn. But the mockup was walked
> plate-by-plate at triage and the operator directed two amendments (A-1, A-2), so *"neither was
> rejected … a run implements them as drawn"* no longer holds for the banner, and the 30-day
> retention accepted here is reopened. `sync-rxdb-conflict-notice-ui` returns to ATTENDED-BLOCKED.
> The rest of decision 80 — that the yes was scoped to the committed artifact, and that it is not
> authority over UI-SPEC's "Explicitly NOT decided here" list — is untouched.

**Decision 81 — `sync-rxdb-schema-and-replication` is DISSOLVED into four cards.** It had already
been fanned out twice (browser delivery 2026-07-26, `sync-proxy-endpoint` 2026-07-28) and was still
carrying four independent mechanisms under eight obligations — collections and the SQL table
contract, the RLS row-visibility port, the replication wiring and `conflictHandler`, and a group of
cache/identity hygiene items. The §1 split rule applies and the reason is the standing one: an
unattended run must never discover mid-night that a card is four cards. Now
`sync-rxdb-collections-and-table-contract` (wave 0, foundation) → `sync-rxdb-row-visibility-rls`
and `sync-rxdb-replication-and-conflict-handler` (parallel-safe, disjoint footprints — SQL/backend
against frontend/client) → `sync-cache-and-identity-hygiene`. **No scope was dropped**; the
original card text is retained beneath the dissolution notice as the record of why each obligation
exists. Two properties of the split are worth stating because they are gains, not bookkeeping:
the hygiene items no longer ride the cycle's largest card under an accepted *"if this card slips,
they slip with it"*, and **the `HQ_SYNC_REST_URL` activation interlock now spans two cards**, which
makes it easier to get wrong — so it is restated on `sync-rxdb-row-visibility-rls`, on
`sync-hard-cutover`, and in the dissolution notice, in addition to `proxy.go`'s env-var comment.

## T-26 — Morning-triage resolutions (2026-07-28, overnight-20260729)

Four forks (D-1 … D-4) walked, plus three findings the run did not report and two record defects,
produced by an adversarial subagent that re-executed every gate in its own clone and probed the
closeout's claims by mutation rather than reading. **The tree was clean and the gates were honest** —
all four cards landed, zero parked, and independent re-execution reproduced 563/569 Playwright,
9/9 Go packages, a byte-identical `sw.js` and a correct version mirror. As in T-25, **what needed
repair was the record, not the tree** — with one addition this cycle: a *decision* needed repair too.

**Two operator-level calls** (the timezone ruling, the sign-off supersession), **one operator
routing call** (the sync-door guard), and the rest decided at triage under the standing rule
*decide mechanism yourself, bring product and intent*.

**Decision 82 — the conflict-notice sign-off is superseded in part, and the amendments are
required.** Decision 80 recorded a verbatim *"Ok, build this"* from an attended session at 18:12,
settling the counting rule "as drawn" and accepting 30-day retention. At triage the mockup was
walked plate-by-plate and the operator asked the question the design could not answer: *"when she
finishes the second, why does it still say three?"* Rule 3's answer — *because three answers were
overwritten and that stays true* — is literally correct and wrong on a phone, because a number in a
coloured banner reads as a **badge**, and badges count outstanding work. The sheet already showed
progress (restored rows turn green); only the banner was frozen. The design had bundled two
unrelated things: *keeping rows so Undo survives* and *what number the banner prints*. **A-1**
unbundles them — rows still never leave except on Dismiss or expiry, and the banner now carries both
figures. **A-2** came from the operator's second test, *"make sure it is clear … exactly what they
were overriding"*: both values are already on screen above the action, including in the collapsed
view, but the button names only what it restores, the batch path writes three overwrites on one tap
with no confirmation, and the collapsed view drops the timestamp the expanded view carries — so the
riskiest action carried the least information. Chosen over letting the sign-off stand (the defect is
real and reaches the screen a crew member sees most) and over discarding decision 80 (the yes was
genuine when given; a ledger that erases reversals cannot be trusted about anything else). Retention
and the "recoverable" chip question are **reopened and deferred** until revised plates exist, because
both are easier to judge against a banner that is no longer misleading. The card returns to
ATTENDED-BLOCKED.

**Decision 83 — the app's timezone is `America/New_York`, everywhere, in one card, fixed forward.**
D-1 asked only whether a *submission's* "today" should be UTC or local. The operator answered with a
rule rather than an answer — **"the apps time zone should be NY time"** — and checking it against the
tree turned a four-site fix into a cross-cutting correction: the codebase is running **two
conflicting timezone regimes**. `users.DefaultTimezone` is `America/New_York` and the Users tab, the
purchasing handler/scheduler fallbacks and `playwright.config.js` agree — while `America/Chicago` is
hardcoded in `inventory/handler.go` (×6, the COGS period-summary window **and** completeness gate
feeding sales-processor's weekly payroll), `inventory/trends.go`, `purchasing/service.go`
(`CurrentWeekStart` — the Monday every purchasing week hangs off), `recipes/cost.go`,
`recipes/scheduler.go`, migrations `0037`/`0042` as column defaults, and `purchasing.html:295`, which
actively **writes** Chicago into cutoff config the backend would otherwise default to New York.
Blast radius stated precisely rather than alarmingly: the COGS date filters are
`COALESCE(event_date, created_at AT TIME ZONE 'America/Chicago')`, so only rows with no extracted
`event_date` are exposed — bounded; but `CurrentWeekStart` and the recipe cost week are
**unconditional** Chicago, so every weekly boundary is currently an hour off the operating day.
Scoped as **one card covering all sites** (piecemeal leaves two boundaries disagreeing, which is
exactly today's bug) and **fix-forward only** (past weekly COGS/payroll figures were already acted
on; restating numbers that paid people is a worse cure than the disease). The card notes the
changeover date so a future reader knows why one boundary moves once.

**Decision 84 — the `/sync/*` activation interlock stays a documented constraint, not a gate.**
Setting `HQ_SYNC_REST_URL` before row-visibility RLS lands would give every logged-in crew member
read *and* write on the whole exposed schema. Chosen over a hard refuse-to-start gate and over
stripping the env handling: nobody sets those variables except deliberately, RLS is scheduled ahead
of any client that needs the door, and a marker gate would mean inventing a "has RLS landed" signal —
speculative machinery guarding a door nobody is reaching for, plus a thing to maintain and to defeat
during legitimate testing. The constraint is already restated on `sync-rxdb-row-visibility-rls`, on
`sync-hard-cutover`, in the dissolution notice and in `proxy.go`.

**Decision 85 — the two pre-existing defects are filed to `BACKLOG.md`, not left in a card body.**
D-4 asked whether to file or fold. Filed, as B-19 and B-20: `BACKLOG.md` is what `/nc-roadmap-round`
consumes, and a defect disclosed only inside a DONE card body and a merge-intent note is a defect
that will be rediscovered rather than scheduled. Both were confirmed genuinely pre-existing against
`25fbc16` by G6 — not damage this run caused and reclassified.

**Decision 86 — the unparseable commit trailers are fixed at the emitter, not by rewriting history.**
The adversary found **14 of 33 commits** carry a `Night-Crew-Card:` trailer `git interpret-trailers`
cannot see: a blank line between it and `Co-Authored-By:` splits the trailer block. Affected are all
of Card C's commits, **all four merge commits**, and the closeout — so Card C is entirely invisible
to trailer-parsing tooling, while all four merge-intent notes assert the trailer as a landed
convention (textually true, mechanically false). Cards A and B rewrote history specifically to fix
this and Card C never did; the orchestrator's own merge commits reproduced the bug. Rewriting four
merge commits to correct a cosmetic-to-tooling defect risks the very record being preserved, for no
behavioural gain. Filed as B-21 with the fix at the emitter.

**Decision 87 — Card C's traversal claim travels as its scope statement, not its summary.** The
adversary reproduced three constructs (`..;/`, `....//`, `%252e%252e`) reaching the upstream carrying
HQ's minted bearer. `proxy.go:340-349` **documents these as deliberately out of scope** with a
measured justification, so this is a known bounded residual and not a defect — but the closeout's
flat *"path traversal rejected with 400 before any upstream connection"* reads as absolute and is
not. The scope statement is the durable claim; the summary is not. Recorded so a future reader
inheriting the summary does not assume a guarantee the code never made.

**Decision 88 — two roadmap record defects corrected.** `sync-proxy-endpoint` was recorded as
**Wave 0**; it ran in **Track B** (Wave 0 was Card A, alone, because it changed how `sw.js` is
generated). The mockup card was dated **2026-07-29**; the run merged every card on **07-28**. Both
corrected in place with the original text noted, per the standing practice that a corrected record
shows its correction.

**Process note — the G4 discipline greps remain vacuous here, and were reported as a vacuum.**
`internal/journal` and `internal/workorder` do not exist in this repo (its Go tree is
`backend/internal/*`), so the standard triage greps return empty because there is nothing to find.
The adversary was briefed to report them **N/A-VACUOUS** rather than clean. This is B-14, still open,
and it is the same silent-green class the run itself hit twice more: `task test` running 19 of 20
spec files without `bddgen` (B-09), a dropped database reading as a passing Go suite (B-16), and —
new this cycle — the orchestrator's own final-tree suite reporting **exit 0 having executed zero
tests**, because Playwright's `webServer` could not start and a `tail` pipeline masked the status.
Three instances in one run is no longer a coincidence; it is the shape of this repo's test harness.

## T-27 — Slate-planning resolution (2026-07-28, ahead of overnight-20260729-2)

Resolved in the attended slate-planning session for `overnight-20260729-2`, not at a morning
triage. Recorded here because `sync-rxdb-collections-and-table-contract` could not enter a slate
carrying an open operator decision (slate skill §1.3, fork gate).

**Decision 89 — the durable conflict record is a personal, per-device undo, stored local-only.**

The OPEN QUESTION raised at morning triage 2026-07-28 was where the record of an overwritten
answer lives. The product question put to the operator: is it an *audit trail a manager can see*,
or a *personal undo for the person holding the phone*?

**Operator answer: personal undo, per-device.** The record exists so the crew member can get their
own value back.

Mechanism decided by the planner (per `preferences` — ask about product and intent, decide the
plumbing yourself and prove it by execution):

- **Local-only.** A local RxDB collection. **No server table, no endpoint, no replication of the
  conflict record itself.** This keeps the signed mockup's contract literally true (UI-SPEC:
  *"no new sync plumbing … no server endpoint"*) rather than quietly widening it.
- **Shape declared replication-ready.** The collection carries `submission_id`, `field_id`, the
  discarded value, and who-and-when — which decision 79 already requires the replicated rows to
  carry, so the fields exist to copy. Promoting this to a cross-device audit trail later is
  *adding a table and a policy*, not a redesign.
- **The consequences the operator surfaced at triage stand and are accepted, not mitigated:** the
  record is per-device (a manager cannot see that a crew member's food-safety reading was
  overwritten), evictable under iOS storage pressure (which is why the mockup carries a
  storage-error plate), and lost on reinstall.
- **Retention** stays the mockup's 30 days as a local sweep. The number itself is reopened and
  belongs to `sync-rxdb-conflict-notice-mockup-amendments`, which must draw it as a visible
  placeholder rather than a settled fact. Implementations read it from one named constant.

Written onto the `sync-rxdb-collections-and-table-contract` roadmap card so no run re-litigates it.

**Decision 90 — B-09 and B-16(b) promoted from the backlog as `test-harness-fail-loud`.**

§15k sets an architecture-blocking bar, deliberately high because insertion re-sequences the build
order. The argument put to the operator and accepted: **every remaining card in this milestone is a
security or correctness card whose only proof is these two suites, and both suites can currently
report success having executed nothing.** Verified by execution at the planning session, not read
from the backlog entry — `task test` still runs 19 of 20 spec files in a fresh worktree three
slates after B-09 was filed, and skip-on-unreachable-DB is **repo-wide**, which is broader than
B-16 states (measured across `recipes`, `workflow`, `inventory`, `receipt`, and `sync`).

The card it blocks is on the same slate: `sync-rxdb-row-visibility-rls`'s entire gate is an
attack-variant Go suite in `internal/sync` proving RLS discriminates, and under today's harness
that suite can report success on zero executed tests. That is not a weaker gate; it is a gate that
lies. B-09 and B-16 marked `promoted → test-harness-fail-loud` in BACKLOG.md.

**B-16(a)** — reviewer prompts must forbid dropping a database the reviewer did not create — is
**not** in the card. It is standing G6 dispatch text and is written into the launch prompt directly,
which is where the backlog entry itself says it belongs.

**Decision 91 — `sync-rxdb-conflict-notice-mockup-amendments` fanned out of
`sync-rxdb-conflict-notice-ui`.**

The parent is ATTENDED-BLOCKED (decision 82) and cannot enter a slate. What blocks it is not code —
it is that the committed plates do not yet show amendments A-1 and A-2, so there is nothing for the
operator to sign. Drafting revised plates is unattended-safe by exactly the argument that produced
the original mockup card: CLAUDE.md gates *production code* behind the sign-off, and the mockup is
the artifact that gate consumes. The parent keeps everything else and **stays ATTENDED-BLOCKED
until the operator signs the revised plates** — the fan-out produces them, it does not discharge
the block.

**Process note — the split that was deliberately NOT performed.**
`sync-rxdb-replication-and-conflict-handler` carries three mechanisms and needs a fan-out before it
can be slated. This session declined to perform it, and the decline is recorded so it reads as a
decision rather than an omission: the obvious seam is plumbing-vs-algorithm, but the
`conflictHandler` half's contract includes *"`conflict$` must surface the discarded value"*, which
needs a live replication instance to emit `conflict$` — i.e. the plumbing half. A split that hands
the algorithm card a requirement it cannot integration-prove produces a DONE that isn't. The seam
needs designing, not guessing, and belongs at the head of the next planning session.

## T-28 — Morning-triage resolutions (2026-07-29, overnight-20260729-2)

Run `overnight-20260729-2` reviewed attended and merged to `dev` as `f35fa56` (`--no-ff`).
3 cards landed (`test-harness-fail-loud`, `sync-rxdb-collections-and-table-contract`,
`sync-rxdb-conflict-notice-mockup-amendments`), 2 parked (`sync-rxdb-row-visibility-rls` —
park note merged, zero code; `app-timezone-unify-new-york` — not merged, branch preserved).
Gate evidence in this section is sourced from an adversarial re-execution in an isolated
worktree with its own `npm ci`, own Go builds and own `hq_adv_*` databases, never from the
closeout's own gate lines. Seven forks in `DECISIONS-NEEDED.md` resolved below.

**Decision 92 — the row-visibility projection is fed by `postgres_fdw` from the substrate to
HQ, and decision 61 is REVERSED.**

Chosen over deferring to `sync-hard-cutover` (option d), native logical replication (b′), a
transactional outbox (b), 2PC (c) and restructuring the assignment write path (e). Card B1
settled the topology tonight in the direction that makes decision 61's contract impossible:
the projection and the mutation are in two different Postgres servers, `max_prepared_transactions`
is `0` at both ends, and `Sign()` is an allowlist that can only emit `authenticated` — so no
transaction can contain both and no restructuring of the mutation changes that. The operator's
requirement, stated as a user story, was that a revoked crew member's phone stop showing the
checklist **on the very next sync, with no window at all**. Only (a) and (d) deliver zero window,
and (d) costs the milestone: `sync-rxdb-row-visibility-rls` cannot land first and three of the
four remaining Activity 1 cards sit behind it. (a) reads HQ's live tables through foreign tables,
so there is no projection to write and "same transaction" is vacuous. The extension was proven
installable at both ends by executing the C symbol. The accepted standing cost: **HQ's Postgres
is on the network path of every RLS row check.** (b′) was rejected on the park note's own warning —
being *nicer* than (b) makes the 3am reach for an async option easier, not safer, and both leave a
stale-permissive window. Decision 61 is not wrong about what is wanted; it was written against a
topology that had not yet been chosen. If `sync-hard-cutover` later co-locates the two databases,
the fdw becomes vestigial and decision 61 comes true structurally — this reversal does not
foreclose that.

**Decision 93 — HQ and sales-processor both move to `America/New_York`, in a coordinated
release, and both contract documents plus assumption A5 are updated.**

Chosen over keeping the money paths on Chicago as a published operating constant, and over
re-affirming decision 83 while letting it silently amend A5. The operator's framing: a payroll
week and a food-cost week must describe the same seven days. The collision was real and verified
first-hand at triage — `21-SALES-PROCESSOR-CONTRACT.md:27` pins `America/Chicago`, `:67` publishes
`(created_at AT TIME ZONE 'America/Chicago')::date BETWEEN from AND to` (the exact expression A1
replaced), `:319` carries **A5 — "If the food truck moves to a different TZ, both repos must
update"**, and `999.2-SALES-PROCESSOR-CONTRACT.md:30` repeats the pin. Decision 83 named
sales-processor only as a downstream consumer and never addressed the published contract, so the
ruling and the contract were both true statements about different things. A1's G6 returned REJECT
and the park was CORRECT: A1 reported *"Nothing parked — no site turned out to be deliberately
Chicago"*, which is false against the repo's own artifacts. **The resuming card's scope is now
wider than A1's was:** it takes migration `0072`, converts every site including `trends.go:89-98`
(which would otherwise leave two 12-week COGS windows on two different zones), AND edits both
contract documents and A5. Nothing ships to prod until sales-processor's matching change is ready
— until both land, one repo is wrong, and the disagreement is one hour at each period edge on rows
with no extracted `event_date`. Nothing is broken today: A1 did not merge, so the tree still
carries ~20 Chicago and ~8 New York sites exactly as it did yesterday.

**Decision 94 — the Setup-tab Badge Reset follows the app's configured timezone, not the
browser's.**

Chosen over keeping today's follow-the-device behaviour and over an explicit picker. The operator's
framing: a crew member opening the app in the morning should see the list their coworker sees, so
the reset must not depend on whose phone saved the form. `inventory.html:2713` currently writes the
browser's zone into `repurchase_reset_config.timezone` on every save, and
`tests/inventory.spec.js:2022` — *"badge reset saves with browser timezone, not hardcoded value"* —
asserts that behaviour and passes. **That test is asserting the defect and must be rewritten, not
worked around.** Someone once chose "follow the device" and left no ledger entry; this decision
records the reversal so the next reader does not rediscover it as a regression. Rides on the
resuming timezone card.

**Decision 95 — a removed question keeps its label, struck through and read-only. This
SUPERSEDES both readings of the mockup's open decision (i).**

The mockup asked the operator to pick between Reading A (a removed-field row counts in the chip
base) and Reading B (it moves to `+N`), drawn over identical data with neither recommended. The
operator answered neither: **"show the deleted question crossed out and read only so that the user
isnt confused."** The plates had drawn the raw field id `fld_prep_sink_temp` in muted monospace, on
the stated grounds that "the template no longer holds a label for it" — **true of the template and
false of the submission.** Verified at triage: `template_snapshot` is `json.Marshal(tmpl)` of the
whole template (`repository.go:695`) and `Field.Label` is on the marshalled struct
(`model.go:44-57`), so the discarded document carries its own frozen label for a field the live
template has dropped. The rider is therefore buildable with **no new schema requirement** — but it
does make the snapshot's *shape* load-bearing for the conflict-notice UI, which promotes B1's
recorded-not-fixed item R-C (`template_snapshot` is `{type:'object'}` with no nested `properties`
and nothing rejects a malformed value) from an open question into a dependency. Recorded as
amendment **A-3** and offered back with consent as preference candidate `ux/C-1`. Consequence for
the counting question: with the row visibly struck through and read-only, Reading A's
"2 answers / Restore all 1 of mine" mismatch is legible on screen rather than arithmetic, so the
headline counts what was taken from the crew member (Reading A) and the `+N` line keeps meaning
only "we couldn't identify" — pooling a perfectly-identified removed question with a genuine
unknown was the worse outcome. **Stated as an inference from the rider, not as operator words** —
if the intent was Reading B's counting with a struck-through label, say so and it changes.

**Decision 96 — the retention window stays 30 days. Decision 80 stands as written.**

Chosen over 14 days (triage's recommendation) and 7 days. Reopened at morning triage 2026-07-28
and drawn in the plates as the placeholder token `⟨30⟩` in a dashed box with `⟨7⟩` beside it and
body copy byte-identical between them, so the screen was demonstrably indifferent to the value.
The accepted costs are the ones the plate names: a longer list to scroll, and a promise the device
may not keep — the record is local-only, per-device and evictable under iOS storage pressure
(ledger T-27 decision 89), so a 30-day claim is more often wrong than a short one. The
storage-error plate is the designed state for exactly that. Implementations read the number from
**one named constant**; no surface restates the literal.

**Decision 97 — the conflict sheet caps at 10 groups with an "and N more" line; no date filter.**

The `edge-many` plate raised this as an open question — "beyond roughly ten groups this sheet needs
a cap or a date filter. Not designed here." Operator chose a cap over a date filter and over
leaving it undesigned. **N = 10 decided by triage** (plumbing, not product): it is the number the
plate itself already names, and 10 groups at the collapsed density is the point where the sheet
stops being scannable on a 480px screen. Rows below the line are not dropped — the sheet shows the
10 most recent groups by document and states plainly how many older ones are not shown, so the
count on the banner still reports the true total. A date filter was rejected as more design and
more code for a case a 1–5 person truck reaches only after an implausible offline stretch inside a
30-day window; if it is ever reached, the cap degrades honestly instead of hiding the overflow.

**Decision 98 — revision 2 of the conflict-notice plates is SIGNED, conditional on amendment
A-3. `sync-rxdb-conflict-notice-ui` is no longer ATTENDED-BLOCKED.**

All 16 plates walked at triage — read back as PNGs with the Read tool, light renders, not described
from the spec (dark renders not inspected; the shoot script measures both and reports parity).
Amendments A-1 and A-2 hold at the worst case, not the easiest: A-1's three banner lines coexist at
480px with no truncation and no ellipsis on the four-answers/two-handled/two-unidentifiable plate;
A-2's confirm names the loss in its title (*"Replace 3 of Dana M.'s answers?"*), lists all three
server values struck through with who saved each and when, and labels its primary button **Replace**
rather than Restore. **The plates are honest about their own boundaries in ways the easier copy was
available and not taken** — `edge-storage` leads with "if Try again doesn't bring it back, it's
gone" before the reassurance, and `empty` refuses to say "nothing was overwritten" because three
different situations produce that exact screen and the app cannot tell them apart. The `limits`
plate draws three things no UI can fix, and A-2 hardens the third (who-and-when on the row) from a
graceful degradation into a **hard requirement on `sync-rxdb-schema-and-replication`** — if that
card declines to carry it, the confirm plate cannot be built as drawn. **The signature is
conditional in one respect only:** `edge-removed`, `openq-count-a` and `openq-count-b` as committed
draw the raw field id, which decision 95 overrides, so the UI card must redraw those plates and
note the deviation in SUMMARY.md per CLAUDE.md's mockup rule. Triage's own reservation, recorded
rather than blocking: `a1-banner` puts four figures on one screen plus a batch button reading a
fifth number, all internally consistent and all documented, but it is a lot of counting at 6am with
wet hands — worth watching in the built UI.

**Decision 99 — decision 61 governs this card by analogy, not by letter.**

Decided by triage, not put to the operator: the card and decision 61 both name the
`app_permissions` mutation, while the projection B2 actually needs is fed by
`template_assignments` (`repository.go:236` DELETE / `:249` re-insert). The substitution was silent
until G6 caught it. It does not change decision 92 — `app_permissions` is likewise on `5433` and
absent from `46011`, so the topology argument holds either way — and decision 92 supersedes the
question entirely by removing the projection. Recorded so the substitution reads as a decision
rather than an omission.

**Decision 100 — armed reds are named by title/grep handle, never by line anchor.**

Decided by triage. `tests/sync.spec.js:1198` is `await p.waitForTimeout(400)` inside a helper's loop
body; it names no test, and the test it used to name is now at `:1372`. It has been known dead since
2026-07-24 with an unactioned migration item filed the same day, and **tonight's slate still armed
it** — so every card told to "expect `:1198`" for five nights was told to expect something
unobservable, and every report saying "it passed" was unfalsifiable. Two cards hit it independently
this run. `:446` `[LST-17]` is live and correct. Going forward a slate's preconditions table carries
a grep handle, and the other armed reds get swept for line anchors. → **B-25**.

**Decision 101 — the gate ladder gets a written definition, and G5 is retired as never-defined.**

Decided by triage, and the finding is worse than `DECISIONS-NEEDED.md` filed it. Every slate since
07-15 and every launch prompt inherit gates **G1–G6** "unchanged from
`reference/overnight-run-plan-20260707.md`" — the adversarial reviewer confirmed by `find` that
**the file does not exist anywhere in the repo.** The contract was recoverable from practice and the
runs used it (G1 build+vet, G2 Go+Playwright, G3 red-first re-verified by G6, G4 `sw.js` idempotence
+ version parity, G6 adversarial review), but **G5 has no definition in any sense the runs use** and
is not practiced. So every run this month has been graded against a ladder with no written
definition. Chosen over changing the prompts to point at whatever currently defines the gates: the
prompts are right about wanting one durable source, and the fix is to supply it. G5 is recorded as
never-defined rather than renumbered, so the gap reads as history instead of a hole. → **B-26**.

**Decision 102 — `inventory.spec.js:883` goes to the backlog, unattributed.**

Decided by triage. `item modal pre-fills search with current line item text` fails with
`Expected "Special Sauce", Received "Test Item"` — **proven pre-existing by reproduction**, not by
argument: G6 ran the preceding specs with B1's new spec file entirely absent and got the
byte-identical failure, and `inventory.spec.js` alone on a fresh database passes 150/150. It is
cross-spec pollution from one of `broadcast-rerender` / `grant-enforcement-parity` / `index`. The
mechanism first proposed — a `.first()` collision over a shared `eventDate` — is **wrong** (the
pending list is `ORDER BY created_at DESC` with no re-sort, so `.first()` is the newest row and
`event_date` is not in the sort key); recorded so a wrong mechanism does not become folklore. The
likelier cause is `seedPendingPurchase` swallowing a failed POST (`tests/inventory.spec.js:70`).
`playwright.config.js` defaults to `retries: 1`, which is why the baseline reads green; cards at
`--retries=0` see it. It did not surface in either the run's or triage's `--retries=0` full suite,
consistent with load/ordering sensitivity. → **B-27**.

**Decision 103 — the three guard defects found at triage ride the resuming work; the merge
stands.**

Decided by triage. The adversarial re-execution reproduced every closeout gate number to the digit
and refuted none, and found three defects the run did not report — **all in guards, not in shipped
behaviour**, which matters because the run's headline card is a guard-integrity card.
(a) `verify-test-harness.sh` Check B runs one aggregate `go test`, so **six of seven packages can
report `ok` on a dropped database while the gate prints PASS**; only a 7-of-7 revert reds it. The
production fix itself is sound — all seven packages exit 1 individually under three different
unreachable-DB shapes, in 0.02s where the DSN resolves, so the fail-loud claim does not depend on
hanging. (b) Check A2's spec-file floor of 20 **lost its bite during this very run**: B1 added
`tests/sync-schema.spec.js`, making 20 static + 1 generated = 21, and nobody ratcheted the floor —
so moving `features/` away and letting `bddgen` emit nothing still reports `20 files → PASS, exit 0`,
re-opening the B-09 detection gap one file wide. (c) `shoot.mjs` measurement 6, the arithmetic check
**added in the repair round** to close the "value present but not right" hole, walks `.plate` with no
population floor while the file's own header asserts six of seven measurements are pinned; renaming
the class yields `0 counting plates reconciled -> PASS`. The other three repaired checks are
genuinely falsifiable, each confirmed by mutation. **The closeout's own diagnosis — "this repo's
characteristic bug is a check whose subject set can go empty" — is correct and not yet cured: all
three findings are that same shape, inside the checks the run added to cure it.** Filed as
**B-22/B-23/B-24** rather than fixed at triage, because a guard repair deserves a red-first test and
triage does not write production code. Also confirmed: **B-21's emitter defect is fixed in
practice** — 0 of 32 commits unparseable under `git interpret-trailers --parse`, all five merge
commits included, against 14 of 33 last run.

---

## T-29 — Morning-triage resolutions (2026-07-31, overnight-20260801)

Run `overnight-20260801` reviewed attended and merged to `dev` (`--no-ff`). **4 of 4 cards
landed, nothing parked**: `app-timezone-unify-new-york`, `sync-rxdb-row-visibility-rls`,
`sync-rxdb-replication-and-conflict-handler`, `sync-rxdb-conflict-notice-ui`. Gate evidence in
this section is sourced from an **adversarial re-execution** in an isolated worktree with its own
builds and its own `hq_adv_*` databases on `TEST_PORT=8296` — never from the closeout's own gate
lines (§15ag.87). Two of three forks in `DECISIONS-NEEDED.md` resolved below; **Fork 3 remains
open and is carried forward.**

🛑 **The closeout's headline gate claim was REFUTED, and this is the first time that has happened
in five triages.** The closeout and HANDOFF both record `G2 (Playwright) --retries=0 exit 0 —
733 passed / 6 skipped / 0 failed of 739`, and commit `3ec25a5` says "gates green on the merged
tree." A clean isolated re-run of the identical tree exits **1**: `1 failed / 6 skipped / 732
passed (23.3m)`, failing on `tests/inventory.spec.js:883` with `Expected "Special Sauce",
Received "Test Item"` — **B-27**, cross-spec pollution, another spec's fixture leaking in. The
run was honest about every other number (all reproduced to the digit) and this one did pass on
its leg; the defect is that a suite carrying a known intermittent red can report green and be
recorded as evidence. **Four prior triages found the gates honest and the prose about them not;
this is the inverse and it is worse.** Recorded so the next reader does not inherit
`3ec25a5`'s claim unchallenged.

**Decision 104 — merge despite the red, because the red is not this run's and `dev` is not
production.**

Chosen over holding the branch until B-27 is fixed, and over rejecting Track B. B-27 predates
this run, is already filed, was touched by no card, and reproduces on a *quiet* box (load ~3.2) —
which is backwards from the load-sensitivity story B-27 itself carries and is its own open
question. Holding four completed cards hostage to a flake filed days earlier would stall a week
of sync work to make a point about a suite that was already not green. The operator's framing was
that the old bug stays theirs to fix rather than becoming a gate on unrelated work. **What was
explicitly accepted:** `dev` now carries a suite that is not green, and nothing about that is
being smoothed over — the merge commit records the exit-1 and names B-27 in full. One consequence
worth stating because it argued the *other* way and was overruled anyway: per **B-33** the
unmerged run branch was the only remaining guard against re-executing an already-landed slate,
and merging disarms it. That guard is now gone until B-33 ships.

**Decision 105 — decision 92 (fdw read-through) STANDS, and is scoped by a new standing rule:
replicate what the open checklist needs, never all collections at once.**

Chosen over reversing decision 92 and over accepting the measured cost as-is. G6 measured the RLS
path at **~23 ms per row, linear** (5 rows → 177 ms; 205 rows → 4,698 ms) and the card named this
its own PARK trigger. Triage's options were framed around the *substrate*; the operator rejected
that framing and located the defect in the *scope* instead — **rider, verbatim: _"This design
seems wrong. An individual checklist seems to be small so only one list should be loaded at a
time, which would just have a few rows as long as all collections aren't loaded at once."_**

Verified in code before recording, rather than accepted on assertion: `startHQReplication`
(`sync-rxdb/client.js:378`) loops over every replicated collection and calls `replicateSupabase`
with `pull: { batchSize: 50 }` and **no selector, filter or query modifier**. Four collections
replicate in full — `templates`, `checklists`, `responses`, `approvals` — and `responses` holds
every field answer of every submission ever taken. So the client does pull whole collections, the
RLS predicate is re-evaluated per row on every page, and 20 pages × 50 rows × 23 ms is precisely
the ~23 s figure Fork 1 reported. **The operator's read is correct and the arithmetic is the
design's, not the substrate's.**

This reframes the cost as a bounded one: at a few dozen rows the same predicate costs well under
a second, and 177 ms for 5 rows is the honest case. It also fixes a second problem nobody had
raised — unbounded phone storage as the business ages, since `responses` grows forever and every
device was to hold all of it. **The standing rule, which binds every remaining sync card:**
replication scope is per-open-checklist, not per-collection; a card may not widen it without a
recorded decision. `sync-hard-cutover` and `sync-cache-and-identity-hygiene` inherit this
directly. **What is explicitly NOT settled:** the 23 ms constant was measured through Docker
loopback NAT, which production does not have — the linear *shape* is structural, the *constant*
is not, and it should be re-measured on production-like topology before any card relies on a
specific number.

> **Amendment to decision 105, same session — the rule is GENERAL, not checklist-specific.**
> Decision 105 was recorded from the operator's first statement, which was scoped to checklists.
> Asked to reword it for the preference store, they restated it as a general principle —
> verbatim: *"fetching should always be done in batches whenever it is seen that a list could
> grow unbounded."* That is wider than replication and wider than this cycle: it binds **any**
> client-side fetch over a collection that can grow without bound, including plain API list
> endpoints, not only RxDB pulls. The checklist scope remains the concrete instance that produced
> it, and `sync-replication-scope-per-checklist` remains the card that discharges it here — but a
> future card adding an unbounded list endpoint is in scope too, and should not read decision 105
> as permission because it is not a sync card. Offered back and recorded with consent as
> **`architecture/C-2`** (pending, not adopted — nothing cites it until the operator renumbers it
> to a free `P-n`).

**Decision 106 — sales-processor gets TWO notices, sent separately, and the June drift goes
first and alone.**

Chosen over one combined notice at the coordinated release, and over investigating the damage
before saying anything. The provenance was verified at source by the adversarial reviewer, not
taken from the card: `875e26c` (2026-06-05 04:24) archived `21-SALES-PROCESSOR-CONTRACT.md`
publishing the completeness gate **without** `COALESCE`, accurate to the code as it then stood;
`cf959bd` (2026-06-06 00:27), quick task `260606-0gh`, added
`COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date)` to
`PeriodSummaryHandler` and **never touched the contract**. The full history of that path shows no
update between the code change and 2026-07-31. Concretely: a May 29 purchase ingested June 2 used
to fall outside the May window and **not** block payroll; since June it **does**. Sales-processor
may have been receiving an undocumented `ready:false` — a blocked payroll run — for eight weeks
with no way to reconcile it against the contract it holds. It goes first and alone because it is
**already live and already affecting them**, while the timezone move is still unshipped; folding
a live eight-week-old payroll defect into a forward-looking release note would let the urgent one
hide behind the scheduled one. The timezone notice follows as coordinated-release comms when both
repos are ready. 🛑 **Neither notice is sent by this triage, and nothing deploys until both repos
land** — one of them is wrong until then.

**Decision 107 — the 🔴 `HQ_SYNC_REST_URL` flag is DISARMED, on evidence.**

Decided by triage, not put to the operator: the flag's own standing rule is *"disarms on
evidence, never by the run asserting it"*, which names an evidence bar and makes triage — the
party holding the evidence — the decider. Escalating it would have handed back the one judgment
the rule assigns here. The evidence is independent of the run: the variable is set **nowhere** in
the tree (every hit is a comment, doc, test name or night-crew artifact); B2's suite ran **live**
against a real substrate (19 numbered attack variants plus positive and control); and withholding
the policies via `SYNC_RLS_SKIP_POLICIES=1` reddens it hard — 17 subtests fail, exit 1 — so the
guard bites rather than merely printing PASS. The population assertions at
`rowvisibility_rls_test.go:596` and `:632` are real `t.Fatalf`s on an empty subject set, which is
the B-22/B-23/B-24 bar. Re-confirmed on the **merged** `dev` tree at triage: 27 subtests,
`--- PASS`, not `--- SKIP`. **Re-arms** whenever a card touches the sync proxy or replication path
or introduces a REST client.

**Decision 108 — a standing evidence rule: `internal/sync` gate evidence must prove the suite
RAN, not merely that the package said `ok`.**

Adopted because **B-36** proves those two are indistinguishable — `resolveSpikeConfig` turns any
docker-compose resolution failure into `t.Skip`, so the ladder's `9 packages ok` line carries zero
information about the security suite. From now on, gate evidence citing `internal/sync` must
include `-run TestRowVisibilityRLS -v` output showing subtests executed. This is a reporting
requirement standing in for a mechanical fix; B-36 carries the real repair.

**Decision 109 — `[LST-17]` STAYS ARMED; the HANDOFF's recommendation to fold it into B-32 is
rejected.**

Decided by triage. The run recommended demoting it on the grounds that it "fired in roughly three
legs and passed in five" and therefore cannot carry evidential weight. But the adversarial probe
tested the guard rather than its frequency: injecting a never-decrement regression into
`getProgress()` reddened `tests/sync.spec.js:446` (`Expected "0/1", Received "1/1 items"`) while
its sibling increment test at `:1006` stayed **green** — a targeted, discriminating tripwire.
Its claimed intermittency was neither reproduced nor refuted (one clean full-suite leg is not a
sample), so the demotion would rest on an unmeasured hunch about a guard that demonstrably works.
🛑 **Consistent with the standing rule that non-reproduction retires nothing** — applied here in
the direction that protects a working guard rather than the direction that retires a red. Also
confirmed: the bare tag `[LST-17]` matches **two** tests, so the full title remains mandatory, and
`tests/sync.spec.js:1198` is a bare `}` — dead, correctly not armed.

**Decision 110 — the attended two-device convergence check is DEFERRED, and B-15 is not yet
scheduled.**

Attention budget presented: ~15–20 min attended against a runbook, versus nothing delegable today
(automating it is card **B-15**, roughly one night unattended). Deferred because nothing is
shipping — the flag re-arms before `task prod:deploy` regardless, and the precache manifest will
change again before it matters, so spending attention now buys a result about a state that will
not be the state at deploy time. **Still ARMED.**

**Fork 3 — the conflict banner's headline figure — REMAINS OPEN.** Not resolved at this triage
and not deferred on its merits; it was dropped from the operator round to keep the question count
workable and is carried forward honestly rather than silently. It is explicitly non-blocking: if
nothing is done, C2 ships as signed and is correct. The question is whether the past-tense figure
(`4 were overwritten`) should headline the banner when the two figures that drive action
(`2 still to review`, `Restore all 2`) are in the smallest type on the screen.

**Two C2 items also carried forward, both the operator's:** the **new mockup deviation** —
production's dark confirm no longer matches the signed `a2-confirm-dark` plate because V-1 was
fixed in `workflows.html` and the signed plate deliberately left alone (SUMMARY.md §1a identifies
the two-line change if re-signing the plate is preferred) — and **F-4**, `PLAN.md` landing after
the implementation while carrying the contract the verifier gate grades against.

**Preference coverage: unmeasured.** `night-crew decisions audit --repo . --run 20260801` returns
*"No gray areas routed through the resolver yet"* — nothing from this run was routed, so coverage
is not low, it is absent. Given the night produced three operator forks and six backlog items,
that is a gap for the next evening's offer-back rather than a number to report.

**Tooling, recorded because it degraded this ritual.** The installed `~/go/bin/night-crew` is
`v3.0.0+1`, built 2026-07-23, and has **no `worktrees`, `run-evidence`, `launch-prompt` or
`workflow` verb**. Step 1's worktree sweep was therefore performed by hand with `git worktree
list` and `--no-merged`, which lacks the real verb's cherry-pick-aware ancestry logic, and
`run-evidence check` could not be run at all (moot — **B-33** is precisely that it is blind here).
This is the tooling-tracks-main case, not a PATH gap: the binary ran fine and the verbs exist on
`main` @ v3.0.2. **Refresh with `task nc:update` before the next ritual depends on a newer verb.**

**Stranded work found by the manual sweep, reported and NOT merged** (a patch-equivalent branch
can still produce a content-duplicating merge when files moved underneath — B-133):
`card/f1-workflow-submission-status-default` (1 commit, 2026-07-25 — a **red-first spec with no
fix behind it**, six days stranded, the one worth attention);
`card/d1-syncspec-deflake` (2 commits, 2026-07-21, head commit is a `Revert`);
`parked/f1-trends-endpoint-20260720b` (2 commits, known park); `docs/claude-md-night-crew`
(5 commits, 2026-07-26).

---

## T-30 — Slate-planning resolutions (2026-07-31 evening, `/nc-slate-plan` for `overnight-20260801-2`)

Resolved inline during slate planning under the §1 fork gate ("resolve inline by default"), not
carried to a morning triage. Both were verified in code before being put to the operator, and both
gate cards that would otherwise have entered the slate carrying an open decision.

**Decision 111 — RxDB push writes mirror the read rule, PER TABLE. Templates stay client-write
deny-all; approvals are approver-only.**

The blocker: `sync-schema/sql/0003_rls_policies.sql` ships SELECT policies only — grepped, zero
`WITH CHECK`, zero `FOR INSERT`/`FOR UPDATE` — so RxDB **push** replication is deny-all. Card
`sync-rxdb-row-visibility-rls` named this explicitly and deferred it to "a follow-up card". **That
card did not exist.** `sync-hard-cutover` makes RxDB the single write path and is therefore blocked
on a card nobody had written, carrying an open product decision: `ResolveEntityAccess` is a *read*
fan-out resolver, and extending it to writes invents a permission semantic rather than porting one.

Put to the operator as three user stories; they chose **mirror reads, per table** over one uniform
predicate and over own-rows-only. The resulting per-table contract, which the new card implements
verbatim:

| Table | SELECT (shipped) | INSERT/UPDATE (new) |
|---|---|---|
| `checklist_templates` | `hq_can_see_template(id)` | **DENY-ALL, deliberately.** The builder keeps the existing REST path; no phone writes a template definition. Asserted by a refusal variant, not left as an absence. |
| `checklist_submissions` | `hq_can_see_template(template_id)` | `WITH CHECK (hq_can_see_template(template_id))` — closes the lie 0003:243 names ("a pushed row claiming a template_id its author can see"). |
| `submission_responses` | `hq_can_see_field(field_id)` | `WITH CHECK (hq_can_see_field(field_id))`. Field-scoped, not submission-scoped — `submission_id` is nullable for offline drafts and that is the whole offline story. |
| `submission_rejections` | **none today — deny-all read** | Approver-only write, and **it also gains a SELECT policy**. |

🛑 **Two consequences the operator's answer forces, recorded so the run does not rediscover them.**

(1) **Approvals must become readable, not only writable.** `submission_rejections` has RLS enabled
with zero policies today (deny-all both ways, decision-with-evidence via variant V17). A device
that can write a row it cannot read back breaks replication — the pull would never return what the
push wrote. So the card adds `for select using (hq_can_see_field(field_id))`, matching
`submission_responses`: the assignee whose field was rejected must read their own feedback, which
is the reject-with-comment path crew members actually depend on.

(2) **This is the FIRST place `assignment_role` is filtered on, and it deliberately breaks an
inherited property.** `hq_can_see_template` never filters on `assignment_role` — an `'approver'`
sees exactly what an `'assignee'` sees, and card `sync-rxdb-row-visibility-rls` recorded that as
"knowing, not accidental" with the standing note that **changing it is a separate card**. The
operator's answer makes writes the place it changes: a new `public.hq_can_approve_template(tid)`
predicate = `EXISTS` an assignment with `assignment_role = 'approver'` **OR** the unconditional
`roles && ARRAY['admin','superadmin']` admin arm. Reads keep the old property untouched; only the
approval WITH CHECK uses the new one. That asymmetry is the decision, not a side effect of it.

**Decision 112 — `api-cache` is NOT retired. `sync-cache-and-identity-hygiene` is re-specified as
per-identity cache partitioning, and this is a planner call with a code citation behind it.**

The card's obligation 3 has carried "the expected answer is to retire the route entirely — once
RxDB replicates, offline data comes from IndexedDB and `api-cache` is obsolete" since decision 57.
**That premise is false, and was already false when written.** `build-sw.js:149` registers
`urlPattern: /\/api\//` — a NetworkFirst route over **every** endpoint in the app, all five tools.
RxDB replicates **four** collections, all of them `workflow`. Retiring the route would take offline
API reads away from Inventory, Users, Onboarding and Purchasing, none of which RxDB has ever
covered. Decision 105's per-open-checklist scope narrows RxDB's coverage further still, so the
argument is weaker now than when it was made, not stronger.

Decided rather than escalated (plumbing, §"decide-plumbing-yourself"): the defect obligation 3
actually names is a **cross-tenant read** — a URL-only cache key with no `Vary`, so user B's device
serves user A's cached API responses. The fix is to key the cache by identity and purge it when
identity changes, which is **the same mechanism** obligation 7 needs for `localStorage['hq_apps']`
(never cleared on logout; `index.html:224` parses the previous user's slug list in the fail-closed
branch) and for the login-without-logout path. The card therefore collapses from four unrelated
errands into one mechanism with three call sites, and stops being blocked on a retirement that
should not happen. Obligation 8 (the stale comment at `tests/sync.spec.js:1584`) rides along
unchanged.

**Also folded, and stated as a planner call rather than a backlog promotion (§15k bar not invoked):**
**B-36** goes into the new write-policies card. B-36 is that `internal/sync` prints `ok` and exits 0
while `t.Skip`-ing the entire RLS attack suite, so the ladder's "9 packages ok" line carries zero
information about the security gate. The new card's whole gate is a suite of exactly that shape, in
exactly that package. Writing a second attack suite into a package whose gate cannot prove it ran
is building on a foundation that needs rip-out — so the fix rides the card that depends on it,
rather than waiting for a night of its own. Ledger T-29 decision 108's reporting rule stands until
it lands.

**Tooling, recorded because it degraded this ritual too.** `~/go/bin/night-crew` is still the
2026-07-23 build with no `workflow` verb (T-29 recorded the same thing); the preflight was run from
a binary built out of the pinned `night-crew-main` worktree (`258d723`, v3.0.2) into scratch.
Separately, `night-crew okr grade --repo .` returns **"no metrics.jsonl found under
.night-crew/runs"** — the live KR grader is **blind in this repo** for the same reason **B-33**
makes `run-evidence check` blind: the run directories are `2026-08-01-autonomous`, not runid-keyed.
The milestone remainder in `slate-20260801-2.md` is therefore reported in **cards from the live
roadmap**, and the KR grades are reported as **unmeasurable here**, not quoted from HANDOFF prose.

**Decision 113 — an uncontested delete beats a concurrent edit, and the discarded edit is REPORTED
and RECOVERABLE. The silent-loss path is closed by reporting, not by changing who wins.**

The blocker: `sync-rxdb/conflict-handler.js:105-160` carries a `🛑 OPEN QUESTION INHERITED BY
sync-hard-cutover` and says of itself *"this card has no standing to answer it."* Under the merge
rule that shipped last night, `_deleted` is not reserved — it merges by the ordinary rule — so a
fork that sets `_deleted: true` against a master that edited some other field produces an
**uncontested** delete: it survives, master's edit lands on a tombstone and is annihilated, and
**nothing is reported**, because no field clashed. Decision 50 does not cover it: decision 50 is
written about FIELD edits and a delete is not a field edit.

**Verified in the backend before putting it to the operator, because the comment's own history
shows how easy it is to get wrong** (an earlier draft claimed HQ hard-deletes none of the four
mirrored tables — false, and the comment says so). HQ hard-deletes **three of the four** from live
production paths: `saveResponse` (`backend/internal/workflow/repository.go:811`) runs
`DELETE FROM submission_responses …` whenever the value is null — **that is unchecking a checkbox**,
the highest-frequency write in the tool; `unsubmitChecklist` (`:1289`, `:1297`) deletes the
`submission_rejections` rows and then the `checklist_submissions` row; and a template edit that
removes fields deletes draft responses (`:321`), with `cleanupOldDrafts` (`:1334`) sweeping on a
schedule. Only `checklist_templates` is delete-free (it archives via `archived_at`). So this is not
an exotic edge — it is the most likely conflict on this schema, and it is unreachable today on
exactly one ground: **no HQ page writes through RxDB yet.** `sync-hard-cutover` opens that path.

Put to the operator as three user stories. They chose **the uncheck wins, and the other party is
told** over *edit-blocks-delete* and over *most-recent-write-wins*.

**Why this is the consistent answer rather than a new policy:** decision 50 already rules that a
genuine same-field clash falls back to master-wins **and `conflict$` must surface it to the user
with the discarded value recoverable.** Decision 113 applies that same principle to the case
decision 50's wording missed. **Who wins does not change** — today's behaviour is already
"uncontested delete survives", pinned by the named test *`_deleted` participates in the merge — an
UNCONTESTED local delete survives*. **What changes is that it stops being silent.**

**The spec line `sync-hard-cutover` inherits:** a delete that annihilates a concurrent edit MUST
emit a conflict record carrying the discarded edit, so it appears on C2's conflict sheet with the
discarded value restorable — the same surface, the same Restore affordance, no new UI component
(which matters: a new component would require mockup sign-off an unattended run cannot obtain).
Rejected framings, recorded so they are not re-derived: *edit-blocks-delete* was rejected because a
crew member on the truck is the party who observed the item was not done and should not be silently
overruled by someone who was not there — and a box that re-checks itself is worse than a reported
loss; *last-write-wins* was rejected because it depends on device clocks agreeing, which is the
exact assumption class the RxDB migration exists to escape.

🛑 **The existing pinned test must be EXTENDED, not replaced.** It asserts who wins and that stays
true; the new assertion is that the annihilated edit is reported. A card that rewrites it to assert
the opposite winner has misread this decision.

## T-31 — Morning-triage resolutions (2026-08-02, `overnight-20260802`)

Night A of a two-night milestone close. 4 of 6 cards landed, nothing parked, no operator-only
fork raised. Reviewed attended; gate evidence below is from an adversarial subagent that
re-executed the gates and applied its own mutations, never from the closeout's own lines
(§15ag.87 — a card's own closeout is not evidence about that card).

**Decision 114 — R1 RATIFIED: the Taskfile is right, the doc was wrong, and four further false
deploy claims went with it.** Card P1 corrected `CLAUDE.md`'s deploy block to match
`Taskfile.yml:178-221`. Verified line by line at triage: `prod:deploy` has **no** `deps:` on `sw`
(contrast `test:` at `:40`, which does); there is **no** `ssh`/`SSH` anywhere in `Taskfile.yml`,
`backend/Taskfile.yml` or `docker-compose.prod.yml`; the container is `yumyums-prod`
(`docker-compose.prod.yml:33`, `PROD_CONTAINER` default at `Taskfile.yml:17`), not `yumyums-hq`;
`prod:rollback` (`:223`) and `PROD_COMPOSE` (`:16`) exist and were undocumented; `PROD_SSH` exists
nowhere. Ratifying R1 ratifies all five corrections. This deserved the moment it got rather than a
nod, because the previous text described a deploy path that does not exist and it is the document
an operator reads *while deploying*. One residual: `CLAUDE.md:236` still says "Go backend in Docker
on Windows box" — now ambiguous rather than false, since it *is* this box.

**Decision 115 — R2 RATIFIED: the precache count moving 29 → 31 under a scope freeze was the right
call.** The slate gave both a mechanism ("exit non-zero if any resolves to a skipped path") and an
invariant ("nothing precached may import something not precached"), labelling the latter the
actionable one; P1 implemented the invariant, so its guard also fires on a target never globbed at
all. That surfaced a live defect with no synthetic case: `log.js` is `src=`'d by **7 of 7**
precached pages and `tab.js` by **5** (counted independently at triage), and neither was in
`globPatterns` — shipped online via `COPY *.html *.js`, broken offline forever, with `tab.js`
applying `#tab=N` before paint so five of seven tools opened on a returning offline client with
every section visible and no switching. Adding two files to a manifest that reaches every phone is a
real product change, which is why it was surfaced rather than buried; the narrow reading would have
exited 0 on the merged tree and left D-KR2's exact subject live. Ratified as judgement, not scope
creep.

**Decision 116 — R3 CONFIRMED: ledger decision 108's reporting rule is KEPT, and amended.** Card A2
initially recorded that B-36's fix retired the rule; its G6 proved that false and triage reproduced
both halves by execution. With the substrate made unresolvable (`DOCKER_HOST` pointed at a dead
socket): `HQ_SYNC_SUBSTRATE_OPTIONAL=1 go test ./internal/sync/` → **`ok … 0.808s`, zero attack
variants run**; the same run without the opt-out → **FAIL** with B-36's message. B-36 closes one road
to a silent skip, not the road. The rule stands, amended: an `internal/sync` result is reportable
only when it cites `-run TestRowVisibilityRLS -v` with subtests **executed** *and* states
`HQ_SYNC_SUBSTRATE_OPTIONAL` was unset. Retiring it would have quietly reopened the hole A2 exists
to close. Noted for future readers: with a live substrate on the box, `=1` does **not** cause a
skip — the gate reads it only when resolution fails.

**Decision 117 — R4 RATIFIED: A2 was right to add `0074_sync_fdw_approver_view.sql` on a card told
it would need no migration.** `assignment_role` deliberately does not cross the FDW
(`sync-schema/sql/0002` §3a), so decision 111's own `hq_can_approve_template` is not evaluable on
the substrate without HQ-side plumbing. The migration is one read-only VIEW and one grant — no
table, column, constraint or role. The slate's park condition is scoped to "a write predicate beyond
decision 111's four rows", which did not fire: `0074` is the plumbing for row 4, not a fifth row.
Should not have parked.

**Decision 118 — R5 RATIFIED: `V18`'s in-place rewrite is authorised, not laundered.** Rewriting a
passing test to match new behaviour is how a regression gets laundered, so it was put to G6 and
re-judged independently at triage. `0004:471` genuinely adds
`submission_rejections_select USING (hq_can_see_field(field_id))`, which makes the old deny-all
assertion factually false rather than inconvenient; the rewrite changed the subtest title so the old
assertion cannot be mistaken for the new one, keeps a refusal half and the `service_role` control,
adds three-way discrimination, and grows the fixture 5 → 8 field templates with **both** population
controls updated to match. The one real hazard is stated in the file itself: the rewritten V18 is
sound only *with* 0004, so a partial revert yields a suite that contradicts the schema.

**Decision 119 — R6: `HQ_SYNC_REST_URL` STAYS ARMED; tonight's evidence proves compliance, it does
not retire the flag.** Verified exhaustively at triage across all tracked files: 21 occurrences,
**every one** a comment, a planning doc, or the constant declaration
(`backend/internal/sync/proxy.go:111`). Zero assignments in `Taskfile.yml`, any `docker-compose*.yml`,
any `.env*`, `tests/`, or CI. The flag is a standing guard, not a one-time check — it re-arms for
Night B, and `sync-hard-cutover` (S1) is the card that first sets it in a real deploy. Untracked
`backend/.env` could not be read (permission denied) and is not in the merged tree either way; that
is recorded as the one unverified corner rather than asserted clean.

**Decision 120 — R7: the four armed reds all passed and NONE is retired.** Matched by full title,
never line anchor (decision 100). B-27 passed in every run of the night across five legs and
seven-plus full suites; per decision 100 that retires nothing and no card claimed it fixed. The
single survivor — `tests/sync.spec.js:1343 › Convergence matrix (W-3) › yes/no answer converges
(live + catch-up)` — is a sixth distinct title in the rotating family recorded as B-45, failing at
14.8s in-suite and passing in isolation at 4.1s. It is **not** laundered as "not flaky". The night
produced the most controlled evidence yet for B-45's real mechanism: the same tree at the same
commit gave **24.5m / 1 failure quiet** versus **51.7m / 7 failures contended**, six of the seven
being 28–34s timeouts, with the contention self-inflicted by the orchestrator. What moves this
suite's distribution is CPU starvation, not test flakiness.

**Decision 121 — E-KR2 stands as MET, with a stated caveat, and the caveat is filed as B-58.**
Engineer-decidable, decided here rather than asked. A2's central finding is real and its fix is
real: the suite could not previously tell its own write policies from mutants (3 of 5 mutations
survived green, 2 of them mutations the file itself named as guarded), the root cause was every
write going out `Prefer: return=representation` so 0003's SELECT policy silently enforced the write
half, and triage confirmed the repair by execution — `rvPushRefused` now issues every refusal twice
(`return=representation` **and** `return=minimal`), and four separate WITH CHECK mutations in 0004
correctly RED the right subtests. **But adversarial mutation found one clause the suite still cannot
discriminate**: `submission_rejections_update`'s `USING`, where substituting `hq_can_see_field` for
`hq_can_approve_field` — the exact change `0004:483`, §5d(2) and `rowvisibility_rls_test.go:1922`
*all three* name as the guarded one — leaves all 54 subtests green. Not a live vulnerability (the
narrow `with check` still delivers the refusal), so the key result is not reopened; but the claim
"the suite can tell its own policies from mutants" does not hold universally, and the honest grade
carries that. Milestone close grades E-KR2 with this caveat visible, not silently green.

**Decision 122 — `docs/claude-md-night-crew` MERGED at triage, five commits and one week late.**
Operator's call, asked and answered. `CLAUDE.md` on the run branch still carried **15 GSD references
and zero night-crew ones**, including an instruction to route all repo edits through GSD commands
this project no longer uses — stale since 2026-07-26 on an abandoned branch. Merging it is what
makes the file describe the project that exists. The pairing is the point: card P1 spent the night
correcting this same file's deploy block for having been false for months, while an entire adjacent
section was stale for the same reason and nobody had looked. **A document nobody merges is
indistinguishable from a document nobody wrote.**

**Decision 123 — `backlog-round.html` is UNTRACKED, restoring decision 52; and the committed-stray
gap is now named.** Merging decision 122's branch moved the precache manifest **31 → 32 files,
2139.2 → 2358.8 KB** — **+219 KB onto every crew phone** — with `node build-sw.js` exiting **0 in
silence**. The file is the 225 KB static roadmap-round viewer that **decision 52 (T-22, FORK 2)
already disposed of** as the operator's own render, "never tracked in any branch", to be left
untracked and undeleted; that branch swept it into a `.planning/` cleanup and committed it.
Restored to exactly the state decision 52 specified — `git rm --cached`, kept on disk, `.gitignore`d
so it cannot recur — and the count is back to 31 with `sw.js` byte-identical. 🛑 **The mechanism is
the durable part: `committedOnlyTransform` reads git HEAD (B-37 / decision 67), so it excludes
untracked strays *by construction*, which is the whole protection decision 58 was written to give.
A stray that gets COMMITTED walks straight past it.** The guard is sound; its unstated precondition
is that nobody commits the file. This was caught only because a human had just ratified R2 and knew
the number was 31 — which is B-54's "enforced by nothing" demonstrated live within an hour of being
filed, and settles B-54's open (a)-or-(b) fork in favour of **(a), write the pin**.

**Decision 124 — the BACKLOG house style is KEPT; the validator is what should move.** Triage-
decidable, decided here. `night-crew backlog check` reports 249 issues across 119 entries, but
`dev`'s pre-run copy already failed 222 across 101 — tonight's 19 entries diverge at the same rate
as the previous 100, so this is a standing grammar mismatch, not rot any run introduced. The
validator wants a bare `new` / `promoted → <card>` / `dropped — reason` plus a separate `lead:`;
this file writes `· new · **destination: <where>** · lead: <line>`. The house style carries strictly
more information — a destination is a routing decision the validator has no field for, and
destinations are what make B-38's "every bullet names its destination" checkable at all. Do not
migrate 119 entries to satisfy a parser. Filed as **B-60** against the night-crew clone. Until it
lands, step 4.5 of `/nc-morning-triage` produces noise rather than signal on this repo — a check
that cannot fail usefully, which is this repo's own characteristic bug class one level up. One real
defect *was* found underneath the noise and fixed: B-49's entry had its renumbering narrative inside
the handle position, making it genuinely unparseable.

**Decision 125 — P2 and P3 go to Night B's slate, sized there, not promised here.** Engineer/PjM-
decidable, decided at triage. Neither was started (a budget decision by the control loop, not a
park). P3 `sync-banner-builder-tab-scope` is the smallest card on the slate (30–50m) and is the
natural first thing to add if Night B has room; P2 `workflow-unsubmit-failnote-reattach` (B-19) is
data loss in the accountability path and needs a back-and-reopen test in `tests/persistence.spec.js`
per CLAUDE.md's persistence rule. Their `BACKLOG.md` entries already read `promoted → P2
(slate-20260802)`, which is accurate: promoted, not started. Final sizing belongs to
`/nc-slate-plan` against S1/P4/P5/P6, not to triage.

---

## T-32 — Morning-triage resolutions (2026-08-02, `overnight-20260803`)

Night B of the two-night milestone close. 30 commits, 3 merges (3 clean, 0 conflicted hunks), 1 card
merged, 1 parked with evidence, 1 stretch card merged. Three operator forks were raised and all
three are resolved below. Gate evidence is from an adversarial subagent that re-executed every gate
in its own private worktree and ran eight feature-removal mutations, never from the closeout's own
lines (a card's own closeout is not evidence about that card).

**Decision 126 — F-1 RESOLVED: the cutover splits reads from writes. RxDB serves reads; HQ's REST
path keeps owning writes. P-KR3's parallel-run prohibition is WAIVED by the operator for this
shape.** Chosen over (i) making the Supabase substrate the truth source and (ii) building a
substrate→HQ propagation path. The card that forced this — S1b `sync-hard-cutover` — parked on a
finding that the cutover is not buildable as specified: RxDB replicates to a *second* Postgres, the
`0002` bridge runs HQ→substrate read-only and carries permissions rather than data, and nothing
carries a checklist row back. Retiring `/saveResponse` would therefore not have *moved* the write,
it would have **detached answers from submission** — a crew member fills a checklist, every answer
persists to IndexedDB and the substrate, and Submit produces an empty checklist because
`SUBMIT_CHECKLIST` builds from HQ's `submission_responses`. The card's own `done_when:` would have
passed while that happened, because it asserts a value survives back-to-list and reopen, which it
would, from local IndexedDB. This also reopens ledger decision 49, whose deciding argument —
*"RxDB replicates rows straight from Postgres and there is no API boundary left to translate at"* —
is false as built. (i) is the honest end state but is a milestone rather than a card, and it makes
HQ unable to read checklists at all whenever the substrate is down; (ii) looks incremental and is
not, since the same-transaction version is already proven impossible (`max_prepared_transactions` is
0 at both ends), leaving only an eventually-consistent design needing its own conflict rule —
decision 92's territory. The waiver is the operator's and is recorded as theirs: a build WO may not
propose this shape, and P-KR3 is otherwise unchanged. Offered back and captured as pending candidate
`architecture/C-3` (not adopted; nothing cites a candidate).

**Decision 127 — F-2 RESOLVED: nothing narrows. Both list tabs stay on REST, so a crew member keeps
seeing a colleague's completed checklist.** This resolves as a consequence of decision 126 rather
than as an independent call, which is why it was not captured as a separate preference. The fork
existed because HQ's REST list returns every submission since `current_date` for everyone — a
product rule recorded only as a code comment in `backend/internal/workflow/repository.go`
(`myChecklists`) — while the substrate's `checklist_submissions_select` is
`hq_can_see_template(template_id)`, so a replicating crew member would have seen submissions on
their own assigned templates only. Rejected: accepting the narrowing with a release note, and adding
a fifth read policy row (which would have needed decision 111 reopened). **B-61 closes with this** —
it was filed by S1a addressed to S1b, and the narrowing it warned about now never occurs. Noted for
the successor card: the team-wide visibility rule is still recorded nowhere but a comment, and
should be written down as a product decision before anything re-opens it.

**Decision 128 — F-3 RESOLVED: one combined notice to sales-processor, amending decision 106.**
Decision 106 had ruled two notices sent separately, the June drift first and alone; that ruling is
amended, not ignored. Chosen over holding to 106 and over sending P6's while retiring A1's. Three
things surfaced after 106 was taken: P6's audit covered every `:NN` row of both contract documents —
**111 rows, 45 wrong** — and only a minority *drifted*, with **22 menu-cogs rows never true at all**,
authored 2026-06-04 at 23:50 from a phase plan thirteen hours after the handler they describe landed
at 10:18; A1's own notice carries an error this audit found (`:31`/A10 attributes a timezone claim to
`/menu-cogs`, which contains no `AT TIME ZONE`), so sending it alone would propagate a fresh error
while apologising for old ones; and A1's notice **was never drafted** — the triage subagent searched
all of git history with `--diff-filter=A` and found exactly one notice file has ever existed, P6's.
A notice scoped to "one expression changed in June" would understate the problem by an order of
magnitude. 🛑 **Nothing has been sent.** The B3/B4/B6–B10 fix-forward corrections are required
before the draft goes anywhere near the counterparty, and the operator reads the corrected draft
before it is delivered. Two questions stay explicitly open and were not decided here: whether any
past `ready:false` run needs reconciling, and `menu_item_name` vs `name`, which only the
counterparty can answer. Offered back and captured as pending candidate `process/C-1`.

**Decision 129 — The triage preflight caught a night-crew binary 223 commits behind `main`, and two
ritual steps were nearly waived as "not deployed."** `night-crew skills preflight` did not exist on
the installed binary (v3.0.2+3, built from `258d723`), and `decisions ratify` and `preferences
ratchet` were absent with it. The tempting read — and the one a standing memory note actually
asserted — was that these verbs are undeployed and the affected steps must be reported as
unperformed. That was wrong: all three have been on `main` since v3.1.0, and `258d723` is an
ancestor of `main`. The clone at `/home/jcole/projects/night-crew-main` had already advanced; only
the binary was stale, because nobody re-ran install after moving it. `task nc:update` converged it
and the preflight then reported all ten declared verbs present. **The rule this establishes:**
a missing subcommand is a stale install until proven otherwise — check the verb against `main`'s
`cmd/nightcrew/main.go` before reporting it as undeployed, because the two cases have opposite fixes
and only one of them is a blocking finding. Had this gone the other way, §3b and §3c would have been
reported as unproven; run in fact, both queues were legitimately **empty** (nothing was decided under
a standing delegation this run), which is a real answer and not a skip.

**Decision 130 — Three closeout claims were falsified at triage; none is a defect in what shipped,
and the merge stands.** (a) S1b's cited evidence for "no production code" —
`git diff overnight-20260803 card/s1b-sync-hard-cutover -- . ':!.night-crew'` — returns 3 files at
HEAD, because P6 merged *after* S1b. The substance is true, verified at the correct base
(`3a71583^1`, genuinely empty, `/saveResponse` still mounted at `main.go:558`), but **the evidence as
written is unreproducible at triage time**, and these SHAs get cited. Merge-intents and closeouts
must cite a diff against a base that does not move underneath them. (b) `night-crew.toml:59-62`
asserts "re-verified at landing" that four tokens select exactly four spec files; measured at HEAD,
`sync` matches **six**. The direction is safe (over-inclusion runs more tests), but it is a false
factual claim inside a gate-configuration file, and a future editor trusting it could narrow the tag
into under-inclusion. Filed **B-78**. (c) The run's "green except the armed reds" was not the
stricter reading: on a **fresh** database the merged tree fails `[LST-17]` and passes
`onboarding:689`, exactly inverting the run's figure. **This is B-76 demonstrated by execution rather
than asserted** — see decision 131. What the triage *did* confirm, on stronger evidence than the run
produced, is the `onboarding:689` attribution: all four cells reproduced, with the fresh-DB legs run
as the **full suite** rather than the spec in isolation, closing the loophole that the isolation
result was itself an artifact.

**Decision 131 — B-76 is upgraded from a filed finding to a measured one, and it should not wait for
"next milestone" on its current framing.** The adversarial run produced three independent data points
that the full-suite figure is not a stable measurement: the merged tree scores **777/6/1 on a fresh
database against the base's 758/6/4** (so the merged tree is strictly *cleaner* than the base it came
from — which is the fact that justified merging); the base fails three `sw-api-cache-partition` tests
on a fresh DB that the merged tree passes; and the shared `hq_test_e2e` now reds **three** onboarding
tests rather than the one the run saw, so the pollution is actively growing between runs. The
uncomfortable half of B-76 — *a red can be an artifact, and so can a green* — is no longer a caution,
it is reproduced. This does not retract any gate in this milestone, and it does not make the merge
wrong; it means every full-suite figure in the milestone, including tonight's, is a measurement of a
tree **and** a database state. B-76 keeps its destination (test isolation / gate integrity, as one
sitting with B-50 and B-35) but is recorded here as the most load-bearing item in that group.

## T-33 — Mid-cycle OKR rewords, attended (2026-08-03 evening)

Taken at an attended planning session before the `20260804` slate, on the evidence in
`reference/okr-completion-plan-20260804.md`. The cycle's OKRs were hand-graded first — the
mechanical grader is blind in this repo (`okr grade` → *"no metrics.jsonl found under
.night-crew/runs"*, `milestone export` → *"no runs after 20260724"*, the same `<runid>` vs
`2026-08-03-autonomous` directory-shape defect as **B-33**/**B-77**), so every grade cites a file
and a line rather than HANDOFF prose.

**Standing before the rewords: 6 MET · 2 PARTIAL · 3 UNMET · 1 UNGRADEABLE.**

**Decision 132 — Five KRs are reworded mid-cycle; one is deliberately not, and grades red.**
Reworded: **D-KR2** (split a/b), **D-KR3**, **E-KR3**, **Q-KR2**, **Q-KR3**. The OKRs were authored
2026-07-25 against an architecture disproved 2026-08-03 — S1b parked on the finding that RxDB
replicates to a second Postgres with nothing carrying a row back, and decision **126** retired the
cutover in favour of reads-on-RxDB / writes-on-REST. Four of the five measured the retired shape or
named an artifact the process never built. The test applied: *a reword is honest when it changes
what is measured while preserving what is protected; it is laundering when it lowers the bar on the
thing the KR exists to guarantee.* Two rewords are **stricter** than what they replace — D-KR3 now
forbids the silent exclusions the original permitted, and D-KR2b names `/api/v1/health` as the one
shortcut that would silently defeat it. 🛑 **E-KR1 was NOT reworded.** Its subject did not move
under it: `sync.js` is still in the tree and both fetch-storm mechanisms are live at `:443-454` and
`:475-479`. It grades **NOT MET**, its two "superseded" backlog items (replay-fetch-storm, the
`sync.js` catch-up gate) are **un-dropped**, and the class carries to the next cycle. Rejected:
rewording E-KR1 to grade green (the laundering case), and backfilling `slate-20260801`/`-20260802`
to repair Q-KR3 (signed artifacts record what was believed at signing — the precedent is the
20260803 orchestrator reverting its own correction to `slate-20260803.md:331`).

**Decision 133 — D-KR2's evidence method changes from screenshots to an in-app version line
(operator's proposal).** A discreet version display, read from the **precached** `version.json` and
never from `/api/v1/health` — the server's value cannot be stale, so an API-fed badge would have
shown the correct version on a phone frozen on the old bundle and *hidden* T-21d rather than caught
it. Chosen over the 2/2 tab screenshots because the evidence is unambiguous (a string matches or it
does not), costs nothing to collect, is readable by any crew member, and — decisively — **removes
the N/A case**: a version line is a client-visible surface every cycle, where the screenshot form
was ungradeable in any cycle that shipped no visible tab. Most of the plumbing already exists;
`build-sw.js:291`'s own comment states the purpose (*"so the frontend can read its own version
without hitting the API"*) and no page displays it. Slated as card **A6**.

**Decision 134 — Build then close, not close first (operator).** The `20260804` night builds the
gate-integrity and live-defect cards; the milestone closes after, on figures the fixed gate
produced. The governing finding is **B-76's mechanism, measured at source this evening**:
`night-crew.toml:33-34` runs `npx playwright test` directly for both `suite` and `subset`, the only
`DROP DATABASE hq_test_e2e` lives in `Taskfile.yml:53-59` under `task test`, and
`playwright.config.js` has **no `globalSetup`** — so no night-crew gate leg has ever reset the e2e
database. Closing first would have cited this milestone's full-suite figures as evidence while
decision 131 had already reproduced their instability.

**Recorded, not decided: the offline-store inventory grew from the 2 classes E-KR3 assumed to 8
across 6 physical stores.** `hq_offline_v1` is two classes with different fates (`submitQueue`
survives an op-log retirement, `syncMeta` does not) and `api-cache` is two classes split by whether
RxDB replicates the underlying rows. One class has **no owner at all** — REST writes land in HQ's
Postgres, RxDB push (unconditional, `client.js:1194`) lands in the substrate, and nothing
reconciles them. Card **A4** publishes this; §8 of the plan records the operator's two-store target
architecture as a destination, which is decision 126 option (i) and milestone-sized, not slated.

---

## T-34 — Morning-triage resolutions (2026-08-03, `overnight-20260804`)

36 commits, 4 merges (4 clean, **0 conflicted hunks**), 4 of 4 slated cards merged, zero parks,
**zero open forks** — `DECISIONS-NEEDED.md`'s Open section was deliberately empty and that emptiness
is a result, not an unwritten section. Gate evidence below is from an adversarial subagent that
re-executed HQ's whole ladder in its own fresh clones and ran mutation probes against every guard
each card claimed; the closeout's own gate lines are cited nowhere (a card's own closeout is not
evidence about that card).

The night's four cards: **A1** `e2e-gate-database-isolation` (B-76), **A2**
`workflows-autosavefield-phantom` (B-65), **A4** `offline-ownership-design-note` (reworded E-KR3),
**A6** `app-version-badge` 🅢 (D-KR2b's evidence method).

**Decision 135 — `overnight-20260804` is merged on independently reproduced gates, over holding for
the one Playwright red.** G1 rc=0/rc=0; G2 (Go) **439 ran / 437 PASS / 0 FAIL / 2 SKIP** across 9
packages with `internal/workflow` running **35** tests and `TestRowVisibilityRLS` 59/59, reproduced
twice (subagent on the branch tip, and again by me on the merged tree, which is byte-identical); G4
31 precached / 0 outside / idempotent / three-way parity 1.4.0. **G2 (Playwright) did NOT reproduce
the closeout's `exit 0, 786 passed`** — an independent full run on the same tree gave **785 passed /
1 failed / 6 skipped, exit 1**, failing `[RUN-10] unsubmit returns checklist to editable draft`. It
is merged anyway because the red was **proven not to belong to this branch**: RUN-10 also goes flaky
on `dev` at `008e3ad` with a freshly created database (3 runs: pass, pass, flaky), and a control tree
— the branch with `dev`'s `workflows.html` — passed, while the only executable `workflows.html` diff
sits inside the fail-photo `.then()` and is unreachable from RUN-10. Rejected: holding the merge
until RUN-10's mechanism is diagnosed, which would delay four landed cards for a defect they did not
introduce; and a 3× full-suite re-run before merging, which measures the flake rather than the
branch. 🛑 **What this does mean is that the closeout's headline figure was a lucky draw**, and
RUN-10 was neither filed nor on the armed-red list, so a triage reader had no way to know a red there
was expected. Filed as **B-131**. A secondary result worth keeping: **B-93's own pass condition was
performed for the first time** — exactly one summary block in the complete 4716-line log, one
`Running N tests` line, highest index 793 = 792 + 1 retry — so the check the orchestrator confessed
it could not perform does in fact pass.

**Decision 136 — F3, the correction-photo coverage gap, is fixed attended at triage rather than
deferred; and its argument guard is asserted on the stored bundle, not on a rehydrated answer.**
Card A2 closed B-65's *naming* half and left its *coverage* half open:
`handleCorrectionPhotoCaptureClick` (`workflows.html:2129-2168`) is the byte-for-byte structural twin
of the chain B-65 broke, and **nothing executed it** — `[FLD-CORRECTION-PHOTO]` injects via
`POST /saveResponse` (the transport bypass FLD-16B's own header names as "the blind spot B-65 lived
in for months") and `tests/workflows.spec.js:684-689` *reimplements* the production write inside
`page.evaluate`, asserting against its own copy of the code. Proved before the fix: planting the
literal B-65 defect (`autoSaveField` at `:2154`) and running every photo/correction test in the suite
returned **9 passed, rc=0**. `[FLD-16C]` now drives presign → PUT → `debouncedSaveField` through
Playwright's filechooser and reds on both mutations (`ReferenceError: autoSaveField is not defined`;
`TypeError: Cannot read properties of undefined`). 🛑 **The guard is on the persisted bundle
(`{_v: null, _correction_photo: <url>}`) rather than on a checkbox state, and that was measured, not
chosen for elegance:** a first draft asserted the answer survived a back-and-reopen and failed,
because `hydrateFieldState:1809` does `delete FIELD_RESPONSES[rej.field_id]` on **every** hydrate of
a rejected submission ("uncheck a top-level field so crew must redo"). The test was wrong, not the
code — which is the whole reason CLAUDE.md's bug-fix protocol runs the test before believing it. No
production code changed. Chosen over filing it for the next night: this is the repo's characteristic
bug class (a test passing on a shape the app never produces), it was found by mutation rather than
inspection, and B-65 sat dormant for months exactly because nobody wrote this test.

**Decision 137 — B-89 and F2 ride the next night; neither is fixed attended.** **B-89** was confirmed
real by execution on a logged-in client with 11 grants — `index.html:241` writes `hq_apps` as
`{uid, apps}`, `sync-rxdb/bootstrap.js:62-71` `Array.isArray`-gates it, so `cachedGrantSlugs()`
returns `[]` and live `window.HQSync.surfaces === []` — but it is **latent, not live**:
`HQSync.surfaces` has no consumer and `startHQReplication` is never called from production code, so
the empty list decides nothing until the cutover card starts replication, at which point it would
scope replication to zero surfaces. **F2** is new and unfiled: `workflows.html:708` throws
`IndexSizeError` on every completed submission (the guard at `:706` tests `life<=0`, then `:707`
decrements, so `arc()` gets a negative radius), observed **28× in one suite run** and present on
`dev` too; the throw escapes the `requestAnimationFrame` callback so `canvas.remove()` never runs and
a frozen full-screen confetti spray stays painted over the app until reload, shipping an ERROR to the
server log each time. Both are next-night cards rather than attended work: F2 is cosmetic plus log
noise on a path with no data loss, and B-89 is a landmine to remove before the cutover, not a fire
today. Filed as **B-132** (F2). Rejected: fixing F2 attended for the sake of a one-line change —
attended minutes this morning are better spent on the two items only the operator can do.

**Decision 138 — B-26's remedy is a repo-local ladder file, not another inlining.** HQ now carries
`.night-crew/knowledge/reference/gate-ladder.md`, and slates and launch prompts for this repo cite
that path and nothing in the night-crew clone. `slate-20260804.md:217` and its launch prompt both
inherited G1–G6 from `reference/overnight-run-plan-20260707.md`, which **has never existed in this
repo**; the 20260804 orchestrator reconstructed the ladder by hand for the fourth time. Rejected:
inlining the ladder in every slate — that is precisely what regressed, since `slate-20260803.md` had
already done it correctly and `slate-20260804.md` reverted to the dangling pointer. A file in this
repo fails visibly at authoring time instead of silently at 3am. The new file also folds in the three
amendments this run earned: the whole-log capture rule (B-93), the `tests/`-anchored filter form
(B-87), and `TEST_DB_NAME` as load-bearing (B-80).

**Decision 139 — the per-leg isolation stanza becomes repo-local too, in the same file, and
`TEST_DB_NAME` joins it.** `launch-20260804.md` carried **no isolation stanza at all** — zero hits
for `TEST_PORT`, `TEST_DB_NAME`, `HQ_RLS_TEST_DB` or `unique` — while `launch-20260803.md:97-98` had
one, so every isolation value used that night came from the orchestrator ad hoc. This is one decision
with B-80, not two: since A1 landed, `webServer.command` DROPs the database it is pointed at as its
first act, so **an unqualified leg is destructive rather than merely noisy** — two legs differing only
in `TEST_PORT` now destroy each other mid-suite. Carrying the stanza in `gate-ladder.md` rather than
in the launch-prompt template is deliberate: the template lives in the night-crew clone and this repo
cannot fix it, whereas a repo-local file is inherited by every future slate for this target
regardless of what the clone's template does.

**Decision 140 — B-105 is answered: HQ keeps branch-and-commit and does not adopt OpenSpec.** The
question had been open and deliberately untouched by every card. Decided rather than carried: the
night-crew slate mechanism already supplies change-level ceremony (a signed slate entry, a
merge-intent note with three durable fields, a conflict-log entry per merge, and an adversarial G6),
and OpenSpec would duplicate that at the cost of a second planning state to reconcile — the exact
thing `.night-crew/knowledge/` exists to be the single copy of. `openspec: absent` is already
re-confirmed at every launch and G3 is N/A on that basis, so this decision changes nothing
operationally and simply stops the question being re-asked. Reversible at any milestone boundary;
what would change the answer is a second target repo needing the same discipline, or a contract
consumed externally enough to want spec deltas of its own.

**Decision 141 — `BACKLOG.md` is NOT mass-rewritten to satisfy `backlog check`, and the verb is
advisory for this file.** `night-crew backlog check --file BACKLOG.md` exits **1** with **290 issues
across 156 entries** (85 unrecognized status, 74 missing lead, 63 missing handle, 63 missing
description). The divergence is structural and predates this run: HQ's convention carries a
`destination: …` field the validator does not know, and 63 of the entries predate handles entirely.
Rewriting 156 historical entries would rewrite the record to satisfy a checker, which is the wrong
direction — the entries are evidence. **New** entries follow the validated shape
(`- **B-NN · Title** — desc · _origin_ · status · lead: …`) so the file converges forward. 🛑 Stated
plainly because triage §4.5 mandates this gate pass before committing and **it does not pass**: this
is a known, named divergence, not a green gate. Filed as **B-133**.

**Operator rider, recorded as a standing rule rather than a one-off choice: _"agents decide
implementation details."_** Given at triage 2026-08-03 in answer to the process-defect question,
which had offered B-26, B-80, B-105 and the backlog divergence for the operator to pick among.
Decisions 138–141 were therefore taken at role level and stated rather than escalated. This extends
the existing standing rule (PM/PjM/Engineer-level calls get decided, not handed up) from *planning*
questions to *mechanism* questions, including ones a skill's own text routes to the operator. It does
not extend to genuine product forks. Offered back as a preference candidate at triage; adoption is
the operator's and nothing cites a candidate.

---

## T-35 — Consumer reply to the sales-processor notice (2026-08-04)

The sales-processor maintainer replied to the combined notice, answering all four sections. Three
answers change HQ's obligations and one closes a question that had been open since 2026-06-06.

⚠️ **Provenance for everything below: the consumer's reported reply.** That repo is not present in
this tree, so HQ cannot verify any of it at source — unlike every verdict in the two contract
audits, which were checked against code. Recorded as reported-and-credible, not measured.

**Decision 142 — §1 is closed with NO restatement of any past figure, on the consumer's evidence,
with one gap named rather than papered over.** The consumer's gate hard-fails before writing a
report or dispatching a transfer, so a spurious `ready:false` would appear as a **missing week on
disk** — and it reports an unbroken weekly run, reports and transfer ledgers both, for every period
**2026-05-31 → 2026-07-19**, no gaps. Its gate went live **2026-06-05**, one day before the two
2026-06-06 changes, so that window is its entire exposure and every run inside it completed. The
opposite direction — a spurious `ready:true` letting a period through — it cannot see, but that is
bounded to the "case 1" bucket (COGS-category, receipt attached, parser failed) and those periods
were acted on weeks ago; the consumer judges it immaterial and asks for no restatement. HQ accepts.
🛑 **The gap, found by HQ reading the reply rather than by the consumer:** the evidence window ends
**2026-07-19 and today is 2026-08-04** — sixteen days and **two weekly periods (ending 07-26 and
08-02) are unaccounted for.** By the reply's own detection method those are precisely where a
spurious block would sit undetected. Filed **B-138**; §1 is closed on the evidence offered, not on
the two weeks nobody has looked at.

**Decision 143 — assumption A5's "COORDINATED TWO-REPO RELEASE — sales-processor must make the
matching change" is RETIRED as false, and the HQ timezone deploy is NOT blocked by the consumer.**
The consumer reports **no timezone code whatsoever**: it sends `from`/`to` as bare calendar dates and
its Mercury gap check only diffs transaction IDs. Its entire "matching change" is a find-replace in
its own handoff documents, which it will do **after** migration `0072` deploys, deliberately not
before. So the dependency A5 asserted does not exist, and it had been gating an HQ deploy on nothing
— including in the notice HQ drafted the day before, whose §4 warned the consumer not to ship early.
That warning was harmless but unnecessary. **What HQ still owes is the changeover date**, which is
the only thing the consumer says it is waiting on. Chosen over keeping A5 as a precaution: a stated
dependency that does not exist is worse than none, because the next person to read it defers a deploy
for it — which is exactly what happened here.

**Decision 144 — `tracked_bank_tx_ids` must stay UNFILTERED, and narrowing it counts as a breaking
change even though it only removes rows.** The consumer decodes it and its Mercury↔HQ gap check
depends on the list being unfiltered by category: that is what lets the check surface miscategorised
and uncategorised transactions, which are invisible to every COGS field on the endpoint. It is the
consumer's stated backstop for exactly those rows. Adding a category filter would not break the
decode — it would **silently blind the check**, which is the worse failure and the harder one to
notice. Recorded on the field's own row in the contract, not only here, because the row is where
someone about to "tidy" the query will be looking. The consumer asked to be told if it ever changes.

**Decision 145 — `/menu-cogs` keeps sending `menu_item_name`; the wire is NOT changed to `name`, and
`menu` is NOT added.** The consumer has **no `/menu-cogs` client at all** — no code, no contract
document on its side — and stated it is indifferent. Decided at role level rather than escalated,
under `delegation/P-1`. Three reasons: the corrected document now **matches the shipped code**, so
changing the wire would move the code to match a document that has already been fixed, inverting the
audit's own governing principle ("we changed no code; every correction moved the documents to match
the shipped behaviour"); there is no client to please in either direction; and a future consumer will
build from the corrected document **plus** the observed wire, which now agree, so the cheapest
correct state is the one that already exists. `menu` stays unimplemented for the same reason —
purely additive, breaks nothing, and serves nobody. 🛑 **The wider finding is that the whole
menu-cogs contract — 64 rows audited, 27 wrong, 22 never true from the day it was written — never
had a consumer.** The document drifted from the code and the code answered nobody. Recorded at the
top of that document so the next reader does not mistake live surface for load-bearing surface.

---

## T-36 — §1 closed on the extended window (2026-08-04)

The consumer answered HQ's B-138 challenge by extending its own check. Two decisions: one closes
the question §1 opened on 2026-06-06, the other opens a new one on HQ's side that the exchange
exposed.

**Decision 146 — B-138 is RESOLVED and §1 is CLOSED with no restatement, on evidence that now covers
the full window.** The two periods B-138 named (ending **2026-07-26** and **2026-08-02**) have **no
run at any pipeline stage** — classify, PDF, CSV and transfer ledger all stop at 07-19. Nothing was
blocked because **nothing ran**: both were operator skips, the business was closed. HQ's own
`/period-summary` independently returns `ready: true`, zero pending, zero unlinked for both periods,
which is consistent. Combined with the first reply's unbroken 2026-05-31 → 2026-07-19 run, there is
no spurious block anywhere in the exposure window, and the consumer asks for no restatement. HQ
accepts and closes. 🛑 **B-138 was worth raising even though it found nothing**: the answer was
sixteen days of unexamined data, and the *reason* it found nothing — the weeks were skips, not blocks
— is what surfaced decision 147.

**Decision 147 — the "unbroken weekly run ⇒ no blocks" detection method was only CONDITIONALLY
valid, and HQ owns the half nobody had raised. Filed B-139.** The consumer volunteered the flaw
rather than being caught by it: *"a week we skip is byte-for-byte indistinguishable on disk from a
week your gate blocks — both leave no report and no ledger."* The method therefore only holds if
every week actually runs. It **does** hold for 2026-05-31 → 2026-07-19, but only because the first
reply asserts every period in that window is *present* — not merely that present ones completed;
that stronger claim is what carries the conclusion, and the weaker phrasing of the second reply
("every week that ran in that window completed") would not have. The consumer is changing its
process to run payroll even on closed weeks. 🛑 **HQ's half:** `/period-summary` logs ten
`slog.Error` calls and **not one line for a successful response** — no record that a request
arrived, for which period, or what `ready` came back as, and no access log on the service-token
routes either. So the only record of a blocked payroll week is an absence of a file on someone
else's disk. HQ spent a triage morning and two rounds of correspondence reconstructing from the
consumer's filesystem what one log line would have stated outright. This is the **B-81 / B-82 /
B-86 / B-93** cluster — "a check cannot tell you what it actually did" — reaching the one endpoint
whose answer another system's payroll depends on. Chosen over treating the consumer's process change
as sufficient: that fix is discipline and depends on someone remembering, whereas a log line is
mechanism and does not. Both are worth having; only one of them is HQ's to do.

**Unchanged by this round:** §2, §3 and §4 of the reply repeat the previous round verbatim and are
already recorded at **T-35 decisions 143–145** (A5's coordinated-release requirement retired,
`tracked_bank_tx_ids` must stay unfiltered, `/menu-cogs` keeps `menu_item_name`). 🛑 **The one thing
HQ still owes: the migration `0072` changeover date**, which the consumer has now named twice as the
only item it is waiting on. It is unblocked — see T-35 decision 143 — and gated only on HQ promoting
`dev` to `main` and deploying.

---

## T-37 — Milestone close: "Sync foundation" (2026-08-05, attended `/nc-milestone-close`)

The cycle opened 2026-07-24/25 and closed 2026-08-05 across 9 runs (`20260725` → `20260804`),
28 landed cards, 99 decisions (49–147), 0 open forks. Full record:
`reference/cycle-closeout-20260805.md`. Aggregate transfer:
`<night-crew clone>/reference/milestones/hq-20260805.md`.

**Decision 148 — the milestone closes graded 8 MET · 1 PARTIAL · 3 NOT MET · 1 UNAUDITABLE
(N=13), and the headline is that it did not deliver what it is named for.** Verified at source
during the close, not carried from any report: `createHQSyncDatabase()` and `startHQReplication()`
appear in production code in exactly one file — `sync-rxdb/bootstrap.js`, as an import (`:49-50`)
and two deferred re-exports (`:83`, `:90`) — and **neither is ever called**; `window.HQSync.db` is
never assigned, so `workflows.html:3590` is dead by construction; the 495 KB
`vendor/rxdb.bundle.js` is precached to every crew phone and does nothing. `ledger.md:2663` had
already recorded the same fact in passing. **Nine runs produced a tested library with zero call
sites**, and every one of the 13 KRs is still correctly graded — four objectives assert delivery
and not one KR measures whether RxDB serves a byte. Chosen over grading the cycle on its KR
arithmetic alone: a close whose scoreboard is honest and whose subject is undelivered must say
both, or the next roadmap round inherits a success. 🛑 **What is real and must not be rebuilt:**
the substrate schema/RLS/write policies (59 `TestRowVisibilityRLS` subtests), RxDB database
creation and the conflict handler proven in a real browser, the JWT bridge, and the
`HQ_SYNC_REST_URL` interlock that kept it all inert as designed.

**Decision 149 — D-KR2a and D-KR2b grade NOT MET by deferral, not N/A, and no later deploy
repairs them.** The operator elected at the close to skip `task prod:deploy` → `task version` →
the returning-client version read, on the stated ground that *"there is nothing new feature-wise
shipping to prod."* Recorded with the correction attached, because the record must not harden a
partial truth: `dev` is **436 commits ahead of `main`** (`32afb39`, 2026-07-24, backend 0.2.2) and
carries backend **0.3.0** / frontend **1.4.0**, migration `0072_app_timezone_new_york.sql`, and
A6's user-visible version badge. No new *tab* ships; it is not true that nothing ships. Graded as
a miss rather than N/A because the precondition **did** fire when A6 landed — the property became
measurable and was not measured — which distinguishes it from the QA-KR4 N/A precedent
(`ledger.md:510`, *"no schema migration shipped"*), where here a migration is precisely what
waits. Per the T-33 amendment's own rule, backfilling a grade after the boundary is prohibited:
both stay NOT MET permanently and a later deploy becomes the *next* cycle's evidence. **Still
owed: the `0072` changeover date** (T-35 decision 143, restated T-36) — this close does not
discharge it.

**Decision 150 — E-KR1 NOT MET stands un-reworded, and Q-KR2 / Q-KR3 close unrepaired.** E-KR1's
subject did not move: `sync.js` is in the tree with both fetch-storm mechanisms live at
`:443-454` and `:475-479`; its two backlog items stay un-dropped and the class carries forward.
Q-KR2 grades **UNAUDITABLE** (it named a `## Red-first` field that has never existed in the
merge-intent format) and Q-KR3 **PARTIAL permanently** (absent from `slate-20260801.md`, partial
in `slate-20260802.md`; signed slates are not backfilled). Five armed reds — **B-27** plus three,
plus **B-131** — carry out of the cycle with **none retired**, per decision 100 and T-31 decision
120: an armed red is retired by diagnosis, never by passing once. **No suite figure is citable at
close**: the `20260804` closeout's `786 passed / exit 0` did not reproduce (`785 passed / 1
failed`, B-131).

**Decision 151 — the close was graded BY HAND because every night-crew milestone verb is blind
here, and the marker's missing `last_run` was left visible rather than patched.** `okr grade`,
`okr audit` and `milestone export` all exit 1 (`no metrics.jsonl found under .night-crew/runs`);
`scorecard` reports `No runs to show.`, which the ritual's step-0 precondition reads as a **pass**
— an emptiness check that cannot tell "all clear" from "I can see nothing". `milestone mark`
exited 0 but wrote the marker with **no `last_run` field**, where `hq-20260724` carries one.
Deliberately not hand-patched to `20260804`: a hand-written value would be indistinguishable to
the next reader from one the tool established, which is the exact failure this close documents.
Binary was **v3.2.0+6** (`f31fff2`, main) and was **not** rebuilt from the dev clone for the
ritual. Filed in the clone as **B-346**.

**Decision 152 — the previous close's transfer cited findings it never captured; corrected in
place and filed.** `hq-20260724.md` claims the CLI blindness was *"filed as B-105"* and that three
tool-implicating findings (B-105 · B-106 · B-107) were transferred. All three are pre-existing
clone entries with unrelated origins — B-105 is the OpenSpec-discipline question — and
`grep 'milestone-close hq'` over the clone's `BACKLOG.md` returns nothing. **The finding was never
filed, which is why it recurred at this close.** 🛑 Note the numbering trap: the clone's B-105 is
not hq's B-105 (per-change discipline) — two namespaces, same number. Filed in the clone as
**B-347**. Both offers were made per-item and approved by the operator.

**Carried into the next cycle:** the deploy + `0072` date · E-KR1's two un-dropped items ·
**B-89** (`cachedGrantSlugs()` returns `[]` unconditionally) and **B-132**, routed to "the next
night" by T-34 decision 137 but never promoted to a card · **B-139** with the B-81/B-82/B-86/B-93
cluster · five armed reds and the armed attended `task sandbox:e2e` flag · 90 open `· new`
backlog items · and the successor the retirement of `sync-hard-cutover` left unauthored —
`sync-live-fill-view`, whose absence is why this milestone reported zero white cards (clone
**B-340**/**B-341**; the operator's response filed as **B-344**/**B-345**).

## T-38 — Morning-triage resolutions (2026-08-05, run `20260806`)

Triage of `overnight-20260806`, attended. The review verdict was sourced from an adversarial
re-execution, not the closeout's own gate lines (§15ag.87): a fresh subagent rebuilt the branch
tree in its own worktree and ran the first full Go suite the final tree had ever had — green, 9
packages, 455 tests, 0 FAIL, `internal/workflow` at 35/35 (not the silent-skip zero) — plus G4
byte-idempotent at 31 precached, and direct attacks on A1's gate-child token (the old skip env now
FAILs in 0.008s), A2's Check B (a silenced package turns it red, reproduced both directions), W0's
NUL removal (fingerprint equivalence independently re-derived), and A4's ladder edits (the fixed
repro command now includes the falsifying test; run verbatim). Every claim attacked, held. One
residual named at the merge decision: no full Playwright suite has run on the final tree.
Merged to `dev` as **`ff1f39a`** on the operator's answer; full Go suite re-run green on the merged
tree; `hq_test_go` (corrupted: goose 73 applied, 72 absent; zero connections) dropped per the
HANDOFF's documented-safe call and recreated green by `task test:go`. Three fork answers were
offered back and all three captured as pending preference candidates on the operator's explicit
yes: **gates/C-1**, **operations/C-1**, **operations/C-2** (pending, not adopted — B-245's lesson
stands until `night-crew preferences adopt` is run).

**Decision 153 — D-1: red-first becomes the named gate RF; G3 stays `N/A`.** The record defined G3
two incompatible ways — `N/A — openspec: absent` (decision 140) and "red-first re-verified by G6"
(decision 101's recovered contract) — and A4's completeness sentence could not be true under both.
Chosen: option (c), a new named gate row **RF (Red-first)** in `reference/gate-ladder.md`, over (a)
redefining G3 as red-first (rejected: the recycled number is what produced the contradiction) and
(b) leaving red-first ungated under an honest `N/A` (rejected: three of four code-changing cards
on run `20260806` forgot the obligation precisely because no gate stood behind it). Decision 140
stands untouched; decision 101's G3 reading is retired with its substance preserved as RF. The
completeness sentence now enumerates G1, G2 (Go), G2 (Playwright), G3, G4, RF, G6. Every future
slate and launch prompt inherits RF as a gate a code-changing card cannot merge without.

**Decision 154 — D-2a: production gets both a nightly dump and PITR.** B-143 (no backup of any
kind) is what converted the 2026-08-06 incident from a bad night into an unrecoverable one: the
image had a rollback tag and the data had nothing. Chosen: nightly `pg_dump` of `yumyums` to a
path outside the Docker volume (Taskfile target + cron line) **and** `archive_mode=on` with a
local WAL archive for point-in-time recovery, over dump-only (the floor alone leaves up to a day
exposed), PITR-only (longer exposed while it is built), and deferral. The dump is the immediate
build; PITR follows. Promoted to card `prod-backup-floor-and-pitr` (roadmap Activity 0). B-143
marked promoted.

**Decision 155 — D-2b: the test suites leave the production cluster.** B-141's mechanism — a test
file holding admin credentials to the cluster that serves `hq.yumyums.kitchen`, with a blocklist
guard that structurally cannot enumerate the names that matter — means any mistake in that file is
a production mistake. Chosen: a separate test-only Postgres container, over a restricted test role
on the shared cluster (cheaper, but production would still share disk and cluster failures with
tests) and over relying on the prefix-guard fix plus backups alone (a guard is a correctness
argument; separation is a structural impossibility argument, and this week demonstrated which one
holds under adversarial review). Promoted to card `test-cluster-separation` (roadmap Activity 0).
B-141's prefix-guard half and B-142 ride the A3 re-gate (`gate-rls-fixture-ownership`), which
stays refused-and-preserved until that attended card runs.

## T-39 — B-145 recovery Phase 3 rulings (2026-08-06, attended, per handoff `reference/handoff-prod-data-recovery-20260806.md`)

Phases 0–2 measured first (all read-only): prod's `production` schema holds crew re-entry from
08-06 that must be preserved (1 admin user `jamal@yumyums.kitchen` with a live session, 1
checklist template, 1 onboarding template, 6 vendors, 108 catalog items, 10 groups, 19 tags, 5
pending purchases auto-backfilled by the worker's default 14-day lookback, 2 confirmed purchase
events); Mercury retains full history to 2023-02-24; Spaces `receipts/` holds 76 files
(2026-04-22 → 08-06, survived the drop); Spaces `toast/` holds 120 date-dirs (2026-03-05 →
07-24, missing days are closed Mon/Tue only); Toast SFTP retains ~27 days (2026-07-10 → 08-05)
— Toast coverage gapless 03-05 → 08-05. New defect filed during Phase 0: **B-146** (prod's
Toast sync silently dead since the 07-28 image rebuild; SFTP key never ships). The backup floor
(decision 154's immediate half) was built and proven BEFORE these rulings executed: `task
prod:backup` (dump + globals, keep 14, small-dump guard) on branch `fix/b145-prod-backup-floor`,
restore drill green against a scratch container (96 tables, counts verified), Windows scheduled
task `YumyumsProdBackup` proven by an observed firing at 08:29:01 then moved to nightly 03:30.

- **Decision 156 — Mercury/receipt backfill horizon: 2026-03-01.** Operator ruling, chosen from
  measured options. Covers the season, the current payroll period, and matches Toast coverage;
  estimated ~80–120 receipts re-enter the pending review queue, the crew's work over days.
  Items/vendors ride this ruling (catalog rebuilds through receipt review by design).
- **Decision 157 — Write-offs: submissions + responses, stock count overrides, sessions, and
  everything outside the inventory app.** Operator ruling, verbatim scope: "everything outside
  of inventory app". No recovery effort for workflow/onboarding history or templates — the crew
  re-authors templates as needed (one checklist + one onboarding template already re-entered).
  Recipes (`usage_pct`) are inventory and stay a hand-rebuild in the Recipes tab; COGS
  attribution is wrong until done.
- **Decision 158 — Sales-processor notice: held by the operator.** Ruling: no draft; the
  operator decides the message and its timing themselves. Candidate preference process/C-1 (one
  complete correction) was surfaced at the asking.
- **Decision 159 — PITR: enable now, attended.** Second half of decision 154. archive_mode on
  the :5433 cluster with the WAL archive outside the pgdata volume, done with the operator
  present since it needs a brief cluster restart.

### T-39 execution outcome (same session, 2026-08-06)

Every ruling executed or explicitly handed off; prod verified healthy after each step.

- **Decision 156 (backfill to 2026-03-01): done, with a defect found and fixed en route.** The
  first attempt (prod restarted with `MERCURY_LOOKBACK_DAYS=160`) died on *"offset exceeded
  50000 — bailing"*: Mercury's `/api/v1/transactions` silently IGNORES `offset` (verified live:
  offsets 0 and 500 return the identical page) — its real contract is cursor pagination via
  `start_after`. Ordinary 14-day windows fit one page, so production never saw it.
  `internal/receipt/mercury.go` now cursors with `start_after` + `order=asc` and fails fast if
  a full page's cursor does not advance; 6 package tests (DB-free, httptest) pass, including a
  regression test reproducing the same-page-forever shape, and the cursor walk was verified
  against the live API (722 raw transactions in the window, page 2 continues with zero
  overlap). Because prod builds from main and promoting dev→main mid-recovery is /nc-release's
  attended act, the backfill ran LOCALLY from the fixed tree via the new one-shot
  `cmd/backfill-receipts` (one ingest cycle, an explicit lookback, NO migrate/seed — safe to
  point at prod from a tree whose migrations are ahead). Result: 149 processed, 58
  auto-created purchase events, 84 pending review, 7 cached, 0 errors. Prod's env was reverted
  to the default 14-day lookback and its worker verified green (cycle complete, 7 cached).
- **Toast (rides 156's class table): done.** SFTP tail 20260725–20260805 pulled and seeded to
  Spaces (`migrate-toast-archive`, 12 uploads), then `sync-toast --from 2026-03-05 --to
  2026-08-05` upserted 1 052 sales rows / 34 menu items across 80 business days
  (2026-03-05→08-02; the 08-03/04/05 exports are header-only). Skipped days are the known
  closed Mon/Tue set.
- **Decision 157 (write-offs): recorded.** No recovery effort spent outside the inventory app.
- **Decision 158 (sales-processor): held by the operator; nothing drafted.**
- **Decision 159 (PITR): done.** `archive_mode=on`, WAL to
  `/mnt/c/Users/jcole/yumyums-backups/wal` (bind mount, outside the pgdata volume),
  `archive_timeout=300`; first segment archived, `pg_stat_archiver` 1/0. `task prod:backup`
  extended with nightly `pg_basebackup -X none` + WAL pruning; 212 MB base on disk.
- **Final counts** (`production` schema): pending_purchases 89 (event dates 02-27→08-05),
  purchase_events 60 (02-28→07-29), purchase_items 263, vendors 20, menu_items 34,
  daily_menu_sales 1 052, users 1, recipes 0.
- **Open, and whose:** crew reviews the 84-receipt queue (days); operator re-invites crew and
  hand-rebuilds recipes (COGS wrong until done); operator decides the sales-processor message;
  B-146 (ship the Toast SFTP key to prod) is cardable; the branch
  `fix/b145-prod-backup-floor` de-confines under night-crew.toml (touches
  `internal/receipt` + Taskfile) so the full Playwright suite gates its merge to dev. The full
  GO suite was deliberately NOT run this session: dev's tree still carries the pre-A3 RLS
  test default (B-141/B-142 unremediated until the attended re-gate), and running it against
  the shared cluster is the incident's own vector. Go-side evidence is the receipt package's
  6/6 (DB-free).

### T-39 addendum — de-confinement gate verdict (2026-08-06, after the outcome note)

The full Playwright suite ran green on the recovery tree: **791 passed, 6 skipped, exit 0**
(28.3 min). A concurrent slate-planning session had already fast-forwarded `dev` through the
recovery branch (slate-20260807 cites the Mercury fix as landed, `4efd265`), so the gate
verdict applies to the `dev` tip itself; no separate merge was needed. `dev` is NOT pushed —
left to the operator alongside the signed slate.

## T-40 — Morning-triage resolutions (2026-08-06, run `20260807`)

Triage of `overnight-20260807` — 3 of 3 cards merged by the run (W0 `test-cluster-separation`,
A2 `shipped-bug-sweep`, S `spike-b-migration-rehearsal`), zero parks, zero conflicted merges,
zero open forks. Merged to `dev` as a `--no-ff` merge after an adversarial subagent
independently reproduced every gate on the final tree in a fresh scratch clone: build+vet
clean; Go 9 packages green with real counts (workflow=35 fresh-run) on :5434; G4 idempotent,
31 precached, 1.4.0 three-way parity; B-89 and B-132 both **mutation-verified** (reverting
each fix turns its shipped test red — B-132's mutation reproduced the exact original
`IndexSizeError` at workflows.html:711); the §1 mis-merge recovery verified (`3820cc9`
reachable from zero refs, the attended commit `eb8e415` untouched, all three merges true
no-ff with the claimed card tips as second parents). The spike's runtime "48 PASS / GREEN"
figure is recorded **unverified by triage** (re-execution needs the substrate stack; its
structure — verdict = exit status, fail-throws — was verified statically, and the run's G6
executed it twice). Go suite re-run green on the merged `dev` tree. Conflict-log audit clean:
3/3 merges logged including clean ones, every entry names its intents, all three
merge-intents carry the three required fields.

- **Decision 160 — Overnight runs git-operate only from a dedicated run worktree, never the
  main checkout.** Operator ruling at triage, chosen over "leave as a one-off incident note",
  after run 20260807's first W0 merge landed on `dev` because a concurrent attended session
  had switched the main checkout mid-card (conflict log §1; recovered in full). Recorded as
  **pending candidate `operations/C-3`** with the operator's explicit consent to the exact
  text — it binds nothing until adopted (`night-crew preferences adopt operations/C-3`,
  the operator's own act at a terminal).

Triage dispositions decided at role level (stated, not asked, per standing practice):

- **Follow-ups graduated to BACKLOG.md as B-147–B-155** (9 entries, validator-clean at the
  pre-merge noise floor of 295). The two 🛑 pre-next-run items: **B-147** (the permanent
  B-132 test rewrites this run's committed evidence — screenshot hardcoded into the run's
  a2-logs dir) and **B-148** (a red spike-B run contaminates the shared substrate and the
  printed recovery command is inoperable). **B-149** is new from the triage adversarial
  review: the uid-mismatch half of the B-89 fix is implemented but unguarded — mutation
  removing just that check leaves the shipped test green.
- **`card/d1-syncspec-deflake` cut** under the standing net-zero rule: fix `4ab162c` + its
  revert `6ee45e0`, `git diff --stat` empty against parent `c1a2393` which is in `dev`; the
  residual's destination was already recorded at a roadmap round (superseded by the
  RxDB/Supabase migration). The three fully-merged card branches (`card/w0-…`, `card/a2-…`,
  `card/s-…`) deleted as merged housekeeping; `overnight-20260807` stays local per the
  branch model.
- **Date-label note:** the run executed **2026-08-06** (commit timestamps 09:56–12:41 EDT;
  wall clock confirmed at triage) — its artifacts stamp "2026-08-07" from the run id. Run
  artifacts left as written; this line is the correction of record.
- **Zero gray areas were routed through `decisions log` by the run** and the ratification
  queue read empty at triage — consistent with DECISIONS-NEEDED's own record that nothing
  reached the routing threshold; nothing was skipped. The one triage fork (decision 160) was
  routed through the resolver at triage.

## T-41 — Morning-triage resolutions (2026-08-07, run `20260807-2`)

Triage of `overnight-20260807-2` — 2 of 2 cards merged by the run (C `spike-c-round-trip`
@ `76801aa`, D `spike-d-realtime-live` @ `7101b1c`), zero parks, zero conflicted merges, zero
open forks. **All four D-KR1 spike verdicts are now recorded — A, B, C, D all GREEN — and the
Activity 3–5 build gate is OPEN**; cutting the build cards is the next slate's act. Merged to
`dev` as a `--no-ff` merge after an adversarial subagent independently reproduced every gate on
the final tree: G1 build+vet clean; G2 Go 9 packages, 456 RUN / 0 FAIL, workflow=35, on :5434;
G4 exit 0, precache 31, committed `sw.js` byte-identical to regenerated, 1.4.0 parity;
discipline-grep vacuity confirmed by execution (B-14); `tests/sync-rxdb-client.spec.js`
re-executed 55/55. **Both spike verdicts were re-executed LIVE by the review** — C green exit 0
(round trip 136 ms) and red `--no-relay` exit 1; D green exit 0 (all three clause shapes
suppressed) and red `--no-filter` exit 1 — with the substrate restore byte-verified on all four
runs. Go suite re-run green on the merged `dev` tree (456 RUN / 0 FAIL, workflow=35 fresh).
Conflict-log audit clean: 2/2 merges logged including clean ones, every entry names its
intents, both merge-intents carry the three required fields. Post-merge worktree sweep clean
for the run's cards; only `card/a3-rls-fixture-own` remains, preserved by decision 155.

Evidence corrections of record (documentation-level; no gate implicated):

- **Spike C's exit contract is 0/1/2/3/64, not 0/1/2/64 as the closeout states** — exit 3 =
  verdict reached but the shared substrate could not be restored (`spike-c-roundtrip.sh:27-31`).
- **`--retries=0` in the two committed Playwright logs is asserted, not evidenced** — no
  command line in the log; corroborated only by zero retry markers and exact 798-count
  arithmetic. Recorded as unverified. Feeds B-155's attestation-header convention.
- **HANDOFF's per-card actuals column for C ("~2h05m wall") exceeds the 1h58m run window** —
  `timings.log` is the measured record (C ≈59m, D ≈59m end-to-end); card-actuals appended
  from it, not from HANDOFF prose.

Triage dispositions decided at role level (stated, not asked, per standing practice):

- **FR-11 filed as B-156, a candidate new named flake** — distinct 1.5 s assertion-mismatch
  shape, not B-32's 30 s-timeout family; on no armed list; neither card could have caused it
  (zero frontend/spec files in either diff, verified by execution). Armed-expected next slate;
  retirement only by diagnosis (decision 100).
- **LST-17 STAYS ARMED.** It passed in BOTH full suites tonight — recorded as evidence, not
  retirement: decision 100 / T-31 decision 120 bind (retired by diagnosis, never by passing),
  and no diagnosis links B-147/B-148's fixes (spike-B harness + screenshot path) to LST-17's
  mechanism in `sync.spec.js`.
- **B-62 CLOSED — answered GREEN by card D**, disposition recorded in the entry; its stated
  destination (`sync-hard-cutover`) was pessimistic — the property was testable against the
  spike stack now, interlock armed throughout.
- **Follow-ups graduated to BACKLOG.md as B-157–B-160** (relay double-fire per `/saveResponse`;
  spike-script hardening one-liners; `--fresh-substrate` destroy footgun; `app_slug` constant /
  no template→app association — spike B's finding #1 resurfaced). Validator at the standing
  noise floor of 295, zero findings against the new entries.
- **Harness lesson carried in the record:** `( … ) &` makes `$!` the subshell pid — teardown
  kills nothing, an orphaned server holds the port, and the next run's health poll goes green
  against a foreign DB. Fixed both ways in C's harness (env-exec + occupied-port refusal);
  any future card spawning a server from a shell copies both halves. C's evidence-hygiene
  gap (env attestation only in a commit message; zero-byte bddgen log) was established as
  benign by D's implementer and needs no re-gate.
- **`card/c-spike-c-round-trip` and `card/d-spike-d-realtime-live` deleted as merged
  housekeeping** (worktrees removed, branches ancestry-merged via the run branch);
  `overnight-20260807-2` stays local per the branch model.
- **Zero gray areas were routed through `decisions log` by the run**, the ratification queue
  read empty, and the ratchet reported nothing survived — all three consistent with
  DECISIONS-NEEDED's own record that nothing reached the routing threshold and nothing was
  decided under delegation; nothing was skipped. Triage itself resolved no operator fork (the
  merge is the ritual's standard act), so nothing was routed at triage either.

## T-42 — Morning-triage resolutions (2026-08-07, run `20260808`)

Triage of `overnight-20260808` — 1 of 1 cards merged by the run (E `spike-e-reconnect-catchup`
@ `0ac5a20`), zero parks, zero conflicted merges, zero open forks. **Spike E GREEN closes
Activity 2 at 5 of 5 (A–E all GREEN) and answers B-161: dark-window catch-up can be trusted —
CONDITIONAL on the relay staying trigger/NOTIFY-driven.** The condition is load-bearing and
must ride the Activity 3–5 build cards' text: the checkpoint pulls on the substrate's
trigger-stamped `_modified` (strict `gt` + id tie-breaker), so a future polling relay on a
business watermark reintroduces the missed-UPDATE hazard exactly (`submitted_at` measured
never advancing after INSERT). Merged to `dev` as a `--no-ff` merge after an adversarial
subagent independently reproduced every gate on the final tree: G1 build+vet clean; G2 Go
9 packages, 456 named tests (454 pass + 2 live-proof skips), workflow=35, on :5434; G2
Playwright **792 passed / 0 failed / 6 skipped** over the identical 798-test set (`[bdd]`
project run via `npx bddgen`); G4 exit 0, precache 31, committed `sw.js` byte-identical to
regenerated, 1.4.0 parity three ways. **Both spike legs re-executed LIVE by the review** —
green exit 0 (INSERT / UPDATE-to-held-row / INSERT all recovered, first post-reconnect pull
resumed from the sever-time checkpoint, scratch Postgres on ephemeral port 51151) and red
`--no-pull` exit 1 (all three missed, liveness control arrived) — substrate teardown
byte-verified on both. GREEN proven non-vacuous by code: the verdict requires the run-unique
sentinel on the already-held row (`spike-e-reconnect.js:314,513-517`). Go suite re-run on the
merged `dev` tree: exit 0, fully cached — the cache hit is itself the proof the merged tree is
content-identical to the tree the review executed minutes earlier. Conflict-log audit clean:
1/1 merges logged (clean merge logged per §15ad.66), the entry names its intent, the intent
carries the three required fields. Post-merge worktree sweep clean for the run's card; only
`card/a3-rls-fixture-own` remains, preserved by decision 155.

Evidence corrections of record (documentation-level; no gate implicated):

- **The three "armed reds" are flakes, not deterministic reds — all three PASSED in triage's
  re-run of the identical tree** (`inventory:883` B-27, `sync:446` [LST-17],
  `receipt-carousel:123`). Their occurrence on the night is attested by ✘ marks in the
  committed log; their deterministic framing in HANDOFF is falsified. Recorded as flake-protocol
  evidence, not as doubt about the run's honesty.
- **The fixture trigger is `BEFORE INSERT OR UPDATE`, not "AFTER" as HANDOFF words it**
  (`hq_sync_checklists_set_modified` — necessarily BEFORE, since only BEFORE can set
  `NEW._modified`). Mechanism claim itself correct.
- **Gate logs cite a rewritten branch head** (`d65273a`/`ef801cb`, same-message
  `filter-branch` twins of `aada295`/`ff778b9`) — gated code content proven equal to merged
  content by diff; evidence-chain hygiene only, feeds B-155's attestation-header convention.
- **`night-crew run-evidence check` reads `no-run-evidence` for this run against hq's layout**
  — it seeks `reference/conflicts-<runid>.md` and `runs/<runid>/` at dev-clone paths while hq
  keeps them under `.night-crew/knowledge/reference/` and `runs/<date>-autonomous/`; the
  closeout record and conflict log verifiably exist. Tool-layout mismatch, not a missing
  closeout.

Triage dispositions decided at role level (stated, not asked, per standing practice):

- **Receipt-carousel red filed as B-162** — B-32's family shape, on no armed list, provably
  not card E's; candidate family member (c), armed-expected next slate, retire only by
  diagnosis (decision 100).
- **LST-17 STAYS ARMED.** Tonight completes the flip-flop: failed in the run's suite after
  passing twice on `20260807-2`, then passed again in triage's re-run — "flipping = still
  flaky", exactly the shape T-41 anticipated. Retirement only by diagnosis.
- **B-161 answered GREEN by card E**, disposition recorded in the entry with the
  trigger/NOTIFY condition stated. (Backlog validator floor moves 295→296; the +1 is B-161's
  done-annotation in B-62's exact grandfathered `promoted → … · done —` pattern; zero
  findings against the four new entries.)
- **Follow-ups graduated to BACKLOG.md as B-163–B-165** (spike-E exit-code hardening incl.
  the `task` wrapper's 201; bare `backend:*` :5433-default guard, from the disclosed
  near-miss; `npx bddgen` load-bearing/undiscoverable — triage's own review initially fell
  into that hole).
- **Launch-prompt convention, from follow-up 5:** isolation DB names in launch prompts must
  be minted in `TEST_DB_NAME_PATTERN`'s shape (`^hq_test(?:_[a-z0-9]+)*$` —
  `scripts/reset-e2e-db.js:115`); the run's rename of the prompt's refused literal to
  `hq_test_e_reconnect` was correct (weakening a guard to fit a prompt was not on the table).
- **The standing-rule-1 near-miss is accepted as disclosed** — guard refused, read-only,
  zero writes, immediately corrected; the run's own reporting (breach, not skip-narration)
  is the wanted behaviour. Mechanical hardening is B-164's lead.
- **`card/spike-e-reconnect-catchup` deleted as merged housekeeping** (worktree already
  removed by the run; branch ancestry-merged via the run branch); `overnight-20260808` stays
  local per the branch model.
- **Zero gray areas were routed through `decisions log` by the run**, the ratification queue
  read empty, and the ratchet reported nothing survived — all three consistent with
  DECISIONS-NEEDED's own record that nothing reached the routing threshold and nothing was
  decided under delegation; nothing was skipped. Triage itself resolved no operator fork (the
  merge is the ritual's standard act), so nothing was routed at triage either. Preference
  coverage reads truthfully over an empty log: the run had no gray areas for a preference to
  address.

## T-43 — Slate-planning resolutions (2026-08-07 evening, slate `20260808-2`)

Attended `/nc-slate-plan --hours 8` session; three rulings taken inline at the fork gate,
before sign-off, all operator's own answers (AskUserQuestion, phrasing-checked):

- **(a) Approvals tab stays on re-fetch** — partial resolution of **B-43**. The Approvals
  list keeps today's REST fetch; it is not sync-served, and the cutover is partial by
  design on that tab. Recorded, not deferred.
- **(b) The My Checklists read path is deliberately left OPEN** — the operator explicitly
  declined to rule it tonight ("Approvals only; keep My Checklists open"). 🛑 **No card may
  decide it**; a card that cannot proceed without the answer parks. B-43 therefore stays
  partially open with a named remainder, not silently carried.
- **(c) Concurrent multi-checklist fill work is a RECORDED PRODUCT REQUIREMENT** — the
  operator, verbatim intent: crew members work multiple checklists at the same time (a
  setup checklist and a food-preparation checklist concurrently). This resolves **B-63**'s
  direction: multiple live per-checklist fill replications at once ARE the design — one per
  open checklist, cancelled on close, checkpoints per-identifier — and the standing
  *"CANCEL BEFORE RE-SCOPING"* banner is to be restated as *"cancel before re-scoping THE
  SAME shape"* (jointly B-63/B-64, rides cards C3/S1 of slate `20260808-2`).

Also decided at this session: **dispatch mode SERIAL** (both projections shown — serial 3
cards above the line w/ ~70m margin vs concurrent 4 w/ ~50m; operator chose serial), and
the **workflow-preflight verdict recorded** (openspec absent, exit 0 → universal mechanics
only in the launch prompt). Slate: `reference/slate-20260808-2.md`; launch prompt:
`reference/launch-20260808-2.md`. Sign-off GRANTED same session.

## T-44 — Morning-triage resolutions (2026-08-08, run `20260809`)

Triage of `overnight-20260809` — **2 of 2 cards merged by the run, zero parks, zero
conflicted merges, zero open forks.** Merged to `dev` as `--no-ff` merge `3e94c4a` after
an adversarial subagent independently reproduced every closeout claim on the final tree —
**nothing was falsified, and no claim was left unverified** (the live GREEN round-trip the
brief expected to be out of reach was reproducible because the Spike A substrate was up).

- **C1 `demo-sync-target`** (Activity 5, milestone close-bar) — merge `0fade6b`. `task
  demo:sync` + `demo-sync.sh`: one field through the real `POST /saveResponse` → NOTIFY
  relay → PostgREST → RxDB-served read, tri-state exit 0/1/2. Verdict **GREEN**. Roadmap
  card flipped PLANNED → DONE by the run. G6 PASS-with-findings (read-surface note; no fix
  round). Cycle ~25m (implement ~14m · G6 ~9m · land ~2m).
- **C2 `spike-exit-code-honesty`** (B-163) — merge `3e6cd5c`. Seven `srcpsql` guards
  (card cited four) + vacuous-green→2 in `spike-e-reconnect.sh`, uncaught-exception→2
  handlers in `rxdb/spike-e-reconnect.js`, Taskfile 201-trap note; all three conflations
  proven red-first 1→2. **B-163 → RESOLVED** in BACKLOG. G6 PASS (no over-correction; no
  fix round). Cycle ~21m (implement ~13m · G6 ~7m · land ~1m).

**Gate evidence (adversarial reproduction on the final tree, not the closeout's own lines):**
G1 build+vet clean; G2 Go `go test ./... -p 1` exit 0, 9 packages, 0 FAIL, DB-coupled tests
genuinely ran on :5434 (internal/sync 158 `=== RUN`, internal/workflow 35 results — not
vacuous skips); G2 Playwright/G3 **N/A-by-footprint** (no `[e2e.seams]` key matches; openspec
absent); G4 `build-sw.js` exit 0, **precache 31**, reachability 0-outside, committed `sw.js`
byte-identical (regen churn is cosmetic — all 31 URL+revision pairs identical); G4 discipline
greps N/A-VACUOUS, verified by `find` (neither `internal/journal` nor `internal/workorder`
exists, B-14). Footprint verified by git: zero `.go`, zero `*replay*`/`*testdata*`,
`workflows.html` read-only, `sw.js` untouched. **C1 tri-state 0/1/2 reproven LIVE** (GREEN
219 ms, RED via `--break-roundtrip`, could-not-run=2) and the `task demo:sync:red`=201 vs
script's true 1 wrapper-trap reproven live; **C2 all five claims** (infra→2, vacuous→2,
uncaught→2, no over-correction, exactly-7-guards with no eighth) reproven/verified. Go suite
re-run on the merged `dev` tree: exit 0, fully cached — the cache hit is itself proof the
merged tree is content-identical to the tree the review executed minutes earlier.

**Conflict-log audit clean:** 2/2 merges logged (both clean, logged per §15ad.66); each entry
names the intent it read; both merge-intents carry the three required fields, filled with real
content, empty sections explicitly marked. C1's Taskfile note (additive `demo:sync` stanza)
and C2's (comment-only note on the disjoint `spike:reconnect` stanza) are non-overlapping —
verified against the +53 Taskfile diff.

Triage dispositions decided at role level (stated, not asked, per standing practice):

- **Read-surface awareness item — surfaced, NOT a fork.** The demo reads through a Node RxDB
  client (`rxdb/spike-c-read.js`), not the browser app UI — it clears the close-bar *letter*
  (real write, RxDB-served read via the identical `replicateSupabase` plugin against the real
  substrate) but exercises no app surface. G6 judged it a legitimate in-card engineer decision
  (the slate's own "too heavy for a clean demo" fallback), MEDIUM operator-awareness, not a
  defect. **It is carried to the operator's `dev-complete-attestation`, not resolved here:** at
  that attended act the operator decides whether a data-plane proof satisfies "the sync
  capability running in my dev environment", or files a follow-up card to drive the real
  browser fill-view against the real substrate. Adversarial review confirmed the note is
  accurately characterized, neither over- nor understated.
- **`gate-rls-fixture-ownership` (A3) stays ARMED** — untouched by this run; remains BLOCKED on
  the attended re-gate (decision 155). No morning evidence bears on it.

Findings of record (documentation-level; no gate implicated):

- **`night-crew run-evidence check` reads `no-run-evidence` for this run — recurrence of the
  T-42 finding.** The binary (v3.3.0+7) seeks `reference/conflicts-<runid>.md` and
  `.night-crew/runs/<runid>/{journal,summary,metrics}` at dev-clone paths while hq keeps them
  under `.night-crew/knowledge/reference/` and `runs/<DATE>-autonomous/`, and it does not read
  `closeout-<runid>.md` at all. The closeout record and conflict log verifiably exist; the
  night ran and closed. Its `card-branch check` half works (read both cards as covered). **A
  clone-side fix (run-evidence path resolution for scaffolded target repos + a
  `closeout-<runid>.md` reader) — NOT an hq run-branch remedy (B-14 discipline).** File
  against the night-crew clone; the run's HANDOFF §"Tooling finding" carries the detail.
- **`task backend:db-test` invoked BARE falls back to its own `:5433` defaults** — the
  prod/dev cluster serving hq.yumyums.kitchen. Surfaced by adversarial review (which invoked
  it directly; it tried `:5433` and only failed because that cluster was down in the review
  environment). Pre-existing and orthogonal to both cards (this run touched no Go); the hazard
  is already *documented* at Taskfile.yml:120-123 and all four wrapper targets (`test:go`,
  `test:`, `test:ui`, `test:all`) hand it `:5434` explicitly — but a direct/naive invocation
  in the real environment would create test databases on the prod cluster (B-141/B-143 class,
  decision 155). Graduated to **B-169**.

Post-merge worktree sweep: `overnight-20260809` and both card branches confirmed on `dev`.
Three worktrees hold pre-existing unmerged work, none tonight's cards — `workspace/hq-scheduling-app`
(31 commits, separate GSD workstream, Jun 7–9), the `main` worktree (1 commit `b89c202`, prod-Docker
feat May 17, the frozen-release model), and `worktree-agent-ae9998ae` (4 commits, shared-API-client
`260703-uu2`, Jul 3–4, genuinely stranded). Each is the operator's call; none proposed for merge
(B-133). No new decision number assigned — the run parked no fork and routed no gray area.

### T-44 addendum (same attended session) — "dev complete" redefined to the operator's intent; card `sync-live-in-dev` cut (decision 161)

The read-surface awareness item, pressed further in the same triage sitting, surfaced a gap the
T-44 disposition **undersold**: it framed the miss as "Node RxDB client vs browser UI," but the
real gap is that the sync data plane runs in **no persistent environment at all**.
`HQ_SYNC_REST_URL`/`HQ_SYNC_REALTIME_URL` are set **nowhere** in the tree (verified across every
`Taskfile*.yml`, `.env`, `docker-compose*.yml`), so the in-server `/sync/*` proxy fails closed
(503) in `dev`, `dev:tailscale` and prod (`cmd/server/main.go:436-438`); the substrate (PostgREST
+ Realtime, `env-up.sh`) and the relay (`cmd/spikec-relay`, a separate binary) are stood up only
inside `demo:sync`'s throwaway stack and torn down when it exits. `task backend:dev:tailscale` runs
the same HQ binary — so it carries the sync *endpoints* (`/ops`, `/ws`, `/api/v1/sync/token`, the
`/sync/*` proxy) — but with the door closed and no substrate/relay running, its RxDB data plane is
dark. The demo proves the capability *works*; it does not make it *usable*.

- **Decision 161 — `dev complete` is redefined to the operator's stated intent, not the chosen
  demo bar.** The roadmap round's close bar ("`task demo:sync` stands up a scripted-fresh
  environment") was a narrowing of the operator's own user story (roadmap.md:31-34: *"the sync
  capability running in my dev environment … something I can actually use"*). Presented the
  letter-vs-intent fork with concrete scenarios; **operator ruled, verbatim: "dev complete should
  be marked my dev environment and something that i can use in the future."** So the milestone is
  **NOT dev-complete** on the demo alone, and `dev-complete-attestation` is NOT attestable yet.
  Chosen over closing on the letter + backlog-capturing the follow-up (rejected: the operator's
  intent was explicitly to never again ship "everything is built" that means "something I cannot
  use without another follow-on milestone" — closing on the letter would reproduce exactly that).
- **Card `sync-live-in-dev` cut** into Activity 5 (PLANNED), ahead of `dev-complete-attestation`:
  (1) persistent substrate + relay as a dev service, (2) `HQ_SYNC_*` wired into the dev targets so
  the `/sync` door opens, (3) real-browser (`workflows.html`) proof against the persistent
  substrate — with a `done_when` block and a **spike gate on leg 3** (browser-against-real-substrate
  was never spiked; `/nc-spike` it before slating, per B-345). `demo-sync-target` stays DONE as the
  data-plane proof. The roadmap close-bar paragraph carries a dated correction blockquote; the
  `dev-complete-attestation` card is reworded to the app-surface read.
- **The T-44 read-surface disposition is superseded by this addendum** — the "carry it to the
  attestation, operator decides then" framing is replaced by a concrete card, because the operator
  has now decided: the app-surface read is required for close, not optional.

---

### Morning triage — run `20260810` (attended 2026-08-09)

Not a decision (no fork was raised). A durable record of what triage did.

- **Merged `overnight-20260810` → `dev` `--no-ff`** (`5e3a025`), operator-approved. Both Activity-5
  cards landed: `sync-live-in-dev-substrate` (`bd03059`) and `sync-live-in-dev-app-proof`
  (`489145e`). **No parks, no operator forks**; the one engineer-level call (Card 2's gate-harness
  form = standalone spike-style, gated on its own exit) was decided in-run and recorded in the
  merge-intent. Ratify and ratchet queues both empty. `decisions log` routed nothing (zero gray
  areas — not manufactured).
- **Gate evidence is from a fresh adversarial reproduction, not the closeout's own lines.** A
  fresh-context subagent rebuilt in its own scratch and re-ran everything on the merged tree:
  `go build`/`go vet` 0; `go test -p 1` on **:5434** all `ok`/0 FAIL with DB-coupled tests genuinely
  executing (`internal/sync` 45 pass +2 benign live-substrate skips, `internal/workflow` 35 pass —
  silent-skip trap defeated). Footprint honest: the only production-code touch is a **comment-only**
  doc block in `spikec_relay.go`; no `workflows.html`/`sync-rxdb/*`/`sw.js`/`tests/*.spec.js` changed;
  precache **31**, repo-hygiene seam count **11**, `night-crew.toml` comment-only; `HQ_SYNC_*`
  dev-targets-only and absent from prod; B-164 must-fix (`HQ_SYNC_DEV_ALLOW_5433`) present, no bare
  live `:5433`. **No `:5433` command ran the night** (B-164). Re-ran `go test` on the merged tree
  post-merge: exit 0, 9 packages ok.
- **Milestone left ONE attended act from close.** `sync-live-in-dev` (Activity 5) is delivered — the
  RxDB sync capability now runs persistently in the dev environment and is proven usable in the app,
  red-first and automated (Card 2). Per decision 161 the close bar is the operator's own
  `dev-complete-attestation`: `task sync:dev:up`, open `workflows.html` in `dev:tailscale`, see a
  field sync in the app, record the ledger line. That attestation run is the ONLY sanctioned place the
  real dev `:5433` coordinate is touched (`HQ_SYNC_DEV_ALLOW_5433=1`, knowingly). Separately, the A3
  attended re-gate (`gate-rls-fixture-ownership`, decision 155) is still owed.
- **Graduated to backlog:** **B-170** — bare `npx playwright test` in the remaining spike scripts
  falls through to a foreign PATH `playwright` (bit Card 2's setup twice; its own harness is hardened,
  the others are not).
- **Two standing findings outside this run** (worktree sweep; NOT merged — remedy is the operator's):
  (1) `workspace/hq-scheduling-app` holds 31 unmerged commits — the GSD scheduling workspace, already
  tracked by the "resume GSD Phase 23" backlog-PRIORITY item on `dev`; (2) `main` carries 3 old
  deploy commits (`b89c202`, `572f370`, `4cd81c3`, May/Jul) whose patch never came back to `dev` —
  pre-existing main↔dev divergence.
- **Tooling note (night-crew clone):** `run-evidence check --run 20260810` false-negatives
  (`no-run-evidence`) because it resolves `.night-crew/runs/20260810/…` + a bare `reference/`, but this
  repo's run dir is `2026-08-10-autonomous/` and reference lives under `.night-crew/knowledge/`. The
  closeout artifact (`closeout-20260810.md`, committed `b97c48a`) was verified directly; the run's
  completion is not in doubt.
