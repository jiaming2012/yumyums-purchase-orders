# Roadmap: Yumyums HQ

## Milestones

- ✅ **v1.0 Operations Console MVP** — Phases 1-5 (shipped 2026-04-14) — [Archive](milestones/v1.0-ROADMAP.md)
- ✅ **v1.1 Inventory App** — Phases 6-8 (shipped 2026-04-14) — [Archive](milestones/v1.1-ROADMAP.md)
- ✅ **v2.0 Backend** — Phases 9-13 (shipped 2026-04-19) — [Archive](milestones/v2.0-ROADMAP.md)
- ✅ **v2.1 Onboarding Video Upgrade** — Phase 1 (shipped 2026-04-20) — [Archive](milestones/v2.1-ROADMAP.md)
- 🚧 **v3.0 Purchase Orders & Shopping Lists** — Phases 14-17 (in progress)

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

_Full phase details archived to [milestones/v2.0-ROADMAP.md](milestones/v2.0-ROADMAP.md)_

<details>
<summary>✅ v2.1 Onboarding Video Upgrade (Phase 1) — SHIPPED 2026-04-20</summary>

- [x] Phase 1: Onboarding Video Upgrade (3/3 plans) — completed 2026-04-20

</details>

### 🚧 v3.0 Purchase Orders & Shopping Lists (In Progress)

**Milestone Goal:** Connect the Purchase Orders tool to real inventory data with a full ordering-to-shopping workflow, cutoff enforcement, and multi-channel alerts via Zoho Cliq/email.

- [x] **Phase 14: PO Backend + Order Form** - Schema migrations, purchasing API, and purchasing.html Tab 1 with live reorder suggestions, item search, added-by attribution, and duplicate detection (completed 2026-04-22)
- [x] **Phase 15: Notion Catalog Seed** - Convert Notion CSV to YAML seed file with images re-hosted to DO Spaces; seeds on first server run (completed 2026-04-22)
- [x] **Phase 16: Cutoff, Approval, and Shopping List** - Cutoff scheduler, admin approval flow, shopping list with photo capture prompts and location enrichment (completed 2026-04-22)
- [ ] **Phase 17: Alerts, Notifications, and Repurchase Badges** - Async alert queue, Zoho Cliq/email delivery, user notification preferences, and inventory badge on shopping completion

## Phase Details

### Phase 14: PO Backend + Order Form
**Goal**: Users can create and edit a real purchase order from live inventory data, with reorder suggestions pre-populated and item search available
**Depends on**: Phase 13 (v2.0 backend)
**Requirements**: PO-01, PO-02, PO-03, PO-04, PO-05, PO-06, PO-07, PO-08
**Success Criteria** (what must be TRUE):
  1. User sees a pre-populated list of items flagged for reorder (current stock below low threshold) when opening the PO form
  2. User can tap a reorder suggestion to add it to the order with the suggested quantity
  3. User can search the item catalog via a fullscreen picker modal and add any item not already in the list
  4. Each line item in the PO shows the item's photo, store location note, suggested quantity, and who added it
  5. User can adjust quantity on each line item and changes persist to the API
  6. When adding an item, user can see if it's already on the PO and its current quantity
**Plans:** 2/2 plans complete
Plans:
- [x] 14-01-PLAN.md -- Backend: migrations, purchasing Go package, route registration
- [x] 14-02-PLAN.md -- Frontend: purchasing.html rewrite with Order tab, item picker, suggestions
**UI hint**: yes

### Phase 15: Notion Catalog Seed
**Goal**: Convert Notion CSV export into a YAML seed file with images re-hosted to DO Spaces; seeds on first run like other YAML fixtures
**Depends on**: Phase 14
**Requirements**: IMP-01, IMP-02
**Success Criteria** (what must be TRUE):
  1. Notion CSV is converted to a YAML seed file following existing fixture patterns (e.g., superadmins.yaml)
  2. Items seed idempotently on first server run (re-runs do not duplicate)
  3. Item photos from the Notion export are downloaded and re-hosted to DO Spaces; the raw Notion S3 URL is never stored in the YAML
  4. Photos load correctly 2+ hours after seed (confirming no reliance on expiring Notion URLs)
