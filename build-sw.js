const { generateSW } = require('workbox-build');
const fs = require('fs');
const { execFileSync } = require('child_process');

// Build artifacts that are deliberately git-ignored yet SHIP. version.json is
// written by writeVersionJson() below and regenerated inside the image by
// backend/Dockerfile (lines 33-44) from the authoritative Frontend constant --
// precisely BECAUSE sw.js precaches it. Everything else must be committed in HEAD
// -- see committedFiles() below; "tracked" is no longer the bar, decision 67.
const GENERATED_BUT_SHIPPED = new Set(['version.json']);

// The precache is built from the COMMITTED set -- what is in HEAD, not what is
// in the working tree and not what is merely staged in the index.
//
// A Workbox precache entry that 404s fails the ENTIRE service-worker install,
// so a single file in the manifest that prod cannot serve bricks updates on
// every phone -- and the symptom is "the PWA stopped updating" with no visible
// cause. The path is not hypothetical: `task sw` runs automatically as a
// dependency of BOTH `task test` and `task prod:deploy`, so a stray *.html in a
// dev machine's repo root gets baked into the committed sw.js and then 404s on
// prod, where git has never heard of it. (backlog-round.html, disposed as FORK 2
// in ledger T-22, was exactly such a file.) Ledger T-23 decision 58.
//
// HEAD, not the index -- ledger T-25 decision 67, which AMENDS decision 58's
// literal text ("the tracked set (`git ls-files`)") in service of its intent.
// `git ls-files` reads the INDEX, so `git add zz-probe.html` alone -- no commit
// anywhere -- was enough to put the file in the manifest. That closes a complete
// trigger path, because `task prod:deploy` (Taskfile.yml:174-210) does NOT run
// `task sw` on the box: it `git reset --hard origin/main` then
// `docker compose build`, so the COMMITTED sw.js is what ships, and a manifest
// built against the index names a URL the image was never built from.
//
// The result is (patterns ∩ working tree ∩ HEAD) ∪ GENERATED_BUT_SHIPPED. A file
// committed but deleted from the working tree cannot be globbed in the first
// place, so the working-tree intersection needs no separate check.
//
// `-r` recurses into trees (without it you get `icons`, not
// `icons/icon-96x96.png`); `-z` gives NUL-separated UNQUOTED paths -- plain
// `--name-only` C-quotes any path containing a space or a non-ASCII byte, which
// would silently drop a legitimately committed asset out of the precache.
function committedFiles() {
  const out = execFileSync('git', ['ls-tree', '-r', '--name-only', '-z', 'HEAD'], { encoding: 'utf8' });
  return new Set(out.split('\0').filter(Boolean));
}

