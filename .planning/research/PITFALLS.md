# Pitfalls Research

**Domain:** Inventory tracking, spending charts, and stock estimation — added to existing vanilla JS PWA (no build step, no framework)
**Researched:** 2026-04-14
**Confidence:** HIGH (charting pitfalls verified via official Chart.js docs + GitHub issues; date pitfalls verified via MDN + community; SW/CDN pitfalls verified via web.dev + MDN)

---

## Critical Pitfalls

---

### Pitfall 1: Chart.js ESM Build Used Instead of UMD Build

**What goes wrong:**
Chart.js v4 ships its primary bundle as an ES module (`chart.js`, `helpers.js`). When you load the wrong CDN path in a `<script>` tag, you get either a silent failure (the global `Chart` variable is never defined) or a CORS/module import error. The app appears to load but the canvas stays blank with no visible error unless the console is open.

**Why it happens:**
The CDN directory listing for Chart.js v4 shows many files. Developers pick `chart.min.js` expecting it to be the browser build, but that's the CJS/ESM entry. The UMD build is specifically named `chart.umd.min.js` — a different file.

**How to avoid:**
Use the explicit UMD path:
```html
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.5.1/dist/chart.umd.min.js"></script>
```
Pin to a specific version (not `@latest`) so the app does not break when Chart.js releases a new major. After loading, `Chart` must be defined as a global — verify with a one-line console check before wiring up any canvas.

**Warning signs:**
- Canvas element exists in the DOM but stays blank
- `Chart is not defined` error in console
- Using `<script src=".../chart.min.js">` (CJS) instead of `chart.umd.min.js`
- Using `<script type="module">` in an otherwise non-module codebase

**Phase to address:** Phase 1 (chart infrastructure setup) — validate the CDN script loads correctly before writing any chart logic.

---

### Pitfall 2: Canvas Sizing Breaks on 480px Mobile Viewport

**What goes wrong:**
Chart.js reads canvas dimensions from its parent container, not from CSS applied directly to the canvas. Common mistakes: setting `width`/`height` attributes on the `<canvas>` element using viewport units (`80vw`, `40vh`), or applying `margin: auto` directly to the canvas. The result is a chart that is either the wrong size, blurry (render resolution mismatched from display size), or that continuously shrinks as Chart.js repeatedly recalculates its container.

A documented bug in Chart.js: if the container uses `%`-based width without explicit overflow control, the chart canvas grows indefinitely on resize.

**Why it happens:**
Developers apply CSS sizing to the `<canvas>` tag directly (natural instinct), but Chart.js sizes the canvas from the parent element's dimensions. CSS on the canvas and Chart.js's internal sizing fight each other.

**How to avoid:**
Always wrap each canvas in a dedicated container div and size the container, not the canvas:
```html
<div style="position:relative;width:100%;max-width:480px;height:220px">
  <canvas id="spend-chart"></canvas>
</div>
```
Set `maintainAspectRatio: false` in Chart.js options when you need to control height independently of width. Never apply `margin`, `width`, or `height` styles directly to the `<canvas>` element.

For the 480px max-width constraint: set the container to `width: 100%` (not a fixed pixel value) so it inherits the `.app` container's max-width naturally.

**Warning signs:**
- Chart renders correctly in Chrome DevTools device emulation but is wrong size on a real phone
- Chart grows slightly larger on each window resize event
- Blurry/pixelated chart lines on Retina/high-DPI screens

**Phase to address:** Phase 1 (chart infrastructure) — establish the container pattern once and reuse it for all charts.

---

### Pitfall 3: Chart Instance Not Destroyed Before Re-Render (Memory Leak + Visual Glitch)

**What goes wrong:**
The app uses the `show(n)` tab-switching pattern from `purchasing.html` and `workflows.html` — tabs toggle `display:none`/`display:block`. When the trends tab is shown again after being hidden, code re-runs to build the chart. If the previous Chart.js instance was not destroyed, two chart instances share the same canvas context. The result: tooltip jitter (both instances respond to hover), double-rendered data points, and a growing memory leak with each tab switch.

