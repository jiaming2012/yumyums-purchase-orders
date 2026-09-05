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
const path = require('path');

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

// ═══════════════════════════════════════════════════════════════════════════
// Camera scanner decode — card camera-scanner-decode (run 20260905, Activity C,
// design §12/§4/§10, F3/F5/F6, D-KR3; spike
// .night-crew/knowledge/spikes/activity-c-scanner-screen/camera-scanner-decode.md)
// ═══════════════════════════════════════════════════════════════════════════
//
// RED-FIRST (greenfield): every test in this describe was written and RUN
// against the pre-change tree (Card 1's shell — no scanner, no
// window.MarketingScan), where it reds as a set. Evidence:
// .night-crew/runs/2026-09-05-autonomous/card5-red.log and the ## Red-first
// section of merge-intents/camera-scanner-decode.md.
//
// What this describe pins (the card's done_when + landed-card obligations):
//   * decode-from-image headless (html5-qrcode file-scan path, spike-proven)
//     and hash-equals-committed-seed-literal (§12/§4 — the replica-key contract);
//   * resolution order: local replica FIRST, then the QR-embedded offer
//     (D-KR3) for a not-yet-replicated customer, else unknownCode (F2);
//   * F3: offline → spentLocally reject; online → NO reject on the local flag
//     (server decides at submit — Card 6's slot);
//   * offline expiry via clock.isExpired (never raw Date.now()) and the clock
//     persist/initialState round-trip across a reload;
//   * F5: offers are DISPLAYED, never auto-picked;
//   * hash caching and per-code serialized enqueue (Card 6's entry point).
//
// All resolution tests seed the LOCAL RxDB collections directly (fixtures
// mirroring supabase/seed.sql literals) — the replica MECHANISM is Card 2's
// substrate-harness-proven surface; this card proves the resolution wiring
// against those fixtures, per the slate's own gate note.

const FIXTURE_1_TOKEN_HASH = 'c5a1641409efd198e5a55417f209eda33500fd199f1fa7fa0d8a2567ee1f9680'; // sha256("card1-test-code-fixture-1") — committed seed literal
const FIXTURE_4_TOKEN_HASH = 'a939afc9a3040327594b0f3c1d3db90a317f93188c114bac807ffdc64eb09097'; // sha256("card1-test-code-fixture-4") — the seeded REDEEMED code
const WALKUP_TOKEN_HASH    = '9dd1e09332d19dfa4055a8d36ae633753b58277bc2101e9ee0e36f7612466d3c'; // sha256("walkup-not-yet-synced-1") — in no replica

const FIXTURE_1_PAYLOAD = 'https://hq.yumyums.kitchen/r/card1-test-code-fixture-1';
const FIXTURE_4_PAYLOAD = 'https://hq.yumyums.kitchen/r/card1-test-code-fixture-4';
// The #10 hybrid with the D-KR3 embedded-offer descriptor (candidate reader
// encoding, locked for real at Activity E): #o=<base64url(JSON)>.
const WALKUP_DESCRIPTOR = 'eyJsYWJlbCI6IkZyZWUgc2lkZSBvZiB3aW5ncyIsImNhbXBhaWduX2lkIjoiYTAwMDAwMDAtMDAwMC00MDAwLTgwMDAtMDAwMDAwMDAwMDAxIiwiZXhwaXJlc19hdCI6IjIwMjgtMDEtMDFUMDA6MDA6MDBaIiwiZmFjZV92YWx1ZSI6Mn0';
const WALKUP_PAYLOAD = 'https://hq.yumyums.kitchen/r/walkup-not-yet-synced-1#o=' + WALKUP_DESCRIPTOR;

// Local-replica rows mirroring supabase/seed.sql (§4 shape, RxDB schema fields).
function fixture1Row(overrides = {}) {
  return Object.assign({
    id: 'c0000000-0000-4000-8000-000000000001',
    token_hash: FIXTURE_1_TOKEN_HASH,
    campaign_id: 'a0000000-0000-4000-8000-000000000001',
    expires_at: '2028-01-01T00:00:00.000Z',
    redeemed_at: null,
    redeemed_by: null,
    updated_at: '2026-09-01T00:00:00.000Z',
  }, overrides);
}
function fixture4RedeemedRow(overrides = {}) {
  return Object.assign({
    id: 'c0000000-0000-4000-8000-000000000004',
    token_hash: FIXTURE_4_TOKEN_HASH,
    campaign_id: 'a0000000-0000-4000-8000-000000000001',
    expires_at: '2028-01-01T00:00:00.000Z',
    redeemed_at: '2026-09-01T12:00:00.000Z',
    redeemed_by: 'test-device-seed',
    updated_at: '2026-09-01T12:00:00.000Z',
  }, overrides);
}

