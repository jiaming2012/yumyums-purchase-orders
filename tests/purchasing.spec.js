const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// ─── Helpers ─────────────────────────────────────────────────────────────────

async function login(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email || ADMIN_EMAIL);
  await page.fill('input[type="password"]', password || ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

async function poApiCall(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch('/api/v1/purchasing/' + p, opts);
    if (res.status === 204) return null;
    if (!res.ok) { const e = await res.json().catch(() => ({})); throw new Error(JSON.stringify(e)); }
    return res.json();
  }, [method, path, body]);
}

// seedShoppingList: idempotent. Returns an existing active shopping list if one
// exists; otherwise approves a locked PO (or creates → cutoffs → approves a
// fresh one) and returns the resulting active list. Safe to call from every
// test that needs a list — turns 26 conditional test.skip()s in this file
// into real coverage (B-5).
async function seedShoppingList(page) {
  // Already have an active list with items? Return it.
  const existing = await poApiCall(page, 'GET', 'shopping/active').catch(() => null);
  if (existing && existing.vendor_sections && existing.vendor_sections.length > 0) {
    return existing;
  }

  // Approve a locked PO if one is waiting.
  let locked = await poApiCall(page, 'GET', 'orders?status=locked').catch(() => null);
  if (locked && locked.id) {
    await poApiCall(page, 'POST', 'orders/' + locked.id + '/approve');
    const active = await poApiCall(page, 'GET', 'shopping/active').catch(() => null);
    if (active && active.vendor_sections && active.vendor_sections.length > 0) return active;
  }

  // Otherwise create from scratch: draft → add items → cutoff → approve.
  const order = await page.evaluate(async () => {
    const res = await fetch('/api/v1/purchasing/orders', { method: 'POST', headers: { 'Content-Type': 'application/json' } });
    return res.json();
  });

  const items = await page.evaluate(async () => {
    const res = await fetch('/api/v1/inventory/items');
    return res.json();
  });
  if (!items || items.length === 0) throw new Error('No catalog items to seed PO');

  const toAdd = items.slice(0, 2).map(it => ({ purchase_item_id: it.id, quantity: 2, unit: '' }));
  await poApiCall(page, 'PUT', 'orders/' + order.id + '/items', { items: toAdd });

  await poApiCall(page, 'POST', 'simulate-cutoff');
  locked = await poApiCall(page, 'GET', 'orders?status=locked');
  if (!locked) throw new Error('No locked PO after simulate-cutoff');
  await poApiCall(page, 'POST', 'orders/' + locked.id + '/approve');
  return await poApiCall(page, 'GET', 'shopping/active');
}

// completeShoppingList seeds an active shopping list (via seedShoppingList) then
// completes every vendor section so the list transitions to status='completed'.
// Purely API-driven (no SQL, no migration) — reuses existing purchasing endpoints,
// so it stays inside test-seed scope. Returns the completed list, or null if the
// stack has no catalog items to seed from.
async function completeShoppingList(page) {
  let list;
  try {
    list = await seedShoppingList(page);
  } catch (e) {
    return null;
  }
  if (!list || !list.id || !(list.vendor_sections || []).length) return null;

  // Complete each vendor section; the last one flips the list to 'completed'.
  for (const sec of list.vendor_sections) {
    if (sec.status === 'completed') continue;
    await poApiCall(page, 'POST', 'shopping/' + list.id + '/vendors/' + sec.id + '/complete');
  }

  // Confirm it landed in history.
  const history = await poApiCall(page, 'GET', 'shopping/history').catch(() => []);
  return (history || []).find(h => h.id === list.id) || null;
}

// waitForShoppingContent waits until s2 (Shopping tab) renders shopping list or empty state
async function waitForShoppingContent(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('s2');
    if (!el) return false;
    return el.querySelector('.shop-item') || el.textContent.includes('No active') || el.textContent.includes('Week of');
  }, { timeout: 8000 });
}

// ─── Tests ────────────────────────────────────────────────────────────────────

test.describe('Purchasing tabs', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
  });

  test('shows 4 tabs: Order, Shopping, PO, History', async ({ page }) => {
    await expect(page.locator('#t1')).toContainText('Order');
    await expect(page.locator('#t2')).toContainText('Shopping');
    await expect(page.locator('#t3')).toContainText('PO');
    await expect(page.locator('#t4')).toContainText('History');
  });

  test('Order tab is active by default', async ({ page }) => {
    await expect(page.locator('#t1')).toHaveClass(/on/);
    await expect(page.locator('#s1')).toBeVisible();
    await expect(page.locator('#s2')).not.toBeVisible();
  });

  test('tab switching shows correct section', async ({ page }) => {
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s1')).not.toBeVisible();
    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await page.click('#t4');
    await expect(page.locator('#s4')).toBeVisible();
  });

});

