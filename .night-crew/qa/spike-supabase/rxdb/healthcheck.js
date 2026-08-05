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
//   3. SCHEMA    — every object all three fixture files create exists, BY NAME:
//                  the three hq_* bridge tables and the five hq_* policies, not
//                  just spike_notes. This leg was missing until the C1 fix
//                  round and its absence was a schema-shaped blind spot: with
//                  hq-bridge-fixture.sql and hq-bridge-policies.sql never
//                  applied, `task spike:health` exited 0 on a database with no
//                  hq_grant_projection, no hq_sync_checklists and no
//                  hq_uid_trap — the exact tables internal/sync's
//                  TestRowVisibilityRLS (59 subtests) drives. That is the same
//                  "container up, no schema" conflation this card exists to
//                  retire, relocated to the other half of the schema, and
//                  env-up.sh:9's claim that exit 0 means "both fixture schemas
//                  applied" was not proven by anything until this leg existed.
//                  It queries the catalog through `docker compose exec db psql`
//                  because pg_policies is not reachable through PostgREST.
//   4. REST      — PostgREST answers, the schema is applied, AND the policy
//                  DISCRIMINATES: alice sees her seed row and NOT bob's. A
//                  single-owner check cannot tell a working policy apart from
//                  a policy that lets everything through, which is why the
//                  fixture seeds two owners and why this leg checks both.
//   5. REALTIME  — the channel reaches SUBSCRIBED over the same token, through
//                  the tenant-Host shim. Anything else (CHANNEL_ERROR,
//                  TIMED_OUT, timeout) is a failure. A CHANNEL_ERROR against a
//                  healthy db is the signature of a db recreate having killed
//                  the replication slot; env-up.sh's reconcile leg restarts
//                  realtime on exactly this FAIL line and re-asserts.
//   6. RXDB      — a real RxDB database is created against the spike schema,
//                  a document is inserted and read back. This is what makes
//                  the verdict "Supabase *and RxDB* up" rather than "three
//                  containers are running".
//
// Deliberately NOT here: replication. proof-push.js / proof-pull.js /
// proof-lww.js are W2's proofs and stay the proofs; this file answers the
// narrower question "is the environment up", which is C1's scope.

import { execFileSync } from 'node:child_process';
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
// 3. Schema — every object the three fixture files create, asserted BY NAME.
//
// Queried through `docker compose exec db psql` rather than PostgREST because
// pg_policies lives in the catalog, which PGRST_DB_SCHEMAS=public does not
// expose. `-p spike-supabase` is what keeps this pointed at the throwaway
// stack and nothing else.
// --------------------------------------------------------------------------
const HQ_TABLES = ['hq_grant_projection', 'hq_sync_checklists', 'hq_uid_trap'];
const HQ_POLICIES = [
    'hq_sync_checklists_select', 'hq_sync_checklists_insert',
    'hq_sync_checklists_update', 'hq_uid_trap_select',
    'hq_grant_projection_select'
];

function psql(sql) {
    return execFileSync(
        'docker',
        ['compose', '-p', 'spike-supabase', 'exec', '-T', 'db',
            'psql', '-U', 'supabase_admin', '-d', 'postgres', '-Atc', sql],
        { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }
    ).trim();
}

try {
    const tables = psql(
        "select tablename from pg_tables where schemaname='public' order by 1"
    ).split('\n').filter(Boolean);
    const policies = psql(
        "select policyname from pg_policies where schemaname='public' order by 1"
    ).split('\n').filter(Boolean);

    const missingTables = [TABLE, ...HQ_TABLES].filter((t) => !tables.includes(t));
    const missingPolicies = HQ_POLICIES.filter((p) => !policies.includes(p));

    if (missingTables.length || missingPolicies.length) {
        // Naming what is missing AND the remedy: a health run that skips the
        // apply cannot fix this, so say which command can.
        fail('schema',
            `missing table(s): ${missingTables.length ? missingTables.join(', ') : 'none'}; ` +
            `missing policy(ies): ${missingPolicies.length ? missingPolicies.join(', ') : 'none'}. ` +
            `Present tables: ${JSON.stringify(tables)}. ` +
            'Remedy: run env-up.sh WITHOUT --health (or with --fresh) so sql/spike-fixture.sql, ' +
            'sql/hq-bridge-fixture.sql and sql/hq-bridge-policies.sql are applied. ' +
            'internal/sync TestRowVisibilityRLS drives the hq_* tables and cannot run without them.');
    } else {
        ok('schema',
            `${[TABLE, ...HQ_TABLES].length} fixture table(s) and ${HQ_POLICIES.length} hq_* policy(ies) present by name`);
    }
} catch (e) {
    // "could not check" is a FAILURE, never a pass.
    fail('schema', `could not query the catalog via 'docker compose -p spike-supabase exec db psql': ${String(e && e.message ? e.message : e)}`);
}

// --------------------------------------------------------------------------
// 4. PostgREST — schema applied AND the policy discriminates
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
// 5. Realtime — SUBSCRIBED over the same token
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
// 6. RxDB — the local engine really works, not merely installed
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
console.log('\nHEALTHCHECK: GREEN — ports, token, schema, rest, realtime, rxdb all asserted');
process.exit(0);
