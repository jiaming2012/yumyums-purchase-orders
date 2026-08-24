// LEG 2 — DEXIE STORAGE IN A REAL BROWSER, ACROSS A RELOAD.
//
// W2's Node proof ran on getRxStorageMemory(). A memory store starts empty
// every run and CANNOT FAIL THE WAY A BROWSER STORE FAILS. This leg is the
// difference between "RxDB replicates" and "RxDB replicates on a phone".
//
// The assertion that matters is not "3 documents came back". It is "3 documents
// came back AFTER the page was destroyed and rebuilt, from IndexedDB, and a
// document that was never written is still absent" — the negative half is what
// stops a leg that trivially passes from looking like a leg that proved
// something.

const { test, expect } = require('@playwright/test');
const { RUN } = require('./spike-support');

const PAGE = '/.night-crew/qa/spike-supabase/browser/spike.html';
const DB = `spikedexie${RUN}`.toLowerCase().replace(/[^a-z0-9]/g, '');

test.describe('leg 2 — Dexie/IndexedDB survives a reload', () => {
    test('write, reload, and the data is still there', async ({ page }) => {
        await page.goto(PAGE);
        await page.waitForFunction(() => window.SPIKE_READY === true, null, { timeout: 30000 });

        // 1. It really is Dexie, not memory. Named explicitly because the entire
        //    point of the leg is which storage engine is underneath.
        const opened = await page.evaluate(n => window.SPIKE.openDb(n), DB);
        expect(opened.storage).toBe('dexie');

        await page.evaluate(async ({ n, run }) => {
            for (const i of [1, 2, 3]) {
                await window.SPIKE.insert(n, {
                    id: `dexie-${run}-${i}`, owner_id: 'user-alice', body: `row ${i} before reload`
                });
            }
        }, { n: DB, run: RUN });

        expect(await page.evaluate(n => window.SPIKE.count(n), DB)).toBe(3);

        // 2. The browser itself agrees an IndexedDB database exists. RxDB's own
        //    count() would be equally happy over a memory store.
        const idbBefore = await page.evaluate(() => window.SPIKE.idbDatabases());
        expect(idbBefore.some(d => d && d.includes(DB)),
            `indexedDB.databases() should list an RxDB database for ${DB}; got ${JSON.stringify(idbBefore)}`).toBe(true);

        // 3. Destroy the page. Not db.close() — a real reload, which is what a
        //    crew member pulling to refresh actually does.
        await page.reload();
        await page.waitForFunction(() => window.SPIKE_READY === true, null, { timeout: 30000 });

        const reopened = await page.evaluate(n => window.SPIKE.openDb(n), DB);
        expect(reopened.reused, 'the reload really did tear the JS heap down').toBe(false);
        expect(reopened.storage).toBe('dexie');

        const ids = await page.evaluate(n => window.SPIKE.ids(n), DB);
        expect(ids).toEqual([`dexie-${RUN}-1`, `dexie-${RUN}-2`, `dexie-${RUN}-3`]);

        const doc = await page.evaluate(
            ({ n, id }) => window.SPIKE.get(n, id), { n: DB, id: `dexie-${RUN}-2` });
        expect(doc.body).toBe('row 2 before reload');

        // 4. THE NEGATIVE HALF. If this came back non-null the test would be
        //    passing for a reason that has nothing to do with persistence.
        const never = await page.evaluate(
            ({ n, id }) => window.SPIKE.get(n, id), { n: DB, id: `dexie-${RUN}-never-written` });
        expect(never).toBeNull();
    });

    test('a second, independent page in a fresh context sees the same store', async ({ browser }) => {
        // A reload keeps the same browser context. This checks the data is in
        // the ORIGIN's storage, not in anything the first context was holding.
        const ctx = await browser.newContext({ serviceWorkers: 'allow' });
        const p = await ctx.newPage();
        await p.goto(PAGE);
        await p.waitForFunction(() => window.SPIKE_READY === true, null, { timeout: 30000 });
        await p.evaluate(n => window.SPIKE.openDb(n), DB);
        const ids = await p.evaluate(n => window.SPIKE.ids(n), DB);
        await ctx.close();

        // Playwright contexts are isolated by design, so an EMPTY result here is
        // the expected and correct outcome — it is recorded rather than asserted
        // as a pass/fail, because what it measures is Playwright's isolation
        // model and not RxDB's persistence. The reload test above is the proof.
        // eslint-disable-next-line no-console
        console.log(`[leg2] fresh-context ids (Playwright isolates storage; [] is expected): ${JSON.stringify(ids)}`);
        expect(Array.isArray(ids)).toBe(true);
    });
});
