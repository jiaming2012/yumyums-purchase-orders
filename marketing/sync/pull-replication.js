// marketing/sync/pull-replication.js — the bounded, checkpointed pull mechanism
// for the Marketing replicas (card rxdb-pull-replica, run 20260905; design
// docs/qr-offline-redemption-handoff.md §4/§5.3/§7.3; spike record
// .night-crew/knowledge/spikes/activity-b-offline-first-replica/rxdb-pull-replica.md).
//
// This file is the productionized form of the spike's proven pull handler, with
// the spike's two binding build-facts honored and its GAP-1 closed:
//
//   1. URL-ENCODING (build-fact): a raw `+00:00` in a query string decodes as a
//      space and breaks PostgREST's timestamptz parse. Every cursor/bound value
//      below goes through encodeURIComponent.
//   2. KEYSET CHECKPOINT, not a bare `gt` cursor (GAP-1): the spiked
//      `updated_at=gt.<ts>` cursor silently SKIPS rows that share the last
//      row's `updated_at` when a batch boundary falls inside the tie group —
//      the silent-miss class is a redemption that never reaches device B. The
//      checkpoint here is `{updated_at, id}` and the server predicate is the
//      compound keyset
//          (updated_at > ts) OR (updated_at = ts AND id > id)
//      with `order=updated_at.asc,id.asc`, so a tie group of ANY size is walked
//      deterministically. (The one-row over-fetch alternative was rejected: a
//      tie group larger than one batch makes `gte`+client-skip return zero new
//      documents while rows remain — a livelock, not just a re-read.)
//
// DEPENDENCY-INJECTED ON PURPOSE — this module imports nothing. The RxDB
// primitive (`replicateRxCollection`), the fetch implementation and the
// Realtime client all arrive as parameters, so the SAME file runs:
//   * in the browser on the committed vendor/rxdb.bundle.js (Dexie storage),
//   * in the Node gate harness on the QA rxdb (memory storage + validation).
// That is what makes marketing/sync/harness/ a real gate on the shipped code.

/** RxDB soft-delete contract column — matches replicateRxCollection's default
 *  `deletedField` and the `_deleted` column Activity A's migration carries. */
export const DELETED_FIELD = '_deleted';

/** The columns both replicas pull. §10: keep the on-device row minimal —
 *  hashed identity + entitlement state, no PII. */
export const REPLICA_SELECT =
  'id,token_hash,campaign_id,expires_at,redeemed_at,redeemed_by,updated_at,_deleted';

/** The pre-history checkpoint: every real row sorts after it. The id floor is
 *  the all-zeros uuid because `codes.id` is a uuid column — an empty-string
 *  floor would not parse as a uuid literal server-side. */
export const EPOCH_CHECKPOINT = Object.freeze({
  updated_at: '1970-01-01T00:00:00+00:00',
  id: '00000000-0000-0000-0000-000000000000',
});

/** Accepts null/undefined (first pull) and legacy `{updated_at}`-only
 *  checkpoints (the spike's shape); always returns the keyset pair. */
export function normalizeCheckpoint(checkpoint) {
  if (!checkpoint || !checkpoint.updated_at) return EPOCH_CHECKPOINT;
  return {
    updated_at: checkpoint.updated_at,
    id: checkpoint.id || EPOCH_CHECKPOINT.id,
  };
}

// PostgREST logic-tree value: double-quote (values containing , . : ( ) must
// be quoted inside or=()), THEN percent-encode so URL decoding restores the
// quoted literal intact (build-fact 1: the `+` in `+00:00` must arrive as %2B).
const treeValue = (v) => encodeURIComponent(`"${String(v).replace(/"/g, '')}"`);

/** The GAP-1 fix, as a query-string fragment. */
export function keysetPredicate(checkpoint) {
  const cp = normalizeCheckpoint(checkpoint);
  const ts = treeValue(cp.updated_at);
  const id = treeValue(cp.id);
  return `or=(updated_at.gt.${ts},and(updated_at.eq.${ts},id.gt.${id}))`;
}

/**
 * Build one bounded, checkpointed pull URL.
 *
 * The expiry bound is OPTIONAL, never removed (card
 * requires-online-replication, run 20260906): `campaigns` has no `expires_at`
 * column, so the unconditional bound drew HTTP 400 on that table
 * (spike build-fact 1). A null/undefined `windowIso` omits the fragment;
 * every bounded caller (codes/offers) is byte-identical. The GAP-1 keyset
 * checkpoint is untouched either way.
 *
 * @param {object} p
 * @param {string} p.restUrl      PostgREST origin (no trailing slash)
 * @param {string} p.table        e.g. 'codes'
 * @param {string} p.select       column list
 * @param {object|null} p.checkpoint  RxDB checkpoint (null on first pull)
 * @param {string} [p.windowIso]  the expiry-window floor, ISO 8601 —
 *                                rows must satisfy expires_at > windowIso;
 *                                null/undefined for a table with no expiry
 * @param {number} p.batchSize
 */
export function buildPullUrl({ restUrl, table, select, checkpoint, windowIso, batchSize }) {
  return (
    `${restUrl}/${table}` +
    `?select=${encodeURIComponent(select)}` +
    `&${keysetPredicate(checkpoint)}` +
    (windowIso == null ? '' : `&expires_at=gt.${encodeURIComponent(windowIso)}`) +
    `&order=updated_at.asc,id.asc` +
    `&limit=${batchSize}`
  );
}

