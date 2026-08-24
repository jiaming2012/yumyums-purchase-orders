// LEG 5 — TOKEN EXPIRY ACROSS AN OFFLINE PERIOD.
//
// Neither W1 nor W2 touched this. The feasibility note lists it as an open
// question in one sentence: "A token minted with a 1 h TTL and a truck that is
// offline longer than that."
//
// The question is not "does an expired token get rejected" — of course it does,
// W1 proved `exp` is enforced. The question is what the CLIENT does about it,
// and whether the state is RECOVERABLE:
//
//   * Does replication die permanently, or retry?
//   * Are the local writes made while offline still there afterwards?
//   * Can a fresh token be injected WITHOUT tearing down the RxDB database and
//     losing the queued writes — i.e. is recovery a token swap, or a re-login?
//
// A real TTL is elapsed here, not mocked. A mocked clock would prove something
// about the mock.
//
// ⚠ AND THE OFFLINE WAIT HAS TO CLEAR A MEASURED 30-SECOND CLIFF.
//   The first version of this leg used TTL 20 s and waited 28 s, and the write
//   PUSHED SUCCESSFULLY. That was not RxDB being clever; PostgREST v12.2.12
//   accepts a token that is past `exp` for roughly 30 more seconds. Measured
//   here on 2026-07-26 against W1's stack, each probe a FRESH never-before-seen
//   token so it is not a JWT-cache artefact:
//
//       exp -5 s -> 200    exp -29 s -> 401 PGRST301 "JWT expired"
//       exp -15 s -> 200   exp -30 s -> 401
//       exp -25 s -> 200   exp -31 s / -35 s / -45 s / -60 s -> 401
//
//   So `exp` IS enforced (W1's finding stands) but with ~30 s of clock-skew
//   leeway. Any HQ token-refresh design that assumes `exp` is a hard edge is
//   wrong by half a minute, and any TEST that waits less than ~30 s past `exp`
//   is measuring the leeway rather than the expiry.

const { test, expect } = require('@playwright/test');
const { RUN, mintToken, rowsById } = require('./spike-support');

const PAGE = '/.night-crew/qa/spike-supabase/browser/spike.html';
const DB = `spiketok${RUN}`.toLowerCase().replace(/[^a-z0-9]/g, '');
const TTL_SECONDS = 20;
// TTL + this must clear the measured ~30 s PostgREST leeway with margin.
const OFFLINE_WAIT_SECONDS = 65;

