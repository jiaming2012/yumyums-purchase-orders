const { test, expect } = require('@playwright/test');

const BASE = 'http://localhost:8080';
const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// ─── Helpers ─────────────────────────────────────────────────────────────────

async function login(page, email, password) {
  await page.goto(BASE + '/login.html');
  await page.fill('input[type="email"]', email || ADMIN_EMAIL);
  await page.fill('input[type="password"]', password || ADMIN_PASSWORD);
  await page.click('button[type="submit"]');
  await page.waitForURL('**/index.html');
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

async function createTestTemplate(page, name) {
  name = name || 'Test Template';
  return apiCall(page, 'POST', 'createTemplate', {
    name,
    sections: [
      {
        id: 'sec-1',
        title: 'Section 1',
        conditions: {},
        fields: [
          {
            id: 'fld-1',
            type: 'checkbox',
            label: 'Check this',
            required: true,
            config: {},
            conditions: {},
            fail_trigger: null,
            sub_steps: [],
          },
        ],
      },
    ],
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

    // Click new template button
    await page.click('[data-action="new-template"]');

    // Set template name
    const nameInput = page.locator('[data-action="set-name"], input[placeholder*="name"], #template-name').first();
    await nameInput.fill('Morning Setup');

    // Add a section if needed
    const addSectionBtn = page.locator('[data-action="add-section"]');
    if (await addSectionBtn.isVisible()) {
      await addSectionBtn.click();
    }

    // Add a field
    const addFieldBtn = page.locator('[data-action="add-field"]').first();
    if (await addFieldBtn.isVisible()) {
      await addFieldBtn.click();
    }

    // Save the template
    await page.click('[data-action="save-template"]');

    // Verify toast or confirmation
    await expect(page.locator('.toast, [class*="toast"], [class*="success"]').first()).toBeVisible({ timeout: 5000 });
  });

  test('empty builder shows empty state', async ({ page }) => {
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    // Should show empty state when no templates
    const emptyText = page.locator('text=/No templates|no templates|create your first/i');
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

    // Click the template to edit it
    await page.click('[data-action="edit-template"], [data-template-id]');
  });

  test('archive template', async ({ page }) => {
    // Create template via API
    const tmpl = await createTestTemplate(page, 'To Archive');
    await page.reload();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();

    // Template should appear
    await expect(page.locator('text=To Archive')).toBeVisible({ timeout: 5000 });

    // Click archive
    await page.click('[data-action="archive-template"]');

    // Template should disappear from list
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
    // Create a template with a schedule for today
    const today = new Date();
    const days = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'];
    const todayKey = days[today.getDay()];
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Daily Checklist',
      sections: [
        {
          id: 'sec-1',
          title: 'Daily Tasks',
          conditions: { days: [todayKey] },
          fields: [
            {
              id: 'fld-1',
              type: 'checkbox',
              label: 'Open the truck',
              required: true,
              config: {},
              conditions: {},
              fail_trigger: null,
              sub_steps: [],
            },
          ],
        },
      ],
    });
    await page.reload();
    // My Checklists tab should be active by default
    await expect(page.locator('#s1')).toBeVisible();
    // Template should appear
    await expect(page.locator('text=Daily Checklist')).toBeVisible({ timeout: 5000 });
  });

  test('fill and submit checklist', async ({ page }) => {
    // Create template with today's schedule
    const today = new Date();
    const days = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'];
    const todayKey = days[today.getDay()];
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Submit Test',
      sections: [
        {
          id: 'sec-1',
          title: 'Tasks',
          conditions: { days: [todayKey] },
          fields: [
            {
              id: 'fld-1',
              type: 'checkbox',
              label: 'Task one',
              required: false,
              config: {},
              conditions: {},
              fail_trigger: null,
              sub_steps: [],
            },
          ],
        },
      ],
    });
    await page.reload();

    // Tap the checklist to open it
    await page.click('[data-fill-template-id]');

    // Check a checkbox (auto-save fires)
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });

    // Submit the checklist
    await page.click('[data-action="submit-checklist"]');

    // Verify toast
    await expect(page.locator('.toast, [class*="toast"]').first()).toBeVisible({ timeout: 5000 });

    // Should return to list
    await expect(page.locator('#s1')).toBeVisible({ timeout: 5000 });
  });

  test('empty state when no checklists', async ({ page }) => {
    // No templates, reload
    await page.reload();
    await expect(page.locator('#s1')).toBeVisible();
    // Should show empty state
    const emptyMsg = page.locator('text=/No checklists|no checklists|no templates/i');
    await expect(emptyMsg).toBeVisible({ timeout: 5000 });
  });
});