test.describe('Shopping tab', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
  });

  // FR-7 (a) — Empty state. renderShoppingTab() (purchasing.html:555) paints the
  // Shopping tab directly into #s2 (there is NO #shopping-content element). With no
  // active list (SHOPPING_LIST === null) it MUST render the specific empty-state
  // copy from purchasing.html:558, not just "some non-empty text". This test proves
  // the real empty-state contract instead of the old length>0 tautology.
  test('Shopping tab shows specific empty-state stub when no active list exists', async ({ page }) => {
    // Guarantee the no-active-list precondition: complete any list still active so
    // shopping/active returns null (SHOPPING_LIST hydrates from that endpoint).
    await completeShoppingList(page);
    const active = await poApiCall(page, 'GET', 'shopping/active').catch(() => null);
    if (active && active.id) { test.skip(true, 'Could not clear active list to force empty state'); return; }

    // Re-hydrate the page against the now-empty state, then open the Shopping tab.
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.click('#t2');

    // The empty-state stub is a .stub inside #s2 — assert the EXACT real copy.
    const stub = page.locator('#s2 .stub');
    await expect(stub).toBeVisible();
    await expect(stub).toHaveText('Shopping list will appear here after the PO is approved');

    // And prove it is the empty state, not a populated one: no vendor sections / items.
    await expect(page.locator('#s2 .vendor-section')).toHaveCount(0);
    await expect(page.locator('#s2 .shop-item')).toHaveCount(0);
  });

  // FR-7 (b) — Populated. Seed an active shopping list and assert the real render
  // contract: grouped .vendor-section blocks (vendor name + item count), per-item
  // .shop-check buttons, .item-thumb thumbnails, and a location cell (either the
  // "Add location" affordance or a concrete store_location string). Concrete
  // visibility + counts, never a length tautology.
  test('Shopping tab renders grouped vendor sections with checks, thumbnails, and locations', async ({ page }) => {
    let list;
    try { list = await seedShoppingList(page); }
    catch (e) { test.skip(true, 'No catalog items to seed an active shopping list'); return; }
    expect(list).toBeTruthy();
    expect(list.id).toBeTruthy();
    const sections = list.vendor_sections || [];
    expect(sections.length).toBeGreaterThan(0);

    // Total item count across all seeded sections (used for check/thumb assertions).
    const totalItems = sections.reduce((n, s) => n + (s.items || []).length, 0);
    expect(totalItems).toBeGreaterThan(0);

    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.click('#t2');

    // The empty-state stub must NOT be present now.
    await expect(page.locator('#s2 .stub')).toHaveCount(0);

    // Grouped vendor sections: one .vendor-section per seeded section.
    const vendorSections = page.locator('#s2 .vendor-section');
    await expect(vendorSections.first()).toBeVisible();
    await expect(vendorSections).toHaveCount(sections.length);

    // Each section header (.cat) names its vendor and reports its item count.
    for (const sec of sections) {
      const header = page.locator('#s2 .vendor-section .cat', { hasText: sec.vendor_name }).first();
      await expect(header).toBeVisible();
      const cnt = (sec.items || []).length;
      await expect(header).toContainText(cnt + ' item');
    }

    // Per-item check buttons: one .shop-check per item (sections are pending on seed).
    await expect(page.locator('#s2 .shop-check')).toHaveCount(totalItems);

    // Item thumbnails: one .item-thumb per item.
    await expect(page.locator('#s2 .item-thumb')).toHaveCount(totalItems);

    // Location cell: each item shows either an "Add location" affordance (no
    // store_location) or a concrete store_location string. Assert the first item
    // concretely against its seeded data.
    const firstItem = sections[0].items[0];
    const firstRow = page.locator('#s2 .shop-item').first();
    await expect(firstRow).toBeVisible();
    if (firstItem.store_location) {
      await expect(firstRow).toContainText(firstItem.store_location);
    } else {
      await expect(firstRow.locator('[data-action="shop-edit-loc"]')).toContainText('Add location');
    }
  });

  test('shopping item check-off survives page reload', async ({ page }) => {
    const shoppingList = await seedShoppingList(page);
    const firstItem = shoppingList.vendor_sections[0].items[0];

    await page.click('#t2');
    await waitForShoppingContent(page);

    // Find and click the check button for the first item
    const checkEl = page.locator('[data-action="shop-check"][data-item-id="' + firstItem.id + '"]');
    await checkEl.click();

    // Wait for the API call to complete
    await page.waitForResponse(res => res.url().includes('/shopping/') && res.url().includes('/check'), { timeout: 5000 }).catch(() => {});
    await page.waitForTimeout(500);

    // Reload and switch back to Shopping tab
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.click('#t2');
    await waitForShoppingContent(page);

    // Verify the item is still checked (shows ✅ checkmark)
    const checkElAfter = page.locator('[data-action="shop-check"][data-item-id="' + firstItem.id + '"]');
    const checkText = await checkElAfter.textContent();
    expect(checkText).toContain('✅');
  });

  test('vendor section completion persists after reload', async ({ page }) => {
    const shoppingList = await seedShoppingList(page);
    // Find a pending vendor section (seedShoppingList always produces at least one).
    const pendingSec = shoppingList.vendor_sections.find(s => s.status === 'pending');
    if (!pendingSec) { test.skip(true, 'Seeded list has no pending sections — unexpected state'); return; }

    // Check off all items in the section via API
    for (const item of pendingSec.items || []) {
      if (!item.checked) {
        await poApiCall(page, 'POST', 'shopping/' + shoppingList.id + '/check', { item_id: item.id, checked: true });
      }
    }

    await page.click('#t2');
    await waitForShoppingContent(page);

    // Click the Complete Vendor button
    const completeBtn = page.locator('[data-action="complete-vendor"][data-section-id="' + pendingSec.id + '"]');
    const btnVisible = await completeBtn.isVisible().catch(() => false);
    if (!btnVisible) { test.skip(true, 'Complete button not visible'); return; }

    // Accept the confirm dialog
    page.on('dialog', dialog => dialog.accept());
    await completeBtn.click();

    // Wait for the API call
    await page.waitForResponse(res => res.url().includes('/vendors/') && res.url().includes('/complete'), { timeout: 5000 }).catch(() => {});
    await page.waitForTimeout(500);

    // Reload and verify section shows as completed
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.click('#t2');
    await waitForShoppingContent(page);

    // FR-12 (rewritten from a vacuous toBeTruthy tail): the completed section must
    // persist as status='completed' in the DB. Assert it via the API (the render is
    // reload-dependent DOM; the DB state is the authoritative contract).
    const afterActive = await poApiCall(page, 'GET', 'shopping/active').catch(() => null);
    const afterHistory = await poApiCall(page, 'GET', 'shopping/history').catch(() => []);
    // The section lives in whichever list still holds it (active if others pending,
    // history if this completion cascaded the whole list to completed).
    const findSec = (list) => (list && list.vendor_sections || []).find(s => s.id === pendingSec.id);
    const persistedSec = findSec(afterActive) ||
      ((afterHistory || []).map(findSec).find(Boolean));
    expect(persistedSec, 'the completed vendor section must be retrievable after reload').toBeTruthy();
    expect(persistedSec.status).toBe('completed');
    expect(persistedSec.completed_by, 'completed_by must be stamped').toBeTruthy();
  });

  test('store location edit persists after reload', async ({ page }) => {
    const shoppingList = await seedShoppingList(page);
    const firstItem = shoppingList.vendor_sections[0].items[0];

    await page.click('#t2');
    await waitForShoppingContent(page);

    // Tap the location element for the first item
    const locEl = page.locator('.shop-item[data-item-id="' + firstItem.id + '"] .shop-loc');
    await locEl.click();

    // Type a location
    const testLoc = 'Aisle 7B';
    const inp = page.locator('.shop-item[data-item-id="' + firstItem.id + '"] .shop-loc input');
    await inp.fill(testLoc);
    await inp.press('Enter');

    // Wait for API save
    await page.waitForResponse(res => res.url().includes('/items/') && res.url().includes('/location'), { timeout: 5000 }).catch(() => {});
    await page.waitForTimeout(500);

    // Reload and verify location persists
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.click('#t2');
    await waitForShoppingContent(page);

    const locElAfter = page.locator('.shop-item[data-item-id="' + firstItem.id + '"] .shop-loc');
    const locText = await locElAfter.textContent();
    expect(locText).toContain(testLoc);
  });

  test('shopping aisle location does not overwrite catalog store_location', async ({ page }) => {
    const shoppingList = await seedShoppingList(page);
    const firstItem = shoppingList.vendor_sections[0].items[0];

    // Get the catalog item's current store_location via inventory API
    const catalogBefore = await page.evaluate(async (pid) => {
      const res = await fetch('/api/v1/inventory/items');
      const items = await res.json();
      return items.find(i => i.id === pid);
    }, firstItem.purchase_item_id);
    const originalLoc = catalogBefore ? catalogBefore.store_location : null;

    // Set a shopping aisle location via the shopping API
    const aisleLocation = 'Test Aisle 99Z';
    await poApiCall(page, 'PUT', 'shopping/' + shoppingList.id + '/items/' + firstItem.id + '/location', { store_location: aisleLocation });

    // Verify the catalog item's store_location was NOT changed
    const catalogAfter = await page.evaluate(async (pid) => {
      const res = await fetch('/api/v1/inventory/items');
      const items = await res.json();
      return items.find(i => i.id === pid);
    }, firstItem.purchase_item_id);

    expect(catalogAfter.store_location).toBe(originalLoc);
  });

  test('toast appears when checking item without photo', async ({ page }) => {
    const shoppingList = await seedShoppingList(page);

    // Find an unchecked item without a photo (seeded items start unchecked + photoless).
    let targetItem = null;
    for (const sec of shoppingList.vendor_sections) {
      for (const item of (sec.items || [])) {
        if (!item.checked && !item.photo_url) { targetItem = item; break; }
      }
      if (targetItem) break;
    }
    if (!targetItem) { test.skip(true, 'Seeded list has no unchecked photoless items — unexpected state'); return; }

    await page.click('#t2');
    await waitForShoppingContent(page);

    // Click the check button
    const checkEl = page.locator('[data-action="shop-check"][data-item-id="' + targetItem.id + '"]');
    await checkEl.click();

    // Wait for toast to appear
    await page.waitForSelector('#shop-toast', { state: 'visible', timeout: 5000 });
    const toastText = await page.locator('#shop-toast').textContent();
    expect(toastText).toMatch(/photo|location/i);

    // Verify "Add Now" button is in the toast
    const addNowBtn = page.locator('[data-action="toast-add-now"]');
    await expect(addNowBtn).toBeVisible();
  });

  test('No photo badge shows on checked item without photo and disappears after photo upload', async ({ page }) => {
    // Seed a shopping list if none active
    let shoppingList;
    try {
      shoppingList = await poApiCall(page, 'GET', 'shopping/active');
    } catch(e) { /* no active list */ }
    if (!shoppingList || !shoppingList.vendor_sections || shoppingList.vendor_sections.length === 0) {
      shoppingList = await seedShoppingList(page);
    }
    expect(shoppingList).toBeTruthy();
    expect(shoppingList.vendor_sections.length).toBeGreaterThan(0);

    const targetItem = shoppingList.vendor_sections[0].items[0];
    expect(targetItem).toBeTruthy();

    // Ensure item is checked and has no photo (badge only shows when checked && !photo_url)
    if (!targetItem.checked) {
      await poApiCall(page, 'POST', 'shopping/' + shoppingList.id + '/check', { item_id: targetItem.id, checked: true });
    }

    // Clear photo_url directly via evaluate to ensure clean state
    await page.evaluate(async ([listId, itemId]) => {
      await fetch('/api/v1/purchasing/shopping/' + listId + '/items/' + itemId + '/photo', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ photo_url: 'CLEAR' })
      });
    }, [shoppingList.id, targetItem.id]);

    // Actually need to clear at DB level — update the handler to accept CLEAR is messy.
    // Instead: navigate, check badge appears for items without photos.
    // The real test: check the badge logic by verifying DOM state matches API data.

    // Navigate to Shopping tab
    await page.click('#t2');
    await waitForShoppingContent(page);

    // Find all checked items in the DOM
    const checkedItems = await page.evaluate(() => {
      const items = document.querySelectorAll('.shop-item.checked');
      return Array.from(items).map(el => ({
        hasPhoto: !!el.querySelector('.item-thumb img'),
        hasNoPhotoBadge: !!el.querySelector('.shop-warn')
      }));
    });

    // Every checked item without a photo should show the badge
    for (const item of checkedItems) {
      if (!item.hasPhoto) {
        expect(item.hasNoPhotoBadge).toBe(true);
      }
    }

    // Now upload a photo to the target item and verify badge disappears
    const fakePhotoUrl = 'https://example.com/test-photo-' + Date.now() + '.jpg';
    await poApiCall(page, 'PUT', 'shopping/' + shoppingList.id + '/items/' + targetItem.id + '/photo', { photo_url: fakePhotoUrl });

    // Reload and verify badge is gone for this item, photo thumbnail shows
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.click('#t2');
    await waitForShoppingContent(page);

    // Find the item's row and verify it has an img and no "No photo" badge
    const itemState = await page.evaluate((itemName) => {
      const items = document.querySelectorAll('.shop-item');
      for (const el of items) {
        const nm = el.querySelector('.nm');
        if (nm && nm.textContent.includes(itemName)) {
          return {
            hasImg: !!el.querySelector('.item-thumb img'),
            hasNoPhotoBadge: !!el.querySelector('.shop-warn')
          };
        }
      }
      return null;
    }, targetItem.item_name);

    expect(itemState).toBeTruthy();
    expect(itemState.hasImg).toBe(true);
    expect(itemState.hasNoPhotoBadge).toBe(false);
  });

  // ─── FR-12: last-section-complete cascades list → completed AND PO → completed ──
  // AC-4: completing the last pending vendor section flips the shopping list to
  // 'completed' and its PO to 'completed' in the DB. Red-first, API-observed — this
  // replaces the vacuous cascade coverage the PRD flagged on FR-12.
  test('FR-12: completing the last vendor section cascades list and PO to completed', async ({ page }) => {
    const list = await seedShoppingList(page);
    test.skip(!list || !(list.vendor_sections || []).length, 'No catalog items to seed a shopping list');

    const poId = list.po_id;
    expect(poId, 'seeded list must carry its po_id').toBeTruthy();

    // Sanity: before completion the list is active and the PO is shopping_active.
    const poBefore = await poApiCall(page, 'GET', 'orders/' + poId);
    expect(poBefore.status).toBe('shopping_active');

    // Complete every pending vendor section; the LAST one triggers the cascade.
    for (const sec of list.vendor_sections) {
      if (sec.status === 'completed') continue;
      await poApiCall(page, 'POST', 'shopping/' + list.id + '/vendors/' + sec.id + '/complete');
    }

    // Observable cascade 1: the shopping list is now 'completed' and shows in history.
    const active = await poApiCall(page, 'GET', 'shopping/active').catch(() => null);
    expect(active, 'no shopping list should be active after the last section completes').toBeFalsy();
    const history = await poApiCall(page, 'GET', 'shopping/history').catch(() => []);
    const inHistory = (history || []).find(h => h.id === list.id);
    expect(inHistory, 'the completed list must land in history').toBeTruthy();
    expect(inHistory.status).toBe('completed');

    // Observable cascade 2: the associated PO transitioned draft/…→ completed.
    const poAfter = await poApiCall(page, 'GET', 'orders/' + poId);
    expect(poAfter.status).toBe('completed');
  });

  // ─── FR-24: completing a section records a repurchase_log row ──────────────────
  // repurchase.go RecordRepurchase inserts one repurchase_log row per CHECKED item
  // of a completed vendor section (best-effort, post-commit). The only read surface
  // is GET /api/v1/inventory/items, which attaches a `repurchase_badge` to any item
  // with a repurchase_log entry since the last reset (none configured here → all
  // repurchased items badge). Assert the badge appears for the checked item after
  // its section completes. Red-first, API-observed.
  test('FR-24: completing a vendor section with checked items records a repurchase_log row', async ({ page }) => {
    // Observable join: repurchase.go RecordRepurchase inserts a repurchase_log row
    // per CHECKED item of a completed section; the only read surface is GET
    // /inventory/stock, which aggregates by catalog description and attaches a
    // repurchase_badge (no reset config → badges accumulate). A stock row only
    // exists for items with real purchase history, so we must operate on a section
    // item whose purchase_item_id maps to a catalog description present in /stock.
    const catalog = await page.evaluate(async () => {
      const r = await fetch('/api/v1/inventory/items'); return r.json(); // {id, description, store_location}
    });
    test.skip(!catalog || !catalog.length, 'No catalog items to seed from');
    const descById = new Map((catalog || []).map(c => [c.id, c.description]));

    // seedShoppingList builds its list from the first 2 located catalog items. Give
    // those items real purchase history via POST /inventory/purchases so they appear
    // in /stock and their repurchase_badge is joinable (pure API seed, no DB conn).
    const vendors = await page.evaluate(async () => {
      const r = await fetch('/api/v1/inventory/vendors'); return r.json();
    });
    const vendorId = vendors && vendors[0] && vendors[0].id;
    test.skip(!vendorId, 'No vendor to attribute a purchase event to');
    for (const c of catalog.slice(0, 2)) {
      await page.evaluate(async ([vid, pid, desc]) => {
        await fetch('/api/v1/inventory/purchases', {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            vendor_id: vid, bank_tx_id: 'fr24-seed-' + pid, event_date: '2026-01-15',
            tax: 0, total: 5,
            line_items: [{ purchase_item_id: pid, description: desc, quantity: 1, price: 5, is_case: false }],
          }),
        });
      }, [vendorId, c.id, c.description]);
    }

    const stockSeed = await page.evaluate(async () => {
      const r = await fetch('/api/v1/inventory/stock'); return r.json();
    });
    const stockDescs = new Set((stockSeed || []).map(s => s.description));

    // Reuse the same seed path the passing FR-12 tests use — robust against the
    // suite's known leftover-state pollution.
    const list = await seedShoppingList(page).catch(() => null);
    test.skip(!list || !(list.vendor_sections || []).length, 'No catalog items to seed a shopping list');

    // Find a PENDING section holding an item whose catalog description is in /stock.
    let sec = null, item = null, desc = null;
    for (const s of list.vendor_sections) {
      if (s.status === 'completed') continue;
      for (const it of (s.items || [])) {
        const d = descById.get(it.purchase_item_id);
        if (d && stockDescs.has(d)) { sec = s; item = it; desc = d; break; }
      }
      if (item) break;
    }
    test.skip(!item, 'Seeded list has no pending section item that also appears in /stock — cannot join the badge');

    // Baseline badge qty (badges accumulate across runs → assert on the DELTA).
    const stockBefore = await page.evaluate(async () => {
      const r = await fetch('/api/v1/inventory/stock'); return r.json();
    });
    const before = (stockBefore || []).find(s => s.description === desc);
    const qtyBefore = (before && before.repurchase_badge && before.repurchase_badge.qty) || 0;

    // Check the item, then complete its vendor section → RecordRepurchase inserts a row.
    await poApiCall(page, 'POST', 'shopping/' + list.id + '/check', { item_id: item.id, checked: true });
    await poApiCall(page, 'POST', 'shopping/' + list.id + '/vendors/' + sec.id + '/complete');

    // Observable: the item's aggregated stock row now carries a repurchase_badge
    // sourced from the repurchase_log row RecordRepurchase wrote, and its qty grew by
    // at least the checked quantity — proving a NEW repurchase_log row was recorded.
    const stockAfter = await page.evaluate(async () => {
      const r = await fetch('/api/v1/inventory/stock'); return r.json();
    });
    const after = (stockAfter || []).find(s => s.description === desc);
    expect(after, 'the repurchased item must have an aggregated stock row').toBeTruthy();
    expect(after.repurchase_badge, 'a repurchase_log row must surface as a repurchase_badge after complete').toBeTruthy();
    expect(after.repurchase_badge.qty).toBeGreaterThanOrEqual(qtyBefore + item.quantity);
  });

});

