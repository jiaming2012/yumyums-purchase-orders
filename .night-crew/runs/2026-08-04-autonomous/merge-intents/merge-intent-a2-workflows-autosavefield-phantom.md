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
- **Green after:** `0cf24b9`. Isolated re-run: `1 passed (10.5s)`. In the full gate below it is
  `✓ 290 … [FLD-16B] (4.0s)`.

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
| `.night-crew/knowledge/designs/fetchstorm-replay-class-superseded.md` | **Added 2026-08-04, A2 fix round (G6 finding 3).** `:56` was the ninth doc site and the only one left uncorrected; it named `autoSaveField` → `POST /saveResponse` as the live write path with no annotation. **Annotated in place rather than added to "deliberately left undone"** — it costs one bracketed line and does not alter the record's meaning (the note's argument turns on the journal, not on the writer's name), and unlike the run artifacts in that list this file is a durable `knowledge/designs/` reference cited by Engineering KR1 as evidence, so a reader is expected to follow it. |

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

   🛑 **RECORDED 2026-08-04, A2 fix round (G6 finding 2) — B-65's second lead is PARTIALLY,
   not fully, discharged, and the note above overstated it.** B-65's lead asked for *"a symbol
   rather than a substring, since substring-matching source is what let a phantom function look
   tested."* What shipped is `expect(src).toContain('debouncedSaveField(')` — **strictly better**
   (the trailing `(` means no comment currently in `workflows.html` satisfies it; G6 confirmed
   none do) but **the same class of check**: it still matches source text, not a resolved symbol.
   **Failure scenario:** someone deletes the real call at `:2219` while writing
   `// call debouncedSaveField(fieldId, value) here` in a comment — the assertion passes again,
   and the exact B-65 shape recurs. **Deliberately NOT fixed in this round:** the round is
   docs-only, the file is code, and `[FLD-16B]` in `tests/persistence.spec.js` now provides
   genuine *behavioural* coverage behind this assertion — which is the coverage that actually
   would have caught B-65. Anyone hardening this later wants a parse or a runtime probe, not a
   longer substring.

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

1. **Historical run artifacts under `.night-crew/runs/*/` are untouched** — six of them name
   `autoSaveField`. They are records of what was believed at the time, and two of them
   (`2026-08-03-autonomous/HANDOFF.md`, `park-s1b-sync-hard-cutover.md`) are the artifacts
   that *found* the phantom. Rewriting a run's record would be worse than leaving it.
2. **The three signed slates and `okr-completion-plan-20260804.md` are untouched**, same
   reason — a signed slate is a record of what was signed.
3. **`docs/codebase/*` was not audited for this phrase.** Out of scope for the time box; the
   files CLAUDE.md's own summary sections derive from are `docs/codebase/`, so it is worth
   a grep at some point. Reported as a note, not filed as a finding.
4. **`docs/data-flow-audit.md` rows 9 and 10 are still open defects** (`REJECTION_FLAGS`
   never reloaded from the server; `WAS_REJECTED` dead code). They were marked BUG and DEAD
   CODE in the audit before this card and still are. Not mine to fix.

## New finding for the orchestrator

**Candidate B-87 — a Playwright path filter matches the ABSOLUTE path, so a worktree
directory name containing a seam token silently changes which specs the confined gate
runs.** This worktree is `hq-worktrees/a2-workflows-autosavefield-phantom`, and the seam
subset for `workflows.html` is `["workflows","persistence","sync","repro-cut-task"]`.
`npx playwright test workflows persistence sync repro-cut-task` from here selected **787
tests — the entire suite** — because the directory component `a2-workflows-…` matches the
`workflows` regex on every file in the tree. Tonight it widened coverage, which is harmless
and in fact strengthens this card's evidence. ~~The direction that is *not* harmless is the
same mechanism under a differently-named worktree: a card branch named, say,
`b-inventory-…` would make `[e2e] subset` select the inventory specs **instead of** the
seam's, and the gate would report a green subset that never ran the seam it was confined to.~~

🛑 **CORRECTED 2026-08-04, A2 fix round — the struck sentence above is WRONG. G6 refuted it at
source and by execution, and it is struck rather than reworded so the record shows what was
claimed and why it did not hold.** The *mechanism* is real; the *consequence* does not follow.
CLI path filters are **OR'd**, not intersected — so a directory-component match can only ever
produce a **superset**, and can never exclude a spec that would otherwise match. The
under-coverage failure mode described above is **unreachable by this mechanism**.

- `node_modules/playwright/lib/util.js:183` `forceRegExp` → `new RegExp(pattern, "gi")` —
  unanchored, matches anywhere in the string.
- `util.js:128` `createFileMatcher` → `re.test(filePath)` against the path handed in; its
  returned closure (`:141-160`) returns `true` on the **first** regex that hits.
- `util.js:124-127` `createFileMatcherFromArguments` folds **all** positional args into **one**
  matcher; `runner/tasks.js:243` pushes that single matcher onto `config.loadFileFilters`, so
  `loadUtils.js:68`'s `loadFileFilters.every(…)` runs over a **one-element** array — hence OR.
