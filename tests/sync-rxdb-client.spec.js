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

// 🛑 G6 CORRECTION (F-2) — THE SIBLING. The fixture used to hold exactly one
// checklist per template and one approval per field id, which made two
// wrong-but-plausible scopes INDISTINGUISHABLE from the correct one. The
// reviewer mutated the fix to `checklists: template_id.eq.<templateId>` and to
// `approvals: field_id.in.(<fieldIds>)` and the suite stayed 6/6 green.
//
// Both mutations are realistic wrong answers, not strawmen: in HQ a template is
// filled DAILY and `checklist_submissions` grows one row per fill forever
// (`template_id.eq` is therefore a textbook C-2 violation), and field ids are
// per-template-VERSION and shared by every submission of that template
// (`field_id.in` therefore pulls every rejection on those fields across all
// submissions ever taken).
//
// `SIBLING` is a second submission of the OPEN checklist's OWN template that
// this device never opened — yesterday's fill of today's checklist. It is what
// separates "scoped to the open checklist" from "scoped to its template".
const SIBLING = {
  checklistId: 'chk-sibling-00-0000-0000-000000000004',
  // Same template, same field ids as OPEN — that is the whole point.
  templateId: null,
  fieldIds: null,
};

const SCOPE_FIXTURE = {
  templates: [
    { id: OPEN.templateId, archived_at: null, _modified: '2026-07-20T00:00:00Z' },
    { id: NEVER.templateId, archived_at: null, _modified: '2026-07-19T00:00:00Z' },
    {
      id: 'tpl-archived-00-0000-0000-000000000003',
      archived_at: '2026-01-01T00:00:00Z',
      _modified: '2026-01-01T00:00:00Z',
    },
  ],
  checklists: [
    { id: OPEN.checklistId, template_id: OPEN.templateId, _modified: '2026-08-02T08:10:00Z' },
    // OUT of scope, and the discriminator: SAME template as the open checklist,
    // different submission. `template_id.eq.<templateId>` returns this row.
    { id: SIBLING.checklistId, template_id: OPEN.templateId, _modified: '2026-08-01T07:00:00Z' },
    { id: NEVER.checklistId, template_id: NEVER.templateId, _modified: '2026-08-01T09:00:00Z' },
  ],
  responses: [
    // In scope: submitted answers on the open checklist.
    {
      id: 'rsp-1', submission_id: OPEN.checklistId, field_id: OPEN.fieldIds[0], _modified: '2026-08-02T08:10:00Z',
    },
    {
      id: 'rsp-2', submission_id: OPEN.checklistId, field_id: OPEN.fieldIds[1], _modified: '2026-08-02T08:10:00Z',
    },
    // In scope: a DRAFT on the open checklist. `submission_id` is null until
    // submit (migration 0012's partial unique index), and drafts are precisely
    // what a crew member fills offline — the collection that must sync best is
    // the one whose FK is absent.
    {
      id: 'rsp-3', submission_id: null, field_id: OPEN.fieldIds[0], _modified: '2026-08-02T08:10:00Z',
    },
    // OUT of scope: a submitted answer on a checklist never opened here.
    {
      id: 'rsp-4', submission_id: NEVER.checklistId, field_id: NEVER.fieldIds[0], _modified: '2026-08-01T09:00:00Z',
    },
    // OUT of scope: someone else's draft, on a field this device never saw.
    {
      id: 'rsp-5', submission_id: null, field_id: NEVER.fieldIds[0], _modified: '2026-08-01T09:00:00Z',
    },
    // OUT of scope, and the F-2 discriminator for this collection: yesterday's
    // SUBMITTED answer on the same template, therefore on an OPEN field id.
    // `field_id.in.(<OPEN fields>)` alone returns this row.
    {
      id: 'rsp-6', submission_id: SIBLING.checklistId, field_id: OPEN.fieldIds[0], _modified: '2026-08-01T07:00:00Z',
    },
  ],
  approvals: [
    {
      id: 'apr-1', submission_id: OPEN.checklistId, field_id: OPEN.fieldIds[0], _modified: '2026-08-02T08:11:00Z',
    },
    {
      id: 'apr-2', submission_id: NEVER.checklistId, field_id: NEVER.fieldIds[0], _modified: '2026-08-01T09:05:00Z',
    },
    // OUT of scope, and the F-2 discriminator: a rejection on an OPEN field id
    // but on YESTERDAY'S submission. `field_id.in.(<OPEN fields>)` returns it.
    {
      id: 'apr-3', submission_id: SIBLING.checklistId, field_id: OPEN.fieldIds[0], _modified: '2026-08-01T07:05:00Z',
    },
  ],
};

