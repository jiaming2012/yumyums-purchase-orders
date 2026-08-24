// sync-rxdb/bootstrap.js — the ONE module an HQ page loads.
//
// Card `sync-rxdb-replication-and-conflict-handler` (overnight-20260801, C1).
//
// ===========================================================================
// 🛑 IMPORT + CONSTRUCTION ONLY. NO WRITE PATH IS SWAPPED HERE.
// ===========================================================================
// This card's footprint in `workflows.html` is one `<script type="module">`
// tag. `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops` →
// `DRAFT_RESPONSES` → `hydrateFieldState` is untouched and stays the live
// persistence path; the swap belongs to `sync-hard-cutover`.
//
// (This comment named `autoSaveField` → `/saveResponse` until B-65. Both halves
// were false: no such function is defined anywhere in the tree, and no frontend
// code posts to /saveResponse. Corrected by card A2, run 20260804.)
//
// 🛑 THE SENTENCE THAT USED TO FOLLOW HERE IS NO LONGER TRUE, AND IS CORRECTED
// RATHER THAN DELETED. It read: "It also does NOT create the RxDB database and
// does NOT start replication, and both omissions are deliberate rather than
// unfinished." That was accurate from 2026-08-01 until card
// `skeleton-one-row-end-to-end` (run 20260808-2, C2), which is the FIRST
// production call site of `createHQSyncDatabase()` and `startHQReplication()`
// in this repo's history. It is now:
//
//   THIS MODULE CREATES THE DATABASE AND STARTS REPLICATION — BUT ONLY FROM
//   `openSyncScope()`, AND ONLY WITH THE `hq_sync_read` FLAG EXPLICITLY ON.
//   THE FLAG IS OFF BY DEFAULT IN EVERY ENVIRONMENT.
//
// The reasons the omission was right are still the reasons the FLAG is off by
// default, so they are kept:
//
//   * `HQ_SYNC_REST_URL` / `HQ_SYNC_REALTIME_URL` are unset in every
//     environment, so the door answers 503 to everything. Starting replication
//     unconditionally would put a permanent retry loop and a stream of console
//     errors on every crew phone, in exchange for nothing.
//     🛑 This used to call the unset variables "still an armed precondition"
//     gated on the row-visibility-RLS card landing. That card MERGED on
//     2026-08-01 (`bbbfc64`; the roadmap flipped it DONE at `914536c`), so the
//     gate named here had been satisfied for five days while still reading as
//     open — and while hiding the precondition that IS open: nothing calls
//     `createHQSyncDatabase()` or `startHQReplication()` yet. That was
//     `sync-hard-cutover`'s job; the milestone reshaped it into Activity 3's
//     walking skeleton, and C2 is where the call finally lands.
//     Corrected on card `repo-hygiene-preconditions`, run 20260806; the last
//     clause corrected again on `skeleton-one-row-end-to-end`, run 20260808-2.
//   * `createHQSyncDatabase()` writes IndexedDB. Creating local state before
//     anything reads it means a schema decision is on every phone before the
//     card that depends on it has been reviewed. The flag is what keeps that
//     true for every crew device while the skeleton is driven in dev.
//
// 🛑 B-64 IS FIXED (card `list-views-decision-recording`, S1, run 20260808-2).
// The scope banner on `startReplication` below now names both scopes, states
// what's live/REST/open today, and carries the "cancel before re-scoping THE
// SAME shape" restatement (B-63/B-64, ledger T-43(c)). Kept as a note rather
// than deleted so a future reader sees why C2's own diff did not fix it: only
// the two sentences C2's diff falsified were C2's to correct; the rest was
// S1's, deliberately sequenced AFTER the lifecycle it documents so it could
// state what is actually live rather than predict it.
//
// What IS done here is the adoption decision 59 was waiting for: a production
// page finally IMPORTS `vendor/rxdb.bundle.js`, so the bundle is now a real
// runtime dependency of `workflows.html` — which is what earns it the
// `globPatterns` precache entry re-added in `build-sw.js`, and which is why
// `backend/Dockerfile` must copy `vendor/` into the image in the same change
// set. A precached URL that 404s fails the ENTIRE service-worker install.
//
// ===========================================================================
// FAILING SOFT IS THE POINT.
// ===========================================================================
// Nothing on the page depends on this module yet. A page whose checklists stop
// working because a not-yet-used sync layer threw would be a strictly worse
// outcome than no sync layer at all, so construction is wrapped and a failure
// is recorded on `window.HQSync.error` rather than thrown. When the cutover
// card makes the page DEPEND on this, that card must revisit the softness —
// noted here so it is a decision then rather than an inheritance.

