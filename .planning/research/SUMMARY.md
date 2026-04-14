# Project Research Summary

**Project:** Yumyums HQ v1.1 — Inventory App
**Domain:** Food truck inventory analytics — purchase history, spending trends, stock estimation as a standalone tool page inside an existing vanilla JS PWA
**Researched:** 2026-04-14
**Confidence:** HIGH (stack and architecture derived from codebase inspection; features from industry sources; pitfalls from official docs and GitHub issues)

## Executive Summary

The v1.1 Inventory milestone adds a standalone `inventory.html` analytics tool to the existing Yumyums HQ PWA. The tool answers three operator questions from existing Baserow purchase data: what was bought and when, where money is going by category, and whether key items are running low. The correct build approach is to follow the established tool page contract exactly — self-contained HTML with inlined CSS/JS/mock data, the shared CSS variable block, pull-to-refresh via `ptr.js`, and service worker coverage via `sw.js`. The only new external dependency is Chart.js 4.5.1 (UMD build), which must be downloaded and served as a local asset rather than loaded from CDN to preserve the existing offline-first guarantee.

The recommended architecture is a 3-tab layout (History / Trends / Stock) using the state-first rendering and event delegation patterns established in `workflows.html`. Mock data must be rich enough to exercise all features — at minimum 12 purchase events across 2 calendar months, 3-4 vendors, and 5 category tags. The spending trends view uses Chart.js bar and line charts; the stock estimation uses a purchase-frequency heuristic requiring no library. The full data chain (PurchaseEvent → Purchase → PurchaseItemGroup → Tag) is the critical dependency: category breakdown and trend charts both require walking this full chain, so the mock data constants must represent it correctly before any rendering code is written.

The biggest risks are Chart.js integration pitfalls (ESM vs UMD bundle, canvas sizing on mobile, chart instance leaks on tab switch, CDN not cached by SW) and date grouping UTC shift bugs. All are preventable by addressing them in Phase 1 before any feature logic is written. One minor conflict exists between ARCHITECTURE.md (CSS-only bars) and STACK.md (Chart.js): resolve in favor of Chart.js served locally, which satisfies both the feature requirement and offline constraint.

## Key Findings

### Recommended Stack

The existing stack (vanilla JS ES2020+, CSS custom properties, SortableJS, Playwright) requires no changes. The single addition is Chart.js 4.5.1 via its UMD bundle, downloaded and committed to the repo at `/lib/chart.umd.min.js`. The UMD build exposes `Chart` as a global usable directly in a `<script>` block — no import statement, no bundler. Date formatting and grouping are handled by the native `Intl.DateTimeFormat` API (available iOS Safari 10+) and hand-rolled reduce logic (~15 lines). Stock level classification is pure JS arithmetic against a `parDays` threshold — no library needed.

**Core technologies:**
- Vanilla JS (ES2020+): all rendering, state, and computation — existing convention, no exception
- CSS custom properties: dark mode and chart theming — `getComputedStyle()` pattern reads resolved CSS vars into Chart.js at render time; no separate theme configuration
- Chart.js 4.5.1 (UMD, local asset): bar and line charts — only no-build-step option with trivial vanilla JS API (`new Chart(ctx, config)`), genuine mobile canvas responsiveness, and ~60KB min+gzip; pinned to 4.5.1 (last release Oct 2024)
- `Intl.DateTimeFormat` (native): date label formatting — no polyfill needed, ships in every target browser including iOS Safari 10+
- Playwright (existing): E2E regression — add `inventory.spec.js` following existing patterns

**What not to use:**
- `chart.js@latest` CDN tag: floating version breaks offline-cached PWA on Chart.js releases
- Recharts, Victory, Nivo: React-dependent, require bundler
- date-fns v2+: no UMD bundle, incompatible with no-build-step constraint
- CDN loading of Chart.js: CDN URLs not in SW precache; charts go blank offline
- ECharts: ~750KB bundle, complex API for bar/line use case

### Expected Features

The full feature set defined in PROJECT.md is all P1 for v1.1. Research against Toast, MarketMan, and NetSuite confirms these as table stakes for a food truck inventory analytics tool.

**Must have (table stakes):**
- Purchase history list with expandable line items — minimum viable view; answers "what did we buy?"
- Vendor filter on history — operators with 2-3 vendors filter immediately on first use
- Category spending breakdown (by Tag) — "where is money going?" is the primary analytics question
- Monthly spending trend chart — visualizes cost drift over time; Chart.js line chart with monthly buckets
- Stock level estimates (Low/Medium/High) per PurchaseItemGroup — frequency heuristic, zero crew effort required
- Reorder suggestions list (Low + Medium items only, sorted by urgency) — actionable output for next PO day
- HQ integration (tile, SW cache bump, users permissions) — tool is unreachable without this

