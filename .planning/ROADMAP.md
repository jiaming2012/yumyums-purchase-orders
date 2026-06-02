# Roadmap: Yumyums HQ

## Milestones

- ✅ **v1.0 Operations Console MVP** — Phases 1-5 (shipped 2026-04-14) — [Archive](milestones/v1.0-ROADMAP.md)
- ✅ **v1.1 Inventory App** — Phases 6-8 (shipped 2026-04-14) — [Archive](milestones/v1.1-ROADMAP.md)
- ✅ **v2.0 Backend** — Phases 9-13 (shipped 2026-04-19) — [Archive](milestones/v2.0-ROADMAP.md)
- ✅ **v2.1 Onboarding Video Upgrade** — Phase 1 (shipped 2026-04-20) — [Archive](milestones/v2.1-ROADMAP.md)
- ✅ **v3.0 Purchase Orders & Shopping Lists** — Phases 14-17 (shipped 2026-04-23) — [Archive](milestones/v3.0-ROADMAP.md)

## Phases

<details>
<summary>✅ v1.0 Operations Console MVP (Phases 1-5) — SHIPPED 2026-04-14</summary>

- [x] Phase 1: Template Builder (3/3 plans) — completed 2026-04-13
- [x] Phase 2: Fill-Out and Conditional Logic (2/2 plans) — completed 2026-04-13
- [x] Phase 3: Photo, Approval, and Integration (2/2 plans) — completed 2026-04-13
- [x] Phase 4: Onboarding App (3/3 plans) — completed 2026-04-14
- [x] Phase 5: Onboarding Builder (2/2 plans) — completed 2026-04-14

</details>

<details>
<summary>✅ v1.1 Inventory App (Phases 6-8) — SHIPPED 2026-04-14</summary>

- [x] Phase 6: Foundation and History Tab (2/2 plans) — completed 2026-04-14
- [x] Phase 7: Stock and Reorder Tab (2/2 plans) — completed 2026-04-14
- [x] Phase 8: Trends and Cost Intelligence Tabs (2/2 plans) — completed 2026-04-14

</details>

<details>
<summary>✅ v2.0 Backend (Phases 9-13) — SHIPPED 2026-04-19</summary>

- [x] Phase 9: Foundation + Auth (4/4 plans) — completed 2026-04-15
- [x] Phase 10: Workflows API (5/5 plans) — completed 2026-04-15
- [x] Phase 10.1: Cross-Device State Sync (5/5 plans) — completed 2026-04-17
- [x] Phase 10.2: Reactive Sync Framework (3/3 plans) — completed 2026-04-17
- [x] Phase 11: Onboarding + Users Admin (6/6 plans) — completed 2026-04-18
- [x] Phase 12: Inventory + Photos + Tile Permissions (6/6 plans) — completed 2026-04-18
- [x] Phase 13: Integration Fixes (2/2 plans) — completed 2026-04-19

</details>

<details>
<summary>✅ v2.1 Onboarding Video Upgrade (Phase 1) — SHIPPED 2026-04-20</summary>

- [x] Phase 1: Onboarding Video Upgrade (3/3 plans) — completed 2026-04-20

</details>

<details>
<summary>✅ v3.0 Purchase Orders & Shopping Lists (Phases 14-17) — SHIPPED 2026-04-23</summary>

- [x] Phase 14: PO Backend + Order Form (2/2 plans) — completed 2026-04-22
- [x] Phase 15: Notion Catalog Seed (2/2 plans) — completed 2026-04-22
- [x] Phase 16: Cutoff, Approval, and Shopping List (5/5 plans) — completed 2026-04-22
- [x] Phase 17: Alerts, Notifications, and Repurchase Badges (5/5 plans) — completed 2026-04-23

</details>

## Active

### Phase 18: Add photos to onboarding checklists

**Goal:** Add a `photo` item type to the onboarding system so crew members can capture and upload photos as part of their training checklists. Includes Builder support (+ Photo button), My Trainings photo capture/upload UI, Manager read-only photo viewing, backend progress tracking with photo URL storage, and section progress counting.
**Requirements:** TBD
**Plans:** 2 plans

Plans:
- [x] 18-01-PLAN.md — DB migration + backend Go changes (value column, photo type in SQL filters, SaveProgress/GetHireTraining updates)
- [ ] 18-02-PLAN.md — Frontend photo support in onboarding.html (CSS, camera capture, upload, thumbnails, Builder + Photo button, progress counting)

### Phase 19: Tab persistence on refresh

**Goal:** Persist active tab across page refresh for all apps using URL hash. When a tab is tapped, update `location.hash`. On page load, read the hash and activate the matching tab.
**Requirements:** TBD
**Plans:** 0 plans

### Phase 20: Require store location before adding item to PO

**Goal:** Items without a store_location should appear in the catalog/item picker under an "Unassigned" section but be blocked from being added to a purchase order until a store location is set. This prevents shopping list items from having no location context.
**Depends on:** Phase 16 (Shopping List)
**Requirements:** TBD
**Plans:** 0 plans

### Phase 21: COGS in sales-processor report + receipt completeness gate before payroll

**Goal:** Display tax-excluded COGS (SUM(qty*price) on `purchase_line_items` joined to `purchase_events` in the weekly range) on the `sales-processor` weekly PDF/CSV report, and hard-fail payroll generation (PDF, OnPay CSV, Mercury transfers) when HQ receipts for the period are not fully ingested + reviewed + catalog-linked. `--force-payroll` flag bypasses for emergencies.

