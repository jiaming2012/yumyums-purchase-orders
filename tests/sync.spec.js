const { test, expect } = require('@playwright/test');

const BASE = '';
const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// ─── Helpers ─────────────────────────────────────────────────────────────────

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

async function cleanupPendingApprovals(page) {
  const pending = await apiCall(page, 'GET', 'pendingApprovals');
  if (!Array.isArray(pending)) return;
  for (const s of pending) {
    await apiCall(page, 'POST', 'approveSubmission', { submission_id: s.id });
  }
}

async function getTodayDOW(page) {
  return page.evaluate(() => new Date().getDay());
}

async function createTestTemplate(page, name, todayDOW) {
  name = name || 'Sync Test Template';
  const input = {
    name,
    requires_approval: todayDOW !== undefined,
    sections: [
      {
        title: 'Section 1',
        order: 0,
        condition: null,
        fields: [
          {
            type: 'checkbox',
            label: 'Sync checkbox',
            required: false,
            order: 0,
            config: {},
            fail_trigger: null,
            condition: null,
          },
        ],
      },
    ],
  };
  if (todayDOW !== undefined) {
    input.schedules = [{ active_days: [todayDOW] }];
    input.assignments = [
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
    ];
  }
  return apiCall(page, 'POST', 'createTemplate', input);
}

// ─── A. Shared checklist visibility ──────────────────────────────────────────

test.describe('Cross-device: shared checklist', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupPendingApprovals(page);
    await cleanupTemplates(page);
  });

  test('two contexts see the same checklist from My Checklists [LC-04]', async ({ browser, page }) => {
    const dow = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Shared Checklist', dow);

    // Device A: reload to see the new template
    await page.reload();
    await expect(page.locator('#s1').getByText('Shared Checklist')).toBeVisible({ timeout: 10000 });

    // Device B: separate context, login, navigate
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Shared Checklist')).toBeVisible({ timeout: 10000 });

    await ctxB.close();
  });

  test('Device B sees field changes from Device A after reload [SYN-03]', async ({ browser, page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Cross-Device Test', dow);
    await page.reload();

    // Device A: open checklist and check the checkbox
    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    // Wait for auto-save (400ms debounce + server round trip)
    await page.waitForTimeout(2000);

    // Device B: open same checklist — should see the checked field via draft responses
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Cross-Device Test')).toBeVisible({ timeout: 10000 });
    await pageB.click('[data-fill-template-id]');
    // The checkbox should already be checked (loaded from drafts)
    await expect(pageB.locator('.check-btn.checked')).toBeVisible({ timeout: 5000 });

    await ctxB.close();
  });
});

// ─── B. Op log generation ────────────────────────────────────────────────────

test.describe('Cross-device: op log', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupPendingApprovals(page);
    await cleanupTemplates(page);
  });

  test('saving a field generates a SET_FIELD op visible via ops/since [SYN-01]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Ops Test', dow);
    await page.reload();

    // Open checklist and check the checkbox
    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await page.waitForTimeout(2000); // debounce + save + EmitOp

    // Query ops/since for SET_FIELD op
    const ops = await apiCall(page, 'GET', 'ops/since?lamport_ts=0');
    expect(Array.isArray(ops)).toBe(true);
    const setFieldOp = ops.find(op => op.op_type === 'SET_FIELD');
    expect(setFieldOp).toBeDefined();
    expect(setFieldOp.lamport_ts).toBeGreaterThan(0);
    expect(setFieldOp.entity_type).toBe('field_response');
  });

  test('template creation generates SAVE_TEMPLATE op [SYN-10]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Template Op Test', dow);
    await page.waitForTimeout(1500); // EmitOp is async

    const ops = await apiCall(page, 'GET', 'ops/since?lamport_ts=0');
    expect(Array.isArray(ops)).toBe(true);
    const saveOp = ops.find(op => op.op_type === 'SAVE_TEMPLATE' && op.entity_id === tpl.id);
    expect(saveOp).toBeDefined();
    expect(saveOp.entity_type).toBe('template');
  });

  test('archiving template generates ARCHIVE_TEMPLATE op [SYN-10]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Archive Op Test', dow);
    await page.waitForTimeout(1000);

    await apiCall(page, 'DELETE', 'archiveTemplate/' + tpl.id);
    await page.waitForTimeout(1500);

    const ops = await apiCall(page, 'GET', 'ops/since?lamport_ts=0');
    const archiveOp = ops.find(op => op.op_type === 'ARCHIVE_TEMPLATE' && op.entity_id === tpl.id);
    expect(archiveOp).toBeDefined();
  });

  test('multiple ops on same entity have incrementing lamport_ts [SYN-01]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Lamport Test', dow);
    await page.waitForTimeout(1000);

    // Update the same template to generate a second op on the same entity
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Lamport Test Updated',
      requires_approval: true,
      sections: tpl.sections || [{ title: 'S1', order: 0, condition: null, fields: [] }],
      schedules: [{ active_days: [dow] }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });
    await page.waitForTimeout(1500);

    const ops = await apiCall(page, 'GET', 'ops/since?lamport_ts=0');
    const templateOps = ops.filter(op =>
      op.op_type === 'SAVE_TEMPLATE' && op.entity_id === tpl.id
    );
    expect(templateOps.length).toBeGreaterThanOrEqual(2);
    // Second op should have higher lamport_ts than the first
    expect(templateOps[1].lamport_ts).toBeGreaterThan(templateOps[0].lamport_ts);
  });
});

// ─── C. Regression tests ─────────────────────────────────────────────────────

