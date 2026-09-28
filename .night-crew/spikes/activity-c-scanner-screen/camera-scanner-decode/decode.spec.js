// decode.spec.js — the browser half of the decode-and-hash-chain spike.
// Chromium loads the plain page (vendored html5-qrcode, no build), decodes the
// generated QR PNG via the library's file-scan path, extracts the token from
// the #10 URL wrapper, digests it with WebCrypto, and the assertions pin all
// three against the values the wrapper script passes in (payload, token, and
// the committed seed literal).
const { test, expect } = require('@playwright/test');
const path = require('path');

test('QR decodes and WebCrypto hash matches the committed seed contract', async ({ page }) => {
  const web = path.join(__dirname, 'web');
  page.on('console', (m) => console.log(`  [page] ${m.type()}: ${m.text()}`));
  await page.goto('file://' + path.join(web, 'decode.html'));
  await page.setInputFiles('#file', path.join(web, 'qr.png'));
  // state:'attached', not the default 'visible' — #out is an empty (hidden)
  // div that only carries data attributes; first run red on exactly this.
  await page.waitForSelector('#out[data-state]', { state: 'attached', timeout: 20_000 });

  const state = await page.getAttribute('#out', 'data-state');
  if (state !== 'done') {
    throw new Error(`page chain failed: ${await page.getAttribute('#out', 'data-err')}`);
  }
  const payload = await page.getAttribute('#out', 'data-payload');
  const token = await page.getAttribute('#out', 'data-token');
  const hash = await page.getAttribute('#out', 'data-hash');
  console.log(`  decoded payload : ${payload}`);
  console.log(`  extracted token : ${token}`);
  console.log(`  webcrypto hash  : ${hash}`);

  expect(payload).toBe(process.env.SPIKE_EXPECT_PAYLOAD);
  expect(token).toBe(process.env.SPIKE_EXPECT_TOKEN);
  expect(hash).toBe(process.env.SPIKE_EXPECT_HASH);
});
