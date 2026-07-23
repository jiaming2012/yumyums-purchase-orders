const { createBdd } = require('playwright-bdd');
const { expect } = require('@playwright/test');

const { Given, When, Then } = createBdd();

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

// Shared state between steps
let deviceAPage;
let deviceBPage;
let deviceBContext;
let inviteLink;

// ─── API helper ─────────────────────────────────────────────────────────────

async function api(page, method, path, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch(p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path, body]);
}

// ─── Cleanup helper ─────────────────────────────────────────────────────────

async function cleanupTestUser(page) {
  const users = await api(page, 'GET', '/api/v1/users');
  if (!Array.isArray(users)) return;
  const jim = users.find(u => u.email === 'jim@gmail.com');
  if (jim) await api(page, 'DELETE', `/api/v1/users/${jim.id}`);
}

// ─── Wait helpers ───────────────────────────────────────────────────────────

async function waitForUserList(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('user-list');
    if (!el) return false;
    return el.querySelector('.row') || el.querySelector('.empty') || el.querySelector('.error-msg');
  });
}

async function waitForEditCard(page) {
  await page.waitForFunction(() => {
    const el = document.getElementById('edit-card');
    if (!el) return false;
    return el.querySelector('.form-wrap') || el.querySelector('.invite-link-panel');
  });
}

async function waitForAccessTab(page) {
  await page.waitForFunction(() => {
    const s2 = document.getElementById('s2');
    return s2 && s2.querySelector('.access-card');
  });
}

// ─── Device A: Given steps ──────────────────────────────────────────────────

Given('User A is logged in on Device A as {string}', async ({ page, browser }, email) => {
  deviceAPage = page;
  await deviceAPage.goto('/login.html');
  await deviceAPage.fill('input[type="email"]', email);
  await deviceAPage.fill('input[type="password"]', ADMIN_PASSWORD);
  await deviceAPage.click('button.btn');
  await deviceAPage.waitForURL(url => !url.pathname.includes('login'));

  // Clean up any leftover test user
  await cleanupTestUser(deviceAPage);

  // Card G1: the chromium project's suite baselines role-grant several apps to
  // team_member in the shared serial DB, and this scenario asserts User B sees
  // EXACTLY the two apps enabled below. Reset team_member out of every app's
  // role_grants first so the scenario starts from the clean slate it describes
  // (user_grants and other roles are preserved).
  await deviceAPage.evaluate(async () => {
    const perms = await (await fetch('/api/v1/apps/permissions')).json();
    for (const app of (perms || [])) {
      const roles = (app.role_grants || []).filter(r => r !== 'team_member');
      if (roles.length !== (app.role_grants || []).length) {
        await fetch('/api/v1/apps/' + app.slug + '/permissions', {
          method: 'PUT', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ role_grants: roles, user_grants: (app.user_grants || []).map(String) }),
        });
      }
    }
  });

  // Create a separate browser context for Device B
  deviceBContext = await browser.newContext();
  deviceBPage = await deviceBContext.newPage();
});

Given('User A navigates to the Users app', async () => {
  await deviceAPage.goto('/users.html');
  await waitForUserList(deviceAPage);
});

Given('User A opens the Access tab', async () => {
  await deviceAPage.click('#t2');
  await waitForAccessTab(deviceAPage);
});

Given('User A enables {string} for team members', async ({}, appName) => {
  // Find the access card for this app and ensure team_member toggle is checked
  const slug = appName.toLowerCase();
  const card = deviceAPage.locator(`#access-${slug}`);
  await expect(card).toBeVisible();

  const toggle = card.locator('[data-action="toggle-perm"][data-role="team_member"]');
  const isChecked = await toggle.isChecked();
  if (!isChecked) {
    await toggle.evaluate(el => el.click());
    await deviceAPage.waitForTimeout(300);
  }
});

// ─── Device A: When steps ───────────────────────────────────────────────────

When('User A clicks {string}', async ({}, buttonText) => {
  if (buttonText === 'Add Crew Member') {
    // Switch back to Users tab first (may be on Access tab)
    await deviceAPage.click('#t1');
    await waitForUserList(deviceAPage);
    await deviceAPage.click('[data-action="show-invite"]');
    await waitForEditCard(deviceAPage);
  } else {
    await deviceAPage.click(`button:has-text("${buttonText}")`);
  }
});