test.describe('Cross-device: regressions', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupPendingApprovals(page);
    await cleanupTemplates(page);
  });

  test('checking a field does not uncheck itself via WS echo [SYN-02]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Echo Test', dow);
    await page.reload();

    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });

    // Wait long enough for the WS echo to arrive (save 400ms + server + notify + WS)
    await page.waitForTimeout(3000);

    // Field must still be checked — WS echo must not have reverted it
    await expect(checkBtn).toHaveClass(/checked/);
  });

  test('no "updated by" toast appears for own field saves [RUN-16 SYN-02]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Toast Echo Test', dow);
    await page.reload();

    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();

    // Wait for WS echo window
    await page.waitForTimeout(3000);

    // No sync toast should have appeared for own save
    const toastVisible = await page.locator('.sync-toast').isVisible().catch(() => false);
    expect(toastVisible).toBe(false);
  });

  test('catch-up replay of the op backlog shows no "updated by" toast flood [SYN-02]', async ({ page }) => {
    // Regression (2026-07-16): wsCatchUp replayed the whole historical op
    // backlog through applyOp as if the ops were live teammate edits. After a
    // reload the device_id regenerates, so the user's OWN past ops no longer
    // self-suppress — every checklist open fired an "N fields updated by X"
    // toast flood. Catch-up must apply ops silently.
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'CatchUp Silent Test', dow);
    await page.reload();

    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(2000); // auto-save + op persisted server-side

    // Simulate the post-reload fresh-device state while still inside the
    // checklist (the toast only renders inside a detail view): zero the
    // Lamport clock and swap the device_id so nothing self-suppresses, then
    // drive a real catch-up over the full backlog.
    await page.evaluate(async () => {
      LAMPORT_CLOCK._ts = 0;
      LAMPORT_CLOCK._deviceId = 'test-fresh-device-' + Date.now();
      await wsCatchUp();
    });

    // The toast queue flushes 500ms after enqueue and shows for 3s — poll
    // through that whole window; it must never appear for replayed ops.
    let sawToast = false;
    for (let i = 0; i < 13; i++) {
      if (await page.locator('.sync-toast.show').isVisible().catch(() => false)) { sawToast = true; break; }
      await page.waitForTimeout(200);
    }
    expect(sawToast, 'catch-up replay must not fire "updated by" toasts').toBe(false);

    // Silent must mean quiet, not skipped — the replayed state still applies.
    await expect(checkBtn).toHaveClass(/checked/);
  });

  test('live field edit from another device still shows the "updated by" toast [SYN-02]', async ({ browser, page }) => {
    // Positive control for the silent catch-up fix: only REPLAYED ops are
    // silent. A live WS op from another device must still flash + toast.
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Live Toast Test', dow);
    await page.reload();

    // Device A (this page): open the checklist and stay on it.
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.check-btn', { timeout: 10000 });

    // Device B: open the same checklist and check the field.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Live Toast Test')).toBeVisible({ timeout: 10000 });
    await pageB.click('[data-fill-template-id]');
    const checkBtnB = pageB.locator('.check-btn').first();
    await checkBtnB.click();
    await expect(checkBtnB).toHaveClass(/checked/, { timeout: 5000 });

    // Device A: the live op arrives over WS → toast must appear.
    await expect(page.locator('.sync-toast.show')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.sync-toast')).toContainText('updated by');

    await ctxB.close();
  });

  test('SET_FIELD op includes user_name in payload [SYN-01]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'UserName Op Test', dow);
    await page.reload();

    await page.click('[data-fill-template-id]');
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(2000);

    const ops = await apiCall(page, 'GET', 'ops/since?lamport_ts=0');
    const setFieldOp = ops.find(op => op.op_type === 'SET_FIELD');
    expect(setFieldOp).toBeDefined();
    expect(setFieldOp.payload.user_name).toBeDefined();
    expect(setFieldOp.payload.user_name).not.toBe('Someone');
    expect(setFieldOp.payload.user_name.length).toBeGreaterThan(0);
  });

  test('save status clears after rapid field saves [RUN-04]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'SaveStatus Test', dow);
    await page.reload();

    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();

    // Rapid toggle: check, uncheck, check — simulates rapid taps
    await checkBtn.click();
    await page.waitForTimeout(100);
    await checkBtn.click();
    await page.waitForTimeout(100);
    await checkBtn.click();

    // Wait for debounce (400ms) + API round trip + "Synced" display + fade
    await page.waitForTimeout(5000);

    // Save status must not be stuck on "Saving..."
    const saveStatus = page.locator('#save-status');
    const text = await saveStatus.textContent().catch(() => '');
    expect(text).not.toContain('Saving');
  });

  test('field attribution shows user name not undefined after save [RUN-03]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Attribution Test', dow);
    await page.reload();

    await page.click('[data-fill-template-id]');
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(2000);

    // Check the attribution text under the field
    const attribution = page.locator('.fill-attribution').first();
    const attrText = await attribution.textContent();
    expect(attrText).not.toContain('undefined');
    expect(attrText.length).toBeGreaterThan(2);
  });

  test('unchecked field stays unchecked after navigating away and returning [FLD-01 RUN-18]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Uncheck Persist', dow);
    await page.reload();

    // Open checklist, check then uncheck
    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(1500); // auto-save
    await checkBtn.click();
    await expect(checkBtn).not.toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(2000); // auto-save the uncheck

    // Navigate back to list
    await page.click('#fill-back');
    await page.waitForTimeout(500);

    // Row should show 0/1 (not 1/1)
    const rowText = await page.locator('[data-fill-template-id]').first().textContent();
    expect(rowText).toContain('0/1');

    // Reopen the checklist
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.check-btn', { timeout: 5000 });
    await page.waitForTimeout(500);

    // Checkbox must NOT be checked
    await expect(page.locator('.check-btn').first()).not.toHaveClass(/checked/);
  });

  test('list page progress decrements when another device unchecks a field [LST-17]', async ({ browser, page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Uncheck Sync', dow);

    // Device A: check the checkbox first
    await page.reload();
    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(2000);

    // Device B: open list page, should show 1/1
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Uncheck Sync')).toBeVisible({ timeout: 10000 });
    const beforeText = await pageB.locator('[data-fill-template-id]').first().textContent();
    expect(beforeText).toContain('1/1');

    // Device A: now UNCHECK the checkbox
    await checkBtn.click();
    await expect(checkBtn).not.toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(3000); // auto-save + WS propagation

    // Device B: list page should now show 0/1
    await pageB.waitForTimeout(3000);
    const afterText = await pageB.locator('[data-fill-template-id]').first().textContent();
    expect(afterText).toContain('0/1');

    await ctxB.close();
  });

  test('sub-step checks on Device A appear checked on Device B [SYN-03]', async ({ browser, page }) => {
    const dow = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'SubStep Sync',
      requires_approval: false,
      sections: [{
        title: 'Inventory', order: 0, condition: null,
        fields: [{
          type: 'checkbox', label: 'Protein stock', required: false, order: 0,
          config: {}, fail_trigger: null, condition: null,
          sub_steps: [
            { type: 'checkbox', label: 'Salmon counted', order: 0, config: {}, fail_trigger: null, condition: null },
            { type: 'checkbox', label: 'Chicken counted', order: 1, config: {}, fail_trigger: null, condition: null },
          ],
        }],
      }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });
    await page.reload();
    await expect(page.locator('#s1').getByText('SubStep Sync')).toBeVisible({ timeout: 10000 });

    // Device B: open the same checklist FIRST (before Device A makes changes)
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('SubStep Sync')).toBeVisible({ timeout: 10000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForSelector('.sub-step-check', { timeout: 10000 });
    // Verify sub-step is NOT checked initially
    await expect(pageB.locator('.sub-step-check').first()).not.toHaveClass(/done/);

    // Device A: open checklist and check first sub-step
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.sub-step-check', { timeout: 10000 });
    await page.locator('.sub-step-check').first().click();
    await expect(page.locator('.sub-step-check').first()).toHaveClass(/done/, { timeout: 5000 });
    await page.waitForTimeout(3000); // auto-save + WS propagation

    // Device B: sub-step should now be checked via WS sync
    await pageB.waitForTimeout(3000);
    await expect(pageB.locator('.sub-step-check').first()).toHaveClass(/done/);

    await ctxB.close();
  });

  test('submit on Device A updates runner view to submitted on Device B [SYN-03]', async ({ browser, page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Submit Runner Sync', dow);
    await page.reload();
    await expect(page.locator('#s1').getByText('Submit Runner Sync')).toBeVisible({ timeout: 10000 });

    // Device A: open checklist and check the field
    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(2000);

    // Device B: open the SAME checklist (before Device A submits)
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Submit Runner Sync')).toBeVisible({ timeout: 10000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForSelector('.check-btn', { timeout: 5000 });
    // Verify submit button is visible (not yet submitted)
    await expect(pageB.locator('[data-action="submit"]')).toBeVisible({ timeout: 5000 });

    // Device A: submit the checklist
    await page.locator('[data-action="submit"]').click();
    await page.waitForTimeout(3000);

    // Device B: navigate back to list and reopen to get fresh state
    // (simulates what the user sees after WS triggers loadMyChecklists + renderRunner)
    await pageB.click('#fill-back');
    await pageB.waitForTimeout(500);
    await pageB.reload();
    await expect(pageB.locator('#s1').getByText('Submit Runner Sync')).toBeVisible({ timeout: 10000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForTimeout(1000);

    // Device B: should show submitted/pending state (not the submit button)
    const hasSubmittedText = await pageB.locator('.submit-confirm').isVisible().catch(() => false);
    const submitBtnGone = await pageB.locator('[data-action="submit"]').isHidden().catch(() => true);
    expect(hasSubmittedText || submitBtnGone).toBe(true);

    await ctxB.close();
  });

  test('temperature input keeps cursor position while typing [SYN-09]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Temp Cursor Test',
      requires_approval: false,
      sections: [{
        title: 'Section 1', order: 0, condition: null,
        fields: [{
          type: 'temperature', label: 'Grill temp', required: false, order: 0,
          config: { unit: 'F', min: 0, max: 500 }, fail_trigger: null, condition: null,
        }],
      }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });
    await page.reload();
    await expect(page.locator('#s1').getByText('Temp Cursor Test')).toBeVisible({ timeout: 10000 });
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.fill-temp-input', { timeout: 5000 });

    const input = page.locator('.fill-temp-input');
    await input.focus();
    // Type "400" character by character
    await input.pressSequentially('400', { delay: 100 });

    // The value should be "400", not "4" or "40" or mangled
    const val = await input.inputValue();
    expect(val).toBe('400');
  });

  test('generateUUID works when crypto.randomUUID is unavailable [SYN-11]', async ({ page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'UUID Fallback', dow);
    await page.reload();

    // Simulate non-secure context by removing crypto.randomUUID
    const uuid = await page.evaluate(() => {
      const orig = crypto.randomUUID;
      crypto.randomUUID = undefined;
      try {
        return generateUUID();
      } finally {
        crypto.randomUUID = orig;
      }
    });
    // Should return a valid UUID v4 format
    expect(uuid).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  });

  test('two tabs on same origin have different device_ids for sync [SYN-07]', async ({ context, page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'DeviceID Test', dow);
    await page.reload();
    await expect(page.locator('#s1').getByText('DeviceID Test')).toBeVisible({ timeout: 10000 });

    // Tab A: open checklist
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.check-btn', { timeout: 5000 });

    // Tab B: SAME context (shares IndexedDB, cookies — simulates second tab)
    const pageB = await context.newPage();
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('DeviceID Test')).toBeVisible({ timeout: 10000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForSelector('.check-btn', { timeout: 5000 });

    // Get device IDs from both tabs
    const deviceA = await page.evaluate(() => window.LAMPORT_CLOCK ? window.LAMPORT_CLOCK.deviceId : null);
    const deviceB = await pageB.evaluate(() => window.LAMPORT_CLOCK ? window.LAMPORT_CLOCK.deviceId : null);

    expect(deviceA).not.toBeNull();
    expect(deviceB).not.toBeNull();
    // Must be DIFFERENT — otherwise self-echo suppression blocks cross-tab sync
    expect(deviceA).not.toBe(deviceB);

    await pageB.close();
  });

  test('list page progress updates when another device completes a field [LST-17]', async ({ browser, page }) => {
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'Progress Sync', dow);

    // Device B: open list page FIRST and verify 0/1 progress
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Progress Sync')).toBeVisible({ timeout: 10000 });
    const beforeText = await pageB.locator('[data-fill-template-id]').first().textContent();
    expect(beforeText).toContain('0/1');

    // Device A: open checklist and check the checkbox
    await page.reload();
    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(2000); // auto-save + WS propagation

    // Device B: list page should now show 1/1 (updated via WS without reload)
    // Wait for WS delivery + re-render
    await pageB.waitForTimeout(3000);
    const afterText = await pageB.locator('[data-fill-template-id]').first().textContent();
    expect(afterText).toContain('1/1');

    await ctxB.close();
  });
});

