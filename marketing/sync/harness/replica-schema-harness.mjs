// marketing/sync/harness/replica-schema-harness.mjs — the codes/offers
// replica-schema gate (card test-integrity-fix, run 20261003; B-465 and
// decision-191 rider 1; goal-ledger spike
// required-campaign-id-needs-a-replica-migration is the recipe).
//
// MARKETING_REPLICA_SCHEMA went to version 1 with `campaign_id` in `required`.
// This is the test that the change cannot brick a phone that already synced,
// and that the invariant `[SP-03b]` used to claim is now a schema fact:
//
//   1. A store created at the v0 shape, holding rows in BOTH `codes` and
//      `offers`, is closed and REOPENED through the SHIPPED
//      marketingCollectionSpec(). Every row is still there, unchanged, at
//      schema version 1.
//   2. At v1 a row WITHOUT `campaign_id` is refused on insert with VD2, in
//      both collections — a code that names no campaign cannot be a row in
//      the replica.
//   3. CONTROL — why it had to be a migration: the same widened `required` at
//      the SAME version 0 is refused on reopen with DB6.
//   4. CONTROL — why every builder must take the entry from
//      marketingCollectionSpec(): the bare v1 schema with no
//      migrationStrategies is refused by addCollections.
//
// 🛑 WHAT THIS DOES NOT PROVE. Storage here is RxDB's MEMORY storage wrapped in
// the ajv validator — its collection state lives in a module-level map that
// close() keeps and only remove() drops, so the reopen is real, in-process.
// The DB6 check and the migration run above the storage layer, so the verdict
// is storage-agnostic by construction; but the phone uses DEXIE, and the phone
// wraps NO validator, so leg 2's VD2 is a fact about validating storages (every
// node harness), not about the browser. The in-page reopen is
// tests/marketing.spec.js [TI-02]; B-441 (the scan_attempts Dexie path) is
// untouched by this file.
//
// No substrate, no database, no network: node + the vendored RxDB only
// (resolved through ./node_modules, the same gitignored symlink the *-run.sh
// scripts create).
//
// 🛑 THE VERDICT IS THE EXIT STATUS, NEVER THE PROSE.
//   exit 0  every leg held.   exit 1  a leg failed.   exit 2  could not run.

let createRxDatabase, addRxPlugin, getRxStorageMemory, RxDBDevModePlugin, disableWarnings,
  RxDBMigrationSchemaPlugin, wrappedValidateAjvStorage;
try {
  ({ createRxDatabase, addRxPlugin } = await import('rxdb'));
  ({ getRxStorageMemory } = await import('rxdb/plugins/storage-memory'));
  ({ RxDBDevModePlugin, disableWarnings } = await import('rxdb/plugins/dev-mode'));
  ({ RxDBMigrationSchemaPlugin } = await import('rxdb/plugins/migration-schema'));
  ({ wrappedValidateAjvStorage } = await import('rxdb/plugins/validate-ajv'));
} catch (e) {
  console.error(`⚠ COULD-NOT-RUN — the vendored rxdb did not resolve (${e && e.code}): ${e && e.message}`);
  console.error('  npm ci in .night-crew/qa/spike-supabase/rxdb, then link it as marketing/sync/harness/node_modules');
  process.exit(2);
}
const {
  MARKETING_REPLICA_SCHEMA, MARKETING_MIGRATION_STRATEGIES, marketingCollectionSpec,
  CODES_COLLECTION, OFFERS_COLLECTION, CAMPAIGNS_COLLECTION,
} = await import('../replicas.js');

disableWarnings();
addRxPlugin(RxDBDevModePlugin);
addRxPlugin(RxDBMigrationSchemaPlugin);

const fail = (msg) => { console.error(`\nRED: ${msg}`); process.exit(1); };
const ok = (msg) => console.log(`  ✓ ${msg}`);
const firstLine = (e) => (String(e && e.message).split('\n').map((s) => s.trim()).find(Boolean) || '').slice(0, 120);
const hardTimeout = setTimeout(() => fail('hard timeout (60s) — a leg never finished'), 60_000);

// The schema as it shipped BEFORE this card (git ec2830c, marketing/sync/
// replicas.js:57-72) — a frozen literal, because the shipped object is v1 now
// and a v0 store can only be built from what v0 was.
const V0 = Object.freeze({
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
  },
  required: ['id', 'token_hash', 'expires_at', 'updated_at'],
  indexes: [['token_hash']],
});

