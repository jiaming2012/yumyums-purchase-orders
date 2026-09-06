// marketing/sync/push-replication.js — the device-owned, PUSH-ONLY
// scan_attempts mechanism (card scan-attempts-push-conflict, run 20260905;
// design docs/qr-offline-redemption-handoff.md §4/§6/§8; spike record
// .night-crew/knowledge/spikes/activity-b-offline-first-replica/scan-attempts-push-conflict.md).
//
// §4's key structural decision: `codes` is server-owned and pull-only (Card 2,
// ./pull-replication.js + ./replicas.js); `scan_attempts` is device-owned and
// pushes the OPPOSITE way. A device can INSERT its own attempts and can NEVER
// read them back (RLS grants insert-with-check only — 403 on SELECT, spike-
// proven), so everything a device knows arrives through its own push outcomes
// plus its codes-side pull replica. The push handler here is the §6 shape,
// proven by the spike: for each locally-queued `pending` attempt →
// POST /rpc/redeem as the device JWT → land the attempt row with the resolved
// status/reason → resolve the local row → return [] (no conflicts — the master
// accepted what we told it). Already-resolved rows re-push as no-ops.
//
// Two spike build-facts are BINDING here:
//
//   1. `scan_attempts.id` is a uuid column — attempt ids are
//      crypto.randomUUID(); an app-prefixed string draws PostgREST 400 on the
//      landing insert (the spike's red first run).
//   2. GAP-1: redeem-then-land is NOT atomic client-side. If landing the
//      attempt row fails after redeem() succeeded, the push retry re-runs the
//      handler and the re-burn answers `already_used` TO THE WINNING DEVICE
//      ITSELF — a naive handler flips the winner's UI. Closed with two belts:
//
//      * Belt 1 — the burn outcome is PERSISTED onto the local row
//        (`burn_ok`/`burn_reason`) immediately after redeem() answers, BEFORE
//        the landing insert. A retry after a failed landing skips redeem()
//        entirely and retries only the landing — the winner is never re-burned.
//      * Belt 2 — for the window belt 1 cannot cover (the redeem RESPONSE lost
//        after the server committed, or a device death between redeem and the
//        burn patch): `already_used` where the codes replica names OUR OWN
//        device as `redeemed_by` is arbitrated as ACCEPTED. The read is the
//        LOCAL codes pull replica — never a scan_attempts read-back (the table
//        is write-only to devices by design).
//
// The losing device's display data — "already used at 6:42pm by device-a" —
// comes from the SAME local codes replica (`winner_device`/`winner_at` on the
// local row), because the loser provably cannot read the winner's attempt row.
//
// F-2 GUARD (card requires-online-replication, run 20260906): an
// `unverified_code=true` attempt (the §19 F2 unknown-code override — its
// local code_id IS the 64-hex token_hash; no code row exists to name) is
// diverted BEFORE the redeem call and lands directly on the distinct path
// (migration 20260906000200: `code_id` nullable + `token_hash` + a check
// constraint), carrying status 'accepted' + both audit flags. Without the
// guard the redeem-first path drew a deterministic HTTP 400 that HEAD-OF-LINE
// POISONED the queue (spike-measured: 12 redeem attempts, 0 landings, every
// later redemption stranded). The guard touches ONLY unverified rows — the
// two GAP-1 belts below are byte-identical for everything else.
//
// DEPENDENCY-INJECTED ON PURPOSE — this module imports nothing. The RxDB
// primitive (`replicateRxCollection`), the fetch implementation and the
// collections arrive as parameters, so the SAME file runs:
//   * in the browser on the committed vendor/rxdb.bundle.js (Dexie storage;
//     the bundle exports replicateRxCollection since Card 2's widening),
//   * in the Node gate harness on the QA rxdb (memory storage + validation).
// That is what makes marketing/sync/harness/push-run.sh a real gate on the
// shipped code.

