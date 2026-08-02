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

  const client = createClient(origin + SYNC_BASE_PATH, PROXY_PLACEHOLDER_KEY, {
    // Never GoTrue. There is no auth server behind the door and there is not
    // meant to be — HQ's own session is the identity, resolved at the door.
    // Supplying `accessToken` is what keeps supabase-js from reaching for one.
    accessToken: async () => PROXY_PLACEHOLDER_KEY,
    global: { fetch: makeSyncFetch(fetchImpl) },
    realtime,
  });

  // B-42 option (i). Installed HERE and not left to the caller: the plugin
  // hard-codes its `postgres_changes` binding, so the client's own `channel()`
  // is the only seam, and a shim a page has to remember to install is a shim
  // that is missing on the page that forgot. No-op until a replication
  // registers a filter — construction still makes no network request.
  installRealtimeFilterShim(client);
  return client;
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
// cannot express `responses`' two-branch scope. Filed as `B-42
// SYNC-REALTIME-SCOPE` in `.night-crew/knowledge/BACKLOG.md`; it is not this
// card's, and it does not
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
//      checklist, which has exactly one template. Scoped `id.eq.<templateId>`,
//      and `templateId` is therefore REQUIRED.
//
//      🛑 G6 CORRECTION (F-5). This used to fall back to `archived_at.is.null`
//      — every non-archived template — when the caller omitted `templateId`.
//      That is a WIDENING TRIGGERED BY AN OMISSION: forgetting an optional
//      argument silently bought the whole (non-archived) collection, which is
//      the exact shape edge 3 refuses. C-2 requires a RECORDED decision to
//      widen, and a forgotten argument is not a decision. The fallback is gone;
//      an absent `templateId` now throws with the other scope keys.
//
//   3. NO SCOPE AT ALL. Refused, loudly, at the call. A default that fell back
//      to the whole collection would widen C-2 silently, and C-2 requires a
//      RECORDED decision to widen. A throw is the only version of this that
//      cannot be reached by accident.
//
// CHECKPOINTS ARE PER-SCOPE — 🛑 G6 CORRECTION (F-1), and the reason this block
// no longer says the opposite.
//
// This card originally kept the scope OUT of `replicationIdentifier` on the
// theory that folding it in would mint a blank checkpoint (a full re-pull) on
// every checklist switch. Measured against the bundle, that reasoning was
// backwards and the resulting behaviour was DATA LOSS:
//
//     this.metaInfoPromise = (async () => {
//       var g = "rx-replication-meta-"
//             + await n.database.hashFunction(
//                 [this.collection.name, this.replicationIdentifier].join("-"));
//       …
//
// The persisted checkpoint is keyed by `[collection.name, replicationIdentifier]`
// and BY NOTHING ELSE — the scope is not part of the key. The pull returns
// `lastOfArray(data)` → `{id, modified}` of the last row IN THE SCOPED RESULT
// SET, and the plugin then ANDs
// `or("_modified".gt.C, and("_modified".eq.C,"id".gt.I))` onto the next pull.
// So with one identifier spanning all scopes:
//
//     open today's checklist   (_modified 2026-08-02T08:10Z) → rows, checkpoint advances
//     open YESTERDAY's         (_modified 2026-08-01T09:00Z) → EVERY row is <= C
//                                                            → ZERO rows, permanently
//
// The cost the old comment was protecting against no longer exists. Before this
// card a blank checkpoint meant re-pulling ALL HISTORY (20 pages × 50 rows).
// After it, a blank checkpoint means re-pulling ONE CHECKLIST — roughly one
// batch. Paying one batch to avoid permanent data loss is not a trade.
//
// So: `replicationIdentifier` is `hq-sync-<table>-<scopeFingerprint>`, where the
// fingerprint is a hash of THAT COLLECTION'S OWN serialized filter. Identical
// scope ⇒ identical identifier ⇒ the checkpoint still RESUMES; different scope
// ⇒ different identifier ⇒ a fresh checkpoint that cannot filter the new scope's
// rows away. Correct by construction rather than by a caller remembering to
// reset something.
//
// 🛑 CALLERS MUST CANCEL BEFORE RE-SCOPING. The plugin's Realtime subscription
// is `client.channel(replicationIdentifier)`. Per-scope identifiers mean a
// re-scope now lands on a DIFFERENT topic (an improvement — two scopes no
// longer share one channel), but a caller that starts a replication for the
// SAME scope twice without cancelling still gets two subscriptions on one
// topic, and the old scope's replication keeps running and keeps writing into
// the same local collections. `sync-hard-cutover` owns the page-level
// start/cancel lifecycle; see `B-42 SYNC-REALTIME-SCOPE`.
// ---------------------------------------------------------------------------

