// spike-e-reconnect.js — the CLIENT leg of night-crew card E `spike-e-reconnect-catchup`.
//
// ⚠ LOCAL SPIKE ONLY. Talks to the throwaway `spike-supabase` stack (spike A) and
//   to a real HQ backend running against the throwaway `spike-e-hq` scratch
//   Postgres. Never HQ's real database, never :5433 (that cluster is PRODUCTION
//   *and* dev — a probe there destroyed the prod DB on 2026-08-06), never :5434.
//
// ═══════════════════════════════════════════════════════════════════════════
// THE QUESTION
//
// Spike C proved the round trip: one write through HQ's real write path reaches a
// RUNNING RxDB client. Spike D proved the live Realtime filter. NEITHER EVER
// SEVERED A CLIENT. Everything Activity 3 wants to build assumes that a crew
// member's phone that was asleep, backgrounded, or on no signal while rows changed
// comes back and catches up. Nothing in this repo has ever measured that.
//
// So: subscribe, OBSERVE replication, sever, write while dark — at least one
// INSERT and at least one UPDATE TO A ROW THE CLIENT ALREADY HOLDS — reconnect,
// and see whether checkpoint pull recovers EVERYTHING.
//
// 🛑 THE UPDATE CASE IS NOT OPTIONAL. An INSERT can be recovered by any full
//    re-read; only an UPDATE to a document already in the local store exercises
//    the checkpoint path, and the checkpoint is the thing under test. A green run
//    that skipped it would be vacuous, so this script refuses to report one: the
//    UPDATE assertion is unconditional and its result is printed by name.
//
// ═══════════════════════════════════════════════════════════════════════════
// WHY THE SEVER IS AT THE CLIENT
//
// The card leaves the mechanism open — kill the socket, block the port, or pause
// the replication — and asks for whichever severs most attributably, preferring
// the one that keeps the SUBSTRATE untouched so spikes A-D keep reproducing.
//
// Severing at the client wins on both counts:
//
//   * it is ATTRIBUTABLE. `rep.cancel()` removes the Realtime channel and stops
//     the pull loop; `removeAllChannels()` + `realtime.disconnect()` then tears
//     the socket down. There is nothing left that could deliver a row, and the
//     script proves it rather than asserting it — see the DARK-WINDOW SILENCE
//     assertion, which re-reads the collection at the end of the dark window and
//     requires it to be byte-identical to the pre-sever snapshot.
//   * it touches NOTHING SHARED. Blocking a port or bouncing the substrate's
//     Realtime container would disturb the very stack spikes A-D's verdicts
//     reproduce against. This severs one Node process's own socket.
//
// And it is the honest model of the real failure: the phone goes away; the server
// does not.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 EXIT CODES — "could not run" is NOT a verdict.
//
//   0  GREEN. Every dark-window change was recovered on reconnect, INCLUDING the
//      UPDATE to the already-held row, within the bound.
//   1  RED — RAN, and catch-up demonstrably misses dark-window changes. This is a
//      SUCCESSFUL SPIKE: it means the build cards need an explicit resync step.
//   2  COULD NOT RUN — a setup failure. Replication never initialised, the HQ API
//      was unreachable, /saveResponse did not return 204, the relay never landed a
//      dark-window row in the substrate at all, the sever leaked (rows arrived
//      while the client was supposed to be dark), or the reconnected client is
//      provably dead. None of these say anything about catch-up and none of them
//      may be reported as a verdict.
//
// spike-e-reconnect.sh maps these straight through to its own contract.
// ═══════════════════════════════════════════════════════════════════════════

import { makeSupabaseClient, mintToken, REST_PORT, REALTIME_PORT, DB_PORT } from './spike-env.js';
import { replicateSupabase } from 'rxdb/plugins/replication-supabase';
import { addRxPlugin, createRxDatabase } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';

