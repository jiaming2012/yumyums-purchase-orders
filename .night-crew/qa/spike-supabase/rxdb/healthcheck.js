// healthcheck.js — the RxDB + substrate half of `env-up.sh`'s verdict.
//
// 🛑 THE VERDICT IS THIS PROCESS'S EXIT STATUS, NEVER ITS PROSE.
//    Every leg below either asserts and continues, or records a failure. The
//    process exits 1 the moment any leg failed. A leg that cannot decide is a
//    FAILURE, never a pass — "could not check" and "checked, fine" must never
//    be the same exit code. That conflation is the silent no-op this whole
//    card exists to make impossible.
//
// ⚠ LOCAL SPIKE ONLY. Drives the throwaway `spike-supabase` compose project
//   and nothing else. Never a hosted Supabase project, never HQ prod, never
//   the dev Postgres on :5433.
//
// What it asserts, in order:
//
//   1. PORTS     — W1's compose publishes bare container ports, so the host
//                  side is Docker-assigned and changes on every `up`.
//                  spike-env.js resolves them; importing it at all proves the
//                  three services are published.
//   2. TOKEN     — minted by W1's *Go* minter, not a JS re-implementation.
//                  HQ's Go backend is the token authority; a JS shortcut here
//                  would prove nothing about the bridge.
//   3. REST      — PostgREST answers, the schema is applied, AND the policy
//                  DISCRIMINATES: alice sees her seed row and NOT bob's. A
//                  single-owner check cannot tell a working policy apart from
//                  a policy that lets everything through, which is why the
//                  fixture seeds two owners and why this leg checks both.
//   4. REALTIME  — the channel reaches SUBSCRIBED over the same token, through
//                  the tenant-Host shim. Anything else (CHANNEL_ERROR,
//                  TIMED_OUT, timeout) is a failure.
//   5. RXDB      — a real RxDB database is created against the spike schema,
//                  a document is inserted and read back. This is what makes
//                  the verdict "Supabase *and RxDB* up" rather than "three
//                  containers are running".
//
// Deliberately NOT here: replication. proof-push.js / proof-pull.js /
// proof-lww.js are W2's proofs and stay the proofs; this file answers the
// narrower question "is the environment up", which is C1's scope.

import {
    makeSupabaseClient, mintToken, makeLocalDb, TABLE, banner,
    REST_PORT, REALTIME_PORT, DB_PORT
} from './spike-env.js';

const failures = [];
function ok(leg, detail) { console.log(`PASS  ${leg}${detail ? ' — ' + detail : ''}`); }
function fail(leg, detail) { console.log(`FAIL  ${leg}${detail ? ' — ' + detail : ''}`); failures.push(leg); }

const REALTIME_TIMEOUT_MS = Number(process.env.SPIKE_RT_TIMEOUT_MS || 30000);

// --------------------------------------------------------------------------
// 1. Ports
// --------------------------------------------------------------------------
banner();
if (REST_PORT && REALTIME_PORT && DB_PORT) {
    ok('ports', `db=${DB_PORT} rest=${REST_PORT} realtime=${REALTIME_PORT}`);
} else {
    fail('ports', 'one or more host ports unresolved');
}

// --------------------------------------------------------------------------
// 2. Token — from the Go minter
// --------------------------------------------------------------------------
let token = null;
try {
    token = mintToken('user-alice');
    if (token.split('.').length !== 3) throw new Error(`not a 3-segment JWS: ${token.slice(0, 40)}…`);
    ok('token', `go-minted, ${token.length} chars, 3 segments`);
} catch (e) {
    fail('token', String(e && e.message ? e.message : e));
}

