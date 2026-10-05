# Extraction — states-screenshots-out-of-tree

Outcome: confirmed, no corrections

Approach used: a throwaway worktree of `dev` on a spike-owned e2e stack; the states spec
run alone; `git status` under the committed screenshot directory read back; the ignore
convention checked with `git check-ignore`; the `SHOT_DIR` constant read statically.
Tool-recorded run: exit 0 first time. Candidate input for the card, not an adoption.

Confirmed: running `tests/states-marketing-stats.spec.js` alone modifies 12 of the 23
tracked PNGs under `.night-crew/runs/2026-10-02-autonomous/logs/h4/states/` (B-486);
`test-screenshots/` is already gitignored; the spec hard-codes the tracked path with no
`STATES_SHOT_DIR` override.

Learned: (1) the dirtying is deterministic, not a flake — a dozen rows differ every run;
(2) the fix is one constant plus a header comment, and the committed H4 set is the
reviewed evidence and must not be rewritten (B-467).

Plan change: none — the card ships the env-overridable default under
`test-screenshots/` and a static assertion that `SHOT_DIR` is untracked.
