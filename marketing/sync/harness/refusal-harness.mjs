// marketing/sync/harness/refusal-harness.mjs — the node half of the B-432
// validation run (card refusal-holds-before-sync, run 20260906-2; see
// refusal-run.sh for the substrate half and the verdict contract).
//
// THIS IS THE OWED GAP-1 VALIDATION RUN (goal ledger Comebacks): spike 02
// (`refusal-holds-during-window`) re-executed with the PROTOTYPE policy
// replaced by the SHIPPED policy source — the f2-run.sh precedent, wrapper-free
// against the shipped guard. Nothing here re-implements the fix: it imports
// `createCampaignPolicySource` from ../replicas.js and drives the SHIPPED
// `marketing/submit-machine.js` on the SHIPPED `lib/xstate.umd.min.js` in mode
// 'throw', so an undeclared (state,event) pair reds this gate rather than being
// modelled away.
//
// The only things written here are the fetch gate (campaigns answers 503 until
// released) and submit-flow.js's `policyFor` coercion, copied verbatim in
// behaviour — the seam under test.
//
// Seven legs:
//   1  window + KNOWN HIGH code   → override REFUSED            (the fix)
//   2  window + genuinely-unknown → override OFFERED + F2 warn  (decision 166)
//   3  window + KNOWN HIGH, PRE-CARD policy shape → OFFERED     (B-432, demonstrated)
//   L  the error latch: unresolved() true, lastError names HTTP 503 attributably
//   4  ready  + HIGH              → still REFUSED               (normal §8)
//   5  ready  + LOW               → OFFERED                     (no over-refusal)
//   6  READY + a campaign whose CODE arrived first → REFUSED    ← build-fact 3's
//      sub-case, the window a bare readiness latch leaves open. Built for real:
//      a new campaign + code inserted server-side, the CODES replica resynced
//      and the CAMPAIGNS replica deliberately NOT.
//   7  the discriminator, landed: two unverified overrides through the SHIPPED
//      enqueue + SHIPPED push handler, read back off the arbiter —
//      `unverified_code=t, policy_unresolved=t` (replica failure) vs
//      `t, f` (genuinely-unknown campaign), both status='accepted'.
//
// MODES (f2-run.sh's precedent):
//   green          the shipped source. Every assertion must hold.
//   red-preserved  legs 1/4/5/6 run against the PRE-CARD policy shape
//                  (`byId.has(id) ? {...} : null`). The SAME assertions must
//                  then FAIL — proof that they catch the defect class rather
//                  than passing by coincidence.

import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { RxDBMigrationSchemaPlugin } from 'rxdb/plugins/migration-schema';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../.night-crew/qa/spike-supabase/rxdb/spike-env.js';
import {
  marketingCollectionSpec,
  startCodesReplica,
  startOffersReplica,
  startCampaignsReplica,
  createCampaignPolicySource,
  CODES_COLLECTION,
  OFFERS_COLLECTION,
  CAMPAIGNS_COLLECTION,
} from '../replicas.js';
import {
  enqueueAttempt,
  makePushHandler,
  startScanAttemptsReplica,
  scanAttemptsCollectionSpec,
  SCAN_ATTEMPTS_COLLECTION,
  SCAN_ATTEMPTS_SCHEMA,
} from '../push-replication.js';

const require = createRequire(import.meta.url);
const X = require('../../../lib/xstate.umd.min.js');
const { createSubmitMachine } = await import('../../submit-machine.js');

const MODE = process.argv[2] || 'green';
if (!['green', 'red-preserved'].includes(MODE)) {
  console.error('usage: refusal-harness.mjs [green|red-preserved]');
  process.exit(64);
}
const env = (k) => {
  const v = process.env[k];
  if (!v) { console.error(`missing env ${k}`); process.exit(2); }
  return v;
};
const JWT = env('RH_JWT');
const DB_CID = env('RH_DB_CID');
const DEVICE = 'device-a';

