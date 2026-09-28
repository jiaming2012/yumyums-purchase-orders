// marketing/sync/harness/harness.mjs — the node half of the card's standalone
// gate (see run.sh for the substrate half and the verdict contract).
//
// GREEN mode drives the PRODUCTION modules (../pull-replication.js,
// ../replicas.js) — dynamically imported so the red modes, which predate those
// files in git history, never touch them. Device-b runs BOTH live replicas;
// device-a burns a code through the committed redeem() RPC over the wire.
//
// RED modes are deliberately defective inline probes (the spike's naive shape,
// NOT the production code) that must FAIL their assertions:
//   red-gap1   — bare `updated_at=gt.<ts>` cursor, batchSize 2, against the
//                5-row same-updated_at tie group → rows silently missed (GAP-1)
//   red-window — no expires_at bound → out-of-window rows land
//
// Discipline inherited from the spike: replication starts ONCE; no restart,
// no manual reSync(); request logs are enumerated (B-216), never inferred.

import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { makeSupabaseClient, REST_PORT } from '../../../.night-crew/qa/spike-supabase/rxdb/spike-env.js';

const MODE = process.argv[2] || 'green';
const env = (k) => {
  const v = process.env[k];
  if (!v) { console.error(`missing env ${k}`); process.exit(2); }
  return v;
};
const JWT_A = env('C2_JWT_A');
const JWT_B = env('C2_JWT_B');
const IN1 = env('C2_IN1');
const OUT1 = env('C2_OUT1');
const TARGET = env('C2_TARGET');
const TARGET_HASH = env('C2_TARGET_HASH');
const OFFER = env('C2_OFFER');
const OFFER_HASH = env('C2_OFFER_HASH');
const GAP_IDS = env('C2_GAP_IDS').split(',');

const REST = `http://127.0.0.1:${REST_PORT}`;
const EPOCH_TS = '1970-01-01T00:00:00+00:00';
const SELECT = 'id,token_hash,campaign_id,expires_at,redeemed_at,redeemed_by,updated_at,_deleted';

// seed.sql fixture contract (fixed UUIDs; supabase/seed.sql)
const FIX = (n) => `c0000000-0000-4000-8000-00000000000${n}`;
const SEED_LIVE = [FIX(1), FIX(2), FIX(4), FIX(5)]; // expire 2028 (0004 redeemed)
const SEED_OUT = FIX(3);                            // expired 2026-01-01 — out of window
const SEED_REDEEMED_HASH =
  'a939afc9a3040327594b0f3c1d3db90a317f93188c114bac807ffdc64eb09097'; // fixture 0004

const fail = (msg) => { console.error(`RED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (150s) — a leg never finished'), 150_000);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const sortedIds = async (col) => (await col.find().exec()).map((d) => d.id).sort();
const eqSets = (a, b) => a.length === b.length && a.every((v, i) => v === b[i]);

addRxPlugin(RxDBDevModePlugin);
async function makeDb(collections) {
  const db = await createRxDatabase({
    name: `c2_${MODE.replace(/-/g, '_')}_${Date.now()}`,
    storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
  });
  await db.addCollections(collections);
  return db;
}

// ═══════════════════════════════════════════════════════════════════════════
// RED PROBES — the defective "before" state, inline on purpose.
// ═══════════════════════════════════════════════════════════════════════════

// The spike's naive handler shape: bare gt cursor, checkpoint carries ONLY
// updated_at. `bounded` toggles the §5.3 window filter.
function naiveHandler({ bounded, log }) {
  return async function pull(checkpoint, batchSize) {
    const cursor = checkpoint?.updated_at ?? EPOCH_TS;
    const windowIso = new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString();
    const url =
      `${REST}/codes?select=${encodeURIComponent(SELECT)}` +
      `&updated_at=gt.${encodeURIComponent(cursor)}` +
      (bounded ? `&expires_at=gt.${encodeURIComponent(windowIso)}` : '') +
      `&order=updated_at.asc,id.asc&limit=${batchSize}`;
    log.push({ cursor, url });
    const res = await fetch(url, { headers: { Authorization: `Bearer ${JWT_B}` } });
    if (res.status !== 200) throw new Error(`pull HTTP ${res.status}`);
    const rows = await res.json();
    console.log(`  pull#${log.length} cursor=${cursor === EPOCH_TS ? 'EPOCH' : cursor} → ${rows.length} row(s)`);
    return {
      documents: rows,
      checkpoint: rows.length
        ? { updated_at: rows[rows.length - 1].updated_at }
        : (checkpoint ?? { updated_at: EPOCH_TS }),
    };
  };
}

