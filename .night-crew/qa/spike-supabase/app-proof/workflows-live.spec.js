// workflows-live.spec.js — Card `sync-live-in-dev-app-proof` (Activity 5, leg 3).
//
// PROMOTED from .night-crew/qa/spike-supabase/browser-live/workflows-live.spec.js
// (spike F, which proved leg 3 GREEN). This is the card's repo red-first spec: it
// drives the REAL production workflows.html — flag hq_sync_read=on, NO page.route
// stub — so its same-origin RxDB replication makes REAL requests to the HQ
// server's /sync/* proxy, which forwards to the persistent Supabase substrate
// under the REAL per-user RLS policy (hq_can_see_field via the FDW). The ONLY
// source the dev surface (#sync-one-row) reads is `db.responses`, so a `served`
// state carrying our sentinel is a claim the DOM can be checked against.
//
// 🛑 THE RED-FIRST FRAMING (differs from the spike's).
// The spike had two mutually-exclusive tests keyed on EXPECT=served|absent. The
// card asks for the SAME served-asserting spec to PASS with the carrier UP and
// FAIL with it DOWN — "the SAME spec fails when the relay is stopped". So this
// spec has ONE test, which ALWAYS asserts `served`. sync-app-proof.sh runs it
// twice: once with the row carried into the substrate (must exit 0) and once with
// it withheld (must exit non-zero — the app stays `waiting`). The exit-code
// asymmetry across those two runs IS the red-first, gated on the spec's exit code
// (never on `task`, B-163). Everything this spec needs is handed in by
// sync-app-proof.sh via env; the spec starts nothing.
const { test, expect } = require('@playwright/test');

const HQ_URL    = process.env.HQ_APP_PROOF_HQ_URL;
const SESSION   = process.env.HQ_APP_PROOF_SESSION;
const CHECKLIST = process.env.HQ_APP_PROOF_CHECKLIST_ID;
const TEMPLATE  = process.env.HQ_APP_PROOF_TEMPLATE_ID;
const FIELD     = process.env.HQ_APP_PROOF_FIELD_ID;
const USER      = process.env.HQ_APP_PROOF_USER_ID;
const SENTINEL  = process.env.HQ_APP_PROOF_SENTINEL;
const DEADLINE  = Number(process.env.HQ_APP_PROOF_DEADLINE_MS || '20000');

function requireEnv() {
  for (const [k, v] of Object.entries({ HQ_URL, SESSION, CHECKLIST, TEMPLATE, FIELD, USER, SENTINEL })) {
    expect(v, `${k} must be provided by sync-app-proof.sh`).toBeTruthy();
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

// The single served-asserting test. Carrier UP -> passes. Carrier withheld -> the
// app stays `waiting`, this expectation times out, the spec exits non-zero. That
// asymmetry, captured by sync-app-proof.sh across its two runs, IS the red-first.
test('the app shows the round trip: one /saveResponse field surfaces in the real workflows.html dev surface via RxDB through the /sync proxy', async ({ page, context }) => {
  requireEnv();
  const logs = [];
  await primePage(page, context, logs);
  await page.goto(devSurfaceUrl(), { waitUntil: 'domcontentloaded' });

  const panel = page.locator('#sync-one-row');
  try {
    await expect(panel).toHaveAttribute('data-state', 'served', { timeout: DEADLINE });
  } catch (e) {
    // On the red-first run this is the EXPECTED failure — dump the diagnostics
    // (the app should be sitting at `waiting`, all four collections fetched 200)
    // and re-throw so the spec exits non-zero.
    await dumpState(page, logs, 'DEV SURFACE DID NOT REACH served');
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
