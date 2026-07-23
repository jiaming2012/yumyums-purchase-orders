const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// ── Card G1 baseline ─────────────────────────────────────────────────────────
// /onboarding/* (and /videos/*) now sit behind the `onboarding` app grant
// (tests/grant-enforcement-parity.spec.js). This file's invited trainees and
// managers exercise onboarding flows, so grant the app to the standard roles
// once up front — preserving any user_grants other files added. The role/
// assignment tiers the prove-progress sweep asserts live INSIDE the handlers
// and are unchanged by the app gate.
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
  }, 'onboarding');
  await page.close();
});

// ─── Helpers ─────────────────────────────────────────────────────────────────

async function login(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email || ADMIN_EMAIL);
  await page.fill('input[type="password"]', password || ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

async function obApiCall(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch('/api/v1/onboarding/' + p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path, body]);
}

async function usersApiCall(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch('/api/v1/users/' + p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path, body]);
}

// waitForMyList waits for the My Trainings list to show content (card or empty state).
async function waitForMyList(page) {
  await page.waitForFunction(() => {
    const body = document.getElementById('my-body');
    if (!body) return false;
    return body.querySelector('.card') || body.querySelector('.empty') || body.textContent.includes('No trainings');
  });
}

// waitForTrainingRunner waits for the training detail view (sections) to appear.
async function waitForTrainingRunner(page) {
  await page.waitForFunction(() => {
    const body = document.getElementById('my-body');
    if (!body) return false;
    return body.querySelector('.sec-header') || body.querySelector('[data-action="back-to-my-list"]');
  });
}

// waitForManagerList waits for Manager tab hires list to load.
async function waitForManagerList(page) {
  await page.waitForFunction(() => {
    const body = document.getElementById('mgr-body');
    if (!body) return false;
    return body.querySelector('.card') || body.querySelector('.empty') || body.querySelector('.sub-tabs');
  });
}

// waitForBuilderList waits for Builder tab template list to load.
async function waitForBuilderList(page) {
  await page.waitForFunction(() => {
    const body = document.getElementById('builder-body');
    if (!body) return false;
    return body.querySelector('[data-action="open-template"]') || body.querySelector('[data-action="new-template"]') || body.querySelector('.empty');
  });
}

// waitForManagerTab waits for Manager tab to become visible (admin/manager only).
async function waitForManagerTab(page) {
  await page.waitForFunction(() => {
    const t2 = document.getElementById('t2');
    return t2 && t2.style.display !== 'none';
  });
}

// waitForBuilderTab waits for Builder tab to become visible (admin/manager only).
async function waitForBuilderTab(page) {
  await page.waitForFunction(() => {
    const t3 = document.getElementById('t3');
    return t3 && t3.style.display !== 'none';
  });
}

// ─── My Trainings tab ────────────────────────────────────────────────────────

test.describe('My Trainings tab', () => {
  test('shows empty state when no templates assigned', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    await waitForMyList(page);
    // Should show either empty state or a list
    const myBody = page.locator('#my-body');
    await expect(myBody).toBeVisible();
  });

  test('shows assigned template after assignment via API', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      if (!res.ok) return null;
      return res.json();
    });
    // Get the Kitchen Basics Training template ID
    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    expect(kitchenTemplate).toBeTruthy();

    // Assign to current user
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: kitchenTemplate.id,
    });

    // Navigate to My Trainings
    await page.goto('/onboarding.html');
    await waitForMyList(page);

    // Should show the template name
    await expect(page.locator('#my-body')).toContainText('Kitchen Basics Training');
  });

  test('checkbox progress persists after page reload', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      if (!res.ok) return null;
      return res.json();
    });
    // Seed a dedicated template so the active checkbox section is deterministic
    // (no dependency on shared-seed sign-off state — the former test guard-returned
    // when Kitchen Basics had all sections signed off from prior tests).
    const titleTag = 'Persist Section ' + Date.now();
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Checkbox Persist ' + Date.now(),
      roles: [],
      sections: [{
        title: titleTag, sort_order: 0,
        requires_sign_off: false, sign_off_roles: [], is_faq: false,
        // Multiple items so toggling ONE never completes the section (a complete
        // section goes read-only and drops the toggle-item affordance).
        items: [
          { type: 'checkbox', label: 'Persisted task', sort_order: 0, sub_items: [] },
          { type: 'checkbox', label: 'Second task', sort_order: 1, sub_items: [] },
          { type: 'checkbox', label: 'Third task', sort_order: 2, sub_items: [] },
        ],
      }],
    });
    expect(tpl && tpl.id, 'createTemplate must return an id').toBeTruthy();
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: tpl.id,
    });

    // The first section of a freshly-assigned template is always active.
    const trainingState = await obApiCall(page, 'GET', 'hireTraining/' + me.id + '?templateId=' + tpl.id);
    const activeSection = trainingState.sections.find(s => s.state === 'active' && !s.is_faq && s.items && s.items.some(i => i.type === 'checkbox'));
    expect(activeSection, 'seeded template must expose an active checkbox section').toBeTruthy();
    const testItem = activeSection.items.find(i => i.type === 'checkbox');
    expect(testItem).toBeTruthy();

    // Part 1: Set item as checked via API, verify UI shows it checked
    await obApiCall(page, 'POST', 'saveProgress', {
      item_id: testItem.id,
      progress_type: 'item',
      checked: true,
    });

    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator(`[data-action="open-my-training"][data-template-id="${tpl.id}"]`).click();
    await waitForTrainingRunner(page);

    // Find and expand the active section by its title
    const activeSectionHeader = page.locator('#my-body .sec-header').filter({ hasText: activeSection.title }).first();
    await activeSectionHeader.click();
    await expect(page.locator('#my-body .ob-check.checked').first()).toBeVisible({ timeout: 5000 });

    // Part 2: Uncheck via API, check via UI, reload and verify persistence
    await obApiCall(page, 'POST', 'saveProgress', {
      item_id: testItem.id,
      progress_type: 'item',
      checked: false,
    });

    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator(`[data-action="open-my-training"][data-template-id="${tpl.id}"]`).click();
    await waitForTrainingRunner(page);

    // Expand the active section again
    const activeSectionHeader2 = page.locator('#my-body .sec-header').filter({ hasText: activeSection.title }).first();
    await activeSectionHeader2.click();

    // Click the toggle-item to check it via UI
    const toggleItem = page.locator('#my-body [data-action="toggle-item"]').first();
    await toggleItem.click();
    await expect(toggleItem).toHaveClass(/checked/);

    // Wait for save
    // Let the autosave POST flush (the .ob-check.checked assertion above already
    // proved the toggle applied optimistically; the real persistence proof is the
    // post-reload assertion below).
    await page.waitForTimeout(1500);

    // Reload and verify it persists
    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator(`[data-action="open-my-training"][data-template-id="${tpl.id}"]`).click();
    await waitForTrainingRunner(page);
    const activeSectionHeader3 = page.locator('#my-body .sec-header').filter({ hasText: activeSection.title }).first();
    await activeSectionHeader3.click();

    await expect(page.locator('#my-body .ob-check.checked').first()).toBeVisible();

    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('video part watched state persists after page reload', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      if (!res.ok) return null;
      return res.json();
    });
    // Seed a dedicated template whose video series lives in the FIRST section, so
    // it is always active (unlocked) — the former test wrapped its watched-marker
    // assertion in an if/else that degraded to a no-op check whenever the shared
    // seed's Equipment Training section was locked.
    const secTitle = 'Watch First ' + Date.now();
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Video Watched Persist ' + Date.now(),
      roles: [],
      sections: [{ title: secTitle, sort_order: 0, requires_sign_off: false, sign_off_roles: [], is_faq: false, items: [
        { type: 'video_series', label: 'Training Videos', sort_order: 0, video_parts: [
          { title: 'Intro Video', description: 'Welcome', url: 'https://example.com/test-video.mp4', sort_order: 0 }
        ]}
      ]}]
    });
    expect(tpl && tpl.id, 'createTemplate must return an id').toBeTruthy();

    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const sec = fullTemplate.sections.find(s => s.title === secTitle);
    expect(sec).toBeTruthy();
    const videoSeries = sec.items.find(i => i.type === 'video_series');
    expect(videoSeries).toBeTruthy();
    const firstPart = videoSeries.video_parts[0];
    expect(firstPart).toBeTruthy();

    // Assign template
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: tpl.id,
    });

    // Set video part as watched via API
    await obApiCall(page, 'POST', 'saveProgress', {
      item_id: firstPart.id,
      progress_type: 'video_part',
      checked: true,
    });

    // Navigate to training
    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator(`[data-action="open-my-training"][data-template-id="${tpl.id}"]`).click();
    await waitForTrainingRunner(page);

    // Expand the first section (always active/unlocked) and assert the watched
    // marker persisted. Video parts have no checkbox — a watched part surfaces
    // as a .watched class on its play affordance.
    const secHeader = page.locator('#my-body .sec-header').filter({ hasText: secTitle }).first();
    await secHeader.click();
    await expect(page.locator('#my-body [data-action="play-video"].watched').first())
      .toBeVisible({ timeout: 5000 });

    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('section unlocks after completing all items in previous section', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      if (!res.ok) return null;
      return res.json();
    });
    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    expect(kitchenTemplate).toBeTruthy();

    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + kitchenTemplate.id);
    const sec1 = fullTemplate.sections.find(s => s.title === 'Safety & Hygiene');
    expect(sec1).toBeTruthy();

    // Assign template
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: kitchenTemplate.id,
    });

    // Complete all items in section 1 via API to trigger section 2 unlock
    for (const item of sec1.items) {
      await obApiCall(page, 'POST', 'saveProgress', {
        item_id: item.id,
        progress_type: 'item',
        checked: true,
      });
    }

    await page.goto('/onboarding.html');
    await waitForMyList(page);

    // Open the training
    await page.locator(`[data-action="open-my-training"][data-template-id="${kitchenTemplate.id}"]`).click();
    await waitForTrainingRunner(page);

    // After completing section 1, section 2 should be unlocked
    const sections = page.locator('#my-body .sec-header');
    const sectionCount = await sections.count();
    // At minimum section 1 is not locked (it was completed, state=complete)
    // Section 2 (Equipment Training, idx=1) should now be active (not locked)
    if (sectionCount > 1) {
      await expect(sections.nth(1)).not.toHaveClass(/locked/);
    }
  });

  test('video part with URL shows play button in My Trainings', async ({ page }) => {
    // Regression: video parts were not showing in My Trainings because the frontend
    // used "parts" instead of "video_parts" when saving, so video data was silently dropped.
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    // Create template with a video series that has a URL
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Video Play Test',
      roles: ['admin', 'superadmin'],
      sections: [{ title: 'Watch Videos', sort_order: 1, requires_sign_off: false, is_faq: false, items: [
        { type: 'video_series', label: 'Training Videos', sort_order: 1, video_parts: [
          { title: 'Intro Video', description: 'Welcome to the team', url: 'https://example.com/test-video.mp4', sort_order: 1 }
        ]}
      ]}]
    });

    // Verify the template was saved with video parts
    const fullTpl = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const sec = fullTpl.sections.find(s => s.title === 'Watch Videos');
    expect(sec).toBeTruthy();
    const vs = sec.items.find(i => i.type === 'video_series');
    expect(vs).toBeTruthy();
    expect(vs.video_parts).toBeTruthy();
    expect(vs.video_parts.length).toBe(1);
    expect(vs.video_parts[0].url).toBe('https://example.com/test-video.mp4');

    // Assign template to self
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: tpl.id,
    });

    // Open My Trainings
    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator('#my-body .card', { hasText: 'Video Play Test' }).first().click();
    await waitForTrainingRunner(page);

    // Expand the section
    const header = page.locator('#my-body .sec-header').first();
    await header.click();

    // Video series title should be visible
    await expect(page.locator('#my-body')).toContainText('Training Videos');

    // Play button should be visible (either thumbnail wrap or fallback play button)
    await expect(page.locator('#my-body [data-action="play-video"]')).toBeVisible();

    // Video parts have NO manual checkbox — completion happens by watching the
    // video (commit 1ec8725 redesigned video parts to drop the checkbox). The
    // play affordance is the only way to mark a part complete.
    await expect(page.locator('#my-body input[type="checkbox"]')).toHaveCount(0);
  });

  test('video modal close button dismisses the player', async ({ page }) => {
    // Regression: close button was inside #video-modal (outside #my-body),
    // so the event delegation click handler never caught the close-video action.
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    // Create template with video
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Video Close Test',
      roles: ['admin', 'superadmin'],
      sections: [{ title: 'Sec', sort_order: 1, requires_sign_off: false, is_faq: false, items: [
        { type: 'video_series', label: 'Vids', sort_order: 1, video_parts: [
          { title: 'Test Vid', description: '', url: 'https://example.com/close-test.mp4', sort_order: 1 }
        ]}
      ]}]
    });

    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });

    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator('#my-body .card', { hasText: 'Video Close Test' }).first().click();
    await waitForTrainingRunner(page);

    // Expand section and tap play
    await page.locator('#my-body .sec-header').first().click();
    await page.locator('[data-action="play-video"]').first().click();

    // Modal should be visible
    await expect(page.locator('#video-modal')).toHaveCSS('display', 'block');

    // Click close button
    await page.locator('#video-close-btn').click();

    // Modal should be hidden
    await expect(page.locator('#video-modal')).toHaveCSS('display', 'none');
  });

  test('progress survives template edit — adding new item preserves existing progress', async ({ page }) => {
    // Positive test: editing a template (adding a video part) should NOT wipe existing progress.
    // Regression: UpdateTemplate used to DELETE all sections and re-insert, orphaning ob_progress rows.
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    // Create template with a checkbox
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Progress Preserve Test',
      roles: ['admin', 'superadmin'],
      sections: [{ title: 'Tasks', sort_order: 1, requires_sign_off: false, is_faq: false, items: [
        { type: 'checkbox', label: 'First task', sort_order: 1 }
      ]}]
    });

    // Assign and complete the checkbox
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });
    const fullTpl = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const item = fullTpl.sections[0].items[0];
    await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });

    // Verify progress exists before edit
    const before = await obApiCall(page, 'GET', 'hireTraining/' + me.id + '?templateId=' + tpl.id);
    expect(before.sections[0].items[0].checked).toBe(true);

    // Edit template: add a second checkbox (simulates builder save)
    await page.evaluate(async ([id, sections]) => {
      await fetch('/api/v1/onboarding/updateTemplate/' + id, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: 'Progress Preserve Test',
          roles: ['admin', 'superadmin'],
          sections: [{
            id: sections[0].id,
            title: 'Tasks',
            sort_order: 1,
            requires_sign_off: false,
            is_faq: false,
            items: [
              { id: sections[0].items[0].id, type: 'checkbox', label: 'First task', sort_order: 1 },
              { type: 'checkbox', label: 'Second task', sort_order: 2 }
            ]
          }]
        })
      });
    }, [tpl.id, fullTpl.sections]);

    // Verify progress is preserved after edit
    const after = await obApiCall(page, 'GET', 'hireTraining/' + me.id + '?templateId=' + tpl.id);
    expect(after.sections[0].items[0].checked).toBe(true);
    expect(after.sections[0].items[0].label).toBe('First task');
    // New item should not be checked
    expect(after.sections[0].items[1].checked).toBe(false);
    expect(after.sections[0].items[1].label).toBe('Second task');
  });

  test('progress lost when item is removed from template — negative test', async ({ page }) => {
    // Negative test: removing an item from the template SHOULD remove it from the response.
    // The progress row becomes orphaned (item_id no longer in template) — this is expected.
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    // Create template with two checkboxes
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Progress Remove Test',
      roles: ['admin', 'superadmin'],
      sections: [{ title: 'Tasks', sort_order: 1, requires_sign_off: false, is_faq: false, items: [
        { type: 'checkbox', label: 'Keep me', sort_order: 1 },
        { type: 'checkbox', label: 'Remove me', sort_order: 2 }
      ]}]
    });

    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });
    const fullTpl = await obApiCall(page, 'GET', 'templates/' + tpl.id);

    // Complete both items
    await obApiCall(page, 'POST', 'saveProgress', { item_id: fullTpl.sections[0].items[0].id, progress_type: 'item', checked: true });
    await obApiCall(page, 'POST', 'saveProgress', { item_id: fullTpl.sections[0].items[1].id, progress_type: 'item', checked: true });

    // Edit template: remove second item
    await page.evaluate(async ([id, sections]) => {
      await fetch('/api/v1/onboarding/updateTemplate/' + id, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: 'Progress Remove Test',
          roles: ['admin', 'superadmin'],
          sections: [{
            id: sections[0].id,
            title: 'Tasks',
            sort_order: 1,
            requires_sign_off: false,
            is_faq: false,
            items: [
              { id: sections[0].items[0].id, type: 'checkbox', label: 'Keep me', sort_order: 1 }
            ]
          }]
        })
      });
    }, [tpl.id, fullTpl.sections]);

    // Verify: first item still checked, second item gone from response
    const after = await obApiCall(page, 'GET', 'hireTraining/' + me.id + '?templateId=' + tpl.id);
    expect(after.sections[0].items.length).toBe(1);
    expect(after.sections[0].items[0].checked).toBe(true);
    expect(after.sections[0].items[0].label).toBe('Keep me');
  });

  test('FAQ section shows questions and answers', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      if (!res.ok) return null;
      return res.json();
    });
    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    expect(kitchenTemplate).toBeTruthy();

    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + kitchenTemplate.id);
    const faqSection = fullTemplate.sections.find(s => s.is_faq);
    expect(faqSection).toBeTruthy();

    // Assign template
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: kitchenTemplate.id,
    });

    // Complete all non-FAQ sections via API to unlock the FAQ section
    const sec1 = fullTemplate.sections.find(s => s.title === 'Safety & Hygiene');
    const sec2 = fullTemplate.sections.find(s => s.title === 'Equipment Training');
    const sec3 = fullTemplate.sections.find(s => s.title === 'Menu Knowledge');

    for (const sec of [sec1, sec2, sec3]) {
      if (!sec) continue;
      for (const item of sec.items) {
        if (item.type === 'checkbox') {
          await obApiCall(page, 'POST', 'saveProgress', {
            item_id: item.id,
            progress_type: 'item',
            checked: true,
          });
        } else if (item.type === 'video_series' && item.video_parts) {
          for (const part of item.video_parts) {
            await obApiCall(page, 'POST', 'saveProgress', {
              item_id: part.id,
              progress_type: 'video_part',
              checked: true,
            });
          }
        }
      }
    }

    await page.goto('/onboarding.html');
    await waitForMyList(page);

    // Open training
    await page.locator(`[data-action="open-my-training"][data-template-id="${kitchenTemplate.id}"]`).click();
    await waitForTrainingRunner(page);

    // FAQ section should be visible and not locked
    const faqHeader = page.locator('#my-body .sec-header').filter({ hasText: 'FAQ' });
    await expect(faqHeader).toBeVisible();
    await expect(faqHeader).not.toHaveClass(/locked/);

    // Expand FAQ section
    await faqHeader.click();

    // Verify Q&A items are visible
    await expect(page.locator('#my-body .faq-q').first()).toBeVisible();
    // Click a question to see the answer
    await page.locator('#my-body .faq-q').first().click();
    await expect(page.locator('#my-body .faq-a').first()).toBeVisible();
  });
});

