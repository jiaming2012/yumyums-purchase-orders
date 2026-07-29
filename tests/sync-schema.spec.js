// tests/sync-schema.spec.js — the schema contract for the RxDB sync collections.
//
// Card `sync-rxdb-collections-and-table-contract` (overnight-20260729-2, B1).
//
// WHY THIS SUITE NEEDS NO BROWSER, NO DATABASE AND NO RxDB RUNTIME.
// The card is schema ONLY — it wires no replication, writes no policy and
// constructs no client. The collection definitions are therefore declared as
// plain data (JSON-Schema objects in `sync-schema/collections.js`) and the SQL
// contract as a plain `.sql` file, and both can be proven by reading them. That
// is deliberate: pulling in the `rxdb` package to assert the shape of an object
// literal would add a dependency this card does not otherwise need, and would
// make "schema only" quietly false.
//
// WHAT THE NEGATIVE TESTS ARE FOR. Two of these assertions are the point of the
// card, not decoration:
//
//   * `_modified` must NOT be declared (ledger decision 78). Declaring it pulls
//     `_modified` into `addDocEqualityToQuery`'s compare-and-swap, so ANY
//     server-side touch becomes a conflict — including ones where no answer
//     changed — routing ordinary bookkeeping into the conflict-notice UI's
//     "a change we couldn't identify" row, the one row from which nothing can be
//     recovered.
//   * `_deleted` must NOT be declared either. RxDB owns that field; the Supabase
//     replication plugin maps the Postgres column onto RxDB's internal deleted
//     flag. Declaring it collides with the plugin's own handling.
//
// A later card adding either field back is the failure this file exists to stop.
// Both assertions are written so that ADDING the field reds the suite — proven
// by mutation before this file was committed green, not assumed.

const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');
const { pathToFileURL } = require('url');

const REPO_ROOT = path.resolve(__dirname, '..');
const MODULE_PATH = path.join(REPO_ROOT, 'sync-schema', 'collections.js');
const SQL_PATH = path.join(REPO_ROOT, 'sync-schema', 'sql', '0001_sync_tables.sql');

// The module is ESM (a browser will eventually `import` it alongside
// vendor/rxdb.bundle.js, which is itself ESM). Playwright specs are CJS, so it
// is reached with a dynamic import — which works from CJS on Node 20.
//
// Loaded INSIDE each test rather than at file scope on purpose: a top-level
// await that throws makes the whole file fail to load, and a spec file that
// reports zero executed tests is this repo's signature silent failure (B-09,
// B-14, B-16). Per-test loading makes a missing module red as N real test
// failures with a real count.
async function loadModule() {
  return import(pathToFileURL(MODULE_PATH).href);
}

function readSql() {
  return fs.readFileSync(SQL_PATH, 'utf8');
}

// The negative SQL assertions below must read STATEMENTS, not prose. The file's
// comments legitimately name the things the file does not do ("no policies",
// "no lamport_ts", "no table for the overwritten-answer record") — that is the
// record of the decision and the most useful thing in the file. Asserting
// against the raw text would force those sentences to be written in code, which
// is how a decision stops being written down. Stripping `--` line comments
// leaves the assertion pointed at what actually executes.
//
// `--` never appears inside the `$$ … $$` bodies in this file (they are three
// lines each, checked), so a naive strip is safe here.
function readSqlStatements() {
  return readSql().replace(/--[^\n]*/g, '').toLowerCase();
}

// ---------------------------------------------------------------------------
// The contract, spelled out here rather than read from the module.
//
// A test that derives its expectations from the artefact under test proves
// nothing — it passes for any artefact. These four rows are the card's text:
// "the RxDB collections for checklists, templates, responses, approvals
// mirroring the current Postgres domain model".
//
// `actor`/`when` are decision 79 — replicated rows CARRY who-and-when, so the
// conflict sheet's "Dana M., 6:12 PM" is real rather than "someone else".
// ---------------------------------------------------------------------------
const REPLICATED = [
  { key: 'templates', table: 'checklist_templates', actor: 'updated_by', when: 'updated_at' },
  { key: 'checklists', table: 'checklist_submissions', actor: 'updated_by', when: 'updated_at' },
  { key: 'responses', table: 'submission_responses', actor: 'answered_by', when: 'answered_at' },
  { key: 'approvals', table: 'submission_rejections', actor: 'rejected_by', when: 'rejected_at' },
];

const LOCAL_KEY = 'conflict_records';

// RxDB owns `_deleted`; decision 78 keeps `_modified` out. Neither may appear in
// any declared schema.
const MUST_NOT_DECLARE = ['_deleted', '_modified'];

