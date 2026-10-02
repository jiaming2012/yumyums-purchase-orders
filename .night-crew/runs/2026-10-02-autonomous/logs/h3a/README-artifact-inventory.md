# Artifact inventory — card H3a, run 20261002

Written during the G6 fix round, because G6's finding **F4** reported that
`G2-go.log`, `G2-playwright.log`, `G4-sw.log`, `evidence-itemselection-untouched.log`
and `evidence-toml-footprint.log` "do not exist".

## What actually happened: the review read the middle commit, not the branch tip

Nothing was lost and nothing was written and then deleted. The five artifacts F4
names were committed — just **one commit later than the code they measure**, which
is unavoidable: three of the five gates (G2 Go, G2 Playwright, G4) can only run
*after* the feature commit exists, and `build-sw.js` reads **git HEAD** rather than
the working tree, so G4 is meaningless before the commit.

Measured, per commit on this branch:

```
$ for c in 232cb2a 2e17ae3 5785a3d; do
    echo "$c -> $(git ls-tree -r --name-only $c -- .night-crew/.../logs/h3a/ | wc -l) log files"; done
232cb2a -> 0 log files      # the merge-intent commit, before any gate ran
2e17ae3 -> 5 log files      # the feature commit: G1 + the four RF logs
5785a3d -> 11 log files     # the gate-log commit: + G2(Go), G2(PW) x2, G4, both evidence logs
```

`2e17ae3` holding exactly `G1-build-vet.log` + `RF-green.log` + `RF-red-1/2/3` is
precisely the five-file set F4 describes. So the review was taken at `2e17ae3`
(or at the branch as it stood before the third commit), not at the tip.

**This is a real reporting defect even so**, and it is mine: a card report that
cites artifact paths must say which commit carries them, because a reviewer
handed "the card's diff" has no reason to assume the evidence lives one commit
further on. Recorded in the merge-intent appendix.

## What was fixed in response

1. **Every log now carries an `EXIT=` marker INSIDE the file** (B-445). Three did
   not: `G4-sw.log` had only `RUN1_EXIT=` / `RUN2_EXIT=`, and the two
   `evidence-*.log` files had no marker at all (they are measurements, not gate
   runs — but a cited artifact should self-certify regardless).
2. **This inventory file**, so the set is enumerable without trusting prose.
3. The fix round's own logs (`RF2-red-g6-fixes.log`, `RF2-green-g6-fixes.log`) and
   the re-run gate logs are committed in the SAME commit as the code they measure
   wherever the gate allows it — i.e. everything except G4, which still has to
   follow its commit for the reason above. The appendix names the SHA.

## The full set at the branch tip

| File | Gate | `EXIT=` |
|---|---|---|
| `G1-build-vet.log` | G1 `go build` + `go vet` | 0 (BUILD_EXIT=0, VET_EXIT=0) |
| `G2-go.log` | G2(Go) full suite, first pass | 0 — 655 PASS / 0 FAIL / 3 SKIP |
| `G2-go-refix.log` | G2(Go) full suite, after the fix round | see file |
| `G2-playwright.log` | G2(PW) ONE full suite under the lock | 1 — 26 failed / 963 passed, red-set diff inside |
| `G2-playwright-confined-rerun.log` | the four non-baseline reds, confined | 1 then 0 |
| `G4-sw.log` | G4 `node build-sw.js` x2 | 0 — precache 48 both runs |
| `G4-sw-refix.log` | G4 after the fix round | see file |
| `RF-red-1-pre-change-tree.log` | RF red, true pre-change tree | 1 |
| `RF-red-2-pre-change-tree-db-at-83.log` | RF red, database at goose 83 | 1 |
| `RF-red-3-no-migration-0084.log` | RF red, 0084 withheld | 1 |
| `RF-green.log` | RF green | 0 |
| `RF2-red-g6-fixes.log` | fix round RED (F2/F3/F5) | 1 |
| `RF2-green-g6-fixes.log` | fix round GREEN, whole `internal/toast` | 0 |
| `evidence-itemselection-untouched.log` | ItemSelection-untouched measurements | 0 |
| `evidence-toml-footprint.log` | `night-crew.toml` needs no edit | 0 |
