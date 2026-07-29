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
   (`CONFLICT_RECORD_RETENTION_DAYS`). ~~A test asserts the literal `30` appears
   exactly once in `sync-schema/collections.js`.~~
   **STRUCK AND RESTATED (amendment 4, below):** the check was a text match and it
   leaked. It now tokenises every numeric literal in
   `sync-schema/collections.js` and asserts exactly one of them EVALUATES to the
   window, so a scattered `30.0` / `3e1` / `0x1e` is caught as well as a bare
   `30`. The number itself is reopened and belongs to
   `sync-rxdb-conflict-notice-mockup-amendments`, so it must stay changeable in
   one place.
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
- ~~The `RESERVED_UNDECLARED_FIELDS` export is a convenience the tests iterate over;
  a merge could inline it without loss, provided the two field names survive.~~
  **STRUCK — THE CLAIM WAS FALSE (amendment 3, below):** no test ever iterated it.
  `tests/sync-schema.spec.js` declares its own `MUST_NOT_DECLARE`, and that
  duplication is deliberate — a negative test that reads the forbidden names from
  the module under test proves nothing. The export is currently imported by
  NOTHING. It is safe to drop for a different reason than the one written here:
  dropping it cannot red the suite. Do NOT "fix" the duplication by pointing the
  test at the export.

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

**Amendment 3 — `RESERVED_UNDECLARED_FIELDS` is imported by nothing, and two
artifacts said otherwise.**
The export's own comment claimed it was *"Exported so the negative test iterates
one list"*, and the "safe to drop" bullet above repeated it. Both were false. A
repo grep returns exactly two hits: the declaration in
`sync-schema/collections.js`, and that bullet. `tests/sync-schema.spec.js`
declares its own `MUST_NOT_DECLARE = ['_deleted', '_modified']`.

The DUPLICATION IS CORRECT and was left in place. A negative test that imports its
forbidden-name list from the module it is testing is tautological: delete
`'_modified'` from the export and the test silently stops checking for it, still
green — the exact escape this run has been catching all night. The repair is to
the two false sentences, not to the test.

The export was KEPT rather than deleted, so the `collections.js` diff for this
repair stays comment-only and no other card that may already be reading the branch
loses a symbol mid-run. Its comment now states plainly that nothing imports it,
why the test deliberately keeps its own list, and that if the replication card
lands and still nothing imports it, it should be deleted rather than left as a
decorative export.

**Amendment 4 — the retention-scatter check was a text match with real escapes;
it now matches by VALUE.**
The old check was `src.match(/(?<![\w.])30(?![\w.])/g)` and counted the hits.
Mutating `sync-schema/collections.js` with a scattered second copy of the window
left the suite GREEN in both of these:

- `const SWEEP_DAYS_FLOAT = 30.0;` — **28 passed.** The trailing-`.` lookahead was
  there to stop `30` matching inside `300`; it also skipped `30.0`, i.e. the very
  guard defeated the check.
- `const SWEEP_DAYS_ARITH = 3 * 10;` — **28 passed.**

The realistic scatter (`const X = 30;`) did red correctly, so the check was not
worthless — it was narrower than its own failure message claimed.

Repaired: every decimal/hex/exponent numeric literal in the file is tokenised and
compared NUMERICALLY. `30`, `30.0`, `3e1` and `0x1e` all count; `300`, `0.30` and
`30_0` correctly do not. The `30.0` mutation now reds with *"numeric literals
evaluating to 30 appear 2 times in collections.js (30, 30.0)"*.

`3 * 10` STILL PASSES, and that is a judgment, not an omission: catching a
computed window needs evaluation, not tokenisation, and no textual check reaches
it (`days * 30` with the 30 in another file, `parseInt('30')`, and a hand-rolled
`ms` arithmetic are the same family). It is also not the failure mode the test
exists for — an author scattering the window writes `30`. Recorded as a KNOWN
LIMIT in the test's own comment so the next reader does not mistake green for
coverage.

SCOPE, stated honestly in the test comment as well: the check reads
`sync-schema/collections.js` ALONE. It says nothing about the rest of the repo —
a second copy of the window in the eventual sweep code, in a mockup, or in the SQL
would not be caught. Within this one file it is a real check, and this file is
where the constant is declared.

