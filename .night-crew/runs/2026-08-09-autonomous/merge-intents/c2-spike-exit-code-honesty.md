# Merge intent — C2 · `spike-exit-code-honesty` (B-163)

Run `20260809` · branch `card/c2-spike-exit-code-honesty` · off `overnight-20260809` @ 0fade6b
(which already includes card 1's `task demo:sync`).

---

## Scope

Fix the three latent exit-code conflations in the Spike-E reconnect harness so an INFRA failure
can never read as "ran and failed" (a false RED that would falsely instruct Activity 3 to add a
resync step). The recorded Spike-E verdict does NOT depend on any of these (B-163) — the fixes are
pure exit-code honesty, and the exit-code probes ARE the regression.

- **(a)** Guard every unguarded `$(srcpsql …)` command substitution in `spike-e-reconnect.sh` that
  can die under `set -euo pipefail` with psql's own (non-2) exit code → `|| cannot_run …` so a
  missing psql / bad coordinate exits **2**, never 1.
- **(b)** The `CLIENT_RC=0` + `B_ROWS≠1` branch that called `red` (exit 1) → now `cannot_run`
  (exit **2**): a GREEN that never exercised the mandatory UPDATE-in-place is VACUOUS, "no verdict",
  never RED.
- **(c)** `rxdb/spike-e-reconnect.js` — the client leg's main flow runs at module top level with the
  guarded setup block covering ONLY lines ~244-291; steps 2-11 are UNPROTECTED. An uncaught
  exception / unhandled rejection there escapes → Node's default exit **1** (false RED). Install a
  top-level handler mapping any uncaught exception/unhandled rejection to `die(SETUP)` → exit **2**.
- **(d)** Taskfile: add a WRAPPER NOTE on `spike:reconnect:red` documenting the 201 trap — gate on
  the SCRIPT, not `task spike:reconnect:red` (go-task returns its own 201 on a failing wrapped
  command, so 1/2 are indistinguishable at the `task` boundary). Target behaviour is UNCHANGED.

Plus the B-163 status flip in `.night-crew/knowledge/BACKLOG.md`.

## The harness's exit-code vocabulary (used verbatim by the fixes)

`spike-e-reconnect.sh` already defines the two failure verbs the card asks for
(`spike-e-reconnect.sh:154-157`):

- `cannot_run() { … exit 2; }` — infrastructure/setup. NOT a verdict.
- `red()        { … exit 1; }` — ran, catch-up disproven. A successful spike.

Both fixes (a) and (b) use `cannot_run` — no new helper is needed. The JS side already has
`die(SETUP=2, …)` / `die(RED=1, …)` (`spike-e-reconnect.js:81-90`); fix (c) reuses `die(SETUP, …)`.

---

## Shared files touched

| File | Why | What must survive a merge | Safe to drop |
|---|---|---|---|
| `Taskfile.yml` | I ADD a wrapper NOTE to the `spike:reconnect` / `spike:reconnect:red` comment block (the 201 trap: gate on the script, not `task`). This is a **disjoint stanza** from card 1's already-merged `demo:sync` block (further down the file, ~517+). NOTHING card 1 wrote is modified — I add a comment above the `spike:reconnect` family and do not touch `demo:sync`, `demo:sync:red`, or the `demo:` namespace. The two blocks do not overlap. | The 201-trap note on `spike:reconnect:red` (gate on the script directly, `; echo "EXIT=$?"`). | Nothing — target `cmd:`/`desc:` behaviour is unchanged; the change is comment-only. |
| `.night-crew/knowledge/BACKLOG.md` | I flip B-163's status (line ~944) from `new` to resolved on run 20260809, branch `card/c2-spike-exit-code-honesty`. A minimal, scoped edit to B-163's entry ONLY. | B-163's resolution marker (run + branch reference). | Nothing else on that entry, and NO other backlog entry is touched. |

**B-168 known standing check failure:** NOT touched. Scope is strictly B-163 (per card instruction).

New files OWNED by this card (no merge risk — did not exist before):
`.night-crew/runs/2026-08-09-autonomous/merge-intents/c2-spike-exit-code-honesty.md` (this file).

Files OWNED by this card (footprint OWNS): `.night-crew/qa/spike-supabase/spike-e-reconnect.sh`,
`.night-crew/qa/spike-supabase/rxdb/spike-e-reconnect.js`.

## Files READ but NOT edited

- `demo-sync.sh`, `spike-c-roundtrip.sh`, `env-up.sh`, `sql/spike-c-relay-trigger.sql`,
  `rxdb/spike-env.js` — read to understand the harness's helpers and the reused wiring; unchanged.
- `night-crew.toml` — read to confirm the seam map (no seam key matches my changed files).

## Empty sections

- **Backend Go**: nothing here. No `.go` file changed (G1/G2-Go N/A-by-footprint).
- **New Playwright specs**: nothing here. No seam-mapped app source touched → G2(Playwright) N/A.
- **Migrations / schema**: nothing here.
- **Served / precached assets**: nothing here. Changed files are a shell script + a node script
  under `.night-crew/`, `Taskfile.yml`, and `BACKLOG.md` — none is precached. Precache stays **31**.
- **G4 discipline greps**: N/A-VACUOUS — neither `internal/journal` nor `internal/workorder`
  exists in this repo (B-14).

---

## Red-first

The test IS the exit code. For each of (a)/(b)/(c) the WRONG code (1) was reproduced against the
current code BEFORE the fix, then the RIGHT code (2) captured AFTER — each with the literal command
+ `echo "EXIT=$?"`. Per the slate PARK note, the infra-failure paths were forced against a
**deliberately-missing coordinate** (a docker container id that does not exist) rather than a full
`task spike:up` bring-up — that is the red-first, not a park. No full live spike was run (B-163: the
recorded verdict does not depend on these).

### (a) infra failure — unguarded `$(srcpsql …)` under `set -euo pipefail`

The probe mirrors the harness's own `srcpsql()` (`spike-e-reconnect.sh:360`) and the unguarded
assignment shape (`:411`) EXACTLY, pointed at a container that does not exist.

```
# BEFORE (current, unguarded) — infra failure reads as RED:
$ probe-a-before.sh
probe(a) BEFORE — current UNGUARDED assignment (spike-e-reconnect.sh:411 shape):
Error: No such container: no-such-container-84684
EXIT=1

# AFTER (guarded with || cannot_run) — infra failure is could-not-run:
$ probe-a-after.sh
probe(a) AFTER — FIXED assignment (guarded with || cannot_run):
Error: No such container: no-such-container-84734
🛑 COULD NOT RUN (not a verdict) — could not count public tables in HQ's scratch Postgres (srcpsql failed)
EXIT=2
```

Result: **1 → 2**. ✅

### (b) vacuous-green — `CLIENT_RC=0` + `B_ROWS≠1`

The probe is the harness's `case "$CLIENT_RC"` block (`spike-e-reconnect.sh:562-571`) verbatim,
driven with `CLIENT_RC=0` (client leg said GREEN) and `B_ROWS=2` (UPDATE was NOT in place → vacuous).

```
# BEFORE (current) — vacuous green wrongly reported as a RED verdict:
$ probe-b-before.sh
probe(b) BEFORE — CLIENT_RC=0, B_ROWS=2 (current code: calls red, exit 1):
🛑 VERDICT: RED — the client leg reported full recovery but field B has 2 draft rows … this green would be vacuous
EXIT=1

# AFTER (calls cannot_run) — vacuous green is could-not-run, no verdict:
$ probe-b-after.sh
probe(b) AFTER — CLIENT_RC=0, B_ROWS=2 (fixed: cannot_run, exit 2):
🛑 COULD NOT RUN (not a verdict) — … A GREEN that never exercised the UPDATE is VACUOUS — no verdict, could-not-run (exit 2), never RED
EXIT=2
```

Result: **1 → 2**. ✅

### (c) uncaught JS exception in the client leg's unprotected top-level region

Models `spike-e-reconnect.js`: a guarded setup `try/catch` (js:244-291) followed by top-level
`await` code (steps 2-11, js:293-609) with NO surrounding try/catch. Both variants forced — a
synchronous `TypeError` and a rejected top-level `await`.

```
# BEFORE (current, no top-level handler) — Node default exit 1 (false RED):
$ node probe-c-before.mjs            # sync throw in unprotected region
… TypeError: Cannot read properties of undefined (reading 'body') …
EXIT=1
$ node probe-c-reject-before.mjs     # rejected top-level await
… Error: an await rejected in the unprotected region …
EXIT=1

# AFTER (process 'uncaughtException'+'unhandledRejection' handlers -> die(SETUP)):
$ node probe-c-after.mjs
🛑 COULD NOT RUN (not a verdict): an uncaught exception escaped the client leg — TypeError: Cannot read properties of undefined (reading 'body') …
EXIT=2
$ node probe-c-reject-after.mjs
🛑 COULD NOT RUN (not a verdict): uncaught: an await rejected in the unprotected region
EXIT=2
```

Result: **1 → 2** (both variants). ✅ (Node 22 surfaces a top-level unhandled rejection as an
uncaught exception; installing both handlers covers every escape.)

### Note on the file's own header

`spike-e-reconnect.js:73-79` deliberately does NOT import `hq-bridge-env.js` because that env
installs handlers that `exit(1)` unconditionally — which would turn every exit-2 into an exit-1.
Fix (c) is the RESOLUTION of that same concern: handlers that exit **2** (the correct could-not-run
code) are exactly what the header wanted instead of exit(1). The header note is updated to say so.
