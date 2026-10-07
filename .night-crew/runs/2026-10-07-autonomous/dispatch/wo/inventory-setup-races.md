---
complexity: followup-batch
---
# WO: inventory-setup-races — what a manager types or adds in Setup stays on screen

## Intent

A manager who opens Inventory Setup and starts typing keeps what they typed when the
page's own loads finish; a nickname they add never disappears because an older response landed
late; an empty-name create says so. The Inventory Setup races stop being a crew-visible bug and
a test flake.

## Spec

In `inventory.html`: (1) a module-level `ITEMS_SEQ` counter; every writer of
`ALL_ITEMS` / `ITEM_GROUPS` (`loadItems()`, the `DOMContentLoaded` preload, the alias handler's
refetch, and any other `api('/api/v1/inventory/items')` caller that assigns) captures `const seq
= ++ITEMS_SEQ` before its fetch and, on response, applies only if `seq >= ITEMS_APPLIED`, then
sets `ITEMS_APPLIED = seq` — last REQUEST wins; a discarded response is logged at debug level.
(2) The Setup add bar (`#new-item-name`, `#new-item-group`) is rendered ONCE per Setup open;
when groups land, only the `<select>`'s options are rebuilt, preserving the selected value and
leaving the input untouched. (3) The create click with an empty name calls the same alert the
no-group path uses, naming the name field ("Enter an item name first"). (4) **The add-nickname
path is made robust while the item list's first fetch is in flight:** the night reproduces the
dropped add with the spike's harness (`spike-k1.spec.js` leg 3 — the first `GET /items` held,
the add clicked; 1 of 2 runs the POST never happened), names the mechanism it finds (a
re-render that detaches the form's handler, a guard on an in-flight load, a lost click target —
whatever the evidence says), and fixes THAT; a fix that merely retries the POST is not a fix.
No other behaviour changes. `sw.js` regenerated (51).

## Gates

Red-first, shown: `[IS-01]` (the spike's leg 1 re-homed into `tests/inventory.spec.js`
with the expectation inverted — groups delayed 600 ms, the typed name survives the response and
the create click POSTs) RED on the pre-change tree; `[IS-02]` through `window` (expose
`ITEMS_SEQ`/`ITEMS_APPLIED` read-only on a debug object the way `MarketingScan` exposes state):
two `GET /items` responses released out of order leave `ALL_ITEMS` at the NEWER one — RED today
(no sequence exists); `[IS-03]` empty-name create → dialog text contains "name" — RED today
(silent); `[IS-04]` with the Setup tab's first `GET /items` held, a nickname added through the UI
has its `POST /items/aliases` observed and its chip rendered, 5× in a row — RED today
(nondeterministic: the spike's run 3 dropped it; the spec runs the scenario five times and
all five must persist). Then the **measurement leg**: `npx playwright test tests/inventory.spec.js -g "Setup
item editor shows alias chips"` 10× alone and the two add-bar tests (`:2919`, `:2931`) 10× alone,
then the `inventory|recipes` seam 5×, all on the merged tree, tallies recorded in HANDOFF —
`bugs.md`'s three entries are retired ONLY if 0 red; otherwise the mechanism observed is reported
and the entries stay. G1; `sw.js` 51; the **full Playwright suite** under the lock (precache
moved). G6 runs `[IS-01]` itself with the delay raised to 1500 ms.

The loop itself executes only the fenced lines below (compile and vet). Every test clause above is proven by the session's recorded red/green evidence, re-run by the orchestrator's separate adversarial review, and by the orchestrator's suites on the merged tree under the lock.

```
go build -C backend ./...
go vet -C backend ./...
```

## Context

the three writers are at `inventory.html:2179` (`loadItems`), `:2835` (preload) and
`:2550` (alias refetch); the add bar is built at `:2273` and read at `:2367`/`:2465`. The spike's
`spike-k1.spec.js` is the working draft for `[IS-01]`. The flaky tests are the MEASUREMENT, do
not edit them. Box rules, B-486 checkout rule (until K3 lands) apply.

