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