const NAIVE_SCHEMA = {
  version: 0,
  primaryKey: 'id',
  type: 'object',
  properties: {
    id: { type: 'string', maxLength: 100 },
    token_hash: { type: 'string', maxLength: 128 },
    campaign_id: { type: 'string', maxLength: 100 },
    expires_at: { type: 'string' },
    redeemed_at: { type: ['string', 'null'] },
    redeemed_by: { type: ['string', 'null'] },
    updated_at: { type: 'string' },
  },
  required: ['id', 'token_hash', 'expires_at', 'updated_at'],
};

async function runNaive({ bounded, batchSize }) {
  const db = await makeDb({ codes: { schema: NAIVE_SCHEMA } });
  const log = [];
  const rep = replicateRxCollection({
    collection: db.codes,
    replicationIdentifier: `c2-${MODE}-${Date.now()}`,
    live: false,
    waitForLeadership: false,
    pull: { handler: naiveHandler({ bounded, log }), batchSize },
  });
  rep.error$.subscribe((e) => fail(`replication error: ${e.message || e}`));
  await rep.awaitInitialReplication();
  return { db, log };
}

if (MODE === 'red-gap1') {
  console.log('# red-gap1 — naive gt-only cursor, batchSize 2, same-updated_at tie group of 5');
  const { db } = await runNaive({ bounded: true, batchSize: 2 });
  const local = await sortedIds(db.codes);
  const missing = GAP_IDS.filter((id) => !local.includes(id));
  if (missing.length > 0) {
    console.error(`  the tie group is ${GAP_IDS.length} rows; the replica holds ${GAP_IDS.length - missing.length}.`);
    fail(`GAP-1 demonstrated — the naive gt cursor SILENTLY MISSED ${missing.length} same-updated_at row(s) at a batch boundary: ${missing.join(', ')}`);
  }
  console.log('  all 5 tie-group rows landed — the naive cursor did NOT miss (probe unexpectedly green)');
  clearTimeout(hardTimeout);
  process.exit(0);
}

if (MODE === 'red-window') {
  console.log('# red-window — pull with NO expires_at bound (batchSize 50)');
  const { db } = await runNaive({ bounded: false, batchSize: 50 });
  const local = await sortedIds(db.codes);
  const landed = [OUT1, SEED_OUT].filter((id) => local.includes(id));
  if (landed.length > 0) {
    fail(`window defect demonstrated — out-of-window row(s) LANDED in the replica: ${landed.join(', ')} (an unbounded pull passes the positive legs and ships dead rows to the truck)`);
  }
  console.log('  out-of-window rows absent — the unbounded probe did NOT land them (probe unexpectedly green)');
  clearTimeout(hardTimeout);
  process.exit(0);
}

// ═══════════════════════════════════════════════════════════════════════════
// GREEN — the production modules, every done_when clause.
// ═══════════════════════════════════════════════════════════════════════════

if (MODE !== 'green') fail(`unknown mode ${MODE}`);

const { startCodesReplica, startOffersReplica, resolveOffers, marketingCollectionSpec } =
  await import('../replicas.js');
const { EPOCH_CHECKPOINT, wireRealtimeResync } = await import('../pull-replication.js');

const db = await makeDb(marketingCollectionSpec());
const codesLog = [];
const offersLog = [];
const codesStream$ = new Subject();
const offersStream$ = new Subject();

const commonDeps = {
  replicateRxCollection,
  restUrl: REST,
  bearer: JWT_B,
  fetchImpl: fetch,
};

// batchSize 2 on codes: FORCES a batch boundary inside the 5-row tie group —
// the GAP-1 validation run this card owes the goal ledger.
const repCodes = startCodesReplica({
  ...commonDeps, collection: db.codes, stream$: codesStream$.asObservable(),
  batchSize: 2, requestLog: codesLog,
});
const repOffers = startOffersReplica({
  ...commonDeps, collection: db.offers, stream$: offersStream$.asObservable(),
  batchSize: 50, requestLog: offersLog,
});
repCodes.error$.subscribe((e) => console.log('!! codes replication error:', e.message || e));
repOffers.error$.subscribe((e) => console.log('!! offers replication error:', e.message || e));

await repCodes.awaitInitialReplication();
await repOffers.awaitInitialReplication();

