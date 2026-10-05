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
   `SELECT 1 FROM menu_items WHERE id = $1 FOR SHARE`. Moved after the UPDATEs, an attached
   source reaches the FK refusal (500) while an unattached one is deleted (200) — the asymmetry
   B-479 reports.
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

(Updated at the end of the card if any fact above changed.)
