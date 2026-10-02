// ═══════════════════════════════════════════════════════════════════════════
// Card H4 `stats-tab-ui` (run 20261002) — roadmap H4, rewritten for DECISION
// 192: the campaign REPORTS live on the BI hub (bi.html #s3, the `bi` grant,
// no manager tier) and the reconciliation QUEUE lives on Marketing
// (marketing.html #s4, manager-only). Spec:
// docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md §2/§3 row 192
// /§5/§6 row H4; goal ledger
// .night-crew/knowledge/spikes/activity-h-designed-tabs/stats-tab-ui.md
// ═══════════════════════════════════════════════════════════════════════════
//
// RED-FIRST (greenfield): every test here was written and RUN against the
// pre-change tree — marketing.html #s4 still the "Soon" placeholder, bi.html
// carrying two hub rows, marketing/stats.js absent. Evidence:
// .night-crew/runs/2026-10-02-autonomous/logs/h4/rf-pw-red.log and the
// ## Red-first section of merge-intents/h4-stats-tab-ui.md.
//
// 🛑 EVERY TEST IN THIS FILE HITS THE REAL ENDPOINTS. There is no page.route()
// anywhere in it. Campaigns and codes are created through the real
// POST /campaigns + POST /campaigns/{id}/codes; decisions are written through
// the real POST /reconciliation/{id}/decline|reopen; and everything is read
// back through the real GET /bi/campaigns/overview|by,
// GET /marketing/reconciliation/queue|declined and GET /marketing/stats/overview.
//
// The two tables an operator CANNOT reach through an API — `toast_orders` (the
// Toast SFTP ingest, card H3a) and `scan_attempts_mirror` (the Supabase mirror
// worker, internal/marketing/mirror.go) — are seeded with psql against the
// coordinates scripts/reset-e2e-db.js resolves, the same way
// tests/sync-one-row.spec.js reads HQ's own rows back. That is the honest
// division: no endpoint exists to create an upstream-ingested row, and
// internal/marketing/recon_fixture_test.go seeds exactly these three tables
// for the Go suite for the same reason.
//
// 🛑 :5434 / role `hqtest` is the TEST cluster, resolved by resolveE2eDb() and
// never hard-coded here. :5433 is dev AND PRODUCTION (ledger decision 155,
// B-141/B-143). Every statement below is a DELETE or an INSERT against the
// three fixture tables; this file issues no DDL.
//
// done_when rows: [MS-01] [MS-02] [MS-03] [MS-04] [MS-05] [MS-06].

const { test, expect } = require('@playwright/test');
const { execFileSync } = require('child_process');
const { resolveE2eDb } = require('../scripts/reset-e2e-db');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const USER_PASSWORD = 'test456';

// ── psql plumbing ───────────────────────────────────────────────────────────
// `psqlUrl`, not `testUrl`: the latter carries TimeZone, which pgx understands
// and libpq does not (the note sync-one-row.spec.js carries).
function psql(sqlText) {
  const db = resolveE2eDb();
  return execFileSync('psql', [db.psqlUrl, '-At', '-v', 'ON_ERROR_STOP=1', '-c', sqlText],
    { encoding: 'utf8' });
}

