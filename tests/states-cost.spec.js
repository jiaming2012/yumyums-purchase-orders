const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

// states-cost.spec.js — the CLAUDE.md self-verification ritual for the Cost tab
// (`#s6`, FR-4 / AC-4). Every row of the PRD "State Enumeration — Cost tab (`#s6`)"
// table (.night-crew/knowledge/prds/PRD-prove-and-surface.md) is FORCED here,
// navigated to, and screenshotted so the PNGs can be read back and compared to
// the visual contract.
//
// PRD rows covered by this file:
//   Empty                          -> 'Empty'                (live endpoint, no daily_menu_sales)
//   Loading                        -> 'Loading'              (delayed route)
//   Error                          -> 'Error'                (500 route)
//   Populated                      -> 'Populated'            (mocked fixture)
//   Edge: sales but no recipe      -> 'Edge: no recipe'      (unallocated:"no recipe" row)
//   Edge: zero-revenue item        -> 'Edge: zero revenue'   (revenue 0, pct null)
//   Edge: ungated user             -> OWNED BY F5 (inventory-tab-gating) — see NOTE below
//
// Phase-specific edge rows added beyond the PRD table (CLAUDE.md "at least 2
// phase-specific edge rows"):
//   Edge: partially-sparse         -> costs present, zero sales across the board
//   Edge: long menu item name      -> layout does not blow out the 480px shell
//   Edge: known 0% food-cost gap   -> the flattering-number row renders as data says
//
// NOTE FOR F5 (`inventory-tab-gating`): the "Edge: ungated user" row belongs to
// F5, not to this card. It slots in as a new `test.describe('Cost tab — gating')`
// block at the BOTTOM of this file: log in as a user WITHOUT the `inventory-cost`
// grant, assert `#t6` / `#s6` are not rendered, assert a direct
// `GET /api/v1/inventory/cost` returns 403, and screenshot as
// `edge-ungated.png` via the same `shot()` helper below.

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

const SHOT_DIR = path.join(__dirname, '..', 'test-results', 'states-cost');
fs.mkdirSync(SHOT_DIR, { recursive: true });

async function shot(page, name) {
  await page.screenshot({ path: path.join(SHOT_DIR, name + '.png'), fullPage: true });
}

async function login(page) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// Navigate straight to the Cost tab via the shared #tab= hash contract (tab.js).
async function openCostTab(page) {
  await page.goto('/inventory.html#tab=6');
  await page.waitForSelector('#s6:visible');
}

const WINDOW = { from: '2026-04-27', to: '2026-07-19', weeks: 12 };

const ID = {
  wrap: '11111111-1111-1111-1111-111111111111',
  salmon: '22222222-2222-2222-2222-222222222222',
  fries: '33333333-3333-3333-3333-333333333333',
  comped: '44444444-4444-4444-4444-444444444444',
  longname: '55555555-5555-5555-5555-555555555555',
  lemonade: '66666666-6666-6666-6666-666666666666',
};

// The populated fixture deliberately carries ALL THREE null shapes plus the
// known 0%-food-cost gap row, so one screenshot proves the whole contract.
const POPULATED = {
  window: WINDOW,
  rows: [
    { menu_item_id: ID.wrap, menu_item_name: 'Chicken Wrap', menu_group: 'Wraps',
      units_sold: 240, revenue: 1800.00, ingredient_cost_total: 522.00,
      margin: 1278.00, food_cost_pct: 29.00, unallocated: null },
    { menu_item_id: ID.salmon, menu_item_name: 'Salmon Bowl', menu_group: 'Bowls',
      units_sold: 143, revenue: 1287.00, ingredient_cost_total: 402.11,
      margin: 884.89, food_cost_pct: 31.24, unallocated: null },
    { menu_item_id: ID.fries, menu_item_name: 'Truffle Fries', menu_group: 'Sides',
      units_sold: 88, revenue: 440.00, ingredient_cost_total: 281.60,
      margin: 158.40, food_cost_pct: 64.00, unallocated: null },
    // null shape 1 — zero revenue (comped): pct null, negative margin is REAL.
    { menu_item_id: ID.comped, menu_item_name: 'Comped Special', menu_group: 'Specials',
      units_sold: 4, revenue: 0, ingredient_cost_total: 11.20,
      margin: -11.20, food_cost_pct: null, unallocated: null },
    // null shape 2 — sales but no recipe: cost/margin/pct all null + reason string.
    { menu_item_id: ID.longname,
      menu_item_name: 'Seasonal Harvest Bowl with Roasted Root Vegetables and Tahini Drizzle',
      menu_group: 'Bowls', units_sold: 31, revenue: 402.50,
      ingredient_cost_total: null, margin: null, food_cost_pct: null,
      unallocated: 'no recipe' },
    // KNOWN GAP (routed to operator): recipe exists, zero in-window ingredient
    // spend -> a flattering 0% food cost / 100% margin. Rendered as the data
    // says; deliberately NOT special-cased into "—".
    { menu_item_id: ID.lemonade, menu_item_name: 'House Lemonade', menu_group: 'Beverages',
      units_sold: 512, revenue: 1024.00, ingredient_cost_total: 0,
      margin: 1024.00, food_cost_pct: 0, unallocated: null },
  ],
  // FULL orderings (not pre-truncated) — the client slices to its strip length.
  movers: {
    by_food_cost_pct: {
      best: [ID.lemonade, ID.wrap, ID.salmon, ID.fries],
      worst: [ID.fries, ID.salmon, ID.wrap, ID.lemonade],
    },
    by_margin: {
      best: [ID.wrap, ID.lemonade, ID.salmon, ID.fries, ID.comped],
      worst: [ID.comped, ID.fries, ID.salmon, ID.lemonade, ID.wrap],
    },
  },
};

