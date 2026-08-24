> **RESOLVED 2026-07-25 — recorded as `ledger.md` T-22, decisions 49–54.**
> FORK 1 → option (a), add the client half; split out as the roadmap card
> `workflow-submission-status-client-half`, repriced to **seven** call sites, not four
> (decision 49). FORK 2 → disposed: `backlog-round.html` is the operator's own
> `/nc-roadmap-round` render, not a run artifact; left untracked (decision 52). FORK 3 →
> **field-level three-way merge**, same-field clashes falling back to master-wins **plus a
> `conflict$` notice**; the 2026-07-24 signing of last-write-wins is struck (decision 50).
> FORK 4 → **stay gateway-less** with a permanent client-construction helper, plus a rider to
> pin `@supabase/supabase-js` and add an upgrade smoke test (decision 51).
>
> Two corrections from the attended adversarial pass that change how this document reads:
> FORK 1's *"nothing renders at all"* is wrong — an **editable checklist with a live
> `#submit-btn`** renders, and a second submit writes a second row. FORK 3's `conflict$`
> caveat is wrong in the conservative direction — it fires **per document**, carrying the
> document id.
>
> Kept as the analysis record.

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

---

## FORK 3 (W2 `sync-spike-rxdb-replication`) — RxDB's conflict behaviour is not the last-write-wins the decision assumed

**Status of the card: not blocked.** W2 completed with a **GO** on RxDB. This is
a finding that needs a product decision, not a failure. Nothing was "fixed" in
code, deliberately — the card's instruction was to record what actually happens
and let it route.

### What was assumed

The explore session (2026-07-24) chose **last-write-wins, with no custom conflict
handler**.

### What was observed

Constructed case, run twice with identical results
(`.night-crew/qa/spike-supabase/rxdb/proof-lww.js`): one agreed document; client
goes offline; **Postgres edited first (T1)**; **RxDB edited second (T2 > T1)**,
so the local write is *strictly later* in wall-clock time; client reconnects.

```
local  body after reconnect : REMOTE-EDIT (written first, T1)
remote body after reconnect : REMOTE-EDIT (written first, T1)
replication errors surfaced : 0 []
conflict handler invocations: 1
    newDocumentState.body  : LOCAL-EDIT (written second, T2)    <- the local (later) write
    realMasterState.body   : REMOTE-EDIT (written first, T1)    <- what the server actually held
    handler CHOSE          : REMOTE-EDIT (written first, T1)
```