// ── leg 0: the shipped schema is the three-part shape, and the frozen v0
//    literal differs from it ONLY by version + required ─────────────────────
console.log('── leg 0: the shipped schema ──');
{
  const S = MARKETING_REPLICA_SCHEMA;
  console.log(`  shipped MARKETING_REPLICA_SCHEMA: version=${S.version} required=[${S.required.join(',')}]`);
  if (S.version !== 1) fail(`expected version 1, got ${S.version}`);
  if (!S.required.includes('campaign_id')) fail('campaign_id is not in required');
  const same = (k) => JSON.stringify(S[k]) === JSON.stringify(V0[k]);
  for (const k of ['primaryKey', 'type', 'properties', 'indexes']) {
    if (!same(k)) fail(`the frozen v0 literal and the shipped schema disagree on '${k}' — v1 changed more than version + required, and the identity strategy may no longer be lossless`);
  }
  const widened = S.required.filter((r) => !V0.required.includes(r));
  const narrowed = V0.required.filter((r) => !S.required.includes(r));
  if (widened.join() !== 'campaign_id' || narrowed.length) fail(`required moved by more than + campaign_id (added [${widened}], removed [${narrowed}])`);
  const spec = marketingCollectionSpec();
  for (const col of [CODES_COLLECTION, OFFERS_COLLECTION]) {
    if (spec[col].schema !== S) fail(`${col} is not built from MARKETING_REPLICA_SCHEMA`);
    if (spec[col].migrationStrategies !== MARKETING_MIGRATION_STRATEGIES) fail(`marketingCollectionSpec() carries no migrationStrategies for '${col}'`);
    if (typeof spec[col].migrationStrategies[1] !== 'function') fail(`'${col}' has no strategy for version 1`);
  }
  ok('version 1, campaign_id required, strategies carried for BOTH codes and offers; v0→v1 differs by version + required only');
}

// ONE storage object across every open (see the header).
const storage = wrappedValidateAjvStorage({ storage: getRxStorageMemory() });
const NAME = `replica_schema_${Date.now()}`;
const row = (n, campaign) => ({
  id: `c0000000-0000-4000-8000-00000000000${n}`,
  token_hash: String(n).repeat(64),
  campaign_id: campaign,
  expires_at: '2028-01-01T00:00:00.000Z',
  redeemed_at: n === 2 ? '2026-09-02T00:00:00.000Z' : null,
  redeemed_by: n === 2 ? 'device-a' : null,
  updated_at: '2026-09-01T00:00:00.000Z',
});
const ROWS = [row(1, 'a0000000-0000-4000-8000-000000000001'), row(2, 'a0000000-0000-4000-8000-000000000002')];
const plain = (doc) => { const j = doc.toJSON(); delete j._deleted; return j; };

async function open(name, collections) {
  const db = await createRxDatabase({ name, storage });
  try { await db.addCollections(collections); } catch (e) { await db.close(); throw e; }
  return db;
}