// ---------------------------------------------------------------------------
// Deliberately NOT importing hq-bridge-env.js — it installs process-level
// unhandledRejection/uncaughtException handlers that exit(1) unconditionally,
// which would turn every exit-2 "could not run" into an exit-1 "verdict". Spike C
// records the same reason. spike-env.js (spike A's) has no such handlers and is
// imported unchanged, read-only.
//
// 🛑 BUT the client leg's OWN main flow (steps 2-11 below) runs at MODULE TOP
// LEVEL — only the setup block near step 1 is wrapped in try/catch. An uncaught
// exception or unhandled rejection anywhere in the unprotected region escapes and
// Node exits 1 — a FALSE RED, the exact conflation this card (B-163 (c)) fixes.
// The fix is NOT to import hq-bridge-env's exit(1) handlers; it is to install our
// OWN handlers that exit 2 (the honest "could not run"). That is precisely what
// the note above wanted instead of exit(1). Installed immediately below, before
// any top-level code runs.
// ---------------------------------------------------------------------------

const SETUP = 2;
const RED = 1;

function die(code, msg, detail) {
    console.error(`\n${code === SETUP ? '🛑 COULD NOT RUN (not a verdict)' : '🛑 VERDICT: RED'}: ${msg}`);
    if (detail !== undefined) {
        console.error(typeof detail === 'string' ? detail : JSON.stringify(detail, null, 2));
    }
    process.exit(code);
}

// 🛑 B-163 (c). The main flow (steps 2-11) runs at module top level, unguarded.
// An uncaught exception / unhandled rejection there would otherwise let Node exit
// 1 — a FALSE RED. These handlers map ANY such escape to exit 2 (COULD NOT RUN),
// because an unexpected crash says NOTHING about catch-up. This is the honest
// counterpart to the exit(1) handlers the header note refused: same mechanism,
// correct code. `die` calls process.exit synchronously, so the first handler to
// fire wins and no second overlapping message is printed.
const escaped = (kind) => (e) => {
    const detail = e && e.stack ? e.stack : (e && e.message ? e.message : String(e));
    die(SETUP, `an ${kind} escaped the client leg's main flow — an unexpected crash is NOT a catch-up verdict`, detail);
};
process.on('uncaughtException', escaped('uncaught exception'));
process.on('unhandledRejection', escaped('unhandled promise rejection'));

function required(name) {
    const v = process.env[name];
    if (!v) die(SETUP, `environment variable ${name} is unset — this script is only ever run by spike-e-reconnect.sh`);
    return v;
}

const API_BASE = required('SPIKE_E_API_BASE');
const SESSION = required('SPIKE_E_SESSION');
const USER_ID = required('SPIKE_E_USER_ID');
const FIELD_A = required('SPIKE_E_FIELD_A');   // dark-window INSERT #1
const FIELD_B = required('SPIKE_E_FIELD_B');   // pre-sever save, then dark-window UPDATE
const FIELD_C = required('SPIKE_E_FIELD_C');   // dark-window INSERT #2
const FIELD_D = required('SPIKE_E_FIELD_D');   // post-reconnect LIVENESS control
const RUN = required('SPIKE_E_RUN_ID');
const SYNC_TABLE = process.env.SPIKE_E_SYNC_TABLE || 'hq_sync_checklists';
const DEADLINE_MS = Number(process.env.SPIKE_E_DEADLINE_MS || 20000);
const INIT_TIMEOUT_MS = Number(process.env.SPIKE_E_INIT_TIMEOUT_MS || 30000);
const RELAY_SETTLE_MS = Number(process.env.SPIKE_E_RELAY_SETTLE_MS || 30000);
const NO_PULL = process.env.SPIKE_E_NO_PULL === '1';

// Sentinels. Every value this run writes has never existed before, so no
// assertion can pass on a leftover row from an earlier run.
const S = (tag) => `spikee-${RUN}-${tag}-${Date.now()}`;
const SENT_B1 = S('B1-presever');
const SENT_A = S('A-darkinsert');
const SENT_B2 = S('B2-darkupdate');
const SENT_C = S('C-darkinsert');
const SENT_D = S('D-liveness');

