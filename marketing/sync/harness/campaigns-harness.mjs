// marketing/sync/harness/campaigns-harness.mjs — the node half of the
// campaigns-replica gate (card requires-online-replication, run 20260906; see
// campaigns-run.sh for the substrate half and the verdict contract). A SIBLING
// of Cards 2/3/4's harnesses — their landed gates stay byte-identical.
//
// Drives the PRODUCTION marketing/sync modules against the LOCAL
// spike-supabase substrate with the BUILT migrations + seed applied
// (Activity A's two files + this card's 20260906000100/200). What it proves:
//
//   1. OPTIONAL BOUND (owed item 1): the shipped makePullHandler serves
//      `campaigns` with NO windowBound (spike build-fact 1: unconditional
//      `expires_at=gt.` drew HTTP 400 — campaigns has no such column), while
//      the codes pull URL STILL carries its bound — optional means optional,
//      never removed. Both URLs enumerated from the request log (B-216).
//   2. THE REPLICA (the spike-chosen mechanism): startCampaignsReplica —
//      the same replicateRxCollection + makePullHandler + GAP-1 keyset
//      checkpoint the codes replica uses — lands both seeded campaigns with
//      their seeded flags.
//   3. THE REFUSAL ARMS ON REAL DATA (spike 02's four-run matrix, on the
//      PRODUCTION policy source): createCampaignPolicySource feeds the
//      SHIPPED submit-machine on the SHIPPED vendored xstate, mode 'throw' —
//      HIGH (requires_online=true) refuses the override even with
//      canOverride:true; LOW offers it; with NO policy source both are
//      overridable (the negative that makes it a proof). Zero undeclared
//      (state,event) pairs, actor alive — Card 6's 460-pair strictness proof
//      survives the swap.
//   4. THE FLIP RE-DELIVERS (done_when clause 3): a POST-sync campaign
//      DOWNGRADE via a plain `update campaigns set requires_online=false` —
//      updated_at NOT stamped by the writer; the migration's touch trigger
//      must stamp it — is (a) NOT seen with no nudge (a pull replica does not
//      poll), (b) re-delivered on the next RESYNC, and (c) visible to the
//      policy source (the refusal DISARMS — the operator's downgrade reaches
//      the device).
//
// Discipline inherited from Card 2: request logs are enumerated, never
// inferred; the verdict is the exit status, never the prose.

import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../.night-crew/qa/spike-supabase/rxdb/spike-env.js';
import { makePullHandler } from '../pull-replication.js';
import {
  marketingCollectionSpec,
  startCodesReplica,
  startCampaignsReplica,
  createCampaignPolicySource,
  codesWindowBound,
  CODES_COLLECTION,
  CAMPAIGNS_COLLECTION,
  CAMPAIGNS_SELECT,
} from '../replicas.js';

const require = createRequire(import.meta.url);
const X = require('../../../lib/xstate.umd.min.js');
const { createSubmitMachine } = await import('../../submit-machine.js');

const env = (k) => {
  const v = process.env[k];
  if (!v) { console.error(`missing env ${k}`); process.exit(2); }
  return v;
};
const JWT = env('C8_JWT');
const DB_CID = env('C8_DB_CID');

const REST = `http://127.0.0.1:${REST_PORT}`;
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (120s) — a leg never finished'), 120_000);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
async function until(fn, ms, what) {
  const deadline = Date.now() + ms;
  for (;;) {
    const v = await fn();
    if (v) return v;
    if (Date.now() > deadline) fail(`timeout (${ms}ms): ${what}`);
    await sleep(100);
  }
}

const sql = (text) => execFileSync(
  'docker',
  ['exec', '-i', DB_CID, 'psql', '-U', 'supabase_admin', '-d', 'postgres', '-v', 'ON_ERROR_STOP=1', '-qtA'],
  { input: text, encoding: 'utf8' },
).trim();

// The committed seed fixtures (supabase/seed.sql — referenced BY VALUE).
const LOW = { code: 'c0000000-0000-4000-8000-000000000001', campaign: 'a0000000-0000-4000-8000-000000000001' };
const HIGH = { code: 'c0000000-0000-4000-8000-000000000005', campaign: 'a0000000-0000-4000-8000-000000000002' };

