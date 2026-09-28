// xstate-load.spec.js — Chromium loads the vendored single-file XState build
// from a plain page (no bundler) and runs a parallel-machine transition.
const { test, expect } = require('@playwright/test');
const path = require('path');

test('vendored xstate ESM loads in a plain page and drives a parallel machine', async ({ page }) => {
  page.on('console', (m) => console.log(`  [page] ${m.type()}: ${m.text()}`));
  await page.goto('file://' + path.join(__dirname, 'web', 'xstate-load.html'));
  await page.waitForSelector('body[data-state]', { state: 'attached', timeout: 15_000 });
  const state = await page.getAttribute('body', 'data-state');
  if (state !== 'ok') {
    throw new Error(`browser load failed: state=${state} err=${await page.getAttribute('body', 'data-err')} result=${await page.getAttribute('body', 'data-result')}`);
  }
  console.log(`  parallel snapshot in-browser: ${await page.getAttribute('body', 'data-result')}`);
  expect(state).toBe('ok');
});