/**
 * The local scan-attempts queue schema.
 *
 * Server-taxonomy fields mirror §4's scan_attempts columns exactly — `status`
 * stays `pending | accepted | rejected` and `reason`
 * `already_used | expired | not_found | null`; this module adds NO terminal
 * status (the slate's PARK line). Five fields are LOCAL-ONLY bookkeeping and
 * render data, never sent in the landing insert:
 *
 *   burn_ok / burn_reason  — the persisted redeem() outcome (GAP-1 belt 1)
 *   landed                 — the server attempt row is confirmed inserted
 *   winner_device / winner_at — the loser's render data, filled from the
 *                               codes pull replica
 */
export const SCAN_ATTEMPTS_SCHEMA = {
  // v1 (card refusal-holds-before-sync, run 20260906-2): + policy_unresolved.
  // 🛑 THE FIRST RxDB SCHEMA MIGRATION IN THIS TREE. Three things move
  // together or the Scan page bricks — see SCAN_ATTEMPTS_MIGRATION_STRATEGIES
  // below and `rxdb/plugins/migration-schema` in vendor/src/rxdb-hq-entry.mjs.
  version: 1,
  primaryKey: 'id',
  type: 'object',
  properties: {
    id: { type: 'string', maxLength: 36 },        // crypto.randomUUID() — uuid column (build-fact 1)
    code_id: { type: 'string', maxLength: 100 },
    device_id: { type: 'string', maxLength: 100 },
    scanned_at: { type: 'string' },
    status: { type: 'string' },                   // pending | accepted | rejected (§4, unchanged)
    reason: { type: ['string', 'null'] },         // already_used | expired | not_found | null
    offline_override: { type: 'boolean' },        // §13 permissioned override
    override_by: { type: ['string', 'null'] },
    unverified_code: { type: 'boolean' },         // §19 F2
    // The B-432 discriminator (card refusal-holds-before-sync): true when the
    // device could NOT resolve §8 campaign policy for this attempt — the
    // campaigns replica was empty, still delivering, or erroring. Paired with
    // unverified_code it separates a campaigns-replica FAILURE override (t,t)
    // from a genuinely-unknown-campaign override (t,f); both stay
    // status='accepted' (§9/§19 taxonomy unchanged, no new terminal status).
    // Captured at SCAN time, never at push time — by the moment the queue
    // drains the replica may well have recovered, and the record is about what
    // the device knew when the crew member forced the submit.
    policy_unresolved: { type: 'boolean' },
    pos_order_number: { type: ['string', 'null'] },
    pos_business_date: { type: 'string' },        // UTC date of scanned_at (§13's card owns Toast semantics)
    redeemed_value: { type: ['number', 'null'] },
    // ---- local-only, never in the landing insert ----
    burn_ok: { type: ['boolean', 'null'] },
    burn_reason: { type: ['string', 'null'] },
    landed: { type: 'boolean' },
    winner_device: { type: ['string', 'null'] },
    winner_at: { type: ['string', 'null'] },
  },
  required: ['id', 'code_id', 'device_id', 'scanned_at', 'status', 'pos_business_date'],
  indexes: [['code_id']],
};

/** Collection name the browser database and the harness share. */
export const SCAN_ATTEMPTS_COLLECTION = 'scan_attempts';

