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
