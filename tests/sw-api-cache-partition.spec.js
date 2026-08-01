const { test, expect } = require('@playwright/test');

// ---------------------------------------------------------------------------
// `api-cache` cross-tenant disclosure — the RED this card exists for.
// Ledger T-23 decision 57 (the defect), T-30 decision 112 (the mechanism).
//
// build-sw.js registers ONE NetworkFirst route on /\/api\// covering EVERY
// endpoint in all five tools. It has no Vary, no cacheKeyWillBeUsed and no
// matchOptions, so the cache key is the bare URL and the session cookie is not
// part of it. On a shared truck phone that makes `api-cache` a cross-tenant
// read: whatever user A loaded is served to user B the moment the network leg
// fails — which on a food truck is routine, not exotic.
//
// 🛑 THIS FILE IS THE ONE PLACE IN THE SUITE THAT RUNS A REAL SERVICE WORKER.
// playwright.config.js sets serviceWorkers:'block' globally, and every other
// api-cache test (tests/index.spec.js) therefore seeds the cache BY HAND and
// asserts on what the PAGE does about it. Hand-seeding cannot show the
// disclosure actually happening, because nothing reads the cache without the
// SW. The card's guard-integrity bar (B-22/B-23/B-24) requires the leak be
// shown happening with the leaked payload visible, so this file opts back in
// with test.use({serviceWorkers:'allow'}) — scoped to this file only. Each
// Playwright test gets a fresh BrowserContext, so the registration and its
// CacheStorage cannot leak into any other spec.
// ---------------------------------------------------------------------------

test.use({ serviceWorkers: 'allow' });

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// The service worker precaches 29 files on install; that plus two logins and a
// user invite does not fit the 30s default.
test.setTimeout(120000);

// login.html redirects to '/', which serves index.html. Do NOT follow with a
// goto('/index.html'): ptr.js reloads the page on `controllerchange` when the
// worker claims it, and that reload races the goto ("Navigation to
// /index.html is interrupted by another navigation to /").
async function login(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email || ADMIN_EMAIL);
  await page.fill('input[type="password"]', password || ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// A page is only routed through the SW once a worker CONTROLS it. waitForFunction
// (not evaluate) because ptr.js's controllerchange reload tears the execution
// context down underneath us; waitForFunction re-injects across navigations.
async function awaitSwControl(page) {
  await page.waitForFunction(() => !!navigator.serviceWorker.controller, null, { timeout: 30000 });
  await page.waitForTimeout(1500); // let the controllerchange reload settle
  await page.waitForFunction(() => !!navigator.serviceWorker.controller, null, { timeout: 30000 });
}

async function apiCacheKeys(page) {
  return page.evaluate(async () => {
    if (!(await caches.keys()).includes('api-cache')) return [];
    const c = await caches.open('api-cache');
    return (await c.keys()).map(r => r.url);
  });
}

// Invite + accept a fresh team_member. Returns their credentials. Must run as
// the currently-logged-in admin.
//
// 🛑 accept-invite MINTS A SESSION for the new user, so this call leaves the
// page logged in AS the invitee, not as the admin who ran it. Re-login as admin
// before the next privileged call. (Discovered as a 403 on /api/v1/users.)
async function createTeamMember(page, tag) {
  const email = `b1-${tag}-${Date.now()}@yumyums.kitchen`;
  const invite = await page.evaluate(async (e) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'B1', last_name: 'Tenant', email: e, roles: ['team_member'] }),
    });
    return res.json();
  }, email);
  expect(invite.invite_path).toBeTruthy();
  const token = invite.invite_path.split('token=')[1];
  await page.evaluate(async (t) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: 'test456' }),
    });
  }, token);
  return { email, password: 'test456' };
}