/**
 * The character class a scope id may use.
 *
 * 🛑 G6 CORRECTION (F-4). `serializeFilter` interpolates scope values into
 * PostgREST's `or=` logic-tree grammar, where `,` `(` `)` and `"` are
 * STRUCTURE. A value carrying them rewrites the predicate rather than filling
 * it in — `checklistId: 'x,"id".not.is.null'` emits a tree that is true for
 * every row, i.e. the whole table, reached THROUGH the thing this block calls a
 * gate. Ids in HQ are internally-generated UUIDs so there is no untrusted path
 * today; that is a reason the fix is cheap, not a reason to skip it.
 *
 * Deliberately a whitelist rather than a UUID shape: it admits UUIDs, admits
 * the synthetic ids the tests and the conflict spike use, and admits nothing
 * that PostgREST's grammar can read as structure. `serializeFilter` ALSO quotes
 * the value; the two are independent and both are kept.
 */
const SCOPE_ID_RE = /^[A-Za-z0-9_-]+$/;

function assertScopeId(label, value) {
  if (typeof value !== 'string' || value === '') {
    throw new Error(
      `[hq-sync] scope.${label} is required — it is part of the scope the `
      + 'replication is narrowed to (preference architecture/C-2).',
    );
  }
  if (!SCOPE_ID_RE.test(value)) {
    throw new Error(
      `[hq-sync] scope.${label} must match ${SCOPE_ID_RE} — a value carrying `
      + "PostgREST logic-tree punctuation (, ( ) \") could rewrite the scope "
      + `predicate instead of filling it in. Got: ${JSON.stringify(value)}`,
    );
  }
  return value;
}

// ---------------------------------------------------------------------------
// 🛑 THE SECOND SCOPE — card `sync-cutover-list-scope` (S1a, overnight-20260803).
//
// WHAT WAS WRONG WITH ONE SCOPE, and it was not wrong for the reason it looks.
// Everything above is correct for the view preference `architecture/C-2` names —
// the checklist FILL view — and it is NOT the view a crew member lands on.
// `workflows.html` opens on **My Checklists** (a list over every submission this
// user can see) and its second tab is **Approvals** (a list over every rejection
// awaiting them). Neither can name a single `checklistId`, and `normalizeScope`
// above throws without one. Filed as `B-43`.
//
// THE OPERATOR DECIDED IT ON 2026-08-02 EVENING: **LISTS STAY LIVE — THE SCOPE
// IS WIDENED.** `reference/slate-20260803.md` is the recorded decision, and it
// AMENDS ledger T-29 decision 105 rather than repealing it:
//
//     per-open-checklist for the fill collections;
//     per-user-with-a-date-floor for the two list collections;
//     NEVER ALL HISTORY, NEVER ALL USERS.
//
// ── 🛑 THE DATE FLOOR IS NOT A CONVENIENCE. IT IS THE PRICE. ────────────────
//
// `B-42` recorded, and the G6 round of run 20260802 corrected the wording to
// make it unmissable: RxDB's downstream ONLY ADDS. Nothing evicts, there is no
// retention sweep for the four replicated collections, and A1's per-checklist
// scope moved the per-phone bound from *all history* to *opened checklists* —
// an improvement, not a bound. A per-user list scope widens it again. The floor
// is what puts the bound back, so it is REQUIRED and `normalizeScope` throws
// without it. A floor that can be forgotten is the F-5 shape: a widening
// triggered by an omission, which C-2 says needs a recorded decision and an
// omission is not one.
//
// ── 🛑 WHY THE PER-USER HALF IS NOT `assigned_to.eq.<userId>` ──────────────
//
// The slate wrote the list scope as `checklists: assigned_to.eq.<userId>`.
// MEASURED AGAINST `sync-schema/sql/0001_sync_tables.sql`: THERE IS NO SUCH
// COLUMN, on that table or on any of the four. `checklist_submissions` carries
// `submitted_by` and `reviewed_by`; `submission_rejections` carries
// `rejected_by`. Assignment lives in `template_assignments`, which is not a
// replicated collection and is not queryable by a PostgREST client at all — the
// foreign tables are revoked from `authenticated` (0002 §4) precisely so a GET
// cannot read HQ's role map, and `hq_template_assignees` is reachable only from
// inside the SECURITY DEFINER policy functions.
//
// So the literal clause needs a QUERYABLE KEY ON THE ROW, which is this card's
// PARK trigger. Rather than park the night on a spelling, the per-user half is
// expressed through the two mechanisms that already exist and are stronger:
//
//   1. `scope.templateIds` — the templates assigned to THIS user, which the
//      list page already holds (it renders them). `checklist_templates.id` and
//      `checklist_submissions.template_id` are both queryable columns, so
//      "this user's assigned set" is expressible today with no schema change.
//      REQUIRED and NON-EMPTY, for the same F-5 reason the floor is.
//
//   2. RLS — `checklist_submissions_select` is `hq_can_see_template(template_id)`
//      and `submission_rejections_select` is `hq_can_see_field(field_id)`. That
//      is the per-user narrowing, it is read LIVE per row through the FDW rather
//      than from a token (0003 §4), and it cannot be forged by a client. The
//      client scope is a BOUND; the server is the GATE. Proved discriminating by
//      `TestRowVisibilityRLS/LIST-1..3` — alice's list scope returns alice's
//      rows and refuses bob's, against a `service_role` BYPASSRLS control.
//
// 🛑 `scope.userId` THEREFORE APPEARS IN NO FILTER CLAUSE, and that is stated
// rather than hidden. It is the scope's IDENTITY: it goes into the fingerprint,
// hence into `replicationIdentifier`, hence into RxDB's checkpoint key. A shared
// truck phone that switches crew member MUST mint a new identifier or the second
// user resumes the first user's cursor and their own rows are filtered away —
// exactly the F-1 data loss, on a different axis.
//
// 🛑 ONE BEHAVIOUR CHANGE THIS BUYS, RECORDED NOT HIDDEN. HQ's REST list
// (`myChecklists`, backend/internal/workflow/repository.go) returns EVERY
// submission since `current_date` with no per-user filter at all — "checklists
// are team objects, all members see all submissions". The RxDB-backed list is
// necessarily narrower: RLS admits only the user's own assigned templates. That
// is a product-visible difference at cutover and is filed as **B-61** with a
// destination, not decided here.
// ---------------------------------------------------------------------------

