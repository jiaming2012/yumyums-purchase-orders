// hq-reset.js — put spike A's shared tables back exactly as this run found them.
//
// Card S `spike-b-migration-rehearsal` (Spike B). Run from spike-b-migration.sh's
// teardown, alongside the destruction of the scratch HQ-shaped Postgres.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 WHY A REHEARSAL HAS TO CLEAN UP AFTER ITSELF
//
// hq_sync_checklists and hq_grant_projection belong to spike A and are SHARED.
// backend/internal/sync/jwtbridge_rls_test.go's service_role CONTROL asserts an
// EXACT full-table row set — it is the control that proves the RLS variants are
// refusing rows that are genuinely present, so it cannot be a subset check.
// Every row this spike leaves behind therefore REDS A COMMITTED GO SUITE.
//
// That is not hypothetical. It was measured on this card's own first G2 run:
//
//   --- FAIL: TestJWTBridgeRLS/CONTROL/service_role_BYPASSRLS_proves_the_rows_are_there
//       service_role sees ALL rows: expected rows [chk-alice-inv-1 chk-alice-ops-1
//       chk-alice-ops-2 chk-bob-ops-1], got [0e000000-... 0e000000-... ... ]
//
// plus V9 and V12's controls and the post-variant control re-take — four
// subtests, all from leftover migrated rows and nothing else.
//
// The finding generalises past this card: ANY future work that writes into the
// spike substrate has to restore it, or it breaks a suite that has nothing to do
// with it. The scratch Postgres gets destroyed at the end of every run; the
// shared substrate gets reset at the end of every run; those are the same rule
// applied to the two halves.
// ═══════════════════════════════════════════════════════════════════════════
//
// Deletes ONLY the keys in the manifest hq-migrate.js wrote — never a bare
// unfiltered DELETE, so spike A's own seed (chk-alice-*, hq-user-alice/bob) is
// untouchable by construction and its captures keep reproducing.

import { readFileSync } from 'node:fs';
import { check, rest, mintTokenAs, SYNC_TABLE, PROJ_TABLE, MANIFEST, RUN } from './hq-bridge-env.js';

if (!MANIFEST) {
    console.error('hq-reset: SPIKE_B_MANIFEST is unset — nothing to reset from. '
        + 'Run this through spike-b-migration.sh.');
    process.exit(2);
}

let manifest;
try {
    manifest = JSON.parse(readFileSync(MANIFEST, 'utf8'));
} catch (e) {
    // No manifest means the run never reached the load, so it never wrote to the
    // shared tables. Nothing to undo; that is a clean no-op, not a failure.
    console.log(`hq-reset: no manifest at ${MANIFEST} (${e.code || e.message}) — `
        + 'the run never loaded anything into the substrate; nothing to reset.');
    process.exit(0);
}

const rowIds = manifest.rowIds || [];
const projUsers = manifest.projUsers || [];
console.log(`# hq-reset — removing ${rowIds.length} migrated row(s) and `
    + `${projUsers.length} projected user(s) written by run ${manifest.run || RUN}`);

const inList = (vals) => `in.(${encodeURIComponent(vals.map((v) => `"${v}"`).join(','))})`;
const svc = mintTokenAs(`svc-reset-${RUN}`, 'service_role');

if (rowIds.length) {
    const d = await rest('DELETE', `${SYNC_TABLE}?id=${inList(rowIds)}`, { token: svc });
    check(d.status < 300, `DELETE ${SYNC_TABLE} (scoped to the manifest) -> HTTP ${d.status}`, d.text);
    const left = await rest('GET', `${SYNC_TABLE}?select=id&id=${inList(rowIds)}`, { token: svc });
    check(left.status === 200 && left.json.length === 0,
        `no migrated row remains in ${SYNC_TABLE}`, left.text);
}

if (projUsers.length) {
    const d = await rest('DELETE', `${PROJ_TABLE}?user_id=${inList(projUsers)}`, { token: svc });
    check(d.status < 300, `DELETE ${PROJ_TABLE} (scoped to the manifest) -> HTTP ${d.status}`, d.text);
    const left = await rest('GET', `${PROJ_TABLE}?select=user_id&user_id=${inList(projUsers)}`, { token: svc });
    check(left.status === 200 && left.json.length === 0,
        `no migrated grant remains in ${PROJ_TABLE}`, left.text);
}

// Spike A's own seed must still be there. Deleting the fixture's keys and
// deleting the table are indistinguishable without this line.
const seed = await rest('GET', `${SYNC_TABLE}?select=id&order=id`, { token: svc });
check(seed.status === 200 && seed.json.length > 0,
    `spike A's seed survives the reset (${SYNC_TABLE} still holds ${seed.json?.length ?? '?'} row(s): `
    + `${(seed.json || []).map((r) => r.id).join(', ')})`, seed.text);

console.log('RESET: OK — spike A\'s shared substrate is back as this run found it.');
process.exit(0);
