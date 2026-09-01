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

// B-10. `logout()` does `await purgeDeviceIdentity(); window.location.href='/login.html'`
// (index.html:271-272). The `await` is load-bearing: purgeDeviceIdentity() deletes
// `api-cache` and `hq-identity` asynchronously, and `window.location.href` tears
// this page down. If the redirect fires BEFORE the deletes complete, the previous
// user's rows can survive on the phone for the next one — the exact disclosure the
// comment above the call names.
//
// The existing "logout deletes api-cache" test proves the deletes HAPPEN, not that
// they COMPLETE before navigation: with `caches.delete` resolving in microseconds,
// dropping the `await` still lets both finish before the browser actually unloads,
// so that test stays green either way. This test forces the race: it makes
// `caches.delete` resolve SLOWLY, then asserts the redirect has NOT fired while the
// clear is still pending — i.e. the await is honored. Drop the `await` and the
// navigation happens up front, well before the slow delete resolves → this reds.
test('B-10: logout awaits the device purge before redirecting (the await is honored)', async ({ page }) => {
  await login(page);
  await page.goto('/index.html');
  await page.waitForSelector('.user-bar');

  const DELETE_DELAY_MS = 1500;
  // Slow every caches.delete() down and stamp when the LAST one actually resolves.
  // The stamp lives on window; the value is read out via a Playwright binding that
  // survives the page teardown navigation triggers.
  const deleteResolvedAt = { t: 0 };
  await page.exposeFunction('__b10_recordDeleteResolved', (t) => { deleteResolvedAt.t = t; });
  await page.evaluate((delay) => {
    const realDelete = window.caches.delete.bind(window.caches);
    window.caches.delete = (name) => new Promise((resolve) => {
      setTimeout(async () => {
        const ok = await realDelete(name);
        // Every delete resolution stamps; the last one to run wins, and it is the
        // one that must precede navigation if the await chain is honored.
        window.__b10_recordDeleteResolved(Date.now());
        resolve(ok);
      }, delay);
    });
  }, DELETE_DELAY_MS);

  // The navigation to login.html is what we time against the delete resolution.
  const navPromise = page.waitForURL((url) => url.pathname.includes('login'), { timeout: 15000 });

  page.on('dialog', (dialog) => dialog.accept());
  const clickedAt = Date.now();
  await page.click('#btn-logout');

  // Mid-purge probe: at a moment WELL BEFORE the slow delete resolves, the page
  // must still be on index.html — the redirect has not fired because logout() is
  // still awaiting purgeDeviceIdentity(). If the await were dropped, navigation
  // would already be in flight here.
  await page.waitForTimeout(DELETE_DELAY_MS / 3); // ~500ms, << 1500ms
  expect(
    page.url(),
    'redirect fired before the device purge completed — the await was dropped (B-10)',
  ).not.toContain('login.html');

  // Now let it finish and confirm it DOES eventually redirect.
  await navPromise;
  const navigatedAt = Date.now();

  // The load-bearing ordering, in wall-clock: the slow delete resolved before the
  // navigation completed, and the whole thing took at least the delete delay.
  expect(deleteResolvedAt.t, 'the slow delete never resolved').toBeGreaterThan(0);
  expect(navigatedAt - clickedAt).toBeGreaterThanOrEqual(DELETE_DELAY_MS);
  expect(deleteResolvedAt.t).toBeLessThanOrEqual(navigatedAt);
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
// service worker, in tests/sw-api-cache-partition.spec.js [B1-XT-01]. These
// cover the halves of the same mechanism that live in page code and therefore
// run with the suite's normal serviceWorkers:'block'.
//
// 🛑 WHICH TEST GATES WHICH FILE (B1 fix round). [B1-XT-03] observes device
// state AFTER index.html has run, and index.html's establishIdentity() drops
// hq_apps on its own — so [B1-XT-03] passes with login.html doing NOTHING. It is
// a guard on the index.html half and is now titled as one. login.html's own
// purge is gated by [B1-XT-06] / [B1-XT-07], which freeze the redirect target so
// the only code that can have run is login.html's.
// ---------------------------------------------------------------------------

// Everything identity-scoped this app puts on a device, read in one shot.
async function readDeviceState(page) {
  return page.evaluate(async () => {
    const names = 'caches' in window ? await caches.keys() : [];
    let identity = null;
    if (names.includes('hq-identity')) {
      const r = await (await caches.open('hq-identity')).match('/__hq_identity');
      if (r) identity = (await r.text()).trim();
    }
    let apiKeys = [];
    if (names.includes('api-cache')) {
      apiKeys = (await (await caches.open('api-cache')).keys()).map(r => r.url);
    }
    return { names, identity, apiKeys, apps: localStorage.getItem('hq_apps') };
  });
}

// Plant the PREVIOUS user's device state: their identity token, one api-cache
// entry keyed to them, and their cached tile list. `uid` is deliberately a uuid
// no account owns, so signing in as anyone is an identity CHANGE.
const FOREIGN_UID = '00000000-0000-4000-8000-0000000dead0';

async function seedPreviousUserDeviceState(page, uid) {
  await page.evaluate(async (u) => {
    localStorage.setItem('hq_apps', JSON.stringify({ uid: u, apps: [{ slug: 'users', name: 'Users', icon: '\u{1F465}' }] }));
    const id = await caches.open('hq-identity');
    await id.put('/__hq_identity', new Response(u, { headers: { 'Content-Type': 'text/plain' } }));
    const api = await caches.open('api-cache');
    await api.put('/api/v1/users?__hq_id=' + u, new Response('[{"email":"previous-user-canary@yumyums.kitchen"}]', { headers: { 'Content-Type': 'application/json' } }));
  }, uid);
}

// Make the post-sign-in destination an inert document with no scripts, so the
// ONLY code that can have touched device state is login.html's. Without this the
// destination is index.html, whose establishIdentity() does the same hygiene —
// which is exactly why [B1-XT-03] could not tell the two apart.
async function freezeRedirectTarget(page, pathname) {
  await page.route(url => url.pathname === pathname, route => route.fulfill({
    status: 200,
    contentType: 'text/html',
    body: '<!doctype html><title>frozen</title><p id="frozen-redirect-target">frozen</p>',
  }));
}

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

test('a second user signing in never inherits the previous user cached tile list [B1-XT-03]', async ({ page }) => {
  test.setTimeout(60000);
  // Obligation 7(b), END TO END: B logs in while A's session is live and
  // logout() never runs. This measures the OUTCOME across both halves — after
  // the flow, the tile list on the device belongs to B, not A.
  //
  // 🛑 It does NOT gate login.html. It was titled "signing in on login.html
  // drops…" until the B1 fix round, and the reviewer showed it passes 13/13 with
  // login.html's purge deleted entirely — because index.html's
  // establishIdentity() sees prev !== id and drops hq_apps itself. Renamed to
  // what it measures. login.html's own purge is gated by [B1-XT-06]/[B1-XT-07].
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

  // 🛑 UN-VACUUMED (B1 fix round). This was `if (afterRaw) { … }` — with
  // hq_apps absent the test asserted NOTHING and passed, the exact
  // B-22/B-23/B-24 shape this spec file is careful to avoid elsewhere. The
  // value is not optional: the `.grid` wait above only completes once
  // filterTilesByPermissions has run, and the only path to it after a purge is
  // the /api/v1/me/apps arm, which calls writeCachedApps(deviceId, apps) first.
  // So the envelope MUST exist, and it must be B's.
  const afterRaw = await page.evaluate(() => localStorage.getItem('hq_apps'));
  expect(afterRaw).toBeTruthy();
  const after = JSON.parse(afterRaw);
  expect(Array.isArray(after)).toBe(false);
  expect(after.uid).toBeTruthy();
  expect(after.uid).not.toBe(adminEnvelope.uid);
});

// ---------------------------------------------------------------------------
// BLOCKING-2 (B1 fix round). The entire login.html half — obligation 7(b) — was
// defended by ZERO tests. The reviewer removed purgeDeviceIdentity() from
// signIn() (M4) and then from BOTH call sites (M4b) and the suite went 13/13
// green each time.
//
// What ships without these: any later card simplifies login.html — dropping the
// awaited purge is an attractive "why block the redirect?" cleanup — and the
// merge-intent §2.4 cost lands with the suite still green.
// ---------------------------------------------------------------------------

test('login.html signIn() purges the previous user device state before the redirect lands [B1-XT-06]', async ({ page }) => {
  test.setTimeout(60000);
  await page.goto('/login.html');
  await seedPreviousUserDeviceState(page, FOREIGN_UID);

  // Subject set NON-EMPTY before the act — "it is gone afterwards" is a claim
  // about nothing if it was never there.
  const before = await readDeviceState(page);
  expect(before.names).toContain('api-cache');
  expect(before.names).toContain('hq-identity');
  expect(before.identity).toBe(FOREIGN_UID);
  expect(before.apiKeys.length).toBeGreaterThan(0);
  expect(before.apps).toBeTruthy();

  await freezeRedirectTarget(page, '/');

  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForSelector('#frozen-redirect-target', { timeout: 20000 });

  const after = await readDeviceState(page);
  console.log('[B1-XT-06] device state after signIn():', JSON.stringify(after));
  // index.html has NOT run — the redirect target is inert. Anything still here
  // was left here by login.html.
  expect(after.names).not.toContain('api-cache');
  expect(after.names).not.toContain('hq-identity');
  expect(after.identity).toBeNull();
  expect(after.apps).toBeNull();
});

test('login.html acceptInvite() purges the previous user device state before the redirect lands [B1-XT-07]', async ({ page }) => {
  test.setTimeout(60000);
  // The second call site. [B1-XT-06] alone reds under a mutation that removes
  // BOTH purges, but not under one that removes only this one.
  await login(page);
  const email = `b1idx-inv-${Date.now()}@yumyums.kitchen`;
  const invite = await page.evaluate(async (e) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'B1', last_name: 'Inv', email: e, roles: ['team_member'] }),
    });
    return res.json();
  }, email);
  expect(invite.invite_path).toBeTruthy();
  const token = invite.invite_path.split('token=')[1];

  await page.goto('/login.html?token=' + encodeURIComponent(token));
  await page.waitForSelector('#invite-form', { state: 'visible', timeout: 20000 });
  await seedPreviousUserDeviceState(page, FOREIGN_UID);

  const before = await readDeviceState(page);
  expect(before.names).toContain('api-cache');
  expect(before.names).toContain('hq-identity');
  expect(before.apps).toBeTruthy();

  await freezeRedirectTarget(page, '/index.html');

  await page.fill('#invite-pw', 'test4567');
  await page.fill('#invite-pw2', 'test4567');
  await page.click('#invite-form button.btn');
  await page.waitForSelector('#frozen-redirect-target', { timeout: 20000 });

  const after = await readDeviceState(page);
  console.log('[B1-XT-07] device state after acceptInvite():', JSON.stringify(after));
  expect(after.names).not.toContain('api-cache');
  expect(after.names).not.toContain('hq-identity');
  expect(after.apps).toBeNull();
});