/**
 * The DATE FLOOR's shape. A calendar date, or a full ISO-8601 instant.
 *
 * Deliberately its own whitelist rather than `SCOPE_ID_RE`, which admits no `:`
 * or `.` and would reject every timestamp — and deliberately not "anything
 * `Date.parse` likes", which admits `2026-08-02, whatever` and hands F-4 a
 * second door. Nothing this regex admits can carry PostgREST logic-tree
 * punctuation (`,` `(` `)` `"`). `serializeFilter` quotes it as well; the two
 * are independent and both are kept.
 */
const SCOPE_DATE_RE = /^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}:\d{2}(\.\d{1,6})?(Z|[+-]\d{2}:\d{2})?)?$/;

function assertDateFloor(value) {
  if (typeof value !== 'string' || value === '' || !SCOPE_DATE_RE.test(value)
    || Number.isNaN(Date.parse(value))) {
    throw new Error(
      '[hq-sync] scope.since is required on a LIST scope and must be a date floor '
      + '("YYYY-MM-DD" or a full ISO-8601 instant). The date floor is the bound the '
      + 'widening was granted on condition of: nothing evicts a replicated row '
      + '(B-42), so a list scope without one pulls all history onto a phone that '
      + `never deletes anything. Got: ${JSON.stringify(value)}`,
    );
  }
  return value;
}

/**
 * Normalise + validate a replication scope. TWO SHAPES, one function, because
 * every caller path must go through the same refusals.
 *
 * FILL (the open checklist — ledger T-29 decision 105, unchanged):
 * @param {string} scope.checklistId  the open checklist (`checklist_submissions.id`). REQUIRED.
 * @param {string} scope.templateId   its template. REQUIRED — see edge 2 / F-5.
 * @param {string[]} [scope.fieldIds] the open checklist's field ids — what makes
 *   an offline draft (`submission_id IS NULL`) attributable to this checklist.
 *
 * LIST (My Checklists / Approvals — the 2026-08-02 amendment):
 * @param {'list'} scope.mode         REQUIRED to select this shape.
 * @param {string} scope.userId       whose lists these are. The scope's IDENTITY,
 *   not a filter clause — see the block above.
 * @param {string} scope.since        the DATE FLOOR. REQUIRED.
 * @param {string[]} scope.templateIds the templates assigned to this user.
 *   REQUIRED and NON-EMPTY.
 */