// ---------------------------------------------------------------------------
// leg 1 — the two bounds, enumerated both ways (done_when: out-of-window rows
// provably do NOT land; §5.3 vs offers-window differential on IN1)
// ---------------------------------------------------------------------------
const codesIds = await sortedIds(db.codes);
const offersIds = await sortedIds(db.offers);
const wantCodes = [...SEED_LIVE, IN1, TARGET, OFFER, ...GAP_IDS].sort();
const wantOffers = [...SEED_LIVE, TARGET, OFFER, ...GAP_IDS].sort();
console.log(`# leg 1 — codes replica after initial sync (${codesIds.length} docs):`);
for (const id of codesIds) console.log(`    ${id}`);
if (!eqSets(codesIds, wantCodes)) fail(`codes replica set mismatch — want [${wantCodes}] got [${codesIds}]`);
console.log(`# leg 1 — offers replica after initial sync (${offersIds.length} docs):`);
for (const id of offersIds) console.log(`    ${id}`);
if (!eqSets(offersIds, wantOffers)) fail(`offers replica set mismatch — want [${wantOffers}] got [${offersIds}]`);
if (codesIds.includes(OUT1) || codesIds.includes(SEED_OUT)) fail('out-of-window row in codes replica');
if (offersIds.includes(IN1)) fail('expired row in OFFERS replica — the two windows did not differ');
console.log('  leg 1 HELD — §5.3 window on codes (IN1 present, out-of-window absent both ways),');
console.log('               offers bounded to live rows (IN1 absent there — the bounds differ per contract)');

// ---------------------------------------------------------------------------
// leg 2 — GAP-1: the same-updated_at batch boundary, walked and enumerated
// ---------------------------------------------------------------------------
const gapDocs = await db.codes.find({ selector: { id: { $in: GAP_IDS } } }).exec();
if (gapDocs.length !== GAP_IDS.length) fail(`tie group incomplete: ${gapDocs.length}/${GAP_IDS.length} landed`);
const tieTs = [...new Set(gapDocs.map((d) => d.updated_at))];
if (tieTs.length !== 1) fail(`tie premise broken — group carries ${tieTs.length} distinct updated_at values`);
const lastGapId = [...GAP_IDS].sort()[GAP_IDS.length - 1];
const boundaryReqs = codesLog.filter(
  (r) => r.checkpoint.updated_at === tieTs[0] && GAP_IDS.includes(r.checkpoint.id) && r.checkpoint.id !== lastGapId
);
console.log(`# leg 2 — GAP-1 boundary. tie updated_at=${tieTs[0]}; ${boundaryReqs.length} request(s) resumed MID-GROUP:`);
for (const r of boundaryReqs) console.log(`    cursor=(${r.checkpoint.updated_at}, ${r.checkpoint.id})`);
if (boundaryReqs.length === 0) fail('no pull request resumed from inside the tie group — the boundary was not exercised');
console.log('  leg 2 HELD — a batch boundary fell inside the tie group, the keyset checkpoint walked it, all 5 landed');

// ---------------------------------------------------------------------------
// leg 3 — checkpoint discipline from the enumerated request log (B-216)
// ---------------------------------------------------------------------------
if (codesLog.length < 2) fail('only one codes pull request — nothing exercised resumption');
if (codesLog[0].checkpoint.updated_at !== EPOCH_CHECKPOINT.updated_at) fail('first pull did not start from the epoch checkpoint');
const preSubscribeCount = codesLog.length;

// §7.3 wiring: ONE channel on public.codes fans RESYNC into BOTH replicas —
// refetch on every SUBSCRIBED (incl. re-SUBSCRIBED), frames are nudges only.
const sb = makeSupabaseClient(JWT_B);
let subscribed = false;
const channel = wireRealtimeResync({
  realtimeClient: sb,
  table: 'codes',
  emitResync: () => { codesStream$.next('RESYNC'); offersStream$.next('RESYNC'); },
  channelName: 'c2-marketing-nudge',
  onStatus: (status, err) => {
    console.log(`  realtime channel: ${status}${err ? ` (${err.message ?? err})` : ''}`);
    if (status === 'SUBSCRIBED') subscribed = true;
  },
});
const subDeadline = Date.now() + 20_000;
while (!subscribed && Date.now() < subDeadline) await sleep(100);
if (!subscribed) fail('realtime channel never reached SUBSCRIBED — the nudge path is dead');

// the SUBSCRIBED refetch (§7.3) must itself appear in the request log,
// resuming from the real checkpoint (never re-reading the world)
const refetchDeadline = Date.now() + 10_000;
while (codesLog.length === preSubscribeCount && Date.now() < refetchDeadline) await sleep(100);
if (codesLog.length === preSubscribeCount) fail('SUBSCRIBED did not trigger a refetch pull (§7.3)');
const refetch = codesLog[preSubscribeCount];
if (refetch.checkpoint.updated_at === EPOCH_CHECKPOINT.updated_at) fail('the SUBSCRIBED refetch re-read from epoch — not checkpointed');
console.log(`# leg 3 — checkpoint discipline (${codesLog.length} codes requests so far):`);
console.log(`    #1 epoch; SUBSCRIBED refetch resumed from (${refetch.checkpoint.updated_at}, ${refetch.checkpoint.id})`);
console.log('  leg 3 HELD — epoch first, every resume carries the keyset cursor, SUBSCRIBED refetches');