const REST = `http://127.0.0.1:${REST_PORT}`;
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (180s) — a leg never finished'), 180_000);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
async function until(fn, ms, what) {
  const deadline = Date.now() + ms;
  for (;;) {
    const v = await fn();
    if (v) return v;
    if (Date.now() > deadline) return null;
    await sleep(100);
  }
}
const sql = (text) => execFileSync(
  'docker',
  ['exec', '-i', DB_CID, 'psql', '-U', 'supabase_admin', '-d', 'postgres', '-v', 'ON_ERROR_STOP=1', '-qtA'],
  { input: text, encoding: 'utf8' },
).trim();

// The committed seed fixtures (supabase/seed.sql — referenced BY VALUE).
const HIGH = { code: 'c0000000-0000-4000-8000-000000000005', campaign: 'a0000000-0000-4000-8000-000000000002' };
const LOW = { code: 'c0000000-0000-4000-8000-000000000001', campaign: 'a0000000-0000-4000-8000-000000000001' };

console.log(`# mode: ${MODE}`);
console.log(`# SCAN_ATTEMPTS_SCHEMA.version = ${SCAN_ATTEMPTS_SCHEMA.version} `
  + `(policy_unresolved declared: ${'policy_unresolved' in SCAN_ATTEMPTS_SCHEMA.properties})`);

// ---------------------------------------------------------------------------
// The device. ajv validation is ON (wrappedValidateAjvStorage) — spike 03's
// error VD2 was "the shipped schema REJECTS policy_unresolved at v0", so the
// fact that leg 7 inserts it at all is itself a measurement.
// ---------------------------------------------------------------------------
let campaignsUp = false;
const gatedFetch = async (url, opts) => (campaignsUp
  ? fetch(url, opts)
  : new Response('harness: campaigns endpoint down', { status: 503 }));

addRxPlugin(RxDBDevModePlugin);
// 🛑 REQUIRED since SCAN_ATTEMPTS_SCHEMA went to version 1 (card
// refusal-holds-before-sync, run 20260906-2): rxdb runs
// `autoMigrate && version !== 0 && await migratePromise()` on every
// collection creation, and without this plugin that call THROWS
// ("You are using a function which must be overwritten by a plugin") —
// addCollections rejects and the harness dies before its first leg. The
// browser gets the same registration in marketing/scan-page.js.
addRxPlugin(RxDBMigrationSchemaPlugin);

