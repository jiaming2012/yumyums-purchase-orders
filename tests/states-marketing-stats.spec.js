// ═══════════════════════════════════════════════════════════════════════════
// Card H4 `stats-tab-ui` — the CLAUDE.md self-verification ritual.
// ═══════════════════════════════════════════════════════════════════════════
//
// This environment is headless, so every row of the State Enumeration Table is
// FORCED, SCREENSHOT, and then the PNG is READ BACK with the multimodal Read
// tool and compared against the visual contract. What the SUMMARY reports is
// what was OBSERVED in the PNG, not what the code intended.
//
// 🛑 Screenshots go to .night-crew/runs/2026-10-02-autonomous/logs/h4/states/
// and are COMMITTED. They do NOT go under test-results/ — Playwright wipes its
// outputDir at the start of every run, and card 2 lost its cited evidence that
// way before its own full suite had finished.
//
// ── which rows ride a FIXTURE and which hit the REAL endpoint ──────────────
//   BI (bi.html #s3)
//     success        REAL     seeded mirror rows, GET /bi/campaigns/overview|by
//     empty          REAL     fixture tables cleared, real 200 with no attempts
//     loading        FIXTURE  a delayed route — a real server answers too fast
//     error          FIXTURE  a 500 the server will not produce on demand
//     no-bi-grant    FIXTURE  403 {"error":"forbidden","missing_grant":"bi"}
//     offline        FIXTURE  route.abort(), which has no server-side trigger
//     long content   FIXTURE  40 slices with 90-character labels
//   Marketing (marketing.html #s4)
//     success        REAL     seeded mirror rows, GET /reconciliation/queue
//     empty          REAL     fixture tables cleared
//     loading        FIXTURE  delayed route
//     error          FIXTURE  500
//     locked         REAL     a real team_member, a real 403 managers_only
//     offline        FIXTURE  route.abort()
//     long content   FIXTURE  30 rows with long campaign + item names
//
// The behavioural done_when rows [MS-01]–[MS-06] all hit the REAL endpoints —
// see tests/marketing-stats.spec.js, which contains no page.route() at all.

const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');
const { resolveE2eDb } = require('../scripts/reset-e2e-db');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const USER_PASSWORD = 'test456';

const SHOT_DIR = path.join(__dirname, '..', '.night-crew', 'runs',
  '2026-10-02-autonomous', 'logs', 'h4', 'states');
fs.mkdirSync(SHOT_DIR, { recursive: true });
test.use({ viewport: { width: 393, height: 852 } });

