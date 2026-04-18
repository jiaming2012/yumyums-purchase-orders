const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

async function login(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email || ADMIN_EMAIL);
  await page.fill('input[type="password"]', password || ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

test('index.html shows user display name after login', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');
  const greeting = await page.locator('.greeting').textContent();
  expect(greeting.trim().length).toBeGreaterThan(0);
  expect(greeting).toMatch(/^Hi,\s+\S/);
});

test('logout button redirects to login.html', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');
  await page.click('#btn-logout');
  await page.waitForURL(url => url.pathname.includes('login'));
  expect(page.url()).toContain('login.html');
});

test('superadmin sees all tiles after restricted user logged out', async ({ page }) => {
  // Step 1: Login as admin, create a restricted team_member user (ignore 409 if already exists)
  await login(page);
  const uniqueEmail = 'tiletest-' + Date.now() + '@yumyums.kitchen';
  const inviteRes = await page.evaluate(async (email) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'TileTest', last_name: 'User', email, roles: ['team_member'] })
    });
    return res.json();
  }, uniqueEmail);
  expect(inviteRes.invite_path).toBeTruthy();
  // Accept invite to set password
  const token = inviteRes.invite_path.split('token=')[1];
  await page.evaluate(async (t) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: 'test456' })
    });
  }, token);

  // Step 2: Login fresh as superadmin and count tiles
  await login(page);
  await page.goto('/index.html');
  // Wait for grid to become visible (filterTilesByPermissions sets visibility:'')
  await page.waitForFunction(() => {
    var g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden' && g.querySelectorAll('.tile').length > 0;
  }, { timeout: 5000 });
  const adminTileCount = await page.locator('.grid .tile:visible').count();
  expect(adminTileCount).toBeGreaterThanOrEqual(4);

  // Step 3: Logout superadmin
  page.once('dialog', d => d.accept());
  await page.click('#btn-logout');
  await page.waitForURL(url => url.pathname.includes('login'));

  // Step 4: Login as restricted user — should see fewer tiles
  await login(page, uniqueEmail, 'test456');
  await page.goto('/index.html');
  await page.waitForFunction(() => {
    var g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden';
  }, { timeout: 5000 });
  await page.waitForTimeout(300);
  const restrictedTileCount = await page.locator('.grid .tile:visible').count();
  expect(restrictedTileCount).toBeLessThan(adminTileCount);

  // Step 5: Logout restricted user
  page.once('dialog', d => d.accept());
  await page.click('#btn-logout');
  await page.waitForURL(url => url.pathname.includes('login'));

  // Step 6: Login as superadmin again — MUST see all tiles again
  await login(page);
  await page.goto('/index.html');
  await page.waitForFunction(() => {
    var g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden' && g.querySelectorAll('.tile').length > 0;
  }, { timeout: 5000 });
  const adminTileCountAfter = await page.locator('.grid .tile:visible').count();
  expect(adminTileCountAfter).toBe(adminTileCount);
});

test('after logout, visiting index.html redirects to login.html (session cleared)', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');
  await page.click('#btn-logout');
  await page.waitForURL(url => url.pathname.includes('login'));
  // Now navigate directly to index.html — session should be gone
  await page.goto('/index.html');
  await page.waitForURL(url => url.pathname.includes('login'));
  expect(page.url()).toContain('login.html');
});
