// flag-replicates.mjs — the RxDB half of spike 01. Two candidate mechanisms for
// getting `campaigns.requires_online` onto an offline device, run SIDE BY SIDE
// against the built Activity A schema, and then asked the question that decides
// the design: does a FLIP of the flag reach a replica that is already
// checkpointed?
//
//   Mechanism A — a `campaigns` pull replica (its own replicateRxCollection).
//   Mechanism B — the existing codes pull widened with PostgREST's FK embed
//                 `campaigns(requires_online)`, so the flag rides the code row.
//
// Both are driven by the SHIPPED marketing/sync/pull-replication.js — this
// spike imports the production module rather than re-implementing it, because
// the card's job is to extend that exact file.
//
// argv: <deviceJwt> <flipCampaignId> <dbContainerId>
// exit 0 a mechanism carried the flag AND the flip; exit 1 none did.

import { execFileSync } from 'node:child_process';
import { Subject } from 'rxjs';
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { replicateRxCollection } from 'rxdb/plugins/replication';
import { REST_PORT } from '../../../../qa/spike-supabase/rxdb/spike-env.js';
import {
  makePullHandler,
  startPullReplica,
  keysetPredicate,
  REPLICA_SELECT,
} from '../../../../../marketing/sync/pull-replication.js';

const [JWT, FLIP_CAMPAIGN, DB_CID] = process.argv.slice(2);
if (!JWT || !FLIP_CAMPAIGN || !DB_CID) {
  console.error('usage: flag-replicates.mjs <deviceJwt> <flipCampaignId> <dbContainerId>');
  process.exit(2);
}

const REST = `http://127.0.0.1:${REST_PORT}`;
const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const hardTimeout = setTimeout(() => fail('hard timeout (120s)'), 120_000);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

const sql = (text) => execFileSync(
  'docker',
  ['exec', '-i', DB_CID, 'psql', '-U', 'supabase_admin', '-d', 'postgres', '-v', 'ON_ERROR_STOP=1', '-qtA'],
  { input: text, encoding: 'utf8' },
).trim();

const FINDINGS = [];
const note = (s) => { FINDINGS.push(s); console.log(`  · ${s}`); };

// ---------------------------------------------------------------------------
// Leg 3 — can the SHIPPED makePullHandler serve `campaigns` unchanged?
//
// buildPullUrl() unconditionally appends `expires_at=gt.<windowIso>`; campaigns
// has no expires_at column. This is a build-fact the card needs BEFORE it plans
// the module change, so it is measured, not assumed.
// ---------------------------------------------------------------------------
console.log('\n── leg 3: the shipped makePullHandler against `campaigns`, unchanged ──');
{
  const h = makePullHandler({
    restUrl: REST,
    table: 'campaigns',
    windowBound: () => new Date(0).toISOString(),
    bearer: JWT,
    fetchImpl: fetch,
    select: 'id,requires_online,updated_at',
  });
  let outcome;
  try {
    await h(null, 10);
    outcome = 'HTTP 200 — the shipped handler serves campaigns as-is';
  } catch (e) {
    outcome = String(e.message || e);
  }
  note(`shipped handler on campaigns → ${outcome}`);
  if (!/HTTP 400/.test(outcome) && !/HTTP 200/.test(outcome)) {
    fail(`unexpected outcome from the shipped handler on campaigns: ${outcome}`);
  }
  if (/HTTP 400/.test(outcome)) {
    note('→ the card owes buildPullUrl an OPTIONAL bound (campaigns has no expires_at); '
       + 'the keyset checkpoint itself needs no change — proven below');
  }
}

// ---------------------------------------------------------------------------
// Local database — one per mechanism, both memory-storage with ajv validation.
// ---------------------------------------------------------------------------
addRxPlugin(RxDBDevModePlugin);
const db = await createRxDatabase({
  name: `spike_flag_${Date.now()}`,
  storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
});
await db.addCollections({
  // Mechanism A's on-device row: §10 minimal — the policy flag and nothing that
  // is not needed to apply it. No name, no face_value (PII-free by shape).
  campaigns: {
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: { type: 'string', maxLength: 100 },
        requires_online: { type: 'boolean' },
        updated_at: { type: 'string' },
      },
      required: ['id', 'requires_online', 'updated_at'],
    },
  },
  // Mechanism B: the shipped codes row plus the embedded campaign object.
  codes: {
    schema: {
      version: 0,
      primaryKey: 'id',
      type: 'object',
      properties: {
        id: { type: 'string', maxLength: 100 },
        token_hash: { type: 'string', maxLength: 128 },
        campaign_id: { type: 'string', maxLength: 100 },
        expires_at: { type: 'string' },
        redeemed_at: { type: ['string', 'null'] },
        redeemed_by: { type: ['string', 'null'] },
        updated_at: { type: 'string' },
        campaigns: {
          type: 'object',
          properties: { requires_online: { type: 'boolean' } },
        },
      },
      required: ['id', 'token_hash', 'expires_at', 'updated_at'],
      indexes: [['token_hash']],
    },
  },
});