When('User A fills in the invite form:', async ({}, dataTable) => {
  const rows = dataTable.rows();
  for (const [field, value] of rows) {
    switch (field.toLowerCase()) {
      case 'first name':
        await deviceAPage.fill('#f-first', value);
        break;
      case 'last name':
        await deviceAPage.fill('#f-last', value);
        break;
      case 'email':
        await deviceAPage.fill('#f-email', value);
        break;
      case 'role': {
        // Team Member is already selected by default via .on class.
        // If a different role is requested, click to toggle.
        const roleMap = { 'Team Member': 'team_member', 'Manager': 'manager', 'Admin': 'admin' };
        const roleKey = roleMap[value];
        if (roleKey) {
          // Ensure only the requested role is selected
          const chips = deviceAPage.locator('#f-roles .role-chip');
          const count = await chips.count();
          for (let i = 0; i < count; i++) {
            const chip = chips.nth(i);
            const chipRole = await chip.getAttribute('data-role');
            const isOn = await chip.evaluate(el => el.classList.contains('on'));
            if (chipRole === roleKey && !isOn) {
              await chip.click();
            } else if (chipRole !== roleKey && isOn) {
              await chip.click();
            }
          }
        }
        break;
      }
      case 'employee type': {
        // Click the matching emp type chip (W2 or 1099)
        const empChip = deviceAPage.locator(`#f-emp-type .role-chip[data-emptype="${value}"]`);
        const isOn = await empChip.evaluate(el => el.classList.contains('on'));
        if (!isOn) await empChip.click();
        break;
      }
      case 'starting salary':
        await deviceAPage.fill('#f-salary', value);
        break;
    }
  }
});

When('User A submits the invite', async () => {
  await deviceAPage.click('[data-action="submit-invite"]');
  await deviceAPage.waitForFunction(() => {
    const el = document.getElementById('edit-card');
    return el && el.querySelector('.invite-link-panel');
  });
});

Then('User A should see the invite link panel', async () => {
  await expect(deviceAPage.locator('.invite-link-panel h2')).toContainText('Invite Link');
  await expect(deviceAPage.locator('.invite-url')).toContainText('/login.html?token=');
});

// ─── Copy invite link and open on Device B ──────────────────────────────────

When('User A copies the invite link', async () => {
  inviteLink = await deviceAPage.locator('.invite-url').textContent();
  inviteLink = inviteLink.trim();
  // Extract the path from the full URL
  const url = new URL(inviteLink);
  inviteLink = url.pathname + url.search;
});

When('User B opens the invite link on Device B', async () => {
  await deviceBPage.goto(inviteLink);
});

Then('User B should see the welcome form', async () => {
  await expect(deviceBPage.locator('#invite-form')).toBeVisible({ timeout: 10000 });
  await expect(deviceBPage.locator('#welcome-heading')).toContainText('Welcome, Jim');
  await expect(deviceBPage.locator('#invite-pw')).toBeVisible();
});

// ─── Device B: Onboarding form ──────────────────────────────────────────────

When('User B fills in the onboarding form:', async ({}, dataTable) => {
  const rows = dataTable.rows();
  for (const [field, value] of rows) {
    switch (field.toLowerCase()) {
      case 'password':
        await deviceBPage.fill('#invite-pw', value);
        await deviceBPage.fill('#invite-pw2', value);
        break;
      case 'toast pos':
        await deviceBPage.fill('#invite-toast', value);
        break;
      case 'cashapp id':
        await deviceBPage.fill('#invite-cashapp', value);
        break;
      case 'phone number':
        await deviceBPage.fill('#invite-phone', value);
        break;
    }
  }
});

When('User B submits the onboarding form', async () => {
  await deviceBPage.locator('#invite-form button.btn').click();
  await deviceBPage.waitForURL(url => url.pathname.includes('index.html') || url.pathname === '/', { timeout: 10000 });
});

Then('User B should be redirected to the home screen', async () => {
  const url = deviceBPage.url();
  expect(url).toMatch(/\/(index\.html)?(\?.*)?$/);
});

// ─── Device B: Verify visible apps ─────────────────────────────────────────

Then('User B should see exactly {int} app tiles', async ({}, count) => {
  // Wait for the grid to be filtered by permissions
  await deviceBPage.waitForFunction(() => {
    const grid = document.querySelector('.grid');
    return grid && grid.style.visibility !== 'hidden';
  });
  // Count visible active tiles (not "Soon" placeholders)
  const visibleTiles = deviceBPage.locator('.grid .tile:not([style*="display: none"]):not([style*="display:none"])');
  await expect(visibleTiles).toHaveCount(count);
});

Then('User B should see the {string} app', async ({}, appName) => {
  const tile = deviceBPage.locator(`.grid .tile:not([style*="display: none"]):not([style*="display:none"]) .tile-title:has-text("${appName}")`);
  await expect(tile).toBeVisible();
});