test('offline, user B is served user A cached /api/v1/users roster from api-cache [B1-XT-01]', async ({ page, context }) => {
  await login(page);
  await awaitSwControl(page);

  // A canary that exists ONLY in the roster A is entitled to read. If it comes
  // back out of B's fetch, that is the disclosure — not an inference about it.
  const canary = await createTeamMember(page, 'canary');
  await login(page);
  const tenantB = await createTeamMember(page, 'tenantb');

  // ── User A: the superadmin. Loads the team roster while online. ──────────
  await login(page);
  await awaitSwControl(page);

  const rosterA = await page.evaluate(async () => {
    const res = await fetch('/api/v1/users');
    return { status: res.status, body: await res.text() };
  });
  expect(rosterA.status).toBe(200);
  expect(rosterA.body).toContain(canary.email);

  // The NetworkFirst route wrote it. Confirm the subject set is NON-EMPTY
  // before asserting anything about it (B-22/B-23/B-24: a guard printing PASS
  // over an empty set is not evidence).
  const keysAfterA = await apiCacheKeys(page);
  expect(keysAfterA.length).toBeGreaterThan(0);
  expect(keysAfterA.some(u => u.includes('/api/v1/users'))).toBe(true);

  // ── User B logs in on the SAME device without A ever logging out. ────────
  // This is obligation 7(b): no logout() runs, so nothing in app code clears
  // api-cache, and login.html does no cache hygiene of its own.
  await login(page, tenantB.email, tenantB.password);
  await awaitSwControl(page);

  // ── The truck loses LTE. B's request fails at the network leg. ───────────
  await context.setOffline(true);
  await context.route('**/api/v1/users**', route => route.abort('internetdisconnected'));

  const servedToB = await page.evaluate(async () => {
    const res = await fetch('/api/v1/users');
    return { status: res.status, body: await res.text() };
  });

  console.log('[B1-XT-01] status served to user B:', servedToB.status);
  console.log('[B1-XT-01] body served to user B  :', servedToB.body.slice(0, 400));

  // THE DISCLOSURE: B is a team_member with no Users grant, offline, and the
  // service worker hands them the roster A loaded — every colleague's email.
  expect(servedToB.body).not.toContain(canary.email);
  expect(servedToB.status).not.toBe(200);
});

test('api-cache entries are keyed by the identity that fetched them [B1-XT-02]', async ({ page }) => {
  // The structural half. Even if every purge call site were deleted, a cache
  // key that carries the fetching identity cannot be read back under another
  // one. This asserts the key SHAPE, which is what a later diff can be checked
  // against — see merge-intent-b1-sync-cache-and-identity-hygiene.md.
  await login(page);
  await awaitSwControl(page);

  const me = await page.evaluate(async () => (await (await fetch('/api/v1/me?_probe=1')).json()));
  expect(me.id).toBeTruthy();

  // Wait for checkAuth() to have established the device identity token, then
  // start from an empty api-cache. Both make the assertion below about entries
  // written under a KNOWN identity rather than about the page-load race between
  // checkAuth and the onboarding-badge fetches.
  //
  // .catch() is deliberate: on the UNFIXED tree this token never appears, and
  // the point of this test is to fail on the cache-key assertion (which prints
  // the unpartitioned keys) rather than to time out here with nothing to show.
  await page.waitForFunction(async () => {
    if (!(await caches.keys()).includes('hq-identity')) return false;
    const c = await caches.open('hq-identity');
    return !!(await c.match('/__hq_identity'));
  }, null, { timeout: 10000 }).catch(() => {});
  await page.evaluate(async () => { await caches.delete('api-cache'); });

  await page.evaluate(async () => { await fetch('/api/v1/users'); });
  await page.evaluate(async () => { await fetch('/api/v1/me/apps'); });

  const keys = await apiCacheKeys(page);
  console.log('[B1-XT-02] api-cache keys:', JSON.stringify(keys, null, 2));
  // Non-empty subject set FIRST — otherwise the forEach below passes vacuously.
  expect(keys.length).toBeGreaterThan(0);
  for (const url of keys) {
    expect(new URL(url).searchParams.get('__hq_id')).toBe(String(me.id));
  }
});