// ---------------------------------------------------------------------------
// Mechanism A — the campaigns pull handler. Deliberately built from the SHIPPED
// keysetPredicate(): the GAP-1 compound checkpoint that closed the codes
// replica's silent-miss class carries over to campaigns UNCHANGED, and only the
// expiry bound has to become optional. That is the minimal module change, and
// it is proven here rather than designed on paper.
// ---------------------------------------------------------------------------
const logA = [];
function campaignsPullHandler(checkpoint, batchSize) {
  const url =
    `${REST}/campaigns` +
    `?select=${encodeURIComponent('id,requires_online,updated_at')}` +
    `&${keysetPredicate(checkpoint)}` +
    `&order=updated_at.asc,id.asc` +
    `&limit=${batchSize}`;
  logA.push(url);
  return fetch(url, { headers: { Authorization: `Bearer ${JWT}`, Accept: 'application/json' } })
    .then(async (res) => {
      if (res.status !== 200) throw new Error(`campaigns pull HTTP ${res.status}`);
      const rows = await res.json();
      const last = rows[rows.length - 1];
      return {
        documents: rows,
        checkpoint: rows.length
          ? { updated_at: last.updated_at, id: last.id }
          : (checkpoint || { updated_at: '1970-01-01T00:00:00+00:00', id: '00000000-0000-0000-0000-000000000000' }),
      };
    });
}

const streamA$ = new Subject();
const repA = startPullReplica({
  replicateRxCollection,
  collection: db.campaigns,
  replicationIdentifier: 'spike-campaigns-pull',
  pullHandler: campaignsPullHandler,
  stream$: streamA$,
  batchSize: 50,
});

// ---------------------------------------------------------------------------
// Mechanism B — the SHIPPED pull handler, select widened with the FK embed.
// ---------------------------------------------------------------------------
const logB = [];
const streamB$ = new Subject();
const repB = startPullReplica({
  replicateRxCollection,
  collection: db.codes,
  replicationIdentifier: 'spike-codes-embed-pull',
  pullHandler: makePullHandler({
    restUrl: REST,
    table: 'codes',
    windowBound: () => new Date(Date.now() - 2 * 24 * 3600 * 1000).toISOString(),
    bearer: JWT,
    fetchImpl: fetch,
    select: `${REPLICA_SELECT},campaigns(requires_online)`,
    requestLog: logB,
  }),
  stream$: streamB$,
  batchSize: 50,
});

repA.error$.subscribe((e) => console.error('  ! replica A error:', String(e?.message || e)));
repB.error$.subscribe((e) => console.error('  ! replica B error:', String(e?.message || e)));

console.log('\n── initial sync (both mechanisms) ──');
await repA.awaitInSync();
await repB.awaitInSync();

const readA = async () => (await db.campaigns.findOne(FLIP_CAMPAIGN).exec())?.requires_online;
const readB = async () => {
  const docs = await db.codes.find({ selector: { campaign_id: FLIP_CAMPAIGN } }).exec();
  return docs.length ? docs[0].campaigns?.requires_online : undefined;
};

const a0 = await readA();
const b0 = await readB();
const bRows = (await db.codes.find({ selector: { campaign_id: FLIP_CAMPAIGN } }).exec()).length;
note(`A (campaigns replica): campaign ${FLIP_CAMPAIGN.slice(-4)} landed, requires_online=${a0}`);
note(`B (codes + FK embed): ${bRows} code row(s) landed carrying campaigns.requires_online=${b0}`);
if (a0 !== false) fail(`mechanism A did not land the seeded flag (expected false, got ${a0})`);
if (b0 !== false) fail(`mechanism B did not land the embedded flag (expected false, got ${b0})`);

