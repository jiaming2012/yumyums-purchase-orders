// marketing/sync/harness/f2-harness.mjs — the node half of the F-2 guard gate
// (card requires-online-replication, run 20260906; see f2-run.sh for the
// substrate half and the verdict contract). A SIBLING of Cards 2/3/4's
// harnesses — their landed gates stay byte-identical.
//
// This file IS the owed GAP-1 validation run: it re-executes spike 03
// (f2-push-poison-and-guard) against the SHIPPED guard — the spike wrapped the
// shipped handler from the outside; here the guard lives INSIDE
// ../push-replication.js's makePushHandler and the landing-path DDL is the
// COMMITTED migration 20260906000200 (applied by f2-run.sh's apply_all), so
// what this proves is the production surface, wrapper-free.
//
// Legs (green mode):
//   (a) endpoint enumeration, unchanged facts: a 64-hex code_id draws a
//       non-success from BOTH raw endpoints (/rpc/redeem first — p_code uuid;
//       the landing insert too — code_id is STILL uuid, merely nullable).
//       The guard must therefore sit BEFORE the redeem call. Measured, not
//       recalled from the spike.
//   (b) the validation leg: the F-2 queue shape — an unknown-code override
//       (code_id = 64-hex token_hash, unverified_code=true) queued FIRST, a
//       legitimate attempt for a freshly-minted LIVE code behind it
//       (build-fact 5: codes are single-use — every leg mints its own) — run
//       through the SHIPPED makePushHandler on a live push replica. Asserts:
//       both attempts land; the audit row is stored
//       (code_id NULL, token_hash=<hash>, offline_override=t,
//       unverified_code=t, status=accepted — the §9/§19 verdict recorded in
//       the merge intent); the request log shows ZERO redeem calls for the
//       unverified attempt (guard BEFORE redeem, enumerated per B-216) and
//       exactly one for the legitimate one.
//
// RED mode (red-unflagged): the same queue with the 64-hex row's
// unverified_code flag STRIPPED — the guard's discriminator gone, the row
// takes the redeem-first path, the deterministic 400 head-of-line-poisons the
// queue and the leg-(b) assertions red (exit 1). That failing exit
// demonstrates the assertion catches the defect class the guard exists for.
//
// Discipline inherited from Card 2: request logs are enumerated (B-216), never
// inferred; the verdict is the exit status, never the prose.

import { execFileSync } from 'node:child_process';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../.night-crew/qa/spike-supabase/rxdb/spike-env.js';
import {
  enqueueAttempt,
  makePushHandler,
  startScanAttemptsReplica,
  scanAttemptsCollectionSpec,
  SCAN_ATTEMPTS_COLLECTION,
} from '../push-replication.js';
import { marketingCollectionSpec, CODES_COLLECTION } from '../replicas.js';

const MODE = process.argv[2] || 'green';
if (!['green', 'red-unflagged'].includes(MODE)) {
  console.error('usage: f2-harness.mjs [green|red-unflagged]');
  process.exit(64);
}
const env = (k) => {
  const v = process.env[k];
  if (!v) { console.error(`missing env ${k}`); process.exit(2); }
  return v;
};
const JWT = env('F2_JWT');       // device-a — must equal the JWT sub (RLS with-check)
const DB_CID = env('F2_DB_CID');
const DEVICE = 'device-a';

const REST = `http://127.0.0.1:${REST_PORT}`;
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (120s) — a leg never finished'), 120_000);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const sql = (text) => execFileSync(
  'docker',
  ['exec', '-i', DB_CID, 'psql', '-U', 'supabase_admin', '-d', 'postgres', '-v', 'ON_ERROR_STOP=1', '-qtA'],
  { input: text, encoding: 'utf8' },
).trim();

const auth = { Authorization: `Bearer ${JWT}`, 'Content-Type': 'application/json' };

