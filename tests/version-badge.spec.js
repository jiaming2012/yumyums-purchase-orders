const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

// ═══════════════════════════════════════════════════════════════════════════
// A6 · app-version-badge — ledger T-33 decision 133, OKR D-KR2b.
//
// 🛑 THIS SPEC'S JOB IS TO FAIL IF THE BADGE IS EVER REROUTED TO THE API.
//
// The version line exists to catch ONE defect: a phone running a STALE bundle
// while the server is fine (the T-21d class). `version.json` is precached and
// served cache-first, so it reports what THIS DEVICE has — the only value
// capable of being stale. `/api/v1/health`'s `frontend_version` is always
// current, so a badge fed from it would print the right number on a frozen
// phone and HIDE the very defect it exists to reveal.
//
// A test that merely asserts "some version renders" would pass on that wrong
// implementation. So every test here serves the two sources DIFFERENT values
// and pins the badge to the FILE's value:
//
//   version.json      -> 9.9.9-from-file   (what the badge must show)
//   /api/v1/health    -> 0.0.1-from-api    (what it must never show as ITS value)
//
// Reroute the badge to /api/v1/health and tests 1, 2 and 3 all red.
//
// 🛑 ON `serviceWorkers: 'block'` (playwright.config.js, B-15 — do not change
// that line). These tests CANNOT exercise a real Workbox precache; `version.json`
// is served over plain HTTP here. So the guarantee is split in two, and both
// halves are asserted:
//   (a) THIS spec pins the URL the page reads its version FROM, and pins the
//       absence of any API fallback when that URL is unavailable.
//   (b) test 5 (below) pins that `version.json` is in the committed precache
//       manifest in `sw.js` — which is what makes (a)'s URL cache-first, i.e.
//       device-local and staleable, at runtime.
// Neither half alone is the property; together they are.
// ═══════════════════════════════════════════════════════════════════════════

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

const FILE_VERSION = '9.9.9-from-file';
const API_VERSION = '0.0.1-from-api';

// Parse the Workbox precache MANIFEST — `url:"…"` entries — rather than
// grepping sw.js for the string. Same idiom as tests/sw-manifest.spec.js:17-22
// (duplicated deliberately: requiring another *.spec.js would register ITS
// tests into this file's suite).
//
// 🛑 WHY THE PARSE AND NOT A STRING MATCH. This half of the spec is what covers
// the gap `serviceWorkers: 'block'` leaves — it is the only assertion that
// `version.json` is served CACHE-FIRST, i.e. device-local and therefore
// staleable, which is the whole premise of the badge. A regex like
// /["']version\.json["']/ is satisfied by ANY mention of the file in sw.js,
// including a `runtimeCaching` `urlPattern` — a configuration in which the file
// sits OUTSIDE the precache and the guarantee is silently gone while this test
// stays green. So it must be the strong half, not the loose one.
function precachedUrls(swSource) {
  const urls = [];
  const re = /url:"([^"]+)"/g;
  let m;
  while ((m = re.exec(swSource)) !== null) urls.push(m[1]);
  return urls;
}

async function login(page) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// Serve /api/v1/health with a frontend_version that is DELIBERATELY not the
// file's. Anything the page shows carrying this string came from the API.
async function stubHealth(page, frontendVersion) {
  await page.route('**/api/v1/health', async route => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'ok',
        backend_version: '0.0.0',
        frontend_version: frontendVersion,
        git_sha: 'stub',
        built_at: 'stub',
      }),
    });
  });
}

test('version line shows the value from version.json, NOT the value from /api/v1/health', async ({ page }) => {
  await login(page);

  const requested = [];
  page.on('request', r => requested.push(r.url()));

  await page.route('**/version.json', route =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ frontend: FILE_VERSION }),
    }),
  );
  await stubHealth(page, API_VERSION);

  await page.goto('/index.html');

  const line = page.locator('#version-line');
  await expect(line).toBeVisible();

  // The badge's OWN value — the primary — is the file's.
  await expect(page.locator('#version-cached')).toHaveText('v' + FILE_VERSION);

  // And the page really did read the file.
  expect(requested.some(u => u.endsWith('/version.json'))).toBe(true);

  // The API's value may appear only as the clearly-labelled COMPARISON, never
  // as the badge's own value.
  const cached = await page.locator('#version-cached').textContent();
  expect(cached).not.toContain(API_VERSION);
  expect(await line.getAttribute('data-version')).toBe(FILE_VERSION);
});

