// proof-pull.js — PULL ONLY. Write directly in Postgres, observe RxDB converge
// WITHOUT restarting the client.
//
// "Without a client restart" is the whole assertion. A pull that only works
// because the process was restarted (or because someone called reSync() by
// hand) proves the initial-sync path and nothing else — and initial sync is not
// what an offline-first PWA needs. So this script:
//   * starts replication ONCE, up top,
//   * never calls reSync(), never cancels, never re-creates the collection,
//   * and writes to Postgres over a raw fetch that RxDB knows nothing about.
//
// FAILURE THIS CATCHES: the Realtime subscription silently not being live. W1
// proved (sharp edge, proof R3) that a phx_join can reply {"status":"ok"} while
// the postgres_changes subscription has actually FAILED, with the real error
// arriving later on a separate `system` frame. A client that only checks the
// join reply believes it is subscribed and then simply never receives anything.
// This script would hang on that, and the timeout below turns that hang into a
// visible, attributable failure rather than a green run.
import { makeSupabaseClient, makeLocalDb, mintToken, TABLE, RUN, REST_PORT, banner }
    from './spike-env.js';
import { replicateSupabase } from 'rxdb/plugins/replication-supabase';
import { firstValueFrom, filter, timeout } from 'rxjs';

banner();
const token = mintToken('user-alice');
const sb = makeSupabaseClient(token);
const db = await makeLocalDb(`pull_${RUN}`);
const notes = db.notes;

const AUTH = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };
const restUrl = (p) => `http://127.0.0.1:${REST_PORT}/${p}`;

const rep = replicateSupabase({
    replicationIdentifier: `pull-${RUN}`,
    collection: notes,
    client: sb,
    tableName: TABLE,
    waitForLeadership: false,
    live: true,
    pull: { batchSize: 50 },
    push: { batchSize: 50 }
});
rep.error$.subscribe((e) => console.log('!! replication error:', e.message || e));

await rep.awaitInitialReplication();
console.log('initial replication done; local doc count =', await notes.count().exec());
console.log('--- from here on the client is NEVER restarted and reSync() is NEVER called ---');

// ---------------------------------------------------------------------------
// PULL CASE 1 — a row that has never existed locally (remote INSERT)
// ---------------------------------------------------------------------------
const insertId = `pull-ins-${RUN}`;
const t0 = Date.now();
const insRes = await fetch(restUrl(TABLE), {
    method: 'POST',
    headers: { ...AUTH, Prefer: 'return=representation' },
    body: JSON.stringify({ id: insertId, owner_id: 'user-alice', body: 'born in Postgres' })
});
console.log(`postgrest INSERT HTTP ${insRes.status}`);

const arrivedDoc = await firstValueFrom(
    notes.findOne(insertId).$.pipe(filter((d) => !!d), timeout(30000))
);
console.log(`PULL/insert converged in ${Date.now() - t0} ms ->`,
    JSON.stringify({ id: arrivedDoc.id, body: arrivedDoc.body }));

// ---------------------------------------------------------------------------
// PULL CASE 2 — a row that DOES exist locally, changed remotely (remote UPDATE)
//
// Separate from case 1 on purpose: an insert can arrive through a full re-read,
// whereas an update to a document the client already holds is the case that
// actually exercises the checkpoint + conflict path. A pull implementation can
// pass case 1 and fail case 2.
// ---------------------------------------------------------------------------
const t1 = Date.now();
const updRes = await fetch(restUrl(`${TABLE}?id=eq.${insertId}`), {
    method: 'PATCH',
    headers: { ...AUTH, Prefer: 'return=representation' },
    body: JSON.stringify({ body: 'edited in Postgres, never touched locally' })
});
console.log(`postgrest UPDATE HTTP ${updRes.status}`);

const updatedDoc = await firstValueFrom(
    notes.findOne(insertId).$.pipe(
        filter((d) => !!d && d.body === 'edited in Postgres, never touched locally'),
        timeout(30000)
    )
);
console.log(`PULL/update converged in ${Date.now() - t1} ms ->`,
    JSON.stringify({ id: updatedDoc.id, body: updatedDoc.body }));

// ---------------------------------------------------------------------------
// PULL CASE 3 — soft delete. RxDB replication is soft-delete only; a hard
// DELETE is invisible to the pull handler. Setting _deleted=true is the only
// way a removal reaches an offline replica.
// ---------------------------------------------------------------------------
const t2 = Date.now();
const delRes = await fetch(restUrl(`${TABLE}?id=eq.${insertId}`), {
    method: 'PATCH', headers: AUTH, body: JSON.stringify({ _deleted: true })
});
console.log(`postgrest SOFT-DELETE HTTP ${delRes.status}`);

const goneDoc = await firstValueFrom(
    notes.findOne(insertId).$.pipe(filter((d) => d === null), timeout(30000))
);
console.log(`PULL/soft-delete converged in ${Date.now() - t2} ms -> findOne returns ${goneDoc}`);

console.log('PULL: PROVEN — insert, update and soft-delete made in Postgres all reached '
    + 'the running RxDB client with no restart and no manual reSync');

await rep.cancel();
await db.close();
process.exit(0);
