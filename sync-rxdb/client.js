// sync-rxdb/client.js — the PERMANENT client-construction helper.
//
// Card `sync-rxdb-replication-and-conflict-handler` (overnight-20260801, C1).
// Ledger decisions **51** (gateway-less + a permanent helper), **69** (the
// origin shape is SAME-ORIGIN, proxied by HQ's Go backend), **56** (umbrella
// slugs), **50** (the conflict handler this wires in).
//
// ===========================================================================
// THE DOOR, AND WHY THE CLIENT MUST BE BENT TO FIT IT.
// ===========================================================================
// HQ fronts the sync substrate itself: `backend/internal/sync/proxy.go` mounts
// `/sync/*` on HQ's own origin, with exactly two rooms —
//
//     /sync/rest/*        → PostgREST   (path-less, serves /<table> at its root)
//     /sync/realtime/*    → Realtime    (Phoenix socket at /socket/websocket)
//
// 🛑 THE CLIENT NEVER HOLDS THE SUBSTRATE CREDENTIAL. The door mints a bridge
// token PER REQUEST from HQ's own session and injects it; a caller-supplied
// `Authorization` header or `apikey` parameter is DELIBERATELY DISCARDED
// (proxy.go: `out.Header.Del("apikey")`, `out.Header.Set("Authorization", …)`,
// and `q.Set("apikey", tok)` on the Realtime path). So:
//
//   * nothing here fetches `/api/v1/sync/token`, and
//   * nothing here attaches a bearer, and
//   * the Realtime URL carries NO `apikey` parameter — the door sets it.
//
// The fetch shim below strips both headers on the way out anyway, so "the
// client never holds the credential" is literally true and testable rather
// than merely intended.
//
// Authentication is HQ's own session cookie, which rides along because the
// door is SAME-ORIGIN (decision 69). That is most of what decision 69 bought.
//
// ===========================================================================
// 🛑 THE COUPLING THIS FILE EXISTS TO ABSORB — AND WHY IT IS PINNED.
// ===========================================================================
// `@supabase/supabase-js` assumes ONE origin fronted by Kong. Given a base URL
// it derives, in its constructor:
//
//     new URL('rest/v1',     base)   →  <base>/rest/v1
//     new URL('realtime/v1', base)   →  <base>/realtime/v1     (+ '/websocket')
//
// (Measured against the committed bundle, not read from docs: the constructor
// is `JE` and the trailing-slash normaliser is `VE` in `vendor/rxdb.bundle.js`.)
//
// Decision 51 chose to stay GATEWAY-LESS rather than run Kong purely so a
// client library's constructor need not be told two URLs. Kong would cost a
// container, route config, and securing it AS THE FRONT DOOR, and it would
// reverse the one simplification the spike bought.
//
// The price is this file, and the price is stated honestly: the coupling is
// NOT to `global.fetch` and `realtime.transport` — those are public extension
// points and are stable. The coupling is to the ASSUMPTION about how the
// library derives `<baseUrl>/rest/v1`. An upgrade that changes that derivation
// breaks HQ silently, in production, with a 404 that looks like a network
// blip. Hence `assertVendorPin()` below and
// `tests/sync-rxdb-client.spec.js`, which fail LOUDLY on upgrade — three-way,
// across `vendor/package.json` (the input), the generated bundle's
// `VENDOR_BUILD` (the output) and `PINNED_VENDOR` (what this code asserts).
// Regenerating the bundle against different versions without updating the pin
// reds the suite instead of shipping a different engine to the truck.

import {
  createClient,
  createRxDatabase,
  getRxStorageDexie,
  replicateSupabase,
  defaultConflictHandler,
  VENDOR_BUILD,
} from '../vendor/rxdb.bundle.js';

import { REPLICATED_COLLECTIONS, LOCAL_COLLECTIONS } from '../sync-schema/collections.js';
import { createHQConflictHandler, describeConflict } from './conflict-handler.js';

export { describeConflict };

// ---------------------------------------------------------------------------
// The door's addresses. Written once, here.
// ---------------------------------------------------------------------------