// ---------------------------------------------------------------------------
// FLIP #1 — the operator makes the campaign online-only, WITHOUT stamping
// updated_at. This is what a plain `update campaigns set requires_online=true`
// does today: campaigns has no touch trigger (enumerated by the shell half).
// ---------------------------------------------------------------------------
console.log('\n── flip #1: requires_online → true, updated_at NOT stamped ──');
sql(`update public.campaigns set requires_online = true where id = '${FLIP_CAMPAIGN}';`);
streamA$.next('RESYNC'); streamB$.next('RESYNC');
await repA.awaitInSync(); await repB.awaitInSync();
await sleep(500);
const a1 = await readA();
const b1 = await readB();
note(`after unstamped write + RESYNC → A=${a1}, B=${b1}`);
if (a1 !== false || b1 !== false) {
  note('UNEXPECTED: a mechanism saw an unstamped write — re-check the trigger enumeration');
} else {
  note('→ BOTH mechanisms are blind to an unstamped flag write: the checkpoint is '
     + 'updated_at, so a writer that does not advance it is invisible. The card owes '
     + 'the write path an explicit `updated_at = now()` (the same rule decision 163 '
     + 'put on redeem()) or a touch trigger on campaigns.');
}

// ---------------------------------------------------------------------------
// FLIP #2 — same change, updated_at stamped. First WITHOUT a nudge (does a
// pull replica poll? it must not — a device that polls burns the radio all
// shift), then with one.
// ---------------------------------------------------------------------------
console.log('\n── flip #2: requires_online → true WITH updated_at = now() ──');
// Mechanism B's fate turns on whether the CODE rows' own updated_at moves when
// the CAMPAIGN's policy changes. Measure it across the flip — "recently
// touched" would be true merely because the seed just ran, which would read as
// evidence for the opposite conclusion.
const codesMaxBefore = sql(
  `select coalesce(max(updated_at)::text,'(none)') from public.codes where campaign_id = '${FLIP_CAMPAIGN}';`,
);
sql(`update public.campaigns set requires_online = true, updated_at = now() where id = '${FLIP_CAMPAIGN}';`);

console.log('  (3s with NO RESYNC nudge — proving the replica does not poll)');
await sleep(3000);
const aNoNudge = await readA();
note(`after stamped write, no nudge → A=${aNoNudge}`);
if (aNoNudge !== false) {
  note('UNEXPECTED: replica A advanced without a nudge — it polls; re-derive the nudge requirement');
} else {
  note('→ confirmed: no nudge, no update. A campaigns replica NEEDS a nudge source, and '
     + '`campaigns` is NOT in the supabase_realtime publication (enumerated above) — so '
     + 'the card must either add it to the publication or fan the codes channel\'s RESYNC '
     + 'into the campaigns replica.');
}

streamA$.next('RESYNC'); streamB$.next('RESYNC');
await repA.awaitInSync(); await repB.awaitInSync();
await sleep(500);
const a2 = await readA();
const b2 = await readB();
note(`after stamped write + RESYNC → A=${a2}, B=${b2}`);

// ---------------------------------------------------------------------------
// The verdict.
// ---------------------------------------------------------------------------
console.log('\n── enumerated pull requests ──');
console.log(`  A (campaigns): ${logA.length} request(s)`);
logA.forEach((u, i) => console.log(`    A#${i + 1} ${u}`));
console.log(`  B (codes+embed): ${logB.length} request(s)`);
logB.forEach((r, i) => console.log(`    B#${i + 1} cp=${JSON.stringify(r.checkpoint)}`));

const codesMaxAfter = sql(
  `select coalesce(max(updated_at)::text,'(none)') from public.codes where campaign_id = '${FLIP_CAMPAIGN}';`,
);
const codesUpdatedAtMoved = codesMaxAfter !== codesMaxBefore;

console.log('\n── conclusion ──');
if (a2 !== true) {
  fail(`mechanism A did NOT carry the flip even with updated_at stamped and a nudge (A=${a2}). `
     + `Mechanism B: ${b2}. Neither candidate carries a policy change — the card's premise is false `
     + `and its design must change before any night is spent on it.`);
}
note('mechanism A (campaigns replica) CARRIES the flip — flag and change both reach the device');
if (b2 === true) {
  note('mechanism B ALSO carried the flip — unexpected; codes.updated_at must have moved. '
     + 'Both mechanisms are viable; choose on cost, not correctness.');
} else {
  note(`mechanism B is PERMANENTLY STALE — the embedded flag is re-read only when the CODE row's `
     + `own updated_at advances, and across the campaign flip it did NOT move `
     + `(max(codes.updated_at) before=${codesMaxBefore} after=${codesMaxAfter}, moved=${codesUpdatedAtMoved}). `
     + `A campaign whose policy changes while its codes sit still never re-delivers. `
     + `REJECT "embed the flag in the codes pull".`);
}

await repA.cancel();
await repB.cancel();
await db.close();
clearTimeout(hardTimeout);

console.log('\nFINDINGS');
FINDINGS.forEach((f) => console.log(`  - ${f}`));
process.exit(0);
