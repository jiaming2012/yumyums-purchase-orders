# Design findings — attended run 2026-07-09 (night-crew, for morning triage)

> These are **night-crew design changes**, not HQ changes. Night-crew is frozen at
> `e4b43ba` for this run — none are implemented tonight. They join the post-run
> batch (with the PRD-verifier gate + the cadence-delegation decision).

## Meta-principle: NO ORPHAN INPUTS

Every artifact a workflow step **consumes** must have a guaranteed **producer**
upstream in the workflow. No load-bearing artifact may depend on the operator
remembering to request it. When a step reads X, some prior step must own creating
X — otherwise the workflow has a hole.

Two instances found this run:

## Finding 1 — the roadmap has consumers but no producer

**Evidence:**
- `/nc-okr-session` reads the roadmap "only if it has one; absence is normal."
- `/nc-pm-session` reads the roadmap "only if it has one; absence is normal."
- `/nc-slate-plan` **hard-depends** on it ("infer the next activity in roadmap
  order — hard rule", "fan out the roadmap cards") — and reads night-crew's own
  dogfood paths (`usm/roadmap.txt`, `ROADMAP.md`), so for a generic target it is
  both **unproduced and path-ambiguous**.
- `night-crew init` scaffolds CONTEXT/bugs/ledger/preferences/rulebook — **but no
  roadmap.**

→ Result: three steps read a roadmap, one requires it, **zero create it.** Tonight's
roadmap exists only because the operator asked.

**Proposed fix (3 coordinated changes):**
1. **`/nc-okr-session` produces the roadmap alongside `okrs.md`** — the roadmap
   sequences the activities that deliver the KRs. Primary fix: a guaranteed
   producer at the once-per-cycle, attended, cycle-planning moment.
2. **`night-crew init` scaffolds a roadmap stub** at a canonical target path
   (e.g. `.night-crew/knowledge/roadmap.md`), killing slate-plan's path ambiguity
   for generic targets.
3. **Flip "absence is normal" → "absence routes to `/nc-okr-session`"** in the PM
   session and slate-plan. Precondition, not shrug. slate-plan should assert the
   *target's* roadmap exists and read it (not night-crew's dogfood roadmap).

## Finding 2 — the PRD-verifier gate (from the PM session)

Already logged in `sign-off.md`. Same principle: the overnight crew's draft-PRD
quality bar has no automated producer/checker; tonight the operator was the manual
gate. Build the verifier calibrated by tonight's manual gating.

## Finding 3 — cadence-delegation decision (from the PM session)

Already logged: operator sets the quality threshold; the PjM owns throughput per
pass. Record in night-crew DESIGN/ledger so the next cycle doesn't re-litigate it.