// The HIGH fixture (card requires-online-replication): seed code …0005 belongs
// to campaign …0002 — the $40 catering credit, requires_online=true. These are
// the committed supabase/seed.sql literals, mirrored the way the campaigns
// pull replica lands them (§10 minimal row: id + flag + updated_at).
const FIXTURE_5_TOKEN_HASH = '60f4743622b18f559fb115e1c3329fad70e0168a3b05328361792477615db7cf'; // sha256("card1-test-code-fixture-5")
const FIXTURE_5_PAYLOAD = 'https://hq.yumyums.kitchen/r/card1-test-code-fixture-5';
function fixture5HighRow(overrides = {}) {
  return Object.assign({
    id: 'c0000000-0000-4000-8000-000000000005',
    token_hash: FIXTURE_5_TOKEN_HASH,
    campaign_id: 'a0000000-0000-4000-8000-000000000002',
    expires_at: '2028-01-01T00:00:00.000Z',
    redeemed_at: null,
    redeemed_by: null,
    updated_at: '2026-09-01T00:00:00.000Z',
  }, overrides);
}
function campaignHighRow(overrides = {}) {
  return Object.assign({
    id: 'a0000000-0000-4000-8000-000000000002',
    requires_online: true,
    updated_at: '2026-09-01T00:00:00.000Z',
  }, overrides);
}
// The LOW campaign — …0001, requires_online=false, the $2 free-side offer that
// fixture1Row/fixture4RedeemedRow belong to (committed seed.sql literals).
//
// 🛑 Seeding this is NOT decoration (card refusal-holds-before-sync). Since the
// policy source became FAIL-CLOSED for a known code whose campaign is
// unresolved, a test that seeds `codes`/`offers` but no `campaigns` row is
// modelling the B-432 window, not an ordinary offline scan — and the gate it
// gets is `requires-online-unresolved`, correctly. Every test below that means
// "an ordinary, offline-eligible campaign" must say so by seeding it.
function campaignLowRow(overrides = {}) {
  return Object.assign({
    id: 'a0000000-0000-4000-8000-000000000001',
    requires_online: false,
    updated_at: '2026-09-01T00:00:00.000Z',
  }, overrides);
}

async function openScanner(page) {
  await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);
  await page.goto('/marketing.html');
  await page.waitForFunction(() => window.MarketingScan && window.MarketingScan.booted === true);
}

async function seedLocal(page, docs) {
  await page.evaluate(async (d) => {
    const MS = window.MarketingScan;
    for (const row of (d.codes || [])) await MS.collections.codes.upsert(row);
    for (const row of (d.offers || [])) await MS.collections.offers.upsert(row);
    // Tolerant on purpose (card requires-online-replication): on the
    // pre-change tree the campaigns collection does not exist, and the
    // red-first run must observe the actual policy behavior (the override
    // being offered), not a seeding crash.
    if (MS.collections.campaigns) {
      for (const row of (d.campaigns || [])) await MS.collections.campaigns.upsert(row);
    }
  }, docs);
}

async function scanText(page, payload) {
  await page.evaluate(async (p) => { await window.MarketingScan.scanText(p); }, payload);
}

async function scanImage(page, fixtureFile) {
  await page.setInputFiles('#scan-file', path.join(__dirname, 'fixtures', fixtureFile));
  await page.waitForSelector('#scan-result[data-kind]', { state: 'attached' });
}

