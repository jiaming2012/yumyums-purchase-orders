# Merge intent — Card H1 `test-harness-fail-loud`

Branch: `card/h1-test-harness-fail-loud` (cut from `overnight-20260729-2` @ `9bd9a72`)
Written BEFORE implementation, as the card's first commit. Required per DESIGN §15ad.65.

## Card in one line

Make a broken test environment **fail** instead of **pass**. (1) `task test`'s `deps` omits
`bdd:gen`, so every worktree runs 19 of 20 spec files with no error and no skip line. (2) Where
`DB_TEST_URL` is **set but unreachable**, the Go suite `t.Skip`s and prints a bare `ok` —
destroying the environment is indistinguishable from passing it. Keep `t.Skip` **only** for
*unset*.

## Shared files touched

- **`Taskfile.yml` (repo ROOT) — the card's core, and the one file other cards declare as a HARD
  CONSTRAINT ("untouched").** This card is the exception: it is the card that exists to change it.
  The edit is **one line** — `bdd:gen` added to `test:`'s `deps` list at `:28-30` — plus the
  comment explaining why. **No task is added, renamed or removed; no `vars:` default is changed;
  `test:ui`, `test:go`, `bdd:`, `bdd:gen`, `test:all`, `sw`, and every `prod:*` target are
  untouched.** A merge that has to pick a side must keep `bdd:gen` in `test:`'s deps and take
  everything else from the other side. `backend/Taskfile.yml` is NOT touched.