**Should have (competitive differentiators — add post-validation):**
- Average price tracking per item — detects supplier price increases across purchase events
- Spend-per-vendor summary — supports vendor negotiation with aggregate totals
- Date range filter on trends — meaningful once 6+ months of real data exists

**Defer to v2+:**
- Backend integration with live Baserow data — mock phase must validate UX first
- Recipe costing / BOM — requires a separate menu data model not in scope
- Push notifications for low stock — requires backend infrastructure
- Real-time stock counting / physical counts — too much crew overhead; frequency heuristic is the right tradeoff at this scale

**Key differentiator vs. commercial tools:** Zero manual count effort. The stock estimation heuristic uses existing purchase event data — no crew action required. Commercial tools (MarketMan, Toast Inventory) require daily physical counts.

### Architecture Approach

`inventory.html` is a fully self-contained standalone tool page following the established tool page contract: shared CSS variable block, back link to `index.html`, SW registration, `ptr.js` pull-to-refresh. All CSS, JS, and mock data live inline in the file — no imports, no shared modules. Four existing files require targeted modifications.

**Major components:**

1. `inventory.html` — standalone tool: 3-tab layout (History / Trends / Stock), all mock data constants, all rendering logic, Chart.js chart management
2. Mock data layer — six constants (`VENDORS`, `PURCHASE_EVENTS`, `PURCHASES`, `PURCHASE_ITEMS`, `PURCHASE_ITEM_GROUPS`, `TAGS`) defined before rendering code; partial denormalization (`vendorName` inline on events, `tagNames[]` inline on groups) to keep render functions readable
3. State + rendering engine — `ACTIVE_TAB`, `FILTER_TAG` state variables; `show(n)` dispatches to `renderHistory()` / `renderTrends()` / `renderStock()`; event delegation via `data-action` on container divs for dynamic lists
4. Chart management — module-level `let spendChart = null` references per canvas; destroy-before-recreate on every tab show; CSS var colors resolved at render time via `getComputedStyle()`
5. SW + HQ integration — `inventory.html` and `/lib/chart.umd.min.js` both added to `ASSETS` in `sw.js`; cache version string bumped; `index.html` BI tile replaced with active Inventory tile; `users.html` permissions added

**Files modified:**

| File | Change |
|------|--------|
| `inventory.html` | NEW — standalone tool page |
| `index.html` | Replace "BI" soon-tile with active Inventory tile |
| `sw.js` | Bump `CACHE` version; add `./inventory.html` and `./lib/chart.umd.min.js` to `ASSETS` |
| `users.html` | Add `{slug:'inventory', name:'Inventory', icon:'📦'}` to `APPS`; add `inventory` key to `DEFAULT_PERMS` |
| `tests/` | New `inventory.spec.js` Playwright spec |

### Critical Pitfalls

1. **Chart.js ESM build instead of UMD** — use `chart.umd.min.js` explicitly (not `chart.min.js`); verify `typeof Chart !== 'undefined'` after script load; blank canvas with no error is the tell-tale symptom

2. **CDN Chart.js not cached by service worker** — download `chart.umd.min.js` to `/lib/`, add to `ASSETS` in `sw.js`, test with DevTools Offline mode; CDN opaque responses can silently fail `cache.addAll()` and discard the entire precache

3. **Chart instance not destroyed before re-render** — keep a module-level `let spendChart = null` per canvas; always call `chartRef.destroy()` before `new Chart()`; never initialize a chart while its canvas container is `display:none` (produces a 0×0 chart)

4. **Date grouping UTC shift** — `new Date("2026-04-14")` parses as midnight UTC, which is the previous calendar day in US timezones; use `parseLocalDate(str)` that splits `YYYY-MM-DD` and constructs with `new Date(y, m-1, d)`; use composite `YYYY-MM` keys for monthly grouping (not bare month numbers)

5. **Insufficient mock data** — minimum 12 purchase events, 2 calendar months, 3-4 vendors, 4-5 tags, 8-10 distinct items with varying purchase frequency; if all items resolve to the same stock level the algorithm is not exercised; sparse mock hides UX design flaws that surface with real data

## Implications for Roadmap

The build order is determined by three constraints: (1) mock data must exist before any rendering, (2) Chart.js integration risks must be resolved before feature logic is layered on top, and (3) the History tab validates the base data join chain before the derived Trends tab is built.

### Phase 1: Infrastructure and Data Foundation

**Rationale:** All downstream rendering depends on correct Chart.js integration and a well-structured mock dataset. All six Chart.js pitfalls (ESM/UMD, canvas sizing, instance lifecycle, offline caching, UTC dates, sparse data) surface in Phase 1 if deliberately tested — they cannot be retrofitted without rework.

