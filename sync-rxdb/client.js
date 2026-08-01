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
import {
  createHQConflictHandler,
  describeConflict,
  conflictOptsOf,
} from './conflict-handler.js';

export { describeConflict, conflictOptsOf };

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
// REPLICATION SCOPE — preference `architecture/C-2`, ledger T-29 decision 105.
//
//   "Any client-side fetch or replication over a collection that can grow
//    without bound is batched and scoped — never pulled whole. Scope it to what
//    the current view actually needs (for workflows, the open checklist).
//    Widening the scope requires a recorded decision."
//
// WHAT WAS WRONG. `startHQReplication` looped all four replicated collections
// with `pull:{batchSize:50}` and no selector, filter or query modifier. Two
// consequences, and the second is the one that would have reached the truck:
// the RLS predicate was re-evaluated per row on every page (the whole of Fork
// 1's ~23 s figure), and `responses` grows forever while every phone was to
// hold all of it.
//
// WHERE THE SCOPE ATTACHES. `rxdb/plugins/replication-supabase` builds its pull
// as (measured against the committed bundle, not read from docs):
//
//     var S = client.from(tableName).select("*");
//     if (pull?.queryBuilder) {
//       var R = pull.queryBuilder({ query: S, lastPulledCheckpoint: g, batchSize: _ });
//       R && (S = R);
//     }
//     if (g) { S = S.or('"_modified".gt.…'); }     // the checkpoint
//     S = S.order(…).order(…).limit(_);
//
// `pull.queryBuilder` is therefore the single supported seam, it runs BEFORE
// the checkpoint clause, and PostgREST ANDs the two — so the scope narrows the
// resume rather than fighting it.
//
// 🛑 WHAT THIS DOES *NOT* SCOPE, stated rather than implied. The plugin's live
// path subscribes `postgres_changes` for `{event:'*', schema:'public', table}`
// with NO filter, and feeds every event straight into the local store. A row
// belonging to a checklist this device never opened, CHANGED while the page is
// open, still arrives. The plugin exposes no seam for it (`pull.modifier` is
// applied to stream documents but cannot drop one — the downstream does
// `documents.map(modifier)` with no null filter), and Realtime's own
// `postgres_changes` filter accepts a single `col=op.value` clause, which
// cannot express `responses`' two-branch scope. Filed as `SYNC-REALTIME-SCOPE`
// in `.night-crew/knowledge/BACKLOG.md`; it is not this card's, and it does not
// reduce what the pull scope buys (the pull is the unbounded leg — the stream
// is bounded by what other people change while you are looking).
//
// THE THREE SCOPING EDGES, AND WHO DECIDED THEM. All three were decided here,
// by C-2, and none of them needed a schema or a policy change — every key below
// was already declared by `sync-schema/collections.js`, which this card leaves
// byte-unchanged. The card's PARK trigger did not fire.
//
//   1. DRAFT RESPONSES. A draft has `submission_id IS NULL` until submit
//      (migration 0012's partial unique index), and drafts are exactly what a
//      crew member fills offline — so `submission_id.eq.<open>` alone would
//      drop the one thing this collection exists to carry. C-2 says "what the
//      current view actually needs", not "what already has a foreign key": the
//      scope is the open checklist's submitted rows OR a draft on one of the
//      open checklist's OWN field ids. It is not "all my drafts everywhere",
//      which would be a different, unbounded set.
//
//   2. TEMPLATES. Bounded in principle (one row per template) but not pulled
//      whole either — C-2's rule is about the current view, and the view is one
//      checklist. Scoped to the open checklist's template when the caller names
//      one; otherwise to the non-archived set, which is what a launcher list
//      shows. `archived_at` was already declared.
//
//   3. NO SCOPE AT ALL. Refused, loudly, at the call. A default that fell back
//      to the whole collection would widen C-2 silently, and C-2 requires a
//      RECORDED decision to widen. A throw is the only version of this that
//      cannot be reached by accident.
// ---------------------------------------------------------------------------

