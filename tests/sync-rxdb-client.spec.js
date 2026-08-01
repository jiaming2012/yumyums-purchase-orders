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
// [SCOPE-01] REPLICATION SCOPE — preference `architecture/C-2`, ledger T-29
// decision 105. Card `sync-replication-scope-per-checklist` (20260802, A1).
//
// THE DEFECT THIS SECTION PROVES AND THEN PINS.
//
// `startHQReplication` looped all four replicated collections with
// `pull:{batchSize:50}` and NO selector, filter or query modifier — so every
// device replicated every field answer of every submission ever taken. Two
// consequences: the RLS predicate is re-evaluated per row on every page (the
// whole of Fork 1's ~23 s figure), and `responses` grows forever on a phone
// that was only ever meant to hold the checklist in front of the crew member.
//
// HOW THIS IS MEASURED RATHER THAN ASSERTED. The shipped
// `rxdb/plugins/replication-supabase` builds its pull like this — read out of
// `vendor/rxdb.bundle.js`, not out of docs:
//
//     var S = e.client.from(e.tableName).select("*");
//     if (e.pull?.queryBuilder) {
//       var R = e.pull.queryBuilder({ query: S, lastPulledCheckpoint: g, batchSize: _ });
//       R && (S = R);
//     }
//     if (g) { S = S.or('"_modified".gt.…');  }        // the checkpoint, ANDed
//     S = S.order(…).order(…).limit(_);
//
// So `pull.queryBuilder` is the one supported place a scope can be attached,
// it runs BEFORE the checkpoint clause, and PostgREST ANDs the two. The
// harness below reproduces exactly that call shape, records the PostgREST
// filters the queryBuilder emits, and EVALUATES them over a fixture holding
// two checklists — one the device opened and one it never did.
// ===========================================================================

// --- the fixture: one checklist opened, one never opened -------------------
const OPEN = {
  checklistId: 'chk-open-0000-0000-0000-000000000001',
  templateId: 'tpl-open-0000-0000-0000-000000000001',
  fieldIds: ['fld-open-a', 'fld-open-b'],
};
const NEVER = {
  checklistId: 'chk-never-000-0000-0000-000000000002',
  templateId: 'tpl-never-000-0000-0000-000000000002',
  fieldIds: ['fld-never-x'],
};

const SCOPE_FIXTURE = {
  templates: [
    { id: OPEN.templateId, archived_at: null },
    { id: NEVER.templateId, archived_at: null },
    { id: 'tpl-archived-00-0000-0000-000000000003', archived_at: '2026-01-01T00:00:00Z' },
  ],
  checklists: [
    { id: OPEN.checklistId, template_id: OPEN.templateId },
    { id: NEVER.checklistId, template_id: NEVER.templateId },
  ],
  responses: [
    // In scope: submitted answers on the open checklist.
    { id: 'rsp-1', submission_id: OPEN.checklistId, field_id: OPEN.fieldIds[0] },
    { id: 'rsp-2', submission_id: OPEN.checklistId, field_id: OPEN.fieldIds[1] },
    // In scope: a DRAFT on the open checklist. `submission_id` is null until
    // submit (migration 0012's partial unique index), and drafts are precisely
    // what a crew member fills offline — the collection that must sync best is
    // the one whose FK is absent.
    { id: 'rsp-3', submission_id: null, field_id: OPEN.fieldIds[0] },
    // OUT of scope: a submitted answer on a checklist never opened here.
    { id: 'rsp-4', submission_id: NEVER.checklistId, field_id: NEVER.fieldIds[0] },
    // OUT of scope: someone else's draft, on a field this device never saw.
    { id: 'rsp-5', submission_id: null, field_id: NEVER.fieldIds[0] },
  ],
  approvals: [
    { id: 'apr-1', submission_id: OPEN.checklistId, field_id: OPEN.fieldIds[0] },
    { id: 'apr-2', submission_id: NEVER.checklistId, field_id: NEVER.fieldIds[0] },
  ],
};

// The rows a device that never opened `NEVER.checklistId` must NOT end up
// holding. Named per collection so a shrunk fixture reds rather than passes.
const MUST_NOT_HOLD = {
  templates: [NEVER.templateId],
  checklists: [NEVER.checklistId],
  responses: ['rsp-4', 'rsp-5'],
  approvals: ['apr-2'],
};
const MUST_HOLD = {
  templates: [OPEN.templateId],
  checklists: [OPEN.checklistId],
  responses: ['rsp-1', 'rsp-2', 'rsp-3'],
  approvals: ['apr-1'],
};

