// hq-bridge-env.js — everything hq-migrate.js and hq-verify.js need in common.
//
// Card S `spike-b-migration-rehearsal` (Spike B). Lives inside rxdb/ rather than
// a directory of its own for one concrete reason: Node resolves bare specifiers
// (`rxdb`, `@supabase/supabase-js`) by walking UP from the importing file, and
// the spike's node_modules is `rxdb/node_modules`. A sibling directory would not
// see it. spike-env.js is imported, never modified — spike A's GREEN verdict has
// to keep reproducing byte-for-byte after this card lands.
//
// ⚠ LOCAL SPIKE ONLY. Talks to the throwaway `spike-supabase` stack and to the
//   throwaway `spike-b-hq` scratch Postgres. Never HQ, never :5433 (that cluster
//   is PRODUCTION), never :5434.

import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { addRxPlugin, createRxDatabase } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';
import { JWT_SECRET, REST_PORT, REALTIME_PORT, DB_PORT } from './spike-env.js';

export { REST_PORT, REALTIME_PORT, DB_PORT };

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SPIKE_DIR = path.resolve(HERE, '..');

// The two substrate tables Spike B migrates INTO. Both are spike A's
// (sql/hq-bridge-fixture.sql + sql/hq-bridge-policies.sql) and neither is
// modified by this card — the migration is the only new thing.
export const SYNC_TABLE = 'hq_sync_checklists';
export const PROJ_TABLE = 'hq_grant_projection';

// The scratch HQ-shaped Postgres container, handed down by spike-b-migration.sh.
// Deliberately NOT discovered by name-guessing here: the script owns the
// container's lifetime and passes its id, so these scripts can never attach to
// something they did not create.
export const HQ_CID = process.env.SPIKE_B_HQ_CID || '';
if (!HQ_CID) {
    console.error('hq-bridge-env: SPIKE_B_HQ_CID is unset — run this through '
        + '.night-crew/qa/spike-supabase/spike-b-migration.sh, which creates the '
        + 'scratch HQ-shaped Postgres and exports its container id.');
    process.exit(2);
}

export const RUN = process.env.SPIKE_B_RUN_ID || `b${Date.now()}`;

// ---------------------------------------------------------------------------
// 1. Reading the HQ-shaped source.
//
// Over `docker exec psql`, not a JS Postgres driver, for the same reason
// spike-env.js mints its token by shelling out to Go: adding a dependency to
// prove a thing does not need one is noise. It also means the host needs no
// Postgres client, exactly as env-up.sh notes for the Supabase side.
// ---------------------------------------------------------------------------
export function srcPsql(sql) {
    try {
        return execFileSync(
            'docker',
            ['exec', '-i', HQ_CID, 'psql', '-U', 'hq', '-d', 'hq_source', '-t', '-A', '-c', sql],
            { encoding: 'utf8', maxBuffer: 32 * 1024 * 1024 }
        ).trim();
    } catch (e) {
        // Surface psql's OWN diagnostic, not Node's 40-line execFileSync dump
        // with the whole query re-quoted inside an Error message. When a spike
        // reds, the reason has to be readable in the captured log — that log IS
        // the deliverable.
        const why = (e.stderr || '').trim() || e.message;
        throw new SpikeFailure(`psql against the HQ-shaped source failed:\n${why}\n--- query ---\n${sql.trim()}`);
    }
}

// json_agg the whole result so the transport is JSON rather than delimiter
// parsing. coalesce to '[]' so an empty result is an empty array, not the empty
// string — a silent [] and a silent '' are exactly the pair that lets a broken
// extraction look like a working one with no data.
export function srcJson(sql) {
    const out = srcPsql(`select coalesce(json_agg(t), '[]'::json)::text from (${sql}) t`);
    return JSON.parse(out || '[]');
}

// ---------------------------------------------------------------------------
// 2. The TRANSFORM. This is the actual migration logic and it is deliberately
//    all here, in SQL, in one place a reviewer can read.
// ---------------------------------------------------------------------------