**Scope:**
1. HQ endpoint `GET /api/v1/inventory/period-summary?from=&to=` returning cogs_excl_tax, cogs_incl_tax, purchase_event_count, and completeness block (`ready` bool + pending_review IDs + unlinked line_item IDs). No live Mercury fetch.
2. HQ bearer-token check via new env var `HQ_INVENTORY_SERVICE_TOKEN` so sales-processor can call without a user session.
3. `sales-processor/service/external/hq.go` HTTPClient + GetPeriodSummary. **(HANDED OFF to sales-processor team — see 21-SALES-PROCESSOR-CONTRACT.md; no HQ-side PLAN.md)**
4. `COGS` and `COGSInclTax` on `WeeklySummary`, rendered in `Show()` after Net Sales. **(HANDED OFF)**
5. `--force-payroll` flag + gate before `writePDF` / OnPay CSV / Mercury transfers. **(HANDED OFF)**
6. Tests: HQ integration test on period-summary (ready=true + each not-ready path); sales-processor unit test on gate decision with mocked HQClient. **(HQ portion in Plan 03)**
7. Docs: update `hq/CLAUDE.md` receipt pipeline section + `sales-processor/README.md` env vars / flag. **(HQ portion in Plan 03; sales-processor portion HANDED OFF)**

**Acceptance:**
- Week with unconfirmed `pending_purchases` → sales-processor exits non-zero with blocker IDs listed.
- Same week + `--force-payroll` → proceeds, PDF includes COGS lines.
- Fully-ingested week → exits 0, PDF includes COGS lines.

**Constraints:**
- sales-processor and hq/backend run against **different** Postgres instances → must communicate via HTTP.
- Discarded `pending_purchases` (`discarded_at IS NOT NULL`) count as resolved.
- Date range: `pending_purchases.created_at::date BETWEEN from AND to`.
- Per-menu-item COGS attribution (recipe/BOM mapping) is **out of scope** — deferred to a future phase.

**Depends on:** Nothing — the receipt pipeline and `purchase_events` / `purchase_line_items` schemas already exist (shipped v1.1).
**Requirements:** TBD (no formal REQ-IDs — see roadmap Acceptance bullets as the source of truth)
**Plans:** 3/3 plans complete

Plans:
- [x] 21-01-PLAN.md — Add PeriodSummary types and PeriodSummaryHandler to internal/inventory (Wave 1)
- [x] 21-02-PLAN.md — Add ServiceTokenMiddleware + unit tests to internal/auth (Wave 1, parallel with 21-01)
- [x] 21-03-PLAN.md — Wire route into main.go, integration tests against hq_test DB, update CLAUDE.md (Wave 2, depends on 21-01 and 21-02)
- ✋ 21-SALES-PROCESSOR-CONTRACT.md — HTTP contract for the sales-processor team (no PLAN.md — separate repo, hand-off document)

### Phase 999.1: Tab persistence on refresh (moved from Phase 18)

**Goal:** Persist active tab across page refresh for all apps using URL hash. When a tab is tapped, update `location.hash`. On page load, read the hash and activate the matching tab.
**Requirements:** TBD
**Plans:** 1 plan (needs renumbering)

## Backlog

### Phase 999.2: Per-menu-item COGS attribution via recipe/BOM mapping (BACKLOG)

**Goal:** Allocate ingredient cost from `purchase_line_items` to individual menu items so the weekly report can show "Jerked Chicken Sliders sold N units, ingredient cost $X". Requires a recipe/BOM table linking `purchase_items` → `menu_items` with a usage percentage (or per-unit consumption) per ingredient per dish.

**Scope:**
1. HQ DB migration: `menu_items` (id, name, toast_item_id, active, created_at) + `recipes` (id, menu_item_id FK, purchase_item_id FK, usage_pct OR units_consumed_per_serving, unit; unique on (menu_item_id, purchase_item_id)). Seed `menu_items` from existing distinct `Sale.Name` values in sales-processor DB.
2. HQ endpoint `GET /api/v1/inventory/menu-cogs?from=&to=` taking per-menu-item units-sold map and returning `[{menu_item_name, units_sold, ingredient_cost_per_unit, ingredient_cost_total}]`.
3. UI: new Recipes tab/subview in `inventory.html` for managing usage % per ingredient per menu item. Reuse the existing `pending_purchases` catalog item picker pattern.
4. `sales-processor` extension: aggregate `Sale.Name` → units sold for the week, POST to `/menu-cogs`, render "COGS by Menu Item" breakdown in `WeeklySummary.Show()`.
5. Show menu items with no recipe configured as a residual "Unallocated COGS" row (warn, don't block) so the gap is visible.

**Depends on:** Phase 21 (reuses period-summary endpoint patterns, HQ service-token auth, sales-processor HTTPClient pattern).

**Tradeoffs:** Usage % is faster to set up but accuracy drifts with case-size changes. Per-unit consumption (e.g. "1 slider = 0.05 lb chicken") is precise but needs unit conversions. Start with %; upgrade later if numbers don't reconcile to total purchase cost.

**Acceptance:**
- Owner can open inventory app → Recipes tab → pick a menu item → add ingredients with usage %.
- Weekly report shows COGS by menu item AND a residual "Unallocated COGS" row that approaches zero as recipes fill in.
- Sum of menu-item COGS + unallocated equals total purchase-line-item COGS for the week (sanity check).

**Out of scope:**
- Real-time ingredient consumption tracking (allocation math at report time only).
- Multi-vendor / multi-size purchases of the same ingredient (assume single canonical `purchase_item` per ingredient for now).

**Requirements:** TBD
**Plans:** 0 plans

Plans:
- [ ] TBD (promote with /gsd-review-backlog when ready)