/** What supabase-js is handed. It appends `/rest/v1` and `/realtime/v1`. */
export const SYNC_BASE_PATH = '/sync';
/** proxy.go `ProxyRESTPrefix`. */
export const REST_PREFIX = '/sync/rest';
/** proxy.go `ProxyRealtimePrefix`. */
export const REALTIME_PREFIX = '/sync/realtime';
/** Self-hosted Realtime's Phoenix socket, which is NOT at Kong's `/realtime/v1`. */
export const REALTIME_SOCKET_PATH = '/sync/realtime/socket/websocket';
/**
 * The Phoenix wire version. supabase-js 2.109.0 defaults to `2.0.0` (measured),
 * which selects a different serializer; self-hosted Realtime is spoken to at
 * `1.0.0` — the version the spike's `rtwatch` connected with. Set through the
 * library's own `realtime.vsn` option so the serializer and the URL cannot
 * disagree, rather than by rewriting the query string underneath it.
 */
export const REALTIME_VSN = '1.0.0';

/**
 * supabase-js REQUIRES a non-empty key or its constructor throws. This is not a
 * credential and is not secret: the door deletes it and substitutes a token
 * minted for the session user. It is a placeholder whose only job is to satisfy
 * an argument check, and the fetch shim strips it from the wire regardless.
 */
export const PROXY_PLACEHOLDER_KEY = 'hq-same-origin-proxy';

// ---------------------------------------------------------------------------
// The version pin (decision 51's rider).
// ---------------------------------------------------------------------------

/** Mirrors `vendor/package.json`. Asserted three ways — see the header. */
export const PINNED_VENDOR = { rxdb: '17.4.0', supabaseJs: '2.109.0' };

/**
 * Throw — loudly, at construction, before a single request goes out — if the
 * committed bundle was regenerated against different versions.
 *
 * Silence here is the failure mode being guarded: an upgrade that moves the
 * `<baseUrl>/rest/v1` derivation produces 404s that look like network blips on
 * a truck with bad LTE.
 */
export function assertVendorPin(build = VENDOR_BUILD, pinned = PINNED_VENDOR) {
  if (!build || build.rxdb !== pinned.rxdb || build.supabaseJs !== pinned.supabaseJs) {
    throw new Error(
      '[hq-sync] vendored engine does not match the pin. '
      + `expected rxdb=${pinned.rxdb} supabase-js=${pinned.supabaseJs}, `
      + `bundle reports rxdb=${build && build.rxdb} supabase-js=${build && build.supabaseJs}. `
      + 'The same-origin client helper is coupled to how supabase-js derives '
      + '<baseUrl>/rest/v1 — re-verify sync-rxdb/client.js before moving the pin.',
    );
  }
  return build;
}

// ---------------------------------------------------------------------------
// URL rewriting. Pure functions, so the shims are testable without a network.
// ---------------------------------------------------------------------------

/**
 * `/sync/rest/v1/<table>` → `/sync/rest/<table>`.
 *
 * Kong would serve PostgREST under `/rest/v1`; HQ's door strips `/sync/rest`
 * and PostgREST serves at its own root, so the `/v1` segment must go. Only the
 * segment IMMEDIATELY after the REST prefix is removed — a table or RPC
 * legitimately named `v1` deeper in the path is left alone, and a path that is
 * not under the REST prefix is returned untouched.
 */
export function rewriteRestPath(pathname) {
  if (pathname === REST_PREFIX + '/v1') return REST_PREFIX;
  if (pathname.startsWith(REST_PREFIX + '/v1/')) {
    return REST_PREFIX + pathname.slice((REST_PREFIX + '/v1').length);
  }
  return pathname;
}

/**
 * `<origin>/sync/realtime/v1/websocket?apikey=…&vsn=…`
 *   → `<origin>/sync/realtime/socket/websocket?vsn=1.0.0`
 *
 * Two edits, both deliberate:
 *   1. the Phoenix socket lives at `/socket/websocket`, not at Kong's
 *      `/realtime/v1/websocket`;
 *   2. `apikey` is REMOVED. proxy.go `q.Set("apikey", tok)` would replace it
 *      anyway, but sending a placeholder credential the door then overwrites
 *      is a lie in every access log between here and there.
 */
export function rewriteRealtimeUrl(input) {
  const u = new URL(String(input));
  if (u.pathname.startsWith(REALTIME_PREFIX)) {
    u.pathname = REALTIME_SOCKET_PATH;
  }
  u.searchParams.delete('apikey');
  return u.toString();
}

/**
 * A WebSocket subclass that re-points the handshake at the door.
 *
 * `realtime.transport` is a public extension point of supabase-js, so this half
 * of the shim is stable across upgrades in a way the URL derivation is not.
 */
