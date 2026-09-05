// f2-push-poison.mjs — F-2 run, not reasoned about.
//
// Under test: the SHIPPED marketing/sync/push-replication.js (enqueueAttempt,
// makePushHandler, startScanAttemptsReplica) against the real PostgREST as a
// device JWT, fed exactly what marketing/submit-flow.js:240 composes for an
// unknown-code offline override:  code_id = token_hash  (64 hex).
//
// argv: <deviceJwt> <dbContainerId>   exit 0 all legs agreed; exit 1 one did not.

import { execFileSync } from 'node:child_process';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';
import {
  enqueueAttempt,
  makePushHandler,
  startScanAttemptsReplica,
  scanAttemptsCollectionSpec,
  SCAN_ATTEMPTS_COLLECTION,
} from '../../../../../marketing/sync/push-replication.js';
import { marketingCollectionSpec, CODES_COLLECTION } from '../../../../../marketing/sync/replicas.js';

const [JWT, DB_CID] = process.argv.slice(2);
if (!JWT || !DB_CID) { console.error('usage: f2-push-poison.mjs <deviceJwt> <dbContainerId>'); process.exit(2); }

const REST = `http://127.0.0.1:${REST_PORT}`;
const DEVICE = 'device-a'; // must equal the JWT sub (RLS with-check)
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (180s)'), 180_000);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const sql = (text) => execFileSync(
  'docker',
  ['exec', '-i', DB_CID, 'psql', '-U', 'supabase_admin', '-d', 'postgres', '-v', 'ON_ERROR_STOP=1', '-qtA'],
  { input: text, encoding: 'utf8' },
).trim();

const auth = { Authorization: `Bearer ${JWT}`, 'Content-Type': 'application/json' };

// The F2 shape: a token that hashed to something real, but names no code row.
const UNKNOWN_HASH = 'f'.repeat(8) + '0123456789abcdef'.repeat(3) + 'abcdefff';   // 64 hex
const serverRows = () => Number(sql('select count(*) from public.scan_attempts;'));

// A code is SINGLE USE — redeem() burns it. Every leg that pushes a legitimate
// attempt gets its OWN freshly-minted live code, or the second leg's redeem()
// answers `already_used` and the handler blocks awaiting a winner it can never
// see (this spike runs no codes pull replica). Measured the hard way on the
// first run: reusing one seeded code made leg (c2) red for a harness reason
// that looked exactly like a product finding.
let codeSeq = 0;
function freshCode(tag) {
  const id = sql('select gen_random_uuid();');
  codeSeq += 1;
  sql(`insert into public.codes (id, token_hash, campaign_id, expires_at)
       values ('${id}', 'spike-f2-${tag}-${codeSeq}', 'a0000000-0000-4000-8000-000000000001', now() + interval '1 day');`);
  return id;
}

// RxDB wraps handler errors; dig out something a reader can act on.
const errText = (e) => String(
  e?.parameters?.errors?.[0]?.message
  || e?.parameters?.error?.message
  || e?.message
  || e,
).split('\n')[0];

if (UNKNOWN_HASH.length !== 64) fail(`test fixture wrong: hash length ${UNKNOWN_HASH.length}`);

// ---------------------------------------------------------------------------
// LEG (a) — which endpoint refuses it, and with what status?
// ---------------------------------------------------------------------------
console.log('\n── leg (a): the raw endpoints, given a 64-hex code_id ──');
const redeemRes = await fetch(`${REST}/rpc/redeem`, {
  method: 'POST', headers: auth,
  body: JSON.stringify({ p_code: UNKNOWN_HASH, p_device: DEVICE }),
});
const redeemBody = await redeemRes.text();
console.log(`  POST /rpc/redeem      → HTTP ${redeemRes.status}  ${redeemBody.slice(0, 160)}`);

const landRes = await fetch(`${REST}/scan_attempts`, {
  method: 'POST', headers: { ...auth, Prefer: 'return=minimal' },
  body: JSON.stringify({
    id: crypto.randomUUID(), code_id: UNKNOWN_HASH, device_id: DEVICE,
    scanned_at: new Date().toISOString(), status: 'accepted', reason: null,
    offline_override: true, override_by: 'spike', unverified_code: true,
    pos_order_number: 'A-1', pos_business_date: new Date().toISOString().slice(0, 10),
    redeemed_value: null,
  }),
});
const landBody = await landRes.text();
console.log(`  POST /scan_attempts   → HTTP ${landRes.status}  ${landBody.slice(0, 160)}`);

