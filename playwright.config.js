const { defineConfig } = require('@playwright/test');
const { defineBddConfig } = require('playwright-bdd');
// scripts/reset-e2e-db.js is the ONE place the e2e Postgres coordinates are
// computed and the ONE place the database is reset. It carries what used to be
// duplicated here:
//   * DB_PORT 5434 + role `hqtest` = yumyums-test-pg, the TEST-ONLY container
//     (docker-compose.test.yml, `task test:db:up`). 🛑 NOT :5433 — that is
//     yumyums-dev-pg, which serves https://hq.yumyums.kitchen, and pointing a
//     harness that issues DROP DATABASE at it is BACKLOG B-141 (ledger decision
//     155, card `test-cluster-separation`). Host :5432 is infra-postgres-1
//     (slack-trading). This file used to carry its OWN copy of that default;
//     now there is one copy, in the helper.
//   * TEST_DB_NAME defaults to hq_test_e2e — Playwright's OWN database. The Go
//     suite owns hq_test_go. They shared one hq_test until 2026-07-21 (audit
//     surface #3): Go TestMains TRUNCATE `users`, so a concurrent `task test:go`
//     would log every browser context out mid-suite. Separate databases make the
//     collision impossible.
//   * the guard that refuses to DROP anything not named like a test database.
const { resolveE2eDb } = require('./scripts/reset-e2e-db');

const db = resolveE2eDb();
// TEST_DB_NAME / TEST_PORT allow running multiple isolated stacks in parallel
// (each against its own database + server port).
//
// The default test port is deliberately NOT the dev-server port (8089). With
// reuseExistingServer:true (local runs), if the test stack targeted 8089 and a
// dev server were already running there, Playwright would silently REUSE it and
// run the whole suite against the live dev database — corrupting dev data and
// failing every state-dependent test. Defaulting to 8199 makes a local test run
// always spawn its own clean server, even with a dev server up on 8089.
const testPort = process.env.TEST_PORT || '8199';
const testDbUrl = db.testUrl;

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
    // 🛑 `node scripts/reset-e2e-db.js &&` IS THE B-76 FIX. Do not move it, and
    // in particular do not "tidy" it into a `globalSetup`.
    //
    // Before it, the only DROP/CREATE of hq_test_e2e in the repo lived inside
    // `task test`. night-crew.toml runs `npx playwright test` DIRECTLY for both
    // [e2e] suite and [e2e] subset, so no gate leg — full or subset — had ever
    // reset the database. Every suite figure this milestone quoted was taken
    // against an accumulating dataset: a red could be an artifact of a previous
    // run, and so could a GREEN.
    //
    // Why here and nowhere else:
    //   * a `globalSetup` runs AFTER this server. See
    //     node_modules/playwright/lib/runner/tasks.js:100-110 —
    //     createGlobalSetupTasks appends createPluginSetupTasks(config), which is
    //     where the webServer plugin starts the server, BEFORE config.globalSetups.
    //     Dropping the database there would drop it out from under a server that
    //     had already connected, migrated and seeded it.
    //   * top-level code in THIS file is re-executed in every worker process
    //     (the worker load carries TEST_WORKER_INDEX=0) and also runs under
    //     `npx bddgen` and `npx playwright test --list`.
    //   * webServer.command runs exactly once, in the parent, strictly before any
    //     test and strictly before the server exists — and cannot be skipped by a
    //     CLI argument, so it fires on the SUBSET path too.
    //   * with NIGHTCREW_ENV_URL set this whole object is undefined and no reset
    //     runs, which is correct: the provisioned stack owns its own database.
    //
    // The server re-runs goose migrations on startup, so a bare CREATE is enough.
    //
    // TOAST_SYNC_INTERVAL=0 disables the Toast in-process worker so the
    // server starts without TOAST_SFTP_KEY_PATH credentials. The Toast
    // worker is not exercised by E2E tests; cmd/sync-toast covers ingest.
    // MERCURY_API_KEY= / ANTHROPIC_API_KEY= (blank) override any real keys
    // the Taskfile dotenv loads from backend/.env — otherwise the receipt
    // worker and the on-demand POST /sync-receipts would ingest LIVE Mercury
    // transactions into the test DB mid-suite (all sync tests mock these
    // routes at the network layer, so nothing needs the real path).
    // 🛑 `node scripts/write-version-json.js &&` IS THE B-92 FIX, and it sits
    // in this chain for exactly B-76's reason: this command runs once, in the
    // parent, before any test and before the server exists, and no CLI argument
    // can skip it — so it fires on night-crew.toml's `subset` leg too.
    //
    // `version.json` is a git-ignored build artifact that nonetheless SHIPS
    // (sw.js precaches it; index.html's version line reads it). `task test`
    // generates it via its `sw` dep, but night-crew.toml:33-34 runs
    // `npx playwright test` DIRECTLY and this server serves `STATIC_DIR=../` —
    // the bare worktree. Without this link, `GET /version.json` 404s in any
    // worktree where `node build-sw.js` has never run, and
    // tests/version-badge.spec.js reds for a reason that has nothing to do with
    // the change under test. A gate that hands every future card a red it did
    // not cause is the thing A1 landed to prevent.
    //
    // 🛑 It is NOT `node build-sw.js`. That reads git HEAD and rewrites sw.js,
    // so a gate leg would dirty the tree mid-run and race the B-37 committed-
    // artifact invariant. scripts/write-version-json.js writes the one file and
    // nothing else, from the same payload definition build-sw.js uses.
    //
    // 🛑 It runs AFTER the reset, not before it. The reset is B-76's fix and
    // must stay the first link; version.json is a static file on disk and has
    // no interaction with the database either way.
    //
    // ZOHO_CLIQ_* / SMTP_* blanked too (cross-contamination audit 2026-07-21,
    // surface #5): the root Taskfile's `dotenv: ['backend/.env']` injects LIVE
    // credentials into every task launched from the main checkout, and the alert
    // queue is NOT gated by E2E_DISABLE_SCHEDULERS — alertQ.Start runs
    // unconditionally and purchasing/service.go NotifyVendorComplete enqueues
    // from a request path the suite exercises. Without these, an E2E run can
    // deliver a real Cliq message and a real SMTP email to live crew.
    command: `node scripts/reset-e2e-db.js && node scripts/write-version-json.js && cd backend && PORT=${testPort} DB_URL="${testDbUrl}" STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= go run ./cmd/server/`,
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
