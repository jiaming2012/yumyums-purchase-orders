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

async function usersApiCall(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch('/api/v1/users/' + p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path, body]);
}

// waitForUserList waits for the user list to finish loading (rows or empty state).
async function waitForUserList(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('user-list');
    if (!el) return false;
    return el.querySelector('.row') || el.querySelector('.empty') || el.querySelector('.error-msg');
  });
}

// waitForEditCard waits for the edit card to render a form.
// We wait for .form-wrap or .invite-link-panel to appear inside #edit-card.
// s2 visibility is NOT checked here — show(2) is synchronous so the content
// and visibility arrive in the same microtask after editUser() completes.
async function waitForEditCard(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('edit-card');
    if (!el) return false;
    return el.querySelector('.form-wrap') || el.querySelector('.invite-link-panel');
  });
}

// ─── User List ────────────────────────────────────────────────────────────────

test.describe('User List', () => {
  test('shows admin user in user list on load', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);

    // The superadmin should appear in the team list
    await expect(page.locator('#user-list')).toContainText('Jamal');
  });

  test('shows skeleton loading state before data arrives', async ({ page }) => {
    await login(page);

    // Intercept users API to delay response
    await page.route('/api/v1/users', async route => {
      await new Promise(r => setTimeout(r, 500));
      await route.continue();
    });

    await page.goto('/users.html');

    // Skeleton rows should appear while loading
    const skeleton = page.locator('#user-list .skeleton-row').first();
    await expect(skeleton).toBeVisible({ timeout: 3000 });

    // After load completes, rows should replace skeletons
    await waitForUserList(page);
    await expect(page.locator('#user-list .row').first()).toBeVisible();
  });
});

// ─── Invite Flow ─────────────────────────────────────────────────────────────

test.describe('Invite Flow', () => {
  test('can create new user via invite form', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);

    // Tap "Add Crew Member"
    await page.click('[data-action="show-invite"]');
    await waitForEditCard(page);

    // Fill the invite form
    const ts = Date.now();
    const email = `invite.test.${ts}@yumyums.kitchen`;
    await page.fill('#f-first', 'Tester');
    await page.fill('#f-last', 'McTest');
    await page.fill('#f-email', email);
    await page.selectOption('#f-role', 'team_member');

    // Submit
    await page.click('[data-action="submit-invite"]');

    // Invite link panel should appear
    await page.waitForFunction(() => {
      const el = document.getElementById('edit-card');
      return el && el.querySelector('.invite-link-panel');
    });
    await expect(page.locator('.invite-link-panel h2')).toContainText('Invite Link');
    await expect(page.locator('.invite-url')).toContainText('/login.html?token=');

    // User should appear in list with "Invited" text
    // (user-list is in s1 which may be display:none while s2 is showing — check text content not visibility)
    await expect(page.locator('#user-list')).toContainText('Tester');
    await expect(page.locator('#user-list')).toContainText('Invited');
  });

  test('invite link copy button works', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);

    await page.click('[data-action="show-invite"]');
    await waitForEditCard(page);

    await page.fill('#f-first', 'Copy');
    await page.fill('#f-last', 'Test');
    await page.fill('#f-email', `copy.test.${Date.now()}@yumyums.kitchen`);

    await page.click('[data-action="submit-invite"]');
    await page.waitForFunction(() => {
      const el = document.getElementById('edit-card');
      return el && el.querySelector('.invite-link-panel');
    });

    const copyBtn = page.locator('[data-action="copy-link"]');
    await copyBtn.click();

    // Button text should temporarily change to "Copied!"
    await expect(copyBtn).toHaveText('Copied!', { timeout: 2000 });
  });
});

// ─── Accept Invite Flow ───────────────────────────────────────────────────────

