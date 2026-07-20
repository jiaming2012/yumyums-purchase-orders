# PRD — Prove & surface (Inventory Trends/Cost + live-sync convergence coverage)

<!-- ✅ SIGNED OFF by the operator 2026-07-19 (attended /nc-pm-session + /nc-pm-grill-back;
"keep as is and sign off"). Frozen intent contract — the cycle's blocking gate (roadmap
Activity 1). Sign-off record + accepted Assumptions: runs/2026-07-19-attended/sign-off.md.
`night-crew prd validate` → prd: valid. -->

> **Cycle:** "Prove & surface" — trust the sync, surface the numbers (opened 2026-07-19
> by `/nc-okr-session`). A MIXED cycle.
> **Role:** the cycle's **blocking gate** (roadmap Activity 1, `prd-prove-and-surface`):
> no build WO dispatches before this PRD is signed.
> **Trigger:** the just-closed "Nothing silently lost" cycle's own close note — "fold in
> the QA-coverage findings" (ledger T-17) — plus **3 operator-play escaped live-sync
> defects** of one class
> (`reference/qa-gap-20260717-live-sync-access.md`) and **2 carried PARTIALs** (Eng
> "`task test` exits 0"; Delivery "median WO cycle"). Feature scope operator-chosen at
> the OKR session: Trends = weekly-spend-by-group; Cost = margin table + movers; inline
> SVG/CSS charts (no new dep); gating = Users `app_permissions`; Cost-in-prod = accept
> sparse.
> **Two halves.** SURFACE: ship the Inventory **Trends** and **Cost** tabs (both "coming
> soon" stubs today — `inventory.html:993-998`, inline cards `:272-285`), gated through
> the Users app. TRUST: turn the escaped-defect class into a written convergence
> coverage contract, add real `sync`-package unit coverage, and retire the last test
> waiver.
> **Semantic anchor for the trust half:** the convergence contract is the
> {op-type}×{editor}×{observer/derived-view} matrix. Each of the 3 escaped defects maps
> to ≥1 matrix cell that reddens on the pre-fix build (§Escaped-defect closure).
> **Enumeration provenance (§15t two-pass):** pass-1 walked the Inventory tab UI
> (`inventory.html` tabs `#s5/#s6`, `renderTrends`/`renderCost`) and the Users grant UI;
> pass-2 swept the **no-screen** surfaces — the service-token aggregation endpoints
> (`period-summary`, `menu-cogs`), the cookie-auth-vs-service-token split, the `/me`
> app-list resolver, the absence of any permission middleware, the `daily_menu_sales`
> Toast-ingest source, `TOAST_SYNC_INTERVAL=0` in prod, the `sync` PG-`LISTEN` fan-out
> and op emitters (`EmitOp`/`EmitOpTx`), and the op-type router. Pass-2 corrected two
> OKR premises (unlinked-line handling differs between the two existing endpoints; the
> `sync` package is not literally 0-tests) — both folded below.
> **Operator Brief:** `.night-crew/runs/2026-07-19-attended/intake/operator-brief.md`.
> **Grill-back (2026-07-19):** 3 gray areas **resolved** by the operator — per-tab grants
> (the go-forward gating convention), Cost movers shown by **both** food-cost-% and margin
> dollars, unlinked-spend **excluded from group buckets + "Unlinked $X" note**. 4 delegated
> + queued survivors recorded as §Assumptions (the sign-off accepts these). No gray area
> reached sign-off without a door.

## Objective

Deliver the cycle's two commitments as one signed, traceable plan. **SURFACE:** two new
Inventory tabs — Trends (weekly spend by item-group, 12 weeks, inline chart + table)
and Cost (per-menu-item units/revenue/ingredient-cost/margin/food-cost-% + best/worst
movers ranked **both** by food-cost-% and by margin dollars) — each numerically correct
against a seeded fixture, each rendering an honest empty/low-data state where sales are
absent, and each **gated behind its own server-enforced per-tab grant** (no grant → that
tab hidden AND a distinct 403; grill-back 2026-07-19, operator: per-tab is the go-forward
convention). **TRUST:** convert the escaped
live-sync defect class into a systematic {op-type}×{editor}×{derived-view} convergence
matrix (each of the 3 escapes mapped to a would-have-caught cell), give `ResolveEntityAccess`
real role×assignment unit coverage, and retire waiver #1 so `task test` finally exits 0.
This PRD is the build denominator the **Delivery**, **Engineering**, and **QA**
objectives grade against; it advances the **Product** objective — KR1 (this PRD as the
blocking gate, 100% requirement trace — §Trace table), KR2 (each escaped defect → ≥1
matrix cell — §Escaped-defect closure), KR3 (the gating decision recorded as one signed
delegated decision), KR4 (all 12 `· new` backlog items routed — §Routing).

## Operators & users

- **Owner/admin (the numbers reader)** — the primary user of Trends and Cost. Wants to
  see where the money goes week-to-week and which menu items make or lose margin. Grants
  and revokes access to the two tabs in the Users app.
- **A granted crew member** — a non-admin the owner has given the Inventory-intelligence
  grant to; sees both tabs and their data.
- **An un-granted logged-in user** — must see **no** Trends/Cost tab and be **denied
  server-side** if they call the data endpoint directly (the negative case FR-5 enforces;
  today any authed user can call the inventory endpoints — there is no permission
  middleware).
- **The sales-processor service (indirect)** — already calls the service-token
  `period-summary`/`menu-cogs` endpoints; the new **cookie-auth** Trends/Cost endpoints
  are distinct and must not disturb those contracts.
- **Crew member / owner-editor / manager-approver (the sync-trust half)** — the same
  three roles from `PRD-data-integrity`; here they are the {editor}×{observer} axes of
  the convergence matrix. The admin/superadmin editing a checklist assigned to someone
  else is the exact untested cell that let ESC-1 escape.
- **The overnight crew (indirect)** — builds against these requirements; the PjM slates
  the cards within the signed design-before-build sequence.

## Named invariants & trace anchors

Every requirement traces to a **reproduced escape** (`ESC-n`) or a **named user-outcome
invariant** (`INV-n`); the full mapping is in §Trace table.

- **INV-A (surface honestly)** — a Trends/Cost tab renders its real data when data
  exists and an explicit empty/low-data state when it does not; it never crashes, never
  shows a spinner forever, and never shows a number it cannot back with rows.
- **INV-B (numbers are sums, not vibes)** — every Trends week×group cell equals the SUM
  of the matching line-item spend (tax-prorated consistently with `menu-cogs`); every
  Cost row's `margin = gross_amount − ingredient_cost_total` and `food_cost_% =
  ingredient_cost_total / gross_amount` match a hand-computed fixture to the cent.
- **INV-C (the gate is server-enforced, not cosmetic; per-tab)** — access to **each**
  tab is decided by the server against **that tab's own grant**: no Trends grant → the
  Trends endpoint returns a distinct 403 AND the Trends tab does not render; likewise for
  Cost, independently. Hiding a tab client-side without denying its endpoint is a
  violation (there is no permission middleware today — `main.go` confirms only
  logged-in-vs-not + ad-hoc `isAdmin()`). Per-tab (not one bundled grant) is the operator's
  go-forward convention (grill-back 2026-07-19).
- **INV-D (convergence — the escaped-defect class)** — a live op reaches **every
  entitled device** (assignees ∪ admins/superadmins ∪ the author) and each receiving
  device **reconciles every derived view** (field value, correction banner,
  edit-vs-readonly mode, list progress count), not just the raw field. A live op that
  arrives but leaves a derived view rendering stale cache is a violation.
- **INV-E (reversible by construction)** — any schema migration this cycle carries a
  proven up→down→up down-migration and a pre-deploy backup; no irreversible schema change
  reaches prod.
- **ESC-1** — cross-user live-sync access (2026-07-17): an admin/superadmin's own edits
  fanned out to assignees only and never reached their own 2nd device
  (`ResolveEntityAccess` excluded non-assignee admins; fixed, `sync/access_test.go`).
- **ESC-2** — live approval-state convergence (2026-07-18): a `REJECT_ITEM` op reached
  the device but the client re-rendered from a stale `MY_SUBMISSIONS` cache — the
  rejection reason never surfaced live and the observer's list count stayed frozen on the
  submission snapshot (fixed; `sync.spec.js` `RJT-LIVE-01/02/03`).
- **ESC-3** — sub-step rejection dead-end (2026-07-18): rejecting a **sub-step** stored
  the comment but rendered it nowhere (banner drawn only at parent-field level;
  `tplFieldIds` excluded sub-step ids), and a `require_photo` on a non-photo field was an
  unsatisfiable dead-end (fixed; `workflows.spec.js` `APR-SUBSTEP-0718` / `APR-DEADEND-0718`).

## Requirements

Every requirement is falsifiable and carries its KR trace inline; the escape/invariant
trace is in §Trace table.

### Design gate

- **FR-0** — An OpenSpec design change covering (a) the **gating model** — how a
  tab-level grant is represented in `app_permissions` (dedicated slug vs a sub-permission
  column — the operator-delegated decision, FR-6) AND the net-new **enforcement path** (a
  `RequirePermission`-style check on the new cookie-auth Trends/Cost data endpoints; the
  `/me` `MeAppsHandler`-style resolver extension that drives tab visibility; the Users
  admin-UI grant surface; `inventory` slug registration in `hq_apps`); (b) the **two
  aggregation endpoints** — by-week×by-group spend (tax-proration copying
  `recipes/handler.go:74-77`; the signed NULL-`purchase_item_id` rule) and the **margin
  join** (add `SUM(daily_menu_sales.gross_amount)` revenue to the `menu-cogs` shape;
  `margin`, `food_cost_%`, movers ordering); (c) the **convergence coverage contract**
  shape (which matrix cells, which derived views) — is **operator-signed before any
  Feature build card dispatches** (auditable from ledger timestamps). *(OpenSpec change +
  sign-off)* — traces to **Delivery KR1**.

### SURFACE — Trends

- **FR-1** — A new **cookie-auth** aggregation endpoint returns total spend bucketed by
  `date_trunc('week', event_date)` × `purchase_items.group_id` over a **12-week window**
  (delegated default — makes the requirement falsifiable; changeable later), **tax-prorated
  consistently with `menu-cogs`**
  (`SUM((pli.quantity*pli.price) * COALESCE(pe.total/NULLIF(pe.total-pe.tax,0),1))`), and
  handles NULL-`purchase_item_id` (unlinked) line items **per the signed rule** (FR-6b).
  It is distinct from the service-token `period-summary` and must not alter it. Red-first
  Go test: every week×group cell = SUM of matching line-item spend on a ≥8-week/≥2-group
  seeded fixture. *(GET `/api/v1/inventory/trends`, cookie-auth, gated by FR-5)* — traces
  to **Eng KR1**, **Delivery KR2**, INV-B.
- **FR-2** — The Trends tab (`#s5`, replacing the `renderTrends` stub at
  `inventory.html:993-995`) renders an **inline SVG/CSS** weekly-spend-by-group chart + a
  table over the window. Ships with `tests/states-trends.spec.js` forcing every row of
  the §Trends State Enumeration table (empty / loading / error / populated + no-data +
  ungated), each screenshot read back and compared to the visual contract. *(`inventory.html`
  `#s5`)* — traces to **Delivery KR2**, **QA KR3**, INV-A.

