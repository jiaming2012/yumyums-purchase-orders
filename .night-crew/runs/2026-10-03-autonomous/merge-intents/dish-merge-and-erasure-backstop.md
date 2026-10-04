# Merge intent — Card 3 (I3) · `dish-merge-and-erasure-backstop`

Run `20261003` · branch `card/i3-dish-merge-and-erasure-backstop` off `overnight-20261003` at
`ec2830c` · Track B (backend), first card of the track. Card 4 (`atomic-scan-dedupe`) is
branched from this branch's tip — 3-way merge, never squash.

## Shared files touched (outside the card's footprint, or shared with another card)

| File | Why |
|---|---|
| `CLAUDE.md` | **One line.** The inventory "Merge:" bullet said "Menu items in the Recipes tab can be merged the same way"; no frontend calls the dish merge (triage T-62 finding 1), so the line now says the dish merge is API-only (`POST /api/v1/inventory/recipes/merge`) and that it re-points recipes, campaigns and codes. Named in the slate's footprint and Shared-surfaces table; restated here because it is the project-instructions file. |
| `.night-crew/knowledge/roadmap.md` | This card's own line only: `PLANNED` → `LANDED` (branch named; the orchestrator adds the merge SHA). |
| `.night-crew/runs/2026-10-03-autonomous/merge-intents/dish-merge-and-erasure-backstop.md` | This note. |
| `.night-crew/runs/2026-10-03-autonomous/logs/dish-merge-and-erasure-backstop/*` | This card's red/green and gate logs. Nothing else under the run directory. |

Inside the footprint: `backend/internal/db/migrations/0086_merge_repoint_and_erasure_backstop.sql`
(new), `backend/internal/recipes/repository.go`, `backend/internal/recipes/repository_test.go`,
`backend/internal/marketing/erasure_test.go` (new — it also carries this card's migration
Down round-trip leg, so no existing `zz_`/`zzz_` file is edited).

One engineer-level addition inside `0086`, stated: before adding `qr_scans_subscriber_id_fkey`
the Up blanks any `qr_scans.subscriber_id` that names no existing subscriber (0 rows on a
clean database). The column never had an FK, and a single dangling id would otherwise fail the
whole migration at deploy. It deletes nothing.

**Not touched:** `sw.js` and every precached file (count stays 51), `night-crew.toml` (no key,
no token), `backend/internal/marketing/landing.go` and `landing_test.go` (Card 4's),
`backend/internal/marketing/subscribers_test.go` (Card 1's egress-guard edit), any HTML/JS,
`BACKLOG.md` (the slate assigns this card no BACKLOG disposition).

## What must survive any merge

1. **Migration number `0086`.** Fixed by the slate; Card 4 owns `0087`. Five ALTERs, Down included.
2. **`ON DELETE CASCADE` exists on exactly one FK: `subscriber_events.subscriber_id`.** The other
   four are `SET NULL`. A cascade anywhere else is decision 194's explicit rejection.
3. **`qr_scans.short → qr_codes(short)` keeps its plain FK.** A scanned code is undeletable by
   design; `TestScannedCodeDeleteIsRefused` asserts the `23503 qr_scans_short_fkey` refusal. A
   merge that "fixes" that test by adding a cascade is scope drift.
4. **The two re-point statements in `recipes.MergeMenuItem`** (`campaigns_admin.item_id`,
   `qr_codes.item_id`), inside the existing transaction, before the `DELETE FROM menu_items`.
   The function's returned figure is now the TOTAL rows re-pointed (recipes + campaigns +
   codes) — the slate's first-named option ("in the existing `rows` figure"), so
   `handler.go` and the `rows_re_pointed` wire key are untouched; the value only differs
   from before when a campaign or code named the source dish, which previously was a 500.
5. **The Down round-trip leg restores with `db.Migrate(pool)`, never a literal**, and derives
   its version from the migration filename.

## What is safe to drop

Nothing here. Every file is either the contract or its proof. The evidence logs are safe to
drop only in the sense that the suite does not read them.

## done_when clauses — stubs and fixtures

| Clause | Rides a stub or fixture? |
|---|---|
| `TestRepository_MergeMenuItem_RePointsCampaignsAndCodes` red (`23503`) → green | No stub. Real Postgres on :5434; rows seeded by plain `INSERT`; the merge is the production function. |
| `TestSubscriberDeleteCascadesTimelineAndBlanksScans` red → green | No stub. A plain `DELETE FROM subscribers` against the migrated schema. |
| `TestCodeDeleteBlanksFirstTouch` red → green | No stub. A plain `DELETE FROM qr_codes`. |
| `TestScannedCodeDeleteIsRefused` green (`23503 qr_scans_short_fkey`) | No stub. |
| Migration Down round-trip | No stub. `db.MigrateTo` / `db.Migrate` on the package's test database; `pg_constraint` read back. |
| Go counts (`-p 1`) | The full Go suite's `internal/sync` leg reads the already-up `spike-supabase` substrate (not this card's). |

