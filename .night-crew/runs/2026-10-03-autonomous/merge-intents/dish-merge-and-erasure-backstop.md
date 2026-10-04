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
