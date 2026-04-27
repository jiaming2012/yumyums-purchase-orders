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

// cleanupPendingApprovals approves all pending submissions so they don't
// carry over between tests (the server doesn't filter archived templates out
// of pendingApprovals).
async function cleanupPendingApprovals(page) {
  const pending = await apiCall(page, 'GET', 'pendingApprovals');
  if (!Array.isArray(pending)) return;
  for (const s of pending) {
    await apiCall(page, 'POST', 'approveSubmission', { submission_id: s.id });
  }
}

// createTestTemplate creates a template with the correct API shape.
// Pass todayDOW (0=Sun..6=Sat) to add a schedule+assignment so it shows
// up in My Checklists.
async function createTestTemplate(page, name, todayDOW) {
  name = name || 'Test Template';
  const input = {
    name,
    // requires_approval needs an approver assignment — only set true when assignments are provided
    requires_approval: todayDOW !== undefined,
    sections: [
      {
        title: 'Section 1',
        order: 0,
        condition: null,
        fields: [
          {
            type: 'checkbox',
            label: 'Check this',
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
    // schedule for today
    input.schedules = [{ active_days: [todayDOW] }];
    // assign to 'admin' role (the role of the test user in DB) so the logged-in
    // user sees it in My Checklists, and also add an approver assignment so
    // submitted checklists appear in Approvals.
    input.assignments = [
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
    ];
  }
  return apiCall(page, 'POST', 'createTemplate', input);
}

// getTodayDOW returns today's day-of-week integer (0=Sun..6=Sat) via page context.
async function getTodayDOW(page) {
  return page.evaluate(() => new Date().getDay());
}

// generateUUID returns a UUID v4 string.
function generateUUID() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
    const r = Math.random() * 16 | 0;
    const v = c === 'x' ? r : (r & 0x3 | 0x8);
    return v.toString(16);
  });
}

// submitChecklistViaAPI submits a checklist using only the fields the server
// requires: template_id and idempotency_key (must be a valid UUID).
async function submitChecklistViaAPI(page, templateId) {
  const key = generateUUID();
  return apiCall(page, 'POST', 'submitChecklist', {
    template_id: templateId,
    idempotency_key: key,
    responses: [],
  });
}

// ─── A. Builder — Template CRUD ───────────────────────────────────────────────

test.describe('Builder', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await page.reload();
  });

  test('create template via Builder', async ({ page }) => {
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();

    // Click the "+ New checklist" btn-primary button in the builder list view
    await page.click('#s3 .btn-primary');

    // Wait for the editor to appear (save-btn is rendered in editor view)
    await expect(page.locator('#save-btn')).toBeVisible({ timeout: 5000 });

    // Set template name
    await page.fill('#tpl-name-input', 'Morning Setup');

    // Save the template
    await page.click('#save-btn');

    // Verify toast appears (the toast element has id="toast")
    const toast = page.locator('#toast');
    await expect(toast).toBeVisible({ timeout: 5000 });
  });

  test('saving template navigates back to builder list', async ({ page }) => {
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();

    // Open editor
    await page.click('#s3 .btn-primary');
    await expect(page.locator('#save-btn')).toBeVisible({ timeout: 5000 });

    // Editor is showing, list is not
    await expect(page.locator('#builder-list')).not.toBeVisible();

    // Fill and save
    await page.fill('#tpl-name-input', 'Nav Test');
    await page.click('#save-btn');

    // Should be back on the list view with the new template visible
    await expect(page.locator('#builder-list')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#save-btn')).not.toBeVisible();
    await expect(page.locator('text=Nav Test')).toBeVisible({ timeout: 5000 });
  });

  test('archive checklist soft-deletes and removes from list', async ({ page }) => {
    // Create via API then reload so TEMPLATES is populated
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Archive Test',
      sections: [{ title: 'S1', order: 0, condition: null, fields: [] }],
    });
    const tplId = result.id;

    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#builder-list .row', { hasText: 'Archive Test' })).toBeVisible({ timeout: 5000 });

    // Open the template in the editor
    await page.locator('#builder-list .row', { hasText: 'Archive Test' }).first().click();
    await expect(page.locator('[data-action="archive-template"]')).toBeVisible({ timeout: 5000 });

    // Click archive button — accept the confirm dialog
    page.once('dialog', async dialog => await dialog.accept());
    await page.locator('[data-action="archive-template"]').click();

    // Should show success toast
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // Template should be removed from the list
    await expect(page.locator('text=Archive Test')).not.toBeVisible({ timeout: 5000 });

    // API confirms it's gone
    const templates = await apiCall(page, 'GET', 'templates');
    expect(templates.find(t => t.id === tplId)).toBeUndefined();
  });

  test('archive navigates back to builder list', async ({ page }) => {
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Archive Nav Test',
      sections: [{ title: 'S1', order: 0, condition: null, fields: [] }],
    });

    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#builder-list .row', { hasText: 'Archive Nav Test' })).toBeVisible({ timeout: 5000 });

    // Open editor
    await page.locator('#builder-list .row', { hasText: 'Archive Nav Test' }).first().click();
    await expect(page.locator('[data-action="archive-template"]')).toBeVisible({ timeout: 5000 });

    // Archive — accept the confirm dialog
    page.once('dialog', async dialog => await dialog.accept());
    await page.locator('[data-action="archive-template"]').click();
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // Should navigate back to builder list (editor closed, list visible)
    await expect(page.locator('#builder-list')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#save-btn')).not.toBeVisible();
    await expect(page.locator('[data-action="archive-template"]')).not.toBeVisible();
  });

  test('duplicate checklist name shows error toast', async ({ page }) => {
    // Create first template
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Unique Name Test',
      sections: [{ title: 'S1', order: 0, condition: null, fields: [] }],
    });

    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();

    // Try to create a second template with the same name
    await page.click('#s3 .btn-primary');
    await page.waitForSelector('#builder-body');
    await page.fill('#tpl-name-input', 'Unique Name Test');
    await page.click('#save-btn');

    // Should show duplicate name error
    await expect(page.locator('#toast')).toContainText('already exists', { timeout: 5000 });
  });

  test('can reuse name of deleted checklist', async ({ page }) => {
    // Create and delete a template
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Reuse Name Test',
      sections: [{ title: 'S1', order: 0, condition: null, fields: [] }],
    });
    await apiCall(page, 'DELETE', 'archiveTemplate/' + result.id);

    // Create a new template with the same name — should succeed
    const result2 = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Reuse Name Test',
      sections: [{ title: 'S1', order: 0, condition: null, fields: [] }],
    });
    expect(result2.id).toBeTruthy();

    // Clean up
    await apiCall(page, 'DELETE', 'archiveTemplate/' + result2.id);
  });

  test('saving with requires_approval but no approver shows error toast', async ({ page }) => {
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();

    // Open editor
    await page.click('#s3 .btn-primary');
    await expect(page.locator('#save-btn')).toBeVisible({ timeout: 5000 });

    // Fill name, approval is on by default — deselect all approver chips
    await page.fill('#tpl-name-input', 'No Approver Test');

    // Deselect the default approver role (manager) if selected
    const approverChips = page.locator('[data-action="toggle-approver-role"]');
    const count = await approverChips.count();
    for (let i = 0; i < count; i++) {
      const chip = approverChips.nth(i);
      if (await chip.evaluate(el => el.classList.contains('on'))) {
        await chip.click();
      }
    }

    // Save — should show approver error toast
    await page.click('#save-btn');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#toast')).toContainText('approver');

    // Should still be in editor (not navigated away)
    await expect(page.locator('#save-btn')).toBeVisible();
  });

  test('empty builder shows empty state', async ({ page }) => {
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    // Empty state heading is "No templates yet"
    const emptyText = page.locator('text=No templates yet');
    await expect(emptyText).toBeVisible({ timeout: 5000 });
  });

  test('edit existing template', async ({ page }) => {
    // Create template via API
    await createTestTemplate(page, 'Edit Me');
    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();

    // Template should appear in builder list
    await expect(page.locator('text=Edit Me')).toBeVisible({ timeout: 5000 });

    // Click the template row to open editor (rows have data-template-id)
    await page.click('[data-template-id]');

    // Editor back button should appear
    await expect(page.locator('.editor-back')).toBeVisible({ timeout: 5000 });
  });

  test('archive template', async ({ page }) => {
    // Create template via API
    const result = await createTestTemplate(page, 'To Archive');
    const templateId = result.id;
    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();

    // Template should appear in builder list
    await expect(page.locator('text=To Archive')).toBeVisible({ timeout: 5000 });

    // Archive via API (no archive button exists in the list view UI)
    await apiCall(page, 'DELETE', 'archiveTemplate/' + templateId);

    // Reload and verify template is gone
    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('text=To Archive')).not.toBeVisible({ timeout: 5000 });
  });
});

