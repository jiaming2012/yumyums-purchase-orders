// window-signals.mjs — the B-432 window, observed on the SHIPPED replica handle.
//
// Everything under test is shipped code:
//   * marketing/sync/replicas.js startCampaignsReplica — the campaigns replica
//     exactly as scan-page.js starts it (shipped makePullHandler underneath,
//     checkpoint-only URL, CAMPAIGNS_SELECT)
//   * marketing/sync/replicas.js createCampaignPolicySource — the Map mirror
//     submit-flow.js feeds its policy seam from
//
// The only thing written here is the fetch gate (synthetic 503 until released)
// and the stopwatch. argv: <deviceJwt>. exit 0 all measurements agreed.

import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';
import {
  startCampaignsReplica,
  createCampaignPolicySource,
  marketingCollectionSpec,
} from '../../../../../marketing/sync/replicas.js';

const [JWT] = process.argv.slice(2);
if (!JWT) { console.error('usage: window-signals.mjs <deviceJwt>'); process.exit(2); }

const REST = `http://127.0.0.1:${REST_PORT}`;
const t0 = Date.now();
const ts = () => `t+${String(Date.now() - t0).padStart(5)}ms`;
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const hardTimeout = setTimeout(() => fail('hard timeout (90s)'), 90_000);

// The seeded HIGH campaign — the $40 catering credit, requires_online=true.
const HIGH_CAMPAIGN = 'a0000000-0000-4000-8000-000000000002';

// ---------------------------------------------------------------------------
// The gated fetch: the B-432 window made controllable. Until released, the
// campaigns endpoint answers 503 — the shipped pull handler throws
// "[marketing-sync] pull campaigns answered HTTP 503" on every attempt.
// ---------------------------------------------------------------------------
let passthrough = false;
let syntheticServed = 0;
const gatedFetch = async (url, opts) => {
  if (passthrough) return fetch(url, opts);
  syntheticServed += 1;
  return new Response('spike: campaigns endpoint down', { status: 503 });
};

addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `spike_window_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections(marketingCollectionSpec());

// The policy source attaches BEFORE the replica starts — scan-page order.
const policySource = createCampaignPolicySource(db.campaigns);

const errors = [];
const stream$ = new Subject();
const handle = startCampaignsReplica({
  replicateRxCollection,
  collection: db.campaigns,
  restUrl: REST,
  bearer: JWT,
  fetchImpl: gatedFetch,
  stream$,
  replicationIdentifier: 'spike-window-campaigns',
});

// ---------------------------------------------------------------------------
// Enumerate the handle surface — the set, not a sample (B-216).
// ---------------------------------------------------------------------------
console.log('── the shipped handle surface (QA rxdb; vendor bundle grepped in the .sh) ──');
const SURFACE = ['error$', 'active$', 'remoteEvents$', 'awaitInitialReplication', 'awaitInSync', 'reSync', 'cancel'];
const surface = {};
for (const name of SURFACE) {
  const v = handle[name];
  surface[name] = typeof v === 'function' ? 'function'
    : v && typeof v.subscribe === 'function' ? 'observable'
      : typeof v;
  console.log(`  ${name.padEnd(26)} ${surface[name]}`);
}
if (surface['error$'] !== 'observable') fail('error$ is not an observable on the shipped handle — no erroring signal exists');
if (surface.awaitInitialReplication !== 'function') fail('awaitInitialReplication missing — no readiness signal exists');

handle.error$.subscribe((err) => {
  const inner = err && err.parameters && Array.isArray(err.parameters.errors)
    ? err.parameters.errors.map((e) => String((e && e.message) || e)).join(' | ')
    : '';
  errors.push({ at: Date.now() - t0, msg: String((err && err.message) || err), inner });
  console.log(`  ${ts()}  error$ →  ${errors[errors.length - 1].msg}${inner ? `  [inner: ${inner}]` : ''}`);
});

let readyAt = null;
let sizeAtReady = null;
const readiness = handle.awaitInitialReplication().then(() => {
  readyAt = Date.now() - t0;
  sizeAtReady = policySource.size(); // measured SYNCHRONOUSLY at the resolve tick
});

// ---------------------------------------------------------------------------
// Leg A — erroring: awaitInitialReplication must stay pending while error$
// emits. 6.5s covers the first attempt plus one default-retryTime (5s) cycle.
// ---------------------------------------------------------------------------
console.log('\n── leg A: the campaigns endpoint answers 503 (the B-432 window) ──');
const SENTINEL = Symbol('pending');
const raceA = await Promise.race([readiness.then(() => 'resolved'), sleep(6500).then(() => SENTINEL)]);
console.log(`  ${ts()}  awaitInitialReplication: ${raceA === SENTINEL ? 'still pending' : 'RESOLVED'}; error$ emissions: ${errors.length}; synthetic 503s served: ${syntheticServed}`);
if (raceA !== SENTINEL) fail('awaitInitialReplication resolved against an endpoint that never answered 200 — "ready" would be a lie in the window');
if (errors.length < 1) fail('error$ never emitted while the pull was failing — erroring is indistinguishable from in-flight');
const attributable = errors.some((e) => (e.msg + e.inner).includes('503'));
console.log(`  failure attributable in the emission (contains "503"): ${attributable}`);

// Leg A2 — a LATE subscriber must be characterized: does error$ replay?
const lateSeen = [];
handle.error$.subscribe(() => lateSeen.push(Date.now() - t0));
await sleep(400);
console.log(`  ${ts()}  late error$ subscriber emissions within 400ms: ${lateSeen.length} (replay ${lateSeen.length ? 'YES' : 'no'})`);
const lateReplay = lateSeen.length > 0;

// ---------------------------------------------------------------------------
// Leg B — recovery: release the gate; the SAME handle must reach ready with
// no restart. reSync() is the shipped nudge (scan-page resync()); the default
// 5s retry cycle is the fallback, so the bound is 12s.
// ---------------------------------------------------------------------------
console.log('\n── leg B: endpoint recovers; same handle, no restart ──');
passthrough = true;
try { handle.reSync(); } catch (e) { console.log(`  reSync() threw: ${e.message}`); }
const raceB = await Promise.race([readiness.then(() => 'resolved'), sleep(12_000).then(() => SENTINEL)]);
if (raceB === SENTINEL) fail('awaitInitialReplication did not resolve within 12s of recovery — the fix would need a replica restart, a different card shape');
console.log(`  ${ts()}  awaitInitialReplication RESOLVED at t+${readyAt}ms (same handle, no restart)`);

// Leg B2 — the subscription race: was the policy Map populated AT the resolve
// tick, or only after a settle? Either answer is a finding; the card's
// readiness latch must respect the measured one.
await sleep(150);
const sizeAfterSettle = policySource.size();
console.log(`  policy-source Map size at the resolve tick: ${sizeAtReady}; after 150ms settle: ${sizeAfterSettle}`);
if (sizeAfterSettle !== 2) fail(`expected both seeded campaigns in the policy Map after settle, got ${sizeAfterSettle}`);
const p = policySource.policyFor(HIGH_CAMPAIGN);
if (!p || p.requiresOnline !== true) fail(`policyFor(HIGH) after recovery answered ${JSON.stringify(p)} — expected {requiresOnline:true}`);

// Leg B3 — errors stop once recovered (the states really are disjoint).
const errCountAtReady = errors.length;
await sleep(1000);
if (errors.length !== errCountAtReady) fail(`error$ kept emitting after recovery (${errCountAtReady} → ${errors.length})`);
console.log(`  error$ silent for 1s after recovery (count stable at ${errCountAtReady})`);

// ---------------------------------------------------------------------------
// Conclusion.
// ---------------------------------------------------------------------------
console.log('\n── conclusion ──');
console.log('  · ERRORING is observable: error$ emitted while awaitInitialReplication stayed pending;');
console.log(`    the emission ${attributable ? 'carries' : 'does NOT carry'} the HTTP status (attributability ${attributable ? 'free' : 'needs the card to wrap the handler'}).`);
console.log(`  · error$ ${lateReplay ? 'REPLAYS to late subscribers' : 'does NOT replay to late subscribers'} — ${lateReplay ? 'a latch is free' : 'the card must latch errors itself (subscribe before start, hold the last error)'}.`);
console.log('  · READY is observable: awaitInitialReplication resolved on the SAME handle after recovery — no restart.');
console.log(`  · The readiness→Map race is real and measured: size ${sizeAtReady} at the resolve tick vs ${sizeAfterSettle} after settle —`);
console.log('    the card\'s readiness latch gates on the MAP being fed, not on the bare promise, unless the tick measurement said 2.');
console.log('  · IN-FLIGHT is the remaining state: promise pending + error$ silent. All three disjoint.');

await handle.cancel();
await db.close();
clearTimeout(hardTimeout);
process.exit(0);
