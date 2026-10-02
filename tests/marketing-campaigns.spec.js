// ═══════════════════════════════════════════════════════════════════════════
// Campaigns section of marketing.html (#s2) — card h2-campaigns-tab-ui
// (run 20261002, Activity H / roadmap H2, design of record Current Campaigns 1–8)
// ═══════════════════════════════════════════════════════════════════════════
//
// RED-FIRST (greenfield): every test below was written and RUN against the
// pre-change tree, where #s2 is still the "Soon" placeholder and
// marketing/campaigns.js does not exist. The red is a missing-element timeout
// on `#mc-root` / `#mc-new` / `#mc-locked`. Evidence:
// .night-crew/runs/2026-10-02-autonomous/logs/h2/RF-red-first.log and the
// ## Red-first section of merge-intents/h2-campaigns-tab-ui.md.
//
// The card's done_when rows:
//   [MC-01] create sheet mints N codes and lands on "N codes ready"
//   [MC-02] list renders funnel + money strip
//   [MC-03] code sheet Share calls navigator.share with the PNG file
//           (headless chromium exposes NEITHER navigator.share NOR
//            navigator.canShare — spike web-share-files-enumerated, chromium
//            row. The call can therefore only be observed by installing the
//            API in the page; the companion test asserts the UN-STUBBED
//            fallback beside it, which is the real browser's real branch.)
//   [MC-04] team_member sees the Locked state (real 403 managers_only)
//   [MC-05] offline shows the last-synced list + disabled create
//
// Endpoints are H1's REAL ones (`/api/v1/marketing/*`, merged at 40b5846).
// Nothing in this file mocks a marketing route; the only page-level stub is
// navigator.share/canShare in [MC-03a].

const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const USER_PASSWORD = 'test456';

