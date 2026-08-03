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

---

## Merge 2 — A2 `workflows-autosavefield-phantom` → `overnight-20260804`

**CLEAN.** No conflicts, no hunks resolved. A2 branched off the run branch *after* A1 merged, so the
two cards' shared files (`BACKLOG.md`, `roadmap.md`) were already reconciled by construction — this
is the serial dispatch doing the job concurrency would have made harder.

- **Cards involved:** A2 alone. A1 was already in the base.
- **Files:** 15 files, 594+/46−. `workflows.html` (the fix), `tests/persistence.spec.js` (the
  red-first `[FLD-16B]`), `sw.js`, `CLAUDE.md`, `sync-rxdb/bootstrap.js`,
  `sync-rxdb/conflict-notice-ui.js`, `docs/data-flow-audit.md`, `README.md`,
  `.claude/skills/save-project/SKILL.md`, `prds/PRD-operations-hardening.md`,
  `designs/fetchstorm-replay-class-superseded.md`, `tests/sync-rxdb-client.spec.js`,
  `BACKLOG.md`, `roadmap.md`, and the card's merge-intent note.
- **Merge-intent read:** `merge-intents/merge-intent-a2-workflows-autosavefield-phantom.md`.
  Must-survive carries 4 items; item 4 records that B-65's "match a symbol not a substring" lead is
  **partially, not fully, discharged**, with the failure scenario stated.
- **Outside the stated footprint:** five files —`docs/data-flow-audit.md`, `README.md`,
  `.claude/skills/save-project/SKILL.md`, `PRD-operations-hardening.md`,
  `tests/sync-rxdb-client.spec.js`, plus `designs/fetchstorm-replay-class-superseded.md` from the
  fix round. All disclosed. The slate named four doc sites; **eight** were wrong, and the transport
  (`POST /saveResponse`) was a second phantom the slate did not know about. Correcting `CLAUDE.md`
  while leaving its downstream restatements would have made the correction unfollowable.
- **🛑 The PRD was appended to, not rewritten.** Both corrections sit in blockquotes *below* the
  signed requirement text, each stating the requirement itself is unchanged. Annotation is a card's
  to do; rewriting signed requirement text is not. G6 verified this specifically.
- **🛑 PARK trigger honoured, not driven through.** `CLAUDE.md`'s persistence rule survives intact —
  G6's separate verdict. Only the false function name, transport and call shape changed. The
  persisted-states list went 7 → 9 (*widening* the rule's coverage), and step 2 of the
  add-a-field-type procedure got **stricter**, not looser.
- **Resolution taken:** none required — `--no-ff` merge, ort strategy, no conflicts.
- **Gate result after the merge:** G4 `node build-sw.js` run **after** the merge commit (B-37) →
  **31 files precached** (unmoved), reachability 18 parsed / 30 resolved / 0 outside, frontend
  1.4.0; `sw.js` unchanged and tree clean on both runs ⇒ idempotent. The card's own G2 was the
  **full 787-test suite** — not by choice but because of B-87 — `1 failed / 6 skipped / 780 passed`,
  the one failure being armed red `yes/no answer converges (live + catch-up)` matched by full title,
  which **stays ARMED**. First figures this milestone measured against a **fresh** database.
- **G6 verdict:** APPROVE-WITH-NOTES, and it **refuted a claimed finding at source** — B-87's
  proposed consequence (a confined gate silently running the *wrong* specs) does not follow, because
  the CLI path filters are OR'd and can only widen. Filed with the corrected consequence, wrong
  version struck in place.
