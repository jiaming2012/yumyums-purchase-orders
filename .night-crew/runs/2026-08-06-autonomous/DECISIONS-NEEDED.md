# Decisions needed — run `overnight-20260806`

> **Two items.** **No card parked on an operator fork.** D-1 was surfaced by G6 review; D-2 by an
> incident during G6 review. Six of the seven dispatched cards completed their work (A5 was cut on
> budget and never dispatched); A3 was refused at its gate, which is a failed gate rather than a park.
>
> Raised by the orchestrator, not delegated away: everything else this run met was decided
> and stated (see HANDOFF.md §"Decided, not escalated"). This one changes what a gate
> *guarantees* for every future card, which is the operator's to settle.

---

## D-1 · What is G3, and therefore what does "that is the whole ladder" guarantee?

**Status:** open · **Raised by:** G6 on card `a4-gate-ladder` · **Marked in:** `.night-crew/knowledge/reference/gate-ladder.md`, immediately below the completeness sentence

### As a night-crew operator, I want one ruling on what G3 means, so that "a card has run every gate" is either true or visibly false — and so red-first is either gated or admittedly ungated.

Card A4 closed B-26's residual by stating in the ladder that **there is no G5** (correctly — ledger
decision 101 already retired the number, so A4 executed a standing decision rather than legislating
a gate). In doing so it wrote a completeness sentence:

> *"The ladder is G1, G2 (Go), G2 (Playwright), G3, G4, G6. That is the whole ladder. A card that
> has run those has run every gate this repo has."*

That sentence ratifies a **G3 that the record defines two incompatible ways**:

| Source | G3 is… | Citation |
|---|---|---|
| The ladder's own table row | `N/A — openspec: absent` | decision 140, `ledger.md:2697` |
| Decision 101's recovered contract | *"red-first re-verified by G6"* | decision 101, `ledger.md:1932` |

**Why this is not academic.** Red-first is a **live, graded, mandatory obligation this very run** —
Q-KR3, in its first gradeable cycle. Under A4's completeness sentence as written, red-first is **not
a gate at all**. And the run's own evidence says the obligation needs *more* enforcement, not less:
**three of the four code-changing cards shipped without their `## Red-first` section** and each had
to be sent back for it.

**Why it is yours and not mine.** Ruling either way adds or removes an obligation on every future
card and every future launch prompt. By the same logic A4's own PARK note applies to defining G5,
this is an operator call. Nothing in A4 turns on the ruling; cards that change code do.

### The options, as they would actually land

**(a) G3 means red-first, re-verified by G6.** The ladder's `N/A — openspec: absent` row is wrong
and becomes a real gate row. Red-first stops being a KR that cards forget and becomes a gate they
cannot pass without. Cost: every card now has a sixth gate to satisfy explicitly, and the launch
prompt's Q-KR3 clause becomes redundant with it.

**(b) G3 means the OpenSpec per-change gate, which this repo does not have.** The `N/A` row stands,
and red-first is an **ungated obligation** carried by the KR alone. Honest, and consistent with
decision 140 keeping HQ on branch-and-commit. Cost: the thing three of four cards forgot tonight
stays un-gated, resting on each card remembering it.

**(c) Split them.** Keep G3 as `N/A — openspec: absent` and give red-first its **own name** rather
than a recycled number — the ladder already says a real fifth gate would get "a name, not the
number 5". Costs one new gate row; retires the ambiguity permanently.

### Orchestrator's recommendation
**(c)**, then **(a)** as second choice. Tonight's evidence is that red-first fails precisely because
it is an obligation with no gate behind it, and a recycled number is what produced this
contradiction in the first place. But this is a recommendation, not a decision — I have not applied
it, and A4's sentence is marked open rather than softened.

---

## D-2 · The production posture — a test suite holds admin credentials to the production cluster, and nothing is backed up

**Status:** open · **Raised by:** realised during G6 review of `a3-rls-fixture-own` · **Bugs:** B-141, B-143

### As the owner of a working food-truck business, I want to know that a test run cannot delete my operating data — and that if something does, I can get it back.

Tonight a test suite dropped the production database. You have already made the immediate call
(restart, rebuild empty) and prod is back up. These are the two structural questions it leaves,
and both are yours because both trade cost against risk in ways only you can price.

**(a) Should the test suites and production share one Postgres cluster?**
`defaultHQAdminURL` in the RLS suite is `postgres://yumyums:yumyums@localhost:5433/postgres` — the
same cluster that serves `hq.yumyums.kitchen`. That means *any* mistake in that file is a
production mistake, and B-141's guard defect was only the first one to find the edge. Separating
them (a second container, or a distinct role without `DROP DATABASE`) would have made tonight
survivable **independently of any backup**. Cost: another container to run and one more DSN to keep
straight.

**(b) What backup do you want?** Verified tonight: no PITR (`archive_mode=off`), no dump target
anywhere in the repo, no dump files on the box, and the alternate volume empty. `task prod:deploy`
ships a `prod:rollback` for the **image** and nothing for the **data** — which is exactly why the
image was reversible tonight and the database was not.

- **Floor:** a nightly `pg_dump` of `yumyums` to a path outside the Docker volume. A Taskfile target
  plus a cron line. Would have made tonight a twenty-minute restore.
- **Ceiling worth having:** `archive_mode=on` with a local WAL archive, giving point-in-time
  recovery to the second before the `DROP`.

**Orchestrator's recommendation:** do **(b) floor** immediately — it is an hour of work and it is
the difference between last night and a shrug — and treat **(a)** as the real fix to schedule. I
have not implemented either; a run has no business changing production posture, and you have just
had one demonstration of why.

---

## Not decisions — recorded so triage does not mistake them for open forks

These were **decided and stated** during the run, per the standing rule that role-level calls get
decided rather than handed up. Listed here only so the morning reader can see them and object:

| Call | Decision | Where |
|---|---|---|
| `card/d1-syncspec-deflake` flagged as stranded work at launch | Proven net-zero (tip tree byte-identical to merge-base); not a launch gate | HANDOFF §Launch |
| A5 `shipped-bug-sweep` (budget-gated stretch) | **Cut.** Its estimate + the ~30m closeout was not in hand | HANDOFF §Budget |
| A1's Playwright gate, overlapped by an unlocked Go suite | **Discarded and re-run**, not reasoned about | `conflicts-20260806.md` §3 |
| Track B dispatched A4 before A2 merged | Files provably disjoint; stated with its reason | HANDOFF §Deviations |
| `hq_test_go` corruption repair | Deferred past the run rather than dropping a shared DB mid-flight | HANDOFF §Next actions |