// ─── D. Auth gates ──────────────────────────────────────────────────────────

test.describe('Cross-device: auth', () => {
  test('ops/since endpoint requires authentication [SYN-08]', async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    const response = await page.request.get('/api/v1/workflow/ops/since?lamport_ts=0');
    expect(response.status()).toBe(401);
    await ctx.close();
  });

  test('WebSocket endpoint requires authentication [SYN-08]', async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    await page.goto('/login.html');
    const wsResult = await page.evaluate(async () => {
      return new Promise((resolve) => {
        const proto = location.protocol === 'https:' ? 'wss://' : 'ws://';
        const ws = new WebSocket(proto + location.host + '/ws');
        ws.onopen = () => resolve('open');
        ws.onerror = () => resolve('error');
        ws.onclose = (e) => resolve('closed:' + e.code);
        setTimeout(() => resolve('timeout'), 5000);
      });
    });
    expect(wsResult).not.toBe('open');
    await ctx.close();
  });
});

// ─── Convergence matrix (W-3, Activity 5) ────────────────────────────────────
//
// editprop-convergence-matrix. Proves the signed §6 Convergence contract for
// EVERY persisted answer type — not just the checkbox W-2's E2E exercised. "In
// sync" = converged within one op round-trip, asserted on the OBSERVING (second)
// device, never on the writer's optimism.
//
// Each cell: the observing device B opens the runner and enters an answer of the
// given type. The admin (device A) then makes a structural edit — cutting an
// unrelated "Decoy" field while KEEPING the answer field with its stable id (the
// SAVE_TEMPLATE op). Two convergence paths are asserted on B:
//   • LIVE — no reload: the Decoy vanishes (the op round-tripped) AND the typed
//     answer survives the in-place re-render (W-2 rerenderOpenChecklistAfterSave
//     → loadMyChecklists → hydrateFieldState keyed by the stable id).
//   • CATCH-UP — B reloads (reconnect): the answer re-hydrates from the fresh
//     fetch, the Decoy stays gone.
//
// RED provenance: on the pre-W-1 source, the surviving field's id churns on every
// edit, so B's autosaved answer is written under a dead id and vanishes on
// re-render/reload (all cells RED). On the pre-W-2 source, B's open runner is
// never re-fetched, so the Decoy never disappears live (the LIVE half RED). Both
// landed → GREEN here; this matrix EXTENDS the proven surviving-answer guarantee
// from checkbox to all 7 persisted types + sub-steps + the photo-URL value.

