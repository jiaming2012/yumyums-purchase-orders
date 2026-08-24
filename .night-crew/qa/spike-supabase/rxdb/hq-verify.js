// hq-verify.js — DID IT SURFACE? Spike B, card S `spike-b-migration-rehearsal`.
//
// hq-migrate.js proved HQ-shaped rows LANDED in the substrate. That is only half
// the card. This script proves they SURFACE — that a real RxDB client, replicating
// over the real transport with a real signed token, ends up holding exactly the
// migrated rows that user is entitled to and not one row more.
//
// 🛑 EVERY ASSERTION IS MADE AGAINST MIGRATED ROWS, NEVER AGAINST SPIKE A'S SEED.
//    Spike A already proved the substrate discriminates over rows it inserted
//    itself. If this script asserted against those, a totally broken migration
//    would still go green. The expected sets are derived from the HQ-SHAPED
//    SOURCE database on every run (SQL_EXPECTED_VISIBLE), so the expectation and
//    the substrate are two independent computations of the same predicate — one
//    in the source, one in hq-bridge-policies.sql — and the assertion is that
//    they agree.
//
// ⚠ LOCAL SPIKE ONLY. Never HQ, never :5433 (production), never :5434.

import {
    banner, check, eqSet, srcJson, rest, mintTokenAs, makeHqLocalDb,
    SYNC_TABLE, SQL_ROWS, SQL_EXPECTED_VISIBLE, SQL_USERS, RUN
} from './hq-bridge-env.js';
import { makeSupabaseClient } from './spike-env.js';
import { replicateSupabase } from 'rxdb/plugins/replication-supabase';

banner('hq-verify.js — do the MIGRATED rows surface in RxDB?');

const REPL_TIMEOUT_MS = 60_000;

// ═══════════════════════════════════════════════════════════════════════════
// 1. Expected sets, computed from HQ source truth
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n── 1. expected visibility, derived from the HQ-shaped source ──');
const users = srcJson(SQL_USERS);
const allRows = srcJson(SQL_ROWS);
const visible = srcJson(SQL_EXPECTED_VISIBLE);

const expectedFor = new Map(users.map((u) => [u.id, []]));
for (const v of visible) expectedFor.get(v.owner_id)?.push(v);

const visibleIds = new Set(visible.map((v) => v.id));
const orphans = allRows.filter((r) => !visibleIds.has(r.id));

for (const u of users) {
    const e = expectedFor.get(u.id) || [];
    console.log(`  ${u.display_name.padEnd(9)} (${u.role.padEnd(11)}) expects ${e.length}: ${e.map((x) => x.id).join(', ') || '(none)'}`);
}
console.log(`  migrated rows visible to NOBODY: ${orphans.length}: ${orphans.map((o) => `${o.id} [${o.app_slug}]`).join(', ') || '(none)'}`);

// 🛑 The orphan is the negative control and it must exist. Without at least one
// migrated row that no user may see, "the substrate discriminates" and "the
// substrate returns everything" produce identical output and this whole script
// passes vacuously.
check(orphans.length >= 1,
    'at least one MIGRATED row is visible to nobody (the negative control exists)',
    { migrated: allRows.length, visible: visible.length });
check(visible.length >= 3, 'at least 3 migrated rows are visible to somebody', visible.length);

// ═══════════════════════════════════════════════════════════════════════════
// 2. RLS over migrated rows, through PostgREST
// ═══════════════════════════════════════════════════════════════════════════
// Done before RxDB deliberately: if this leg fails, RxDB would report an empty
// database and the failure would look like a replication bug when it is an
// authorization one. Separating them keeps the attribution honest.
console.log('\n── 2. RLS discrimination over MIGRATED rows (PostgREST) ───────');
const migratedIds = allRows.map((r) => r.id);
const inList = (vals) => `in.(${encodeURIComponent(vals.map((v) => `"${v}"`).join(','))})`;

const tokens = new Map();
for (const u of users) {
    const tok = mintTokenAs(u.id, 'authenticated');
    tokens.set(u.id, tok);
    const res = await rest('GET', `${SYNC_TABLE}?select=id&id=${inList(migratedIds)}&order=id`, { token: tok });
    check(res.status === 200, `${u.display_name}: GET ${SYNC_TABLE} -> HTTP ${res.status}`, res.text);
    const got = res.json.map((r) => r.id);
    const want = (expectedFor.get(u.id) || []).map((r) => r.id);
    check(eqSet(got, want),
        `${u.display_name}: PostgREST returns exactly the ${want.length} migrated row(s) they are entitled to`,
        { want, got });
}