async function shot(page, name) {
  await page.screenshot({ path: path.join(SHOT_DIR, name + '.png'), fullPage: true });
}
async function login(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// ── psql fixture plumbing (see tests/marketing-stats.spec.js' long note) ───
function psql(sqlText) {
  const db = resolveE2eDb();
  return execFileSync('psql', [db.psqlUrl, '-At', '-v', 'ON_ERROR_STOP=1', '-c', sqlText],
    { encoding: 'utf8' });
}
function resetMarketingFixture() {
  psql(`DELETE FROM reconciliation_decisions;
        DELETE FROM scan_attempts_mirror;
        DELETE FROM toast_orders;
        DELETE FROM qr_scans;
        DELETE FROM qr_codes;
        DELETE FROM campaigns_admin;`);
}
const DAY = new Date(Date.now() - 24 * 3600 * 1000).toISOString().slice(0, 10);
const T = (hh, mm) => `${DAY} ${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}:00+00`;
const U = n => `bbbbbbbb-0000-4000-8000-00000000000${n}`;

function insertOrder(num, openedAt, amountCents, discountCents) {
  psql(`INSERT INTO toast_orders
          (business_date, order_number, order_id, opened_at, amount_cents, discount_cents, total_cents)
        VALUES ('${DAY}','${num}','ord-${num}','${openedAt}',${amountCents},${discountCents},${amountCents - discountCents})`);
}
function insertAttempt(id, scannedAt, opts) {
  const o = opts || {};
  const q = v => (v === null || v === undefined ? 'NULL' : `'${v}'`);
  psql(`INSERT INTO scan_attempts_mirror
          (id, code_id, campaign_id, device_id, scanned_at, status, offline_override,
           unverified_code, token_hash, pos_order_number, pos_business_date, match_status)
        VALUES ('${id}', ${q(o.codeId)}, ${q(o.campaignId)}, 'tablet-1', '${scannedAt}',
                'accepted', ${o.override ? 'true' : 'false'}, ${o.codeId ? 'false' : 'true'},
                ${o.codeId ? 'NULL' : `'hash-${id}'`},
                ${q(o.orderNumber)}, '${DAY}', 'unmatched')`);
}
async function createCampaign(page, name, fv) {
  const out = await page.evaluate(async ([nm, v]) => {
    const r = await fetch('/api/v1/marketing/campaigns', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: nm, offer_text: nm + ' offer', face_value_cents: v, landing: 'signup', runs_days: 60, channels: [{ channel: 'truck_sign' }] }),
    });
    return { status: r.status, body: await r.text() };
  }, [name, fv]);
  expect(out.status, `POST /campaigns → ${out.body}`).toBe(201);
  const body = JSON.parse(out.body);
  return { campaign: body.campaign, code: body.codes[0] };
}
async function seedReal(page) {
  resetMarketingFixture();
  const { campaign: camp, code } = await createCampaign(page, 'Taco Tuesday States', 500);
  insertOrder('101', T(17, 0), 3000, 500);
  insertOrder('102', T(17, 10), 2000, 0);
  insertOrder('103', T(17, 20), 1500, 300);
  insertAttempt(U(1), T(17, 1), { codeId: code.id, campaignId: camp.id, orderNumber: '101' });
  insertAttempt(U(2), T(17, 11), { codeId: code.id, campaignId: camp.id, orderNumber: '102' });
  insertAttempt(U(3), T(17, 5), { codeId: code.id, campaignId: camp.id, orderNumber: null });
  insertAttempt(U(4), T(17, 6), { codeId: code.id, campaignId: camp.id, orderNumber: '999' });
  insertAttempt(U(5), T(17, 21), { orderNumber: '103', override: true });
}

