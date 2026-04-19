const { test, expect } = require('@playwright/test');

const BASE = '';
const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// ─── Helpers ─────────────────────────────────────────────────────────────────

async function login(page) {
  await page.goto(BASE + '/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
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

function generateUUID() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

async function getTodayDOW(page) {
  return page.evaluate(() => new Date().getDay());
}

async function createTestTemplate(page, name, todayDOW) {
  name = name || 'Test Template';
  const input = {
    name,
    requires_approval: true,
    sections: [{
      title: 'Section 1', order: 0, condition: null,
      fields: [{ type: 'checkbox', label: 'Check this', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
    }],
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

async function cleanupPendingApprovals(page) {
  const pending = await apiCall(page, 'GET', 'pendingApprovals');
  if (!Array.isArray(pending)) return;
  for (const s of pending) {
    await apiCall(page, 'POST', 'approveSubmission', { submission_id: s.id });
  }
}

async function cleanupTemplates(page) {
  await cleanupPendingApprovals(page);
  const templates = await apiCall(page, 'GET', 'templates');
  if (!Array.isArray(templates)) return;
  for (const t of templates) {
    await apiCall(page, 'DELETE', 'archiveTemplate/' + t.id);
  }
}

// ─── Persistence Tests ──────────────────────────────────────────────────────
//
// These tests verify that data written via the API is durable — a fresh GET
// returns what was saved. Since the Go server is stateless (all state lives
// in Postgres), round-trip persistence through the API proves restart survival.
//
// Add new tests here as features are implemented.
//

test.describe('Persistence', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
  });

  // ─── Templates ──────────────────────────────────────────────────────────

  test('template with sections and fields round-trips through API', async ({ page }) => {
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Round Trip Template',
      requires_approval: false,
      sections: [{
        title: 'Equipment',
        order: 0,
        condition: null,
        fields: [
          { type: 'checkbox', label: 'Generator running', required: true, order: 0, config: {}, fail_trigger: null, condition: null },
          { type: 'temperature', label: 'Grill temp', required: true, order: 1, config: { unit: 'F', min: 300, max: 500 }, fail_trigger: { type: 'out_of_range', min: 300, max: 500 }, condition: null },
        ],
      }],
    });
    expect(result.id).toBeTruthy();

    // Fresh GET — verify full structure persisted
    const templates = await apiCall(page, 'GET', 'templates');
    const found = templates.find(t => t.id === result.id);
    expect(found).toBeTruthy();
    expect(found.name).toBe('Round Trip Template');
    expect(found.sections).toHaveLength(1);
    expect(found.sections[0].title).toBe('Equipment');
    expect(found.sections[0].fields).toHaveLength(2);
    expect(found.sections[0].fields[0].label).toBe('Generator running');
    expect(found.sections[0].fields[1].type).toBe('temperature');
  });

  // ─── Assignments & Approvers ────────────────────────────────────────────

  test('assignments and approvers persist', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Assignment Test',
      requires_approval: true,
      sections: [{
        title: 'S1', order: 0, condition: null,
        fields: [{ type: 'checkbox', label: 'Item', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
      }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'member', assignment_role: 'assignee' },
        { assignee_type: 'user', assignee_id: 'user-123', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
      schedules: [{ active_days: [todayDOW] }],
    });

    // Fresh GET
    const templates = await apiCall(page, 'GET', 'templates');
    const found = templates.find(t => t.id === result.id);

    // Assignments
    expect(found.assignments).toHaveLength(3);
    const assignees = found.assignments.filter(a => a.assignment_role === 'assignee');
    const approvers = found.assignments.filter(a => a.assignment_role === 'approver');
    expect(assignees).toHaveLength(2);
    expect(approvers).toHaveLength(1);
    expect(approvers[0].assignee_id).toBe('admin');

    // Schedules
    expect(found.schedules).toHaveLength(1);
    expect(found.schedules[0].active_days).toContain(todayDOW);
  });

  // ─── Submissions ───────────────────────────────────────────────────────

  test('submitted checklist persists in pending approvals', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Submit Persist',
      requires_approval: true,
      sections: [{
        title: 'S1', order: 0, condition: null,
        fields: [{ type: 'checkbox', label: 'Done', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
      }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
      schedules: [{ active_days: [todayDOW] }],
    });

    await apiCall(page, 'POST', 'submitChecklist', {
      template_id: tpl.id,
      idempotency_key: generateUUID(),
      responses: [],
    });

    // Fresh GET — submission appears in pending
    const approvals = await apiCall(page, 'GET', 'pendingApprovals');
    const found = (approvals || []).find(s => s.template_id === tpl.id);
    expect(found).toBeTruthy();
    expect(found.status).toBe('pending');
  });

  // ─── Approval state ────────────────────────────────────────────────────

  test('approved submission no longer appears in pending', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Approval State',
      requires_approval: true,
      sections: [{
        title: 'S1', order: 0, condition: null,
        fields: [{ type: 'checkbox', label: 'Done', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
      }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
      schedules: [{ active_days: [todayDOW] }],
    });

    const sub = await apiCall(page, 'POST', 'submitChecklist', {
      template_id: tpl.id,
      idempotency_key: generateUUID(),
      responses: [],
    });
    await apiCall(page, 'POST', 'approveSubmission', { submission_id: sub.id });

    // Fresh GET — approved submission is gone from pending
    const approvals = await apiCall(page, 'GET', 'pendingApprovals');
    const found = (approvals || []).find(s => s.template_id === tpl.id);
    expect(found).toBeFalsy();
  });

  // ─── Archive ───────────────────────────────────────────────────────────

  test('archived template does not appear in template list', async ({ page }) => {
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Archive Me',
      requires_approval: false,
      sections: [{
        title: 'S1', order: 0, condition: null,
        fields: [{ type: 'checkbox', label: 'X', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
      }],
    });

    await apiCall(page, 'DELETE', 'archiveTemplate/' + result.id);

    // Fresh GET — archived template is excluded
    const templates = await apiCall(page, 'GET', 'templates');
    const found = (templates || []).find(t => t.id === result.id);
    expect(found).toBeFalsy();
  });

  // ─── Template update ──────────────────────────────────────────────────

  test('updated template reflects changes on next read', async ({ page }) => {
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Original Name',
      requires_approval: false,
      sections: [{
        title: 'S1', order: 0, condition: null,
        fields: [{ type: 'checkbox', label: 'Old', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
      }],
    });

    await apiCall(page, 'PUT', 'updateTemplate/' + result.id, {
      name: 'Updated Name',
      requires_approval: true,
      sections: [{
        title: 'New Section', order: 0, condition: null,
        fields: [
          { type: 'text', label: 'New field', required: true, order: 0, config: {}, fail_trigger: null, condition: null },
        ],
      }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });

    // Fresh GET
    const templates = await apiCall(page, 'GET', 'templates');
    const found = templates.find(t => t.id === result.id);
    expect(found.name).toBe('Updated Name');
    expect(found.requires_approval).toBe(true);
    expect(found.sections[0].title).toBe('New Section');
    expect(found.sections[0].fields[0].label).toBe('New field');
    expect(found.assignments).toHaveLength(1);
  });

  // ─── Section day visibility ────────────────────────────────────────

  test('section with show-on-days matching today is visible in checklist runner', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Day Visible Test',
      requires_approval: false,
      sections: [
        {
          title: 'Everyday Section',
          order: 0,
          condition: { days: [todayDOW] },
          fields: [{ type: 'checkbox', label: 'Visible item', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
        },
        {
          title: 'Hidden Section',
          order: 1,
          condition: { days: [(todayDOW + 1) % 7] },
          fields: [{ type: 'checkbox', label: 'Hidden item', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
        },
        {
          title: 'No Condition Section',
          order: 2,
          condition: null,
          fields: [{ type: 'checkbox', label: 'Always visible', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
        },
      ],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      ],
      schedules: [{ active_days: [todayDOW] }],
    });
    expect(result.id).toBeTruthy();

    // Full page reload to load My Checklists with new template
    // Trigger My Checklists reload without full page navigation
    await page.evaluate(() => { show(1); });
    await page.waitForTimeout(3000);
    const row = page.locator('[data-fill-template-id]').first();
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Section matching today should be visible
    await expect(page.locator('text=Everyday Section')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=Visible item')).toBeVisible();

    // Section NOT matching today should be hidden
    await expect(page.locator('text=Hidden Section')).not.toBeVisible();
    await expect(page.locator('text=Hidden item')).not.toBeVisible();

    // Section with no condition should be visible
    await expect(page.locator('text=No Condition Section')).toBeVisible();
    await expect(page.locator('text=Always visible')).toBeVisible();
  });

  test('section condition days persist through save and reload', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Day Persist Test',
      requires_approval: false,
      sections: [{
        title: 'Weekday Only',
        order: 0,
        condition: { days: [1, 2, 3, 4, 5] },
        fields: [{ type: 'checkbox', label: 'Check', required: false, order: 0, config: {}, fail_trigger: null, condition: null }],
      }],
    });

    // Fresh GET — condition should persist
    const templates = await apiCall(page, 'GET', 'templates');
    const found = templates.find(t => t.id === result.id);
    expect(found.sections[0].condition).toBeTruthy();
    expect(found.sections[0].condition.days).toEqual([1, 2, 3, 4, 5]);
  });

  // ─── Draft response persistence ───────────────────────────────────

  test('checked items persist after page reload', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Draft Persist Test', todayDOW);

    // Go to workflows, open the checklist
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Check the first checkbox
    const firstCheck = page.locator('.check-btn').first();
    await firstCheck.click();
    await expect(firstCheck).toHaveClass(/checked/, { timeout: 5000 });

    // Wait for auto-save to fire
    await page.waitForTimeout(1000);

    // Full page reload (simulates closing and reopening)
    await page.goto(BASE + '/workflows.html');
    await expect(page.locator('#s1')).toBeVisible();

    // Re-open the same checklist
    const row2 = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row2).toBeVisible({ timeout: 10000 });
    await row2.click();

    // First item should still be checked
    const firstCheckAfter = page.locator('.check-btn').first();
    await expect(firstCheckAfter).toHaveClass(/checked/, { timeout: 5000 });
  });

  test('draft progress bar reflects saved items after reload', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Progress Persist Test', todayDOW);

    // Save a draft response via API for the template's field
    const templates = await apiCall(page, 'GET', 'templates');
    const found = templates.find(t => t.id === tpl.id);
    const fieldId = found.sections[0].fields[0].id;
    await apiCall(page, 'POST', 'saveResponse', { field_id: fieldId, value: true });

    // Full page reload to pick up drafts
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });

    // Progress text should show 1 of 1 items (createTestTemplate has 1 field)
    await expect(row.locator('text=1/1')).toBeVisible({ timeout: 5000 });
  });

  // ─── Submitted checklist persistence ──────────────────────────────

  test('submitted checklist shows as submitted after reload', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Submit Reload Test', todayDOW);

    // Open checklist, check item, submit
    await page.goto(BASE + '/workflows.html?t=' + Date.now());
    await page.waitForTimeout(3000);
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(1000);

    await page.click('[data-action="submit"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // Reload — submitted status and checked items should persist
    await page.goto(BASE + '/workflows.html');
    const row2 = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row2).toBeVisible({ timeout: 10000 });

    // Should show pending badge or submitted indicator
    const hasPending = await row2.locator('text=/Pending|Submitted/i').isVisible({ timeout: 5000 }).catch(() => false);
    expect(hasPending).toBe(true);

    // Open the checklist — should show read-only with checkmark
    await row2.click();
    await expect(page.locator('.submit-confirm')).toBeVisible({ timeout: 5000 });
    // In readonly mode, the checked field shows ✓ text instead of check-btn
    await expect(page.locator('text=✓')).toBeVisible({ timeout: 5000 });
  });

  test('submitted checklist fields are not blank after reload', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Fields Reload Test', todayDOW);

    // Submit via API with a response
    const templates = await apiCall(page, 'GET', 'templates');
    const found = templates.find(t => t.id === tpl.id);
    const fieldId = found.sections[0].fields[0].id;

    // Save draft first (auto-save), then submit
    await apiCall(page, 'POST', 'saveResponse', { field_id: fieldId, value: true });
    const key = generateUUID();
    await apiCall(page, 'POST', 'submitChecklist', {
      template_id: tpl.id,
      idempotency_key: key,
      responses: [],
    });

    // Reload and verify field is still checked
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // In readonly mode (submitted), field shows ✓ text instead of interactive check-btn
    await expect(page.locator('.submit-confirm')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('text=✓')).toBeVisible({ timeout: 5000 });

    // Attribution should not show "undefined"
    const attribution = page.locator('.fill-attribution').first();
    await expect(attribution).toBeVisible({ timeout: 5000 });
    await expect(attribution).not.toContainText('undefined');
  });

  // ─── Draft survives back-and-reopen ───────────────────────────────

  test('checked field survives back-to-list and reopen without losing state', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    // Create template with 2 fields
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Back Reopen Test',
      requires_approval: false,
      sections: [{
        title: 'Section 1',
        order: 0,
        condition: null,
        fields: [
          { type: 'checkbox', label: 'Field A', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
          { type: 'checkbox', label: 'Field B', required: false, order: 1, config: {}, fail_trigger: null, condition: null },
        ],
      }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      ],
      schedules: [{ active_days: [todayDOW] }],
    });

    // Open My Checklists
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });

    // Open checklist, check Field A
    await row.click();
    const checkA = page.locator('.check-btn').first();
    await checkA.click();
    await expect(checkA).toHaveClass(/checked/, { timeout: 5000 });
    // Wait for auto-save to fire (400ms debounce + network)
    await page.waitForTimeout(1500);

    // Back to list
    await page.click('#fill-back');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });

    // List should show 1/2 items
    const rowAfterBack = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(rowAfterBack).toBeVisible({ timeout: 5000 });
    await expect(rowAfterBack.locator('text=1/2')).toBeVisible({ timeout: 5000 });

    // Reopen the same checklist
    await rowAfterBack.click();

    // Field A should STILL be checked
    const checkAAfter = page.locator('.check-btn').first();
    await expect(checkAAfter).toHaveClass(/checked/, { timeout: 5000 });

    // Field B should NOT be checked
    const checkBAfter = page.locator('.check-btn').nth(1);
    await expect(checkBAfter).not.toHaveClass(/checked/);

    // Progress should still show 1/2
    const progressText = page.locator('.progress-line');
    await expect(progressText).toContainText('1 of 2');
  });

  // ─── Yes/No button highlight ──────────────────────────────────────

  test('yes/no button is highlighted after save and reopen', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'YesNo Highlight Test',
      requires_approval: false,
      sections: [{
        title: 'Section 1', order: 0, condition: null,
        fields: [
          { type: 'yes_no', label: 'All good?', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
        ],
      }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
      schedules: [{ active_days: [todayDOW] }],
    });

    // Open checklist
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Click "Yes"
    await page.click('[data-action="set-yes"]');
    const yesBtn = page.locator('[data-action="set-yes"]');
    await expect(yesBtn).toHaveClass(/on/, { timeout: 5000 });
    await page.waitForTimeout(1500);

    // Back to list and reopen
    await page.click('#fill-back');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });
    const row2 = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row2).toBeVisible({ timeout: 5000 });
    await row2.click();

    // Yes button should still be highlighted
    const yesBtnAfter = page.locator('[data-action="set-yes"]');
    await expect(yesBtnAfter).toHaveClass(/on/, { timeout: 5000 });

    // No button should NOT be highlighted
    const noBtnAfter = page.locator('[data-action="set-no"]');
    await expect(noBtnAfter).not.toHaveClass(/on/);
  });

  // ─── Text field quote handling ────────────────────────────────────

  test('text field without quotes renders without quotes after reload', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Text No Quotes',
      requires_approval: false,
      sections: [{
        title: 'Notes Section', order: 0, condition: null,
        fields: [
          { type: 'text', label: 'Notes', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
        ],
      }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
      schedules: [{ active_days: [todayDOW] }],
    });

    // Open checklist and type a note
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    const textarea = page.locator('.fill-textarea').first();
    await textarea.fill('hello world');
    await textarea.blur();
    await page.waitForTimeout(1500);

    // Back and reopen
    await page.click('#fill-back');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });
    const row2 = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row2).toBeVisible({ timeout: 5000 });
    await row2.click();

    // Text should be exactly "hello world" — no quotes, no escaping
    const textareaAfter = page.locator('.fill-textarea').first();
    await expect(textareaAfter).toHaveValue('hello world');
  });

  test('text field with quotes renders with quotes after reload', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Text With Quotes',
      requires_approval: false,
      sections: [{
        title: 'Notes Section', order: 0, condition: null,
        fields: [
          { type: 'text', label: 'Notes', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
        ],
      }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
      schedules: [{ active_days: [todayDOW] }],
    });

    // Open checklist and type a note with quotes
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    const textarea = page.locator('.fill-textarea').first();
    await textarea.fill('She said "hello" to me');
    await textarea.blur();
    await page.waitForTimeout(1500);

    // Back and reopen
    await page.click('#fill-back');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });
    const row2 = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row2).toBeVisible({ timeout: 5000 });
    await row2.click();

    // Text should preserve quotes exactly
    const textareaAfter = page.locator('.fill-textarea').first();
    await expect(textareaAfter).toHaveValue('She said "hello" to me');
  });

  // ─── Temperature back-and-reopen ───────────────────────────────────

  test('temperature value survives back-to-list and reopen', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Temp Reopen Test',
      requires_approval: false,
      sections: [{
        title: 'Checks', order: 0, condition: null,
        fields: [
          { type: 'temperature', label: 'Grill temp', required: false, order: 0, config: { unit: 'F', min: 300, max: 500 }, fail_trigger: null, condition: null },
        ],
      }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
      schedules: [{ active_days: [todayDOW] }],
    });

    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Enter temperature
    const tempInput = page.locator('input[type="number"]').first();
    await tempInput.fill('375');
    await tempInput.dispatchEvent('change');
    await page.waitForTimeout(1500);

    // Back to list
    await page.locator('#fill-back').scrollIntoViewIfNeeded();
    await page.click('#fill-back');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });

    // Progress should show 1/1
    const rowAfter = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(rowAfter).toBeVisible({ timeout: 5000 });
    await expect(rowAfter.locator('text=1/1')).toBeVisible({ timeout: 5000 });

    // Reopen
    await rowAfter.click();

    // Temperature should still be 375
    const tempAfter = page.locator('input[type="number"]').first();
    await expect(tempAfter).toHaveValue('375');
  });

  // ─── Sub-steps back-and-reopen ────────────────────────────────────

  test('sub-step checks survive back-to-list and reopen', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'SubStep Reopen Test',
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
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
      schedules: [{ active_days: [todayDOW] }],
    });

    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Check first sub-step (Salmon)
    const subStepBtn = page.locator('.sub-step-check').first();
    await expect(subStepBtn).toBeVisible({ timeout: 5000 });
    await subStepBtn.click();
    await expect(subStepBtn).toHaveClass(/done/, { timeout: 5000 });
    await page.waitForTimeout(1500);

    // Back to list
    await page.locator('#fill-back').scrollIntoViewIfNeeded();
    await page.click('#fill-back');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });

    // Reopen
    const rowAfter = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(rowAfter).toBeVisible({ timeout: 5000 });
    await rowAfter.click();

    // First sub-step should still be checked
    const subStepAfter = page.locator('.sub-step-check').first();
    await expect(subStepAfter).toHaveClass(/done/, { timeout: 5000 });

    // Second sub-step should NOT be checked
    const subStep2After = page.locator('.sub-step-check').nth(1);
    await expect(subStep2After).not.toHaveClass(/done/);
  });

  // ─── Fail note persistence ────────────────────────────────────────

  test('corrective action note and severity survive back-and-reopen', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    // Create template with a temperature field that has a fail trigger
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Fail Note Test',
      requires_approval: false,
      sections: [{
        title: 'Checks', order: 0, condition: null,
        fields: [{
          type: 'temperature', label: 'Grill temp', required: true, order: 0,
          config: { unit: 'F', min: 300, max: 500 },
          fail_trigger: { type: 'out_of_range', min: 300, max: 500 },
          condition: null,
        }],
      }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
      schedules: [{ active_days: [todayDOW] }],
    });

    // Open checklist
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Enter out-of-range temperature to trigger fail card
    const tempInput = page.locator('input[type="number"]').first();
    await tempInput.fill('2');
    await tempInput.dispatchEvent('change');
    await page.waitForTimeout(500);

    // Fail card should appear
    const failCard = page.locator('.fail-card');
    await expect(failCard).toBeVisible({ timeout: 5000 });

    // Fill corrective action note
    const failTextarea = failCard.locator('textarea');
    await failTextarea.fill('Grill needs repair');
    await failTextarea.blur();
    await page.waitForTimeout(500);

    // Select severity (Minor)
    await page.click('[data-action="set-severity"][data-severity="minor"]');
    await page.waitForTimeout(1500);

    // Back to list — scroll to top first to ensure back button is visible
    await page.locator('#fill-back').scrollIntoViewIfNeeded();
    await page.click('#fill-back');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });

    // Reopen
    const row2 = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row2).toBeVisible({ timeout: 5000 });
    await row2.click();

    // Fail card should still be visible (temperature is still out of range)
    const failCardAfter = page.locator('.fail-card');
    await expect(failCardAfter).toBeVisible({ timeout: 5000 });

    // Corrective action note should be preserved
    const failTextareaAfter = failCardAfter.locator('textarea');
    await expect(failTextareaAfter).toHaveValue('Grill needs repair');

    // Severity should still be Minor (highlighted)
    const minorBtn = page.locator('[data-action="set-severity"][data-severity="minor"]');
    await expect(minorBtn).toHaveClass(/on/, { timeout: 5000 });
  });

  test('fail photo survives back-to-list and reopen as https:// URL', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);

    // Create template with a yes/no field — selecting No triggers the fail card
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Fail Photo Test',
      requires_approval: false,
      sections: [{
        title: 'Safety Check', order: 0, condition: null,
        fields: [{
          type: 'yes_no', label: 'Equipment OK?', required: true, order: 0,
          config: {},
          fail_trigger: null,
          condition: null,
        }],
      }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
      schedules: [{ active_days: [todayDOW] }],
    });

    // Get the field ID for later assertions
    const templates = await apiCall(page, 'GET', 'templates');
    const found = templates.find(t => t.id === tpl.id);
    const fieldId = found.sections[0].fields[0].id;
    const fakePublicUrl = 'https://spaces.example.com/checklists/test/fail-' + fieldId + '.jpg';

    // Open checklist
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Mark the yes/no field as No — this triggers the fail card
    const noBtn = page.locator('[data-action="set-no"][data-fld-id="' + fieldId + '"]');
    await expect(noBtn).toBeVisible({ timeout: 5000 });
    await noBtn.click();
    await page.waitForTimeout(300);

    // Fail card should appear
    const failCard = page.locator('.fail-card');
    await expect(failCard).toBeVisible({ timeout: 5000 });

    // Wait for the initial No-answer auto-save debounce to fire (400ms) so it doesn't overwrite
    // the photo we inject next
    await page.waitForTimeout(600);

    // Inject the fail note photo directly via saveResponse to simulate a successful presign upload
    // (camera UI is not available in test environment)
    await apiCall(page, 'POST', 'saveResponse', {
      field_id: fieldId,
      value: { _v: false, _fail_note: { note: '', severity: '', photo: fakePublicUrl } },
    });

    // Wait for the saveResponse to land
    await page.waitForTimeout(200);

    // Back to list
    await page.locator('#fill-back').scrollIntoViewIfNeeded();
    await page.click('#fill-back');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });

    // Reopen the checklist
    const row2 = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row2).toBeVisible({ timeout: 5000 });
    await row2.click();

    // Fail card should still be visible (No answer was persisted)
    const failCardAfter = page.locator('.fail-card');
    await expect(failCardAfter).toBeVisible({ timeout: 5000 });

    // Photo should be shown as an img with https:// src (NOT blob:)
    const photoImg = failCardAfter.locator('img.photo-thumb');
    await expect(photoImg).toBeVisible({ timeout: 5000 });
    const imgSrc = await photoImg.getAttribute('src');
    expect(imgSrc).toMatch(/^https:\/\//);
    expect(imgSrc).not.toMatch(/^blob:/);
    expect(imgSrc).toBe(fakePublicUrl);
  });
});
