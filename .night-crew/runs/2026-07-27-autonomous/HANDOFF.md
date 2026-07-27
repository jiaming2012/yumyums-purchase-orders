# HANDOFF — run `overnight-20260727`

**Branch:** `overnight-20260727` (cut from `dev` @ `e1c40a8`) · **Slate:**
`.night-crew/knowledge/reference/slate-20260727.md` (batch sign-off 2026-07-26) ·
**Dispatch:** Wave 0 alone, then serial — as signed.

**2 of 3 slated cards landed and merged. Nothing parked. Card C was deliberately not started.**

You are the morning reader. The short version: **the night's most valuable output was a rejection.**
Card B shipped a fix that would have silently destroyed crew-entered checklist answers, G6 caught it
by execution, and the repair reverted to exactly the half the operator's decision had authorized.
Card A landed clean. Card C was not opened, on purpose, and that decision is yours to endorse or
overturn.

The things that want your attention are **D-1** (an operator call the run refused to make),
**D-6** (a trap that will break production if Card C is dispatched as written), and the Card C
go/no-go itself.

---

## Per-card outcomes

| Card | Verdict | G6 | Merge |
|---|---|---|---|
| **A** `pwa-cache-and-build-hygiene` (Wave 0) | **LANDED** | APPROVE-WITH-NOTES, 8 findings, 1 repair round | `c14865b` |
| **B** `workflow-offline-double-submit` | **LANDED (repaired)** | **REJECT** → repaired → re-review APPROVE-WITH-NOTES | `d8f5d8e` |
| **C** `sync-rxdb-schema-and-replication` | **NOT STARTED — deliberate** | — | — |

23 commits on the run branch. Every commit carries a `Night-Crew-Card:` trailer.

---

## Card A — the cross-tenant disclosure is closed, and the claim about it is honest

All five acceptance items green: `caches.delete('api-cache')` on logout (awaited before the
redirect), `checkAuth`'s offline branch fails closed on identity, `build-sw.js` globs the tracked
set, the vendored bundle is out of the precache, `sw.js` regenerated and committed. Precache
**23 files / 1947.1 KB → 22 / 1455.6 KB**.

**G6 refuted the implementer's honesty claim, and the repair narrowed it rather than defending it.**
The implementer declared one test vacuous but missed that the "fail closed on identity" third of
decision 57 is a **behavioral no-op with zero coverage** — G6 deleted `removeUserHeader()` and all
three new tests still passed, then showed by call-site census and an executed A/B that the line can
never fire on today's single render path. The code stays as defense-in-depth; `index.html`, the
roadmap card, and the merge-intent note now record what was *measured*. **See D-3** — "fail closed"
means closed-on-failure, never closed-on-wrong-identity.

Second repair closed a real vacuity: the vendored-bundle guard read the committed artifact rather
than rebuilding, so a config-only regression would have passed. It now rebuilds, and was shown red
under a config-only stub before commit.

**G6 hygiene was real:** two fresh clones of its own, per-test stubbing, executed A/B, and a
cold rebuild. It also caught that the implementer's "before" precache figure was synthetic (the
delta was still correct, and it reproduced both to the decimal).

---

## Card B — G6 REJECTED it, and the reject was right

**The shipped fix reused both `id` and `idempotency_key`.** Reusing `id` turns the queue write from
append into **replace**, and offline-entered answers have no other durable home. So the second
Submit built an empty payload and overwrote the only copy of the first. G6 measured all four
variants end-to-end:

| variant | server rows | crew's answer |
|---|---|---|
| pre-fix | **2** (the original bug) | preserved |
| **as shipped** | 1 | **GONE — zero recorded answers** |
| **key-only** (what decision 60 authorized) | 1 ✓ | **preserved** ✓ |

A food-safety checklist submitting with zero answers, silently, with a success toast — inside the
card's own headline scenario. And **ledger decision 60 authorized reusing the key, not the `id`**:
the unauthorized half caused the loss, and the authorized half alone met the goal.

**Why the card's own tests could not see it:** DBL-01/02 answer the field *while online*, going
offline only after the 1500 ms autosave, so a server-side draft repopulated the second payload. One
line's difference.