// --- a PostgREST stand-in that both RECORDS and EVALUATES ------------------
//
// It answers the subset of the PostgREST builder the supabase replication
// plugin and this card's queryBuilder actually use. Anything else throws, so a
// filter written in an operator this harness cannot evaluate fails loudly
// instead of being silently ignored (which would make the test pass by
// accident — B-22/B-23/B-24).

function splitTop(s, sep) {
  const out = [];
  let depth = 0;
  let cur = '';
  for (const ch of s) {
    if (ch === '(') depth++;
    if (ch === ')') depth--;
    if (ch === sep && depth === 0) { out.push(cur); cur = ''; continue; }
    cur += ch;
  }
  out.push(cur);
  return out;
}

// `"col".op.value` | `and(…)` | `or(…)`  → a row predicate.
function parseClause(raw) {
  const s = raw.trim();
  if (s.startsWith('and(')) {
    const parts = splitTop(s.slice(4, -1), ',').map(parseClause);
    return (row) => parts.every((p) => p(row));
  }
  if (s.startsWith('or(')) {
    const parts = splitTop(s.slice(3, -1), ',').map(parseClause);
    return (row) => parts.some((p) => p(row));
  }
  const m = /^"([^"]+)"\.([a-z]+)\.(.*)$/s.exec(s);
  if (!m) throw new Error('[scope-harness] cannot parse PostgREST clause: ' + s);
  const [, col, op, rest] = m;
  if (op === 'eq') return (row) => String(row[col]) === rest;
  if (op === 'is') {
    if (rest !== 'null') throw new Error('[scope-harness] only is.null is supported: ' + s);
    return (row) => row[col] === null || row[col] === undefined;
  }
  if (op === 'in') {
    const vals = splitTop(rest.replace(/^\(|\)$/g, ''), ',').map((v) => v.trim().replace(/^"|"$/g, ''));
    return (row) => vals.includes(String(row[col]));
  }
  throw new Error('[scope-harness] unsupported PostgREST operator: ' + op);
}

function fakeQuery(rows, log) {
  const preds = [];
  const q = {
    select(cols) { log.push(['select', cols]); return q; },
    eq(col, val) { log.push(['eq', col, val]); preds.push((r) => String(r[col]) === String(val)); return q; },
    is(col, val) {
      log.push(['is', col, val]);
      if (val !== null) throw new Error('[scope-harness] only .is(col,null) is supported');
      preds.push((r) => r[col] === null || r[col] === undefined);
      return q;
    },
    in(col, vals) {
      log.push(['in', col, vals]);
      const set = vals.map(String);
      preds.push((r) => set.includes(String(r[col])));
      return q;
    },
    or(expr) {
      log.push(['or', expr]);
      const parts = splitTop(expr, ',').map(parseClause);
      preds.push((r) => parts.some((p) => p(r)));
      return q;
    },
    order() { return q; },
    limit() { return q; },
    // What PostgREST would return for the accumulated (ANDed) filters.
    rows() { return rows.filter((r) => preds.every((p) => p(r))); },
  };
  return q;
}

// Drive `startHQReplication` with an injected `replicate`, then run each
// collection's pull exactly the way the vendored plugin does.
async function pullEachCollection(scope) {
  const { startHQReplication } = await loadClient();
  const captured = {};
  const db = {
    templates: {}, checklists: {}, responses: {}, approvals: {}, conflict_records: {},
  };
  startHQReplication(db, {}, {
    scope,
    waitForLeadership: false,
    replicate: (o) => {
      captured[o.collectionKey || o.tableName] = o;
      return { conflict$: { subscribe() {} } };
    },
  });

  const byKey = {
    templates: 'checklist_templates',
    checklists: 'checklist_submissions',
    responses: 'submission_responses',
    approvals: 'submission_rejections',
  };
  const out = {};
  for (const [key, table] of Object.entries(byKey)) {
    const opts = captured[key] || captured[table];
    if (!opts) throw new Error('[scope-harness] no replication started for ' + key);
    const log = [];
    let query = fakeQuery(SCOPE_FIXTURE[key], log).select('*');
    const qb = opts.pull && opts.pull.queryBuilder;
    if (qb) {
      const next = qb({ query, lastPulledCheckpoint: undefined, batchSize: opts.pull.batchSize });
      if (next) query = next;
    }
    out[key] = { rows: query.rows().map((r) => r.id), log, opts, hasQueryBuilder: !!qb };
  }
  return out;
}