// The F2 shape submit-flow.js:240 composes for an unknown code: the local
// code_id IS the token_hash (64 hex — no code row exists to name).
const UNKNOWN_HASH = 'f'.repeat(8) + '0123456789abcdef'.repeat(3) + 'abcdefff';
if (UNKNOWN_HASH.length !== 64) fail(`fixture wrong: hash length ${UNKNOWN_HASH.length}`);
const serverRows = () => Number(sql('select count(*) from public.scan_attempts;'));

// Build-fact 5 (spike correction 2): a code is SINGLE USE — the leg mints its
// own fresh live code, or redeem() answers already_used and the shipped
// handler's belt 2 correctly blocks awaiting a winner this harness never
// supplies (no codes pull replica runs here). That blocking is a DESIGN
// PROPERTY, not a defect.
const MINTED = [];
function freshCode(tag) {
  const id = sql('select gen_random_uuid();');
  sql(`insert into public.codes (id, token_hash, campaign_id, expires_at)
       values ('${id}', 'f2h-${tag}-${id}', 'a0000000-0000-4000-8000-000000000001', now() + interval '1 day');`);
  MINTED.push(id);
  return id;
}

// RxDB wraps handler errors; dig out something a reader can act on.
const errText = (e) => String(
  e?.parameters?.errors?.[0]?.message
  || e?.parameters?.error?.message
  || e?.message
  || e,
).split('\n')[0];

console.log(`# mode: ${MODE}`);

// ---------------------------------------------------------------------------
// LEG (a) — the raw endpoints, given a 64-hex code_id. Enumerated so the
// guard's placement (BEFORE redeem) rests on a measurement in THIS tree.
// ---------------------------------------------------------------------------
console.log('\n── leg (a): the raw endpoints, given a 64-hex code_id ──');
const redeemRes = await fetch(`${REST}/rpc/redeem`, {
  method: 'POST', headers: auth,
  body: JSON.stringify({ p_code: UNKNOWN_HASH, p_device: DEVICE }),
});
console.log(`  POST /rpc/redeem      → HTTP ${redeemRes.status}  ${(await redeemRes.text()).slice(0, 120)}`);
const landRes = await fetch(`${REST}/scan_attempts`, {
  method: 'POST', headers: { ...auth, Prefer: 'return=minimal' },
  body: JSON.stringify({
    id: crypto.randomUUID(), code_id: UNKNOWN_HASH, device_id: DEVICE,
    scanned_at: new Date().toISOString(), status: 'accepted', reason: null,
    offline_override: true, override_by: 'f2-harness', unverified_code: true,
    pos_order_number: 'A-1', pos_business_date: new Date().toISOString().slice(0, 10),
    redeemed_value: null,
  }),
});
console.log(`  POST /scan_attempts   → HTTP ${landRes.status}  ${(await landRes.text()).slice(0, 120)}`);
if (redeemRes.status === 200) fail('/rpc/redeem ACCEPTED a 64-hex p_code — the premise under the guard is gone; re-derive');
if (landRes.status === 201) fail('/scan_attempts ACCEPTED a 64-hex code_id — the column stopped being uuid; re-derive');
console.log('  → both refuse the raw shape; the guard must divert BEFORE the redeem call');

