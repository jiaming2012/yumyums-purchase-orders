const { generateSW } = require('workbox-build');

async function build() {
  const { count, size } = await generateSW({
    swDest: 'sw.js',
    globDirectory: '.',
    globPatterns: [
      '*.html',
      'ptr.js',
      'manifest.json',
      'icons/**/*.png',
    ],
    globIgnores: [
      'node_modules/**',
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
}

build().catch(err => {
  console.error('SW build failed:', err);
  process.exit(1);
});
