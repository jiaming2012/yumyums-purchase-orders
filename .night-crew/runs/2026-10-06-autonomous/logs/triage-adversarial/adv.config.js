// adversarial scratch: no webServer, no DB — blank page + the vendored decoder only
module.exports = { testDir: './tests', timeout: 0, retries: 0, workers: 1, outputDir: process.env.TEST_OUTPUT_DIR || 'test-results-adv', use: { browserName: 'chromium', headless: true } };
