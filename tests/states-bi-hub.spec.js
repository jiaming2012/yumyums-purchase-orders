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
//   Edge: retry        -> a Retry that succeeds inside a section refreshes the hub
//                         row it came from (no stale "Status unavailable")
//   Edge: contrast     -> the muted badge keeps ≥4.5:1 in dark mode
//
// ── THE THIRD ROW (card H4 `stats-tab-ui`, run 20261002, decision 192) ──
// Campaigns joined the hub as #t3/#s3 — the campaign reports re-homed off the
// Marketing page. It is a ROW, not a redesign, which is the claim this file now
// checks: the row count moved 2 -> 3 and the Campaigns row carries the hub's own
// four states (empty / loading / error / success) by the SAME contract as
// Trends and Food cost. Its badge is a WARNING tone, not a muted reading —
// "N need a look" is work waiting, which is the Inventory-hub idiom, whereas
// "$3,058 this wk" is a reading.

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

// ── card H4: the Campaigns row's payload (GET /api/v1/bi/campaigns/overview) ──
// Only the keys the HUB reads: needs_look (the badge) and period (the subtitle).
const CAMPAIGNS = {
  period: '30d',
  funnel: { scans: 12, signups: 0, codes_sent: 0, redeemed: 5 },
  money: {
    revenue_cents: 6500, discount_cents: 1800, discount_basis: 'mixed', net_cents: 4700,
    per_dollar: 3.62, discount_implied_cents: 2000, discount_actual_cents: 800,
    discount_unknown_rows: 1, unattributed_redeemed: 1, unattributed_revenue_cents: 1500,
    avg_order_cents_with: 2166, avg_order_cents_without: null,
  },
  reconciliation: {
    matched: 3, open: 3, declined: 0, orphan_rate: 0.4, threshold: 0.1,
    orphan_rate_basis: 'unmatched_and_orphans_excl_duplicate_scan',
    orphan_numerator: 2, orphan_denominator: 5,
  },
  needs_look: { overrides: 1, orphans: 1, unmatched: 1 },
  signups_basis: 'unavailable', codes_sent_basis: 'unavailable',
};
// Nothing scanned in the period: a known window with no redemptions. No badge
// (never "0 need a look"), and the subtitle keeps the period.
const EMPTY_CAMPAIGNS = {
  period: '30d',
  funnel: { scans: 0, signups: 0, codes_sent: 0, redeemed: 0 },
  money: {
    revenue_cents: 0, discount_cents: 0, discount_basis: 'implied', net_cents: 0,
    per_dollar: null, discount_implied_cents: 0, discount_actual_cents: 0,
    discount_unknown_rows: 0, unattributed_redeemed: 0, unattributed_revenue_cents: 0,
    avg_order_cents_with: null, avg_order_cents_without: null,
  },
  reconciliation: {
    matched: 0, open: 0, declined: 0, orphan_rate: null, threshold: 0.1,
    orphan_rate_basis: 'unmatched_and_orphans_excl_duplicate_scan',
    orphan_numerator: 0, orphan_denominator: 0,
  },
  needs_look: { overrides: 0, orphans: 0, unmatched: 0 },
  signups_basis: 'unavailable', codes_sent_basis: 'unavailable',
};
const CAMPAIGNS_BY = {
  dim: 'campaign', period: '30d', rows: [], signups_basis: 'unavailable',
  totals: {
    key: 'totals', label: 'Total', scans: 0, signups: 0, redeemed: 0,
    revenue_cents: 0, discount_cents: 0, discount_basis: 'implied', net_cents: 0,
    per_dollar: null, discount_implied_cents: 0, discount_actual_cents: 0,
    discount_unknown_rows: 0, unattributed_redeemed: null, unattributed_revenue_cents: null,
    avg_order_cents_with: null, avg_order_cents_without: null,
  },
};

const json = body => route => route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
const fail = route => route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"boom"}' });
async function mock(page, trends, cost, campaigns) {
  await page.route('**/api/v1/inventory/trends', trends === 'fail' ? fail : json(trends));
  await page.route('**/api/v1/inventory/cost', cost === 'fail' ? fail : json(cost));
  // card H4: the Campaigns row is fed too, so every row on this hub is
  // deterministic. Defaulting to CAMPAIGNS keeps the pre-existing call sites
  // (which pass two arguments) meaningful instead of racing a real endpoint.
  const c = campaigns === undefined ? CAMPAIGNS : campaigns;
  await page.route('**/api/v1/bi/campaigns/overview**', c === 'fail' ? fail : json(c));
  await page.route('**/api/v1/bi/campaigns/by**', c === 'fail' ? fail : json(CAMPAIGNS_BY));
}
async function openHub(page) {
  await page.goto('/bi.html');
  await page.waitForSelector('#s0:visible');
}
async function settled(page) {
  await page.waitForFunction(() => !document.querySelector('#hub-b1.skel')
    && !document.querySelector('#hub-b2.skel') && !document.querySelector('#hub-b3.skel'));
}

