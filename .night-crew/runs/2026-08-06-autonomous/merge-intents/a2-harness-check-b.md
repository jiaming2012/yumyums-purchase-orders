# Merge intent — A2 · `gate-harness-check-b-per-package`

Run `20260806` · branch `card/a2-harness-check-b` · based at `ef314e0`
Closes Q-KR2 (the KR on red-first mutation evidence), closes B-22 (Check B graded the
disjunction of seven packages, not their conjunction). Also closes, in the G6 fix round, a
B-23-class hand-typed constant inside the same check.

## What this card changes

`scripts/verify-test-harness.sh` Check B ran ONE aggregate
`go test -count=1 -p 1 $DB_PKGS` across all seven DB-backed packages and passed on
`DEAD_STATUS -ne 0`. `go test` exits non-zero if **any** package fails, so the check is an
OR: six of seven packages can lose fail-loud and Check B still prints PASS. This card makes
it a per-package loop requiring **all seven** to exit non-zero, and asserts the package
count it iterated so a shrinking `DB_PKGS` announces itself instead of quietly narrowing
the check.

## Red-first

Q-KR3 wants this as a **section**, not a commit trailer. The substance existed only in commit
`73547a0`'s message and a source comment; it is carried here so the note itself is gradeable.
This is the KR's first gradeable cycle. Everything below is measured on this host; where the
card recorded nothing, that is said rather than reconstructed.

**The named check:** `Check B of scripts/verify-test-harness.sh`. The harness script *is* the
test here — this card's deliverable is a check, so its red-first evidence is that check going
red on a tree where the property it defends is violated, and green on a tree where it is not.
There is no Playwright spec and no Go test; asserting one would be a false citation.

### The tree the red was captured against

🛑 **The card recorded no SHA.** The implementer's red capture is described in `73547a0`'s
message and in the note above ("reverted before any commit") but never pinned to a commit.
What *can* be established, and is, by object hash rather than by assertion:

| SHA | `scripts/verify-test-harness.sh` blob | `backend/` tree |
|---|---|---|
| `ef314e0` — run-branch base | `174c0b9e…` | `94d1d6bf…` |
| `3e5e4d7` — merge-intent commit, written before implementing | `174c0b9e…` — same | `94d1d6bf…` — same |
| `73547a0` — the implementing commit | `29ec1496…` — the fix | `94d1d6bf…` — same |

`3e5e4d7` touches one markdown file and nothing else, so for **both** the file under test and
the entire `backend/` tree it is byte-identical to `ef314e0`. The red was therefore captured
against a working tree content-identical to `ef314e0` **plus** the six-package mutation, and
the honest statement is: **`ef314e0` for the code, `3e5e4d7` for the checkout** — the two are
indistinguishable here. Which of the two the implementer actually had checked out is **not
recoverable** from the artifacts and is not claimed. The `backend/` tree hash being identical
across all three SHAs is also the independent proof that the mutations were reverted: no
committed diff on this branch touches Go source.

### The red

On that tree, with fail-loud removed from six of the seven DB-coupled packages (only
`internal/workflow` left honest), the **old aggregate** Check B printed, verbatim:

    PASS  go test exited 1 with DB_TEST_URL=postgres://…/hq_test_go_dropped_by_a_reviewer

while a package-by-package probe of the same tree read
`workflow exit 1 · receipt/inventory/auth/purchasing/recipes/sync exit 0`. A check written to
catch a harness reporting success while measuring nothing was reporting success while measuring
one seventh of what it claimed. That is B-22.

Additionally, in this fix round, the **F4 red** — a tree carrying a DB-coupled package absent
from `$DB_PKGS`. Captured against the fix-round working tree (`73547a0` + the F1–F7 edits) with
a throwaway `backend/internal/f4probe/f4probe_test.go` importing `internal/testdb`:

    FAIL  $DB_PKGS does not match the DB-coupled packages this tree actually has.
          Iterated 7 · expected 8 · derived from the tree 8
          DB-coupled but MISSING from $DB_PKGS: internal/f4probe
          In $DB_PKGS but NOT DB-coupled:       none