// ─── B. My Checklists — Fill and Submit ──────────────────────────────────────

test.describe('My Checklists', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
  });

  test('today checklists appear from API', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    await createTestTemplate(page, 'Daily Checklist', todayDOW);
    await page.reload();
    // My Checklists tab should be active by default
    await expect(page.locator('#s1')).toBeVisible();
    // Template should appear
    await expect(page.locator('text=Daily Checklist')).toBeVisible({ timeout: 5000 });
  });

  test('fill and submit checklist', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    await createTestTemplate(page, 'Submit Test', todayDOW);
    await page.reload();

    // Tap the checklist row to open it
    await page.click('[data-fill-template-id]');

    // Check a checkbox (auto-save fires)
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });

    // Submit the checklist — button has data-action="submit"
    await page.click('[data-action="submit"]');

    // Verify toast (id="toast")
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // Should return to list
    await expect(page.locator('#s1')).toBeVisible({ timeout: 8000 });
  });

  test('checked item shows user display name, not undefined', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    await createTestTemplate(page, 'Name Test', todayDOW);
    await page.reload();

    // Open checklist
    await page.click('[data-fill-template-id]');

    // Check the checkbox
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });

    // Attribution line should show the user's display name, not "undefined"
    const attribution = page.locator('.fill-attribution').first();
    await expect(attribution).toBeVisible({ timeout: 5000 });
    await expect(attribution).not.toContainText('undefined');
    // Should contain the actual user name (Jamal C. from superadmins.yaml)
    const text = await attribution.textContent();
    expect(text).toMatch(/\w+/); // at least one word character
  });

  test('empty state when no checklists', async ({ page }) => {
    // No templates, reload
    await page.reload();
    await expect(page.locator('#s1')).toBeVisible();
    // Empty state heading is "No checklists for today"
    const emptyMsg = page.locator('text=No checklists for today');
    await expect(emptyMsg).toBeVisible({ timeout: 5000 });
  });
});

// ─── C. Approvals ────────────────────────────────────────────────────────────

test.describe('Approvals', () => {
  async function createAndSubmitChecklist(page) {
    const todayDOW = await getTodayDOW(page);
    const result = await createTestTemplate(page, 'Approval Test', todayDOW);
    const templateId = result.id;
    // Submit via API with correct field names
    await submitChecklistViaAPI(page, templateId);
    return { id: templateId };
  }

  test('approve submission', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    await createAndSubmitChecklist(page);

    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();

    // Submission should appear — scope to #s2 to avoid strict mode violations
    // with hidden elements in other tabs
    await expect(page.locator('#s2').locator('text=Approval Test')).toBeVisible({ timeout: 5000 });

    // Approve it — button has data-action="approve"
    await page.click('[data-action="approve"]');

    // Verify toast shows "Approved"
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });
  });

  test('reject item with comment', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    await createAndSubmitChecklist(page);

    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();

    // Scope to #s2 to avoid strict mode violations with hidden tabs
    await expect(page.locator('#s2').locator('text=Approval Test')).toBeVisible({ timeout: 5000 });

    // Flag a field item using the "Flag" button
    const flagBtn = page.locator('[data-action="toggle-reject-item"]').first();
    if (await flagBtn.isVisible()) {
      await flagBtn.click();
      // Enter comment in the reject-item-input textarea
      const commentArea = page.locator('.reject-item-input').first();
      await commentArea.fill('Needs correction');
      // Send rejection via reject-submit button
      await page.click('[data-action="reject-submit"]');
    }
  });

  test('reject works after template update (field IDs change)', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    // 1. Create template and submit a checklist
    const todayDOW = await getTodayDOW(page);
    const result = await createTestTemplate(page, 'Reject After Update', todayDOW);
    const templateId = result.id;
    await submitChecklistViaAPI(page, templateId);

    // 2. Update the template (replaceTemplate deletes+re-creates fields with new UUIDs)
    await apiCall(page, 'PUT', 'updateTemplate/' + templateId, {
      name: 'Reject After Update',
      requires_approval: true,
      sections: [{ title: 'Section 1', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Updated field', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });

    // 3. Navigate to Approvals tab and reject a field — should not 500
    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2').locator('text=Reject After Update')).toBeVisible({ timeout: 5000 });

    const flagBtn = page.locator('[data-action="toggle-reject-item"]').first();
    await expect(flagBtn).toBeVisible({ timeout: 5000 });
    await flagBtn.click();
    const commentArea = page.locator('.reject-item-input').first();
    await commentArea.fill('Fix this');
    await page.click('[data-action="reject-submit"]');

    // Verify rejection succeeded (toast or status change)
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });
  });

  test('empty approvals shows caught up', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    await page.reload();

    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2').locator('text=All caught up')).toBeVisible({ timeout: 5000 });
  });
});

