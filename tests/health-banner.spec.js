const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

async function login(page) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL((url) => !url.pathname.includes('login'));
}

// health stubs whatever /api/v1/health should answer. `null` means the request
// itself fails — the case the old inline banner swallowed in a bare catch{}.
async function stubHealth(page, body) {
  await page.route(/\/api\/v1\/health/, async (route) => {
    if (body === null) return route.abort('failed');
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
  });
}

const OK = { status: 'ok', storage: 'ok', sync_substrate: 'ok', toast_sync: { status: 'ok' } };

test.describe('Server-health banners', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('a healthy server shows nothing at all', async ({ page }) => {
    await stubHealth(page, OK);
    await page.goto('/index.html');
    await expect(page.locator('#health-banners')).toBeHidden();
  });

  test('an unreachable server is SAID, not swallowed', async ({ page }) => {
    // The regression this guards: the old banner ended in catch(e){}, so the
    // single most important failure produced silence.
    await stubHealth(page, null);
    await page.goto('/index.html');
    const box = page.locator('#health-banners');
    await expect(box).toBeVisible();
    await expect(box).toContainText("Can't reach the HQ server");
  });

  test('unreachable storage names the consequence, not the subsystem', async ({ page }) => {
    await stubHealth(page, Object.assign({}, OK, { storage: 'unreachable' }));
    await page.goto('/index.html');
    await expect(page.locator('#health-banners')).toContainText('uploads will fail');
  });

  test('a failing Toast sync is surfaced — it used to be in health and shown nowhere', async ({ page }) => {
    await stubHealth(page, Object.assign({}, OK, { toast_sync: { status: 'failing' } }));
    await page.goto('/index.html');
    await expect(page.locator('#health-banners')).toContainText('Toast sales sync is failing');
  });

  test('an unreachable sync substrate is surfaced', async ({ page }) => {
    await stubHealth(page, Object.assign({}, OK, { sync_substrate: 'unreachable' }));
    await page.goto('/index.html');
    await expect(page.locator('#health-banners')).toContainText('Scan & redeem is offline');
  });

  test('"unconfigured" is silence, not a warning', async ({ page }) => {
    // The normal state outside a sync/storage deploy. Warning about it on every
    // load is how a crew learns to ignore the banner that matters.
    await stubHealth(page, Object.assign({}, OK, { storage: 'unconfigured', sync_substrate: 'unconfigured' }));
    await page.goto('/index.html');
    await expect(page.locator('#health-banners')).toBeHidden();
  });

  test('the admin prompt appears ONCE however many warnings, and carries a diagnostic', async ({ page }) => {
    await stubHealth(page, { status: 'ok', storage: 'unreachable', sync_substrate: 'unreachable', toast_sync: { status: 'failing' } });
    await page.goto('/index.html');
    await expect(page.locator('.warnbanner')).toHaveCount(3);
    await expect(page.locator('.warnbanner-admin')).toHaveCount(1);
    const diag = page.locator('.warnbanner-diag');
    // Names the failing checks so the report that reaches an administrator is
    // actionable rather than "the app is weird".
    await expect(diag).toContainText('storage: unreachable');
    await expect(diag).toContainText('toast_sync: failing');
    await expect(diag).toContainText('sync_substrate: unreachable');
  });

  test('the health fetch bypasses the service-worker cache', async ({ page }) => {
    // The desktop/phone split: /api/ matches build-sw.js's NetworkFirst rule,
    // so a flaky phone was served a stale `storage: ok` from api-cache and
    // showed no banner while a desktop on the same account did. A unique query
    // string keeps the request from matching that rule at all.
    const urls = [];
    await page.route(/\/api\/v1\/health/, async (route) => {
      urls.push(route.request().url());
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(OK) });
    });
    await page.goto('/index.html');
    // index.html makes TWO health calls: the version line's (which still goes
    // through the SW's NetworkFirst rule — recorded as best-effort in
    // build-sw.js and deliberately not changed here) and the banner's. Only
    // the banner's must be uncacheable, so assert one exists rather than
    // assuming ordering.
    await expect.poll(() => urls.filter((u) => /\?_=\d+/.test(u)).length).toBeGreaterThan(0);
  });

  test('a banner clears without a reload once the server recovers', async ({ page }) => {
    let broken = true;
    await page.route(/\/api\/v1\/health/, async (route) => {
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify(Object.assign({}, OK, { storage: broken ? 'unreachable' : 'ok' })),
      });
    });
    await page.goto('/index.html');
    await expect(page.locator('#health-banners')).toContainText('uploads will fail');
    broken = false;
    // Re-check fires on focus as well as on the 60s timer, which is the pattern
    // a phone actually follows (backgrounded, then reopened).
    await page.evaluate(() => document.dispatchEvent(new Event('visibilitychange')));
    await expect(page.locator('#health-banners')).toBeHidden();
  });

  test('each tool page warns only about what its user can act on', async ({ page }) => {
    await stubHealth(page, { status: 'ok', storage: 'unreachable', sync_substrate: 'ok', toast_sync: { status: 'failing' } });
    // Inventory reads Toast for COGS; it does not upload photos.
    await page.goto('/inventory.html');
    await expect(page.locator('#health-banners')).toContainText('Toast sales sync is failing');
    await expect(page.locator('#health-banners')).not.toContainText('uploads will fail');
    // Onboarding captures photos; a Toast warning there is noise.
    await page.goto('/onboarding.html');
    await expect(page.locator('#health-banners')).toContainText('uploads will fail');
    await expect(page.locator('#health-banners')).not.toContainText('Toast sales sync');
  });
});
