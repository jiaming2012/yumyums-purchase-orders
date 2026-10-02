const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

// states-marketing-campaigns.spec.js — the CLAUDE.md self-verification ritual
// for the Campaigns section (marketing.html #s2, card h2-campaigns-tab-ui,
// run 20261002). Every row of the card's State Enumeration Table is FORCED
// here, screenshotted, and the PNGs are read back with the multimodal Read
// tool; the observations are reported in the card's return, not the intentions.
//
// ┌─ STATE ENUMERATION TABLE ───────────────────────────────────────────────┐
// │ Row          Trigger                        Visual contract            │
// ├─────────────────────────────────────────────────────────────────────────┤
// │ empty        FIXTURE {campaigns:[]}         "No campaigns yet" card +  │
// │                                             New campaign still enabled │
// │ loading      FIXTURE, 1.5s delay            3 skeleton cards,          │
// │                                             aria-busy, no numbers      │
// │ error        FIXTURE 500                     red banner "Couldn't load  │
// │                                             campaigns" + Retry (UI-R6) │
// │ success      REAL create + REAL list        funnel legs + money strip  │
// │                                             + "Per $1 —" pill          │
// │ locked       REAL team_member → 403          "Managers only" card, no   │
// │              managers_only                  list, no create            │
// │ offline      REAL list then                 warn banner "Last synced   │
// │              context.setOffline(true)       <time>", cards still on    │
// │                                             screen, create DISABLED    │
// │ not          REAL create (HQ_SYNC_REST_URL  amber "Not on tablets yet" │
// │ projected    unset → projected_at null)     pill on the card + sheet   │
// │ long         REAL create, 94-char           clamped to 40 chars + "…"  │
// │ content      offer_text                     + "more"; tap shows all    │
// └─────────────────────────────────────────────────────────────────────────┘
//
// Which rows ride a fixture and which hit the real endpoint is stated in
// merge-intents/h2-campaigns-tab-ui.md and is binding for G6: 5 of 8 are real.
// empty / loading / error ride `page.route` because the real endpoint cannot
// produce those conditions on demand (the e2e database is shared across the
// suite and the functional spec sorts before this file).

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const USER_PASSWORD = 'test456';
const SHOT_DIR = path.join(__dirname, '..', 'test-results', 'states-marketing-campaigns');
fs.mkdirSync(SHOT_DIR, { recursive: true });
test.use({ viewport: { width: 393, height: 852 } });