/**
 * Normalise + validate a replication scope.
 *
 * @param {object} scope
 * @param {string} scope.checklistId  the open checklist (`checklist_submissions.id`). REQUIRED.
 * @param {string} [scope.templateId] its template, when known.
 * @param {string[]} [scope.fieldIds] the open checklist's field ids — what makes
 *   an offline draft (`submission_id IS NULL`) attributable to this checklist.
 */
export function normalizeScope(scope) {
  if (!scope || typeof scope !== 'object') {
    throw new Error(
      '[hq-sync] startHQReplication requires a scope: replication is scoped to the '
      + 'open checklist and is never pulled whole (preference architecture/C-2). '
      + 'Pass {scope:{checklistId, templateId, fieldIds}}.',
    );
  }
  if (typeof scope.checklistId !== 'string' || scope.checklistId === '') {
    throw new Error(
      '[hq-sync] scope.checklistId is required — it is the open checklist the '
      + 'replication is scoped to (preference architecture/C-2).',
    );
  }
  const fieldIds = (scope.fieldIds || []).filter((f) => typeof f === 'string' && f !== '');
  return {
    checklistId: scope.checklistId,
    templateId: typeof scope.templateId === 'string' && scope.templateId !== ''
      ? scope.templateId
      : null,
    fieldIds,
  };
}

/**
 * The scope for ONE collection, as a declarative filter tree.
 *
 * Kept declarative rather than emitted straight onto a PostgREST builder so the
 * rule can be read, tested and diffed as data. `applyScope` is the only thing
 * that turns it into calls.
 *
 * @returns {object|null} a filter node, or null for "this collection carries no
 *   scope key" — which no collection currently returns, and which `applyScope`
 *   treats as a programming error rather than as permission to pull whole.
 */
export function scopeFilterFor(collectionKey, scope) {
  const s = normalizeScope(scope);
  switch (collectionKey) {
    case 'templates':
      return s.templateId
        ? { op: 'eq', column: 'id', value: s.templateId }
        : { op: 'is', column: 'archived_at', value: null };
    case 'checklists':
      return { op: 'eq', column: 'id', value: s.checklistId };
    case 'responses':
      // Edge 1. Submitted rows on the open checklist, OR a draft on one of its
      // own fields. With no field ids in hand there is nothing that makes a
      // draft attributable, so the draft branch is omitted rather than widened.
      return s.fieldIds.length
        ? {
          op: 'or',
          clauses: [
            { op: 'eq', column: 'submission_id', value: s.checklistId },
            {
              op: 'and',
              clauses: [
                { op: 'is', column: 'submission_id', value: null },
                { op: 'in', column: 'field_id', values: s.fieldIds },
              ],
            },
          ],
        }
        : { op: 'eq', column: 'submission_id', value: s.checklistId };
    case 'approvals':
      return { op: 'eq', column: 'submission_id', value: s.checklistId };
    default:
      return null;
  }
}

/**
 * Serialise a filter node into PostgREST's embedded `or=`/`and=` grammar.
 * Columns are quoted the way the vendored plugin quotes its own checkpoint
 * clause, so the two compose without a quoting disagreement.
 */
export function serializeFilter(node) {
  switch (node.op) {
    case 'eq': return `"${node.column}".eq.${node.value}`;
    case 'is': return `"${node.column}".is.null`;
    case 'in': return `"${node.column}".in.(${node.values.join(',')})`;
    case 'and': return `and(${node.clauses.map(serializeFilter).join(',')})`;
    case 'or': return `or(${node.clauses.map(serializeFilter).join(',')})`;
    default: throw new Error('[hq-sync] unknown scope filter op: ' + node.op);
  }
}

