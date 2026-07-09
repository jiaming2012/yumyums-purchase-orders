# OKRs

<!-- Cycle: HQ hardening — first night-crew guinea-pig run. Vehicle: make every feature
in all 5 apps (Operations/workflows, Inventory, Onboarding, Users, Purchasing) fully
functional end-to-end, and make every written test actually verify behavior — not pass
by coincidence. Weighted to establishing night-crew operational baselines (§4, §10). -->

## Product

### Objective: Every one of the 5 apps has a written, testable definition of "fully functional end-to-end," and that enumeration proves near-complete against what the build actually surfaces.
- 5/5 apps (workflows, inventory, onboarding, users, purchasing) have a hardening PRD enumerating their critical E2E flows, delivered as an early blocking gate before WO build begins.
- Enumeration recall ≥ 90%: original enumerated flows ÷ (original + flows discovered during WO build) ≥ 0.90 — Product foresaw at least 90% of the flows actually worked this cycle.
- At least 90% of committed WOs trace to a PRD-enumerated flow (traceability baseline, §15j.42).

## Delivery

### Objective: Ship the WOs that close every broken or unverified flow, and record the night-crew cadence baseline.
- 100% of flows marked broken/unverified in the app PRDs have a shipped WO by cycle end.
- Median WO cycle time recorded over at least 5 WOs — establishes the baseline this cycle; no pass/fail target (target set next cycle once real data exists).

## Engineering

### Objective: Every enumerated flow works end-to-end against a real database; no known-broken flow ships.
- 0 known-broken flows remain across the 5 apps at cycle end (denominator = Product's enumerated flow list).
- Full E2E suite runs green against localhost Postgres — 0 pre-existing red tests (`task test` exits 0).

## QA

### Objective: The suite actually verifies behavior — no test goes green without exercising the flow it names, and every repaired test is proven to catch a real break.
- Vacuous tests: 23 → 0 — every conditional `test.skip()` (15) and silent guard-return (8) becomes a real seeded assertion or is deleted.
- 100% of Product-enumerated critical flows have at least 1 E2E test that drives the real flow and asserts observable DB/UI state.
- 100% of repaired or added flow tests carry a recorded red-first proof — shown failing when the feature is broken, passing after the fix (bug-fix protocol).
