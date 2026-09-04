# Extraction — marketing-tile-and-page

Outcome: confirmed

Approach used: the standard HQ tile + page mechanics, proven in a throwaway
worktree at `dev` HEAD with a commit between each leg (build-sw.js reads git
HEAD, decision 67): stub `marketing.html` + `index.html` tile link, committed,
`node build-sw.js` rebuild. The card itself is deliberately well-trodden work;
the spike exists because the mechanical invariant has bitten before (B-37
silent precache drop; B-13 committed-sw.js-ships).

Confirmed: all three legs of the invariant. Baseline on current `dev` HEAD is
exactly **31** precached files, exit 0. A committed marketing.html + tile link
moves it to exactly **32** with `marketing.html` in the manifest. And the
negative — the falsifiable half: an un-precached
`<script src="marketing-missing.js">` reference makes build-sw.js exit
non-zero naming `marketing.html -> marketing-missing.js`, so the B-37
reachability guard demonstrably guards the exact file this card adds. Worktree
and branch torn down clean.

Learned: (nothing new — the SW mechanics are exactly as documented; no
surprises to price in.)

Plan change: none — the card proceeds as roadmapped. One in-card obligation
restated: the CLAUDE.md precache-count line ("currently 31") moves to 32 in
the SAME change set that lands the page, or the documented invariant goes
stale.
