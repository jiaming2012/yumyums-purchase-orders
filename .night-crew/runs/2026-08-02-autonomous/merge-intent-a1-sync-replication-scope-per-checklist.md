# Merge intent — A1 `sync-replication-scope-per-checklist`

Run `20260802`, Track A, branch `card/a1-sync-replication-scope-per-checklist`,
cut off `overnight-20260802`.

## Shared files touched

| File | Why it is outside my own module |
|---|---|
| `.night-crew/knowledge/roadmap.md` | The status flip required in the same change set. **I touch ONE bullet only** — `sync-replication-scope-per-checklist` (roadmap line ~1635). Every other bullet on that page is another card's and must not be taken from this branch. |
| `.night-crew/knowledge/BACKLOG.md` | Scope-freeze destination for ~~one discovery (`SYNC-REALTIME-SCOPE`). Append-only — one new entry at the end of the file. Nothing existing is edited.~~ → **two discoveries, and one of them EDITS an existing line of my own.** (a) `SYNC-REALTIME-SCOPE`, **renumbered `B-39` → `B-42`** because three legs on run `20260802` each independently claimed B-39 and the pre-step's B-39/B-40/B-41 are already merged to the run branch and keep those numbers. (b) A NEW entry `B-43` (the two list views have no expressible scope), appended at the end. Both entries are mine and mine only; the renumber touches only my own B-39 heading, no other card's line. |
| `.night-crew/runs/2026-08-02-autonomous/merge-intent-a1-sync-replication-scope-per-checklist.md` | This note. Uniquely named for this card; no other card writes it. |

Everything else this card changes is inside its own declared footprint:
`sync-rxdb/client.js` and `tests/sync-rxdb-client.spec.js`.

**No schema change.** `sync-schema/collections.js` is byte-unchanged — the scope
keys the filter needs (`checklists.id`, `responses.submission_id`,
`responses.field_id`, `approvals.submission_id`, `templates.id`,
`templates.archived_at`) were all already declared by B1. **No policy change.**
`sync-schema/sql/` is byte-unchanged. The card's PARK trigger therefore did not
fire.

## What must survive any merge

> **G6 fix round, run `20260802`.** The list below was rewritten after an
> adversarial review returned two BLOCKING findings. Where a line was wrong it
> is struck **in place** with what replaced it — items 6 and 7 are new, and item
> 5's "must survive" bar is strictly higher than it was. This list is what
> `sync-hard-cutover` is held to on Night B.

1. **`sync-rxdb/client.js` — `startHQReplication` REFUSES to run unscoped.**
   A missing/incomplete `opts.scope` throws. This is the enforcement of
   preference `architecture/C-2`; if a merge resolution restores a
   default-to-whole-collection path, the preference is silently widened and the
   card is undone. The throw is the point, not a guard-rail.
   **Two things the fix round hardened, both of which must survive with it:**
   (a) `scope.templateId` is now **mandatory** — the old optional-with-fallback
   shape widened `templates` to every non-archived row *on a forgotten
   argument* (F-5), and a widening triggered by an omission is not a recorded
   decision; (b) the no-scope-case refusal is raised while **building** the
   query builder, not inside `pull.handler`, which the vendored plugin wraps in
   `try{…}catch{ retry }` — a refusal in there is a silent unbounded spin, not
   a refusal (F-3).
2. **Every replicated collection carries a `pull.queryBuilder`.** Four of four —
   `templates`, `checklists`, `responses`, `approvals`. A merge that keeps three
   and drops one re-opens the unbounded pull on the dropped collection.
3. **`responses` keeps its two-branch scope** — submitted rows for the open
   checklist `OR` draft rows (`submission_id is null`) restricted to the open
   checklist's `field_id` set. Collapsing it to the single `submission_id.eq`
   branch drops every offline draft, which is the one thing this collection
   exists to sync.
4. **`tests/sync-rxdb-client.spec.js` — the `[SCOPE-01]` describe block**, which
   evaluates the emitted PostgREST filters over a multi-checklist row fixture
   and asserts the never-opened checklist's rows are not returned. It carries
   its own non-empty-subject-set assertions (B-22/B-23/B-24) — those assertions
   must survive with it or the test can pass by returning nothing.
