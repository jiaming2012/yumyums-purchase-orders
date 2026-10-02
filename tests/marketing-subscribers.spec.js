// ═══════════════════════════════════════════════════════════════════════════
// Marketing · Subscribers tab — card H5 `subscribers-tab` (run 20261002)
// spec docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md §2/§5,
// design "Current Subscribers 1–2", goal ledger
// .night-crew/knowledge/spikes/activity-h-designed-tabs/subscribers-tab.md
// ═══════════════════════════════════════════════════════════════════════════
//
// RED-FIRST (greenfield): every test here was written and RUN against the
// pre-change tree — marketing.html #s3 still the "Soon" placeholder,
// marketing/subscribers.js absent, /api/v1/marketing/subscribers unregistered.
// Evidence: .night-crew/runs/2026-10-02-autonomous/logs/h5/rf-pw-red.log and
// the ## Red-first section of merge-intents/h5-subscribers-tab.md.
//
// 🛑 EVERY TEST IN THIS FILE HITS THE REAL ENDPOINTS. Subscribers are created
// through the real POST /subscribers/import/toast-guests route and read back
// through the real GET /subscribers — there is no page.route() in this file.
// The fixture-driven rows live in tests/states-marketing-subscribers.spec.js
// and are named there.
//
// done_when rows: [SB-01] [SB-02] [SB-03] [SB-04].

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

