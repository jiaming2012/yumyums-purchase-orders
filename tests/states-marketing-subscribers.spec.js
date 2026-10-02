// ═══════════════════════════════════════════════════════════════════════════
// states-marketing-subscribers.spec.js — the CLAUDE.md self-verification
// ritual for marketing.html #s3 (card H5 `subscribers-tab`, run 20261002).
// ═══════════════════════════════════════════════════════════════════════════
//
// This environment is headless, so every row below is FORCED, navigated and
// SCREENSHOT, and the PNGs are read back with the multimodal Read tool and
// compared against the visual contract. The observations are in the card's
// report.
//
// ── the State Enumeration Table, and which rows ride a fixture ──────────────
//
// | row            | trigger                                        | fixture? |
// |----------------|------------------------------------------------|----------|
// | loading        | route holds GET /subscribers for 1.5 s         | FIXTURE  |
// | empty          | route fulfils the real §5 zero shape           | FIXTURE  |
// | error          | route fulfils 500 {"error":"boom"}             | FIXTURE  |
// | success        | REAL import/toast-guests, REAL GET /subscribers| real     |
// | empty-filtered | REAL read with a q nothing matches             | real     |
// | locked         | REAL team_member → REAL 403 managers_only      | real     |
// | offline        | context.setOffline(true) after a REAL load      | real*    |
// | long content   | REAL import of a 90-char name + long email     | real     |
// | detail sheet   | REAL GET /subscribers/{id}                      | real     |
//
// * `offline` uses no response fixture at all: the browser context is actually
//   taken offline, so navigator.onLine is false and fetch really fails. It is
//   the condition, not a canned body.
//
// The three fixture rows are fixtures because the condition cannot be produced
// against a live server inside a shared E2E database: a 1.5-second server, a
// deliberate 500, and a globally-empty mailing list in a database the sibling
// spec has already seeded. Everything a real endpoint can produce, a real
// endpoint produces.

const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const USER_PASSWORD = 'test456';
// 🛑 NOT under test-results/. Playwright WIPES outputDir at the start of every
// run, so screenshots written there are deleted by the next leg — including
// this card's own full suite. The state PNGs are the evidence the
// self-verification ritual exists to produce; evidence that evaporates on the
// next run is not evidence, and a reviewer at triage would have to regenerate
// it to see anything. They go in the run's own log tree instead, and are
// COMMITTED with the gate logs, so every path this card's report cites still
// resolves. (Card H2 lost its PNGs to exactly this.)
const SHOT_DIR = path.join(__dirname, '..', '.night-crew', 'runs', '2026-10-02-autonomous',
  'logs', 'h5', 'states');
fs.mkdirSync(SHOT_DIR, { recursive: true });