export function makeRealtimeTransport(WebSocketImpl) {
  return class HQRealtimeSocket extends WebSocketImpl {
    constructor(url, protocols, options) {
      super(rewriteRealtimeUrl(url), protocols, options);
    }
  };
}

/**
 * The fetch shim: rewrite the REST path and strip every client-supplied
 * credential before the request leaves the page.
 */
export function makeSyncFetch(fetchImpl) {
  return function hqSyncFetch(input, init) {
    const raw = typeof input === 'string' ? input : (input && input.url) || String(input);
    const u = new URL(raw, typeof location !== 'undefined' ? location.origin : undefined);
    u.pathname = rewriteRestPath(u.pathname);
    u.searchParams.delete('apikey');

    const headers = new Headers((init && init.headers) || (input && input.headers) || {});
    // The door mints per request and injects; anything we send is discarded.
    // Deleting it here means the browser never puts a placeholder credential on
    // the wire in the first place.
    headers.delete('authorization');
    headers.delete('apikey');

    return fetchImpl(u.toString(), Object.assign({}, init, {
      headers,
      // Same-origin by decision 69 — this is the HQ session cookie the door
      // authenticates on. `same-origin` is fetch's default, stated explicitly
      // because everything downstream depends on it.
      credentials: 'same-origin',
    }));
  };
}

// ---------------------------------------------------------------------------
// Obligation 4 — umbrella slugs (decision 56).
// ---------------------------------------------------------------------------

/**
 * The go-forward convention is that grants are PER-TAB, not bundled per app.
 * Umbrella slugs are the one live exception, and they are real in shipped code:
 * `backend/cmd/server/main.go` mounts
 *   `RequirePermission("inventory-trends", "inventory")` and
 *   `RequirePermission("inventory-cost",   "inventory")`,
 * so a user holding `inventory` genuinely reaches both tabs. A surface built
 * naively from the narrow claim hides two things the user is entitled to.
 *
 * The token's `hq_grants` claim is the NARROW list and is advisory only
 * (`backend/internal/sync/jwtbridge.go` says so, and points the expansion at
 * the client) — the live grant projection is the gate. This table is therefore
 * a rendering aid, never an authorization decision.
 */
export const UMBRELLA_SLUGS = {
  inventory: ['inventory-trends', 'inventory-cost'],
};

/**
 * Expand a claimed grant list into the set of surfaces actually reachable.
 * Idempotent, order-independent, de-duplicated, sorted.
 */
export function expandGrantSlugs(slugs) {
  const out = new Set();
  for (const s of slugs || []) {
    if (typeof s !== 'string' || s === '') continue;
    out.add(s);
    for (const child of UMBRELLA_SLUGS[s] || []) out.add(child);
  }
  return [...out].sort();
}

// ---------------------------------------------------------------------------
// The client.
// ---------------------------------------------------------------------------

/**
 * Construct the supabase-js client bound to HQ's same-origin door.
 *
 * Pure construction — NO network request is made here, so a page may call it
 * on load whether or not the substrate is configured. (With
 * `HQ_SYNC_REST_URL` unset, which is its normal and currently REQUIRED state,
 * the door answers 503 to everything.)
 *
 * @param {object} [opts]
 * @param {string} [opts.origin]        defaults to `location.origin`.
 * @param {function} [opts.fetchImpl]   defaults to the global `fetch`.
 * @param {function} [opts.WebSocketImpl] defaults to the global `WebSocket`.
 */
export function createHQSupabaseClient(opts = {}) {
  assertVendorPin();

  const origin = opts.origin
    || (typeof location !== 'undefined' ? location.origin : undefined);
  if (!origin) throw new Error('[hq-sync] no origin: pass {origin} outside a browser');

  const fetchImpl = opts.fetchImpl
    || (typeof globalThis !== 'undefined' && globalThis.fetch
      ? globalThis.fetch.bind(globalThis)
      : undefined);
  if (!fetchImpl) throw new Error('[hq-sync] no fetch implementation available');

  const WebSocketImpl = opts.WebSocketImpl
    || (typeof globalThis !== 'undefined' ? globalThis.WebSocket : undefined);

  const realtime = { vsn: REALTIME_VSN, params: { eventsPerSecond: 20 } };
  if (WebSocketImpl) realtime.transport = makeRealtimeTransport(WebSocketImpl);

  return createClient(origin + SYNC_BASE_PATH, PROXY_PLACEHOLDER_KEY, {
    // Never GoTrue. There is no auth server behind the door and there is not
    // meant to be — HQ's own session is the identity, resolved at the door.
    // Supplying `accessToken` is what keeps supabase-js from reaching for one.
    accessToken: async () => PROXY_PLACEHOLDER_KEY,
    global: { fetch: makeSyncFetch(fetchImpl) },
    realtime,
  });
}