`SCRIPT_EXIT=1`, `real 10m37.472s`. Diagnostics: `/tmp/h1-harness-77048-9ptvsj/`.

The pre-fix branch on that same tree is `[ "$DEAD_PROBED" -eq "$EXPECTED_DB_PKGS" ]` with
`DEAD_PROBED=7` (the loop iterates `$DB_PKGS`, which does not list `f4probe`) and
`EXPECTED_DB_PKGS=7` (the hand-typed constant), so it printed PASS. That is stated **from
inspection of the arithmetic, not from a measured run** — it is deterministic and was not
separately executed, and this note says so rather than implying a probe that was not taken.

The throwaway package was deleted; `backend/internal/` is back to its 17 committed directories
and the derivation is back to 7.

### The green after

The card recorded **no end-to-end harness run at all**. One was run by the reviewer
(`harness OK`, `SCRIPT_EXIT=0`, `real 10m39.450s`) against `73547a0`. This fix round modifies
the script, so that run is superseded.

Two green runs were taken here. The first (`real 10m37.104s`, diagnostics
`/tmp/h1-harness-95200-prXhh3/`) ran a working tree that differed from the final commits by
the F5 **comment block only** — the cost figures were refined after it finished. Rather than
quote a run of a tree that was not shipped, it was re-run. **The figures below are the second
run, against the tree at `b829701` byte-for-byte** (`sha256 b1279ee9…` on
`scripts/verify-test-harness.sh`, identical to the file the commits reconstruct).

Five graded lines, verbatim:

    ── A · task test regenerates the BDD specs ─────────────────────────────
      PASS  task test's dependency chain runs 'npx bddgen'

    ── A2 · Playwright resolves at least 20 spec files ────────────────────
      PASS  Total: 797 tests in 29 files (floor: 20)

    ── B · DB_TEST_URL set + unreachable ⇒ non-zero exit, EVERY package ────
        ./internal/workflow/       exit 1   fails loud
        ./internal/receipt/        exit 1   fails loud
        ./internal/inventory/      exit 1   fails loud
        ./internal/auth/           exit 1   fails loud
        ./internal/purchasing/     exit 1   fails loud
        ./internal/recipes/        exit 1   fails loud
        ./internal/sync/           exit 1   fails loud
      PASS  all 7 DB-backed packages exited non-zero, individually, with
            DB_TEST_URL=postgres://…@127.0.0.1:5433/hq_test_go_dropped_by_a_reviewer
      PASS  Check B iterated 7 packages, and $DB_PKGS is exactly the set of
            packages whose tests import internal/testdb (derived from the tree, not
            typed: 7)

    ── B2 · DB_TEST_URL unset ⇒ still skips (no over-correction) ───────────
      PASS  go test exited 0 with DB_TEST_URL unset (skip-on-unset preserved)

    harness OK — the suite is capable of reporting failure.
    real 10m41.742s   SCRIPT_EXIT=0

Run under the shared suite mutex, backgrounded — a foreground shell call SIGTERMs at 10min and
this check takes ~10m40s, so a foreground run of it looks like a failure and is not. It also
sat queued on the mutex behind another card's Go legs for several minutes before starting;
that is the lock working, not a hang.

Diagnostics: `/tmp/h1-harness-43553-Xre9LO/`. The three harness runs of this round wrote to
`…-77048-9ptvsj` (F4 red), `…-95200-prXhh3` (first green) and `…-43553-Xre9LO` (this one) —
three distinct directories, which is F7's fix demonstrating itself.

Check B's own per-package cost in this run: workflow 0.049 · receipt 0.037 · inventory 0.022 ·
auth 0.030 · purchasing 0.023 · recipes 0.681 · **sync 9.662** — consistent with the 15–20s,
sync-dominated figure the corrected F5 comment now states, and nothing like the 0.2s it used
to claim.