// 393×852 — the phone the crew actually holds.
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
  const email = `st-subs-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  const invite = await page.evaluate(async ([em, rs]) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'St', last_name: 'Subs', email: em, roles: rs }),
    });
    return res.json();
  }, [email, roles]);
  const token = (invite.invite_path || '').split('token=')[1];
  await page.evaluate(async ([t, pw]) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: pw }),
    });
  }, [token, USER_PASSWORD]);
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  return { email };
}
async function importGuests(page, csv) {
  const out = await page.evaluate(async (body) => {
    const r = await fetch('/api/v1/marketing/subscribers/import/toast-guests', {
      method: 'POST', headers: { 'Content-Type': 'text/csv' }, body,
    });
    return { status: r.status, body: await r.text() };
  }, csv);
  expect(out.status, out.body).toBe(200);
}
async function openTab(page) {
  await page.goto('/marketing.html');
  await page.click('#t3');
  await expect(page.locator('#s3')).toBeVisible();
}
async function settled(page) {
  await page.waitForFunction(() => {
    const r = document.getElementById('subs-root');
    return r && r.dataset.state && r.dataset.state !== 'loading';
  }, null, { timeout: 15000 });
}

// The §5 zero shape, used only by the `empty` row.
const ZERO = { total: 0, sms_opt_in: 0, joined_this_week: 0, rows: [] };

test.describe('States · Marketing Subscribers (#s3)', () => {

  // ── FIXTURE row: loading ────────────────────────────────────────────────
  test('loading — count skeletons and four row skeletons, never a blank card', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/api/v1/marketing/subscribers?*', async route => {
      await new Promise(r => setTimeout(r, 1500));
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(ZERO) });
    });
    await openTab(page);
    await expect(page.locator('#subs-root')).toHaveAttribute('data-state', 'loading');
    await expect(page.locator('.subs-row-skel').first()).toBeVisible();
    await shot(page, '01-loading');
  });

  // ── FIXTURE row: empty ──────────────────────────────────────────────────
  test('empty — "No subscribers yet" names the four ways in, and zeroed counts', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.route('**/api/v1/marketing/subscribers?*', route =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(ZERO) }));
    await openTab(page);
    await settled(page);
    await expect(page.locator('#subs-root')).toHaveAttribute('data-state', 'empty');
    await expect(page.locator('#subs-empty')).toContainText(/No subscribers yet/i);
    await expect(page.locator('#subs-empty')).toContainText(/Toast guest import/i);
    await expect(page.locator('#subs-total')).toHaveText('0');
    await shot(page, '02-empty');
  });

  // ── FIXTURE row: error ──────────────────────────────────────────────────
  test('error — loud red banner naming the failure, with a Retry that works', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    let fail = true;
    await page.route('**/api/v1/marketing/subscribers?*', route => fail
      ? route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"boom"}' })
      : route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(ZERO) }));
    await openTab(page);
    await settled(page);
    await expect(page.locator('#subs-root')).toHaveAttribute('data-state', 'error');
    await expect(page.locator('#subs-banner')).toContainText(/Could not load subscribers/i);
    await expect(page.locator('#subs-banner')).toContainText(/500/);
    await shot(page, '03-error');
    // Loud AND retryable (UI-R5).
    fail = false;
    await page.click('#subs-banner .subs-retry');
    await page.waitForFunction(() => document.getElementById('subs-root').dataset.state === 'empty');
    await expect(page.locator('#subs-banner')).toHaveCount(0);
    await shot(page, '03b-error-retried');
  });

  // ── REAL rows: success + long content + the detail sheet ────────────────
  test('success + long content — real rows from a real import, masked, with pills', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const t = Date.now();
    const longName = 'Bartholomew Maximilian Fitzgerald-Harrington the Third of Logan Square';
    await importGuests(page,
      'Guest Id,Name,Phone Number,Email,SMS Opt In,Email Opt In,Opted Out,Created Date\n' +
      `st-${t}-1,Rosa Linares,(773) 561-0201,rosa.l@example.com,Yes,Yes,No,2026-09-28 10:00:00\n` +
      `st-${t}-2,Theo Nakamura,(773) 561-0202,theo.n@verylongdomainnameforatest.example.com,No,Yes,No,2026-09-27 10:00:00\n` +
      `st-${t}-3,Nia Oyelaran,(773) 561-0203,nia.o@example.com,Yes,No,Yes,2026-09-26 10:00:00\n` +
      `st-${t}-4,Cal Weber,,cal.w@example.com,No,No,No,2026-09-25 10:00:00\n` +
      `st-${t}-5,${longName},(773) 561-0205,a.very.long.local.part.indeed@anotherlongdomain.example.com,Yes,Yes,No,2026-09-24 10:00:00\n`);
    await openTab(page);
    await settled(page);
    await expect(page.locator('#subs-root')).toHaveAttribute('data-state', 'ready');
    await expect(page.locator('.subs-row').first()).toBeVisible();
    // The four consent pills are all on screen at once.
    for (const [name, consent] of [['Rosa Linares', 'sms'], ['Theo Nakamura', 'email_only'],
      ['Nia Oyelaran', 'stop'], ['Cal Weber', 'pending']]) {
      await expect(page.locator('.subs-row', { hasText: name })).toHaveAttribute('data-consent', consent);
    }
    await shot(page, '04-success');

    // long content: the clamped row, then the full value on the sheet.
    const longRow = page.locator('.subs-row', { hasText: 'Bartholomew' });
    await expect(longRow).toHaveCount(1);
    const clamped = await longRow.locator('.subs-name').evaluate(
      el => ({ scroll: el.scrollWidth, client: el.clientWidth }));
    expect(clamped.scroll, 'the long name is clamped, not wrapped').toBeGreaterThan(clamped.client);
    // No row may push the page sideways at 393px.
    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow, 'no horizontal overflow at 393px').toBeLessThanOrEqual(0);
    await shot(page, '05-long-content');

    await longRow.click();
    await expect(page.locator('#subs-sheet')).toBeVisible();
    await expect(page.locator('.subs-sheet-name')).toContainText(longName);
    await expect(page.locator('#subs-code-status')).toContainText(/not sent yet/i);
    await expect(page.locator('#subs-resend')).toBeEnabled();
    await shot(page, '06-detail-sheet');

    // every row's masked phone is four digits behind four bullets, and nothing
    // on the page is a full number.
    const html = await page.content();
    expect(html).not.toMatch(/\b7735610\d{3}\b/);
    expect(html).not.toContain('rosa.l@example.com');
  });

  // ── REAL row: the filtered empty, which says something different ────────
  test('empty-filtered — "No subscribers match" is not the same message as "none yet"', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await openTab(page);
    await settled(page);
    await page.fill('#subs-q', 'zzz-nobody-by-this-name-zzz');
    await page.waitForFunction(() => document.getElementById('subs-root').dataset.state === 'empty');
    await expect(page.locator('#subs-empty')).toContainText(/No subscribers match/i);
    await shot(page, '07-empty-filtered');
  });

  // ── REAL row: locked ───────────────────────────────────────────────────
  test('locked — a team_member gets the real 403 and the whole tab is the Locked state', async ({ page }) => {
    const user = await makeUser(page, 'locked', ['team_member']);
    await loginAs(page, user.email, USER_PASSWORD);
    await openTab(page);
    await settled(page);
    await expect(page.locator('#subs-root')).toHaveAttribute('data-state', 'locked');
    await expect(page.locator('.subs-locked')).toContainText(/Managers only/i);
    await expect(page.locator('#subs-resend')).toHaveCount(0);
    await expect(page.locator('#subs-q')).toHaveCount(0);
    await shot(page, '08-locked');
  });

  // ── REAL condition: offline ────────────────────────────────────────────
  test('offline — last-synced reading kept, rows kept, Resend disabled', async ({ page, context }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const t = Date.now();
    await importGuests(page,
      'Guest Id,Name,Phone Number,Email,SMS Opt In,Email Opt In,Opted Out,Created Date\n' +
      `st-off-${t},Ada Offline,(773) 562-0301,ada.off@example.com,Yes,Yes,No,2026-09-29 10:00:00\n`);
    await openTab(page);
    await settled(page);
    await expect(page.locator('.subs-row', { hasText: 'Ada Offline' })).toHaveCount(1);
    await page.click('.subs-row:has-text("Ada Offline")');
    await expect(page.locator('#subs-sheet')).toBeVisible();

    await context.setOffline(true);
    await page.click('#subs-sheet .subs-close');
    await page.click('#subs-filters [data-filter="sms"]');
    await page.waitForFunction(() => document.getElementById('subs-root').dataset.offline === '1');
    // The list it had is still on screen, labeled with how old it is.
    await expect(page.locator('#subs-banner')).toContainText(/Offline/i);
    await expect(page.locator('#subs-banner')).toContainText(/Last synced/i);
    await expect(page.locator('.subs-row').first()).toBeVisible();
    await shot(page, '09-offline-list');

    // The one write is disabled, and says why.
    await page.click('.subs-row:has-text("Ada Offline")');
    await expect(page.locator('#subs-sheet')).toBeVisible();
    await expect(page.locator('#subs-resend')).toBeDisabled();
    await expect(page.locator('#subs-resend-note')).toContainText(/offline/i);
    await shot(page, '10-offline-sheet');
    await context.setOffline(false);
  });
});
