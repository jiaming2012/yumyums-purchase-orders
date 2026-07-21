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
