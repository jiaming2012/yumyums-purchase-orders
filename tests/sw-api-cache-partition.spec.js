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
  // The structural half. This asserts the key SHAPE, which is what a later diff
  // can be checked against — see merge-intent-b1-sync-cache-and-identity-hygiene.md.
  //
  // 🛑 WHAT THIS DEFENDS, honestly. This comment used to claim a key carrying
  // the fetching identity "cannot be read back under another one". That is
  // FALSE and the B1 fix round's reviewer proved it by execution, three ways:
  //   (i)   the victim's uuid is IN the key, and `caches.open('api-cache')
  //         .keys()` enumerates it to any script on the page;
  //   (ii)  `caches.open('api-cache').match(<that key>)` returns another
  //         partition's full body directly — CacheStorage is same-origin and
  //         JS-readable, so the partition is not a boundary against page script
  //         at ALL;
  //   (iii) page JS can `put()` a victim uuid into `/__hq_identity` and the
  //         worker will then serve that partition at 200.
  // `workflows.html` loads SortableJS from a CDN, so "a script on the page" is
  // not hypothetical in this app.
  //
  // What the partition DOES defend is the shared-device, honest-user case: user
  // B, using the app as intended on a phone user A held earlier, is never
  // SERVED A's rows by the offline fallback. That is obligation 7 and it is the
  // whole of it. An XSS or a hostile dependency is a different threat with a
  // different mitigation (CSP, self-hosting the CDN) and this mechanism does not
  // claim it. Merge-intent §2.4 states the same limit; keep the two in step.
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

  // 🛑 `fetch()` does NOT await the cachePut. Reading the keys straight after
  // the two fetches races the second write, and in EVERY observed green run of
  // this test exactly ONE key was present — the `keys.length > 0` floor held,
  // so the test was not vacuous, but it was asserting over half the subject set
  // it sets up and one scheduling change away from asserting over none. Wait for
  // BOTH writes to be observable before reading. (B1 fix round, NON-BLOCKING-8.)
  //
  // This waits on PRESENCE, never on partitioning: on a tree with the partition
  // deleted both keys still land (unpartitioned), so this still falls through to
  // the assertion below, which prints them. It cannot convert a red into a
  // timeout with nothing to show.
  await page.waitForFunction(async () => {
    if (!(await caches.keys()).includes('api-cache')) return false;
    const c = await caches.open('api-cache');
    const urls = (await c.keys()).map(r => r.url);
    return urls.some(u => u.includes('/api/v1/users')) && urls.some(u => u.includes('/api/v1/me/apps'));
  }, null, { timeout: 20000 });

  const keys = await apiCacheKeys(page);
  console.log('[B1-XT-02] api-cache keys:', JSON.stringify(keys, null, 2));
  // Non-empty subject set FIRST — otherwise the loop below passes vacuously.
  // Both endpoints, not "at least one": the count is the subject-set floor.
  expect(keys.length).toBeGreaterThanOrEqual(2);
  expect(keys.filter(u => u.includes('/api/v1/users')).length).toBe(1);
  expect(keys.filter(u => u.includes('/api/v1/me/apps')).length).toBe(1);
  for (const url of keys) {
    expect(new URL(url).searchParams.get('__hq_id')).toBe(String(me.id));
  }
});

// ---------------------------------------------------------------------------
// BLOCKING-1 (B1 fix round). `cacheWillUpdate` was a hook NO test defended: the
// reviewer deleted it whole and the suite went 13/13 green.
//
// [B1-XT-02] cannot catch it STRUCTURALLY — it waits for `hq-identity` to
// appear, THEN clears `api-cache`, THEN fetches, so by construction every write
// it observes happens with an identity already present. The boot window the hook
// exists to close is never entered there. This test enters it.
//
// What ships without this guard: on any device between page load and the
// `/api/v1/me` answer — and PERMANENTLY on any device where `index.html` has
// never run — API responses are written under `__hq_id=anon`, a partition every
// subsequent user of that phone shares. A smaller instance of the exact bug this
// card exists to close, and the diff would be green.
// ---------------------------------------------------------------------------
test('with no device identity the worker writes NOTHING to api-cache [B1-XT-05]', async ({ page }) => {
  await login(page);
  await awaitSwControl(page);

  // ── POSITIVE CONTROL, first and in the same test. ────────────────────────
  // `fetch()` does not await the cachePut, so "nothing is in the cache" a
  // moment after a fetch is NOT by itself evidence of anything — it is equally
  // consistent with a write that simply has not landed. This leg measures that
  // a write IS observable, with identity present, well inside the window the
  // negative leg then waits out. Without it the assertion below is the vacuous
  // shape B-22/B-23/B-24 name.
  await page.evaluate(async () => { await caches.delete('api-cache'); });
  await page.evaluate(async () => { await fetch('/api/v1/users'); });
  const control = Date.now();
  await page.waitForFunction(async () => {
    if (!(await caches.keys()).includes('api-cache')) return false;
    const c = await caches.open('api-cache');
    return (await c.keys()).some(r => r.url.includes('/api/v1/users'));
  }, null, { timeout: 20000 });
  console.log('[B1-XT-05] control: write observable after', Date.now() - control, 'ms');

  // ── THE BOOT WINDOW: the device cannot name its user. ────────────────────
  await page.evaluate(async () => {
    await caches.delete('hq-identity');
    await caches.delete('api-cache');
  });
  await page.evaluate(async () => { await fetch('/api/v1/users'); });
  await page.waitForTimeout(6000); // >> the control's observed latency

  const keys = await apiCacheKeys(page);
  console.log('[B1-XT-05] api-cache keys with no identity:', JSON.stringify(keys));

  // The window must actually have been ENTERED. If anything re-established the
  // token mid-test, everything below would be a statement about the wrong
  // state — so prove the precondition rather than assume it.
  const identityGone = await page.evaluate(async () => {
    if (!(await caches.keys()).includes('hq-identity')) return true;
    const r = await (await caches.open('hq-identity')).match('/__hq_identity');
    return !r || !(await r.text()).trim();
  });
  expect(identityGone).toBe(true);

  // No identity ⇒ `cacheKeyWillBeUsed` would key every write `__hq_id=anon`.
  // The presence of ANY such key is the shared-partition bug arriving.
  expect(keys.filter(u => new URL(u).searchParams.get('__hq_id') === 'anon')).toEqual([]);
  expect(keys.some(u => u.includes('/api/v1/users'))).toBe(false);
});