// The retired op-log sync layer stamped these onto three of the four mirrored
// tables. They are substrate of the mechanism being REPLACED and must not be
// carried into the RxDB model, where the checkpoint is `_modified` (undeclared,
// server-side) and ordering is the replication protocol's business.
const MUST_NOT_CARRY = ['lamport_ts', 'device_id', 'op_type'];

test.describe('sync schema — collection definitions', () => {
  // Anti-vacuous guard. Every data-driven assertion below iterates REPLICATED;
  // if that list were ever emptied, the whole describe block would pass having
  // asserted nothing. Exit 0 on zero executed assertions is the failure this
  // repo keeps rediscovering.
  test('the fixture list is non-empty and names the four collections the card owns', () => {
    expect(REPLICATED.length).toBe(4);
    expect(REPLICATED.map((c) => c.key).sort())
      .toEqual(['approvals', 'checklists', 'responses', 'templates']);
  });

  test('the module exports exactly the four replicated collections and one local collection', async () => {
    const mod = await loadModule();
    expect(Object.keys(mod.REPLICATED_COLLECTIONS).sort())
      .toEqual(['approvals', 'checklists', 'responses', 'templates']);
    expect(Object.keys(mod.LOCAL_COLLECTIONS)).toEqual([LOCAL_KEY]);
  });

  for (const c of REPLICATED) {
    test(`${c.key} — satisfies the RxDB schema contract and maps to ${c.table}`, async () => {
      const mod = await loadModule();
      const def = mod.REPLICATED_COLLECTIONS[c.key];
      expect(def, `collection "${c.key}" is not declared`).toBeTruthy();

      // The Supabase replication plugin is given `tableName`; the mapping from
      // the product-facing collection name to HQ's table name is declared, not
      // inferred, so B2's RLS work can name the same table without guessing.
      expect(def.table).toBe(c.table);
      expect(def.replicated).toBe(true);

      const s = def.schema;
      expect(s.version).toBe(0);
      expect(s.primaryKey).toBe('id');
      expect(s.type).toBe('object');
      expect(typeof s.properties).toBe('object');

      // RxDB requires a maxLength on a string primary key (it sizes the index).
      expect(s.properties.id.type).toBe('string');
      expect(typeof s.properties.id.maxLength).toBe('number');
      expect(s.properties.id.maxLength).toBeGreaterThan(0);

      expect(Array.isArray(s.required)).toBe(true);
      expect(s.required).toContain('id');
      for (const r of s.required) {
        expect(Object.keys(s.properties), `required field "${r}" is not declared`).toContain(r);
      }
    });

    // ---- decision 78 / RxDB reserved fields — THE NEGATIVE TEST -------------
    test(`${c.key} — does NOT declare _modified or _deleted (decision 78)`, async () => {
      const mod = await loadModule();
      const props = Object.keys(mod.REPLICATED_COLLECTIONS[c.key].schema.properties);
      for (const forbidden of MUST_NOT_DECLARE) {
        expect(
          props,
          `"${forbidden}" must not be declared on "${c.key}" — see ledger decision 78`,
        ).not.toContain(forbidden);
      }
      // Also assert it is not smuggled in via `required` or an index.
      expect(mod.REPLICATED_COLLECTIONS[c.key].schema.required || [])
        .not.toContain('_modified');
      expect(mod.REPLICATED_COLLECTIONS[c.key].schema.indexes || [])
        .not.toContain('_modified');
    });

    // ---- decision 79 -------------------------------------------------------
    test(`${c.key} — carries who-and-when (${c.actor} + ${c.when}) (decision 79)`, async () => {
      const mod = await loadModule();
      const s = mod.REPLICATED_COLLECTIONS[c.key].schema;
      expect(Object.keys(s.properties)).toContain(c.actor);
      expect(Object.keys(s.properties)).toContain(c.when);
      expect(s.required).toContain(c.actor);
      expect(s.required).toContain(c.when);
      expect(s.properties[c.when].format).toBe('date-time');
    });

    test(`${c.key} — carries none of the retired op-log columns`, async () => {
      const mod = await loadModule();
      const props = Object.keys(mod.REPLICATED_COLLECTIONS[c.key].schema.properties);
      for (const dead of MUST_NOT_CARRY) {
        expect(props, `"${dead}" belongs to the op-log layer being retired`).not.toContain(dead);
      }
    });
  }
});

