// marketing/sync/harness/push-harness.mjs — the node half of Card 3's standalone
// gate (card scan-attempts-push-conflict, run 20260905; see push-run.sh for the
// substrate half and the verdict contract). A SIBLING of Card 2's harness.mjs —
// Card 2's landed gate stays byte-identical.
//
// GREEN mode drives the PRODUCTION modules (../push-replication.js for the push
// side, ../replicas.js for the codes-side pull replica the loser's flip renders
// from) — dynamically imported so the red mode, which predates push-replication.js
// in git history, never touches it.
//
// RED mode (red-gap1) is a deliberately defective inline probe — the spike's
// naive handler shape (redeem → land → patch; no persisted burn outcome, no
// own-device arbitration) — under an INJECTED network failure on the FIRST
// scan_attempts landing insert. The push retry re-runs redeem(), the re-burn
// answers already_used to the device that in fact WON, and the naive handler
// flips the winner's local row to rejected/already_used. The "winner stays
// accepted" assertion failing (exit 1) IS GAP-1's window, demonstrated.
//
// Discipline inherited from Card 2: request logs are enumerated (B-216), never
// inferred; the verdict is the exit status, never the prose.

import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { RxDBMigrationSchemaPlugin } from 'rxdb/plugins/migration-schema';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../.night-crew/qa/spike-supabase/rxdb/spike-env.js';

const MODE = process.argv[2] || 'green';
const env = (k) => {
  const v = process.env[k];
  if (!v) { console.error(`missing env ${k}`); process.exit(2); }
  return v;
};
const JWT_A = env('C3_JWT_A');
const JWT_B = env('C3_JWT_B');
const TARGET = env('C3_TARGET');   // the two-device race code
const W1 = env('C3_W1');           // land-fails-after-redeem code (GAP-1 belt 1)
const W2 = env('C3_W2');           // redeem-response-lost code (GAP-1 belt 2)