// ---------------------------------------------------------------------------
// LEG 1 — the optional bound, both directions, URLs enumerated.
// ---------------------------------------------------------------------------
console.log('\n── leg 1: optional expiry bound — campaigns unbounded, codes STILL bounded ──');
{
  const campaignsLog = [];
  const h = makePullHandler({
    restUrl: REST, table: 'campaigns', bearer: JWT, fetchImpl: fetch,
    select: CAMPAIGNS_SELECT, requestLog: campaignsLog,
    // NO windowBound — the owed optional bound. Pre-card this drew HTTP 400.
  });
  const out = await h(null, 10);
  console.log(`  campaigns URL: ${campaignsLog[0].url}`);
  console.log(`  campaigns pull → ${out.documents.length} row(s)`);
  if (campaignsLog[0].url.includes('expires_at')) fail('the campaigns pull URL still carries an expires_at bound');
  if (out.documents.length < 2) fail(`expected the 2 seeded campaigns, got ${out.documents.length}`);

  const codesLog = [];
  const hc = makePullHandler({
    restUrl: REST, table: 'codes', bearer: JWT, fetchImpl: fetch,
    windowBound: () => codesWindowBound(Date.now), requestLog: codesLog,
  });
  const outc = await hc(null, 10);
  console.log(`  codes URL    : ${codesLog[0].url}`);
  if (!/expires_at=gt\./.test(codesLog[0].url)) fail('the codes pull URL LOST its expiry bound — optional must never mean removed');
  if (!outc.documents.length) fail('the bounded codes pull returned no rows');
  console.log('  → optional means optional: campaigns unbounded, codes bounded, both HTTP 200');
}

