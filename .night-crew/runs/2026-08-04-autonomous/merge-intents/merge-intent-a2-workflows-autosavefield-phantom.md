# Merge intent — A2 `workflows-autosavefield-phantom`

- **Run:** `overnight-20260804`
- **Branch:** `card/a2-workflows-autosavefield-phantom`
- **Base commit:** `bb6ff80` — every diff claim in this note is measured against
  `bb6ff80`, never against run-branch HEAD (T-32 decision 130a).
- **Closes:** B-65

---

## What this card does, in one line

`workflows.html:2219` called `autoSaveField(...)` — a function that exists nowhere in
the tree — as the last statement of the fail-photo upload chain. It is replaced with the
real live write path, `debouncedSaveField(fldId, resp ? resp.value : null)`, and the five
documentation sites that named the phantom as the live write path are corrected.

---

## Red-first

- **Named test:** `fail photo captured through the camera path survives back-to-list and
  reopen [FLD-16B]` — `tests/persistence.spec.js`.
- **Captured RED against:** tree `bb6ff80` + the test file alone (no production change
  present). Command:
  `TEST_PORT=8202 TEST_DB_NAME=hq_test_e2e_a2 npx playwright test tests/persistence.spec.js -g "FLD-16B" --retries=0`
- **Red output, verbatim from the run:**

  ```
  Expected pattern: not /Fail photo upload failed/
  Received string:      "Fail photo upload failed: ReferenceError: autoSaveField is not defined
      at http://localhost:8202/workflows.html:2219:9
  Failed to load resource: net::ERR_NAME_NOT_RESOLVED"
  ```

  and, from the server's client-log relay in the same run:

  ```
  "message":"Fail photo upload failed: autoSaveField is not defined\nReferenceError: autoSaveField is not defined\n    at http://localhost:8202/workflows.html:2219:9"
  ```

  This is a genuine, reachable `ReferenceError` in shipped code — not a dead branch.
- **Green after:** recorded below once the fix commit lands. *(filled in at fix commit —
  see "Green after" at the bottom of this file.)*

### Why the existing suite never caught it

