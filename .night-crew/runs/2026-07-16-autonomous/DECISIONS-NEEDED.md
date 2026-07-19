# DECISIONS-NEEDED — overnight-20260716

> **RESOLVED 2026-07-16 — recorded as `ledger.md` T-14 (morning-triage resolutions
> 2026-07-16).** No forks were parked (the PARK trigger did not fire); the cycle gate was
> ratified and signed off, the run merged to `dev` `--no-ff` (`7f57d14`). Kept as the run's
> analysis record. Milestone boundary — next move is `/nc-okr-session`.

> Forks/exceptions surfaced by the cycle-gate closeout run that need an operator
> decision at morning triage. Empty header = nothing parked.

_Status: **NO OPEN FORKS.** Card 1's full-suite run completed; all 38 Playwright reds +
the 1 Go-unit red map to a documented category, and none of the 4 this-cycle-repaired
flows regressed. The PARK trigger did not fire. Gate attests clean under the operator's
2026-07-15 "Attest & waive" posture. This file stays empty._

## §A — Uncategorized suite reds (Card 1 PARK trigger)

_(none — Card 1 ran the full suite on an isolated pg16 stack: **387 passed · 38 failed ·
0 flaky · 6 skipped** (Playwright) + **1 env-gated Go-unit red** (`internal/receipt`,
no `ANTHROPIC_API_KEY`). Every red categorized against a documented cause; a 7-test
fix-adjacent isolation re-run confirmed the split (1 pollution → green in isolation, 6
structural/seed-dependent). 0 uncategorized reds → no PARK. Evidence: `reference/cycle-closeout-20260716.md` §1.)_
