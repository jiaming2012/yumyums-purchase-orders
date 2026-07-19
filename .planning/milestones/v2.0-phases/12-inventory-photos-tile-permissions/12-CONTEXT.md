# Phase 12: Inventory + Photos + Tile Permissions - Context

**Gathered:** 2026-04-18
**Status:** Ready for planning

<domain>
## Phase Boundary

Persist purchase events and vendor data in Postgres (replacing inventory.html mock data). Build a background receipt ingestion pipeline that pulls from Mercury banking API, parses receipts with Claude Haiku, validates totals, and auto-creates purchase events — routing mismatches to an in-app review queue. Wire checklist photo fields to DO Spaces via presigned URLs. Filter HQ launcher tiles by user permissions.

Requirements: INVT-01, INVT-02, INVT-03, PHOT-01, PHOT-02, TILE-01

</domain>

<decisions>
## Implementation Decisions

### Photo Storage
- **D-01:** DO Spaces for all photo/file storage — S3-compatible, $5/mo for 250GB, already on Digital Ocean.
- **D-02:** Type-prefixed bucket paths: `checklists/{submission_id}/{field_id}.jpg`, `receipts/{event_id}/original.pdf`, `receipts/{event_id}/parsed.json`.
- **D-03:** Presigned PUT URLs from Go backend — frontend uploads directly to Spaces. No file proxying through the server.

