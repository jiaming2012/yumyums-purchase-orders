// ─── FROZEN-AT-SUBMIT / EDIT-PROPAGATION SEMANTIC ACCEPTANCE PAIR ─────────────
// Activity 5, W-3 (editprop-convergence-matrix). This file is the recorded
// red→green baseline pair for the editprop lifecycle — the Delivery KR. It began
// as the 2026-07-16 operator repro ("cut a task mid-run; two devices don't stay
// in sync; a refresh doesn't help"). With W-1 (stable field identity) and W-2
// (broadcast re-render) landed, that raw repro is GREEN; this rewrite promotes it
// to the two load-bearing acceptance tests of the signed design's §6 Convergence
// contract + Frozen-at-submit lifecycle (FR-6/FR-7, INV-3, A-5):
//
//   AC-6a — a mid-run template edit RE-RENDERS open devices with surviving
//           answers INTACT, asserted on the observing (second) device, converged
//           within one op round-trip (no reload), and again after reconnect.
//   AC-6b — a submitted record's rendered review is BYTE-IDENTICAL to what was
//           submitted, regardless of later template edits (frozen snapshot).
//
// RED provenance (see the card's G3): AC-6a goes RED on the pre-W-1/W-2 source
// (field-id churn writes the surviving save under a dead id → the answer vanishes
// on re-render; and the open runner is never re-fetched, so the cut never
// propagates live). AC-6b goes RED if the submitted review is ever re-derived
// from the LIVE template instead of the frozen template_snapshot (a later edit
// would leak into the record). Both are GREEN on the current tree.
//
// The BROADER per-type convergence matrix (all 7 persisted types + sub-steps +
// photo-URL + submit/unsubmit + list progress, live + catch-up) lives in
// tests/sync.spec.js under "Convergence matrix (W-3)".

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
  const pending = await apiCall(page, 'GET', 'pendingApprovals');
  if (Array.isArray(pending)) {
    for (const s of pending) await apiCall(page, 'POST', 'approveSubmission', { submission_id: s.id });
  }
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
  await page.waitForSelector('.check-btn, .fill-field', { timeout: 10000 });
}

// Fingerprint the runner's rendered field list: the (id, label) sequence. Field
// ids and labels in a submitted review come from the FROZEN snapshot, so this is
// immune to any later edit of the live template — the AC-6b byte-identity probe.
async function fieldFingerprint(page) {
  return page.evaluate(() => {
    return Array.from(document.querySelectorAll('#fill-body .fill-field')).map(el => ({
      id: el.getAttribute('data-field-id'),
      label: (el.querySelector('.fill-field-label') || {}).textContent || '',
    }));
  });
}

// ─── AC-6a ────────────────────────────────────────────────────────────────────