5. ~~**The `[SCOPE-01]` fixture's never-opened rows.** If a merge shrinks the
   fixture to one checklist the test still passes and proves nothing.~~
   → **NOT ENOUGH, and the shipped fixture already had the hazard this line
   warns about (F-2).** Never-opened rows are necessary but not sufficient: with
   one checklist per template and one approval per field id, `checklists:
   template_id.eq.<templateId>` and `approvals: field_id.in.(<fieldIds>)` are
   **indistinguishable** from the correct per-checklist scope, and the reviewer
   confirmed both mutations survive 6/6 green. What must survive is the
   **SIBLING** rows — `chk-sibling-…` (a second submission of the OPEN
   checklist's OWN template that this device never opened), `apr-3` (a rejection
   on an OPEN field id but a different `submission_id`), and `rsp-6` (a
   submitted response, same shape) — together with the three explicit
   discriminator assertions in the fixture-liveness test that assert those rows
   exist. Drop any of them and the fixture stops telling per-checklist scoping
   apart from per-template or per-field scoping.
6. 🛑 **NEW — `replicationIdentifier` MUST CARRY THE SCOPE (F-1).** It is
   `hq-sync-<table>-<fingerprint of that collection's own serialized filter>`.
   RxDB keys its checkpoint meta store by `hash([collection.name,
   replicationIdentifier])` and by nothing else, and the pull's checkpoint is
   the last row of the **scoped** result set — so a scope-free identifier means
   one checkpoint across all scopes, and switching to an OLDER checklist pulls
   **zero rows, permanently**. **A merge that restores the plain
   `hq-sync-${table}` identifier re-introduces silent data loss**, and it will
   look like a simplification, because the code comment that shipped with the
   card argued for exactly that. That comment is now corrected in place; the
   corrected version must survive with the code. Pinned by the `[SCOPE-02]`
   describe block, which runs the plugin's pull construction TWICE through a
   meta store keyed the way RxDB keys its own — including the control that a
   re-minted checkpoint returns the same set, and the counter-test that the
   SAME scope still RESUMES (so the fix cannot be "disable checkpoints").
7. 🛑 **NEW — scope values are validated and quoted (F-4).** `assertScopeId`'s
   `/^[A-Za-z0-9_-]+$/` whitelist and `serializeFilter`'s value quoting are two
   independent halves of one fix; keep both. Without them a scope value
   carrying PostgREST logic-tree punctuation (`,` `(` `)` `"`) rewrites the
   predicate into one true for every row — the whole table, reached *through*
   the thing this card calls a gate. Not exploitable today (ids are
   internally-generated), which is why it was cheap, not why it is optional.

## What `sync-hard-cutover` inherits from this card

- **It must CANCEL before re-scoping.** Each replication state owns a Realtime
  channel named after its `replicationIdentifier` and keeps writing into the
  local collections. Nothing in `startHQReplication` enforces cancel-then-start;
  the page lifecycle that opens a different checklist owns it. Stated in the
  `startHQReplication` docstring and in `bootstrap.js`.
- **`B-42 SYNC-REALTIME-SCOPE`** (renumbered from B-39) — the live
  `postgres_changes` leg is still unscoped.
- **`B-43`** — "My Checklists" and "Approvals" are LIST views over many
  submissions and the scope model is singular by construction, so either those
  pages are not RxDB-backed or someone records a C-2 widening. Unrecorded before
  the G6 round.

## What is safe to drop

- ~~The prose in `sync-rxdb/client.js`'s new `REPLICATION SCOPE` header comment
  block.~~ → **Mostly still true, with ONE carve-out.** The block is
  documentation of decisions recorded elsewhere (ledger T-29 decision 105,
  preference `architecture/C-2`) and losing most of it costs a reader, not a
  behaviour — **except the `CHECKPOINTS ARE PER-SCOPE` sub-block**, which does
  not merely document item 6 but explains why its obvious-looking
  "simplification" is data loss. That one is load-bearing against a future
  reader, and a merge that keeps the code but restores the old comment has set
  a trap.
- The `.night-crew/knowledge/BACKLOG.md` append, **provided** the
  `SYNC-REALTIME-SCOPE` (B-42) and list-view (B-43) findings reach the backlog
  by some other route. It is a record, not a mechanism. **The B-39 → B-42
  renumber, however, is not optional** — B-39/B-40/B-41 belong to the pre-step
  and are already on the run branch.
- This note.

## Conflicts I expect

`.night-crew/knowledge/roadmap.md` and `.night-crew/knowledge/BACKLOG.md` are
touched by most cards on the slate; ordinary git conflicts there are expected and
are the orchestrator's to resolve. Take my roadmap bullet, take my BACKLOG
append, take nothing else from this branch in those two files.