console.log('# spike-e-reconnect.js');
console.log(`# substrate: rest=${REST_PORT} realtime=${REALTIME_PORT} db=${DB_PORT}`);
console.log(`# hq api:    ${API_BASE}`);
console.log(`# hq user:   ${USER_ID}`);
console.log(`# mode:      ${NO_PULL ? 'NO-PULL (red-first capture: realtime-only recovery)' : 'checkpoint pull ARMED'}`);
console.log(`# bound:     ${DEADLINE_MS} ms`);
console.log(`# run:       ${RUN}`);

// Mirrors spike A's hq_sync_checklists exactly, and is byte-identical to the
// schema spike-c-read.js uses. `_deleted` and `_modified` are NOT declared:
// RxDB owns `_deleted`, and leaving `_modified` undeclared keeps it a purely
// server-stamped pull cursor. This card needs to SEE `_modified` — it reads it
// straight off PostgREST with a probe client instead of declaring it, so the
// collection stays byte-identical to spike C's and the two are comparable.
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
let sbLive = null;      // the replicating client (severed mid-run)
let sbProbe = null;     // an independent REST-only client used to observe the substrate
let rtChannel = null;   // the red path's realtime-only subscription

async function shutdown() {
    try { if (rep) await rep.cancel(); } catch { /* teardown is best-effort */ }
    try { if (rtChannel && sbLive) await sbLive.removeChannel(rtChannel); } catch { /* ignore */ }
    try { if (sbLive) { sbLive.removeAllChannels(); sbLive.realtime.disconnect(); } } catch { /* ignore */ }
    try { if (sbProbe) { sbProbe.removeAllChannels(); sbProbe.realtime.disconnect(); } } catch { /* ignore */ }
    try { if (db) await db.close(); } catch { /* ignore */ }
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function docs() {
    return (await db.checklists.find().exec()).map((d) => d.toJSON());
}
function fingerprint(list) {
    return list
        .map((d) => `${d.id} ${d.body}`)
        .sort()
        .join('');
}
function findBySentinel(list, sentinel) {
    return list.find((d) => (d.body || '').includes(sentinel)) || null;
}

// The REAL write. POST /api/v1/workflow/saveResponse on a real HQ server, with a
// real hq_session cookie minted by a real POST /api/v1/auth/login, through the
// real auth middleware and the real `operations` grant gate. Nothing about this
// card weakens spike C's write path — it reuses it verbatim.
async function realWrite(label, fieldId, value) {
    let res;
    try {
        res = await fetch(`${API_BASE}/api/v1/workflow/saveResponse`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', Cookie: `hq_session=${SESSION}` },
            body: JSON.stringify({ field_id: fieldId, value })
        });
    } catch (e) {
        await shutdown();
        die(SETUP, `the HQ API was unreachable at ${API_BASE} during write ${label}`, e && e.message ? e.message : String(e));
    }
    const text = await res.text();
    console.log(`   write ${label}: HTTP ${res.status} field=${fieldId} value=${value}`);
    // 204 is the documented success. 200 accepted so a future handler change does
    // not red this spike for the wrong reason. Anything else is SETUP, never a
    // verdict: a write that did not happen says nothing about catch-up.
    if (res.status !== 204 && res.status !== 200) {
        await shutdown();
        die(SETUP, `/saveResponse returned HTTP ${res.status} for write ${label} — the write never happened, so there is nothing to recover`, text);
    }
}

// Observe the SUBSTRATE directly, independent of the replicating client. This is
// what makes "the client is dark" separable from "the relay never wrote": during
// the dark window the substrate is polled through this probe and must go AHEAD of
// the client.
async function substrateRows() {
    const { data, error } = await sbProbe
        .from(SYNC_TABLE)
        .select('id,owner_id,app_slug,body,_modified')
        .order('_modified', { ascending: true });
    if (error) throw new Error(`substrate probe select failed: ${error.message || JSON.stringify(error)}`);
    return data || [];
}

