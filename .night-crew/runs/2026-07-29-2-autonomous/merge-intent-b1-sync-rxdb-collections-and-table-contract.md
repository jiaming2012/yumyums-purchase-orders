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

- ~~`package.json` / `package-lock.json` — **no npm dependency is added.**~~
  **STRUCK AND RESTATED (amendment 1, below):** the sentence was correct about
  dependencies but became ambiguous once this card added a *new* nested
  `sync-schema/package.json`. Restated precisely:
  **the ROOT `package.json` and `package-lock.json` are NOT touched, and no npm
  dependency is added anywhere.** The collection schemas are declared as plain
  data validated by tests that need no RxDB runtime, which is what "schema only,
  wires no replication" implies. The `@supabase/supabase-js` pin belongs to the
  EXCLUDED card `sync-rxdb-replication-and-conflict-handler`, not here. The new
  `sync-schema/package.json` is a three-key, zero-dependency file whose only job
  is `"type": "module"` — see amendment 1.
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
- `sync-schema/package.json` — **added during the build, see amendment 1.** Three
  keys, zero dependencies, `"type": "module"`.
- `sync-schema/sql/0001_sync_tables.sql` — the self-hosted per-table contract DDL
  for the four replicated tables. **Schema and grants only; contains no
  `CREATE POLICY` STATEMENT.** (Its comments do name the words, to record why no
  policy is here — which is why the test strips comments before asserting. See
  amendment 2.)
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
6. **`sync-schema/sql/0001_sync_tables.sql` contains no `CREATE POLICY` statement.**
   RLS is ENABLED with zero policies, which is deny-all — the correct state until
   `sync-rxdb-row-visibility-rls` (B2) writes the predicates. A merge that adds a
   permissive policy here silently opens the door B2 exists to guard.
   Verified by execution, not by reading: the file was run twice against a scratch
   Postgres seeded with `anon`/`authenticated` roles and a `supabase_realtime`
   publication (both runs exit 0, so it is idempotent), and the catalog then showed
   `relrowsecurity = t` with `policy_count = 0` on all four tables. `SET ROLE
   authenticated; SELECT` returned **0 rows** where the owner saw 1. The scratch
   database and both roles were dropped afterwards.

7. **The `_modified` trigger must stay.** Same execution run: an INSERT supplying
   `_modified = '1999-01-01'` came back stamped with `now()`. Without the trigger a
   skewed client clock silently poisons every replica's pull cursor.

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

---

## Amendments made during the build (B-11 — the whole note was re-read, and the
## one contradicted line above is struck rather than merely superseded)

**Amendment 1 — `sync-schema/package.json` was added; it is NOT a dependency.**
The note originally listed three new files and asserted `package.json` was not
touched. Both statements needed correcting, and the contradicted line is struck
above rather than left standing.

What happened: the repo root `package.json` carries no `"type"` field, so Node
parses every `.js` under it as CommonJS. `sync-schema/collections.js` is authored
as an ES module (its eventual consumer is a `<script type="module">` in the PWA,
beside the already-committed ESM `vendor/rxdb.bundle.js`), so the schema tests
could not import it — measured, not predicted: `SyntaxError: Unexpected token
'export'` on 20 of 28 tests. `sync-schema/package.json` scopes `"type": "module"`
to that one directory. It declares **no dependencies and no scripts**, adds
nothing to `npm ci` (the root has no `workspaces` key), and browsers ignore it
entirely. The alternative — renaming to `.mjs` — was rejected because the Go
backend serves these assets and `.mjs` is not in every mime table.

**Amendment 2 — three negative SQL assertions read statements, not prose.**
`tests/sync-schema.spec.js` originally asserted the strings `create policy`,
`conflict_record` and `lamport_ts` were absent from the raw SQL text. They were
present — in the *comments that record why each is absent*. Rather than delete
those sentences (which are the most useful thing in the file), the test now strips
`--` line comments before asserting. The assertions still fail on a real
statement: proven by mutation, see the card report.

**Nothing else in this note changed.** The shared-file list is still exactly one
entry (`.night-crew/knowledge/roadmap.md`, status flip). `sw.js`, `version.json`,
`sync.js`, `workflows.html`, `backend/**` and every pre-existing spec file remain
untouched.
