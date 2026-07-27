# Merge conflict log — run `overnight-20260727`

Required per DESIGN §15ad.66. **Every merge gets an entry, clean or conflicted**, so an empty
log can never be read as "no conflicts" when it actually means "the logging never ran".

Run branch: `overnight-20260727`, cut from `dev` @ `e1c40a8`.
Dispatch: Wave 0 (Card A) alone, then Cards B and C **serial** — both own `workflows.html`.

---

## Merge 1 — Card A `pwa-cache-and-build-hygiene` → `overnight-20260727`

**Merge commit:** `c14865b` · **Card branch:** `card/a-pwa-cache-and-build-hygiene` (9 commits,
cut from `overnight-20260727` @ `e1c40a8`) · **Result: CLEAN — no content conflicts.**

**Cards involved:** Card A only. It was dispatched alone as Wave 0 precisely so the generated
`sw.js` surface would settle before Cards B and C run `task sw`, so by construction there was no
second card to collide with.

**Files and hunks:** 8 files, +501 / −24.

| File | Note |
|---|---|
| `index.html` | +71 — `clearApiCache`, `evictCachedIdentity`, cache-busted probe, and the repair-1 scope comment |
| `build-sw.js` | tracked-set glob (`git ls-files`) + `GENERATED_BUT_SHIPPED` allowlist; vendor glob dropped |
| `sw.js` | regenerated artifact, 22 files / 1455.6 KB |
| `tests/index.spec.js` | +98 — 3 new tests |
| `tests/sw-manifest.spec.js` | +86 — new file, 3 new tests |
| `.night-crew/knowledge/roadmap.md` | card status flip + repair-1 claim narrowing |
| `.night-crew/runs/2026-07-27-autonomous/merge-intents/a-pwa-cache-and-build-hygiene.md` | new |
| `.night-crew/runs/2026-07-27-autonomous/timings.log` | new |

**Intents read:** only Card A's merge-intent note existed at this point; there was no second
intent to weigh it against. Its "must survive any merge" list was checked against the merged
tree and holds. Note that repair 1 **rewrote its item 3** — the `removeUserHeader()` call is now
recorded as *safe to drop on a conflict*, with an explicit instruction that a merge must not
restore the stronger "strips any name already on screen" wording. Cards B and C do not touch
`index.html` or `build-sw.js`, so this list is not expected to come under pressure again.

**One collision, and it was mine, not the card's.** The first `git merge` aborted:

    error: The following untracked working tree files would be overwritten by merge:
            .night-crew/runs/2026-07-27-autonomous/timings.log

The orchestrator had been writing its own observed spans to that path in the main worktree while
the card was independently committing its implementer spans to the same path in its worktree.
**Resolution: union, not either-side.** The orchestrator's copy was saved aside, removed so the
merge could proceed, and then re-appended under an `ORCH A ` prefix beneath a comment that
explains the two line families (implementer stamp-to-stamp inside the worktree vs
orchestrator-observed wall clock). No timing record was discarded. Going forward the orchestrator
writes only `ORCH `-prefixed lines, so the two never collide again.

**Gate result after the merge:** see the closeout for the final-tree sweep. Pre-merge, on the card
branch: full Playwright **549 passed / 0 failed / 0 flaky / 6 skipped of 555, 22.8m at
`--retries=0`**; Go `go test ./... -count=1 -p 1` all packages green; `node build-sw.js` cold-build
verified **byte-identical** to the committed `sw.js` by the orchestrator after the repair leg.

**G6:** APPROVE-WITH-NOTES, 8 findings, one bounded repair round applied before merge (F1 claim
narrowing, F2 vacuity closure). Findings F3, F6, F7 deliberately left for the operator; F4 binds
Card C and has been carried into its dispatch prompt.