const REST = `http://127.0.0.1:${REST_PORT}`;
const fail = (msg) => { console.error(`RED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (150s) — a leg never finished'), 150_000);
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

addRxPlugin(RxDBDevModePlugin);
// 🛑 REQUIRED since SCAN_ATTEMPTS_SCHEMA went to version 1 (card
// refusal-holds-before-sync, run 20260906-2): rxdb runs
// `autoMigrate && version !== 0 && await migratePromise()` on every
// collection creation, and without this plugin that call THROWS
// ("You are using a function which must be overwritten by a plugin") —
// addCollections rejects and the harness dies before its first leg. The
// browser gets the same registration in marketing/scan-page.js.
addRxPlugin(RxDBMigrationSchemaPlugin);

async function makeDb(name, collections) {
  const db = await createRxDatabase({
    name: `c3_${name}_${Date.now()}`,
    storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
  });
  await db.addCollections(collections);
  return db;
}

// A recording fetch wrapper with an optional per-request fault injector.
// Every request is enumerated {method, url} (B-216); `inject(method, url)` may
// return 'fail-now' (throw before sending — landing never reaches the server)
// or 'lose-response' (send for real, then throw — the server committed, the
// client never heard). Returns the wrapper plus its trace.
function recordingFetch({ inject } = {}) {
  const trace = [];
  const injected = [];
  const wrapper = async (url, init = {}) => {
    const method = (init.method || 'GET').toUpperCase();
    trace.push({ method, url: String(url) });
    const order = inject ? inject(method, String(url)) : null;
    if (order === 'fail-now') {
      injected.push({ order, method, url: String(url) });
      throw new TypeError('injected: network failure before the request was sent');
    }
    const res = await fetch(url, init);
    if (order === 'lose-response') {
      injected.push({ order, method, url: String(url) });
      throw new TypeError('injected: response lost after the server committed');
    }
    return res;
  };
  return { wrapper, trace, injected };
}

const isLanding = (m, u) => m === 'POST' && u.includes('/scan_attempts');
const isRedeem = (m, u) => m === 'POST' && u.includes('/rpc/redeem');

// ═══════════════════════════════════════════════════════════════════════════
// RED — the naive handler under the land-fails-after-redeem window.
// ═══════════════════════════════════════════════════════════════════════════

const NAIVE_ATTEMPT_SCHEMA = {
  version: 0,
  primaryKey: 'id',
  type: 'object',
  properties: {
    id: { type: 'string', maxLength: 100 },
    code_id: { type: 'string' },
    device_id: { type: 'string' },
    scanned_at: { type: 'string' },
    status: { type: 'string' },
    reason: { type: ['string', 'null'] },
    offline_override: { type: 'boolean' },
    unverified_code: { type: 'boolean' },
    pos_business_date: { type: 'string' },
  },
  required: ['id', 'code_id', 'device_id', 'scanned_at', 'status', 'pos_business_date'],
};

if (MODE === 'red-gap1') {
  console.log('# red-gap1 — naive redeem-then-land handler; FIRST landing insert fails after a successful redeem()');
  const db = await makeDb('red', { attempts: { schema: NAIVE_ATTEMPT_SCHEMA } });
  let landCalls = 0;
  const { wrapper, trace, injected } = recordingFetch({
    inject: (m, u) => (isLanding(m, u) && ++landCalls === 1 ? 'fail-now' : null),
  });
  const auth = { Authorization: `Bearer ${JWT_A}`, 'Content-Type': 'application/json' };
  const redeemVerdicts = [];

  // the spike's §6 handler shape, verbatim in spirit: no persisted burn
  // outcome, no own-device arbitration — resolve straight from the verdict.
  async function naivePush(rows) {
    for (const row of rows) {
      const state = row.newDocumentState;
      if (state.status !== 'pending') continue;
      const burn = await wrapper(`${REST}/rpc/redeem`, {
        method: 'POST', headers: auth,
        body: JSON.stringify({ p_code: state.code_id, p_device: 'device-a' }),
      });
      if (burn.status !== 200) throw new Error(`redeem HTTP ${burn.status}`);
      const body = await burn.json();
      const verdict = Array.isArray(body) ? body[0] : body;
      redeemVerdicts.push(verdict);
      console.log(`  redeem() → ok=${verdict.ok} reason=${verdict.reason ?? '-'}`);
      const status = verdict.ok ? 'accepted' : 'rejected';
      const reason = verdict.ok ? null : verdict.reason;
      const land = await wrapper(`${REST}/scan_attempts`, {
        method: 'POST', headers: { ...auth, Prefer: 'return=minimal' },
        body: JSON.stringify({
          id: state.id, code_id: state.code_id, device_id: 'device-a',
          scanned_at: state.scanned_at, status, reason,
          offline_override: true, unverified_code: false,
          pos_business_date: state.pos_business_date,
        }),
      });
      if (land.status !== 201) throw new Error(`scan_attempts insert HTTP ${land.status}`);
      const doc = await db.attempts.findOne(state.id).exec();
      await doc.incrementalPatch({ status, reason });
      console.log(`  local row resolved → ${status}${reason ? `/${reason}` : ''}`);
    }
    return [];
  }

  const now = new Date().toISOString();
  await db.attempts.insert({
    id: crypto.randomUUID(), code_id: W1, device_id: 'device-a',
    scanned_at: now, status: 'pending', reason: null,
    offline_override: true, unverified_code: false, pos_business_date: now.slice(0, 10),
  });
  const attemptId = (await db.attempts.find().exec())[0].id;

  const rep = replicateRxCollection({
    collection: db.attempts,
    replicationIdentifier: `c3-red-${Date.now()}`,
    live: true, waitForLeadership: false, retryTime: 500,
    push: { handler: naivePush, batchSize: 10 },
  });
  rep.error$.subscribe((e) => console.log(`  (replication error, retrying: ${e?.rxdb ? e.parameters?.errors?.[0]?.message ?? e.message : e.message ?? e})`));

  const doc = await until(async () => {
    const d = await db.attempts.findOne(attemptId).exec();
    return d && d.status !== 'pending' ? d : null;
  }, 30_000, 'naive attempt never resolved');

  console.log(`# outcome: local row status=${doc.status} reason=${doc.reason ?? '-'}`);
  console.log(`# injected failures: ${injected.length} (${injected.map((i) => i.order).join(', ')})`);
  console.log(`# redeem() calls: ${trace.filter((t) => isRedeem(t.method, t.url)).length}, verdicts: ${JSON.stringify(redeemVerdicts)}`);
  if (injected.length === 0) fail('probe broken — the landing failure was never injected');
  if (!redeemVerdicts[0]?.ok) fail('probe broken — the first redeem() did not win the burn (this device IS the winner by construction)');

  clearTimeout(hardTimeout);
  await rep.cancel();
  await db.close();
  // THE assertion: this device won the burn (first redeem ok=true, proven
  // above) — its UI must say accepted. The naive shape re-burned after the
  // landing failure and flipped the WINNER to rejected/already_used.
  if (doc.status !== 'accepted') {
    fail(`GAP-1 demonstrated — the WINNING device's local row mis-flipped to ${doc.status}/${doc.reason} after a transient landing failure (the retry re-ran redeem() and believed the already_used echo of its own burn)`);
  }
  console.log('  winner stayed accepted — the naive probe did NOT mis-flip (probe unexpectedly green)');
  process.exit(0);
}

// ═══════════════════════════════════════════════════════════════════════════
// GREEN — the production modules, every done_when clause.
// ═══════════════════════════════════════════════════════════════════════════

if (MODE !== 'green') fail(`unknown mode ${MODE}`);

const {
  scanAttemptsCollectionSpec, enqueueAttempt, makePushHandler, startScanAttemptsReplica,
} = await import('../push-replication.js');
const {
  startCodesReplica, MARKETING_REPLICA_SCHEMA, CODES_COLLECTION,
} = await import('../replicas.js');

// One device = one RxDB db holding the scan_attempts queue AND its own
// codes-side pull replica (Card 2's PRODUCTION module — the loser's display
// data source). The pull stream is nudged on an interval; the nudge transport
// (Realtime vs timer) is orthogonal to this card and Realtime is Card 2's
// already-proven leg.
async function makeDevice(name, jwt, { inject } = {}) {
  const db = await makeDb(name.replace(/-/g, '_'), {
    [CODES_COLLECTION]: { schema: MARKETING_REPLICA_SCHEMA },
    ...scanAttemptsCollectionSpec(),
  });
  const { wrapper, trace, injected } = recordingFetch({ inject });
  const pushLog = [];
  const codesLog = [];
  const stream$ = new Subject();
  let pushCalls = 0;
  const handler = makePushHandler({
    restUrl: REST, bearer: jwt, deviceId: name, fetchImpl: wrapper,
    attemptsCollection: db.scan_attempts, codesCollection: db.codes,
    requestLog: pushLog, winnerWaitMs: 20_000, winnerPollMs: 100,
  });
  const countingHandler = async (rows) => { pushCalls++; return handler(rows); };
  const statusHistory = [];
  db.scan_attempts.$.subscribe((ev) => {
    const s = ev?.documentData?.status;
    if (s && statusHistory[statusHistory.length - 1] !== s) statusHistory.push(s);
  });

  const dev = {
    name, jwt, db, trace, injected, pushLog, codesLog, stream$, statusHistory,
    calls: () => pushCalls,
    reps: [],
    nudge: null,
    async goOnline() {
      dev.reps.push(startCodesReplica({
        replicateRxCollection, collection: db.codes, restUrl: REST, bearer: jwt,
        fetchImpl: wrapper, stream$: stream$.asObservable(), batchSize: 50,
        requestLog: codesLog, replicationIdentifier: `c3-codes-${name}-${Date.now()}`,
      }));
      dev.reps.push(startScanAttemptsReplica({
        replicateRxCollection, collection: db.scan_attempts,
        pushHandler: countingHandler, retryTime: 500,
        replicationIdentifier: `c3-push-${name}-${Date.now()}`,
      }));
      for (const r of dev.reps) r.error$.subscribe((e) => console.log(`  (${name} replication error, retrying: ${e?.message ?? e})`));
      dev.nudge = setInterval(() => stream$.next('RESYNC'), 300);
    },
    async teardown() {
      if (dev.nudge) clearInterval(dev.nudge);
      for (const r of dev.reps) await r.cancel();
      await db.close();
    },
  };
  return dev;
}

const resolved = (db, id) => async () => {
  const d = await db.scan_attempts.findOne(id).exec();
  return d && d.status !== 'pending' && d.landed ? d : null;
};

// ---------------------------------------------------------------------------
// leg A — the offline queue: both devices enqueue for the SAME code before any
// replication runs (§8 double-accept reconstructed), and enqueue dedupes.
// ---------------------------------------------------------------------------
const A = await makeDevice('device-a', JWT_A);
const B = await makeDevice('device-b', JWT_B);
const qa = await enqueueAttempt(A.db.scan_attempts, { code_id: TARGET, device_id: 'device-a', offline_override: true });
const qb = await enqueueAttempt(B.db.scan_attempts, { code_id: TARGET, device_id: 'device-b', offline_override: true });
if (qa.deduped || qb.deduped) fail('fresh enqueue reported deduped');
if (qa.doc.status !== 'pending' || qb.doc.status !== 'pending') fail('queued attempt not pending');
if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(qa.doc.id))
  fail(`attempt id is not a real uuid: ${qa.doc.id} (spike build-fact — app-prefixed strings draw 400)`);
