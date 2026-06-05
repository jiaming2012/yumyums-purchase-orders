# Phase 6: Foundation and History Tab - Context

**Gathered:** 2026-04-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Create `inventory.html` with page shell (4 tabs, Chart.js loaded locally, PWA boilerplate), mock data layer modeled after Baserow purchase schema, HQ integration (launcher tile, SW cache), and a fully functional Purchase History tab with event cards, expandable line items, and vendor dropdown filter. Trends/Stock/Cost tabs show "Coming Soon" placeholders.

</domain>

<decisions>
## Implementation Decisions

### History Tab UX
- **D-01:** Card list layout — each PurchaseEvent is a card showing vendor name, date, and total spend. Tap to expand and see line items below (name, qty, price, case flag).
- **D-02:** Events sorted newest-first by default.
- **D-03:** Vendor filter via `<select>` dropdown at top of tab: "All vendors" default, plus one option per vendor from mock data. Filtering re-renders the event list immediately.
- **D-04:** Line items show: item name, quantity (with "case" badge if isCase), and price formatted as currency.

### Mock Data Design
- **D-05:** Baserow schema structure (field names and relationships) with fabricated values. Focus on UI realism — data can be sourced from real DB later.
- **D-06:** Fabricated data should span 3+ months, 3-4 vendors, 12+ purchase events, 5+ tags, 8-10 item groups with varying purchase frequency. Enough to exercise charts in Phase 8.
- **D-07:** Data model constants: `MOCK_VENDORS`, `MOCK_PURCHASE_EVENTS`, `MOCK_PURCHASES` (line items), `MOCK_PURCHASE_ITEMS`, `MOCK_ITEM_GROUPS`, `MOCK_TAGS`, `MOCK_MENU_ITEMS` (for Phase 8 cost intelligence).
- **D-08:** `parseLocalDate(str)` utility to avoid UTC date shift — splits date string instead of using `new Date("YYYY-MM-DD")`.

### Tab Layout & Navigation
- **D-09:** 4 tabs: History / Trends / Stock / Cost. All visible to all roles (no RBAC gating in mock — note for backend later).
- **D-10:** Tabs use the established `show(n)` pattern with re-render on each tab switch.
- **D-11:** Trends, Stock, and Cost tabs show "Coming Soon" placeholder with brief description of what will appear there.

### HQ Integration
- **D-12:** Add Inventory as a new 8th tile on the HQ launcher (keep BI as "Soon"). Use 📦 emoji. Grid becomes 4×2.
- **D-13:** Add `inventory.html` to `sw.js` ASSETS array and bump cache version.
- **D-14:** Add `{slug:'inventory', name:'Inventory', icon:'📦'}` to `APPS` array in `users.html`.

### Chart.js Integration
- **D-15:** Chart.js 4.5.1 UMD build served as local asset at `lib/chart.umd.min.js` (downloaded to repo, not loaded from CDN). Added to SW ASSETS for offline use.
- **D-16:** Chart.js `<script>` tag loaded before the inline `<script>` block. Canvas wrapped in `position:relative` div with explicit height (220px) for mobile.

### Architecture Notes (for backend later)
- **D-17:** Tab RBAC gating deferred to backend implementation. Comment in code: `// TODO: gate Trends/Cost to manager+ via backend roles`.
- **D-18:** Trends and Cost tab containers use swappable `<div id="trends-container">` / `<div id="cost-container">` pattern for future Metabase iframe replacement (INTG-02).

### Claude's Discretion
- Exact card styling for purchase event cards (follow existing HQ card patterns)
- How "case" badge displays on line items
- "Coming Soon" placeholder text and styling for each tab
- Mock data item names and prices (realistic food truck items)
- Chart.js download method (curl/wget in plan task vs manual)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing App Patterns
- `onboarding.html` — Most recent tool page (1121 lines). Reference for 3-tab → 4-tab extension, `show(n)` with re-render, event delegation, state-first rendering.
- `workflows.html` — Card patterns, expand/collapse, progress bars.
- `index.html` — HQ launcher grid where new tile must be added.

### Data Model Reference
- Baserow export at `/Users/jamal/Downloads/export_e77d0af2-d3d3-4e25-a9cd-2893fc8e2afe/` — Real schema: Vendor, PurchaseEvent, Purchase, PurchaseItem, PurchaseItemGroup, Tag. Use field names and relationships as mock data template.

### Infrastructure
- `sw.js` — Cache list and version. Must add `inventory.html` and `lib/chart.umd.min.js`.
- `users.html` — APPS array for permissions.
- `.planning/codebase/CONVENTIONS.md` — Naming patterns, CSS variables, code style.
- `.planning/codebase/STRUCTURE.md` — "Where to Add New Code" checklist.

### Research
- `.planning/research/STACK.md` — Chart.js 4.5.1 UMD CDN details, dark mode wiring.
- `.planning/research/ARCHITECTURE.md` — Data flow, integration points, build order.
- `.planning/research/PITFALLS.md` — Chart.js UMD path, canvas sizing, chart destroy/recreate, UTC date shift.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- CSS variable block with dark mode — copy from onboarding.html
- PWA boilerplate (viewport meta, manifest link, SW registration, dblclick prevention)
- `ptr.js` — pull-to-refresh, load via script tag
- `show(n)` tab switching with re-render pattern
- Card CSS class from all existing tool pages

### Established Patterns
- State-first rendering: mutate JS state → call render function → DOM updates
- Event delegation with `data-action` on a body container
- `EXPANDED_*` dict for expand/collapse state (from onboarding sections)
- Dropdown `<select>` for filtering (from onboarding builder role selector)

### Integration Points
- `index.html` — Add 8th tile (Inventory 📦) as active link to `inventory.html`
- `sw.js` — Add `'./inventory.html'` and `'./lib/chart.umd.min.js'` to ASSETS, bump version
- `users.html` — Add inventory entry to APPS array

</code_context>

<specifics>
## Specific Ideas

- Use the Baserow export as a reference for realistic mock data field names and relationships, but fabricate values spread across 3+ months for chart exercises
- Chart.js must be a local file (not CDN) to work offline with the service worker's cache-first strategy
- The `parseLocalDate()` utility is critical — must be defined before any date rendering or grouping

</specifics>

<deferred>
## Deferred Ideas

- Tab RBAC gating — backend implementation, noted with TODO comment
- Metabase iframe embed for Trends/Cost — INTG-02 architecture in place but not activated
- Real data sourcing from backend API — mock data for now

</deferred>

---

*Phase: 06-foundation-and-history-tab*
*Context gathered: 2026-04-14*
