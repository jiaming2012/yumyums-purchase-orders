// tests/sync-fill-view.spec.js — THE FILL VIEW READS FROM RxDB.
//
// Card `activate-fill-view-reads` (run 20260808-2, C3). Activity 4.
//
// ===========================================================================
// WHAT THIS FILE PROVES
// ===========================================================================
//
// 1. With `hq_sync_read` ON, an open checklist's field values in the FILL view
//    come out of RxDB — a response row that exists ONLY in the local collection
//    (HQ's REST answer for that submission carries no responses at all) shows up
//    as an answered field, and the runner's progress line counts it.
// 2. With the flag OFF — the default in every environment — the SAME fixture
//    renders the SAME page with the field unanswered, no scope open, no
//    database, no IndexedDB. That is the vacuity pair: (1) is only evidence if
//    (2) fails when the wiring is dead.
// 3. The open/cancel lifecycle: one scope per open checklist, cancelled on
//    close, MULTIPLE LIVE AT ONCE (ledger T-43(c) — crew members work a setup
//    checklist and a food-prep checklist concurrently), and cancelling one does
//    not disturb the other.
// 4. C2's G6 finding F-1: a REJECTED database creation is evicted rather than
//    memoised forever, so one transient IndexedDB failure does not brick sync
//    for the page's lifetime.
//
// ===========================================================================
// 🛑 WHAT IT DOES NOT PROVE, AND WHAT IT MUST NOT BE READ AS
// ===========================================================================
//
// * The SUBSTRATE leg is a `page.route` stub, exactly as C2's
//   `tests/sync-one-row.spec.js` explains at length. HQ Postgres → LISTEN/NOTIFY
//   relay → Supabase → PostgREST is spike C's mechanism and `demo-sync-target`
//   (S2)'s deliverable. Everything DOWNSTREAM of the stub is production code:
//   the vendored supabase-js client, `makeSyncFetch`, `startHQReplication`'s
//   scope plan and `replicationIdentifier`, RxDB's pull handler, the Dexie
//   write, and `workflows.html`'s own read.
// * 🛑 It says NOTHING about the My Checklists list. Ledger T-43(b): that read
//   path is DELIBERATELY OPEN and no card may decide it. The list is rendered by
//   `renderChecklistList()` from the same REST fetch as before, in the flag-on
//   case as well as the flag-off one, and this file asserts that it is.
// * 🛑 It changes no write path. Decision 126 (ledger T-32): RxDB serves READS;
//   `/saveResponse` and `/submitChecklist` keep owning ALL writes.
//   `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops` is untouched
//   (there is no `autoSaveField` — B-65), and `tests/persistence.spec.js` is the
//   suite that guards it.

const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// The flag. One name, defined in `sync-rxdb/bootstrap.js` as SYNC_READ_FLAG;
// spelled here so a rename reds this file rather than silently turning the fill
// view's read path off.
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
// The stub substrate — the transport leg only.
//
// Deliberately a COPY of `tests/sync-one-row.spec.js`'s, not an import: two
// Playwright spec files importing each other's helpers is a coupling that makes
// either one unrunnable alone. The two rules that make it correct are both
// C2's and both re-stated here because getting either wrong silently produces a
// green test that proved nothing:
//
//   * the plugin stops when a batch comes back SHORT of `batchSize`, so one
//     non-empty answer per table is the whole protocol;
//   * 🛑 "has an `or=` param" is NOT the checkpoint test. The fill scope's own
//     filter for `submission_responses` is an `or` node (submitted rows on this
//     checklist OR a draft on one of its fields), so EVERY request for that
//     table carries an `or=`. The plugin's checkpoint clause is the one naming
//     `_modified`.
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

