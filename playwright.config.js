const { defineConfig } = require('@playwright/test');
const { defineBddConfig } = require('playwright-bdd');

const dbHost = process.env.DB_HOST || 'localhost';
// 5433 = yumyums-dev-pg, the container that actually serves HQ. Host :5432 is
// bound by infra-postgres-1 (slack-trading), which has no `yumyums` role.
// MUST stay in sync with backend/Taskfile.yml's DB_PORT default — this file
// carries its OWN default and does not read the Taskfile's.
const dbPort = process.env.DB_PORT || '5433';
const dbUser = process.env.DB_USER || 'yumyums';
const dbPass = process.env.DB_PASS || 'yumyums';
// TEST_DB_NAME / TEST_PORT allow running multiple isolated stacks in parallel
// (each against its own database + server port).
//
// The default test port is deliberately NOT the dev-server port (8089). With
// reuseExistingServer:true (local runs), if the test stack targeted 8089 and a
// dev server were already running there, Playwright would silently REUSE it and
// run the whole suite against the live dev database — corrupting dev data and
// failing every state-dependent test. Defaulting to 8199 makes a local test run
// always spawn its own clean server, even with a dev server up on 8089.
// hq_test_e2e — Playwright's OWN database. The Go suite owns hq_test_go.
// They shared one hq_test until 2026-07-21 (audit surface #3): Go TestMains
// TRUNCATE `users`, so a concurrent `task test:go` would log every browser
// context out mid-suite. Separate databases make the collision impossible.
const testDbName = process.env.TEST_DB_NAME || 'hq_test_e2e';
const testPort = process.env.TEST_PORT || '8199';
const testDbUrl = `postgres://${dbUser}:${dbPass}@${dbHost}:${dbPort}/${testDbName}?sslmode=disable&TimeZone=America/New_York`;

// When night-crew provisions an ephemeral environment it exports that stack's
// base URL as NIGHTCREW_ENV_URL. In that mode we target the provisioned stack
// and skip the local webServer entirely — otherwise Playwright would spin up its
// own server against localhost Postgres and the suite would pass against the
// wrong stack (a silent-green false pass). Unset (local dev / CI): unchanged.
const nightcrewEnvUrl = process.env.NIGHTCREW_ENV_URL;

const bddTestDir = defineBddConfig({
  features: './features/**/*.feature',
  steps: './features/steps/**/*.js',
});

module.exports = defineConfig({
  timeout: 30000,
  retries: 1,
  // Isolate the artifact/trace output dir per stack. Playwright wipes this dir at
  // the start of every run, so two concurrent runs sharing it (e.g. `task test`
  // and `task test:ui`) delete each other's in-flight traces mid-test, producing
  // spurious "ENOENT ...recording.stacks / Test ended" failures. TEST_OUTPUT_DIR
  // (set by task test:ui) gives each stack its own dir.
  outputDir: process.env.TEST_OUTPUT_DIR || 'test-results',
  // This suite shares ONE server + ONE Postgres database across all spec files.
  // Running spec files in parallel makes them stomp on each other's data (and
  // contend on the single go-run server), producing timeout/pollution failures
  // that don't reproduce in isolation. Default to serial. To parallelize, give
  // each worker its own stack via TEST_DB_NAME + TEST_PORT and set PW_WORKERS.
  workers: process.env.PW_WORKERS ? Number(process.env.PW_WORKERS) : 1,
  use: {
    baseURL: nightcrewEnvUrl || `http://localhost:${testPort}`,
    headless: true,
    // Block service worker in tests to prevent caching interference
    serviceWorkers: 'block',
  },
  // Skip the self-spawned server when night-crew hands us a provisioned env.
  webServer: nightcrewEnvUrl ? undefined : {
    // TOAST_SYNC_INTERVAL=0 disables the Toast in-process worker so the
    // server starts without TOAST_SFTP_KEY_PATH credentials. The Toast
    // worker is not exercised by E2E tests; cmd/sync-toast covers ingest.
    // MERCURY_API_KEY= / ANTHROPIC_API_KEY= (blank) override any real keys
    // the Taskfile dotenv loads from backend/.env — otherwise the receipt
    // worker and the on-demand POST /sync-receipts would ingest LIVE Mercury
    // transactions into the test DB mid-suite (all sync tests mock these
    // routes at the network layer, so nothing needs the real path).
    // ZOHO_CLIQ_* / SMTP_* blanked too (cross-contamination audit 2026-07-21,
    // surface #5): the root Taskfile's `dotenv: ['backend/.env']` injects LIVE
    // credentials into every task launched from the main checkout, and the alert
    // queue is NOT gated by E2E_DISABLE_SCHEDULERS — alertQ.Start runs
    // unconditionally and purchasing/service.go NotifyVendorComplete enqueues
    // from a request path the suite exercises. Without these, an E2E run can
    // deliver a real Cliq message and a real SMTP email to live crew.
    command: `cd backend && PORT=${testPort} DB_URL="${testDbUrl}" STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= go run ./cmd/server/`,
    url: `http://localhost:${testPort}/api/v1/health`,
    // Unconditionally false (audit surface #2): reuse has cost four runs. The
    // 8199 default protects against reusing the DEV server, but the same
    // mechanism silently reuses a FOREIGN TEST server from another worktree —
    // running the suite green against someone else's database. Never reuse.
    reuseExistingServer: false,
    timeout: 60000,
  },
  projects: [
    { name: 'chromium', testDir: './tests', use: { browserName: 'chromium' } },
    { name: 'bdd', testDir: bddTestDir, use: { browserName: 'chromium' } },
  ],
});