// ---------------------------------------------------------------------------
// 1. A RUNNING RxDB client, replicating over a real signed token with RLS live.
// ---------------------------------------------------------------------------
const REPL_ID = `spikee-${RUN}`;
let pullCheckpointsSeen = [];

function pullOptions(initialCheckpoint) {
    return {
        batchSize: 100,
        // 🛑 OBSERVATION, NOT MODIFICATION. The supabase plugin hands queryBuilder
        // the checkpoint it is ABOUT to pull from, before it appends its own
        // `_modified > m OR (_modified = m AND id > id)` clause. Returning a falsy
        // value leaves the query untouched (plugin index.ts: `if (maybeNewQuery)`),
        // so this changes nothing and reveals the one number the card is about:
        // whether the reconnect was a CHECKPOINT pull or a full re-read.
        queryBuilder: ({ query, lastPulledCheckpoint }) => {
            pullCheckpointsSeen.push(lastPulledCheckpoint === undefined ? null : lastPulledCheckpoint);
            return undefined;
        },
        ...(initialCheckpoint ? { initialCheckpoint } : {})
    };
}

let preSeverSnapshot;
let heldId = null;
let checkpointAtSever = null;

try {
    addRxPlugin(RxDBDevModePlugin);
    // sub = HQ's real user uuid — the same value the relay writes into owner_id,
    // and the identity axis of hq_sync_checklists' RLS compares the two.
    const token = mintToken(USER_ID, { ttl: '30m' });
    sbLive = makeSupabaseClient(token);
    sbProbe = makeSupabaseClient(mintToken(USER_ID, { ttl: '30m' }));

    db = await createRxDatabase({
        name: `spikee_${RUN}`,
        storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
        ignoreDuplicate: true
    });
    await db.addCollections({ checklists: { schema } });

    const replErrors = [];
    rep = replicateSupabase({
        replicationIdentifier: REPL_ID,
        collection: db.checklists,
        client: sbLive,
        tableName: SYNC_TABLE,
        waitForLeadership: false,
        live: true,
        pull: pullOptions(null)
    });
    rep.error$.subscribe((e) => replErrors.push(e.message || String(e)));

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
        die(SETUP, 'the RxDB client reported replication errors before anything was written', replErrors);
    }

    const initial = await docs();
    console.log(`\n══ 1. the client is LIVE: RxDB holds ${initial.length} doc(s) ══`);
    console.log(`   ${initial.map((d) => d.id).join(', ') || '(none)'}`);
} catch (e) {
    await shutdown();
    die(SETUP, 'could not stand up a running RxDB client against the substrate', e && e.message ? e.message : String(e));
}

// ---------------------------------------------------------------------------
// 2. POSITIVE ARRIVAL — spike D's rule. The client must be OBSERVED replicating
//    BEFORE the sever, or a dead subscription and a recovered one are
//    indistinguishable and the whole run is vacuous.
// ---------------------------------------------------------------------------
console.log(`\n══ 2. POSITIVE ARRIVAL (pre-sever) — a real write must reach the live client ══`);
await realWrite('B1 (pre-sever, field B)', FIELD_B, SENT_B1);

{
    const deadline = Date.now() + DEADLINE_MS;
    let hit = null;
    while (Date.now() < deadline) {
        hit = findBySentinel(await docs(), SENT_B1);
        if (hit) break;
        await sleep(100);
    }
    if (!hit) {
        const all = (await docs()).map((d) => d.id);
        await shutdown();
        die(SETUP, `the pre-sever write never reached the LIVE client within ${DEADLINE_MS} ms — the client was never replicating, so there is no gap to measure. RxDB holds: ${all.join(', ') || '(none)'}`);
    }
    heldId = hit.id;
    console.log(`   OBSERVED: ${SENT_B1} arrived as ${heldId} — the client is provably replicating.`);
}

