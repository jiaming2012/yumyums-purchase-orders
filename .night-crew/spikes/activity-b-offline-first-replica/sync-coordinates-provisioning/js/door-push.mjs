// door-push.mjs — the shipped push replica through the real HQ door, under the
// identity constraint the door imposes (device_id must equal the substituted
// token's sub = the session user's id).
//
// Everything under test is shipped code:
//   * marketing/sync/push-replication.js — enqueueAttempt, makePushHandler,
//     startScanAttemptsReplica (the never-yet-wired page surface)
//   * marketing/sync/replicas.js — startCodesReplica (the winner-data source
//     the push handler reads)
//   * backend: the running server's ProxyHandler (token substitution)
//
// argv: <hqOrigin> <cookie> <sub> <freshCodeA> <freshCodeB>
// exit 0 all measurements agreed.

import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { RxDBMigrationSchemaPlugin } from 'rxdb/plugins/migration-schema';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import {
  startCodesReplica, marketingCollectionSpec,
} from '../../../../../marketing/sync/replicas.js';
import {
  scanAttemptsCollectionSpec, enqueueAttempt, makePushHandler, startScanAttemptsReplica,
} from '../../../../../marketing/sync/push-replication.js';

const [ORIGIN, COOKIE, SUB, FRESH_A, FRESH_B] = process.argv.slice(2);
if (!ORIGIN || !COOKIE || !SUB || !FRESH_A || !FRESH_B) {
  console.error('usage: door-push.mjs <hqOrigin> <cookie> <sub> <freshCodeA> <freshCodeB>');
  process.exit(2);
}

const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (120s)'), 120_000);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const DOOR = `${ORIGIN}/sync/rest`;
const GARBAGE = 'inert-at-the-door';
const cookieFetch = (url, opts = {}) => fetch(url, {
  ...opts,
  headers: { ...(opts.headers || {}), Cookie: `hq_session=${COOKIE}` },
});

addRxPlugin(RxDBDevModePlugin);
// scan_attempts is schema v1 with a device migration (the discriminator card's
// version bump) — the page loads this plugin for the same reason (scan-page.js).
addRxPlugin(RxDBMigrationSchemaPlugin);

async function makeDb(name) {
  const db = await createRxDatabase({
    name,
    storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
  });
  await db.addCollections({ ...marketingCollectionSpec(), ...scanAttemptsCollectionSpec() });
  const codesRep = startCodesReplica({
    replicateRxCollection, collection: db.codes, restUrl: DOOR, bearer: GARBAGE,
    fetchImpl: cookieFetch, stream$: new Subject(),
    replicationIdentifier: `spike-push-codes-${name}`,
  });
  await codesRep.awaitInitialReplication();
  return db;
}

// ---------------------------------------------------------------------------
// Leg (b): deviceId = mint sub — the SHIPPED live replica drains through the door.
// ---------------------------------------------------------------------------
console.log('── leg (b): deviceId = sub, shipped startScanAttemptsReplica, live drain ──');
{
  const db = await makeDb(`spike_push_ok_${Date.now()}`);
  if (!(await db.codes.findOne(FRESH_A).exec())) fail(`fresh code A ${FRESH_A} did not arrive on the codes replica`);

  const { doc } = await enqueueAttempt(db.scan_attempts, { code_id: FRESH_A, device_id: SUB });
  const requestLog = [];
  const handler = makePushHandler({
    restUrl: DOOR, bearer: GARBAGE, deviceId: SUB, fetchImpl: cookieFetch,
    attemptsCollection: db.scan_attempts, codesCollection: db.codes,
    requestLog, winnerWaitMs: 2000,
  });
  const rep = startScanAttemptsReplica({
    replicateRxCollection, collection: db.scan_attempts, pushHandler: handler,
  });
  const errors = [];
  rep.error$.subscribe((e) => errors.push(String((e && e.message) || e)));

  let after = null;
  for (let i = 0; i < 100; i++) {
    after = (await db.scan_attempts.findOne(doc.id).exec()).toJSON();
    if (after.status !== 'pending') break;
    await sleep(200);
  }
  console.log(`  request log: ${requestLog.map((r) => r.kind).join(' → ') || '(empty)'}`);
  console.log(`  attempt: status=${after.status} landed=${after.landed} burn_ok=${after.burn_ok}`);
  if (errors.length) console.log(`  error$ (unexpected): ${errors.join(' | ')}`);
  if (after.status !== 'accepted' || !after.landed) fail('the legit attempt did not resolve accepted+landed through the door');
  if (errors.length) fail('error$ emitted on the legit drain');
  await rep.cancel();
  await db.close();
  console.log('  → the RPC redeem call and the landing insert both compose with the prefix strip;');
  console.log('    the burn records the SESSION USER as the device.');
}

// ---------------------------------------------------------------------------
// Leg (c): deviceId = rogue — the refusal, enumerated, and what it costs.
// ---------------------------------------------------------------------------
console.log('\n── leg (c): deviceId = rogue-device — the RLS refusal, measured ──');
{
  const db = await makeDb(`spike_push_rogue_${Date.now()}`);
  if (!(await db.codes.findOne(FRESH_B).exec())) fail(`fresh code B ${FRESH_B} did not arrive on the codes replica`);

  const { doc } = await enqueueAttempt(db.scan_attempts, { code_id: FRESH_B, device_id: 'rogue-device' });
  const requestLog = [];
  const handler = makePushHandler({
    restUrl: DOOR, bearer: GARBAGE, deviceId: 'rogue-device', fetchImpl: cookieFetch,
    attemptsCollection: db.scan_attempts, codesCollection: db.codes,
    requestLog, winnerWaitMs: 2000,
  });

  let threw = null;
  try {
    await handler([{ newDocumentState: doc.toJSON() }]);
  } catch (e) {
    threw = String((e && e.message) || e);
  }
  const after = (await db.scan_attempts.findOne(doc.id).exec()).toJSON();
  console.log(`  request log: ${requestLog.map((r) => r.kind).join(' → ') || '(empty)'}`);
  console.log(`  handler threw: ${threw ? `YES — ${threw.slice(0, 120)}` : 'no'}`);
  console.log(`  local attempt after: status=${after.status} landed=${after.landed}`);
  if (!threw) {
    if (after.landed) fail('🛑 the rogue-device attempt LANDED — RLS is not binding through the proxy (security finding)');
    fail('the rogue drain neither landed nor threw — an unmeasured third state');
  }
  if (after.landed) fail('🛑 rogue attempt marked landed despite the throw');
  console.log('  → the wrong deviceId is the F-2 throw-retry poison class: the handler throws,');
  console.log('    RxDB would retry forever, and (see the .sh readback) the burn the RPC already');
  console.log('    performed can never record — deviceId MUST come from the mint envelope’s sub.');
  await db.close();
}

clearTimeout(hardTimeout);
process.exit(0);