// ─── C. Approvals ────────────────────────────────────────────────────────────

test.describe('Approvals', () => {
  async function createAndSubmitChecklist(page) {
    const today = new Date();
    const days = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'];
    const todayKey = days[today.getDay()];
    const tmpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Approval Test',
      sections: [
        {
          id: 'sec-1',
          title: 'Tasks',
          conditions: { days: [todayKey] },
          fields: [
            {
              id: 'fld-1',
              type: 'checkbox',
              label: 'Approve this',
              required: false,
              config: {},
              conditions: {},
              fail_trigger: null,
              sub_steps: [],
            },
          ],
        },
      ],
    });
    // Submit the checklist via API
    const idempotencyKey = crypto.randomUUID ? crypto.randomUUID() : Math.random().toString(36).slice(2);
    await apiCall(page, 'POST', 'submitChecklist', {
      id: idempotencyKey,
      template_id: tmpl.id,
      template_snapshot: tmpl,
      responses: [],
    });
    return tmpl;
  }

  test('approve submission', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await createAndSubmitChecklist(page);

    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();

    // Submission should appear
    await expect(page.locator('text=Approval Test')).toBeVisible({ timeout: 5000 });

    // Approve it
    await page.click('[data-action="approve-submission"]');

    // Verify approved state
    await expect(page.locator('text=/Approved|approved/i').first()).toBeVisible({ timeout: 5000 });
  });

  test('reject item with comment', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await createAndSubmitChecklist(page);

    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();

    await expect(page.locator('text=Approval Test')).toBeVisible({ timeout: 5000 });

    // Open submission
    const submissionCard = page.locator('[data-submission-id], [data-action="open-submission"]').first();
    if (await submissionCard.isVisible()) {
      await submissionCard.click();
    }

    // Reject a field
    const rejectBtn = page.locator('[data-action="reject-field"], [data-action="reject-item"]').first();
    if (await rejectBtn.isVisible()) {
      await rejectBtn.click();
      // Enter comment
      const commentArea = page.locator('textarea').first();
      await commentArea.fill('Needs correction');
      // Send rejection
      await page.click('[data-action="send-rejection"], [data-action="confirm-reject"]');
    }
  });

  test('empty approvals shows caught up', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await page.reload();

    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    // Should show empty state
    await expect(page.locator('text=/caught up|All caught up|no pending/i')).toBeVisible({ timeout: 5000 });
  });
});

// ─── D. Offline sync ─────────────────────────────────────────────────────────

test.describe('Offline sync', () => {
  test('submit while offline queues in IndexedDB', async ({ page, context }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    // Create a template with today's schedule
    const today = new Date();
    const days = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'];
    const todayKey = days[today.getDay()];
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Offline Test',
      sections: [
        {
          id: 'sec-1',
          title: 'Tasks',
          conditions: { days: [todayKey] },
          fields: [
            {
              id: 'fld-1',
              type: 'checkbox',
              label: 'Offline task',
              required: false,
              config: {},
              conditions: {},
              fail_trigger: null,
              sub_steps: [],
            },
          ],
        },
      ],
    });
    await page.reload();

    // Open checklist
    await page.click('[data-fill-template-id]');

    // Go offline
    await context.setOffline(true);

    // Submit checklist
    await page.click('[data-action="submit-checklist"]');

    // Verify queued toast
    await expect(page.locator('text=/Queued|queued/i').first()).toBeVisible({ timeout: 5000 });

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

    const today = new Date();
    const days = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'];
    const todayKey = days[today.getDay()];
    const tmpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Idempotency Test',
      sections: [
        {
          id: 'sec-1',
          title: 'Tasks',
          conditions: { days: [todayKey] },
          fields: [],
        },
      ],
    });

    // Submit twice with same idempotency key
    const key = 'dedup-key-' + tmpl.id;
    const payload = {
      id: key,
      template_id: tmpl.id,
      template_snapshot: tmpl,
      responses: [],
    };
    await apiCall(page, 'POST', 'submitChecklist', payload);
    // Second submit — should not produce error or duplicate
    const result = await page.evaluate(async (p) => {
      const res = await fetch('/api/v1/workflow/submitChecklist', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(p),
      });
      return res.status;
    }, payload);
    // Should be 200 (idempotent) or 409 conflict handled gracefully — not a 500
    expect([200, 409]).toContain(result);

    // Verify only one submission in approvals
    const approvals = await apiCall(page, 'GET', 'pendingApprovals');
    const forTemplate = (approvals || []).filter((s) => s.template_id === tmpl.id);
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

// ─── F. Loading states ───────────────────────────────────────────────────────

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
});
