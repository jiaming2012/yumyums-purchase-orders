// tests/states-sync-rxdb-conflict-notice.spec.js
//
// CLAUDE.md's SELF-VERIFICATION RITUAL for `sync-rxdb-conflict-notice-ui`
// (overnight-20260801, C2). "This environment is headless — verify via
// screenshots, not imagination."
//
// Every row of the card's State Enumeration Table (`.planning/phases/
// sync-rxdb-conflict-notice/PLAN.md`) is FORCED here on the real page, then
// screenshotted in BOTH colour schemes so the PNGs can be read back with a
// multimodal Read and compared to the visual contract.
//
// ===========================================================================
// 🔴 GUARD INTEGRITY — B-22/B-23/B-24. READ THIS BEFORE ADDING A ROW.
// ===========================================================================
// A screenshot spec is exactly the kind of check whose subject set can go
// empty: a sheet that rendered NOTHING would screenshot cleanly and pass. So
// every state below declares a POPULATION FLOOR and asserts it before the
// shutter — the number of `.cf` rows, `.cg` groups and banner lines that must
// be on screen. `shot()` refuses to fire on an empty sheet.
//
// Mutation-tested while writing: seeding zero records into the `success` state
// produced a clean, empty, entirely passable PNG. It now reds on the floor.
//
// ===========================================================================
// WHY THE STORE IS SEEDED RATHER THAN PRODUCED.
// ===========================================================================
// The sheet renders from the durable local conflict record; the record is
// written when `conflict$` fires; `conflict$` fires when replication runs; and
// replication is deliberately NOT started in this tree (`HQ_SYNC_REST_URL` is
// unset everywhere and the /sync door answers 503 until row-visibility RLS
// lands). `sync-hard-cutover` switches the producer on. Seeding the store is
// therefore the only honest way to verify these screens today, and the seam it
// uses (`window.HQConflictNotice.mount({store})`) is the same one the cutover
// card will hand a real RxDB collection to.

const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

const SHOT_DIR = path.join(__dirname, '..', 'test-results', 'states-sync-rxdb-conflict-notice');
fs.mkdirSync(SHOT_DIR, { recursive: true });

async function login(page) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL((url) => !url.pathname.includes('login'));
}

// ---------------------------------------------------------------------------
// Fixtures — the signed mockup's own data, so plate and render are comparable
// side by side rather than by translation.
// ---------------------------------------------------------------------------

const DOC_A = 'sub-9f31c4b2-1111-1111-1111-111111111111';
const DOC_B = 'sub-2b70ae00-2222-2222-2222-222222222222';
const NOW = '2026-07-28T09:00:00-04:00';

const BASE = {
  doc_id: DOC_A,
  collection: 'responses',
  submission_id: DOC_A,
  field_removed: false,
  checklist_name: 'Opening — Truck A',
  checklist_date: 'Mon Jul 27',
  overwritten_by: 'u-dana',
  overwritten_by_name: 'Dana M.',
  status: 'open',
  undone: false,
  failure: null,
};

const COOLER = {
  ...BASE,
  id: 'a',
  field_id: 'fld-cooler',
  field_label: 'Walk-in cooler temp',
  field_type: 'temperature',
  display_unit: '°F',
  discarded_value: 38,
  current_value: 41,
  overwritten_at: '2026-07-27T18:12:00-04:00',
};
const SANITIZER = {
  ...BASE,
  id: 'b',
  field_id: 'fld-sanitizer',
  field_label: 'Sanitizer concentration',
  field_type: 'number',
  display_unit: 'ppm',
  discarded_value: 200,
  current_value: 150,
  overwritten_at: '2026-07-27T18:13:00-04:00',
};
const HANDSINK = {
  ...BASE,
  id: 'c',
  field_id: 'fld-sink',
  field_label: 'Hand sink stocked',
  field_type: 'yesno',
  discarded_value: true,
  current_value: false,
  overwritten_at: '2026-07-27T18:14:00-04:00',
};
const FRYER = {
  ...BASE,
  id: 'd',
  field_id: 'fld-fryer',
  field_label: 'Fryer oil filtered',
  field_type: 'yesno',
  discarded_value: true,
  current_value: false,
  overwritten_at: '2026-07-27T18:18:00-04:00',
};
// The diff yielded nothing showable — no discarded value to hand back.
const UNIDENT = (id, at) => ({
  ...BASE,
  id,
  field_id: `fld-unknown-${id}`,
  field_label: null,
  discarded_value: null,
  current_value: null,
  overwritten_at: at,
});
// A-3: the question the owner removed while the phone was offline.
const REMOVED = {
  ...BASE,
  id: 'z',
  field_id: 'fld_prep_sink_temp',
  field_label: 'Prep sink temperature',
  field_type: 'temperature',
  display_unit: '°F',
  field_removed: true,
  discarded_value: 72,
  current_value: undefined,
  overwritten_at: '2026-07-27T18:20:00-04:00',
};
// A-3's FALLBACK: the same row with no label anywhere — which, because nothing
// validates `template_snapshot` (B1's R-C), is also every malformed snapshot.
const REMOVED_NO_LABEL = { ...REMOVED, id: 'z2', field_label: null };