/**
 * Everything a page needs, constructed in one call.
 *
 * @returns {{client, surfaces: string[], conflictHandler, vendor}}
 *
 * `surfaces` is the umbrella-expanded reachable set (obligation 4). It is a
 * RENDERING aid: the server-side grant projection remains the gate.
 */
export function createHQSyncClient(opts = {}) {
  const client = createHQSupabaseClient(opts);
  const conflictHandler = createHQConflictHandler({
    isEqual: defaultConflictHandler.isEqual,
    onClash: opts.onClash,
  });
  return {
    client,
    surfaces: expandGrantSlugs(opts.grants),
    conflictHandler,
    vendor: VENDOR_BUILD,
  };
}

// ---------------------------------------------------------------------------
// The RxDB database. BROWSER ONLY — Dexie needs IndexedDB.
// ---------------------------------------------------------------------------

/**
 * Build the local RxDB database from `sync-schema/collections.js`.
 *
 * The HQ conflict handler is attached to every REPLICATED collection. It is
 * deliberately NOT attached to `conflict_records`: that collection is local by
 * decision 89, never replicates, and therefore never conflicts — giving it a
 * handler would imply otherwise.
 *
 * ⚠ This creates IndexedDB state. It does NOT start replication — see
 * `startHQReplication`.
 */
export async function createHQSyncDatabase(opts = {}) {
  assertVendorPin();

  const db = await createRxDatabase({
    name: opts.name || 'hq_sync',
    storage: opts.storage || getRxStorageDexie(),
    multiInstance: opts.multiInstance !== false,
    ignoreDuplicate: !!opts.ignoreDuplicate,
  });

  const conflictHandler = opts.conflictHandler || createHQConflictHandler({
    isEqual: defaultConflictHandler.isEqual,
    onClash: opts.onClash,
  });

  const defs = {};
  for (const [key, def] of Object.entries(REPLICATED_COLLECTIONS)) {
    defs[key] = { schema: def.schema, conflictHandler };
  }
  for (const [key, def] of Object.entries(LOCAL_COLLECTIONS)) {
    defs[key] = { schema: def.schema };
  }
  await db.addCollections(defs);
  return db;
}

/**
 * Start `replicateSupabase` for each replicated collection.
 *
 * 🛑 NOT CALLED BY ANY PAGE ON THIS CARD. `workflows.html` gets import +
 * construction only; the write-path swap is `sync-hard-cutover`, and
 * `HQ_SYNC_REST_URL` must not be set in any deploy until
 * `sync-rxdb-row-visibility-rls` lands. Calling this today produces 503s from a
 * door that is deliberately shut. It lives here so the shape is reviewable now
 * and so C2 can drive it in a test.
 *
 * @returns {Record<string, object>} replication states, keyed by collection.
 */
export function startHQReplication(db, client, opts = {}) {
  const states = {};
  for (const [key, def] of Object.entries(REPLICATED_COLLECTIONS)) {
    const state = replicateSupabase({
      // Stable across reconnects ON PURPOSE: a different identifier hands the
      // new connection a blank checkpoint, which is a full re-pull rather than
      // a resume (spike `proof-lww.js` depends on the same property).
      replicationIdentifier: `hq-sync-${def.table}`,
      collection: db[key],
      client,
      tableName: def.table,
      // Leader election across tabs is on by default in a browser; leave the
      // library's own default alone unless a caller has a reason.
      waitForLeadership: opts.waitForLeadership !== false,
      live: opts.live !== false,
      pull: { batchSize: opts.pullBatchSize || 50 },
      push: { batchSize: opts.pushBatchSize || 50 },
    });
    if (opts.onConflict) {
      // `conflict$` fires PER DOCUMENT and carries the document id (verified in
      // the shipped bundle: the emission sits inside
      // `Object.entries(conflictsById).map(...)`). `describeConflict` turns one
      // emission into the recoverable-loss rows C2 renders.
      state.conflict$.subscribe((e) => opts.onConflict(describeConflict(e), key));
    }
    states[key] = state;
  }
  return states;
}