const redeemRefuses = redeemRes.status !== 200;
const landRefuses = landRes.status !== 201;
if (!redeemRefuses && !landRefuses) {
  fail('NEITHER endpoint refused a 64-hex code_id — F-2 does not reproduce and the card\'s premise is false');
}
console.log(`  → first refusal: ${redeemRefuses ? '/rpc/redeem (p_code uuid)' : '/scan_attempts (code_id uuid not null)'}`
  + ` — the guard must sit BEFORE that call in the push handler`);

// ---------------------------------------------------------------------------
// The device: the shipped scan_attempts queue + a codes collection (the push
// handler's only winner-data source).
// ---------------------------------------------------------------------------
addRxPlugin(RxDBDevModePlugin);
async function freshDb(tag) {
  const db = await createRxDatabase({
    name: `spike_f2_${tag}_${Date.now()}`,
    storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
  });
  await db.addCollections({ ...scanAttemptsCollectionSpec(), ...marketingCollectionSpec() });
  return db;
}

// ---------------------------------------------------------------------------
// LEG (b) — blast radius. Poison queued FIRST, a legitimate attempt behind it,
// the SHIPPED handler on a live push replica with a short retry.
// ---------------------------------------------------------------------------
console.log('\n── leg (b): blast radius — poison first, a legitimate attempt behind it ──');
const before = serverRows();
{
  const LEGIT_CODE = freshCode('poison');
  const db = await freshDb('poison');
  const attempts = db[SCAN_ATTEMPTS_COLLECTION];
  await enqueueAttempt(attempts, { code_id: UNKNOWN_HASH, device_id: DEVICE, offline_override: true, unverified_code: true, override_by: 'spike' });
  await enqueueAttempt(attempts, { code_id: LEGIT_CODE, device_id: DEVICE, pos_order_number: 'A-2' });

  const requestLog = [];
  const errors = [];
  const rep = startScanAttemptsReplica({
    replicateRxCollection,
    collection: attempts,
    pushHandler: makePushHandler({
      restUrl: REST, bearer: JWT, deviceId: DEVICE, fetchImpl: fetch,
      attemptsCollection: attempts, codesCollection: db[CODES_COLLECTION],
      requestLog, winnerWaitMs: 500, winnerPollMs: 100,
    }),
    retryTime: 500,
  });
  rep.error$.subscribe((e) => errors.push(errText(e)));

  await sleep(6000); // ~12 retry cycles at retryTime 500ms

  const rows = await attempts.find().exec();
  const legit = rows.find((r) => r.code_id === LEGIT_CODE);
  const poison = rows.find((r) => r.code_id === UNKNOWN_HASH);
  const landRequests = requestLog.filter((r) => r.kind === 'land');
  const legitLandRequests = landRequests.filter((r) => r.code_id === LEGIT_CODE);

  console.log(`  push requests attempted: ${requestLog.length} (redeem=${requestLog.filter((r) => r.kind === 'redeem').length}, land=${landRequests.length})`);
  console.log(`  distinct replica errors : ${new Set(errors).size} — e.g. ${[...new Set(errors)][0] || '(none)'}`);
  console.log(`  poison row  : status=${poison.status} burn_ok=${poison.burn_ok} landed=${poison.landed}`);
  console.log(`  legit  row  : status=${legit.status} burn_ok=${legit.burn_ok} landed=${legit.landed}`);
  console.log(`  landing requests EVER made for the legit code: ${legitLandRequests.length}`);
  console.log(`  server scan_attempts rows: ${before} → ${serverRows()}`);

  await rep.cancel();
  await db.close();

  if (legit.status !== 'pending' || legit.landed) {
    console.log('  → the legitimate attempt got through: F-2 does NOT head-of-line poison the queue.');
    console.log('    The card\'s guard is still needed for the unverified row itself, but the');
    console.log('    "retry-poison the queue" framing in the roadmap should be corrected.');
  } else {
    console.log('  → CONFIRMED head-of-line poison: the legitimate attempt never even reached a');
    console.log('    landing request. One unknown-code override strands EVERY later redemption on');
    console.log('    that device until the queue is cleared by hand.');
  }
  if (serverRows() !== before) fail('leg (b) wrote server rows it should not have');
  globalThis.__poisoned = legit.status === 'pending' && !legit.landed;
}

