# Phase 14: PO Backend + Order Form - Context

**Gathered:** 2026-04-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace the purchasing.html mock with a live purchase order form backed by real inventory data. Users can add items from reorder suggestions and catalog search, see photos/store/attribution per item, and adjust quantities. The form persists to a Go + Postgres API. This phase builds the Order tab and the schema foundation for all 4 tabs (Order / Shopping / PO / History), but only the Order tab is fully interactive in this phase. PO, Shopping, and History tabs get their full logic in Phases 16-17.

</domain>

<decisions>
## Implementation Decisions

### PO Form Layout
- **D-01:** Photo card layout per item — thumbnail, name, unit, store location, stepper (+/-), and "Added by" attribution
- **D-02:** Items grouped by category with sticky headers (matches existing Stock tab and mock PO patterns)
- **D-03:** Each card is ~80px tall with photo on the left, info in the middle, stepper on the right

### Reorder Suggestions
- **D-04:** Suggestion banner at top of Order tab showing count ("3 items need restock [+]")
- **D-05:** Tapping [+] opens a checklist of items below stock threshold; user selects which to add with suggested quantities pre-filled
- **D-06:** Suggestions come from existing inventory Stock tab threshold logic (items where qty < group low_threshold)

### Duplicate Handling
- **D-07:** Item picker shows "On PO: [qty]" badge for items already on the current order
- **D-08:** Tapping an already-added item shows popup: "[Item] already on PO — Qty: [N] · Added by: [initials]" with "Go to item" link and "Dismiss"
- **D-09:** Items not on the PO show [+ Add] button in the picker

### Tab Structure
- **D-10:** 4 tabs in this order: Order / Shopping / PO / History
- **D-11:** **Order tab** — Always open for contributions. After weekly cutoff, the current list moves to PO tab and a fresh Order list starts for next week. There is always an active order to add to.
- **D-12:** **PO tab** — Frozen cutoff list. Admin-editable. Vendor-grouped with pricing estimation (like current mock). Approve button generates the Shopping list. Admin blocked from approving if previous week's Shopping list is still active.
- **D-13:** **Shopping tab** — Approved shopping checklist organized by vendor. One submit/complete button per vendor section to support multiple shoppers splitting by vendor. Empty state when nothing is pending.
- **D-14:** **History tab** — Past shopping lists with missing items shown and who completed each list.

### Shopping List Per-Vendor Submit
- **D-15:** Shopping list has one submit button per vendor section, not one global submit — supports multiple shoppers each handling a different vendor/store

### History Attribution
- **D-16:** History entries show who completed each shopping list (per-vendor completion tracked)

### Claude's Discretion
- Empty state design for Order tab (first time, no items added yet)
- Exact stepper button styling (reuse existing mock pattern or refresh)
- Search input placement and debounce timing in item picker
- Skeleton/loading states while fetching inventory data

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing purchasing mock
- `purchasing.html` — Current 3-tab mock with CATS data, stepper pattern, vendor-grouped PO view. This file gets rewritten.

### Inventory integration
- `inventory.html` lines 510-548 — Reorder suggestion rendering logic (threshold comparison, badge generation)
- `inventory.html` lines 1229+ — Fullscreen item picker modal pattern (used in receipt review)
- `inventory.html` lines 530-535 — Stock threshold logic (low_threshold from groups)

### Research
- `.planning/research/ARCHITECTURE.md` — New tables schema (purchase_orders, po_line_items), endpoint design
- `.planning/research/FEATURES.md` — Feature landscape, ordering-to-shopping loop
- `.planning/research/PITFALLS.md` — PO state machine risks, optimistic locking requirement

### Notion data reference
- `/Users/jamal/Downloads/ExportBlock-99603bef-0b4f-4395-91a5-37036f190626-Part-1/All Items 136643f80250811f980cda0a9127c6d8.csv` — Item catalog with categories, photos, stores, quantities (~100 items)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Reorder suggestion logic** (inventory.html): Computes items below `low_threshold` per group — reuse for PO suggestion banner
- **Fullscreen item picker modal** (inventory.html): Existing pattern for searching and selecting items from catalog — adapt for PO item search
- **Stepper +/- buttons** (purchasing.html mock): CSS and interaction pattern exists — refresh with photo card layout
- **`api()` wrapper** (workflows.html): Async fetch with 401 redirect, 204 handling, JSON error parse — reuse for purchasing endpoints

### Established Patterns
- **Event delegation** via `data-action` attributes on container divs — all tool pages use this
- **Tab switching** via `show(n)` function toggling `style.display` — consistent across all pages
- **State-first rendering**: Mutate JS state → call render function → DOM updates from state
- **Sticky category headers**: CSS `position:sticky` used in current mock and inventory Stock tab

### Integration Points
- **Inventory API**: `/api/v1/inventory/items` and `/api/v1/inventory/stock` — source data for PO form
- **Auth/RBAC**: Session cookies + role checks for admin-only PO editing after cutoff
- **SW precache**: `build-sw.js` must include purchasing.html (already does)

</code_context>

<specifics>
## Specific Ideas

- PO tab pricing layout should match current mock: vendor name + total on left/right, line items below with per-item cost
- Reorder suggestion banner should feel like a notification — not intrusive but visible
- "Added by" shows user initials (matching the pattern used in workflows.html for checklist attribution)
- When admin is blocked from approving (previous shopping list still active), show clear message: "Complete last week's shopping list before approving this one"

</specifics>

<deferred>
## Deferred Ideas

- Shopping list check-off and completion flow — Phase 16
- Cutoff enforcement and admin approval — Phase 16
- Alert notifications (cutoff reminder, missing items) — Phase 17
- Repurchase badges on inventory — Phase 17
- Price tracking and cost comparison across vendors — future milestone

</deferred>

---

*Phase: 14-po-backend-order-form*
*Context gathered: 2026-04-22*
