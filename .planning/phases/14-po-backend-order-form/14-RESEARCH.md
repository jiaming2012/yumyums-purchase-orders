# Phase 14: PO Backend + Order Form - Research

**Researched:** 2026-04-22
**Domain:** Go + Postgres backend (new `purchasing` package), vanilla JS frontend rewrite of `purchasing.html`
**Confidence:** HIGH — derived from direct codebase read, existing research docs, and UI-SPEC

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Photo card layout per item — thumbnail, name, unit, store location, stepper (+/-), and "Added by" attribution
- **D-02:** Items grouped by category with sticky headers (matches existing Stock tab and mock PO patterns)
- **D-03:** Each card is ~80px tall with photo on the left, info in the middle, stepper on the right
- **D-04:** Suggestion banner at top of Order tab showing count ("3 items need restock [+]")
- **D-05:** Tapping [+] opens a checklist of items below stock threshold; user selects which to add with suggested quantities pre-filled
- **D-06:** Suggestions come from existing inventory Stock tab threshold logic (items where qty < group low_threshold)
- **D-07:** Item picker shows "On PO: [qty]" badge for items already on the current order
- **D-08:** Tapping an already-added item shows popup: "[Item] already on PO — Qty: [N] · Added by: [initials]" with "Go to item" link and "Dismiss"
- **D-09:** Items not on the PO show [+ Add] button in the picker
- **D-10:** 4 tabs in this order: Order / Shopping / PO / History
- **D-11:** Order tab — Always open for contributions; after weekly cutoff the current list moves to PO tab and a fresh Order list starts
- **D-12:** PO tab — Frozen cutoff list (stub state in Phase 14)
- **D-13:** Shopping tab — Approved shopping checklist (stub state in Phase 14)
- **D-14:** History tab — Past shopping lists (stub state in Phase 14)
- **D-15:** Shopping list has one submit button per vendor section
- **D-16:** History entries show who completed each shopping list (per-vendor completion tracked)

### Claude's Discretion

- Empty state design for Order tab (first time, no items added yet)
- Exact stepper button styling (reuse existing mock pattern or refresh)
- Search input placement and debounce timing in item picker
- Skeleton/loading states while fetching inventory data

### Deferred Ideas (OUT OF SCOPE)

- Shopping list check-off and completion flow — Phase 16
- Cutoff enforcement and admin approval — Phase 16
- Alert notifications (cutoff reminder, missing items) — Phase 17
- Repurchase badges on inventory — Phase 17
- Price tracking and cost comparison across vendors — future milestone
</user_constraints>

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PO-01 | User can view reorder suggestions from inventory Stock tab on the PO form | Suggestions query: `GET /api/v1/purchasing/orders/:id/suggestions` — stock WHERE qty < low_threshold (existing group thresholds). Rendered as inline checklist below banner. |
| PO-02 | User can tap a reorder suggestion to add it to the purchase order | "Add Selected" sends selected items to `PUT /api/v1/purchasing/orders/:id/items`; suggested_qty = low_threshold - current_stock |
| PO-03 | User can search and add items from the inventory catalog (Setup) | Fullscreen picker modal adapted from `inventory.html` `.item-modal` pattern; calls existing `GET /api/v1/inventory/items` |
| PO-04 | Each PO line item shows the item's photo, store location note, and suggested quantity | Requires migration 0039 to add `photo_url` + `store_location` to `purchase_items`; returned in `GET /api/v1/purchasing/orders/:id` response |
| PO-05 | User can adjust quantity on each line item before submission | Stepper +/- with optimistic UI; persists via `PUT /api/v1/purchasing/orders/:id/items` with 600ms debounce |
| PO-06 | PO form is backed by real API data (replaces current purchasing.html mock) | Full rewrite of purchasing.html; `POST /api/v1/purchasing/orders` (get-or-create current week) → renders live data |
| PO-07 | Each PO line item shows who added it to the order | `po_line_items.added_by` UUID FK → users table; API returns user initials; rendered as "Added by: JM" in card middle section |
| PO-08 | When adding an item, user can see if it's already on the PO and its current quantity | Client-side check against current PO state; picker shows "On PO: [qty] [unit]" badge for existing items; tap shows duplicate popup |
</phase_requirements>

