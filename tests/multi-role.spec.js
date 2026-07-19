const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

async function login(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email || ADMIN_EMAIL);
  await page.fill('input[type="password"]', password || ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

async function apiCall(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch(p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path, body]);
}

// Generate a unique email to avoid conflicts between test runs
function uniqueEmail() {
  return `test-multi-role-${Date.now()}@yumyums.kitchen`;
}

test.describe('Multi-role support', () => {

  test('invite user with multiple roles returns roles as array', async ({ page }) => {
    await login(page);
    const email = uniqueEmail();
    const result = await apiCall(page, 'POST', '/api/v1/users/invite', {
      first_name: 'Multi',
      last_name: 'Roler',
      email: email,
      roles: ['manager', 'team_member'],
    });
    expect(result).toBeTruthy();
    expect(result.user).toBeTruthy();
    expect(Array.isArray(result.user.roles)).toBe(true);
    expect(result.user.roles).toContain('manager');
    expect(result.user.roles).toContain('team_member');
    // Cleanup
    await apiCall(page, 'DELETE', `/api/v1/users/${result.user.id}`);
  });

  test('list users returns roles as array not string', async ({ page }) => {
    await login(page);
    const users = await apiCall(page, 'GET', '/api/v1/users');
    expect(Array.isArray(users)).toBe(true);
    expect(users.length).toBeGreaterThan(0);
    // Every user should have roles as an array
    for (const u of users) {
      expect(Array.isArray(u.roles)).toBe(true);
    }
  });

  test('update user roles returns updated roles array', async ({ page }) => {
    await login(page);
    // Create a user to update
    const email = uniqueEmail();
    const created = await apiCall(page, 'POST', '/api/v1/users/invite', {
      first_name: 'Role',
      last_name: 'Update',
      email: email,
      roles: ['team_member'],
    });
    const userId = created.user.id;
    // Update to admin + manager
    const updated = await apiCall(page, 'PATCH', `/api/v1/users/${userId}`, {
      roles: ['admin', 'manager'],
    });
    expect(Array.isArray(updated.roles)).toBe(true);
    expect(updated.roles).toContain('admin');
    expect(updated.roles).toContain('manager');
    // Cleanup
    await apiCall(page, 'DELETE', `/api/v1/users/${userId}`);
  });

  test('onboarding template create with multiple roles returns roles array', async ({ page }) => {
    await login(page);
    const created = await apiCall(page, 'POST', '/api/v1/onboarding/createTemplate', {
      name: `Multi Role Test ${Date.now()}`,
      roles: ['line_cook', 'cashier'],
      sections: [],
    });
    expect(created).toBeTruthy();
    expect(created.id).toBeTruthy();
    // Fetch the template back
    const tmpl = await apiCall(page, 'GET', `/api/v1/onboarding/templates/${created.id}`);
    expect(Array.isArray(tmpl.roles)).toBe(true);
    expect(tmpl.roles).toContain('line_cook');
    expect(tmpl.roles).toContain('cashier');
    // No delete endpoint for templates — verify only
  });

  test('onboarding template update roles from single to multi', async ({ page }) => {
    await login(page);
    // Create with single role
    const created = await apiCall(page, 'POST', '/api/v1/onboarding/createTemplate', {
      name: `Role Update Test ${Date.now()}`,
      roles: ['manager'],
      sections: [],
    });
    expect(created.id).toBeTruthy();
    // Update to multiple roles
    await apiCall(page, 'PUT', `/api/v1/onboarding/updateTemplate/${created.id}`, {
      name: `Role Update Test`,
      roles: ['manager', 'admin'],
      sections: [],
    });
    // Fetch back and verify
    const tmpl = await apiCall(page, 'GET', `/api/v1/onboarding/templates/${created.id}`);
    expect(Array.isArray(tmpl.roles)).toBe(true);
    expect(tmpl.roles).toContain('manager');
    expect(tmpl.roles).toContain('admin');
    // Cleanup - delete via update is not available; just verify and leave
    // (no delete endpoint exists in handler — acceptable)
  });

  test('me endpoint returns roles as array', async ({ page }) => {
    await login(page);
    const me = await apiCall(page, 'GET', '/api/v1/me');
    expect(me).toBeTruthy();
    expect(Array.isArray(me.roles)).toBe(true);
    expect(me.roles.length).toBeGreaterThan(0);
  });

  test('users.html role chips render for edit form', async ({ page }) => {
    await login(page);
    // Load users page
    await page.goto('/users.html');
    // Wait for user list to load
    await page.waitForFunction(() => {
      const ul = document.getElementById('user-list');
      return ul && (ul.querySelector('.row') || ul.querySelector('.empty'));
    });
    // Click first user row
    const firstRow = page.locator('#user-list .row[data-action="edit-user"]').first();
    await firstRow.click();
    // Wait for edit card to render
    await page.waitForFunction(() => {
      const card = document.getElementById('edit-card');
      return card && card.querySelector('#f-roles');
    });
    // Verify role chips are present (users.html uses <button class="role-chip">
    // not native <input type="checkbox">; see users.html:230 / :338).
    const chips = page.locator('#f-roles .role-chip');
    await expect(chips).toHaveCount(3);
    const values = await chips.evaluateAll(btns => btns.map(b => b.dataset.role));
    expect(values).toContain('admin');
    expect(values).toContain('manager');
    expect(values).toContain('team_member');
  });

  test('users.html invite form shows role chips with team_member selected by default', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    // Wait for user list to load
    await page.waitForFunction(() => {
      const ul = document.getElementById('user-list');
      return ul && (ul.querySelector('.row') || ul.querySelector('.empty') || ul.querySelector('button'));
    });
    // Click "Add Crew Member" button
    const addBtn = page.locator('[data-action="show-invite"]');
    await addBtn.click();
    // Wait for invite form
    await page.waitForFunction(() => {
      const card = document.getElementById('edit-card');
      return card && card.querySelector('#f-roles');
    });
    const chips = page.locator('#f-roles .role-chip');
    await expect(chips).toHaveCount(3);
    // team_member chip should be "on" by default (users.html:233 hardcodes it).
    const teamMemberChip = page.locator('#f-roles .role-chip[data-role="team_member"]');
    await expect(teamMemberChip).toHaveClass(/on/);
  });

  test('onboarding.html builder shows role chips', async ({ page }) => {
    await login(page);
    await page.goto('/onboarding.html');
    // Wait for page to init
    await page.waitForFunction(() => document.getElementById('t3') !== null);
    // Switch to Builder tab
    await page.click('#t3');
    // Wait for builder body
    await page.waitForFunction(() => {
      const body = document.getElementById('builder-body');
      return body && (body.querySelector('[data-action="new-template"]') || body.querySelector('[data-action="open-template"]') || body.querySelector('.empty'));
    });
    // Open an existing template (skip create-flow since it triggers a prompt).
    const openBtn = page.locator('[data-action="open-template"]').first();
    const openCount = await openBtn.count();
    if (openCount === 0) {
      test.skip(true, 'No existing onboarding template — skipping role-chip render assertion');
      return;
    }
    await openBtn.click();
    // Wait for editor with role-chips (onboarding.html:1031 — .role-chips
    // container with .role-chip[data-action="tpl-role-chip"] buttons).
    await page.waitForFunction(() => {
      const body = document.getElementById('builder-body');
      return body && body.querySelector('[data-action="tpl-role-chip"]');
    }, { timeout: 5000 });
    const chips = page.locator('#builder-body [data-action="tpl-role-chip"]');
    // onboarding.html:1032 hard-codes 3 target roles (team_member, manager, admin).
    await expect(chips).toHaveCount(3);
  });

});
