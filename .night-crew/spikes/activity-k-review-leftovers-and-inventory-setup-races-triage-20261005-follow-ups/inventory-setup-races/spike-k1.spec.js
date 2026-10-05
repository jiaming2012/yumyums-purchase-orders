// spike-k1.spec.js — THROWAWAY spec for spike 01 of card K1 (inventory-setup-races, B-459 /
// B-478). Copied by the spike script into a worktree's tests/ directory and run there; never
// part of the suite. Self-contained (helpers re-stated from tests/inventory.spec.js).
//
// Nothing in the page is stubbed — only NETWORK TIMING is altered, through page.route, so the
// shipped inventory.html runs its real writers against a real server. Each test PASSES when the
// filed finding REPRODUCES (test 1, test 3) or the control is healthy (test 2).
const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

async function login(page) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL((url) => !url.pathname.includes('login'));
}
async function goTab(page, n) {
  await page.waitForSelector('#s0:visible, #back-hub:visible');
  const back = page.locator('#back-hub');
  if (await back.isVisible()) await back.click();
  await page.locator('#t' + n).click();
}
async function invApiCall(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch('/api/v1/inventory/' + p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path, body]);
}

test.describe('spike K1 — late responses in Inventory Setup (B-478, B-459)', () => {

  test('[K1-SPIKE-1] B-478 PREMISE: groups delayed 600 ms → a name typed right after opening Setup is wiped and create sends no POST', async ({ page }) => {
    await login(page);
    await page.route('**/api/v1/inventory/groups', async (route) => {
      await new Promise((r) => setTimeout(r, 600));
      await route.continue();
    });
    let posts = 0;
    page.on('request', (req) => { if (req.method() === 'POST' && req.url().includes('/api/v1/inventory/items')) posts += 1; });
    await page.goto('/inventory.html');
    await goTab(page, 7);
    await page.waitForSelector('#new-item-name', { timeout: 5000 });
    await page.fill('#new-item-name', 'Spike Typed Item');
    const before = await page.inputValue('#new-item-name');
    await page.waitForTimeout(1500); // the delayed groups response lands and the add bar re-renders
    const after = await page.inputValue('#new-item-name');
    page.on('dialog', async (d) => { await d.accept(); });
    await page.click('[data-action="create-item"]');
    await page.waitForTimeout(500);
    const diag = { before, after, posts, editForms: await page.locator('.item-edit-form').count() };
    console.log('SPIKE-K1-1: ' + JSON.stringify(diag));
    expect(before).toBe('Spike Typed Item');
    expect(after, 'the late re-render wiped the typed name').toBe('');
    expect(posts, 'the create click sent no POST').toBe(0);
  });

  test('[K1-SPIKE-2] CONTROL: no delay → the typed name survives 1.5 s after opening Setup', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html');
    await goTab(page, 7);
    await page.waitForSelector('#new-item-name', { timeout: 5000 });
    // Let loadItems() land first — on a loaded box its two fetches take well over a second,
    // so the control waits for the network to go quiet (what a manager who pauses does).
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(500);
    await page.fill('#new-item-name', 'Spike Control Item');
    await page.waitForTimeout(1500);
    const after = await page.inputValue('#new-item-name');
    console.log('SPIKE-K1-2: ' + JSON.stringify({ after, note: 'typed after networkidle' }));
    expect(after).toBe('Spike Control Item');
  });

  test('[K1-SPIKE-3] B-459 PREMISE: the Setup tab\'s first GET /items held until after an alias POST → the chips revert while the server keeps both', async ({ page }) => {
    await login(page);
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    const ts = Date.now();
    const groups = await invApiCall(page, 'GET', 'groups');
    const gid = groups && groups.length ? groups[0].id : null;
    const created = await invApiCall(page, 'POST', 'items', { description: 'Spike Chip Item ' + ts, group_id: gid });
    expect(created && created.id).toBeTruthy();
    const seededAlias = 'Seeded Nick ' + ts;
    await page.evaluate(async ([id, alias]) => {
      await fetch('/api/v1/inventory/items/aliases', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ purchase_item_id: id, alias }) });
    }, [created.id, seededAlias]);
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Arm: hold ONLY the first GET /items from now on (the one opening Setup fires); the alias
    // handler's own refetch passes through.
    let held = null;
    await page.route('**/api/v1/inventory/items', (route) => {
      const isGet = route.request().method() === 'GET';
      if (isGet && !held) { held = route; return; }
      route.continue();
    });
    await goTab(page, 7);
    await expect.poll(() => held !== null, { timeout: 5000 }).toBe(true);
    const row = page.locator('.item-row[data-id="' + created.id + '"]');
    await row.click();
    const form = page.locator('.item-edit-form[data-item-id="' + created.id + '"]');
    await expect(form).toBeVisible();
    await expect(form.locator('.alias-chip')).toContainText(seededAlias);
    const typedAlias = 'Typed Nick ' + ts;
    await form.locator('.item-alias-input').fill(typedAlias);
    await form.locator('button[data-action="add-item-alias"]').click();
    // Give the handler's own POST + refetch every chance to paint the second chip while the
    // stale first GET is still held; record what the view shows either way (not an assertion —
    // the finding is the server/view mismatch after release).
    let chipsWhileHeld = 0;
    for (let i = 0; i < 25; i += 1) {
      chipsWhileHeld = await form.locator('.alias-chip').count();
      if (chipsWhileHeld >= 2) break;
      await page.waitForTimeout(200);
    }
    // Release the stale, pre-POST response now.
    await held.continue();
    await page.waitForTimeout(1500);
    const chips = await form.locator('.alias-chip').count();
    const chipText = (await form.locator('.alias-chips').innerText().catch(() => '')).trim();
    const items = await invApiCall(page, 'GET', 'items');
    const it = (items || []).find((i) => i.id === created.id);
    const serverHasTyped = !!(it && (it.aliases || []).includes(typedAlias));
    const diag = { chipsWhileHeld, chipsAfterRelease: chips, viewHasTyped: chipText.includes(typedAlias), serverHasTyped, serverAliases: it ? it.aliases : null, addDropped: !serverHasTyped };
    console.log('SPIKE-K1-3: ' + JSON.stringify(diag));
    // CORRECTED PREMISE (runs 2 and 3 of this spike, 2026-10-05): B-459's diagnosed mechanism —
    // "the stale pre-POST GET /items lands after the handler's refetch and reverts the chips
    // while the server keeps both" — did NOT reproduce in either run. Run 2: view AND server both
    // held the typed nickname. Run 3, same timing: NEITHER did — the add never reached the server
    // (no POST persisted), a different failure. In no run did the view disagree with the server.
    // This leg records the invariant both runs showed and logs whether the add was dropped; the
    // card's B-459 half is therefore the add path's robustness + sequencing hardening, proven by
    // measurement, not the filed overwrite fix.
    expect(diag.viewHasTyped, 'the view and the server AGREE about the typed nickname (B-459\'s "view lies" mechanism does not reproduce)').toBe(serverHasTyped);
  });
});