---

## Summary

Phase 14 replaces the 90-line hardcoded `purchasing.html` mock with a live API-backed tool. The backend side adds a new `internal/purchasing` Go package (mirroring `internal/inventory`), two Goose migrations (purchase_orders + po_line_items tables, plus an ALTER on purchase_items for photo_url/store_location), and three new endpoints for Phase 14 scope only. The frontend side rewrites `purchasing.html` into a 4-tab shell where only the Order tab is fully interactive — the other three tabs show stub content.

The primary complexity in this phase is the backend: the get-or-create PO logic (always returns the current week's draft), the suggestions query (JOIN against existing stock/threshold data), and the upsert-based line item save pattern. The frontend complexity is the item picker modal adaptation and duplicate detection. Both are well-understood patterns with working code already in the codebase to reference directly.

**Primary recommendation:** Build backend first (migrations → handler → routes), then frontend. The frontend depends on the API shape; defining Go types first eliminates guessing in JS.

---

## Standard Stack

### Core

| Component | Version / Pattern | Purpose | Why Standard |
|-----------|------------------|---------|--------------|
| Go `internal/purchasing` package | Go 1.21+ (matches existing) | HTTP handlers, types, service logic | Mirrors `internal/inventory` structure exactly |
| pgx v5 (`pgxpool.Pool`) | v5 (existing) | Postgres queries | Already used everywhere; same connection pool |
| `chi` router | v5 (existing) | Route mounting | Already used in main.go |
| Goose migrations | existing pattern | Schema changes | All migrations in `backend/internal/db/migrations/` |
| Vanilla JS + CSS custom properties | existing | Frontend | Project convention — no framework |

### New API Endpoints (Phase 14 scope only)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/purchasing/orders` | POST | Get-or-create current week's draft PO |
| `/api/v1/purchasing/orders/:id` | GET | Get PO with line items + user initials |
| `/api/v1/purchasing/orders/:id/items` | PUT | Upsert line items (full-state send, qty=0 removes) |
| `/api/v1/purchasing/orders/:id/suggestions` | GET | Reorder suggestions from stock thresholds |

**Not in Phase 14:** lock, approve, shopping list, cutoff config, notifications, badges. Those are Phases 16-17.

### New Migrations (Phase 14)

| File | Content |
|------|---------|
| `0034_purchase_orders.sql` | `purchase_orders` + `po_line_items` tables |
| `0035_purchase_items_photo_store.sql` | `ALTER TABLE purchase_items ADD COLUMN photo_url TEXT, ADD COLUMN store_location TEXT` |

The next migration number is `0034` — confirmed by reading `backend/internal/db/migrations/` (last existing is `0033_stock_count_overrides.sql`).

**Version verification:** No npm packages introduced in this phase.

---

## Architecture Patterns

### Recommended Project Structure

```
backend/internal/purchasing/
├── handler.go      — HTTP handlers for Phase 14 routes
├── types.go        — PurchaseOrder, POLineItem, OrderSuggestion Go types
└── service.go      — get-or-create PO, suggestions query, upsert line items
```

The scheduler, notifier, and remaining endpoints come in Phase 16-17. Keeping Phase 14 to 3 files prevents over-engineering an incomplete feature.

### Pattern 1: Get-or-Create Current Week's PO

**What:** `POST /api/v1/purchasing/orders` always returns the current week's draft PO, creating it if none exists. "Current week" = Monday of the current week in the business timezone (America/Chicago, see blocker in STATE.md).

**When to use:** Called on page load of the Order tab; idempotent.

**Example:**
```sql
-- Upsert on week_start; return existing if present
INSERT INTO purchase_orders (week_start, status)
VALUES ($1, 'draft')
ON CONFLICT (week_start) DO UPDATE SET week_start = EXCLUDED.week_start
RETURNING id, week_start, status, created_at;
```

Then fetch line items separately in the same handler. Return combined JSON shape.

**Note:** Week start computed in Go as: find Monday of current week in `time.LoadLocation("America/Chicago")`, then convert to UTC for storage. STATE.md flags this timezone as a blocker — default to `America/Chicago` in the migration, confirm with owner.

### Pattern 2: Upsert-by-natural-key for PO Line Items

**What:** `PUT /api/v1/purchasing/orders/:id/items` accepts the full item list, upserts all rows. Quantity = 0 means remove (or keep dimmed — handled by frontend only sending qty > 0 items, OR by the backend deleting qty=0 rows on upsert).

**Implementation approach:** Frontend sends full list of items with qty > 0 on debounced save. Backend does bulk upsert:

```sql
INSERT INTO po_line_items (po_id, purchase_item_id, quantity, unit, added_by, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (po_id, purchase_item_id)
DO UPDATE SET quantity = EXCLUDED.quantity, updated_at = now()
WHERE purchase_orders.status = 'draft';  -- join guard
```

Guard against writes to non-draft POs: check `purchase_orders.status = 'draft'` in the WHERE clause or a pre-check. Return 409 if PO is not in draft status.

**Status enforcement:** Phase 14 only creates draft POs and writes to draft POs. The 409 path exists but won't be triggered until Phase 16 introduces locking.

### Pattern 3: Suggestions Query

**What:** `GET /api/v1/purchasing/orders/:id/suggestions` returns items below their group's `low_threshold`.

**Implementation:**
```sql
SELECT
  pi.id,
  pi.description,
  pi.photo_url,
  pi.store_location,
  ig.name AS group_name,
  ig.low_threshold,
  COALESCE(sco.quantity, SUM(pli.quantity)::int, 0) AS current_stock,
  GREATEST(1, ig.low_threshold - COALESCE(sco.quantity, SUM(pli.quantity)::int, 0)) AS suggested_qty
FROM purchase_items pi
JOIN item_groups ig ON ig.id = pi.group_id
LEFT JOIN purchase_line_items pli ON pli.purchase_item_id = pi.id
LEFT JOIN stock_count_overrides sco ON sco.item_description = pi.description
GROUP BY pi.id, pi.description, pi.photo_url, pi.store_location, ig.name, ig.low_threshold, sco.quantity
HAVING COALESCE(sco.quantity, SUM(pli.quantity)::int, 0) < ig.low_threshold
ORDER BY ig.name, pi.description;
```

This reuses the exact same stock computation as `GetStockHandler` — no new logic, just a different WHERE/HAVING.

### Pattern 4: Frontend State Machine

**What:** Follow the established state-first rendering pattern from `inventory.html`:

```
loadPage()
  → POST /api/v1/purchasing/orders  → PO_STATE = {id, week_start, line_items[]}
  → GET  /api/v1/purchasing/orders/:id/suggestions → SUGGESTIONS = []
  → GET  /api/v1/inventory/items    → ALL_ITEMS = []
  → renderOrder()
```

Module-level state vars:
```javascript
let PO_STATE = null;           // {id, week_start, status, line_items: [...]}
let SUGGESTIONS = [];          // items below threshold
let ALL_ITEMS = [];            // full catalog for picker
let ITEM_GROUPS = [];          // for category grouping
let SUGGESTION_OPEN = false;   // suggestion checklist visible
let PICKER_OPEN = false;       // item picker modal visible
```

### Pattern 5: Item Picker Modal

Adapted from `inventory.html` lines 1229-1300 (`.item-modal` pattern). Key differences for PO:
- Items already in `PO_STATE.line_items` show "On PO: [qty] [unit]" badge + no [+ Add] button
- Tapping an on-PO item triggers the duplicate popup (D-08), not selection
- Tapping an off-PO item calls `addItemToPO(itemId)` → appends to `PO_STATE.line_items` → triggers debounced save → re-renders

### Pattern 6: `api()` Wrapper

Reuse verbatim from `inventory.html` line 227-234:
```javascript
async function api(path, opts) {
  const res = await fetch(path, opts);
  if (res.status === 401) { window.location.href = '/login.html'; throw new Error('unauthorized'); }
  if (res.status === 204) return null;
  const data = await res.json();
  if (!res.ok) throw Object.assign(new Error(data.error || 'api_error'), { status: res.status, data });
  return data;
}
```

### Recommended Project Structure (purchasing.html)

```javascript
// Module-level state
let PO_STATE = null, SUGGESTIONS = [], ALL_ITEMS = [], ITEM_GROUPS = [];
let SUGGESTION_OPEN = false, ITEM_MODAL_OPEN = false;
let DEBOUNCE_TIMER = null;

// Init
async function init() { await Promise.all([loadPO(), loadItems()]); }

// Rendering
function renderOrder()  { /* renders tab 1 from PO_STATE */ }
function renderSuggestions() { /* inline checklist below banner */ }
function openItemModal() { /* fullscreen picker */ }
function renderItemModal(query) { /* filtered list with on-PO badges */ }
function showDuplicatePopup(item) { /* bottom-sheet popup */ }

// Data mutations
async function addItemToPO(itemId, qty) { /* append to PO_STATE + debounced save */ }
async function addSelectedSuggestions(selected) { /* bulk add from checklist */ }
function updateQty(itemId, delta) { /* optimistic +/- → debounced save */ }
async function savePOItems() { /* PUT /api/v1/purchasing/orders/:id/items */ }

// Helpers
function show(n) { /* tab switch — extends existing pattern to 4 tabs */ }
```

### Anti-Patterns to Avoid

- **Per-item CRUD endpoints:** Do not build separate POST/PATCH/DELETE for each line item. Use single PUT with full state. See ARCHITECTURE.md Anti-Pattern 1.
- **Synchronous data-dependent rendering:** Do not render the tab before `PO_STATE` is loaded. Show "Loading order..." muted text first.
- **Inline onclick on dynamic elements:** Use event delegation via a single click listener on the Order tab container, routing via `data-action` attributes. Project convention from `inventory.html`.
- **Separate Go files for each handler:** Do not split handlers into per-endpoint files. Follow `inventory/handler.go` pattern — one handler file per package.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Fullscreen item picker modal | New modal from scratch | Adapt `.item-modal` from `inventory.html` lines 1229-1300 | Already built, styled, handles search, keyboard, cancel |
| Stock level computation | Duplicate query logic | Reuse exact same COALESCE(sco.quantity, SUM(pli.quantity)) pattern from GetStockHandler | One source of truth for stock levels |
| Category grouping and sticky headers | New group rendering | Reuse `.cat` class and group-by-group_name pattern from `purchasing.html` mock | Already styled with sticky positioning |
| Stepper buttons | New stepper component | Adapt `.stp` class from `purchasing.html` lines 35-39 | Already styled, 32px touch targets, active state |
| Week start calculation | Custom date math | Use Go `time.LoadLocation` + find Monday of current week | DST-aware, correct for America/Chicago |
| Goose migration | Custom migration runner | Goose `-- +goose Up / Down` pattern | Already the project standard |

**Key insight:** This phase is almost entirely assembly — wiring existing UI patterns to new API endpoints. The new code is primarily the Go handler and SQL queries; the JS is adapting existing patterns.

---

## Common Pitfalls

### Pitfall 1: purchase_items Has No photo_url or store_location Yet

**What goes wrong:** Frontend tries to render thumbnails and location notes; fields are null/missing; JS crashes or shows broken images.

**Why it happens:** Migration 0035 (ALTER TABLE purchase_items) must run before any data can populate these fields. In the test environment the database starts empty — items imported later in Phase 15 will fill these in. Phase 14 must render gracefully when both fields are null.

**How to avoid:**
- Render a grey placeholder div (item name initial, e.g. "S" for Salmon) when `photo_url` is null — same pattern inventory.html uses for items without photos.
- `store_location` null: omit the location line from the card middle section silently.
- Migration 0035 must run in the correct order (after 0034, before any API call that returns items in PO context).

**Warning signs:** `img src=""` rendered in DOM; JS errors accessing `item.photo_url` without null check.

### Pitfall 2: PO Suggestions Query Requires Items to Have group_id

**What goes wrong:** Items without a `group_id` (no group assigned) are excluded from suggestions because the JOIN on `item_groups` is INNER. If the test catalog has ungrouped items, they never appear as suggestions even when at zero stock.

**Why it happens:** The threshold (low_threshold) lives on the group. Items with no group have no threshold — logically correct, but the JOIN silently drops them from the list.

**How to avoid:** Document that items with no group will not appear in suggestions. This is correct behavior — not a bug. If the test env has no groups set up, the suggestions list will be empty. The empty state on the Order tab must handle zero suggestions gracefully (hide banner entirely).

### Pitfall 3: Week Start Timezone Mismatch

**What goes wrong:** `POST /api/v1/purchasing/orders` computes Monday of current week in UTC. For a Chicago user (UTC-6), Sunday 7 PM local time is Monday 1 AM UTC — the server creates a new PO for "next week" while the user thinks they're on "this week's" order.

**How to avoid:**
- Compute week start in America/Chicago timezone in Go: `time.Now().In(loc)` → find Monday → convert back to UTC for storage.
- STATE.md blocker: confirm timezone with owner before writing the migration default. Default to `America/Chicago`.
- The `week_start` column is a `DATE` (not TIMESTAMPTZ) — store as the local calendar date, e.g. `2026-04-20` (Monday). No timezone conversion needed on read; it's a human-readable week label.

**Warning signs:** Users see "Week of Apr 20" when it's actually still Sunday.

### Pitfall 4: Optimistic UI Rollback on Failed Save

**What goes wrong:** Stepper +/- updates quantity in DOM immediately (optimistic). If the PUT request fails, the displayed quantity doesn't match server state. On page reload, the quantity reverts — user has no feedback that their change was lost.

**How to avoid:**
- On PUT failure: roll back the in-memory `PO_STATE.line_items[i].quantity` to the pre-edit value and re-render.
- Show inline error: "Couldn't save. Check connection and try again." (from UI-SPEC copywriting contract).
- This rollback is the same pattern used in `inventory.html` for stock count updates.

### Pitfall 5: Item Modal Search Triggers on Every Keystroke

**What goes wrong:** `ALL_ITEMS` can have ~100+ items after Phase 15 import. Filtering on every keystroke with a synchronous loop is fast, but re-rendering the full list on every keypress causes visual jank on older phones.

**How to avoid:** 300ms debounce on the search input event (per UI-SPEC interaction contract). Filter client-side against `ALL_ITEMS` — no network call needed for search. Limit rendered results to 30 items (same as existing `.slice(0, 30)` in inventory.html item modal, line 1263).

### Pitfall 6: Service Worker Cache After Rewrite

**What goes wrong:** After rewriting purchasing.html, the service worker still serves the old cached version to users who have the PWA installed. The rewrite appears to have no effect.

**How to avoid:** Run `task sw` (or `node build-sw.js`) after every HTML/JS change. Workbox generates content-hashed precache entries — the SW automatically serves the new version on next activation. CLAUDE.md convention: "Run `task sw` after changing HTML/JS files."

---

## Code Examples

Verified patterns from existing codebase:

### Go Handler Pattern (from inventory/handler.go)

```go
// Source: backend/internal/inventory/handler.go
func ListVendorsHandler(pool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rows, err := pool.Query(r.Context(), `SELECT id, name FROM vendors ORDER BY name`)
        if err != nil {
            log.Printf("query: %v", err)
            writeError(w, http.StatusInternalServerError, "internal_error")
            return
        }
        defer rows.Close()
        items := []SomeType{}
        for rows.Next() {
            var v SomeType
            if err := rows.Scan(&v.ID, &v.Name); err != nil { ... }
            items = append(items, v)
        }
        writeJSON(w, http.StatusOK, items)
    }
}
```

All purchasing handlers follow this exact closure pattern. No exceptions.

### Goose Migration Pattern (from 0033_stock_count_overrides.sql)

```sql
-- +goose Up
BEGIN;
CREATE TABLE purchase_orders (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  week_start DATE NOT NULL UNIQUE,
  status     TEXT NOT NULL DEFAULT 'draft'
               CHECK (status IN ('draft', 'locked', 'approved')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMIT;

-- +goose Down
BEGIN;
DROP TABLE IF EXISTS purchase_orders;
COMMIT;
```

### Route Registration (from main.go lines 373-395)

```go
// Source: backend/cmd/server/main.go
r.Route("/purchasing", func(r chi.Router) {
    r.Post("/orders", purchasing.GetOrCreateOrderHandler(pool))
    r.Get("/orders/{id}", purchasing.GetOrderHandler(pool))
    r.Put("/orders/{id}/items", purchasing.UpsertLineItemsHandler(pool))
    r.Get("/orders/{id}/suggestions", purchasing.GetSuggestionsHandler(pool))
})
```

Goes inside the authenticated `r.Group` block after the `/inventory` route block.

### Frontend Event Delegation Pattern (from inventory.html)

```javascript
// Source: inventory.html — data-action routing pattern
document.addEventListener('click', function(e) {
    var el = e.target.closest('[data-action]');
    if (!el) return;
    var action = el.dataset.action;
    if (action === 'open-item-picker') openItemModal();
    if (action === 'add-to-po') addItemToPO(el.dataset.itemId);
    if (action === 'dismiss-suggestions') { SUGGESTION_OPEN = false; renderSuggestions(); }
    // ... etc
});
```

### Stock COALESCE Pattern (from GetStockHandler)

```go
// Source: backend/internal/inventory/handler.go line 386
// Reuse this exact pattern in suggestions query
LEFT JOIN stock_count_overrides sco ON sco.item_description = sub.description
-- COALESCE picks override if present, falls back to summed purchase history
COALESCE(sco.quantity, SUM(pli.quantity)::int, 0) AS current_stock
```

---

## State of the Art

| Old Approach | Current Approach | Impact |
|--------------|------------------|--------|
| Hardcoded CATS array in purchasing.html mock | Live data from `/api/v1/purchasing/orders` + `/api/v1/inventory/items` | Phase 14 output |
| 3-tab Form/Locked/PO layout | 4-tab Order/Shopping/PO/History layout | Tab structure matches full v3.0 milestone |
| Static stepper with no persistence | Debounced PUT to `po_line_items` table | PO survives page reload |
| No reorder suggestions | Suggestions from stock threshold JOIN | Workflow closes the inventory → purchasing loop |

**Confirmed not-yet-done:**
- `purchase_items.photo_url` and `store_location` columns do not exist yet — verified by reading `0024_inventory.sql` (CREATE TABLE purchase_items has only id, description, group_id)
- `StockItem` type has no `PhotoURL` or `StoreLocation` fields — confirmed from `types.go`
- No `internal/purchasing` package exists yet — confirmed by reading `backend/internal/` directory

---

## Open Questions

1. **Business timezone confirmation**
   - What we know: STATE.md flags "Confirm business timezone (likely `America/Chicago`) before writing migration 0034 default"
   - What's unclear: Whether the owner operates on Chicago time or a different central US timezone
   - Recommendation: Default migration to `America/Chicago`. Planner should add a task note: "owner to confirm before deploy." Do not block Phase 14 on this — the PO form works regardless; the timezone only affects week-start boundary edge cases.

2. **User initials format**
   - What we know: The mock shows "JM MK TR" (first+last initials). The `users` table has `name` column. `added_by` on `po_line_items` is a UUID FK to users.
   - What's unclear: Whether the API should return pre-computed initials or the full display name (let frontend compute initials).
   - Recommendation: API returns `added_by_name TEXT` (full display name). Frontend computes initials: `name.split(' ').map(w => w[0]).join('').toUpperCase().slice(0,2)`. Matches the pattern used in `workflows.html` for checklist attribution.

3. **`PUT /orders/:id/items` — full-replace vs. delta**
   - What we know: Architecture research recommends full-state send (qty > 0 items only). This matches `pending_purchases.items` and `stock_count_overrides` patterns.
   - What's unclear: Should qty=0 items be deleted from `po_line_items` or kept with qty=0?
   - Recommendation: Delete rows where qty reaches 0 via the frontend (don't include them in the PUT payload). Backend upserts only what's sent. This keeps the table clean and avoids an "is qty=0 a deletion or a zero-order?" ambiguity.

---

## Environment Availability

Step 2.6: SKIPPED — Phase 14 has no new external dependencies beyond what already runs in the project. The Go backend, Postgres, and static file server are all operational. No new services (Zoho, SMTP, Notion) are introduced in this phase.

---

## Project Constraints (from CLAUDE.md)

These directives apply to all work in this phase:

| Constraint | Source | Applies To |
|------------|--------|------------|
| No build step — plain HTML, vanilla JS | CLAUDE.md Constraints | purchasing.html rewrite |
| Mobile-first, max-width 480px | CLAUDE.md Constraints | All new UI |
| CSS variables and dark mode support | CLAUDE.md Conventions | purchasing.html `:root` block |
| `task sw` / `node build-sw.js` after HTML/JS changes | CLAUDE.md Conventions | Every task that touches purchasing.html |
| Event delegation via `data-action` attributes | CLAUDE.md / inventory.html pattern | All dynamic UI in purchasing.html |
| `SCREAMING_SNAKE_CASE` for module constants, `camelCase` for functions | CLAUDE.md Conventions | purchasing.html JS |
| State-first rendering: mutate state → call render → DOM updates | CLAUDE.md Architecture | renderOrder(), renderSuggestions() |
| No localStorage — all data from API | CLAUDE.md Conventions | PO_STATE must come from API |
| GSD workflow enforcement — no direct edits outside GSD | CLAUDE.md GSD | All file changes |
| Double-tap zoom prevention via `dblclick` listener | CLAUDE.md / existing pages | purchasing.html boilerplate |
| `api()` wrapper for all fetch calls (401 redirect, 204 handling) | inventory.html established pattern | All purchasing.html API calls |
| Go backend: pgx v5, chi router, goose migrations | Existing stack | purchasing package |
| Playwright E2E tests via `task test` | CLAUDE.md Conventions | Any test additions |

---

## Sources

### Primary (HIGH confidence)
- `/Users/jamal/projects/yumyums/hq/backend/internal/inventory/handler.go` — handler pattern, stock query with COALESCE, writeJSON/writeError helpers
- `/Users/jamal/projects/yumyums/hq/backend/internal/inventory/types.go` — Go type patterns; confirmed photo_url/store_location absent from PurchaseItem
- `/Users/jamal/projects/yumyums/hq/backend/internal/db/migrations/0024_inventory.sql` — purchase_items schema; confirmed no photo_url column
- `/Users/jamal/projects/yumyums/hq/backend/internal/db/migrations/0033_stock_count_overrides.sql` — confirmed next migration is 0034
- `/Users/jamal/projects/yumyums/hq/backend/cmd/server/main.go` — route registration pattern, confirmed no `/purchasing` routes yet
- `/Users/jamal/projects/yumyums/hq/purchasing.html` — full mock read; reusable CSS classes (.cat, .row, .stp, .tabs, .hd, .pill)
- `/Users/jamal/projects/yumyums/hq/inventory.html` lines 227-234 — api() wrapper verbatim
- `/Users/jamal/projects/yumyums/hq/inventory.html` lines 509-548 — stock rendering + reorder section logic
- `/Users/jamal/projects/yumyums/hq/inventory.html` lines 1229-1300 — item picker modal pattern
- `.planning/phases/14-po-backend-order-form/14-CONTEXT.md` — locked decisions, deferred scope
- `.planning/phases/14-po-backend-order-form/14-UI-SPEC.md` — component specs, copywriting contract, interaction contract
- `.planning/research/ARCHITECTURE.md` — schema design, endpoint design, Go package structure, anti-patterns
- `.planning/research/FEATURES.md` — feature landscape, ordering-to-shopping loop
- `.planning/research/PITFALLS.md` — PO state machine risks, timezone pitfalls, optimistic locking requirement

### Secondary (MEDIUM confidence)
- `.planning/STATE.md` — confirmed timezone blocker, roadmap decisions
- `.planning/REQUIREMENTS.md` — PO-01 through PO-08 confirmed as Phase 14 scope

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — read directly from live codebase
- Architecture: HIGH — derived from existing handler and migration patterns in the repo
- Pitfalls: HIGH — photo_url absence confirmed by reading schema; timezone risk confirmed by STATE.md; SW cache confirmed by CLAUDE.md convention

**Research date:** 2026-04-22
**Valid until:** 2026-05-22 (stable codebase; only invalidated by schema changes to purchase_items or new competing purchasing routes)
