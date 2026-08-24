# Merge intent — A1 · `gate-rls-count-assertion`

Run `20260806`, branch `card/a1-rls-count-assert`, based at `ef314e0`.
Closes Q-KR1's residual ("the 59 count asserted rather than inferred") and closes **B-36**.

## What this card is NOT

B-36's *mechanism* (typed opt-out `HQ_SYNC_SUBSTRATE_OPTIONAL=1` as the only skip door;
unresolvable substrate without it ⇒ `t.Fatalf`) landed on `dev` at commit `4615661`
(2026-08-01). **This card does not touch it.** No change to `spikeResolution`,
`resolveSpikeConfig`, `spikeSubstrateOptional`, `spikeGate`, or any of the three tests that
already pin them.

## Files touched

| File | Why |
|---|---|
| `backend/internal/sync/spikestack_gate_test.go` | **All new code lands here.** Additive-only: the subtest-count assertion (structural + execution-backed) and the exit-code asymmetry pin. Nothing existing in the file is edited. |
| `.night-crew/knowledge/BACKLOG.md` | B-36's entry marked resolved, with the fix-vs-file dates and the B-38 channel-gap note. |

**`backend/internal/sync/rowvisibility_rls_test.go` — NOT EDITED. Zero bytes changed.**
**`backend/internal/sync/jwtbridge_rls_test.go` — NOT EDITED. Zero bytes changed.**

Both were in the card's declared footprint; the design deliberately kept them out of the
diff (see next section). If a merge shows a hunk in either file coming from this branch,
that hunk is not ours — reject it.

## 🛑🛑 A3, READ THIS FIRST — this card DOUBLES B-35's collision window

**B-35 is the bug A3 exists to fix, and this card makes it twice as likely to fire before
A3 lands.** Stated here because the card shipped without stating it (fix-round finding F4).

`TestRowVisibilitySubtestCount_Executed` re-invokes `go test -run '^TestRowVisibilityRLS$'`
as a **nested** process. That child re-enters `rvConnect`, which opens with

    DROP DATABASE IF EXISTS <rvHQDatabase()> WITH (FORCE)
    CREATE DATABASE <rvHQDatabase()>

(`rowvisibility_rls_test.go:400-403`). So on this branch a single `go test ./...` performs
**TWO** drop/create cycles of the RLS fixture database where it previously performed one —
the parent's own `TestRowVisibilityRLS` and the child's. Under this repo's normal shape
(three tracks dispatched concurrently) **the window in which another agent's fixture can be
destroyed is doubled**, and B-35's default database name is a shared constant
(`hq_test_b2_fdw`) that any leg which forgets `HQ_RLS_TEST_DB` will land on.

**A3's fix — unique-per-process or fail-loud-on-unset — covers the nested run automatically**,
because the child inherits `HQ_RLS_TEST_DB` through `childEnv`. Nothing here needs undoing.
What A3 must *know* is that its own before/after measurements of B-35 are being taken against
a tree whose drop/create count per `go test ./...` is 2, not 1.

🛑 **Do not "fix" this by deleting the nested run.** The nested run IS the execution-backed
half of the count assertion — the thing Q-KR1 asked for. The correct order is: A3 makes the
fixture name safe, and the doubled cycle stops mattering.

The two nested runs added by `TestGateChildGuard_IsNotASkipDoor` (fix round) do **not** add
to this: both stop at the recursion guard, before `resolveSpikeConfig`, `rvConnect`, Postgres
or the network.

### Measured cost

`go test -count=1 -v ./internal/sync/` on the pre-fix branch commit `71dbd28`, as the reviewer
measured it: **18.465s** — `TestRowVisibilityRLS` 6.84s, `TestRowVisibilitySubtestCount_Executed`
**7.76s**, `TestSubstrateGate_ExitCodeAsymmetry` **2.12s**. The card claimed "+8s"; the honest
implementation-round figure is **~+10s, roughly doubling the package**.

Re-measured on the fix-round tree, same command, under the suite mutex:

| | |
|---|---|
| package wall clock | **36.3s** (`real 0m37.104s` incl. build) |
| `TestRowVisibilitySubtestCount_Executed` | 7.85s |
| `TestSubstrateGate_ExitCodeAsymmetry` | 1.94s |
| `TestGateChildGuard_IsNotASkipDoor` *(new, fix round)* | 1.94s |
| `_Structural` + `_CountsWhatItClaims` | 0.00s |
| **this card's own total** | **≈ 11.7s** |

