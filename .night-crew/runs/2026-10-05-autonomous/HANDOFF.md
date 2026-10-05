# HANDOFF — overnight run `20261005` (night of 2026-10-04 → morning 2026-10-05)

> **TRIAGED 2026-10-05 — merged to `dev` at `57a2083`; ledger T-66 (decision 206); receipt
> `reference/triage-20261005.md`.** Standing flags after triage:
> - **Run not driven through night-crew's run loop — RESOLVED** (decision 206): the next launch drives the loop. Re-arms if that night's closeout reports `loop check` refused.
> - **`atomic-scan-dedupe` full browser-suite run — cleared**: ran on the merged tree (merge 1's owed gate) and the triage reviewer's `marketing|recipes` subset.
> - **The two un-gated behaviours under "Review findings" — filed**, B-482 (refusal screen under a throwing policy source) and B-483 (silent photo drop); the lock placement as B-484, the two error shapes as B-485, the schema-only `ON DELETE SET NULL` assertion as B-487.
> - **Count dangling `qr_scans.subscriber_id` on production before the deploy carrying `0086` — ARMED** (B-474). **`0087` removes past repeat-tap scan rows on first deploy — ARMED as a deploy note** (decision 203, shipped as built).
> - **`inventory.spec.js:2186` — unchanged**, B-459 (diagnosed race, no card yet); with B-478 it is the obvious Inventory Setup pairing for a coming slate.
> - **Private scratch path per agent** (process note, card-actuals lesson 3) — moot once the loop dispatches; re-arms only if a night runs off the loop again.

**Branch:** `overnight-20261005`, cut from `dev` at `588f188`. Nothing pushed, `main` untouched, no
deploy. **Slate:** `reference/slate-20261005.md`, signed 2026-10-04. **3 cards** — Activity I's
`atomic-scan-dedupe` and Activity J's two. Launched 22:02, closeout written ~01:10
America/New_York (~3 h 10 m, against the slate's 4 h 40 m – 5 h 45 m projection).

## 🛑 Read these five things first

1. **3 of 3 cards landed. Nothing is parked; there is no decision waiting for you.**
   `DECISIONS-NEEDED.md` is empty by content, not by omission.
2. **The final full browser suite on the complete tree is `EXIT_PW=1`: 23 failed against tonight's
   base of 24 — and two of the 23 did not fail in tonight's base.** Neither is attributed to
   tonight's work: `onboarding.spec.js:2233` passed alone 3/3 on both trees;
   `inventory.spec.js:2186` failed alone 1 of 3 on the final tree AND 1 of 3 on the base tree, with
   the same message. See "Reds". All 70 marketing tests passed on first attempt.
3. **The night was NOT run through night-crew's run loop, and the end-of-night `loop check`
   REFUSED** (`no run summary at .night-crew/runs/20261005/summary.json`). At launch the
   orchestrator ran the night as the saved prompt describes (a worktree and a session per card) on
   the strength of your answer to the same question at run 20261003's launch — **you were not asked
   again tonight.** This is that choice's stated consequence, not a clean pass. If you want the
   question asked every night, say so at triage.
4. **Card 2's fix is wider than the slate's wording, and one of its four gates is narrower than the
   slate assumed.** The busy guard also covers the scan-by-text entry point (no crew path needed
   that). And the "campaign policy source throws → refuse" case is gated only at the predicate
   level: the page's own refusal screen in that condition is tested by nothing, and the fallback the
   slate named (`campaigns-run.sh`) does not cover it either. Detail under "Review findings".
5. **Two deploy-time flags are unchanged and still yours:** count dangling
   `qr_scans.subscriber_id` rows on production before the deploy that carries migration `0086`
   (backlog B-474), and migration `0087` (landed tonight) removes past repeat-tap scan rows on first
   deploy — your decision 203, shipped as built.

## Per-card outcomes

| # | Card | Merge | Review | Fix round | Notes |
|---|---|---|---|---|---|
| 1 | `atomic-scan-dedupe` (I4) | `f980e12` | none new — reviewed in run 20261003; the merge needed no judgment | — | A merge, not a build. One conflict, the expected one (its own roadmap line). **Its owed full browser suite ran on the merged tree: 22 failed, none outside tonight's base.** Full Go 454 / 0 / 3 (+5 = the card's tests). |
| 3 | `merge-target-and-blanking-guards` (J2) | `42fbd67` | APPROVE-WITH-FINDINGS — 0 blocking, 3 record, 3 note | wording only (`dc36a51`) | Reviewer drove the merge over real HTTP: missing target → `404 target_not_found`, dish and sales row intact; self-merge still 400; valid merge still re-points; target deleted mid-request → 404. Both mutations re-run by the reviewer and red. |
| 2 | `photo-scan-guard-and-lookup-arms` (J1) | `b0bc48c` | APPROVE-WITH-FINDINGS — 0 blocking, 2 record, 3 note | wording only (`d6e7111`) | Reviewer reverted the page fix and watched all three photo specs red, re-ran two of the four mutations (each redded exactly its own spec), and ran the whole marketing spec: 70 / 0 / 0. **The other two mutations (`[SV-08]`, `[SV-09]`) were not re-run by the reviewer.** |

Both new cards got a fresh-context review on contract + diff + evidence. Both fix rounds changed
wording only (no test or product line), so there was no mutation to re-run after them.

## What changed for the crew

- **Scanner:** a photo picked while the phone is still checking a code with the server is now
  ignored until that check finishes, instead of leaving the customer's offer on screen with no
  submit button. The result area already reads "Checking with the server…" during that wait; there
  is no new message. The crew member re-picks the photo afterwards.
- **Public QR landing:** several taps of the same code from the same phone inside ten minutes now
  count as one scan even when they arrive at the same instant. Two taps that straddle a ten-minute
  boundary count twice (decision 195, as designed).
- **Dish merge (API-only — no screen calls it):** merging into a dish id that does not exist is
  refused with a 404 instead of silently deleting the source dish and its sales rows.
- Nothing else is crew-visible: the rest is tests.

## Gate evidence on the FINAL tree `aa7545d` (Cards 1 + 3 + 2)

| Gate | Result | Log |
|---|---|---|
| **G1** | `EXIT_BUILD=0`, `EXIT_VET=0` | `logs/final-G1-G2go.log` |
| **G2 (Go)** | **`EXIT_TEST=0` — 456 pass / 0 fail / 3 skip, 15 packages**, counts checked; `TestRowVisibilityRLS` 59 subtests; `HQ_SYNC_SUBSTRATE_OPTIONAL` + `HQ_SYNC_GATE_CHILD` unset in-log. Base 449 / 0 / 3; +5 Card 1, +2 Card 3 | `logs/final-G1-G2go.log` |
| **G2 (Playwright)** | **`EXIT_PW=1` — 23 failed / 5 flaky / 7 skipped / 1044 passed (51.4 m)**, 1079 tests, exactly one summary block, config retries (1). Marketing 70 / 70 green on first attempt | `logs/final-pw.log`, `logs/final-reds.txt` |
| **G3** | N/A — `openspec: absent`, re-confirmed by `workflow preflight` at launch | — |
| **G4** | **51 precached**, committed file reproduces on two runs, reachability 38 / 67 / 0 outside. 51 → 51 at every merge | `logs/final-G4-sw.log`, `logs/merge{1,2,3}-*.log` |
| **RF** | both code-changing cards showed their reds first (Card 3: 2 reds; Card 2: the pre-change photo red + four mutation reds) | per-card `logs/<card>/` |
| **G6** | 2 of 2 new cards reviewed in fresh context; both returned findings, none blocking | this file, conflict log |

There is no G5. The G4 discipline greps are N/A-VACUOUS in this repo (B-14).

## Reds — what is and is not known

**Base, measured tonight on `dev@588f188`:** Go clean (449 / 0 / 3). Playwright 24 failed / 2 flaky /
7 skipped / 1039 passed (52.4 m), 1072 tests. 23 of the 24 are in run 20261003's base list; the 24th
is `inventory.spec.js:2931` (the Setup add-bar race, B-478).
- 🛑 **The base run's `EXIT_PW` was not captured.** The final-gate agent overwrote the base agent's
  running wrapper script (same filename in a shared scratch directory). The suite had already
  printed its one summary block, so the counts stand; the exit code is inferred from "24 failed".
- **Tonight's base ran with the config's one retry; run 20261003's base ran with none.** The two
  base lists are not like-for-like. All three of tonight's full suites used the same setting, so
  tonight's comparisons are.

**Final tree, compared with tonight's base:**

| Test | In the full suite | Alone, final tree | Alone, base tree |
|---|---|---|---|
| `inventory.spec.js:2186` Setup item editor shows alias chips | **failed** (`toHaveCount` expected 2, received 1) | 1 red of 3 | 1 red of 3, same message |
| `onboarding.spec.js:2233` manager can reject a signed-off section | **failed** | 0 red of 3 | 0 red of 3 |
| `inventory.spec.js:1469` create new item via Items tab | flaky (green on retry) | 1 red of 3 | 0 red of 3 |
| `inventory.spec.js:2145` linking a receipt line learns an alias | flaky | 0 red of 3 | 0 red of 3 |
| `sw-api-cache-partition.spec.js:92` [B1-XT-01] | flaky | 0 red of 3 | 0 red of 3 |

- `inventory.spec.js:2186` and `:1469` were both in run 20261003's base red list; `:2186` reds
  alone at the same rate and with the same message on both trees. No card tonight changes
  `inventory.html` or its JS; Card 3 changes only the dish-merge function, which no screen calls.
- No test was red 2 of 3 or worse alone on the final tree, so the 5-run extension was not owed.
- **Three runs per tree is a small sample.** It shows these are not exclusive to tonight's tree; it
  does not show they are harmless. `onboarding.spec.js:2233` also failed only-in-suite on run
  20261003.
- Base reds that did not fail on the final tree: `inventory.spec.js:2767`, `:2931` (green);
  `sync.spec.js:2976` (flaky).
- **Card 1's own tree (`f980e12`):** 22 failed / 2 flaky / 7 skipped / 1041 passed — no failed test
  outside base; its one first-attempt red outside base (`inventory.spec.js:1469`) 0 red / 3 green
  alone.
- **Card 3's inventory/recipes subset on its own branch:** 244 passed / 19 failed / 1 skipped, all
  19 in `inventory.spec.js`; one of them, `inventory.spec.js:2112`, was red once there and 3/3
  green alone on both trees. It did not fail in any of tonight's three full suites.

**B-481 measurement (the once-red persistence test):** `persistence.spec.js:1463` alone ten times
on the base tree, database reset and server restart each time: **0 red / 10 green**
(`logs/base-b481.log`). Recorded only; nothing filed, nothing fixed.

## Decisions waiting

**None.** No card parked and no question was routed through `night-crew decisions log` tonight.
Engineer-level calls made and stated, not asked:
- Card 2's guard covers scan-by-text as well as the photo path (the implementer's call; the
  reviewer judged it accurate and unnecessary for any crew path, and harmless — no product code
  calls it).