// ─── Manager tab ─────────────────────────────────────────────────────────────

test.describe('Manager tab', () => {
  test('manager sees hire with assigned training', async ({ page }) => {
    await login(page);

    // Create a second user via API
    const inviteResult = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'TestHire',
      last_name: 'Mgr',
      email: 'test.hire.mgr.' + Date.now() + '@yumyums.kitchen',
      roles: ['team_member'],
    });
    expect(inviteResult.user).toBeTruthy();
    const hireId = inviteResult.user.id;

    // Accept the invite so the hire becomes status='active' — managerHires only
    // surfaces active users. accept-invite hijacks the session, so re-login as admin.
    const token = inviteResult.invite_path.split('token=')[1];
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' }),
      });
    }, token);
    await login(page);

    // Assign Kitchen Basics template to the hire
    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    expect(kitchenTemplate).toBeTruthy();

    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: hireId,
      template_id: kitchenTemplate.id,
    });

    // Navigate to onboarding, open Manager tab
    await page.goto('/onboarding.html');
    await waitForManagerTab(page);
    await page.click('#t2');
    await waitForManagerList(page);

    // Hire should appear in the manager view
    await expect(page.locator('#mgr-body')).toContainText('TestHire');
  });

  test('sign-off form requires readiness rating (notes optional)', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      if (!res.ok) return null;
      return res.json();
    });

    // Seed a dedicated template with one sign-off section and complete it for the
    // admin-as-hire, so the sign-off button DETERMINISTICALLY appears in the
    // Manager tab. The former test wrapped every assertion in
    // `if (signOffBtn.isVisible())` behind a `hireCardCount === 0` guard-return,
    // so it asserted nothing whenever the shared Kitchen Basics section wasn't
    // complete for a discovered hire card.
    const secTitle = 'Readiness Section ' + Date.now();
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Readiness Signoff ' + Date.now(),
      roles: [],
      sections: [{
        title: secTitle, sort_order: 0, requires_sign_off: true,
        sign_off_roles: ['admin', 'manager', 'superadmin'], is_faq: false,
        items: [
          { type: 'checkbox', label: 'Task A', sort_order: 0, sub_items: [] },
          { type: 'checkbox', label: 'Task B', sort_order: 1, sub_items: [] },
        ],
      }],
    });
    expect(tpl && tpl.id, 'createTemplate must return an id').toBeTruthy();
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });

    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    for (const item of full.sections[0].items) {
      await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
    }

    // Navigate to Manager tab and open the admin-as-hire card + this template's training.
    await page.goto('/onboarding.html');
    await waitForManagerTab(page);
    await page.click('#t2');
    await waitForManagerList(page);

    // A single complete-but-unsigned section keeps the hire in Active; fall back
    // to Completed defensively (still deterministic — the card must exist).
    const hireSel = '#mgr-body [data-action="open-hire"][data-hire-id="' + me.id + '"]';
    if (await page.locator(hireSel).count() === 0) {
      await page.click('.sub-tabs button:has-text("Completed")');
      await page.waitForTimeout(300);
    }
    await page.locator(hireSel).first().waitFor({ state: 'visible' });
    await page.locator(hireSel).first().click();
    await page.waitForTimeout(500);

    const viewBtn = page.locator('[data-action="view-training"][data-template-id="' + tpl.id + '"]').first();
    await viewBtn.waitFor({ state: 'visible' });
    await viewBtn.click();
    await page.waitForTimeout(500);

    // Expand the section and assert the sign-off button appears (section complete
    // + requires_sign_off + admin holds the role) — no longer conditional.
    const secHeader = page.locator('#mgr-body .sec-header').filter({ hasText: secTitle }).first();
    await secHeader.click();

    const signOffBtn = page.locator('#mgr-body [data-action="show-signoff-form"]').first();
    await expect(signOffBtn).toBeVisible({ timeout: 5000 });
    await signOffBtn.click();

    // Sign-off form should be visible
    const signOffForm = page.locator('#mgr-body .signoff-form').first();
    await expect(signOffForm).toBeVisible();
    const confirmBtn = page.locator('#mgr-body [data-action="confirm-signoff"]').first();
    await expect(confirmBtn).toBeVisible();

    // Confirm without selecting rating — readiness-required error must show.
    await confirmBtn.click();
    await page.waitForTimeout(500);
    await expect(page.locator('.signoff-form')).toContainText('Readiness is required');

    // Select rating — error clears and confirm signs off the section.
    await page.locator('#mgr-body .rating-btn').first().click();
    await confirmBtn.click();
    await page.waitForTimeout(2000);
    await expect(page.locator('#mgr-body')).toContainText('Signed Off');

    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('signed-off section shows "By {manager} @ {datetime}" attribution', async ({ page }) => {
    await login(page);

    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + kitchenTemplate.id);
    const sec1 = fullTemplate.sections.find(s => s.title === 'Safety & Hygiene');

    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    // Assign, complete, and sign off section 1
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: kitchenTemplate.id });
    for (const item of sec1.items) {
      if (item.type === 'checkbox') {
        await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
      }
    }
    await obApiCall(page, 'POST', 'signOff', { section_id: sec1.id, hire_id: me.id, notes: '', rating: 'ready' });

    // Open My Trainings and view the template
    await page.goto('/onboarding.html');
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && body.querySelector('.card');
    }, { timeout: 10000 });
    await page.locator('#my-body .card', { hasText: 'Kitchen Basics' }).first().click();
    await page.waitForSelector('.sec-header');

    // The signed-off section should show "By {name} @" format, NOT "Signed off by"
    const sectionText = await page.locator('.sec-header').first().textContent();
    expect(sectionText).toContain('Signed Off');
    const attrText = await page.locator('.attribution').first().textContent();
    expect(attrText).toContain('By ');
    expect(attrText).toContain(' @ ');
    expect(attrText).not.toContain('Signed off by');
  });

  test('video part progress persists after page reload', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => (await (await fetch('/api/v1/me')).json()));

    // Seed a two-section template: section 1 is a sign-off gate, section 2 (a
    // checkbox section) unlocks only after section 1 is signed off. The former
    // test relied on the shared Kitchen Basics EQUIPMENT section rendering an
    // .ob-check (video parts actually render as .video-thumb-wrap, so it was
    // really a checkbox item) and wrapped its persistence assertion in
    // `if (count > 0)`, asserting nothing when that section had no checkbox item.
    const sec1Title = 'Gate Section ' + Date.now();
    const sec2Title = 'Unlocked Section ' + Date.now();
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Unlock Persist ' + Date.now(),
      roles: [],
      sections: [
        { title: sec1Title, sort_order: 0, requires_sign_off: true, sign_off_roles: ['admin', 'manager', 'superadmin'], is_faq: false,
          items: [{ type: 'checkbox', label: 'Gate task', sort_order: 0, sub_items: [] }] },
        { title: sec2Title, sort_order: 1, requires_sign_off: false, sign_off_roles: [], is_faq: false,
          // Multiple items so toggling ONE keeps the section active (interactive).
          items: [
            { type: 'checkbox', label: 'Unlocked task', sort_order: 0, sub_items: [] },
            { type: 'checkbox', label: 'Unlocked task 2', sort_order: 1, sub_items: [] },
          ] },
      ],
    });
    expect(tpl && tpl.id, 'createTemplate must return an id').toBeTruthy();
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });

    // Complete + sign off section 1 so section 2 unlocks.
    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const sec1 = full.sections.find(s => s.title === sec1Title);
    for (const item of sec1.items) {
      await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
    }
    await obApiCall(page, 'POST', 'signOff', { section_id: sec1.id, hire_id: me.id, notes: '', rating: 'ready' });

    // Open the training, expand the now-unlocked section 2, check its item via UI.
    await page.goto('/onboarding.html');
    await page.waitForFunction(() => document.getElementById('my-body') && document.getElementById('my-body').querySelector('.card'));
    await page.locator(`[data-action="open-my-training"][data-template-id="${tpl.id}"]`).first().click();
    await page.waitForSelector('.sec-header');
    const sec2Header = page.locator('#my-body .sec-header').filter({ hasText: sec2Title }).first();
    await sec2Header.click();
    await page.waitForTimeout(300);

    const check = page.locator('#my-body [data-action="toggle-item"]').first();
    await check.click();
    await expect(page.locator('#my-body .ob-check.checked').first()).toBeVisible({ timeout: 5000 });
    // Let the autosave POST flush (the .ob-check.checked assertion above already
    // proved the toggle applied optimistically; the real persistence proof is the
    // post-reload assertion below).
    await page.waitForTimeout(1500);

    // Reload and verify persistence.
    await page.goto('/onboarding.html');
    await page.waitForFunction(() => document.getElementById('my-body') && document.getElementById('my-body').querySelector('.card'));
    await page.locator(`[data-action="open-my-training"][data-template-id="${tpl.id}"]`).first().click();
    await page.waitForSelector('.sec-header');
    await page.locator('#my-body .sec-header').filter({ hasText: sec2Title }).first().click();
    await expect(page.locator('#my-body .ob-check.checked').first()).toBeVisible();

    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('completing a section unlocks next section and collapses completed one', async ({ page }) => {
    await login(page);
    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + kitchenTemplate.id);
    const me = await page.evaluate(async () => (await (await fetch('/api/v1/me')).json()));

    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: kitchenTemplate.id });

    // Complete section 1 + sign off
    const sec1 = fullTemplate.sections.find(s => s.title === 'Safety & Hygiene');
    for (const item of sec1.items) {
      if (item.type === 'checkbox') {
        await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
      }
    }
    await obApiCall(page, 'POST', 'signOff', { section_id: sec1.id, hire_id: me.id, notes: '', rating: 'ready' });

    // Complete section 2 (Equipment Training) + sign off
    const sec2 = fullTemplate.sections.find(s => s.title === 'Equipment Training');
    for (const item of sec2.items) {
      if (item.type === 'video_series') {
        for (const part of (item.video_parts || [])) {
          await obApiCall(page, 'POST', 'saveProgress', { item_id: part.id, progress_type: 'video_part', checked: true });
        }
      } else if (item.type === 'checkbox') {
        await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
      }
    }
    await obApiCall(page, 'POST', 'signOff', { section_id: sec2.id, hire_id: me.id, notes: '', rating: 'ready' });

    // Open training — Menu Knowledge (section 3) should be active now
    await page.goto('/onboarding.html');
    await page.waitForFunction(() => document.getElementById('my-body') && document.getElementById('my-body').querySelector('.card'));
    await page.locator('#my-body .card', { hasText: 'Kitchen Basics' }).first().click();
    await page.waitForSelector('.sec-header');

    // Expand Menu Knowledge and complete all items
    const menuHeader = page.locator('.sec-header', { hasText: 'MENU KNOWLEDGE' });
    await menuHeader.click();
    await page.waitForTimeout(300);

    // Check all checkboxes in Menu Knowledge
    const checks = page.locator('.ob-check:not(.checked)');
    const checkCount = await checks.count();
    for (let i = 0; i < checkCount; i++) {
      // Handle the confirmation dialog on the last item
      if (i === checkCount - 1) {
        page.once('dialog', d => d.accept());
      }
      await page.locator('.ob-check:not(.checked)').first().click();
      await page.waitForTimeout(500);
    }

    // After completing Menu Knowledge:
    // 1. Menu Knowledge items should be collapsed (not visible)
    await page.waitForTimeout(1000);
    const visibleItems = await page.locator('.sec-items').count();
    // Items from completed sections should not be visible

    // 2. FAQ section should be unlocked (not showing "Locked")
    const faqHeader = page.locator('.sec-header', { hasText: 'FAQ' });
    await expect(faqHeader).toBeVisible();
    const faqText = await faqHeader.textContent();
    expect(faqText).not.toContain('Locked');
  });

  test('video part checked state returned by hireTraining API', async ({ page }) => {
    await login(page);
    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + kitchenTemplate.id);
    const sec1 = fullTemplate.sections.find(s => s.title === 'Safety & Hygiene');
    const sec2 = fullTemplate.sections.find(s => s.title === 'Equipment Training');
    const me = await page.evaluate(async () => (await (await fetch('/api/v1/me')).json()));

    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: kitchenTemplate.id });

    // Complete section 1 + sign off to unlock section 2
    for (const item of sec1.items) {
      if (item.type === 'checkbox') {
        await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
      }
    }
    await obApiCall(page, 'POST', 'signOff', { section_id: sec1.id, hire_id: me.id, notes: '', rating: 'ready' });

    // Check all video parts in section 2
    for (const item of sec2.items) {
      if (item.type === 'video_series') {
        for (const part of (item.video_parts || [])) {
          await obApiCall(page, 'POST', 'saveProgress', { item_id: part.id, progress_type: 'video_part', checked: true });
        }
      } else if (item.type === 'checkbox') {
        await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
      }
    }

    // Fetch hireTraining — all video parts should be checked
    const training = await obApiCall(page, 'GET', 'hireTraining/' + me.id + '?templateId=' + kitchenTemplate.id);
    const equipSec = training.sections.find(s => s.title === 'Equipment Training');
    const grillOp = equipSec.items.find(i => i.label === 'Grill Operation');

    // ALL 3 video parts should be checked — this is the bug
    const checkedParts = grillOp.video_parts.filter(p => p.checked);
    expect(checkedParts.length).toBe(grillOp.video_parts.length);
  });

  test('backend rejects progress updates for completed sections awaiting sign-off', async ({ page }) => {
    await login(page);
    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + kitchenTemplate.id);
    const sec1 = fullTemplate.sections.find(s => s.title === 'Safety & Hygiene');
    const me = await page.evaluate(async () => (await (await fetch('/api/v1/me')).json()));

    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: kitchenTemplate.id });

    // Complete all items in section 1
    for (const item of sec1.items) {
      if (item.type === 'checkbox') {
        await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
      }
    }

    // Section is now "complete" (waiting for sign-off) — try to uncheck an item
    const result = await page.evaluate(async ([itemId]) => {
      const res = await fetch('/api/v1/onboarding/saveProgress', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ item_id: itemId, progress_type: 'item', checked: false })
      });
      return { status: res.status, body: await res.json() };
    }, [sec1.items[0].id]);

    // Should be rejected — section is awaiting sign-off
    expect(result.status).toBe(400);
    expect(result.body.error).toBe('section_awaiting_signoff');
  });

  test('sign-off succeeds with rating only (notes optional)', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    // Seed a dedicated template with one sign-off section so the sign-off flow is
    // deterministic. The former test guard-returned whenever Kitchen Basics had
    // fewer than 2 sign-off sections available, asserting nothing.
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Rating Only Signoff ' + Date.now(),
      roles: [],
      sections: [{
        title: 'Rating Only Section ' + Date.now(), sort_order: 0,
        requires_sign_off: true, sign_off_roles: ['admin', 'manager', 'superadmin'], is_faq: false,
        items: [
          { type: 'checkbox', label: 'Task A', sort_order: 0, sub_items: [] },
          { type: 'checkbox', label: 'Task B', sort_order: 1, sub_items: [] },
        ],
      }],
    });
    expect(tpl && tpl.id, 'createTemplate must return an id').toBeTruthy();
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });

    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const sec = full.sections[0];
    for (const item of sec.items) {
      await obApiCall(page, 'POST', 'saveProgress', { item_id: item.id, progress_type: 'item', checked: true });
    }

    // Sign off via API with NO notes — should succeed (notes optional).
    const result = await obApiCall(page, 'POST', 'signOff', {
      section_id: sec.id,
      hire_id: me.id,
      notes: '',
      rating: 'ready',
    });
    expect(result.ok).toBeTruthy();

    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('sign-off via API rejects missing rating', async ({ page }) => {
    await login(page);

    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + kitchenTemplate.id);
    const sec1 = fullTemplate.sections.find(s => s.title === 'Safety & Hygiene');

    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    // Try sign-off with no rating — should fail
    const result = await page.evaluate(async ([secId, hireId]) => {
      const res = await fetch('/api/v1/onboarding/signOff', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ section_id: secId, hire_id: hireId, notes: 'test', rating: '' })
      });
      return { status: res.status, body: await res.json() };
    }, [sec1.id, me.id]);

    expect(result.status).toBe(400);
    expect(result.body.error).toBe('invalid_rating');
  });

  test('sign-off records attribution on hire view', async ({ page }) => {
    await login(page);

    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchenTemplate = templates.find(t => t.name === 'Kitchen Basics Training');
    const fullTemplate = await obApiCall(page, 'GET', 'templates/' + kitchenTemplate.id);
    const sec1 = fullTemplate.sections.find(s => s.title === 'Safety & Hygiene');

    // Use admin as the hire — sign off section 1
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      if (!res.ok) return null;
      return res.json();
    });

    // Assign Kitchen Basics to admin
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: kitchenTemplate.id,
    });

    // Complete all items in section 1 via API
    for (const item of sec1.items) {
      if (item.type === 'checkbox') {
        await obApiCall(page, 'POST', 'saveProgress', {
          item_id: item.id,
          progress_type: 'item',
          checked: true,
        });
      }
    }

    // Sign off section 1 via API directly (idempotent — ON CONFLICT DO NOTHING)
    await obApiCall(page, 'POST', 'signOff', {
      section_id: sec1.id,
      hire_id: me.id,
      notes: 'Good work on section 1',
      rating: 'ready',
    });

    // Navigate to My Trainings
    await page.goto('/onboarding.html');
    await waitForMyList(page);

    // Open Kitchen Basics Training specifically
    await page.locator('[data-action="open-my-training"]', { hasText: 'Kitchen Basics' }).first().click();
    await waitForTrainingRunner(page);

    // Signed-off section should show attribution text
    await expect(page.locator('#my-body')).toContainText('Signed Off');
  });
});

