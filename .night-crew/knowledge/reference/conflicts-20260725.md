# Conflict log — overnight-20260725

> One entry per merge to `overnight-20260725`, clean or conflicted (§15ad.66): cards involved,
> files and hunks, the merge-intents read, the resolution taken, and the gate result after it.
> Clean merges get a one-line entry — an empty log must never read as "no conflicts" when it
> means "the logging never ran". Committed with each merge; audited at morning triage.

## Merge 1 — F1 `workflow-submission-status-default` (server half) → `overnight-20260725`

**CLEAN** — no conflicts (first merge of the run; the run branch had not moved since the card
branched). Cards involved: F1 only.

**Attended fold, not an orchestrator merge.** The run's driving session ended after
`F1_IMPL_START` (09:17:20) without merging; `merge-intents/` in the run directory was empty on
disk because F1's note was committed on the card branch rather than written to the run tree.
This merge was performed by the operator's attended session on 2026-07-25 to resume the night.

**Merge-intent read:** F1's, `.night-crew/runs/2026-07-25-autonomous/merge-intents/f1-workflow-submission-status-default.md`
(77 lines, committed at `a5f9bf2` *before* implementation, as the discipline requires).

**Range folded:** `a5f9bf2..d419888` — the merge-intent, the red-first Go tests, the
`repository.go` fix, and the resubmit fixture cleanup. Files: `backend/internal/workflow/repository.go`,
`backend/internal/workflow/submission_status_test.go` (new), `backend/internal/workflow/resubmit_photo_gate_test.go`.

**Deliberately excluded:** `0b53d46`, the red-first Playwright test in `tests/workflows.spec.js`.
Its own commit message states it is red "before the client half," and no client half was written
— `workflows.html` is untouched. That file sits outside F1's signed footprint (slate-20260725
card 1(c): owns `repository.go`; touches `*_test.go` and the workflows/persistence specs), and
the slate makes the E2E leg conditional — *"if the leak is provable at the E2E layer."* It is
proven at the Go layer by `b9a5cfb`, which is what the card required. The commit remains on
`card/f1-workflow-submission-status-default`, unmerged and undeleted, for morning triage to
dispose of.

**Gates after merge:** `go build ./...` + `go vet ./...` exit 0; `go test ./... -count=1 -p 1`
green across 9 packages (alerts, auth, inventory, purchasing, receipt, recipes, sync, toast,
workflow). Run on the card tree at `d419888`, whose content is identical to the merged tree.

**Gate note worth carrying into triage:** `go test ./...` *without* `-p 1` reds the workflow
package with FK violations on `checklist_templates_created_by_fkey` — parallel packages share
`hq_test_go` and their TestMains truncate `users` cross-package. This is the documented
cross-contamination surface #3, not an F1 defect. `-p 1` is load-bearing and the launch prompt
(§141) already says so.

**Not yet run:** the subset Playwright leg (`workflows|persistence`). F1's footprint is
seam-confined, so the subset is the correct suite; it is owed before F1 can be called done.