**Why it happens:**
Chart.js attaches event listeners and stores a reference to the canvas context internally. When a new `new Chart(ctx, config)` call is made on a canvas that already has a Chart.js instance, Chart.js v4 throws a warning and exhibits undefined behavior — sometimes silently rendering both, sometimes only rendering one.

**How to avoid:**
Keep a module-level reference to each chart instance. Destroy before recreating:
```js
let spendChart = null;

function renderSpendChart(data) {
  if (spendChart) { spendChart.destroy(); spendChart = null; }
  const ctx = document.getElementById('spend-chart');
  spendChart = new Chart(ctx, { ... });
}
```
Call `renderSpendChart` from the tab-show logic, not from page init. Do not call `new Chart()` at page load if the tab starts hidden — a hidden canvas has zero dimensions and Chart.js will render with incorrect sizes.

**Warning signs:**
- Chart tooltips flicker or show duplicate values on hover
- `Canvas is already in use. Chart with ID X must be destroyed before the canvas with ID Y can be reused` warning in console
- Memory usage grows each time the Trends tab is visited

**Phase to address:** Phase 1 (chart rendering) — establish destroy/recreate pattern in the first chart implementation.

---

### Pitfall 4: Date Grouping Shifts by One Day Due to UTC Parsing

**What goes wrong:**
Purchase history records have date strings like `"2026-04-14"`. When you do `new Date("2026-04-14")`, JavaScript parses it as midnight UTC — not midnight local time. On devices in UTC-4 (Eastern) or UTC-5 (Central), this date becomes April 13 at 8pm or 7pm local time. Weekly and monthly grouping logic that calls `.getMonth()` or `.getDay()` on the result then assigns the event to the wrong week or month.

For a food truck in the US, purchase history grouped by "week of April 14" will silently show as "week of April 13" for all US timezones. Monthly spend totals will be off by one day at month boundaries.

**Why it happens:**
ISO 8601 date-only strings (`YYYY-MM-DD`) are parsed as UTC per spec. ISO date-time strings with a time component are parsed as local time. This asymmetry is a well-documented JavaScript footgun.

**How to avoid:**
Parse date-only strings by splitting manually rather than using `new Date()`:
```js
function parseLocalDate(str) {
  const [y, m, d] = str.split('-').map(Number);
  return new Date(y, m - 1, d); // local midnight, no UTC shift
}
```
Never pass `YYYY-MM-DD` strings directly to `new Date()` in grouping or comparison logic. Use this parser everywhere in the inventory module.

**Warning signs:**
- Weekly spending totals are off by one event at the week boundary
- Events dated on the 1st of the month appearing in the previous month's totals
- Any code with `new Date(dateString)` where `dateString` is `YYYY-MM-DD` format

**Phase to address:** Phase 1 (mock data + date utilities) — define `parseLocalDate` as a shared utility before any grouping logic is written.

---

### Pitfall 5: CDN Script Not Cached by Service Worker — Blank Charts Offline

**What goes wrong:**
The existing service worker uses a cache-first strategy for a hardcoded `ASSETS` list. Chart.js is loaded from a CDN (`cdn.jsdelivr.net`). The CDN URL is not in the precache list. When the device is offline (the PWA use case), the Chart.js script fails to load, `Chart` is undefined, and the Inventory tab shows blank canvases with no error message to the user.

Cross-origin `cache.addAll()` calls also fail silently if the CDN returns a non-2xx response — because CDN responses are "opaque," failures in one URL can cause the entire precache to be discarded.

**Why it happens:**
The current SW pattern works fine for same-origin assets. Adding a CDN dependency breaks the offline assumption without any visible warning — everything works on WiFi and breaks offline.