// Purchases exist, Toast sales do not: every revenue 0, every margin negative,
// every pct null. This MUST read as "no sales data yet", not "everything is
// losing money".
const PARTIALLY_SPARSE = {
  window: WINDOW,
  rows: [
    { menu_item_id: ID.wrap, menu_item_name: 'Chicken Wrap', menu_group: 'Wraps',
      units_sold: 0, revenue: 0, ingredient_cost_total: 522.00,
      margin: -522.00, food_cost_pct: null, unallocated: null },
    { menu_item_id: ID.salmon, menu_item_name: 'Salmon Bowl', menu_group: 'Bowls',
      units_sold: 0, revenue: 0, ingredient_cost_total: 402.11,
      margin: -402.11, food_cost_pct: null, unallocated: null },
    { menu_item_id: ID.fries, menu_item_name: 'Truffle Fries', menu_group: 'Sides',
      units_sold: 0, revenue: 0, ingredient_cost_total: 281.60,
      margin: -281.60, food_cost_pct: null, unallocated: null },
  ],
  movers: {
    by_food_cost_pct: { best: [], worst: [] },
    by_margin: {
      best: [ID.fries, ID.salmon, ID.wrap],
      worst: [ID.wrap, ID.salmon, ID.fries],
    },
  },
};

function mockCost(page, body, opts = {}) {
  return page.route('**/api/v1/inventory/cost*', async route => {
    if (opts.delayMs) await new Promise(r => setTimeout(r, opts.delayMs));
    if (opts.status && opts.status >= 400) {
      return route.fulfill({ status: opts.status, contentType: 'application/json',
        body: JSON.stringify({ error: 'internal_error' }) });
    }
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
  });
}

