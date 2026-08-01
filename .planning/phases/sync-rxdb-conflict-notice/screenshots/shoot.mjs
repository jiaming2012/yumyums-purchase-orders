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
//   - A-2 names the loss   -> every DESTRUCTIVE control must say what it
//                             replaces. Population is scoped by what the control
//                             DOES, not by how it is labelled.
//   - A-1 arithmetic       -> the printed figures reconciled against the rows
//                             actually drawn beneath them.
//   - collapse keeps exits -> a row with nothing to restore keeps its Dismiss.
// Each prints a PASS/FAIL line and sets a non-zero exit code on failure, so the
// check cannot quietly rot.
//
// REPAIR ROUND (overnight-20260729-2, card C1). Four of these checks selected
// their population in a way that could not catch the defect they exist for, and
// were widened. Recorded here because the failure mode is generic and will
// recur:
//   * m3 walked every .cn-banner but never asserted HOW MANY there are, so
//     DELETING a banner was invisible to it. Now pinned to EXPECTED_BANNERS.
//   * m5 selected by LABEL (`/^Restore/`), which silently excluded `Retry` and
//     `Restoring…` on plate-error — the same destructive write, on the one plate
//     where the crew member has already failed once — and, worse, EVAPORATED
//     under a rename while reporting "0 controls, 0 silent -> PASS". Now scoped
//     by behaviour (everything that is not one of the five non-destructive
//     labels) with a floor on the population.
//   * m6 (new) — presence of a still-to-review figure was checked; its VALUE was
//     not. `sed s/1 still to review/99 still to review/` passed green.
//   * m7 (new) — collapse removed the only exit from a row that has no Restore
//     to hide. Nothing measured it.
// A check scoped to the place a fix was made is the same escape as a criterion
// scoped to the members that already pass.
//
// HARDENING ROUND (same run, third gate). The repair round pinned the
// population of m3, m5 and m7 and left the other two population-walking checks
// unpinned, which is the same defect one round later:
//   * m2 (tap targets) walked the controls that EXIST. Deleting every
//     `.cf-done-undo` Undo — the only escape from a mis-tapped Restore, and the
//     control row 18 exists for — printed "58 measured, 0 under 44px -> PASS",
//     exit 0. Now floored at EXPECTED_TAP_TARGETS.
//   * m4 (banner lines) had it too: deleting a `.cn-banner-sub` cause line slid
//     24 -> 23 and still passed. Now floored at EXPECTED_BANNER_LINES.
// Six of the seven measurements now pin their population; m1 (page overflow) is
// not a population walk and has nothing to pin.
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

// Expected populations. These are FLOORS/PINS, not decoration: without them a
// deletion or a rename makes a check pass vacuously while reporting success.
const EXPECTED_BANNERS = 8;      // .cn-banner elements in the whole file
const EXPECTED_DESTRUCTIVE_ROW = 12; // .cf-btn + .cg-all controls that WRITE
const EXPECTED_DESTRUCTIVE = 13;     // ... plus the confirm's own .cfm-go
const EXPECTED_UNREC_ROWS = 6;   // .cf.unrec rows, each of which must keep a Dismiss
const EXPECTED_TAP_TARGETS = 62; // interactive elements matching TAP_TARGETS
const EXPECTED_BANNER_LINES = 24; // .cn-banner-hd/-open/-unid/-sub lines measured

