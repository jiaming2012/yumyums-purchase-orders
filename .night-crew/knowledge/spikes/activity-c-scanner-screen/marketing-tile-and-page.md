# Spikes — marketing-tile-and-page

Activity: Activity C — The scanner screen (staff redemption at the window)

> No `usm/roadmap.txt` on this target — hand-run convention (full preamble in
> `../activity-b-offline-first-replica/rxdb-pull-replica.md`; substrate not
> involved here). This goal's spike runs entirely in a THROWAWAY git worktree
> on a throwaway branch — the real working tree, `dev`, and every remote are
> untouched, and nothing is pushed.

## The goal, and which legs need a spike

The card (roadmap Activity C): add the Marketing tile + `TILE_SLUGS` entry,
create `marketing.html`, seed `('marketing','Marketing','📢')`, regenerate
`sw.js` (precache count moves 31 → 32 deliberately). done_when includes
`build-sw.js` exits 0.

This is deliberately well-trodden HQ work (tile, page shell, permission seed —
all have shipped before), so most of the card has **no novel falsifiable
premise** — recorded here per FR-13a rather than dodged with a "no spike
needed" line. The one premise worth a runnable check is the card's mechanical
invariant, because it has bitten before (B-37 silent precache drop; B-13
committed-sw.js-ships): **adding a page moves the precache exactly as the card
predicts, and the reachability guard actually fires when the new page
references something un-precached.** A guard that would pass a broken
marketing.html makes the card's done_when vacuous — the negative leg is the
falsifiable half.

## Spike: sw-precache-invariant

- proves: in a throwaway worktree of current `dev` HEAD, (a) baseline:
  `node build-sw.js` exits 0 and precaches exactly 31 files (the documented
  invariant holds before the card moves it); (b) positive: adding a stub
  `marketing.html` + an `index.html` tile link, COMMITTED (build-sw reads git
  HEAD, decision 67 — an uncommitted page must not and does not ship),
  rebuilds to exit 0 with exactly 32 precached files including
  `marketing.html`; (c) negative: making the stub reference a script that is
  not precached (`<script src="marketing-missing.js">`), committed, makes
  `build-sw.js` exit NON-zero and name `marketing.html` as the referrer — the
  reachability guard demonstrably guards the exact file this card adds.
- plan: `git worktree add` a throwaway branch at HEAD under the session
  scratchpad; symlink the repo's `node_modules` into it (build-sw needs
  workbox); run the three legs with a commit between each; count entries by
  the `url:` pattern in the generated sw.js AND cross-check build-sw's own
  "N files precached" line; tear down worktree + branch on exit (trap), leaving
  `git worktree list` and `git branch` clean.
- script: .night-crew/spikes/activity-c-scanner-screen/marketing-tile-and-page/01-sw-precache-invariant.sh

## Verdict (run 2026-09-04, hand-run per the no-story-map convention)

- **sw-precache-invariant: passed** — exit 0, first run. (a) baseline on
  current `dev` HEAD: `SW built: 31 files precached`, exit 0 — the documented
  invariant holds; (b) a committed stub marketing.html + index.html tile link
  moved the precache to exactly **32** with `marketing.html` in the manifest;
  (c) the negative: an un-precached `<script src="marketing-missing.js">`
  reference made build-sw.js exit 1 with
  `marketing.html -> marketing-missing.js` named in the failure — the B-37
  guard demonstrably guards the exact file this card adds. Worktree + branch
  removed clean (trap verified — `git branch --list 'spike-sw-*'` empty after).

**Conclusion:** the card's SW mechanics are exactly as documented; the
CLAUDE.md precache-count line moves 31 → 32 when the card lands (update it in
the same change set). No surprises to price in.

## Corrections

- none — no agent-reached corrections. The script passed on its first run.

## Review

- n/a — no corrections, so no batch review is owed for this goal (§4 fires
  only when the ledger holds agent-reached corrections).