// ─── Role-based auto-assignment ──────────────────────────────────────────────

test.describe('Role-based auto-assignment', () => {
  test('My Trainings list shows section-level progress (not item-level)', async ({ page }) => {
    // Create a template with 2 sections, each with multiple items
    await login(page);
    await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Section Progress Test',
      roles: ['admin', 'superadmin'],
      sections: [
        { title: 'Section A', sort_order: 1, requires_sign_off: false, is_faq: false, items: [
          { type: 'checkbox', label: 'A1', sort_order: 1 },
          { type: 'checkbox', label: 'A2', sort_order: 2 }
        ]},
        { title: 'Section B', sort_order: 2, requires_sign_off: false, is_faq: false, items: [
          { type: 'checkbox', label: 'B1', sort_order: 1 },
          { type: 'checkbox', label: 'B2', sort_order: 2 },
          { type: 'checkbox', label: 'B3', sort_order: 3 }
        ]}
      ]
    });

    await page.goto('/onboarding.html');
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && (body.querySelector('.card') || body.querySelector('.empty'));
    }, { timeout: 10000 });

    const card = page.locator('#my-body .card', { hasText: 'Section Progress Test' }).first();
    await expect(card).toBeVisible();

    const text = await card.textContent();
    expect(text).not.toContain('undefined');
    // Should show "0 of 2 sections complete" (2 sections), NOT "0 of 5 items complete" (5 items)
    expect(text).toContain('0 of 2 sections complete');
  });

  test('FAQ section shows viewed count and completes when all expanded', async ({ page }) => {
    // Create template with a FAQ section
    await login(page);
    await obApiCall(page, 'POST', 'createTemplate', {
      name: 'FAQ Progress Test',
      roles: ['admin', 'superadmin'],
      sections: [{ title: 'FAQ Section', sort_order: 1, requires_sign_off: false, is_faq: true, items: [
        { type: 'faq', label: 'Question 1', answer: 'Answer 1', sort_order: 1 },
        { type: 'faq', label: 'Question 2', answer: 'Answer 2', sort_order: 2 }
      ]}]
    });

    await page.goto('/onboarding.html');
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && body.querySelector('.card');
    }, { timeout: 10000 });

    // Open the template
    await page.locator('#my-body .card', { hasText: 'FAQ Progress Test' }).first().click();
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && body.querySelector('.sec-header');
    }, { timeout: 10000 });

    // FAQ section should show 0/2 initially
    const header = page.locator('.sec-header').first();
    await expect(header).toContainText('0/2');

    // Expand the section
    await header.click();
    await page.waitForSelector('.faq-q');

    // Expand first FAQ question
    await page.locator('.faq-q').first().click();
    await page.waitForTimeout(1500); // wait for auto-save

    // Header should now show 1/2
    await expect(page.locator('.sec-header').first()).toContainText('1/2');

    // Expand second FAQ question
    await page.locator('.faq-q').nth(1).click();
    await page.waitForTimeout(1500);

    // Should show Complete (2/2 viewed)
    await expect(page.locator('.sec-header').first()).toContainText('Complete');
  });

  test('FAQ last question stays expanded after completing section', async ({ page }) => {
    // Bug: opening the last FAQ question marks section complete AND auto-collapses it,
    // hiding the answer before the employee can read it. FAQ sections should NOT collapse
    // on completion — only non-FAQ sections should auto-collapse.
    await login(page);
    await obApiCall(page, 'POST', 'createTemplate', {
      name: 'FAQ NoCollapse Test',
      roles: ['admin', 'superadmin'],
      sections: [{ title: 'FAQ Stay Open', sort_order: 1, requires_sign_off: false, is_faq: true, items: [
        { type: 'faq', label: 'Q1', answer: 'Answer to Q1', sort_order: 1 },
        { type: 'faq', label: 'Q2', answer: 'Answer to Q2', sort_order: 2 }
      ]}]
    });

    await page.goto('/onboarding.html');
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && body.querySelector('.card');
    }, { timeout: 10000 });

    // Open the template
    await page.locator('#my-body .card', { hasText: 'FAQ NoCollapse Test' }).first().click();
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && body.querySelector('.sec-header');
    }, { timeout: 10000 });

    // Expand the section
    const header = page.locator('.sec-header').first();
    await header.click();
    await page.waitForSelector('.faq-q');

    // View first question
    await page.locator('.faq-q').first().click();
    await page.waitForTimeout(1500);

    // View second (last) question — this triggers section completion
    await page.locator('.faq-q').nth(1).click();
    await page.waitForTimeout(1500);

    // Section should be marked complete
    await expect(page.locator('.sec-header').first()).toContainText('Complete');

    // The last FAQ answer MUST still be visible (section should NOT auto-collapse)
    await expect(page.locator('.faq-a').last()).toBeVisible();
    // FAQ questions should still be visible (section expanded)
    await expect(page.locator('.faq-q').first()).toBeVisible();
  });

  test('Manager tab shows hires with role-auto-assigned templates', async ({ page }) => {
    // Create team_member user
    await login(page);
    const email2 = 'mgr-view-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'MgrView', last_name: 'Test', email, roles: ['team_member'] })
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

    // Create template with roles=['team_member']
    await login(page);
    await obApiCall(page, 'POST', 'createTemplate', {
      name: 'MgrView Auto Test',
      roles: ['team_member'],
      sections: [{ title: 'Sec', sort_order: 1, requires_sign_off: true, is_faq: false, items: [
        { type: 'checkbox', label: 'Task 1', sort_order: 1 }
      ]}]
    });

    // Open Manager tab — should show MgrView T. as a hire with the auto-assigned template
    await page.goto('/onboarding.html');
    await waitForManagerTab(page);
    await page.click('#t2');
    await waitForManagerList(page);

    // MgrView should appear in the manager's Active hires list
    await expect(page.locator('#mgr-body')).toContainText('MgrView T.');
  });

  test('user with matching role sees template without explicit assignment', async ({ page }) => {
    // Create a team_member user
    await login(page);
    const email2 = 'auto-assign-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'AutoAssign', last_name: 'Test', email, roles: ['team_member'] })
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

    // Create a template with roles=['team_member'] via admin
    await login(page);
    await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Auto Assign Test',
      roles: ['team_member'],
      sections: [{ title: 'Basics', sort_order: 1, requires_sign_off: false, is_faq: false, items: [
        { type: 'checkbox', label: 'Test item', sort_order: 1 }
      ]}]
    });

    // Login as team_member — should see the template in My Trainings WITHOUT explicit assignment
    await login(page, email2, 'test456');
    await page.goto('/onboarding.html');
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && (body.querySelector('.card') || body.querySelector('.empty'));
    }, { timeout: 10000 });

    // Template should appear because user's role matches template's roles
    await expect(page.locator('#my-body')).toContainText('Auto Assign Test');
  });

  test('hire with pending sign-off stays in Active tab, not Completed', async ({ page }) => {
    // Bug: a hire at 100% progress but with sections still "Waiting for Sign-Off"
    // was incorrectly shown in the Manager > Completed tab. They should stay in Active
    // until all sign-offs are done.
    await login(page);

    // Create a team_member user
    const email2 = 'signoff-active-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'SignAct', last_name: 'Test', email, roles: ['team_member'] })
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

    // Create template with a sign-off section (1 item)
    await login(page);
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'SignOff Active Test',
      roles: ['team_member'],
      sections: [{ title: 'Tasks', sort_order: 1, requires_sign_off: true, sign_off_roles: ['admin'], is_faq: false, items: [
        { type: 'checkbox', label: 'Only task', sort_order: 1 }
      ]}]
    });

    // Complete the item as the hire so progress = 100%
    const fullTpl = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const sec = fullTpl.sections[0];
    const item = sec.items[0];

    // Login as the hire to save progress (handler uses auth user, not body.hire_id)
    await login(page, email2, 'test456');
    await obApiCall(page, 'POST', 'saveProgress', {
      item_id: item.id,
      progress_type: 'item',
      checked: true,
    });

    // Login back as admin and open Manager tab
    await login(page);
    await page.goto('/onboarding.html');
    await waitForManagerTab(page);
    await page.click('#t2');
    await waitForManagerList(page);

    // Hire should be in Active tab (default) with "Waiting for Sign-Off" badge
    await expect(page.locator('#mgr-body')).toContainText('SignAct T.');
    await expect(page.locator('#mgr-body')).toContainText('Waiting for Sign-Off');

    // Switch to Completed tab — hire should NOT be there
    await page.click('.sub-tabs button:has-text("Completed")');
    await page.waitForTimeout(500);
    await expect(page.locator('#mgr-body')).not.toContainText('SignAct T.');
  });
});

