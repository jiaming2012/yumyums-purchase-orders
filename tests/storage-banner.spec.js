const { test, expect } = require('@playwright/test');

// ═══════════════════════════════════════════════════════════════════════════
// B-172 · storage-banner — a dead object-storage account must be explicit.
//
// The S3 client is constructed offline from static creds, so a canceled
// account (the 2026-08 DigitalOcean shape) used to start clean: no log, no
// health signal, uploads failing one at a time and stored photos rendering as
// broken images. The backend half is the HeadBucket probe surfaced as the
// "storage" field of /api/v1/health; this spec pins the frontend half — the
// launcher banner — plus the live presence of the field itself.
//
// Banner contract: visible ONLY when storage === 'unreachable'. 'unconfigured'
// is a deliberate dev/test state (upload endpoints already 503 with a named
// error) and must NOT alarm the crew; 'ok' is silence.
// ═══════════════════════════════════════════════════════════════════════════

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

async function login(page) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

function stubHealth(page, storage) {
  const body = {
    status: 'ok',
    backend_version: '0.0.0',
    frontend_version: '0.0.0',
    git_sha: 'stub',
    built_at: 'stub',
  };
  if (storage !== undefined) body.storage = storage;
  return page.route('**/api/v1/health', route =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) }),
  );
}

test('banner shows when health reports storage unreachable', async ({ page }) => {
  await login(page);
  await stubHealth(page, 'unreachable');

  await page.goto('/index.html');

  const banner = page.locator('#storage-banner');
  await expect(banner).toBeVisible();
  await expect(banner).toContainText(/storage unreachable/i);
});

test('banner stays hidden when storage is ok', async ({ page }) => {
  await login(page);
  await stubHealth(page, 'ok');

  await page.goto('/index.html');

  await expect(page.locator('#storage-banner')).toBeHidden();
});

test('banner stays hidden when storage is unconfigured', async ({ page }) => {
  await login(page);
  await stubHealth(page, 'unconfigured');

  await page.goto('/index.html');

  await expect(page.locator('#storage-banner')).toBeHidden();
});

test('banner stays hidden when health omits the storage field (old backend)', async ({ page }) => {
  await login(page);
  await stubHealth(page, undefined);

  await page.goto('/index.html');

  await expect(page.locator('#storage-banner')).toBeHidden();
});

test('live /api/v1/health carries the storage field', async ({ page }) => {
  // Unstubbed: the real backend must report one of the three states. In this
  // test stack STORAGE_* env is normally absent → 'unconfigured'; the assert
  // is membership, not a specific value, so a creds-bearing dev env stays
  // green too.
  const res = await page.request.get('/api/v1/health');
  expect(res.ok()).toBe(true);
  const health = await res.json();
  expect(['ok', 'unreachable', 'unconfigured']).toContain(health.storage);
});