// One /saveResponse fires the relay TWICE (the save, then sync/ops.go's async
// lamport stamp — spikec_relay.go records the measurement). Settle so the second
// projection lands BEFORE the checkpoint is captured; otherwise a stray late
// write would show up as a "dark-window change" that was never dark.
await sleep(4000);
preSeverSnapshot = await docs();
console.log(`   settled: RxDB holds ${preSeverSnapshot.length} doc(s) at the moment of severing`);

// ---------------------------------------------------------------------------
// 3. THE CHECKPOINT the client carries into the dark. Read straight off the
//    substrate for the rows the client holds — printed, so the reconnect's
//    "checkpoint pull" is a number in the log and not a claim.
// ---------------------------------------------------------------------------
try {
    const rows = await substrateRows();
    const held = new Set(preSeverSnapshot.map((d) => d.id));
    const mine = rows.filter((r) => held.has(r.id));
    if (mine.length) {
        const last = mine[mine.length - 1];
        checkpointAtSever = { id: last.id, modified: last._modified };
    }
} catch (e) {
    await shutdown();
    die(SETUP, 'could not read the substrate through the probe client', e && e.message ? e.message : String(e));
}
console.log(`\n══ 3. checkpoint at sever: ${JSON.stringify(checkpointAtSever)} ══`);

// ---------------------------------------------------------------------------
// 4. THE SEVER.
// ---------------------------------------------------------------------------
console.log(`\n══ 4. SEVERING the client ══`);
await rep.cancel();                 // removes the Realtime channel, stops the pull loop
rep = null;
sbLive.removeAllChannels();
sbLive.realtime.disconnect();       // and the socket itself
sbLive = null;
console.log('   replication cancelled, all channels removed, Realtime socket disconnected.');
console.log('   the client is DARK. Nothing in this process can now deliver a row into RxDB.');

// ---------------------------------------------------------------------------
// 5. THE DARK WINDOW — N real writes, through HQ's real write path.
//    N = 3: two INSERTs and, mandatorily, one UPDATE to a row the client HOLDS.
// ---------------------------------------------------------------------------
console.log(`\n══ 5. DARK WINDOW — 3 changes through POST /api/v1/workflow/saveResponse ══`);
await realWrite('A (INSERT — field never saved before)', FIELD_A, SENT_A);
await realWrite(`B2 (UPDATE — field B again; the client already holds ${heldId})`, FIELD_B, SENT_B2);
await realWrite('C (INSERT — field never saved before)', FIELD_C, SENT_C);

// The relay has to have landed all three in the substrate, or a "miss" on
// reconnect would mean the relay failed rather than that catch-up failed. That is
// a harness failure, so it is exit 2 and never a verdict.
console.log(`\n   waiting up to ${RELAY_SETTLE_MS} ms for the relay to project all three into the substrate`);
let substrateAtDarkEnd = [];
{
    const deadline = Date.now() + RELAY_SETTLE_MS;
    let ok = false;
    while (Date.now() < deadline) {
        substrateAtDarkEnd = await substrateRows();
        const blob = substrateAtDarkEnd.map((r) => r.body).join('');
        if (blob.includes(SENT_A) && blob.includes(SENT_B2) && blob.includes(SENT_C)) { ok = true; break; }
        await sleep(250);
    }
    if (!ok) {
        await shutdown();
        die(SETUP, `the relay did not project all three dark-window changes into ${SYNC_TABLE} within ${RELAY_SETTLE_MS} ms — that is a relay/harness failure, not a catch-up finding`, substrateAtDarkEnd.map((r) => ({ id: r.id, _modified: r._modified })));
    }
}
{
    const updated = substrateAtDarkEnd.find((r) => r.id === heldId);
    console.log(`   substrate now holds ${substrateAtDarkEnd.length} row(s); the already-held row ${heldId} reads _modified=${updated ? updated._modified : '(MISSING)'}`);
    if (!updated || !(updated.body || '').includes(SENT_B2)) {
        await shutdown();
        die(SETUP, `the dark-window UPDATE did not land on the SAME substrate row the client holds (${heldId}) — without that, the mandatory UPDATE case cannot be measured`, updated || '(row absent)');
    }
    console.log(`   CONFIRMED: the UPDATE re-wrote ${heldId} in place (same primary key, new body) — this is the already-held-row case.`);
}