- Card 2 added a seventh spec, `[PS-01b]` (photo then photo) — the only one that drives the defect
  through entries the crew can reach.
- Card 3's error is a sentinel `ErrMergeTargetNotFound`; its test has a third leg (a
  campaign-attached source into a missing target also answers 404, was 500).
- The orchestrator did not re-ask the run-loop question at launch (item 3 above).

## Review findings that were NOT fixed (none filed as backlog items tonight — yours to file or drop at triage)

- **The page's refusal screen under a throwing campaign-policy source is gated by nothing.**
  `[SV-11]` proves the predicate fails closed; no test drives the page there, and the shipped
  policy source (a Map lookup) cannot throw. The slate's premise that `campaigns-run.sh` gates this
  arm was wrong (read, not run).
- **A photo picked during the wait is dropped with no feedback of its own**; one picked while an
  earlier photo is still decoding is dropped with no "Checking…" text at all. Within the slate's
  "no new copy".
- **Card 3's `FOR SHARE` lock and in-transaction placement are pinned by no test** — they held in
  the reviewer's manual race probes; the test still passes with the guard moved after the first
  UPDATE.
- **A missing SOURCE dish merged into a real target still answers `200 {"rows_re_pointed":0}`**, and
  **a non-uuid target still answers 500** — both pre-existing, out of scope.
