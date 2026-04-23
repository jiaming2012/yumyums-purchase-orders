# Phase 7: Stock and Reorder Tab - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace the Stock tab "Coming Soon" placeholder with a fully functional stock level view: item groups grouped by tag category with Low/Medium/High badges derived from a sales-based consumption algorithm, expandable item detail with last purchase info, reorder suggestions, and manual stock level override with reason.

</domain>

<decisions>
## Implementation Decisions

### Stock Level Display
- **D-01:** Items grouped by tag category (Beef, Produce, Supplies, etc.) with emoji headers. Each item group shows name + colored stock badge (🟢 High / 🟡 Medium / 🔴 Low).
- **D-02:** Tap an item row to expand inline showing: last purchased date, last vendor, average purchase frequency, last price, estimated remaining quantity, and Override Level button.
- **D-03:** Tag sections are collapsible. Items within each tag sorted by urgency (Low first).

### Estimation Algorithm
- **D-04:** Sales-based consumption algorithm, NOT time-based. Stock = totalPurchased - totalConsumed. Consumed is derived from sales data × ingredient ratios.
- **D-05:** Add `MOCK_SALES` data — weekly sales counts per menu item (e.g., `{menuItemId:'cheesesteak', weekOf:'2026-04-07', qty:45}`). Spans the same date range as purchase events.
- **D-06:** `MOCK_MENU_ITEMS` already exists in inventory.html with ingredient ratios. Use these to calculate: consumed = sum(salesQty × ingredientRatio) for each purchase item group.
- **D-07:** Stock level thresholds based on remaining quantity relative to average weekly consumption: High (> 1 week supply), Medium (0.5-1 week supply), Low (< 0.5 week supply). Claude determines exact formula.
- **D-08:** Items with no purchase history default to "Unknown" (grey badge).

### Reorder Suggestions UX
- **D-09:** Claude's discretion — decide between a separate highlighted section at top of Stock tab vs inline badges, based on what works best for the mock data distribution and mobile viewport.

### Manual Override Flow
- **D-10:** Tap "Override Level" in expanded item detail to show inline override form: 3 radio buttons (Low/Medium/High), a reason text input, and a Save Override button.
- **D-11:** Override replaces the calculated badge visually. An "Overridden" indicator appears next to the badge so it's clear this is a manual level, not calculated.
- **D-12:** Override is stored in a `STOCK_OVERRIDES` dict (in-memory, resets on refresh). Mock journal entry pattern — real backend would persist this as a journal entry.
- **D-13:** User can clear an override to return to the calculated level.

### Claude's Discretion
- Reorder suggestions layout (separate section vs inline — D-09)
- Exact stock formula and threshold values
- How "estimated remaining" quantity is displayed (lbs, units, or generic)
- Empty state if no items in a tag category
- Color coding for badges (use semantic colors from CSS variables)
- How MOCK_SALES data is structured (weekly buckets vs individual transactions)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing Inventory Code
- `inventory.html` — Current implementation with 4-tab shell, mock data layer (`MOCK_ITEM_GROUPS`, `MOCK_TAGS`, `MOCK_MENU_ITEMS`, `MOCK_PURCHASES`, `MOCK_PURCHASE_EVENTS`), `parseLocalDate()`, event delegation pattern.

### Phase 6 Context
- `.planning/phases/06-foundation-and-history-tab/06-CONTEXT.md` — Mock data design decisions, tab layout, architecture notes.

### Research
- `.planning/research/FEATURES.md` — Stock estimation approaches, reorder suggestion patterns.
- `.planning/research/ARCHITECTURE.md` — Data flow for stock calculations.

### Data Model Reference
- Baserow export at `/Users/jamal/Downloads/export_e77d0af2-d3d3-4e25-a9cd-2893fc8e2afe/` — PurchaseItemGroup and Tag relationships.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `MOCK_ITEM_GROUPS` (10 groups), `MOCK_TAGS` (5 tags), `MOCK_MENU_ITEMS` (4 items with ingredient ratios) — already in inventory.html
- `MOCK_PURCHASES` (64 line items) and `MOCK_PURCHASE_EVENTS` (14 events) — source data for "total purchased"
- Expand/collapse pattern from History tab (`EXPANDED_EVENTS` dict) — reuse as `EXPANDED_STOCK_ITEMS`
- `parseLocalDate()` utility for date calculations

### Established Patterns
- Event delegation on tab containers (`data-action` routing)
- State-first rendering: mutate state → call render function → DOM updates
- Card/row expand-collapse with dict tracking
- `show(n)` re-renders on tab switch

### Integration Points
- Replace `#stock-container` Coming Soon placeholder with `renderStock()` function
- Add `MOCK_SALES` data constant alongside existing mock data
- Add `STOCK_OVERRIDES` state dict
- sw.js cache bump after implementation

</code_context>

<specifics>
## Specific Ideas

- Stock estimation is consumption-based (sales × ingredient ratios), not time-based — this is the key differentiator from simple inventory tools
- Override flow creates a mock "journal entry" pattern that the backend will implement as a real audit trail
- Tag grouping mirrors how the owner thinks about inventory — by food category, not alphabetically

</specifics>

<deferred>
## Deferred Ideas

- Backend journal entry persistence for overrides — mock pattern in place
- Integration with Purchasing app for actual reorder actions — display only for now
- isCase weighting in consumption calculations — keep it simple for v1.1

</deferred>

---

*Phase: 07-stock-and-reorder-tab*
*Context gathered: 2026-04-14*