/**
 * THE DEVICE MIGRATION STRATEGY (card refusal-holds-before-sync) — the first
 * one in this tree, so it is written out as a named mechanism rather than an
 * inline literal.
 *
 * What this protects: a crew phone that went offline mid-shift is holding
 * `pending` attempts in v0 shape — unsent redemptions, each one a coupon a
 * customer already walked away with. RxDB stores documents per schema version,
 * so a v1 collection reaches v0 data ONLY through a migration. The three
 * mechanisms that must ship together:
 *
 *   1. `version: 1` on the schema above;
 *   2. this strategy map, passed to addCollections;
 *   3. `RxDBMigrationSchemaPlugin`, bundled in vendor/rxdb.bundle.js and
 *      registered with addRxPlugin by the page (scan-page.js).
 *
 * Miss (3) and rxdb 17.4.0's `autoMigrate && version !== 0 &&
 * await migratePromise()` hits the un-plugged prototype stub, which THROWS —
 * addCollections rejects and the Scan section renders "Scanner failed to
 * start". Miss (2) and RxDB refuses the collection for a missing strategy.
 *
 * 🛑 `autoMigrate: false` is NOT the shortcut. It creates the v1 collection
 * happily and leaves every v0 document stranded in the old storage instance,
 * invisible to the push replica — silently dropped redemptions, which is the
 * exact harm this activity exists to prevent.
 *
 * The strategy itself is total and lossless: `policy_unresolved` did not exist
 * when these rows were written, so the honest value is `false` — those
 * attempts were recorded under the pre-B-432 policy path, where the device
 * always believed it had an answer. Returning `null` (RxDB's "drop this
 * document") is never correct here for the same reason autoMigrate:false is
 * not: a queued attempt is evidence, not cache.
 */
export const SCAN_ATTEMPTS_MIGRATION_STRATEGIES = {
  1: (oldDoc) => ({ ...oldDoc, policy_unresolved: false }),
};

/** addCollections() argument — Card 2's marketingCollectionSpec() pattern. */
export function scanAttemptsCollectionSpec() {
  return {
    [SCAN_ATTEMPTS_COLLECTION]: {
      schema: SCAN_ATTEMPTS_SCHEMA,
      migrationStrategies: SCAN_ATTEMPTS_MIGRATION_STRATEGIES,
    },
  };
}

/**
 * Record a scan — the scanner's (Card 5's) entry point. Insert-only and
 * offline-safe: no network is touched here; the push replica drains the queue
 * whenever connectivity exists.
 *
 * Dedupe rule (local persistence shape, this card's call): at most one LIVE
 * (`pending` or `accepted`) attempt per code per device. A repeat scan of the
 * same code answers the existing doc with `deduped: true` — which is also
 * load-bearing for GAP-1 belt 2: an `already_used` naming our own device can
 * only mean THIS attempt's lost burn, never a second local scan racing our own
 * earlier win. A `rejected` attempt does NOT block a fresh scan (the crew gets
 * a fresh verdict row).
 *
 * @param {object} attemptsCollection  the SCAN_ATTEMPTS_COLLECTION RxCollection
 * @param {object} fields  {code_id, device_id, offline_override?, override_by?,
 *                          unverified_code?, policy_unresolved?,
 *                          pos_order_number?, redeemed_value?}
 * @param {{now?: function, generateId?: function}} [opts]  clock/uuid injection
 * @returns {Promise<{doc: object, deduped: boolean}>}
 */
export async function enqueueAttempt(attemptsCollection, {
  code_id, device_id,
  offline_override = false, override_by = null,
  unverified_code = false, pos_order_number = null, redeemed_value = null,
  // Card refusal-holds-before-sync, ADDITIVE-OPTIONAL (the pos_business_date
  // precedent): the B-432 discriminator. 🛑 This destructure IS a whitelist —
  // spike 03 measured the field being silently dropped here when it was not
  // named. A caller passing it and a stored row lacking it is the failure mode
  // this line exists to close.
  policy_unresolved = false,
  // Card 6 (redemption-submit-flow), ADDITIVE-OPTIONAL: the §13 business date
  // computed from the Toast-cutoff constant (submit-support.js). Absent, the
  // original UTC-date-of-scanned_at behavior is byte-identical — existing
  // callers and harnesses unchanged.
  pos_business_date = null,
}, { now = Date.now, generateId = () => globalThis.crypto.randomUUID() } = {}) {
  const existing = await attemptsCollection.find({ selector: { code_id } }).exec();
  const live = existing.find((d) => d.status !== 'rejected');
  if (live) return { doc: live, deduped: true };
  const scannedIso = new Date(now()).toISOString();
  const doc = await attemptsCollection.insert({
    id: generateId(),
    code_id,
    device_id,
    scanned_at: scannedIso,
    status: 'pending',
    reason: null,
    offline_override,
    override_by,
    unverified_code,
    policy_unresolved,
    pos_order_number,
    pos_business_date: pos_business_date || scannedIso.slice(0, 10),
    redeemed_value,
    burn_ok: null,
    burn_reason: null,
    landed: false,
    winner_device: null,
    winner_at: null,
  });
  return { doc, deduped: false };
}