// The substrate's wire shape per `sync-schema/sql/0001_sync_tables.sql`: the flat
// body plus `_deleted` and the trigger-stamped `_modified` pull cursor. The
// plugin strips both back off before the row reaches the collection.
function substrateRow(row) {
  return { ...row, _deleted: false, _modified: '2026-08-08T00:00:00.000000Z' };
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

/**
 * A checklist whose SUBMISSION carries no responses at all, plus the one
 * response row the stubbed substrate will serve for it.
 *
 * 🛑 The submission is what gives the fill scope a real `checklistId` to be
 * scoped BY (`checklist_submissions.id`) — replication is per-open-checklist and
 * is never pulled whole (ledger T-29 decision 105), so a checklist with no
 * submission row has nothing to name and opens no scope. Submitting with
 * `responses: []` is therefore the fixture that makes the RxDB read visible:
 * HQ's own REST answer for this checklist is EMPTY, so an answered field in the
 * runner can only have come from the local collection.
 */
async function checklistWithNoRestResponses(page, name) {
  const { templateId, fieldId } = await createOneFieldTemplate(page, name);
  await api(page, 'POST', 'submitChecklist', {
    template_id: templateId, idempotency_key: uuid(), responses: [],
  });
  const mine = await api(page, 'GET', 'myChecklists');
  const submission = (mine.submissions || []).find((s) => s.template_id === templateId);
  expect(submission, 'the fixture produced a submission to scope to').toBeTruthy();
  expect(
    (submission.responses || []).length,
    'the fixture must carry NO REST responses, or an answered field proves nothing',
  ).toBe(0);
  return { templateId, fieldId, checklistId: submission.id, templateName: name };
}

// ===========================================================================
// [FILL-01] The read.
// ===========================================================================
test.describe('[FILL-01] the fill view reads the open checklist from RxDB', () => {
  test('a response that exists ONLY in RxDB answers the field in the runner', async ({ page }) => {
    test.setTimeout(180000);
    await login(page);
    const name = 'C3 Fill Read ' + Date.now();
    const fx = await checklistWithNoRestResponses(page, name);

    // The row the substrate serves. It has no counterpart in HQ's REST answer.
    await stubSubstrate(page, {
      submission_responses: [substrateRow({
        id: uuid(),
        submission_id: fx.checklistId,
        field_id: fx.fieldId,
        value: true,
        answered_by: uuid(),
        answered_at: '2026-08-08T00:00:00.000Z',
      })],
    });

    await page.goto(`/workflows.html?${FLAG}=on`);
    const row = page.locator(`[data-fill-template-id="${fx.templateId}"]`);
    await expect(row).toBeVisible({ timeout: 30000 });

    // 🛑 T-43(b). The LIST row is rendered from REST and REST says 0 of 1 — the
    // list read path is untouched by this card and must stay that way even with
    // the flag on.
    await expect(row).toContainText('0/1 items');

    await row.click();

    // The runner. Its field state is hydrated from REST first (nothing), then
    // overlaid from the RxDB collection this checklist's scope replicated.
    const progress = page.locator('#fill-body .progress-line');
    await expect(progress).toContainText('1 of 1 items complete', { timeout: 60000 });

    // ...and it really did come out of the collection, not off a REST body.
    const state = await page.evaluate(async (fid) => {
      const db = window.HQSync && window.HQSync.db;
      const doc = db ? await db.responses.findOne({ selector: { field_id: fid } }).exec() : null;
      return {
        db: !!db,
        collectionValue: doc ? doc.value : null,
        openFillIds: window.HQFillSync.openIds(),
        openScopeKeys: window.HQSync.openScopeKeys().length,
      };
    }, fx.fieldId);
    expect(state.db, 'the flag-on fill view created the database').toBe(true);
    expect(state.collectionValue).toBe(true);
    expect(state.openFillIds, 'exactly the open checklist has a live fill scope')
      .toEqual([fx.checklistId]);
    expect(state.openScopeKeys).toBe(1);

    // Closing the checklist cancels its scope. One per open checklist, cancelled
    // on close — ledger T-43(c).
    await page.click('#fill-back');
    await expect(page.locator(`[data-fill-template-id="${fx.templateId}"]`)).toBeVisible();
    await expect
      .poll(async () => page.evaluate(() => window.HQFillSync.openIds().length), { timeout: 15000 })
      .toBe(0);
    const afterClose = await page.evaluate(() => window.HQSync.openScopeKeys().length);
    expect(afterClose).toBe(0);
  });

  // The vacuity check. The assertion above is only evidence if the flag is what
  // produced it — on the SAME fixture, on the SAME page, in the SAME file.
  test('with the flag OFF the same fixture renders unanswered — no scope, no database, no IndexedDB', async ({ page }) => {
    test.setTimeout(180000);
    await login(page);
    const name = 'C3 Fill Read Off ' + Date.now();
    const fx = await checklistWithNoRestResponses(page, name);
    await stubSubstrate(page, {
      submission_responses: [substrateRow({
        id: uuid(),
        submission_id: fx.checklistId,
        field_id: fx.fieldId,
        value: true,
        answered_by: uuid(),
        answered_at: '2026-08-08T00:00:00.000Z',
      })],
    });

    // No flag in the URL, and the flag is OFF by default in every environment.
    await page.goto('/workflows.html');
    const row = page.locator(`[data-fill-template-id="${fx.templateId}"]`);
    await expect(row).toBeVisible({ timeout: 30000 });
    await row.click();
    await expect(page.locator('#fill-body .progress-line')).toContainText('0 of 1 items complete');

    // Give any (incorrect) async database creation a chance to land — C1's B-88
    // guard samples early and is timing-blind (C2's G6 finding F-1 against it).
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
        openFillIds: window.HQFillSync.openIds(),
        names,
      };
    });
    expect(state.readEnabled).toBe(false);
    expect(state.dbUndefined).toBe(true);
    expect(state.openFillIds).toEqual([]);
    expect(state.names).toEqual([]);
  });
});

