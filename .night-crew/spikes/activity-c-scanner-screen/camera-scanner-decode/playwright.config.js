// Spike-local Playwright config — NO repo webServer (the root config boots the
// Go dev stack; this spike needs only Chromium + a file:// page).
const { defineConfig } = require('@playwright/test');

module.exports = defineConfig({
  testDir: __dirname,
  testMatch: 'decode.spec.js',
  timeout: 30_000,
  use: { browserName: 'chromium' },
  reporter: [['list']],
});
