---
complexity: small-additive
---
# WO: states-screenshots-out-of-tree — the states specs stop dirtying the tree

## Intent

Running the marketing-stats states spec never leaves the tree dirty, so gate legs
and merges stop needing a checkout rule, and the committed H4 evidence stays exactly as reviewed.

## Spec

In `tests/states-marketing-stats.spec.js`: `const SHOT_DIR = process.env.STATES_SHOT_DIR
|| path.join(__dirname, '..', 'test-screenshots', 'marketing-stats');` and a header comment
replacing the "COMMITTED" paragraph: screenshots default to the ignored `test-screenshots/`; a run
that wants durable evidence sets `STATES_SHOT_DIR` to its own logs tree and commits there (the
`.gitignore` convention); the H4 set under `.night-crew/runs/2026-10-02-autonomous/logs/h4/states/`
is the reviewed evidence and is NOT rewritten. `night-crew.toml`'s marketing roll-call comment
notes the change (no key, no token). No PNG is modified or moved.

## Gates

Red-first: a static assertion in the spec's own file (a `test('[SS-01] SHOT_DIR is
untracked')` that runs `git ls-files --error-unmatch` on the resolved dir's parent, or a Node
check) RED today, green after; run the spec alone in a fresh worktree → `git status --porcelain`
EMPTY (the spike's leg a, inverted); the 16 rows still pass; `git diff --stat --
.night-crew/runs/2026-10-02-autonomous/logs/h4/states/` empty after the run. `marketing` seam
confined; the final suite is its full gate. G6 runs the spec once and checks `git status`.

The loop itself executes only the fenced lines below (compile and vet). Every test clause above is proven by the session's recorded red/green evidence, re-run by the orchestrator's separate adversarial review, and by the orchestrator's suites on the merged tree under the lock.

```
go build -C backend ./...
go vet -C backend ./...
```

## Context

the spike's `01-…sh` is the recipe for the clean-tree check. B-467: never
"restore" the H4 PNGs — they are untouched by construction. Box rules apply.

TRIAGE LAUNCH NOTE (B-493, triage receipt 20261006 — this card covers BOTH directories): a second states spec, `tests/states-marketing-subscribers.spec.js` (its directory constant near line 51), rewrites eight committed PNGs under `.night-crew/runs/2026-10-02-autonomous/logs/h5/states/` the same way. Apply the SAME change to it: default to the ignored `test-screenshots/` (its own subdirectory), honour `STATES_SHOT_DIR`, same header-comment treatment, the same kind of static untracked-path assertion, and prove `git diff --stat -- .night-crew/runs/2026-10-02-autonomous/logs/h5/states/` is empty after running it. The committed h5 set, like h4, is NOT rewritten, moved or re-captured. Also grep `tests/states-*.spec.js` for any other spec writing under a tracked directory and report what you find (fix it the same way only if it is the same one-constant shape; otherwise report it and leave it).

DONE-WHEN (the card's Gates section — these are the clauses you must prove and report on; references to G6, the full Playwright suite, the 10×/5× measurement leg and `sw.js` are the orchestrator's legs): Red-first: a static assertion in the spec's own file (a `test('[SS-01] SHOT_DIR is
untracked')` that runs `git ls-files --error-unmatch` on the resolved dir's parent, or a Node
check) RED today, green after; run the spec alone in a fresh worktree → `git status --porcelain`
EMPTY (the spike's leg a, inverted); the 16 rows still pass; `git diff --stat --
.night-crew/runs/2026-10-02-autonomous/logs/h4/states/` empty after the run. `marketing` seam
confined; the final suite is its full gate. G6 runs the spec once and checks `git status`.

Slate lead (mechanism-of-record pointers): Roadmap Activity K card K3; BACKLOG B-486; triage T-66; goal ledger
`spikes/activity-k-…/states-screenshots-out-of-tree.md` (**binding build-facts** — 16 rows pass,
12 of 23 tracked PNGs modified after one run; `test-screenshots/` already ignored; `SHOT_DIR`
hard-coded) and its extraction record (no corrections).

PARK note (narrow, operator-only — if you meet one of these, STOP and report it as a park; never decide it): moving or re-capturing the committed H4 set; a
`night-crew.toml` key or token; changing which rows the spec forces. The default path's exact
name is the night's.

RUN RULES — run 20261007, repo yumyums/hq (binding; from the signed launch prompt `.night-crew/knowledge/reference/launch-20261007.md` and the triage receipt `.night-crew/knowledge/reference/triage-20261006.md`, both of which you may read in full):

Authority and limits. Batch sign-off was given by the operator on 2026-10-05; do not pause for per-change sign-off. Never push, never tag, never deploy, never touch `main`. You never choose a product fork — what to build, what a milestone means. If the card cannot be built as specified without one, STOP and report it as a park with the question written out; do not improvise. Editing a file outside the card's footprint is NOT a park and not a breach — say so in the merge-intent note. NO MIGRATIONS tonight: a card that adds one is scope drift, stated and not shipped.

How your work is recorded (this night rides night-crew's run loop). You are working in a loop-owned git worktree on a loop-owned branch. Do NOT run `git commit`, do not create or switch branches, do not stash: leave every change in the working tree and the loop commits your whole diff as one commit when you report through the result contract. The commit-level mechanics (the `Night-Crew-Run: 20261007` trailer, the merge) are applied by the orchestrator, not by you. A park is reported through the result contract as `halt`, with the operator question written out in the summary. You cannot spawn subagents for review; the separate adversarial review is the orchestrator's.

Merge-intent note (REQUIRED, the FIRST file you write, before implementing): write `.night-crew/runs/2026-10-07-autonomous/merge-intents/states-screenshots-out-of-tree.md` — the shared files you will touch (each file outside your own area, one line of why), what must survive any merge, what is safe to drop. Empty fields say "nothing here" explicitly. It also states, per done-when clause, which rows ride a stub or fixture: tonight's permitted stubs are network TIMING (`page.route` delays and holds) and, for the scanner card, the sync door's `page.route` mock — nothing else. A stub of the behaviour under test is a defect.

Per-change mechanics (this repo has NO OpenSpec — create no `openspec/` scaffolding). Red-first evidence: show each named test RED on the pre-change tree (the done-when text names the spike recipe to use as the red) and then GREEN; save both outputs, with the exact commands and exit lines, under `.night-crew/runs/2026-10-07-autonomous/logs/states-screenshots-out-of-tree/` (they ride your diff). Flip this card's Activity K line in `.night-crew/knowledge/roadmap.md` from `PLANNED` to `LANDED` (the orchestrator adds the merge SHA); otherwise roadmap.md is append-only. Do NOT edit `.night-crew/knowledge/BACKLOG.md` or `bugs.md` — the orchestrator flips those at its merge. No `night-crew.toml` key or token (that is a park). Write nothing under `.night-crew/qa/` (the loop refuses the commit).

Service worker. `sw.js` is generated from git HEAD, so it CANNOT be regenerated correctly from an uncommitted tree: do NOT hand-edit it and do NOT include `sw.js` or `version.json` in your diff — if a test target rewrote them, run `git checkout -- sw.js version.json` before you report. The orchestrator regenerates `sw.js` at the merged HEAD and checks the precache count is still 51. You add or remove no precached file.

Suites. You run the CONFINED gates named in your done-when text: your new tests, the spec file(s) you touched, and the seam named for your card. You do NOT run the full Playwright suite (55 min), and you do not run the 10×/5× measurement leg — the orchestrator runs both on the merged tree under the lock; say plainly in your report that they are owed. If your card changes Go code, run ONE full Go suite (`go test ./... -p 1 -count=1`) under the lock and report counts.

Box rules. `export PATH="/usr/local/go/bin:$PATH"` before every Go or Playwright leg (without it Playwright's webServer dies `go: not found` / exit 127, which is not a test failure). 🛑 Tests run on Postgres port 5434 (`yumyums-test-pg`, role `hqtest`, password `hqtest`; coordinates via `task test:targets`) — NEVER 5433, which is the dev AND production cluster; no suite, probe or psql may point at 5433. Use your own databases and port: `TEST_PORT=8283` and `TEST_DB_NAME=hq_test_e2e_k3_20261007` for Playwright (with `DB_HOST=localhost DB_PORT=5434 DB_USER=hqtest DB_PASS=hqtest`); Go database `hq_test_go_k3` on :5434 for `DB_TEST_URL`. `go test ./... -p 1` — `-p 1` is load-bearing; `DB_TEST_URL` MUST be set or the Go suite exits 0 while skipping every DB test: report test counts, not `ok`. 🛑 `internal/recipes` tests need a MIGRATED database: boot the built server once against the fresh Go database until the log says "database migrations applied successfully", then stop it — the recipe is `migrate_go_db` in `.night-crew/spikes/activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/_golib.sh`. Only ONE full suite runs on the box at a time: every full Playwright suite AND every full Go suite runs under `flock /tmp/hq-full-suite.lock` (another full suite — the orchestrator's base measurement — may be holding it; waiting is normal); a `TestRowVisibilityRLS` red is re-run alone before it is reported, both exit lines stated. 🛑 B-477: in a fresh worktree make THREE symlinks before the first Playwright leg — `node_modules`, `marketing/sync/harness/node_modules`, `.night-crew/qa/spike-supabase/rxdb/node_modules` — each pointing at the main checkout's (`/home/jcole/projects/hq/...`); run `npx bddgen` once; then warm the backend build (`cd backend && go build -o /dev/null ./cmd/server/`) BEFORE Playwright's 60 s webServer window. If `webServer` wedges, hand-provision the stack and point `NIGHTCREW_ENV_URL` at it — first resort, not last. 🛑 B-486 / B-493: after EVERY Playwright leg and before you report, run `git checkout -- .night-crew/runs/2026-10-02-autonomous/logs/h4/states/ .night-crew/runs/2026-10-02-autonomous/logs/h5/states/` — two states specs rewrite committed PNGs there; they must never appear in your diff (the screenshots card removes the cause; until it has merged the rule binds every card, including that one for anything its own change does not cover). Never use `git add`. The local `spike-supabase` substrate is up and is read-only to you: never `--fresh`, never tear it down. 🛑 Do not call `docker info` on this box tonight — it hangs indefinitely in a Docker Desktop CLI plugin; `docker ps` / `docker inspect` / `docker compose` work.

Report honestly: for each done-when clause say what you RAN and what you OBSERVED (command, exit line, counts), which clauses ride a stub or fixture, and anything you did not run. A clause you could not prove is reported as unproven, never as passed.

## Result

(reserved for the loop — sessions report through the result contract)