- **`backend/internal/testdb/testdb.go` — NEW package, OUTSIDE the card's stated footprint
  (`Taskfile.yml`, `backend/internal/*/**_test.go`). Disclosed deliberately.** It holds the single
  copy of the failure message and the TestMain-shaped gate, so the eight conversion sites share
  ONE pattern rather than eight drifting inline copies (the card says "copy that pattern; do not
  invent a second one" — eight copies is eight patterns waiting to happen). It is imported **only
  from `_test.go` files**, so it is compiled into no shipped binary. New file, unique to this
  card, zero conflict surface. If a merge must drop it, the eight call sites must be inlined —
  dropping the package alone breaks the build.
- **Five `TestMain` conversion sites** — each keeps its own fallback DSN and its own
  migrate/seed steps; only the two `os.Exit(m.Run())` early-outs change:
  - `backend/internal/workflow/stable_identity_test.go:20-42`
  - `backend/internal/receipt/worker_test.go:20-42`
  - `backend/internal/inventory/period_summary_test.go:23-47`
  - `backend/internal/auth/permission_test.go:35-60`
  - `backend/internal/purchasing/scheduler_cron_test.go:34-57`
  These are `TestMain`s: the per-test `t.Skip("DB unreachable — set DB_TEST_URL")` lines scattered
  through `requires_approver_test.go`, `sync_receipts_test.go`, `failnote_upsert_test.go`,
  `approval_feedback_test.go`, `resubmit_photo_gate_test.go`, `submission_status_test.go`,
  `reprocess_pending_test.go`, `trends_test.go` and the rest all key off the package-level
  `testPool == nil` that `TestMain` sets. **Fixing the five `TestMain`s fixes all of them** — with
  `DB_TEST_URL` set and dead, the binary now exits before any of those skips is reached. Those
  downstream files are therefore **NOT edited**, even though the card names line numbers in some
  of them. That is deliberate: converting them individually would be the second pattern.
- **Three per-test-helper conversion sites** — these already `t.Skip` on *unset* first, so by the
  time they reach connect/ping the URL is always explicitly set and the branch is unconditionally
  `t.Fatalf`:
  - `backend/internal/recipes/helpers_test.go:33-40` (`setupTestDB`)
  - `backend/internal/sync/access_test.go:27-34` (`setupAccessTestDB`)
  - `backend/internal/sync/jwtbridge_test.go:167-174` (`hqTestPool`)
- `scripts/verify-test-harness.sh` — **NEW file, also outside the stated footprint. Disclosed.**
  The card's two reds, made re-runnable so G6 can falsify them by checkout instead of by trusting
  pasted output. ~~Check A: `task --dry test` must list `npx bddgen`.~~ **STRUCK — see "Repair
  round" (N2): Check A alone grades only the mechanism; the graded property now lives in the new
  Check A2 (spec-file count ≥ 20).** ~~Check B: with `DB_TEST_URL`
  pointed at a dead port, the Go suite must exit non-zero.~~ **STRUCK — see "Late additions" (2):
  the default DSN names a nonexistent database on a LIVE Postgres, not a dead port.** New file, no
  conflict surface. Safe to drop *after* the merge lands — it asserts, it does not implement.
- `.night-crew/knowledge/roadmap.md` — this card's `PLANNED` → `DONE` status flip at ~`:528`, in
  the same change set as the work. Every card tonight edits its own card in this file; conflicts
  are per-card and **both sides should be kept**.
- `.night-crew/runs/2026-07-29-2-autonomous/merge-intent-h1-test-harness-fail-loud.md` — this
  note. New file, unique to this card. No conflict surface.

### Late additions

_(appended only if implementation forces a file outside the list above; "nothing here" if it stays
clean. Per B-11, if a later round contradicts anything above, the contradicted line is STRUCK in
place — not merely appended to.)_

**No file outside the list above was edited.** The footprint held exactly as declared: the root
`Taskfile.yml`, the new `backend/internal/testdb/testdb.go`, the five `TestMain`s, the three
per-test helpers, `scripts/verify-test-harness.sh`, the roadmap flip, and this note. Five things a
merge should nonetheless know, and **one line above is struck** because implementation contradicted
it:

1. **`node_modules/` was installed in this worktree via `npm ci`.** Git-ignored, part of no commit.
   **`package-lock.json` was NOT modified** — `npm ci` installs *from* the lock, it does not write
   it. Hard constraint 1 holds. `.features-gen/` was likewise generated by `npx bddgen`; it is
   git-ignored (`.gitignore:9`) and is part of no commit either — which is precisely why `task
   test` failing to regenerate it was invisible.
2. **🛑 STRUCK ABOVE: the script's Check B does NOT use a dead port by default.** The note said it
   would, and the card says "point `DB_TEST_URL` at a dead port". On this WSL2 host a closed TCP
   port **black-holes the SYN instead of refusing it**, so every package sat ~120s before printing
   its (pre-fix) `ok`:
   `ok github.com/yumyums/hq/internal/workflow 120.047s`. Same verdict, 500× slower, and the delay
   is an artifact of the host rather than of the harness. The default DSN therefore names a
   database that **does not exist on a Postgres that does** — which is both instant *and* the exact
   shape of the B-16 incident (a reviewer ran `DROP DATABASE hq_test_go_c` mid-run). The dead-port
   variant is retained behind `H1_DEAD_PORT=1`. **Both forms were observed red pre-fix and green
   post-fix**; only the default changed.
3. **Two files carry pre-existing `gofmt` drift that this card did NOT introduce and did NOT
   fix**: `receipt/worker_test.go` and `workflow/stable_identity_test.go` were already
   `gofmt -l`-dirty at `9bd9a72` and still are. Reformatting them would have buried a 15-line diff
   in a whole-file rewrite. Fourteen other files in `backend/internal/` are in the same state; it
   is a repo-wide condition, not a card artifact.
4. **Databases created by this card, and the only ones it may drop: `hq_test_go_h1` and
   `hq_test_e2e_h1`.** Per B-16(a), nothing else on this host was dropped. The Playwright run used
   `TEST_PORT=8210` / `TEST_OUTPUT_DIR=test-results-h1`, and deliberately did **not** invoke `task
   test` directly — `task test`'s recipe `DROP DATABASE hq_test_e2e`, which a concurrently
   dispatched card may be mid-run against. The dependency chain it fixes was verified by
   `task --dry test` instead, and the suite invoked with the same env under card-local names.
5. **No version moved.** `version.go`'s `Frontend` and `package.json`'s `"version"` are both
   `1.2.2`, unchanged from `9bd9a72`. `node build-sw.js` is idempotent on this tree (clean after).
6. **Nothing was PARKED.** The card's PARK trigger — the 20th spec file, or any Go package, going
   red once the harness stops lying — did **not** fire. The newly-enabled 20th file
   (`user-invite-onboarding.feature.spec.js`) ran as test `[569/569]` and **passed**, and the full
   Go suite is green against a real database with real per-package timings (no sub-second
   DB-backed package). ~~Both `sync.spec.js` reds the launch prompt armed as expected
   (`:446` [LST-17] and `:1198`) also passed on this run.~~ **STRUCK — see "Repair round" (N1):
   `:1198` is a DEAD line anchor, not a test. `tests/sync.spec.js:446` [LST-17] is live, correct,
   and did pass; `:1198` names nothing, so "it passed" was never a claim this note could make.**

## Repair round (post-G6, `overnight-20260729-2`)

G6 returned **APPROVE-WITH-NITS, no blocking findings**: the fix is correct and verified. This
round changed **`scripts/verify-test-harness.sh` and this note only**. The fix itself —
`Taskfile.yml`, `backend/internal/testdb/testdb.go`, the five `TestMain`s and the three helpers —
is **byte-identical to `7fa97b7`** (`git diff 7fa97b7 -- <those ten paths>` is empty). Two holes in
the card's own *verification script* were closed, and **each repair was itself falsified before it
shipped** — a repair to a falsifiability hole that was not falsified is not a repair.

**N3 — Check B2's package list omitted the five `TestMain` packages, which is precisely where the
unset path is fragile.** B2 ran only `./internal/recipes/ ./internal/sync/`: the two **helper**-based
packages, whose unset path this card left structurally unchanged (they `t.Skip` on unset *before*
they ever touch the DSN, so they cannot regress). The five `TestMain` packages are where the new and
delicate `requested := dbURL != ""` **must be computed before the fallback** — and B2 never looked at
them. G6 proved the hole by mutation: move that one line *after* the fallback in
`receipt/worker_test.go` and the package genuinely FAILS on unset while the old B2 still printed
`PASS  go test exited 0 with DB_TEST_URL unset (skip-on-unset preserved)` and the script exited 0 —
the exact over-correction B2 exists to catch, reported green. **Fix:** B2 now runs the same
`$DB_PKGS` list as Check B — all eight converted sites. **Falsified:** the mutation was reproduced in
a scratch copy of the worktree; the old two-package command exits **0**, the repaired seven-package
command exits **1** with `FAIL github.com/yumyums/hq/internal/receipt`, and the full script prints
`FAIL … skip-on-unset was over-corrected into a failure` and exits **1**.

**N2 — Check A asserted the mechanism, not the property.** It grepped `task --dry test` for
`npx bddgen` and then printed `Total: N tests in M files` as an **ungraded** corroborating line. A
tree where `bddgen` ran but emitted zero features — moved features dir, renamed glob, generator that
exits 0 on empty input — passed Check A. The card's property is *"the suite runs every spec file the
repo has."* **Fix:** new graded **Check A2** runs `npx bddgen` (idempotent, ~2s, and what the
repaired dep chain now does) then asserts the resolved spec-file count is **≥ 20**. A missing
`node_modules` is a FAIL, not a silent skip — an ungraded check is not a passed check. The floor
ratchets up as spec files are added and must never ratchet down silently.
`H1_MIN_SPEC_FILES` overrides it, for red-proving and nothing else. **Falsified:** with
`H1_MIN_SPEC_FILES=21` the check prints
`FAIL  Total: 569 tests in 20 files — BELOW the floor of 21` and the script exits **1**.