export function normalizeScope(scope) {
  if (!scope || typeof scope !== 'object') {
    throw new Error(
      '[hq-sync] startHQReplication requires a scope: replication is scoped and is '
      + 'never pulled whole (preference architecture/C-2). Pass either the FILL '
      + 'scope {scope:{checklistId, templateId, fieldIds}} or the LIST scope '
      + '{scope:{mode:"list", userId, since, templateIds}}.',
    );
  }

  if (scope.mode === 'list') {
    // The identity. Validated with the same whitelist the fill ids use, because
    // it is interpolated into `replicationIdentifier` (and a caller could
    // reasonably expect to put it in a clause later).
    if (typeof scope.userId !== 'string' || scope.userId === '') {
      throw new Error(
        '[hq-sync] scope.userId is required on a LIST scope — it is the scope\'s '
        + 'identity and part of the replication identifier, so two crew members on '
        + 'one truck phone do not inherit each other\'s checkpoint.',
      );
    }
    assertScopeId('userId', scope.userId);
    assertDateFloor(scope.since);
    const templateIds = (scope.templateIds || [])
      .filter((t) => typeof t === 'string' && t !== '');
    if (templateIds.length === 0) {
      throw new Error(
        '[hq-sync] scope.templateIds is required and must be NON-EMPTY on a LIST '
        + 'scope — it is the per-user half of the bound (the templates assigned to '
        + 'this user), and no replicated table carries a queryable `assigned_to` '
        + 'key to use instead. An empty or omitted set would leave `templates` '
        + 'unbounded, which is the F-5 shape: a widening bought by forgetting an '
        + 'argument. A user with no assignments has nothing to replicate — do not '
        + 'start a replication for them.',
      );
    }
    templateIds.forEach((t, i) => assertScopeId(`templateIds[${i}]`, t));
    return {
      mode: 'list', userId: scope.userId, since: scope.since, templateIds,
    };
  }

  if (scope.mode !== undefined && scope.mode !== 'fill') {
    throw new Error(
      `[hq-sync] unknown scope.mode ${JSON.stringify(scope.mode)}. It is "fill" `
      + '(the open checklist: {checklistId, templateId, fieldIds}) or "list" '
      + '(My Checklists / Approvals: {userId, since, templateIds}).',
    );
  }

  if (typeof scope.checklistId !== 'string' || scope.checklistId === '') {
    throw new Error(
      '[hq-sync] scope.checklistId is required — it is the open checklist the '
      + 'replication is scoped to (preference architecture/C-2). For the LIST '
      + 'views pass {mode:"list", userId, since, templateIds} instead.',
    );
  }
  assertScopeId('checklistId', scope.checklistId);
  // REQUIRED since the G6 fix round: an omitted templateId used to widen
  // `templates` to the whole non-archived collection (F-5).
  assertScopeId('templateId', scope.templateId);
  const fieldIds = (scope.fieldIds || []).filter((f) => typeof f === 'string' && f !== '');
  fieldIds.forEach((f, i) => assertScopeId(`fieldIds[${i}]`, f));
  return {
    mode: 'fill',
    checklistId: scope.checklistId,
    templateId: scope.templateId,
    fieldIds,
  };
}

/**
 * A short, stable fingerprint of one collection's serialized scope filter.
 *
 * Feeds `replicationIdentifier`, so it must be deterministic across reloads and
 * across processes — no `Math.random`, no insertion order dependence beyond the
 * filter tree's own. Two independent 32-bit hashes (FNV-1a and djb2) are
 * concatenated: 64 bits over the handful of scopes one device ever holds, so a
 * collision — which would re-introduce F-1 for the colliding pair — is not a
 * practical concern. Synchronous on purpose: `startHQReplication` is not async.
 */