import {
  createHQSyncClient,
  createHQSyncDatabase,
  startHQReplication,
  normalizeScope,
  expandGrantSlugs,
  describeConflict,
  assertVendorPin,
  PINNED_VENDOR,
} from './client.js';
import { createHQConflictHandler, resolveConflict } from './conflict-handler.js';

// The launcher's cached grant list (`index.html` writes it). The claim is
// NARROW — obligation 4 / decision 56: a user holding `inventory` genuinely
// reaches `inventory-trends` and `inventory-cost`, so the reachable set is the
// expansion, not the claim.
//
// Mirrors `index.html:150-179`'s `readIdentityToken()` — the identity token
// lives in the origin-scoped `hq-identity` CacheStorage bucket (`/__hq_identity`),
// not localStorage, and CacheStorage is reachable from any page on this
// origin, not just the launcher that writes it.
async function readIdentityToken() {
  if (!('caches' in window)) return null;
  try {
    const c = await caches.open('hq-identity');
    const r = await c.match('/__hq_identity');
    if (!r) return null;
    const t = (await r.text()).trim();
    return t || null;
  } catch (err) {
    return null;
  }
}

// B-89: `index.html` (since decision 112, T-30) writes `hq_apps` as the
// identity-stamped envelope `{uid, apps}` — index.html:228-241 — never a bare
// array. This used to `Array.isArray`-gate the raw parse directly, which
// rejects that envelope on every real client and silently returns `[]`
// always. Fixed to read the SAME shape `index.html:234-236`'s
// `readCachedApps()` reads: reject a missing envelope, reject a bare array,
// reject a non-array `apps`, and reject a `uid` that does not match the
// identity token this device last verified.
//
// The uid-mismatch case is NOT a park: `index.html:234-236` already ships the
// answer (mirrored here) — a mismatch is treated as "nothing cached," not a
// thrown/surfaced error. This function's own established "nothing cached"
// value is `[]` (already returned today for a missing key or a parse
// exception), so an invalid/mismatched envelope resolves to `[]` too, not
// `index.html`'s `null` — the return type here has always been array-shaped
// for `expandGrantSlugs()`.
async function cachedGrantSlugs() {
  try {
    const raw = localStorage.getItem('hq_apps');
    if (!raw) return [];
    const env = JSON.parse(raw);
    const deviceId = await readIdentityToken();
    if (!deviceId || !env || Array.isArray(env) || env.uid !== deviceId || !Array.isArray(env.apps)) {
      return [];
    }
    return env.apps.map((a) => a && a.slug).filter(Boolean);
  } catch (err) {
    return [];
  }
}

// ===========================================================================
// THE FLAG — `hq_sync_read`. Card `skeleton-one-row-end-to-end`, run 20260808-2.
// ===========================================================================
//
// Before this card "the sync flag" named NOTHING in this repo: no identifier, no
// mechanism, no default existed in the tree while three planning documents
// referred to it (G6 finding F-3, run 20260808-2). This is the definition.
//
//   NAME      `hq_sync_read` — the constant below, and nowhere else.
//   STORAGE   localStorage, value EXACTLY the string 'on'.
//   OVERRIDE  `?hq_sync_read=on` in the URL turns it on and PERSISTS it;
//             `?hq_sync_read=off` clears it. A crew phone has no devtools, so a
//             flag only settable from a console is not a flag anyone can use.
//   DEFAULT   🛑 OFF. Key absent ⇒ off. Any value that is not the literal 'on'
//             ⇒ off. There is no environment, build or deploy in which this
//             defaults on. `HQ_SYNC_REST_URL` is still unset everywhere, so the
//             /sync door still answers 503, so an accidental ON costs a retry
//             loop rather than data — but the default is OFF on purpose and not
//             because the door happens to be shut.
//
// Resolved ONCE, synchronously, at module load — before anything can reach
// `openSyncScope()`. Deliberately not re-read per call: a flag that can change
// under a live replication is a lifecycle question nobody has been asked.
export const SYNC_READ_FLAG = 'hq_sync_read';

