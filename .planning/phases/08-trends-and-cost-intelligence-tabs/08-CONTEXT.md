# Phase 8: Trends and Cost Intelligence Tabs - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Replace the Trends and Cost "Coming Soon" placeholders with fully functional tabs: Trends tab with Chart.js spending charts (bar by category, pie/doughnut for proportions, timeline over time) and tag filter chips; Cost tab with menu item cost breakdown and purchase item reverse lookup. Metabase swap is a TODO comment only (INTG-02).

</domain>

<decisions>
## Implementation Decisions

### Chart Types & Layout (Trends Tab)
- **D-01:** Trends tab uses sub-tabs: "By Category" and "Over Time". Each sub-tab shows its own charts — less scrolling on mobile.
- **D-02:** "By Category" sub-tab shows: horizontal bar chart (spending by tag category) + pie/doughnut chart (proportion) stacked vertically.
- **D-03:** "Over Time" sub-tab shows: line or bar chart with monthly/weekly spending totals. Tag filter applies here too.
- **D-04:** Chart.js 4.5.1 already loaded. Canvas elements wrapped in `position:relative` divs with explicit height (220px). Charts must be destroyed before recreating on tab/sub-tab switch.

### Tag Filter UX
- **D-05:** Tappable chip bar above the charts. Tags: All, 🥩 Proteins, 🥬 Produce, 🧀 Dairy, 📦 Supplies, 🥤 Beverages. Multi-select supported — tap multiple chips to include those categories. "All" resets. Active chips highlighted.
- **D-06:** Filter applies to all charts on the Trends tab — bar, pie, and timeline all reflect the selected tags.

### Cost Intelligence Views (Cost Tab)
- **D-07:** Cost tab uses sub-tabs: "Menu Items" and "Ingredients".
- **D-08:** "Menu Items" sub-tab: card per menu item showing name + estimated total cost. Tap to expand and see ingredient proportion table (ingredient name, amount, unit cost, line cost).
- **D-09:** "Ingredients" sub-tab: card per purchase item group. Tap to expand and see which menu items use it — showing relative percentage, cost contribution, revenue, and return on purchase.
- **D-10:** Cost calculations use `MOCK_MENU_ITEMS` ingredient ratios × average unit prices from `MOCK_PURCHASES`. Revenue from `MOCK_SALES` × menu item sale price. All mock data.

### Metabase Swap (INTG-02)
- **D-11:** TODO comment only — no runtime switching infrastructure. Comments in `renderTrends()` and `renderCost()` explain how to replace with Metabase iframe. The `#trends-container` and `#cost-container` wrapper divs are already in place from Phase 6.

### Claude's Discretion
- Chart color palette (should read from CSS variables for dark mode compatibility — research pitfall)
- Bar chart orientation (horizontal vs vertical) for category spend
- Timeline granularity (weekly vs monthly — weekly aligns with purchasing cadence)
- How to display "return on purchase" in the Ingredients reverse view (percentage, dollar amount, or both)
- Exact chip styling (active vs inactive states)
- Chart.js tooltip configuration
- `MOCK_MENU_ITEMS` sale price field (needed for revenue calculation — add if not present)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing Inventory Code
- `inventory.html` — Current implementation with 4 tabs, Chart.js loaded, all mock data (`MOCK_TAGS`, `MOCK_ITEM_GROUPS`, `MOCK_PURCHASES`, `MOCK_PURCHASE_EVENTS`, `MOCK_MENU_ITEMS`, `MOCK_SALES`), `parseLocalDate()`, event delegation.

### Research
- `.planning/research/STACK.md` — Chart.js 4.5.1 UMD details, dark mode wiring via `getComputedStyle`.
- `.planning/research/PITFALLS.md` — Chart destroy/recreate, canvas sizing, UTC date shift, dark mode color wiring.
- `.planning/research/FEATURES.md` — Spending trend patterns, chart type recommendations.

### Prior Phase Contexts
- `.planning/phases/06-foundation-and-history-tab/06-CONTEXT.md` — Chart.js integration decisions (D-15, D-16), `parseLocalDate()` utility.
- `.planning/phases/07-stock-and-reorder-tab/07-CONTEXT.md` — Sales-based consumption algorithm, `MOCK_SALES` data structure, `MOCK_MENU_ITEMS` ingredient ratios.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Chart.js 4.5.1 at `lib/chart.umd.min.js` — already loaded and verified (`typeof Chart === 'function'`)
- `#trends-container` and `#cost-container` div IDs already in place
- `MOCK_TAGS` with emoji mapping (🥩 Proteins, 🥬 Produce, etc.)
- `MOCK_MENU_ITEMS` with ingredient arrays and `itemGroupId` references
- `MOCK_SALES` with weekly sales counts per menu item
- `parseLocalDate()` for date grouping
- `calcStockLevels()` — can be referenced for average unit price calculations

### Established Patterns
- `show(n)` with re-render on tab switch
- Event delegation on tab containers
- Sub-tab pattern (used in Manager tab Active/Completed — reuse for Trends and Cost sub-tabs)
- Chart destroy before recreate (pitfalls research)

### Integration Points
- Replace `#trends-container` innerHTML (currently "Coming Soon")
- Replace `#cost-container` innerHTML (currently "Coming Soon")
- Add `renderTrends()` and `renderCost()` functions
- Wire into `show()` dispatch for ACTIVE_TAB===2 (Trends) and ACTIVE_TAB===4 (Cost)
- sw.js cache bump after implementation

</code_context>

<specifics>
## Specific Ideas

- Chart colors should use CSS variables via `getComputedStyle` for dark mode — not hardcoded hex values
- Sub-tab pattern keeps mobile viewport clean — no need to scroll past 3 charts
- Ingredient proportions in the Cost tab are the foundation for future AI-assisted cost estimation
- Tag filter chips are multi-select (unlike the vendor dropdown which is single-select)

</specifics>

<deferred>
## Deferred Ideas

- Metabase iframe embed — TODO comment only, infrastructure not built
- AI-assisted cost estimation — backend feature, UI schema in place
- Export chart data as CSV — future enhancement
- Date range picker for trends — could use a dropdown for predefined ranges (last month, last 3 months, all time)

</deferred>

---

*Phase: 08-trends-and-cost-intelligence-tabs*
*Context gathered: 2026-04-14*
