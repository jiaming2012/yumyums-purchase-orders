const { generateSW } = require('workbox-build');
const fs = require('fs');
const path = require('path');
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

// What the LAST run of committedOnlyTransform dropped. Read by the
// import-reachability guard below purely to tell a reader WHY a target is
// missing, because the two causes have different fixes -- see checkImportReachability.
// 🛑 It goes stale silently if the two transforms are reordered or separated;
// keep them adjacent and in this order (merge-intent P1 §2.6).
let lastSkipped = new Set();

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
  lastSkipped = new Set(dropped);
  return { manifest: kept };
}

// ═══ IMPORT REACHABILITY — B-37 ═════════════════════════════════════════════
//
// THE INVARIANT: **nothing precached may import something not precached.**
//
// Loud was not enough. `committedOnlyTransform` above WARNS and the build exits
// 0 anyway, so on 2026-08-01 (`overnight-20260801`, merge 3) a regenerate in the
// middle of a merge shipped a 24-file manifest and nothing failed. Reproduced at
// triage at 22 files / 1481.9 KB against an expected 29 / 2111.1 KB, exit 0 every
// time. `workflows.html` IS precached and carries
//     <script type="module" src="sync-rxdb/bootstrap.js">
// so the shipped worker caches a page whose module entry point it deliberately
// omitted: the page loads from cache, the module 404s or is simply absent
// offline, and the tool is dead on a returning client with nothing on screen
// saying why. That is D-KR2's returning-client parity, directly.
//
// 🛑 THIS IS NOT "EVERYTHING MUST BE PRECACHED", AND MUST NOT BECOME THAT.
// Skipping a genuinely unreferenced file is the FEATURE — it is what keeps a
// dev box's scratch `*.html`, `README.md`, `playwright.config.js` and
// `workbox-*.js` off every crew phone, and what decisions 58/67 exist to do.
// The guard fires only on a file that something ALREADY IN THE MANIFEST
// references. Unreferenced skips still exit 0.
//
// The two reasons a target can be missing are reported separately because the
// fixes differ:
//   * `skipped (not in HEAD)`        -> commit the file.
//   * `not matched by globPatterns`  -> add a glob AND the matching
//     `backend/Dockerfile` copy. Adding the glob alone is decision 59's trap and
//     bricks the install; `tests/sw-manifest.spec.js` guards the pairing.