// ---------------------------------------------------------------------------
// Harness.
// ---------------------------------------------------------------------------

/**
 * Replace the page's mounted conflict notice with one over a seeded store.
 * `storeKind` selects a store whose failure mode cannot be serialised across
 * the evaluate boundary, so it is constructed inside the page.
 */
async function seed(page, records, opts = {}) {
  await page.evaluate(({ recs, storeKind, modelOpts }) => {
    const N = window.HQConflictNotice;
    if (N.instance) N.instance.destroy();
    document.getElementById('conflict-notice').innerHTML = '';
    document.getElementById('conflict-sheet').innerHTML = '';

    let store;
    if (storeKind === 'unreadable') {
      // iOS/Safari eviction under storage pressure, or private browsing. W3
      // named this the largest untested unknown for a phone-first PWA.
      store = {
        all: () => Promise.reject(new Error('IndexedDB evicted')),
        upsert: () => Promise.resolve(),
        patch: () => Promise.resolve(),
        remove: () => Promise.resolve(),
      };
    } else if (storeKind === 'slow') {
      store = {
        all: () => new Promise(() => {}),   // never resolves — cold IndexedDB
        upsert: () => Promise.resolve(),
        patch: () => Promise.resolve(),
        remove: () => Promise.resolve(),
      };
    } else {
      store = N.createMemoryStore(recs);
    }

    N.instance = N.mount({
      store,
      modelOpts,
      // Deterministic: the restore either lands or does not, on demand.
      applyRestore: (row) => {
        if (window.__restoreFails) {
          const e = new Error('nope');
          e.name = window.__restoreFails === 'conflict' ? 'ConflictError' : 'OfflineError';
          return Promise.reject(e);
        }
        return Promise.resolve({ ok: true, row: row.id });
      },
      copyValue: () => Promise.resolve(),
      openChecklist: () => {},
    });
  }, { recs: records, storeKind: opts.storeKind || 'memory', modelOpts: { now: opts.now || NOW, snapshots: opts.snapshots || {} } });
}

async function openSheet(page) {
  await page.evaluate(() => window.HQConflictNotice.instance.open());
  await page.waitForSelector('#conflict-sheet [data-testid="conflict-sheet"]');
}

/**
 * 🔴 THE POPULATION FLOOR. A screenshot spec that renders an empty sheet passes
 * vacuously — this is the check that makes it red instead. Called before EVERY
 * shutter release.
 */
async function assertPopulation(page, floors) {
  const seen = await page.evaluate(() => ({
    banners: document.querySelectorAll('#conflict-notice .cn-banner').length,
    bannerLines: document.querySelectorAll('#conflict-notice .cn-banner-hd, #conflict-notice .cn-banner-open, #conflict-notice .cn-banner-unid, #conflict-notice .cn-banner-sub').length,
    groups: document.querySelectorAll('#conflict-sheet .cg').length,
    rows: document.querySelectorAll('#conflict-sheet .cf').length,
    confirmRows: document.querySelectorAll('#conflict-sheet .cfm-row').length,
    skeletons: document.querySelectorAll('#conflict-sheet .sk').length,
    emptyCards: document.querySelectorAll('#conflict-sheet .sc-empty').length,
    errCards: document.querySelectorAll('#conflict-sheet .sc-err').length,
  }));
  for (const [k, want] of Object.entries(floors)) {
    expect(seen[k], `population floor "${k}": the state rendered ${seen[k]}, contract says ${want}`)
      .toBe(want);
  }
  return seen;
}

/**
 * Screenshot a state in both colour schemes.
 *
 * TWO FRAMES, because the real layout is not the mockup's layout: the sheet is
 * `position:fixed` over My Checklists, so a single full-page capture shows the
 * sheet and HIDES the banner behind it. The mockup draws them stacked on one
 * plate; the app cannot. So `<name>-banner-*.png` is the banner as a crew member
 * meets it (sheet closed, checklist list behind), and `<name>-*.png` is the
 * sheet. Reading only one of the two would miss half of A-1.
 *
 * The viewport is grown to the sheet's own scroll height before the shutter —
 * the sheet scrolls internally, and a 900px window would silently crop the
 * batch control off the bottom of the busiest plates, which is exactly the
 * element A-2 is about.
 */
async function shot(page, name, opts = {}) {
  if (opts.banner) {
    await page.evaluate(() => window.HQConflictNotice.instance.close());
    for (const scheme of ['light', 'dark']) {
      await page.emulateMedia({ colorScheme: scheme });
      await page.screenshot({ path: path.join(SHOT_DIR, `${name}-banner-${scheme}.png`) });
    }
    await openSheet(page);
  }
  const h = await page.evaluate(() => {
    const el = document.querySelector('#conflict-sheet .sc-sheet') || document.body;
    return Math.min(4000, Math.max(600, el.scrollHeight + 40));
  });
  await page.setViewportSize({ width: 480, height: h });
  for (const scheme of ['light', 'dark']) {
    await page.emulateMedia({ colorScheme: scheme });
    await page.screenshot({ path: path.join(SHOT_DIR, `${name}-${scheme}.png`) });
  }
  await page.emulateMedia({ colorScheme: 'light' });
  await page.setViewportSize({ width: 480, height: 900 });
}

