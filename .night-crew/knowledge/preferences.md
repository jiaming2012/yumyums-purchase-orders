# Preferences — architecture preferences

> Scaffolded by `night-crew init`. Operator-owned and weighted, not binding.
> Replace this with the real preferences for this repo before the first evening.

- TODO: record architecture preferences — paradigms, libraries, and patterns to
  prefer or avoid — that night-crew sessions should weigh when designing work.

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