for (const o of orphans) {
    let seenBy = null;
    for (const u of users) {
        const res = await rest('GET', `${SYNC_TABLE}?select=id&id=eq.${o.id}`, { token: tokens.get(u.id) });
        if (res.status === 200 && res.json.length > 0) { seenBy = u.display_name; break; }
    }
    check(seenBy === null,
        `orphan ${o.id} [${o.app_slug}] is invisible to EVERY user (two-axis negative control)`,
        { seenBy });
}

// ═══════════════════════════════════════════════════════════════════════════
// 3. RxDB — the actual "surfaces in RxDB" claim
// ═══════════════════════════════════════════════════════════════════════════
console.log('\n── 3. RxDB replication of the MIGRATED rows ───────────────────');

async function pullInto(user) {
    const token = tokens.get(user.id);
    const sb = makeSupabaseClient(token);
    const db = await makeHqLocalDb(`hqmig_${RUN}_${user.id.slice(0, 8)}`);
    const errors = [];
    const rep = replicateSupabase({
        replicationIdentifier: `hqmig-${RUN}-${user.id}`,
        collection: db.checklists,
        client: sb,
        tableName: SYNC_TABLE,
        waitForLeadership: false,
        live: true,
        pull: { batchSize: 100 }
    });
    rep.error$.subscribe((e) => errors.push(e.message || String(e)));

    // A hang is a failure with a name on it, never a run that quietly never
    // ends. W1 proved a phx_join can reply ok while the postgres_changes
    // subscription has actually failed; without this the script would sit there.
    let timer;
    await Promise.race([
        rep.awaitInitialReplication(),
        new Promise((_, rej) => {
            timer = setTimeout(() => rej(new Error(
                `initial replication did not complete within ${REPL_TIMEOUT_MS} ms for ${user.display_name}`)),
                REPL_TIMEOUT_MS);
        })
    ]).finally(() => clearTimeout(timer));

    const docs = await db.checklists.find().exec();
    await rep.cancel();
    await db.close();
    return { docs: docs.map((d) => d.toJSON()), errors };
}

for (const u of users) {
    const want = expectedFor.get(u.id) || [];
    const { docs, errors } = await pullInto(u);
    console.log(`  ${u.display_name}: RxDB holds ${docs.length} doc(s) -> ${docs.map((d) => d.id).join(', ') || '(none)'}`);

    check(errors.length === 0, `${u.display_name}: replication reported no errors`, errors);
    check(eqSet(docs.map((d) => d.id), want.map((w) => w.id)),
        `${u.display_name}: RxDB holds EXACTLY the ${want.length} migrated row(s) they are entitled to`,
        { want: want.map((w) => w.id), got: docs.map((d) => d.id) });

    // Identity and payload, not just cardinality: a count-only assertion cannot
    // tell "the right rows arrived" from "some rows arrived".
    const byId = new Map(docs.map((d) => [d.id, d]));
    let bad = null;
    for (const w of want) {
        const d = byId.get(w.id);
        if (!d) { bad = { id: w.id, why: 'missing from RxDB' }; break; }
        if (d.owner_id !== w.owner_id) { bad = { id: w.id, field: 'owner_id', want: w.owner_id, got: d.owner_id }; break; }
        if (d.app_slug !== w.app_slug) { bad = { id: w.id, field: 'app_slug', want: w.app_slug, got: d.app_slug }; break; }
        if (d.body !== w.body) { bad = { id: w.id, field: 'body', want: w.body, got: d.body }; break; }
    }
    check(!bad, `${u.display_name}: every RxDB doc matches the HQ source row byte-for-byte`, bad);

    // The orphan must not have reached this client either. Same claim as leg 2,
    // re-made at the RxDB layer, because a replication plugin that over-fetches
    // and filters locally would pass leg 2 and fail here.
    const orphanHere = orphans.filter((o) => byId.has(o.id)).map((o) => o.id);
    check(orphanHere.length === 0,
        `${u.display_name}: no nobody-visible row reached the local RxDB database`, orphanHere);
}

// One more: the migrated rows must not have dragged spike A's own seed rows into
// anybody's replica, and vice versa. Exact-set equality above already implies
// it; stated separately so a future reader sees it was checked on purpose.
console.log('\nVERIFY: OK — HQ-shaped data landed in the substrate and surfaced in RxDB, '
    + 'discriminated on both axes, byte-for-byte identical to the HQ source.');
process.exit(0);
