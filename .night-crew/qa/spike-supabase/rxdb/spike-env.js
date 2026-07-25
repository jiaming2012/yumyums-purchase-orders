// spike-env.js — everything the three proof scripts need in common:
//   * resolve W1's stack's Docker-assigned host ports,
//   * mint an HS256 token with W1's Go minter (NOT a JS re-implementation),
//   * build a @supabase/supabase-js client that can talk to a Kong-less stack,
//   * define the one throwaway RxDB collection.
//
// ⚠ LOCAL SPIKE ONLY. See ../README.md for the banner. The JWT secret below is
//   the throwaway one committed in docker-compose.supabase.yml on purpose.
//
// NOTHING IN HERE IS A PROPOSAL FOR HQ. The collection is a throwaway shaped to
// W1's verified table contract, deliberately NOT HQ's checklist domain model —
// modelling the real domain is the card `sync-rxdb-schema-and-replication`.

import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createClient } from '@supabase/supabase-js';
import WS from 'ws';
import { createRxDatabase, addRxPlugin, defaultConflictHandler } from 'rxdb';
import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const SPIKE_DIR = path.resolve(HERE, '..'); // .night-crew/qa/spike-supabase
const REPO_ROOT = path.resolve(SPIKE_DIR, '..', '..', '..');

// Same throwaway value as docker-compose.supabase.yml's JWT_SECRET /
// PGRST_JWT_SECRET / API_JWT_SECRET. If they ever diverge, PostgREST answers
// 401 JWSError and Realtime answers a bare 403.
export const JWT_SECRET =
    '2508c659af3c4316b0a163a00725d33a9bc4eae75aa35ac9be6a007cacb8251c';

// Realtime is multi-tenant even with one tenant and resolves the tenant from
// the FIRST dot-separated label of the Host header (W1, sharp edge 6).
export const REALTIME_VHOST = 'realtime-dev.localhost';

export const TABLE = 'spike_notes';

// --------------------------------------------------------------------------
// 1. Ports — never hardcoded. W1's compose publishes bare container ports, so
//    Docker assigns the host side and it changes on every `up`.
// --------------------------------------------------------------------------
function composePort(service, containerPort) {
    const out = execFileSync(
        'docker',
        ['compose', '-p', 'spike-supabase', '-f', 'docker-compose.supabase.yml',
            'port', service, String(containerPort)],
        { cwd: REPO_ROOT, encoding: 'utf8' }
    ).trim();
    const port = out.split(':').pop();
    if (!port) throw new Error(`could not resolve host port for ${service}: ${out}`);
    return Number(port);
}

export const REST_PORT = composePort('rest', 3000);
export const REALTIME_PORT = composePort('realtime', 4000);
export const DB_PORT = composePort('db', 5432);

// --------------------------------------------------------------------------
// 2. Token — minted by W1's Go program, on purpose.
//    Re-implementing HS256 in JS here would prove nothing: the whole point of
//    the migration is that HQ's *Go backend* is the token authority.
// --------------------------------------------------------------------------
export function mintToken(sub, { ttl = '1h' } = {}) {
    return execFileSync(
        'go',
        ['run', './mintjwt', '-secret', JWT_SECRET, '-sub', sub, '-ttl', ttl],
        { cwd: SPIKE_DIR, encoding: 'utf8' }
    ).trim();
}

// --------------------------------------------------------------------------
// 3. The supabase-js client, bridged onto a gateway-less stack.
//
//    supabase-js assumes ONE origin fronted by Kong: it derives `<url>/rest/v1`
//    and `<url>/realtime/v1` from a single base URL, both fixed in the
//    constructor. W1 deliberately did not deploy Kong, so PostgREST and
//    Realtime live on two different Docker-assigned host ports and neither
//    serves under those path prefixes.
//
//    Rather than add Kong (that would be a W1 amendment) this bridges in the
//    client with the two extension points supabase-js already exposes:
//      * global.fetch      -> strip the `/rest/v1` prefix, keep the REST port
//      * realtime.transport-> re-point host:port at Realtime, rewrite the path
//                             to /socket/websocket, and set the tenant Host
//                             header W1 proved is required.
//
//    THIS SHIM IS A SPIKE ARTEFACT, NOT A RECOMMENDATION. See README half 2,
//    "What the shim means for the real migration".
// --------------------------------------------------------------------------
const REST_ORIGIN = `http://127.0.0.1:${REST_PORT}`;