// ---------------------------------------------------------------------------
// LEG (c1) — the REJECTED alternative, demonstrated rejected: skip until
// arbitration. No migration; the handler simply passes over unverified rows.
// The queue drains — and the audit row never reaches the server.
// ---------------------------------------------------------------------------
console.log('\n── leg (c1): the rejected alternative — skip-until-arbitration (no migration) ──');
const beforeC1 = serverRows();
{
  const LEGIT_CODE = freshCode('skip');
  const db = await freshDb('skip');
  const attempts = db[SCAN_ATTEMPTS_COLLECTION];
  await enqueueAttempt(attempts, { code_id: UNKNOWN_HASH, device_id: DEVICE, offline_override: true, unverified_code: true, override_by: 'spike' });
  await enqueueAttempt(attempts, { code_id: LEGIT_CODE, device_id: DEVICE, pos_order_number: 'A-3' });

  const shipped = makePushHandler({
    restUrl: REST, bearer: JWT, deviceId: DEVICE, fetchImpl: fetch,
    attemptsCollection: attempts, codesCollection: db[CODES_COLLECTION],
    winnerWaitMs: 500, winnerPollMs: 100,
  });
  // The guard: drop unverified rows from the batch, delegate the rest to the
  // SHIPPED handler unchanged.
  const guarded = (rows) => shipped(rows.filter((r) => !r.newDocumentState.unverified_code));

  const rep = startScanAttemptsReplica({ replicateRxCollection, collection: attempts, pushHandler: guarded, retryTime: 500 });
  await sleep(4000);

  const rows = await attempts.find().exec();
  const legit = rows.find((r) => r.code_id === LEGIT_CODE);
  console.log(`  legit row: status=${legit.status} landed=${legit.landed}  (queue drained: ${legit.status !== 'pending'})`);
  console.log(`  server scan_attempts rows: ${beforeC1} → ${serverRows()} (+${serverRows() - beforeC1})`);
  await rep.cancel();
  await db.close();

  if (legit.status === 'pending') fail('leg (c1): skip-until-arbitration did not even drain the queue');
  const landedAudit = serverRows() - beforeC1;
  if (landedAudit === 2) {
    console.log('  → UNEXPECTED: the unverified attempt landed too — re-derive this leg');
  } else {
    console.log('  → REJECTED, and this is why: the queue drains, but the audit-flagged attempt is');
    console.log('    STRANDED ON THE DEVICE. Decision 166 ratified unknown→false *because* every');
    console.log('    such attempt is audit-flagged; a guard that never lands the row makes the flag');
    console.log('    a lie and takes the §9 reconciliation record with it.');
  }
}

// ---------------------------------------------------------------------------
// LEG (c2) — the guard the card should build: an unverified attempt names no
// code, so it lands with code_id NULL and the token hash it actually has.
// ---------------------------------------------------------------------------
console.log('\n── leg (c2): the guard — a distinct landing path (code_id nullable + token_hash) ──');
const DDL = `
alter table public.scan_attempts alter column code_id drop not null;
alter table public.scan_attempts add column if not exists token_hash text;
alter table public.scan_attempts drop constraint if exists scan_attempts_names_a_code;
alter table public.scan_attempts add constraint scan_attempts_names_a_code
  check (code_id is not null or (unverified_code and token_hash is not null));
notify pgrst, 'reload schema';`;
console.log('  applying (INSIDE the spike — the migration the card owes):');
console.log(DDL.trim().split('\n').map((l) => `    ${l}`).join('\n'));
sql(DDL);
await sleep(1500); // let PostgREST reload its schema cache

