// workflows-live.spec.js — SPIKE F (sync-live-in-dev, leg 3).
//
// 🛑 THE NOVEL INTEGRATION THIS MILESTONE NEVER SPIKED.
//
// C3's fill-view test stubs the substrate with `page.route('**/sync/rest/**')`.
// The close-bar demo (demo-sync.sh) reads with a Node RxDB client, never a
// browser. This spec does neither: it loads the REAL production workflows.html,
// flag hq_sync_read=on, with NO page.route stub, so its same-origin RxDB
// replication makes REAL requests to the HQ server's /sync/* proxy, which
// forwards to the REAL Supabase substrate under the REAL per-user RLS policy.
//
// The ONLY source the dev surface (#sync-one-row) reads is `db.responses` — so a
// `served` state with our sentinel value is a claim the DOM can be checked
// against, not an assertion in a document. Everything this spec needs is handed
// in by spike-f-browser-live.sh via env; the spec starts nothing.
const { test, expect } = require('@playwright/test');

const HQ_URL    = process.env.SPIKE_F_HQ_URL;
const SESSION   = process.env.SPIKE_F_SESSION;
const CHECKLIST = process.env.SPIKE_F_CHECKLIST_ID;
const TEMPLATE  = process.env.SPIKE_F_TEMPLATE_ID;
const FIELD     = process.env.SPIKE_F_FIELD_ID;
const USER      = process.env.SPIKE_F_USER_ID;
const SENTINEL  = process.env.SPIKE_F_SENTINEL;
const DEADLINE  = Number(process.env.SPIKE_F_DEADLINE_MS || '20000');
// EXPECT=served  -> armed pass: the row must surface (assert served + sentinel).
// EXPECT=absent  -> red-first pass: the row must NOT surface within the bound.
const EXPECT    = process.env.SPIKE_F_EXPECT || 'served';

function requireEnv() {
  for (const [k, v] of Object.entries({ HQ_URL, SESSION, CHECKLIST, TEMPLATE, FIELD, USER, SENTINEL })) {
    expect(v, `${k} must be provided by spike-f-browser-live.sh`).toBeTruthy();
  }
}

function devSurfaceUrl() {
  return `/workflows.html?hq_sync_read=on`
    + `&hq_sync_checklist=${encodeURIComponent(CHECKLIST)}`
    + `&hq_sync_template=${encodeURIComponent(TEMPLATE)}`
    + `&hq_sync_field=${encodeURIComponent(FIELD)}`
    + `&hq_sync_user=${encodeURIComponent(USER)}`;
}

// Attach the REAL hq_session cookie and a /sync-request tap for diagnosability.
async function primePage(page, context, logs) {
  await context.addCookies([{ name: 'hq_session', value: SESSION, url: HQ_URL }]);
  page.on('console', (m) => logs.push(`[console.${m.type()}] ${m.text()}`));
  page.on('requestfailed', (r) => logs.push(`[requestfailed] ${r.url()} :: ${r.failure() && r.failure().errorText}`));
  page.on('response', (r) => {
    const u = r.url();
    if (u.includes('/sync/')) logs.push(`[sync ${r.status()}] ${u.replace(HQ_URL, '')}`);
  });
}

async function dumpState(page, logs, label) {
  const state = await page.locator('#sync-one-row').getAttribute('data-state').catch(() => '(none)');
  const note = await page.locator('#sync-one-row-note').textContent().catch(() => '');
  const val = await page.locator('#sync-one-row-value').textContent().catch(() => '');
  console.log(`---- ${label}: data-state=${state} note=${JSON.stringify(note)} value=${JSON.stringify(val)} ----`);
  console.log(logs.join('\n'));
}

test('armed: one field written via /saveResponse surfaces in the real workflows.html dev surface via RxDB through the /sync proxy', async ({ page, context }) => {
  test.skip(EXPECT !== 'served', 'armed pass only');
  requireEnv();
  const logs = [];
  await primePage(page, context, logs);
  await page.goto(devSurfaceUrl(), { waitUntil: 'domcontentloaded' });

  const panel = page.locator('#sync-one-row');
  try {
    await expect(panel).toHaveAttribute('data-state', 'served', { timeout: DEADLINE });
  } catch (e) {
    await dumpState(page, logs, 'DEV SURFACE DID NOT REACH served (armed)');
    throw e;
  }
  // The value on screen came out of db.responses. It must be OUR field's sentinel
  // — not a stray pre-existing substrate row — or the green is a coincidence.
  const value = await panel.locator('#sync-one-row-value').textContent();
  const id = await panel.locator('#sync-one-row-id').textContent();
  console.log(`served: id=${id} value=${value}`);
  expect(value, 'the RxDB-served value must contain the sentinel written through /saveResponse').toContain(SENTINEL);
  expect(id && id.length, 'the served row must carry a real substrate id').toBeGreaterThan(0);
});

test('red-first: with no carrier the same page must NOT reach served within the bound', async ({ page, context }) => {
  test.skip(EXPECT !== 'absent', 'red-first pass only');
  requireEnv();
  const logs = [];
  await primePage(page, context, logs);
  await page.goto(devSurfaceUrl(), { waitUntil: 'domcontentloaded' });

  const panel = page.locator('#sync-one-row');
  // The panel opens replication and sits at `waiting`. If it reaches `served`
  // with NO row carried into the substrate, the armed assertion is vacuous and
  // the whole spike is untrustworthy — so this expectation is the vacuity guard.
  await page.waitForTimeout(DEADLINE);
  const state = await panel.getAttribute('data-state');
  await dumpState(page, logs, 'red-first final');
  expect(state, 'with no row in the substrate the dev surface must not be served — if it is, the assertion is vacuous').not.toBe('served');
});