### SURFACE — Cost

- **FR-3** — A **cookie-auth** margin endpoint (net-new `GET /api/v1/inventory/cost`;
  delegated over `menu-cogs?margin=true` to keep the service-token contract untouched)
  additionally selects `SUM(daily_menu_sales.gross_amount)` as revenue and computes, per
  menu item, `margin = gross_amount − ingredient_cost_total` and `food_cost_% =
  ingredient_cost_total / gross_amount`. `gross_amount` is **not currently exposed by
  `menu-cogs`** — this is net-new. It returns **two movers orderings** (grill-back
  resolution): best/worst by **food-cost-%** AND best/worst by **margin dollars**. **Zero-
  revenue rule:** when `gross_amount = 0`, `food_cost_%` is returned NULL (rendered "—"),
  never a divide-by-zero — the row still carries units + ingredient cost, so INV-B never
  shows a false number. Red-first Go test: each menu item's margin and food-cost-% match a
  hand-computed fixture **to the cent**, both movers orderings are asserted, and the
  zero-revenue row returns NULL food-cost-%. *(GET `/api/v1/inventory/cost`, cookie-auth,
  gated by FR-5)* — traces to **Eng KR2**, **Delivery KR2**, INV-B.
- **FR-4** — The Cost tab (`#s6`, replacing the cost stub at `inventory.html:997-998`)
  renders a **sortable** per-menu-item food-cost table (units / revenue / ingredient cost
  / margin / food-cost %) + **two** best/worst movers highlight strips — one by
  food-cost-%, one by margin dollars (inline SVG/CSS bars) — with an **honest empty/low-data
  state where Toast sales are absent** (`accept-sparse-prod` —
  prod `TOAST_SYNC_INTERVAL=0`, `daily_menu_sales` may be empty). Ships with
  `tests/states-cost.spec.js` forcing every §Cost State Enumeration row. *(`inventory.html`
  `#s6`)* — traces to **Delivery KR2**, **QA KR3**, INV-A.