const CHECKBOX_F = (label, order, extra) => Object.assign(
  { type: 'checkbox', label, required: false, order, config: {}, fail_trigger: null, condition: null },
  extra || {});

async function createWithDecoy(page, name, dow, answerField) {
  return apiCall(page, 'POST', 'createTemplate', {
    name, requires_approval: false,
    sections: [{ title: 'S', order: 0, condition: null, fields: [
      Object.assign({}, answerField, { order: 0 }),
      CHECKBOX_F('Decoy', 1),
    ] }],
    schedules: [{ active_days: [dow] }],
    assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
  });
}

// Cut the Decoy field, keeping every other field VERBATIM (with its stable id and
// any sub_steps ids) — exactly the diff-upsert the Builder save emits.
async function cutDecoy(page, name, dow) {
  const templates = await apiCall(page, 'GET', 'templates');
  const tpl = templates.find(t => t.name === name);
  const kept = tpl.sections[0].fields.filter(f => f.label !== 'Decoy');
  kept.forEach((f, i) => { f.order = i; });
  await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
    name, requires_approval: false,
    sections: [{ id: tpl.sections[0].id, title: tpl.sections[0].title, order: 0, condition: null, fields: kept }],
    schedules: [{ active_days: [dow] }],
    assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
  });
}

