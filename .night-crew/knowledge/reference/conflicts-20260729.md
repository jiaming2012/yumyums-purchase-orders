# Merge conflict log — `overnight-20260729`

> DESIGN §15ad.66. **Every** merge to the run branch gets an entry here, clean or conflicted.
> Clean merges get a one-line entry on purpose — so an empty log can never be misread as
> "no conflicts" when what it actually means is "the logging never ran".
>
> Run: `overnight-20260729` · Slate: `slate-20260729.md` (signed 2026-07-28, 4 cards)
> Merge order: A (Wave 0, alone) → then Track A (B) ∥ Track B (C → D), merged in slate order.
> Merges are performed by the orchestrator only. Conflicts are resolved by reading BOTH sides'
> merge-intent notes under `.night-crew/runs/2026-07-29-autonomous/` and resolving against
> **intent, not text** — then re-running G1+G2.

| # | Card merged | Files / hunks | Verdict | Gate result after merge |
|---|---|---|---|---|
| 1 | A · `precache-manifest-from-head` | 7 files, 0 conflicted hunks | **CLEAN** | G1 green, `sw.js` rebuild idempotent |

---

## 1 — Card A `precache-manifest-from-head` → `overnight-20260729`

**Merge type:** CLEAN. Wave 0, merged alone, first. No other card had landed, so there was
nothing to conflict *with* — this entry exists precisely so that fact is on the record rather
than inferred from an empty file.

**Files carried in (7):**

| File | Note |
|---|---|
| `build-sw.js` | the fix — `git ls-files` (index) → `git ls-tree -r --name-only -z HEAD` (commit) |
| `tests/sw-manifest.spec.js` | test 1 co-moved off `ls-files`; new red-first staged-probe test |
| `sw.js` | regenerated through the **fixed** globber |
| `package.json`, `backend/internal/version/version.go` | Frontend 1.2.0 → 1.2.1, mirrored |
| `.night-crew/knowledge/roadmap.md` | card flipped to DONE |
| `.night-crew/runs/2026-07-29-autonomous/merge-intent-a-…md` | the card's merge-intent note |

**Intents read:** only Card A's. Its note flags three shared surfaces for the cards still to come,
and they are recorded here because later merges will need them:

1. **`build-sw.js`** is shared with the already-landed `pwa-cache-and-build-hygiene`, which
   authored `trackedFiles` / `GENERATED_BUT_SHIPPED`. Card A renamed
   `trackedFiles`→`committedFiles` and `trackedOnlyTransform`→`committedOnlyTransform`. **Any
   future conflict here is a rename collision, not a logic collision.** The property that must
   survive is *"nothing outside HEAD ∪ allowlist reaches the manifest"* — not the spelling.
   `-r` and `-z` are both load-bearing.
2. **`backend/internal/version/version.go` + `package.json`** are shared with every card tonight.
   On conflict: take the **higher** Frontend semver and **re-mirror into both** — CLAUDE.md
   forbids them diverging.
3. **`sw.js`** is generated. Never hand-merge it — take either side, then re-run `node build-sw.js`
   and commit the result. Cards B and C regenerate it *through the globber this card just fixed*,
   which is why A was Wave 0 and landed alone.

**Resolution taken:** none required.

**Gate result after merge:** `go build ./...` + `go vet ./...` green on the merged tree;
`node build-sw.js` at the merge commit leaves the tree clean (22 files / 1463.6 KB, unchanged),
confirming the committed `sw.js` is what the fixed globber produces.

**G6:** APPROVE-WITH-NITS (no blocking findings). Nits 1 and 2 are carried to the closeout as
backlog candidates, not fixed in-run — see HANDOFF.md.

---
