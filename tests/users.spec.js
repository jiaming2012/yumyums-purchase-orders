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
    // Roles: team_member chip is "on" by default in the invite form
    // (users.html:233). No interaction needed unless changing roles.
    await expect(page.locator('#f-roles .role-chip[data-role="team_member"]')).toHaveClass(/on/);

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
      roles: ['team_member'],
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
      roles: ['team_member'],
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
      roles: ['team_member'],
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
      roles: ['team_member'],
    });
    const user2 = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'NickTwo',
      last_name: 'Test',
      email: `nick2.${ts}@yumyums.kitchen`,
      roles: ['team_member'],
    });
    expect(user1.user).toBeTruthy();
    expect(user2.user).toBeTruthy();

    // Set a nickname on user1 via API so it exists
    await page.evaluate(async ([id]) => {
      await fetch(`/api/v1/users/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'NickOne', last_name: 'Test', nickname: 'UniqueName', roles: ['team_member'] }),
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
      roles: ['team_member'],
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
      roles: ['team_member'],
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
      roles: ['team_member'],
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

// ─── Invite link panel ───────────────────────────────────────────────────────

test.describe('Invite link panel', () => {
  test('email notice is prominent and appears above the invite link', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);

    // Open invite form
    await page.locator('[data-action="show-invite"]').click();
    await waitForEditCard(page);

    // Fill in the form
    await page.fill('#f-first', 'PanelTest');
    await page.fill('#f-last', 'User');
    const email = 'panel-test-' + Date.now() + '@yumyums.kitchen';
    await page.fill('#f-email', email);

    // Submit
    await page.locator('[data-action="submit-invite"]').click();

    // Wait for invite link panel
    await page.waitForFunction(() => {
      const el = document.getElementById('edit-card');
      return el && el.querySelector('.invite-link-panel');
    });

    // Email notice should exist and appear BEFORE the invite link textarea
    const emailNotice = page.locator('.invite-email-notice');
    await expect(emailNotice).toBeVisible();
    await expect(emailNotice).toContainText('invite link has been sent');

    // Email notice should come before the invite URL textarea in DOM order
    const noticeIndex = await page.evaluate(() => {
      const panel = document.querySelector('.invite-link-panel');
      const children = Array.from(panel.children);
      const noticeIdx = children.findIndex(el => el.classList.contains('invite-email-notice'));
      const urlIdx = children.findIndex(el => el.classList.contains('invite-url') || el.tagName === 'TEXTAREA');
      return { noticeIdx, urlIdx };
    });
    expect(noticeIndex.noticeIdx).toBeLessThan(noticeIndex.urlIdx);
  });
});

// ─── Access tab ──────────────────────────────────────────────────────────────

test.describe('Access tab', () => {
  test('shows all apps without needing to select a user', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);
    // Click Access tab directly — no user selected
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });
    // Should show "App Permissions" header, not "Select a user first"
    await expect(page.locator('#s2')).toContainText('App Permissions');
    await expect(page.locator('#s2')).not.toContainText('Select a user first');
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
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
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

// ─── Last Name Display ──────────────────────────────────────────────────────

test.describe('Last Name Display', () => {
  test('edit form shows full last name, not abbreviated', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);

    // Click the first user to open edit
    await page.click('[data-action="edit-user"]');
    await waitForEditCard(page);

    // Last name field should show the full last name, not "C." or similar abbreviation
    const lastNameValue = await page.locator('#f-last').inputValue();
    expect(lastNameValue.length).toBeGreaterThan(2);
    expect(lastNameValue).not.toMatch(/^[A-Z]\.$/);
  });
});

// ─── Alert Channel Defaults ─────────────────────────────────────────────────

test.describe('Alert Channel Defaults', () => {
  test('new user has both Zoho Cliq and Email enabled by default', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);

    // Create a new user via API
    const ts = Date.now();
    const email = `notif-default-${ts}@yumyums.kitchen`;
    const result = await page.evaluate(async (e) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'NotifTest', last_name: 'User', email: e, roles: ['team_member'] })
      });
      return res.json();
    }, email);
    expect(result.user).toBeTruthy();

    // Verify API returns both channels
    const user = result.user;
    expect(user.notification_channels).toBeTruthy();
    expect(user.notification_channels).toContain('zoho_cliq');
    expect(user.notification_channels).toContain('email');
    expect(user.notification_channels.length).toBe(2);
  });

  test('alert channel chips show both enabled when editing new user', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);

    // Create user
    const ts = Date.now();
    const email = `notif-chips-${ts}@yumyums.kitchen`;
    const result = await page.evaluate(async (e) => {
      const res = await fetch('/api/v1/users/invite', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'ChipTest', last_name: 'User', email: e, roles: ['team_member'] })
      });
      return res.json();
    }, email);
    const userId = result.user.id;

    // Open edit form for the new user
    await page.reload();
    await waitForUserList(page);
    await page.click(`[data-action="edit-user"][data-user-id="${userId}"]`);
    await waitForEditCard(page);

    // Both alert channel chips should have the "on" class
    const zohoChip = page.locator('#f-notif .role-chip[data-channel="zoho_cliq"]');
    const emailChip = page.locator('#f-notif .role-chip[data-channel="email"]');
    await expect(zohoChip).toHaveClass(/\bon\b/);
    await expect(emailChip).toHaveClass(/\bon\b/);
  });

  test('alert channel chips toggle on and off', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);

    // Open edit form for the first user
    await page.click('[data-action="edit-user"]');
    await waitForEditCard(page);

    const zohoChip = page.locator('#f-notif .role-chip[data-channel="zoho_cliq"]');
    const emailChip = page.locator('#f-notif .role-chip[data-channel="email"]');

    // Get initial states
    const zohoWasOn = await zohoChip.evaluate(el => el.classList.contains('on'));
    const emailWasOn = await emailChip.evaluate(el => el.classList.contains('on'));

    // Toggle Zoho off (or on)
    await zohoChip.click();
    const zohoAfter = await zohoChip.evaluate(el => el.classList.contains('on'));
    expect(zohoAfter).toBe(!zohoWasOn);

    // Toggle Email
    await emailChip.click();
    const emailAfter = await emailChip.evaluate(el => el.classList.contains('on'));
    expect(emailAfter).toBe(!emailWasOn);

    // Toggle both back to restore
    await zohoChip.click();
    await emailChip.click();
  });
});

// ─── Security enforcement (NFR-1..NFR-5) ────────────────────────────────────
//
// Prove-sweep card: users-prove-security. Red-first assertions for the five
// auth/permission NFRs. The non-admin (team_member) session is authored INLINE
// here per tests/multi-role.spec.js — no shared helper module (runbook rule).
//
// Non-admin session lifecycle (all inline, API-driven):
//   1. Log in as the superadmin.
//   2. Invite a team_member  → capture { user.id, invite_path } (token in path).
//   3. Clear the admin cookie, POST /api/v1/auth/accept-invite with the token +
//      a password → server activates the user AND sets the hq_session cookie in
//      THIS browser context. From that point `page` is the team_member.
// FIXTURE GOTCHA honored: for role-dependent facts we read the authoritative
// stored roles from the users-API (invite response / GET /api/v1/users), never
// the masked GET /api/v1/me.

const TEAM_MEMBER_PW = 'TeamMemberPass123';

// tokenFromInvitePath extracts the raw token from "/login.html?token=XYZ".
function tokenFromInvitePath(invitePath) {
  return new URL(invitePath, 'http://x').searchParams.get('token');
}

// inviteTeamMember (as the currently-logged-in admin) creates a status='invited'
// team_member and returns { id, email, invitePath, token }.
async function inviteTeamMember(page, tag) {
  const email = `sec-${tag}-${Date.now()}@yumyums.kitchen`;
  const res = await page.evaluate(async (e) => {
    const r = await fetch('/api/v1/users/invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Sec', last_name: 'Member', email: e, roles: ['team_member'] }),
    });
    return { status: r.status, body: await r.json() };
  }, email);
  expect(res.status).toBe(201);
  // Authoritative roles come from the users-API invite response, not /me.
  expect(res.body.user.roles).toEqual(['team_member']);
  return {
    id: res.body.user.id,
    email,
    invitePath: res.body.invite_path,
    token: tokenFromInvitePath(res.body.invite_path),
  };
}

// becomeTeamMember clears the admin session and activates+logs-in as the invited
// team_member by accepting their invite token. After this, `page` requests carry
// the team_member's hq_session cookie.
async function becomeTeamMember(page, token) {
  await page.context().clearCookies();
  const res = await page.evaluate(async (t) => {
    const r = await fetch('/api/v1/auth/accept-invite', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: 'TeamMemberPass123' }),
    });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, token);
  expect(res.status).toBe(200);
  return res.body;
}

test.describe('Security enforcement', () => {
  // ── NFR-1: every admin endpoint refuses a non-admin with 403 ──────────────
  test('NFR-1: non-admin (team_member) is refused 403 across all 8 admin endpoints', async ({ page }) => {
    await login(page); // admin
    const victim = await inviteTeamMember(page, 'nfr1-victim'); // a second user to target
    const me = await inviteTeamMember(page, 'nfr1-self');
    await becomeTeamMember(page, me.token); // page is now the team_member

    // Confirm the session is genuinely non-admin via the authoritative users-API
    // role, not the masked /me. (A team_member cannot list users, so we assert
    // the 403 on GET /users below IS the proof the guard fired.)
    const calls = await page.evaluate(async ([victimId]) => {
      const j = (r) => r.json().catch(() => null);
      const results = {};
      let r;
      r = await fetch('/api/v1/users');
      results.listUsers = { status: r.status, body: await j(r) };
      r = await fetch('/api/v1/users/invite', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ first_name: 'X', last_name: 'Y', email: 'z@z.z', roles: ['team_member'] }) });
      results.invite = { status: r.status, body: await j(r) };
      r = await fetch(`/api/v1/users/${victimId}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ first_name: 'Hacked' }) });
      results.patchUser = { status: r.status, body: await j(r) };
      r = await fetch(`/api/v1/users/${victimId}/reset-password`, { method: 'POST' });
      results.resetPassword = { status: r.status, body: await j(r) };
      r = await fetch(`/api/v1/users/${victimId}/revoke`, { method: 'POST' });
      results.revoke = { status: r.status, body: await j(r) };
      r = await fetch(`/api/v1/users/${victimId}`, { method: 'DELETE' });
      results.deleteUser = { status: r.status, body: await j(r) };
      r = await fetch('/api/v1/apps/permissions');
      results.getPerms = { status: r.status, body: await j(r) };
      r = await fetch('/api/v1/apps/purchasing/permissions', { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ role_grants: [], user_grants: [] }) });
      results.setPerms = { status: r.status, body: await j(r) };
      return results;
    }, [victim.id]);

    // All 8 admin handlers must refuse with 403 forbidden.
    for (const key of ['listUsers', 'invite', 'patchUser', 'resetPassword', 'revoke', 'deleteUser', 'getPerms', 'setPerms']) {
      expect(calls[key].status, `${key} should be 403`).toBe(403);
      expect(calls[key].body?.error, `${key} error body`).toBe('forbidden');
    }

    // And the refusal was real, not incidental: the victim was NOT patched/deleted.
    // Re-login as admin to verify the victim still exists unmodified.
    await page.context().clearCookies();
    await login(page);
    const victimAfter = await usersApiCall(page, 'GET', '');
    const found = victimAfter.find(u => u.id === victim.id);
    expect(found, 'victim still present after refused delete').toBeTruthy();
    expect(found.first_name).toBe('Sec'); // not "Hacked"

    // Cleanup
    await usersApiCall(page, 'DELETE', victim.id);
    await usersApiCall(page, 'DELETE', me.id);
  });

  // ── NFR-2: invite token is single-use (7-day expiry noted UNTESTABLE) ─────
  test('NFR-2: invite token is single-use — second accept-invite is refused 400 token_expired', async ({ page }) => {
    await login(page);
    const u = await inviteTeamMember(page, 'nfr2');

    // First accept activates the user and sets a session.
    await page.context().clearCookies();
    const first = await page.evaluate(async (t) => {
      const r = await fetch('/api/v1/auth/accept-invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'FirstPass123' }),
      });
      return { status: r.status, body: await r.json().catch(() => null) };
    }, u.token);
    expect(first.status).toBe(200);
    expect(first.body.user.status).toBe('active');

    // Second accept with the SAME token must be refused — token already used.
    await page.context().clearCookies();
    const second = await page.evaluate(async (t) => {
      const r = await fetch('/api/v1/auth/accept-invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token: t, password: 'SecondPass456' }),
      });
      return { status: r.status, body: await r.json().catch(() => null) };
    }, u.token);
    expect(second.status).toBe(400);
    expect(second.body?.error).toBe('token_expired');

    // Only one activation occurred: password from the SECOND attempt did not stick.
    // (We can't read the hash, but the second attempt returned no user + no session.)
    expect(second.body?.user).toBeFalsy();

    // NOTE: the 7-day EXPIRY-BOUNDARY leg (a token past expires_at → refused) is
    // UNTESTABLE here without server time-control: InsertInviteToken hardcodes
    // now()+7d and there is no fixture to backdate expires_at via the API. The
    // single-use guarantee is proven fully above; the expiry predicate shares the
    // same WHERE clause (used_at IS NULL AND expires_at > now()) in ClaimInviteToken.

    // Cleanup
    await login(page);
    await usersApiCall(page, 'DELETE', u.id);
  });

  // ── NFR-3: grant write → /me/apps read round-trip ─────────────────────────
  test('NFR-3: granting an app to a user makes it appear in that user\'s /me/apps, and removing it hides it', async ({ page }) => {
    await login(page);
    const u = await inviteTeamMember(page, 'nfr3');
    await becomeTeamMember(page, u.token); // activate + get session

    const slugsFor = async () => page.evaluate(async () => {
      const r = await fetch('/api/v1/me/apps');
      const apps = await r.json();
      return apps.map(a => a.slug);
    });

    // Baseline: a fresh team_member with no grants sees NO 'purchasing' app.
    const before = await slugsFor();
    expect(before).not.toContain('purchasing');

    // As admin, grant this specific user an individual user_grant on 'purchasing'.
    await page.context().clearCookies();
    await login(page);
    const putGrant = await page.evaluate(async ([uid]) => {
      const r = await fetch('/api/v1/apps/purchasing/permissions', {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role_grants: [], user_grants: [uid] }),
      });
      return { status: r.status, body: await r.json() };
    }, [u.id]);
    expect(putGrant.status).toBe(200);
    expect(putGrant.body.user_grants).toContain(u.id);

    // Back as the team_member: /me/apps now reflects the grant — 'purchasing' appears.
    await becomeTeamMember2(page, u.email);
    const afterGrant = await slugsFor();
    expect(afterGrant, 'purchasing appears after grant').toContain('purchasing');

    // As admin, revoke the grant (empty user_grants replaces the set).
    await page.context().clearCookies();
    await login(page);
    const putRevoke = await page.evaluate(async () => {
      const r = await fetch('/api/v1/apps/purchasing/permissions', {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role_grants: [], user_grants: [] }),
      });
      return { status: r.status, body: await r.json() };
    });
    expect(putRevoke.status).toBe(200);
    expect(putRevoke.body.user_grants).not.toContain(u.id);

    // Back as the team_member: 'purchasing' disappears again.
    await becomeTeamMember2(page, u.email);
    const afterRevoke = await slugsFor();
    expect(afterRevoke, 'purchasing disappears after revoke').not.toContain('purchasing');

    // Cleanup
    await page.context().clearCookies();
    await login(page);
    await usersApiCall(page, 'DELETE', u.id);
  });

  // ── NFR-4: notification-preference admin-or-self + ≥1 channel ─────────────
  test('NFR-4: notification-preference is admin-or-self and requires ≥1 channel', async ({ page }) => {
    await login(page);
    const self = await inviteTeamMember(page, 'nfr4-self');
    const other = await inviteTeamMember(page, 'nfr4-other');
    await becomeTeamMember(page, self.token); // page is now `self` (team_member)

    const notif = await page.evaluate(async ([selfId, otherId]) => {
      const j = (r) => r.json().catch(() => null);
      const out = {};
      let r;
      // self may SET own preference (admin-or-self allow branch)
      r = await fetch(`/api/v1/users/${selfId}/notification-preference`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ notification_channels: ['email'] }) });
      out.selfSet = { status: r.status, body: await j(r) };
      // self may READ own preference
      r = await fetch(`/api/v1/users/${selfId}/notification-preference`);
      out.selfGet = { status: r.status, body: await j(r) };
      // self setting an EMPTY channel set must be refused 400
      r = await fetch(`/api/v1/users/${selfId}/notification-preference`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ notification_channels: [] }) });
      out.selfEmpty = { status: r.status, body: await j(r) };
      // self touching ANOTHER user's preference must be refused 403 (not admin, not self)
      r = await fetch(`/api/v1/users/${otherId}/notification-preference`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ notification_channels: ['email'] }) });
      out.otherSet = { status: r.status, body: await j(r) };
      return out;
    }, [self.id, other.id]);

    // self-allow branch
    expect(notif.selfSet.status, 'self may set own pref').toBe(200);
    expect(notif.selfSet.body.notification_channels).toEqual(['email']);
    expect(notif.selfGet.status, 'self may read own pref').toBe(200);
    expect(notif.selfGet.body.notification_channels).toContain('email');
    // ≥1 channel required
    expect(notif.selfEmpty.status, 'empty channel set refused').toBe(400);
    // admin-or-self refuse branch
    expect(notif.otherSet.status, 'non-admin cannot set another user pref').toBe(403);
    expect(notif.otherSet.body?.error).toBe('forbidden');

    // admin CAN set another user's pref (admin branch of admin-or-self)
    await page.context().clearCookies();
    await login(page);
    const adminSet = await page.evaluate(async ([otherId]) => {
      const r = await fetch(`/api/v1/users/${otherId}/notification-preference`, { method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ notification_channels: ['zoho_cliq'] }) });
      return { status: r.status, body: await r.json().catch(() => null) };
    }, [other.id]);
    expect(adminSet.status, 'admin may set any user pref').toBe(200);

    // Cleanup
    await usersApiCall(page, 'DELETE', self.id);
    await usersApiCall(page, 'DELETE', other.id);
  });

  // ── NFR-5: an unauthenticated API call yields 401 and the UI redirects ────
  test('NFR-5: unauthenticated request to an admin endpoint returns 401 and the UI redirects to login', async ({ page }) => {
    // No login; ensure a clean unauthenticated context.
    await page.goto('/login.html');
    await page.context().clearCookies();

    // Raw API call with no session cookie → 401.
    const status = await page.evaluate(async () => {
      const r = await fetch('/api/v1/users');
      return r.status;
    });
    expect(status).toBe(401);

    // UI leg: users.html on an unauthenticated session redirects to /login.html
    // (users.html:139 — a 401 from its init fetch forces window.location to login).
    await page.goto('/users.html');
    await page.waitForURL(url => url.pathname.includes('login'), { timeout: 10000 });
    expect(page.url()).toContain('login');
  });
});