test('version line reports UNKNOWN rather than falling back to /api/v1/health when version.json is unavailable', async ({ page }) => {
  // This is the test that makes the reroute impossible to sneak past: with the
  // file gone there IS no legitimate version to show, so an implementation that
  // has an API path at all will print API_VERSION here and red.
  await login(page);

  await page.route('**/version.json', route => route.abort());
  await stubHealth(page, API_VERSION);

  await page.goto('/index.html');

  const line = page.locator('#version-line');
  await expect(line).toBeVisible();
  await expect(line).toHaveAttribute('data-state', 'unknown');

  const text = await line.textContent();
  expect(text).not.toContain(API_VERSION);
  expect(await line.getAttribute('data-version')).toBeNull();
});

test('version line flags this device as STALE when the cached bundle differs from the server', async ({ page }) => {
  await login(page);

  await page.route('**/version.json', route =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ frontend: FILE_VERSION }),
    }),
  );
  await stubHealth(page, API_VERSION);

  await page.goto('/index.html');

  const line = page.locator('#version-line');
  await expect(line).toHaveAttribute('data-state', 'stale');
  // Both numbers on screen: the device's (primary) and the server's (comparison).
  await expect(page.locator('#version-cached')).toHaveText('v' + FILE_VERSION);
  await expect(page.locator('#version-server')).toContainText(API_VERSION);
  expect(await line.getAttribute('data-server')).toBe(API_VERSION);
});

test('version line reads CURRENT when the cached bundle matches the server', async ({ page }) => {
  await login(page);

  await page.route('**/version.json', route =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ frontend: FILE_VERSION }),
    }),
  );
  await stubHealth(page, FILE_VERSION);

  await page.goto('/index.html');

  const line = page.locator('#version-line');
  await expect(line).toHaveAttribute('data-state', 'current');
  await expect(page.locator('#version-server')).toHaveText('');
});

test('unmocked, the version line shows the real version.json value and the precache manifest carries that file', async ({ page }) => {
  // Half (b) of the guarantee — see the header. `serviceWorkers: 'block'` means
  // no test in this file can watch Workbox serve the file, so the link between
  // "the page reads version.json" and "that read is device-local and staleable"
  // is asserted against the COMMITTED sw.js manifest instead.
  const root = path.join(__dirname, '..');
  const sw = fs.readFileSync(path.join(root, 'sw.js'), 'utf8');
  const precached = precachedUrls(sw);
  expect(precached.length).toBeGreaterThan(0); // a parse that found nothing must not read as a pass
  expect(precached).toContain('version.json');

  const pkgVersion = JSON.parse(
    fs.readFileSync(path.join(root, 'package.json'), 'utf8'),
  ).version;

  // 🛑 version.json is git-IGNORED on purpose (.gitignore:13) yet SHIPS. Three
  // generators now cover the three places the tree is served:
  // `build-sw.js` (local dev / `task sw` / `task test` via its `sw` dep),
  // `backend/Dockerfile:57-64` (prod, inside the image), and — since B-92 —
  // `scripts/write-version-json.js` as a link of `playwright.config.js`'s
  // `webServer.command`, which is what covers the bare `npx playwright test`
  // that night-crew.toml's [e2e] stanza runs, subset leg included.
  //
  // The precondition stays asserted anyway, because it is now ALSO the test
  // that says the webServer link still fires: if that link is ever dropped or
  // reordered out of the chain, this reads as a named setup error instead of a
  // silent "v—" that looks like a code defect. A skipped precondition reads
  // exactly like a clean pass.
  const res = await page.request.get('/version.json');
  expect(
    res.ok(),
    'GET /version.json returned ' + res.status() + '. This stack has no version.json — ' +
      'it is a generated, git-ignored artifact. `playwright.config.js`\'s webServer.command ' +
      'runs `node scripts/write-version-json.js` for exactly this (B-92); if you are seeing ' +
      'this, that link is gone from the chain, or the stack is a provisioned ' +
      'NIGHTCREW_ENV_URL env whose image did not generate the file.',
  ).toBe(true);
  expect((await res.json()).frontend).toBe(pkgVersion);

  await login(page);
  await page.goto('/index.html');
  await expect(page.locator('#version-cached')).toHaveText('v' + pkgVersion);
  // Real file, real health endpoint, no stubs: the three-way parity means these
  // must agree, so the line must read CURRENT.
  await expect(page.locator('#version-line')).toHaveAttribute('data-state', 'current');
});
