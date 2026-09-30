const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

// states-inventory-nav.spec.js — the CLAUDE.md self-verification ritual for
// B-455. WO-1 trimmed the tab bar and folded Menu into Recipes; WO-2a (this
// table) replaces the bar with a HUB: Inventory opens on a list of
// destinations, each row carrying its own live status, and every section is
// a full page with a back link to the hub. Mockup of record:
// docs/mockups/inventory-nav-trim.html, section "Chosen".
//
// Every row of the State Enumeration Table is FORCED here at the 393×852
// phone viewport, navigated to, and screenshotted so the PNGs can be read
// back and compared to the visual contract.
//
//   Row                                   -> screenshot
//   Hub, admin (six rows until WO-2b)     -> hub-admin
//   Hub, crew (four rows; data 403s)      -> hub-crew                 (edge: real server 403 → "Status unavailable", rows still open)
//   Hub badges populated                  -> hub-badges               (10 to review · 2 below par · Drift · Synced 4 min ago)
//   Hub badges zero                       -> hub-zero                 (no badge, never "0 to review")
//   Hub row loading                       -> hub-loading              (skeleton badge VISIBLE + "Loading…")
//   Hub row error (forced 500)            -> hub-error                (edge: "Status unavailable", row opens)
//   Sync subtitle: never / running        -> hub-sync-never, hub-sync-running
//   Sync failed / cancelled / no stamp    -> hub-sync-failed           (edge: never "Synced N min ago" for a failed run; no epoch arithmetic)
//   Phone back gesture → hub              -> (no shot)                 (edge: hashchange follows the hash; forward reopens the section)
//   #tab=0 on a page without a hub        -> (no shot)                 (edge: tab.js still falls back to tab 1 elsewhere)
//   Pre-paint: hub before boot            -> hub-prepaint             (edge: no Receipts flash while /me hangs)
//   Section page + back to hub            -> section-receipts, back-to-hub
//   Deep link #tab=2 opens Stock directly -> deeplink-stock
//   Crew pastes gated #tab=5              -> deeplink-gated           (edge: lands on the hub, not a tab shell)
//   Legacy #tab=3 deep link               -> legacy-tab3              (edge: old Menu bookmark lands on By dish)
//   Hub, admin, dark scheme               -> hub-dark
//   Receipts, admin (Sync + More)         -> receipts-admin           (WO-1 rows that still hold)
//   More sheet open                       -> more-sheet
//   Deep sync modal from the sheet        -> deep-sync-from-sheet
//   Recipes › By ingredient / By dish ×3  -> recipes-*
//   Setup without the PO link             -> setup

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

// Two stock rows in the wire shape of inventory.StockRow (types.go), one
// needing reorder. The hub's "N below par" counts needs_reorder.
const STOCK = [
  { id: 'pi-1', description: 'Chicken Thighs', display_name: 'Chicken Thighs', group_name: 'Proteins', total_quantity: 4, total_spend: 212,
    avg_price: 3.2, last_purchase_date: '2026-09-27', low_threshold: 20, high_threshold: 40, level: 'low', needs_reorder: true },
  { id: 'pi-2', description: 'Canola Oil', display_name: 'Canola Oil', group_name: 'Dry Goods', total_quantity: 1, total_spend: 38,
    avg_price: 19, last_purchase_date: '2026-09-20', low_threshold: 3, high_threshold: 6, level: 'low', needs_reorder: true },
  { id: 'pi-3', description: 'Buns, Brioche', display_name: 'Brioche Buns', group_name: 'Bread', total_quantity: 96, total_spend: 96,
    avg_price: 0.5, last_purchase_date: '2026-09-28', low_threshold: 24, high_threshold: 120, level: 'medium', needs_reorder: false },
];

function json(body, status) {
  return async (route) => route.fulfill({ status: status || 200, contentType: 'application/json', body: JSON.stringify(body) });
}
async function stubDishes(page, body, status) { await page.route('**/api/v1/inventory/menu-items*', json(body, status)); }
function minutesAgoIso(min) { return new Date(Date.now() - min * 60000).toISOString(); }
function syncRow(over) {
  return Object.assign({ id: 7, started_at: minutesAgoIso(5), finished_at: minutesAgoIso(4), status: 'done', processed: 3,
    auto_created: 1, pending_review: 2, cached: 0, error: null, triggered_by: 'manual', lookback_days: 14 }, over || {});
}