**Post-repair state:** `bash scripts/verify-test-harness.sh` → A PASS, A2 PASS
(`Total: 569 tests in 20 files`), B PASS (exit 1 on the dead DSN), B2 PASS (exit 0 unset across all
seven packages), raw exit **0**. Skip-on-unset confirmed intact:
`env -u DB_TEST_URL -u DATABASE_URL -u TEST_DATABASE_URL go test -count=1 -p 1 ./...` over all nine
packages exits **0**. No database was created this round;
`hq_test_go_h1` / `hq_test_e2e_h1` were reused. Per B-16(a), nothing was dropped.

### Recorded, NOT fixed — for triage

- **N4 — a broken DB now yields *zero* signal from the five `TestMain` packages, not partial
  signal.** With `DB_TEST_URL` set-but-unreachable those packages exit before `m.Run()`, so even the
  **hermetic** tests that previously passed do not execute. This is the intended semantics ("setting
  the variable is a statement of intent") and is documented in `testdb.go`, but the cost is real and
  is not what the card's framing advertises: the tradeoff is *loud failure* against *partial
  hermetic coverage*, and this card chose loud. If triage wants both, the shape would be a gate that
  fails the DB-backed tests individually rather than the binary — a different card. **Not changed
  here.**
- **N1 — `tests/sync.spec.js:1198` is a DEAD line anchor.** Line 1198 is inside a helper's loop body
  (`await p.waitForTimeout(400);`), not a `test(` declaration. The test it used to name is now at
  `:1372` (`test('temperature answer converges (live + catch-up)')`). Dead since the 2026-07-24
  `syncspec-deflake` work, with an unactioned migration item. `:446`
  (`test('list page progress decrements when another device unchecks a field [LST-17]')`) is **live
  and correct**. The stale `:1198` is carried by the slate's preconditions table **and** by this
  note — the note's copy is struck above; **the slate is not this card's file and was not
  edited.** **Not changed here.**

## What must survive any merge

1. **`bdd:gen` is in `task test`'s `deps`.** The property is *`task test` runs every spec file the
   repo has, including the generated ones*; the mechanism is the dep. `bdd:` (`:78`) and the CI
   task (`:102`) already carry it — `test:` was the sole omission. Anything that removes it
   restores a suite that reports success while a whole Playwright project contributes zero tests.
2. **The asymmetry: `DB_TEST_URL` UNSET ⇒ skip; SET but unreachable ⇒ FAIL.** Both halves are
   load-bearing and they are not the same statement. A contributor with no database must still be
   able to run the unit tests — that is why the *unset* skip stays. The bug is the **symmetry**,
   not the skip. A merge that converts the unset case too has broken the card in the other
   direction.
3. **The failure message names the DSN and the stage (connect vs ping).** `pgxpool.New` is lazy,
   so a missing *database* (as opposed to a missing *server*) surfaces only at `Ping`. Collapsing
   the two into one message throws away the one bit that distinguishes "Postgres is down" from
   "someone dropped my database out from under me" — which is exactly the incident (B-16) that
   produced this card.
4. **`TestMain` exits with a non-zero status, not a panic.** `TestMain` has no `*testing.T`. The
   gate prints to stderr and `os.Exit(1)`s so the reason is legible; a panic would bury it under a
   goroutine dump.
5. **Every fallback DSN and every `db.Migrate` / `db.SeedHQApps` call in the five `TestMain`s
   stays exactly as it was.** This card changes only what happens on the two error branches.
6. **The eight sites share one message.** If a merge splits them, they must still say the same
   thing for the same reason.

## What is safe to drop

- **All comment wording**, in `Taskfile.yml`, in `testdb.go`, and at the eight call sites. The
  behaviour matters; the prose does not.
- **`scripts/verify-test-harness.sh`** — an assertion harness, not the fix. Its checks are
  reproducible by hand from this note.
- **The roadmap card's prose.** The **status flip** matters; the wording does not.
- **The exact spelling of the `testdb` package's API** (`Reason`, `ExitIfRequested`). Any shape
  that preserves properties 2–4 above is equivalent.
- **Anything in this note itself.**

## Not done, deliberately

- **No test that the harness newly reveals as failing gets fixed.** The card's PARK trigger is
  exactly this: if enabling `bdd:gen` turns the 20th spec file red, or a Go package turns red once
  the harness stops lying, that is a **pre-existing defect this card exposed** and is parked with
  evidence, not fixed here.
- **The per-test `t.Skip` lines downstream of the five `TestMain`s are NOT individually
  converted** — see the footprint section. Fixing the `TestMain`s subsumes them.
- **`backend/internal/sync/jwtbridge_rls_test.go` is NOT touched.** Its `connectSpike` skips on
  `SPIKE_DB_URL`, not `DB_TEST_URL`, and it is governed by the *separate*, already-correct
  `HQ_SYNC_SPIKE_LIVE` asymmetric gate in `proxy_live_test.go` — the very pattern this card
  copies. It is not a second instance of the bug.
- **`backend/internal/sync/proxy_live_test.go` is NOT touched.** It is the reference. Copying from
  it does not mean editing it.
- **B-06 (`tests/sync.spec.js:1584` stale comment), B-17, B-18, B-19, B-20 are NOT folded in.**
  They belong to other cards.
- **No `openspec/` directory and no OpenSpec mechanics.** `night-crew workflow preflight` reports
  openspec ABSENT for this repo. Universal per-change discipline only (red-first evidence, atomic
  commits, the `Night-Crew-Card:` trailer as one adjacent paragraph, the roadmap flip).
- **No version bump.** This card changes a task-runner dep and test-only Go. It ships no frontend
  asset and no backend behaviour, so neither `version.go` nor `package.json` moves.

## Four-HARD-constraints attestation

1. **Root `package.json` AND `package-lock.json`** — **UNTOUCHED, both.** No devDependency, no
   script, no version move. `npx bddgen` and `npx playwright test` are invoked as they already
   exist. This is the Playwright environment shared by every card and every worktree tonight.
2. **`backend/go.mod`** — **UNTOUCHED.** No dependency added, removed, or version-changed. The new
   `internal/testdb` package imports only `fmt`, `os` and `testing` from the standard library.
3. **`docker-compose.nc.yml`** — **UNTOUCHED.** This card runs against its own
   `hq_test_go_h1` / `hq_test_e2e_h1` databases on the existing `yumyums-dev-pg` (`:5433`) and its
   own `TEST_PORT`; no compose service is added, renamed, or re-ported.
4. **Root `Taskfile.yml`** — **TOUCHED, and this is the one card on the slate for which that is
   the point.** One line: `bdd:gen` joins `backend:db-test` and `sw` in `test:`'s `deps`
   (`:28-30`), plus its explanatory comment. Nothing else in the file is edited. Every other card
   tonight declares this file untouched, so a three-way merge should see no conflict at all; if it
   does, keep the `bdd:gen` dep and take the rest from the other side.
