# Roadmap: Yumyums HQ

## Milestones

- ✅ **v1.0 Operations Console MVP** — Phases 1-5 (shipped 2026-04-14) — [Archive](milestones/v1.0-ROADMAP.md)
- ✅ **v1.1 Inventory App** — Phases 6-8 (shipped 2026-04-14) — [Archive](milestones/v1.1-ROADMAP.md)
- 🚧 **v2.0 Backend** — Phases 9-12 (in progress)

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

### 🚧 v2.0 Backend (In Progress)

**Milestone Goal:** Replace all mock data with a real Go + Postgres backend — auth, workflows persistence, onboarding persistence, inventory, offline sync, and photo storage — so the crew can use the app for real operations on their phones.

- [ ] **Phase 9: Foundation + Auth** — Go server shell, Postgres, Tailscale HTTPS dev access, SW partition, and working login/logout
- [ ] **Phase 10: Workflows API** — Checklist templates, submissions, approval flow, correction loop, and offline sync wired to workflows.html
- [ ] **Phase 11: Onboarding + Users Admin** — Onboarding persistence, user CRUD, role management, and app permissions wired to their respective HTML pages
- [ ] **Phase 12: Inventory + Photos** — Purchase events, vendor data, receipt ingestion, and presigned photo upload wired to inventory.html and workflows.html

## Phase Details

### Phase 9: Foundation + Auth
**Goal**: The Go backend runs, serves the PWA from a single origin over Tailscale HTTPS, and crew can log in with real credentials
**Depends on**: Nothing (first v2.0 phase)
**Requirements**: INFRA-01, INFRA-02, INFRA-03, INFRA-04, AUTH-01, AUTH-02, AUTH-03, AUTH-04
**Success Criteria** (what must be TRUE):
  1. Opening the app URL on a physical phone over Tailscale loads the HQ shell with no certificate errors
  2. Service worker DevTools Cache Storage shows zero `/api/` URLs cached — only static assets
  3. Crew member can enter email + password on login.html and land on the HQ launcher (no mock alert)
  4. Logging out invalidates the session — the browser cannot re-access protected pages without logging in again
  5. An unauthenticated request to any `/api/v1/` endpoint (except `/health`) returns 401
**Plans**: TBD
**UI hint**: yes

### Phase 10: Workflows API
**Goal**: Checklist templates are persisted in Postgres, crew can fill out and submit real checklists, managers can approve or reject, and the app works offline with a pending-sync indicator
**Depends on**: Phase 9
**Requirements**: WKFL-01, WKFL-02, WKFL-03, WKFL-04, SYNC-01, SYNC-02, SYNC-03
**Success Criteria** (what must be TRUE):
  1. A template created in the Builder tab survives a page reload and appears in My Checklists the next day it is scheduled
  2. Crew member submits a filled checklist and it appears in the manager's Approvals tab with field responses and user attribution
  3. Manager can approve or reject individual items; rejection causes the checklist to reappear for the crew member with flagged items
  4. Filling out a checklist while offline queues the submission; going back online syncs it and the pending indicator clears
**Plans**: TBD

### Phase 11: Onboarding + Users Admin
**Goal**: New hire training progress persists across sessions, manager sign-offs are recorded, and the admin can invite crew members and manage permissions through a real API
**Depends on**: Phase 9
**Requirements**: ONBD-01, ONBD-02, ONBD-03, ONBD-04, USER-01, USER-02, USER-03
**Success Criteria** (what must be TRUE):
  1. A hire's checked training items and watched video parts are still checked after closing and reopening the app
  2. Manager signs off on a training section; the sign-off appears in the hire's journal with the manager's name and timestamp
  3. Admin sends an email invite to a new crew member; the invite recipient can set a password and log in
  4. Admin changes a user's role or app permissions in users.html and the change takes effect on next login without a code deploy
**Plans**: TBD
**UI hint**: yes

### Phase 12: Inventory + Photos
**Goal**: Purchase history and vendor data come from Postgres, receipt photos can be uploaded and parsed into purchase events, and checklist photos are stored in object storage
**Depends on**: Phase 9
**Requirements**: INVT-01, INVT-02, INVT-03, PHOT-01, PHOT-02
**Success Criteria** (what must be TRUE):
  1. A purchase event entered via the app appears in the History tab after a page reload with correct vendor, line items, and totals
  2. Uploading a receipt photo triggers OCR parsing and pre-fills a purchase event form for human review before saving
  3. A photo taken during checklist completion (fail note or photo field) is retrievable by the manager in the Approvals tab after the submission is made
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 9 → 10 → 11 → 12

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Template Builder | v1.0 | 3/3 | Complete | 2026-04-13 |
| 2. Fill-Out and Conditional Logic | v1.0 | 2/2 | Complete | 2026-04-13 |
| 3. Photo, Approval, and Integration | v1.0 | 2/2 | Complete | 2026-04-13 |
| 4. Onboarding App | v1.0 | 3/3 | Complete | 2026-04-14 |
| 5. Onboarding Builder | v1.0 | 2/2 | Complete | 2026-04-14 |
| 6. Foundation and History Tab | v1.1 | 2/2 | Complete | 2026-04-14 |
| 7. Stock and Reorder Tab | v1.1 | 2/2 | Complete | 2026-04-14 |
| 8. Trends and Cost Intelligence Tabs | v1.1 | 2/2 | Complete | 2026-04-14 |
| 9. Foundation + Auth | v2.0 | 0/? | Not started | - |
| 10. Workflows API | v2.0 | 0/? | Not started | - |
| 11. Onboarding + Users Admin | v2.0 | 0/? | Not started | - |
| 12. Inventory + Photos | v2.0 | 0/? | Not started | - |