const beforeC2 = serverRows();
{
  const LEGIT_CODE = freshCode('guard');
  const db = await freshDb('guard');
  const attempts = db[SCAN_ATTEMPTS_COLLECTION];
  await enqueueAttempt(attempts, { code_id: UNKNOWN_HASH, device_id: DEVICE, offline_override: true, unverified_code: true, override_by: 'spike' });
  await enqueueAttempt(attempts, { code_id: LEGIT_CODE, device_id: DEVICE, pos_order_number: 'A-4' });

  const shipped = makePushHandler({
    restUrl: REST, bearer: JWT, deviceId: DEVICE, fetchImpl: fetch,
    attemptsCollection: attempts, codesCollection: db[CODES_COLLECTION],
    winnerWaitMs: 500, winnerPollMs: 100,
  });

  // The guard, as a wrapper: unverified rows take the distinct landing path
  // (no redeem() — there is no code row to burn); everything else is the
  // SHIPPED handler, byte-for-byte.
  async function guardedPushHandler(rows) {
    const unverified = rows.filter((r) => r.newDocumentState.unverified_code);
    const rest = rows.filter((r) => !r.newDocumentState.unverified_code);
    for (const row of unverified) {
      const doc = await attempts.findOne(row.newDocumentState.id).exec();
      if (!doc || doc.status !== 'pending') continue;
      const res = await fetch(`${REST}/scan_attempts`, {
        method: 'POST', headers: { ...auth, Prefer: 'return=minimal' },
        body: JSON.stringify({
          id: doc.id,
          code_id: null,                 // it names no code — that is the truth
          token_hash: doc.code_id,       // what it actually has
          device_id: DEVICE,
          scanned_at: doc.scanned_at,
          // TAXONOMY IS THE CARD'S CALL, not this spike's. `accepted` +
          // offline_override + unverified_code is already distinguishable from
          // a real accept, and card 3's slate PARK line said "no new terminal
          // status" — but the card should confirm that against §9/§19 before
          // building on it.
          status: 'accepted', reason: null,
          offline_override: doc.offline_override,
          override_by: doc.override_by ?? null,
          unverified_code: true,
          pos_order_number: doc.pos_order_number ?? null,
          pos_business_date: doc.pos_business_date,
          redeemed_value: doc.redeemed_value ?? null,
        }),
      });
      if (res.status !== 201 && res.status !== 409) {
        throw new Error(`[guard] unverified landing answered HTTP ${res.status} ${await res.text()}`);
      }
      await doc.incrementalPatch({ status: 'accepted', reason: null, landed: true, burn_ok: null });
    }
    return shipped(rest);
  }

  const errors = [];
  const rep = startScanAttemptsReplica({ replicateRxCollection, collection: attempts, pushHandler: guardedPushHandler, retryTime: 500 });
  rep.error$.subscribe((e) => errors.push(errText(e)));
  await sleep(5000);

  const rows = await attempts.find().exec();
  const legit = rows.find((r) => r.code_id === LEGIT_CODE);
  const poison = rows.find((r) => r.code_id === UNKNOWN_HASH);
  console.log(`  legit  row: status=${legit.status} landed=${legit.landed}`);
  console.log(`  unverified row: status=${poison.status} landed=${poison.landed}`);
  console.log(`  replica errors: ${new Set(errors).size} ${[...new Set(errors)].slice(0, 2).join(' | ')}`);
  console.log(`  server scan_attempts rows: ${beforeC2} → ${serverRows()} (+${serverRows() - beforeC2})`);
  console.log('  server-side audit row, as stored:');
  console.log(sql(`select '    ' || coalesce(code_id::text,'(null)') || ' | token_hash=' || coalesce(left(token_hash,12),'(null)')
      || ' | override=' || offline_override || ' | unverified=' || unverified_code || ' | status=' || status
      from public.scan_attempts where unverified_code order by scanned_at desc limit 1;`) || '    (none)');

  await rep.cancel();
  await db.close();

  const problems = [];
  if (legit.status !== 'accepted' || !legit.landed) problems.push(`the legitimate attempt did not land (status=${legit.status})`);
  if (!poison.landed) problems.push(`the unverified attempt did not land (status=${poison.status})`);
  if (serverRows() - beforeC2 !== 2) problems.push(`expected 2 server rows, got ${serverRows() - beforeC2} — the audit row must reach the server`);
  const stored = sql(`select count(*) from public.scan_attempts where unverified_code and code_id is null and token_hash = '${UNKNOWN_HASH}';`);
  if (stored !== '1') problems.push(`the audit row is not stored as (code_id null, token_hash=<hash>) — count=${stored}`);
  if (problems.length) { for (const p of problems) console.error(`  ✗ ${p}`); fail(`${problems.length} disagreement(s) in leg (c2)`); }
  console.log('  → the guard clears it: the legitimate attempt lands, AND the audit-flagged');
  console.log('    unverified attempt lands too — naming no code, carrying the hash it has.');
}

console.log('\n── conclusion ──');
console.log(`  · F-2 reproduces on the real substrate: ${redeemRefuses ? '/rpc/redeem refuses first (p_code uuid)' : '/scan_attempts refuses (code_id uuid not null)'}`);
console.log(`  · blast radius: ${globalThis.__poisoned ? 'HEAD-OF-LINE POISON — one unknown-code override strands every later redemption on that device' : 'no head-of-line block — the roadmap\'s "retry-poison" framing needs correcting'}`);
console.log('  · skip-until-arbitration drains the queue but strands the audit row — REJECTED');
console.log('  · distinct landing path (code_id nullable + token_hash + check constraint) clears both');
console.log('  · OPEN for the card: the server-side status taxonomy for an unverified override');

clearTimeout(hardTimeout);
process.exit(0);
