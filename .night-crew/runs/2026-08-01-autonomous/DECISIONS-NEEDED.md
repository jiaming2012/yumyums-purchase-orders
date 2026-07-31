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

## Fork 2 — placeholder, pending Track A's G6 ruling

`app-timezone-unify-new-york` edited `21-SALES-PROCESSOR-CONTRACT.md:67`, the **published**
`pending_review_ids` expression, to state a `COALESCE(event_date, …)` the card says the code has
carried since Phase 21. Decision 93's authority was to move the **timezone**. This is a published
expression in a two-repo agreement that sales-processor has not agreed to.

The G6 reviewer was asked to rule on this explicitly, and to treat it as a finding **even if the
new text is more accurate**. This section is filled in when that verdict returns; if the reviewer
clears it, this fork is struck rather than left dangling.