**How to avoid:**
Two options — pick one:
1. **Download Chart.js and serve it as a local asset.** Copy `chart.umd.min.js` into the repo (e.g., `/lib/chart.umd.min.js`), add it to `ASSETS` in `sw.js`, reference it with a relative path. This is the correct approach for this project given the no-build-step constraint and offline requirement.
2. If CDN is kept: add Chart.js to a runtime cache with a stale-while-revalidate strategy and explicitly handle the offline fallback case in JS (check `typeof Chart !== 'undefined'` before rendering, show a "Charts unavailable offline" message otherwise).

Option 1 is strongly preferred. It also eliminates the CDN availability risk and removes a third-party dependency from the critical path.

**Warning signs:**
- Chart.js loaded via CDN `<script>` tag but not added to `ASSETS` in `sw.js`
- Inventory page tested only with WiFi, never with DevTools "Offline" mode enabled
- No fallback UI for when the charting library fails to load

**Phase to address:** Phase 1 (chart integration) — decide local-vs-CDN at the start, add to SW cache immediately.

---

### Pitfall 6: Insufficient Mock Data Volume — Trends Look Meaningless

**What goes wrong:**
Three to four purchase history entries are created for the mock. The spending trends chart shows two or three data points on a line chart that looks like a straight line or a V-shape. The inventory estimates show one item as "low" and everything else as "high." The feature looks done but conveys no information — and the design decisions made against sparse data (color choices, label density, chart type) will not hold up when real data arrives.

**Why it happens:**
Developers add just enough data to render the chart without error. Sparse mock data does not stress the layout, the axis label formatting, the category color assignment, or the grouping logic.

**How to avoid:**
The minimum viable mock dataset for this feature:
- At least 12 purchase events (one per week for 3 months)
- At least 3-4 vendors appearing with varying frequency
- At least 4-5 category tags (Proteins, Produce, Supplies, Dairy, Packaging)
- At least 8-10 distinct line items, each appearing in multiple events
- Data that spans two calendar months so the monthly grouping logic is exercised
- At least one item with declining frequency (to show "Low" stock estimate) and one with consistent weekly purchases (to show "High")

Structure the mock as an array of `PurchaseEvent` objects, each with a `date`, `vendor`, and `lineItems` array. Write the grouping logic against this structure — the schema you choose for mock data will directly inform the backend data model.

**Warning signs:**
- Trends chart with fewer than 6 data points
- All inventory items showing the same stock level ("High")
- A chart that looks the same regardless of which category filter is applied
- Date range that does not cross a month boundary

**Phase to address:** Phase 1 (mock data design) — define the full dataset before writing any rendering code.

---

## Moderate Pitfalls

---

### Pitfall 7: Stock Estimation Algorithm With No Defined Edge Cases

**What goes wrong:**
The stock estimation (Low/Medium/High) is derived from purchase frequency. But "frequency" is undefined: frequency compared to what? If an item was bought 3 times in 12 weeks, is that low or high? If an item was not bought at all in the mock data, does it appear as "Low" or is it omitted? If two items have the same category tag, do they share an estimate or get individual estimates?

Without defined edge cases, the algorithm produces inconsistent outputs — some items appear to never go Low, others flicker between states as the mock data is adjusted.

**How to avoid:**
Define the algorithm explicitly before writing code:
- **Frequency baseline:** Compare actual purchase count to expected purchase count. If expected weekly (`par` × number of weeks), an item bought less than 50% of expected = Low, 50-80% = Medium, 80%+ = High.
- **Items with zero purchases:** Always Low. Always shown in the suggestions list.
- **Per-item, not per-category:** Estimates apply to `PurchaseItemGroup` (e.g., "Salmon fillet"), not to the category tag ("Proteins"). Multiple items in Proteins can have different levels.
- **Estimate period:** Fixed window (last 4 weeks) not all-time, so the estimate reflects current behavior.

