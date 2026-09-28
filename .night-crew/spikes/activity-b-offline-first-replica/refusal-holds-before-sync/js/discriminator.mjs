// discriminator.mjs — can the attempt record distinguish "campaigns replica
// failed/unready" from "genuinely unknown campaign"? Measured per layer.
//
// Everything under test is shipped code:
//   * marketing/sync/push-replication.js — SCAN_ATTEMPTS_SCHEMA (the local
//     queue shape), enqueueAttempt (the scanner's entry point), makePushHandler
//     (the F-2-guarded drain)
// The only things written here are the candidate discriminator field
// (`policy_unresolved`), the card's FUTURE landing body (the shipped
// land-unverified shape + that field), and the stopwatch.
//
// argv: <deviceJwt> pre            — server does NOT have the column
//       <deviceJwt> post <liveCodeId> — column added + pgrst reloaded by the .sh

import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';
import {
  SCAN_ATTEMPTS_SCHEMA,
  scanAttemptsCollectionSpec,
  enqueueAttempt,
  makePushHandler,
} from '../../../../../marketing/sync/push-replication.js';

const [JWT, MODE, LIVE_CODE_ID] = process.argv.slice(2);
if (!JWT || !['pre', 'post'].includes(MODE) || (MODE === 'post' && !LIVE_CODE_ID)) {
  console.error('usage: discriminator.mjs <deviceJwt> pre | <deviceJwt> post <liveCodeId>');
  process.exit(2);
}

const REST = `http://127.0.0.1:${REST_PORT}`;
const DEVICE = 'device-a'; // must match the JWT sub (RLS with-check)
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (90s)'), 90_000);

const hex64 = (pair) => pair.repeat(32);
const TOKEN_PRE          = hex64('d4'); // pre-DDL landing probe (never lands)
const TOKEN_DISCRIMINATED = hex64('a1'); // post: replica-failure override
const TOKEN_CONTROL       = hex64('b2'); // post: genuinely-unknown campaign
const TOKEN_DRAIN         = hex64('c3'); // post: drain leg, via shipped handler

// The card's FUTURE landing body — the shipped land-unverified shape
// (push-replication.js, byte-for-byte fields) plus whatever `extra` carries.
const landBody = (id, tokenHash, extra = {}) => ({
  id,
  code_id: null,
  token_hash: tokenHash,
  device_id: DEVICE,
  scanned_at: new Date().toISOString(),
  status: 'accepted',
  reason: null,
  offline_override: true,
  override_by: 'spike-03',
  unverified_code: true,
  pos_order_number: null,
  pos_business_date: new Date().toISOString().slice(0, 10),
  redeemed_value: null,
  ...extra,
});

async function postLanding(body) {
  const res = await fetch(`${REST}/scan_attempts`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${JWT}`, 'Content-Type': 'application/json', Prefer: 'return=minimal' },
    body: JSON.stringify(body),
  });
  const text = res.status === 201 ? '' : await res.text();
  return { status: res.status, text };
}

addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `spike_discriminator_${MODE}_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});

