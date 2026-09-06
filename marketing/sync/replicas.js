// marketing/sync/replicas.js — the Marketing pull replicas (card
// rxdb-pull-replica, run 20260905; campaigns replica + policy source added by
// card requires-online-replication, run 20260906; design
// docs/qr-offline-redemption-handoff.md §5.3/§7.3/§8/§10).
//
// The codes/offers replicas pull the SAME server table, `public.codes`; the
// campaigns replica (below) pulls `public.campaigns` for the §8 policy flag.
// The roadmap left
// "one table or two" as Activity A's schema call, and Activity A built ONE
// table (supabase/migrations/ carries no entitlements table), so:
//
//   * CODES / redemption-state replica — bounded
//     `expires_at > now() - interval '2 days'` (§5.3): the scanner can tell
//     OFFLINE that a code is spent or freshly expired, instead of "unknown".
//   * OFFERS replica — bounded `expires_at > now()` and KEYED ON THE CUSTOMER
//     HASH (§10): under the identity-code model the QR encodes who the
//     customer is, the device hashes it, and `token_hash` IS that hash. A
//     synced customer's full offer list is every live row for their hash —
//     length 0..1 under tonight's one-code-per-offer schema, length N
//     unchanged when the identity-code card lands.
//
// Redemption state is deliberately NOT filtered out of the offers pull (no
// `redeemed_at=is.null` server filter): a row that stops matching a pull
// filter simply stops arriving, so the local copy would stay stale-positive
// forever. Redeemed rows flow through as updates; resolveOffers() excludes
// them locally.
//
// Like pull-replication.js this module imports only its sibling — the RxDB
// primitives arrive injected, so the Node gate harness and the browser run the
// same code. Browser wiring (Cards 5/6): import the vendored bundle, build a
// Subject per replica, and pass `replicateRxCollection` + Dexie collections in.

import {
  makePullHandler,
  startPullReplica,
  REPLICA_SELECT,
} from './pull-replication.js';

export { REPLICA_SELECT };

/** §5.3 — the codes replica keeps recently-expired rows for two days. */
export const CODES_WINDOW_MS = 2 * 24 * 3600 * 1000;

export const codesWindowBound = (now = Date.now) =>
  new Date(now() - CODES_WINDOW_MS).toISOString();

/** The offers replica carries live offers only. */
export const offersWindowBound = (now = Date.now) => new Date(now()).toISOString();

/**
 * One RxDB schema serves both collections — the two replicas differ by pull
 * filter, not by shape. §10: minimal on-device row — hashed identity and
 * entitlement state, no PII. `token_hash` is indexed because it is the lookup
 * key for BOTH surfaces (scan → hash → redemption state; hash → offer list).
 * `_deleted` is not declared: RxDB owns the soft-delete field itself.
 */
export const MARKETING_REPLICA_SCHEMA = {
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
};

/**
 * The campaigns replica's on-device row (card requires-online-replication,
 * run 20260906) — §10 minimal: the §8 policy flag and the checkpoint key,
 * nothing else. No name, no face_value (PII-free by shape). The mechanism is
 * the SAME shipped pull (replicateRxCollection + makePullHandler + the GAP-1
 * keyset checkpoint); only the expiry bound is absent — campaigns has no
 * expires_at (spike build-fact 1; the bound is optional in
 * pull-replication.js, never removed from codes/offers).
 */
export const CAMPAIGNS_REPLICA_SCHEMA = {
  version: 0,
  primaryKey: 'id',
  type: 'object',
  properties: {
    id: { type: 'string', maxLength: 100 },
    requires_online: { type: 'boolean' },
    updated_at: { type: 'string' },
  },
  required: ['id', 'requires_online', 'updated_at'],
};

/** The columns the campaigns replica pulls — mirrors the schema exactly. */
export const CAMPAIGNS_SELECT = 'id,requires_online,updated_at';

/** Collection names the browser database and the harness share. */
export const CODES_COLLECTION = 'codes';
export const OFFERS_COLLECTION = 'offers';
export const CAMPAIGNS_COLLECTION = 'campaigns';

