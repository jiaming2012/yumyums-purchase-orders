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

// ───────────────────────────────────────────────────────────────────────────
// SUPERSEDED, DELIBERATELY — the test that used to live here asserted the
// vendored bundle was NOT in the precache (decision 59). Its own comment named
// the card that would invert it, and that card is
// `sync-rxdb-replication-and-conflict-handler` (overnight-20260801, C1):
// `workflows.html` now carries <script type="module" src="sync-rxdb/bootstrap.js">,
// which statically imports sync-rxdb/client.js, which imports both
// sync-schema/collections.js and vendor/rxdb.bundle.js. The bundle is a real
// runtime dependency of a production page, so it is precached.
//
// It is replaced by TWO successor guards, because decision 59's actual danger
// was never the glob — it was the glob WITHOUT the matching Dockerfile copy.
// ───────────────────────────────────────────────────────────────────────────

test('the RxDB client layer IS precached, from a fresh build and from the committed artifact', () => {
  // REBUILDS as well as reading the committed sw.js. Asserting on the artifact
  // alone guards the wrong thing in both directions: a globPatterns change
  // leaves a stale sw.js — and therefore this test — green until somebody
  // happens to run `task sw`, so a plain `npx playwright test` would miss the
  // config regression entirely. Found at G6 review on the card that wrote the
  // predecessor. Both are checked.
  const REQUIRED = ['vendor/rxdb.bundle.js', 'sync-rxdb/bootstrap.js',
    'sync-rxdb/client.js', 'sync-rxdb/conflict-handler.js',
    'sync-schema/collections.js'];

  const committed = new Set(urlsFrom(fs.readFileSync('sw.js', 'utf8')));
  for (const u of REQUIRED) expect([...committed], `committed sw.js is missing ${u}`).toContain(u);

  const swBackup = fs.readFileSync('sw.js');
  try {
    execFileSync('node', ['build-sw.js'], { encoding: 'utf8' });
    const rebuilt = new Set(urlsFrom(fs.readFileSync('sw.js', 'utf8')));
    for (const u of REQUIRED) expect([...rebuilt], `fresh build is missing ${u}`).toContain(u);
    // The glob form matters as much as its presence: 'vendor/**' would sweep
    // the generator's own inputs onto every crew phone.
    const vendorUrls = [...rebuilt].filter(u => u.startsWith('vendor/'));
    expect(vendorUrls).toEqual(['vendor/rxdb.bundle.js']);
  } finally {
    fs.writeFileSync('sw.js', swBackup);
  }
});

// ───────────────────────────────────────────────────────────────────────────
// 🛑 THE OBLIGATION-5 GUARD. Decision 59's trap, made mechanical.
//
// A precached URL that 404s fails the ENTIRE service-worker install for every
// returning client — the exact bug `pwa-cache-and-build-hygiene` fixed. In prod
// the frontend is served from the EMBEDDED FS (`//go:embed all:public`,
// backend/cmd/server/main.go), which contains exactly what backend/Dockerfile
// stages into `cmd/server/public/`. So "committed in HEAD" (the guard above) is
// necessary and NOT sufficient: `vendor/` was committed for a week while being
// copied into no image at all.
//
// This guard simulates the Dockerfile's two staging steps — the builder-stage
// COPY lines, then the `cp … cmd/server/public/` commands inside the RUN — and
// asserts every precache URL survives both.
//
// It FAILS LOUDLY on a Dockerfile form it cannot model, rather than passing on
// an empty parse (B-22/B-23/B-24: a guard's PASS is not evidence until its
// subject set is shown non-empty).
// ───────────────────────────────────────────────────────────────────────────

