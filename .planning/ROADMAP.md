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

### Phase 22: HQ Toast ingest — SFTP fetcher + menu_items + daily sales aggregate

**Goal:** Make HQ a self-contained Toast SFTP consumer so the menu_items table reflects what's actually being sold without depending on sales-processor. HQ pulls Toast SFTP reports on its own schedule, aggregates per-day sales at ingest, and stores only what it needs (menu items + units sold per day per item). Lays the schema and ingest pipeline that Phase 999.2 (recipes + COGS attribution) will build on.

**Why:** Phase 999.2 needs a live menu_items source. Phase 21 established sales-processor as the consumer of HQ data; reversing that direction (HQ depends on sales-processor) would tangle deployment topology and double the auth surface. HQ runs separately on the Windows box / Cloudflare Tunnel; sales-processor runs separately. Both can be peer SFTP consumers of Toast.

**Scope:**
1. **SFTP client** — port `sales-processor/sftp/default.go` to `backend/internal/toast/sftp.go` (same `golang.org/x/crypto/ssh` + `github.com/pkg/sftp` deps). Reuses the existing SSH key file from sales-processor (`creds/id_rsa`, user `YumYumsExportUser`, host `s-9b0f88558b264dfda.server.transfer.us-east-1.amazonaws.com:22`). Env vars: `TOAST_SFTP_KEY_PATH` (default `creds/id_rsa`), `TOAST_SFTP_USER`, `TOAST_SFTP_HOST`.
2. **CSV parser** — `backend/internal/toast/parse.go` reads `ItemSelectionDetails.csv` and emits `(toast_master_id, name, menu_group, menu, business_date, qty, gross_amount)` rows. Tolerates header drift; skips voided lines (`Void? = true`).
3. **DB migration** — `menu_items` (`id UUID PK`, `toast_master_id TEXT UNIQUE`, `name TEXT`, `menu_group TEXT`, `menu TEXT`, `last_seen DATE`, `created_at TIMESTAMPTZ`) and `daily_menu_sales` (`menu_item_id UUID FK`, `business_date DATE`, `units_sold NUMERIC`, `gross_amount NUMERIC(10,2)`, PK `(menu_item_id, business_date)`).
4. **Sync command** — `backend/cmd/sync-toast/main.go` does one SFTP pull → parse → upsert pass; idempotent (`ON CONFLICT … DO UPDATE` on `daily_menu_sales`, `DO NOTHING` for new `menu_items` other than `last_seen` bump). Cron-friendly exit code.
5. **Background scheduler** — goroutine in `cmd/server` that runs the sync once on startup and every 12 hours after. Configurable via `TOAST_SYNC_INTERVAL`; `0` disables and forces external cron.
6. **Read API** — `GET /api/v1/inventory/menu-items?since=YYYY-MM-DD` returns the current menu_items list for the Recipes tab picker (cookie-auth, not service-token — this is HQ-internal UI).
7. **Minimal UI** — Menu view in `inventory.html` (new tab or section under Setup) listing items + last-seen date + this-week units sold, so the ingest is visible without needing the full Recipes UX yet.

**Depends on:** Phase 21 (reuses chi router structure; no service-token dependency since this endpoint is cookie-auth).

**Tradeoffs:** HQ and sales-processor will both poll Toast SFTP at staggered times. Toast doesn't appear to rate-limit, but credential rotation now affects both repos. Aggregate-at-ingest means HQ can't reconstruct per-order analytics later; sales-processor remains the source of truth for that.

**Acceptance:**
- HQ server starts, runs `sync-toast`, and `menu_items` table fills with distinct items from the last N days of reports.
- `daily_menu_sales` accumulates one row per `(menu_item, date)`; re-running the sync does not duplicate rows.
- Menu view in `inventory.html` shows the items, sorted by `last_seen DESC`, with this-week units sold next to each.
- When sales-processor is stopped entirely, HQ keeps ingesting Toast data and the Menu view stays current.

**Requirements:** TBD (no formal REQ-IDs — see Acceptance bullets above as the source of truth)
**Plans:** 6 plans

Plans:
- [ ] 22-01-PLAN.md — DB migrations 0060_menu_items.sql + 0061_daily_menu_sales.sql (Wave 1)
- [ ] 22-02-PLAN.md — Port sales-processor SFTP client to backend/internal/toast/sftp.go + promote pkg/sftp + crypto/ssh to direct deps (Wave 1, parallel with 22-01)
- [ ] 22-03-PLAN.md — toast package internals: types/config/parser (TDD)/ingest orchestrator (Wave 2, depends on 22-01 + 22-02)
- [ ] 22-04-PLAN.md — toast.StartWorker (12h ticker + cold-start branch) + cmd/sync-toast CLI binary (Wave 3, depends on 22-03)
- [ ] 22-05-PLAN.md — ListMenuItemsHandler + wire toast into cmd/server/main.go (env load + worker start + chi route mount in cookie-auth group) (Wave 4, depends on 22-03 + 22-04)
- [ ] 22-06-PLAN.md — Menu tab in inventory.html (new t3/s3 between Stock and Setup) + `task sw` rebuild + human verification checkpoint (Wave 5, depends on 22-05)

