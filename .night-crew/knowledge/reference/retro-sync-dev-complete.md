# Cycle retrospective — Sync, dev complete (2026-08-28)

> Attended `/nc-retro`, run in the target repo against its own layout: this repo's cycle
> knowledge lives under `.night-crew/knowledge/`, its nights are hand-run slates, and the
> dev-clone mechanics the retro skill names (`dev-close grade`, `usm/roadmap.txt`,
> `DESIGN.md §15`) do not exist here. Every substitution is named below; every absence is
> reported as an absence. Boundary: milestone closed this same sitting (`/nc-milestone-close`,
> marker `hq-20260828`, ledger T-45); latest run branch `overnight-20260810` merged into dev.

## 1. The six sources

1. **Card actuals** — read: `reference/card-actuals.md`, per-run sections for all 7 runs.
2. **Conflict logs** — read: 7 of 7 (`conflicts-20260806` … `conflicts-20260810`).
3. **Closeouts** — read: 7 of 7 (`closeout-20260806` … `closeout-20260810`) plus run HANDOFFs.
4. **Decision audit** — ran; reports **no gray areas have ever routed through the resolver on
   this target**. Not an absence of decisions: forks on this target live in the runs'
   `DECISIONS-NEEDED.md` and the ledger's triage records, and were counted from there.
5. **Team scorecard** — ran; **"No runs to show."** Hand-run nights write no team records;
   all four roles are invisible to it (see §3).
6. **OKR grades** — no `dev-close` record shape exists for a target repo; this milestone's
   close grades stand in (same sitting, judged by reading): **13 MET · 1 PARTIAL · 1 NOT MET**.

## 2. What the cycle cost

**The cycle bought the thing two predecessors failed to deliver — a sync capability the
operator personally watched work — for seven nights, two operator forks, and one 23-day park.**

- **Estimates against actuals.** Medians: 106m28s (N=5, night one, one 185m outlier) → ~68m
  (N=3) → ~59m (N=2) → ~82m (N=4). Build-class bands read calibrated by late cycle. Spike-class
  cards landed at or under the low end of their estimate four consecutive times — spike
  estimates still run high; a finding for the next round's sizing, not a scolding.
- **What the merges cost: zero.** Every merge in all seven logs records **zero conflicted
  hunks** — a full cycle of clean merges under the disjoint-footprint slate discipline, against
  three conflicted merges in the previous cycle. The seam-picking held.
- **What was parked.** Exactly one card: `gate-rls-fixture-ownership`, parked 2026-08-05 →
  2026-08-28 (23 days) on the attended re-gate (decision 155), settled this sitting — the
  preserved branch was lost to a worktree sweep in the interim and the card was re-implemented
  from its recorded leads. T-43(b)'s park guard was never tripped: three runs explicitly
  steered around the open My Checklists path rather than assume an answer.

## 3. The teams — per team, no comparison

The scorecard cannot see any rostered role on this target: hand-run nights write no team
records. Four stories, each the same story with a different name:

- **product (PM)** — unmeasured; the system cannot see it. Its cycle exists only in judgment
  form (§4's two measures). `—` in every cell.
- **delivery (PjM)** — unmeasured; the slate/actuals discipline it ran is visible only through
  `card-actuals.md`, which it maintained by hand all cycle. `—` in every cell.
- **engineering** — unmeasured; its cycle is visible only through the merged cards and gates.
  `—` in every cell.
- **quality (QA)** — unmeasured; its cycle is visible only through the gate ladder's holdings.
  `—` in every cell.

Whether closing this visibility gap is worth a card is the round's call, not the retro's.

## 4. Product-Manager performance — two measures, and neither is the rating

**Forks prevented** — 2 gray areas settled at slate planning before any run could hit them
(T-43 a/b: Approvals scope, My Checklists park guard); 2 escalated mid-run, both on night one
(D-1 gate-ladder definition; D-2 production posture, forced by the incident). After the first
triage, **six consecutive runs escalated nothing**; run 20260810's one engineer-level call was
decided in-run under standing authority and recorded, not escalated.

**PRD-to-KR traceability** — **not computable as specified: no cycle PRD exists.** This cycle
was authored at a roadmap round from the governing handoff
(`handoff-hq-sync-dev-complete-20260805.md`), not a PM-session PRD. Nearest honest substitute:
15/15 KRs traced to named artifacts at authoring (the gradability dry-run), and 0 cards
shipped outside the roadmap's card list.

Neither of these is the PM rating. The order they appear in is arbitrary and carries no
precedence; they are not averaged, weighted, ranked, or combined. The point of this cycle is
real numbers under both so the **next roadmap round** can choose between them; a choice made
here would be the choice the operator explicitly declined to make (§15bk.188).

## 5. What the cycle decided

- **Auto-resolution rate: N/A** — 0 gray areas routed through the resolver, so no escapes were
  possible by that path and no constitutional-tier decision was ever auto-resolved. Everything
  ceiling-adjacent went to the operator (D-2).
- **Delegations:** one engineer-level call decided in-run under standing authority (run
  20260810, gate-harness form), recorded in the merge-intent and ratified by the triage merge.
- **Grades in their three states: derived 0 · judged-by-reading 15 (13 MET / 1 PARTIAL /
  1 NOT MET) · deferred 0.** No fraction is stated because deferred is not a grade.

## 6. The cycle's judgments

1. Red-first became the named gate RF · operator chose it over an honest N/A · decision 153
2. Production gets a nightly dump and PITR · decision 154 — already pending as `operations/C-2`
3. Test suites leave the production cluster — structure over guards · decision 155
4. Backfill horizon 2026-03-01 · decision 156 (one-off recovery ruling)
5. Write-off scope for the destroyed data classes · decision 157 (one-off)
6. Sales-processor notice held, later amended to one combined notice · decision 158 →
   `process/C-1` already pending
7. Approvals stays on re-fetch by design · T-43(a)
8. A card that cannot proceed without an operator answer parks · T-43(b) — refined at this
   retro by the operator into the delegation rule below
9. "Dev complete" redefined to the operator's own hands at the app surface · decision 161
10. Attestation performed and recorded; A3 re-gate executed · T-45 + addendum (2026-08-28)
11. Prod deploy delayed at close · 2026-08-28 (one-off)
12. Four UI preferences consented at the walkthrough documentation sitting · 2026-08-28 —
    already pending (`ux/C-2`, `ux/C-3`, `design/C-1`, `process/C-2`)

**Offer-back (step 7):** 12 judgments listed · eligible operator answers not already pending:
153, 155, 161, T-43(b) — all 4 offered, 0 declined. Eligible but not offered, by name:
154 (already pending as `operations/C-2`), 156, 157, 158, 159, T-43(a) — read as one-off
rulings; any can be asked back. Written, each **pending, not adopted**:

- `gates/C-1` — An obligation that must hold gets a named gate · pending, not adopted
- `architecture/C-2` — Make the mistake impossible — structure over guards · pending, not adopted
- `process/C-3` — Dev complete means the operator ran it in their environment · pending, not adopted
- `delegation/C-1` — Preference-covered questions proceed under citation; park only what no
  preference answers (with its Bound: cheaply-revertible, surfaced-for-review decisions only;
  irreversible/outward-facing acts and the always-escalate ceiling stay the operator's) ·
  pending, not adopted — the store refused the first draft for lacking a Bound; the operator
  consented to the bounded text separately.

Adoption is the operator's own act: `night-crew preferences adopt <category>/<C-n>`.

## 7. What this suggests for the next roadmap — candidates, never decisions

- **The pending queue is now 13 candidates deep and nothing is adopted.** B-245's lesson
  standing at larger scale: an unadopted candidate is an answer the system will ask for again.
  An early adoption sitting would put the citation supply in place before the next cycle's
  first night — and `delegation/C-1`, if adopted, changes how every future park behaves.
- **Spike-class estimates run high** (4 consecutive at/under low end) — resize the class.
- **The shared-substrate gate hazard is now permanent by design** (B-50 aggravation, filed at
  the close): the suites' default resolution names the persistent dev stack. A card to point
  the gate default at an isolated project — or refuse when the resolved stack is the dev one —
  retires an entire class of unattributable red.
- **Four roles are invisible to the scorecard** on this target; decide deliberately whether
  that visibility is worth anything here, or record that it is not.
- **Q-KR4's miss is a template defect, not a discipline defect** — filed as B-376 in the
  night-crew clone at the close; the target-side fix is nothing.
- **The cycle-median producer gap** — filed as B-377 in the clone; if the next OKR page grades
  against a computed number again, name its producing step first.
