# Extraction — campaigns-tab-ui

Outcome: confirmed

Approach used: the standard HQ page-section mechanics proven in a throwaway
worktree at `dev` HEAD with a commit between legs — a new `marketing/*.js`
module referenced from `marketing.html`, `node build-sw.js` rebuild — plus
feature-detected Web Share with an unconditional Save PNG / Copy link
fallback. One spike exit 0 first run; one recorded skip (engine binaries
not installed in the sitting).

Confirmed: the precache invariant moves 48 → 49 exactly when the committed
module is added, and the B-37 reachability guard fails the build naming
`marketing.html` on a dangling reference — so the card's "sw.js regenerated
and committed, count 48 → 49 (→ 51 across H2/H4/H5)" done_when is
mechanical and checkable.

Learned: headless desktop Chromium exposes neither `navigator.share` nor
`navigator.canShare`; the iOS proxy (webkit) was not measured here (GAP-H2-1
in the ledger). Nothing structural depends on the answer: Share is
feature-detected and both fallbacks ship.

Plan change: none — the card ships the fallback unconditionally and re-runs
spike 2 in its own worktree before finalising the Share copy.
