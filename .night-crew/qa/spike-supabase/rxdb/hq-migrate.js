// hq-migrate.js — THE MIGRATION. Spike B, card S `spike-b-migration-rehearsal`.
//
// Reads HQ-shaped rows out of the throwaway `spike-b-hq` Postgres, transforms
// them into the substrate's replication contract, and LOADS THEM THROUGH
// POSTGREST — the substrate's own HTTP API — not by `docker exec psql` into the
// Supabase container.
//
// 🛑 THAT DISTINCTION IS THE CARD. Piping SQL straight into Supabase's Postgres
//    would prove that a Postgres accepts INSERTs, which nobody doubted. Going
//    through PostgREST with a signed token exercises the JWT bridge, the role
//    mapping, the grants and RLS — i.e. the substrate — which is the thing nine
//    nights of planning assumed and nothing had tested.
//
// TWO LANES, DELIBERATELY, AND THE SPLIT IS ITSELF A FINDING:
//
//   SERVICE LANE  (role=service_role)  the bulk migration.
//       A migration must carry rows their owner cannot read. Alice owns an
//       `inventory` submission and holds no live `inventory` grant, so a
//       migration running on ALICE'S token physically cannot insert her own row
//       — hq_sync_checklists_insert's WITH CHECK refuses it. A per-user-token
//       bulk migration is therefore not merely awkward, it is INCAPABLE of
//       moving a real dataset. The bulk lane has to be a service identity.
//       Measured on this stack: service_role has rolbypassrls=t and full table
//       grants from the supabase/postgres image's default privileges, so no
//       schema change was needed to enable it — the lane already exists.
//
//   USER LANE     (role=authenticated)  the ONGOING write path, rehearsed after
//       the bulk load with one positive and one negative, to show the migration
//       did not leave the door open behind it.
//
// ⚠ LOCAL SPIKE ONLY. Never HQ, never :5433 (production), never :5434.

import { writeFileSync } from 'node:fs';
import {
    banner, check, srcJson, rest, mintTokenAs,
    SYNC_TABLE, PROJ_TABLE, SQL_PROJECTION, SQL_ROWS, SQL_USERS, RUN, MANIFEST
} from './hq-bridge-env.js';

banner('hq-migrate.js — HQ-shaped Postgres  ->  Supabase substrate');

const inList = (vals) => `in.(${encodeURIComponent(vals.map((v) => `"${v}"`).join(','))})`;

// ═══════════════════════════════════════════════════════════════════════════
// 1. EXTRACT — read the HQ-shaped source
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n── 1. extract from the HQ-shaped source ───────────────────────');
const users = srcJson(SQL_USERS);
const projection = srcJson(SQL_PROJECTION);
const rows = srcJson(SQL_ROWS);

console.log(`  users      ${users.length}`);
console.log(`  projection ${projection.length}  (app_permissions x hq_apps x users, role tiers expanded)`);
console.log(`  rows       ${rows.length}  (checklist_submissions x checklist_templates, archived excluded)`);
for (const p of projection) console.log(`    grant  ${p.user_id}  ->  ${p.app_slug}`);
for (const r of rows) console.log(`    row    ${r.id}  owner=${r.owner_id}  app=${r.app_slug}`);

// A dataset that cannot fail an assertion is not a fixture. If the extraction
// silently returns nothing, everything downstream passes vacuously — the exact
// class this cycle exists to retire.
check(users.length >= 3, 'source has at least 3 users', users.length);
check(projection.length >= 3, 'projection resolved at least 3 (user, app) pairs', projection.length);
check(rows.length >= 4, 'source projected at least 4 migratable rows', rows.length);

// The archived-template filter must actually have removed something, or the
// filter is untested rather than working.
const totalSubs = Number(srcJson('select count(*)::int as n from checklist_submissions')[0].n);
check(rows.length < totalSubs,
    `archived-template filter dropped rows (${totalSubs} submissions -> ${rows.length} migratable)`,
    { totalSubs, migratable: rows.length });

// The role-tier expansion must actually have expanded something: at least one
// projection pair must come from a row that names no user at all.
const roleOnlyGrants = Number(srcJson(
    'select count(*)::int as n from app_permissions where user_id is null and role is not null')[0].n);
check(roleOnlyGrants > 0,
    `source carries role-tier grants that only the users-join can resolve (${roleOnlyGrants})`,
    roleOnlyGrants);