test.describe('Cost tab (#s6) — State Enumeration', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  // ── PRD row: Empty ──────────────────────────────────────────────────────
  // Forced against the LIVE endpoint: the test DB has no daily_menu_sales rows
  // (TOAST_SYNC_INTERVAL=0), which is exactly the accept-sparse-prod condition.
  test('Empty — no sales data renders the honest low-data card', async ({ page }) => {
    const resp = page.waitForResponse(r => r.url().includes('/api/v1/inventory/cost'));
    await openCostTab(page);
    const r = await resp;
    expect(r.status()).toBe(200);
    const body = await r.json();
    expect(Array.isArray(body.rows)).toBe(true);
    expect(body.rows.length).toBe(0);

    const empty = page.locator('#cost-container .cost-empty');
    await expect(empty).toBeVisible();
    await expect(empty).toContainText('No sales data yet');
    // Honest: never a fabricated number.
    await expect(page.locator('#cost-container')).not.toContainText('NaN');
    await expect(page.locator('#cost-container')).not.toContainText('Infinity');
    await expect(page.locator('#cost-container .cost-table')).toHaveCount(0);
    await shot(page, 'empty');
  });

  // ── PRD row: Loading ────────────────────────────────────────────────────
  test('Loading — skeleton shown in #s6 while the request is in flight', async ({ page }) => {
    await mockCost(page, POPULATED, { delayMs: 2500 });
    await page.goto('/inventory.html#tab=6');
    await page.waitForSelector('#s6:visible');
    await expect(page.locator('#cost-container .skeleton').first()).toBeVisible();
    await shot(page, 'loading');
    // and it resolves rather than hanging on the skeleton forever
    await expect(page.locator('#cost-container .cost-table')).toBeVisible({ timeout: 10000 });
  });

  // ── PRD row: Error ──────────────────────────────────────────────────────
  test('Error — inline error card + Retry, never a blank tab', async ({ page }) => {
    await mockCost(page, null, { status: 500 });
    await openCostTab(page);
    const container = page.locator('#cost-container');
    await expect(container.getByRole('button', { name: /retry/i })).toBeVisible();
    await expect(container).not.toBeEmpty();
    await shot(page, 'error');
  });

  // ── PRD row: Populated ──────────────────────────────────────────────────
  test('Populated — sortable table + two movers strips', async ({ page }) => {
    await mockCost(page, POPULATED);
    await openCostTab(page);

    await expect(page.locator('#cost-container .cost-table')).toBeVisible();
    await expect(page.locator('#cost-container .cost-table tbody tr')).toHaveCount(6);

    // Window is displayed from the server envelope, not recomputed.
    await expect(page.locator('#cost-window')).toContainText('12 weeks');

    // TWO movers strips: by food-cost-% and by margin dollars.
    await expect(page.locator('.cost-mover-strip[data-strip="food_cost_pct"]')).toBeVisible();
    await expect(page.locator('.cost-mover-strip[data-strip="margin"]')).toBeVisible();

    // Movers are sliced client-side from the FULL ordering, capped at 3 a side.
    const bestPct = page.locator('.cost-mover-strip[data-strip="food_cost_pct"] .cost-mover-row[data-side="best"]');
    await expect(bestPct.first()).toContainText('House Lemonade');
    const worstPct = page.locator('.cost-mover-strip[data-strip="food_cost_pct"] .cost-mover-row[data-side="worst"]');
    await expect(worstPct.first()).toContainText('Truffle Fries');

    // No item may appear on BOTH sides of a strip. With a short ordering a
    // naive top-3-of-each does exactly that (4 rankable rows -> Salmon Bowl
    // both "best" and "worst"), which is a contradiction, not a ranking.
    for (const strip of ['food_cost_pct', 'margin']) {
      const sel = '.cost-mover-strip[data-strip="' + strip + '"] .cost-mover-row';
      const best = await page.locator(sel + '[data-side="best"]').evaluateAll(
        els => els.map(e => e.getAttribute('data-menu-item-id')));
      const worst = await page.locator(sel + '[data-side="worst"]').evaluateAll(
        els => els.map(e => e.getAttribute('data-menu-item-id')));
      expect(best.length).toBeLessThanOrEqual(3);
      expect(worst.length).toBeLessThanOrEqual(3);
      expect(best.filter(id => worst.includes(id))).toEqual([]);
    }

    // A negative-margin row legitimately leads the worst-margin strip.
    const worstMargin = page.locator('.cost-mover-strip[data-strip="margin"] .cost-mover-row[data-side="worst"]');
    await expect(worstMargin.first()).toContainText('Comped Special');

    // Numbers render to the cent.
    const salmon = page.locator('.cost-table tbody tr[data-menu-item-id="' + ID.salmon + '"]');
    await expect(salmon.locator('[data-col="revenue"]')).toContainText('1,287.00');
    await expect(salmon.locator('[data-col="cost"]')).toContainText('402.11');
    await expect(salmon.locator('[data-col="margin"]')).toContainText('884.89');
    await expect(salmon.locator('[data-col="pct"]')).toContainText('31.2');

    await shot(page, 'populated');
  });

  test('Populated — table sorts on header click', async ({ page }) => {
    await mockCost(page, POPULATED);
    await openCostTab(page);

    const names = () => page.locator('.cost-table tbody tr').evaluateAll(
      els => els.map(e => e.getAttribute('data-menu-item-id')));

    // Default: revenue descending.
    expect((await names())[0]).toBe(ID.wrap);

    // Sort by food-cost % ascending -> lowest % first (House Lemonade, 0%).
    await page.click('.cost-table th[data-sort="pct"]');
    expect((await names())[0]).toBe(ID.lemonade);

    // Toggle -> descending -> highest % first (Truffle Fries, 64%).
    await page.click('.cost-table th[data-sort="pct"]');
    expect((await names())[0]).toBe(ID.fries);

    // Units descending.
    await page.click('.cost-table th[data-sort="units"]');
    const u = await names();
    expect(u[0]).toBe(ID.lemonade);

    await shot(page, 'populated-sorted');
  });

  // ── PRD row: Edge — sales but no recipe (unallocated) ───────────────────
  test('Edge: no recipe — revenue + units shown, cost/margin/% marked, never a silent 0', async ({ page }) => {
    await mockCost(page, POPULATED);
    await openCostTab(page);

    const row = page.locator('.cost-table tbody tr[data-menu-item-id="' + ID.longname + '"]');
    await expect(row.locator('[data-col="units"]')).toContainText('31');
    await expect(row.locator('[data-col="revenue"]')).toContainText('402.50');
    // The three nulls render as an em-dash, NOT as $0.00 / 0%.
    await expect(row.locator('[data-col="cost"]')).toHaveText('—');
    await expect(row.locator('[data-col="margin"]')).toHaveText('—');
    await expect(row.locator('[data-col="pct"]')).toHaveText('—');
    // ...and the reason string from the endpoint is surfaced verbatim.
    await expect(row.locator('.cost-unalloc')).toContainText('no recipe');

    // It is absent from BOTH movers strips (no % and no margin to rank on).
    await expect(page.locator('.cost-mover-row[data-menu-item-id="' + ID.longname + '"]')).toHaveCount(0);

    await shot(page, 'edge-no-recipe');
  });

  // ── PRD row: Edge — zero-revenue item ───────────────────────────────────
  test('Edge: zero revenue — % renders "—", units + cost + negative margin still shown', async ({ page }) => {
    await mockCost(page, POPULATED);
    await openCostTab(page);

    const row = page.locator('.cost-table tbody tr[data-menu-item-id="' + ID.comped + '"]');
    await expect(row.locator('[data-col="units"]')).toContainText('4');
    await expect(row.locator('[data-col="cost"]')).toContainText('11.20');
    // The negative margin is REAL and must be shown.
    await expect(row.locator('[data-col="margin"]')).toContainText('11.20');
    await expect(row.locator('[data-col="margin"]')).toContainText('-');
    // No divide-by-zero, no Infinity, no 0%.
    await expect(row.locator('[data-col="pct"]')).toHaveText('—');
    await expect(row).not.toContainText('Infinity');
    await expect(row).not.toContainText('NaN');

    await shot(page, 'edge-zero-revenue');
  });

  // ── Phase-specific edge: partially-sparse (costs, no sales) ─────────────
  test('Edge: partially-sparse — reads as "no sales data yet", not "everything is losing money"', async ({ page }) => {
    await mockCost(page, PARTIALLY_SPARSE);
    await openCostTab(page);

    const note = page.locator('#cost-container .cost-sparse-note');
    await expect(note).toBeVisible();
    await expect(note).toContainText('No sales data yet');

    // The margin column is NOT dressed up as a loss verdict: no movers strips
    // are shown when nothing in the window has revenue to rank against.
    await expect(page.locator('.cost-mover-strip')).toHaveCount(0);

    // The real ingredient spend is still visible — nothing is hidden.
    await expect(page.locator('.cost-table tbody tr')).toHaveCount(3);
    const wrap = page.locator('.cost-table tbody tr[data-menu-item-id="' + ID.wrap + '"]');
    await expect(wrap.locator('[data-col="cost"]')).toContainText('522.00');
    await expect(wrap.locator('[data-col="pct"]')).toHaveText('—');

    await shot(page, 'edge-partially-sparse');
  });

  // ── Phase-specific edge: long menu item name ───────────────────────────
  test('Edge: long menu item name — does not blow out the 480px mobile shell', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await mockCost(page, POPULATED);
    await openCostTab(page);

    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow).toBeLessThanOrEqual(1);

    await shot(page, 'edge-long-name');
  });

  // ── Phase-specific edge: the KNOWN 0%-food-cost gap ────────────────────
  // Recipe exists but no in-window ingredient spend -> the endpoint publishes
  // 0 cost / full margin / 0%. This is a KNOWN OPEN FORK routed to the operator
  // (an honest fix needs a new reason string = design amendment). The tab
  // renders it as the data says: it is NOT special-cased into "—", because that
  // would misrepresent genuinely zero-cost items.
  test('Edge: known 0%-food-cost gap — rendered as the data says, not papered over', async ({ page }) => {
    await mockCost(page, POPULATED);
    await openCostTab(page);

    const row = page.locator('.cost-table tbody tr[data-menu-item-id="' + ID.lemonade + '"]');
    await expect(row.locator('[data-col="cost"]')).toContainText('0.00');
    await expect(row.locator('[data-col="pct"]')).toContainText('0.0');
    // NOT the null treatment.
    await expect(row.locator('[data-col="pct"]')).not.toHaveText('—');

    await shot(page, 'edge-zero-cost-gap');
  });
});