async function makeMember(page) {
  await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  const email = `h4-states-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  const invite = await page.evaluate(async (em) => {
    const r = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'H4', last_name: 'States', email: em, roles: ['team_member'] }),
    });
    return r.json();
  }, email);
  const token = (invite.invite_path || '').split('token=')[1];
  await page.evaluate(async ([t, pw]) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: pw }),
    });
  }, [token, USER_PASSWORD]);
  return email;
}

// ── fixture bodies ────────────────────────────────────────────────────────
const json = body => route => route.fulfill({
  status: 200, contentType: 'application/json', body: JSON.stringify(body),
});
const status = (code, body) => route => route.fulfill({
  status: code, contentType: 'application/json', body: JSON.stringify(body),
});
const money = over => Object.assign({
  revenue_cents: 6500, discount_cents: 1800, discount_basis: 'mixed', net_cents: 4700,
  per_dollar: 3.62, discount_implied_cents: 2000, discount_actual_cents: 800,
  discount_unknown_rows: 1, unattributed_redeemed: 1, unattributed_revenue_cents: 1500,
  avg_order_cents_with: 2166, avg_order_cents_without: null,
}, over || {});
const OVERVIEW = {
  period: '30d',
  funnel: { scans: 0, signups: 0, codes_sent: 0, redeemed: 5 },
  money: money(),
  reconciliation: {
    matched: 3, open: 3, declined: 0, orphan_rate: 0.4, threshold: 0.1,
    orphan_rate_basis: 'unmatched_and_orphans_excl_duplicate_scan',
    orphan_numerator: 2, orphan_denominator: 5,
  },
  needs_look: { overrides: 1, orphans: 1, unmatched: 1 },
  signups_basis: 'unavailable', codes_sent_basis: 'unavailable',
};
const LONG = 'Nduja & Pickled Fennel Porchetta Sandwich with Salsa Verde — Limited Autumn Run';
const LONG_BY = {
  dim: 'campaign', period: '30d',
  rows: Array.from({ length: 40 }, (_, i) => Object.assign({
    key: 'c' + i, label: `${LONG} #${i + 1}`, scans: 100 - i, signups: 0, redeemed: 40 - i,
  }, money({ revenue_cents: 123456 - i * 100, per_dollar: 12.34 }), { unattributed_redeemed: null, unattributed_revenue_cents: null })),
  totals: Object.assign({ key: 'totals', label: 'Total', scans: 4000, signups: 0, redeemed: 800 },
    money({ revenue_cents: 9876543 }), { unattributed_redeemed: null, unattributed_revenue_cents: null }),
  signups_basis: 'unavailable',
};
const BY = {
  dim: 'campaign', period: '30d',
  rows: [
    Object.assign({ key: 'c1', label: 'Taco Tuesday', scans: 0, signups: 0, redeemed: 4 },
      money({ revenue_cents: 5000, discount_cents: 1500, net_cents: 3500, per_dollar: 3.33, discount_implied_cents: 2000, discount_actual_cents: 500, discount_unknown_rows: 0 }),
      { unattributed_redeemed: null, unattributed_revenue_cents: null }),
    Object.assign({ key: 'unattributed', label: 'Unattributed', scans: 0, signups: 0, redeemed: 1 },
      money({ revenue_cents: 1500, discount_cents: 300, net_cents: 1200, per_dollar: 5, discount_implied_cents: 0, discount_actual_cents: 300, discount_unknown_rows: 1 }),
      { unattributed_redeemed: null, unattributed_revenue_cents: null }),
  ],
  totals: Object.assign({ key: 'totals', label: 'Total', scans: 0, signups: 0, redeemed: 5 }, money(),
    { unattributed_redeemed: null, unattributed_revenue_cents: null }),
  signups_basis: 'unavailable',
};
function qrow(n, bucket, over) {
  return Object.assign({
    bucket, id: U(n), scanned_at: T(12, n) .replace(' ', 'T').replace('+00', 'Z'),
    business_date: DAY, device_id: 'tablet-1', order_number: null,
    offline_override: bucket === 'override', override_by: null,
    unverified_code: bucket === 'override', policy_unresolved: false, device_reason: null,
    redeemed_value_cents: null, campaign_id: 'c1', campaign_name: 'Wing Wednesday',
    face_value_cents: 200, code_short: 'BBBBB2', channel: 'truck_sign', channel_label: null,
    item_name: '6pc Wings', order: null, suggestion: null, decision: null,
  }, over || {});
}
const QUEUE = (() => {
  const overrides = [qrow(4, 'override')];
  const orphans = [
    qrow(1, 'orphan', { suggestion: { order_number: '202', opened_at: T(12, 4).replace(' ', 'T').replace('+00', 'Z'), amount_cents: 1800, discount_cents: 200, voided: false, gap_seconds: 240, basis: 'window' } }),
    qrow(2, 'orphan', { suggestion: { order_number: '202', opened_at: T(12, 4).replace(' ', 'T').replace('+00', 'Z'), amount_cents: 1800, discount_cents: 200, voided: false, gap_seconds: 30360, basis: 'business_date' } }),
  ];
  const unmatched = [qrow(3, 'unmatched', { order_number: '888' })];
  return { queue: [...overrides, ...orphans, ...unmatched], overrides, orphans, unmatched, matched_count: 3, declined_count: 1 };
})();
const LONG_QUEUE = (() => {
  const orphans = Array.from({ length: 30 }, (_, i) => qrow(1, 'orphan', {
    id: `cccccccc-0000-4000-8000-${String(i).padStart(12, '0')}`,
    campaign_name: `${LONG} #${i + 1}`, item_name: LONG, code_short: 'ZZZZZ' + (i % 10),
  }));
  return { queue: orphans, overrides: [], orphans, unmatched: [], matched_count: 0, declined_count: 0 };
})();
const EMPTY_QUEUE = { queue: [], overrides: [], orphans: [], unmatched: [], matched_count: 0, declined_count: 0 };
const DECLINED = {
  declined: [Object.assign(qrow(9, 'declined'), {
    decision: { decision: 'declined', reason: 'other', note: 'Till drawer jammed', decided_by: 'Jamal M.', decided_at: T(13, 0).replace(' ', 'T').replace('+00', 'Z') },
    reason: 'other', note: 'Till drawer jammed', decided_by: 'Jamal M.',
    decided_at: T(13, 0).replace(' ', 'T').replace('+00', 'Z'),
  })],
};

