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

- `saveResponse` now computes `lamport_ts = current + 1` and folds it into the SAME
  INSERT/upsert as the save (SELECT-then-INSERT inside a short tx). It returns the
  stamped `lamport_ts`.
- The `field_response` op emission from `SaveResponseHandler` now uses a new
  `opsync.EmitOpForStampedEntity` — it inserts the op row + fires `pg_notify` with the
  lamport_ts `saveResponse` already stamped, and **does NOT** re-`UPDATE
  submission_responses`. So there is exactly ONE row write to `submission_responses` per
  save. The op row still carries the matching lamport_ts, and the pg_notify still fires.

LWW semantics are preserved: the stamped value is still `current + 1` (monotonic, always
wins the guard), and the op row's lamport_ts equals the row's lamport_ts.

## Shared files touched

- **`backend/internal/sync/ops.go`** — adds one new exported function
  `EmitOpForStampedEntity` (fire-and-forget op-row insert + notify, NO entity-table
  update). Existing `EmitOp`, `EmitOpTx`, `InsertOpAndNotify`, `updateEntityLamportTS`,
  `CheckLWW`, `nextLamportTS*` are **UNCHANGED**. **Card 8 `app-slug-association` also
  touches `internal/sync`** — this card adds an isolated new function + does not modify
  any existing symbol, so a merge conflict is unlikely; if one arises, keep both the new
  `EmitOpForStampedEntity` and Card 8's additions.
- **`backend/internal/workflow/repository.go`** — `saveResponse` signature changes from
  `error` to `(int64, error)` (returns the stamped lamport_ts). Its only callers are
  `SaveResponseHandler` and tests.
- **`backend/internal/workflow/handler.go`** — `SaveResponseHandler` uses the returned
  lamport_ts + `EmitOpForStampedEntity` instead of `EmitOp`. Other EmitOp call sites
  (template/submission entity types) are UNCHANGED — B-157 is scoped to
  `submission_responses` only.

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
