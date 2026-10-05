# Activity K — Review leftovers and Inventory Setup races (triage 20261005 follow-ups)

> Ledgers live in `activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/`
> (the slug the spike gate derives from the activity title); this README sits beside that
> directory because the gate reads every `.md` inside it as a goal ledger.

Four goal ledgers, one per card, authored at the slate sitting of 2026-10-05 — the SECOND slate of
that sitting (run `20261007`, ledger T-69), asked for by the operator after slate `20261006`
(Activity E) was signed: "add a second slate after this one" for the bug fixes B-482 … B-487 and
the Inventory Setup races B-459 / B-478. Runs are tool-recorded (`night-crew spikes run`, binary
v3.6.1+4). Scripts live under
`.night-crew/spikes/activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/<goal>/`,
with two shared libraries beside the goal directories: `_pwlib.sh` (a Playwright spike in a
throwaway worktree on a spike-owned e2e stack) and `_golib.sh` (a Go spike in a throwaway
worktree on a fresh spike-owned Go database).

Each spike is a red-first BASELINE: it reproduces the filed finding by execution on today's `dev`
(the J precedent). Substrate discipline: `:5434` only, databases `hq_test_spike_k<n>_20261007` /
`hq_test_spike_k<n>_go`, worktrees `../hq-worktrees/spike-k<n>-20261007` removed on every exit
path, ports 8341–8343. Never `:5433`. No substrate (spike-supabase) is needed by any spike here.
