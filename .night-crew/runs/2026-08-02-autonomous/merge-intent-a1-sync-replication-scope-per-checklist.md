# Merge intent — A1 `sync-replication-scope-per-checklist`

Run `20260802`, Track A, branch `card/a1-sync-replication-scope-per-checklist`,
cut off `overnight-20260802`.

## Shared files touched

| File | Why it is outside my own module |
|---|---|
| `.night-crew/knowledge/roadmap.md` | The status flip required in the same change set. **I touch ONE bullet only** — `sync-replication-scope-per-checklist` (roadmap line ~1635). Every other bullet on that page is another card's and must not be taken from this branch. |
| `.night-crew/knowledge/BACKLOG.md` | Scope-freeze destination for one discovery (`SYNC-REALTIME-SCOPE`). Append-only — one new entry at the end of the file. Nothing existing is edited. |
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

1. **`sync-rxdb/client.js` — `startHQReplication` REFUSES to run unscoped.**
   A missing/incomplete `opts.scope` throws. This is the enforcement of
   preference `architecture/C-2`; if a merge resolution restores a
   default-to-whole-collection path, the preference is silently widened and the
   card is undone. The throw is the point, not a guard-rail.
2. **Every replicated collection carries a `pull.queryBuilder`.** Four of four —
   `templates`, `checklists`, `responses`, `approvals`. A merge that keeps three
   and drops one re-opens the unbounded pull on the dropped collection.
3. **`responses` keeps its two-branch scope** — submitted rows for the open
   checklist `OR` draft rows (`submission_id is null`) restricted to the open
   checklist's `field_id` set. Collapsing it to the single `submission_id.eq`
   branch drops every offline draft, which is the one thing this collection
   exists to sync.
4. **`tests/sync-rxdb-client.spec.js` — the `[SCOPE-01]` describe block**, which
   evaluates the emitted PostgREST filters over a two-checklist row fixture and
   asserts the never-opened checklist's rows are not returned. It carries its
   own non-empty-subject-set assertions (B-22/B-23/B-24) — those assertions must
   survive with it or the test can pass by returning nothing.
5. **The `[SCOPE-01]` fixture's never-opened rows.** If a merge shrinks the
   fixture to one checklist the test still passes and proves nothing.

## What is safe to drop

- The prose in `sync-rxdb/client.js`'s new `REPLICATION SCOPE` header comment
  block. It is documentation of decisions recorded elsewhere (ledger T-29
  decision 105, preference `architecture/C-2`); losing it costs a reader, not a
  behaviour.
- The `.night-crew/knowledge/BACKLOG.md` append, **provided** the
  `SYNC-REALTIME-SCOPE` finding reaches the backlog by some other route. It is a
  record, not a mechanism.
- This note.

## Conflicts I expect

`.night-crew/knowledge/roadmap.md` and `.night-crew/knowledge/BACKLOG.md` are
touched by most cards on the slate; ordinary git conflicts there are expected and
are the orchestrator's to resolve. Take my roadmap bullet, take my BACKLOG
append, take nothing else from this branch in those two files.