test.describe('[SCOPE-01] replication is scoped to the open checklist (C-2)', () => {
  test('the fixture really does hold rows for a checklist this device never opened', () => {
    // B-22/B-23/B-24: every assertion below is only evidence because these
    // subject sets are non-empty. A shrunk fixture reds here first.
    for (const key of ['templates', 'checklists', 'responses', 'approvals']) {
      expect(MUST_NOT_HOLD[key].length, `${key}: no never-opened rows to prove anything with`)
        .toBeGreaterThan(0);
      expect(MUST_HOLD[key].length, `${key}: no in-scope rows, so "returns nothing" would pass`)
        .toBeGreaterThan(0);
      const ids = SCOPE_FIXTURE[key].map((r) => r.id);
      for (const id of MUST_NOT_HOLD[key]) expect(ids).toContain(id);
      for (const id of MUST_HOLD[key]) expect(ids).toContain(id);
    }
    expect(SCOPE_FIXTURE.responses.length).toBe(5);
  });

  test('the vendored plugin still offers pull.queryBuilder as the scoping seam', () => {
    // If an upgrade removes this extension point the scope silently stops
    // being applied and every phone goes back to holding everything.
    const bundle = fs.readFileSync(BUNDLE_PATH, 'utf8');
    expect(bundle).toContain('queryBuilder');
    expect(bundle).toContain('lastPulledCheckpoint');
  });

  test('a device does NOT hold rows for a checklist it never opened', async () => {
    const pulled = await pullEachCollection({
      checklistId: OPEN.checklistId,
      templateId: OPEN.templateId,
      fieldIds: OPEN.fieldIds,
    });

    for (const key of ['templates', 'checklists', 'responses', 'approvals']) {
      expect(pulled[key].hasQueryBuilder, `${key} pulls with NO query modifier — the whole collection`)
        .toBe(true);
      for (const id of MUST_NOT_HOLD[key]) {
        expect(pulled[key].rows, `${key}: replicated ${id}, which belongs to a checklist never opened`)
          .not.toContain(id);
      }
      // ...and it did not pass by returning nothing.
      expect(pulled[key].rows.sort()).toEqual([...MUST_HOLD[key]].sort());
    }
  });

  test('an offline DRAFT on the open checklist still replicates', async () => {
    // The scope must not be `submission_id.eq.<id>` alone: a draft has no
    // submission yet, and drafts are the offline case this whole layer exists
    // for. `rsp-3` is that row; `rsp-5` is the same shape on a field this
    // device never saw and must stay out.
    const pulled = await pullEachCollection({
      checklistId: OPEN.checklistId,
      templateId: OPEN.templateId,
      fieldIds: OPEN.fieldIds,
    });
    expect(pulled.responses.rows).toContain('rsp-3');
    expect(pulled.responses.rows).not.toContain('rsp-5');
  });

  test('replication REFUSES to start unscoped — C-2 is not a default, it is a gate', async () => {
    const { startHQReplication } = await loadClient();
    const db = { templates: {}, checklists: {}, responses: {}, approvals: {} };
    const replicate = () => ({ conflict$: { subscribe() {} } });
    expect(() => startHQReplication(db, {}, { replicate }))
      .toThrow(/scope/i);
    expect(() => startHQReplication(db, {}, { replicate, scope: {} }))
      .toThrow(/checklistId/);
  });

  test('scoped AND batched — the pull keeps its batch size', async () => {
    // C-2 says batched AND scoped. Scoping must not quietly drop the batching.
    const pulled = await pullEachCollection({
      checklistId: OPEN.checklistId,
      templateId: OPEN.templateId,
      fieldIds: OPEN.fieldIds,
    });
    for (const key of ['templates', 'checklists', 'responses', 'approvals']) {
      expect(pulled[key].opts.pull.batchSize).toBe(50);
      expect(pulled[key].opts.push.batchSize).toBe(50);
    }
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
