const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

// states-bi-hub.spec.js — the CLAUDE.md self-verification ritual for the BI
// hub (bi.html `#s0`, B-455 / WO-2b). The two reports keep their own state
// tables (states-trends / states-cost); this file forces the HUB's rows:
//
//   Loading            -> badge skeletons, "Loading…" subtitles
//   Empty              -> no badge (never "$0 this wk"), descriptive subtitles
//   Populated          -> "$N this wk" (latest week: cells + unlinked) and
//                         "N% avg" (mean food-cost % over dishes that have one),
//                         window subtitles as plain text
//   Error              -> "Status unavailable", no badge, row still opens
//   Edge: half-error   -> one row errors, the other still carries its reading
//   Edge: deep link    -> #tab=2 opens Food cost with the "BI · Food cost" back
//                         link; the phone back gesture returns to the hub
//   Edge: 393px        -> every row ≥44px, the badge never overlaps the title

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const SHOT_DIR = path.join(__dirname, '..', 'test-results', 'states-bi-hub');
fs.mkdirSync(SHOT_DIR, { recursive: true });
test.use({ viewport: { width: 393, height: 852 } });

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

const W = ['2026-07-13', '2026-07-20', '2026-07-27', '2026-08-03', '2026-08-10', '2026-08-17', '2026-08-24', '2026-08-31', '2026-09-07', '2026-09-14', '2026-09-21', '2026-09-28'];
const cells = [];
W.forEach((w, i) => {
  cells.push({ week_start: w, group_id: 'g1', spend: 1500 + i * 40 });
  cells.push({ week_start: w, group_id: 'g2', spend: 500 + i * 10 });
  cells.push({ week_start: w, group_id: 'g3', spend: 300 + i * 8 });
});
// Latest week (Sep 28): 1940 + 610 + 388 = 2938, plus 120 unlinked = 3058.
const TRENDS = {
  window: { from: W[0], to: '2026-10-04', weeks: 12 },
  groups: [{ id: 'g1', name: 'Proteins' }, { id: 'g2', name: 'Produce' }, { id: 'g3', name: 'Dry goods' }],
  cells, unlinked: [{ week_start: W[11], spend: 120 }], unlinked_total: 120,
  completeness: { pending_total: 0, pending_count: 0, unitemized_remainder: 0, reconciles_to_cogs_excl_tax: 30000 },
};
const EMPTY_TRENDS = { window: { from: W[0], to: '2026-10-04', weeks: 12 }, groups: [], cells: [], unlinked: [], unlinked_total: 0, completeness: { pending_total: 0, pending_count: 0, unitemized_remainder: 0, reconciles_to_cogs_excl_tax: 0 } };
// (31 + 24 + 40) / 3 = 31.67 → "32% avg"; the null row is left out of the mean.
const COST = {
  window: { from: '2026-09-21', to: '2026-09-27', weeks: 1 },
  rows: [
    { menu_item_id: 'm1', menu_item_name: 'Smash Burger', menu_group: 'Burgers', units_sold: 120, revenue: 1440, ingredient_cost_total: 446.4, margin: 993.6, food_cost_pct: 31, unallocated: null },
    { menu_item_id: 'm2', menu_item_name: 'Loaded Fries', menu_group: 'Sides', units_sold: 80, revenue: 560, ingredient_cost_total: 134.4, margin: 425.6, food_cost_pct: 24, unallocated: null },
    { menu_item_id: 'm3', menu_item_name: 'Lemonade', menu_group: 'Drinks', units_sold: 60, revenue: 240, ingredient_cost_total: 96, margin: 144, food_cost_pct: 40, unallocated: null },
    { menu_item_id: 'm4', menu_item_name: 'Water', menu_group: 'Drinks', units_sold: 10, revenue: 0, ingredient_cost_total: null, margin: null, food_cost_pct: null, unallocated: 'no recipe' },
  ],
  movers: { by_food_cost_pct: { best: ['m2'], worst: ['m3'] }, by_margin: { best: ['m1'], worst: ['m3'] } },
};
const EMPTY_COST = { window: { from: '2026-09-21', to: '2026-09-27', weeks: 1 }, rows: [], movers: { by_food_cost_pct: { best: [], worst: [] }, by_margin: { best: [], worst: [] } } };

const json = body => route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
const fail = route => route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"boom"}' });
async function mock(page, trends, cost) {
  await page.route('**/api/v1/inventory/trends', trends === 'fail' ? fail : json(trends));
  await page.route('**/api/v1/inventory/cost', cost === 'fail' ? fail : json(cost));
}
async function openHub(page) {
  await page.goto('/bi.html');
  await page.waitForSelector('#s0:visible');
}
async function settled(page) {
  await page.waitForFunction(() => !document.querySelector('#hub-b1.skel') && !document.querySelector('#hub-b2.skel'));
}