test.describe('leg 5 — a 1 h token and a truck offline longer than that', () => {
    test('expiry while offline is recoverable by swapping the token, with no data loss', async ({ page, context }) => {
        await page.goto(PAGE);
        await page.waitForFunction(() => window.SPIKE_READY === true, null, { timeout: 30000 });
        await page.evaluate(t => { window.SPIKE.tag = 'tok'; }, 'tok');

        const shortToken = mintToken('user-alice', { ttl: `${TTL_SECONDS}s` });
        const beforeId = `tok-${RUN}-before`;
        const offlineId = `tok-${RUN}-offline`;

        await page.evaluate(n => window.SPIKE.openDb(n), DB);
        await page.evaluate(async ({ n, i }) => {
            await window.SPIKE.insert(n, { id: i, owner_id: 'user-alice', body: 'written while the token was valid' });
        }, { n: DB, i: beforeId });

        await page.evaluate(({ n, t }) => window.SPIKE.startRep('tok', {
            dbName: n, token: t, base: '', identifier: `tok-${n}`,
            waitForLeadership: false, live: true
        }), { n: DB, t: shortToken });

        expect(await page.evaluate(() => window.SPIKE.awaitInitial('tok', 30000))).toBe('ok');
        expect(await page.evaluate(() => window.SPIKE.awaitInSync('tok', 30000))).toBe('in-sync');

        // Sanity: the first write really did land while the token was good.
        {
            const { rows } = await rowsById(shortToken, [beforeId]);
            expect(rows.length, 'the pre-expiry write must be in Postgres').toBe(1);
        }

        // ---- go offline, write locally, and let the token die out there ------
        await context.setOffline(true);
        await page.evaluate(async ({ n, i }) => {
            await window.SPIKE.insert(n, { id: i, owner_id: 'user-alice', body: 'written OFFLINE, after the token expired' });
        }, { n: DB, i: offlineId });

        // Elapse the real TTL, plus the measured PostgREST skew leeway, plus
        // margin. Anything shorter and this leg silently tests the leeway.
        await new Promise(r => setTimeout(r, OFFLINE_WAIT_SECONDS * 1000));

        // The local write must be intact while offline. If RxDB dropped it here
        // the rest of the leg would be moot.
        //
        // Asserted BY ID, not by a total count: replication has already pulled
        // every pre-existing user-alice row from the fixture table into the
        // local store, so the total is whatever the shared spike table happens
        // to hold. A count assertion here would be a test of the fixture's
        // history, not of RxDB's durability.
        expect(await page.evaluate(({ n, i }) => window.SPIKE.get(n, i), { n: DB, i: offlineId }),
            'the offline write must survive in Dexie while the network is gone').not.toBeNull();

        // ---- come back online with a token that is now EXPIRED ---------------
        await context.setOffline(false);
        await page.evaluate(() => window.SPIKE.repReSync('tok'));
        await new Promise(r => setTimeout(r, 8000));

        const expiredState = await page.evaluate(() => window.SPIKE.repState('tok'));
        // eslint-disable-next-line no-console
        console.log(`[leg5] replication state with an EXPIRED token: ${JSON.stringify(expiredState)}`);
        expect(expiredState.errorCount,
            'coming back online with an expired token must surface an error, not sync silently').toBeGreaterThan(0);
        // WHICH error matters. 'the network is gone' and 'your token expired'
        // demand different responses from the app, and a leg that accepted
        // either would not have caught the 30 s leeway above.
        expect(expiredState.kinds,
            `expected a jwt-expired classification; got ${JSON.stringify(expiredState)}`).toContain('jwt-expired');

        // The offline write must still be queued, not discarded by the failure.
        expect(await page.evaluate(({ n, i }) => window.SPIKE.get(n, i), { n: DB, i: offlineId }),
            'an auth failure must not discard the queued local write').not.toBeNull();

        // It must NOT have reached Postgres — that is what makes the recovery
        // below a real recovery and not a re-observation of an earlier success.
        {
            const probe = mintToken('user-alice');
            const { rows } = await rowsById(probe, [offlineId]);
            expect(rows.length, 'the offline write must NOT be in Postgres yet').toBe(0);
        }

        // ---- RECOVERY: swap the token, do not restart replication ------------
        // supabase-js calls the `accessToken` callback on EVERY request
        // (SupabaseClient wires fetchWithAuth to _getAccessToken), so a fresh
        // token can be handed in without tearing the RxDB database down — which
        // is what makes the queued offline write survivable.
        const freshToken = mintToken('user-alice', { ttl: '1h' });
        await page.evaluate(t => window.SPIKE.setToken('current', t), freshToken);
        await page.evaluate(() => window.SPIKE.repReSync('tok'));

        const sync = await page.evaluate(() => window.SPIKE.awaitInSync('tok', 45000));
        // eslint-disable-next-line no-console
        console.log(`[leg5] awaitInSync after token swap: ${sync}`);

        // Give the push a moment even if awaitInSync resolved on the pull side.
        await new Promise(r => setTimeout(r, 5000));

        const { rows } = await rowsById(freshToken, [offlineId]);
        // eslint-disable-next-line no-console
        console.log(`[leg5] offline-written row in Postgres after recovery: ${JSON.stringify(rows)}`);
        expect(rows.length, 'the write made offline behind an expired token must reach Postgres after a token swap').toBe(1);
        expect(rows[0].body).toContain('written OFFLINE');
    });
});