/** Every interactive control in the design must be a >=44px target. */
async function assertTapTargets(page, floor) {
  const r = await page.evaluate(() => {
    const sel = '#conflict-notice .cn-banner-go, #conflict-sheet .cf-btn, #conflict-sheet .cg-all, #conflict-sheet .cf-done-undo, #conflict-sheet .sc-close, #conflict-sheet .sc-err button, #conflict-sheet .cfm-go, #conflict-sheet .cfm-cancel';
    const bad = [];
    const all = document.querySelectorAll(sel);
    for (const el of all) {
      const b = el.getBoundingClientRect();
      if (b.width < 44 || b.height < 44) {
        bad.push(`${el.className} "${(el.textContent || '').trim().slice(0, 20)}" ${Math.round(b.width)}x${Math.round(b.height)}`);
      }
    }
    return { n: all.length, bad };
  });
  // Floor first: walking the controls that EXIST means deleting one is
  // invisible. That is the exact escape the mockup's own hardening round closed.
  expect(r.n, 'tap-target population below floor — a control was deleted or renamed').toBeGreaterThanOrEqual(floor);
  expect(r.bad, 'controls under 44px').toEqual([]);
}

/** No banner line may truncate or ellipsise at 480px. A-1's PARK trigger. */
async function assertNoBannerTruncation(page, floor) {
  const r = await page.evaluate(() => {
    const sel = '#conflict-notice .cn-banner-hd, #conflict-notice .cn-banner-open, #conflict-notice .cn-banner-unid, #conflict-notice .cn-banner-sub';
    const els = [...document.querySelectorAll(sel)];
    const bad = [];
    for (const el of els) {
      const cs = getComputedStyle(el);
      if (el.scrollWidth > el.clientWidth + 1) bad.push(`clipped: "${el.textContent.trim().slice(0, 40)}"`);
      if (cs.textOverflow === 'ellipsis' && cs.overflow !== 'visible') bad.push(`ellipsised: "${el.textContent.trim().slice(0, 40)}"`);
    }
    return {
      n: els.length,
      bad,
      pageOverflow: document.documentElement.scrollWidth > document.documentElement.clientWidth,
    };
  });
  expect(r.n, 'banner-line population below floor — a line was deleted').toBeGreaterThanOrEqual(floor);
  expect(r.bad).toEqual([]);
  expect(r.pageOverflow, 'the page scrolls sideways at 480px').toBe(false);
}

// ---------------------------------------------------------------------------

