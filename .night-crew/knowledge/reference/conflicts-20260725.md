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

> 🛑 **SUPERSEDED LATER THE SAME DAY — read Merge 2 below.** The subset was the wrong suite for
> this card. W1's full-suite leg surfaced two reds that the orchestrator then attributed to F1's
> `d1674d3` by measurement. **The `DONE` flip recorded above was reverted**; F1 needs a client
> half. See DECISIONS-NEEDED.md FORK 1.

## Merge 2 — W1 `sync-spike-stack-and-jwt-bridge` → `overnight-20260725`

**ONE CONFLICT, resolved.** Cards involved: W1 (and F1 historically, via the shared instrumentation
file). Merge commit `51d0c02`, `--no-ff`.

**Merge-intents read — both sides, as §15ad.65 requires:**
- W1's: `.night-crew/runs/2026-07-25-autonomous/merge-intents/w1-sync-spike-stack-and-jwt-bridge.md`
  (114 lines, committed at `1acb297` at 10:54:47, **before** the first implementation commit
  `e2a4ca4` at 11:07:15 — the discipline held, and G6 corroborated the ordering from `git log
  --reverse`). It declares `timings.log` an append-only shared surface.
- The orchestrator's side: `0430aaf`, a single `W1_IMPL_START` stamp on the same file.

**The conflict:** `.night-crew/runs/2026-07-25-autonomous/timings.log`, one hunk. Both sides appended
to the end of the file — the classic shared-instrumentation collision, and the only file both sides
touched.

**Resolution — union, chronological, nothing dropped.** Both intents agree the file is append-only
and additive; there is no competing behaviour to adjudicate, only ordering. `W1_IMPL_START`
(10:45:53) precedes W1's own leg block, so it was placed first and W1's block kept verbatim.

**One substantive merge-time correction, made deliberately.** W1's leg comment concluded its two
full-suite reds were *"pre-existing on `overnight-20260725`."* That inference is **false**, and W1
could not have known — a card in a worktree cannot check out another branch to test it. The
orchestrator can, and did (below). Resolving *against intent rather than text*: W1's intent was to
flag the correlation and refuse attribution, which was correct behaviour. So its measurements are
kept **verbatim** and the correction is **appended beneath, clearly attributed to the
orchestrator**, rather than edited into the card's own words. A card's record should read as what
the card actually found.

**The attribution the correction rests on** — both specs, same box, same Postgres, separate DBs and
ports:

| Tree | F1 present? | Result |
|---|---|---|
| `dev` @ `d37fb10` | no | **2 passed** (2.6 m), load 2.84 → 3.28 |
| run branch @ `c14cbce` | yes | **2 failed**, load 3.01 → 2.48 |

→ `tests/repro-cut-task.spec.js:153` and `tests/sync.spec.js:1581` are an **F1 regression**, not
flakes, not `:1198`, not pre-existing. Parked as **FORK 1**; F1's roadmap flip reverted in the same
change set as this entry.

**Blast-radius attestation — checked, not taken on trust.** `git diff --name-only
overnight-20260725..card/w1-... -- backend/go.mod package.json docker-compose.nc.yml Taskfile.yml`
→ **empty**. Product/test check `-- 'backend/**' 'tests/**' '*.html' 'sync.js'` → **empty**. Across
the whole run (`dev..HEAD`) the four HARD files remain untouched, and the only product delta is
F1's `repository.go` plus its two test files. W1's merge intent stated all four per item; the claim
holds.

**Gates after merge:** G1 `go build ./...` exit 0, G2 `go vet ./...` exit 0, re-run on the merged
tree as the conflict rules require. The full Playwright suite was **not** re-run post-merge, and
deliberately: W1's merge adds no product or test bytes, so the merged tree's product content is
bit-identical to `c14cbce`, which had already been measured. Re-running 22 minutes of suite to
observe the same two known reds would have bought nothing.

**G6:** PASS-WITH-FINDINGS — 8 findings, **all non-blocking**, none touching the verdict. The
reviewer independently reproduced the RLS discrimination pair and the per-subscriber Realtime RLS
proof against the live stack and confirmed the runbook works as written for someone who did not
write it. The findings worth carrying: `HTTP <code>` lines annotated onto `curl` commands that
lack `-w` (a small breach of the document's own "real captured output" standard), three proof steps
missing their "names the failure it would catch" sentence, a `go.sum` carrying both
`coder/websocket` v1.8.14 and v1.8.15 while `go.mod` pins v1.8.14, and one off-by-one filename
citation. Per the slate, G6 did **not** relitigate the signed inline-throwaway-credentials
decision; it checked only that the labelling is loud, and it is.

## Merge 3 — W2 `sync-spike-rxdb-replication` → `overnight-20260725`

**ONE CONFLICT, resolved.** Cards involved: W2, colliding with the orchestrator's own park document.
Merge commit `ba22744`, `--no-ff`. Merged **after** a G6 revision round (below), not before.

**Merge-intents read — both sides:**
- W2's: `.night-crew/runs/2026-07-25-autonomous/merge-intents/w2-sync-spike-rxdb-replication.md`,
  committed at `0ae53b5` as its **first** commit, before any implementation. It attests **per item**
  across **five** constraints (the four HARD files plus `docker-compose.supabase.yml` UNMODIFIED,
  which W1 owns and W2 only consumes) and declares `README.md`, `sync-rxdb-feasibility-spike.md`,
  `timings.log` and `DECISIONS-NEEDED.md` as append-only shared surfaces.
- The orchestrator's side: `396b97e`, which created `DECISIONS-NEEDED.md` with FORK 1 (the F1
  regression) and FORK 2.

