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
   composite literal. **If A3 restructures the suite into a table-driven form or moves the
   subtests behind a helper, the structural counter must be updated in the same change** —
   it will fail loudly (count mismatch) rather than silently under-count, because the
   execution counter cross-checks it against a real `-v` run.

## What MUST survive any merge

- `wantRowVisibilitySubtests` and both counters in `spikestack_gate_test.go` — this is the
  entire deliverable. Do not drop either counter in favour of the other: the structural one
  runs with **no substrate**, the execution one is the one Q-KR1 asked for.
- The exit-code asymmetry test (`TestSpikeSubstrateGate_ExitCodeAsymmetry`): stripped
  substrate + no opt-out ⇒ **non-zero**; opt-out set ⇒ **zero**. This is B-36's property
  pinned as an executable, not as prose in a banner.
- The recursion guard env var `HQ_SYNC_GATE_CHILD=1`. Every subprocess-spawning test in this
  file skips when it is set. Removing it makes `go test ./...` fork-bomb.
- B-36's closure text in `BACKLOG.md`, including the fix-vs-file dates.

## What is safe to drop

- Nothing in the source diff. It is additive and self-contained in one file.
- In `BACKLOG.md`, if another card also edits B-36's entry, keep whichever text records
  **both** (a) the mechanism landing at `4615661` on 2026-08-01 and (b) the count assertion
  landing on this branch; the prose framing is negotiable, those two facts are not.

## Anything else

Nothing here — this card ships no migration, no frontend asset, no `night-crew.toml` key,
no new dependency, and no change to `sw.js`'s precache set.
