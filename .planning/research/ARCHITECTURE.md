# Architecture Research

**Domain:** Inventory app integration into existing Yumyums HQ PWA
**Researched:** 2026-04-14
**Confidence:** HIGH (derived from direct codebase inspection; no external sources needed for this integration question)

---

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      PWA Shell Layer                         │
│  index.html — launcher grid (2-column tile grid)             │
│  manifest.json  ·  sw.js  ·  ptr.js                          │
├─────────────────────────────────────────────────────────────┤
│                    Tool Pages (standalone)                    │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐             │
│  │purchasing  │  │workflows   │  │onboarding  │             │
│  │  .html     │  │  .html     │  │  .html     │             │
│  └────────────┘  └────────────┘  └────────────┘             │
│  ┌────────────┐  ┌────────────┐                             │
│  │ users.html │  │inventory   │  ← NEW (v1.1)               │
│  │            │  │  .html     │                             │
│  └────────────┘  └────────────┘                             │
├─────────────────────────────────────────────────────────────┤
│                   Shared Infrastructure                      │
│  ptr.js (pull-to-refresh)  ·  CSS variables (inline/page)   │
│  Service Worker (cache-first, covers all ASSETS)            │
├─────────────────────────────────────────────────────────────┤
│                     Mock Data Layer                          │
│  Hardcoded JS constants in each page's <script> block        │
│  No shared state between pages — each page is self-contained │
└─────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Implementation |
|-----------|----------------|----------------|
| `index.html` | Launcher grid; links to all active tools | `<a class="tile active">` per live tool; `<div class="tile soon">` for placeholders |
| `sw.js` | Cache-first PWA coverage across all pages | `CACHE` version string + `ASSETS` array; both must be updated for every new page |
| `inventory.html` | Purchase history, spending trends, stock estimates | New standalone page following the tool page contract |
| `users.html` APPS | Permission registry for users management | `APPS` array + `DEFAULT_PERMS` object; inventory must be added here |
| `ptr.js` | Pull-to-refresh gesture on iOS standalone | `<script src="ptr.js">` at bottom of every page |

---

## Recommended Project Structure

The project is flat — no subdirectories for tool pages. `inventory.html` follows the same flat layout as every other tool.

```
/                           # repo root — all files flat
├── inventory.html          # NEW — standalone inventory tool
├── index.html              # MODIFIED — add active tile for Inventory
├── sw.js                   # MODIFIED — bump CACHE version, add inventory.html to ASSETS
├── users.html              # MODIFIED — add inventory to APPS + DEFAULT_PERMS
├── purchasing.html         # UNCHANGED
├── workflows.html          # UNCHANGED
├── onboarding.html         # UNCHANGED
├── ptr.js                  # UNCHANGED
├── manifest.json           # UNCHANGED
└── tests/
    └── workflows.spec.js   # MODIFIED — add Playwright tests for inventory
```

### Structure Rationale

- **Flat layout:** Every tool lives at root level. Inventory follows the same pattern — no subdirectory, no abstraction layer.
- **Self-contained page:** All CSS, all JS, and all mock data live inline in `inventory.html`. No imports, no shared modules. This matches every other tool.

---

## Architectural Patterns

### Pattern 1: Tool Page Contract

**What:** Every tool page must implement a minimum set of boilerplate to participate in the PWA correctly.
**When to use:** Every new page added to HQ, including `inventory.html`.
**Trade-offs:** Boilerplate is duplicated per page. Acceptable given the no-build-step constraint and small number of pages.

**Required elements for inventory.html:**
```html
<!-- 1. Shared CSS variable block (identical across all pages) -->
<style>
:root{--bg:#f5f5f3;--card:#fff;--txt:#1a1a1a;--mut:#6b6b6b;--brd:rgba(0,0,0,0.08);--info-bg:#e6f1fb;--info-tx:#0c447c;--warn-bg:#faeeda;--warn-tx:#854f0b}
@media(prefers-color-scheme:dark){:root{--bg:#1a1a1a;--card:#262626;--txt:#f0f0f0;--mut:#999;--brd:rgba(255,255,255,0.1);--info-bg:#0c447c;--info-tx:#b5d4f4;--warn-bg:#633806;--warn-tx:#fac775}}
</style>

<!-- 2. Back link to HQ -->
<a class="back" href="index.html">HQ</a>

<!-- 3. Double-tap prevention + SW registration (inline) -->
<script>
document.addEventListener('dblclick', e => e.preventDefault());
if ('serviceWorker' in navigator) navigator.serviceWorker.register('sw.js');
</script>

<!-- 4. Pull-to-refresh (last script on page) -->
<script src="ptr.js"></script>
```