// ---------------------------------------------------------------------------
// 6. DARK-WINDOW SILENCE. The sever is a MEASUREMENT, not a claim.
// ---------------------------------------------------------------------------
const nowDocs = await docs();
if (fingerprint(nowDocs) !== fingerprint(preSeverSnapshot)) {
    await shutdown();
    die(SETUP, 'the collection CHANGED while the client was supposed to be dark — the sever leaked, so nothing measured after it means anything', {
        before: preSeverSnapshot.map((d) => d.id),
        after: nowDocs.map((d) => d.id)
    });
}
console.log(`\n══ 6. DARK-WINDOW SILENCE VERIFIED ══`);
console.log(`   substrate: ${substrateAtDarkEnd.length} row(s)   ·   dark client: ${nowDocs.length} doc(s) — unchanged, byte for byte.`);
console.log('   the gap is real and measured, not assumed.');

// ---------------------------------------------------------------------------
// 7. RECONNECT.
//    GREEN path : checkpoint pull armed  — replicateSupabase resumes from the
//                 checkpoint the client carried into the dark.
//    RED path   : realtime-only          — the SAME client, the SAME applier, the
//                 SAME assertions, with the checkpoint pull leg absent. Realtime
//                 replays nothing that happened before the subscription existed,
//                 so it can only ever see the future.
// ---------------------------------------------------------------------------
console.log(`\n══ 7. RECONNECT (${NO_PULL ? 'realtime-only — NO checkpoint pull' : 'checkpoint pull armed'}) ══`);
pullCheckpointsSeen = [];
const reconnectToken = mintToken(USER_ID, { ttl: '30m' });
sbLive = makeSupabaseClient(reconnectToken);

if (NO_PULL) {
    // The applier. Byte-for-byte the transform the supabase plugin's own realtime
    // handler performs (plugin index.ts rowToDoc + pullStream$), applied locally
    // so the RED leg's assertion set is IDENTICAL to the GREEN leg's — the only
    // difference between the two runs is whether the checkpoint pull happened.
    const applied = [];
    const applyRow = async (row) => {
        if (!row || row._deleted) return;
        const doc = { id: row.id, owner_id: row.owner_id, app_slug: row.app_slug, body: row.body };
        try { await db.checklists.bulkUpsert([doc]); applied.push(row.id); } catch (e) { applied.push(`ERR:${e.message}`); }
    };
    let status = null;
    rtChannel = sbLive
        .channel(`spikee-rtonly-${RUN}`)
        .on('postgres_changes', { event: '*', schema: 'public', table: SYNC_TABLE },
            (payload) => { if (payload.eventType !== 'DELETE') void applyRow(payload.new); })
        .subscribe((s) => { status = s; });

    const deadline = Date.now() + INIT_TIMEOUT_MS;
    while (Date.now() < deadline && status !== 'SUBSCRIBED') {
        if (status === 'CHANNEL_ERROR' || status === 'TIMED_OUT') break;
        await sleep(100);
    }
    if (status !== 'SUBSCRIBED') {
        await shutdown();
        die(SETUP, `the realtime-only reconnect never reached SUBSCRIBED (last status: ${status}) — a dead subscription would make the red vacuous`);
    }
    console.log('   realtime-only subscription is SUBSCRIBED. NO checkpoint pull was issued.');
    globalThis.__rtApplied = applied;
} else {
    const replErrors = [];
    rep = replicateSupabase({
        replicationIdentifier: REPL_ID,
        collection: db.checklists,
        client: sbLive,
        tableName: SYNC_TABLE,
        waitForLeadership: false,
        live: true,
        // The checkpoint the client carried into the dark, passed explicitly so the
        // resume point is a value in this log rather than opaque internal state.
        // RxDB prefers its OWN stored checkpoint when it has one; either way the
        // value actually used is captured by queryBuilder above and printed below.
        pull: pullOptions(checkpointAtSever)
    });
    rep.error$.subscribe((e) => replErrors.push(e.message || String(e)));
    let t2;
    try {
        await Promise.race([
            rep.awaitInitialReplication(),
            new Promise((_, rej) => {
                t2 = setTimeout(() => rej(new Error(
                    `the reconnected replication did not complete an initial pull within ${INIT_TIMEOUT_MS} ms`)), INIT_TIMEOUT_MS);
            })
        ]).finally(() => clearTimeout(t2));
    } catch (e) {
        await shutdown();
        die(SETUP, 'the reconnected client could not complete its first pull', e && e.message ? e.message : String(e));
    }
    if (replErrors.length) {
        await shutdown();
        die(SETUP, 'the reconnected client reported replication errors', replErrors);
    }
    console.log(`   reconnected. checkpoints handed to the pull handler: ${JSON.stringify(pullCheckpointsSeen)}`);
    // 🛑 "via checkpoint pull" made executable. A first pull with an UNDEFINED
    // checkpoint is a FULL RE-READ, which recovers everything for a reason that
    // has nothing to do with the checkpoint and would not survive a real dataset.
    if (!pullCheckpointsSeen.length || pullCheckpointsSeen[0] === null) {
        await shutdown();
        die(SETUP, 'the reconnected client pulled with NO checkpoint — that is a full re-read, not the checkpoint pull this card is measuring. The result would say nothing about catch-up at scale.', pullCheckpointsSeen);
    }
    console.log(`   the first post-reconnect pull resumed FROM A CHECKPOINT: ${JSON.stringify(pullCheckpointsSeen[0])}`);
}