export function resolveSyncReadFlag(loc, store) {
  const location_ = loc || (typeof location !== 'undefined' ? location : null);
  let store_ = store;
  if (store_ === undefined) {
    try { store_ = typeof localStorage !== 'undefined' ? localStorage : null; } catch (err) { store_ = null; }
  }
  // The URL override is applied first so it can persist, then the answer is
  // read back from the store — one source of truth, whichever way it was set.
  try {
    if (location_ && location_.search) {
      const q = new URLSearchParams(location_.search).get(SYNC_READ_FLAG);
      if (q === 'on' && store_) store_.setItem(SYNC_READ_FLAG, 'on');
      else if (q !== null && store_) store_.removeItem(SYNC_READ_FLAG);
    }
  } catch (err) { /* a hostile URL or a blocked store must not break the page */ }
  try {
    return !!store_ && store_.getItem(SYNC_READ_FLAG) === 'on';
  } catch (err) {
    return false;
  }
}

// ===========================================================================
// THE CALL SITE — the first in this repo's history.
// ===========================================================================
//
// 🛑 DECISION 126 (ledger T-32, 2026-08-02), VERBATIM AND CITED:
//
//     "The cutover splits reads from writes. RxDB serves reads; HQ's REST path
//      keeps owning writes."
//
//    `POST /api/v1/workflow/saveResponse` and `POST /submitChecklist` keep
//    owning ALL writes. This module starts a PULL-and-PUSH replication because
//    that is the plugin's shape, but nothing on any HQ page writes into these
//    collections — `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops`
//    is byte-untouched and remains the one write channel (D-08; there is no
//    `autoSaveField`, B-65). This card changes READS ONLY. A build card may not
//    propose that split; it carries it.
//
// 🛑 DECISION 105 (ledger T-29) + preference architecture/C-2: replication is
//    scoped — per-open-checklist for the fill collections, per-user-with-a-date-
//    floor for the two list collections — and is NEVER pulled whole. There is no
//    unscoped path here: `normalizeScope` throws, synchronously, before a
//    database is touched. Widening the scope needs a recorded decision, and this
//    card widens nothing.
//
// 🛑 SPIKE E'S CONDITION (ledger T-42, run 20260808), VERBATIM:
//
//     Trusted checkpoint catch-up with NO explicit resync step is valid ONLY
//     while the relay stays trigger/NOTIFY-driven (spike C's mechanism). A
//     POLLING relay on a business watermark reintroduces the missed-UPDATE
//     hazard exactly — `submitted_at` never advances after INSERT, so a poller
//     watching it sleeps through every update to a row it already holds.
//
//    This call site contemplates NO POLLING. It starts no interval, watches no
//    business watermark, and adds no explicit resync step — because the
//    condition holds: the checkpoint pulls on the substrate's trigger-stamped
//    `_modified` (strict `gt` plus an id tie-breaker), and the relay is spike
//    C's LISTEN/NOTIFY one. 🛑 IF THE RELAY EVER BECOMES A POLLER, AN EXPLICIT
//    RESYNC STEP COMES BACK AS A REQUIREMENT OF THIS CALL SITE. Whoever changes
//    the relay owns that; it is written here so they meet it.
//
// ---------------------------------------------------------------------------
// SHAPE, and why it is this shape (ledger T-43(c), the operator's own words):
// crew members work MULTIPLE CHECKLISTS CONCURRENTLY — a setup checklist and a
// food-preparation checklist at the same time. So multiple live per-checklist
// replications at once ARE the design, not an edge case:
//
//   * ONE database, shared. `ensureDatabase()` memoises the PROMISE, not the
//     resolved value, so two concurrent first calls cannot race two
//     `createRxDatabase()` calls onto one IndexedDB name.
//   * ONE registry entry per SCOPE. Different scopes run side by side; the same
//     scope twice returns the SAME handle rather than starting a second
//     replication on one Realtime topic. That is the mechanical half of
//     "cancel before re-scoping THE SAME shape" (B-63/B-64) — re-opening a shape
//     you already hold is a no-op instead of a leak.
//   * `handle.cancel()` cancels that scope's states and drops that entry. Other
//     scopes keep running.
//
// `replicationIdentifier` already carries the scope fingerprint (client.js, G6
// F-1), so per-scope checkpoints are independent by construction — two open
// checklists cannot filter each other's rows away.
//
// 🛑 AND SINCE C3 (`activate-fill-view-reads`) THE FINGERPRINT CARRIES THE CREW
// MEMBER — C2's G6 finding F-2. Two open checklists could not filter each
// other's rows away; two crew members on ONE truck phone could, because the fill
// scope had no user dimension anywhere and its emitted filters name no user
// either. `normalizeScope` now REQUIRES `userId` on the fill shape, exactly as
// it has on the list shape since S1a, and `scopeKey()` below carries it too so
// the registry and the identifier cannot disagree.
//
// 🛑 A FAILED OPEN IS EVICTED FROM BOTH MEMOS (F-1). `databasePromise` and the
// `openScopes` entry are cleared when they reject, so a transient IndexedDB
// failure costs a retry rather than the page's whole sync layer. The consumer
// that made this matter is C3's fill view: the dev skeleton opened one scope
// once, a crew member opens one per checklist all shift.
// ---------------------------------------------------------------------------

