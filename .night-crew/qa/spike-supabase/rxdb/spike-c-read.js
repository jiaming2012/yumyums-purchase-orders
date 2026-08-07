// spike-c-read.js — the ROUND-TRIP leg of night-crew card C `spike-c-round-trip`.
//
// ⚠ LOCAL SPIKE ONLY. Talks to the throwaway `spike-supabase` stack (spike A)
//   and to a real HQ backend running against the throwaway `spike-c-hq` scratch
//   Postgres. Never HQ's real database, never :5433 (that cluster is
//   PRODUCTION), never :5434.
//
// ═══════════════════════════════════════════════════════════════════════════
// WHY THE WRITE IS MADE FROM INSIDE THIS PROCESS
//
// The card's claim is that a row written through the REAL write path appears in
// an RxDB-served read WITHIN BOUNDED TIME. Two things follow:
//
//   * the RxDB client must ALREADY BE RUNNING when the write happens. A client
//     started afterwards proves the initial-sync path and nothing else, and
//     initial sync is not what an offline-first PWA needs. So this script
//     starts replication, waits for initial replication, snapshots what it
//     holds, and only THEN issues the write.
//   * the bound has to be measured across the write, not around it. Splitting
//     the write into the shell would put process startup, `docker exec` and
//     `curl` inside the measured window and make the number meaningless.
//
// The write is still the real one: a real HTTP POST to
// /api/v1/workflow/saveResponse on a real HQ server, carrying a real hq_session
// cookie minted by a real POST /api/v1/auth/login, through the real auth
// middleware and the real `operations` grant gate.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 EXIT CODES — "could not run" is NOT a verdict.
//
//   0  the row written through /saveResponse reached the RxDB collection
//   1  RAN AND THE MECHANISM IS DISPROVEN — the deadline expired with the row
//      absent, or it arrived carrying the wrong thing. This is the RED verdict
//      and it is a SUCCESSFUL spike outcome.
//   2  COULD NOT RUN — a setup failure. Replication never initialised, the HQ
//      API was unreachable, /saveResponse did not return 204, the sentinel was
//      somehow already present. None of these say anything about the mechanism
//      and none of them may be reported as a verdict.
//
// spike-c-roundtrip.sh maps these straight through to its own contract.
// ═══════════════════════════════════════════════════════════════════════════

import { makeSupabaseClient, mintToken, REST_PORT, REALTIME_PORT, DB_PORT } from './spike-env.js';
import { replicateSupabase } from 'rxdb/plugins/replication-supabase';
import { addRxPlugin, createRxDatabase } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';

// ---------------------------------------------------------------------------
// Deliberately NOT importing hq-bridge-env.js, even though it has a schema and
// helpers this could reuse. That module installs process-level
// unhandledRejection/uncaughtException handlers that exit(1) unconditionally —
// correct for spike B, fatal here, because it would turn every exit-2 "could
// not run" into an exit-1 "mechanism disproven". Keeping the two apart is what
// keeps this script's exit-code contract true. spike-env.js (spike A's) has no
// such handlers and is imported unchanged.
// ---------------------------------------------------------------------------

const SETUP = 2;   // could not run
const RED = 1;     // ran, mechanism disproven

function die(code, msg, detail) {
    console.error(`\n${code === SETUP ? '🛑 COULD NOT RUN' : '🛑 ROUND TRIP RED'}: ${msg}`);
    if (detail !== undefined) {
        console.error(typeof detail === 'string' ? detail : JSON.stringify(detail, null, 2));
    }
    process.exit(code);
}

function required(name) {
    const v = process.env[name];
    if (!v) die(SETUP, `environment variable ${name} is unset — this script is only ever run by spike-c-roundtrip.sh`);
    return v;
}

const API_BASE = required('SPIKE_C_API_BASE');
const SESSION = required('SPIKE_C_SESSION');
const FIELD_ID = required('SPIKE_C_FIELD_ID');
const USER_ID = required('SPIKE_C_USER_ID');
const SENTINEL = required('SPIKE_C_SENTINEL');
const SYNC_TABLE = process.env.SPIKE_C_SYNC_TABLE || 'hq_sync_checklists';
const DEADLINE_MS = Number(process.env.SPIKE_C_DEADLINE_MS || 20000);
const INIT_TIMEOUT_MS = Number(process.env.SPIKE_C_INIT_TIMEOUT_MS || 30000);
const RUN = process.env.SPIKE_C_RUN_ID || `c${Date.now()}`;

