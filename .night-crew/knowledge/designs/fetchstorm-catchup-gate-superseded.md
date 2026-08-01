# Architectural argument — the catch-up SAVE_TEMPLATE re-fetch cannot recur after cutover

> **Item:** `BACKLOG.md` ~~**`sync.js` catch-up fetch-storm gate**~~ · origin `overnight-20260724`
> S1 observation (reported, not fixed) · **dropped — superseded by the RxDB/Supabase migration**,
> `/nc-roadmap-round` 2026-07-25 (`roadmap.md:1768`).
> **Why a note and not a test:** Engineering KR1 allows *"exactly 1 reviewed architectural-argument
> note per item in `.night-crew/knowledge/designs/`"* where no regression test is constructible. No
> test is constructible here: `sync-hard-cutover` deletes `sync.js` (`roadmap.md:1698-1699`), and
> with it both the trigger and the observable this item's test would need. §4 argues that rather
> than asserting it.
> **Sibling note:** `fetchstorm-replay-class-superseded.md`. That item is about branches that were
> **never gated**; this one is about a branch that **was gated, on the wrong axis**. Same
> supersession, different defect — which is why they are two notes and not one.
> **Status of the argument:** it describes the **post-cutover** architecture. The mechanism below
> is live in the tree today at the line numbers given. This is not a claim that it is fixed now.

## 1. The item, stated as its own record states it

*"`applyOp`'s SAVE_TEMPLATE branch re-fetches `myChecklists` per replayed op whenever a runner is
open (`sync.js` ~491) — a device catching up on a large journal fires an un-awaited fetch per op
(the storm behind every FLD-LIVE-02 red). A one-line gate/debounce fixes the app-level behavior,
but it is a production `sync.js` change and RE-ARMS the attended two-device convergence check —
schedule it with that cost priced in."*

## 2. The original mechanism, verified in the shipped tree

**This one is subtler than its sibling, and the subtlety is the point: the branch *is* gated.**

`applyOp`'s `SAVE_TEMPLATE` / `ARCHIVE_TEMPLATE` branch (`sync.js:484-505`) has two arms:

- **Runner open** — `sync.js:491-492`:
  `if (typeof fillState !== 'undefined' && fillState.activeTemplate) { rerenderOpenChecklistAfterSave(op, silent); }`
- **No runner open** — `sync.js:493-504`: gated on `!silent`, with an in-code comment that spells
  out the reasoning: *"Gated to LIVE ops (!silent): a catch-up/reconnect replay must NOT fire a
  fetch per SAVE_TEMPLATE op — the page-load's own `loadMyChecklists` already reconciled the list,
  so replaying the backlog with per-op fetches would be a needless fetch storm that blocks the main
  thread."*

The second arm gets the gate. **The first arm does not.** The condition it branches on is *"is a
runner open"*, not *"is this a live op"* — so a silent catch-up replay **with a checklist open**
takes the ungated arm, once per replayed op.

What that arm does: `rerenderOpenChecklistAfterSave` (`sync.js:521-541`) calls
`await loadMyChecklists()` — a whole-list re-fetch plus an in-place runner re-render. And `applyOp`
is **not** `async`, so the replay loop in `wsCatchUp` (`sync.js:309-313`) does not await it:

```
for (const op of ops) { await LAMPORT_CLOCK.receive(op.lamport_ts); applyOp(op, true); }
```

N replayed `SAVE_TEMPLATE` ops therefore start **N concurrent, un-awaited** list re-fetches, each
of which re-renders the runner the crew member is actively filling. That is why the record names
it as *"the storm behind every FLD-LIVE-02 red"* — the harm is not only the requests, it is a stale
snapshot landing mid-fill and clobbering optimistic UI.

**Three preconditions, same as its sibling but assembled differently:** (1) a per-op replay loop
supplying N; (2) a branch table whose condition is *view state* rather than *op liveness*; (3) a
re-fetch of a whole collection wired to an in-place re-render of live UI. The defect is the
**mismatch between what the branch tests and what the cost depends on** — a gate on the wrong axis
is not a weaker gate, it is a gate that does not apply.

## 3. The structural change that supersedes it

`sync-hard-cutover` replaces both write paths with a single RxDB store and retires `sync.js`,
`backend/internal/sync/` and `/saveResponse` entirely (`roadmap.md:1693-1699`).

**(a) There is no per-op arrival, so there is nothing left to gate — correctly or otherwise.**
The whole defect is a *conditional attached to an op-arrival hook*. Post-cutover, convergence is
`replicateSupabase` (`sync-rxdb/client.js:376-397`) pulling changed **documents** from a resumed
checkpoint in batches of 50. There is no callback that runs once per historical event, so there is
no place where someone could write a condition that tests the wrong variable. The class of bug
"gated on view state instead of liveness" requires a gate; the gate required a hook; the hook is
gone.