test.describe('Accept Invite Flow', () => {
  test('accept-invite page shows welcome heading', async ({ page }) => {
    await login(page);

    // Create a user via API and get the invite path
    const result = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'Welcome',
      last_name: 'Test',
      email: `welcome.test.${Date.now()}@yumyums.kitchen`,
      role: 'team_member',
    });
    expect(result.invite_path).toBeTruthy();
    const invitePath = result.invite_path; // e.g. /login.html?token=...

    // Clear session cookies to simulate unauthenticated state
    await page.context().clearCookies();

    // Navigate to the accept-invite URL
    await page.goto(invitePath);

    // Welcome heading should appear with the user's first name
    await expect(page.locator('#welcome-heading')).toContainText('Welcome, Welcome');

    // Password fields should be visible, email field should NOT be visible
    await expect(page.locator('#invite-pw')).toBeVisible();
    await expect(page.locator('#invite-pw2')).toBeVisible();
    await expect(page.locator('#login-form')).not.toBeVisible();
  });

  test('accept-invite sets password and redirects to index', async ({ page }) => {
    await login(page);

    const email = `accept.test.${Date.now()}@yumyums.kitchen`;
    const result = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'Accept',
      last_name: 'Flow',
      email,
      role: 'team_member',
    });
    expect(result.invite_path).toBeTruthy();

    // Clear session and navigate to invite page
    await page.context().clearCookies();
    await page.goto(result.invite_path);

    // Wait for the invite form to become visible (async showAcceptInviteMode must complete)
    await expect(page.locator('#invite-form')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('#invite-pw')).toBeVisible();

    // Set password using the specific submit button inside #invite-form
    const newPassword = 'TestPass123';
    await page.fill('#invite-pw', newPassword);
    await page.fill('#invite-pw2', newPassword);
    await page.locator('#invite-form button.btn').click();

    // Should redirect to index.html
    await page.waitForURL(url => url.pathname.includes('index.html') || url.pathname === '/');
  });

  test('expired token shows error on accept-invite page', async ({ page }) => {
    // Navigate to login with an invalid token (not logged in)
    await page.goto('/login.html?token=invalid_token_that_does_not_exist');

    // Error message should appear
    await expect(page.locator('#err')).toBeVisible();
    await expect(page.locator('#err')).toContainText('expired');
  });
});

// ─── Edit User ────────────────────────────────────────────────────────────────