- `runner/projectUtils.js:147-161` `collectFilesForProject` collects from the resolved absolute
  `testDir`; `runner/loadUtils.js:63-71` feeds those absolute paths to the filter.
- Empirical (`--list --project=chromium`, re-measured in this fix round): `workflows` → **86
  tests / 1 file** from `/home/jcole/projects/hq`, but **786 / 26 files** — the entire chromium
  suite — from this worktree. `workflows inventory` → **238** = 86 + 152 (`inventory` alone =
  152), confirming the OR. And decisively: `persistence` **alone** from this worktree → **32 /
  1 file**, i.e. a token the directory name does *not* contain is not widened at all.

**Corrected consequence, as filed in B-87:** a worktree directory name containing a filter token
silently turns a confined subset into the **full suite**. The gate **over-runs** — its runtime,
its isolation assumptions, and its "which tests ran" claim are all not what they say. This is a
sibling of B-76 in kind — *the harness cannot tell you what it actually ran* — but it is **not**
a silent-skip risk, and the confined-gate-misses-a-regression direction is unreachable this way.

Lead (recorded, not implemented): anchor the `[e2e.seams]` tokens to `tests/…` so they cannot
match a directory component, and have the gate print the selected file list.

---

## Gate evidence (G1–G4; G6 is the orchestrator's)

Isolation used, all three, per B-80: `TEST_PORT=8202`, `TEST_DB_NAME=hq_test_e2e_a2`,
`HQ_RLS_TEST_DB=hq_rls_a2_0804`. The Go suite additionally got its own
`hq_test_go_a2` rather than the shared `hq_test_go`. `HQ_SYNC_REST_URL` **unset**;
`HQ_SYNC_SUBSTRATE_OPTIONAL` **unset**. Legs were sequenced, never overlapped —
`sync.spec.js` is load-sensitive and was in the selection.

- **G1** — from `backend/`: `go build ./...` exit **0**; `go vet ./...` exit **0**.
- **G2 (Go)** — `go test -p 1 -count=1 ./...` exit **0**. Counts, not just `ok`:
  **439 tests ran, 437 passed, 0 failed, 2 skipped.** Per package —
  alerts 3, auth 18, inventory 72, purchasing 25, receipt 72, recipes 61, sync 142,
  toast 11, **workflow 35** (non-zero, so `DB_TEST_URL` took effect — B-35's failure mode
  did not fire). The 2 skips are `TestProxyLive_RealtimeUpgrade` and
  `TestProxyLive_RESTRequest`. RLS evidence as required: `-run TestRowVisibilityRLS -v`
  shows the subtests **ran** — **59 subtests PASSED** under FLOOR / CONTROL / POSITIVE /
  NEGATIVE, `ok github.com/yumyums/hq/internal/sync 7.416s`.
- **G2 (Playwright)** — `npx bddgen` exit **0**, then `npx playwright test … --retries=0`.
  **Exactly one summary block, one `Running N tests` header** (verified by grep; not two).
  Verbatim:

  ```
    1 failed
      [chromium] › tests/sync.spec.js:1343:3 › Convergence matrix (W-3): surviving answers converge across devices › yes/no answer converges (live + catch-up)
    6 skipped
    780 passed (21.2m)
  ```

  **Judged against "green except the armed reds," matched by FULL TITLE.** The single
  failure IS armed red #1, `yes/no answer converges (live + catch-up)`. It **stays ARMED** —
  nothing here retires it. The other three armed reds passed, which **also retires nothing**
  (decision 100 / T-31 decision 120): `item modal pre-fills search with current line item
  text` ✓, `a queued submission still lends its idempotency_key at 7:30pm CT [A1-TZ-02]` ✓,
  `submitted checklist survives builder edit with assignment change [LC-02]` ✓.

  🛑 **This was the FULL suite (787), not the confined subset — and not by choice.** See
  candidate B-87 below: the subset invocation selected every file in the tree because this
  worktree's directory name contains the token `workflows`. Reported as the full suite,
  which is strictly more evidence than the card was owed.

  🛑 **Measured against a FRESH database.** A1 landed earlier tonight, so
  `scripts/reset-e2e-db.js` DROP/CREATEs `hq_test_e2e_a2` as the first act of the
  invocation. This is among the first cards in this milestone whose figures are comparable
  to nothing before it — earlier runs' numbers were taken against an accumulating DB.

- **G3** — **N/A.** `openspec: absent`, re-confirmed at launch. No OpenSpec scaffolding
  created.
- **G4** — `node build-sw.js`: **31 files precached (2162.2 KB)**, frontend version 1.4.0.
  **Precache count unchanged at 31** — no asset added or removed, so B-37's silent drop did
  not recur. Import reachability: 18 precached files parsed, 30 local references resolved,
  0 outside the precache. **Idempotent** — a second run left the tree clean. Version parity
  holds: `version.go Frontend = "1.4.0"` ≡ `package.json "1.4.0"` ≡ `version.json
  {"frontend":"1.4.0"}`. Regenerated and committed **after** the content commits (B-37).

---

## Green after
