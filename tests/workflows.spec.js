const { test, expect } = require('@playwright/test');

// ─── Navigation & Layout ────────────────────────────────────────────────────

test.describe('Navigation', () => {
  test('HQ launcher has Operations tile linking to workflows', async ({ page }) => {
    await page.goto('/index.html');
    const tile = page.locator('a[href="workflows.html"]');
    await expect(tile).toBeVisible();
    await expect(tile).toContainText('Operations');
  });

  test('workflows page has three tabs', async ({ page }) => {
    await page.goto('/workflows.html');
    await expect(page.locator('#t1')).toContainText('My Checklists');
    await expect(page.locator('#t2')).toContainText('Approvals');
    await expect(page.locator('#t3')).toContainText('Builder');
  });

  test('My Checklists is the default active tab', async ({ page }) => {
    await page.goto('/workflows.html');
    await expect(page.locator('#t1')).toHaveClass(/on/);
    await expect(page.locator('#s1')).toBeVisible();
    await expect(page.locator('#s2')).toBeHidden();
    await expect(page.locator('#s3')).toBeHidden();
  });

  test('tab switching works', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#s1')).toBeHidden();
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s3')).toBeHidden();
  });
});

// ─── My Checklists (Fill-Out) ───────────────────────────────────────────────

test.describe('My Checklists', () => {
  test('shows today\'s checklists with progress bars', async ({ page }) => {
    await page.goto('/workflows.html');
    await expect(page.locator('[data-fill-template-id="tpl_setup"]')).toBeVisible();
    await expect(page.locator('[data-fill-template-id="tpl_closing"]')).toBeVisible();
    // Aggregate progress text
    await expect(page.locator('text=items complete across')).toBeVisible();
  });

  test('opening a checklist shows runner with progress', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    await expect(page.locator('#fill-back')).toBeVisible();
    await expect(page.locator('.progress-line')).toBeVisible();
  });

  test('back button returns to checklist list', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    await page.click('#fill-back');
    await expect(page.locator('[data-fill-template-id="tpl_closing"]')).toBeVisible();
  });
});

// ─── Checkbox Field ─────────────────────────────────────────────────────────

test.describe('Checkbox field', () => {
  test('toggling checkbox shows checkmark and attribution', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/);
    await expect(checkBtn).toContainText('✓');
    // Attribution should show
    await expect(page.locator('.fill-attribution').first()).toContainText('Jamal');
  });

  test('toggling checkbox again unchecks it', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    const checkBtn = page.locator('.check-btn').first();
    await checkBtn.click();
    await expect(checkBtn).toHaveClass(/checked/);
    await checkBtn.click();
    await expect(checkBtn).not.toHaveClass(/checked/);
  });
});

// ─── Yes/No Field ───────────────────────────────────────────────────────────

test.describe('Yes/No field', () => {
  test('tapping Yes highlights the pill', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    const yesPill = page.locator('[data-action="set-yes"]').first();
    await yesPill.click();
    await expect(yesPill).toHaveClass(/on/);
  });

  test('tapping No shows fail card', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    const noPill = page.locator('[data-action="set-no"]').first();
    await noPill.click();
    await expect(noPill).toHaveClass(/on/);
    await expect(page.locator('.fail-card').first()).toBeVisible();
  });

  test('switching from No to Yes removes fail card', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    await page.locator('[data-action="set-no"]').first().click();
    await expect(page.locator('.fail-card').first()).toBeVisible();
    await page.locator('[data-action="set-yes"]').first().click();
    await expect(page.locator('.fail-card')).toHaveCount(0);
  });
});

// ─── Temperature Field ──────────────────────────────────────────────────────

test.describe('Temperature field', () => {
  test('entering out-of-range temp shows warning and fail card', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    const tempInput = page.locator('.fill-temp-input').first();
    await tempInput.fill('200');
    await tempInput.dispatchEvent('input');
    // Wait for re-render
    await page.waitForTimeout(200);
    await expect(page.locator('.temp-warn').first()).toBeVisible();
    await expect(page.locator('.fail-card').first()).toBeVisible();
  });

  test('entering in-range temp shows no warning', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    const tempInput = page.locator('.fill-temp-input').first();
    await tempInput.fill('400');
    await tempInput.dispatchEvent('input');
    await page.waitForTimeout(200);
    await expect(page.locator('.temp-warn')).toHaveCount(0);
  });
});

// ─── Fail Card / Corrective Action ─────────────────────────────────────────