**The conflict:** `.night-crew/runs/2026-07-25-autonomous/DECISIONS-NEEDED.md`, **add/add** — the
file did not exist at W2's branch point (`51d0c02`) and both sides created it.

**Resolution — concatenation, both sides kept whole, and it needed no adjudication.** W2 anticipated
this exactly: it wrote its half as a **title-less `h2` fragment** so it would concatenate under
whatever title the orchestrator's copy carried, checked `overnight-20260725` at `396b97e`, saw FORK
1 and FORK 2 already taken, and **numbered its own forks 3 and 4** — then left an HTML merge-note
saying "if both sides exist, KEEP BOTH." That note was dropped at resolution, its instruction having
been discharged; everything else survives byte-for-byte. Result: one document, one title, FORK 1–4
in order.

**This is what a merge-intent is for.** The collision was designed out before it happened rather than
adjudicated after, and the merger's job reduced to deleting three conflict markers.

**`timings.log` auto-merged clean** — union, no markers. The two cards appended to disjoint regions
because W2 branched after W1's block was already committed.

**G6 — PASS-WITH-FINDINGS, and it caught a real one. One BLOCKING finding, revised and re-checked.**
The reviewer reproduced all three proofs green first try, re-ran the LWW experiment and got the
card's result a third time, and verified every mechanism claim verbatim against the shipped RxDB
source. But the card had written that the discarded write leaves **"no signal an application could
subscribe to."** That is false: `RxReplicationState` exposes `conflict$`
(`plugins/replication/index.js:44,51,287-289`), and G6's own probe showed it emitting one event
carrying the discarded document.

**Why that blocked rather than being a nitpick:** the error inflated the cost of DECISIONS-NEEDED
FORK 3 option 4 ("surface the conflict to the user") — making a `conflict$.subscribe(...)` plus UI
read as new plumbing in the sync layer — **inside a document written to help the operator choose
between options.** A wrong fact in a decision aid is worse than a wrong fact in a report.

**Revision round dispatched to the card, not patched by the orchestrator** (the dispatch model keeps
diffs out of the control loop's context; the slate budgets a revision round for exactly this). The
card verified `conflict$` independently — source *and* a fresh empirical probe — agreed plainly that
it was wrong, corrected all three operator-facing documents, re-priced option 4, and **found two
further caveats G6 had not raised**: `conflict$` is fed by a plain `Subject`, not a `ReplaySubject`
(a late subscriber gets nothing), and the event fires per replication, not per document, carrying no
user-facing text. Both are now recorded so the correction is not over-read in the other direction.
Revision commit `7262e3f`.

**Blast-radius attestation — checked, not taken on trust.** `git diff --name-only 51d0c02..HEAD` over
`backend/go.mod`, root `package.json`, `package-lock.json`, `docker-compose.nc.yml`, `Taskfile.yml`,
`docker-compose.supabase.yml`, `backend/internal`, `tests`, `*.html`, `sync.js` → **empty**. The
appends are strictly additive after the revision (README **681/0**, design note **242/0** — zero
deletions), and W1's runbook half 1 was verified **byte-identical** by diffing the first 704 lines
against `51d0c02`. The harness carries its own `package.json` + lockfile under
`.night-crew/qa/spike-supabase/rxdb/`, as required.

**Gates after merge:** G1 `go build ./...` exit 0, G2 `go vet ./...` exit 0, re-run on the merged
tree. The full Playwright suite was **not** re-run post-merge, deliberately and for the same reason
as merge 2: W2 adds no product or test bytes, so the merged tree's product content is bit-identical
to what W2 already measured (538 passed / 2 failed / 1 flaky / 6 skipped, 30.5 m, load 1.58 → 4.20).
Its 2 failures are **exactly** the two known F1 reds and there was **no third red**.

**Whole-run sweep:** `git diff --name-only dev..HEAD` over the four HARD files → **empty**. Across
all three cards the only product delta is F1's `repository.go` plus its two test files.
