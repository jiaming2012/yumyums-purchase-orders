// playwright.spike.config.js — a SEPARATE Playwright project for the browser
// spike, and the reason it has to be separate is the whole point of leg 3.
//
// ============================================================================
// THE REPO-WIDE `serviceWorkers: 'block'` SETTING IS NOT TOUCHED.
// ============================================================================
// The root `playwright.config.js:60` sets `use.serviceWorkers = 'block'`. That
// setting is LOAD-BEARING for 500+ existing tests: HQ's Workbox SW precaches
// every page and would serve stale HTML into the middle of a suite that mutates
// state on every step. Relaxing it repo-wide to prove one spike leg would be an
// unmapped blast radius across the entire test suite.
//
// So the service-worker leg is proved HERE, in a dedicated config that enables
// service workers for these specs and these specs only. Nothing in this file is
// read by `task test`; the root config's `projects` do not include it and its
// testDir is outside `./tests`.
//
// Run it explicitly:
//   cd .night-crew/qa/spike-supabase/browser
//   npx playwright test -c playwright.spike.config.js
//
// (@playwright/test resolves up the tree to the repo-root node_modules. This
//  directory deliberately does NOT install its own copy — see package.json.)

const { defineConfig } = require('@playwright/test');

const httpPort = process.env.SPIKE_HTTP_PORT || '8497';

module.exports = defineConfig({
    testDir: './specs',
    // Legs 4 and 5 involve real leader-election timeouts and a real token TTL
    // elapsing. 30 s (the root config's timeout) is not enough for either.
    timeout: 180000,
    // A spike proves things once. A retry here would let a flaky leg look
    // settled, which is the opposite of what a verdict needs.
    retries: 0,
    // Legs share one Postgres fixture table and one origin's IndexedDB.
    // Serial, for the same reason the root suite is serial.
    workers: 1,
    outputDir: 'test-results',
    reporter: [['list']],
    use: {
        baseURL: `http://127.0.0.1:${httpPort}`,
        headless: true,
        // ↓↓↓ THE ONE LINE THIS SEPARATE CONFIG EXISTS FOR ↓↓↓
        serviceWorkers: 'allow',
        // Chromium only: HQ's install target is iOS/Android, but Playwright's
        // WebKit is not Safari-on-iOS and a green here would over-claim. What
        // this suite can and cannot say about Safari is stated in the verdict.
        browserName: 'chromium'
    },
    webServer: {
        command: 'node serve.mjs',
        url: `http://127.0.0.1:${httpPort}/__spike/config`,
        cwd: __dirname,
        reuseExistingServer: false,
        timeout: 30000,
        stdout: 'pipe'
    }
});
