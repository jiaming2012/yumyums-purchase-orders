# Merge intent — Card 4 (I4) · `atomic-scan-dedupe`

Run `20261003` · branch `card/i4-atomic-scan-dedupe`, cut from Card 3's unmerged branch tip
`4adf1ad` (`card/i3-dish-merge-and-erasure-backstop`) · Track B (backend), second card of the
track. Card 3's commits are this branch's history and are not modified, reverted or amended —
3-way merge, never squash; merge Card 3 first.

## Shared files touched (outside the card's footprint, or shared with another card)

| File | Why |
|---|---|
| `.night-crew/knowledge/roadmap.md` | This card's own line only: `PLANNED` → `LANDED` (branch named; the orchestrator adds the merge SHA). |
| `.night-crew/runs/2026-10-03-autonomous/merge-intents/atomic-scan-dedupe.md` | This note. |
| `.night-crew/runs/2026-10-03-autonomous/logs/atomic-scan-dedupe/*` | This card's red/green and gate logs. Nothing else under the run directory. |

Inside the footprint: `backend/internal/db/migrations/0087_qr_scans_dedupe_bucket.sql` (new),
`backend/internal/marketing/landing.go`, `backend/internal/marketing/landing_test.go` (the three
named tests, the pre-existing-duplicates test and the migration Down round-trip leg all live
here, so no `zz_`/`zzz_`/`erasure_test.go` file is edited).

**Not touched:** `sw.js` and every precached file (count stays 51), `night-crew.toml` (no key,
no token), every Card 3 file (`0086_*.sql`, `recipes/repository.go`, `erasure_test.go`,
`CLAUDE.md`), `subscribers_test.go` (Card 1's), any HTML/JS, `BACKLOG.md` (the slate assigns
this card no BACKLOG disposition; decision 195 came from DECISIONS-NEEDED D-2, not a B-item).

## Engineer-level decisions, stated

1. **What the Up does with duplicates that are already in the table.** The shipped statement let
   concurrent hits through, so a live table can hold several rows with one
   `(short, ip_hash, bucket)`, and a unique index build fails on the first such group — which
   would fail the migration and keep the server from booting at deploy. The Up therefore
   de-duplicates before it builds the index, in the same transaction:
   - Two rows that share a 10-minute tumbling bucket are by construction less than ten minutes
     apart, so every surplus row in a group is a row the shipped 10-minute rule meant to refuse
     and the race let in. Removing it makes the historic count what the rule always defined; it
     removes no scan the old rule would have counted.
   - One row per group is kept: a row that a signup already claimed (`subscriber_id` set) is
     preferred, otherwise the earliest (`scanned_at`, then `id`).
   - Surplus rows with no `subscriber_id` are **deleted**.
   - A surplus row that carries a `subscriber_id` (a second claimed row in one group) is **never
     deleted** — it is a converted visit. Its `ip_hash` is blanked instead, so it stays in the
     table and in the count as an anonymous scan and stops colliding with the index.
   - Rows with `NULL ip_hash` are not touched (they never dedupe).
   - On a table with no duplicates this deletes and updates 0 rows.
   The Down does not restore deleted rows (they were not scans by the rule's own definition).
   Tested by `TestMigration0087CollapsesPreExistingDuplicates`.
2. **The statement is extracted into `insertScan(ctx, db, …)`** (same package, unexported) so the
   concurrency test can run the production statement on twelve already-open connections. The
   handler path (`logScan`) calls it with the pool; no route, header or response changes.
3. **`scanDedupeWindow` (a Go constant) is removed.** The `$5` interval parameter disappears from
   the statement (spike learning 3), and a constant nothing reads would claim to control a
   window that now lives in the migration's `date_bin('10 minutes', …)`. The window itself is
   unchanged at ten minutes.
4. **Bucket epoch `2000-01-01 00:00:00+00`, index name `qr_scans_dedupe_idx`** — both exactly as
   spike 02 printed them.

## What must survive any merge

1. **Migration number `0087`**, after Card 3's `0086`. Up + Down.
2. **The column and index in the spike's shape**: `bucket timestamptz GENERATED ALWAYS AS
   (date_bin('10 minutes', scanned_at, timestamptz '2000-01-01 00:00:00+00')) STORED`; unique
   index `qr_scans_dedupe_idx ON qr_scans (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL`.
3. **The insert is ONE statement** ending `ON CONFLICT (short, ip_hash, bucket) WHERE ip_hash IS
   NOT NULL DO NOTHING`. The `WHERE` in the conflict target must match the partial index or
   Postgres rejects the statement (42P10). No read-then-write, no advisory lock.
4. **`NULL ip_hash` inserts every time** and the scans metric stays a plain `count(*)`
   (`campaigns.go`, `stats.go` — not edited).
5. **The de-duplication step runs before the index build** in the Up (decision 1 above).
6. **`TestLandingDedupeIsAtomicUnderConcurrency` keeps its barrier**: twelve connections opened
   first, twelve goroutines released together. Rewritten to spawn-and-go it passes against the
   broken statement.
7. **The Down round-trip leg restores with `db.Migrate(pool)`, never a literal**, and derives its
   version from the migration filename.

## What is safe to drop

Nothing here. The evidence logs are safe to drop only in the sense that the suite does not read
them.

## Known and accepted (not introduced silently)

- **Tumbling bucket, not sliding window** (decision 195, spike correction 2): two taps a second
  apart that straddle a bucket edge count twice. `TestLandingCountsAgainAfterWindow` asserts it.
- **`TestLandingLogsScanAndRedirectsWithUTM` is untouched**; its "repeat scan → still 1 row" step
  makes two real-clock requests a few milliseconds apart, so it would count 2 if those two
  requests straddled a bucket edge — a window of milliseconds once every ten minutes. Not
  changed (the slate says untouched); stated so a once-in-a-long-while red there is recognised.
- **Image rollback with the schema at 87**: the previous binary's `INSERT … WHERE NOT EXISTS`
  still works; a racing duplicate now gets a unique violation, which `logScan` already logs and
  swallows — the customer is still redirected.

## done_when clauses — stubs and fixtures

| Clause | Rides a stub or fixture? |
|---|---|
| `TestLandingDedupeIsAtomicUnderConcurrency` red → green (12 open connections, one barrier, ×5) | No stub. Real Postgres on :5434; twelve `pgxpool` connections acquired before the barrier; the statement run is the production `insertScan`. Campaign + code seeded through the real create-campaign route. |
| `TestLandingAnonymousScansNeverDedupe` green | No stub. Production `insertScan` twice with a NULL `ip_hash`. |
| `TestLandingCountsAgainAfterWindow` green | No stub. The EARLIER scan of each pair is a fixture row inserted with an explicit `scanned_at` (the clock cannot be moved); the later one is the production `insertScan`. |
| `TestLandingLogsScanAndRedirectsWithUTM` untouched and green | No stub; not edited. |
| Migration Down round-trip | No stub. `db.MigrateTo` / `db.Migrate` on the package's test database; `information_schema` / `pg_indexes` read back. |
| Pre-existing duplicates (decision 1) | Fixture rows inserted at schema version 86 with explicit `scanned_at`, then the real Up. |
| Go counts (`-p 1`) | The full Go suite's `internal/sync` leg reads the already-up `spike-supabase` substrate (not this card's). |
