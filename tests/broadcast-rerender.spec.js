// ─── EDIT-PROPAGATION BROADCAST RE-RENDER (Activity 5, W-2) ──────────────────
// FR-4 / FR-5, INV-3 / INV-6, C5. Covers three sub-behaviors of the
// editprop-broadcast-rerender card:
//
//   A) A live SAVE_TEMPLATE op re-renders the OPEN unsubmitted checklist to the
//      template's new shape WITHOUT a reload — a surviving field keeps its
//      answer (stable id → hydrateFieldState), a newly-added field renders empty.
//   B) The silent gate: a catch-up replay (silent=true) surfaces NO toast; a
//      genuinely live edit (silent=false) may surface one (42eeb39 no-toast rule).
//   D) A schedule edit that drops TODAY removes the open checklist LIVE from the
//      device (C5 warned-live-removal — honoring half).
//
// RED on the pre-change frontend: applyOp's SAVE_TEMPLATE branch only calls
// loadTemplates() (Builder list); the open runner is never re-fetched/re-rendered,
// so (A) the new field never appears without a reload, (B) the live edit shows no
// toast, and (D) the runner stays open showing the dropped checklist.

const { test, expect } = require('@playwright/test');

const BASE = '';
const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

async function login(page, email, password) {
  await page.goto(BASE + '/login.html');
  await page.fill('input[type="email"]', email || ADMIN_EMAIL);
  await page.fill('input[type="password"]', password || ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

async function apiCall(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch('/api/v1/workflow/' + p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path, body]);
}

async function cleanupTemplates(page) {
  const templates = await apiCall(page, 'GET', 'templates');
  if (!Array.isArray(templates)) return;
  for (const t of templates) await apiCall(page, 'DELETE', 'archiveTemplate/' + t.id);
}

const getDOW = (page) => page.evaluate(() => new Date().getDay());

const CHECKBOX = (label, order, id) => ({
  ...(id ? { id } : {}), type: 'checkbox', label, required: false, order,
  config: null, fail_trigger: null, condition: null,
});

async function openRunner(page) {
  await page.click('[data-fill-template-id]');
  await page.waitForSelector('.check-btn', { timeout: 10000 });
}

test.describe('editprop broadcast re-render', () => {
  test('A) live SAVE_TEMPLATE re-renders open runner: surviving answer kept, new field empty', async ({ browser, page }) => {
    test.setTimeout(90000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getDOW(page);

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Rerender Checklist', requires_approval: false,
      sections: [{ title: 'Closing', order: 0, condition: null, fields: [
        CHECKBOX('Wipe counters', 0), CHECKBOX('Check fridge temps', 1),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Device B (crew) opens the checklist and checks the SURVIVING field.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Rerender Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(pageB);
    const wipeB = pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    await wipeB.click();
    await expect(wipeB).toHaveClass(/checked/, { timeout: 5000 });
    await pageB.waitForTimeout(1800); // auto-save

    // Admin edits the template: keep both fields (with ids) + ADD a new one.
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Rerender Checklist');
    const wipe = tpl.sections[0].fields.find(f => f.label === 'Wipe counters');
    const fridge = tpl.sections[0].fields.find(f => f.label === 'Check fridge temps');
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Rerender Checklist', requires_approval: false,
      sections: [{ title: 'Closing', order: 0, condition: null, fields: [
        CHECKBOX('Wipe counters', 0, wipe.id),
        CHECKBOX('Check fridge temps', 1, fridge.id),
        CHECKBOX('Restock napkins', 2),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Device B did NOT reload. The new field must appear LIVE and the surviving
    // field must still be checked.
    await expect(pageB.locator('.fill-field', { hasText: 'Restock napkins' })).toBeVisible({ timeout: 10000 });
    const wipeAfter = pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    await expect(wipeAfter).toHaveClass(/checked/, { timeout: 5000 });
    const napkinsBtn = pageB.locator('.fill-field', { hasText: 'Restock napkins' }).locator('.check-btn');
    await expect(napkinsBtn).not.toHaveClass(/checked/);

    await ctxB.close();
  });

  test('B) silent gate: catch-up replay shows no toast, live edit surfaces one', async ({ page }) => {
    test.setTimeout(60000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getDOW(page);

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Silent Checklist', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [CHECKBOX('Task one', 0)] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });
    const templates = await apiCall(page, 'GET', 'templates');
    const tplId = templates.find(t => t.name === 'Silent Checklist').id;

    await page.reload();
    await expect(page.locator('#s1').getByText('Silent Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(page);

    const fakeOp = (silent) => page.evaluate(([tid]) => {
      window.applyOp({
        op_type: 'SAVE_TEMPLATE', device_id: 'ghost-device', user_id: 'ghost',
        entity_id: tid, payload: { template_id: tid },
        server_ts: new Date().toISOString(), lamport_ts: 999999,
      }, window.__silent);
    }, [tplId]);

    // Catch-up replay (silent=true) → NO toast.
    await page.evaluate(() => { window.__silent = true; });
    await fakeOp();
    await page.waitForTimeout(1500);
    expect(await page.locator('#sync-toast.show').count()).toBe(0);

    // Genuinely live edit (silent=false) → toast surfaces.
    await page.evaluate(() => { window.__silent = false; });
    await fakeOp();
    await expect(page.locator('#sync-toast.show')).toBeVisible({ timeout: 4000 });
    await expect(page.locator('#sync-toast')).toHaveText(/updated/i);
  });

  test('D) schedule drops today: open runner is removed live from the device', async ({ browser, page }) => {
    test.setTimeout(90000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getDOW(page);
    const notToday = (dow + 3) % 7;

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Drop Today Checklist', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [CHECKBOX('Prep', 0)] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Device B (crew) opens the checklist.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Drop Today Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(pageB);
    await expect(pageB.locator('.check-btn')).toHaveCount(1);

    // Admin drops TODAY from the schedule (keeps the field).
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Drop Today Checklist');
    const prep = tpl.sections[0].fields.find(f => f.label === 'Prep');
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Drop Today Checklist', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [CHECKBOX('Prep', 0, prep.id)] }],
      schedules: [{ active_days: [notToday] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Device B (no reload): the open runner is removed live — no check button
    // remains and the checklist is gone from today's list.
    await expect(pageB.locator('.check-btn')).toHaveCount(0, { timeout: 10000 });
    await expect(pageB.locator('#s1').getByText('Drop Today Checklist')).toHaveCount(0);

    await ctxB.close();
  });
});

// ─── INV-6: Builder discard warning naming the crew count ─────────────────────
// Before a save that CUTS a field (or a schedule change that DROPS today) while
// crew have unsubmitted answers today, the Builder warns the admin naming the
// count and proceeds only on explicit confirm (the INV-1 "loss is only ever a
// warned operator action" branch). RED on the pre-change build: there is no
// draftHolderCount endpoint and no warning, so #save-btn saves silently and no
// dialog is ever surfaced.

// Open the Builder editor for a template and cut/edit its state in place, then
// wire a dialog capturer, click Save, and return the dialog message (or null).
async function saveViaBuilderCapturingDialog(page, tplId, mutate) {
  await page.click('#t3'); // Builder tab
  await page.waitForTimeout(500);
  await page.evaluate((id) => { window.__loaded = window.loadTemplates(); }, tplId);
  await page.evaluate(async (id) => {
    await window.__loaded;
    window.openEditor(id);
  }, tplId);
  await page.waitForSelector('#save-btn', { timeout: 10000 });
  await page.evaluate(mutate);
  await page.evaluate(() => { window.renderBuilder(); });
  await page.waitForSelector('#save-btn', { timeout: 10000 });

  let dialogMsg = null;
  page.once('dialog', async (d) => { dialogMsg = d.message(); await d.accept(); });
  await page.click('#save-btn');
  await page.waitForTimeout(2500); // count round-trip + dialog + save
  return dialogMsg;
}

test.describe('INV-6 builder discard warning', () => {
  test('E) cutting a field with a crew draft warns naming the count, confirm discards', async ({ page }) => {
    test.setTimeout(90000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getDOW(page);

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Cut Warn Checklist', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [
        CHECKBOX('Field A', 0), CHECKBOX('Field B', 1),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Cut Warn Checklist');
    const fieldA = tpl.sections[0].fields.find(f => f.label === 'Field A');

    // A crew member leaves an unsubmitted draft on Field A (open runner + check).
    await page.reload();
    await expect(page.locator('#s1').getByText('Cut Warn Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(page);
    await page.locator('.fill-field', { hasText: 'Field A' }).locator('.check-btn').click();
    await page.waitForTimeout(1800); // auto-save
    // Back to list so we can switch to the Builder.
    await page.click('#fill-back').catch(() => {});

    // Admin cuts Field A in the Builder and saves.
    const msg = await saveViaBuilderCapturingDialog(page, tpl.id, () => {
      const t = state.activeTemplate;
      t.sections[0].fields = t.sections[0].fields.filter(f => f.label !== 'Field A');
    });

    expect(msg, 'a discard warning must be surfaced').toBeTruthy();
    expect(msg).toMatch(/1 crew have unsubmitted answers on fields you.?re removing/i);

    // Confirm proceeded: Field A is gone and its draft was discarded.
    const after = await apiCall(page, 'GET', 'templates');
    const tplAfter = after.find(t => t.name === 'Cut Warn Checklist');
    expect(tplAfter.sections[0].fields.some(f => f.label === 'Field A')).toBe(false);
    const cnt = await apiCall(page, 'GET', 'draftHolderCount?field_ids=' + fieldA.id);
    expect(cnt.count).toBe(0);
  });

  test('F) cutting a field with NO drafts surfaces no warning', async ({ page }) => {
    test.setTimeout(90000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getDOW(page);

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'No Warn Checklist', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [
        CHECKBOX('Field A', 0), CHECKBOX('Field B', 1),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'No Warn Checklist');

    // No drafts created. Cut Field A and save → no dialog.
    const msg = await saveViaBuilderCapturingDialog(page, tpl.id, () => {
      const t = state.activeTemplate;
      t.sections[0].fields = t.sections[0].fields.filter(f => f.label !== 'Field A');
    });
    expect(msg, 'no warning when nobody has drafts').toBeNull();

    const after = await apiCall(page, 'GET', 'templates');
    const tplAfter = after.find(t => t.name === 'No Warn Checklist');
    expect(tplAfter.sections[0].fields.some(f => f.label === 'Field A')).toBe(false);
  });

  test('G) dropping today from the schedule with a crew draft warns naming the count', async ({ page }) => {
    test.setTimeout(90000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getDOW(page);
    const notToday = (dow + 3) % 7;

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Sched Warn Checklist', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [CHECKBOX('Prep', 0)] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Sched Warn Checklist');

    // Crew draft on Prep.
    await page.reload();
    await expect(page.locator('#s1').getByText('Sched Warn Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(page);
    await page.locator('.fill-field', { hasText: 'Prep' }).locator('.check-btn').click();
    await page.waitForTimeout(1800);
    await page.click('#fill-back').catch(() => {});

    // Admin drops today from the schedule and saves.
    const msg = await saveViaBuilderCapturingDialog(page, tpl.id, () => {
      state.activeTemplate.active_days = [(new Date().getDay() + 3) % 7];
    });
    expect(msg, 'a schedule-drop warning must be surfaced').toBeTruthy();
    expect(msg).toMatch(/1 crew have this open today/i);

    // Confirmed: schedule no longer includes today.
    const after = await apiCall(page, 'GET', 'templates');
    const tplAfter = after.find(t => t.name === 'Sched Warn Checklist');
    expect(tplAfter.schedules[0].active_days).toContain(notToday);
    expect(tplAfter.schedules[0].active_days).not.toContain(dow);
  });
});
