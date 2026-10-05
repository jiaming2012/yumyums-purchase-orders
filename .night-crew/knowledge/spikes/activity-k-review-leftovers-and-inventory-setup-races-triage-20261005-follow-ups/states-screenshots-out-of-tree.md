# Spikes — states-screenshots-out-of-tree

Activity: Activity K — Review leftovers and Inventory Setup races (triage 20261005 follow-ups)

> Tool-run (`night-crew spikes run`). One Playwright spike in a THROWAWAY worktree off `dev`
> (`../hq-worktrees/spike-k3-20261007`) on a spike-owned e2e stack
> (`hq_test_spike_k3_20261007`, `TEST_PORT=8342`, `:5434`). Never `:5433`.

## The goal, and which legs need a spike

The card (K3 — B-486; adversarial review of run `20261005`, triage T-66):
`tests/states-marketing-stats.spec.js` writes its screenshots into a COMMITTED directory under
`.night-crew/runs/2026-10-02-autonomous/logs/h4/states/`, so every gate leg that selects the
`marketing` seam leaves the tree dirty. One premise a script can settle now: run that spec alone
in a fresh worktree and read `git status` — tracked PNGs modified; and the `.gitignore`
convention the fix relies on (`test-screenshots/` ignored, `STATES_SHOT_DIR` override) exists.

## Spike: running-the-states-spec-dirties-tracked-files

- proves: (a) in a fresh worktree of `dev`, `npx playwright test
  tests/states-marketing-stats.spec.js` (its own exit code ignored — the premise is the side
  effect) leaves `git status --porcelain --
  .night-crew/runs/2026-10-02-autonomous/logs/h4/states/` NON-EMPTY, every path a tracked PNG;
  (b) `git check-ignore test-screenshots/anything.png` succeeds — the ignored home the fix
  points the default at already exists; (c) the spec's `SHOT_DIR` is a hard-coded tracked path
  with no env override (static read of the one `const`).
- plan: worktree, warm build, run the spec on the spike stack, read the worktree's status, run
  the ignore check, grep the constant.
- script: .night-crew/spikes/activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/states-screenshots-out-of-tree/01-running-the-states-spec-dirties-tracked-files.sh

### Runs

- 2026-10-05T15:07:35Z · exit 0 · passed

## Verdict (tool-run 2026-10-05)

- **running-the-states-spec-dirties-tracked-files: passed** on its first run (exit 0, 2.8 m of
  Playwright): (c) `SHOT_DIR` is hard-coded to `.night-crew/runs/2026-10-02-autonomous/logs/h4/states`
  with no `STATES_SHOT_DIR` override; 23 tracked PNGs under it; (b) `test-screenshots/` is
  gitignored — the convention the fix uses exists; (a) `tests/states-marketing-stats.spec.js`
  alone, 16 passed, left **12 tracked PNGs modified** (`bi-loading.png`, `ms-declined-bucket.png`,
  `ms-loading.png`, `ms-locked.png`, `ms-long.png`, …) — B-486 reproduced by execution. Restored
  in the worktree; the main tree untouched.

## Corrections

- none agent-reached. Precision carried into the card: 16 of the spec's rows re-render
  identically and 12 differ run to run (timestamps, loading states), so "commit a reviewed set
  once" means the H4 set stays as it is and the spec simply stops writing there.

## Comebacks

- none.

## Review

- signed: operator, 2026-10-05 — covers 0 correction(s) (reviewed at the slate sitting of
  2026-10-05, slate-20261007 §4 batch sign-off; no corrections to review).