DONE-WHEN (the card's Gates section — these are the clauses you must prove and report on; references to G6, the full Playwright suite, the 10×/5× measurement leg and `sw.js` are the orchestrator's legs): Red-first, shown: `[IS-01]` (the spike's leg 1 re-homed into `tests/inventory.spec.js`
with the expectation inverted — groups delayed 600 ms, the typed name survives the response and
the create click POSTs) RED on the pre-change tree; `[IS-02]` through `window` (expose
`ITEMS_SEQ`/`ITEMS_APPLIED` read-only on a debug object the way `MarketingScan` exposes state):
two `GET /items` responses released out of order leave `ALL_ITEMS` at the NEWER one — RED today
(no sequence exists); `[IS-03]` empty-name create → dialog text contains "name" — RED today
(silent); `[IS-04]` with the Setup tab's first `GET /items` held, a nickname added through the UI
has its `POST /items/aliases` observed and its chip rendered, 5× in a row — RED today
(nondeterministic: the spike's run 3 dropped it; the spec runs the scenario five times and
all five must persist). Then the **measurement leg**: `npx playwright test tests/inventory.spec.js -g "Setup
item editor shows alias chips"` 10× alone and the two add-bar tests (`:2919`, `:2931`) 10× alone,
then the `inventory|recipes` seam 5×, all on the merged tree, tallies recorded in HANDOFF —
`bugs.md`'s three entries are retired ONLY if 0 red; otherwise the mechanism observed is reported
and the entries stay. G1; `sw.js` 51; the **full Playwright suite** under the lock (precache
moved). G6 runs `[IS-01]` itself with the delay raised to 1500 ms.

Slate lead (mechanism-of-record pointers): Roadmap Activity K card K1 (the card's "Spike-corrected scope" paragraph is the mechanism of
record); BACKLOG B-459, B-478; triage T-66 (receipt `reference/triage-20261005.md`); goal ledger
`spikes/activity-k-…/inventory-setup-races.md` (**binding build-facts** — leg 1: groups delayed
600 ms → `{"before":"Spike Typed Item","after":"","posts":0,"editForms":0}`; control after
networkidle keeps the name; leg 3, two runs under the filed B-459 timing: view and server AGREE
both times — run 2 both hold the nickname, run 3 neither does, the add having never reached the
server) and its extraction record (one signed correction).

PARK note (narrow, operator-only — if you meet one of these, STOP and report it as a park; never decide it): changing what Setup shows or does beyond these three
behaviours (a redesign of the add bar, a new field); a `night-crew.toml` key. The counter's
name, the debug-object shape and the alert copy are the night's.

RUN RULES — run 20261007, repo yumyums/hq (binding; from the signed launch prompt `.night-crew/knowledge/reference/launch-20261007.md` and the triage receipt `.night-crew/knowledge/reference/triage-20261006.md`, both of which you may read in full):

Authority and limits. Batch sign-off was given by the operator on 2026-10-05; do not pause for per-change sign-off. Never push, never tag, never deploy, never touch `main`. You never choose a product fork — what to build, what a milestone means. If the card cannot be built as specified without one, STOP and report it as a park with the question written out; do not improvise. Editing a file outside the card's footprint is NOT a park and not a breach — say so in the merge-intent note. NO MIGRATIONS tonight: a card that adds one is scope drift, stated and not shipped.