// The five labels that do NOT overwrite anything anyone else saved. Everything
// else inside '.cf-btn, .cg-all, .cfm-go' writes over a server value and is
// therefore in scope for A-2, whatever it happens to be called.
const NON_DESTRUCTIVE = ['Keep theirs', 'Dismiss', 'Open checklist', 'Copy value', 'Cancel'];

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
    // Same hole every other measurement had: this walks the controls that
    // EXIST, so DELETING one passes green while the printed count slides down.
    // Deleting all four `.cf-done-undo` Undo controls — the only escape from a
    // mis-tapped Restore — reported "58 measured, 0 under 44px -> PASS".
    const n_ok = n >= EXPECTED_TAP_TARGETS;
    if (bad.length || !n_ok) failed = true;
    console.log(`  [${scheme}] tap targets: ${n} measured (expected >=${EXPECTED_TAP_TARGETS}), ${bad.length} under 44px -> ${(bad.length || !n_ok) ? 'FAIL' : 'PASS'}`);
    if (!n_ok) console.log(`      POPULATION below floor: ${n} controls, need ${EXPECTED_TAP_TARGETS} — an interactive control was deleted or renamed out of the selector`);
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
    // The per-banner shape above iterates the banners that EXIST, so deleting
    // one is invisible to it — it would report "7 banners, 0 carrying only one
    // -> PASS". Pin the population.
    const b3_count_ok = b3.total === EXPECTED_BANNERS;
    if (b3.bad.length || !b3_count_ok) failed = true;
    console.log(`  [${scheme}] A-1 two figures: ${b3.total} banners (expected ${EXPECTED_BANNERS}), ${b3.bad.length} carrying only one -> ${(b3.bad.length || !b3_count_ok) ? 'FAIL' : 'PASS'}`);
    if (!b3_count_ok) console.log(`      banner COUNT is ${b3.total}, expected ${EXPECTED_BANNERS} — a banner was added or deleted`);
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
    // Same hole again: iterating the lines that EXIST means DELETING one is
    // invisible — dropping a .cn-banner-sub cause line slides 24 -> 23 and
    // still reports PASS. Pin the population.
    const b4_count_ok = b4.total >= EXPECTED_BANNER_LINES;
    if (b4.bad.length || !b4_count_ok) failed = true;
    console.log(`  [${scheme}] A-1 banner lines: ${b4.total} measured (expected >=${EXPECTED_BANNER_LINES}), ${b4.bad.length} truncated -> ${(b4.bad.length || !b4_count_ok) ? 'FAIL' : 'PASS'}`);
    if (!b4_count_ok) console.log(`      POPULATION below floor: ${b4.total} lines, need ${EXPECTED_BANNER_LINES} — a banner line was deleted`);
    for (const b of b4.bad) console.log(`      ${b}`);

    // ── measurement 5: A-2 — a DESTRUCTIVE control must name what it replaces ─
    // Population is scoped by BEHAVIOUR, not by label. The r1-era version tested
    // `/^Restore/` on the label, which excluded `Retry` and `Restoring…` on
    // plate-error — the same destructive write as `Restore mine`, on the one
    // plate where the crew member has already failed once — and evaporated
    // entirely under a rename, reporting "0 controls, 0 silent -> PASS".
    const b5 = await page.evaluate((allow) => {
        const out = [];
        const rows = [];   // .cf-btn + .cg-all
        const confirms = []; // .cfm-go
        for (const el of document.querySelectorAll('.cf-btn, .cg-all, .cfm-go')) {
            // The visible primary label: the first element child when the control
            // has a sub-label, otherwise the whole control.
            const head = el.firstElementChild || el;
            const label = head.textContent.replace(/\s+/g, ' ').trim();
            const full = el.textContent.replace(/\s+/g, ' ').trim();
            if (allow.includes(label)) continue;   // writes nothing of anyone else's
            const sub = el.querySelector('.cf-btn-s, .cg-all-s');
            // The claim must live in the SUB-LABEL where one exists — stripping
            // sub-labels is the mutation this check has to survive. Where there
            // is none (the confirm's own commit button) the whole label carries it.
            const hay = sub ? sub.textContent.replace(/\s+/g, ' ').trim() : full;
            (el.classList.contains('cfm-go') ? confirms : rows).push(label);
            if (!/replac/i.test(hay)) {
                out.push(`${el.closest('.plate')?.id || '?'}: "${label}" — nothing about what it replaces (${sub ? `sub-label "${hay}"` : 'no sub-label'})`);
            }
        }
        return { bad: out, rows: rows.length, confirms: confirms.length };
    }, NON_DESTRUCTIVE);
    const b5_total = b5.rows + b5.confirms;
    // Floors, so a rename or a deletion REDS instead of passing on an empty set.
    const b5_pop_ok = b5.rows >= EXPECTED_DESTRUCTIVE_ROW && b5_total >= EXPECTED_DESTRUCTIVE;
    if (b5.bad.length || !b5_pop_ok) failed = true;
    console.log(`  [${scheme}] A-2 destructive names the loss: ${b5_total} destructive controls (${b5.rows} row/batch + ${b5.confirms} confirm; expected >=${EXPECTED_DESTRUCTIVE_ROW} + >=${EXPECTED_DESTRUCTIVE - EXPECTED_DESTRUCTIVE_ROW}), ${b5.bad.length} silent about what they replace -> ${(b5.bad.length || !b5_pop_ok) ? 'FAIL' : 'PASS'}`);
    if (!b5_pop_ok) console.log(`      POPULATION below floor: ${b5.rows} row/batch (need ${EXPECTED_DESTRUCTIVE_ROW}), ${b5_total} total (need ${EXPECTED_DESTRUCTIVE}) — a destructive control was renamed into the non-destructive allowlist, or deleted`);
    for (const b of b5.bad) console.log(`      ${b}`);

    // ── measurement 6: A-1 — the printed figures RECONCILE with the rows ─────
    // Measurement 3 checks a still-to-review figure is PRESENT. That is not the
    // same as it being RIGHT: `sed s/1 still to review/99 still to review/`
    // passed measurement 3 green. This one does the arithmetic.
    //
    // Open decision (i) — whether a removed-field row counts in the chip base
    // (Reading A) or moves to +N (Reading B) — is NOT settled by this card, so a
    // removed-field row is treated as assignable to either bucket and the plate
    // must balance under EXACTLY ONE of the two readings. A plate that balances
    // under neither is wrong; the check names which reading each plate uses
    // without preferring one.
    const b6 = await page.evaluate(() => {
        const norm = t => (t || '').replace(/\s+/g, ' ').trim();
        const num = (t, re) => { const m = norm(t).match(re); return m ? parseInt(m[1], 10) : null; };
        const out = [];
        const readings = [];
        let checked = 0;

        const classify = (cf) => {
            // bucket: 'answer' | 'extra' | 'ambiguous'
            let bucket = 'answer';
            // A removed-field row. TWO renderings since AMENDMENT A-3 (decision
            // 95): `.cf-q-struck` is the question's own frozen label, struck
            // through and read-only, and `.cf-q-gone` is the raw-field-id
            // FALLBACK for a snapshot that carries no label — which, because
            // nothing validates `template_snapshot` (R-C), is also every
            // malformed one. Both are the same KIND of row and must classify
            // the same way; keying only on `.cf-q-gone` would have silently
            // reclassified every redrawn row as an ordinary answer, and
            // measurement 6 would have gone on printing PASS against the wrong
            // arithmetic.
            if (cf.querySelector('.cf-q-struck, .cf-q-gone')) bucket = 'ambiguous';
            else if (cf.querySelector('.cf-v.none')) bucket = 'extra';      // nothing to recover
            // state: 'handled' | 'open'   (counting rule 6)
            const handled = !!(cf.querySelector('.cf-done') || cf.querySelector('.cf-kept'));
            return { bucket, state: handled ? 'handled' : 'open' };
        };

        for (const plate of document.querySelectorAll('.plate')) {
            const banner = plate.querySelector('.cn-banner');
            const groups = [...plate.querySelectorAll('.cg')];
            if (!banner && !groups.length) continue;
            checked++;
            const pid = plate.id;

            const rowsOf = g => [...g.querySelectorAll('.cf')].map(classify);
            const gs = groups.map(g => ({
                chip: norm(g.querySelector('.cg-count')?.textContent),
                rows: rowsOf(g)
            }));

            const hd = banner ? num(banner.querySelector('.cn-banner-hd')?.textContent, /^(\d+)/) : null;
            const openEl = banner ? banner.querySelector('.cn-banner-open') : null;
            const openTxt = norm(openEl?.textContent);
            const unidEl = banner ? banner.querySelector('.cn-banner-unid') : null;
            const unid = unidEl ? num(unidEl.textContent, /\+\s*(\d+)/) : null;

            const tryReading = (amb) => {   // amb: 'answer' | 'extra'
                const errs = [];
                const bucketOf = r => (r.bucket === 'ambiguous' ? amb : r.bucket);
                const all = gs.flatMap(g => g.rows);
                const answers = all.filter(r => bucketOf(r) === 'answer');
                const extras = all.filter(r => bucketOf(r) === 'extra');
                const open = answers.filter(r => r.state === 'open').length;
                const handled = answers.filter(r => r.state === 'handled').length;

                if (banner) {
                    if (hd === null) errs.push('headline carries no number');
                    else if (hd !== answers.length) errs.push(`headline ${hd} != ${answers.length} answer rows drawn`);

                    const all_rev = norm(openTxt).match(/All (\d+) reviewed/);
                    const still = num(openTxt, /(\d+) still to review/);
                    const hnd = num(openTxt, /(\d+) handled/);
                    if (all_rev) {
                        if (open !== 0) errs.push(`"All N reviewed" but ${open} row(s) are untouched/failed/in-flight`);
                        if (parseInt(all_rev[1], 10) !== answers.length) errs.push(`"All ${all_rev[1]} reviewed" != ${answers.length} answer rows`);
                    } else if (still === null) {
                        errs.push(`still-to-review line carries no figure: "${openTxt}"`);
                    } else {
                        if (still !== open) errs.push(`"${still} still to review" != ${open} untouched/failed/in-flight answer rows`);
                        if (hnd !== null && hnd !== handled) errs.push(`"${hnd} handled" != ${handled} restored/kept answer rows`);
                        if (hnd !== null && still + hnd !== hd) errs.push(`${still} still + ${hnd} handled != headline ${hd}`);
                    }

                    if (unid === null && extras.length) errs.push(`${extras.length} unidentifiable row(s) drawn but no "+N" banner line`);
                    if (unid !== null && unid !== extras.length) errs.push(`"+${unid}" banner line != ${extras.length} unidentifiable rows`);
                }

                for (const g of gs) {
                    const base = num(g.chip, /^(\d+)/);
                    const plus = num(g.chip, /\+\s*(\d+)/) || 0;
                    const gAns = g.rows.filter(r => bucketOf(r) === 'answer').length;
                    const gExt = g.rows.filter(r => bucketOf(r) === 'extra').length;
                    if (base === null) { errs.push(`chip "${g.chip}" carries no number`); continue; }
                    if (base !== gAns) errs.push(`chip base ${base} != ${gAns} answer rows in that group ("${g.chip}")`);
                    if (plus !== gExt) errs.push(`chip +${plus} != ${gExt} unidentifiable rows in that group ("${g.chip}")`);
                    if (base + plus !== g.rows.length) errs.push(`chip ${base}+${plus} != ${g.rows.length} rows drawn ("${g.chip}")`);
                }
                return errs;
            };

            const hasAmb = gs.some(g => g.rows.some(r => r.bucket === 'ambiguous'));
            const eA = tryReading('answer');
            const eB = hasAmb ? tryReading('extra') : null;
            if (!hasAmb) {
                if (eA.length) out.push(`${pid}: ${eA.join('; ')}`);
                else readings.push(`${pid}=exact`);
            } else if (!eA.length && eB.length) readings.push(`${pid}=Reading A`);
            else if (eA.length && !eB.length) readings.push(`${pid}=Reading B`);
            else if (!eA.length && !eB.length) readings.push(`${pid}=either (ambiguous both ways)`);
            else out.push(`${pid}: balances under NEITHER reading — A: ${eA.join('; ')} | B: ${eB.join('; ')}`);
        }
        return { bad: out, checked, readings };
    });
    if (b6.bad.length) failed = true;
    console.log(`  [${scheme}] A-1 arithmetic: ${b6.checked} counting plates reconciled, ${b6.bad.length} disagreeing -> ${b6.bad.length ? 'FAIL' : 'PASS'}`);
    if (scheme === 'light') console.log(`      readings: ${b6.readings.join(', ')}`);
    for (const b of b6.bad) console.log(`      ${b}`);

    // ── measurement 7: collapse never removes a row's ONLY exit ──────────────
    // A row with no discarded value has no Restore/Keep pair for collapse to
    // hide, and Dismiss is the only way it ever leaves the sheet (counting rule
    // 3). Collapse must therefore leave its actions alone. Nothing measured this
    // before, and plate-a1-banner drew two such rows with no actions at all.
    const b7 = await page.evaluate(() => {
        const out = [];
        const rows = [...document.querySelectorAll('.cf.unrec')];
        for (const cf of rows) {
            const acts = [...cf.querySelectorAll('.cf-acts .cf-btn')].map(b => b.textContent.replace(/\s+/g, ' ').trim());
            const plate = cf.closest('.plate')?.id || '?';
            const q = (cf.querySelector('.cf-q, .cf-q-gone')?.textContent || '').trim().slice(0, 34);
            if (!acts.includes('Dismiss')) out.push(`${plate}: "${q}" has no Dismiss — actions drawn: [${acts.join(', ')}]`);
        }
        return { bad: out, total: rows.length };
    });
    const b7_pop_ok = b7.total >= EXPECTED_UNREC_ROWS;
    if (b7.bad.length || !b7_pop_ok) failed = true;
    console.log(`  [${scheme}] no-restore rows keep an exit: ${b7.total} .cf.unrec rows (expected >=${EXPECTED_UNREC_ROWS}), ${b7.bad.length} with no Dismiss -> ${(b7.bad.length || !b7_pop_ok) ? 'FAIL' : 'PASS'}`);
    if (!b7_pop_ok) console.log(`      POPULATION below floor: ${b7.total} rows, need ${EXPECTED_UNREC_ROWS} — an unrecoverable row was deleted`);
    for (const b of b7.bad) console.log(`      ${b}`);

    await ctx.close();
}
await browser.close();

if (failed) {
    console.error('\nself-verification FAILED — see FAIL lines above');
    process.exit(1);
}
console.log('\nself-verification PASS — 16 plates x 2 schemes, no overflow, all targets >=44px,\n  ' + EXPECTED_BANNERS + ' banners each carrying both figures, no banner line truncated,\n  every destructive control names the loss, every printed figure reconciles with the\n  rows drawn beneath it, and no row lost its only exit to collapse');
