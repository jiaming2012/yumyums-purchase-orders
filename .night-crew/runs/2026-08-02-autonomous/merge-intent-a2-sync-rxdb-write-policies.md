# Merge intent — A2 `sync-rxdb-write-policies`

Run `20260802`, Track A, branch `card/a2-sync-rxdb-write-policies`, cut off
`overnight-20260802` **after A1 merged** (base `2dc4eef`).

Authority: ledger **T-30 decision 111** (`ledger.md:2171`), whose four-row table
is the complete signed specification of what write predicates exist. The card
also carries **B-36** (the `internal/sync` silent-skip), folded in by decision
111's own "Also folded" paragraph.

## Shared files touched

| File | Why it is outside my own module |
|---|---|
| `backend/internal/db/migrations/0074_sync_fdw_approver_view.sql` | **NEW, and the one line of this note to read first.** A goose migration under HQ's own migration directory — outside "substrate-side policies plus Go tests". It creates ONE read-only VIEW (`hq_sync_template_approvers`) and ONE grant. It alters **no table, no column, no constraint**. See "The migration, and why it is not the park trigger" below. **Migration number 0074 was verified free on the merged run branch** (`git ls-tree overnight-20260802 -- backend/internal/db/migrations` tops out at 0073). |
| `sync-schema/sql/0002_hq_fdw.sql` | §3d — a FOURTH foreign table (`hq_template_approvers`) over that view, and its entry on §4's revoke list. 0002 is B2's file; I append one `if to_regclass(...) is null` block in the existing `DO $$` and one `revoke all` line. **I change no existing foreign table.** In particular `hq_template_assignees` stays byte-identical: §3a's stated property — "does NOT carry `assignment_role` … so it cannot be filtered on by accident" — is preserved literally, which is why the approver arm is a separate relation rather than a widened one. |
| `.night-crew/knowledge/roadmap.md` | The status flip required in the same change set. **I touch ONE bullet only** — `sync-rxdb-write-policies` (roadmap line ~896). Every other bullet is another card's. |
| `backend/internal/sync/jwtbridge_rls_test.go` | **ADDED BY THE G6 FIX ROUND.** `do` is split into `do` (unchanged default, `return=representation`) + `doPrefer`, and the two `prefer*` constants are added. This is the F1 seam — see "what must survive" item 10. It is shared with any card touching `internal/sync` tonight; the change is additive and no existing call site's behaviour moves. |
| `.night-crew/knowledge/BACKLOG.md` | Scope-freeze destination. Append-only, at the end of the file, numbers **B-47, B-48, B-49, B-51** — ~~B-46+~~ my B-46 was RENUMBERED TO **B-49** at triage because the B1 leg merged first and kept B-46, and **B-50 is reserved** for the substrate-isolation finding. B-48 is re-specified in place by the G6 fix round (its original lead guarded a non-defect); B-51 is new (finding F4). Nothing else existing is edited. |
| `sw.js` / `version.json` | G4 regeneration only (`node build-sw.js`), run AFTER the last content commit because the manifest globs `git ls-tree HEAD`. Mechanical; take whichever side is regenerated last and re-run `node build-sw.js`. |
| `.night-crew/runs/2026-08-02-autonomous/merge-intent-a2-sync-rxdb-write-policies.md` | This note. Uniquely named for this card. |

Everything else is inside the declared footprint: `sync-schema/sql/0004_write_policies.sql`
(new), `backend/internal/sync/rowvisibility_rls_test.go`,
`backend/internal/sync/spikestack_gate_test.go`.

**No HQ table changes. No backend Go source changes** — `backend/internal/sync/*.go`
non-test files are byte-unchanged. `HQ_SYNC_REST_URL` is not set, referenced or
implied anywhere in this change set; the interlock stays armed.

## The migration, and why it is not the park trigger

The roadmap bullet's parenthetical park trigger reads *"a schema change to HQ's
own tables … it must not need a `backend/migrations/` file"*. **The signed slate
narrowed the park conditions to exactly one**: *"PARK if: a write predicate is
needed beyond decision 111's four rows (operator-only)."* No such predicate was
needed — every policy below is one of decision 111's four rows.

The migration exists because decision 111's **own** consequence (2) names
`public.hq_can_approve_template(tid)` = `EXISTS` an assignment with
`assignment_role = 'approver'` **OR** the admin arm — and `assignment_role` does
not cross the FDW today, by 0002 §3a's deliberate design. The predicate the
operator signed is not evaluable on the substrate without it. This is plumbing
for a decided product rule, not a product fork: a read-only view, no table
touched, the same shape as its sibling 0073, and reads keep INHERITED PROPERTY 1
byte-for-byte because the new relation contains **only approver rows** and the
read path never names it.