test.describe('BI hub — B-455 WO-2b state table', () => {

  test('Loading: both badges are skeletons and subtitles read Loading…', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/trends', async route => { await new Promise(r => setTimeout(r, 1500)); await json(TRENDS)(route); });
    await page.route('**/api/v1/inventory/cost', async route => { await new Promise(r => setTimeout(r, 1500)); await json(COST)(route); });
    await page.route('**/api/v1/bi/campaigns/overview**', async route => { await new Promise(r => setTimeout(r, 1500)); await json(CAMPAIGNS)(route); });
    await page.route('**/api/v1/bi/campaigns/by**', async route => { await new Promise(r => setTimeout(r, 1500)); await json(CAMPAIGNS_BY)(route); });
    await openHub(page);
    await expect(page.locator('#hub-b1')).toHaveClass(/skel/);
    await expect(page.locator('#hub-b2')).toHaveClass(/skel/);
    await expect(page.locator('#hub-b3')).toHaveClass(/skel/);
    await expect(page.locator('#hub-s1')).toHaveText('Loading…');
    await expect(page.locator('#hub-s2')).toHaveText('Loading…');
    await expect(page.locator('#hub-s3')).toHaveText('Loading…');
    await shot(page, 'loading');
  });

  test('Empty: no badge on either row, descriptive subtitles, rows still open', async ({ page }) => {
    await login(page);
    await mock(page, EMPTY_TRENDS, EMPTY_COST, EMPTY_CAMPAIGNS);
    await openHub(page);
    await settled(page);
    await expect(page.locator('#hub-b1')).toBeHidden();
    await expect(page.locator('#hub-b2')).toBeHidden();
    await expect(page.locator('#hub-b3')).toBeHidden();
    await expect(page.locator('#hub-b1')).toHaveText('');
    await expect(page.locator('#hub-s1')).toHaveText('Spend by group · 12 weeks');
    await expect(page.locator('#hub-s2')).toHaveText('Margin per dish · Sep 21–27');
    // card H4: an empty period keeps the period in the subtitle and shows NO
    // badge — never "0 need a look", the same rule the other two rows follow.
    await expect(page.locator('#hub-s3')).toHaveText('Campaign money · Last 30 days');
    await shot(page, 'empty');
    await page.locator('#t1').click();
    await expect(page.locator('#s1 .tr-empty')).toBeVisible();
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
    await expect(page.locator('#hub-s2')).toHaveText('Margin per dish · Sep 21–27');
    await expect(page.locator('#hub-s2')).not.toContainText('&');
    // card H4: the Campaigns badge is the open-work count, in the WARN tone
    // (work waiting), not the muted reading tone the other two rows use.
    await expect(page.locator('#hub-b3')).toHaveText('3 need a look');
    await expect(page.locator('#hub-b3')).not.toHaveClass(/mut/);
    await expect(page.locator('#hub-s3')).toHaveText('Campaign money · Last 30 days');
    await shot(page, 'populated');
    await page.emulateMedia({ colorScheme: 'dark' });
    await shot(page, 'populated-dark');
  });

  test('Error: both calls fail → "Status unavailable", no badge, and the row still opens to the error card', async ({ page }) => {
    await login(page);
    await mock(page, 'fail', 'fail', 'fail');
    await openHub(page);
    await settled(page);
    await expect(page.locator('#hub-s1')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-s2')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-s3')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-b1')).toBeHidden();
    await expect(page.locator('#hub-b2')).toBeHidden();
    await expect(page.locator('#hub-b3')).toBeHidden();
    await shot(page, 'error');
    await page.locator('#t2').click();
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#cost-container')).toContainText('Couldn’t load food cost');
  });

  test('Edge: one call fails — that row says so, the other keeps its reading', async ({ page }) => {
    await login(page);
    await mock(page, 'fail', COST);
    await openHub(page);
    await settled(page);
    await expect(page.locator('#hub-s1')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-b1')).toBeHidden();
    await expect(page.locator('#hub-b2')).toHaveText('32% avg');
    await expect(page.locator('#hub-b3')).toHaveText('3 need a look');
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
    // card H4 (run 20261002): Campaigns joined the hub, so this is 3.
    expect(rows).toHaveLength(3);
    for (const r of rows) { expect(r.h).toBeGreaterThanOrEqual(44); expect(r.badgeLeft).toBeGreaterThanOrEqual(r.titleRight - 0.5); }
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow).toBeLessThanOrEqual(1);
    await expect(page.locator('#s0')).toHaveAttribute('aria-label', 'Business Intelligence');
    await shot(page, 'edge-393');
  });

  test('Edge: retry — a Retry that succeeds inside a section refreshes its hub row', async ({ page }) => {
    await login(page);
    let costCalls = 0;
    await page.route('**/api/v1/inventory/trends', json(TRENDS));
    await page.route('**/api/v1/bi/campaigns/overview**', json(CAMPAIGNS));
    await page.route('**/api/v1/bi/campaigns/by**', json(CAMPAIGNS_BY));
    // Boot fails; opening the section re-fetches and fails again; the Retry
    // inside the report is the call that succeeds.
    await page.route('**/api/v1/inventory/cost', route => (++costCalls <= 2 ? fail(route) : json(COST)(route)));
    await openHub(page);
    await settled(page);
    await expect(page.locator('#hub-s2')).toHaveText('Status unavailable');
    await page.locator('#t2').click();
    await expect(page.locator('#cost-container')).toContainText('Couldn’t load food cost');
    await page.locator('#cost-container button', { hasText: 'Retry' }).click();
    await expect(page.locator('#s2 .cost-table')).toBeVisible();
    await page.locator('#back-hub').click();
    await expect(page.locator('#s0')).toBeVisible();
    await expect(page.locator('#hub-b2')).toHaveText('32% avg');
    await expect(page.locator('#hub-s2')).toHaveText('Margin per dish · Sep 21–27');
    await shot(page, 'edge-retry');
  });

  test('Edge: contrast — the muted badge reads at ≥4.5:1 in dark mode', async ({ page }) => {
    await login(page);
    await mock(page, TRENDS, COST);
    await page.emulateMedia({ colorScheme: 'dark' });
    await openHub(page);
    await settled(page);
    const ratio = await page.locator('#hub-b1').evaluate(el => {
      const rgb = s => s.match(/[\d.]+/g).map(Number);
      const lum = ([r, g, b]) => { const f = c => { c /= 255; return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4); }; return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b); };
      // Composite the translucent badge background over the row card behind it.
      const card = rgb(getComputedStyle(el.closest('.hub-row')).backgroundColor);
      const bg = rgb(getComputedStyle(el).backgroundColor); const a = bg[3] === undefined ? 1 : bg[3];
      const over = [0, 1, 2].map(i => bg[i] * a + card[i] * (1 - a));
      const fg = rgb(getComputedStyle(el).color);
      const [l1, l2] = [lum(fg), lum(over)];
      return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05);
    });
    expect(ratio).toBeGreaterThanOrEqual(4.5);
  });
  // ── card H4 `stats-tab-ui` (run 20261002): the Campaigns ROW itself ───────
  // The four hub states above now cover all three rows. What is specific to the
  // new row is that it behaves like a hub DESTINATION and not like a redesign:
  // it opens a full-page section, names itself in the back crumb, and the phone
  // back gesture returns to the hub — exactly the Trends / Food cost contract.
  test('Edge: the Campaigns row opens #tab=3 with the BI · Campaigns back link, and back returns to the hub', async ({ page }) => {
    await login(page);
    await mock(page, TRENDS, COST);
    await openHub(page);
    await settled(page);
    await expect(page.locator('#t3 .hub-t')).toHaveText('Campaigns');
    await page.locator('#t3').click();
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#s0')).toBeHidden();
    expect(await page.evaluate(() => location.hash)).toBe('#tab=3');
    await expect(page.locator('#back-hub')).toContainText('Campaigns');
    await expect(page.locator('#back-hq')).toBeHidden();
    await page.goBack();
    await expect(page.locator('#s0')).toBeVisible();
    await expect(page.locator('#s3')).toBeHidden();
    await shot(page, 'edge-campaigns-row');
  });

  // The two REPORTS THAT WERE ALREADY HERE must be untouched by the new row:
  // a deep link to each still opens it, with its own crumb and its own reading.
  test('Edge: Trends and Food cost are unchanged by the third row', async ({ page }) => {
    await login(page);
    await mock(page, TRENDS, COST);
    await page.goto('/bi.html#tab=1');
    await page.waitForSelector('#s1:visible');
    await expect(page.locator('#back-hub')).toContainText('Trends');
    await expect(page.locator('#s3')).toBeHidden();
    await page.goto('/bi.html#tab=2');
    await page.waitForSelector('#s2:visible');
    await expect(page.locator('#back-hub')).toContainText('Food cost');
    await expect(page.locator('#s2 .cost-table')).toBeVisible();
    await expect(page.locator('#s3')).toBeHidden();
  });
});