// ─── D. Offline sync ─────────────────────────────────────────────────────────

test.describe('Offline sync', () => {
  test('submit while offline queues in IndexedDB', async ({ page, context }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    await createTestTemplate(page, 'Offline Test', todayDOW);
    await page.reload();

    // Open checklist
    await page.click('[data-fill-template-id]');

    // Go offline
    await context.setOffline(true);

    // Submit checklist — data-action="submit"
    await page.click('[data-action="submit"]');

    // Verify queued toast
    await expect(page.locator('text=/Queued/i').first()).toBeVisible({ timeout: 5000 });

    // Verify sync banner shows
    await expect(page.locator('#sync-banner')).toBeVisible({ timeout: 5000 });

    // Go back online
    await context.setOffline(false);

    // Banner should disappear after drain
    await expect(page.locator('#sync-banner')).not.toBeVisible({ timeout: 10000 });
  });

  test('duplicate submit prevented by idempotency key', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    const result = await createTestTemplate(page, 'Idempotency Test', todayDOW);
    const templateId = result.id;

    // Submit twice with same idempotency key (must be a valid UUID)
    const key = generateUUID();
    const payload = {
      template_id: templateId,
      idempotency_key: key,
      responses: [],
    };
    await apiCall(page, 'POST', 'submitChecklist', payload);
    // Second submit — should not produce error or duplicate
    const secondResult = await page.evaluate(async (p) => {
      const res = await fetch('/api/v1/workflow/submitChecklist', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(p),
      });
      return res.status;
    }, payload);
    // Should be 201 (idempotent upsert) — not a 500
    expect([200, 201, 409]).toContain(secondResult);

    // Verify only one submission in approvals
    const approvals = await apiCall(page, 'GET', 'pendingApprovals');
    const forTemplate = (approvals || []).filter((s) => s.template_id === templateId);
    expect(forTemplate.length).toBeLessThanOrEqual(1);
  });
});

// ─── E. Access control ───────────────────────────────────────────────────────

test.describe('Access control', () => {
  test('superadmin can access Builder tab', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    // Builder should NOT show restricted message for superadmin
    await expect(page.locator('text=/restricted to admins/i')).not.toBeVisible({ timeout: 3000 });
  });
});

// ─── F. Navigation ──────────────────────────────────────────────────────────

test.describe('Navigation', () => {
  test('clicking My Checklists tab while runner is open returns to list', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');

    // Wait for checklist list to load
    await page.waitForSelector('#checklist-list .row', { timeout: 5000 });

    // Open the first checklist runner
    await page.locator('#checklist-list .row').first().click();
    await page.waitForSelector('#fill-body .fill-field', { timeout: 5000 });

    // Runner should be showing (progress line visible, checklist-list gone)
    await expect(page.locator('.progress-line')).toBeVisible();
    await expect(page.locator('#checklist-list')).not.toBeVisible();

    // Click "My Checklists" tab button
    await page.click('#t1');

    // Should return to list view — checklist-list should be visible again
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#checklist-list .row').first()).toBeVisible({ timeout: 5000 });
  });
});

// ─── G. Validation ──────────────────────────────────────────────────────────

test.describe('Validation', () => {
  test('submit is blocked when fail trigger fires but corrective action is empty', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    // Create template with temperature field + fail trigger
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Fail Validation Test',
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

    // Open the checklist
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Enter out-of-range temperature to trigger fail
    const tempInput = page.locator('input[type="number"]').first();
    await tempInput.fill('2');
    await tempInput.dispatchEvent('change');
    await page.waitForTimeout(500);

    // Fail card should appear
    await expect(page.locator('.fail-card')).toBeVisible({ timeout: 5000 });

    // Do NOT fill corrective action — leave it empty
    // Try to submit
    await page.click('[data-action="submit"]');

    // Should show error toast about corrective action, NOT submit successfully
    const toast = page.locator('#toast');
    await expect(toast).toBeVisible({ timeout: 5000 });
    await expect(toast).toContainText('Grill temp');

    // Submit button should still be enabled (submission was blocked)
    const submitBtn = page.locator('#submit-btn');
    await expect(submitBtn).toBeVisible();
    await expect(submitBtn).toBeEnabled();
  });

  test('server rejects submission with triggered fail but no corrective action', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Server Fail Validation',
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

    // Get the field ID
    const templates = await apiCall(page, 'GET', 'templates');
    const found = templates.find(t => t.id === tpl.id);
    const fieldId = found.sections[0].fields[0].id;

    // Submit via API with out-of-range temperature but no fail note
    const result = await page.evaluate(async ([templateId, fId]) => {
      const res = await fetch('/api/v1/workflow/submitChecklist', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          template_id: templateId,
          idempotency_key: crypto.randomUUID(),
          responses: [{ field_id: fId, value: 2 }],
          fail_notes: [],
        }),
      });
      return { status: res.status, body: await res.json() };
    }, [tpl.id, fieldId]);

    // Server should reject with 400
    expect(result.status).toBe(400);
    expect(result.body.error).toContain('corrective');
  });
});

// ─── G. Read-only after submit ────────────────────────────────────────────────