// becomeTeamMember2 re-establishes the team_member session by password login
// (the user is already active from an earlier accept-invite). Used when NFR-3
// needs to hop back to the team_member after admin operations.
async function becomeTeamMember2(page, email) {
  await page.context().clearCookies();
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', TEAM_MEMBER_PW);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// ─── UI + Access hardening (FR-2,4,6,9,10,11,15,16,17,18,19) ─────────────────
//
// Prove-sweep card: users-prove-ui-access. Red-first assertions that name the
// observable UI/DB behavior of the Users tool's UNPROVEN flows. All fixtures are
// admin-session, API-seeded (superadmin login), inline — no shared helper module.

test.describe('Users UI + Access hardening', () => {
  // ── FR-2: list load-failure renders the inline error state ────────────────
  test('FR-2: users list fetch failure renders "Could not load team." error state', async ({ page }) => {
    await login(page);
    // Force the list fetch to fail with a 500 BEFORE navigating to users.html.
    await page.route('**/api/v1/users', route => {
      route.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"internal_error"}' });
    });
    await page.goto('/users.html');
    // The load catch (users.html:213) calls renderError('user-list','Could not load team.')
    // which writes a .error-msg node — the observable error-branch behavior.
    const errMsg = page.locator('#user-list .error-msg');
    await expect(errMsg).toBeVisible({ timeout: 5000 });
    await expect(errMsg).toContainText('Could not load team.');
  });

  // ── FR-4: invite validation — 422 on missing fields, 409 on duplicate ─────
  test('FR-4: invite is refused 422 on empty fields and 409 on duplicate email', async ({ page }) => {
    await login(page);
    const ts = Date.now();
    const email = `dup.invite.${ts}@yumyums.kitchen`;

    // 422 validation_error: missing first_name.
    const empty = await page.evaluate(async (e) => {
      const r = await fetch('/api/v1/users/invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: '', last_name: 'NoFirst', email: e, roles: ['team_member'] }),
      });
      return { status: r.status, body: await r.json().catch(() => null) };
    }, `empty.${email}`);
    expect(empty.status).toBe(422);
    expect(empty.body.error).toBe('validation_error');

    // Seed a real user, then re-invite the SAME email → 409 email_already_exists.
    const first = await page.evaluate(async (e) => {
      const r = await fetch('/api/v1/users/invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Dup', last_name: 'Original', email: e, roles: ['team_member'] }),
      });
      return { status: r.status, body: await r.json() };
    }, email);
    expect(first.status).toBe(201);

    const dup = await page.evaluate(async (e) => {
      const r = await fetch('/api/v1/users/invite', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ first_name: 'Dup', last_name: 'Second', email: e, roles: ['team_member'] }),
      });
      return { status: r.status, body: await r.json().catch(() => null) };
    }, email);
    expect(dup.status).toBe(409);
    expect(dup.body.error).toBe('email_already_exists');
  });

  // ── FR-6: reinvite from list issues a fresh token + shows a toast ─────────
  test('FR-6: reinvite on an invited row mints a fresh reset token and shows a "resent" toast', async ({ page }) => {
    await login(page);
    const ts = Date.now();
    const seed = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'Reinv', last_name: 'Target',
      email: `reinvite.${ts}@yumyums.kitchen`, roles: ['team_member'],
    });
    expect(seed.user).toBeTruthy();
    const userId = seed.user.id;
    const originalToken = tokenFromInvitePath(seed.invite_path);

    // Capture the reset-password response so we can assert a FRESH token was minted.
    let mintedPath = null;
    await page.route(`**/api/v1/users/${userId}/reset-password`, async route => {
      const resp = await route.fetch();
      const json = await resp.json().catch(() => ({}));
      mintedPath = json.reset_path || null;
      await route.fulfill({ response: resp });
    });

    await page.goto('/users.html');
    await waitForUserList(page);

    // Tap the row-level Reinvite button (only present on invited users).
    const reinviteBtn = page.locator(`[data-action="reinvite"][data-user-id="${userId}"]`);
    await expect(reinviteBtn).toBeVisible();
    await reinviteBtn.click();

    // A toast confirms the resend (users.html:587).
    const toast = page.locator('#toast');
    await expect(toast).toContainText('resent', { timeout: 3000 });

    // The minted token must be a fresh one (different from the original invite token).
    expect(mintedPath).toBeTruthy();
    const freshToken = tokenFromInvitePath(mintedPath);
    expect(freshToken).toBeTruthy();
    expect(freshToken).not.toBe(originalToken);
  });

  // ── FR-9: email field is immutable (readonly) on the edit form ────────────
  test('FR-9: edit form renders the email field read-only', async ({ page }) => {
    await login(page);
    const ts = Date.now();
    const seed = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'Immut', last_name: 'Email',
      email: `immut.email.${ts}@yumyums.kitchen`, roles: ['team_member'],
    });
    const userId = seed.user.id;

    await page.goto('/users.html');
    await waitForUserList(page);
    await page.click(`[data-action="edit-user"][data-user-id="${userId}"]`);
    await waitForEditCard(page);

    // The email input must carry the readonly attribute — it cannot be changed.
    const emailField = page.locator('#f-email');
    await expect(emailField).toHaveAttribute('readonly', '');
    await expect(emailField).not.toBeEditable();
  });

  // ── FR-10: server rejects an invalid notification channel with 400 ────────
  test('FR-10: PATCH with an invalid notification channel returns 400', async ({ page }) => {
    await login(page);
    const ts = Date.now();
    const seed = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'Chan', last_name: 'Bad',
      email: `chan.bad.${ts}@yumyums.kitchen`, roles: ['team_member'],
    });
    const userId = seed.user.id;

    // Empty channel set → 400 (server requires ≥1).
    const emptyRes = await page.evaluate(async (id) => {
      const r = await fetch(`/api/v1/users/${id}`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notification_pref: [] }),
      });
      return r.status;
    }, userId);
    expect(emptyRes).toBe(400);

    // Unknown channel value → 400.
    const badRes = await page.evaluate(async (id) => {
      const r = await fetch(`/api/v1/users/${id}`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notification_pref: ['carrier_pigeon'] }),
      });
      return r.status;
    }, userId);
    expect(badRes).toBe(400);
  });

  // ── FR-11: server rejects an invalid IANA timezone with 400 ───────────────
  test('FR-11: PATCH with an invalid IANA timezone returns 400', async ({ page }) => {
    await login(page);
    const ts = Date.now();
    const seed = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'Tz', last_name: 'Bad',
      email: `tz.bad.${ts}@yumyums.kitchen`, roles: ['team_member'],
    });
    const userId = seed.user.id;

    const badTz = await page.evaluate(async (id) => {
      const r = await fetch(`/api/v1/users/${id}`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ timezone: 'Mars/Olympus_Mons' }),
      });
      return r.status;
    }, userId);
    expect(badTz).toBe(400);

    // Sanity: a valid IANA zone is accepted (200) — proves the 400 above is the
    // validation firing, not a blanket rejection.
    const goodTz = await page.evaluate(async (id) => {
      const r = await fetch(`/api/v1/users/${id}`, {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ timezone: 'America/Chicago' }),
      });
      return r.status;
    }, userId);
    expect(goodTz).toBe(200);
  });

  // ── FR-15: inline "Get Invite Link" surfaces the link in the edit card ────
  test('FR-15: inline Get Invite Link mints a token and renders the link inline', async ({ page }) => {
    await login(page);
    const ts = Date.now();
    // An invited user with NO cached URL renders the "Get Invite Link" button.
    const seed = await usersApiCall(page, 'POST', 'invite', {
      first_name: 'Inline', last_name: 'Link',
      email: `inline.link.${ts}@yumyums.kitchen`, roles: ['team_member'],
    });
    const userId = seed.user.id;

    await page.goto('/users.html');
    await waitForUserList(page);
    await page.click(`[data-action="edit-user"][data-user-id="${userId}"]`);
    await waitForEditCard(page);

    // Before: the inline container shows the button and no link yet.
    const getLinkBtn = page.locator('[data-action="get-invite-link"]');
    await expect(getLinkBtn).toBeVisible();
    await expect(page.locator('#invite-inline .invite-url')).toHaveCount(0);

    await getLinkBtn.click();

    // After: a link with a real /login.html?token= URL is rendered inline.
    const inlineUrl = page.locator('#invite-inline .invite-url');
    await expect(inlineUrl).toBeVisible({ timeout: 5000 });
    await expect(inlineUrl).toContainText('/login.html?token=');
  });

  // ── FR-16/17: Access-tab role toggle persists across a reload ─────────────
  test('FR-16/17: toggling a role permission persists via PUT and survives a reload', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });

    // Pick a stable, addressable toggle (first app's "manager" role) so we can
    // re-locate the SAME toggle after reload by its slug+role data attributes.
    const firstSlug = await page.locator('[data-action="toggle-perm"]').first().evaluate(el => el.dataset.slug);
    const toggle = page.locator(`[data-action="toggle-perm"][data-slug="${firstSlug}"][data-role="manager"]`);
    const wasChecked = await toggle.isChecked();

    // Flip it. The handler pushes to APPS_PERMS then PUTs the full grant set.
    await toggle.evaluate(el => el.click());
    // Wait for the PUT to land (savePermissions is fire-and-forget on release).
    await page.waitForResponse(r =>
      r.url().includes(`/api/v1/apps/${firstSlug}/permissions`) && r.request().method() === 'PUT'
    );

    // Reload — the toggle state must now come from the persisted DB, not memory.
    await page.reload();
    await waitForUserList(page);
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });
    const toggleAfter = page.locator(`[data-action="toggle-perm"][data-slug="${firstSlug}"][data-role="manager"]`);
    expect(await toggleAfter.isChecked()).toBe(!wasChecked);

    // Restore original state so sibling tests are not polluted.
    await toggleAfter.evaluate(el => el.click());
    await page.waitForResponse(r =>
      r.url().includes(`/api/v1/apps/${firstSlug}/permissions`) && r.request().method() === 'PUT'
    );
  });

  // ── FR-18/19: individual grant chip add + remove persist across reload ────
  test('FR-18/19: adding then removing an individual user grant persists across reload', async ({ page }) => {
    await login(page);
    const ts = Date.now();
    // Seed a distinctly-named user so we can find their grant chip by text.
    const grantee = await usersApiCall(page, 'POST', 'invite', {
      first_name: `Grantee${ts}`, last_name: 'Chip',
      email: `grantee.${ts}@yumyums.kitchen`, roles: ['team_member'],
    });
    expect(grantee.user).toBeTruthy();
    const granteeId = String(grantee.user.id);
    const granteeName = `Grantee${ts}`; // display name is "Grantee<ts> C." — chip/opt text contains this substring

    await page.goto('/users.html');
    await waitForUserList(page);
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });

    // Work against the first app card. Select the grantee in its picker + Add.
    const firstSlug = await page.locator('.add-grant select').first().evaluate(el => el.id.replace('pick-', ''));
    // Option value is the grantee's user id; label is the (abbreviated) display name.
    await page.selectOption(`#pick-${firstSlug}`, granteeId);
    await page.click(`[data-action="add-grant"][data-slug="${firstSlug}"]`);
    await page.waitForResponse(r =>
      r.url().includes(`/api/v1/apps/${firstSlug}/permissions`) && r.request().method() === 'PUT'
    );

    // FR-18: chip appears. Reload → chip STILL there (persisted).
    const card = page.locator('.access-card').first();
    await expect(card.locator('.chip', { hasText: granteeName })).toBeVisible();

    await page.reload();
    await waitForUserList(page);
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });
    const cardAfterAdd = page.locator(`.access-card`).filter({ has: page.locator('.chip', { hasText: granteeName }) }).first();
    await expect(cardAfterAdd.locator('.chip', { hasText: granteeName })).toBeVisible();

    // FR-19: remove the chip via its × button. Reload → chip GONE (persisted).
    await cardAfterAdd.locator('.chip', { hasText: granteeName }).locator('.chip-rm').click();
    await page.waitForResponse(r =>
      r.url().includes(`/api/v1/apps/${firstSlug}/permissions`) && r.request().method() === 'PUT'
    );
    await expect(page.locator('.chip', { hasText: granteeName })).toHaveCount(0);

    await page.reload();
    await waitForUserList(page);
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });
    await expect(page.locator('.chip', { hasText: granteeName })).toHaveCount(0);
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// F5 · inventory-tab-gating — the per-tab grant surface (design §1.3 station 4)
// ═══════════════════════════════════════════════════════════════════════════
//
// Option (i) claims the Users admin UI needs ZERO backend change: the two new
// hq_apps rows fall out of GetAppPermissions (which returns every enabled app)
// and render with the standard role/user toggles. This block is the proof of
// that claim, not a restatement of it — if SeedHQApps or the Access-list render
// ever stops surfacing a gated tab, the grant becomes unreachable through the
// only UI that can issue it, and the gate silently becomes a lockout.
test.describe('Access tab — per-tab inventory grants', () => {
  test('the two per-tab slugs are grantable through the Access list', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });

    // Both tab cards render, with their own toggle sets.
    for (const slug of ['inventory-trends', 'inventory-cost']) {
      const card = page.locator('#access-' + slug);
      await expect(card).toHaveCount(1);
      await expect(card.locator('input[data-action="toggle-perm"]')).toHaveCount(3);
    }
    // ...alongside, not instead of, the umbrella app card.
    await expect(page.locator('#access-inventory')).toHaveCount(1);
  });

  test('a per-tab role grant round-trips through PUT /apps/{slug}/permissions', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');

    const saved = await page.evaluate(async () => {
      await fetch('/api/v1/apps/inventory-trends/permissions', {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role_grants: ['manager'], user_grants: [] }),
      });
      const perms = await (await fetch('/api/v1/apps/permissions')).json();
      return (perms || []).find(a => a.slug === 'inventory-trends');
    });
    expect(saved.role_grants).toContain('manager');

    // Clean up so later specs see a pristine grant table.
    await page.evaluate(async () => {
      await fetch('/api/v1/apps/inventory-trends/permissions', {
        method: 'PUT', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ role_grants: [], user_grants: [] }),
      });
    });
  });

  // §1.6 rider (c), SIGNED as expected behavior: a tab grant without the
  // umbrella app grant leaves the launcher tile hidden while the direct URL
  // works. The Users UI must say so, or an admin issues a grant whose effect
  // they cannot see from the launcher and reads it as broken.
  test('a tab card carries the "grant Inventory too" nudge', async ({ page }) => {
    await login(page);
    await page.goto('/users.html');
    await waitForUserList(page);
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });

    await expect(page.locator('#access-inventory-trends .tab-grant-note')).toHaveCount(1);
    await expect(page.locator('#access-inventory-trends .tab-grant-note')).toContainText('Inventory');
    await expect(page.locator('#access-inventory .tab-grant-note')).toHaveCount(0);
  });
});