// ---------------------------------------------------------------------------
// The device: production collection spec (campaigns included), production
// replicas for codes + campaigns.
// ---------------------------------------------------------------------------
addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `c8_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections(marketingCollectionSpec());

const codesStream$ = new Subject();
const campaignsStream$ = new Subject();
const campaignsLog = [];
const codesRep = startCodesReplica({
  replicateRxCollection, collection: db[CODES_COLLECTION],
  restUrl: REST, bearer: JWT, fetchImpl: fetch, stream$: codesStream$,
  replicationIdentifier: 'c8-codes-pull',
});
const campaignsRep = startCampaignsReplica({
  replicateRxCollection, collection: db[CAMPAIGNS_COLLECTION],
  restUrl: REST, bearer: JWT, fetchImpl: fetch, stream$: campaignsStream$,
  requestLog: campaignsLog,
  replicationIdentifier: 'c8-campaigns-pull',
});
codesRep.error$.subscribe((e) => console.error('  ! codes replica error:', String(e?.message || e)));
campaignsRep.error$.subscribe((e) => console.error('  ! campaigns replica error:', String(e?.message || e)));

console.log('\n── leg 2: the campaigns replica (production startCampaignsReplica) ──');
await codesRep.awaitInSync();
await campaignsRep.awaitInSync();
const readFlag = async (id) => (await db[CAMPAIGNS_COLLECTION].findOne(id).exec())?.requires_online;
{
  const camps = await db[CAMPAIGNS_COLLECTION].find().exec();
  for (const c of camps) console.log(`  campaign ${c.id.slice(-4)} requires_online=${c.requires_online}`);
  if (camps.length !== 2) fail(`expected both seeded campaigns, got ${camps.length}`);
  if (await readFlag(LOW.campaign) !== false) fail('LOW campaign did not land requires_online=false');
  if (await readFlag(HIGH.campaign) !== true) fail('HIGH campaign did not land requires_online=true');
  console.log(`  → both seeded flags landed via ${campaignsLog.length} enumerated pull(s)`);
}

// ---------------------------------------------------------------------------
// LEG 3 — the refusal arms on real data: spike 02's matrix on the PRODUCTION
// policy source. Machine in mode 'throw': an undeclared pair reds the gate.
// ---------------------------------------------------------------------------
console.log('\n── leg 3: the §8 refusal on the PRODUCTION policy source (4 runs, mode throw) ──');
const policySource = createCampaignPolicySource(db[CAMPAIGNS_COLLECTION]);
// 🛑 This guard used to read `policyFor(HIGH.campaign) !== null`. Under the
// fail-closed source (card refusal-holds-before-sync) that is VACUOUS — a
// known campaign id never answers null any more, so it would pass instantly on
// an EMPTY Map and leg 3's LOW run would red on a spurious over-refusal. Wait
// on the Map itself, which is what "the source saw the synced campaigns"
// always meant.
await until(() => policySource.size() === 2, 5000, 'policy source never saw the synced campaigns');

// submit-flow.js's policyFor, verbatim in behavior — the seam the card feeds.
function policyFor(CAMPAIGN_POLICY, campaignId) {
  if (!CAMPAIGN_POLICY) return false;
  try {
    const p = CAMPAIGN_POLICY(campaignId);
    return !!(p && p.requiresOnline);
  } catch (e) { return false; }
}

async function run(label, fixture, source) {
  const doc = await db[CODES_COLLECTION].findOne(fixture.code).exec();
  if (!doc) fail(`${label}: code ${fixture.code} not in the codes replica`);
  const requiresOnline = policyFor(source, doc.campaign_id);
  const trips = [];
  const m = createSubmitMachine(X, { canOverride: true }, [], { mode: 'throw', onTrip: (t) => trips.push(t) });
  m.send('SCAN');
  m.send('QR_DECODED', { code: doc.token_hash });
  m.send('RESOLVED', { kind: 'offerReady', requiresOnline });
  m.send('ORDER_OK');
  m.send('CONN_DOWN');
  m.send('SUBMIT');
  const overrideAvailable = m.flags().overrideAvailable;
  m.send('OVERRIDE_REQUEST');
  const after = m.scan();
  return { label, requiresOnline, overrideAvailable, after, alive: m.alive(), trips: trips.length };
}

const runs = [
  await run('replica + HIGH', HIGH, policySource.policyFor),
  await run('replica + LOW ', LOW, policySource.policyFor),
  await run('none    + HIGH', HIGH, null),
  await run('none    + LOW ', LOW, null),
];
for (const r of runs) {
  console.log(`  ${r.label} | requiresOnline=${String(r.requiresOnline).padEnd(5)} | overrideAvailable=${String(r.overrideAvailable).padEnd(5)} | afterOVERRIDE_REQUEST=${r.after} | alive=${r.alive} trips=${r.trips}`);
}
{
  const [rHigh, rLow, nHigh, nLow] = runs;
  const problems = [];
  for (const r of runs) {
    if (!r.alive || r.trips) problems.push(`${r.label}: actor alive=${r.alive}, trips=${r.trips} — a new undeclared pair`);
  }
  if (rHigh.requiresOnline !== true || rHigh.overrideAvailable !== false || rHigh.after !== 'blockedOffline') {
    problems.push('replica + HIGH: the §8 refusal did not arm on the replicated flag');
  }
  if (rLow.requiresOnline !== false || rLow.overrideAvailable !== true || rLow.after !== 'overrideConfirm') {
    problems.push('replica + LOW: the offline-eligible campaign was over-refused');
  }
  if (nHigh.overrideAvailable !== true || nLow.overrideAvailable !== true) {
    problems.push('no-policy negative: expected BOTH overridable (the honest unknown→false default)');
  }
  if (problems.length) { for (const p of problems) console.error(`  ✗ ${p}`); fail(`${problems.length} disagreement(s) in leg 3`); }
  console.log('  → armed on real data, not over-refusing, provably dead without a source, zero new pairs');
}

// ---------------------------------------------------------------------------
// LEG 4 — the DOWNGRADE flip: plain UPDATE (writer does NOT stamp updated_at —
// the migration's touch trigger must), no polling, re-delivery on RESYNC,
// policy source follows.
// ---------------------------------------------------------------------------
console.log('\n── leg 4: post-sync campaign downgrade — plain UPDATE, trigger-stamped, RESYNC re-delivers ──');
const tsBefore = sql(`select updated_at::text from public.campaigns where id = '${HIGH.campaign}';`);
sql(`update public.campaigns set requires_online = false where id = '${HIGH.campaign}';`);
const tsAfter = sql(`select updated_at::text from public.campaigns where id = '${HIGH.campaign}';`);
console.log(`  updated_at: ${tsBefore} → ${tsAfter} (moved=${tsBefore !== tsAfter})`);
if (tsBefore === tsAfter) fail('the touch trigger did NOT stamp updated_at on a plain UPDATE — the flip is invisible to every checkpointed replica (spike build-fact 2)');

console.log('  (2s with NO nudge — a pull replica must not poll)');
await sleep(2000);
if (await readFlag(HIGH.campaign) !== true) fail('the replica advanced without a nudge — it polls, which burns the radio all shift');
console.log(`  no nudge → replica still reads true (no polling), pulls so far: ${campaignsLog.length}`);

campaignsStream$.next('RESYNC');
await campaignsRep.awaitInSync();
await until(async () => (await readFlag(HIGH.campaign)) === false, 5000, 'the RESYNC did not re-deliver the downgrade');
console.log(`  RESYNC → replica reads false (re-delivered), pulls so far: ${campaignsLog.length}`);
await until(() => {
  const p = policySource.policyFor(HIGH.campaign);
  return p && p.requiresOnline === false;
}, 5000, 'the policy source never followed the downgrade');
console.log('  → the operator\'s downgrade reaches the device policy on the next RESYNC — the refusal disarms');

// Restore the seeded state (throwaway substrate hygiene; not a verdict).
try { sql(`update public.campaigns set requires_online = true where id = '${HIGH.campaign}';`); } catch (e) { /* reset_bare next run */ }

policySource.stop();
await codesRep.cancel();
await campaignsRep.cancel();
await db.close();
clearTimeout(hardTimeout);
console.log('\nall legs held');
process.exit(0);
