// tests/sync-rxdb-client.spec.js — the same-origin client-construction helper.
//
// Card `sync-rxdb-replication-and-conflict-handler` (overnight-20260801, C1).
// Ledger decisions 51 (gateway-less + permanent helper), 69 (same-origin door),
// 56 (umbrella slugs), 59 (the vendored bundle's adoption).
//
// Two halves:
//
//   1. NODE — the URL rewriting, the credential stripping, the umbrella
//      expansion and the version pin. These are pure functions and the real
//      `createClient`, driven with an injected fetch/WebSocket, so they need no
//      browser and no server. The library's URL derivation is MEASURED here,
//      which is the whole point of decision 51's rider: an upgrade that moves
//      `<baseUrl>/rest/v1` reds this file instead of 404-ing on a truck.
//
//   2. BROWSER — that `workflows.html` really does import the bundle and really
//      does construct the client, and that the RxDB collections really carry
//      HQ's conflict handler rather than RxDB's default. A pure-function suite
//      cannot prove wiring; this half can.

const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');
const { pathToFileURL } = require('url');

const REPO_ROOT = path.resolve(__dirname, '..');
const CLIENT_PATH = path.join(REPO_ROOT, 'sync-rxdb', 'client.js');
const BUNDLE_PATH = path.join(REPO_ROOT, 'vendor', 'rxdb.bundle.js');

async function loadClient() {
  return import(pathToFileURL(CLIENT_PATH).href);
}
async function loadBundle() {
  return import(pathToFileURL(BUNDLE_PATH).href);
}

// A WebSocket stand-in that records the handshake URL and never opens a socket.
function recordingSocket(sink) {
  return class FakeSocket {
    constructor(url) { sink.push(url); this.readyState = 0; this.binaryType = 'arraybuffer'; }
    send() {}
    close() {}
    addEventListener() {}
    removeEventListener() {}
  };
}

// ===========================================================================
// THE VERSION PIN — decision 51's rider, three-way.
// ===========================================================================
test.describe('vendor pin — fails LOUDLY on upgrade', () => {
  test('vendor/package.json, the generated bundle and PINNED_VENDOR all agree', async () => {
    const pkg = JSON.parse(fs.readFileSync(path.join(REPO_ROOT, 'vendor', 'package.json'), 'utf8'));
    const { VENDOR_BUILD } = await loadBundle();
    const { PINNED_VENDOR } = await loadClient();

    // The INPUT to build-vendor.sh.
    expect(pkg.dependencies['@supabase/supabase-js']).toBe(PINNED_VENDOR.supabaseJs);
    expect(pkg.dependencies.rxdb).toBe(PINNED_VENDOR.rxdb);
    // The OUTPUT, stamped in by build-vendor.sh from the resolved lockfile.
    expect(VENDOR_BUILD.supabaseJs).toBe(PINNED_VENDOR.supabaseJs);
    expect(VENDOR_BUILD.rxdb).toBe(PINNED_VENDOR.rxdb);
    // ...and the lockfile is what build-vendor.sh actually resolves from.
    const lock = JSON.parse(fs.readFileSync(path.join(REPO_ROOT, 'vendor', 'package-lock.json'), 'utf8'));
    const locked = lock.packages['node_modules/@supabase/supabase-js'];
    expect(locked.version).toBe(PINNED_VENDOR.supabaseJs);
  });

  test('assertVendorPin throws with an actionable message when the bundle moves', async () => {
    const { assertVendorPin, PINNED_VENDOR } = await loadClient();
    expect(() => assertVendorPin({ rxdb: PINNED_VENDOR.rxdb, supabaseJs: '2.110.0' }))
      .toThrow(/rest\/v1/);
    expect(() => assertVendorPin({ rxdb: '18.0.0', supabaseJs: PINNED_VENDOR.supabaseJs }))
      .toThrow(/does not match the pin/);
    expect(() => assertVendorPin(null)).toThrow(/does not match the pin/);
    // ...and does NOT throw on the real one. (No argument means "the bundle's
    // own VENDOR_BUILD", which is the production call.)
    expect(() => assertVendorPin()).not.toThrow();
  });
});

// ===========================================================================
// THE COUPLING, MEASURED — this is what the pin exists to protect.
// ===========================================================================
test.describe('supabase-js URL derivation (the coupling decision 51 accepted)', () => {
  test('given <origin>/sync it derives <origin>/sync/rest/v1 and <origin>/sync/realtime/v1', async () => {
    // If THIS test ever changes, `rewriteRestPath` and `rewriteRealtimeUrl`
    // below are wrong and the helper is silently 404-ing in production.
    const { createClient } = await loadBundle();
    // Node 20 has no global WebSocket and supabase-js constructs its Realtime
    // client eagerly, so a transport must be supplied even to read a URL.
    const c = createClient('http://hq.test/sync', 'k', {
      accessToken: async () => 'k',
      global: { fetch: async () => new Response('[]') },
      realtime: { transport: recordingSocket([]) },
    });
    expect(c.rest.url.toString()).toBe('http://hq.test/sync/rest/v1');
    expect(c.realtimeUrl.href).toBe('ws://hq.test/sync/realtime/v1');
  });
});