async function loginAs(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// makeUser — same shape as tests/marketing.spec.js (this suite's
// per-file-helper convention). Leaves the browser logged in as ADMIN.
async function makeUser(page, tag, roles) {
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  const email = `mc-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  const invite = await page.evaluate(async ([em, rs]) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Camp', last_name: 'Tester', email: em, roles: rs }),
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
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  return { id: String(invite.user.id), email };
}

// createCampaign mints a campaign through the REAL endpoint, from the page's
// own session. Returns the 201 body ({campaign, codes, warnings?}).
async function createCampaign(page, body) {
  const out = await page.evaluate(async (b) => {
    const res = await fetch('/api/v1/marketing/campaigns', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(b),
    });
    return { status: res.status, body: await res.json() };
  }, body);
  expect(out.status, `POST /campaigns -> ${JSON.stringify(out.body)}`).toBe(201);
  return out.body;
}

// openCampaigns gets to the Campaigns section and waits for the list to settle.
async function openCampaigns(page) {
  await page.goto('/marketing.html');
  await page.click('#t2');
  await expect(page.locator('#mc-root')).toBeVisible();
  await page.waitForFunction(() => {
    const r = document.getElementById('mc-root');
    return r && r.dataset.status && r.dataset.status !== 'loading' && r.dataset.status !== 'idle';
  });
}

test.describe('Campaigns section (#s2) — card h2-campaigns-tab-ui', () => {

  // ─────────────────────────────────────────────────────────────────────────
  test('[MC-01] create sheet mints N codes and lands on "N codes ready"', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await openCampaigns(page);

    await page.click('#mc-new');
    await expect(page.locator('#mc-view-create')).toBeVisible();

    await page.fill('#mc-f-name', 'Wing Wednesday MC01');
    await page.fill('#mc-f-offer', '$2 off any 6pc wings');
    await page.fill('#mc-f-value', '2.00');
    await page.fill('#mc-f-days', '14');

    // Channels are chips, not a <select> (Current Campaigns 2).
    await page.click('#mc-chips [data-channel="truck_sign"]');
    await page.click('#mc-chips [data-channel="flyer"]');
    await page.click('#mc-chips [data-channel="instagram"]');

    // The payload preview is visible on the sheet BEFORE the save.
    await expect(page.locator('#mc-preview')).toContainText('hq.yumyums.kitchen/q/');
    // The button counts the codes the save will mint.
    await expect(page.locator('#mc-submit')).toContainText('Create campaign + 3 codes');

    await page.click('#mc-submit');

    await expect(page.locator('#mc-view-ready')).toBeVisible();
    await expect(page.locator('#mc-ready-head')).toContainText('3 codes ready');
    const rows = page.locator('#mc-ready-list .mc-ready-row');
    await expect(rows).toHaveCount(3);
    // Each row names its channel and its 6-char short code — content, not a
    // container (UI-R5).
    await expect(rows.nth(0)).toContainText('Truck sign');
    await expect(rows.nth(1)).toContainText('Flyer');
    await expect(rows.nth(2)).toContainText('Instagram');
    for (let i = 0; i < 3; i++) {
      const short = await rows.nth(i).locator('.mc-short').innerText();
      expect(short.trim(), `code ${i} short`).toMatch(/^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{6}$/);
    }
  });

  // ─────────────────────────────────────────────────────────────────────────
  test('[MC-02] list renders funnel + money strip', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const made = await createCampaign(page, {
      name: 'Funnel Strip MC02',
      offer_text: '$3 off a combo',
      face_value_cents: 300,
      runs_days: 21,
      channels: [{ channel: 'flyer' }, { channel: 'sms' }],
    });
    await openCampaigns(page);

    const card = page.locator(`.mc-card[data-id="${made.campaign.id}"]`);
    await expect(card).toBeVisible();
    await expect(card.locator('.mc-name')).toHaveText('Funnel Strip MC02');

    // Funnel — the three legs, labelled, with the values the API serves
    // (H1 ships scans real and signups/redeemed as a stated 0).
    const funnel = card.locator('.mc-funnel');
    await expect(funnel).toContainText('Scans');
    await expect(funnel).toContainText('Signups');
    await expect(funnel).toContainText('Redeemed');
    await expect(funnel.locator('[data-leg="scans"] .mc-n')).toHaveText(
      String(made.campaign.funnel.scans));
    await expect(funnel.locator('[data-leg="signups"] .mc-n')).toHaveText(
      String(made.campaign.funnel.signups));
    await expect(funnel.locator('[data-leg="redeemed"] .mc-n')).toHaveText(
      String(made.campaign.funnel.redeemed));

    // Money strip — revenue / discount / net + the Per $1 pill, rendered from
    // whatever `money` block the endpoint ships (H1's zero shape today).
    const money = card.locator('.mc-money');
    await expect(money).toContainText('Revenue');
    await expect(money).toContainText('Discount');
    await expect(money).toContainText('Net');
    await expect(money.locator('[data-m="revenue"] .mc-v')).toHaveText('$0.00');
    await expect(money.locator('[data-m="discount"] .mc-v')).toHaveText('$0.00');
    await expect(money.locator('[data-m="net"] .mc-v')).toHaveText('$0.00');
    // per_dollar is null in the zero shape — "no opinion" renders as an em
    // dash, never as $0.00 (UI-R3: a null must not coerce to a number).
    expect(made.campaign.money.per_dollar).toBeNull();
    await expect(card.locator('.mc-per')).toHaveText('Per $1 —');
  });

  // ─────────────────────────────────────────────────────────────────────────
  test('[MC-03a] code sheet Share calls navigator.share with the PNG file (API installed by the test)', async ({ page }) => {
    // Headless chromium exposes neither navigator.share nor navigator.canShare
    // (spike web-share-files-enumerated). Installing them is the ONLY way to
    // observe the call; the un-stubbed fallback is [MC-03b] below.
    await page.addInitScript(() => {
      window.__shared = [];
      Object.defineProperty(navigator, 'canShare', {
        configurable: true,
        value: (data) => !!(data && Array.isArray(data.files) && data.files.length),
      });
      Object.defineProperty(navigator, 'share', {
        configurable: true,
        value: async (data) => {
          window.__shared.push({
            title: data.title || '',
            text: data.text || '',
            files: (data.files || []).map(f => ({ name: f.name, type: f.type, size: f.size })),
          });
        },
      });
    });

    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const made = await createCampaign(page, {
      name: 'Share Sheet MC03a',
      offer_text: 'Free side with any sandwich',
      face_value_cents: 400,
      runs_days: 7,
      channels: [{ channel: 'table_tent' }],
    });
    await openCampaigns(page);

    await page.click(`.mc-card[data-id="${made.campaign.id}"] [data-action="open-detail"]`);
    await expect(page.locator('#mc-view-detail')).toBeVisible();
    await page.click('#mc-codes .mc-code-row >> nth=0');
    await expect(page.locator('#mc-view-code')).toBeVisible();

    // The big QR is the PNG endpoint, and it actually decoded as an image.
    const qr = page.locator('#mc-qr');
    await expect(qr).toBeVisible();
    await expect(qr).toHaveAttribute('src', /\/api\/v1\/marketing\/codes\/.*\.png/);
    await expect.poll(() => qr.evaluate(el => el.naturalWidth)).toBeGreaterThan(0);

    await expect(page.locator('#mc-share')).toBeVisible();
    await page.click('#mc-share');

    await expect.poll(() => page.evaluate(() => window.__shared.length)).toBe(1);
    const call = await page.evaluate(() => window.__shared[0]);
    expect(call.files).toHaveLength(1);
    expect(call.files[0].type).toBe('image/png');
    expect(call.files[0].name).toMatch(/\.png$/);
    expect(call.files[0].size, 'the shared PNG has bytes').toBeGreaterThan(0);
    expect(call.files[0].name).toContain(made.codes[0].short);
  });

  // ─────────────────────────────────────────────────────────────────────────
  test('[MC-03b] with no Web Share API the code sheet renders Save PNG + Copy link (un-stubbed)', async ({ page }) => {
    // NOTHING is installed here. This is the real headless-chromium branch and
    // the one a desktop browser takes.
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const made = await createCampaign(page, {
      name: 'Fallback MC03b',
      offer_text: '$1 off lemonade',
      face_value_cents: 100,
      runs_days: 7,
      channels: [{ channel: 'menu_board' }],
    });
    await openCampaigns(page);
    await page.click(`.mc-card[data-id="${made.campaign.id}"] [data-action="open-detail"]`);
    await page.click('#mc-codes .mc-code-row >> nth=0');
    await expect(page.locator('#mc-view-code')).toBeVisible();

    expect(await page.evaluate(() => typeof navigator.canShare)).toBe('undefined');
    await expect(page.locator('#mc-share')).toHaveCount(0);
    await expect(page.locator('#mc-save')).toBeVisible();
    await expect(page.locator('#mc-save')).toContainText('Save PNG');
    await expect(page.locator('#mc-copy')).toBeVisible();
    await expect(page.locator('#mc-copy')).toContainText('Copy link');
    await expect(page.locator('#mc-print')).toContainText('Print');
    // The payload is readable even when nothing can share it.
    await expect(page.locator('#mc-payload')).toContainText(made.codes[0].payload_url);
  });

  // ─────────────────────────────────────────────────────────────────────────
  test('[MC-04] team_member sees the Locked state', async ({ page }) => {
    const member = await makeUser(page, 'tm', ['team_member']);
    await loginAs(page, member.email, USER_PASSWORD);

    // The real envelope H1 ships, read from the member's own session.
    const refused = await page.evaluate(async () => {
      const res = await fetch('/api/v1/marketing/campaigns');
      return { status: res.status, body: await res.json() };
    });
    expect(refused.status).toBe(403);
    expect(refused.body.error).toBe('managers_only');

    await page.goto('/marketing.html');
    await page.click('#t2');
    await expect(page.locator('#mc-locked')).toBeVisible();
    await expect(page.locator('#mc-locked')).toContainText('Managers only');
    // Locked means locked: no list, no create affordance.
    await expect(page.locator('#mc-new')).toHaveCount(0);
    await expect(page.locator('#mc-list')).toHaveCount(0);
  });

  // ─────────────────────────────────────────────────────────────────────────
  // G6 F1 (P1) — regression. Written BEFORE the fix and confirmed RED
  // (`Expected: "Sandwich board" / Received: "S"`): the `otherLabel` field was
  // routed through the full render() path, which rebuilds the create sheet with
  // innerHTML and destroys the input the thumb is in. Every test this card had
  // written used page.fill(), a ONE-SHOT value set that never re-enters the
  // field — so nothing typed character by character, and nothing caught it.
  // "Other" is the one channel of nine that needs free text, and it was
  // unusable: a manager either shipped a campaign labelled "S" forever or was
  // blocked by the module's own channel_label_required guard.
  //
  // 🛑 pressSequentially is load-bearing here. page.fill() passes on the broken
  // code. Any future test of a text field in this sheet must type, not fill.
  test('[MC-06] typing the "Other" channel label survives more than one keystroke', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await openCampaigns(page);
    await page.click('#mc-new');
    await expect(page.locator('#mc-view-create')).toBeVisible();
    await page.click('#mc-chips [data-channel="other"]');
    await expect(page.locator('#mc-f-other')).toBeVisible();

    await page.locator('#mc-f-other').click();
    await page.locator('#mc-f-other').pressSequentially('Sandwich board', { delay: 20 });
    await expect(page.locator('#mc-f-other')).toHaveValue('Sandwich board');
    // The caret has to still be in the field a thumb was typing into.
    expect(await page.evaluate(() => document.activeElement && document.activeElement.id))
      .toBe('mc-f-other');

    // The other text fields in the same sheet take the same guarantee.
    await page.locator('#mc-f-name').click();
    await page.locator('#mc-f-name').pressSequentially('Board Special', { delay: 15 });
    await expect(page.locator('#mc-f-name')).toHaveValue('Board Special');
    expect(await page.evaluate(() => document.activeElement && document.activeElement.id))
      .toBe('mc-f-name');

    // And the label the manager typed is what actually reaches the server and
    // comes back on the code — not its first letter.
    await page.fill('#mc-f-offer', '$2 off anything');
    await page.fill('#mc-f-value', '2.00');
    await page.click('#mc-submit');
    await expect(page.locator('#mc-ready-head')).toContainText('1 code ready');
    await expect(page.locator('#mc-ready-list .mc-ready-row').first())
      .toContainText('Sandwich board');
  });

  // ─────────────────────────────────────────────────────────────────────────
  // G6 F2 — regression. Written BEFORE the fix and confirmed RED: a failed
  // WRITE (PATCH /campaigns/{id}) set S.detailError, which renderDetail() turns
  // into a whole-view replacement reading "Couldn't load this campaign". The
  // manager lost the Money card and the code rows they were reading, and was
  // told a LOAD had failed, which it had not. Its sibling doToggleCode() was
  // already non-destructive; this aligns them.
  test('[MC-07] a failed Pause reports the write, non-destructively — the detail stays on screen', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const made = await createCampaign(page, {
      name: 'Pause Fails MC07',
      offer_text: '$2 off a wrap',
      face_value_cents: 200,
      runs_days: 9,
      channels: [{ channel: 'flyer' }, { channel: 'sms' }],
    });
    await openCampaigns(page);
    await page.click(`.mc-card[data-id="${made.campaign.id}"] [data-action="open-detail"]`);
    await expect(page.locator('#mc-money')).toBeVisible();
    await expect(page.locator('#mc-codes .mc-code-row')).toHaveCount(2);

    // Only the WRITE fails. The read that built this view already succeeded.
    await page.route('**/api/v1/marketing/campaigns/*', async route => {
      if (route.request().method() === 'PATCH') {
        await route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"boom"}' });
      } else {
        await route.fallback();
      }
    });
    await page.click('#mc-pause-campaign');

    // Loud about the write that failed...
    await expect(page.locator('#mc-detail-note')).toContainText('Could not pause');
    await expect(page.locator('#mc-detail-note')).toContainText('boom');
    // ...and it must NOT claim a load failed, nor take the page away.
    await expect(page.locator('#mc-view-detail')).not.toContainText('Couldn’t load this campaign');
    await expect(page.locator('#mc-money')).toBeVisible();
    await expect(page.locator('#mc-codes .mc-code-row')).toHaveCount(2);
    await expect(page.locator('#mc-detail-name')).toHaveText('Pause Fails MC07');
    // Retry is the button itself, still there and still armed (UI-R6).
    await expect(page.locator('#mc-pause-campaign')).toBeEnabled();
    await expect(page.locator('#mc-pause-campaign')).toContainText('Pause campaign');
  });

  // ─────────────────────────────────────────────────────────────────────────
  test('[MC-05] offline shows the last-synced list + disabled create', async ({ page, context }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const made = await createCampaign(page, {
      name: 'Offline Cache MC05',
      offer_text: '$5 off $25',
      face_value_cents: 500,
      runs_days: 30,
      channels: [{ channel: 'receipt' }],
    });
    await openCampaigns(page);
    const card = page.locator(`.mc-card[data-id="${made.campaign.id}"]`);
    await expect(card).toBeVisible();
    await expect(page.locator('#mc-root')).toHaveAttribute('data-status', 'ok');
    await expect(page.locator('#mc-new')).toBeEnabled();

    // Go offline and ask again. Nothing is mocked — the browser refuses the
    // request the way a truck with no bars does.
    await context.setOffline(true);
    await page.click('#mc-refresh');
    await expect(page.locator('#mc-root')).toHaveAttribute('data-status', 'offline');

    // The last-synced list is still on screen, and it says when it was synced.
    await expect(card).toBeVisible();
    await expect(card.locator('.mc-name')).toHaveText('Offline Cache MC05');
    await expect(page.locator('#mc-banner')).toContainText('Last synced');
    // Create is disabled, because a campaign cannot be minted offline.
    await expect(page.locator('#mc-new')).toBeDisabled();
    // Retry is the way out (UI-R6).
    await expect(page.locator('#mc-banner [data-action="reload"]')).toBeVisible();

    await context.setOffline(false);
    await page.click('#mc-banner [data-action="reload"]');
    await expect(page.locator('#mc-root')).toHaveAttribute('data-status', 'ok');
    await expect(page.locator('#mc-new')).toBeEnabled();
  });
});
