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

**P3a — a reproduction attempt must sample the RIGHT CONDITION, or it proves nothing.**
Added the same day, because this failed immediately and instructively.

`sync.spec.js:1198` was declared "decisively refuted" on ~20 consecutive greens from two
independent parties. Re-run **under a concurrent Playwright suite**, it produced **4 reds in
25 legs (16%; 20% under contention)**. Both earlier parties had sampled the *quiet*
condition — and ~20 greens off-condition is entirely consistent with a 20% on-condition rate
(p ≈ 1.2%).

The failed reasoning is worth naming because it looks rigorous: a p-value was computed
against a **50% unconditional** rate nobody had claimed, using samples drawn from a
condition where the bug does not fire. The arithmetic was correct and the conclusion was
wrong.

**So before closing anything under P3, state the condition you sampled** — load, concurrency,
data volume, journal depth — and whether it is the condition the original report described.
"Could not reproduce **on a quiet box**" is a finding. "Could not reproduce" is not.

Corollary: absence of reproduction in N legs bounds a rate *conditional on the sampled
condition only*. 5/5 green at low load bounds nothing about high load (p(0 red in 5 | 20%)
≈ 0.33).

---

- TODO: record the remaining architecture preferences — paradigms, libraries, and patterns
  to prefer or avoid — that night-crew sessions should weigh when designing work.

## Artifact naming — BINDING (fleet-standard numeric suffix, adopted 2026-07-22; supersedes the 2026-07-20 cycle-letter rule)

**Rule: real authoring date + a NUMERIC collision suffix (fleet standard).**

```
slate-YYYYMMDD.md            slate-YYYYMMDD-2.md          e.g. slate-20260722.md, slate-20260722-2.md
overnight-YYYYMMDD           overnight-YYYYMMDD-2         e.g. overnight-20260722, overnight-20260722-2
runs/YYYY-MM-DD-autonomous   runs/YYYY-MM-DD-2-autonomous
```

- `YYYYMMDD` is the **real calendar date the slate is authored**, taken from the
  system clock — never "tomorrow," never inferred from the previous artifact's name.
- The **first run of a real date carries NO suffix**; the second is `-2`, the third
  `-3`, … A numeric suffix is a collision counter for that date, nothing more.
- **Sort run-ids with `sort -V`** (version sort), never plain `sort` — plain sort
  orders `-10` before `-2`, and puts an unsuffixed run after its own `-2`.
- **Every date appearing in prose is the real calendar date**, always — sign-off dates,
  triage dates, decision dates. Labels live in filenames and branch names only.

**Source of the standard.** This is night-crew's own run-id convention, defined by the
`fix-overnight-ergonomics` change and matched by its tooling as
`^overnight-[0-9]{8}(-[0-9]+)?$` (see the dev skills `nc-status`, `nc-morning-triage`,
`nc-slate-plan`, `nc-run`). hq conforms to it so the fleet tooling reads hq's runs
instead of skipping them. The **cycle-letter rule is retired** — it was an hq-local
invention that the fleet matcher never recognised, which is precisely why
`overnight-20260720c` was skipped by `/nc-status`.

> **⚠ DEPLOYMENT CAVEAT — the numeric matcher is NOT yet on night-crew `main`.** Verified
> 2026-07-22: `main`'s `nc-status` and the **installed** `~/.claude/skills/nc-status`
> both still carry the OLD matcher `^overnight-[0-9]+$`, which skips a `-N` suffix just
> as it skipped a letter (the `-` breaks `[0-9]+$`). The `fix-overnight-ergonomics`
> change is archived on night-crew **dev** and is NOT an ancestor of `main`. So adopting
> this rule makes hq *conformant and drift-proof going forward*, but it does **not** make
> the deployed tooling read hq's runs until that change reaches `main` AND the user-level
> skills are re-synced. Per [[nc-tooling-tracks-main]], hq rituals track main — so this
> rule is the go-forward convention, and the tooling upgrade is the separate, blocking
> half. Do not expect `/nc-status` to see numeric hq runs until then.

### Why — the drift the *letter* rule replaced (retained; still true)

The rule before the letter rule named each slate for "the morning after," assuming **one
run per night**. Cadence is several cycles per real day, so labels advanced one per
*cycle* while the calendar advanced one per *day*. Measured 2026-07-20: labels had
ratcheted **+3 days ahead of reality** — `slate-20260721`, `slate-20260722`, and what
would have been `slate-20260723` were all authored on 2026-07-20 (08:19, 16:26, 22:18).
The letter rule fixed the drift; the numeric rule keeps that fix and additionally
conforms to the fleet matcher. The damage was never the filenames — it was **labels
leaking into prose that reads as factual history** (commit `b5f3952` titled "morning
triage 2026-07-22" but authored 2026-07-20). A sign-off line is a factual claim about
when the operator consented; keep it the real date.

### Legacy artifacts — INCLUDING the sole letter artifact, left as opaque identifiers

Two generations of pre-conformance labels exist and are **all left as-is**:

1. Future-dated morning-after labels `slate-20260712` … `slate-20260722` (and their
   `overnight-*` branches and `runs/` dirs).
2. The **one** cycle-letter artifact: `slate-20260720c.md`, branch `overnight-20260720c`,
   `runs/2026-07-20c-autonomous/`.

**None are renamed.** They are load-bearing cross-references across the ledger, HANDOFFs,
and every prior slate — and `20260720c` is additionally baked into **5 pushed, immutable
commit messages** (`c2cfc13` merge, `771a0da` T-20, `bcd5ed0` closeout, two sign-offs) and
an **already-merged** branch. Renaming the file but not the commits would recreate the exact
split-identity corruption this whole convention exists to prevent — for zero tooling benefit,
since the deployed matcher skips `-N` too (see the caveat above). Treat any
pre-2026-07-22 label as an **opaque identifier, not a date**; to date one, read its git
author date.

**For the record (mapping, not applied):** by run order, 2026-07-20 ran three cycles —
cycle 1 = `overnight-20260721`, cycle 2 = `overnight-20260722` (both future-dated legacy),
cycle 3 = `overnight-20260720c` (the slate says so: *"the third cycle of 2026-07-20"*).
Under the numeric rule cycle 3 would be `overnight-20260720-3` — but there is **no**
`-1`/`-2` sharing that base (they are the mislabelled `20260721`/`20260722`), so a bare
`-3` would itself be misleading. This is a second reason the mapping is documented rather
than applied. There is no plain `overnight-20260720`.

### Follow-up — the blocking half (night-crew side, NOT hq)

The goal "fleet tooling reads hq's runs" needs the **night-crew** side, not more hq
renames: (a) promote `fix-overnight-ergonomics` to night-crew `main`, and (b) re-sync the
user-level `~/.claude/skills/nc-*` so the installed matcher becomes
`^overnight-[0-9]{8}(-[0-9]+)?$`. Until both land, deployed `/nc-status` skips every hq
run regardless of its suffix form. The user-level skills are **shared across every
night-crew target repo**, so they are not edited from inside an hq ritual.