const openScopes = new Map();
let databasePromise = null;

// A stable key for a NORMALISED scope. Not `scopeFingerprint` — that hashes ONE
// collection's filter; this identifies the whole scope, and being readable in a
// debugger is worth more here than being short.
//
// 🛑 THE DELIMITER IS THE ESCAPE SEQUENCE `\0`, NOT A RAW NUL BYTE, AND THAT IS
// B-70. A literal U+0000 in a source file under `sync-rxdb/` puts GNU grep into
// binary mode on the whole file: `grep -n 'export' sync-rxdb/client.js` printed
// nothing and exited 1 on a file with 29 matches, which makes every
// `done_when: "grep returns nothing"` criterion unreliable IN THE PASSING
// DIRECTION. `client.js`'s `scopeFingerprint` carried exactly that byte and was
// fixed the same way. The escape is the SAME byte at runtime, so keys are
// unchanged; the file stays 7-bit clean. Guarded by tests/repo-hygiene.spec.js
// ('no source file under sync-rxdb/ contains a NUL byte'), which caught this
// card writing raw NULs here on its first full-suite leg. Do not "tidy" it back.
// 🛑 `s.userId` IS IN THE FILL BRANCH SINCE C3 (`activate-fill-view-reads`, run
// 20260808-2) — C2's G6 finding F-2. `normalizeScope` now requires it, and the
// registry key must agree with the replication identifier: if two crew members
// on one truck phone opened the same checklist and collapsed to ONE registry
// entry here, the second would be handed the first's handle and never start
// their own replication — the same data loss the identifier fix prevents, one
// layer up, and invisible because `openScopeKeys()` would report one live scope
// and be telling the truth about its own map.
function scopeKey(scope) {
  const s = normalizeScope(scope);
  return s.mode === 'list'
    ? ['list', s.userId, s.since, s.templateIds.slice().sort().join('+')].join('\0')
    : ['fill', s.userId, s.checklistId, s.templateId, s.fieldIds.slice().sort().join('+')].join('\0');
}

function ensureDatabase() {
  if (!databasePromise) {
    // No `storage` override: `createHQSyncDatabase()` uses `getRxStorageDexie()`.
    // 🛑 NEVER a memory storage on any offline-capable path — C1's B-88 guard
    // detects a live RxDB instance by scanning `indexedDB.databases()`, and a
    // memory-backed instance would be invisible to it (G6 finding F-2).
    //
    // 🛑 `HQSync.createDatabase`, THE PROPERTY, NOT THE IMPORTED BINDING — and
    // that is a testability seam of the same kind as `startHQReplication`'s
    // `opts.replicate`, not an indirection for its own sake. C2's G6 could not
    // force F-1 at runtime (undefining `window.indexedDB` resolved anyway, since
    // Dexie holds its own reference), which is why the finding shipped PLAUSIBLE
    // rather than CONFIRMED — an unforceable failure is an untestable fix.
    // Production never replaces this property; the default IS
    // `createHQSyncDatabase`, assigned in the object literal below.
    const pending = HQSync.createDatabase().then((db) => {
      HQSync.db = db;
      return db;
    });
    // 🛑 EVICT ON REJECTION — C2's G6 finding F-1, and this is the whole fix.
    // A memoised promise that REJECTED is permanent for the page's lifetime:
    // every later `openSyncScope()` gets the same stale rejection back, so one
    // transient IndexedDB failure (quota exhausted on a crew phone, a corrupt
    // Dexie store) bricked the fill view's reads until a reload. Clearing the
    // slot makes the next call try again. The `catch` handler is attached to a
    // DERIVED promise, so it marks nothing as handled for the caller — the
    // rejection still propagates to whoever awaited `ensureDatabase()`.
    // The identity check is what makes it safe under concurrency: if a later
    // call has already installed a fresh promise, this one must not null it.
    pending.catch(() => {
      if (databasePromise === pending) databasePromise = null;
    });
    databasePromise = pending;
  }
  return databasePromise;
}