**The later write lost.** The behaviour is not last-write-wins; it is
unconditional **master-wins**. No clock participated in the decision at all —
the mechanism is a compare-and-swap against the assumed master state, and RxDB's
`defaultConflictHandler` resolves every conflict by returning `realMasterState`
(its own source comment: *"The default conflict handler will always drop the fork
state and use the master state instead."*). Client clock skew is irrelevant.

**And the loss is silent by default.** `error$` emitted zero events, nothing was
thrown, and nothing reaches a user without code written to put it there. Out of
the box, the crew member's edit simply never happened as far as they can tell.

**But it is not unobservable** — this was stated wrongly in the first draft of
this fork and is corrected here, because it changes the price of option 4 below.
`replicateSupabase()` returns an `RxReplicationState` that exposes a
**`conflict$`** observable alongside `error$`
(`rxdb/dist/esm/plugins/replication/index.js:44,51,287-289`). Re-running the
scenario above with a `conflict$` subscription added gives **`error$` 0 events,
`conflict$` 1 event**, whose `input.newDocumentState` is the discarded local
write in full (`body: "LOCAL-EDIT (written second, T2)"`) and whose `output` is
the server state that replaced it. So the app *can* be told, and can be told
*what was lost* — it just isn't, unless someone subscribes.

### Why this needs you rather than an engineer

Concretely for HQ: a crew member completes a checklist on a phone with no signal
in the truck; a manager edits the same submission from the office; the phone
reconnects. **As configured today, the crew member's work is dropped without a
word** — from inside the app the offline edit simply never happened. (What
changes that is a `conflict$` subscription, above — but that is a decision
someone has to make, and nobody has made it.)

For a product whose stated core value is *"accountability — who checked what"*,
whether that is acceptable is a product call, and which rule replaces it is a
domain call that an engineer cannot make alone. HQ's rows are frequently
multi-actor (a submission has a submitter *and* an approver), so "the owner
wins" does not express it either.

### The shape of the options (not a recommendation — the choice is yours)

RxDB supports a per-collection custom `conflictHandler`; that is the hook, and
the implementation is small once the rule is decided. **What is expensive is the
rule, not the plumbing** — the hook for deciding a winner exists, and so does
the signal (`conflict$`) for telling the user a decision was made.

1. **Accept master-wins as-is** — cheapest, and defensible for fields only one
   role ever edits. Requires deciding that silent loss is acceptable, or scoping
   sync to rows where collisions cannot occur. Note this pairs cheaply with 4:
   master-wins *plus a visible notice* is a materially different product from
   master-wins in silence, and costs a subscription.
2. **Genuine last-write-wins** — what was assumed. Needs a timestamp that
   actually participates in the decision, and a decision about *whose* clock;
   note the server stamps `_modified` via trigger, so a client-authoritative
   timestamp would be new work and would reintroduce clock-skew risk.
3. **Field-level merge** — both edits survive when they touch different fields.
   Fits checklists well (different people fill different lines); most work.
4. **Surface the conflict to the user** — orthogonal to 1–3 and arguably worth
   doing under any of them. **Cheaper than the first draft of this fork implied.**
   The signal already exists and already carries the lost document:
   `replicationState.conflict$.subscribe(e => …)` gives you
   `e.input.newDocumentState` (what the crew member had) and `e.output` (what
   replaced it). The cost is a subscription plus UI — deciding what to show,
   when, and whether the crew member can recover their value — **not new
   plumbing in the sync layer.** Two caveats: `conflict$` is a plain `Subject`,
   so you must subscribe where the replication is constructed or you miss the
   event; and it fires per replication, carrying no user-facing text of its own.

### Where the evidence lives

- Runbook: `.night-crew/qa/spike-supabase/README.md`, half 2 step 5.
- Verdict: `.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md`,
  section "THE FINDING".
- Reproduce: `cd .night-crew/qa/spike-supabase/rxdb && npm ci && node proof-lww.js`
  (needs W1's stack up and `/usr/local/go/bin` on PATH).
- Reproduce the `conflict$` result: add `rep.conflict$.subscribe(c => console.log(JSON.stringify(c)))`
  next to the existing `rep.error$.subscribe(...)` in `proof-lww.js`'s
  `startReplication()` and re-run. One line; runbook half 2 step 5 has the
  captured output.

**This blocks sizing `sync-rxdb-schema-and-replication` accurately** — the
conflict policy is that card's real work — but it does not block W2 merging.

---

## FORK 4 (W2 `sync-spike-rxdb-replication`, minor) — Kong, or a permanent client shim?

Lower stakes, but it should be decided rather than inherited by accident.

`@supabase/supabase-js` freezes `<baseUrl>/rest/v1` and `<baseUrl>/realtime/v1`
in its constructor — it assumes **one origin behind Kong**. W1 deliberately did
not deploy Kong, so a stock `createClient()` cannot reach either service in our
stack.

W2 bridged it in ~25 lines inside the harness, using only the extension points
supabase-js already exposes (`global.fetch`, `realtime.transport`). It worked
first try, so this is not a blocker. But the migration must choose:

- **Run Kong** — supabase-js sees the single origin it expects; one more service
  to run, configure and secure.
- **Stay gateway-less and keep a small permanent client-construction helper in
  HQ** — one fewer service, but standing HQ code coupled to a supabase-js
  internal (its derived path prefixes) that could move in a minor release.

Either is viable. Owner: whichever card owns client construction
(`sync-jwt-bridge-endpoint` is the likely home). Evidence: runbook half 2, step 1
and "What the shim means for the real migration".