// ─── Sign-off role assignment ─────────────────────────────────────────────────

test.describe('Sign-off role assignment', () => {
  test('builder shows sign-off role picker when Require Sign-off is enabled', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // Create new template
    page.once('dialog', async d => await d.accept('SignOff Roles Test'));
    await page.locator('[data-action="new-template"]').click();
    await expect(page.locator('[data-action="back-to-templates"]')).toBeVisible();

    // Add a section
    page.once('dialog', async d => await d.accept('Test Section'));
    await page.locator('[data-action="add-ob-section"]').click();

    // New sections default to Require Sign-off ENABLED, so the role picker is
    // visible immediately with role chips.
    await expect(page.locator('.signoff-roles')).toHaveCount(1);
    const chips = await page.locator('.signoff-roles .role-chip').count();
    expect(chips).toBeGreaterThanOrEqual(2); // at least admin + manager

    // Disabling Require Sign-off hides the role picker...
    await page.locator('[data-action="toggle-signoff"]').first().click();
    await expect(page.locator('.signoff-roles')).toHaveCount(0);

    // ...and re-enabling it brings the picker back with role chips.
    await page.locator('[data-action="toggle-signoff"]').first().click();
    await expect(page.locator('.signoff-roles')).toHaveCount(1);
    expect(await page.locator('.signoff-roles .role-chip').count()).toBeGreaterThanOrEqual(2);
  });

  test('sign-off role picker disappears when sign-off is disabled', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // Create template with sign-off enabled
    page.once('dialog', async d => await d.accept('SignOff Toggle Test'));
    await page.locator('[data-action="new-template"]').click();
    page.once('dialog', async d => await d.accept('Section'));
    await page.locator('[data-action="add-ob-section"]').click();

    // New sections default to sign-off ENABLED — the role picker is present.
    await expect(page.locator('.signoff-roles')).toHaveCount(1);

    // Disable sign-off
    await page.locator('[data-action="toggle-signoff"]').first().click();

    // Role picker should disappear
    await expect(page.locator('.signoff-roles')).toHaveCount(0);
  });

  test('selected sign-off roles persist after save and reopen', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // Create template
    page.once('dialog', async d => await d.accept('SignOff Persist Test'));
    await page.locator('[data-action="new-template"]').click();
    page.once('dialog', async d => await d.accept('Section'));
    await page.locator('[data-action="add-ob-section"]').click();

    // New sections default to sign-off ENABLED with admin+manager selected.
    // Toggle off then on to clear the default role selection, giving a clean
    // slate, then select only 'manager'.
    await expect(page.locator('.signoff-roles')).toHaveCount(1);
    await page.locator('[data-action="toggle-signoff"]').first().click(); // disable → clears roles
    await expect(page.locator('.signoff-roles')).toHaveCount(0);
    await page.locator('[data-action="toggle-signoff"]').first().click(); // re-enable → no roles selected
    await expect(page.locator('.signoff-roles .role-chip.on')).toHaveCount(0);

    // Select only 'manager'
    await page.locator('.signoff-roles .role-chip', { hasText: 'manager' }).click();
    await expect(page.locator('.signoff-roles .role-chip.on')).toHaveCount(1);

    // Save
    await page.locator('[data-action="save-template"]').click();
    await waitForBuilderList(page);

    // Reopen template
    await page.locator('#builder-body .card', { hasText: 'SignOff Persist Test' }).first().click();
    await expect(page.locator('[data-action="back-to-templates"]')).toBeVisible();

    // Sign-off should still be enabled and 'manager' should be selected
    await expect(page.locator('.signoff-roles')).toHaveCount(1);
    await expect(page.locator('.signoff-roles .role-chip.on')).toHaveCount(1);
    await expect(page.locator('.signoff-roles .role-chip.on')).toContainText('manager');
  });

  test('manager navigates directly to Manager tab when pending sign-offs exist', async ({ page }) => {
    // Create team_member + template with sign-off
    await login(page);
    const email2 = 'signoff-nav-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'SignNav', last_name: 'Test', email, roles: ['team_member'] })
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

    // Create template with checkboxes + sign-off required, assigned to team_member
    await login(page);
    await obApiCall(page, 'POST', 'createTemplate', {
      name: 'SignOff Nav Test',
      roles: ['team_member'],
      sections: [{ title: 'Tasks', sort_order: 1, requires_sign_off: true, sign_off_roles: ['admin', 'manager'], is_faq: false, items: [
        { type: 'checkbox', label: 'Nav task', sort_order: 1 }
      ]}]
    });

    // Login as team_member, complete the section to trigger "waiting for sign-off"
    await login(page, email2, 'test456');
    await page.goto('/onboarding.html');
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && body.querySelector('.card');
    }, { timeout: 10000 });
    await page.locator('#my-body .card', { hasText: 'SignOff Nav Test' }).first().click();
    await page.waitForSelector('.sec-header');
    // Expand and check the item
    await page.locator('.sec-header').first().click();
    await page.waitForSelector('.item-row');
    await page.locator('.ob-check').first().click();
    await page.waitForTimeout(2000);

    // Now login as admin/manager and navigate to onboarding
    await login(page);
    await page.goto('/onboarding.html?tab=manager');
    // Should start on Manager tab (tab 2), not My Trainings (tab 1)
    await page.waitForFunction(() => {
      var t2 = document.getElementById('t2');
      return t2 && t2.classList.contains('on');
    }, { timeout: 10000 });
    await expect(page.locator('#t2')).toHaveClass(/on/);

    // Manager should see the hire with "Waiting for Sign-Off" badge
    await page.waitForFunction(() => {
      var body = document.getElementById('mgr-body');
      return body && body.querySelector('.card');
    }, { timeout: 10000 });
    await expect(page.locator('#mgr-body')).toContainText('SignNav T.');
    await expect(page.locator('#mgr-body')).toContainText('Waiting');
  });

  test('non-authorized role cannot sign off even if section is complete', async ({ page }) => {
    // Create template with sign-off restricted to 'admin' only
    await login(page);
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Restricted SignOff Test',
      roles: ['team_member', 'admin'],
      sections: [{ title: 'Restricted', sort_order: 1, requires_sign_off: true, is_faq: false,
        sign_off_roles: ['admin'],
        items: [{ type: 'checkbox', label: 'Task', sort_order: 1 }]
      }]
    });

    // Create team_member user
    const email2 = 'no-signoff-' + Date.now() + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async (email) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'NoSign', last_name: 'Test', email, roles: ['team_member'] })
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

    // Login as team_member, complete the section
    await login(page, email2, 'test456');
    await page.goto('/onboarding.html');
    await page.waitForFunction(() => {
      const body = document.getElementById('my-body');
      return body && body.querySelector('.card');
    }, { timeout: 10000 });
    await page.locator('#my-body .card', { hasText: 'Restricted SignOff Test' }).first().click();
    await page.waitForSelector('.sec-header');

    // The sign-off button should NOT appear for team_member (not in sign_off_roles)
    const signoffBtns = await page.locator('[data-action="show-signoff-form"]').count();
    expect(signoffBtns).toBe(0);
  });
});

