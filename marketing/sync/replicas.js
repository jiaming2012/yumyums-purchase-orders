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
 * The §8 policy lookup, fed by the campaigns replica — the function
 * submit-flow.js hands to its policy seam (the shape setCampaignPolicy
 * expects: campaignId → {requiresOnline} | null).
 *
 * Synchronous by design: the machine's RESOLVED event needs the answer at
 * resolve time, so this keeps a reactive-query-fed Map mirror of the local
 * collection (RxDB queries exclude soft-deleted rows, so a deleted campaign
 * honestly drops back to unknown). Unknown campaign → null, and the caller's
 * policyFor coerces null to false — the ratified unknown→false default
 * (decision 166) survives for the cases that are GENUINELY unknown; this card
 * removes "unknown" for replicated campaigns, it does not change what unknown
 * means.
 *
 * @param {object} campaignsCollection  the CAMPAIGNS_COLLECTION RxCollection
 * @returns {{policyFor: function(string): ({requiresOnline: boolean}|null),
 *            size: function(): number, stop: function(): void}}
 */
export function createCampaignPolicySource(campaignsCollection) {
  const byId = new Map();
  const sub = campaignsCollection.find().$.subscribe((docs) => {
    byId.clear();
    for (const d of docs) byId.set(d.id, !!d.requires_online);
  });
  return {
    policyFor: (campaignId) =>
      (byId.has(campaignId) ? { requiresOnline: byId.get(campaignId) } : null),
    size: () => byId.size,
    stop: () => sub.unsubscribe(),
  };
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
