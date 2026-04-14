# Feature Research: Inventory Tracking, Spending Trends, Stock Estimation

**Domain:** Food truck inventory / purchasing analytics — mobile-first PWA, mock data only
**Researched:** 2026-04-14
**Confidence:** MEDIUM — industry patterns verified via multiple sources; exact UX decisions are judgment calls for this specific small-team context

---

## Context: What This Milestone Is

v1.1 adds a standalone `inventory.html` tool to the existing HQ PWA. It surfaces Baserow purchasing data (already being captured) to answer three operator questions:

1. **What did we buy, when, and how much did we spend?** (purchase history)
2. **Where is our money going, by category and over time?** (spending trends)
3. **Are we running low on anything we buy regularly?** (stock estimation)

This is a read-only analytics view. No ordering, no barcode scanning, no real-time sync. All data is hardcoded mock JS arrays that reflect the Baserow schema.

---

## Data Model (from Baserow — drives all feature decisions)

| Entity | Fields | Role |
|--------|--------|------|
| `Vendor` | name | Supplier (e.g., Restaurant Depot, Sysco) |
| `PurchaseEvent` | date, vendor, total, tax | One shopping trip |
| `Purchase` | name, qty, price, isCase | Line item on a PurchaseEvent |
| `PurchaseItem` | — | Canonical item record |
| `PurchaseItemGroup` | name | Logical grouping (e.g., Brisket, Corn, Soap) |
| `Tag` | name | Category (Beef, Produce, Supplies, Cleaning, etc.) |

Tags categorize PurchaseItemGroups. One group can have multiple tags.

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features the owner will immediately look for. Missing these makes the tool feel like a stub.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Purchase history list | Any purchasing tool shows what was bought and when — this is the minimum viable view | LOW | List of PurchaseEvents sorted by date descending; tap to expand line items |
| Line-item detail per purchase event | "What did I buy on that trip?" is the first follow-up question after seeing a date/total | LOW | Expand/collapse card showing each Purchase row (name, qty, price, isCase flag) |
| Vendor filter on history | Owner shops at 2-3 vendors; filtering by vendor is instant orientation | LOW | Toggle chips at top; filter PurchaseEvents by vendor name |
| Total spend visible at a glance | Every purchasing summary shows total cost — operators scan for it | LOW | Show event total and per-item subtotal (qty × price) |
| Category spending breakdown | "How much did I spend on beef vs produce?" is the first analytics question for food cost management | MEDIUM | Group PurchaseEvents → Purchases → PurchaseItemGroups → Tags; sum spend per tag |
| Time-based spending view | Spending by week or month — standard in every inventory/spending tool reviewed | MEDIUM | Bar chart (monthly totals) or list grouped by month; select time range |
| Stock level indicators (low/medium/high) | Any inventory tool that tracks purchases infers stock status — operators expect a traffic-light view | MEDIUM | Derived from purchase frequency + days since last purchase per PurchaseItemGroup |
| Reorder flag / alert | Items that need restocking should be visually distinct — industry standard for any inventory view | LOW | Badge or color indicator on low-stock items; no action required (display only) |

---

### Differentiators (Competitive Advantage)

Features that add meaningful value beyond what a basic purchase log provides, and are appropriate for a 1-5 person food truck operation.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Category spending trend chart | Visualizes where food cost is going over time by tag — helps owner catch category drift (e.g., beef costs creeping up) | MEDIUM | Stacked bar or grouped bar chart, monthly buckets, colored by tag; Chart.js or pure SVG via Canvas API; no external dependency needed for simple bars |
| Spend-per-vendor summary | "Am I spending more at Restaurant Depot or Sysco?" — actionable for negotiation | LOW | Aggregate PurchaseEvent totals by vendor; simple sorted list with total |
| Average price tracking per item | Detects price increases across purchase events for the same item — early warning on supplier price changes | MEDIUM | Compare price per unit across PurchaseEvents for same Purchase name or PurchaseItemGroup; flag if latest > 30-day average |
| Reorder suggestions list | Pre-built list of items at low/medium stock for copy-paste into next PO — reduces cognitive load on order day | LOW-MEDIUM | Filter PurchaseItemGroups by low/medium stock level; display as a consolidated "Next PO" list; display only, no action |
| isCase annotation on line items | Shows whether an item was bought as a case (bulk) vs. unit — changes how stock estimation works | LOW | Display-only flag already in data model; drives stock estimation logic (case = higher unit count) |