// ---------------------------------------------------------------------------
// The conflict record — decision 89. Personal undo, per device, LOCAL ONLY.
// ---------------------------------------------------------------------------
test.describe('sync schema — the local conflict record (decision 89)', () => {
  test('is declared local, replication-ready in shape, and carries no table mapping', async () => {
    const mod = await loadModule();
    const def = mod.LOCAL_COLLECTIONS[LOCAL_KEY];
    expect(def, 'the conflict-record collection is not declared').toBeTruthy();

    // No server table, no endpoint, no replication OF THIS COLLECTION.
    expect(def.replicated).toBe(false);
    expect(def.local).toBe(true);
    expect(def.table).toBeUndefined();

    const props = Object.keys(def.schema.properties);
    // Shape declared replication-ready: promoting this later is adding a table
    // and a policy, not a redesign.
    for (const f of ['submission_id', 'field_id', 'discarded_value',
      'overwritten_by', 'overwritten_at']) {
      expect(props, `the conflict record must carry "${f}"`).toContain(f);
    }
    expect(def.schema.primaryKey).toBe('id');
    for (const forbidden of MUST_NOT_DECLARE) {
      expect(props).not.toContain(forbidden);
    }
  });

  test('retention is ONE named constant, and the number 30 is not scattered', async () => {
    const mod = await loadModule();
    expect(typeof mod.CONFLICT_RECORD_RETENTION_DAYS).toBe('number');
    expect(mod.CONFLICT_RECORD_RETENTION_DAYS).toBe(30);

    // The number is REOPENED — it belongs to
    // `sync-rxdb-conflict-notice-mockup-amendments` and may change at triage.
    // It must therefore be changeable in exactly one place. Counting the bare
    // literal in the source is the only check that actually enforces that.
    const src = fs.readFileSync(MODULE_PATH, 'utf8');
    const bare = src.match(/(?<![\w.])30(?![\w.])/g) || [];
    expect(bare.length, `the literal 30 appears ${bare.length} times in collections.js; it must appear exactly once`)
      .toBe(1);
  });
});

// ---------------------------------------------------------------------------
// The self-hosted per-table SQL contract (feasibility spike W1, six items).
// ---------------------------------------------------------------------------
test.describe('sync schema — the self-hosted per-table SQL contract', () => {
  test('the SQL file exists and creates exactly the four replicated tables', () => {
    const sql = readSql().toLowerCase();
    const created = [...sql.matchAll(/create\s+table\s+(?:if\s+not\s+exists\s+)?public\.([a-z_]+)/g)]
      .map((m) => m[1]).sort();
    expect(created).toEqual(REPLICATED.map((c) => c.table).sort());
  });

  for (const c of REPLICATED) {
    test(`${c.table} — carries all six items of the self-hosted table contract`, () => {
      const sql = readSql().toLowerCase();

      // 1-3: the three replication-contract columns, inside THIS table's block.
      const block = sql.match(
        new RegExp(`create\\s+table\\s+(?:if\\s+not\\s+exists\\s+)?public\\.${c.table}\\s*\\(([\\s\\S]*?)\\n\\);`),
      );
      expect(block, `no create-table block found for ${c.table}`).toBeTruthy();
      const cols = block[1];
      expect(cols).toMatch(/id\s+text\s+primary key/);
      expect(cols).toMatch(/_deleted\s+boolean\s+not null\s+default false/);
      expect(cols).toMatch(/_modified\s+timestamptz\s+not null\s+default now\(\)/);

      // 4: the BEFORE INSERT OR UPDATE trigger that stamps _modified server-side.
      expect(sql).toMatch(
        new RegExp(`before\\s+insert\\s+or\\s+update\\s+on\\s+public\\.${c.table}`),
      );

      // 5: RLS enabled, anon revoked, authenticated granted.
      expect(sql).toContain(`alter table public.${c.table} enable row level security`);
      expect(sql).toMatch(new RegExp(`revoke\\s+all\\s+on\\s+public\\.${c.table}\\s+from\\s+anon`));
      expect(sql).toMatch(new RegExp(`grant[\\s\\S]{0,60}on\\s+public\\.${c.table}\\s+to\\s+authenticated`));

      // 6: Realtime enrolment + the pre-image Realtime needs to evaluate RLS.
      expect(sql).toMatch(
        new RegExp(`alter\\s+publication\\s+supabase_realtime\\s+add\\s+table\\s+public\\.${c.table}`),
      );
      expect(sql).toContain(`alter table public.${c.table} replica identity full`);
    });
  }

  test('writes NO policy — RLS predicates belong to sync-rxdb-row-visibility-rls', () => {
    // RLS ENABLED with zero policies is deny-all, which is the correct state
    // until B2 ports ResolveEntityAccess. A permissive policy landing here would
    // silently open the door that card exists to guard.
    expect(readSqlStatements()).not.toMatch(/create\s+policy/);
  });

  test('declares NO table for the overwritten-answer record (decision 89)', () => {
    // Personal undo, per device, local-only. The absence of a table here IS the
    // decision — see sync-schema/collections.js.
    const sql = readSqlStatements();
    expect(sql).not.toContain('conflict_record');
    expect(sql).not.toContain('discarded_value');
  });

  test('carries no lamport_ts column — the op-log layer is being retired', () => {
    expect(readSqlStatements()).not.toContain('lamport_ts');
  });
});
