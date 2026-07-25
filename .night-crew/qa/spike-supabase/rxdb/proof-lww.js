// proof-lww.js — OBSERVE the conflict behaviour. Do not assume it.
//
// The explore session chose "last-write-wins, no custom conflict handler". This
// script builds one concurrent-write case and records what ACTUALLY happens:
// which side's value survives, which clock (if any) decided it, and whether the
// losing write disappears with or without a signal the app could react to.
//
// THE CASE, and why it is shaped this way:
//   1. one document, agreed on both sides;
//   2. the client goes offline (replication cancelled);
//   3. Postgres is edited FIRST  (t = T1);
//   4. RxDB is edited SECOND (t = T2 > T1) — the LOCAL write is strictly LATER
//      in wall-clock time;
//   5. the client reconnects.
//
// Step 4 is the load-bearing one. Under genuine last-write-wins the later write
// — the local one — must survive. If the earlier remote write survives instead,
// the configuration is NOT last-write-wins, whatever it was assumed to be, and
// no amount of clock skew explains it.
//
// FAILURE THIS CATCHES: shipping an offline-first PWA on the belief that a
// crew member's offline edit is safe until something newer overwrites it,
// when in fact reconnecting can discard it.
import {
    makeSupabaseClient, makeLocalDb, mintToken, observingConflictHandler,
    TABLE, RUN, REST_PORT, banner
} from './spike-env.js';
import { replicateSupabase } from 'rxdb/plugins/replication-supabase';

banner();
const token = mintToken('user-alice');
const sb = makeSupabaseClient(token);

const conflictLog = [];
const db = await makeLocalDb(`lww_${RUN}`, {
    conflictHandler: observingConflictHandler((e) => conflictLog.push(e))
});
const notes = db.notes;

const AUTH = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };
const restUrl = (p) => `http://127.0.0.1:${REST_PORT}/${p}`;
const id = `lww-${RUN}`;
const errors = [];

async function readRemote() {
    const r = await fetch(restUrl(`${TABLE}?id=eq.${id}&select=*`), { headers: AUTH });
    return (await r.json())[0];
}

function startReplication(tag) {
    const rep = replicateSupabase({
        // A DIFFERENT replicationIdentifier per connection would give the second
        // connection a blank checkpoint. Keeping it the same is what makes this
        // a genuine reconnect of the SAME replica rather than a fresh client.
        replicationIdentifier: `lww-${RUN}`,
        collection: notes,
        client: sb,
        tableName: TABLE,
        waitForLeadership: false,
        live: true,
        pull: { batchSize: 50 },
        push: { batchSize: 50 }
    });
    rep.error$.subscribe((e) => {
        errors.push(`${tag}: ${e.message || e}`);
        console.log(`!! replication error (${tag}):`, e.message || e);
    });
    return rep;
}

// --- 1. agree ---------------------------------------------------------------
const repA = startReplication('A');
await repA.awaitInitialReplication();
await notes.insert({ id, owner_id: 'user-alice', body: 'agreed-original' });
await repA.awaitInSync();
console.log('1. agreed state       remote:', JSON.stringify(await readRemote()));

// --- 2. go offline ----------------------------------------------------------
await repA.cancel();
console.log('2. replication cancelled — the client is now "offline"');

// --- 3. remote edit FIRST ---------------------------------------------------
const rRes = await fetch(restUrl(`${TABLE}?id=eq.${id}`), {
    method: 'PATCH', headers: { ...AUTH, Prefer: 'return=representation' },
    body: JSON.stringify({ body: 'REMOTE-EDIT (written first, T1)' })
});
console.log(`3. remote edit HTTP ${rRes.status} at T1 ->`, JSON.stringify(await readRemote()));

// A visible gap, so "later" is not a scheduling accident.
await new Promise((r) => setTimeout(r, 1500));

// --- 4. local edit SECOND (strictly later wall-clock) -----------------------
const localWriteAt = new Date().toISOString();
await (await notes.findOne(id).exec()).patch({ body: 'LOCAL-EDIT (written second, T2)' });
console.log(`4. local edit at T2=${localWriteAt} -> local body now:`,
    (await notes.findOne(id).exec()).body);

// --- 5. reconnect -----------------------------------------------------------
console.log('5. reconnecting...');
const repB = startReplication('B');
await repB.awaitInitialReplication();
await repB.awaitInSync();
await new Promise((r) => setTimeout(r, 2000)); // let any late frames land

const finalLocal = await notes.findOne(id).exec();
const finalRemote = await readRemote();

console.log('');
console.log('=========================== OBSERVED ===========================');
console.log('local  body after reconnect :', finalLocal ? finalLocal.body : null);
console.log('remote body after reconnect :', finalRemote.body);
console.log('remote _modified            :', finalRemote._modified,
    '  (stamped by the Postgres trigger, i.e. the SERVER clock)');
console.log('local write happened at     :', localWriteAt, '  (the CLIENT clock)');
console.log('replication errors surfaced :', errors.length, errors);
console.log('conflict handler invocations:', conflictLog.length);
for (const c of conflictLog) {
    console.log('  - assumedMasterState.body:', c.assumedMasterState && c.assumedMasterState.body);
    console.log('    newDocumentState.body  :', c.newDocumentState && c.newDocumentState.body,
        '   <- the local (later) write');
    console.log('    realMasterState.body   :', c.realMasterState && c.realMasterState.body,
        '   <- what the server actually held');
    console.log('    handler CHOSE          :', c.chosen && c.chosen.body);
}

const localSurvived = finalLocal && finalLocal.body.startsWith('LOCAL-EDIT');
const remoteSurvived = finalRemote.body.startsWith('REMOTE-EDIT');
console.log('');
console.log('VERDICT: the LATER (local) write ' + (localSurvived ? 'SURVIVED' : 'WAS DISCARDED'));
console.log('VERDICT: the EARLIER (remote) write ' + (remoteSurvived ? 'SURVIVED' : 'was overwritten'));
console.log(localSurvived
    ? 'CONSISTENT with last-write-wins.'
    : 'NOT last-write-wins. The winner is the MASTER (server) state regardless of '
      + 'which write happened later; no timestamp participated in the decision.');
console.log('the losing write was ' + (errors.length === 0
    ? 'discarded SILENTLY — nothing was emitted on error$ for the app to react to.'
    : 'accompanied by an error$ emission.'));
console.log('================================================================');

await repB.cancel();
await db.close();
process.exit(0);