**(b) The refresh and the render stop being two things that must agree.**
The old design's cost came from `loadMyChecklists()` — *fetch the whole list, then re-render the
open runner from it*. Under RxDB the runner renders from a reactive query over the local
collection. A pulled document updates the store, and the open view updates because it is a view of
the store. There is no "re-fetch so the runner can be re-rendered" step to fire per op, because
there is no fetch in the render path at all. **This also disarms the harm the record cared about
most:** the clobber-mid-fill was a *stale whole-list snapshot* overwriting a newer optimistic
value. The replacement merges per document through the conflict handler attached to every
replicated collection (`createHQSyncDatabase`, `sync-rxdb/client.js:338-361`), so a concurrent edit
produces a *reported, recoverable* conflict row rather than a silent overwrite by a snapshot that
happened to arrive later.

**(c) The multiplier is deleted with the journal.** Post-cutover catch-up work is
`ceil(changed_documents / batchSize)`. A backlog of N template edits to one template converges as
**one** document, once — the term that scaled with N does not exist.

**(d) The cost the record attached to the fix disappears with the fix.** The item's own text warns
that a one-line gate *"is a production `sync.js` change and RE-ARMS the attended two-device
convergence check — schedule it with that cost priced in."* That price is paid once, by the
cutover, which re-establishes convergence against the new store as its own gated work. Patching
`sync.js` would have bought a re-armed manual check for a file scheduled for deletion — which is
precisely the reasoning behind the `dropped — superseded` disposition, recorded here so a future
reader does not mistake the drop for neglect.

## 4. Why this is unconstructible rather than merely unobserved

A regression test for this item has to construct the exact conjunction: **a silent catch-up replay,
carrying N `SAVE_TEMPLATE` ops, with a runner open**, and then assert that the number of
`myChecklists` re-fetches does not scale with N. Each of those four clauses names something the
cutover removes:

| The test needs | After cutover |
|---|---|
| a silent catch-up replay | `wsCatchUp` deleted with `sync.js`; replaced by a checkpointed document pull with no per-op step |
| N `SAVE_TEMPLATE` ops | no client-visible op log; `backend/internal/sync` deleted |
| the runner-open arm of a branch table | no `applyOp`, no successor branch table, no view-state condition on an arrival hook |
| a counted `myChecklists` re-fetch | the runner renders from a reactive local query; there is no fetch in the render path to count |

The test could be written. It would go green — **on an empty subject set.** Zero re-fetches of an
endpoint the app no longer calls, during a replay that no longer occurs, in a branch that no longer
exists.

🛑 **A green from an empty population is the escape this repo keeps catching, and shipping one here
would be self-inflicted.** `B-36`: a security suite prints `ok` and exits 0 while skipping every
subtest. `B-22`: harness Check B aggregates with OR. The conflict-notice mockup's hardening round:
deleting every Undo control produced *"58 measured, 0 under 44px -> PASS"*, raw exit 0. The
cheapest way to fail E-KR1 in substance while passing it on paper is to write two vacuous specs and
call the KR met. The KR's own text anticipates this and offers the note instead — **that is why
this file exists, and it is an affirmative choice, not a concession.**

Stated as a falsifiable claim rather than a comfort: *there is no configuration of the
post-cutover architecture in which the number of network reads issued during convergence is a
function of the number of historical template-edit events.* If someone can construct one, this
note is wrong and a test is owed.

## 5. What this note does NOT claim

- **It does not claim the item is fixed today.** `sync.js:491` is ungated in the tree right now,
  and the FLD-LIVE-02 reds it produced are part of this cycle's flake record. Superseded ≠ repaired.
- **It does not claim RxDB reads only what it needs.** `startHQReplication` currently loops **all
  four** collections with `pull:{batchSize:50}` and **no selector** — the ~23 s figure behind
  Fork 1, and the reason ledger **T-29 decision 105** fixes replication scope at
  per-open-checklist and card `sync-replication-scope-per-checklist` exists to land the pull
  filter. A card may not widen that scope without a recorded decision. **The storm class is
  retired by construction; over-fetching in general is a live, owned, separately-tracked
  concern.**
- **It does not claim the conflict story is proven.** (b) above says a per-document merge replaces
  a whole-snapshot clobber. That the merge behaves correctly is
  `sync-rxdb-replication-and-conflict-handler`'s and the conflict-notice UI's evidence, not this
  note's — cited here for the shape of the change, not borrowed as proof.
- **It lapses if the cutover lands short.** If `sync.js` survives in any form, or a compatibility
  shim re-introduces a per-message dispatch with a view-state condition, this argument is void and
  a test is owed again. Checkable at the cycle gate with
  `git log --diff-filter=D -- sync.js` plus a grep for any surviving per-op arrival hook.
