const { defineConfig } = require('@playwright/test');
const { defineBddConfig } = require('playwright-bdd');

const dbHost = process.env.DB_HOST || 'localhost';
const dbPort = process.env.DB_PORT || '5432';
const dbUser = process.env.DB_USER || 'yumyums';
const dbPass = process.env.DB_PASS || 'yumyums';
const testDbUrl = `postgres://${dbUser}:${dbPass}@${dbHost}:${dbPort}/hq_test?sslmode=disable&TimeZone=America/New_York`;

const bddTestDir = defineBddConfig({
  features: './features/**/*.feature',
  steps: './features/steps/**/*.js',
});

module.exports = defineConfig({
  timeout: 30000,
  retries: 1,
  use: {
    baseURL: 'http://localhost:8089',
    headless: true,
    // Block service worker in tests to prevent caching interference
    serviceWorkers: 'block',
  },
  webServer: {
    // TOAST_SYNC_INTERVAL=0 disables the Toast in-process worker so the
    // server starts without TOAST_SFTP_KEY_PATH credentials. The Toast
    // worker is not exercised by E2E tests; cmd/sync-toast covers ingest.
    command: `cd backend && PORT=8089 DB_URL="${testDbUrl}" STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml TOAST_SYNC_INTERVAL=0 go run ./cmd/server/`,
    url: 'http://localhost:8089/api/v1/health',
    reuseExistingServer: !process.env.CI,
    timeout: 60000,
  },
  projects: [
    { name: 'chromium', testDir: './tests', use: { browserName: 'chromium' } },
    { name: 'bdd', testDir: bddTestDir, use: { browserName: 'chromium' } },
  ],
});
