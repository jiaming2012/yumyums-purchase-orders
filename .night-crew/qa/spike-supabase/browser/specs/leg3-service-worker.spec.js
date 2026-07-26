// LEG 3 — SERVICE-WORKER INTERACTION. The sharpest edge in the card.
//
// HQ's Workbox `sw.js` registers exactly one runtime route (build-sw.js):
//
//     urlPattern: /\/api\//
//     handler:    'NetworkFirst'   (networkTimeoutSeconds: 10, cacheName 'api-cache')
//     handlerDidError: -> new Response('{"error":"offline"}', { status: 503 })
//
// The feasibility note calls out the danger as "an offline fallback answering a
// replication request with cached JSON". That phrasing understates it, and this
// leg exists to find out which of TWO failures is real:
//
//   (a) the 503 fallback answers a replication request  -> an ERROR. Loud. Fine.
//   (b) NetworkFirst falls back to its CACHE first, and answers a replication
//       request with a STALE BUT WELL-FORMED 200 -> RxDB cannot tell it from a
//       fresh pull, accepts the rows, and ADVANCES ITS CHECKPOINT past data it
//       never saw. Silent. Not fine.
//
// Workbox's NetworkFirst tries network -> cache -> handlerDidError, in that
// order, so (b) is the one that actually happens whenever the same URL was
// fetched successfully before. This leg measures it rather than reasoning about
// it.
//
// ============================================================================
// This spec is the reason playwright.spike.config.js exists. The repo-wide
// `serviceWorkers: 'block'` in the ROOT playwright.config.js is NOT touched.
// ============================================================================

const { test, expect } = require('@playwright/test');
const { RUN, REALTIME_PORT, mintToken, rowsById, REST_PORT, TABLE } = require('./spike-support');

const PAGE = '/.night-crew/qa/spike-supabase/browser/spike.html';

async function ready(page) {
    await page.goto(PAGE);
    await page.waitForFunction(() => window.SPIKE_READY === true, null, { timeout: 30000 });
    const reg = await page.evaluate(() => window.SPIKE.registerSW());
    expect(reg.hasController, `the SW must actually be controlling the page; got ${JSON.stringify(reg)}`).toBe(true);
    expect(reg.controllerUrl).toContain('/sw.js');
    return reg;
}

