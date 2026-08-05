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
// It also does NOT create the RxDB database and does NOT start replication,
// and both omissions are deliberate rather than unfinished:
//
//   * `HQ_SYNC_REST_URL` / `HQ_SYNC_REALTIME_URL` are unset in every
//     environment, so the door answers 503 to everything. Starting replication
//     today would put a permanent retry loop and a stream of console errors on
//     every crew phone, in exchange for nothing.
//     🛑 This used to call the unset variables "still an armed precondition"
//     gated on the row-visibility-RLS card landing. That card MERGED on
//     2026-08-01 (`bbbfc64`; the roadmap flipped it DONE at `914536c`), so the
//     gate named here had been satisfied for five days while still reading as
//     open — and while hiding the precondition that IS open: nothing calls
//     `createHQSyncDatabase()` or `startHQReplication()` yet. That is
//     `sync-hard-cutover`'s job, and it is what this cycle exists to deliver.
//     Corrected on card `repo-hygiene-preconditions`, run 20260806.
//   * `createHQSyncDatabase()` writes IndexedDB. Creating local state before
//     anything reads it means a schema decision made tonight is already on
//     every phone before the card that depends on it has been reviewed.
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
function cachedGrantSlugs() {
  try {
    const raw = localStorage.getItem('hq_apps');
    if (!raw) return [];
    const apps = JSON.parse(raw);
    return Array.isArray(apps) ? apps.map((a) => a && a.slug).filter(Boolean) : [];
  } catch (err) {
    return [];
  }
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
  // Deferred, for `sync-hard-cutover` and for C2's tests to drive.
  createDatabase: createHQSyncDatabase,
  // 🛑 REQUIRES a scope — `startReplication(db, client, {scope:{checklistId,
  // templateId, fieldIds}})`. It throws without one, and `checklistId` AND
  // `templateId` are both mandatory. Replication is scoped to the open
  // checklist and is never pulled whole (preference architecture/C-2, ledger
  // T-29 decision 105); a default would widen that silently.
  // 🛑 CANCEL the previous states before starting a re-scoped replication.
  startReplication: startHQReplication,
  // Pure helpers C2 renders against.
  describeConflict,
  resolveConflict,
  createConflictHandler: createHQConflictHandler,
  expandGrantSlugs,
  assertVendorPin,
};

try {
  const built = createHQSyncClient({ grants: cachedGrantSlugs() });
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