/**
 * Make the replicateRxCollection push.handler — the §6 shape with GAP-1's two
 * belts. Enumerable: pass `requestLog` and every outbound request is recorded
 * {kind, attempt_id, code_id, url} BEFORE it is sent (B-216).
 *
 * Retry mechanics (this card's call): the handler THROWS on any unresolvable
 * step — redeem HTTP failure, landing network failure, winner unresolvable —
 * and RxDB's own push retry (`retryTime` on startScanAttemptsReplica) re-runs
 * it; the local row stays honestly `pending`. It never guesses: flipping a row
 * on a guess is GAP-1's harm. Landing idempotency: HTTP 409 on our own uuid
 * means a previous landing succeeded and its response was lost — treated as
 * landed.
 *
 * @param {object} cfg
 * @param {string} cfg.restUrl            PostgREST origin (no trailing slash)
 * @param {string|function} cfg.bearer    device JWT (or getter)
 * @param {string} cfg.deviceId           this device — MUST match the JWT sub
 *                                        (RLS with-check) and is what belt 2
 *                                        compares codes.redeemed_by against
 * @param {function} cfg.fetchImpl
 * @param {object} cfg.attemptsCollection the local scan_attempts RxCollection
 * @param {object} cfg.codesCollection    Card 2's codes-replica RxCollection —
 *                                        the ONLY winner-data source
 * @param {Array}  [cfg.requestLog]
 * @param {number} [cfg.winnerWaitMs=10000]  bounded await for the codes replica
 *                                           to name a winner on already_used
 * @param {number} [cfg.winnerPollMs=100]
 * @param {function} [cfg.now]
 */