### Phase 22.1: HQ Toast ingest re-architecture — Spaces-first ingest worker (no SFTP in read path); HQ-owned SFTP→Spaces sync worker + CLI; one-time sales-processor archive → DO Spaces migration CLI; local cache at backend/cache/toast/YYYYMMDD/ (INSERTED)

**Goal:** [Urgent work - to be planned]
**Requirements**: TBD
**Depends on:** Phase 22
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 22.1 to break down)

### Phase 999.1: Tab persistence on refresh (moved from Phase 18)

**Goal:** Persist active tab across page refresh for all apps using URL hash. When a tab is tapped, update `location.hash`. On page load, read the hash and activate the matching tab.
**Requirements:** TBD
**Plans:** 1 plan (needs renumbering)

## Backlog

### Phase 999.2: Per-menu-item COGS attribution via recipe/BOM mapping (BACKLOG)

**Goal:** Allocate ingredient cost from `purchase_line_items` to individual menu items so the weekly report can show "Jerked Chicken Sliders sold N units, ingredient cost $X". Builds on Phase 22's `menu_items` + `daily_menu_sales` ingest by adding a recipe/BOM table linking `purchase_items` → `menu_items` with a usage percentage per ingredient per dish.

**Approach:** Rough usage % (not per-unit BOM). Setup cost ~1 evening for the whole menu; accuracy tuned weekly via a reconciliation residual.

**Scope:**
1. **DB migration** — `recipes` (id, menu_item_id FK → `menu_items` from Phase 22, purchase_item_id FK, usage_pct NUMERIC(5,2), updated_at; unique on (menu_item_id, purchase_item_id)).
2. **Per-ingredient view (primary UI)** — Recipes tab in `inventory.html` defaults to ingredient-first: each row shows an ingredient, its week-to-date spend, and the menu items it's allocated to with their %. This is where the user edits — never per-menu-item alone — so the sum constraint is always visible.
3. **Sum constraint** — per `purchase_item_id`, `SUM(usage_pct)` cannot exceed 100. Save-time validation; UI surfaces "Unallocated: X%" and "Over-allocated: blocked" as a live running total.
4. **Weekly drift check** — after each weekly COGS compute, surface "ingredients where Unallocated > 20%" (likely missing menu item mapping) and "ingredients where actual usage diverges from your % by > 30% based on this week's mix" (suggested rebalance).
5. **HQ endpoint** — `POST /api/v1/inventory/menu-cogs` (service-token, peer of Phase 21's period-summary). Body: `{from, to, menu_items: [{toast_master_id, units_sold}]}`. Returns: `[{menu_item_name, units_sold, ingredient_cost_per_unit, ingredient_cost_total}]` plus an `unallocated_cogs` total.
6. **sales-processor consumer** — aggregates `Sale.Name → units sold` for the week, POSTs to `/menu-cogs`, renders "COGS by Menu Item" + "Unallocated COGS" rows in `WeeklySummary.Show()`. **(HANDED OFF)**

**Depends on:** Phase 22 (menu_items table + ingest pipeline) and Phase 21 (service-token auth pattern).

**Tradeoffs:** Usage % drifts with menu mix changes — the weekly drift check is the mitigation, not a fix. Per-unit BOM is more precise but the data-entry cost is ~1 week vs ~1 evening; deferred until % stops being good enough.

**Acceptance:**
- Owner opens inventory app → Recipes tab → sees ingredients sorted by week's spend, each with allocated menu items and `Unallocated: X%`.
- Trying to save a recipe that would push an ingredient's sum > 100% is blocked with a message naming which other menu item to reduce.
- Weekly report shows COGS by menu item AND an `Unallocated COGS` row.
- Drift check fires after each weekly compute, listing ingredients where the mapping looks stale.

**Out of scope:**
- Real-time ingredient consumption tracking (allocation math at report time only).
- Multi-vendor / multi-size purchases of the same ingredient (assume single canonical `purchase_item` per ingredient for now).

**Requirements:** TBD
**Plans:** 0 plans

Plans:
- [ ] TBD (promote with /gsd-review-backlog when ready)