const dupe = await enqueueAttempt(A.db.scan_attempts, { code_id: TARGET, device_id: 'device-a', offline_override: true });
if (!dupe.deduped || dupe.doc.id !== qa.doc.id) fail('re-enqueue of a live attempt did not dedupe to the existing doc');
console.log('# leg A HELD — both devices hold a pending offline attempt for the same code; ids are real uuids; re-enqueue dedupes');

// ---------------------------------------------------------------------------
// leg B — concurrent push: exactly one accepted, the other rejected/already_used
// ---------------------------------------------------------------------------
await Promise.all([A.goOnline(), B.goOnline()]);
const [docA, docB] = await Promise.all([
  until(resolved(A.db, qa.doc.id), 45_000, 'device-a attempt never resolved+landed'),
  until(resolved(B.db, qb.doc.id), 45_000, 'device-b attempt never resolved+landed'),
]);
const rows = [
  { device: 'device-a', status: docA.status, reason: docA.reason, winner_device: docA.winner_device, winner_at: docA.winner_at },
  { device: 'device-b', status: docB.status, reason: docB.reason, winner_device: docB.winner_device, winner_at: docB.winner_at },
];
console.log('# leg B — local rows after the concurrent push:');
for (const r of rows) console.log(`    ${JSON.stringify(r)}`);
const accepted = rows.filter((r) => r.status === 'accepted');
const rejected = rows.filter((r) => r.status === 'rejected');
if (accepted.length !== 1) fail(`expected exactly 1 accepted local row, got ${accepted.length}`);
if (rejected.length !== 1) fail(`expected exactly 1 rejected local row, got ${rejected.length}`);
if (rejected[0].reason !== 'already_used') fail(`loser's reason is ${rejected[0].reason}, not already_used`);
console.log(`  leg B HELD — exactly one accepted (${accepted[0].device}); handler invocations device-a=${A.calls()} device-b=${B.calls()}`);
if (A.calls() > 6 || B.calls() > 6) fail('push handler invocation count exploded — the resolution write-back loops');