---

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Real-time stock counting / physical counts | "Accurate inventory" sounds valuable | Requires crew to manually count every item on the truck — 1-5 person operation; adds daily overhead with no backend to store counts | Use purchase-frequency estimation (low/med/high) which requires zero crew effort |
| Barcode / QR scanning | Seems like it would speed up receiving | Requires device camera permission flow + barcode library + item database — massive complexity for mock stage; not in scope per PROJECT.md | Manual line-item entry already captured in Baserow |
| Predictive reorder quantities (EOQ / ML) | "Smart" suggestions are appealing | EOQ formula requires demand rate, holding cost, order cost — data the food truck doesn't have; ML needs 6+ months of history | Simple "days since last purchase" heuristic is sufficient and understandable |
| Push notifications for low stock | "Alert me when I'm low" is a natural request | Requires backend + push subscription infrastructure; out of scope for mock phase | Visual badge in app on next visit is sufficient |
| Recipe costing / food cost % | Connects inventory to menu pricing | Requires menu item → ingredient bill-of-materials mapping — a separate data model not in Baserow; significant scope expansion | Spending by category gives directional food cost insight without the BOM complexity |
| Waste tracking | Used by enterprise restaurant groups | Adds a daily manual data entry workflow on top of existing purchasing data entry — doubles operator burden | Not needed at 1-5 person scale; spoilage is managed informally |
| Supplier comparison / price benchmarking | "Am I getting good prices?" | Requires external price database or competitor quotes — no data source available | Average price tracking per item across the owner's own historical purchases is sufficient |
| Integration with purchasing.html ordering flow | "Connect inventory to ordering" sounds obvious | purchasing.html is a separate mockup with its own data model; connecting them requires shared state architecture — a bigger refactor than this milestone | Reorder suggestions list provides the bridge without coupling the implementations |

---

## Feature Dependencies

```
PurchaseEvent list (history)
    └──requires──> Purchase line items (detail)
                       └──requires──> PurchaseItemGroup mapping (for grouping)
                                          └──requires──> Tag data (for category breakdown)

Category spending breakdown
    └──requires──> PurchaseEvent + Purchase + PurchaseItemGroup + Tag (full chain)

Spending trend chart
    └──requires──> Category spending breakdown (chart is a time-series of the breakdown)

Stock level estimation (low/medium/high)
    └──requires──> PurchaseEvent dates per PurchaseItemGroup (frequency + recency)
    └──enhanced-by──> isCase flag (case purchase = higher quantity, slower depletion)

Reorder suggestions list
    └──requires──> Stock level estimation (only surfaces low/medium items)

Spend-per-vendor summary
    └──requires──> PurchaseEvent vendor field (simple aggregation, no chain needed)

Average price tracking
    └──requires──> Multiple PurchaseEvents with same item (needs 2+ events in mock data)
```

### Dependency Notes

- **Full tag chain is the critical dependency:** Category breakdown and trend charts both require walking PurchaseEvent → Purchase → PurchaseItemGroup → Tag. The mock data must represent this full chain correctly, or the most valuable features won't work.
- **Stock estimation is independent of the chart:** It operates on dates and frequency, not spend amounts. Can be built and tested separately.
- **Reorder suggestions are a view on top of stock estimation:** No new data logic needed — just filter the stock level output.
- **isCase enhances but does not block stock estimation:** A simpler version can ignore it; a more accurate version weights case purchases as higher quantity.

---

## Stock Estimation Logic (how "low/medium/high" works without physical counts)

This is the core algorithm question. Industry research confirms the right approach for a data-constrained food truck is **purchase-frequency + recency heuristic**, not EOQ or ML.

### Heuristic

For each PurchaseItemGroup:

1. **Purchase frequency:** How many days on average between purchases? (total span ÷ event count)
2. **Days since last purchase:** How many days since the most recent PurchaseEvent containing this group?
3. **Depletion ratio:** `days_since_last / average_purchase_interval`
   - 0.0–0.5 → **HIGH** (bought recently, likely well stocked)
   - 0.5–1.0 → **MEDIUM** (halfway through typical cycle)
   - 1.0+ → **LOW** (overdue for reorder by historical pattern)

### isCase adjustment