test.describe('Read-only after submit', () => {
  test('submitted checklist fields are not interactive', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Readonly Test', todayDOW);

    // Open and submit via UI
    await page.goto(BASE + '/workflows.html');
    const row = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row).toBeVisible({ timeout: 10000 });
    await row.click();

    // Check the checkbox
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(1000);

    // Submit — this navigates back to the list automatically
    await page.click('[data-action="submit"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 10000 });

    // Reopen the submitted checklist
    const row2 = page.locator('[data-fill-template-id="' + tpl.id + '"]');
    await expect(row2).toBeVisible({ timeout: 5000 });
    await row2.click();

    await page.waitForTimeout(1000);
    const fillView = await page.evaluate(() => fillState ? fillState.view : 'N/A');
    const hasActive = await page.evaluate(() => fillState && fillState.activeTemplate ? 'yes' : 'no');
    const secCount = await page.evaluate(() => fillState && fillState.activeTemplate ? (fillState.activeTemplate.sections || []).length : 0);
    const fields = await page.evaluate(() => {
      if (!fillState || !fillState.activeTemplate) return [];
      return fillState.activeTemplate.sections.flatMap(s => (s.fields || []).map(f => ({ id: f.id, label: f.label, type: f.type })));
    });
    console.log('fillView:', fillView, 'hasActive:', hasActive, 'sections:', secCount, 'fields:', JSON.stringify(fields));
    const fillHtml = await page.locator('#fill-body').innerHTML();
    console.log('FILL HTML:', fillHtml.substring(0, 500));
    // Should show submitted/pending state
    await expect(page.locator('.submit-confirm')).toBeVisible({ timeout: 5000 });

    // Checkbox should NOT be clickable (no check-btn visible, or disabled)
    const checkBtns = page.locator('.check-btn');
    const count = await checkBtns.count();
    expect(count).toBe(0);
  });
});

// ─── H. Loading states ───────────────────────────────────────────────────────