// ---------------------------------------------------------------------------
// 8. THE ASSERTION. Identical on both paths.
// ---------------------------------------------------------------------------
console.log(`\n══ 8. CATCH-UP — waiting up to ${DEADLINE_MS} ms for all 3 dark-window changes ══`);
const t0 = Date.now();
let result = { insertA: false, updateB: false, insertC: false };
let elapsed = 0;
{
    const deadline = t0 + DEADLINE_MS;
    while (Date.now() < deadline) {
        const all = await docs();
        const held = all.find((d) => d.id === heldId) || null;
        result = {
            insertA: !!findBySentinel(all, SENT_A),
            // 🛑 THE MANDATORY CASE: the SAME primary key the client already held,
            // now carrying the dark-window value. A new id would be an insert, not
            // an update, and would not exercise the checkpoint path at all.
            updateB: !!(held && (held.body || '').includes(SENT_B2)),
            insertC: !!findBySentinel(all, SENT_C)
        };
        if (result.insertA && result.updateB && result.insertC) break;
        await sleep(100);
    }
    elapsed = Date.now() - t0;
}
const heldAfter = (await docs()).find((d) => d.id === heldId) || null;
console.log(`   INSERT  (field A, new row)                  : ${result.insertA ? 'RECOVERED' : 'MISSED'}`);
console.log(`   UPDATE  (field B, row ${heldId} already held): ${result.updateB ? 'RECOVERED' : 'MISSED'}`);
console.log(`   INSERT  (field C, new row)                  : ${result.insertC ? 'RECOVERED' : 'MISSED'}`);
console.log(`   after ${elapsed} ms the already-held row reads: ${heldAfter ? (heldAfter.body || '').slice(0, 220) : '(absent)'}`);