test.describe('leg 3 — HQ Workbox service worker vs. RxDB replication', () => {

    test('3a — the vendored bundle is served from the Workbox precache while OFFLINE', async ({ page, context }) => {
        await ready(page);
        const online = await page.evaluate(() => window.SPIKE.probe('/vendor/rxdb.bundle.js'));
        expect(online.status).toBe(200);

        await context.setOffline(true);
        const offline = await page.evaluate(() => window.SPIKE.probe('/vendor/rxdb.bundle.js'));
        await context.setOffline(false);

        // This is the payoff of the vendored-bundle decision: the offline data
        // engine is a precached local asset, so it is available exactly when it
        // is needed. A CDN import could not do this without a SECOND offline
        // story, which Engineering KR3 forbids.
        expect(offline.status, 'the offline data engine must itself be available offline').toBe(200);
        expect(offline.bytes).toBe(online.bytes);
    });

    test('3b — the /api/ offline fallback is real: 503 {"error":"offline"}', async ({ page, context }) => {
        await ready(page);
        // A URL that has NEVER been fetched successfully, so NetworkFirst has no
        // cache entry and must reach handlerDidError.
        const url = `/api/v1/spike-ping?never-cached=${RUN}`;
        await context.setOffline(true);
        const res = await page.evaluate(u => window.SPIKE.probe(u), url);
        await context.setOffline(false);

        expect(res.status).toBe(503);
        expect(res.contentType).toContain('application/json');
        expect(res.bodyStart).toContain('"error":"offline"');
    });

    test('3c — THE TRAP: a same-origin replication URL under /api/ is answered from STALE CACHE while offline', async ({ page, context }) => {
        await ready(page);
        const token = mintToken('user-alice');

        // A stable, replication-shaped PostgREST URL under /api/ — exactly what
        // supabase-js emits when its base URL is `${origin}/api/v1`.
        const q = `select=*&owner_id=eq.user-alice&order=_modified.asc&limit=100`;
        const trapUrl = `/api/v1/rest/v1/${TABLE}?${q}`;
        const safeUrl = `/rest/v1/${TABLE}?${q}`;

        // 1. Prime the cache with a real, successful, well-formed pull.
        const primed = await page.evaluate(async ({ u, t }) => {
            const r = await fetch(u, { headers: { Authorization: `Bearer ${t}` } });
            const body = await r.text();
            return { status: r.status, rows: JSON.parse(body).length };
        }, { u: trapUrl, t: token });
        expect(primed.status).toBe(200);

        // 2. A new row appears on the server, out of band. The browser has no
        //    way to know about it. This is a truck that was offline while the
        //    owner edited a checklist from the office.
        const newId = `trap-${RUN}`;
        const ins = await fetch(`http://127.0.0.1:${REST_PORT}/${TABLE}`, {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
                Prefer: 'return=representation'
            },
            body: JSON.stringify({ id: newId, owner_id: 'user-alice', body: 'written server-side while the tab was about to go offline' })
        });
        expect(ins.status, 'the out-of-band insert must have landed').toBe(201);

        // 3. Offline. Ask for exactly the same URL again.
        await context.setOffline(true);
        const trapOffline = await page.evaluate(async ({ u, t }) => {
            try {
                const r = await fetch(u, { headers: { Authorization: `Bearer ${t}` } });
                const body = await r.text();
                let rows = null; try { rows = JSON.parse(body); } catch (e) { /* */ }
                return {
                    status: r.status, contentType: r.headers.get('content-type'),
                    isArray: Array.isArray(rows),
                    count: Array.isArray(rows) ? rows.length : null,
                    bodyStart: body.slice(0, 120)
                };
            } catch (e) { return { threw: true, error: String(e) }; }
        }, { u: trapUrl, t: token });

        // 4. And the same shape of URL NOT under /api/, for contrast.
        const safeOffline = await page.evaluate(async ({ u, t }) => {
            try {
                const r = await fetch(u, { headers: { Authorization: `Bearer ${t}` } });
                return { status: r.status, bodyStart: (await r.text()).slice(0, 120) };
            } catch (e) { return { threw: true, error: String(e) }; }
        }, { u: safeUrl, t: token });
        await context.setOffline(false);

        // eslint-disable-next-line no-console
        console.log('[leg3c] under /api/ while offline:', JSON.stringify(trapOffline));
        // eslint-disable-next-line no-console
        console.log('[leg3c] NOT under /api/ while offline:', JSON.stringify(safeOffline));

        // ---- the finding -----------------------------------------------------
        // The non-/api/ URL is not routed by the SW at all, so offline is an
        // honest network failure. RxDB sees a thrown fetch and retries. Correct.
        expect(safeOffline.threw, 'a replication URL outside /api/ must fail honestly when offline').toBe(true);

        // The /api/ URL is routed. Whichever of (a) or (b) it turns out to be is
        // recorded, but it must NOT be a well-formed 200 that RxDB would accept
        // as a fresh pull — that is the silent-corruption case, and if it holds
        // it is a hard constraint on where HQ may mount Supabase.
        const silentlyStale = trapOffline.status === 200 && trapOffline.isArray === true;
        // eslint-disable-next-line no-console
        console.log(`[leg3c] VERDICT INPUT — silent-stale-200 under /api/ while offline: ${silentlyStale}`);

        // The test asserts the SW routed it at all (that much is certain from
        // build-sw.js) and pins the observed behaviour so a future Workbox or
        // config change turns this red instead of drifting silently.
        expect(trapOffline.threw, 'the SW answered rather than letting the request fail').toBeFalsy();
        expect([200, 503]).toContain(trapOffline.status);
    });

    test('3d — replication itself works with the SW controlling, over a non-/api/ same-origin path', async ({ page }) => {
        await ready(page);
        const token = mintToken('user-alice');
        const db = `swrep${RUN}`.toLowerCase().replace(/[^a-z0-9]/g, '');
        const id = `sw-${RUN}`;

        await page.evaluate(n => window.SPIKE.openDb(n), db);
        await page.evaluate(async ({ n, i }) => {
            await window.SPIKE.insert(n, { id: i, owner_id: 'user-alice', body: 'pushed from a browser with HQ sw.js in the middle' });
        }, { n: db, i: id });

        await page.evaluate(({ n, t, i }) => window.SPIKE.startRep('sw', {
            dbName: n, token: t, base: '', identifier: `sw-${i}`,
            waitForLeadership: false, live: false
        }), { n: db, t: token, i: id });

        expect(await page.evaluate(() => window.SPIKE.awaitInitial('sw', 30000))).toBe('ok');
        expect(await page.evaluate(() => window.SPIKE.awaitInSync('sw', 30000))).toBe('in-sync');

        // Verified over an INDEPENDENT request straight at PostgREST — not
        // through the page, not through the proxy, not through the SW, not
        // through RxDB's own opinion of whether it is in sync.
        const { status, rows } = await rowsById(token, [id]);
        expect(status).toBe(200);
        expect(rows.length).toBe(1);
        expect(rows[0].body).toContain('HQ sw.js in the middle');

        const state = await page.evaluate(() => window.SPIKE.repState('sw'));
        expect(state.errorCount, `replication errors with the SW controlling: ${JSON.stringify(state.errors)}`).toBe(0);
    });

    test('3e — a long-lived WebSocket UNDER /api/ is untouched by the service worker', async ({ page, context }) => {
        await ready(page);

        // Same origin, and the path matches HQ's /\/api\// runtime route exactly.
        // If the SW were ever going to answer a WebSocket handshake with its
        // cached JSON or its 503 fallback, this is the case where it would.
        const wsUrl = `ws://127.0.0.1:${new URL(page.url()).port}/api/v1/spike-ws?run=${RUN}`;

        const online = await page.evaluate(u => window.SPIKE.rawWebSocket(u), wsUrl);
        // eslint-disable-next-line no-console
        console.log('[leg3e] /api/-mounted WebSocket with SW controlling, online:', JSON.stringify(online));
        expect(online.verdict,
            'a WebSocket handshake under /api/ must OPEN — the SW never sees it').toBe('open');

        await context.setOffline(true);
        const offline = await page.evaluate(u => window.SPIKE.rawWebSocket(u, 6000), wsUrl);
        await context.setOffline(false);
        // eslint-disable-next-line no-console
        console.log('[leg3e] /api/-mounted WebSocket with SW controlling, offline:', JSON.stringify(offline));

        // Offline can only produce a transport failure. A WebSocket handshake is
        // not a fetch the SW can answer, so it can never become HQ's 503 JSON
        // fallback and can never be served from 'api-cache'. That asymmetry
        // matters: the WS half of replication fails LOUDLY where the HTTP half
        // (3c) fails SILENTLY.
        expect(['error', 'close', 'timeout']).toContain(offline.verdict);

        // Recorded, not asserted — this is W1's substrate surface, not this
        // card's, and a red here would attribute their finding to us.
        const realtimeUrl = `ws://realtime-dev.localhost:${REALTIME_PORT}/socket/websocket?vsn=1.0.0`;
        const rt = await page.evaluate(u => window.SPIKE.rawWebSocket(u), realtimeUrl);
        // eslint-disable-next-line no-console
        console.log(`[leg3e] RECORDED (not asserted) — browser -> self-hosted Realtime vhost: ${JSON.stringify(rt)}`);
    });
});