// ---------------------------------------------------------------------------
// leg C — the loser's flip carries the winner, and the data PROVABLY comes
// from its own codes-side pull replica (never a scan_attempts read-back)
// ---------------------------------------------------------------------------
const loser = rejected[0].device === 'device-a' ? A : B;
const winner = accepted[0].device;
if (rejected[0].winner_device !== winner) fail(`loser carries winner_device=${rejected[0].winner_device}, expected ${winner}`);
if (!rejected[0].winner_at) fail('loser does not carry the winning time');
const loserCodeDoc = await loser.db.codes.findOne(TARGET).exec();
if (!loserCodeDoc) fail("loser's codes replica never pulled the target code");
if (loserCodeDoc.redeemed_by !== rejected[0].winner_device || loserCodeDoc.redeemed_at !== rejected[0].winner_at)
  fail("loser's flip data does not match its own codes replica row — the display data did not come from the pull replica");
const attemptReads = [...A.trace, ...B.trace].filter((t) => t.method === 'GET' && t.url.includes('/scan_attempts'));
if (attemptReads.length > 0) fail(`the mechanism read scan_attempts back: ${JSON.stringify(attemptReads)}`);
console.log(`  leg C HELD — "already used at ${rejected[0].winner_at} by ${rejected[0].winner_device}" rendered from the loser's OWN codes pull replica; zero scan_attempts reads in either device's trace`);

