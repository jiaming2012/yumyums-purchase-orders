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
  // index.html logout uses native confirm(); Playwright auto-dismisses unhandled
  // dialogs, so we must accept it explicitly for logout to proceed.
  page.on('dialog', dialog => dialog.accept());
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

test('onboarding badge only shows when user has incomplete trainings', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  // Wait for grid to be visible and permissions applied
  await page.waitForFunction(() => {
    var g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden';
  }, { timeout: 5000 });
  await page.waitForTimeout(1000); // wait for onboarding API call

  // Check if user has any onboarding trainings assigned
  const trainings = await page.evaluate(async () => {
    try {
      const res = await fetch('/api/v1/onboarding/myTrainings');
      if (!res.ok) return [];
      return await res.json();
    } catch(e) { return []; }
  });

  const tile = page.locator('#tile-onboarding');
  if (await tile.count() === 0) return; // tile not visible (permission filtered)

  if (!trainings || trainings.length === 0) {
    // No trainings assigned — badge should NOT appear
    await expect(tile.locator('.badge-warn')).toHaveCount(0);
  } else {
    const hasIncomplete = trainings.some(t => (t.progress_pct || 0) < 100);
    if (hasIncomplete) {
      await expect(tile.locator('.badge-warn')).toHaveCount(1);
    } else {
      await expect(tile.locator('.badge-warn')).toHaveCount(0);
    }
  }
});

test('onboarding badge does not reappear after visiting onboarding and returning', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  await page.waitForFunction(() => {
    var g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden';
  }, { timeout: 5000 });
  await page.waitForTimeout(1000);

  // Record whether badge exists on first load
  const tile = page.locator('#tile-onboarding');
  if (await tile.count() === 0) return; // tile not visible
  const badgeCountBefore = await tile.locator('.badge-warn').count();

  // Navigate to onboarding and back
  await page.goto('/onboarding.html');
  await page.waitForTimeout(500);
  await page.goto('/index.html');
  await page.waitForFunction(() => {
    var g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden';
  }, { timeout: 5000 });
  await page.waitForTimeout(1000);

  // Badge state should be the same as before — no phantom badge appearing
  const tile2 = page.locator('#tile-onboarding');
  if (await tile2.count() === 0) return;
  const badgeCountAfter = await tile2.locator('.badge-warn').count();
  expect(badgeCountAfter).toBe(badgeCountBefore);
  // Should never have more than 1 badge
  expect(badgeCountAfter).toBeLessThanOrEqual(1);
});

test('after logout, visiting index.html redirects to login.html (session cleared)', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');
  page.on('dialog', dialog => dialog.accept());
  await page.click('#btn-logout');
  await page.waitForURL(url => url.pathname.includes('login'));
  // Now navigate directly to index.html — session should be gone
  await page.goto('/index.html');
  await page.waitForURL(url => url.pathname.includes('login'));
  expect(page.url()).toContain('login.html');
});

// ---------------------------------------------------------------------------
// api-cache hygiene — ledger T-23 decision 57.
//
// build-sw.js:60-78 configures ONE NetworkFirst route on /\/api\// with no
// Vary, no cacheKeyWillBeUsed and no matchOptions: the cache key is the URL and
// Authorization is not in it. On a shared truck phone that makes `api-cache` a
// cross-tenant read — and nothing in app code ever cleared it.
//
// NOTE: playwright.config.js sets serviceWorkers:'block', so the SW never
// installs here and never populates `api-cache` itself. These tests therefore
// seed `api-cache` by hand — exactly the entries the NetworkFirst route would
// write — and assert on what the PAGE does about them. The window.caches API is
// available because http://localhost is a secure context.
// ---------------------------------------------------------------------------

const STALE_IDENTITY = '{"display_name":"Ghost Of User A","email":"ghost@yumyums.kitchen"}';

async function seedApiCache(page, entries) {
  await page.evaluate(async (items) => {
    const c = await caches.open('api-cache');
    for (const [url, body] of items) {
      await c.put(new Request(url), new Response(body, {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }));
    }
  }, entries);
}

test('logout deletes api-cache so the next user is not served the previous one', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');

  await seedApiCache(page, [
    ['/api/v1/me', STALE_IDENTITY],
    ['/api/v1/workflow/templates', '[{"id":1,"name":"User A private template"}]'],
  ]);
  expect(await page.evaluate(() => caches.keys())).toContain('api-cache');

  page.on('dialog', dialog => dialog.accept());
  await page.click('#btn-logout');
  await page.waitForURL(url => url.pathname.includes('login'));

  // Same origin after the redirect, so the CacheStorage is the same one.
  const remaining = await page.evaluate(() => caches.keys());
  expect(remaining).not.toContain('api-cache');
});