/** addCollections() argument covering all three replicas. */
export function marketingCollectionSpec() {
  return {
    [CODES_COLLECTION]: { schema: MARKETING_REPLICA_SCHEMA },
    [OFFERS_COLLECTION]: { schema: MARKETING_REPLICA_SCHEMA },
    [CAMPAIGNS_COLLECTION]: { schema: CAMPAIGNS_REPLICA_SCHEMA },
  };
}

function startReplica(deps, { table, windowBound, select, replicationIdentifier }) {
  const {
    replicateRxCollection, collection, restUrl, bearer, fetchImpl, stream$,
    batchSize = 50, clock, now = clock ? clock.now : Date.now,
    requestLog, waitForLeadership = false,
  } = deps;
  const pullHandler = makePullHandler({
    restUrl,
    table,
    // A table with no expiry column runs checkpoint-only (windowBound omitted).
    windowBound: windowBound ? () => windowBound(now) : undefined,
    bearer,
    fetchImpl,
    requestLog,
    clock,
    ...(select ? { select } : {}),
  });
  return startPullReplica({
    replicateRxCollection,
    collection,
    replicationIdentifier,
    pullHandler,
    stream$,
    batchSize,
    waitForLeadership,
  });
}

/**
 * The codes / redemption-state replica (§5.3 window).
 * @param {object} deps  {replicateRxCollection, collection, restUrl, bearer,
 *                        fetchImpl, stream$, batchSize?, clock?, now?,
 *                        requestLog?, waitForLeadership?, replicationIdentifier?}
 *   `clock` (card clock-offset-on-sync, §5.1): a createSyncClock instance
 *   (clock.js). Every successful pull calibrates it from the response's Date
 *   header, and — unless an explicit `now` is injected — every window bound
 *   follows clock.now, so a skewed device clock stops distorting the §5.3
 *   window. Offline expiry checks are clock.isExpired(expires_at); pass
 *   {now: clock.now} into resolveOffers. An explicit `now` still wins
 *   (harness/test injection precedence).
 */
export function startCodesReplica(deps) {
  return startReplica(deps, {
    table: 'codes',
    windowBound: codesWindowBound,
    replicationIdentifier: deps.replicationIdentifier || 'marketing-codes-pull',
  });
}

/** The offers replica (§10 — live rows only, resolved by customer hash). */
export function startOffersReplica(deps) {
  return startReplica(deps, {
    table: 'codes',
    windowBound: offersWindowBound,
    replicationIdentifier: deps.replicationIdentifier || 'marketing-offers-pull',
  });
}

/**
 * The campaigns replica (card requires-online-replication) — carries the §8
 * `requires_online` policy flag to the device, AND a change to it: the
 * spike-decisive leg was the FLIP (a campaign downgraded while its codes sit
 * still never re-delivers through a codes-embed — that alternative is CLOSED,
 * with evidence). No expiry bound: campaigns has no expires_at column.
 * Server side, migration 20260906000100 makes the table self-announcing
 * (supabase_realtime membership + a touch trigger) so the same
 * wireRealtimeResync mechanism codes use nudges this replica too.
 */
export function startCampaignsReplica(deps) {
  return startReplica(deps, {
    table: 'campaigns',
    windowBound: null,
    select: CAMPAIGNS_SELECT,
    replicationIdentifier: deps.replicationIdentifier || 'marketing-campaigns-pull',
  });
}

/**
 * The settle tick after `awaitInitialReplication()` resolves before the source
 * calls itself ready (card refusal-holds-before-sync, build-fact 2). Spike 01
 * measured NO race on memory storage — the Map already held both campaigns AT
 * the resolve tick — but the browser runs Dexie, whose write pipeline is not
 * the same shape. Cheap insurance on a path that runs once per session.
 */
export const POLICY_SETTLE_MS = 150;

