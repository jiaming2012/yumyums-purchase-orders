// shoot.mjs — self-verification render + measurement for mockup.html.
//
// CLAUDE.md: "This environment is headless — verify via screenshots, not
// imagination." One PNG per State Enumeration Table row per colour scheme, at
// 480px wide (HQ's mobile column). The PNGs are read back with a multimodal
// Read and compared row-by-row against each row's visual contract.
//
// Several `done_when:` rows are NOT eye-checkable and are measured here instead
// of being grepped out of the stylesheet:
//   - horizontal overflow  -> document scrollWidth vs clientWidth at 480px
//   - touch targets        -> getBoundingClientRect over EVERY interactive
//                             element in the design (not just the two classes
//                             that were already known to pass)
//   - A-1 two figures      -> EVERY banner must carry both the "what happened"
//                             headline and a still-to-review figure. A banner
//                             carrying one number is the exact defect ledger
//                             T-26 decision 82 was filed against.
//   - A-1 no truncation    -> every banner LINE measured for overflow at 480px.
//                             This is the card's PARK trigger, so it is measured
//                             rather than asserted: if the two figures cannot be
//                             drawn on a phone without truncation, the mockup
//                             answers the wrong question.
//   - A-2 names the loss   -> every control whose label begins with "Restore"
//                             must also say what it replaces.
// Each prints a PASS/FAIL line and sets a non-zero exit code on failure, so the
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
    ['success',          'plate-success'],
    ['a1-banner',        'plate-a1-banner'],
    ['outcomes',         'plate-outcomes'],
    ['empty',            'plate-empty'],
    ['loading',          'plate-loading'],
    ['error',            'plate-error'],
    ['edge-novalue',     'plate-edge-novalue'],
    ['edge-many',        'plate-edge-many'],
    ['a2-confirm',       'plate-a2-confirm'],
    ['edge-longvalue',   'plate-edge-longvalue'],
    ['edge-removed',     'plate-edge-removed'],
    ['edge-storage',     'plate-edge-storage'],
    ['openq-count-a',    'plate-openq-count-a'],
    ['openq-count-b',    'plate-openq-count-b'],
    ['openq-retention',  'plate-openq-retention'],
    ['limits',           'plate-limits']
];

// Every interactive element of the DESIGN. `.cap`, `.note` and `.doc-hd` are
// mockup chrome and are deliberately out of scope.
const TAP_TARGETS = '.cf-btn, .cg-all, .cf-done-undo, .sc-close, .cn-banner-go, .sc-err button, .sc-empty button, .cfm-go, .cfm-cancel';

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

    // ── measurement 3: A-1 — EVERY banner carries BOTH figures ─────────────
    // "3 answers were overwritten" alone is the defect. A still-to-review
    // figure must be present and must be a number (or the all-clear form).
    const b3 = await page.evaluate(() => {
        const out = [];
        const re = /(\d+ still to review|All \d+ reviewed)/;
        for (const b of document.querySelectorAll('.cn-banner')) {
            const hd = b.querySelector('.cn-banner-hd');
            const open = b.querySelector('.cn-banner-open');
            const plate = b.closest('.plate')?.id || '?';
            if (!hd) { out.push(`${plate}: no .cn-banner-hd (what happened)`); continue; }
            if (!open) { out.push(`${plate}: no .cn-banner-open (still-to-review figure) — banner reads only "${hd.textContent.trim()}"`); continue; }
            const t = open.textContent.replace(/\s+/g, ' ').trim();
            if (!re.test(t)) out.push(`${plate}: still-to-review line does not carry a figure: "${t}"`);
        }
        return { bad: out, total: document.querySelectorAll('.cn-banner').length };
    });
    if (b3.bad.length) failed = true;
    console.log(`  [${scheme}] A-1 two figures: ${b3.total} banners, ${b3.bad.length} carrying only one -> ${b3.bad.length ? 'FAIL' : 'PASS'}`);
    for (const b of b3.bad) console.log(`      ${b}`);

    // ── measurement 4: A-1 — no banner LINE truncates at 480px ─────────────
    // The card's PARK trigger. Measured per line, not per page: the page can
    // pass the overflow check while a line inside the banner is clipped.
    const b4 = await page.evaluate(() => {
        const out = [];
        const sel = '.cn-banner-hd, .cn-banner-open, .cn-banner-unid, .cn-banner-sub';
        for (const el of document.querySelectorAll(sel)) {
            const plate = el.closest('.plate')?.id || '?';
            const cs = getComputedStyle(el);
            const clipped = el.scrollWidth > el.clientWidth + 1;
            const ellipsis = cs.textOverflow === 'ellipsis' && cs.overflow !== 'visible';
            if (clipped || ellipsis) {
                out.push(`${plate} .${el.className}: scrollWidth=${el.scrollWidth} clientWidth=${el.clientWidth}${ellipsis ? ' text-overflow:ellipsis' : ''} "${el.textContent.replace(/\s+/g, ' ').trim().slice(0, 40)}"`);
            }
        }
        return { bad: out, total: document.querySelectorAll(sel).length };
    });
    if (b4.bad.length) failed = true;
    console.log(`  [${scheme}] A-1 banner lines: ${b4.total} measured, ${b4.bad.length} truncated -> ${b4.bad.length ? 'FAIL' : 'PASS'}`);
    for (const b of b4.bad) console.log(`      ${b}`);

    // ── measurement 5: A-2 — a Restore control must name what it replaces ──
    const b5 = await page.evaluate(() => {
        const out = [];
        let total = 0;
        for (const el of document.querySelectorAll('.cf-btn, .cg-all, .cfm-go')) {
            const t = el.textContent.replace(/\s+/g, ' ').trim();
            if (!/^Restore/i.test(t)) continue;
            total++;
            if (!/replac/i.test(t)) out.push(`${el.closest('.plate')?.id || '?'}: "${t}"`);
        }
        return { bad: out, total };
    });
    if (b5.bad.length) failed = true;
    console.log(`  [${scheme}] A-2 restore names the loss: ${b5.total} Restore controls, ${b5.bad.length} silent about what they replace -> ${b5.bad.length ? 'FAIL' : 'PASS'}`);
    for (const b of b5.bad) console.log(`      ${b}`);

    await ctx.close();
}
await browser.close();

if (failed) {
    console.error('\nself-verification FAILED — see FAIL lines above');
    process.exit(1);
}
console.log('\nself-verification PASS — 16 plates x 2 schemes, no overflow, all targets >=44px,\n  every banner carries both figures, no banner line truncated, every Restore names the loss');