// ===========================================================================
// [FILL-02] The lifecycle — multiple live fill scopes at once (T-43(c)).
// ===========================================================================
test.describe('[FILL-02] the fill view holds multiple concurrent per-checklist scopes', () => {
  test('setup + food prep live at once, pairwise-distinct identifiers, and cancelling one leaves the other', async ({ page }) => {
    test.setTimeout(120000);
    await login(page);
    await stubSubstrate(page, {});
    await page.goto(`/workflows.html?${FLAG}=on`);
    await page.waitForFunction(
      () => window.HQFillSync && window.HQSync && window.HQSync.readEnabled === true,
      null,
      { timeout: 30000 },
    );

    const out = await page.evaluate(async () => {
      // The operator's own example: a setup checklist and a food-preparation
      // checklist, open at the same time, on one phone.
      const setup = await window.HQFillSync.open(
        'chk-setup-c3', 'tpl-setup-c3', ['fld-setup-1'],
      );
      const foodprep = await window.HQFillSync.open(
        'chk-foodprep-c3', 'tpl-foodprep-c3', ['fld-foodprep-1'],
      );
      const ids = (h) => Object.values(h.states).map((s) => s.replicationIdentifier);
      const res = {
        both: window.HQFillSync.openIds().sort(),
        sharedDatabase: setup.db === foodprep.db,
        idsSetup: ids(setup),
        idsFoodprep: ids(foodprep),
        // Re-opening a checklist already held is a no-op, not a second
        // replication on one Realtime topic.
        sameHandle: (await window.HQFillSync.open(
          'chk-setup-c3', 'tpl-setup-c3', ['fld-setup-1'],
        )) === setup,
      };
      await window.HQFillSync.close('chk-foodprep-c3');
      res.afterClosingOne = window.HQFillSync.openIds();
      await window.HQFillSync.close('chk-setup-c3');
      res.afterClosingBoth = window.HQFillSync.openIds();
      return res;
    });

    expect(out.both).toEqual(['chk-foodprep-c3', 'chk-setup-c3']);
    expect(out.sharedDatabase).toBe(true);
    expect(out.sameHandle).toBe(true);
    expect(out.idsSetup).toHaveLength(4);
    expect(out.idsFoodprep).toHaveLength(4);
    // 🛑 Pairwise distinct. A shared identifier is ONE checkpoint across two open
    // checklists (RxDB keys it by `[collection.name, replicationIdentifier]` and
    // nothing else), which is the permanent-row-loss shape.
    const all = [...out.idsSetup, ...out.idsFoodprep];
    expect(new Set(all).size).toBe(all.length);
    // Cancelling one leaves the other running — they are not a single lifecycle.
    expect(out.afterClosingOne).toEqual(['chk-setup-c3']);
    expect(out.afterClosingBoth).toEqual([]);
  });

  test('the fill scope carries the crew member — two users on one phone do not share a checkpoint [F-2]', async ({ page }) => {
    test.setTimeout(120000);
    await login(page);
    await stubSubstrate(page, {});
    await page.goto(`/workflows.html?${FLAG}=on`);
    await page.waitForFunction(
      () => window.HQFillSync && window.HQSync && window.HQSync.readEnabled === true,
      null,
      { timeout: 30000 },
    );

    const out = await page.evaluate(async () => {
      const scope = { checklistId: 'chk-shared-phone', templateId: 'tpl-shared-phone', fieldIds: ['fld-1'] };
      const a = await window.HQSync.openSyncScope({ ...scope, userId: 'usr-crew-a' });
      const b = await window.HQSync.openSyncScope({ ...scope, userId: 'usr-crew-b' });
      const ids = (h) => Object.values(h.states).map((s) => s.replicationIdentifier);
      const res = {
        distinctHandles: a !== b,
        openCount: window.HQSync.openScopeKeys().length,
        idsA: ids(a),
        idsB: ids(b),
        // ...and a fill scope with no crew member is refused outright.
        refusesAnonymous: await (async () => {
          try {
            await window.HQSync.openSyncScope(scope);
            return null;
          } catch (err) { return String((err && err.message) || err); }
        })(),
      };
      await a.cancel();
      await b.cancel();
      return res;
    });

    expect(out.distinctHandles).toBe(true);
    expect(out.openCount).toBe(2);
    expect(out.idsA.filter((id) => out.idsB.includes(id))).toEqual([]);
    expect(out.refusesAnonymous).toMatch(/userId/);
  });
});