// ---------------------------------------------------------------------------
// 9. LIVENESS CONTROL. Run on BOTH paths, ALWAYS, even when 8 already failed.
//    This is what makes the red attributable: a realtime-only client that misses
//    the dark window but CATCHES a post-reconnect write is provably alive, so the
//    miss is the ABSENT CHECKPOINT PULL and not a dead socket.
// ---------------------------------------------------------------------------
console.log(`\n══ 9. LIVENESS CONTROL — one post-reconnect write must arrive ══`);
await realWrite('D (post-reconnect liveness, field D)', FIELD_D, SENT_D);
let live = false;
{
    const deadline = Date.now() + DEADLINE_MS;
    while (Date.now() < deadline) {
        if (findBySentinel(await docs(), SENT_D)) { live = true; break; }
        await sleep(100);
    }
}
console.log(`   post-reconnect write: ${live ? 'ARRIVED' : 'DID NOT ARRIVE'}`);
if (NO_PULL) {
    console.log(`   rows the realtime-only applier accepted after reconnect: ${JSON.stringify(globalThis.__rtApplied || [])}`);
}

// ---------------------------------------------------------------------------
// 10. THE WATERMARK FINDING. Measured on the real write path's own rows, in this
//     run — not read off the source.
// ---------------------------------------------------------------------------
console.log(`\n══ 10. WATERMARK SEMANTICS (measured, this run) ══`);
let finding = {};
try {
    const rows = await substrateRows();
    const after = rows.find((r) => r.id === heldId) || null;
    const before = substrateAtDarkEnd.find((r) => r.id === heldId) || null;
    let answeredBefore = null, answeredAfter = null;
    try { answeredBefore = JSON.parse(preSeverSnapshot.find((d) => d.id === heldId).body).answered_at; } catch { /* leave null */ }
    try { answeredAfter = JSON.parse(after.body).answered_at; } catch { /* leave null */ }
    finding = {
        row: heldId,
        hq_answered_at_before_update: answeredBefore,
        hq_answered_at_after_update: answeredAfter,
        hq_answered_at_advanced: !!(answeredBefore && answeredAfter && answeredAfter > answeredBefore),
        substrate_modified_at_dark_end: before ? before._modified : null,
        substrate_modified_now: after ? after._modified : null,
        checkpoint_carried_into_the_dark: checkpointAtSever,
        checkpoints_handed_to_the_pull_handler: pullCheckpointsSeen
    };
} catch (e) {
    finding = { error: e && e.message ? e.message : String(e) };
}
console.log(JSON.stringify(finding, null, 2));
console.log('   ^ hq_answered_at_* is submission_responses.answered_at as HQ stamped it, carried');
console.log('     verbatim in the relay-projected body. substrate_modified_* is the column the');
console.log('     RxDB supabase plugin actually checkpoints on (_modified, trigger-stamped).');

// ---------------------------------------------------------------------------
// 11. VERDICT.
// ---------------------------------------------------------------------------
const recovered = result.insertA && result.updateB && result.insertC;

if (!live) {
    // Nothing after the reconnect can be trusted if the reconnected client cannot
    // receive at all. That is a harness failure on either path, never a verdict.
    await shutdown();
    die(SETUP, `the reconnected client did not receive a post-reconnect write within ${DEADLINE_MS} ms either — it is not alive, so the catch-up result above is vacuous`);
}

if (!recovered) {
    await shutdown();
    const missed = Object.entries(result).filter(([, v]) => !v).map(([k]) => k);
    die(RED, `catch-up MISSED dark-window changes: ${missed.join(', ')}.`
        + ` The reconnected client is provably alive (the post-reconnect write arrived), so the miss is the`
        + ` ${NO_PULL ? 'ABSENT CHECKPOINT PULL — this is the expected RED-FIRST capture' : 'CATCH-UP MECHANISM ITSELF'}.`
        + ` A build card that assumes reconnect self-heals needs an explicit resync step.`);
}

console.log('\n══════════════════════════════════════════════════════════');
console.log('✅ FULL RECOVERY — every dark-window change reached the reconnected client.');
console.log(`   INSERT + INSERT + UPDATE-to-an-already-held-row, all 3, in ${elapsed} ms (bound ${DEADLINE_MS} ms).`);
console.log(`   The UPDATE case was EXERCISED and RECOVERED: ${heldId} was in the local store before`);
console.log('   the sever and carries the dark-window value after it.');
console.log('══════════════════════════════════════════════════════════');

await shutdown();
process.exit(0);
