# DECISIONS-NEEDED — run 20260907

**Nothing is parked.** No card hit a PARK condition, no merge was unsettleable, no
unplanned lifecycle row appeared, and no question was routed through
`night-crew decisions log` — zero gray areas arose, so there are zero delegated
decides awaiting ratification and zero escalations. Stated so an empty file and a
file nobody wrote don't look alike.

Two items travel to triage as *findings*, not forks (no operator decision is required
to close the run; both are recorded in HANDOFF.md §Findings):

1. **`FILL-04` (`tests/sync-fill-view.spec.js:451`) failed once in the full suite,
   outside the four-red baseline**, and passed 9/9 in standalone isolation on the same
   tree. Same one-off-under-load shape as B-437 (run 20260906-2). Candidate for a
   B-437-style flake filing at triage — filing a bug is triage's call, not a fork.
2. **B-433's four-red baseline no longer matches observation two runs in a row**:
   `B1-XT-01` passed on 20260906-2 and again tonight. Whether to re-measure the
   baseline is triage's housekeeping, not a fork.
