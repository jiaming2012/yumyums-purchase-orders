# Sign-off — attended PM session + grill-back, 2026-07-19

**Status:** ✅ SIGNED OFF by operator (reply "keep as is and sign off", 2026-07-19 evening).
**Run:** `2026-07-19-attended` (evening PM session of the "Prove & surface" cycle, opened
this morning by `/nc-okr-session`).

## What is being signed

The **"Prove & surface" PRD** as this cycle's frozen intent contract — the blocking gate
(roadmap Activity 1, `prd-prove-and-surface`) the overnight planner builds work orders from.

- PRD: `.night-crew/knowledge/prds/PRD-prove-and-surface.md` (`night-crew prd validate`
  → `prd: valid`; re-validated after the grill-back edits).
- OKRs: unchanged this session, `night-crew okr validate` → `okrs: valid`.
- Operator Brief: `.night-crew/runs/2026-07-19-attended/intake/operator-brief.md`
  (synthesized from the signed OKRs; no inbox this evening).
- Shape: **13 FRs (FR-0…FR-12) + 5 NFRs**, two halves — **SURFACE** (Trends
  weekly-spend-by-group endpoint + tab; Cost margin endpoint + tab; per-tab
  server-enforced gating) and **TRUST** (`sync` unit coverage; systematic
  {op-type}×{editor}×{derived-view} convergence matrix; escaped-defect→cell mapping;
  waiver-#1 retirement) + 2 process items (per-card timing; prod-alert-dup guard).
- Traceability: 100% of requirements trace to a reproduced escape (ESC-1/2/3) or a named
  invariant (INV-A…E), and to a Product/Delivery/Engineering/QA KR (§Trace table).
- Escaped-defect closure: 3/3 operator-play escapes mapped to a would-have-caught matrix
  cell (§Escaped-defect closure).
- State Enumeration: per-tab tables for Trends and Cost (empty/loading/error/populated +
  ≥2 edge rows incl. unlinked-spend, zero-revenue, ungated).

## Grill-back resolutions (3 resolved by the operator, folded + re-validated)

1. **Per-tab grants** — Trends and Cost each gated by their own independent grant (a user
   may hold one without the other). Recorded as the operator's **go-forward gating
   convention**, not a one-off (glossary; INV-C; FR-5/FR-6a). Saved to project memory.
2. **Movers = both rankings** — the Cost tab shows best/worst by **food-cost-%** AND by
   **margin dollars**, side by side (FR-3/FR-4).
3. **Unlinked spend** — excluded from Trends group buckets, surfaced as an **"Unlinked
   $X"** note so weekly totals reconcile to actual receipts (FR-6b).

## Assumptions accepted (the grill survivors — delegated / queued)

Signing off = accepting these:

- **A1 (delegated)** — Trends window = **12 weeks** (fixed default; makes FR-1 falsifiable).
- **A2 (delegated)** — Cost `food_cost_%` when `gross_amount = 0` → **NULL / "—"**, never
  divide-by-zero; row keeps units + ingredient cost.
- **A3 (delegated)** — new **cookie-auth** `GET /inventory/trends` + `/inventory/cost`
  endpoints; the service-token `period-summary`/`menu-cogs` contracts held invariant.
- **A4 (queued → Activity-2 design gate)** — the `app_permissions` per-tab
  **representation** (two dedicated slugs vs a sub-permission column) is settled at the
  design sign-off, **not** tonight. The observable per-tab rule is fixed regardless. (Kept
  queued at operator instruction, 2026-07-19 — it is the design gate's call and no
  requirement depends on it; it also decides migration-vs-no-migration against the
  reversibility KR.)
- **A5 (queued → observe; delegated fallback)** — prod-alert-dup: observe the Cliq channel
  over the cycle (FR-12); if duplicates appear, disable the **dev-side** emission. (Kept
  queued at operator instruction — the trigger is empirical/future; the fallback action is
  pre-decided.)
- **A6 (delegated)** — FR-9 "reddens on the pre-fix build" is satisfied by each fix's
  recorded historical red-first run (ESC-1 `access_test.go`; ESC-2 `RJT-LIVE-*`; ESC-3
  `APR-*`), not a fresh revert unless cheap.
- **A7 (confirmed)** — accept-sparse-prod: shipping Cost to prod may mean shipping the
  honest empty state where `daily_menu_sales` is absent — an accepted PASS.

0 gray areas reached this table without a door; 0 forks queued to morning triage beyond
A4/A5 (both with a named later owner).

## Routing (Product KR4 — 12/12 `· new` backlog items routed)

Ratified at sign-off and mirrored into `BACKLOG.md` (`grep -c '· new' BACKLOG.md` → **0**):
**4 promoted** (`sync-pkg-unit-coverage`, `convergence-matrix-systematic`,
`waiver1-isolation-fix`, `percard-timing-instrumentation`), **1 folded** (gate
run-mechanics → rides `percard-timing-instrumentation`), **7 deferred** with a written
reason (editprop tidy-ups ×3, low-priority test-hardening, stale-state hygiene,
offline-harness-dependent ×2). **0 dropped.** Authoritative record: PRD §Routing.

## After sign-off

- Safe clear-point: this conversation may now be cleared.
- Next: `/nc-slate-plan` sizes the night. The **Trust track** (Activity 3: FR-7/8/9/10)
  has no design-gate dependency and may start as soon as the PRD lands; the **Feature
  track** (Activity 4) is serialized after the **Activity-2 design sign-off** (FR-0).
- The Activity-2 design sign-off is a **separate, second attended gate** — tonight's
  signature does not pre-sign the design (it carries A4's representation decision).