async function openHub(page) {
  await page.goto('/inventory.html');
  await page.waitForLoadState('networkidle');
  await expect(page.locator('#s0')).toBeVisible();
}

test.describe('Inventory hub — B-455 WO-2a state table', () => {

  test('Hub, admin: four rows in daily-first order, back link is HQ, no hash', async ({ page }) => {
    await login(page);
    await openHub(page);
    await expect(page.locator('.hub-row')).toHaveCount(4);
    await expect(page.locator('.hub-row .hub-t')).toHaveText(['Receipts', 'Stock', 'Recipes', 'Setup']);
    await expect(page.locator('#back-hq')).toBeVisible();
    await expect(page.locator('#back-hub')).toBeHidden();
    for (const id of ['#s1', '#s2', '#s4', '#s7']) await expect(page.locator(id)).toBeHidden();
    expect(await page.evaluate(() => location.hash)).toBe('');
    // UI-R: every row is a ≥44px target and the badge never overlaps the title.
    const rows = await page.locator('.hub-row').evaluateAll(rs => rs.map(r => {
      const t = r.querySelector('.hub-t').getBoundingClientRect(); const b = r.querySelector('.hub-b');
      return { h: r.getBoundingClientRect().height, titleRight: t.right, badgeLeft: b && b.offsetParent ? b.getBoundingClientRect().left : Infinity };
    }));
    for (const r of rows) { expect(r.h).toBeGreaterThanOrEqual(44); expect(r.badgeLeft).toBeGreaterThanOrEqual(r.titleRight - 0.5); }
    await expect(page.locator('#t7')).toHaveClass(/dim/);
    // The hub is a nav of buttons, not a list with non-listitem children.
    await expect(page.locator('#s0')).toHaveAttribute('aria-label', 'Inventory');
    expect(await page.locator('#s0').getAttribute('role')).not.toBe('list');
    // Dimming Setup must not push its text below readable contrast: the title
    // and subtitle keep full opacity; only the icon and chevron fade.
    const dim = await page.locator('#t7').evaluate(r => ({
      row: getComputedStyle(r).opacity, title: getComputedStyle(r.querySelector('.hub-t')).opacity,
      icon: getComputedStyle(r.querySelector('.hub-ic')).opacity }));
    expect(+dim.row).toBe(1); expect(+dim.title).toBe(1); expect(+dim.icon).toBeLessThan(1);
    await shot(page, 'hub-admin');
  });

  test('Hub, crew: four rows; the server 403 reads as "Status unavailable" and rows still open', async ({ page }) => {
    const crew = await makeCrewUser(page, 'hub');
    await loginAs(page, crew.email, crew.password);
    await openHub(page);
    await expect(page.locator('#t5')).toHaveCount(0);
    await expect(page.locator('#t6')).toHaveCount(0);
    await expect(page.locator('#t3')).toHaveCount(0);
    await expect(page.locator('.hub-row .hub-t')).toHaveText(['Receipts', 'Stock', 'Recipes', 'Setup']);
    // No inventory grant → purchases/stock/drift 403 → the row says so, and no
    // badge pretends to be a count. The row is still a way in.
    await expect(page.locator('#hub-s1')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-s2')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-s4')).toHaveText('Status unavailable');
    for (const id of ['#hub-b1', '#hub-b2', '#hub-b4']) {
      await expect(page.locator(id)).toBeEmpty();
      await expect(page.locator(id)).not.toHaveClass(/skel/);
    }
    await shot(page, 'hub-crew');
    await page.locator('#t2').click();
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#back-hub')).toContainText('Stock');
  });

  test('Hub badges populated: 10 to review · 2 below par · Drift · Synced 4 min ago', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/stock', json(STOCK));
    await page.route('**/api/v1/inventory/recipes/drift', json({ sections: [{ kind: 'unallocated', heading: '3 unallocated', items: [] }] }));
    await page.route('**/api/v1/inventory/sync-receipts/status', json(syncRow()));
    await openHub(page);
    await page.evaluate(() => { PENDING_PURCHASES = new Array(10).fill({}); updatePendingBadge(); });
    await expect(page.locator('#hub-b1')).toHaveText('10 to review');
    await expect(page.locator('#hub-s1')).toHaveText('Synced 4 min ago');
    await expect(page.locator('#hub-b2')).toHaveText('2 below par');
    await expect(page.locator('#hub-b4')).toHaveText('Drift');
    // The row's accessible name carries the status so a screen reader hears
    // "Receipts, Synced 4 min ago, 10 to review".
    await expect(page.locator('#t1')).toHaveAccessibleName(/Receipts.*Synced 4 min ago.*10 to review/);
    await shot(page, 'hub-badges');
  });

  test('Hub badges zero: no badge at all, never "0 to review"', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/stock', json([STOCK[2]]));
    await page.route('**/api/v1/inventory/recipes/drift', json({}));
    await page.route('**/api/v1/inventory/purchases/pending', json([]));
    await page.route('**/api/v1/inventory/sync-receipts/status', json(syncRow()));
    await openHub(page);
    await expect(page.locator('#hub-s1')).toHaveText('Synced 4 min ago');
    for (const id of ['#hub-b1', '#hub-b2', '#hub-b4']) {
      await expect(page.locator(id)).toBeEmpty();
      await expect(page.locator(id)).toBeHidden();
    }
    await expect(page.locator('#hub-s2')).toHaveText('Levels and reorder suggestions');
    await expect(page.locator('#hub-s4')).toHaveText('By ingredient · By dish');
    await shot(page, 'hub-zero');
  });

  test('Hub row loading: skeleton badge and "Loading…" until the call lands', async ({ page }) => {
    await login(page);
    let release;
    const gate = new Promise(r => { release = r; });
    await page.route('**/api/v1/inventory/stock', async (route) => { await gate; await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(STOCK) }); });
    await page.route('**/api/v1/inventory/recipes/drift', async (route) => { await gate; await route.fulfill({ status: 200, contentType: 'application/json', body: '{}' }); });
    await page.goto('/inventory.html');
    await expect(page.locator('#s0')).toBeVisible();
    await expect(page.locator('#hub-b2')).toHaveClass(/skel/);
    await expect(page.locator('#hub-b2'), 'the skeleton badge is painted, not hidden by :empty').toBeVisible();
    await expect(page.locator('#hub-s2')).toHaveText('Loading…');
    await expect(page.locator('#hub-s4')).toHaveText('Loading…');
    await shot(page, 'hub-loading');
    release();
    await expect(page.locator('#hub-b2')).toHaveText('2 below par');
    await expect(page.locator('#hub-b2')).not.toHaveClass(/skel/);
    await expect(page.locator('#hub-s4')).toHaveText('By ingredient · By dish');
  });

  test('Edge: a status call that fails reads "Status unavailable" and the row still opens', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/stock', json({ error: 'boom' }, 500));
    await page.route('**/api/v1/inventory/recipes/drift', json({ error: 'boom' }, 500));
    await openHub(page);
    await expect(page.locator('#hub-s2')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-b2')).toBeEmpty();
    await expect(page.locator('#hub-s4')).toHaveText('Status unavailable');
    await expect(page.locator('#hub-b4')).toBeEmpty();
    await shot(page, 'hub-error');
    await page.locator('#t2').click();
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#stock-list')).toContainText('load stock levels');
  });

  test('Sync subtitle: "Not synced yet" with no run, "Syncing now…" while one runs', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/sync-receipts/status', json(null));
    await openHub(page);
    await expect(page.locator('#hub-s1')).toHaveText('Not synced yet');
    await shot(page, 'hub-sync-never');
    await page.unroute('**/api/v1/inventory/sync-receipts/status');
    await page.route('**/api/v1/inventory/sync-receipts/status', json(syncRow({ status: 'running', finished_at: null })));
    await page.evaluate(() => refreshSyncStatus());
    await expect(page.locator('#hub-s1')).toHaveText('Syncing now…');
    await shot(page, 'hub-sync-running');
    await page.evaluate(() => stopSyncPoll());
  });

  test('Edge: a failed or cancelled sync is named, never "Synced N min ago"', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/sync-receipts/status', json(syncRow({ status: 'failed', error: 'Mercury 502' })));
    await openHub(page);
    await expect(page.locator('#hub-s1')).toHaveText('Last sync failed');
    await shot(page, 'hub-sync-failed');
    await page.unroute('**/api/v1/inventory/sync-receipts/status');
    await page.route('**/api/v1/inventory/sync-receipts/status', json(syncRow({ status: 'cancelled' })));
    await page.evaluate(() => refreshSyncStatus());
    await expect(page.locator('#hub-s1')).toHaveText('Last sync cancelled');
    // A done row with no usable timestamp must not print epoch arithmetic.
    await page.unroute('**/api/v1/inventory/sync-receipts/status');
    await page.route('**/api/v1/inventory/sync-receipts/status', json(syncRow({ started_at: null, finished_at: null })));
    await page.evaluate(() => refreshSyncStatus());
    await expect(page.locator('#hub-s1')).toHaveText('Synced');
    await page.evaluate(() => stopSyncPoll());
  });

  test('Edge: the phone back gesture returns from a section to the hub', async ({ page }) => {
    await login(page);
    await openHub(page);
    await page.locator('#t2').click();
    await expect(page.locator('#s2')).toBeVisible();
    await page.goBack();
    await expect(page.locator('#s0')).toBeVisible();
    await expect(page.locator('#s2')).toBeHidden();
    expect(await page.evaluate(() => location.hash)).toBe('');
    await page.goForward();
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#back-hub')).toContainText('Stock');
  });

  test('Edge: #tab=0 on a page without a hub still paints its first tab', async ({ page }) => {
    // tab.js gained a home slot for the hub; every other tabbed page must keep
    // falling back to tab 1 for an out-of-range hash instead of hiding all.
    await login(page);
    await page.goto('/purchasing.html#tab=0');
    await expect(page.locator('#s1')).toBeVisible();
    await expect(page.locator('#t1')).toHaveClass(/on/);
  });

  test('Edge: the hub is painted before boot — no Receipts flash while /me hangs', async ({ page }) => {
    await login(page);
    let release;
    const gate = new Promise(r => { release = r; });
    await page.route('**/api/v1/me', async (route) => { await gate; await route.continue(); });
    await page.goto('/inventory.html');
    // tab.js (data-home="0") has run; the page script is parked on /me.
    await expect(page.locator('#s0')).toBeVisible();
    await expect(page.locator('#s1')).toBeHidden();
    await shot(page, 'hub-prepaint');
    release();
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#s0')).toBeVisible();
  });

  test('Section page: tapping Receipts opens it full-page with a back link; back returns to the hub', async ({ page }) => {
    await login(page);
    await openHub(page);
    await page.locator('#t1').click();
    await expect(page.locator('#s1')).toBeVisible();
    await expect(page.locator('#s0')).toBeHidden();
    await expect(page.locator('#back-hq')).toBeHidden();
    await expect(page.locator('#back-hub')).toBeVisible();
    await expect(page.locator('#back-hub')).toHaveText(/Inventory\s*·\s*Receipts/);
    await expect(page.locator('#t1')).toHaveClass(/on/);
    expect(await page.evaluate(() => location.hash)).toBe('#tab=1');
    await shot(page, 'section-receipts');
    await page.locator('#back-hub').click();
    await expect(page.locator('#s0')).toBeVisible();
    await expect(page.locator('#s1')).toBeHidden();
    await expect(page.locator('#back-hq')).toBeVisible();
    await expect(page.locator('#t1')).not.toHaveClass(/on/);
    expect(await page.evaluate(() => location.hash)).toBe('');
    await shot(page, 'back-to-hub');
  });

  test('Deep link #tab=2 opens Stock directly, hub hidden, back link to Inventory', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html#tab=2');
    await page.waitForSelector('#s2:visible');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#s0')).toBeHidden();
    await expect(page.locator('#back-hub')).toContainText('Stock');
    await expect(page.locator('#t2')).toHaveClass(/on/);
    await shot(page, 'deeplink-stock');
  });

  test('Edge: a crew member pasting gated #tab=5 lands on the hub, not a tab shell', async ({ page }) => {
    const crew = await makeCrewUser(page, 'gate');
    await loginAs(page, crew.email, crew.password);
    await page.goto('/inventory.html#tab=5');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#s5')).toHaveCount(0);
    await expect(page.locator('#s0')).toBeVisible();
    await expect(page.locator('#s1')).toBeHidden();
    expect(await page.evaluate(() => location.hash)).toBe('');
    await shot(page, 'deeplink-gated');
  });

  test('Edge: a legacy #tab=3 bookmark lands on Recipes › By dish', async ({ page }) => {
    await login(page);
    await stubDishes(page, DISHES);
    await page.goto('/inventory.html#tab=3');
    await page.waitForSelector('#s4:visible');
    await expect(page.locator('#t4')).toHaveClass(/on/);
    await expect(page.locator('#back-hub')).toContainText('Recipes');
    await expect(page.locator('#recipes-by-dish')).toBeVisible();
    await expect(page.locator('#menu-list')).toContainText('Sorrel Iced Tea');
    expect(await page.evaluate(() => location.hash)).toBe('#tab=4&view=dish');
    await shot(page, 'legacy-tab3');
  });
});