/**
 * Open — or re-use — a live replication for ONE scope.
 *
 * 🛑 REFUSES SYNCHRONOUSLY when the flag is off: it THROWS before returning a
 * promise, before `createHQSyncDatabase` is referenced, before any `await`. That
 * is stronger than an async rejection on purpose. C1's B-88 guard samples
 * `window.HQSync.db` at end of load and is timing-blind to a database creation
 * that resolves later (G6 finding F-1), so the flag-off path must never BEGIN
 * async database creation — not merely fail to finish it. There is no path in
 * this tree on which creation starts and is then abandoned.
 *
 * 🛑 A REJECTED OPEN IS EVICTED, not memoised (C2's G6 finding F-1). Both the
 * shared database promise and this registry entry are dropped when the open
 * fails, so a retry can succeed, `openScopeKeys()` never reports a dead scope as
 * live, and the caller is never handed a handle whose `cancel()` is unreachable.
 *
 * @param {object} scope FILL `{userId, checklistId, templateId, fieldIds}` or
 *   LIST `{mode:'list', userId, since, templateIds}`. Validated by
 *   `normalizeScope` — 🛑 `userId` is REQUIRED on BOTH shapes since C3 (F-2).
 * @returns {Promise<{key, scope, db, states, cancel}>}
 */
function openSyncScope(scope) {
  if (!HQSync.readEnabled) {
    throw new Error(
      `[hq-sync] refusing to open a replication scope: the ${SYNC_READ_FLAG} flag is off. `
      + `It is OFF by default in every environment; set localStorage['${SYNC_READ_FLAG}'] = 'on' `
      + `or load the page with ?${SYNC_READ_FLAG}=on. Nothing may read from RxDB on a code path `
      + 'that can execute offline while the flag is off (B-88).',
    );
  }
  if (!HQSync.client) {
    throw new Error(
      '[hq-sync] refusing to open a replication scope: the client failed to construct. '
      + 'See window.HQSync.error.',
    );
  }
  // Throws on an absent or malformed scope — synchronously, before any database
  // work, so a refused scope costs nothing (decision 105).
  const key = scopeKey(scope);
  const existing = openScopes.get(key);
  if (existing) return existing;

  const handle = (async () => {
    const db = await ensureDatabase();
    const states = startHQReplication(db, HQSync.client, { scope });
    return {
      key,
      scope: normalizeScope(scope),
      db,
      states,
      async cancel() {
        openScopes.delete(key);
        await Promise.all(Object.values(states).map((s) => s.cancel()));
      },
    };
  })();
  // Registered BEFORE the first await resolves, so a second synchronous call for
  // the same scope joins this one rather than starting a rival replication.
  openScopes.set(key, handle);
  // 🛑 ...AND DE-REGISTERED IF IT REJECTS — C2's G6 finding F-1, the second half.
  // Without this the map holds a promise that will only ever reject: the scope is
  // reported live by `openScopeKeys()`, its `cancel()` can never be reached (the
  // handle it lives on was never produced), and every retry of the same scope
  // gets the original failure back forever. The `catch` is on a DERIVED promise
  // so the rejection still reaches the caller, and the identity check means a
  // retry that has already re-registered is not evicted by the loser.
  handle.catch(() => {
    if (openScopes.get(key) === handle) openScopes.delete(key);
  });
  return handle;
}