// resetMarketingFixture clears the fixture tables this file owns. It deletes
// campaigns + codes as well as the two mirrored tables because the stats engine
// groups by campaign and counts a code's scans — another spec file's leftover
// campaign would add a row to the by-campaign slice and change the funnel.
// Every marketing spec file creates its own campaigns through the API inside
// its own tests, so clearing them between files is safe (and Playwright runs
// one worker, files serially — playwright.config.js).
function resetMarketingFixture() {
  psql(`DELETE FROM reconciliation_decisions;
        DELETE FROM scan_attempts_mirror;
        DELETE FROM toast_orders;
        DELETE FROM qr_scans;
        DELETE FROM qr_codes;
        DELETE FROM campaigns_admin;`);
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
  const email = `stats-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  const invite = await page.evaluate(async ([em, rs]) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Stats', last_name: 'Tester', email: em, roles: rs }),
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

// ── grant plumbing (the shape tests/marketing.spec.js uses) ─────────────────
async function getSlugPerms(page, slug) {
  return page.evaluate(async (s) => {
    const perms = await (await fetch('/api/v1/apps/permissions')).json();
    return (perms || []).find(a => a.slug === s) || null;
  }, slug);
}
async function putSlugPerms(page, slug, body) {
  const status = await page.evaluate(async ([s, b]) => {
    const r = await fetch('/api/v1/apps/' + s + '/permissions', {
      method: 'PUT', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(b),
    });
    return r.status;
  }, [slug, body]);
  expect(status, `PUT /apps/${slug}/permissions`).toBe(200);
}

// ── API fixture builders (REAL routes) ─────────────────────────────────────
async function createCampaign(page, name, faceValueCents) {
  const out = await page.evaluate(async ([nm, fv]) => {
    const res = await fetch('/api/v1/marketing/campaigns', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: nm, offer_text: nm + ' offer', face_value_cents: fv,
        landing: 'signup', runs_days: 60, channels: [{ channel: 'truck_sign' }],
      }),
    });
    return { status: res.status, body: await res.text() };
  }, [name, faceValueCents]);
  expect(out.status, `POST /campaigns → ${out.body}`).toBe(201);
  // §5 row 2's envelope is {campaign, codes:[…]} — one code per channel, minted
  // in the campaign's own transaction, so the fixture gets both from one call.
  const body = JSON.parse(out.body);
  return { campaign: body.campaign, code: body.codes[0] };
}

// ── the mirrored-attempt fixture ───────────────────────────────────────────
// Business date is fixed at "yesterday" so `period=30d` always contains it and
// no test is a midnight flake.
function fixtureDay() {
  const d = new Date(Date.now() - 24 * 3600 * 1000);
  return d.toISOString().slice(0, 10);
}
const DAY = fixtureDay();
// Scans and orders sit at a fixed wall-clock time on that date, in UTC, so the
// ±30-minute suggestion window arithmetic is deterministic.
const T = (hh, mm) => `${DAY} ${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}:00+00`;

function insertOrder(num, openedAt, amountCents, discountCents) {
  psql(`INSERT INTO toast_orders
          (business_date, order_number, order_id, opened_at, amount_cents, discount_cents, total_cents)
        VALUES ('${DAY}','${num}','ord-${num}','${openedAt}',${amountCents},${discountCents},${amountCents - discountCents})`);
}

// insertAttempt writes one ACCEPTED mirrored attempt. `opts`:
//   codeId / campaignId — null for the unattributable override shape (D-4's
//     live shape; migration 0084's CHECK then requires unverified+override+hash)
//   orderNumber — null for the orphan bucket
//   override — the offline-override bucket
function insertAttempt(id, scannedAt, opts) {
  const o = opts || {};
  const q = v => (v === null || v === undefined ? 'NULL' : `'${v}'`);
  const override = o.override ? 'true' : 'false';
  const unverified = o.codeId ? 'false' : 'true';
  const tokenHash = o.codeId ? 'NULL' : `'hash-${id}'`;
  psql(`INSERT INTO scan_attempts_mirror
          (id, code_id, campaign_id, device_id, scanned_at, status, offline_override,
           unverified_code, token_hash, pos_order_number, pos_business_date, match_status)
        VALUES ('${id}', ${q(o.codeId)}, ${q(o.campaignId)}, 'tablet-1', '${scannedAt}',
                'accepted', ${override}, ${unverified}, ${tokenHash},
                ${q(o.orderNumber)}, '${DAY}', 'unmatched')`);
}

const U = n => `aaaaaaaa-0000-4000-8000-00000000000${n}`;

// moneyFixture is the shape every money assertion in this file is read against.
// Deliberately mirrors the G6 live shape (logs/h3b/wire-shapes-for-card-h4.log):
// attributed rows AND one unattributable override carrying real matched money,
// which is what makes the `unattributed` row and the 10% orphan line both real.
//
//   orders  101 $30.00 (disc $5.00) · 102 $20.00 (disc $0) · 103 $15.00 (disc $3.00)
//   a1 code+campaign, order 101      -> matched
//   a2 code+campaign, order 102      -> matched
//   a3 code+campaign, order NULL     -> ORPHAN      (counts in the orphan rate)
//   a4 code+campaign, order '999'    -> UNMATCHED   (counts in the orphan rate)
//   a5 override, no code/campaign, order 103 -> OVERRIDE, matched, UNATTRIBUTED
//
//   funnel.redeemed 5 · matched 3 · open 3 · declined 0
//   orphan_numerator 2 / denominator 5 = 40% — over the 10% line
//   revenue  $65.00 · discount $18.00 · net $47.00
//   discount_implied $20.00 vs discount_actual $8.00  -> the split line renders
//   discount_unknown_rows 1                            -> the no-campaign note
//   unattributed_redeemed 1 · unattributed_revenue $15.00
async function seedMoneyFixture(page) {
  resetMarketingFixture();
  const { campaign: camp, code } = await createCampaign(page, 'Taco Tuesday H4', 500);
  insertOrder('101', T(17, 0), 3000, 500);
  insertOrder('102', T(17, 10), 2000, 0);
  insertOrder('103', T(17, 20), 1500, 300);
  insertAttempt(U(1), T(17, 1), { codeId: code.id, campaignId: camp.id, orderNumber: '101' });
  insertAttempt(U(2), T(17, 11), { codeId: code.id, campaignId: camp.id, orderNumber: '102' });
  insertAttempt(U(3), T(17, 5), { codeId: code.id, campaignId: camp.id, orderNumber: null });
  insertAttempt(U(4), T(17, 6), { codeId: code.id, campaignId: camp.id, orderNumber: '999' });
  insertAttempt(U(5), T(17, 21), { orderNumber: '103', override: true });
  return { camp, code };
}

// queueFixture is the fixture the QUEUE tests read.
//
// 🛑 THE SUGGESTION-CARRYING ROWS ARE `unmatched`, NOT `orphan`, AND THAT IS THE
// SERVER'S RULE, NOT A CHOICE HERE. internal/marketing/reconciliation.go's
// reconRow computes a nearest-order suggestion for ONE bucket:
//
//     if bucket == "unmatched" { row.Suggestion = reconNearestOrder(a, orders) }
//
// so an `orphan` (no order number at all) comes back with `suggestion: null`
// however near a till order sits. Card H4 renders what the server sends — a
// chip when there is a suggestion, and an honest "No till order is near this
// scan. Type the number off the ticket." when there is not — rather than
// computing a suggestion of its own, which would be a second arithmetic over
// the ±30-minute window. See the card report: the orphan bucket is arguably
// where the chips would help most, and that is card H3b's handler to change.
//
// Two unmatched rows on purpose, to exercise BOTH of D-3's bases:
//   U(1) scanned 12:08, order 202 opened 12:04 -> 4 min, basis `window`
//   U(2) scanned 20:30, nearest is still 202   -> 8h26m, OUTSIDE the window,
//                                                 so basis `business_date`
async function seedQueueFixture(page) {
  resetMarketingFixture();
  const { campaign: camp, code } = await createCampaign(page, 'Wing Wednesday H4', 200);
  insertOrder('201', T(12, 0), 2500, 400);
  insertOrder('202', T(12, 4), 1800, 200);
  insertAttempt(U(1), T(12, 8), { codeId: code.id, campaignId: camp.id, orderNumber: '888' });
  insertAttempt(U(2), T(20, 30), { codeId: code.id, campaignId: camp.id, orderNumber: '889' });
  insertAttempt(U(3), T(12, 9), { codeId: code.id, campaignId: camp.id, orderNumber: null });
  insertAttempt(U(4), T(12, 10), { orderNumber: null, override: true });
  return { camp, code };
}

// ── page helpers ───────────────────────────────────────────────────────────
async function openBiCampaigns(page) {
  await page.goto('/bi.html#tab=3');
  await expect(page.locator('#s3')).toBeVisible();
  await page.waitForFunction(() => {
    const r = document.getElementById('bi-campaigns-root');
    return r && r.dataset.state && r.dataset.state !== 'loading';
  }, null, { timeout: 15000 });
}
async function openQueue(page) {
  await page.goto('/marketing.html');
  await page.click('#t4');
  await expect(page.locator('#s4')).toBeVisible();
  await page.waitForFunction(() => {
    const r = document.getElementById('ms-stats-root');
    return r && r.dataset.state && r.dataset.state !== 'loading';
  }, null, { timeout: 15000 });
}

test.describe('Marketing stats — card H4', () => {

  // ───────────────────────────────────────────────────────────────────────────
  // [MS-01] BI overview renders funnel + revenue/discount/net + orphan rate
  //         with the 10% marker.
  // ───────────────────────────────────────────────────────────────────────────
  test('[MS-01] the BI Campaigns overview renders the funnel, revenue/discount/net and the orphan rate against the 10% line', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedMoneyFixture(page);
    await openBiCampaigns(page);

    const root = page.locator('#bi-campaigns-root');
    await expect(root).toHaveAttribute('data-state', 'ready');

    // Funnel — the four §5 steps.
    //
    // 🛑 Card H5's migration 0085 IS IN THIS BASE, so `subscribers` and
    // `subscriber_events` both exist and statsLoad reports signups_basis
    // "subscribers" / codes_sent_basis "subscriber_events" — NOT "unavailable".
    // These zeros are therefore REAL, STATED zeros ("nobody signed up"), and
    // rendering them as 0 is correct. The em-dash rule (an "unavailable" basis
    // is an EM DASH, never 0 — wire-shapes note 3) cannot be exercised against
    // this base at all, so it rides a FIXTURE: see the "BI unavailable basis"
    // row in tests/states-marketing-stats.spec.js. Asserting the dash here
    // would be asserting a basis the server no longer returns.
    await expect(page.locator('#bic-funnel .bic-step[data-k="scans"] .bic-step-v')).toHaveText('0');
    await expect(page.locator('#bic-funnel .bic-step[data-k="redeemed"] .bic-step-v')).toHaveText('5');
    await expect(page.locator('#bic-funnel .bic-step[data-k="signups"] .bic-step-v')).toHaveText('0');
    await expect(page.locator('#bic-funnel .bic-step[data-k="codes_sent"] .bic-step-v')).toHaveText('0');
    // Nothing is unavailable, so there is no dash to explain.
    await expect(page.locator('.bic-basis-note')).toHaveCount(0);

    // Money — all three lines, from the NESTED `money` block (§5's asymmetry).
    await expect(page.locator('#bic-money .bic-money-row[data-k="revenue"] .bic-money-v')).toHaveText('$65.00');
    await expect(page.locator('#bic-money .bic-money-row[data-k="discount"] .bic-money-v')).toHaveText('−$18.00');
    await expect(page.locator('#bic-money .bic-money-row[data-k="net"] .bic-money-v')).toHaveText('$47.00');

    // decision 190's decomposition, shown BECAUSE implied ($20.00) and actual
    // ($8.00) differ on this fixture.
    const split = page.locator('#bic-money .bic-discount-split');
    await expect(split).toBeVisible();
    await expect(split).toContainText('implied −$20.00');
    await expect(split).toContainText('actual −$8.00');

    // discount_unknown_rows = 1: one redemption had no campaign to price it, so
    // the implied comparison is a floor. Said on screen, never a silent zero.
    await expect(page.locator('#bic-money .bic-unknown-note')).toContainText('1 redemption');

    // The health card: the rate, the 10% line it is graded against, AND the
    // basis the server names for its own numerator (D-5 is PARKED — the figure
    // is rendered self-describing, never re-derived here).
    const health = page.locator('#bic-health');
    await expect(health.locator('.bic-orphan-rate')).toHaveText('40.0%');
    await expect(health).toContainText('10%');
    await expect(health.locator('.bic-orphan-basis')).toContainText('2 of 5');
    await expect(health.locator('.bic-orphan-basis')).toContainText('unmatched_and_orphans_excl_duplicate_scan');
    // 40% is over the line, so the card says so rather than only showing a number.
    await expect(health).toHaveClass(/over/);
    await expect(health.locator('.bic-h[data-k="matched"] b')).toHaveText('3');
    await expect(health.locator('.bic-h[data-k="open"] b')).toHaveText('3');
    await expect(health.locator('.bic-h[data-k="declined"] b')).toHaveText('0');

    // D-4: the money that no campaign could claim is stated at period scope,
    // beside the funnel — not left to be inferred from a $0.00 somewhere.
    await expect(page.locator('#bic-money .bic-unattributed')).toContainText('$15.00');
  });

  // ───────────────────────────────────────────────────────────────────────────
  // [MS-02] by-item rows carry discount and Per $1.
  // ───────────────────────────────────────────────────────────────────────────
  test('[MS-02] the by-item slice carries a discount and a Per $1 on every row, and keeps the unattributed money visible', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedMoneyFixture(page);
    await openBiCampaigns(page);

    // Slice links switch dimension in place.
    await page.click('#bic-slices .bic-slice[data-dim="item"]');
    const slice = page.locator('#bic-slice');
    await expect(slice).toHaveAttribute('data-dim', 'item');
    // Money mode is what carries discount + Per $1 (the Funnel ⇄ Money toggle).
    await page.click('#bic-slice .bic-toggle[data-mode="money"]');
    await expect(slice).toHaveAttribute('data-mode', 'money');

    // Two rows on this fixture: the four coded redemptions group under "Any
    // item" (no menu item on campaign or code); the unattributable override has
    // no resolvable code, so its item is unanswerable and it groups under Direct.
    const rows = slice.locator('.bic-row');
    await expect(rows).toHaveCount(2);
    for (const key of ['any', 'direct']) {
      const row = slice.locator(`.bic-row[data-key="${key}"]`);
      await expect(row, `by-item row ${key} rendered`).toHaveCount(1);
      await expect(row.locator('.bic-c[data-k="discount"]')).not.toHaveText('');
      await expect(row.locator('.bic-c[data-k="per_dollar"]')).not.toHaveText('');
    }
    // Σ over the rows is the overview — which is only true because nothing is
    // filtered out. "Any item": $50.00 / −$15.00 / 3.33. Direct: $15.00 / −$3.00 / 5.00.
    await expect(slice.locator('.bic-row[data-key="any"] .bic-c[data-k="discount"]')).toHaveText('−$15.00');
    await expect(slice.locator('.bic-row[data-key="any"] .bic-c[data-k="per_dollar"]')).toHaveText('3.33');
    await expect(slice.locator('.bic-row[data-key="direct"] .bic-c[data-k="discount"]')).toHaveText('−$3.00');
    await expect(slice.locator('.bic-row[data-key="direct"] .bic-c[data-k="per_dollar"]')).toHaveText('5.00');
    await expect(slice.locator('.bic-totals .bic-c[data-k="discount"]')).toHaveText('−$18.00');

    // D-4, the keystone: on the by-CAMPAIGN slice the unattributable row is a
    // REAL, LABELLED row and is never filtered out — filtering it is what makes
    // the slice stop summing to the overview.
    await page.click('#bic-slices .bic-slice[data-dim="campaign"]');
    await expect(slice).toHaveAttribute('data-dim', 'campaign');
    const un = slice.locator('.bic-row[data-key="unattributed"]');
    await expect(un).toHaveCount(1);
    await expect(un).toContainText('Unattributed');
    await expect(un.locator('.bic-c[data-k="revenue"]')).toHaveText('$15.00');
  });

  // ───────────────────────────────────────────────────────────────────────────
  // [MS-03] the decline sheet requires a reason and posts reason + note.
  // ───────────────────────────────────────────────────────────────────────────
  test('[MS-03] the decline sheet will not post without a reason, requires a note for Other, and posts both', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedQueueFixture(page);
    await openQueue(page);

    const root = page.locator('#ms-stats-root');
    await expect(root).toHaveAttribute('data-state', 'ready');
    // Four open rows: 1 override, 1 orphan, 2 unmatched.
    await expect(page.locator('#msq-head')).toContainText('4 redemptions need a look');

    // "Can't match…" is on EVERY row, including the override.
    await expect(page.locator('.msq-row .msq-decline')).toHaveCount(4);

    const row = page.locator(`.msq-row[data-id="${U(1)}"]`);
    await row.locator('.msq-decline').click();
    const sheet = page.locator('#msq-sheet-decline');
    await expect(sheet).toBeVisible();

    // The sheet states, in plain words, what declining does — it is a judgment
    // that leaves the redemption out of the matched money, not a delete.
    await expect(sheet.locator('.msq-decline-what')).toContainText('matched');

    // No reason chosen → Save is refused, and the reason why is on screen.
    await expect(sheet.locator('#msq-decline-save')).toBeDisabled();

    // reason=other → the note is required, and Save stays refused until it has one.
    await sheet.locator('.msq-reason[data-reason="other"]').click();
    await expect(sheet.locator('#msq-note')).toBeVisible();
    await expect(sheet.locator('#msq-decline-save')).toBeDisabled();
    await sheet.locator('#msq-note').fill('   ');
    await expect(sheet.locator('#msq-decline-save'), 'whitespace is not an explanation').toBeDisabled();

    // A real reason + note posts both.
    await sheet.locator('.msq-reason[data-reason="customer_left"]').click();
    await expect(sheet.locator('#msq-decline-save')).toBeEnabled();
    await sheet.locator('#msq-note').fill('Walked off before the till rang it up');
    await sheet.locator('#msq-decline-save').click();
    await expect(sheet).toBeHidden();

    // The row left the open queue and the header count fell with it.
    await expect(page.locator(`.msq-row[data-id="${U(1)}"]`)).toHaveCount(0);
    await expect(page.locator('#msq-head')).toContainText('3 redemptions need a look');

    // And the REAL endpoint holds the reason AND the note (read back through
    // /reconciliation/declined, not out of the DOM).
    const declined = await page.evaluate(async () => {
      const r = await fetch('/api/v1/marketing/reconciliation/declined');
      return r.json();
    });
    const mine = (declined.declined || []).find(d => d.id === U(1));
    expect(mine, 'the declined attempt came back from the endpoint').toBeTruthy();
    expect(mine.reason).toBe('customer_left');
    expect(mine.note).toBe('Walked off before the till rang it up');
  });

  // ───────────────────────────────────────────────────────────────────────────
  // [MS-04] the declined bucket shows the note and a Reopen.
  // ───────────────────────────────────────────────────────────────────────────
  test('[MS-04] the declined bucket shows the reason, the note and who decided, and Reopen puts the row back in the queue', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedQueueFixture(page);
    // Decline through the REAL endpoint so this test reads the bucket, not the sheet.
    const posted = await page.evaluate(async (id) => {
      const r = await fetch(`/api/v1/marketing/reconciliation/${id}/decline`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ reason: 'other', note: 'Till drawer jammed, no ticket' }),
      });
      return { status: r.status, body: await r.text() };
    }, U(2));
    expect(posted.status, `decline → ${posted.body}`).toBe(200);

    await openQueue(page);
    const bucket = page.locator('.msq-section[data-bucket="declined"]');
    await expect(bucket).toBeVisible();
    await expect(bucket.locator('.msq-section-h')).toContainText('1');

    const row = bucket.locator(`.msq-row[data-id="${U(2)}"]`);
    await expect(row).toHaveCount(1);
    await expect(row.locator('.msq-decl-reason')).toContainText('Other');
    // The NOTE is the whole reason `other` demands one — it must be on screen.
    await expect(row.locator('.msq-decl-note')).toContainText('Till drawer jammed, no ticket');
    await expect(row).toContainText('Jamal');

    // Reopen returns it to the open queue.
    await expect(page.locator('#msq-head')).toContainText('3 redemptions need a look');
    await row.locator('.msq-reopen').click();
    await expect(page.locator('#msq-head')).toContainText('4 redemptions need a look');
    // U(2) carries a typed order number Toast does not have, so reopening it
    // puts it back in UNMATCHED — the bucket the server's own bucket() derives,
    // which is why this card never guesses a bucket from the decision kind.
    await expect(page.locator('.msq-section[data-bucket="unmatched"]')
      .locator(`.msq-row[data-id="${U(2)}"]`)).toHaveCount(1);
  });

  // ───────────────────────────────────────────────────────────────────────────
  // [MS-05] an empty period renders "No redemptions yet" on BOTH pages.
  // ───────────────────────────────────────────────────────────────────────────
  test('[MS-05] an empty period renders "No redemptions yet" on the BI report AND on the Marketing queue', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    resetMarketingFixture();

    await openBiCampaigns(page);
    await expect(page.locator('#bi-campaigns-root')).toHaveAttribute('data-state', 'empty');
    await expect(page.locator('#bi-campaigns-root')).toContainText('No redemptions yet');
    // Blank render = defect: the section still names itself.
    await expect(page.locator('#s3')).toContainText('Campaigns');
    // An empty period states no money figure at all rather than three confident
    // zeros (the defect the ritual caught on another card tonight).
    await expect(page.locator('#bic-money')).toHaveCount(0);

    await openQueue(page);
    await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'empty');
    await expect(page.locator('#ms-stats-root')).toContainText('No redemptions yet');
    await expect(page.locator('#s4')).toContainText('Redemption stats');
  });

  // ───────────────────────────────────────────────────────────────────────────
  // [MS-06] `bi` without `marketing`: the BI row renders, the queue is Locked.
  // ───────────────────────────────────────────────────────────────────────────
  test('[MS-06] a user holding bi but not marketing reads the BI Campaigns row and gets the Locked state on the Marketing queue', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedMoneyFixture(page);
    const user = await makeUser(page, 'bi-only', ['team_member']);

    // Hold `bi` as an individual grant; hold NOTHING on `marketing`.
    const bi = await getSlugPerms(page, 'bi');
    expect(bi, 'the bi app must exist to be grantable').toBeTruthy();
    const mk = await getSlugPerms(page, 'marketing');
    expect(mk, 'the marketing app must exist to be strippable').toBeTruthy();
    await putSlugPerms(page, 'bi', {
      role_grants: bi.role_grants || [],
      user_grants: [...(bi.user_grants || []).map(String), user.id],
    });
    await putSlugPerms(page, 'marketing', {
      role_grants: [], user_grants: (mk.user_grants || []).map(String),
    });

    try {
      await loginAs(page, user.email, USER_PASSWORD);

      // The BI hub carries the Campaigns row and it reads, on the `bi` grant
      // alone — decision 192: no manager tier on this pair.
      await page.goto('/bi.html');
      await expect(page.locator('#s0')).toBeVisible();
      await expect(page.locator('#t3 .hub-t')).toHaveText('Campaigns');
      await openBiCampaigns(page);
      await expect(page.locator('#bi-campaigns-root')).toHaveAttribute('data-state', 'ready');
      await expect(page.locator('#bic-money .bic-money-row[data-k="revenue"] .bic-money-v')).toHaveText('$65.00');

      // The Marketing queue refuses them — and refuses VISIBLY, as the designed
      // Locked state, never as a blank panel.
      await openQueue(page);
      await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'locked');
      await expect(page.locator('#s4 .ms-locked')).toBeVisible();
      await expect(page.locator('#s4')).toContainText('Redemption stats');
      await expect(page.locator('#s4 .badge')).toHaveCount(0);
    } finally {
      await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await putSlugPerms(page, 'marketing', {
        role_grants: mk.role_grants || [], user_grants: (mk.user_grants || []).map(String),
      });
      await putSlugPerms(page, 'bi', {
        role_grants: bi.role_grants || [], user_grants: (bi.user_grants || []).map(String),
      });
    }
  });

  // ── the manager tier, from the other side ────────────────────────────────
  // A team_member who DOES hold `marketing` is refused by the handler's own
  // tier with {"error":"managers_only"} — a different envelope, the same
  // designed Locked state. Both shapes are rendered; neither is a blank panel.
  test('a team_member with the marketing grant gets the Locked state from managers_only', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedQueueFixture(page);
    const user = await makeUser(page, 'member', ['team_member']);
    await loginAs(page, user.email, USER_PASSWORD);
    await openQueue(page);
    await expect(page.locator('#ms-stats-root')).toHaveAttribute('data-state', 'locked');
    await expect(page.locator('#s4 .ms-locked')).toContainText('Managers only');
  });

  // ── the add-order sheet, and D-3's two suggestion bases ──────────────────
  test('the add-order sheet offers the nearest order with its basis and gap, and matching clears the row', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedQueueFixture(page);
    await openQueue(page);

    // U(1) is UNMATCHED (order '888' typed, Toast has no such order), scanned
    // 12:08; order 202 opened 12:04 — 4 minutes, inside the ±30min window, so
    // the server's basis is `window`. (reconRow suggests for this bucket only.)
    await page.locator(`.msq-row[data-id="${U(1)}"] .msq-fix`).click();
    const sheet = page.locator('#msq-sheet-order');
    await expect(sheet).toBeVisible();
    const chip = sheet.locator('.msq-sug[data-order="202"]');
    await expect(chip).toHaveCount(1);
    // D-3: the basis and the gap are RENDERED, so a far-away order is never
    // presented as a confident nearest match.
    await expect(chip).toContainText('4m');
    await expect(chip).toContainText('within 30 min');

    await chip.click();
    await expect(sheet.locator('#msq-order-input')).toHaveValue('202');
    await sheet.locator('#msq-order-save').click();
    await expect(sheet).toBeHidden();
    await expect(page.locator(`.msq-row[data-id="${U(1)}"]`)).toHaveCount(0);
    await expect(page.locator('#msq-head')).toContainText('3 redemptions need a look');

    // U(2) scanned 20:30; the nearest order is 202 at 12:04 — 8h26m away, OUTSIDE
    // the window, so the server falls to the advisory `business_date` rung and
    // the chip must say so rather than look like a match. D-3 is exactly this:
    // an unconfirmed opened_at zone can put the nearest order an hour or more
    // away, and the UI must not dress that up as a confident match.
    await page.locator(`.msq-row[data-id="${U(2)}"] .msq-fix`).click();
    const chip2 = page.locator('#msq-sheet-order .msq-sug[data-order="202"]');
    await expect(chip2).toContainText('same business date');
    await expect(chip2).toContainText('8h');
  });

  // ── a bad order number is reported, not swallowed ────────────────────────
  test('an order number Toast does not have is reported on the sheet and the row stays open', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedQueueFixture(page);
    await openQueue(page);
    await page.locator(`.msq-row[data-id="${U(1)}"] .msq-fix`).click();
    const sheet = page.locator('#msq-sheet-order');
    await sheet.locator('#msq-order-input').fill('40404');
    await sheet.locator('#msq-order-save').click();
    await expect(sheet.locator('.msq-err')).toContainText('40404');
    await expect(sheet).toBeVisible();
    await sheet.locator('#msq-order-close').click();
    await expect(page.locator(`.msq-row[data-id="${U(1)}"]`)).toHaveCount(1);
  });

  // ── the Marketing health card is what P-KR3 / Q-KR2 read ─────────────────
  test('the Marketing queue carries its own health card with matched / open / declined and the orphan rate', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await seedMoneyFixture(page);
    await openQueue(page);
    const health = page.locator('#msq-health');
    await expect(health.locator('.msq-h[data-k="matched"] b')).toHaveText('3');
    await expect(health.locator('.msq-h[data-k="open"] b')).toHaveText('3');
    await expect(health.locator('.msq-h[data-k="declined"] b')).toHaveText('0');
    await expect(health.locator('.msq-orphan')).toHaveText('40.0%');
    // The basis travels with the figure here too (D-5 parked).
    await expect(health.locator('.msq-orphan-basis')).toContainText('2 of 5');
    await expect(health).toContainText('10%');
  });
});
