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
`HQ_SYNC_REST_URL` must still not be set in any deploy. ~~The four B1 tables remain
RLS-enabled with zero policies (deny-all), which is exactly where B1 left them.~~
**STRUCK as imprecise (repair round, same family as park-note C-4): that was a claim
about a live database that had not been measured.** Corrected: `0001_sync_tables.sql`
declares the four tables RLS-enabled with **zero `CREATE POLICY`** — the only
occurrence of that phrase in the file is a comment asserting the absence, and
`tests/sync-schema.spec.js` guards it. **Live, the four tables are not present in the
substrate at all** (queried on 46011 this round: `NONE_PRESENT`), because B1's SQL has
not been applied anywhere. Deny-all-by-declaration, unapplied in fact. Either way this
card weakened nothing.
The door is still shut; this card did not open it and did not half-open it.

---

## 🔧 Repair round — corrections applied after G6 verification

G6 independently re-executed every probe on both servers and found the park
**CORRECT**. **The verdict is unchanged: PARKED.** It also found four inaccuracies in
the park note, which have been applied there and are listed here so the merger sees
them from this note too:

| | Correction | Where it landed |
|---|---|---|
| C-1 | `postgres_fdw`/`dblink` are **installable at both ends**, not merely "available, uninstalled" at one — and HQ was never measured. Re-measured here by executing `CREATE EXTENSION` on **both** servers in rolled-back transactions. Changes the cost of fork option (a) from two obstacles to one. | park note §3 (E4 struck, E4a–E4c added), §5 option (a) |
| C-2 | Native logical replication added to the fork as **option (b′)** — crash-safe, zero code in the mutation path, but still asynchronous and therefore in the same bounded-lag risk class as (b). | park note §5 |
| C-3 | The note silently substituted `template_assignments` for the `app_permissions` the card and decision 61 name. Conclusion unchanged (identical topology, measured); but **decision 61's literal text may have been written about a different table.** Flagged for the operator, not decided. | park note §2b |
| C-4 | The claim that the shared spike stack was "read only" was **false** and is struck in place, with what was actually written and by what. | park note §6 |

**This note was re-read whole (B-11).** The one line in it that the repair round
contradicts is struck above. The false read-only claim lives in the park note, not
here, and is struck there.

**Disclosure the merger should have from this note directly:** the shared spike
Supabase stack at `127.0.0.1:46011` **was written to** during this card's work —
by `TestJWTBridgeRLS`'s own fixture and policy files (which it applies on every run),
and by this round's rolled-back `CREATE EXTENSION` probe. No database was created or
dropped; nothing persisted from the probe; B1's four tables were never reached. Full
account in park note §6.

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

**Corroboration added in the repair round — G6 found it, and this note had asserted
the point without demonstrating it.** G6's independent verdict is that the no-red
decision was **RIGHT**, and the repo backs it directly: `hq_grant_projection` is
*written* from exactly two places repo-wide — `jwtbridge_rls_test.go` (`:512`, `:518`,
`:590`, `:595`) and `hq-bridge-fixture.sql` (`:153`). A test and a fixture. Every
other occurrence is a select, a policy, a proxy route mapping, or prose. **There is no
push-on-change writer anywhere in this repo to imitate.** The pattern decision 61
assumes exists in production exists only in the spike — which is why a green suite
here would have proved something about a fixture, and why its absence is the honest
outcome rather than a gap.

**On the 16/16 above.** G6 could not confirm the original run executed
`TestJWTBridgeRLS` rather than reporting a green banked by the 2026-07-26 card B run
— `hq_grant_projection` held 2 rows beforehand, which either explains. The ambiguity
is not resolvable after the fact, so the record is replaced with a first-hand one:
the repair round ran `go test ./internal/sync/ -run TestJWTBridgeRLS -v -count=1` at
**2026-07-29 02:02:20–02:02:23 UTC** — **PASS, 16/16 subtests**, both `service_role`
BYPASSRLS controls green. Nothing in this branch depends on that green; it is stated
so the record is unambiguous. See park note §7a.