test.describe('Inventory hub — dark scheme', () => {
  test.use({ colorScheme: 'dark' });

  test('Hub, admin, dark: rows are cards on the dark ground and the badge keeps its warn colour', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/stock', json(STOCK));
    await openHub(page);
    await expect(page.locator('#hub-b2')).toHaveText('2 below par');
    const c = await page.evaluate(() => {
      const row = getComputedStyle(document.querySelector('.hub-row')); const body = getComputedStyle(document.body);
      const b = getComputedStyle(document.getElementById('hub-b2'));
      return { row: row.backgroundColor, body: body.backgroundColor, badge: b.backgroundColor };
    });
    expect(c.row, 'row card must contrast with the page ground').not.toBe(c.body);
    expect(c.badge, 'badge must be painted, not transparent').not.toBe('rgba(0, 0, 0, 0)');
    await shot(page, 'hub-dark');
  });
});

test.describe('Inventory navigation — WO-1 rows that still hold', () => {

  test('Receipts, admin: one control row (Sync + More), queue directly below', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html#tab=1');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('#sync-receipts-btn')).toBeVisible();
    await expect(page.locator('#sync-more-btn')).toBeVisible();
    await expect(page.locator('#deep-sync-toggle')).toHaveCount(0);
    await expect(page.locator('#s1 > #reprocess-all-btn')).toHaveCount(0);
    const row = await page.locator('.sync-row').boundingBox();
    const filter = await page.locator('#vendor-filter').boundingBox();
    expect(filter.y - (row.y + row.height), 'queue controls start within one chip-height of the sync row').toBeLessThan(90);
    await shot(page, 'receipts-admin');
  });

  test('More sheet: Deep sync… and Retry parse (all), with a labeled Close', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html#tab=1');
    await page.waitForLoadState('networkidle');
    await page.locator('#sync-more-btn').click();
    await expect(page.locator('#sync-more-overlay')).toHaveClass(/on/);
    await expect(page.locator('#deep-sync-open')).toBeVisible();
    await expect(page.locator('#reprocess-all-btn')).toBeVisible();
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
    await page.goto('/inventory.html#tab=1');
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
    await expect(card).toContainText('$95.40');
    await expect(card).toContainText('$18.00');
    await expect(card).toContainText('$113.40');
    await expect(page.locator('.menu-dish.selected')).toHaveCount(1);
    await shot(page, 'recipes-by-dish-populated');
  });

  test('Recipes › By dish, error: inline error with a retry, not a blank list', async ({ page }) => {
    await login(page);
    await stubDishes(page, { error: 'boom' }, 500);
    await page.goto('/inventory.html#tab=4');
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
    await expect(page.locator('#back-hub')).toContainText('Setup');
    await shot(page, 'setup');
  });

  test('tapping a dish does not refetch or skeleton-flash the dish list', async ({ page }) => {
    await login(page);
    let fetches = 0;
    await page.route('**/api/v1/inventory/menu-items*', async (route) => {
      fetches++;
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(DISHES) });
    });
    await page.goto('/inventory.html#tab=4&view=dish');
    await page.waitForSelector('#s4:visible');
    await expect(page.locator('#menu-list')).toContainText('Sorrel Iced Tea');
    const before = fetches;
    await page.locator('[data-menu-item-id="mi-0000-0002"]').click();
    await expect(page.locator('.menu-dish.selected')).toHaveAttribute('data-menu-item-id', 'mi-0000-0002');
    await page.locator('#rv-ingredient').click();
    await page.locator('#rv-dish').click();
    await expect(page.locator('#menu-list')).toContainText('Sorrel Iced Tea');
    expect(fetches, 'a tap and a view flip reuse the loaded list').toBe(before);
  });

  test('Escape closes the More sheet and focus returns to More', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html#tab=1');
    await page.waitForLoadState('networkidle');
    await page.locator('#sync-more-btn').click();
    await expect(page.locator('#sync-more-overlay')).toHaveClass(/on/);
    await expect(page.locator('#deep-sync-open')).toBeFocused();
    await page.keyboard.press('Escape');
    await expect(page.locator('#sync-more-overlay')).not.toHaveClass(/on/);
    await expect(page.locator('#sync-more-btn')).toBeFocused();
  });
});