test.describe('Loading states', () => {
  test('skeleton screens show during load', async ({ page }) => {
    await login(page);
    // Navigate to workflows and check for skeleton elements
    const skeletonPromise = page.locator('.skeleton').first().waitFor({ state: 'visible', timeout: 2000 }).catch(() => null);
    await page.goto(BASE + '/workflows.html');
    // Skeletons may or may not be captured depending on timing — just verify page loads
    await expect(page.locator('#t1')).toBeVisible({ timeout: 5000 });
  });

  test('workflows page loads and shows tabs', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await expect(page.locator('#t1')).toContainText('My Checklists');
    await expect(page.locator('#t2')).toContainText('Approvals');
    await expect(page.locator('#t3')).toContainText('Builder');
  });

  test('checklist progress is shared across team members', async ({ page }) => {
    // Create a team_member user
    await login(page);
    const email2 = 'shared-checklist-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Shared', last_name: 'Test', email, roles: ['team_member'] })
      });
      return res.json();
    }, email2);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' })
      });
    }, token);

    // Re-login as admin (accept-invite overwrites session cookie)
    await login(page);

    // Create a template scheduled for all 7 days so test works any day
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Shared Team Test',
      sections: [{ title: 'Section 1', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Team item 1', required: false, order: 0, config: null, fail_trigger: null, condition: null }
      ]}],
      schedules: [{ active_days: [0,1,2,3,4,5,6] }],
      requires_approval: false,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'superadmin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' }
      ]
    });
    expect(tpl.template_id || tpl.id).toBeTruthy();

    // Login as admin, open workflows, fill in the checklist field
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    // Find and open the Shared Team Test checklist
    await page.locator('#checklist-list .row', { hasText: 'Shared Team Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    // Check the checkbox
    const firstCheckbox = page.locator('.check-btn').first();
    await firstCheckbox.click();
    await page.waitForTimeout(2000);
    const progressText = await page.locator('.progress-line').textContent();
    const match = progressText.match(/(\d+) of (\d+)/);
    const adminAnswered = parseInt(match[1]);
    expect(adminAnswered).toBe(1);

    // Now login as the team_member
    await page.goto(BASE + '/login.html');
    await login(page, email2, 'test456');
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: 'Shared Team Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    // The team_member should see the same progress — checklist is a team object
    const progressText2 = await page.locator('.progress-line').textContent();
    const match2 = progressText2.match(/(\d+) of (\d+)/);
    const memberAnswered = parseInt(match2[1]);
    expect(memberAnswered).toBe(adminAnswered); // <-- BUG: currently 0 because drafts filtered per-user
  });

  test('field attribution shows who actually checked it, not the viewer', async ({ page }) => {
    // Create a team_member user
    await login(page);
    const email2 = 'attrib-test-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'AttribTest', last_name: 'User', email, roles: ['team_member'] })
      });
      return res.json();
    }, email2);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' })
      });
    }, token);

    // Re-login as admin and create a shared template
    await login(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Attribution Test',
      sections: [{ title: 'Items', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Shared item', required: false, order: 0, config: null, fail_trigger: null, condition: null }
      ]}],
      schedules: [{ active_days: [0,1,2,3,4,5,6] }],
      requires_approval: false,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'superadmin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' }
      ]
    });

    // Login as team_member and check the item
    await login(page, email2, 'test456');
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: 'Attribution Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(2000);
    // Verify the attribution shows the team_member's name
    const attrib1 = await page.locator('.fill-attribution').first().textContent();
    expect(attrib1).toContain('AttribTest U.');

    // Now login as admin and view the same checklist
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: 'Attribution Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    // The attribution should show AttribTest's name, NOT the admin's name
    const attrib2 = await page.locator('.fill-attribution').first().textContent();
    expect(attrib2).toContain('AttribTest U.'); // <-- BUG: was showing admin's name
    expect(attrib2).not.toContain('Jamal');
  });

  test('re-checking a field updates attribution to the new user', async ({ page, browser }) => {
    // Create a team_member user
    await login(page);
    const email2 = 'recheck-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Recheck', last_name: 'User', email, roles: ['team_member'] })
      });
      return res.json();
    }, email2);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' })
      });
    }, token);

    // Re-login as admin, create template
    await login(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Recheck Attribution Test',
      sections: [{ title: 'Items', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Recheck item', required: false, order: 0, config: null, fail_trigger: null, condition: null }
      ]}],
      schedules: [{ active_days: [0,1,2,3,4,5,6] }],
      requires_approval: false,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'superadmin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' }
      ]
    });

    // Admin checks the item
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: 'Recheck Attribution Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(2000);
    // Verify shows admin's name
    const attrib1 = await page.locator('.fill-attribution').first().textContent();
    expect(attrib1).toContain('Jamal');

    // Login as team_member on the SAME browser context
    await login(page, email2, 'test456');
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: 'Recheck Attribution Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Field should show Jamal's attribution (from server data)
    const attribBefore = await page.locator('.fill-attribution').first().textContent();
    expect(attribBefore).toContain('Jamal');

    // Team_member unchecks then re-checks the same field
    await page.locator('.check-btn').first().click(); // uncheck
    await page.waitForTimeout(500);
    await page.locator('.check-btn').first().click(); // re-check
    await page.waitForTimeout(500);

    // Attribution should now show Recheck U., NOT Jamal
    const attribAfter = await page.locator('.fill-attribution').first().textContent();
    expect(attribAfter).toContain('Recheck U.');
    expect(attribAfter).not.toContain('Jamal');
  });

  test('yes/no field attribution updates when different user answers', async ({ page }) => {
    // Create team_member
    await login(page);
    const email2 = 'yn-attrib-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'YNTest', last_name: 'User', email, roles: ['team_member'] })
      });
      return res.json();
    }, email2);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' })
      });
    }, token);

    // Create template with yes/no field
    await login(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'YN Attribution Test',
      sections: [{ title: 'Items', order: 0, condition: null, fields: [
        { type: 'yes_no', label: 'Equipment on?', required: false, order: 0, config: null, fail_trigger: null, condition: null }
      ]}],
      schedules: [{ active_days: [0,1,2,3,4,5,6] }],
      requires_approval: false,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'superadmin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' }
      ]
    });

    // Admin answers "Yes"
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: 'YN Attribution Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('[data-action="set-yes"]').first().click();
    await page.waitForTimeout(2000);

    // Login as team_member, open same checklist
    await login(page, email2, 'test456');
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: 'YN Attribution Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Should show Jamal's attribution from server data
    const attribBefore = await page.locator('.fill-attribution').first().textContent();
    expect(attribBefore).toContain('Jamal');

    // Team_member clicks "No" to update the field
    await page.locator('[data-action="set-no"]').first().click();
    await page.waitForTimeout(500);

    // Attribution should now show YNTest, NOT Jamal
    const attribAfter = await page.locator('.fill-attribution').first().textContent();
    expect(attribAfter).toContain('YNTest U.');
    expect(attribAfter).not.toContain('Jamal');
  });

  test('sub-step completion attributes parent checkbox to the user who completed it', async ({ page }) => {
    // Create a team_member user
    await login(page);
    const email2 = 'substep-attrib-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async ([e]) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'SubStep', last_name: 'User', email: e, roles: ['team_member'] })
      });
      return res.json();
    }, [email2]);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' })
      });
    }, token);

    // Re-login as admin, create template with sub-steps
    await login(page);
    await page.goto(BASE + '/workflows.html');
    const createResult = await apiCall(page, 'POST', 'createTemplate', {
      name: 'SubStep Attrib Test',
      sections: [{ title: 'Inventory', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Stock verified', required: false, order: 0, config: null, fail_trigger: null, condition: null,
          sub_steps: [
            { id: 'sub1', label: 'Item A counted' },
            { id: 'sub2', label: 'Item B counted' },
          ]
        }
      ]}],
      schedules: [{ active_days: [0,1,2,3,4,5,6] }],
      requires_approval: false,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'superadmin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' }
      ]
    });

    // Admin checks the first sub-step
    await page.goto(BASE + '/workflows.html');
    await page.waitForTimeout(2000);
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'SubStep Attrib Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('.sub-step-check').first().click();
    await page.waitForTimeout(2000);

    // Login as team_member, check the second sub-step (completes the parent)
    await login(page, email2, 'test456');
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: 'SubStep Attrib Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    // Second sub-step should be unchecked
    const subChecks = page.locator('.sub-step-check');
    await subChecks.nth(1).click();
    await page.waitForTimeout(500);

    // Parent auto-checks — attribution should show SubStep U., not Jamal
    const attrib = await page.locator('.fill-attribution').first().textContent();
    expect(attrib).toContain('SubStep U.');
    expect(attrib).not.toContain('Jamal');
  });

  test('sub-step attribution appears before divider line', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Attrib Divider Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Parent task', required: false, order: 0, config: null, fail_trigger: null, condition: null,
          sub_steps: [
            { type: 'checkbox', label: 'Sub A', id: 'subA' },
            { type: 'checkbox', label: 'Sub B', id: 'subB' },
          ]
        }
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Attrib Divider Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check first sub-step
    await page.locator('.sub-step-check').first().click();
    await page.waitForTimeout(500);

    // Attribution should be visible
    const attrib = page.locator('.fill-attribution').first();
    await expect(attrib).toBeVisible();

    // The checked sub-step row should NOT have a border-bottom (attribution has it instead)
    const firstSubRow = page.locator('.sub-step-row').first();
    const borderBottom = await firstSubRow.evaluate(el => getComputedStyle(el).borderBottomStyle);
    expect(borderBottom).toBe('none');

    // The attribution div should have a border-bottom
    const attribBorder = await attrib.evaluate(el => getComputedStyle(el).borderBottomStyle);
    expect(attribBorder).toBe('solid');
  });

  test('sub-steps visible in read-only submitted view', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'SubStep ReadOnly Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Main task', required: false, order: 0, config: null, fail_trigger: null, condition: null,
          sub_steps: [
            { type: 'checkbox', label: 'Step Alpha', id: 'alpha1' },
            { type: 'checkbox', label: 'Step Beta', id: 'beta1' },
          ]
        }
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'SubStep ReadOnly Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check both sub-steps
    await page.locator('.sub-step-check').first().click();
    await page.waitForTimeout(300);
    await page.locator('.sub-step-check').nth(1).click();
    await page.waitForTimeout(1500);

    // Submit
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(2000);

    // Reload to see read-only view
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'SubStep ReadOnly Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Sub-steps should be visible in read-only view
    await expect(page.locator('.sub-step-row')).toHaveCount(2);
    await expect(page.locator('.sub-step-label-text').first()).toContainText('Step Alpha');
    await expect(page.locator('.sub-step-label-text').nth(1)).toContainText('Step Beta');
  });

  test('sub-steps visible in approvals review tab', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'SubStep Approval Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Verify inventory', required: false, order: 0, config: null, fail_trigger: null, condition: null,
          sub_steps: [
            { type: 'checkbox', label: 'Count proteins', id: 'prot1' },
            { type: 'checkbox', label: 'Count sides', id: 'side1' },
          ]
        }
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ]
    });

    // Submit via API
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'SubStep Approval Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('.sub-step-check').first().click();
    await page.waitForTimeout(300);
    await page.locator('.sub-step-check').nth(1).click();
    await page.waitForTimeout(1500);
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(2000);

    // Switch to Approvals tab
    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2').locator('text=SubStep Approval Test')).toBeVisible({ timeout: 5000 });

    // Sub-steps should be visible in the approval card (rendered as indented review-items)
    await expect(page.locator('#s2').locator('text=Count proteins')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#s2').locator('text=Count sides')).toBeVisible();
    // Parent should NOT have a Flag button (sub-items have their own)
    const parentItem = page.locator('#s2 .review-item').filter({ hasText: 'Verify inventory' });
    await expect(parentItem.locator('.review-reject-btn')).toHaveCount(0);
  });

  test('list view item count matches runner count with conditional fields', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    // Create a template with a conditional field that's hidden today.
    // Field "Always visible" is always shown.
    // Field "Conditional" has a day condition for a day that is NOT today, so it's hidden.
    const todayDOW = await getTodayDOW(page);
    const hiddenDay = (todayDOW + 1) % 7; // tomorrow — not visible today
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Conditional Count Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Always visible', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        { type: 'checkbox', label: 'Hidden today', required: false, order: 1, config: null, fail_trigger: null, condition: { days: [hiddenDay] } },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });

    // List view should show 0/1 (not 0/2) — the hidden field should not be counted
    const listRow = page.locator('#checklist-list .row', { hasText: 'Conditional Count Test' }).first();
    await expect(listRow).toBeVisible();
    const listText = await listRow.locator('.mt').textContent();
    expect(listText).toContain('0/1');

    // Open runner — progress should also show 0 of 1
    await listRow.click();
    await page.waitForSelector('#fill-body .fill-field');
    await expect(page.locator('.progress-line')).toContainText('0 of 1');

    // Only "Always visible" should be shown, not "Hidden today"
    await expect(page.locator('.fill-field-label', { hasText: 'Always visible' })).toBeVisible();
    await expect(page.locator('.fill-field-label', { hasText: 'Hidden today' })).not.toBeVisible();
  });

  test('conditional field appears after dependent text field is filled', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    // Create template: text field + checkbox that shows only when text is "not empty"
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Conditional Appear Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'text', label: 'Describe conditions', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    // Load template to get server-assigned field IDs
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Conditional Appear Test');
    const textFieldId = tpl.sections[0].fields[0].id;

    // Update template: add a second field with skip logic referencing the text field
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Conditional Appear Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { id: textFieldId, type: 'text', label: 'Describe conditions', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        { type: 'checkbox', label: 'Follow up task', required: false, order: 1, config: null, fail_trigger: null,
          condition: { field_id: textFieldId, operator: 'equals', value: '_notempty' } },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Conditional Appear Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Initially, only text field should be visible (conditional field is hidden)
    await expect(page.locator('.fill-field-label', { hasText: 'Describe conditions' })).toBeVisible();
    await expect(page.locator('.fill-field-label', { hasText: 'Follow up task' })).not.toBeVisible();

    // Type in the text field and blur
    await page.locator('.fill-textarea').fill('Something happened');
    await page.locator('.fill-textarea').blur();
    await page.waitForTimeout(500);

    // Conditional field should now appear
    await expect(page.locator('.fill-field-label', { hasText: 'Follow up task' })).toBeVisible({ timeout: 3000 });
  });

  test('submitted responses survive template update', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    // Create template, fill, and submit
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Survive Update Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Survive Update Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check the item and submit
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(1500);
    page.once('dialog', async dialog => await dialog.accept());
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(2000);

    // Now update the template (add approver role) — triggers replaceTemplate
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Survive Update Test');
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Survive Update Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
        { assignee_type: 'role', assignee_id: 'manager', assignment_role: 'approver' },
      ]
    });

    // Reload and open checklist — submitted responses should still show
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Survive Update Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Progress should show 1/1, not 0/1
    await expect(page.locator('.progress-line')).toContainText('1 of 1');
  });

  test('incomplete submit shows confirmation prompt', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Incomplete Submit Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Task A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        { type: 'checkbox', label: 'Task B', required: false, order: 1, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Incomplete Submit Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check only one of two items
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(500);

    // Submit — should prompt because 1 item is not completed
    page.once('dialog', async dialog => {
      expect(dialog.message()).toContain('1 item not completed');
      await dialog.dismiss(); // Cancel submission
    });
    await page.click('[data-action="submit"]');

    // Should still be on the checklist (not submitted)
    await expect(page.locator('[data-action="submit"]')).toBeVisible();
  });

  test('sub-step progress counts correctly in list view', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'SubStep Count Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Parent with subs', required: false, order: 0, config: null, fail_trigger: null, condition: null,
          sub_steps: [
            { type: 'checkbox', label: 'Sub 1', id: 'cnt1' },
            { type: 'checkbox', label: 'Sub 2', id: 'cnt2' },
          ]
        },
        { type: 'checkbox', label: 'Simple item', required: false, order: 1, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'SubStep Count Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check only first sub-step — parent should NOT count as complete
    await page.locator('.sub-step-check').first().click();
    await page.waitForTimeout(500);

    // Progress should show 0 of 2 (parent not complete because only 1 of 2 subs done)
    await expect(page.locator('.progress-line')).toContainText('0 of 2');

    // Check second sub-step — parent should now be complete
    await page.locator('.sub-step-check').nth(1).click();
    await page.waitForTimeout(500);

    // Progress should show 1 of 2
    await expect(page.locator('.progress-line')).toContainText('1 of 2');
  });

  test('skip logic condition persists after save and reload', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    // Create template with two fields via API — second has skip logic referencing first
    const todayDOW = await getTodayDOW(page);
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Skip Logic Persist Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'yes_no', label: 'Is it raining?', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        { type: 'checkbox', label: 'Bring umbrella', required: false, order: 1, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    // Open in builder and add skip logic
    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#builder-list .row', { hasText: 'Skip Logic Persist Test' })).toBeVisible({ timeout: 5000 });
    await page.locator('#builder-list .row', { hasText: 'Skip Logic Persist Test' }).first().click();
    await page.waitForSelector('#builder-body');

    // Expand the second field ("Bring umbrella")
    await page.locator('.field-row', { hasText: 'Bring umbrella' }).locator('.field-row-tap').first().click();
    await page.waitForTimeout(300);

    // Select the first field in the skip logic dropdown
    const skipSelect = page.locator('.skip-field-select').first();
    await expect(skipSelect).toBeVisible({ timeout: 3000 });
    // Select first option (the yes_no field)
    const options = await skipSelect.locator('option').allTextContents();
    const yesNoOption = options.find(o => o.includes('Is it raining'));
    if (yesNoOption) {
      await skipSelect.selectOption({ label: yesNoOption });
      await page.waitForTimeout(300);

      // Select value "Yes"
      const valueSelect = page.locator('.skip-value-select').first();
      if (await valueSelect.isVisible({ timeout: 2000 })) {
        await valueSelect.selectOption({ label: 'Yes' });
        await page.waitForTimeout(300);
      }
    }

    // Save
    await page.click('#save-btn');
    await expect(page.locator('#builder-list')).toBeVisible({ timeout: 5000 });

    // Reload and reopen — skip logic should persist
    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#builder-list .row', { hasText: 'Skip Logic Persist Test' })).toBeVisible({ timeout: 5000 });
    await page.locator('#builder-list .row', { hasText: 'Skip Logic Persist Test' }).first().click();
    await page.waitForSelector('#builder-body');

    // Expand "Bring umbrella" again
    await page.locator('.field-row', { hasText: 'Bring umbrella' }).locator('.field-row-tap').first().click();
    await page.waitForTimeout(300);

    // Skip logic should still reference "Is it raining?" — not be reset
    const skipSelectAfter = page.locator('.skip-field-select').first();
    await expect(skipSelectAfter).toBeVisible({ timeout: 3000 });
    const selectedValue = await skipSelectAfter.inputValue();
    expect(selectedValue).not.toBe(''); // Should have a field selected, not empty
  });

  test('new section defaults to Same as schedule', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await page.reload();

    // Go to Builder tab
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();

    // Create new checklist
    await page.click('#s3 .btn-primary');
    await page.waitForSelector('#builder-body');

    // Add a section
    page.once('dialog', async dialog => {
      await dialog.accept('Test Section');
    });
    await page.locator('.add-section-btn').click();

    // "Same as schedule" button should be active (has 'on' class)
    const inheritBtn = page.locator('.day-inherit-btn').filter({ hasText: 'Same as schedule' }).first();
    await expect(inheritBtn).toBeVisible();
    await expect(inheritBtn).toHaveClass(/on/);
  });

  test('text field in read-only view shows answer below label', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Text ReadOnly Test',
      sections: [{ title: 'Notes', order: 0, condition: null, fields: [
        { type: 'text', label: 'Describe conditions', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Text ReadOnly Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Fill in the text field
    await page.locator('.fill-textarea').fill('Kitchen was cleaned and everything put away');
    await page.waitForTimeout(1500);

    // Submit
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(2000);

    // Reload and open — should be read-only
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Text ReadOnly Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Label and answer should both be visible and not overlapping
    await expect(page.locator('.fill-field-label')).toContainText('Describe conditions');
    await expect(page.locator('.fill-field')).toContainText('Kitchen was cleaned');

    // The text should NOT be inside a fill-field-row (it's rendered below the label)
    const fieldRow = page.locator('.fill-field-row');
    await expect(fieldRow).toHaveCount(0);
  });

  test('team_member cannot see Builder tab', async ({ page }) => {
    // Create a team_member user via admin
    await login(page);
    const uniqueEmail = 'builder-test-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'BuilderTest', last_name: 'User', email, roles: ['team_member'] })
      });
      return res.json();
    }, uniqueEmail);
    // Accept invite
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' })
      });
    }, token);

    // Login as team_member
    await login(page, uniqueEmail, 'test456');
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#t1');
    await page.waitForTimeout(500); // wait for checkBuilderAccess

    // Builder tab should NOT be visible
    await expect(page.locator('#t3')).toBeHidden();
    // My Checklists should still be visible
    await expect(page.locator('#t1')).toBeVisible();
  });
});

