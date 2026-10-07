# Merge intent — states-screenshots-out-of-tree (K3, run 20261007)

Card: running the marketing states specs stops leaving the tree dirty (BACKLOG B-486 — the
Stats-tab states spec rewrites committed PNGs on every run; B-493 — the Subscribers-tab states
spec does the same to a second directory; triage T-66 and the 20261006 triage launch note).

Behaviour after this card: a gate leg that includes the `marketing` seam leaves `git status`
exactly as it found it. Screenshots land in the ignored `test-screenshots/` unless a run sets
`STATES_SHOT_DIR`. The reviewed H4 and H5 PNG sets stay byte-for-byte as committed.

## Shared files I touch (outside my own area)

- `tests/states-marketing-stats.spec.js` — the card's subject. The `SHOT_DIR` constant, the
  header-comment paragraph that said "COMMITTED", and ONE new test `[SS-01]` in its own
  `test.describe` at the end. None of the 16 existing rows is edited.
- `tests/states-marketing-subscribers.spec.js` — the same three edits (constant, comment, one
  new `[SS-02]` test), per the triage launch note (B-493). None of its 9 existing rows is edited.
- `night-crew.toml` — COMMENT ONLY, in the marketing seam's roll call: a note that the two
  states specs no longer write under a tracked directory. No key, no token, no count change.
- `.night-crew/knowledge/roadmap.md` — one word on this card's Activity K line,
  `PLANNED` → `LANDED`. Append-only otherwise.

## What must survive any merge

- Both `SHOT_DIR` constants reading `process.env.STATES_SHOT_DIR || <DEFAULT_SHOT_DIR>`, with
  the defaults `test-screenshots/marketing-stats` and `test-screenshots/marketing-subscribers`.
  A merge that restores either hard-coded `.night-crew/runs/2026-10-02-autonomous/logs/h{4,5}/states`
  path brings the dirty tree back.
- The `[SS-01]` and `[SS-02]` tests (the default directory holds no git-tracked file and is
  gitignored). They are what turns the regression red.
- The committed PNGs under `.night-crew/runs/2026-10-02-autonomous/logs/h4/states/` (23 files)
  and `.../logs/h5/states/` (13 files) exactly as they are at the base. This card does not
  modify, move or re-capture any of them (B-467: they are never "restored" either — they are
  untouched by construction).

## What is safe to drop

- Nothing in the code change. The raw logs under
  `.night-crew/runs/2026-10-07-autonomous/logs/states-screenshots-out-of-tree/` are evidence,
  not behaviour.

## Stubs and fixtures, per done-when clause

| Clause | Rides a stub or fixture? |
|---|---|
| `[SS-01]` / `[SS-02]` red-first, then green | No stub. A real `git ls-files` and `git check-ignore` against the real repository. |
| Run each spec alone → `git status --porcelain` adds nothing | No stub. Real Playwright run, real server, real database on :5434. |
| The 16 Stats rows (and the 9 Subscribers rows) still pass | Unchanged by this card: the rows' own pre-existing `page.route` fixtures (loading / error / offline / long content / sheets, listed in each spec's header) are as they were. This card adds none. |
| `git diff --stat` empty for `h4/states/` and `h5/states/` after the run | No stub. |
| `marketing` seam confined | Same as above — pre-existing fixtures only. |
| Full Playwright suite, 10×/5× measurement leg, `sw.js` 51, G6 | The orchestrator's — not run by this card. |

No stub of the behaviour under test. No migration. No Go change. No `night-crew.toml` key or
token. `sw.js` / `version.json` are not in the diff. No PNG is in the diff.

## Notes for the merge

- Nothing outside the card's footprint as widened by the triage launch note was edited.
- Once this card has merged, the standing "run `git checkout -- …/h4/states/ …/h5/states/` after
  every Playwright leg" rule (B-486 / B-493) has no cause left for these two specs.