// app_permissions ⋈ hq_apps ⋈ users  →  flat (user_id, app_slug) pairs.
//
// 🛑 THE `users` JOIN IS THE LOAD-BEARING HALF. HQ grants an app EITHER to a
// role tier (role set, user_id null) OR to one user. hq_grant_projection is
// flat, so role-tier grants must be EXPANDED through users.role. A migration
// that only copied the user-shaped rows passes every test written against a
// user-only fixture and drops most real grants in production.
export const SQL_PROJECTION = `
  select u.id::text as user_id, a.slug as app_slug
  from app_permissions p
  join hq_apps a on a.id = p.app_id and a.enabled
  join users u
    on (p.user_id is not null and u.id = p.user_id)
    or (p.role    is not null and u.role = p.role)
  group by 1, 2
  order by 1, 2`;

// checklist_submissions ⋈ checklist_templates (⋈ submission_responses) →
// the hq_sync_checklists replication shape (id, owner_id, app_slug, body).
//
// Three things this is NOT: it is not a column copy (body is a JOIN-derived
// aggregate), it is not unfiltered (archived templates are excluded), and it is
// not blind to HQ's draft shape (`r.submission_id = s.id` naturally excludes the
// submission_id IS NULL draft rows, which belong to no submission).
//
// uuid → text is the real impedance mismatch of this migration: HQ's keys are
// uuid, the sync contract's primary key is text. `::text` is the whole of it,
// and the assertions downstream check that identity survived it.
export const SQL_ROWS = `
  select s.id::text           as id,
         s.submitted_by::text as owner_id,
         t.app_slug           as app_slug,
         json_build_object(
           'template',          t.name,
           'status',            s.status,
           'submitted_at',      s.submitted_at::text,
           'responses',         (select count(*) from submission_responses r
                                  where r.submission_id = s.id),
           -- jsonb_array_length, NOT json_array_length: template_snapshot is
           -- JSONB in HQ (0011_checklist_submissions.sql) and the json_* family
           -- does not accept it. Caught by this script's own first run.
           'snapshot_sections', jsonb_array_length(s.template_snapshot -> 'sections')
         )::text as body
  from checklist_submissions s
  join checklist_templates t on t.id = s.template_id
  where t.archived_at is null
  order by s.id`;

// What each user MUST be able to see after the migration, derived from HQ source
// truth rather than hardcoded: own the row AND hold a live grant on its app.
// This is the source-side statement of the same predicate hq-bridge-policies.sql
// enforces in the substrate; the assertions compare the two.
export const SQL_EXPECTED_VISIBLE = `
  select r.owner_id, r.id, r.app_slug, r.body
  from (${SQL_ROWS}) r
  join (${SQL_PROJECTION}) g
    on g.user_id = r.owner_id and g.app_slug = r.app_slug
  order by r.owner_id, r.id`;

export const SQL_USERS = `select id::text as id, display_name, role from users order by id`;

// ---------------------------------------------------------------------------
// 3. Tokens. Minted by the Go minter, same as spike A — HQ's Go backend is the
//    token authority, and re-implementing HS256 in JS would prove nothing.
//
//    Spike B needs the `-role` flag spike-env.js's mintToken() does not expose,
//    because the migration runs on TWO lanes with two different Postgres roles.
//    Rather than edit spike A's file (read-only to this card) the flag is passed
//    here.
// ---------------------------------------------------------------------------
export function mintTokenAs(sub, role = 'authenticated', ttl = '30m') {
    return execFileSync(
        'go',
        ['run', './mintjwt', '-secret', JWT_SECRET, '-sub', sub, '-role', role, '-ttl', ttl],
        { cwd: SPIKE_DIR, encoding: 'utf8' }
    ).trim();
}

// ---------------------------------------------------------------------------
// 4. PostgREST. Every load and every read-back goes through the substrate's own
//    HTTP API — not `docker exec psql` into the Supabase container. That is the
//    difference between "rows exist in a Postgres that happens to be Supabase's"
//    and "rows landed in the substrate", and it is the whole claim of the card.
// ---------------------------------------------------------------------------
export const restUrl = (p) => `http://127.0.0.1:${REST_PORT}/${p}`;