if (MODE === 'pre') {
  // ── leg 1: the shipped local write path's treatment of the field ──────────
  await db.addCollections(scanAttemptsCollectionSpec());
  const col = db.scan_attempts;

  console.log('── leg 1: shipped enqueueAttempt handed policy_unresolved:true ──');
  const { doc } = await enqueueAttempt(col, {
    code_id: hex64('e5'), device_id: DEVICE,
    offline_override: true, unverified_code: true,
    policy_unresolved: true, // ← the field under test
  });
  const stored = doc.toJSON();
  const enqueueKept = Object.prototype.hasOwnProperty.call(stored, 'policy_unresolved');
  console.log(`  stored local row carries policy_unresolved: ${enqueueKept} — ${enqueueKept ? 'no enqueue plumbing needed' : 'DROPPED at the destructure; the card owes enqueue plumbing'}`);

  console.log('\n── leg 2: direct insert on the SHIPPED schema under ajv validation ──');
  let schemaVerdict;
  try {
    await col.insert({ ...stored, id: crypto.randomUUID(), code_id: hex64('e6'), policy_unresolved: true });
    const back = await col.findOne({ selector: { code_id: hex64('e6') } }).exec();
    const kept = back && Object.prototype.hasOwnProperty.call(back.toJSON(), 'policy_unresolved');
    schemaVerdict = kept ? 'ACCEPTED and stored — no schema bump needed' : 'accepted but STRIPPED — schema bump needed';
  } catch (e) {
    schemaVerdict = `REJECTED (${String(e.message || e).slice(0, 120)}…) — the card owes a schema version bump + device migration`;
  }
  console.log(`  ${schemaVerdict}`);

  console.log('\n── leg 3: the extended landing body against the PRE-migration server ──');
  const pre = await postLanding(landBody(crypto.randomUUID(), TOKEN_PRE, { policy_unresolved: true }));
  console.log(`  HTTP ${pre.status}  ${pre.text.slice(0, 160)}`);
  if (pre.status >= 200 && pre.status < 300) {
    fail('the pre-migration server ACCEPTED the unknown column — either the DDL leaked from a prior run (reset_bare failed) or the premise is wrong');
  }
  console.log('  → non-2xx measured. The shipped push handler THROWS on any non-2xx landing and RxDB');
  console.log('    retries the batch (the F-2 measured loop: 12 attempts, 0 landings, queue stranded).');
  console.log('    BUILD-FACT: the server migration must land BEFORE any client sends the field —');
  console.log('    a new client against a stale server reproduces the poison class.');
} else {
  // ── POST mode: column exists, PostgREST reloaded ──────────────────────────
  console.log('── leg 4: the discriminated override lands ──');
  const disc = await postLanding(landBody(crypto.randomUUID(), TOKEN_DISCRIMINATED, { policy_unresolved: true }));
  console.log(`  HTTP ${disc.status}  ${disc.text.slice(0, 120)}`);
  if (disc.status !== 201) fail(`discriminated landing answered HTTP ${disc.status} — the discriminator cannot land even with the column present`);

  console.log('\n── leg 5: the control — genuinely-unknown campaign, shipped body, no field ──');
  const ctl = await postLanding(landBody(crypto.randomUUID(), TOKEN_CONTROL));
  console.log(`  HTTP ${ctl.status}  ${ctl.text.slice(0, 120)}`);
  if (ctl.status !== 201) fail(`control landing answered HTTP ${ctl.status}`);

  console.log('\n── leg 6: the shipped handler drains an EXTENDED local queue (compat + no head-of-line) ──');
  // The card's future client: the shipped schema + the discriminator field.
  const extSchema = structuredClone(SCAN_ATTEMPTS_SCHEMA);
  extSchema.properties.policy_unresolved = { type: 'boolean' };
  await db.addCollections({
    scan_attempts: { schema: extSchema },
    codes: {
      schema: {
        version: 0, primaryKey: 'id', type: 'object',
        properties: {
          id: { type: 'string', maxLength: 100 },
          redeemed_by: { type: ['string', 'null'] },
          redeemed_at: { type: ['string', 'null'] },
        },
        required: ['id'],
      },
    },
  });
  const attempts = db.scan_attempts;

  // Discriminated unverified attempt FIRST (the head-of-line position)…
  const discLocal = await attempts.insert({
    id: crypto.randomUUID(), code_id: TOKEN_DRAIN, device_id: DEVICE,
    scanned_at: new Date().toISOString(), status: 'pending', reason: null,
    offline_override: true, override_by: 'spike-03', unverified_code: true,
    pos_order_number: null, pos_business_date: new Date().toISOString().slice(0, 10),
    redeemed_value: null, burn_ok: null, burn_reason: null, landed: false,
    winner_device: null, winner_at: null,
    policy_unresolved: true, // held locally by the extended schema
  });
  // …then a legitimate redeem on the freshly-minted live code behind it.
  const { doc: legit } = await enqueueAttempt(attempts, { code_id: LIVE_CODE_ID, device_id: DEVICE });

  const requestLog = [];
  const handler = makePushHandler({
    restUrl: REST, bearer: JWT, deviceId: DEVICE, fetchImpl: fetch,
    attemptsCollection: attempts, codesCollection: db.codes,
    requestLog, winnerWaitMs: 2000,
  });
  try {
    await handler([
      { newDocumentState: discLocal.toJSON() },
      { newDocumentState: legit.toJSON() },
    ]);
  } catch (e) {
    fail(`the shipped handler THREW on the extended queue: ${e.message} — head-of-line, the poison class`);
  }

  console.log('  request log (enumerated):');
  for (const r of requestLog) console.log(`    ${r.kind.padEnd(16)} code_id=${String(r.code_id).slice(0, 16)}…`);
  const kinds = requestLog.map((r) => r.kind);
  if (kinds.filter((k) => k === 'redeem').length !== 1) fail(`expected exactly 1 redeem call (the legit attempt), saw ${kinds.filter((k) => k === 'redeem').length}`);
  if (kinds[0] !== 'land-unverified') fail(`expected the discriminated attempt to take the F-2 distinct landing path first, saw ${kinds[0]}`);

  const discAfter = (await attempts.findOne(discLocal.id).exec()).toJSON();
  const legitAfter = (await attempts.findOne(legit.id).exec()).toJSON();
  console.log(`  discriminated: status=${discAfter.status} landed=${discAfter.landed} policy_unresolved(local)=${discAfter.policy_unresolved}`);
  console.log(`  legit:         status=${legitAfter.status} landed=${legitAfter.landed} burn_ok=${legitAfter.burn_ok}`);
  if (discAfter.status !== 'accepted' || !discAfter.landed) fail('the discriminated attempt did not resolve accepted+landed');
  if (legitAfter.status !== 'accepted' || !legitAfter.landed) fail('the legitimate attempt behind it did not land — head-of-line');
  console.log('  → the shipped handler is FORWARD-COMPATIBLE with the extended local row, and its');
  console.log('    whitelisted landing body provably drops the field (the c3 server row reads back');
  console.log('    policy_unresolved=false in the .sh readback) — the card owes the ONE landing-body line.');
}

await db.close();
clearTimeout(hardTimeout);
process.exit(0);