**The repair** (5 items) reverted `id:` to `generateUUID()` — the line is byte-identical to base, I
verified that myself — added **DBL-04** (offline *before* the click) as the regression guard, sorted
`drainQueue` by `queuedAt`, corrected a **factually false** justification that had reached the
durable record, and recorded the knowingly-accepted costs.

**The false justification is worth reading.** The record claimed key-only reuse would 409 and be
evicted as `duplicate_submission`. Measured: **201 twice, identical submission id**, evicted on the
success path; `duplicate_submission` appears **nowhere in `backend/`** — that branch in `sync.js` is
dead code. The same sentence dismissed "would submit the stale response set first" as a *downside*,
when that is precisely the property that saves the data. The sign was inverted. The implementer's
own note back: both the false claim and the unauthorized `id` reuse "were written from code-reading
rather than execution, and both were wrong in the same direction."

**Independent re-review** (fresh reviewer) confirmed the blocker closed by its own measurement
(`rows=1 responsesPerRow=[1] values=[["true"]]`) and confirmed **DBL-04 is a deterministic sole
guard** — red on both run and retry under the `id`-reuse stub while DBL-01/02/03 stay green.

---

## Card C — NOT STARTED, and why

**This was an orchestrator decision at 02:40 EDT, taken under the slate's own budget-discipline
clause:** *"if the clock or quota is tight when B lands, stop cleanly rather than opening C."*

The evidence:

| Card | Slate estimate | Actual (impl + G6 + repair + merge) |
|---|---|---|
| A | 30–50 m | **~1 h 45 m** |
| B | 45–90 m | **~2 h 50 m** |

Both ran **~2–2.5× estimate once review and repair were counted** — and the 20260726 ledger already
warned to budget a G6 repair round per card by default. Card C is priced at **120–240 m before
review**, is the riskiest card in the slate, is first-of-kind, and carries **five binding
obligations and three explicit park triggers**. Opening it at 02:40 meant finishing somewhere
between 06:00 and 10:00, with 30 minutes of closeout owed off the top. **Card C is last in the
slate by design precisely so this call stays available**, and the honest read is that it could not
have been finished cleanly.

Starting and abandoning it mid-flight would have been strictly worse than not starting: it would
have left an unmerged worktree, a half-designed conflict handler, and a park that wasn't a real
park.

🛑 **Before Card C is dispatched, two things must go into its slate entry — see D-6.** The important
one: **re-adding the vendored bundle to the precache (its obligation 5) will break production**
unless `vendor/` is also added to `backend/Dockerfile`, which never copies it. That would re-create
the exact bug Card A just fixed. It also means decision 59 was under-argued — justified on bandwidth
when the real justification was a broken SW install shipping to every returning client.

---

## Gate evidence on the FINAL MERGED TREE

Run after both merges **and** after the post-merge anchor-fix commit — this is the tree you would be
merging to `dev`. A re-run was required rather than inherited: the anchor fix edits comments in
`workflows.html`, and Workbox's precache manifest carries a per-entry content revision hash, so
`sw.js` moved and the tested artifact was no longer the committed one.

**FINAL-TREE SUITE: 554 passed / 0 failed / 0 flaky / 6 skipped of 560, 21.8 m, `--retries=0`, exit 0.**
Fresh DB (`hq_test_final27`), `TEST_PORT=8221`. **Zero hard failures on the tree you would merge.**

> 🛑 **The final tree ran 560 tests; both card legs ran 559 and 555. The difference is not code —
> it is a silent coverage gap in every card worktree. See D-9.** The main repo has a git-ignored
> `.features-gen/` directory (playwright-bdd output) giving it a 20th spec file and the `[bdd]`
> project; card worktrees have no such directory, so **every card's "full suite" was 19 of 20
> files.** The omitted BDD test passes on the final tree, so nothing regressed — but no card
> actually measured it. Verified by `--list` on both trees: final `Total: 560 tests in 20 files`,
> card B `Total: 559 tests in 19 files`.
>
> Per-card figures, all at `--retries=0`, fresh DB per leg:
> - Card A branch: **549 passed / 0 failed / 0 flaky / 6 skipped of 555**, 22.8 m
> - Card B branch (post-repair): **553 passed / 0 failed / 0 flaky / 6 skipped of 559**, 22.2 m
> - Go, both cards: `go test ./... -count=1 -p 1` — **10 packages ok, 0 failed**, including
>   `internal/workflow` (the `-p 1` matters; without it that package reds with a known-false
>   `checklist_templates_created_by_fkey` violation)

