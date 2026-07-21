const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

// states-trends.spec.js — the CLAUDE.md self-verification ritual for the Trends
// tab (`#s5`, FR-1 / FR-6b), rendering GET /api/v1/inventory/trends
// (design §2.2 AS AMENDED 2026-07-20 — decisions 29/30/31).
//
// Every row of this card's State Enumeration Table is FORCED here, navigated to,
// and screenshotted so the PNGs can be read back and compared to the visual
// contract.
//
// Rows covered by this file:
//   Empty                      -> 'empty'                (live endpoint, no purchases)
//   Loading                    -> 'loading'              (delayed route)
//   Error                      -> 'error'                (500 route)
//   Populated                  -> 'populated'            (mocked fixture)
//   Edge: net-zero cell        -> 'edge-net-zero'        (CARRIED CAVEAT 1)
//   Edge: offsetting unitemized-> 'edge-offsetting'      (CARRIED CAVEAT 2)
//   Edge: rounding drift       -> 'edge-drift'           (CARRIED CAVEAT 3)
//   Edge: all-unlinked window  -> 'edge-all-unlinked'    (no groups, money exists)
//   Edge: long group name      -> 'edge-long-name'       (480px shell holds)
//   Edge: ungated user         -> OWNED BY F5 — see NOTE below
//
// NOTE FOR F5 (`inventory-tab-gating`): the "ungated user" row belongs to F5.
// It slots in as a new `test.describe('Trends tab — gating')` block at the
// BOTTOM of this file: log in as a user WITHOUT the `inventory-trends` grant,
// assert `#t5` / `#s5` are not rendered, assert a direct
// `GET /api/v1/inventory/trends` returns 403, screenshot as `edge-ungated.png`
// via the same `shot()` helper. This card additionally asserts (see
// 'no top-level listener binds to a Trends node') that removing #s5 from the
// DOM does not throw at parse time — the hazard F4's review carried forward.

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

const SHOT_DIR = path.join(__dirname, '..', 'test-results', 'states-trends');
fs.mkdirSync(SHOT_DIR, { recursive: true });

async function shot(page, name) {
  await page.screenshot({ path: path.join(SHOT_DIR, name + '.png'), fullPage: true });
}

async function login(page) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// Navigate straight to the Trends tab via the shared #tab= hash contract (tab.js).
async function openTrendsTab(page) {
  await page.goto('/inventory.html#tab=5');
  await page.waitForSelector('#s5:visible');
}

const WINDOW = { from: '2026-04-27', to: '2026-07-19', weeks: 12 };

// The 12 ISO week starts (Monday) of WINDOW, in order.
const WEEKS = [
  '2026-04-27', '2026-05-04', '2026-05-11', '2026-05-18',
  '2026-05-25', '2026-06-01', '2026-06-08', '2026-06-15',
  '2026-06-22', '2026-06-29', '2026-07-06', '2026-07-13',
];

const G = {
  proteins: 'aaaaaaaa-0000-0000-0000-000000000001',
  produce:  'aaaaaaaa-0000-0000-0000-000000000002',
  bev:      'aaaaaaaa-0000-0000-0000-000000000003',
  dry:      'aaaaaaaa-0000-0000-0000-000000000004',
};

const GROUPS = [
  { id: G.bev,      name: 'Beverages' },
  { id: G.dry,      name: 'Dry Goods' },
  { id: G.proteins, name: 'Proteins' },
  { id: G.produce,  name: 'Produce' },
  // The D2 pseudo-group is always pinned last by the server.
  { id: 'ungrouped', name: 'Ungrouped' },
];

function round2(n) { return Math.round(n * 100) / 100; }

// Deterministic pseudo-random spend so the fixture is stable across runs but
// does not look hand-flattened.
function spendFor(weekIdx, base, swing) {
  return round2(base + swing * Math.sin(weekIdx * 1.1) + (weekIdx % 3) * 7.5);
}

function buildCells(opts = {}) {
  const skip = opts.skip || [];   // ["<week>|<groupId>", …] — omitted cells
  const cells = [];
  WEEKS.forEach((wk, i) => {
    [
      [G.proteins, 380, 90],
      [G.produce, 190, 55],
      [G.bev, 96, 30],
      [G.dry, 61, 18],
      ['ungrouped', 24, 9],
    ].forEach(([gid, base, swing]) => {
      if (skip.includes(wk + '|' + gid)) return;
      cells.push({ week_start: wk, group_id: gid, spend: spendFor(i, base, swing) });
    });
  });
  return cells;
}

