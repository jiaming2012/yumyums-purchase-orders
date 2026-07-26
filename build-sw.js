const { generateSW } = require('workbox-build');
const fs = require('fs');

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
      // Vendored, pre-built browser bundles (vendor/build-vendor.sh). Committed
      // output, precached content-hashed exactly like every other LOCAL asset —
      // which is the entire point: a food-truck PWA's offline data engine must
      // not live behind a CDN the truck cannot reach.
      //
      // Deliberately '*.bundle.js' and NOT a bare 'vendor/**'. globIgnores'
      // 'node_modules/**' is a TOP-LEVEL pattern and does not match
      // 'vendor/node_modules/**'; measured 2026-07-26, a bare 'vendor/**' sweeps
      // 8,919 files / 67 MB of vendor/node_modules into the precache manifest.
      // The narrow glob also keeps the generator's own inputs (package-lock.json,
      // src/*.mjs, build-vendor.sh) off the phone — they are build inputs, and
      // nothing on the truck should download them.
      'vendor/*.bundle.js',
    ],
    globIgnores: [
      'node_modules/**',
      'vendor/node_modules/**',
      'backend/**',
      'tests/**',
      '.planning/**',
      '.claude/**',
    ],
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
