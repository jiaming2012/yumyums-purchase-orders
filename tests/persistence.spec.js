const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:8089';
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

async function cleanupTemplates(page) {
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
    // Block SW to prevent caching stale JS in tests
    await page.route('**/sw.js', route => route.fulfill({ body: '', contentType: 'application/javascript' }));
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

    // Should show pending approval badge (template has requires_approval: true)
    await expect(row2.locator('text=/Pending/i')).toBeVisible({ timeout: 5000 });

    // Open the checklist — field should still be checked
    await row2.click();
    const checkAfter = page.locator('.check-btn').first();
    await expect(checkAfter).toHaveClass(/checked/, { timeout: 5000 });
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

    const checkBtn = page.locator('.check-btn').first();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });

    // Attribution should not show "undefined"
    const attribution = page.locator('.fill-attribution').first();
    await expect(attribution).toBeVisible({ timeout: 5000 });
    await expect(attribution).not.toContainText('undefined');
  });
});
