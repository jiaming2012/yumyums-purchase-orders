// shoot.mjs — self-verification render for mockup.html.
//
// CLAUDE.md: "This environment is headless — verify via screenshots, not
// imagination." One PNG per State Enumeration Table row per colour scheme, at
// 480px wide (HQ's mobile column). The PNGs are read back with a multimodal
// Read and compared row-by-row against each row's visual contract.
//
// Run from the repo root:  node .planning/phases/sync-rxdb-conflict-notice/screenshots/shoot.mjs
// Playwright is resolved from whichever clone has it installed; nothing is
// installed into this worktree (this card touches no package.json).

import { chromium } from 'playwright';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const here = path.dirname(fileURLToPath(import.meta.url));
const page_url = 'file://' + path.resolve(here, '..', 'mockup.html');

const PLATES = [
    ['success',        'plate-success'],
    ['empty',          'plate-empty'],
    ['loading',        'plate-loading'],
    ['error',          'plate-error'],
    ['edge-novalue',   'plate-edge-novalue'],
    ['edge-many',      'plate-edge-many'],
    ['edge-removed',   'plate-edge-removed'],
    ['edge-storage',   'plate-edge-storage'],
    ['limits',         'plate-limits']
];

const browser = await chromium.launch();
for (const scheme of ['light', 'dark']) {
    const ctx = await browser.newContext({
        viewport: { width: 480, height: 900 },
        deviceScaleFactor: 2,
        colorScheme: scheme
    });
    const page = await ctx.newPage();
    await page.goto(page_url);
    for (const [name, id] of PLATES) {
        const el = page.locator('#' + id);
        await el.scrollIntoViewIfNeeded();
        await el.screenshot({ path: path.join(here, `${name}-${scheme}.png`) });
        console.log(`${name}-${scheme}.png`);
    }
    await ctx.close();
}
await browser.close();