/** Apply a filter node to a PostgREST query builder. */
export function applyScope(query, node) {
  if (!node) {
    throw new Error(
      '[hq-sync] a replicated collection with no scope key would be pulled whole '
      + '(preference architecture/C-2). Give it one, or do not replicate it.',
    );
  }
  switch (node.op) {
    case 'eq': return query.eq(node.column, node.value);
    case 'is': return query.is(node.column, null);
    case 'in': return query.in(node.column, node.values);
    // A top-level `or`/`and` goes on the wire as one `or=`/`and=` parameter.
    case 'or': return query.or(node.clauses.map(serializeFilter).join(','));
    case 'and': return query.or(serializeFilter(node));
    default: throw new Error('[hq-sync] unknown scope filter op: ' + node.op);
  }
}

/**
 * The `pull.queryBuilder` the replication plugin calls. Returns the narrowed
 * query; the plugin then ANDs its own checkpoint clause onto it.
 */
export function makePullQueryBuilder(collectionKey, scope) {
  const node = scopeFilterFor(collectionKey, scope);
  return ({ query }) => applyScope(query, node);
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
 * 🛑 `opts.scope` IS REQUIRED — see the REPLICATION SCOPE block above. There is
 * no unscoped call. `{checklistId}` at minimum; add `templateId` and `fieldIds`
 * so the template and the offline drafts come with it.
 *
 * @returns {Record<string, object>} replication states, keyed by collection.
 */
export function startHQReplication(db, client, opts = {}) {
  const states = {};
  // Validate ONCE, before a single replication is started: a scope that is
  // going to be refused should refuse before half the collections are live.
  const scope = normalizeScope(opts.scope);
  // Testability seam ONLY. Defaults to the vendored plugin; a test injects a
  // recorder so the per-collection replication options can be read without a
  // browser, an IndexedDB or a substrate. Production never passes it.
  const replicate = opts.replicate || replicateSupabase;
  for (const [key, def] of Object.entries(REPLICATED_COLLECTIONS)) {
    const state = replicate({
      // Stable across reconnects ON PURPOSE: a different identifier hands the
      // new connection a blank checkpoint, which is a full re-pull rather than
      // a resume (spike `proof-lww.js` depends on the same property).
      //
      // 🛑 The identifier does NOT carry the scope, and that is deliberate. RxDB
      // keys its checkpoint by this string; folding the checklist id in would
      // mint a fresh identifier — and therefore a blank checkpoint, a full
      // re-pull — every time the crew member opened a different checklist,
      // which is the cost this card exists to remove.
      replicationIdentifier: `hq-sync-${def.table}`,
      // Read by the injected recorder in tests; the plugin ignores unknown keys.
      collectionKey: key,
      collection: db[key],
      client,
      tableName: def.table,
      // Leader election across tabs is on by default in a browser; leave the
      // library's own default alone unless a caller has a reason.
      waitForLeadership: opts.waitForLeadership !== false,
      live: opts.live !== false,
      // BATCHED **AND** SCOPED — both halves of C-2, in one place.
      pull: {
        batchSize: opts.pullBatchSize || 50,
        queryBuilder: makePullQueryBuilder(key, scope),
      },
      push: { batchSize: opts.pushBatchSize || 50 },
    });
    if (opts.onConflict) {
      // `conflict$` fires PER DOCUMENT and carries the document id (verified in
      // the shipped bundle: the emission sits inside
      // `Object.entries(conflictsById).map(...)`). `describeConflict` turns one
      // emission into the recoverable-loss rows C2 renders.
      //
      // 🛑 G6 CORRECTION (C2). `describeConflict` re-runs `resolveConflict` to
      // derive that row set, and it used to run with its own defaults — so a
      // caller who customised `reservedFields`/`provenanceFields` at
      // `createHQConflictHandler` got a clash list here that DISAGREED with what
      // the handler had actually done. The handler's own options are threaded
      // through, read off the collection the handler was attached to, so the
      // sheet reports the decision that was made rather than the one the
      // defaults would have made.
      const conflictOpts = opts.conflictOpts
        || conflictOptsOf(db[key] && db[key].conflictHandler);
      state.conflict$.subscribe((e) => opts.onConflict(describeConflict(e, conflictOpts), key));
    }
    states[key] = state;
  }
  return states;
}