async function shot(page, name) {
  await page.screenshot({ path: path.join(SHOT_DIR, name + '.png'), fullPage: true });
}
async function loginAs(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}
async function makeUser(page, tag, roles) {
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  const email = `st-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  const invite = await page.evaluate(async ([em, rs]) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'States', last_name: 'Tester', email: em, roles: rs }),
    });
    return res.json();
  }, [email, roles]);
  const token = (invite.invite_path || '').split('token=')[1];
  expect(token, 'invite token').toBeTruthy();
  await page.evaluate(async ([t, pw]) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: pw }),
    });
  }, [token, USER_PASSWORD]);
  return { email };
}
async function createCampaign(page, body) {
  const out = await page.evaluate(async (b) => {
    const res = await fetch('/api/v1/marketing/campaigns', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(b),
    });
    return { status: res.status, body: await res.json() };
  }, body);
  expect(out.status, JSON.stringify(out.body)).toBe(201);
  return out.body;
}
// open navigates to the Campaigns section. `#tab=2` is the module's own deep
// link (it calls the page's show(), it does not change it).
async function open(page) {
  await page.goto('/marketing.html#tab=2');
  await expect(page.locator('#mc-root')).toBeVisible();
}
async function settled(page) {
  await page.waitForFunction(() => {
    const r = document.getElementById('mc-root');
    return r && r.dataset.status !== 'loading' && r.dataset.status !== 'idle';
  });
}

const json = body => route => route.fulfill({
  status: 200, contentType: 'application/json', body: JSON.stringify(body),
});

test.describe('Campaigns (#s2) — State Enumeration Table', () => {

  test('Row: empty — "No campaigns yet", and New campaign is still the way forward', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    // FIXTURE: a shared e2e database cannot be guaranteed campaign-free.
    await page.route('**/api/v1/marketing/campaigns?*', json({ campaigns: [] }));
    await open(page);
    await settled(page);
    await expect(page.locator('#mc-root')).toHaveAttribute('data-status', 'empty');
    await expect(page.locator('#mc-empty')).toContainText('No campaigns yet');
    await expect(page.locator('#mc-new')).toBeEnabled();
    await expect(page.locator('#mc-list')).toHaveCount(0);
    await shot(page, 'empty');
  });

  test('Row: loading — three skeletons, aria-busy, and no number on screen yet', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    // FIXTURE: the real endpoint cannot be made slow on demand.
    await page.route('**/api/v1/marketing/campaigns?*', async route => {
      await new Promise(r => setTimeout(r, 1500));
      await json({ campaigns: [] })(route);
    });
    await open(page);
    await expect(page.locator('#mc-root')).toHaveAttribute('data-status', 'loading');
    await expect(page.locator('#mc-list')).toHaveAttribute('aria-busy', 'true');
    await expect(page.locator('#mc-list .mc-skel')).toHaveCount(3);
    // A skeleton must not be a zero: no funnel or money text while loading.
    await expect(page.locator('#mc-view-list')).not.toContainText('Scans');
    await expect(page.locator('#mc-view-list')).not.toContainText('Per $1');
    await shot(page, 'loading');
  });

  test('Row: error — loud banner naming the failure, with Retry (UI-R6)', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    // FIXTURE: a real 500 is not producible on demand.
    await page.route('**/api/v1/marketing/campaigns?*', route => route.fulfill({
      status: 500, contentType: 'application/json', body: '{"error":"boom"}',
    }));
    await open(page);
    await settled(page);
    await expect(page.locator('#mc-root')).toHaveAttribute('data-status', 'error');
    await expect(page.locator('#mc-banner')).toContainText('Couldn’t load campaigns');
    await expect(page.locator('#mc-banner')).toContainText('boom');
    await expect(page.locator('#mc-banner [data-action="reload"]')).toContainText('Retry');
    // "No campaigns yet" is a FACT the server stated; a failed load has not
    // stated it. The first read-back of this screenshot showed the red banner
    // and the empty card together, which reads as "it failed AND you have
    // none" — two different claims, one of them unfounded.
    await expect(page.locator('#mc-empty')).toHaveCount(0);
    await expect(page.locator('#mc-list')).toHaveCount(0);
    await expect(page.locator('#mc-view-list')).not.toContainText('No campaigns yet');
    await shot(page, 'error');
    await page.emulateMedia({ colorScheme: 'dark' });
    await shot(page, 'error-dark');
  });

  test('Row: success — funnel legs and the money strip, from the REAL endpoint', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const made = await createCampaign(page, {
      name: 'Taco Tuesday',
      offer_text: '$2 off any taco plate',
      face_value_cents: 200,
      runs_days: 21,
      channels: [{ channel: 'truck_sign' }, { channel: 'instagram' }, { channel: 'flyer' }],
    });
    await open(page);
    await settled(page);
    const card = page.locator(`.mc-card[data-id="${made.campaign.id}"]`);
    await expect(card.locator('.mc-name')).toHaveText('Taco Tuesday');
    await expect(card.locator('.mc-funnel')).toContainText('Scans');
    await expect(card.locator('[data-m="revenue"] .mc-v')).toHaveText('$0.00');
    await expect(card.locator('.mc-per')).toHaveText('Per $1 —');
    await expect(card.locator('.mc-meta')).toContainText('3 codes');
    await shot(page, 'success');
    await page.emulateMedia({ colorScheme: 'dark' });
    await shot(page, 'success-dark');
  });

  test('Row: locked — a real team_member gets 403 managers_only and the Managers only card', async ({ page }) => {
    const member = await makeUser(page, 'locked', ['team_member']);
    await loginAs(page, member.email, USER_PASSWORD);
    const refused = await page.evaluate(async () => {
      const res = await fetch('/api/v1/marketing/campaigns');
      return { status: res.status, body: await res.json() };
    });
    expect(refused.status).toBe(403);
    expect(refused.body.error).toBe('managers_only');
    await open(page);
    await settled(page);
    await expect(page.locator('#mc-root')).toHaveAttribute('data-status', 'locked');
    await expect(page.locator('#mc-locked')).toContainText('Managers only');
    await expect(page.locator('#mc-new')).toHaveCount(0);
    await expect(page.locator('#mc-list')).toHaveCount(0);
    // No Retry here on purpose: a permission refusal is not a thing to retry.
    await expect(page.locator('#mc-banner')).toBeHidden();
    await shot(page, 'locked');
  });

  test('Row: offline — the last-synced list stays on screen and create is disabled', async ({ page, context }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const made = await createCampaign(page, {
      name: 'Rainy Day Combo',
      offer_text: '$4 off a combo',
      face_value_cents: 400,
      runs_days: 10,
      channels: [{ channel: 'table_tent' }],
    });
    await open(page);
    await settled(page);
    await expect(page.locator(`.mc-card[data-id="${made.campaign.id}"]`)).toBeVisible();
    // REAL offline: nothing is mocked, the browser refuses the request.
    await context.setOffline(true);
    await page.click('#mc-refresh');
    await expect(page.locator('#mc-root')).toHaveAttribute('data-status', 'offline');
    await expect(page.locator('#mc-banner')).toContainText('Last synced');
    await expect(page.locator('#mc-new')).toBeDisabled();
    await expect(page.locator(`.mc-card[data-id="${made.campaign.id}"] .mc-name`))
      .toHaveText('Rainy Day Combo');
    await shot(page, 'offline');
    await context.setOffline(false);
  });

  test('Row: not projected — projected_at null renders the "Not on tablets yet" pill', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    // REAL: HQ_SYNC_REST_URL is unset on the test stack, so H1's handler
    // genuinely leaves projected_at NULL and warns not_projected.
    const made = await createCampaign(page, {
      name: 'Not On Tablets',
      offer_text: '$6 off a family pack',
      face_value_cents: 600,
      runs_days: 14,
      channels: [{ channel: 'google_ads' }],
    });
    expect(made.campaign.projected_at).toBeNull();
    expect(made.warnings).toContain('not_projected');
    await open(page);
    await settled(page);
    const card = page.locator(`.mc-card[data-id="${made.campaign.id}"]`);
    await expect(card.locator('.mc-pill-np')).toContainText('Not on tablets yet');
    await shot(page, 'not-projected');
    // The same fact is projected on the detail header (UI-R7).
    await card.locator('[data-action="open-detail"]').click();
    await expect(page.locator('#mc-view-detail .mc-pill-np')).toContainText('Not on tablets yet');
    await shot(page, 'not-projected-detail');
  });

  test('Row: long content — a 94-char offer clamps with "more" and opens in full on tap', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const LONG = 'Two dollars off any six piece wings, every Wednesday, dine in or take out, no limit';
    expect(LONG.length).toBeGreaterThan(40);
    const made = await createCampaign(page, {
      name: 'Wing Wednesday All Day Long Name',
      offer_text: LONG,
      face_value_cents: 200,
      runs_days: 30,
      channels: [{ channel: 'other', channel_label: 'Neighbourhood noticeboard' }],
    });
    await open(page);
    await settled(page);
    const card = page.locator(`.mc-card[data-id="${made.campaign.id}"]`);
    const offer = card.locator('.mc-offer');
    await expect(offer).toContainText('…');
    await expect(offer.locator('.mc-more')).toHaveText('more');
    const clamped = await offer.innerText();
    expect(clamped.length, 'clamped shorter than the full offer').toBeLessThan(LONG.length);
    expect(clamped).not.toContain('no limit');
    await shot(page, 'long-content');
    await offer.click();
    await expect(offer).toContainText('no limit');
    await expect(offer.locator('.mc-more')).toHaveText('less');
    // No horizontal scroll at 393px even with the longest content on screen.
    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow).toBeLessThanOrEqual(1);
    await shot(page, 'long-content-expanded');
  });

  test('Edge: 393px — every tappable row clears 44px and the code sheet fits', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const made = await createCampaign(page, {
      name: 'Touch Target Check',
      offer_text: '$1 off a drink',
      face_value_cents: 100,
      runs_days: 7,
      channels: [{ channel: 'receipt' }, { channel: 'sms' }],
    });
    await open(page);
    await settled(page);
    await page.locator(`.mc-card[data-id="${made.campaign.id}"] [data-action="open-detail"]`).click();
    await expect(page.locator('#mc-codes .mc-code-row').first()).toBeVisible();
    const heights = await page.locator('#mc-codes .mc-code-row').evaluateAll(
      rs => rs.map(r => r.getBoundingClientRect().height));
    expect(heights.length).toBe(2);
    for (const h of heights) expect(h).toBeGreaterThanOrEqual(44);
    await shot(page, 'edge-detail-393');
    await page.locator('#mc-codes .mc-code-row').first().click();
    await expect(page.locator('#mc-qr')).toBeVisible();
    await expect.poll(() => page.locator('#mc-qr').evaluate(el => el.naturalWidth))
      .toBeGreaterThan(0);
    const qrBox = await page.locator('#mc-qr').boundingBox();
    expect(qrBox.width).toBeLessThanOrEqual(393);
    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow).toBeLessThanOrEqual(1);
    await shot(page, 'edge-code-sheet-393');
    await page.emulateMedia({ colorScheme: 'dark' });
    await shot(page, 'edge-code-sheet-dark');
  });

  test('Edge: the create sheet with channels picked — payload preview and the counted button', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await open(page);
    await settled(page);
    await page.click('#mc-new');
    await expect(page.locator('#mc-view-create')).toBeVisible();
    await shot(page, 'edge-create-blank');
    await page.fill('#mc-f-name', 'Flyer Drop');
    await page.fill('#mc-f-offer', 'Free side with any sandwich');
    await page.fill('#mc-f-value', '3.50');
    await page.click('#mc-chips [data-channel="flyer"]');
    await page.click('#mc-chips [data-channel="truck_sign"]');
    await expect(page.locator('#mc-submit')).toContainText('Create campaign + 2 codes');
    await expect(page.locator('#mc-preview .mc-preview-row')).toHaveCount(2);
    await shot(page, 'edge-create-filled');
    await page.click('#mc-submit');
    await expect(page.locator('#mc-ready-head')).toContainText('2 codes ready');
    // The not-projected sentence on this screen is a BLOCK pill, and .mc-pill
    // is nowrap — the first read-back of this screenshot showed it clipped at
    // the card edge mid-word. Assert it wraps instead of overflowing.
    const np = page.locator('#mc-view-ready .mc-pill-block');
    await expect(np).toContainText('Not on tablets yet');
    const clipped = await np.evaluate(el => el.scrollWidth - el.clientWidth);
    expect(clipped, 'the not-projected sentence must wrap, not clip').toBeLessThanOrEqual(1);
    await shot(page, 'edge-codes-ready');
  });
});
