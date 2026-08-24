# merge-intent — A1 `e2e-gate-database-isolation`

Run `overnight-20260804`, Wave 0, first and alone. Branch `card/a1-e2e-gate-database-isolation`,
cut off `overnight-20260804` at **`c9e9e9e`**. Every diff claim in this note is measured against
`c9e9e9e`, not against the run branch HEAD (T-32 decision 130a).

Written BEFORE implementing (§15ad.65). Updated in place only for facts that changed.

Closes **B-76**. Fixes **B-81** (scoped fix round, after G6 returned APPROVE-WITH-NOTES).

**Fix round, 2026-08-04 — appended, card NOT reopened.** G6 returned APPROVE-WITH-NOTES. One code
fix was applied (**B-81**, the stdout-swallowed reset banner) plus seven backlog filings
(**B-80**–**B-86**) and a correction to B-76's own resolution note. Everything below that predates
the fix round still holds as written; the fix round's additions are marked. Gates re-run: **G4**
(`node build-sw.js` twice) and one **subset Playwright leg**. The full 786-test suite and the Go
suite were **not** re-run and did not need to be — the fix-round diff is one `console` call in
`scripts/reset-e2e-db.js`, no Go code and no production code.

---

## Shared files touched — each line says why

| File | Why this card touches it |
|---|---|
| `scripts/reset-e2e-db.js` | **New. The card's own file.** The single place the e2e database is reset and the single place its Postgres coordinates are computed. Carries the loud guard that refuses any database whose name is not a test database. **Fix round:** its CLI branch now prints the reset banner with `console.error`, not `console.log` — **B-81**. Nothing else in the file moved: not the guard, not the DROP/CREATE, not `require.main === module`. |
| `playwright.config.js` | The fix itself: `webServer.command` now begins with `node scripts/reset-e2e-db.js &&`, and the duplicated coordinate constants at `:4-27` now come from `scripts/reset-e2e-db.js` so there is one source of truth. 🛑 **`serviceWorkers: 'block'` is untouched** (B-15). |
| `tests/db-isolation.spec.js` | **New.** The red-first test. Cross-process by construction — it checks for a marker left by the previous invocation, then leaves one for the next. |
| `Taskfile.yml` | `test:` and `test:ui:` had their own inline `psql … DROP/CREATE` blocks. Those are now the *only other* reset in the repo and would be a second source of truth; both are replaced by a pointer to the config's reset. `test:ui:`'s block also carried `DB_PORT` default **5432** where the rest of the repo uses **5433** — deleting it removes that inconsistency rather than perpetuating it. |
| `.night-crew/knowledge/BACKLOG.md` | B-76 marked resolved with the evidence. **Fix round:** **B-80**–**B-86** filed (numbers allocated by the orchestrator, not invented by this card), **B-81** filed as RESOLVED with its commit SHA, and 🛑 **one sentence inside B-76's own resolution note struck and corrected** — see below. |
| `.night-crew/knowledge/roadmap.md` | The card's status flip, required in the same change set. Filed next to `test-harness-fail-loud`, which is the same theme (a harness that cannot report failure) and the precedent for a gate card carrying a roadmap bullet. |
| `.night-crew/runs/2026-08-04-autonomous/merge-intents/` | This note. |

**Outside the slate's stated footprint:** `scripts/reset-e2e-db.js` (the slate said "a new spec
under `tests/`" and named the three config files; the helper module is new and is named here so
the merge does not meet it cold).

**`night-crew.toml` is NOT touched, and that is a finding, not an omission.** The slate's footprint
listed it, on the expectation that `[e2e] suite`/`subset` would have to be pointed at a resetting
target. They do not: with the reset inside `webServer.command`, the existing
`npx playwright test` / `npx playwright test "{tags}"` lines reset the database as they stand. No
new `night-crew.toml` key is needed, so the PARK trigger did not fire. `git diff c9e9e9e -- night-crew.toml`
is empty.

**Also NOT touched:** any `backend/**` file, any `*.html`, `sw.js`, `package.json`,
`backend/internal/version/version.go`. `git diff c9e9e9e --stat` names no production code — this
card changes only the harness. (Verified against `c9e9e9e`; see the card report for the SHAs.)

---

## What must survive any merge

1. **`webServer.command` must still begin with `node scripts/reset-e2e-db.js &&`.** That
   placement is the whole card. Two placements that look equivalent are not:
   - a **`globalSetup`** runs AFTER the web server —
     `node_modules/playwright/lib/runner/tasks.js:100-110` puts `createPluginSetupTasks` (the
     `webServer` plugin) ahead of `config.globalSetups` — so it would drop the database out from
     under a server that had already migrated and seeded it;
   - **top-level code in `playwright.config.js`** re-runs in every worker process (verified: the
     worker load carries `TEST_WORKER_INDEX=0`) and also runs under `npx bddgen` and
     `npx playwright test --list`.
   A merge that "tidies" the reset into either of those breaks it in a way the suite will report
   as a product failure.