// The rows a device that only ever opened `OPEN.checklistId` must NOT end up
// holding. Named per collection so a shrunk fixture reds rather than passes.
const MUST_NOT_HOLD = {
  templates: [NEVER.templateId],
  checklists: [NEVER.checklistId, SIBLING.checklistId],
  responses: ['rsp-4', 'rsp-5', 'rsp-6'],
  approvals: ['apr-2', 'apr-3'],
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
  // Values may be double-quoted (which is how the scope emits them since the
  // G6 F-4 fix) or bare (which is how the vendored plugin emits its own
  // checkpoint clause). Both reach this parser, so both are read here.
  if (op === 'eq') return (row) => String(row[col]) === unquote(rest);
  if (op === 'gt') return (row) => String(row[col]) > unquote(rest);
  if (op === 'is') {
    if (rest !== 'null') throw new Error('[scope-harness] only is.null is supported: ' + s);
    return (row) => row[col] === null || row[col] === undefined;
  }
  if (op === 'in') {
    const vals = splitTop(rest.replace(/^\(|\)$/g, ''), ',').map((v) => unquote(v.trim()));
    return (row) => vals.includes(String(row[col]));
  }
  throw new Error('[scope-harness] unsupported PostgREST operator: ' + op);
}

// PostgREST's logic-tree grammar lets a value be double-quoted so it can carry
// `,` `(` `)` without being read as structure. Strip one layer, unescaping the
// way the grammar escapes.
function unquote(raw) {
  const s = raw.trim();
  if (s.length >= 2 && s.startsWith('"') && s.endsWith('"')) {
    return s.slice(1, -1).replace(/\\(["\\])/g, '$1');
  }
  return s;
}

function fakeQuery(rows, log) {
  const preds = [];
  const orderBy = [];
  let lim = null;
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
    // Real, not a no-op: the plugin derives its next checkpoint from
    // `lastOfArray(data)`, so which row is LAST is load-bearing. It orders by
    // `(_modified asc, id asc)` — read out of the bundle, same place the
    // checkpoint clause is.
    order(col, o) { orderBy.push([col, !o || o.ascending !== false]); return q; },
    limit(n) { lim = n; return q; },
    // What PostgREST would return for the accumulated (ANDed) filters.
    rows() {
      let out = rows.filter((r) => preds.every((p) => p(r)));
      for (const [col, asc] of [...orderBy].reverse()) {
        out = [...out].sort((a, b) => {
          const x = String(a[col]);
          const y = String(b[col]);
          if (x === y) return 0;
          return (x < y ? -1 : 1) * (asc ? 1 : -1);
        });
      }
      return lim == null ? out : out.slice(0, lim);
    },
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
      // Keyed by `tableName`, which the plugin needs anyway. The card shipped
      // an extra `collectionKey` option purely so this recorder could key on
      // it; the G6 round deleted it as dead production code (F-8) and this is
      // the proof it was never needed.
      captured[o.tableName] = o;
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
    expect(SCOPE_FIXTURE.responses.length).toBe(6);
    // 🛑 F-2. These four rows are what makes the fixture DISCRIMINATING rather
    // than merely populated. Without them, `checklists: template_id.eq.<tpl>`
    // and `approvals/responses: field_id.in.(<fields>)` are indistinguishable
    // from the correct per-checklist scope, and both mutations pass.
    expect(
      SCOPE_FIXTURE.checklists.filter((r) => r.template_id === OPEN.templateId).length,
      'need >1 checklist on the OPEN template or `template_id.eq` looks correct',
    ).toBeGreaterThan(1);
    expect(
      SCOPE_FIXTURE.approvals.filter(
        (r) => OPEN.fieldIds.includes(r.field_id) && r.submission_id !== OPEN.checklistId,
      ).length,
      'need an approval on an OPEN field id but a different submission, or `field_id.in` looks correct',
    ).toBeGreaterThan(0);
    expect(
      SCOPE_FIXTURE.responses.filter(
        (r) => OPEN.fieldIds.includes(r.field_id)
          && r.submission_id !== null && r.submission_id !== OPEN.checklistId,
      ).length,
      'need a SUBMITTED response on an OPEN field id but a different submission',
    ).toBeGreaterThan(0);
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

  test('scope values are QUOTED, so an id cannot rewrite the predicate [F-4]', async () => {
    const { serializeFilter, startHQReplication } = await loadClient();
    // The column was quoted and the value was not, so `,` inside a value was
    // read as grammar. Both halves of the fix are asserted: the value is now
    // quoted, AND a value carrying logic-tree punctuation is refused outright.
    expect(serializeFilter({ op: 'eq', column: 'id', value: 'abc' })).toBe('"id".eq."abc"');
    expect(serializeFilter({ op: 'in', column: 'field_id', values: ['a', 'b'] }))
      .toBe('"field_id".in.("a","b")');

    const db = { templates: {}, checklists: {}, responses: {}, approvals: {} };
    const replicate = () => ({ conflict$: { subscribe() {} } });
    expect(() => startHQReplication(db, {}, {
      replicate,
      scope: {
        // The reviewer's payload: a predicate true for every row, reached
        // THROUGH the thing this card calls a gate.
        checklistId: 'x,"id".not.is.null',
        templateId: OPEN.templateId,
      },
    })).toThrow(/checklistId must match/);
  });

  test('an omitted templateId is REFUSED, not widened to every template [F-5]', async () => {
    // This used to fall back to `archived_at.is.null` — the whole non-archived
    // collection — on a FORGOTTEN OPTIONAL ARGUMENT. C-2 requires a recorded
    // decision to widen; forgetting an argument is not one.
    const { startHQReplication, scopeFilterFor } = await loadClient();
    const db = { templates: {}, checklists: {}, responses: {}, approvals: {} };
    const replicate = () => ({ conflict$: { subscribe() {} } });
    expect(() => startHQReplication(db, {}, {
      replicate,
      scope: { checklistId: OPEN.checklistId },
    })).toThrow(/templateId/);
    // ...and there is no code path left that returns the widening filter.
    expect(scopeFilterFor('templates', {
      checklistId: OPEN.checklistId,
      templateId: OPEN.templateId,
    })).toEqual({ op: 'eq', column: 'id', value: OPEN.templateId });
  });

  test('a collection with no scope case refuses EAGERLY, not inside pull.handler [F-3]', async () => {
    // The plugin wraps `pull.handler` in `try{…}catch{ emit RC_PULL; retry }`,
    // an unbounded loop feeding an error stream nobody subscribes to. A
    // refusal raised in there is not a refusal, it is a spin. Assert the throw
    // happens while BUILDING the builder.
    const { scopePlanFor, makePullQueryBuilder } = await loadClient();
    const scope = { checklistId: OPEN.checklistId, templateId: OPEN.templateId };
    expect(() => scopePlanFor('a_fifth_collection', scope)).toThrow(/pulled whole/);
    expect(() => makePullQueryBuilder('a_fifth_collection', scope)).toThrow(/pulled whole/);
    // The bundle really does swallow handler throws into a retry.
    const bundle = fs.readFileSync(BUNDLE_PATH, 'utf8');
    expect(bundle).toContain('RC_PULL');
  });
});

// ===========================================================================
// [SCOPE-02] THE CHECKPOINT IS PART OF THE SCOPE — G6 finding F-1.
//
// THE DEFECT THIS SECTION PROVES AND THEN PINS. The card shipped
// `replicationIdentifier: "hq-sync-" + table`, deliberately WITHOUT the scope,
// and recorded that as a settled decision in the code comment and in the
// roadmap. Measured against `vendor/rxdb.bundle.js` it is data loss:
//
//   this.metaInfoPromise = (async () => {
//     var g = "rx-replication-meta-"
//           + await n.database.hashFunction(
//               [this.collection.name, this.replicationIdentifier].join("-"));
//
// The persisted checkpoint is keyed by `[collection.name, replicationIdentifier]`
// and BY NOTHING ELSE — the scope is not in the key. The pull's returned
// checkpoint is `lastOfArray(data)` → `{id, modified}` of the last row IN THE
// SCOPED RESULT SET, and the next pull ANDs
// `or("_modified".gt.C, and("_modified".eq.C,"id".gt.I))` onto the query. So one
// identifier across scopes means:
//
//   open today's checklist   → rows, checkpoint = today's newest _modified
//   open YESTERDAY's         → every row is <= C → ZERO rows, permanently
//
// The harness below runs the plugin's pull construction TWICE, carrying the
// checkpoint through a meta store keyed exactly the way RxDB keys its own.
// ===========================================================================

const TABLE_BY_KEY = {
  templates: 'checklist_templates',
  checklists: 'checklist_submissions',
  responses: 'submission_responses',
  approvals: 'submission_rejections',
};

// What a device scoped to `NEVER` must receive. Non-empty per collection, so
// "it returned nothing" cannot pass.
const IN_SCOPE_OF_NEVER = {
  templates: [NEVER.templateId],
  checklists: [NEVER.checklistId],
  responses: ['rsp-4', 'rsp-5'],
  approvals: ['apr-2'],
};

async function startCaptured(scope) {
  const { startHQReplication } = await loadClient();
  const captured = {};
  const db = {
    templates: {}, checklists: {}, responses: {}, approvals: {}, conflict_records: {},
  };
  startHQReplication(db, {}, {
    scope,
    waitForLeadership: false,
    replicate: (o) => { captured[o.tableName] = o; return { conflict$: { subscribe() {} } }; },
  });
  return captured;
}

// ONE pull, in the vendored plugin's own order: queryBuilder, THEN the
// checkpoint `.or()`, THEN order(_modified).order(id).limit(batchSize), and the
// next checkpoint is the LAST row of what came back. When nothing comes back
// the plugin returns `undefined` and RxDB keeps the checkpoint it had — which
// is exactly what makes the defect permanent rather than one-shot.
function onePull(opts, rows, checkpoint) {
  const log = [];
  let query = fakeQuery(rows, log).select('*');
  const next = opts.pull.queryBuilder({
    query, lastPulledCheckpoint: checkpoint, batchSize: opts.pull.batchSize,
  });
  if (next) query = next;
  if (checkpoint) {
    const { modified: C, id: I } = checkpoint;
    query = query.or(`"_modified".gt.${C},and("_modified".eq.${C},"id".gt.${I})`);
  }
  query = query
    .order('_modified', { ascending: true })
    .order('id', { ascending: true })
    .limit(opts.pull.batchSize);
  const data = query.rows();
  const last = data[data.length - 1];
  return {
    ids: data.map((r) => r.id),
    checkpoint: last ? { id: last.id, modified: last._modified } : checkpoint,
  };
}

/**
 * One device opening a sequence of checklists. `meta` is keyed the way RxDB
 * keys its checkpoint meta store — `[collection.name, replicationIdentifier]`
 * and nothing else. If the identifier does not carry the scope, the second
 * scope inherits the first scope's checkpoint. That is the whole bug.
 */
async function deviceOpens(scopes, meta = new Map()) {
  const steps = [];
  for (const scope of scopes) {
    // eslint-disable-next-line no-await-in-loop
    const captured = await startCaptured(scope);
    const step = {};
    for (const [key, table] of Object.entries(TABLE_BY_KEY)) {
      const opts = captured[table];
      if (!opts) throw new Error('[scope-harness] no replication started for ' + key);
      const metaKey = [key, opts.replicationIdentifier].join('-');
      const res = onePull(opts, SCOPE_FIXTURE[key], meta.get(metaKey));
      meta.set(metaKey, res.checkpoint);
      step[key] = { ids: res.ids, identifier: opts.replicationIdentifier };
    }
    steps.push(step);
  }
  return steps;
}

const SCOPE_OPEN = {
  checklistId: OPEN.checklistId, templateId: OPEN.templateId, fieldIds: OPEN.fieldIds,
};
const SCOPE_NEVER = {
  checklistId: NEVER.checklistId, templateId: NEVER.templateId, fieldIds: NEVER.fieldIds,
};

test.describe('[SCOPE-02] the checkpoint is per-scope, not per-table (F-1)', () => {
  test('RxDB really keys the checkpoint meta store by [collection.name, replicationIdentifier]', () => {
    // If this stops being true the whole finding changes shape, so read it out
    // of the shipped bundle rather than trusting the analysis.
    const bundle = fs.readFileSync(BUNDLE_PATH, 'utf8');
    expect(bundle).toContain('rx-replication-meta-');
    expect(bundle).toContain('[this.collection.name,this.replicationIdentifier].join("-")');
    // ...and the checkpoint clause the next pull ANDs on.
    expect(bundle).toContain('".gt.');
    expect(bundle).toContain('".eq.');
  });

  test('opening an OLDER checklist after a newer one still replicates its rows', async () => {
    const [first, second] = await deviceOpens([SCOPE_OPEN, SCOPE_NEVER]);

    // Sanity: the first scope really did advance a checkpoint.
    expect(first.responses.ids).toEqual(['rsp-1', 'rsp-2', 'rsp-3']);

    // 🛑 THE FINDING. Before the fix every collection here came back EMPTY —
    // `NEVER`'s rows are all `_modified` older than `OPEN`'s, so the carried
    // checkpoint filtered every one of them out, permanently.
    for (const key of Object.keys(TABLE_BY_KEY)) {
      expect(
        second[key].ids.sort(),
        `${key}: switching to an older checklist replicated nothing — the previous `
        + "scope's checkpoint is still filtering",
      ).toEqual([...IN_SCOPE_OF_NEVER[key]].sort());
    }
  });

  test('...and that is the SAME set a device with a fresh checkpoint would get', async () => {
    // The control the finding names: re-mint the checkpoint and the older
    // checklist's rows come back. If this control ever disagreed with the test
    // above, the test above would be asserting the wrong "full set".
    const [control] = await deviceOpens([SCOPE_NEVER]);
    for (const key of Object.keys(TABLE_BY_KEY)) {
      expect(control[key].ids.sort()).toEqual([...IN_SCOPE_OF_NEVER[key]].sort());
    }
  });

  test('a different scope mints a different replicationIdentifier — per collection', async () => {
    const [first, second] = await deviceOpens([SCOPE_OPEN, SCOPE_NEVER]);
    for (const key of Object.keys(TABLE_BY_KEY)) {
      expect(first[key].identifier, `${key}: identifier is not scoped`)
        .not.toBe(second[key].identifier);
      // Still readable, still prefixed by the table it replicates.
      expect(first[key].identifier).toContain(`hq-sync-${TABLE_BY_KEY[key]}-`);
    }
  });

  test('the SAME scope still RESUMES — the fix did not just disable checkpoints', async () => {
    // The property the original comment was protecting. It survives: identical
    // scope ⇒ identical identifier ⇒ the checkpoint is reused, so re-opening
    // the same checklist re-pulls nothing.
    const meta = new Map();
    const [first, again] = await deviceOpens([SCOPE_OPEN, SCOPE_OPEN], meta);
    for (const key of Object.keys(TABLE_BY_KEY)) {
      expect(first[key].identifier).toBe(again[key].identifier);
      expect(again[key].ids, `${key}: re-opening the same checklist re-pulled rows`)
        .toEqual([]);
    }
  });

  test('the fingerprint is deterministic and scope-sensitive', async () => {
    const { scopeFingerprint, scopePlanFor } = await loadClient();
    expect(scopeFingerprint('abc')).toBe(scopeFingerprint('abc'));
    expect(scopeFingerprint('abc')).not.toBe(scopeFingerprint('abd'));
    expect(scopeFingerprint('abc')).toMatch(/^[0-9a-f]{16}$/);
    // Two scopes differing only in fieldIds must not share a `responses`
    // checkpoint — the emitted filter differs, so the identifier must too.
    const a = scopePlanFor('responses', SCOPE_OPEN);
    const b = scopePlanFor('responses', { ...SCOPE_OPEN, fieldIds: [OPEN.fieldIds[0]] });
    expect(a.fingerprint).not.toBe(b.fingerprint);
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
