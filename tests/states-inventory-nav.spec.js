const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

// states-inventory-nav.spec.js — the CLAUDE.md self-verification ritual for
// B-455 / WO-1: the Inventory tab bar trimmed from seven tabs to six, Menu
// folded into Recipes as the "By dish" view, the three stacked sync controls
// collapsed into one row (Sync Receipts + an admin-only More sheet), and the
// Purchase Orders link dropped from Setup. Mockup of record:
// docs/mockups/inventory-nav-trim.html (option A).
//
// Every row of the State Enumeration Table is FORCED here at the 393×852
// phone viewport, navigated to, and screenshotted so the PNGs can be read
// back and compared to the visual contract.
//
//   Row                              -> screenshot
//   Receipts, admin (Sync + More)    -> receipts-admin
//   Receipts, crew (Sync only)       -> receipts-crew            (edge: gated user sees FOUR tabs)
//   More sheet open                  -> more-sheet
//   Deep sync modal from the sheet   -> deep-sync-from-sheet     (edge: the range picker is still reachable)
//   Recipes › By ingredient          -> recipes-by-ingredient
//   Recipes › By dish, empty         -> recipes-by-dish-empty
//   Recipes › By dish, populated     -> recipes-by-dish-populated (dish selected, summary above)
//   Recipes › By dish, error         -> recipes-by-dish-error    (500 route → inline retry)
//   Setup without the PO link        -> setup
//   Legacy #tab=3 deep link          -> legacy-tab3              (edge: old Menu bookmark lands on By dish)
//   Tab bar with a 10-count badge    -> tab-bar-badge            (edge: no label wraps at 393px)
//   Receipts, admin, dark scheme     -> receipts-admin-dark      (the operator's own screenshot was dark)

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const GATE_PASSWORD = 'test456';

const SHOT_DIR = path.join(__dirname, '..', 'test-results', 'states-inventory-nav');
fs.mkdirSync(SHOT_DIR, { recursive: true });

test.use({ viewport: { width: 393, height: 852 } });

async function shot(page, name) {
  await page.screenshot({ path: path.join(SHOT_DIR, name + '.png'), fullPage: true });
}

async function loginAs(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

async function login(page) { await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD); }