**Delivers:** `inventory.html` page shell with boilerplate, 3 empty tab sections, Chart.js loaded locally and verified (`typeof Chart !== 'undefined'`), SW updated with bumped cache version and new ASSETS, `index.html` tile activated, `parseLocalDate` utility defined, all six mock data constants defined and inspectable in browser console.

**Addresses:** HQ integration (tile, SW, users permissions), full mock data layer with realistic food truck data spanning 2+ calendar months

**Avoids:** Pitfall 1 (ESM/UMD), Pitfall 2 (canvas sizing — establish container div pattern once), Pitfall 4 (UTC date shift — define `parseLocalDate` before any grouping), Pitfall 5 (CDN offline), Pitfall 6 (sparse mock), Pitfall 8 (year-boundary group key — use `YYYY-MM` composite keys)

**Playwright tests:** Page loads from HQ tile, back link returns to HQ, 3 tabs present and clickable, SW registers, `chart.umd.min.js` loaded (check via `page.evaluate(() => typeof Chart)`)

### Phase 2: Purchase History Tab

**Rationale:** History is the simplest tab — pure data rendering, no computation, no charts. It validates the full data join chain (Event → Purchase → Vendor) before more derived views are built. Data model errors surface here in the simplest possible context.

**Delivers:** Sorted purchase event list (newest-first) with vendor name, date, total; tappable expand/collapse showing line items (name, qty, price, isCase badge); vendor filter chips at top.

**Addresses:** Purchase history list, line-item detail, vendor filter, total spend visibility, isCase annotation display

**Avoids:** localStorage anti-pattern (keep all data inside `inventory.html`); renders use event delegation, not per-element handlers

**Playwright tests:** History tab renders events sorted newest-first; tapping a row expands line items; vendor filter chips narrow the visible events

### Phase 3: Stock Estimation Tab

**Rationale:** Stock estimation is algorithmically self-contained — no chart dependency, no full tag chain needed. Building it before Trends validates the PurchaseItemGroup → Purchase lookup before the more complex tag traversal required by the Trends view.

**Delivers:** Low/Medium/High badge per PurchaseItemGroup using purchase-frequency heuristic; reorder suggestions section at top of tab (Low + Medium items only, sorted by urgency then category); days-since-last-purchase annotation for owner context.

**Addresses:** Stock level estimates, reorder suggestions list

**Avoids:** Pitfall 7 (undefined algorithm edge cases — define as a spec comment before writing code: items with zero purchases default to Low, per-item not per-category, 4-week estimation window); Pitfall 9 (suggestions sort order — Low before Medium, Proteins before Packaging)

**Playwright tests:** All item groups render with stock level badges; Low items appear in reorder suggestions; High items do not appear in suggestions; at least one item at each level (Low/Medium/High) is visible

### Phase 4: Spending Trends Tab

**Rationale:** Most visually complex tab; depends on the full tag traversal chain (Event → Purchase → PurchaseItemGroup → Tag) and Chart.js. Built last so all prior tabs are validated and stable. The Chart.js destroy/recreate pattern from Phase 1 is exercised in earnest here.

**Delivers:** Monthly spending trend chart (Chart.js line, time-series); category breakdown horizontal bar chart (Chart.js bar); both using CSS var colors resolved at render time for correct dark mode behavior.

**Addresses:** Category spending breakdown, spending trend chart, category-level cost visibility

**Avoids:** Pitfall 3 (chart instance not destroyed — call `destroy()` before every `new Chart()`); Pitfall 8 (YYYY-MM composite keys for monthly grouping); UX pitfall of wrong chart type (line for time-series, horizontal bar for category comparison — not the same chart)

**Playwright tests:** Trends tab renders; canvas elements exist with nonzero dimensions in both light and dark mode; switching away and back does not double data points or produce console warnings about canvas reuse

### Phase Ordering Rationale

- Phase 1 before everything: Chart.js pitfalls and mock data quality cannot be retrofitted; these are the foundation every other phase stands on
- Phase 2 before Phase 3 and 4: History validates the base data joins; if the mock data structure has errors they surface here without chart complexity in the way
- Phase 3 before Phase 4: Stock estimation is independent and simpler; builds confidence in the PurchaseItemGroup lookup before the more complex tag chain traversal required by Trends
- Phase 4 last: Most complex rendering and most external-dependency surface area; prior phases deliver a complete, useful tool even if Phase 4 is not yet complete

### Research Flags

All phases have well-documented patterns — no `/gsd:research-phase` is needed for any phase.