🛑 **Do not read the 18.5s → 36.3s difference as this card's cost.** The package wall clock is
load-sensitive: in the same fix-round run the *pre-existing*
`TestResolveEntityAccess_RoleAssignmentCartesian` alone took 8.68s and `TestRowVisibilityRLS`
7.39s. The defensible figure is the sum of the tests this card adds — **≈ +11.7s**, of which
the fix round contributes **+1.9s** — on every `go test ./...` in the repo. Accepted
deliberately: the alternative is a coverage claim nobody re-checks.

## 🛑 What card A3 (`gate-rls-fixture-ownership`) must preserve

A3 owns `rowvisibility_rls_test.go`'s setup/fixture-ownership path and runs after this card
in Track A. This card creates **one coupling** between the two files, and it is a read-only,
one-directional one:

1. **`spikestack_gate_test.go` now contains the constant `wantRowVisibilitySubtests = 59`
   and a test that counts the top-level `t.Run(...)` calls inside `TestRowVisibilityRLS`
   in `rowvisibility_rls_test.go`.** Two independent counters assert it:
   - a **structural** counter that parses `rowvisibility_rls_test.go` with `go/ast`,
     counting `t.Run` calls in the function body (not descending into `t.Run` closures)
     and multiplying by the length of any enclosing `for … range []T{…}` composite literal;
   - an **execution** counter that re-invokes `go test -run '^TestRowVisibilityRLS$' -v`
     as a subprocess and counts distinct `TestRowVisibilityRLS/<name>` top-level subtests.

2. **A3 may freely change fixture ownership, `rvConnect`, `rvSeed*`, teardown, helpers and
   the *bodies* of any subtest — none of that is observed here.** The count assertion reads
   only the *number of top-level `t.Run` registrations* in `TestRowVisibilityRLS`.

3. **If A3 adds or removes a top-level `t.Run` in `TestRowVisibilityRLS` (or changes the
   length of the `for _, ft := range []struct{ name, path string }{…}` case list around
   line 1296), both counters go red and A3 must bump
   `wantRowVisibilitySubtests` in `spikestack_gate_test.go` in the same commit.** That is
   the feature, not a collision: a suite that quietly loses a case must announce itself.
   The failure message names the constant, the file and the line to change.

4. **A3 must NOT satisfy a red count by deleting or relaxing the assertion.** If A3's change
   legitimately changes the case count, bump the constant and say so in its report. If A3
   cannot reconcile the number, that is a finding for triage, not a waiver.

5. The structural counter's AST walk assumes the current shape: top-level `t.Run` calls
   directly in `TestRowVisibilityRLS`'s body or directly inside a `for … range` over a
   non-empty composite literal. **If A3 restructures the suite into a table-driven form or
   moves the subtests behind a helper, the structural counter must be updated in the same
   change.**

   🛑 **The first draft of this paragraph claimed the execution counter always cross-checks
   the structural one, and that claim was wrong** (fix-round finding F3). The execution
   counter needs a substrate; on a machine that legitimately opted out it **SKIPS**, and the
   structural counter is then the only counter in the tree. So the loudness has to come from
   the walker itself, and after the fix round it does:

   - a **C-style `for i := 0; i < n; i++`** containing `t.Run` is now **reported as
     unreadable**, not scored. Before the fix it scored **1 with zero reports** — a
     four-case block silently counted as one. `TestRVTopLevelSubtestCount_CountsWhatItClaims`
     now carries a C-style fixture that fails if that regresses.
   - a `for … range` over anything that is not a non-empty composite literal (a variable, a
     call, an integer count) was already reported and still is; the report now names the
     shape.
   - `if`/`switch` around a registration is deliberately **not** reported — that makes a
     registration conditional rather than multiplied, and it is exactly the disagreement the
     two counters exist to surface.

6. 🛑 **A red count is not automatically a constant bump, and the failure text no longer says
   it is.** The original message said *"If you deliberately added or removed a variant, bump
   `wantRowVisibilitySubtests`"* — advice that is exactly wrong for the restructure case,
   where bumping ratifies a walker that can no longer read the source and blinds the counter
   permanently. The message now enumerates three causes (deliberate change / lost
   registration / unreadable restructure), names the substrate-skip caveat above, and tells
   the reader to confirm the walker still understands the shape **before** deciding whether
   the number is a real change. A3 should read the failure, not just the constant.