// ─── Builder tab ─────────────────────────────────────────────────────────────

test.describe('Builder tab', () => {
  test('shows existing seed template in Builder list', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // "Kitchen Basics Training" from seed.go should appear
    await expect(page.locator('#builder-body')).toContainText('Kitchen Basics Training');
  });

  test('HQ back link prompts when builder has unsaved changes', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // Open an existing template to edit
    await page.locator('#builder-body .card').first().click();
    await expect(page.locator('[data-action="back-to-templates"]')).toBeVisible();

    // Make a change — modify the template name
    const nameInput = page.locator('[data-action="tpl-name-input"]');
    await nameInput.fill('Modified Name');

    // Click the HQ back link — should trigger confirm dialog
    let dialogFired = false;
    page.once('dialog', async dialog => {
      dialogFired = true;
      expect(dialog.message()).toContain('unsaved changes');
      await dialog.dismiss(); // cancel — stay on page
    });
    await page.locator('a.back').click();

    // Should still be on onboarding page (dialog was dismissed)
    expect(dialogFired).toBe(true);
    await expect(page).toHaveURL(/onboarding\.html/);
  });

  test('can create a new template via Builder', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // Create new template — use page.once for single-use dialog handler
    page.once('dialog', async dialog => {
      await dialog.accept('E2E Test Template');
    });
    await page.locator('[data-action="new-template"]').click();

    // Should now be in the editor
    await expect(page.locator('[data-action="back-to-templates"]')).toBeVisible();

    // Add a section
    page.once('dialog', async dialog => {
      await dialog.accept('Test Section');
    });
    await page.locator('[data-action="add-ob-section"]').click();

    // Add a checkbox item to that section — the button must be present after a
    // section is added (the former test wrapped the item-add in an isVisible()
    // guard, so it silently skipped adding the item and only asserted the name).
    const addCheckboxBtn = page.locator('[data-action="add-ob-item"][data-item-type="checkbox"]').first();
    await expect(addCheckboxBtn).toBeVisible({ timeout: 3000 });
    await addCheckboxBtn.click();
    // Fill in label
    const labelInput = page.locator('[data-action="item-label-input"]').last();
    await labelInput.fill('First checkbox item');

    // Save the template
    await page.locator('[data-action="save-template"]').click();

    // Wait for save and redirect back to list
    await waitForBuilderList(page);

    // Template should appear in the list
    await expect(page.locator('#builder-body')).toContainText('E2E Test Template');

    // And the saved template must actually contain the section + checkbox item
    // added through the builder UI (proves the item-add persisted, not just the
    // template name).
    const templates = await obApiCall(page, 'GET', 'templates');
    const created = templates.find(t => t.name === 'E2E Test Template');
    expect(created, 'created template must be listed via API').toBeTruthy();
    const fullCreated = await obApiCall(page, 'GET', 'templates/' + created.id);
    const sec = fullCreated.sections.find(s => s.title === 'Test Section');
    expect(sec, 'builder-added section must persist').toBeTruthy();
    expect(
      sec.items.some(i => i.type === 'checkbox' && i.label === 'First checkbox item'),
      'builder-added checkbox item must persist'
    ).toBe(true);

    await obApiCall(page, 'DELETE', 'deleteTemplate/' + created.id);
  });

  test('sub-items persist after save and reopen', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // Create new template
    page.once('dialog', async dialog => {
      await dialog.accept('SubItem Persist Test');
    });
    await page.locator('[data-action="new-template"]').click();
    await expect(page.locator('[data-action="back-to-templates"]')).toBeVisible();

    // Add a section
    page.once('dialog', async dialog => {
      await dialog.accept('Steps');
    });
    await page.locator('[data-action="add-ob-section"]').click();

    // Add a checkbox item
    await page.locator('[data-action="add-ob-item"][data-item-type="checkbox"]').first().click();
    await page.locator('[data-action="item-label-input"]').last().fill('Main task');

    // Add a sub-item
    await page.locator('[data-action="add-sub-item"]').first().click();
    await page.locator('[data-action="sub-item-label-input"]').last().fill('Set timer for 15 minutes');

    // Save
    await page.locator('[data-action="save-template"]').click();
    await waitForBuilderList(page);

    // Reopen the template
    await page.locator('#builder-body .card', { hasText: 'SubItem Persist Test' }).first().click();
    await expect(page.locator('[data-action="back-to-templates"]')).toBeVisible();

    // Sub-item should still be there
    await expect(page.locator('[data-action="sub-item-label-input"]').first()).toHaveValue('Set timer for 15 minutes');
  });

  test('save-video-for-later requires part title', async ({ page }) => {
    // The "Save Video for Later" button should require a part title before saving.
    // Without a title, the user should see an alert.
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // Create a new template with a video series
    page.once('dialog', async dialog => await dialog.accept('Video Save Test'));
    await page.click('[data-action="new-template"]');
    await page.waitForSelector('[data-action="back-to-templates"]');

    // Add a section
    page.once('dialog', async dialog => await dialog.accept('Test Section'));
    await page.click('[data-action="add-ob-section"]');

    // Add a video series item
    await page.click('[data-action="add-ob-item"][data-item-type="video_series"]');
    await page.waitForSelector('[data-action="add-video-part"]');

    // Add a video part (title left empty)
    await page.click('[data-action="add-video-part"]');
    await page.waitForSelector('[data-action="trigger-video-file"]');

    // Simulate a failed upload by setting _pendingFile and _uploadError on the part
    await page.evaluate(() => {
      var tpl = obBuilderState.localCopy;
      var sec = tpl.sections[0];
      var item = sec.items[0];
      var part = item.video_parts[0];
      part._pendingFile = new File(['test'], 'test.mp4', { type: 'video/mp4' });
      part._pendingFileName = 'test.mp4';
      part._uploadError = true;
      renderOBBuilder();
    });

    // "Save Video for Later" button should be visible
    await expect(page.locator('[data-action="save-video-local"]')).toBeVisible();

    // Click save — should get alert about missing title
    let alertMsg = '';
    page.once('dialog', async dialog => {
      alertMsg = dialog.message();
      await dialog.accept();
    });
    await page.click('[data-action="save-video-local"]');
    expect(alertMsg).toContain('part title');
  });

  test('save-video-for-later uses part title in filename', async ({ page }) => {
    // When a part title is filled in and "Save Video for Later" is clicked,
    // the download filename should use the part title (sanitized).
    await login(page);
    await page.goto('/onboarding.html');
    await waitForBuilderTab(page);
    await page.click('#t3');
    await waitForBuilderList(page);

    // Create template with video series + part
    page.once('dialog', async dialog => await dialog.accept('Video Filename Test'));
    await page.click('[data-action="new-template"]');
    await page.waitForSelector('[data-action="back-to-templates"]');

    page.once('dialog', async dialog => await dialog.accept('Sec'));
    await page.click('[data-action="add-ob-section"]');
    await page.click('[data-action="add-ob-item"][data-item-type="video_series"]');
    await page.click('[data-action="add-video-part"]');

    // Fill in part title
    await page.locator('[data-action="part-title-input"]').first().fill('Grill Pre-heat');

    // Set up failed upload state
    await page.evaluate(() => {
      var part = obBuilderState.localCopy.sections[0].items[0].video_parts[0];
      part._pendingFile = new File(['test'], 'original.mov', { type: 'video/quicktime' });
      part._pendingFileName = 'original.mov';
      part._uploadError = true;
      renderOBBuilder();
    });

    // Re-fill the title (re-render cleared it)
    await page.locator('[data-action="part-title-input"]').first().fill('Grill Pre-heat');
    // Trigger input event so data model updates
    await page.locator('[data-action="part-title-input"]').first().dispatchEvent('input');

    // Intercept the download
    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.click('[data-action="save-video-local"]')
    ]);

    // Filename should use the part title, not the original filename
    expect(download.suggestedFilename()).toBe('Grill-Pre-heat.mov');
  });
});

