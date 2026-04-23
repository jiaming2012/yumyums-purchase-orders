const { defineConfig } = require('@playwright/test');

const dbHost = process.env.DB_HOST || 'localhost';
const dbPort = process.env.DB_PORT || '5432';
const dbUser = process.env.DB_USER || 'yumyums';
const dbPass = process.env.DB_PASS || 'yumyums';
const testDbUrl = `postgres://${dbUser}:${dbPass}@${dbHost}:${dbPort}/hq_test?sslmode=disable&TimeZone=America/New_York`;

module.exports = defineConfig({
  testDir: './tests',
  timeout: 30000,
  retries: 1,
  use: {
    baseURL: 'http://localhost:8089',
    headless: true,
    // Block service worker in tests to prevent caching interference
    serviceWorkers: 'block',
  },
  webServer: {
    command: `cd backend && PORT=8089 DB_URL="${testDbUrl}" STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml go run ./cmd/server/`,
    url: 'http://localhost:8089/api/v1/health',
    reuseExistingServer: !process.env.CI,
    timeout: 60000,
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
});
