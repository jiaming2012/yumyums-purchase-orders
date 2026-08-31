# Merge intent — `cdc-single-fire`

Run `20260901`, Track B. Closes **B-157**.

## The bug (B-157)

One `/saveResponse` (one user save) fires any row-level CDC trigger on
`submission_responses` **twice**, because a single user action produced **two row
writes** to that table:

1. `backend/internal/workflow/repository.go` — `saveResponse`'s `INSERT ... ON CONFLICT
   DO UPDATE` (the save itself). Did **not** touch `lamport_ts`.
2. `backend/internal/sync/ops.go` — `EmitOp` → `updateEntityLamportTS`'s
   `UPDATE submission_responses SET lamport_ts` (the LWW stamp, fired async by the same
   handler).

A row-level trigger can't tell the two apart → 2× events per user action.

## The fix — one logical save = one row write

- `saveResponse` gains a `stampLamportTS int64` parameter and returns
  `(int64, error)`. When `stampLamportTS > 0` it folds that value into the SAME
  INSERT/upsert as the save (`lamport_ts = EXCLUDED.lamport_ts`) and RETURNs it — one row
  write. When `stampLamportTS <= 0` it runs the ORIGINAL bare upsert unchanged (leaves
  `lamport_ts` alone).
- `SaveResponseHandler` computes the winning `lamport_ts` up front via the new
  `opsync.NextLamportTS` (== `current + 1`, the same value `EmitOp` would have assigned),
  passes it to `saveResponse` (the one stamped write), then emits the op row via the new
  `opsync.EmitOpForStampedEntity` — inserts the op row + fires `pg_notify` with that same
  lamport_ts and **does NOT** re-`UPDATE submission_responses`. Exactly ONE row write per
  save; the op row carries the matching lamport_ts; pg_notify still fires.

### Critical scoping — the `/ops` path is deliberately left alone

The op-journal path (`POST /ops` → `workflowOpRouter` → `SaveResponseFunc`) is a SEPARATE
write channel where the sync handler runs the LWW **conflict check** (`CheckLWW`) AFTER
the business write and then stamps the CLIENT's `req.LamportTS` itself
(`EmitOpWithConflictCheck`). That path calls `SaveResponseFunc(..., 0)` — **stamp
disabled** — so `saveResponse` must NOT touch `lamport_ts` there. Stamping in
`saveResponse` on that path would move `lamport_ts` before `CheckLWW` reads it and could
turn a legitimately-winning client op into a false conflict. B-157 names `/saveResponse`;
the fix is scoped to it. Any residual `/ops` double-write is a different endpoint and out
of this card's scope.

LWW semantics preserved on `/saveResponse`: the stamped value is still `current + 1`
(monotonic, always wins the guard), and the op row's lamport_ts equals the row's
lamport_ts (asserted by the RF test).

## Shared files touched

- **`backend/internal/sync/ops.go`** — adds THREE new symbols: exported
  `EmitOpForStampedEntity` (fire-and-forget op-row insert + notify, NO entity-table
  update), exported `NextLamportTS` (thin wrapper over the existing private
  `nextLamportTS`), and unexported `insertOpRowAndNotify` (the op-journal half of
  `InsertOpAndNotify`). Existing `EmitOp`, `EmitOpTx`, `InsertOpAndNotify`,
  `updateEntityLamportTS`, `CheckLWW`, `nextLamportTS*` are **UNCHANGED**. **Card 8
  `app-slug-association` also touches `internal/sync`** — this card only ADDS symbols and
  modifies no existing one, so a merge conflict is unlikely; if one arises, keep both
  these three additions and Card 8's.
- **`backend/internal/workflow/repository.go`** — `saveResponse` signature changes from
  `(…, userID string) error` to `(…, userID string, stampLamportTS int64) (int64, error)`.
  Callers: `SaveResponseHandler` (stamp = NextLamportTS), `cmd/server/main.go`
  `workflowOpRouter` (stamp = 0), and tests (stamp = 0).
- **`backend/internal/workflow/handler.go`** — `SaveResponseHandler` computes the stamp,
  passes it to `saveResponse`, and emits via `EmitOpForStampedEntity` (or falls back to
  `EmitOp` on the delete path / a NextLamportTS read error). Other EmitOp call sites
  (template/submission entity types) are UNCHANGED — B-157 is scoped to
  `submission_responses` only.
- **`backend/cmd/server/main.go`** — the one `SaveResponseFunc` call in `workflowOpRouter`
  gains the `, 0` stamp arg (no stamp on the /ops path — see scoping above).

## What must survive any merge

1. **`saveResponse` stamps `lamport_ts` inline in the save write** — this is the whole
   fix. If a merge reverts it to a bare INSERT that omits lamport_ts, the double-fire
   returns.
2. **`SaveResponseHandler` must NOT call the entity-table-updating `EmitOp` for
   field_response** — it must use `EmitOpForStampedEntity` (op row + notify only). If a
   merge restores `EmitOp` here, the second write returns.
3. **The RF trigger-count test** (`internal/workflow/cdc_single_fire_test.go`) — installs
   an `AFTER INSERT OR UPDATE ON submission_responses` counting trigger and asserts ONE
   fire per save. It reds (count=2) on the pre-change tree.

## What is safe to drop

Nothing. `saveResponse`'s new return value is consumed by the handler; keep it.