- Still open from before: a valid dish merge drops the source's sales rows by cascade (B-469).
- **Record corrections applied in place** (the reviewers' own wording): Card 3 — two merge-intent
  sentences and the B-480 backlog line; Card 2 — one spec comment and one merge-intent sentence.
  The slate's "four erasure tests" is also a miscount (three, plus the 0086 round-trip); the slate
  was not edited.

## Not verified by anyone tonight

- The `[SV-08]` and `[SV-09]` mutation reds, by anyone other than the card that wrote them.
- A real camera decode (headless; the reviewer stubbed the camera start and called the real callback).
- The scan-time server read through the real door / PostgREST (the e2e stack has no substrate; the
  server's row is the one permitted stub).
- The dish merge raced against the real handler held mid-transaction; the base binary's behaviour
  for a non-uuid target (judged pre-existing by reading).
- How many production rows migrations `0086` and `0087` would touch — the run may not read `:5433`.
- `campaigns-run.sh` (not run by anyone: it resets the substrate).

## Process notes

- **Two agents shared one scratch directory and one overwrote the other's script** — the cause of
  the uncaptured base exit code and of an orphaned base server that held the suite lock idle for
  ~8 minutes (23:00:46 → 23:08:58). The orchestrator's briefs did not give each agent a private
  scratch path; that is the orchestrator's defect.
- **The orchestrator's first post-merge Go check after merge 2 was invalid** — an empty, unmigrated
  database, 56 setup failures in `recipes`. Re-run with the schema in place: 133 / 0 / 1. Both runs
  are in `logs/merge2-confined.log`.
- **Launch findings stated, not asked:** three card branches hold un-landed work —
  `card/i4-atomic-scan-dedupe` (tonight's Card 1, now merged), `card/a3-rls-fixture-own` and
  `card/s2-demo-sync-target` (B-442, your 2026-09-05 ruling: leave in place). Untouched.
- The suite lock was again the pacing item: three full browser suites at 49–52 m each, serial.
  Reviews started when code was committed; cards ran confined gates and the final suite was the
  combined gate.
- All commits tonight carry `Night-Crew-Run: 20261005` as a real git trailer.
- Scratch databases left on `:5434`: `hq_test_go_{base,c1,j1,j2,g6j1,g6j2,m2,final}`,
  `hq_test_e2e_{base,c1,j1,j2,g6j1,g6j2,final,finalbase}_20261005`, `hq_rls_*_20261005`. Substrate
  still up (reconciled at launch, `EXIT=0`, never `--fresh`).
- Worktrees left under `hq-worktrees/`: `j1-photo-scan-guard-and-lookup-arms`,
  `j2-merge-target-and-blanking-guards`, `i4-atomic-scan-dedupe` (all merged now),
  `base-20261005`, `gate-c1-20261005`, `final-20261005`, `g6-j1-20261005`, `g6-j2-20261005`,
  plus `g6-j2-build`. Nothing was removed.
- Workers at closeout: `night-crew workers check` → "no pollers on any queue", exit 0 (01:05).
- Scorecard: `night-crew scorecard --repo .` → `EXIT_SCORECARD=0`, run 20261005 rendered, all four
  roles record-backed (`logs/scorecard-render.log`). The record was written by hand, as on every
  previous night here; the four team ratings are the orchestrator's judgment, not a measurement.

## Next actions

1. `/nc-morning-triage` — review and merge `overnight-20261005`.
2. Decide whether the two un-gated behaviours under "Review findings" become backlog items.
3. Before any deploy carrying `0086` / `0087`: the production count (B-474).

`git log --oneline dev..overnight-20261005` is the full record of the night.