## Red-first

Q-KR3 wants this as a **section in the merge-intent**, not as a commit trailer. The substance
existed only in commit `2b3a50f`'s body, and Q-KR3 is measured by `.night-crew/runs/`, not by
`git log` — so it is carried here, where it is gradeable. This is the first gradeable cycle,
and A1 was the **third** card of run `20260806` to ship without this section (fix-round
finding F1).

**Every row below was RE-RUN in the fix round against the fixed tree** — not transcribed from
the implementation round. Script and logs: `scratchpad/a1fix/mutations.sh`, `m1…m6*.log`.
Isolation: `DB_TEST_URL=…/hq_test_a1fix_0806_go` (created and migrated to goose **74** for
this leg), `HQ_RLS_TEST_DB=hq_rls_a1fix_0806`, `HQ_SYNC_SUBSTRATE_OPTIONAL` **unset**,
`HQ_SYNC_GATE_CHILD` **unset**, spike substrate up and used read-only. Every mutation was
reverted from a byte-identical backup before the next; `git status` after the last one showed
only the three files this card intends to change.

**Tree captured red against:** each mutation is applied on top of the fix-round tree
(base `ef314e0` → `50115dc` → `2b3a50f` → `71dbd28` → the fix-round commits). **Green after:**
the tree with the mutation reverted, re-run in the same script (`GREEN_AFTER_EXIT=0`, all
eight named tests).

| # | Mutation | Named test(s) | RED (observed) | GREEN |
|---|---|---|---|---|
| **M1** | delete `{"V20 hq_template_approvers", …}` from the `V15-17,20` range case list in `rowvisibility_rls_test.go` | `TestRowVisibilitySubtestCount_Structural` + `_Executed` | `EXIT=1`. Structural: *"registers **58** top-level subtests, want 59"*. Execution: *"reported **58** depth-1 subtests AT RUNTIME, want 59"* — and the nested `TestRowVisibilityRLS` itself still printed `--- PASS`, which is the whole point | `EXIT=0`, 59/59 |
| **M2** | delete a whole top-level `t.Run` (`FLOOR/HQ's three views are populated`) | same two | `EXIT=1`, **both** counters 58/59 | `EXIT=0`, 59/59 |
| **M3** | `spikeResolution`'s `default:` `spikeGateFail` → `spikeGateSkip` (B-36 reverted to its filed behaviour) | `TestSubstrateGate_ExitCodeAsymmetry` | `EXIT=1`. Arm 1 **FAIL** — *"🛑 B-36 IS BACK … the row-visibility attack suite did not run, and the package said `ok`"*. Arm 2 **PASS** (the opt-out is supposed to keep working). The asymmetry is what fails, not the pair | `EXIT=0`, both arms |
| **M4** | `resolveSpikeConfig`'s substrate `t.Fatalf` → `t.Skipf` | the three **pre-existing** pins vs. the new one | 🛑 **`TestSpikeResolution_OptOutIsTheOnlySkipDoor`, `TestSpikeSubstrateOptional_IsExplicit` and `TestSpikeGate_Asymmetry` all still `--- PASS`, package `ok 0.006s`, `EXIT=0`** — while `TestSubstrateGate_ExitCodeAsymmetry` **FAILs** with *"B-36 IS BACK"*, `EXIT=1`. This is the necessity proof: the pre-existing pins cover the decision function, and the decision function is not the exit code | `EXIT=0` |
| **M5** | *(fix round, F2)* the recursion guard reverted to `if os.Getenv(gateChildEnv) == "1" { t.Skip }` | `TestGateChildGuard_IsNotASkipDoor` | `EXIT=1` — *"🛑 THE SKIP DOOR IS OPEN AGAIN"*. And the reviewer's own command run against the mutated guard: `HQ_SYNC_GATE_CHILD=1 go test -run '^(TestRowVisibilitySubtestCount_Executed\|TestSubstrateGate_ExitCodeAsymmetry)$'` ⇒ **`ok … 0.008s`, `EXIT=0`, nothing ran**. The same command on the fixed tree ⇒ `EXIT=1`, both tests FAIL naming the reason | `EXIT=0`, 9 subtests |
| **M6** | *(fix round, F3)* remove `case *ast.ForStmt:` from `rvTopLevelSubtestCount` | `TestRVTopLevelSubtestCount_CountsWhatItClaims` | `EXIT=1` — *"a C-style `for` containing t.Run was not reported as unreadable (**0 reports, want 1**). It **scored 2**"* — i.e. a 4-case loop counted as one registration, silently, exactly as the reviewer described | `EXIT=0` |

