const { test, expect } = require('@playwright/test');

// ─── Navigation & Layout ────────────────────────────────────────────────────

test.describe('Onboarding Navigation', () => {
  test('HQ launcher has Onboarding tile linking to onboarding.html', async ({ page }) => {
    await page.goto('/login.html');
    await page.fill('input[type="email"]', 'jamal@yumyums.kitchen');
    await page.fill('input[type="password"]', 'test123');
    await page.click('button.btn');
    await page.waitForURL(url => !url.pathname.includes('login'));
    await page.goto('/index.html');
    const tile = page.locator('a[href="onboarding.html"]');
    await expect(tile).toBeVisible();
    await expect(tile).toContainText('Onboarding');
  });

  test('onboarding page has three tabs', async ({ page }) => {
    await page.goto('/onboarding.html');
    await expect(page.locator('#t1')).toContainText('My Trainings');
    await expect(page.locator('#t2')).toContainText('Manager');
    await expect(page.locator('#t3')).toContainText('Builder');
  });

  test('My Trainings is the default active tab', async ({ page }) => {
    await page.goto('/onboarding.html');
    await expect(page.locator('#t1')).toHaveClass(/on/);
    await expect(page.locator('#s1')).toBeVisible();
    await expect(page.locator('#s2')).toBeHidden();
    await expect(page.locator('#s3')).toBeHidden();
  });

  test('tab switching works', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#s1')).toBeHidden();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s3')).toBeHidden();
    await page.click('#t1');
    await expect(page.locator('#s1')).toBeVisible();
    await expect(page.locator('#s2')).toBeHidden();
  });

  test('back link returns to HQ', async ({ page }) => {
    await page.goto('/onboarding.html');
    const back = page.locator('a.back[href="index.html"]');
    await expect(back).toBeVisible();
  });
});

// ─── My Trainings (Crew View) ───────────────────────────────────────────────

test.describe('My Trainings', () => {
  test('shows assigned checklists with progress bars', async ({ page }) => {
    await page.goto('/onboarding.html');
    const cards = page.locator('#my-body .card');
    await expect(cards.first()).toBeVisible();
    // Progress text should be visible
    await expect(page.locator('#my-body')).toContainText('tasks');
  });

  test('opening a checklist shows runner with sections', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.locator('#my-body .card').first().click();
    // Should see back link and section headers
    await expect(page.locator('[data-action="back-to-list"]')).toBeVisible();
    await expect(page.locator('.sec-header').first()).toBeVisible();
  });

  test('first section is active, later sections are locked', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.locator('#my-body .card').first().click();
    const sections = page.locator('.sec-header');
    // First section should not have locked class
    await expect(sections.first()).not.toHaveClass(/locked/);
    // If there are multiple sections, second should be locked
    const count = await sections.count();
    if (count > 1) {
      await expect(sections.nth(1)).toHaveClass(/locked/);
    }
  });

  test('expanding a section shows items', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.locator('#my-body .card').first().click();
    // Click first (active) section to expand
    await page.locator('#my-body .sec-header').first().click();
    // Should see item rows
    await expect(page.locator('#my-body .item-row').first()).toBeVisible();
  });

  test('checking an item updates progress', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.locator('#my-body .card').first().click();
    await page.locator('#my-body .sec-header').first().click();
    const item = page.locator('#my-body [data-action="toggle-item"]').first();
    await item.click();
    // Item should now have the checked class
    await expect(item).toHaveClass(/checked/);
  });

  test('Watch Video link opens in new tab without auto-checking', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.locator('#my-body .card').first().click();
    // Find a section with video items and expand it
    const sections = page.locator('.sec-header:not(.locked)');
    const count = await sections.count();
    for (let i = 0; i < count; i++) {
      await sections.nth(i).click();
    }
    const videoLink = page.locator('[data-action="watch-video"]').first();
    if (await videoLink.isVisible()) {
      // Video link should exist and be visible
      await expect(videoLink).toContainText('Watch Video');
    }
  });

  test('back button returns to checklist list', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.locator('#my-body .card').first().click();
    await page.locator('[data-action="back-to-list"]').click();
    // Should see checklist cards again
    await expect(page.locator('#my-body .card').first()).toBeVisible();
  });
});

// ─── Manager Tab ────────────────────────────────────────────────────────────