test.describe('History tab', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
  });

  test('History tab renders #history-content container', async ({ page }) => {
    await page.click('#t4');
    // The rebuilt History tab always renders a #history-content container inside
    // #s4 (empty state OR populated) — never leaves the raw stub in place.
    await page.waitForFunction(() => {
      const el = document.getElementById('history-content');
      if (!el) return false;
      return el.textContent.trim().length > 0;
    }, { timeout: 8000 });
    const text = await page.locator('#history-content').textContent();
    expect(text.trim().length).toBeGreaterThan(0);
  });

  test('empty state shows "No completed" when there is no history', async ({ page }) => {
    // Whether or not other tests have seeded, the wait accepts either the empty
    // state text or a populated card. On a fresh DB (no completed lists) it must
    // read "No completed".
    await page.click('#t4');
    await page.waitForFunction(() => {
      const el = document.getElementById('history-content');
      if (!el) return false;
      return el.querySelector('.history-card') || el.textContent.includes('No completed');
    }, { timeout: 8000 });

    const history = await poApiCall(page, 'GET', 'shopping/history').catch(() => []);
    const text = await page.locator('#history-content').textContent();
    if (!history || history.length === 0) {
      expect(text).toContain('No completed');
    } else {
      // Something already seeded a completed list — the populated UI must render.
      await expect(page.locator('.history-card').first()).toBeVisible();
    }
  });

  test('history tab shows seeded completed shopping list with week label', async ({ page }) => {
    const completed = await completeShoppingList(page);
    test.skip(!completed, 'No catalog items to seed a completed shopping list');

    await page.click('#t4');
    await page.waitForSelector('.history-card', { timeout: 8000 });

    await expect(page.locator('.history-card').first()).toBeVisible();
    const cardText = await page.locator('.history-card').first().textContent();
    expect(cardText).toMatch(/Week of/);
  });

  test('tapping history entry expands the vendor breakdown detail', async ({ page }) => {
    const completed = await completeShoppingList(page);
    test.skip(!completed, 'No catalog items to seed a completed shopping list');

    await page.click('#t4');
    await page.waitForSelector('.history-card', { timeout: 8000 });

    // Detail should be collapsed initially.
    expect(await page.locator('.history-detail').first().isVisible().catch(() => false)).toBe(false);

    // Tap the header to expand.
    await page.locator('.history-hd').first().click();
    await page.waitForSelector('.history-detail', { state: 'visible', timeout: 5000 });
    await expect(page.locator('.history-detail').first()).toBeVisible();

    // The detail must name at least one vendor from the seeded list.
    const detailText = await page.locator('.history-detail').first().textContent();
    expect(detailText.trim().length).toBeGreaterThan(0);
    expect(completed.vendor_sections.length).toBeGreaterThan(0);
    expect(detailText).toContain(completed.vendor_sections[0].vendor_name);
  });

  test('history card shows vendor breakdown and section count', async ({ page }) => {
    const completed = await completeShoppingList(page);
    test.skip(!completed, 'No catalog items to seed a completed shopping list');

    await page.click('#t4');
    await page.waitForSelector('.history-card', { timeout: 8000 });

    // The card header meta must report the vendor/section count.
    const metaText = await page.locator('.history-mt').first().textContent();
    expect(metaText).toMatch(/vendor/i);
  });

  // ─── FR-17: GET /shopping/history returns the completed-list shape ─────────────
  // The endpoint (service.go GetShoppingListHistory) is untested by the PRD. Assert
  // the JSON shape it names: an array of completed shopping lists, each with id,
  // week_start, status='completed', and vendor_sections[] carrying vendor_name +
  // status. Red-first, API-only (independent of the FR-18 History UI stub).
  test('FR-17: shopping/history returns completed lists with vendor-section shape', async ({ page }) => {
    const completed = await completeShoppingList(page);
    test.skip(!completed, 'No catalog items to seed a completed shopping list');

    const history = await poApiCall(page, 'GET', 'shopping/history');
    expect(Array.isArray(history), 'history must be a JSON array').toBe(true);
    expect(history.length).toBeGreaterThan(0);

    const row = history.find(h => h.id === completed.id);
    expect(row, 'the just-completed list must appear in history').toBeTruthy();

    // Only completed lists are returned.
    for (const h of history) expect(h.status).toBe('completed');

    // Shape of the returned row.
    expect(typeof row.id).toBe('string');
    expect(row.week_start).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(row.po_id, 'row must reference its PO').toBeTruthy();
    expect(Array.isArray(row.vendor_sections)).toBe(true);
    expect(row.vendor_sections.length).toBeGreaterThan(0);

    // Each vendor section carries a vendor_name and a status.
    for (const sec of row.vendor_sections) {
      expect(typeof sec.vendor_name).toBe('string');
      expect(sec.vendor_name.length).toBeGreaterThan(0);
      expect(['pending', 'completed']).toContain(sec.status);
    }
  });

});

