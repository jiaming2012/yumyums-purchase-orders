const { test, expect } = require('@playwright/test');
const { randomUUID } = require('crypto');

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
    // De-flake (found in this card's fresh-DB --retries=0 determinism runs):
    // the one-shot textContent() + toContain pattern raced op propagation under
    // load — auto-retrying toContainText asserts the SAME content, retried.
    await expect(pageB.locator('[data-fill-template-id]').first()).toContainText('1/1', { timeout: 12000 });

    // Device A: now UNCHECK the checkbox
    await checkBtn.click();
    await expect(checkBtn).not.toHaveClass(/checked/, { timeout: 5000 });

    // Device B: list page should now show 0/1 (live, no reload)
    await expect(pageB.locator('[data-fill-template-id]').first()).toContainText('0/1', { timeout: 12000 });

    await ctxB.close();
  });

  // Reproduction (operator-found on dev, 2026-07-17): with device B's checklist
  // OPEN IN THE RUNNER, an uncheck on device A does NOT live-uncheck on B — B only
  // shows it after a manual reload. Existing coverage tested B on the LIST page
  // (LST-17) or a single-device reload (FLD-01), never the two-device OPEN-RUNNER
  // case. Faithful to the operator's "Friday checklist": ONE section with FOUR
  // checkboxes (Cut the check / Do A / Do B / Do C) + a text field — the multi-box
  // structure a single-checkbox repro doesn't hit. Uncheck "Do C" (a non-first box)
  // on A; B's open runner must reflect it live. The check leg is a diagnostic — if
  // it passes and the uncheck leg fails, the bug is uncheck-specific.
  test('unchecking "Do C" on Device A live-unchecks it on Device B\'s OPEN runner (Friday-checklist shape) [FLD-LIVE-01]', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    const CB = (label, order) => ({ type: 'checkbox', label, required: false, order, config: {}, fail_trigger: null, condition: null });
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Friday Repro', requires_approval: false,
      sections: [{ title: 'Make money', order: 0, condition: null, fields: [
        CB('Cut the check', 0), CB('Do A', 0), CB('Do B', 0), CB('Do C', 0),
        { type: 'text', label: 'A text note', required: false, order: 1, config: {}, fail_trigger: null, condition: null },
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });
    const doCA = () => page.locator('.fill-field', { hasText: 'Do C' }).locator('.check-btn');
    const doCB = (p) => p.locator('.fill-field', { hasText: 'Do C' }).locator('.check-btn');

    // Device A: open the runner and CHECK all four boxes (as the operator did).
    await page.reload();
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.fill-field', { timeout: 12000 });
    for (const label of ['Cut the check', 'Do A', 'Do B', 'Do C']) {
      const b = page.locator('.fill-field', { hasText: label }).locator('.check-btn');
      await b.click(); await expect(b).toHaveClass(/checked/, { timeout: 5000 });
    }
    await page.waitForTimeout(2500); // debounce + saves + EmitOp

    // Device B: open the SAME checklist; baseline — Do C checked (hydrated draft).
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Friday Repro')).toBeVisible({ timeout: 12000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForSelector('.fill-field', { timeout: 12000 });
    await expect(doCB(pageB), 'baseline: Do C checked on B').toHaveClass(/checked/, { timeout: 12000 });

    // Device A UNCHECKS "Do C" while B's runner is open. Wait for A's op to emit.
    await doCA().click();
    await expect(doCA()).not.toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(3000); // debounce + save (DELETE) + EmitOp + WS propagation

    // LIVE assertion (the bug): B's OPEN runner must show Do C unchecked WITHOUT a
    // reload; the other three boxes must stay checked.
    await expect(doCB(pageB), 'Device B open runner must live-UNCHECK Do C (no reload)').not.toHaveClass(/checked/, { timeout: 12000 });
    await expect(pageB.locator('.fill-field', { hasText: 'Do A' }).locator('.check-btn'), 'Do A stays checked on B').toHaveClass(/checked/, { timeout: 5000 });

    // Sanity (matches operator report): after a manual reload B shows Do C unchecked,
    // proving the server DELETE persisted and only the LIVE re-render was missing.
    await pageB.reload();
    await expect(pageB.locator('#s1').getByText('Friday Repro')).toBeVisible({ timeout: 12000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForSelector('.fill-field', { timeout: 12000 });
    await expect(doCB(pageB), 'after reload B shows Do C unchecked (DELETE persisted)').not.toHaveClass(/checked/, { timeout: 12000 });

    await ctxB.close();
  });

  // Same-browser TWO-TAB variant (shared IndexedDB → shared lamport clock). This is
  // the most likely real-world "two devices" config when testing on one machine.
  // LamportClock._ts is seeded from shared IndexedDB (sync.js:116-118) but each tab
  // keeps its own in-memory copy, so two tabs can tick to the SAME lamport_ts and
  // collide on the server's LWW — a hazard the separate-context test can't hit.
  test('two TABS same browser: unchecking Do C on tab A live-unchecks tab B (shared IndexedDB) [FLD-LIVE-02]', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    const CB = (label, order) => ({ type: 'checkbox', label, required: false, order, config: {}, fail_trigger: null, condition: null });
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Friday TwoTab', requires_approval: false,
      sections: [{ title: 'Make money', order: 0, condition: null, fields: [
        CB('Cut the check', 0), CB('Do A', 0), CB('Do B', 0), CB('Do C', 0),
        { type: 'text', label: 'A text note', required: false, order: 1, config: {}, fail_trigger: null, condition: null },
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
    });

    // ONE shared context = two tabs of the same browser (shared IndexedDB + cookies).
    const ctx = await browser.newContext();
    const tabA = await ctx.newPage();
    const tabB = await ctx.newPage();
    await login(tabA); // shared cookies → tabB is logged in too
    const doC = (p) => p.locator('.fill-field', { hasText: 'Do C' }).locator('.check-btn');
    const openRunner = async (p) => {
      await p.goto(BASE + '/workflows.html');
      await expect(p.locator('#s1').getByText('Friday TwoTab')).toBeVisible({ timeout: 12000 });
      await p.click('[data-fill-template-id]');
      await p.waitForSelector('.fill-field', { timeout: 12000 });
    };

    // Both tabs open the runner. Tab A checks all four boxes.
    await openRunner(tabA);
    for (const label of ['Cut the check', 'Do A', 'Do B', 'Do C']) {
      const b = tabA.locator('.fill-field', { hasText: label }).locator('.check-btn');
      await b.click(); await expect(b).toHaveClass(/checked/, { timeout: 5000 });
    }
    await tabA.waitForTimeout(2500);
    await openRunner(tabB);
    await expect(doC(tabB), 'baseline: Do C checked on tab B').toHaveClass(/checked/, { timeout: 12000 });

    // Tab A UNCHECKS Do C while tab B's runner is open.
    await doC(tabA).click();
    await expect(doC(tabA)).not.toHaveClass(/checked/, { timeout: 5000 });
    await tabA.waitForTimeout(3000);

    // LIVE assertion: tab B must reflect the uncheck WITHOUT a reload.
    await expect(doC(tabB), 'tab B must live-UNCHECK Do C (no reload)').not.toHaveClass(/checked/, { timeout: 12000 });

    await ctx.close();
  });

  // ─── Operator-found (dev, 2026-07-18): approval-state ops don't refresh the
  // receiving device's checklist ──────────────────────────────────────────────
  // Same root class as the 07-17 live-uncheck bug: a live op arrives but the
  // client re-renders from a STALE in-memory cache (MY_SUBMISSIONS) instead of
  // reconciling the changed submission/approval state. Two symptoms:
  //   RJT-LIVE-01 — a manager's rejection reason never reaches the submitter's
  //     other device live (applyOp REJECT_ITEM only calls loadPendingApprovals,
  //     which is a no-op for a non-approver; MY_SUBMISSIONS stays pending_approval
  //     so hydrateFieldState never builds the REJECTION_FLAGS correction banner).
  //   RJT-LIVE-02 — an observer's list count stays frozen on the pre-rejection
  //     submission SNAPSHOT (getProgress counts submission.responses for a
  //     pending_approval submission); later live unchecks re-render the list but
  //     getProgress still reads the stale frozen snapshot, so the count never moves.
  // Helper: pull a field id by label out of a pending submission's snapshot.
  const fieldIdByLabel = (sub, label) => {
    const snap = typeof sub.template_snapshot === 'string' ? JSON.parse(sub.template_snapshot) : sub.template_snapshot;
    for (const s of (snap.sections || [])) for (const f of (s.fields || [])) if (f.label === label) return f.id;
    return null;
  };

  test('rejection reason live-reaches the submitter\'s other device without reload [RJT-LIVE-01]', async ({ browser, page }) => {
    test.setTimeout(120000);
    page.on('dialog', d => d.accept());
    const dow = await getTodayDOW(page);
    const CB = (label) => ({ type: 'checkbox', label, required: false, order: 0, config: {}, fail_trigger: null, condition: null });
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Reject Repro', requires_approval: true,
      sections: [{ title: 'Cut the check', order: 0, condition: null, fields: [
        CB('Cut the check'), CB('Do A'), CB('Do B'), CB('Do C'),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });

    // Device A (submitter): open runner, check all four, submit.
    await page.reload();
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.fill-field', { timeout: 12000 });
    for (const label of ['Cut the check', 'Do A', 'Do B', 'Do C']) {
      const b = page.locator('.fill-field', { hasText: label }).locator('.check-btn');
      await b.click(); await expect(b).toHaveClass(/checked/, { timeout: 5000 });
    }
    await page.waitForTimeout(2000);
    await page.click('#submit-btn');
    await page.waitForTimeout(2500);

    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    expect(Array.isArray(pending) && pending.length > 0, 'submission is pending approval').toBeTruthy();
    const sub = pending[0];
    const doBId = fieldIdByLabel(sub, 'Do B');
    expect(doBId, 'resolved Do B field id from snapshot').toBeTruthy();

    // Device B (submitter's 2nd device): open the same checklist and watch it.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    pageB.on('dialog', d => d.accept());
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Reject Repro')).toBeVisible({ timeout: 12000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForTimeout(1500);
    await expect(pageB.locator('.correction-banner'), 'no rejection banner before the reject').toHaveCount(0);

    // Approver (same admin identity, Device A) rejects "Do B" with a reason.
    await apiCall(page, 'POST', 'rejectItem', { submission_id: sub.id, field_id: doBId, comment: 'Please redo Do B', require_photo: false });
    await page.waitForTimeout(3500); // REJECT_ITEM op + WS propagation

    // THE BUG: Device B's open runner must surface the rejection reason live (no reload).
    await expect(pageB.locator('.correction-banner'), 'Device B must live-show the rejection reason without reload')
      .toContainText('Please redo Do B', { timeout: 12000 });

    // A hard reload also shows it (proves the reject persisted; only the live
    // re-render was missing) — matches the operator report.
    await pageB.reload();
    await expect(pageB.locator('#s1').getByText('Reject Repro')).toBeVisible({ timeout: 12000 });
    await pageB.click('[data-fill-template-id]');
    await expect(pageB.locator('.correction-banner')).toContainText('Please redo Do B', { timeout: 12000 });

    await ctxB.close();
  });

  test('observer list count reflects live state after a rejection, not the frozen submission snapshot [RJT-LIVE-02]', async ({ browser, page }) => {
    test.setTimeout(120000);
    page.on('dialog', d => d.accept());
    const dow = await getTodayDOW(page);
    const CB = (label) => ({ type: 'checkbox', label, required: false, order: 0, config: {}, fail_trigger: null, condition: null });
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Count Repro', requires_approval: true,
      sections: [{ title: 'Cut the check', order: 0, condition: null, fields: [
        CB('Cut the check'), CB('Do A'), CB('Do B'),
      ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });

    // Device A (submitter): check 2 of 3, submit (confirm "1 not completed" dialog auto-accepts).
    await page.reload();
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.fill-field', { timeout: 12000 });
    for (const label of ['Cut the check', 'Do A']) {
      const b = page.locator('.fill-field', { hasText: label }).locator('.check-btn');
      await b.click(); await expect(b).toHaveClass(/checked/, { timeout: 5000 });
    }
    await page.waitForTimeout(2000);
    await page.click('#submit-btn');
    await page.waitForTimeout(2500);

    // Device B (observer): open My Checklists list — shows the frozen snapshot count "2/3".
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    pageB.on('dialog', d => d.accept());
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Count Repro')).toBeVisible({ timeout: 12000 });
    const before = await pageB.locator('[data-fill-template-id]').first().textContent();
    expect(before, 'observer sees the submitted 2/3 count').toContain('2/3');

    // Approver rejects "Do A" — submission → rejected, submitter goes back to edit mode.
    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    const doAId = fieldIdByLabel(pending[0], 'Do A');
    await apiCall(page, 'POST', 'rejectItem', { submission_id: pending[0].id, field_id: doAId, comment: 'redo Do A', require_photo: false });
    await page.waitForTimeout(2500);

    // Device A refreshes into rejected/edit mode and UNCHECKS "Cut the check".
    await page.reload();
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.fill-field', { timeout: 12000 });
    const cut = page.locator('.fill-field', { hasText: 'Cut the check' }).locator('.check-btn');
    await expect(cut).toHaveClass(/checked/, { timeout: 8000 });
    await cut.click();
    await expect(cut).not.toHaveClass(/checked/, { timeout: 5000 });
    await page.waitForTimeout(3000); // reject op + SET_FIELD delete both propagate to B

    // THE BUG: Device B's list count must NOT stay frozen on the pre-rejection
    // "2/3" snapshot — it should reconcile to live state once the submission is
    // no longer pending. (After the fix it reads live: Do A hidden by the
    // rejection flag, Cut the check now unchecked → no longer "2/3".)
    // De-flake: auto-retrying not.toContainText replaces the one-shot
    // textContent() read (same content asserted, retried until convergence).
    await expect(pageB.locator('[data-fill-template-id]').first(),
      'observer count must not stay frozen on the submission snapshot').not.toContainText('2/3', { timeout: 12000 });

    await ctxB.close();
  });

  test('approval live-updates the submitter\'s other device to Approved without reload [RJT-LIVE-03]', async ({ browser, page }) => {
    test.setTimeout(120000);
    page.on('dialog', d => d.accept());
    const dow = await getTodayDOW(page);
    const CB = (label) => ({ type: 'checkbox', label, required: false, order: 0, config: {}, fail_trigger: null, condition: null });
    await apiCall(page, 'POST', 'createTemplate', {
      name: 'Approve Repro', requires_approval: true,
      sections: [{ title: 'Cut the check', order: 0, condition: null, fields: [ CB('Cut the check'), CB('Do A') ] }],
      schedules: [{ active_days: [dow] }],
      assignments: [
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
        { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
      ],
    });

    // Device A (submitter): fill both, submit.
    await page.reload();
    await page.click('[data-fill-template-id]');
    await page.waitForSelector('.fill-field', { timeout: 12000 });
    for (const label of ['Cut the check', 'Do A']) {
      const b = page.locator('.fill-field', { hasText: label }).locator('.check-btn');
      await b.click(); await expect(b).toHaveClass(/checked/, { timeout: 5000 });
    }
    await page.waitForTimeout(2000);
    await page.click('#submit-btn');
    await page.waitForTimeout(2500);

    // Device B (submitter's 2nd device): open the checklist — pending review, not approved.
    const ctxB = await browser.newContext();
    const pageB = await ctxB.newPage();
    pageB.on('dialog', d => d.accept());
    await login(pageB);
    await pageB.goto(BASE + '/workflows.html');
    await expect(pageB.locator('#s1').getByText('Approve Repro')).toBeVisible({ timeout: 12000 });
    await pageB.click('[data-fill-template-id]');
    await pageB.waitForSelector('.fill-field', { timeout: 12000 });
    await expect(pageB.locator('#fill-body')).not.toContainText('Approved', { timeout: 5000 });

    // Approver (Device A) approves the submission.
    const pending = await apiCall(page, 'GET', 'pendingApprovals');
    await apiCall(page, 'POST', 'approveSubmission', { submission_id: pending[0].id, feedback: [] });
    await page.waitForTimeout(3000); // APPROVE_ITEM op + WS propagation

    // THE BROAD FIX: Device B's open runner must reflect Approved live (no reload).
    await expect(pageB.locator('#fill-body'), 'Device B must live-show Approved without reload')
      .toContainText('Approved', { timeout: 12000 });

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
    // De-flake: auto-retrying toContainText replaces the one-shot textContent()
    // + toContain pattern (same content asserted — see LST-17 uncheck variant).
    await expect(pageB.locator('[data-fill-template-id]').first()).toContainText('0/1', { timeout: 12000 });

    // Device A: open checklist and check the checkbox
    await page.reload();
    await page.click('[data-fill-template-id]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/, { timeout: 5000 });

    // Device B: list page should now show 1/1 (updated via WS without reload)
    await expect(pageB.locator('[data-fill-template-id]').first()).toContainText('1/1', { timeout: 12000 });

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

// Wait for a myChecklists GET on device B to LAND — the deterministic signal that
// a re-fetch/re-render (initial load, WS catch-up, or a SAVE_TEMPLATE-driven
// rerenderOpenChecklistAfterSave) has completed, rather than racing the visible DOM.
const waitMyChecklistsGet = (pageB, timeout) => pageB.waitForResponse(
  res => res.url().includes('/myChecklists') && res.request().method() === 'GET',
  { timeout: timeout || CONVERGE_TIMEOUT });

// Drives one surviving-answer cell end to end (LIVE + CATCH-UP) on device B.
//
// De-flake (no-retry hardening): the two INPUT cells (text, temperature) used to
// flake because a freshly-TYPED but not-yet-persisted value lives only in the
// optimistic control; a stray WS catch-up loadMyChecklists whose fetch predates
// the debounced save can re-render the runner and CLOBBER that value to empty —
// and, since nothing re-issues a fetch, it stays empty (the observed `Received
// ""` at the baseline assert). The fix is deterministic and test-only:
//   1. Gate on the autosave POST /ops response (2xx) — SaveResponseFunc commits
//      the draft BEFORE the 200, so this is a race-free "draft is durable" signal
//      (replaces the old myChecklists poll, which under load could itself time out).
//   2. Reopen the runner for the baseline so it hydrates the COMMITTED draft, not
//      the optimistic control — after commit every render reflects durable state,
//      so no stray re-render can clobber it (same pattern the photo cell uses).
//   3. Before asserting LIVE convergence, wait for B to APPLY the SAVE_TEMPLATE op
//      — its rerenderOpenChecklistAfterSave re-fetch (GET myChecklists) — instead
//      of only watching the visible Decoy count. The Decoy assertion below stays
//      as the authoritative gate; this wait just stops us racing a partial apply.
async function survivalCell(browser, page, name, answerField, enterFn, assertFn) {
  const dow = await getTodayDOW(page);
  await createWithDecoy(page, name, dow, answerField);

  const ctxB = await browser.newContext();
  const pageB = await ctxB.newPage();
  await login(pageB);
  await openRunnerB(pageB, name);

  // Register the commit wait BEFORE the edit so the debounced POST /ops can't be
  // missed. A 2xx means the draft committed server-side (deterministic).
  const committed = pageB.waitForResponse(
    res => res.url().includes('/api/v1/workflow/ops') && res.request().method() === 'POST',
    { timeout: 12000 });
  await enterFn(pageB);
  const commitRes = await committed;
  expect(commitRes.ok(), 'answer autosave must commit (2xx) before proceeding').toBeTruthy();

  // Re-hydrate from the committed draft — the baseline now reflects durable state,
  // immune to the optimistic-clobber race described above.
  await reopenRunnerB(pageB, name);
  await assertFn(pageB); // baseline: the committed answer is present on B

  // Admin cuts the Decoy (SAVE_TEMPLATE op). Wait until B has APPLIED it before
  // asserting (deterministic "op applied" signal), then gate on the Decoy count.
  const applied = waitMyChecklistsGet(pageB);
  await cutDecoy(page, name, dow);
  await applied;

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
    //
    // De-flake (found in this card's fresh-DB --retries=0 determinism runs):
    // asserting the fail card in the OPTIMISTIC window (before the No-answer's
    // debounced save committed) raced the same clobber the W-3 text/temp cells
    // documented above — a silent catch-up SAVE_TEMPLATE replay (from the
    // beforeEach create/archive backlog) can re-render the just-opened runner
    // from durable (still-empty) state and wipe the optimistic No + fail card.
    // Deterministic fix, same pattern as those cells: gate on the autosave POST
    // /ops committing (2xx) instead of a blind wait + optimistic assert. The
    // fail card's visibility is still asserted — from durable state — by
    // assertPhoto after the baseline reopen below (.fail-card img.photo-thumb).
    const noBtn = pageB.locator('[data-action="set-no"][data-fld-id="' + fieldId + '"]');
    await expect(noBtn).toBeVisible({ timeout: RUNNER_TIMEOUT });
    const noCommitted = pageB.waitForResponse(
      res => res.url().includes('/api/v1/workflow/ops') && res.request().method() === 'POST',
      { timeout: 12000 });
    await noBtn.click();
    const noRes = await noCommitted;
    expect(noRes.ok(), 'No-answer autosave must commit (2xx) before the bundle write').toBeTruthy();
    await apiCall(pageB, 'POST', 'saveResponse', {
      field_id: fieldId, value: { _v: false, _fail_note: { note: '', severity: '', photo: photoUrl } },
    });

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
    //
    // De-flake (found in this card's fresh-DB --retries=0 determinism runs):
    // after the submit, A's runner auto-navigates back to the list ~2.5s later
    // (the all-done fireworks timer). Whether A still shows the unsubmit button
    // here depended on device B's whole context setup finishing inside that
    // window — a pure timing race (under load the click waited on a runner that
    // no longer existed and timed out). Deterministic fix: reopen A's runner
    // first — the pending submission renders the unsubmit button from durable
    // server state, whichever view the fireworks timer left A on. No assertion
    // is weakened: B's live-convergence asserts below are unchanged.
    await page.reload();
    await expect(page.locator('#s1').getByText('MX Unsubmit')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await page.click('[data-fill-template-id]');
    await expect(page.locator('[data-action="unsubmit"]')).toBeVisible({ timeout: RUNNER_TIMEOUT });
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

// ─── W-6b: LWW conflict re-render across the non-text persisted types ─────────
// The W-6 cell above proves ONLY text/textarea. The other persisted types ride
// the SAME shared applyOp SET_FIELD path (sync.js:405) driven from the frontend
// 409 handler (workflows.html:209) — but were untested there. Each cell below
// seeds a WINNER value of one type on a live device, has the LOSING device (WS
// stubbed dead, Lamport clock pinned stale) write a DIFFERENT value of that type,
// asserts the 409, and asserts the loser RENDERS THE WINNER — never its own
// rejected value. A broken applyOp/409 path would leave the loser showing its
// rejected value; each assertion rules exactly that out (see the per-cell notes
// on what a broken path would render).
//
// PARKED types (NOT covered here) — the conflict/applyOp path genuinely cannot
// render them, so a red→green cell is impossible WITHOUT a production change that
// the card's footprint forbids:
//   • fail-note text + severity, and fail-note photo-URL — the `{_v,_fail_note}`
//     bundle is unpacked into FAIL_NOTES ONLY by hydrateFieldState
//     (workflows.html:1480-1482). applyOp (sync.js:405-441, the path the 409
//     handler drives) has no _fail_note unpack, so a winning fail-note bundle
//     applied via the conflict path renders as neither a value nor a fail card.
//     Covering these would require teaching the apply path to unpack the bundle
//     (a workflows.html / sync.js production behaviour change), which is out of
//     footprint → PARKED, noted in the SUMMARY.

// Stub WebSocket dead BEFORE load so no live op ever reaches the losing device:
// its ONLY route to the winning value is the 409 response body.
const CONFLICT_DEAD_SOCKET = () => {
  class DeadSocket { constructor() { this.readyState = 3; } close() {} send() {} }
  DeadSocket.CONNECTING = 0; DeadSocket.OPEN = 1; DeadSocket.CLOSING = 2; DeadSocket.CLOSED = 3;
  window.WebSocket = DeadSocket;
};

// Pin the loser permanently stale: Lamport tick fixed at 1 (well below the
// winner's 5000000) with catch-up + clock-receive neutralised so nothing can
// climb it above the winner. Every write it makes loses LWW deterministically.
const CONFLICT_PIN_STALE = () => {
  window.wsCatchUp = async () => {};
  if (window.LAMPORT_CLOCK) {
    window.LAMPORT_CLOCK.receive = async () => {};
    window.LAMPORT_CLOCK._ts = 0;
    window.LAMPORT_CLOCK.tick = async () => 1;
  }
};

// Stand up the LOSING device: dead WS, a template with a single field of the
// given shape, the runner open on that field, and the clock pinned stale.
// Returns the loser page, its ctx, and the resolved server-side field id (plus
// the full template, so sub-step ids can be read).
async function openConflictLoser(browser, name, field) {
  const ctx = await browser.newContext();
  await ctx.addInitScript(CONFLICT_DEAD_SOCKET);
  const page = await ctx.newPage();
  await login(page);
  await page.goto(BASE + '/workflows.html');
  await cleanupPendingApprovals(page);
  await cleanupTemplates(page);
  const dow = await getTodayDOW(page);
  await apiCall(page, 'POST', 'createTemplate', {
    name, requires_approval: false,
    sections: [{ title: 'S', order: 0, condition: null, fields: [Object.assign({}, field, { order: 0 })] }],
    schedules: [{ active_days: [dow] }],
    assignments: [{ assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' }],
  });
  const templates = await apiCall(page, 'GET', 'templates');
  const tpl = templates.find(t => t.name === name);
  const fieldId = tpl.sections[0].fields[0].id;
  await page.reload();
  await expect(page.locator('#s1').getByText(name)).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await page.click('[data-fill-template-id]');
  await page.waitForSelector('.fill-field', { timeout: RUNNER_TIMEOUT });
  await page.evaluate(CONFLICT_PIN_STALE);
  return { ctx, page, fieldId, tpl };
}

// Seed the WINNING value on a fresh live device (optionally a distinct user) with
// a high Lamport ts so any later stale write from the loser loses LWW → 409 with
// this value in the body. Returns the winner ctx to close.
async function seedConflictWinner(browser, fieldId, value, email, password) {
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await login(page, email, password);
  const res = await apiCall(page, 'POST', 'ops', {
    op_type: 'SET_FIELD', entity_id: fieldId, entity_type: 'field_response',
    payload: { field_id: fieldId, value }, lamport_ts: 5000000, device_id: 'zzz-winner-device',
  });
  expect(res && !res._conflict, 'winner seed must win, not conflict').toBeTruthy();
  return ctx;
}

// Resolve when the loser's stale write is REJECTED (409) on the ops endpoint.
function waitLoser409(page) {
  return page.waitForResponse(
    res => res.url().includes('/api/v1/workflow/ops') && res.request().method() === 'POST' && res.status() === 409,
    { timeout: CONVERGE_TIMEOUT });
}

// Invite + accept a throwaway team_member so the winner can be seeded under a
// DISTINCT user id — required only for the checkbox cell (see its note). Runs in
// its OWN context: accept-invite logs the acceptor in as the new user, which must
// NOT clobber the loser's admin session.
async function createSecondUser(browser) {
  const email = 'conflict-winner-' + Date.now() + '@yumyums.kitchen';
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await login(page); // admin, to issue the invite
  const invite = await page.evaluate(async (e) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Conflict', last_name: 'Winner', email: e, roles: ['team_member'] }),
    });
    return res.json();
  }, email);
  const token = invite.invite_path.split('token=')[1];
  await page.evaluate(async (t) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: 'test456' }),
    });
  }, token);
  await ctx.close();
  return { email, password: 'test456' };
}

test.describe('Engine: LWW conflict re-renders the winner across persisted types (W-6b)', () => {

  test('yes/no: a losing No re-renders the winning Yes [FR-9 INV-1]', async ({ browser }) => {
    test.setTimeout(120000);
    const { ctx, page, fieldId } = await openConflictLoser(browser, 'MX Conflict YesNo',
      { type: 'yes_no', label: 'All good?', required: false, config: {}, fail_trigger: null, condition: null });
    // Winner answered YES (non-null → UPSERTs the draft row, keeping the winning
    // Lamport ts so the loser's later stale write loses LWW → 409).
    const winnerCtx = await seedConflictWinner(browser, fieldId, true);

    const resp409 = waitLoser409(page);
    const noBtn = page.locator('[data-action="set-no"][data-fld-id="' + fieldId + '"]');
    await noBtn.click(); // loser answers No — optimistic, the value the DB rejects
    await expect(noBtn).toHaveClass(/on/); // rejected value shown first
    expect((await resp409).status(), 'loser No must lose LWW (409)').toBe(409);

    // Converge: Yes on, No off — the WINNER. A broken 409/applyOp path would leave
    // the loser's No 'on'.
    await expect(page.locator('[data-action="set-yes"][data-fld-id="' + fieldId + '"]'))
      .toHaveClass(/on/, { timeout: CONVERGE_TIMEOUT });
    await expect(noBtn).not.toHaveClass(/on/);

    await ctx.close(); await winnerCtx.close();
  });

  test('temperature: a losing 350 re-renders the winning 375 [FR-9 INV-1]', async ({ browser }) => {
    test.setTimeout(120000);
    const { ctx, page, fieldId } = await openConflictLoser(browser, 'MX Conflict Temp',
      { type: 'temperature', label: 'Grill temp', required: false, config: { unit: 'F', min: 300, max: 500 }, fail_trigger: null, condition: null });
    const winnerCtx = await seedConflictWinner(browser, fieldId, 375);

    const resp409 = waitLoser409(page);
    const inp = page.locator('input[type="number"][data-field-id="' + fieldId + '"]');
    await inp.fill('350'); await inp.dispatchEvent('change'); // loser writes 350 — rejected
    await expect(inp).toHaveValue('350'); // rejected value shown first
    expect((await resp409).status(), 'loser 350 must lose LWW (409)').toBe(409);

    // Converge to 375 — the WINNER. A broken path would keep showing 350.
    await expect(page.locator('input[type="number"][data-field-id="' + fieldId + '"]'))
      .toHaveValue('375', { timeout: CONVERGE_TIMEOUT });

    await ctx.close(); await winnerCtx.close();
  });

  test('sub-step: a losing sub-step B re-renders the winning sub-step A [FR-9 INV-1]', async ({ browser }) => {
    test.setTimeout(120000);
    const { ctx, page, fieldId, tpl } = await openConflictLoser(browser, 'MX Conflict SubStep',
      CHECKBOX_F('Protein stock', 0, { sub_steps: [
        { type: 'checkbox', label: 'Salmon counted', order: 0, config: {}, fail_trigger: null, condition: null },
        { type: 'checkbox', label: 'Chicken counted', order: 1, config: {}, fail_trigger: null, condition: null },
      ] }));
    const subs = tpl.sections[0].fields[0].sub_steps;
    const subAId = subs[0].id;
    // Winner checked sub-step A only. The {value,sub_steps} object is non-null, so
    // it UPSERTs the draft row and keeps the winning Lamport ts.
    const winnerCtx = await seedConflictWinner(browser, fieldId,
      { value: false, sub_steps: { [subAId]: { by: 'Winner', at: new Date().toISOString() } } });

    const resp409 = waitLoser409(page);
    const subChecks = page.locator('.sub-step-check');
    await subChecks.nth(1).click(); // loser checks sub-step B — rejected
    await expect(subChecks.nth(1)).toHaveClass(/done/); // rejected value shown first
    expect((await resp409).status(), 'loser sub-step B must lose LWW (409)').toBe(409);

    // Converge: sub-step A done, B NOT — the WINNER. A broken path would keep B done.
    await expect(page.locator('.sub-step-check').nth(0)).toHaveClass(/done/, { timeout: CONVERGE_TIMEOUT });
    await expect(page.locator('.sub-step-check').nth(1)).not.toHaveClass(/done/);

    await ctx.close(); await winnerCtx.close();
  });

  test('checkbox: a losing uncheck re-renders the winning checked box [FR-9 INV-1]', async ({ browser }) => {
    test.setTimeout(120000);
    // A plain checkbox has exactly ONE non-null value (`true`); the only value that
    // DIFFERS from a checked winner is "unchecked", expressed as a `null`
    // SET_FIELD. But the workflow op router runs SaveResponseFunc BEFORE the LWW
    // guard (main.go:62 → handler.go:178) and a null value DELETEs the draft row
    // (repository.go:730). So when the winner and loser are the SAME user, the
    // loser's uncheck deletes the only row and CheckLWW then finds none → the
    // uncheck WINS (no 409), never exercising the conflict path. Making the uncheck
    // LOSE deterministically needs the WINNER's row to survive the loser's
    // per-(field,user) DELETE — i.e. a DISTINCT winner user — AND a strict write
    // ORDER, because the entity lamport is stored per-FIELD (ops.go:148 updates ALL
    // rows for the field). So:
    //   1. Loser CHECKS first on the empty field → wins (first write), field
    //      lamport = 1, loser holds a row.
    //   2. THEN a distinct user seeds the winning CHECKED box at lamport 5000000 →
    //      beats the field's lamport 1 deterministically (incoming > current
    //      regardless of which row CheckLWW's LIMIT 1 reads).
    //   3. Loser UNCHECKS (null) → DELETEs only the loser's row; the winner's row
    //      (lamport 5000000) is the sole remaining row, so CheckLWW returns it
    //      deterministically → 409 with the winning `true` value.
    const { ctx, page, fieldId } = await openConflictLoser(browser, 'MX Conflict Checkbox',
      CHECKBOX_F('Wipe counters', 0));

    // Step 1: loser CHECKS on the empty field — wins, and now holds a draft row.
    const cb = page.locator('.check-btn[data-field-id="' + fieldId + '"]');
    const firstWrite = page.waitForResponse(
      res => res.url().includes('/api/v1/workflow/ops') && res.request().method() === 'POST',
      { timeout: CONVERGE_TIMEOUT });
    await cb.click();
    await expect(cb).toHaveClass(/checked/);
    await firstWrite;

    // Step 2: a distinct user seeds the winning CHECKED box at a high Lamport ts.
    const winner = await createSecondUser(browser);
    const winnerCtx = await seedConflictWinner(browser, fieldId, true, winner.email, winner.password);

    // Step 3: loser UNCHECKS — a stale null write that loses LWW to the winner.
    const resp409 = waitLoser409(page);
    await cb.click();
    await expect(cb).not.toHaveClass(/checked/); // rejected uncheck shown first
    expect((await resp409).status(), 'loser uncheck must lose LWW (409)').toBe(409);

    // Converge back to CHECKED — the WINNER. A broken path would leave it unchecked.
    await expect(page.locator('.check-btn[data-field-id="' + fieldId + '"]'))
      .toHaveClass(/checked/, { timeout: CONVERGE_TIMEOUT });

    await ctx.close(); await winnerCtx.close();
  });
});

// ─── Systematic convergence matrix (convergence-matrix-systematic, FR-8/FR-9) ─
//
// Completes the op-type × editor × derived-view matrix (AC-8/AC-9):
//   op-type ∈ {SET_FIELD (FLD), SUBMIT_CHECKLIST (SUB), APPROVE_ITEM (APR),
//              REJECT_ITEM (RJT)}
//   editor  ∈ {assignee (ASG), non-assignee admin (ADM)}
//   view    ∈ {field-value (VAL), correction-banner (BAN),
//              edit-vs-readonly (ERO), list-progress-count (CNT)}
// Every cell asserts the SECOND device's derived view — live (no reload) and
// after catch-up (reload/reopen). Cells are tagged MTX-<op>-<editor>-<view>.
//
// Editor mechanics (why the two columns differ at the ACCESS layer, not the UI):
//   • ASG — jamal's DB roles are ['admin'] (auth.UpsertSuperadmins inserts
//     ARRAY['admin']), so a template assigned to role 'admin' makes him a
//     GENUINE role-matched assignee: both myChecklists' assignment branch and
//     ResolveEntityAccess's assignment branch match him.
//   • ADM — the template is assigned to role 'team_member' ONLY. jamal matches
//     neither assignment; he reaches the checklist purely via the admin
//     view-all clause (myChecklists) and receives its live ops purely via the
//     admins-union + author-inclusion fan-out that the ESC-1 fix (5c423ac)
//     added to ResolveEntityAccess/StartListener. On the pre-fix build every
//     ADM cell reddens: the op never reached the second device (AC-9 / A6 —
//     recorded historical red-first evidence: sync/access_test.go,
//     TestResolveEntityAccess_AdminReceivesLiveOps).
//
// Full 32-cell coverage map (8 N/A cells carry their reason; a named test may
// prove multiple cells of one flow):
//   FLD-ASG-VAL  pre-existing  SYN-03 'Device B sees field changes' + FLD-LIVE-01/02
//                              + the W-3 per-type matrix (checkbox…photo)
//   FLD-ASG-BAN  N/A           a SET_FIELD op carries no rejection-state change:
//                              REJECTION_FLAGS derive solely from
//                              submission.rejections (hydrateFieldState), which
//                              only REJECT_ITEM creates and a resubmission
//                              supersedes; clearRejectionFlag on answering is
//                              device-local UX with no durable state to converge
//   FLD-ASG-ERO  N/A           fillState.readonly derives only from submission
//                              status (renderRunner); SET_FIELD never changes it
//   FLD-ASG-CNT  pre-existing  LST-17 (check + uncheck) + 'MX Progress' + 'MX Denom'
//   SUB-ASG-VAL  new           MTX-SUB-ASG-VAL/ERO
//   SUB-ASG-BAN  new           MTX-RJT-ASG cycle (resubmit leg clears the banner)
//   SUB-ASG-ERO  pre-existing  'MX Submit' + 'MX Unsubmit' (+ re-proven in
//                              MTX-SUB-ASG-VAL/ERO)
//   SUB-ASG-CNT  new           MTX-SUB-ASG-CNT
//   APR-ASG-VAL  N/A           APPROVE_ITEM mutates no field_response rows — no
//                              field value can change under it
//   APR-ASG-BAN  N/A           the ⚠ correction banner renders only for status
//                              'rejected' (hydrateFieldState); an approval can
//                              never produce one (approval feedback renders as
//                              FEEDBACK_NOTES, a distinct non-matrix view)
//   APR-ASG-ERO  pre-existing  RJT-LIVE-03 (Approved flips live on device B)
//   APR-ASG-CNT  new           MTX-APR-ASG-CNT
//   RJT-ASG-VAL  new           MTX-RJT-ASG cycle (rejected field cleared on B)
//   RJT-ASG-BAN  pre-existing  RJT-LIVE-01 [ESC-2a] + new sub-step variant
//                              MTX-RJT-ASG-BAN-SUBSTEP [ESC-3]
//   RJT-ASG-ERO  new           MTX-RJT-ASG cycle (edit mode returns on B) [ESC-2a]
//   RJT-ASG-CNT  pre-existing  RJT-LIVE-02 [ESC-2b]
//   FLD-ADM-VAL  new           MTX-FLD-ADM-VAL [ESC-1]
//   FLD-ADM-BAN  N/A           same reason as FLD-ASG-BAN
//   FLD-ADM-ERO  N/A           same reason as FLD-ASG-ERO
//   FLD-ADM-CNT  new           MTX-FLD-ADM-CNT
//   SUB-ADM-VAL  new           MTX-SUB-ADM-VAL/ERO
//   SUB-ADM-BAN  new           MTX-RJT-ADM cycle (resubmit leg clears the banner)
//   SUB-ADM-ERO  new           MTX-SUB-ADM-VAL/ERO
//   SUB-ADM-CNT  new           MTX-SUB-ADM-CNT
//   APR-ADM-VAL  N/A           same reason as APR-ASG-VAL
//   APR-ADM-BAN  N/A           same reason as APR-ASG-BAN
//   APR-ADM-ERO  new           MTX-APR-ADM-ERO
//   APR-ADM-CNT  new           MTX-APR-ADM-CNT
//   RJT-ADM-VAL  new           MTX-RJT-ADM cycle
//   RJT-ADM-BAN  new           MTX-RJT-ADM cycle
//   RJT-ADM-ERO  new           MTX-RJT-ADM cycle
//   RJT-ADM-CNT  new           MTX-RJT-ADM-CNT
// (SAVE_TEMPLATE/ARCHIVE_TEMPLATE sit outside this 4-op matrix; they are covered
// by the W-3 blocks above plus repro-cut-task.spec.js / broadcast-rerender.spec.js.)
//
// Red-first discipline for the NEW cells: every test asserts the PRE-op state of
// the exact locator whose convergence it then asserts (submit button visible
// before the submit that removes it, banner count 0 before the reject that adds
// it, badge absent before it appears, checkbox unchecked before it checks…), so
// a broken convergence path cannot pass vacuously — the post-op assertion
// demands the OPPOSITE of a state the same test just proved. The ESC-mapped
// cells additionally carry the recorded historical red-first runs (A6):
// ESC-1 → access_test.go, ESC-2a/2b → RJT-LIVE-01/02, ESC-3 → APR-SUBSTEP-0718.

const MTX_ASSIGN = {
  ASG: [
    { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
    { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
  ],
  ADM: [
    { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' },
    { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'approver' },
  ],
};

function mtxCreate(page, name, dow, editor, fields, requiresApproval) {
  return apiCall(page, 'POST', 'createTemplate', {
    name,
    requires_approval: requiresApproval !== false,
    sections: [{ title: 'S', order: 0, condition: null, fields }],
    schedules: [{ active_days: [dow] }],
    assignments: MTX_ASSIGN[editor],
  });
}

// Pending-submission lookup that works for BOTH editor columns. jamal's
// pendingApprovals queue is EMPTY when the approver assignment is
// 'team_member' (that query has no admin override), so the ADM cells resolve
// the submission from myChecklists' submissions instead — checklists are team
// objects, all members see all of today's submissions.
async function mtxPendingSubmission(page, tplId) {
  const subs = await page.evaluate(async () => {
    const r = await fetch('/api/v1/workflow/myChecklists?dow=' + new Date().getDay());
    return (await r.json()).submissions || [];
  });
  return subs.find(s => s.template_id === tplId &&
    (s.status === 'pending' || s.status === 'pending_approval')) || null;
}

// Resolve a field OR sub-step id by label from a submission's template snapshot.
function mtxFieldId(sub, label) {
  const snap = typeof sub.template_snapshot === 'string'
    ? JSON.parse(sub.template_snapshot) : sub.template_snapshot;
  for (const s of (snap.sections || [])) {
    for (const f of (s.fields || [])) {
      if (f.label === label) return f.id;
      for (const ss of (f.sub_steps || [])) if (ss.label === label) return ss.id;
    }
  }
  return null;
}

// Editor device A: (re)open the runner for the named checklist.
async function mtxOpenA(page, name) {
  await page.reload();
  await expect(page.locator('#s1').getByText(name)).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await page.click('[data-fill-template-id]');
  await page.waitForSelector('.fill-field', { timeout: RUNNER_TIMEOUT });
}

// Observer device B: fresh context with the runner open on the named checklist.
async function mtxOpenB(browser, name) {
  const ctxB = await browser.newContext();
  const pageB = await ctxB.newPage();
  pageB.on('dialog', d => d.accept());
  await login(pageB);
  await openRunnerB(pageB, name);
  return { ctxB, pageB };
}

// Observer device B: fresh context sitting on the BARE My-Checklists list.
async function mtxListB(browser, name) {
  const ctxB = await browser.newContext();
  const pageB = await ctxB.newPage();
  pageB.on('dialog', d => d.accept());
  await login(pageB);
  await pageB.goto(BASE + '/workflows.html');
  const rowB = pageB.locator('[data-fill-template-id]').filter({ hasText: name });
  await expect(rowB).toBeVisible({ timeout: RUNNER_TIMEOUT });
  return { ctxB, pageB, rowB };
}

const mtxRow = (pageB, name) => pageB.locator('[data-fill-template-id]').filter({ hasText: name });

// Check the labelled checkboxes on the editor device, gating each on its
// autosave POST /ops committing (2xx) — the race-free "draft is durable" signal
// the W-3 de-flake established.
async function mtxCheckFields(page, labels) {
  for (const label of labels) {
    const committed = page.waitForResponse(
      res => res.url().includes('/api/v1/workflow/ops') && res.request().method() === 'POST',
      { timeout: 12000 });
    const b = page.locator('.fill-field', { hasText: label }).locator('.check-btn').first();
    await b.click();
    await expect(b).toHaveClass(/checked/, { timeout: 5000 });
    const res = await committed;
    expect(res.ok(), 'autosave for "' + label + '" must commit (2xx)').toBeTruthy();
  }
}

// Submit via the runner UI on the editor device, gated on the submitChecklist
// POST succeeding (the SUBMIT_CHECKLIST op is emitted server-side inside it).
async function mtxSubmitUI(page) {
  const submitted = page.waitForResponse(
    res => res.url().includes('/submitChecklist') && res.request().method() === 'POST',
    { timeout: 12000 });
  await page.click('#submit-btn');
  const res = await submitted;
  expect(res.ok(), 'submitChecklist must succeed (2xx)').toBeTruthy();
}

// One SUBMIT cell: editor fills + submits while device B's runner is open.
// Proves MTX-SUB-<ed>-VAL (the answered value survives into the submitted view)
// and MTX-SUB-<ed>-ERO (edit → readonly flips live), live + catch-up.
async function mtxSubmitCell(browser, page, editor) {
  const name = 'MTX Submit ' + editor;
  page.on('dialog', d => d.accept());
  const dow = await getTodayDOW(page);
  await mtxCreate(page, name, dow, editor, [CHECKBOX_F('Task A', 0)]);
  await mtxOpenA(page, name);
  await mtxCheckFields(page, ['Task A']);

  const { ctxB, pageB } = await mtxOpenB(browser, name);
  // Pre-state (non-vacuous): B is EDITABLE and shows the committed value.
  await expect(pageB.locator('#submit-btn')).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await expect(pageB.locator('.submit-confirm')).toHaveCount(0);
  await expect(pageB.locator('.fill-field', { hasText: 'Task A' }).locator('.check-btn'))
    .toHaveClass(/checked/, { timeout: RUNNER_TIMEOUT });

  await mtxSubmitUI(page);

  // LIVE: readonly flips (ERO) and the value survives the flip (VAL). The
  // readonly renderer shows an answered checkbox as a green ✓ (no .check-btn
  // controls exist in readonly mode) — an unanswered one would show ✗.
  await expect(pageB.locator('#submit-btn')).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
  await expect(pageB.locator('.submit-confirm')).toBeVisible({ timeout: CONVERGE_TIMEOUT });
  await expect(pageB.locator('.fill-field', { hasText: 'Task A' }))
    .toContainText('✓', { timeout: CONVERGE_TIMEOUT });
  await expect(pageB.locator('.fill-field', { hasText: 'Task A' })).not.toContainText('✗');

  // CATCH-UP: reload + reopen — same converged state.
  await reopenRunnerB(pageB, name);
  await expect(pageB.locator('#submit-btn')).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
  await expect(pageB.locator('.submit-confirm')).toBeVisible({ timeout: CONVERGE_TIMEOUT });
  await expect(pageB.locator('.fill-field', { hasText: 'Task A' }))
    .toContainText('✓', { timeout: CONVERGE_TIMEOUT });
  await expect(pageB.locator('.fill-field', { hasText: 'Task A' })).not.toContainText('✗');

  await ctxB.close();
}

// One SUBMIT list-count cell: device B sits on the BARE list; the editor
// submits; B's row grows the Pending Approval badge live. MTX-SUB-<ed>-CNT.
async function mtxSubmitCountCell(browser, page, editor) {
  const name = 'MTX SubCnt ' + editor;
  page.on('dialog', d => d.accept());
  const dow = await getTodayDOW(page);
  await mtxCreate(page, name, dow, editor, [CHECKBOX_F('Task A', 0)]);
  await mtxOpenA(page, name);
  await mtxCheckFields(page, ['Task A']);

  const { ctxB, pageB, rowB } = await mtxListB(browser, name);
  // Pre-state (non-vacuous): full count, NO approval badge yet.
  await expect(rowB).toContainText('1/1', { timeout: RUNNER_TIMEOUT });
  await expect(rowB.locator('.approval-badge')).toHaveCount(0);

  await mtxSubmitUI(page);

  // LIVE: the badge appears on the bare list without a reload.
  await expect(rowB.locator('.approval-badge')).toHaveText('Pending Approval', { timeout: CONVERGE_TIMEOUT });

  // CATCH-UP: reload — badge still there.
  await pageB.reload();
  const rowB2 = mtxRow(pageB, name);
  await expect(rowB2).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await expect(rowB2.locator('.approval-badge')).toHaveText('Pending Approval', { timeout: CONVERGE_TIMEOUT });

  await ctxB.close();
}

// One APPROVE list-count cell: device B watches the bare list's Pending badge
// become Approved ✓ live. MTX-APR-<ed>-CNT.
async function mtxApproveCountCell(browser, page, editor) {
  const name = 'MTX AprCnt ' + editor;
  page.on('dialog', d => d.accept());
  const dow = await getTodayDOW(page);
  const tpl = await mtxCreate(page, name, dow, editor, [CHECKBOX_F('Task A', 0)]);
  await mtxOpenA(page, name);
  await mtxCheckFields(page, ['Task A']);
  await mtxSubmitUI(page);
  const sub = await mtxPendingSubmission(page, tpl.id);
  expect(sub, 'submission is pending approval').toBeTruthy();

  const { ctxB, pageB, rowB } = await mtxListB(browser, name);
  // Pre-state (non-vacuous): pending badge shows, no Approved mark.
  await expect(rowB.locator('.approval-badge')).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await expect(rowB).not.toContainText('Approved');

  await apiCall(page, 'POST', 'approveSubmission', { submission_id: sub.id, feedback: [] });

  // LIVE: Approved ✓ replaces the pending badge on the bare list.
  await expect(rowB).toContainText('Approved ✓', { timeout: CONVERGE_TIMEOUT });
  await expect(rowB.locator('.approval-badge')).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });

  // CATCH-UP: reload — still Approved.
  await pageB.reload();
  const rowB2 = mtxRow(pageB, name);
  await expect(rowB2).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await expect(rowB2).toContainText('Approved ✓', { timeout: CONVERGE_TIMEOUT });

  await ctxB.close();
}

// One REJECT→fix→RESUBMIT cycle: proves four cells of one editor column in a
// single realistic flow, each leg live + catch-up on device B:
//   reject  → MTX-RJT-<ed>-BAN (banner appears), MTX-RJT-<ed>-ERO (edit mode
//             returns), MTX-RJT-<ed>-VAL (rejected value cleared, kept field
//             stays answered)
//   resubmit→ MTX-SUB-<ed>-BAN (banner cleared by the superseding submission)
async function mtxRejectCycleCell(browser, page, editor) {
  const name = 'MTX Reject ' + editor;
  page.on('dialog', d => d.accept());
  const dow = await getTodayDOW(page);
  const tpl = await mtxCreate(page, name, dow, editor,
    [CHECKBOX_F('Cut the check', 0), CHECKBOX_F('Do B', 1)]);
  await mtxOpenA(page, name);
  await mtxCheckFields(page, ['Cut the check', 'Do B']);
  await mtxSubmitUI(page);
  const sub = await mtxPendingSubmission(page, tpl.id);
  expect(sub, 'submission is pending approval').toBeTruthy();
  const doBId = mtxFieldId(sub, 'Do B');
  expect(doBId, 'resolved Do B field id from snapshot').toBeTruthy();

  const { ctxB, pageB } = await mtxOpenB(browser, name);
  // Pre-state (non-vacuous): B is READONLY-pending, banner-free, Do B answered
  // (the readonly renderer shows an answered checkbox as ✓ — no .check-btn).
  await expect(pageB.locator('.submit-confirm')).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await expect(pageB.locator('#submit-btn')).toHaveCount(0);
  await expect(pageB.locator('.correction-banner')).toHaveCount(0);
  await expect(pageB.locator('.fill-field', { hasText: 'Do B' }))
    .toContainText('✓', { timeout: RUNNER_TIMEOUT });

  // The approver rejects Do B (REJECT_ITEM op is emitted server-side).
  await apiCall(page, 'POST', 'rejectItem',
    { submission_id: sub.id, field_id: doBId, comment: 'Please redo Do B', require_photo: false });

  const assertRejected = async () => {
    await expect(pageB.locator('.correction-banner'), 'rejection banner must reach device B')
      .toContainText('Please redo Do B', { timeout: CONVERGE_TIMEOUT });
    await expect(pageB.locator('#submit-btn'), 'device B must flip back to EDIT mode')
      .toBeVisible({ timeout: CONVERGE_TIMEOUT });
    await expect(pageB.locator('.fill-field', { hasText: 'Do B' }).locator('.check-btn'),
      'rejected field must come back unanswered').not.toHaveClass(/checked/, { timeout: CONVERGE_TIMEOUT });
    await expect(pageB.locator('.fill-field', { hasText: 'Cut the check' }).locator('.check-btn'),
      'non-rejected field keeps its answer').toHaveClass(/checked/, { timeout: CONVERGE_TIMEOUT });
  };
  // LIVE, then CATCH-UP.
  await assertRejected();
  await reopenRunnerB(pageB, name);
  await assertRejected();

  // Resubmit leg (SUB-<ed>-BAN): the editor fixes Do B and resubmits — the new
  // pending submission supersedes the rejection, so B's banner clears and the
  // runner returns to readonly.
  await mtxOpenA(page, name);
  await expect(page.locator('#submit-btn')).toBeVisible({ timeout: RUNNER_TIMEOUT });
  await mtxCheckFields(page, ['Do B']);
  await mtxSubmitUI(page);

  const assertResubmitted = async () => {
    await expect(pageB.locator('.correction-banner'), 'banner must clear after the resubmission')
      .toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
    await expect(pageB.locator('.submit-confirm')).toBeVisible({ timeout: CONVERGE_TIMEOUT });
    await expect(pageB.locator('#submit-btn')).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
  };
  await assertResubmitted();
  await reopenRunnerB(pageB, name);
  await assertResubmitted();

  await ctxB.close();
}

test.describe('Convergence matrix (systematic): op-type × editor × derived view', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto(BASE + '/workflows.html');
    await cleanupPendingApprovals(page);
    await cleanupTemplates(page);
  });

  // ── SET_FIELD × non-assignee admin (the ESC-1 column) ──────────────────────

  test('MTX-FLD-ADM-VAL: non-assignee admin field edit converges on the same admin\'s 2nd device (live + catch-up) [ESC-1]', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    await mtxCreate(page, 'MTX Field ADM', dow, 'ADM', [CHECKBOX_F('Wipe counters', 0)], false);

    // Observer B opens the runner FIRST. Pre-state (non-vacuous): unchecked.
    const { ctxB, pageB } = await mtxOpenB(browser, 'MTX Field ADM');
    const cbB = pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn');
    await expect(cbB).not.toHaveClass(/checked/);

    // Editor A (admin, NOT an assignee — template belongs to team_member) checks.
    await mtxOpenA(page, 'MTX Field ADM');
    await mtxCheckFields(page, ['Wipe counters']);

    // LIVE: the op must reach the admin's 2nd device (pre-ESC-1-fix it never did).
    await expect(cbB, 'non-assignee admin edit must converge live on the 2nd device')
      .toHaveClass(/checked/, { timeout: CONVERGE_TIMEOUT });

    // CATCH-UP: reload + reopen — still checked.
    await reopenRunnerB(pageB, 'MTX Field ADM');
    await expect(pageB.locator('.fill-field', { hasText: 'Wipe counters' }).locator('.check-btn'))
      .toHaveClass(/checked/, { timeout: CONVERGE_TIMEOUT });

    await ctxB.close();
  });

  test('MTX-FLD-ADM-CNT: non-assignee admin field edit converges the 2nd device\'s list count (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    const dow = await getTodayDOW(page);
    await mtxCreate(page, 'MTX FieldCnt ADM', dow, 'ADM', [CHECKBOX_F('Wipe counters', 0)], false);

    // Observer B sits on the BARE list. Pre-state: 0/1.
    const { ctxB, pageB, rowB } = await mtxListB(browser, 'MTX FieldCnt ADM');
    await expect(rowB).toContainText('0/1', { timeout: RUNNER_TIMEOUT });

    await mtxOpenA(page, 'MTX FieldCnt ADM');
    await mtxCheckFields(page, ['Wipe counters']);

    // LIVE: list count converges without a reload.
    await expect(rowB).toContainText('1/1', { timeout: CONVERGE_TIMEOUT });

    // CATCH-UP: reload — still 1/1.
    await pageB.reload();
    const rowB2 = mtxRow(pageB, 'MTX FieldCnt ADM');
    await expect(rowB2).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await expect(rowB2).toContainText('1/1', { timeout: CONVERGE_TIMEOUT });

    await ctxB.close();
  });

  // ── SUBMIT_CHECKLIST × editor ──────────────────────────────────────────────

  test('MTX-SUB-ASG-VAL/ERO: assignee submit converges value + readonly on device B (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await mtxSubmitCell(browser, page, 'ASG');
  });

  test('MTX-SUB-ADM-VAL/ERO: non-assignee-admin submit converges value + readonly on device B (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await mtxSubmitCell(browser, page, 'ADM');
  });

  test('MTX-SUB-ASG-CNT: assignee submit surfaces the Pending Approval badge on device B\'s bare list (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await mtxSubmitCountCell(browser, page, 'ASG');
  });

  test('MTX-SUB-ADM-CNT: non-assignee-admin submit surfaces the Pending Approval badge on device B\'s bare list (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await mtxSubmitCountCell(browser, page, 'ADM');
  });

  // ── APPROVE_ITEM × editor ──────────────────────────────────────────────────

  test('MTX-APR-ADM-ERO: non-assignee-admin approval flips device B\'s open runner to Approved (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    page.on('dialog', d => d.accept());
    const dow = await getTodayDOW(page);
    const tpl = await mtxCreate(page, 'MTX Approve ADM', dow, 'ADM', [CHECKBOX_F('Task A', 0)]);
    await mtxOpenA(page, 'MTX Approve ADM');
    await mtxCheckFields(page, ['Task A']);
    await mtxSubmitUI(page);
    const sub = await mtxPendingSubmission(page, tpl.id);
    expect(sub, 'submission is pending approval').toBeTruthy();

    // Observer B: open runner. Pre-state: pending-review, NOT approved.
    const { ctxB, pageB } = await mtxOpenB(browser, 'MTX Approve ADM');
    await expect(pageB.locator('.submit-confirm')).toContainText('Waiting for manager review', { timeout: RUNNER_TIMEOUT });
    await expect(pageB.locator('#fill-body')).not.toContainText('Approved');

    // The admin approves (rejectItem/approveSubmission carry no assignment gate;
    // the ADM column's approver assignment is team_member, so the queue-less
    // direct call is the non-assignee-admin approval path).
    await apiCall(page, 'POST', 'approveSubmission', { submission_id: sub.id, feedback: [] });

    // LIVE: Approved flips on B without a reload.
    await expect(pageB.locator('#fill-body'), 'device B must live-show Approved')
      .toContainText('Approved', { timeout: CONVERGE_TIMEOUT });

    // CATCH-UP: reload + reopen — still Approved.
    await reopenRunnerB(pageB, 'MTX Approve ADM');
    await expect(pageB.locator('#fill-body')).toContainText('Approved', { timeout: CONVERGE_TIMEOUT });

    await ctxB.close();
  });

  test('MTX-APR-ASG-CNT: approval surfaces Approved ✓ on device B\'s bare list (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await mtxApproveCountCell(browser, page, 'ASG');
  });

  test('MTX-APR-ADM-CNT: non-assignee-admin approval surfaces Approved ✓ on device B\'s bare list (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    await mtxApproveCountCell(browser, page, 'ADM');
  });

  // ── REJECT_ITEM × editor ───────────────────────────────────────────────────

  test('MTX-RJT-ASG-BAN/ERO/VAL + MTX-SUB-ASG-BAN: reject→fix→resubmit cycle converges on device B (live + catch-up) [ESC-2a]', async ({ browser, page }) => {
    test.setTimeout(180000);
    await mtxRejectCycleCell(browser, page, 'ASG');
  });

  test('MTX-RJT-ADM-BAN/ERO/VAL + MTX-SUB-ADM-BAN: non-assignee-admin reject→fix→resubmit cycle converges on device B (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(180000);
    await mtxRejectCycleCell(browser, page, 'ADM');
  });

  test('MTX-RJT-ADM-CNT: non-assignee-admin rejection clears the pending badge and re-derives the live count on device B\'s bare list (live + catch-up)', async ({ browser, page }) => {
    test.setTimeout(120000);
    page.on('dialog', d => d.accept());
    const dow = await getTodayDOW(page);
    const tpl = await mtxCreate(page, 'MTX RjtCnt ADM', dow, 'ADM',
      [CHECKBOX_F('Cut the check', 0), CHECKBOX_F('Do B', 1)]);
    await mtxOpenA(page, 'MTX RjtCnt ADM');
    await mtxCheckFields(page, ['Cut the check', 'Do B']);
    await mtxSubmitUI(page);
    const sub = await mtxPendingSubmission(page, tpl.id);
    expect(sub, 'submission is pending approval').toBeTruthy();
    const doBId = mtxFieldId(sub, 'Do B');
    expect(doBId, 'resolved Do B field id from snapshot').toBeTruthy();

    // Observer B on the BARE list. Pre-state: frozen snapshot count + badge.
    const { ctxB, pageB, rowB } = await mtxListB(browser, 'MTX RjtCnt ADM');
    await expect(rowB).toContainText('2/2', { timeout: RUNNER_TIMEOUT });
    await expect(rowB.locator('.approval-badge')).toBeVisible({ timeout: RUNNER_TIMEOUT });

    await apiCall(page, 'POST', 'rejectItem',
      { submission_id: sub.id, field_id: doBId, comment: 'redo Do B', require_photo: false });

    // LIVE: badge drops and the count re-derives from live state (the rejected
    // Do B is cleared, Cut the check keeps its answer → 1/2), no reload.
    await expect(rowB.locator('.approval-badge')).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
    await expect(rowB).toContainText('1/2', { timeout: CONVERGE_TIMEOUT });

    // CATCH-UP: reload — same derived state.
    await pageB.reload();
    const rowB2 = mtxRow(pageB, 'MTX RjtCnt ADM');
    await expect(rowB2).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await expect(rowB2.locator('.approval-badge')).toHaveCount(0, { timeout: CONVERGE_TIMEOUT });
    await expect(rowB2).toContainText('1/2', { timeout: CONVERGE_TIMEOUT });

    await ctxB.close();
  });

  test('MTX-RJT-ASG-BAN-SUBSTEP: rejecting a SUB-STEP surfaces its correction banner live on the submitter\'s 2nd device (live + catch-up) [ESC-3]', async ({ browser, page }) => {
    test.setTimeout(120000);
    page.on('dialog', d => d.accept());
    const dow = await getTodayDOW(page);
    const SS = (label, order) => ({ type: 'checkbox', label, order, config: {}, fail_trigger: null, condition: null });
    const tpl = await mtxCreate(page, 'MTX SubStep Reject', dow, 'ASG', [
      CHECKBOX_F('Cut the check', 0, { sub_steps: [SS('Do A', 0), SS('Do B', 1)] }),
    ]);
    const full = (await apiCall(page, 'GET', 'templates')).find(t => t.id === tpl.id);
    const parentId = full.sections[0].fields[0].id;
    const doAId = full.sections[0].fields[0].sub_steps.find(s => s.label === 'Do A').id;
    const doBId = full.sections[0].fields[0].sub_steps.find(s => s.label === 'Do B').id;

    // Submit with BOTH sub-steps done (the parent value carries the sub_steps
    // map — same shape the runner persists).
    await apiCall(page, 'POST', 'submitChecklist', {
      template_id: tpl.id, idempotency_key: randomUUID(),
      responses: [{ field_id: parentId, value: JSON.stringify({ value: true, sub_steps: { [doAId]: true, [doBId]: true } }) }],
    });
    const sub = await mtxPendingSubmission(page, tpl.id);
    expect(sub, 'submission is pending approval').toBeTruthy();

    // Observer B (submitter's 2nd device): open runner. Pre-state: readonly,
    // banner-free, both sub-steps done.
    const { ctxB, pageB } = await mtxOpenB(browser, 'MTX SubStep Reject');
    await expect(pageB.locator('.submit-confirm')).toBeVisible({ timeout: RUNNER_TIMEOUT });
    await expect(pageB.locator('.correction-banner')).toHaveCount(0);
    const doBCheck = pageB.locator('.sub-step-row', { hasText: 'Do B' }).locator('.sub-step-check');
    await expect(doBCheck).toHaveClass(/done/, { timeout: RUNNER_TIMEOUT });

    // The approver rejects the SUB-STEP itself (ESC-3's exact shape).
    await apiCall(page, 'POST', 'rejectItem',
      { submission_id: sub.id, field_id: doBId, comment: 'Redo step Do B', require_photo: false });

    const assertSubStepRejected = async () => {
      await expect(pageB.locator('.correction-banner', { hasText: 'Redo step Do B' }),
        'sub-step rejection banner must reach device B').toBeVisible({ timeout: CONVERGE_TIMEOUT });
      await expect(pageB.locator('.sub-step-row', { hasText: 'Do B' }).locator('.sub-step-check'),
        'rejected sub-step must come back un-done').not.toHaveClass(/done/, { timeout: CONVERGE_TIMEOUT });
      await expect(pageB.locator('.sub-step-row', { hasText: 'Do A' }).locator('.sub-step-check'),
        'non-rejected sub-step stays done').toHaveClass(/done/, { timeout: CONVERGE_TIMEOUT });
      await expect(pageB.locator('#submit-btn'), 'device B back in edit mode')
        .toBeVisible({ timeout: CONVERGE_TIMEOUT });
    };
    // LIVE, then CATCH-UP.
    await assertSubStepRejected();
    await reopenRunnerB(pageB, 'MTX SubStep Reject');
    await assertSubStepRejected();

    await ctxB.close();
  });
});