// ===========================================================================
// URL REWRITING.
// ===========================================================================
test.describe('the door\'s addresses', () => {
  const REST_CASES = [
    ['/sync/rest/v1/checklist_templates', '/sync/rest/checklist_templates'],
    ['/sync/rest/v1/rpc/hq_can_see_template', '/sync/rest/rpc/hq_can_see_template'],
    ['/sync/rest/v1', '/sync/rest'],
    // A table legitimately named v1 deeper in the path is NOT touched.
    ['/sync/rest/v1/v1', '/sync/rest/v1'],
    // Not under the REST prefix — untouched.
    ['/sync/realtime/v1/websocket', '/sync/realtime/v1/websocket'],
    ['/api/v1/workflow/templates', '/api/v1/workflow/templates'],
  ];

  test('the REST rewrite case table is non-empty', () => {
    // B-22/B-23/B-24: the loop below is only evidence if it has a population.
    expect(REST_CASES.length).toBe(6);
  });

  for (const [input, expected] of REST_CASES) {
    test(`rewriteRestPath: ${input} -> ${expected}`, async () => {
      const { rewriteRestPath } = await loadClient();
      expect(rewriteRestPath(input)).toBe(expected);
    });
  }

  test('rewriteRealtimeUrl points at /sync/realtime/socket/websocket and DROPS apikey', async () => {
    const { rewriteRealtimeUrl } = await loadClient();
    const out = rewriteRealtimeUrl('ws://hq.test/sync/realtime/v1/websocket?apikey=SECRET&vsn=1.0.0');
    const u = new URL(out);
    expect(u.pathname).toBe('/sync/realtime/socket/websocket');
    expect(u.searchParams.get('vsn')).toBe('1.0.0');
    // 🛑 The door sets apikey. A client that sends one is sending a credential
    // the door then discards — a lie in every log between here and there.
    expect(u.searchParams.has('apikey')).toBe(false);
    expect(out).not.toContain('SECRET');
  });

  test('the constants match backend/internal/sync/proxy.go', async () => {
    const { REST_PREFIX, REALTIME_PREFIX, REALTIME_SOCKET_PATH, SYNC_BASE_PATH } = await loadClient();
    const go = fs.readFileSync(path.join(REPO_ROOT, 'backend', 'internal', 'sync', 'proxy.go'), 'utf8');
    // Read out of the Go source rather than restated, so a rename of the door's
    // prefix reds here instead of 404-ing at runtime.
    expect(go).toContain(`ProxyPrefix         = "${SYNC_BASE_PATH}"`);
    expect(go).toContain(`ProxyRESTPrefix     = "${REST_PREFIX}"`);
    expect(go).toContain(`ProxyRealtimePrefix = "${REALTIME_PREFIX}"`);
    expect(REALTIME_SOCKET_PATH.startsWith(REALTIME_PREFIX + '/')).toBe(true);
  });
});

