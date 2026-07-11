# PRD — Inventory app hardening

> **Cycle:** HQ hardening — first night-crew guinea-pig run.
> **App:** Inventory — `inventory.html` (7 tabs, ~2498 lines), `/api/v1/inventory/*`
> (Go packages `internal/inventory` + `internal/recipes`), Postgres. The **largest
> and riskiest** app in the cycle: it spans a Mercury→Claude receipt-ingest pipeline,
> an item/vendor catalog, stock-count overrides, and a recipe/BOM COGS-attribution
> system with a weekly drift-check cron and two service-token endpoints consumed by
> an external service (sales-processor).
> **Depth:** *Enumerate + mark only.* This PRD is the honest flow map with per-flow
> status and a falsifiable definition of "working." It does **not** fix anything —
> each BROKEN/UNPROVEN flow becomes a work order after sign-off.
> **Role of this doc:** the Inventory instance of the pattern set by the Operations
> exemplar (`PRD-operations-hardening.md`). Same section shape; grouped by the 7 tabs
> plus a cross-cutting/backend section.
> **Enumeration provenance (for the ≥90% recall KR):** Pass 1 read `inventory.html`
> (7 tabs) + the Go router in `cmd/server/main.go` (the actual registered endpoint
> set — **41 routes**, enumerated in §Verification) + `tests/inventory.spec.js`
> (~135 tests) + `tests/recipes.spec.js` (18 tests). **Pass 2 — an independent
> cross-check (G4 gate) angled at the named blind spots** — the backend-only logic
> (`sync-receipts` Mercury pull, `reprocess-all`, the Monday-09:00 drift-check cron)
> and the two service-contract endpoints (`GET /period-summary`, `GET /menu-cogs`,
> Bearer-token-guarded, unset→503, **no UI surfaces them**) — added the flows a
> UI-first pass structurally cannot see. Recall note in §Success metrics.
> **Scope note (mirrors the exemplar's G2 resolution):** "critical flows" is read as
> **all** real flows — a flow is in-scope if a real person (crew/manager/admin)
> triggers it, OR an external service (sales-processor) or a scheduler (the cron)
> drives it. The two service endpoints and the cron have no human trigger but are
> load-bearing, so they are in-scope. Completeness of the whole set + the pass-2
> cross-check is what keeps recall honest.

## Objective

Produce a written, agreed enumeration of every end-to-end flow in the Inventory app
— from the Mercury receipt-ingest pipeline (bank pull → Claude Haiku parse → DO
Spaces upload → pending-review queue → manual confirm) through the item/vendor/group
catalog, stock-count overrides, and the recipe/BOM per-menu-item COGS attribution
(ingredient-first slider allocation, sum-per-ingredient ≤ 100 enforcement, weekly
drift check) — with each flow honestly marked **WORKING**, **UNPROVEN**, or
**BROKEN**, and each backed by a falsifiable definition of "working." This is the
largest of the cycle's five app hardening PRDs. It advances the **Product objective**:
KR-1 (5/5 apps have a hardening PRD enumerating their E2E flows, delivered as an
early blocking gate) and KR-2 (enumeration recall ≥ 90%). Its status tally is the
denominator the **Engineering** (0 known-broken flows at cycle end) and **QA** (every
flow has a real, asserting test) objectives grade against.

## Operators & users

- **Crew member / operator (team_member)** — the primary Inventory user. Reviews the
  pending-receipt queue on a phone: opens a parsed receipt, links each line item to a
  catalog item via the fullscreen picker, corrects tax, confirms when the line total
  matches the bank transaction (or discards). Records manual stock counts with a
  required reason. Browses stock levels and reorder suggestions.
- **Admin / owner** — everything the operator can do, plus admin-gated actions:
  "Reprocess All Pending," the repurchase-badge reset schedule, and (per an inline
  TODO) the future Trends/Cost gating. Edits the recipe/BOM allocation that drives
  per-menu-item COGS.
- **sales-processor (external service, indirect)** — a peer backend that calls
  `GET /period-summary` (Phase 21: weekly COGS + a receipt-completeness gate for
  payroll) and `GET /menu-cogs` (Phase 999.2: per-menu-item COGS attribution).
  Authenticates with the shared `HQ_INVENTORY_SERVICE_TOKEN` Bearer; no UI surfaces
  these. HQ is the truth source for units_sold.
- **The drift-check scheduler (indirect)** — a Monday-09:00-Chicago cron that compares
  each ingredient's allocation against the prior week, writes `drift_check_results`,
  and fires a Cliq alert; the Recipes-tab banner reads its output.
- **The overnight crew (indirect)** — consumes this enumeration + status as the
  work-order backlog; a flow marked BROKEN/UNPROVEN here is a candidate WO.

## Requirements

Status legend: **WORKING** = flow works E2E *and* a test drives it and asserts
observable DB/UI state — the test may be a Playwright E2E spec **or a real asserting
Go unit/integration test** (the WORKING mark names which layer proves it) ·
**UNPROVEN** = flow appears to work in code but no test verifies the behavior it
names (missing or vacuous test) · **BROKEN** = flow is **confirmed** incomplete/
stubbed by code inspection, or a test reveals it does not do what it claims. Every
requirement traces to an OKR key result.

**G1 resolution (confirmed-only BROKEN):** BROKEN is reserved for *confirmed*
breakage. Only the Trends and Cost tabs qualify — both are static "coming soon"
placeholders with the stub quoted at file:line. A flow I merely *suspect* stays
**UNPROVEN** with a confirm-absence step.

**A note on the Go integration tests (load-bearing for WORKING marks):** the Go tests
in `internal/inventory` and `internal/recipes` guard their bodies with
`if !dbReachable { t.Skip("DB_TEST_URL not reachable...") }`. This is an
**environment guard, not a vacuous skip** — the test asserts real DB/HTTP state when
run against the localhost Postgres the cycle mandates. Per the exemplar's rule ("a
flow with a real asserting Go test can be WORKING even if the E2E spec doesn't cover
it"), these count as WORKING and each WORKING mark below says **[Go]** or **[E2E]**.

### Tab 1 — Purchases (receipt-review pipeline)

- **FR-1** — The pending-review queue lists every unconfirmed, undiscarded pending
  purchase (badge "Needs Review", or "Missing Receipt" for
  `reason='no_attachment_on_bank_tx'`), newest first. *(GET
  `/purchases/pending`; `handler.go:603` `ListPendingPurchasesHandler`;
  `inventory.html:802-852`)* — **WORKING [Go+E2E]** (`TestListPendingPurchases_ReceiptURLsField`
  period_summary_test.go:2009 asserts the receipt_urls JSONB round-trips; INVT-03 E2E
  asserts the badge/queue renders) — traces to Product KR-1, QA KR-2.
- **FR-2** — Opening a pending card renders the fullscreen review form with pre-filled
  vendor + date + line items; the operator can add/edit line items and tax, with a
  live grand-total recalculation. *(`inventory.html:709-800, 1207-1265`)* —
  **WORKING [E2E]** (review-form-opens, add-line-item, edit-tax-updates-total tests)
  — traces to Product KR-1, QA KR-2.
- **FR-3** — Each line-item name links to a catalog item via the fullscreen picker
  modal; auto-match runs case-insensitively on form open, unlinked names show an
  orange border, and the selection persists to `pending_purchases.items` JSONB so it
  survives reload. *(PUT `/purchases/pending-items`; `handler.go:240`
  `UpdatePendingItemsHandler`; `inventory.html:1953-2090`)* — **UNPROVEN** (the E2E
  suite drives the picker + border-flip visually — "selecting item in modal changes
  border" — but **no test asserts the PUT persists + survives reload**, and
  `UpdatePendingItemsHandler` has no Go test) — traces to QA KR-1, QA KR-2.
- **FR-4** — Confirm is blocked until the parsed line total matches the bank
  transaction (±0.01) and every line item is catalog-linked; on confirm the pending
  row becomes a real `purchase_event` + `purchase_line_items`, vendor is upserted
  title-cased, and `confirmed_at`/`confirmed_by` are stamped. *(POST
  `/purchases/confirm`; `handler.go:645` `ConfirmPendingPurchaseHandler`, total-match
  at :731, empty-items gate at :709)* — **WORKING [Go+E2E]**
  (`TestConfirmPending_RejectsTotalMismatchWith422` :1674,
  `TestConfirmPending_RejectsEmptyItemsWhenReceiptAttached` :1590,
  `TestConfirmPending_AcceptsEmptyItemsWhenNoAttachment` :1632 assert the DB gates;
  E2E asserts the disabled/enabled Confirm button + the "linked to a catalog item"
  inline error) — traces to Engineering KR-1, QA KR-2.
- **FR-5** — The operator can discard a pending purchase (marks `discarded_at`, drops
  it from the queue). *(POST `/purchases/discard`; `handler.go:816`
  `DiscardPendingPurchaseHandler`; `inventory.html:1337-1350`)* — **UNPROVEN** (no Go
  test on the handler; E2E asserts the discard button is *visible* but never clicks it
  to assert the row disappears — confirm-absence step: drive discard → assert row
  gone + `discarded_at` set) — traces to QA KR-1.
- **FR-6** — "Retry parse" re-arms a stuck pending row for the worker: it clears
  `parse_error` (or, when items are present but mismatch the bank total, clears items
  and sets the "could not be parsed" reason), returning 422 `nothing_to_retry` when
  there is nothing to reset. *(POST `/purchases/pending/{id}/retry-parse`;
  `handler.go:872` `RetryParsePendingPurchaseHandler`; `inventory.html:1147-1203`)* —
  **WORKING [Go+E2E]** (6 Go tests — ClearsParseError :1763, 404 :1810, 422
  row_not_pending :1836, 422 nothing_to_retry :1871, ItemsMismatch_Accepted :1904,
  ItemsMatchTotals_StillRejected :1969; E2E asserts the parse-error line + retry
  button visibility) — traces to Engineering KR-1, QA KR-2.
- **FR-7** — Clicking "Retry parse" while no sync is running auto-triggers a sync and
  flips the row to "Reparsing…" (single `/retry-parse` + single `/sync-receipts`
  POST, button disabled). *(`inventory.html:1147-1203`)* — **WORKING [E2E]**
  (retry-parse-auto-triggers-sync test, inventory.spec.js:3254-3350) — traces to
  Product KR-1, QA KR-2.
- **FR-8** — "Sync Receipts" triggers an on-demand Mercury ingest run: single POST
  disables the button and shows "Syncing…", a 3s status poll (visibility-aware)
  tracks the run, and on completion a dismissible chip shows the processed/
  auto-added/pending/skipped counts; the running state survives a mid-run reload via
  `GET /sync-receipts/status`. *(POST `/sync-receipts` + GET
  `/sync-receipts/status`; `sync_receipts.go:53,128`; `inventory.html:422-479`)* —
  **WORKING [Go+E2E]** (`TestSyncReceipts_SingleFlight_Returns409` :116,
  `..._Goroutine_UpdatesRowToDone` :174, `..._RecoversFromPanic` :223,
  `TestSyncReceiptsStatus_ReturnsLookbackDays` :18 assert the single-flight index +
  terminal-row write + panic recovery + lookback injection; E2E asserts the button
  states, the survive-reload behavior, and the summary-chip copy) — traces to
  Engineering KR-1, QA KR-2.
- **FR-9** — "Reprocess All Pending" (admin-only) re-runs the parse/validate/persist
  pipeline for every still-pending row **from the stored DO Spaces receipt URLs, with
  no Mercury calls**, unifying legacy single-URL and multi-URL rows, single-flighted
  through `receipt_sync_runs`. *(POST `/purchases/reprocess-all`;
  `reprocess_pending.go:44` `ReprocessAllPendingHandler`; `inventory.html:490-508`)* —
  **WORKING [Go]** (`TestReprocessAllPendingHandler_QueuesPerRowProcessing` :126,
  `..._ReturnsConflictWhenSyncAlreadyRunning` :213, `..._SelectsAndRoutesRows` :257
  assert row selection, 409 single-flight, and legacy/multi-URL COALESCE routing; the
  E2E only asserts the *result chip* copy, not the trigger) — traces to Engineering
  KR-1, QA KR-2.
- **FR-10** — "View Original Receipt" opens a fullscreen overlay with an image/PDF
  carousel (PDF detected by extension, rendered in an iframe with an open-in-new-tab
  link). *(client-only, URLs pre-fetched; `inventory.html:1028-1128`)* —
  **WORKING [E2E]** (view-receipt-overlay + PDF-iframe tests,
  inventory.spec.js:1181-1246, 3125-3171) — traces to Product KR-1, QA KR-2.
- **FR-11** — The Purchases history lists confirmed purchase events (vendor + total,
  tap to expand line items), filterable by vendor and paginated. *(GET `/purchases?
  vendor_id=&page=`; `handler.go:273` `ListPurchaseEventsHandler`;
  `inventory.html:1001-1004, 862-866`)* — **UNPROVEN** (E2E asserts cards render + the
  vendor-filter dropdown exists, but the filter/pagination assertions are guarded and
  `ListPurchaseEventsHandler` has no Go test; confirm-absence step: seed 2 vendors,
  assert the filter narrows the list) — traces to QA KR-1.

### Tab 2 — Stock

- **FR-12** — The Stock tab lists aggregated stock levels grouped by item group,
  each item classified low/medium/high against the group's thresholds
  (defaults low=3/high=10), with reorder suggestions surfaced at the top. *(GET
  `/stock`; `handler.go:370` `GetStockHandler`, `ClassifyStockLevel` in stock.go)* —
  **UNPROVEN** (the *classifier* is unit-tested — `TestClassifyStockLevel` stock_test.go:5
  asserts all 7 branches — but **`GetStockHandler` itself (the SQL aggregation +
  COALESCE + badge) has no test**, and the E2E only asserts a stock-item-or-empty-state
  renders; confirm-absence step: seed purchases, assert the aggregated quantity +
  level) — traces to QA KR-1, QA KR-2.
- **FR-13** — Stock quantity uses `COALESCE(stock_count_overrides.quantity,
  SUM(line_item quantity))` — a manual override wins over the derived sum. *(GET
  `/stock`; `handler.go:384`)* — **UNPROVEN** (the COALESCE is in the query but no
  test drives an override-then-read to assert the override value is returned) —
  traces to Engineering KR-1, QA KR-1.
- **FR-14** — A manual stock count writes a `stock_count_overrides` row and **requires
  a non-empty reason** (preset chips: Counted shelf / Spoiled item / Damaged item),
  upserting on `item_description`. *(POST `/stock/count`; `handler.go:503`
  `UpdateStockCountHandler`, reason-required at :522; `inventory.html:970-983,
  1355-1437`)* — **UNPROVEN** (the handler validates `reason_required` and the UI shows
  the chips + override form, but **no test submits an override to assert the row is
  written / the reason is enforced / the count reflects in Stock** — E2E only asserts
  the override form becomes visible; confirm-absence step: POST without reason → 400,
  POST with reason → row exists) — traces to Engineering KR-1, QA KR-1.
- **FR-15** — Stock-tab navigation aids: tapping a reorder suggestion scrolls to and
  expands the stock item; "View in Setup" magic-links to the Setup tab with that item
  expanded; collapsing a group collapses its expanded children; "Expand all" opens
  every item in a group. *(client-only; `inventory.html:1366-1420`)* —
  **UNPROVEN** (collapse-group-collapses-children and expand-all E2E tests exist, but
  the reorder-scroll and View-in-Setup magic-links are untested) — traces to QA KR-1.

### Tab 3 — Menu (read-only, Toast)

- **FR-16** — The Menu tab renders Toast menu items (name, group·subgroup, last-sold,
  units-sold-this-week) read-only; empty state when the API returns `[]`. *(GET
  `/menu-items`, `toast.ListMenuItemsHandler`; `inventory.html:868-903`)* —
  **UNPROVEN** (E2E drives render via a `route.fulfill` **stub**, not the real
  endpoint — asserts the row shape but not that the live handler returns Toast data;
  the `since=YYYY-MM-DD` contract *is* asserted at inventory.spec.js:159-193) —
  traces to QA KR-1, QA KR-2.
- **FR-17** — Tapping a Menu card jumps to the Recipes tab with that menu item's cost
  summary auto-selected. *(client-only; `inventory.html:2353-2365`)* —
  **WORKING [E2E]** (menu-card-jumps-to-Recipes test, inventory.spec.js:240-268 —
  synthetic DOM but asserts tab switch + summary replacement) — traces to Product
  KR-1, QA KR-2.

### Tab 4 — Recipes (BOM / per-menu-item COGS)

- **FR-18** — The Recipes tab lists ingredients (purchase_items) as collapsed rows
  showing last-week spend + unallocated %, sorted by spend; expanding a row shows each
  menu item that uses the ingredient with a 5%-snap allocation slider. *(GET
  `/inventory/recipes`; `recipes/handler.go:335` `ListRecipesHandler` →
  `ListIngredientsWithSpend`; `inventory.html:2098-2209`)* — **WORKING [Go+E2E]**
  (`TestListRecipes_HappyPath` handler_test.go:413 + `TestListRecipes_DefaultWindow`
  :483 assert the envelope + default Chicago window; recipes.spec.js:32-71 asserts the
  `{ingredients:[…]}` shape) — traces to Product KR-1, QA KR-2.
- **FR-19** — Dragging a slider updates the chip + fill gradient live (input event, no
  PUT); on release (change event) it autosaves via `PUT /inventory/recipes/{id}`, and
  setting a slider to 0% deletes the allocation. *(PUT/DELETE `/inventory/recipes/{id}`;
  `recipes/handler.go:425,470`; `inventory.html:2367-2413`)* — **WORKING [Go+E2E]**
  (`TestUpdateRecipe_HappyPath` handler_test.go:198, `TestDeleteRecipe_RemovesRow`
  :289 assert the DB write/delete; recipes.spec.js:203-307 asserts release-fires-PUT
  and input-updates-chip-without-PUT) — traces to Product KR-1, QA KR-2.
- **FR-20** — Per-ingredient allocation is capped: the server rejects any change that
  would push the sum of `usage_pct` for one purchase_item over 100, returning 422
  `{error:"sum_exceeds_100", conflict_menu_item, conflict_pct}`; the frontend rolls
  the slider + chip back and shows an inline error. Also enforces the 5%-snap
  (non-multiple-of-5 → 422 `invalid_usage_pct`) and 0..100 range. *(POST/PUT
  `/inventory/recipes`; `recipes/handler.go:306,393,446`, `ErrSumExceeds100`;
  `inventory.html:2277-2303`)* — **WORKING [Go]** (`TestCreateRecipe_SumExceeds100_NamesLargestSibling`
  handler_test.go:105, `TestUpdateRecipe_SumExceeds100_NamesSibling` :236,
  `TestCreateRecipe_InvalidSnapReturns422` :173, `TestRepository_CreateRecipe_SumExceeds100_Rollsback`
  repository_test.go:38 assert the envelope + DB rollback; **the frontend slider-rollback
  path is UNPROVEN — no E2E drives a slider past 100 to assert the rollback + inline
  error**, see NFR-8) — traces to Engineering KR-1, QA KR-2.
- **FR-21** — "+ Add menu item" opens a fullscreen picker (90-day lookback, already-
  allocated items filtered out) and POSTs a new allocation at 5%; a menu item can be
  merged into another (re-points all recipe rows, deletes the source, cannot merge
  into self). *(POST `/inventory/recipes`, POST `/inventory/recipes/merge`;
  `recipes/handler.go:371,495`; `inventory.html:2416-2494`)* — **WORKING [Go]**
  (`TestCreateRecipe_HappyPath` handler_test.go:78, `TestCreateRecipe_DuplicateReturns409`
  :147, `TestMergeMenuItem_RePointsRows` :330, `TestMergeMenuItem_SelfReturns400` :379
  assert create/dup/merge/self-guard; the picker *UI* create-flow is UNPROVEN) —
  traces to Product KR-1, QA KR-2.
- **FR-22** — The Recipes-tab drift banner reads `GET /inventory/recipes/drift` and
  renders per-ingredient drift links that scroll to + expand the affected row; a clean
  week returns `{}` and hides the banner. *(GET `/inventory/recipes/drift`;
  `recipes/handler.go:529` `DriftBannerHandler`; `inventory.html:2121-2137`)* —
  **WORKING [Go]** (`TestDriftBannerHandler_ReturnsLatest` scheduler_test.go:246,
  `..._EmptyObjectWhenNoRows` :274 assert the latest-payload / empty-`{}` contract;
  recipes.spec.js:32-71 asserts the endpoint returns 200 JSON — but the **live-tab
  banner render is UNPROVEN**: the render test injects synthetic `DRIFT_BANNER` state
  rather than consuming the endpoint, see NFR-9) — traces to Engineering KR-1, QA KR-2.
- **FR-23** — Selecting a menu item name shows a summary card breaking its ingredient
  cost down by allocation; a clear button restores the placeholder. *(client-only;
  `inventory.html:2211-2244`)* — **UNPROVEN** (the placeholder-and-clear E2E asserts
  only the literal placeholder string round-trips, not the cost breakdown math) —
  traces to QA KR-1.

### Tab 5 — Trends · Tab 6 — Cost

- **FR-24 (Trends)** — Documented as "coming soon." **Confirmed a static stub.**
  `renderTrends()` injects a fixed `.coming-soon` block and makes **no API call and
  holds no state**. *(`inventory.html:993-995`; container + TODO comment at
  :272-277 — `<!-- TODO: gate Trends to manager+ via backend roles (INTG-01) -->`)* —
  **BROKEN (confirmed stub — unbuilt)** — traces to Product KR-1 (the tab is enumerated
  and honestly marked unbuilt; it is not a candidate fix-WO this cycle, it is a
  future feature). See §Out of scope.
- **FR-25 (Cost)** — Documented as "coming soon." **Confirmed a static stub.**
  `renderCost()` injects a fixed `.coming-soon` block, no API, no state.
  *(`inventory.html:997-999`; container + TODO comment at :279-284)* —
  **BROKEN (confirmed stub — unbuilt)** — traces to Product KR-1. See §Out of scope.

### Tab 7 — Setup (items / groups / vendors / tags)

- **FR-26** — Items: list (grouped by group + store_location), search-filter, create
  (group required, title-cased), inline-edit (name/group/store_location/
  location_in_store/photo), and merge (re-points line items, syncs descriptions to
  target, deletes source, cannot merge into self). *(GET/POST/PUT `/items`, POST
  `/items/merge`; `handler.go:1044,1073,1111,177`; `inventory.html:1446-1710`)* —
  **WORKING [E2E] (merge + create-validation) / UNPROVEN (list/edit)** — the merge
  and create-without-group paths are E2E-asserted (merge-source-deleted :1346,
  cannot-merge-self :1360, create-rejected-without-group :1418), but plain
  list/edit/search have no asserting test and none of `ListItems/CreateItem/UpdateItem`
  has a Go test. Counted **once** as UNPROVEN (the un-asserted majority governs). —
  traces to Product KR-1, QA KR-1.
- **FR-27** — Item photo upload: file → client-side JPEG convert/resize → `POST
  /api/v1/photos/upload` (path_prefix `items`) → `photo_url` stored on the item.
  *(`inventory.html:1723-1754`)* — **UNPROVEN** (upload + error path untested; E2E
  asserts only that the photo UI area renders) — traces to QA KR-1.
- **FR-28** — Groups: create (name), and edit low/high stock thresholds (validated
  low<high, both ≥0), with a live "medium range" label. *(GET/POST/PUT `/groups`;
  `handler.go:1147,1194,1222`; `inventory.html:1536-1696`)* — **UNPROVEN** (the
  *threshold API* is exercised — a PUT with a negative value returns 400
  (inventory.spec.js:1825) and the Stock tab reflects a threshold change
  (:1755) — but **the group-settings UI form save is not driven to assert
  persistence**, and no Go test covers the group handlers; confirm-absence step:
  drive the Stock-Settings form → assert the new thresholds persist) — traces to
  Engineering KR-1, QA KR-1.
- **FR-29** — Vendors: list, create (title-cased), inline-edit, and merge (re-points
  purchase_events, deletes source, cannot merge into self / 404 on unknown source).
  *(GET/POST/PUT `/vendors`, POST `/vendors/merge`; `handler.go:38,65,95,125`;
  `inventory.html:1839-1910`)* — **WORKING [E2E] (merge) / UNPROVEN (list/create/edit)**
  — merge is E2E-asserted (merge-source-deleted :1293, cannot-merge-self :1319,
  invalid-source-404 :1330); list/create/edit have no asserting test and no Go test.
  Counted **once** as UNPROVEN. — traces to Product KR-1, QA KR-1.
- **FR-30** — Tags: the tab lists tags (read-only surface feeding group tagging).
  *(GET `/tags`; `handler.go:1263` `ListTagsHandler`)* — **UNPROVEN** (no test drives
  the endpoint) — traces to QA KR-1.
- **FR-31** — Repurchase-badge reset (admin-only): view + edit the weekly reset
  schedule (day-of-week chips + time). *(GET `/purchasing/repurchase-reset`, PUT
  `/purchasing/repurchase-reset/config`; `inventory.html:1762-1821`)* — **UNPROVEN**
  (surfaced from the Inventory Setup tab but backed by the *purchasing* package; the
  edit-and-persist path is untested here; the badge itself renders in
  `GetStockHandler`'s REP-01 block, also untested) — traces to QA KR-1.

### Cross-cutting (non-functional / platform guarantees)

- **NFR-1 (name normalization contract)** — `normalizeItemName` (`cases.Title`) title-
  cases receipt/item/vendor text on confirm, item create, and vendor create; the
  frontend `titleCase()` mirrors it. *(`handler.go:22-24`)* — **UNPROVEN** (applied in
  multiple handlers but no test asserts the title-cased output; note a **confirmed
  inconsistency**: `UpdateItemHandler` at :1111 does **not** normalize on edit — a
  latent drift, flagged for the WO) — traces to Engineering KR-1, QA KR-1.
- **NFR-2 (merge guardrails)** — Every merge (vendor / item / menu item) re-points all
  FKs in a transaction, deletes the source, and refuses merge-into-self. *(`handler.go:125,177`,
  `recipes/handler.go:495`)* — **WORKING [Go+E2E]** (recipes merge Go-tested :330/:379;
  vendor + item merge E2E-tested :1293/:1346) — traces to Engineering KR-1, QA KR-2.
  *(Governs the merge halves of FR-21, FR-26, FR-29.)*
- **NFR-3 (401 auth redirect)** — A 401 on any Inventory API call redirects to
  `/login.html`. *(`inventory.html`, `invApiCall` wrapper ~:316)* — **UNPROVEN** (no
  test drives a 401 to assert the redirect) — traces to QA KR-1.
- **NFR-4 (single-flight sync)** — `sync-receipts` and `reprocess-all` share the
  `receipt_sync_runs` partial-unique-on-`status='running'` index; a concurrent trigger
  gets 409, and a panicking goroutine still writes a terminal `failed` row (no orphan
  `running`). *(`sync_receipts.go:53-119`, `reprocess_pending.go:44-193`)* —
  **WORKING [Go]** (single-flight-409 + goroutine-done + panic-recovery tests, both
  packages) — traces to Engineering KR-1, QA KR-2. *(Governs FR-8, FR-9.)*

### Additional flows — pass-2 cross-check (backend-only + service contracts)

The independent pass 2, angled at the blind spots a UI-first read cannot see, added
the following. **These are the highest-value additions** — two are consumed only by an
external service, one is a scheduler with no UI trigger, and one is the actual Mercury
pull that the whole Purchases tab depends on. The recall arithmetic is in
§Success metrics.

- **NFR-5 (period-summary service contract, Phase 21)** — `GET /period-summary?from=&to=`
  returns COGS (excl/incl tax) + a per-vendor breakdown + a **receipt-completeness
  gate** (`ready` false while any blocking pending row or unlinked line item exists in
  range) for sales-processor's weekly payroll. Bearer-guarded by
  `HQ_INVENTORY_SERVICE_TOKEN`; unset → 503; bad/reversed dates → 400. **No UI
  surfaces it.** *(`handler.go:1298` `PeriodSummaryHandler`; registered under the
  service-token group, main.go:441)* — **WORKING [Go]** (`TestPeriodSummary`
  period_summary_test.go:320 is a large multi-subtest suite: ready-true/false,
  pending-blocks, unlinked-blocks, discarded-doesn't-block, no-attachment-blocks,
  400-on-bad-date, placeholder-in-COGS, by-vendor shape/sum/order, e2e empty-items
  confirm increments COGS) — traces to Engineering KR-1, QA KR-2, Product KR-1.
- **NFR-6 (menu-cogs service contract, Phase 999.2)** — `GET /menu-cogs?from=&to=`
  returns per-menu-item COGS attribution (units_sold + ingredient_cost_per_unit +
  ingredient_cost_total, tax-prorated, units=0 → per_unit null); `?breakdown=true`
  adds per-ingredient detail + a structured unallocated breakdown. Same
  `HQ_INVENTORY_SERVICE_TOKEN` Bearer (unset → 503; wrong/missing → 401); dates
  validated. **No UI surfaces it.** *(`recipes/handler.go:44` `MenuCogsHandler`;
  registered service-token group, main.go:442)* — **WORKING [Go+E2E]**
  (`TestMenuCogs_HappyPath_SummaryMode` menu_cogs_test.go:58, `..._BreakdownMode` :119,
  `..._UnitsSoldZero_PerUnitIsNull` :188, `..._FromMissing_400` :226, `..._FromAfterTo_400`
  :242, `..._BearerMissing_401` :286, `..._ServiceTokenUnset_503` :300,
  `..._BearerWrong_401` :315; recipes.spec.js:169-199 also asserts the 401/503 gate) —
  traces to Engineering KR-1, QA KR-2, Product KR-1.
- **NFR-7 (Monday-09:00 drift-check cron)** — A 15-minute ticker gated to Monday
  09:00–09:14 Chicago computes each ingredient's drift vs the prior week, skips when
  Toast ingest is stale (<5 of 7 prior days have sales), writes an idempotent
  `drift_check_results` row (PK `week_start`, ON CONFLICT DO NOTHING), and — only when
  drift is found — enqueues a Cliq alert. **No UI trigger; the banner (FR-22) reads
  its output.** *(`recipes/scheduler.go:29-161`; started at main.go:627)* —
  **WORKING [Go]** (`TestRunDriftWeek_IdempotentOnSecondCall` scheduler_test.go:154,
  `..._SkipsWhenIngestStale` :175, `..._EnqueuesCliqWhenDriftFound` :195,
  `..._NoEnqueueWhenClean` :230, plus 4 `TestComputeDrift_*` tests :39-119 and
  `TestFormatCliqMessage_IncludesDeepLink` :133 asserting the compute + alert path) —
  traces to Engineering KR-1, QA KR-2.
- **NFR-8 (slider sum>100 rollback — frontend half of FR-20)** — When a slider
  release triggers a 422 `sum_exceeds_100`, the frontend rolls the slider value, chip
  text, and fill gradient back and shows an inline "Can't go above 100% — {conflict}
  is already at {pct}%" error. *(`inventory.html:2277-2303`)* — **UNPROVEN** (the
  server 422 is Go-tested under FR-20, but **no E2E drags a slider past the ceiling to
  assert the client-side rollback + inline error** — the highest-priority Recipes gap)
  — traces to QA KR-1, Engineering KR-1.
- **NFR-9 (drift banner live consumption — frontend half of FR-22)** — The Recipes tab
  actually fetches `/drift` on load and renders the returned payload (not just
  synthetic state). *(`inventory.html:2098-2137`)* — **UNPROVEN** (the render test
  injects `DRIFT_BANNER` synthetically; no test proves the tab consumes the live
  endpoint and renders its payload) — traces to QA KR-1.

## Acceptance criteria

Surface-anchored, Given/When/Then. These define "working" for representative flows;
every enumerated flow inherits the pattern of *drive-the-real-flow + assert-observable-
state* (the WORKING bar).

- **AC-1 (FR-4, confirm total-match):** *Given* a pending purchase whose line-item
  total differs from the bank transaction by more than $0.01, *When* the operator
  submits confirm, *Then* `/purchases/confirm` returns 422 `{error:"total_mismatch",
  line_total, bank_total}` and no `purchase_event` row is created; *When* the totals
  match and every line is catalog-linked, *Then* a `purchase_event` +
  `purchase_line_items` are written and the pending row is stamped `confirmed_at`.
- **AC-2 (FR-3, item-link persistence — UNPROVEN):** *Given* a pending purchase with
  an unlinked line item, *When* the operator picks a catalog item and reloads the
  page, *Then* the selection is still present (persisted to `pending_purchases.items`)
  and the orange unlinked border is gone.
- **AC-3 (FR-14, stock override + reason — UNPROVEN):** *Given* the stock override
  form, *When* the operator submits a count with an empty reason, *Then*
  `/stock/count` returns 400 `reason_required`; *When* a reason is supplied, *Then* a
  `stock_count_overrides` row exists and the Stock tab shows the overridden quantity
  (proving FR-13's COALESCE).
- **AC-4 (FR-20 / NFR-8, allocation cap + rollback):** *Given* a purchase_item already
  allocated to 100% across menu items, *When* a slider release would push a sibling
  over the ceiling, *Then* `PUT /inventory/recipes/{id}` returns 422
  `{error:"sum_exceeds_100", conflict_menu_item, conflict_pct}` **[Go-proven]** *and*
  the frontend rolls the slider/chip back and shows the inline error **[UNPROVEN — the
  E2E half of the AC]**.
- **AC-5 (NFR-5, period-summary completeness gate):** *Given* a date range containing
  one blocking pending row (`reason='no_attachment_on_bank_tx'`), *When* sales-processor
  calls `GET /period-summary`, *Then* `completeness.ready` is false and the row's id is
  in `pending_review_ids`; *When* that row is confirmed or discarded, *Then* `ready`
  becomes true.
- **AC-6 (NFR-5/NFR-6, service-token gate):** *Given* `HQ_INVENTORY_SERVICE_TOKEN` is
  unset, *When* either service endpoint is called, *Then* it returns 503; *Given* it is
  set, *When* the call carries a wrong or missing Bearer, *Then* 401.
- **AC-7 (NFR-7, drift cron):** *Given* two consecutive drift ticks in the Monday
  09:00 window, *When* the second fires, *Then* exactly one `drift_check_results` row
  exists for that `week_start` (idempotent) and — when drift was found — exactly one
  Cliq alert was enqueued; *Given* fewer than 5 of the prior 7 days have sales, *Then*
  no row is written.
- **AC-8 (FR-5, discard — UNPROVEN):** *Given* a pending purchase, *When* the operator
  discards it, *Then* it disappears from the queue and its `discarded_at` is set, and
  it no longer blocks the period-summary completeness gate.

## Verification plan

- **Environment:** localhost Postgres (`brew postgresql@16`) — the E2E suite and the
  Go integration tests both require a local DB. The Go tests self-skip with
  `t.Skip("DB_TEST_URL not reachable...")` when the DB is absent; **that skip is an
  environment guard, not a vacuous test** — under the mandated localhost DB they run
  and assert. Playwright blocks service workers (`serviceWorkers: 'block'`).
- **Suites:**
  - E2E: `tests/inventory.spec.js` (~135 tests) + `tests/recipes.spec.js` (18).
  - Go: `internal/inventory/{period_summary,sync_receipts,reprocess_pending,stock}_test.go`
    and `internal/recipes/{handler,menu_cogs,scheduler,repository}_test.go` — these
    are **real asserting integration tests** and are what proves most backend-only and
    service-contract flows.
  - Run per-flow during iteration (`npx playwright test <file> -g "<name>"` or `go
    test ./internal/inventory/... -run <name>`), full suite (`task test` / `go test
    ./...`) at gate.
- **This PRD specifies the test each flow needs; it does not write them (resolves
  G4).** Writing/repairing a test is itself a work order — this doc names the
  assertion, the WO delivers it.
- **What each status turns into downstream:**
  - **WORKING** flows: a **test-audit WO** — spot-check the existing test is
    non-vacuous. For the Go tests, confirm the DB-guard skip is *not* silently hiding
    a red under the localhost DB. If vacuous/hidden-red, it drops to UNPROVEN.
  - **UNPROVEN** flows: a **test-only WO first** — write a seeded, red-first assertion
    (failing when the flow is broken, passing after). The flow graduates to a **fix
    WO** only if the test goes red.
  - **UNPROVEN with a confirm-absence step** (FR-3, FR-5, FR-11, FR-12, FR-14, FR-28,
    NFR-1, NFR-8, NFR-9): the WO opens by inspecting the handler/render path; if the
    behavior is confirmed missing/stubbed it is re-marked **BROKEN** + a code-fix WO.
  - **BROKEN** flows (FR-24 Trends, FR-25 Cost): **not** fix-WOs this cycle — they are
    unbuilt future features, enumerated + marked honestly (see §Out of scope).
- **Vacuous / stale-test findings (QA KR-1):** the E2E suite has **no `test.skip` /
  `test.fixme`**, but carries **~40 data-dependent early-return guards** (e.g.
  `if (await pending.count() === 0) return;` — inventory.spec.js merge/modal/reorder
  tests) that silently pass when seeding is a no-op; and several **synthetic-state**
  Recipes tests (drift banner render at recipes.spec.js:310, summary placeholder at
  :348) that assert a literal string rather than driving the real endpoint. Each is a
  QA-KR-1 cleanup candidate.
- **Endpoints in scope (41 registered routes):**
  - **Inventory package (cookie-auth), under `/api/v1/inventory`:** GET/POST/PUT
    `/vendors`, POST `/vendors/merge`, GET/POST `/purchases`, GET `/purchases/pending`,
    POST `/sync-receipts`, GET `/sync-receipts/status`, POST `/purchases/reprocess-all`,
    POST `/purchases/confirm`, POST `/purchases/discard`, POST
    `/purchases/pending/{id}/retry-parse`, PUT `/purchases/pending-items`, POST
    `/purchases/pending-seed`, GET `/stock`, POST `/stock/count`, GET/POST/PUT
    `/items`, POST `/items/merge`, GET/POST/PUT `/groups`, GET `/tags`, GET
    `/menu-items` (toast).
  - **Recipes group (cookie-auth), under `/api/v1/inventory/recipes`:** GET `/`, POST
    `/`, PUT `/{id}`, DELETE `/{id}`, POST `/merge`, GET `/drift`.
  - **Service-token group (Bearer, no cookie):** GET `/inventory/period-summary`, GET
    `/inventory/menu-cogs`.
  - **Client-side upload + purchasing cross-links surfaced in the Setup tab:** POST
    `/api/v1/photos/upload`, GET/PUT `/api/v1/purchasing/repurchase-reset[/config]`.

### Status tally (the denominator downstream objectives grade against)

Total requirements enumerated: **40** (31 FR + 9 NFR) — 33 first-pass + 7 from the
pass-2 cross-check (see recall note). Every requirement is counted exactly once; the
three status rows below list disjoint IDs that sum to 40.

| Status | Count | Flows |
|---|---|---|
| **WORKING** | 19 | FR-1, FR-2, FR-4, FR-6, FR-7, FR-8, FR-9, FR-10, FR-17, FR-18, FR-19, FR-20, FR-21, FR-22, NFR-2, NFR-4, NFR-5, NFR-6, NFR-7 |
| **UNPROVEN** | 19 | FR-3, FR-5, FR-11, FR-12, FR-13, FR-14, FR-15, FR-16, FR-23, FR-26, FR-27, FR-28, FR-29, FR-30, FR-31, NFR-1, NFR-3, NFR-8, NFR-9 |
| **BROKEN** | 2 | FR-24 (Trends stub, inventory.html:993-995), FR-25 (Cost stub, inventory.html:997-999) |

**Sum check: 19 + 19 + 2 = 40 = total.** ✓ Each row's stated count equals the number
of IDs it lists (19 / 19 / 2), and the three disjoint sets partition the full set of
40 enumerated requirements. Recall denominator = 40.

*(19 UNPROVEN + 2 BROKEN = 21 candidate work orders. The 2 BROKEN are unbuilt future
features (Trends/Cost), not this cycle's fix backlog — so the cycle's Engineering
"0 known-broken" denominator over **built** flows is the 19 UNPROVEN, each of which
must reach WORKING or be explicitly waived. FR-20 is WORKING on its server half but
carries the UNPROVEN frontend-rollback half, tracked separately as NFR-8; FR-22 the
same, tracked as NFR-9 — no double-count, the FR sits in WORKING for its proven half
and the NFR sits in UNPROVEN for its unproven half.)*

### Activity-2 confirm-absence sweep record (2026-07-11, G6-passed)

Two-pass static audit (pass 1 UI-flow, pass 2 backend-only / service-contract
cross-check) of all 19 UNPROVEN flows against `inventory.html` +
`backend/internal/{inventory,recipes,toast}/*`; adversarial G6 stub-hunt of every no-UI
handler + the two frontend-half flows. **Result: 0 graduations — all 19 stay UNPROVEN;
FR-24/25 remain the only (waived) BROKEN.** Tally unchanged: WORKING 19 · UNPROVEN 19 ·
BROKEN 2. Every UNPROVEN flow's handler/render path has a real, present body (no stub,
no no-op render, no never-firing validation, no dangling reference) — present-but-untested.

**NFR-1 normalization ruling (G6-confirmed):** `normalizeItemName` IS called on all three
NAMED contract surfaces — CreateVendor (`handler.go:78`), CreateItem (`:1092`), Confirm
line-items (`:762`) — so the named contract is present-but-untested → UNPROVEN. **Two
latent normalization gaps flagged for the NFR-1 WO (neither a G3 BROKEN):** (1)
`UpdateItemHandler` (`:1129-1131`) writes `input.Description` raw — item *edit* doesn't
normalize (edit isn't in the named contract); (2) `ConfirmPendingPurchaseHandler`
upserts the **vendor** raw (`:660-664`) while line-items ARE normalized — FR-4's
"vendor upserted title-cased" text is inaccurate for the vendor field. Both are text-drift
gaps (behavior present, output un-normalized), folded into one NFR-1 WO note; FR-4 stays
WORKING (its total-match/empty-items gates are Go-tested; only the vendor-normalization
sub-claim is off).

| Flow | Present at | Confirm-note |
|---|---|---|
| FR-3 | `handler.go:254-267` | `UpdatePendingItems` persists items JSONB (guarded, 404 on 0 rows) |
| FR-5 | `handler.go:828-843` | discard sets `discarded_at`, drops from queue |
| FR-11 | `handler.go:286-306` | vendor-filter + `LIMIT/OFFSET` pagination real |
| FR-12 | `handler.go:380-432` | stock aggregation + group-threshold classify present |
| FR-13 | `handler.go:384` | `COALESCE(sco.quantity, sub.total_quantity)` present |
| FR-14 | `handler.go:522-529` | `reason_required` 400 gate + upsert on `item_description` |
| FR-15 | `inventory.html:969,1398` | View-in-Setup + reorder `scrollIntoView` present |
| FR-16 | `internal/toast/handler.go:43-63` | real `ListMenuItemsHandler` (live Toast+sales join, `since` validation) |
| FR-23 | `inventory.html:2223-2233` | real cost-breakdown math (`alloc = spend*(pct/100)`) + clear |
| FR-26 | `handler.go:1044,1073,1111` | ListItems/CreateItem/UpdateItem all real (list/edit half untested) |
| FR-27 | `inventory.html:1744` | `photos/upload` POST present |
| FR-28 | `handler.go:1194,1222-1242` | CreateGroup + UpdateGroup `low<high`/≥0 validation present |
| FR-29 | `handler.go:38,65,95` | ListVendors/CreateVendor/UpdateVendor real (list/create/edit half untested) |
| FR-30 | `handler.go:1263-1284` | `ListTags` real query + response |
| FR-31 | `inventory.html:1764,1816` | repurchase-reset GET + PUT config present |
| NFR-1 | `handler.go:78,762,1092` (present); `:1129-1131`, `:660-664` (gaps) | 3 named surfaces normalize; edit + confirm-vendor un-normalized → WO note |
| NFR-3 | `inventory.html:316` | 401 → `/login.html` redirect present |
| NFR-8 | `inventory.html:2277-2303` | slider/chip/gradient rollback + inline error present |
| NFR-9 | `inventory.html:2100-2106` | `loadRecipes` fetches live `/drift`, populates `DRIFT_BANNER` |

## Out of scope

- The other four apps (Operations, Onboarding, Users, Purchasing) — separate PRDs.
  Note the repurchase-badge reset (FR-31) is *surfaced* in the Inventory Setup tab but
  *implemented* in the purchasing package; it is enumerated here because a real
  Inventory user drives it, but its handler hardening belongs to the Purchasing PRD.
- **Fixing** any flow — this PRD enumerates and marks; work orders fix.
- **Building the Trends and Cost tabs (FR-24, FR-25).** They are confirmed unbuilt
  stubs and are marked BROKEN for honesty, but **standing up their charts is net-new
  feature work, not hardening** — out of scope for this cycle per the brief's
  hardening-only constraint. The hardening artifact they produce is the enumeration
  itself (they exist as tabs and must not silently appear functional).
- Any net-new feature, field type, or endpoint (hardening only).
- Changing the build (static HTML + vanilla JS front end, Go + Postgres back end; no
  framework, no new dependency).
- The internals of the receipt worker (`internal/receipt` — Mercury client, Claude
  Haiku parse, DO Spaces upload). This PRD covers the *inventory* handlers that
  trigger and consume it (FR-8, FR-9) and the pending-queue it feeds; the worker's own
  parse/upload correctness is a separate surface. The actual `RunIngestCycle` Mercury
  pull is therefore UNPROVEN-at-this-boundary (FR-8 proves the trigger/single-flight,
  not the pull's parse accuracy).

## Success metrics

- **Enumeration recall ≥ 90%** — `enumerated ÷ (enumerated + discovered-during-WO-
  build) ≥ 0.90`. Denominator: the **40** requirements above plus any flow the build
  surfaces. *(Product KR-2.)*
  - **Empirical finding (guinea-pig signal, largest surface):** pass 1 (UI + endpoints
    + specs) enumerated **33**; the pass-2 cross-check added **7** — the two
    service-token contract endpoints (NFR-5 period-summary, NFR-6 menu-cogs), the
    Monday-09:00 drift cron (NFR-7), the single-flight invariant spanning both sync
    handlers (NFR-4), the frontend halves of the two Recipes contracts (NFR-8 slider
    rollback, NFR-9 live drift consumption), and the normalization contract + its
    UpdateItem inconsistency (NFR-1). **Single-pass recall ≈ 33/40 = 82.5%, well under
    the 90% bar** — even lower than Operations' 85%, exactly as predicted for the
    largest surface. The two-pass total (40) clears it only because the cross-check ran.
    **Lesson reinforced: on a large app the UI-first pass is structurally blind to
    service-only and scheduler-only flows; the cross-check is mandatory, not optional.**
- **5/5 apps gate** — this is the Inventory PRD, 1 of 5, same shape as the exemplar.
  *(Product KR-1.)*
- **0 known-broken *built* flows** at cycle end — the 19 UNPROVEN either reach WORKING
  (a real asserting test drives them) or are explicitly waived; the 2 BROKEN stubs
  (Trends/Cost) are waived as unbuilt-future, not fixed. *(Engineering KR-1.)*
- **100% of UNPROVEN flows have a shipped WO** by cycle end. *(Delivery KR-1.)*
- **Every WORKING flow's test is non-vacuous** — including auditing that the Go tests'
  DB-guard skip is not masking a red under the localhost DB, and cleaning the ~40
  data-dependent early-return guards + the synthetic-state Recipes tests. *(QA KR-1,
  KR-2, KR-3.)*
</content>
</invoke>
