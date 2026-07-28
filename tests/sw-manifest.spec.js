// Precache-manifest guards — ledger T-23 decisions 58 and 59.
//
// A Workbox precache entry that 404s fails the ENTIRE service-worker install,
// so both failure modes guarded here are invisible in development and present
// on a crew phone as "the PWA stopped updating", with no error anyone can see.
const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');

// Build artifacts that are deliberately git-ignored yet SHIP: build-sw.js
// writes version.json locally, and backend/Dockerfile:33-44 regenerates it into
// the image from the authoritative Frontend constant precisely because sw.js
// precaches it. Anything NOT on this list must be committed in HEAD.
const GENERATED_BUT_SHIPPED = ['version.json'];

function urlsFrom(swSource) {
  const urls = [];
  const re = /url:"([^"]+)"/g;
  let m;
  while ((m = re.exec(swSource)) !== null) urls.push(m[1]);
  return urls;
}

// HEAD, not the index. This used to read `git ls-files`, matching build-sw.js as
// it was — and therefore agreeing with the bug ledger T-25 decision 67 names: a
// staged-but-uncommitted file is in `ls-files`, so an sw.js that precached one
// would have passed this test. `prod:deploy` ships the COMMITTED sw.js against a
// tree reset to origin/main, so HEAD is the set prod can actually serve.
function committedFiles() {
  return new Set(
    execFileSync('git', ['ls-tree', '-r', '--name-only', '-z', 'HEAD'], { encoding: 'utf8' })
      .split('\0')
      .filter(Boolean),
  );
}

test('every URL in the committed precache manifest is committed in HEAD', async () => {
  const urls = urlsFrom(fs.readFileSync('sw.js', 'utf8'));
  expect(urls.length).toBeGreaterThan(0);
  const committed = committedFiles();
  const strays = urls.filter(u => !committed.has(u) && !GENERATED_BUT_SHIPPED.includes(u));
  expect(strays).toEqual([]);
});

test('the vendored bundle is not in the precache', async () => {
  // Decision 59: 495 KiB / 34% of the precache over LTE on every crew phone for
  // an asset no page imports. sync-rxdb-schema-and-replication re-adds the
  // globPatterns entry when a page actually imports the bundle.
  //
  // REBUILDS rather than reading the committed sw.js. Asserting on the artifact
  // alone guards the wrong thing: re-adding 'vendor/**/*.bundle.js' to
  // globPatterns leaves a stale sw.js — and therefore this test — green until
  // somebody happens to run `task sw`, so a plain `npx playwright test` would
  // miss the config regression entirely. Found at G6 review. Both the committed
  // artifact AND a fresh build are checked below.
  const committed = urlsFrom(fs.readFileSync('sw.js', 'utf8')).filter(u => u.startsWith('vendor/'));
  expect(committed).toEqual([]);

  const swBackup = fs.readFileSync('sw.js');
  try {
    execFileSync('node', ['build-sw.js'], { encoding: 'utf8' });
    const rebuilt = urlsFrom(fs.readFileSync('sw.js', 'utf8')).filter(u => u.startsWith('vendor/'));
    expect(rebuilt).toEqual([]);
  } finally {
    fs.writeFileSync('sw.js', swBackup);
  }
});

test('a STAGED-but-uncommitted file never enters a freshly built manifest', async () => {
  // Ledger T-25 decision 67 — amends decision 58's literal text in service of its
  // intent. Decision 58 said "the tracked set (`git ls-files`)", but `git ls-files`
  // reads the git INDEX: `git add zz-probe.html` alone is enough to put the file in
  // the precache manifest, with no commit anywhere.
  //
  // That is a complete trigger path, not a hypothetical. `task sw` runs as a
  // dependency of BOTH `task test` and `task prod:deploy`; `task prod:deploy`
  // (Taskfile.yml:174-210) does NOT run `task sw` on the box -- it
  // `git reset --hard origin/main` then `docker compose build`, so the COMMITTED
  // sw.js is what ships. A manifest built against the index therefore ships a URL
  // that HEAD has never contained, it 404s in the image, and a 404 fails the ENTIRE
  // service-worker install for every returning client.
  //
  // The fix is `git ls-tree -r --name-only -z HEAD` -- read the commit, not the index.
  const probe = 'zz-sw-manifest-staged-probe.html';
  const swBackup = fs.readFileSync('sw.js');
  fs.writeFileSync(probe, '<!doctype html><title>staged probe</title>\n');
  try {
    // Stage it and ONLY stage it. This is the whole point: no commit is made, so
    // HEAD does not contain the file while the index does.
    execFileSync('git', ['add', '--', probe], { encoding: 'utf8' });
    expect(
      execFileSync('git', ['ls-files', '--', probe], { encoding: 'utf8' }).trim(),
      'probe must actually be staged, or this test proves nothing',
    ).toBe(probe);
    expect(
      execFileSync('git', ['ls-tree', '--name-only', 'HEAD', '--', probe], { encoding: 'utf8' }).trim(),
      'probe must NOT be in HEAD, or this test proves nothing',
    ).toBe('');

    // build-sw.js hard-codes swDest:'sw.js'; restore it in `finally` rather than
    // teaching the build script a test-only knob. Playwright runs workers:1.
    execFileSync('node', ['build-sw.js'], { encoding: 'utf8' });
    const rebuilt = urlsFrom(fs.readFileSync('sw.js', 'utf8'));
    expect(rebuilt).not.toContain(probe);
  } finally {
    try {
      execFileSync('git', ['rm', '--cached', '--force', '--quiet', '--', probe], { encoding: 'utf8' });
    } catch (e) {
      /* not staged (add threw) — nothing to unstage */
    }
    fs.writeFileSync('sw.js', swBackup);
    fs.rmSync(path.resolve(probe), { force: true });
  }
});

test('an untracked file in the repo root never enters a freshly built manifest', async () => {
  // Decision 58, reproduced directly. `task sw` runs automatically as a
  // dependency of BOTH `task test` and `task prod:deploy`, so an untracked
  // *.html sitting in the repo root of a dev machine gets baked into the
  // committed sw.js — and then 404s on prod, where git has never heard of it.
  // backlog-round.html (disposed as FORK 2 in ledger T-22) is exactly such a
  // file, and was still sitting in the main checkout when this card was written.
  const probe = 'zz-sw-manifest-probe.html';
  const swBackup = fs.readFileSync('sw.js');
  fs.writeFileSync(probe, '<!doctype html><title>untracked probe</title>\n');
  try {
    // build-sw.js hard-codes swDest:'sw.js'; restore it in `finally` rather than
    // teaching the build script a test-only knob. Playwright runs workers:1, so
    // nothing else is reading sw.js during this window.
    execFileSync('node', ['build-sw.js'], { encoding: 'utf8' });
    const rebuilt = urlsFrom(fs.readFileSync('sw.js', 'utf8'));
    expect(rebuilt).not.toContain(probe);
  } finally {
    fs.writeFileSync('sw.js', swBackup);
    fs.rmSync(path.resolve(probe), { force: true });
  }
});