Items bought as cases deplete more slowly. A simple multiplier (e.g., 1.5×) on the interval when the last purchase was a case purchase is sufficient for mock data.

### Edge cases to handle in mock data

- **Only one purchase event:** Cannot calculate interval; default to MEDIUM
- **Irregular purchase items (seasonal or one-off):** Flag as MEDIUM; no clear cycle to track
- **Never purchased (in PurchaseItemGroup but no events):** Omit from stock view or show as N/A

---

## MVP Definition

### Launch With (v1.1 milestone)

Per PROJECT.md Active requirements — build all of these:

- [ ] **Purchase history view** — PurchaseEvents list sorted by date, expandable line items, vendor filter — why essential: answers "what did we buy?"
- [ ] **Category spending breakdown** — total spend per Tag across all time or a date range — why essential: answers "where is money going?"
- [ ] **Spending trend chart** — monthly spend per category as a bar chart — why essential: the "trends" requirement; visualizes drift over time
- [ ] **Stock level estimates** — low/medium/high badge per PurchaseItemGroup using frequency heuristic — why essential: answers "are we running low?"
- [ ] **Reorder suggestions** — filtered list of low/medium items — why essential: actionable output from stock estimation
- [ ] **HQ integration** — launcher tile, SW cache bump, Users permissions — why essential: the tool must be accessible from HQ

### Add After Validation (v1.x)

- [ ] **Average price tracking per item** — trigger: owner asks "is Sysco charging me more?" after using history view
- [ ] **Spend-per-vendor summary** — trigger: if owner has 3+ vendors and wants to compare
- [ ] **Date range filter on trends** — trigger: once 6+ months of mock data exists and comparisons are meaningful

### Future Consideration (v2+)

- [ ] **Backend integration with real Baserow data** — defer: mock phase must validate UX first
- [ ] **Recipe costing / BOM** — defer: requires separate menu data model
- [ ] **Push notifications for low stock** — defer: requires backend

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Purchase history list + line items | HIGH | LOW | P1 |
| Vendor filter on history | HIGH | LOW | P1 |
| Category spending breakdown | HIGH | MEDIUM | P1 |
| Stock level estimates (low/med/high) | HIGH | MEDIUM | P1 |
| Reorder suggestions list | HIGH | LOW (depends on stock estimates) | P1 |
| Spending trend chart | MEDIUM | MEDIUM | P1 |
| HQ integration (tile, SW, permissions) | HIGH | LOW | P1 |
| Spend-per-vendor summary | MEDIUM | LOW | P2 |
| Average price tracking | MEDIUM | MEDIUM | P2 |
| Date range filter | MEDIUM | MEDIUM | P2 |
| isCase-weighted stock estimation | LOW | LOW | P2 |

**Priority key:**
- P1: Must have for v1.1 launch (all in scope per PROJECT.md)
- P2: Add after validation
- P3: Future/out of scope

---

## UX Patterns for This Domain

Research across restaurant and food service inventory apps surfaces consistent UX patterns that match the existing HQ design system:

### Purchase History View
- **Card-per-event:** One card per PurchaseEvent showing date, vendor, total. Tap to expand line items.
- **Sorted newest-first:** Default sort is reverse chronological.
- **Vendor chips at top:** Toggle filter buttons (like tab pills in existing HQ pages).
- **Line item display:** Name, qty × unit, price per unit, subtotal. isCase shown as a pill badge.

### Category Spending Breakdown
- **Horizontal bar chart:** Bar length = total spend, color = tag category. Most effective for category comparison on narrow mobile screens (480px).
- **Dollar amounts labeled:** Show exact spend on each bar, not just relative length.
- **Sorted by spend descending:** Biggest categories first (usually Beef, then Produce).

### Spending Trend Chart
- **Vertical bar chart, monthly buckets:** X-axis = month, Y-axis = total spend. Stacked or grouped bars by category.
- **Simple is better:** No tooltips that require hover (mobile); show values inline or on tap.
- **Canvas API or inline SVG:** No charting library needed for basic bars; avoids external dependency. If complexity warrants, Chart.js via CDN (same pattern as SortableJS).

### Stock Level View
- **Traffic light badges:** Red pill = LOW, yellow pill = MEDIUM, green pill = HIGH. Color-coded at a glance.
- **Grouped by category (Tag):** Beef items together, Produce items together — matches how the owner thinks about ordering.
- **Days-since-last annotation:** "Last purchased 12 days ago" gives the owner context for the estimate.

