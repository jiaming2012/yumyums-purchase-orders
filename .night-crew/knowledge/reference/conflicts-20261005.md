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

## Merge 2 — `card/j2-merge-target-and-blanking-guards` @ `dc36a51` → `42fbd67` · CLEAN

- **Cards involved:** Card 3 `merge-target-and-blanking-guards` (J2), cut from `3bd6a9b` (the run
  branch with Card 1 already merged). No conflict; no file shared with Card 1's change set except
  `roadmap.md` / the run directory, on different lines.
- **Intent read:** `merge-intents/merge-target-and-blanking-guards.md` — must survive: the target
  read inside the existing transaction before any UPDATE; the three re-point statements and the
  delete unchanged; no migration.
- **Review (fresh context, at `51fcfb2`):** APPROVE-WITH-FINDINGS — 0 blocking, 3 record, 3 note.
  The reviewer drove the merge over real HTTP (missing target → `404 target_not_found`, dish and
  sales row intact; self-merge still 400; valid merge still re-points; target deleted mid-request →
  404 after the lock wait) and re-ran both mutations itself (guard removed → red; `0086`'s UPDATE
  removed → `23503`, red).
- **Fix round after review:** wording only, `dc36a51` — the three record corrections as the
  reviewer wrote them (merge-intent ×2, BACKLOG B-480 line). No code or test line changed
  (`git diff --stat 51fcfb2 dc36a51`: 2 docs files), so there was no mutation to re-run; said here
  so the absence is not read as a skipped step.
- **Gate after (confined, `logs/merge2-confined.log`):** `EXIT_BUILD=0`, `EXIT_VET=0`;
  `internal/marketing` + `internal/recipes` `EXIT_TEST=0`, 133 pass / 0 fail / 1 skip; `sw.js`
  regenerated at the merged HEAD, no diff, 51. **The first Go run in that log is invalid and is the
  orchestrator's mistake:** it created an empty database and ran `recipes` before any package had
  migrated it (56 setup failures, `relation "drift_check_results" does not exist`); re-run with the
  schema in place is the result above. The final full suite on the complete tree is this card's
  full gate.
