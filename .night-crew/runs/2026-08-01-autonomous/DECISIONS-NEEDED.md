# Decisions needed — run 20260801

Open forks for morning triage. The run does not decide these; it records them.

---

## Fork 1 — Does decision 92 survive the measured per-row cost of the fdw RLS path?

**Card:** `sync-rxdb-row-visibility-rls` (Track B)
**Source:** G6 adversarial review, finding F3. Measured by the reviewer in its own stack with
`EXPLAIN ANALYZE` under `set role authenticated` — not estimated, not extrapolated.

| rows in `checklist_templates` | execution time |
|---|---|
| 5 | **177 ms** |
| 205 | **4,698 ms** |

That is **~23 ms per row, linear**. The predicate is evaluated per row with no batching, so the
shape is a per-row foreign scan and it does not improve with scale. Some portion is loopback /
Docker NAT and would differ in production, but the shape is structural, not environmental.

**Why this is an operator question and not a run question.** Sign-off accepted the standing cost
as *"HQ's Postgres is on the network path of every RLS row check."* That sentence is true, and it
does not convey ~23 ms **per row**. A 1,000-row collection is ~23 s for a plain list, and RxDB
initial replication pulls whole collections — so this lands directly on the cards that were to
build on it.

**🛑 This is the card's own named PARK trigger**, verbatim from the slate: *"PARK if … an
operator-only question about the accepted network-path cost surfaces."* It surfaced.

**What the run did instead of parking, and why — the operator can overrule this.** The card's work
is complete, G6-approved, and free of scope breaches. Parking would have discarded it. The work was
therefore carried onto the run branch **with this fork recorded**, on the reasoning that the run
branch is reviewed at triage *before* it reaches `dev` — so merging here does not commit the
architecture, and triage still owns the call with the work intact rather than thrown away. If that
reasoning is wrong, the remedy is to reject Track B at triage; nothing has been merged to `dev`.

**The decision.** Does decision 92 (fdw read-through) stand as the row-visibility substrate, given
this number? It should be settled **before** the RxDB replication cards build further on it.

Contributing context, all recorded elsewhere and not re-litigated here:
- The design **fails closed**, observed (G6 finding F5): an unwired environment refuses with
  `08001 could not connect`, not a calm empty set.
- SELECT policies only; writes remain deny-all, so RxDB **push** replication is refused until a
  follow-up card writes `WITH CHECK` policies. That follow-up is a separate card either way.

---

## Fork 2 — What is sales-processor told, and when? It is now TWO notices, not one.

**Card:** `app-timezone-unify-new-york` (Track A)
**Source:** G6 adversarial review, finding F1. **The reviewer REFUTED the card's provenance claim
at source.** 🛑 **This fork stays OPEN. Do not strike it.**

The card amended `21-SALES-PROCESSOR-CONTRACT.md:67` — the **published** `pending_review_ids`
expression — and justified it as *"the `COALESCE` is not new."* That justification is false:

- Phase 21's contract was archived in `875e26c` (2026-06-05 04:24). Its `:67` read
  `(created_at AT TIME ZONE 'America/Chicago')::date` with **no `COALESCE`** — and that was
  **accurate to the code as it then stood.**
- `COALESCE` entered on **2026-06-06 00:27** in `cf959bd`, a *separate quick task* `260606-0gh`
  (`.planning/quick/260606-0gh-completeness-gate-filters-pending-review/`). That task's own PLAN.md
  reasons explicitly about *"a sales-processor query for May 25–31"* — it **changed which rows the
  completeness gate returns** — and it **did not update the contract.**

**So the true history is the inverse of what the card published.** The contract was right; HQ
changed the gate's population ~8 weeks ago without telling the counterparty. The consequence is not
cosmetic: **under the published expression a late-discovered receipt did NOT block payroll; under
the shipped code it does.** Sales-processor may have been receiving an undocumented `ready:false`
since June 2026.

**The decision, which is the operator's alone:**
1. **The timezone move** (decision 93) — a coordinated release, already understood.
2. **The June 2026 completeness-gate drift** — previously undisclosed, affecting payroll gating,
   and discovered only because this card touched the document.

These are two separate notices with different urgencies and different blast radii. What
sales-processor is told, in what order, and whether anything is owed for the June-to-now window,
is not a call the run can make.

**Why this was not parked.** The reviewer noted this satisfies the card's own PARK trigger — *"the
coordinated-release sequencing needs an operator call beyond decision 93"* — but concluded it does
not warrant discarding the card, because **no HQ behavior moves here**: only a document moves, and
it moves toward behavior that shipped 8 weeks ago. The card's factual errors (F1 provenance, F2
changeover dates) are being corrected on the branch before merge. The architecture question is
recorded here for triage, with the work intact. Same reasoning as Fork 1, and equally overrulable.

🛑 **Nothing here authorizes a deploy.** Until sales-processor lands its matching change, one repo
is wrong.