Note the script's comment cites **three** `internal/sync` measurements (8.388 / 8.464 /
14.177s) and this note cites **four**, adding 9.662s. The fourth is *this* run — it did not
exist when the comment was written, and editing the comment to add it would change the tree
this green was taken against, which is the regress that produced the first green's caveat
above. 9.662s falls inside the range the comment already states, so the comment is not stale;
it is simply one datum short and says nothing false. A future editor should fold it in.

## The seven individual mutation probes (Q-KR2 evidence)

`slate-20260806.md:67` names this card's evidence shape as **seven individual mutation
probes**. The implementer mutated six packages *at once* and snapshotted per-package exits.
That proves the disjunction is real, but it does not prove, per package, that mutating package
*k* **alone** reds the new check — the property Q-KR2 is actually asking about. Rows 4 and 6
were done independently by the G6 reviewer; rows 1, 2, 3, 5 and 7 were done in this fix round,
each package mutated **alone** with `git checkout -- backend/` between rows.

The old-check column is the pre-fix assertion re-run verbatim on the mutated tree:
one aggregate `go test -count=1 -p 1 $DB_PKGS` under the dead DSN, graded `-ne 0`.

| # | Package | Mutation (applied alone) | OLD aggregate check | NEW per-package loop | Names it? |
|---|---|---|---|---|---|
| 1 | `internal/workflow` | `stable_identity_test.go:25` `requested := dbURL != ""` → `requested := false` | **PASS** — aggregate exit 1 | **FAIL** | `./internal/workflow/` |
| 2 | `internal/receipt` | `worker_test.go:25` same `requested :=` mutation | **PASS** — aggregate exit 1 | **FAIL** | `./internal/receipt/` |
| 3 | `internal/inventory` | `period_summary_test.go:28` same `requested :=` mutation | **PASS** — aggregate exit 1 | **FAIL** | `./internal/inventory/` |
| 4 | `internal/auth` | `permission_test.go:40` same `requested :=` mutation | **PASS** | **FAIL** | `./internal/auth/` |
| 5 | `internal/purchasing` | `scheduler_cron_test.go:39` same `requested :=` mutation | **PASS** — aggregate exit 1 | **FAIL** | `./internal/purchasing/` |
| 6 | `internal/recipes` | `helpers_test.go` `t.Fatal(testdb.Reason(…))` → `t.Skip(…)` | **PASS** | **FAIL** | `./internal/recipes/` |
| 7 | `internal/sync` | `access_test.go` + `jwtbridge_test.go` `t.Fatal(testdb.Reason(…))` → `t.Skip(…)` | **PASS** — aggregate exit 1 | **FAIL** | `./internal/sync/` |

Rows 1, 2, 3, 5, 7 run in this fix round; rows 4 and 6 by the G6 reviewer. **7 of 7 packages:
the old check passes on a tree where that package alone has lost fail-loud; the new loop fails
and names the package.** No committed diff on this branch touches `backend/` — the `backend/`
tree hash is identical at `ef314e0`, `3e5e4d7`, `73547a0` and HEAD.

## Shared files touched

| File | Why |
|---|---|
| `scripts/verify-test-harness.sh` | The card's subject. Owned solely by A2 this run — no other card in slate-20260806 declares it. |
| `.night-crew/runs/2026-08-06-autonomous/merge-intents/a2-harness-check-b.md` | This file. New path, card-unique name; cannot collide. |

Nothing else. No backend Go source, no frontend, no test specs, no `Taskfile.yml`, no
`night-crew.toml`, no `package.json`, no `sw.js`.

