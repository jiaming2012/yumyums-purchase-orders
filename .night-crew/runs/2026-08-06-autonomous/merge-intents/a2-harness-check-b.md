# Merge intent — A2 · `gate-harness-check-b-per-package`

Run `20260806` · branch `card/a2-harness-check-b` · based at `ef314e0`
Closes Q-KR2, closes B-22.

## What this card changes

`scripts/verify-test-harness.sh` Check B ran ONE aggregate
`go test -count=1 -p 1 $DB_PKGS` across all seven DB-backed packages and passed on
`DEAD_STATUS -ne 0`. `go test` exits non-zero if **any** package fails, so the check is an
OR: six of seven packages can lose fail-loud and Check B still prints PASS. This card makes
it a per-package loop requiring **all seven** to exit non-zero, and asserts the package
count it iterated so a shrinking `DB_PKGS` announces itself instead of quietly narrowing
the check.

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
the old aggregate Check B prints PASS anyway. Those edits exist only inside the red
capture and are `git checkout`-reverted before the implementing commit. **No committed diff
on this branch touches `backend/`** — verify with `git diff ef314e0..HEAD --stat`.

## What MUST survive any merge

1. **Check B is a per-package loop, not an aggregate `go test`.** The whole card is that
   the OR is a defect. If a merge resolution restores a single `go test $DB_PKGS` invocation
   in Check B, the card is undone and B-22 is reopened. The loop variable is `p`; the pass
   condition is that **every** package exited non-zero.
2. **The package-count assertion.** `H1_DB_PKG_COUNT` (default 7) is compared against the
   number of entries actually iterated. Its purpose is that a future edit trimming `DB_PKGS`
   fails loudly rather than silently narrowing the check — the same class of defect as the
   two-package B2 that G6 caught during H1. Do not drop it, and do not "fix" a red by
   lowering the constant without a written reason.
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
  new PASS/FAIL lines. Take either side.
- The per-package log filenames (`/tmp/h1-deadport-<pkg>.log`) and whether the loop keeps
  one log per package or concatenates. Only the exit-status arithmetic is load-bearing.
- The name `H1_DB_PKG_COUNT` — if another card introduces a conflicting convention for
  override variables, rename it. The assertion must remain.

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
