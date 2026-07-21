# Preferences — architecture preferences

> Scaffolded by `night-crew init`. Operator-owned and weighted, not binding.
> Replace this with the real preferences for this repo before the first evening.

## Test isolation and determinism — STRONG (stated by the operator 2026-07-21)

Three preferences stated verbatim during `overnight-20260720c` triage. Phrased as
"almost always my preference," so treat them as **strong defaults that require a stated
reason to depart from** — not as absolute rules.

### P1 — Run on a clean DB with fixtures

> *"It's almost always my preference to run on a clean DB with fixtures in order to
> preserve determinism (unless there is a specific reason not to do so)."*

**Applies to:** every suite entry point, and any gate that makes a suite-green claim.

**Why it was stated.** `tests/sync.spec.js:525` (`FLD-LIVE-02`) looked order-dependent and
was promoted to a flake-fix card. It is neither flaky nor order-dependent. The `ops` table
is an append-only sync journal that never shrinks; `task test` drops and recreates
`hq_test` but a bare `npx playwright test` does not. So the journal accumulates **across
runs**, and the test passes at 98 ops and fails at 614+. The failure is a function of how
many times the suite has run since the last clean.

**Consequence for the `cycle-gate` card:** its DB precondition is now decided — **it runs
on a clean DB.** A gate that resets and a gate that doesn't are measuring different
systems, and only the clean one is reproducible. This resolves DECISIONS-NEEDED §0b.

**Standing implication:** a suite entry point that does *not* reset is a defect in the
entry point, not a quirk to be worked around in the tests. If a test needs accumulated
state, it seeds it as a fixture rather than inheriting it from history.

### P2 — Separate schemas; never cross-contaminate

> *"It's almost always my preference to use separate schemas so as to not cross
> contaminate (e.g. separate schemas for dev, qa, etc); this should be applied
> everywhere — investigate any surfaces where cross contamination is possible."*

**Applies to:** databases and schemas first, but the operator's intent is broader —
**any shared mutable resource across environments, runs, suites, or sessions.**

Known contamination surfaces at time of writing (full audit:
`reference/cross-contamination-surfaces-20260721.md`):

- **DB/schema separation** between dev, test, and any QA environment.
- **Append-only tables** that survive resets — `ops` is the confirmed case; audit for others.
- **Client-side persistence** — IndexedDB, localStorage, service-worker caches, persisted
  Lamport clocks.
- **Ports and containers** — the `:8199` latch has recurred across at least three runs;
  `yumyums-e2e-pg` is shared between concurrent sessions.
- **The working tree itself.** Confirmed live on 2026-07-21: five Claude Code processes
  held `cwd=/home/jcole/projects/hq`, one alive five days, and one wrote to the tree while
  believing it was on `dev` when `overnight-20260720c` was actually checked out. One
  checkout, one git index, many writers. Worktrees additionally share `refs/stash` — which
  is why `git stash` is already prohibited in a worktree here.

**Standing implication for run mechanics:** "the box is quiet" is a claim to be *measured*,
never assumed. Serial dispatch within one run does not make the machine idle.

### P3 — Unreproducible means resolved, until it recurs

> *"If an issue cannot be reproduced, then mark it as resolved until / unless it appears
> again."*

**Applies to:** any flake, intermittent failure, or defect report that survives a genuine
reproduction attempt without reproducing.

**Why it was stated.** Two separate reports consumed real capacity on this basis:
`sync.spec.js:1198` was promoted into a slate as *"proven flaky — red 1-of-2 legs"* and
then produced ~20 consecutive greens across two independent parties; `workflows.spec.js:2223`
(`RUN-10`) was reported as a gate blocker and did not reproduce at all.

**How to apply it honestly.** Close it, and record: what was attempted, under what
conditions, and what failure rate the attempt bounds. A streak **bounds** a rate; it never
proves zero. Distinguish:

- **"Not flaky"** — no mechanism, no reproduction. Close it.
- **"Rare, mechanism understood, bounded at N%"** — a real mechanism is identified but
  fires rarely. This is *not* the same as resolved, and must not be laundered into it.

Reopening on recurrence is expected and is not a process failure. Carrying an
unreproducible claim forward as established fact **is**.

---

- TODO: record the remaining architecture preferences — paradigms, libraries, and patterns
  to prefer or avoid — that night-crew sessions should weigh when designing work.

## Artifact naming — BINDING (adopted 2026-07-20)

**Rule: real authoring date + a mandatory intra-day cycle letter.**

```
slate-YYYYMMDD<letter>.md          e.g. slate-20260720c.md
overnight-YYYYMMDD<letter>         e.g. overnight-20260720c
runs/YYYY-MM-DD<letter>-autonomous e.g. runs/2026-07-20c-autonomous
```

- `YYYYMMDD` is the **real calendar date the slate is authored**, taken from the
  system clock — never "tomorrow," never inferred from the previous artifact's name.
- The **cycle letter is mandatory even for the first cycle of a day** (`a`, then `b`,
  `c`, …). It is what guarantees a new artifact can never collide with a legacy label.
- **Every date appearing in prose is the real calendar date**, always — sign-off dates,
  triage dates, decision dates. Labels live in filenames and branch names only.

### Why — the drift this replaced

The prior rule named each slate for "the morning after," which silently assumes **one
run per night**. The actual cadence is several cycles per real day, and each new slate
took the next day-number, so labels advanced one per *cycle* while the calendar advanced
one per *day*. Measured 2026-07-20: labels had ratcheted **+3 days ahead of reality** —
`slate-20260721`, `slate-20260722`, and what would have been `slate-20260723` were all
authored on 2026-07-20 (08:19, 16:26, 22:18).

The damage was not the filenames; it was **labels leaking into prose that reads as
factual history**. Commit `b5f3952` is titled "morning triage 2026-07-22" but was authored
2026-07-20 21:56, while the HANDOFF body it committed correctly says "Triaged 2026-07-20"
— two date sources inside one artifact. A slate's sign-off line is a factual claim about
when the operator consented; under the old rule it was routinely off by days.

### Legacy artifacts

Labels `slate-20260712` … `slate-20260722` (and their `overnight-*` branches and `runs/`
directories) are **future-dated and are left as-is** — they are load-bearing cross-references
across the ledger, HANDOFFs, and every prior slate, and rewriting them would corrupt a record
whose only value is reliability. Treat any pre-2026-07-20c label as an **opaque identifier,
not a date**; to date a legacy artifact, read its git author date.

The one-time consequence: `slate-20260720c` sorts *before* legacy files it postdates. The
overlap ends once the real calendar passes 2026-07-22.

### Follow-up not yet done

`~/.claude/skills/nc-slate-plan/SKILL.md` still documents the old rule
("`reference/slate-YYYYMMDD.md` — dated for the MORNING after"). That skill is **user-level
and shared across every night-crew target repo**, so it was deliberately not edited mid-ritual.
Update it between cycles, not during one.