- **Phase 1:** Chart.js UMD CDN integration, SW caching, PWA boilerplate — all pitfalls documented with prevention steps in PITFALLS.md; standard patterns
- **Phase 2:** Pure data rendering — established patterns in existing HQ tool pages; no novel elements
- **Phase 3:** Stock heuristic algorithm — fully specified in FEATURES.md and ARCHITECTURE.md with edge cases enumerated
- **Phase 4:** Chart.js bar and line configuration — STACK.md provides exact code patterns including dark mode CSS var resolution

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Chart.js 4.5.1 UMD verified against official docs, jsDelivr directory, and GitHub releases; `Intl.DateTimeFormat` verified via MDN; date-fns CDN incompatibility verified via official discussion thread |
| Features | MEDIUM | Industry feature patterns verified across Toast, MarketMan, NetSuite, Deliverect; exact UX decisions are judgment calls specific to 1-5 person food truck context; no direct comparables at this scale |
| Architecture | HIGH | Derived entirely from direct codebase inspection of all existing tool pages; patterns confirmed by `workflows.html` v1.0 verification; no external sources required |
| Pitfalls | HIGH | Chart.js sizing and instance pitfalls verified via official docs and GitHub issues #5805 and #11592; UTC date shift verified via MDN spec; SW opaque response behavior verified via web.dev caching guide |

**Overall confidence:** HIGH

### Gaps to Address

- **ARCHITECTURE.md vs STACK.md conflict on charting approach:** ARCHITECTURE.md recommends CSS-only bars to avoid external dependencies; STACK.md recommends Chart.js. Resolution: use Chart.js served as a local asset — this satisfies both the feature requirement and the offline constraint. CSS-only bars remain an acceptable fallback if Chart.js introduces unexpected complexity in Phase 1.

- **`parDays` field not in existing Baserow schema:** The stock estimation heuristic requires each `PurchaseItemGroup` to have a `parDays` value (expected repurchase interval). This field is not in the current Baserow schema description. The mock can define it inline; the backend will need to accommodate it when real data is integrated. Flag as a backend schema decision for later.

- **Tab role access gating:** FEATURES.md recommends History (Tab 1) be crew-accessible and Tabs 2-3 be manager+. The current mock has no auth gate — this is intentional for the mock phase but must be documented as a known divergence from the target permission model.

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection: `index.html`, `sw.js`, `users.html`, `purchasing.html`, `onboarding.html`, `workflows.html`, `ptr.js`
- [Chart.js GitHub releases — v4.5.1, Oct 13 2024](https://github.com/chartjs/Chart.js/releases)
- [Chart.js Responsive Charts docs — container sizing requirements](https://www.chartjs.org/docs/latest/configuration/responsive.html)
- [Chart.js API docs — destroy() method](https://www.chartjs.org/docs/latest/developers/api.html)
- [Chart.js GitHub Issue #5805 — canvas grows indefinitely in %-width container](https://github.com/chartjs/Chart.js/issues/5805)
- [Chart.js GitHub Issue #11592 — ESM-only CDN version in vanilla JS](https://github.com/chartjs/Chart.js/issues/11592)
- [jsDelivr chart.js@4.5.1/dist/ — UMD build filename confirmed](https://cdn.jsdelivr.net/npm/chart.js@4.5.1/dist/)
- [MDN: Intl.DateTimeFormat — iOS Safari 10+ compatibility](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat)
- [web.dev: Caching — CDN opaque responses and cache.addAll() failure](https://web.dev/learn/pwa/caching)
- [MDN: Service Worker caching strategies](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Caching)

### Secondary (MEDIUM confidence)
- [Toast — Food Truck Inventory Management](https://pos.toasttab.com/blog/on-the-line/food-truck-inventory-management) — industry-standard feature set
- [NetSuite — Restaurant Inventory Management Guide](https://www.netsuite.com/portal/resource/articles/inventory-management/restaurant-inventory-management.shtml) — stock level patterns and spend tracking
- [Deliverect — Stock Management for Food Service](https://www.deliverect.com/en/blog/fulfilment/importance-of-effective-stock-management-for-food-service-success) — food service-specific patterns
- [AppMaster — Inventory Reorder Suggestions Min/Max](https://appmaster.io/blog/inventory-reorder-suggestions-min-max-app) — reorder UX patterns
- [Highcharts — Line vs Bar Chart](https://www.highcharts.com/blog/best-practices/line-chart-vs-bar-chart-choosing-the-right-one-for-your-objectives-and-data/) — chart type selection for spending trends

### Tertiary (LOW confidence)
- [date-fns CDN discussion — v2+ no UMD](https://github.com/orgs/date-fns/discussions/2193) — used only to rule out date-fns; GitHub discussion, not official docs
- [DEV: JavaScript Date Time Zone Gotcha](https://dev.to/davo_man/the-javascript-date-time-zone-gotcha-that-trips-up-everyone-20lf) — UTC shift background reading

---
*Research completed: 2026-04-14*
*Ready for roadmap: yes*
