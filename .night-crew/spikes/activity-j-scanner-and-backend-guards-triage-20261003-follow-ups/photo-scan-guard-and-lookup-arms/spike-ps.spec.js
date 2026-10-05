// spike-ps.spec.js — THROWAWAY spec for spike 01 of card J1 (photo-scan-guard-and-lookup-arms,
// B-475). Copied by the spike script into a worktree's tests/ directory and run there; it is
// never part of the suite. Self-contained on purpose: the helpers it needs from
// tests/marketing.spec.js (loginAs, mockSyncTransports, fixture rows, the lookup URL matcher)
// are per-file in that suite and not exported, so the minimum is re-stated here.
//
// What it drives (the adversarial reviewer's reproduction, ledger T-64 / B-475):
//   1. provision the scanner through the SHIPPED path (sync door mocked at the network layer),
//      seed code B (fixture 1) into the LOCAL replicas, hold the server lookup route;
//   2. scan never-seen code A (online → the resolver asks the server; the route holds);
//   3. while it waits, pick a PHOTO of code B (#scan-file → onFilePicked → doScan(B));
//   4. release A's lookup as a live row.
// PREMISE: A's offer card renders with NO #ms-order (the submit machine is still `resolving`).
// CONTROL: the same scan with no photo picked renders A's offer WITH #ms-order.
//
// Each test PASSES when its expectation holds — so a green run means the stuck state
// REPRODUCES (test 1) and the control is healthy (test 2). The spike script reads both.
const { test, expect } = require('@playwright/test');
const path = require('path');
const crypto = require('crypto');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const MINT_SUB = 'spike-j1-mint-sub';

const FIXTURE_1_TOKEN_HASH = 'c5a1641409efd198e5a55417f209eda33500fd199f1fa7fa0d8a2567ee1f9680'; // sha256("card1-test-code-fixture-1") — tests/fixtures/qr-fixture-1.png
const svHash = (token) => crypto.createHash('sha256').update(token).digest('hex');
const svPayload = (token) => `https://hq.yumyums.kitchen/r/${token}`;

function fixture1Row() {
  return {
    id: 'c0000000-0000-4000-8000-000000000001',
    token_hash: FIXTURE_1_TOKEN_HASH,
    campaign_id: 'a0000000-0000-4000-8000-000000000001',
    expires_at: '2028-01-01T00:00:00.000Z',
    redeemed_at: null,
    redeemed_by: null,
    updated_at: '2026-09-01T00:00:00.000Z',
  };
}
function campaignLowRow() {
  return { id: 'a0000000-0000-4000-8000-000000000001', name: 'Free side of wings', requires_online: false, updated_at: '2026-09-01T00:00:00.000Z' };
}
function svServerRow(id) {
  return { id, campaign_id: 'a0000000-0000-4000-8000-000000000001', expires_at: '2028-01-01T00:00:00+00:00', redeemed_at: null, redeemed_by: null };
}
const isLookupUrl = (url) => url.pathname.endsWith('/sync/rest/codes') && url.searchParams.has('token_hash');

async function loginAs(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL((url) => !url.pathname.includes('login'));
}

// The sync door, mocked at the network layer the way tests/marketing.spec.js does it.
async function mockSyncTransports(page) {
  await page.route('**/api/v1/sync/token', async (route) => {
    await route.fulfill({
      status: 200, contentType: 'application/json',
      body: JSON.stringify({ token: 'spike-bridge-token', expires_at: Math.floor(Date.now() / 1000) + 900, sub: MINT_SUB, role: 'authenticated', grants: ['marketing'] }),
    });
  });
  await page.route('**/sync/rest/**', async (route) => {
    const req = route.request();
    const u = new URL(req.url());
    if (req.method() === 'GET' && u.pathname.endsWith('/codes')) {
      // the pull batch (no token_hash filter) — empty; lookups are routed separately below
      return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    }
    if (req.method() === 'GET' && u.pathname.endsWith('/campaigns')) {
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([campaignLowRow()]) });
    }
    if (req.method() === 'POST' && u.pathname.endsWith('/scan_attempts')) {
      return route.fulfill({ status: 201, contentType: 'application/json', body: '' });
    }
    return route.fulfill({ status: 503, body: 'unmocked sync transport' });
  });
}