**Temporary mutations, reverted before any commit:** the Red-first demonstration mutates
the fail-loud gate in six of the seven DB packages (`internal/receipt`,
`internal/inventory`, `internal/auth`, `internal/purchasing` — the `requested :=` line in
each `TestMain`; `internal/recipes`, `internal/sync` — the helper `t.Fatal` sites) to prove
the old aggregate Check B prints PASS anyway. The fix round adds two more classes of
temporary mutation: the five **individual** probes (one package at a time, `git checkout --
backend/` between rows) and a throwaway `backend/internal/f4probe/f4probe_test.go` for the
F4 proof. All of them exist only inside a red capture and are reverted or deleted before any
commit. **No committed diff on this branch touches `backend/`** — verify with
`git diff ef314e0..HEAD --stat`, or more strongly with
`git rev-parse ef314e0:backend HEAD:backend`, which return the same tree object.

## What MUST survive any merge

1. **Check B is a per-package loop, not an aggregate `go test`.** The whole card is that
   the OR is a defect. If a merge resolution restores a single `go test $DB_PKGS` invocation
   in Check B, the card is undone and B-22 is reopened. The loop variable is `p`; the pass
   condition is that **every** package exited non-zero.
2. **The `$DB_PKGS` assertion, and the fact that its expectation is DERIVED.** The check
   compares the iterated packages against the set of directories whose `*_test.go` import
   `internal/testdb`, computed from the tree at run time — **not** against a hand-typed `7`.
   Both the count and the set membership are graded, and the failure names the offending
   package. Its purpose is two-sided: an edit trimming `$DB_PKGS` fails loudly rather than
   silently narrowing the check (the two-package B2 that G6 caught during H1), *and* a newly
   added DB-coupled package that nobody listed fails loudly rather than sitting ungraded.
   `H1_DB_PKG_COUNT` remains, but only as the prove-it-can-go-red override on the count; it
   does not suppress the set comparison. A merge that restores a hand-typed constant reopens
   B-23 and undoes the F4 fix. Do not "fix" a red by setting the override.
3. **`DB_PKGS` still lists all seven packages** and is still shared verbatim between Check B
   and Check B2. B2 deliberately runs the same list (see its comment block); a merge that
   gives them separate lists breaks the symmetry guard.
4. **Check B2 is unchanged.** It is the guard against over-correcting and this card did not
   need to weaken it: B2's aggregate `-eq 0` is already an AND (aggregate exit 0 ⇒ every
   package exited 0), so making B honest required no change to what B2 asserts. Any merge
   that alters B2's assertion is out of this card's scope and should be questioned.
5. **`H1_DEAD_PORT` stays opt-in and defaulted OFF.** The live-server/missing-database DSN is
   0.02s per package; the dead-port variant is ~110s per package on this WSL2 host. A
   per-package loop multiplies that by seven. Flipping the default would turn a 0.2s check
   into a 13-minute one.

## What is safe to drop

- Wording, comment prose, box-drawing characters and the exact `printf` formatting of the
  new PASS/FAIL lines. Take either side — **except** the branched aggregate note in Check B's
  FAIL text, which is a correctness fix, not prose (see F3 below).
- The exact log filenames and the `mktemp -d` layout under `$H1_LOGDIR`. What must survive is
  that the paths are **per-run and unique**, not the naming scheme. A merge that restores a
  fixed `/tmp/h1-*.log` reopens F7.
- The name `H1_DB_PKG_COUNT` — if another card introduces a conflicting convention for
  override variables, rename it. The assertion must remain.

## Fix round (G6 returned MERGE WITH NOTE)

The mechanism was upheld — independently falsified at 1, 2, 6 and 7 packages in both
fail-loud families — and is unchanged. Six defects around it were corrected.