test.describe('BI hub — B-455 WO-2b state table', () => {

  test('Loading: both badges are skeletons and subtitles read Loading…', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/trends', async route => { await new Promise(r => setTimeout(r, 1500)); await json(TRENDS)(route); });
    await page.route('**/api/v1/inventory/cost', async route => { await new Promise(r => setTimeout(r, 1500)); await json(COST)(route); });
    await openHub(page);
    await expect(page.locator('#hub-b1')).toHaveClass(/skel/);
    await expect(page.locator('#hub-b2')).toHaveClass(/skel/);
    await expect(page.locator('#hub-s1')).toHaveText('Loading…');
    await expect(page.locator('#hub-s2')).toHaveText('Loading…');
    await shot(page, 'loading');
  });

  test('Empty: no badge on either row, descriptive subtitles, rows still open', async ({ page }) => {
    await login(page);
    await mock(page, EMPTY_TRENDS, EMPTY_COST);
    await openHub(page);
    await settled(page);
    await expect(page.locator('#hub-b1')).toBeHidden();
    await expect(page.locator('#hub-b2')).toBeHidden();
    await expect(page.locator('#hub-b1')).toHaveText('');
    await expect(page.locator('#hub-s1')).toContainText('Spend by group');
    await expect(page.locator('#hub-s2')).toContainText('Margin per dish');
    await page.locator('#t1').click();
    await expect(page.locator('#s1 .tr-empty')).toBeVisible();
    await shot(page, 'empty');
  });

  test('Populated: latest-week spend and average food cost as muted readings; subtitles are plain text', async ({ page }) => {
    await login(page);
    await mock(page, TRENDS, COST);
    await openHub(page);
    await settled(page);
    await expect(page.locator('#hub-b1')).toHaveText('$3,058 this wk');
    await expect(page.locator('#hub-b1')).toHaveClass(/mut/);
    await expect(page.locator('#hub-b2')).toHaveText('32% avg');
    await expect(page.locator('#hub-b2')).toHaveClass(/mut/);
    await expect(page.locator('#hub-s1')).toHaveText('Spend by group · 12 weeks');
    // The window label is TEXT: no entity source leaks ("&middot;", "&ndash;"), and
    // it names the dates, not a "1 weeks" count.
    await expect(page.locator('#hub-s2')).toHaveText('Margin per dish · Sep 21 – Sep 27');
    await expect(page.locator('#hub-s2')).not.toContainText('&');
    await shot(page, 'populated');
    await page.emulateMedia({ colorScheme: 'dark' });
    await shot(page, 'populated-dark');
  });

  test('Error: both calls fail → "Status unavailable", no badge, and the row still opens to the error card', async ({ page }) => {
    await login(page);
    await mock(page, 'fail', 'fail');
    await openHub(page);
    await settled(page);
    await expect(page.locator('#hub-s1')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-s2')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-b1')).toBeHidden();
    await expect(page.locator('#hub-b2')).toBeHidden();
    await page.locator('#t2').click();
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#cost-container')).toContainText('Couldn’t load food cost');
    await shot(page, 'error');
  });

  test('Edge: one call fails — that row says so, the other keeps its reading', async ({ page }) => {
    await login(page);
    await mock(page, 'fail', COST);
    await openHub(page);
    await settled(page);
    await expect(page.locator('#hub-s1')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-b1')).toBeHidden();
    await expect(page.locator('#hub-b2')).toHaveText('32% avg');
    await shot(page, 'edge-half-error');
  });

  test('Edge: deep link #tab=2 opens Food cost with the BI · Food cost back link; back gesture returns to the hub', async ({ page }) => {
    await login(page);
    await mock(page, TRENDS, COST);
    await page.goto('/bi.html#tab=2');
    await page.waitForSelector('#s2:visible');
    await expect(page.locator('#back-hub')).toBeVisible();
    await expect(page.locator('#back-hub')).toContainText('Food cost');
    await expect(page.locator('#back-hq')).toBeHidden();
    await expect(page.locator('#s2 .cost-table')).toBeVisible();
    // Reach the hub, open Trends, then pop the history entry the section pushed.
    await page.locator('#back-hub').click();
    await expect(page.locator('#s0')).toBeVisible();
    expect(await page.evaluate(() => location.hash)).toBe('');
    await page.locator('#t1').click();
    await expect(page.locator('#s1')).toBeVisible();
    await page.goBack();
    await expect(page.locator('#s0')).toBeVisible();
    await expect(page.locator('#s1')).toBeHidden();
    await shot(page, 'edge-deep-link');
  });

  test('Edge: 393px — every row is a ≥44px target and the badge never overlaps the title', async ({ page }) => {
    await login(page);
    await mock(page, TRENDS, COST);
    await openHub(page);
    await settled(page);
    const rows = await page.locator('.hub-row').evaluateAll(rs => rs.map(r => {
      const t = r.querySelector('.hub-t').getBoundingClientRect(); const b = r.querySelector('.hub-b');
      return { h: r.getBoundingClientRect().height, titleRight: t.right, badgeLeft: b && b.offsetParent ? b.getBoundingClientRect().left : Infinity };
    }));
    expect(rows).toHaveLength(2);
    for (const r of rows) { expect(r.h).toBeGreaterThanOrEqual(44); expect(r.badgeLeft).toBeGreaterThanOrEqual(r.titleRight - 0.5); }
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow).toBeLessThanOrEqual(1);
    await expect(page.locator('#s0')).toHaveAttribute('aria-label', 'Business Intelligence');
  });
});