console.log(`# spike-c-read.js`);
console.log(`# substrate: rest=${REST_PORT} realtime=${REALTIME_PORT} db=${DB_PORT}`);
console.log(`# hq api:    ${API_BASE}`);
console.log(`# hq user:   ${USER_ID}`);
console.log(`# field:     ${FIELD_ID}`);
console.log(`# sentinel:  ${SENTINEL}`);
console.log(`# bound:     ${DEADLINE_MS} ms`);
console.log(`# run:       ${RUN}`);

// The projected row's shape. Mirrors spike A's hq_sync_checklists exactly.
// `_deleted` and `_modified` are NOT declared, for the reason spike-env.js
// records: RxDB owns `_deleted`, and leaving `_modified` undeclared keeps it a
// purely server-stamped pull cursor.
const schema = {
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        owner_id: { type: 'string', maxLength: 100 },
        app_slug: { type: 'string', maxLength: 100 },
        body: { type: 'string' }
    },
    required: ['id', 'owner_id', 'app_slug', 'body']
};

let rep = null;
let db = null;
async function shutdown() {
    try { if (rep) await rep.cancel(); } catch { /* teardown is best-effort */ }
    try { if (db) await db.close(); } catch { /* teardown is best-effort */ }
}

// ---------------------------------------------------------------------------
// 1. A RUNNING RxDB client, replicating over a real signed token with RLS live.
// ---------------------------------------------------------------------------
let docs;
try {
    addRxPlugin(RxDBDevModePlugin);
    // sub = HQ's real user uuid. That is the same value the relay writes into
    // owner_id, and the identity axis of hq_sync_checklists' RLS policy compares
    // the two. If they did not match, the row would land in the substrate and
    // still be invisible here — which is a distinct and much nastier failure
    // than "the row never arrived", so it is worth being able to tell apart.
    const token = mintToken(USER_ID, { ttl: '30m' });
    const sb = makeSupabaseClient(token);
    db = await createRxDatabase({
        name: `spikec_${RUN}`,
        storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
        ignoreDuplicate: true
    });
    await db.addCollections({ checklists: { schema } });

    const replErrors = [];
    rep = replicateSupabase({
        replicationIdentifier: `spikec-${RUN}`,
        collection: db.checklists,
        client: sb,
        tableName: SYNC_TABLE,
        waitForLeadership: false,
        live: true,
        pull: { batchSize: 100 }
    });
    rep.error$.subscribe((e) => replErrors.push(e.message || String(e)));

    // A hang is a failure with a name on it, never a run that quietly never
    // ends. Spike A proved a phx_join can reply {"status":"ok"} while the
    // postgres_changes subscription has actually FAILED, the real error arriving
    // later on a separate `system` frame.
    let t;
    await Promise.race([
        rep.awaitInitialReplication(),
        new Promise((_, rej) => {
            t = setTimeout(() => rej(new Error(
                `initial replication did not complete within ${INIT_TIMEOUT_MS} ms`)), INIT_TIMEOUT_MS);
        })
    ]).finally(() => clearTimeout(t));

    if (replErrors.length) {
        await shutdown();
        die(SETUP, 'the RxDB client reported replication errors before the write', replErrors);
    }

    docs = (await db.checklists.find().exec()).map((d) => d.toJSON());
    console.log(`\n── the client is LIVE before the write: RxDB holds ${docs.length} doc(s) ──`);
    console.log(`   ${docs.map((d) => d.id).join(', ') || '(none)'}`);
} catch (e) {
    await shutdown();
    die(SETUP, 'could not stand up a running RxDB client against the substrate', e && e.message ? e.message : String(e));
}

