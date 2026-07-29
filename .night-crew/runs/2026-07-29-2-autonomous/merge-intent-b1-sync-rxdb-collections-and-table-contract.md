# Merge intent — Card B1 `sync-rxdb-collections-and-table-contract`

Run `overnight-20260729-2`, branch `card/b1-sync-rxdb-collections-and-table-contract`,
cut from `overnight-20260729-2`.

Written BEFORE implementing, per the run's mandatory mechanics. If a repair round
contradicts anything below, the contradicted line is struck through in place —
not merely appended to (B-11).

---

## Shared files touched (files that exist on the base branch)

- `.night-crew/knowledge/roadmap.md` — **status flip only.** The card's own bullet
  moves from `PLANNED — SLATE-READY` to `DONE`. One bullet, one card. No other
  card's text is edited. This is the only file outside my own new files that this
  card writes.

Nothing else. Specifically **NOT** touched:

- `package.json` / `package-lock.json` — **no npm dependency is added.** The
  collection schemas are declared as plain data validated by tests that need no
  RxDB runtime, which is what "schema only, wires no replication" implies. The
  `@supabase/supabase-js` pin belongs to the EXCLUDED card
  `sync-rxdb-replication-and-conflict-handler`, not here.
- `sw.js` / `version.json` — `build-sw.js`'s `globPatterns` are an explicit
  allow-list (`*.html`, `ptr.js`, `sync.js`, `manifest.json`, `version.json`,
  `icons/**/*.png`). A new `sync-schema/**` directory matches none of them, so the
  precache manifest cannot move. Verified at G4 by a clean tree after
  `node build-sw.js`.
- `sync.js`, `workflows.html` — **NOT edited.** Explicitly out of scope on the card.
- `backend/**` — **NOT edited.** No Go source, no goose migration. The sync-DB SQL
  targets the *self-hosted Supabase Postgres*, not HQ's Postgres, so it must never
  become a `backend/internal/db/migrations/*.sql` entry; putting it there would run
  it against HQ's own database.
- `backend/internal/version/version.go` — **NOT edited.** No shipped frontend asset
  and no backend behaviour changes, so neither semver moves.
- `tests/*.spec.js` other than the one new file below — **NOT edited.** B-06
  (`tests/sync.spec.js:1584` stale comment) and B-17..B-21 belong to other cards
  and are deliberately left alone.

## New files this card owns (nothing else may claim them)

- `sync-schema/collections.js` — the four replicated RxDB collection definitions
  (`templates`, `checklists`, `responses`, `approvals`), the LOCAL conflict-record
  collection, and the single named retention constant.
- `sync-schema/sql/0001_sync_tables.sql` — the self-hosted per-table contract DDL
  for the four replicated tables. **Schema and grants only; contains no
  `CREATE POLICY`.**
- `tests/sync-schema.spec.js` — the red-first schema-validation suite.
- This note.

## What must survive any merge

1. **`_modified` is NOT declared in any collection schema (decision 78).** The
   negative test in `tests/sync-schema.spec.js` exists precisely so a later card
   cannot "helpfully" add it back. If a merge resolution ever deletes that test,
   the decision is gone with it.
2. **`_deleted` is NOT declared either** — RxDB owns that field; the Supabase
   replication plugin maps the Postgres column onto RxDB's internal deleted flag.
   Same negative test.
3. **Every replicated row carries who-and-when (decision 79).** Two fields per
   collection. Dropping them degrades the conflict sheet's *"Dana M., 6:12 PM"* to
   *"someone else"*.
4. **The conflict record is LOCAL (decision 89).** No server table, no endpoint, no
   replication. `sync-schema/sql/0001_sync_tables.sql` must stay free of any
   conflict-record table, and the collection must stay free of a `table` mapping.
   Both are asserted by tests.
5. **The retention window is read from exactly one named constant**
   (`CONFLICT_RECORD_RETENTION_DAYS`). A test asserts the literal `30` appears
   exactly once in `sync-schema/collections.js`. The number itself is reopened and
   belongs to `sync-rxdb-conflict-notice-mockup-amendments`, so it must stay
   changeable in one place.
6. **`sync-schema/sql/0001_sync_tables.sql` contains no `CREATE POLICY`.** RLS is
   ENABLED with zero policies, which is deny-all — the correct state until
   `sync-rxdb-row-visibility-rls` (B2) writes the predicates. A merge that adds a
   permissive policy here silently opens the door B2 exists to guard.

## What is safe to drop

- The prose comments inside `sync-schema/collections.js` and
  `sync-schema/sql/0001_sync_tables.sql` are load-bearing as *record*, not as
  behaviour. If a merge conflict lands in a comment block, keeping either side is
  safe for the tests.
- The `RESERVED_UNDECLARED_FIELDS` export is a convenience the tests iterate over;
  a merge could inline it without loss, provided the two field names survive.

## Conflict risk with the other concurrently-dispatched cards

- Track A (`app-timezone-unify-new-york`) and the harness card
  (`test-harness-fail-loud`) touch `backend/**`, `Taskfile.yml`, and existing spec
  files. Zero overlap with this card's footprint.
- `test-harness-fail-loud` may change how `task test` enumerates spec files. This
  card ADDS a spec file (`tests/sync-schema.spec.js`), which is additive to any
  enumeration fix; it does not edit the enumeration.
- Card B2 (`sync-rxdb-row-visibility-rls`) will add policies. It should add them in
  its OWN SQL file rather than editing `0001_sync_tables.sql`, so the "no
  `CREATE POLICY` here" assertion above stays meaningful.
