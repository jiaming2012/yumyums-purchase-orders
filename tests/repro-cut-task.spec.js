// ─── REPRO SPEC (temporary) ──────────────────────────────────────────────────
// Reproduces the operator report (2026-07-16): "cut a task from the Friday
// checklist, run it from two devices — checking/unchecking on one device does
// not show or persist on the other; refreshing a tab also doesn't work."
//
// Hypothesis under test: replaceTemplate (repository.go) deletes + re-inserts
// every field with NEW ids on any template edit. Devices that already have the
// checklist open keep the OLD field ids; their saves write draft responses
// under dead field ids (the field_id FK was dropped in 0051/0053/0054, so the
// server accepts them silently). Nothing propagates; nothing survives reload.
//
// ACTIVE as of the `editprop-stable-field-identity` card (FR-2/FR-3,
// INV-2/INV-4). The .skip is removed: `updateTemplate` now diff-upserts by the
// Builder-sent field ids, so a field that survives an edit keeps ONE permanent
// checklist_fields.id for life. The three assertions below — a check on a
// SURVIVING field propagates live and survives reload on both devices — are the
// post-edit STABLE-IDENTITY check. They go RED on the delete-and-reinsert build
// (field-id churn writes the save under a dead id; nothing propagates or
// survives reload) and GREEN on the diff-upsert build. The frozen-at-submit /
// cut-field-live-rerender semantic is layered on by the broadcast-rerender card,
// which builds on this stable-identity foundation.

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
  for (const t of templates) {
    await apiCall(page, 'DELETE', 'archiveTemplate/' + t.id);
  }
}

async function getTodayDOW(page) {
  return page.evaluate(() => new Date().getDay());
}

test.describe('REPRO: template edit while checklist is open on other devices', () => {
  test('cut a task mid-run → cross-device sync and persistence of remaining fields', async ({ browser, page }) => {
    test.setTimeout(90000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getTodayDOW(page);

    // "Friday checklist" with two tasks.
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Friday Checklist',
      requires_approval: false,
      sections: [{ title: 'Closing', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Wipe counters', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        { type: 'checkbox', label: 'Check fridge temps', required: false, order: 1, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Device A opens the checklist.
    await page.reload();
    await expect(page.locator('#s1').getByText('Friday Checklist')).toBeVisible({ timeout: 10000 });
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.check-btn', { timeout: 10000 });

    // Device B opens the same checklist.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Friday Checklist')).toBeVisible({ timeout: 10000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForSelector('.check-btn', { timeout: 10000 });

    // ── Baseline: live sync works BEFORE the template edit ──────────────────
    const wipeA = page.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    const wipeB = pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    await wipeA.click();
    await expect(wipeA).toHaveClass(/checked/, { timeout: 5000 });
    await expect(wipeB).toHaveClass(/checked/, { timeout: 10000 });
    console.log('[repro] baseline: pre-edit cross-device sync OK');

    // Uncheck and settle so the edit starts from a clean slate.
    await wipeA.click();
    await expect(wipeB).not.toHaveClass(/checked/, { timeout: 10000 });
    await page.waitForTimeout(1500);

    // ── The "cut": remove 'Check fridge temps' from the template, exactly as
    // the Builder's save does (remaining field keeps its id in the payload).
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Friday Checklist');
    const wipeField = tpl.sections[0].fields.find(f => f.label === 'Wipe counters');
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Friday Checklist',
      requires_approval: false,
      sections: [{ title: 'Closing', order: 0, condition: null, fields: [
        { id: wipeField.id, type: 'checkbox', label: 'Wipe counters', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });
    console.log('[repro] template edited: "Check fridge temps" cut, "Wipe counters" kept (old id ' + wipeField.id + ')');

    // ── Post-edit: Device A (still on the OLD render) checks 'Wipe counters'.
    await wipeA.click();
    await expect(wipeA).toHaveClass(/checked/, { timeout: 5000 }); // optimistic UI
    await page.waitForTimeout(2500); // auto-save + WS window

    // Symptom 1: does it show on Device B (also still open)?
    const bSawIt = await wipeB.evaluate(el => el.classList.contains('checked')).catch(() => false);
    console.log('[repro] symptom 1 — live propagation to open Device B: ' + (bSawIt ? 'WORKS' : 'BROKEN'));

    // Symptom 2: does a refreshed tab see it?
    await pageB.reload();
    await expect(pageB.locator('#s1').getByText('Friday Checklist')).toBeVisible({ timeout: 10000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForSelector('.check-btn', { timeout: 10000 });
    const wipeBAfterReload = pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    const bReloadSawIt = await wipeBAfterReload.evaluate(el => el.classList.contains('checked')).catch(() => false);
    console.log('[repro] symptom 2 — visible on Device B after reload: ' + (bReloadSawIt ? 'WORKS' : 'BROKEN'));

    // Symptom 3: does Device A's own check survive ITS refresh?
    await page.reload();
    await expect(page.locator('#s1').getByText('Friday Checklist')).toBeVisible({ timeout: 10000 });
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.check-btn', { timeout: 10000 });
    const wipeAAfterReload = page.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    const aReloadSawIt = await wipeAAfterReload.evaluate(el => el.classList.contains('checked')).catch(() => false);
    console.log('[repro] symptom 3 — Device A own check survives reload: ' + (aReloadSawIt ? 'WORKS' : 'BROKEN'));

    // Where did the save actually go? Inspect the draft responses server-side.
    const state = await apiCall(page, 'GET', 'myChecklistState?template_id=' + tpl.id).catch(() => null);
    console.log('[repro] myChecklistState after the fact: ' + JSON.stringify(state).slice(0, 400));

    await ctxB.close();

    // The test PASSES if the product behaves correctly — i.e. these assert the
    // DESIRED behavior, so a repro shows up as a failure with the log above.
    expect(bSawIt, 'post-edit check must propagate live to Device B').toBe(true);
    expect(bReloadSawIt, 'post-edit check must be visible on Device B after reload').toBe(true);
    expect(aReloadSawIt, 'post-edit check must survive Device A reload').toBe(true);
  });
});
