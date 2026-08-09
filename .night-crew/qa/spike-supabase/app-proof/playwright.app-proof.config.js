// playwright.app-proof.config.js — Card `sync-live-in-dev-app-proof` (Activity 5, leg 3).
//
// PROMOTED from browser-live/playwright.browser-live.config.js (spike F). Drives a
// REAL Chromium against the REAL workflows.html served by the REAL HQ server the
// harness (sync-app-proof.sh) started — NOT a bespoke spike.html, and NOT
// serve.mjs. There is deliberately no `webServer` block: the HQ server, the
// persistent substrate and the FDW pointing are all stood up by the bash harness
// before this ever runs, and this config must never start a second server of its
// own (that server would have a different database and a closed /sync proxy — the
// orphan-attach failure the spike headers warn about).
//
// @playwright/test resolves up the tree to the repo-root node_modules — nothing is
// installed here.
const { defineConfig } = require('@playwright/test');

const baseURL = process.env.HQ_APP_PROOF_HQ_URL;
if (!baseURL) {
  // Fail LOUD at config load, not with a confusing per-test error. The harness
  // owns this variable; its absence means the harness is being bypassed.
  throw new Error('HQ_APP_PROOF_HQ_URL is unset — this config is driven by sync-app-proof.sh, run that instead');
}

module.exports = defineConfig({
  testDir: __dirname,
  testMatch: /workflows-live\.spec\.js$/,
  // Generous: one initial RxDB replication through the proxy plus render. The
  // per-assertion bound is HQ_APP_PROOF_DEADLINE_MS inside the spec; this is only
  // the outer ceiling so a hung page fails rather than hanging the harness.
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
