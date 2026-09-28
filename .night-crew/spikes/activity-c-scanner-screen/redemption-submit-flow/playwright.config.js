// Spike-local Playwright config — NO repo webServer.
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: __dirname,
  testMatch: 'xstate-load.spec.js',
  timeout: 30_000,
  use: { browserName: 'chromium' },
  reporter: [['list']],
});
