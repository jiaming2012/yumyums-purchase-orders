# Conflict log — run `20261005`

One entry per merge onto `overnight-20261005`, clean or conflicted (§15ad.66). An entry that says
"clean" means the merge was logged and had no conflict — not that logging was skipped.

## Merge 1 — `card/i4-atomic-scan-dedupe` @ `8251ac9` → `f980e12` · CONFLICTED (1 file, expected)

- **Cards involved:** Card 1 `atomic-scan-dedupe` (I4) only. Built and reviewed in run `20261003`;
  merged tonight 3-way, both parents kept (`588f188`, `8251ac9`), never squashed.
- **Files and hunks:** `.night-crew/knowledge/roadmap.md`, one hunk — the card's own status line.
  The branch flipped `PLANNED` → `LANDED (run 20261003 …)` on the 2026-10-03 text; `dev` had since
  rewritten the same line at triage ("parked … settled as decision 203 … owes its full
  browser-suite run and a merge"). Nothing under `backend/` conflicted: `git log 4adf1ad..dev --
  backend/internal/marketing/landing.go landing_test.go` is empty and `0087` is the highest
  migration on both sides.
- **Intents read:** `.night-crew/runs/2026-10-03-autonomous/merge-intents/atomic-scan-dedupe.md`
  (what must survive: `0087` after `0086`; generated column + partial unique index verbatim; the
  ONE-statement insert with the matching `WHERE` in its conflict target; `NULL ip_hash` inserts
  every time; de-dup before the index build; the barrier test; Down round-trip via `db.Migrate`).
  The roadmap line is listed there as "this card's own line only".
- **Resolution:** as the slate prescribed — `dev`'s text, with this card's line flipped to
  `LANDED (run 20261005, branch card/i4-atomic-scan-dedupe, merge f980e12)` and the "owes its full
  browser-suite run and a merge" clause dropped (tonight discharges it). The header paragraph that
  still says the card "stays PLANNED" is `dev`'s triage prose and was left as written
  (append-only); triage owns that narrative. No judgment about behaviour was involved, so **no new
  G6** — the resolution is one status line.
- **Gate after:** G4 — `sw.js` regenerated at the merged HEAD twice, no diff, 51 precached
  (`logs/merge1-G4-sw.log`). G1 + full Go + the OWED full Playwright suite on the merged tree:
  **see "Merge 1 — gate result" below** (written when the suite, which queues on the box-wide
  lock behind the base-reds leg, reports).
