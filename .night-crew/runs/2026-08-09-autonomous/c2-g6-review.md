# G6 adversarial review — C2 `spike-exit-code-honesty` (B-163, run 20260809)

Fresh-context adversarial reviewer. Inputs limited to the slate entry, the diff, and independently
reproduced evidence. It re-ran all three red-first transitions itself (BEFORE and AFTER), and hunted
specifically for over-correction (a genuine red masked as could-not-run).

## Verdict: PASS  ·  no fix round  ·  safe to merge

Every criterion holds under independent reproduction. The fix does the three conflation-fixes exactly
and does NOT over-correct: every genuine RED (exit 1) still exits 1, and green still exits 0.

## Per-criterion

| Criterion | Result | Evidence (independently observed) |
|---|---|---|
| (a) infra→2 | PASS | base exit 1; guarded exit 2; prints COULD NOT RUN, not a red verdict; the past-assignment sentinel never printed in BEFORE, proving `set -e` killed at the assignment |
| (b) vacuous→2 | PASS | base `red` exit 1; fixed `cannot_run` exit 2; branch genuinely vacuous (client GREEN + B_ROWS≠1 = UPDATE-in-place never exercised), not a masked failure |
| (c) uncaught→2 (both shapes) | PASS | sync throw: 1→2; rejected top-level await: 1→2; both print COULD NOT RUN; both surface as `uncaughtException` in Node 22 |
| No over-correction (real red still 1, green still 0) | PASS | `die(RED)` → 1; `die(RED)` after async `shutdown()` → 1; bare `process.exit(1)` → 1; `process.exit(0)` → 0 — handlers do NOT swallow/convert intentional exits |
| Completeness (all unguarded substitutions found) | PASS | 7 guarded, exactly right; printf-arg + `mapfile < <()` proven not to die under `set -e`, correctly left unguarded |
| Taskfile note inert | PASS | comment-only (every added line `#`); no cmd:/desc:/deps: change; all 4 targets resolve; `demo:sync` intact |
| Footprint clean (precache 31, BACKLOG scoped) | PASS | 5 files, all in-footprint; sw.js byte-identical to base (no served asset touched → build-sw a no-op → 31 by construction); BACKLOG = 1 line, B-163 only |

## Over-correction finding (the probe that mattered most): NONE
- **JS:** `die(RED)` exits 1 with both handlers installed — confirmed three ways. `process.exit()` is synchronous and does not route through `uncaughtException`/`unhandledRejection`, so the honest RED is never converted to 2. Exactly one `die(RED)` call site (line 618), inside `if (!recovered)` after a fully try/catch-wrapped `shutdown()` that cannot throw. The one fire-and-forget promise (`void applyRow`, red path only) internally try/catches its await → can never trigger `unhandledRejection`.
- **Shell:** each `|| cannot_run` fires ONLY when `srcpsql` exits nonzero (docker/psql itself failed = genuinely could-not-run). A real answer of `0`/`2`/empty passes through with exit 0 to the verdict logic — verified in a scratch harness. `B_ROWS=2` does NOT short-circuit; it reaches the vacuous-green eval. No completed-but-wrong round-trip answer is masked as 2.

## Completeness finding
Agrees with SEVEN. 8 bare `VAR="$(srcpsql…)"` assignments exist; `APPLIED` (line 410) carries its own `|| echo 0` and deliberately does not die (value optional), leaving 7 that die under `set -e` — all 7 guarded (`TABLES`, `HQ_USER_ID`, `SUBMIT_TRG`, `RESP_TRG`, `B_ROWS`, `A_ROWS`, `C_ROWS`). printf-arg substitutions and `mapfile < <(srcpsql…)` provably do NOT die under `set -euo pipefail` (verified) — leaving them unguarded is correct; mapfile results separately guarded by `[ ${#FIELDS[@]} -ge 4 ]`.

## Contract preservation
green=0, real RED (CLIENT_RC=1)=1, client-cannot-run=2, out-of-contract=2, substrate-restore-failure=3 (untouched). Invocation-count delta corroborates: `red "` 2→1, `cannot_run "` 41→49 (−1 red +8 cannot_run = 7 guards + 1 vacuous conversion). `node --check` passes.

## Minor note (not blocking)
G6 could not run `build-sw.js` in the worktree (no `workbox-build` in worktree node_modules), so precache-31 was proven by CONSTRUCTION (no served/precached asset in the diff; sw.js byte-identical to base) rather than by execution. The orchestrator verifies by execution on the main checkout post-merge.
