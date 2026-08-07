# DECISIONS-NEEDED — run `overnight-20260807-2`

**No open sections. Nothing was parked tonight.**

- Zero cards parked; 2 of 2 merged (C, D — see HANDOFF.md). Both spike verdicts GREEN.
- Zero gray areas routed through `night-crew decisions log` — none reached the routing
  threshold. Both cards' PARK-note conditions stayed untriggered: C's chosen mechanism
  (LISTEN/NOTIFY relay) keeps decision 126's shape (writes stay on `/saveResponse`/REST),
  touches nothing on :5433, and adopts no external service; D proved the filter without any
  schema/RLS change and without touching deploy config. Nothing was decided under a
  delegation policy, so nothing awaits ratification.
- The choice of WHICH mechanism to prove was the spike's own call per the slate ("that
  finding is the deliverable") — recorded as a finding, not a fork.

Items that are **operator work but not forks** are in HANDOFF.md §"Attended work still
waiting" (unchanged) and §"Follow-ups" (triage rulings on FR-11's flake handle, LST-17's
armed status, and B-62's close — rulings, not forks; the protocols that govern them are
already decided).