const db = await createRxDatabase({
  name: `rh_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections({ ...marketingCollectionSpec(), ...scanAttemptsCollectionSpec() });

const deps = (collection, fetchImpl, replicationIdentifier) => ({
  replicateRxCollection, collection, restUrl: REST, bearer: JWT,
  fetchImpl, stream$: new Subject(), replicationIdentifier,
});
const codesStream$ = new Subject();
const codesRep = startCodesReplica({ ...deps(db[CODES_COLLECTION], fetch, 'rh-codes'), stream$: codesStream$ });
const offersRep = startOffersReplica(deps(db[OFFERS_COLLECTION], fetch, 'rh-offers'));

// 🛑 BUILD-FACT 1's ORDER, reproduced exactly as scan-page.js does it: create
// the source, start the replica, attach the latch — no `await` in between.
// error$ does not replay; an attach one tick later sees nothing.
const src = createCampaignPolicySource(db[CAMPAIGNS_COLLECTION]);
const campaignsRep = startCampaignsReplica(deps(db[CAMPAIGNS_COLLECTION], gatedFetch, 'rh-campaigns'));
src.attach(campaignsRep);

await codesRep.awaitInSync();
await offersRep.awaitInSync();
const codesLanded = (await db[CODES_COLLECTION].find().exec()).length;
console.log(`\n── replica state: codes+offers IN SYNC (${codesLanded} code rows); campaigns erroring (503) ──`);
if (!codesLanded) fail('no code rows landed — the premise "the device knows the code" is unbuilt');

// ---------------------------------------------------------------------------
// The policies under test.
// ---------------------------------------------------------------------------
const shippedPolicy = src.policyFor;                     // THE SHIPPED SOURCE

// The PRE-CARD shape, reconstructed here so B-432 can be demonstrated rather
// than asserted (and so red-preserved has something to be red against). This
// is what `createCampaignPolicySource` returned before this card.
const mapMirror = new Map();
db[CAMPAIGNS_COLLECTION].find().$.subscribe((docs) => {
  mapMirror.clear();
  for (const d of docs) mapMirror.set(d.id, !!d.requires_online);
});
const preCardPolicy = (campaignId) => {
  const flag = mapMirror.get(campaignId);
  return flag === undefined ? null : { requiresOnline: flag };
};

const POLICY_UNDER_TEST = MODE === 'green' ? shippedPolicy : preCardPolicy;

// submit-flow.js's policyFor, verbatim in behaviour — the coercion under test.
function policyFor(CAMPAIGN_POLICY, campaignId) {
  if (!CAMPAIGN_POLICY) return false;
  try {
    const p = CAMPAIGN_POLICY(campaignId);
    return !!(p && p.requiresOnline);
  } catch (e) { return false; }
}

async function fixture(codeId, kind) {
  if (codeId === null) {
    return { kind, codeId: '(unknown)', tokenHash: 'f'.repeat(64), campaignId: null };
  }
  const doc = await db[CODES_COLLECTION].findOne(codeId).exec();
  if (!doc) fail(`code ${codeId} not in the codes replica`);
  return { kind, codeId, tokenHash: doc.token_hash, campaignId: doc.campaign_id };
}

function run(label, fx, policySource) {
  const requiresOnline = policyFor(policySource, fx.campaignId);
  const trips = [];
  const m = createSubmitMachine(X, { canOverride: true }, [], { mode: 'throw', onTrip: (t) => trips.push(t) });
  m.send('SCAN');
  m.send('QR_DECODED', { code: fx.tokenHash });
  m.send('RESOLVED', { kind: fx.kind, requiresOnline });
  if (fx.kind === 'offerReady') m.send('ORDER_OK');
  m.send('CONN_DOWN');
  m.send('SUBMIT');
  const scanAtBlock = m.scan();
  const overrideAvailable = m.flags().overrideAvailable;
  m.send('OVERRIDE_REQUEST');
  return {
    label, requiresOnline, scanAtBlock, overrideAvailable,
    scanAfterRequest: m.scan(), unverifiedWarning: m.flags().unverifiedWarning,
    alive: m.alive(), trips: trips.length,
  };
}

// ---------------------------------------------------------------------------
// LEGS 1–3 — inside the window.
// ---------------------------------------------------------------------------
console.log('\n── the window (campaigns replica erroring, nothing delivered) ──');
const wHigh = run('window + HIGH (under test)', await fixture(HIGH.code, 'offerReady'), POLICY_UNDER_TEST);
const wUnknown = run('window + unknown code    ', await fixture(null, 'unknownCode'), POLICY_UNDER_TEST);
const wPreCard = run('window + HIGH (PRE-CARD) ', await fixture(HIGH.code, 'offerReady'), preCardPolicy);

// ---------------------------------------------------------------------------
// LEG L — the error latch (build-fact 1). error$ does not replay, so the
// source had to subscribe before the replica got going and hold this itself.
// ---------------------------------------------------------------------------
console.log('\n── the error latch ──');
await until(() => src.lastError() !== null, 15_000, 'error latch');
console.log(`  attached=${src.attached()} ready=${src.ready()} unresolved=${src.unresolved()}`);
console.log(`  lastError: ${src.lastError()}`);
const latchAttributable = /503/.test(String(src.lastError() ?? ''));

// ---------------------------------------------------------------------------
// RECOVERY — the same handle, no teardown (spike 01).
// ---------------------------------------------------------------------------
console.log('\n── recovery: campaigns endpoint comes up; the source waits for ready+settle ──');
campaignsUp = true;
try { campaignsRep.reSync(); } catch (e) { /* the retry cycle covers it */ }
const gotReady = await until(() => src.ready(), 20_000, 'readiness');
if (!gotReady) fail('the campaigns replica never reached ready within 20s of recovery');
console.log(`  ready; policy Map holds ${src.size()} campaigns; unresolved=${src.unresolved()}; lastError=${src.lastError()}`);

const rHigh = run('ready + HIGH             ', await fixture(HIGH.code, 'offerReady'), POLICY_UNDER_TEST);
const rLow = run('ready + LOW              ', await fixture(LOW.code, 'offerReady'), POLICY_UNDER_TEST);

// ---------------------------------------------------------------------------
// LEG 6 — THE CODES-ARRIVE-FIRST SUB-CASE (build-fact 3, binding).
//
// The replica is READY. A brand-new campaign is published server-side with a
// code on it; only the CODES replica is nudged. The device now holds a code it
// knows, naming a campaign it has never seen — the window a bare readiness
// latch leaves wide open, and the reason the shipped predicate is not gated on
// readiness.
// ---------------------------------------------------------------------------
console.log('\n── leg 6: codes-arrive-first (replica READY, a new campaign\'s code lands alone) ──');
const NEW_CAMPAIGN = sql('select gen_random_uuid();');
const NEW_CODE = sql('select gen_random_uuid();');
sql(`insert into public.campaigns (id, name, face_value, requires_online)
     values ('${NEW_CAMPAIGN}', 'rh codes-arrive-first', 40, true);`);
sql(`insert into public.codes (id, token_hash, campaign_id, expires_at)
     values ('${NEW_CODE}', 'rh-caf-${NEW_CODE}', '${NEW_CAMPAIGN}', now() + interval '1 day');`);
codesStream$.next('RESYNC');
await codesRep.awaitInSync();
const landedNewCode = await until(async () => !!(await db[CODES_COLLECTION].findOne(NEW_CODE).exec()), 15_000, 'new code');
if (!landedNewCode) fail('the new code never reached the codes replica — leg 6 has no premise');
const campaignsInMap = src.size();
const newCampaignInMap = mapMirror.has(NEW_CAMPAIGN);
console.log(`  new code landed; campaigns Map holds ${campaignsInMap}; the NEW campaign is in it: ${newCampaignInMap}`);
if (newCampaignInMap) {
  console.log('  ⚠ the campaigns replica delivered the new campaign anyway — leg 6 cannot pose its question this run');
}
const cafRun = run('READY + codes-arrive-first', await fixture(NEW_CODE, 'offerReady'), POLICY_UNDER_TEST);

const runs = [wHigh, wUnknown, wPreCard, rHigh, rLow, cafRun];
console.log('\n── the runs ──');
for (const r of runs) {
  console.log(
    `  ${r.label} | requiresOnline=${String(r.requiresOnline).padEnd(5)} | blocked=${r.scanAtBlock} | `
    + `overrideAvailable=${String(r.overrideAvailable).padEnd(5)} | afterOVERRIDE_REQUEST=${r.scanAfterRequest} | `
    + `unverifiedWarning=${String(r.unverifiedWarning).padEnd(5)} | alive=${r.alive} trips=${r.trips}`,
  );
}

// ---------------------------------------------------------------------------
// LEG 7 — the discriminator, LANDED. Shipped enqueueAttempt → shipped push
// handler → the real arbiter, then read back. done_when clause 2, end to end.
// ---------------------------------------------------------------------------
console.log('\n── leg 7: the discriminator lands distinguishable (shipped enqueue + shipped push handler) ──');
const attempts = db[SCAN_ATTEMPTS_COLLECTION];
const HASH_FAIL = 'a'.repeat(64);   // replica-failure override
const HASH_UNKN = 'b'.repeat(64);   // genuinely-unknown-campaign override
await enqueueAttempt(attempts, {
  code_id: HASH_FAIL, device_id: DEVICE, offline_override: true,
  override_by: 'rh-harness', unverified_code: true, policy_unresolved: true,
  pos_order_number: 'RH-1',
});
await enqueueAttempt(attempts, {
  code_id: HASH_UNKN, device_id: DEVICE, offline_override: true,
  override_by: 'rh-harness', unverified_code: true, policy_unresolved: false,
  pos_order_number: 'RH-2',
});
const localFail = await attempts.findOne({ selector: { code_id: HASH_FAIL } }).exec();
const localUnkn = await attempts.findOne({ selector: { code_id: HASH_UNKN } }).exec();
console.log(`  local rows (ajv-validated): policy_unresolved ${localFail?.policy_unresolved} / ${localUnkn?.policy_unresolved}`);

const requestLog = [];
const pushErrors = [];
const pushRep = startScanAttemptsReplica({
  replicateRxCollection,
  collection: attempts,
  pushHandler: makePushHandler({
    restUrl: REST, bearer: JWT, deviceId: DEVICE, fetchImpl: fetch,
    attemptsCollection: attempts, codesCollection: db[CODES_COLLECTION],
    requestLog, winnerWaitMs: 500, winnerPollMs: 100,
  }),
  retryTime: 500,
});
pushRep.error$.subscribe((e) => pushErrors.push(String(e?.message || e).split('\n')[0]));
await until(async () => (await attempts.find({ selector: { landed: true } }).exec()).length === 2, 20_000, 'both landings');
await sleep(300);

const kinds = [...new Set(requestLog.map((r) => r.kind))];
console.log(`  push requests: ${requestLog.length} — kinds: ${kinds.join(', ') || '(none)'}`);
console.log(`  redeem calls on the unverified rows: ${requestLog.filter((r) => r.kind === 'redeem').length} (the F-2 guard must divert BEFORE redeem)`);
if (pushErrors.length) console.log(`  distinct push errors: ${[...new Set(pushErrors)].join(' | ')}`);

let table = '';
try {
  // t/f, the shape the spike's distinguishability table used (PG renders a
  // boolean as 'true'/'false' under ||, which is not that shape).
  table = sql(`select left(token_hash, 8) || '… | ' || status
      || ' | override=' || (case when offline_override then 't' else 'f' end)
      || ' | unverified=' || (case when unverified_code then 't' else 'f' end)
      || ' | policy_unresolved=' || (case when policy_unresolved then 't' else 'f' end)
    from public.scan_attempts where token_hash in ('${HASH_FAIL}','${HASH_UNKN}') order by token_hash;`);
} catch (e) {
  table = `(read-back failed: ${String(e.message || e).split('\n')[0]})`;
}
console.log('  server rows:');
for (const line of table.split('\n')) console.log(`    ${line}`);
const srvFail = table.includes(`${HASH_FAIL.slice(0, 8)}… | accepted | override=t | unverified=t | policy_unresolved=t`);
const srvUnkn = table.includes(`${HASH_UNKN.slice(0, 8)}… | accepted | override=t | unverified=t | policy_unresolved=f`);
const newStatuses = sql("select coalesce(string_agg(distinct status, ','), '(none)') from public.scan_attempts;");
console.log(`  distinct statuses on the arbiter: ${newStatuses}`);

await pushRep.cancel();

// Cleanup (throwaway substrate hygiene; failures here are not a verdict).
try {
  sql(`delete from public.scan_attempts where token_hash in ('${HASH_FAIL}','${HASH_UNKN}');`);
  sql(`delete from public.codes where id = '${NEW_CODE}';`);
  sql(`delete from public.campaigns where id = '${NEW_CAMPAIGN}';`);
} catch (e) { /* reset_bare on the next run covers it */ }

src.stop();
await codesRep.cancel();
await offersRep.cancel();
await campaignsRep.cancel();
await db.close();

// ---------------------------------------------------------------------------
// Assertions — the SAME set in both modes. red-preserved is DEMONSTRATED by
// these failing (exit 1); if they pass there, they do not catch the defect
// class and this harness is not evidence.
// ---------------------------------------------------------------------------
const problems = [];
for (const r of runs) {
  if (!r.alive) problems.push(`${r.label}: the actor died — ${r.trips} undeclared pair(s)`);
  if (r.trips) problems.push(`${r.label}: ${r.trips} undeclared (state,event) pair(s) tripped — the fix left the policy seam`);
  if (r.scanAtBlock !== 'blockedOffline') problems.push(`${r.label}: expected blockedOffline at SUBMIT, got ${r.scanAtBlock}`);
}

// 1 — the fix: a KNOWN high code in the window is REFUSED.
if (wHigh.requiresOnline !== true) problems.push('window+HIGH: the shipped source did not fail closed (requiresOnline=false)');
if (wHigh.overrideAvailable !== false) problems.push('window+HIGH: override OFFERED during the window — B-432 is NOT closed');
if (wHigh.scanAfterRequest !== 'blockedOffline') problems.push(`window+HIGH: OVERRIDE_REQUEST moved the screen to ${wHigh.scanAfterRequest}`);

// 2 — decision 166 survives the fail-closed predicate.
if (wUnknown.requiresOnline !== false) problems.push('window+unknown: decision 166 did not survive (requiresOnline read true)');
if (wUnknown.overrideAvailable !== true) problems.push('window+unknown: F2 override REFUSED — fail-closed over-reached into decision 166\'s ratified path');
if (wUnknown.scanAfterRequest !== 'overrideConfirm') problems.push(`window+unknown: expected the §13 confirmation, got ${wUnknown.scanAfterRequest}`);
if (wUnknown.unverifiedWarning !== true) problems.push('window+unknown: the F2 unverified warning did not arm');

// 3 — B-432 demonstrated beside the fix, in this same harness run.
if (wPreCard.requiresOnline !== false || wPreCard.overrideAvailable !== true) {
  problems.push('window+HIGH(PRE-CARD): the pre-card null→false coercion did NOT fail open — the premise is gone, re-derive');
}

// L — the latch.
if (!src.attached()) problems.push('the policy source never attached to the campaigns replica');
if (!latchAttributable) problems.push('the error latch held nothing naming the HTTP status — build-fact 1\'s attributability is gone');

// 4/5 — readiness lifts the gate both ways, and does not over-refuse.
if (rHigh.requiresOnline !== true || rHigh.overrideAvailable !== false) problems.push('ready+HIGH: the normal §8 refusal did not hold after readiness');
if (rLow.requiresOnline !== false || rLow.overrideAvailable !== true) problems.push('ready+LOW: override refused on an offline-eligible campaign after ready — the fix would kill the offline path');

// 6 — the codes-arrive-first sub-case (build-fact 3, binding).
if (newCampaignInMap) {
  problems.push('leg 6 could not pose its question: the campaigns replica delivered the new campaign before the run');
} else {
  if (cafRun.requiresOnline !== true) problems.push('codes-arrive-first: a READY replica + an unseen campaign did NOT fail closed — a bare readiness latch would leave exactly this open');
  if (cafRun.overrideAvailable !== false) problems.push('codes-arrive-first: override OFFERED on a code whose campaign has never been seen');
}

// 7 — the discriminator, landed and distinguishable.
if (localFail?.policy_unresolved !== true || localUnkn?.policy_unresolved !== false) {
  problems.push('the local rows lost policy_unresolved — the enqueue plumbing or the v1 schema dropped it');
}
if (requestLog.filter((r) => r.kind === 'redeem').length !== 0) {
  problems.push('the F-2 guard called redeem() on an unverified attempt — head-of-line poison class');
}
if (!srvFail) problems.push('the replica-failure override did NOT land accepted|t|t on the arbiter');
if (!srvUnkn) problems.push('the genuinely-unknown-campaign override did NOT land accepted|t|f on the arbiter');
if (/pending/.test(newStatuses) || !/^[a-z,]+$/.test(newStatuses.replace(/\(none\)/, 'none'))) {
  // informational only — the taxonomy check below is the assertion
}
for (const s of newStatuses.split(',').filter(Boolean)) {
  if (!['accepted', 'rejected', 'pending'].includes(s)) {
    problems.push(`a NEW terminal status reached the arbiter: '${s}' — §9/§19 taxonomy broken (the card's PARK line)`);
  }
}

console.log('\n── conclusion ──');
if (problems.length) {
  for (const p of problems) console.error(`  ✗ ${p}`);
  fail(`${problems.length} disagreement(s)`);
}
console.log('  · the refusal HOLDS through the window on the SHIPPED source — known HIGH code, campaigns');
console.log('    undelivered → overrideAvailable=false, OVERRIDE_REQUEST a no-op, even with canOverride=true.');
console.log('  · decision 166 SURVIVES: a code naming no campaign keeps the F2 override + unverified warning.');
console.log('  · B-432 DEMONSTRATED beside it: the pre-card null→false shape offers the override on the same');
console.log('    code in the same window.');
console.log('  · the error latch holds an ATTRIBUTABLE failure (the HTTP status) with no replay available.');
console.log('  · readiness lifts the gate both ways, and the CODES-ARRIVE-FIRST sub-case is still refused —');
console.log('    the window a bare readiness latch would have left open.');
console.log('  · the discriminator lands distinguishable on the real arbiter, both status=accepted, no new');
console.log('    terminal status, and the F-2 guard still diverts before redeem().');
console.log('  · zero undeclared (state,event) pairs across every run (mode: throw).');

clearTimeout(hardTimeout);
process.exit(0);