// A team_member with no inventory grants — the crew. Mirrors makeGatedUser in
// states-cost.spec.js (grants appended, never replaced).
async function makeCrewUser(page, tag) {
  const email = `nav-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  await login(page);
  const invite = await page.evaluate(async (em) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Nav', last_name: 'Crew', email: em, roles: ['team_member'] }),
    });
    return res.json();
  }, email);
  const token = (invite.invite_path || '').split('token=')[1];
  expect(token, 'invite token').toBeTruthy();
  await page.evaluate(async ([t, pw]) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: pw }),
    });
  }, [token, GATE_PASSWORD]);
  return { email, password: GATE_PASSWORD };
}

const DISHES = [
  { id: 'mi-0000-0001', master_id: 'M-1', name: 'Jerk Chicken Bowl', menu: 'Main', menu_group: 'Bowls', menu_subgroup: null,
    last_seen: '2026-09-28', created_at: '2026-09-01T12:00:00Z', units_sold_this_week: 142, gross_this_week: 1846.0 },
  { id: 'mi-0000-0002', master_id: 'M-2', name: 'Loaded Plantain Fries', menu: 'Main', menu_group: 'Sides', menu_subgroup: 'Hot',
    last_seen: '2026-09-28', created_at: '2026-09-01T12:00:00Z', units_sold_this_week: 88, gross_this_week: 616.0 },
  { id: 'mi-0000-0003', master_id: 'M-3', name: 'Sorrel Iced Tea', menu: 'Drinks', menu_group: 'Drinks', menu_subgroup: null,
    last_seen: '2026-09-27', created_at: '2026-09-01T12:00:00Z', units_sold_this_week: 201, gross_this_week: 804.0 },
];

async function stubDishes(page, body, status) {
  await page.route('**/api/v1/inventory/menu-items*', async (route) => {
    await route.fulfill({ status: status || 200, contentType: 'application/json', body: JSON.stringify(body) });
  });
}

test.describe('Inventory navigation — B-455 WO-1 state table', () => {

  test('Receipts, admin: one control row (Sync + More), queue directly below', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#sync-receipts-btn')).toBeVisible();
    await expect(page.locator('#sync-more-btn')).toBeVisible();
    // Contract: nothing but the chip may sit between the control row and the
    // vendor filter — the deep-sync checkbox and the full-width Retry Parse
    // button are gone.
    await expect(page.locator('#deep-sync-toggle')).toHaveCount(0);
    await expect(page.locator('#s1 > #reprocess-all-btn')).toHaveCount(0);
    const row = await page.locator('.sync-row').boundingBox();
    const filter = await page.locator('#vendor-filter').boundingBox();
    expect(filter.y - (row.y + row.height), 'queue controls start within one chip-height of the sync row').toBeLessThan(90);
    await shot(page, 'receipts-admin');
  });

  test('Receipts, crew: Sync only, and the bar shows exactly four tabs', async ({ page }) => {
    const crew = await makeCrewUser(page, 'receipts');
    await loginAs(page, crew.email, crew.password);
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#sync-receipts-btn')).toBeVisible();
    await expect(page.locator('#sync-more-btn')).toBeHidden();
    // The crew never had Trends/Cost; with Menu gone their bar is the four-tab
    // bar from the mockup: Receipts / Stock / Recipes / Setup.
    await expect(page.locator('#t5')).toHaveCount(0);
    await expect(page.locator('#t6')).toHaveCount(0);
    await expect(page.locator('#t3')).toHaveCount(0);
    await expect(page.locator('.tabs button')).toHaveCount(4);
    await expect(page.locator('.tabs button')).toHaveText(['Receipts', 'Stock', 'Recipes', 'Setup']);
    await shot(page, 'receipts-crew');
  });

  test('More sheet: Deep sync… and Retry parse (all), with a labeled Close', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    await page.locator('#sync-more-btn').click();
    await expect(page.locator('#sync-more-overlay')).toHaveClass(/on/);
    await expect(page.locator('#deep-sync-open')).toBeVisible();
    await expect(page.locator('#reprocess-all-btn')).toBeVisible();
    // UI-R2: the way out is labeled and ≥44px.
    const close = page.locator('#sync-more-close');
    await expect(close).toHaveText('Close');
    const box = await close.boundingBox();
    expect(box.height).toBeGreaterThanOrEqual(44);
    for (const id of ['#deep-sync-open', '#reprocess-all-btn']) {
      const b = await page.locator(id).boundingBox();
      expect(b.height, id + ' touch target').toBeGreaterThanOrEqual(44);
    }
    await shot(page, 'more-sheet');
  });

  test('Edge: the deep-sync range picker is still reachable, from the sheet', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    await page.locator('#sync-more-btn').click();
    await page.locator('#deep-sync-open').click();
    await expect(page.locator('#sync-more-overlay')).not.toHaveClass(/on/);
    await expect(page.locator('#deep-sync-overlay')).toHaveClass(/on/);
    await expect(page.locator('#deep-from')).not.toHaveValue('');
    await shot(page, 'deep-sync-from-sheet');
  });

  test('Recipes › By ingredient is the default view; the summary card is not on it', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html#tab=4');
    await page.waitForSelector('#s4:visible');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#rv-ingredient')).toHaveClass(/on/);
    await expect(page.locator('#recipes-by-ingredient')).toBeVisible();
    await expect(page.locator('#recipes-by-dish')).toBeHidden();
    await shot(page, 'recipes-by-ingredient');
  });

  test('Recipes › By dish, empty: "No menu items" under the placeholder summary', async ({ page }) => {
    await login(page);
    await stubDishes(page, []);
    await page.goto('/inventory.html#tab=4');
    // tab.js paints #s4 before the page script has booted; a click that lands
    // in that gap finds no listener. Wait for boot (its API calls) to settle.
    await page.waitForLoadState('networkidle');
    await page.locator('#rv-dish').click();
    await expect(page.locator('#recipes-by-dish')).toBeVisible();
    await expect(page.locator('#menu-list')).toContainText('No menu items');
    await expect(page.locator('#recipes-summary-card')).toContainText('Tap a dish to see its ingredient cost');
    expect(await page.evaluate(() => location.hash)).toBe('#tab=4&view=dish');
    await shot(page, 'recipes-by-dish-empty');
  });

  test('Recipes › By dish, populated: tapping a dish selects it and its cost renders above', async ({ page }) => {
    await login(page);
    await stubDishes(page, DISHES);
    await page.goto('/inventory.html#tab=4&view=dish');
    await page.waitForSelector('#s4:visible');
    await expect(page.locator('#recipes-by-dish')).toBeVisible();
    await expect(page.locator('#menu-list')).toContainText('Jerk Chicken Bowl');
    await expect(page.locator('#menu-list')).toContainText('142');
    // Give the bowl two ingredients so the summary has real math to show.
    await page.evaluate(() => {
      RECIPES_DATA = [
        { purchase_item_id: 'pi-a', description: 'Chicken Thighs', last_week_spend: 212.0, sum_pct: 45,
          recipes: [{ id: 'r-a', menu_item_id: 'mi-0000-0001', menu_item_name: 'Jerk Chicken Bowl', menu_group: 'Bowls', usage_pct: 45 }] },
        { purchase_item_id: 'pi-b', description: 'Jasmine Rice', last_week_spend: 60.0, sum_pct: 30,
          recipes: [{ id: 'r-b', menu_item_id: 'mi-0000-0001', menu_item_name: 'Jerk Chicken Bowl', menu_group: 'Bowls', usage_pct: 30 }] },
      ];
    });
    await page.locator('[data-menu-item-id="mi-0000-0001"]').click();
    const card = page.locator('#recipes-summary-card');
    await expect(card).toContainText('Jerk Chicken Bowl');
    await expect(card).toContainText('$95.40'); // 212 × 45%
    await expect(card).toContainText('$18.00'); // 60 × 30%
    await expect(card).toContainText('$113.40');
    await expect(page.locator('.menu-dish.selected')).toHaveCount(1);
    await shot(page, 'recipes-by-dish-populated');
  });

  test('Recipes › By dish, error: inline error with a retry, not a blank list', async ({ page }) => {
    await login(page);
    await stubDishes(page, { error: 'boom' }, 500);
    await page.goto('/inventory.html#tab=4');
    // tab.js paints #s4 before the page script has booted; a click that lands
    // in that gap finds no listener. Wait for boot (its API calls) to settle.
    await page.waitForLoadState('networkidle');
    await page.locator('#rv-dish').click();
    await expect(page.locator('#menu-list')).toContainText('load menu items');
    await shot(page, 'recipes-by-dish-error');
  });

  test('Setup: Items / Vendors, no link out to Purchase Orders', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html#tab=7');
    await page.waitForSelector('#s7:visible');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#s7 a[href="purchasing.html"]')).toHaveCount(0);
    await expect(page.locator('#st1')).toHaveText('Items');
    await expect(page.locator('#st2')).toHaveText('Vendors');
    await shot(page, 'setup');
  });

  test('Edge: a legacy #tab=3 bookmark lands on Recipes › By dish', async ({ page }) => {
    await login(page);
    await stubDishes(page, DISHES);
    await page.goto('/inventory.html#tab=3');
    await page.waitForSelector('#s4:visible');
    await expect(page.locator('#t4')).toHaveClass(/on/);
    await expect(page.locator('#recipes-by-dish')).toBeVisible();
    await expect(page.locator('#menu-list')).toContainText('Sorrel Iced Tea');
    expect(await page.evaluate(() => location.hash)).toBe('#tab=4&view=dish');
    await shot(page, 'legacy-tab3');
  });

  test('Edge: the tab bar with a 10-count badge stays on one line at 393px', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => { PENDING_PURCHASES = new Array(10).fill({}); updateHistoryTabLabel(); });
    await expect(page.locator('#t1')).toContainText('Receipts');
    await expect(page.locator('#t1 .tab-badge')).toHaveText('10');
    await expect(page.locator('.tabs button')).toHaveCount(6);
    // Count real line boxes of each label's text node (the corner badge is
    // absolutely positioned and must not count): one line each, or it wrapped.
    const lines = await page.locator('.tabs button').evaluateAll(bs => bs.map(b => {
      const r = document.createRange(); r.selectNode(b.firstChild);
      return new Set(Array.from(r.getClientRects()).map(x => Math.round(x.top))).size;
    }));
    expect(lines, 'every tab label is a single line box').toEqual(lines.map(() => 1));
    await shot(page, 'tab-bar-badge');
  });
});

test.describe('Inventory navigation — dark scheme', () => {
  test.use({ colorScheme: 'dark' });

  test('Receipts, admin, dark: the More button reads as a secondary control', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#sync-more-btn')).toBeVisible();
    const colors = await page.evaluate(() => {
      const a = getComputedStyle(document.getElementById('sync-receipts-btn'));
      const b = getComputedStyle(document.getElementById('sync-more-btn'));
      return { primary: a.backgroundColor, more: b.backgroundColor };
    });
    expect(colors.more, 'More must not share the primary green').not.toBe(colors.primary);
    await shot(page, 'receipts-admin-dark');
  });
});
