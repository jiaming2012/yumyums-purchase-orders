// LEG 1 — DELIVERY. Does the committed vendored bundle actually load in a real
// browser, from local files only, with no build step and no CDN?
//
// This is the leg that decides whether candidate 3 from the slate's planner
// decision is real. If it fails, PARK trigger (i) is live.

const { test, expect } = require('@playwright/test');

const PAGE = '/.night-crew/qa/spike-supabase/browser/spike.html';

test.describe('leg 1 — vendored bundle delivery', () => {
    test('the bundle loads via a plain <script type="module"> from a local path', async ({ page }) => {
        const consoleErrors = [];
        page.on('console', m => { if (m.type() === 'error') consoleErrors.push(m.text()); });
        page.on('pageerror', e => consoleErrors.push('pageerror: ' + String(e)));

        // Every request the page makes, so "no CDN" is a MEASUREMENT and not an
        // assertion about source code. A vendored bundle whose runtime quietly
        // fetches from unpkg would defeat the whole reason for choosing it.
        const hosts = new Set();
        page.on('request', r => { try { hosts.add(new URL(r.url()).host); } catch (e) { /* */ } });

        await page.goto(PAGE);
        await page.waitForFunction(() => window.SPIKE_READY === true, null, { timeout: 30000 });

        expect(consoleErrors, 'no console errors while importing the bundle').toEqual([]);

        const build = await page.evaluate(() => window.SPIKE.VENDOR_BUILD);
        expect(build.rxdb).toBe('17.4.0');
        expect(build.supabaseJs).toBe('2.109.0');
        expect(build.esbuild).toBe('0.28.1');

        // RXDB_VERSION comes from inside rxdb itself, VENDOR_BUILD.rxdb from the
        // lockfile at generate time. Cross-checking them is FORK 4's "fails
        // loudly on upgrade" tripwire: regenerating against a different version
        // without updating this spec turns the next run red.
        const rxdbVersion = await page.evaluate(() => window.SPIKE.RXDB_VERSION);
        expect(rxdbVersion).toBe(build.rxdb);

        const external = [...hosts].filter(h => !h.startsWith('127.0.0.1') && !h.startsWith('localhost'));
        expect(external, 'the page reached ZERO external hosts — no CDN in the offline data path').toEqual([]);
    });

    test('the bundle exports every symbol a real client would need', async ({ page }) => {
        await page.goto(PAGE);
        await page.waitForFunction(() => window.SPIKE_READY === true, null, { timeout: 30000 });
        const present = await page.evaluate(() => ({
            createRxDatabase: typeof window.SPIKE.openDb,
            dexie: typeof window.SPIKE.openDb,
            client: typeof window.SPIKE.client,
            startRep: typeof window.SPIKE.startRep
        }));
        expect(present).toEqual({
            createRxDatabase: 'function', dexie: 'function',
            client: 'function', startRep: 'function'
        });
    });
});