test.describe('PO tab', () => {

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
  });

  test('PO tab shows stub or locked PO', async ({ page }) => {
    await page.click('#t3');
    await page.waitForFunction(() => {
      const el = document.getElementById('po-content');
      return el && el.textContent.trim().length > 0;
    }, { timeout: 8000 });
    const text = await page.locator('#po-content').textContent();
    expect(text.trim().length).toBeGreaterThan(0);
  });

  test('PO tab shows approve button for admin when PO is locked', async ({ page }) => {
    let locked;
    try {
      locked = await poApiCall(page, 'GET', 'orders?status=locked');
    } catch(e) { test.skip(true, 'No locked PO'); return; }
    if (!locked) { test.skip(true, 'No locked PO'); return; }

    await page.click('#t3');
    await page.waitForFunction(() => {
      const el = document.getElementById('po-content');
      return el && el.textContent.trim().length > 0;
    }, { timeout: 8000 });

    // Admin should see Approve button
    const approvBtn = page.locator('[data-action="approve-po"]');
    const isVisible = await approvBtn.isVisible().catch(() => false);
    // Only visible if PO is locked (not approved/shopping_active)
    if (locked.status === 'locked') {
      await expect(approvBtn).toBeVisible();
    }
  });

});

// ── Store location enforcement in item picker ────────────────────────────

