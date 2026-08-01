# Architectural argument — the replay fetch-storm class cannot recur after cutover

> **Item:** `BACKLOG.md` ~~**Replay fetch-storm class is NOT fully closed**~~ · origin
> `overnight-20260722` S1 G6 · **dropped — superseded by the RxDB/Supabase migration**,
> `/nc-roadmap-round` 2026-07-25 (`roadmap.md:1767`).
> **Why a note and not a test:** Engineering KR1 requires *"0 of the 2 superseded fetch-storm-class
> backlog items (replay-fetch-storm, `sync.js` catch-up gate) reproduce against the new
> architecture — evidenced by exactly 1 regression test per item… OR, where no such test is
> constructible, exactly 1 reviewed architectural-argument note per item in
> `.night-crew/knowledge/designs/`."* No test is constructible: `sync-hard-cutover` deletes
> `sync.js` and `backend/internal/sync` outright (`roadmap.md:1698-1699`), so the subject the test
> would assert about does not survive into the architecture the KR asks about. §4 argues that
> point rather than asserting it.
> **Sibling note:** `fetchstorm-catchup-gate-superseded.md` — the *other* superseded item. They
> share a supersession but not a mechanism, and each is argued on its own.
> **Status of the argument:** it describes the **post-cutover** architecture. Until
> `sync-hard-cutover` lands, both mechanisms below are still live in the tree, at the line numbers
> given. This note is not a claim that the bug is fixed today.

## 1. The item, stated as its own record states it

*"S1 gated `SUBMIT_CHECKLIST`, but G6's enumeration of every branch in `applyOp` found
`loadPendingApprovals()` and `loadTemplates()` still fire an **ungated per-op re-fetch** — a
catch-up with N APPROVE ops still storms the approvals queue, N SAVE_TEMPLATE ops still storm the
Builder list. Same root cause, same one-line fix pattern, deliberate-by-omission (the in-code
comments say 'always refreshes')."*

## 2. The original mechanism, verified in the shipped tree

Three parts, all of them present in `sync.js` at the time of writing:

1. **A journal replayed from the client's own position.** `wsCatchUp` (`sync.js:303-315`) issues
   `GET ops/since?lamport_ts=<ts>`, sorts the returned ops, and runs
   `for (const op of ops) { await LAMPORT_CLOCK.receive(op.lamport_ts); applyOp(op, true); }`.
   A fresh context starts at Lamport 0, so the loop replays the **entire historical journal**.
2. **A hand-written per-op dispatch that decides what to re-read.** `applyOp` (`sync.js:401`) is a
   branch table over `op_type`. Each branch independently chooses which whole-collection reads to
   re-issue. Some branches are gated on `!silent || fillState.activeTemplate`; two are not:
   - `sync.js:460-461` — *"The approvers' queue always refreshes."*
     `if (typeof loadPendingApprovals === 'function') loadPendingApprovals();`
   - `sync.js:485-486` — *"The Builder template list always refreshes."*
     `if (typeof loadTemplates === 'function') loadTemplates();`
   Neither consults `silent`. Both fire on every replayed op of their type.
3. **Re-reads implemented as whole-collection HTTP GETs against endpoints separate from the
   store.** `loadPendingApprovals` and `loadTemplates` are `workflows.html` functions that fetch a
   full list and re-render from the response. `applyOp` is not `async` and the loop does not await
   it, so N replayed ops start N concurrent un-awaited requests.

**The failure needs all three at once.** A journal supplies the multiplier N; the branch table
supplies a decision point where a re-read can be attached ungated; the whole-collection GET
supplies the cost. Remove any one and there is no storm — which is what makes the next section an
argument rather than a hope.

## 3. The structural change that supersedes it

`sync-hard-cutover` replaces both write paths in `workflows.html` — `autoSaveField` → `POST
/saveResponse` and the WebSocket/ops-log broadcast — with a single RxDB store replicated against
self-hosted Supabase, and **retires `sync.js`, `backend/internal/sync/` and `/saveResponse`
entirely** (`roadmap.md:1693-1699`). Hard swap, no parallel run.

Take the three parts in turn.

**(1) The journal is replaced by a checkpoint, not by a shorter journal.**
`startHQReplication` (`sync-rxdb/client.js:376-397`) starts `replicateSupabase` per collection with
a `replicationIdentifier` that the code comments call *"stable across reconnects ON PURPOSE: a
different identifier hands the new connection a blank checkpoint, which is a full re-pull rather
than a resume."* A reconnecting client resumes from its checkpoint and pulls **documents that
changed**, in batches (`pull: { batchSize: 50 }`). The unit of catch-up is a document at its
current value, not an event in a history. Ten edits to one checkbox are one document, once — where
the op log made them ten replays.