// ===========================================================================
// [FILL-03] C2's G6 finding F-1 — a rejected creation must not be cached.
// ===========================================================================
test.describe('[FILL-03] a transient storage failure does not brick sync until reload (C2 G6 F-1)', () => {
  test('a rejected createDatabase is EVICTED — the next open succeeds, and no dead scope is reported live', async ({ page }) => {
    test.setTimeout(120000);
    await login(page);
    await stubSubstrate(page, {});
    await page.goto(`/workflows.html?${FLAG}=on`);
    await page.waitForFunction(() => window.HQSync && window.HQSync.readEnabled === true, null, { timeout: 30000 });

    const out = await page.evaluate(async () => {
      const scope = { userId: 'usr-f1', checklistId: 'chk-f1', templateId: 'tpl-f1', fieldIds: ['fld-f1'] };
      // 🛑 THE FORCING SEAM. G6 could not force this finding at runtime — Dexie
      // holds its own IndexedDB reference, so undefining `window.indexedDB`
      // resolved anyway, which is why F-1 shipped PLAUSIBLE rather than
      // CONFIRMED. `ensureDatabase()` therefore calls `HQSync.createDatabase`
      // (the property, already exported for exactly this kind of driving) rather
      // than the module-private import, so a test can make the creation fail the
      // way a quota-exhausted phone does. Production never replaces it.
      const real = window.HQSync.createDatabase;
      window.HQSync.createDatabase = () => Promise.reject(new Error('transient IndexedDB failure'));

      let firstError = null;
      try { await window.HQSync.openSyncScope(scope); } catch (err) {
        firstError = String((err && err.message) || err);
      }
      const keysAfterFailure = window.HQSync.openScopeKeys();

      // The storage comes back — a quota freed, a corrupt store recreated.
      window.HQSync.createDatabase = real;
      let secondError = null;
      let recovered = false;
      try {
        const handle = await window.HQSync.openSyncScope(scope);
        recovered = !!(handle && handle.db);
        await handle.cancel();
      } catch (err) { secondError = String((err && err.message) || err); }

      return {
        firstError,
        keysAfterFailure,
        secondError,
        recovered,
        keysAtEnd: window.HQSync.openScopeKeys(),
      };
    });

    expect(out.firstError, 'the failure could not be forced — the createDatabase seam is missing')
      .toMatch(/transient IndexedDB failure/);
    expect(out.keysAfterFailure, 'a scope whose open REJECTED is still reported live by '
      + 'openScopeKeys(), and its cancel() is unreachable').toEqual([]);
    expect(out.secondError, 'the rejected promise was memoised — one transient storage '
      + 'failure bricked sync for the page\'s lifetime').toBe(null);
    expect(out.recovered, 'the retry after a transient failure did not produce a database').toBe(true);
    expect(out.keysAtEnd).toEqual([]);
  });
});
