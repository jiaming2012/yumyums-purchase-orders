# Merge intent — F1 `workflow-submission-status-default`

Branch: `card/f1-workflow-submission-status-default` (cut from `overnight-20260725`)
Written BEFORE implementation, as the card's first commit.

## Card in one line

`checklist_submissions.status` is left at `'pending'` forever for submissions against a
`requires_approval:false` template, because `submitChecklist` never sets the column and
nothing else moves it. Normalize the insert to `'completed'` (already permitted by the
existing CHECK — no migration), and gate `pendingApprovals` on the submission's own
`template_snapshot->>'requires_approval'` so no-approval submissions cannot leak into a
real approver's queue.

## Shared files touched

- `.night-crew/runs/2026-07-25-autonomous/merge-intents/f1-workflow-submission-status-default.md`
  — this note. New file, unique to this card. No conflict surface.
- `.night-crew/runs/2026-07-25-autonomous/timings.log` — append-only per-leg timing lines,
  prefixed `F1 `. Every card appends its own prefixed lines; conflicts, if any, are
  append-order only and both sides should be kept.

Everything else is inside this card's own package (`backend/internal/workflow/`) or its
tests. Any file added to this list during implementation is appended below under
"Late additions" with the evidence that forced it.

### Late additions

_(to be filled in only if implementation forces a file outside `backend/internal/workflow/`;
"nothing here" if it stays clean)_

## What must survive any merge

1. **`submitChecklist` sets `status` explicitly at insert.** The `INSERT INTO
   checklist_submissions (...)` in `backend/internal/workflow/repository.go` must carry a
   `status` column whose value is `'completed'` when the template does not require approval
   and `'pending'` when it does. A merge that drops the column from the insert list silently
   restores the bug — the DB default backfills `'pending'` with no error anywhere.
2. **`pendingApprovals` gates on the submission's own snapshot, not the live template flag.**
   The predicate `(s.template_snapshot->>'requires_approval')::boolean IS NOT FALSE` (or the
   documented `t.requires_approval` fallback, if the snapshot is proven not to carry the key)
   must remain in the `WHERE` clause alongside the existing `s.status = 'pending'` and
   `ta.assignment_role = 'approver'` conditions. This is the line that makes historical
   `'pending'` rows harmless without a data migration. Dropping it re-opens the leak for
   every row already in prod.
3. **Both regression tests.** The invariant test (a no-approval submission's status is not
   left at `'pending'`) and the leak test (a no-approval template *with* an approver
   assignment does not put its submissions in that approver's pending queue). Both were
   written red-first. They are the only thing that will catch a re-regression.

## What is safe to drop

- Comment wording, test names, and the exact phrasing of any explanatory block comment.
- Test helper placement — if another card moves shared workflow-test helpers into a
  `helpers_test.go`, this card's helpers can be relocated freely as long as the two
  assertions above still run.
- The `timings.log` lines. They are a record, not a behaviour.
- Anything in this note itself.

## Not done, deliberately

- **No backfill of historical `'pending'` rows.** Decided in the slate. The
  `pendingApprovals` gate neutralizes them immediately; a data migration over prod history
  is an attended act. Residue is bounded and recorded in the card report.
- **No migration.** `'completed'` is already in the existing CHECK constraint and is
  currently unused. No new lifecycle value, no schema change.

## Four-HARD-constraints attestation

1. `backend/go.mod` — **UNTOUCHED**. No dependency added, removed, or version-changed.
2. Root `package.json` — **UNTOUCHED**. The Playwright environment for every card in the
   repo is unchanged.
3. `docker-compose.nc.yml` — **UNTOUCHED**. This card's ephemeral Postgres is brought up
   under project name `nc-f1` with a ports override supplied from a scratch file OUTSIDE
   the repo.
4. Root `Taskfile.yml` — **UNTOUCHED**. Suites were run by invoking `go test` / `npx
   playwright test` directly with `DB_HOST` / `DB_PORT` pointed at the ephemeral cluster.