test.describe('conflict notice — State Enumeration Table', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 480, height: 900 });
    await login(page);
    await page.goto('/workflows.html');
    await page.waitForFunction(() => !!window.HQConflictNotice);
  });

  // ── base: empty ─────────────────────────────────────────────────────────
  test('empty — no banner; copy scoped to the record and to the retention window', async ({ page }) => {
    await seed(page, []);
    await openSheet(page);
    await assertPopulation(page, { banners: 0, groups: 0, rows: 0, emptyCards: 1 });

    // A flat "Nothing was overwritten" is BANNED: a non-leader tab, an
    // unsubscribed replication and an evicted store all produce this identical
    // screen, so it is a claim the app is not in a position to make — on the
    // screen it shows most often.
    const copy = await page.textContent('#conflict-sheet .sc-empty');
    expect(copy).not.toMatch(/nothing was overwritten/i);
    expect(copy).toMatch(/This is what HQ caught and kept/);

    // Decision 96 — the number comes from the ONE named constant.
    const days = await page.textContent('#conflict-sheet [data-testid="retention-days"]');
    const { CONFLICT_RECORD_RETENTION_DAYS } = await import(
      require('url').pathToFileURL(path.join(__dirname, '..', 'sync-schema', 'collections.js')).href
    );
    expect(Number(days)).toBe(CONFLICT_RECORD_RETENTION_DAYS);

    await assertTapTargets(page, 1);
    await shot(page, 'empty');
  });

  // ── base: loading ───────────────────────────────────────────────────────
  test('loading — two skeletons, no spinner, NO COUNT in the header', async ({ page }) => {
    await seed(page, [], { storeKind: 'slow' });
    await openSheet(page);
    await page.waitForSelector('#conflict-sheet .sk');
    await assertPopulation(page, { banners: 0, groups: 0, rows: 0, skeletons: 2 });

    // The header must not claim a number it does not have yet.
    const head = (await page.textContent('#conflict-sheet .sc-hd')).trim();
    expect(head).toBe('Overwritten answersDone');
    expect(head).not.toMatch(/\d/);
    await shot(page, 'loading');
  });

  // ── base: success ───────────────────────────────────────────────────────
  test('success — banner carries BOTH A-1 figures; a restored row names the value back and keeps Undo', async ({ page }) => {
    await seed(page, [COOLER, { ...SANITIZER, status: 'restored' }]);
    await openSheet(page);
    const seen = await assertPopulation(page, { banners: 1, groups: 1, rows: 2 });
    expect(seen.bannerLines).toBeGreaterThanOrEqual(3);

    expect(await page.textContent('.cn-banner-hd')).toBe('2 answers were overwritten');
    // 🛑 A-1. Two figures, not one. A run that prints only the headline
    // reinstates the defect decision 82 was filed against.
    expect((await page.textContent('.cn-banner-open')).replace(/\s+/g, ' ').trim())
      .toBe('1 still to review · 1 handled');
    expect(await page.textContent('.cg-count')).toBe('2 answers');

    // A-2.1 — the action names what it REPLACES, not only what it restores.
    expect(await page.textContent('.cf[data-state="open"] .cf-btn-pri .cf-btn-s')).toMatch(/replaces/);
    // The restored row names the value that came back and keeps a bordered Undo.
    const done = await page.textContent('.cf[data-state="restored"] .cf-done');
    expect(done).toMatch(/200/);
    expect(done).toMatch(/Undo/);

    await assertNoBannerTruncation(page, 3);
    await assertTapTargets(page, 4);
    await shot(page, 'success', { banner: true });
  });

  // ── base: error ─────────────────────────────────────────────────────────
  test('error — a FAILED restore is still to review, and the count does not move', async ({ page }) => {
    await seed(page, [
      { ...COOLER, status: 'failed', failure: 'offline' },
      { ...HANDSINK, status: 'restoring' },
    ]);
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 1, rows: 2 });

    // 🛑 A-1 rule 3's hard case. If this read 1 or 0, the definition would be
    // contradicted by its own screen.
    expect(await page.textContent('.cn-banner-open')).toBe('2 still to review');
    // headline == chip == rows drawn, on a plate whose rows have already
    // failed / started — which is what a static render can prove about "the
    // count does not decrement".
    expect(await page.textContent('.cn-banner-hd')).toBe('2 answers were overwritten');
    expect(await page.textContent('.cg-count')).toBe('2 answers');

    // Both values stay on screen above the red block.
    const failed = page.locator('.cf[data-state="failed"]');
    await expect(failed.locator('.cf-v.mine .cf-v-val')).toHaveText(/38/);
    await expect(failed.locator('.cf-v').nth(1).locator('.cf-v-val')).toHaveText(/41/);
    const err = await failed.locator('.cf-err').textContent();
    expect(err).toMatch(/on this list/);
    expect(err).toMatch(/Retry/);
    // It must NOT promise an automatic retry — nothing here commits to one.
    expect(err).not.toMatch(/keep trying|automatically/i);
    // Retry is the same destructive write as Restore, so it names the loss too.
    expect(await failed.locator('.cf-btn-pri .cf-btn-s').textContent()).toMatch(/replaces/);

    // In-flight: both buttons disabled, label says what is happening.
    const flight = page.locator('.cf[data-state="restoring"]');
    await expect(flight.locator('.cf-btn-pri')).toHaveAttribute('aria-disabled', 'true');
    await expect(flight.locator('.cf-btn-sec')).toHaveAttribute('aria-disabled', 'true');
    expect(await flight.locator('.cf-btn-pri').textContent()).toMatch(/Restoring/);

    await assertNoBannerTruncation(page, 3);
    await shot(page, 'error', { banner: true });
  });

  // ── edge: row already handled ───────────────────────────────────────────
  test('edge/outcomes — Keep theirs and Undo, drawn: headline frozen, second figure moving', async ({ page }) => {
    await seed(page, [
      { ...COOLER, status: 'kept' },
      { ...SANITIZER, status: 'open', undone: true },
    ]);
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 1, rows: 2 });

    // One plate, two numbers, only one of them moving — A-1 in a single render.
    expect(await page.textContent('.cn-banner-hd')).toBe('2 answers were overwritten');
    expect((await page.textContent('.cn-banner-open')).replace(/\s+/g, ' ').trim())
      .toBe('1 still to review · 1 handled');
    expect(await page.textContent('.cg-count')).toBe('2 answers');

    const kept = await page.textContent('.cf[data-state="kept"] .cf-kept');
    expect(kept).toMatch(/Kept theirs/);
    expect(kept).toMatch(/41/);      // names the value now STANDING
    expect(kept).toMatch(/Undo/);
    expect(await page.textContent('.cf-undone')).toMatch(/Undone/);

    await assertTapTargets(page, 4);
    await shot(page, 'outcomes', { banner: true });
  });

  // ── edge: no discarded value available ──────────────────────────────────
  test('edge/novalue — no Restore, "Not recoverable", and TWO actions (not one)', async ({ page }) => {
    await seed(page, [FRYER, UNIDENT('u1', '2026-07-27T21:05:00-04:00')]);
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 1, rows: 2 });

    expect(await page.textContent('.cn-banner-hd')).toBe('1 answer was overwritten');
    expect(await page.textContent('.cn-banner-open')).toBe('1 still to review');
    expect(await page.textContent('.cn-banner-unid')).toBe("+ 1 change we couldn't identify");
    expect(await page.textContent('.cg-count')).toBe('1 answer +1');

    // PER ROW, not per plate: the ordinary row above it DOES have Restore.
    const unrec = page.locator('.cf.unrec');
    await expect(unrec).toHaveCount(1);
    await expect(unrec.locator('.cf-v.none .cf-v-val')).toHaveText('Not recoverable');
    expect(await unrec.locator('.cf-btn').count()).toBe(2);
    expect(await unrec.textContent()).toMatch(/Open checklist/);
    expect(await unrec.textContent()).toMatch(/Dismiss/);
    expect(await unrec.textContent()).not.toMatch(/Restore/);
    expect(await page.locator('.cf:not(.unrec) .cf-btn-pri').textContent()).toMatch(/Restore mine/);

    await assertNoBannerTruncation(page, 3);
    await shot(page, 'edge-novalue', { banner: true });
  });

  // ── edge: several conflicts at once ─────────────────────────────────────
  test('edge/many — ONE banner, per-group chips, collapse sheet-wide, attribution parity', async ({ page }) => {
    await seed(page, [
      COOLER, SANITIZER, HANDSINK,
      { ...FRYER, id: 'e', doc_id: DOC_B, submission_id: DOC_B, checklist_name: 'Closing — Truck A', overwritten_at: '2026-07-27T21:02:00-04:00' },
      {
        ...BASE, id: 'f', doc_id: DOC_B, submission_id: DOC_B, checklist_name: 'Closing — Truck A',
        field_id: 'fld-grease', field_label: 'Grease trap note', field_type: 'text',
        discarded_value: 'Emptied, smelled off', current_value: 'Emptied',
        overwritten_at: '2026-07-27T21:04:00-04:00',
      },
    ]);
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 2, rows: 5 });

    expect(await page.textContent('.cn-banner-hd')).toBe('5 answers were overwritten');
    expect(await page.textContent('.cn-banner-open')).toBe('5 still to review');
    expect(await page.locator('.cg-count').allTextContents()).toEqual(['3 answers', '2 answers']);

    // Collapse applies to the WHOLE sheet — a 2-row group beside a 3-row one
    // must not render in a different style; a mixed sheet reads as a bug.
    await expect(page.locator('#conflict-sheet [data-collapsed="true"]')).toHaveCount(1);
    expect(await page.locator('.cf .cf-btn').count()).toBe(0);
    expect(await page.locator('.cg-all').count()).toBe(2);

    // 🛑 A-2.3 — attribution PARITY. r1's counter-example was a bare "Dana M."
    // on the collapsed view, so the riskiest action carried the least
    // information. Name AND time on every one of the five rows.
    const whos = await page.locator('.cf-v-who').allTextContents();
    expect(whos).toHaveLength(5);
    for (const w of whos) expect(w).toMatch(/Dana M\., \d{1,2}:\d{2}\s?(AM|PM)/);

    // A-2: the batch names what it replaces and says it asks first.
    for (const s of await page.locator('.cg-all-s').allTextContents()) {
      expect(s).toMatch(/replaces/);
      expect(s).toMatch(/asks first/);
    }

    await assertTapTargets(page, 4);
    await shot(page, 'edge-many', { banner: true });
  });

  // ── edge: batch override confirm ────────────────────────────────────────
  test('edge/confirm — A-2: the tap does NOT write; the confirm lists every value it replaces', async ({ page }) => {
    await seed(page, [COOLER, SANITIZER, HANDSINK]);
    await openSheet(page);
    await page.click('.cg-all');
    await page.waitForSelector('[data-testid="conflict-confirm"]');
    await assertPopulation(page, { confirmRows: 3 });

    // The title NAMES THE LOSS and counts it — not "Restore mine?".
    expect(await page.textContent('.cfm-hd')).toBe("Replace 3 of Dana M.'s answers?");
    // The three values are LISTED, never summarised as "3 answers" — a number
    // is the thing a crew member can agree to without reading.
    const qs = await page.locator('.cfm-q').allTextContents();
    expect(qs).toEqual(['Walk-in cooler temp', 'Sanitizer concentration', 'Hand sink stocked']);
    for (const row of await page.locator('.cfm-row').all()) {
      const gone = row.locator('.cf-v.gone');
      await expect(gone).toHaveCount(1);
      // struck through, in the destructive colour, WITH who and when
      const deco = await gone.locator('.cf-v-val').evaluate((el) => getComputedStyle(el).textDecorationLine);
      expect(deco).toContain('line-through');
      expect(await gone.locator('.cf-v-who').textContent()).toMatch(/Dana M\., \d{1,2}:\d{2}\s?(AM|PM)/);
      await expect(row.locator('.cf-v.mine')).toHaveCount(1);
    }
    // Cancel at equal weight, BEFORE the destructive control.
    const order = await page.locator('.cfm-acts > span').evaluateAll((els) => els.map((e) => e.className.split(' ')[0]));
    expect(order).toEqual(['cfm-cancel', 'cfm-go']);
    expect(await page.textContent('.cfm-go')).toMatch(/^Replace 3 answers/);

    // 🛑 THE WRITE DID NOT GO THROUGH ON THE TAP.
    expect(await page.locator('.cf[data-state="restored"]').count()).toBe(0);

    await assertTapTargets(page, 2);
    await shot(page, 'edge-confirm');

    // ...and Cancel leaves everything exactly as it was.
    await page.click('.cfm-cancel');
    await expect(page.locator('[data-testid="conflict-confirm"]')).toHaveCount(0);
    expect(await page.locator('.cf[data-state="restored"]').count()).toBe(0);
  });

  // ── edge: partly handled AND unidentifiable together (A-1's worst case) ──
  test('edge/a1-banner — FOUR banner lines coexist at 480px, and collapse takes no row exit', async ({ page }) => {
    await seed(page, [
      COOLER, HANDSINK,
      { ...SANITIZER, status: 'restored' },
      { ...FRYER, status: 'kept' },
      UNIDENT('u1', '2026-07-27T18:30:00-04:00'),
      UNIDENT('u2', '2026-07-27T19:00:00-04:00'),
    ]);
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 1, rows: 6, bannerLines: 4 });

    expect(await page.textContent('.cn-banner-hd')).toBe('4 answers were overwritten');
    expect((await page.textContent('.cn-banner-open')).replace(/\s+/g, ' ').trim())
      .toBe('2 still to review · 2 handled');
    expect(await page.textContent('.cn-banner-unid')).toBe("+ 2 changes we couldn't identify");
    expect(await page.textContent('.cg-count')).toBe('4 answers +2');

    // Counting rule 8, BOTH halves, under collapse:
    await expect(page.locator('#conflict-sheet [data-collapsed="true"]')).toHaveCount(1);
    // (a) handled rows keep their outcome strip AND their Undo — after a batch
    //     restore every row is green on a collapsed sheet, and if collapse ate
    //     Undo a batched mis-tap would be irreversible.
    expect(await page.locator('.cf-done-undo').count()).toBe(2);
    // (b) a row with nothing to restore keeps Open checklist + Dismiss —
    //     collapse hides the Restore/Keep PAIR and these rows have none, so
    //     there is nothing for it to take. Dismiss is their only exit.
    const unrec = page.locator('.cf.unrec');
    await expect(unrec).toHaveCount(2);
    for (const r of await unrec.all()) {
      expect(await r.locator('.cf-btn').count()).toBe(2);
      expect(await r.textContent()).toMatch(/Dismiss/);
    }
    // Counting rule 7: the batch is the still-to-review figure, not the base.
    expect(await page.textContent('.cg-all > span:first-child')).toBe('Restore all 2 of mine');

    await assertNoBannerTruncation(page, 4);
    await assertTapTargets(page, 6);
    await shot(page, 'a1-banner', { banner: true });
  });

  // ── edge: long value / long question text ───────────────────────────────
  test('edge/longvalue — wraps inside the card, never truncated, no sideways scroll', async ({ page }) => {
    const NOTE = 'Grease trap emptied at 9:40 but it smelled off — I\'d get someone out before the weekend. Also the back hood filter is loose again, third time this month, and the fryer took a while to come down to temp so it may need a look.';
    const TOKEN = 'RESTOCK-2026-07-27T18:44:02Z-TRUCKA-OPENING-9f31c4b2e7d5a1c8f0e3b6d94a2c7e15';
    await seed(page, [
      {
        ...BASE, id: 'n', doc_id: DOC_B, submission_id: DOC_B,
        checklist_name: 'Closing — Truck A (evening shift, Riverside lot)',
        field_id: 'fld-note', field_type: 'text',
        field_label: 'Anything the opening crew should know about the fryer, hood filters or the grease trap before tomorrow?',
        discarded_value: NOTE, current_value: 'Emptied',
        overwritten_at: '2026-07-27T21:02:00-04:00',
      },
      {
        ...BASE, id: 'r', doc_id: DOC_B, submission_id: DOC_B,
        checklist_name: 'Closing — Truck A (evening shift, Riverside lot)',
        field_id: 'fld-ref', field_type: 'text', field_label: 'Vendor reference on the delivery slip',
        discarded_value: TOKEN, current_value: null,
        overwritten_at: '2026-07-27T21:04:00-04:00',
      },
    ]);
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 1, rows: 2 });

    // NEVER truncated: the value is the thing being recovered, so an ellipsis
    // would hide the payload the crew member came for.
    const shown = await page.locator('.cf-v.mine .cf-v-val').allTextContents();
    expect(shown[0]).toContain('hood filter is loose again');
    expect(shown[1]).toContain(TOKEN);
    const clipped = await page.locator('.cf-v-val').evaluateAll((els) => els
      .filter((el) => el.scrollWidth > el.clientWidth + 1 || getComputedStyle(el).textOverflow === 'ellipsis')
      .map((el) => el.textContent.slice(0, 30)));
    expect(clipped).toEqual([]);
    // The 88px label column keeps its width.
    const labW = await page.locator('.cf-v-lab').first().evaluate((el) => el.getBoundingClientRect().width);
    expect(Math.round(labW)).toBe(88);
    // The page does not scroll sideways at 480px.
    const ov = await page.evaluate(() => ({
      s: document.documentElement.scrollWidth, c: document.documentElement.clientWidth,
    }));
    expect(ov.s).toBe(ov.c);

    await shot(page, 'edge-longvalue');
  });

  // ── edge: conflict on a field since removed (A-3) ───────────────────────
  test('edge/removed — A-3: the question keeps its LABEL, struck through and read-only', async ({ page }) => {
    await seed(page, [COOLER, REMOVED], { snapshots: {} });
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 1, rows: 2 });

    const row = page.locator('.cf.unrec');
    await expect(row).toHaveCount(1);
    const title = row.locator('.cf-q-struck');
    // 🛑 A-3 (decision 95). Operator's words: "show the deleted question crossed
    // out and read only so that the user isnt confused." NOT the raw field id,
    // NOT monospace — the same type as any other question title.
    await expect(title).toHaveText('Prep sink temperature');
    expect(await title.evaluate((el) => getComputedStyle(el).textDecorationLine)).toContain('line-through');
    expect(await title.evaluate((el) => getComputedStyle(el).fontFamily)).not.toMatch(/mono/i);
    expect(await row.locator('.cf-q-gone').count()).toBe(0);
    // Read-only: no Restore anywhere on the row; Copy value is the recovery.
    expect(await row.textContent()).not.toMatch(/Restore/);
    expect(await row.textContent()).toMatch(/Copy value/);
    expect(await row.textContent()).toMatch(/Dismiss/);
    // The value is still shown in full — it is the thing being recovered.
    await expect(row.locator('.cf-v.mine .cf-v-val')).toHaveText(/72/);

    // Counting follows READING A (decision 95): it is in the chip base, and the
    // resulting `Restore all 1` under a `2 answers` chip is the accepted
    // consequence A-3 makes legible on screen rather than arithmetic.
    expect(await page.textContent('.cg-count')).toBe('2 answers');
    expect(await page.textContent('.cn-banner-hd')).toBe('2 answers were overwritten');
    expect(await page.locator('.cn-banner-unid').count()).toBe(0);

    await shot(page, 'edge-removed', { banner: true });
  });

  // ── edge: A-3's fallback, i.e. a MALFORMED template_snapshot ────────────
  test('edge/removed-fallback — a snapshot with no label falls back to the raw field id, and does not throw', async ({ page }) => {
    const errors = [];
    page.on('pageerror', (e) => errors.push(String(e)));
    // 🛑 B1's R-C, promoted to a dependency by A-3: nothing rejects a malformed
    // `template_snapshot`. Handing the renderer junk must produce the r2
    // fallback rendering, not an exception and not a blank row.
    await seed(page, [COOLER, REMOVED_NO_LABEL], {
      snapshots: { [DOC_A]: { sections: 'this is not a template' } },
    });
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 1, rows: 2 });

    const row = page.locator('.cf.unrec');
    await expect(row.locator('.cf-q-gone')).toHaveText('fld_prep_sink_temp');
    // Drawn EXACTLY as revision 2 drew it: muted monospace, visibly not a title.
    expect(await row.locator('.cf-q-gone').evaluate((el) => getComputedStyle(el).fontFamily)).toMatch(/mono/i);
    expect(await row.locator('.cf-q-struck').count()).toBe(0);
    expect(await row.textContent()).toMatch(/Copy value/);
    expect(errors, 'a malformed template_snapshot threw into the page').toEqual([]);

    await shot(page, 'edge-removed-fallback', { banner: true });
  });

  // ── edge: local conflict log unreadable ─────────────────────────────────
  test('edge/storage — both halves, bad one first, and half (a) is not muted-only', async ({ page }) => {
    await seed(page, [], { storeKind: 'unreadable' });
    await openSheet(page);
    await assertPopulation(page, { banners: 0, groups: 0, rows: 0, errCards: 1 });

    const ps = await page.locator('#conflict-sheet .sc-err p').allTextContents();
    expect(ps).toHaveLength(2);
    // (a) FIRST: the record may be permanently gone and the answers cannot be
    //     put back. This is the one screen where something really is
    //     unrecoverable, so it is the one that must say so.
    expect(ps[0]).toMatch(/it's gone/);
    expect(ps[0]).toMatch(/can't be put back/);
    // (b) and the checklists themselves are fine.
    expect(ps[1]).toMatch(/checklists are not affected/);
    // Half (a) must not be muted-only.
    const strongColor = await page.locator('#conflict-sheet .sc-err p b').first()
      .evaluate((el) => getComputedStyle(el).color);
    const mutedColor = await page.locator('#conflict-sheet .sc-err p').first()
      .evaluate((el) => getComputedStyle(el).color);
    expect(strongColor).not.toBe(mutedColor);
    // No fabricated count anywhere.
    expect(await page.textContent('#conflict-sheet .sc-hd')).not.toMatch(/\d/);

    await assertTapTargets(page, 2);
    await shot(page, 'edge-storage');
  });

  // ── edge: past the cap (decision 97) ────────────────────────────────────
  test('edge/cap — 10 groups shown, "and N more", and the banner reports the TRUE total', async ({ page }) => {
    const recs = [];
    for (let i = 0; i < 13; i++) {
      const doc = `sub-cap${i}-0000-0000-0000-00000000000${i % 10}`;
      recs.push({
        ...COOLER, id: `cap${i}`, doc_id: doc, submission_id: doc,
        checklist_name: `Checklist ${i}`,
        overwritten_at: `2026-07-2${(i % 7) + 1}T18:12:00-04:00`,
      });
    }
    await seed(page, recs, { now: '2026-08-01T09:00:00-04:00' });
    await openSheet(page);
    await assertPopulation(page, { banners: 1, groups: 10, rows: 10 });

    // 🛑 Rows below the line are NOT dropped and the banner still reports the
    // TRUE total — a cap that also capped the count would under-report the loss
    // on the one night the feature matters.
    expect(await page.textContent('.cn-banner-hd')).toBe('13 answers were overwritten');
    expect(await page.textContent('[data-testid="cap-line"]')).toMatch(/and 3 more checklists/);
    // No date filter was added (decision 97 rejected one).
    expect(await page.locator('#conflict-sheet input[type="date"]').count()).toBe(0);

    await shot(page, 'edge-cap', { banner: true });
  });
});