test('checkAuth evicts cached /api/v1/me responses instead of trusting them', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');

  // The previous session's identity, plus a leftover cache-busted probe.
  await seedApiCache(page, [
    ['/api/v1/me', STALE_IDENTITY],
    ['/api/v1/me?_=1700000000000', STALE_IDENTITY],
  ]);

  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');
  await page.waitForFunction(async () => {
    const c = await caches.open('api-cache');
    return (await c.keys()).length >= 0;
  });

  const leftovers = await page.evaluate(async () => {
    const c = await caches.open('api-cache');
    return (await c.keys())
      .map(r => new URL(r.url).pathname)
      .filter(p => p === '/api/v1/me');
  });
  expect(leftovers).toEqual([]);

  // And the stale name is never painted.
  await expect(page.locator('.greeting')).not.toContainText('Ghost Of User A');
});

test('the /api/v1/me identity probe is cache-busted so api-cache cannot answer it', async ({ page }) => {
  await login(page);

  const meRequests = [];
  page.on('request', req => {
    const u = new URL(req.url());
    if (u.pathname === '/api/v1/me') meRequests.push(u.search);
  });

  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');

  expect(meRequests.length).toBeGreaterThan(0);
  // A bare '/api/v1/me' is answerable from the URL-keyed api-cache. A per-load
  // buster can never hit it: the probe is a live answer or an error, never
  // somebody else's identity.
  for (const search of meRequests) expect(search).not.toBe('');
});

// ---------------------------------------------------------------------------
// Identity hygiene — obligations 7(a) and 7(b), ledger T-23 decision 70,
// re-specified as one mechanism by T-30 decision 112.
//
// The cross-tenant DISCLOSURE these guard is shown end-to-end, through a real
// service worker, in tests/sw-api-cache-partition.spec.js [B1-XT-01]. These two
// cover the halves of the same mechanism that live in page code and therefore
// run with the suite's normal serviceWorkers:'block'.
// ---------------------------------------------------------------------------

async function inviteTeamMember(page, tag) {
  const email = `b1idx-${tag}-${Date.now()}@yumyums.kitchen`;
  const invite = await page.evaluate(async (e) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'B1', last_name: 'Idx', email: e, roles: ['team_member'] }),
    });
    return res.json();
  }, email);
  expect(invite.invite_path).toBeTruthy();
  // 🛑 accept-invite MINTS A SESSION for the invitee — the page is logged in as
  // them after this call, not as the admin who ran it.
  await page.evaluate(async (t) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: 'test456' }),
    });
  }, invite.invite_path.split('token=')[1]);
  return { email, password: 'test456' };
}

test('signing in on login.html drops the previous user cached tile list [B1-XT-03]', async ({ page }) => {
  test.setTimeout(60000);
  // Obligation 7(b): B logs in while A's session is live. logout() never runs,
  // so login.html has to do the hygiene itself.
  await login(page);
  const member = await inviteTeamMember(page, 'tiles');
  await login(page);

  await page.goto('/index.html');
  await page.waitForFunction(() => {
    const g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden';
  }, { timeout: 10000 });

  const adminRaw = await page.evaluate(() => localStorage.getItem('hq_apps'));
  expect(adminRaw).toBeTruthy();
  const adminEnvelope = JSON.parse(adminRaw);
  // The list is identity-STAMPED now, not a bare array whose owner is unknown.
  expect(Array.isArray(adminEnvelope)).toBe(false);
  expect(adminEnvelope.uid).toBeTruthy();
  expect(Array.isArray(adminEnvelope.apps)).toBe(true);

  // B signs in. No logout anywhere in this flow.
  await login(page, member.email, member.password);
  await page.goto('/index.html');
  await page.waitForFunction(() => {
    const g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden';
  }, { timeout: 10000 });

  const afterRaw = await page.evaluate(() => localStorage.getItem('hq_apps'));
  if (afterRaw) {
    const after = JSON.parse(afterRaw);
    expect(after.uid).not.toBe(adminEnvelope.uid);
  }
});

test('the fail-closed branch paints no tiles from an unowned hq_apps [B1-XT-04]', async ({ page }) => {
  test.setTimeout(60000);
  // Obligation 7(a): index.html used to parse localStorage['hq_apps']
  // unconditionally, so the branch that CANNOT verify who is holding the phone
  // painted the previous user's launcher anyway.
  await login(page);
  await page.goto('/index.html');
  await page.waitForFunction(() => {
    const g = document.querySelector('.grid');
    return g && g.style.visibility !== 'hidden';
  }, { timeout: 10000 });

  // Plant the previous user's slug list in the legacy bare-array shape — the
  // exact value every build before this card wrote — and remove the device's
  // identity token so it cannot name its user.
  await page.evaluate(async () => {
    localStorage.setItem('hq_apps', JSON.stringify([{ slug: 'users', name: 'Users', icon: '👥' }]));
    if ('caches' in window) await caches.delete('hq-identity');
  });

  // Both identity legs fail: this is the offline truck.
  await page.route('**/api/v1/me**', route => route.abort());
  await page.goto('/index.html');
  await page.waitForTimeout(2000);

  const planted = await page.evaluate(() => localStorage.getItem('hq_apps'));
  expect(planted).toBeTruthy(); // subject set non-empty — the guard has something to refuse
  const visibility = await page.evaluate(() => document.querySelector('.grid').style.visibility);
  expect(visibility).toBe('hidden');
});
