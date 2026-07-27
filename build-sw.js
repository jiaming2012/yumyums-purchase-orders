const { generateSW } = require('workbox-build');
const fs = require('fs');
const { execFileSync } = require('child_process');

// Build artifacts that are deliberately git-ignored yet SHIP. version.json is
// written by writeVersionJson() below and regenerated inside the image by
// backend/Dockerfile (lines 33-44) from the authoritative Frontend constant --
// precisely BECAUSE sw.js precaches it. Everything else must be tracked.
const GENERATED_BUT_SHIPPED = new Set(['version.json']);

// The precache is built from the TRACKED set, not the working tree.
//
// A Workbox precache entry that 404s fails the ENTIRE service-worker install,
// so a single untracked file that matches globPatterns bricks updates on every
// phone -- and the symptom is "the PWA stopped updating" with no visible cause.
// The path is not hypothetical: `task sw` runs automatically as a dependency of
// BOTH `task test` and `task prod:deploy`, so a stray *.html in a dev machine's
// repo root gets baked into the committed sw.js and then 404s on prod, where
// git has never heard of it. (backlog-round.html, disposed as FORK 2 in ledger
// T-22, was exactly such a file.) Ledger T-23 decision 58.
//
// Filtering the generated manifest is equivalent to globbing `git ls-files`:
// the result is (patterns ∩ working tree ∩ tracked), and a tracked file that is
// missing from the working tree cannot be globbed in the first place.
function trackedFiles() {
  const out = execFileSync('git', ['ls-files', '-z'], { encoding: 'utf8' });
  return new Set(out.split('\0').filter(Boolean));
}

function trackedOnlyTransform(manifest) {
  const tracked = trackedFiles();
  const dropped = [];
  const kept = manifest.filter(entry => {
    if (tracked.has(entry.url) || GENERATED_BUT_SHIPPED.has(entry.url)) return true;
    dropped.push(entry.url);
    return false;
  });
  // Loud, not silent: an untracked asset someone MEANT to ship should show up
  // here as "git add it", rather than as a dead service worker two deploys on.
  for (const url of dropped) {
    console.warn(`  skipped (untracked): ${url}`);
  }
  return { manifest: kept };
}

// Write version.json so the frontend can read its own version without hitting the API.
// Semver only — dynamic fields (git_sha, built_at) live in /api/v1/health.
// package.json "version" mirrors the Frontend constant in backend/internal/version/version.go.
function writeVersionJson() {
  const pkg = JSON.parse(fs.readFileSync('package.json', 'utf8'));
  const payload = { frontend: pkg.version };
  fs.writeFileSync('version.json', JSON.stringify(payload) + '\n');
  return payload;
}

async function build() {
  const version = writeVersionJson();

  const { count, size } = await generateSW({
    swDest: 'sw.js',
    globDirectory: '.',
    globPatterns: [
      '*.html',
      'ptr.js',
      'sync.js',
      'manifest.json',
      'version.json',
      'icons/**/*.png',
      // NO 'vendor/**/*.bundle.js' HERE, DELIBERATELY. Ledger T-23 decision 59.
      //
      // The vendored RxDB bundle (vendor/build-vendor.sh output) is 495 KiB —
      // 34% of the precache, 25.4% of its total bytes — and NO page imports it
      // yet. Precaching it costs every crew phone that download over LTE for an
      // asset nothing loads. It stays out until a page actually imports it;
      // `sync-rxdb-schema-and-replication` re-adds the entry on adoption
      // (roadmap rider 5 on that card), which is also the card where an
      // offline-availability failure would be actionable.
      //
      // When it comes back, it comes back as 'vendor/**/*.bundle.js' and NOT as
      // either of these two neighbouring traps — both learned the hard way by
      // sync-rxdb-browser-delivery-spike, which added the original entry:
      //   * NOT a bare 'vendor/**'. That sweeps the generator's own inputs
      //     (package-lock.json, src/*.mjs, build-vendor.sh) onto the phone, and
      //     — absent the 'vendor/node_modules/**' globIgnore below — 8,919
      //     files / 67 MB of node_modules, because globIgnores' 'node_modules/**'
      //     is a TOP-LEVEL pattern that does not match 'vendor/node_modules/**'.
      //   * NOT 'vendor/*.bundle.js' (single level). That would SILENTLY OMIT a
      //     future vendor/<sub>/foo.bundle.js — and a bundle silently absent
      //     from the precache looks fine in development and fails offline on the
      //     truck, the exact silent-drop class build-vendor.sh's 5 MiB guard
      //     exists to prevent.
    ],
    globIgnores: [
      'node_modules/**',
      'vendor/node_modules/**',
      'backend/**',
      'tests/**',
      '.planning/**',
      '.claude/**',
    ],
    // Decision 58 — see trackedOnlyTransform above. This runs AFTER the glob and
    // after content-hashing, and drops anything git does not know about.
    manifestTransforms: [trackedOnlyTransform],
    // Static assets: cache-first (same as before)
    // No need to configure — precacheAndRoute handles this automatically

    // API calls: network-first with offline JSON fallback
    runtimeCaching: [
      {
        urlPattern: /\/api\//,
        handler: 'NetworkFirst',
        options: {
          networkTimeoutSeconds: 10,
          cacheName: 'api-cache',
          plugins: [
            {
              handlerDidError: async () => {
                return new Response('{"error":"offline"}', {
                  status: 503,
                  headers: { 'Content-Type': 'application/json' },
                });
              },
            },
          ],
        },
      },
    ],

    skipWaiting: true,
    clientsClaim: true,
    cleanupOutdatedCaches: true,
    maximumFileSizeToCacheInBytes: 5 * 1024 * 1024,
  });

  console.log(`SW built: ${count} files precached (${(size / 1024).toFixed(1)} KB)`);
  console.log(`Frontend version: ${version.frontend}`);
}

build().catch(err => {
  console.error('SW build failed:', err);
  process.exit(1);
});