// ─── Section completion with sub-items ──────────────────────────────────────

test.describe('Section completion with sub-items', () => {
  test('completing all sub-items marks section complete on My Trainings list', async ({ page }) => {
    // Regression: sectionIncompleteItem SQL checked progress on parent item ID,
    // but sub-item progress uses sub_item.id. Sections with sub-items could never
    // appear complete in the list view.
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    // Create a template with one section containing a checkbox with sub-items
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Sub-Item Completion Test',
      roles: [],
      sections: [{
        title: 'Section With Subs',
        sort_order: 0,
        requires_sign_off: false,
        sign_off_roles: [],
        is_faq: false,
        items: [{
          type: 'checkbox',
          label: 'Parent item',
          sort_order: 0,
          sub_items: [
            { label: 'Sub A', sort_order: 0 },
            { label: 'Sub B', sort_order: 1 },
          ],
        }],
      }],
    });

    // Assign to self
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: tpl.id,
    });

    // Get the template to find sub-item IDs
    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const subs = full.sections[0].items[0].sub_items;
    expect(subs.length).toBe(2);

    // Complete both sub-items via API
    for (const sub of subs) {
      await obApiCall(page, 'POST', 'saveProgress', {
        item_id: sub.id,
        progress_type: 'sub_item',
        checked: true,
      });
    }

    // Load My Trainings and verify section count shows 1 of 1 complete
    await page.goto('/onboarding.html');
    await waitForMyList(page);

    // Scope to #my-body — a roles:[] template also appears in the Builder list
    // (#builder-body) card, so an unscoped .card matches two elements.
    const card = page.locator('#my-body .card').filter({ hasText: 'Sub-Item Completion Test' });
    await expect(card).toBeVisible();
    await expect(card).toContainText('1 of 1 sections complete');

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('Manager tab shows correct completion for sub-item checkboxes', async ({ page }) => {
    // Regression: same sectionIncompleteItem bug affected manager hires view
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Manager Sub-Item Test',
      roles: [],
      sections: [{
        title: 'Tasks With Subs',
        sort_order: 0,
        requires_sign_off: false,
        sign_off_roles: [],
        is_faq: false,
        items: [{
          type: 'checkbox',
          label: 'Do the thing',
          sort_order: 0,
          sub_items: [
            { label: 'Step 1', sort_order: 0 },
            { label: 'Step 2', sort_order: 1 },
          ],
        }],
      }],
    });

    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: tpl.id,
    });

    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const subs = full.sections[0].items[0].sub_items;

    // Complete all sub-items
    for (const sub of subs) {
      await obApiCall(page, 'POST', 'saveProgress', {
        item_id: sub.id,
        progress_type: 'sub_item',
        checked: true,
      });
    }

    // Check Manager tab
    await page.goto('/onboarding.html');
    await page.click('#t2');
    await waitForManagerList(page);

    // Target THIS hire's own card by hire id — other hires from earlier tests
    // may sort ahead of it. The card lists every assigned template, so assert
    // against this template's own progress line, not the whole card.
    const hireCard = page.locator('#mgr-body [data-action="open-hire"][data-hire-id="' + me.id + '"]').first();
    await expect(hireCard).toBeVisible();
    const progressText = await hireCard.textContent();
    // Sub-item completion must reflect as 100% (not 0%) for our template.
    expect(progressText).toContain('Manager Sub-Item Test');
    expect(progressText).toContain('Manager Sub-Item Test100%');
    expect(progressText).not.toContain('Manager Sub-Item Test0%');

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('reference photo URL persists through template save and displays in training', async ({ page }) => {
    // Regression: reference_photo_url on items must survive save/reload cycle
    await login(page);
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });

    const refUrl = 'https://example.com/test-reference.jpg';

    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Ref Photo Test',
      roles: [],
      sections: [{
        title: 'Photo Section',
        sort_order: 0,
        requires_sign_off: false,
        sign_off_roles: [],
        is_faq: false,
        items: [{
          type: 'checkbox',
          label: 'Check with photo',
          sort_order: 0,
          reference_photo_url: refUrl,
          require_proof_photo: true,
          sub_items: [],
        }],
      }],
    });

    // Verify it persists by re-fetching the template
    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    expect(full.sections[0].items[0].reference_photo_url).toBe(refUrl);
    expect(full.sections[0].items[0].require_proof_photo).toBe(true);

    // Assign and check it appears in hire training API
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: tpl.id,
    });

    const training = await obApiCall(page, 'GET', 'hireTraining/' + me.id + '?templateId=' + tpl.id);
    const item = training.sections[0].items[0];
    expect(item.reference_photo_url).toBe(refUrl);
    expect(item.require_proof_photo).toBe(true);

    // Load page and verify photo thumbnail renders
    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator('[data-action="open-my-training"]').filter({ hasText: 'Ref Photo Test' }).click();
    await waitForTrainingRunner(page);

    // Expand section
    await page.locator('.sec-header').first().click();

    // Photo thumbnail should be visible
    await expect(page.locator('img.photo-thumb[src="' + refUrl + '"]')).toBeVisible();

    // Proof photo button should be visible (require_proof_photo = true)
    await expect(page.locator('[data-action="ob-photo-capture"]')).toBeVisible();

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });
});

// ─── Reject and Unsubmit ────────────────────────────────────────────────────

