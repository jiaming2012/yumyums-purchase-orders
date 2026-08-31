# Conflict log — run 20260901

One entry per merge (clean or conflicted), per §15ad.66. A clean merge gets a
one-line "no conflict" entry so an empty log never reads as "no conflicts" when
it means "the logging never ran". Morning triage audits this file.

**Run:** `overnight-20260901` (off `dev` @ `55aa6f8`). 11 cards, 3 concurrent tracks
(A: cards 1–3 · B: cards 4–8 · C: cards 9–11). Orchestrator alone merges to the run
branch, in landing order. `sw.js` regeneration rule: every card touching a precached
file runs `task sw` + commits `sw.js` in its own change set; after the LAST of Cards 7
and 11 lands, re-run `task sw` on the merged tree and commit the regeneration with the
merge (precache count 31 — an unexplained move is B-37).

---
