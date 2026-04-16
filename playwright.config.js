const { defineConfig } = require('@playwright/test');

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
    command: 'cd backend && PORT=8089 DB_URL="postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable&TimeZone=America/New_York" STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml go run ./cmd/server/',
    url: 'http://localhost:8089/api/v1/health',
    reuseExistingServer: !process.env.CI,
    timeout: 60000,
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
});
