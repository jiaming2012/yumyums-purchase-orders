# Merge intent — Card 3 (J2) · `merge-target-and-blanking-guards`

Run `20261005` · branch `card/j2-merge-target-and-blanking-guards` off `overnight-20261005` at
`3bd6a9b` (the run-branch tip that already holds Card 1's merge and migration `0087`) · Track B
(backend), last card of the track. BACKLOG B-479 (a dish merge into a dish that does not exist
answers 200 and deletes the source dish) and B-480 (the step in migration `0086` that blanks
dangling scan references has no test).

## Shared files touched (outside the card's own packages)

| File | Why |
|---|---|
| `.night-crew/knowledge/BACKLOG.md` | B-479 and B-480 dispositions only: `promoted → …` gains `landed → merge-target-and-blanking-guards`. Append-only; Card 2 edits B-475 / B-476 in the same file (different lines). |
| `.night-crew/knowledge/roadmap.md` | This card's own Activity J line only: `PLANNED` → `LANDED` (branch named; the orchestrator adds the merge SHA). |
| `.night-crew/runs/2026-10-05-autonomous/merge-intents/merge-target-and-blanking-guards.md` | This note. |
| `.night-crew/runs/2026-10-05-autonomous/logs/merge-target-and-blanking-guards/*` | This card's red / green / gate logs. Nothing else under the run directory. |

Inside the footprint: `backend/internal/recipes/repository.go`, `handler.go`,
`repository_test.go`; `backend/internal/marketing/erasure_test.go`.

**Not touched:** any migration (none added, `0086` and `0087` byte-identical to the base),
`sw.js` and every precached file, any HTML/JS, `night-crew.toml`, `CLAUDE.md`,
`backend/internal/marketing/landing.go` / `landing_test.go` (Card 1's).

## What must survive any merge

1. **The target read comes BEFORE the three re-point statements, inside the same transaction**:
   `SELECT 1 FROM menu_items WHERE id = $1 FOR SHARE`. Moved after the campaign UPDATE, an
   attached source reaches the FK refusal (500) while an unattached one still gets 404 — the
   asymmetry returns. (Corrected after review: a guard after the UPDATEs does not delete the
   unattached source.)
2. **The three re-point statements and the `DELETE FROM menu_items` are unchanged** (card I3's
   contract; `TestRepository_MergeMenuItem_RePointsCampaignsAndCodes` pins them).
3. **`404 target_not_found`** on the handler's existing error shape; `400 cannot_merge_into_self`
   stays.
4. **`TestMigration0086BlanksDanglingScanReferences` restores the schema to HEAD with
   `db.Migrate(pool)`** (never a literal) and derives `85` from the `0086` filename.
5. **No migration in this card's diff.** One appearing in a merge result is not this card's.

## What is safe to drop

Nothing here. The evidence logs are safe to drop only in the sense that no suite reads them.

## done_when clauses — stubs and fixtures

| Clause | Rides a stub or fixture? |
|---|---|
| `TestMergeMenuItem_MissingTargetIsRefused` red → green | No stub. Real Postgres on :5434; rows seeded by plain `INSERT`; the production function, and the production handler through `httptest` (no auth middleware in front — the route's auth is not this card's claim). |
| `TestMigration0086BlanksDanglingScanReferences` red (mutated `0086`) → green | No stub. `db.MigrateTo` / `db.Migrate` on the package database; the mutation is applied to the worktree file, run, and reverted — never committed. |
| I3's re-point test, four erasure tests, three round-trips untouched and green | No stub. |
| Go counts (`-p 1`), `inventory|recipes` seam | The seam runs the repo's Playwright stack on this card's own port and database. |

## Facts at the end of the card

Nothing above changed. Additions:

- **The test carries a third leg the slate did not name** — an ATTACHED source (a campaign names
  the dish) into the missing target answers the same `404 target_not_found` and the campaign's
  `item_id` is unchanged. Pre-change that shape answered 500 (`23503
  campaigns_admin_item_id_fkey`); it is the extraction's "both shapes answer 404 the same way".
- **The error is a sentinel**, `recipes.ErrMergeTargetNotFound` (`recipes: target_not_found`),
  beside `ErrRecipeNotFound`. The handler matches it by `strings.Contains`, the existing arm's idiom.
- **Unchanged and stated:** a target that is not a uuid at all still answers `500 internal_error`
  (`22P02`), exactly as before; the dish survives (probed with a throwaway test, not committed).
- **The migration test's cleanup empties the five marketing tables before `db.Migrate`**, so a
  `0086` that cannot swallow the dangling row leaves one red, not a cascade — observed in the
  mutated run: 1 FAIL, the other six PASS, schema back at 87.
- **`CLAUDE.md`'s Merge bullet is still true** and was not edited; it does not mention the
  missing-target refusal.

## Red-first and gates (logs under `logs/merge-target-and-blanking-guards/`, `:5434`, `hq_test_go_j2`)

| Evidence | Log | Observed |
|---|---|---|
| RED 1, pre-change tree | `red-1-recipes-prechange.log` | `returned (0, nil)`; dish rows 0, sales rows 0; handler `200 {"rows_re_pointed":0}`; attached leg `500` (`23503`). `EXIT=1` |
| GREEN 1 | `green-1-recipes.log` | 3/3 sub-legs PASS, `EXIT=0` |
| RED 2, `0086` minus the three-line UPDATE (numstat 0/3) | `red-2-marketing-0086-update-removed.log` | `23503 qr_scans_subscriber_id_fkey` on `ADD CONSTRAINT`; the three erasure tests and the three round-trips (six in all) PASS. `EXIT=1`. Mutation reverted, never committed |
| GREEN 2, shipped `0086` | `green-2-marketing-0086-shipped.log` | 7/7 PASS, `EXIT=0` |
| G1 | `g1.log` | `EXIT_BUILD=0`, `EXIT_VET=0` |
| Go packages, `-p 1` | `go-packages.log` | recipes 61 PASS / 0 FAIL / 0 SKIP; marketing 72 PASS / 0 FAIL / 1 SKIP (`TestProjectionConfiguredUpsertsToSubstrate`, skipped on the base too). `EXIT_TEST=0` |
| Seam `inventory|recipes`, `--retries=0` | `seam-inventory-recipes.log` | 244 passed / **19 failed** / 1 skipped, `EXIT_PW=1`. All 19 in `tests/inventory.spec.js`; `recipes.spec.js` all green |
| Isolation of the 3 reds not in run 20261003's recorded reds, `--repeat-each=3` | `isolate-card-tree.log`, `isolate-prechange-tree.log` | card tree 6/9 (`:2931` red 2/3, `:2406` red 1/3); pre-change production code 5/9 (`:2931` red 3/3, `:2406` red 1/3). `:2112` 3/3 green on BOTH — its seam red was not reproduced on either tree |
| Backlog check | `backlog-check.log` | `backlog: valid — 276 entries`, exit 0 |

The seam's other 16 reds are, line for line, in run 20261003's `base-reds.txt` / `final-reds.txt`
(the Inventory cluster in `bugs.md`); `tests/inventory.spec.js` and `inventory.html` are unchanged
since. Tonight's own base Playwright leg had not finished when this card ran, so the comparison
is against the previous run's record plus the isolation above — not against tonight's base.