### SURFACE — Gating

- **FR-5** — Each tab's data endpoint is **server-enforced behind that tab's own grant**
  (grill-back resolution — per-tab, not one bundled grant): a session user **without** the
  Trends grant receives a distinct **403** from the Trends endpoint AND the Trends tab does
  not render; the Cost grant gates the Cost endpoint + tab **independently** (a user may
  hold one and not the other). Wires a net-new `RequirePermission`-style check (none exists
  today), the `/me` resolver extension that drives per-tab visibility, the Users admin-UI
  grants (one toggle per tab), and the tab slugs in `hq_apps`. **0 logged-in-only bypass
  paths** — no client-only gate. Red-first **with-grant / without-grant** test pair per tab
  (Go for the endpoint 403/200; E2E for the tab render/hide), including the mixed case
  (Trends-only, Cost-hidden). *(new middleware/check + `me` + `users.html`/`inventory.html`)*
  — traces to **Eng KR3**, **Product KR3**, **QA KR4**, INV-C.
- **FR-6** — Two decisions are recorded: **(6a — queued to the design gate)** the
  `app_permissions` **representation** — two dedicated per-tab slugs (e.g.
  `inventory-trends`, `inventory-cost`) vs a per-tab sub-permission column — is settled at
  the Activity-2 design sign-off, not pre-decided here; the **observable rule is the
  acceptance contract regardless of representation** (per FR-5, per-tab). **(6b — resolved
  at grill-back)** the NULL-`purchase_item_id` (unlinked line) rule for Trends by-group
  totals is **exclude from group buckets, surface an "Unlinked $X" completeness note**
  (unlinked lines have no `group_id`; consistent with `period-summary` reporting them and
  `menu-cogs` excluding them from allocation) — money is never silently dropped from the
  weekly totals. *(PRD record + design ratification)* — traces to **Product KR3**.

### TRUST — convergence coverage

- **FR-7** — The `sync` package gets real `ResolveEntityAccess` unit coverage across
  **all {role}×{assignment} combinations** (recipient resolution unions admins/superadmins
  + assignees; the author is added by the listener — `listener.go:63-72`). **Baseline
  correction:** the package is **not** literally 0-tests — `sync/access_test.go` already
  holds 2 DB-gated tests (`TestResolveEntityAccess_AdminReceivesLiveOps`, the ESC-1
  red-first regression, + `…_EmptyWhenNoMatchingUsers`), both skipping without a test DB;
  `CheckLWW`, `EmitOp*`, `OpsSince`, `OpHandler`, and the listener have **none**. The
  requirement: extend `ResolveEntityAccess` coverage to the full role×assignment
  cartesian, and keep the ESC-1 regression proven **red against the pre-fix code**.
  *(`backend/internal/sync/access_test.go`)* — traces to **QA KR1**, ESC-1.