// ---------------------------------------------------------------------------
// leg D — GAP-1 belt 1, the card's owed validation run: the WINNER survives an
// injected landing failure after a successful redeem(), without flipping.
// ---------------------------------------------------------------------------
let w1LandCalls = 0;
const D = await makeDevice('device-a', JWT_A, {
  inject: (m, u) => (isLanding(m, u) && ++w1LandCalls === 1 ? 'fail-now' : null),
});
const qd = await enqueueAttempt(D.db.scan_attempts, { code_id: W1, device_id: 'device-a', offline_override: true });
await D.goOnline();
const docD = await until(resolved(D.db, qd.doc.id), 45_000, 'W1 attempt never resolved+landed');
const dRedeems = D.pushLog.filter((r) => r.kind === 'redeem' && r.code_id === W1);
const dLands = D.pushLog.filter((r) => r.kind === 'land' && r.attempt_id === qd.doc.id);
console.log(`# leg D — land-fails-after-redeem window: status=${docD.status}, redeem requests=${dRedeems.length}, landing requests=${dLands.length}, injected=${D.injected.length}, status history=[${D.statusHistory.join(' → ')}]`);
if (!D.statusHistory.includes('accepted')) fail('harness defect — the status-history subscription observed no terminal state; the transient-flip assertion would be vacuous');
if (D.injected.length !== 1) fail(`probe broken — expected exactly 1 injected landing failure, got ${D.injected.length}`);
if (docD.status !== 'accepted') fail(`GAP-1 OPEN — the winner's row ended ${docD.status}/${docD.reason} after a transient landing failure`);
if (D.statusHistory.includes('rejected')) fail(`GAP-1 OPEN — the winner's row TRANSIENTLY flipped: [${D.statusHistory.join(' → ')}]`);
if (dRedeems.length !== 1) fail(`redeem() ran ${dRedeems.length}× — the persisted burn outcome did not suppress the re-burn`);
if (dLands.length < 2) fail(`landing was not retried (${dLands.length} request(s)) — the failed insert was silently dropped`);
console.log('  leg D HELD — burn outcome persisted before landing: ONE redeem, landing retried, winner never flipped (GAP-1 belt 1 validated)');

// ---------------------------------------------------------------------------
// leg E — GAP-1 belt 2: the redeem RESPONSE is lost after the server committed
// the burn; the retry's already_used names our own device → accepted.
// ---------------------------------------------------------------------------
let w2Redeems = 0;
const E = await makeDevice('device-b', JWT_B, {
  inject: (m, u) => (isRedeem(m, u) && ++w2Redeems === 1 ? 'lose-response' : null),
});
const qe = await enqueueAttempt(E.db.scan_attempts, { code_id: W2, device_id: 'device-b', offline_override: true });
await E.goOnline();
const docE = await until(resolved(E.db, qe.doc.id), 45_000, 'W2 attempt never resolved+landed');
const eRedeems = E.pushLog.filter((r) => r.kind === 'redeem' && r.code_id === W2);
console.log(`# leg E — redeem-response-lost window: status=${docE.status}, redeem requests=${eRedeems.length}, injected=${E.injected.length}, status history=[${E.statusHistory.join(' → ')}]`);
if (E.injected.length !== 1) fail(`probe broken — expected exactly 1 lost redeem response, got ${E.injected.length}`);
if (docE.status !== 'accepted') fail(`GAP-1 OPEN — already_used echoing our OWN burn ended ${docE.status}/${docE.reason} instead of accepted`);
if (E.statusHistory.includes('rejected')) fail(`GAP-1 OPEN — the winner's row transiently flipped: [${E.statusHistory.join(' → ')}]`);
if (eRedeems.length !== 2) fail(`expected 2 redeem requests (lost + retry), got ${eRedeems.length}`);
console.log('  leg E HELD — own-device already_used arbitrated as accepted via the codes replica (GAP-1 belt 2 validated)');

// ---------------------------------------------------------------------------
// leg F — push-only still holds at the API surface: a device cannot read
// scan_attempts back.
// ---------------------------------------------------------------------------
const probe = await fetch(`${REST}/scan_attempts?select=id`, { headers: { Authorization: `Bearer ${JWT_A}` } });
const probeBody = await probe.text();
console.log(`# leg F — device SELECT scan_attempts → HTTP ${probe.status}`);
const ownIds = [qa.doc.id, qb.doc.id, qd.doc.id, qe.doc.id];
if (probe.status === 200 && ownIds.some((id) => probeBody.includes(id)))
  fail('a device can read scan_attempts back — push-only does not hold');
console.log('  leg F HELD — the write-only property stands (403 / no rows visible)');

// ---------------------------------------------------------------------------
// enumerated request logs (B-216)
// ---------------------------------------------------------------------------
for (const [label, dev] of [['device-a(race)', A], ['device-b(race)', B], ['W1', D], ['W2', E]]) {
  console.log(`# push request log — ${label} (${dev.pushLog.length} requests):`);
  for (const [i, r] of dev.pushLog.entries())
    console.log(`    #${i + 1} ${r.kind} attempt=${r.attempt_id?.slice(0, 8)}… code=${r.code_id?.slice(0, 8)}…`);
}

clearTimeout(hardTimeout);
await Promise.all([A.teardown(), B.teardown(), D.teardown(), E.teardown()]);
console.log('ALL LEGS HELD');
process.exit(0);