function buildUnlinked() {
  // Not every week has unlinked spend — sparse, like the wire.
  return WEEKS.filter((_, i) => i % 2 === 0)
    .map((wk, i) => ({ week_start: wk, spend: round2(31.4 + i * 6.2) }));
}

function sumCells(cells) { return round2(cells.reduce((a, c) => a + c.spend, 0)); }
function sumUnlinked(u) { return round2(u.reduce((a, c) => a + c.spend, 0)); }

// Build a response whose published identity figure is EXACTLY consistent with
// the display cells — the well-behaved case.
function makeResponse(over = {}) {
  const cells = over.cells || buildCells();
  const unlinked = over.unlinked !== undefined ? over.unlinked : buildUnlinked();
  const pendingTotal = over.pending_total !== undefined ? over.pending_total : 240.00;
  const pendingCount = over.pending_count !== undefined ? over.pending_count : 3;
  const unlinkedTotal = round2(sumUnlinked(unlinked));
  const consistent = round2(sumCells(cells) + unlinkedTotal + pendingTotal);
  return {
    window: WINDOW,
    groups: over.groups || GROUPS,
    cells,
    unlinked,
    unlinked_total: unlinkedTotal,
    completeness: {
      pending_total: pendingTotal,
      pending_count: pendingCount,
      unitemized_remainder: over.unitemized_remainder !== undefined
        ? over.unitemized_remainder : 18.45,
      reconciles_to_cogs_excl_tax: over.reconciles_to_cogs_excl_tax !== undefined
        ? over.reconciles_to_cogs_excl_tax : consistent,
    },
  };
}

const POPULATED = makeResponse();

// CARRIED CAVEAT 1 — trends.go drops any week×group cell whose lines net to
// exactly zero (`if spend == 0 { continue }`). On the wire that is byte-identical
// to "no activity". Proteins is missing for 2026-06-01 here.
const NET_ZERO = makeResponse({ cells: buildCells({ skip: ['2026-06-01|' + G.proteins] }) });

// CARRIED CAVEAT 2 — unitemized_remainder sums ALGEBRAICALLY. A near-zero net can
// hide two large offsetting coverage gaps, and the endpoint publishes no count.
// $0.03 net must NOT read as "coverage is fine".
const OFFSETTING = makeResponse({ unitemized_remainder: 0.03 });

// CARRIED CAVEAT 3 — display cells are each penny-rounded; the published identity
// figure is one window-wide sum rounded once. They can legitimately disagree.
// The payroll-facing number is completeness.reconciles_to_cogs_excl_tax.
const DRIFT = (() => {
  const base = makeResponse();
  base.completeness.reconciles_to_cogs_excl_tax =
    round2(base.completeness.reconciles_to_cogs_excl_tax - 0.04);
  return base;
})();

// Money exists but nothing is linked to a catalog item: groups/cells empty,
// unlinked non-empty. This must NOT render the "no spending" empty card.
const ALL_UNLINKED = makeResponse({
  groups: [], cells: [], pending_total: 0, pending_count: 0, unitemized_remainder: 0,
});

const LONG_NAME = makeResponse({
  groups: GROUPS.map(g => g.id === G.produce
    ? { id: g.id, name: 'Produce, Herbs, Leafy Greens and Cold-Chain Perishables' }
    : g),
});

const EMPTY = {
  window: WINDOW, groups: [], cells: [], unlinked: [], unlinked_total: 0,
  completeness: {
    pending_total: 0, pending_count: 0,
    unitemized_remainder: 0, reconciles_to_cogs_excl_tax: 0,
  },
};

function mockTrends(page, body, opts = {}) {
  return page.route('**/api/v1/inventory/trends*', async route => {
    if (opts.delayMs) await new Promise(r => setTimeout(r, opts.delayMs));
    if (opts.status && opts.status >= 400) {
      return route.fulfill({
        status: opts.status, contentType: 'application/json',
        body: JSON.stringify({ error: 'internal_error' }),
      });
    }
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
  });
}

// Nothing in this tab may ever render a fabricated or broken number.
async function assertNoJunkNumbers(page) {
  const host = page.locator('#trends-container');
  await expect(host).not.toContainText('NaN');
  await expect(host).not.toContainText('Infinity');
  await expect(host).not.toContainText('undefined');
  await expect(host).not.toContainText('$null');
}

