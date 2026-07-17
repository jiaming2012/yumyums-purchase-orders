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