- **FR-8** — The convergence E2E matrix is extended from `SET_FIELD`-only to
  **{op-type ∈ field / submit / approve / reject}** × **{editor ∈ assignee /
  non-assignee-admin}** × **{derived-view ∈ field-value / correction-banner /
  edit-vs-readonly-mode / list-progress-count}**, each cell **red-first then green across
  ≥2 devices** — **0 matrix cells red at cycle end**. *(`sync.spec.js`,
  `repro-cut-task.spec.js`, `broadcast-rerender.spec.js`; `sync.js`/`workflows.html` only
  if a determinism seam is needed)* — traces to **Eng KR4**, **Product KR2**, **QA**,
  INV-D.
- **FR-9** — **Escaped-defect closure:** each of ESC-1, ESC-2, ESC-3 maps to **≥1 named
  matrix cell** (§Escaped-defect closure table) that would have caught it — the cell
  reddens on the pre-fix build. **0 escaped defects lack a would-have-caught cell.**
  *(the §Escaped-defect closure table, audited at the cycle gate)* — traces to **Product
  KR2**, ESC-1/2/3.
- **FR-10** — **Waiver-#1 retirement:** the 1 isolation-confirmed cross-test DB-pollution
  red (`tests/workflows.spec.js › approved checklist shows Approved badge and cannot be
  resubmitted` [LST-08] — `#toast` hidden in the full suite, passes in isolation) is fixed
  by isolating its state dependency, then the full suite is re-run to confirm **literal
  `task test` exit-0**, formally retiring carried waiver #1. Red-first: the full-suite red
  is the baseline. Test-only footprint (no production change). *(test isolation in
  `workflows.spec.js`)* — traces to **Eng KR5**.

### DELIVERY — process

- **FR-11** — **Per-card wall-clock instrumentation** is the invariant build-run output
  for **100% of this cycle's build cards** (the `-0718` harness-measured table as standing
  practice), so the cycle gate computes a real **median WO cycle time vs the T-14 baseline
  (N=23 / 22m28s)** — retiring the carried Delivery PARTIAL. *(run-mechanics; no product
  code)* — traces to **Delivery KR3**.
- **FR-12** — **Prod-alert-duplication guard:** with Mercury receipt worker / alert queue
  / Zoho Cliq now live in prod against the **same external accounts as dev**, either **0
  duplicate Cliq alerts** are observed over the cycle **OR** one side is demonstrably
  disabled — 0 duplicate-alert incidents left unhandled (observed on the Cliq channel,
  recorded in the ledger). *(prod ops / alert queue)* — traces to **Delivery KR4**.

### Non-functional

- **NFR-1** — Every fix/build requirement lands **red-first**: the test fails against the
  unfixed/unbuilt state, recorded in the WO record, then flips green. — traces to **QA KR2**.
- **NFR-2** — House build unchanged: static HTML + vanilla JS frontend, Go + Postgres
  backend; the charts are **inline SVG/CSS** — **no new charting library, framework, or
  dependency**. — traces to Brief hard constraint, OKR "no new dep".
- **NFR-3** — **Reversibility:** any schema migration this cycle (e.g. a gating
  sub-permission column, if FR-6a chooses one) carries a down-migration proven by an
  up→down→up cycle in the WO record, and any prod deploy including a migration banks 1
  pre-deploy DB backup artifact. — traces to **QA KR4/reversibility**, INV-E.
- **NFR-4** — **Prod parity:** both new tabs reach **production** on
  `https://hq.yumyums.kitchen` behind the gate — `task version` shows prod
  backend/frontend == local `version.go` constants (**0 version drift**), and **2/2 tabs
  are screenshot-verified on prod** (Trends showing live weekly-spend-by-group; Cost
  rendering an honest empty/low-data state where prod sales are absent). — traces to
  **Delivery KR2**.
- **NFR-5** — **Sequencing:** this PRD (Activity 1) blocks **all** build; the signed
  design (FR-0, Activity 2) blocks the **Feature** build track only; the **Trust** track
  (FR-7/8/9/10) has no design-gate dependency and may start once the PRD lands; the
  process items (FR-11/12) run alongside. — traces to **Delivery objective**, roadmap
  sequencing rule.

## Escaped-defect closure (Product KR2 — 3/3 escapes → would-have-caught cell)