🛑 **M4 and M5 are the two rows that matter most, and they say the same thing about different
code.** M4: pinning a decision function does not pin the exit code the ladder reads. M5: a
guard that skips on any value is a gate that can be turned off from a shell. Both are "a check
whose subject set can go empty", which is this repo's characteristic defect — and M5 was found
*in this card*, by review, after the card shipped.

## What MUST survive any merge

- `wantRowVisibilitySubtests` and both counters in `spikestack_gate_test.go` — this is the
  entire deliverable. Do not drop either counter in favour of the other: the structural one
  runs with **no substrate**, the execution one is the one Q-KR1 asked for.
- The exit-code asymmetry test — **`TestSubstrateGate_ExitCodeAsymmetry`**, declared at
  `spikestack_gate_test.go:969` on the pre-fix commit. (The first draft of this artifact
  named it `TestSpikeSubstrateGate_ExitCodeAsymmetry`, which **does not exist**; a merge
  resolver following the artifact literally would have found no such test. Corrected in the
  fix round, finding F6.) Stripped substrate + no opt-out ⇒ **non-zero**; opt-out set ⇒
  **zero**. This is B-36's property pinned as an executable, not as prose in a banner.
- The recursion guard env var `HQ_SYNC_GATE_CHILD`, **and specifically its token form**.
  Every subprocess-spawning test in this file stops when it is set — it *skips* only for a
  value the parent minted, and **fails hard for any other value**. 🛑 Do not "simplify" this
  back to `== "1"`: that is what the card originally shipped, and one exported shell variable
  then disarmed both the count assertion and the exit-code pin while the package printed `ok`
  and exited 0 (fix-round finding F2 — see the Red-first section). `TestGateChildGuard_IsNotASkipDoor`
  pins both arms. Removing the guard entirely makes `go test ./...` fork-bomb.
- B-36's closure text in `BACKLOG.md`, including the fix-vs-file dates.

## What is safe to drop

- Nothing in the source diff. It is additive and self-contained in one file.
- In `BACKLOG.md`, if another card also edits B-36's entry, keep whichever text records
  **both** (a) the mechanism landing at `4615661` on 2026-08-01 and (b) the count assertion
  landing on this branch; the prose framing is negotiable, those two facts are not.

## Gates, re-run in the fix round

All legs serialized under the shared suite mutex
(`flock scratchpad/playwright.lock`), each launched detached so the 10-minute
foreground cap could not orphan one. Isolation for this leg: `TEST_PORT=3112`,
`TEST_DB_NAME=hq_test_a1fix_0806`, `HQ_RLS_TEST_DB=hq_rls_a1fix_0806`,
`DB_TEST_URL=…/hq_test_a1fix_0806_go` — **created fresh and migrated to goose 74**, because
the shared `hq_test_go` is corrupted (goose 73 applied, 72 absent) and an unmigrated
isolation DB produced 20 spurious FAILs for a reviewer earlier tonight.

| Gate | Result |
|---|---|
| **G1** | `go build ./...` **0**, `go vet ./...` **0**, both from `backend/` |
| **G2 (Go)** | `go test -p 1 -count=1 -v ./...` **EXIT=0**. **9** packages with tests, **245** top-level tests, **455** results, **0** FAIL. `internal/workflow` **35** ✔. `internal/sync` **47 top-level / 158 results** (see the count table below) |
| **G2 (Go) · `internal/sync` evidence** | `go test -count=1 -run TestRowVisibilityRLS -v` ⇒ **EXIT=0**, `--- PASS: TestRowVisibilityRLS (6.51s)`, **59** depth-1 subtests. `HQ_SYNC_SUBSTRATE_OPTIONAL` **unset**; `HQ_SYNC_GATE_CHILD` **unset**. The same 59 is independently visible inside the full `./...` run |
| **G2 (Playwright)** | 🛑 **The card's first Playwright result is DISCARDED, not cited.** It overlapped A2's 11-minute `verify-test-harness.sh` (a `go test` over 7 packages), and the slate rule on overlap is unconditional. Re-run clean, alone under the mutex: `npx bddgen` **0**, then `npx playwright test --retries=0` ⇒ **EXIT=0**, **791 passed + 6 skipped = 797 across 29 files**, exactly **ONE** summary block counted over the complete log (`grep -c "passed ("` = 1, never a tail), **23.5m** wall |
| **G3** | N/A — `openspec: absent`, ledger §T-34 decision 140 |
| **G4** | **Not run, correctly.** The diff is Go test files + markdown; no precached asset, no `sw.js` change, no `package.json`/`version.go` movement |

