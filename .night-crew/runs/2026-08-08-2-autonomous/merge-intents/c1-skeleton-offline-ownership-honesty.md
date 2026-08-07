# Merge intent — C1 · `skeleton-offline-ownership-honesty`

Run `20260808-2` · branch `card/c1-skeleton-offline-ownership-honesty` · closes **B-88**.

---

## Scope

The rule *"nothing may read from RxDB on a code path that can execute offline"* was enforced by
three `expect(src).not.toContain(...)` assertions over SOURCE TEXT in
`tests/sync-rxdb-client.spec.js` (`HQSync.createDatabase`, `HQSync.startReplication`,
`HQSync.client`). `workflows.html`'s `defaultStore()` (~line 3589, the conflict-notice wiring)
reads `window.HQSync.db` — a fourth route none of the three named — and the guard is green today
only because no database exists yet, not because it would catch one appearing. The very next card
on this slate (`skeleton-one-row-end-to-end`, C2) is the first to call `createHQSyncDatabase()` in
production, so the gap was about to become live.

Fix: delete the three source-text `.not.toContain` assertions and replace them with an
OBJECT-level assertion against a REAL running page — `window.HQSync.db` must be `undefined` at end
of `/workflows.html` load, plus a defense-in-depth check that no RxDB-backed IndexedDB database
exists either (RxDB's Dexie storage always backs a collection with a real IndexedDB database, so
this holds regardless of which global variable, if any, still points at a live instance).

A source-text ban on the literal `HQSync.db` would have been the WRONG fix, not just an
incomplete one: `defaultStore()`'s read is already gated (`db && db.conflict_records`) — exactly
the shape a flag-gated read path is supposed to have — so banning the spelling would red on code
that is already safe. The rule is about the object the page ends up holding, not the identifiers
in the file.

---

## Shared files touched

| File | Why |
|---|---|
| `tests/sync-rxdb-client.spec.js` | Owned by this card — the three `.not.toContain` assertions replaced with the object-level browser test (B-88). |
| `.night-crew/knowledge/roadmap.md` | Card status flip for `skeleton-offline-ownership-honesty`, per that file's own convention. |

`workflows.html` was read for the investigation (confirmed `defaultStore()`'s exact read at
~line 3589) but **not edited** — the card's own risk note said edits there would only be needed
"if the guard needs a hook," and it does not: `window.HQSync.db` is already directly readable from
`page.evaluate()`, no production hook required.

`sync-rxdb/bootstrap.js` was touched ONLY as a temporary, uncommitted RF probe (see below) and is
byte-identical to its pre-card state in every commit on this branch — confirmed via `git diff
--stat` showing no changes before each commit below.