// ===========================================================================
// THE CONSTRUCTED CLIENT — driven end to end with injected transports.
// ===========================================================================
test.describe('createHQSupabaseClient — same-origin, gateway-less, credential-free', () => {
  test('a REST query lands on /sync/rest/<table> with no /v1 and no credentials', async () => {
    const { createHQSupabaseClient } = await loadClient();
    const calls = [];
    const fetchImpl = async (url, init) => {
      calls.push({ url, headers: [...new Headers(init.headers).keys()], credentials: init.credentials });
      return new Response('[]', { status: 200, headers: { 'Content-Type': 'application/json' } });
    };
    const client = createHQSupabaseClient({
      origin: 'http://hq.test', fetchImpl, WebSocketImpl: recordingSocket([]),
    });
    await client.from('checklist_templates').select('*').limit(1);

    expect(calls).toHaveLength(1);
    const u = new URL(calls[0].url);
    expect(u.origin).toBe('http://hq.test');
    expect(u.pathname).toBe('/sync/rest/checklist_templates');
    expect(u.pathname).not.toContain('/v1');
    // 🛑 decision 69 / proxy.go: the client never holds the substrate
    // credential. The door mints per request and injects.
    expect(calls[0].headers).not.toContain('authorization');
    expect(calls[0].headers).not.toContain('apikey');
    expect(u.searchParams.has('apikey')).toBe(false);
    // The HQ session cookie is what the door authenticates on.
    expect(calls[0].credentials).toBe('same-origin');
  });

  test('the realtime handshake goes to /sync/realtime/socket/websocket?vsn=1.0.0', async () => {
    const { createHQSupabaseClient, REALTIME_VSN } = await loadClient();
    const sockets = [];
    const client = createHQSupabaseClient({
      origin: 'http://hq.test',
      fetchImpl: async () => new Response('[]'),
      WebSocketImpl: recordingSocket(sockets),
    });
    client.channel('probe').subscribe(() => {});
    await new Promise((r) => setTimeout(r, 1200));

    expect(sockets.length, 'no websocket handshake was attempted').toBeGreaterThan(0);
    const u = new URL(sockets[0]);
    expect(u.protocol).toBe('ws:');
    expect(u.pathname).toBe('/sync/realtime/socket/websocket');
    expect(u.searchParams.get('vsn')).toBe(REALTIME_VSN);
    expect(REALTIME_VSN).toBe('1.0.0');
    expect(u.searchParams.has('apikey')).toBe(false);
    await client.removeAllChannels();
  });

  test('nothing anywhere in the client layer fetches /api/v1/sync/token', () => {
    // The proxy mints per request. A client that also minted would be a second
    // identity path, and the door discards what it sends anyway.
    for (const f of ['client.js', 'conflict-handler.js', 'bootstrap.js']) {
      const src = fs.readFileSync(path.join(REPO_ROOT, 'sync-rxdb', f), 'utf8');
      const code = src.replace(/^\s*\/\/.*$/gm, ''); // the comments may DISCUSS it
      expect(code, `${f} must not call the token endpoint`).not.toContain('/api/v1/sync/token');
      expect(code, `${f} must not set a bearer`).not.toMatch(/Bearer\s/);
    }
  });

  test('construction makes no network request at all', async () => {
    const { createHQSupabaseClient } = await loadClient();
    let called = 0;
    createHQSupabaseClient({
      origin: 'http://hq.test',
      fetchImpl: async () => { called++; return new Response('[]'); },
      WebSocketImpl: recordingSocket([]),
    });
    await new Promise((r) => setTimeout(r, 200));
    // A page constructs this on load with the door deliberately shut (503).
    expect(called).toBe(0);
  });

  test('createHQSyncClient exposes the HQ conflict handler, not RxDB\'s default', async () => {
    const { createHQSyncClient } = await loadClient();
    const built = createHQSyncClient({
      origin: 'http://hq.test',
      fetchImpl: async () => new Response('[]'),
      WebSocketImpl: recordingSocket([]),
      grants: ['inventory'],
    });
    const out = built.conflictHandler.resolve({
      assumedMasterState: { id: 'a', name: 'base', requires_approval: false },
      newDocumentState: { id: 'a', name: 'FORK', requires_approval: false },
      realMasterState: { id: 'a', name: 'base', requires_approval: true },
    }, 'replication-resolve-conflict');
    // RxDB's default would answer `{name:'base', requires_approval:true}`.
    expect(out.name).toBe('FORK');
    expect(out.requires_approval).toBe(true);
  });
});

// ===========================================================================
// OBLIGATION 4 — umbrella slugs (decision 56).
// ===========================================================================
test.describe('expandGrantSlugs — obligation 4', () => {
  test('inventory expands to the two per-tab surfaces the user can actually reach', async () => {
    const { expandGrantSlugs } = await loadClient();
    expect(expandGrantSlugs(['inventory']))
      .toEqual(['inventory', 'inventory-cost', 'inventory-trends']);
  });

  test('the umbrella table matches what main.go actually mounts', () => {
    // The expansion is only correct because RequirePermission takes the
    // umbrella as a second candidate. Read it out of the Go source rather than
    // trusting a table that can drift.
    const go = fs.readFileSync(path.join(REPO_ROOT, 'backend', 'cmd', 'server', 'main.go'), 'utf8');
    expect(go).toContain('RequirePermission(pool, "inventory-trends", "inventory")');
    expect(go).toContain('RequirePermission(pool, "inventory-cost", "inventory")');
  });

  test('a narrow per-tab grant is NOT widened — expansion is one-way', async () => {
    const { expandGrantSlugs } = await loadClient();
    expect(expandGrantSlugs(['inventory-trends'])).toEqual(['inventory-trends']);
  });

  test('idempotent, de-duplicating, order-independent, and empty-safe', async () => {
    const { expandGrantSlugs } = await loadClient();
    const once = expandGrantSlugs(['operations', 'inventory']);
    expect(expandGrantSlugs(once)).toEqual(once);
    expect(expandGrantSlugs(['inventory', 'inventory', 'inventory-cost'])).toEqual(once.filter((s) => s.startsWith('inventory')));
    expect(expandGrantSlugs([])).toEqual([]);
    expect(expandGrantSlugs(undefined)).toEqual([]);
    expect(expandGrantSlugs([null, '', 7, 'operations'])).toEqual(['operations']);
  });
});

