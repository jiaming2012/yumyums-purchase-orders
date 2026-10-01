# Spikes — campaigns-tab-ui

Activity: Activity H — Campaign admin, subscribers, stats (the designed tabs)

> Hand-run convention (see `campaign-codes-api.md` header). No substrate
> involved; spike 1 uses a throwaway git worktree, spike 2 Playwright engines.

## The goal, and which legs need a spike

The card (H2): the Campaigns section of `marketing.html` — list with the money
strip, the one-sheet create, "N codes ready", detail with the Money card, the
code sheet whose primary action is Share. Two mechanical premises: the
precache invariant moves exactly as the card predicts when it adds
`marketing/campaigns.js` (B-37 / B-13 have bitten before), and the Web Share
API's availability is a known matrix so the fallback is built, not guessed.

## Spike: sw-precache-moves-48-to-49

- proves: (a) `build-sw.js` on current HEAD exits 0 with exactly 48 precached
  files; (b) a COMMITTED stub `marketing/campaigns.js` referenced from
  `marketing.html` moves it to exactly 49 including the module; (c) a
  dangling `<script src>` makes the build exit non-zero and NAME
  `marketing.html` (the reachability guard guards the file this card edits).
- plan: throwaway worktree + branch under TMPDIR, node_modules symlinked, a
  commit between legs (build-sw reads git HEAD, decision 67), torn down on exit.
- script: .night-crew/spikes/activity-h-designed-tabs/campaigns-tab-ui/01-sw-precache-moves-48-to-49.sh

## Spike: web-share-files-enumerated

- proves: for each Playwright engine (chromium, webkit as the iOS-Safari
  proxy, firefox) the set `typeof navigator.share`, `typeof
  navigator.canShare`, `canShare({files:[png]})` — the finding is the matrix
  (B-216), and the card builds Save PNG / Copy link as the fallback wherever
  `canShare` is not `true`.
- plan: one Playwright script, three engines, headless `evaluate`.
- script: .night-crew/spikes/activity-h-designed-tabs/campaigns-tab-ui/02-web-share-files-enumerated.sh
- skipped: webkit and firefox binaries were not installed on this machine
  (`npx playwright install webkit firefox` ran for 25+ minutes in the sitting
  and had not finished); chromium alone answered `share: undefined, canShare:
  undefined` (headless desktop Chromium exposes neither). **Consequence the
  card absorbs regardless of the matrix:** Share must feature-detect
  (`navigator.canShare?.({files})`) and fall back to Save PNG + Copy link —
  the design already shows both. The spike re-runs at morning triage once the
  binaries exist, or in the card's own worktree; a different matrix changes
  copy, not structure.

## Verdict (hand-run 2026-10-01)

- **sw-precache-moves-48-to-49: passed** — exit 0, first run: baseline 48 →
  49 with `marketing/campaigns.js` in the manifest; the dangling reference
  exited non-zero with `marketing.html -> marketing/missing-for-spike.js`.
- **web-share-files-enumerated: could-not-run (run 1, run 2) → skipped** —
  see the skip line above; chromium's row is recorded (`undefined` /
  `undefined` / no canShare).

## Corrections

- none — no agent-reached corrections (the skip is a recorded skip, not a change to a premise or a script)

## Comebacks

- gap: GAP-H2-1 — the share matrix for the iOS proxy (webkit) is unmeasured on this machine; validates when the spike re-runs with the binaries installed and the card's fallback copy is checked against the result

## Review

- signed: operator, 2026-10-01 — covers 0 correction(s) (the recorded skip travels to the slate sign-off per the skill's rule)