/**
 * Make a replicateRxCollection pull.handler — bounded (windowBound) and
 * checkpointed (keyset). Enumerable: pass `requestLog` and every request is
 * recorded {checkpoint, url} BEFORE it is sent (B-216 — evidence is the
 * request log, never an inference from results).
 *
 * @param {object} cfg
 * @param {string} cfg.restUrl
 * @param {string} cfg.table
 * @param {function(): string} [cfg.windowBound]  re-evaluated per request so
 *   the window SLIDES (§5.3). OPTIONAL: omit for a table with no `expires_at`
 *   (campaigns) — the pull is then checkpoint-only. Bounded callers unchanged.
 * @param {string|function(): string} cfg.bearer  device JWT (or getter)
 * @param {function} cfg.fetchImpl
 * @param {string} [cfg.select]
 * @param {Array}  [cfg.requestLog]
 * @param {object} [cfg.clock]  a createSyncClock instance (clock.js). When
 *   given, EVERY successful (HTTP 200) pull response calibrates it from the
 *   response's Date header — §5.1's serverNow source (card clock-offset-on-sync;
 *   spike-proven, no extra endpoint). Optional and additive: clock-less
 *   callers get byte-identical behavior.
 * @param {function} [cfg.onSuccess]  called on EVERY HTTP-200 pull, right
 *   beside the clock capture — the successful-pull edge the B-439 fix keys on
 *   (card sync-coordinates-provisioning). Spike 04 measured this as the ONLY
 *   edge that fires in BOTH recovery shapes (with-docs AND recovery-EMPTY,
 *   where zero new rows arrive) and NEVER on a failed cycle (the throw above
 *   precedes it) — `error$`/`active$`/`remoteEvents$` all fire on 503 cycles
 *   too and are disqualified by measurement. Optional and additive; a thrown
 *   observer never fails a pull that delivered rows.
 */
export function makePullHandler({ restUrl, table, windowBound, bearer, fetchImpl, select = REPLICA_SELECT, requestLog, clock, onSuccess }) {
  return async function pullHandler(checkpoint, batchSize) {
    const cp = normalizeCheckpoint(checkpoint);
    const url = buildPullUrl({ restUrl, table, select, checkpoint: cp, windowIso: windowBound ? windowBound() : null, batchSize });
    if (requestLog) requestLog.push({ checkpoint: cp, url });
    const token = typeof bearer === 'function' ? bearer() : bearer;
    const res = await fetchImpl(url, {
      headers: { Authorization: `Bearer ${token}`, Accept: 'application/json' },
    });
    if (res.status !== 200) {
      throw new Error(`[marketing-sync] pull ${table} answered HTTP ${res.status}`);
    }
    // Capture BEFORE parsing the body: the offset feeds the NEXT request's
    // window bound. A missing/unparseable Date header skips the capture and
    // keeps the previous offset — observable via clock.captures, never fatal
    // to a pull that delivered rows.
    if (clock) clock.captureFromResponse(res);
    // The successful-pull edge (B-439): fired only after the status check —
    // never on a failed cycle — and independent of the body, so it fires on
    // the recovery-EMPTY shape too.
    if (onSuccess) { try { onSuccess(); } catch (e) { /* an observer must not fail a pull that delivered */ } }
    const rows = await res.json();
    const last = rows[rows.length - 1];
    return {
      documents: rows,
      // Keyset checkpoint: BOTH halves advance together. An empty batch keeps
      // the previous checkpoint (never regresses to epoch).
      checkpoint: rows.length ? { updated_at: last.updated_at, id: last.id } : cp,
    };
  };
}

/**
 * §7.3 Realtime wiring: emit RESYNC into pull.stream$ on EVERY
 * postgres_changes frame AND on EVERY 'SUBSCRIBED' status — including every
 * re-SUBSCRIBED after a reconnect. Realtime has no delivery guarantee and no
 * replay, so a (re)subscription always triggers a checkpoint-resumed refetch;
 * frames are treated as nudges only, never as data.
 *
 * @param {object} p
 * @param {object} p.realtimeClient  supabase-js client (channel/removeChannel)
 * @param {string} p.table
 * @param {function} p.emitResync    called with no args; caller forwards into
 *                                   every replica stream$ fed by this table
 * @param {string} [p.channelName]
 * @param {function} [p.onStatus]    (status, err) observer for logs/tests
 * @returns the channel (pass to realtimeClient.removeChannel on teardown)
 */
export function wireRealtimeResync({ realtimeClient, table, emitResync, channelName, onStatus }) {
  return realtimeClient
    .channel(channelName || `marketing-sync-${table}`)
    .on('postgres_changes', { event: '*', schema: 'public', table }, () => emitResync())
    .subscribe((status, err) => {
      if (onStatus) onStatus(status, err);
      if (status === 'SUBSCRIBED') emitResync();
    });
}

/**
 * Start one live, pull-only replica. Thin on purpose: everything that varies
 * between the browser and the harness is a parameter.
 *
 * @param {object} p
 * @param {function} p.replicateRxCollection  from the vendored bundle (browser)
 *                                            or the QA rxdb (harness)
 * @param {object} p.collection               the RxCollection
 * @param {string} p.replicationIdentifier
 * @param {function} p.pullHandler            from makePullHandler
 * @param {object} p.stream$                  observable emitting 'RESYNC'
 *                                            (vendor bundle exports Subject)
 * @param {number} [p.batchSize=50]
 * @param {boolean} [p.waitForLeadership=false]  true needs the leader-election
 *                                               plugin added by the caller
 */
export function startPullReplica({ replicateRxCollection, collection, replicationIdentifier, pullHandler, stream$, batchSize = 50, waitForLeadership = false }) {
  return replicateRxCollection({
    collection,
    replicationIdentifier,
    live: true,
    waitForLeadership,
    pull: { handler: pullHandler, batchSize, stream$ },
  });
}
