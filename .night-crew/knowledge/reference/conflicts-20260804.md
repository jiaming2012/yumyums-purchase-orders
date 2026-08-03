# Conflict log — run `overnight-20260804`

Per §15ad.66: **every** merge gets an entry, clean or conflicted. A clean merge gets a one-line
entry, so an empty log never reads as "no conflicts" when it means "the logging never ran."

Each entry: the cards involved, files and hunks, the merge-intent notes read, the resolution taken,
and the gate result after it.

---

<!-- entries appended in merge order -->

## Merge 1 — A1 `e2e-gate-database-isolation` → `overnight-20260804`

**CLEAN.** No conflicts, no hunks resolved. First card of the run; nothing to collide with.

- **Cards involved:** A1 alone.
- **Files:** `playwright.config.js`, `Taskfile.yml`, `scripts/reset-e2e-db.js` (new),
  `tests/db-isolation.spec.js` (new), `BACKLOG.md`, `roadmap.md`, and the card's merge-intent note.
  7 files, 638+/40−.
- **Merge-intent read:** `merge-intents/merge-intent-a1-e2e-gate-database-isolation.md`. Its
  must-survive list carries 7 items; item 7 (the reset banner reaching stderr) was *promoted* from
  "safe to drop" during the fix round, because dropping it is exactly B-81.
- **Outside the stated footprint:** `scripts/reset-e2e-db.js` — a new module; the slate named only
  "a new spec under `tests/`" plus the three config files. Disclosed in the note. Legitimate.
- **`night-crew.toml` NOT touched** — a result, not an omission. The slate listed it expecting
  `[e2e] suite`/`subset` would need repointing; they didn't. The PARK trigger never fired.
- **Resolution taken:** none required — `--no-ff` merge, ort strategy, no conflicts.
- **Gate result after the merge:** G1 `go build ./...` rc=0 · `go vet ./...` rc=0 (from `backend/`).
  G4 `node build-sw.js` → **31 files precached** (unmoved), reachability 18 parsed / 30 resolved /
  0 outside, frontend 1.4.0; second run left the tree clean ⇒ idempotent. Full G2 is deferred to
  the closeout's final-tree gate, per the run's normal shape.
- **G6 verdict:** APPROVE-WITH-NOTES. Acceptance gate independently re-derived by execution.