// makeUser invites + activates a user, then returns the browser to ADMIN.
// (Same shape as tests/marketing.spec.js — this suite's per-file-helper
// convention.)
async function makeUser(page, tag, roles) {
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  const email = `subs-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  const invite = await page.evaluate(async ([em, rs]) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Subs', last_name: 'Tester', email: em, roles: rs }),
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

// importGuests POSTs a Toast guest CSV through the REAL upload route.
async function importGuests(page, csv) {
  const out = await page.evaluate(async (body) => {
    const r = await fetch('/api/v1/marketing/subscribers/import/toast-guests', {
      method: 'POST', headers: { 'Content-Type': 'text/csv' }, body,
    });
    return { status: r.status, body: await r.text() };
  }, csv);
  expect(out.status, `import/toast-guests → ${out.body}`).toBe(200);
  return JSON.parse(out.body);
}

// Every test seeds its own guests with a unique local tag, because the E2E
// database is shared across this file's tests and `subscribers.phone_e164` is
// globally unique — two tests reusing one number would merge each other's rows.
function guestCSV(rows) {
  return 'Guest Id,Name,Phone Number,Email,SMS Opt In,Email Opt In,Opted Out,Created Date\n' +
    rows.map(r => [r.id, r.name, r.phone, r.email, r.sms ? 'Yes' : 'No',
      r.email_ok ? 'Yes' : 'No', r.opted_out ? 'Yes' : 'No', r.created || '2026-09-20 10:00:00'].join(',')).join('\n') + '\n';
}

// openSubscribers lands on marketing.html, switches to the Subscribers tab and
// waits for the list to settle out of its loading state.
async function openSubscribers(page) {
  await page.goto('/marketing.html');
  await page.click('#t3');
  await expect(page.locator('#s3')).toBeVisible();
  await page.waitForFunction(() => {
    const r = document.getElementById('subs-root');
    return r && r.dataset.state && r.dataset.state !== 'loading';
  }, null, { timeout: 15000 });
}

test.describe('Marketing · Subscribers (card H5)', () => {

  // ── [SB-01] ──────────────────────────────────────────────────────────────
  test('[SB-01] list masks phone to last 4 — and the full number never reaches the browser', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const tag = `sb01-${Date.now()}`;
    await importGuests(page, guestCSV([
      { id: `${tag}-a`, name: 'Dana Mask', phone: '(773) 555-4821', email: 'dana.mask@example.com', sms: true, email_ok: true },
    ]));

    // (a) the WIRE: what the browser actually received carries the mask and
    //     not the value. This is the privacy contract, asserted on the
    //     response body rather than on the rendering.
    const wire = await page.evaluate(async () => {
      const r = await fetch('/api/v1/marketing/subscribers');
      return { status: r.status, text: await r.text() };
    });
    expect(wire.status).toBe(200);
    expect(wire.text, 'phone_last4 is on the wire').toContain('"phone_last4":"4821"');
    expect(wire.text, 'the full phone is NOT on the wire').not.toContain('7735554821');
    expect(wire.text, 'the E.164 value is NOT on the wire').not.toContain('+17735554821');
    expect(wire.text, 'the full email is NOT on the wire').not.toContain('dana.mask@example.com');
    expect(wire.text, 'there is no phone_e164 key at all').not.toContain('phone_e164');
    expect(wire.text, 'email_masked carries the domain only').toContain('"email_masked":"d•••@example.com"');

    // (b) the RENDER: the designed cell.
    await openSubscribers(page);
    const row = page.locator('.subs-row', { hasText: 'Dana Mask' });
    await expect(row).toHaveCount(1);
    await expect(row.locator('.subs-phone')).toHaveText('•••• 4821');

    // (c) the WHOLE DOM: no full number anywhere on the page, masked or not.
    const html = await page.content();
    expect(html).not.toContain('7735554821');
    expect(html).not.toContain('555-4821');
    expect(html).not.toContain('dana.mask@example.com');
  });

  // ── [SB-02] ──────────────────────────────────────────────────────────────
  test('[SB-02] STOP renders an opted-out row, and the Opted out chip isolates it', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const tag = `sb02-${Date.now()}`;
    await importGuests(page, guestCSV([
      { id: `${tag}-stop`, name: 'Nia Stopped', phone: '(773) 556-0102', email: 'nia.stop@example.com', sms: true, opted_out: true },
      { id: `${tag}-ok`, name: 'Rosa Active', phone: '(773) 556-0103', email: 'rosa.active@example.com', sms: true },
    ]));
    await openSubscribers(page);

    const stopped = page.locator('.subs-row', { hasText: 'Nia Stopped' });
    await expect(stopped).toHaveCount(1);
    // The row is visibly an opted-out row, not just a differently-coloured
    // one: the pill says STOP and the row carries the data attribute the
    // styling and the tests both key off.
    await expect(stopped).toHaveAttribute('data-consent', 'stop');
    await expect(stopped.locator('.subs-consent')).toHaveText('STOP');

    // opted_out_at WINS over a still-true sms_consent — the whole reason the
    // state is surfaced.
    const active = page.locator('.subs-row', { hasText: 'Rosa Active' });
    await expect(active).toHaveAttribute('data-consent', 'sms');

    // The chip isolates the suppression list.
    await page.click('#subs-filters [data-filter="opted_out"]');
    await page.waitForFunction(() => document.getElementById('subs-root').dataset.filter === 'opted_out');
    await expect(page.locator('.subs-row', { hasText: 'Nia Stopped' })).toHaveCount(1);
    await expect(page.locator('.subs-row', { hasText: 'Rosa Active' })).toHaveCount(0);

    // And the SMS chip excludes them, because that chip answers "who may an
    // SMS blast reach".
    await page.click('#subs-filters [data-filter="sms"]');
    await page.waitForFunction(() => document.getElementById('subs-root').dataset.filter === 'sms');
    await expect(page.locator('.subs-row', { hasText: 'Nia Stopped' })).toHaveCount(0);
    await expect(page.locator('.subs-row', { hasText: 'Rosa Active' })).toHaveCount(1);
  });

  // ── [SB-03] ──────────────────────────────────────────────────────────────
  test('[SB-03] the sheet shows identity-code status, the consent trail and the timeline', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const tag = `sb03-${Date.now()}`;
    await importGuests(page, guestCSV([
      { id: `${tag}-a`, name: 'Theo Sheet', phone: '(773) 557-0104', email: 'theo.sheet@example.com', sms: true, email_ok: true },
    ]));
    await openSubscribers(page);
    await page.click('.subs-row:has-text("Theo Sheet")');
    const sheet = page.locator('#subs-sheet');
    await expect(sheet).toBeVisible();

    // Identity-code status. Activity E owns the code itself, so the honest
    // status derived from the timeline is "not sent yet" — and it must SAY so
    // rather than render a blank (UI-R: blank render = defect).
    await expect(sheet.locator('#subs-code-status')).toHaveText(/not sent yet/i);

    // The consent trail is the §4 columns spelled out, including the evidence
    // line the adapter wrote.
    await expect(sheet.locator('#subs-consent-trail')).toContainText(/SMS/i);
    await expect(sheet.locator('#subs-consent-trail')).toContainText(/toast guest import/i);

    // The timeline renders the events, newest first, in words.
    const events = sheet.locator('.subs-event');
    await expect(events.first()).toContainText(/Signed up/i);
    await expect(events).not.toHaveCount(0);

    // The sheet masks exactly like the list does.
    await expect(sheet.locator('.subs-phone').first()).toHaveText('•••• 0104');
    const html = await page.content();
    expect(html).not.toContain('7735570104');
    expect(html).not.toContain('theo.sheet@example.com');
  });

  // ── [SB-04] ──────────────────────────────────────────────────────────────
  test('[SB-04] Resend records an event and sends nothing', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    const tag = `sb04-${Date.now()}`;
    await importGuests(page, guestCSV([
      { id: `${tag}-a`, name: 'Cal Resend', phone: '(773) 558-0105', email: 'cal.resend@example.com', sms: true },
    ]));

    // Watch EVERY request the page makes from here on. A send would be an
    // outbound call; there must be none beyond the two HQ reads and the one
    // HQ write.
    const requests = [];
    page.on('request', r => requests.push(r.method() + ' ' + r.url()));

    await openSubscribers(page);
    await page.click('.subs-row:has-text("Cal Resend")');
    const sheet = page.locator('#subs-sheet');
    await expect(sheet).toBeVisible();

    const [resp] = await Promise.all([
      page.waitForResponse(r => r.url().includes('/resend') && r.request().method() === 'POST'),
      page.click('#subs-resend'),
    ]);
    expect(resp.status(), 'resend answers 202 Accepted, never 200').toBe(202);
    const body = await resp.json();
    expect(body.sent, 'the body says, in words, that nothing was sent').toBe(false);

    // The UI says a request was RECORDED, and says in words that nothing went
    // out. Asserted as CONTENT (UI-R4), not as the absence of a substring: the
    // honest note necessarily contains the word "sent", in the sentence
    // "Nothing has been sent".
    const note = sheet.locator('#subs-resend-note');
    await expect(note).toContainText(/Resend requested/i);
    await expect(note).toContainText(/Nothing has been sent/i);
    await expect(note, 'the note must never claim a code went out').not.toContainText(/code sent/i);
    // And the status still reads not-sent, because a request is not a send.
    await expect(sheet.locator('#subs-code-status')).toHaveText(/not sent yet/i);

    // The timeline gained `resend_requested` and NOT `code_sent`.
    const detail = await page.evaluate(async () => {
      const id = document.getElementById('subs-sheet').dataset.id;
      const r = await fetch('/api/v1/marketing/subscribers/' + id);
      return r.json();
    });
    const kinds = detail.events.map(e => e.kind);
    expect(kinds, 'resend_requested is on the timeline').toContain('resend_requested');
    expect(kinds, 'code_sent is NOT — that is the send\'s event and the Stats funnel counts it').not.toContain('code_sent');
    expect(detail.identity_code.status).toBe('not_sent');
    expect(detail.identity_code.requested_at, 'the request is timestamped').toBeTruthy();

    // Nothing left the origin. Every request the page made is same-origin HQ.
    const foreign = requests.filter(u => !u.includes('localhost') && !u.startsWith('GET data:'));
    expect(foreign, `the page made offsite requests: ${foreign.join(', ')}`).toEqual([]);
  });

  // ── the Locked state, against the real 403 ───────────────────────────────
  test('a team_member sees the Locked state, from the real 403 managers_only', async ({ page }) => {
    const user = await makeUser(page, 'locked', ['team_member']);
    await loginAs(page, user.email, USER_PASSWORD);
    const probe = await page.evaluate(async () => {
      const r = await fetch('/api/v1/marketing/subscribers');
      return { status: r.status, body: await r.text() };
    });
    expect(probe.status).toBe(403);
    expect(JSON.parse(probe.body).error).toBe('managers_only');

    await openSubscribers(page);
    await expect(page.locator('#subs-root')).toHaveAttribute('data-state', 'locked');
    await expect(page.locator('.subs-locked')).toContainText(/manager/i);
    // A locked tab offers no write affordance at all.
    await expect(page.locator('#subs-resend')).toHaveCount(0);
  });

  // ── the empty state, against a real empty read ───────────────────────────
  test('the empty state names what is missing rather than rendering nothing', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await openSubscribers(page);
    // The shared E2E database accumulates rows from this file's other tests,
    // so force empty with a query no seeded name can satisfy — the same read
    // path, the same render, a genuinely empty result.
    await page.fill('#subs-q', 'zzzz-no-such-subscriber-zzzz');
    await page.waitForFunction(() => {
      const r = document.getElementById('subs-root');
      return r.dataset.state === 'empty' || r.dataset.state === 'ready';
    });
    await expect(page.locator('#subs-empty')).toBeVisible();
    await expect(page.locator('#subs-empty')).toContainText(/no subscribers/i);
    await expect(page.locator('.subs-row')).toHaveCount(0);
  });
});
