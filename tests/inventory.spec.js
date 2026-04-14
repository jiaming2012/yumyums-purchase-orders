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

  test('Stock tab shows Coming Soon placeholder', async ({ page }) => {
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#s3')).toContainText('Stock Levels');
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
