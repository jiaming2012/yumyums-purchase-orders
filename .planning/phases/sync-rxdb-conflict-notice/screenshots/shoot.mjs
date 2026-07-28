// shoot.mjs — self-verification render + measurement for mockup.html.
//
// CLAUDE.md: "This environment is headless — verify via screenshots, not
// imagination." One PNG per State Enumeration Table row per colour scheme, at
// 480px wide (HQ's mobile column). The PNGs are read back with a multimodal
// Read and compared row-by-row against each row's visual contract.
//
// Two `done_when:` rows are NOT eye-checkable and are measured here instead of
// being grepped out of the stylesheet:
//   - horizontal overflow  -> document scrollWidth vs clientWidth at 480px
//   - touch targets        -> getBoundingClientRect over EVERY interactive
//                             element in the design (not just the two classes
//                             that were already known to pass)
// Both print a PASS/FAIL line and set a non-zero exit code on failure, so the
// check cannot quietly rot.
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
    ['outcomes',       'plate-outcomes'],
    ['empty',          'plate-empty'],
    ['loading',        'plate-loading'],
    ['error',          'plate-error'],
    ['edge-novalue',   'plate-edge-novalue'],
    ['edge-many',      'plate-edge-many'],
    ['edge-longvalue', 'plate-edge-longvalue'],
    ['edge-removed',   'plate-edge-removed'],
    ['edge-storage',   'plate-edge-storage'],
    ['limits',         'plate-limits']
];

// Every interactive element of the DESIGN. `.cap`, `.note` and `.doc-hd` are
// mockup chrome and are deliberately out of scope.
const TAP_TARGETS = '.cf-btn, .cg-all, .cf-done-undo, .sc-close, .cn-banner-go, .sc-err button, .sc-empty button';

let failed = false;

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

    // ── measurement 1: no horizontal overflow at 480px ─────────────────────
    const ov = await page.evaluate(() => ({
        scrollWidth: document.documentElement.scrollWidth,
        clientWidth: document.documentElement.clientWidth
    }));
    const ov_ok = ov.scrollWidth === ov.clientWidth;
    if (!ov_ok) failed = true;
    console.log(`  [${scheme}] overflow: scrollWidth=${ov.scrollWidth} clientWidth=${ov.clientWidth} -> ${ov_ok ? 'PASS' : 'FAIL'}`);

    // ── measurement 2: every interactive element is a >=44px target ────────
    const bad = await page.evaluate((sel) => {
        const out = [];
        for (const el of document.querySelectorAll(sel)) {
            const r = el.getBoundingClientRect();
            if (r.width < 44 || r.height < 44) {
                out.push(`${el.className || el.tagName} "${(el.textContent || '').trim().slice(0, 22)}" ${Math.round(r.width)}x${Math.round(r.height)}`);
            }
        }
        return out;
    }, TAP_TARGETS);
    const n = await page.locator(TAP_TARGETS).count();
    if (bad.length) failed = true;
    console.log(`  [${scheme}] tap targets: ${n} measured, ${bad.length} under 44px -> ${bad.length ? 'FAIL' : 'PASS'}`);
    for (const b of bad) console.log(`      ${b}`);

    await ctx.close();
}
await browser.close();

if (failed) {
    console.error('\nself-verification FAILED — see FAIL lines above');
    process.exit(1);
}
console.log('\nself-verification PASS — 11 plates x 2 schemes, no overflow, all targets >=44px');
