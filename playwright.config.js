const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: './tests',
  timeout: 30000,
  use: {
    baseURL: 'http://localhost:8080',
    headless: true,
  },
  webServer: {
    command: 'cd backend && DB_URL=postgres://postgres:postgres@localhost:5432/hq_test?sslmode=disable STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml go run ./cmd/server/',
    url: 'http://localhost:8080/api/v1/health',
    reuseExistingServer: !process.env.CI,
    timeout: 30000,
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
});
