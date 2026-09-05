// refusal-arms.mjs — the §8 refusal driven by a REAL replicated flag.
//
// Everything under test is shipped code:
//   * marketing/sync/pull-replication.js  — the pull mechanism (spike 01 chose
//     the campaigns-replica mechanism; this is that mechanism, minus the
//     expiry bound the module must learn to make optional)
//   * marketing/submit-machine.js on lib/xstate.umd.min.js — loaded exactly the
//     way tests/machine/run-conformance.mjs loads them, in mode 'throw' so an
//     undeclared (state,event) pair reds this spike instead of being modelled
//     away
//
// The only thing written here is the POLICY LOOKUP — the function the card will
// hand to submit-flow.js's setCampaignPolicy(). It reads the campaigns replica
// and nothing else; there is no literal anywhere in the refusal path.
//
// argv: <deviceJwt>   exit 0 all four runs agreed; exit 1 one did not.

import { createRequire } from 'node:module';
import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';
import {
  makePullHandler,
  startPullReplica,
  keysetPredicate,
} from '../../../../../marketing/sync/pull-replication.js';
import {
  MARKETING_REPLICA_SCHEMA,
  marketingCollectionSpec,
  codesWindowBound,
  CODES_COLLECTION,
} from '../../../../../marketing/sync/replicas.js';

const require = createRequire(import.meta.url);
const X = require('../../../../../lib/xstate.umd.min.js');
const { createSubmitMachine } = await import('../../../../../marketing/submit-machine.js');

const [JWT] = process.argv.slice(2);
if (!JWT) { console.error('usage: refusal-arms.mjs <deviceJwt>'); process.exit(2); }

const REST = `http://127.0.0.1:${REST_PORT}`;
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (120s)'), 120_000);

// The seeded fixtures (supabase/seed.sql). HIGH is the $40 catering credit —
// requires_online=true, the campaign the §8 refusal exists for.
const HIGH = { code: 'c0000000-0000-4000-8000-000000000005', campaign: 'a0000000-0000-4000-8000-000000000002' };
const LOW  = { code: 'c0000000-0000-4000-8000-000000000001', campaign: 'a0000000-0000-4000-8000-000000000001' };

// ---------------------------------------------------------------------------
// The device: a codes replica (shipped, unchanged) + a campaigns replica (the
// mechanism spike 01 chose).
// ---------------------------------------------------------------------------
addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `spike_refusal_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections({
  ...marketingCollectionSpec(),
  campaigns: {
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: { type: 'string', maxLength: 100 },
        requires_online: { type: 'boolean' },
        updated_at: { type: 'string' },
      },
      required: ['id', 'requires_online', 'updated_at'],
    },
  },
});
console.log(`#   codes schema under test: shipped MARKETING_REPLICA_SCHEMA v${MARKETING_REPLICA_SCHEMA.version}`);

const codesStream$ = new Subject();
const codesRep = startPullReplica({
  replicateRxCollection,
  collection: db[CODES_COLLECTION],
  replicationIdentifier: 'spike-refusal-codes',
  pullHandler: makePullHandler({
    restUrl: REST, table: 'codes',
    windowBound: () => codesWindowBound(Date.now),
    bearer: JWT, fetchImpl: fetch,
  }),
  stream$: codesStream$,
});

const campaignsStream$ = new Subject();
const campaignsRep = startPullReplica({
  replicateRxCollection,
  collection: db.campaigns,
  replicationIdentifier: 'spike-refusal-campaigns',
  pullHandler: (checkpoint, batchSize) => {
    const url =
      `${REST}/campaigns?select=${encodeURIComponent('id,requires_online,updated_at')}` +
      `&${keysetPredicate(checkpoint)}&order=updated_at.asc,id.asc&limit=${batchSize}`;
    return fetch(url, { headers: { Authorization: `Bearer ${JWT}`, Accept: 'application/json' } })
      .then(async (res) => {
        if (res.status !== 200) throw new Error(`campaigns pull HTTP ${res.status}`);
        const rows = await res.json();
        const last = rows[rows.length - 1];
        return {
          documents: rows,
          checkpoint: rows.length ? { updated_at: last.updated_at, id: last.id } : checkpoint,
        };
      });
  },
  stream$: campaignsStream$,
});

await codesRep.awaitInSync();
await campaignsRep.awaitInSync();

const landedCodes = (await db[CODES_COLLECTION].find().exec()).length;
const landedCampaigns = await db.campaigns.find().exec();
console.log(`\n── replica state (enumerated) ──`);
console.log(`  codes rows: ${landedCodes}`);
for (const c of landedCampaigns) console.log(`  campaign ${c.id.slice(-4)} requires_online=${c.requires_online}`);
if (landedCampaigns.length !== 2) fail(`expected both seeded campaigns in the replica, got ${landedCampaigns.length}`);

// ---------------------------------------------------------------------------
// THE POLICY LOOKUP — the function the card hands to setCampaignPolicy().
// Reads the campaigns replica; no network, no literal. Returns the shape
// submit-flow.js's policyFor() already expects: {requiresOnline}.
//
// Unknown campaign → null, and submit-flow's policyFor coerces that to false —
// the ratified unknown→false default (decision 166) survives this card for the
// cases that are GENUINELY unknown (a code not in the replica), which is the
// point: the card removes "unknown" for replicated campaigns, it does not
// change what unknown means.
// ---------------------------------------------------------------------------
const campaignsById = new Map(landedCampaigns.map((c) => [c.id, c]));
const replicaPolicy = (campaignId) => {
  const c = campaignsById.get(campaignId);
  return c ? { requiresOnline: !!c.requires_online } : null;
};
const noPolicy = null; // today's shipped state: CAMPAIGN_POLICY never set