test.describe('Camera scanner decode (card camera-scanner-decode)', () => {

  // ── decode-from-image + the §12/§4 hash contract ───────────────────────────

  test('a printed QR decodes from an image and its on-device hash equals the committed seed literal', async ({ page }) => {
    await openScanner(page);
    // Nothing seeded and no descriptor on this payload → unknownCode (F2 —
    // display here; the override path is submit-time, Card 6). The load-bearing
    // assertion is the hash attribute: the browser's WebCrypto SHA-256 of the
    // token EXTRACTED from the #10 URL wrapper IS the committed seed literal.
    await scanImage(page, 'qr-fixture-1.png');
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-token-hash', FIXTURE_1_TOKEN_HASH);
    await expect(result).toHaveAttribute('data-kind', 'unknownCode');
    await expect(result).toContainText('Code not recognized');
    // §12 hygiene: the raw token never lands in the DOM — only its hash.
    const html = await result.innerHTML();
    expect(html).not.toContain('card1-test-code-fixture-1');
  });

  // ── done_when: synced customer's offer, offline, from the replica ──────────

  test('done_when: a synced customer\'s offer resolves OFFLINE from the local replica (image decode)', async ({ page }) => {
    await openScanner(page);
    await seedLocal(page, { offers: [fixture1Row()], codes: [fixture1Row()] });
    await page.context().setOffline(true);
    await scanImage(page, 'qr-fixture-1.png');
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'offerReady');
    await expect(result).toHaveAttribute('data-source', 'replica');
    await expect(result).toHaveAttribute('data-token-hash', FIXTURE_1_TOKEN_HASH);
    const rows = result.locator('.offer-row');
    await expect(rows).toHaveCount(1);
    await expect(rows.first()).toHaveAttribute('data-code-id', 'c0000000-0000-4000-8000-000000000001');
    await expect(rows.first()).toContainText('Expires');
    // F5: the app DISPLAYS offers; staff apply in Toast by hand. No pick/apply
    // control exists inside an offer row.
    await expect(rows.first().locator('button')).toHaveCount(0);
    await expect(result.locator('.result-note')).toContainText('Toast');
    await expect(result.locator('.result-note')).toContainText('never auto-applies');
  });

  // ── done_when: un-synced customer falls back to the QR-embedded offer ──────

  test('done_when: an un-synced customer falls back OFFLINE to the QR-embedded offer, marked unverified (image decode)', async ({ page }) => {
    await openScanner(page);
    // Nothing seeded — the walk-up customer's hash is in NO replica (D-KR3).
    await page.context().setOffline(true);
    await scanImage(page, 'qr-embedded.png');
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'embeddedOffer');
    await expect(result).toHaveAttribute('data-source', 'embedded');
    await expect(result).toHaveAttribute('data-token-hash', WALKUP_TOKEN_HASH);
    await expect(result).toContainText('Free side of wings');
    // The trust note (roadmap Activity E trust note): embedded is display-only,
    // unauthenticated — staff see that it is not server-verified.
    await expect(result.locator('.result-note')).toContainText('not yet verified');
    // Submit-time handling (F2) is Card 6's — its mount slot is present.
    await expect(result.locator('#scan-submit-slot')).toHaveCount(1);
  });

  // ── resolution order: replica beats embedded; neither → unknownCode ────────

  test('resolution order: the local replica beats the QR-embedded offer when both exist', async ({ page }) => {
    await openScanner(page);
    await seedLocal(page, {
      offers: [fixture1Row({ id: 'c0000000-0000-4000-8000-00000000w001', token_hash: WALKUP_TOKEN_HASH })],
      codes:  [fixture1Row({ id: 'c0000000-0000-4000-8000-00000000w001', token_hash: WALKUP_TOKEN_HASH })],
    });
    await scanText(page, WALKUP_PAYLOAD); // carries the descriptor too
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'offerReady');
    await expect(result).toHaveAttribute('data-source', 'replica');
    // The embedded label is NOT what renders — the replica list is.
    await expect(result.locator('.offer-row')).toHaveCount(1);
  });

  test('a token in neither replica with no descriptor resolves unknownCode (F2 display)', async ({ page }) => {
    await openScanner(page);
    await scanText(page, 'https://hq.yumyums.kitchen/r/never-seen-token-xyz');
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'unknownCode');
    await expect(result).toContainText('Code not recognized');
  });

  test('a payload that is not a Yumyums code resolves invalidPayload, loudly', async ({ page }) => {
    await openScanner(page);
    await scanText(page, 'https://example.com/not-our-shape');
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'invalidPayload');
    await expect(result).toContainText('Not a Yumyums code');
  });

  // ── F3: stale local "already used" ─────────────────────────────────────────

  test('F3 offline: a locally-redeemed code rejects immediately as spentLocally', async ({ page }) => {
    await openScanner(page);
    await seedLocal(page, { codes: [fixture4RedeemedRow()], offers: [fixture4RedeemedRow()] });
    // Card 6 landed the REAL reachability probe (#13): the resolver's online
    // answer is now the machine's, so OFFLINE is forced through the probe —
    // not assumed from Card 5's () => false default (which Card 6's boot
    // replaces, racing this test's scan).
    await page.waitForFunction(() => window.MarketingSubmit && window.MarketingSubmit.booted === true);
    await killProbe(page);
    await scanText(page, FIXTURE_4_PAYLOAD);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'spentLocally');
    await expect(result).toContainText('Already used');
    await expect(result).toContainText('test-device-seed'); // when + which device (F3/§16)
    // A hard offline reject mounts NO submit slot.
    await expect(result.locator('#scan-submit-slot')).toHaveCount(0);
  });

  test('F3 online: the local redeemed flag does NOT reject — the server decides at submit', async ({ page }) => {
    await openScanner(page);
    await seedLocal(page, { codes: [fixture4RedeemedRow()], offers: [fixture4RedeemedRow()] });
    // The ONLINE direction of the 9c6f04e race fix: this test's
    // setOnlineProbe(() => true) can be overwritten mid-scan by Card 6's boot
    // installing the machine-fed probe while the machine still reads its
    // honest offline start. Same remedy as the offline sibling — wait for the
    // submit flow, then force ONLINE through the REAL probe (the /health
    // endpoint is live in this test). Assertions unchanged.
    await page.waitForFunction(() => window.MarketingSubmit && window.MarketingSubmit.booted === true);
    await page.evaluate(() => window.MarketingSubmit.probeNow());
    await expect(page.locator('#scan-conn')).toHaveAttribute('data-conn', 'online');
    await scanText(page, FIXTURE_4_PAYLOAD);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'deferToServer');
    await expect(result).toContainText('used on this device');
    await expect(result).toContainText('server has the final say');
    // The server call happens at submit — Card 6 mounts into this slot.
    await expect(result.locator('#scan-submit-slot')).toHaveCount(1);
  });

  // ── §5.1: offline expiry via the offset clock, and its reload round-trip ───

  test('offline expiry uses clock.isExpired — a +2h server offset expires a code raw Date.now() calls live', async ({ page }) => {
    await openScanner(page);
    const soon = new Date(Date.now() + 60 * 60 * 1000).toISOString(); // +1h device time
    await seedLocal(page, {
      codes:  [fixture1Row({ expires_at: soon })],
      offers: [fixture1Row({ expires_at: soon })],
    });
    // Server is 2h ahead of the device (offset = serverNow − deviceNow = +2h):
    // clock.now() > expires_at even though Date.now() < expires_at.
    await page.evaluate(() => {
      window.MarketingScan.clock.captureFromResponse({
        headers: { get: () => new Date(Date.now() + 2 * 60 * 60 * 1000).toUTCString() },
      });
    });
    await scanText(page, FIXTURE_1_PAYLOAD);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'expiredLocally');
    await expect(result).toContainText('Expired');
  });

  test('the clock state round-trips a reload (persist → initialState) so a reloaded-offline device keeps its offset', async ({ page }) => {
    await openScanner(page);
    // A capture persists…
    await page.evaluate(() => {
      window.MarketingScan.clock.captureFromResponse({
        headers: { get: () => new Date(Date.now() + 123456).toUTCString() },
      });
    });
    const persisted = await page.evaluate(() => JSON.parse(localStorage.getItem('hq_marketing_clock_v1')).offset_ms);
    expect(Math.abs(persisted - 123456)).toBeLessThan(5000); // Date-header = whole-second resolution + eval latency
    // …and a reload boots FROM the persisted state (0 captures — the
    // reloaded-offline device), still carrying the offset.
    await page.reload();
    await page.waitForFunction(() => window.MarketingScan && window.MarketingScan.booted === true);
    const after = await page.evaluate(() => ({
      offset: window.MarketingScan.clock.offsetMs,
      captures: window.MarketingScan.clock.captures,
    }));
    expect(Math.abs(after.offset - 123456)).toBeLessThan(5000);
    expect(after.captures).toBe(0);
  });

  // ── engineering-call guards: hash caching, serialized enqueue, F6 re-scan ──

  test('hash caching: repeated scans of one token digest once', async ({ page }) => {
    await openScanner(page);
    await scanText(page, FIXTURE_1_PAYLOAD);
    await scanText(page, FIXTURE_1_PAYLOAD);
    const stats = await page.evaluate(() => window.MarketingScan.hasherStats());
    expect(stats.misses).toBe(1);
    expect(stats.hits).toBeGreaterThanOrEqual(1);
  });

  test('enqueue is serialized per code: two rapid same-code enqueues insert exactly one live attempt', async ({ page }) => {
    await openScanner(page);
    // The landed Card 3 note: enqueueAttempt's dedupe is find-then-insert, NOT
    // atomic — two concurrent raw calls can both insert. Card 5's wrapper
    // serializes per code_id; Card 6 must enqueue through it.
    const out = await page.evaluate(async () => {
      const MS = window.MarketingScan;
      const fields = { code_id: 'c0000000-0000-4000-8000-000000000002', device_id: 'dev-c5-test' };
      const [a, b] = await Promise.all([MS.enqueue(fields), MS.enqueue(fields)]);
      const docs = await MS.collections.scan_attempts
        .find({ selector: { code_id: fields.code_id } }).exec();
      return { deduped: [a.deduped, b.deduped].sort(), count: docs.length };
    });
    expect(out.count).toBe(1);
    expect(out.deduped).toEqual([false, true]);
  });

  test('F6: re-scanning the same code re-shows the result instead of erroring or duplicating', async ({ page }) => {
    await openScanner(page);
    await seedLocal(page, { offers: [fixture1Row()], codes: [fixture1Row()] });
    await scanText(page, FIXTURE_1_PAYLOAD);
    await scanText(page, FIXTURE_1_PAYLOAD);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'offerReady');
    await expect(result.locator('.offer-row')).toHaveCount(1);
  });

  // ── the camera leg's ERROR state (UI-R6) — the live leg is ATTENDED ────────

  test('camera failure is loud and retryable (headless has no camera; live decode is the attended morning check)', async ({ page }) => {
    await openScanner(page);
    await page.click('[data-action="start-camera"]');
    const err = page.locator('#scanner-host .cam-error');
    await expect(err).toBeVisible();
    await expect(err).toContainText('Camera unavailable');
    // Retry affordance: the same labeled action, still present and clickable.
    await expect(page.locator('[data-action="start-camera"]')).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Redemption submit flow — card redemption-submit-flow (run 20260905,
// Activity C, design §8/§13/§16/§19.4 F1/F2/F6, P-KR4; spike
// .night-crew/knowledge/spikes/activity-c-scanner-screen/redemption-submit-flow.md)
// ═══════════════════════════════════════════════════════════════════════════
//
// RED-FIRST (greenfield): every test in this describe was written and RUN
// against the pre-change tree (Cards 1–5 landed, no submit flow: no
// lib/xstate.umd.min.js, no marketing/submit-*.js, no #scan-conn indicator,
// no window.MarketingSubmit), where it reds as a set. Evidence:
// .night-crew/runs/2026-09-05-autonomous/card6-red.log and the ## Red-first
// section of merge-intents/redemption-submit-flow.md.
//
// What this describe pins (the roadmap's done_when, verbatim clauses):
//   * confirm-then-burn (§13): NO path to "redeemed" without a
//     validator-passing Toast order number — zero POSTs without it;
//   * the three offline branches under a KILLED reachability probe:
//     no-permission blocked / entitlement override offered / high-value
//     (requires_online=true) refused even WITH the entitlement;
//   * the persistent indicator flips and submit re-arms LIVE when the probe
//     recovers — no page interaction (P-KR4);
//   * an unknownCode offline override writes offline_override=true AND
//     unverified_code=true (F2) through the serialized enqueue;
//   * F6 session semantics: same-code re-scan is a no-op; a different code
//     prompts finish-current-customer; dismiss returns to the interrupted
//     state; the machine's modeled unexpectedEvent state is loud + retryable.
//
// The machine itself is gated in Node (tests/machine/: 18-sequence
// conformance + lockstep fuzz with per-step liveness, both in throw mode);
// this describe drives the PAGE — the production 'model' build.

const SECOND_TOKEN_PAYLOAD = 'https://hq.yumyums.kitchen/r/second-customer-c6-token';
const UNKNOWN_TOKEN_PAYLOAD = 'https://hq.yumyums.kitchen/r/never-seen-c6-override-token';

async function openSubmitScanner(page, email = ADMIN_EMAIL, password = USER_PASSWORD) {
  if (email === ADMIN_EMAIL) password = ADMIN_PASSWORD;
  await loginAs(page, email, password);
  await page.goto('/marketing.html');
  await page.waitForFunction(() =>
    window.MarketingScan && window.MarketingScan.booted === true
    && window.MarketingSubmit && window.MarketingSubmit.booted === true);
}

// Kill / restore the #13 reachability signal deterministically: the probe
// targets GET /api/v1/health (the same origin the submit POST needs), and
// probeNow() resolves after the probe result has been fed to the machine —
// tests never wait on the 10s production cadence.
async function killProbe(page) {
  await page.route('**/api/v1/health**', (r) => r.abort());
  await page.evaluate(() => window.MarketingSubmit.probeNow());
  await expect(page.locator('#scan-conn')).toHaveAttribute('data-conn', 'offline');
}
async function restoreProbe(page) {
  await page.unroute('**/api/v1/health**');
  await page.evaluate(() => window.MarketingSubmit.probeNow());
}

// Intercept the online submit endpoint: capture request bodies, answer with a
// scripted verdict (the dev/test arbiter is fail-closed 503 by design — Card
// 7's contract — so verdicts are mocked at the network layer).
async function mockRedeem(page, result = 'redeemed') {
  const calls = [];
  await page.route('**/api/v1/marketing/redeem', async (route) => {
    calls.push(route.request().postDataJSON());
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ result, race_lost_reconciled: false }),
    });
  });
  return calls;
}