`tests/persistence.spec.js` already had `[FLD-16]`, "fail photo survives back-to-list and
reopen as https:// URL" — but it injects the photo with `POST /saveResponse` directly,
skipping every line of client code between the presign response and the write. And
`tests/sync-rxdb-client.spec.js:1460` asserted `expect(src).toContain('autoSaveField')`,
which passed on **comment text alone**. Two tests that read as coverage of a function that
does not exist. `[FLD-16B]` drives the real path (presign + PUT intercepted at the network
layer, hidden file input fed through Playwright's filechooser).

The `ReferenceError` never reached `page.on('pageerror')`: the upload chain's own
`.catch()` swallows it, and because `FAIL_NOTES[fldId].photo` is mutated one line *before*
the throw, the thumbnail still rendered. The crew member saw "photo attached" and nothing
was persisted.

---

## Shared files touched

| File | Why |
|---|---|
| `CLAUDE.md` | Its "Workflows Data Persistence Rule" prescribes `autoSaveField(fieldId, value)` as the mandatory write path. **The rule is right; only the name and call shape are wrong.** Renamed to `debouncedSaveField`, corrected the pipeline to `submitOp('SET_FIELD') → POST /ops`, and corrected the metadata-bundling step to describe what `debouncedSaveField` actually reads (`store.get('failNotes', …)` + `CORRECTION_PHOTOS`). The 4-step procedure, the 7-persisted-states list, the required-test template and the "feature is not complete without this test" bar all survive verbatim in force. |
| `.night-crew/knowledge/roadmap.md` | Two prose sites name `autoSaveField` → `/saveResponse` as the live path; plus the A2 card status flip for this run. |
| `.night-crew/knowledge/BACKLOG.md` | B-65 marked resolved. |
| `sw.js` | Regenerated after the final content commit (`workflows.html` is precached). `build-sw.js` reads **git HEAD**, so regeneration is deliberately last (B-37). Expect the merger to re-run it — idempotent, not a conflict. |
| `sync-rxdb/bootstrap.js` | Header comment names `autoSaveField` → `/saveResponse` as the untouched live path. Corrected in place; the "IMPORT + CONSTRUCTION ONLY" claim it is making is unchanged and still true. |
| `sync-rxdb/conflict-notice-ui.js` | Same, one comment block. Corrected in place. |
| `docs/data-flow-audit.md` | **Outside the card's stated footprint.** All 7 rows of the state-inventory table and the rule paragraph name `autoSaveField` → `POST /saveResponse`. This is the file CLAUDE.md points at for the full state inventory — leaving it would have made the CLAUDE.md correction unfollowable. |
| `README.md` | **Outside the stated footprint.** Line 49 carries the same phantom pipeline. |
| `.claude/skills/save-project/SKILL.md` | **Outside the stated footprint.** Line 53 restates the persistence rule to a skill that runs on every save. |
| `tests/sync-rxdb-client.spec.js` | **Outside the stated footprint, but INSIDE this card's gate subset** (the `sync` token selects it). Its `expect(src).toContain('autoSaveField')` was the substring assertion that let the phantom look tested; it goes red the moment the phantom is removed. Replaced with symbol assertions on the real path, per the lead recorded in B-65 itself. |
| `workflows.html` | The fix (line 2219) plus two large banner comments (`:320`, `:3553`) that state the phantom as the live path. |

---

## What must survive any merge

1. **`workflows.html`'s fail-photo write call.** It must call `debouncedSaveField`, and it
   must pass `resp ? resp.value : null` — **not** the original
   `FIELD_RESPONSES[fldId].value || FIELD_RESPONSES[fldId]`. That expression was wrong on
   its own terms: a fail card exists precisely when the answer is falsy, so
   `false || resp` evaluates to the whole response **object**, which would have been
   written as the field's value. The correct shape mirrors the correction-photo path at
   `:2151`, which has been right all along.
2. **`tests/persistence.spec.js` `[FLD-16B]`.** This is the card's red-first artifact and
   the CLAUDE.md-mandated back-and-reopen test for the fail-photo path. It must not be
   weakened into another `saveResponse` injection — the injection is exactly what hid the
   bug.
3. **CLAUDE.md's persistence rule as a *rule*.** If a merge has to choose, keep the rule's
   requirements (one write path, cached, rehydrated on reopen; required back-and-reopen
   test) over any wording of mine. Only the function name and call shape were corrected.
4. **`tests/sync-rxdb-client.spec.js`'s intent** — "no user write path was rerouted".
   The assertions now name symbols (`debouncedSaveField`, `submitOp('SET_FIELD'`) instead
   of matching a substring that a comment could satisfy. Keep the symbol form.

## What is safe to drop

- Any of my comment rewording in `sync-rxdb/bootstrap.js`, `sync-rxdb/conflict-notice-ui.js`
  or the two `workflows.html` banner comments, **provided the resulting text does not name
  `autoSaveField` as a live path.** If a conflicting hunk wins, that is fine as long as the
  phantom does not come back in it.
- `sw.js` in its entirety — regenerate it after the merge, do not hand-resolve it.
- My wording in `docs/data-flow-audit.md` and `README.md`, on the same condition as above.

## Conflicts I expect

- `.night-crew/knowledge/roadmap.md` and `BACKLOG.md` — every card in this run edits them.
  Resolve by union.
- `sw.js` — regenerate, never merge.
- **Nothing else.** No other card on this slate is declared against `workflows.html`.

## Backend / migrations / API surface

**Nothing here.** No file under `backend/` is touched, no migration, no API contract, no
config, no dependency. Verified by `git diff --stat bb6ff80..HEAD`.

## Deliberately left undone

*(filled in at the final commit — see the report.)*

---

## Green after

*(filled in at the fix commit.)*
