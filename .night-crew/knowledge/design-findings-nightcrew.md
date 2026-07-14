# Night-crew design findings — pending graduation to night-crew main

> **Purpose.** A standing, cross-run sink for findings that are **night-crew framework
> changes, not HQ-app changes** — surfaced while dogfooding night-crew on HQ. Distinct from
> `BACKLOG.md` (HQ-app items). Mirrors the per-run `runs/<date>/design-findings.md` precedent
> (`runs/2026-07-09-attended/design-findings.md`), but durable across runs so findings aren't
> orphaned.
>
> **The loop this file feeds (operator intent, 2026-07-14):** entries here → a post-run/
> post-cycle **batch against the night-crew main repo** → the fix lands in main → the **tool-pin
> advance** (roadmap.md "Tool pin" standing rule) propagates it to HQ and every other project
> pulling from that pin. Today the sweep is un-triggered (no guaranteed producer); the desired
> end state is a feedback step — ideally at cycle/roadmap completion or in `/nc-morning-triage`
> — that graduates these to main and advances the pin.

## Open findings

### NF-1 · `/nc-slate-plan` budget sizing is systematically over-conservative (under-packs the night)
- **Symptom (persistent on HQ):** slates finish far under budget. `overnight-20260714` used
  **~2h26m of a 6h budget (~41%)** and barely dented the token/quota budget.
  `reference/card-actuals.md` documents the estimate bias directly: cards ran **6–10× under**
  first-of-kind estimates (first slate), then **2–4× under** (second slate; e.g. `ops-nfr3`
  23m actual vs 45–75m estimate).
- **Root cause (in night-crew main, the `/nc-slate-plan` skill — §2b cut-line logic):** the
  cut line is drawn against the estimate's **high end**, with a full ~30 min closeout reserve
  and conservative stretch-gating, so the night is under-packed even as the actuals ledger
  shows the estimates themselves are already high. The skill treats the budget as a cautious
  ceiling rather than a floor to fill.
- **Operator directive (2026-07-14):** treat the stated night budget as a **LOWER limit**, not
  a conservative upper bound. Size slates ambitiously; lean **concurrent** dispatch when
  footprints are disjoint (multiplies throughput); set generous stretch so the run never idles
  with budget in hand. The quality bar (G1–G6, red-first) is the guardrail — throughput should
  push against it. Prefer "attempt more, exit clean-early" over sandbagging.
- **Proposed fix (night-crew main):** re-anchor §2b estimation on the *recorded actuals'
  central tendency* (not the first-of-kind high end); make the cut line pack to the budget with
  a small fixed margin; and, since actuals now show wide headroom, make **concurrent** the
  default recommendation whenever §5's disjoint-footprint criteria hold. Calibrate against
  `card-actuals.md` size-class ranges.
- **Scope note:** the fix belongs in night-crew **main**, not HQ — HQ is only where the symptom
  is persistent. Until the pin advances, `/nc-slate-plan` on HQ compensates by sizing
  ambitiously per the operator directive above.

### NF-2 · The design-findings feedback loop has no guaranteed producer / no cycle-completion trigger
- **Symptom:** `design-findings.md` — the channel for framework findings → night-crew main —
  has fired **exactly once** (2026-07-09). Nothing *triggers* a findings sweep; a finding is
  captured only if someone remembers to write it. This is the same **NO-ORPHAN-INPUTS**
  violation that the first design-findings doc (Finding 1) was itself written to fix.
- **Operator intent (2026-07-14):** there should be a feedback step — "maybe after the roadmap
  is done" (cycle/roadmap completion) or folded into `/nc-morning-triage` — that (a) sweeps
  persistent framework findings into a durable sink (this file), (b) batches them against
  night-crew main, and (c) advances the tool pin so all consuming projects inherit the fixes.
- **Proposed fix (night-crew main):** add a guaranteed producer for the design-findings sweep at
  a cycle boundary. Candidates: a step in `/nc-okr-session` (cycle start reviews last cycle's
  findings) or a closeout step in `/nc-morning-triage` / a new cycle-complete skill. Make this
  standing file (`knowledge/design-findings-nightcrew.md`) the canonical target, scaffolded by
  `night-crew init` alongside the other knowledge files.