const HQSync = {
  ready: false,
  error: null,
  // Constructed eagerly (pure, no network).
  client: null,
  conflictHandler: null,
  surfaces: [],
  vendor: null,
  pinned: PINNED_VENDOR,
  // 🛑 NO LONGER MERELY "deferred, for `sync-hard-cutover` and for C2's tests to
  // drive" — that sentence was true until C3 and this diff falsified it, so it is
  // corrected rather than deleted. `ensureDatabase()` CALLS THIS PROPERTY. It is
  // the default and the only implementation; a test replaces it to force the
  // storage failure C2's G6 could not force (finding F-1), and restores it.
  createDatabase: createHQSyncDatabase,
  // 🛑 REQUIRES a scope — TWO SHAPES, validated by `normalizeScope`. The shape
  // of record is `sync-rxdb/client.js`'s `startHQReplication` docblock and its
  // REPLICATION SCOPE design block above it; not restated field-by-field here
  // so there is ONE copy to go stale rather than two (B-64's own lesson):
  //
  //   FILL  `{userId, checklistId, templateId, fieldIds}` — one open checklist.
  //   LIST  `{mode:'list', userId, since, templateIds}` — a list view, per-user
  //         with a date floor (ledger T-29 decision 105, amended 2026-08-02).
  //
  // `startReplication` throws without a scope, synchronously; a default would
  // widen decision 105 silently.
  //
  // 🛑 WHAT IS LIVE TODAY (ledger T-43, run 20260808-2):
  //   * C2's one-row dev surface (`#sync-one-row` in `workflows.html`), behind
  //     the `hq_sync_read` flag — OFF by default in every environment.
  //   * C3's checklist FILL view (ALSO behind the `hq_sync_read` flag, same
  //     gate as above): one replication PER OPEN CHECKLIST, many live
  //     at once — a setup checklist and a food-prep checklist worked
  //     concurrently is a recorded product requirement (T-43(c)) — each
  //     cancelled on close (`workflows.html`'s `HQFillSync`).
  //
  // 🛑 WHAT STAYS REST, TODAY, REGARDLESS OF THE FLAG:
  //   * BOTH list views. Neither `renderMyChecklists()` nor the Approvals tab
  //     is served by this module; both still render from a fetch.
  //   * Approvals, specifically, BY RULING — ledger T-43(a): the Approvals tab
  //     stays on re-fetch, a partial resolution of B-43, recorded rather than
  //     silently carried.
  //
  // 🛑 WHAT IS OPEN — T-43(b), THE OPERATOR'S RULING, NOT A GAP THIS COMMENT
  // FILLS: the My Checklists read path. The operator explicitly declined to
  // rule it this cycle ("Approvals only; keep My Checklists open") — no card
  // may decide it or predict an outcome; a card that cannot proceed without
  // the answer parks. (Card `list-views-decision-recording`, S1, run
  // 20260808-2 — this banner's own card — did not need to decide it to write
  // this.)
  //
  // 🛑 CANCEL BEFORE RE-SCOPING THE SAME SHAPE (B-63/B-64's corrected wording,
  // ledger T-43(c) — replaces the earlier "cancel before re-scoping", full
  // stop, which under two-plus concurrent shapes read as "opening a second
  // checklist must cancel the first" and is exactly what T-43(c) overturns).
  // `openSyncScope()` makes re-opening a shape you already hold a no-op (the
  // registry returns the existing handle); the rule is about a caller that
  // bypasses the registry or forgets `.cancel()` for the SAME shape, not about
  // holding several DIFFERENT shapes at once — that is the design, not a leak.
  startReplication: startHQReplication,

  // ── The walking skeleton (card `skeleton-one-row-end-to-end`, C2). ────────
  // The flag's name, so a page never spells it a second time.
  SYNC_READ_FLAG,
  // Resolved ONCE, here, at module load. FALSE by default in every environment.
  readEnabled: resolveSyncReadFlag(),
  // 🛑 STAYS `undefined` UNLESS `openSyncScope()` created it — which requires the
  // flag. This is the exact property C1's B-88 guard asserts is undefined at end
  // of load; it is assigned in exactly one place (`ensureDatabase()`).
  db: undefined,
  // The call site. Refuses SYNCHRONOUSLY with the flag off; see its docblock.
  openSyncScope,
  // Inspection: which scopes are live right now. Multiple concurrent
  // per-checklist scopes are the design (ledger T-43(c)).
  openScopeKeys: () => [...openScopes.keys()],

  // Pure helpers C2 renders against.
  describeConflict,
  resolveConflict,
  createConflictHandler: createHQConflictHandler,
  expandGrantSlugs,
  assertVendorPin,
};

try {
  // Top-level await — this module is loaded `type="module"` (workflows.html),
  // which is the one thing that makes awaiting cachedGrantSlugs()'s CacheStorage
  // read possible at the module's own top level, no wrapper needed.
  const built = createHQSyncClient({ grants: await cachedGrantSlugs() });
  HQSync.client = built.client;
  HQSync.conflictHandler = built.conflictHandler;
  HQSync.surfaces = built.surfaces;
  HQSync.vendor = built.vendor;
  HQSync.ready = true;
} catch (err) {
  HQSync.error = err;
  if (typeof console !== 'undefined' && console.warn) {
    console.warn('[hq-sync] client construction failed; the page is unaffected', err);
  }
}

window.HQSync = HQSync;

export default HQSync;