🛑 **0073's banner lists the columns a future migration may no longer retype.
0074 ADDS ONE: `template_assignments.assignment_role`.** That is stated in 0074
itself; it is the standing cost of the operator's answer.

## What must survive any merge

1. 🛑 **`sync-schema/sql/0004_write_policies.sql` is decision 111's four rows and
   nothing else.** Row 1 (`checklist_templates`) is **an absence on purpose** —
   the file contains no INSERT/UPDATE policy for it, and says so at length. **A
   merge that "completes the set" by adding one has reversed a signed operator
   decision**, and it will look like tidying, because the other three tables all
   have one. The refusal is asserted by variants W1–W3, not left as a gap.
2. **`checklist_submissions` WITH CHECK is `hq_can_see_template(template_id)`,
   FIELD-for-template on the same axis as the read.** ~~and the UPDATE policy
   carries BOTH `using` and `with check`. Dropping the `with check` half of the
   UPDATE leaves W5 open: `using` alone lets a client move a row it can see INTO
   a template it cannot. That is the whole reason Postgres has two clauses, and
   W5 is the variant that proves it.~~

   🛑 **STRUCK — G6 finding F3. The struck sentence was wrong about PostgreSQL
   and a merge must not act on it.** Postgres SUBSTITUTES the `using` expression
   for an omitted `with check`, and for an UPDATE it applies the table's SELECT
   policy to the new row whether or not the statement returns anything. On this
   policy `using`, `with check` and 0003's SELECT predicate are all the SAME
   expression, so **dropping the `with check` line changes nothing the database
   does** — measured, suite green, `pg_policies.with_check` NULL. What must
   survive here is the PREDICATE (`hq_can_see_template(template_id)`, on the
   template axis and no other), not the presence of a particular clause. The
   reachable defect on this table is a **widened INSERT `with check`**, which
   W4/W6 catch only because every refusal is now also sent with
   `Prefer: return=minimal` — see item 10.
3. **`submission_responses` WITH CHECK is `hq_can_see_field(field_id)` —
   FIELD-scoped, never submission-scoped.** `submission_id` is nullable for
   offline drafts; a submission-scoped write predicate refuses every draft push
   and would look like a working policy. WP3 (a draft push with
   `submission_id: null` landing) is what says so.
4. **`submission_rejections` gains a SELECT policy.** `for select using
   (hq_can_see_field(field_id))`. Consequence (1) of decision 111: a device that
   can write a row it cannot read back breaks replication. **This deliberately
   changes what V18 asserted** — V18 is rewritten in place from "deny-all for
   admins too" to "the read is field-scoped, the WRITE is approver-only", and a
   merge restoring the old V18 restores a suite that contradicts the shipped
   policy. A1's G6 proved V18's old form; that proof is superseded, not lost.