async function biReady(page) {
  await page.waitForFunction(() => {
    const r = document.getElementById('bi-campaigns-root');
    return r && r.dataset.state && r.dataset.state !== 'loading';
  }, null, { timeout: 15000 });
}
async function msReady(page) {
  await page.waitForFunction(() => {
    const r = document.getElementById('ms-stats-root');
    return r && r.dataset.state && r.dataset.state !== 'loading';
  }, null, { timeout: 15000 });
}
async function gotoBi(page) {
  await page.goto('/bi.html#tab=3');
  await expect(page.locator('#s3')).toBeVisible();
}
async function gotoMs(page) {
  await page.goto('/marketing.html');
  await page.click('#t4');
  await expect(page.locator('#s4')).toBeVisible();
}
async function noOverflow(page) {
  const o = await page.evaluate(() =>
    document.documentElement.scrollWidth - document.documentElement.clientWidth);
  expect(o, 'the page body must never scroll horizontally at 393px').toBeLessThanOrEqual(1);
}

// ═══════════════════ BI · Campaigns (bi.html #s3) ═══════════════════════════
test.describe('States · BI Campaigns report', () => {

  test('BI success (REAL): funnel, money, the split line, the unknown-rows note, the health card and both slice rows', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedReal(page);
    await gotoBi(page);
    await biReady(page);
    await expect(page.locator('#bi-campaigns-root')).toHaveAttribute('data-state', 'ready');
    await expect(page.locator('#bic-money .bic-money-row[data-k="revenue"] .bic-money-v')).toHaveText('$65.00');
    await noOverflow(page);
    await shot(page, 'bi-success');
    await page.emulateMedia({ colorScheme: 'dark' });
    await shot(page, 'bi-success-dark');
  });

  test('BI empty (REAL): "No redemptions yet", and NO money figures at all', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    resetMarketingFixture();
    await gotoBi(page);
    await biReady(page);
    await expect(page.locator('#bi-campaigns-root')).toHaveAttribute('data-state', 'empty');
    await expect(page.locator('#bi-campaigns-root')).toContainText('No redemptions yet');
    await expect(page.locator('#bic-money')).toHaveCount(0);
    await expect(page.locator('#bic-health')).toHaveCount(0);
    await shot(page, 'bi-empty');
  });

  test('BI loading (FIXTURE): skeletons, and no figure on screen before one is known', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/api/v1/bi/campaigns/overview**', async r => { await new Promise(s => setTimeout(s, 2500)); await json(OVERVIEW)(r); });
    await page.route('**/api/v1/bi/campaigns/by**', async r => { await new Promise(s => setTimeout(s, 2500)); await json(BY)(r); });
    await gotoBi(page);
    await expect(page.locator('#bi-campaigns-root')).toHaveAttribute('data-state', 'loading');
    await expect(page.locator('#bi-campaigns-root .bic-skel').first()).toBeVisible();
    await expect(page.locator('#bi-campaigns-root')).not.toContainText('$');
    await shot(page, 'bi-loading');
  });

  test('BI error (FIXTURE): one honest failure card with a Retry, and NOT a figure beside it', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/api/v1/bi/campaigns/overview**', status(500, { error: 'internal_error' }));
    await page.route('**/api/v1/bi/campaigns/by**', status(500, { error: 'internal_error' }));
    await gotoBi(page);
    await biReady(page);
    await expect(page.locator('#bi-campaigns-root')).toHaveAttribute('data-state', 'error');
    await expect(page.locator('.bic-error')).toContainText('Couldn’t load');
    await expect(page.locator('.bic-retry')).toBeVisible();
    // The ritual caught exactly this on another card tonight: an error state
    // must not also claim "no data", and must carry no zeros.
    await expect(page.locator('#bi-campaigns-root')).not.toContainText('No redemptions yet');
    await expect(page.locator('#bi-campaigns-root')).not.toContainText('$0.00');
    await shot(page, 'bi-error');
  });

  test('BI no-bi-grant (FIXTURE): a 403 names the missing grant instead of an empty report', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/api/v1/bi/campaigns/**', status(403, { error: 'forbidden', missing_grant: 'bi' }));
    await gotoBi(page);
    await biReady(page);
    await expect(page.locator('#bi-campaigns-root')).toHaveAttribute('data-state', 'locked');
    await expect(page.locator('.bic-locked')).toContainText('BI');
    await shot(page, 'bi-locked');
  });

  test('BI offline (FIXTURE): the failure says it is the network and stays retryable', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/api/v1/bi/campaigns/**', r => r.abort('internetdisconnected'));
    await gotoBi(page);
    await biReady(page);
    await expect(page.locator('#bi-campaigns-root')).toHaveAttribute('data-state', 'error');
    await expect(page.locator('.bic-error')).toContainText('offline');
    await expect(page.locator('.bic-retry')).toBeVisible();
    await shot(page, 'bi-offline');
  });

  test('BI long content (FIXTURE): 40 slices with 90-char labels — no clipped word, no sideways scroll', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/api/v1/bi/campaigns/overview**', json(OVERVIEW));
    await page.route('**/api/v1/bi/campaigns/by**', json(LONG_BY));
    await gotoBi(page);
    await biReady(page);
    await expect(page.locator('#bic-slice .bic-row')).toHaveCount(40);
    await noOverflow(page);
    await shot(page, 'bi-long');
  });
});

