// ═══════════════════════════════════════════════════════════════════════════
// Grant-enforcement parity — every app the Users tab can grant is enforced
// ═══════════════════════════════════════════════════════════════════════════
//
// WHY THIS FILE EXISTS (card G1, run 2026-07-24)
//
// The Users tab offers a grant toggle for every hq_apps row (11 slugs), but
// before this card the backend enforced exactly 2 of them (inventory-trends,
// inventory-cost — both F5). The operator ruling (T-20 decision 36, verbatim):
//
//     "If an employee does not have access to the app (or access to the
//      app's tab), then they should NOT be able to access the view / tab /
//      data."
//
// The grant model is a DATA boundary, not a UI convenience. This file is the
// guard that keeps it one:
//
//   1. PARITY (source-derived, anti-vacuity — the `EXCEPTIONS` shape from
//      tests/ops-authz-coverage.spec.js): the app list is DERIVED from
//      db.SeedHQApps, the enforced list is DERIVED from RequirePermission
//      mounts in main.go, and every seeded slug must be enforced somewhere or
//      recorded N/A-with-reason below. Adding an app to SeedHQApps without
//      gating its endpoints fails this suite; so does letting an N/A entry rot.
//
//   2. PAIRS (runtime): per migrated app, a real ungranted logged-in user gets
//      403 {"error":"forbidden","missing_grant":<slug>} on that app's data
//      endpoints, and the same user WITH the grant gets 200. The without-grant
//      half is the red-first proof for card G1: before the migration those
//      endpoints returned 200 to an ungranted account.
//
// DELIBERATELY OUTSIDE ANY APP GATE (documented exceptions, not oversights):
//   • /api/v1/auth/* (login, logout, invite-info, accept-invite) — the access-
//     resolution plumbing itself; gating it is self-defeating.
//   • /api/v1/me, /api/v1/me/apps — the launcher's grant resolver; it must
//     serve ungranted users to tell them they are ungranted.
//   • /api/v1/health, /api/v1/logs — infra.
//   • /api/v1/users/{id}/notification-preference (GET/PUT) — admin-OR-SELF by
//     contract (users.spec.js NFR-4); the self branch serves the caller's OWN
//     row, which is profile plumbing, not Users-app data about others.
//   • /api/v1/inventory/period-summary, /api/v1/inventory/menu-cogs — machine-
//     to-machine service-token surfaces (HQ_INVENTORY_SERVICE_TOKEN), not
//     session-grant surfaces.
//   • /api/v1/sync/token — the Supabase bridge mint (card `sync-jwt-bridge-
//     endpoint`). Same category as /me: access-resolution plumbing that must
//     serve an UNGRANTED user, so it can hand them a token whose live grant
//     projection lets them reach nothing. Gating it behind a grant would be
//     circular, and choosing WHICH grant gates the sync bridge would be
//     inventing a permission concept. It is inside the cookie group and takes
//     no user-id parameter — identity comes only from the session, so there is
//     no mint-for-someone-else path. Authorization for the DATA it unlocks is
//     enforced downstream by RLS against a live grant projection, not by this
//     endpoint; see backend/internal/sync/jwtbridge.go and
//     .night-crew/qa/spike-supabase/sql/hq-bridge-policies.sql.
//   • /api/v1/photos/* — PARKED (card G1 park trigger i): a cross-app utility
//     called by workflows.html, purchasing.html, inventory.html AND
//     onboarding.html. Which grant governs it is an operator question; until
//     answered it stays authenticated-only, and this note is the record.
//
// The four placeholder slugs (payroll, scheduling, hiring, bi) are N/A below,
// each with a reason and a stale-N/A tripwire — never silently skipped.

const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';
const USER_PASSWORD = 'test456';

const MAIN_GO = path.join(__dirname, '..', 'backend', 'cmd', 'server', 'main.go');
const DB_GO = path.join(__dirname, '..', 'backend', 'internal', 'db', 'db.go');

