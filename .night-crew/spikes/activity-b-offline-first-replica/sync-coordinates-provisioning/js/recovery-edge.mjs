// recovery-edge.mjs — which SHIPPED observable marks a successful POST-ready
// pull cycle, so the B-439 latch can clear on the recovery edge?
//
// Everything under test is shipped code: startCampaignsReplica +
// createCampaignPolicySource (replicas.js), createSyncClock (clock.js — its
// .captures counter increments on every HTTP-200 pull, the §5.1 seam
// scan-page.js already injects). Written here: the gate, the stopwatch, the
// per-phase signal tally. env: DB_CID (for the mid-error row touch).
// argv: <deviceJwt>. exit 0 = at least one signal carries BOTH recovery shapes.

import { execFileSync } from 'node:child_process';
import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';
import {
  startCampaignsReplica, createCampaignPolicySource, marketingCollectionSpec,
} from '../../../../../marketing/sync/replicas.js';
import { createSyncClock } from '../../../../../marketing/sync/clock.js';

const [JWT] = process.argv.slice(2);
if (!JWT) { console.error('usage: recovery-edge.mjs <deviceJwt>'); process.exit(2); }
const DB_CID = process.env.DB_CID;
if (!DB_CID) { console.error('DB_CID env missing'); process.exit(2); }

const REST = `http://127.0.0.1:${REST_PORT}`;
const t0 = Date.now();
const ts = () => `t+${String(Date.now() - t0).padStart(6)}ms`;
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const hardTimeout = setTimeout(() => fail('hard timeout (180s)'), 180_000);

const LOW_CAMPAIGN = 'a0000000-0000-4000-8000-000000000001';
const touchRow = () => execFileSync('docker', [
  'exec', DB_CID, 'psql', '-U', 'supabase_admin', '-d', 'postgres', '-qtA', '-c',
  `update public.campaigns set requires_online = requires_online where id = '${LOW_CAMPAIGN}';`,
]);

// The gate: healthy passthrough ↔ synthetic 503.
let gate503 = false;
const gatedFetch = async (url, opts) => {
  if (gate503) return new Response('spike: campaigns endpoint down', { status: 503 });
  return fetch(url, opts);
};

addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `spike_recovery_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections(marketingCollectionSpec());

const clock = createSyncClock({});
const policySource = createCampaignPolicySource(db.campaigns);
const rep = startCampaignsReplica({
  replicateRxCollection, collection: db.campaigns, restUrl: REST, bearer: JWT,
  fetchImpl: gatedFetch, stream$: new Subject(), clock,
  replicationIdentifier: 'spike-recovery-campaigns',
});
policySource.attach(rep);

// Signal tally, reset per phase.
let tally = null;
const resetTally = () => { tally = { error$: 0, active$: [], remoteEvents$: 0, capturesDelta: 0, capturesBase: clock.captures }; };
resetTally();
rep.error$.subscribe(() => { tally.error$ += 1; console.log(`  ${ts()}  error$ emitted`); });
rep.active$.subscribe((v) => { tally.active$.push(v); console.log(`  ${ts()}  active$ → ${v}`); });
if (rep.remoteEvents$ && rep.remoteEvents$.subscribe) {
  rep.remoteEvents$.subscribe(() => { tally.remoteEvents$ += 1; console.log(`  ${ts()}  remoteEvents$ emitted`); });
}
const snap = (label) => {
  tally.capturesDelta = clock.captures - tally.capturesBase;
  const t = { ...tally, active$: tally.active$.join(',') };
  console.log(`  [${label}] error$=${t['error$']} active$=[${t['active$']}] remoteEvents$=${t.remoteEvents$} captures+=${t.capturesDelta} unresolved()=${policySource.unresolved()} lastError=${JSON.stringify(policySource.lastError())}`);
  return t;
};

// Phase A — healthy → ready.
console.log('── phase A: healthy initial replication → ready ──');
await rep.awaitInitialReplication();
await sleep(400); // the shipped settle tick + margin
if (policySource.unresolved() !== false) fail('not resolved after clean initial replication');
snap('A: ready');

// Phase B — post-ready 503 blip: the B-439 premise, reproduced.
console.log('\n── phase B: post-ready 503 — the latch latches (B-439 premise) ──');
resetTally();
gate503 = true;
rep.reSync();
for (let i = 0; i < 150 && tally.error$ < 1; i++) await sleep(100);
const B = snap('B: erroring');
if (tally.error$ < 1) fail('no error$ emission on a post-ready 503 — the sibling finding does not extend past ready');
if (policySource.unresolved() !== true) fail('unresolved() stayed false during a post-ready error — B-439 mis-modeled (it claims the latch OVER-reports)');

// Phase C1 — recovery WITH docs (a row changed while erroring).
console.log('\n── phase C1: recovery WITH docs ──');
touchRow();
resetTally();
gate503 = false;
rep.reSync();
for (let i = 0; i < 150 && tally.capturesDelta + (clock.captures - tally.capturesBase) < 1; i++) { await sleep(100); tally.capturesDelta = clock.captures - tally.capturesBase; }
await sleep(500);
const C1 = snap('C1: recovered+docs');
if (policySource.unresolved() !== true) fail('unresolved() self-cleared after recovery — then B-439 is already fixed and this spike is measuring a ghost');

// Phase D — error again.
console.log('\n── phase D: second 503 ──');
resetTally();
gate503 = true;
rep.reSync();
for (let i = 0; i < 150 && tally.error$ < 1; i++) await sleep(100);
snap('D: erroring');
if (tally.error$ < 1) fail('no error$ on the second blip');

// Phase C2 — recovery EMPTY (no rows changed — the worst case).
console.log('\n── phase C2: recovery EMPTY (zero new rows) ──');
resetTally();
gate503 = false;
rep.reSync();
for (let i = 0; i < 150 && (clock.captures - tally.capturesBase) < 1; i++) await sleep(100);
await sleep(500);
const C2 = snap('C2: recovered+empty');

// Verdict: per candidate, fires on success in BOTH shapes, and is
// distinguishable from the erroring phase.
console.log('\n── verdict table (candidate: C1 / C2 / distinguishable-from-error) ──');
const candidates = [
  ['clock.captures', C1.capturesDelta >= 1, C2.capturesDelta >= 1, B.capturesDelta === 0],
  ['remoteEvents$', C1.remoteEvents$ >= 1, C2.remoteEvents$ >= 1, B.remoteEvents$ === 0],
  ['active$ transition', C1['active$'].length >= 1, C2['active$'].length >= 1, false /* also transitions while erroring — not distinguishing alone */],
];
let usable = 0;
for (const [name, c1, c2, dist] of candidates) {
  const ok = c1 && c2 && dist;
  usable += ok ? 1 : 0;
  console.log(`  ${name.padEnd(20)} C1=${c1 ? 'fires' : 'silent'}  C2=${c2 ? 'fires' : 'silent'}  distinguishable=${dist}  ${ok ? '← USABLE EDGE' : ''}`);
}
if (usable < 1) {
  console.error('\nno shipped signal marks the successful post-ready cycle in both shapes;');
  console.error('the fix must tap the pull-handler seam replicas.js itself constructs.');
  process.exit(1);
}

await db.close();
clearTimeout(hardTimeout);
process.exit(0);
