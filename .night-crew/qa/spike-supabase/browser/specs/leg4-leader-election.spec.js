// LEG 4 — MULTI-TAB LEADER ELECTION.
//
// W2's Node harness set `waitForLeadership: false` on all three proofs, and
// said so plainly (README half 2, sharp edge 10): "that is a harness
// concession — a real browser client should leave it at the default." The
// default is TRUE. Nothing has ever run it that way.
//
// Two questions, and only the second one is interesting:
//   1. With two tabs open, does exactly ONE win leadership?  (expected yes)
//   2. When the leader tab GOES AWAY, does the survivor take over — or does the
//      crew member's second tab sit there, silently not replicating, until they
//      close and reopen the app? A food truck with two tabs open on one phone
//      is not an exotic scenario; it is Tuesday.

const { test, expect } = require('@playwright/test');
const { RUN, mintToken } = require('./spike-support');

const PAGE = '/.night-crew/qa/spike-supabase/browser/spike.html';
const DB = `spikelead${RUN}`.toLowerCase().replace(/[^a-z0-9]/g, '');

async function boot(ctx, tag) {
    const p = await ctx.newPage();
    await p.goto(PAGE);
    await p.waitForFunction(() => window.SPIKE_READY === true, null, { timeout: 30000 });
    await p.evaluate(t => { window.SPIKE.tag = t; }, tag);
    await p.evaluate(n => window.SPIKE.openDb(n), DB);
    await p.evaluate(n => window.SPIKE.watchLeadership(n), DB);
    return p;
}

async function waitLeader(pages, ms = 45000) {
    const t0 = Date.now();
    while (Date.now() - t0 < ms) {
        const flags = [];
        for (const p of pages) flags.push(await p.evaluate(() => window.SPIKE.leaderWon()));
        if (flags.some(Boolean)) return flags;
        await new Promise(r => setTimeout(r, 500));
    }
    const flags = [];
    for (const p of pages) flags.push(await p.evaluate(() => window.SPIKE.leaderWon()));
    return flags;
}

test.describe('leg 4 — multi-tab leader election at the BROWSER default', () => {
    test('exactly one of two tabs leads, and the survivor takes over when it closes', async ({ browser }) => {
        // One context, two pages = two tabs of one browser profile, which is
        // what shares a BroadcastChannel and an IndexedDB. Two CONTEXTS would
        // be two browser profiles and would prove nothing about tabs.
        const ctx = await browser.newContext({ serviceWorkers: 'allow' });
        const tabA = await boot(ctx, 'A');
        const tabB = await boot(ctx, 'B');

        const token = mintToken('user-alice');

        // Both tabs start replication at the BROWSER DEFAULT — waitForLeadership
        // is left undefined so replicateSupabase's own default (true) applies.
        // That is the configuration nothing has ever run.
        for (const [p, key] of [[tabA, 'A'], [tabB, 'B']]) {
            await p.evaluate(({ n, t, k }) => window.SPIKE.startRep(k, {
                dbName: n, token: t, base: '', identifier: 'leg4-shared',
                waitForLeadership: undefined, live: true
            }), { n: DB, t: token, k: key });
        }

        const flags = await waitLeader([tabA, tabB]);
        // eslint-disable-next-line no-console
        console.log(`[leg4] leadership after both tabs started: A=${flags[0]} B=${flags[1]}`);

        expect(flags.filter(Boolean).length, 'exactly one tab must lead').toBe(1);

        const leaderIdx = flags.findIndex(Boolean);
        const leader = [tabA, tabB][leaderIdx];
        const follower = [tabA, tabB][1 - leaderIdx];

        // The follower's replication must NOT have run. If it had, leader
        // election would be decorative and two tabs would double-push.
        const followerState = await follower.evaluate(k => window.SPIKE.repState(k), leaderIdx === 0 ? 'B' : 'A');
        // eslint-disable-next-line no-console
        console.log(`[leg4] follower replication state before handover: ${JSON.stringify(followerState)}`);
        expect(followerState.started, 'the follower must not have replicated while a leader existed').toBe(false);

        // ---- the interesting half: kill the leader tab ----------------------
        const t0 = Date.now();
        await leader.close();

        let tookOver = false;
        while (Date.now() - t0 < 60000) {
            tookOver = await follower.evaluate(() => window.SPIKE.leaderWon());
            if (tookOver) break;
            await new Promise(r => setTimeout(r, 500));
        }
        const handoverMs = Date.now() - t0;
        // eslint-disable-next-line no-console
        console.log(`[leg4] survivor took over: ${tookOver} after ${handoverMs} ms`);

        expect(tookOver, `the surviving tab must take over leadership; it did not within ${handoverMs} ms`).toBe(true);

        // Winning the election is only half the answer. The question a crew
        // member cares about is whether their remaining tab STARTS REPLICATING,
        // and `waitForLeadership` gating means "leader" and "replicating" are
        // two different facts. Assert the second one.
        const key = leaderIdx === 0 ? 'B' : 'A';
        const t1 = Date.now();
        let started = false;
        while (Date.now() - t1 < 45000) {
            started = (await follower.evaluate(k => window.SPIKE.repState(k), key)).started;
            if (started) break;
            await new Promise(r => setTimeout(r, 500));
        }
        const after = await follower.evaluate(k => window.SPIKE.repState(k), key);
        // eslint-disable-next-line no-console
        console.log(`[leg4] survivor replication state ${Date.now() - t1} ms after handover: ${JSON.stringify(after)}`);
        expect(started,
            'the survivor must not just WIN leadership, it must actually begin replicating').toBe(true);

        await ctx.close();
    });
});
