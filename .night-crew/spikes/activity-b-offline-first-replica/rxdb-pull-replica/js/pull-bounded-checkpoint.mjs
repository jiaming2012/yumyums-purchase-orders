// pull-bounded-checkpoint.mjs — the RxDB half of the pull spike. Device-b runs
// a live, pull-only replica of `codes` via replicateRxCollection with a custom
// PostgREST pull handler (the §3 "chosen" shape) + a Realtime nudge (§7.3:
// RESYNC on every SUBSCRIBED and on every postgres_changes frame).
//
// argv: <jwtDeviceA> <jwtDeviceB> <targetId> <inWindowId> <outWindowId>
// exit 0 all legs held; exit 1 a leg failed.
//
// Discipline inherited from proof-pull.js: replication starts ONCE; no
// restart, no manual reSync(); the redemption is written through the wire by
// a DIFFERENT identity (device-a, via POST /rpc/redeem — the committed RPC).

import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { makeSupabaseClient, REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';

const [jwtA, jwtB, TARGET, IN1, OUT1] = process.argv.slice(2);
if (!jwtA || !jwtB || !TARGET || !IN1 || !OUT1) {
  console.error('usage: pull-bounded-checkpoint.mjs <jwtA> <jwtB> <target> <inWindow> <outWindow>');
  process.exit(2);
}
const fail = (msg) => { console.error(`RED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (90s) — convergence never happened'), 90_000);

const REST = `http://127.0.0.1:${REST_PORT}`;
const EPOCH = '1970-01-01T00:00:00+00:00';
const windowBound = () => new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString();

// ---------------------------------------------------------------------------
// device-b's local replica
// ---------------------------------------------------------------------------
addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `spike_pull_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections({
  codes: {
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: { type: 'string', maxLength: 100 },
        token_hash: { type: 'string' },
        campaign_id: { type: 'string' },
        expires_at: { type: 'string' },
        redeemed_at: { type: ['string', 'null'] },
        redeemed_by: { type: ['string', 'null'] },
        updated_at: { type: 'string' },
      },
      required: ['id', 'token_hash', 'expires_at', 'updated_at'],
    },
  },
});

// ---------------------------------------------------------------------------
// custom pull handler — bounded + checkpointed, request log enumerated (B-216)
// ---------------------------------------------------------------------------
const pullLog = [];
const pullStream$ = new Subject();

async function pullHandler(checkpoint, batchSize) {
  const cursor = checkpoint?.updated_at ?? EPOCH;
  const url =
    `${REST}/codes?select=id,token_hash,campaign_id,expires_at,redeemed_at,redeemed_by,updated_at,_deleted` +
    `&updated_at=gt.${encodeURIComponent(cursor)}` +
    `&expires_at=gt.${encodeURIComponent(windowBound())}` +
    `&order=updated_at.asc,id.asc&limit=${batchSize}`;
  pullLog.push({ cursor, url });
  const res = await fetch(url, { headers: { Authorization: `Bearer ${jwtB}` } });
  if (res.status !== 200) throw new Error(`pull HTTP ${res.status}`);
  const rows = await res.json();
  console.log(`  pull#${pullLog.length} cursor=${cursor === EPOCH ? 'EPOCH' : cursor} → ${rows.length} row(s)`);
  return {
    documents: rows,
    checkpoint: rows.length
      ? { updated_at: rows[rows.length - 1].updated_at, id: rows[rows.length - 1].id }
      : (checkpoint ?? { updated_at: EPOCH }),
  };
}

// Realtime nudge (§7.3): RESYNC on postgres_changes AND on every SUBSCRIBED.
const sb = makeSupabaseClient(jwtB);
let subscribed = false;
const channel = sb
  .channel('spike-pull-nudge')
  .on('postgres_changes', { event: '*', schema: 'public', table: 'codes' }, () => {
    console.log('  realtime frame → RESYNC');
    pullStream$.next('RESYNC');
  })
  .subscribe((status, err) => {
    console.log(`  realtime channel: ${status}${err ? ` (${err.message ?? err})` : ''}`);
    if (status === 'SUBSCRIBED') { subscribed = true; pullStream$.next('RESYNC'); }
  });

const rep = replicateRxCollection({
  collection: db.codes,
  replicationIdentifier: `spike-pull-${Date.now()}`,
  live: true,
  waitForLeadership: false,
  pull: { handler: pullHandler, batchSize: 50, stream$: pullStream$.asObservable() },
});
rep.error$.subscribe((e) => console.log('!! replication error:', e.message || e));

await rep.awaitInitialReplication();

// ---------------------------------------------------------------------------
// leg (a) — bounded initial sync, enumerated
// ---------------------------------------------------------------------------
const localIds = (await db.codes.find().exec()).map((d) => d.id).sort();
console.log(`# leg (a) — local replica after initial sync (${localIds.length} docs):`);
for (const id of localIds) console.log(`    ${id}`);
if (!localIds.includes(TARGET)) fail('live target code did not land on initial sync');
if (!localIds.includes(IN1)) fail('in-window expired code (expired-1d) did not land — the window filter is too tight');
if (localIds.includes(OUT1)) fail('OUT-of-window code (expired-5d) landed — the pull is not bounded');
const seedExpiredOld = 'c0000000-0000-4000-8000-000000000003'; // seed fixture, expired 2026-01-01 — far out of window
if (localIds.includes(seedExpiredOld)) fail('seed fixture …0003 (expired 2026-01-01) landed — the pull is not bounded');
console.log('  leg (a) HELD — in-window present, out-of-window absent');

// wait for the realtime channel before burning, else the nudge can be missed
const subDeadline = Date.now() + 20_000;
while (!subscribed && Date.now() < subDeadline) await new Promise((r) => setTimeout(r, 100));
if (!subscribed) fail('realtime channel never reached SUBSCRIBED — the nudge path is dead');

// ---------------------------------------------------------------------------
// leg (c) setup — device-a burns the target through the committed redeem() RPC
// ---------------------------------------------------------------------------
console.log('# leg (c) — device-a burns the target via POST /rpc/redeem');
const burn = await fetch(`${REST}/rpc/redeem`, {
  method: 'POST',
  headers: { Authorization: `Bearer ${jwtA}`, 'Content-Type': 'application/json' },
  body: JSON.stringify({ p_code: TARGET, p_device: 'device-a' }),
});
const burnBody = await burn.json();
console.log(`  /rpc/redeem HTTP ${burn.status} → ${JSON.stringify(burnBody)}`);
const verdict = Array.isArray(burnBody) ? burnBody[0] : burnBody;
if (burn.status !== 200 || !verdict?.ok) fail('the committed redeem() did not accept the live target');

// convergence on the RUNNING replica — no restart, no manual reSync
const t0 = Date.now();
let converged = null;
while (Date.now() - t0 < 30_000) {
  const doc = await db.codes.findOne(TARGET).exec();
  if (doc && doc.redeemed_by === 'device-a' && doc.redeemed_at) { converged = doc; break; }
  await new Promise((r) => setTimeout(r, 250));
}
if (!converged) fail('the redemption never surfaced on device-b\'s running replica (30s)');
console.log(`  converged in ${Date.now() - t0} ms → redeemed_by=${converged.redeemed_by} redeemed_at=${converged.redeemed_at}`);
console.log('  leg (c) HELD — a real redeem() propagated to the second device live');

// ---------------------------------------------------------------------------
// leg (b) — the checkpoint was honored, from the enumerated request log
// ---------------------------------------------------------------------------
console.log(`# leg (b) — pull request log (${pullLog.length} requests):`);
for (const [i, p] of pullLog.entries()) console.log(`    #${i + 1} cursor=${p.cursor}`);
if (pullLog.length < 2) fail('only one pull request — nothing exercised resumption');
if (pullLog[0].cursor !== EPOCH) fail('first pull did not start from the epoch cursor');
const post = pullLog.slice(1).filter((p) => p.cursor !== EPOCH);
if (post.length === 0) fail('no post-initial pull carried a non-epoch updated_at cursor — the pull is not checkpointed');
console.log(`  leg (b) HELD — ${post.length} post-initial request(s) resumed from a real updated_at cursor`);

clearTimeout(hardTimeout);
await rep.cancel();
await sb.removeChannel(channel);
await db.close();
console.log('ALL LEGS HELD');
process.exit(0);