// ═══════════════════════════════════════════════════════════════════════════
// 2. RESET the migration target — scoped to exactly this fixture's keys
// ═══════════════════════════════════════════════════════════════════════════
// A rehearsal starts from nothing every time or its counts mean nothing. The
// delete is filtered to the ids and users this fixture owns, so spike A's own
// seed rows (chk-alice-*, hq-user-alice/bob) are untouched and its captures keep
// reproducing.
console.log('\n── 2. reset the migration target (scoped to this fixture) ─────');
const svc = mintTokenAs(`svc-migrator-${RUN}`, 'service_role');
const rowIds = rows.map((r) => r.id);
const projUsers = [...new Set(projection.map((p) => p.user_id))];

// Record every key this run is about to put into spike A's SHARED tables, before
// putting any of them there, so hq-reset.js can take them all back out even if
// this process dies in the middle of the load. See MANIFEST's note in
// hq-bridge-env.js for what goes wrong when a rehearsal leaves rows behind.
if (MANIFEST) {
    // The two user-lane probe ids are minted later (§4) but written here, so a
    // run dying between the probe insert and its cleanup still leaves them
    // inside hq-reset.js's scoped delete — a probe row that escapes the
    // manifest contaminates the shared substrate persistently and reds
    // TestJWTBridgeRLS's exact-row-set control (B-148).
    const probeIds = [`probe-ok-${RUN}`, `probe-refused-${RUN}`];
    writeFileSync(MANIFEST, JSON.stringify(
        { run: RUN, rowIds: [...rowIds, ...probeIds], projUsers }, null, 2));
    console.log(`  migrated-key manifest -> ${MANIFEST} (incl. ${probeIds.length} probe id(s))`);
}

const delRows = await rest('DELETE', `${SYNC_TABLE}?id=${inList(rowIds)}`, { token: svc });
check(delRows.status < 300, `DELETE ${SYNC_TABLE} (scoped) -> HTTP ${delRows.status}`, delRows.text);
const delProj = await rest('DELETE', `${PROJ_TABLE}?user_id=${inList(projUsers)}`, { token: svc });
check(delProj.status < 300, `DELETE ${PROJ_TABLE} (scoped) -> HTTP ${delProj.status}`, delProj.text);

const preRows = await rest('GET', `${SYNC_TABLE}?select=id&id=${inList(rowIds)}`, { token: svc });
check(preRows.status === 200 && preRows.json.length === 0,
    'target is empty of this fixture before the load', preRows.text);

// ═══════════════════════════════════════════════════════════════════════════
// 3. LOAD — service lane, through PostgREST
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n── 3. load through PostgREST (service lane) ───────────────────');

// Control plane first. hq_grant_projection is what the RLS entitlement axis
// reads live, so loading rows before their grants would make every read-back
// empty for reasons that had nothing to do with the data lane.
const projRes = await rest('POST', PROJ_TABLE,
    { token: svc, body: projection, prefer: 'return=representation' });
check(projRes.status === 201, `POST ${PROJ_TABLE} -> HTTP ${projRes.status}`, projRes.text);
check(projRes.json.length === projection.length,
    `${PROJ_TABLE} accepted all ${projection.length} pairs`, projRes.json.length);

const rowsRes = await rest('POST', SYNC_TABLE,
    { token: svc, body: rows, prefer: 'return=representation' });
check(rowsRes.status === 201, `POST ${SYNC_TABLE} -> HTTP ${rowsRes.status}`, rowsRes.text);
check(rowsRes.json.length === rows.length,
    `${SYNC_TABLE} accepted all ${rows.length} rows`, rowsRes.json.length);

// ═══════════════════════════════════════════════════════════════════════════
// 4. READ BACK through the substrate and compare against source truth
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n── 4. read back through PostgREST and diff against source ─────');
const back = await rest('GET',
    `${SYNC_TABLE}?select=id,owner_id,app_slug,body&id=${inList(rowIds)}&order=id`, { token: svc });
check(back.status === 200, `GET ${SYNC_TABLE} -> HTTP ${back.status}`, back.text);
check(back.json.length === rows.length,
    `substrate holds all ${rows.length} migrated rows`, back.json.length);