| Escaped defect | Would-have-caught matrix cell (reddens on pre-fix build) | Reddened because |
|---|---|---|
| **ESC-1** cross-user access (07-17) | {op=`SET_FIELD`} × {editor=**non-assignee admin/superadmin**} × {observer=same admin's 2nd device} × {derived=field value} | pre-fix `ResolveEntityAccess` excluded non-assignee admins → op never reached the 2nd device |
| **ESC-2a** rejection reason not live (07-18) | {op=`REJECT_ITEM`} × {actor=approver} × {observer=submitter's 2nd device} × {derived=**correction banner + edit-vs-readonly mode**} | pre-fix `applyOp REJECT_ITEM` refreshed only the Approvals queue → banner never rendered on the submitter's device |
| **ESC-2b** observer count frozen (07-18) | {op=`REJECT_ITEM` then `SET_FIELD`} × {observer=admin/manager} × {derived=**list progress count**} | `getProgress` counted the frozen `submission.responses` snapshot; observer's cached submission never refreshed to `rejected` |
| **ESC-3** sub-step rejection dead-end (07-18) | {op=`REJECT_ITEM` on a **sub-step**} × {observer=submitter} × {derived=**correction banner on the sub-step row**} | pre-fix banner drew only at parent-field level; `tplFieldIds` excluded sub-step ids → nothing rendered |

All three escaped defects are already fixed; this cycle's contract is that the matrix
**cell that would have caught each** is present and reddens on the pre-fix build — the
class stops escaping to play.

## Trace table (Product KR1 — 100% of requirements)

| Req | Traces to (reproduced escape \| named invariant) | OKR KR |
|---|---|---|
| FR-0 | design gate — INV-C + FR-6 delegation | Delivery KR1 |
| FR-1 | INV-B — new by-week×by-group endpoint | Eng KR1 · Delivery KR2 |
| FR-2 | INV-A — Trends tab honest states | Delivery KR2 · QA KR3 |
| FR-3 | INV-B — margin join (revenue net-new) | Eng KR2 · Delivery KR2 |
| FR-4 | INV-A — Cost tab honest empty/low-data | Delivery KR2 · QA KR3 |
| FR-5 | INV-C — net-new server-enforced gate | Eng KR3 · Product KR3 · QA KR4 |
| FR-6 | INV-C — gating decision recorded (delegated) | Product KR3 |
| FR-7 | ESC-1 — `ResolveEntityAccess` role×assignment coverage | QA KR1 |
| FR-8 | INV-D — systematic convergence matrix | Eng KR4 · Product KR2 · QA |
| FR-9 | ESC-1/2/3 — each → ≥1 would-have-caught cell | Product KR2 |
| FR-10 | Eng debt — LST-08 isolation → literal `task test` exit-0 | Eng KR5 |
| FR-11 | Delivery instrumentation — median vs T-14 | Delivery KR3 |
| FR-12 | Delivery — prod-alert-dup guard | Delivery KR4 |
| NFR-1 | QA discipline | QA KR2 |
| NFR-2 | Brief hard constraint (no new dep) | Brief / OKR |
| NFR-3 | INV-E — reversible migrations | QA KR4 |
| NFR-4 | prod parity | Delivery KR2 |
| NFR-5 | operator-signed gate sequencing | Delivery objective |

## State Enumeration — Trends tab (`#s5`) — QA KR3

| State | Trigger | Visual contract |
|---|---|---|
| Empty | Granted user, endpoint returns 0 weeks (no confirmed purchases in window) | "No spend recorded in the last 8–12 weeks" empty card; no chart axes, no table rows; no crash |
| Loading | Tab opened, request in flight | Skeleton/spinner in `#s5`; no stale chart from a prior tab |
| Error | Endpoint 5xx / network fail | Inline error card + Retry (mirrors the existing `#s5` retry pattern at `inventory.html:621`); no blank tab |
| Populated | ≥8 weeks × ≥2 groups of confirmed spend | Inline SVG/CSS bar chart (week on x, stacked/grouped by item-group) + table; each cell = SUM of its line items (INV-B) |
| Edge: unlinked spend present | Confirmed lines with NULL `purchase_item_id` in the window | Group buckets exclude them; an **"Unlinked $X"** completeness note is shown (FR-6b) — totals never silently drop the money |
| Edge: ungated user | Logged-in user **without the Trends grant** | Tab `#s5` **not rendered**; direct Trends-endpoint call → **403** (INV-C) — independent of the Cost grant |

## State Enumeration — Cost tab (`#s6`) — QA KR3

| State | Trigger | Visual contract |
|---|---|---|
| Empty | Granted user, no `daily_menu_sales` rows in window (prod `TOAST_SYNC_INTERVAL=0`) | Honest low-data card: "No sales data yet — food-cost appears once sales sync" (accept-sparse-prod); no NaN, no divide-by-zero, no crash |
| Loading | Tab opened, request in flight | Skeleton/spinner in `#s6` |
| Error | Endpoint 5xx / network fail | Inline error card + Retry; no blank tab |
| Populated | Seeded window with sales + recipes | Sortable table (units / revenue / ingredient cost / margin / food-cost %); `margin`/`food_cost_%` match the fixture to the cent (INV-B); **two** movers strips — best/worst by food-cost-% and by margin dollars |
| Edge: sales but no recipe (unallocated) | Menu item with `daily_menu_sales` rows but no/partial recipe allocation | Row shows revenue + units but ingredient-cost/margin marked "no recipe / partial" (mirrors `menu-cogs` unallocated reason strings) — not a silent 0 |
| Edge: zero-revenue item | Menu item with units but `gross_amount = 0` (comped/free) | `food_cost_%` renders "—" (NULL), not a divide-by-zero or ∞; units + ingredient cost still shown (FR-3 zero-revenue rule) |
| Edge: ungated user | Logged-in user **without the Cost grant** | Tab `#s6` **not rendered**; direct Cost-endpoint call → **403** (INV-C) — independent of the Trends grant |

## Routing (the 12 `· new` backlog items through three doors — Product KR4)

No inbox items existed this evening (`.night-crew/inbox/` empty). The 12 `· new`
`BACKLOG.md` items are routed below (three doors; ratifying the roadmap's intended
routing table, adjusted with the two-pass pass-2 corrections). The `· new` markers are
flipped to their routed status **at sign-off** (not at draft), so grill-back can still
adjust; the target for Product KR4 is `grep -c '· new' BACKLOG.md == 0` at cycle open.

| # | Backlog item (`· new`) | Door | Disposition |
|---|---|---|---|
| 1 | Cross-user live-sync access matrix + `sync` unit coverage | **promoted** | `sync-pkg-unit-coverage` + `convergence-matrix-systematic` (Activity 3) → FR-7/FR-8/FR-9 |
| 2 | Live approval-state convergence coverage | **promoted** | `convergence-matrix-systematic` (Activity 3) → FR-8/FR-9 (ESC-2) |
| 3 | `suite-isolation-approved-checklist` (retire waiver #1) | **promoted** | `waiver1-isolation-fix` (Activity 3) → FR-10 (operator chose graduate 2026-07-19) |
| 4 | Per-card wall-clock instrumentation (standing output) | **promoted** | `percard-timing-instrumentation` (Activity 3) → FR-11 |
| 5 | Gate run-mechanics: `CI=1` + explicit pre-migration | **folded** | rides `percard-timing-instrumentation` / cycle-gate run-mechanics (FR-11) — process, not a KR of its own |
| 6 | Transactional op emission for Create/Archive (INV-1 parity) | **deferred** | small editprop tidy-up; stays BACKLOG (OKR note) — revisit if it rides a sync card; reason recorded |
| 7 | Fail-note conflict live-render on `applyOp`/409 (`_fail_note` unpack) | **deferred** | out-of-footprint (needs `_fail_note` unpack on the apply path); BACKLOG tidy-up; reason recorded |
| 8 | Atomic approval + feedback (`approveSubmission` tx) | **deferred** | small editprop tidy-up; BACKLOG, not a KR this cycle; reason recorded |
| 9 | Onboarding persistence tests: `waitForResponse` over fixed wait | **deferred** | low-priority test-hardening; BACKLOG; reason recorded |
| 10 | Runner — failed photo upload leaves a partial saved value | **deferred** | stale-state hygiene, off-theme (no durable loss); BACKLOG; reason recorded |
| 11 | Offline submit idempotency under IndexedDB failure (suspected) | **deferred** | needs the offline-IndexedDB harness (not built this cycle); BACKLOG; reason recorded |
| 12 | Lamport clock corruption → catch-up gap (suspected) | **deferred** | same offline-harness dependency; BACKLOG; reason recorded |

**Summary:** 4 promoted (→ FR-7/8/9/10/11), 1 folded (→ FR-11 run-mechanics), 7 deferred
with a written reason. 0 dropped. All 12 accounted for.

## Glossary (domain terms pinned at grill-back)

- **Per-tab grant** — access to an Inventory-intelligence tab is its own independent
  grant (Trends grant, Cost grant), not a bundled "inventory intelligence" permission. A
  user may hold one without the other. **This is the operator's go-forward convention for
  gating going forward** (grill-back 2026-07-19), not a one-off for these two tabs.
- **Mover** — a menu item highlighted as best/worst on the Cost tab. Ambiguous in the
  OKR; pinned here to mean **both** rankings shown side by side: (1) by **food-cost-%**
  (ingredient cost ÷ revenue — efficiency), and (2) by **margin dollars** (revenue −
  ingredient cost, summed over the window — absolute earner).
- **Unlinked spend** — confirmed receipt line spend whose line has no
  `purchase_item_id` (never linked to a catalog item), hence no `group_id`. In Trends it
  is **excluded from the per-group buckets** and surfaced as a single **"Unlinked $X"**
  total so weekly totals still reconcile to actual receipts.
- **Accept-sparse-prod** — the Cost tab's prod-acceptance bar: it PASSES prod
  verification by **rendering honestly** (the low-data/empty state) where
  `daily_menu_sales` is absent (`TOAST_SYNC_INTERVAL=0`). Numeric correctness is proven on
  seeded dev fixtures, not live prod sales.

## Assumptions (grill-back survivors — the sign-off accepts these)

Every gray area surfaced at the grill-back exited exactly one door. The three **resolved**
by the operator are folded above (per-tab grants; both movers rankings; unlinked
exclude+note). The **delegated** (PM's recorded choice) and **queued** (a later decision)
survivors — which signing this batch accepts — are:

| # | Gray area | Door | Recorded resolution |
|---|---|---|---|
| A1 | Trends window length | **delegated** | 12 weeks (fixed default; makes FR-1/AC-1 falsifiable; trivially changeable) |
| A2 | Cost `food_cost_%` when `gross_amount = 0` | **delegated** | return NULL / render "—"; never divide-by-zero; row keeps units + ingredient cost (FR-3) |
| A3 | New endpoint shape | **delegated** | net-new cookie-auth `GET /inventory/trends` + `/inventory/cost`; the service-token `period-summary`/`menu-cogs` contracts held invariant |
| A4 | `app_permissions` per-tab **representation** (two slugs vs sub-permission column) | **queued** | settled at the Activity-2 design sign-off (FR-6a); the observable per-tab rule is fixed regardless |
| A5 | Prod-alert-dup remediation side | **queued** | observe the Cliq channel over the cycle (FR-12); if dupes appear, **delegated fallback**: disable the **dev-side** alert emission (prod is the live one) — recorded in the ledger |
| A6 | FR-9 "reddens on the pre-fix build" evidence | **delegated** | satisfied by each fix's **recorded historical red-first run** (ESC-1 `access_test.go`; ESC-2 `RJT-LIVE-*`; ESC-3 `APR-*`), not a fresh revert, unless a revert is cheap |
| A7 | Cost prod deliverable may be a mostly-empty screen | **confirmed** | accept-sparse-prod (operator OKR decision): shipping Cost to prod may mean shipping the honest empty state; that is an accepted PASS |

## Acceptance criteria

Surface-anchored, Given/When/Then.

- **AC-0** (FR-0) — Given the gating + endpoints + convergence-contract OpenSpec change is
  drafted, When the operator signs it, Then the sign-off timestamp precedes every
  **Feature** build WO dispatch in the ledger (audited at cycle gate); the Trust-track WOs
  may precede it.
- **AC-1** (FR-1) — Given a seeded fixture of ≥8 weeks × ≥2 item-groups of confirmed
  purchases, When `GET /api/v1/inventory/trends` is called for the 12-week window, Then
  every returned week×group cell equals the tax-prorated SUM of its matching
  `purchase_line_items`, And unlinked (NULL `purchase_item_id`) lines are excluded from the
  group buckets but returned as an "Unlinked $X" total (FR-6b), And the service-token
  `period-summary` response is byte-unchanged (red-first Go test).
- **AC-2** (FR-2) — Given the Trends endpoint returns populated data, When a granted user
  opens the Trends tab, Then `#s5` renders an inline SVG/CSS chart + table with no external
  request; And Given the endpoint returns 0 weeks, Then `#s5` shows the empty-state card,
  not a blank tab or a crash (both rows screenshot-verified in `states-trends.spec.js`).
- **AC-3** (FR-3) — Given a seeded window with `daily_menu_sales` + recipes, When
  `GET /api/v1/inventory/cost` is called, Then each menu item's `margin = gross_amount −
  ingredient_cost_total` and `food_cost_% = ingredient_cost_total / gross_amount` match the
  hand-computed fixture to the cent, And **both** movers orderings (by food-cost-% and by
  margin dollars) are asserted, And a menu item with `gross_amount = 0` returns
  `food_cost_%` = NULL (no divide-by-zero) (red-first Go test).
- **AC-4** (FR-4) — Given the Cost endpoint returns rows, When a granted user opens the
  Cost tab, Then `#s6` renders a sortable table + **two** movers strips (food-cost-% and
  margin dollars); And Given `daily_menu_sales` is empty (prod-like), Then `#s6` shows the
  honest low-data card with no NaN / divide-by-zero / crash (both rows screenshot-verified
  in `states-cost.spec.js`).
- **AC-5** (FR-5) — Given a logged-in user **without the Trends grant**, When they call
  `GET /api/v1/inventory/trends` directly, Then the server returns a distinct **403** and
  no data, And the Trends tab does not render; And Given the **same** user **holds the Cost
  grant**, Then `GET /api/v1/inventory/cost` returns **200** and the Cost tab renders — the
  two gates are independent (red-first per-tab with-grant/without-grant Go + E2E pairs,
  including the mixed Trends-only case).
- **AC-6** (FR-6) — Given the design gate, When the operator signs it, Then the chosen
  per-tab `app_permissions` representation (two slugs vs a sub-permission column) is
  recorded in the OpenSpec change (6a); And the Trends unlinked-line rule (6b:
  exclude-from-buckets + "Unlinked $X" note) is encoded such that the returned per-group
  totals plus the Unlinked total equal the window's confirmed spend.
- **AC-7** (FR-7) — Given a test DB, When the `sync` package tests run, Then
  `ResolveEntityAccess` is asserted across every {role}×{assignment} combination (recipient
  set = admins/superadmins ∪ assignees), And the ESC-1 regression
  (`TestResolveEntityAccess_AdminReceivesLiveOps`) is proven **red against the pre-fix
  code** and green after.
- **AC-8** (FR-8) — Given two devices with the same checklist open, When each cell of
  {field/submit/approve/reject} × {assignee/non-assignee-admin} × {field-value/
  correction-banner/readonly-mode/list-count} is driven on one device, Then the other
  device's corresponding derived view converges live and after reconnect — **0 red cells**
  at cycle end (red-first E2E matrix, ≥2 devices).
- **AC-9** (FR-9) — Given the pre-fix build of each of ESC-1/2/3, When its mapped matrix
  cell (§Escaped-defect closure) runs, Then the cell **reddens**; And on the current build
  it is green — 0 escaped defects without a would-have-caught cell (audited at the gate).
- **AC-10** (FR-10) — Given the full deterministic suite on an isolated pg16, When
  `task test` runs after the LST-08 isolation fix, Then it exits **0** (the prior full-suite
  red on `workflows.spec.js › approved checklist …` is the recorded red baseline), formally
  retiring waiver #1.
- **AC-11** (FR-11) — Given every build card this cycle, When it closes, Then its WO record
  carries a per-card wall-clock measurement; And at the cycle gate, Then a median WO cycle
  time is computed against the T-14 baseline (N=23 / 22m28s).
- **AC-12** (FR-12) — Given Mercury/alert-queue/Cliq live in prod, When a receipt/alert
  event occurs over the cycle, Then either 0 duplicate Cliq alerts are observed OR one side
  is demonstrably disabled — the outcome recorded in the ledger.
- **AC-NFR4** (NFR-4) — Given both tabs merged, When the operator runs `task prod:deploy`
  then `task version`, Then prod backend/frontend == local `version.go` constants (0 drift),
  And both tabs are screenshot-verified on `https://hq.yumyums.kitchen` behind the gate.

## Verification plan

- **Environment:** the ephemeral Docker pg16 stack (`docker-compose.nc.yml`) — the
  canonical local DB path (ledger 2026-07-14); Go units run against an isolated,
  pre-migrated pg16 (gate run-mechanics, FR-11 / backlog item 5).
- **Red-first protocol (NFR-1):** every fix/build card records the failing run before the
  work — bug-fix protocol per CLAUDE.md; only the new tests run during iteration, full
  suite at card close.
- **Numeric correctness (FR-1/FR-3):** red-first Go tests seed a known fixture and assert
  every aggregation cell against a hand-computed value (to the cent for margin);
  tax-proration is asserted to match `menu-cogs`; the service-token `period-summary`/
  `menu-cogs` responses are asserted unchanged (regression guard on the existing contracts).
- **Gate (FR-5):** a with-grant/without-grant Go test pair on the endpoint (403 vs 200) +
  an E2E pair on tab render/hide; a negative test proves the **direct endpoint call** is
  denied for an ungated user (no client-only bypass).
- **Convergence matrix (FR-8/FR-9):** two Playwright contexts against one DB; rows =
  {SET_FIELD, SUBMIT_CHECKLIST, APPROVE_ITEM, REJECT_ITEM} × {assignee editor,
  non-assignee-admin editor} × {field-value, correction-banner, edit-vs-readonly,
  list-progress-count}; each cell asserts the **second device's** observed derived view,
  proven red-first on the pre-fix build for the ESC-1/2/3 cells.
- **Sync unit coverage (FR-7):** `sync/access_test.go` extended to the full role×assignment
  cartesian; the DB-gated skip is documented (needs `DB_TEST_URL`/`TEST_DATABASE_URL`).
- **Self-verification ritual (FR-2/FR-4):** `states-trends.spec.js` / `states-cost.spec.js`
  force each State-Enumeration row (fixture/mock/DB seed), screenshot each, and the PNGs are
  read back and compared row-by-row to the visual contract — reported as *observed*, not
  intended.
- **Verifier subagent gate (UI phases):** per CLAUDE.md, one verifier subagent scores each
  `done_when`/State-Enumeration row from the spec + diff + screenshots only, before any
  Trends/Cost SUMMARY.md is written.
- **Cycle-end gates:** `task test` exit 0 on the deterministic stack (FR-10 retires waiver
  #1); `task version` prod == local (NFR-4); median WO cycle time computed (FR-11).

## Out of scope

- **Enabling Toast sales sync in production** — the Cost tab accepts thin prod data
  (`TOAST_SYNC_INTERVAL=0` stays); enabling Toast is not a dependency this cycle.
- **Pre-deciding the fine-grained permission representation** — FR-6a is *recorded* here
  and *settled* at the Activity-2 design sign-off (slug vs sub-permission column).
- **The small editprop follow-ups** (transactional op emission for Create/Archive, atomic
  approval+feedback, fail-note conflict live-render) — deferred to backlog (routing items
  6–8), not requirements this cycle.
- **The deferred harness/fixture work** — photo-S3 harness, offline-IndexedDB harness,
  onboarding video fixture (routing items 11–12 depend on the offline harness).
- **Net-new crew-facing features** and the other apps beyond the carried convergence work —
  this cycle surfaces Inventory numbers and hardens sync.
- **Changing the service-token `period-summary`/`menu-cogs` contracts** — the new tabs use
  distinct cookie-auth endpoints; the sales-processor contracts are held invariant.

## Success metrics

- **Trends/Cost live in prod behind the gate:** 2/2 tabs on `https://hq.yumyums.kitchen`,
  `task version` prod == local (0 drift), both screenshot-verified (NFR-4).
- **Numbers are sums, not vibes:** 100% of Trends week×group cells = SUM of their line
  items; 100% of Cost margins/food-cost-% match the fixture to the cent (INV-B).
- **The gate is real (per-tab):** 100% of ungated direct endpoint calls return 403, 0
  return 200, for **each** tab independently (incl. the mixed Trends-only/Cost-hidden
  case); 0 logged-in-only bypass paths (INV-C).
- **Convergence matrix:** 0 red cells at cycle end; 3/3 escaped defects carry a
  would-have-caught cell that reddens on the pre-fix build (Product KR2).
- **`sync` coverage:** `ResolveEntityAccess` asserted across the full role×assignment
  cartesian; ESC-1 regression proven red-first vs pre-fix.
- **`task test` exit 0** on the deterministic stack — waiver #1 formally retired (Eng KR5).
- **Cycle time measured, not narrated:** a real median WO cycle time computed vs T-14
  (N=23 / 22m28s); 100% of build cards per-card timed (Delivery KR3).
- **Routing complete:** `grep -c '· new' BACKLOG.md == 0` at cycle open — 12/12 routed
  (Product KR4).
- **Reversibility:** 0 irreversible schema changes reach prod; any migration proven
  up→down→up + 1 pre-deploy backup (INV-E, QA KR4).