// A pre-existing sentinel would make the whole measurement vacuous — the row
// would "arrive" instantly having been there all along. This is the check that
// makes a green mean something.
if (docs.some((d) => (d.body || '').includes(SENTINEL))) {
    await shutdown();
    die(SETUP, `the sentinel ${SENTINEL} is ALREADY present in RxDB before the write — the measurement would be vacuous`);
}

// ---------------------------------------------------------------------------
// 2. THE REAL WRITE. POST /api/v1/workflow/saveResponse.
// ---------------------------------------------------------------------------
console.log('\n── the write: POST /api/v1/workflow/saveResponse ──');
const t0 = Date.now();
let res;
try {
    res = await fetch(`${API_BASE}/api/v1/workflow/saveResponse`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Cookie: `hq_session=${SESSION}` },
        body: JSON.stringify({ field_id: FIELD_ID, value: SENTINEL })
    });
} catch (e) {
    await shutdown();
    die(SETUP, `the HQ API was unreachable at ${API_BASE}`, e && e.message ? e.message : String(e));
}
const writeBody = await res.text();
console.log(`   HTTP ${res.status} ${writeBody ? `· ${writeBody.slice(0, 200)}` : '(no content)'}`);
// 204 is the documented success (handler.go: "Returns 204 No Content"). 200 is
// accepted too so a future handler change does not red the spike for the wrong
// reason; anything else is a SETUP failure, never a verdict — a write that did
// not happen says nothing about whether writes propagate.
if (res.status !== 204 && res.status !== 200) {
    await shutdown();
    die(SETUP, `/saveResponse returned HTTP ${res.status} — the write never happened, so there is nothing to measure`, writeBody);
}

// ---------------------------------------------------------------------------
// 3. THE READ. Does it reach the RUNNING client, and how fast?
// ---------------------------------------------------------------------------
console.log(`\n── the read: waiting up to ${DEADLINE_MS} ms for the row in RxDB ──`);
const deadline = t0 + DEADLINE_MS;
let arrived = null;
while (Date.now() < deadline) {
    const all = (await db.checklists.find().exec()).map((d) => d.toJSON());
    arrived = all.find((d) => (d.body || '').includes(SENTINEL)) || null;
    if (arrived) break;
    await new Promise((r) => setTimeout(r, 100));
}
const elapsed = Date.now() - t0;

if (!arrived) {
    const all = (await db.checklists.find().exec()).map((d) => d.id);
    await shutdown();
    console.error(`\n   after ${elapsed} ms the RxDB collection holds: ${all.join(', ') || '(none)'}`);
    die(RED, `the row written through /saveResponse did NOT reach the RxDB-served read within ${DEADLINE_MS} ms.`
        + ` There is no mechanism carrying HQ-Postgres writes into the substrate.`);
}

console.log(`   ARRIVED in ${elapsed} ms -> ${arrived.id}`);
console.log(`   ${JSON.stringify(arrived, null, 2)}`);

// Arrival is not enough. Three independent things have to be true of the
// document, and each one can fail while the others hold:
const failures = [];
if (arrived.owner_id !== USER_ID) {
    failures.push(`owner_id is ${arrived.owner_id}, expected HQ's real user uuid ${USER_ID}`);
}
let parsed = null;
try { parsed = JSON.parse(arrived.body); } catch { failures.push('body is not JSON'); }
if (parsed) {
    if (parsed.field_id !== FIELD_ID) {
        failures.push(`body.field_id is ${parsed.field_id}, expected ${FIELD_ID}`);
    }
    if (parsed.value !== SENTINEL) {
        failures.push(`body.value is ${JSON.stringify(parsed.value)}, expected ${JSON.stringify(SENTINEL)}`);
    }
}
if (failures.length) {
    await shutdown();
    die(RED, 'the row arrived but does not carry what was written', failures);
}

console.log(`\n   PASS  a row written through /saveResponse reached a RUNNING RxDB client`);
console.log(`   PASS  it carries HQ's real user uuid in owner_id (identity axis intact through the projection)`);
console.log(`   PASS  it carries the exact field_id and value the write path was given`);
console.log(`\nROUND TRIP CLOSED in ${elapsed} ms (bound ${DEADLINE_MS} ms)`);

await shutdown();
process.exit(0);
