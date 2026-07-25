# Decisions needed — run `overnight-20260725`

> Open forks from tonight's run. Each is stated with the evidence behind it and the options as
> they actually stand. **The run does not decide these** — it executes signed specs and parks
> anything that would require a new decision. Resolve at `/nc-morning-triage`.

---

## FORK 1 — F1's fix regressed two E2E tests. The client half is REQUIRED, not optional. 🛑

**Status: PARKED. This is F1's own named park trigger (ii), fired after the card had already
been merged.** The card entry said, verbatim: *"an existing test or fixture depends on
no-approval submissions reading `'pending'` → that is a contract question, park it rather than
editing the expectation."* That is exactly what happened. I have not edited the tests, not
reverted the fix, and not chosen a repair. This fork is yours.

### What is broken

Two E2E tests fail on the run branch and pass without F1:

| Spec | Test |
|---|---|
| `tests/repro-cut-task.spec.js:153` | AC-6b — submitted record byte-identical after later template edits |
| `tests/sync.spec.js:1581` | Convergence matrix (W-3) — unsubmit transition converges live |

Both fail the same way: after clicking `[data-action="submit"]`, **`.submit-confirm` never
renders**.

### How it was attributed — measured, not inferred

W1's full-suite leg surfaced these two reds and correctly **refused to attribute them** (its diff
touches zero product bytes), flagging the correlation for the orchestrator. The orchestrator then
ran both specs on two trees, same box, same Postgres, separate databases and ports:

| Tree | F1 present? | Result |
|---|---|---|
| `dev` worktree @ `d37fb10` | **no** | **2 passed** (2.6 m) — load 2.84 → 3.28 |
| run branch @ `c14cbce` | **yes** | **2 failed** — load 3.01 → 2.48 |

Not a flake (both fail on isolated re-run at lower load, per W1's own repeat leg), not
`tests/sync.spec.js:1198`, not pre-existing. **`d1674d3` is the cause.**

### Why it happens — mechanism, read from the source

`d1674d3` made `submitChecklist` write `status = 'completed'` for a `requires_approval:false`
template. The Go layer emits the raw DB status (`model.go:67`) and **never emits `'submitted'`** —
grep for a `"submitted"` literal in non-test `backend/internal/workflow/*.go` returns nothing.

`workflows.html` recognises exactly four server statuses:

```
:2093  isSubmitted = submission.status === 'submitted'
:2094  isPending   = status === 'pending_approval' || status === 'pending'
:2095  isApproved  = status === 'approved'
```

`.submit-confirm` is rendered only by those three branches (`:2100`, `:2102`, `:2104`).
**`'completed'` matches none of them, so nothing renders at all.** The same four-value gate also
guards submission hydration at `:2411` and `:2453`, which is why the AC-6b *rendered review* test
fails too.

**The consequence for the roadmap's premise:** the card was slated on the belief that the stuck
`'pending'` was *"harmless today."* It was the opposite of harmless — it was **load-bearing**.
Pre-fix, a no-approval submission came back `'pending'`, hit the `isPending` branch, and rendered
*"Submitted for approval by X. Waiting for manager review."* — semantically **wrong copy** for a
template that needs no approval, but a rendered element, so the suite stayed green. F1 removed the
value the client was accidentally relying on and put nothing in its place.

**The server-side fix is right; it is just half a change.**

### This also answers the open triage question about `0b53d46`

You left this open for triage: *"commit `0b53d46` … a red-first Playwright test with no client
half. Either it gets a `workflows.html` fix as its own card … or it's dropped as redundant with
the Go-layer proof."*

**It is not redundant.** It is the test for the missing half, and the missing half is now proven
necessary by two independent specs the author of `0b53d46` was not looking at. Recommend keeping
that commit and pairing it with whichever option below you choose.

### The options — I am not picking one

- **(a) Add the client half.** Teach `workflows.html` that `'completed'` is a submitted-and-final
  state. Fixes the wrong copy as a side effect (routes to the `isSubmitted` branch, *"Checklist
  submitted. Thanks, X!"*). Largest surface: four call sites (`:2093`, `:2411`, `:2453`, and the
  optimistic value at `:2717`). **This is what `0b53d46` was reaching for.**
- **(b) Revert `d1674d3`** and document the invariant instead. The roadmap card explicitly offered
  this: *"Normalize on submit **or document the invariant explicitly**."* Cheapest, and keeps the
  wrong copy.
- **(c) Map at the API boundary** — store `'completed'`, emit `'submitted'` to clients. Keeps the
  DB clean and the client untouched, at the cost of a translation layer that will surprise the next
  reader.

**Note for whichever you choose:** the frozen-at-submit snapshot gate in the same commit
(`pendingApprovals` on `(s.template_snapshot->>'requires_approval')::boolean IS NOT FALSE`) is
**independent of this fork and is not implicated** — it is the part that closes the actual
approvals leak, and both failing tests are about client rendering, not the queue.

### Why the merge gate did not catch it before the fold

F1 was slated as **seam-confined** — its footprint mapped entirely to `[e2e.seams]`, so it paid the
`workflows|persistence` subset (which I ran this morning: **102 passed, green**). **Neither failing
spec is in that subset.** `tests/sync.spec.js` and `tests/repro-cut-task.spec.js` both exercise the
submit seam that F1 changed, so the seam map is **wrong** for `backend/internal/workflow`
submission-status changes.

**Recommend as a follow-up in its own right:** extend the `[e2e.seams]` mapping so a change to
`submitChecklist` de-confines to those two specs as well. Tonight's evidence is that a green subset
bought false confidence.

---

## FORK 2 — untracked `backlog-round.html` in the main checkout (minor, FYI)

A 225 KB untracked `backlog-round.html` (title: "Backlog") appeared in `/home/jcole/projects/hq`
at **11:26**, during W1's run. It has **never been tracked** in any branch. No card wrote it —
both cards worked in isolated worktrees, and G6 left its worktree byte-clean. `uptime` reports two
logged-in users, so the likely author is a concurrent session of yours.

**I left it exactly as found** — not committed, not deleted, not moved. It is outside every card's
footprint and not mine to dispose of. Delete it or commit it as you see fit.