/**
 * The §8 policy lookup, fed by the campaigns replica — the function
 * submit-flow.js hands to its policy seam (the shape setCampaignPolicy
 * expects: campaignId → {requiresOnline, unresolved} | null).
 *
 * ───────────────────────────────────────────────────────────────────────────
 * 🛑 FAIL-CLOSED FOR A KNOWN CODE WHOSE CAMPAIGN IS UNRESOLVED — B-432.
 * ───────────────────────────────────────────────────────────────────────────
 *
 * This function used to answer `null` for any campaign not in its Map, and
 * submit-flow.js coerces `null → false`. That is a fail-OPEN: a known,
 * replicated, entitlement-bearing `requires_online = true` code whose CAMPAIGN
 * had not yet arrived was offline-overridable. Demonstrated, not asserted —
 * spike 02 row 3 drove the shipped machine through it with the real replicas
 * (`overrideAvailable = true`, OVERRIDE_REQUEST → `overrideConfirm` on the $40
 * catering credit), and the browser half reproduces in the branch-3 e2e with
 * only the `campaigns:` seed removed.
 *
 * The predicate, and why it is this one (build-fact 3):
 *
 *   campaignId == null  → null      A genuinely-unknown CODE. It names no
 *                                   campaign, so there is nothing to resolve
 *                                   and nothing to fail closed about. The
 *                                   caller coerces null → false and the
 *                                   offline override stays available with the
 *                                   F2 unverified warning. **Decision 166's
 *                                   ratified affordance survives BY
 *                                   CONSTRUCTION** — a known code always names
 *                                   a campaign, so "genuinely unknown" can
 *                                   never enter the arm below.
 *
 *   in the Map          → the flag  Unchanged (run 20260906).
 *
 *   KNOWN but absent    → {requiresOnline: true, unresolved: true}
 *                                   Empty replica, still delivering, erroring,
 *                                   OR a brand-new campaign whose codes
 *                                   arrived first. Note this is NOT gated on
 *                                   replica readiness: a readiness latch alone
 *                                   leaves the codes-arrive-first window open
 *                                   (build-fact 3), and this predicate
 *                                   subsumes it.
 *
 * `unresolved` rides on the answer so the caller can tell "refused because the
 * operator said so" from "refused because we could not tell" — the render and
 * the attempt record both need that distinction, and neither may guess it.
 *
 * Synchronous by design: the machine's RESOLVED event needs the answer at
 * resolve time, so this keeps a reactive-query-fed Map mirror of the local
 * collection.
 *
 * 🛑 A server-side campaign DELETE does not reach this Map (B-434(b), the
 * comment this replaces got it wrong): `public.campaigns` has no `_deleted`
 * column — unlike `codes` — and `CAMPAIGNS_SELECT` does not ask for one, so no
 * soft-delete tombstone is ever pulled and the local row simply persists.
 * There is therefore no "drops back to unknown" behavior to rely on. It is
 * fail-SAFE in direction (a stale `requires_online = true` keeps refusing, and
 * a stale `false` is the flag the operator last published), but it is a real
 * gap in the delete path — carried as B-434(b)'s disposition, not claimed as
 * working.
 *
 * ── the error latch (build-fact 1) ────────────────────────────────────────
 * `attach(replicationState)` wires the campaigns replica in. Two facts make
 * the shape non-obvious, both MEASURED by spike 01:
 *
 *   * `error$` does NOT replay to late subscribers (0 emissions to a
 *     subscriber attached mid-error), so the source must subscribe BEFORE the
 *     replica gets going and hold the last error itself. `attach` is called
 *     synchronously on the line after `startCampaignsReplica()` returns, with
 *     no `await` in between — put one there and the latch silently never
 *     latches.
 *   * the emission carries the pull handler's thrown message verbatim
 *     (`"[marketing-sync] pull campaigns answered HTTP 503"`), so the HTTP
 *     status is attributable for free — no wrapper needed.
 *
 * `unresolved()` is the SOURCE-level reading — "the campaigns replica has not
 * delivered / is erroring" — and is a different question from a single
 * campaign's answer. It feeds the `policy_unresolved` discriminator for
 * genuinely-unknown codes, where there is no campaign id to ask about. With no
 * replica attached at all it reads FALSE: nothing failed, because nothing was
 * started (today's page, before provisioning lands).
 *
 * @param {object} campaignsCollection  the CAMPAIGNS_COLLECTION RxCollection
 * @param {{settleMs?: number}} [opts]
 * @returns {{policyFor: function(string): ({requiresOnline: boolean, unresolved: boolean}|null),
 *            attach: function(object): object, unresolved: function(): boolean,
 *            ready: function(): boolean, attached: function(): boolean,
 *            lastError: function(): (string|null),
 *            size: function(): number, stop: function(): void}}
 */