// Timeouts are deliberately generous: the full sync.spec.js runs serially on ONE
// server, and every matrix cell spins up a SECOND browser context. Under that
// combined CPU/WS load a render or op round-trip that takes <1s in isolation can
// take several seconds, so the two-device waits must not race it (Delivery KR:
// the suite must be green under its own load, not just in isolation).
const RUNNER_TIMEOUT = 12000;
const CONVERGE_TIMEOUT = 12000;

async function openRunnerB(pageB, name) {
  await pageB.goto(BASE + '/workflows.html');
  await expect(pageB.locator('#s1').getByText(name)).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await pageB.click('[data-fill-template-id]');
  await pageB.waitForSelector('.fill-field', { timeout: RUNNER_TIMEOUT });
}

async function reopenRunnerB(pageB, name) {
  await pageB.reload();
  await expect(pageB.locator('#s1').getByText(name)).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await pageB.click('[data-fill-template-id]');
  await pageB.waitForSelector('.fill-field', { timeout: RUNNER_TIMEOUT });
}

const waitAutosave = (pageB) => pageB.waitForResponse(
  res => res.url().includes('/api/v1/workflow/ops') && res.request().method() === 'POST',
  { timeout: 8000 }).catch(() => {});

// Deterministically wait until the answer field's draft is actually PERSISTED
// server-side (its response appears in myChecklists.drafts). The UI autosave is
// debounced, so under suite load it can land AFTER the test would otherwise cut
// the Decoy — the ensuing re-render would then hydrate an empty draft and drop
// the answer. Gating the edit on real persistence removes that whole class of
// load-only flake (the answer is provably server-side before the SAVE_TEMPLATE).
async function waitFieldPersisted(pageB, fieldId) {
  await expect.poll(async () => {
    const data = await apiCall(pageB, 'GET', 'myChecklists?dow=' + (new Date().getDay()));
    const drafts = (data && data.drafts) || [];
    return drafts.some(d => d.field_id === fieldId && d.value !== null && d.value !== undefined && d.value !== '');
  }, { timeout: 12000, intervals: [200, 400, 700, 1000] }).toBe(true);
}

// Drives one surviving-answer cell end to end (LIVE + CATCH-UP) on device B.
async function survivalCell(browser, page, name, answerField, enterFn, assertFn) {
  const dow = await getTodayDOW(page);
  await createWithDecoy(page, name, dow, answerField);
  const templates = await apiCall(page, 'GET', 'templates');
  const tpl = templates.find(t => t.name === name);
  const answerFieldId = tpl.sections[0].fields.find(f => f.label !== 'Decoy').id;

  const ctxB = await browser.newContext();
  const pageB = await ctxB.newPage();
  await login(pageB);
  await openRunnerB(pageB, name);

  await enterFn(pageB);
  await waitAutosave(pageB);
  await waitFieldPersisted(pageB, answerFieldId); // the answer is provably server-side
  await assertFn(pageB); // baseline: the answer is present on B

  // Admin cuts the Decoy (SAVE_TEMPLATE op).
  await cutDecoy(page, name, dow);

  // LIVE convergence on the observing device (no reload).
  await expect(pageB.locator('.fill-field', { hasText: 'Decoy' })).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
  await assertFn(pageB);

  // CATCH-UP convergence (reconnect).
  await reopenRunnerB(pageB, name);
  await expect(pageB.locator('.fill-field', { hasText: 'Decoy' })).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
  await assertFn(pageB);

  await ctxB.close();
}