### Reorder Suggestions
- **Flat list of LOW + MEDIUM items only:** No high-stock items; this is a "to buy" list.
- **Grouped by vendor (if known):** Helps owner consolidate into one PO per vendor.
- **Copy-friendly layout:** Simple list that can be read off the phone while writing a physical PO.

---

## Dependencies on Existing HQ Features

| Existing Feature | How Inventory Depends On It |
|------------------|-----------------------------|
| `sw.js` service worker | Must add `inventory.html` to ASSETS array and bump cache version |
| `index.html` launcher grid | Add an active tile linking to `inventory.html` |
| Users app permissions | Inventory access should be role-controlled (owner/manager vs crew) |
| CSS variables + dark mode | `inventory.html` uses the same `:root` variable block and `prefers-color-scheme` |
| `ptr.js` pull-to-refresh | Include on `inventory.html` like all other tool pages |
| MOCK_* data pattern | Follow the same `SCREAMING_SNAKE_CASE` constant pattern for mock purchase data |

---

## Tab Structure Recommendation

Following the 3-tab pattern established in `workflows.html` and `onboarding.html`:

| Tab | Label | Who Uses It | What It Shows |
|-----|-------|-------------|---------------|
| 1 | History | All roles | Purchase event list with line items and vendor filter |
| 2 | Trends | Manager+ | Category spending breakdown + monthly trend chart |
| 3 | Inventory | Manager+ | Stock level estimates + reorder suggestions |

Tab 1 (History) is crew-accessible — useful for receiving verification. Tabs 2 and 3 are manager-level analytics.

---

## Competitor Feature Analysis

| Feature | MarketMan (restaurant) | Toast Inventory | Our Approach |
|---------|----------------------|-----------------|--------------|
| Purchase history | Full invoice log with line items | PO tracking | Card list per event, expandable line items |
| Category breakdown | Cost by category chart | Category spend report | Bar chart by Tag |
| Stock levels | Count-based (manual entry) | Count-based | Frequency-based heuristic (no manual counts) |
| Reorder suggestions | Auto-generated POs | Min/max alerts | Display-only list for manual PO creation |
| Mobile UX | App (native iOS/Android) | App (native) | PWA, 480px mobile-first |
| Setup complexity | High (invoice import, recipe mapping) | High | Zero — mock data, no configuration |

**Key differentiator vs. commercial tools:** Zero manual count effort. The stock estimation heuristic uses existing purchase event data — no crew action required. Commercial tools require daily physical counts to be accurate. For a 1-5 person food truck this is the right tradeoff.

---

## Sources

- [Toast — Food Truck Inventory Management](https://pos.toasttab.com/blog/on-the-line/food-truck-inventory-management) — industry-standard features for food truck inventory
- [NetSuite — Restaurant Inventory Management Guide](https://www.netsuite.com/portal/resource/articles/inventory-management/restaurant-inventory-management.shtml) — stock level types, spend tracking patterns
- [Deliverect — Stock Management for Food Service](https://www.deliverect.com/en/blog/fulfilment/importance-of-effective-stock-management-for-food-service-success) — food service-specific stock management patterns
- [AppMaster — Inventory Reorder Suggestions Min/Max](https://appmaster.io/blog/inventory-reorder-suggestions-min-max-app) — min/max reorder UX patterns
- [Interlake Mecalux — Optimal Stock Level](https://www.interlakemecalux.com/blog/optimal-stock-level) — stock level calculation methods
- [Highcharts — Line vs Bar Chart](https://www.highcharts.com/blog/best-practices/line-chart-vs-bar-chart-choosing-the-right-one-for-your-objectives-and-data/) — chart type decision for spending trends
- [Square + MarketMan Integration](https://www.businesswire.com/news/home/20260402238712/en/Square-and-MarketMan-Launch-Advanced-Inventory-Management-Integration-for-Restaurants) — current state of restaurant inventory tooling (2026)
- [GetApp — Food Truck POS with Inventory Management](https://www.getapp.com/all-software/food-truck-pos-systems/f/inventory-management/) — competitive landscape

---
*Feature research for: Yumyums HQ v1.1 Inventory App*
*Researched: 2026-04-14*