test.describe('Corrective action', () => {
  test('fail card has note textarea and severity pills', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    await page.locator('[data-action="set-no"]').first().click();
    const failCard = page.locator('.fail-card').first();
    await expect(failCard.locator('.fail-note-input')).toBeVisible();
    await expect(failCard.locator('.severity-pill')).toHaveCount(3);
  });

  test('severity pill toggles', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    await page.locator('[data-action="set-no"]').first().click();
    const majorPill = page.locator('.severity-pill', { hasText: 'Major' }).first();
    await majorPill.click();
    await expect(majorPill).toHaveClass(/on/);
  });
});

// ─── Progress Tracking ──────────────────────────────────────────────────────

// ─── Sub-Steps ──────────────────────────────────────────────────────────────

test.describe('Sub-steps', () => {
  test('checkbox with sub-steps shows sub-step items', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    // "Prep sauce for tomorrow" has 3 sub-steps
    await expect(page.locator('.sub-step-row')).toHaveCount(3);
    await expect(page.locator('.sub-step-label-text', { hasText: 'Add sugar' })).toBeVisible();
    await expect(page.locator('.sub-step-label-text', { hasText: 'Add ketchup' })).toBeVisible();
    await expect(page.locator('.sub-step-label-text', { hasText: 'Add soy sauce' })).toBeVisible();
  });

  test('completing all sub-steps auto-checks parent', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    // Parent should not be checked yet
    const parentBtn = page.locator('.check-btn[data-has-subs="true"]');
    await expect(parentBtn).not.toHaveClass(/checked/);
    // Check all 3 sub-steps
    const subChecks = page.locator('.sub-step-check');
    await subChecks.nth(0).click();
    await subChecks.nth(1).click();
    await subChecks.nth(2).click();
    // Parent should now be checked
    await expect(page.locator('.check-btn[data-has-subs="true"]')).toHaveClass(/checked/);
  });

  test('unchecking parent clears all sub-steps', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    // Complete all sub-steps
    const subChecks = page.locator('.sub-step-check');
    await subChecks.nth(0).click();
    await subChecks.nth(1).click();
    await subChecks.nth(2).click();
    // Uncheck parent
    await page.locator('.check-btn[data-has-subs="true"]').click();
    // All sub-steps should be unchecked
    await expect(page.locator('.sub-step-check.done')).toHaveCount(0);
  });
});

// ─── Progress Tracking ──────────────────────────────────────────────────────

test.describe('Progress', () => {
  test('progress counter updates as items are completed', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    await expect(page.locator('.progress-line')).toContainText('0 of');
    await page.locator('.check-btn').first().click();
    await expect(page.locator('.progress-line')).toContainText('1 of');
  });
});

// ─── Submit Flow ────────────────────────────────────────────────────────────

test.describe('Submit', () => {
  test('submitting approval-required checklist shows approval toast and pending badge', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    // Complete all fields quickly
    const yesButtons = page.locator('[data-action="set-yes"]');
    const count = await yesButtons.count();
    for (let i = 0; i < count; i++) {
      await yesButtons.nth(i).click();
    }
    // Fill temp if visible
    const tempInputs = page.locator('.fill-temp-input');
    const tempCount = await tempInputs.count();
    for (let i = 0; i < tempCount; i++) {
      await tempInputs.nth(i).fill('400');
      await tempInputs.nth(i).dispatchEvent('input');
    }
    // Check all checkboxes
    const checkBtns = page.locator('.check-btn:not(.checked)');
    const checkCount = await checkBtns.count();
    for (let i = 0; i < checkCount; i++) {
      await checkBtns.nth(i).click();
    }
    // Submit
    await page.click('[data-action="submit"]');
    // Should show toast
    await expect(page.locator('#toast')).toContainText('approval');
    // Back on list, should show pending badge
    await expect(page.locator('.approval-badge')).toBeVisible();
  });

  test('submitting non-approval checklist navigates back to list', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    // Complete non-photo fields
    const checkBtns = page.locator('.check-btn');
    const count = await checkBtns.count();
    for (let i = 0; i < count; i++) {
      await checkBtns.nth(i).click();
    }
    const yesButtons = page.locator('[data-action="set-yes"]');
    const yesCount = await yesButtons.count();
    for (let i = 0; i < yesCount; i++) {
      await yesButtons.nth(i).click();
    }
    // Submit (photo field not completed, so no fireworks — but submit still works)
    await page.click('[data-action="submit"]');
    // Should navigate back to checklist list
    await page.waitForTimeout(3000);
    await expect(page.locator('[data-fill-template-id]').first()).toBeVisible();
    await expect(page.locator('#toast')).toContainText('submitted');
  });
});

// ─── Approval Flow ──────────────────────────────────────────────────────────