5. 🛑 **The read/write asymmetry on `assignment_role` — `hq_can_approve_*` is
   used by the rejection policies ONLY.** `hq_can_see_template` and
   `hq_can_see_field` are byte-unchanged in 0003. A merge that "makes reads
   match writes" silently blinds every approver to the checklist they approve
   (0073 §1's warning) — and one that makes writes match reads hands every
   assignee the power to reject their own work. **W9 is the variant that fails
   in the second direction and POSITIVE/WP5 in the first.** Keep both.

   🛑 **AND KEEP THE APPROVER-UPDATE PAIR, WP8/W16 — G6 finding F2.** W9 is an
   INSERT refusal and W11 is refused by `using`, so before the fix round
   **nothing in the suite reached `submission_rejections_update`'s `with
   check`** — and that is the ONE write predicate in 0004 strictly narrower than
   its own table's SELECT policy (`hq_can_approve_field` ⊂ `hq_can_see_field`),
   which makes it the only one Postgres will not enforce for us. Setting it to
   `true` left the whole suite green while an approver moved a rejection onto
   work she is merely ASSIGNED to — an assignee rejecting her own work by
   UPDATE. **WP8** (the same PATCH to a field she DOES approve — must land) and
   **W16** (to a field she merely SEES — must be refused) are a matched pair on
   the identical probe; neither means anything alone. They need the fixture
   field **`fldApprover2`**, a second field under `tplApprover`, and the
   population floors that moved 7 → 8 with it.
6. **`hq_can_approve_field` nests EXISTS the same way `hq_can_see_field` does.**
   Written the obvious way — `hq_can_approve_template(<lookup fid>)` — an
   unresolvable field yields NULL and the ADMIN ARM IS STILL TRUE, so an admin
   could write a rejection onto a field nobody has ever heard of. W8 asks carol
   exactly that.
7. 🛑 **B-36 — `resolveSpikeConfig` no longer skips on an unresolvable
   substrate.** The skip is gated on `HQ_SYNC_SUBSTRATE_OPTIONAL=1`; anything
   else is `t.Fatal`. **A merge that restores the bare `return spikeConfig{}, false`
   returns the package to printing `ok` while running zero attack variants**, and
   with it goes every piece of E-KR2 evidence in the tree. Pinned by
   `TestSpikeGate_Asymmetry`'s new opt-out row and by
   `TestSpikeSubstrateOptional_IsTheOnlySkipDoor`.
8. **The write-side red mode `SYNC_RLS_SKIP_WRITE_POLICIES=1`** and its teardown
   (`tearDownWritePolicies`). It withholds 0004 while keeping 0003, which is the
   only state in which the write POSITIVES fail and the write refusals still
   pass. Without it the only reproducible red is "RLS off", under which every
   refusal fails and every positive passes — the exact blind spot the suite's
   own header measured (12 of 19 variants surviving an empty subject set).
9. **The two new population floors** — `hq_sync_template_approvers` (HQ side)
   and `hq_template_approvers` (across the wire). The approver relation is the
   one this card added; an empty one makes WP5/WP6 fail loudly and W9/W10 pass
   vacuously, and only the floor tells those apart.
10. 🛑 **`rvPushRefused` — EVERY WRITE REFUSAL GOES OUT UNDER BOTH `Prefer`
    HEADERS. G6 finding F1, and it is the root of this card's discriminating
    power.** `spikeStack.do` used to set `Prefer: return=representation`
    unconditionally, so every write went through PostgREST with RETURNING and
    Postgres applied **0003's SELECT policy to the new row**. The read
    predicates are identical to (rows 2, 3) or broader than (row 4) the write
    predicates, so the READ policy was silently enforcing the write half and the
    suite could not tell 0004 from 0003: **3 of 5 mutations to the shipped write
    predicates survived fully green**, W8's rejection arm among them — it was
    reading a 403 issued by `hq_can_see_field`. A merge that collapses
    `rvPushRefused` back into a single `s.do` call, or that restores the
    unconditional header inside `do`, **returns the suite to proving nothing
    about this file** while staying green. `doPrefer` is the seam; `do` keeps
    the old default so the read half is byte-unchanged.
11. **The post-run teardown in `rvConnect`.** A green run now takes 0004's seven
    policies back down via `t.Cleanup`. The substrate is SHARED — one compose
    project, one PostgREST, one `public` schema, and `HQ_RLS_TEST_DB` isolates
    only the HQ-side FDW source — so a green run of this suite left
    `submission_rejections_*` resident and reddened a sibling leg whose tree
    still carried the pre-rewrite V18. Dropping the cleanup makes an unmerged
    card change what other branches measure. It removes the residue, not the
    race; the race is filed as **B-50**.

## What is safe to drop

- The prose banners in `0004_write_policies.sql` and the new header block in
  `rowvisibility_rls_test.go`. They document decisions recorded in the ledger;
  losing them costs a reader, not a behaviour — **except** 0004 §2's "WHAT THIS
  FILE DELIBERATELY DOES NOT CONTAIN", which is the only in-tree statement that
  `checklist_templates`' missing write policy is a decision rather than an
  oversight. That one is load-bearing against a future tidier (item 1).
- The `.night-crew/knowledge/BACKLOG.md` append, **provided** the findings reach
  the backlog by some other route. It is a record, not a mechanism.
- This note.

## Conflicts I expect

`.night-crew/knowledge/roadmap.md`, `.night-crew/knowledge/BACKLOG.md` and
`sw.js`/`version.json` are touched by most cards on the slate; ordinary git
conflicts there are the orchestrator's to resolve. **Take my roadmap bullet, take
my BACKLOG append, take nothing else from this branch in those files, and re-run
`node build-sw.js` after the last merge rather than taking either side.**

`backend/internal/sync/rowvisibility_rls_test.go` and
`backend/internal/sync/spikestack_gate_test.go` are shared with any other card
touching `internal/sync` tonight. My edits to the row-visibility suite are
**additive except V18**, which is rewritten in place for the reason in item 4.
