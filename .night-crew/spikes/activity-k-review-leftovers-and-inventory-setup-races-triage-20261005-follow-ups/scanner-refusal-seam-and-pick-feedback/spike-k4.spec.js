// spike-k4.spec.js — THROWAWAY spec for spike 01 of card K4 (scanner-refusal-seam-and-pick-
// feedback, B-482 / B-483). Copied by the spike script into a worktree's tests/ directory and
// run there; never part of the suite. Scaffolding re-stated from the Activity J spike spec
// (the sync door mocked at the network layer; everything else is the SHIPPED page).
//
//   [K4-SPIKE-1] B-483 PREMISE: a photo picked while code A's lookup is held is refused by the
//                J1 guard with NO feedback — result text identical before/after, no prompt,
//                no "pick again"/"Finish checking" copy anywhere.
//   [K4-SPIKE-2] B-482 PREMISE: patching window.MarketingScan.campaignPolicy.policyFor to
//                throw AFTER boot does not change an offline scan's result kind — the page
//                captured the function once at boot, so no post-boot seam reaches the refusal.
//   [K4-SPIKE-3] CONTROL for 2: the same offline scan with no patch — records the kind the
//                premise compares against.
const { test, expect } = require('@playwright/test');
const path = require('path');
const crypto = require('crypto');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const MINT_SUB = 'spike-k4-mint-sub';
const FIXTURE_1_TOKEN_HASH = 'c5a1641409efd198e5a55417f209eda33500fd199f1fa7fa0d8a2567ee1f9680'; // sha256("card1-test-code-fixture-1")
const FIXTURE_1_PAYLOAD = 'https://hq.yumyums.kitchen/r/card1-test-code-fixture-1';
const svPayload = (token) => `https://hq.yumyums.kitchen/r/${token}`;

function fixture1Row() {
  return { id: 'c0000000-0000-4000-8000-000000000001', token_hash: FIXTURE_1_TOKEN_HASH, campaign_id: 'a0000000-0000-4000-8000-000000000001', expires_at: '2028-01-01T00:00:00.000Z', redeemed_at: null, redeemed_by: null, updated_at: '2026-09-01T00:00:00.000Z' };
}
function campaignLowRow() {
  return { id: 'a0000000-0000-4000-8000-000000000001', name: 'Free side of wings', requires_online: false, updated_at: '2026-09-01T00:00:00.000Z' };
}
const isLookupUrl = (url) => url.pathname.endsWith('/sync/rest/codes') && url.searchParams.has('token_hash');

async function loginAs(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL((url) => !url.pathname.includes('login'));
}
async function mockSyncTransports(page) {
  await page.route('**/api/v1/sync/token', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/json',
      body: JSON.stringify({ token: 'spike-bridge-token', expires_at: Math.floor(Date.now() / 1000) + 900, sub: MINT_SUB, role: 'authenticated', grants: ['marketing'] }) });
  });
  await page.route('**/sync/rest/**', async (route) => {
    const req = route.request(); const u = new URL(req.url());
    if (req.method() === 'GET' && u.pathname.endsWith('/codes')) return route.fulfill({ status: 200, contentType: 'application/json', body: '[]' });
    if (req.method() === 'GET' && u.pathname.endsWith('/campaigns')) return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([campaignLowRow()]) });
    if (req.method() === 'POST' && u.pathname.endsWith('/scan_attempts')) return route.fulfill({ status: 201, contentType: 'application/json', body: '' });
    return route.fulfill({ status: 503, body: 'unmocked sync transport' });
  });
}
async function openProvisionedScanner(page) {
  await mockSyncTransports(page);
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  await page.goto('/marketing.html');
  await page.waitForFunction(() => window.MarketingScan && window.MarketingScan.booted === true && window.MarketingSubmit && window.MarketingSubmit.booted === true);
  await page.waitForFunction(() => document.getElementById('scan-status').textContent.includes('Replica synced'));
  await expect(page.locator('#scan-conn')).toHaveAttribute('data-conn', 'online');
  await page.evaluate(async (d) => {
    const MS = window.MarketingScan;
    await MS.collections.codes.upsert(d.code);
    await MS.collections.offers.upsert(d.code);
    if (MS.collections.campaigns) await MS.collections.campaigns.upsert(d.campaign);
  }, { code: fixture1Row(), campaign: campaignLowRow() });
}
async function holdLookups(page) { const held = []; await page.route(isLookupUrl, (route) => { held.push(route); }); return held; }
async function scanInBackground(page, payload) { await page.evaluate((p) => { window.__k4Scan = window.MarketingScan.scanText(p); }, payload); }
async function pageTextHas(page, needle) { return page.evaluate((n) => document.body.innerText.toLowerCase().includes(n.toLowerCase()), needle); }