2. **The guard in `assertTestDatabaseName` must stay, and must stay a throw.** `TEST_DB_NAME`
   that does not match `/^hq_test(?:_[a-z0-9]+)*$/` must abort the run naming the database and
   the pattern. B-16/B-35's lesson runs in both directions: a harness that quietly destroys the
   wrong database is as bad as one that quietly destroys nothing. Do not soften it to a warning
   and do not add an env-var escape hatch.
3. **The reset is DROP + CREATE, not TRUNCATE.** B-76's own lead records that a truncate-and-reseed
   is *strictly weaker* — it fixes accumulation and not anything a run created outside the
   migration set. `tests/db-isolation.spec.js` pins the stronger semantics: its marker table is
   not in the migration set, so a truncate-only implementation leaves it standing and the test reds.
4. **`tests/db-isolation.spec.js` must keep leaving its marker behind.** A version that only reads
   and never writes passes vacuously forever. The write is what makes the NEXT invocation the
   assertion.
5. **The `NIGHTCREW_ENV_URL` skip must stay.** In that mode `webServer` is `undefined`, no reset
   runs, and the database these coordinates name is not the one under test. Asserting there would
   grade the wrong database.
6. **`playwright.config.js:60`'s `serviceWorkers: 'block'` is untouched (B-15).** Byte-identical
   to `c9e9e9e`.
7. **(Fix round) The reset banner must be printed with `console.error`, not `console.log`.** This
   is **B-81** and it is not a style preference. `scripts/reset-e2e-db.js` runs as the first link
   of `webServer.command`, and Playwright's webServer plugin pipes the child's **stdout only when
   `webServer.stdout === 'pipe'`**, which defaults to `'ignore'`
   (`node_modules/playwright/lib/plugins/webServerPlugin.js:126`;
   `node_modules/playwright/types/test.d.ts:10285-10289`). stderr **is** piped by default
   (`:10281-10283`) — which is why the Go server's slog lines reach a gate log and, before this
   fix, the banner did not. Since psql emits no NOTICE on a successful DROP, a gate log then
   carried **zero** evidence the reset had run: item 1 above was true and unobservable. A merge
   that "tidies" this back to `console.log` restores exactly the silence B-76 is about. The
   banner's **content** must also survive — it names the database and the `host:port`, and that
   content is the evidence.

---

## What is safe to drop

- The **prose volume** of the header comment in `scripts/reset-e2e-db.js` and of the block in
  `tests/db-isolation.spec.js`. Compress freely; behaviour is pinned by the test.
- The exact **wording** of the guard's throw message and of the test's failure message, provided
  both still name the offending database.
- ~~The `console.log` line in `reset-e2e-db.js`'s CLI branch, if a merge prefers a quiet gate log —
  though a silent reset is indistinguishable from no reset, which is B-76's own mechanism, so
  prefer keeping it.~~ 🛑 **RETRACTED by the fix round. This bullet was wrong and is the reason
  B-81 existed.** The line is now `console.error` and it is **must-survive item 7**, not
  safe-to-drop. It was never a nicety: with it removed — or, as shipped, present but on a stream
  Playwright discards — a gate log contains no evidence the database was reset, which is B-76's
  own mechanism restated. A merge that drops it, or moves it back to stdout, silently un-does the
  observability half of this card.
- The `psqlUrl` field's *name*. It exists only because libpq rejects the `TimeZone` query
  parameter pgx accepts; any equivalent construction is fine.
- The **deletion** of the two inline `psql` blocks in `Taskfile.yml`. If a merge would rather keep
  them, keeping them is harmless (an extra DROP/CREATE before a DROP/CREATE) — it is only a
  second source of truth, not a defect. **Do not resolve the conflict the other way**, i.e. by
  keeping the Taskfile blocks and dropping the `webServer.command` prefix.

---

## Red-first

**Named test:** `tests/db-isolation.spec.js` ›
`E2E database isolation (B-76) › the e2e database carries nothing over from the previous invocation`

**Captured RED against:** the working tree at base commit **`c9e9e9e`** — `playwright.config.js`
byte-identical to `c9e9e9e`, with only the new test and its helper added and **the fix not yet
applied** (no `node scripts/reset-e2e-db.js` in `webServer.command`).

**How the red was taken — the SUBSET path, twice:**

