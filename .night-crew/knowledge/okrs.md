# OKRs

<!-- ✅ SIGNED OFF by the operator 2026-07-19 (attended /nc-okr-session). These OKRs stand as
the cycle's committed knowledge; the evening /nc-pm-session traces its PRD requirements to them.

Cycle: "Prove & surface" — trust the sync, surface the numbers. A MIXED cycle opened
2026-07-19 (attended /nc-okr-session) after "Nothing silently lost" closed the same day
(markdown-mode close, ledger T-17; prod live for the first time in ~2mo at backend 0.1.3 /
frontend 1.0.3, 405 commits merged to main). Two halves, both traced below:

  (1) TRUST — close the live-sync convergence QA hole. Last cycle proved convergence only for
  the assignee editing their own checklist via SET_FIELD ops; during operator play THREE escaped
  defects of the same class surfaced and were fixed reactively/red-first (cross-user access:
  sync pkg had 0 Go tests; approval-state convergence: submit/approve/reject re-rendered from
  stale cache, derived views never reconciled; sub-step rejection dead-end). This cycle makes the
  coverage systematic so the class stops escaping to play. Plus the two carried PARTIALs from
  last cycle: Eng "task test exits 0" (1 isolation-pollution red → formally retire waiver #1)
  and Delivery "median WO cycle" (per-card wall-clock instrumentation → a real computable median).
  Plus a prod-alert-dup guard now that Mercury/alert-queue/Cliq run in prod against the SAME
  external accounts as dev.

  (2) SURFACE — ship the Inventory Trends + Cost tabs (both "coming soon" stubs today,
  inventory.html:272-285 / 993-998). Operator-chosen scope (2026-07-19): TRENDS = weekly spend
  by item-group over ~8-12 wks (chart + table; new by-week×by-group aggregation endpoint).
  COST = per-menu-item food-cost table (units, revenue, ingredient cost, margin, food-cost %)
  PLUS a top/bottom movers highlight (margin join: gross_amount − ingredient_cost_total).
  Charting = inline SVG/CSS bars (NO new dependency — keeps the static/minimal-deps ethos).
  GATING = "under the Users app": access controlled by the existing app_permissions grant model
  (table + admin UI real; `inventory` slug registered) — but tab-level granularity AND real
  enforcement are net-new (grants are whole-app only today; there is NO permission middleware,
  only logged-in-vs-not). The KRs fix only the observable bar (no grant → no tab + server 403);
  the slug-vs-sub-permission modeling is a design-gate call.

  Operator decisions folded (2026-07-19 OKR session): mixed theme; feature = Trends+Cost;
  Cost-in-prod = ACCEPT SPARSE (margin proven on seeded dev fixtures; prod-acceptance = renders
  honestly with an empty/low-data state where Toast sales are absent — prod Toast is inert,
  TOAST_SYNC_INTERVAL=0, no dependency on enabling it this cycle). PM recommendations the operator
  did not override: gating modeling left to the design gate; the small editprop follow-ups
  (transactional op emission Create/Archive, atomic approval+feedback, fail-note conflict
  live-render) stay BACKLOG tidy-ups, not KRs; prod-alert-dup is one modest Delivery KR, not its
  own objective. Previous cycle archived at reference/okrs-2026-07-16-nothing-silently-lost.md +
  reference/roadmap-2026-07-16-nothing-silently-lost.md. -->

## Product

### Objective: Both halves are enumerated and specified before build — the escaped-defect class becomes a written coverage contract, and the two new tabs become falsifiable observable behaviors.
- A cycle PRD is delivered as a BLOCKING gate before any build WO, covering (a) Trends/Cost per-tab state-enumeration + `done_when` observable behaviors and (b) the live-sync convergence coverage contract (the {viewer}×{editor}×{op-type}×{derived-view} matrix); 100% of its requirements trace to either a reproduced escape or a named user-outcome (trace table in the PRD, audited at the cycle gate).
- Escaped-defect closure: each of the 3 operator-play escaped defects (cross-user live-sync access, live approval-state convergence, sub-step rejection dead-end) maps to ≥ 1 named matrix cell that would have caught it red-first — 0 escaped defects lack a would-have-caught cell (auditable table in the PRD).
- The Inventory-intelligence gating decision (tab-level gate via the Users `app_permissions` model; slug-vs-sub-permission modeling) is recorded as 1 operator-delegated, sign-off-ratified decision in the PRD, with the observable rule encoded as the acceptance contract: a user WITHOUT the grant sees no Trends/Cost tab and is denied server-side; a granted user sees both.
- 12/12 backlog items marked `· new` at cycle open (2026-07-19) are routed through a door — folded into the PRD, promoted to a roadmap card, or deferred with a written reason in BACKLOG.md; 0 `· new` markers remain (auditable via `grep -c '· new' BACKLOG.md`).

## Delivery

### Objective: The feature and the trust fixes reach production behind a signed design, and cycle-time becomes measured, not narrated.
- The cycle design (Trends/Cost + gating + convergence-coverage contract) is operator-signed BEFORE any build WO is dispatched — 0 build WOs start ahead of the signed design (auditable from ledger timestamps).
- Both new Inventory tabs reach PRODUCTION, not just dev — 2/2 tabs live behind the gate on `https://hq.yumyums.kitchen` with 0 version drift (`task version` shows prod backend/frontend == local `version.go` constants): Trends showing live weekly-spend-by-group, Cost rendering correctly with an honest empty/low-data state where Toast sales are absent (Cost margin correctness itself is proven on seeded dev fixtures, not prod data — operator: accept sparse prod). 2/2 tabs screenshot-verified on prod.
- Per-card wall-clock instrumentation is a standing build-run output for 100% of this cycle's build cards, and the cycle's median WO cycle time is computed against the T-14 baseline (N=23 / 22m28s) — retiring the carried Delivery PARTIAL (a real this-cycle median exists in the run records).
- Prod-alert-duplication guard: with Mercury / alert-queue / Zoho Cliq now live in prod against the same external accounts as dev, either 0 duplicate Cliq alerts are observed over the cycle OR one side is demonstrably disabled — 0 duplicate-alert incidents left unhandled (observed on the Cliq channel, recorded in the ledger).

## Engineering

### Objective: The two aggregation views are numerically correct, the gate is server-enforced, and the convergence matrix covers the full escape surface — with `task test` finally at exit-0.
- Trends correctness: for a seeded fixture spanning ≥ 8 weeks across ≥ 2 item-groups, every week×group cell returned by the new aggregation endpoint equals the SUM of matching line-item spend (tax-prorated consistently with `period-summary`), and NULL-`purchase_item_id` (unlinked) line items are handled per the signed rule — asserted by a red-first Go test.
- Cost correctness: for a seeded window, each menu item's `margin = gross_amount − ingredient_cost_total` and `food_cost_% = ingredient_cost_total / gross_amount` match a hand-computed fixture to the cent, and the top/bottom food-cost-% movers ordering is asserted — red-first Go test.
- Gate enforced server-side: a session user WITHOUT the grant receives a distinct 403 from the Trends/Cost data endpoint AND the tab does not render, while a granted user receives 200 + the tab — 0 logged-in-only bypass paths (no client-only gate); red-first with-grant/without-grant test pair.
- Convergence matrix systematic: the E2E matrix is extended from SET_FIELD-only to {op-type ∈ field / submit / approve / reject} × {editor ∈ assignee / non-assignee-admin} × {derived-view ∈ field-value / correction-banner / readonly-mode / list progress-count}, each cell red-first then green across ≥ 2 devices — 0 matrix cells red at cycle end.
- `task test` exits 0 on the deterministic stack — the 1 isolation-confirmed cross-test DB-pollution red (`tests/workflows.spec.js › approved checklist shows Approved badge …` LST-08) is fixed so literal `task test` = exit-0, FORMALLY retiring carried waiver #1 (Eng PARTIAL → PASS; red-first: the full-suite red is the baseline).

## QA

### Objective: The sync package gets real coverage, every fix lands red-first, the new tabs pass the self-verification ritual, and prod mutations stay reversible.
- `sync` package: 0 → covered — `ResolveEntityAccess` is tested across all {role}×{assignment} combinations (recipient resolution includes admins + author + assignees), and the escaped cross-user access defect carries a red-first unit test proven against the pre-fix code.
- 100% of this cycle's fix-WOs carry red-run evidence in the WO record (the test failed before the fix; bug-fix protocol) — denominator = all WOs classified fix.
- Feature self-verification ritual: the Trends and Cost tabs each ship a `tests/states-<tab>.spec.js` State-Enumeration spec that forces empty / loading / error / populated + ≥ 2 edge rows (no-sales-data → Cost shows an honest gap, not a crash; ungated-user → tab hidden + 403) and screenshots each row, with the PNGs read back and compared row-by-row against the visual contract — 0 State-Enumeration rows unverified.
- Reversibility: 100% of schema migrations shipped this cycle (e.g. any gating sub-permission column) have a down-migration proven by an up→down→up cycle run green in the WO record, and 100% of prod deploys that include a migration bank 1 pre-deploy DB backup artifact — 0 irreversible schema changes reach prod.