Nothing else. No backend Go file, no other HTML tool page, no `night-crew.toml` entry, no
`docker-compose*.yml`, no `Taskfile.yml`, no version bump (not this card's job).

---

## What must survive any merge

- **The new test's name and location**: `tests/sync-rxdb-client.spec.js`, inside
  `describe('workflows.html actually imports and constructs the client')`, titled `'nothing reads
  from RxDB on a code path that can execute offline — window.HQSync.db is undefined at end of load
  (B-88)'`. Any card that reshapes this spec file must keep an equivalent object-level assertion —
  do not let it regress back to a source-text check.
- **The assertion itself**: after `login(page)` → `page.goto('/workflows.html')` → wait for
  `window.HQSync !== undefined`, `window.HQSync.db` must be `=== undefined`, AND
  `indexedDB.databases()` must contain no database whose name matches `/rxdb|hq_sync/i`.
- **The two positive `toContain` assertions** in the sibling test (`debouncedSaveField(`,
  `submitOp('SET_FIELD'`) are UNCHANGED — this card touched only the three negative assertions.

## THE FLAG-OFF CONTRACT — what C2 (`skeleton-one-row-end-to-end`) must satisfy

C2 lands the flag and the first production call site of `createHQSyncDatabase()` /
`startHQReplication()`. This card's guard is the contract C2 has to keep green:

> **With the sync flag OFF (the default), page load of `/workflows.html` must leave
> `window.HQSync.db` `undefined`.** `createHQSyncDatabase()` may be called by C2's code only inside
> a branch gated on the flag being ON — never unconditionally at module load, and never assigned to
> `window.HQSync.db` before that gate is checked. If C2's flag defaults ON in any environment this
> suite runs against, or if the database is created before the flag check, this test goes red and
> that is correct — it is B-88 recurring, not a flaky guard.

C2 is free to make `window.HQSync.db` become defined **when the flag is explicitly ON** — that is
exactly what C2 exists to build, and this test does not run in that configuration. What it pins
down is the OFF state, which must remain the default this suite exercises.

## What is safe to drop

- The prose/comments explaining *why* (informative, not load-bearing).
- The RF probe logs under `.night-crew/runs/2026-08-08-2-autonomous/c1-gates/` are evidence only,
  regenerable by re-running the commands recorded below.

---

## Red-first

This card's deliverable is a test file, not application behavior — there is no production code
whose defect needed a red test written against it. Two probes instead, both against the CURRENT
tree, per the RF gate's "capture the old guard's inadequacy concretely" option.

**Probe A — the shipped guard's blindness to the fourth route, concretely.** A Node script
evaluated the CURRENT (unmodified, pre-fix) shipped assertions plus a hypothetical fourth
(`expect(src).not.toContain('HQSync.db')`) against `workflows.html`'s actual source:

```
$ node -e '<probe script — see command below>'
green  not.toContain("HQSync.createDatabase")  present=false
green  not.toContain("HQSync.startReplication")  present=false
green  not.toContain("HQSync.client")  present=false
RED   not.toContain("HQSync.db (the 4th route, unwatched by the shipped guard)")  present=true
EXIT=1
```

Full command + log: `.night-crew/runs/2026-08-08-2-autonomous/c1-gates/rf-probe-a.log`. This
proves, mechanically rather than by inspection, that `workflows.html` already contains the literal
`HQSync.db` (the route the roadmap card names) and that the three assertions the shipped guard
actually runs are indifferent to it — all three stay green regardless.

**Probe B — the new object-level test actually catches a violation.** Wrote the new test first
(the code under test), then exercised it against three tree states:

1. **Baseline (unmodified tree)** — `window.HQSync.db` genuinely undefined (nothing in the tree
   creates a database yet):
   ```
   TEST_PORT=4311 TEST_DB_NAME=hq_test_c1impl HQ_RLS_TEST_DB=hq_test_c1impl_rls \
     npx playwright test tests/sync-rxdb-client.spec.js -g "B-88" --project=chromium
   1 passed (2.5m)
   ```
   Log: `c1-gates/rf-newtest-baseline-green.log`.

2. **Injected violation** — temporarily added one line to `sync-rxdb/bootstrap.js` right before
   `window.HQSync = HQSync;`: `HQSync.db = { conflict_records: {} };` (simulating an unguarded
   future card setting `.db` on every load, flag or no flag). Same command:
   ```
   ✘ nothing reads from RxDB on a code path that can execute offline — window.HQSync.db is
     undefined at end of load (B-88)
     Error: expect(received).toBe(expected)
     Expected: true
     Received: false
       > 1548 |     expect(state.dbUndefined).toBe(true);
   1 failed
   EXIT=1
   ```
   Log: `c1-gates/rf-newtest-injected-red.log`. **RED, exit 1** — the new guard catches exactly the
   violation the old three assertions were blind to.

3. **Reverted** — `sync-rxdb/bootstrap.js` restored to its exact pre-probe state (`git diff --stat`
   confirmed empty before this card's commits). Same command:
   ```
   ✓ nothing reads from RxDB on a code path that can execute offline — window.HQSync.db is
     undefined at end of load (B-88) (929ms)
   1 passed (7.2s)
   EXIT=0
   ```
   Log: `c1-gates/rf-newtest-postrevert-green.log`. **GREEN, exit 0.**

Together, Probes A and B are the red-before-green pair the ladder asks for: A shows the defect
(the shipped guard's blind spot) mechanically, on the CURRENT tree, before any fix; B shows the
replacement guard is a real, working gate — red on the violation, green on its absence — not a
tautology that always passes.

---

---

## G2 (Playwright) — two non-armed reds observed, investigated, ruled flake-trail

The full suite (de-confined — `tests/sync-rxdb-client.spec.js` is not listed under any
`night-crew.toml` `[e2e.seams]` key) ran 799 tests in one summary block (one `Running 799 tests`
header, one final `2 failed / 6 skipped / 791 passed (22.5m)` triplet — confirmed via
`grep -c "^Running "` = 1 and `grep -c "passed ("` = 1, so the run is valid, not invalidated):

```
2) [chromium] › tests/inventory.spec.js:3124:3 › Retry parse button (260607-koi) › is hidden when
   parse_error is empty
   Error: page.waitForLoadState: Test timeout of 30000ms exceeded.

2) [chromium] › tests/sync.spec.js:1327:3 › Convergence matrix (W-3): surviving answers converge
   across devices › checkbox answer converges (live + catch-up)
   Error: expect(locator).toHaveClass(expected) failed — Timeout: 5000ms
```

Neither is the armed-reds baseline (`inventory.spec.js:883` B-27, `sync.spec.js:446` LST-17,
`receipt-carousel.spec.js:123` B-162) — all three of those actually PASSED in this run
(verified by grep: lines 223, 712, 1327 of the full log are `✓` for exactly those three tests).
So these two are reds beyond the baseline and needed their own accounting, not a pass-through.

**Could this card's diff plausibly cause them?** No: `git diff --stat 549e83a HEAD` (base of this
branch → tip) touches exactly `tests/sync-rxdb-client.spec.js` and this run's own docs/logs —
zero lines of `workflows.html`, zero lines of any `backend/` package, zero lines of
`sync-rxdb/bootstrap.js` (confirmed byte-identical via `git diff --stat` before every commit,
see Red-first §Probe B). Both failing tests are in unrelated spec files exercising unrelated
features (a receipt pending-review card's retry-parse visibility; a cross-device checkbox
convergence matrix) that share no code path with client construction or `window.HQSync.db`.

**Reproduced?** Re-ran both, isolated, `tests/`-anchored, same env
(`TEST_PORT=4311 TEST_DB_NAME=hq_test_c1impl HQ_RLS_TEST_DB=hq_test_c1impl_rls`), `--retries=0`:

```
npx playwright test tests/inventory.spec.js tests/sync.spec.js \
  -g "is hidden when parse_error is empty|checkbox answer converges \(live \+ catch-up\)" \
  --retries=0 --project=chromium
...
2 passed (15.2s)
EXIT=0
```

**Did not reproduce.** Both pass cleanly in isolation. Ruled: flake-trail evidence, not a card
failure — consistent with the gate ladder's own note that `sync.spec.js` is "the load-sensitive
one that must not share a box" (decision 100's flake trail) and with `inventory.spec.js:3124`'s
failure mode (a bare `networkidle` timeout at test 87-of-799-deep into a 22.5-minute run, not an
assertion mismatch). Not silently dropped: named here, and both full-suite and isolated-rerun logs
are committed under `c1-gates/` for anyone who wants to re-litigate the call.

Log: `c1-gates/rerun-suspect-reds.log`. Full-suite log: `c1-gates/g2-pw-full.log`.

---

## Nothing else

No PARK condition was hit. No backend Go change. No API contract change. No `docker-compose*.yml`,
no `Taskfile.yml`. `main`, `dev`, and every other card's worktree/branch (in particular
`a3-rls-fixture-own`, untouchable per decision 155) are untouched.