Document this algorithm in a code comment before implementation. The UX copy ("Low stock — consider ordering") depends on these rules being stable.

**Warning signs:**
- Stock level derivation logic spread across multiple functions with no single source of truth
- `par` value from the purchasing mock not being reused as the frequency baseline
- Mock data designed to make every item show "Medium" to avoid deciding the edge cases

**Phase to address:** Phase 2 (inventory estimates) — define algorithm as a spec comment before writing the estimation function.

---

### Pitfall 8: Weekly vs. Monthly Grouping Keys Collide Across Years

**What goes wrong:**
Spending trend data is grouped by week or month. A common shortcut: use `"${month}-${year}"` or just `month` as the grouping key. If mock data spans a year boundary (December to January), `month = 12` and `month = 1` are not adjacent without the year qualifier. If the key is just the month number, January data from two different years merges into one bucket.

**Why it happens:**
Mock data that stays within one calendar year never surfaces this bug. It only appears when the date range crosses January.

**How to avoid:**
Always use composite keys that include the year:
- Monthly: `"${year}-${String(month).padStart(2, '0')}"` — produces `"2026-01"`, `"2026-02"`, which sort correctly as strings
- Weekly: Use ISO week number with year: derive as `"${year}-W${weekNumber}"` using a manual ISO week calculation (do not use `toLocaleDateString('en', {week: 'narrow'})` — output format varies by locale)

Sort grouped keys alphabetically after grouping — this produces correct chronological order for YYYY-MM keys without needing a date sort.

**Warning signs:**
- Group key is a bare month number (`1`, `2`, ..., `12`)
- Mock data that never crosses a year boundary (artificially safe)
- Trend chart with a non-monotonic time axis (data points out of order)

**Phase to address:** Phase 1 (date grouping utilities) — write and test grouping keys before building chart data pipelines.

---

### Pitfall 9: Reorder Suggestions List Rendered for All Low Items Without a Useful Order

**What goes wrong:**
The reorder suggestions feature flags Low and Medium items for the next PO. If the list is rendered in insertion order from the mock data array, items appear in an arbitrary sequence. For a food truck, the owner needs to see the most critical items (out of stock, daily-use proteins) at the top, not in data entry order.

Additionally, if Low and Medium items are mixed without visual distinction, the urgency signal is lost — the owner cannot quickly scan for "must-order today" vs. "order when convenient."

**How to avoid:**
Sort the suggestions list: Low stock before Medium, then by category (Proteins first — highest daily use and perishability), then alphabetically within category. Use a fixed category priority array to define sort order rather than inferring it. Display Low and Medium with distinct visual treatments (different pill colors, not just label text).

**Warning signs:**
- Suggestions list rendered with `ITEMS.filter(i => i.level !== 'High')` without any subsequent sort
- Low and Medium items shown with the same visual treatment, differentiated only by label text
- Proteins appearing below Packaging in the suggestions list

