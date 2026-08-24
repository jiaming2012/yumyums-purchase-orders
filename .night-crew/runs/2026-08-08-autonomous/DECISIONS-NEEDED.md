# DECISIONS-NEEDED — run `overnight-20260808`

**No open sections. Nothing was parked tonight.**

- Zero cards parked; 1 of 1 merged (E — see HANDOFF.md). Spike verdict GREEN, exit 0, UPDATE
  case exercised and recovered.
- Zero gray areas routed through `night-crew decisions log` — none reached the routing
  threshold. The card's PARK-note conditions stayed untriggered: the chosen severing
  mechanism (client-side `rep.cancel()` + realtime disconnect) touches nothing on :5433,
  adopts no external service, and leaves decision 126's read/write split intact (all writes
  went through `POST /api/v1/workflow/saveResponse`). Nothing was decided under a delegation
  policy, so nothing awaits ratification.
- Two mechanism-level calls were decided and stated, not escalated (per the
  decide-plumbing-yourself rule): (1) the `TEST_DB_NAME` rename to `hq_test_e_reconnect` —
  the launch prompt's literal value is refused by `scripts/reset-e2e-db.js`'s guard pattern,
  and weakening a guard to fit a prompt was not on the table; (2) severing at the client
  rather than the substrate, per the card's own stated preference (keeps A–D reproducible).
- The 🛑 standing-rule-1 near-miss (bare `task backend:db-test` → guarded, refused, read-only
  :5433 contact, zero writes) is a **disclosure, not a fork** — the event already happened
  and the remedy (report it) is applied in HANDOFF.md §"Standing-rule-1 near-miss". Whether
  to harden further (follow-up 4) is an ordinary triage call, not an operator-only fork.

Items that are **operator work but not forks** are in HANDOFF.md §"Attended work still
waiting" (unchanged) and §"Follow-ups" (the receipt-carousel red's filing and LST-17's
armed-status evidence are triage rulings under already-decided protocols).