export function createCampaignPolicySource(campaignsCollection, { settleMs = POLICY_SETTLE_MS } = {}) {
  const byId = new Map();
  const sub = campaignsCollection.find().$.subscribe((docs) => {
    byId.clear();
    for (const d of docs) byId.set(d.id, !!d.requires_online);
  });

  let attached = false;
  let ready = false;
  let lastError = null;   // LATCHED — error$ never replays (build-fact 1)
  let errSub = null;
  let settleTimer = null;

  function attach(replicationState) {
    if (!replicationState || attached) return api;
    attached = true;
    // FIRST, before anything awaits: the emission we must not miss is the one
    // on the very first pull attempt (spike 01 saw it at t+145ms).
    try {
      errSub = replicationState.error$.subscribe((err) => {
        lastError = String((err && err.message) || err);
      });
    } catch (e) { /* a handle without error$ is not a reason to brick the scan page */ }
    Promise.resolve()
      .then(() => replicationState.awaitInitialReplication())
      .then(() => new Promise((r) => { settleTimer = setTimeout(r, settleMs); }))
      .then(() => { ready = true; lastError = null; })
      .catch(() => { /* stays unresolved; error$ carries the attributable reason */ });
    return api;
  }

  const api = {
    policyFor(campaignId) {
      // Decision 166: a genuinely-unknown CODE names no campaign. Answering
      // null here is what keeps its offline override alive.
      if (campaignId === null || campaignId === undefined || campaignId === '') return null;
      if (byId.has(campaignId)) return { requiresOnline: byId.get(campaignId), unresolved: false };
      return { requiresOnline: true, unresolved: true };   // fail closed — B-432
    },
    attach,
    // Source-level: has the campaigns replica delivered? Sticky on error until
    // initial replication resolves (over-reporting "unresolved" refuses
    // nothing — the fail-closed arm keys on the Map, never on this).
    unresolved: () => (attached ? (!ready || lastError !== null) : false),
    ready: () => ready,
    attached: () => attached,
    lastError: () => lastError,
    size: () => byId.size,
    stop: () => {
      sub.unsubscribe();
      if (errSub) errSub.unsubscribe();
      if (settleTimer) clearTimeout(settleTimer);
    },
  };
  return api;
}

/**
 * Offline offer resolution — the §10 lookup. Local replica only: no network
 * is touched (RxDB queries never leave the device), so this answers with the
 * radio off.
 *
 * @param {object} offersCollection  the OFFERS_COLLECTION RxCollection
 * @param {string} tokenHash         the customer hash the device computed
 *                                   from the scanned identity QR
 * @param {{now?: function}} [opts]  clock injection (§5.1 offset card)
 * @returns {Promise<Array<{code_id, campaign_id, expires_at}>>} the customer's
 *   live offers — redeemed and locally-expired rows excluded. Empty array for
 *   an unknown hash: the caller's signal to fall back to the QR's embedded
 *   offer (Activity E's card, not this one).
 */
export async function resolveOffers(offersCollection, tokenHash, { now = Date.now } = {}) {
  const docs = await offersCollection
    .find({ selector: { token_hash: tokenHash } })
    .exec();
  const nowMs = now();
  return docs
    .filter((d) => !d.redeemed_at && Date.parse(d.expires_at) > nowMs)
    .map((d) => ({ code_id: d.id, campaign_id: d.campaign_id, expires_at: d.expires_at }))
    .sort((a, b) => Date.parse(a.expires_at) - Date.parse(b.expires_at));
}
