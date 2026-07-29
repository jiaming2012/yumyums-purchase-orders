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
  pasted output. Check A: `task --dry test` must list `npx bddgen`. Check B: with `DB_TEST_URL`
  pointed at a dead port, the Go suite must exit non-zero. New file, no conflict surface. Safe to
  drop *after* the merge lands — it asserts, it does not implement.
- `.night-crew/knowledge/roadmap.md` — this card's `PLANNED` → `DONE` status flip at ~`:528`, in
  the same change set as the work. Every card tonight edits its own card in this file; conflicts
  are per-card and **both sides should be kept**.
- `.night-crew/runs/2026-07-29-2-autonomous/merge-intent-h1-test-harness-fail-loud.md` — this
  note. New file, unique to this card. No conflict surface.

### Late additions

_(appended only if implementation forces a file outside the list above; "nothing here" if it stays
clean. Per B-11, if a later round contradicts anything above, the contradicted line is STRUCK in
place — not merely appended to.)_

**Pending — closed out after the gates.**

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
