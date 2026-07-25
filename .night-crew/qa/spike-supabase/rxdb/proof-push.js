// proof-push.js — PUSH ONLY. Write locally in RxDB, then observe the row in
// Postgres.
//
// WHY THIS IS A SEPARATE SCRIPT FROM proof-pull.js: a single script that starts
// replication and then checks "the data is on both sides" cannot tell you which
// direction carried it. Splitting them is the whole point — a one-directional
// proof dressed up as bidirectional is the most common way this kind of spike
// fools itself.
//
// FAILURE THIS CATCHES: RxDB's push handler silently swallowing a PostgREST
// rejection (RLS refusal, missing grant, column mismatch) and the replication
// state still reporting "in sync" because it has nothing queued. The
// verification below deliberately re-reads the row over a SEPARATE HTTP call
// rather than trusting RxDB's own view of the world.
import { makeSupabaseClient, makeLocalDb, mintToken, TABLE, RUN, REST_PORT, banner }
    from './spike-env.js';
import { replicateSupabase } from 'rxdb/plugins/replication-supabase';

banner();
const token = mintToken('user-alice');
const sb = makeSupabaseClient(token);
const db = await makeLocalDb(`push_${RUN}`);
const notes = db.notes;

const id = `push-${RUN}`;

const rep = replicateSupabase({
    replicationIdentifier: `push-${RUN}`,
    collection: notes,
    client: sb,
    tableName: TABLE,
    // waitForLeadership defaults to TRUE. In a browser that is right (one tab
    // replicates). In a one-process Node harness it is just a way to hang.
    waitForLeadership: false,
    live: true,
    pull: { batchSize: 50 },
    push: { batchSize: 50 }
});
// An unhandled replication error is the thing most likely to be mistaken for
// "it just did not sync". Print every one.
rep.error$.subscribe((e) => console.log('!! replication error:', e.message || e));

await rep.awaitInitialReplication();
console.log('initial replication done');

console.log(`local insert  id=${id}`);
await notes.insert({ id, owner_id: 'user-alice', body: 'written locally in RxDB' });

await rep.awaitInSync();
console.log('awaitInSync resolved');

// --- verification over an INDEPENDENT request, not through RxDB -------------
const res = await fetch(
    `http://127.0.0.1:${REST_PORT}/${TABLE}?id=eq.${id}&select=*`,
    { headers: { Authorization: `Bearer ${token}` } }
);
console.log(`postgrest verify HTTP ${res.status}`);
const rows = await res.json();
console.log('row in postgres:', JSON.stringify(rows, null, 2));

console.log(rows.length === 1 && rows[0].body === 'written locally in RxDB'
    ? 'PUSH: PROVEN — the locally-created RxDB document exists as a Postgres row'
    : 'PUSH: NOT PROVEN');

await rep.cancel();
await db.close();
process.exit(0);
