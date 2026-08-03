# Preferences — architecture

> Paradigms, service boundaries, storage and messaging choices.
>
> Weighted opinions, not binding rules: a session may deviate, but must say why.
> Add entries as `P-1`, `P-2`, … in the shape of the commented example below —
> `night-crew preferences validate` checks that shape, never your judgment.

<!--
## P-1 · A short title naming the leaning

- **Preference:** what to do, stated so a session can act on it.
- **Why (operator):** the reasoning, in your own words — this is what gets cited back to you.
- **Weight:** strong — and then whatever qualifier you want (start with strong, moderate, or weak).
- **Evidence:** optional link to a research note or a past decision.
- **Recorded:** 2026-01-01
-->

## Pending — proposed, not adopted

> Candidates offered back from your own answers, recorded with your consent.
> They are **not preferences yet**: nothing cites them, nothing validates them, and no
> command promotes them. To adopt one, renumber it to the next free `P-n`, move it up
> above this section, and delete what you don't want. To drop one, delete it.

## C-1 · The app's timezone is America/New_York

- **Preference:** Every date and time boundary the app computes — submission "today", the purchasing week, the COGS/payroll period, recipe cost weeks, scheduled jobs — resolves in America/New_York. Never UTC, never a hardcoded America/Chicago, never the device's local zone. Where a stored default exists it mirrors users.DefaultTimezone rather than restating a literal.
- **Why (operator):** The apps time zone should be NY time.
- **Weight:** strong
- **Evidence:** ledger T-26 decision 83; card app-timezone-unify-new-york. Found because the codebase was running two conflicting regimes — New York in the user-facing defaults, America/Chicago in the COGS completeness gate, CurrentWeekStart, the recipe cost week, and two migration column defaults.
- **Recorded:** 2026-07-28
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.

## C-2 · Fetch in batches whenever a list can grow unbounded

- **Preference:** Any client-side fetch or replication over a collection that can grow without bound is batched and scoped — never pulled whole. Scope it to what the current view actually needs (for workflows, the open checklist) and page the remainder. This binds RxDB replication, API list endpoints, and any future sync layer: a full-collection pull is a design defect, not a default. Widening the scope requires a recorded decision.
- **Why (operator):** fetching should always be done in batches whenever it is seen that a list could grow unbounded
- **Weight:** strong
- **Evidence:** ledger T-29 decision 105 (overnight-20260801 morning triage) — startHQReplication replicates four collections with no selector at pull batchSize 50; measured ~23 ms/row through the fdw RLS path, ~23 s for a 1,000-row pull, and unbounded phone storage since `responses` grows forever
- **Recorded:** 2026-07-31
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.

## C-3 · Never cut a write path over to a store the readers don't read

- **Preference:** A write path may only move to a different datastore once a proven path carries those rows back to every reader that depends on them — reports, payroll, submit, approvals. Where no such path exists, split reads from writes: the new store may serve reads while the existing path keeps owning writes. Do not swap the write path and rely on local persistence to make it look correct. A done_when: asserting "the value survives reload" does not prove the write landed where readers look, and will pass while data is being lost.
- **Why (operator):** Chose "read on sync, write on REST" at morning triage 2026-08-02 — taking the read/write split, and waiving P-KR3's parallel-run prohibition to get it, rather than accept a cutover that could produce a silently empty submit. (Selected from offered options; no additional reason stated.)
- **Weight:** strong
- **Evidence:** run 20260803, card S1b sync-hard-cutover PARKED. RxDB replicates to the self-hosted Supabase substrate; /submit reads HQ's Postgres; the 0002 bridge is HQ→substrate, read-only, and carries permissions not data. Reopens ledger decision 49, whose deciding premise ("no API boundary left to translate at") is false as built.
- **Operator:** jac475@cornell.edu
- **Recorded:** 2026-08-02
- **Offered at:** an attended session
- **Consent:** recorded on the operator's explicit yes to this item.