// ===========================================================================
// BROWSER — the wiring a pure-function suite cannot prove.
// ===========================================================================
test.describe('workflows.html actually imports and constructs the client', () => {
  // workflows.html redirects to login.html on a 401, so the module would never
  // get a chance to run against an anonymous page.
  async function login(page) {
    await page.goto('/login.html');
    await page.fill('input[type="email"]', 'jamal@yumyums.kitchen');
    await page.fill('input[type="password"]', 'test123');
    await page.click('button.btn');
    await page.waitForURL((url) => !url.pathname.includes('login'));
  }

  test('window.HQSync is constructed, pinned and umbrella-expanded', async ({ page }) => {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));
    await page.addInitScript(() => {
      localStorage.setItem('hq_apps', JSON.stringify([{ slug: 'inventory' }, { slug: 'operations' }]));
    });
    await login(page);
    await page.goto('/workflows.html');
    await page.waitForFunction(() => window.HQSync !== undefined, null, { timeout: 15000 });

    const state = await page.evaluate(() => ({
      ready: window.HQSync.ready,
      error: window.HQSync.error ? String(window.HQSync.error) : null,
      surfaces: window.HQSync.surfaces,
      vendor: window.HQSync.vendor,
      hasClient: !!window.HQSync.client,
      hasFrom: typeof window.HQSync.client.from === 'function',
    }));

    expect(state.error).toBeNull();
    expect(state.ready).toBe(true);
    expect(state.hasClient).toBe(true);
    expect(state.hasFrom).toBe(true);
    expect(state.vendor.rxdb).toBe('17.4.0');
    expect(state.vendor.supabaseJs).toBe('2.109.0');
    // Obligation 4, observed in a real page: the cached claim is
    // [inventory, operations]; the reachable set is four.
    expect(state.surfaces).toEqual([
      'inventory', 'inventory-cost', 'inventory-trends', 'operations',
    ]);
    expect(errors).toEqual([]);
  });

  test('the RxDB collections carry HQ\'s conflict handler, not RxDB\'s default', async ({ page }) => {
    // THE WIRING PROOF. The pure suite pins the merge rule; this pins that the
    // rule is what a real RxDB collection would consult. Dexie needs IndexedDB,
    // so this half can only run in a browser.
    await login(page);
    await page.goto('/workflows.html');
    await page.waitForFunction(() => window.HQSync !== undefined, null, { timeout: 15000 });

    const result = await page.evaluate(async () => {
      const db = await window.HQSync.createDatabase({
        name: 'hq_sync_wiring_probe_' + Date.now(),
        multiInstance: false,
      });
      const collections = Object.keys(db.collections);
      const out = db.templates.conflictHandler.resolve({
        assumedMasterState: {
          id: 't', name: 'base', requires_approval: false,
          created_at: 'x', updated_by: null, updated_at: 'x',
        },
        newDocumentState: {
          id: 't', name: 'FORK RENAME', requires_approval: false,
          created_at: 'x', updated_by: null, updated_at: 'x',
        },
        realMasterState: {
          id: 't', name: 'base', requires_approval: true,
          created_at: 'x', updated_by: null, updated_at: 'x',
        },
      }, 'replication-resolve-conflict');
      const resolved = await out;
      await db.remove();
      return { collections, name: resolved.name, requires_approval: resolved.requires_approval };
    });

    // All four replicated collections plus the LOCAL conflict record (decision 89).
    expect(result.collections.sort()).toEqual([
      'approvals', 'checklists', 'conflict_records', 'responses', 'templates',
    ]);
    // RxDB's default would have answered 'base'. Both edits survive.
    expect(result.name).toBe('FORK RENAME');
    expect(result.requires_approval).toBe(true);
  });

  test('no user write path was rerouted — workflows.html still autosaves through the API', () => {
    // This card is import + construction ONLY; the write-path swap belongs to
    // `sync-hard-cutover`. A merge that quietly reroutes a write reds here.
    const src = fs.readFileSync(path.join(REPO_ROOT, 'workflows.html'), 'utf8');
    expect(src).toContain('autoSaveField');
    expect(src).toContain('saveResponse');
    // The page must not call the RxDB layer for anything at all yet.
    expect(src).not.toContain('HQSync.createDatabase');
    expect(src).not.toContain('HQSync.startReplication');
    expect(src).not.toContain('HQSync.client');
  });
});