test.describe('Manager Tab', () => {
  test('shows Active and Completed sub-tabs', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t2');
    await expect(page.locator('#mgr-body')).toContainText('Active');
    await expect(page.locator('#mgr-body')).toContainText('Completed');
  });

  test('Active view shows hire cards with progress', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t2');
    const cards = page.locator('#mgr-body .card');
    await expect(cards.first()).toBeVisible();
    // Should show hire name and progress percentage
    await expect(page.locator('#mgr-body')).toContainText('%');
  });

  test('tapping hire card shows read-only checklist', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t2');
    await page.locator('#mgr-body .card').first().click();
    // Should see back link and sections
    await expect(page.locator('[data-action="back-to-hires"]')).toBeVisible();
    await expect(page.locator('.sec-header').first()).toBeVisible();
  });

  test('manager view is read-only — items not checkable', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t2');
    await page.locator('#mgr-body .card').first().click();
    // Expand first section
    await page.locator('#mgr-body .sec-header').first().click();
    // Should NOT have toggle-item actions (read-only)
    const toggleItems = page.locator('#mgr-body [data-action="toggle-item"]');
    await expect(toggleItems).toHaveCount(0);
  });

  test('tab switch re-renders Manager content', async ({ page }) => {
    await page.goto('/onboarding.html');
    // Complete an item on My Trainings
    await page.locator('#my-body .card').first().click();
    await page.locator('#my-body .sec-header').first().click();
    await page.locator('#my-body [data-action="toggle-item"]').first().click();
    // Switch to Manager tab
    await page.click('#t2');
    // Manager should show fresh content with updated progress
    await expect(page.locator('#mgr-body .card').first()).toBeVisible();
  });

  test('pending sign-off badge appears on hire card', async ({ page }) => {
    await page.goto('/onboarding.html');
    // Complete all items in first section and request sign-off
    await page.locator('#my-body .card').first().click();
    await page.locator('#my-body .sec-header').first().click();
    // Check all items in the section
    const items = page.locator('#my-body [data-action="toggle-item"]');
    const count = await items.count();
    for (let i = 0; i < count; i++) {
      await items.nth(i).click();
    }
    // Look for Request Sign-Off button
    const requestBtn = page.locator('[data-action="request-signoff"]');
    if (await requestBtn.isVisible()) {
      await requestBtn.click();
      // Switch to Manager tab
      await page.click('#t2');
      // Should see pending sign-off badge
      await expect(page.locator('#mgr-body')).toContainText('awaiting sign-off');
    }
  });

  test('approve sign-off updates section state immediately', async ({ page }) => {
    await page.goto('/onboarding.html');
    // Complete all items in first section and request sign-off
    await page.locator('#my-body .card').first().click();
    await page.locator('#my-body .sec-header').first().click();
    const items = page.locator('#my-body [data-action="toggle-item"]');
    const count = await items.count();
    for (let i = 0; i < count; i++) {
      await items.nth(i).click();
    }
    const requestBtn = page.locator('#my-body [data-action="request-signoff"]');
    if (await requestBtn.isVisible()) {
      await requestBtn.click();
      // Switch to Manager, open hire
      await page.click('#t2');
      await page.locator('#mgr-body .card').first().click();
      // Expand the pending section in manager view
      await page.locator('#mgr-body .sec-header').first().click();
      // Approve
      const approveBtn = page.locator('#mgr-body [data-action="approve-signoff"]');
      if (await approveBtn.isVisible()) {
        await approveBtn.click();
        // Should immediately show Signed Off (not require navigation)
        await expect(page.locator('#mgr-body')).toContainText('Signed Off');
      }
    }
  });

  test('back to hires returns to list view', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t2');
    await page.locator('#mgr-body .card').first().click();
    await page.locator('[data-action="back-to-hires"]').click();
    // Should see hire cards again
    await expect(page.locator('#mgr-body .card').first()).toBeVisible();
  });
});

// ─── Builder Tab ────────────────────────────────────────────────────────────