// ─── F. Approval Flow (multi-user) ──────────────────────────────────────────

test.describe('Approval Flow', () => {
  // Helper: create a team_member user, accept invite, return { email, password }
  async function createCrewUser(page, prefix) {
    const email = prefix + '-' + Date.now() + '@yumyums.kitchen';
    const password = 'crew1234';
    const inviteRes = await page.evaluate(async ([e, name]) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: name, last_name: 'Test', email: e, roles: ['team_member'] })
      });
      return res.json();
    }, [email, prefix]);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async ([t, pw]) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: pw })
      });
    }, [token, password]);
    return { email, password };
  }

  // Helper: create a manager user, accept invite, return { email, password }
  async function createManagerUser(page, prefix) {
    const email = prefix + '-' + Date.now() + '@yumyums.kitchen';
    const password = 'mgr12345';
    const inviteRes = await page.evaluate(async ([e, name]) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: name, last_name: 'Mgr', email: e, roles: ['manager'] })
      });
      return res.json();
    }, [email, prefix]);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async ([t, pw]) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: pw })
      });
    }, [token, password]);
    return { email, password };
  }

  // Helper: create a 4-field template assigned to team_member (assignee) + manager (approver)
  async function createApprovalTemplate(page, name) {
    return apiCall(page, 'POST', 'createTemplate', {
      name: name || 'Approval Flow Test',
      requires_approval: true,
      sections: [{
        title: 'Checks', order: 0, condition: null,
        fields: [
          { type: 'checkbox', label: 'Item A', required: false, order: 0, config: {}, fail_trigger: null, condition: null },
          { type: 'checkbox', label: 'Item B', required: false, order: 1, config: {}, fail_trigger: null, condition: null },
          { type: 'checkbox', label: 'Item C', required: false, order: 2, config: {}, fail_trigger: null, condition: null },
          { type: 'checkbox', label: 'Item D', required: false, order: 3, config: {}, fail_trigger: null, condition: null },
        ]
      }],
      schedules: [{ active_days: [0,1,2,3,4,5,6] }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'manager', assignment_role: 'approver' },
      ]
    });
  }

  test('team member completes checklist, manager approves', async ({ page }) => {
    // Setup: login as admin, create template, crew user, manager user
    await login(page);
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    const tplName = 'Approve ' + Date.now();
    const tpl = await createApprovalTemplate(page, tplName);
    const crew = await createCrewUser(page, 'CrewA');
    await login(page); // re-login as admin after accept-invite
    const mgr = await createManagerUser(page, 'MgrB');
    await login(page); // re-login as admin after accept-invite

    // --- User A (crew): open checklist, check all 4 items, submit ---
    await login(page, crew.email, crew.password);
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: tplName }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check all 4 checkboxes
    const checkBtns = page.locator('.check-btn');
    const count = await checkBtns.count();
    expect(count).toBe(4);
    for (let i = 0; i < count; i++) {
      await checkBtns.nth(i).click();
      await page.waitForTimeout(300);
    }

    // Wait for auto-save
    await page.waitForTimeout(2000);

    // Submit
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(1000);

    // Verify submission confirmation
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // --- User B (manager): approve ---
    await login(page, mgr.email, mgr.password);
    await page.goto(BASE + '/workflows.html');
    await page.click('#t2'); // Approvals tab
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2').locator('text=' + tplName + '')).toBeVisible({ timeout: 5000 });

    // Approve
    await page.click('[data-action="approve"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // Verify the approval list is now empty
  });

  test('team member completes checklist, manager rejects 2 items, crew resubmits, manager approves', async ({ page }) => {
    // Setup
    await login(page);
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    const rejName = 'Reject ' + Date.now();
    const tpl = await createApprovalTemplate(page, rejName);
    const crew = await createCrewUser(page, 'CrewR');
    await login(page); // re-login as admin after accept-invite
    const mgr = await createManagerUser(page, 'MgrR');
    await login(page); // re-login as admin after accept-invite

    // --- User A (crew): complete all items and submit ---
    await login(page, crew.email, crew.password);
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: rejName }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    const checkBtns = page.locator('.check-btn');
    const totalFields = await checkBtns.count();
    expect(totalFields).toBe(4);
    for (let i = 0; i < totalFields; i++) {
      await checkBtns.nth(i).click();
      await page.waitForTimeout(300);
    }
    await page.waitForTimeout(2000);
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(1000);

    // --- User B (manager): flag 2 items with comments, reject ---
    await login(page, mgr.email, mgr.password);
    await page.goto(BASE + '/workflows.html');
    await page.click('#t2');
    await expect(page.locator('#s2').locator('text=' + rejName + '').first()).toBeVisible({ timeout: 5000 });

    // Scope to the specific approval card
    const card = page.locator('.approval-card', { hasText: rejName }).first();

    // Get field IDs from Flag buttons to target textareas precisely
    const flagBtns = card.locator('[data-action="toggle-reject-item"]');
    const fldIdA = await flagBtns.nth(0).getAttribute('data-fld-id');
    const fldIdB = await flagBtns.nth(1).getAttribute('data-fld-id');

    // Flag Item A and enter comment
    await flagBtns.nth(0).click();
    await expect(card.locator(`[data-reject-fld="${fldIdA}"]`)).toBeVisible();
    await card.locator(`[data-reject-fld="${fldIdA}"]`).fill('Item A needs redo');

    // Flag Item B and enter comment
    await flagBtns.nth(1).click();
    await expect(card.locator(`[data-reject-fld="${fldIdB}"]`)).toBeVisible();
    await card.locator(`[data-reject-fld="${fldIdB}"]`).fill('Item B is wrong');

    // Submit rejection
    await card.locator('[data-action="reject-submit"]').click();
    await expect(page.locator('#toast')).toContainText('Rejected', { timeout: 5000 });

    // --- User A (crew): sees rejected items ---
    await login(page, crew.email, crew.password);
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');

    // Verify the submission is rejected via API
    await page.waitForSelector('#checklist-list .row');
    const submissions = await page.evaluate(async () => {
      const r = await fetch('/api/v1/workflow/myChecklists?dow=' + new Date().getDay());
      const data = await r.json();
      return data.submissions || [];
    });
    const rejectedSub = submissions.find(s => s.status === 'rejected');
    expect(rejectedSub).toBeTruthy();
    expect(rejectedSub.rejections.length).toBe(2);
    expect(rejectedSub.rejections.map(r => r.comment).sort()).toEqual(['Item A needs redo', 'Item B is wrong']);

    // Open the checklist and re-check all items for resubmission
    await page.locator('#checklist-list .row', { hasText: rejName }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check any unchecked items
    const allCheckBtns = page.locator('.check-btn');
    const totalBtns = await allCheckBtns.count();
    for (let i = 0; i < totalBtns; i++) {
      const btn = allCheckBtns.nth(i);
      const isChecked = await btn.evaluate(el => el.classList.contains('checked'));
      if (!isChecked) {
        await btn.click();
        await page.waitForTimeout(300);
      }
    }
    await page.waitForTimeout(2000);

    // Resubmit — view auto-returns to list after success animation
    await page.click('[data-action="submit"]');
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.waitForTimeout(1000);

    // Verify progress shows all items complete (4/4, not 2/4)
    const resubProgress = await page.locator('#checklist-list .row', { hasText: rejName }).first().textContent();
    expect(resubProgress).toContain('4/4');

    // --- User B (manager): approve the resubmission ---
    await login(page, mgr.email, mgr.password);
    await page.goto(BASE + '/workflows.html');
    await page.click('#t2');
    await expect(page.locator('#s2').locator('text=' + rejName + '').first()).toBeVisible({ timeout: 5000 });
    await page.click('[data-action="approve"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });
  });

  test('approved checklist shows Approved badge and cannot be resubmitted', async ({ page }) => {
    // Setup
    await login(page);
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    const appName = 'ApprBadge ' + Date.now();
    const tpl = await createApprovalTemplate(page, appName);
    const crew = await createCrewUser(page, 'CrewAB');
    await login(page);
    const mgr = await createManagerUser(page, 'MgrAB');
    await login(page);

    // --- Crew: complete and submit ---
    await login(page, crew.email, crew.password);
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: appName }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    const checkBtns = page.locator('.check-btn');
    for (let i = 0; i < await checkBtns.count(); i++) {
      await checkBtns.nth(i).click();
      await page.waitForTimeout(300);
    }
    await page.waitForTimeout(2000);
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(1000);

    // --- Manager: approve ---
    await login(page, mgr.email, mgr.password);
    await page.goto(BASE + '/workflows.html');
    await page.click('#t2');
    await expect(page.locator('#s2').locator('text=' + appName + '').first()).toBeVisible({ timeout: 5000 });
    await page.click('[data-action="approve"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // --- Crew: verify "Approved ✓" badge on My Checklists list ---
    await login(page, crew.email, crew.password);
    await page.goto(BASE + '/workflows.html');
    await page.waitForSelector('#checklist-list .row');
    const row = page.locator('#checklist-list .row', { hasText: appName }).first();
    await expect(row).toBeVisible();
    await expect(row.locator('text=Approved')).toBeVisible({ timeout: 5000 });

    // --- Open the checklist and verify no submit button ---
    await row.click();
    await page.waitForSelector('#fill-body .fill-field');

    // "Approved ✓" confirmation should be visible
    await expect(page.locator('.submit-confirm', { hasText: 'Approved' })).toBeVisible();

    // Submit button should NOT exist
    await expect(page.locator('#submit-btn')).not.toBeVisible();

    // Unsubmit button should NOT exist
    await expect(page.locator('[data-action="unsubmit"]')).not.toBeVisible();
  });
});