test.describe('Trends tab (#s5) — State Enumeration', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  // ── Row: Empty ──────────────────────────────────────────────────────────
  test('Empty — no confirmed COGS in the window renders the honest empty card', async ({ page }) => {
    await mockTrends(page, EMPTY);
    await openTrendsTab(page);
    const empty = page.locator('#trends-container .tr-empty');
    await expect(empty).toBeVisible();
    await expect(empty).toContainText('No confirmed spending yet');
    // The window is still stated — an empty chart is a claim about a period.
    await expect(page.locator('#trends-container .tr-window')).toContainText('12 weeks');
    await expect(page.locator('#trends-container svg.tr-svg')).toHaveCount(0);
    await expect(page.locator('#trends-container .tr-table')).toHaveCount(0);
    await assertNoJunkNumbers(page);
    await shot(page, 'empty');
  });

  // ── Row: Loading ────────────────────────────────────────────────────────
  test('Loading — skeleton shown in #s5 while the request is in flight', async ({ page }) => {
    await mockTrends(page, POPULATED, { delayMs: 2500 });
    await page.goto('/inventory.html#tab=5');
    await page.waitForSelector('#s5:visible');
    await expect(page.locator('#trends-container .skeleton').first()).toBeVisible();
    await expect(page.locator('#trends-container svg.tr-svg')).toHaveCount(0);
    await shot(page, 'loading');
  });

  // ── Row: Error ──────────────────────────────────────────────────────────
  test('Error — 500 renders an inline error with Retry, never a zeroed chart', async ({ page }) => {
    await mockTrends(page, null, { status: 500 });
    await openTrendsTab(page);
    const host = page.locator('#trends-container');
    await expect(host).toContainText('Couldn’t load spending trends.');
    await expect(host.locator('button', { hasText: 'Retry' })).toBeVisible();
    // A failed load must never fall back to an all-zero chart.
    await expect(page.locator('#trends-container svg.tr-svg')).toHaveCount(0);
    await assertNoJunkNumbers(page);
    await shot(page, 'error');
  });

  // ── Row: Populated ──────────────────────────────────────────────────────
  test('Populated — chart, legend, completeness block and table all render', async ({ page }) => {
    await mockTrends(page, POPULATED);
    await openTrendsTab(page);

    await expect(page.locator('#trends-container .tr-window')).toContainText('12 weeks');

    // The chart: one stacked column per week, one legend entry per group + Unlinked.
    const svg = page.locator('#trends-container svg.tr-svg');
    await expect(svg).toBeVisible();
    await expect(page.locator('#trends-container .tr-col')).toHaveCount(12);
    await expect(page.locator('#trends-container .tr-legend-item'))
      .toHaveCount(GROUPS.length + 1);
    await expect(page.locator('#trends-container .tr-legend'))
      .toContainText('Unlinked');

    // COMPLETENESS — Decisions 30 and 31 exist so unattributed money is visible.
    const rec = page.locator('#trends-container .tr-recon');
    await expect(rec).toBeVisible();
    await expect(rec).toContainText('Awaiting review');
    await expect(rec).toContainText('3 receipts');
    await expect(rec).toContainText('$240.00');
    await expect(rec).toContainText('Unlinked');
    await expect(rec).toContainText('$' + POPULATED.unlinked_total.toFixed(2));

    // CARRIED CAVEAT 3 — the hero is the payroll-facing figure, by name.
    const hero = page.locator('#trends-container .tr-hero-val');
    await expect(hero).toHaveText(
      '$' + POPULATED.completeness.reconciles_to_cogs_excl_tax
        .toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }));
    await expect(page.locator('#trends-container .tr-hero'))
      .toContainText('Confirmed COGS');

    // The table view — the light-mode relief obligation for a categorical palette.
    await expect(page.locator('#trends-container .tr-table')).toBeVisible();
    await expect(page.locator('#trends-container .tr-table tbody tr')).toHaveCount(12);

    await assertNoJunkNumbers(page);
    await shot(page, 'populated');
  });

  // ── Row: Populated → week drill-down ────────────────────────────────────
  test('Populated — tapping a column opens that week’s per-group breakdown', async ({ page }) => {
    await mockTrends(page, POPULATED);
    await openTrendsTab(page);
    await page.locator('#trends-container .tr-col').nth(11).click();
    const detail = page.locator('#trends-container .tr-week-detail');
    await expect(detail).toBeVisible();
    await expect(detail).toContainText('Proteins');
    await expect(detail).toContainText('Unlinked');
    await assertNoJunkNumbers(page);
    await shot(page, 'populated-week-detail');
  });

  // ── Edge row 1 (CARRIED CAVEAT 1): a week×group cell that nets to zero ──
  // The server drops it, so it is indistinguishable from "no activity". The UI
  // must SAY SO rather than let the gap read as a confident zero.
  test('Edge: net-zero cell — omitted cells are disclosed, never drawn as $0.00', async ({ page }) => {
    await mockTrends(page, NET_ZERO);
    await openTrendsTab(page);

    // The standing disclosure is present whenever the chart renders.
    await expect(page.locator('#trends-container .tr-zero-note'))
      .toContainText('nets to exactly $0');

    // Drill into the affected week: the missing group is named as "no cell",
    // NOT rendered as $0.00.
    await page.locator('#trends-container .tr-col[data-week="2026-06-01"]').click();
    const row = page.locator('#trends-container .tr-week-detail [data-group-id="' + G.proteins + '"]');
    await expect(row).toBeVisible();
    await expect(row).toContainText('no cell');
    await expect(row).not.toContainText('$0.00');
    await assertNoJunkNumbers(page);
    await shot(page, 'edge-net-zero');
  });

  // ── Edge row 2 (CARRIED CAVEAT 2): offsetting unitemized remainder ──────
  // $0.03 net can hide two large offsetting coverage gaps. It must not be
  // rendered as a single reassuring number.
  test('Edge: offsetting unitemized — near-zero net is labelled net, not clean', async ({ page }) => {
    await mockTrends(page, OFFSETTING);
    await openTrendsTab(page);
    const un = page.locator('#trends-container .tr-unitemized');
    await expect(un).toBeVisible();
    await expect(un).toContainText('$0.03');
    // The word "net" carries the whole warning — it must be on screen unexpanded.
    await expect(un).toContainText('net');
    // And it must never be reported as an all-clear.
    await expect(un).not.toContainText('fully itemized');
    await expect(un).not.toContainText('All receipts itemized');

    // The drill-down explaining the algebraic sum is reachable.
    await un.locator('[data-action="toggle-unitemized"]').click();
    await expect(page.locator('#trends-container .tr-unitemized-detail'))
      .toContainText('offsetting');
    await assertNoJunkNumbers(page);
    await shot(page, 'edge-offsetting');
  });

  // ── Edge row 3 (CARRIED CAVEAT 3): per-cell rounding drift ──────────────
  test('Edge: rounding drift — hero stays the payroll figure and the gap is named', async ({ page }) => {
    await mockTrends(page, DRIFT);
    await openTrendsTab(page);

    // The hero is reconciles_to_cogs_excl_tax — NOT the sum of display cells.
    await expect(page.locator('#trends-container .tr-hero-val')).toHaveText(
      '$' + DRIFT.completeness.reconciles_to_cogs_excl_tax
        .toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 }));

    // The disagreement with the chart's own arithmetic is disclosed, not hidden.
    const drift = page.locator('#trends-container .tr-drift');
    await expect(drift).toBeVisible();
    await expect(drift).toContainText('rounding');
    await assertNoJunkNumbers(page);
    await shot(page, 'edge-drift');
  });

  // ── Edge row 4: money exists, nothing is linked ─────────────────────────
  test('Edge: all-unlinked — unlinked-only window is a chart, not the empty card', async ({ page }) => {
    await mockTrends(page, ALL_UNLINKED);
    await openTrendsTab(page);
    await expect(page.locator('#trends-container .tr-empty')).toHaveCount(0);
    await expect(page.locator('#trends-container svg.tr-svg')).toBeVisible();
    await expect(page.locator('#trends-container .tr-legend')).toContainText('Unlinked');
    await expect(page.locator('#trends-container .tr-recon'))
      .toContainText('$' + ALL_UNLINKED.unlinked_total.toFixed(2));
    await assertNoJunkNumbers(page);
    await shot(page, 'edge-all-unlinked');
  });

  // ── Edge row 5: long group name in a 480px shell ────────────────────────
  test('Edge: long group name — the 480px shell does not scroll horizontally', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await mockTrends(page, LONG_NAME);
    await openTrendsTab(page);
    await expect(page.locator('#trends-container .tr-legend'))
      .toContainText('Cold-Chain Perishables');
    // The page body must never scroll horizontally; wide content scrolls in its
    // own container.
    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow).toBeLessThanOrEqual(1);
    await assertNoJunkNumbers(page);
    await shot(page, 'edge-long-name');
  });

  // ── Hazard carried from F4's review ─────────────────────────────────────
  // F5 will remove #s5 / #trends-container for un-granted users. An unguarded
  // top-level getElementById(...).addEventListener would throw at PARSE time and
  // break the ENTIRE inventory page — every tab — for exactly those users.
  test('Hazard: removing the Trends nodes never throws and never breaks other tabs', async ({ page }) => {
    const errors = [];
    page.on('pageerror', e => errors.push(e.message));

    // Strip #s5 and #t5 out of the document before any script runs — the shape
    // F5's server-side gating will produce.
    await page.route('**/inventory.html', async route => {
      const res = await route.fetch();
      let html = await res.text();
      html = html.replace(/<div id="s5"[\s\S]*?<\/div>\s*<div id="s6"/, '<div id="s6"');
      html = html.replace(/<button id="t5"[\s\S]*?<\/button>/, '');
      await route.fulfill({ response: res, body: html, headers: { ...res.headers(), 'content-type': 'text/html' } });
    });

    await page.goto('/inventory.html#tab=1');
    await page.waitForSelector('#s1');
    expect(await page.locator('#s5').count()).toBe(0);

    // Other tabs must still work — the whole point of the hazard.
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await page.click('#t6');
    await expect(page.locator('#s6')).toBeVisible();
    // A stray click where the Trends chart would have been must be inert.
    await page.mouse.click(200, 300);

    expect(errors).toEqual([]);
    await shot(page, 'hazard-nodes-removed');
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// F5 · inventory-tab-gating — the "ungated user" State-Enumeration row
// ═══════════════════════════════════════════════════════════════════════════
//
// Design §1.2 (the observable rule), §1.4 (Option (i): per-tab hq_apps slugs),
// §1.6 (the mixed Trends-only case), §8 amendment 1 (umbrella: an `inventory`
// app grant covers every gated tab of that app).
//
// The pair this block owes, per tab:
//   WITHOUT grant -> #tN/#sN absent AND a direct GET returns 403 with the
//                    distinct envelope {"error":"forbidden","missing_grant":…}
//   WITH grant    -> tab renders AND the endpoint returns 200
//
// The direct-fetch half is the load-bearing one: hiding a tab client-side while
// leaving the endpoint reachable is explicitly a violation (§1.2 rule 3,
// "0 logged-in-only bypass").

const GATE_PASSWORD = 'test456';

async function loginAs(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// makeGatedUser invites a team_member, accepts the invite, then (as admin)
// grants exactly the slugs asked for. Returns the new user's credentials.
//
// Grants are APPENDED to each slug's existing user_grants: the PUT endpoint is a
// full replace per slug, so a naive write would silently revoke the grant a
// previously-created test user still holds.
async function makeGatedUser(page, tag, grantSlugs) {
  const email = `f5-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;

  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  const invite = await page.evaluate(async (em) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Gate', last_name: 'Tester', email: em, roles: ['team_member'] }),
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

  // accept-invite rotated the cookie onto the new user — go back to admin to grant.
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  const userId = await page.evaluate(async (em) => {
    const list = await (await fetch('/api/v1/users')).json();
    const u = (list || []).find(x => x.email === em);
    return u ? String(u.id) : null;
  }, email);
  expect(userId, 'new user id').toBeTruthy();

  for (const slug of grantSlugs) {
    await page.evaluate(async ([s, uid]) => {
      const perms = await (await fetch('/api/v1/apps/permissions')).json();
      const app = (perms || []).find(a => a.slug === s);
      const users = ((app && app.user_grants) || []).map(String);
      if (!users.includes(uid)) users.push(uid);
      await fetch('/api/v1/apps/' + s + '/permissions', {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role_grants: (app && app.role_grants) || [], user_grants: users }),
      });
    }, [slug, userId]);
  }

  return { email, password: GATE_PASSWORD, id: userId };
}

// probe performs a same-origin fetch from the page session and reports the raw
// status + body — the direct-endpoint negative that proves the gate is served,
// not merely drawn.
async function probe(page, path) {
  return page.evaluate(async (p) => {
    const r = await fetch(p);
    return { status: r.status, body: await r.text() };
  }, path);
}

test.describe('Trends tab — gating', () => {

  // ── WITHOUT the grant ────────────────────────────────────────────────────
  test('Edge: ungated user — #t5/#s5 absent, GET /inventory/trends returns 403', async ({ page }) => {
    const errors = [];
    page.on('pageerror', e => errors.push(e.message));

    const user = await makeGatedUser(page, 'trends-ungated', []);
    await loginAs(page, user.email, user.password);
    await page.goto('/inventory.html');
    await page.waitForSelector('#s1');

    // 1. The endpoint is DENIED — asserted FIRST, deliberately. The server 403
    //    is the gate; the hidden tab is only UX. If this assertion is the one
    //    that fails, the feature is a facade regardless of what the DOM shows.
    const res = await probe(page, '/api/v1/inventory/trends');
    expect(res.status).toBe(403);
    const env = JSON.parse(res.body);
    expect(env.error).toBe('forbidden');
    expect(env.missing_grant).toBe('inventory-trends');

    // 2. And the tab does not render — neither the button nor the panel.
    await expect(page.locator('#t5')).toHaveCount(0);
    await expect(page.locator('#s5')).toHaveCount(0);
    await expect(page.locator('#trends-container')).toHaveCount(0);

    // 3. Null-safety: removing the Trends nodes must not break the page for the
    //    very users being gated. Other tabs still switch; nothing threw.
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await page.click('#t1');
    await expect(page.locator('#s1')).toBeVisible();
    expect(errors).toEqual([]);

    await shot(page, 'edge-ungated');
  });

  test('Ungated deep link (#tab=5) falls back instead of rendering a tab shell', async ({ page }) => {
    const errors = [];
    page.on('pageerror', e => errors.push(e.message));

    const user = await makeGatedUser(page, 'trends-deeplink', []);
    await loginAs(page, user.email, user.password);
    await page.goto('/inventory.html#tab=5');
    await page.waitForSelector('#s1');

    await expect(page.locator('#s5')).toHaveCount(0);
    // A pasted URL lands on a real tab, not a blank page.
    await expect(page.locator('#s1')).toBeVisible();
    expect(errors).toEqual([]);

    await shot(page, 'edge-ungated-deeplink');
  });

  // ── WITH the grant ───────────────────────────────────────────────────────
  test('Granted user — #t5/#s5 render and GET /inventory/trends returns 200', async ({ page }) => {
    const user = await makeGatedUser(page, 'trends-granted', ['inventory-trends']);
    await loginAs(page, user.email, user.password);
    await page.goto('/inventory.html');
    await page.waitForSelector('#s1');

    await expect(page.locator('#t5')).toHaveCount(1);
    await page.click('#t5');
    await expect(page.locator('#s5')).toBeVisible();

    const res = await probe(page, '/api/v1/inventory/trends');
    expect(res.status).toBe(200);

    await shot(page, 'gated-granted-trends');
  });

  // ── The umbrella rider (design §8 amendment 1) ───────────────────────────
  test('Umbrella: a whole-app `inventory` grant alone opens Trends', async ({ page }) => {
    const user = await makeGatedUser(page, 'trends-umbrella', ['inventory']);
    await loginAs(page, user.email, user.password);
    await page.goto('/inventory.html');
    await page.waitForSelector('#s1');

    await expect(page.locator('#t5')).toHaveCount(1);
    await expect(page.locator('#t6')).toHaveCount(1);
    expect((await probe(page, '/api/v1/inventory/trends')).status).toBe(200);
    expect((await probe(page, '/api/v1/inventory/cost')).status).toBe(200);

    await shot(page, 'gated-umbrella');
  });

  // ── §1.6 — the mixed case, testable only because F1/F3 landed first ──────
  test('Mixed: Trends-only grant renders #t5, hides #t6, and 403s /inventory/cost', async ({ page }) => {
    const errors = [];
    page.on('pageerror', e => errors.push(e.message));

    const user = await makeGatedUser(page, 'mixed-trends-only', ['inventory-trends']);
    await loginAs(page, user.email, user.password);
    await page.goto('/inventory.html');
    await page.waitForSelector('#s1');

    // Trends: visible + served.
    await expect(page.locator('#t5')).toHaveCount(1);
    await page.click('#t5');
    await expect(page.locator('#s5')).toBeVisible();
    expect((await probe(page, '/api/v1/inventory/trends')).status).toBe(200);

    // Cost: absent + denied, independently.
    await expect(page.locator('#t6')).toHaveCount(0);
    await expect(page.locator('#s6')).toHaveCount(0);
    const cost = await probe(page, '/api/v1/inventory/cost');
    expect(cost.status).toBe(403);
    expect(JSON.parse(cost.body).missing_grant).toBe('inventory-cost');

    expect(errors).toEqual([]);
    await shot(page, 'edge-mixed-trends-only');
  });
});