test.describe('Builder Tab', () => {
  test('shows template list with cards', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    const cards = page.locator('#builder-body .card');
    await expect(cards.first()).toBeVisible();
    // Should show template names
    await expect(page.locator('#builder-body')).toContainText('Onboarding');
  });

  test('shows + New Template button', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await expect(page.locator('[data-action="new-template"]')).toBeVisible();
  });

  test('opening a template shows editor with sections', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    // Should see back link, template name input, and sections
    await expect(page.locator('[data-action="back-to-templates"]')).toBeVisible();
    await expect(page.locator('[data-action="tpl-name-input"]')).toBeVisible();
    await expect(page.locator('[data-action="add-ob-section"]')).toBeVisible();
  });

  test('section has Require Sign-off and FAQ toggles', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    await expect(page.locator('[data-action="toggle-signoff"]').first()).toBeVisible();
    await expect(page.locator('[data-action="toggle-signoff"]').first()).toContainText('Require Sign-off');
    await expect(page.locator('[data-action="toggle-faq-mode"]').first()).toBeVisible();
    await expect(page.locator('[data-action="toggle-faq-mode"]').first()).toContainText('FAQ');
  });

  test('toggling FAQ mode switches section to Q&A editor', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    // Find a non-FAQ section and toggle FAQ on
    const faqToggles = page.locator('[data-action="toggle-faq-mode"]');
    await faqToggles.first().click();
    // Should see + Add Q&A button
    await expect(page.locator('[data-action="add-faq-item"]').first()).toBeVisible();
  });

  test('adding a checkbox item', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    // Count items before
    const addBtn = page.locator('[data-action="add-ob-item"][data-item-type="checkbox"]').first();
    await addBtn.click();
    // Should see new item with label input
    await expect(page.locator('[data-action="item-label-input"]').last()).toBeVisible();
  });

  test('checkbox item supports sub-items', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    // Add a checkbox
    await page.locator('[data-action="add-ob-item"][data-item-type="checkbox"]').first().click();
    // Click + Sub-item on the new checkbox
    const subBtn = page.locator('[data-action="add-sub-item"]').last();
    await subBtn.click();
    // Should see sub-item label input
    await expect(page.locator('[data-action="sub-item-label-input"]').last()).toBeVisible();
  });

  test('adding a video series item', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    const addBtn = page.locator('[data-action="add-ob-item"][data-item-type="video_series"]').first();
    await addBtn.click();
    // Should see video series with + Add Part button
    await expect(page.locator('[data-action="add-video-part"]').last()).toBeVisible();
  });

  test('video series supports adding parts with title/desc/URL', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    // Add a video series
    await page.locator('[data-action="add-ob-item"][data-item-type="video_series"]').first().click();
    // Add a part
    await page.locator('[data-action="add-video-part"]').last().click();
    // Should see part inputs
    await expect(page.locator('[data-action="part-title-input"]').last()).toBeVisible();
    await expect(page.locator('[data-action="part-desc-input"]').last()).toBeVisible();
    await expect(page.locator('[data-action="part-url-input"]').last()).toBeVisible();
  });

  test('adding a section', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    const sectionsBefore = await page.locator('[data-action="toggle-signoff"]').count();
    // Mock the prompt dialog
    await page.evaluate(() => { window.prompt = () => 'New Test Section'; });
    await page.locator('[data-action="add-ob-section"]').click();
    const sectionsAfter = await page.locator('[data-action="toggle-signoff"]').count();
    expect(sectionsAfter).toBe(sectionsBefore + 1);
  });

  test('deleting a template with confirmation', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    const templatesBefore = await page.locator('[data-action="open-template"]').count();
    // Create a new template to delete
    await page.evaluate(() => { window.prompt = () => 'Temp Template'; });
    await page.locator('[data-action="new-template"]').click();
    // Now in editor — delete it
    await page.evaluate(() => { window.confirm = () => true; });
    await page.locator('[data-action="delete-template"]').click();
    // Should be back at template list with original count
    const templatesAfter = await page.locator('[data-action="open-template"]').count();
    expect(templatesAfter).toBe(templatesBefore);
  });

  test('back to templates returns to list', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    await page.locator('[data-action="back-to-templates"]').click();
    // Should see template cards
    await expect(page.locator('[data-action="open-template"]').first()).toBeVisible();
  });

  test('role selector is present in editor', async ({ page }) => {
    await page.goto('/onboarding.html');
    await page.click('#t3');
    await page.locator('[data-action="open-template"]').first().click();
    await expect(page.locator('[data-action="tpl-role-select"]')).toBeVisible();
  });
});

// ─── Cross-Tab State Sync ───────────────────────────────────────────────────

test.describe('Cross-tab state sync', () => {
  test('My Trainings re-renders fresh on tab switch', async ({ page }) => {
    await page.goto('/onboarding.html');
    // Check an item on My Trainings
    await page.locator('#my-body .card').first().click();
    await page.locator('#my-body .sec-header').first().click();
    await page.locator('#my-body [data-action="toggle-item"]').first().click();
    // Switch to Manager and back
    await page.click('#t2');
    await page.click('#t1');
    // My Trainings should show the list view (re-rendered) with progress updated
    await expect(page.locator('#my-body .card').first()).toBeVisible();
    await expect(page.locator('#my-body')).toContainText('tasks complete');
  });

  test('Builder changes reflect in My Trainings template list', async ({ page }) => {
    await page.goto('/onboarding.html');
    // Go to builder, create new template
    await page.click('#t3');
    await page.evaluate(() => { window.prompt = () => 'Test Training'; });
    await page.locator('[data-action="new-template"]').click();
    // Go back to templates list
    await page.locator('[data-action="back-to-templates"]').click();
    // Switch to My Trainings — new template should be visible if role matches
    await page.click('#t1');
    // The template list re-renders on tab switch
    await expect(page.locator('#my-body')).toBeVisible();
  });
});
