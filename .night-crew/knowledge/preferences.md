# Preferences — SUPERSEDED POINTER + retained appendix

> 🛑 **This file is NOT read by any tooling, and has not been since the categorized
> preferences folder landed.** night-crew's scaffolder says so itself
> (`internal/onboarding/onboarding.go:83-87`: the folder *"replaces the flat
> preferences.md earlier versions scaffolded; a repo that already has that file keeps it
> — it is simply no longer read"*). It survived here only because agents opened it by
> path. Migrated at morning triage **2026-07-25** (ledger T-22); `night-crew preferences
> validate` now passes on the folder.
>
> **The live store is `.night-crew/knowledge/preferences/`.** Read it with
> `night-crew preferences list --repo .`.

## Where each rule went

| Was here as | Now lives at | Notes |
|---|---|---|
| §"Test isolation and determinism" P1 | `preferences/process.md` **P-1** | verbatim operator quote retained in **Why** |
| §"Test isolation and determinism" P2 | `preferences/process.md` **P-2** | verbatim operator quote retained in **Why** |
| §"Test isolation and determinism" P3 | `preferences/process.md` **P-3** | verbatim operator quote retained in **Why** |
| §"Test isolation and determinism" P3a | `preferences/process.md` **P-4** | promoted from a corollary to its own entry — `P3a` is not a valid id under the grammar |
| §"Artifact naming" (BINDING) | `preferences/constitutional.md` **P-6** | **placed on the constitutional list with `Threshold: always`**, not flattened to an ordinary `strong` leaning. The rubric accepts only `strong`/`moderate`/`weak` as the comparable keyword, so `BINDING` could not survive as a weight; the constitutional list is the store's actual mechanism for a rule that always binds. Operator-approved at the 2026-07-25 triage. |

> **Historical citations are why this file is not deleted.** At least ten slates, handoffs
> and run records cite `preferences.md §"Artifact naming"` or `preferences.md P2/P3a` by
> path. Those are immutable records; the pointer above is how a reader follows them.

---

# Retained appendix — rationale and legacy-artifact rules

> These three sections have **no home in the entry grammar** (which is `Preference` /
> `Why` / `Weight` / `Recorded`) and are retained here in full. The legacy-artifact
> reading rule in particular is still operationally live: **treat any pre-2026-07-22
> label as an opaque identifier, not a date.** `constitutional.md` P-6 cites this
> appendix as its Evidence.

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
