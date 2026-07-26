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

// dev-mode + its mandatory ajv validator wrapper (W2 sharp edge 9: dev-mode
// throws DVM1 unless the storage is wrapped). Kept IN the spike bundle on
// purpose — a spike wants loud errors. See build-vendor.sh for the measured
// size cost of carrying it, and runbook half 3 for the recommendation that a
// production bundle drop both.
export { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
export { wrappedValidateAjvStorage } from 'rxdb/plugins/validate-ajv';

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