class SpikeWebSocket extends WS {
    constructor(address, protocols, options) {
        const u = new URL(address);
        u.protocol = 'ws:';
        u.host = `127.0.0.1:${REALTIME_PORT}`;
        // supabase-js asks for /realtime/v1/websocket (the Kong route).
        // Self-hosted Realtime's Phoenix socket is mounted at /socket/websocket
        // — the exact path W1's rtwatch connected to.
        u.pathname = '/socket/websocket';
        super(u.toString(), protocols, {
            ...(options || {}),
            headers: { ...((options && options.headers) || {}), Host: REALTIME_VHOST }
        });
    }
}

export function makeSupabaseClient(token) {
    return createClient(REST_ORIGIN, token, {
        accessToken: async () => token,
        global: {
            fetch: (input, init) => {
                const url = new URL(typeof input === 'string' ? input : input.url);
                // PostgREST serves at the root, not under /rest/v1.
                url.pathname = url.pathname.replace(/^\/rest\/v1/, '');
                return fetch(url.toString(), init);
            }
        },
        realtime: { transport: SpikeWebSocket, params: { eventsPerSecond: 20 } }
    });
}

// --------------------------------------------------------------------------
// 4. The ONE throwaway collection.
//
//    Mirrors sql/spike-fixture.sql exactly: text primary key, plus the two
//    replication-contract columns. Note what is and is not in `properties`:
//
//    * `_deleted` is NOT declared. RxDB owns that field itself; the plugin maps
//      the Postgres column onto RxDB's internal deleted flag.
//    * `_modified` is NOT declared either. That is a real semantic choice, not
//      an omission: replicateSupabase only round-trips `_modified` into the
//      document when the schema declares it, and `addDocEqualityToQuery` only
//      includes `_modified` in its compare-and-swap when the schema declares
//      it. Leaving it out keeps `_modified` purely a server-stamped pull
//      cursor, which is what sql/spike-fixture.sql's trigger makes it.
//      See proof-lww.js for what that means when two writers collide.
// --------------------------------------------------------------------------
export const spikeNotesSchema = {
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
        id: { type: 'string', maxLength: 100 },
        owner_id: { type: 'string', maxLength: 100 },
        body: { type: 'string' }
    },
    required: ['id', 'owner_id', 'body']
};

// An OBSERVING conflict handler. It delegates every decision to RxDB's own
// defaultConflictHandler, so behaviour is byte-for-byte unchanged — it only
// prints what the default handler was asked and what it answered.
//
// This is deliberately a probe and NOT a fix. The card's question is "what does
// the no-custom-handler configuration actually do", and rewriting the handler
// to make it do something nicer would destroy the very answer we are after.
export function observingConflictHandler(log) {
    return {
        isEqual(a, b, ctx) {
            return defaultConflictHandler.isEqual(a, b, ctx);
        },
        resolve(input) {
            const chosen = defaultConflictHandler.resolve(input);
            log({
                assumedMasterState: input.assumedMasterState,
                newDocumentState: input.newDocumentState,
                realMasterState: input.realMasterState,
                chosen
            });
            return chosen;
        }
    };
}

let devModeAdded = false;
export async function makeLocalDb(name, { conflictHandler } = {}) {
    if (!devModeAdded) {
        // dev-mode gives real schema/typing errors instead of silent misbehaviour.
        // It is a development-only plugin and prints a banner on stdout.
        addRxPlugin(RxDBDevModePlugin);
        devModeAdded = true;
    }
    const db = await createRxDatabase({
        name,
        // Memory storage: free/Apache-2.0, and the honest choice for a Node
        // harness. See README half 2 for what a memory-storage Node proof does
        // and does NOT establish about the browser.
        //
        // The ajv wrapper is NOT optional here: with dev-mode enabled RxDB
        // refuses to create a database whose storage has no top-level schema
        // validator and throws DVM1. Measured, not guessed — see README half 2,
        // sharp edge 9.
        storage: wrappedValidateAjvStorage({ storage: getRxStorageMemory() }),
        ignoreDuplicate: true
    });
    await db.addCollections({
        notes: conflictHandler
            ? { schema: spikeNotesSchema, conflictHandler }
            : { schema: spikeNotesSchema }
    });
    return db;
}

// A per-run id prefix. Every proof writes ids that have never existed before,
// which is what makes re-running the harness safe with no reset step: no run
// can collide with a previous run's rows.
export const RUN = process.env.SPIKE_RUN_ID || `r${Date.now()}`;

export function banner() {
    console.log(`# stack: rest=${REST_PORT} realtime=${REALTIME_PORT} db=${DB_PORT}  run=${RUN}`);
}