### Pattern 2: State-First Rendering with Tab Switch Re-render

**What:** Mutate JS state variables, then call a render function. On tab switch, re-render the target section entirely from state (not diffed). This pattern was established in `workflows.html` and fixed a real stale-state bug found during v1.0 verification.
**When to use:** Any view with multiple tabs inside `inventory.html` (History / Trends / Stock).
**Trade-offs:** Slightly more DOM work per switch; negligible at this data scale. The benefit is that stale state bugs are impossible.

```javascript
// State at top of script block
let ACTIVE_TAB = 1;
let FILTER_TAG = null;

function show(n) {
  [1, 2, 3].forEach(i => {
    document.getElementById('s' + i).style.display = i === n ? '' : 'none';
    document.getElementById('t' + i).className = i === n ? 'on' : '';
  });
  ACTIVE_TAB = n;
  render();
}

function render() {
  if (ACTIVE_TAB === 1) renderHistory();
  if (ACTIVE_TAB === 2) renderTrends();
  if (ACTIVE_TAB === 3) renderStock();
}
```

### Pattern 3: Event Delegation for Dynamic Lists

**What:** One `click` listener on a container `div` routes actions via `data-action` attributes. Introduced in `workflows.html`; use it in `inventory.html` for any list built via `innerHTML` template strings.
**When to use:** Purchase event list, stock group list — any dynamically built list.
**Trade-offs:** Slightly less intuitive than per-element handlers. Eliminates re-attaching listeners after each DOM rebuild.

```javascript
document.getElementById('history-list').addEventListener('click', e => {
  const row = e.target.closest('[data-action]');
  if (!row) return;
  const action = row.dataset.action;
  const id = row.dataset.id;
  if (action === 'expand-event') toggleEventDetail(id);
});
```

---

## Data Flow

### Mock Data Model (Baserow schema → JS constants)

The Baserow relational schema maps to flat JS constants. Denormalize where it simplifies rendering — the mock does not enforce referential integrity.

```
VENDORS          [{id, name, address}]
PURCHASE_EVENTS  [{id, bankTxId, date, vendorId, tax, total, totalUnits, totalCases}]
PURCHASES        [{id, name, qty, isCase, price, purchaseItemId, purchaseEventId}]
PURCHASE_ITEMS   [{id, description, groupIds[]}]
PURCHASE_ITEM_GROUPS  [{id, name, tagIds[], itemIds[]}]
TAGS             [{id, name}]
```

Denormalization strategy: `PURCHASE_EVENTS` can carry an inline `vendorName` string to avoid a join on every render. `PURCHASE_ITEM_GROUPS` can carry inline `tagNames[]`. Only preserve IDs where cross-view joins are genuinely needed.

### Request Flow (mock — no network)

```
User taps tab or filter
         ↓
show(n) / setFilter(tag)
         ↓
Mutate state variables (ACTIVE_TAB, FILTER_TAG)
         ↓
render() dispatches to renderHistory() / renderTrends() / renderStock()
         ↓
Derived data computed inline (join constants, aggregate, sort)
         ↓
innerHTML template string → DOM updated
```

### Key Data Flows

1. **Purchase History tab:** Iterate `PURCHASE_EVENTS` sorted by date descending. For each event, look up vendor name. On row expand, filter `PURCHASES` by `purchaseEventId` to render line items. Collapse/expand via `data-action="expand-event"` on the row.

2. **Spending Trends tab:** Group `PURCHASES` → `purchaseItemId` → `PURCHASE_ITEMS.groupIds` → `PURCHASE_ITEM_GROUPS.tagIds` → `TAGS`. Sum `price * qty` per tag per time bucket (week or month). Render as CSS-only horizontal bar chart: each bar is a `div` with `width` set as a percentage of the maximum bucket value. No canvas, no chart library — zero external dependencies.

3. **Stock Estimation tab:** For each `PURCHASE_ITEM_GROUP`, find the most recent `PURCHASE` for items in that group. Compute days since last purchase. Assign `low / medium / high` based on a simple threshold heuristic. Surface `low` and `medium` groups at the top as reorder suggestions.

### Stock Level Heuristic (mock implementation)

```javascript
function stockLevel(groupId) {
  // Find most recent purchase event for any item in this group
  const groupItemIds = PURCHASE_ITEM_GROUPS.find(g => g.id === groupId)?.itemIds || [];
  const recentPurchase = PURCHASES
    .filter(p => groupItemIds.includes(p.purchaseItemId))
    .sort((a, b) => {
      const dateA = PURCHASE_EVENTS.find(e => e.id === a.purchaseEventId)?.date || '';
      const dateB = PURCHASE_EVENTS.find(e => e.id === b.purchaseEventId)?.date || '';
      return dateB.localeCompare(dateA);
    })[0];

  if (!recentPurchase) return 'low';
  const event = PURCHASE_EVENTS.find(e => e.id === recentPurchase.purchaseEventId);
  const daysSince = (Date.now() - new Date(event.date)) / 86400000;

  if (daysSince < 4) return 'high';
  if (daysSince < 8) return 'medium';
  return 'low';
}
```

