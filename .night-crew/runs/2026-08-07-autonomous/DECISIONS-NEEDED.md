# DECISIONS-NEEDED — run `overnight-20260807`

> **RESOLVED 2026-08-06 — recorded as ledger T-40.** No forks were open; the one process
> observation below (runs git-operate from a dedicated worktree) was put to the operator and
> ruled — decision 160, pending candidate `operations/C-3`.

**No open sections. Nothing was parked tonight.**

- Zero cards parked; 3 of 3 merged (W0, A2, S — see HANDOFF.md).
- Zero gray areas routed through `night-crew decisions log` — none reached the routing
  threshold. Nothing was decided under a delegation policy, so nothing awaits ratification.
- A2's uid-mismatch question (the card's named park condition) did not become a fork: the
  card mirrored `index.html`'s already-shipped behavior verbatim, per the slate's own
  carve-out, and G6 confirmed no new product decision was smuggled in.

Items that are **operator work but not forks** (already ruled or already owned) are listed in
HANDOFF.md §"Attended work still waiting" and §"Follow-ups" — none needs a ruling to proceed,
only attended hands.

One process observation triage may want to turn into a standing rule (not a fork, a
recommendation): **runs should operate from a dedicated worktree, never the main checkout** —
a concurrent attended session moved the main checkout mid-run and the orchestrator's first W0
merge landed on `dev` (recovered in full; conflict log §1).
