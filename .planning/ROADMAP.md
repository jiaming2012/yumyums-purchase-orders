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

### v2.0 Backend (In Progress)

**Milestone Goal:** Replace all mock data with a real Go + Postgres backend — auth, workflows persistence, onboarding persistence, inventory, offline sync, and photo storage — so the crew can use the app for real operations on their phones.

- [x] **Phase 9: Foundation + Auth** — Go server shell, Postgres, Tailscale HTTPS dev access, SW partition, and working login/logout (completed 2026-04-15)
- [x] **Phase 10: Workflows API** — Checklist templates, submissions, approval flow, correction loop, and offline sync wired to workflows.html (completed 2026-04-15)
- [x] **Phase 10.1: Cross-Device State Sync** — Op-log, WebSocket hub, real-time fan-out, Lamport clocks, conflict resolution, sync UX (completed 2026-04-17)
- [x] **Phase 10.2: Reactive Sync Framework** — Shared Store with collection-level subscriptions, single write channel (POST /ops), shared JS modules for all tools (completed 2026-04-17)
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
**Plans**: 4 plans

Plans:
- [x] 09-01-PLAN.md — Go server shell with chi router + SW fetch partition
- [x] 09-02-PLAN.md — Postgres with goose migrations + superadmin config
- [x] 09-03-PLAN.md — Auth service, middleware, login/logout/me handlers
- [x] 09-04-PLAN.md — Wire login.html to real API + Tailscale phone verify

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
**Plans**: 5 plans

Plans:
- [x] 10-01-PLAN.md — Migrations + Go model structs + template seed
- [x] 10-02-PLAN.md — Repository + handlers + route wiring
- [x] 10-03-PLAN.md — Frontend big-bang mock swap + skeleton/error UI
- [x] 10-04-PLAN.md — Offline sync (IndexedDB + banner + drain)
- [x] 10-05-PLAN.md — Playwright E2E rewrite + phone verification

### Phase 10.1: Cross-Device State Sync (INSERTED)

**Goal:** Real-time cross-user/cross-device state sync via op-log + WebSocket fan-out, with LWW conflict resolution and visual sync indicators
**Depends on:** Phase 10
**Requirements**: SYNC-10.1-01, SYNC-10.1-02, SYNC-10.1-03, SYNC-10.1-04, SYNC-10.1-05, SYNC-10.1-06, SYNC-10.1-07, SYNC-10.1-08, SYNC-10.1-09, SYNC-10.1-10, SYNC-10.1-11, SYNC-10.1-12, SYNC-10.1-13, SYNC-10.1-14
**Success Criteria** (what must be TRUE):
  1. A field change on one device appears on another device in real time with a blue flash highlight
  2. A checklist submission on one device refreshes the approvals tab on the manager's device
  3. Closing and reopening the app catches up on missed ops without data loss
  4. Conflicting field edits resolve deterministically (higher lamport_ts wins) with visual feedback
**Plans**: 5 plans

Plans:
- [x] 10.1-01-PLAN.md — DB migrations (ops table + lamport_ts) + sync/ops.go package
- [x] 10.1-02-PLAN.md — WebSocket hub + Postgres LISTEN/NOTIFY listener + route wiring
- [x] 10.1-03-PLAN.md — Wire EmitOp into all 7 workflow handlers
- [x] 10.1-04-PLAN.md — Frontend LamportClock + WebSocket client + applyOp + 409 handling
- [x] 10.1-05-PLAN.md — Sync UX (blue flash + grouped toast) + device verification

**UI hint**: yes

### Phase 10.2: Reactive Sync Framework (INSERTED)

**Goal:** Extract shared Store with collection-level subscriptions + single write channel (POST /ops). All mutations flow through submitOp → server → entity tables + ops + notify → all clients. Optimistic updates preserved. Store, sync client, and applyOp become shared JS modules reusable by onboarding/inventory/users.
**Depends on:** Phase 10.1
**Requirements**: RSYNC-01, RSYNC-02, RSYNC-03, RSYNC-04, RSYNC-05, RSYNC-06, RSYNC-07
**Success Criteria** (what must be TRUE):
  1. sync.js is a shared module loaded by workflows.html providing Store, LamportClock, WS client, submitOp
  2. All workflow state (checklists, submissions, drafts, responses) lives in the reactive Store with collection-level subscribers
  3. All mutations flow through submitOp → POST /ops → server business logic + InsertOpAndNotify → WebSocket fan-out
  4. Self-echo suppression works via real device_id (no _recentSaves timing hack)
  5. All 166 existing tests pass at each migration step boundary
**Plans**: 3 plans

Plans:
- [x] 10.2-01-PLAN.md — Create sync.js (Store + LamportClock + WS + IndexedDB + api + submitOp) + Workbox precache
- [x] 10.2-02-PLAN.md — Wire workflows.html to sync.js (Store collections + subscribers + debouncedSaveField)
- [x] 10.2-03-PLAN.md — Backend POST /ops endpoint + switch submitOp to POST /ops + eliminate _recentSaves

### Phase 11: Onboarding + Users Admin
**Goal**: New hire training progress persists across sessions, manager sign-offs are recorded, and the admin can invite crew members and manage permissions through a real API
**Depends on**: Phase 9
**Requirements**: ONBD-01, ONBD-02, ONBD-03, ONBD-04, USER-01, USER-02, USER-03
**Success Criteria** (what must be TRUE):
  1. A hire's checked training items and watched video parts are still checked after closing and reopening the app
  2. Manager signs off on a training section; the sign-off appears in the hire's journal with the manager's name and timestamp
  3. Admin sends an email invite to a new crew member; the invite recipient can set a password and log in
  4. Admin changes a user's role or app permissions in users.html and the change takes effect on next login without a code deploy
**Plans**: 6 plans
**UI hint**: yes

Plans:
- [x] 11-01-PLAN.md — DB migrations (users naming + ob_* tables) + auth service update
- [x] 11-02-PLAN.md — Users admin backend (handlers + route wiring)
- [x] 11-03-PLAN.md — users.html big-bang API swap + login.html accept-invite mode
- [x] 11-04-PLAN.md — Onboarding backend (handlers + seed + route wiring)
- [x] 11-05-PLAN.md — onboarding.html big-bang API swap + SW rebuild
- [ ] 11-06-PLAN.md — E2E test rewrite (onboarding + users)

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
Phases execute in numeric order: 9 → 10 → 10.1 → 10.2 → 11 → 12

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
| 9. Foundation + Auth | v2.0 | 4/4 | Complete   | 2026-04-15 |
| 10. Workflows API | v2.0 | 5/5 | Complete   | 2026-04-15 |
| 10.1 Cross-Device State Sync | v2.0 | 5/5 | Complete   | 2026-04-17 |
| 10.2 Reactive Sync Framework | v2.0 | 3/3 | Complete    | 2026-04-17 |
| 11. Onboarding + Users Admin | v2.0 | 5/6 | In Progress|  |
| 12. Inventory + Photos | v2.0 | 0/? | Not started | - |