function expandSource(src) {
  // Only the forms the Dockerfile actually uses: a literal path, or a single
  // `*` glob in the basename. Anything else is refused rather than guessed.
  if (!src.includes('*')) return [src];
  const dir = path.dirname(src);
  const base = path.basename(src);
  if (base.indexOf('*') !== base.lastIndexOf('*') || dir.includes('*')) {
    throw new Error(`sw-manifest guard cannot model the COPY/cp source "${src}"`);
  }
  const re = new RegExp('^' + base.split('*').map(s => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('.*') + '$');
  const abs = path.resolve(dir === '.' ? '.' : dir);
  if (!fs.existsSync(abs)) return [];
  return fs.readdirSync(abs)
    .filter(f => re.test(f) && fs.statSync(path.join(abs, f)).isFile())
    .map(f => (dir === '.' ? f : `${dir}/${f}`));
}

function filesUnder(repoPath) {
  const abs = path.resolve(repoPath);
  if (!fs.existsSync(abs)) return [];
  if (fs.statSync(abs).isFile()) return [repoPath];
  const out = [];
  for (const e of fs.readdirSync(abs, { withFileTypes: true })) {
    if (e.name === 'node_modules') continue; // .dockerignore excludes it
    out.push(...filesUnder(path.posix.join(repoPath, e.name)));
  }
  return out;
}

/** Repo-relative paths present at /src in the builder stage, per the COPY lines. */
function builderStageFiles(dockerfile) {
  const present = new Map(); // /src-relative path -> repo-relative path
  const copyLines = dockerfile
    .split('\n')
    .map(l => l.trim())
    .filter(l => l.startsWith('COPY ') && !l.includes('--from='));
  expect(copyLines.length, 'no COPY lines parsed from backend/Dockerfile').toBeGreaterThan(0);

  for (const line of copyLines) {
    const args = line.slice(5).trim().split(/\s+/);
    const dest = args.pop();
    if (!dest.startsWith('./')) continue; // runtime stage (/app/...) — not the builder
    const destDir = dest.replace(/^\.\//, '').replace(/\/$/, '');
    for (const rawSrc of args) {
      for (const src of expandSource(rawSrc)) {
        const isDir = fs.existsSync(src) && fs.statSync(src).isDirectory();
        for (const f of filesUnder(src)) {
          // Docker: a DIRECTORY source copies its CONTENTS into dest; a FILE
          // source lands at dest/<basename>.
          const rel = isDir ? path.posix.relative(src, f) : path.posix.basename(f);
          const at = destDir ? path.posix.join(destDir, rel) : rel;
          present.set(at, f);
        }
      }
    }
  }
  return present;
}

/** URLs the embedded FS can serve, per the `cp … cmd/server/public/` commands. */
function imagePublicUrls(dockerfile, stage) {
  const served = new Set();
  // build-sw.js's GENERATED_BUT_SHIPPED counterpart: the Dockerfile writes it.
  if (/> *cmd\/server\/public\/version\.json/.test(dockerfile)) served.add('version.json');

  const cpCmds = dockerfile.match(/cp (?:-r )?[^;\n]*cmd\/server\/public\/[^\s;\\]*/g) || [];
  expect(cpCmds.length, 'no staging `cp` commands parsed from backend/Dockerfile').toBeGreaterThan(0);

  for (const cmd of cpCmds) {
    const args = cmd.replace(/^cp (?:-r )?/, '').trim().split(/\s+/);
    const dest = args.pop().replace(/^cmd\/server\/public\/?/, '');
    for (const rawSrc of args) {
      // Sources are written relative to backend/ as `../x`.
      const src = rawSrc.replace(/^\.\.\//, '');
      const matches = [...stage.keys()].filter(k => k === src || k.startsWith(src + '/'));
      if (src.includes('*')) {
        const re = new RegExp('^' + src.split('*').map(s => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('[^/]*') + '$');
        for (const k of stage.keys()) if (re.test(k)) served.add(dest ? path.posix.join(dest, k) : k);
        continue;
      }
      expect(matches.length, `backend/Dockerfile stages "${rawSrc}" but no COPY put it in the builder image`).toBeGreaterThan(0);
      const srcIsDir = matches.some(k => k.startsWith(src + '/'));
      for (const k of matches) {
        // `cp -r ../icons cmd/server/public/` creates public/icons/...
        const rel = srcIsDir ? path.posix.join(path.posix.basename(src), path.posix.relative(src, k)) : path.posix.basename(k);
        served.add(dest ? path.posix.join(dest, rel) : rel);
      }
    }
  }
  return served;
}

test('every precached URL is actually staged into the production image', () => {
  const dockerfile = fs.readFileSync(path.join('backend', 'Dockerfile'), 'utf8');
  const stage = builderStageFiles(dockerfile);
  const served = imagePublicUrls(dockerfile, stage);
  const urls = urlsFrom(fs.readFileSync('sw.js', 'utf8'));

  // Anti-vacuous: all three subject sets must be non-empty, and the served set
  // must be big enough that a parse failure cannot look like a pass.
  expect(stage.size, 'builder-stage file set is empty — the parse failed').toBeGreaterThan(10);
  expect(served.size, 'image public set is empty — the parse failed').toBeGreaterThan(10);
  expect(urls.length, 'precache manifest is empty').toBeGreaterThan(10);
  // The parse is only meaningful if it reproduces the assets we know ship.
  for (const known of ['index.html', 'workflows.html', 'ptr.js', 'sync.js',
    'manifest.json', 'version.json']) {
    expect([...served], `the Dockerfile parse lost a known-shipped asset: ${known}`).toContain(known);
  }

  const missing = urls.filter(u => !served.has(u));
  expect(
    missing,
    'these URLs are precached but backend/Dockerfile never copies them into '
    + 'cmd/server/public/. In prod they 404, and a single 404 fails the ENTIRE '
    + 'service-worker install for every returning client. Ledger decision 59.',
  ).toEqual([]);
});

test('the Obligation-5 guard actually fails when a precached asset is not staged', () => {
  // B-22/B-23/B-24 in person: the guard above prints PASS, so prove it can
  // print FAIL. Remove the vendor COPY from an in-memory Dockerfile and the
  // bundle must fall out of the served set.
  const dockerfile = fs.readFileSync(path.join('backend', 'Dockerfile'), 'utf8');
  const mutated = dockerfile.replace('COPY vendor/rxdb.bundle.js ./vendor/\n', '');
  expect(mutated, 'the vendor COPY line was not found — this guard is testing nothing')
    .not.toBe(dockerfile);

  const stage = builderStageFiles(mutated);
  expect([...stage.keys()]).not.toContain('vendor/rxdb.bundle.js');
  // ...and the staging `cp ../vendor` now has nothing to copy, which is itself
  // an assertion failure inside imagePublicUrls. Either way the guard reds.
  let reddened = false;
  try {
    const served = imagePublicUrls(mutated, stage);
    reddened = !served.has('vendor/rxdb.bundle.js');
  } catch (e) {
    reddened = true;
  }
  expect(reddened, 'removing the vendor COPY left the guard green').toBe(true);
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
