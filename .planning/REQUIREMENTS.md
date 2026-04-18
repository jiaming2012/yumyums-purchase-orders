# Requirements: Yumyums HQ — Backend

**Defined:** 2026-04-15
**Core Value:** Replace all mock data with a real Go + Postgres backend — auth, persistence, offline sync, and receipt ingestion — so the crew can use the app for real operations.

## v2.0 Requirements

### Foundation & Infrastructure

- [ ] **INFRA-01**: Go binary serves PWA static files via embed.FS and API via chi router from same origin
- [x] **INFRA-02**: Postgres 16 database with goose migrations for schema management
- [ ] **INFRA-03**: Tailscale Serve provides HTTPS dev access for mobile device testing
- [ ] **INFRA-04**: Service worker fetch handler partitioned — network-first for `/api/*`, cache-first for static files

### Auth & Sessions

- [ ] **AUTH-01**: User can log in with email + password via POST `/api/v1/auth/login` and receive httpOnly session cookie
- [ ] **AUTH-02**: User can log out via POST `/api/v1/auth/logout` which invalidates the session
- [ ] **AUTH-03**: Protected API endpoints reject unauthenticated requests with 401
- [ ] **AUTH-04**: login.html wired to real auth API (replacing mock `alert()`)

### Workflows Persistence

- [x] **WKFL-01**: Templates, sections, and fields persisted to Postgres (replacing MOCK_TEMPLATES)
- [x] **WKFL-02**: Checklist submissions saved with field responses, user attribution, and timestamps
- [x] **WKFL-03**: Approval flow persisted — pending, approved, rejected states with manager notes
- [x] **WKFL-04**: workflows.html fetches data from API instead of hardcoded JS arrays

### Onboarding Persistence

- [x] **ONBD-01**: Onboarding templates, sections, items, FAQ Q&A persisted to Postgres
- [x] **ONBD-02**: Training progress (checked items, video parts watched) saved per hire
- [x] **ONBD-03**: Section sign-off journal entries persisted with manager, reason, timestamp
- [x] **ONBD-04**: onboarding.html fetches data from API instead of hardcoded JS arrays

### Inventory Persistence

- [x] **INVT-01**: Vendors, purchase events, and line items persisted to Postgres (replacing mock data)
- [ ] **INVT-02**: inventory.html fetches purchase data from API for History, Stock, Trends, and Cost tabs
- [ ] **INVT-03**: Receipt ingestion pipeline — upload receipt image, OCR, map to purchase items, human review

### Offline Sync

- [x] **SYNC-01**: Checklist completions queued in IndexedDB when offline
- [x] **SYNC-02**: Queue replays on `online` event with idempotency keys preventing duplicates
- [x] **SYNC-03**: User sees visual indicator of pending offline submissions

### Users Admin

- [x] **USER-01**: Admin can invite new users via API (email invite flow)
- [x] **USER-02**: Admin can manage user roles and app permissions via API
- [x] **USER-03**: users.html wired to real admin API (replacing mock data)

### Photos

- [x] **PHOT-01**: Photo upload via presigned URLs (evaluate Zoho Stratus vs DO Spaces for cost)
- [x] **PHOT-02**: Photos stored and retrievable for checklist evidence and corrective action documentation

### Tile Permissions

- [x] **TILE-01**: index.html launcher grid filtered by GET /api/v1/me/apps — users only see tiles for apps they have permission to access

## Future Requirements

Deferred to future milestones.

### Food Cost Intelligence
- **COST-01**: Server-side ingredient ratio derivation from purchase + sales data
- **COST-02**: AI-assisted cost estimation pipeline
- **COST-03**: Revenue data integration for true food cost percentage

### Advanced
- **ADVN-01**: Push notifications when checklists are due
- **ADVN-02**: HTMX frontend migration (replace vanilla fetch calls)
- **ADVN-03**: Metabase iframe embedding for Trends/Cost tabs

## Out of Scope

| Feature | Reason |
|---------|--------|
| Food cost calculations | Deferred to future milestone — needs more data |
| Multi-location support | Single food truck operation |
| Real-time collaboration | 1 person per checklist, last-write-wins acceptable |
| HTMX migration | Future milestone — vanilla JS works for now |
| Barcode scanning | Hardware dependency, overkill for food truck scale |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INFRA-01 | Phase 9 | Pending |
| INFRA-02 | Phase 9 | Complete |
| INFRA-03 | Phase 9 | Pending |
| INFRA-04 | Phase 9 | Pending |
| AUTH-01 | Phase 9 | Pending |
| AUTH-02 | Phase 9 | Pending |
| AUTH-03 | Phase 9 | Pending |
| AUTH-04 | Phase 9 | Pending |
| WKFL-01 | Phase 10 | Complete |
| WKFL-02 | Phase 10 | Complete |
| WKFL-03 | Phase 10 | Complete |
| WKFL-04 | Phase 10 | Complete |
| SYNC-01 | Phase 10 | Complete |
| SYNC-02 | Phase 10 | Complete |
| SYNC-03 | Phase 10 | Complete |
| ONBD-01 | Phase 11 | Complete |
| ONBD-02 | Phase 11 | Complete |
| ONBD-03 | Phase 11 | Complete |
| ONBD-04 | Phase 11 | Complete |
| USER-01 | Phase 11 | Complete |
| USER-02 | Phase 11 | Complete |
| USER-03 | Phase 11 | Complete |
| INVT-01 | Phase 12 | Complete |
| INVT-02 | Phase 12 | Pending |
| INVT-03 | Phase 12 | Pending |
| PHOT-01 | Phase 12 | Complete |
| PHOT-02 | Phase 12 | Complete |
| TILE-01 | Phase 12 | Complete |

**Coverage:**
- v2.0 requirements: 28 total
- Mapped to phases: 27
- Unmapped: 0

---
*Requirements defined: 2026-04-15*