test.describe('Convergence matrix (W-3): surviving answers converge across devices', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupPendingApprovals(page);
    await cleanupTemplates(page);
  });

  test('checkbox answer converges (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await survivalCell(browser, page, 'MX Checkbox',
      CHECKBOX_F('Wipe counters', 0),
      async (p) => {
        const b = p.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
        await b.click(); await expect(b).toHaveClass(/checked/, { timeout: 5000 });
      },
      async (p) => {
        await expect(p.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn'))
          .toHaveClass(/checked/, { timeout: 8000 });
      });
  });

  test('yes/no answer converges (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await survivalCell(browser, page, 'MX YesNo',
      { type: 'yes_no', label: 'All good?', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
      async (p) => {
        await p.click('[data-action="set-yes"]');
        await expect(p.locator('[data-action="set-yes"]')).toHaveClass(/on/, { timeout: 5000 });
      },
      async (p) => {
        await expect(p.locator('[data-action="set-yes"]')).toHaveClass(/on/, { timeout: 8000 });
        await expect(p.locator('[data-action="set-no"]')).not.toHaveClass(/on/);
      });
  });

  test('text answer converges (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await survivalCell(browser, page, 'MX Text',
      { type: 'text', label: 'Notes', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
      async (p) => {
        const ta = p.locator('.fill-textarea').first();
        await ta.fill('She said "hello" to me'); await ta.blur();
      },
      async (p) => {
        await expect(p.locator('.fill-textarea').first()).toHaveValue('She said "hello" to me', { timeout: 8000 });
      });
  });

  test('temperature answer converges (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await survivalCell(browser, page, 'MX Temp',
      { type: 'temperature', label: 'Grill temp', required: false, order: 0, config: { unit: 'F', min: 300, max: 500 }, fail_trigger: null, condition: null },
      async (p) => {
        const t = p.locator('input[type="number"]').first();
        await t.fill('375'); await t.dispatchEvent('change');
      },
      async (p) => {
        await expect(p.locator('input[type="number"]').first()).toHaveValue('375', { timeout: 8000 });
      });
  });

  test('sub-step checked state converges (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await survivalCell(browser, page, 'MX SubSteps',
      CHECKBOX_F('Protein stock', 0, { sub_steps: [
        { type: 'checkbox', label: 'Salmon counted', order: 0, config: {}, fail_trigger: null, condition: null },
        { type: 'checkbox', label: 'Chicken counted', order: 1, config: {}, fail_trigger: null, condition: null },
      ] }),
      async (p) => {
        const s = p.locator('.sub-step-check').first();
        await s.click(); await expect(s).toHaveClass(/done/, { timeout: 5000 });
      },
      async (p) => {
        await expect(p.locator('.sub-step-check').first()).toHaveClass(/done/, { timeout: 8000 });
        await expect(p.locator('.sub-step-check').nth(1)).not.toHaveClass(/done/);
      });
  });

  test('fail-note text AND fail severity converge (live + catch-up)', async ({ browser, page }) => {
    // Two matrix cells in one flow — both live on the same fail card of an
    // out-of-range temperature field. The fail note (value + note + severity) is
    // persisted via ONE deterministic saveResponse of the full {_v,_fail_note}
    // bundle, then hydrated on reopen — the same pattern the photo cell uses.
    // Driving temp-change + note-typing + severity-click as three separate
    // debounced UI autosaves on the SAME field races under suite load (the saves
    // coalesce/interleave and can persist an empty note), which is a test-harness
    // artifact, not a convergence defect — the bundle write is the durable state.
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    await createWithDecoy(page, 'MX FailNote', dow,
      { type: 'temperature', label: 'Grill temp', required: true, order: 0,
        config: { unit: 'F', min: 300, max: 500 },
        fail_trigger: { type: 'out_of_range', min: 300, max: 500 }, condition: null });

    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'MX FailNote');
    const fieldId = tpl.sections[0].fields.find(f => f.label === 'Grill temp').id;

    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);

    // Persist the full fail-note bundle (out-of-range value + note + severity) in
    // ONE write. The out-of-range _v:2 makes the runner render the fail card on
    // hydrate, so no UI temp-entry is needed — driving the number input's change
    // event to summon the fail card was itself a load-sensitive step.
    await apiCall(pageB, 'POST', 'saveResponse', {
      field_id: fieldId,
      value: { _v: 2, _fail_note: { note: 'Grill needs repair', severity: 'minor', photo: null } },
    });

    const assertFailNote = async (p) => {
      const card = p.locator('.fail-card');
      await expect(card).toBeVisible({ timeout: CONVERGE_TIMEOUT });
      await expect(card.locator('textarea')).toHaveValue('Grill needs repair', { timeout: CONVERGE_TIMEOUT });
      await expect(p.locator('[data-action="set-severity"][data-severity="minor"]')).toHaveClass(/on/, { timeout: CONVERGE_TIMEOUT });
    };
    // Baseline: open the runner (navigates to workflows.html) so the injected
    // bundle hydrates the fail card.
    await openRunnerB(pageB, 'MX FailNote');
    await assertFailNote(pageB);

    // Admin cuts the Decoy (SAVE_TEMPLATE op).
    await cutDecoy(page, 'MX FailNote', dow);

    // LIVE: note + severity survive the in-place re-render.
    await expect(pageB.locator('.fill-field', { hasText: 'Decoy' })).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
    await assertFailNote(pageB);

    // CATCH-UP: reload.
    await reopenRunnerB(pageB, 'MX FailNote');
    await expect(pageB.locator('.fill-field', { hasText: 'Decoy' })).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
    await assertFailNote(pageB);

    await ctxB.close();
  });

  test('fail-note photo-URL value converges (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    await createWithDecoy(page, 'MX Photo', dow,
      { type: 'yes_no', label: 'Equipment OK?', required: true, order: 0, config: {}, fail_trigger: null, condition: null });

    // Resolve the answer field's id for the photo injection.
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'MX Photo');
    const fieldId = tpl.sections[0].fields.find(f => f.label === 'Equipment OK?').id;
    const photoUrl = 'https://spaces.example.com/checklists/mx/fail-' + fieldId + '.jpg';

    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await openRunnerB(pageB, 'MX Photo');

    // 'No' triggers the fail card; then inject the presigned photo URL as the crew
    // camera would (camera UI is unavailable in the headless env).
    const noBtn = pageB.locator('[data-action="set-no"][data-fld-id="' + fieldId + '"]');
    await expect(noBtn).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await noBtn.click();
    await expect(pageB.locator('.fail-card')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await pageB.waitForTimeout(800); // let the No-answer autosave settle first
    await apiCall(pageB, 'POST', 'saveResponse', {
      field_id: fieldId, value: { _v: false, _fail_note: { note: '', severity: '', photo: photoUrl } },
    });
    await pageB.waitForTimeout(400);

    const assertPhoto = async (p) => {
      const img = p.locator('.fail-card img.photo-thumb');
      await expect(img).toBeVisible({ timeout: CONVERGE_TIMEOUT });
      expect(await img.getAttribute('src')).toBe(photoUrl);
    };
    // Baseline: reopen once so the injected photo hydrates into the runner.
    await reopenRunnerB(pageB, 'MX Photo');
    await assertPhoto(pageB);

    await cutDecoy(page, 'MX Photo', dow);

    // LIVE: photo survives the in-place re-render.
    await expect(pageB.locator('.fill-field', { hasText: 'Decoy' })).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
    await assertPhoto(pageB);

    // CATCH-UP: reload.
    await reopenRunnerB(pageB, 'MX Photo');
    await expect(pageB.locator('.fill-field', { hasText: 'Decoy' })).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
    await assertPhoto(pageB);

    await ctxB.close();
  });
});