function committedOnlyTransform(manifest) {
  const committed = committedFiles();
  const dropped = [];
  const kept = manifest.filter(entry => {
    if (committed.has(entry.url) || GENERATED_BUT_SHIPPED.has(entry.url)) return true;
    dropped.push(entry.url);
    return false;
  });
  // Loud, not silent: an asset someone MEANT to ship should show up here as
  // "git add it AND commit it", rather than as a dead service worker two deploys
  // on. Staged-but-uncommitted lands in this list too, and that is the point.
  for (const url of dropped) {
    console.warn(`  skipped (not in HEAD): ${url}`);
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

      // ═══ THE RxDB CLIENT LAYER — ADOPTED. Ledger T-23 decision 59. ═══════
      //
      // These three entries came IN together, on the card decision 59 named
      // (`sync-rxdb-replication-and-conflict-handler`, overnight-20260801),
      // because that is the card on which `workflows.html` finally carries
      //     <script type="module" src="sync-rxdb/bootstrap.js">
      // and bootstrap.js statically imports sync-rxdb/client.js, which imports
      // BOTH sync-schema/collections.js AND vendor/rxdb.bundle.js.
      //
      // Until then the bundle was deliberately EXCLUDED: 495 KiB, 34% of the
      // precache and 25.4% of its bytes, downloaded over LTE onto every crew
      // phone for an asset no page loaded. That reason has now expired — the
      // page loads it — and an offline-availability failure is finally
      // actionable on the card that owns the feature.
      //
      // 🛑 THE GLOB FORM IS LOAD-BEARING. 'vendor/**/*.bundle.js' and NOT
      // either of these two neighbouring traps, both learned the hard way by
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
      //
      // 🛑🛑 AND THE TRAP DECISION 59 CAME WITH: RE-ADDING A GLOB ALONE BREAKS
      // PRODUCTION. Before this card, `backend/Dockerfile` copied only
      // `*.html *.js manifest.json`, `icons` and `lib` into the image — NOTHING
      // under vendor/, sync-rxdb/ or sync-schema/. A precached URL that 404s
      // fails the ENTIRE service-worker install for every returning client, and
      // the symptom is "the PWA stopped updating" with no visible cause (the
      // exact bug `pwa-cache-and-build-hygiene` fixed). The matching Dockerfile
      // copies landed in the same commit as these three lines, and
      // `tests/sw-manifest.spec.js` now asserts — mechanically, for EVERY entry
      // — that each precached URL is covered by a Dockerfile copy into
      // cmd/server/public/. Add a glob here without adding the copy there and
      // the suite reds instead of the truck going dark.
      'vendor/**/*.bundle.js',
      'sync-rxdb/*.js',
      'sync-schema/collections.js',
    ],
    globIgnores: [
      'node_modules/**',
      'vendor/node_modules/**',
      'backend/**',
      'tests/**',
      '.planning/**',
      '.claude/**',
    ],
    // Decisions 58 + 67 — see committedOnlyTransform above. This runs AFTER the
    // glob and after content-hashing, and drops anything HEAD does not contain.
    manifestTransforms: [committedOnlyTransform],
    // Static assets: cache-first (same as before)
    // No need to configure — precacheAndRoute handles this automatically

    // API calls: network-first with offline JSON fallback, PARTITIONED BY
    // IDENTITY.
    //
    // ═══ THE api-cache PARTITION — ledger T-30 decision 112 ══════════════════
    //
    // This ONE route covers /\/api\// — every endpoint in all five tools. It is
    // NOT retired and must not be: RxDB replicates four collections, all of them
    // `workflow`, so retiring the route takes offline API reads away from
    // Inventory, Users, Onboarding and Purchasing, which RxDB has never covered.
    // Decision 105's per-open-checklist scope narrows RxDB further still. The
    // struck "retire it once RxDB replicates" premise (decision 57) was false
    // when written; decision 112 replaced it with what follows.
    //
    // THE DEFECT the two hooks below close: with no Vary, no cacheKeyWillBeUsed
    // and no matchOptions, the cache key was the bare URL and the session cookie
    // was not part of it. On a shared truck phone, whatever user A loaded was
    // served to user B the moment the network leg failed — which on a food truck
    // is routine, not exotic. Reproduced end-to-end, with the leaked payload
    // printed, in tests/sw-api-cache-partition.spec.js [B1-XT-01]: a team_member
    // with no Users grant was handed the full team roster at HTTP 200.
    //
    // 🛑 WHERE THE IDENTITY COMES FROM, AND WHY IT IS A Cache AND NOT ANYTHING
    // ELSE. A service worker cannot read `localStorage`. It cannot read the
    // session cookie either: it is HttpOnly (backend/internal/auth/handler.go:61)
    // and the `Cookie` header is attached by the network stack AFTER the fetch
    // event, so it is not on the request these hooks see. CacheStorage is the one
    // store the page and the worker can both reach without adding an IndexedDB
    // module to the precache. index.html writes the token; both ends must move in
    // the same commit if either moves. See
    // .night-crew/runs/2026-08-02-autonomous/merge-intent-b1-sync-cache-and-identity-hygiene.md §2.
    runtimeCaching: [
      {
        urlPattern: /\/api\//,
        handler: 'NetworkFirst',
        options: {
          networkTimeoutSeconds: 10,
          cacheName: 'api-cache',
          plugins: [
            {
              // The partition. `__hq_id` is a CACHE KEY discriminator only —
              // Workbox still fetches the ORIGINAL url over the network, so this
              // parameter never reaches the server. Two users on one phone get
              // two disjoint key spaces for the same URL.
              //
              // 🛑 Deleting this hook re-opens the cross-tenant read in full and
              // nothing else notices: build-sw.js still exits 0, the precache is
              // still 29 files, the SW still installs. Only [B1-XT-01] and
              // [B1-XT-02] fail.
              cacheKeyWillBeUsed: async ({ request }) => {
                let id = 'anon';
                try {
                  const c = await caches.open('hq-identity');
                  const r = await c.match('/__hq_identity');
                  if (r) {
                    const t = (await r.text()).trim();
                    if (t) id = t;
                  }
                } catch (e) { /* no identity readable — stays 'anon' */ }
                const u = new URL(request.url);
                u.searchParams.set('__hq_id', id);
                return new Request(u.href);
              },
              // No identity, no write. This closes the boot window between page
              // load and the /api/v1/me answer, during which the token does not
              // exist yet. Without it that window writes an `anon` partition
              // which every subsequent user of the phone shares — a smaller
              // version of the same bug. Returning null means "do not cache".
              cacheWillUpdate: async ({ response }) => {
                try {
                  const c = await caches.open('hq-identity');
                  const r = await c.match('/__hq_identity');
                  if (r && (await r.text()).trim()) return response;
                } catch (e) { /* fall through */ }
                return null;
              },
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
