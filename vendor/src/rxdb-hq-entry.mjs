// vendor/src/rxdb-hq-entry.mjs — the ENTRY POINT for the committed browser bundle.
//
// This file is INPUT to ../build-vendor.sh. It is not loaded by anything at
// runtime; the browser loads the generated ../rxdb.bundle.js instead.
//
// It exists to make the bundle's public surface an explicit, reviewable list
// rather than "whatever `import * from 'rxdb'` happens to drag in". Adding an
// export here is a deliberate act that widens what ships to the truck.
//
// ⚠ SPIKE ARTEFACT (card `sync-rxdb-browser-delivery-spike`, 2026-07-26).
//    This proves a DELIVERY PATH. It does not adopt one: no production HQ page
//    imports this bundle. Adoption is `sync-rxdb-schema-and-replication`.

export {
    createRxDatabase,
    removeRxDatabase,
    addRxPlugin,
    defaultConflictHandler,
    RXDB_VERSION
} from 'rxdb';

// Dexie/IndexedDB — the whole point of the browser leg. NOT memory storage.
export { getRxStorageDexie } from 'rxdb/plugins/storage-dexie';

// The replication plugin W2's Node proof exercised, unchanged.
export { replicateSupabase } from 'rxdb/plugins/replication-supabase';

// Multi-tab leader election. `waitForLeadership` defaults to TRUE in a browser;
// W2's Node harness had to force it false (sharp edge 10). This export is what
// lets a page observe which tab won.
export { RxDBLeaderElectionPlugin } from 'rxdb/plugins/leader-election';

// ==========================================================================
// DELIBERATELY ABSENT: `rxdb/plugins/dev-mode` and `rxdb/plugins/validate-ajv`.
//
// This was going to be a size trade-off — dev-mode + its mandatory ajv wrapper
// cost 201,671 bytes raw / 57,174 bytes gzip (708,556/205,765 with them,
// 506,885/148,591 without; measured 2026-07-26). It is not a size trade-off.
// It is a correctness decision, and the measurement that forced it is:
//
//   node_modules/rxdb/dist/esm/plugins/dev-mode/dev-mode-tracking.js
//
// `addRxPlugin(RxDBDevModePlugin)` calls `addDevModeTrackingIframe()`, which
// appends a hidden 1×1 <iframe src="https://rxdb.info/html/dev-mode-iframe.html">
// to document.body. Its own comment reads "Only run this in browser AND
// localhost AND dev-mode. Make sure this is never used in production by
// someone." — and in the shipped 17.4.0 build the `!isLocalHost()` term of that
// guard IS COMMENTED OUT. The only thing that suppresses it is a paid premium
// flag. OBSERVED, not read: leg 1 caught the page reaching host `rxdb.info`.
//
// For a food-truck PWA whose entire reason for adopting RxDB is offline
// correctness, shipping a third-party phone-home (which hashes location.host)
// into every crew member's page load is not a trade-off to weigh. It is the
// thing we are trying not to do.
//
// Without dev-mode the ajv wrapper is also unnecessary — W2's sharp edge 9
// (dev-mode throws DVM1 on an unwrapped storage) only applies WITH dev-mode.
// Plain getRxStorageDexie() is used directly.
//
// Cost of the exclusion, stated honestly: no schema/typing checks, and RxDB
// errors arrive as bare codes with an rxdb.info doc link instead of prose. A
// developer who wants them adds `rxdb/plugins/dev-mode` to this file, runs
// build-vendor.sh, and MUST NOT commit the result.
// ==========================================================================

// The Supabase client. FORK 4 (T-22 decision 51) settled on a gateway-less
// stack plus a permanent client-construction helper; this is the constructor
// that helper will wrap.
export { createClient } from '@supabase/supabase-js';

// --------------------------------------------------------------------------
// Upgrade tripwire — FORK 4's "smoke test that fails loudly on upgrade".
//
// These values are stamped in by build-vendor.sh at generate time from the
// ACTUAL resolved versions in vendor/package-lock.json, not from a hand-typed
// constant. A smoke test asserts them; regenerating the bundle against
// different versions without updating the test therefore fails loudly instead
// of silently shipping a different engine to the truck.
// --------------------------------------------------------------------------
export const VENDOR_BUILD = {
    rxdb: __VENDOR_RXDB_VERSION__,
    supabaseJs: __VENDOR_SUPABASE_VERSION__,
    esbuild: __VENDOR_ESBUILD_VERSION__,
    generatedBy: 'vendor/build-vendor.sh'
};