| # | Defect | Fix | Proof |
|---|---|---|---|
| F1 | No `## Red-first` section; substance lived only in `73547a0`'s message. Q-KR3 grades the section. | The section above, with all three elements and an explicit statement of what is not recoverable. | This file. |
| F2 | Q-KR2's evidence shape is *seven individual* probes; the card ran one six-package mutation. | Five packages mutated alone in this round (`workflow`, `receipt`, `inventory`, `purchasing`, `sync`); reviewer's two folded in. | The 7-row table above. |
| F3 | Check B's FAIL text claimed unconditionally that an aggregate `go test` "would still have exited non-zero here" — **false** when all seven are silent, a case that reaches that branch. B-17/B-24 shape: a false durable claim inside the check written to stop checks lying. | The sentence branches on `DEAD_LOUD` (packages that did fail). Partial case keeps the B-22 note; all-silent case says the aggregate would have gone red too, and locates the loop's value in the partial case and in *naming* the package. | Both branches are reachable and are written from the measurement, not asserted. |
| F4 | `EXPECTED_DB_PKGS="${H1_DB_PKG_COUNT:-7}"` was hand-typed and its comment claimed a ratchet no code implemented — a NEW DB-coupled package left out of `$DB_PKGS` kept both at 7 and PASSED. B-23 class. | Derived from the tree: directories whose `*_test.go` import `internal/testdb`. Count **and** set membership graded; the FAIL names the offending package. | Throwaway `internal/f4probe` ⇒ `FAIL … Iterated 7 · expected 8 · derived 8 · MISSING: internal/f4probe`, `SCRIPT_EXIT=1`. Removed; derivation back to 7 and green. |
| F5 | "~0.02s of Postgres round-trip per package" understated Check B by ~700×. | Corrected to the measured per-package figures with the sync-dominance called out, and the reusable *relative* claim kept. | Measured this round from the script's own logs: workflow 0.019 · receipt 0.020 · inventory 0.018 · auth 0.018 · purchasing 0.020 · recipes 0.629 · **sync 8.464** ⇒ 18.7s wall end to end. Across four runs `internal/sync` read 8.388 / 8.464 / 9.662 / 14.177s, so the doc states 15–20s and sync-dominated rather than a constant. |
| F7 | Fixed `/tmp` log paths, now seven wide; `basename` keys collide on shared leaf names; the stale pre-card `/tmp/h1-deadport.log` is never rewritten and reads as current. | All diagnostics move to a per-run `mktemp -d "/tmp/h1-harness-$$-XXXXXX"` (`$H1_LOGDIR`, overridable), printed as the script's first line. Per-package logs keyed by full-path slug, not `basename`. | The two harness runs this round wrote to distinct directories; the red run's path is quoted in its own output. |

### Recorded for triage, deliberately NOT fixed here

- **F6 — the per-package PASS line overclaims.** It prints `fails loud` on evidence that only
  says *exited non-zero*. A **deleted** package, or one that does not compile, also exits
  non-zero and so satisfies Check B (both measured exit 1). Check B2 catches those — it runs
  the same list with `DB_TEST_URL` unset and expects exit 0 — so the ladder as a whole is not
  blind, but B2 reports the wrong diagnosis: "skip-on-unset was over-corrected" for a package
  that is simply gone. **Pre-existing, not a regression** — old Check B's aggregate had the
  same hole. The wording is left as-is rather than half-corrected; the accurate fix is for
  Check B to distinguish *failed the DB gate* from *failed to build*, e.g. by grading
  `go vet` or a build-only pass separately. New card.
- **F8 — Check B2's failure is undiagnostic.** It names no package, exactly as old Check B
  did not, and for the same reason: one aggregate `go test $DB_PKGS`. It deserves the same
  per-package loop this card gave Check B, which would cost it nothing new — the packages are
  already compiled by then. Not done here because it is a second mechanism change in the same
  file and would put the card's scope back where §1.4 split it. New card.

## Card A4 (`gate-ladder-completeness`), which runs after me in Track B

**Nothing here.** I do **not** touch `.night-crew/knowledge/reference/gate-ladder.md`. A4
owns that file outright and has no preservation obligation toward this card. I read the
ladder; I did not write to it.

If A4 wants to cite this card, the accurate statement is: *Check B of
`scripts/verify-test-harness.sh` grades all seven DB-backed packages individually as of run
20260806; before that it graded their disjunction.*

## Other cards this run

- **C1**, **Track A**: no declared overlap with `scripts/verify-test-harness.sh`.
- The global Playwright/Go suite lock is shared infrastructure, not a file — no merge
  implication.
