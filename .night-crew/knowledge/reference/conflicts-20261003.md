# Conflict log — run `20261003`

One entry per merge into `overnight-20261003`, clean or conflicted (§15ad.66).

## Merge 1 — Card I3 `dish-merge-and-erasure-backstop` → `ed4f568` · CLEAN

- **Cards involved:** I3 only. The run branch held only launch docs (`e68b584`) since the cut.
- **Branch:** `card/i3-dish-merge-and-erasure-backstop` @ `85acf89`. Review ran at `4adf1ad`;
  `4adf1ad..85acf89` is gate logs + the merge-intent's observed lines, no code (checked by diff stat).
- **Files / hunks:** no conflict. Three-way `--no-ff`, not squashed (Card 4 is branched off this tip).
- **Intent read:** `merge-intents/dish-merge-and-erasure-backstop.md` — shared files `CLAUDE.md`
  (the one "Merge:" bullet) and `roadmap.md` (its own line); must survive: migration `0086`, the two
  re-point statements in `MergeMenuItem`, `qr_scans.short` FK untouched.
- **Resolution:** none needed.
- **Gates after merge (`logs/merge1-*.log`):** G1 `EXIT_BUILD=0` `EXIT_VET=0`; confined G2-Go
  (`internal/recipes` + `internal/marketing`, `-p 1`) `EXIT_TEST=0`, 126 pass / 0 fail / 1 skip; G4
  `sw.js` regenerated at merged HEAD — 51 precached, byte-identical, clean on the second run.
  The FULL Go and Playwright suites were NOT re-run at this merge (the suite lock is the night's
  bottleneck and the merged code is identical to the tree the card gated); the final full suite on
  the complete tree covers it.
- **Card's own full-suite gates, as reported — both exited non-zero, neither attributed to the card:**
  G2-Go `EXIT_TEST=1` (448 pass / 1 fail / 3 skip; the red is
  `TestRVClaimFixtureDatabase_RefusesExisting` in `internal/sync`, a 32 s timeout dropping its probe
  database while other suites shared the cluster; alone under the lock it passes in 0.54 s,
  `EXIT_TEST=0`; a clean full-suite `EXIT_TEST=0` was NOT obtained). G2-Playwright `EXIT_PW=1`,
  28 failed / 6 skipped / 1028 passed vs base 25 / 7 / 1030; the 3 extra reds
  (`inventory.spec.js:2931`, `recipes.spec.js:216`, `onboarding.spec.js:2233`) are 3/3 green
  isolated on the card tree, two of them 1/3 red on the base tree, and the diff holds no HTML/JS.
  Three samples per tree is thin; `onboarding.spec.js:2233` reproduced on neither tree.
- **G6:** APPROVE-WITH-FINDINGS, 0 P1 / 0 P2 / 5 P3 (see HANDOFF).

## Merge 2 — Card I1 `test-integrity-fix` → `8a7065d` · CLEAN

- **Cards involved:** I1 onto a run branch already holding I3 (`ed4f568`).
- **Branch:** `card/i1-test-integrity-fix` @ `f09d314`. Review ran at `d14248a`; `73521b1` adds gate
  logs; `adcef9d` + `f09d314` are the G6 P2 fix round (comment-only change in
  `marketing/sync/replicas.js` — checked by diff: zero non-comment lines — plus harness leg 5,
  `[TI-01]`/`[TI-02]`, regenerated `sw.js`). The fix round was NOT sent back through a second fresh
  review; the orchestrator checked its diff shape only.
- **Files / hunks:** no conflict. `roadmap.md` auto-merged (I1 and I3 flip different lines).
  Three-way `--no-ff`, not squashed (Card 2 is branched off `73521b1`).
- **Intent read:** `merge-intents/test-integrity-fix.md` — outside-footprint files
  `marketing/sync/harness/push-harness.mjs` (takes the spec entry; bare v1 schema is refused with
  COL12) and new `replica-schema-harness.mjs`; must survive: the harness IMPORT of
  `failClosed`/`policyFor`, schema v1 + strategies for both `codes` and `offers`, the egress
  allowlist. No overlap with I3's intent (`backend/internal/marketing` tests are disjoint files).