test('the same crew member re-authenticating keeps their offline dataset [B1-XT-08]', async ({ page }) => {
  test.setTimeout(60000);
  // NON-BLOCKING-7 (B1 fix round). purgeDeviceIdentity() used to delete
  // api-cache on EVERY successful sign-in, including the same person
  // re-authenticating after a session expiry. That threw away their whole
  // offline dataset for no security gain — there is no other tenant's data on
  // the device to disclose, and establishIdentity()'s prune already covers the
  // identity-CHANGE case. On a truck that is the difference between "reload
  // works with no LTE" and "nothing works until the signal comes back".
  //
  // 🛑 This test and [B1-XT-06] bracket the comparison from both sides: break it
  // open (always purge) and this reds; break it shut (never purge) and
  // [B1-XT-06] reds.
  await login(page);
  const me = await page.evaluate(async () => (await (await fetch('/api/v1/me?_=' + Date.now(), { cache: 'no-store' })).json()));
  expect(me.id).toBeTruthy();

  await page.goto('/login.html');
  await seedPreviousUserDeviceState(page, String(me.id)); // SAME user, not foreign

  const before = await readDeviceState(page);
  expect(before.identity).toBe(String(me.id));
  expect(before.apiKeys.length).toBeGreaterThan(0);

  await freezeRedirectTarget(page, '/');

  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForSelector('#frozen-redirect-target', { timeout: 20000 });

  const after = await readDeviceState(page);
  console.log('[B1-XT-08] device state after same-user re-login:', JSON.stringify(after));
  expect(after.identity).toBe(String(me.id));
  expect(after.apiKeys).toEqual(before.apiKeys);
  expect(after.apps).toBe(before.apps);
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
