// push-lost-race.mjs — two RxDB device clients, each with a locally-queued
// pending scan_attempt for the SAME code, pushing concurrently. The push
// handler is the §6 shape: batch pending attempts through redeem(), land the
// attempt row server-side, write the outcome back onto the local row. The
// devices can NEVER read scan_attempts back (push-only RLS), so everything a
// device knows arrives through its own push outcomes + the codes replica.
//
// argv: <jwtDeviceA> <jwtDeviceB> <codeId>
// exit 0 all client legs held; exit 1 a leg failed.

import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';

const [jwtA, jwtB, CODE] = process.argv.slice(2);
if (!jwtA || !jwtB || !CODE) {
  console.error('usage: push-lost-race.mjs <jwtA> <jwtB> <codeId>');
  process.exit(2);
}
const fail = (msg) => { console.error(`RED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (90s)'), 90_000);
const REST = `http://127.0.0.1:${REST_PORT}`;
const today = new Date().toISOString().slice(0, 10);

addRxPlugin(RxDBDevModePlugin);

const attemptSchema = {
  version: 0,
  primaryKey: 'id',
  type: 'object',
  properties: {
    id: { type: 'string', maxLength: 100 },
    code_id: { type: 'string' },
    device_id: { type: 'string' },
    scanned_at: { type: 'string' },
    status: { type: 'string' },            // pending | accepted | rejected
    reason: { type: ['string', 'null'] },
    offline_override: { type: 'boolean' },
    unverified_code: { type: 'boolean' },
    pos_business_date: { type: 'string' },
    winner_device: { type: ['string', 'null'] },  // local-only render data for the flip
    winner_at: { type: ['string', 'null'] },
  },
  required: ['id', 'code_id', 'device_id', 'scanned_at', 'status', 'pos_business_date'],
};

async function makeDevice(name, jwt) {
  const db = await createRxDatabase({
    name: `spike_push_${name}_${Date.now()}`,
    storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
  });
  await db.addCollections({ attempts: { schema: attemptSchema } });

  const auth = { Authorization: `Bearer ${jwt}`, 'Content-Type': 'application/json' };
  const outcomes = [];   // write-backs to apply after the push pass
  let handlerCalls = 0;

  // the §6 push handler: pending → redeem() → land attempt row → queue write-back
  async function pushHandler(rows) {
    handlerCalls++;
    for (const row of rows) {
      const doc = row.newDocumentState;
      if (doc.status !== 'pending') continue; // idempotence: resolved rows re-push as no-ops
      const burn = await fetch(`${REST}/rpc/redeem`, {
        method: 'POST', headers: auth,
        body: JSON.stringify({ p_code: doc.code_id, p_device: name }),
      });
      if (burn.status !== 200) throw new Error(`redeem HTTP ${burn.status}`);
      const body = await burn.json();
      const verdict = Array.isArray(body) ? body[0] : body;
      const status = verdict.ok ? 'accepted' : 'rejected';
      const reason = verdict.ok ? null : verdict.reason;
      console.log(`  [${name}] redeem(${doc.code_id.slice(0, 8)}…) → ok=${verdict.ok} reason=${reason ?? '-'}`);

      const land = await fetch(`${REST}/scan_attempts`, {
        method: 'POST', headers: { ...auth, Prefer: 'return=minimal' },
        body: JSON.stringify({
          id: doc.id, code_id: doc.code_id, device_id: name,
          scanned_at: doc.scanned_at, status, reason,
          offline_override: true, unverified_code: false,
          pos_business_date: doc.pos_business_date,
        }),
      });
      if (land.status !== 201) throw new Error(`scan_attempts insert HTTP ${land.status} for ${name}`);
      outcomes.push({ id: doc.id, status, reason });
    }
    return []; // no conflicts — the master accepted what we told it
  }

  return { name, db, auth, outcomes, pushHandler, calls: () => handlerCalls };
}

const A = await makeDevice('device-a', jwtA);
const B = await makeDevice('device-b', jwtB);

// ---------------------------------------------------------------------------
// the offline queue: BOTH devices accept the same code locally, pre-replication
// ---------------------------------------------------------------------------
for (const dev of [A, B]) {
  // schema fact (first run's 400): scan_attempts.id is uuid — the device must
  // generate REAL uuids, not app-prefixed strings. crypto.randomUUID() is the
  // on-device generator the card inherits.
  await dev.db.attempts.insert({
    id: crypto.randomUUID(),
    code_id: CODE, device_id: dev.name,
    scanned_at: new Date().toISOString(),
    status: 'pending', reason: null,
    offline_override: true, unverified_code: false,
    pos_business_date: today, winner_device: null, winner_at: null,
  });
}
console.log('# both devices hold a pending attempt for the same code (offline double-accept reconstructed)');

// ---------------------------------------------------------------------------
// concurrent push — replication starts on both at once
// ---------------------------------------------------------------------------
const reps = [A, B].map((dev) => replicateRxCollection({
  collection: dev.db.attempts,
  replicationIdentifier: `spike-push-${dev.name}`,
  live: true,
  waitForLeadership: false,
  push: { handler: dev.pushHandler, batchSize: 10 },
}));
reps.forEach((r, i) => r.error$.subscribe((e) => console.log(`!! [${[A, B][i].name}] replication error:`, e.message || e)));
await Promise.all(reps.map((r) => r.awaitInitialReplication()));

// write the outcomes back onto the local rows (the flip), then settle
for (const dev of [A, B]) {
  for (const o of dev.outcomes) {
    const patch = { status: o.status, reason: o.reason };
    if (o.status === 'rejected') {
      // the render data for "already used at 6:42pm" comes from the codes side
      const r = await fetch(`${REST}/codes?select=redeemed_by,redeemed_at&id=eq.${CODE}`, { headers: dev.auth });
      const [codeRow] = await r.json();
      patch.winner_device = codeRow?.redeemed_by ?? null;
      patch.winner_at = codeRow?.redeemed_at ?? null;
    }
    const doc = await dev.db.attempts.findOne(o.id).exec();
    await doc.patch(patch);
  }
}
await Promise.all(reps.map((r) => r.awaitInSync()));

// ---------------------------------------------------------------------------
// client-side legs
// ---------------------------------------------------------------------------
const aDocs = await A.db.attempts.find().exec();
const bDocs = await B.db.attempts.find().exec();
const all = [...aDocs, ...bDocs].map((d) => ({
  device: d.device_id, status: d.status, reason: d.reason,
  winner_device: d.winner_device, winner_at: d.winner_at,
}));
console.log('# local rows after the round trip:');
for (const r of all) console.log(`    ${JSON.stringify(r)}`);

const accepted = all.filter((r) => r.status === 'accepted');
const rejected = all.filter((r) => r.status === 'rejected');
if (accepted.length !== 1) fail(`expected exactly 1 accepted local row, got ${accepted.length}`);
if (rejected.length !== 1) fail(`expected exactly 1 rejected local row, got ${rejected.length}`);
if (rejected[0].reason !== 'already_used') fail(`loser's reason is ${rejected[0].reason}, not already_used`);
if (rejected[0].winner_device !== accepted[0].device) fail('loser does not carry the winning device');
if (!rejected[0].winner_at) fail('loser does not carry the winning time');
console.log(`  LOST-RACE FLIP HELD — ${rejected[0].device} shows "already used at ${rejected[0].winner_at} by ${rejected[0].winner_device}"`);

// write-back must not loop: give live push a beat, then check handler counts
await new Promise((r) => setTimeout(r, 1500));
console.log(`  handler invocations: device-a=${A.calls()} device-b=${B.calls()}`);
if (A.calls() > 4 || B.calls() > 4) fail('push handler invocation count exploded — the write-back loops');

// push-only holds: the device role must NOT be able to read scan_attempts
const probe = await fetch(`${REST}/scan_attempts?select=id`, { headers: A.auth });
const probeBody = await probe.text();
console.log(`  device SELECT scan_attempts → HTTP ${probe.status}`);
const ownIds = [...A.outcomes, ...B.outcomes].map((o) => o.id);
if (probe.status === 200 && ownIds.some((id) => probeBody.includes(id))) {
  fail('a device can read scan_attempts back — push-only does not hold');
}

clearTimeout(hardTimeout);
await Promise.all(reps.map((r) => r.cancel()));
await A.db.close(); await B.db.close();
console.log('ALL CLIENT LEGS HELD');
process.exit(0);
