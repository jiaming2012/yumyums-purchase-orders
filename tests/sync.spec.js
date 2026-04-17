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

  test('two contexts see the same checklist from My Checklists', async ({ browser, page }) => {
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

  test('Device B sees field changes from Device A after reload', async ({ browser, page }) => {
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

  test('saving a field generates a SET_FIELD op visible via ops/since', async ({ page }) => {
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

  test('template creation generates SAVE_TEMPLATE op', async ({ page }) => {
    const dow = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Template Op Test', dow);
    await page.waitForTimeout(1500); // EmitOp is async

    const ops = await apiCall(page, 'GET', 'ops/since?lamport_ts=0');
    expect(Array.isArray(ops)).toBe(true);
    const saveOp = ops.find(op => op.op_type === 'SAVE_TEMPLATE' && op.entity_id === tpl.id);
    expect(saveOp).toBeDefined();
    expect(saveOp.entity_type).toBe('template');
  });

  test('archiving template generates ARCHIVE_TEMPLATE op', async ({ page }) => {
    const dow = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Archive Op Test', dow);
    await page.waitForTimeout(1000);

    await apiCall(page, 'DELETE', 'archiveTemplate/' + tpl.id);
    await page.waitForTimeout(1500);

    const ops = await apiCall(page, 'GET', 'ops/since?lamport_ts=0');
    const archiveOp = ops.find(op => op.op_type === 'ARCHIVE_TEMPLATE' && op.entity_id === tpl.id);
    expect(archiveOp).toBeDefined();
  });

  test('multiple ops on same entity have incrementing lamport_ts', async ({ page }) => {
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

  test('checking a field does not uncheck itself via WS echo', async ({ page }) => {
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

  test('no "updated by" toast appears for own field saves', async ({ page }) => {
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

  test('SET_FIELD op includes user_name in payload', async ({ page }) => {
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

  test('save status clears after rapid field saves', async ({ page }) => {
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

  test('field attribution shows user name not undefined after save', async ({ page }) => {
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

  test('unchecked field stays unchecked after navigating away and returning', async ({ page }) => {
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

  test('list page progress decrements when another device unchecks a field', async ({ browser, page }) => {
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

  test('sub-step checks on Device A appear checked on Device B', async ({ browser, page }) => {
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

  test('submit on Device A updates runner view to submitted on Device B', async ({ browser, page }) => {
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

  test('temperature input keeps cursor position while typing', async ({ page }) => {
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

  test('list page progress updates when another device completes a field', async ({ browser, page }) => {
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
  test('ops/since endpoint requires authentication', async ({ browser }) => {
    const ctx = await browser.newContext();
    const page = await ctx.newPage();
    const response = await page.request.get('/api/v1/workflow/ops/since?lamport_ts=0');
    expect(response.status()).toBe(401);
    await ctx.close();
  });

  test('WebSocket endpoint requires authentication', async ({ browser }) => {
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