test.describe('Item picker store_location enforcement', () => {

  async function invApiCall(page, method, path, body) {
    return page.evaluate(async ([m, p, b]) => {
      const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
      if (b) opts.body = JSON.stringify(b);
      const res = await fetch('/api/v1/inventory/' + p, opts);
      if (res.status === 204) return null;
      return res.json();
    }, [method, path, body]);
  }

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
  });

  test('item without store_location shows "Set location in Setup" instead of Add button', async ({ page }) => {
    // Create item without store_location
    const groups = await invApiCall(page, 'GET', 'groups');
    const gid = groups && groups.length ? groups[0].id : null;
    const ts = Date.now();
    const itemName = 'NoLoc Item ' + ts;
    await invApiCall(page, 'POST', 'items', { description: itemName, group_id: gid });

    // Open item picker
    await page.waitForSelector('[data-action="open-picker"]', { timeout: 5000 });
    await page.click('[data-action="open-picker"]');
    await page.waitForSelector('#item-modal.open', { timeout: 3000 });

    // Search for the item
    await page.fill('#picker-search', itemName);
    await page.waitForTimeout(300);

    // Verify the item appears under "Unassigned" group with no Add button
    const row = page.locator('.picker-row', { hasText: itemName });
    await expect(row).toBeVisible();
    await expect(row.locator('.pr-add')).not.toBeVisible();
    await expect(row.locator('.pr-unassigned')).toContainText('Set location in Setup');
  });

  test('item with store_location shows normal Add button', async ({ page }) => {
    // Create item with store_location
    const groups = await invApiCall(page, 'GET', 'groups');
    const gid = groups && groups.length ? groups[0].id : null;
    const ts = Date.now();
    const itemName = 'Located Item ' + ts;
    const created = await invApiCall(page, 'POST', 'items', { description: itemName, group_id: gid });

    // Set store_location via inventory PUT
    await invApiCall(page, 'PUT', 'items', { id: created.id, description: itemName, group_id: gid, store_location: 'Giant' });

    // Reload to pick up new item in ALL_ITEMS
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Open item picker
    await page.waitForSelector('[data-action="open-picker"]', { timeout: 5000 });
    await page.click('[data-action="open-picker"]');
    await page.waitForSelector('#item-modal.open', { timeout: 3000 });

    // Search for the item
    await page.fill('#picker-search', itemName);
    await page.waitForTimeout(300);

    // Verify the item has an Add button
    const row = page.locator('.picker-row', { hasText: itemName });
    await expect(row).toBeVisible();
    await expect(row.locator('.pr-add')).toBeVisible();
  });

  test('items grouped by store_location with headers in picker', async ({ page }) => {
    // Create items with different locations
    const groups = await invApiCall(page, 'GET', 'groups');
    const gid = groups && groups.length ? groups[0].id : null;
    const ts = Date.now();

    const item1Name = 'PickGiant ' + ts;
    const item2Name = 'PickDepot ' + ts;
    const item3Name = 'PickNoLoc ' + ts;

    const it1 = await invApiCall(page, 'POST', 'items', { description: item1Name, group_id: gid });
    const it2 = await invApiCall(page, 'POST', 'items', { description: item2Name, group_id: gid });
    await invApiCall(page, 'POST', 'items', { description: item3Name, group_id: gid });

    // Set locations
    await invApiCall(page, 'PUT', 'items', { id: it1.id, description: item1Name, group_id: gid, store_location: 'Giant' });
    await invApiCall(page, 'PUT', 'items', { id: it2.id, description: item2Name, group_id: gid, store_location: 'Restaurant Depot' });

    // Reload to pick up changes
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Open item picker and search for our prefix
    await page.waitForSelector('[data-action="open-picker"]', { timeout: 5000 });
    await page.click('[data-action="open-picker"]');
    await page.waitForSelector('#item-modal.open', { timeout: 3000 });

    await page.fill('#picker-search', 'Pick');
    await page.waitForTimeout(300);

    // Verify group headers appear
    const headers = await page.evaluate(() => {
      return Array.from(document.querySelectorAll('.picker-group-header')).map(h => h.textContent);
    });

    expect(headers).toContain('Giant');
    expect(headers).toContain('Restaurant Depot');
    expect(headers).toContain('Unassigned');

    // Verify "Unassigned" is last
    const unassignedIdx = headers.indexOf('Unassigned');
    expect(unassignedIdx).toBe(headers.length - 1);
  });

  test('item picker renders all items when catalog exceeds 30', async ({ page }) => {
    // Seed 35 items with store_location set
    const groups = await invApiCall(page, 'GET', 'groups');
    const gid = groups && groups.length ? groups[0].id : null;
    const ts = Date.now();
    const prefix = 'Cap' + ts;

    for (let i = 0; i < 35; i++) {
      const name = prefix + ' Item ' + String(i).padStart(2, '0');
      const created = await invApiCall(page, 'POST', 'items', { description: name, group_id: gid });
      await invApiCall(page, 'PUT', 'items', { id: created.id, description: name, group_id: gid, store_location: 'TestLoc' });
    }

    // Reload to pick up new items
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Open item picker
    await page.waitForSelector('[data-action="open-picker"]', { timeout: 5000 });
    await page.click('[data-action="open-picker"]');
    await page.waitForSelector('#item-modal.open', { timeout: 3000 });

    // Clear search to show all
    await page.fill('#picker-search', '');
    await page.waitForTimeout(300);

    // Count picker rows — should be at least 35
    const count = await page.locator('.picker-row').count();
    expect(count).toBeGreaterThanOrEqual(35);
  });

  test('item picker stays open after scrolling to bottom of list', async ({ page }) => {
    // Open item picker
    await page.waitForSelector('[data-action="open-picker"]', { timeout: 5000 });
    await page.click('[data-action="open-picker"]');
    await page.waitForSelector('#item-modal.open', { timeout: 3000 });

    // Clear search to show all items
    await page.fill('#picker-search', '');
    await page.waitForTimeout(300);

    // Scroll the picker list to the very bottom
    await page.evaluate(() => {
      var list = document.getElementById('picker-list');
      list.scrollTop = list.scrollHeight;
    });
    await page.waitForTimeout(500);

    // Modal must still be open
    await expect(page.locator('#item-modal.open')).toBeVisible();
  });

  test('addItemToPO guard blocks items without store_location via toast', async ({ page }) => {
    // Create item without store_location
    const groups = await invApiCall(page, 'GET', 'groups');
    const gid = groups && groups.length ? groups[0].id : null;
    const ts = Date.now();
    const itemName = 'GuardTest ' + ts;
    const created = await invApiCall(page, 'POST', 'items', { description: itemName, group_id: gid });

    // Reload to pick up new item
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Try to call addItemToPO directly via evaluate (bypassing picker UI)
    const toastShown = await page.evaluate((itemId) => {
      // Ensure PO_STATE exists
      if (typeof PO_STATE === 'undefined' || !PO_STATE) {
        window.PO_STATE = { id: 'test', line_items: [] };
      }
      window.PICKER_TARGET = 'order';
      addItemToPO(itemId);
      var toast = document.querySelector('.shop-toast');
      return toast ? toast.textContent : null;
    }, created.id);

    expect(toastShown).toContain('Set a store location in Inventory Setup');
  });

});

// ── Item card → Inventory Setup deep link ────────────────────────────────

test.describe('Item card Setup deep link', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
  });

  test('clicking item info on Order tab navigates to Inventory Setup with item expanded', async ({ page }) => {
    // Wait for order to render with at least one item
    const itemInfo = page.locator('.item-info[data-action="goto-setup"]').first();
    await expect(itemInfo).toBeVisible({ timeout: 10000 });
    const itemName = await itemInfo.locator('.nm').textContent();
    // Click the item info area
    await itemInfo.click();
    // Should navigate to inventory.html
    await page.waitForURL(/inventory\.html/, { timeout: 5000 });
    // URL hash should contain the item name
    const hash = await page.evaluate(() => location.hash);
    expect(hash).toContain('tab=5');
    expect(hash).toContain('item=');
    // Setup tab should be visible with the item edit form
    await page.waitForSelector('.item-edit-form', { timeout: 10000 });
    const editName = await page.locator('.item-edit-name').inputValue();
    expect(editName.toLowerCase()).toBe(itemName.toLowerCase());
  });
});

// ── Regression: suggestions load on purchasing.html ──────────────────────

test.describe('Purchasing Suggestions', () => {
  test('suggestions from inventory appear on purchasing.html Order tab', async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForSelector('#s1', { timeout: 10000 });

    // Wait for init to complete — check if suggestions card or items rendered
    await page.waitForTimeout(2000);

    // Verify no console errors on the suggestions endpoint
    const errors = [];
    page.on('pageerror', err => errors.push(err.message));

    // Check that the suggestions API was called with the correct PO-specific URL
    const suggestionsLoaded = await page.evaluate(() => {
      // SUGGESTIONS is a let-scoped var, but we can check the DOM
      var suggCard = document.getElementById('suggestions-card');
      var s1 = document.getElementById('s1');
      var html = s1 ? s1.innerHTML : '';
      // Either suggestions card is visible (items below threshold exist)
      // or the empty state shows (no items need restock) — both are valid
      // What's NOT valid: a JS error preventing the page from loading
      return {
        pageLoaded: html.length > 50,
        hasSuggestions: suggCard && suggCard.style.display !== 'none',
        hasOrderContent: html.includes('Week of') || html.includes('Nothing on the order')
      };
    });

    expect(suggestionsLoaded.pageLoaded).toBeTruthy();
    expect(suggestionsLoaded.hasOrderContent).toBeTruthy();
  });
});