**LST-17, the standing armed red:** `tests/sync.spec.js:446` *"list page progress decrements
when another device unchecks a field"* — ledger decisions 44 and 77, `slate-20260729:48`
(*"Expect it; it is not a regression"*). 🛑 **In this run it PASSED** (`✓ 621`), as did its
sibling at `sync.spec.js:1006`. Reported as observed. A pass is not evidence the underlying
defect is fixed — it is an intermittent — so the armed-red status is unchanged and a future
run seeing it red should still not treat that as a regression.

## 🛑 The test count this package reports has MOVED — record it before it is read as wrong

`internal/sync` ran **142** tests (top-level + subtests) in the evidence lines of prior runs,
and those lines are still in the ladder's history. On this branch it runs **158**, and every
one of the extra 16 is deliberate:

| | |
|---|---|
| baseline on `dev` | **142** |
| + `TestRowVisibilitySubtestCount_Structural`, `_Executed`, `TestRVTopLevelSubtestCount_CountsWhatItClaims`, `TestSubstrateGate_ExitCodeAsymmetry` | +4 top-level |
| + `TestSubstrateGate_ExitCodeAsymmetry`'s two arms | +2 subtests |
| *(implementation round subtotal — the figure the reviewer predicted)* | **148** |
| + `TestGateChildGuard_IsNotASkipDoor` *(fix round, F2)* | +1 top-level |
| + its 7 classifier cases and 2 end-to-end arms | +9 subtests |
| **this branch** | **158** |

Measured, not derived: `go test -p 1 -count=1 -v ./...`, `internal/sync` block = **47
top-level tests, 158 results**, 2 SKIPs (`TestProxyLive_RealtimeUpgrade`,
`TestProxyLive_RESTRequest` — the pre-existing `HQ_SYNC_SPIKE_LIVE` opt-in, unrelated to this
card). Whole-suite total **455** results across **9** packages with tests (prior runs cite
"~439"); `internal/workflow` **35**, as the ladder requires. A later run reading 158 against a
remembered 142 should read this table, not file a bug.

## Anything else

- **No migration, no frontend asset, no `night-crew.toml` key, no new dependency, no change to
  `sw.js`'s precache set.** G4 is therefore not run for this card and correctly so.
- 🛑 **`night-crew.toml` has NO footprint entry for `backend/internal/sync`** (it maps
  `backend/internal/workflow`, `…/inventory`, `…/recipes`, `…/auth`, `…/purchasing`,
  `…/onboarding`, `…/users` and stops). A Go-test-only diff has no Playwright coupling so
  nothing escaped here — this fix round ran the **full** suite regardless — but the gap is the
  shape CLAUDE.md warns about and it is worth a triage line. Not fixed here: `night-crew.toml`
  is W0's file this run.
- 🛑 **Card A4 owns `reference/gate-ladder.md`; this card must not touch it.** The reviewer's
  finding F7 — line 27 still instructs a human to *"cite `-run TestRowVisibilityRLS -v`
  showing subtests ran (59)"*, i.e. the manual inference this card replaces — **is routed to
  A4** and is a real dependency: until A4 lands, the ladder still describes the old ritual
  while the tree asserts it. Nothing in this branch conflicts with A4's edit.
- **A citation slip that predates this card and was deliberately left alone.**
  `spikestack_gate_test.go:79` (base `ef314e0`, from the 2026-08-01 B-36 fix commit `4615661`,
  not from this branch) says *"ledger T-30, decision 111's 'Also folded' paragraph"*. Decision
  111 (`ledger.md:2171`) is the RxDB per-table write-policy call; the "Also folded" paragraph
  (`ledger.md:2232-2239`) is a separate planner call that carries no decision number. The
  identical slip in `BACKLOG.md` **was** this card's, and is corrected on this branch
  (fix-round finding F9). The one in the Go banner is left untouched so the card's
  "additive-only, nothing existing in the file is edited" claim stays literally true — it is a
  one-line doc fix for whoever next edits that banner.
