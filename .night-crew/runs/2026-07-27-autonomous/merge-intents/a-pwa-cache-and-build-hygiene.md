# Merge intent — Card A `pwa-cache-and-build-hygiene`

Branch: `card/a-pwa-cache-and-build-hygiene` (cut from `overnight-20260727` @ `e1c40a8`)
Written BEFORE implementation, as the card's first commit. Required per DESIGN §15ad.65.

## Card in one line

Three unrelated-in-cause, same-file hygiene changes from ledger T-23 decisions 57, 58, 59:
clear `api-cache` on logout and stop `checkAuth` from ever painting an identity it could not
verify (**a live cross-tenant disclosure on shipped crew phones**); make `build-sw.js` precache
the **tracked** set rather than the working tree, so an untracked file in the repo root can no
longer 404 and fail the entire service-worker install; and drop the vendored RxDB bundle from
the precache until a page actually imports it.

## Shared files touched

- `index.html` — **the card, half 1**. `logout()` (`:141-145`) and `checkAuth()` (`:146-172`).
  Any other card touching `index.html` tonight collides here; the orchestrator resolves at
  merge. Nothing else on tonight's slate names `index.html` in its footprint.
- `build-sw.js` — **the card, halves 2 and 3**. `globPatterns` (drop the `vendor/**` entry) and
  a new tracked-set filter over the generated precache manifest.
  **`sync-rxdb-browser-delivery-spike` (landed 2026-07-26) also owns this file** — it is the
  card that ADDED the `vendor/**/*.bundle.js` entry and its long explanatory comment. This card
  removes the entry and rewrites the comment to say why. A merge that restores the entry
  restores the 495 KiB.
- `tests/index.spec.js` — new red-first tests for the logout cache clear and the fail-closed
  identity probe, appended to the existing file. Conflicts should be resolved by **keeping both
  sides' tests**.
- `tests/sw-manifest.spec.js` — new file, unique to this card. Static assertions over the
  committed `sw.js` precache manifest (no vendored bundle; every precached URL is git-tracked).
  No conflict surface.
- `sw.js` / `version.json` — regenerated mechanically by `node build-sw.js`. Pure build output.
  See "safe to drop".
- `.night-crew/knowledge/roadmap.md` — the `pwa-cache-and-build-hygiene` card status flip, in
  the same change set as the work. Single-card edit, ~line 219.
- `.night-crew/runs/2026-07-27-autonomous/merge-intents/a-pwa-cache-and-build-hygiene.md` —
  this note. New file, unique to this card. No conflict surface.
- `.night-crew/runs/2026-07-27-autonomous/timings.log` — append-only per-leg timing lines
  prefixed `A `. Every card appends its own prefixed lines; conflicts are append-order only and
  both sides should be kept.

### Late additions

_(appended only if implementation forces a file outside the list above; "nothing here" if it
stays clean)_

**Nothing here** — the footprint held as planned. `index.html`, `build-sw.js`, the two test
files, the two regenerated build artifacts, the roadmap flip, this note, and the timings log.

## What must survive any merge

1. **`caches.delete('api-cache')` runs on logout, and is awaited before the redirect.** Fire-and-
   forget is not equivalent: `window.location.href` tears the page down, and an unawaited
   `caches.delete` can lose the race and leave the previous user's rows on the phone — which is
   the entire disclosure. The `try`/`catch` and the `'caches' in window` guard must survive too:
   a throw here must never block the logout itself.
2. **`checkAuth` never treats a possibly-cached `/api/v1/me` as a verified identity.** Two
   mechanisms, both load-bearing, and dropping either re-opens the hole:
   (a) every cached identity response is **evicted from `api-cache` before** the probe is
   issued, and (b) the probe itself carries a per-load cache-buster so the URL-keyed
   `api-cache` (no `Vary`, no `cacheKeyWillBeUsed` — `build-sw.js:60-78`) structurally cannot
   answer it. Keep (a) without (b) and a concurrent SW write re-seeds the entry; keep (b)
   without (a) and the stale entry for the bare URL simply sits there for the next code path
   that reads it.
