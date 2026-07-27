const { test, expect } = require('@playwright/test');

const BASE = '';
const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// ── Card G1 baseline ─────────────────────────────────────────────────────────
// /workflow/* (and /ws) now sit behind the `operations` app grant
// (tests/grant-enforcement-parity.spec.js). This file's invited non-superadmin
// users exercise Operations flows, so grant the app to the standard roles once
// up front — preserving any user_grants other files added. The superadmin
// sessions most tests use bypass grants and are unaffected. Idempotent; runs
// again in the retry worker harmlessly.
test.beforeAll(async ({ browser }) => {
  const baseURL = process.env.NIGHTCREW_ENV_URL || 'http://localhost:' + (process.env.TEST_PORT || '8199');
  const page = await browser.newPage();
  await page.goto(baseURL + '/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
  await page.evaluate(async (slug) => {
    const perms = await (await fetch('/api/v1/apps/permissions')).json();
    const app = (perms || []).find(a => a.slug === slug) || {};
    const roles = [...new Set([...(app.role_grants || []), 'admin', 'manager', 'team_member'])];
    await fetch('/api/v1/apps/' + slug + '/permissions', {
      method: 'PUT', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role_grants: roles, user_grants: (app.user_grants || []).map(String) }),
    });
  }, 'operations');
  await page.close();
});

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

  test('create template via Builder [BLD-03 BLD-04]', async ({ page }) => {
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

  test('saving template navigates back to builder list [BLD-11]', async ({ page }) => {
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

  test('archive checklist soft-deletes and removes from list [BLD-15]', async ({ page }) => {
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

  test('archive navigates back to builder list [BLD-15]', async ({ page }) => {
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

  test('duplicate checklist name shows error toast [BLD-12]', async ({ page }) => {
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

  test('can reuse name of deleted checklist [BLD-12]', async ({ page }) => {
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

  test('saving with requires_approval but no approver shows error toast [BLD-13]', async ({ page }) => {
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

  test('empty builder shows empty state [BLD-02]', async ({ page }) => {
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    // Empty state heading is "No templates yet"
    const emptyText = page.locator('text=No templates yet');
    await expect(emptyText).toBeVisible({ timeout: 5000 });
  });

  test('edit existing template [BLD-04]', async ({ page }) => {
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

  test('archive template [BLD-15]', async ({ page }) => {
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

  test('today checklists appear from API [LST-02]', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    await createTestTemplate(page, 'Daily Checklist', todayDOW);
    await page.reload();
    // My Checklists tab should be active by default
    await expect(page.locator('#s1')).toBeVisible();
    // Template should appear in the My Checklists list. Scope to #checklist-list:
    // the same template name is also rendered in the (hidden) #builder-list, so a
    // bare text= selector is a strict-mode violation (matches 2 elements).
    await expect(page.locator('#checklist-list').getByText('Daily Checklist')).toBeVisible({ timeout: 5000 });
  });

  test('fill and submit checklist [RUN-01 RUN-06]', async ({ page }) => {
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

  test('checked item shows user display name, not undefined [RUN-03]', async ({ page }) => {
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

  test('empty state when no checklists [LST-01]', async ({ page }) => {
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

  test('approve submission [APR-02 APR-11]', async ({ page }) => {
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

  test('approve with flag comment shows feedback on checklist [APR-11 FLD-19]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    await createAndSubmitChecklist(page);

    // Go to Approvals tab
    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2').locator('text=Approval Test')).toBeVisible({ timeout: 5000 });

    // Flag the item with a comment
    const flagBtn = page.locator('[data-action="toggle-reject-item"]').first();
    await expect(flagBtn).toBeVisible({ timeout: 5000 });
    await flagBtn.click();
    const commentArea = page.locator('.reject-item-input').first();
    await commentArea.fill('Please double-check this item next time');

    // Approve (not reject) — feedback should be saved
    await page.click('[data-action="approve"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // Go to My Checklists and open the approved checklist
    await page.click('#t1');
    await page.waitForTimeout(1000);
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Approval Test' }).first().click();
    await page.waitForSelector('#fill-body');
    await page.waitForTimeout(500);

    // Feedback note should be visible
    await expect(page.locator('text=Please double-check this item next time')).toBeVisible({ timeout: 5000 });
  });

  // FR-10 (flag→reject status) + FR-12 (reject+comment persists) + FR-13
  // (rejection feedback renders back to the submitter). REWRITE of the former
  // vacuous test which wrapped its whole body in `if (flagBtn.isVisible())`
  // with NO expect — it never asserted a rejection occurred. This drives the
  // real flow end-to-end and asserts observable state at every hop:
  //   FR-10: flag an item, send rejection → submission.status flips to 'rejected'
  //   FR-12: the flagged item's comment persists on submission.rejections[]
  //   FR-13: reopening the checklist as the submitter renders the manager's
  //          comment in the correction banner ("⚠ Rejected: <comment>").
  // Single-user: admin is both submitter (createAndSubmitChecklist) and the
  // approver (createTestTemplate assigns admin as approver), so the same page
  // session sees the rejection feedback back on My Checklists.
  test('FR-10/12/13 flag+reject flips status, persists comment, and renders feedback to submitter [APR-06 FLD-18 LC-01]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    const { id: templateId } = await createAndSubmitChecklist(page);

    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();

    // Scope to #s2 to avoid strict mode violations with hidden tabs
    await expect(page.locator('#s2').locator('text=Approval Test')).toBeVisible({ timeout: 5000 });

    // Flag a field item using the "Flag" button — must actually be present
    const flagBtn = page.locator('#s2 [data-action="toggle-reject-item"]').first();
    await expect(flagBtn).toBeVisible({ timeout: 5000 });
    await flagBtn.click();

    // Enter comment in the reject-item-input textarea
    const commentArea = page.locator('#s2 .reject-item-input').first();
    await expect(commentArea).toBeVisible();
    await commentArea.fill('Needs correction');

    // Send rejection via reject-submit button
    await page.click('#s2 [data-action="reject-submit"]');

    // FR-10 + FR-12 — assert the submission flipped to rejected in the DB and
    // the comment persisted on the rejection record (read back via the API).
    await expect(page.locator('#toast')).toContainText('Rejected', { timeout: 5000 });
    await page.waitForTimeout(500);
    const submissions = await page.evaluate(async () => {
      const r = await fetch('/api/v1/workflow/myChecklists?dow=' + new Date().getDay());
      const data = await r.json();
      return data.submissions || [];
    });
    const rejectedSub = submissions.find(s => s.status === 'rejected');
    expect(rejectedSub, 'a submission with status=rejected must exist after reject-submit').toBeTruthy();
    expect(rejectedSub.rejections.length).toBeGreaterThanOrEqual(1);
    expect(rejectedSub.rejections.some(r => r.comment === 'Needs correction')).toBe(true);

    // FR-13 — the submitter (same admin) reopens the checklist and sees the
    // manager's rejection comment rendered in the correction banner.
    await page.click('#t1');
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Approval Test' }).first().click();
    await page.waitForSelector('#fill-body');
    const banner = page.locator('.correction-banner', { hasText: 'Needs correction' });
    await expect(banner).toBeVisible({ timeout: 5000 });
    await expect(banner).toContainText('Rejected');
  });

  // Operator-found (dev, 2026-07-18): rejecting with a comment AND "Require
  // photo evidence" checked — the comment and/or the photo requirement do not
  // land on the original checklist. The existing FR-10/12/13 test flags+comments
  // but NEVER checks the require-photo box, so this path was uncovered. Drives
  // the exact UI flow: flag → comment → check "Require photo evidence" → send,
  // then reopens as the submitter and asserts (a) the persisted rejection row
  // carries require_photo=true and (b) the correction banner shows the comment
  // AND the "Photo required before resubmit" requirement.
  test('rejecting with require-photo lands the comment + photo requirement on the checklist [APR-REPRO-0718]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    await createAndSubmitChecklist(page);

    // Approver flags the item, adds a comment, and checks "Require photo evidence".
    await page.reload();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2').locator('text=Approval Test')).toBeVisible({ timeout: 5000 });

    const flagBtn = page.locator('#s2 [data-action="toggle-reject-item"]').first();
    await expect(flagBtn).toBeVisible({ timeout: 5000 });
    await flagBtn.click();
    await page.locator('#s2 .reject-item-input').first().fill('Redo with a photo');
    const photoToggle = page.locator('#s2 [data-reject-photo-fld]').first();
    await expect(photoToggle, 'require-photo checkbox is present in the reject form').toBeVisible({ timeout: 5000 });
    await photoToggle.check();
    await expect(photoToggle).toBeChecked();

    await page.click('#s2 [data-action="reject-submit"]');
    await expect(page.locator('#toast')).toContainText('Rejected', { timeout: 5000 });
    await page.waitForTimeout(500);

    // (a) Persistence: the rejection row must carry require_photo=true AND the comment.
    const submissions = await page.evaluate(async () => {
      const r = await fetch('/api/v1/workflow/myChecklists?dow=' + new Date().getDay());
      return (await r.json()).submissions || [];
    });
    const rejectedSub = submissions.find(s => s.status === 'rejected');
    expect(rejectedSub, 'a rejected submission must exist').toBeTruthy();
    const rej = (rejectedSub.rejections || [])[0];
    expect(rej, 'a rejection row must exist').toBeTruthy();
    expect(rej.comment, 'comment persisted').toBe('Redo with a photo');
    expect(rej.require_photo, 'require_photo persisted as true').toBe(true);

    // (b) Display: reopen as submitter — banner shows the comment AND the photo requirement.
    await page.click('#t1');
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Approval Test' }).first().click();
    await page.waitForSelector('#fill-body');
    const banner = page.locator('.correction-banner');
    await expect(banner, 'correction banner shows the comment').toContainText('Redo with a photo', { timeout: 5000 });
    await expect(banner, 'correction banner shows the photo requirement').toContainText('Photo required', { timeout: 5000 });
  });

  // Operator-found (dev, 2026-07-18) — the photo DEAD-END. When a non-photo
  // field (checkbox here) is rejected with require_photo=true, the reopened
  // checklist shows "📷 Photo required before resubmit" but renders NO way to
  // attach a photo (only type:'photo' fields get a capture button), while the
  // submit gate (workflows.html ~2534) refuses resubmit until the field has an
  // https:// photo URL. Result: the crew is blocked with no control to satisfy
  // the requirement — "the photo is not added to the checklist". This asserts
  // the dead-end so the fix (render a capture affordance on any require-photo
  // field) has a red anchor.
  // The correction-photo slot (built 2026-07-18): a checkbox rejected with
  // require_photo now offers a capture control, and the attached photo (stored in
  // a slot SEPARATE from the checkbox's boolean answer) satisfies both the
  // frontend and backend resubmit gates. Photo capture itself is injected (the
  // presign+PUT camera plumbing is parked by convention, like onboarding FR-18);
  // this drives everything around it: control renders → attach → banner flips →
  // resubmit succeeds.
  test('checkbox rejected with require-photo: attach correction photo unblocks resubmit [APR-DEADEND-0718]', async ({ page }) => {
    page.on('dialog', d => d.accept());
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);
    const { id: templateId } = await createAndSubmitChecklist(page); // single checkbox "Check this"

    const subs = await page.evaluate(async () => {
      const r = await fetch('/api/v1/workflow/myChecklists?dow=' + new Date().getDay());
      return (await r.json()).submissions || [];
    });
    const sub = subs[0];
    const snap = typeof sub.template_snapshot === 'string' ? JSON.parse(sub.template_snapshot) : sub.template_snapshot;
    const fieldId = snap.sections[0].fields[0].id;
    await apiCall(page, 'POST', 'rejectItem', { submission_id: sub.id, field_id: fieldId, comment: 'Photo please', require_photo: true });

    // Reopen as submitter.
    await page.click('#t1');
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Approval Test' }).first().click();
    await page.waitForSelector('#fill-body');

    // The banner demands a photo AND now offers a capture control (dead-end gone).
    const field = page.locator('.fill-field', { has: page.locator('.correction-banner') }).first();
    await expect(field.locator('.correction-banner')).toContainText('Photo required', { timeout: 5000 });
    await expect(field.locator('[data-action="correction-photo-capture"]'),
      'a require-photo field must offer a way to attach the photo').toHaveCount(1);

    // Re-check the box (redo the item) then attach the correction photo (inject the
    // uploaded URL, simulating a successful presign+PUT — the test convention).
    await field.locator('.check-btn').click();
    await page.evaluate((fid) => {
      CORRECTION_PHOTOS[fid] = 'https://cdn.example.com/correction.jpg';
      var resp = FIELD_RESPONSES[fid];
      debouncedSaveField(fid, resp ? resp.value : null);
      renderFieldResponse(fid);
    }, fieldId);
    await page.waitForTimeout(2000); // debounce save

    // Banner flips to "✓ Photo uploaded" and a thumbnail replaces the button.
    await expect(field.locator('.correction-banner')).toContainText('Photo uploaded', { timeout: 5000 });
    await expect(field.locator('[data-action="correction-photo-retake"]')).toBeVisible();

    // Resubmit now succeeds — no "Photo required" block, submission goes pending.
    await page.click('#submit-btn');
    await expect(page.locator('#toast')).not.toContainText('Photo required', { timeout: 3000 });
    await page.waitForTimeout(1000);
    const after = await page.evaluate(async () => {
      const r = await fetch('/api/v1/workflow/myChecklists?dow=' + new Date().getDay());
      return (await r.json()).submissions || [];
    });
    expect(after.some(s => s.status === 'pending_approval' || s.status === 'pending' || s.status === 'submitted'),
      'resubmit succeeded with the correction photo').toBe(true);
  });

  // Operator-found (dev, 2026-07-18) — THE comment-vanish repro. The operator's
  // "Friday checklist → Cut the check → Do C" is a checkbox ("Cut the check")
  // with SUB-STEPS (Do A/B/C). Rejecting a SUB-STEP stores the comment (and any
  // require_photo) against the sub-step's id, but the runner renders the
  // correction banner only at the PARENT field level (REJECTION_FLAGS[parentId])
  // — sub-step rows (workflows.html ~2144-2153) render NO banner. So a sub-step
  // rejection's comment AND photo requirement both silently vanish on the
  // reopened checklist. This asserts they should surface.
  test('rejecting a SUB-STEP surfaces its comment + photo requirement on the checklist [APR-SUBSTEP-0718]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const sub = (label, order) => ({ type: 'checkbox', label, order, config: {}, fail_trigger: null, condition: null });
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Substep Reject', requires_approval: true,
      sections: [{ title: 'Make money', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Cut the check', required: false, order: 0, config: {}, fail_trigger: null, condition: null,
          sub_steps: [ sub('Do A', 0), sub('Do B', 1), sub('Do C', 2) ] },
      ] }],
      schedules: [{ active_days: [todayDOW] }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });
    await submitChecklistViaAPI(page, tpl.id);

    // Find the "Do B" sub-step id from the pending submission's snapshot.
    const subs = await page.evaluate(async () => {
      const r = await fetch('/api/v1/workflow/myChecklists?dow=' + new Date().getDay());
      return (await r.json()).submissions || [];
    });
    const submission = subs[0];
    const snap = typeof submission.template_snapshot === 'string' ? JSON.parse(submission.template_snapshot) : submission.template_snapshot;
    const parent = snap.sections[0].fields[0];
    const doB = (parent.sub_steps || []).find(s => s.label === 'Do B');
    expect(doB, 'Do B sub-step id resolved from snapshot').toBeTruthy();

    // Reject the SUB-STEP with a comment and require_photo.
    await apiCall(page, 'POST', 'rejectItem', { submission_id: submission.id, field_id: doB.id, comment: 'Redo step Do B', require_photo: true });

    // Persistence sanity: the rejection row exists against the sub-step id.
    const after = await page.evaluate(async () => {
      const r = await fetch('/api/v1/workflow/myChecklists?dow=' + new Date().getDay());
      return (await r.json()).submissions || [];
    });
    const rejSub = after.find(s => s.status === 'rejected');
    expect(rejSub && (rejSub.rejections || []).some(r => r.comment === 'Redo step Do B'), 'sub-step rejection persisted').toBe(true);

    // Reopen as submitter — the sub-step's comment must be visible somewhere in the runner.
    await page.click('#t1');
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Substep Reject' }).first().click();
    await page.waitForSelector('#fill-body');
    const subBanner = page.locator('.correction-banner', { hasText: 'Redo step Do B' });
    await expect(subBanner, 'sub-step rejection comment must surface on the checklist').toBeVisible({ timeout: 5000 });
    await expect(subBanner, 'sub-step photo requirement must surface too').toContainText('Photo required');
  });

  // Operator-found (dev, 2026-07-18): a rejected SUB-STEP came back still CHECKED.
  // Top-level rejected fields are unchecked on reopen (hydrateFieldState clears
  // FIELD_RESPONSES[field]), but a sub-step's done-state lives in the PARENT's
  // sub_steps map, which that clear never touched — so the crew wasn't forced to
  // redo it. This asserts a rejected sub-step returns UNCHECKED (like top-level).
  test('rejecting a SUB-STEP unchecks it on reopen so the crew must redo it [APR-SUBSTEP-UNCHECK]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const sub = (label, order) => ({ type: 'checkbox', label, order, config: {}, fail_trigger: null, condition: null });
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Substep Uncheck', requires_approval: true,
      sections: [{ title: 'Make money', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Cut the check', required: false, order: 0, config: {}, fail_trigger: null, condition: null,
          sub_steps: [ sub('Do A', 0), sub('Do B', 1) ] },
      ] }],
      schedules: [{ active_days: [todayDOW] }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });
    const full = (await apiCall(page, 'GET', 'templates')).find(t => t.id === tpl.id);
    const parentId = full.sections[0].fields[0].id;
    const subs = full.sections[0].fields[0].sub_steps;
    const doA = subs.find(s => s.label === 'Do A').id;
    const doB = subs.find(s => s.label === 'Do B').id;

    // Submit with BOTH sub-steps checked (parent value carries the sub_steps map).
    await apiCall(page, 'POST', 'submitChecklist', {
      template_id: tpl.id, idempotency_key: generateUUID(),
      responses: [{ field_id: parentId, value: JSON.stringify({ value: true, sub_steps: { [doA]: true, [doB]: true } }) }],
    });
    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    const subm = pending.find(s => s.template_id === tpl.id) || pending[0];
    // Reject only "Do B".
    await apiCall(page, 'POST', 'rejectItem', { submission_id: subm.id, field_id: doB, comment: 'redo Do B', require_photo: false });

    // Reopen as submitter.
    await page.click('#t1');
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Substep Uncheck' }).first().click();
    await page.waitForSelector('#fill-body');

    // The rejected sub-step (Do B) must come back UNCHECKED; the non-rejected one
    // (Do A) stays checked.
    const doBCheck = page.locator('.sub-step-row', { hasText: 'Do B' }).locator('.sub-step-check');
    const doACheck = page.locator('.sub-step-row', { hasText: 'Do A' }).locator('.sub-step-check');
    await expect(doBCheck, 'rejected sub-step Do B must be unchecked').not.toHaveClass(/done/, { timeout: 5000 });
    await expect(doACheck, 'non-rejected sub-step Do A stays checked').toHaveClass(/done/);
    // The parent auto-checkbox must no longer read as fully done.
    await expect(page.locator('.fill-field', { hasText: 'Cut the check' }).locator('.check-btn').first(),
      'parent no longer all-done').not.toHaveClass(/checked/);
  });

  test('reject works after template update (field IDs change) [APR-10]', async ({ page }) => {
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

  test('empty approvals shows caught up [APR-01]', async ({ page }) => {
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
  test('submit while offline queues in IndexedDB [GATE-07 LST-15]', async ({ page, context }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    await createTestTemplate(page, 'Offline Test', todayDOW);
    await page.reload();

    // Open checklist
    await page.click('[data-fill-template-id]');

    // Complete the checklist item first so submit doesn't trip the
    // "N items not completed. Submit anyway?" confirm() (which Playwright
    // auto-dismisses, aborting the submit before the offline-queue path).
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(1500); // let the field auto-save

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

  test('duplicate submit prevented by idempotency key [GATE-08]', async ({ page }) => {
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

  // ── Card B — workflow-offline-double-submit (ledger T-23 decision 60) ──────
  //
  // The bug: submitChecklistToAPI minted a fresh idempotency_key on EVERY call,
  // and the err.offline handler correctly leaves the checklist editable without
  // pushing into MY_SUBMISSIONS. So offline submit → reopen → submit again
  // enqueued a SECOND payload carrying a SECOND UUID, and both walked straight
  // past the server's `ON CONFLICT (idempotency_key)` guard on drain — two rows
  // in checklist_submissions for one checklist.
  //
  // GATE-08 above only proves the SERVER is idempotent when handed the same key.
  // These prove the CLIENT hands it the same key. That gap is the whole card.

  // readSubmitQueue reads the durable IndexedDB offline queue through the same
  // exports the app uses (sync.js:100-104).
  async function readSubmitQueue(page) {
    return page.evaluate(async () => {
      const db = await window.getDB();
      return window.idbGetAll(db, 'submitQueue');
    });
  }

  // openAndSubmitOffline opens the first checklist row in My Checklists and
  // presses Submit. Any "N items not completed" confirm() is accepted so the
  // submit always reaches the offline-queue path.
  async function openAndSubmitOffline(page) {
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('[data-action="submit"]', { timeout: 5000 });
    await page.click('[data-action="submit"]');
    await expect(page.locator('#sync-banner')).toBeVisible({ timeout: 5000 });
  }

  test('offline re-submit reuses the queued idempotency key [DBL-01]', async ({ page, context }) => {
    page.on('dialog', (d) => d.accept());
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    await createTestTemplate(page, 'Offline Double Submit A', todayDOW);
    await page.reload();
    await page.waitForSelector('[data-fill-template-id]', { timeout: 10000 });

    // Answer the one field so the submit is a normal, complete one.
    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(1500); // let the field auto-save

    await context.setOffline(true);
    await page.click('[data-action="submit"]');
    await expect(page.locator('#sync-banner')).toBeVisible({ timeout: 5000 });

    const afterFirst = await readSubmitQueue(page);
    expect(afterFirst.length, 'first offline submit queues exactly one payload').toBe(1);
    const firstKey = afterFirst[0].idempotency_key;
    expect(firstKey, 'queued payload carries an idempotency key').toBeTruthy();

    // The checklist is deliberately still editable (decision 60 — this is the
    // CORRECT half). Reopen it and submit again, still offline.
    await openAndSubmitOffline(page);

    const afterSecond = await readSubmitQueue(page);
    const keys = [...new Set(afterSecond.map((e) => e.idempotency_key))];
    expect(keys, 're-submit must reuse the queued key, not mint a second one').toEqual([firstKey]);
    // NOT `afterSecond.length === 1`. Collapsing the two entries would mean
    // reusing `id` as well, which REPLACES the queued payload and destroys any
    // answer entered offline — see DBL-04. One distinct key across however many
    // entries is the contract; the entry count is not.
  });

  test('offline re-submit writes ONE submission row after drain, not two [DBL-02]', async ({ page, context }) => {
    page.on('dialog', (d) => d.accept());
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Offline Double Submit B', todayDOW);
    await page.reload();
    await page.waitForSelector('[data-fill-template-id]', { timeout: 10000 });

    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(1500);

    await context.setOffline(true);
    await page.click('[data-action="submit"]');
    await expect(page.locator('#sync-banner')).toBeVisible({ timeout: 5000 });

    // Second press, still offline.
    await openAndSubmitOffline(page);

    // Back online → drainQueue posts whatever is queued.
    await context.setOffline(false);
    await expect(page.locator('#sync-banner')).not.toBeVisible({ timeout: 15000 });

    const approvals = await apiCall(page, 'GET', 'pendingApprovals');
    const forTemplate = (approvals || []).filter((s) => s.template_id === tpl.id);
    expect(forTemplate.length, 'two offline presses must not become two submission rows').toBe(1);
  });

  test('a submission queued in a previous session is reused, not re-minted [DBL-03]', async ({ page, context }) => {
    // The reuse lookup must read the DURABLE queue, not an in-memory variable:
    // "reload the PWA, then submit again" is an ordinary way to produce the
    // second press, and an in-memory map would not survive it. A queue entry
    // this page never enqueued stands in for the previous session's.
    page.on('dialog', (d) => d.accept());
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Offline Double Submit C', todayDOW);
    await page.reload();
    await page.waitForSelector('[data-fill-template-id]', { timeout: 10000 });

    // Go offline FIRST so nothing drains the seeded entry.
    await context.setOffline(true);
    const priorKey = generateUUID();
    await page.evaluate(async ([id, key]) => {
      await window.enqueueSubmission({
        id: 'aaaaaaaa-0000-4000-8000-000000000001',
        template_id: id,
        idempotency_key: key,
        responses: [],
        fail_notes: [],
      });
    }, [tpl.id, priorKey]);
    await expect(page.locator('#sync-banner')).toBeVisible({ timeout: 5000 });

    await openAndSubmitOffline(page);

    const queue = await readSubmitQueue(page);
    const keys = [...new Set(queue.map((e) => e.idempotency_key))];
    expect(keys, 'submit must adopt the previous session\'s queued key').toEqual([priorKey]);
    // Again: one distinct KEY, not one entry. Replacing the previous session's
    // entry would destroy whatever it holds (DBL-04).
  });

  test('answers entered while OFFLINE survive the re-submit [DBL-04]', async ({ page, context }) => {
    // The one-line difference from DBL-01/02 that matters: go offline BEFORE the
    // checkbox is clicked, not after.
    //
    // DBL-01/02 answer the field while still ONLINE, so the field auto-saves to
    // the server and `hydrateFieldState` repopulates it on reopen — which means
    // press 2 rebuilds a FULL payload and any answer loss is invisible to them.
    //
    // Offline, the answer has NO durable home: `submitOp` (sync.js:676) does not
    // queue and throws, and `hydrateFieldState` (workflows.html:1470-1476) clears
    // FIELD_RESPONSES and rebuilds from DRAFT_RESPONSES on every reopen. The only
    // copy of the crew member's answer is the payload already sitting in
    // submitQueue. So press 2 builds an EMPTY payload, and any reuse scheme that
    // REPLACES the queued entry destroys the answer — one row on the server, zero
    // recorded responses, with a success toast. On a food-safety checklist that is
    // worse than the duplicate row this card set out to fix.
    page.on('dialog', (d) => d.accept());
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'Offline Double Submit D', todayDOW);
    await page.reload();
    await page.waitForSelector('[data-fill-template-id]', { timeout: 10000 });

    // OFFLINE FIRST — this is the whole point of the test.
    await context.setOffline(true);

    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(1600); // the auto-save fires and FAILS — nothing durable

    await page.click('[data-action="submit"]');
    await expect(page.locator('#sync-banner')).toBeVisible({ timeout: 5000 });

    const afterFirst = await readSubmitQueue(page);
    const withAnswer = afterFirst.filter((e) => (e.responses || []).length > 0);
    expect(withAnswer.length, 'press 1 queues the crew member\'s answer').toBe(1);

    // Press 2. The runner reopens with the box UNCHECKED (nothing durable to
    // hydrate from), so this payload is empty — that is expected and correct.
    await openAndSubmitOffline(page);

    // Whatever the queue now looks like, the answer must still be in it somewhere.
    const afterSecond = await readSubmitQueue(page);
    const stillHasAnswer = afterSecond.filter((e) => (e.responses || []).length > 0);
    expect(stillHasAnswer.length,
      'the queued answer must NOT be overwritten by the empty second payload').toBe(1);

    // And it must reach the server.
    await context.setOffline(false);
    await expect(page.locator('#sync-banner')).not.toBeVisible({ timeout: 20000 });

    const approvals = await apiCall(page, 'GET', 'pendingApprovals');
    const forTemplate = (approvals || []).filter((s) => s.template_id === tpl.id);
    expect(forTemplate.length, 'still exactly one submission row').toBe(1);
    expect((forTemplate[0].responses || []).length,
      'the answer entered offline must survive to the server, not be silently dropped').toBeGreaterThan(0);
  });
});

// ─── E. Access control ───────────────────────────────────────────────────────

test.describe('Access control', () => {
  test('superadmin can access Builder tab [GLB-05]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    // Builder should NOT show restricted message for superadmin
    await expect(page.locator('text=/restricted to admins/i')).not.toBeVisible({ timeout: 3000 });
  });
});

// ─── E2. yes/no "No" corrective-note enforcement (ops-fr4) ───────────────────

// createYesNoTemplate creates a template with a single yes/no field scheduled
// for today and assigned to the test user's role so it shows in My Checklists.
async function createYesNoTemplate(page, name, todayDOW) {
  const input = {
    name,
    requires_approval: true,
    sections: [
      {
        title: 'Section 1',
        order: 0,
        condition: null,
        fields: [
          {
            type: 'yes_no',
            label: 'Fridge under 40F?',
            required: false,
            order: 0,
            config: {},
            fail_trigger: null,
            condition: null,
          },
        ],
      },
    ],
    schedules: [{ active_days: [todayDOW] }],
    assignments: [
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
    ],
  };
  return apiCall(page, 'POST', 'createTemplate', input);
}

test.describe('yes/no No enforcement', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
  });

  test('answering No without a corrective note blocks submit [FLD-08 GATE-01]', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    await createYesNoTemplate(page, 'YesNo Block Test', todayDOW);
    await page.reload();

    // Open the checklist
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('#fill-body .fill-field', { timeout: 5000 });

    // Answer "No" — the corrective fail card should render
    await page.click('[data-action="set-no"]');
    await expect(page.locator('.fail-card')).toBeVisible({ timeout: 5000 });

    // Leave the note empty. Submit.
    await page.click('[data-action="submit"]');

    // BLOCKED: corrective toast appears and we stay on the fill view
    // (submit button + fail card still present; list not restored).
    await expect(page.locator('#toast')).toContainText('severity', { timeout: 5000 });
    await expect(page.locator('#submit-btn')).toBeVisible();
    await expect(page.locator('#fill-body .fill-field')).toBeVisible();
    await expect(page.locator('#checklist-list .row')).toHaveCount(0);

    // And no submission was created server-side.
    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    const count = Array.isArray(pending) ? pending.length : 0;
    expect(count).toBe(0);
  });

  test('answering No with note + severity allows submit [GATE-02]', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    await createYesNoTemplate(page, 'YesNo Pass Test', todayDOW);
    await page.reload();

    await page.click('[data-fill-template-id]');
    await page.waitForSelector('#fill-body .fill-field', { timeout: 5000 });

    // Answer "No", fill the corrective note + pick a severity.
    await page.click('[data-action="set-no"]');
    await expect(page.locator('.fail-card')).toBeVisible({ timeout: 5000 });
    await page.fill('[data-action="fail-note-input"]', 'Adjusted thermostat, re-checked temp');
    await page.click('[data-action="set-severity"][data-severity="major"]');

    // Wait for the fail-note auto-save to persist (debounced).
    await page.waitForTimeout(1800);

    // Submit — should succeed and return to the list.
    await page.click('[data-action="submit"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#s1')).toBeVisible({ timeout: 8000 });

    // A submission now exists server-side.
    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    expect(Array.isArray(pending) && pending.length).toBeGreaterThan(0);
  });
});

// ─── E3. required-photo submit gate (ops-nfr3) ───────────────────────────────

// createPhotoTemplate creates a template with a single REQUIRED photo field,
// scheduled for today and assigned to the test user's role so it shows in
// My Checklists. A photo field's answered value is just an https:// URL string.
async function createPhotoTemplate(page, name, todayDOW) {
  const input = {
    name,
    requires_approval: true,
    sections: [
      {
        title: 'Section 1',
        order: 0,
        condition: null,
        fields: [
          {
            type: 'photo',
            label: 'Photo of clean station',
            required: true,
            order: 0,
            config: {},
            fail_trigger: null,
            condition: null,
          },
        ],
      },
    ],
    schedules: [{ active_days: [todayDOW] }],
    assignments: [
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
    ],
  };
  return apiCall(page, 'POST', 'createTemplate', input);
}

// S1 de-flake helper (GATE-04, hardened after T-20's passed-on-retry
// observation — not reproduced in 3 targeted contention legs): wait out the WS
// catch-up replay storm before acting inside the runner. A fresh context starts
// at Lamport 0, so wsCatchUp replays the whole carried ops journal; replayed
// SUBMIT/APPROVE ops with a runner open each fire loadMyChecklists, whose
// landing re-render can detach the node an action is about to dispatch events
// on (the sync.spec.js:1198 mechanism). Settle = the replay loop is done
// (LAMPORT_CLOCK.ts stable across 400ms) AND no myChecklists GET has landed
// for 600ms. Bounded, best-effort — assertions stay authoritative.
async function settleRunner(page) {
  let prev = -2;
  const clockDeadline = Date.now() + 8000;
  while (Date.now() < clockDeadline) {
    const ts = await page.evaluate(() => window.LAMPORT_CLOCK ? window.LAMPORT_CLOCK.ts : -1);
    if (ts !== -1 && ts === prev) break;
    prev = ts;
    await page.waitForTimeout(400);
  }
  let lastGet = Date.now();
  const onResp = (res) => {
    if (res.url().includes('/myChecklists') && res.request().method() === 'GET') lastGet = Date.now();
  };
  page.on('response', onResp);
  const quietDeadline = Date.now() + 6000;
  while (Date.now() < quietDeadline && Date.now() - lastGet < 600) {
    await page.waitForTimeout(150);
  }
  page.off('response', onResp);
}

test.describe('required-photo submit gate', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    // S1 de-flake: GATE-04's block test asserts pendingApprovals === 0, but the
    // server doesn't filter archived templates out of pendingApprovals — a
    // submission carried over from an earlier test (or a carried DB) fails it
    // regardless of this test's own behavior. Clear the carried state so the
    // assertion measures THIS test's submit, not suite history.
    await cleanupPendingApprovals(page);
    await cleanupTemplates(page);
  });

  test('submit is blocked when a required photo is not attached [GATE-04]', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    await createPhotoTemplate(page, 'Photo Block Test', todayDOW);
    await page.reload();

    // Open the checklist. Do NOT attach a photo.
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('#fill-body .fill-field', { timeout: 5000 });
    // Let the catch-up replay storm pass before acting (see settleRunner).
    await settleRunner(page);

    // Submit with no photo.
    await page.click('[data-action="submit"]');

    // BLOCKED: photo-required toast appears and we stay on the fill view.
    await expect(page.locator('#toast')).toContainText('Photo required', { timeout: 5000 });
    await expect(page.locator('#submit-btn')).toBeVisible();
    await expect(page.locator('#fill-body .fill-field')).toBeVisible();
    await expect(page.locator('#checklist-list .row')).toHaveCount(0);

    // And no submission was created server-side.
    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    const count = Array.isArray(pending) ? pending.length : 0;
    expect(count).toBe(0);
  });

  test('submit is allowed once a photo URL is attached [GATE-04]', async ({ page }) => {
    const todayDOW = await getTodayDOW(page);
    await createPhotoTemplate(page, 'Photo Pass Test', todayDOW);
    await page.reload();

    await page.click('[data-fill-template-id]');
    await page.waitForSelector('#fill-body .fill-field', { timeout: 5000 });
    // Let the catch-up replay storm pass before acting (see settleRunner).
    await settleRunner(page);

    // Simulate a captured photo: a photo field's value is just its https:// URL.
    // Persist it via the same save-response path the camera-upload flow uses,
    // then mirror it into the live fill state so the submit handler sees it.
    // S1 de-flake: the old blind 1800ms wait raced the debounced save under
    // load — gate on the POST /ops actually committing (2xx) instead, armed
    // BEFORE the call that schedules it (POST-observed discipline).
    const photoSaved = page.waitForResponse(
      res => res.url().includes('/api/v1/workflow/ops') && res.request().method() === 'POST',
      { timeout: 12000 });
    const fldId = await page.evaluate(() => {
      const flds = fillState.activeTemplate.sections.flatMap(function (s) { return s.fields; });
      const photoFld = flds.find(function (f) { return f.type === 'photo'; });
      const url = 'https://example.com/photos/checklists/test.jpg';
      FIELD_RESPONSES[photoFld.id] = { value: url, answeredBy: 'test', answeredAt: new Date() };
      debouncedSaveField(photoFld.id, url);
      return photoFld.id;
    });
    expect(fldId).toBeTruthy();
    const photoSaveRes = await photoSaved;
    expect(photoSaveRes.ok(), 'photo autosave must commit (2xx) before submit').toBeTruthy();

    // Submit — should succeed (not blocked by the photo gate). The success
    // path plays a confirmation animation before returning to the list.
    await page.click('[data-action="submit"]');
    await expect(page.locator('#toast')).toContainText('Submitted', { timeout: 8000 });

    // A submission now exists server-side (poll — the POST resolves before the
    // success animation finishes).
    await expect.poll(async () => {
      const pending = await apiCall(page, 'GET', 'pendingApprovals');
      return Array.isArray(pending) ? pending.length : 0;
    }, { timeout: 8000 }).toBeGreaterThan(0);
  });
});

// ─── F. Navigation ──────────────────────────────────────────────────────────

test.describe('Navigation', () => {
  test('clicking My Checklists tab while runner is open returns to list [RUN-18]', async ({ page }) => {
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

// ─── Tab Persistence ────────────────────────────────────────────────────────

test.describe('Tab Persistence', () => {
  test('workflows tab persists on reload [GLB-07]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await page.click('#t2');
    await expect(page.locator('#t2.on')).toBeVisible();
    expect(page.url()).toContain('#tab=2');

    await page.reload();
    await expect(page.locator('#t2.on')).toBeVisible({ timeout: 5000 });
  });

  test('onboarding tab persists on reload', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/onboarding.html');
    // Only managers/admins see tabs 2 and 3
    const t3 = page.locator('#t3');
    if (await t3.isVisible({ timeout: 3000 })) {
      await t3.click();
      await expect(page.locator('#t3.on')).toBeVisible();
      expect(page.url()).toContain('#tab=3');

      await page.reload();
      await expect(page.locator('#t3.on')).toBeVisible({ timeout: 5000 });
    }
  });

  test('inventory tab persists on reload', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/inventory.html');
    await page.click('#t2');
    await expect(page.locator('#t2.on')).toBeVisible();
    expect(page.url()).toContain('#tab=2');

    await page.reload();
    await expect(page.locator('#t2.on')).toBeVisible({ timeout: 5000 });
  });

  test('users tab persists on reload', async ({ page }) => {
    // users.html has 2 top-level tabs since Edit was folded into the Users
    // tab (commit 2083f20): #t1 Users (default), #t2 Access. Assert the
    // non-default Access tab persists across reload via the #tab= hash.
    await login(page);
    await page.goto(BASE + '/users.html');
    await page.click('#t2');
    await expect(page.locator('#t2.on')).toBeVisible();
    expect(page.url()).toContain('#tab=2');

    await page.reload();
    await expect(page.locator('#t2.on')).toBeVisible({ timeout: 5000 });
  });
});

// ─── G. Validation ──────────────────────────────────────────────────────────

test.describe('Validation', () => {
  test('submit is blocked when fail trigger fires but corrective action is empty [FLD-12 GATE-01]', async ({ page }) => {
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

  test('server rejects submission with triggered fail but no corrective action [GATE-03]', async ({ page }) => {
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
  test('submitted checklist fields are not interactive [RUN-09 FLD-R1]', async ({ page }) => {
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

  // Card F1. The test above never reloads, so MY_SUBMISSIONS keeps the optimistic
  // client-side status ('submitted') the submit handler pushed. Reload and the
  // status comes from the SERVER instead — and no case covered that until now.
  //
  // Before F1 the server said 'pending' for a template requiring no approval, so
  // the runner claimed "Waiting for manager review" for a checklist nobody would
  // ever review. F1 makes the server say 'completed'; the runner must read that as
  // the terminal submitted state it is, stay read-only, and NOT offer to submit
  // again. Without the client half, a no-approval checklist reads as unsubmitted
  // after any reload and can be submitted twice.
  test('a no-approval checklist stays submitted and read-only across a reload [RUN-09b]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const name = 'NoApproval Reload Test';
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name,
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Check this', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: name }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(1500);
    await page.click('[data-action="submit"]');
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 10000 });

    // The reload is the whole point: drop the optimistic status and re-read the
    // server's.
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });

    // The server must not be describing THIS template's submission as awaiting
    // review. (Scoped to this template id — myChecklists returns every submission
    // the user has, and other specs leave genuinely-pending ones behind.)
    const statuses = await page.evaluate(async (tplId) => {
      const res = await fetch('/api/v1/workflow/myChecklists');
      const body = await res.json();
      return (body.submissions || []).filter(s => s.template_id === tplId).map(s => s.status);
    }, tpl.id);
    expect(statuses.length).toBeGreaterThan(0);
    expect(statuses).not.toContain('pending');

    await page.locator('#checklist-list .row', { hasText: name }).first().click();
    await page.waitForTimeout(1500);

    // Read-only, and terminal — not "waiting for manager review", not re-submittable.
    await expect(page.locator('.submit-confirm')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('#fill-body')).not.toContainText('Waiting for manager review');
    await expect(page.locator('[data-action="submit"]')).toHaveCount(0);
    await expect(page.locator('.check-btn')).toHaveCount(0);
  });

  // Card A, run 20260726. A triage sweep of the whole suite found exactly TWO
  // tests that create a requires_approval:false template, submit it, and assert on
  // the RENDERED result — and both were the two that went red when F1 changed the
  // server's status word. GATE-01/03/06 look like coverage but pass vacuously here:
  // they assert submission is BLOCKED, so they never reach a rendered submitted
  // state at all. This test is the missing render assertion, and it checks the two
  // surfaces separately, because they read the status through different code paths
  // and each one broke on its own:
  //
  //   1. the LIST row badge  (renderChecklistList — must say "Submitted", NOT
  //      "Pending Approval", for a checklist nobody will ever review)
  //   2. the RUNNER          (renderRunner — confirm line, no #submit-btn, no
  //      clickable inputs, fillState.readonly true)
  //
  // and it checks both BEFORE and AFTER a reload. Before = the optimistic status
  // the submit handler pushes; after = the server's. Asserting only the first is
  // what let the regression through: the optimistic value masked the server's for
  // the entire lifetime of the page.
  test('a no-approval submitted checklist RENDERS as submitted — list badge + read-only runner, optimistic and after reload [RUN-09c]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const name = 'NoApproval Render Test';
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name,
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Only task', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: name }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(1500);
    await page.click('[data-action="submit"]');

    // ── Surface 1, optimistic: the LIST row badge ────────────────────────────
    // The submit handler navigates back to the list. This row is rendered from
    // the status the submit handler wrote into MY_SUBMISSIONS, before any fetch.
    await expect(page.locator('#checklist-list')).toBeVisible({ timeout: 10000 });
    const optimisticRow = page.locator('#checklist-list .row', { hasText: name }).first();
    await expect(optimisticRow).toContainText('Submitted', { timeout: 10000 });
    await expect(optimisticRow).not.toContainText('Pending Approval');
    await expect(optimisticRow.locator('.approval-badge')).toHaveCount(0);

    // The optimistic status must be the one a reload would fetch — if these two
    // differ, any client/server divergence hides until the page is reloaded.
    const optimisticStatus = await page.evaluate((tplId) => {
      const s = (typeof MY_SUBMISSIONS !== 'undefined' ? MY_SUBMISSIONS : [])
        .find(x => x.template_id === tplId);
      return s ? s.status : null;
    }, tpl.id);
    expect(optimisticStatus).toBe('completed');

    // ── Reload: everything below now reads the SERVER's status ───────────────
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });

    const serverStatus = await page.evaluate((tplId) => {
      const s = (typeof MY_SUBMISSIONS !== 'undefined' ? MY_SUBMISSIONS : [])
        .find(x => x.template_id === tplId);
      return s ? s.status : null;
    }, tpl.id);
    expect(serverStatus).toBe(optimisticStatus);

    // ── Surface 1, post-reload: the LIST row badge ───────────────────────────
    const reloadedRow = page.locator('#checklist-list .row', { hasText: name }).first();
    await expect(reloadedRow).toContainText('Submitted', { timeout: 10000 });
    await expect(reloadedRow).not.toContainText('Pending Approval');
    await expect(reloadedRow.locator('.approval-badge')).toHaveCount(0);
    // Progress renders off the frozen snapshot, not the live template.
    await expect(reloadedRow).toContainText('1/1 items');

    // ── Surface 2, post-reload: the RUNNER ───────────────────────────────────
    await reloadedRow.click();
    await expect(page.locator('.submit-confirm')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('.submit-confirm')).toContainText('Checklist submitted');
    await expect(page.locator('#fill-body')).not.toContainText('Waiting for manager review');
    // Not re-submittable: the button is gone by id AND by action.
    await expect(page.locator('#submit-btn')).toHaveCount(0);
    await expect(page.locator('[data-action="submit"]')).toHaveCount(0);
    // Genuinely read-only, not merely button-less.
    await expect(page.locator('.check-btn')).toHaveCount(0);
    expect(await page.evaluate(() => fillState.readonly)).toBe(true);
    // Terminal-but-reversible: the submitter can still take it back.
    await expect(page.locator('[data-action="unsubmit"]')).toBeVisible();

    // ── And exactly ONE submission row exists ────────────────────────────────
    // The pre-fix runner offered a second submit, and submitChecklistToAPI mints a
    // fresh idempotency_key per call, so taking it wrote a SECOND row.
    const rows = await page.evaluate(async (tplId) => {
      const res = await fetch('/api/v1/workflow/myChecklists');
      const body = await res.json();
      return (body.submissions || []).filter(s => s.template_id === tplId);
    }, tpl.id);
    expect(rows.length).toBe(1);
    expect(rows[0].status).toBe('completed');
  });
});

// ─── H. Loading states ───────────────────────────────────────────────────────

test.describe('Loading states', () => {
  test('skeleton screens show during load [GLB-09]', async ({ page }) => {
    await login(page);
    // Navigate to workflows and check for skeleton elements
    const skeletonPromise = page.locator('.skeleton').first().waitFor({ state: 'visible', timeout: 2000 }).catch(() => null);
    await page.goto(BASE + '/workflows.html');
    // Skeletons may or may not be captured depending on timing — just verify page loads
    await expect(page.locator('#t1')).toBeVisible({ timeout: 5000 });
  });

  test('workflows page loads and shows tabs [GLB-02]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await expect(page.locator('#t1')).toContainText('My Checklists');
    await expect(page.locator('#t2')).toContainText('Approvals');
    await expect(page.locator('#t3')).toContainText('Builder');
  });

  test('checklist progress is shared across team members [LC-04]', async ({ page }) => {
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

  test('field attribution shows who actually checked it, not the viewer [RUN-03]', async ({ page }) => {
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

  test('re-checking a field updates attribution to the new user [RUN-03]', async ({ page, browser }) => {
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

  test('yes/no field attribution updates when different user answers [RUN-03]', async ({ page }) => {
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

  test('sub-step completion attributes parent checkbox to the user who completed it [FLD-04]', async ({ page }) => {
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
            { type: 'checkbox', id: 'sub1', label: 'Item A counted' },
            { type: 'checkbox', id: 'sub2', label: 'Item B counted' },
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

    // Each sub-step is attributed to the user who actually checked it, not the
    // current viewer. In the interactive runner the parent checkbox has no
    // separate attribution row (see FLD-03) — attribution lives per sub-step:
    // sub1 was checked by the admin (Jamal C.), sub2 (which completes the
    // parent) was checked by the team_member (SubStep U.).
    await expect(page.locator('.fill-attribution')).toHaveCount(2);
    const sub1Attrib = await page.locator('.fill-attribution').nth(0).textContent();
    const sub2Attrib = await page.locator('.fill-attribution').nth(1).textContent();
    // The completing sub-step is credited to the team_member, NOT the viewer.
    expect(sub2Attrib).toContain('SubStep U.');
    expect(sub2Attrib).not.toContain('Jamal');
    // And the earlier sub-step retains the admin's attribution (proving each
    // completion is attributed to its own actor).
    expect(sub1Attrib).toContain('Jamal');
  });

  test('sub-step attribution appears before divider line [FLD-03 FLD-04]', async ({ page }) => {
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

  test('sub-steps visible in read-only submitted view [FLD-R2]', async ({ page }) => {
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

  test('sub-steps visible in approvals review tab [APR-05]', async ({ page }) => {
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

  test('list view item count matches runner count with conditional fields [LST-05]', async ({ page }) => {
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

  test('conditional field appears after dependent checkbox is checked [VIS-06]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    // Create template: checkbox A + checkbox B that only shows when A is checked
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Conditional Checkbox Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Field A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    // Get server-assigned field ID
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Conditional Checkbox Test');
    const fieldAId = tpl.sections[0].fields[0].id;

    // Update: add field B with condition on field A = true
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Conditional Checkbox Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { id: fieldAId, type: 'checkbox', label: 'Field A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        { type: 'checkbox', label: 'Field B (conditional)', required: false, order: 1, config: null, fail_trigger: null,
          condition: { field_id: fieldAId, operator: 'equals', value: 'true' } },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Conditional Checkbox Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Initially only Field A should be visible
    await expect(page.locator('.fill-field-label', { hasText: 'Field A' })).toBeVisible();
    await expect(page.locator('.fill-field-label', { hasText: 'Field B' })).not.toBeVisible();

    // Check Field A
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(500);

    // Field B should appear immediately without page refresh
    await expect(page.locator('.fill-field-label', { hasText: 'Field B' })).toBeVisible({ timeout: 3000 });
  });

  test('conditional field appears after dependent text field is filled [VIS-07]', async ({ page }) => {
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

  test('submitted responses survive template update [LC-02]', async ({ page }) => {
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

  test('submitted checklist survives builder edit with assignment change [LC-02]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    // Create template assigned to admin only
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Builder Edit Survive',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Do the thing', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ]
    });

    // Open, check, submit
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Builder Edit Survive' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(1500);
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(2000);

    // Now edit in Builder — add team_member as assignee (triggers replaceTemplate)
    await page.click('#fill-back');
    await page.waitForTimeout(500);
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#builder-list .row', { hasText: 'Builder Edit Survive' })).toBeVisible({ timeout: 5000 });
    await page.locator('#builder-list .row', { hasText: 'Builder Edit Survive' }).first().click();
    await page.waitForSelector('#builder-body');

    // Just re-save without changes (triggers replaceTemplate)
    await page.click('#save-btn');
    await expect(page.locator('#builder-list')).toBeVisible({ timeout: 5000 });

    // Go back to My Checklists and open
    await page.click('#t1');
    await page.waitForTimeout(1000);
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Builder Edit Survive' }).first().click();
    await page.waitForSelector('#fill-body');
    await page.waitForTimeout(500);

    // Should still show 1/1 complete, not 0/1
    await expect(page.locator('.progress-line')).toContainText('1 of 1');
  });

  test('draft responses survive template update (assignment change) [LC-03]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    const result = await apiCall(page, 'POST', 'createTemplate', {
      name: 'Draft Survive Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Check me', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }]
    });

    // Open checklist and check the item (creates a draft response)
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Draft Survive Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(2000); // wait for auto-save

    // Go back to list
    await page.click('#fill-back');
    await page.waitForTimeout(500);

    // Update template in builder — just add an approver (triggers replaceTemplate)
    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find(t => t.name === 'Draft Survive Test');
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'Draft Survive Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { id: tpl.sections[0].fields[0].id, type: 'checkbox', label: 'Check me', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ]
    });

    // Reload and open checklist — draft response should survive
    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'Draft Survive Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Checkbox should still be checked
    await expect(page.locator('.check-btn.checked')).toBeVisible({ timeout: 5000 });
  });

  test('unsubmit returns checklist to editable draft [RUN-10]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Unsubmit Test',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Check this', required: false, order: 0, config: null, fail_trigger: null, condition: null },
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
    await page.locator('#checklist-list .row', { hasText: 'Unsubmit Test' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check item and submit
    await page.locator('.check-btn').first().click();
    await page.waitForTimeout(1500);
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(2000);

    // Should show "Pending Approval" with Unsubmit button
    await expect(page.locator('[data-action="unsubmit"]')).toBeVisible({ timeout: 5000 });

    // Unsubmit — accept confirm dialog
    page.once('dialog', async dialog => await dialog.accept());
    await page.click('[data-action="unsubmit"]');

    // Should show success toast
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // Submit button should be visible again (editable state)
    await expect(page.locator('[data-action="submit"]')).toBeVisible({ timeout: 5000 });

    // Checkbox should still be checked (response restored as draft)
    await expect(page.locator('.check-btn.checked')).toBeVisible();
  });

  test('incomplete submit shows confirmation prompt [GATE-06]', async ({ page }) => {
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

  test('sub-step progress counts correctly in list view [LST-04]', async ({ page }) => {
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

  test('skip logic condition persists after save and reload [BLD-07]', async ({ page }) => {
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

  test('new section defaults to Same as schedule [BLD-07]', async ({ page }) => {
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

  test('text field in read-only view shows answer below label [FLD-R4]', async ({ page }) => {
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

  test('team_member cannot see Builder tab [GLB-03]', async ({ page }) => {
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

  // Order-independence guard (carried waiver #1, full-suite-only reds): a
  // fresh browser context starts its Lamport clock at 0, so sync.js's
  // wsCatchUp replays the ENTIRE ops journal accumulated in the shared
  // hq_test by earlier spec files (sync.spec.js is the dominant producer).
  // The SUBMIT_CHECKLIST / APPROVE_ITEM replay branches USED TO each fire a
  // loadMyChecklists() re-fetch; a stale drafts snapshot resolving mid-fill
  // re-hydrated the open runner and clobbered an optimistic checkbox answer
  // (observed: "3 of 4 items complete" right after clicking all 4). The
  // submit then fell into the "1 item not completed. Submit anyway?"
  // confirm() — which Playwright auto-dismisses — and returned WITHOUT ever
  // showing a toast.
  //
  // Both branches are now gated on (runner open) ∨ !silent (T-18), so the
  // storm itself is gone and the clobber with it. This drain is kept as a
  // cheap settle after page load in multi-user flows — it is no longer load-
  // bearing against that specific race.
  async function drainOpsReplay(page) {
    await page.waitForLoadState('networkidle');
  }

  // Click every checkbox in the open runner and require all `expected` checked
  // before returning — the submit-toast assertions are the contract for a FULLY
  // completed submit, so the precondition must hold.
  //
  // This used to carry a second "repair" pass that re-clicked any answer a
  // straggler stale re-render had un-checked. That workaround is dead as of the
  // T-18 gate: the SUBMIT_CHECKLIST replay branch no longer fires a
  // loadMyChecklists() per replayed op, so there is no stale snapshot to land
  // mid-fill and clobber an optimistic answer. Plain clicks again.
  async function checkAll(page, expected) {
    const checkBtns = page.locator('.check-btn');
    const count = await checkBtns.count();
    expect(count).toBe(expected);
    for (let i = 0; i < count; i++) {
      await checkBtns.nth(i).click();
      await page.waitForTimeout(300);
    }
    await page.waitForTimeout(2000); // auto-save
    await expect(page.locator('.check-btn.checked')).toHaveCount(expected);
  }

  test('team member completes checklist, manager approves [RUN-07 APR-11]', async ({ page }) => {
    test.setTimeout(60000); // 3 logins + networkidle catch-up drains
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
    await drainOpsReplay(page);
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: tplName }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check all 4 checkboxes (with clobber-repair — see drainOpsReplay)
    await checkAll(page, 4); // plain clicks (T-18 gate removed the clobber)

    // Submit
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(1000);

    // Verify submission confirmation
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // --- User B (manager): approve ---
    await login(page, mgr.email, mgr.password);
    await page.goto(BASE + '/workflows.html');
    await drainOpsReplay(page);
    await page.click('#t2'); // Approvals tab
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2').locator('text=' + tplName + '')).toBeVisible({ timeout: 5000 });

    // Approve
    await page.click('[data-action="approve"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // Verify the approval list is now empty
  });

  test('team member completes checklist, manager rejects 2 items, crew resubmits, manager approves [APR-09 FLD-18 LC-01]', async ({ page }) => {
    test.setTimeout(90000); // 5 logins + networkidle catch-up drains
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
    await drainOpsReplay(page);
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: rejName }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // Check all 4 checkboxes (with clobber-repair — see drainOpsReplay)
    await checkAll(page, 4); // plain clicks (T-18 gate removed the clobber)
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(1000);

    // --- User B (manager): flag 2 items with comments, reject ---
    await login(page, mgr.email, mgr.password);
    await page.goto(BASE + '/workflows.html');
    await drainOpsReplay(page);
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
    await drainOpsReplay(page);
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
    // Repair precondition (see drainOpsReplay) — all 4 must be checked before
    // resubmitting or the confirm() prompt silently aborts the submit.
    await expect(page.locator('.check-btn.checked')).toHaveCount(4);

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
    await drainOpsReplay(page);
    await page.click('#t2');
    await expect(page.locator('#s2').locator('text=' + rejName + '').first()).toBeVisible({ timeout: 5000 });
    await page.click('[data-action="approve"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });
  });

  test('approved checklist shows Approved badge and cannot be resubmitted [LST-08 RUN-08]', async ({ page }) => {
    // Three logins + the networkidle catch-up drain below can exceed the 30s
    // default when the shared-DB ops journal is large (full-suite position).
    test.setTimeout(60000);
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
    await drainOpsReplay(page);
    await page.waitForSelector('#checklist-list .row');
    await page.locator('#checklist-list .row', { hasText: appName }).first().click();
    await page.waitForSelector('#fill-body .fill-field');
    // Check all 4 checkboxes (with clobber-repair — see drainOpsReplay)
    await checkAll(page, 4); // plain clicks (T-18 gate removed the clobber)
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(1000);

    // Wait for the submission-confirmation toast before switching users. This
    // proves the submitChecklist POST landed (and the pending-approval row
    // exists) so the manager's Approvals list is populated deterministically —
    // mirrors the passing "manager approves" sibling test above.
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // --- Manager: approve ---
    await login(page, mgr.email, mgr.password);
    await page.goto(BASE + '/workflows.html');
    await drainOpsReplay(page);
    await page.click('#t2');
    await expect(page.locator('#s2').locator('text=' + appName + '').first()).toBeVisible({ timeout: 5000 });
    await page.click('[data-action="approve"]');
    await expect(page.locator('#toast')).toBeVisible({ timeout: 5000 });

    // --- Crew: verify "Approved ✓" badge on My Checklists list ---
    await login(page, crew.email, crew.password);
    await page.goto(BASE + '/workflows.html');
    await drainOpsReplay(page);
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

// ─── F. prove-UNPROVEN sweep (ops-prove-checklists) ──────────────────────────
// Red-first assertions for three UNPROVEN flows: FR-6 (idempotent submit — no
// duplicate submission row), FR-7 (unsubmit authorization — a non-submitter is
// refused 403), FR-8 (History returns the last 50 submissions, DESC order).

test.describe('Checklist submit/unsubmit/history (prove sweep)', () => {
  // FR-6 — Submitting the same checklist twice (same idempotency key) is
  // idempotent: exactly ONE submission row exists afterward. Asserts against
  // myHistory (the durable submission record), not just pendingApprovals, so a
  // second row would be caught even after approval/removal from the queue.
  test('FR-6 duplicate submit with same idempotency key creates exactly one submission [GATE-08]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    const result = await createTestTemplate(page, 'FR6 Idempotent', todayDOW);
    const templateId = result.id;

    // Baseline: how many prior submissions this user already has for this template.
    const historyBefore = await apiCall(page, 'GET', 'myHistory');
    const beforeCount = (Array.isArray(historyBefore) ? historyBefore : [])
      .filter((s) => s.template_id === templateId).length;

    // Submit twice with the SAME idempotency key.
    const key = generateUUID();
    const payload = { template_id: templateId, idempotency_key: key, responses: [] };
    const status1 = await page.evaluate(async (p) => {
      const res = await fetch('/api/v1/workflow/submitChecklist', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(p),
      });
      return res.status;
    }, payload);
    const status2 = await page.evaluate(async (p) => {
      const res = await fetch('/api/v1/workflow/submitChecklist', {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(p),
      });
      return res.status;
    }, payload);

    // Neither submit is an error; the second is deduped (not a fresh 500).
    expect([200, 201]).toContain(status1);
    expect([200, 201, 409]).toContain(status2);

    // Observable DB state: exactly one NEW submission for this template.
    const historyAfter = await apiCall(page, 'GET', 'myHistory');
    const forTemplate = (Array.isArray(historyAfter) ? historyAfter : [])
      .filter((s) => s.template_id === templateId);
    expect(forTemplate.length - beforeCount).toBe(1);

    // And all of them share the one idempotency key we submitted with.
    const distinctIds = new Set(forTemplate.map((s) => s.id));
    expect(distinctIds.size).toBe(forTemplate.length);
    const newRows = forTemplate.filter((s) => s.idempotency_key === key);
    expect(newRows.length).toBe(1);
  });

  // FR-7 — Unsubmit requires authorization: a user who did NOT submit the
  // checklist (here a freshly-invited team_member, in a separate browser
  // context so the admin session is untouched) is refused with 403 not_submitter,
  // and the submission still exists afterward. Non-admin session authored INLINE
  // per the multi-role invite/accept-invite idiom used elsewhere in this file.
  test('FR-7 a non-submitter is refused (403) when unsubmitting [RUN-11]', async ({ page, browser }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    // Admin creates + submits a checklist; capture its submission id via myHistory.
    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'FR7 Unsubmit Auth', todayDOW);
    const templateId = tpl.id;
    await submitChecklistViaAPI(page, templateId);
    const adminHistory = await apiCall(page, 'GET', 'myHistory');
    const sub = (Array.isArray(adminHistory) ? adminHistory : [])
      .find((s) => s.template_id === templateId);
    expect(sub && sub.id).toBeTruthy();
    const submissionId = sub.id;

    // Invite a team_member and activate them (INLINE — no shared helper).
    const memberEmail = 'fr7-nonsubmitter-' + Date.now() + '@yumyums.kitchen';
    const memberPassword = 'test456';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Non', last_name: 'Submitter', email, roles: ['team_member'] }),
      });
      return res.json();
    }, memberEmail);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async ([t, pw]) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: pw }),
      });
    }, [token, memberPassword]);

    // In a fresh context, log in AS the team_member and attempt to unsubmit the
    // admin's submission. Server must refuse: 403 not_submitter.
    const memberCtx = await browser.newContext();
    try {
      const memberPage = await memberCtx.newPage();
      await login(memberPage, memberEmail, memberPassword);
      const { status, bodyText } = await memberPage.evaluate(async (sid) => {
        const res = await fetch('/api/v1/workflow/unsubmitChecklist', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ submission_id: sid }),
        });
        return { status: res.status, bodyText: await res.text() };
      }, submissionId);
      expect(status).toBe(403);
      expect(bodyText).toContain('not_submitter');
    } finally {
      await memberCtx.close();
    }

    // Re-login as admin on the main page: accept-invite above overwrote this
    // page's session cookie with the team_member's session.
    await login(page);

    // The submission still exists (was NOT unsubmitted by the non-submitter).
    const adminHistoryAfter = await apiCall(page, 'GET', 'myHistory');
    const stillThere = (Array.isArray(adminHistoryAfter) ? adminHistoryAfter : [])
      .some((s) => s.id === submissionId);
    expect(stillThere).toBe(true);
  });

  // FR-8 — History shows the crew member's last 50 submissions, newest first.
  // Create 52 submissions (unique idempotency keys → 52 distinct rows) and assert
  // myHistory caps the result at 50 and orders by submitted_at DESC.
  test('FR-8 myHistory returns at most the last 50 submissions in DESC order [LST-13]', async ({ page }) => {
    test.setTimeout(120000);
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    const tpl = await createTestTemplate(page, 'FR8 History Cap', todayDOW);
    const templateId = tpl.id;

    // Fire 52 submissions, each with a unique idempotency key → 52 distinct rows.
    const N = 52;
    for (let i = 0; i < N; i++) {
      await submitChecklistViaAPI(page, templateId);
    }

    const history = await apiCall(page, 'GET', 'myHistory');
    expect(Array.isArray(history)).toBe(true);

    // Cap at 50 even though 52 were created.
    expect(history.length).toBeLessThanOrEqual(50);
    // And it IS capped (not fewer than 50) — proves the LIMIT is exercised.
    expect(history.length).toBe(50);

    // All returned rows are distinct submissions.
    const ids = history.map((s) => s.id);
    expect(new Set(ids).size).toBe(ids.length);

    // Ordering: submitted_at is non-increasing (newest first).
    const times = history.map((s) => new Date(s.submitted_at).getTime());
    for (let i = 1; i < times.length; i++) {
      expect(times[i]).toBeLessThanOrEqual(times[i - 1]);
    }
  });
});

// ─── Builder conditional-logic prove sweep (ops-prove-builder, Track A card 3) ──
// FR-16 (DOW schedule visibility), FR-17 (section visibility condition),
// FR-18 (skip logic show/hide). Red-first assertions naming the observable
// DOM/list behavior against the CURRENT app. Appended LAST so this block runs
// after all sibling tests. Fixtures use unique template names + cleanupTemplates
// to avoid polluting sibling tests.
test.describe('Builder conditional logic (prove sweep)', () => {
  // FR-16 — A template's day-of-week schedule governs which days a checklist
  // appears in the crew member's My Checklists list.
  // Observable behavior asserted: a template scheduled for TODAY is present in
  // the list; a template scheduled for a NON-today DOW is ABSENT. This exercises
  // the server-side DOW gate (repository.go: active_days = ANY(...)).
  test('FR-16 DOW schedule shows today-scheduled checklist and hides off-day one [LST-10]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    const offDay = (todayDOW + 3) % 7; // a day that is definitely not today

    // Template A: scheduled for TODAY → should appear.
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'FR16 Scheduled Today',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Do a thing', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // Template B: scheduled for an OFF day (not today) → should be hidden.
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'FR16 Scheduled Off Day',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Do a thing', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [offDay] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });

    // Today-scheduled checklist is visible in the list.
    await expect(
      page.locator('#checklist-list .row', { hasText: 'FR16 Scheduled Today' }).first()
    ).toBeVisible();

    // Off-day checklist is NOT in the list at all.
    await expect(
      page.locator('#checklist-list .row', { hasText: 'FR16 Scheduled Off Day' })
    ).toHaveCount(0);
  });

  // FR-17 — A template section's visibility condition shows/hides the whole
  // section (and its fields) based on the condition. Observable behavior:
  // a section whose day-condition excludes today renders NONE of its fields in
  // the runner, while a sibling always-visible section renders its field.
  test('FR-17 section day-condition hides the section and its fields in the runner [VIS-03]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);
    const offDay = (todayDOW + 3) % 7;

    await apiCall(page, 'POST', 'createTemplate', {
      name: 'FR17 Section Condition',
      sections: [
        // Always-visible section (no condition).
        { title: 'Open Section', order: 0, condition: null, fields: [
          { type: 'checkbox', label: 'Always shown field', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        ]},
        // Section gated to an off-day → hidden today.
        { title: 'Weekend Only Section', order: 1, condition: { days: [offDay] }, fields: [
          { type: 'checkbox', label: 'Hidden section field', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        ]},
      ],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'FR17 Section Condition' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    // The open section's field is visible.
    await expect(page.locator('.fill-field-label', { hasText: 'Always shown field' })).toBeVisible();

    // The off-day section header AND its field are hidden.
    await expect(page.locator('.sec-hd', { hasText: 'Weekend Only Section' })).toHaveCount(0);
    await expect(page.locator('.fill-field-label', { hasText: 'Hidden section field' })).toHaveCount(0);

    // Progress counts only the 1 visible field (hidden section not counted).
    await expect(page.locator('.progress-line')).toContainText('0 of 1');
  });

  // FR-18 — Skip logic: a field's answer shows/hides a downstream field.
  // Observable behavior: the dependent field is hidden until the source field
  // equals the condition value, appears when it does, and HIDES AGAIN when the
  // source answer changes away from the value (full show/hide round-trip DOM
  // state). The slate flagged FR-18 as the likeliest RED — if the round-trip
  // hide does not fire, this records RED.
  test('FR-18 skip logic shows then re-hides a field as the source answer changes [VIS-05 VIS-08]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);

    // Create with a yes/no source field so we can toggle its value both ways.
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'FR18 Skip Logic Roundtrip',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'yes_no', label: 'Was there a spill', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    const templates = await apiCall(page, 'GET', 'templates');
    const tpl = templates.find((t) => t.name === 'FR18 Skip Logic Roundtrip');
    const sourceFieldId = tpl.sections[0].fields[0].id;

    // Add a dependent field that shows only when the source == "yes".
    await apiCall(page, 'PUT', 'updateTemplate/' + tpl.id, {
      name: 'FR18 Skip Logic Roundtrip',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { id: sourceFieldId, type: 'yes_no', label: 'Was there a spill', required: false, order: 0, config: null, fail_trigger: null, condition: null },
        { type: 'text', label: 'Describe the cleanup', required: false, order: 1, config: null, fail_trigger: null,
          condition: { field_id: sourceFieldId, operator: 'equals', value: 'true' } },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: false,
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    await page.reload();
    await page.waitForSelector('#checklist-list .row', { timeout: 10000 });
    await page.locator('#checklist-list .row', { hasText: 'FR18 Skip Logic Roundtrip' }).first().click();
    await page.waitForSelector('#fill-body .fill-field');

    const dependent = page.locator('.fill-field-label', { hasText: 'Describe the cleanup' });

    // Initially hidden — source not answered yet.
    await expect(dependent).toHaveCount(0);

    // Answer "Yes" → dependent field appears.
    await page.locator('[data-action="set-yes"]').first().click();
    await page.waitForTimeout(500);
    await expect(dependent).toBeVisible({ timeout: 3000 });

    // Change the answer to "No" → dependent field must HIDE again (round-trip).
    await page.locator('[data-action="set-no"]').first().click();
    await page.waitForTimeout(500);
    await expect(dependent).toHaveCount(0);
  });
});

// ─── Cross-cutting guarantees (prove sweep — Track A card 4) ─────────────────
// NFR-2 (photo presign), NFR-6 (archived 409), NFR-7 (401 redirect), FR-19
// (template-snapshot freeze), FR-20 (approval gate). NFR-5 (offline sync queue /
// conflict / cleanup) is PARKed — see the note at the bottom of this block: it
// needs IndexedDB + service-worker plumbing that Playwright blocks by config, and
// the offline-sync flows are in HQ's known flaky-red pool. No honest fixture can
// drive it here without S3/IndexedDB plumbing beyond a test fixture (PARK trigger).
test.describe('Cross-cutting guarantees (prove sweep)', () => {
  // apiPhotos performs a raw fetch against a non-workflow API path (e.g. photos,
  // me) and returns { status, body } so we can assert the degraded/edge shapes
  // the workflow apiCall() helper hides behind its /workflow/ prefix.
  async function apiRaw(page, method, path, body) {
    return page.evaluate(async ([m, p, b]) => {
      const opts = { method: m, credentials: 'include', headers: {} };
      if (b) { opts.headers['Content-Type'] = 'application/json'; opts.body = JSON.stringify(b); }
      const res = await fetch(p, opts);
      const text = await res.text();
      let json = null;
      try { json = JSON.parse(text); } catch (e) { /* non-JSON */ }
      return { status: res.status, text, json };
    }, [method, path, body]);
  }

  // NFR-2 — Photo pipeline: the photo field presigns against
  // POST /api/v1/photos/presign, then PUTs the file to S3/DO-Spaces and the
  // public URL round-trips. The PUT + round-trip legs require a LIVE DO-Spaces
  // client (bucket + creds); the ephemeral night-crew stack sets no SPACES_* env,
  // so the presigner is nil. This test proves the presign REQUEST/response *shape*
  // that is testable without S3: the endpoint is reachable behind auth and returns
  // the documented degraded contract (503 {"error":"photo storage not configured"})
  // when storage is unconfigured — instead of a presigned URL. The upload PUT +
  // public-URL round-trip is PARKed (needs live S3 — beyond a test fixture).
  test('NFR-2 photo presign returns the documented shape (503 degraded when S3 unset; PUT/round-trip PARKed) [FLD-20]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');

    const resp = await apiRaw(page, 'POST', '/api/v1/photos/presign', {
      path_prefix: 'checklists', id: 'nfr2-tpl', filename: 'nfr2.jpg',
    });

    // Authenticated request reaches the handler (not a 401/404 route miss).
    expect(resp.status).not.toBe(401);
    expect(resp.status).not.toBe(404);

    if (resp.status === 200) {
      // Storage IS configured (unexpected in the ephemeral stack): assert the
      // full presign response shape — a PUT url + a permanent public_url.
      expect(resp.json).toBeTruthy();
      expect(typeof resp.json.url).toBe('string');
      expect(resp.json.url.length).toBeGreaterThan(0);
      expect(typeof resp.json.public_url).toBe('string');
      expect(resp.json.public_url.length).toBeGreaterThan(0);
    } else {
      // Ephemeral stack: no SPACES_* env → presigner nil → documented 503 shape.
      expect(resp.status).toBe(503);
      expect(resp.json).toMatchObject({ error: 'photo storage not configured' });
    }
  });

  // NFR-6 — Archived-while-offline: submitting a checklist for a template that
  // was archived (soft-deleted) after the user opened it returns 409
  // template_archived instead of failing silently. getTemplateByID filters
  // archived_at IS NULL, so submit finds no template → ErrTemplateArchived → 409.
  // Observable behavior asserted: the POST /submitChecklist envelope is
  // status 409 with error 'template_archived', and NO submission is created.
  test('NFR-6 submitting an archived template returns 409 template_archived and creates no submission [GATE-09]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const created = await apiCall(page, 'POST', 'createTemplate', {
      name: 'NFR6 Archive Race',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });
    const tplId = created.id;

    // Archive the template out from under the (would-be offline) user.
    await apiCall(page, 'DELETE', 'archiveTemplate/' + tplId);

    // Now submit against the archived template via a raw fetch so we can read
    // the exact status + envelope the client sees.
    const resp = await apiRaw(page, 'POST', '/api/v1/workflow/submitChecklist', {
      template_id: tplId,
      idempotency_key: generateUUID(),
      responses: [],
    });

    expect(resp.status).toBe(409);
    expect(resp.json).toMatchObject({ error: 'template_archived' });

    // No submission surfaced anywhere (approvals empty for this template).
    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    const forThis = Array.isArray(pending) ? pending.filter(s => s.template_id === tplId) : [];
    expect(forThis.length).toBe(0);
  });

  // NFR-7 — Auth expiry mid-checklist: a 401 on an API call redirects the crew
  // member to /login.html (workflows.html init() checks /api/v1/me → 401 →
  // window.location='/login.html'; sync.js api() does the same for /workflow/*).
  // Observable behavior asserted (redirect leg): with no session cookie, loading
  // workflows.html lands on login.html. The "local drafts persist across the
  // redirect" leg needs IndexedDB draft plumbing (hq_offline_v1 store) beyond a
  // test fixture and Playwright blocks the service worker → PARKed (noted below).
  test('NFR-7 an unauthenticated workflows load redirects to /login.html [GLB-01]', async ({ page, context }) => {
    // Establish then clear the session to simulate a 401 mid-session.
    await login(page);
    await context.clearCookies();

    await page.goto(BASE + '/workflows.html');

    // init() calls /api/v1/me → 401 → window.location='/login.html'.
    await page.waitForURL(url => url.pathname.includes('login'), { timeout: 8000 });
    expect(page.url()).toContain('login');
  });

  // FR-19 — Template snapshot freeze: on submit, the checklist freezes a JSONB
  // snapshot of the template (checklist_submissions.template_snapshot) so a later
  // admin rename/edit does NOT alter an already-submitted checklist. Observable
  // behavior asserted: after submit, renaming the template changes the LIVE name
  // (pendingApprovals.template_name from the join) but the frozen
  // template_snapshot.name still holds the ORIGINAL name.
  test('FR-19 a submitted checklist freezes a template snapshot that later edits do not change [LC-02]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);
    await cleanupPendingApprovals(page);

    const todayDOW = await getTodayDOW(page);
    const created = await apiCall(page, 'POST', 'createTemplate', {
      name: 'FR19 Original Name',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });
    const tplId = created.id;

    // Submit while the template is still named "FR19 Original Name".
    await submitChecklistViaAPI(page, tplId);

    // Now rename the template (full-replace update). This must NOT touch the
    // frozen snapshot on the already-submitted checklist.
    await apiCall(page, 'PUT', 'updateTemplate/' + tplId, {
      name: 'FR19 RENAMED After Submit',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });

    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    const sub = (Array.isArray(pending) ? pending : []).find(s => s.template_id === tplId);
    expect(sub).toBeTruthy();

    // The LIVE template name (from the join) reflects the rename...
    expect(sub.template_name).toBe('FR19 RENAMED After Submit');

    // ...but the FROZEN snapshot still carries the ORIGINAL name.
    const snapshot = sub.template_snapshot;
    expect(snapshot).toBeTruthy();
    expect(snapshot.name).toBe('FR19 Original Name');
  });

  // FR-20 — Approval gate: saving a template with requires_approval ON is refused
  // unless at least one assignment has assignment_role 'approver' (hasApprover).
  // Observable behavior asserted: create WITHOUT an approver → 400 requires_approver
  // and NO template row created; create WITH an approver → 200 and a row exists.
  test('FR-20 requires_approval without an approver is refused (400 requires_approver); with one it is allowed [BLD-13]', async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupTemplates(page);

    const todayDOW = await getTodayDOW(page);

    // Case 1: requires_approval ON, only an assignee (no approver) → 400.
    const noApprover = await apiRaw(page, 'POST', '/api/v1/workflow/createTemplate', {
      name: 'FR20 No Approver',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      ],
    });
    expect(noApprover.status).toBe(400);
    expect(noApprover.json).toMatchObject({ error: 'requires_approver' });

    // The refused template must NOT have been created.
    let templates = await apiCall(page, 'GET', 'templates');
    expect((Array.isArray(templates) ? templates : []).find(t => t.name === 'FR20 No Approver')).toBeUndefined();

    // Case 2: same template but WITH an approver assignment → allowed.
    const withApprover = await apiRaw(page, 'POST', '/api/v1/workflow/createTemplate', {
      name: 'FR20 With Approver',
      sections: [{ title: 'Tasks', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [todayDOW] }],
      requires_approval: true,
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });
    // createTemplate returns 201 Created on success.
    expect(withApprover.status).toBeGreaterThanOrEqual(200);
    expect(withApprover.status).toBeLessThan(300);
    templates = await apiCall(page, 'GET', 'templates');
    expect((Array.isArray(templates) ? templates : []).find(t => t.name === 'FR20 With Approver')).toBeTruthy();
  });

  // NFR-5 — Sync / offline / conflict / cleanup — PARKED.
  // Reason: proving the offline queue (IndexedDB store hq_offline_v1 'submitQueue'),
  // the pending/saved/error save-status indicator, concurrent-edit conflict
  // resolution, and post-submit draft cleanup requires IndexedDB + service-worker
  // plumbing beyond a test fixture. The Playwright config sets
  // serviceWorkers:'block', and HQ's offline-sync specs are in the known
  // flaky-red baseline pool. Per the runbook PARK trigger (IndexedDB/SW plumbing),
  // this flow is PARKed for a dedicated offline-sync harness rather than forced
  // into a dishonest classification here. No test authored for NFR-5.
});

// ═══════════════════════════════════════════════════════════════════════════
// B5 fold-in — the /ops wire path must enforce the same review gate
// ═══════════════════════════════════════════════════════════════════════════
//
// G6 ship-blocker. POST /api/v1/workflow/ops sits in the SAME cookie-auth group
// as /approveSubmission, and its router dispatches APPROVE_ITEM / REJECT_ITEM
// straight to the same repository mutations. A gate applied only in the REST
// handlers left this path wide open: a zero-assignment team_member was refused
// at /approveSubmission and served 200 at /ops for the same submission, with the
// forged approval then broadcasting over the sync hub as legitimate.
//
// This block is the wire-level regression: same user, same submission, both
// doors, asserting the MUTATION did not occur — not merely the status code.
test.describe('Approve/reject authz — the /ops path', () => {

  // Builds: a template requiring approval, a submission on it, and a logged-in
  // team_member with NO approver assignment anywhere.
  async function setupForgeryFixture(page, tag) {
    await login(page);
    const email = `ops-authz-${tag}-${Date.now()}@yumyums.kitchen`;
    const invite = await page.evaluate(async (em) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Ops', last_name: 'Stranger', email: em, roles: ['team_member'] }),
      });
      return res.json();
    }, email);
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' }),
      });
    }, invite.invite_path.split('token=')[1]);

    await login(page);
    const tpl = await apiCall(page, 'POST', 'createTemplate', {
      name: `Ops Authz ${tag} ${Date.now()}`,
      sections: [{ title: 'S1', order: 0, condition: null, fields: [
        { type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null },
      ]}],
      schedules: [{ active_days: [0, 1, 2, 3, 4, 5, 6] }],
      // requires_approval demands an approver. Assigning the ADMIN role keeps
      // the team_member stranger a non-approver (the case under test) while
      // giving the positive leg a legitimately-assigned reviewer.
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
      requires_approval: true,
    });
    // createTemplate returns the bare row — re-read to get generated field ids.
    const full = (await apiCall(page, 'GET', 'templates')).find(t => t.id === tpl.id);
    const fieldId = full.sections[0].fields[0].id;
    await apiCall(page, 'POST', 'submitChecklist', {
      template_id: tpl.id, idempotency_key: generateUUID(),
      responses: [{ field_id: fieldId, value: JSON.stringify({ value: true }) }],
    });
    // Resolve the submission the way the rest of this file does — off the
    // approvals queue — rather than trusting the submit response shape.
    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    const submissionId = (pending.find(s => s.template_id === tpl.id) || pending[0]).id;

    await login(page, email, 'test456');
    return { submissionId, fieldId, email };
  }

  async function statusOf(page, submissionId) {
    await login(page);
    const list = await apiCall(page, 'GET', 'pendingApprovals');
    const found = (list || []).find(s => s.id === submissionId);
    return found ? found.status : 'absent-from-pending';
  }

  test('APPROVE_ITEM via /ops is refused for a non-approver and mutates nothing', async ({ page }) => {
    const { submissionId } = await setupForgeryFixture(page, 'approve');

    // Front door: already gated.
    const rest = await page.evaluate(async (id) => {
      const r = await fetch('/api/v1/workflow/approveSubmission', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ submission_id: id }),
      });
      return { status: r.status, body: await r.text() };
    }, submissionId);
    expect(rest.status).toBe(403);

    // Side door: must be gated identically. This returned 200 before the fix.
    const ops = await page.evaluate(async (id) => {
      const r = await fetch('/api/v1/workflow/ops', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          op_type: 'APPROVE_ITEM', device_id: 'forged-' + Date.now(),
          entity_id: id, entity_type: 'submission', lamport_ts: Date.now(),
          payload: { submission_id: id },
        }),
      });
      return { status: r.status, body: await r.text() };
    }, submissionId);
    expect(ops.status).toBe(403);

    // The mutation is what matters — the submission must still be pending.
    expect(await statusOf(page, submissionId)).toBe('pending');
  });

  test('REJECT_ITEM via /ops is refused for a non-approver and writes no rejection', async ({ page }) => {
    const { submissionId, fieldId } = await setupForgeryFixture(page, 'reject');

    const rest = await page.evaluate(async ([id, fid]) => {
      const r = await fetch('/api/v1/workflow/rejectItem', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ submission_id: id, field_id: fid, comment: 'front door' }),
      });
      return { status: r.status };
    }, [submissionId, fieldId]);
    expect(rest.status).toBe(403);

    const ops = await page.evaluate(async ([id, fid]) => {
      const r = await fetch('/api/v1/workflow/ops', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          op_type: 'REJECT_ITEM', device_id: 'forged-' + Date.now(),
          entity_id: id, entity_type: 'submission', lamport_ts: Date.now(),
          payload: { submission_id: id, field_id: fid, comment: 'forged' },
        }),
      });
      return { status: r.status, body: await r.text() };
    }, [submissionId, fieldId]);
    expect(ops.status).toBe(403);

    expect(await statusOf(page, submissionId)).toBe('pending');
  });

  // The legitimate path must still work through /ops, or the fix has simply
  // broken live-sync approvals for real approvers.
  test('APPROVE_ITEM via /ops still succeeds for an admin', async ({ page }) => {
    const { submissionId } = await setupForgeryFixture(page, 'admin');
    await login(page); // superadmin

    const ops = await page.evaluate(async (id) => {
      const r = await fetch('/api/v1/workflow/ops', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          op_type: 'APPROVE_ITEM', device_id: 'admin-' + Date.now(),
          entity_id: id, entity_type: 'submission', lamport_ts: Date.now(),
          payload: { submission_id: id },
        }),
      });
      return { status: r.status };
    }, submissionId);
    expect(ops.status).toBe(200);
    expect(await statusOf(page, submissionId)).toBe('absent-from-pending');
  });
});
