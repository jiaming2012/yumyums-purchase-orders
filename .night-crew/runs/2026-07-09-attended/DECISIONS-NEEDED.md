> **RESOLVED 2026-07-10 — recorded as `ledger.md` "Morning-triage resolutions (2026-07-10)".**
> D-1 → intentionally API/scheduler-only (Activity-4 WOs test-only). D-2 → `CLAUDE.md` fixed
> Mockup → Active at triage. D-3 → Trends/Cost waived as unbuilt-future (out of the Eng-KR
> denominator). D-4 → graduated to `BACKLOG.md`, rides the Users Activity-4 WO. All 4 PRDs
> signed; split-PARK non-fire upheld. This file kept as the analysis record.

# DECISIONS-NEEDED — overnight-20260710 (morning triage)

> **Reader:** the operator, at `/nc-morning-triage`. **Local file** (gitignored `runs/`).
> **Cards parked this run: 0.** Every Activity-1 PRD drafted and G6-passed. This file holds
> **non-blocking triage decisions** the enumeration surfaced — none blocks signing off the
> PRDs; each is a product/scope call for you, mostly feeding Activity 2–4 WO scoping.

## No parked cards

- **Nothing was parked.** All four cards (Users, Onboarding, Purchasing, Inventory) completed
  their two passes + G6 and committed. No footprint breach, no unresolvable status call, no
  quota stall.
- **The Inventory split-PARK trigger did NOT fire.** The slate authorized parking + splitting
  Inventory into two PRDs if the receipt-ingest pipeline and recipe/COGS system proved to be
  two independent sub-apps each rivaling a full PRD. They are **not** independent — both sit on
  `purchase_events` / `purchase_line_items` / the shared item catalog, and the completeness
  semantics thread through both (period-summary's gate reads the pipeline's pending state;
  menu-cogs reads confirmed line-items joined to recipes). **Verdict: KEEP-SINGLE**, reached by
  the drafter and independently upheld by the G6 reviewer. One coherent 40-requirement doc.
  *No decision required — recorded for the audit trail.*

## Triage decisions (non-blocking — resolve during/after sign-off)

### D-1 — Purchasing: 5 admin endpoints with no UI surface *(product call)*
`POST /simulate-cutoff`, `POST /orders/{id}/lock`, `POST /orders/{id}/unlock`, and the three
`/repurchase-reset*` routes are real, routed handlers with **zero UI affordance** in
`purchasing.html` (reachable only via direct API, or the scheduler for auto-lock). Enumerated
and marked UNPROVEN (handlers exist; behavior untested). **Decision:** is each intentionally
API-/scheduler-only, or does it need a UI surface? Your answer scopes whether the Activity-4
WOs are test-only or test-plus-UI-build. *(Captured in `PRD-purchasing-hardening.md`; not a
defect — G6 confirmed adequate treatment.)*

### D-2 — Purchasing: CLAUDE.md "Mockup" label is stale *(doc-update WO)*
The repo-root `CLAUDE.md` still lists Purchasing as "Mockup," but it has 21 routed endpoints,
a transactional service layer, a 15-min scheduler, a live 4-tab UI, and a 31 KB spec. The PRD's
Scope note corrects the label *in the PRD*, but **CLAUDE.md itself was not edited** (outside this
card's footprint — enumerate-only). **Decision:** approve a small doc-update WO to fix the
`CLAUDE.md` Current-Tools table (Purchasing: Mockup → Active).

### D-3 — Inventory: Trends + Cost tabs waived as unbuilt-future *(confirm the waiver)*
FR-24 (Trends) and FR-25 (Cost) are confirmed BROKEN — static `.coming-soon` stubs at
`inventory.html:993-999`, no API/state. The PRD **waives** them rather than filing fix-WOs, on
the grounds that standing up the charts is net-new feature work, not hardening. **Decision:**
confirm the waiver (build the charts in a later feature cycle) — or, if you'd rather not ship
dead tabs, a small WO to hide/remove the placeholders. Either way they stay out of the
"0 known-broken flows" Engineering-KR denominator only if explicitly waived here.

### D-4 — Users: two stale E2E tests target dead DOM *(test-repair WO, Activity 3/4)*
`tests/users.spec.js` has two tests (Access-tab: "shows all apps…", "can toggle a role
permission…") that navigate via `#t3`/`#s3` — DOM removed in the 3-tab→2-tab refactor (Access
now renders into `#s2`). The *features work*; the *tests can't run*. Marked UNPROVEN (stale-test),
not BROKEN. **Decision:** fold the test-repair into the Users Activity-3 audit / Activity-4
prove-UNPROVEN WO (repoint to `#t2`/`#s2`). Low effort, noted so it isn't lost.

## Note for the process (not a decision — carry to the post-run design batch)

Single-pass enumeration recall came in **under 90% on 4 of 5 apps** (Users 73%, Purchasing 77%,
Inventory 82.5%, Operations 85%; only Onboarding cleared at 91.2%). This strongly reinforces the
existing two-pass mandate and the sign-off's "second pass is mandatory, not optional" finding.
The consistent blind spot is **backend-only / no-UI surface** (auth enforcement, crons,
service-token contracts). Recommend the process rule name that angle explicitly. See HANDOFF
§G4 recall note. *(Deferred to the post-run design batch — night-crew stays frozen this run.)*
</content>
