# HANDOFF — overnight-20260710 (for the morning of 2026-07-10)

> **✅ TRIAGED 2026-07-10** — reviewed attended, merged to `dev` `--no-ff`, all 4 PRDs
> signed (roadmap DRAFTING → DONE; Product KR-1 gate closed). Forks D-1..D-4 resolved,
> recorded in `ledger.md` (2026-07-10). D-4 graduated to `BACKLOG.md`. `dev` pushed to origin.
> **Standing flag (re-arms for Activities 2–5):** the attended E2E gate is UNRUN this cycle —
> Activities 2–5 (confirm-absence, WORKING-audit, prove-&-fix, cycle-gate) require **localhost
> Postgres + the full E2E suite green (0 pre-existing reds)**, which this enumerate-only run
> deliberately did not touch. That gate must be run before the Activity-5 cycle closeout.

> **Run:** `overnight-20260710` (cut from `dev`). **Branch never pushed; main untouched.**
> **Slate:** `.night-crew/runs/2026-07-09-attended/slate-20260710.md` (batch-signed 2026-07-09).
> **Depth:** enumerate + mark only — 4 PRD docs, no app code / no tests / no DB / no E2E.
> **Reader:** the operator, at morning triage. Start here, then `/nc-morning-triage`.

## TL;DR

All **four** Activity-1 app-hardening PRDs drafted, G6-passed, committed. **Zero cards
parked.** The Inventory split-PARK trigger did **not** fire (adjudicated KEEP-SINGLE by
drafter *and* G6 independently). Every card's Activity-1 roadmap row flipped
**PLANNED → DRAFTING** (awaiting your triage sign-off; DRAFTING → DONE happens at triage,
not overnight). 4 `docs(night-crew):` commits on the run branch, one per PRD, plus this
closeout.

## Per-card outcomes

| Card | Outcome | Tally (W / U / B = total) | Recall (single-pass) | G6 verdict | Commit |
|---|---|---|---|---|---|
| `users-hardening-prd` | ✅ drafted | 10 / 16 / 0 = **26** | 19/26 ≈ **73%** | REVISE → pass | ✅ |
| `onboarding-hardening-prd` | ✅ drafted | 23 / 11 / 0 = **34** | 31/34 ≈ **91.2%** | REVISE → pass | ✅ |
| `purchasing-hardening-prd` | ✅ drafted | 7 / 18 / 1 = **26** | 20/26 ≈ **77%** | REVISE → pass | ✅ |
| `inventory-hardening-prd` | ✅ drafted | 19 / 19 / 2 = **40** | 33/40 ≈ **82.5%** | **ACCEPT** (1st pass) | ✅ |
| **4 apps total** | 4 drafted / 0 parked | **59 / 64 / 3 = 126** | — | 3 REVISE + 1 ACCEPT | 4 |

*(For reference, the signed exemplar Operations was 10 / 17 / 0 = 27 at 85% single-pass.
Across all 5 apps: 69 WORKING / 81 UNPROVEN / 3 BROKEN = 153 enumerated requirements.)*

## G4 recall note — the headline finding of this run

**The two-pass mandate is empirically vindicated, hard.** Single-pass recall across the
four new apps: Users **73%**, Purchasing **77%**, Inventory **82.5%**, Onboarding **91.2%**.
**Three of four came in UNDER the 90% KR on a single pass** — only Onboarding cleared it.
Combined with Operations' 85%, that's **4 of 5 apps under 90% single-pass.** The cross-check
was load-bearing every time and added genuinely backend-only flows the UI-first pass is
structurally blind to:
- **Users** +7 — invite-token lifecycle, 403 role refusals, admin-or-self tier, grant round-trip (≈⅔ of Users' risk lives in backend auth enforcement).
- **Purchasing** +6 — the entire cron/scheduler surface (auto-lock, reminder, low-stock, repurchase-reset) has no UI.
- **Inventory** +7 — the two `HQ_INVENTORY_SERVICE_TOKEN` service-contract endpoints (period-summary, menu-cogs) + the Monday drift cron + single-flight invariant, none UI-surfaced.
- **Onboarding** +3 — seed idempotency, dual manager-vs-hire progress query, the reopen-video-section latent bug.

**Carry to the process:** one enumeration pass is not enough for ≥90% recall. Keep the
cross-check non-negotiable, and angle it at *backend-only / no-UI* surfaces — that's where
every miss hid this run.

## G6 verdicts (all four cleared the gate)

- **Users → REVISE→pass.** G6 caught a G3 mis-mark: FR-16/FR-17 were marked BROKEN, but the
  *flows work* — only stale E2E tests target dead DOM from a 3-tab→2-tab refactor. Re-marked
  UNPROVEN (stale-test), tally reconciled to 0 BROKEN. Fixed, re-verified, committed.
- **Onboarding → REVISE→pass.** G6 caught an internally-inconsistent tally (header said 32
  total / 19 WORKING while bodies had 34 / 23). Pure arithmetic; recounted from bodies to
  23/11/0 = 34, recall restated 31/34. No status changed. Fixed, committed.
- **Purchasing → REVISE→pass.** G6 caught a self-contradicting route count (23 claimed vs 21
  in source and in the PRD's own endpoint list). Fixed to 21 in all three places; added a
  simulate-cutoff coverage note. FR-18 BROKEN independently confirmed. Committed.
- **Inventory → ACCEPT first pass.** All five gates passed, both BROKEN marks (Trends/Cost)
  independently confirmed at cited lines, tally arithmetic exact, KEEP-SINGLE upheld
  independently. No revise loop.

## The 3 confirmed-BROKEN flows (the only ones that cleared the confirmed-only bar)

All three are genuine, cited stubs — not reputation marks:
1. **Purchasing FR-18 — History tab.** Static stub at `purchasing.html:156`; no `renderHistory`,
   no `#history-content`, no `GET /shopping/history` call in the frontend. Backend endpoint
   (FR-17) is real — the UI flow is absent. → test-repair/build WO after triage.
2. **Inventory FR-24 — Trends tab.** `renderTrends()` at `inventory.html:993-995` injects fixed
   `.coming-soon` HTML; no API/state. **Waived as unbuilt-future** (charts are net-new feature
   work, not hardening) — confirm the waiver at triage.
3. **Inventory FR-25 — Cost tab.** `renderCost()` at `inventory.html:997-999`, same as Trends.
   Same waiver.

## What's committed vs local

- **Committed to the run branch** (`docs(night-crew):`): the 4 PRD files under
  `.night-crew/knowledge/prds/`, the 4 `roadmap.md` row flips, `reference/card-actuals.md`,
  and this HANDOFF (force-added — `runs/` is otherwise gitignored).
- **Local only** (gitignored `runs/`, on disk for you): `DECISIONS-NEEDED.md`, the slate,
  sign-off, design-findings, intake.

## Next step for the operator

Run **`/nc-morning-triage`**: review the run branch, merge to `dev`, sign off each PRD to flip
its roadmap row DRAFTING → DONE (closes the 5/5 Product KR-1 gate), and resolve the
`DECISIONS-NEEDED.md` items below. Once all four are signed, Activity 1 is complete and
Activities 2–5 (confirm-absence sweeps, WORKING test-audits, prove-&-fix WOs, cycle gate)
unblock — those need localhost Postgres + the E2E suite, which this run deliberately did not.
</content>
