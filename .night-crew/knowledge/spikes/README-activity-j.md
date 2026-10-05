# Activity J — Scanner and backend guards (triage 20261003 follow-ups)

> Ledgers live in `activity-j-scanner-and-backend-guards-triage-20261003-follow-ups/` (the slug the
> spike gate derives from the activity title); this README sits beside that directory because the
> gate reads every `.md` inside it as a goal ledger.

Two goal ledgers, one per card, authored at the slate sitting of 2026-10-04 (the evening of the
morning triage T-64 that filed B-474 … B-481 and named B-475 as the fix in decision 205). Runs are
tool-recorded (`night-crew spikes run --activity … --goal …`, binary v3.6.1+4), so the ledgers'
`### Runs` lines are the verdict the gate reads. Scripts live under
`.night-crew/spikes/activity-j-scanner-and-backend-guards-triage-20261003-follow-ups/<goal>/`.

Substrate discipline: `:5434` (`yumyums-test-pg`, role `hqtest`) for everything, in spike-owned
databases `hq_test_spike_j<n>_20261005` / `hq_test_spike_j<n>_go` (the reset guard's pattern);
every mutation happens in a THROWAWAY git worktree under `../hq-worktrees/spike-j<n>-20261005`
that the script removes on every exit path. Never `:5433`. No substrate (spike-supabase) is
needed by any spike here — the scanner spikes mock the sync door at the network layer the way
`tests/marketing.spec.js` does.
