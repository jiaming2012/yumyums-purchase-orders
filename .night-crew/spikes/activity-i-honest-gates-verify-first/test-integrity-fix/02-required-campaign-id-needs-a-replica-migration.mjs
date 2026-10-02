// 02-required-campaign-id-needs-a-replica-migration.mjs — node half of the spike of the
// same name (see the .sh header for what each leg proves and what exit 0/1/2 mean).
// Imports rxdb exactly as marketing/sync/harness/campaigns-harness.mjs does: memory
// storage wrapped in the ajv validator, dev-mode + migration-schema plugins.
import { createRxDatabase, addRxPlugin } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin, disableWarnings } from 'rxdb/plugins/dev-mode';
import { RxDBMigrationSchemaPlugin } from 'rxdb/plugins/migration-schema';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { MARKETING_REPLICA_SCHEMA, CODES_COLLECTION } from '../../../../marketing/sync/replicas.js';

disableWarnings();
addRxPlugin(RxDBDevModePlugin);
addRxPlugin(RxDBMigrationSchemaPlugin);

const red = (msg) => { console.error(`\n  ✗ ${msg}`); process.exit(1); };
const cannot = (msg) => { console.error(`\n  ⚠ ${msg}`); process.exit(2); };
const ok = (msg) => console.log(`  ✓ ${msg}`);
const firstLine = (e, n) => (String(e && e.message).split('\n').map((s) => s.trim()).find(Boolean) || '').slice(0, n);

// ONE storage object across every open: rxdb's memory storage keeps collection state in
// a module-level map (COLLECTION_STATES); close() marks the instance closed and keeps the
// data, remove() deletes it. Reusing the same database NAME is what makes the reopen find it.
const storage = wrappedValidateAjvStorage({ storage: getRxStorageMemory() });
const NAME = `spike_i1_required_${Date.now()}`;
const COL = CODES_COLLECTION;

const V0 = MARKETING_REPLICA_SCHEMA;                       // shipped: version 0, campaign_id NOT required
const V0_REQ = { ...V0, required: [...V0.required, 'campaign_id'] };   // the "one-line edit"
const V1_REQ = { ...V0_REQ, version: 1 };                               // the rider as a migration

const ROW = {
  id: 'code-spike-i1',
  token_hash: 'a'.repeat(64),
  campaign_id: 'a0000000-0000-4000-8000-000000000002',
  expires_at: new Date(Date.now() + 86_400_000).toISOString(),
  redeemed_at: null,
  redeemed_by: null,
  updated_at: new Date().toISOString(),
};

async function open(schema, extra = {}) {
  const db = await createRxDatabase({ name: NAME, storage });
  try {
    await db.addCollections({ [COL]: { schema, ...extra } });
  } catch (e) {
    await db.close();
    throw e;
  }
  return db;
}

console.log(`shipped MARKETING_REPLICA_SCHEMA: version=${V0.version} required=[${V0.required.join(',')}]`);
if (V0.required.includes('campaign_id')) cannot('the shipped schema already requires campaign_id — the rider has landed; nothing to measure');

// (1) v0: insert one row with campaign_id, close (NOT remove)
console.log('\n── (1) open v0, insert one row with campaign_id, close ──');
{
  const db = await open(V0);
  await db[COL].insert(ROW);
  const n = (await db[COL].find().exec()).length;
  await db.close();
  ok(`inserted; ${n} row in '${COL}'; db.close() (not remove)`);
}

// (1b) the reopen is REAL: same schema, same name → the row is still there
console.log('\n── (1b) reopen at the same v0 schema — is the row still there? ──');
{
  const db = await open(V0);
  const doc = await db[COL].findOne(ROW.id).exec();
  await db.close();
  if (!doc) cannot('the memory storage did NOT keep the row across close/reopen — a persistent storage (Dexie + fake-indexeddb) would be needed for step 2 to mean anything');
  ok(`row present after close/reopen (campaign_id=${doc.campaign_id}) — close/reopen persists in-process`);
}

// (2) same version 0, required + campaign_id → RxDB must refuse
console.log('\n── (2) reopen at version 0 with required + campaign_id ──');
let code2 = null;
try {
  const db = await open(V0_REQ);
  const doc = await db[COL].findOne(ROW.id).exec();
  await db.close();
  red(`RxDB ACCEPTED the widened required at the same version (row ${doc ? 'present' : 'absent'}) — the rider WOULD be a one-line edit; the ledger's sizing premise is wrong`);
} catch (e) {
  code2 = e && e.code;
  const prev = e?.parameters?.previousSchemaHash, next = e?.parameters?.schemaHash;
  ok(`refused: rxdb error code=${code2} (${firstLine(e, 110)})`);
  if (prev || next) ok(`previousSchemaHash=${prev} schemaHash=${next}`);
  if (!code2) red(`the refusal is not an RxError (no .code): ${e}`);
}

// (3) version 1 + required + migrationStrategies → the row is carried over
console.log('\n── (3) reopen at version 1 with required + campaign_id and migrationStrategies {1: d => d} ──');
let row3 = null;
{
  let db;
  try {
    db = await open(V1_REQ, { migrationStrategies: { 1: (d) => d } });
  } catch (e) {
    red(`version 1 + migrationStrategies was REFUSED: code=${e && e.code} ${firstLine(e, 160)}`);
  }
  const doc = await db[COL].findOne(ROW.id).exec();
  row3 = doc ? doc.toJSON() : null;
  if (!doc) { await db.close(); red('version 1 opened but the v0 row did NOT migrate — the identity strategy lost it'); }
  ok(`migrated: row present at v1, campaign_id=${doc.campaign_id}, schema.version=${db[COL].schema.version}`);

  // (4) at v1 a row naming NO campaign cannot exist on the device
  console.log('\n── (4) at version 1, insert a row WITHOUT campaign_id ──');
  try {
    const { campaign_id, ...noCampaign } = { ...ROW, id: 'code-spike-i1-nocampaign', token_hash: 'b'.repeat(64) };
    await db[COL].insert(noCampaign);
    await db.close();
    red('a row WITHOUT campaign_id was ACCEPTED at v1 — [SP-03b]\'s row can still exist on the device; the retirement rationale is wrong');
  } catch (e) {
    ok(`refused: rxdb error code=${e && e.code} (${firstLine(e, 110)})`);
  }
  await db.close();
}

console.log(`\nsummary: (2) code=${code2}  (3) row migrated=${!!row3}  (4) campaign_id-less row refused at v1`);
process.exit(0);