export function makePushHandler({
  restUrl, bearer, deviceId, fetchImpl,
  attemptsCollection, codesCollection,
  requestLog, winnerWaitMs = 10_000, winnerPollMs = 100, now = Date.now,
}) {
  const authHeaders = () => ({
    Authorization: `Bearer ${typeof bearer === 'function' ? bearer() : bearer}`,
    'Content-Type': 'application/json',
  });

  // Await the LOCAL codes replica naming who burned the code. Bounded; null on
  // timeout. Never a server read of scan_attempts (write-only), and not a
  // server read of codes either — the replica IS the device's view (§5.2).
  async function awaitWinner(codeId) {
    const deadline = now() + winnerWaitMs;
    for (;;) {
      const c = await codesCollection.findOne(codeId).exec();
      if (c && c.redeemed_by) return { winner_device: c.redeemed_by, winner_at: c.redeemed_at ?? null };
      if (now() >= deadline) return null;
      await new Promise((r) => setTimeout(r, winnerPollMs));
    }
  }

  return async function pushHandler(rows) {
    for (const row of rows) {
      const state = row.newDocumentState;
      if (state.status !== 'pending') continue; // idempotence: resolved rows re-push as no-ops
      const doc = await attemptsCollection.findOne(state.id).exec();
      if (!doc || doc.status !== 'pending') continue;

      // ── 0. the F-2 guard, BEFORE the burn (card requires-online-replication,
      // §19 F2 / §9; spike-measured): an unverified attempt names NO code —
      // its local code_id IS the scanned token_hash (64 hex, submit-flow's
      // recorded call), and feeding that to /rpc/redeem draws a deterministic
      // HTTP 400 (22P02 on p_code uuid) that head-of-line poisons the whole
      // queue (12 redeem attempts, 0 landings, every later redemption
      // stranded). There is nothing to burn, so it skips redeem() entirely
      // and lands directly on the distinct path: code_id NULL + token_hash +
      // the audit flags, status 'accepted' (§9's taxonomy — offline overrides
      // are the accepted attempts reconciled FIRST; no new terminal status).
      // Skip-until-arbitration was run and REJECTED (it strands the audit
      // row on-device). Server-side arbitration of the hash is §19 F2's
      // "when sync arbitrates" clause — Activity D's surface, after landing.
      if (doc.unverified_code) {
        const landUrl = `${restUrl}/scan_attempts`;
        if (requestLog) requestLog.push({ kind: 'land-unverified', attempt_id: doc.id, code_id: doc.code_id, url: landUrl });
        const land = await fetchImpl(landUrl, {
          method: 'POST',
          headers: { ...authHeaders(), Prefer: 'return=minimal' },
          body: JSON.stringify({
            id: doc.id,
            code_id: null,                 // it names no code — that is the truth
            token_hash: doc.code_id,       // what it actually has
            device_id: deviceId,
            scanned_at: doc.scanned_at,
            status: 'accepted',
            reason: null,
            offline_override: doc.offline_override,
            override_by: doc.override_by ?? null,
            unverified_code: true,
            // The B-432 discriminator (card refusal-holds-before-sync). THIS
            // is the row the done_when's second half is about: paired with
            // unverified_code it separates a campaigns-replica FAILURE
            // override (t,t) from a genuinely-unknown-campaign override (t,f),
            // both landing status='accepted'. 🛑 Only send this once
            // migration 20260906000300 is applied — pre-migration PostgREST
            // answers HTTP 400 PGRST204 and this handler THROWS, which is the
            // F-2 head-of-line poison class (spike 03).
            policy_unresolved: !!doc.policy_unresolved,
            pos_order_number: doc.pos_order_number ?? null,
            pos_business_date: doc.pos_business_date,
            redeemed_value: doc.redeemed_value ?? null,
          }),
        });
        // 409 = duplicate key on our own uuid: a previous landing succeeded
        // and its response was lost. Landed (same idempotency rule as below).
        if (land.status !== 201 && land.status !== 409) {
          throw new Error(`[marketing-sync] unverified scan_attempts insert answered HTTP ${land.status}`);
        }
        await doc.incrementalPatch({ status: 'accepted', reason: null, landed: true });
        continue;
      }

      // ── 1. burn — skipped when an outcome is already persisted (belt 1) ──
      let burnOk = doc.burn_ok;
      let burnReason = doc.burn_reason;
      let winner = doc.winner_device ? { winner_device: doc.winner_device, winner_at: doc.winner_at } : null;
      if (burnOk === null || burnOk === undefined) {
        const url = `${restUrl}/rpc/redeem`;
        if (requestLog) requestLog.push({ kind: 'redeem', attempt_id: doc.id, code_id: doc.code_id, url });
        const res = await fetchImpl(url, {
          method: 'POST',
          headers: authHeaders(),
          body: JSON.stringify({ p_code: doc.code_id, p_device: deviceId }),
        });
        if (res.status !== 200) throw new Error(`[marketing-sync] redeem answered HTTP ${res.status}`);
        const body = await res.json();
        const verdict = Array.isArray(body) ? body[0] : body;
        burnOk = !!verdict.ok;
        burnReason = verdict.ok ? null : verdict.reason ?? null;

        if (!burnOk && burnReason === 'already_used') {
          const w = await awaitWinner(doc.code_id);
          if (w === null) {
            // Cannot arbitrate own-win vs lost-race yet — retry, never guess.
            throw new Error(`[marketing-sync] already_used but the codes replica names no winner for ${doc.code_id} yet — retrying`);
          }
          if (w.winner_device === deviceId) {
            // Belt 2: the echo of our OWN burn (lost response / died before
            // the burn patch). This attempt WON.
            burnOk = true;
            burnReason = null;
          } else {
            winner = w; // the genuine lost race — render data for the flip
          }
        }
        // Belt 1: persist BEFORE the landing insert. A landing failure below
        // retries the landing only; redeem() never re-runs for this attempt.
        await doc.incrementalPatch({
          burn_ok: burnOk, burn_reason: burnReason,
          winner_device: winner?.winner_device ?? null,
          winner_at: winner?.winner_at ?? null,
        });
      }

      // ── 2. land the attempt row (the server row carries §4 fields only) ──
      const status = burnOk ? 'accepted' : 'rejected';
      const reason = burnOk ? null : burnReason;
      const landUrl = `${restUrl}/scan_attempts`;
      if (requestLog) requestLog.push({ kind: 'land', attempt_id: doc.id, code_id: doc.code_id, url: landUrl });
      const land = await fetchImpl(landUrl, {
        method: 'POST',
        headers: { ...authHeaders(), Prefer: 'return=minimal' },
        body: JSON.stringify({
          id: doc.id,
          code_id: doc.code_id,
          device_id: deviceId,
          scanned_at: doc.scanned_at,
          status,
          reason,
          offline_override: doc.offline_override,
          override_by: doc.override_by ?? null,
          unverified_code: doc.unverified_code,
          // Carried here too, deliberately. Under the fail-closed predicate a
          // KNOWN code's override implies its campaign resolved, so this is
          // `false` on every path that exists today — except the one B-436
          // names (the policy source failing to CONSTRUCT at all, where
          // submit-flow still coerces to false and the override survives).
          // Landing a constant costs nothing; landing a lie about an
          // unresolved policy is how B-432 stayed invisible for a run.
          policy_unresolved: !!doc.policy_unresolved,
          pos_order_number: doc.pos_order_number ?? null,
          pos_business_date: doc.pos_business_date,
          redeemed_value: doc.redeemed_value ?? null,
        }),
      });
      // 409 = duplicate key on our own uuid: a previous landing succeeded and
      // its response was lost. Landed.
      if (land.status !== 201 && land.status !== 409) {
        throw new Error(`[marketing-sync] scan_attempts insert answered HTTP ${land.status}`);
      }

      // ── 3. resolve the local row — the UI flip the scanner observes ──
      await doc.incrementalPatch({ status, reason, landed: true });
    }
    return []; // no conflicts — the master accepted what we told it
  };
}

/**
 * Start the live, push-only scan_attempts replica. Thin on purpose: everything
 * that varies between the browser and the harness is a parameter.
 *
 * @param {object} p
 * @param {function} p.replicateRxCollection  from the vendored bundle (browser)
 *                                            or the QA rxdb (harness)
 * @param {object} p.collection               the scan_attempts RxCollection
 * @param {function} p.pushHandler            from makePushHandler
 * @param {string} [p.replicationIdentifier]
 * @param {number} [p.batchSize=10]
 * @param {boolean} [p.waitForLeadership=false]
 * @param {number} [p.retryTime=5000]         RxDB's retry pause after a thrown
 *                                            handler — the drain cadence for
 *                                            transient failures
 */
export function startScanAttemptsReplica({
  replicateRxCollection, collection, pushHandler,
  replicationIdentifier = 'marketing-scan-attempts-push',
  batchSize = 10, waitForLeadership = false, retryTime = 5_000,
}) {
  return replicateRxCollection({
    collection,
    replicationIdentifier,
    live: true,
    waitForLeadership,
    retryTime,
    push: { handler: pushHandler, batchSize },
  });
}