test.describe('AC-6a: mid-run edit re-renders open devices, surviving answers intact', () => {
  test('cut a task mid-run → surviving checkbox stays checked on the observing device (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getDOW(page);

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Friday Checklist', requires_approval: false,
      sections: [{ title: 'Closing', order: 0, condition: null, fields: [
        CHECKBOX('Wipe counters', 0), CHECKBOX('Check fridge temps', 1),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Observing device B (crew) opens the checklist and checks the SURVIVING field.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Friday Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(pageB);
    const wipeB = pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    await wipeB.click();
    await expect(wipeB).toHaveClass(/checked/, { timeout: 5000 });
    await pageB.waitForTimeout(1800); // auto-save

    // Admin (device A) cuts 'Check fridge temps', keeping 'Wipe counters' with its
    // stable id — exactly what the Builder save emits.
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Friday Checklist');
    const wipe = tpl.sections[0].fields.find(f => f.label === 'Wipe counters');
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Friday Checklist', requires_approval: false,
      sections: [{ title: 'Closing', order: 0, condition: null, fields: [
        CHECKBOX('Wipe counters', 0, wipe.id),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // LIVE (one op round-trip, no reload) on the observing device:
    //   • the cut field is gone (the SAVE_TEMPLATE op converged), and
    //   • the surviving field is STILL checked (answer intact through re-render).
    await expect(pageB.locator('.fill-field', { hasText: 'Check fridge temps' })).toHaveCount(0, { timeout: 10000 });
    const wipeLive = pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    await expect(wipeLive).toHaveClass(/checked/, { timeout: 5000 });

    // CATCH-UP (reconnect): reload the observing device; the answer survives the
    // fresh fetch + hydrate under the stable id, and the cut field stays gone.
    await pageB.reload();
    await expect(pageB.locator('#s1').getByText('Friday Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(pageB);
    await expect(pageB.locator('.fill-field', { hasText: 'Check fridge temps' })).toHaveCount(0);
    const wipeReload = pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    await expect(wipeReload).toHaveClass(/checked/, { timeout: 5000 });

    await ctxB.close();
  });
});

// ─── AC-6b ────────────────────────────────────────────────────────────────────

test.describe('AC-6b: submitted record byte-identical after later template edits', () => {
  test('a later edit (rename + add + cut) does not change the submitted record’s rendered review', async ({ page }) => {
    test.setTimeout(120000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    const dow = await getDOW(page);

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Frozen Checklist', requires_approval: false,
      sections: [{ title: 'Closing', order: 0, condition: null, fields: [
        CHECKBOX('Wipe counters', 0), CHECKBOX('Check fridge temps', 1),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Fill + submit (requires_approval false → 'submitted', freezes the snapshot).
    await page.reload();
    await expect(page.locator('#s1').getByText('Frozen Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(page);
    // Complete BOTH fields so submit has no "incomplete — submit anyway?" confirm.
    await page.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn').click();
    await page.locator('.fill-field', { hasText: 'Check fridge temps' }).locator('.check-btn').click();
    await page.waitForTimeout(1500);
    await page.locator('[data-action="submit"]').click();
    await expect(page.locator('.submit-confirm')).toBeVisible({ timeout: 10000 });

    // Fingerprint the submitted (read-only) review BEFORE the edit.
    const before = await fieldFingerprint(page);
    expect(before.map(f => f.label)).toEqual(['Wipe counters', 'Check fridge temps']);
    // Snapshot the server-side frozen template_snapshot too (byte string). Select
    // by the snapshot's template name — a no-approval submission surfaces in
    // MY_SUBMISSIONS as status 'pending', so a status filter would miss it.
    const snapOf = () => page.evaluate(() => {
      var subs = (typeof MY_SUBMISSIONS !== "undefined" ? MY_SUBMISSIONS : []);
      var s = subs.find(function(x) {
        var snap = typeof x.template_snapshot === 'string' ? JSON.parse(x.template_snapshot) : x.template_snapshot;
        return snap && snap.name === 'Frozen Checklist';
      });
      return s ? (typeof s.template_snapshot === 'string' ? s.template_snapshot : JSON.stringify(s.template_snapshot)) : null;
    });
    const snapBefore = await snapOf();
    expect(snapBefore).toBeTruthy();

    // Admin edits the LIVE template AFTER submit: rename a field, add a field, cut
    // a field — a maximally disruptive structural change.
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Frozen Checklist');
    const wipe = tpl.sections[0].fields.find(f => f.label === 'Wipe counters');
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Frozen Checklist', requires_approval: false,
      sections: [{ title: 'Closing', order: 0, condition: null, fields: [
        CHECKBOX('Wipe counters RENAMED', 0, wipe.id), // rename surviving field
        CHECKBOX('Brand new field', 1),                // add
        // 'Check fridge temps' cut
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Sanity: the LIVE template really changed.
    const after = await apiCall(page, 'GET', 'templates');
    const tplAfter = after.find(t => t.id === tpl.id);
    expect(tplAfter.sections[0].fields.map(f => f.label).sort())
      .toEqual(['Brand new field', 'Wipe counters RENAMED']);

    // Reopen the submitted record on the same device (re-render from snapshot).
    await page.reload();
    await expect(page.locator('#s1').getByText('Frozen Checklist')).toBeVisible({ timeout: 10000 });
    await openRunner(page);
    await expect(page.locator('.submit-confirm')).toBeVisible({ timeout: 10000 });

    // BYTE-IDENTICAL: the rendered review is unchanged — same field ids, same
    // labels, in the same order — NOT the renamed/added/cut live shape.
    const afterFp = await fieldFingerprint(page);
    expect(afterFp).toEqual(before);
    expect(afterFp.map(f => f.label)).toEqual(['Wipe counters', 'Check fridge temps']);
    expect(afterFp.map(f => f.label)).not.toContain('Wipe counters RENAMED');
    expect(afterFp.map(f => f.label)).not.toContain('Brand new field');

    // The frozen server snapshot string is unchanged.
    const snapAfter = await snapOf();
    expect(snapAfter).toBe(snapBefore);
  });
});