**Plans:** 2/2 plans complete
Plans:
- [ ] 15-01-PLAN.md -- Migration, group fixtures, conversion script, generated YAML
- [x] 15-02-PLAN.md -- Extend SeedInventoryFixtures to load purchase_items.yaml on startup

### Phase 16: Cutoff, Approval, and Shopping List
**Goal**: Admin can configure a weekly cutoff that auto-locks the PO, approve the locked PO to generate a shopping checklist, and crew can execute the shopping run with check-off, photo capture, and inline location notes
**Depends on**: Phase 14
**Requirements**: CUT-01, CUT-02, CUT-03, CUT-04, SHOP-01, SHOP-02, SHOP-03, SHOP-04, SHOP-05, SHOP-06, SHOP-07, SHOP-08
**Success Criteria** (what must be TRUE):
  1. Admin can set a recurring weekly cutoff day and time; PO automatically locks when the deadline passes
  2. After cutoff, non-admin users cannot edit the PO; admin sees an override option
  3. Admin can approve a locked PO, which instantly creates an immutable shopping checklist snapshot
  4. Shopper sees each item with its photo and store location; tapping the location icon reveals the full note
  5. Shopper can edit store location notes inline and check off items while shopping
  6. When checking off an item that has no photo or location, shopper is prompted to add them (can skip but must confirm skip each time)
  7. Shopper can upload a photo for an item that doesn't have one
  8. Tapping "Complete" on the shopping list sends a report of any unchecked items via the configured alert channel
**Plans:** 5/5 plans complete
Plans:
- [x] 16-01-PLAN.md -- Migrations (cutoff_config, status extend, vendor FK, shopping tables) + Go types
- [x] 16-02-PLAN.md -- Backend state machine (lock/unlock/approve), cutoff scheduler, simulate-cutoff
- [x] 16-03-PLAN.md -- Backend shopping list CRUD, check-off, vendor completion, photo/location update
- [x] 16-04-PLAN.md -- Frontend PO tab (locked view, cutoff pill config, admin edit, approve button)
- [x] 16-05-PLAN.md -- Frontend Shopping tab (check-off, toast/badge, photo, location) + History tab
**UI hint**: yes

### Phase 17: Alerts, Notifications, and Repurchase Badges
**Goal**: The system delivers cutoff reminders and shopping completion alerts via Zoho Cliq or email based on user preference, and inventory Stock tab shows a "Repurchased" badge after shopping is confirmed
**Depends on**: Phase 16
**Requirements**: ALRT-01, ALRT-02, ALRT-03, ALRT-04, ALRT-05, ALRT-06, REP-01, REP-02
**Success Criteria** (what must be TRUE):
  1. User receives a reminder alert via their configured channel (Zoho Cliq or email) before the cutoff deadline
  2. Shopping completion triggers an alert listing any missing/unchecked items, delivered to configured channels
  3. Users can configure their notification preference (Zoho Cliq or email) in the Users tab; at least one channel is required
  4. Inventory Stock tab items show a "Repurchased +[Qty]" badge after the shopping list is marked complete
  5. Badge resets on the admin-configured schedule (weekly reset date is timezone-aware)
**Plans:** 2 plans
Plans:
- [x] 14-01-PLAN.md -- Backend: migrations, purchasing Go package, route registration
- [ ] 14-02-PLAN.md -- Frontend: purchasing.html rewrite with Order tab, item picker, suggestions
**UI hint**: yes

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 14. PO Backend + Order Form | v3.0 | 2/2 | Complete    | 2026-04-22 |
| 15. Notion Catalog Import | v3.0 | 1/2 | Complete    | 2026-04-22 |
| 16. Cutoff, Approval, and Shopping List | v3.0 | 5/5 | Complete   | 2026-04-22 |
| 17. Alerts, Notifications, and Repurchase Badges | v3.0 | 0/? | Not started | - |