**Phase to address:** Phase 2 (reorder suggestions UI) — define sort order and visual hierarchy before writing the render function.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Hardcoded category tags in the estimation function | Avoids a config object | Adding a new category requires touching the algorithm, not just the mock data | MVP only — extract to a `CATEGORIES` constant in the same phase |
| Using `new Date(dateStr)` directly | One less utility function | UTC-shift bugs in all US timezones for date-only strings | Never — use `parseLocalDate()` |
| Loading Chart.js from CDN | No file to commit | Charts blank offline; CDN outage or version float breaks the app | Never for this PWA — copy the file locally |
| Rendering all purchase history rows on page load | Simple code | History with 200+ events causes a slow paint on low-end phones | Acceptable for mock phase; add pagination before real data |
| Single chart instance per canvas at module level | Avoids destroy/recreate complexity | Multiple tabs with charts all share one variable — wrong chart gets destroyed | Never — use one variable per canvas, named clearly |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Chart.js + `show(n)` tab pattern | Initializing chart at page load while the canvas is `display:none` — produces a 0-width chart | Initialize chart inside the `show()` call for the tab that contains the canvas, after setting `display:block` |
| Chart.js + dark mode CSS vars | Using CSS `var(--txt)` directly in Chart.js config — JS cannot resolve CSS custom properties | Read the computed color: `getComputedStyle(document.documentElement).getPropertyValue('--txt').trim()` and pass the resolved hex value to Chart.js |
| Chart.js + service worker | CDN script not in SW precache list | Copy `chart.umd.min.js` locally, add to `ASSETS` array in `sw.js`, bump cache version |
| Mock date strings + grouping | Parsing `"2026-04-01"` with `new Date()` applies UTC shift | Use the `parseLocalDate(str)` split-parse utility for all date-only strings |
| Spending trends + category filter | Re-creating the chart dataset on every filter tap without destroying the previous instance | Call `chart.destroy()` before `new Chart()`, or update `chart.data` in-place and call `chart.update()` |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Rendering all purchase history rows into the DOM at once | Page load pause; scroll jank on low-end Android | Render only the most recent 20 events; add "load more" button | At ~100 events in the mock array |
| Recalculating stock estimates on every render pass | UI stutter when switching between inventory items | Calculate estimates once on page load, cache in a derived object, re-render from cache | Negligible for mock scale; matters when real data arrives |
| Chart.js on a hidden canvas (display:none) | Chart renders at 0×0 or throws a sizing error | Always show the canvas container before calling `new Chart()` | Every tab switch if initialization order is wrong |
| Large inline JS data constant parsed at page load | 50-100ms parse delay on low-end phones | Keep mock data to a reasonable volume (12-24 events); structure as compact objects | At ~500 purchase event objects in the inline constant |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Bar chart for spending trends instead of line chart | Harder to see trend direction over time; bars work better for category comparison | Use a line chart for time-series spending; use a horizontal bar chart for category breakdown (two different chart types, two different views) |
| Chart legend with full category names on 480px width | Legend text wraps or overflows, obscuring the chart | Use colored dots/pills below the chart instead of Chart.js's built-in legend; or limit to 4 categories max with short labels |
| Stock level badges using only color (red = Low, yellow = Medium) | Invisible for color-blind users | Pair color with a text label: "LOW", "MED", "HIGH" — never use color as the only signal |
| Showing all purchase history in a flat list sorted by date descending | Most recent purchase visible; vendor/category context requires scanning | Group by week with a week header, showing vendor and total per week — matches mental model of "what did I buy this week" |
| Reorder suggestions with no action affordance | User sees what to order but cannot act on it from this screen | Add a "Start PO" button that links back to purchasing.html — even if it's just a link, it closes the loop |

---

## "Looks Done But Isn't" Checklist