test.describe('Approval flow', () => {
  async function submitSetupForApproval(page) {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    const yesButtons = page.locator('[data-action="set-yes"]');
    const count = await yesButtons.count();
    for (let i = 0; i < count; i++) await yesButtons.nth(i).click();
    const tempInputs = page.locator('.fill-temp-input');
    const tempCount = await tempInputs.count();
    for (let i = 0; i < tempCount; i++) {
      await tempInputs.nth(i).fill('400');
      await tempInputs.nth(i).dispatchEvent('input');
    }
    const checkBtns = page.locator('.check-btn:not(.checked)');
    const checkCount = await checkBtns.count();
    for (let i = 0; i < checkCount; i++) await checkBtns.nth(i).click();
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(500);
  }

  test('approvals tab shows pending submission with item list', async ({ page }) => {
    await submitSetupForApproval(page);
    await page.click('#t2');
    await expect(page.locator('.approval-card')).toBeVisible();
    await expect(page.locator('.review-item')).toHaveCount.above;
    // At least one review item visible
    const items = page.locator('.review-item');
    expect(await items.count()).toBeGreaterThan(0);
  });

  test('approve shows approved status with unapprove option', async ({ page }) => {
    await submitSetupForApproval(page);
    await page.click('#t2');
    await page.click('[data-action="approve"]');
    // Photo is incomplete, so comment is required
    if (await page.locator('.approve-reason').isVisible()) {
      await page.locator('.approve-reason').fill('Photo not needed');
      await page.click('[data-action="approve-confirm"]');
    }
    await expect(page.locator('#toast')).toContainText('Approved');
    // Card moves to APPROVED section with green badge and unapprove button
    await expect(page.locator('#approvals-body').getByText('Approved ✓')).toBeVisible();
    await expect(page.locator('[data-action="unapprove"]')).toBeVisible();
  });

  test('unapprove requires reason and returns to pending', async ({ page }) => {
    await submitSetupForApproval(page);
    await page.click('#t2');
    await page.click('[data-action="approve"]');
    // Handle incomplete approval comment if needed
    if (await page.locator('.approve-reason').isVisible()) {
      await page.locator('.approve-reason').fill('Photo not needed');
      await page.click('[data-action="approve-confirm"]');
    }
    await page.waitForTimeout(300);
    // Tap unapprove
    await page.click('[data-action="unapprove"]');
    await expect(page.locator('.unapprove-reason')).toBeVisible();
    // Empty confirm does nothing
    await page.click('[data-action="unapprove-confirm"]');
    await expect(page.locator('.unapprove-reason')).toBeVisible();
    // Add reason and confirm
    await page.locator('.unapprove-reason').fill('Approved by accident');
    await page.click('[data-action="unapprove-confirm"]');
    await expect(page.locator('#toast')).toContainText('Unapproved');
    // Card should be back in PENDING section with approve/reject buttons
    await expect(page.locator('[data-action="approve"]')).toBeVisible();
  });

  test('reject requires flagging at least one item', async ({ page }) => {
    await submitSetupForApproval(page);
    await page.click('#t2');
    await page.click('[data-action="reject-submit"]');
    await expect(page.locator('#toast')).toContainText('Flag at least one item');
  });

  test('full reject flow: flag item, add comment, confirm', async ({ page }) => {
    await submitSetupForApproval(page);
    await page.click('#t2');
    // Flag first item
    await page.locator('[data-action="toggle-reject-item"]').first().click();
    await expect(page.locator('.reject-item-form')).toBeVisible();
    // Add comment
    await page.locator('.reject-item-input').first().fill('Please redo this item');
    // Submit rejection
    await page.click('[data-action="reject-submit"]');
    await expect(page.locator('#toast')).toContainText('Rejected');
    // Approval card should be gone
    await expect(page.locator('.approval-card')).toHaveCount(0);
  });

  test('rejected item shows correction banner on checklist', async ({ page }) => {
    await submitSetupForApproval(page);
    await page.click('#t2');
    await page.locator('[data-action="toggle-reject-item"]').first().click();
    await page.locator('.reject-item-input').first().fill('Fix this');
    await page.click('[data-action="reject-submit"]');
    // Go back to My Checklists
    await page.click('#t1');
    await page.click('[data-fill-template-id="tpl_setup"]');
    await expect(page.locator('.correction-banner').first()).toContainText('Rejected');
  });
});

// ─── Incomplete Submission Approval ──────────────────────────────────────────