**Amendment 5 — the `tests/inventory.spec.js:883` explanation in the card report
was WRONG about the mechanism. The conclusion (not this card's fault) stands and
is now independently proven.**
The card report explained the failure as a `.first()` selector colliding over a
shared `eventDate: '2026-04-15'` tie. That mechanism is not real, and triage must
not inherit it as fact:

- The pending list is served `ORDER BY created_at DESC`
  (`backend/internal/inventory/handler.go:612`), and `renderHistoryList`
  (`inventory.html:973-991`) applies NO re-sort — it concatenates
  `PENDING_PURCHASES` ahead of `PURCHASES` in the order received. So
  `page.locator('[data-action="review-pending"]').first()` is the NEWEST pending
  row. `event_date` is not in the sort key at all; the shared `'2026-04-15'` is
  irrelevant.
- A likelier mechanism: `seedPendingPurchase` SWALLOWS a failed POST —
  `if (!res.ok) return null;` (`tests/inventory.spec.js:70`). A silently-failed
  seed leaves the newest pending row belonging to the PRECEDING test
  (`tests/inventory.spec.js:861`, which seeds a line item literally named
  `'test item'`), so `:883`'s `expect(searchVal).toBe('Special Sauce')` receives
  `'Test Item'`. That is a swallowed-failure + shared-fixture-DB story, not a
  tie-break story.
- Also worth recording for triage: `playwright.config.js:43` sets `retries: 1`.
  This card's `--retries=0` runs were therefore STRICTER than a normal
  `task test`, which likely explains why the baseline reads green — a first-attempt
  failure of this shape is retried and passes on a re-seeded attempt.

**NOT FIXED, deliberately.** `tests/inventory.spec.js` is outside this card and
opportunistic fixes are forbidden. Nothing in `tests/inventory.spec.js`,
`inventory.html` or `backend/**` was touched. G6 reproduced the identical failure
with `tests/sync-schema.spec.js` entirely absent from the run, and
`inventory.spec.js` alone on a fresh DB passes 150/150 — so the card is cleared
either way, and only the explanation needed correcting.

---

## RECORDED, NOT FIXED — for triage, not for tonight

These are real gaps found at G6. No code action was taken on any of them; each is
written down so triage sees it rather than rediscovering it.

**R-A. Nothing ties the JS schema to the SQL DDL.** No test cross-checks the
property lists in `sync-schema/collections.js` against the column lists in
`sync-schema/sql/0001_sync_tables.sql`. G6 hand-diffed all four and they match
exactly — `checklist_templates` 8 columns, `checklist_submissions` 11,
`submission_responses` 6, `submission_rejections` 7, plus the server-only
`_deleted`/`_modified` pair the client schemas deliberately omit. But nothing
MECHANICAL prevents drift, and *"mirroring the current Postgres domain model"* is
this card's actual contract — so the contract's central claim is the one thing
held by hand. A good backlog candidate (a single generated cross-check would cover
all four collections and every future one). **Deliberately not built tonight:**
it is new test surface on a card that has already passed its gate, and widening
the card to add it is exactly what a repair round must not do.

**R-B. The spec was RELAXED after the red commit — the red was measured against a
stricter spec than the one that went green.** `git diff 701fb52 3c28e28 --
tests/sync-schema.spec.js` is +20/-6 (`3c28e28` = the card's last commit before
this repair round): three negative SQL assertions
(`create policy`, `conflict_record`, `lamport_ts`) moved from asserting against the
raw file text to asserting against a comment-stripped `readSqlStatements()` — see
amendment 2. G6 confirmed the relaxation cannot mask an EXECUTING statement: a real
`create policy` with a trailing `--` comment still reds. The relaxation is
therefore sound, and the reason for it (keeping the prose that records why each
thing is absent) is a good one. State it plainly anyway: the red-first evidence
for those three assertions was produced under the stricter form.

**R-C. `template_snapshot: { type: 'object' }` has no nested `properties`, so it
stays OPEN.** That is deliberate — constraining the snapshot's shape here would
make this schema a second, drifting definition of the builder's output. The
recorded risk: `vendor/rxdb.bundle.js` exports no dev-mode plugin and no validation
plugin, so nothing in the committed runtime would reject an open object today. The
risk materialises only if a later card adds dev-mode or a schema validator, at
which point an unconstrained `object` may be refused or may pass anything through
unchecked. `sync-rxdb-row-visibility-rls` (B2) and
`sync-rxdb-replication-and-conflict-handler` inherit an OPEN QUESTION, not a
hidden one.

---

**Everything else in this note stands.** The shared-file list is still exactly one
entry (`.night-crew/knowledge/roadmap.md`, status flip). `sw.js`, `version.json`,
`sync.js`, `workflows.html`, `backend/**` and every pre-existing spec file —
including `tests/inventory.spec.js` — remain untouched. No collection schema, no
SQL statement and no assertion's MEANING changed in this repair: the
`sync-schema/collections.js` diff is comment-only, `sync-schema/sql/` is untouched,
and the single assertion edited (retention scatter) was made STRICTER, never
looser.