```
TEST_PORT=8201 npx playwright test "db-isolation" --retries=0   # 1st → 1 passed
TEST_PORT=8201 npx playwright test "db-isolation" --retries=0   # 2nd → 1 failed  ← THE RED
```

`npx playwright test "db-isolation"` is exactly what `night-crew.toml:34`'s
`subset = 'npx playwright test "{tags}"'` expands to — a positional path regex.

Verbatim red:

```
Error: hq_test_e2e still holds the e2e_isolation_marker table left by an earlier invocation
(92539-1785767763704 @ 2026-08-03 14:36:04.035468+00). The database was NOT reset before this
run started, so every result in this suite — red OR green — may be an artifact of accumulated
state (B-76). ...
Expected: ""
Received: "e2e_isolation_marker"
  1 failed
```

Corroborating measurement taken the same minute, on `hq_test_e2e` with no run in flight and
after both subset invocations: `sessions 890 · ops 717 · template_assignments 288 ·
checklist_fields 268 · checklist_sections 193 · checklist_templates 190 ·
checklist_submissions 121`. Those rows are from run `20260803` and earlier. The subset path had
just run twice and removed none of them.

**Green after:** commit **`cd144dd`** — the one-line change that makes `webServer.command` begin
`node scripts/reset-e2e-db.js &&`. The same subset invocation, run three times consecutively:

```
TEST_PORT=8201 npx playwright test "db-isolation" --retries=0
  ✓  1 [chromium] › tests/db-isolation.spec.js:43:3 › E2E database isolation (B-76) ›
       the e2e database carries nothing over from the previous invocation
  1 passed        ← and again, and again: 3 consecutive invocations, 3 passes
```

`hq_test_e2e` immediately afterwards: `purchase_items=106 goose_db_version=75 item_group_tags=20
tags=19 ob_items=13 hq_apps=11 item_groups=10 checklist_fields=7`. `sessions`, `ops`,
`template_assignments`, `checklist_templates` and `checklist_submissions` are **gone** — every row
now present is migration-seeded or was created by the invocation that is looking at it.

**The guard, verified in both directions:**

```
TEST_DB_NAME=hq     node scripts/reset-e2e-db.js       → exit 1, "refusing to reset database \"hq\""
TEST_DB_NAME=hq_dev npx playwright test "db-isolation" → "Process from config.webServer was not
                                                          able to start. Exit code: 1"
```

In the second case the Go server never started and **zero tests ran** — the run aborts rather than
proceeding against an unreset database.

