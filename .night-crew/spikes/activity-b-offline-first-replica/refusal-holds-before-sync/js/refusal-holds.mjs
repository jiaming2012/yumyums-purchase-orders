// refusal-holds.mjs — the B-432 fix shape, proven at the seam against the
// shipped machine, with the fail-open demonstrated beside it.
//
// Everything under test is shipped code:
//   * marketing/sync/replicas.js — startCodesReplica / startOffersReplica /
//     startCampaignsReplica / createCampaignPolicySource (the real mirror)
//   * marketing/submit-machine.js on lib/xstate.umd.min.js, mode 'throw' — an
//     undeclared (state,event) pair reds this spike instead of being modelled
//     away
//
// The only things written here are the fetch gate (campaigns 503 until
// released), the READINESS LATCH, and the PROTOTYPE fail-closed policy — the
// function shape the card would make createCampaignPolicySource/submit-flow
// carry:
//
//     campaignId == null            → null            (genuinely-unknown code —
//                                                      decision 166's ratified
//                                                      unknown→false survives)
//     campaignId != null, NOT ready → {requiresOnline: true}   ← THE FIX
//     campaignId != null, ready     → the shipped Map answer (null if absent)
//
// argv: <deviceJwt>. exit 0 all five runs agreed.

import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { createRequire } from 'node:module';
import { REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';
import {
  startCodesReplica,
  startOffersReplica,
  startCampaignsReplica,
  createCampaignPolicySource,
  marketingCollectionSpec,
  CODES_COLLECTION,
  OFFERS_COLLECTION,
} from '../../../../../marketing/sync/replicas.js';

const require = createRequire(import.meta.url);
const X = require('../../../../../lib/xstate.umd.min.js');
const { createSubmitMachine } = await import('../../../../../marketing/submit-machine.js');

const [JWT] = process.argv.slice(2);
if (!JWT) { console.error('usage: refusal-holds.mjs <deviceJwt>'); process.exit(2); }

const REST = `http://127.0.0.1:${REST_PORT}`;
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const hardTimeout = setTimeout(() => fail('hard timeout (120s)'), 120_000);

// The seeded fixtures (supabase/seed.sql). HIGH is the $40 catering credit —
// requires_online=true, the campaign the §8 refusal exists for.
const HIGH = { code: 'c0000000-0000-4000-8000-000000000005', campaign: 'a0000000-0000-4000-8000-000000000002' };
const LOW  = { code: 'c0000000-0000-4000-8000-000000000001', campaign: 'a0000000-0000-4000-8000-000000000001' };

// ---------------------------------------------------------------------------
// The device: codes + offers replicas pulled for real (the done_when's
// premise — the device KNOWS the code); campaigns behind the 503 gate (the
// B-432 window, controllable).
// ---------------------------------------------------------------------------
let campaignsUp = false;
const gatedFetch = async (url, opts) => (campaignsUp
  ? fetch(url, opts)
  : new Response('spike: campaigns endpoint down', { status: 503 }));

addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `spike_refusal_holds_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections(marketingCollectionSpec());

const deps = (collection, fetchImpl, replicationIdentifier) => ({
  replicateRxCollection, collection, restUrl: REST, bearer: JWT,
  fetchImpl, stream$: new Subject(), replicationIdentifier,
});
const codesRep = startCodesReplica(deps(db[CODES_COLLECTION], fetch, 'spike-rh-codes'));
const offersRep = startOffersReplica(deps(db[OFFERS_COLLECTION], fetch, 'spike-rh-offers'));
const campaignsRep = startCampaignsReplica(deps(db.campaigns, gatedFetch, 'spike-rh-campaigns'));

await codesRep.awaitInSync();
await offersRep.awaitInSync();
const codesLanded = (await db[CODES_COLLECTION].find().exec()).length;
console.log(`── replica state: codes+offers IN SYNC (${codesLanded} code rows); campaigns replica erroring (503) ──`);
if (!codesLanded) fail('no code rows landed — the premise "the device knows the code" is unbuilt');

// ---------------------------------------------------------------------------
// The policy sources.
// ---------------------------------------------------------------------------
const src = createCampaignPolicySource(db.campaigns); // SHIPPED mirror

// The readiness latch (spike 01's finding: gate on the MAP being fed, not the
// bare promise — hence the settle).
let campaignsReady = false;
const readiness = campaignsRep.awaitInitialReplication().then(async () => {
  await sleep(150);
  campaignsReady = true;
});

// THE PROTOTYPE — the card's fail-closed shape.
const prototypePolicy = (campaignId) => {
  if (campaignId == null) return null;                 // genuinely-unknown code (166)
  if (!campaignsReady) return { requiresOnline: true }; // the window: fail CLOSED
  return src.policyFor(campaignId);                     // ready: replica truth
};

// TODAY'S SHIPPED semantics: the Map mirror with no readiness gate.
const shippedPolicy = (campaignId) => src.policyFor(campaignId);

// submit-flow.js policyFor, verbatim in behaviour — the coercion under test.
function policyFor(CAMPAIGN_POLICY, campaignId) {
  if (!CAMPAIGN_POLICY) return false;
  try {
    const p = CAMPAIGN_POLICY(campaignId);
    return !!(p && p.requiresOnline);
  } catch (e) { return false; }
}

// ---------------------------------------------------------------------------
// One run of the shipped machine. kind 'offerReady' drives the sibling
// spike's exact sequence; kind 'unknownCode' drives the F2 path (no ORDER_OK —
// the UI never sends it there).
// ---------------------------------------------------------------------------
async function run(label, { kind, codeId, tokenHash, campaignId }, policySource) {
  const requiresOnline = policyFor(policySource, campaignId);
  const trips = [];
  const effects = [];
  const m = createSubmitMachine(X, { canOverride: true }, effects,
    { mode: 'throw', onTrip: (t) => trips.push(t) });

  m.send('SCAN');
  m.send('QR_DECODED', { code: tokenHash });
  m.send('RESOLVED', { kind, requiresOnline });
  if (kind === 'offerReady') m.send('ORDER_OK');
  m.send('CONN_DOWN');
  m.send('SUBMIT');
  const scanAtBlock = m.scan();
  const overrideAvailable = m.flags().overrideAvailable;
  m.send('OVERRIDE_REQUEST');
  const scanAfterRequest = m.scan();
  const unverifiedWarning = m.flags().unverifiedWarning;

  return {
    label, codeId, requiresOnline, scanAtBlock, overrideAvailable,
    scanAfterRequest, unverifiedWarning, alive: m.alive(), trips: trips.length,
  };
}

async function fixture(codeId, kind) {
  if (codeId === null) {
    // the genuinely-unknown code: a hash in NO replica, campaign unresolvable
    return { kind, codeId: '(unknown)', tokenHash: 'f'.repeat(64), campaignId: null };
  }
  const doc = await db[CODES_COLLECTION].findOne(codeId).exec();
  if (!doc) fail(`code ${codeId} not in the replica`);
  return { kind, codeId, tokenHash: doc.token_hash, campaignId: doc.campaign_id };
}

console.log('\n── the window (campaigns replica erroring, nothing delivered) ──');
const wHigh    = await run('proto  + window + HIGH   ', await fixture(HIGH.code, 'offerReady'), prototypePolicy);
const wUnknown = await run('proto  + window + unknown', await fixture(null, 'unknownCode'), prototypePolicy);
const wShipped = await run('SHIPPED+ window + HIGH   ', await fixture(HIGH.code, 'offerReady'), shippedPolicy);

console.log('\n── recovery: campaigns endpoint comes up; latch waits for ready+settle ──');
campaignsUp = true;
try { campaignsRep.reSync(); } catch (e) { /* retry cycle covers it */ }
await Promise.race([readiness, sleep(15_000)]);
if (!campaignsReady) fail('campaigns replica never reached ready within 15s of recovery');
console.log(`  ready; policy Map holds ${src.size()} campaigns`);

const rHigh = await run('proto  + ready  + HIGH   ', await fixture(HIGH.code, 'offerReady'), prototypePolicy);
const rLow  = await run('proto  + ready  + LOW    ', await fixture(LOW.code, 'offerReady'), prototypePolicy);

const runs = [wHigh, wUnknown, wShipped, rHigh, rLow];
console.log('\n── the five runs ──');
for (const r of runs) {
  console.log(
    `  ${r.label} | requiresOnline=${String(r.requiresOnline).padEnd(5)} | blocked=${r.scanAtBlock} | `
    + `overrideAvailable=${String(r.overrideAvailable).padEnd(5)} | afterOVERRIDE_REQUEST=${r.scanAfterRequest} | `
    + `unverifiedWarning=${String(r.unverifiedWarning).padEnd(5)} | alive=${r.alive} trips=${r.trips}`,
  );
}

// ---------------------------------------------------------------------------
// Assertions.
// ---------------------------------------------------------------------------
const problems = [];
for (const r of runs) {
  if (!r.alive) problems.push(`${r.label}: the actor died — ${r.trips} undeclared pair(s)`);
  if (r.trips) problems.push(`${r.label}: ${r.trips} undeclared (state,event) pair(s) tripped`);
  if (r.scanAtBlock !== 'blockedOffline') problems.push(`${r.label}: expected blockedOffline at SUBMIT, got ${r.scanAtBlock}`);
}

// 1 — the fix: a known HIGH code in the window is REFUSED.
if (wHigh.requiresOnline !== true) problems.push('proto+window+HIGH: fail-closed did not read requiresOnline=true');
if (wHigh.overrideAvailable !== false) problems.push('proto+window+HIGH: override OFFERED during the window — B-432 survives the fix shape');
if (wHigh.scanAfterRequest !== 'blockedOffline') problems.push(`proto+window+HIGH: OVERRIDE_REQUEST moved the screen to ${wHigh.scanAfterRequest}`);

// 2 — F2 survives: the genuinely-unknown code keeps its decided affordance.
if (wUnknown.requiresOnline !== false) problems.push('proto+window+unknown: decision 166 default did not survive (requiresOnline read true)');
if (wUnknown.overrideAvailable !== true) problems.push('proto+window+unknown: F2 override REFUSED — fail-closed over-reached into decision 166\'s ratified path');
if (wUnknown.scanAfterRequest !== 'overrideConfirm') problems.push(`proto+window+unknown: expected the §13 confirmation, got ${wUnknown.scanAfterRequest}`);
if (wUnknown.unverifiedWarning !== true) problems.push('proto+window+unknown: the F2 "neither offer nor prior use verifiable" warning did not arm');

// 3 — the negative: TODAY's shipped semantics fail open in the same window.
if (wShipped.requiresOnline !== false) problems.push('SHIPPED+window+HIGH: expected the null→false coercion (the B-432 mechanism)');
if (wShipped.overrideAvailable !== true) problems.push('SHIPPED+window+HIGH: expected the override OFFERED — B-432 did not reproduce; re-examine the premise');

// 4/5 — readiness lifts the gate both ways.
if (rHigh.requiresOnline !== true || rHigh.overrideAvailable !== false) problems.push('proto+ready+HIGH: the normal §8 refusal did not hold after readiness');
if (rLow.requiresOnline !== false || rLow.overrideAvailable !== true) problems.push('proto+ready+LOW: override refused on an offline-eligible campaign after ready — the fix would kill the offline path');
if (rLow.scanAfterRequest !== 'overrideConfirm') problems.push(`proto+ready+LOW: expected the §13 confirmation, got ${rLow.scanAfterRequest}`);

console.log('\n── conclusion ──');
if (problems.length) {
  for (const p of problems) console.error(`  ✗ ${p}`);
  fail(`${problems.length} disagreement(s)`);
}
console.log('  · the refusal HOLDS through the window: known HIGH code, campaigns undelivered →');
console.log('    overrideAvailable=false, OVERRIDE_REQUEST a no-op — even with canOverride=true.');
console.log('  · decision 166 SURVIVES: the genuinely-unknown code keeps the F2 override + unverified warning.');
console.log('  · B-432 DEMONSTRATED beside it: the shipped null→false coercion offers the override on the');
console.log('    same code in the same window.');
console.log('  · readiness lifts the gate both ways: HIGH stays refused, LOW recovers its override.');
console.log('  · zero undeclared (state,event) pairs across all five runs (mode: throw) — the fix is');
console.log('    expressible ENTIRELY at the policy seam; Card 6\'s 460-pair strictness proof is untouched.');

await codesRep.cancel();
await offersRep.cancel();
await campaignsRep.cancel();
await db.close();
clearTimeout(hardTimeout);
process.exit(0);
