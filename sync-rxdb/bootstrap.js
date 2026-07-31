// sync-rxdb/bootstrap.js — the ONE module an HQ page loads.
//
// Card `sync-rxdb-replication-and-conflict-handler` (overnight-20260801, C1).
//
// ===========================================================================
// 🛑 IMPORT + CONSTRUCTION ONLY. NO WRITE PATH IS SWAPPED HERE.
// ===========================================================================
// This card's footprint in `workflows.html` is one `<script type="module">`
// tag. `autoSaveField` → `/saveResponse` → `DRAFT_RESPONSES` →
// `hydrateFieldState` is untouched and stays the live persistence path; the
// swap belongs to `sync-hard-cutover`.
//
// It also does NOT create the RxDB database and does NOT start replication,
// and both omissions are deliberate rather than unfinished:
//
//   * `HQ_SYNC_REST_URL` / `HQ_SYNC_REALTIME_URL` are unset in every
//     environment (still an armed precondition until
//     `sync-rxdb-row-visibility-rls` lands), so the door answers 503 to
//     everything. Starting replication today would put a permanent retry loop
//     and a stream of console errors on every crew phone, in exchange for
//     nothing.
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
