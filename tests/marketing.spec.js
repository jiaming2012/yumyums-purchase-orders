// ═══════════════════════════════════════════════════════════════════════════
// Marketing tile + page shell + permission seed — card marketing-tile-and-page
// (run 20260905, Activity C, design docs/qr-offline-redemption-handoff.md §16)
// ═══════════════════════════════════════════════════════════════════════════
//
// RED-FIRST (greenfield): every test in this file was written and RUN against
// the pre-change tree, where it reds because the behavior does not exist —
// no marketing tile, no marketing.html, no 'marketing' / 'marketing-offline-
// override' rows out of SeedHQApps. Evidence:
// .night-crew/runs/2026-09-05-autonomous/card1-red.log and the ## Red-first
// section of merge-intents/marketing-tile-and-page.md.
//
// What this file pins (the card's done_when, a–c; d is build-sw.js):
//   (a) a team_member WITH the app grant sees the Marketing tile and, on
//       marketing.html, the Scan section (the only live sub-section tonight);
//   (b) a NON-granted user sees no Marketing tile;
//   (c) the offline_override ENTITLEMENT renders as a grantable row in the
//       Users access editor (standard role toggles + individual user grants),
//       and a per-user grant on it round-trips to that user's /me/apps.
//
// The entitlement is deliberately an hq_apps grant-surface row
// (`marketing-offline-override`) so it reuses every existing station —
// GetAppPermissions → renderAccess, PUT /apps/{slug}/permissions, /me/apps,
// RequirePermission. It is an ENTITLEMENT, not a gated tab: the eventual
// enforcement (redemption-submit-flow / gstate cards) must check it with NO
// umbrella slug — holding the `marketing` app grant must never imply the
// override. backend/internal/auth/marketing_seed_test.go pins that narrowness
// at the gate itself.

const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const USER_PASSWORD = 'test456';