// --------------------------------------------------------------------------
// 3. PostgREST — schema applied AND the policy discriminates
// --------------------------------------------------------------------------
if (token) {
    try {
        const sb = makeSupabaseClient(token);
        const { data, error } = await sb.from(TABLE).select('*');
        if (error) {
            fail('rest', `PostgREST error: ${error.message || JSON.stringify(error)}`);
        } else if (!Array.isArray(data)) {
            fail('rest', `expected an array, got ${typeof data}`);
        } else {
            const owners = [...new Set(data.map((r) => r.owner_id))].sort();
            const sawAlice = data.some((r) => r.id === 'note-alice-1');
            const sawBob = data.some((r) => r.id === 'note-bob-1');
            if (!sawAlice) {
                // Either the fixture never applied, or the policy denies everything.
                fail('rest', `alice cannot see her own seed row note-alice-1 (rows=${data.length}, owners=${JSON.stringify(owners)})`);
            } else if (sawBob) {
                // RLS off, or a policy that lets everything through.
                fail('rest', `alice can see bob's row note-bob-1 — RLS is NOT discriminating (owners=${JSON.stringify(owners)})`);
            } else {
                ok('rest', `spike_notes readable; alice sees ${data.length} row(s), owners=${JSON.stringify(owners)}, bob's row correctly hidden`);
            }
        }
    } catch (e) {
        fail('rest', String(e && e.message ? e.message : e));
    }
} else {
    fail('rest', 'skipped — no token to authenticate with (a skipped leg is a failure, not a pass)');
}

// --------------------------------------------------------------------------
// 4. Realtime — SUBSCRIBED over the same token
// --------------------------------------------------------------------------
let rtClient = null;
if (token) {
    try {
        rtClient = makeSupabaseClient(token);
        const status = await new Promise((resolve) => {
            const t = setTimeout(() => resolve('TIMEOUT'), REALTIME_TIMEOUT_MS);
            rtClient
                .channel(`healthcheck-${Date.now()}`)
                .on('postgres_changes', { event: '*', schema: 'public', table: TABLE }, () => {})
                .subscribe((s, err) => {
                    if (s === 'SUBSCRIBED' || s === 'CHANNEL_ERROR' || s === 'TIMED_OUT' || s === 'CLOSED') {
                        clearTimeout(t);
                        resolve(err ? `${s} (${err.message || err})` : s);
                    }
                });
        });
        if (status === 'SUBSCRIBED') ok('realtime', 'channel SUBSCRIBED');
        else fail('realtime', `channel status ${status} (expected SUBSCRIBED)`);
    } catch (e) {
        fail('realtime', String(e && e.message ? e.message : e));
    }
} else {
    fail('realtime', 'skipped — no token (a skipped leg is a failure, not a pass)');
}

// --------------------------------------------------------------------------
// 5. RxDB — the local engine really works, not merely installed
// --------------------------------------------------------------------------
let db = null;
try {
    db = await makeLocalDb(`healthcheck${Date.now()}`);
    const id = `hc-${Date.now()}`;
    await db.notes.insert({ id, owner_id: 'user-alice', body: 'healthcheck row' });
    const back = await db.notes.findOne(id).exec();
    if (!back) fail('rxdb', 'inserted document did not read back');
    else if (back.body !== 'healthcheck row') fail('rxdb', `read back wrong body: ${back.body}`);
    else ok('rxdb', `local RxDB database created, doc ${id} inserted and read back`);
} catch (e) {
    fail('rxdb', String(e && e.message ? e.message : e));
} finally {
    try { if (db) await db.close(); } catch { /* teardown noise must not change the verdict */ }
    try { if (rtClient) await rtClient.removeAllChannels(); } catch { /* ditto */ }
}

// --------------------------------------------------------------------------
// Verdict
// --------------------------------------------------------------------------
if (failures.length) {
    console.log(`\nHEALTHCHECK: RED — ${failures.length} leg(s) failed: ${failures.join(', ')}`);
    process.exit(1);
}
console.log('\nHEALTHCHECK: GREEN — ports, token, rest, realtime, rxdb all asserted');
process.exit(0);