// ── leg 1: v0 store with rows → reopened through the SHIPPED spec ───────────
console.log('\n── leg 1: a v0 store holding rows reopens at v1 through marketingCollectionSpec() ──');
{
  const db0 = await open(NAME, { [CODES_COLLECTION]: { schema: V0 }, [OFFERS_COLLECTION]: { schema: V0 } });
  for (const col of [CODES_COLLECTION, OFFERS_COLLECTION]) await db0[col].bulkInsert(ROWS);
  const n0 = (await db0[CODES_COLLECTION].find().exec()).length + (await db0[OFFERS_COLLECTION].find().exec()).length;
  await db0.close();   // NOT remove()
  if (n0 !== 4) fail(`v0 seed: expected 4 rows across codes+offers, got ${n0}`);
  ok(`v0 store '${NAME}': 2 rows in codes, 2 in offers; db.close()`);

  let db1;
  try {
    db1 = await open(NAME, marketingCollectionSpec());
  } catch (e) {
    fail(`the v0 store did NOT reopen through the shipped spec: code=${e && e.code} ${firstLine(e)} — this is the "Scanner failed to start" brick`);
  }
  for (const col of [CODES_COLLECTION, OFFERS_COLLECTION]) {
    if (db1[col].schema.version !== 1) fail(`${col} reopened at schema version ${db1[col].schema.version}, not 1`);
    const docs = (await db1[col].find({ sort: [{ id: 'asc' }] }).exec()).map(plain);
    if (docs.length !== ROWS.length) fail(`${col}: ${docs.length} row(s) after the reopen, expected ${ROWS.length} — the migration LOST rows`);
    for (let i = 0; i < ROWS.length; i += 1) {
      for (const k of Object.keys(ROWS[i])) {
        if (docs[i][k] !== ROWS[i][k]) fail(`${col} row ${ROWS[i].id}: field '${k}' is ${JSON.stringify(docs[i][k])} after migration, was ${JSON.stringify(ROWS[i][k])}`);
      }
    }
    ok(`${col}: ${docs.length}/${ROWS.length} rows present at schema.version=1, every field unchanged`);
  }
  if (!db1[CAMPAIGNS_COLLECTION]) fail('the shipped spec did not build the campaigns collection');

  // ── leg 2: at v1, a row that names NO campaign is refused (VD2) ───────────
  console.log('\n── leg 2: at v1 a row without campaign_id is refused — the [SP-03b] invariant as a schema fact ──');
  for (const col of [CODES_COLLECTION, OFFERS_COLLECTION]) {
    const { campaign_id, ...noCampaign } = { ...row(9, 'x') };
    let code = null;
    try {
      await db1[col].insert(noCampaign);
    } catch (e) { code = (e && e.code) || `no-code: ${firstLine(e)}`; }
    if (code === null) fail(`${col} ACCEPTED a row without campaign_id at v1 — the row [SP-03b] tested can still exist under a validating storage`);
    if (code !== 'VD2') fail(`${col} refused the row, but with ${code}, not VD2`);
    if (await db1[col].findOne(noCampaign.id).exec()) fail(`${col}: the refused row is in the collection anyway`);
    ok(`${col}: refused with VD2, nothing written`);
  }
  // …and a row WITH one is still accepted (the refusal is about the missing
  // field, not about writes).
  await db1[CODES_COLLECTION].insert(row(3, 'a0000000-0000-4000-8000-000000000001'));
  ok('codes: a row WITH campaign_id inserts at v1');
  await db1.close();
}

// ── leg 3 (control): the same `required` at the SAME version is DB6 ─────────
console.log('\n── leg 3 (control): widening `required` WITHOUT the version bump is refused on reopen ──');
{
  const name = `${NAME}_bare`;
  const db0 = await open(name, { [CODES_COLLECTION]: { schema: V0 } });
  await db0[CODES_COLLECTION].insert(ROWS[0]);
  await db0.close();
  const V0_REQ = { ...V0, required: [...MARKETING_REPLICA_SCHEMA.required] };
  let code = null;
  try {
    const db = await open(name, { [CODES_COLLECTION]: { schema: V0_REQ } });
    await db.close();
  } catch (e) { code = (e && e.code) || `no-code: ${firstLine(e)}`; }
  if (code === null) fail('RxDB ACCEPTED the widened required at version 0 — the migration would be unnecessary, and the ledger\'s premise wrong');
  if (code !== 'DB6') fail(`refused, but with ${code}, not DB6`);
  ok('refused with DB6 — a bare `required` edit bricks every store that holds the collection');
}

// ── leg 4 (control): v1 with NO strategies is refused ───────────────────────
console.log('\n── leg 4 (control): the bare v1 schema, no migrationStrategies, is refused by addCollections ──');
{
  let code = null;
  try {
    const db = await open(`${NAME}_nostrategy`, { [CODES_COLLECTION]: { schema: MARKETING_REPLICA_SCHEMA } });
    await db.close();
  } catch (e) { code = (e && e.code) || `no-code: ${firstLine(e)}`; }
  if (code === null) fail('RxDB accepted a v1 collection with no migration strategy — part (2) of the three-part shape would be optional');
  ok(`refused with ${code} — every builder must take the entry from marketingCollectionSpec()`);
}

clearTimeout(hardTimeout);
console.log('\nall legs held — memory storage + ajv; the Dexie path is tests/marketing.spec.js [TI-02]');
process.exit(0);