async function loginAs(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// makeUser invites a user with the given roles and activates them. Leaves the
// browser logged in as ADMIN. Returns { id, email }. (Same shape as
// tests/grant-enforcement-parity.spec.js — duplicated per this suite's
// per-file-helper convention.)
async function makeUser(page, tag, roles) {
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  const email = `mkt-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  const invite = await page.evaluate(async ([em, rs]) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Mkt', last_name: 'Tester', email: em, roles: rs }),
    });
    return res.json();
  }, [email, roles]);
  const token = (invite.invite_path || '').split('token=')[1];
  expect(token, 'invite token').toBeTruthy();
  await page.evaluate(async ([t, pw]) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: pw }),
    });
  }, [token, USER_PASSWORD]);
  // accept-invite rotated the cookie onto the new user — go back to admin.
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  return { id: String(invite.user.id), email };
}

// Read one slug's current permission row set (as admin, via the API).
async function getSlugPerms(page, slug) {
  return page.evaluate(async (s) => {
    const perms = await (await fetch('/api/v1/apps/permissions')).json();
    return (perms || []).find(a => a.slug === s) || null;
  }, slug);
}

// Full-replace one slug's permission set; expects 200.
async function putSlugPerms(page, slug, body) {
  const status = await page.evaluate(async ([s, b]) => {
    const r = await fetch('/api/v1/apps/' + s + '/permissions', {
      method: 'PUT', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(b),
    });
    return r.status;
  }, [slug, body]);
  expect(status, `PUT /apps/${slug}/permissions`).toBe(200);
}

// The launcher grid stays visibility:hidden until filterTilesByPermissions has
// applied the /me/apps answer — wait for that before judging any tile, or a
// pre-filter "hidden" reads as a false pass for the ungranted case.
async function waitForFilteredGrid(page) {
  await page.waitForFunction(() => {
    const grid = document.querySelector('.grid');
    return grid && grid.style.visibility === '';
  });
}

test.describe('Marketing tile + permission seed (card marketing-tile-and-page)', () => {

  // ── Seed state first: these run before anything in this file mutates grants ──

  test('SEED: SeedHQApps registers marketing + marketing-offline-override with their first-registration grants', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);

    const marketing = await getSlugPerms(page, 'marketing');
    expect(marketing, 'hq_apps row `marketing` is seeded and enabled').toBeTruthy();
    expect(marketing.name).toBe('Marketing');
    // §16 permissions table: scan/redeem min role is team_member, so the APP
    // grant (tab access) seeds to all three roles.
    for (const role of ['admin', 'manager', 'team_member']) {
      expect(marketing.role_grants, `marketing seeded to role ${role}`).toContain(role);
    }

    const override = await getSlugPerms(page, 'marketing-offline-override');
    expect(override, 'hq_apps row `marketing-offline-override` is seeded and enabled').toBeTruthy();
    // Fork #12 (operator-resolved): the entitlement seeds true for admin users
    // ONLY — managers/team_members get it by explicit grant, never by role
    // implication.
    expect(override.role_grants, 'offline_override seeded to admin').toContain('admin');
    expect(override.role_grants, 'offline_override NOT seeded to manager').not.toContain('manager');
    expect(override.role_grants, 'offline_override NOT seeded to team_member').not.toContain('team_member');
  });

  // ── done_when (a): granted team_member sees the tile and the Scan section ──

  test('a team_member with the app grant sees the Marketing tile on the launcher', async ({ page }) => {
    const user = await makeUser(page, 'granted', ['team_member']);
    await loginAs(page, user.email, USER_PASSWORD);
    await page.goto('/');
    await waitForFilteredGrid(page);
    const tile = page.locator('a.tile[href="marketing.html"]');
    await expect(tile, 'Marketing tile renders for a granted team_member').toHaveCount(1);
    await expect(tile).toBeVisible();
    await expect(tile.locator('.tile-title')).toHaveText('Marketing');
  });

  test('marketing.html shows Scan live and the other three sub-sections as labeled placeholders', async ({ page }) => {
    const user = await makeUser(page, 'scan', ['team_member']);
    await loginAs(page, user.email, USER_PASSWORD);
    await page.goto('/marketing.html');

    // Four sub-sections (§16): Scan / Campaigns / Subscribers / Redemption stats.
    await expect(page.locator('.tabs button')).toHaveCount(4);
    await expect(page.locator('#t1')).toHaveText('Scan');
    await expect(page.locator('#t2')).toHaveText('Campaigns');
    await expect(page.locator('#t3')).toHaveText('Subscribers');
    await expect(page.locator('#t4')).toHaveText('Stats');

    // Scan is the default, LIVE section: visible, with the scanner host the
    // camera card (camera-scanner-decode) mounts into.
    await expect(page.locator('#s1')).toBeVisible();
    await expect(page.locator('#s1')).toContainText('Scan to Redeem');
    await expect(page.locator('#scanner-host')).toHaveCount(1);
    await expect(page.locator('#s2')).toBeHidden();

    // The other three are labeled placeholders, not blank space (UI-R rule:
    // blank render = defect) — each names what it will hold and carries the
    // launcher's "Soon" badge convention.
    await page.click('#t2');
    await expect(page.locator('#s2')).toBeVisible();
    await expect(page.locator('#s2 .badge')).toHaveText('Soon');
    await expect(page.locator('#s2')).toContainText('Campaigns');
    await expect(page.locator('#s1')).toBeHidden();

    await page.click('#t3');
    await expect(page.locator('#s3')).toBeVisible();
    await expect(page.locator('#s3 .badge')).toHaveText('Soon');
    await expect(page.locator('#s3')).toContainText('Subscribers');

    await page.click('#t4');
    await expect(page.locator('#s4')).toBeVisible();
    await expect(page.locator('#s4 .badge')).toHaveText('Soon');
    await expect(page.locator('#s4')).toContainText('Redemption stats');
  });

  // ── done_when (b): a non-granted user sees no tile ─────────────────────────

  test('a user without the marketing grant sees NO Marketing tile', async ({ page }) => {
    const user = await makeUser(page, 'ungranted', ['team_member']);

    // Strip the seeded marketing role grants (preserving any individual user
    // grants), so this team_member holds nothing on the app.
    const before = await getSlugPerms(page, 'marketing');
    expect(before, 'marketing app must exist to be strippable').toBeTruthy();
    await putSlugPerms(page, 'marketing', {
      role_grants: [], user_grants: (before.user_grants || []).map(String),
    });

    try {
      await loginAs(page, user.email, USER_PASSWORD);
      await page.goto('/');
      await waitForFilteredGrid(page);
      const tile = page.locator('a.tile[href="marketing.html"]');
      // The tile exists in the static grid (filterTilesByPermissions only
      // toggles existing tiles) but must be display:none for this user.
      await expect(tile).toHaveCount(1);
      await expect(tile, 'Marketing tile hidden for an ungranted user').toBeHidden();
    } finally {
      // Restore the seeded grant state for later specs in the shared serial DB.
      await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await putSlugPerms(page, 'marketing', {
        role_grants: before.role_grants || [],
        user_grants: (before.user_grants || []).map(String),
      });
    }
  });

  // ── done_when (c): the entitlement is a grantable row in the access editor ──

  test('the offline_override entitlement renders as a grantable row in the Users access editor', async ({ page }) => {
    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
    await page.goto('/users.html');
    await page.waitForFunction(() => {
      const el = document.getElementById('user-list');
      return el && (el.querySelector('.row') || el.querySelector('.empty') || el.querySelector('.error-msg'));
    });
    await page.click('#t2');
    await page.waitForFunction(() => {
      const s2 = document.getElementById('s2');
      return s2 && s2.querySelector('.access-card');
    });

    // The entitlement card renders alongside (not instead of) the app card.
    await expect(page.locator('#access-marketing')).toHaveCount(1);
    const card = page.locator('#access-marketing-offline-override');
    await expect(card, 'entitlement row renders in the access editor').toHaveCount(1);

    // Standard grant machinery: 3 role toggles + the individual-grant station
    // ("grantable to any role" — admin, manager, or team_member).
    await expect(card.locator('input[data-action="toggle-perm"]')).toHaveCount(3);
    await expect(card.locator('.add-grant select, .chips, .chips-empty').first()).toBeAttached();
  });

  test('a per-user offline_override grant round-trips to that user\'s /me/apps', async ({ page }) => {
    const user = await makeUser(page, 'override', ['team_member']);
    const before = await getSlugPerms(page, 'marketing-offline-override');
    expect(before, 'entitlement surface must exist').toBeTruthy();

    try {
      // Grant the entitlement to this specific user (role grants untouched).
      await putSlugPerms(page, 'marketing-offline-override', {
        role_grants: before.role_grants || [],
        user_grants: [...(before.user_grants || []).map(String), user.id],
      });

      await loginAs(page, user.email, USER_PASSWORD);
      const slugs = await page.evaluate(async () => {
        const apps = await (await fetch('/api/v1/me/apps')).json();
        return (apps || []).map(a => a.slug);
      });
      expect(slugs, 'entitlement reaches the user\'s grant projection').toContain('marketing-offline-override');
    } finally {
      await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await putSlugPerms(page, 'marketing-offline-override', {
        role_grants: before.role_grants || [],
        user_grants: (before.user_grants || []).map(String),
      });
    }
  });
});