test.describe('Edit User', () => {
  test('can edit user name and role', async ({ page }) => {
    await login(page);

    // Create a user to edit
    const ts = Date.now();
    const inviteResult = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'EditMe',
      last_name: 'Please',
      email: `edit.me.${ts}@yumyums.kitchen`,
      role: 'team_member',
    });
    expect(inviteResult.user).toBeTruthy();
    const userId = inviteResult.user.id;

    await page.goto('/users.html');
    await waitForUserList(page);

    // Verify the user appears in the list before clicking
    await expect(page.locator('#user-list')).toContainText('EditMe');

    // Click on the user row
    const userRow = page.locator(`[data-action="edit-user"][data-user-id="${userId}"]`);
    await userRow.click();
    await waitForEditCard(page);

    // Change the first name
    await page.fill('#f-first', 'EditedName');
    await page.click('[data-action="save-user"]');

    // Should return to user list with updated name
    await waitForUserList(page);
    await expect(page.locator('#user-list')).toContainText('EditedName');
  });

  test('nickname collision shows 409 error', async ({ page }) => {
    await login(page);
    const ts = Date.now();

    // Create two users
    const user1 = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'NickOne',
      last_name: 'Test',
      email: `nick1.${ts}@yumyums.kitchen`,
      role: 'team_member',
    });
    const user2 = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'NickTwo',
      last_name: 'Test',
      email: `nick2.${ts}@yumyums.kitchen`,
      role: 'team_member',
    });
    expect(user1.user).toBeTruthy();
    expect(user2.user).toBeTruthy();

    // Set a nickname on user1 via API so it exists
    await page.evaluate(async ([id]) => {
      await fetch(`/api/v1/users/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'NickOne', last_name: 'Test', nickname: 'UniqueName', role: 'team_member' }),
      });
    }, [user1.user.id]);

    await page.goto('/users.html');
    await waitForUserList(page);

    // Open user2 for editing
    const userRow = page.locator(`[data-action="edit-user"][data-user-id="${user2.user.id}"]`);
    await userRow.click();
    await waitForEditCard(page);

    // Try to set user2's nickname to user1's nickname
    await page.fill('#f-nick', 'UniqueName');
    await page.click('[data-action="save-user"]');

    // Nickname error should appear
    const nickErr = page.locator('#nick-err');
    await expect(nickErr).toBeVisible();
    await expect(nickErr).toContainText('taken');
  });
});

// ─── Destructive Actions ──────────────────────────────────────────────────────

test.describe('Destructive Actions', () => {
  test('force logout revokes user sessions', async ({ page }) => {
    await login(page);
    const ts = Date.now();

    // Create a user
    const inviteResult = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'ForceOut',
      last_name: 'User',
      email: `force.out.${ts}@yumyums.kitchen`,
      role: 'team_member',
    });
    expect(inviteResult.user).toBeTruthy();
    const userId = inviteResult.user.id;

    await page.goto('/users.html');
    await waitForUserList(page);

    // Open user for editing
    const userRow = page.locator(`[data-action="edit-user"][data-user-id="${userId}"]`);
    await userRow.click();
    await waitForEditCard(page);

    // Force logout with dialog confirmation
    page.once('dialog', dialog => dialog.accept());
    await page.click('[data-action="force-logout"]');

    // Should return to user list without error
    await waitForUserList(page);
    // Toast should show session revoked
    const toast = page.locator('#toast');
    await expect(toast).toContainText('revoked', { timeout: 3000 });
  });

  test('delete user removes from list', async ({ page }) => {
    await login(page);
    const ts = Date.now();
    const uniqueName = `Del${ts}`;

    const inviteResult = await usersApiCall(page, 'POST', 'invite', {
      first_name: uniqueName,
      last_name: 'User',
      email: `delete.me.${ts}@yumyums.kitchen`,
      role: 'team_member',
    });
    expect(inviteResult.user).toBeTruthy();
    const userId = inviteResult.user.id;

    await page.goto('/users.html');
    await waitForUserList(page);

    // Verify user appears in list
    await expect(page.locator('#user-list')).toContainText(uniqueName);

    // Open edit form
    const userRow = page.locator(`[data-action="edit-user"][data-user-id="${userId}"]`);
    await userRow.click();
    await waitForEditCard(page);

    // Delete with dialog confirmation
    page.once('dialog', dialog => dialog.accept());
    await page.click('[data-action="delete-user"]');

    // Should return to user list
    await waitForUserList(page);

    // This specific user should no longer appear (using the row selector)
    await expect(page.locator(`[data-action="edit-user"][data-user-id="${userId}"]`)).not.toBeVisible();
  });
});

// ─── Password Reset ───────────────────────────────────────────────────────────

test.describe('Password Reset', () => {
  test('reset password generates new token link', async ({ page }) => {
    await login(page);
    const ts = Date.now();

    // Create a user to reset
    const inviteResult = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'ResetPw',
      last_name: 'User',
      email: `reset.pw.${ts}@yumyums.kitchen`,
      role: 'team_member',
    });
    expect(inviteResult.user).toBeTruthy();
    const userId = inviteResult.user.id;

    await page.goto('/users.html');
    await waitForUserList(page);

    // Open edit form
    const userRow = page.locator(`[data-action="edit-user"][data-user-id="${userId}"]`);
    await userRow.click();
    await waitForEditCard(page);

    // Tap "Reset Password"
    await page.click('[data-action="reset-password"]');

    // Password Reset Link panel should appear
    await page.waitForFunction(() => {
      const el = document.getElementById('edit-card');
      return el && el.querySelector('.invite-link-panel');
    });
    await expect(page.locator('.invite-link-panel h2')).toContainText('Password Reset Link');
    await expect(page.locator('.invite-url')).toContainText('/login.html?token=');
  });
});

// ─── Access tab ──────────────────────────────────────────────────────────────

test.describe('Access tab', () => {
  test('shows all apps without needing to select a user', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);
    // Click Access tab directly — no user selected
    await page.click('#t3');
    await page.waitForFunction(() => {
      const s3 = document.getElementById('s3');
      return s3 && s3.querySelector('.access-card');
    });
    // Should show "App Permissions" header, not "Select a user first"
    await expect(page.locator('#s3')).toContainText('App Permissions');
    await expect(page.locator('#s3')).not.toContainText('Select a user first');
    // Should have at least one app card with role toggles
    const cards = await page.locator('.access-card').count();
    expect(cards).toBeGreaterThanOrEqual(1);
    // Each card should have role toggle checkboxes
    const toggles = await page.locator('[data-action="toggle-perm"]').count();
    expect(toggles).toBeGreaterThanOrEqual(3); // at least 3 roles per app
  });

  test('can toggle a role permission on an app', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);
    await page.click('#t3');
    await page.waitForFunction(() => {
      const s3 = document.getElementById('s3');
      return s3 && s3.querySelector('.access-card');
    });
    // Find the first toggle checkbox and get its initial state
    const firstToggle = page.locator('[data-action="toggle-perm"]').first();
    const wasChecked = await firstToggle.isChecked();
    // Toggle it via its parent label (toggle switch)
    await firstToggle.evaluate(el => el.click());
    await page.waitForTimeout(300);
    // Verify it changed
    const isNowChecked = await firstToggle.isChecked();
    expect(isNowChecked).toBe(!wasChecked);
    // Toggle it back to restore state
    await firstToggle.evaluate(el => el.click());
    await page.waitForTimeout(300);
    const restoredState = await firstToggle.isChecked();
    expect(restoredState).toBe(wasChecked);
  });
});