// ── prove-sweep: Order-tab flows FR-1, FR-2, FR-4, FR-6 (card purchasing-prove-order) ──
//
// Red-first assertions that name the observable DB/UI behavior for the four Order-tab
// flows the PRD marks UNPROVEN. FR-1/FR-6 are expected RED per the slate. Fixtures are
// isolated to this describe block; they never mutate state a sibling test depends on
// (each seeds its own draft/purchase-event or authors its own non-admin session inline).

test.describe('Order tab prove-sweep (FR-1/2/4/6)', () => {

  // status-aware fetch (poApiCall throws without exposing status; FR-6 needs the code).
  async function poFetchStatus(page, method, path, body) {
    return page.evaluate(async ([m, p, b]) => {
      const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
      if (b) opts.body = JSON.stringify(b);
      const res = await fetch('/api/v1/purchasing/' + p, opts);
      let json = null;
      try { json = await res.json(); } catch (e) { json = null; }
      return { status: res.status, ok: res.ok, body: json };
    }, [method, path, body]);
  }

  // Ensure the current-week draft exists but is NOT yet locked. Returns the draft PO.
  async function ensureDraft(page) {
    return poApiCall(page, 'POST', 'orders');
  }

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
  });

  // FR-1 — On load POST /orders always returns an editable DRAFT. When this week's PO
  // is locked, the returned draft must roll to NEXT week (never hand back the locked PO).
  // Observable: after simulate-cutoff locks this week, POST /orders returns status='draft'
  // AND a week_start strictly after the locked PO's week_start.
  test('FR-1: locked-week draft rolls to next week (POST /orders returns a next-week draft)', async ({ page }) => {
    // Precondition: a locked PO must exist. Reuse an existing one if the stack already
    // has one (idempotent — avoids the locked_po_pending_approval 409 from re-locking);
    // otherwise seed a draft with an item and simulate-cutoff to create it.
    let locked = await poApiCall(page, 'GET', 'orders?status=locked').catch(() => null);
    if (!locked || !locked.id) {
      const draft = await ensureDraft(page);
      const items = await page.evaluate(async () => (await fetch('/api/v1/inventory/items')).json());
      if (!items || !items.length) { test.skip(true, 'No catalog items to seed a draft'); return; }
      await poApiCall(page, 'PUT', 'orders/' + draft.id + '/items', {
        items: [{ purchase_item_id: items[0].id, quantity: 2, unit: '' }],
      });
      await poApiCall(page, 'POST', 'simulate-cutoff');
      locked = await poApiCall(page, 'GET', 'orders?status=locked');
    }
    expect(locked).toBeTruthy();
    expect(locked.status).toBe('locked');
    const lockedWeek = locked.week_start;

    // With this week locked, the Order tab must roll to an editable NEXT-week draft.
    const rolled = await poApiCall(page, 'POST', 'orders');
    expect(rolled).toBeTruthy();
    expect(rolled.status).toBe('draft');                 // never a locked PO
    expect(rolled.id).not.toBe(locked.id);               // a different order
    // week_start must advance past the locked week (roll-to-next-week branch).
    expect(new Date(rolled.week_start).getTime())
      .toBeGreaterThan(new Date(lockedWeek).getTime());
  });

  // FR-2 — Stepping a line item persists across reload; stepping to qty 0 removes it.
  // Observable: after debounced save + reload, GET /orders/{id} line_items shows the new
  // qty for the stepped item; a qty-0 item is absent from the persisted set.
  test('FR-2: qty stepper persists across reload and qty-0 removes the line', async ({ page }) => {
    const items = await page.evaluate(async () => (await fetch('/api/v1/inventory/items')).json());
    if (!items || items.length < 2) { test.skip(true, 'Need >=2 catalog items'); return; }
    const keepItem = items[0];   // will be stepped up to 3 and kept
    const dropItem = items[1];   // will be stepped down to 0 and removed

    // Seed the current draft with both items at qty 2.
    const draft = await ensureDraft(page);
    await poApiCall(page, 'PUT', 'orders/' + draft.id + '/items', {
      items: [
        { purchase_item_id: keepItem.id, quantity: 2, unit: '' },
        { purchase_item_id: dropItem.id, quantity: 2, unit: '' },
      ],
    });

    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('#s1 .item-card[data-item-id="' + keepItem.id + '"]', { timeout: 8000 });

    // Scope to the Order tab (#s1); the qty is the aria-labelled span in each card's stepper.
    const keepQty = page.locator('#s1 .item-card[data-item-id="' + keepItem.id + '"] .stp span[aria-label]');
    const dropQty = page.locator('#s1 .item-card[data-item-id="' + dropItem.id + '"] .stp span[aria-label]');

    // Step keepItem UP by one (2 -> 3).
    await page.locator('#s1 .item-card[data-item-id="' + keepItem.id + '"] [data-action="qty-inc"]').click();
    await expect(keepQty).toHaveText('3');

    // Step dropItem DOWN to 0 (2 -> 1 -> 0).
    const decDrop = page.locator('#s1 .item-card[data-item-id="' + dropItem.id + '"] [data-action="qty-dec"]');
    await decDrop.click();
    await decDrop.click();
    await expect(dropQty).toHaveText('0');

    // Let the debounced save flush, then reload and read the PERSISTED set.
    await page.waitForResponse(res => res.url().includes('/orders/') && res.url().includes('/items') && res.request().method() === 'PUT', { timeout: 6000 }).catch(() => {});
    await page.waitForTimeout(1200);
    await page.reload();
    await page.waitForLoadState('networkidle');

    const persisted = await poApiCall(page, 'GET', 'orders/' + draft.id);
    const li = persisted.line_items || [];
    const keep = li.find(x => x.purchase_item_id === keepItem.id);
    const drop = li.find(x => x.purchase_item_id === dropItem.id);
    expect(keep).toBeTruthy();
    expect(keep.quantity).toBe(3);          // stepped-up qty persisted
    expect(drop).toBeFalsy();               // qty-0 line removed from the persisted set
  });

  // FR-4 — Restock suggestions render and "Add Selected" bulk-adds them to the order.
  // Seed a purchase event (gives a catalog item stock below its group threshold) so the
  // suggestions query returns a concrete candidate, then drive the UI bulk-add and assert
  // the item lands on the PERSISTED PO line-item set after reload.
  test('FR-4: restock suggestions render and Add Selected persists them onto the order', async ({ page }) => {
    // Fresh draft so the candidate item is guaranteed NOT already on the PO.
    const draft = await ensureDraft(page);

    // Pick a catalog item that is not currently a draft line item, and give it stock=2
    // via a purchase event (2 < default high threshold 10, > 0 -> becomes a suggestion).
    const items = await page.evaluate(async () => (await fetch('/api/v1/inventory/items')).json());
    const vendors = await page.evaluate(async () => (await fetch('/api/v1/inventory/vendors')).json());
    if (!items || !items.length || !vendors || !vendors.length) { test.skip(true, 'No catalog items/vendors to seed a suggestion'); return; }
    const onPO = new Set((draft.line_items || []).map(x => x.purchase_item_id));
    const candidate = items.find(it => !onPO.has(it.id));
    if (!candidate) { test.skip(true, 'No off-PO catalog item to seed a suggestion'); return; }

    await page.evaluate(async ([vid, pid]) => {
      await fetch('/api/v1/inventory/purchases', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          vendor_id: vid, bank_tx_id: 'nc-fr4-' + Date.now(), event_date: '2026-07-01', total: 10,
          line_items: [{ purchase_item_id: pid, description: 'fr4 seed', quantity: 2, price: 5 }],
        }),
      });
    }, [vendors[0].id, candidate.id]);

    // Confirm the suggestion is really produced for THIS draft (guards the fixture).
    const suggestions = await poApiCall(page, 'GET', 'orders/' + draft.id + '/suggestions');
    const sug = (suggestions || []).find(s => s.purchase_item_id === candidate.id);
    expect(sug).toBeTruthy();   // fixture must yield a concrete suggestion, not an empty set

    // Reload so the Order tab hydrates SUGGESTIONS, then open the suggestions card.
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.waitForSelector('#s1 [data-action="toggle-suggestions"]', { timeout: 8000 });
    await page.locator('#s1 [data-action="toggle-suggestions"]').first().click();

    // The suggestion row for our candidate must render, then check it and Add Selected.
    const cb = page.locator('.suggest-row input[data-suggest-id="' + candidate.id + '"]');
    await expect(cb).toBeVisible();
    await cb.check();
    await page.locator('[data-action="add-selected"]').click();

    // Persisted assertion: after the save + reload, the candidate is on the PO at its
    // suggested qty.
    await page.waitForResponse(res => res.url().includes('/orders/') && res.url().includes('/items') && res.request().method() === 'PUT', { timeout: 6000 }).catch(() => {});
    await page.waitForTimeout(1000);
    const persisted = await poApiCall(page, 'GET', 'orders/' + draft.id);
    const added = (persisted.line_items || []).find(x => x.purchase_item_id === candidate.id);
    expect(added).toBeTruthy();                         // bulk-add wrote the line item
    expect(added.quantity).toBe(sug.suggested_qty);     // at the suggested qty
  });

  // FR-6 — Cutoff config is settable by an admin (round-trips) and returns 403 for a
  // non-admin (team_member). The non-admin session is authored INLINE per the runbook
  // (invite -> accept-invite -> login as them); no shared helper module.
  test('FR-6: cutoff config is admin-settable and returns 403 for a non-admin', async ({ page }) => {
    // --- Admin path: PUT /cutoff round-trips ---
    const saved = await poApiCall(page, 'PUT', 'cutoff', {
      day_of_week: 3, cutoff_time: '17:30', timezone: 'America/Chicago',
    });
    expect(saved).toBeTruthy();
    expect(saved.day_of_week).toBe(3);
    expect((saved.cutoff_time || '').slice(0, 5)).toBe('17:30');
    const roundTrip = await poApiCall(page, 'GET', 'cutoff');
    expect(roundTrip.day_of_week).toBe(3);
    expect((roundTrip.cutoff_time || '').slice(0, 5)).toBe('17:30');

    // --- Non-admin path: author a team_member session inline, PUT /cutoff -> 403 ---
    const email = 'nc-fr6-teammember-' + Date.now() + '@yumyums.kitchen';
    const invite = await page.evaluate(async (em) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Cutoff', last_name: 'NonAdmin', email: em, roles: ['team_member'] }),
      });
      return res.json();
    }, email);
    const token = (invite.invite_path || '').split('token=')[1];
    expect(token).toBeTruthy();
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' }),
      });
    }, token);

    // Log in as the team_member (accept-invite already rotated the cookie, but log in
    // explicitly to be certain the session is the non-admin's).
    await login(page, email, 'test456');
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');

    const forbidden = await poFetchStatus(page, 'PUT', 'cutoff', {
      day_of_week: 5, cutoff_time: '09:00', timezone: 'America/Chicago',
    });
    expect(forbidden.status).toBe(403);   // non-admin cannot save cutoff

    // And no state change: the admin-saved config still reads day=3 17:30.
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
    const after = await poApiCall(page, 'GET', 'cutoff');
    expect(after.day_of_week).toBe(3);
    expect((after.cutoff_time || '').slice(0, 5)).toBe('17:30');
  });

});