// ---------------------------------------------------------------------------
// LEG (b) — the validation run: shipped handler, no wrapper.
// ---------------------------------------------------------------------------
console.log(`\n── leg (b): poison-first queue through the SHIPPED handler (${MODE}) ──`);
addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `f2h_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections({ ...scanAttemptsCollectionSpec(), ...marketingCollectionSpec() });
const attempts = db[SCAN_ATTEMPTS_COLLECTION];

const before = serverRows();
const LEGIT_CODE = freshCode('legit');

// The unknown-code override FIRST (the head-of-line position), the legitimate
// attempt behind it. In red-unflagged mode the discriminator is stripped.
await enqueueAttempt(attempts, {
  code_id: UNKNOWN_HASH, device_id: DEVICE,
  offline_override: true, override_by: 'f2-harness',
  unverified_code: MODE === 'green', // red-unflagged: the flag lost → redeem-first path
});
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
await sleep(5000); // ~10 retry cycles at retryTime 500ms — plenty to land or to poison

const rows = await attempts.find().exec();
const legit = rows.find((r) => r.code_id === LEGIT_CODE);
const unv = rows.find((r) => r.code_id === UNKNOWN_HASH);
const redeems = requestLog.filter((r) => r.kind === 'redeem');
const unvRedeems = redeems.filter((r) => r.code_id === UNKNOWN_HASH);
const legitRedeems = redeems.filter((r) => r.code_id === LEGIT_CODE);
const after = serverRows();

console.log(`  push requests: ${requestLog.length} — kinds: ${[...new Set(requestLog.map((r) => r.kind))].join(', ') || '(none)'}`);
console.log(`  redeem calls  : unverified=${unvRedeems.length} legit=${legitRedeems.length}`);
console.log(`  unverified row: status=${unv.status} landed=${unv.landed}`);
console.log(`  legit row     : status=${legit.status} landed=${legit.landed}`);
console.log(`  server scan_attempts rows: ${before} → ${after} (+${after - before})`);
console.log(`  distinct replica errors: ${new Set(errors).size}${errors.length ? ` — e.g. ${[...new Set(errors)][0]}` : ''}`);

// Tolerant of the PRE-migration schema (no token_hash column) so the
// red-first run reports its assertion failures instead of crashing here.
let audit;
try {
  audit = sql(`select coalesce(code_id::text,'(null)') || '|' || coalesce(token_hash,'(null)')
    || '|' || offline_override || '|' || unverified_code || '|' || status
    from public.scan_attempts where token_hash = '${UNKNOWN_HASH}';`);
} catch (e) {
  audit = '(query failed — the landing-path migration is not applied: no token_hash column)';
}
console.log(`  audit row (server): ${audit || '(none)'}`);

await rep.cancel();
await db.close();

// Cleanup (throwaway substrate hygiene; failures here are not a verdict).
try {
  sql(`delete from public.scan_attempts where token_hash = '${UNKNOWN_HASH}' or code_id in ('${MINTED.join("','")}');`);
  sql(`delete from public.codes where id in ('${MINTED.join("','")}');`);
} catch (e) { /* reset_bare on the next run covers it */ }

// ---------------------------------------------------------------------------
// The assertions — the same set in both modes; red-unflagged is DEMONSTRATED
// by these failing (exit 1).
// ---------------------------------------------------------------------------
const problems = [];
if (unvRedeems.length !== 0) problems.push(`the unverified attempt reached /rpc/redeem ${unvRedeems.length}× — the guard must sit BEFORE the burn`);
if (unv.status !== 'accepted' || !unv.landed) problems.push(`the audit-flagged attempt did not land (status=${unv.status}, landed=${unv.landed}) — stranding it falsifies decision 166's reasoning`);
if (legit.status !== 'accepted' || !legit.landed) problems.push(`the legitimate attempt behind it did not land (status=${legit.status}, landed=${legit.landed}) — head-of-line poison`);
if (legitRedeems.length !== 1) problems.push(`expected exactly 1 redeem for the legitimate code, got ${legitRedeems.length}`);
if (after - before !== 2) problems.push(`expected +2 server rows, got +${after - before}`);
if (audit !== `(null)|${UNKNOWN_HASH}|true|true|accepted`) problems.push(`audit row shape wrong: ${audit || '(none)'} — expected (null)|${UNKNOWN_HASH}|true|true|accepted`);

console.log('\n── conclusion ──');
if (problems.length) {
  for (const p of problems) console.error(`  ✗ ${p}`);
  fail(`${problems.length} disagreement(s)`);
}
console.log('  · the guard diverts the unverified attempt BEFORE redeem() (zero redeem calls for it),');
console.log('    lands it on the distinct path (code_id null + token_hash + flags, status accepted),');
console.log('    and the legitimate attempt behind it lands — no head-of-line poison.');
clearTimeout(hardTimeout);
process.exit(0);