**Full suite after the fix** (`TEST_PORT=8201`, `--retries=0`, nothing else running, ONE summary
block): `Running 786 tests using 1 worker` → **780 passed / 6 skipped / 0 failed, 21.1m**. All four
armed reds passed, matched by FULL TITLE; per decision 100 that **retires nothing** and this card
claims no fix for any of them. 🛑 **This is the first hq full-suite figure in this milestone taken
against a database that was reset before the run** — the pre-fix baseline of 785/778/6/**1** was
taken against an accumulating one, and the single failure it carried,
`tests/onboarding.spec.js:689 › Manager tab › sign-off form requires readiness rating (notes
optional)`, is exactly the test B-76's four-way matrix attributed to accumulated state. It **passed
here at 4.5s**, which corroborates the attribution and is not a claim that this card fixed a
product defect. The `+1` in the test count is `tests/db-isolation.spec.js` itself.


### Fix round — B-81, the banner, verified BY EXECUTION

The B-81 fix is one `console` call, so "I changed `console.log` to `console.error`" is **not**
evidence it works. The property under test is *the banner reaches Playwright's output*, and that
was measured, on the subset shape, with an A/B control inside a **single** invocation: a temporary
`console.log` control line was carried alongside the `console.error` banner, so both streams were
exercised by the same process in the same run.

```
TEST_PORT=8221 TEST_DB_NAME=hq_test_e2e_a1fix HQ_RLS_TEST_DB=hq_rls_a1fix_0804 \
  npx playwright test "db-isolation" --retries=0
```

Line 1 of Playwright's output, verbatim:

```
[WebServer] ── reset hq_test_e2e_a1fix on localhost:5433 ──
```

and over the same log:

```
grep -c CONTROL-STDOUT <log>   ->   0
```

**Same process, same run: the stderr banner reaches the log; the stdout control is discarded.**
That is the mechanism named in must-survive item 7, demonstrated rather than argued. The control
line was then reverted — only the `console.log` → `console.error` change and its explanatory
comment ship.

Suite result on that leg: **`1 passed`**. It was the *second* consecutive invocation against
`hq_test_e2e_a1fix`, so it also re-proves the reset itself — the marker table left by invocation 1
was gone, which is the assertion `tests/db-isolation.spec.js` makes.

**G4, re-run:** `node build-sw.js` twice → `31 files precached (2160.5 KB)` **both times**, and
`git status` clean of `sw.js` / `version.json` after each. The precache count is unmoved, which is
the expectation here — `scripts/reset-e2e-db.js` is not in the precache set (B-37's silent-drop
sentinel would have been the finding had it moved).

**Not re-run, deliberately:** the full 786-test suite and the Go suite. The fix-round diff is one
`console` call in a harness script; it touches no Go code and no production code, and the full
suite figure above was taken after the card's own fix, which this does not alter.

**Isolation used for the fix round** (this is B-80's remedy applied to the fix round itself):
`TEST_PORT=8221`, `TEST_DB_NAME=hq_test_e2e_a1fix`, `HQ_RLS_TEST_DB=hq_rls_a1fix_0804`. 🛑 The
card's original evidence above was taken on `TEST_PORT=8201` against the **bare default**
`hq_test_e2e` — see the note on B-80 below, which is exactly about that.

---

## Fix round — G6 findings filed

G6 returned **APPROVE-WITH-NOTES**. One note was fixed (B-81, above); the rest are **filings only**
and were deliberately **not** fixed — fixing them is scope creep on a card that has already passed
G6. All numbers were allocated by the orchestrator. Filed in `.night-crew/knowledge/BACKLOG.md`:

| # | One line | Where |
|---|---|---|
| **B-80** | Every invocation now DROPs the shared default `hq_test_e2e`, and legs are not issued a `TEST_DB_NAME` — two concurrent legs destroy each other rather than merely polluting each other | `scripts/reset-e2e-db.js:111-131`, `:77`; `playwright.config.js:109` |
| **B-81** | The reset banner went to stdout, which `webServer` discards — **RESOLVED this round** | `scripts/reset-e2e-db.js:140` |
| **B-82** | `tests/db-isolation.spec.js` never runs on a seam-confined subset leg; no `[e2e.seams]` tag matches `db-isolation` | `night-crew.toml:34`, `:36-60` |
| **B-83** | The reset guard is name-only — host, port and credentials unchecked | `scripts/reset-e2e-db.js:93-104`, `:71-91` |
| **B-84** | With `NIGHTCREW_ENV_URL` set the reset is inert and the test skips rather than failing — correct behaviour, but the one silent shape | `playwright.config.js:65`; `tests/db-isolation.spec.js:48-51` |
| **B-85** | `task bdd` points at `DB_PORT: '5432'`, which black-holes SYN here; pre-existing, but the failure now presents as a 60s stall on the reset | `Taskfile.yml:82-86`; `playwright.config.js:116` |
| **B-86** | `verify-test-harness.sh`'s A2 floor is stale at 20 while the repo resolves 27 | `scripts/verify-test-harness.sh:78`, `:74-75` |

### 🛑 The false `TEST_DB_NAME` premise — corrected in B-76, and it was never in this note

**This note does not contain the false premise and never did** — grep it for `TEST_DB_NAME` and the
only hits are the guard requirement (must-survive item 2) and the two guard-verification commands.
Nothing here claims cards are issued their own database. Recorded explicitly so a merger does not
have to re-derive that.

**B-76's resolution note in `BACKLOG.md` did contain it**, and it has been struck in place. The
sentence declined the per-run-database alternative on the grounds that *"the collision half is
already carried by `TEST_DB_NAME` + `TEST_PORT` (every card is issued its own pair)"*. **Both
halves of that are false**, and the code says so:

- Cards are **not** issued a `TEST_DB_NAME`. The isolation stanza in `launch-20260803.md:97-98`
  names `HQ_RLS_TEST_DB`, `TEST_PORT` and a scratchpad directory — and **tonight's own
  `launch-20260804.md` carries no isolation stanza at all** (grep it for `unique`, `TEST_PORT`,
  `TEST_DB_NAME`, `HQ_RLS_TEST_DB`: every one absent, as in `slate-20260804.md`). This card's own
  implementer ran `TEST_PORT=8201` against the bare `hq_test_e2e`, which is the demonstration.
- A per-card name does **not** "defeat `assertTestDatabaseName`'s point-blank name check".
  `TEST_DB_NAME=hq_test_e2e_a1fix` matches `/^hq_test(?:_[a-z0-9]+)*$/` and was measured resetting
  cleanly through the gate path this round — it is the database every figure in the fix-round
  evidence above was taken against.

A resolved entry justifying itself with a false premise is how the next reader gets misled, which
is why the correction is written into B-76 itself and not only into B-80.