export function scopeFingerprint(serialized) {
  let fnv = 0x811c9dc5;
  let djb = 5381;
  for (let i = 0; i < serialized.length; i++) {
    const c = serialized.charCodeAt(i);
    fnv = Math.imul(fnv ^ c, 0x01000193) >>> 0;
    djb = ((Math.imul(djb, 33) >>> 0) + c) >>> 0;
  }
  return fnv.toString(16).padStart(8, '0') + djb.toString(16).padStart(8, '0');
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
  if (s.mode === 'list') return listScopeFilterFor(collectionKey, s);
  switch (collectionKey) {
    case 'templates':
      // No fallback. `templateId` is required by `normalizeScope`, so there is
      // no "caller forgot the argument" path that widens this to the whole
      // non-archived collection (F-5).
      return { op: 'eq', column: 'id', value: s.templateId };
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
 * The LIST scope's per-collection filter — the 2026-08-02 amendment.
 *
 * Every clause below uses a column that ALREADY EXISTS in
 * `sync-schema/sql/0001_sync_tables.sql`. Nothing here needs a new column, a
 * view or a queryable key, which is why this card's PARK trigger did not fire —
 * recorded so the next reader does not re-derive it.
 *
 * @returns {object|null} a filter node, or null for a collection with no list
 *   shape (which none currently is — `applyScope`/`scopePlanFor` treat null as a
 *   programming error, never as permission to pull whole).
 */
function listScopeFilterFor(collectionKey, s) {
  switch (collectionKey) {
    case 'templates':
      // The assigned set. NOT `archived_at.is.null` — that is the widening F-5
      // deleted, and it is the whole non-archived collection.
      return { op: 'in', column: 'id', values: s.templateIds };
    case 'checklists':
      // BOTH halves of the amended rule, and one row of the test fixture proves
      // each: `template_id.in` is "never all users" (the assigned set),
      // `submitted_at.gte` is "never all history" (the floor).
      return {
        op: 'and',
        clauses: [
          { op: 'in', column: 'template_id', values: s.templateIds },
          { op: 'gte', column: 'submitted_at', value: s.since },
        ],
      };
    case 'responses':
      // 🛑 THE FLOOR ALONE, AND THE REASON IS NOT LAZINESS.
      // `submission_responses` carries no `template_id`, and its `field_id`
      // resolves to a template only through `checklist_fields`, which is NOT a
      // replicated collection and is not queryable over the door. Scoping by
      // `submission_id` is refused for the same reason 0003 §5c refuses it for
      // the read policy: a DRAFT has `submission_id IS NULL`, and drafts are
      // exactly what a crew member fills offline. So the expressible client-side
      // bound is the floor; the per-user narrowing is
      // `submission_responses_select` = `hq_can_see_field(field_id)`, live per
      // row through the FDW.
      return { op: 'gte', column: 'answered_at', value: s.since };
    case 'approvals':
      // Same shape and the same reason: `submission_rejections` carries no
      // template_id and no approver column at all (`rejected_by` is who WROTE
      // the rejection, which would hide from an assignee the feedback written
      // ABOUT them — the reject-with-comment path V18/WP7 exist for). The floor
      // is the bound; `submission_rejections_select` = `hq_can_see_field` is the
      // gate, and it admits the approver AND the assignee, which is what
      // decision 111 consequence (1) chose.
      return { op: 'gte', column: 'rejected_at', value: s.since };
    default:
      return null;
  }
}

// ---------------------------------------------------------------------------
// THE LIVE LEG — B-42 option (i), with the residual named rather than implied.
//
// MEASURED against the committed bundle, not read from docs. The plugin's live
// subscription is:
//
//     e.client.channel(e.replicationIdentifier)
//      .on("postgres_changes", {event:"*", schema:"public", table:e.tableName}, …)
//
// — no `filter` key, and no option that would let a caller supply one. So the
// PULL is scoped (above) and the LIVE leg is not: a row belonging to a checklist
// this device never opened, CHANGED while the page is open, still lands in
// IndexedDB and nothing evicts it. `B-42 SYNC-REALTIME-SCOPE`, widened to a
// fourth collection by `B-49` when `submission_rejections` gained a SELECT
// policy.
//
// B-42 offered three options in ascending cost. This card applies **(i)**: the
// single `column=op.value` clause Realtime's `postgres_changes` does accept,
// applied where the scope is expressible as one clause. Since the plugin gives
// no seam, the filter is injected at the CLIENT's `channel()` — the same
// category of shim as `makeSyncFetch` and `makeRealtimeTransport`, i.e. a public
// extension point on the client object rather than a fork of the plugin
// (option (iii)).
//
// 🛑 THE FILTER IS A COARSE PRE-FILTER, NOT THE SCOPE. Where the pull scope is
// two clauses (`checklists` under a list scope) the live filter carries the ONE
// that RLS does not already deliver. RLS is evaluated per subscriber on the
// Realtime leg too, so the USER axis is already bounded server-side; the axis it
// does not bound is TIME, which is why the floor is the clause that ships.
// The pull remains authoritative — a filter that admits a superset costs noise,
// never correctness.
//
// 🛑 AND `responses` GETS NOTHING. See `realtimeFilterFor`.
// ---------------------------------------------------------------------------

/**
 * The single `column=op.value` clause for one collection's live subscription,
 * or `null` for "this collection's live leg stays unfiltered".
 *
 * 🛑 `responses` IS ALWAYS null, in BOTH modes, and there are two reasons — the
 * second is the one that would bite:
 *
 *   1. UNDER A FILL SCOPE THE PREDICATE IS GENUINELY TWO-BRANCH:
 *      `or(submission_id.eq.X, and(submission_id.is.null, field_id.in.(…)))`.
 *      One `column=op.value` clause cannot express it, and the branch that would
 *      have to go is the DRAFT branch — the collection's whole reason to exist.
 *
 *   2. UNDER A LIST SCOPE the predicate IS one clause (`answered_at.gte.<since>`)
 *      and it is still refused, because `answered_at` is CLIENT-STAMPED while
 *      the pull cursor `_modified` is TRIGGER-stamped (0001). An offline draft
 *      answered yesterday and pushed today would be dropped by a live filter on
 *      `answered_at` and admitted by the pull — the live leg would go silently
 *      blind to exactly the late-arriving offline write this system exists to
 *      converge. Filtering in one mode and not the other would also make the
 *      live leg's coverage depend on which tab is open, and a filter that is
 *      present sometimes reads as a guarantee it is not.
 *
 * The residual is therefore UNCHANGED for `responses` and is recorded at the
 * call site in `startHQReplication` as well as here. B-42 stays open on it.
 */
export function realtimeFilterFor(collectionKey, scope) {
  const s = normalizeScope(scope);
  if (collectionKey === 'responses') return null;
  if (s.mode === 'list') {
    switch (collectionKey) {
      case 'templates': return `id=in.(${s.templateIds.join(',')})`;
      // The floor — the axis RLS does not bound. See the block above.
      case 'checklists': return `submitted_at=gte.${s.since}`;
      case 'approvals': return `rejected_at=gte.${s.since}`;
      default: return null;
    }
  }
  switch (collectionKey) {
    case 'templates': return `id=eq.${s.templateId}`;
    case 'checklists': return `id=eq.${s.checklistId}`;
    case 'approvals': return `submission_id=eq.${s.checklistId}`;
    default: return null;
  }
}

/** Where a client's per-channel Realtime filters live. Non-enumerable. */
const REALTIME_FILTER_REGISTRY = '__hqRealtimeFilters';

function realtimeFilterRegistry(client) {
  if (!client[REALTIME_FILTER_REGISTRY]) {
    Object.defineProperty(client, REALTIME_FILTER_REGISTRY, {
      value: Object.create(null), enumerable: false, configurable: true, writable: false,
    });
  }
  return client[REALTIME_FILTER_REGISTRY];
}

/**
 * Record the filter a given replication's channel must subscribe with.
 * Keyed by `replicationIdentifier`, which is exactly the name the plugin passes
 * to `client.channel(...)` — so the shim needs no other agreement with it.
 */
export function registerRealtimeFilter(client, replicationIdentifier, filter) {
  if (!client || typeof client !== 'object') return client;
  if (filter === null || filter === undefined) return client;
  realtimeFilterRegistry(client)[replicationIdentifier] = filter;
  return client;
}

/**
 * Wrap `client.channel` so a registered filter is merged into the plugin's
 * hard-coded `postgres_changes` binding config.
 *
 * Idempotent — installing twice is a no-op, so a caller who constructs a client
 * and a caller who receives one cannot double-wrap.
 *
 * Deliberately conservative: it touches ONLY `postgres_changes` bindings, ONLY
 * on channels that have a registered filter, and ONLY when the caller did not
 * supply a `filter` of its own. Everything else is passed through untouched, so
 * an upgrade that starts supplying its own filter wins rather than silently
 * losing to ours (and `tests/sync-rxdb-client.spec.js` reds when the plugin's
 * binding shape moves).
 */
export function installRealtimeFilterShim(client) {
  if (!client || typeof client.channel !== 'function') return client;
  const filters = realtimeFilterRegistry(client);
  if (client.__hqRealtimeShimInstalled) return client;
  Object.defineProperty(client, '__hqRealtimeShimInstalled', {
    value: true, enumerable: false, configurable: true,
  });

  const channel = client.channel.bind(client);
  client.channel = function hqChannel(name, ...rest) {
    const ch = channel(name, ...rest);
    if (!ch || typeof ch.on !== 'function') return ch;
    const on = ch.on.bind(ch);
    // Mutating THE INSTANCE (not the prototype) is what keeps chaining honest:
    // supabase-js's `.on()` returns the channel, so `.on(...).on(...)` still
    // hits this override, and `.subscribe()` is untouched.
    ch.on = function hqOn(type, config, callback) {
      const filter = filters[name];
      if (filter && type === 'postgres_changes' && config && config.filter === undefined) {
        return on(type, Object.assign({}, config, { filter }), callback);
      }
      return on(type, config, callback);
    };
    return ch;
  };
  return client;
}

/**
 * Serialise a filter node into PostgREST's embedded `or=`/`and=` grammar.
 * Columns are quoted the way the vendored plugin quotes its own checkpoint
 * clause, so the two compose without a quoting disagreement.
 *
 * 🛑 G6 CORRECTION (F-4). VALUES are quoted too. The column was quoted and the
 * value was not, so a value carrying `,` `(` `)` was read as grammar. Values
 * are already whitelisted by `assertScopeId` — this is the second, independent
 * half, kept because the whitelist lives at a different call site and could be
 * relaxed without anyone noticing this depended on it.
 */
function quoteValue(v) {
  return `"${String(v).replace(/\\/g, '\\\\').replace(/"/g, '\\"')}"`;
}

export function serializeFilter(node) {
  switch (node.op) {
    case 'eq': return `"${node.column}".eq.${quoteValue(node.value)}`;
    // The LIST scope's date floor. Quoted like every other value (F-4).
    case 'gte': return `"${node.column}".gte.${quoteValue(node.value)}`;
    case 'is': return `"${node.column}".is.null`;
    case 'in': return `"${node.column}".in.(${node.values.map(quoteValue).join(',')})`;
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
    case 'gte': return query.gte(node.column, node.value);
    case 'is': return query.is(node.column, null);
    case 'in': return query.in(node.column, node.values);
    // A top-level `or` goes on the wire as one `or=` parameter.
    case 'or': return query.or(node.clauses.map(serializeFilter).join(','));
    // 🛑 G6 CORRECTION (F-9). A top-level `and` used to route through `.or()`,
    // emitting `or=(and(a,b))` — equivalent, but it reads as a bug and would be
    // copied. PostgREST ANDs top-level filters already, so applying each clause
    // in turn is both correct and what the shape says.
    case 'and': return node.clauses.reduce((q, c) => applyScope(q, c), query);
    default: throw new Error('[hq-sync] unknown scope filter op: ' + node.op);
  }
}

/**
 * The `pull.queryBuilder` the replication plugin calls. Returns the narrowed
 * query; the plugin then ANDs its own checkpoint clause onto it.
 *
 * 🛑 G6 CORRECTION (F-3). The null-node refusal is raised HERE, at build time,
 * not inside the returned handler. The plugin wraps `pull.handler` in
 * `try{…}catch{ emit RC_PULL; await retry }` — an unbounded retry loop feeding
 * an error stream nobody subscribes to — so a fifth replicated collection with
 * no `case` in `scopeFilterFor` would have spun forever instead of refusing.
 * `applyScope` keeps its own throw as a second line for direct callers.
 */
/**
 * The scope's IDENTITY, prefixed onto the fingerprint input.
 *
 * 🛑 WHY IT IS NOT ENOUGH TO HASH THE SERIALIZED FILTER ALONE (S1a). Under a
 * LIST scope, `approvals` serialises to `"rejected_at".gte."<since>"` and
 * NOTHING ELSE — no user appears in it, because no replicated table carries a
 * queryable per-user key. Two crew members signing into one truck phone on the
 * same day would therefore hash to the SAME `replicationIdentifier`, and RxDB
 * keys its persisted checkpoint by `[collection.name, replicationIdentifier]`
 * and by nothing else (see the CHECKPOINT block above). The second user would
 * resume the first user's cursor and their own rows would be filtered away
 * permanently — F-1's data loss, reached down a different road.
 *
 * The identity is also what keeps a LIST scope and a FILL scope from ever
 * sharing a checkpoint, which they must not: their result sets are different
 * shapes over the same tables.
 */
function scopeIdentity(s) {
  return s.mode === 'list' ? `list:${s.userId}` : `fill:${s.checklistId}`;
}

export function scopePlanFor(collectionKey, scope) {
  const s = normalizeScope(scope);
  const node = scopeFilterFor(collectionKey, s);
  if (!node) {
    throw new Error(
      `[hq-sync] collection "${collectionKey}" has no scope filter, so it would be `
      + 'pulled whole (preference architecture/C-2). Add a case to scopeFilterFor, '
      + 'or do not replicate it.',
    );
  }
  const serialized = serializeFilter(node);
  return {
    node,
    serialized,
    fingerprint: scopeFingerprint(`${scopeIdentity(s)} ${serialized}`),
    // B-42 option (i). `null` for `responses` — see `realtimeFilterFor`.
    realtimeFilter: realtimeFilterFor(collectionKey, s),
    queryBuilder: ({ query }) => applyScope(query, node),
  };
}

export function makePullQueryBuilder(collectionKey, scope) {
  return scopePlanFor(collectionKey, scope).queryBuilder;
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
 * no unscoped call, and there are TWO shapes:
 *
 *   FILL  `{checklistId, templateId, fieldIds}` — the open checklist. Both ids
 *         mandatory (an omitted `templateId` used to widen `templates` to the
 *         whole non-archived collection — G6 F-5); add `fieldIds` so the
 *         offline drafts come with it.
 *   LIST  `{mode:'list', userId, since, templateIds}` — My Checklists /
 *         Approvals, per the 2026-08-02 operator decision. `since` is the DATE
 *         FLOOR and is MANDATORY; `templateIds` must be non-empty.
 *
 * 🛑 CANCEL BEFORE RE-SCOPING. Each returned state owns a Realtime channel
 * named after its `replicationIdentifier` and keeps writing into `db[key]`.
 * Starting a second scope without `.cancel()`ing the first leaves two live
 * replications on the same local collections. Nothing here enforces it —
 * `sync-hard-cutover` owns the page lifecycle that must.
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
  // 🛑 EVERY collection's scope plan is computed BEFORE a single replication is
  // started (F-3). A collection with no scope case must refuse at the call, not
  // lazily inside `pull.handler` where the plugin's `catch{ retry }` would turn
  // the refusal into an unbounded silent loop — and it must refuse before three
  // of the four collections are already live.
  const plans = {};
  for (const key of Object.keys(REPLICATED_COLLECTIONS)) {
    plans[key] = scopePlanFor(key, scope);
  }
  for (const [key, def] of Object.entries(REPLICATED_COLLECTIONS)) {
    const plan = plans[key];
    const replicationIdentifier = `hq-sync-${def.table}-${plan.fingerprint}`;
    // 🛑 B-42 OPTION (i), AND ITS RESIDUAL, AT THE CALL SITE.
    //
    // Registered BEFORE `replicate(...)`, because the plugin opens its channel
    // inside `start()` and a filter registered afterwards would apply to the
    // NEXT subscription rather than this one.
    //
    // `plan.realtimeFilter` is null for exactly one collection — `responses` —
    // and `registerRealtimeFilter` treats null as "register nothing", so its
    // live subscription goes out exactly as wide as it does today. THE RESIDUAL
    // IS THEREFORE: a `submission_responses` row outside this scope, CHANGED
    // while the page is open, still reaches this device and nothing evicts it.
    // That is B-42 for `responses` only, still open, and the reason is in
    // `realtimeFilterFor`'s docblock (two-branch predicate under a fill scope;
    // a client-stamped `answered_at` that would blind the live leg to
    // late-arriving offline drafts under a list scope).
    registerRealtimeFilter(client, replicationIdentifier, plan.realtimeFilter);
    const state = replicate({
      // 🛑 THE SCOPE IS PART OF THE IDENTIFIER (G6 F-1). RxDB keys the persisted
      // checkpoint by `hash([collection.name, replicationIdentifier])` and by
      // nothing else, so one identifier across scopes means ONE checkpoint
      // across scopes — and since the checkpoint is `{modified}` of the last row
      // of the SCOPED result set, switching to an older checklist filters every
      // one of its rows away, permanently. See the CHECKPOINT block above.
      //
      // Stable for a GIVEN scope, so reconnecting to the same checklist still
      // RESUMES rather than re-pulling (the property spike `proof-lww.js`
      // depends on). Changing scope is the only thing that mints a new one, and
      // the re-pull that buys is one checklist's rows — about one batch.
      replicationIdentifier,
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
        queryBuilder: plan.queryBuilder,
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
