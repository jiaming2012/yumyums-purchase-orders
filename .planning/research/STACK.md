# Stack Research

**Domain:** Mobile-first vanilla JS PWA — Inventory / Spending Trends / Stock Estimation
**Researched:** 2026-04-14
**Scope:** NEW additions only for v1.1 Inventory milestone. Existing stack (vanilla JS, CSS custom properties, SortableJS, Playwright) is validated and not re-researched here.
**Confidence:** HIGH for Chart.js; HIGH for date handling; HIGH for stock estimation approach

---

## What This Milestone Needs (and What It Does Not)

The v1.1 inventory app requires:

1. **Charting** — spend-over-time line charts and category breakdown bar charts (read-only, not interactive drill-down)
2. **Date handling** — grouping purchase events by week/month, formatting date labels on chart axes
3. **Stock estimation logic** — classify items as low/medium/high based on purchase frequency relative to a configurable par level (pure computation, no library needed)

It does NOT need:
- A new drag-and-drop library (inventory view is read-only)
- Any new form handling (no builder in this milestone)
- Real-time or WebSocket updates (all data is hardcoded mock arrays)
- A new router (this is a standalone `inventory.html` page like all other tools)

---

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Vanilla JS (ES2020+) | native | Data computation, rendering, state | Existing convention; no exception for this milestone |
| CSS custom properties | native | Dark mode, chart theming | `getComputedStyle(root).getPropertyValue('--text')` pattern lets Chart.js read existing CSS variables at render time — no separate theme config needed |
| Chart.js | **4.5.1** | Bar and line charts | Only library that is (a) CDN-loadable via UMD bundle, (b) genuinely mobile-responsive via canvas auto-resize, (c) has a trivial vanilla JS API (`new Chart(ctx, config)`), and (d) weighs ~60KB min+gzip — acceptable for a single-page tool. v4.x is the current major; last release Oct 2024. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| None (Intl.DateTimeFormat) | native browser API | Date label formatting | `Intl.DateTimeFormat` covers all formatting needed: `"Apr 2026"`, `"Week of Apr 7"`, month grouping. No external library required — ships in every modern browser including iOS Safari 10+. |
| None (custom grouping logic) | hand-rolled | Weekly/monthly spend aggregation | Group an array of `PurchaseEvent` objects by month key (`YYYY-MM`) using `reduce()`. ~15 lines of JS. No library improves on this for the use case. |
| None (par-level heuristic) | hand-rolled | Low/medium/high stock classification | Compare days-since-last-purchase against per-item par level thresholds. Pure JS arithmetic. See Stock Estimation Logic section below. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| Playwright (existing) | E2E regression tests | Add `inventory.spec.js` following the pattern in `tests/workflows.spec.js`. Test chart canvas presence by asserting `canvas` element exists, not chart internals. |
| python3 -m http.server | Local dev server | Unchanged from existing workflow |

---

## Installation

No npm packages required for this milestone. Chart.js loads from CDN.

```html
<!-- Add to inventory.html, before closing </body> -->
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.5.1/dist/chart.umd.min.js"></script>
```

**Pin to 4.5.1 explicitly.** Do not use `@latest` — floating tags break after cache busts and make debugging version mismatches harder. 4.5.1 is the current release as of Oct 2024.

The UMD bundle exposes `Chart` as a global, usable directly in a `<script>` block without `import`.

---

## Integration Points

### Chart.js + CSS Variables (Dark Mode)

Chart.js uses canvas, which does not inherit CSS variables automatically. Read variables at render time:

```javascript
const root = document.documentElement;
const style = getComputedStyle(root);
const textColor = style.getPropertyValue('--text').trim();
const mutedColor = style.getPropertyValue('--mut').trim();
const cardColor = style.getPropertyValue('--card').trim();

const chart = new Chart(ctx, {
  type: 'bar',
  data: { ... },
  options: {
    color: textColor,
    plugins: {
      legend: { labels: { color: textColor } }
    },
    scales: {
      x: { ticks: { color: mutedColor }, grid: { color: cardColor } },
      y: { ticks: { color: mutedColor }, grid: { color: cardColor } }
    }
  }
});
```

This pattern works with the existing `prefers-color-scheme` dark mode setup because CSS variables are already resolved to the correct dark/light value by the time JS runs.

If the user switches color scheme mid-session (rare on mobile), destroy and recreate the chart. Chart.js v4 exposes `chart.destroy()` for cleanup.

### Chart.js + Mobile Responsiveness

Chart.js resizes to fill its container by default (`responsive: true`). Wrap each canvas in a fixed-height container div to prevent vertical stretching on narrow screens:

```html
<div style="position:relative;height:220px">
  <canvas id="spend-chart"></canvas>
</div>
```

220px is the recommended height for a phone-width chart (fits well at 480px max-width without scrolling past the fold).

### Stock Estimation Logic (No Library)

Classification is a simple time-since-last-purchase heuristic:

```javascript
// Returns 'low' | 'medium' | 'high'
function stockLevel(item) {
  const daysSinceLastPurchase = (Date.now() - item.lastPurchasedAt) / 86400000;
  if (daysSinceLastPurchase > item.parDays * 1.5) return 'low';
  if (daysSinceLastPurchase > item.parDays * 0.75) return 'medium';
  return 'high';
}
```

Where `parDays` is the expected repurchase interval stored on each `PurchaseItemGroup` in the mock data. This requires no library — only `Date.now()` and arithmetic.

### Date Grouping (No Library)

Group `PurchaseEvent[]` by month for trend charts:

```javascript
// Returns { 'Jan 2026': 420.50, 'Feb 2026': 310.00, ... }
function groupByMonth(events) {
  return events.reduce((acc, e) => {
    const label = new Intl.DateTimeFormat('en-US', { month: 'short', year: 'numeric' })
      .format(new Date(e.date));
    acc[label] = (acc[label] || 0) + e.totalCost;
    return acc;
  }, {});
}
```

`Intl.DateTimeFormat` is available in all iOS Safari versions since 10 (2016) and all Android Chrome. No polyfill needed.

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Chart.js 4.5.1 via CDN UMD | ApexCharts | ApexCharts has better SVG quality and annotation support — choose it if interactive drill-down, zoom, or click-to-filter are needed. For read-only summary charts, Chart.js is simpler to integrate with no build step. |
| Chart.js 4.5.1 via CDN UMD | Recharts / Victory | Both require React — not compatible with this codebase. Eliminated immediately. |
| Chart.js 4.5.1 via CDN UMD | D3.js | D3 is a low-level toolkit, not a chart library. Appropriate when you need custom chart geometries. For bar + line charts, Chart.js requires 80% less code. |
| Chart.js 4.5.1 via CDN UMD | ECharts (Apache) | ECharts has a larger bundle (~750KB min) and a more complex options API. Better for enterprise dashboards with many chart types. Overkill here. |
| Chart.js 4.5.1 via CDN UMD | Chartist.js | Chartist is effectively unmaintained (last release 2019). Do not use. |
| Native Intl.DateTimeFormat | date-fns | date-fns v2+ no longer ships a UMD bundle and requires a bundler or ES module setup. Not compatible with the no-build-step constraint. Intl covers all date formatting needed here. |
| Hand-rolled stock heuristic | Any inventory forecasting library | No inventory library exists as a CDN script. All are npm packages targeting Node.js or React apps. The heuristic is 10 lines of JS. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `chart.js@latest` CDN tag | Floating version; cache busting after a Chart.js release causes unexpected behavior in offline-cached PWA | Pin to `chart.js@4.5.1` |
| `temporalio/web` (old UI image) | Unrelated, but mentioned in project CLAUDE.md — do not confuse with UI library | N/A |
| Recharts, Victory, Nivo | All React-dependent, require npm + bundler | Chart.js UMD |
| date-fns v2+ via CDN | No UMD bundle ships in v2+; ESM import requires `type="module"` which conflicts with existing inline script pattern | `Intl.DateTimeFormat` (native) |
| Chartist.js | Last release 2019; unmaintained | Chart.js 4.5.1 |
| D3.js for basic charts | 80KB+ for a problem Chart.js solves in 5 lines | Chart.js 4.5.1 |
| ECharts | ~750KB bundle; complex API for simple bar/line use case | Chart.js 4.5.1 |

---

## Stack Patterns by Variant

**If only category breakdown (no time-series) is needed:**
- Use `type: 'bar'` with horizontal orientation — better for long category names (Beef, Produce, etc.) on narrow mobile screens
- `indexAxis: 'y'` in Chart.js 4.x flips to horizontal

**If time-series spending trends are needed (current milestone):**
- Use `type: 'line'` with `tension: 0.3` for a smooth curve
- Group data by month using the `groupByMonth()` helper above
- Limit to last 6 months to keep the chart readable at 480px width

**If multiple categories on one chart are needed:**
- Use `type: 'bar'` with `datasets` array, one dataset per category
- Use `stacked: true` on both axes for a stacked bar view — easier to read total spend at a glance

**If dark mode chart theming is unreliable:**
- Listen for `prefers-color-scheme` media query change and call `chart.destroy()` then re-init
- This is a rare edge case; most mobile users don't switch mid-session

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| chart.js@4.5.1 | Modern browsers (Chrome 88+, Safari 14+, Firefox 85+) | Canvas API required; all installed PWA targets support this |
| chart.js@4.5.1 | iOS Safari 14+ | PWA install requires iOS 14.5+ anyway; no conflict |
| chart.js@4.5.1 | SortableJS 1.15.7 | No conflict; they operate on different DOM elements |
| Intl.DateTimeFormat | iOS Safari 10+ / Chrome 24+ | Universal; no polyfill needed |

---

## Sources

- [Chart.js GitHub releases — v4.5.1, Oct 13 2024](https://github.com/chartjs/Chart.js/releases)
- [Chart.js Installation docs — UMD bundle, CDN options](https://www.chartjs.org/docs/latest/getting-started/installation.html)
- [Chart.js Integration docs — vanilla JS script tag usage](https://www.chartjs.org/docs/latest/getting-started/integration.html)
- [jsDelivr chart.js@4.5.1/dist/ — UMD bundle confirmed](https://cdn.jsdelivr.net/npm/chart.js@4.5.1/dist/)
- [cdnjs Chart.js — 4.5.0 latest on cdnjs (jsDelivr has 4.5.1)](https://cdnjs.com/libraries/Chart.js/)
- [date-fns CDN discussion — v2+ no UMD, CDN difficult](https://github.com/orgs/date-fns/discussions/2193) — MEDIUM confidence (GitHub discussion, not official docs)
- [Chart.js dark mode discussion — CSS variables + canvas approach](https://github.com/chartjs/Chart.js/discussions/9214)
- [Intl.DateTimeFormat MDN compatibility — iOS Safari 10+](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/DateTimeFormat) — HIGH confidence

---

*Stack research for: Yumyums HQ v1.1 Inventory App — charting and stock estimation additions*
*Researched: 2026-04-14*
