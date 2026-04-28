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

### Phase 18: Tab persistence on refresh

**Goal:** Persist active tab across page refresh for all apps using URL hash. When a tab is tapped, update `location.hash`. On page load, read the hash and activate the matching tab.
**Requirements:** TBD
**Plans:** 0 plans

### Phase 19: Require store location before adding item to PO

**Goal:** Items without a store_location should appear in the catalog/item picker under an "Unassigned" section but be blocked from being added to a purchase order until a store location is set. This prevents shopping list items from having no location context.
**Depends on:** Phase 16 (Shopping List)
**Requirements:** TBD
**Plans:** 0 plans

## Backlog

(empty)