### Receipt Ingestion Pipeline
- **D-04:** Background Go worker (cron or Temporal) pulls transactions from Mercury banking API. NOT an in-app upload flow. Modeled after the existing baserow-revenue-api pattern.
- **D-05:** Claude Haiku for receipt parsing — extracts vendor, line items (description, quantity, unit, price), summary (subtotal, tax, total). Replaces Gemini from the baserow project.
- **D-06:** Validation checks item calculated total vs bank transaction amount. Mismatches go to a PendingPurchases review queue.
- **D-07:** In-app review queue in inventory.html — pre-filled form showing parsed receipt data. User corrects errors, confirms to save. Nothing saves without human approval.
- **D-08:** Subtotal/total mismatch warning (ported from baserow-revenue-api's ValidateReceiptData pattern) — warns user if line item sum doesn't match receipt subtotal.
- **D-09:** Auto-creates vendors and purchase items if they don't exist (fuzzy match existing via DerivePurchaseItem pattern from baserow).
- **D-10:** Mercury is first provider. Architecture should support adding Xero and other providers later.

### Inventory Schema + API
- **D-11:** Normalized tables: vendors, item_groups, purchase_items, purchase_events, purchase_line_items, tags (with junction table). Matches Phase 10 patterns.
- **D-12:** Big-bang mock→API swap for inventory.html — replace all MOCK_ arrays with API calls. Delete all mock data.
- **D-13:** History tab wired to real purchase events from Postgres. Stock and Trends tabs compute from purchase data only — sales data integration deferred (no POS system yet).

### Checklist Photo Wiring
- **D-14:** Replace blob URLs with presigned upload immediately after camera capture. Request presigned URL → upload blob to Spaces → store Spaces URL in field response. Existing thumbnail/preview UI unchanged — just swap src from blob to https URL.

### Inventory Seed Data
- **D-15:** YAML fixtures following the baserow-revenue-api purchase_item_groups.yaml pattern. Go seed function loads on startup. Vendors, item groups with tags, and purchase items.

### Tile Permissions
- **D-16:** index.html fetches GET /api/v1/me/apps on load. Tiles not in the response are hidden completely (removed from grid). User only sees what they have permission to access.
- **D-17:** Cache last-known /me/apps response in localStorage. Show cached tiles immediately on load, refresh in background. Works offline with last-known permissions.

### Claude's Discretion
- Inventory table schema details (exact columns, indexes, constraints)
- API endpoint naming (RPC-style matching Phase 10/11 patterns)
- Mercury API integration details (auth, pagination, date range handling)
- Receipt parsing prompt design for Claude Haiku
- DO Spaces bucket name, region, CORS policy
- Background worker scheduling (cron interval or Temporal workflow)
- Review queue UI layout in inventory.html
- E2E test approach for the pipeline (mock Mercury API responses)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Receipt Pipeline Reference
- `/Users/jamal/projects/yumyums/baserow-revenue-api/src/main.go` — Full receipt ingestion pipeline pattern: Mercury → AI parse → validate → auto-create. Port this flow to the HQ Go backend.
- `/Users/jamal/projects/yumyums/baserow-revenue-api/src/services/` — FetchReceipts, ParseReceipt, ValidateReceiptData, DerivePurchaseItem
- `/Users/jamal/projects/yumyums/baserow-revenue-api/src/models/` — BaserowPendingPurchase, PurchaseEvent, PurchaseTable data models
- `/Users/jamal/projects/yumyums/baserow-revenue-api/fixtures/purchase_item_groups.yaml` — YAML fixture pattern for seed data

### Backend Foundation
- `backend/internal/auth/service.go` — Auth patterns, session management
- `backend/internal/workflow/` — RPC handler patterns, existing photo field stubs
- `backend/internal/db/migrations/` — Existing migrations 0001-0023 for schema patterns
- `backend/cmd/server/main.go` — Route registration, seed patterns

### API Design
- `docs/user-management-api.md` — GET /api/v1/me/apps contract for tile permissions

### Existing Frontend
- `inventory.html` — 933-line implementation with MOCK_VENDORS, MOCK_PURCHASE_EVENTS, MOCK_PURCHASES, MOCK_ITEM_GROUPS, MOCK_PURCHASE_ITEMS, MOCK_TAGS, MOCK_MENU_ITEMS, MOCK_SALES
- `workflows.html` — Existing camera capture UI (photo-modal, photo-thumb, photo-capture-btn) — needs presigned upload wiring
- `index.html` — Launcher grid with static tiles — needs /me/apps filtering

### Prior Phase Context
- `.planning/phases/10-workflows-api/10-CONTEXT.md` — Photo fields deferred (D-26), big-bang swap pattern (D-17)
- `.planning/phases/11-onboarding-users-admin/11-CONTEXT.md` — Users admin API, role model

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `backend/internal/workflow/` — RPC handler pattern, auto-save, presigned URL patterns
- `backend/internal/auth/` — Session management, middleware, UserFromContext
- `backend/internal/db/` — Pool setup, goose migrations, seed patterns
- `workflows.html` camera UI — Full photo capture modal with preview, retake, confirm flow
- `index.html` — checkAuth() already fetches /api/v1/me, extend to fetch /me/apps

### Established Patterns
- Chi router with middleware groups
- Goose embedded SQL migrations
- RPC-style write endpoints
- State-first rendering in frontend
- Big-bang mock→API swap

### Integration Points
- `backend/cmd/server/main.go` — Add inventory routes, photo presign endpoint, background worker init
- `inventory.html` — Replace all MOCK_ arrays with API calls
- `workflows.html` — Swap blob URLs to presigned upload flow in photo capture
- `index.html` — Add /me/apps fetch + tile filtering + localStorage cache

</code_context>

<specifics>
## Specific Ideas

- Receipt pipeline is a BACKGROUND PROCESS, not an in-app upload — receipts come from Mercury banking API, not user cameras
- The baserow-revenue-api pattern (main.go) is the implementation blueprint — port the flow, swap Gemini for Claude Haiku, swap Baserow for Postgres
- Mercury is the first banking provider — architecture should support adding Xero later
- ValidateReceiptData pattern from baserow validates subtotal vs bank transaction amount — mismatches route to PendingPurchases for human review
- DerivePurchaseItem pattern does fuzzy matching of vendor/item names against existing records
- Stock/Trends tabs compute from purchase data only — no sales data integration yet (no POS system)

</specifics>

<deferred>
## Deferred Ideas

- Xero integration as additional receipt provider — future phase
- POS/sales data integration for true food cost calculations — future milestone (COST-01, COST-02, COST-03)
- Real-time stock counting / barcode scanning — out of scope
- Push notifications for pending review items — future phase

</deferred>

---

*Phase: 12-inventory-photos-tile-permissions*
*Context gathered: 2026-04-18*