**Test count grew 549 → 559** across the night: +6 from Card A, +4 from Card B. Both deltas were
verified by `npx playwright test --list` against the base tree by an independent reviewer, not
asserted.

**Four HARD constraints:** empty diff vs `dev` for root `package.json`, root `package-lock.json`,
`backend/go.mod`, `docker-compose.nc.yml`, root `Taskfile.yml` — re-verified after every repair.
**`backend/` is untouched by the entire run.** Card B's park trigger was never approached.

**No `openspec/` directory was created.** Preflight verdict ABSENT; only universal mechanics
asserted. B-105 remains open (**D-8**).

---

## Standing flags

| Flag | Status |
|---|---|
| **Attended two-device convergence check** (carried since 2026-07-22, re-armed 2026-07-26) | 🔴 **STILL YOURS.** Card B changed the live offline submit path. Cannot be done unattended. **You owe it, and it now matters more** — the submit path's queue semantics changed. |
| **`tests/sync.spec.js:1198`** (proven ~16–20 % flake) | ✅ Did not red on any leg, all night — 4 full-suite runs at `--retries=0`. Zero retries consumed. **Still armed**; this is 4 clean samples, not a fix. |
| **`tests/purchasing.spec.js:1407`** (FR-13) | ✅ Passed on every full-suite leg. |
| **`tests/sync.spec.js:1584` stale comment** (ledger decision 66) | 🔴 **STILL OPEN.** Card B did not touch that file. Folds into the next card that does — Card C. |
| **`api-cache` cross-tenant disclosure** | 🟡 **Logout half CLOSED** (Card A). Two siblings remain unowned — **D-2**. Structural half still belongs to Card C. |

---

## What morning triage should do

1. **Endorse or overturn the Card C no-go** (**D-6**). If you want it, its slate entry needs the
   Dockerfile trap written in first.
2. **Answer D-1** — `git ls-files` reads the index, so a staged-but-uncommitted file still enters
   the precache manifest and still 404s in prod. `git ls-tree -r --name-only HEAD` closes it, but
   that varies decision 58's literal text, so the run refused to decide.
3. **Route D-2** (two unowned disclosures of the same shape) and **D-4 + D-5** (one follow-up card:
   bound `findQueuedSubmission` to the current period, age out stale queue entries, and add the
   missing `ON CONFLICT` on `submission_fail_notes`).
4. **Audit `conflicts-20260727.md`** — 2 merges, 2 entries. Merge 1 clean; merge 2 had one genuine
   conflict in `timings.log`, resolved by **union with nothing discarded**. **D-7** proposes the
   durable fix so a third night doesn't pay for it again.
5. **Record this run's actuals.** `card-actuals.md`'s `overnight-20260726` section is complete (it
   was backfilled at triage *after* the slate was signed, so the launch prompt's "no row exists"
   premise was already stale). This run's rows are owed.

---

## The lesson this run is evidence for

**Both cards' implementers made their worst error the same way: reasoning from code where execution
was available.** Card A claimed a branch "strips any name already on screen" — it can never fire.
Card B claimed key-only reuse would 409 — it returns 201 twice. Neither was checked by running it.
Both were caught by reviewers who *ran the thing*.

The G6 gate is not overhead to be trimmed. On this run it was the difference between shipping and
silently deleting a crew member's food-safety checklist. **Budget a repair round per card and treat
its absence as the exception** — the 20260726 ledger said this, and tonight is the second
consecutive night of evidence for it.

---

**Final:** `git log --oneline dev..overnight-20260727` — 23 commits. Branch is **local only**;
nothing was pushed, no tags, no deploys, `main` never touched. All card worktrees left intact and
clean.