- [ ] **Chart dark mode:** Chart renders with correct colors in dark mode — verify on a device with system dark mode enabled, not just in browser with DevTools dark mode simulation
- [ ] **Chart offline:** Inventory page loads and charts render with device in airplane mode — confirms Chart.js is in SW cache
- [ ] **Date grouping:** Weekly totals are correct for events at the week boundary (Sunday/Monday) — check with mock data that has events on both sides of a week boundary
- [ ] **Stock level logic:** At least one item shows "Low" and one shows "High" in the mock — if all items are the same level, the algorithm is not exercised
- [ ] **Reorder suggestions:** Items at "Low" stock actually appear in the suggestions list — verify the filter is checking the correct field name
- [ ] **Tab switch cleanup:** Switch to Trends tab, switch away, switch back — chart should render correctly on the second visit without doubling data points
- [ ] **SW cache bump:** `sw.js` cache version is incremented and `inventory.html` (and any local JS/CSS files) are in the `ASSETS` array before the first deploy

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| ESM build loaded instead of UMD | LOW | Swap the CDN script URL to the `chart.umd.min.js` path; test; done |
| Canvas sizing broken on mobile | MEDIUM | Wrap all canvases in container divs, add `position:relative` and explicit height, set `maintainAspectRatio:false`; test on real device |
| Chart instance leak discovered late | MEDIUM | Audit every `new Chart()` call; add `if (chartRef) chartRef.destroy()` before each; extract to a `createChart(id, config)` helper that handles cleanup |
| Date grouping UTC shift discovered after mock review | MEDIUM | Replace all `new Date(str)` calls with `parseLocalDate(str)`; recheck all grouping outputs; update mock data if boundary events shifted |
| CDN Chart.js not cached by SW (discovered at demo time) | LOW | Download `chart.umd.min.js`, commit as `/lib/chart.umd.min.js`, update script src, add to `ASSETS`, bump SW cache version |
| Mock data too sparse (discovered during design review) | LOW | Add more `PurchaseEvent` objects to the inline array — no structural changes required if data schema was defined correctly upfront |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| ESM vs UMD CDN build (Pitfall 1) | Phase 1: Chart infrastructure | `typeof Chart !== 'undefined'` check after script load; visible chart renders |
| Canvas sizing on mobile (Pitfall 2) | Phase 1: Chart infrastructure | Test on 375px-wide real device or accurate emulator |
| Chart instance not destroyed (Pitfall 3) | Phase 1: Chart rendering | Switch Trends tab 3x; verify no console warnings, no doubling |
| UTC date shift in grouping (Pitfall 4) | Phase 1: Mock data + date utilities | Unit-verify `parseLocalDate` output equals midnight local time |
| CDN not in SW cache (Pitfall 5) | Phase 1: Service worker integration | Load page in DevTools Offline mode; charts must still render |
| Insufficient mock data (Pitfall 6) | Phase 1: Mock data design | Chart must show at least 8 data points spanning 2 calendar months |
| Stock algorithm edge cases (Pitfall 7) | Phase 2: Inventory estimates | At least one item at each level (Low/Medium/High) visible in mock |
| Year-boundary group key collision (Pitfall 8) | Phase 1: Date grouping utilities | Verify composite `YYYY-MM` key format is used consistently |
| Suggestions list sort order (Pitfall 9) | Phase 2: Reorder suggestions UI | Proteins appear before Packaging; Low before Medium in the list |

---

## Sources

- [Chart.js Responsive Charts docs — container sizing requirements](https://www.chartjs.org/docs/latest/configuration/responsive.html)
- [Chart.js API docs — destroy() method](https://www.chartjs.org/docs/latest/developers/api.html)
- [Chart.js GitHub Issue #5805 — canvas grows indefinitely in %-width container](https://github.com/chartjs/Chart.js/issues/5805)
- [Chart.js GitHub Issue #11592 — ESM-only CDN version cannot be imported in vanilla JS](https://github.com/chartjs/Chart.js/issues/11592)
- [Chart.js jsDelivr dist directory — UMD build filename confirmed as chart.umd.min.js, v4.5.1](https://cdn.jsdelivr.net/npm/chart.js@latest/dist/)
- [DEV: The JavaScript Date Time Zone Gotcha That Trips Up Everyone](https://dev.to/davo_man/the-javascript-date-time-zone-gotcha-that-trips-up-everyone-20lf)
- [MDN: Intl.DateTimeFormat — correct date formatting and timezone handling](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat)
- [web.dev: Caching — CDN opaque responses and cache.addAll() failure behavior](https://web.dev/learn/pwa/caching)
- [MDN: Service Worker caching strategies](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Caching)
- [MagicBell: Offline-First PWAs — CDN caching pitfalls](https://www.magicbell.com/blog/offline-first-pwas-service-worker-caching-strategies)

---
*Pitfalls research for: inventory tracking + spending charts + stock estimation added to vanilla JS PWA*
*Researched: 2026-04-14*