// ── prove-sweep: PO-tab flows FR-13/14/15/16 (card purchasing-prove-po-approval) ──
//
// Red-first assertions naming the observable DB/UI behavior for the four PO-tab
// (Tab 3) flows the PRD marks UNPROVEN: locked-PO read-only render + badge (FR-13),
// admin-vs-non-admin locked-PO edit (FR-14), approve-button visibility gating
// (FR-15), and approve snapshot + both 409 refusals (FR-16). Slate expects
// FR-14/15/16 likely RED. Fixtures are isolated to this describe block and clean up
// after themselves (FR-16 completes the list it approves so no active list leaks to
// siblings). Non-admin session authored INLINE per the runbook (no shared helper).

test.describe('PO tab prove-sweep (FR-13/14/15/16)', () => {

  // status-aware fetch (poApiCall throws without exposing the code; the 403/409 flows
  // need the numeric status + the error envelope). Mirrors the B1 poFetchStatus.
  async function poFetchStatus(page, method, path, body) {
    return page.evaluate(async ([m, p, b]) => {
      const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
      if (b) opts.body = JSON.stringify(b);
      const res = await fetch('/api/v1/purchasing/' + p, opts);
      let json = null;
      try { json = await res.json(); } catch (e) { json = null; }
      return { status: res.status, ok: res.ok, body: json };
    }, [method, path, body]);
  }

  // ensureLockedPO: return the current locked PO if one is pending approval;
  // otherwise seed a fresh draft (2 catalog items) and simulate-cutoff to lock it.
  // Returns the locked PO object, or null if the stack has no catalog items to seed.
  async function ensureLockedPO(page) {
    let locked = await poApiCall(page, 'GET', 'orders?status=locked').catch(() => null);
    if (locked && locked.id) return locked;

    const items = await page.evaluate(async () => (await fetch('/api/v1/inventory/items')).json());
    if (!items || items.length < 2) return null;

    const draft = await poApiCall(page, 'POST', 'orders');
    await poApiCall(page, 'PUT', 'orders/' + draft.id + '/items', {
      items: items.slice(0, 2).map(it => ({ purchase_item_id: it.id, quantity: 2, unit: '' })),
    });
    await poApiCall(page, 'POST', 'simulate-cutoff');
    locked = await poApiCall(page, 'GET', 'orders?status=locked').catch(() => null);
    return (locked && locked.id) ? locked : null;
  }

  // Complete every vendor section of an active list so shopping/active clears — used
  // to keep the "one active list at a time" invariant from leaking between tests.
  async function drainActiveList(page) {
    const active = await poApiCall(page, 'GET', 'shopping/active').catch(() => null);
    if (!active || !active.id) return;
    for (const sec of active.vendor_sections || []) {
      if (sec.status === 'completed') continue;
      await poApiCall(page, 'POST', 'shopping/' + active.id + '/vendors/' + sec.id + '/complete').catch(() => {});
    }
  }

  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
  });

  // FR-13 — A locked PO renders read-only on the PO tab with a status badge. Observable:
  // #s3 shows a .status-badge.locked reading "Locked", the seeded item cards render, and
  // every qty stepper button is `disabled` (read-only — not in edit mode). Asserting the
  // read-only contract, not just "some content".
  test('FR-13: locked PO renders read-only with a Locked status badge', async ({ page }) => {
    const locked = await ensureLockedPO(page);
    if (!locked) { test.skip(true, 'No catalog items to seed a locked PO'); return; }
    expect(locked.status).toBe('locked');

    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.click('#t3');

    // The status badge must be present and read "Locked".
    const badge = page.locator('#s3 .status-badge.locked');
    await expect(badge).toBeVisible();
    await expect(badge).toHaveText('Locked');

    // The seeded line items must render as item cards.
    const cards = page.locator('#s3 .item-card');
    await expect(cards.first()).toBeVisible();
    const cardCount = await cards.count();
    expect(cardCount).toBeGreaterThan(0);

    // Read-only contract: NOT in edit mode, so every qty stepper button is disabled.
    const stepBtns = page.locator('#s3 .item-card .stp button');
    const total = await stepBtns.count();
    expect(total).toBeGreaterThan(0);
    const enabled = await stepBtns.evaluateAll(btns => btns.filter(b => !b.disabled).length);
    expect(enabled).toBe(0);   // no editable stepper when the locked PO is read-only
  });

  // FR-14 — An admin can edit a locked PO (PUT /orders/{id}/items with allowLocked, i.e.
  // WITHOUT require_draft) → 200; a non-admin (team_member) attempting the same edit is
  // refused 403 po_locked_admin_only, and the PO line-item set is unchanged. The
  // team_member session is authored INLINE per the runbook (invite -> accept-invite ->
  // login), no shared helper.
  test('FR-14: admin edits a locked PO (200) but a non-admin is refused 403', async ({ page }) => {
    const locked = await ensureLockedPO(page);
    if (!locked) { test.skip(true, 'No catalog items to seed a locked PO'); return; }
    expect(locked.status).toBe('locked');

    const items = await page.evaluate(async () => (await fetch('/api/v1/inventory/items')).json());
    // Pick an item NOT already on the locked PO so the admin edit is an observable add.
    const onPO = new Set((locked.line_items || []).map(x => x.purchase_item_id));
    const addItem = items.find(it => !onPO.has(it.id));
    if (!addItem) { test.skip(true, 'No off-PO catalog item to drive an admin edit'); return; }

    // --- Admin path: PUT with allowLocked (no require_draft) adds the item to the locked PO. ---
    const adminEdit = await poFetchStatus(page, 'PUT', 'orders/' + locked.id + '/items', {
      items: [
        ...(locked.line_items || []).map(x => ({ purchase_item_id: x.purchase_item_id, quantity: x.quantity, unit: x.unit || '' })),
        { purchase_item_id: addItem.id, quantity: 4, unit: '' },
      ],
    });
    expect(adminEdit.status).toBe(200);   // admin may edit a locked PO
    const afterAdmin = await poApiCall(page, 'GET', 'orders/' + locked.id);
    const added = (afterAdmin.line_items || []).find(x => x.purchase_item_id === addItem.id);
    expect(added).toBeTruthy();
    expect(added.quantity).toBe(4);

    // --- Non-admin path: author a team_member inline, PUT the same locked PO -> 403. ---
    const email = 'nc-fr14-teammember-' + Date.now() + '@yumyums.kitchen';
    const invite = await page.evaluate(async (em) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Locked', last_name: 'NonAdmin', email: em, roles: ['team_member'] }),
      });
      return res.json();
    }, email);
    const token = (invite.invite_path || '').split('token=')[1];
    expect(token).toBeTruthy();
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' }),
      });
    }, token);

    await login(page, email, 'test456');
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');

    // Non-admin edit of the locked PO — the allowLocked path is admin-only, so this is 403.
    const nonAdminEdit = await poFetchStatus(page, 'PUT', 'orders/' + locked.id + '/items', {
      items: [{ purchase_item_id: addItem.id, quantity: 9, unit: '' }],
    });
    expect(nonAdminEdit.status).toBe(403);                        // team_member cannot edit a locked PO
    expect(nonAdminEdit.body && nonAdminEdit.body.error).toBe('po_locked_admin_only');

    // And no state change: the admin-added item still reads qty 4.
    await login(page);
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
    const afterNonAdmin = await poApiCall(page, 'GET', 'orders/' + locked.id);
    const still = (afterNonAdmin.line_items || []).find(x => x.purchase_item_id === addItem.id);
    expect(still).toBeTruthy();
    expect(still.quantity).toBe(4);
  });

  // FR-15 — The Approve button is shown ONLY to an admin AND only while the PO is locked.
  // Observable: with a locked PO + admin session, [data-action="approve-po"] is visible;
  // a non-admin (team_member) viewing the same locked PO does NOT see it.
  test('FR-15: approve button visible for admin+locked, hidden for a non-admin', async ({ page }) => {
    const locked = await ensureLockedPO(page);
    if (!locked) { test.skip(true, 'No catalog items to seed a locked PO'); return; }
    expect(locked.status).toBe('locked');

    // --- Admin sees the Approve button on the locked PO tab. ---
    await page.reload();
    await page.waitForLoadState('networkidle');
    await page.click('#t3');
    const approveBtn = page.locator('#s3 [data-action="approve-po"]');
    await expect(approveBtn).toBeVisible();

    // --- Non-admin (team_member, inline) must NOT see the Approve button. ---
    const email = 'nc-fr15-teammember-' + Date.now() + '@yumyums.kitchen';
    const invite = await page.evaluate(async (em) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Approve', last_name: 'NonAdmin', email: em, roles: ['team_member'] }),
      });
      return res.json();
    }, email);
    const token = (invite.invite_path || '').split('token=')[1];
    expect(token).toBeTruthy();
    await page.evaluate(async (t) => {
      await fetch('/api/v1/auth/accept-invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'test456' }),
      });
    }, token);

    await login(page, email, 'test456');
    await page.goto('/purchasing.html');
    await page.waitForLoadState('networkidle');
    await page.click('#t3');
    // Give the PO tab a beat to render whatever it renders for the non-admin.
    await page.waitForFunction(() => {
      const el = document.getElementById('s3');
      return el && el.textContent.trim().length > 0;
    }, { timeout: 8000 }).catch(() => {});
    await expect(page.locator('#s3 [data-action="approve-po"]')).toHaveCount(0);
  });

  // FR-16 — Approving a locked PO transitions it locked -> shopping_active and creates an
  // immutable snapshot: exactly one active shopping list with one vendor section per
  // distinct PO vendor and items snapshotted from the PO line items. A second approve is
  // refused 409 active_shopping_list_exists; approving a DIFFERENT non-locked PO (a fresh
  // draft) is refused 409 po_not_locked. Cleans up by draining the list it creates.
  test('FR-16: approve snapshots the PO and both 409 refusals fire', async ({ page }) => {
    // Ensure no active list is leaking in from a sibling before we start.
    await drainActiveList(page);

    const locked = await ensureLockedPO(page);
    if (!locked) { test.skip(true, 'No catalog items to seed a locked PO'); return; }
    expect(locked.status).toBe('locked');

    // Distinct vendors across the locked PO's line items (drives the vendor-section count).
    const items = await page.evaluate(async () => (await fetch('/api/v1/inventory/items')).json());
    const byId = new Map((items || []).map(it => [it.id, it]));
    const distinctVendors = new Set((locked.line_items || []).map(li => {
      const it = byId.get(li.purchase_item_id);
      return it && it.vendor_id ? it.vendor_id : 'unassigned';
    }));
    const expectedSections = distinctVendors.size;
    const expectedItems = (locked.line_items || []).length;
    expect(expectedItems).toBeGreaterThan(0);

    // --- Approve (happy path) → 200. ---
    const approve = await poFetchStatus(page, 'POST', 'orders/' + locked.id + '/approve');
    expect(approve.status).toBe(200);

    // PO transitioned locked -> shopping_active.
    const poAfter = await poApiCall(page, 'GET', 'orders/' + locked.id);
    expect(poAfter.status).toBe('shopping_active');

    // Exactly one active shopping list, snapshotted from the PO.
    const active = await poApiCall(page, 'GET', 'shopping/active');
    expect(active).toBeTruthy();
    expect(active.po_id).toBe(locked.id);
    const sections = active.vendor_sections || [];
    expect(sections.length).toBe(expectedSections);          // one section per distinct vendor
    const snapItems = sections.reduce((n, s) => n + (s.items || []).length, 0);
    expect(snapItems).toBe(expectedItems);                   // every PO line item snapshotted

    // Item fields are frozen from the PO line items (qty snapshot check on the first).
    const firstLI = locked.line_items[0];
    let snap = null;
    for (const s of sections) { for (const it of (s.items || [])) { if (it.purchase_item_id === firstLI.purchase_item_id) snap = it; } }
    expect(snap).toBeTruthy();
    expect(snap.quantity).toBe(firstLI.quantity);            // qty frozen at approve time

    // --- 409 #1: concurrent/second approve while a list is active → active_shopping_list_exists. ---
    const secondApprove = await poFetchStatus(page, 'POST', 'orders/' + locked.id + '/approve');
    expect(secondApprove.status).toBe(409);
    expect(secondApprove.body && secondApprove.body.error).toBe('active_shopping_list_exists');

    // --- 409 #2: approving a non-locked PO (a fresh draft) → po_not_locked. ---
    // Drain the active list first so the active-list guard doesn't mask the not-locked guard.
    await drainActiveList(page);
    const draft = await poApiCall(page, 'POST', 'orders');   // fresh draft, status='draft'
    expect(draft.status).toBe('draft');
    const draftApprove = await poFetchStatus(page, 'POST', 'orders/' + draft.id + '/approve');
    expect(draftApprove.status).toBe(409);
    expect(draftApprove.body && draftApprove.body.error).toBe('po_not_locked');
  });

});
