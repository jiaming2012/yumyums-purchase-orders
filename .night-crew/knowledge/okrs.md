# OKRs

Milestone: Sync, dev complete — the RxDB layer serves reads in the operator's dev environment, proven by a demo script the operator runs personally.

<!-- Authored 2026-08-05, attended `/nc-roadmap-round`, in the same sitting as
`.night-crew/knowledge/roadmap.md` (DESIGN §15j.42). Previous cycle archived at
`reference/okrs-2026-08-05-sync-foundation.md`; its close at `reference/cycle-closeout-20260805.md`
(ledger §T-37, decisions 148–152).

🛑 THE ONE LESSON THIS PAGE IS BUILT AROUND. The previous cycle's twelve key results were all
honestly gradable and NOT ONE MEASURED DELIVERY — four objectives asserted that the sync rewrite
"ships", and every KR measured sequencing, parity, cadence or coverage. The close graded 8 MET
with the capability undelivered and ZERO production call sites, and no KR could see it.

So, three authoring rules applied to every KR below (handoff §7):
  1. EACH OBJECTIVE HAS AT LEAST ONE KR THAT MEASURES ITS OWN CLAIM DIRECTLY. Where an objective
     says a thing works, a KR asserts that the thing was RUN and passed.
  2. EVERY KR NAMES ITS `(measured by: …)` ARTIFACT, and the artifact either exists today or is a
     named deliverable of a card on this roadmap. No KR rests on a hope.
  3. THE GRADE WAS DRY-RUN IN THIS SITTING — see "Gradability dry-run" at the foot. "Could not be
     graded" must be impossible to discover at the close. (Mechanism filed in the night-crew clone
     as B-343; applied by hand here.)

A fourth rule, from the same close: a KR must not be failable by desirable behaviour. Spike C
returning a RED verdict is a SUCCESSFUL spike — D-KR1 and E-KR1 say so explicitly, because
discovering a false premise early is exactly what this cycle is buying. -->

## Product

### Objective: The sync capability is something the operator can actually use — demonstrated by their own hands in dev, not asserted by a card count.

- **P-KR1 — exactly 1 recorded run of `task demo:sync`, performed by the operator in the dev environment, passes the round-trip**: one field written through the real write path (`/saveResponse`) surfacing in an RxDB-served read, on 1 real checklist (measured by: `.night-crew/knowledge/ledger.md`) 🛑 **No card status, KR grade or closeout substitutes for that line, and the milestone may not close without it** — it is the whole point of the cycle.
- **P-KR2 — at least 1 production call site of `createHQSyncDatabase()` and `startHQReplication()` exists at close**, on a code path that executes when the flag is on, against a baseline of **0** captured this sitting (measured by: `sync-rxdb/bootstrap.js`) 🛑 This KR exists because its exact negation is what the last cycle shipped: 6 repo hits, all comments, imports and deferred re-exports, and no call.
- **P-KR3 — exactly 1 dated decision resolves the list-view scope question (B-43) BEFORE the card that would discover it merges**: either My Checklists and Approvals are recorded as staying on REST (a partial cutover **by design**), or a C-2 widening is recorded with its bound (measured by: `.night-crew/knowledge/ledger.md`).

## Delivery

### Objective: The route is proven by runnable scripts before the nights spend themselves on it, and the milestone ends dev complete rather than merely built.

- **D-KR1 — all 4 spikes (A–D) record a green-or-red verdict by a SCRIPT, not by prose, and 0 Activity 3–5 build cards are dispatched before those 4 verdicts land** (measured by: `.night-crew/qa/spike-supabase/`) 🛑 **A spike whose script reports RED grades this KR MET** — the verdict is the deliverable, and a disproven premise found in an afternoon is the outcome being bought. Dispatch timestamps come from `.night-crew/runs/`.
- **D-KR2 — 1 committed `task demo:sync` target distinguishes 3 outcomes — passed · ran-and-failed · could-not-run — with all 3 paths invoked at least once** (measured by: `Taskfile.yml`) 🛑 The third is the one that matters: a demo that silently no-ops reproduces the class this milestone exists to retire.
- **D-KR3 — a per-card cycle-time median is computed with N ≥ 8, 100% of excluded cards listed with a reason, and 1 stated sensitivity check**, against the **103m (N=11)** baseline from the "Sync foundation" cycle (measured by: `.night-crew/knowledge/reference/card-actuals.md`) 🛑 Carries **B-39** forward — two consecutive cycles produced un-countable cards because only the implementer leg was stamped. Stamp G6-start / G6-return / fix-return per card this time.
- **D-KR4 — prod parity is verified at least 1 time this cycle with 0 drift**: `task version` shows prod backend/frontend == local `version.go` constants (measured by: `.night-crew/knowledge/ledger.md`) 🛑 **Inherited unmet by deliberate deferral** (T-37 decision 149) — `dev` is 436 commits ahead of `main` and carries migration `0072`, whose changeover date is owed to the sales-processor maintainer.

## Engineering

### Objective: The read path is real and bounded — RxDB serves what it is scoped to serve, and the offline-ownership rule is pinned by something that would notice its removal.