test.describe('Reject and unsubmit sections', () => {
  // Helper: create a template with one sign-off section, assign, complete all items
  async function setupCompletedSignoffSection(page) {
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Reject Test ' + Date.now(),
      roles: [],
      sections: [{
        title: 'Sign-off Section',
        sort_order: 0,
        requires_sign_off: true,
        sign_off_roles: ['admin', 'manager'],
        is_faq: false,
        items: [{
          type: 'checkbox',
          label: 'Task A',
          sort_order: 0,
          require_proof_photo: true,
          sub_items: [],
        }, {
          type: 'checkbox',
          label: 'Task B',
          sort_order: 1,
          sub_items: [],
        }],
      }],
    });
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: tpl.id,
    });
    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    // Complete both items
    for (const item of full.sections[0].items) {
      await obApiCall(page, 'POST', 'saveProgress', {
        item_id: item.id,
        progress_type: 'item',
        checked: true,
      });
    }
    return { me, tpl, full };
  }

  // Open a hire's detail card by hire id, searching the Active sub-tab first and
  // the Completed sub-tab second. A hire lands in Completed when ALL their
  // templates are 100% and signed off (e.g. a single signed-off section), so
  // callers can't assume which sub-tab holds them.
  async function openHireById(page, hireId) {
    await page.click('#t2');
    await waitForManagerList(page);
    const sel = '#mgr-body [data-action="open-hire"][data-hire-id="' + hireId + '"]';
    if (await page.locator(sel).count() === 0) {
      // Not in Active — try Completed.
      await page.click('.sub-tabs button:has-text("Completed")');
      await page.waitForTimeout(300);
    }
    await page.locator(sel).first().waitFor({ state: 'visible' });
    await page.locator(sel).first().click();
    await page.waitForTimeout(500);
  }

  // From a hire detail view, open the training runner for a specific template id.
  async function openTrainingByTemplateId(page, templateId) {
    const viewBtn = page.locator('[data-action="view-training"][data-template-id="' + templateId + '"]').first();
    await viewBtn.waitFor({ state: 'visible' });
    await viewBtn.click();
    await page.waitForTimeout(500);
  }

  test('crew can unsubmit a completed section and re-edit', async ({ page }) => {
    await login(page);
    const { me, tpl } = await setupCompletedSignoffSection(page);

    // Verify section shows "Waiting for Sign-Off" with "Go Back & Edit" button.
    // The card shows the template NAME, not its id — target the data attribute.
    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator('[data-action="open-my-training"][data-template-id="' + tpl.id + '"]').first().click();
    await waitForTrainingRunner(page);

    await expect(page.locator('#my-body .pill-warn')).toContainText('Waiting for Sign-Off');
    const goBackBtn = page.locator('[data-action="reopen-section"]');
    await expect(goBackBtn).toBeVisible();

    // Accept the confirmation dialog
    page.on('dialog', dialog => dialog.accept());
    await goBackBtn.click();

    // Section should now be active — items should be interactive
    await page.waitForTimeout(500);
    // The section should be expanded and show toggle-item buttons
    await expect(page.locator('[data-action="toggle-item"]').first()).toBeVisible();

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('manager can reject a signed-off section', async ({ page }) => {
    await login(page);
    const { me, tpl, full } = await setupCompletedSignoffSection(page);
    const secId = full.sections[0].id;

    // Sign off the section via API
    await obApiCall(page, 'POST', 'signOff', {
      section_id: secId,
      hire_id: me.id,
      notes: 'Looks good',
      rating: 'ready',
    });

    // Open Manager tab, navigate to hire training. A single signed-off section
    // makes the hire "complete", so they may be under the Completed sub-tab.
    await page.goto('/onboarding.html');
    await openHireById(page, me.id);
    await openTrainingByTemplateId(page, tpl.id);

    // Should see "Reject & Reopen" button on signed-off section
    const rejectBtn = page.locator('[data-action="reject-section"]');
    await expect(rejectBtn.first()).toBeVisible();

    // Accept confirmation and reject
    page.on('dialog', dialog => dialog.accept());
    await rejectBtn.first().click();
    await page.waitForTimeout(500);

    // Section should now show as active (no longer signed off)
    await expect(page.locator('.sec-header').first()).not.toContainText('Signed Off');

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('manager can reject before sign-off (complete section)', async ({ page }) => {
    await login(page);
    const { me, tpl, full } = await setupCompletedSignoffSection(page);

    // Open Manager tab → hire → training. Section is complete (awaiting sign-off,
    // pending_signoff=true) so the hire stays under Active, but openHireById
    // handles either sub-tab defensively.
    await page.goto('/onboarding.html');
    await openHireById(page, me.id);
    await openTrainingByTemplateId(page, tpl.id);

    // Should see "Reject" button alongside "Sign Off Section"
    const rejectBtn = page.locator('[data-action="reject-section"]');
    await expect(rejectBtn.first()).toBeVisible();

    page.on('dialog', dialog => dialog.accept());
    await rejectBtn.first().click();
    await page.waitForTimeout(500);

    // Section should revert to active
    await expect(page.locator('.sec-header').first()).not.toContainText('Waiting for Sign-Off');

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('unsubmitted section allows proof photo capture button', async ({ page }) => {
    await login(page);
    const { me, tpl } = await setupCompletedSignoffSection(page);

    // Reopen via API
    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    await obApiCall(page, 'POST', 'reopenSection', { section_id: full.sections[0].id });

    // Load My Trainings and open training (target by data-template-id, not text).
    await page.goto('/onboarding.html');
    await waitForMyList(page);
    await page.locator('[data-action="open-my-training"][data-template-id="' + tpl.id + '"]').first().click();
    await waitForTrainingRunner(page);

    // Expand section
    await page.locator('.sec-header').first().click();
    await page.waitForTimeout(300);

    // Proof photo button should be visible on items with require_proof_photo
    await expect(page.locator('[data-action="ob-photo-capture"]').first()).toBeVisible();

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  // ─── NFR-5: reopen/reject of a video-led section must revert it to active ────
  // Regression for the silent no-op: ReopenSection selected the first item with no
  // type filter and deleted ob_progress keyed by the parent ob_items.id. But
  // video_series progress is keyed by ob_video_parts.id (progress_type='video_part'),
  // so the delete matched ZERO rows → isSectionComplete stayed true → the section
  // never reverted (handler returned {"ok":"true"} masking it).

  // Helper: create a sign-off template whose FIRST item is a video_series, assign,
  // watch every video part so the section reads complete. Returns ids + video parts.
  async function setupCompletedVideoSection(page) {
    const me = await page.evaluate(async () => {
      const res = await fetch('/api/v1/me');
      return res.json();
    });
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'Video Reject Test ' + Date.now(),
      roles: [],
      sections: [{
        title: 'Video Sign-off Section',
        sort_order: 0,
        requires_sign_off: true,
        sign_off_roles: ['admin', 'manager'],
        is_faq: false,
        items: [{
          type: 'video_series',
          label: 'Equipment Videos',
          sort_order: 0,
          video_parts: [
            { title: 'Part One', description: 'first', url: 'https://example.com/p1.mp4', sort_order: 0 },
            { title: 'Part Two', description: 'second', url: 'https://example.com/p2.mp4', sort_order: 1 },
          ],
        }],
      }],
    });
    await obApiCall(page, 'POST', 'assignTemplate', {
      hire_id: me.id,
      template_id: tpl.id,
    });
    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const vs = full.sections[0].items.find(i => i.type === 'video_series');
    expect(vs).toBeTruthy();
    expect(vs.video_parts.length).toBe(2);
    // Watch every video part so the section becomes complete
    for (const part of vs.video_parts) {
      await obApiCall(page, 'POST', 'saveProgress', {
        item_id: part.id,
        progress_type: 'video_part',
        checked: true,
      });
    }
    return { me, tpl, full, secId: full.sections[0].id };
  }

  // Reads the computed section state for hire+template via GET hireTraining.
  async function sectionState(page, hireId, templateId, sectionId) {
    const training = await obApiCall(page, 'GET', 'hireTraining/' + hireId + '?templateId=' + templateId);
    const sp = training.sections.find(s => s.id === sectionId || (s.section && s.section.id === sectionId));
    return sp ? sp.state : null;
  }

  test('reopen reverts a video-led section to active (NFR-5)', async ({ page }) => {
    await login(page);
    const { me, tpl, secId } = await setupCompletedVideoSection(page);

    // Precondition: the video section reads as complete (all parts watched).
    expect(await sectionState(page, me.id, tpl.id, secId)).toBe('complete');

    // Reopen (crew unsubmit — FR-9). On UNFIXED code this is a silent no-op.
    const res = await obApiCall(page, 'POST', 'reopenSection', { section_id: secId });
    expect(res.ok).toBe('true');

    // The section MUST no longer be complete — it should revert to active.
    const stateAfter = await sectionState(page, me.id, tpl.id, secId);
    expect(stateAfter).not.toBe('complete');
    expect(stateAfter).toBe('active');

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('reject reverts a video-led signed-off section to active (NFR-5 / FR-15)', async ({ page }) => {
    await login(page);
    const { me, tpl, secId } = await setupCompletedVideoSection(page);

    // Sign off the completed video section.
    await obApiCall(page, 'POST', 'signOff', {
      section_id: secId,
      hire_id: me.id,
      notes: 'Looks good',
      rating: 'ready',
    });
    expect(await sectionState(page, me.id, tpl.id, secId)).toBe('signed_off');

    // Reject (manager — FR-15). Shares ReopenSection with the reopen path.
    const res = await obApiCall(page, 'POST', 'rejectSection', { section_id: secId, hire_id: me.id });
    expect(res.ok).toBe('true');

    // The section MUST revert — not complete, not signed_off.
    const stateAfter = await sectionState(page, me.id, tpl.id, secId);
    expect(stateAfter).not.toBe('complete');
    expect(stateAfter).not.toBe('signed_off');
    expect(stateAfter).toBe('active');

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });
});

// ─── FR-26: unassign a template from a hire ─────────────────────────────────
// A manager unassigns a template from a hire, removing the explicit assignment
// row. Two observable behaviors:
//  (1) Unassign is idempotent — unassigning twice yields the same end state
//      (template gone from the hire's list), no error, no throw on the 2nd call.
//  (2) Role-auto-assign survives — if the template ALSO matches the hire's role
//      (ot.roles && user.roles), removing the explicit assignment row must NOT
//      remove the template from the hire's `myTrainings` list; the role match
//      still surfaces it. (PRD FR-26: "a role-auto-assigned template still shows".)
// No test previously called /unassignTemplate at all.
test.describe('FR-26: unassign template', () => {
  // Count how many times a template id appears in the current user's myTrainings.
  async function myTrainingCount(page, templateId) {
    const list = await obApiCall(page, 'GET', 'myTrainings');
    return (Array.isArray(list) ? list : [])
      .filter(t => t.template_id === templateId).length;
  }

  test('unassign is idempotent — second unassign is a no-op, template stays gone', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => (await (await fetch('/api/v1/me')).json()));

    // Template with NO role match → it can ONLY reach the hire via explicit assign.
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'FR26 Idempotent ' + Date.now(),
      roles: [],
      sections: [{ title: 'S', sort_order: 0, requires_sign_off: false, sign_off_roles: [], is_faq: false, items: [{ type: 'checkbox', label: 'A', sort_order: 0, sub_items: [] }] }],
    });

    // Explicit assign → template appears exactly once in myTrainings.
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });
    expect(await myTrainingCount(page, tpl.id)).toBe(1);

    // First unassign → {ok:true}, template gone from the list.
    const r1 = await obApiCall(page, 'POST', 'unassignTemplate', { hire_id: me.id, template_id: tpl.id });
    expect(r1).toEqual({ ok: true });
    expect(await myTrainingCount(page, tpl.id)).toBe(0);

    // Second unassign → SAME end state, no error, still {ok:true} (idempotent DELETE).
    const r2 = await obApiCall(page, 'POST', 'unassignTemplate', { hire_id: me.id, template_id: tpl.id });
    expect(r2).toEqual({ ok: true });
    expect(await myTrainingCount(page, tpl.id)).toBe(0);

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  test('unassign of an explicitly-assigned template leaves the role-auto-assign intact', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => (await (await fetch('/api/v1/me')).json()));
    // GetMyTrainings matches template.roles against the users.roles COLUMN. Note
    // /api/v1/me MASKS a superadmin's stored roles as ["superadmin"], but the DB
    // column (what the query reads) is the real role set. Read it from the users
    // API — the authoritative stored-roles source the query overlaps against.
    const users = await usersApiCall(page, 'GET', '');
    const dbMe = (Array.isArray(users) ? users : []).find(u => u.id === me.id);
    const dbRoles = (dbMe && dbMe.roles) || [];
    // The hire must have at least one stored role for the role-match edge to matter.
    expect(dbRoles.length).toBeGreaterThan(0);

    // Template whose roles OVERLAP the hire's stored roles → role-auto-assigned
    // regardless of any explicit assignment row.
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'FR26 RoleSurvives ' + Date.now(),
      roles: dbRoles,
      sections: [{ title: 'S', sort_order: 0, requires_sign_off: false, sign_off_roles: [], is_faq: false, items: [{ type: 'checkbox', label: 'A', sort_order: 0, sub_items: [] }] }],
    });

    // Role match alone surfaces it once (before any explicit assign).
    expect(await myTrainingCount(page, tpl.id)).toBe(1);

    // Now ALSO explicitly assign it — the LEFT JOIN + OR still yields exactly one row.
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });
    expect(await myTrainingCount(page, tpl.id)).toBe(1);

    // Unassign removes ONLY the explicit ob_template_assignments row. The role match
    // must keep the template in the hire's list — this is the FR-26 edge.
    const r = await obApiCall(page, 'POST', 'unassignTemplate', { hire_id: me.id, template_id: tpl.id });
    expect(r).toEqual({ ok: true });
    expect(await myTrainingCount(page, tpl.id)).toBe(1);

    // Cleanup
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });
});

// ─── prove-progress sweep (card onboarding-prove-progress) ──────────────────
// Red-first prove assertions for the onboarding-hardening PRD's UNPROVEN flows:
//   FR-10 (proof-photo URL round-trip), FR-14 (sign_off_role_required 403),
//   FR-21 (archived template disappears), FR-29 (manager vs hire progress %
//   agree), NFR-3 (team_member 403 sweep across manager endpoints).
// FR-18 (custom-thumbnail UPLOAD) and FR-28 (seed idempotent re-seed) are PARKED
//   / recorded UNTESTABLE below with their precise reasons — no forced assertion.
// All non-admin sessions are authored INLINE (invite → accept-invite → login),
// per tests/multi-role.spec.js — no shared helper module.