test.describe('Incomplete submission approval', () => {
  async function submitSetupPartial(page) {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    // Only complete one field, leave others incomplete
    const yesButtons = page.locator('[data-action="set-yes"]');
    if (await yesButtons.count() > 0) await yesButtons.first().click();
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(500);
  }

  test('incomplete items show red X in approvals tab', async ({ page }) => {
    await submitSetupPartial(page);
    await page.click('#t2');
    // At least one red ✗ should be visible (incomplete items)
    const redX = page.locator('.review-check', { hasText: '✗' });
    expect(await redX.count()).toBeGreaterThan(0);
    // At least one green ✓ (completed item)
    const greenCheck = page.locator('.review-check', { hasText: '✓' });
    expect(await greenCheck.count()).toBeGreaterThan(0);
  });

  test('approving with incomplete items requires a comment', async ({ page }) => {
    await submitSetupPartial(page);
    await page.click('#t2');
    await page.click('[data-action="approve"]');
    // Should show comment textarea, not approve immediately
    await expect(page.locator('.approve-reason')).toBeVisible();
    // Empty confirm does nothing
    await page.click('[data-action="approve-confirm"]');
    await expect(page.locator('.approve-reason')).toBeVisible();
    // Add reason and confirm
    await page.locator('.approve-reason').fill('Photo not needed today — indoor event');
    await page.click('[data-action="approve-confirm"]');
    await expect(page.locator('#toast')).toContainText('Approved');
  });

  test('approving with incomplete items shows comment form, completing it approves', async ({ page }) => {
    await submitSetupPartial(page);
    await page.click('#t2');
    await page.click('[data-action="approve"]');
    // Comment form should be visible (incomplete items)
    await expect(page.locator('.approve-reason')).toBeVisible();
    await page.locator('.approve-reason').fill('Acceptable for today');
    await page.click('[data-action="approve-confirm"]');
    await expect(page.locator('#toast')).toContainText('Approved');
    await expect(page.locator('#approvals-body').getByText('Approved ✓')).toBeVisible();
  });
});

// ─── Unsubmit ───────────────────────────────────────────────────────────────

test.describe('Unsubmit', () => {
  test('unsubmit button appears as red button after submitting', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    await page.locator('[data-action="set-yes"]').first().click();
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(500);
    await page.click('[data-fill-template-id="tpl_setup"]');
    const unsubmitBtn = page.locator('[data-action="unsubmit"]');
    await expect(unsubmitBtn).toBeVisible();
    await expect(unsubmitBtn).toContainText('Unsubmit');
  });

  test('unsubmit returns checklist to editable state', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_setup"]');
    await page.locator('[data-action="set-yes"]').first().click();
    await page.click('[data-action="submit"]');
    await page.waitForTimeout(500);
    await page.click('[data-fill-template-id="tpl_setup"]');
    await page.click('[data-action="unsubmit"]');
    await expect(page.locator('#toast')).toContainText('unsubmitted');
    // Should show submit button again (not unsubmit)
    await expect(page.locator('[data-action="submit"]')).toBeVisible();
  });
});

// ─── Builder Tab ────────────────────────────────────────────────────────────

test.describe('Builder', () => {
  test('builder shows checklist list', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('#t3');
    await expect(page.locator('#builder-body')).toBeVisible();
    await expect(page.locator('#builder-body')).toContainText('Setup Checklist');
  });

  test('can create a new checklist', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('#t3');
    await page.click('text=+ New checklist');
    // Should be in editor view with name input
    await expect(page.locator('#tpl-name-input')).toBeVisible();
  });

  test('can add a section', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('#t3');
    await page.click('text=+ New checklist');
    // Use dialog for section name
    page.on('dialog', dialog => dialog.accept('Test Section'));
    await page.click('text=+ Add section');
    await expect(page.locator('#builder-body')).toContainText('Test Section');
  });
});

// ─── URL Hash Persistence ───────────────────────────────────────────────────

test.describe('URL hash persistence', () => {
  test('opening a checklist sets hash', async ({ page }) => {
    await page.goto('/workflows.html');
    await page.click('[data-fill-template-id="tpl_closing"]');
    expect(page.url()).toContain('#checklist=tpl_closing');
  });

  test('navigating with hash opens the checklist directly', async ({ page }) => {
    await page.goto('/workflows.html#checklist=tpl_closing');
    await expect(page.locator('#fill-back')).toBeVisible();
    await expect(page.locator('.progress-line')).toBeVisible();
  });

  test('back button clears hash', async ({ page }) => {
    await page.goto('/workflows.html#checklist=tpl_closing');
    await page.click('#fill-back');
    expect(page.url()).not.toContain('#checklist');
  });
});
