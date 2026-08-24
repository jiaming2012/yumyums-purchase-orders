// tests/sync-one-row.spec.js — THE WALKING SKELETON, one row, end to end.
//
// Card `skeleton-one-row-end-to-end` (run 20260808-2, C2). Activity 3.
//
// ===========================================================================
// WHAT THIS FILE PROVES, AND WHAT IT DOES NOT
// ===========================================================================
//
// PROVES: one checklist answer written through HQ's REAL write path
// (`POST /api/v1/workflow/saveResponse`, real session cookie, real auth
// middleware, real grant gate, real repository SQL, real Postgres) is read back
// on `/workflows.html` OUT OF AN RxDB COLLECTION — the first production call
// site of `createHQSyncDatabase()` + `startHQReplication()` in this repo's
// history — behind an explicit flag that is OFF by default.
//
// DOES NOT PROVE: that the SUBSTRATE half of the chain works. HQ Postgres →
// (LISTEN/NOTIFY relay) → Supabase substrate → PostgREST is spike C's
// mechanism, proven by `.night-crew/qa/spike-supabase/spike-c-roundtrip.sh`
// (exit-code contract, real containers) and NOT production code yet. Running a
// three-service Supabase stack inside the Playwright suite is `demo-sync-target`
// (S2)'s job — `task demo:sync`, the milestone's close bar.
//
// So this file supplies the relay+substrate leg as a `page.route` STUB, and it
// is a stub in ONE narrow sense only: it is the transport. The ROW IT SERVES IS
// NOT INVENTED — it is read back out of HQ's own `submission_responses` table
// with psql, after `/saveResponse` wrote it, and re-shaped into the sync
// contract exactly as `sql/spike-c-relay-trigger.sql` + spike C's relay do
// (`_modified`, `_deleted`, flat body). If HQ's write path did not persist the
// answer, the query returns nothing and this test fails at step 3, before a
// browser is ever opened.
//
// Everything downstream of the stub is PRODUCTION CODE, unmocked: the vendored
// supabase-js client, `makeSyncFetch`'s path rewriting and credential
// stripping, `startHQReplication`'s scope plan and `replicationIdentifier`,
// RxDB's pull handler, the Dexie/IndexedDB write, and the page's own read.
//
// ===========================================================================
// 🛑 THE FLAG IS OFF BY DEFAULT AND THAT IS C1's CONTRACT, NOT A CONVENIENCE
// ===========================================================================
// `tests/sync-rxdb-client.spec.js`'s B-88 guard asserts that at end of load
// `window.HQSync.db` is `undefined` AND no `/rxdb|hq_sync/i` IndexedDB database
// exists. This card lands the code that would violate it, so the OFF case is
// asserted here too, from the other side: the vacuity check below proves the
// flag actually does something, so the OFF assertion is not passing because the
// wiring is dead.

const { test, expect } = require('@playwright/test');
const { execFileSync } = require('child_process');
const { resolveE2eDb } = require('../scripts/reset-e2e-db');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// The flag. One name, defined in `sync-rxdb/bootstrap.js`; spelled here so a
// rename reds this file rather than silently turning the skeleton off.
const FLAG = 'hq_sync_read';

async function login(page) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL((url) => !url.pathname.includes('login'));
}

async function api(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch('/api/v1/workflow/' + p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path, body]);
}

function uuid() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : ((r & 0x3) | 0x8);
    return v.toString(16);
  });
}

