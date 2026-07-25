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

**Not yet run at fold time:** the subset Playwright leg (`workflows|persistence`). F1's footprint is
seam-confined, so the subset is the correct suite; it is owed before F1 can be called done.

### Addendum — subset Playwright leg, run 2026-07-25 10:38–10:44 on the merged tree (`850775e`)

**GREEN.** `DB_HOST=localhost DB_PORT=46413 CI=1 npx playwright test "workflows|persistence"` →
**102 passed, 1 skipped, 0 failed.** Run on the run branch itself (main checkout at `850775e`),
against the ephemeral pg16 (`nc-f1-postgres-1`, Docker-assigned host port 46413) with a freshly
dropped/created `hq_test_e2e`.

**Measured runtime — this is the cycle's first seam-confined card, and the slate asked for the
actual, not the estimate:** **6 m 18 s wall clock** (Playwright self-reported **6.6 m**). The slate's
QA KR3 table estimated **~8–12 m**; the real subset is **~half to two-thirds of the low end**. Carry
this into the ledger as the first measured subset actual — future seam-confined cards should be
priced against 6–7 m, not 8–12 m.

**Measured box load (P2/P3a — the condition sampled, not an assumption):** 1-min load average
**1.83 at start, 3.60 at end**; 13 unrelated Docker containers resident throughout (other projects'
Postgres/Temporal/observability stacks). The box was **not** idle. The green therefore bounds the
loaded-ish condition, not a quiet one. `tests/sync.spec.js` is **not** in this subset, so the
`:1198` flake had no exposure here — which was the stated reason to run F1 first.

**Two aborted attempts preceded it, both harness faults, no tests executed** (recorded in
`timings.log`): (1) the non-interactive shell's PATH lacks `/usr/local/go/bin`, so Playwright's
`webServer` died with `go: not found` / exit 127; (2) `env PATH=$PATH …` unquoted, and this box's
PATH contains spaces (`/mnt/c/Program Files/…`). Neither touched the tree. Worth carrying into
triage as a run-mechanics note: **`export PATH=/usr/local/go/bin:$PATH` is required before any
Playwright leg in this environment.**

**F1 is now complete on its server half.** Roadmap card flipped `PLANNED` → `DONE` in this change
set. The excluded `0b53d46` (red-first Playwright test, no client half) is unchanged by this and
remains for morning triage.