- **E-KR1 — exactly 1 written verdict establishes whether the HQ-Postgres → substrate → RxDB-read path exists, and by what mechanism**, backed by spike C's script output (measured by: `.night-crew/knowledge/designs/`) 🛑 **A NO verdict grades MET** — that is the premise decision 126 measured false on night nine of nine. A **silent or absent** verdict grades NOT MET.
- **E-KR2 — 0 cards widen replication scope beyond per-open-checklist without a recorded decision** (measured by: `sync-rxdb/client.js`) 🛑 Standing rule, T-29 decision 105; cross-checked against `ledger.md` decisions.
- **E-KR3 — the offline-ownership rule is enforced by 1 assertion over the OBJECT rather than over source text, with 0 read routes into RxDB left unnamed by it**, plus 1 mutation check showing it reddens when a route is added (measured by: `tests/sync-rxdb-client.spec.js`) 🛑 Closes **B-88**: today's 3 `not.toContain` assertions at `:1468-1470` miss `window.HQSync.db` — the exact route `workflows.html:3590` uses — and are green only because the database does not exist.
- **E-KR4 — both of the 2 fetch-storm-class items are either retired with 1 regression test each, or restated with their surviving mechanism named in 1 note each** (measured by: `.night-crew/knowledge/designs/`) 🛑 **Inherited NOT MET and un-reworded**: `sync.js` is still in the tree with both mechanisms live at `:443-454` and `:475-479`. This cycle does not remove `sync.js`, so an honest restatement is an acceptable outcome — silence is not.

## QA

### Objective: A green means the check ran. Every gate reading this cycle rests on can distinguish "passed" from "never executed".

- **Q-KR1 — 0 configurations exist in which `internal/sync` exits 0 with `TestRowVisibilityRLS` unrun and `HQ_SYNC_SUBSTRATE_OPTIONAL` unset**, with the 59-subtest count asserted rather than inferred (measured by: `backend/internal/sync`) 🛑 Closes **B-36**. The probe: strip docker from `PATH`, assert non-zero exit.
- **Q-KR2 — `verify-test-harness.sh` Check B reds when any 1 of the 7 packages loses fail-loud, not only when all 7 do**, with all 7 probed individually (measured by: `verify-test-harness.sh`) 🛑 Closes **B-22**. Today it aggregates with OR, so 6 of 7 can report `ok` on a dropped database and the gate still prints PASS.
- **Q-KR3 — 100% of WOs whose deliverable includes a code change carry a `## Red-first` section**: the named test, the tree it was captured red against, and the green after; a documentation/audit/spike WO records `n/a — no code change` explicitly (measured by: `.night-crew/runs/`) 🛑 **An absent section grades this KR down.** This is its **first gradeable cycle** — it graded UNAUDITABLE last time because the field had never existed, and exactly 1 merge-intent carried the heading.
- **Q-KR4 — 100% of slate documents carry, exactly once, a `Gate cost` section** stating expected full-suite runtime and that night's load-sensitivity risks (measured by: `.night-crew/knowledge/reference/`) 🛑 **Backfilling a signed slate is prohibited** — a missing section grades this down and is not repaired retroactively.

---

## Gradability dry-run — performed 2026-08-05, in this sitting

Authoring rule 3, applied. Each KR checked against the artifact it names, **before** sign-off, so
that "could not be graded" cannot arrive as a surprise at the close.

| KR | `measured by:` artifact | Exists today? | Verdict |
|---|---|---|---|
| P-KR1 | `ledger.md` | ✅ file exists; the attestation line is `dev-complete-attestation`'s deliverable | **gradable** |
| P-KR2 | `sync-rxdb/bootstrap.js` | ✅ **baseline captured this sitting: 0 call sites** | **gradable** |
| P-KR3 | `ledger.md` | ✅ exists | **gradable** |
| D-KR1 | `.night-crew/qa/spike-supabase/` | ✅ directory exists (last cycle's spike harness) | **gradable** |
| D-KR2 | `Taskfile.yml` | ✅ exists; the target is `demo-sync-target`'s deliverable | **gradable** |
| D-KR3 | `reference/card-actuals.md` | ✅ median + Excluded + Sensitivity sections all present from last cycle | **gradable** |
| D-KR4 | `ledger.md` | ✅ `task version` exists and runs | **gradable** |
| E-KR1 | `.night-crew/knowledge/designs/` | ✅ directory exists, holds 8 notes | **gradable** |
| E-KR2 | `sync-rxdb/client.js` | ✅ exists | **gradable** |
| E-KR3 | `tests/sync-rxdb-client.spec.js` | ✅ exists — the assertion being replaced is at `:1468-1470` | **gradable** |
| E-KR4 | `.night-crew/knowledge/designs/` | ✅ both superseded notes already there | **gradable** |
| Q-KR1 | `backend/internal/sync` | ✅ exists; probe already reproduced at triage | **gradable** |
| Q-KR2 | `verify-test-harness.sh` | ✅ exists | **gradable** |
| Q-KR3 | `.night-crew/runs/` | ✅ exists; merge-intent format established | **gradable** |
| Q-KR4 | `.night-crew/knowledge/reference/` | ✅ exists; slate format established | **gradable** |

**15 of 15 gradable. 0 UNAUDITABLE by construction.** 🛑 Two KRs (D-KR1, E-KR1) carry an explicit
**disproven-premise-grades-MET** clause, so a spike doing its job cannot redden the page — the
failure mode the previous cycle's grading lesson named.

🛑 **`night-crew okr validate` requires the parenthesized `(measured by: X)` form.** Prose
`Measured by:` and em-dash `— measured by:` both parse as *"names no measurement artifact"* while
the page still reports `okrs: valid` at exit 0 — i.e. the producer check fails silently unless the
warnings are read. Established by probe at this round; worth a night-crew backlog entry.
