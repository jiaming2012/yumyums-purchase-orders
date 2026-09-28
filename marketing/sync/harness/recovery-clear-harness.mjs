// marketing/sync/harness/recovery-clear-harness.mjs — the node half of the
// B-439 validation gate (card sync-coordinates-provisioning, run 20260907;
// see recovery-clear-run.sh for the substrate half and the verdict contract).
//
// THIS IS SPIKE 04 (.night-crew/spikes/activity-b-offline-first-replica/
// sync-coordinates-provisioning/js/recovery-edge.mjs) RE-EXECUTED AGAINST THE
// SHIPPED CLEAR — the refusal-run.sh precedent: wrapper-free, the same
// construction the spike used (shipped startCampaignsReplica + shipped
// createCampaignPolicySource + attach, live PostgREST, gated fetch
// 200→503→200), with the verdict inverted to what the card ships. Where the
// spike measured the latch STUCK `unresolved()=true` through TWO full
// recoveries, the shipped clear must now take it back to `false` in BOTH
// recovery shapes:
//
//   C1  recovery WITH docs   (a row touched while erroring)
//   C2  recovery EMPTY       (zero new rows — the shape that defeats any
//                             docs-based signal; only the successful-pull
//                             edge, the one clock.captures witnesses, fires)
//
// …while the latch itself still LATCHES during each error phase (B and D):
// a clear that disarms the latch would fail-open the very reading B-432's
// discriminator records. Both directions asserted.
//
// RED evidence: this harness run against the PRE-fix tree exits 1 at phase C1
// (latch stuck — the spike's measurement reproduced); the log is committed at
// .night-crew/runs/2026-09-07-autonomous/c1-red-recovery-clear.log. The
// UNMODIFIED spike script, run post-fix, exits 1 at ITS phase C1 with
// "unresolved() self-cleared after recovery — then B-439 is already fixed" —
// the spike's own ghost-check confirming the clear (log:
// c1-spike04-rerun-postfix.log).
//
// env: DB_CID (for the mid-error row touch). argv: <deviceJwt>.
// exit 0 = the latch clears in both shapes AND re-latches on each error.

import { execFileSync } from 'node:child_process';
import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../.night-crew/qa/spike-supabase/rxdb/spike-env.js';
import {
  startCampaignsReplica, createCampaignPolicySource, marketingCollectionSpec,
} from '../replicas.js';
import { createSyncClock } from '../clock.js';

const [JWT] = process.argv.slice(2);
if (!JWT) { console.error('usage: recovery-clear-harness.mjs <deviceJwt>'); process.exit(2); }
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

// The gate: healthy passthrough ↔ synthetic 503 (the spike's exact shape).
let gate503 = false;
const gatedFetch = async (url, opts) => {
  if (gate503) return new Response('harness: campaigns endpoint down', { status: 503 });
  return fetch(url, opts);
};

addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `harness_recovery_clear_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections(marketingCollectionSpec());

const clock = createSyncClock({});
const policySource = createCampaignPolicySource(db.campaigns);
const rep = startCampaignsReplica({
  replicateRxCollection, collection: db.campaigns, restUrl: REST, bearer: JWT,
  fetchImpl: gatedFetch, stream$: new Subject(), clock,
  replicationIdentifier: 'harness-recovery-clear-campaigns',
});
policySource.attach(rep);

let errors = 0;
rep.error$.subscribe(() => { errors += 1; console.log(`  ${ts()}  error$ emitted`); });

const snap = (label) => {
  console.log(`  [${label}] captures=${clock.captures} unresolved()=${policySource.unresolved()} lastError=${JSON.stringify(policySource.lastError())}`);
};
const awaitUnresolved = async (want, label) => {
  for (let i = 0; i < 150 && policySource.unresolved() !== want; i++) await sleep(100);
  snap(label);
  return policySource.unresolved() === want;
};

// Phase A — healthy → ready.
console.log('── phase A: healthy initial replication → ready ──');
await rep.awaitInitialReplication();
await sleep(400); // the shipped settle tick + margin
if (policySource.unresolved() !== false) fail('not resolved after clean initial replication');
snap('A: ready');

// Phase B — post-ready 503: the latch must STILL latch (fail-safe direction).
console.log('\n── phase B: post-ready 503 — the latch must still latch ──');
gate503 = true;
rep.reSync();
for (let i = 0; i < 150 && errors < 1; i++) await sleep(100);
if (errors < 1) fail('no error$ emission on a post-ready 503');
if (!(await awaitUnresolved(true, 'B: erroring'))) fail('unresolved() stayed false during a post-ready error — the clear DISARMED the latch (fail-open)');

// Phase C1 — recovery WITH docs: THE SHIPPED CLEAR (spike 04 measured stuck).
console.log('\n── phase C1: recovery WITH docs — the shipped clear ──');
touchRow();
gate503 = false;
rep.reSync();
if (!(await awaitUnresolved(false, 'C1: recovered+docs')))
  fail('B-439 NOT fixed: unresolved() stuck true after a with-docs recovery (the spike-04 measurement, unrepaired)');

// Phase D — second 503: the latch re-latches (the clear is an edge, not a disarm).
console.log('\n── phase D: second 503 — the latch re-latches ──');
const errBefore = errors;
gate503 = true;
rep.reSync();
for (let i = 0; i < 150 && errors <= errBefore; i++) await sleep(100);
if (errors <= errBefore) fail('no error$ on the second blip');
if (!(await awaitUnresolved(true, 'D: erroring'))) fail('unresolved() stayed false during the SECOND error — the clear disarmed re-latching');

// Phase C2 — recovery EMPTY: zero new rows, the worst case for the clear.
console.log('\n── phase C2: recovery EMPTY (zero new rows) ──');
gate503 = false;
rep.reSync();
if (!(await awaitUnresolved(false, 'C2: recovered+empty')))
  fail('B-439 NOT fixed in the recovery-EMPTY shape: no docs arrived and the latch never cleared — the clear is keyed on the wrong edge');

console.log('\nboth recovery shapes clear, both error phases latch — the B-439 clear is the successful-pull edge and nothing else.');
await db.close();
clearTimeout(hardTimeout);
process.exit(0);
