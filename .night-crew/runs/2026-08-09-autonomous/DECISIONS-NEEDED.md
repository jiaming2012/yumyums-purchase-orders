# DECISIONS-NEEDED — run `20260809`

**No forks were parked this run.** Both cards landed; neither hit a PARK trigger. What follows is a single
operator-**awareness** item — a "know this before you attest" note, NOT a blocking fork. It is recorded here
so morning triage surfaces it before the milestone-close decision, because it bears on that decision.

---

## 1. (AWARENESS, not a park) — the demo's read surface is a Node RxDB client, not the app UI

**What was decided, and by whom.** The slate made the demo's read surface an **engineer-level decision**
("taken here, not an operator fork"), with the C2 `#sync-one-row` dev surface as an explicit fallback "if
scripting the full fill view proves too heavy for a clean demo." The card exercised that latitude.

**What the demo actually reads through.** `task demo:sync` writes one field through the **real**
`POST /api/v1/workflow/saveResponse` path (real HQ binary, real auth/grant gate, real Postgres, real 75
migrations) and observes it surface in an RxDB-served read via the **identical `replicateSupabase` plugin the
browser's `startHQReplication` uses**, against the **real** Spike A substrate (PostgREST + Realtime) — not a
`page.route` stub. G6 independently confirmed the round trip is genuine and non-vacuous (117 ms; real HQ uuid,
exact field_id + value; substrate restored byte-identical).

**The caveat.** That read runs through a **Node.js RxDB client** (`rxdb/spike-c-read.js`) — **no browser, no
Playwright, no `page.goto`, no app DOM**. It proves the **data-plane** round trip (Postgres → NOTIFY relay →
substrate → RxDB replication) one layer below the UI. The C3 fill-view and C2 `#sync-one-row` browser surfaces
are never pointed at the real substrate in-repo (their tests stub it), so driving either against the real
substrate is the "too heavy for a clean demo" the fallback clause anticipated.

**Why you (operator) should consciously accept or reject this before attesting.** The close bar is *"the operator
ran `task demo:sync` and saw it pass."* It passes, and it clears the roadmap's close-bar **letter** ("one real
checklist", "RxDB-served read", real write). But if your mental model of *"the sync capability running in my dev
environment"* includes **seeing it in the app**, this demo does not show that.

- **If a data-plane proof satisfies you** → attest and close; nothing further is needed.
- **If you want an app-surface read** → file a small follow-up card to drive the real browser fill-view (C3,
  `hq_sync_read` ON) against the real Spike A substrate before close. G6 judged this a legitimate deferral, not
  a defect — so it is your call on the milestone's bar, not a correctness gap.

**G6 severity:** MEDIUM, operator-awareness. **Not a fix round.** (Full detail: `c1-g6-review.md` §Finding 1.)

## (LOW, informational) — `demo-sync.sh` is a documented re-export of `spike-c-roundtrip.sh`

The verdict logic is inherited from Spike C's proven harness; `demo-sync.sh` adds the close-bar framing, its own
exit-2 precondition for a missing harness, close-bar flag vocabulary, and the 0/1/2/3 verdict prose. G6 judged
this a deliberate, defensible packaging (forking ~630 lines of thrice-debugged harness would create a second copy
of the verdict that could drift), not a bare alias. Recorded for awareness; no action needed.