// ---------------------------------------------------------------------------
// leg 4 — the done_when core: device-a burns via the committed redeem();
// device-b's RUNNING replicas converge, no restart, no manual reSync
// ---------------------------------------------------------------------------
console.log('# leg 4 — device-a burns the target via POST /rpc/redeem');
const burn = await fetch(`${REST}/rpc/redeem`, {
  method: 'POST',
  headers: { Authorization: `Bearer ${JWT_A}`, 'Content-Type': 'application/json' },
  body: JSON.stringify({ p_code: TARGET, p_device: 'device-a' }),
});
const burnBody = await burn.json();
console.log(`  /rpc/redeem HTTP ${burn.status} → ${JSON.stringify(burnBody)}`);
const verdict = Array.isArray(burnBody) ? burnBody[0] : burnBody;
if (burn.status !== 200 || !verdict?.ok) fail('the committed redeem() did not accept the live target');

const t0 = Date.now();
let convergedCodes = 0, convergedOffers = 0;
while (Date.now() - t0 < 30_000) {
  if (!convergedCodes) {
    const d = await db.codes.findOne(TARGET).exec();
    if (d && d.redeemed_by === 'device-a' && d.redeemed_at) convergedCodes = Date.now() - t0;
  }
  if (!convergedOffers) {
    const d = await db.offers.findOne(TARGET).exec();
    if (d && d.redeemed_by === 'device-a' && d.redeemed_at) convergedOffers = Date.now() - t0;
  }
  if (convergedCodes && convergedOffers) break;
  await sleep(100);
}
if (!convergedCodes) fail('the redemption never surfaced on device-b\'s running CODES replica (30s)');
if (!convergedOffers) fail('the redemption never surfaced on device-b\'s running OFFERS replica (30s)');
const finalCp = codesLog[codesLog.length - 1].checkpoint;
if (finalCp.updated_at === EPOCH_CHECKPOINT.updated_at) fail('the convergence pull re-read from epoch');
console.log(`  converged: codes ${convergedCodes} ms, offers ${convergedOffers} ms after redeem (no restart, no manual reSync)`);
console.log('  leg 4 HELD — a real redeem() propagated to the second device\'s running replicas live');

// ---------------------------------------------------------------------------
// leg 5 — OFFLINE: replication cancelled, channel removed, radio "off";
// resolution answers from the local replica alone (done_when: a synced
// customer's full offer list resolves offline)
// ---------------------------------------------------------------------------
await repCodes.cancel();
await repOffers.cancel();
await sb.removeChannel(channel);
console.log('# leg 5 — offline (both replications cancelled, realtime channel removed)');

const custOffers = await resolveOffers(db.offers, OFFER_HASH);
if (custOffers.length !== 1 || custOffers[0].code_id !== OFFER)
  fail(`synced customer's offer list wrong offline — want [${OFFER}] got ${JSON.stringify(custOffers)}`);
const burned = await resolveOffers(db.offers, TARGET_HASH);
if (burned.length !== 0) fail('the freshly burned code still resolves as an offer offline');
const seedRedeemed = await resolveOffers(db.offers, SEED_REDEEMED_HASH);
if (seedRedeemed.length !== 0) fail('a redeemed seed code resolves as an offer');
const unknown = await resolveOffers(db.offers, 'c2-hash-never-synced');
if (unknown.length !== 0) fail('an unknown hash resolved offers — the embedded-offer fallback signal is broken');
const in1Doc = await db.codes.findOne(IN1).exec();
if (!in1Doc) fail('recently-expired code invisible offline — the scanner would say "unknown" instead of "expired"');
console.log(`  offers(${OFFER_HASH.slice(0, 24)}…) → [${custOffers[0].code_id}] (expires ${custOffers[0].expires_at})`);
console.log('  burned hash → []; redeemed seed hash → []; unknown hash → [] (fallback signal)');
console.log('  leg 5 HELD — the synced customer\'s full offer list resolves with the radio off');

// ---------------------------------------------------------------------------
// the enumerated request logs (B-216)
// ---------------------------------------------------------------------------
console.log(`# request log — codes (${codesLog.length} requests):`);
for (const [i, r] of codesLog.entries())
  console.log(`    #${i + 1} cursor=(${r.checkpoint.updated_at === EPOCH_CHECKPOINT.updated_at ? 'EPOCH' : r.checkpoint.updated_at}, ${r.checkpoint.id.slice(0, 8)}…)`);
console.log(`# request log — offers (${offersLog.length} requests):`);
for (const [i, r] of offersLog.entries())
  console.log(`    #${i + 1} cursor=(${r.checkpoint.updated_at === EPOCH_CHECKPOINT.updated_at ? 'EPOCH' : r.checkpoint.updated_at}, ${r.checkpoint.id.slice(0, 8)}…)`);
console.log(`# sample pull URL (codes #1): ${codesLog[0].url}`);

clearTimeout(hardTimeout);
await db.close();
console.log('ALL LEGS HELD');
process.exit(0);
