# Roadmap: Yumyums HQ

## Milestones

- ✅ **v1.0 Operations Console MVP** — Phases 1-5 (shipped 2026-04-14) — [Archive](milestones/v1.0-ROADMAP.md)
- 🚧 **v1.1 Inventory App** — Phases 6-8 (in progress)

## Phases

<details>
<summary>✅ v1.0 Operations Console MVP (Phases 1-5) — SHIPPED 2026-04-14</summary>

- [x] Phase 1: Template Builder (3/3 plans) — completed 2026-04-13
- [x] Phase 2: Fill-Out and Conditional Logic (2/2 plans) — completed 2026-04-13
- [x] Phase 3: Photo, Approval, and Integration (2/2 plans) — completed 2026-04-13
- [x] Phase 4: Onboarding App (3/3 plans) — completed 2026-04-14
- [x] Phase 5: Onboarding Builder (2/2 plans) — completed 2026-04-14

</details>

### 🚧 v1.1 Inventory App (In Progress)

**Milestone Goal:** Standalone inventory tool showing purchase history, category spending trends, stock level estimates, and food cost intelligence — all derived from mock Baserow purchase data, no backend required.

- [x] **Phase 6: Foundation and History Tab** - inventory.html shell with Chart.js loaded locally, mock data layer, HQ integration, and fully functional Purchase History tab (completed 2026-04-14)
- [x] **Phase 7: Stock and Reorder Tab** - Low/Medium/High stock badges per item group via purchase-frequency heuristic, reorder suggestions list, manual override (completed 2026-04-14)
- [ ] **Phase 8: Trends and Cost Intelligence Tabs** - Chart.js spending charts and food cost per menu item, architected for future Metabase iframe replacement

## Phase Details

### Phase 6: Foundation and History Tab
**Goal**: Users can reach the Inventory tool from HQ, browse past purchase events with expandable line items, and filter by vendor — with Chart.js available offline and mock data rich enough to exercise all features
**Depends on**: Phase 5 (existing PWA conventions established)
**Requirements**: HIST-01, HIST-02, HIST-03, HIST-04, INTG-01
**Success Criteria** (what must be TRUE):
  1. User can tap the Inventory tile on the HQ launcher and land on inventory.html with 4 tabs visible (History / Trends / Stock / Cost)
  2. User can see a list of purchase events sorted newest-first, each showing vendor, date, and total spend
  3. User can tap a purchase event row to expand it and see line items with name, quantity, price, and case flag
  4. User can tap a vendor filter chip to narrow the event list to that vendor only
  5. inventory.html loads and renders charts correctly when the device is offline (Chart.js served from SW cache, not CDN)
**Plans:** 2/2 plans complete
Plans:
- [x] 06-01-PLAN.md — Foundation: inventory.html shell, Chart.js local asset, mock data, HQ integration
- [x] 06-02-PLAN.md — History tab rendering, vendor filter, Playwright E2E tests
**UI hint**: yes

### Phase 7: Stock and Reorder Tab
**Goal**: Users can see at-a-glance stock level estimates for every item group and get a prioritized reorder suggestions list — with no manual counting required
**Depends on**: Phase 6
**Requirements**: STCK-01, STCK-02, STCK-03
**Success Criteria** (what must be TRUE):
  1. User can see every item group with a Low, Medium, or High badge derived from purchase recency (not manual entry)
  2. User can see a Reorder Suggestions section listing only Low and Medium items, sorted by urgency
  3. User can tap an item's stock badge and override it with a different level and a reason, and see the override reflected immediately
**Plans:** 2/2 plans complete
Plans:
- [x] 07-01-PLAN.md — Stock estimation algorithm, MOCK_SALES data, tag-grouped Stock tab UI with badges and reorder suggestions
- [x] 07-02-PLAN.md — Manual override flow, Playwright E2E tests, SW cache bump
**UI hint**: yes

### Phase 8: Trends and Cost Intelligence Tabs
**Goal**: Users can see where money is going by category and over time via charts, and can drill into estimated food cost per menu item — with the Trends and Cost tabs structured so they can be replaced by Metabase iframes without touching History or Stock
**Depends on**: Phase 7
**Requirements**: TRND-01, TRND-02, TRND-03, TRND-04, COST-01, COST-02, COST-03, INTG-02
**Success Criteria** (what must be TRUE):
  1. User can see a bar chart of total spending grouped by tag category (Beef, Produce, Supplies, etc.)
  2. User can see a doughnut chart showing each tag's proportion of total spend
  3. User can see a monthly trend line chart showing spend over time, and tap a tag chip to filter it to one category
  4. User can tap a menu item (e.g., Cheesesteak) and see an estimated cost with an ingredient proportion table
  5. User can tap a purchase item (e.g., beef) and see which menu items use it with relative percentages
**Plans:** 2 plans
Plans:
- [ ] 08-01-PLAN.md — Trends tab: Chart.js bar/doughnut/line charts with tag filter chips and sub-tabs
- [ ] 08-02-PLAN.md — Cost tab: menu item cost breakdown, ingredient reverse-lookup, Playwright tests, SW cache bump
**UI hint**: yes

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Template Builder | v1.0 | 3/3 | Complete | 2026-04-13 |
| 2. Fill-Out and Conditional Logic | v1.0 | 2/2 | Complete | 2026-04-13 |
| 3. Photo, Approval, and Integration | v1.0 | 2/2 | Complete | 2026-04-13 |
| 4. Onboarding App | v1.0 | 3/3 | Complete | 2026-04-14 |
| 5. Onboarding Builder | v1.0 | 2/2 | Complete | 2026-04-14 |
| 6. Foundation and History Tab | v1.1 | 2/2 | Complete   | 2026-04-14 |
| 7. Stock and Reorder Tab | v1.1 | 2/2 | Complete   | 2026-04-14 |
| 8. Trends and Cost Intelligence Tabs | v1.1 | 0/2 | Planning | - |