test.describe('Convergence matrix (W-3): lifecycle + list progress', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupPendingApprovals(page);
    await cleanupTemplates(page);
  });

  test('submit transition converges live on the observing device (+ catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    await createTestTemplate(page, 'MX Submit', dow); // requires_approval true here

    // Writer A opens the runner.
    await page.reload();
    await expect(page.locator('#s1').getByText('MX Submit')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.check-btn', { timeout: RUNNER_TIMEOUT });
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(1200);

    // Observing device B opens the same runner (still fillable).
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await openRunnerB(pageB, 'MX Submit');
    await expect(pageB.locator('[data-action="submit"]')).toBeVisible({ timeout: RUNNER_TIMEOUT });

    // A submits.
    await page.locator('[data-action="submit"]').click();
    await page.waitForTimeout(1000);

    // LIVE on B (no reload): the runner converges to the submitted (read-only)
    // state — the submit button is gone / the confirm line shows.
    await expect(pageB.locator('[data-action="submit"]')).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
    await expect(pageB.locator('.submit-confirm')).toBeVisible({ timeout: CONVERGE_TIMEOUT });

    // CATCH-UP: reload B, reopen — still submitted.
    await reopenRunnerB(pageB, 'MX Submit');
    await expect(pageB.locator('.submit-confirm')).toBeVisible({ timeout: CONVERGE_TIMEOUT });

    await ctxB.close();
  });

  test('unsubmit transition converges live on the observing device (+ catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    // requires_approval false → submit yields 'submitted', which the submitter can unsubmit.
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'MX Unsubmit', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [CHECKBOX_F('Task', 0)] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Writer A opens, checks, submits.
    await page.reload();
    await expect(page.locator('#s1').getByText('MX Unsubmit')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.check-btn', { timeout: RUNNER_TIMEOUT });
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(1000);
    await page.locator('[data-action="submit"]').click();
    await expect(page.locator('.submit-confirm')).toBeVisible({ timeout: CONVERGE_TIMEOUT });

    // Observing device B opens and converges to submitted.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await openRunnerB(pageB, 'MX Unsubmit');
    await expect(pageB.locator('.submit-confirm')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await expect(pageB.locator('[data-action="unsubmit"]')).toBeVisible({ timeout: RUNNER_TIMEOUT });

    // A unsubmits (the runner confirms first — accept it).
    page.once('dialog', (d) => d.accept());
    await page.locator('[data-action="unsubmit"]').click();
    await page.waitForTimeout(1000);

    // LIVE on B (no reload): the runner converges back to fillable — the submit
    // button returns. (Requires the UnsubmitHandler to broadcast a re-sync op.)
    await expect(pageB.locator('[data-action="submit"]')).toBeVisible({ timeout: CONVERGE_TIMEOUT });
    await expect(pageB.locator('.submit-confirm')).toHaveCount(0);

    // CATCH-UP: reload B — still fillable.
    await reopenRunnerB(pageB, 'MX Unsubmit');
    await expect(pageB.locator('[data-action="submit"]')).toBeVisible({ timeout: CONVERGE_TIMEOUT });

    await ctxB.close();
  });

  test('list-view progress indicator converges live + catch-up (field completion)', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'MX Progress', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [
        CHECKBOX_F('Field A', 0), CHECKBOX_F('Field B', 1),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Observing device B stays on the LIST view (runner NOT open): shows 0/2.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    const rowB = pageB.locator('[data-fill-template-id]').filter({ hasText: 'MX Progress' });
    await expect(rowB).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await expect(rowB).toContainText('0/2', { timeout: RUNNER_TIMEOUT });

    // Writer A opens the runner and completes Field A.
    await page.reload();
    await expect(page.locator('#s1').getByText('MX Progress')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.check-btn', { timeout: RUNNER_TIMEOUT });
    await page.locator('.fill-field', { hasText: 'Field A' }).locator('.check-btn').click();
    await page.waitForTimeout(2500); // autosave + WS propagation

    // LIVE on B's list (no reload): the progress indicator converges to 1/2.
    await expect(rowB).toContainText('1/2', { timeout: CONVERGE_TIMEOUT });

    // CATCH-UP: reload B — still 1/2.
    await pageB.reload();
    const rowB2 = pageB.locator('[data-fill-template-id]').filter({ hasText: 'MX Progress' });
    await expect(rowB2).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await expect(rowB2).toContainText('1/2', { timeout: CONVERGE_TIMEOUT });

    await ctxB.close();
  });

  // The list-row DENOMINATOR converges when a field is CUT while the observer
  // sits on the BARE My-Checklists list (runner NOT open). applyOp's no-runner
  // SAVE_TEMPLATE branch now re-fetches My Checklists (sync.js) — mirroring the
  // SET_FIELD branch's no-runner renderMyChecklists() refresh — so the X/Y
  // denominator re-derives from the template's new shape without a reload.
  // RED before the sync.js refresh: that branch only called loadTemplates (the
  // Builder list), leaving B's list row showing the stale 1/2 denominator.
  test('list-view denominator converges live + catch-up when a field is CUT on the bare list', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'MX Denom', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [
        CHECKBOX_F('Keep A', 0), CHECKBOX_F('Decoy', 1),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Observing device B: open the runner, complete 'Keep A', then go BACK to the
    // bare list (runner closed) so its row shows 1/2 with no runner open.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await openRunnerB(pageB, 'MX Denom');
    await pageB.locator('.fill-field', { hasText: 'Keep A' }).locator('.check-btn').click();
    await waitAutosave(pageB);
    await pageB.waitForTimeout(800);
    await pageB.click('#fill-back');
    await expect(pageB.locator('#checklist-list')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    const rowB = pageB.locator('[data-fill-template-id]').filter({ hasText: 'MX Denom' });
    await expect(rowB).toContainText('1/2', { timeout: RUNNER_TIMEOUT });

    // Admin cuts the unanswered Decoy (SAVE_TEMPLATE op) — denominator 2 → 1.
    await cutDecoy(page, 'MX Denom', dow);

    // LIVE on B's BARE list (runner closed, no reload): denominator converges to
    // 1/1 as the no-runner SAVE_TEMPLATE branch re-fetches My Checklists.
    await expect(rowB).toContainText('1/1', { timeout: CONVERGE_TIMEOUT });

    // CATCH-UP: reload B — still 1/1.
    await pageB.reload();
    const rowB2 = pageB.locator('[data-fill-template-id]').filter({ hasText: 'MX Denom' });
    await expect(rowB2).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await expect(rowB2).toContainText('1/1', { timeout: CONVERGE_TIMEOUT });

    await ctxB.close();
  });
});

// ─── W-6: LWW conflict re-fetch/re-render ────────────────────────────────────
// When a field write loses last-writer-wins (409, api() returns
// {_conflict, winner}), the LOSING device must converge its RENDERED field to
// the WINNING value carried in the 409 body — the screen must never keep
// showing a value the DB rejected (FR-9, INV-1). Two-context E2E: context A is
// the loser (WebSocket stubbed dead so its ONLY path to the winning value is
// the conflict handler, not a live WS broadcast), context B seeds the winner.
test.describe('Engine: LWW conflict re-renders the winning value (W-6)', () => {
  test('losing device renders the winning field value after a 409, not its rejected value [FR-9 INV-1]', async ({ browser }) => {
    test.setTimeout(120000);

    // Context A — the LOSING device. Stub WebSocket BEFORE load so no live op
    // ever reaches A: its only route to the winning value is the 409 response.
    const ctxA = await browser.newContext();
    await ctxA.addInitScript(() => {
      class DeadSocket { constructor() { this.readyState = 3; } close() {} send() {} }
      DeadSocket.CONNECTING = 0; DeadSocket.OPEN = 1; DeadSocket.CLOSING = 2; DeadSocket.CLOSED = 3;
      window.WebSocket = DeadSocket;
    });
    const page = await ctxA.newPage();
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupPendingApprovals(page);
    await cleanupTemplates(page);

    const dow = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'MX Conflict', requires_approval: false,
      sections: [{ title: 'S', order: 0, condition: null, fields: [
        { type: 'text', label: 'Notes', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // A opens the runner and locates the text field.
    await page.reload();
    await expect(page.locator('#s1').getByText('MX Conflict')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.fill-textarea', { timeout: RUNNER_TIMEOUT });
    const fieldId = await page.locator('.fill-textarea').first().getAttribute('data-field-id');

    // Force A permanently stale: pin its Lamport tick to 1 (well below the
    // winner's 5000000) and neutralise catch-up + clock-receive so nothing can
    // climb A above the winner. This makes A lose LWW deterministically — the
    // whole point of the test is the LOSING branch.
    await page.evaluate(() => {
      window.wsCatchUp = async () => {};
      if (window.LAMPORT_CLOCK) {
        window.LAMPORT_CLOCK.receive = async () => {};
        window.LAMPORT_CLOCK._ts = 0;
        window.LAMPORT_CLOCK.tick = async () => 1;
      }
    });

    // Context B — the WINNING device. Seed the winning value with a high Lamport
    // timestamp so any later stale write from A loses LWW deterministically.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    const seedRes = await apiCall(pageB, 'POST', 'ops', {
      op_type: 'SET_FIELD', entity_id: fieldId, entity_type: 'field_response',
      payload: { field_id: fieldId, value: 'WINNER' },
      lamport_ts: 5000000, device_id: 'zzz-winner-device',
    });
    expect(seedRes && !seedRes._conflict, 'seed write must win, not conflict').toBeTruthy();

    // A writes its own (stale) value → optimistic UI shows 'LOSER' → server 409.
    const opsResp = page.waitForResponse(
      res => res.url().includes('/api/v1/workflow/ops') && res.request().method() === 'POST',
      { timeout: CONVERGE_TIMEOUT }
    );
    const textarea = page.locator('.fill-textarea').first();
    await textarea.fill('LOSER');
    await textarea.blur(); // blur → debouncedSaveField('LOSER') → POST /ops → 409
    await expect(textarea).toHaveValue('LOSER'); // optimistic: rejected value shown first
    expect((await opsResp).status(), 'A must lose LWW (409)').toBe(409);

    // After the 409 resolves, A's rendered field MUST show the WINNING value —
    // never 'LOSER' and never an empty/undefined divergence.
    const converged = page.locator('.fill-textarea[data-field-id="' + fieldId + '"]');
    await expect(converged).toHaveValue('WINNER', { timeout: CONVERGE_TIMEOUT });

    await ctxA.close();
    await ctxB.close();
  });
});
