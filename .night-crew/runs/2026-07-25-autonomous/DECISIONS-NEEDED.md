<!-- MERGE NOTE (card W2): this file did not exist at W2's branch point
     (overnight-20260725 @ 51d0c02), so W2 created it containing ONLY the W2
     sections below, starting at heading level 2 and with no document title, so
     that it CONCATENATES cleanly under whatever title/earlier sections the
     orchestrator's copy already has. If both sides exist at merge time, KEEP
     BOTH — the orchestrator's FORK 1 (F1 regression) and FORK 2 sections must
     survive intact. W2 overwrote nothing.

     NUMBERING: W2 checked `overnight-20260725` at 396b97e and found the
     orchestrator's copy already using FORK 1 and FORK 2, so W2's forks are
     numbered 3 and 4. If the numbering has moved again by merge time,
     renumber these two — the content, not the number, is the payload. -->

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
