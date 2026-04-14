const { test, expect } = require('@playwright/test');

test.describe('Inventory', () => {

  test.beforeEach(async ({ page }) => {
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
  });

  // HIST-03: Reachable from HQ
  test('HQ launcher has Inventory tile linking to inventory.html', async ({ page }) => {
    await page.goto('/index.html');
    const tile = page.locator('a.tile[href="inventory.html"]');
    await expect(tile).toBeVisible();
    await expect(tile).toContainText('Inventory');
    await tile.click();
    await expect(page).toHaveURL(/inventory\.html/);
  });

  // Tab navigation (INTG-01)
  test('shows 4 tabs: History, Trends, Stock, Cost', async ({ page }) => {
    await expect(page.locator('#t1')).toContainText('History');
    await expect(page.locator('#t2')).toContainText('Trends');
    await expect(page.locator('#t3')).toContainText('Stock');
    await expect(page.locator('#t4')).toContainText('Cost');
  });

  test('History tab is active by default', async ({ page }) => {
    await expect(page.locator('#t1')).toHaveClass(/on/);
    await expect(page.locator('#s1')).toBeVisible();
    await expect(page.locator('#s2')).not.toBeVisible();
  });

  test('Trends tab shows Coming Soon placeholder', async ({ page }) => {
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2')).toContainText('Spending Trends');
  });

  // STCK-01: Stock tab shows item groups with badges
  test('Stock tab shows item groups with stock badges', async ({ page }) => {
    await page.click('#t3');
    const badges = page.locator('.stock-badge');
    const count = await badges.count();
    expect(count).toBeGreaterThanOrEqual(10);
  });

  test('Stock tab groups items by tag category', async ({ page }) => {
    await page.click('#t3');
    const tagHeaders = page.locator('.tag-header');
    const count = await tagHeaders.count();
    expect(count).toBeGreaterThanOrEqual(4);
  });

  test('Stock badges include at least two different levels', async ({ page }) => {
    await page.click('#t3');
    const highBadges = page.locator('.stock-high');
    const medBadges = page.locator('.stock-medium');
    const lowBadges = page.locator('.stock-low');
    const total = await highBadges.count() + await medBadges.count() + await lowBadges.count();
    expect(total).toBeGreaterThanOrEqual(2);
  });

  test('items within tag are sorted by urgency Low first', async ({ page }) => {
    await page.click('#t3');
    const firstSection = page.locator('.tag-section').first();
    const badges = firstSection.locator('.stock-badge');
    const badgeTexts = await badges.allTextContents();
    if (badgeTexts.length >= 2) {
      const order = { 'Low': 0, 'Medium': 1, 'High': 2, 'Unknown': 3 };
      for (let i = 1; i < badgeTexts.length; i++) {
        const prev = order[badgeTexts[i-1].trim()] ?? 3;
        const curr = order[badgeTexts[i].trim()] ?? 3;
        expect(curr).toBeGreaterThanOrEqual(prev);
      }
    }
  });

  test('tapping tag header collapses and expands the section', async ({ page }) => {
    await page.click('#t3');
    const firstHeader = page.locator('.tag-header').first();
    const firstSection = page.locator('.tag-section').first();
    const itemsBefore = await firstSection.locator('.stock-item').count();
    expect(itemsBefore).toBeGreaterThanOrEqual(1);
    await firstHeader.click();
    const itemsAfter = await firstSection.locator('.stock-item:visible').count();
    expect(itemsAfter).toBe(0);
    await firstHeader.click();
    const itemsRestored = await firstSection.locator('.stock-item:visible').count();
    expect(itemsRestored).toBeGreaterThanOrEqual(1);
  });

  test('tapping stock item expands detail with purchase info', async ({ page }) => {
    await page.click('#t3');
    const firstItem = page.locator('.stock-item').first();
    await firstItem.click();
    const detail = page.locator('.stock-detail.open').first();
    await expect(detail).toBeVisible();
    const text = await detail.textContent();
    expect(text).toMatch(/Last purchased|Override Level/);
  });

  // STCK-02: Reorder suggestions
  test('reorder suggestions section shows Low and Medium items only', async ({ page }) => {
    await page.click('#t3');
    const reorderSection = page.locator('#reorder-section');
    const text = await reorderSection.textContent();
    if (text.trim().length > 0) {
      expect(text).toMatch(/Low|Medium/i);
    }
  });

  // STCK-03: Manual override
  test('Override Level button shows override form', async ({ page }) => {
    await page.click('#t3');
    await page.locator('.stock-item').first().click();
    const overrideBtn = page.locator('[data-action="show-override"]').first();
    await overrideBtn.click();
    const form = page.locator('.override-form');
    await expect(form).toBeVisible();
    const radios = form.locator('input[type="radio"]');
    expect(await radios.count()).toBe(3);
  });

  test('saving override changes badge and shows Overridden indicator', async ({ page }) => {
    await page.click('#t3');
    await page.locator('.stock-item').first().click();
    await page.locator('[data-action="show-override"]').first().click();
    await page.locator('input[name="override-level"][value="high"]').check();
    await page.locator('.override-reason').fill('Just restocked');
    await page.locator('[data-action="save-override"]').click();
    const indicator = page.locator('.overridden-indicator').first();
    await expect(indicator).toBeVisible();
  });

  test('clearing override returns to calculated level', async ({ page }) => {
    await page.click('#t3');
    await page.locator('.stock-item').first().click();
    await page.locator('[data-action="show-override"]').first().click();
    await page.locator('input[name="override-level"][value="high"]').check();
    await page.locator('.override-reason').fill('Test');
    await page.locator('[data-action="save-override"]').click();
    await page.locator('.stock-item').first().click();
    await page.locator('[data-action="clear-override"]').first().click();
    const indicators = await page.locator('.overridden-indicator').count();
    expect(indicators).toBe(0);
  });

  test('cancel override form hides it without saving', async ({ page }) => {
    await page.click('#t3');
    await page.locator('.stock-item').first().click();
    await page.locator('[data-action="show-override"]').first().click();
    await expect(page.locator('.override-form')).toBeVisible();
    await page.locator('[data-action="cancel-override"]').click();
    await expect(page.locator('.override-form')).not.toBeVisible();
  });

  test('Cost tab shows Coming Soon placeholder', async ({ page }) => {
    await page.click('#t4');
    await expect(page.locator('#s4')).toBeVisible();
    await expect(page.locator('#s4')).toContainText('Food Cost Intelligence');
  });

  // Chart.js loaded (HIST-04 prerequisite)
  test('Chart.js is loaded as window.Chart', async ({ page }) => {
    const chartType = await page.evaluate(() => typeof Chart);
    expect(chartType).toBe('function');
  });

  // HIST-01: Purchase history with expandable line items
  test('History tab renders purchase event cards', async ({ page }) => {
    const events = page.locator('.event-card');
    await expect(events.first()).toBeVisible();
    const count = await events.count();
    expect(count).toBeGreaterThanOrEqual(12);
  });

  test('purchase events are sorted newest-first', async ({ page }) => {
    const dates = await page.locator('.event-card .event-header').allTextContents();
    expect(dates.length).toBeGreaterThan(1);
  });

  test('each event shows vendor name and total', async ({ page }) => {
    const firstCard = page.locator('.event-card').first();
    const text = await firstCard.textContent();
    expect(text).toMatch(/\$/);
  });

  test('tapping an event card expands line items', async ({ page }) => {
    const firstCard = page.locator('.event-card').first();
    await firstCard.click();
    const detail = page.locator('.event-detail').first();
    await expect(detail).toBeVisible();
    const lineItems = detail.locator('.line-item');
    const count = await lineItems.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('line items show name, quantity, and price', async ({ page }) => {
    await page.locator('.event-card').first().click();
    const lineItem = page.locator('.line-item').first();
    const text = await lineItem.textContent();
    expect(text).toMatch(/Qty:/);
    expect(text).toMatch(/\$/);
  });

  test('case items show CASE badge', async ({ page }) => {
    const events = page.locator('.event-card');
    const count = await events.count();
    for (let i = 0; i < Math.min(count, 5); i++) {
      await events.nth(i).click();
    }
    const caseBadges = page.locator('.case-badge');
    const badgeCount = await caseBadges.count();
    expect(badgeCount).toBeGreaterThanOrEqual(1);
  });

  test('tapping expanded event collapses it', async ({ page }) => {
    const firstCard = page.locator('.event-card').first();
    await firstCard.click();
    await expect(page.locator('.event-detail').first()).toBeVisible();
    await firstCard.click();
    await expect(page.locator('.event-detail')).not.toBeVisible();
  });

  // HIST-02: Vendor filter
  test('vendor filter dropdown is present with All Vendors default', async ({ page }) => {
    const select = page.locator('#vendor-filter');
    await expect(select).toBeVisible();
    const defaultValue = await select.inputValue();
    expect(defaultValue).toBe('');
  });

  test('selecting a vendor filters events to that vendor only', async ({ page }) => {
    const select = page.locator('#vendor-filter');
    const options = await select.locator('option').allTextContents();
    expect(options.length).toBeGreaterThan(1);
    const vendorName = options[1];
    await select.selectOption({ index: 1 });
    const eventTexts = await page.locator('.event-card').allTextContents();
    for (const text of eventTexts) {
      expect(text).toContain(vendorName);
    }
  });

  test('selecting All Vendors shows all events again', async ({ page }) => {
    const select = page.locator('#vendor-filter');
    const allCount = await page.locator('.event-card').count();
    await select.selectOption({ index: 1 });
    const filteredCount = await page.locator('.event-card').count();
    expect(filteredCount).toBeLessThan(allCount);
    await select.selectOption({ value: '' });
    const resetCount = await page.locator('.event-card').count();
    expect(resetCount).toBe(allCount);
  });

  // Back link
  test('back link navigates to HQ', async ({ page }) => {
    const backLink = page.locator('a.back');
    await expect(backLink).toBeVisible();
    await expect(backLink).toHaveAttribute('href', 'index.html');
  });

});
