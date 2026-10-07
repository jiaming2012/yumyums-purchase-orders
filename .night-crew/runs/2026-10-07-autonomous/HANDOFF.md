# HANDOFF — overnight run `20261007` (night of 2026-10-06 → morning 2026-10-07)

**Branch:** `overnight-20261007`, cut from `dev` at `c549c51`. Nothing pushed, `main` untouched, no
deploy, no migration. **Slate:** `reference/slate-20261007.md`, signed 2026-10-05 — 4 cards,
Activity K, serial, on night-crew's run loop. Launched 01:00, run submitted 01:18, loop run closed
04:57, final suites done 06:00, closeout written ~06:25 America/New_York.

## Read these five things first

1. **4 of 4 cards landed.** Each rode the loop, passed a separate adversarial review that mutated
   the code and watched the new tests go red, and is merged on `overnight-20261007`. Nothing parked.
2. **What changes for the crew:** in Inventory Setup, a name typed into the add bar and a nickname
   added to an item now stay put while the page finishes loading, and Add with an empty name says
   so. Everything else tonight is API behaviour and test hygiene — no other screen changes.
3. **The Inventory Setup flake is measured gone:** the three tests were 30 of 30 green alone and red
   in 0 of 5 seam repeats. The cause of the dropped nickname was not the one filed — a broken
   catalog image was re-rendering the whole list about every 70 ms and emptying the box.
4. **Card 4 was built narrowed, per your decision 214:** only the refusal gate; no photo-pick
   feedback. Its review raises one thing for you: the test hook it adds is not limited to test
   environments (see "For triage").
5. **The final browser suite exits 1, as the base does** — 22 failed against the base's 25, and
   **no test is red on the final tree that is not red on untouched `dev`**. The full Go suite exits 0.

## Per-card outcomes

| # | Card | Outcome | Loop commit → merge on `overnight-20261007` | Build + loop review |
|---|---|---|---|---|
| 1 | `inventory-setup-races` | **LANDED** | `3fb748a` → `b9d7b62` | 01:20 → 02:51 (91 m) |
| 2 | `dish-merge-shapes-and-backstop-tests` | **LANDED** | `bc705a5` → `05c9ccd` | 02:51 → 03:32 (40 m) |
| 3 | `states-screenshots-out-of-tree` | **LANDED** (covers `h4` and `h5`) | `45f42a4` → `e5563ab` | 03:32 → 04:22 (50 m) |
| 4 | `scanner-refusal-seam-and-pick-feedback` | **LANDED, narrowed** (B-482 only) | `cbf5598` → `16090f7` | 04:22 → 04:48 (26 m) |

Roadmap Activity K: four lines `LANDED` with SHAs. Backlog: B-459, B-478, B-482, B-484, B-485,
B-486, B-487 → `landed`; `backlog check` exit 0 after each. `night-crew.toml`: 8 comment lines.

## Gate evidence on the final tree (`4a93ab5`)

| Leg | Result |
|---|---|
| Go, full, `-p 1`, migrated database, under the lock | `EXIT_TEST=0` — 461 pass / 0 fail / 3 skip, 15 packages (base 456 / 0 / 3) |
| Browser suite, full, config's 1 retry, under the lock | `EXIT_PW=1` — 22 failed / 2 flaky / 6 skipped / 1056 passed, 42.3 m |
| Red on the final tree and NOT on tonight's base | **none** |
| Red on the base and green on the final tree | 3 — create new item via Items tab; creating item opens edit form; set store location from Setup |
| Flaky on the final tree | `onboarding.spec.js:696`, `:2233` — both 5 of 5 green alone on both trees |
| `git status` after the final suite | **empty** — Card 3 holds; no screenshot restore was needed |
| Precache | 51 at launch, 51 after Card 1's and Card 4's `sw.js` regeneration, each idempotent |
| Build + vet on the merged tree | exit 0 |
| Separate adversarial review | PASS on all four (details per merge in `reference/conflicts-20261007.md`) |

**Base (untouched `dev@c549c51`):** Go 456 / 0 / 3, exit 0. Browser 25 failed / 2 flaky / 6 skipped /
1046 passed, 42.4 m, exit 1. Against last night's base: `onboarding:2233` and `persistence:1463`
green, `inventory:2919` flaky instead of red, `inventory:2931` red instead of green.

**Card 1's own full suite** (its merged tree `fb4badb`): 23 failed / 0 flaky / 1054 passed, exit 1;
one red not in tonight's base — `onboarding:2233`, since shown 5 of 5 green alone on both trees.

## The Inventory measurement (Card 1's merged tree, no retries)

| Test | Alone | In the `inventory\|recipes` seam |
|---|---|---|
| Setup item editor shows alias chips (`:2186`) | 10 of 10 green | red in 0 of 5 |
| create item without group shows alert (`:2919`) | 10 of 10 green | red in 0 of 5 |
| creating item opens edit form with store location dropdown (`:2931`) | 10 of 10 green | red in 0 of 5 |