test.describe('spike K4 — refused pick feedback (B-483) and the policy seam (B-482)', () => {

  test('[K4-SPIKE-1] B-483 PREMISE: a photo picked mid-check is refused silently', async ({ page }) => {
    await openProvisionedScanner(page);
    const held = await holdLookups(page);
    await scanInBackground(page, svPayload('spike-k4-never-seen-code-a'));
    await expect.poll(() => held.length, { timeout: 5000 }).toBe(1);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'checkingServer');
    const before = (await result.innerText()).trim();
    await page.setInputFiles('#scan-file', path.join(__dirname, 'fixtures', 'qr-fixture-1.png'));
    await page.waitForTimeout(1500);
    const after = (await result.innerText()).trim();
    const diag = {
      kind: await result.getAttribute('data-kind'), prompt: await page.locator('#scan-prompt').count(),
      textUnchanged: before === after, pickAgain: await pageTextHas(page, 'pick again'), finishChecking: await pageTextHas(page, 'Finish checking'),
      note: await page.locator('#scan-note').count(),
    };
    console.log('SPIKE-K4-1: ' + JSON.stringify(diag));
    expect(diag.kind).toBe('checkingServer');
    expect(diag.prompt).toBe(0);
    expect(diag.textUnchanged, 'the refused pick changed nothing on screen').toBe(true);
    expect(diag.pickAgain || diag.finishChecking, 'no feedback copy exists today').toBe(false);
    await held[0].fulfill({ status: 200, contentType: 'application/json', body: '[]' });
  });

  test('[K4-SPIKE-3] CONTROL: offline scan of a held low-value code, no patch', async ({ page }) => {
    await openProvisionedScanner(page);
    await page.context().setOffline(true);
    await page.evaluate(async (p) => { await window.MarketingScan.scanText(p); }, FIXTURE_1_PAYLOAD);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', /offerReady|refused|requiresOnline/);
    const diag = { kind: await result.getAttribute('data-kind'), refusalCopy: await pageTextHas(page, 'verification is required') };
    console.log('SPIKE-K4-3: ' + JSON.stringify(diag));
  });

  test('[K4-SPIKE-2] B-482 PREMISE: policyFor patched to throw AFTER boot → the same result as the control', async ({ page }) => {
    await openProvisionedScanner(page);
    const patched = await page.evaluate(() => {
      const MS = window.MarketingScan;
      if (!MS.campaignPolicy || typeof MS.campaignPolicy.policyFor !== 'function') return false;
      MS.campaignPolicy.policyFor = () => { throw new Error('spike-k4-throw'); };
      return true;
    });
    expect(patched, 'window.MarketingScan.campaignPolicy.policyFor exists to patch').toBe(true);
    await page.context().setOffline(true);
    await page.evaluate(async (p) => { await window.MarketingScan.scanText(p); }, FIXTURE_1_PAYLOAD);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', /offerReady|refused|requiresOnline|unknownCode/);
    const diag = { kind: await result.getAttribute('data-kind'), refusalCopy: await pageTextHas(page, 'verification is required') };
    console.log('SPIKE-K4-2: ' + JSON.stringify(diag));
    // Premise: the throw never reached the page — the offer still renders, no refusal copy.
    expect(diag.kind).toBe('offerReady');
    expect(diag.refusalCopy).toBe(false);
  });
});