async function scanAndReady(page, payload, orderNumber) {
  await scanText(page, payload);
  await expect(page.locator('#ms-order')).toBeVisible();
  await page.fill('#ms-order', orderNumber);
}

test.describe('Redemption submit flow (card redemption-submit-flow)', () => {

  // ── the persistent indicator exists from boot (done_when: visible indicator) ──

  test('a persistent online/offline indicator renders from boot and reads online on a live probe', async ({ page }) => {
    await openSubmitScanner(page);
    const conn = page.locator('#scan-conn');
    await expect(conn).toBeVisible();
    await expect(conn).toHaveAttribute('data-conn', 'online');
    await expect(conn).toContainText('Online');
    // …and it is still there mid-session (persistent, not a toast).
    await seedLocal(page, { offers: [fixture1Row()], codes: [fixture1Row()] });
    await scanText(page, FIXTURE_1_PAYLOAD);
    await expect(conn).toBeVisible();
  });

  // ── §13 confirm-then-burn: order number COMPLETES the redemption ───────────

  test('confirm-then-burn: no path to redeemed without a validator-passing Toast order number', async ({ page }) => {
    await openSubmitScanner(page);
    const calls = await mockRedeem(page, 'redeemed');
    await seedLocal(page, { offers: [fixture1Row()], codes: [fixture1Row()] });
    await scanText(page, FIXTURE_1_PAYLOAD);

    // The submit flow mounts in Card 5's slot with a required order field.
    const flow = page.locator('#ms-flow');
    await expect(flow).toHaveAttribute('data-mstate', 'offerReady');
    const submit = page.locator('[data-action="ms-submit"]');
    await expect(submit).toBeVisible();
    await expect(submit, 'no order number -> submit disabled').toBeDisabled();

    // Format validation (TOAST_ORDER_NUMBER_PLACEHOLDER_PATTERN): a
    // non-matching entry is named invalid and does NOT arm the submit.
    await page.fill('#ms-order', '12ab');
    await expect(page.locator('#ms-order-err')).toBeVisible();
    await expect(submit).toBeDisabled();
    await expect(flow).toHaveAttribute('data-mstate', 'offerReady');

    // A validator-passing number arms it (ORDER_OK -> readyToSubmit)…
    await page.fill('#ms-order', '4321');
    await expect(page.locator('#ms-order-err')).toBeHidden();
    await expect(flow).toHaveAttribute('data-mstate', 'readyToSubmit');
    await expect(submit).toBeEnabled();

    // …and only then can the redemption complete.
    expect(calls.length, 'zero POSTs before a valid order number').toBe(0);
    await submit.click();
    await expect(flow).toHaveAttribute('data-mstate', 'redeemed');
    await expect(flow).toContainText('Redeemed');
    expect(calls.length).toBe(1);
    const body = calls[0];
    expect(body.token_hash).toBe(FIXTURE_1_TOKEN_HASH);
    expect(body.order_number).toBe('4321');
    expect(body.device_id).toBeTruthy();
    expect(body.offline_override).toBe(false);
    expect(body.unverified_code).toBe(false);
    expect(typeof body.scanned_at).toBe('string');
  });

  // ── the three offline branches, each under a KILLED probe ─────────────────

  test('offline branch 1: no entitlement -> blocked, no override affordance', async ({ page }) => {
    const user = await makeUser(page, 'c6-noperm', ['team_member']);
    await openSubmitScanner(page, user.email, USER_PASSWORD);
    const calls = await mockRedeem(page);
    await seedLocal(page, {
      offers: [fixture1Row()], codes: [fixture1Row()], campaigns: [campaignLowRow()],
    });
    await killProbe(page);
    await scanAndReady(page, FIXTURE_1_PAYLOAD, '4321');
    await page.click('[data-action="ms-submit"]');

    const gate = page.locator('#ms-gate');
    await expect(gate).toBeVisible();
    await expect(gate).toHaveAttribute('data-branch', 'no-permission');
    await expect(gate).toContainText('connect to redeem');
    await expect(page.locator('[data-action="ms-override"]')).toHaveCount(0);
    expect(calls.length, 'nothing reached the server').toBe(0);
  });

  test('offline branch 2: the entitlement offers force-submit behind the §13 confirmation', async ({ page }) => {
    await openSubmitScanner(page); // admin holds marketing-offline-override (seeded, #12)
    const calls = await mockRedeem(page);
    await seedLocal(page, {
      offers: [fixture1Row()], codes: [fixture1Row()], campaigns: [campaignLowRow()],
    });
    await killProbe(page);
    await scanAndReady(page, FIXTURE_1_PAYLOAD, '8642');
    await page.click('[data-action="ms-submit"]');

    const gate = page.locator('#ms-gate');
    await expect(gate).toHaveAttribute('data-branch', 'override');
    await page.click('[data-action="ms-override"]');

    // The §13 confirmation, verbatim risk wording.
    const confirm = page.locator('#ms-confirm');
    await expect(confirm).toBeVisible();
    await expect(confirm).toContainText('double-redemption');
    await page.click('[data-action="ms-confirm-override"]');

    // Terminal-class "queued" card (spike design call), audit-flagged attempt.
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'overridePending');
    await expect(page.locator('#ms-flow')).toContainText('Queued');
    // The audit-flagged write is async behind the state flip — poll for it.
    const readAttempts = () => page.evaluate(async (codeId) => {
      const docs = await window.MarketingScan.collections.scan_attempts
        .find({ selector: { code_id: codeId } }).exec();
      return docs.map((d) => ({
        offline_override: d.offline_override, unverified_code: d.unverified_code,
        pos_order_number: d.pos_order_number, pos_business_date: d.pos_business_date,
        status: d.status,
      }));
    }, 'c0000000-0000-4000-8000-000000000001');
    await expect.poll(async () => (await readAttempts()).length, { timeout: 5000 }).toBe(1);
    const attempt = await readAttempts();
    expect(attempt[0].offline_override).toBe(true);
    expect(attempt[0].unverified_code).toBe(false); // a KNOWN code — F2's flag is for unknown ones
    expect(attempt[0].pos_order_number).toBe('8642');
    expect(attempt[0].pos_business_date).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    expect(attempt[0].status).toBe('pending'); // queued for sync arbitration
    expect(calls.length, 'an offline override never posts directly').toBe(0);
  });

  test('offline branch 3: requires_online=true refuses the override even WITH the entitlement (§8)', async ({ page }) => {
    await openSubmitScanner(page); // admin — entitlement held, and still refused
    const calls = await mockRedeem(page);
    // REAL DATA (card requires-online-replication, run 20260906): NO
    // setCampaignPolicy injection. The HIGH fixture — code …0005 → campaign
    // …0002, requires_online=true, the committed seed literals — is seeded
    // into the local replicas exactly as the campaigns pull replica lands it;
    // the DEFAULT policy source reads that replica. On the pre-change tree
    // this test reds with data-branch "override": the $40 code is overridable
    // offline exactly like the $2 one (the spike's rows 3–4).
    await seedLocal(page, {
      offers: [fixture5HighRow()],
      codes: [fixture5HighRow()],
      campaigns: [campaignHighRow()],
    });
    await killProbe(page);
    await scanAndReady(page, FIXTURE_5_PAYLOAD, '4321');
    await page.click('[data-action="ms-submit"]');

    const gate = page.locator('#ms-gate');
    await expect(gate).toHaveAttribute('data-branch', 'requires-online');
    await expect(gate).toContainText(/can.t verify/i);
    await expect(gate).toContainText('try again');
    await expect(page.locator('[data-action="ms-override"]'), 'no override even for a holder of the entitlement').toHaveCount(0);
    expect(calls.length).toBe(0);
  });

  // ── B-432: the refusal must hold BEFORE the campaigns replica has delivered ──
  //
  // RED-FIRST (card refusal-holds-before-sync, run 20260906-2): this is the
  // shipped branch-3 test above with ONLY the `campaigns:` seed removed — the
  // exact morning-triage reproduction, zero production code mutated. It models
  // the window B-432 names: the codes/offers replicas have delivered the $40
  // `requires_online=true` code, the campaigns replica has NOT (first sync, a
  // campaigns pull that 5xx's while codes succeeds, or a new campaign whose
  // codes arrive first). On the pre-change tree the policy source's Map is
  // empty, `policyFor` answers null, submit-flow coerces null → false, and the
  // gate renders data-branch="override": the high-value code is offline-
  // overridable by an entitlement holder. Evidence:
  // .night-crew/runs/2026-09-06-2-autonomous/c1-red-branch3-nocampaign.log
  //
  // 🛑 The refusal here is NOT unconditional — it holds because the CODE is
  // known (`campaign_id` is non-null) and its campaign is unresolved. A
  // genuinely-unknown code (no campaign named at all) keeps its override and
  // its unverified warning — decision 166, pinned by the F2 test below.
  test('B-432: a requires_online=true code is refused while its campaign is UNRESOLVED (branch-3 minus the campaigns: seed)', async ({ page }) => {
    await openSubmitScanner(page); // admin — entitlement held, and still refused
    const calls = await mockRedeem(page);
    await seedLocal(page, {
      offers: [fixture5HighRow()],
      codes: [fixture5HighRow()],
      // NO campaigns row — the campaigns replica has not delivered.
    });
    await killProbe(page);
    await scanAndReady(page, FIXTURE_5_PAYLOAD, '4321');
    await page.click('[data-action="ms-submit"]');

    const gate = page.locator('#ms-gate');
    await expect(gate).toBeVisible();
    // The unresolved case gets its OWN branch + copy (build-fact 6, decided
    // against UI-R3/R6): "online verification is required" asserts a fact this
    // device does not have — the policy is unresolved, not known-true.
    await expect(gate).toHaveAttribute('data-branch', 'requires-online-unresolved');
    await expect(gate).toContainText(/can.t verify/i);
    await expect(gate).toContainText('try again');
    await expect(gate).toContainText(/sync/i);
    await expect(
      page.locator('[data-action="ms-override"]'),
      'no override for a KNOWN code whose campaign has not replicated',
    ).toHaveCount(0);
    expect(calls.length).toBe(0);
  });

  // ── P-KR4: the submit control transitions ON ITS OWN when the probe recovers ──

  test('P-KR4: indicator flips and submit re-arms LIVE when reachability returns — zero page interaction', async ({ page }) => {
    await openSubmitScanner(page);
    await mockRedeem(page);
    await seedLocal(page, {
      offers: [fixture1Row()], codes: [fixture1Row()], campaigns: [campaignLowRow()],
    });
    await killProbe(page);
    await scanAndReady(page, FIXTURE_1_PAYLOAD, '4321');
    await page.click('[data-action="ms-submit"]');
    await expect(page.locator('#ms-gate')).toBeVisible();
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'blockedOffline');

    // Reachability returns; NOTHING else is touched.
    await restoreProbe(page);
    await expect(page.locator('#scan-conn')).toHaveAttribute('data-conn', 'online');
    await expect(page.locator('#ms-flow'), 'the gate resumed the pre-gate state on its own').toHaveAttribute('data-mstate', 'readyToSubmit');
    const submit = page.locator('[data-action="ms-submit"]');
    await expect(submit).toBeEnabled();
    // …and the resumed control actually submits.
    await submit.click();
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'redeemed');
  });

  // ── F2: the unknown-code override writes BOTH flags ───────────────────────

  test('F2: an unknownCode offline override warns unverifiable and writes offline_override=true AND unverified_code=true', async ({ page }) => {
    await openSubmitScanner(page);
    const calls = await mockRedeem(page);
    await killProbe(page);
    await scanText(page, UNKNOWN_TOKEN_PAYLOAD);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'unknownCode');
    const tokenHash = await result.getAttribute('data-token-hash');
    expect(tokenHash).toMatch(/^[0-9a-f]{64}$/);

    await page.fill('#ms-order', '99');
    await page.click('[data-action="ms-submit"]');
    await expect(page.locator('#ms-gate')).toHaveAttribute('data-branch', 'override');
    await page.click('[data-action="ms-override"]');

    // F2's decided wording: NEITHER the offer NOR prior use can be verified.
    const confirm = page.locator('#ms-confirm');
    await expect(confirm).toContainText('offer');
    await expect(confirm).toContainText('prior use');
    await expect(confirm).toContainText('verified');
    await page.click('[data-action="ms-confirm-override"]');
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'overridePending');

    // engineering call: an unknown code's local code_id IS its token_hash.
    // The write is async behind the state flip — poll for it.
    const readAttempts = () => page.evaluate(async (codeId) => {
      const docs = await window.MarketingScan.collections.scan_attempts
        .find({ selector: { code_id: codeId } }).exec();
      return docs.map((d) => ({
        offline_override: d.offline_override,
        unverified_code: d.unverified_code,
        policy_unresolved: d.policy_unresolved,
      }));
    }, tokenHash);
    await expect.poll(async () => (await readAttempts()).length, { timeout: 5000 }).toBe(1);
    const attempt = await readAttempts();
    expect(attempt[0].offline_override).toBe(true);
    expect(attempt[0].unverified_code).toBe(true);
    // done_when clause 2, the CONTROL half (card refusal-holds-before-sync):
    // the campaigns policy source is healthy here, so this override is a
    // genuinely-unknown-campaign one — t,f. The paired t,t case is the test
    // below; the two together are what "distinguishable in the attempt record"
    // means.
    expect(attempt[0].policy_unresolved).toBe(false);
    expect(calls.length).toBe(0);
  });

  // ── done_when clause 2: the two overrides are DISTINGUISHABLE in the record ──
  //
  // "the campaigns-replica failure path is distinguishable from a genuinely-
  // unknown campaign in the attempt record." Both land unverified_code=true
  // offline overrides; `policy_unresolved` is what tells them apart, and
  // neither gets a new terminal status (§9/§19 taxonomy unchanged — the card's
  // PARK line).
  //
  // The failure is injected at the policy seam rather than by breaking a real
  // replica: the page under test never starts one (sync provisioning is a
  // later card), and the seam is the shipped surface the browser would read
  // from a broken source anyway. The REAL replica-erroring path — error$
  // latching an attributable HTTP status while awaitInitialReplication stays
  // pending — is proved against the live substrate by
  // marketing/sync/harness/refusal-run.sh.
  test('done_when(2): a replica-FAILURE override lands policy_unresolved=true, distinguishable from the genuinely-unknown one', async ({ page }) => {
    await openSubmitScanner(page);
    const calls = await mockRedeem(page);
    // The campaigns replica is unusable: every known campaign is unresolved,
    // and the source says so about itself.
    await page.evaluate(() => {
      window.MarketingSubmit.setCampaignPolicy(
        (campaignId) => (campaignId ? { requiresOnline: true, unresolved: true } : null),
        () => true,
      );
    });
    await killProbe(page);
    await scanText(page, UNKNOWN_TOKEN_PAYLOAD);
    const result = page.locator('#scan-result');
    await expect(result).toHaveAttribute('data-kind', 'unknownCode');
    const tokenHash = await result.getAttribute('data-token-hash');

    // decision 166 still holds under a FAILING source: a code that names no
    // campaign keeps its override and its unverified warning.
    await page.fill('#ms-order', '77');
    await page.click('[data-action="ms-submit"]');
    await expect(page.locator('#ms-gate')).toHaveAttribute('data-branch', 'override');
    await page.click('[data-action="ms-override"]');
    await expect(page.locator('#ms-confirm')).toContainText('prior use');
    await page.click('[data-action="ms-confirm-override"]');
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'overridePending');

    const readAttempts = () => page.evaluate(async (codeId) => {
      const docs = await window.MarketingScan.collections.scan_attempts
        .find({ selector: { code_id: codeId } }).exec();
      return docs.map((d) => ({
        offline_override: d.offline_override,
        unverified_code: d.unverified_code,
        policy_unresolved: d.policy_unresolved,
        status: d.status,
      }));
    }, tokenHash);
    await expect.poll(async () => (await readAttempts()).length, { timeout: 5000 }).toBe(1);
    const [attempt] = await readAttempts();
    expect(attempt.offline_override).toBe(true);
    expect(attempt.unverified_code).toBe(true);
    expect(attempt.policy_unresolved, 'the replica-failure discriminator').toBe(true);
    expect(attempt.status, 'no new terminal status — §9/§19 taxonomy unchanged').toBe('pending');
    expect(calls.length).toBe(0);
  });

  // ── rider B-434(c): campaignPolicyFor had no test. The fail-closed predicate
  // gives it one, and it is the predicate itself that is worth pinning — the
  // three arms are what keep B-432 closed AND decision 166 alive.
  test('B-434(c): the policy predicate — resolved flag / KNOWN-but-unresolved fails closed / no campaign stays null (decision 166)', async ({ page }) => {
    await openSubmitScanner(page);
    // Only the HIGH campaign is replicated. …0001 is deliberately absent.
    await seedLocal(page, {
      codes: [fixture5HighRow()], offers: [fixture5HighRow()], campaigns: [campaignHighRow()],
    });
    const HIGH_ID = 'a0000000-0000-4000-8000-000000000002';
    const ABSENT_ID = 'a0000000-0000-4000-8000-000000000001';

    const probe = () => page.evaluate(async ([high, absent]) => {
      const S = window.MarketingSubmit;
      // The reactive Map mirror settles a tick after the upsert.
      for (let i = 0; i < 50 && !(S.campaignPolicyFor(high) || {}).requiresOnline; i++) {
        await new Promise((r) => setTimeout(r, 50));
      }
      return {
        high: S.campaignPolicyFor(high),
        absent: S.campaignPolicyFor(absent),
        none: S.campaignPolicyFor(null),
        unresolvedHigh: S.policyUnresolvedFor(high),
        unresolvedAbsent: S.policyUnresolvedFor(absent),
        unresolvedNone: S.policyUnresolvedFor(null),
      };
    }, [HIGH_ID, ABSENT_ID]);

    const p = await probe();
    // 1. Replicated campaign → its actual flag, and it is NOT unresolved.
    expect(p.high).toEqual({ requiresOnline: true, unresolved: false });
    expect(p.unresolvedHigh).toBe(false);
    // 2. KNOWN campaign id that has not replicated → FAIL CLOSED. This is
    //    B-432: pre-card this arm answered null, which the flow coerced to
    //    false and handed the crew a Force-submit button.
    expect(p.absent).toEqual({ requiresOnline: true, unresolved: true });
    expect(p.unresolvedAbsent).toBe(true);
    // 3. A code that names NO campaign still answers null — decision 166's
    //    ratified unknown→false default, preserved by construction.
    expect(p.none, 'decision 166: a genuinely-unknown code is not fail-closed').toBeNull();
    // …and with a healthy source the discriminator for it is false, which is
    // what makes the failure case above distinguishable.
    expect(p.unresolvedNone).toBe(false);
  });

  // ── F6: full session semantics (Card 5 shipped re-show only; this card owns F6) ──

  test('F6: same-code re-scan is a no-op; a different code prompts; dismiss returns to the interrupted state', async ({ page }) => {
    await openSubmitScanner(page);
    await mockRedeem(page);
    await seedLocal(page, { offers: [fixture1Row()], codes: [fixture1Row()] });
    await scanText(page, FIXTURE_1_PAYLOAD);
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'offerReady');

    // Same code again: no-op / re-show — state unchanged, no prompt.
    await scanText(page, FIXTURE_1_PAYLOAD);
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'offerReady');
    await expect(page.locator('#scan-prompt')).toHaveCount(0);
    await expect(page.locator('#scan-result')).toHaveAttribute('data-kind', 'offerReady');

    // Progress, then a DIFFERENT code: finish-current-customer prompt.
    await page.fill('#ms-order', '55');
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'readyToSubmit');
    await scanText(page, SECOND_TOKEN_PAYLOAD);
    const prompt = page.locator('#scan-prompt');
    await expect(prompt).toBeVisible();
    await expect(prompt).toContainText('Finish the current customer');

    // Dismiss: back to the interrupted state — progress preserved (F1's principle).
    await page.click('[data-action="ms-dismiss"]');
    await expect(prompt).toHaveCount(0);
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'readyToSubmit');
    await expect(page.locator('#ms-order')).toHaveValue('55');
    await expect(page.locator('#scan-result')).toHaveAttribute('data-kind', 'offerReady');

    // Prompt again; this time move on — the session clears for the next customer.
    await scanText(page, SECOND_TOKEN_PAYLOAD);
    await expect(prompt).toBeVisible();
    await page.click('[data-action="ms-prompt-next"]');
    await expect(prompt).toHaveCount(0);
    await expect(page.locator('#scan-result')).toBeHidden();
    // The new customer scans fresh (no stale prompt from the dead session).
    await scanText(page, SECOND_TOKEN_PAYLOAD);
    await expect(page.locator('#scan-result')).toHaveAttribute('data-kind', 'unknownCode');
  });

  // ── the ⚠️ card: already used, when + which device (Card 3's flip source) ──

  test('already_used verdict renders when + which device from the local codes replica', async ({ page }) => {
    await openSubmitScanner(page);
    await mockRedeem(page, 'already_used');
    // The replica knows the winner (Card 3's flip data source: codes pull, never scan_attempts).
    await seedLocal(page, {
      codes: [fixture4RedeemedRow({ redeemed_by: 'device-zeta' })],
      offers: [fixture4RedeemedRow({ redeemed_by: 'device-zeta' })],
    });
    await scanText(page, FIXTURE_4_PAYLOAD); // online -> deferToServer -> server decides
    await expect(page.locator('#scan-result')).toHaveAttribute('data-kind', 'deferToServer');
    await page.fill('#ms-order', '31');
    await page.click('[data-action="ms-submit"]');
    const flow = page.locator('#ms-flow');
    await expect(flow).toHaveAttribute('data-mstate', 'alreadyUsed');
    await expect(flow).toContainText('Already used');
    await expect(flow).toContainText('device-zeta');
  });

  // ── the strict machine's production shape: modeled, loud, retryable ───────

  test('an undeclared machine event raises the modeled unexpectedEvent card — loud, named, retryable, never a dead actor', async ({ page }) => {
    await openSubmitScanner(page);
    await seedLocal(page, { offers: [fixture1Row()], codes: [fixture1Row()] });
    await scanText(page, FIXTURE_1_PAYLOAD);
    await page.evaluate(() => { window.MarketingSubmit.machine.send('BOGUS_EVENT'); });

    const oops = page.locator('#ms-unexpected');
    await expect(oops).toBeVisible();
    await expect(oops).toContainText('BOGUS_EVENT'); // names the event (loud, UI-R6)
    const alive = await page.evaluate(() => window.MarketingSubmit.machine.alive());
    expect(alive, 'the production build never kills the actor').toBe(true);

    // Retryable: back to the interrupted state, session intact.
    await page.click('[data-action="ms-retry"]');
    await expect(page.locator('#ms-flow')).toHaveAttribute('data-mstate', 'offerReady');
  });
});