- **Not pinned by any test (reviewer's notes, carried):** the `FOR SHARE` lock and the
  in-transaction placement held only in the reviewer's manual race probes; the test also passes
  with the guard after the first UPDATE. A missing SOURCE into a real target still answers
  `200 {"rows_re_pointed":0}`; a non-uuid target still answers 500 — both pre-existing.

## Merge 3 — `card/j1-photo-scan-guard-and-lookup-arms` @ `d6e7111` → `b0bc48c` · CLEAN

- **Cards involved:** Card 2 `photo-scan-guard-and-lookup-arms` (J1), cut from `3bd6a9b`; merged
  after Card 3. `BACKLOG.md` and `roadmap.md` were edited by both cards on different lines and
  auto-merged; no hunk needed a hand.
- **Intents read:** `merge-intents/photo-scan-guard-and-lookup-arms.md` and Card 3's. Nothing in
  either names a file the other must keep a particular way.
- **Review (fresh context, at `dcc5ad4`):** APPROVE-WITH-FINDINGS — 0 blocking, 2 record, 3 note.
  The reviewer re-ran two mutations itself (`[SV-10]` non-200 → `null`; `[SV-11]` `catch` →
  `return false` — each redded exactly its own spec), reverted the page fix and watched all three
  `[PS-*]` specs red on the missing submit control, and ran the whole marketing spec on a fresh
  database: 70 passed / 0 failed / 0 skipped. `[SV-08]` and `[SV-09]` mutations were NOT re-run by
  the reviewer (the card's own logs are their only evidence).
- **Fix round after review:** wording only, `d6e7111` — one comment line in
  `tests/marketing.spec.js` (`[SV-10]` names 503 too) and one merge-intent sentence (`[PS-01]`'s
  closing submit rides `mockRedeem`). No test or product line changed, so no mutation was re-run;
  said so the absence is not read as a skipped step.
- **Gate after (confined, `logs/merge3-confined.log`):** `EXIT_BUILD=0`, `EXIT_VET=0`; `sw.js`
  regenerated at the merged HEAD twice — no diff against the card's committed file, 51 precached;
  backlog check exit 0 (276 entries). The final full suite on the complete tree is the full gate.
- **Carried for the morning reader (reviewer's verdicts on what the card flagged):**
  1. The busy guard is wider than the slate's wording — it also covers the scan-by-text entry
     point. No crew path needed that (the page has only camera and photo); it is needed for the
     spike's own recipe and `[PS-01]`. `[PS-01b]` (photo then photo), an extra spec, is the only
     one driving the defect through crew-reachable entries. No product code calls scan-by-text.
  2. `[SV-11]` gates the fail-closed arm at the predicate level only. The page's refusal render
     under a throwing policy source is gated by NOTHING, and the slate's stated fallback was
     wrong: `campaigns-run.sh` never passes a throwing source (read, not run).
  3. A photo picked during the wait is dropped with no feedback of its own; one picked while an
     earlier photo is still decoding is dropped with no "Checking…" text at all. Within the
     slate's "no new copy".

## Merge 1 — gate result (the OWED full gate on `f980e12`)

- **G1:** `EXIT_BUILD=0`, `EXIT_VET=0`.
- **Full Go (`-p 1`, under the lock):** `EXIT_TEST=0` — 454 pass / 0 fail / 3 skip, 15 packages;
  base tonight 449 / 0 / 3, so +5 as the slate predicted. The five named tests passed by name;
  `0087`'s Down round-trip passed as `TestMigration0087DedupeBucketDownAndUpRoundTrip`;
  `TestRowVisibilityRLS` 59 subtests (`logs/merge1-G1-G2go.log`).
- **Full Playwright (under the lock, one summary block, config retries = 1):** `EXIT_PW=1` —
  22 failed / 2 flaky / 7 skipped / 1041 passed, 49.1 m. Tonight's base on `dev@588f188`, same
  retries setting: 24 failed / 2 flaky / 7 skipped / 1039 passed. **No failed test outside the base
  set**; `logs/merge1-reds.txt`, `logs/base-reds.txt`.
- **Isolation:** the one first-attempt red outside tonight's base, `inventory.spec.js:1469`
  (green on retry in the suite; it IS in run 20261003's base list), ran alone 3× on the merged
  tree: 0 red / 3 green (`logs/merge1-isolation.log`). It was not isolated on the base tree — a
  test that never reds alone on the merged tree leaves nothing to compare.
- **Verdict against the card's done_when:** met — no red reproduces only on the merged tree.

## Final tree `aa7545d` — the combined gate for merges 2 and 3

G1 0 / 0; full Go `EXIT_TEST=0`, 456 / 0 / 3; full Playwright `EXIT_PW=1`, 23 failed / 5 flaky /
7 skipped / 1044 passed against tonight's base 24 / 2 / 7 / 1039; marketing 70 / 70. Two failed
tests outside base — `inventory.spec.js:2186` (1 red of 3 alone on BOTH trees, same message) and
`onboarding.spec.js:2233` (0 of 3 on both) — neither exclusive to the merged tree. `sw.js` 51,
reproduces. Detail and the isolation table: `HANDOFF.md` "Reds", `logs/final-isolation.log`.

**Summary:** three merges, one conflict (merge 1, `roadmap.md`, the card's own status line,
resolved as the slate prescribed), two clean. No merge needed a judgment between two cards'
behaviour; no card parked on a conflict. Two fix rounds after review, both wording only — nothing
to diff-check beyond the two docs/comment diffs and no mutation to re-run. The closeout commit
touches no evidence artefact a review signed (B-467): the card log directories are as merged.