// ─── Derivation 1: what slugs does SeedHQApps register? ─────────────────────
function seededSlugs() {
  const src = fs.readFileSync(DB_GO, 'utf8');
  const start = src.indexOf('INSERT INTO hq_apps');
  if (start < 0) {
    throw new Error(`grant-parity: could not find "INSERT INTO hq_apps" in ${DB_GO}. ` +
      'If SeedHQApps moved or changed shape, update this parser — the parity guard is ' +
      'worthless the moment it stops seeing the app list.');
  }
  const end = src.indexOf('ON CONFLICT', start);
  if (end < 0) {
    throw new Error('grant-parity: found the hq_apps INSERT but not its ON CONFLICT terminator.');
  }
  const block = src.slice(start, end);
  const slugs = [...block.matchAll(/\('([a-z0-9-]+)'\s*,/g)].map(m => m[1]);
  if (slugs.length < 11) {
    throw new Error(`grant-parity: derived only ${slugs.length} slug(s) from SeedHQApps — below ` +
      'the floor of 11. Either apps were genuinely removed (lower the floor deliberately, in the ' +
      'same commit) or this parser silently narrowed. A shrinking derived set is a vacuous pass.');
  }
  if (new Set(slugs).size !== slugs.length) {
    throw new Error(`grant-parity: SeedHQApps lists a duplicate slug: ${slugs.join(', ')}`);
  }
  return slugs.sort();
}

// ─── Derivation 2: which slugs does main.go actually mount a gate for? ──────
// A slug counts as enforced if it appears in ANY RequirePermission mount,
// narrow or umbrella — the umbrella position is how the whole-app slugs
// `inventory` etc. cover their tab gates, and the narrow position is how an
// app slug gates its own base surface.
function enforcedSlugs() {
  const src = fs.readFileSync(MAIN_GO, 'utf8');
  const out = new Set();
  for (const line of src.split('\n')) {
    const t = line.trim();
    if (t.startsWith('//')) continue; // a commented-out mount enforces nothing
    for (const m of t.matchAll(/auth\.RequirePermission\(pool\s*,\s*((?:"[a-z0-9-]+"\s*,?\s*)+)\)/g)) {
      for (const s of m[1].matchAll(/"([a-z0-9-]+)"/g)) out.add(s[1]);
    }
  }
  if (out.size === 0) {
    throw new Error('grant-parity: derived ZERO RequirePermission mounts from main.go. ' +
      'The parser is broken or the middleware was renamed — fix the derivation, do not delete it.');
  }
  return [...out].sort();
}

// ─── N/A-with-reason: placeholder apps with no endpoints ────────────────────
// Each entry must stay true: the stale-N/A test asserts the slug has no
// /api/v1 route in main.go AND no RequirePermission mount. The moment payroll
// (etc.) grows an endpoint, its entry here must be deleted and the endpoint
// gated — that is the point of recording these instead of skipping them.
const NA_WITH_REASON = {
  payroll: 'placeholder — no backend endpoints exist; index.html tile is a static "Soon" badge',
  scheduling: 'placeholder — no backend endpoints exist; index.html tile is a static "Soon" badge',
  hiring: 'placeholder — no backend endpoints exist; index.html tile is a static "Soon" badge',
  bi: 'placeholder — no backend endpoints exist; index.html tile is a static "Soon" badge',
  marketing: 'page shell only (card marketing-tile-and-page, run 20260905) — marketing.html has no '
    + 'backend endpoint yet; the tile is grant-gated client-side via /me/apps. The scanner/submit '
    + 'endpoints (cards camera-scanner-decode, redemption-submit-flow, gstate-arbitration-machine) '
    + 'must mount RequirePermission("marketing") and delete this entry.',
  'marketing-offline-override': 'entitlement surface seeded ahead of its enforcer — the offline '
    + 'force-submit endpoint (redemption-submit-flow / gstate cards) must mount '
    + 'RequirePermission("marketing-offline-override") with NO umbrella slug (an app grant must '
    + 'never imply the override — QR design §13/§16 fork #12) and delete this entry. The narrow-'
    + 'mount contract is already pinned at the gate by '
    + 'backend/internal/auth/marketing_seed_test.go.',
};

// ─── Helpers (duplicated per-file, matching this suite's convention) ────────

async function loginAs(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email);
  await page.fill('input[type="password"]', password);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

// makeUser invites a user with the given roles and activates them. Leaves the
// browser logged in as ADMIN. Returns { id, email }.
async function makeUser(page, tag, roles) {
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  const email = `grant-par-${tag}-${Date.now()}-${Math.floor(Math.random() * 1e4)}@yumyums.kitchen`;
  const invite = await page.evaluate(async ([em, rs]) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Grant', last_name: 'Parity', email: em, roles: rs }),
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

// setSlugPerms does a read-modify-write on one slug's permission row set.
// mutate receives { role_grants, user_grants } and returns the replacement.
async function setSlugPerms(page, slug, mutate) {
  const current = await page.evaluate(async (s) => {
    const perms = await (await fetch('/api/v1/apps/permissions')).json();
    const app = (perms || []).find(a => a.slug === s);
    return {
      role_grants: (app && app.role_grants) || [],
      user_grants: ((app && app.user_grants) || []).map(String),
    };
  }, slug);
  const next = mutate(current);
  const status = await page.evaluate(async ([s, body]) => {
    const r = await fetch('/api/v1/apps/' + s + '/permissions', {
      method: 'PUT', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    return r.status;
  }, [slug, next]);
  expect(status, `PUT /apps/${slug}/permissions`).toBe(200);
  return current;
}

async function probe(page, path_) {
  return page.evaluate(async (p) => {
    const r = await fetch(p);
    let body = null;
    try { body = await r.json(); } catch (e) { body = null; }
    return { status: r.status, body };
  }, path_);
}

// ─── The per-app contract ───────────────────────────────────────────────────
// probes are representative DATA reads for the app: list endpoints that return
// 200 to any caller holding the grant (no additional role tier in the way).
// The users app is probed with an `admin`-ROLE user precisely because roles
// and grants are separate axes: without the grant even an admin-role caller
// must be refused; with it, isAdmin lets them through.
const APP_PAIRS = [
  {
    slug: 'operations', roles: ['team_member'],
    probes: ['/api/v1/workflow/templates', '/api/v1/workflow/myChecklists'],
  },
  {
    slug: 'inventory', roles: ['team_member'],
    // The card's red-first list: these returned 200 to an ungranted account.
    probes: [
      '/api/v1/inventory/purchases',
      '/api/v1/inventory/items',
      '/api/v1/inventory/stock',
      '/api/v1/inventory/recipes/',
      '/api/v1/inventory/menu-items',
    ],
    // GET /inventory/items deliberately also opens to the `purchasing` grant
    // (cross-app catalog read for the order form), so the WITHOUT test must
    // strip purchasing role grants too or a purchasing baseline from another
    // spec file would leak through the umbrella.
    stripSlugs: ['inventory', 'purchasing'],
  },
  {
    slug: 'purchasing', roles: ['team_member'],
    probes: ['/api/v1/purchasing/cutoff', '/api/v1/purchasing/shopping/active'],
  },
  {
    slug: 'onboarding', roles: ['team_member'],
    // /templates is additionally manager-or-admin ROLE-gated inside the
    // handler, so the granted team_member still (correctly) gets a role 403
    // there — it stays in the WITHOUT set (the grant gate must fire first,
    // naming missing_grant) but not in the granted 200 set.
    probes: ['/api/v1/onboarding/myTrainings', '/api/v1/onboarding/templates'],
    grantedProbes: ['/api/v1/onboarding/myTrainings'],
  },
  {
    slug: 'users', roles: ['admin'],
    probes: ['/api/v1/users', '/api/v1/apps/permissions'],
  },
];

// ═══════════════════════════════════════════════════════════════════════════

test.describe('Grant-enforcement parity', () => {

  // ── (c) PARITY: source-derived, no server needed ─────────────────────────
  test('PARITY-EQ: every SeedHQApps slug is enforced by a RequirePermission mount or recorded N/A-with-reason', async () => {
    const seeded = seededSlugs();
    const enforced = enforcedSlugs();
    const na = Object.keys(NA_WITH_REASON);

    // No slug may be both enforced and N/A — a gated placeholder means the
    // N/A entry is stale fiction.
    const contradictions = na.filter(s => enforced.includes(s));
    expect(contradictions,
      `slug(s) recorded N/A but actually mounted behind RequirePermission: ${contradictions.join(', ')}. ` +
      'Delete the stale N/A entries — an N/A that describes a world that no longer exists is a lie.'
    ).toEqual([]);

    // Every N/A reason must be non-empty — "recorded with reason", mechanically.
    for (const [slug, reason] of Object.entries(NA_WITH_REASON)) {
      expect(typeof reason === 'string' && reason.length > 10,
        `N/A entry for ${slug} carries no reason`).toBe(true);
    }

    // The core parity claim: seeded ⊆ enforced ∪ N/A.
    const unenforced = seeded.filter(s => !enforced.includes(s) && !na.includes(s));
    expect(unenforced,
      `SeedHQApps registers slug(s) with NO RequirePermission mount and NO N/A record: ${unenforced.join(', ')}. ` +
      'The Users tab renders a grant toggle for every seeded slug — a seeded-but-unenforced slug is ' +
      'a toggle that gates nothing, which is the exact defect card G1 closed. Gate the endpoints or ' +
      'record the slug N/A-with-reason here, on purpose.'
    ).toEqual([]);

    // And the mirror: a mount naming an unseeded slug fails closed for every
    // non-superadmin forever (no Users-tab toggle can ever grant it).
    const phantom = enforced.filter(s => !seeded.includes(s));
    expect(phantom,
      `RequirePermission mount(s) name slug(s) SeedHQApps never registers: ${phantom.join(', ')}. ` +
      'No grant can ever satisfy these gates — either seed the slug or fix the mount.'
    ).toEqual([]);
  });

  test('PARITY-NA-FRESH: N/A placeholder slugs still have no routes in main.go', async () => {
    const src = fs.readFileSync(MAIN_GO, 'utf8');
    for (const slug of Object.keys(NA_WITH_REASON)) {
      const routeRe = new RegExp(`r\\.(Route|Get|Post|Put|Patch|Delete|Handle)\\(\\s*"[^"]*/${slug}`);
      expect(routeRe.test(src),
        `main.go now registers a route for placeholder slug "${slug}". Its N/A entry in ` +
        'tests/grant-enforcement-parity.spec.js is stale: gate the new endpoints behind ' +
        `RequirePermission("${slug}") and delete the N/A record.`
      ).toBe(false);
    }
  });

  // ── (b) PAIRS: with-grant / without-grant, per migrated app ──────────────
  for (const app of APP_PAIRS) {
    test(`WITHOUT ${app.slug}: ungranted ${app.roles.join('+')} gets 403 {forbidden, missing_grant} on every data probe`, async ({ page }) => {
      const user = await makeUser(page, `no-${app.slug}`, app.roles);

      // Defence against baseline role_grants left by other spec files in the
      // shared serial DB: strip role grants for this slug AND any umbrella
      // slug that can open one of its probes (preserving other users'
      // individual grants), probe, then restore.
      const stripSlugs = app.stripSlugs || [app.slug];
      const saved = {};
      for (const s of stripSlugs) {
        saved[s] = await setSlugPerms(page, s, cur => ({
          role_grants: [], user_grants: cur.user_grants,
        }));
      }

      try {
        await loginAs(page, user.email, USER_PASSWORD);
        for (const p of app.probes) {
          const res = await probe(page, p);
          expect(res.status, `${p} must be REFUSED for an ungranted account`).toBe(403);
          expect(res.body && res.body.error, `${p} envelope.error`).toBe('forbidden');
          expect(res.body && res.body.missing_grant,
            `${p} envelope.missing_grant names the narrow slug an admin would issue`
          ).toBe(app.slug);
        }
      } finally {
        await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
        for (const s of stripSlugs) {
          await setSlugPerms(page, s, cur => ({
            role_grants: saved[s].role_grants, user_grants: cur.user_grants,
          }));
        }
        await page.evaluate(async (uid) => {
          await fetch('/api/v1/users/' + uid, { method: 'DELETE' });
        }, user.id);
      }
    });

    test(`WITH ${app.slug}: the same probes serve 200 once the user holds the ${app.slug} grant`, async ({ page }) => {
      const user = await makeUser(page, `yes-${app.slug}`, app.roles);
      await setSlugPerms(page, app.slug, cur => ({
        role_grants: cur.role_grants,
        user_grants: cur.user_grants.includes(user.id) ? cur.user_grants : [...cur.user_grants, user.id],
      }));

      try {
        await loginAs(page, user.email, USER_PASSWORD);
        for (const p of (app.grantedProbes || app.probes)) {
          const res = await probe(page, p);
          expect(res.status, `${p} must SERVE a granted account`).toBe(200);
        }
      } finally {
        await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
        await setSlugPerms(page, app.slug, cur => ({
          role_grants: cur.role_grants,
          user_grants: cur.user_grants.filter(id => id !== user.id),
        }));
        await page.evaluate(async (uid) => {
          await fetch('/api/v1/users/' + uid, { method: 'DELETE' });
        }, user.id);
      }
    });
  }

  // ── Roles are a separate axis: a grant does not manufacture a role tier ──
  test('AXIS: users grant on a team_member does NOT open the admin-only users list', async ({ page }) => {
    const user = await makeUser(page, 'axis-tm', ['team_member']);
    await setSlugPerms(page, 'users', cur => ({
      role_grants: cur.role_grants,
      user_grants: cur.user_grants.includes(user.id) ? cur.user_grants : [...cur.user_grants, user.id],
    }));
    try {
      await loginAs(page, user.email, USER_PASSWORD);
      const res = await probe(page, '/api/v1/users');
      // Passes the grant gate, then isAdmin refuses: 403 WITHOUT missing_grant.
      expect(res.status).toBe(403);
      expect(res.body && res.body.error).toBe('forbidden');
      expect(res.body && res.body.missing_grant,
        'a role refusal must not masquerade as a grant refusal').toBeFalsy();
    } finally {
      await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await setSlugPerms(page, 'users', cur => ({
        role_grants: cur.role_grants,
        user_grants: cur.user_grants.filter(id => id !== user.id),
      }));
      await page.evaluate(async (uid) => {
        await fetch('/api/v1/users/' + uid, { method: 'DELETE' });
      }, user.id);
    }
  });

  // ── The self-plumbing exception, asserted so it cannot silently widen ────
  test('EXCEPTION notification-preference: self access needs NO users grant (admin-or-self contract)', async ({ page }) => {
    const user = await makeUser(page, 'notif-self', ['team_member']);
    try {
      await loginAs(page, user.email, USER_PASSWORD);
      const res = await page.evaluate(async (uid) => {
        const r = await fetch('/api/v1/users/' + uid + '/notification-preference', {
          method: 'PUT', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ notification_channels: ['email'] }),
        });
        return { status: r.status, body: await r.json().catch(() => null) };
      }, user.id);
      expect(res.status,
        'an ungranted user must still reach their OWN notification preference — ' +
        'this endpoint is profile plumbing (users.spec.js NFR-4), deliberately outside the users gate'
      ).toBe(200);
    } finally {
      await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
      await page.evaluate(async (uid) => {
        await fetch('/api/v1/users/' + uid, { method: 'DELETE' });
      }, user.id);
    }
  });
});
