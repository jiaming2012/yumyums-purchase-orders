# Operator Brief — 2026-07-09 (CONFIRMED)

> Night-crew frozen at `e4b43ba` (branch `dev`, clean tree) for this run — the
> tool-under-test is not modified until the run closes (clean attribution). Gaps
> found tonight (PRD-verifier gate, cadence design decision) accumulate to
> night-crew's HANDOFF for a post-run batch, not a pre-run fix.

## Outcome

<!-- what is true tomorrow — not solution language -->

Tomorrow we have a written, agreed list of everything a person can actually *do*
in the Operations app — every checklist journey from a crew member opening a
template to a manager approving the finished checklist — and each item on that
list is honestly marked **working**, **broken**, or **unproven** (a test exists
but doesn't really check it). The list is precise enough that the overnight crew
can build against it without guessing, and complete enough that we'd be surprised
if a flow we use daily were missing from it.

## Why / OKR

<!-- why this, and which OKR/KR it advances -->

This is the first of the five app-hardening write-ups the cycle commits to, and
it's the example the other four copy. It advances the Product objective's first
key result (every app has a hardening PRD that enumerates its critical
end-to-end flows) and its second (we foresaw at least 90% of the flows that turn
out to matter). Operations goes first because it's the most built-out app, so its
flow list is the best-grounded and sets the pattern.

## Hard constraints

<!-- these become spec verbatim -->

- Every listed flow must be something a real person actually does, traceable from
  the screen through the API to the database — no aspirational flows.
- Each requirement points to a specific OKR key result; if it doesn't, it's out.
- Every flow carries an honest status: working / broken / unproven.
- This is hardening, not new features — make what exists fully work and be
  genuinely tested; do not add capability.
- No change to the house build: static HTML + vanilla JS front end, Go + Postgres
  back end, no framework, no new dependencies.

## Decisions made vs delegated

<!-- what you have decided vs left to PM judgment; unclaimed = delegated -->

**Decided by the operator:**
- Tonight's *attended* work covers the Operations app only — the deep exemplar.
- This write-up *lists and marks* the flows; it does not itself fix anything —
  fixes are separate work orders that follow the sign-off ("enumerate + mark only").
- **Cadence is delegated to the PjM, not hand-picked.** How many of the other
  four apps' PRDs draft in the overnight pass is the planner's call against the
  night budget + the quality bar — not an operator decision. The other four are
  overnight draft-for-sign-off, reviewed at morning triage.
- Quality bar for the exemplar = the OKR's own terms: enumeration recall ≥ 90%,
  honest per-flow status, every flow traceable frontend → API → database.

**Left to PM judgment (delegated):**
- How finely to split one "flow" from the next (the right granularity).
- Which flows count as "critical" versus nice-to-have.
- Whether any half-built corner of Operations should be treated as a mockup
  (enumerated but not counted against the working/broken tally).

## Known unknowns

<!-- what you know you do not know -->

- How many of the existing Operations tests are hollow — skipped, guarded, or
  asserting nothing. The cycle's QA count is 23 hollow tests suite-wide; the
  Operations share is unknown until we look.
- Whether any daily-used Operations flow is currently broken against a real
  database.

## References

<!-- links, each with why it matters -->

- `.night-crew/knowledge/okrs.md` — the key results every requirement must trace to.
- `CLAUDE.md` → "workflows.html Key Concepts" and "Workflows Data Persistence
  Rule" — the persistence contract every field type must honor (the seven
  persisted states).
- `docs/data-flow-audit.md` — the full state-persistence inventory.

## Out of scope

<!-- explicit non-goals -->

- The other four apps (Inventory, Onboarding, Users, Purchasing).
- Fixing the flows — this write-up enumerates and marks; work orders fix.
- Any net-new feature.