**(2) There is no successor to `applyOp`, and that is the load-bearing point.**
The storm did not come from ops existing; it came from a hand-written branch table being the place
where "an op arrived" is translated into "so re-read these collections." RxDB has no such
translation step: the pull writes documents into the local collection, and the UI reads the local
collection reactively. Read and convergence are **one mechanism**, where `sync.js` had two that had
to be kept in agreement by hand. There is no file after cutover in which a reviewer could write
`loadPendingApprovals()` next to a comment saying *"always refreshes"*, because there is no
per-message hook to write it in and no `loadPendingApprovals()` to call.

**(3) The whole-collection GET is deleted along with its endpoints.**
`loadPendingApprovals` and `loadTemplates` read the workflow REST surface that the store replaces;
`backend/internal/sync` is deleted and `/saveResponse` removed by the same card. The expensive
operation the multiplier was multiplying no longer exists as a callable.

**Bound, stated as a bound rather than as an absence:** post-cutover network work on catch-up is
`ceil(changed_documents / batchSize)` requests. It is a function of **how much changed**, never of
**how many events occurred**. The original defect is precisely a cost proportional to event count
at constant changed-state; that proportionality has no term left to live in.

## 4. Why this is unconstructible rather than merely unobserved

A regression test for this item must be able to *express* the failure. Its assertion subject is
necessarily some form of *"count the re-fetches of the approvals queue / template list issued
during catch-up, and assert it does not scale with the number of replayed ops."* Every noun in that
sentence is deleted by the cutover:

| The test needs | After cutover |
|---|---|
| a catch-up replay to trigger | `wsCatchUp` deleted with `sync.js` — the replacement is a checkpointed pull with no per-op step |
| N ops of one type to vary | no client-visible op log; `backend/internal/sync` deleted |
| a per-op dispatch to leave ungated | no `applyOp` and no successor — the pull writes documents directly |
| a counted endpoint to observe | `loadPendingApprovals` / `loadTemplates` and their REST reads are what the store replaces |

So the test could still be *written*, and it would pass — because **its subject set would be
empty**. It would count zero re-fetches of an endpoint nobody calls, triggered by a replay that
never happens, and print green.

🛑 **That is not evidence, and this repo has spent a month proving it.** A check whose subject set
can go empty is this codebase's characteristic escape: `B-36` (a security suite that prints `ok`
and exits 0 while skipping every subtest), `B-22` (harness Check B aggregating with OR), and the
`sync-rxdb-conflict-notice-mockup-amendments` hardening round, where deleting every Undo control
turned the whole measurement suite green at *"58 measured, 0 under 44px -> PASS"*. Writing a
vacuous green here would **weaken** E-KR1 rather than satisfy it — it would convert "the mechanism
is gone" into "a test we can no longer fail passed." The KR's own escape hatch exists for exactly
this shape, and this is the affirmative reason to take it, not a shortfall being excused.

The honest converse also holds: **if the storm were still constructible, a test would be owed.**
The claim here is not that testing is inconvenient. It is that the three structural preconditions
in §2 are individually absent from the post-cutover architecture, so there is no arrangement of the
new system that produces the old behaviour.

## 5. What this note does NOT claim

- **It does not claim over-fetching is solved.** RxDB has its own, *different*, already-measured
  cost problem: `startHQReplication` today loops all four collections with `pull:{batchSize:50}`
  and **no selector**, which produced the ~23 s figure behind Fork 1 and means every phone would
  hold every response ever taken. That is why ledger **T-29 decision 105** exists (replication
  scope is per-open-checklist, never all collections at once) and why card
  `sync-replication-scope-per-checklist` is on the slate. **The fetch-storm *class* is retired; the
  general category "reads more than it needs" is not, and it now has a named owner and a scope rule
  a card may not widen without a recorded decision.** Retiring a class by replacing its mechanism
  is not the same as never over-reading again, and this note does not pretend otherwise.
- **It does not claim the bug is fixed today.** `sync.js:461` and `sync.js:486` are ungated in the
  tree right now. The item is superseded, not repaired — the distinction is the whole reason the
  roadmap disposition reads *"dropped — superseded"* rather than *"resolved."*
- **It does not discharge the cutover's own gates.** If `sync-hard-cutover` lands anything short of
  the full retirement in `roadmap.md:1698-1699` — if `sync.js` survives in any form, or a
  compatibility shim re-introduces a per-message dispatch — **this argument lapses and a test is
  owed again.** That is the condition to re-check at the cycle gate, and it is checkable by
  `git log --diff-filter=D -- sync.js`.
