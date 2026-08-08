// playwright.browser-live.config.js — SPIKE F (sync-live-in-dev, leg 3).
//
// This config drives a REAL Chromium against the REAL workflows.html served by
// the REAL HQ server the harness (spike-f-browser-live.sh) started — NOT the
// spike's bespoke spike.html, and NOT serve.mjs. There is deliberately no
// `webServer` block: the HQ server, the substrate and the FDW repoint are all
// stood up by the bash harness before this ever runs, and this config must
// never start a second server of its own (that server would have a different
// database and a closed /sync proxy — the exact orphan-attach failure spike C's
// header warns about).
//
// @playwright/test resolves up the tree to the repo-root node_modules, the same
// way browser/playwright.spike.config.js does — nothing is installed here.
const { defineConfig } = require('@playwright/test');

const baseURL = process.env.SPIKE_F_HQ_URL;
if (!baseURL) {
  // Fail LOUD at config load, not with a confusing per-test error. The harness
  // owns this variable; its absence means the harness is being bypassed.
  throw new Error('SPIKE_F_HQ_URL is unset — this config is driven by spike-f-browser-live.sh, run that instead');
}

module.exports = defineConfig({
  testDir: __dirname,
  testMatch: /workflows-live\.spec\.js$/,
  // Generous: one initial RxDB replication through the proxy plus render. The
  // per-assertion bound is SPIKE_F_DEADLINE_MS inside the spec; this is only the
  // outer ceiling so a hung page fails rather than hanging the harness.
  timeout: 120000,
  retries: 0,
  workers: 1,
  fullyParallel: false,
  reporter: [['list']],
  use: {
    baseURL,
    headless: true,
    browserName: 'chromium',
    // The dev surface is inline module JS reading RxDB directly — it needs no
    // service worker, and blocking it (the repo-wide default) guarantees a stale
    // precached workflows.html can never serve in place of the one the HQ server
    // just built from this tree.
    serviceWorkers: 'block',
    ignoreHTTPSErrors: true,
  },
});
