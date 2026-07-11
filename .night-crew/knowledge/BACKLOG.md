# Backlog — advisory items that ride future cards

> Durable parking lot for triage-surfaced items that aren't roadmap cards yet but must
> survive run-to-run (HANDOFF is rewritten each run). Format: `title · description · origin ·
> status`. Promote to a roadmap card with `promoted → <card>`; drop with `dropped — reason`
> (struck through, kept as record).

- **Users stale-E2E repair** · `tests/users.spec.js` has two Access-tab tests navigating dead
  `#t3`/`#s3` DOM (removed in the 3-tab→2-tab refactor; Access now renders into `#s2`).
  Features work; tests can't run — marked UNPROVEN (stale-test), not BROKEN. Repoint
  `#t3`/`#s3` → `#t2`/`#s2`. Folds into the **Users Activity-4 prove-UNPROVEN WO** (low effort).
  · origin: triage 2026-07-10 (D-4) · new

## Activity-4 fix-cards (from Activity-2 confirm-absence graduations — code-fix + regression-test WOs)

> Distinct from test-only prove-UNPROVEN WOs: these are **confirmed-BROKEN** flows where a cited
> line proves the behavior absent. Each = code fix (front+back) + red-first regression test.

- **Operations FR-4 — yes/no "No" corrective-action enforcement** · A "No" answer never blocks
  submit: `evaluateFailTrigger` handles only `out_of_range` (`workflows.html:1656-1668`), yes/no
  fields carry no `fail_trigger` (`workflows.html:558,724`), submit validation short-circuits on
  `!f.fail_trigger` (`workflows.html:2398-2405`), server `validateFailNotes` checks only
  `out_of_range` (`handler.go:80,101`). The "No" fail card (`workflows.html:2068`) is cosmetic.
  Fix: make a failing "No" require a corrective fail note front+back (mirror the temperature path)
  + red-first AC-3 test. · origin: overnight-20260712 ops-confirm-absence (G6-passed) · new
- **Operations NFR-3 — photo-required-at-submit enforcement** · No photo gate on submit/resubmit:
  frontend checks only note+severity (`workflows.html:2397-2419`), the `fld-photo-required` toggle
  + reject `require_photo` feed a banner only (`workflows.html:2024-2025`), backend
  `validateFailNotes` has no photo check (`handler.go:54-88`), submit runs one validation with no
  photo gate (`handler.go:458`). `PhotoURL` is storable (`model.go:92`) but never required.
  Fix: block submit/resubmit until a required photo is attached, front+back, + red-first test.
  · origin: overnight-20260712 ops-confirm-absence (G6-passed) · new