// ---------------------------------------------------------------------------
// Interactions. The states above are seeded; these DRIVE the wiring, because a
// seeded render proves the screen and not the behaviour behind it.
// ---------------------------------------------------------------------------

test.describe('conflict notice — the recovery path actually runs', () => {
  test.beforeEach(async ({ page }) => {
    await page.setViewportSize({ width: 480, height: 900 });
    await login(page);
    await page.goto('/workflows.html');
    await page.waitForFunction(() => !!window.HQConflictNotice);
  });

  test('Restore -> green + Undo -> back to two values, with the headline never moving', async ({ page }) => {
    await seed(page, [COOLER, SANITIZER]);
    await openSheet(page);
    expect(await page.textContent('.cn-banner-hd')).toBe('2 answers were overwritten');
    expect(await page.textContent('.cn-banner-open')).toBe('2 still to review');

    await page.locator('.cf[data-row="a"] .cf-btn-pri').click();
    await expect(page.locator('.cf[data-row="a"][data-state="restored"]')).toHaveCount(1);
    // (b) STANDS — the row did not leave and the headline did not move.
    expect(await page.locator('.cf').count()).toBe(2);
    expect(await page.textContent('.cn-banner-hd')).toBe('2 answers were overwritten');
    expect((await page.textContent('.cn-banner-open')).replace(/\s+/g, ' ').trim())
      .toBe('1 still to review · 1 handled');

    await page.locator('.cf[data-row="a"] .cf-done-undo').click();
    await expect(page.locator('.cf[data-row="a"][data-state="open"]')).toHaveCount(1);
    expect(await page.textContent('.cf[data-row="a"] .cf-undone')).toMatch(/Undone/);
    expect(await page.textContent('.cn-banner-open')).toBe('2 still to review');
  });

  test('a restore that does not land leaves the row failed, both values on screen, and still to review', async ({ page }) => {
    await seed(page, [COOLER]);
    await page.evaluate(() => { window.__restoreFails = 'offline'; });
    await openSheet(page);
    await page.locator('.cf[data-row="a"] .cf-btn-pri').click();
    await expect(page.locator('.cf[data-row="a"][data-state="failed"]')).toHaveCount(1);
    expect(await page.textContent('.cn-banner-open')).toBe('1 still to review');
    await expect(page.locator('.cf[data-row="a"] .cf-v.mine .cf-v-val')).toHaveText(/38/);
    // Retry is drawn, and it names what it replaces.
    expect(await page.textContent('.cf[data-row="a"] .cf-btn-pri')).toMatch(/Retry/);
  });

  test('the batch confirm COMMITS only on Replace, and only the still-to-review rows', async ({ page }) => {
    await seed(page, [COOLER, SANITIZER, { ...HANDSINK, status: 'kept' }]);
    await openSheet(page);
    // Counting rule 7 — 3 in the chip, 2 in the batch.
    expect(await page.textContent('.cg-count')).toBe('3 answers');
    expect(await page.textContent('.cg-all > span:first-child')).toBe('Restore all 2 of mine');
    await page.click('.cg-all');
    await page.click('.cfm-go');
    await expect(page.locator('.cf[data-state="restored"]')).toHaveCount(2);
    // 🛑 The deliberately KEPT row was NOT re-written. A batch tap that reversed
    // the crew member's own decision is the opposite of an override that states
    // what it destroys.
    await expect(page.locator('.cf[data-row="c"][data-state="kept"]')).toHaveCount(1);
    // ...and every restored row still carries an Undo under collapse.
    expect(await page.locator('.cf-done-undo').count()).toBe(3);
  });

  test('Dismiss is the only way a row leaves the sheet', async ({ page }) => {
    await seed(page, [FRYER, UNIDENT('u1', '2026-07-27T21:05:00-04:00')]);
    await openSheet(page);
    expect(await page.locator('.cf').count()).toBe(2);
    await page.locator('.cf.unrec').getByText('Dismiss').click();
    await expect(page.locator('.cf')).toHaveCount(1);
    expect(await page.locator('.cn-banner-unid').count()).toBe(0);
    expect(await page.textContent('.cg-count')).toBe('1 answer');
  });
});
