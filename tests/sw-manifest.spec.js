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
// precaches it. Anything NOT on this list must be tracked.
const GENERATED_BUT_SHIPPED = ['version.json'];

function urlsFrom(swSource) {
  const urls = [];
  const re = /url:"([^"]+)"/g;
  let m;
  while ((m = re.exec(swSource)) !== null) urls.push(m[1]);
  return urls;
}

function trackedFiles() {
  return new Set(
    execFileSync('git', ['ls-files', '-z'], { encoding: 'utf8' })
      .split('\0')
      .filter(Boolean),
  );
}

test('every URL in the committed precache manifest is a tracked file', async () => {
  const urls = urlsFrom(fs.readFileSync('sw.js', 'utf8'));
  expect(urls.length).toBeGreaterThan(0);
  const tracked = trackedFiles();
  const strays = urls.filter(u => !tracked.has(u) && !GENERATED_BUT_SHIPPED.includes(u));
  expect(strays).toEqual([]);
});

test('the vendored bundle is not in the precache', async () => {
  // Decision 59: 495 KiB / 34% of the precache over LTE on every crew phone for
  // an asset no page imports. sync-rxdb-schema-and-replication re-adds the
  // globPatterns entry when a page actually imports the bundle.
  const vendored = urlsFrom(fs.readFileSync('sw.js', 'utf8')).filter(u => u.startsWith('vendor/'));
  expect(vendored).toEqual([]);
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