Seam totals over five repeats: 16 / 15 / 17 / 17 / 15 failed of 268. Fifteen tests are red every
time — the standing Inventory cluster, untouched and still undiagnosed. Three others wobble:
`inventory:2112` (1 of 5), `recipes:216` (2 of 5), `states-inventory-nav:288` (2 of 5).
**Isolation, 5× alone on the final tree and on the base:** `states-inventory-nav:288` is red 3 of 5
on **both** trees — an existing flake, not tonight's; the other two are 5 of 5 green on both.
`bugs.md`'s `:2186` section carries the retirement note. The seam repeats ran while other suites
were on the box, so the wobble rates are under load.

## For triage — findings, none blocking, to file or drop

- **The scanner's test hook is honoured on any device.** A script on the page that sets
  `window.__MARKETING_POLICY_SOURCE__` before boot can turn refusals into offers for that page
  load. The signed card asked for exactly this read and an existing setter already allows the same
  after boot; the reviewer rates it low–medium. Closing it is one small card (accept the hook only
  on localhost or behind a test-server flag).
- **A photo that fails to load is cleared, and the next Save stores no photo** (Inventory Setup,
  rated major by the loop's review). Exists on `dev` today; not introduced tonight.
- **Merging two dishes that share an ingredient answers 500** (major). Exists today; API-only.
- **Dish ids without hyphens or in braces now answer 400** where the database used to accept them
  (deliberate, pinned). No screen calls this endpoint.
- **Card 1 is wider than its four items:** every field of an open item editor now survives a
  re-render, and a purchase-order prefill is applied once per Setup open.
- **Two tests pin less than they claim:** `[IS-01]` does not pin "picked group kept when groups
  land"; the in-place image swap is not pinned alone.
- Twenty deferred review findings are in `ledger.md` as `## LDG 20261007/…` entries, written there
  by the loop. That file is this repo's decision record; whether loop findings belong in it is yours.
- `states-inventory-nav:288` fails about half the time alone on `dev`.

## How the night ran — and what the tooling did

- **Own scheduler lane again** (`night-crew-hq`): the night-crew clone signed its own slate
  `20261007` the same day. Worker built from night-crew `8c42997`, stopped at 05:00. Both lanes
  showed no pollers at 06:13 (`logs/workers-check-close.log`).
- **The loop carried a night where cards land.** One `night-crew run`, four work orders, width 1,
  8 h deadline; `RUN_EXIT=0`; `loop check` exit 0, "all 4 card(s) accounted for on the loop".
  Scorecard record committed; `night-crew scorecard --repo .` exit 0.
- **The loop reviews each card itself** (two lenses, two rounds, automatic patches) before it
  commits. On every card it ended "review cycle exhausted — last patch not re-reviewed"; my
  separate review covered those last patches. On Card 1 the loop's second round caught a real
  regression its own first patch had introduced (an unsaved photo lost on Save) and fixed it.
- **The loop's own end-of-run browser stage did not run.** It builds a Docker stack; the build
  hung in a Windows credential helper. I ended the helper and the build then failed, so the loop
  recorded "failed: environment". No test verdict came from the loop; the final suite above is the gate.
- **The same class of hang stalled `env-up.sh` for ~10 minutes at launch** (`docker info`). About
  twenty stale `powershell.exe` helpers from other sessions are still on the box; I left them.
- **The loop lands cards as squash commits on its own branch**, so merging a later card's commit
  replays the earlier card's documents: Card 2 conflicted in `roadmap.md` and `ledger.md`
  (resolved, logged). Cards 3 and 4 were brought over as their own change only.
- **The measurement leg took 105 minutes, not the slated ~20** — each solo run boots a server.

## Engineer-level calls made tonight, listed so they can be overturned

- The three stranded items the launch checks named were not re-asked: `card/a3-rls-fixture-own`,
  `card/s2-demo-sync-target` (your ruling, B-442) and last night's parked build at `fe45768`.
- Card 4 lifted narrowed (decision 214); Card 3 extended to the second screenshot directory
  (triage's B-493 note). Cards 1–3's Intent / Spec / Gates otherwise verbatim.
- Loop gates were compile + vet; test clauses rode the session's evidence, my review and my suites.
- The final suite ran while Card 1's seam repeats were still on the box, to save an hour.
- Ended two hung Windows helper processes, both children of my own commands.

## Left in place, deliberately (nothing was deleted)

- Loop branches `night-crew/run/20261007` and the four `night-crew/20261007/<card>` branches; the
  loop's merge worktree under `.night-crew/runs/20261007/`.
- Scratch worktrees under `hq-worktrees/`: `base-20261007`, `k1-merged-20261007`,
  `final-20261007`, `g6-k1-20261007`, `g6-k1-base-20261007`, `g6-k2-20261007`, `g6-k3-20261007`,
  `g6-k4-20261007`, `g6-k4-base-20261007` (one holds an untracked probe spec).
- Test databases on :5434 named `hq_test_*_20261007`, `hq_test_go_base`, `hq_test_go_final`,
  `hq_test_go_g6k2*`. **:5433 was never touched.**
- Carried unchanged: the two long-stranded card branches; `fe45768`; namespace `night-crew-hq`.

## Next actions

1. `/nc-morning-triage` — review and merge `overnight-20261007`; file or drop the findings above.
2. Decide whether the scanner test hook gets a guard before the next deploy.
3. Activity E is untouched by tonight and still waits on its next slate (B-496, then the two cards).