// ---------------------------------------------------------------------------
// THE RELAY, IN MINIATURE.
//
// 🛑 COORDINATES COME FROM `scripts/reset-e2e-db.js` AND NOWHERE ELSE. That
// helper is the one place the e2e Postgres coordinates are computed — :5434,
// role `hqtest`, the TEST-ONLY container (`docker-compose.test.yml`). It is
// NEVER :5433 (yumyums-dev-pg, which serves https://hq.yumyums.kitchen — ledger
// decision 155, B-141/B-143) and never :5432 (infra-postgres-1). This function
// only ever SELECTs; it issues no DDL and no writes of any kind.
//
// `psqlUrl`, not `testUrl`: the latter carries `TimeZone`, which pgx understands
// and libpq does not.
// ---------------------------------------------------------------------------
function readResponseRowFromHQ(fieldId) {
  const db = resolveE2eDb();
  const sql = `
    select coalesce(json_agg(row_to_json(r)), '[]'::json)::text
    from (
      select id::text            as id,
             submission_id::text as submission_id,
             field_id::text      as field_id,
             value               as value,
             answered_by::text   as answered_by,
             to_char(answered_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.USZ') as answered_at
      from submission_responses
      where field_id = '${fieldId}'
    ) r`;
  const out = execFileSync('psql', [db.psqlUrl, '-At', '-v', 'ON_ERROR_STOP=1', '-c', sql], {
    encoding: 'utf8',
  });
  return JSON.parse(out.trim());
}

// The substrate's wire shape, per `sync-schema/sql/0001_sync_tables.sql`:
// a flat body plus `_deleted` and the trigger-stamped `_modified` pull cursor.
// The plugin strips both back off before the row reaches the collection.
function toSubstrateRow(row) {
  return Object.assign({}, row, { _deleted: false, _modified: '2026-08-08T00:00:00.000000Z' });
}

// ---------------------------------------------------------------------------
// The stub substrate. PostgREST answers a `GET` with a JSON array; the plugin
// asks for one table at a time and stops when a batch comes back short of
// `batchSize`, so a single non-empty answer per table is the whole protocol.
//
// The checkpoint clause is honoured rather than ignored: a second pull for a
// table already served answers `[]`, which is what a real substrate with no
// newer rows does. Ignoring it would loop.
//
// 🛑 "Has an `or=` param" is NOT the checkpoint test, and getting that wrong
// makes this stub answer `[]` to the FIRST pull as well. The fill scope's own
// filter for `submission_responses` is an `or` node (client.js `scopeFilterFor`:
// submitted rows on this checklist OR a draft on one of its fields), so every
// request for that table carries an `or=`. The plugin's checkpoint clause is the
// one naming `_modified` — `or=("_modified".gt.<ts>,and("_modified".eq.…))`.
// ---------------------------------------------------------------------------
async function stubSubstrate(page, rowsByTable) {
  const served = new Set();
  await page.route('**/sync/rest/**', async (route) => {
    const url = new URL(route.request().url());
    const table = url.pathname.split('/').filter(Boolean).pop();
    const resumed = url.searchParams.getAll('or').some((v) => v.includes('_modified'));
    const rows = (!resumed && !served.has(table) && rowsByTable[table]) || [];
    served.add(table);
    await route.fulfill({
      status: 200,
      headers: {
        'Content-Type': 'application/json; charset=utf-8',
        'Content-Range': `0-${Math.max(rows.length - 1, 0)}/*`,
      },
      body: JSON.stringify(rows),
    });
  });
}

// A template with exactly one checkbox field, assigned + scheduled for today.
async function createOneFieldTemplate(page, name) {
  const dow = await page.evaluate(() => new Date().getDay());
  const tpl = await api(page, 'POST', 'createTemplate', {
    name,
    requires_approval: false,
    sections: [{
      title: 'Section 1',
      order: 0,
      condition: null,
      fields: [{
        type: 'checkbox', label: 'Check this', required: false,
        order: 0, config: {}, fail_trigger: null, condition: null,
      }],
    }],
    schedules: [{ active_days: [dow] }],
    assignments: [
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
    ],
  });
  const templates = await api(page, 'GET', 'templates');
  const found = templates.find((t) => t.id === tpl.id);
  return { templateId: tpl.id, fieldId: found.sections[0].fields[0].id };
}