3. **The fail-closed branch: an unverified identity paints no name and removes any name already
   painted.** Explicitly *not* a redirect — the offline branch must stay reachable, because on a
   food truck `NetworkFirst` falling back is routine, not exotic (decision 57). What must not
   survive is the previous user's name on this user's screen.
4. **`build-sw.js` builds its precache from the git-tracked set.** The mechanism is a
   `manifestTransforms` filter against `git ls-files`; what must survive is the *property*
   (nothing untracked reaches the manifest), not the particular spelling. A Workbox precache
   entry that 404s fails the **entire** SW install, so the failure mode this guards is "the PWA
   silently stops updating on every phone".
5. **The vendored bundle stays OUT of `globPatterns`.** −495 KiB, −34% of the precache, for an
   asset no page imports. `sync-rxdb-schema-and-replication` re-adds it on adoption (decision
   59, roadmap rider 5 on that card).
6. **Both new tests.** `tests/index.spec.js`'s two cache tests go red on the pre-fix tree;
   `tests/sw-manifest.spec.js` is what stops the vendored bundle silently returning to the
   precache the next time somebody edits `globPatterns`.

## What is safe to drop

- **`sw.js` and `version.json`** — regenerated by `task sw` / `node build-sw.js`. Take either
  side of a conflict and re-run `task sw`; the content hashes are derived, not authored.
- Comment wording — including the rewritten `globPatterns` block comment and the fail-closed
  explanation in `checkAuth`. The behaviour matters; the prose does not.
- Test names and the exact cache-buster parameter name (`_`).
- The `timings.log` lines. They are a record, not a behaviour.
- The roadmap card's prose. The **status flip** matters; the wording does not.
- Anything in this note itself.

## Not done, deliberately

- **No cache-key design.** Keying `api-cache` by identity — `Vary`, `cacheKeyWillBeUsed`,
  `matchOptions` — is **deferred by decision 57 to `sync-rxdb-schema-and-replication`**, which
  is expected to retire the `/\/api\//` route entirely once RxDB replicates. This card does the
  clearing and the fail-closed identity check only. The `runtimeCaching` block in `build-sw.js`
  is **not edited by this card**.
- **No change to `login.html`.** An identity transition that happens *without* a logout (B logs
  in while A's session is still live) is not closed by the logout clear; it is closed by the
  `checkAuth` eviction + cache-buster instead. Touching `login.html` was not needed.
- **No change to the `APPS_CACHE_KEY` localStorage tile cache.** It is a different mechanism
  from `api-cache`, no decision covers it, and the card is scoped to identity.
- **No `openspec/` directory.** This repo has none, `night-crew workflow preflight` reports
  ABSENT, and B-105 (which per-change discipline hq adopts) is an open operator question. It is
  not answered here by importing another repo's convention.

## Four-HARD-constraints attestation

1. **Root `package.json` AND `package-lock.json`** — **UNTOUCHED**, both files, individually
   confirmed. This is the Playwright environment shared by every card and every worktree
   tonight; no devDep is added, no version moved, no script edited. `version.json` (a generated
   artifact) and `vendor/package.json` (the vendored bundle's own generator manifest) are
   different files; neither root file is written. `build-sw.js` still *reads* root
   `package.json` to mirror the frontend semver — read-only, unchanged.
2. **`backend/go.mod`** — **UNTOUCHED**. No dependency added, removed, or version-changed. This
   card is frontend + build script + tests; it compiles no new Go.
3. **`docker-compose.nc.yml`** — **UNTOUCHED**. This card runs Playwright against its own
   `TEST_PORT=8211` / `TEST_DB_NAME=hq_test_a27`; no compose service is added, renamed, or
   re-ported. The `spike-supabase` stack is left running for card C.
4. **Root `Taskfile.yml`** — **UNTOUCHED**. `task sw` is invoked as it already exists; no task
   is added, no var default changed. The suite is run by invoking `npx playwright test`
   directly with `CI=1` and explicit env.