This heuristic is intentionally simple. It answers "when did we last buy from this group?" not "how much stock remains." Real consumption tracking is explicitly out of scope.

---

## Integration Points

### New vs Modified Components

| Component | Change Type | What Changes |
|-----------|-------------|--------------|
| `inventory.html` | NEW | Standalone tool page; 3-tab layout (History / Trends / Stock); all CSS/JS/mock data inline |
| `index.html` | MODIFIED | Activate inventory tile; convert the existing "BI" soon-tile to an active Inventory tile (see tile placement rationale below) |
| `sw.js` | MODIFIED | Bump `CACHE` version string; add `'./inventory.html'` to `ASSETS` array |
| `users.html` | MODIFIED | Add `{slug:'inventory', name:'Inventory', icon:'📦'}` to `APPS`; add `inventory` key to `DEFAULT_PERMS` |
| `tests/workflows.spec.js` | MODIFIED | Add Playwright E2E tests for inventory (or new `tests/inventory.spec.js` file) |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `index.html` → `inventory.html` | HTML anchor link | One-way navigation; no state passed between pages |
| `inventory.html` → `index.html` | Back link (`‹ HQ`) | Standard `.back` pattern used on all tool pages |
| `users.html` APPS → permissions | In-page JS constants | `inventory` slug in `APPS` makes it appear in the Access tab; `DEFAULT_PERMS` sets role defaults |
| `sw.js` ASSETS → `inventory.html` | Cache list | Page returns 404 from cache in offline mode until added to ASSETS and cache version is bumped |

### Tile Placement Decision

The `index.html` grid has one immediately relevant placeholder: the "BI" tile (`📊`, "Sales and cost trends"). Inventory overlaps with BI conceptually. Two options:

- **Option A (recommended):** Replace the "BI" soon-tile with an active Inventory tile (`📦`). BI is explicitly out of scope. Inventory is the concrete v1.1 deliverable. Cleaner grid — no placeholder clutter.
- **Option B:** Add a new tile alongside BI (grid becomes 2×4). Adds visual weight; BI placeholder has no near-term plan.

Recommendation: Option A. Replace BI. If BI gets built later it gets its own tile back then.

### Permissions Default

```javascript
// users.html — add to APPS array
{slug:'inventory', name:'Inventory', icon:'📦'}

// users.html — add to DEFAULT_PERMS
inventory: {admin: true, manager: true, team_member: false}
```

Owner and managers track spending trends and stock levels. Crew does not need access in the mock.

---

## Suggested Build Order

Dependencies determine this order. Each step is testable before the next begins.

### Step 1: Page Shell + SW Integration

Create `inventory.html` with the tool page contract boilerplate, 3-tab header (History / Trends / Stock), and empty section containers. Update `sw.js` (bump cache version, add to ASSETS) and `index.html` (active tile). This proves the page loads, caches, and is reachable from HQ before any feature logic is written.

Playwright tests: page loads from HQ tile, back link returns to HQ, all three tabs are present and clickable, SW registers correctly.

### Step 2: Mock Data Constants

Define all mock data at the top of the `inventory.html` script block: `VENDORS`, `PURCHASE_EVENTS`, `PURCHASES`, `PURCHASE_ITEMS`, `PURCHASE_ITEM_GROUPS`, `TAGS`. Use realistic food truck data (salmon, produce, sauces, containers). No rendering yet — data layer in place and inspectable via browser console.

No Playwright test needed here (data is tested implicitly by render steps below).

### Step 3: Purchase History Tab

Implement `renderHistory()` — sorted list of purchase events with vendor name, date, total. Tappable rows expand to show line items (name, qty, price). This is pure data rendering with no computation — just joins and template strings. Most straightforward tab; validates the data model before derived views are built.

Playwright tests: history tab renders events, each event row shows vendor and total, tap to expand shows line items.

### Step 4: Stock Estimation Tab

Implement `renderStock()` — `stockLevel()` heuristic per group, low/medium/high badge, reorder suggestions section at top for flagged groups. Build before Trends because the algorithm is self-contained and the output (stock levels) is more immediately useful to the owner than charts.

Playwright tests: stock tab renders all item groups, low/medium items appear in reorder suggestions, high items do not, badges display correct level labels.