export async function rest(method, p, { token, body, prefer } = {}) {
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers.Authorization = `Bearer ${token}`;
    if (prefer) headers.Prefer = prefer;
    const res = await fetch(restUrl(p), {
        method,
        headers,
        body: body === undefined ? undefined : JSON.stringify(body)
    });
    const text = await res.text();
    let json = null;
    try { json = text ? JSON.parse(text) : null; } catch { /* non-JSON body kept in .text */ }
    return { status: res.status, json, text };
}

// ---------------------------------------------------------------------------
// 5. The RxDB side.
//
//    The schema mirrors hq_sync_checklists' four data columns. `_deleted` and
//    `_modified` are NOT declared, for exactly the reasons spike-env.js records
//    for spike_notes: RxDB owns `_deleted` itself, and leaving `_modified`
//    undeclared keeps it a purely server-stamped pull cursor.
// ---------------------------------------------------------------------------
export const hqChecklistSchema = {
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        owner_id: { type: 'string', maxLength: 100 },
        app_slug: { type: 'string', maxLength: 100 },
        body: { type: 'string' }
    },
    required: ['id', 'owner_id', 'app_slug', 'body']
};

let devModeAdded = false;
export async function makeHqLocalDb(name) {
    if (!devModeAdded) { addRxPlugin(RxDBDevModePlugin); devModeAdded = true; }
    const db = await createRxDatabase({
        name,
        // Memory storage + the ajv validator wrapper. The wrapper is not
        // optional: with dev-mode on, RxDB throws DVM1 for a storage with no
        // top-level schema validator (spike A README half 2, sharp edge 9).
        storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
        ignoreDuplicate: true
    });
    await db.addCollections({ checklists: { schema: hqChecklistSchema } });
    return db;
}

// ---------------------------------------------------------------------------
// 6. Assertion primitive.
//
// 🛑 There is no `warn()` in this file and there must never be one. A step that
//    cannot decide is a FAILURE — spike A's env-up.sh:18-27 rule, carried
//    forward. `check` throws; the callers let it kill the process with a
//    non-zero exit. That exit status IS the verdict.
// ---------------------------------------------------------------------------
export class SpikeFailure extends Error {}

// A failed assertion must leave a READABLE line in the captured log, because the
// captured log is the spike's evidence. Node's default unhandled-rejection dump
// buries the reason under a stack trace; this prints the reason and exits 1 —
// and 1 is still 1, so the verdict is unchanged either way.
process.on('unhandledRejection', (e) => {
    console.error(`\n🛑 SPIKE FAIL: ${e && e.message ? e.message : e}`);
    if (e && !(e instanceof SpikeFailure) && e.stack) console.error(e.stack);
    process.exit(1);
});
process.on('uncaughtException', (e) => {
    console.error(`\n🛑 SPIKE FAIL: ${e && e.message ? e.message : e}`);
    if (e && !(e instanceof SpikeFailure) && e.stack) console.error(e.stack);
    process.exit(1);
});

export function check(cond, label, detail) {
    if (cond) {
        console.log(`  PASS  ${label}`);
        return;
    }
    console.log(`  FAIL  ${label}`);
    if (detail !== undefined) console.log(`        ${typeof detail === 'string' ? detail : JSON.stringify(detail)}`);
    throw new SpikeFailure(label);
}

export const sortIds = (a) => [...a].sort();
export const eqSet = (a, b) =>
    a.length === b.length && sortIds(a).every((v, i) => v === sortIds(b)[i]);

export function banner(what) {
    console.log(`# ${what}`);
    console.log(`# substrate: rest=${REST_PORT} realtime=${REALTIME_PORT} db=${DB_PORT}`);
    console.log(`# hq source container: ${HQ_CID}`);
    console.log(`# run: ${RUN}`);
}
