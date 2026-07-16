# Sign-off — attended PM session + grill-back, 2026-07-16

**Status:** ✅ SIGNED OFF by operator (reply "sign off", 2026-07-16 evening).
**Run:** `2026-07-16-attended` (evening PM session of the "Nothing silently lost" cycle).

## What is being signed

The **checklist data-integrity PRD** as this cycle's frozen intent contract — the
blocking gate (roadmap Activity 1) the overnight planner builds work orders from.

- PRD: `.night-crew/knowledge/prds/PRD-data-integrity.md` (`night-crew prd validate`
  → `prd: valid`; re-validated after each grill-back amendment).
- OKRs: amended twice in-session, `night-crew okr validate` → `okrs: valid`.
- Shape: **9 FRs + 3 NFRs** under the **frozen-at-submit** semantic — stable field
  identity · loud rejection · edit broadcast with re-render · transactional op
  emission · frozen-at-submit lifecycle · multi-device convergence matrix · 2
  engine-trust fixes (approval feedback, 409 refetch). **No schema migration.**
- Traceability: 100% of requirements trace to the reproduced Friday failure or a
  named invariant (INV-1…6), and to a Product/Delivery/Engineering/QA KR.
- Routing: 0 intake items (inbox empty); 10 pass-2 sweep finds all through a door
  (6 folded · 3 backlogged · 1 dropped with reason); the morning's 15-item routing
  record amended to final destinations. 0 items unaccounted.

## The two grill-back reversals (recorded, ledger G-1/G-2)

1. **G-1:** premise correction — prod has **no active Operations users** → the
   morning's "stages 1–2 ship first" sequencing deleted.
2. **G-2:** semantic put head-to-head — operator delegated with a hard
   multi-device-sync bar → PM chose **frozen-at-submit** over run-pinned versioning;
   versioning schema demoted to backlog; old stages 1–2 revived as the permanent
   architecture.

## Assumptions accepted (the grill survivors)

Signing off = accepting these:

- **A-1 — Semantic (delegated → PM chose):** frozen-at-submit. An unsubmitted
  checklist always shows the current template on every device; submit freezes the
  record forever; rejection reopens it live.
- **A-2 — Rejection lifecycle (PM proposal, re-checked at the FR-1 design gate):**
  frozen record never mutates; redo is live against the current shape carrying
  prior answers; flags on since-cut fields dissolve visibly; resubmit re-freezes.
  (Operator walked the 5-field example 2026-07-16 — "works for me.")
- **A-3 — Explicit discard (PM proposal, re-checked at the FR-1 design gate):**
  cutting a field discards its unsubmitted answers; the Builder warns first, naming
  the count (INV-6).
- **A-4 — Scope growth from the sync bar:** transactional op emission (FR-5) and
  the full convergence matrix (FR-7: 7 field types + sub-steps + submit/unsubmit +
  progress bars) are requirements, not stretch. PjM sizes the cards.
- **A-5 — Falsifiability bound:** live "in sync" = converged within one op
  round-trip, asserted on the observing device.
- **A-6 — FR-8 test method:** approval-feedback failure forced via a Go test seam;
  the production change is only the error surfacing.
- **A-7 — No migration expected:** QA migration KRs may have denominator 0 this
  cycle.
- **A-8 — Second catch-point:** rejection rules, discard rule, day-boundary (C5),
  and the per-surface convergence contract are finalized in the FR-1 OpenSpec
  design, operator-signed **before any build card dispatches**.

0 gray areas reached this table without a door; 0 forks queued to morning triage.

## After sign-off

- Safe clear-point: this conversation may be cleared.
- Next: `/nc-slate-plan` sizes the night from roadmap Activities 2/5/6 (Activity 4's
  `editprop-openspec-design` is attended and gates Activity 5's build cards).
- The FR-1 design sign-off is a **separate, second attended gate** — tonight's
  signature does not pre-sign the design.