async function openProvisionedScanner(page) {
  await mockSyncTransports(page);
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  await page.goto('/marketing.html');
  await page.waitForFunction(() =>
    window.MarketingScan && window.MarketingScan.booted === true
    && window.MarketingSubmit && window.MarketingSubmit.booted === true);
  await page.waitForFunction(() => document.getElementById('scan-status').textContent.includes('Replica synced'));
  await expect(page.locator('#scan-conn')).toHaveAttribute('data-conn', 'online');
  // Code B (the photo) is HELD locally — the mid-wait scan resolves instantly from the replica.
  await page.evaluate(async (d) => {
    const MS = window.MarketingScan;
    await MS.collections.codes.upsert(d.code);
    await MS.collections.offers.upsert(d.code);
    if (MS.collections.campaigns) await MS.collections.campaigns.upsert(d.campaign);
  }, { code: fixture1Row(), campaign: campaignLowRow() });
}

// Hold every lookup (never-seen code A asks the server) until the test releases it.
async function holdLookups(page) {
  const held = [];
  await page.route(isLookupUrl, (route) => { held.push(route); });
  return held;
}

async function scanInBackground(page, payload) {
  await page.evaluate((p) => { window.__psScan = window.MarketingScan.scanText(p); }, payload);
}

test.describe('spike J1 — photo pick during a held server lookup (B-475)', () => {

  test('[PS-SPIKE-1] PREMISE: photo of code B picked while A waits → A renders offerReady with NO #ms-order (stuck)', async ({ page }) => {
    await openProvisionedScanner(page);
    const held = await holdLookups(page);
    const tokenA = 'spike-j1-never-seen-code-a';
    await scanInBackground(page, svPayload(tokenA));
    await expect.poll(() => held.length, { timeout: 5000 }).toBe(1);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'checkingServer');

    // Mid-wait: the crew picks a photo of code B.
    await page.setInputFiles('#scan-file', path.join(__dirname, 'fixtures', 'qr-fixture-1.png'));
    // The F6 gate refuses B — "Finish the current customer first" — while A is still resolving.
    await expect(page.locator('#scan-prompt')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('#scan-prompt .ms-head')).toHaveText('Finish the current customer first');

    // The server answers A: live.
    await held[0].fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([svServerRow('c0000000-0000-4000-8000-0000000000a1')]) });
    await page.evaluate(() => window.__psScan);
    await expect(result).toHaveAttribute('data-kind', 'offerReady');
    await expect(result).toHaveAttribute('data-source', 'server');

    // Give the submit machine every chance to mount its slot — then assert it did not.
    await page.waitForTimeout(1500);
    const diag = await page.evaluate(() => ({
      msOrder: document.querySelectorAll('#ms-order').length,
      slot: document.querySelectorAll('#scan-submit-slot').length,
      prompt: !!document.getElementById('scan-prompt'),
      kind: document.getElementById('scan-result').getAttribute('data-kind'),
    }));
    console.log('SPIKE-PS-1 after release: ' + JSON.stringify(diag));
    await expect(page.locator('#ms-order'), 'the stuck state: offer shown, no order-number field').toHaveCount(0);
    await expect(page.locator('[data-action="ms-submit"]')).toHaveCount(0);
  });

  test('[PS-SPIKE-2] CONTROL: the same held-then-released scan with no photo picked → A renders WITH #ms-order', async ({ page }) => {
    await openProvisionedScanner(page);
    const held = await holdLookups(page);
    const tokenA = 'spike-j1-never-seen-code-a-control';
    await scanInBackground(page, svPayload(tokenA));
    await expect.poll(() => held.length, { timeout: 5000 }).toBe(1);
    await expect(page.locator('#scan-result')).toHaveAttribute('data-kind', 'checkingServer');
    await held[0].fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([svServerRow('c0000000-0000-4000-8000-0000000000a2')]) });
    await page.evaluate(() => window.__psScan);
    await expect(page.locator('#scan-result')).toHaveAttribute('data-kind', 'offerReady');
    await expect(page.locator('#ms-order')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#scan-prompt')).toHaveCount(0);
  });
});