How your work is recorded (this night rides night-crew's run loop). You are working in a loop-owned git worktree on a loop-owned branch. Do NOT run `git commit`, do not create or switch branches, do not stash: leave every change in the working tree and the loop commits your whole diff as one commit when you report through the result contract. The commit-level mechanics (the `Night-Crew-Run: 20261007` trailer, the merge) are applied by the orchestrator, not by you. A park is reported through the result contract as `halt`, with the operator question written out in the summary. You cannot spawn subagents for review; the separate adversarial review is the orchestrator's.

Merge-intent note (REQUIRED, the FIRST file you write, before implementing): write `.night-crew/runs/2026-10-07-autonomous/merge-intents/inventory-setup-races.md` — the shared files you will touch (each file outside your own area, one line of why), what must survive any merge, what is safe to drop. Empty fields say "nothing here" explicitly. It also states, per done-when clause, which rows ride a stub or fixture: tonight's permitted stubs are network TIMING (`page.route` delays and holds) and, for the scanner card, the sync door's `page.route` mock — nothing else. A stub of the behaviour under test is a defect.

Per-change mechanics (this repo has NO OpenSpec — create no `openspec/` scaffolding). Red-first evidence: show each named test RED on the pre-change tree (the done-when text names the spike recipe to use as the red) and then GREEN; save both outputs, with the exact commands and exit lines, under `.night-crew/runs/2026-10-07-autonomous/logs/inventory-setup-races/` (they ride your diff). Flip this card's Activity K line in `.night-crew/knowledge/roadmap.md` from `PLANNED` to `LANDED` (the orchestrator adds the merge SHA); otherwise roadmap.md is append-only. Do NOT edit `.night-crew/knowledge/BACKLOG.md` or `bugs.md` — the orchestrator flips those at its merge. No `night-crew.toml` key or token (that is a park). Write nothing under `.night-crew/qa/` (the loop refuses the commit).

Service worker. `sw.js` is generated from git HEAD, so it CANNOT be regenerated correctly from an uncommitted tree: do NOT hand-edit it and do NOT include `sw.js` or `version.json` in your diff — if a test target rewrote them, run `git checkout -- sw.js version.json` before you report. The orchestrator regenerates `sw.js` at the merged HEAD and checks the precache count is still 51. You add or remove no precached file.

Suites. You run the CONFINED gates named in your done-when text: your new tests, the spec file(s) you touched, and the seam named for your card. You do NOT run the full Playwright suite (55 min), and you do not run the 10×/5× measurement leg — the orchestrator runs both on the merged tree under the lock; say plainly in your report that they are owed. If your card changes Go code, run ONE full Go suite (`go test ./... -p 1 -count=1`) under the lock and report counts.

Box rules. `export PATH="/usr/local/go/bin:$PATH"` before every Go or Playwright leg (without it Playwright's webServer dies `go: not found` / exit 127, which is not a test failure). 🛑 Tests run on Postgres port 5434 (`yumyums-test-pg`, role `hqtest`, password `hqtest`; coordinates via `task test:targets`) — NEVER 5433, which is the dev AND production cluster; no suite, probe or psql may point at 5433. Use your own databases and port: `TEST_PORT=8281` and `TEST_DB_NAME=hq_test_e2e_k1_20261007` for Playwright (with `DB_HOST=localhost DB_PORT=5434 DB_USER=hqtest DB_PASS=hqtest`); Go database `hq_test_go_k1` on :5434 for `DB_TEST_URL`. `go test ./... -p 1` — `-p 1` is load-bearing; `DB_TEST_URL` MUST be set or the Go suite exits 0 while skipping every DB test: report test counts, not `ok`. 🛑 `internal/recipes` tests need a MIGRATED database: boot the built server once against the fresh Go database until the log says "database migrations applied successfully", then stop it — the recipe is `migrate_go_db` in `.night-crew/spikes/activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/_golib.sh`. Only ONE full suite runs on the box at a time: every full Playwright suite AND every full Go suite runs under `flock /tmp/hq-full-suite.lock` (another full suite — the orchestrator's base measurement — may be holding it; waiting is normal); a `TestRowVisibilityRLS` red is re-run alone before it is reported, both exit lines stated. 🛑 B-477: in a fresh worktree make THREE symlinks before the first Playwright leg — `node_modules`, `marketing/sync/harness/node_modules`, `.night-crew/qa/spike-supabase/rxdb/node_modules` — each pointing at the main checkout's (`/home/jcole/projects/hq/...`); run `npx bddgen` once; then warm the backend build (`cd backend && go build -o /dev/null ./cmd/server/`) BEFORE Playwright's 60 s webServer window. If `webServer` wedges, hand-provision the stack and point `NIGHTCREW_ENV_URL` at it — first resort, not last. 🛑 B-486 / B-493: after EVERY Playwright leg and before you report, run `git checkout -- .night-crew/runs/2026-10-02-autonomous/logs/h4/states/ .night-crew/runs/2026-10-02-autonomous/logs/h5/states/` — two states specs rewrite committed PNGs there; they must never appear in your diff (the screenshots card removes the cause; until it has merged the rule binds every card, including that one for anything its own change does not cover). Never use `git add`. The local `spike-supabase` substrate is up and is read-only to you: never `--fresh`, never tear it down. 🛑 Do not call `docker info` on this box tonight — it hangs indefinitely in a Docker Desktop CLI plugin; `docker ps` / `docker inspect` / `docker compose` work.

Report honestly: for each done-when clause say what you RAN and what you OBSERVED (command, exit line, counts), which clauses ride a stub or fixture, and anything you did not run. A clause you could not prove is reported as unproven, never as passed.

## Result

(reserved for the loop — sessions report through the result contract)
