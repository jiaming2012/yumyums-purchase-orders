# Merge intent — Card B2 `sync-rxdb-row-visibility-rls`

Run `overnight-20260729-2`, branch `card/b2-sync-rxdb-row-visibility-rls`,
cut from `overnight-20260729-2` (`95a2657`, B1 merged).

Written BEFORE implementing, per the run's mandatory mechanics. If a repair round
contradicts anything below, the contradicted line is struck through in place —
not merely appended to (B-11).

---

## 🛑 OUTCOME: PARKED. The card's own PARK trigger fired during orientation.

**This branch contains NO production code, NO SQL, NO policies, and NO migration.**
It contains two documents: this note and the park note beside it.

The card's park trigger, verbatim:

> if porting `ResolveEntityAccess` reveals that the projection **cannot** be
> written in the same transaction as the `app_permissions` mutation without
> restructuring that mutation, **PARK**.

It fired. The projection must live in the **sync substrate's Postgres** (where the
RLS policies evaluate); every mutation that would keep it live runs in **HQ's
Postgres**. These are two separate database servers, a fact established by this
card's own dependency B1 (merged tonight). A write to one cannot participate in a
transaction on the other. Full evidence, measured not assumed, is in
`park-b2-sync-rxdb-row-visibility-rls.md`.

Consequence for the merger: **nothing about the activation interlock changes.**
`HQ_SYNC_REST_URL` must still not be set in any deploy. The four B1 tables remain
RLS-enabled with zero policies (deny-all), which is exactly where B1 left them.
The door is still shut; this card did not open it and did not half-open it.

---

## Shared files touched (files that exist on the base branch)

**Nothing here.** This branch adds two new files under
`.night-crew/runs/2026-07-29-2-autonomous/` and edits nothing that exists on the
base branch.

Specifically **NOT** touched, each because the park means the work was not done:

- `.night-crew/knowledge/roadmap.md` — **no status flip.** The card's bullet stays
  `PLANNED — SLATE-READY`. A parked card that flipped itself to DONE would tell the
  3am merger the door is closable when it is not. The roadmap edit this card owes
  is the *park disposition*, and that is morning triage's to write with the
  operator, not mine to assert unattended.
- `backend/internal/db/migrations/` — **NO MIGRATION WAS CREATED.** The run's known
  collision on `0072` is unaffected: **A1 takes `0072` uncontested.** There is
  nothing here to renumber at merge.
- `sync-schema/sql/` — no second SQL file. B1's `0001_sync_tables.sql` is untouched,
  and its deny-all state is intact.
- `backend/internal/sync/` — no Go file added or edited. In particular
  `ops.go:474` `ResolveEntityAccess` is **unchanged**: it was read, not modified.
  No attack-variant test file was committed (see "red-first" below).
- `package.json`, `version.go`, `sw.js`, any `.html` — untouched. No version bump is
  owed because no shipped surface changed.
- Nothing in any deploy path, compose file, or example env. **`HQ_SYNC_REST_URL` is
  not set, referenced, or added anywhere by this branch.**

## What must survive any merge

1. **The park note itself** (`park-b2-sync-rxdb-row-visibility-rls.md`). It is the
   card's entire deliverable and it carries the fork morning triage has to resolve.
   If a conflict ever puts it at risk, keep it.
2. **The `roadmap.md` bullet staying un-flipped.** If some other card's merge
   incidentally flips it, that is wrong — revert the flip.
3. **The `0072` migration number staying with A1.**

## What is safe to drop

This note, once the park note has been folded into `DECISIONS-NEEDED.md` and the
run's `HANDOFF.md` carries the outcome. It is bookkeeping about a branch with no
code in it. The park note is not safe to drop; this one is.

## Red-first evidence — stated plainly, because its absence is deliberate

The card's gate is a red-first attack-variant suite. **No such suite is committed,
and that is the honest outcome of the park, not a skipped step.**

A suite could have been written and made green: create the projection tables, seed
them by hand in a fixture the way `hq-bridge-fixture.sql` seeds
`hq_grant_projection`, write the policies, and watch permitted principals admitted
and forbidden ones refused. It would have looked exactly like the 16/16 the JWT
bridge card banked.

It would also have been a proof about a table **nothing in production writes**. The
premise the whole proof rests on — that the projection is live — is precisely the
thing the park says cannot be built tonight. On a run whose first card exists
because a suite that runs nothing reports `ok`, shipping a security proof whose
premise is fictional is the same defect one level up. So it was not shipped.