## Red-first

Logs under `.night-crew/runs/2026-10-03-autonomous/logs/dish-merge-and-erasure-backstop/`.
All on `:5434`, database `hq_test_go_i3`. The tests were committed red (`3d3c5c7`) before any
production change.

| Test | RED (tree, log, exit) | GREEN (log, exit) |
|---|---|---|
| `TestRepository_MergeMenuItem_RePointsCampaignsAndCodes` | pre-change tree — `red-1-recipes-prechange.log`: `23503 campaigns_admin_item_id_fkey`, `EXIT=1`. **Second red, migration applied but no re-point** — `red-2-recipes-migration-only.log`: `item_id = NULL, want B` on both rows, `EXIT=1` (the backstop alone does not satisfy the test; the re-point is load-bearing) | `green-recipes.log` (whole package, 60 PASS / 0 FAIL / 0 SKIP), `EXIT=0` |
| `TestSubscriberDeleteCascadesTimelineAndBlanksScans` | pre-change tree — `red-1-marketing-prechange.log`: `23503 subscriber_events_subscriber_id_fkey`, `EXIT=1` | `green-marketing-erasure.log`, `EXIT=0` |
| `TestCodeDeleteBlanksFirstTouch` | pre-change tree — same log: `23503 subscribers_source_short_fkey`, `EXIT=1` | same log, `EXIT=0` |
| `TestScannedCodeDeleteIsRefused` | not a red-first test: PASS on the pre-change tree (same red log) and PASS after — it pins the refusal `23503 qr_scans_short_fkey` | same log, `EXIT=0` |
| `TestMigration0086ErasureBackstopDownAndUpRoundTrip` | pre-change tree — same red log: all five `confdeltype` assertions fail and no `0086` file is found | same log, `EXIT=0`; the pre-existing `zz_` (0083) and `zzz_` (0085) round-trips also run 0086's Down and pass |

The first attempt at the recipes red hit `42P01` on a freshly created, unmigrated database
(the `recipes` package does not migrate) — the wrong reason; it was re-run after
`internal/marketing`'s `TestMain` migrated the database, and the log says so in its header.

## Gates observed (at `4adf1ad`, the last code-bearing commit; later commits are logs and this note)

Same logs directory.

- **G1** — `g1.log`: `EXIT_BUILD=0`, `EXIT_VET=0`.
- **G2-Go** — `g2-go.log`: `EXIT_TEST=1`; 449 top-level tests = 448 PASS / **1 FAIL** / 3 SKIP
  (base 444 / 0 / 3; +5 are this card's), 14 packages `ok`, `internal/sync` FAIL. The one red is
  `TestRVClaimFixtureDatabase_RefusesExisting` — `free hq_rls_claimprobe_p13161: timeout:
  context deadline exceeded` (dropping its own probe database timed out, 32 s) — in a package
  this card does not touch. `TestRowVisibilityRLS` itself passed in that run. Re-run alone under
  the lock — `g2-go-rerun-claimprobe.log`: PASS in 0.54 s, `EXIT_TEST=0`. Both exit lines stand;
  the full-suite line is `EXIT_TEST=1`, not re-run as a whole.
- **G2-Playwright** — `g2-pw.log`: one summary block, **28 failed / 6 skipped / 1028 passed**,
  `EXIT_PW=1` (base tonight: 25 / 7 / 1030). All 25 base reds are red here. Three more:
  `inventory.spec.js:2931` and `onboarding.spec.js:2233` (both in the 29-red set named in
  `bugs.md`, green in tonight's single base sample) and `recipes.spec.js:216` (named in
  `bugs.md` as a suspected ⅓-pass race). Isolation, `--repeat-each=3 --retries=0`, both trees:
  card tree `isolate-card-tree.log` 9/9 green, `EXIT_PW=0`; base tree `ec2830c`
  `isolate-base-tree.log` 7/9 — `inventory:2931` red 1/3 and `recipes:216` red 1/3 **on the
  base**, `EXIT_PW=1`. `onboarding:2233` was 3/3 green on both trees, so its full-suite red is
  NOT reproduced in isolation on either tree; what excludes it is mechanical — this card's diff
  contains no HTML/JS and no onboarding, inventory-handler or workflow code, and none of the three
  tests calls the dish merge or any marketing route.
- **G4** — `g4.log`: `sw.js` regenerated → tree clean (idempotent), precache **51**, identical
  to the base commit's `sw.js`.
- **B-467 hazard:** the suite rewrote 18 tracked PNGs under
  `.night-crew/runs/2026-10-02-autonomous/`; restored with `git checkout --` before committing,
  none committed.