- **Resolution:** none needed.
- **Gates after merge (`logs/merge2-*.log`):** G1 `EXIT_BUILD=0` `EXIT_VET=0`; confined G2-Go
  (`internal/recipes` + `internal/marketing/...`) `EXIT_TEST=0`, 130 pass / 0 fail / 1 skip; schema
  harness exit 0, all five legs held; G4 `sw.js` regenerated at merged HEAD — 51 precached, no diff
  against the card's committed file. Full Go / Playwright suites NOT re-run at this merge; the final
  full suite covers the combined tree.
- **Card's own full-suite gates, as reported (at `73521b1`, before the fix round):** G2-Go
  `EXIT_TEST=0`, 444 / 0 / 3. G2-Playwright `EXIT=1`, 28 failed / 7 skipped / 1028 passed vs base
  25 / 7 / 1030; 4 extra reds (`inventory.spec.js:2931`, `onboarding.spec.js:2233`, `:2268`,
  `sync.spec.js:1327`) passed 12/12 isolated on the card tree; isolation NOT repeated on the base
  tree; `sync.spec.js:1327` is in no previously measured set. Fix round: `tests/marketing.spec.js`
  whole file 54 passed, `EXIT=0`.
- **G6:** APPROVE-WITH-FINDINGS, 0 P1 / 1 P2 (fixed in the fix round) / 6 P3 (see HANDOFF).

## Merge 3 — Card I2 `scan-time-verify` → `3986161` (+ `fb3a35d`) · ONE CONFLICT, resolved by regeneration

- **Cards involved:** I2 onto a run branch holding I3 (`ed4f568`) and I1 with its fix round (`8a7065d`).
  I2 was branched from I1's pre-fix-round tip `73521b1`, so this merge is where I1's fix round and
  I2 first meet.
- **Branch:** `card/i2-scan-time-verify` @ `7ca1adb`. Review ran at `424b2e5`; `e45728e..7ca1adb` are
  the G6 fix round (tests `[SV-05]`/`[SV-06]`/`[SV-07]`, the "Checking with the server…" state, the
  unusable-row handling, regenerated `sw.js`, logs). The fix round was NOT sent back through a
  second fresh review. Three-way `--no-ff`, not squashed (branched off an unmerged base).
- **Conflict:** `sw.js` only — I1's fix round and I2 each regenerated it from different trees.
  `tests/marketing.spec.js` and `roadmap.md` auto-merged (I1's fix round edits `[TI-01]`/`[TI-02]`;
  I2 appends its own `describe`).
- **Intents read:** `test-integrity-fix.md` and `scan-time-verify.md`. Both say `sw.js` is a generated
  artifact that is safe to drop and must be regenerated. I2's must-survive list: the lookup stays
  inside the online-and-not-held branch of step 3 (guarded by `[SV-04]`/`[SV-05]`), `[SV-03b]` (the
  timeout's only guard), no change to `submit-flow.js` / `submit-machine.js`.
- **Resolution:** `sw.js` NOT hand-merged. The card's copy was taken as a placeholder in the merge
  commit `3986161`, then the file was regenerated at that merged HEAD and committed as `fb3a35d`
  (it differed from the placeholder, as expected). A second regeneration at `fb3a35d` leaves the
  tree clean. 51 precached, reachability 38 / 67 / 0 outside (`logs/merge3-G4-sw.log`).
- **Gates after merge:** the FINAL full Go + full Playwright run on the complete tree `fb3a35d` is
  this merge's gate — see `logs/final-*.log` and HANDOFF.
- **Card's own full-suite gates, as reported (at `ebcb384`, before the fix round):** G2-Go
  `EXIT_TEST=0`, 444 / 0 / 3. G2-Playwright `EXIT=1`, 27 failed / 7 skipped / 1035 passed vs base
  25 / 7 / 1030; the 3 reds outside base behaved the same on both trees in 3× isolation. Fix round:
  `tests/marketing.spec.js` whole file 63 passed, `EXIT=0`.
- **G6:** APPROVE-WITH-FINDINGS, 0 P1 / 2 P2 (both fixed) / 4 P3 (see HANDOFF).