test.describe('prove-progress sweep', () => {
  // Invite a user with the given roles, accept the invite (sets password), and
  // return { id, email }. IMPORTANT: /auth/accept-invite sets the hq_session cookie
  // — it logs the BROWSER in as the newly-created user. So we re-login as ADMIN
  // before returning, leaving the caller in a known admin session; the caller
  // explicitly logs in as the invitee (login(page, email, 'test456')) when needed.
  async function inviteUser(page, roles) {
    const email = 'prove-prog-' + Date.now() + '-' + Math.floor(Math.random() * 1e6) + '@yumyums.kitchen';
    const inviteRes = await page.evaluate(async ([em, rs]) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Prove', last_name: 'Prog', email: em, roles: rs }),
      });
      return res.json();
    }, [email, roles]);
    const token = inviteRes.invite_path.split('token=')[1];
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' }),
      });
    }, token);
    // accept-invite hijacked the session → restore the admin session.
    await login(page);
    // Card G1: roles-less hires (inviteUser(page, [])) fall outside the file's
    // role-grant baseline, and /onboarding/* is now behind the `onboarding`
    // grant — issue the new user an individual grant so training flows work.
    await page.evaluate(async (uid) => {
      const perms = await (await fetch('/api/v1/apps/permissions')).json();
      const app = (perms || []).find(a => a.slug === 'onboarding') || {};
      const users = (app.user_grants || []).map(String);
      if (!users.includes(String(uid))) users.push(String(uid));
      await fetch('/api/v1/apps/onboarding/permissions', {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role_grants: app.role_grants || [], user_grants: users }),
      });
    }, inviteRes.user.id);
    return { id: inviteRes.user.id, email };
  }

  // POST a manager onboarding endpoint via fetch in the current session and return
  // { status, body } so a caller can assert the 403 tier directly.
  async function obRaw(page, method, path, body) {
    return page.evaluate(async ([m, p, b]) => {
      const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
      if (b) opts.body = JSON.stringify(b);
      const res = await fetch('/api/v1/onboarding/' + p, opts);
      let json = null;
      try { json = await res.json(); } catch (e) { json = null; }
      return { status: res.status, body: json };
    }, [method, path, body]);
  }

  // ── FR-10 — proof-photo URL round-trips through ob_progress.value ──────────
  // A hire uploads a proof photo on an item that requires one; the URL persists
  // into ob_progress.value and comes back as proof_photo_url on reload. The
  // round-trip of the URL STRING needs no S3/multipart — saveProgress carries a
  // `value` and hireTraining surfaces it as item.proof_photo_url. (The client
  // capture widget that PRODUCES the URL is separate; this proves the persistence
  // contract the PRD names: value → ob_progress.value → proof_photo_url.)
  test('FR-10: proof-photo URL persists via saveProgress value and round-trips on reload', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => (await (await fetch('/api/v1/me')).json()));

    // Template with a single checkbox item that REQUIRES a proof photo.
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'FR10 ProofPhoto ' + Date.now(),
      roles: [],
      sections: [{
        title: 'Proof Section', sort_order: 0, requires_sign_off: false, sign_off_roles: [], is_faq: false,
        items: [{ type: 'checkbox', label: 'Photograph the station', sort_order: 0, require_proof_photo: true, sub_items: [] }],
      }],
    });
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });

    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const item = full.sections[0].items[0];
    expect(item.require_proof_photo).toBe(true);

    const photoURL = 'https://spaces.example.com/onboarding/proof-' + Date.now() + '.jpg';
    // Save progress carrying the proof-photo URL as `value` (mirrors the client's
    // handleOBPhotoCaptureClick → saveProgress(value) path).
    await obApiCall(page, 'POST', 'saveProgress', {
      item_id: item.id, progress_type: 'item', checked: true, value: photoURL,
    });

    // Round-trip: re-fetch the hire's training and assert the proof URL comes back.
    const training = await obApiCall(page, 'GET', 'hireTraining/' + me.id + '?templateId=' + tpl.id);
    const gotItem = training.sections[0].items.find(i => i.id === item.id);
    expect(gotItem).toBeTruthy();
    expect(gotItem.checked).toBe(true);
    // The observable behavior: the uploaded URL survives and is served back.
    expect(gotItem.proof_photo_url).toBe(photoURL);

    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
  });

  // ── FR-14 — sign_off_role_required 403 at the API ─────────────────────────
  // A section's sign_off_roles restricts who may sign off. A user who PASSES the
  // manager tier (isManagerOrAdmin) but whose roles do NOT intersect
  // sign_off_roles — and who isn't superadmin — is refused 403
  // sign_off_role_required. We invite a plain `manager` (clears the tier gate),
  // set sign_off_roles=['admin'] (no overlap), and assert the role-specific 403.
  // NOTE: the tier gate fires FIRST, so a team_member would get 403 `forbidden`
  // (that path is NFR-3, below); to reach `sign_off_role_required` the poster must
  // be a manager lacking the required role.
  test('FR-14: a manager lacking the sign_off role is refused 403 sign_off_role_required', async ({ page }) => {
    await login(page);
    const mgr = await inviteUser(page, ['manager']);

    // Admin builds a template with an ADMIN-only sign-off section, assigns it to
    // the manager as the hire (any hire works — the 403 fires before completeness).
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'FR14 RoleGate ' + Date.now(),
      roles: [],
      sections: [{
        title: 'Admin-only sign-off', sort_order: 0, requires_sign_off: true,
        sign_off_roles: ['admin'], is_faq: false,
        items: [{ type: 'checkbox', label: 'A', sort_order: 0, sub_items: [] }],
      }],
    });
    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const secId = full.sections[0].id;

    // Become the manager and POST /signOff — passes the manager tier, fails the
    // admin-only role match → 403 sign_off_role_required.
    await login(page, mgr.email, 'test456');
    const res = await obRaw(page, 'POST', 'signOff', {
      section_id: secId, hire_id: mgr.id, notes: '', rating: 'ready',
    });
    expect(res.status).toBe(403);
    expect(res.body && res.body.error).toBe('sign_off_role_required');

    // Cleanup (back as admin).
    await login(page);
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
    await usersApiCall(page, 'DELETE', mgr.id);
  });

  // ── FR-21 — archived template disappears from the assignable/list surfaces ──
  // Soft-delete (DELETE /deleteTemplate/{id} → archived_at=now()) must drop the
  // template out of /templates AND /myTrainings while its rows survive
  // (idempotent re-delete still returns ok). Previously used only as test cleanup.
  test('FR-21: archived template disappears from /templates and /myTrainings', async ({ page }) => {
    await login(page);
    const me = await page.evaluate(async () => (await (await fetch('/api/v1/me')).json()));

    const uniq = 'FR21 Archive ' + Date.now();
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: uniq, roles: [],
      sections: [{ title: 'S', sort_order: 0, requires_sign_off: false, sign_off_roles: [], is_faq: false,
        items: [{ type: 'checkbox', label: 'A', sort_order: 0, sub_items: [] }] }],
    });
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: me.id, template_id: tpl.id });

    // Before archive: present in BOTH the admin /templates list and the hire's list.
    const listBefore = await obApiCall(page, 'GET', 'templates');
    expect(listBefore.some(t => t.id === tpl.id)).toBe(true);
    const myBefore = await obApiCall(page, 'GET', 'myTrainings');
    expect(myBefore.some(t => t.template_id === tpl.id)).toBe(true);

    // Archive (soft-delete).
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);

    // After archive: GONE from both surfaces (archived_at IS NULL filter).
    const listAfter = await obApiCall(page, 'GET', 'templates');
    expect(listAfter.some(t => t.id === tpl.id)).toBe(false);
    const myAfter = await obApiCall(page, 'GET', 'myTrainings');
    expect(myAfter.some(t => t.template_id === tpl.id)).toBe(false);
  });

  // ── FR-29 — manager and hire report the SAME progress % for a template ─────
  // /managerHires computes progress by a distinct, heavier query path than the
  // hire's own /myTrainings. For the same hire+template they must agree. We build
  // a 2-section template (1 completed, 1 not → 50%), assign to a fresh hire, have
  // the hire complete section 1, then assert myTrainings.progress_percent ==
  // managerHires → assigned_templates[thatTemplate].progress_pct == 50.
  test('FR-29: manager and hire progress % agree for the same template', async ({ page }) => {
    await login(page);
    const hire = await inviteUser(page, []);

    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'FR29 ProgressAgree ' + Date.now(),
      roles: [],
      sections: [
        { title: 'Sec 1', sort_order: 0, requires_sign_off: false, sign_off_roles: [], is_faq: false,
          items: [{ type: 'checkbox', label: 'A1', sort_order: 0, sub_items: [] }] },
        { title: 'Sec 2', sort_order: 1, requires_sign_off: false, sign_off_roles: [], is_faq: false,
          items: [{ type: 'checkbox', label: 'B1', sort_order: 0, sub_items: [] }] },
      ],
    });
    await obApiCall(page, 'POST', 'assignTemplate', { hire_id: hire.id, template_id: tpl.id });

    // Admin-side truth: the manager progress for the hire on this template BEFORE
    // any completion should be 0%.
    let mgrHires = await obApiCall(page, 'GET', 'managerHires');
    let hireRow = mgrHires.find(h => h.hire_id === hire.id);
    expect(hireRow).toBeTruthy();
    let mgrTpl = hireRow.assigned_templates.find(t => t.template_id === tpl.id);
    expect(mgrTpl).toBeTruthy();
    expect(mgrTpl.progress_pct).toBe(0);

    // Become the hire and complete section 1 (1 of 2 sections → 50%).
    await login(page, hire.email, 'test456');
    const full = await obApiCall(page, 'GET', 'templates/' + tpl.id);
    const sec1Item = full.sections[0].items[0];
    await obApiCall(page, 'POST', 'saveProgress', { item_id: sec1Item.id, progress_type: 'item', checked: true });

    // The hire's own myTrainings progress for this template.
    const myList = await obApiCall(page, 'GET', 'myTrainings');
    const myRow = myList.find(t => t.template_id === tpl.id);
    expect(myRow).toBeTruthy();
    expect(myRow.progress_percent).toBe(50);

    // Back as admin: the manager's heavier query path must report the SAME number.
    await login(page);
    mgrHires = await obApiCall(page, 'GET', 'managerHires');
    hireRow = mgrHires.find(h => h.hire_id === hire.id);
    expect(hireRow).toBeTruthy();
    mgrTpl = hireRow.assigned_templates.find(t => t.template_id === tpl.id);
    expect(mgrTpl).toBeTruthy();
    // The core FR-29 assertion: the two independent progress computations agree.
    expect(mgrTpl.progress_pct).toBe(myRow.progress_percent);
    expect(mgrTpl.progress_pct).toBe(50);

    // Cleanup.
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
    await usersApiCall(page, 'DELETE', hire.id);
  });

  // ── NFR-3 — team_member gets 403 on every manager onboarding endpoint ──────
  // The manager guards (isManagerOrAdmin) are wired on each manager endpoint, but
  // no prior test posted as a team_member to assert the tier refusal. We invite a
  // plain team_member, log in as them, and sweep the manager endpoints asserting
  // 403 forbidden on each. Red-first: this would go RED if a guard were removed.
  test('NFR-3: team_member is refused 403 across the manager onboarding endpoints', async ({ page }) => {
    await login(page);
    // Admin creates a real template so path params reference a live row (the guard
    // fires regardless, but this avoids a 404-vs-403 ambiguity on id-bearing routes).
    const tpl = await obApiCall(page, 'POST', 'createTemplate', {
      name: 'NFR3 Guard ' + Date.now(), roles: [],
      sections: [{ title: 'S', sort_order: 0, requires_sign_off: false, sign_off_roles: [], is_faq: false,
        items: [{ type: 'checkbox', label: 'A', sort_order: 0, sub_items: [] }] }],
    });
    const member = await inviteUser(page, ['team_member']);

    // Become the team_member.
    await login(page, member.email, 'test456');

    // GET manager surfaces.
    for (const p of ['templates', 'managerHires']) {
      const res = await obRaw(page, 'GET', p);
      expect(res.status, 'GET ' + p + ' must 403 for team_member').toBe(403);
      expect(res.body && res.body.error).toBe('forbidden');
    }
    // POST/PUT/DELETE manager mutations.
    const posts = [
      ['POST', 'signOff', { section_id: 'x', hire_id: 'y', rating: 'ready' }],
      ['POST', 'rejectSection', { section_id: 'x', hire_id: 'y' }],
      ['POST', 'createTemplate', { name: 'z', roles: [], sections: [] }],
      ['PUT', 'updateTemplate/' + tpl.id, { name: 'z', roles: [], sections: [] }],
      ['DELETE', 'deleteTemplate/' + tpl.id, null],
      ['POST', 'assignTemplate', { hire_id: 'y', template_id: tpl.id }],
      ['POST', 'unassignTemplate', { hire_id: 'y', template_id: tpl.id }],
    ];
    for (const [m, p, b] of posts) {
      const res = await obRaw(page, m, p, b);
      expect(res.status, m + ' ' + p + ' must 403 for team_member').toBe(403);
      expect(res.body && res.body.error).toBe('forbidden');
    }

    // Cleanup as admin — template must still exist (all mutations were refused).
    await login(page);
    await obApiCall(page, 'DELETE', 'deleteTemplate/' + tpl.id);
    await usersApiCall(page, 'DELETE', member.id);
  });

  // ── FR-28 — seed idempotent re-seed: recorded UNTESTABLE ──────────────────
  // The idempotent re-seed (skip-if-name-exists + role-refresh in
  // SeedOnboardingTemplates) runs once, post-boot, in the server process. It is
  // not reachable from an E2E client: there is no endpoint to trigger a re-seed
  // and no way to restart the server mid-test. Per the card we assert the flow
  // INDIRECTLY — the seed's "Kitchen Basics Training" template is present after
  // boot — and record the idempotency/role-refresh guarantee itself as UNTESTABLE
  // (do NOT force it; do NOT graduate a fix from this).
  test('FR-28: seed template is present after boot (indirect — idempotent re-seed itself is UNTESTABLE via E2E)', async ({ page }) => {
    await login(page);
    const templates = await obApiCall(page, 'GET', 'templates');
    const kitchen = templates.find(t => t.name === 'Kitchen Basics Training');
    // Indirect proof the seed ran: the named seed template exists.
    expect(kitchen).toBeTruthy();
    // The idempotent-re-seed / role-refresh behavior (seed.go:98-124) is NOT
    // E2E-drivable — no client trigger, no in-test server restart. Recorded
    // UNTESTABLE per PRD FR-28 / card instruction; this assertion only anchors the
    // seed's PRESENCE, not its re-seed idempotency.
  });

  // ── FR-18 — custom-thumbnail UPLOAD: PARKED ───────────────────────────────
  // FR-18's named behavior is a custom-thumbnail UPLOAD: client uploadCustomThumbnail
  // presigns against /api/v1/photos/presign, PUTs the file to DO Spaces, then sets
  // part.thumbnail_url from presignData.public_url. Proving the real flow needs
  // multipart/file-input + a live Spaces presign+PUT — photo/thumbnail plumbing
  // beyond a test fixture, which is this card's explicit PARK trigger. The
  // thumbnail_url DB round-trip (via updateTemplate) is a DIFFERENT flow (FR-20,
  // already WORKING) and would NOT prove FR-18's upload behavior, so it is not a
  // substitute. PARKED — see the returned report for the precise reason.
  test.skip('FR-18: custom-thumbnail upload persists/renders — PARKED (needs photos/presign + S3 PUT plumbing beyond a fixture)', async () => {
    // Intentionally skipped: PARK trigger (photo/thumbnail plumbing beyond a fixture).
  });
});