### Step 5: Spending Trends Tab

Implement `renderTrends()` — CSS bar chart showing spending by tag over time. Most visually complex tab; built last so earlier tabs are already working. Tag filter (tap to isolate one category) is a stretch goal within this step.

Playwright tests: trends tab renders, bars have nonzero width, all tag labels appear, filtering by tag narrows the display.

### Step 6: Users Permissions Entry

Add `inventory` to `APPS` and `DEFAULT_PERMS` in `users.html`. This is an independent users.html change with no effect on `inventory.html`.

Playwright test: inventory appears in the users Access tab for admin users.

---

## Anti-Patterns

### Anti-Pattern 1: Sharing Data Between Pages via localStorage

**What people do:** Store computed spending totals or stock levels in `localStorage` so `purchasing.html` or `index.html` can read them.
**Why it's wrong:** Violates the self-contained page contract. Creates hidden coupling. Produces stale-data bugs when mock data changes. `localStorage` is explicitly out of scope per PROJECT.md constraints.
**Do this instead:** Keep all mock data inside `inventory.html`. If another page needs similar data in the future, duplicate the relevant constants. Duplication is acceptable at mock stage.

### Anti-Pattern 2: Using a Chart Library for Trends

**What people do:** Add `<script src="https://cdn.jsdelivr.net/npm/chart.js">` for the spending bar chart.
**Why it's wrong:** Adds an external CDN dependency (only SortableJS is permitted in this codebase). Breaks offline capability. Overkill for 4-6 tag categories on a mobile screen.
**Do this instead:** CSS-only horizontal bars. Each bar is a `<div>` where `width` is set as a percentage of the maximum bucket value computed inline. Zero dependencies, offline-safe, readable on mobile.

### Anti-Pattern 3: Deep Normalization in Mock Constants

**What people do:** Faithfully replicate every Baserow M2M join table as its own JS array, then write multi-hop join traversal for every render.
**Why it's wrong:** The mock does not enforce referential integrity. Excessive normalization produces verbose render functions without gaining any correctness benefit.
**Do this instead:** Denormalize where it simplifies rendering. Embed `vendorName` directly on `PURCHASE_EVENTS`. Embed `tagNames[]` directly on `PURCHASE_ITEM_GROUPS`. Retain IDs only where a join is genuinely needed across views.

### Anti-Pattern 4: Forgetting to Bump SW Cache Version

**What people do:** Add `inventory.html` to `ASSETS` but forget to change the `CACHE` version string (or vice versa).
**Why it's wrong:** Old service worker stays active. Users get a cached 404 when navigating to the new page, or new page loads but old SW serves stale assets. Silently broken, hard to diagnose on device.
**Do this instead:** Always update BOTH in the same commit: `CACHE = 'yumyums-v43'` AND `'./inventory.html'` added to `ASSETS`. Per CLAUDE.md: bump before every deploy and human-verify checkpoint.

### Anti-Pattern 5: Rendering Stock Estimates on History Tab Load

**What people do:** Compute stock levels during `renderHistory()` and display them inline next to purchase events.
**Why it's wrong:** Stock estimation is derived from all purchase events, not just one. Mixing it into the history view makes the algorithm harder to isolate, test, and adjust independently.
**Do this instead:** Keep stock estimation strictly in `renderStock()`. Let the tabs be single-responsibility. If cross-tab callouts are needed (e.g., "this item is low stock" badge on history rows), add that as a late enhancement after the Stock tab is working.

---

## Scaling Considerations

This architecture is intentionally scoped to the mock phase. The component and data boundaries are designed so the transition to a real backend is a data layer swap, not an architectural rewrite.

| Concern | Mock Phase | With Backend |
|---------|------------|--------------|
| Purchase data | `const PURCHASE_EVENTS = [...]` inline | `GET /api/purchases?from=&to=` |
| Stock levels | `stockLevel()` date heuristic | Computed server-side from inventory counts |
| Spending trends | Inline aggregation of mock arrays | `GET /api/spending/by-tag?range=30d` |
| Auth/roles | No gate (mock) | Check session role before rendering tabs |
| Offline | Cache-first covers static page | Same; data freshness becomes a concern |

---

## Sources

- Direct codebase inspection: `index.html`, `sw.js`, `users.html`, `purchasing.html`, `onboarding.html`, `workflows.html`, `ptr.js`
- `.planning/PROJECT.md` — milestone requirements, out-of-scope constraints
- `CLAUDE.md` — project conventions (SW cache bump, static-only, naming patterns, event delegation)

---
*Architecture research for: Yumyums HQ v1.1 Inventory App integration*
*Researched: 2026-04-14*