// HTML attributes. A `src` on <script> is the whole script graph; modulepreload
// is here so that adding one does not silently create an unguarded edge.
const HTML_SCRIPT_SRC = /<script\b[^>]*\bsrc\s*=\s*["']([^"']+)["']/gi;
const HTML_MODULEPRELOAD = /<link\b[^>]*\brel\s*=\s*["']modulepreload["'][^>]*\bhref\s*=\s*["']([^"']+)["']/gi;
// ES module specifiers. `[^;'"]` deliberately matches newlines so a multi-line
// `import {\n a,\n b\n} from './client.js'` is seen, and cannot run past a
// statement end or into a string literal.
const JS_FROM = /\b(?:import|export)\s+[^;'"]*?\bfrom\s*["']([^"']+)["']/g;
const JS_SIDE_EFFECT = /\bimport\s*["']([^"']+)["']/g;
const JS_DYNAMIC = /\bimport\s*\(\s*["']([^"']+)["']\s*\)/g;

// The parse is only meaningful if it reproduces edges we KNOW exist. One per
// mechanism: an HTML src="" and a real module-graph hop.
//
// 🛑 S1 `sync-hard-cutover`, THIS IS THE ROW YOU WILL TRIP OVER. If a canary
// reference legitimately goes away, REPLACE it with whatever takes its place.
// Do not delete the row and do not delete the check: a guard whose subject set
// silently empties reports PASS having verified nothing (B-22/B-23/B-24).
const REACHABILITY_CANARIES = [
  ['index.html', 'ptr.js'],
  ['workflows.html', 'sync-rxdb/bootstrap.js'],
];

function matchAll(re, source) {
  re.lastIndex = 0;
  const out = [];
  let m;
  while ((m = re.exec(source)) !== null) out.push(m[1]);
  return out;
}

// Absolute or protocol-relative — someone else's server, not our manifest.
// (workflows.html and onboarding.html both load SortableJS from unpkg.)
function isExternal(spec) {
  return /^[a-z][a-z0-9+.-]*:/i.test(spec) || spec.startsWith('//');
}

// 🛑 HTML AND JS DO NOT AGREE ON WHAT A PATH IS, AND COLLAPSING THE TWO RULES
// BREAKS THE BUILD. An HTML src="log.js" IS a path relative to the document. A
// bare ES specifier is NOT a path — it is a resolver name.
// `vendor/rxdb.bundle.js` contains `from "ws"`; read as a path it is a missing
// file and every build fails. Only './', '../' and '/' denote a path in JS.
function isJsPathSpecifier(spec) {
  return spec.startsWith('./') || spec.startsWith('../') || spec.startsWith('/');
}

/** Local references a precached file makes, as written. */
function collectLocalRefs(url, source) {
  const refs = [];
  if (path.posix.extname(url).toLowerCase() === '.html') {
    refs.push(...matchAll(HTML_SCRIPT_SRC, source));
    refs.push(...matchAll(HTML_MODULEPRELOAD, source));
  }
  // Runs over HTML too — inline <script type="module"> blocks are module code.
  for (const spec of [
    ...matchAll(JS_FROM, source),
    ...matchAll(JS_SIDE_EFFECT, source),
    ...matchAll(JS_DYNAMIC, source),
  ]) {
    if (isJsPathSpecifier(spec)) refs.push(spec);
  }
  return [...new Set(refs.filter(spec => !isExternal(spec)))];
}

/** Manifest URL a reference resolves to, or null if it leaves the repo. */
function resolveRef(fromUrl, spec) {
  const clean = spec.split('#')[0].split('?')[0];
  if (!clean) return null;
  const joined = clean.startsWith('/')
    ? clean.slice(1)
    : path.posix.join(path.posix.dirname(fromUrl), clean);
  const resolved = path.posix.normalize(joined);
  return resolved.startsWith('..') ? null : resolved;
}

/**
 * @param {string[]} manifestUrls final precache URLs
 * @param {object}   opts .readFile(url) -> source; .skipped Set of dropped URLs
 * @returns {{filesParsed:number, refsFound:number, refsByFile:Map, violations:Array}}
 */
function checkImportReachability(manifestUrls, opts = {}) {
  const readFile = opts.readFile || (url => fs.readFileSync(url, 'utf8'));
  const skipped = opts.skipped || new Set();
  const precached = new Set(manifestUrls);
  const refsByFile = new Map();
  const violations = [];
  let refsFound = 0;

  for (const url of manifestUrls) {
    const ext = path.posix.extname(url).toLowerCase();
    if (ext !== '.html' && ext !== '.js') continue;
    const resolvedRefs = [];
    for (const spec of collectLocalRefs(url, readFile(url))) {
      const target = resolveRef(url, spec);
      if (target === null) continue;
      refsFound++;
      resolvedRefs.push(target);
      if (precached.has(target)) continue;
      violations.push({
        from: url,
        spec,
        target,
        reason: skipped.has(target)
          ? 'skipped (not in HEAD) — commit it'
          : 'not matched by globPatterns — add a glob AND the backend/Dockerfile copy',
      });
    }
    refsByFile.set(url, resolvedRefs);
  }
  return { filesParsed: refsByFile.size, refsFound, refsByFile, violations };
}

/**
 * 🛑 A GUARD PRINTING PASS IS NOT EVIDENCE UNTIL ITS SUBJECT SET IS SHOWN
 * NON-EMPTY (B-22/B-23/B-24). Three checks, because each fails independently:
 * a manifest with no parseable files, a parse that finds no references at all,
 * and a parse that still runs but has stopped seeing a known edge.
 * @returns {string[]} reasons the guard cannot be trusted; empty means live.
 */
function reachabilityVacuityFaults(result) {
  const faults = [];
  if (result.filesParsed === 0) {
    faults.push('parsed ZERO precached .html/.js files — the manifest or the extension filter is broken');
  }
  if (result.refsFound === 0) {
    faults.push('found ZERO local references across the whole precache — the reference regexes are broken');
  }
  for (const [from, target] of REACHABILITY_CANARIES) {
    const refs = result.refsByFile.get(from);
    if (!refs || !refs.includes(target)) {
      faults.push(
        `canary lost: "${from}" no longer yields "${target}". If that reference legitimately ` +
        'moved, REPLACE the canary in REACHABILITY_CANARIES with its successor — do not delete it.',
      );
    }
  }
  return faults;
}

function importReachabilityTransform(manifest) {
  const result = checkImportReachability(manifest.map(e => e.url), { skipped: lastSkipped });

  const faults = reachabilityVacuityFaults(result);
  if (faults.length) {
    console.error('\nimport-reachability guard CANNOT BE TRUSTED — it would have passed vacuously:');
    for (const f of faults) console.error(`  ✗ ${f}`);
    throw new Error('import-reachability guard is vacuous');
  }

  // Always printed, pass or fail: the subject set, in numbers.
  console.log(
    `import reachability: ${result.filesParsed} precached files parsed, ` +
    `${result.refsFound} local references resolved, ${result.violations.length} outside the precache`,
  );

  if (result.violations.length) {
    console.error('\n🛑 PRECACHED FILES REFERENCE PATHS THAT ARE NOT PRECACHED.');
    console.error('   A page served from the precache whose script graph is not in the precache');
    console.error('   is dead on a returning client, offline, with nothing on screen saying why.\n');
    for (const v of result.violations) {
      console.error(`  ${v.from}  ->  ${v.spec}   [${v.target}]`);
      console.error(`      ${v.reason}`);
    }
    console.error('');
    throw new Error(
      `import-reachability check failed: ${result.violations.length} reference(s) resolve outside the precache`,
    );
  }
  return { manifest };
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

      // ═══ FOUND BY THE REACHABILITY GUARD ABOVE, NOT BY A HUMAN. B-37. ═════
      //
      // Both were referenced by precached pages and globbed by NOTHING, since
      // the ad-hoc commit that introduced them (`cfe7edc`, "more bug fixes").
      // `log.js` is on all 7 precached pages; `tab.js` on 5. They ship into the
      // image via `COPY *.html *.js` so they have always worked ONLINE, and have
      // always failed on a returning client with no network — which is the only
      // condition under which a precache matters at all.
      //
      // `tab.js` is the load-bearing one: it applies `#tab=N` synchronously
      // BEFORE paint, so without it Inventory, Users, Purchasing, Onboarding and
      // Operations open with every tab section visible at once and no switching.
      // Silent, offline-only, and invisible in development.
      //
      // Named individually, NOT as a bare '*.js': that glob sweeps build-sw.js,
      // playwright.config.js, sw.js itself and six workbox-*.js runtime chunks
      // onto every crew phone. Same trap as 'vendor/**' below, one directory up.
      //
      // No backend/Dockerfile change needed — `COPY *.html *.js` and
      // `cp ../*.js cmd/server/public/` already stage both, so decision 59's
      // pairing guard in tests/sw-manifest.spec.js stays green. A FUTURE glob
      // addition may not be so lucky; check that test before adding one.
      'log.js',
      'tab.js',

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
    //
    // 🛑 ORDER IS LOAD-BEARING, BOTH WAYS. The reachability guard must run LAST,
    // against the manifest that will actually ship — run it first and it checks a
    // set that still contains uncommitted files and can never see a skip. And it
    // THROWS rather than reporting, so `sw.js` is never written on a failure: a
    // guard that writes a bad artifact and then exits non-zero leaves something a
    // hurried hand can `git add`. B-37.
    manifestTransforms: [committedOnlyTransform, importReachabilityTransform],
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

// Exported so tests/sw-manifest.spec.js can drive the reachability guard over
// synthetic manifests -- proving it goes RED on a violation and stays GREEN on an
// unreferenced skip -- without a commit dance around the real tree.
module.exports = {
  collectLocalRefs,
  resolveRef,
  checkImportReachability,
  reachabilityVacuityFaults,
  REACHABILITY_CANARIES,
};

if (require.main === module) {
  build().catch(err => {
    console.error('SW build failed:', err.message || err);
    process.exit(1);
  });
}