// submit-flow.js:117-123, verbatim in behaviour — the seam the card feeds.
function policyFor(CAMPAIGN_POLICY, campaignId) {
  if (!CAMPAIGN_POLICY) return false;
  try {
    const p = CAMPAIGN_POLICY(campaignId);
    return !!(p && p.requiresOnline);
  } catch (e) { return false; }
}

// ---------------------------------------------------------------------------
// One run of the shipped machine, driven exactly as conformance seq 10 drives
// it — except `requiresOnline` comes from the replica, not from a literal.
// ---------------------------------------------------------------------------
async function run(label, fixture, policySource) {
  const doc = await db[CODES_COLLECTION].findOne(fixture.code).exec();
  if (!doc) return { label, error: `code ${fixture.code} not in the replica` };
  const requiresOnline = policyFor(policySource, doc.campaign_id);

  const trips = [];
  const effects = [];
  const m = createSubmitMachine(X, { canOverride: true }, effects,
    { mode: 'throw', onTrip: (t) => trips.push(t) });

  m.send('SCAN');
  m.send('QR_DECODED', { code: doc.token_hash });
  m.send('RESOLVED', { kind: 'offerReady', requiresOnline });
  m.send('ORDER_OK');
  m.send('CONN_DOWN');
  m.send('SUBMIT');
  const scanAtBlock = m.scan();
  const overrideAvailable = m.flags().overrideAvailable;
  m.send('OVERRIDE_REQUEST');
  const scanAfterRequest = m.scan();

  return {
    label,
    campaign: doc.campaign_id.slice(-4),
    requiresOnline,
    scanAtBlock,
    overrideAvailable,
    scanAfterRequest,
    alive: m.alive(),
    trips: trips.length,
  };
}

const runs = [
  await run('replica + HIGH', HIGH, replicaPolicy),
  await run('replica + LOW ', LOW,  replicaPolicy),
  await run('none    + HIGH', HIGH, noPolicy),
  await run('none    + LOW ', LOW,  noPolicy),
];

console.log('\n── the four runs ──');
for (const r of runs) {
  if (r.error) fail(`${r.label}: ${r.error}`);
  console.log(
    `  ${r.label} | campaign ${r.campaign} | requiresOnline=${String(r.requiresOnline).padEnd(5)} | `
    + `blocked=${r.scanAtBlock} | overrideAvailable=${String(r.overrideAvailable).padEnd(5)} | `
    + `afterOVERRIDE_REQUEST=${r.scanAfterRequest} | alive=${r.alive} trips=${r.trips}`,
  );
}

// ---------------------------------------------------------------------------
// Assertions.
// ---------------------------------------------------------------------------
const [rHigh, rLow, nHigh, nLow] = runs;
const problems = [];

for (const r of runs) {
  if (!r.alive) problems.push(`${r.label}: the actor died — ${r.trips} undeclared pair(s)`);
  if (r.trips) problems.push(`${r.label}: ${r.trips} undeclared (state,event) pair(s) tripped`);
  if (r.scanAtBlock !== 'blockedOffline') problems.push(`${r.label}: expected blockedOffline at SUBMIT, got ${r.scanAtBlock}`);
}

// armed: the real flag reaches the machine and refuses
if (rHigh.requiresOnline !== true) problems.push('replica + HIGH: the replicated flag did not read true');
if (rHigh.overrideAvailable !== false) problems.push('replica + HIGH: override was OFFERED on a requires_online campaign (§8 violated)');
if (rHigh.scanAfterRequest !== 'blockedOffline') problems.push(`replica + HIGH: OVERRIDE_REQUEST moved the screen to ${rHigh.scanAfterRequest} — a manager forced a high-value code offline`);

// and the same mechanism must NOT over-refuse the offline-eligible campaign
if (rLow.requiresOnline !== false) problems.push('replica + LOW: the replicated flag did not read false');
if (rLow.overrideAvailable !== true) problems.push('replica + LOW: override was refused on an offline-eligible campaign — the card would kill the offline path');
if (rLow.scanAfterRequest !== 'overrideConfirm') problems.push(`replica + LOW: expected the §13 confirmation, got ${rLow.scanAfterRequest}`);

// the negative: today, with no policy source, BOTH are overridable
if (nHigh.overrideAvailable !== true || nLow.overrideAvailable !== true) {
  problems.push('policy-source-none: expected BOTH codes overridable (the shipped unknown→false default) — '
    + `got HIGH=${nHigh.overrideAvailable} LOW=${nLow.overrideAvailable}`);
}

console.log('\n── conclusion ──');
if (problems.length) {
  for (const p of problems) console.error(`  ✗ ${p}`);
  fail(`${problems.length} disagreement(s)`);
}
console.log('  · the §8 refusal ARMS on a flag that a real replica pulled off the built schema:');
console.log('      HIGH (requires_online=true)  → overrideAvailable=false, OVERRIDE_REQUEST is a no-op');
console.log('      LOW  (requires_online=false) → overrideAvailable=true,  OVERRIDE_REQUEST → overrideConfirm (§13)');
console.log('  · and it is PROVABLY DEAD without the flag: with no policy source — today\'s shipped');
console.log('    state — the $40 catering-credit code is overridable offline exactly like the $2 one.');
console.log('  · the machine stayed alive with ZERO undeclared pairs across all four runs (mode: throw),');
console.log('    so feeding requiresOnline from a replica introduces no new (state,event) surface.');

await codesRep.cancel();
await campaignsRep.cancel();
await db.close();
clearTimeout(hardTimeout);
process.exit(0);