// ═══════════════ Marketing · reconciliation queue (#s4) ════════════════════
test.describe('States · Marketing reconciliation queue', () => {

  test('MS success (REAL): the header count, the health card, and all three buckets in ladder order', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedReal(page);
    await gotoMs(page);
    await msReady(page);
    await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'ready');
    await expect(page.locator('#msq-head')).toContainText('3 redemptions need a look');
    const buckets = await page.locator('.msq-section').evaluateAll(
      els => els.map(e => e.dataset.bucket));
    expect(buckets.slice(0, 3)).toEqual(['override', 'orphan', 'unmatched']);
    await noOverflow(page);
    await shot(page, 'ms-success');
    await page.emulateMedia({ colorScheme: 'dark' });
    await shot(page, 'ms-success-dark');
  });

  test('MS empty (REAL): "No redemptions yet" and no health figures', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    resetMarketingFixture();
    await gotoMs(page);
    await msReady(page);
    await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'empty');
    await expect(page.locator('#ms-stats-root')).toContainText('No redemptions yet');
    await expect(page.locator('#msq-health')).toHaveCount(0);
    await shot(page, 'ms-empty');
  });

  test('MS locked (REAL): a team_member gets Managers only, and the section still names itself', async ({ page }) => {
    const email = await makeMember(page);
    await login(page, email, USER_PASSWORD);
    await gotoMs(page);
    await msReady(page);
    await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'locked');
    await expect(page.locator('.ms-locked')).toContainText('Managers only');
    await expect(page.locator('#s4')).toContainText('Redemption stats');
    await expect(page.locator('#s4 .badge')).toHaveCount(0);
    // No control at all is offered to someone who may not act.
    await expect(page.locator('#s4 .msq-decline')).toHaveCount(0);
    await expect(page.locator('#s4 .msq-fix')).toHaveCount(0);
    await shot(page, 'ms-locked');
  });

  test('MS loading (FIXTURE): skeletons, and no count claimed before one is known', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/reconciliation/queue**', async r => { await new Promise(s => setTimeout(s, 2500)); await json(QUEUE)(r); });
    await page.route('**/marketing/stats/overview**', async r => { await new Promise(s => setTimeout(s, 2500)); await json(OVERVIEW)(r); });
    await page.route('**/reconciliation/declined**', async r => { await new Promise(s => setTimeout(s, 2500)); await json(DECLINED)(r); });
    await gotoMs(page);
    await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'loading');
    await expect(page.locator('#ms-stats-root .msq-skel').first()).toBeVisible();
    await expect(page.locator('#msq-head')).toHaveCount(0);
    await shot(page, 'ms-loading');
  });

  test('MS error (FIXTURE): one failure card with a Retry, no "No redemptions yet", no zeros', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/reconciliation/**', status(500, { error: 'internal_error' }));
    await page.route('**/marketing/stats/overview**', status(500, { error: 'internal_error' }));
    await gotoMs(page);
    await msReady(page);
    await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'error');
    await expect(page.locator('.msq-error')).toContainText('Couldn’t load');
    await expect(page.locator('.msq-retry')).toBeVisible();
    await expect(page.locator('#ms-stats-root')).not.toContainText('No redemptions yet');
    await expect(page.locator('#ms-stats-root')).not.toContainText('0 redemptions need a look');
    await shot(page, 'ms-error');
  });

  test('MS offline (FIXTURE): the failure says it is the network and stays retryable', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/reconciliation/**', r => r.abort('internetdisconnected'));
    await page.route('**/marketing/stats/overview**', r => r.abort('internetdisconnected'));
    await gotoMs(page);
    await msReady(page);
    await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'error');
    await expect(page.locator('.msq-error')).toContainText('offline');
    await shot(page, 'ms-offline');
  });

  test('MS long content (FIXTURE): 30 rows with 90-char names — the row keeps both buttons reachable', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/reconciliation/queue**', json(LONG_QUEUE));
    await page.route('**/reconciliation/declined**', json({ declined: [] }));
    await page.route('**/marketing/stats/overview**', json(OVERVIEW));
    await gotoMs(page);
    await msReady(page);
    await expect(page.locator('.msq-row')).toHaveCount(30);
    // Both actions stay on screen and stay 44px, however long the name is.
    const sizes = await page.locator('.msq-row').first().locator('.msq-fix, .msq-decline')
      .evaluateAll(els => els.map(e => ({ h: e.getBoundingClientRect().height, r: e.getBoundingClientRect().right })));
    expect(sizes.length).toBe(2);
    for (const s of sizes) { expect(s.h).toBeGreaterThanOrEqual(44); expect(s.r).toBeLessThanOrEqual(394); }
    await noOverflow(page);
    await shot(page, 'ms-long');
  });

  test('MS sheets (FIXTURE): the add-order sheet shows basis + gap; the decline sheet states what declining does', async ({ page }) => {
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/reconciliation/queue**', json(QUEUE));
    await page.route('**/reconciliation/declined**', json(DECLINED));
    await page.route('**/marketing/stats/overview**', json(OVERVIEW));
    await gotoMs(page);
    await msReady(page);

    await page.locator(`.msq-row[data-id="${U(1)}"] .msq-fix`).click();
    await expect(page.locator('#msq-sheet-order')).toBeVisible();
    await expect(page.locator('#msq-sheet-order .msq-sug[data-order="202"]')).toContainText('within 30 min');
    await shot(page, 'ms-sheet-order');
    await page.locator('#msq-order-close').click();

    // The business_date rung, rendered as advisory rather than as a match.
    await page.locator(`.msq-row[data-id="${U(2)}"] .msq-fix`).click();
    await expect(page.locator('#msq-sheet-order .msq-sug[data-order="202"]')).toContainText('same business date');
    await shot(page, 'ms-sheet-order-businessdate');
    await page.locator('#msq-order-close').click();

    await page.locator(`.msq-row[data-id="${U(1)}"] .msq-decline`).click();
    await expect(page.locator('#msq-sheet-decline')).toBeVisible();
    await expect(page.locator('#msq-decline-save')).toBeDisabled();
    await page.locator('.msq-reason[data-reason="other"]').click();
    await shot(page, 'ms-sheet-decline');

    // And the declined bucket, with its note and its Reopen.
    await page.locator('#msq-decline-close').click();
    const bucket = page.locator('.msq-section[data-bucket="declined"]');
    await expect(bucket.locator('.msq-decl-note')).toContainText('Till drawer jammed');
    await expect(bucket.locator('.msq-reopen')).toBeVisible();
    await shot(page, 'ms-declined-bucket');
  });
});