const bySrc = new Map(rows.map((r) => [r.id, r]));
let mismatch = null;
for (const got of back.json) {
    const want = bySrc.get(got.id);
    if (!want) { mismatch = { id: got.id, why: 'row in substrate is not in source' }; break; }
    for (const f of ['owner_id', 'app_slug', 'body']) {
        if (got[f] !== want[f]) { mismatch = { id: got.id, field: f, want: want[f], got: got[f] }; break; }
    }
    if (mismatch) break;
}
check(!mismatch, 'every migrated row matches source byte-for-byte (id, owner_id, app_slug, body)', mismatch);

// uuid -> text is the one real impedance mismatch in this migration. Assert the
// keys survived it intact rather than assuming a cast is free.
const uuidRe = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
check(back.json.every((r) => uuidRe.test(r.id) && uuidRe.test(r.owner_id)),
    'HQ uuid primary keys survived the cast into the text-keyed sync contract');

const projBack = await rest('GET',
    `${PROJ_TABLE}?select=user_id,app_slug&user_id=${inList(projUsers)}&order=user_id,app_slug`,
    { token: svc });
check(projBack.status === 200 && projBack.json.length === projection.length,
    `substrate holds all ${projection.length} projected grants`, projBack.text);

// ═══════════════════════════════════════════════════════════════════════════
// 5. USER-LANE REHEARSAL — the ongoing write path, after the bulk load
// ═══════════════════════════════════════════════════════════════════════════
// Not a repeat of spike A's attack variants: those ran against spike A's own
// seed. This runs against MIGRATED grants — i.e. it asks whether the projection
// this migration just wrote actually governs writes.
console.log('\n── 5. user-lane rehearsal against the MIGRATED grants ─────────');

// Pick a user who holds a grant, and an app they do NOT hold.
const grantsByUser = new Map();
for (const p of projection) {
    if (!grantsByUser.has(p.user_id)) grantsByUser.set(p.user_id, new Set());
    grantsByUser.get(p.user_id).add(p.app_slug);
}
const allSlugs = [...new Set(rows.map((r) => r.app_slug))];
let probeUser = null, heldSlug = null, unheldSlug = null;
for (const [uid, slugs] of grantsByUser) {
    const missing = allSlugs.find((s) => !slugs.has(s));
    if (missing) { probeUser = uid; heldSlug = [...slugs][0]; unheldSlug = missing; break; }
}
check(probeUser !== null,
    'fixture contains a user holding one app and not another (needed for the two-lane probe)',
    { grantsByUser: [...grantsByUser].map(([u, s]) => [u, [...s]]), allSlugs });
console.log(`  probe user ${probeUser}: holds '${heldSlug}', does NOT hold '${unheldSlug}'`);

const userTok = mintTokenAs(probeUser, 'authenticated');

const okId = `probe-ok-${RUN}`;
const okRes = await rest('POST', SYNC_TABLE, {
    token: userTok, prefer: 'return=representation',
    body: [{ id: okId, owner_id: probeUser, app_slug: heldSlug, body: 'user-lane probe (held app)' }]
});
check(okRes.status === 201,
    `POSITIVE: user writes to an app the MIGRATED projection grants -> HTTP ${okRes.status}`,
    okRes.text);

const noId = `probe-refused-${RUN}`;
const noRes = await rest('POST', SYNC_TABLE, {
    token: userTok, prefer: 'return=representation',
    body: [{ id: noId, owner_id: probeUser, app_slug: unheldSlug, body: 'user-lane probe (unheld app)' }]
});
check(noRes.status >= 400,
    `NEGATIVE: same user, app the migrated projection does NOT grant -> HTTP ${noRes.status} (refused)`,
    noRes.text);

// And the refusal must be a real refusal, not a 4xx that still wrote.
const landed = await rest('GET', `${SYNC_TABLE}?select=id&id=eq.${noId}`, { token: svc });
check(landed.status === 200 && landed.json.length === 0,
    'the refused row is genuinely absent from the substrate (not a 4xx that still wrote)',
    landed.text);

// Clean the positive probe row away so the RxDB assertions in hq-verify.js see
// exactly the migrated set and nothing else.
const cleanup = await rest('DELETE', `${SYNC_TABLE}?id=eq.${okId}`, { token: svc });
check(cleanup.status < 300, 'user-lane probe row removed', cleanup.text);

console.log('\nMIGRATE: OK — %d grants and %d HQ-shaped rows are in the substrate, '
    + 'loaded through PostgREST, byte-identical to source.', projection.length, rows.length);
process.exit(0);