test.describe('skeleton — one row, real write path to an RxDB-served read', () => {
  test('a value written by POST /saveResponse is served to the page out of RxDB, behind the flag', async ({ page }) => {
    test.setTimeout(120000);
    await login(page);

    // ── 1. A real template, a real field. ─────────────────────────────────
    const { templateId, fieldId } = await createOneFieldTemplate(page, 'C2 One Row ' + Date.now());

    // ── 2. THE REAL WRITE PATH. Not a fixture, not a seeded row. ──────────
    await api(page, 'POST', 'saveResponse', { field_id: fieldId, value: true });

    // Submitting moves the draft (`submission_id IS NULL`) onto the submission
    // (repository.go, "Move draft responses"), which is what gives the fill
    // scope a real `checklistId` to be scoped BY. The row's id, value,
    // answered_by and answered_at are the ones /saveResponse wrote.
    await api(page, 'POST', 'submitChecklist', {
      template_id: templateId, idempotency_key: uuid(), responses: [],
    });
    const mine = await api(page, 'GET', 'myChecklists');
    const submission = (mine.submissions || []).find((s) => s.template_id === templateId);
    expect(submission, 'the real write path produced a submission to scope to').toBeTruthy();
    const checklistId = submission.id;

    // ── 3. The relay leg: read HQ's OWN row back. ─────────────────────────
    const hqRows = readResponseRowFromHQ(fieldId);
    expect(hqRows.length, 'POST /saveResponse persisted exactly one response row').toBe(1);
    const hqRow = hqRows[0];
    expect(hqRow.value, 'the persisted value is the one the write path was given').toBe(true);
    expect(hqRow.submission_id).toBe(checklistId);

    // ── 4. The substrate, stubbed at the transport only. ──────────────────
    await stubSubstrate(page, { submission_responses: [toSubstrateRow(hqRow)] });

    // ── 5. The flag ON, and the dev surface's scope, both explicit. ───────
    // 🛑 `hq_sync_user` is REQUIRED since card `activate-fill-view-reads` (C3) —
    // this card's own G6 finding F-2. The fill scope's persisted checkpoint had
    // no crew member in its key, so `normalizeScope` now refuses a fill scope
    // without one. `answered_by` on the row HQ's write path just wrote IS this
    // crew member, so it is the honest value to scope by rather than a literal.
    await page.goto(
      `/workflows.html?${FLAG}=on`
      + `&hq_sync_checklist=${checklistId}`
      + `&hq_sync_template=${templateId}`
      + `&hq_sync_field=${fieldId}`
      + `&hq_sync_user=${hqRow.answered_by}`,
    );

    const surface = page.locator('#sync-one-row');
    await expect(surface).toBeVisible({ timeout: 30000 });
    await expect(surface).toHaveAttribute('data-state', 'served', { timeout: 60000 });

    // ── 6. The row on screen is the row HQ wrote. ─────────────────────────
    await expect(page.locator('#sync-one-row-id')).toHaveText(hqRow.id);
    await expect(page.locator('#sync-one-row-field')).toHaveText(fieldId);
    await expect(page.locator('#sync-one-row-value')).toHaveText('true');
    // Named on the element, so "where did this come from" is answerable from
    // the DOM alone and cannot drift from the code that filled it.
    await expect(surface).toHaveAttribute('data-source', 'rxdb');

    // ── 7. And it really came out of the collection, not off a REST body. ─
    const fromCollection = await page.evaluate(async (fid) => {
      const db = window.HQSync && window.HQSync.db;
      if (!db) return { db: false };
      const doc = await db.responses.findOne({ selector: { field_id: fid } }).exec();
      return { db: true, id: doc && doc.id, value: doc && doc.value };
    }, fieldId);
    expect(fromCollection.db, 'the flag-on path created the database').toBe(true);
    expect(fromCollection.id).toBe(hqRow.id);
    expect(fromCollection.value).toBe(true);
  });

  // The vacuity check for the OFF case. C1's B-88 guard asserts the OFF state;
  // it is only meaningful if the ON state is reachable, which the test above
  // establishes. This asserts the pair on ONE tree, in ONE file, so a future
  // change that quietly kills the wiring cannot leave the OFF assertion passing
  // for the wrong reason.
  test('the flag is OFF by default — no database, no IndexedDB, and the surface stays absent', async ({ page }) => {
    await login(page);
    await page.goto('/workflows.html');
    await page.waitForFunction(() => window.HQSync !== undefined, null, { timeout: 15000 });
    // Give any (incorrect) async database creation a chance to land — G6-F1's
    // finding against C1's guard is that sampling early is timing-blind.
    await page.waitForTimeout(2000);

    const state = await page.evaluate(async () => {
      let names = [];
      if (typeof indexedDB !== 'undefined' && typeof indexedDB.databases === 'function') {
        names = (await indexedDB.databases()).map((d) => d.name)
          .filter((n) => /rxdb|hq_sync/i.test(n || ''));
      }
      return {
        readEnabled: window.HQSync.readEnabled,
        dbUndefined: window.HQSync.db === undefined,
        names,
      };
    });
    expect(state.readEnabled).toBe(false);
    expect(state.dbUndefined).toBe(true);
    expect(state.names).toEqual([]);
    await expect(page.locator('#sync-one-row')).toBeHidden();
  });

  // Decision 105 / T-43(c): multiple live per-checklist fill replications at
  // once ARE the design (crew members work a setup checklist and a food-prep
  // checklist concurrently). C3 builds the fill view on this call site, so the
  // shape it needs is asserted HERE, at the call site, before C3 depends on it:
  // two DIFFERENT scopes are two handles with disjoint replication identifiers;
  // the SAME scope twice is one handle, not two live replications on one topic.
  test('the call site holds multiple concurrent per-checklist scopes, and is idempotent per scope', async ({ page }) => {
    test.setTimeout(120000);
    await login(page);
    await stubSubstrate(page, {});
    await page.goto(`/workflows.html?${FLAG}=on`);
    await page.waitForFunction(() => window.HQSync && window.HQSync.readEnabled === true, null, { timeout: 15000 });

    const out = await page.evaluate(async () => {
      // `userId` REQUIRED since card `activate-fill-view-reads` (C3) — G6 F-2.
      const a = {
        userId: 'usr-aaa', checklistId: 'chk-aaa', templateId: 'tpl-aaa', fieldIds: ['fld-a1'],
      };
      const b = {
        userId: 'usr-aaa', checklistId: 'chk-bbb', templateId: 'tpl-bbb', fieldIds: ['fld-b1'],
      };
      const ha = await window.HQSync.openSyncScope(a);
      const hb = await window.HQSync.openSyncScope(b);
      const haAgain = await window.HQSync.openSyncScope({ ...a });
      const ids = (h) => Object.values(h.states).map((s) => s.replicationIdentifier).sort();
      const res = {
        sameHandleForSameScope: ha === haAgain,
        distinctHandles: ha !== hb,
        openCount: window.HQSync.openScopeKeys().length,
        sharedDatabase: ha.db === hb.db,
        idsA: ids(ha),
        idsB: ids(hb),
      };
      await hb.cancel();
      res.openCountAfterCancel = window.HQSync.openScopeKeys().length;
      await ha.cancel();
      return res;
    });

    expect(out.sameHandleForSameScope).toBe(true);
    expect(out.distinctHandles).toBe(true);
    expect(out.openCount).toBe(2);
    expect(out.sharedDatabase).toBe(true);
    expect(out.idsA).toHaveLength(4);
    expect(out.idsB).toHaveLength(4);
    // Pairwise distinct — a shared identifier is one checkpoint across two
    // scopes, which is G6 F-1's permanent-row-loss shape.
    expect(out.idsA.filter((id) => out.idsB.includes(id))).toEqual([]);
    expect(out.openCountAfterCancel).toBe(1);
  });
});
